# Tetra Memory + IslandKernel Production Control

## Status Contract

status_file: `PLAN.md`
attempt_log: `ATTEMPTS.md`
durable_notes: `NOTES.md`
workflow_dir: `.workflow/memory-islands-production-v1/`
external_plan: `/home/tetra/Downloads/tetra-memory-islands-production-plan.md`
update_memory_after: every_packet, every_validation_gate, every_blocker
check_control_before: phase_change, strategic_pivot, expensive_step,
release_gate_change

## Human Priorities

primary_priority: production_ready_supported_island_memory_surface
secondary_priority: host_leak_reliability_and_release_attestation
third_priority: claim_honesty_and_conservative_non_goals
fourth_priority: preserve_unrelated_dirty_worktree_changes

## Scope Knobs

allowed_files:
- compiler island/memory packages named by the external plan
- `cli/internal/actornet` host lifecycle files/tests
- memory/island validators, scripts, workflows, and docs named by the plan
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- `.workflow/memory-islands-production-v1/`
- `reports/memory-islands-production/`

protected_files:
- unrelated dirty files not required by the current packet
- previous `.workflow/memory-production-ready-v1/` final evidence, except
  additive cross-links
- generated dumps unless explicitly requested

max_blast_radius: implement the external IslandKernel production plan while
keeping changes tied to explicit `MEM-ISLAND-*` packets and validators.

## Resource Knobs

max_runtime_per_step: none
max_parallel_jobs: repo_default
network_allowed: false_for_normal_work
external_api_allowed: false_for_normal_work
go_cache: persistent `.cache/go-build-memory-islands-*` or
`${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-memory-islands-*`

## Decision Gates

require_approval_for:
- destructive_change
- revert_or_delete_unrelated_dirty_file
- dependency_change
- broad target parity claim
- arbitrary unsafe proof claim
- full actor runtime scheduler proof claim
- persistent memory subsystem invention if packages remain absent
- performance/official benchmark/fastest-language claim
- scope_expansion_beyond_external_plan

## Latest Human Nudge

Implement the full plan from
`/home/tetra/Downloads/tetra-memory-islands-production-plan.md` using
goal-forge, goal-loop, and define-goal discipline.

## Latest Batch

latest_completed: `MEM-ISLAND-P05` BCE typed proof IR slice
latest_evidence: RED/GREEN added PLIR `ProofTerm` metadata for bounds-check
proofs, typed term verifier checks for subject base/index/range plus island
epoch/base fields, validation `ProofReport` propagation, memoryfacts/report/CLI
typed proof fields, and explicit-island allocplan identity projection from
PLIR; latest P05 package sweep, compiler report sweep, `git diff --check`, and
`graphify update .` passed with `21934 nodes`, `68314 edges`,
`1191 communities`
next_recommended: start `MEM-ISLAND-P06` storage/lowering truth; keep noalias,
storage, and island-move proof terms explicit, sanitizer traps for P10,
independent verifier for P11, and release attestation for P16
