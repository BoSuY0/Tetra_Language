# Surface Block Memory Budget Implementation Plan

**Goal:** Complete `SURFBLOCK-P18` by making Block-system reports carry conservative, bounded
memory/cache/performance evidence without claiming Electron benchmark superiority.
**Context:** P12-P17 prove Block rendering, platform evidence, examples, validators, and release
gates. P18 adds budget facts for the Block scene itself.
**Execution:** Use TDD: add failing validator/runtime tests first, implement schema and
deterministic budget generation, update docs/nonclaims, then verify with targeted tests, overclaim
search, Block gate smoke, `git diff --check`, and `graphify update .`.

## Task 1 - RED Validator Coverage

**Goal:** Missing or fake memory budget evidence must fail.

**Files:** `tools/validators/surface/report_test.go`, `tools/validators/surface/report.go`.

**Approach:** Add tests for missing `block_system.memory_budget`, unbounded caches, mismatched
framebuffer/cache totals, and broad Electron-style performance claims.

**Verification:** `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surfblock-p18-red GOTMPDIR=$(pwd)/.cache/go-tmp-surfblock-p18-red go test -buildvcs=false ./tools/validators/surface -run 'Memory|Budget|Cache|Performance|Block' -count=1`.

**Done when:** The tests fail before the schema exists or before validation enforces it.

## Task 2 - Runtime Budget Evidence

**Goal:** Generated Block reports include deterministic budget numbers tied to real report fields.

**Files:** `tools/cmd/surface-runtime-smoke/main.go`,
`tools/cmd/surface-runtime-smoke/main_test.go`.

**Approach:** Compute a `BlockMemoryBudgetReport` from frames, component count, paint/text/asset
command counts, glyph/asset cache usage, and deterministic stress loop counts. Keep RSS optional and
explicitly marked as not measured when unavailable.

**Verification:** `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surfblock-p18-green GOTMPDIR=$(pwd)/.cache/go-tmp-surfblock-p18-green go test -buildvcs=false ./tools/cmd/surface-runtime-smoke -run 'Memory|Budget|Stress|Block' -count=1`.

**Done when:** Headless, linux-x64, and wasm32-web Block scenarios all emit passing memory budget
evidence.

## Task 3 - Gate And Docs Nonclaims

**Goal:** Release evidence and docs describe bounded local budget evidence only.

**Files:** `scripts/release/surface/block-system-gate.sh`, `docs/spec/surface_v1.md`,
`docs/user/surface_guide.md`, `docs/release/surface_v1_release_contract.md`.

**Approach:** Keep the existing Block gate, regenerate reports with the new field, and document that
P18 is local budget evidence, not an official Electron comparison or broad performance claim.

**Verification:** run the P18 forbidden broad-performance-claim search across
`docs README.md compiler lib examples scripts tools .github`; it must return no
source matches.

**Done when:** Docs mention cache/frame budget scope and nonclaims without banned broad-performance
phrases.

## Task 4 - Evidence Closeout

**Goal:** Record P18 with same-commit artifacts.

**Files:** `GOAL.md`, `reports/surface-block/p18-budget/`.

**Approach:** Run validator/runtime tests, regenerate a Block gate report under
`reports/surface-block/p18-budget`, validate hashes, run `git diff --check`, run
`graphify update .`, and update `GOAL.md`.

**Verification:** The P18 GREEN command list passes in the current repo.

**Done when:** `SURFBLOCK-P18` has command and report evidence; P19 remains open.
