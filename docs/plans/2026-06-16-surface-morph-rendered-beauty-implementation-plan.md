# Surface Morph Rendered Beauty Implementation Plan

Date: 2026-06-16
Status: `MRB-00` through `MRB-13` complete with evidence, with post-MRB-13 same-commit identity, wasm browser-canvas target, linux real-window target, renderer-owned stable proof guard, and all-supported-target renderer-owned stable proof follow-ups complete; final verdict `PARTIAL` because clean checkout audit, stable promotion, product claim, and final signoff still have remaining blockers.

## Goal

Make Morph the actual Tetra Surface beauty layer end to end:

```text
Morph Capsule -> resolved visual scene -> Block scene -> render commands -> real pixels -> pixel golden evidence -> product claim
```

The goal is not to create a second design system beside Morph. Morph remains the place where visual intent, materials, recipes, state lenses, and motion live. Block remains the only core UI primitive. Renderer, evidence, examples, and release gates must prove that Morph-authored UI becomes polished pixels across supported Surface targets.

## Current Truth

The current Surface stack already has the right boundary on paper:

- `docs/spec/surface_v1.md` defines Surface v1 as a Block-first UI surface with Linux real-window, app-shell, and wasm browser-canvas scope.
- `docs/spec/surface_block_contract.md` keeps Block as the only core primitive and treats compatibility widgets as recipes or helper APIs.
- `docs/spec/surface_morph.md` correctly says Morph is an experimental evidence layer whose tokens, materials, affordances, state lenses, motion, and recipes expand into `lib.core.block` Blocks.
- `examples/projects/tetra_control_center/docs/surface-flagship-contract.md` already frames the flagship UI as Surface-owned, with Morph and Block as the authoring/rendering boundary.

But the implementation is not yet a rendered beauty system:

- `lib/core/morph.tetra` defines recipes and paint/material intent, but there is no complete evidence chain from Morph recipe to real rendered pixels.
- `lib/core/draw.tetra` still uses low-fidelity placeholders for text and approximate primitives.
- `lib/core/block.parts/text_state.tetra` has richer specs, but compact `BlockProps` evidence loses too much visual detail for a beauty renderer if used alone.
- `examples/surface_migration_tetra_control_center.tetra` draws the flagship-like UI manually through rect/text calls instead of proving a Morph-authored scene.
- `tools/cmd/surface-runtime-smoke/linux_probes.go` can use a precomputed Block-system frame for the real-window probe, which is useful infrastructure evidence but not product visual evidence.
- `tools/cmd/surface-visual-diff/main.go` can synthesize a self-equal checksum and pass with `DiffPixels: 0`, so it is not yet a true pixel golden gate.
- `scripts/release/surface/surface-product-slice-gate.sh` still records `product_claim: false` and `final_signoff: false`, which is correct until this plan lands.

## Design Decision

Morph is the beauty layer.

That means:

- Morph owns visual language: tokens, materials, recipes, density, states, motion, and scene-level composition.
- Block owns portable primitive UI representation.
- Renderer owns deterministic conversion from Block scene plus visual specs into pixels.
- Validators own proof that the pipeline is real and source-linked.
- Release gates own claim discipline.

Do not add `Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, `Modal`, or similar as new core primitives. They may exist only as Morph recipes, helper APIs, examples, or compatibility facades that expand to Block.

Do not promote Morph to stable because recipes exist. Promote it only when Morph-authored UI renders through the real pipeline and passes visual evidence gates.

## Non-Goals

- No Electron, React, DOM, CSS, or browser sidecar dependency.
- No claim of GPU acceleration unless a real GPU path is implemented and validated.
- No macOS or Windows product claim while Surface v1 marks those targets unsupported.
- No full rich text, bidi, shaping, screen reader parity, signing, updater, or network app claim unless separate gates prove them.
- No replacement of Block as the core primitive.
- No second design system that duplicates Morph.

## Execution Order

### Task 0 - Freeze Baseline Truth

Goal:
Create a factual baseline before implementation so later work cannot drift into local-only success.

Files and surfaces:

- `AGENTS.md`
- `docs/spec/surface_v1.md`
- `docs/spec/surface_morph.md`
- `docs/spec/surface_block_contract.md`
- `docs/spec/current_supported_surface.md`
- `docs/plans/2026-06-13-surface-electron-competitor-product-slice.md`
- `docs/plan/2026-06-13-surface-electron-competitor-platform-plan.md`
- `docs/plans/2026-06-10-gpt-55-pro-tetra-surface-beauty-analysis-prompt.md`
- `lib/core/morph.tetra`
- `lib/core/block.tetra`
- `lib/core/block.parts/text_state.tetra`
- `lib/core/draw.tetra`
- `tools/cmd/surface-runtime-smoke/`
- `tools/cmd/surface-visual-diff/`
- `tools/validators/surface/`
- `scripts/release/surface/surface-product-slice-gate.sh`
- `examples/surface_migration_tetra_control_center.tetra`
- `examples/projects/tetra_control_center/docs/surface-flagship-contract.md`

Commands:

```bash
git status --short
git rev-parse HEAD
rg -n "product_claim|final_signoff|visual-regression|renderBlockSystemFrameSizedRGBA|draw\\.text|recipe_expands_to_block" docs lib tools scripts examples
```

Done when:

- A baseline audit names the exact current gaps.
- Existing nonclaims remain intact.
- No product beauty claim is made from current placeholder evidence.

### Task 1 - Define the Morph Rendered Beauty Contract

Goal:
Specify what must be proven before Morph can be called the rendered beauty layer.

Files:

- Add `docs/spec/surface_morph_rendered_beauty.md`
- Add `docs/spec/surface_morph_rendered_beauty_contract.json`
- Update `docs/spec/surface_morph.md`
- Update `docs/spec/surface_v1.md`
- Add or extend validator fixtures under `tools/validators/surface/testdata/`
- Add `tools/cmd/validate-surface-morph-rendered-beauty/`

Approach:

- Define report kind `tetra.surface.morph-rendered-beauty.v1`.
- Require source-linked evidence for:
  - Morph source or capsule hash;
  - token graph hash;
  - recipe expansion list;
  - resolved Morph visual scene;
  - Block scene snapshot;
  - render command stream;
  - actual RGBA or PNG frame artifact;
  - pixel golden comparison;
  - supported target;
  - negative guards.
- Reject metadata-only checksums.
- Reject self-golden generation in release mode.
- Reject precomputed fixture frames as product evidence.
- Reject hidden DOM, CSS, React, Electron, native widgets, or extra core UI primitives.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-morph-beauty \
  go test -buildvcs=false ./tools/cmd/validate-surface-morph-rendered-beauty ./tools/validators/surface \
  -run 'MorphRenderedBeauty|Visual' -count=1

GOCACHE=$(pwd)/.cache/go-build-surface-morph-beauty go clean -cache
```

