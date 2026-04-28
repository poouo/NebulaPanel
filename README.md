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

1. 创建 `docker-compose.yml` 文件：

```yaml
version: "3.8"

services:
  nebula-panel:
    image: poouo/nebula-panel:latest
    container_name: nebula-panel
    restart: unless-stopped
    ports:
      - "3000:3000"
    volumes:
      - ./data:/data
    environment:
      - DB_PATH=/data/nebula.db
      - LISTEN=:3000
      - ADMIN_USER=admin
      - ADMIN_PASS=admin123
```

2. 启动服务：

```bash
docker-compose up -d
```

3. 访问面板：
打开浏览器访问 `http://你的服务器IP:3000`，使用默认账号 `admin` 和密码 `admin123` 登录。登录后请及时修改密码。

## Agent 节点安装

在面板的“Agents”页面中，点击“Install Script”按钮，即可获取一键安装脚本。

在目标节点服务器上执行以下命令即可完成安装：

```bash
curl -sL http://你的面板IP:3000/static/agent/install.sh | bash -s install http://你的面板IP:3000 你的通信密钥
```

Agent 安装后会自动作为 systemd 服务运行，并定期向面板汇报心跳和流量数据。

## 卸载 Agent

如果需要卸载 Agent，可以在节点服务器上执行：

```bash
curl -sL http://你的面板IP:3000/static/agent/install.sh | bash -s uninstall
```

## 许可证

本项目采用 MIT 许可证开源。
