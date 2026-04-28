# NebulaPanel 改造说明（v2 重构）

> 本次改造以"安全、零依赖、可观测、可扩展"为目标，对 NebulaPanel 进行了较大幅度的服务端、Agent、前端重构。所有改动已合并到 `main` 分支（commit `c1b364f`）。

## 一、改造目标速览

| 主题 | 目标 |
| :--- | :--- |
| Agent | 用 Go 重写，预编译二进制，一键安装脚本拉取部署 |
| 节点 | 由面板下发，Agent 用开源内核（Xray-core）生成并运行 |
| Agent 信息 | 自动上报，面板可手工修改备注、入口 IP，默认显示上报 IP |
| 项目链接 | 面板侧边栏底部加入开源项目地址 |
| 日志 | Agent 本地落盘，按文件大小切割，保留 30 天，**不依赖数据库** |
| 审计 | 面板新增审计功能，下发屏蔽域名规则给 Agent，**默认关闭** |
| 账号安全 | 数据库内 bcrypt 哈希；登录全程**不传输明文**，采用 SHA‑256 + HMAC 挑战‑响应 |

## 二、整体架构

```
┌────────────────────────────┐  AES‑GCM 加密心跳/下发    ┌────────────────────────────┐
│        NebulaPanel         │ ─────────────────────────▶ │   nebula-agent (Go 二进制) │
│  Web/API/SQLite/审计中心    │ ◀───────────────────────── │   Xray-core 由 Agent 启动  │
└────────────────────────────┘                            └────────────────────────────┘
        ▲                                                          │
        │ HTTPS + SHA‑256 / HMAC‑SHA256 挑战响应登录                │ 30 天本地日志（无 DB）
   浏览器（前端）                                                   ▼
                                                            /opt/nebula-agent/logs/
```

## 三、关键改动清单

### 3.1 Go 语言 Agent（全新模块 `agent/`）

| 路径 | 说明 |
| :--- | :--- |
| `agent/cmd/main.go` | 进程入口；解析配置、启动心跳、加载内核 |
| `agent/internal/config` | KEY=VALUE 配置加载（systemd EnvironmentFile 兼容） |
| `agent/internal/client` | 与面板通讯客户端，AES‑GCM 加密 + 时间戳防重放 |
| `agent/internal/sysinfo` | CPU/内存/网络/主机名采集（gopsutil） |
| `agent/internal/core` | Xray 配置生成与进程管理；变更才重启 |
| `agent/internal/logger` | lumberjack 文件日志，**30 天保留**，无数据库 |
| `web/static/agent/bin/` | 预编译二进制：`nebula-agent-linux-amd64` / `linux-arm64`，含 SHA256SUMS |
| `web/static/agent/install.sh` | 一键安装脚本，自动识别架构，从面板优先、GitHub 兜底拉取二进制 |

### 3.2 面板后端

- `internal/db/db.go`：新增表 `audit_rules`、为 `agents` 增加 `remark`、`entry_ip` 字段，为 `nodes` 增加 `agent_id` 字段，自动迁移。
- `internal/api/extra_handlers.go`（新增）：
  - `PUT /api/agents/{id}` 修改备注/入口 IP/名称
  - `GET/POST/PUT/DELETE /api/audit/rules` 审计规则 CRUD
  - `buildAgentSpec(agentID)`：心跳响应携带"绑定到该 Agent 的节点列表 + 审计开关 + 屏蔽域名规则"
- `internal/api/handlers.go`：
  - 节点 CRUD 支持 `agent_id`，列表返回 `agent_id`
  - Agent 列表返回 `remark`、`entry_ip`，并新增 `display_ip`（优先显示备注 IP）
  - 心跳处理改为：签名校验 → 入库统计 → `buildAgentSpec` 加密回包
  - 登录链路改造（详见 3.4）
- `internal/auth/`：
  - `passhash.go`：统一 `bcrypt(SHA256(password))` 流程，提供创建/校验/迁移函数
  - `verifier.go`：`auth_verifiers` 表保存 client_hash 摘要，用于挑战‑响应
  - `challenge.go`：内存 challenge 池，60 秒过期，一次性消费

### 3.3 面板前端

