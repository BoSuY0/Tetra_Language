# Tetra Surface/UI Production Control

## Status Contract

status_file: `PLAN.md`
attempt_log: `ATTEMPTS.md`
durable_notes: `NOTES.md`
workflow_dir: `.workflow/surface-ui-production-v1/`
external_plan: `/home/tetra/Downloads/surface-ui-production-implementation-plan.md`
update_memory_after: every_packet, every_validation_gate, every_blocker
check_control_before: phase_change, strategic_pivot, expensive_step,
release_gate_change

## Human Priorities

primary_priority: production_ready_scoped_surface_v1
secondary_priority: honest_target_scoped_release_evidence
third_priority: release_gate_freshness_and_fake_evidence_rejection
fourth_priority: preserve_unrelated_dirty_worktree_changes

## Scope Knobs

allowed_files:
- Surface libraries, examples, compiler/lowering/proof files named by the plan
- `scripts/release/surface/`
- `scripts/release/safe-view-lifetime/`
- `scripts/analysis/`
- Surface validators and related command validators
- CI/release workflows
- docs/spec/user/release files named by the plan
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- `.workflow/surface-ui-production-v1/`
- `reports/surface-ui-production-*`

protected_files:
- unrelated dirty files not required by the current packet
- previous `.workflow/memory-*` final evidence except additive cross-links
- generated dumps unless explicitly requested

max_blast_radius: implement the external Surface/UI production plan while
keeping changes tied to explicit `SURFPROD-*` packets and validators.

## Resource Knobs

max_runtime_per_step: none
max_parallel_jobs: repo_default
network_allowed: false_for_normal_work
external_api_allowed: false_for_normal_work
go_cache: persistent `.cache/go-build-surface-*` or
`${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-surface-*`

## Decision Gates

require_approval_for:
- destructive_change
- revert_or_delete_unrelated_dirty_file
- dependency_change
- broad cross-platform UI claim
- unsupported target promotion
- GPU/native-widget/DOM/React/rich-text/full-screen-reader claim
- release gate pass from starter or blocked evidence
- scope_expansion_beyond_external_plan

## Latest Human Nudge

Implement the full plan from
`/home/tetra/Downloads/surface-ui-production-implementation-plan.md` using
goal-forge, goal-loop, and define-goal discipline.

## Latest Batch

latest_completed: `SURFPROD-P12` docs/non-claims/user guide correction
latest_attempted: `SURFPROD-P13` final production release candidate gate
latest_evidence:
fresh final Surface release, experimental regression, safe-view lifetime, and
API stability gates under `reports/surface-ui-production-final/`;
`GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-prod-final go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`;
focused Surface/UI race gate;
`GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-prod-final bash scripts/ci/test.sh`;
docs/manifest/hash validators;
`git diff --check`;
manifest idempotence diff;
`graphify update .`.
latest_caveat: exact `git diff --exit-code -- docs/generated/manifest.json`
is non-zero because the tracked generated manifest changed with
`core.island_reset`; final temp generation idempotence passed at
`reports/surface-ui-production-final/generated-manifest-check/manifest.generated.json`.
next_recommended: resolve or explicitly accept the generated-manifest
cleanliness caveat; only then call `update_goal complete`.
