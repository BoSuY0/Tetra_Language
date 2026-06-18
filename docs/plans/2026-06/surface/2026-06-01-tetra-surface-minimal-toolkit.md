# Tetra Surface Minimal Toolkit Implementation Plan

**Goal:** Implement the experimental minimal Surface toolkit slice from
`/home/tetra/Downloads/tetra_surface_minimal_toolkit_plan.md`.

**Context:** Surface already has runtime, text/focus/input, component-tree, and
component-tree API evidence. This milestone adds reusable pure-Tetra widgets on
top of that API without promoting a production toolkit.

**Execution:** Follow TDD for the validator/source/smoke surfaces, then execute
in small batches with fresh verification after each batch.

## Task 1: RED Coverage For Toolkit Contract

- **Goal:** Add tests that fail until toolkit support exists.
- **Files:** `tools/validators/surface/report_test.go`,
  `tools/scriptstest/release_surface_smoke_test.go`,
  `tools/cmd/surface-runtime-smoke/main_test.go`.
- **Approach:** Require a `tetra.surface.toolkit.v1` block, new minimal-toolkit
  smoke modes, new scripts in the release gate, and source scans proving
  `examples/surface_toolkit_form.tetra` imports `lib.core.widgets` without
  local demo widget structs or structural tree writes.
- **Verification:** Targeted `go test` for the touched packages must fail for
  missing toolkit/schema/script/source evidence before implementation.
- **Done when:** Failures point at the intended missing toolkit artifacts.

## Task 2: Pure-Tetra Toolkit Module And Example

- **Goal:** Add reusable `Text`, `Button`, `TextBox`, `Row`, `Column`, and
  `Panel` helpers.
- **Files:** Add `lib/core/widgets.tetra`; add
  `examples/surface_toolkit_form.tetra`; minimally extend
  `lib/core/component.tetra` only where the existing tree helpers need
  additive 9-node support.
- **Approach:** Keep helper returns slot-safe. Use `inout` mutation and `Int`
  status/action values. Preserve existing component-tree behavior.
- **Verification:** Build/run through targeted smoke mode once runtime support
  exists; source-scan tests must pass.
- **Done when:** The example exits `1`, draws a visible form, routes TextBox,
  Submit, Reset, status, and resize through reusable widgets.

## Task 3: Runtime Smoke And Toolkit Evidence

- **Goal:** Emit strict reports for headless, linux real-window, and browser
  canvas minimal-toolkit modes.
- **Files:** `tools/cmd/surface-runtime-smoke/main.go` and tests; scripts under
  `scripts/release/surface`.
- **Approach:** Add modes `headless-minimal-toolkit`,
  `linux-x64-real-window-minimal-toolkit`, and
  `wasm32-web-browser-canvas-minimal-toolkit`. Reuse existing browser and
  Wayland evidence paths, adding a deterministic toolkit scenario.
- **Verification:** Each new script runs and validates a report under
  `/tmp/tetra-surface-minimal-toolkit-review`.
- **Done when:** Reports contain runtime, component-tree, component-tree-api,
  and toolkit evidence.

## Task 4: Validator Rules And Negative Fixtures

- **Goal:** Reject fake or overclaimed toolkit evidence.
- **Files:** `tools/validators/surface/report.go`,
  `tools/validators/surface/report_test.go`.
- **Approach:** Add toolkit report structs and validation for schema, source,
  module, widget set, reusable source helpers, no production claim, no DOM/user
  JS/platform widget claim, tree/API evidence, event routing, reset/status,
  resize/frame changes, and correct host evidence.
- **Verification:** Positive fixture passes and negative tests fail with
  targeted diagnostics.
- **Done when:** Validator enforces all minimal-toolkit DoD checks that can be
  proven from report JSON.

## Task 5: Gate, Docs, Registry, Manifest, Graphify

- **Goal:** Integrate the milestone into release docs and generated metadata.
- **Files:** `scripts/release/surface/gate.sh`,
  `scripts/release/surface/README.md`, `docs/spec/surface_v1.md`,
  `docs/user/surface_guide.md`, `docs/spec/current_supported_surface.md`,
  `docs/user/examples_index.md`, `compiler/features.go`,
  `docs/generated/manifest.json`, `graphify-out/*`.
- **Approach:** Add experimental `ui.surface-minimal-toolkit`, update existing
  Surface status without production overclaim, regenerate manifest, run
  Graphify update after code edits.
- **Verification:** `verify-docs`, `gen-manifest`, `validate-manifest`, and
  `graphify update .`.
- **Done when:** Docs and registry describe experimental toolkit evidence and
  the aggregate gate runs new scripts.

## Task 6: Full Completion Audit

- **Goal:** Prove every plan requirement against current evidence.
- **Verification:** Run:
  - `go test ./... ./compiler/... ./cli/... ./tools/... -count=1`
  - `go run ./tools/cmd/verify-docs`
  - `go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json`
  - `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
  - `git diff --check`
  - `graphify update .`
  - all three minimal-toolkit smoke scripts
  - individual `validate-surface-runtime` for all three toolkit reports
  - `bash scripts/release/surface/gate.sh --report-dir /tmp/tetra-surface-release-gate-review`
- **Done when:** Every command passes and final reports are preserved in
  `/tmp/tetra-surface-minimal-toolkit-review` and
  `/tmp/tetra-surface-release-gate-review`.
