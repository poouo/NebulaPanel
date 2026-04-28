# NebulaPanel

NebulaPanel 是一款轻量级、高性能的多用户机场管理面板。它采用 Go 语言编写，内置 SQLite 数据库，支持 Docker 单容器一键部署。通过加密的 Agent 通信机制，您可以轻松管理分布在各地的节点服务器。

## 核心功能

- **多用户管理**：支持用户注册、流量限制、速率限制、到期时间设置。
- **订阅管理**：支持 Clash/Mihomo、Surge、Base64 等多种订阅格式，支持自定义订阅模板。
- **流量统计**：实时统计用户上下行流量，提供按小时聚合的流量趋势图表。
- **加密通信**：面板与 Agent 之间采用 AES-256-GCM 加密通信，确保数据安全。
- **一键部署**：提供 Docker 和 Docker Compose 部署方式，单容器即可运行完整服务。
- **数据备份**：支持在后台一键导出和导入所有配置及数据（JSON 格式）。
- **安全机制**：内置 SVG 图形验证码，防止暴力破解和恶意注册。

## 快速部署

推荐使用 Docker Compose 进行部署。

1. 克隆仓库：

```bash
git clone https://github.com/poouo/NebulaPanel.git
cd NebulaPanel
```

2. 启动服务：

```bash
docker-compose up -d
```

也可以自定义 `docker-compose.yml`：

```yaml
services:
  nebula-panel:
    build: .
    container_name: nebula-panel
    restart: unless-stopped
    ports:
      - "3001:3001"
    volumes:
      - ./data:/data
    environment:
      - DB_PATH=/data/nebula.db
      - LISTEN=:3001
      - ADMIN_USER=admin
      - ADMIN_PASS=admin123
```

3. 访问面板：
打开浏览器访问 `http://你的服务器IP:3001`，使用默认账号 `admin` 和密码 `admin123` 登录。登录后请及时修改密码。

## Agent 节点安装

在面板 **Agents** 页面点击 **+ 添加 Agent**，面板会为该 Agent 生成专属的 Token，并在弹窗中给出一键安装命令。将命令复制到目标服务器执行即可自动上线：

```bash
bash <(curl -sL https://你的面板域名/static/agent/install.sh) install https://你的面板域名 <AGENT_TOKEN>
```

安装脚本会自动完成以下工作：

1. 从 [XTLS/Xray-core](https://github.com/XTLS/Xray-core/releases/latest) 下载对应架构的 **最新版** 内核，安装到 `/opt/nebula-agent/xray/`。
2. 通过 `POST /api/agent/bootstrap` 以 Token 换取通信密钥。
3. 安装 NebulaPanel Agent 二进制到 `/usr/local/bin/nebula-agent`，写入 systemd 单元并启动服务。
4. Agent 启动后采集 CPU 核心/型号、内存/硬盘/OS/内核/Load Avg/实时网速等详细信息和 Xray 版本，以 AES-256-GCM 加密上报。面板根据 Token 自动绑定记录，无需再手工填 IP/端口。

在 **Agents** 页面容易看见 Agent 的实时状态：

- 支持 **卡片视图 / 列表视图** 一键切换；
- 页面打开时 Agent 自动切换到快速心跳（约 3–5s），离开后回落正常间隔；
- 页面每 5s 自动刷新，展示 CPU%、内存%、硬盘%、实时网速、核心数、Xray 版本、Agent 版本等；
- 支持 **一键重启**，指令随心跳下发、systemd 拉起。

### 卸载 Agent
```bash
bash <(curl -sL https://你的面板域名/static/agent/install.sh) uninstall
```

### 数据目录与迁移
面板容器的数据目录挂载在 Docker Compose 当前目录下的 `./data`。

- **迁移**：`docker compose down` 后复制整个项目目录到新服务器，再 `docker compose up -d` 即可。
- **清空**：`docker compose down && rm -rf ./data && docker compose up -d`。

## 许可证

本项目采用 [MIT](LICENSE) 许可证开源。