Done when:

- Positive and negative fixtures prove the contract.
- Existing specs say Morph remains experimental until this report passes.

### Task 2 - Preserve a Real Block Scene Snapshot

Goal:
Stop relying on compact `BlockProps` alone for beauty evidence. Keep Block as the ABI primitive, but preserve enough resolved visual data for rendering and validation.

Files:

- `lib/core/block.tetra`
- `lib/core/block.parts/text_state.tetra`
- `lib/core/morph.tetra`
- Possible new module: `lib/core/morph_scene.tetra`
- `tools/cmd/surface-runtime-smoke/`
- `tools/validators/surface/`

Approach:

- Investigate whether the Tetra-side APIs can expose full layout, paint, text, image, input, state, motion, and accessibility specs without breaking current Block ABI constraints.
- Add a scene snapshot/report model that preserves full visual specs.
- Keep compact `BlockProps` for ABI/evidence compatibility.
- Make renderer-facing evidence use the full scene snapshot, not only compressed slot props.
- Ensure recipe helper APIs still expand to Block and do not become hidden runtime widgets.

Verification:

```bash
./tetra check examples/surface_morph_studio_shell.tetra
GOCACHE=$(pwd)/.cache/go-build-surface-scene \
  go test -buildvcs=false ./tools/validators/surface -run 'Block|Morph|Scene' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-scene go clean -cache
```

Done when:

- Morph recipe expansion can emit a renderable scene snapshot.
- Validators can inspect rich visual specs.
- Block remains the only core primitive.

### Task 3 - Implement Render Command Stream v1

Goal:
Introduce a deterministic render command stream produced from the resolved Block scene.

Files:

- `tools/cmd/surface-runtime-smoke/render_rgba.go`
- Possible new package: `tools/internal/surfacerender/`
- `tools/cmd/surface-runtime-smoke/reports.go`
- `tools/validators/surface/block_paint_validation.go`
- `tools/validators/surface/report_types.go`

Approach:

- Start in Go tooling for evidence stability, then wire the runtime path deeper after proof.
- Produce commands for:
  - fills;
  - gradients;
  - image fills;
  - borders;
  - radius clips;
  - shadows;
  - overlays;
  - focus outlines;
  - text runs;
  - icons;
  - clips and layers.
- Include source node IDs and Morph recipe IDs in each relevant command.
- Ensure handcrafted fixture frames cannot satisfy product visual evidence.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-render \
  go test -buildvcs=false ./tools/internal/surfacerender ./tools/cmd/surface-runtime-smoke ./tools/validators/surface \
  -run 'RenderCommand|BlockPaint|Morph' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-render go clean -cache
```

Done when:

- A Morph-authored fixture produces a command stream.
- The command stream produces a deterministic frame.
- Validators reject command streams that are not source-linked.

### Task 4 - Replace Placeholder Text/Icon Evidence for Beauty Path

Goal:
Beauty evidence must show real text/icon rasterization, not only rectangle markers.

Files:

- `lib/core/draw.tetra`
- `lib/core/block.parts/text_state.tetra`
- `tools/cmd/surface-runtime-smoke/render_rgba.go`
- `tools/internal/surfacerender/`
- `tools/validators/surface/block_text_validation.go`
- `tools/validators/surface/block_asset_validation.go`

Approach:

- Add a deterministic software baseline for text and icons.
- Use a bounded built-in atlas or deterministic font fallback first.
- Keep claims narrow: readable deterministic text evidence, not full typography parity.
- Require glyph evidence that is distinguishable from current text marker rectangles.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-text \
  go test -buildvcs=false ./tools/internal/surfacerender ./tools/validators/surface \
  -run 'Text|Glyph|Icon|Asset' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-text go clean -cache
```

