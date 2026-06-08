#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/post-v0.4"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh [--report-dir DIR]

Runs executable Linux-x64 Memory Production Core smoke and writes tetra.memory.production.v1 evidence.
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

check_report_dir_fresh() {
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi

  if [[ -L "$find_report_dir" ]]; then
    if [[ -d "$find_report_dir" ]]; then
      local symlink_entry
      symlink_entry="$(find -H -- "$find_report_dir" -mindepth 1 -print -quit)"
      if [[ -n "$symlink_entry" ]]; then
        echo "refusing to reuse non-empty report directory: $report_dir" >&2
        echo "choose a fresh --report-dir so stale reports cannot be reused" >&2
        exit 2
      fi
    fi
    echo "refusing to use symlink report directory: $report_dir" >&2
    echo "choose a real fresh --report-dir so reports cannot escape the selected directory" >&2
    exit 2
  fi

  if [[ ( -e "$find_report_dir" || -L "$find_report_dir" ) && ! -d "$find_report_dir" ]]; then
    echo "refusing to use non-directory report path: $report_dir" >&2
    echo "choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ ! -d "$find_report_dir" ]]; then
    return 0
  fi
  local first_entry
  first_entry="$(find -H -- "$find_report_dir" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    echo "refusing to reuse non-empty report directory: $report_dir" >&2
    echo "choose a fresh --report-dir so stale reports cannot be reused" >&2
    exit 2
  fi
}

json_escape() {
  local value="$1"
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  printf '%s' "$value"
}

check_report_dir_fresh
mkdir -p -- "$report_dir"
report_path="$report_dir/memory-production-linux-x64.json"
targets_path="$report_dir/targets.json"
memory_fuzz_dir="$report_dir/memory-fuzz-tier1"
memory_release_manifest_path="$report_dir/memory-release-manifest.json"

go run ./tools/cmd/memory-production-smoke --report "$report_path"
go run ./tools/cmd/validate-memory-production --report "$report_path"
go run ./cli/cmd/tetra targets --format=json > "$targets_path"
go run ./tools/cmd/validate-targets --report "$targets_path"
go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir "$memory_fuzz_dir"
go run ./tools/cmd/validate-memory-fuzz-oracle --report "$memory_fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$memory_fuzz_dir"
git_head="$(git rev-parse --verify HEAD)"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
cat > "$memory_release_manifest_path" <<MANIFEST
{
  "schema": "tetra.memory.release-manifest.v1",
  "target": "linux-x64",
  "git_head": "$git_head",
  "generated_at": "$generated_at",
  "report_dir": ".",
  "hash_manifest": "artifact-hashes.json",
  "commands": [
    {"name": "memory-production-smoke", "command": "go run ./tools/cmd/memory-production-smoke --report $(json_escape "$report_path")"},
    {"name": "target-report", "command": "go run ./cli/cmd/tetra targets --format=json > $(json_escape "$targets_path")"},
    {"name": "validate-targets", "command": "go run ./tools/cmd/validate-targets --report $(json_escape "$targets_path")"},
    {"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir")"},
    {"name": "validate-memory-fuzz-oracle", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report $(json_escape "$memory_fuzz_dir")/memory-fuzz-oracle.json --artifact-dir $(json_escape "$memory_fuzz_dir")"},
    {"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir") --out $(json_escape "$report_dir")/artifact-hashes.json"},
    {"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest $(json_escape "$report_dir")/artifact-hashes.json"}
  ],
  "artifacts": [
    {"path": "memory-production-linux-x64.json", "kind": "memory_production_report", "schema": "tetra.memory.production.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-production-smoke --report $(json_escape "$report_path")"},
    {"path": "targets.json", "kind": "target_report", "target": "linux-x64", "command": "go run ./cli/cmd/tetra targets --format=json > $(json_escape "$targets_path")"},
    {"path": "memory-fuzz-tier1/memory-fuzz-oracle.json", "kind": "memory_fuzz_oracle_report", "schema": "tetra.memory-fuzz.oracle.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir")"},
    {"path": "memory-fuzz-tier1/summary.json", "kind": "memory_fuzz_summary", "schema": "tetra.memory-fuzz-short.summary.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir")"},
    {"path": "artifact-hashes.json", "kind": "artifact_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1", "target": "linux-x64", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir") --out $(json_escape "$report_dir")/artifact-hashes.json"}
  ]
}
MANIFEST
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-memory-production --report "$report_path" --manifest "$memory_release_manifest_path" --report-dir "$report_dir"

echo "memory production linux-x64 smoke report: $report_path"
echo "memory production target capability report: $targets_path"
echo "memory production Tier 1 fuzz oracle report: $memory_fuzz_dir/memory-fuzz-oracle.json"
echo "memory production Tier 1 fuzz oracle summary: $memory_fuzz_dir/summary.json"
echo "memory production release manifest: $memory_release_manifest_path"
echo "memory production linux-x64 artifact hashes: $report_dir/artifact-hashes.json"
