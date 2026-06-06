# MEM-RELEASE-013 Control

## Status Contract

status_file: `PLAN.md`
attempt_log: `ATTEMPTS.md`
durable_notes: `NOTES.md`
workflow_dir: `.workflow/memory-release-v13/`
update_memory_after: every_status_capture, every_validation_gate
check_control_before: phase_change, strategic_pivot, expensive_step,
dirty_entry_decision

## Human Priorities

primary_priority: release_evidence_integrity
secondary_priority: preserve_unrelated_dirty_worktree_changes

## Scope Knobs

allowed_files:
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- `.workflow/memory-release-v13/`
- `reports/memory-release-v13/`
- `reports/memory-fuzz-short/v13/`
- existing memory evidence/docs/manifest files only if a validator requires an
  additive evidence reference

protected_files:
- unrelated dirty files from `git status --short`
- existing v10/v11/v12 workflow artifacts unless referenced additively
- `docs/assets/` unless explicitly classified and approved
- source files unrelated to release evidence freeze

max_blast_radius: memory evidence freeze and dirty worktree triage only.

## Resource Knobs

max_runtime_per_step: none
max_parallel_jobs: repo_default
network_allowed: false_for_normal_work
external_api_allowed: false_for_normal_work
go_cache: persistent `.cache/go-build-memory-v13-release-*` or
`${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-memory-v13-release-*`

## Decision Gates

require_approval_for:
- destructive_change
- revert_or_delete_unrelated_dirty_file
- archive_unrelated_dirty_file
- dependency_change
- new_memory_semantics
- broad_fuzz_runtime_requirement
- target_parity_claim
- arbitrary_unsafe_proof_claim
- long_nightly_run_as_mandatory_tier1
- clean_release_claim_while_status_dirty
- performance_claim
- scope_expansion

## Sidecar Inputs

sidecar_apply_cadence: between_phases_only
nudge_file: none
human_overlay_file: none
review_queue_file: none

## Latest Human Nudge

Use the 2026-06-06 v12 accepted verdict as baseline and start
`MEM-RELEASE-013` as a release/evidence hygiene slice. Freeze and classify
dirty worktree state, regenerate v13 fuzz evidence, run release validators, and
preserve nonclaims.