Done when:

- Beauty reports include non-marker text/icon evidence.
- Validators reject marker-only text frames for Morph beauty claims.

### Task 5 - Build a True Pixel Golden Gate

Goal:
Replace self-equal visual evidence with a real pixel golden comparison.

Files:

- `tools/cmd/surface-visual-diff/main.go`
- `tools/validators/surface/visual.go`
- `tools/validators/surface/testdata/`
- `scripts/release/surface/visual-gate.sh`
- Possible new directory: `reports/surface/goldens/`

Approach:

- Read actual RGBA or PNG artifacts produced by runtime smoke.
- Compare against checked-in or release-managed golden artifacts.
- Allow golden updates only through explicit write-golden mode.
- Disallow write-golden mode inside release/product gates.
- Add negative cases:
  - `self_golden_rejected`;
  - `metadata_checksum_rejected`;
  - `fixture_frame_only_rejected`;
  - `missing_png_or_rgba_artifact_rejected`.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-visual \
  go test -buildvcs=false ./tools/cmd/surface-visual-diff ./tools/validators/surface \
  -run 'Visual|Golden|Checksum' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-visual go clean -cache
```

Done when:

- Current-frame checksum and golden checksum come from different artifacts.
- Product gate cannot pass by generating a checksum from metadata.
- Visual diff includes real artifact paths and pixel counts.

### Task 6 - Remove Precomputed Frames from Product Visual Evidence

Goal:
Keep real-window infrastructure probes, but stop counting precomputed frames as product visual proof.

Files:

- `tools/cmd/surface-runtime-smoke/linux_probes.go`
- `tools/cmd/surface-runtime-smoke/process.go`
- `tools/cmd/surface-runtime-smoke/headless_trace.go`
- `tools/cmd/surface-runtime-smoke/wasm_browser.go`
- `tools/cmd/surface-runtime-smoke/reports.go`
- `tools/validators/surface/runtime_validation.go`
- `tools/validators/surface/block_system_validation.go`

Approach:

- Present app-produced frames for product visual evidence.
- If a host probe still needs a synthetic frame, label it as host-probe-only.
- Add validator rules so `renderBlockSystemFrameSizedRGBA` output cannot satisfy Morph rendered beauty or product slice visual evidence.
- Tie accepted frames to app source, Morph recipe hash, Block scene hash, and render command hash.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-runtime \
  go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface \
  -run 'Runtime|BlockSystem|Precomputed|ProductEvidence' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-runtime go clean -cache
```

Done when:

- Precomputed frames are allowed only as explicit infrastructure probe evidence.
- Product visual evidence must be app-produced and source-linked.

### Task 7 - Add Morph Rendered Beauty Reports

Goal:
Create a first-class report that proves Morph becomes pixels.

Files:

- `tools/validators/surface/report_types.go`
- Possible new file: `tools/validators/surface/morph_rendered_beauty_validation.go`
- `tools/cmd/validate-surface-morph-rendered-beauty/`
- `tools/cmd/surface-runtime-smoke/reports.go`
- `scripts/release/surface/`

Approach:

- Emit `tetra.surface.morph-rendered-beauty.v1` reports from runtime smoke or a dedicated collector.
- Include:
  - Morph capsule/source identity;
  - token coverage;
  - recipe coverage;
  - scene snapshot hash;
  - render command stream hash;
  - pixel artifact hash;
  - golden comparison result;
  - target and scenario names;
  - negative guard status.
- Make the report consumable by release gates and claim scanner.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-morph-report \
  go test -buildvcs=false ./tools/cmd/validate-surface-morph-rendered-beauty ./tools/validators/surface \
  -run 'MorphRenderedBeauty|Report' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-morph-report go clean -cache
```

Done when:

- A passing report proves the full Morph-to-pixels chain.
- A failing report clearly names which link is missing.

### Task 8 - Migrate the Flagship Surface to Morph-Authored Rendering

Goal:
Make the Tetra Studio Shell flagship prove Morph beauty through real rendered output.

Files:

- `examples/surface_migration_tetra_control_center.tetra`
- Possible new source: `examples/surface_morph_rendered_studio_shell.tetra`
- `examples/projects/tetra_control_center/docs/surface-flagship-contract.md`
- `tools/cmd/surface-runtime-smoke/scenarios_*.go`
- `scripts/release/surface/surface-product-slice-gate.sh`

Approach:

- Decide whether to mutate the existing migration source or add a new Morph-rendered flagship source. Prefer adding a new source first if it avoids breaking existing migration evidence.
- Replace manual visual composition with Morph recipes as the primary authoring layer.
- Cover flagship screens from the contract:
  - home or dashboard shell;
  - project or package view;
  - run or diagnostics view;
  - settings or preferences view;
  - command palette or action surface.
- Keep generated UI Block-based.
- Avoid adding app-specific widgets as core primitives.

Verification:

```bash
./tetra check examples/surface_morph_rendered_studio_shell.tetra

