#!/usr/bin/env bash
# Remove TraceLog (systemd unit, binary, config). Optionally remove data and system user.
# Usage:
#   curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash
#   curl -sSL .../uninstall.sh | sudo bash -s -- --yes   # non-interactive: delete everything
#   TRACELOG_UNINSTALL_YES=1 curl -sSL ... | sudo -E bash

set -euo pipefail

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tracelog"
DATA_DIR="/var/lib/tracelog"
SERVICE_USER="tracelog"
BINARY="${INSTALL_DIR}/tracelog"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

YES=0
for a in "$@"; do
	[[ "$a" == "--yes" ]] && YES=1
done
[[ "${TRACELOG_UNINSTALL_YES:-}" == "1" ]] && YES=1

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo -e "${RED}Run as root, e.g.:${NC}"
	echo "  curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash"
	exit 1
fi

echo -e "${CYAN}TraceLog uninstall${NC}"

if systemctl is-active --quiet tracelog 2>/dev/null; then
	systemctl stop tracelog
	echo "Stopped tracelog service."
fi
if systemctl is-enabled --quiet tracelog 2>/dev/null; then
	systemctl disable tracelog
	echo "Disabled tracelog service."
fi

rm -f /etc/systemd/system/tracelog.service
systemctl daemon-reload 2>/dev/null || true
echo "Removed systemd unit."

# TraceLog nginx site + snippet (from production install.sh)
NGINX_SITE_NAME="tracelog"
rm -f "/etc/nginx/sites-enabled/${NGINX_SITE_NAME}" 2>/dev/null || true
rm -f "/etc/nginx/sites-available/${NGINX_SITE_NAME}" 2>/dev/null || true
rm -f "/etc/nginx/conf.d/${NGINX_SITE_NAME}.conf" 2>/dev/null || true
rm -f /etc/nginx/snippets/tracelog-proxy.conf 2>/dev/null || true
rm -f /etc/nginx/snippets/tracelog-subpath-loc.conf 2>/dev/null || true
rm -f /etc/nginx/conf.d/tracelog-subpath-map.conf 2>/dev/null || true
if command -v nginx &>/dev/null && nginx -t 2>/dev/null; then
	systemctl reload nginx 2>/dev/null || true
	echo "Removed nginx TraceLog site (if present)."
fi

[[ -f "$BINARY" ]] && rm -f "$BINARY" && echo "Removed ${BINARY}."

[[ -d "$CONFIG_DIR" ]] && rm -rf "$CONFIG_DIR" && echo "Removed ${CONFIG_DIR}."

if [[ "$YES" == "1" ]]; then
	rm -rf "$DATA_DIR"
	echo "Removed ${DATA_DIR}."
	if id "$SERVICE_USER" &>/dev/null; then
		userdel "$SERVICE_USER" 2>/dev/null || userdel -f "$SERVICE_USER" 2>/dev/null || true
		echo "Removed system user ${SERVICE_USER}."
	fi
else
	echo ""
	read -r -p "Delete database and all data under ${DATA_DIR}? [y/N] " ans || true
	if [[ "${ans:-}" == "y" || "${ans:-}" == "Y" ]]; then
		rm -rf "$DATA_DIR"
		echo "Removed ${DATA_DIR}."
		if id "$SERVICE_USER" &>/dev/null; then
			userdel "$SERVICE_USER" 2>/dev/null || userdel -f "$SERVICE_USER" 2>/dev/null || true
			echo "Removed system user ${SERVICE_USER}."
		fi
	else
		echo -e "${YELLOW}Data kept at ${DATA_DIR}.${NC}"
		if id "$SERVICE_USER" &>/dev/null; then
			echo -e "${YELLOW}System user ${SERVICE_USER} kept (owns data). Remove later: sudo userdel ${SERVICE_USER}${NC}"
		fi
	fi
fi

echo -e "${GREEN}TraceLog has been removed from this system.${NC}"
