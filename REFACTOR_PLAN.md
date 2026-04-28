# NebulaPanel 改造方案

## 1. 目标概述
根据需求，对 NebulaPanel 进行以下改造：
1. **Agent 改造**：使用 Go 语言重写 Agent，编译为二进制文件，提供一键安装脚本。
2. **节点下发与生成**：面板添加节点时，将配置下发给 Agent，Agent 使用开源内核（如 Xray/Mihomo）生成并运行节点。
3. **Agent 信息完善**：面板支持对 Agent 添加备注、自定义入口 IP（默认使用上报 IP）。
4. **面板 UI 完善**：在面板中加入本开源项目的 GitHub 链接。
5. **Agent 日志管理**：Agent 端本地保存日志，保留 30 天，不依赖数据库。
6. **审计功能**：面板增加审计功能（默认关闭），可配置 block 网址并下发给 Agent 阻断访问。
7. **登录安全**：用户密码在数据库中哈希保存，登录时采用挑战-响应（Challenge-Response）机制，避免明文传输密码。

## 2. 详细设计

### 2.1 登录安全改造（挑战-响应机制）
- **后端**：
  - 新增获取 Challenge 的接口 `GET /api/login/challenge?username=xxx`，生成随机字符串（如 32 字节 hex）并存入内存（带短时效，如 1 分钟）。
  - 登录接口 `POST /api/login` 接收 `username` 和 `response`（以及验证码）。
  - 验证逻辑：从数据库获取用户的 bcrypt hash，使用相同的算法（如 HMAC-SHA256(hash, challenge)）在服务端计算 expected_response，与客户端提交的 response 比较。
  - *注意*：由于 bcrypt hash 只能单向验证，标准的挑战-响应通常需要客户端知道明文密码或其等价物。为了不传输明文，客户端可以先将密码进行一次 SHA256，服务端数据库保存的是 SHA256(password) 的 bcrypt hash。登录时，客户端请求 challenge，然后提交 `HMAC-SHA256(SHA256(password), challenge)`。服务端无法直接验证这个 HMAC，因为服务端只有 bcrypt hash。
  - **更可行的方案**：客户端使用 PBKDF2 或 SHA256 将密码哈希后传输，服务端将其视为“原密码”进行 bcrypt 校验。为了防止重放，可以在传输层加一层非对称加密（如 RSA）或使用动态盐。
  - **最终方案**：前端在提交前，请求一个一次性公钥（或使用 HTTPS 保护下的简单哈希）。考虑到已有 HTTPS 建议，最简单且满足“不直接明文传输”的方法是：前端提交 `SHA256(password)`，后端数据库中保存的是 `bcrypt(SHA256(password))`。这样传输的不是原始明文密码。若要严格的挑战-响应：
    - 注册时：前端提交 `ClientHash = SHA256(password)`，后端保存 `DBHash = bcrypt(ClientHash)`。
    - 登录时：前端获取 `Challenge`，计算 `Response = HMAC-SHA256(ClientHash, Challenge)`。后端由于只有 `bcrypt(ClientHash)`，无法直接验证 HMAC。
    - 因此，采用 **SRP (Secure Remote Password)** 协议或简单的 **前端哈希传输**。为降低复杂度，采用前端哈希传输：前端提交 `SHA256(password)`，后端验证 `bcrypt(SHA256(password))`。这满足了“不直接明文传输”的要求。

### 2.2 Agent Go 语言重写
- **语言与编译**：使用 Go 1.22 编写，支持跨平台编译（Linux amd64/arm64）。
- **功能模块**：
  - **心跳与系统信息**：定期（如 30s）收集 CPU、内存、网络、Uptime，加密上报给面板。
  - **配置拉取**：心跳响应中或定期拉取面板下发的节点配置和审计规则。
  - **内核管理**：内置或自动下载开源内核（如 Xray-core），根据下发的节点配置生成 `config.json`，并启动/重启内核进程。
  - **审计阻断**：在生成的内核配置中，利用内核的路由规则（Routing）实现指定网址的 block（如 Xray 的 `domain` 规则，outbound 设为 `block`）。
  - **日志管理**：使用 Go 的日志库（如 `lumberjack`）实现本地日志轮转，保留 30 天，纯文件存储，无数据库依赖。

### 2.3 面板后端改造
- **数据库变更**：
  - `agents` 表新增字段：`remark` (TEXT), `entry_ip` (TEXT)。
  - 新增 `audit_rules` 表：`id`, `name`, `domain`, `enabled`。
  - 新增 `agent_nodes` 表或在心跳接口中下发该 Agent 负责的节点列表。
- **接口变更**：
  - `PUT /api/agents/{id}`：支持更新 `remark` 和 `entry_ip`。
  - `GET /api/agents/{id}/config`：Agent 获取自身配置（节点列表、审计规则）。
  - 审计规则的 CRUD 接口。
- **节点下发逻辑**：
  - 节点表中需增加 `agent_id` 字段，表示该节点由哪个 Agent 承载。
  - Agent 心跳时，面板返回该 Agent 的节点配置和全局开启的审计规则。

### 2.4 面板前端改造
- **Agent 列表**：显示备注和入口 IP，支持编辑。
- **节点管理**：添加/编辑节点时，可选择所属的 Agent。
- **审计管理**：新增“审计规则”页面，支持添加 block 网址，全局开关。
- **开源链接**：在侧边栏或底部加入 GitHub 项目链接。
- **登录页**：引入 `crypto-js` 或原生 `crypto.subtle` 进行密码哈希。

### 2.5 一键安装脚本
- 编写 `install.sh`，根据系统架构下载对应的预编译 Go Agent 二进制文件。
- 生成 `systemd` 服务文件，配置面板 URL 和通信密钥。

## 3. 实施步骤
1. **数据库迁移**：修改 `internal/db/db.go`，增加新字段和表。
2. **后端 API 开发**：实现 Agent 备注、入口 IP、审计规则、节点关联的接口。
3. **前端 UI 开发**：更新页面，集成密码哈希。
4. **Go Agent 开发**：实现核心逻辑（心跳、配置拉取、内核启停、日志轮转）。
5. **联调与测试**：验证全流程。
6. **编译与发布**：编写构建脚本，更新安装脚本。
