# Tetra Surface Release Promotion v1 Attempts and Evidence

This file records evidence by section, not full command transcripts.

## Completed Evidence

- Sections 1-12: release contract, target evidence, schemas, validators,
  release gate, experimental gate, and release-gate scaffolding are represented
  by current `scripts/release/surface`, `tools/validators/surface`,
  `tools/cmd/validate-surface-runtime`, and release docs artifacts.
- Section 13: safe-view lifetime integration passed focused semantics checks and
  the safe-view lifetime gate with release evidence.
- Section 14: feature registry promotion passed feature registry checks,
  manifest/docs validators, release-state validator, release gate, hygiene, and
  Graphify update.
- Section 15: docs promotion added `docs/release/surface_v1_release_audit.md`,
  linked it from release notes and `ui.surface-core`, regenerated manifest, and
  passed validators, hygiene, and Graphify update.
- Section 16: API docs/stdlib stability added
  `scripts/release/surface/api-stability-gate.sh`; stable `lib.core` Surface
  module docs and no-experimental-import checks passed with docs/manifest
  validators, hygiene, and Graphify update.
- Section 17: CI integration added `.github/workflows/ci.yml`
  `surface-release-readiness-linux`; focused workflow tests, `actionlint`,
  shell syntax, `git diff --check`, and Graphify update passed. A broader
  `go test ./tools/scriptstest` run exposed unrelated dirty-workspace root
  failures and was recorded as caveat, not release-surface regression evidence.

## Completed Attempt: Section 18 Release Examples

Status: completed.

Evidence gathered:

- Added `examples/surface_release_counter.tetra` using stable
  `lib.core.surface`, `lib.core.draw`, `lib.core.component`,
  `lib.core.widgets`, `lib.core.style`, and `lib.core.accessibility`.
- Updated Linux/browser counter smoke scripts to use
  `examples/surface_release_counter.tetra`.
- Updated runtime-smoke to map release counter evidence to
  `examples.surface_release_counter` and to render strict browser expected
  frames for the release counter source.
- Added source/compiler/runtime tests for the release counter example.
- Focused tests passed for `tools/scriptstest`, `tools/cmd/surface-runtime-smoke`,
  `compiler/tests/semantics`, `tools/cmd/validate-surface-runtime`, and
  `tools/validators/surface` slices.
- Headless, Linux real-window, browser canvas, API stability, experimental, and
  final release gates passed with Section 18 artifact dirs under `/tmp`.
- Docs/manifest validators and `git diff --check` passed.
- Graphify update passed and rebuilt `20320 nodes`, `64403 edges`, and
  `1117 communities`.

## Current Attempt: Section 19 Negative Anti-Fake Tests

Status: completed.

Evidence gathered:

- Graphify MCP identified `tools/scriptstest/release_surface_smoke_test.go`,
  `tools/validators/surface/release_negative_test.go`,
  `tools/validators/surface/report.go`, and `tools/cmd/verify-docs/main.go`
  as the relevant source/runtime/docs anti-fake areas.
- Sidecar drift to the Ideal Master Plan was detected and repaired before
  Section 19 edits continued.
- Added `TestReleaseSurfaceExamplesRejectFakePromotionSources` to reject local
  demo widget structs, manual `TreeNode` structural writes, frontend framework
  markers, DOM/user JS, and legacy `.ui.*` markers in release examples.
- Updated `examples/surface_release_text_input.tetra` to use
  `lib.core.widgets` and `lib.core.style` instead of local demo widget structs;
  updated `examples/surface_release_accessibility.tetra` to use
  `lib.core.style`.
- Added named runtime negative tests for all Section 19 fake promotion paths
  and added `tools/validators/surface/testdata/release_negative/legacy_sidecars.json`.
- Added `verifySurfaceReleaseDocs` and tests to reject Surface docs fake claims
  for macOS/Windows current support, metadata-only production accessibility,
  DOM UI, user JS allowance, missing unsupported targets, and missing release
  gate command.
- RED evidence:
  source scan failed on missing style import, runtime negative failed on missing
  `legacy_sidecars.json`, and docs tests failed on undefined
  `verifySurfaceReleaseDocs`.
