#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir_arg="reports/memory-islands-surface-production"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh [--report-dir DIR]

Runs the integrated Memory + Islands + Surface production evidence gate. The
gate requires fresh report directories, Memory production evidence, island
proof verifier evidence, proof-fuzz evidence, islands debug sanitizer evidence,
Surface v1 release evidence, Surface experimental regression evidence, Safe
View lifetime evidence, Surface API stability evidence, docs/manifest checks,
same-commit metadata, and final artifact hash integrity.
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
  export GOCACHE="$repo_root/.cache/go-build-memory-islands-surface-production-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-memory-islands-surface-production-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "memory_islands_surface_gate:")"
memory_report_dir="$report_dir_arg/memory"
surface_release_dir="$report_dir_arg/surface-release-v1"
surface_experimental_dir="$report_dir_arg/surface-experimental-regression"
safe_view_dir="$report_dir_arg/safe-view-lifetime"
surface_api_dir="$report_dir_arg/surface-api-stability-v1"
islands_debug_report="$report_dir/islands-debug-smoke.json"
islands_debug_report_arg="$report_dir_arg/islands-debug-smoke.json"
integrated_manifest_path="$report_dir/memory-islands-surface-production-manifest.json"

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "$value"
}

bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$memory_report_dir"
go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug --report "$islands_debug_report"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$islands_debug_report"
bash scripts/release/surface/release-gate.sh --report-dir "$surface_release_dir"
bash scripts/release/surface/gate.sh --report-dir "$surface_experimental_dir"
bash scripts/release/safe-view-lifetime/gate.sh --report-dir "$safe_view_dir"
bash scripts/release/surface/api-stability-gate.sh --report-dir "$surface_api_dir"
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

git_head="$(git rev-parse --verify HEAD)"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat > "$integrated_manifest_path" <<MANIFEST
{
  "schema": "tetra.memory-islands-surface.production-gate.v1",
  "status": "pass",
  "git_head": $(json_string "$git_head"),
  "generated_at": $(json_string "$generated_at"),
  "report_dir": ".",
  "hash_manifest": "artifact-hashes.json",
  "commands": [
    {"name": "memory-production-gate", "command": "bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir $(json_escape "$memory_report_dir")"},
    {"name": "islands-debug-smoke", "command": "go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug --report $(json_escape "$islands_debug_report_arg")"},
    {"name": "validate-islands-debug-smoke", "command": "go run ./tools/cmd/smoke-report-to-checklist --validate-only --report $(json_escape "$islands_debug_report_arg")"},
    {"name": "surface-release-gate", "command": "bash scripts/release/surface/release-gate.sh --report-dir $(json_escape "$surface_release_dir")"},
    {"name": "surface-experimental-regression-gate", "command": "bash scripts/release/surface/gate.sh --report-dir $(json_escape "$surface_experimental_dir")"},
    {"name": "safe-view-lifetime-gate", "command": "bash scripts/release/safe-view-lifetime/gate.sh --report-dir $(json_escape "$safe_view_dir")"},
    {"name": "surface-api-stability-gate", "command": "bash scripts/release/surface/api-stability-gate.sh --report-dir $(json_escape "$surface_api_dir")"},
    {"name": "validate-manifest", "command": "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json"},
    {"name": "verify-docs", "command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json"},
    {"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir_arg") --out $(json_escape "$report_dir_arg")/artifact-hashes.json"},
    {"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest $(json_escape "$report_dir_arg")/artifact-hashes.json"},
    {"name": "integrated-release-validator", "command": "go run ./tools/cmd/validate-memory-islands-surface-production --report-dir $(json_escape "$report_dir_arg") --current-git-head $git_head"}
  ],
  "artifacts": [
    {"path": "memory/memory-production-linux-x64.json", "kind": "memory_production_report", "schema": "tetra.memory.production.v1"},
    {"path": "memory/memory-release-manifest.json", "kind": "memory_release_manifest", "schema": "tetra.memory.release-manifest.v1"},
    {"path": "memory/artifact-hashes.json", "kind": "memory_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
    {"path": "memory/island-proof-verifier.json", "kind": "island_proof_verifier_report", "schema": "tetra.island.proof.v1"},
    {"path": "memory/island-proof-memory-report.json", "kind": "island_proof_memory_report", "schema": "tetra.memory-report.v1"},
    {"path": "memory/memory-fuzz-tier1/island-proof-fuzz-summary.json", "kind": "island_proof_fuzz_summary", "schema": "tetra.island-proof-fuzz-summary.v1"},
    {"path": "memory/ram-contract/ram-contract-report.json", "kind": "ram_contract_report", "schema": "tetra.ram-contract-report.v1"},
    {"path": "memory/ram-contract/memory-grade-report.json", "kind": "ram_memory_grade_report", "schema": "tetra.memory-grade-report.v1"},
    {"path": "memory/ram-contract/proof-store-summary.json", "kind": "ram_proof_store_summary", "schema": "tetra.proof-store-summary.v1"},
    {"path": "memory/ram-contract/validation-pipeline-coverage.json", "kind": "ram_validation_pipeline_coverage", "schema": "tetra.validation-pipeline-coverage.v1"},
    {"path": "memory/ram-contract/heap-blockers.json", "kind": "ram_heap_blockers", "schema": "tetra.ram-blockers.v1"},
    {"path": "memory/ram-contract/copy-blockers.json", "kind": "ram_copy_blockers", "schema": "tetra.ram-blockers.v1"},
    {"path": "memory/ram-contract/fuzz/ram-contract-fuzz-oracle.json", "kind": "ram_contract_fuzz_oracle", "schema": "tetra.ram-contract-fuzz-oracle.v1"},
    {"path": "islands-debug-smoke.json", "kind": "islands_debug_smoke_report", "schema": "tetra.release.v0_2_0.smoke-report.v1"},
    {"path": "surface-release-v1/surface-release-summary.json", "kind": "surface_release_summary", "schema": "tetra.surface.release.v1"},
    {"path": "surface-release-v1/artifact-hashes.json", "kind": "surface_release_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
    {"path": "surface-experimental-regression/artifact-hashes.json", "kind": "surface_experimental_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
    {"path": "safe-view-lifetime/safe-view-lifetime-summary.json", "kind": "safe_view_lifetime_summary", "schema": "tetra.safe-view-lifetime.gate.v1"},
    {"path": "surface-api-stability-v1/surface-api-stability-summary.json", "kind": "surface_api_stability_summary", "schema": "tetra.surface.api-stability.v1"},
    {"path": "artifact-hashes.json", "kind": "integrated_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"}
  ]
}
MANIFEST

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-memory-islands-surface-production --report-dir "$report_dir" --current-git-head "$git_head"

echo "Memory/Islands/Surface integrated production gate reports: $report_dir"
echo "Memory/Islands/Surface integrated manifest: $integrated_manifest_path"
echo "Memory/Islands/Surface integrated artifact hashes: $report_dir/artifact-hashes.json"
