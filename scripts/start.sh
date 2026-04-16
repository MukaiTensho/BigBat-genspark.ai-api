#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] docker is not installed"
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "[ERROR] docker daemon is not running"
  echo "[HINT] start Docker Desktop first"
  exit 1
fi

echo "[STEP] Trying to start existing image without build"
if docker compose up -d --no-build; then
  echo "[OK] started from existing image"
  docker compose ps
  exit 0
fi

echo "[WARN] no existing image, trying normal startup"
docker compose up -d
echo "[OK] started"
docker compose ps