- GREEN evidence passed: focused source/runtime/docs/semantics tests,
  `go test ./tools/cmd/verify-docs -count=1`, `validate-manifest`,
  `verify-docs`, API stability gate, experimental gate, final release gate,
  release-state validator, full Surface validator package tests, validate
  Surface runtime/release-state package tests, `git diff --check`, scoped
  whitespace scan, and Graphify update.
- Section 19 artifact anchors:
  `/tmp/tetra-surface-section19-api-stability`,
  `/tmp/tetra-surface-section19-experimental-gate`, and
  `/tmp/tetra-surface-section19-release-gate`.
- Graphify evidence:
  `graphify update .` passed and rebuilt `20482 nodes`, `64808 edges`, and
  `1130 communities`.

Next action:

- Create `.workflow/tetra-surface-release-promotion-v1/final-report.md` for
  Section 20 using Section 19 release-gate evidence and known limitation scope.

## Completed Attempt: Section 20 Final Release Audit Artifact

Status: completed.

Evidence gathered:

- Created `.workflow/tetra-surface-release-promotion-v1/final-report.md`.
- The report contains required goal/status/scope, supported targets,
  unsupported targets, implemented areas, command list, report paths, artifact
  hash path, final dump paths, release summary evidence, and known limitations.
- Verification passed:
  marker scan over final report, `validate-manifest`, `verify-docs`,
  `validate-surface-release-state` against
  `/tmp/tetra-surface-section19-release-gate`, `git diff --check`, and scoped
  whitespace scan.

Next action:

- Run the Sections 21-22 final Definition of Done and command matrix, preserving
  exact evidence and recording any unrelated broad-test failures separately.

## Current Attempt: Sections 21-22 Final Verification

Status: completed.

Evidence gathered:

- Current release gate, experimental regression gate, safe-view lifetime gate,
  docs generation, manifest/docs/API/release-state validators, and release
  summary inspection passed against current `/tmp/tetra-surface-*` report dirs.
- Broad `go test ./compiler/... ./cli/... ./tools/... -count=1` failed on
  `compiler/tests/safety` and `tools/scriptstest`:
  exported-signature ABI diagnostics now mask several intended secret-taint
  diagnostics, and two package-private compiler root tests are not yet
  documented/allowed by the no-wrapper structure test.
- Repaired the broad-test blocker by updating safety fixtures to create
  `secret.i32` inside ABI-safe exported functions, adding explicit `repr(C)` to
  the aggregate consent-token fixture, and documenting/allowing
  `compatibility_stability_v1_test.go` plus `runtime_hardening_v1_test.go`.
- Focused repair evidence passed:
  `go test ./compiler/tests/safety -run 'TestPrivacySecretTaintBeyondFunctionSignatures|TestPlan250RuntimeRejectsAggregateConsentTokensInExportedSignatures' -count=1`
  and
  `go test ./tools/scriptstest -run 'TestNoWrapperCompilerRootTestsAreDocumentedExceptions' -count=1`.
- Final broad evidence passed:
  `go test ./compiler/... ./cli/... ./tools/... -count=1`,
  `go test ./... ./compiler/... ./cli/... ./tools/... -count=1`, and
  `bash scripts/ci/test.sh`.
- Final docs/release evidence passed: regenerated manifest, validated manifest,
  verified docs, generated API docs into
  `/tmp/tetra-surface-release-v1-current/artifacts/tetra-docs.md`, validated API
  docs, refreshed `/tmp/tetra-surface-release-v1-current/artifact-hashes.json`,
  validated artifact hashes, and validated release state as `current` for
  `surface-v1-linux-web`.
- Final hygiene evidence passed: `git diff --check`, scoped whitespace scan,
  `graphify update .` (`20489 nodes`, `64821 edges`, `1128 communities`), and
  `go run ./create_dumps.go`.
- Final dump artifacts:
  `dumps/tetra_language_dump_20260603_201342Z_part_001.md` and
  `dumps/tetra_language_dump_20260603_201342Z_part_002.md`.

Final state:

- Surface v1 is current/release-ready for linux-x64 real-window and wasm32-web
  browser-canvas Surface scope, with headless as release evidence target.
