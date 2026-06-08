#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/full-platform-ui-runtime"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/macos-ui-runtime-smoke.sh [--report-dir DIR]

Writes the macOS platform UI runtime smoke report. The report is only a
production pass on a real macOS target-host runner with runtime-backed UI
evidence; unsupported hosts write a blocked report and fail.
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
report_path="$report_dir/macos-ui-runtime.json"
external_report="${TETRA_MACOS_UI_RUNTIME_REPORT:-}"
expected_version="$("./tetra" version 2>/dev/null || go run ./cli/cmd/tetra version)"
expected_git_head="$(git rev-parse HEAD)"

if [[ -n "$external_report" ]]; then
  if [[ ! -f "$external_report" ]]; then
    echo "error: TETRA_MACOS_UI_RUNTIME_REPORT does not name a readable file: $external_report" >&2
    exit 2
  fi
  cp -- "$external_report" "$report_path"
  go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path" --expected-version "$expected_version" --expected-git-head "$expected_git_head"
  echo "macOS platform UI runtime report imported from TETRA_MACOS_UI_RUNTIME_REPORT: $report_path"
  exit 0
fi

go run ./tools/cmd/platform-ui-runtime-smoke \
  --target macos-x64 \
  --report "$report_path"
