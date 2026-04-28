#!/bin/bash
# NebulaPanel Agent - One-Click Install / Uninstall Script
# Usage:
#   Install:   curl -sL <URL>/static/agent/install.sh | bash -s install <PANEL_URL> <COMM_KEY>
#   Uninstall: curl -sL <URL>/static/agent/install.sh | bash -s uninstall

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

INSTALL_DIR="/opt/nebula-agent"
SERVICE_NAME="nebula-agent"
REPO_URL="https://raw.githubusercontent.com/poouo/NebulaPanel/main/agent/nebula-agent.sh"

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_deps() {
    local missing=""
    for cmd in curl openssl xxd base64; do
        if ! command -v "$cmd" &>/dev/null; then
            missing="$missing $cmd"
        fi
    done
    if [ -n "$missing" ]; then
        log_warn "Missing dependencies:$missing"
        log_info "Installing dependencies..."
        if command -v apt-get &>/dev/null; then
            apt-get update -qq && apt-get install -y -qq curl openssl xxd coreutils
        elif command -v yum &>/dev/null; then
            yum install -y curl openssl vim-common coreutils
        else
            log_error "Cannot install dependencies automatically. Please install:$missing"
            exit 1
        fi
    fi
}

install_agent() {
    local panel_url="$1"
    local comm_key="$2"

    if [ -z "$panel_url" ] || [ -z "$comm_key" ]; then
        echo ""
        echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
        echo -e "${CYAN}║       NebulaPanel Agent Installer        ║${NC}"
        echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
        echo ""
        read -p "Panel URL (e.g. http://your-server:3000): " panel_url
        read -p "Communication Key: " comm_key
        echo ""
    fi

    if [ -z "$panel_url" ] || [ -z "$comm_key" ]; then
        log_error "Panel URL and Communication Key are required"
        exit 1
    fi

    log_info "Installing NebulaPanel Agent..."

    # Stop existing service
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log_info "Stopping existing agent..."
        systemctl stop "$SERVICE_NAME"
    fi

    # Create directory
    mkdir -p "$INSTALL_DIR"

    # Download agent script - try GitHub first, fallback to panel
    log_info "Downloading agent script..."
    if curl -sL --connect-timeout 10 "$REPO_URL" -o "$INSTALL_DIR/nebula-agent.sh" 2>/dev/null; then
        log_info "Downloaded from GitHub"
    else
        log_warn "GitHub timeout, downloading from panel..."
        if curl -sL --connect-timeout 15 "${panel_url}/static/agent/nebula-agent.sh" -o "$INSTALL_DIR/nebula-agent.sh" 2>/dev/null; then
            log_info "Downloaded from panel"
        else
            log_error "Failed to download agent script"
            exit 1
        fi
    fi
    chmod +x "$INSTALL_DIR/nebula-agent.sh"

    # Write config
    cat > "$INSTALL_DIR/agent.conf" <<EOF
PANEL_URL=${panel_url}
COMM_KEY=${comm_key}
HEARTBEAT_INTERVAL=30
TRAFFIC_INTERVAL=60
EOF
    chmod 600 "$INSTALL_DIR/agent.conf"
    log_info "Config written to $INSTALL_DIR/agent.conf"

    # Create systemd service
    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=NebulaPanel Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/bin/bash ${INSTALL_DIR}/nebula-agent.sh
Restart=always
RestartSec=10
EnvironmentFile=${INSTALL_DIR}/agent.conf
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Enable and start
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║    NebulaPanel Agent Installed!           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    log_info "Install directory: $INSTALL_DIR"
    log_info "Config file: $INSTALL_DIR/agent.conf"
    log_info "Service: $SERVICE_NAME"
    echo ""
    log_info "Commands:"
    echo "  Status:  systemctl status $SERVICE_NAME"
    echo "  Logs:    journalctl -u $SERVICE_NAME -f"
    echo "  Restart: systemctl restart $SERVICE_NAME"
    echo "  Stop:    systemctl stop $SERVICE_NAME"
    echo ""
}

uninstall_agent() {
    echo ""
    echo -e "${YELLOW}╔══════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║    NebulaPanel Agent Uninstaller          ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════╝${NC}"
    echo ""

    log_info "Stopping agent service..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true

    log_info "Removing service file..."
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload

    log_info "Removing agent files..."
    rm -rf "$INSTALL_DIR"

    echo ""
    log_info "NebulaPanel Agent has been completely uninstalled!"
    echo ""
}

show_help() {
    echo ""
    echo "NebulaPanel Agent Installer"
    echo ""
    echo "Usage:"
    echo "  $0 install [PANEL_URL] [COMM_KEY]   Install agent"
    echo "  $0 uninstall                         Uninstall agent"
    echo "  $0 help                              Show this help"
    echo ""
    echo "Examples:"
    echo "  curl -sL URL/static/agent/install.sh | bash -s install http://panel:3000 your_key"
    echo "  curl -sL URL/static/agent/install.sh | bash -s uninstall"
    echo ""
}

# ── Main ──
check_root

case "${1:-install}" in
    install)
        check_deps
        install_agent "$2" "$3"
        ;;
    uninstall)
        uninstall_agent
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
