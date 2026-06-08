#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-headless-release-text-input-smoke.sh [--report-dir DIR]

Runs deterministic headless Tetra Surface release text-input smoke.
The report uses tetra.surface.text-input.v1 and proves the owned UTF-8 buffer
baseline: ASCII/UTF-8 insertion, caret home/end/arrows, selection replacement,
backspace/delete, owned-copy clipboard transfer, composition commit/cancel, and
safe-view lifetime checks.
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
report_path="$report_dir/surface-headless-release-text-input.json"

go run ./tools/cmd/surface-runtime-smoke --mode headless-release-text-input --source examples/surface_release_text_input.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release text-input
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface headless release text-input runtime smoke report: $report_path"
echo "Surface headless release text-input artifact hashes: $report_dir/artifact-hashes.json"
