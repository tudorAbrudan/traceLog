#!/usr/bin/env bash
# TraceLog installer: (1) GitHub release tarball (2) system go install (3) download official Go from go.dev, then go install.
# Override bootstrap Go versions: TRACELOG_BOOTSTRAP_GO="1.24.3 1.23.5"
# Uninstall: curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash
set -euo pipefail

# Set by try_release_download when a release is found; default for messages only.
REPO="tudorAbrudan/tracelog"
BINARY_NAME="tracelog"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tracelog"
DATA_DIR="/var/lib/tracelog"
SERVICE_USER="tracelog"
DEFAULT_PORT=8090

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

info()  { echo -e "${BLUE}  $1${NC}"; }
ok()    { echo -e "${GREEN}  ✓ $1${NC}"; }
warn()  { echo -e "${YELLOW}  ! $1${NC}"; }
error() { echo -e "${RED}  ✗ $1${NC}"; }

header() {
    echo ""
    echo -e "${CYAN}  ╔══════════════════════════════════════╗${NC}"
    echo -e "${CYAN}  ║     ${BOLD}TraceLog Installer${NC}${CYAN}               ║${NC}"
    echo -e "${CYAN}  ╚══════════════════════════════════════╝${NC}"
    echo ""
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    case "$OS" in
        linux) ;;
        darwin) ;;
        *) error "Unsupported OS: $OS"; exit 1 ;;
    esac

    info "Detected: ${OS} ${ARCH}"
}

# Try GitHub release tarball (no Go required)
try_release_download() {
    info "Trying GitHub release binary..."

    local LATEST=""
    local r=""
    for r in "tudorAbrudan/tracelog" "tudorAbrudan/traceLog"; do
        LATEST=$(curl -sL "https://api.github.com/repos/${r}/releases/latest" | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -n "$LATEST" ] && [ "$LATEST" != "null" ]; then
            REPO="$r"
            break
        fi
        LATEST=""
    done

    if [ -z "$LATEST" ]; then
        warn "No GitHub release found for this repo."
        return 1
    fi

    local ARCHIVE="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
    local DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

    if ! curl -sL -f -o "/tmp/${ARCHIVE}" "$DOWNLOAD_URL"; then
        warn "Release download failed: $DOWNLOAD_URL"
        rm -f "/tmp/${ARCHIVE}"
        return 1
    fi

    rm -f "/tmp/${BINARY_NAME}"
    tar -xzf "/tmp/${ARCHIVE}" -C /tmp "${BINARY_NAME}" 2>/dev/null || tar -xzf "/tmp/${ARCHIVE}" -C /tmp
    rm -f "/tmp/${ARCHIVE}"
    if [ ! -f "/tmp/${BINARY_NAME}" ]; then
        warn "Archive did not contain ${BINARY_NAME}"
        return 1
    fi

    chmod +x "/tmp/${BINARY_NAME}"
    sudo mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    ok "Installed release to ${INSTALL_DIR}/${BINARY_NAME}"
    return 0
}

# Fallback: go install (needs Go; GOTOOLCHAIN=auto can install a newer toolchain)
install_via_go() {
    info "Installing via Go (go install)..."
    if ! command -v go &>/dev/null; then
        return 1
    fi
    export GOTOOLCHAIN=auto
    local tmp
    tmp=$(mktemp -d)
    export GOBIN="$tmp"
    if ! go install github.com/tudorAbrudan/tracelog/cmd/tracelog@latest; then
        rm -rf "$tmp"
        return 1
    fi
    if [ ! -f "$tmp/${BINARY_NAME}" ]; then
        rm -rf "$tmp"
        return 1
    fi
    chmod +x "$tmp/${BINARY_NAME}"
    sudo mv "$tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "$tmp"
    ok "Installed via go install to ${INSTALL_DIR}/${BINARY_NAME}"
    return 0
}

