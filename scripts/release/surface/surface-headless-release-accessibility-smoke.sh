#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-headless-release-accessibility-smoke.sh [--report-dir DIR]

Runs deterministic headless Tetra Surface release accessibility smoke.
The report validates tetra.surface.accessibility-tree.v1 platform-bridge-v1
for examples/surface_release_accessibility.tetra with metadata export,
headless platform bridge shape, honest screen_reader_evidence naming, and no
metadata-only production claim.
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
report_path="$report_dir/surface-headless-release-accessibility.json"

go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility --source examples/surface_release_accessibility.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release surface-v1
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface headless release accessibility runtime smoke report: $report_path"
echo "Surface headless release accessibility artifact hashes: $report_dir/artifact-hashes.json"
