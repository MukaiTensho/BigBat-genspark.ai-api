#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[STEP] Installing/updating container app"
./scripts/start.sh

OS_NAME="$(uname -s)"
if [ "$OS_NAME" = "Darwin" ]; then
  echo "[STEP] Registering launchd service"
  ./scripts/service/install-launchd.sh
  echo "[OK] Bigbat deployed as launchd service"
  echo "[INFO] It will auto-start when this user logs in after reboot"
elif [ "$OS_NAME" = "Linux" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    echo "[STEP] Registering systemd service"
    ./scripts/service/install-systemd.sh
    echo "[OK] Bigbat deployed as systemd service"
    echo "[INFO] It will auto-start on reboot"
  else
    echo "[WARN] systemctl not found, skipped service registration"
    echo "[INFO] Container is still running with docker restart policy"
  fi
else
  echo "[WARN] Unsupported OS for service installer: $OS_NAME"
  echo "[INFO] Container is still running with docker restart policy"
fi

echo "[INFO] Admin UI: http://127.0.0.1:7055/admin/ui"
