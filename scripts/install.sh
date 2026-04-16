#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] docker is not installed"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  echo "[ERROR] docker compose is required"
  exit 1
fi

if [ ! -f ".env" ]; then
  cp .env.example .env
  echo "[INFO] .env created from .env.example"
  echo "[INFO] please edit .env and set GS_COOKIE before first use"
fi

# Ensure GS_COOKIE is safely quoted and protected.
python3 - <<'PY'
from pathlib import Path
p = Path('.env')
if not p.exists():
    raise SystemExit(0)
lines = p.read_text(encoding='utf-8').splitlines()
out = []
i = 0
while i < len(lines):
    line = lines[i]
    if line.startswith('GS_COOKIE='):
        k, v = line.split('=', 1)
        while i + 1 < len(lines):
            nxt = lines[i + 1]
            if not nxt or nxt.lstrip().startswith('#') or '=' in nxt:
                break
            i += 1
            v += '\n' + lines[i]
        v = v.replace('$$', '__DOLLAR__').replace('$', '$$').replace('__DOLLAR__', '$$')
        escaped = v.replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n')
        out.append(f'{k}="{escaped}"')
    else:
        out.append(line)
    i += 1
p.write_text('\n'.join(out) + '\n', encoding='utf-8')
PY

docker compose up -d --build
echo "[OK] bigbat installed and started"
echo "[INFO] admin ui: http://127.0.0.1:7055/admin/ui"
