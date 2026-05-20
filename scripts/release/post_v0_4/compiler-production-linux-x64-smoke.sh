#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/post-v0.4"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh [--report-dir DIR]

Runs executable Linux-x64 Compiler Production smoke and writes tetra.compiler.production.v1 evidence.
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
report_path="$report_dir/compiler-production-linux-x64.json"

go run ./tools/cmd/compiler-production-smoke --report "$report_path"
go run ./tools/cmd/validate-compiler-production --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "compiler production linux-x64 smoke report: $report_path"
echo "compiler production linux-x64 artifact hashes: $report_dir/artifact-hashes.json"
