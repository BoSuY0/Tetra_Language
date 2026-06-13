#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir_arg="reports/ram-contract-release"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh [--report-dir DIR]

Runs Linux-x64 RAM Contract Compiler smoke evidence and writes
tetra.ram-contract-report.v1, tetra.memory-grade-report.v1, proof-store,
pipeline coverage, blocker, fuzz-oracle, and artifact hash evidence.
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
      report_dir_arg="$2"
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
source "$repo_root/scripts/release/surface/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-ram-contract-release"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-ram-contract-release"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "ram_contract_gate:")"
git_head="$(git rev-parse --verify HEAD)"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
fixture_src="$report_dir/ram_contract_fixture.tetra"
fixture_out="$report_dir/ram-contract-fixture"
manifest_path="$report_dir/ram-contract-release-manifest.json"

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "$value"
}

cat > "$fixture_src" <<'TETRA'
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2048)
    xs[0] = 7
    return xs[0]
TETRA

go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-contract-report --emit-memory-report --emit-alloc-report -o "$fixture_out" "$fixture_src"

mv "$fixture_out.ram-contract.json" "$report_dir/ram-contract-report.json"
mv "$fixture_out.memory-grade.json" "$report_dir/memory-grade-report.json"
mv "$fixture_out.proof-store-summary.json" "$report_dir/proof-store-summary.json"
mv "$fixture_out.validation-pipeline-coverage.json" "$report_dir/validation-pipeline-coverage.json"
mv "$fixture_out.heap-blockers.json" "$report_dir/heap-blockers.json"
mv "$fixture_out.copy-blockers.json" "$report_dir/copy-blockers.json"
mv "$fixture_out.ram-contract.txt" "$report_dir/ram-contract-summary.md"

go run ./tools/cmd/validate-ram-contract-report --report "$report_dir/ram-contract-report.json"
go run ./tools/cmd/validate-memory-grade-report --report "$report_dir/memory-grade-report.json"
go run ./tools/cmd/validate-proof-store-summary --report "$report_dir/proof-store-summary.json"
go run ./tools/cmd/validate-validation-pipeline-coverage --report "$report_dir/validation-pipeline-coverage.json"
go run ./tools/cmd/validate-heap-blockers --report "$report_dir/heap-blockers.json"
go run ./tools/cmd/validate-copy-blockers --report "$report_dir/copy-blockers.json"

go run ./tools/cmd/ram-contract-fuzz-short --report-dir "$report_dir/fuzz" --git-head "$git_head"
go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report "$report_dir/fuzz/ram-contract-fuzz-oracle.json" --current-git-head "$git_head"

cat > "$manifest_path" <<MANIFEST
{
  "schema": "tetra.ram-contract.release-manifest.v1",
  "status": "pass",
  "target": "linux-x64",
  "git_head": "$git_head",
  "generated_at": "$generated_at",
  "report_dir": ".",
  "hash_manifest": "artifact-hashes.json",
  "commands": [
    {"name": "ram-contract-build", "command": "go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-contract-report --emit-memory-report --emit-alloc-report -o $(json_escape "$fixture_out") $(json_escape "$fixture_src")"},
    {"name": "validate-ram-contract-report", "command": "go run ./tools/cmd/validate-ram-contract-report --report $(json_escape "$report_dir")/ram-contract-report.json"},
    {"name": "validate-memory-grade-report", "command": "go run ./tools/cmd/validate-memory-grade-report --report $(json_escape "$report_dir")/memory-grade-report.json"},
    {"name": "validate-proof-store-summary", "command": "go run ./tools/cmd/validate-proof-store-summary --report $(json_escape "$report_dir")/proof-store-summary.json"},
    {"name": "validate-validation-pipeline-coverage", "command": "go run ./tools/cmd/validate-validation-pipeline-coverage --report $(json_escape "$report_dir")/validation-pipeline-coverage.json"},
    {"name": "validate-heap-blockers", "command": "go run ./tools/cmd/validate-heap-blockers --report $(json_escape "$report_dir")/heap-blockers.json"},
    {"name": "validate-copy-blockers", "command": "go run ./tools/cmd/validate-copy-blockers --report $(json_escape "$report_dir")/copy-blockers.json"},
    {"name": "ram-contract-fuzz-short", "command": "go run ./tools/cmd/ram-contract-fuzz-short --report-dir $(json_escape "$report_dir")/fuzz --git-head $git_head"},
    {"name": "validate-ram-contract-fuzz-oracle", "command": "go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report $(json_escape "$report_dir")/fuzz/ram-contract-fuzz-oracle.json --current-git-head $git_head"},
    {"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir") --out $(json_escape "$report_dir")/artifact-hashes.json"},
    {"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest $(json_escape "$report_dir")/artifact-hashes.json"},
    {"name": "ram-contract-release-validator", "command": "go run ./tools/cmd/validate-ram-contract-release --report-dir $(json_escape "$report_dir") --current-git-head $git_head"}
  ],
  "artifacts": [
    {"path": "ram-contract-report.json", "kind": "ram_contract_report", "schema": "tetra.ram-contract-report.v1"},
    {"path": "memory-grade-report.json", "kind": "memory_grade_report", "schema": "tetra.memory-grade-report.v1"},
    {"path": "proof-store-summary.json", "kind": "proof_store_summary", "schema": "tetra.proof-store-summary.v1"},
    {"path": "validation-pipeline-coverage.json", "kind": "validation_pipeline_coverage", "schema": "tetra.validation-pipeline-coverage.v1"},
    {"path": "heap-blockers.json", "kind": "heap_blockers", "schema": "tetra.ram-blockers.v1"},
    {"path": "copy-blockers.json", "kind": "copy_blockers", "schema": "tetra.ram-blockers.v1"},
    {"path": "fuzz/ram-contract-fuzz-oracle.json", "kind": "ram_contract_fuzz_oracle", "schema": "tetra.ram-contract-fuzz-oracle.v1"},
    {"path": "artifact-hashes.json", "kind": "artifact_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"}
  ],
  "non_claims": [
    "no Memory 100% claim",
    "no full formal proof claim",
    "no official benchmark or fastest-language claim",
    "local Linux-x64 scoped RAM contract evidence only"
  ]
}
MANIFEST

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-ram-contract-release --report-dir "$report_dir" --current-git-head "$git_head"

echo "RAM contract linux-x64 smoke reports: $report_dir"
echo "RAM contract report: $report_dir/ram-contract-report.json"
echo "RAM memory grade report: $report_dir/memory-grade-report.json"
echo "RAM proof store summary: $report_dir/proof-store-summary.json"
echo "RAM validation pipeline coverage: $report_dir/validation-pipeline-coverage.json"
echo "RAM contract artifact hashes: $report_dir/artifact-hashes.json"
