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
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-memory-production-release"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-memory-production-release"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

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
ram_measurement_path="$report_dir/ram-measurement.json"
targets_path="$report_dir/targets.json"
memory_fuzz_dir="$report_dir/memory-fuzz-tier1"
ram_contract_dir="$report_dir/ram-contract"
island_proof_path="$report_dir/island-proof-verifier.json"
island_proof_memory_report_path="$report_dir/island-proof-memory-report.json"
memory_release_manifest_path="$report_dir/memory-release-manifest.json"
git_head="$(git rev-parse --verify HEAD)"

go run ./tools/cmd/memory-production-smoke --report "$report_path" --ram-measurement-report "$ram_measurement_path" --git-head "$git_head"
go run ./tools/cmd/validate-memory-production --report "$report_path"
go run ./cli/cmd/tetra targets --format=json > "$targets_path"
go run ./tools/cmd/validate-targets --report "$targets_path"
go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir "$memory_fuzz_dir" --git-head "$git_head"
go run ./tools/cmd/validate-memory-fuzz-oracle --report "$memory_fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$memory_fuzz_dir" --current-git-head "$git_head"
bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
cat > "$island_proof_memory_report_path" <<ISLAND_MEMORY_REPORT
{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "release-memory-production",
      "function_id": "island-proof-verifier-fixture",
      "site_id": "island:release:borrow:1",
      "source_fact_id": "fact:release:island-proof:1",
      "source_stage": "validation",
      "claim": "island_proof_verified",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "alias_state": "unique",
      "island_id": "island:release:0",
      "epoch": 1,
      "base_id": "alloc:release:island:0",
      "proof_id": "proof:release:island:borrow:1",
      "proof_kind": "island_epoch",
      "proof_subject_base_id": "alloc:release:island:0",
      "proof_operation": "island_borrow",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "cost_class": "instrumentation_only",
      "reason": "release fixture proving independent island verifier gate"
    }
  ]
}
ISLAND_MEMORY_REPORT
cat > "$island_proof_path" <<ISLAND_PROOF
{
  "schema": "tetra.island.proof.v1",
  "producer": "tools/validators/islandproof/release-fixture",
  "producer_command": "go run ./tools/cmd/validate-island-proof",
  "git_head": "$git_head",
  "generated_at": "$generated_at",
  "proofs": [
    {
      "proof_id": "proof:release:island:borrow:1",
      "operation": "island_borrow",
      "proof_kind": "island_epoch",
      "subject_base_id": "alloc:release:island:0",
      "island_id": "island:release:0",
      "epoch": 1,
      "source_fact_id": "fact:release:island-proof:1",
      "claim": "island_proof_verified",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "dominance": "entry dominates release island borrow",
      "distinct_live_islands": ["island:release:0", "island:release:1"]
    }
  ]
}
ISLAND_PROOF
go run ./tools/cmd/validate-island-proof --proof "$island_proof_path" --memory-report "$island_proof_memory_report_path" --current-git-head "$git_head" --require-same-commit
cat > "$memory_release_manifest_path" <<MANIFEST
{
  "schema": "tetra.memory.release-manifest.v1",
  "target": "linux-x64",
  "git_head": "$git_head",
  "generated_at": "$generated_at",
  "report_dir": ".",
  "hash_manifest": "artifact-hashes.json",
  "commands": [
    {"name": "memory-production-smoke", "command": "go run ./tools/cmd/memory-production-smoke --report $(json_escape "$report_path") --ram-measurement-report $(json_escape "$ram_measurement_path") --git-head $git_head"},
    {"name": "target-report", "command": "go run ./cli/cmd/tetra targets --format=json > $(json_escape "$targets_path")"},
    {"name": "validate-targets", "command": "go run ./tools/cmd/validate-targets --report $(json_escape "$targets_path")"},
    {"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir") --git-head $git_head"},
    {"name": "validate-memory-fuzz-oracle", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report $(json_escape "$memory_fuzz_dir")/memory-fuzz-oracle.json --artifact-dir $(json_escape "$memory_fuzz_dir") --current-git-head $git_head"},
    {"name": "ram-contract-gate", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"name": "island-proof-verifier", "command": "go run ./tools/cmd/validate-island-proof --proof $(json_escape "$island_proof_path") --memory-report $(json_escape "$island_proof_memory_report_path") --current-git-head $git_head --require-same-commit"},
    {"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir") --out $(json_escape "$report_dir")/artifact-hashes.json"},
    {"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest $(json_escape "$report_dir")/artifact-hashes.json"}
  ],
  "artifacts": [
    {"path": "memory-production-linux-x64.json", "kind": "memory_production_report", "schema": "tetra.memory.production.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-production-smoke --report $(json_escape "$report_path") --ram-measurement-report $(json_escape "$ram_measurement_path") --git-head $git_head"},
    {"path": "ram-measurement.json", "kind": "ram_measurement_report", "schema": "tetra.memory.ram-measurement.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-production-smoke --report $(json_escape "$report_path") --ram-measurement-report $(json_escape "$ram_measurement_path") --git-head $git_head"},
    {"path": "targets.json", "kind": "target_report", "target": "linux-x64", "command": "go run ./cli/cmd/tetra targets --format=json > $(json_escape "$targets_path")"},
    {"path": "memory-fuzz-tier1/memory-fuzz-oracle.json", "kind": "memory_fuzz_oracle_report", "schema": "tetra.memory-fuzz.oracle.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir") --git-head $git_head"},
    {"path": "memory-fuzz-tier1/summary.json", "kind": "memory_fuzz_summary", "schema": "tetra.memory-fuzz-short.summary.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir") --git-head $git_head"},
    {"path": "memory-fuzz-tier1/island-proof-fuzz-summary.json", "kind": "memory_fuzz_island_proof_summary", "schema": "tetra.island-proof-fuzz-summary.v1", "target": "linux-x64", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $(json_escape "$memory_fuzz_dir") --git-head $git_head"},
    {"path": "ram-contract/ram-contract-release-manifest.json", "kind": "ram_contract_release_manifest", "schema": "tetra.ram-contract.release-manifest.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/ram-contract-report.json", "kind": "ram_contract_report", "schema": "tetra.ram-contract-report.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/memory-grade-report.json", "kind": "ram_memory_grade_report", "schema": "tetra.memory-grade-report.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/proof-store-summary.json", "kind": "ram_proof_store_summary", "schema": "tetra.proof-store-summary.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/validation-pipeline-coverage.json", "kind": "ram_validation_pipeline_coverage", "schema": "tetra.validation-pipeline-coverage.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/heap-blockers.json", "kind": "ram_heap_blockers", "schema": "tetra.ram-blockers.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/copy-blockers.json", "kind": "ram_copy_blockers", "schema": "tetra.ram-blockers.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/fuzz/ram-contract-fuzz-oracle.json", "kind": "ram_contract_fuzz_oracle", "schema": "tetra.ram-contract-fuzz-oracle.v1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "ram-contract/artifact-hashes.json", "kind": "ram_contract_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1", "target": "linux-x64", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $(json_escape "$ram_contract_dir")"},
    {"path": "island-proof-verifier.json", "kind": "island_proof_verifier_report", "schema": "tetra.island.proof.v1", "target": "linux-x64", "command": "go run ./tools/cmd/validate-island-proof --proof $(json_escape "$island_proof_path") --memory-report $(json_escape "$island_proof_memory_report_path") --current-git-head $git_head --require-same-commit"},
    {"path": "island-proof-memory-report.json", "kind": "island_proof_memory_report", "schema": "tetra.memory-report.v1", "target": "linux-x64", "command": "go run ./tools/cmd/validate-island-proof --proof $(json_escape "$island_proof_path") --memory-report $(json_escape "$island_proof_memory_report_path") --current-git-head $git_head --require-same-commit"},
    {"path": "artifact-hashes.json", "kind": "artifact_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1", "target": "linux-x64", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir") --out $(json_escape "$report_dir")/artifact-hashes.json"}
  ]
}
MANIFEST
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-memory-production --report "$report_path" --manifest "$memory_release_manifest_path" --report-dir "$report_dir" --current-git-head "$git_head"

echo "memory production linux-x64 smoke report: $report_path"
echo "memory production RAM measurement report: $ram_measurement_path"
echo "memory production target capability report: $targets_path"
echo "memory production Tier 1 fuzz oracle report: $memory_fuzz_dir/memory-fuzz-oracle.json"
echo "memory production Tier 1 fuzz oracle summary: $memory_fuzz_dir/summary.json"
echo "memory production RAM contract report: $ram_contract_dir/ram-contract-report.json"
echo "memory production island proof fuzz summary: $memory_fuzz_dir/island-proof-fuzz-summary.json"
echo "memory production island proof verifier: $island_proof_path"
echo "memory production island proof memory report: $island_proof_memory_report_path"
echo "memory production release manifest: $memory_release_manifest_path"
echo "memory production linux-x64 artifact hashes: $report_dir/artifact-hashes.json"
