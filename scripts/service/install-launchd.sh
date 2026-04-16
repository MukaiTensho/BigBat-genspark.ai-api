#!/usr/bin/env bash
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  echo "[ERROR] launchd installer is only for macOS"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
PLIST_SRC="$ROOT_DIR/scripts/service/com.bigbat.service.plist"
PLIST_DST="$HOME/Library/LaunchAgents/com.bigbat.service.plist"

if [ ! -f "$PLIST_SRC" ]; then
  echo "[ERROR] missing plist template: $PLIST_SRC"
  exit 1
fi

mkdir -p "$HOME/Library/LaunchAgents"

sed "s#__BIGBAT_ROOT__#$ROOT_DIR#g" "$PLIST_SRC" > "$PLIST_DST"

launchctl unload "$PLIST_DST" >/dev/null 2>&1 || true
launchctl load "$PLIST_DST"

echo "[OK] launchd service installed: com.bigbat.service"
echo "[INFO] check status: launchctl list | grep bigbat"
