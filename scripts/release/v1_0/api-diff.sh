#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

report_dir=""
baseline="docs/baselines/api-diff-baseline.v1alpha1.json"
enforce_mode="no-change"
write_baseline="false"
release_artifact="tetra.release.v1_0.api-diff-report.v1alpha1"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/v1_0/api-diff.sh [--report-dir DIR] [--baseline PATH] [--enforce MODE] [--write-baseline]

Options:
  --report-dir DIR    Output directory for generated API docs + diff report.
  --baseline PATH     Baseline JSON path (default: docs/baselines/api-diff-baseline.v1alpha1.json).
  --enforce MODE      one of: none, no-breaking, no-change (default: no-change).
  --write-baseline    Regenerate baseline JSON from current docs before diff.

Artifact mapping:
  tetra.release.v1_0.api-diff-report.v1alpha1
USAGE
}

require_flag_value() {
  local flag="$1"
  local description="$2"
  if [[ $# -lt 3 || -z "${3:-}" ]]; then
    echo "release/v1_0/api-diff: ${flag} requires ${description}" >&2
    exit 2
  fi
}

normalize_relative_dash_path() {
  local path="$1"
  if [[ "$path" == -* ]]; then
    printf './%s' "$path"
  else
    printf '%s' "$path"
  fi
}

check_report_dir_fresh() {
  if [[ -L "$report_dir" && -d "$report_dir" ]]; then
    echo "release/v1_0/api-diff: refusing to use symlink report path: $report_dir" >&2
    echo "release/v1_0/api-diff: choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ ( -e "$report_dir" || -L "$report_dir" ) && ! -d "$report_dir" ]]; then
    echo "release/v1_0/api-diff: refusing to use non-directory report path: $report_dir" >&2
    echo "release/v1_0/api-diff: choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ -d "$report_dir" ]] && [[ -n "$(find "$report_dir" -mindepth 1 -print -quit)" ]]; then
    echo "release/v1_0/api-diff: refusing to reuse non-empty report directory: $report_dir" >&2
    echo "release/v1_0/api-diff: choose a fresh --report-dir so stale reports cannot be reused" >&2
    exit 2
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      require_flag_value "$1" "a directory" "${2:-}"
      report_dir="$2"
      shift 2
      ;;
    --baseline)
      require_flag_value "$1" "a path" "${2:-}"
      baseline="$2"
      shift 2
      ;;
    --enforce)
      require_flag_value "$1" "a mode" "${2:-}"
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
      echo "release/v1_0/api-diff: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  state_home="${XDG_STATE_HOME:-${HOME:?HOME must be set}/.local/state}"
  report_dir="$state_home/tetra-language/v1-api-diff-$(date -u +%Y%m%d-%H%M%S)"
fi
report_dir="$(normalize_relative_dash_path "$report_dir")"

check_report_dir_fresh
if [[ "$write_baseline" != "true" ]] && [[ ! -f "$baseline" ]]; then
  echo "release/v1_0/api-diff: missing baseline $baseline" >&2
  echo "release/v1_0/api-diff: rerun with --write-baseline to create it" >&2
  exit 1
fi
mkdir -p -- "$report_dir"
docs_out="$report_dir/api-docs.md"
diff_out="$report_dir/api-diff.json"

# Candidate docs should use tracked examples so local untracked WIP files do not
# change release-gate output.
mapfile -t tracked_examples < <(git ls-files 'examples/*.tetra')
if [[ "${#tracked_examples[@]}" -eq 0 ]]; then
  echo "release/v1_0/api-diff: no tracked examples found under examples/" >&2
  exit 1
fi
go run ./tools/cmd/gen-docs "${tracked_examples[@]}" >"$docs_out"
go run ./tools/cmd/validate-api-docs --docs "$docs_out"

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
