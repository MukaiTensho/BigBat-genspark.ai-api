#!/usr/bin/env bash
set -euo pipefail

# Big Bat -> Claude Code launcher
# Usage:
#   BIGBAT_BASE_URL="http://100.92.199.24:7055" BIGBAT_API_KEY="123456" ./scripts/start-claudecode.sh
#   ./scripts/start-claudecode.sh --help

if [[ "${1:-}" == "--help" ]]; then
  cat <<'EOF'
Launch Claude Code with Big Bat Anthropic-compatible endpoint.

Environment variables:
  BIGBAT_BASE_URL   Base URL of Big Bat (default: http://100.92.199.24:7055)
  BIGBAT_API_KEY    API key for Big Bat (required if ANTHROPIC_API_KEY is unset)
  BIGBAT_MODEL      Model name (default: opus4.6)

Examples:
  BIGBAT_BASE_URL="http://192.168.31.135:7055" BIGBAT_API_KEY="123456" ./scripts/start-claudecode.sh
  BIGBAT_API_KEY="123456" BIGBAT_MODEL="claude-opus-4-6" ./scripts/start-claudecode.sh
EOF
  exit 0
fi

BASE_URL="${BIGBAT_BASE_URL:-http://100.92.199.24:7055}"
MODEL="${BIGBAT_MODEL:-opus4.6}"
API_KEY="${BIGBAT_API_KEY:-${ANTHROPIC_API_KEY:-}}"

if [[ -z "$API_KEY" ]]; then
  echo "[ERROR] Missing API key. Set BIGBAT_API_KEY or ANTHROPIC_API_KEY."
  exit 1
fi

# Remove conflicting auth-token mode.
unset ANTHROPIC_AUTH_TOKEN

export ANTHROPIC_BASE_URL="$BASE_URL"
export ANTHROPIC_API_KEY="$API_KEY"
export ANTHROPIC_MODEL="$MODEL"

echo "[INFO] ANTHROPIC_BASE_URL=$ANTHROPIC_BASE_URL"
echo "[INFO] ANTHROPIC_MODEL=$ANTHROPIC_MODEL"

if command -v curl >/dev/null 2>&1; then
  if ! curl -sS "$ANTHROPIC_BASE_URL/v1/messages/health" -H "x-api-key: $ANTHROPIC_API_KEY" >/dev/null 2>&1; then
    echo "[WARN] Health probe request failed. Claude Code may still start but could error."
  fi
fi

if command -v claude >/dev/null 2>&1; then
  exec claude "$@"
fi

if command -v claude-code >/dev/null 2>&1; then
  exec claude-code "$@"
fi

echo "[ERROR] Neither 'claude' nor 'claude-code' command found in PATH."
echo "[HINT] Install Claude Code CLI first, then rerun this script."
exit 1