# No system Go: download official Go tarball from go.dev, then go install (GOTOOLCHAIN=auto pulls module toolchain).
install_via_bootstrap_go() {
    if ! command -v curl &>/dev/null; then
        warn "curl is required to download Go; install it (e.g. apt install curl) and re-run."
        return 1
    fi
    info "No system Go found; downloading official Go from go.dev (~150MB, one-time)..."
    local work
    work=$(mktemp -d)
    local versions
    if [ -n "${TRACELOG_BOOTSTRAP_GO:-}" ]; then
        versions="$TRACELOG_BOOTSTRAP_GO"
    else
        versions="1.24.3 1.23.5 1.22.10"
    fi
    local ver tarball url gbin
    for ver in $versions; do
        tarball="go${ver}.${OS}-${ARCH}.tar.gz"
        url="https://go.dev/dl/${tarball}"
        info "Trying Go ${ver}..."
        if ! curl -sL -f -o "${work}/${tarball}" "$url"; then
            rm -f "${work}/${tarball}"
            continue
        fi
        if ! tar -C "$work" -xzf "${work}/${tarball}"; then
            rm -f "${work}/${tarball}"
            rm -rf "${work}/go"
            continue
        fi
        rm -f "${work}/${tarball}"
        export PATH="${work}/go/bin:${PATH}"
        export GOTOOLCHAIN=auto
        gbin=$(mktemp -d)
        export GOBIN="$gbin"
        if go install github.com/tudorAbrudan/tracelog/cmd/tracelog@latest && [ -f "$gbin/${BINARY_NAME}" ]; then
            chmod +x "$gbin/${BINARY_NAME}"
            sudo mv "$gbin/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
            rm -rf "$gbin" "${work}/go" "$work"
            ok "Installed via bootstrap Go ${ver} + go install → ${INSTALL_DIR}/${BINARY_NAME}"
            return 0
        fi
        rm -rf "$gbin"
        rm -rf "${work}/go"
    done
    rm -rf "$work"
    warn "Bootstrap Go + go install failed (network, disk space, or module build error)."
    return 1
}

obtain_binary() {
    if try_release_download; then
        return 0
    fi
    if install_via_go; then
        return 0
    fi
    if install_via_bootstrap_go; then
        return 0
    fi
    error "Could not install TraceLog."
    error "Fix: ensure curl works, disk space ~300MB free, outbound HTTPS; or publish a GitHub release; or apt install golang-go and re-run."
    exit 1
}

# Create system user
create_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        ok "User '${SERVICE_USER}' already exists"
        return
    fi

    if [ "$OS" = "linux" ]; then
        sudo useradd -r -s /bin/false -d "$DATA_DIR" "$SERVICE_USER" 2>/dev/null || true
        ok "Created system user '${SERVICE_USER}'"

        # Add to docker group if docker is available
        if command -v docker &>/dev/null; then
            sudo usermod -aG docker "$SERVICE_USER" 2>/dev/null || true
            ok "Added to docker group"
        fi

        # Add to adm group for log access
        sudo usermod -aG adm "$SERVICE_USER" 2>/dev/null || true
    fi
}

# Create directories
create_dirs() {
    sudo mkdir -p "$CONFIG_DIR" "$DATA_DIR" "${DATA_DIR}/backups"
    sudo chown -R "$SERVICE_USER":"$SERVICE_USER" "$DATA_DIR" 2>/dev/null || sudo chown -R "$(whoami)" "$DATA_DIR"
    ok "Data directory: ${DATA_DIR}"
}

# Auto-detect services
detect_services() {
    info "Detecting services..."

    DOCKER_FOUND=false
    DOCKER_COUNT=0
    if command -v docker &>/dev/null && docker ps -q &>/dev/null 2>&1; then
        DOCKER_FOUND=true
        DOCKER_COUNT=$(docker ps -q 2>/dev/null | wc -l | tr -d ' ')
        ok "Docker found (${DOCKER_COUNT} containers running)"
    fi

    NGINX_LOG=""
    if [ -f /var/log/nginx/access.log ]; then
        NGINX_LOG="/var/log/nginx/access.log"
        ok "nginx found (${NGINX_LOG})"
    fi

    SYSLOG=""
    if [ -f /var/log/syslog ]; then
        SYSLOG="/var/log/syslog"
        ok "System logs (${SYSLOG})"
    elif [ -f /var/log/messages ]; then
        SYSLOG="/var/log/messages"
        ok "System logs (${SYSLOG})"
    fi
}

