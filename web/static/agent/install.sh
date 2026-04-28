#!/usr/bin/env bash
# NebulaPanel Agent one-click installer.
#
# Usage:
#   bash <(curl -fsSL http://PANEL/static/agent/install.sh) install <PANEL_URL> <AGENT_TOKEN>
#   bash <(curl -fsSL http://PANEL/static/agent/install.sh) uninstall
#   bash <(curl -fsSL http://PANEL/static/agent/install.sh) update-xray
#
# What it does:
#   1. Downloads the matching nebula-agent Go binary (first from the panel,
#      falls back to the GitHub mirror).
#   2. Downloads the latest Xray-core release from XTLS/Xray-core and unpacks
#      it to /opt/nebula-agent/xray/xray so the agent can run the kernel.
#   3. Writes /opt/nebula-agent/agent.conf with PANEL_URL, AGENT_TOKEN and
#      (optionally) the panel's COMM_KEY for encrypted heartbeats.
#   4. Installs a systemd unit (nebula-agent.service), starts it, and shows a
#      short status summary.

set -Eeuo pipefail

ACTION="${1:-install}"
PANEL_ARG="${2:-}"
TOKEN_ARG="${3:-}"

INSTALL_DIR="/opt/nebula-agent"
XRAY_DIR="${INSTALL_DIR}/xray"
BIN_PATH="${INSTALL_DIR}/nebula-agent"
CONF_PATH="${INSTALL_DIR}/agent.conf"
LOG_DIR="/var/log/nebula-agent"
SERVICE_PATH="/etc/systemd/system/nebula-agent.service"

GH_BIN_BASE="https://raw.githubusercontent.com/poouo/NebulaPanel/main/web/static/agent/bin"

c_red()    { printf '\033[31m%s\033[0m\n' "$*"; }
c_green()  { printf '\033[32m%s\033[0m\n' "$*"; }
c_yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
c_blue()   { printf '\033[34m%s\033[0m\n' "$*"; }
log()  { c_green "==> $*"; }
warn() { c_yellow "[!] $*"; }
err()  { c_red "[x] $*"; }

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    err "Please run as root (use sudo)."
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l|armv7)  echo "arm"   ;;
    *) err "Unsupported CPU architecture: $(uname -m)"; exit 1 ;;
  esac
}

xray_asset_for() {
  case "$1" in
    amd64) echo "Xray-linux-64.zip" ;;
    arm64) echo "Xray-linux-arm64-v8a.zip" ;;
    arm)   echo "Xray-linux-arm32-v7a.zip" ;;
    *) err "Unsupported arch for xray: $1"; exit 1 ;;
  esac
}

ensure_tools() {
  local need=()
  command -v curl  >/dev/null 2>&1 || need+=(curl)
  command -v unzip >/dev/null 2>&1 || need+=(unzip)
  if [ "${#need[@]}" -eq 0 ]; then return; fi
  warn "Installing prerequisites: ${need[*]}"
  if   command -v apt-get >/dev/null 2>&1; then apt-get update -y >/dev/null && apt-get install -y "${need[@]}" >/dev/null
  elif command -v dnf     >/dev/null 2>&1; then dnf install -y "${need[@]}" >/dev/null
  elif command -v yum     >/dev/null 2>&1; then yum install -y "${need[@]}" >/dev/null
  elif command -v apk     >/dev/null 2>&1; then apk add --no-cache "${need[@]}" >/dev/null
  else err "Please install these packages manually: ${need[*]}"; exit 1
  fi
}

download() {
  # $1=url $2=dest  → returns 0 on success
  curl -fsSL --retry 3 --connect-timeout 20 "$1" -o "$2"
}

install_agent_bin() {
  local arch; arch="$(detect_arch)"
  local primary="${PANEL_URL}/static/agent/bin/nebula-agent-linux-${arch}"
  local fallback="${GH_BIN_BASE}/nebula-agent-linux-${arch}"
  log "Downloading nebula-agent binary"
  c_blue   "    primary : ${primary}"
  if ! download "${primary}" "${BIN_PATH}.new"; then
    warn "Primary failed, trying GitHub mirror."
    c_blue "    fallback: ${fallback}"
    download "${fallback}" "${BIN_PATH}.new" || {
      err "Failed to download nebula-agent binary from both sources."
      exit 1
    }
  fi
  chmod +x "${BIN_PATH}.new"
  mv -f "${BIN_PATH}.new" "${BIN_PATH}"
  log "Agent binary installed at ${BIN_PATH}"
}

