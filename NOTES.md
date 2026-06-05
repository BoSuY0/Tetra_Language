# Tetra Surface Release Promotion v1 Notes

## Canonical Objective

Implement the full Surface Release Promotion v1 plan:

- `/home/tetra/Downloads/tetra_surface_release_promotion_v1_full_plan.md`

The active goal is not the Ideal Master Plan. If sidecars drift to Ideal,
restore the Surface Release Promotion state before continuing.

## Current Implementation Area

Surface Release Promotion v1 is implemented and verified.

Likely areas to inspect before editing:

- `.workflow/tetra-surface-release-promotion-v1/final-report.md`
- `docs/release/surface_v1_release_audit.md`
- `docs/release/surface_v1_release_notes.md`
- `docs/release/surface_v1_release_contract.md`
- `docs/spec/surface_v1.md`
- `/tmp/tetra-surface-release-v1-current`
- `/tmp/tetra-surface-experimental-regression-current`
- `/tmp/tetra-safe-view-lifetime-surface-release-current`
- `scripts/release/surface/release-gate.sh`
- `scripts/release/surface/gate.sh`
- `scripts/release/safe-view-lifetime/gate.sh`
- `compiler/tests/safety/effects_test.go`
- `compiler/tests/safety/plan250_safety_runtime_test.go`
- `tools/scriptstest/no_wrapper_structure_test.go`
- `compiler/tests/README.md`

Completed broad-test blocker:

- The Section 22 broad-test blocker was repaired: safety fixtures now reach the
  intended secret-taint diagnostics without exported `secret.i32` ABI
  parameters, aggregate consent-token fixtures use explicit `repr(C)`, and
  `compatibility_stability_v1_test.go` plus `runtime_hardening_v1_test.go` are
  documented/allowed root-test exceptions.

Final evidence:

- Current report dirs:
  `/tmp/tetra-surface-release-v1-current`,
  `/tmp/tetra-surface-experimental-regression-current`, and
  `/tmp/tetra-safe-view-lifetime-surface-release-current`.
- Final report:
  `.workflow/tetra-surface-release-promotion-v1/final-report.md`.
- Final dumps:
  `dumps/tetra_language_dump_20260603_201342Z_part_001.md` and
  `dumps/tetra_language_dump_20260603_201342Z_part_002.md`.

## Section 18 Completed Summary

- `examples/surface_release_counter.tetra` is the release counter example.
- Linux and browser counter smoke scripts use that release source.
- Runtime-smoke remaps release counter component evidence to the
  `examples.surface_release_counter` module and uses a release-counter browser
  expected frame.
- Section 18 evidence anchors live under:
  `/tmp/tetra-surface-section18-release-counter-linux`,
  `/tmp/tetra-surface-section18-release-counter-browser`,
  `/tmp/tetra-surface-section18-experimental-gate`,
  `/tmp/tetra-surface-section18-release-gate`, and
  `/tmp/tetra-surface-section18-api-stability`.

## Section 19 Completed Summary

- Source anti-fake scans now cover release examples for stable core imports,
  no local demo widget structs, no manual structural `TreeNode` writes, no
  frontend framework markers, no DOM/user JS, and no `.ui.*` evidence.
- Runtime negative tests now include the exact Section 19 named rejection tests
  and a release-negative `legacy_sidecars.json` fixture.
- Docs validation now rejects Surface v1 fake-promotion docs claims and requires
  unsupported target evidence plus the release gate command.
- Verification anchors:
  `/tmp/tetra-surface-section19-api-stability`,
  `/tmp/tetra-surface-section19-experimental-gate`, and
  `/tmp/tetra-surface-section19-release-gate`.

## Section 20 Completed Summary

- Final report artifact:
  `.workflow/tetra-surface-release-promotion-v1/final-report.md`.
- It records Surface Release Promotion v1 as complete for
  `surface-v1-linux-web`, supported/unsupported scope, implemented areas,
  command list, report paths, artifact hash path, final dump paths, release
  summary evidence, and known limitations.
- Verification anchors: final-report marker scan, `validate-manifest`,
  `verify-docs`, `validate-surface-release-state`, `git diff --check`, and
  scoped whitespace scan passed.

## Cache Discipline

Use:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-release
```

Do not use `/tmp` as Go cache.
