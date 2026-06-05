#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-release-text-input-smoke.sh [--report-dir DIR]

Runs the linux-x64 Tetra Surface release text-input smoke.
The gate builds examples/surface_release_text_input.tetra, records real-window
RGBA/native-input evidence for the text-input path, and writes a strict
tetra.surface.text-input.v1 production baseline report without claiming full
platform clipboard or platform IME completion.
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
report_path="$report_dir/surface-linux-x64-release-text-input.json"

go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-text-input --source examples/surface_release_text_input.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release surface-v1
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64 release text-input runtime smoke report: $report_path"
echo "Surface linux-x64 release text-input artifact hashes: $report_dir/artifact-hashes.json"
