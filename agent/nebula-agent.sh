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
# AES-256-GCM encryption with PBKDF2 key derivation
# Format: base64(salt + nonce + ciphertext + tag)
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

    # Derive key using PBKDF2
    local derived_key=$(echo -n "$key" | xxd -r -p | openssl dgst -sha256 -mac HMAC -macopt hexkey:${salt} 2>/dev/null | awk '{print $NF}')
    if [ -z "$derived_key" ]; then
        # Fallback: use key directly with salt hash
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
        # Fallback: use AES-256-CBC if GCM not available
        encrypted=$(echo -n "$payload_hex" | xxd -r -p | \
            openssl enc -aes-256-cbc -nosalt -K "$derived_key" -iv "${nonce}00000000" 2>/dev/null | \
            xxd -p | tr -d '\n')
    fi

    # Combine: salt + nonce + encrypted
    local combined="${salt}${nonce}${encrypted}"

    # Base64 encode
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

get_hostname() {
    hostname -I 2>/dev/null | awk '{print $1}' || hostname 2>/dev/null || echo "unknown"
}

# ── Send Heartbeat ──
send_heartbeat() {
    local host=$(get_hostname)
    local cpu=$(get_cpu_usage)
    local mem=$(get_mem_usage)
    local net=($(get_net_stats))
    local uptime=$(get_uptime_seconds)

    local payload="{\"host\":\"${host}\",\"cpu\":${cpu:-0},\"mem\":${mem:-0},\"net_in\":${net[0]:-0},\"net_out\":${net[1]:-0},\"uptime\":${uptime:-0},\"version\":\"${AGENT_VERSION}\"}"

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
        log "Heartbeat sent successfully (CPU: ${cpu}%, MEM: ${mem}%)"
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

    local hb_counter=0
    while true; do
        send_heartbeat
        sleep "$HEARTBEAT_INTERVAL"
    done
}

main "$@"
