#!/bin/bash
# NebulaPanel Agent - Encrypted Communication
# This script runs as a systemd service on target servers
# It reports heartbeat and traffic data to the panel using AES-256-GCM encryption

set -e

# Load config
CONF_FILE="/opt/nebula-agent/agent.conf"
if [ -f "$CONF_FILE" ]; then
    source "$CONF_FILE"
fi

PANEL_URL="${PANEL_URL:-http://localhost:3001}"
COMM_KEY="${COMM_KEY:-}"
HEARTBEAT_INTERVAL="${HEARTBEAT_INTERVAL:-30}"
TRAFFIC_INTERVAL="${TRAFFIC_INTERVAL:-60}"
AGENT_VERSION="1.0.0"

# ── Logging ──
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [NebulaAgent] $1"
}

# ── Encryption using OpenSSL ──
encrypt_payload() {
    local plaintext="$1"
    local key="$COMM_KEY"

    if [ -z "$key" ]; then
        log "ERROR: No communication key configured"
        return 1
    fi

    # Generate random salt (16 bytes) and nonce (12 bytes)
    local salt=$(openssl rand -hex 16)
    local nonce=$(openssl rand -hex 12)

    # Derive key using HMAC-SHA256
    local derived_key=$(echo -n "$key" | xxd -r -p | openssl dgst -sha256 -mac HMAC -macopt hexkey:${salt} 2>/dev/null | awk '{print $NF}')
    if [ -z "$derived_key" ]; then
        derived_key=$(echo -n "${salt}${key}" | openssl dgst -sha256 2>/dev/null | awk '{print $NF}')
    fi

    # Add timestamp (8 bytes, little-endian)
    local ts=$(date +%s)
    local ts_hex=$(printf '%016x' "$ts" | sed 's/\(..\)/\1\n/g' | tac | tr -d '\n')

    # Combine timestamp + plaintext
    local payload_hex="${ts_hex}$(echo -n "$plaintext" | xxd -p | tr -d '\n')"

    # Encrypt with AES-256-GCM
    local encrypted=$(echo -n "$payload_hex" | xxd -r -p | \
        openssl enc -aes-256-gcm -nosalt -K "$derived_key" -iv "$nonce" 2>/dev/null | \
        xxd -p | tr -d '\n')

    if [ -z "$encrypted" ]; then
        encrypted=$(echo -n "$payload_hex" | xxd -r -p | \
            openssl enc -aes-256-cbc -nosalt -K "$derived_key" -iv "${nonce}00000000" 2>/dev/null | \
            xxd -p | tr -d '\n')
    fi

    local combined="${salt}${nonce}${encrypted}"
    echo -n "$combined" | xxd -r -p | base64 -w 0
}

# ── System Metrics ──
get_cpu_usage() {
    local cpu=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' 2>/dev/null || echo "0")
    echo "${cpu%.*}"
}

get_mem_usage() {
    local mem=$(free | awk '/Mem:/ {printf "%.1f", $3/$2*100}' 2>/dev/null || echo "0")
    echo "$mem"
}

get_mem_total() {
    free -b 2>/dev/null | awk '/Mem:/ {print $2}' || echo "0"
}

get_net_stats() {
    local iface=$(ip route | grep default | awk '{print $5}' | head -1)
    if [ -z "$iface" ]; then iface="eth0"; fi
    local rx=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo "0")
    local tx=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo "0")
    echo "$rx $tx"
}

get_uptime_seconds() {
    awk '{print int($1)}' /proc/uptime 2>/dev/null || echo "0"
}

get_hostname_info() {
    hostname 2>/dev/null || echo "unknown"
}

get_host_ip() {
    hostname -I 2>/dev/null | awk '{print $1}' || echo ""
}

get_os_info() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "${PRETTY_NAME:-$ID $VERSION_ID}"
    else
        uname -s 2>/dev/null || echo "unknown"
    fi
}

get_arch_info() {
    uname -m 2>/dev/null || echo "unknown"
}

# ── Send Heartbeat ──
send_heartbeat() {
    local hostname_val=$(get_hostname_info)
    local host_ip=$(get_host_ip)
    local cpu=$(get_cpu_usage)
    local mem=$(get_mem_usage)
    local mem_total=$(get_mem_total)
    local net=($(get_net_stats))
    local uptime_val=$(get_uptime_seconds)
    local os_info=$(get_os_info)
    local arch_info=$(get_arch_info)

    # Use IP as host identifier
    local host_id="${host_ip:-${hostname_val}}"

    local payload="{\"hostname\":\"${hostname_val}\",\"host\":\"${host_id}\",\"cpu\":${cpu:-0},\"mem\":${mem:-0},\"mem_total\":${mem_total:-0},\"net_in\":${net[0]:-0},\"net_out\":${net[1]:-0},\"uptime\":${uptime_val:-0},\"version\":\"${AGENT_VERSION}\",\"os\":\"${os_info}\",\"arch\":\"${arch_info}\"}"

    local encrypted=$(encrypt_payload "$payload")
    if [ -z "$encrypted" ]; then
        log "ERROR: Failed to encrypt heartbeat"
        return 1
    fi

    local response=$(curl -s --connect-timeout 10 -X POST \
        -H "Content-Type: application/json" \
        -d "{\"data\":\"${encrypted}\"}" \
        "${PANEL_URL}/api/agent/heartbeat" 2>/dev/null)

    if echo "$response" | grep -q '"code":0'; then
        log "Heartbeat OK (Host: ${hostname_val}, IP: ${host_id}, CPU: ${cpu}%, MEM: ${mem}%)"
    else
        log "ERROR: Heartbeat failed: $response"
    fi
}

# ── Main Loop ──
main() {
    log "NebulaPanel Agent v${AGENT_VERSION} starting..."
    log "Panel URL: ${PANEL_URL}"
    log "Heartbeat interval: ${HEARTBEAT_INTERVAL}s"

    if [ -z "$COMM_KEY" ]; then
        log "FATAL: Communication key not configured!"
        log "Please set COMM_KEY in ${CONF_FILE}"
        exit 1
    fi

    while true; do
        send_heartbeat
        sleep "$HEARTBEAT_INTERVAL"
    done
}

main "$@"
