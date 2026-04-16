#!/usr/bin/env bash
set -euo pipefail

if ! command -v systemctl >/dev/null 2>&1; then
  echo "[ERROR] systemctl not found"
  exit 1
fi

sudo systemctl disable --now bigbat.service >/dev/null 2>&1 || true
sudo rm -f /etc/systemd/system/bigbat.service
sudo systemctl daemon-reload

echo "[OK] systemd service removed"
