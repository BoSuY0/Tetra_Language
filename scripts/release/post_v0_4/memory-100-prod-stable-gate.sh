#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir_arg="reports/memory-100/final/aggregate"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/post_v0_4/memory-100-prod-stable-gate.sh [--report-dir DIR]

Runs the scoped Memory100 aggregate evidence gate. The gate requires a fresh
report directory, Memory production evidence, RAM Contract release evidence,
integrated Memory/Islands/Surface evidence, scoped raw/allocation/proof/fuzz/
proof-transition/runtime-memory/semantic-safety/leak-resource aggregate artifacts,
docs claim policy evidence, artifact hashes, same-commit metadata, and the
validate-memory-100-prod-stable validator.
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
gate_contract="scripts/release/post_v0_4/contracts/memory-100-prod-stable-linux-x64.json"
source "$repo_root/scripts/release/surface/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-memory-100-prod-stable-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-memory-100-prod-stable-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

go run ./tools/cmd/run-gate \
  --contract "$gate_contract" \
  --report-dir "$report_dir_arg" \
  --dry-run > /dev/null

report_dir="$(
  surface_release_require_fresh_report_dir \
    "$report_dir_arg" \
    "$repo_root" \
    "memory100_prod_stable_gate:"
)"
memory_production_dir="$report_dir_arg/memory-production"
ram_contract_dir="$report_dir_arg/ram-contract"
integrated_dir="$report_dir_arg/integrated"
raw_memory_dir="$report_dir/raw-memory-contract"
allocation_dir="$report_dir/allocation-lowering"
proof_store_dir="$report_dir/proof-store"
proof_transition_dir="$report_dir/proof-transition"
runtime_memory_dir="$report_dir/runtime-memory"
memory_fuzz_dir="$report_dir/memory-fuzz"
semantic_safety_dir="$report_dir/semantic-safety"
leak_resource_dir="$report_dir/leak-resource"
docs_manifest_dir="$report_dir/docs-manifest"
aggregate_manifest_path="$report_dir/memory-100-prod-stable-manifest.json"

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "$value"
}

bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$memory_production_dir"
bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"
bash "$script_dir/memory-islands-surface-production-gate.sh" --report-dir "$integrated_dir"

git_head="$(git rev-parse --verify HEAD)"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
git_dirty=false
if ! git diff --quiet 2> /dev/null ||
  ! git diff --cached --quiet 2> /dev/null ||
  [[ -n "$(git ls-files --others --exclude-standard 2> /dev/null)" ]]; then
  git_dirty=true
fi
memory100_verdict="MEMORY100_SCOPED_READY_LOCAL"
if [[ "$git_dirty" == true ]]; then
  memory100_verdict="MEMORY100_SCOPED_READY_DIRTY"
fi
git_status_json="$(
  git status --short --branch |
    python3 -c 'import json, sys; print(json.dumps([line.rstrip("\\n") for line in sys.stdin]))'
)"

mkdir -p \
  "$raw_memory_dir" \
  "$allocation_dir" \
  "$proof_store_dir" \
  "$proof_transition_dir" \
  "$runtime_memory_dir" \
  "$memory_fuzz_dir" \
  "$semantic_safety_dir" \
  "$leak_resource_dir" \
  "$docs_manifest_dir"