# Create admin user and generate password
create_admin() {
    info "Creating admin account..."
    ADMIN_PASS=$(sudo -u "$SERVICE_USER" "${INSTALL_DIR}/${BINARY_NAME}" user create admin 2>/dev/null | grep "Password:" | awk '{print $2}') || \
    ADMIN_PASS=$("${INSTALL_DIR}/${BINARY_NAME}" user create admin 2>/dev/null | grep "Password:" | awk '{print $2}')

    if [ -z "$ADMIN_PASS" ]; then
        ADMIN_PASS="check-tracelog-output"
        warn "Could not capture password. Run: tracelog user create admin"
    fi
}

# Generate config file
generate_config() {
    local mode="${1:-serve}"
    local hub_url="${2:-}"
    local api_key="${3:-}"

    if [ "$mode" = "serve" ]; then
        cat <<YAML | sudo tee "${CONFIG_DIR}/config.yaml" > /dev/null
mode: serve
port: ${DEFAULT_PORT}
bind_address: "0.0.0.0"
data_dir: ${DATA_DIR}
collect:
  interval_seconds: 10
  system: true
  docker: ${DOCKER_FOUND}
YAML
    else
        cat <<YAML | sudo tee "${CONFIG_DIR}/config.yaml" > /dev/null
mode: agent
hub_url: "${hub_url}"
api_key: "${api_key}"
collect:
  interval_seconds: 10
  system: true
  docker: ${DOCKER_FOUND}
YAML
    fi

    ok "Config: ${CONFIG_DIR}/config.yaml"
}

# Create systemd service
create_service() {
    if [ "$OS" != "linux" ]; then
        warn "Systemd not available on ${OS}. Start manually: tracelog serve"
        return
    fi

    cat <<UNIT | sudo tee /etc/systemd/system/tracelog.service > /dev/null
[Unit]
Description=TraceLog Server Monitoring
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=${SERVICE_USER}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} serve --port ${DEFAULT_PORT}
Restart=always
RestartSec=5
WorkingDirectory=${DATA_DIR}
Environment=HOME=${DATA_DIR}

[Install]
WantedBy=multi-user.target
UNIT

    sudo systemctl daemon-reload
    sudo systemctl enable tracelog
    sudo systemctl start tracelog
    ok "Service created and started"
}

# Check firewall
check_firewall() {
    if command -v ufw &>/dev/null && sudo ufw status | grep -q "active"; then
        if ! sudo ufw status | grep -q "$DEFAULT_PORT"; then
            echo ""
            read -p "  Port ${DEFAULT_PORT} may be blocked by ufw. Open it? [Y/n] " -r REPLY
            REPLY=${REPLY:-Y}
            if [[ "$REPLY" =~ ^[Yy]$ ]]; then
                sudo ufw allow "$DEFAULT_PORT"/tcp
                ok "Firewall: port ${DEFAULT_PORT} opened"
            fi
        fi
    fi
}

# Get server IP
get_ip() {
    IP=$(hostname -I 2>/dev/null | awk '{print $1}') || IP=$(ipconfig getifaddr en0 2>/dev/null) || IP="YOUR-SERVER-IP"
}

# Main
main() {
    header

    MODE="${1:-}"

    detect_platform

    # For now, try to use existing binary or inform user
    if command -v tracelog &>/dev/null; then
        ok "TraceLog already installed: $(tracelog version 2>/dev/null || echo 'unknown version')"
    else
        obtain_binary
    fi

    if [ "$MODE" = "agent" ]; then
        echo ""
        read -p "  Hub URL: " HUB_URL
        read -p "  API Key: " API_KEY
        generate_config "agent" "$HUB_URL" "$API_KEY"
    else
        create_user
        create_dirs
        detect_services
        create_admin
        generate_config "serve"
    fi

    create_service
    check_firewall
    get_ip

    echo ""
    echo -e "${CYAN}  ┌──────────────────────────────────────┐${NC}"
    if [ "$MODE" = "agent" ]; then
        echo -e "${CYAN}  │  ${BOLD}Agent connected to hub${NC}${CYAN}               │${NC}"
    else
        echo -e "${CYAN}  │  ${GREEN}${BOLD}Open: http://${IP}:${DEFAULT_PORT}${NC}${CYAN}${NC}"
        echo -e "${CYAN}  │  ${BOLD}Login: admin / ${ADMIN_PASS}${NC}${CYAN}${NC}"
    fi
    echo -e "${CYAN}  └──────────────────────────────────────┘${NC}"
    echo ""
}

main "$@"
