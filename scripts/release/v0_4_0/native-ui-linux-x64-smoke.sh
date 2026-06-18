#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/v0.4.0"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh [--report-dir DIR]

Runs executable Linux-x64 native UI runtime smoke and writes tetra.ui.native-runtime.v1 evidence.
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
    -h | --help)
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
report_path="$report_dir/native-ui-linux-x64.json"

go run ./tools/cmd/native-ui-runtime-smoke --report "$report_path"
go run ./tools/cmd/validate-native-ui-runtime --report "$report_path"

echo "native UI linux-x64 runtime smoke report: $report_path"
