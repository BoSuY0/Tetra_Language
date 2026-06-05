# Tetra Surface Accessibility Metadata Tree v1 Final Report

## Goal

Implement the experimental Surface accessibility metadata tree milestone from
`/home/tetra/Downloads/tetra_surface_accessibility_metadata_tree_v1_plan.md`.

## Summary

`Tetra Surface Accessibility Metadata Tree v1` is complete as an experimental
metadata-only accessibility milestone. The repository now has pure-Tetra
`lib.core.accessibility` helpers layered over `lib.core.component` and
`lib.core.widgets`, a dedicated
`examples/surface_accessibility_settings.tetra` settings form, strict
`tetra.surface.accessibility-tree.v1` report generation, validator rules,
negative tests, three release smoke scripts, aggregate Surface gate integration,
docs, feature registry, generated manifest, Graphify updates, and this final
workflow report.

The milestone proves roles, labels, values, states, bounds, parent-child
relationships, label relationships, focus order, reading order, actions,
status updates, snapshots, metadata checksum changes, bounds checksum changes,
frame checksum changes, and artifact scans across headless, linux-x64
real-window, and wasm32-web browser-canvas evidence.

This is not platform accessibility host integration, Linux AT-SPI, macOS AX,
Windows UI Automation, browser DOM/ARIA accessibility, screen-reader
validation, or production Surface accessibility support.

## Files Changed

- `lib/core/accessibility.tetra`
- `lib/core/widgets.tetra`
- `lib/core/component.tetra`
- `examples/surface_accessibility_settings.tetra`
- `tools/cmd/surface-runtime-smoke/main.go`
- `tools/validators/surface/report.go`
- `tools/validators/surface/report_test.go`
- `tools/scriptstest/release_surface_smoke_test.go`
- `scripts/tools/surface_browser_canvas_host.mjs`
- `scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh`
- `scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh`
- `scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh`
- `scripts/release/surface/gate.sh`
- `scripts/release/surface/README.md`
- `docs/spec/surface_v1.md`
- `docs/spec/current_supported_surface.md`
- `docs/user/surface_guide.md`
- `docs/user/examples_index.md`
- `compiler/features.go`
- `docs/generated/manifest.json`
- `compiler/internal/runtimeabi/small_heap.go`
- `compiler/internal/backend/x64core/emit.go`
- `compiler/internal/allocplan/plan.go`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`
- `README.md`

## New API And Example

`lib/core/accessibility.tetra` defines metadata roles, values, actions,
`NodeMetadata`, `Snapshot`, settings-count validation, and deterministic bounds
and metadata seed helpers. `lib/core/widgets.tetra` exposes accessibility-aware
TextBox, Button, and Status helpers so example apps do not write structural
metadata fields directly.

`examples/surface_accessibility_settings.tetra` uses `lib.core.widgets` and
`lib.core.accessibility` for this exact 12-node tree:

```text
AccessibilitySettingsApp -> Panel -> Column ->
TitleText, NameLabel, NameTextBox, EmailLabel, EmailTextBox,
ButtonRow -> SaveButton, ResetButton, StatusText
```

It proves NameLabel/EmailLabel label relationships, NameTextBox ->
EmailTextBox -> SaveButton -> ResetButton focus order, reading order,
edit/press/save/reset actions, StatusText updates, Reset clearing both
TextBoxes, resize relayout to 480x320, and redraw evidence.

## Report Schema And Validator Rules

Reports add `accessibility_tree.schema =
tetra.surface.accessibility-tree.v1` with:

- `accessibility_level = metadata-tree-v1`
- `source = examples/surface_accessibility_settings.tetra`
- `module = lib.core.accessibility`
- `widget_module = lib.core.widgets`
- `experimental:true`
- `production_claim:false`
- `platform_host_integration:false`
- `dom_aria_integration:false`
- `screen_reader_evidence:false`
- `derived_from_component_tree:true`
- `uses_component_tree_api:true`
- `uses_widget_toolkit:true`
- `manual_bookkeeping:false`
- `no_dom_ui:true`
- `no_user_js:true`
- `no_platform_widgets:true`
- `no_legacy_sidecars:true`

The validator rejects missing/wrong accessibility tree evidence, production or
platform host claims, DOM/ARIA claims, screen-reader claims, manual
bookkeeping, unknown roles, duplicate or unknown component IDs, bounds
mismatches, missing label relationships, focus order mismatches, reading order
mismatches, state mismatches, static-only snapshots, unchanged metadata/bounds
checksums, Node-only browser evidence, user JavaScript evidence, and legacy
sidecars.

During full verification, existing in-flight small-heap allocation tests in the
dirty worktree also required runtime ABI metadata and x64 helper wiring to be
made coherent. Those additions are included because otherwise the required
repo-wide verification commands could not pass.

## Smoke Scripts

- `scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh`
- `scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh`
- `scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh`

The aggregate `scripts/release/surface/gate.sh` runs and revalidates all three
accessibility reports together with the existing Surface matrix.

## Preserved Reports And Dump

- `/tmp/tetra-surface-accessibility-metadata-review`
- `/tmp/tetra-surface-release-gate-review`
- `/tmp/tetra-safe-view-lifetime-gate-review`
- `dumps/tetra_language_dump_20260601_162847Z.txt`

## Verification Commands

All final reruns below passed in this session:

```sh
go test ./tools/validators/surface ./tools/scriptstest ./tools/cmd/surface-runtime-smoke -count=1
bash -n scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh
bash -n scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh
bash -n scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh
bash -n scripts/release/surface/gate.sh
bash scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh --report-dir /tmp/tetra-surface-accessibility-metadata-review
bash scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh --report-dir /tmp/tetra-surface-accessibility-metadata-review
bash scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh --report-dir /tmp/tetra-surface-accessibility-metadata-review
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-accessibility-metadata-review/surface-headless-accessibility-metadata.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-accessibility-metadata-review/surface-linux-x64-real-window-accessibility-metadata.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-accessibility-metadata-review/surface-wasm32-web-browser-canvas-accessibility-metadata.json
bash scripts/release/surface/gate.sh --report-dir /tmp/tetra-surface-release-gate-review
bash scripts/release/safe-view-lifetime/gate.sh --report-dir /tmp/tetra-safe-view-lifetime-gate-review
go test ./compiler/... ./cli/... ./tools/... -count=1
go test ./... ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test.sh
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Persistent Go build caches used for evidence runs lived under
`$HOME/.cache/tetra-language/...` and were cleaned with `go clean -cache`.

## Known Dirty Worktree Note

The repository had a large unrelated dirty worktree before this goal batch,
including many compiler, CLI, backend, runtime, docs, release, Graphify, and
workflow changes. This work did not revert or normalize unrelated edits. Where
repo-wide verification exposed in-flight small-heap allocation edits, the
minimal fixes were made only to satisfy the required verification surface.

## Explicit Non-Goals

- No production Surface accessibility support.
- No platform accessibility host export.
- No Linux AT-SPI, macOS AX, or Windows UI Automation.
- No browser DOM/ARIA accessibility evidence.
- No screen-reader validation.
- No IME, clipboard, rich text, or Unicode grapheme editing.
- No GPU renderer.
- No virtual DOM or reactive framework.
- No final trait-object component ABI or witness-table component dispatch.
- No Windows/macOS Surface host evidence.
