# Control: Memory Ideal Vertical Slice v8 Report Integrity

## Status Contract

status_file: `.workflow/memory-ideal-vertical-slice-v8-report/plan.md`
attempt_log: `.workflow/memory-ideal-vertical-slice-v8-report/attempts.md`
durable_notes: `.workflow/memory-ideal-vertical-slice-v8-report/notes.md`
update_memory_after: every RED/GREEN cluster or gate result
check_control_before: phase_change, strategic_pivot, expensive_step

## Human Priorities

primary_priority: evidence_quality
secondary_priority: behavior_preservation

## Scope Knobs

allowed_files:
- `compiler/internal/memoryfacts/**`
- `tools/cmd/validate-memory-report/**`
- `tools/cmd/validate-memory-correlation/**`
- `docs/audits/memory-ideal-vslice-v8-report-*.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v8-report/**`
- `GOAL.md`

protected_files:
- unrelated dirty files outside v8 scope

max_blast_radius: memory report/correlation integrity only

## Resource Knobs

max_runtime_per_step: use focused gates before broad gates
max_parallel_jobs: parallel independent reads/tests; avoid concurrent broad CI
network_allowed: false unless explicitly needed
external_api_allowed: false

## Decision Gates

require_approval_for:
- scope_expansion
- dependency_change
- broad_schema_redesign
- performance_or_target_claim
- runtime_or_ffi_safety_claim
- destructive_cleanup_of_dirty_worktree

## Sidecar Inputs

sidecar_apply_cadence: before_phase_change
nudge_file: none
human_overlay_file: none
review_queue_file: none

## Latest Human Nudge

Start `MEM-REPORT-008` after accepted v7, prioritize claim/projection
integrity before new semantic slices.
