#!/usr/bin/env bash
# TraceLog installer: (1) GitHub release tarball (2) system go install (3) download official Go from go.dev, then go install.
# Override bootstrap Go versions: TRACELOG_BOOTSTRAP_GO="1.24.3 1.23.5"
# Uninstall: curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash
#
# Production (Linux hub install, default):
#   - tracelog listens on 127.0.0.1:8090 only
#   - nginx reverse proxy on port 80 (and HTTPS via Let's Encrypt when configured)
# Opt out (dev / direct port): TRACELOG_NO_PROXY=1
# HTTPS (optional): set DNS A/AAAA to this host, then e.g.:
#   sudo TRACELOG_DOMAIN=monitor.example.com TRACELOG_LETSENCRYPT_EMAIL=you@example.com bash scripts/install.sh
#
# Subpath on an existing site (e.g. https://cadourile.ro/tracelog/):
#   sudo TRACELOG_URL_PREFIX=/tracelog TRACELOG_NGINX_SITE=cadourile.ro bash scripts/install.sh
# Optional: TRACELOG_PUBLIC_DOMAIN=cadourile.ro (for installer banner; defaults to site file name if it looks like a hostname)
# With TRACELOG_NGINX_SITE, the installer tries to add inside the 443 server { }:
#   include /etc/nginx/snippets/tracelog-subpath-loc.conf;
# Otherwise add that line manually to your SSL vhost.
set -euo pipefail

# Set by try_release_download when a release is found; default for messages only.
REPO="tudorAbrudan/tracelog"
BINARY_NAME="tracelog"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tracelog"
DATA_DIR="/var/lib/tracelog"
SERVICE_USER="tracelog"
DEFAULT_PORT=8090
NGINX_SITE_NAME="tracelog"

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

# Create admin user and generate password (same DB path as systemd via TRACELOG_DATA_DIR)
create_admin() {
    info "Creating admin account..."
    local out ec
    set +e
    out=$(sudo -u "$SERVICE_USER" env TRACELOG_DATA_DIR="$DATA_DIR" HOME="$DATA_DIR" "${INSTALL_DIR}/${BINARY_NAME}" user create admin 2>&1)
    ec=$?
    set -e
    ADMIN_PASS=$(printf '%s\n' "$out" | grep "Password:" | head -1 | awk '{print $2}')
    if [ -z "$ADMIN_PASS" ]; then
        if [ "$ec" -ne 0 ]; then
            warn "$out"
        fi
        warn "Could not capture admin password (user may already exist)."
        warn "Use the web UI (first-time setup) or run:"
        warn "  sudo -u ${SERVICE_USER} env TRACELOG_DATA_DIR=${DATA_DIR} ${INSTALL_DIR}/${BINARY_NAME} user reset-password admin"
    fi
}

