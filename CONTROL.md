# CONTROL

## Status Contract

status_file: `PLAN.md`
attempt_log: `ATTEMPTS.md`
durable_notes: `NOTES.md`
goal_contract: `GOAL.md`
canonical_plan: `docs/plans/2026-06-16-surface-morph-rendered-beauty-implementation-plan.md`
update_memory_after: every_meaningful_attempt
check_control_before: phase_change, strategic_pivot, expensive_step, subagent_result_integration

## Human Priorities

primary_priority: visual_polish
secondary_priority: evidence_quality

## Scope Knobs

allowed_files:
- `docs/spec/surface_*`
- `docs/surface/**`
- `docs/release/**`
- `docs/generated/manifest.json`
- `lib/core/**`
- `tools/cmd/surface-runtime-smoke/**`
- `tools/cmd/surface-visual-diff/**`
- `tools/cmd/validate-surface-*`
- `tools/validators/surface/**`
- `tools/internal/surfacerender/**`
- `cli/cmd/tetra/**`
- `scripts/release/surface/**`
- `examples/surface_*`
- `examples/projects/tetra_control_center/**`
- `reports/**`
- `reports/stabilization/**`
- `graphify-out/**`

protected_files:
- unrelated dirty worktree files outside the active task scope
- `.git/**`
- external caches outside repo-specific `.cache/`

max_blast_radius: Surface Morph rendered beauty stack and directly related release evidence.

## Subagent Policy

read_only_allowed:
- `agent_type=explorer` (`gpt-5.4-mini`)
- `agent_type=explorer_fast` (`gpt-5.3-codex-spark`)

write_allowed:
- `agent_type=worker`
- model: `gpt-5.5`
- reasoning: `xhigh`
- required spawn flag: `fork_context=true`

write_disallowed:
- any delegated editing agent with another model
- any delegated editing agent without `fork_context=true`
- overlapping write scopes between worker agents

If the required write-agent configuration is unavailable, do not substitute a
different delegated editor. Ask the user or continue only with parent-agent
edits where appropriate.

## Resource Knobs

network_allowed: false_by_default_for_implementation
external_api_allowed: approval_required
max_parallel_read_agents: 3
max_parallel_write_agents: 1_per_disjoint_scope

## Decision Gates

require_approval_for:
- strategic_pivot
- destructive_change
- dependency_change
- schema_or_migration_change_outside_surface_evidence
- public_api_change_outside_morph_block_surface
- scope_expansion
- unsupported_product_claim

## Latest Human Nudge

Create and run a `/goal` for the full Morph rendered beauty plan. Use subagents
with strict model policy: read-only only `gpt-5.4-mini` / `gpt-5.3-codex-spark`;
delegated editing only `gpt-5.5`, reasoning `xhigh`, `fork_context=true`.
