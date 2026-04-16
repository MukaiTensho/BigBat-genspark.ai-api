#!/usr/bin/env bash
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  echo "[ERROR] launchd uninstaller is only for macOS"
  exit 1
fi

PLIST_DST="$HOME/Library/LaunchAgents/com.bigbat.service.plist"

launchctl unload "$PLIST_DST" >/dev/null 2>&1 || true
rm -f "$PLIST_DST"

echo "[OK] launchd service removed"
