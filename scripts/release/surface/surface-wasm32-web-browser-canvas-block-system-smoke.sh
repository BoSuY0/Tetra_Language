#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface-block/wasm32-web-browser-canvas"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-block-system-smoke.sh [--report-dir DIR]

Runs wasm32-web browser-canvas Tetra Surface Block-system smoke.
The report validates tetra.surface.block-system.v1 through a compiler-owned loader
and browser canvas RGBA readback, records browser canvas input evidence,
requires no user JS and no DOM UI sidecars, and rejects Node-only promotion.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --report-dir requires a value" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
report_path="$report_dir/surface-block-system-wasm32-web.json"
blocked_path="$report_dir/surface-block-system-wasm32-web.blocked.json"
wasm_path="$report_dir/surface-wasm32-web-browser-canvas-block-system-artifacts/surface-block-system.wasm"

browser_runner=""
for candidate in chromium chromium-browser google-chrome chrome; do
  if command -v "$candidate" >/dev/null 2>&1; then
    browser_runner="$candidate"
    break
  fi
done

if [[ -z "$browser_runner" ]]; then
  cat > "$blocked_path" <<BLOCKED
{
  "schema": "tetra.surface.block-system.blocked.v1",
  "target": "wasm32-web",
  "runtime": "surface-wasm32-web",
  "status": "blocked",
  "reason": "browser-canvas runner unavailable; Node-only evidence is not accepted for wasm32-web Block-system production claims"
}
BLOCKED
  echo "Surface wasm32-web browser-canvas Block-system smoke blocked: $blocked_path" >&2
  exit 1
fi

go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system --source examples/surface_block_system.tetra --report "$report_path"
go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$wasm_path"
go run ./tools/cmd/validate-surface-block-report --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface wasm32-web browser-canvas Block-system runtime smoke report: $report_path"
echo "Surface wasm32-web browser-canvas Block-system artifact hashes: $report_dir/artifact-hashes.json"
