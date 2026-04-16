#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  docker compose down --remove-orphans || true
fi

docker image rm -f bigbat-bigbat:latest >/dev/null 2>&1 || true
docker network rm bigbat_default >/dev/null 2>&1 || true

if [ "${1:-}" = "--purge-env" ]; then
  rm -f .env
  echo "[INFO] .env removed"
fi

echo "[OK] bigbat uninstalled"