GOCACHE=$(pwd)/.cache/go-build-surface-flagship \
  go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface \
  -run 'Flagship|Morph|Runtime|Visual' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-flagship go clean -cache
```

Done when:

- The flagship source is Morph-authored.
- The same source produces scene, commands, pixels, and golden evidence.
- The flagship contract no longer rests on manual placeholder drawing.

MRB-08 result, 2026-06-16:

- Added `examples/surface_morph_rendered_studio_shell.tetra` as the clean
  Morph-authored flagship evidence source and left the older manual migration
  source intact as historical/manual evidence.
- Runtime/visual/MRB artifacts live under `reports/surface/`:
  `mrb08-flagship-runtime.json`, `mrb08-flagship-visual.json`, and
  `mrb08-flagship-morph-rendered-beauty.json`.
- MRB-12 update: `go run -buildvcs=false ./cli/cmd/tetra check
  examples/surface_morph_rendered_studio_shell.tetra` now passes after the
  flagship source stopped importing `lib.core.draw` and calls the Morph-owned
  `morph.render_studio_shell_frame` helper before presenting real
  `surface.Frame` values.

### Task 9 - Update Developer Loop and Inspector

Goal:
Make the development experience show Morph beauty evidence, not only low-level Block traces.

Files:

- `cli/cmd/tetra/surface_dev.go`
- `tools/cmd/surface-inspector/main.go`
- `tools/validators/surface/dev_workflow_validation.go`
- `tools/validators/surface/inspector_validation.go`
- `docs/surface/` or existing user docs

Approach:

- `tetra surface dev` should regenerate visual artifacts when Morph tokens, recipes, scenes, or source files change.
- Surface inspector should show:
  - Morph tokens;
  - recipe expansions;
  - Block scene nodes;
  - render commands;
  - pixel frame artifacts;
  - golden diff result.
- Keep inspector evidence tied to source and report hashes.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-dev \
  go test -buildvcs=false ./cli/cmd/tetra ./tools/cmd/surface-inspector ./tools/validators/surface \
  -run 'SurfaceDev|Inspector|Morph' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-dev go clean -cache
```

Done when:

- A developer can edit Morph visual intent and see real rendered diff artifacts.
- Inspector can explain the full Morph-to-pixels path.

MRB-09 result, 2026-06-16:

- Added shared `MorphToPixelsChainReport` summary evidence under
  `tools/validators/surface`, sourced from validated
  `tetra.surface.morph-rendered-beauty.v1` reports.
- `tetra surface dev --morph-rendered-beauty-report <path>` now records
  source-linked `morph_to_pixels` evidence for token graph, recipe expansion,
  Block scene, render command stream, frame artifact, golden artifact, and diff
  metrics while preserving the fast-rebuild/no-hot-reload contract.
- `tools/cmd/surface-inspector` now accepts
  `--runtime-report morph-rendered-beauty:<path>` and emits inspector sections
  for recipe expansions, Block scene nodes, render commands, frame artifacts,
  golden diff, and the same source-linked hash chain.
- Smoke evidence lives under `reports/surface/mrb09-dev-workflow-smoke/` and
  `reports/surface/mrb09-inspector-smoke/`; both generated and validated real
  Morph flagship runtime, visual, MRB, frame, golden, and diff artifacts.
- Verification passed:
  `GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-surface-dev" go test -buildvcs=false ./cli/cmd/tetra ./tools/cmd/surface-inspector ./tools/validators/surface -run 'SurfaceDev|Inspector|Morph' -count=1`,
  `go test -buildvcs=false ./tools/cmd/validate-surface-dev-workflow ./tools/cmd/validate-surface-inspector -run 'SurfaceDev|Inspector|Morph|Validate' -count=1`,
  `bash scripts/release/surface/surface-dev-workflow-smoke.sh --report-dir reports/surface/mrb09-dev-workflow-smoke`,
  `bash scripts/release/surface/surface-inspector-smoke.sh --report-dir reports/surface/mrb09-inspector-smoke`,
  focused script wiring tests, `git diff --check`, and `graphify update .`.

### Task 10 - Update Templates and Reference Apps

Goal:
New Surface apps should start on the Morph-rendered path.

Files:

- `cli/cmd/tetra/new_surface_app.go`
- `scripts/release/surface/surface-template-smoke.sh`
- `scripts/release/surface/surface-reference-apps-smoke.sh`
- `tools/validators/surface/template_validation.go`
- `tools/validators/surface/reference_apps_validation.go`
- Existing reference apps under `examples/` or `examples/projects/`

Approach:

- Generated apps should use Morph recipes for visible UI.
- Reference apps should cover:
  - command palette;
  - dashboard;
  - settings;
  - editor shell;
  - glass or elevated panel;
  - focus and disabled states.
- Every reference app should produce a Morph rendered beauty report or be clearly marked as non-product/infrastructure only.

Verification:

```bash
bash scripts/release/surface/surface-template-smoke.sh
bash scripts/release/surface/surface-reference-apps-smoke.sh
```

Done when:

- New app templates no longer demonstrate manual placeholder drawing as the main path.
- Reference apps provide reusable beauty evidence.

Completion evidence, 2026-06-16:

