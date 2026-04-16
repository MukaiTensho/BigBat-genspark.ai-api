#!/usr/bin/env bash
set -euo pipefail

if [ "$(uname -s)" = "Darwin" ]; then
  echo "[ERROR] systemd installer is for Linux only"
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "[ERROR] systemctl not found"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
UNIT_PATH="/etc/systemd/system/bigbat.service"

cat <<EOF | sudo tee "$UNIT_PATH" >/dev/null
[Unit]
Description=Bigbat Docker Service
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=oneshot
WorkingDirectory=$ROOT_DIR
ExecStart=$ROOT_DIR/scripts/start.sh
ExecStop=$ROOT_DIR/scripts/uninstall.sh
RemainAfterExit=yes
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now bigbat.service
echo "[OK] systemd service installed: bigbat.service"
