#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

OS_NAME="$(uname -s)"
if [ "$OS_NAME" = "Darwin" ]; then
  ./scripts/service/uninstall-launchd.sh || true
elif [ "$OS_NAME" = "Linux" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    ./scripts/service/uninstall-systemd.sh || true
  fi
fi

if [ "${1:-}" = "--purge-env" ]; then
  ./scripts/uninstall.sh --purge-env
else
  ./scripts/uninstall.sh
fi

echo "[OK] Bigbat service and app removed"