- `bash scripts/release/surface/surface-template-smoke.sh --report-dir reports/surface/mrb10-template-smoke`
  passed and produced template `morph_to_pixels` evidence sourced from
  `reports/surface/mrb10-template-smoke/templates/studio-shell/src/main.tetra`.
- `bash scripts/release/surface/surface-reference-apps-smoke.sh --report-dir reports/surface/mrb10-reference-apps-smoke`
  passed and produced nine product reference-app Morph-to-pixels chains; the
  migration compatibility app is explicitly `infrastructure_only`.
- Focused and broader Go validation passed for template/reference validators,
  MRB report chain output, runtime source retargeting, and visual-diff related
  packages; `graphify update .` passed after implementation changes.

### Task 11 - Harden Claims and Documentation

Goal:
Prevent docs or release notes from claiming Morph beauty before evidence exists.

Files:

- `tools/validators/surface/claims.go`
- `tools/cmd/validate-surface-claims/`
- `docs/spec/surface_morph.md`
- `docs/spec/surface_v1.md`
- `docs/surface/`
- `docs/release/`
- `scripts/release/surface/surface-docs-claims-gate.sh`

Approach:

- Add claim rules for phrases such as:
  - Morph production beauty;
  - Electron-quality UI;
  - React-quality UI;
  - production-ready Morph;
  - pixel-perfect Surface.
- Require a same-commit `tetra.surface.morph-rendered-beauty.v1` report for any such claim.
- Keep language honest when evidence is infrastructure-only.

Verification:

```bash
GOCACHE=$(pwd)/.cache/go-build-surface-claims \
  go test -buildvcs=false ./tools/cmd/validate-surface-claims ./tools/validators/surface \
  -run 'Claim|Morph|Beauty' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-claims go clean -cache

bash scripts/release/surface/surface-docs-claims-gate.sh
```

Done when:

- Claim scanner blocks unsupported beauty/product claims.
- Docs explain Morph as the intended beauty layer without overstating support.

Completion evidence, 2026-06-16:

- `validate-surface-claims` now detects Morph production beauty,
  Electron-quality UI, React-quality UI, production-ready Morph, and
  pixel-perfect Surface wording. Beauty/quality claims require a valid
  same-commit `tetra.surface.morph-rendered-beauty.v1` report; production-ready
  Morph wording additionally requires `product_claim` and `final_signoff`.
- Docs wording was narrowed in `docs/spec/surface_morph_rendered_beauty.md`,
  `docs/spec/surface_morph.md`, `docs/user/surface_electron_comparison.md`,
  and `docs/user/surface_cookbook.md`.
- `scripts/release/surface/surface-docs-claims-gate.sh` was added as the
  standalone docs/claims gate.
- Evidence passed:
  `go test -buildvcs=false ./tools/cmd/validate-surface-claims ./tools/validators/surface -run 'Claim|Morph|Beauty' -count=1`,
  `go test -buildvcs=false ./tools/cmd/validate-surface-claims ./tools/validators/surface -count=1`,
  `bash scripts/release/surface/surface-docs-claims-gate.sh`,
  `bash scripts/release/surface/surface-docs-claims-gate.sh --report-dir reports/surface/mrb10-template-smoke`,
  `go run -buildvcs=false ./tools/cmd/validate-surface-claims --root "$PWD" --report-dir reports/surface/mrb10-template-smoke --report-dir reports/surface/mrb10-reference-apps-smoke`,
  `go test -buildvcs=false ./tools/scriptstest -run 'ReleaseSurfaceDocsClaimsGate|ReleaseSurfaceFinalReleaseGateRunsCurrentSurfaceV1Evidence' -count=1`,
  `git diff --check`, and `graphify update .`.

### Task 12 - Add Integrated Morph Rendered Beauty Gate

Goal:
Create one release gate that proves the whole chain before product-slice signoff.

Files:

- Add `scripts/release/surface/morph-rendered-beauty-gate.sh`
- Update `scripts/release/surface/visual-gate.sh`
- Update `scripts/release/surface/surface-product-slice-gate.sh`
- Possibly update `scripts/release/surface/product-gate.sh`
- `reports/stabilization/`

Approach:

Gate order:

1. Validate Morph and Block contracts.
2. Build Morph rendered flagship source.
3. Produce resolved Morph scene report.
4. Produce Block scene snapshot.
5. Produce render command stream.
6. Produce actual target frame artifacts.
7. Compare against pixel goldens.
8. Validate inspector/dev/template/reference coverage.
9. Validate docs and claims.
10. Write artifact hashes and final summary.

Use persistent Go build cache:

```bash
export GOCACHE="$(pwd)/.cache/go-build-surface-morph-beauty-gate"
```

Clean it after evidence:

```bash
GOCACHE="$(pwd)/.cache/go-build-surface-morph-beauty-gate" go clean -cache
```

Verification:

```bash
bash scripts/release/surface/morph-rendered-beauty-gate.sh
bash scripts/release/surface/surface-product-slice-gate.sh
```

Done when:

- Product-slice summary can move from `product_claim: false` to true only when the new Morph rendered beauty gate passes.
- If Linux real-window or browser target is unavailable on a machine, the gate reports `BLOCKED` for that target instead of claiming success.

MRB-12 result, 2026-06-16:

