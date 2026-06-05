#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh [--report-dir DIR]

Runs the linux-x64-real-window Tetra Surface toolkit reuse smoke.
The gate builds examples/surface_toolkit_settings.tetra, opens a real Linux window
with native input evidence, and validates toolkit-reuse-v1 reusable
tetra.surface.toolkit.v1 lib.core.widgets routing instead of platform widgets
or manual tree bookkeeping.
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
report_path="$report_dir/surface-linux-x64-real-window-toolkit-reuse.json"

go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-toolkit-reuse --source examples/surface_toolkit_settings.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64 real-window toolkit reuse runtime smoke report: $report_path"
echo "Surface linux-x64 real-window toolkit reuse artifact hashes: $report_dir/artifact-hashes.json"