# Generate config file (serve mode uses BIND_ADDRESS: 127.0.0.1 behind nginx, or 0.0.0.0 if TRACELOG_NO_PROXY=1)
generate_config() {
    local mode="${1:-serve}"
    local hub_url="${2:-}"
    local api_key="${3:-}"

    if [ "$mode" = "serve" ]; then
        local url_line=""
        if [ -n "${URL_PREFIX:-}" ]; then
            url_line="url_path_prefix: ${URL_PREFIX}"$'\n'
        fi
        cat <<YAML | sudo tee "${CONFIG_DIR}/config.yaml" > /dev/null
mode: serve
port: ${DEFAULT_PORT}
bind_address: "${BIND_ADDRESS}"
data_dir: ${DATA_DIR}
${url_line}collect:
  interval_seconds: 10
  system: true
  docker: ${DOCKER_FOUND:-false}
YAML
    else
        cat <<YAML | sudo tee "${CONFIG_DIR}/config.yaml" > /dev/null
mode: agent
hub_url: "${hub_url}"
api_key: "${api_key}"
collect:
  interval_seconds: 10
  system: true
  docker: ${DOCKER_FOUND:-false}
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

    local bind_flag=""
    if [ "${BIND_ADDRESS}" = "127.0.0.1" ]; then
        bind_flag=" --bind 127.0.0.1"
    fi

    cat <<UNIT | sudo tee /etc/systemd/system/tracelog.service > /dev/null
[Unit]
Description=TraceLog Server Monitoring
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=${SERVICE_USER}
Environment=TRACELOG_DATA_DIR=${DATA_DIR}
$( [ -n "${URL_PREFIX:-}" ] && echo "Environment=TRACELOG_URL_PREFIX=${URL_PREFIX}" )
ExecStart=${INSTALL_DIR}/${BINARY_NAME} serve --port ${DEFAULT_PORT}${bind_flag}$( [ -n "${URL_PREFIX:-}" ] && echo " --url-prefix ${URL_PREFIX}" )
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
    if ! command -v ufw &>/dev/null || ! sudo ufw status 2>/dev/null | grep -q "active"; then
        return 0
    fi
    if [ "${PROD_PROXY:-0}" = "1" ]; then
        local need=()
        sudo ufw status | grep -qE '(^| )80/tcp' || need+=("80/tcp")
        sudo ufw status | grep -qE '(^| )443/tcp' || need+=("443/tcp")
        if [ "${#need[@]}" -gt 0 ]; then
            echo ""
            read -p "  ufw is active; open HTTP/HTTPS (${need[*]}) for nginx? [Y/n] " -r REPLY
            REPLY=${REPLY:-Y}
            if [[ "$REPLY" =~ ^[Yy]$ ]]; then
                for p in "${need[@]}"; do
                    sudo ufw allow "$p"
                done
                ok "Firewall: ${need[*]} allowed (TraceLog stays on 127.0.0.1:${DEFAULT_PORT})"
            fi
        fi
        return 0
    fi
    if ! sudo ufw status | grep -q "$DEFAULT_PORT"; then
        echo ""
        read -p "  Port ${DEFAULT_PORT} may be blocked by ufw. Open it? [Y/n] " -r REPLY
        REPLY=${REPLY:-Y}
        if [[ "$REPLY" =~ ^[Yy]$ ]]; then
            sudo ufw allow "$DEFAULT_PORT"/tcp
            ok "Firewall: port ${DEFAULT_PORT} opened"
        fi
    fi
}

# Install nginx via package manager (best effort)
pkg_install_nginx() {
    if command -v nginx &>/dev/null; then
        return 0
    fi
    info "Installing nginx..."
    if command -v apt-get &>/dev/null; then
        sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq
        sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nginx
    elif command -v dnf &>/dev/null; then
        sudo dnf install -y nginx
    elif command -v yum &>/dev/null; then
        sudo yum install -y nginx
    elif command -v zypper &>/dev/null; then
        sudo zypper install -y nginx
    else
        warn "Could not install nginx automatically; install nginx, then re-run this script or add a site manually."
        return 1
    fi
    ok "nginx installed"
}

# Write nginx site for TraceLog (WebSocket-safe map)
write_nginx_site() {
    local domain="${TRACELOG_DOMAIN:-}"
    local use_default_server=""
    local server_names="_"

    if [ -n "$domain" ]; then
        server_names="$domain"
        use_default_server=""
    else
        use_default_server=" default_server"
    fi

    local listen_v4="listen 80${use_default_server};"
    local listen_v6
    if [ -z "$domain" ]; then
        listen_v6="listen [::]:80${use_default_server};"
    else
        listen_v6="listen [::]:80;"
    fi

    sudo mkdir -p /etc/nginx/snippets
    sudo tee /etc/nginx/snippets/tracelog-proxy.conf > /dev/null <<'SNIP'
proxy_pass http://127.0.0.1:8090;
proxy_http_version 1.1;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $connection_upgrade;
proxy_read_timeout 86400;
proxy_send_timeout 86400;
SNIP

    if [ -d /etc/nginx/sites-available ] && [ -d /etc/nginx/sites-enabled ]; then
        sudo tee "/etc/nginx/sites-available/${NGINX_SITE_NAME}" > /dev/null <<EOF
map \$http_upgrade \$connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    ${listen_v4}
    ${listen_v6}
    server_name ${server_names};

    location / {
        include snippets/tracelog-proxy.conf;
    }
}
EOF
        sudo ln -sf "/etc/nginx/sites-available/${NGINX_SITE_NAME}" "/etc/nginx/sites-enabled/${NGINX_SITE_NAME}"
        if [ -z "$domain" ] && [ -f /etc/nginx/sites-enabled/default ]; then
            sudo rm -f /etc/nginx/sites-enabled/default
            info "Disabled default nginx site so TraceLog is the port 80 default (restore from /etc/nginx/sites-available/default if needed)."
        fi
    else
        if [ -z "$domain" ] && [ -f /etc/nginx/conf.d/default.conf ]; then
            sudo mv /etc/nginx/conf.d/default.conf "/etc/nginx/conf.d/default.conf.bak-tracelog-$(date +%s)" 2>/dev/null || true
            info "Renamed default /etc/nginx/conf.d/default.conf so TraceLog can bind port 80 (backup with .bak-tracelog- prefix)."
        fi
        sudo tee "/etc/nginx/conf.d/${NGINX_SITE_NAME}.conf" > /dev/null <<EOF