- Added `scripts/release/surface/morph-rendered-beauty-gate.sh` and wired it
  into `scripts/release/surface/surface-product-slice-gate.sh`.
- The product-slice gate now requires Morph rendered beauty artifacts,
  categories, and summary validation before any product/final signoff.
- `examples/surface_morph_rendered_studio_shell.tetra` remains Morph-authored:
  it no longer imports `lib.core.draw`; it renders frames through
  `morph.render_studio_shell_frame` and presents real `surface.Frame` values.
- `lib/core/morph.tetra` owns the temporary Morph frame-render helper and now
  declares `Effects: mem`; `docs/user/standard_library_guide.md` was updated to
  match.
- Fresh evidence:
  `reports/surface/mrb12-morph-rendered-beauty-gate-verify-20260616195220/morph-rendered-beauty-gate-summary.json`
  has schema `tetra.surface.morph-rendered-beauty.gate.v1`, status
  `validated_with_target_blockers`, `pass=true`, `product_claim=false`, and
  `final_signoff=false`.
- Fresh product-slice evidence:
  `reports/surface/mrb12-product-slice-gate-verify-20260616195413/surface-product-slice-summary.json`
  has schema `tetra.surface.product-slice-summary.v1`, flagship source
  `examples/surface_morph_rendered_studio_shell.tetra`,
  `morph_rendered_beauty=validated`, `pass=true`, `product_claim=false`, and
  `final_signoff=false`.
- Fresh browser-canvas runtime evidence:
  `reports/surface/mrb12-morph-source-wasm-20260616194952/wasm32-web-browser-canvas-block-system.json`
  shows `wasm32-web-browser-canvas-input` host evidence with browser canvas and
  input enabled for the Morph flagship source.
- Target blockers remain explicit in the MRB gate summary:
  `linux-x64-real-window` and `wasm32-web-browser-canvas` are `BLOCKED` for
  Morph rendered beauty product claim in this integrated gate, and do not
  create product claims.
- Read-only MRB-12 review found no hard blocker, but flagged the Morph-owned
  `render_studio_shell_frame` helper as an architectural risk if it is treated
  as a second renderer path. It is documented in `lib/core/morph.tetra` as an
  MRB-12 evidence bridge only; `MRB-13` must not promote Morph to stable unless
  this is replaced by, or explicitly constrained beneath, renderer-owned
  Block-first proof.
- Verification passed:
  `go test -buildvcs=false ./tools/cmd/surface-runtime-smoke -run 'TestMorphRenderedFlagshipSourcePresentsSurfaceFrames|TestMorphFlagshipScenarioProducesRenderedBeautyReport' -count=1`,
  `go run -buildvcs=false ./cli/cmd/tetra check examples/surface_morph_rendered_studio_shell.tetra`,
  `go run -buildvcs=false ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system --source examples/surface_morph_rendered_studio_shell.tetra --report reports/surface/mrb12-morph-source-wasm-20260616194952/wasm32-web-browser-canvas-block-system.json`,
  `go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface ./tools/scriptstest ./tools/cmd/validate-surface-product-slice ./tools/cmd/validate-surface-claims -count=1`,
  `go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
  `bash scripts/release/surface/morph-rendered-beauty-gate.sh --report-dir reports/surface/mrb12-morph-rendered-beauty-gate-verify-20260616195220`,
  `bash scripts/release/surface/surface-product-slice-gate.sh --report-dir reports/surface/mrb12-product-slice-gate-verify-20260616195413`,
  and scoped `git diff --check`.

### Task 13 - Stable Promotion Audit

Goal:
Promote Morph only after same-commit end-to-end evidence exists.

Files:

- `docs/spec/surface_morph.md`
- Possible new file: `docs/spec/surface_morph_stable_candidate.md`
- `docs/spec/current_supported_surface.md`
- `scripts/release/surface/surface-product-slice-gate.sh`
- `reports/stabilization/`
- `graphify-out/`

Approach:

- Re-run from a clean checkout or a clearly reported dirty tree.
- Require all required reports to reference the same commit.
- Require visual goldens to be stable and source-linked.
- Require claims/docs/package manifest to agree.
- Audit `morph.render_studio_shell_frame`: stable promotion must either replace
  this MRB-12 evidence bridge with renderer-owned Block-first rendering, or keep
  it explicitly experimental/nonclaim so it cannot become a second core
  primitive or renderer path.
- Run `graphify update .` after implementation changes.
- Do not promote if any target is synthetic-only, metadata-only, or unsupported without explicit nonclaim.

Verification:

```bash
git status --short
graphify update .
bash scripts/release/surface/morph-rendered-beauty-gate.sh
bash scripts/release/surface/surface-product-slice-gate.sh
```

Done when:

- Morph has a stable-candidate document backed by same-commit reports.
- Release summary records final signoff only for actually supported and verified targets.

MRB-13 result, 2026-06-16:

- Added
  `reports/stabilization/surface_morph_rendered_beauty_mrb_13_stable_promotion_audit.md`.
- Updated `docs/spec/surface_morph_stable_candidate.md` with the MRB-13
  stable-promotion denial.
- Ran fresh audit gates:
  `reports/surface/mrb13-morph-rendered-beauty-gate-audit-20260616200009/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb13-product-slice-gate-audit-20260616200041/surface-product-slice-summary.json`.
- Stable candidate validator passed as a design-freeze guard:
  `go test -buildvcs=false ./tools/cmd/validate-surface-morph-stable-candidate -count=1`
  and
  `go run -buildvcs=false ./tools/cmd/validate-surface-morph-stable-candidate --contract docs/spec/surface_morph_stable_candidate_contract.json`.
- Promotion decision: Morph remains `EXPERIMENTAL`; no
  `PROD_STABLE_SCOPED` Morph claim is made.
- Blocking promotion facts:
  dirty worktree (`706` short-status entries at audit time),
  `main...origin/main [behind 12]`,
  MRB gate status `validated_with_target_blockers`,
  `linux-x64-real-window` and `wasm32-web-browser-canvas` still blocked for
  Morph rendered beauty product claim,
  product/MRB summaries keep `product_claim=false` and `final_signoff=false`,
  reports lack machine-visible `git_commit` fields, and
  `morph.render_studio_shell_frame` remains an MRB-12 evidence bridge rather
  than renderer-owned stable proof.

Post-MRB-13 same-commit identity follow-up, 2026-06-16:

- Added machine-visible `git_commit` alias evidence to new MRB reports,
  Morph-to-pixels chains, MRB gate summaries, and product-slice summaries.
- Validators now require `git_commit` to be 40-hex and equal to `git_head` for
  this evidence chain.
- Fresh gates passed:
  `reports/surface/mrb-git-identity-morph-rendered-beauty-gate-20260616171311/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-git-identity-product-slice-gate-20260616171359/surface-product-slice-summary.json`.
- This removes the missing `git_commit` blocker for newly generated evidence,
  but the plan remains `PARTIAL` until target blockers, dirty-worktree audit,
  product/final signoff, and renderer-owned stable proof are resolved.

Post-MRB-13 wasm browser-canvas target follow-up, 2026-06-16:

- Added `wasm32-web-browser-canvas-morph` as a first-class Morph target runtime
  evidence mode for `examples/surface_morph_rendered_studio_shell.tetra`.
- Browser-canvas Morph runtime evidence now carries app-produced
  `product_visual` RGBA frame artifacts, `browser-canvas-rgba` render command
  streams, browser input/canvas host evidence, and Morph-to-pixels MRB reports
  without inventing a synthetic `block_system` for the target path.
- `surface-visual-diff`, `validate-surface-morph-report`, and shared Surface
  validators now accept this target evidence only when the runtime frames are
  source/hash-linked product visual frames.
- Fresh gates passed:
  `reports/surface/mrb-wasm-browser-canvas-morph-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-wasm-browser-canvas-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
