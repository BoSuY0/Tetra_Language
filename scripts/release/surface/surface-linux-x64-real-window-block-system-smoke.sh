#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface-block/linux-x64-real-window"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-real-window-block-system-smoke.sh [--report-dir DIR]

Runs linux-x64 real Linux window Tetra Surface Block-system smoke.
The report validates tetra.surface.block-system.v1 through Wayland shm RGBA
frame presentation, native input/state evidence, checksums, and honest
blocked status when a Wayland display host is unavailable.
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
report_path="$report_dir/surface-block-system-linux-x64.json"
blocked_path="$report_dir/surface-block-system-linux-x64.blocked.json"

if [[ -z "${WAYLAND_DISPLAY:-}" ]]; then
  cat >"$blocked_path" <<'JSON'
{
  "schema": "tetra.surface.block-system.blocked.v1",
  "status": "blocked",
  "target": "linux-x64",
  "reason": "WAYLAND_DISPLAY is not set; linux-x64 real-window Block evidence was not promoted from headless"
}
JSON
  echo "Surface linux-x64 real-window Block-system smoke blocked: $blocked_path" >&2
  exit 1
fi

go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system --source examples/surface_block_system.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-block-report --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64 real-window Block-system runtime smoke report: $report_path"
echo "Surface linux-x64 real-window Block-system artifact hashes: $report_dir/artifact-hashes.json"
