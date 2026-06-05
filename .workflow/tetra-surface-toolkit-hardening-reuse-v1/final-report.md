# Tetra Surface Toolkit Hardening + Reuse v1 Final Report

## Goal

Implement the experimental Surface toolkit hardening and reuse milestone from
`/home/tetra/Downloads/tetra_surface_toolkit_hardening_reuse_v1_plan.md`.

## Summary

`Tetra Surface Toolkit Hardening + Reuse v1` is complete as an experimental
toolkit milestone: reusable pure-Tetra widget helpers are validated across
multiple Surface examples and three runtime targets, with strict toolkit,
component-tree, component-tree-api, input, frame, source-scan, and artifact
evidence.

Browser-canvas trace collection also now retries the transient `pending`
`--dump-dom` trace boundary and has a regression test for that retry path.

This is not a production Surface toolkit claim.

## Files Changed

- `lib/core/widgets.tetra`
- `lib/core/component.tetra`
- `examples/surface_toolkit_settings.tetra`
- `tools/cmd/surface-runtime-smoke/main.go`
- `tools/cmd/surface-runtime-smoke/main_test.go`
- `tools/validators/surface/report.go`
- `tools/validators/surface/report_test.go`
- `tools/scriptstest/release_surface_smoke_test.go`
- `scripts/tools/surface_browser_canvas_host.mjs`
- `scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh`
- `scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh`
- `scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh`
- `scripts/release/surface/gate.sh`
- `scripts/release/surface/README.md`
- `docs/spec/surface_v1.md`
- `docs/spec/current_supported_surface.md`
- `docs/user/surface_guide.md`
- `docs/user/examples_index.md`
- `compiler/features.go`
- `docs/generated/manifest.json`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`

## New Example

`examples/surface_toolkit_settings.tetra` is the second reusable toolkit app.
It uses `lib.core.widgets` and `lib.core.component` helpers for:

- Panel, Column, Text, TextBox, Row, Button construction.
- NameTextBox and EmailTextBox independent text buffers.
- SaveButton and ResetButton action routing.
- StatusText updates after Save and Reset.
- Resize relayout from 320x240 to 480x320.
- Component tree validation and dispatch path helpers.

The example does not define local Text/Button/TextBox/Row/Column/Panel widget
structs and does not write tree structural fields directly.

## New Scripts

- `scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh`
- `scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh`
- `scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh`

The aggregate `scripts/release/surface/gate.sh` runs all three scripts and
revalidates all three reuse reports.

## Report Schema And Validator Rules

The existing `tetra.surface.toolkit.v1` block now covers toolkit reuse with:

- `toolkit_level = toolkit-reuse-v1`
- `reuse_level = multi-form-widget-reuse-v1`
- `sources` covering `examples/surface_toolkit_form.tetra` and
  `examples/surface_toolkit_settings.tetra`
- `example_count = 2`
- `text_box_count = 2`
- `button_count = 2`
- `multi_textbox_evidence = true`
- `multi_form_evidence = true`

The validator rejects missing/wrong toolkit blocks, production claims,
manual bookkeeping, demo-local widget struct claims, single-example reuse
claims, missing second TextBox routing, unfocused TextBox mutation, missing
StatusText updates, missing resize relayout, unchanged frame checksums,
DOM/user-JS claims, Node-only browser claims, and missing artifact scans.

## Commands Run

Baseline before reuse work:

```sh
bash scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh --report-dir /tmp/tetra-surface-toolkit-baseline
bash scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh --report-dir /tmp/tetra-surface-toolkit-baseline
bash scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh --report-dir /tmp/tetra-surface-toolkit-baseline
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-baseline/surface-headless-minimal-toolkit.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-baseline/surface-linux-x64-real-window-minimal-toolkit.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-baseline/surface-wasm32-web-browser-canvas-minimal-toolkit.json
```

Focused and reuse evidence:

```sh
go test ./tools/validators/surface -run Toolkit -count=1
go test ./tools/cmd/surface-runtime-smoke -run Toolkit -count=1
go test ./tools/cmd/surface-runtime-smoke -run 'Toolkit|BrowserCanvasTraceRetries|CollectWASM32WebBrowserCanvasProcessEvidenceRecordsBrowserTrace' -count=1
go test ./tools/scriptstest -run Surface -count=1
bash scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh --report-dir /tmp/tetra-surface-toolkit-reuse-review
bash scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh --report-dir /tmp/tetra-surface-toolkit-reuse-review
bash scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh --report-dir /tmp/tetra-surface-toolkit-reuse-review
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-reuse-review/surface-headless-toolkit-reuse.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-reuse-review/surface-linux-x64-real-window-toolkit-reuse.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-toolkit-reuse-review/surface-wasm32-web-browser-canvas-toolkit-reuse.json
```

Release and safety gates:

```sh
bash scripts/release/surface/gate.sh --report-dir /tmp/tetra-surface-release-gate-current
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-release-gate-current/surface-headless-toolkit-reuse.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-release-gate-current/surface-linux-x64-real-window-toolkit-reuse.json
go run ./tools/cmd/validate-surface-runtime --report /tmp/tetra-surface-release-gate-current/surface-wasm32-web-browser-canvas-toolkit-reuse.json
bash scripts/release/safe-view-lifetime/gate.sh --report-dir /tmp/tetra-safe-view-lifetime-gate-current
```

Full verification:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
go test ./... ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test.sh
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs
git diff --check
graphify update .
```

All commands above passed in this session with persistent Go build caches under
`$HOME/.cache/tetra-language/...`.

## Preserved Reports

- `/tmp/tetra-surface-toolkit-reuse-review`
- `/tmp/tetra-surface-release-gate-current`
- `/tmp/tetra-safe-view-lifetime-gate-current`

## Known Unrelated Dirty Worktree

Before this goal batch, the repository already contained a large unrelated
dirty worktree spanning compiler, CLI, runtime, docs, Graphify, and release
artifacts. This goal did not revert or normalize those unrelated changes.
The relevant observed status included many modified compiler/CLI/runtime files,
untracked Surface foundation files from earlier milestones, and existing
`.workflow/*` reports. Goal-specific work stayed scoped to the files listed
above.

## Explicit Non-Goals

- No production Surface toolkit promotion.
- No Windows/macOS real-host toolkit proof.
- No GPU renderer.
- No platform accessibility host tree.
- No IME, clipboard, rich text, or Unicode grapheme editing.
- No virtual DOM or reactive framework.
- No final trait-object component ABI or witness-table component dispatch.
- No DOM UI or user JavaScript app logic.
- No legacy `.ui.html`, `.ui.web.mjs`, or `.ui.json` sidecar path.
