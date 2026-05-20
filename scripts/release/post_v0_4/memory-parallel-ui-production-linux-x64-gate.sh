#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/post-v0.4"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh [--report-dir DIR]

Runs the ordered Linux-x64 production evidence gates for:
  1. tetra.memory.production.v1
  2. tetra.parallel.production.v1
  3. tetra.ui.desktop-runtime.v1
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

bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$report_dir"
bash "$script_dir/parallel-production-linux-x64-smoke.sh" --report-dir "$report_dir"
bash "$script_dir/ui-production-runtime-linux-x64-smoke.sh" --report-dir "$report_dir"

go run ./tools/cmd/validate-memory-production --report "$report_dir/memory-production-linux-x64.json"
go run ./tools/cmd/validate-parallel-production --report "$report_dir/parallel-production-linux-x64.json"
go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"
go run ./tools/cmd/validate-post-v04-production-audit --report-dir "$report_dir" --write --out "$report_dir/post-v0.4-production-audit.json"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-post-v04-production-audit --report-dir "$report_dir"

echo "post-v0.4 Memory/Parallelism/UI production gate report dir: $report_dir"
echo "required schemas: tetra.memory.production.v1, tetra.parallel.production.v1, tetra.ui.desktop-runtime.v1, tetra.release.post_v0_4.memory_parallel_ui_completion_audit.v1"
echo "artifact hashes: $report_dir/artifact-hashes.json"
