#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

report_dir=""
baseline="docs/baselines/api-diff-baseline.v1alpha1.json"
enforce_mode="no-change"
write_baseline="false"
release_artifact="tetra.release.v0_2_0.api-diff-report.v1alpha1"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_api_diff.sh [--report-dir DIR] [--baseline PATH] [--enforce MODE] [--write-baseline]

Options:
  --report-dir DIR    Output directory for generated API docs + diff report.
  --baseline PATH     Baseline JSON path (default: docs/baselines/api-diff-baseline.v1alpha1.json).
  --enforce MODE      one of: none, no-breaking, no-change (default: no-change).
  --write-baseline    Regenerate baseline JSON from current docs before diff.

Artifact mapping:
  tetra.release.v0_2_0.api-diff-report.v1alpha1
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      report_dir="$2"
      shift 2
      ;;
    --baseline)
      baseline="$2"
      shift 2
      ;;
    --enforce)
      enforce_mode="$2"
      shift 2
      ;;
    --write-baseline)
      write_baseline="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release_v1_0_api_diff: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  report_dir="/tmp/tetra-v1-api-diff-$(date -u +%Y%m%d-%H%M%S)"
fi

mkdir -p "$report_dir"
docs_out="$report_dir/api-docs.md"
diff_out="$report_dir/api-diff.json"

# Candidate docs should use tracked examples so local untracked WIP files do not
# change release-gate output.
mapfile -t tracked_examples < <(git ls-files 'examples/*.tetra')
if [[ "${#tracked_examples[@]}" -eq 0 ]]; then
  echo "release_v1_0_api_diff: no tracked examples found under examples/" >&2
  exit 1
fi
go run ./tools/cmd/gen-docs "${tracked_examples[@]}" >"$docs_out"
go run ./tools/cmd/validate-api-docs --docs "$docs_out"

if [[ "$write_baseline" != "true" ]] && [[ ! -f "$baseline" ]]; then
  echo "release_v1_0_api_diff: missing baseline $baseline" >&2
  echo "release_v1_0_api_diff: rerun with --write-baseline to create it" >&2
  exit 1
fi

args=(
  --docs "$docs_out"
  --baseline "$baseline"
  --diff-out "$diff_out"
  --enforce "$enforce_mode"
)
if [[ "$write_baseline" == "true" ]]; then
  args+=(--write-baseline)
fi

node scripts/tools/api_diff_report.mjs "${args[@]}"

echo "api-diff docs: $docs_out"
echo "api-diff report: $diff_out"
echo "api-diff baseline: $baseline"
echo "api-diff artifact: $release_artifact"
