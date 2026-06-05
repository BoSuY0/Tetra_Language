# Tetra Surface Release Promotion v1 Control

## State

- Canonical plan: `/home/tetra/Downloads/tetra_surface_release_promotion_v1_full_plan.md`
- Active section: complete.
- Completed through: Sections 21-22 Definition of Done and Final Verification.
- Current blocker: none.
- Go cache: `GOCACHE=$(pwd)/.cache/go-build-surface-release`

## Continuation Checklist

1. Re-read `AGENTS.md`, `GOAL.md`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and
   `CONTROL.md`.
2. Confirm the active objective is Surface Release Promotion v1, not the Ideal
   Master Plan.
3. Use Graphify MCP first for architecture/code navigation.
4. Inspect concrete files before editing.
5. For bugs/failing gates, use systematic debugging before fixes.
6. For implementation, prefer TDD when practical.
7. Keep the current section narrow.
8. Run focused tests before broad gates.
9. Run docs/manifest validators when docs or registry files change.
10. Run hygiene checks and `git diff --check` before claiming completion.
11. Run `graphify update .` after modifying code files.

## Current Section Gate

Sections 21-22 are complete:

- every Definition of Done item has current evidence;
- final release, experimental regression, and safe-view lifetime gates passed;
- docs/manifest/API/release-state validators passed;
- required broad tests and CI script passed;
- hygiene, Graphify, and final dump commands passed;
- final report documents dirty worktree status and current evidence anchors.
