# Tetra Surface Release Promotion v1 Execution Plan

Canonical plan:

- `/home/tetra/Downloads/tetra_surface_release_promotion_v1_full_plan.md`

Operational rules:

- Work one vertical slice at a time.
- Use Graphify MCP before architecture/code navigation.
- Inspect concrete files before editing.
- Prefer TDD for feature and bugfix work when practical.
- Verify before marking any section complete.

## Section Status

- [x] Sections 1-12: release contract, release evidence slices, schemas,
  validators, and release-gate scaffolding.
- [x] Section 13: Safe View Lifetime integration.
- [x] Section 14: Feature registry promotion.
- [x] Section 15: Docs promotion.
- [x] Section 16: API docs and stdlib stability.
- [x] Section 17: CI integration.
- [x] Section 18: Release examples.
- [x] Section 19: Negative anti-fake tests.
- [x] Section 20: Final release audit artifact.
- [x] Section 21+: final command matrix, failure modes, completion checklist,
  and goal closeout from the plan.

## Current Section: Complete

Acceptance targets from the plan:

- All Definition of Done items have current evidence.
- Final release gate, experimental regression gate, safe-view lifetime gate,
  focused/broad tests, docs/manifest/API checks, release-state validator,
  hygiene checks, Graphify update, and final dumps passed.
- Dirty worktree status and the Section 22 broad-test repair are recorded
  without reverting unrelated work.

Immediate checklist:

- [x] Inspect current DoD items and derive a command/evidence matrix.
- [x] Run final gates into `/tmp/tetra-surface-release-v1-current`,
  `/tmp/tetra-surface-experimental-regression-current`, and
  `/tmp/tetra-safe-view-lifetime-surface-release-current`.
- [x] Run docs, manifest, API, release-state, hygiene, Graphify, and final dump commands.
- [x] Run broad tests and repair the safety/no-wrapper blocker with focused
  evidence before rerunning the broad matrix.