- This removes the `wasm32-web-browser-canvas` target blocker for newly
  generated MRB evidence. The plan remains `PARTIAL` until the remaining
  `linux-x64-real-window` target blocker, dirty-worktree audit, product/final
  signoff, and renderer-owned stable proof are resolved.

Post-MRB-13 linux real-window target follow-up, 2026-06-16:

- Added `linux-x64-real-window-morph` as a first-class Morph target runtime
  evidence mode for `examples/surface_morph_rendered_studio_shell.tetra`.
- Linux real-window Morph runtime evidence now carries app-produced
  `product_visual` RGBA frame artifacts, `wayland-shm-rgba` render command
  streams, real-window/native-input host evidence, and Morph-to-pixels MRB
  reports without promoting `host_probe_only` or precomputed frames to product
  visual evidence.
- `surface-visual-diff`, `validate-surface-morph-report`, and shared Surface
  validators now accept this target evidence only when the runtime frames are
  source/hash-linked product visual frames.
- Fresh gates passed:
  `reports/surface/mrb-linux-real-window-morph-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-linux-real-window-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
- This removes the `linux-x64-real-window` target blocker for newly generated
  MRB evidence. The plan remains `PARTIAL` until dirty-worktree audit,
  product/final signoff, and renderer-owned stable proof are resolved.

Post-MRB-13 renderer-owned stable proof guard, 2026-06-16:

- Added explicit `renderer_stable_proof` evidence to
  `tetra.surface.morph-rendered-beauty.v1` reports and validators.
- Current generated reports intentionally mark the pixel owner as
  `morph-evidence-bridge`, with `renderer_owned=false`,
  `bridge_owned_pixels=true`, and `stable_promotion_eligible=false`.
- Product/final claims now require renderer-owned stable proof; bridge-owned
  pixels cannot satisfy promotion even when target artifacts validate.
- MRB gate summaries now separate `target_blockers` from
  `stable_promotion_blockers`.
- The stable-candidate contract and validator now require a
  `renderer-owned stable proof` promotion gate.
- Fresh gates passed:
  `reports/surface/mrb-renderer-proof-guard-gate-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-renderer-proof-guard-product-slice-20260616-verify/surface-product-slice-summary.json`.
- This makes the final proof gap machine-checkable but does not promote Morph.
  The plan remains `PARTIAL` until dirty-worktree audit, product/final signoff,
  and actual renderer-owned stable proof are resolved.

Post-MRB-13 headless renderer-owned stable proof follow-up, 2026-06-16:

- Added deterministic `RenderCommandStreamReport` -> RGBA rendering in
  `tools/internal/surfacerender`.
- Render commands now carry the paint payload needed for renderer-owned pixels:
  color plus border/shadow width, blur, and offsets.
- Headless Morph frame order 1 is now rendered from the command stream, written
  as the runtime frame artifact, and rebound into the command-stream checksum.
- `buildMorphRenderedBeautyReport` rerenders the command stream and only sets
  `renderer_stable_proof.pixel_owner=surface-renderer` when the renderer
  checksum matches pixel-golden evidence.
- Fresh gates passed:
  `reports/surface/mrb-renderer-owned-headless-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-renderer-owned-headless-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