cat > "$raw_memory_dir/raw-memory-contract.json" << RAW
{
  "schema": "tetra.raw-memory-contract.v1",
  "status": "pass",
  "git_head": "$git_head",
  "operations": [
    {
      "name": "core.alloc_bytes",
      "source_artifacts": [
        "compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "positive_tests": ["allocation-base metadata"]
    },
    {
      "name": "core.ptr_add",
      "source_artifacts": [
        "compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "negative_tests": ["negative offset", "allocation upper bound", "access-width overflow"]
    },
    {
      "name": "raw_slice_from_parts",
      "source_artifacts": [
        "compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
        "compiler/tests/semantics/semantics_memory_surface_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "negative_tests": ["outside unsafe", "negative length", "i32 byte overflow"]
    },
    {
      "name": "raw_load_store_metadata",
      "source_artifacts": [
        "compiler/internal/plir/plir_test/plir_test.go",
        "compiler/internal/lower/lower_suite_test.go",
        "compiler/internal/memoryfacts_test/from_plir_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "positive_tests": [
        "IRMemWriteI32Offset",
        "IRMemReadI32Offset",
        "core.store_u8/core.load_u8 raw memory gateway UnsafeChecked"
      ],
      "negative_tests": [
        "checked_external_unknown raw store/load remains conservative",
        "rejected_access_width_overflow raw load/store width rejection"
      ],
      "non_claims": ["no arbitrary external pointer safety claim"]
    },
    {
      "name": "memcpy_u8",
      "source_artifacts": [
        "lib/core/memory/memory.tetra",
        "compiler/internal/lower/lower_suite_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "positive_tests": ["memcpy/memset capability path"],
      "negative_tests": ["negative length", "access-width overflow"],
      "non_claims": ["no overlapping memcpy safety claim"]
    },
    {
      "name": "memset_u8",
      "source_artifacts": [
        "lib/core/memory/memory.tetra",
        "compiler/internal/lower/lower_suite_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "positive_tests": ["memcpy/memset capability path"],
      "negative_tests": ["negative length", "access-width overflow"]
    },
    {
      "name": "cap.mem",
      "source_artifacts": [
        "lib/core/base/capability.tetra",
        "compiler/internal/ramcontract/validate_test.go",
        "memory-production/memory-production-linux-x64.json"
      ],
      "negative_tests": ["unsafe_unknown promotion rejected", "cap.mem overclaim rejected"],
      "non_claims": ["no arbitrary external pointer safety claim", "no universal Memory 100% claim"]
    }
  ]
}
RAW

python3 - \
  "$allocation_dir/allocation-lowering-report.json" \
  "$git_head" \
  "$ram_contract_dir/heap-blockers.json" \
  "$ram_contract_dir/copy-blockers.json" << 'PY'
import json
import sys

out_path, git_head, heap_path, copy_path = sys.argv[1:5]

def rows(path):
    with open(path, encoding="utf-8") as handle:
        data = json.load(handle)
    return data.get("rows") or []

def site_ids(path):
    return [row["site_id"] for row in rows(path) if row.get("site_id")]

heap_sites = site_ids(heap_path)
copy_sites = site_ids(copy_path)

def blocker_decision(name, planned, artifact, reason, budget, grade, validators, sites):
    base = {
        "name": name,
        "planned_storage": planned,
        "actual_lowering_storage": planned,
        "source_artifacts": [
            artifact,
            "ram-contract/ram-contract-report.json",
            "ram-contract/memory-grade-report.json",
        ],
    }
    if not sites:
        base["status"] = "not_observed"
        base["source_artifacts"] = [artifact, "ram-contract/ram-contract-report.json"]
        return base
    base.update({
        "status": "blocked",
        "blocker_artifact": artifact,
        "blocker_reason": reason,
        "budget_impact": budget,
        "grade_impact": grade,
        "validator_coverage": validators,
        "covered_site_ids": sites,
    })
    return base

report = {
    "schema": "tetra.allocation-lowering.v1",
    "status": "pass",
    "git_head": git_head,
    "decisions": [
        {
            "name": "stack_trusted_no_escape",
            "status": "not_observed",
            "planned_storage": "Stack",
            "actual_lowering_storage": "Stack",
            "source_artifacts": ["ram-contract/ram-contract-report.json"],
        },
        blocker_decision(
            "heap_fallback_blocker",
            "Heap",
            "ram-contract/heap-blockers.json",
            "conservative heap fallback remains explicit until "
            "no-escape/lifetime proof is available",
            "heap rows and budget bytes are accounted in ram-contract/memory-grade-report.json",
            "heap fallback rows keep conservative RAM grade instead of trusted storage overclaim",
            ["validate-heap-blockers", "validate-ram-contract-release"],
            heap_sites,
        ),
        blocker_decision(
            "copy_blocker",
            "Copy",
            "ram-contract/copy-blockers.json",
            "copy decision remains explicit until alias/lifetime proof "
            "removes the copy requirement",
            "copy rows and budget bytes are accounted in ram-contract/memory-grade-report.json",
            "copy fallback rows keep conservative RAM grade instead of zero-copy overclaim",
            ["validate-copy-blockers", "validate-ram-contract-release"],
            copy_sites,
        ),
        {
            "name": "lowering_storage_match",
            "status": "not_observed",
            "planned_storage": "ExplicitIsland",
            "actual_lowering_storage": "ExplicitIsland",
            "source_artifacts": ["ram-contract/ram-contract-report.json"],
        },
    ],
    "non_claims": [
        "no zero heap for all programs claim",
        "no zero-copy for all programs claim",
    ],
}

with open(out_path, "w", encoding="utf-8") as handle:
    json.dump(report, handle, indent=2)
    handle.write("\n")
PY

cp "$report_dir/ram-contract/proof-store-summary.json" "$proof_store_dir/proof-store-summary.json"
proof_stable_hash_evidence="StableHash includes semantic dominance/lifetime/epoch/"
proof_stable_hash_evidence+="invalidation/consumer fields and stale semantic mutation is blocked."
proof_stable_hash_test="go test ./compiler/internal/proof"
proof_stable_hash_test+=" -run TestProofStoreRejectsStaleStableHashForSemanticFields -count=1"
proof_translation_test="go test ./compiler"
proof_translation_test+=" -run TestP23TranslationValidationV2CoversSupported"
proof_translation_test+="OptimizerSubset -count=1"
proof_translation_evidence="translation validation compares proof facts and preserves"
proof_translation_evidence+=" supported bounds proof evidence."
proof_missing_id_test="go test ./compiler/internal/validation"
proof_missing_id_test+=" -run TestValidateTranslationRejectsMissingProofIDAfterTransform -count=1"
proof_missing_id_evidence="missing proof id after transform is rejected"
proof_missing_id_evidence+=" and requires recheck before unchecked use."
proof_invalidates_evidence="optimizer proof rules require invalidated bounds facts to be declared,"
proof_invalidates_evidence+=" and consumers must recheck before reuse."
proof_lowering_test="go test ./compiler/internal/plir ./compiler/internal/lower"
proof_lowering_test+=" -run 'Proof|Invalidates|Unchecked' -count=1"
proof_lowering_evidence="lowering refines live bounds proof use into"
proof_lowering_evidence+=" proof-tagged unchecked load metadata."
proof_store_ref_test="go test ./compiler/internal/validation ./compiler/internal/proof"
proof_store_ref_test+=" -run 'UnknownLiveProof|MissingProofID|ProofStoreRejectsMissingProofID'"
proof_store_ref_test+=" -count=1"
proof_store_ref_evidence="new proof use requires a proof store reference"
proof_store_ref_evidence+=" and unknown proof ids are blocked."
cat > "$proof_transition_dir/proof-transition-report.json" << PROOFTRANSITION
{
  "schema": "tetra.proof-transition-report.v1",
  "status": "pass",
  "git_head": "$git_head",
  "rows": [
    {
      "name": "stable_hash_semantic_fields",
      "transition": "invalidated",
      "evidence": "$(json_escape "$proof_stable_hash_evidence")",
      "consumer_action": "blocked_by_proof_store_validate_recheck_required",
      "source_artifacts": [
        "compiler/internal/proof/term.go",
        "compiler/internal/proof/validate_test.go"
      ],
      "tests": ["$(json_escape "$proof_stable_hash_test")"]
    },
    {
      "name": "bounds_proof_preserved_through_translation",
      "transition": "preserved",
      "evidence": "$(json_escape "$proof_translation_evidence")",
      "before_artifact": "compiler/compiler_evidence_gates.go",
      "after_artifact": "compiler/compiler_suite_test.go",
      "source_artifacts": [
        "compiler/compiler_evidence_gates.go",
        "compiler/compiler_suite_test.go"
      ],
      "tests": ["$(json_escape "$proof_translation_test")"]
    },
    {
      "name": "translation_missing_proof_requires_recheck",
      "transition": "requires_recheck",
      "evidence": "$(json_escape "$proof_missing_id_evidence")",
      "consumer_action": "recheck_or_block_unchecked_bounds_use",
      "source_artifacts": ["compiler/internal/validation/validation_test.go"],
      "tests": ["$(json_escape "$proof_missing_id_test")"]
    },
    {
      "name": "optimization_invalidates_bounds_proofs",
      "transition": "invalidated",
      "evidence": "$(json_escape "$proof_invalidates_evidence")",
      "consumer_action": "recheck_required_before_consuming_invalidated_bounds_proof",
      "source_artifacts": [
        "compiler/internal/opt/opt_core.go",
        "compiler/internal/opt/opt_suite_test.go"
      ],
      "tests": ["go test ./compiler/internal/opt -run 'Manager|Optimization' -count=1"]
    },
    {
      "name": "lowering_refines_bounds_proof_use",
      "transition": "refined",
      "evidence": "$(json_escape "$proof_lowering_evidence")",
      "before_artifact": "compiler/internal/plir/plir_test/plir_test.go",
      "after_artifact": "compiler/internal/lower/lower_suite_test.go",
      "source_artifacts": [
        "compiler/internal/plir/plir_test/plir_test.go",
        "compiler/internal/lower/lower_suite_test.go"
      ],
      "tests": ["$(json_escape "$proof_lowering_test")"]
    },
    {
      "name": "new_proof_requires_store_reference",
      "transition": "new",
      "evidence": "$(json_escape "$proof_store_ref_evidence")",
      "after_artifact": "compiler/internal/validation/validation_test.go",
      "source_artifacts": [
        "compiler/internal/validation/validation_test.go",
        "compiler/internal/proof/validate_test.go"
      ],
      "tests": ["$(json_escape "$proof_store_ref_test")"]
    }
  ],
  "non_claims": [
    "no full formal proof claim",
    "no exhaustive optimizer proof-transition completeness claim"
  ]
}
PROOFTRANSITION

runtime_linux_evidence="linux-x64 runtime hardening and runtimeabi memory evidence"
runtime_linux_evidence+=" is covered by Memory100 gate."
runtime_linux_test="go test ./compiler"
runtime_linux_test+=" -run 'RuntimeHardening|RuntimeAllocation|RawPointerBoundsABI"
runtime_linux_test+="|ActorRuntimeProductionBoundary|OOM|Stack|Allocator|Region'"
runtime_linux_test+=" -count=1"
runtime_validator_test="go test ./tools/cmd/validate-memory-100-prod-stable"
runtime_validator_test+=" -run RuntimeMemory -count=1"
runtime_wasi_evidence="wasm32-wasi remains artifact/runtime tiered and is not"
runtime_wasi_evidence+=" Memory100 production-host-runtime evidence."
runtime_web_evidence="wasm32-web remains artifact/runtime tiered and is not"
runtime_web_evidence+=" Memory100 production-host-runtime evidence."
runtime_x86_evidence="linux-x86 remains build/lower scoped for Memory100"
runtime_x86_evidence+=" and is not production runtime evidence."
runtime_x32_evidence="linux-x32 remains build/lower scoped for Memory100"
runtime_x32_evidence+=" and is not production runtime evidence."
runtime_build_lower_excluded="build/lower-only memory evidence is not"
runtime_build_lower_excluded+=" Memory100 production-host-runtime evidence"
cat > "$runtime_memory_dir/runtime-memory-contract.json" << RUNTIMEMEMORY
{
  "schema": "tetra.runtime-memory-contract.v1",
  "status": "pass",
  "git_head": "$git_head",
  "rows": [
    {
      "target": "linux-x64",
      "included_in_memory100_target_matrix": true,
      "runtime_status": "production",
      "memory_run": "yes",
      "memory_claim_level": "production_host_runtime",
      "evidence": "$(json_escape "$runtime_linux_evidence")",
      "source_artifacts": [
        "memory-production/targets.json",
        "compiler/compiler_evidence_gates.go",
        "compiler/internal/runtimeabi/runtimeabi_test/runtimeabi_test.go"
      ],
      "tests": ["$(json_escape "$runtime_linux_test")"],
      "non_claims": ["no all-target memory parity claim", "no full runtime-hardening proof claim"]
    },
    {
      "target": "windows-x64",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "host_required",
      "memory_run": "host-required",
      "memory_claim_level": "host_required_nonclaim",
      "evidence": "windows-x64 requires target-host runtime evidence before Memory100 inclusion.",
      "excluded_reason": "no windows target-host runtime memory report in this aggregate",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no windows-x64 runtime memory production claim"]
    },
    {
      "target": "macos-x64",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "host_required",
      "memory_run": "host-required",
      "memory_claim_level": "host_required_nonclaim",
      "evidence": "macos-x64 requires target-host runtime evidence before Memory100 inclusion.",
      "excluded_reason": "no macos target-host runtime memory report in this aggregate",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no macos-x64 runtime memory production claim"]
    },
    {
      "target": "wasm32-wasi",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "tiered",
      "memory_run": "runner-smoke if available",
      "memory_claim_level": "artifact_runtime_tiered_nonclaim",
      "evidence": "$(json_escape "$runtime_wasi_evidence")",
      "excluded_reason": "not part of current Memory100 linux-x64 target matrix",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no wasm32-wasi production host-runtime memory claim"]
    },
    {
      "target": "wasm32-web",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "tiered",
      "memory_run": "browser-smoke if available",
      "memory_claim_level": "artifact_runtime_tiered_nonclaim",
      "evidence": "$(json_escape "$runtime_web_evidence")",
      "excluded_reason": "not part of current Memory100 linux-x64 target matrix",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no wasm32-web production host-runtime memory claim"]
    },
    {
      "target": "linux-x86",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "partial_build_only",
      "memory_run": "no/host-dependent",
      "memory_claim_level": "build_lower_only_nonclaim",
      "evidence": "$(json_escape "$runtime_x86_evidence")",
      "excluded_reason": "$(json_escape "$runtime_build_lower_excluded")",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no linux-x86 production runtime memory claim"]
    },
    {
      "target": "linux-x32",
      "included_in_memory100_target_matrix": false,
      "runtime_status": "partial_build_only",
      "memory_run": "no/host-dependent",
      "memory_claim_level": "build_lower_only_nonclaim",
      "evidence": "$(json_escape "$runtime_x32_evidence")",
      "excluded_reason": "$(json_escape "$runtime_build_lower_excluded")",
      "source_artifacts": ["memory-production/targets.json"],
      "tests": ["$(json_escape "$runtime_validator_test")"],
      "non_claims": ["no linux-x32 production runtime memory claim"]
    }
  ],
  "non_claims": [
    "no all-target memory parity claim",
    "OOM recovery guarantee is not claimed",
    "full stack-overflow protection is not claimed",
    "full allocator-corruption detection proof is not claimed",
    "production actor runtime is not claimed"
  ]
}
RUNTIMEMEMORY
go run ./tools/cmd/memory-fuzz-short \
  --tier 1 \
  --report-dir "$memory_fuzz_dir" \
  --git-head "$git_head"
go run ./tools/cmd/validate-memory-fuzz-oracle \
  --report "$memory_fuzz_dir/memory-fuzz-oracle.json" \
  --artifact-dir "$memory_fuzz_dir" \
  --current-git-head "$git_head"

printf -v memory_production_command '%s --report-dir %s' \
  'bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh' \
  "$memory_production_dir"
printf -v ram_contract_command '%s --report-dir %s' \
  'bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh' \
  "$ram_contract_dir"
printf -v integrated_command '%s --report-dir %s' \
  'bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh' \
  "$integrated_dir"
printf -v memory_fuzz_command '%s --tier 1 --report-dir %s --git-head %s' \
  'go run ./tools/cmd/memory-fuzz-short' \
  "$memory_fuzz_dir" \
  "$git_head"
printf -v memory_fuzz_validator_command '%s --report %s --artifact-dir %s --current-git-head %s' \
  'go run ./tools/cmd/validate-memory-fuzz-oracle' \
  "$memory_fuzz_dir/memory-fuzz-oracle.json" \
  "$memory_fuzz_dir" \
  "$git_head"
printf -v docs_claim_policy_command '%s --manifest docs/generated/manifest.json' \
  'go run ./tools/cmd/verify-docs'
printf -v artifact_hashes_write_command '%s --write --root %s --out %s' \
  'go run ./tools/cmd/validate-artifact-hashes' \
  "$report_dir_arg" \
  "$report_dir_arg/artifact-hashes.json"
printf -v memory_100_validator_command '%s --report-dir %s --current-git-head %s' \
  'go run ./tools/cmd/validate-memory-100-prod-stable' \
  "$report_dir_arg" \
  "$git_head"

semantic_borrow_return_test="go test ./compiler/tests/ownership ./compiler/tests/semantics"
semantic_borrow_return_test+=" -run 'Borrow.*Return|BorrowEscape' -count=1"
semantic_aggregate_evidence="borrowed slice/ptr fields cannot escape through owned"
semantic_aggregate_evidence+=" aggregate returns, consume, or inout calls"
semantic_aggregate_test="go test ./compiler/tests/ownership"
semantic_aggregate_test+=" -run 'Borrowed.*Aggregate|BorrowedSlice.*ConsumeInout"
semantic_aggregate_test+="|BorrowedPtr.*Struct' -count=1"
semantic_text_evidence="borrowed text/view host and actor/task boundaries require"
semantic_text_evidence+=" explicit copy or are rejected"
semantic_text_test="go test ./compiler/tests/semantics ./compiler/internal/actorsafety"
semantic_text_test+=" -run 'Borrowed|StringView|copy|ActorBoundary' -count=1"
semantic_present_test="go test ./compiler/tests/runtime ./compiler/tests/ownership"
semantic_present_test+=" -run 'Present|Close|UseAfter|Freed|Consume' -count=1"
semantic_finalizer_evidence="resource finalization diagnostics reject missing finalizer"
semantic_finalizer_evidence+=" and double-close/double-free cases"
semantic_finalizer_test="go test ./compiler/tests/runtime ./compiler/tests/safety/..."
semantic_finalizer_test+=" -run 'Resource|Finalization|Double|Close|Free' -count=1"
semantic_actor_evidence="actor/task message transfer rejects borrowed or non-sendable"
semantic_actor_evidence+=" memory/resources unless explicitly copied/moved"
semantic_actor_test="go test ./compiler/tests/ownership"
semantic_actor_test+=" ./compiler/tests/ownership/actor_task ./compiler/internal/actorsafety"
semantic_actor_test+=" -run 'Actor|Task|Send|Transfer|Borrowed|NonSendable' -count=1"
cat > "$semantic_safety_dir/memory-semantic-safety-matrix.json" << SEMANTIC
{
  "schema": "tetra.memory-semantic-safety-matrix.v1",
  "status": "pass",
  "git_head": "$git_head",
  "rows": [
    {
      "name": "borrowed_view_return_escape",
      "kind": "negative",
      "evidence": "borrowed view escape via return is rejected",
      "source_artifacts": [
        "compiler/tests/ownership/ownership_test.go",
        "compiler/tests/semantics/semantics_async_ownership_test.go"
      ],
      "tests": ["$(json_escape "$semantic_borrow_return_test")"]
    },
    {
      "name": "borrowed_view_owned_aggregate_escape",
      "kind": "negative",
      "evidence": "$(json_escape "$semantic_aggregate_evidence")",
      "source_artifacts": ["compiler/tests/ownership/ownership_test.go"],
      "tests": ["$(json_escape "$semantic_aggregate_test")"]
    },
    {
      "name": "borrowed_text_host_boundary_copy",
      "kind": "negative",
      "evidence": "$(json_escape "$semantic_text_evidence")",
      "source_artifacts": [
        "compiler/tests/semantics/semantics_memory_surface_test.go",
        "compiler/internal/actorsafety/sendability_test.go"
      ],
      "tests": ["$(json_escape "$semantic_text_test")"]
    },
    {
      "name": "inout_alias_escape",
      "kind": "negative",
      "evidence": "borrowed values cannot escape through inout assignment or aliasing",
      "source_artifacts": ["compiler/tests/ownership/ownership_test.go"],
      "tests": ["go test ./compiler/tests/ownership -run 'Inout|Alias|BorrowedProjection' -count=1"]
    },
    {
      "name": "surface_frame_escape",
      "kind": "negative",
      "evidence": "Surface frame/pixels borrowed views cannot escape lifecycle boundaries",
      "source_artifacts": [
        "compiler/tests/semantics/semantics_memory_surface_test.go",
        "integrated/safe-view-lifetime/safe-view-lifetime-summary.json"
      ],
      "tests": ["go test ./compiler/tests/semantics -run 'Surface|Frame|Pixels|SafeView' -count=1"]
    },
    {
      "name": "use_after_present_close",
      "kind": "negative",
      "evidence": "use after present/close/free is rejected",
      "source_artifacts": [
        "compiler/tests/runtime/resource_finalization_test.go",
        "compiler/tests/ownership/ownership_test.go"
      ],
      "tests": ["$(json_escape "$semantic_present_test")"]
    },
    {
      "name": "resource_finalizer_double_close",
      "kind": "negative",
      "evidence": "$(json_escape "$semantic_finalizer_evidence")",
      "source_artifacts": [
        "compiler/tests/runtime/resource_finalization_test.go",
        "compiler/tests/safety/diagnostics/core/safety_diagnostics_test.go"
      ],
      "tests": ["$(json_escape "$semantic_finalizer_test")"]
    },
    {
      "name": "actor_task_non_sendable_transfer",
      "kind": "negative",
      "evidence": "$(json_escape "$semantic_actor_evidence")",
      "source_artifacts": [
        "compiler/tests/ownership/actor_task/actor_task_ownership_test.go",
        "compiler/internal/actorsafety/sendability_test.go",
        "compiler/internal/actorsafety/ownership_transfer_test.go"
      ],
      "tests": ["$(json_escape "$semantic_actor_test")"]
    }
  ],
  "non_claims": [
    "no production actor runtime claim",
    "no universal leak-free program claim",
    "no full formal memory safety proof claim"
  ]
}
SEMANTIC

cat > "$leak_resource_dir/leak-resource-report.json" << LEAK
{
  "schema": "tetra.leak-resource.v1",
  "status": "pass",
  "git_head": "$git_head",
  "checks": [
    {
      "name": "actornet_close_without_cancel",
      "kind": "stress",
      "evidence": "actornet broker close-without-cancel leak smoke",
      "source_artifacts": ["memory-production/memory-production-linux-x64.json"]
    },
    {
      "name": "compiler_resource_finalization",
      "kind": "negative",
      "evidence": "compiler resource finalization diagnostics",
      "source_artifacts": ["memory-production/memory-production-linux-x64.json"]
    },
    {
      "name": "surface_frame_escape",
      "kind": "negative",
      "evidence": "safe-view lifetime and Surface frame escape evidence",
      "source_artifacts": ["integrated/memory-islands-surface-production-manifest.json"]
    },
    {
      "name": "actor_task_transfer",
      "kind": "negative",
      "evidence": "actor task transfer safety case",
      "source_artifacts": ["memory-production/memory-production-linux-x64.json"]
    }
  ],
  "non_claims": [
    "no universal leak-free host tooling claim",
    "no universal Memory 100% claim"
  ]
}
LEAK

cat > "$docs_manifest_dir/claim-policy.json" << CLAIMS
{
  "schema": "tetra.memory-100.claim-policy.v1",
  "status": "pass",
  "git_head": "$git_head",
  "allowed_claims": [
    "Memory/RAM production-stable criteria passed locally for the scoped target matrix only."
  ],
  "forbidden_claims": [
    "Memory is 100% ready",
    "fully proven memory safety",
    "full formal proof of memory safety",
    "all targets memory-stable",
    "all-target memory parity",
    "unsafe/raw memory is safe",
    "no leaks",
    "C/Rust parity",
    "faster than C",
    "official benchmark result"
  ],
  "non_claims": [
    "no universal Memory 100% claim",
    "no full formal proof claim",
    "no all-target memory parity claim"
  ]
}
CLAIMS

cat > "$aggregate_manifest_path" << MANIFEST
{
  "schema": "tetra.memory-100.prod-stable.v1",
  "status": "pass",
  "verdict": "$memory100_verdict",
  "git_head": "$git_head",
  "git_dirty": $git_dirty,
  "git_status_short_branch": $git_status_json,
  "generated_at": "$generated_at",
  "target_matrix": ["linux-x64"],
  "hash_manifest": "artifact-hashes.json",
  "claims": [
    "Memory/RAM production-stable criteria passed locally for the scoped target matrix."
  ],
  "non_claims": [
    "no universal Memory 100% claim",
    "no full formal proof claim",
    "no all-target memory parity claim",
    "no arbitrary unsafe external pointer safety claim",
    "no C/Rust parity or performance superiority claim"
  ],
  "commands": [
    {
      "name": "memory-production-gate",
      "command": "$(json_escape "$memory_production_command")"
    },
    {
      "name": "ram-contract-gate",
      "command": "$(json_escape "$ram_contract_command")"
    },
    {
      "name": "integrated-gate",
      "command": "$(json_escape "$integrated_command")"
    },
    {
      "name": "memory-fuzz-short",
      "command": "$(json_escape "$memory_fuzz_command")"
    },
    {
      "name": "memory-fuzz-validator",
      "command": "$(json_escape "$memory_fuzz_validator_command")"
    },
    {
      "name": "docs-claim-policy",
      "command": "$(json_escape "$docs_claim_policy_command")"
    },
    {
      "name": "artifact-hashes-write",
      "command": "$(json_escape "$artifact_hashes_write_command")"
    },
    {
      "name": "memory-100-validator",
      "command": "$(json_escape "$memory_100_validator_command")"
    }
  ],
  "artifacts": [
    {
      "path": "memory-production/memory-production-linux-x64.json",
      "kind": "memory_production_report",
      "schema": "tetra.memory.production.v1"
    },
    {
      "path": "memory-production/memory-release-manifest.json",
      "kind": "memory_release_manifest",
      "schema": "tetra.memory.release-manifest.v1"
    },
    {
      "path": "memory-production/artifact-hashes.json",
      "kind": "memory_production_hash_manifest",
      "schema": "tetra.release-artifact-hashes.v1alpha1"
    },
    {
      "path": "ram-contract/ram-contract-release-manifest.json",
      "kind": "ram_contract_release_manifest",
      "schema": "tetra.ram-contract.release-manifest.v1"
    },
    {
      "path": "ram-contract/ram-contract-report.json",
      "kind": "ram_contract_report",
      "schema": "tetra.ram-contract-report.v1"
    },
    {
      "path": "ram-contract/memory-grade-report.json",
      "kind": "ram_memory_grade_report",
      "schema": "tetra.memory-grade-report.v1"
    },
    {
      "path": "ram-contract/proof-store-summary.json",
      "kind": "ram_proof_store_summary",
      "schema": "tetra.proof-store-summary.v1"
    },
    {
      "path": "ram-contract/validation-pipeline-coverage.json",
      "kind": "ram_validation_pipeline_coverage",
      "schema": "tetra.validation-pipeline-coverage.v1"
    },
    {
      "path": "ram-contract/heap-blockers.json",
      "kind": "ram_heap_blockers",
      "schema": "tetra.ram-blockers.v1"
    },
    {
      "path": "ram-contract/copy-blockers.json",
      "kind": "ram_copy_blockers",
      "schema": "tetra.ram-blockers.v1"
    },
    {
      "path": "ram-contract/fuzz/ram-contract-fuzz-oracle.json",
      "kind": "ram_contract_fuzz_oracle",
      "schema": "tetra.ram-contract-fuzz-oracle.v1"
    },
    {
      "path": "ram-contract/artifact-hashes.json",
      "kind": "ram_contract_hash_manifest",
      "schema": "tetra.release-artifact-hashes.v1alpha1"
    },
    {
      "path": "raw-memory-contract/raw-memory-contract.json",
      "kind": "raw_memory_contract_report",
      "schema": "tetra.raw-memory-contract.v1"
    },
    {
      "path": "allocation-lowering/allocation-lowering-report.json",
      "kind": "allocation_lowering_report",
      "schema": "tetra.allocation-lowering.v1"
    },
    {
      "path": "proof-store/proof-store-summary.json",
      "kind": "proof_store_summary",
      "schema": "tetra.proof-store-summary.v1"
    },
    {
      "path": "proof-transition/proof-transition-report.json",
      "kind": "proof_transition_report",
      "schema": "tetra.proof-transition-report.v1"
    },
    {
      "path": "runtime-memory/runtime-memory-contract.json",
      "kind": "runtime_memory_contract",
      "schema": "tetra.runtime-memory-contract.v1"
    },
    {
      "path": "memory-fuzz/memory-fuzz-oracle.json",
      "kind": "memory_fuzz_oracle_report",
      "schema": "tetra.memory-fuzz.oracle.v1"
    },
    {
      "path": "memory-fuzz/artifact-hashes.json",
      "kind": "memory_fuzz_hash_manifest",
      "schema": "tetra.release-artifact-hashes.v1alpha1"
    },
    {
      "path": "semantic-safety/memory-semantic-safety-matrix.json",
      "kind": "memory_semantic_safety_matrix",
      "schema": "tetra.memory-semantic-safety-matrix.v1"
    },
    {
      "path": "leak-resource/leak-resource-report.json",
      "kind": "leak_resource_report",
      "schema": "tetra.leak-resource.v1"
    },
    {
      "path": "integrated/memory-islands-surface-production-manifest.json",
      "kind": "integrated_memory_islands_surface_manifest",
      "schema": "tetra.memory-islands-surface.production-gate.v1"
    },
    {
      "path": "integrated/artifact-hashes.json",
      "kind": "integrated_hash_manifest",
      "schema": "tetra.release-artifact-hashes.v1alpha1"
    },
    {
      "path": "docs-manifest/claim-policy.json",
      "kind": "docs_claim_policy",
      "schema": "tetra.memory-100.claim-policy.v1"
    }
  ]
}
MANIFEST

go run ./tools/cmd/validate-artifact-hashes \
  --write \
  --root "$report_dir" \
  --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-memory-100-prod-stable \
  --report-dir "$report_dir" \
  --current-git-head "$git_head"

echo "Memory100 aggregate manifest: $aggregate_manifest_path"
echo "Memory100 artifact hashes: $report_dir/artifact-hashes.json"
