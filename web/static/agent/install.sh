#!/usr/bin/env bash
# NebulaPanel Agent (Go) one-click installer / uninstaller.
#
# Usage:
#   bash install.sh install <PANEL_URL> <COMM_KEY> [AGENT_PORT]
#   bash install.sh uninstall
#
# Pulls the prebuilt Go agent binary from the panel (with GitHub fallback),
# installs it to /opt/nebula-agent, configures /opt/nebula-agent/agent.conf
# and registers a systemd service called `nebula-agent`.

set -euo pipefail

INSTALL_DIR="/opt/nebula-agent"
BIN_PATH="${INSTALL_DIR}/nebula-agent"
CONF_PATH="${INSTALL_DIR}/agent.conf"
LOG_DIR="${INSTALL_DIR}/logs"
SERVICE_FILE="/etc/systemd/system/nebula-agent.service"
GH_BIN_BASE="https://raw.githubusercontent.com/poouo/NebulaPanel/main/web/static/agent/bin"

c_red()    { printf '\033[31m%s\033[0m\n' "$*"; }
c_green()  { printf '\033[32m%s\033[0m\n' "$*"; }
c_yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
c_blue()   { printf '\033[34m%s\033[0m\n' "$*"; }

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    c_red "[!] Please run this script as root (or with sudo)."
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) c_red "[!] Unsupported CPU architecture: $(uname -m)"; exit 1 ;;
  esac
}

ensure_curl() {
  if command -v curl >/dev/null 2>&1; then return; fi
  c_yellow "[*] Installing curl..."
  if   command -v apt-get >/dev/null 2>&1; then apt-get update -y && apt-get install -y curl
  elif command -v yum     >/dev/null 2>&1; then yum install -y curl
  elif command -v apk     >/dev/null 2>&1; then apk add --no-cache curl
  else c_red "[!] Cannot find apt/yum/apk to install curl, please install it manually."; exit 1
  fi
}

download_binary() {
  local panel_url="$1" arch="$2" target="$3"
  local panel_path="${panel_url%/}/static/agent/bin/nebula-agent-linux-${arch}"
  local gh_path="${GH_BIN_BASE}/nebula-agent-linux-${arch}"

  c_blue "[*] Downloading agent binary (arch=${arch})"
  c_blue "    primary : ${panel_path}"
  if curl -fsSL --connect-timeout 15 -o "${target}" "${panel_path}"; then
    c_green "[+] Downloaded from panel."
    return 0
  fi
  c_yellow "[~] Panel download failed, falling back to GitHub..."
  c_blue   "    fallback: ${gh_path}"
  if curl -fsSL --connect-timeout 30 -o "${target}" "${gh_path}"; then
    c_green "[+] Downloaded from GitHub."
    return 0
  fi
  c_red "[!] Failed to download agent binary from both panel and GitHub."
  exit 1
}

install_agent() {
  local panel_url="${1:-}" comm_key="${2:-}" agent_port="${3:-9527}"

  if [ -z "${panel_url}" ] || [ -z "${comm_key}" ]; then
    c_red "[!] Usage: $0 install <PANEL_URL> <COMM_KEY> [AGENT_PORT]"
    exit 1
  fi

  require_root
  ensure_curl

  systemctl stop nebula-agent 2>/dev/null || true

  local arch; arch="$(detect_arch)"
  install -d -m 0755 "${INSTALL_DIR}" "${LOG_DIR}"

  download_binary "${panel_url}" "${arch}" "${BIN_PATH}.new"
  chmod +x "${BIN_PATH}.new"
  mv -f "${BIN_PATH}.new" "${BIN_PATH}"

  cat > "${CONF_PATH}" <<EOF
PANEL_URL=${panel_url}
COMM_KEY=${comm_key}
AGENT_PORT=${agent_port}
LOG_DIR=${LOG_DIR}
LOG_RETENTION_DAYS=30
HEARTBEAT_INTERVAL=15
EOF
  chmod 0600 "${CONF_PATH}"

  cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=NebulaPanel Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${CONF_PATH}
ExecStart=${BIN_PATH} -config ${CONF_PATH}
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable nebula-agent >/dev/null 2>&1 || true
  systemctl restart nebula-agent

  c_green "[OK] NebulaPanel Agent installed."
  c_green "    Binary  : ${BIN_PATH}"
  c_green "    Config  : ${CONF_PATH}"
  c_green "    Logs    : ${LOG_DIR} (retention: 30 days)"
  c_green "    Service : systemctl status nebula-agent"
}

uninstall_agent() {
  require_root
  systemctl stop nebula-agent 2>/dev/null || true
  systemctl disable nebula-agent 2>/dev/null || true
  rm -f "${SERVICE_FILE}"
  systemctl daemon-reload || true
  rm -rf "${INSTALL_DIR}"
  c_green "[OK] NebulaPanel Agent removed."
}

main() {
  local cmd="${1:-install}"; shift || true
  case "${cmd}" in
    install)   install_agent "$@" ;;
    uninstall) uninstall_agent ;;
    *) c_red "Usage: $0 {install <PANEL_URL> <COMM_KEY> [AGENT_PORT] | uninstall}"; exit 1 ;;
  esac
}

main "$@"
