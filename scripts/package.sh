#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PROJECT_NAME="bigbat"
VERSION="$(date +%Y%m%d-%H%M%S)"
DIST_DIR="$ROOT_DIR/dist"
STAGE_DIR="$DIST_DIR/${PROJECT_NAME}-${VERSION}"

mkdir -p "$DIST_DIR"
rm -rf "$STAGE_DIR"
mkdir -p "$STAGE_DIR"

copy_if_exists() {
  local src="$1"
  local dst="$2"
  if [ -e "$src" ]; then
    cp -R "$src" "$dst"
  fi
}

copy_if_exists "$ROOT_DIR/cmd" "$STAGE_DIR/"
copy_if_exists "$ROOT_DIR/internal" "$STAGE_DIR/"
copy_if_exists "$ROOT_DIR/scripts" "$STAGE_DIR/"

for file in \
  "README.md" \
  "DEPLOY.md" \
  "go.mod" \
  "Dockerfile" \
  "docker-compose.yml" \
  ".env.example" \
  ".gitignore"; do
  if [ -f "$ROOT_DIR/$file" ]; then
    cp "$ROOT_DIR/$file" "$STAGE_DIR/$file"
  fi
done

cat > "$STAGE_DIR/QUICKSTART.md" <<'EOF'
# QUICKSTART

## 1) Edit env

```bash
cp .env.example .env
```

Set at least:

- `GS_COOKIE=...`
- `API_SECRET=...` (optional)

## 2) One-command deploy as service

```bash
./scripts/deploy-service.sh
```

## 3) Open admin ui

`http://127.0.0.1:7055/admin/ui`

## 4) Remove service and app

```bash
./scripts/remove-service.sh
```

Purge `.env` too:

```bash
./scripts/remove-service.sh --purge-env
```
EOF

tar -czf "$DIST_DIR/${PROJECT_NAME}-${VERSION}.tar.gz" -C "$DIST_DIR" "${PROJECT_NAME}-${VERSION}"

echo "[OK] package created: $DIST_DIR/${PROJECT_NAME}-${VERSION}.tar.gz"