map \$http_upgrade \$connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    ${listen_v4}
    ${listen_v6}
    server_name ${server_names};

    location / {
        include snippets/tracelog-proxy.conf;
    }
}
EOF
    fi
}

# Optional: Let's Encrypt when domain + email are set
maybe_certbot() {
    local domain="${TRACELOG_DOMAIN:-}"
    local email="${TRACELOG_LETSENCRYPT_EMAIL:-}"
    if [ -z "$domain" ] || [ -z "$email" ]; then
        if [ -n "$domain" ] && [ -z "$email" ]; then
            warn "TRACELOG_DOMAIN is set but TRACELOG_LETSENCRYPT_EMAIL is empty — skipping certbot. Set the email and re-run, or run: sudo certbot --nginx -d ${domain}"
        fi
        return 0
    fi
    if [ ! -d "/etc/letsencrypt/live/${domain}" ]; then
        info "Requesting TLS certificate (certbot)..."
        if ! command -v certbot &>/dev/null; then
            if command -v apt-get &>/dev/null; then
                sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq certbot python3-certbot-nginx
            elif command -v dnf &>/dev/null; then
                sudo dnf install -y certbot python3-certbot-nginx
            elif command -v yum &>/dev/null; then
                sudo yum install -y certbot python3-certbot-nginx
            else
                warn "certbot not found; install certbot with the nginx plugin, then run: sudo certbot --nginx -d ${domain}"
                return 0
            fi
        fi
        if sudo certbot --nginx -d "$domain" --non-interactive --agree-tos -m "$email" --redirect; then
            ok "HTTPS enabled for ${domain}"
        else
            warn "certbot failed (DNS must point here first). HTTP via nginx still works."
        fi
    else
        ok "Existing Let's Encrypt cert for ${domain} — run sudo certbot renew if needed"
    fi
}

# WebSocket map in conf.d (http context) + location snippet for server { } include
write_nginx_subpath_snippets() {
    sudo mkdir -p /etc/nginx/snippets /etc/nginx/conf.d
    sudo tee /etc/nginx/conf.d/tracelog-subpath-map.conf > /dev/null <<'MAP'
map $http_upgrade $tracelog_conn {
    default upgrade;
    '' close;
}
MAP
    sudo tee /etc/nginx/snippets/tracelog-subpath-loc.conf > /dev/null <<EOF
location ${URL_PREFIX}/ {
    proxy_pass http://127.0.0.1:${DEFAULT_PORT}/;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
    proxy_set_header Upgrade \$http_upgrade;
    proxy_set_header Connection \$tracelog_conn;
    proxy_read_timeout 86400;
    proxy_send_timeout 86400;
}
EOF
    ok "Wrote /etc/nginx/conf.d/tracelog-subpath-map.conf and /etc/nginx/snippets/tracelog-subpath-loc.conf"
}