install_xray() {
  local arch asset url tmp
  arch="$(detect_arch)"
  asset="$(xray_asset_for "$arch")"
  url="https://github.com/XTLS/Xray-core/releases/latest/download/${asset}"
  log "Downloading latest Xray-core (${asset})"
  mkdir -p "$XRAY_DIR"
  tmp="$(mktemp -t xray-core.XXXXXX.zip)"
  if ! download "$url" "$tmp"; then
    warn "GitHub download failed. If your server has no direct GitHub access, try a mirror."
    rm -f "$tmp"
    return 1
  fi
  unzip -qo "$tmp" -d "$XRAY_DIR"
  rm -f "$tmp"
  chmod +x "${XRAY_DIR}/xray"
  local ver
  ver=$("${XRAY_DIR}/xray" version 2>/dev/null | head -n1 || true)
  log "Xray-core installed: ${ver:-unknown}"
}

fetch_comm_key() {
  # Exchange the agent token for the panel's comm_key via a one-shot
  # bootstrap endpoint. The token is validated server-side and the comm_key
  # is never exposed to anonymous callers.
  local key=""
  local payload="{\"token\":\"${AGENT_TOKEN}\"}"
  if out=$(curl -fsSL --connect-timeout 10 -X POST -H "Content-Type: application/json" \
      --data "${payload}" "${PANEL_URL}/api/agent/bootstrap" 2>/dev/null); then
    key=$(printf '%s' "$out" | sed -n 's/.*"comm_key":"\([^"]*\)".*/\1/p')
  fi
  printf '%s' "$key"
}

write_config() {
  local comm_key="$1"
  log "Writing ${CONF_PATH}"
  install -d -m 0755 "$INSTALL_DIR" "$LOG_DIR"
  cat > "$CONF_PATH" <<CONF
# Managed by NebulaPanel installer.
PANEL_URL=${PANEL_URL}
AGENT_TOKEN=${AGENT_TOKEN}
COMM_KEY=${comm_key}
HEARTBEAT_INTERVAL=15
TRAFFIC_INTERVAL=60
LOG_DIR=${LOG_DIR}
WORK_DIR=${INSTALL_DIR}
XRAY_BIN=${XRAY_DIR}/xray
CONF
  chmod 600 "$CONF_PATH"
}

write_service() {
  log "Writing ${SERVICE_PATH}"
  cat > "$SERVICE_PATH" <<UNIT
[Unit]
Description=NebulaPanel Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${CONF_PATH}
ExecStart=${BIN_PATH} -c ${CONF_PATH}
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  systemctl enable nebula-agent >/dev/null 2>&1 || true
}

action_install() {
  need_root
  if [ -z "${PANEL_ARG}" ] || [ -z "${TOKEN_ARG}" ]; then
    err "Usage: install <PANEL_URL> <AGENT_TOKEN>"
    exit 1
  fi
  PANEL_URL="${PANEL_ARG%/}"
  AGENT_TOKEN="${TOKEN_ARG}"
  ensure_tools

  log "Installing NebulaPanel Agent"
  c_blue "    Panel : ${PANEL_URL}"
  c_blue "    Token : ${AGENT_TOKEN:0:8}…"

  systemctl stop nebula-agent 2>/dev/null || true

  install_agent_bin
  install_xray || warn "Xray install failed; agent will start but proxy is disabled until you retry with: bash install.sh update-xray"

  local comm_key
  comm_key="$(fetch_comm_key)"
  if [ -z "$comm_key" ]; then
    warn "Could not auto-fetch panel comm_key. Put the panel's COMM_KEY into ${CONF_PATH} and restart the agent."
  fi

  write_config "$comm_key"
  write_service

  systemctl restart nebula-agent
  sleep 1
  if systemctl is-active --quiet nebula-agent; then
    log "NebulaPanel Agent is up and running."
    log "View logs: journalctl -u nebula-agent -f"
  else
    err "Agent failed to start. Recent journal output:"
    journalctl -u nebula-agent -n 50 --no-pager || true
    exit 1
  fi
}

action_uninstall() {
  need_root
  log "Uninstalling NebulaPanel Agent"
  systemctl stop nebula-agent 2>/dev/null || true
  systemctl disable nebula-agent 2>/dev/null || true
  rm -f "$SERVICE_PATH"
  systemctl daemon-reload || true
  rm -rf "$INSTALL_DIR"
  log "Done. Logs kept at ${LOG_DIR} (delete manually if not needed)."
}

action_update_xray() {
  need_root
  PANEL_URL="${PANEL_ARG%/}"
  install_xray
  systemctl restart nebula-agent 2>/dev/null || true
}

case "$ACTION" in
  install)     action_install ;;
  uninstall)   action_uninstall ;;
  update-xray) action_update_xray ;;
  *)
    echo "Usage: $0 {install <panel_url> <agent_token> | uninstall | update-xray}"
    exit 1
    ;;
esac
