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
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] go is required for offline rebuild"
  exit 1
fi

if ! docker image inspect bigbat-bigbat:latest >/dev/null 2>&1; then
  echo "[ERROR] local image bigbat-bigbat:latest not found"
  echo "[HINT] first successful build must be done online"
  exit 1
fi

TMP_DIR="$ROOT_DIR/.tmp"
BIN_PATH="$TMP_DIR/bigbat-linux"
mkdir -p "$TMP_DIR"

GOARCH_VALUE="$(go env GOARCH)"
echo "[STEP] Building local linux binary (GOARCH=$GOARCH_VALUE)"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH_VALUE" go build -o "$BIN_PATH" ./cmd/bigbat

TMP_CONTAINER="bigbat-offline-$$"
cleanup() {
  docker rm -f "$TMP_CONTAINER" >/dev/null 2>&1 || true
  rm -f "$BIN_PATH"
}
trap cleanup EXIT

echo "[STEP] Updating image binary"
docker create --name "$TMP_CONTAINER" bigbat-bigbat:latest >/dev/null
docker cp "$BIN_PATH" "$TMP_CONTAINER":/app/bigbat
docker commit "$TMP_CONTAINER" bigbat-bigbat:latest >/dev/null

echo "[STEP] Restarting service"
docker compose up -d --no-build
docker compose ps

echo "[OK] Offline rebuild done"