# Insert include into the HTTPS (listen …443) server block matching the site name.
# $1 = vhost filename (e.g. cadourile.ro) under sites-enabled or sites-available.
inject_nginx_subpath_include_into_vhost() {
    local site="$1"
    local f=""
    for d in /etc/nginx/sites-enabled /etc/nginx/sites-available; do
        if [ -f "$d/$site" ]; then
            f="$d/$site"
            break
        fi
    done
    if [ -z "$f" ]; then
        warn "No nginx vhost file named $site under sites-enabled or sites-available."
        return 1
    fi

    if sudo grep -qF 'tracelog-subpath-loc.conf' "$f"; then
        ok "nginx vhost $f already includes TraceLog subpath snippet"
        return 0
    fi

    local host_label="${TRACELOG_PUBLIC_DOMAIN:-$site}"
    local host_grep_re
    host_grep_re=$(printf '%s' "$host_label" | sed 's/\./\\./g')

    local inc_line='    include /etc/nginx/snippets/tracelog-subpath-loc.conf;'
    local bak="${f}.bak-tracelog-$(date +%s)"
    sudo cp -a "$f" "$bak" || return 1

    local tmp
    tmp=$(mktemp)
    local inserted=0
    local found_listen=0
    local lines_after_listen=0

    while IFS= read -r line || [ -n "$line" ]; do
        if echo "$line" | grep -qE 'listen[[:space:]]+(\[::\]:)?443([^0-9]|$)'; then
            found_listen=1
            lines_after_listen=0
        fi
        if [ "$found_listen" = 1 ]; then
            lines_after_listen=$((lines_after_listen + 1))
        fi
        printf '%s\n' "$line"
        if [ "$inserted" = 0 ] && [ "$found_listen" = 1 ] && echo "$line" | grep -q 'server_name' && echo "$line" | grep -qE "$host_grep_re"; then
            printf '%s\n' "$inc_line"
            inserted=1
            found_listen=0
        fi
        if [ "$found_listen" = 1 ] && [ "$lines_after_listen" -gt 48 ]; then
            found_listen=0
        fi
    done < <(sudo cat "$f") > "$tmp"

    if [ "$inserted" != 1 ]; then
        warn "Could not auto-insert include (no listen 443 + server_name matching ${host_label} in $f)."
        rm -f "$tmp"
        sudo rm -f "$bak"
        return 1
    fi

    sudo mv "$tmp" "$f" || return 1
    ok "Inserted TraceLog include into $f (backup: $bak)"
    if sudo nginx -t 2>/dev/null; then
        sudo systemctl reload nginx 2>/dev/null || sudo systemctl restart nginx
        ok "nginx reloaded"
        return 0
    fi
    warn "nginx -t failed after edit; restoring $f from backup"
    sudo mv "$bak" "$f"
    sudo nginx -t || true
    return 1
}

# Subpath behind existing vhost (TRACELOG_URL_PREFIX=/tracelog)
setup_nginx_subpath_only() {
    if [ "$OS" != "linux" ] || [ -z "${URL_PREFIX:-}" ]; then
        return 0
    fi
    info "Subpath mode: TraceLog at ${URL_PREFIX}/ (nginx snippets + systemd URL prefix)..."
    if ! pkg_install_nginx; then
        return 1
    fi
    write_nginx_subpath_snippets

    local inject_ok=0
    if [ -n "${TRACELOG_NGINX_SITE:-}" ]; then
        if inject_nginx_subpath_include_into_vhost "${TRACELOG_NGINX_SITE}"; then
            inject_ok=1
        fi
    fi

    if sudo nginx -t; then
        sudo systemctl enable nginx 2>/dev/null || true
        if [ "$inject_ok" != 1 ]; then
            sudo systemctl reload nginx 2>/dev/null || sudo systemctl restart nginx
            ok "nginx loaded TraceLog subpath map (conf.d)"
        fi
    else
        warn "nginx -t failed; fix config then: sudo nginx -t && sudo systemctl reload nginx"
        return 1
    fi

    if [ "$inject_ok" != 1 ]; then
        echo ""
        warn "Add this line inside your HTTPS server { } for your site (after server_name is fine):"
        info "  include /etc/nginx/snippets/tracelog-subpath-loc.conf;"
        warn "Or re-run with: TRACELOG_NGINX_SITE=cadourile.ro (vhost filename under sites-enabled)"
        warn "Then: sudo nginx -t && sudo systemctl reload nginx"
    fi
}

