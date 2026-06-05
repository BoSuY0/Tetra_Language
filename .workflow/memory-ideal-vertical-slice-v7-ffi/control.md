# Control: Memory Ideal Vertical Slice v7 FFI

## Status Contract

status_file: `.workflow/memory-ideal-vertical-slice-v7-ffi/plan.md`
attempt_log: `.workflow/memory-ideal-vertical-slice-v7-ffi/attempts.md`
durable_notes: `.workflow/memory-ideal-vertical-slice-v7-ffi/notes.md`
update_memory_after: every_red_green_cluster
check_control_before: phase_change, strategic_pivot, expensive_step, schema_change

## Human Priorities

primary_priority: evidence_quality
secondary_priority: minimal_blast_radius

## Scope Knobs

allowed_files:
- `compiler/internal/memoryfacts/**`
- `compiler/internal/memorymodel/**`
- `compiler/internal/semantics/**`
- `compiler/internal/plir/**`
- `compiler/internal/lower/**`
- `compiler/internal/runtimeabi/raw_pointer_bounds.go`
- `compiler/internal/runtimeabi/raw_pointer_bounds_test.go`
- `compiler/ffi_target.go`
- `compiler/ffi_target_diagnostics_test.go`
- `compiler/tests/semantics/**`
- `compiler/tests/ownership/**`
- `tools/cmd/validate-memory-report/**`
- `tools/cmd/validate-memory-correlation/**`
- `tools/cmd/validate-memory-fuzz-oracle/**`
- `docs/audits/memory-ideal-vslice-v7-ffi-*.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/spec/unsafe.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v7-ffi/**`
- `GOAL.md`

protected_files:
- global `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and `CONTROL.md`
- unrelated Surface release workflow files
- unrelated target parity/performance files
- unrelated production actor/task/runtime files

max_blast_radius: v7 memory FFI/external pointer evidence slice only

## Resource Knobs

max_runtime_per_step: none
max_parallel_jobs: repo_default
network_allowed: false
external_api_allowed: false

## Decision Gates

require_approval_for:
- strategic_pivot
- destructive_change
- dependency_change
- schema_expansion_outside_memory_report_schema
- public_api_change_outside_v7_memory_evidence
- scope_expansion
- safe_wrapper_promotion_claim
- arbitrary_ffi_lifetime_claim
- arbitrary_external_pointer_safety_claim
- performance_or_target_parity_claim

## Sidecar Inputs

sidecar_apply_cadence: before_phase_change
nudge_file: none
human_overlay_file: none
review_queue_file: none

## Latest Human Nudge

Proceed with `MEM-FFI-007` as the next vertical slice after accepted v6. Keep
the claim narrow: external/FFI/raw pointer boundary trust must remain
conservative or rejected unless compiler-owned proof exists.
