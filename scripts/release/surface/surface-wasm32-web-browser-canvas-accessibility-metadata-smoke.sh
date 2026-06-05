#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh [--report-dir DIR]

Runs wasm32-web browser-canvas Tetra Surface accessibility metadata smoke.
The report proves tetra.surface.accessibility-tree.v1 metadata-tree evidence
in a real browser canvas host and rejects Node-only, DOM/ARIA, user-JS,
platform widget, and legacy sidecar promotion evidence.
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
report_path="$report_dir/surface-wasm32-web-browser-canvas-accessibility-metadata.json"

go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-accessibility-metadata --source examples/surface_accessibility_settings.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface wasm32-web browser-canvas accessibility metadata runtime smoke report: $report_path"
echo "Surface wasm32-web browser-canvas accessibility metadata artifact hashes: $report_dir/artifact-hashes.json"