# Production reverse proxy: nginx → 127.0.0.1:8090 (whole host, not subpath)
setup_nginx_standalone_hub() {
    if [ "${PROD_PROXY:-0}" != "1" ]; then
        return 0
    fi
    if [ "$OS" != "linux" ]; then
        return 0
    fi
    info "Configuring nginx reverse proxy (production)..."
    if ! pkg_install_nginx; then
        return 1
    fi
    write_nginx_site
    if sudo nginx -t; then
        sudo systemctl enable nginx 2>/dev/null || true
        sudo systemctl restart nginx
        ok "nginx serving TraceLog"
    else
        warn "nginx configuration test failed; fix /etc/nginx and run: sudo nginx -t && sudo systemctl reload nginx"
        return 1
    fi
    maybe_certbot
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

    PROD_PROXY=0
    BIND_ADDRESS="0.0.0.0"
    URL_PREFIX=""
    if [ -n "${TRACELOG_URL_PREFIX:-}" ]; then
        local p
        p=$(echo "${TRACELOG_URL_PREFIX}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [ -n "$p" ] && [ "$p" != "/" ]; then
            case "$p" in
                /*) ;;
                *) p="/$p" ;;
            esac
            p="${p%/}"
            URL_PREFIX="$p"
        fi
    fi

    if [ "$OS" = "linux" ] && [ "$MODE" != "agent" ]; then
        if [ "${TRACELOG_NO_PROXY:-}" != "1" ]; then
            PROD_PROXY=1
            BIND_ADDRESS="127.0.0.1"
        fi
    fi

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
    if [ "$OS" = "linux" ] && [ "$MODE" != "agent" ]; then
        if [ -n "${URL_PREFIX:-}" ]; then
            setup_nginx_subpath_only || warn "Subpath nginx snippets incomplete — see messages above."
        elif [ "${PROD_PROXY}" = "1" ]; then
            setup_nginx_standalone_hub || warn "nginx setup incomplete — TraceLog is on http://127.0.0.1:${DEFAULT_PORT} only until nginx is fixed."
        fi
    fi
    check_firewall
    get_ip

    local pub_host=""
    pub_host="${TRACELOG_PUBLIC_DOMAIN:-}"
    if [ -z "$pub_host" ] && [ -n "${TRACELOG_NGINX_SITE:-}" ] && [[ "${TRACELOG_NGINX_SITE}" == *.* ]]; then
        pub_host="${TRACELOG_NGINX_SITE}"
    fi
    if [ -z "$pub_host" ] && [ -n "${TRACELOG_DOMAIN:-}" ]; then
        pub_host="${TRACELOG_DOMAIN}"
    fi

    echo ""
    echo -e "${CYAN}  ┌──────────────────────────────────────┐${NC}"
    if [ "$MODE" = "agent" ]; then
        echo -e "${CYAN}  │  ${BOLD}Agent connected to hub${NC}${CYAN}               │${NC}"
    else
        local url_msg=""
        if [ -n "${URL_PREFIX:-}" ]; then
            if [ -n "$pub_host" ]; then
                echo -e "${CYAN}  │  ${GREEN}${BOLD}Open: https://${pub_host}${URL_PREFIX}/${NC}${CYAN}"
                echo -e "${CYAN}  │  ${BOLD}Hub URL for agents: https://${pub_host}${URL_PREFIX}${NC}${CYAN}"
            else
                echo -e "${CYAN}  │  ${GREEN}${BOLD}Open: https://YOUR_DOMAIN${URL_PREFIX}/${NC}${CYAN}"
                echo -e "${CYAN}  │  ${BOLD}Set TRACELOG_PUBLIC_DOMAIN or TRACELOG_NGINX_SITE=cadourile.ro for a concrete URL.${NC}${CYAN}"
                echo -e "${CYAN}  │  ${BOLD}Hub URL for agents: https://YOUR_DOMAIN${URL_PREFIX}${NC}${CYAN}"
            fi
        elif [ "${PROD_PROXY}" = "1" ]; then
            if [ -n "${TRACELOG_DOMAIN:-}" ] && [ -f "/etc/letsencrypt/live/${TRACELOG_DOMAIN}/fullchain.pem" ]; then
                url_msg="https://${TRACELOG_DOMAIN}/"
            elif [ -n "${TRACELOG_DOMAIN:-}" ]; then
                url_msg="http://${TRACELOG_DOMAIN}/ (enable HTTPS: set TRACELOG_LETSENCRYPT_EMAIL and re-run install, or certbot --nginx)"
            else
                url_msg="http://${IP}/"
            fi
            echo -e "${CYAN}  │  ${GREEN}${BOLD}Open: ${url_msg}${NC}${CYAN}${NC}"
            echo -e "${CYAN}  │  ${BOLD}(TraceLog listens on 127.0.0.1:${DEFAULT_PORT} — use nginx only.)${NC}${CYAN}"
        else
            echo -e "${CYAN}  │  ${GREEN}${BOLD}Open: http://${IP}:${DEFAULT_PORT}${NC}${CYAN}${NC}"
        fi
        if [ -n "${ADMIN_PASS:-}" ]; then
            echo -e "${CYAN}  │  ${BOLD}Login: admin / ${ADMIN_PASS}${NC}${CYAN}${NC}"
        else
            echo -e "${CYAN}  │  ${YELLOW}Login: use web setup (no users yet) or reset admin password (see warnings above).${NC}${CYAN}"
        fi
    fi
    echo -e "${CYAN}  └──────────────────────────────────────┘${NC}"
    echo ""
}

main "$@"