- 登录、注册、改密、新增/编辑用户：表单中**不再传输 `password` 明文**，改为传 `client_hash = SHA‑256(password)`，并在登录时先 `GET /api/login/challenge?username=…` 拿到随机 challenge，再用 `HMAC‑SHA256(client_hash, challenge)` 计算 `response` 提交。
- Agent 列表新增"备注 / 入口 IP"列与编辑弹窗。
- 节点表单新增"绑定 Agent"下拉。
- 新增"审计"页面：开关默认关闭，规则可启用/停用、备注，规则形态支持 `domain.com` / `regexp:^…$` / `keyword:xxx`（沿用 Xray 的 routing 语法）。
- 侧栏底部加入开源项目链接 `https://github.com/poouo/NebulaPanel`。

### 3.4 账号密码安全（重点）

| 环节 | 旧实现 | 新实现 |
| :--- | :--- | :--- |
| 数据库存储 | bcrypt(plain) | `bcrypt(SHA256(plain))`；同时保存 `verifier = SHA256(client_hash)` |
| 登录请求体 | `{username, password}` 明文 | `{username, client_hash, challenge, response}`，**无明文** |
| 校验流程 | 直接 bcrypt.Compare | 1) 校验 challenge 在内存池且未过期；2) 比对 `HMAC(client_hash, challenge) == response`；3) 比对 bcrypt 摘要；任一不通过则失败 |
| 兼容老账户 | — | 第一次老账号登录时若仅有明文 password 字段，回退按"客户端直传 bcrypt 校验"，成功后即时迁移为新方案 |
| 抗重放 | 无 | challenge 一次性，60s 过期；同时 Agent 通讯沿用现有 AES‑GCM + 时间戳 |

### 3.5 审计功能

- 默认 `audit_enabled = false`，开启后才会下发规则，关闭后下发空规则即时生效。
- 规则字段：`id`/`domain`/`remark`/`enabled`，与 Xray routing rule 兼容。
- Agent 收到 spec 后写入 `xray.json` 的 `routing.rules`，将匹配域名指向 `blackhole` 出站达到 block 效果。
- 修改规则会在下一次 Agent 心跳（默认 15 秒）内生效。

### 3.6 日志体系

- Agent 启动时初始化 lumberjack：`agent.log` 单文件 50MB 切割，保留 30 个备份，**MaxAge=30 天**，自动按 GZIP 压缩。
- 启动一个 goroutine 每天凌晨清理过期文件。
- 整个 Agent 进程不打开任何数据库连接。

## 四、部署 / 升级

### 4.1 面板侧

```bash
git pull
CGO_ENABLED=1 go build -o /usr/local/bin/nebula-panel ./cmd
systemctl restart nebula-panel
```

迁移会自动执行：补齐 `agents.remark`、`agents.entry_ip`、`nodes.agent_id`、新增 `audit_rules`、`auth_verifiers` 表。无需手工 SQL。

### 4.2 Agent 侧

在面板"Agent → 添加"中复制安装命令，例如：

```bash
bash <(curl -sL https://raw.githubusercontent.com/poouo/NebulaPanel/main/web/static/agent/install.sh \
  || curl -sL https://your-panel/static/agent/install.sh) \
  install https://your-panel <COMM_KEY>
```

脚本会：

1. 自动识别 `amd64` / `arm64`；
2. 优先从面板下载二进制，失败则 GitHub raw 兜底；
3. 写入 `/opt/nebula-agent/agent.conf` 与 systemd unit；
4. 启动 `nebula-agent.service`，日志位于 `/opt/nebula-agent/logs/agent.log`。

卸载：`bash install.sh uninstall`。

## 五、验收

| 项 | 验证方式 | 结果 |
| :--- | :--- | :--- |
| 旧账号登录 | `admin/admin123` 走兼容链路 | 通过 |
| 新挑战‑响应登录 | challenge → response 校验 | 通过 |
| Agent 自动注册 | 启动后 5s 内出现在 `/api/agents` | 通过 |
| 备注/入口 IP | `PUT /api/agents/1` 写入后 `display_ip` 优先返回 | 通过 |
| 节点下发 | 创建带 `agent_id` 的节点后 Agent 生成 `xray.json` 含对应 inbound | 通过 |
| 审计 block | 启用 audit + 规则后 `xray.json` routing 出现 blackhole 规则 | 通过 |
| 30 天日志 | logger 启动日志显示 `retention=30 days` | 通过 |
| 无数据库依赖 | `agent/` 模块 `go.sum` 不含任何 SQL 驱动 | 通过 |

## 六、后续可扩展点

1. Agent 端可以接入更多内核（sing‑box、hysteria）→ `core` 包做接口抽象。
2. 审计可在面板增加"匹配命中统计"，由 Agent 上报 routing 命中数即可。
3. 登录可在前端增加 WebAuthn / TOTP 二次验证。
