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

echo "[STEP] Trying normal rebuild via Docker Hub"
if docker compose build && docker compose up -d; then
  echo "[OK] Rebuild succeeded via Docker Hub"
  docker compose ps
  exit 0
fi

echo "[WARN] Online rebuild failed, trying offline hot-rebuild"

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] go is not installed; cannot perform offline rebuild"
  exit 1
fi

if ! docker image inspect bigbat-bigbat:latest >/dev/null 2>&1; then
  echo "[ERROR] local image bigbat-bigbat:latest not found"
  echo "[HINT] wait for Docker Hub network recovery, then run ./scripts/rebuild.sh again"
  exit 1
fi

TMP_DIR="$ROOT_DIR/.tmp"
BIN_PATH="$TMP_DIR/bigbat-linux"
mkdir -p "$TMP_DIR"

GOARCH_VALUE="$(go env GOARCH)"
echo "[STEP] Building linux binary locally (GOARCH=$GOARCH_VALUE)"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH_VALUE" go build -o "$BIN_PATH" ./cmd/bigbat

TMP_CONTAINER="bigbat-hotrebuild-$$"
cleanup() {
  docker rm -f "$TMP_CONTAINER" >/dev/null 2>&1 || true
  rm -f "$BIN_PATH"
}
trap cleanup EXIT

echo "[STEP] Patching binary into existing image"
docker create --name "$TMP_CONTAINER" bigbat-bigbat:latest >/dev/null
docker cp "$BIN_PATH" "$TMP_CONTAINER":/app/bigbat
docker commit "$TMP_CONTAINER" bigbat-bigbat:latest >/dev/null

echo "[STEP] Restarting service with patched image"
docker compose up -d --no-build
docker compose ps

echo "[OK] Offline hot-rebuild completed"
