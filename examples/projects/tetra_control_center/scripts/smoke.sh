#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$PROJECT_ROOT/../../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/build"
PORT="${TCC_SMOKE_PORT:-8766}"
AUDIT_LOG="${TCC_SMOKE_AUDIT:-/tmp/tetra-control-center-smoke-audit.jsonl}"

cd "$REPO_ROOT"

python3 -m unittest discover -s examples/projects/tetra_control_center/tests -v
./tetra check examples/projects/tetra_control_center
mkdir -p "$BUILD_DIR"
./tetra build -target wasm32-web -o "$BUILD_DIR/tetra_control_center.wasm" examples/projects/tetra_control_center

python3 examples/projects/tetra_control_center/backend/tcc_backend.py \
  --snapshot \
  --audit-log "$AUDIT_LOG" \
  > /tmp/tetra-control-center-snapshot.json

python3 - <<'PY'
import json
from pathlib import Path

snapshot = json.loads(Path("/tmp/tetra-control-center-snapshot.json").read_text())
assert snapshot["dashboard"]["cpu"]["status"] in {"supported", "unsupported"}
assert "DREAM MACHINES" in snapshot["hardware"]["dmi"]["sys_vendor"]
assert snapshot["profiles"]["available"] == ["quiet", "balanced", "performance", "custom"]
assert snapshot["fans"]["control"]["status"] in {"supported", "unsupported"}
assert "diagnostics" in snapshot
PY

(
  cd "$PROJECT_ROOT"
  python3 -m backend.tcc_backend \
  --host 127.0.0.1 \
  --port "$PORT" \
  --project-root "$PROJECT_ROOT" \
    --audit-log "$AUDIT_LOG"
) &
SERVER_PID=$!
trap 'kill "$SERVER_PID" >/dev/null 2>&1 || true' EXIT

for _ in $(seq 1 50); do
  if curl -fsS "http://127.0.0.1:$PORT/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

curl -fsS "http://127.0.0.1:$PORT/api/snapshot" >/dev/null
curl -fsS -X POST "http://127.0.0.1:$PORT/api/profile" \
  -H 'Content-Type: application/json' \
  --data '{"profile":"quiet","dry_run":true}' \
  >/dev/null

if command -v chromium >/dev/null 2>&1; then
  for screen in Dashboard Profiles "Fans/Backends" Diagnostics Logs Settings; do
    encoded="${screen//\//%2F}"
    chromium --headless --disable-gpu --no-sandbox --virtual-time-budget=5000 \
      --dump-dom "http://127.0.0.1:$PORT/?screen=$encoded" \
      | grep -F "<h2>$screen</h2>" >/dev/null
  done
else
  echo "chromium not found; skipped browser DOM smoke"
fi

echo "tetra-control-center smoke ok"