- This resolves renderer-owned stable proof for `headless` only. The plan
  remains `PARTIAL` until `linux-x64-real-window` and
  `wasm32-web-browser-canvas` also have renderer-owned stable proof, the dirty
  worktree is audited, and product/final signoff are intentionally enabled.

Post-MRB-13 all-supported-target renderer-owned stable proof follow-up,
2026-06-16:

- Added source-linked Morph flagship render commands that reproduce the
  `morph.render_studio_shell_frame(false)` pixel path for target reports.
- `buildMorphRenderedBeautyReport` now allows any supported renderer to become
  renderer-owned only when `RenderCommandStreamRGBA` byte-for-byte matches the
  pixel-golden frame checksum.
- `linux-x64-real-window` and `wasm32-web-browser-canvas` MRB reports now use
  the same checksum proof path as `headless`; the gate summary derives
  `renderer_owned_stable_targets` from the actual MRB reports instead of fixed
  bridge-owned target lists.
- Fresh evidence:
  `reports/surface/mrb-target-renderer-owned-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-target-renderer-owned-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
- This still does not promote Morph. The plan remains `PARTIAL` until clean
  checkout audit, `product_claim=true`, and `final_signoff=true` are
  intentionally completed.

Post-MRB-13 promotion-mode signoff follow-up, 2026-06-16:

- Added explicit MRB report signoff flags so product/final claims are an
  intentional promotion action, not a default gate side effect.
- `surface-runtime-smoke` now sets `product_claim=true` and
  `final_signoff=true` only when requested and only when the report has
  `git_dirty=false` plus renderer-owned stable proof.
- `morph-rendered-beauty-gate.sh` and `surface-product-slice-gate.sh` now accept
  `--product-claim --final-signoff`, reject that promotion mode on a dirty
  checkout before heavy evidence generation, and keep the default path as a
  validated nonclaim.
- `validate-surface-product-slice` now accepts both the safe default state and a
  clean promoted state with nested MRB signoff, empty stable promotion blockers,
  and renderer-owned stable proof for all supported targets.
- Fresh evidence:
  `reports/stabilization/surface_morph_rendered_beauty_promotion_mode_audit.md`,
  `reports/surface/mrb-promotion-aware-gate-default-20260616-verify/`, and
  `reports/surface/mrb-promotion-aware-product-default-20260616-verify/`.
- This still does not promote Morph from the current checkout. The plan remains
  `PARTIAL` until promotion mode is run from a clean checkout or clean isolated
  worktree and produces `git_dirty=false`, `product_claim=true`, and
  `final_signoff=true`.

## Acceptance Criteria

This plan is complete only when all of the following are true:

- Morph is still the single beauty layer; there is no duplicate design system.
- Block is still the only core primitive.
- Morph recipes expand to Block scenes.
- Block scenes preserve enough visual spec data for rendering.
- Renderer emits deterministic command streams from app/Morph source.
- Runtime produces actual RGBA or PNG artifacts.
- Visual diff compares real artifacts to separate goldens.
- Precomputed or metadata-only evidence cannot pass product gates.
- Flagship UI is authored through Morph and produces real rendered evidence.
- Developer loop and inspector expose the Morph-to-pixels chain.
- Templates and reference apps demonstrate the same path.
- Claim scanner blocks unsupported beauty/product language.
- Product-slice release gate passes on supported targets or reports exact blockers.
- `graphify update .` has been run after implementation changes.

## Risk Register

- Scene snapshot design may expose limits in current Tetra value/ABI representation. If so, keep Block ABI compact and add sidecar scene evidence for renderer/validator use.
- Text rendering can become a large project. Keep the first pass deterministic and honest: readable raster evidence, not full typography parity.
- Pixel goldens may be noisy across hosts. Normalize artifacts and start with deterministic software rendering before making broad platform claims.
- Real-window/browser availability can block end-to-end evidence on some machines. Gates must report target blockers instead of downgrading requirements silently.
- Existing release scripts may have accumulated infrastructure-only assumptions. Convert them incrementally, with negative fixtures for each removed shortcut.

## Recommended Implementation Strategy

Work in small verified slices:

1. Contract and negative guards.
2. Scene snapshot and command stream.
3. Real frame artifacts and pixel goldens.
4. Morph flagship migration.
5. Developer loop, templates, reference apps.
6. Claims, docs, product gates.
7. Stable promotion audit.

After each slice:

- inspect the diff;
- run the targeted verification command;
- update the plan progress or linked implementation notes;
- avoid claiming `DONE` until the integrated Morph rendered beauty gate passes.
