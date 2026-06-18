# Surface Electron Competitor Product Slice Implementation Plan

**Goal:** Turn Tetra Surface from an evidence-rich UI subsystem into a developer-visible app
platform slice that can credibly be shown as a bounded Electron alternative.

**Context:** Current Surface/Morph work has strong primitives, validators, reports, examples, and
release gates, but it still reads as infrastructure. The next milestone must make a real app feel
shippable: create a flagship Surface application, make Morph recipes the authoring default, prove
the developer loop, package the app, and publish an honest Electron comparison matrix.

**Execution:** Use `subagent-driven-development` or `executing-plans` task-by-task. Do not start
broad implementation until the flagship scope and acceptance gates below are accepted.

**Path note:** The request named `docs/plan/`; this repository stores execution plans in
`docs/plans/YYYY-MM-DD-<topic>.md`, so this plan follows the local convention.

## Verified Starting Facts

- `docs/spec/surface/surface_v1.md` defines the bounded `surface-v1-linux-web` scope: Linux
  real-window/app-shell and wasm32-web browser-canvas are current scoped Surface targets; macOS,
  Windows, and wasm32-wasi Surface UI remain unsupported for production claims.
- `docs/release/surface/surface_product_readiness_audit.md` records `BLOCKED_DIRTY_CHECKOUT`; the
  product gate evidence passed in that checkout, but final clean same-commit `PROD_STABLE_SCOPED`
  readiness was not granted.
- `docs/plans/2026-06/product-memory-toon/2026-06-12-surface-app-shell-electron-feature-ledger.md`
  already accounts for Electron-like app-shell rows through the scoped Linux app-shell ledger; it
  explicitly avoids a broad Electron API compatibility claim.
- `docs/user/surface/surface_guide.md` documents `tetra surface dev`, `tetra new surface-app`, the
  reference app suite, package smoke, crash report smoke, i18n smoke, widget migration smoke, and
  the current Surface claim tiers.
- `cli/cmd/tetra/tetra_commands.go` currently exposes `tetra surface dev`; no verified
  `tetra surface package` subcommand exists yet.
- `cli/cmd/tetra/tetra_commands.go` currently supports six templates: `command-palette`, `settings`,
  `dashboard`, `editor-shell`, `multi-window-notes`, and `web-canvas`; no verified `studio` or
  `control-center` template exists yet.
- `examples/projects/tetra_control_center` already exists as a Tetra-first project with a Python
  helper and a small web host sidecar. Its architecture document states that the sidecars exist
  because the current Tetra UI surface does not yet provide the rich dashboard layout engine
  directly.
- `examples/surface/migration/surface_migration_tetra_control_center.tetra` already exists as a pure
  Surface migration example for the Tetra Control Center shape.
- `scripts/release/surface/product-gate.sh` is the current scoped product evidence gate and writes
  `tetra.surface.product-summary.v1` reports.

## Product Decision

Use **Tetra Control Center / Studio Shell** as the first flagship product slice. The existing
`examples/projects/tetra_control_center` gives the work a real app shape, while
`examples/surface/migration/surface_migration_tetra_control_center.tetra` gives a pure Surface
migration seed.

The product claim is not "Surface implements Electron." The claim is:

> Tetra Surface can ship a bounded Linux/web desktop-style app without Electron, Chromium as an app
> runtime, React, DOM-authored application UI, CSS runtime, user JavaScript app logic, or
> platform-native widget UI.

## Acceptance Criteria

The slice is complete only when all of these are true:

1. A flagship Surface app launches as a real linux-x64 Surface app and as a wasm32-web
   browser-canvas app inside the existing scoped target boundaries.
2. The flagship core UI is authored through Surface/Block/Morph rather than the existing web host
   sidecar. Any remaining Python/helper/web boundary is documented as integration plumbing, not
   Surface UI capability.
3. The app includes product-grade surfaces: navigation, dashboard/content panels, command palette,
   settings, logs/output, status bar, modal/dialog or blocked-pass dialog row, error/crash surface,
   and app-shell state.
4. Morph recipes are the default authoring shape for product UI, and they expand to Block evidence
   with no new core widget primitive promotion.
5. `tetra surface dev` works on the flagship slice and emits `tetra.surface.dev-workflow.v1`
   evidence for token, recipe, and source changes.
6. Packaging evidence exists for linux-x64 and wasm32-web artifacts, including local assets,
   install/run proof for Linux, and update-channel manifest evidence inside the current nonclaim
   boundaries.
7. An honest Electron comparison document exists with green/amber/red rows and explicit nonclaims
   for Electron API compatibility, all-platform support, GPU renderer parity, native widget parity,
   full rich text, full bidi, full screen-reader validation, signing/notarization, and automatic
   network updates.
8. The product gate or a new product-slice gate includes the flagship evidence, claim scanner
   coverage, artifact hashes, and docs/manifest verification.
9. Final status is not reported as `DONE` until the relevant commands pass on a clean or explicitly
   scoped checkout and the evidence paths are recorded.

## Non-Goals

- Do not claim full Electron API compatibility.
- Do not claim macOS or Windows Surface production support.
- Do not claim GPU rendering, native platform widgets, React compatibility, DOM-authored app UI, CSS
  runtime compatibility, or user JavaScript app logic.
- Do not claim production signing, notarization, automatic network updates, or remote asset fetching
  unless separate evidence is added.
- Do not turn `lib.core.widgets` into the future core primitive set. Keep it as compatibility while
  Morph recipes and Block carry the new direction.
- Do not hide the existing dirty-checkout final readiness blocker.

## Task 0: Baseline Scope Freeze

**Goal:** Freeze the exact before-state so later work cannot silently turn into a local demo or a
docs-only claim.

**Files:** Inspect `docs/spec/surface/surface_v1.md`,
`docs/release/surface/surface_product_readiness_audit.md`, `docs/user/surface/surface_guide.md`,
`docs/plans/2026-06/product-memory-toon/2026-06-12-surface-app-shell-electron-feature-ledger.md`,
`examples/projects/tetra_control_center/README.md`,
`examples/projects/tetra_control_center/docs/architecture.md`,
`examples/surface/migration/surface_migration_tetra_control_center.tetra`,
`cli/cmd/tetra/tetra_commands.go`, and `cli/cmd/tetra/tetra_commands.go`.

**Approach:** Record the current claim tier, supported targets, unsupported targets, app-shell
ledger status, existing template list, current developer loop, and the Control Center sidecar
boundary. Add an investigation note for any exact command or report path not verified in the current
checkout.

**Verification:**

```sh
git status --short
git rev-parse HEAD
rg -n "surface dev|new surface-app|surface-package|product-gate|Electron|NONCLAIM" docs cli scripts examples
```

**Done when:** The implementation issue/branch notes name the exact current claim tier,
dirty-checkout state, target boundary, and known nonclaims before any code changes.

**Notes:** The current checkout may be dirty. Do not revert unrelated work.

## Task 1: Flagship App Contract

**Goal:** Define the flagship Surface app contract before implementation.

**Files:** Modify or add docs under `examples/projects/tetra_control_center/` and, if needed, add a
short product-slice contract under `docs/spec/` or `docs/design/`. Inspect
`examples/surface/migration/surface_migration_tetra_control_center.tetra` before deciding whether to
extend it or create a new flagship source.

**Approach:** Specify the visible app shell: navigation, dashboard, profiles/actions or
project/status panels, command palette, settings, logs/output, status bar, diagnostic/error view,
and app-shell lifecycle rows. Separate pure Surface UI from integration helpers. If the hardware
Control Center domain is too narrow for the Electron comparison, keep the app shape but name the
slice `Tetra Studio Shell` in docs and templates.

**Verification:**

```sh
./tetra check examples/surface/migration/surface_migration_tetra_control_center.tetra
bash examples/projects/tetra_control_center/scripts/smoke.sh
```

**Done when:** The contract states what is Surface-owned, what remains helper plumbing, which
screens must render, and which app-shell features are claimed, scoped, or blocked-pass.

**Notes:** The existing Control Center smoke uses a Python helper and a web host sidecar. Promotion
to Surface flagship requires removing that sidecar from the core UI claim, not pretending it is
already pure Surface.

## Task 2: Morph Product Recipe Kit

**Goal:** Make Morph recipes feel like the default app authoring surface rather than scattered
examples.

**Files:** Inspect/modify `lib/core/morph/morph.tetra`, Morph examples under
`examples/surface/morph_core/surface_morph_*.tetra`, Block examples under
`examples/surface/block_*/surface_block_*.tetra`, `docs/spec/surface/morph/surface_morph.md`,
`docs/spec/surface/morph/surface_morph_stable_candidate.md`, and
`docs/user/surface/surface_morph_recipe_cookbook.md`.

**Approach:** Add or harden recipes for product UI shapes: app shell, sidebar, toolbar, tabs, split
pane, status bar, command item, settings form, log row, metric tile, toast, modal/dialog surface,
empty state, and error panel. Keep the recipe output as Block evidence and reject new core widget
primitive claims.

**Verification:**

```sh
bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-product-slice/morph
go run ./tools/cmd/validate-surface-morph-report \
  --report reports/surface-product-slice/morph/surface-headless-morph.json
```

**Done when:** The flagship app can be authored mostly through named Morph recipes, and the Morph
gate proves recipe-to-Block evidence without production Morph overclaiming.

**Notes:** If the existing Morph gate report path differs, inspect the script output and update the
verification command before executing.

## Task 3: Surface-Native Flagship App

**Goal:** Build the flagship app as a Surface/Block/Morph app, not a web-hosted preview.

**Files:** Likely modify `examples/surface/migration/surface_migration_tetra_control_center.tetra`
or add a new verified flagship example under `examples/`. If a full project shape is needed, inspect
`examples/projects/tetra_control_center/` and decide whether to add `src/surface_main.tetra` or a
sibling project. Modify tests under `tools/scriptstest/` and validators under
`tools/validators/surface/` only when the evidence schema requires it.

**Approach:** Implement the app screens as pure Tetra Surface state and Block/Morph composition.
Include scripted interactions for navigation, profile or project selection, command palette action,
settings edit, log selection, error/retry state, and refresh/rebuild action. Keep helper data mocked
or bounded unless a safe runtime integration already exists.

**Verification:**

```sh
./tetra check examples/surface/migration/surface_migration_tetra_control_center.tetra
go run ./tools/cmd/surface-runtime-smoke \
  --mode headless-block-system \
  --source examples/surface/migration/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/headless-block-system.json
go run ./tools/cmd/surface-runtime-smoke \
  --mode linux-x64-real-window-block-system \
  --source examples/surface/migration/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/linux-x64-real-window-block-system.json
go run ./tools/cmd/surface-runtime-smoke \
  --mode wasm32-web-browser-canvas-block-system \
  --source examples/surface/migration/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/wasm32-web-browser-canvas-block-system.json
```

**Done when:** The same flagship source has validated headless, linux-x64 real-window, and
wasm32-web browser-canvas evidence, with visible app-like screens and no web-host sidecar in the
core UI claim.

**Notes:** Real-window evidence may be blocked without a display host. If so, write the blocker
explicitly and do not promote headless evidence to a real-window claim. If the existing
`surface-runtime-smoke` modes retarget the source path but still emit generic Block-system evidence,
add a flagship-specific smoke mode before claiming this task complete.

## Task 4: Developer Loop That Feels Productive

**Goal:** Make the developer path feel like app development rather than report generation.

**Files:** Inspect/modify `cli/cmd/tetra/tetra_commands.go`, `cli/cmd/tetra/tetra_suite_test.go`,
`scripts/release/surface/surface-dev-workflow-smoke.sh`, `tools/cmd/validate-surface-dev-workflow/`,
and docs in `docs/user/surface/surface_guide.md`.

**Approach:** Run `tetra surface dev` against the flagship source or a generated flagship fixture.
Ensure the report captures token, recipe, and source changes. Do not claim hot reload unless real
hot reload exists; keep the current "fast rebuild" language.

**Verification:**

```sh
bash scripts/release/surface/surface-dev-workflow-smoke.sh \
  --report-dir reports/surface-product-slice/dev-workflow
go run ./tools/cmd/validate-surface-dev-workflow \
  --report reports/surface-product-slice/dev-workflow/surface-dev-workflow.json
```

**Done when:** A developer can run a single documented command for the flagship fast rebuild loop
and get a validated `tetra.surface.dev-workflow.v1` report.

**Notes:** If the existing smoke remains fixture-only, add a follow-up task to point it at the
flagship app or add a second flagship-specific smoke.

## Task 5: Project Template / Onboarding

**Goal:** Make the first five minutes look like an app platform.

**Files:** Inspect/modify `cli/cmd/tetra/tetra_commands.go`, `cli/cmd/tetra/tetra_suite_test.go`,
`scripts/release/surface/surface-template-smoke.sh`, and template docs in
`docs/user/surface/surface_guide.md`.

**Approach:** Add a `studio-shell` or `control-center` template only after the flagship contract is
stable. Generated projects must include `Capsule.t4`, `src/main.tetra`, design tokens, design
recipes, README, tests, linux-x64 and wasm32-web targets, and no forbidden runtime vocabulary. If
adding a new template is too much for this slice, document the existing `editor-shell` plus
`dashboard` path as the first onboarding story and leave the new template as a follow-up.

**Verification:**

```sh
bash scripts/release/surface/surface-template-smoke.sh \
  --report-dir reports/surface-product-slice/templates
go test ./cli/cmd/tetra -run 'SurfaceApp|Surface|Template' -count=1
```

**Done when:** The onboarding path can generate, check, build, run, inspect, visually test, and
package a product-shaped Surface app without Electron, React, DOM app UI, CSS runtime, platform
widgets, or user JS app logic.

**Notes:** `tetra new surface-app --template studio-shell` is not a verified command today. Treat it
as implementation work, not documentation truth, until the CLI and smoke tests prove it.

## Task 6: Packaging And Update Story

**Goal:** Turn the flagship into a shippable artifact story.

**Files:** Inspect/modify `scripts/release/surface/surface-package-smoke.sh`,
`tools/cmd/validate-surface-package/`, `tools/validators/surface/surface_morph_release.go`, and
package docs under `docs/user/surface/surface_guide.md` or `docs/release/`.

**Approach:** Retarget or extend the package smoke so the flagship app, not only the command-palette
reference app, has linux-x64 and wasm32-web package evidence. Keep local asset hashes, install/run
proof, web bundle proof, and hash-pinned update manifest. Add a `tetra surface package` CLI only
after an investigation confirms the command shape and tests; otherwise keep the script as the
verified path.

**Verification:**

```sh
bash scripts/release/surface/surface-package-smoke.sh \
  --report-dir reports/surface-product-slice/package
go run ./tools/cmd/validate-surface-package \
  --report reports/surface-product-slice/package/surface-package.json
```

**Done when:** The package report proves linux-x64 and wasm32-web flagship artifacts, local asset
manifests, installed Linux run evidence, web bundle evidence, and explicit nonclaims for
signing/notarization/automatic network updates.

**Notes:** Do not document `tetra surface package` as current until the CLI subcommand exists and
tests prove it.

## Task 7: Electron Comparison And Claim Governance

**Goal:** Make the competitive story honest and visible.

**Files:** Add a comparison doc under `docs/user/` or `docs/release/`; update
`docs/spec/surface/surface_v1.md`, `docs/release/surface/surface_v1_release_contract.md`,
`docs/release/surface/surface_v1_release_notes.md`, and claim validators only as needed.

**Approach:** Publish a green/amber/red matrix. Green rows should be scoped features with evidence,
amber rows should be partial or blocked-pass scoped features, and red rows should be
unsupported/nonclaims. Include app window, multi-window notes, menu, clipboard, IME/composition,
accessibility metadata and bridge evidence, crash/error diagnostics, packaging, dev loop, inspector,
templates, file dialog, file picker, notifications, tray, deep links, macOS, Windows, GPU, rich
text, and native widgets.

**Verification:**

```sh
go run ./tools/cmd/validate-surface-claims \
  --root . \
  --report-dir reports/surface-product-slice/product-gate
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:** The comparison doc explains why Surface is a bounded Electron alternative without
overstating API parity or platform parity, and the claim scanner accepts the wording.

**Notes:** Regenerating `docs/generated/manifest.json` may expose unrelated dirty-worktree diffs.
Treat that as a final readiness blocker, not as a reason to skip docs verification.

## Task 8: Product-Slice Gate

**Goal:** Make the flagship evidence part of a repeatable gate.

**Files:** Inspect/modify `scripts/release/surface/product-gate.sh`,
`scripts/release/surface/release-gate.sh`, `tools/scriptstest/`, and
`tools/cmd/validate-surface-product-summary/`.

**Approach:** Add a product-slice category for the flagship app, or add a separate
`surface-product-slice-gate.sh` if changing the current product gate would blur the existing release
contract. The gate must require fresh report directories, artifact hashes, claim validation,
manifest validation, docs verification, and category summaries.

**Verification:**

```sh
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-slice/product-gate
go run ./tools/cmd/validate-surface-product-summary \
  --report-dir reports/surface-product-slice/product-gate
go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/surface-product-slice/product-gate/artifact-hashes.json
```

**Done when:** The product-slice gate can be run from a fresh report directory and fails if the
flagship evidence, claims, docs, manifest, or artifact hashes are missing/stale.

**Notes:** Keep the final verdict blocked if the checkout is dirty. Do not convert product-slice
evidence into a final release signoff automatically.

## Task 9: CI And GitHub Actions Wiring

**Goal:** Make the slice visible in automation without pretending CI has passed before it runs.

**Files:** Inspect/modify `.github/workflows/ci.yml`,
`.github/workflows/full-platform-ui-runtime.yml`, and any workflow tests under `tools/scriptstest/`.

**Approach:** Add a manually triggerable or pull-request-scoped job for the product-slice gate after
it is stable locally. Keep full-platform target-host jobs separate from the Linux/web Surface
product slice. Do not make macOS or Windows production claims from build-only or unsupported
target-host evidence.

**Verification:**

```sh
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
go test ./tools/scriptstest -run 'Workflow|Surface|Product|Release' -count=1
```

**Done when:** The workflow syntax is linted, script tests prove the gate is wired, and GitHub
Actions can run the product-slice job without weakening existing release gates.

**Notes:** If remote GitHub Actions execution is unavailable in the current environment, report that
as `PARTIAL` and keep local workflow validation as the evidence.

## Task 10: Final Validation And Handoff

**Goal:** Prove the slice end-to-end and document what is still outside scope.

**Files:** Update the final plan progress, relevant release/user docs, and evidence paths. Run
Graphify after code changes per repo instructions.

**Approach:** Run focused tests first, then the product-slice gate, then docs and manifest checks.
Use persistent repo-local Go cache and temp paths for manual Go commands.

**Verification:**

```sh
export GOCACHE="$(pwd)/.cache/go-build-surface-product-slice"
export GOTMPDIR="$(pwd)/.cache/go-tmp-surface-product-slice"
mkdir -p "$GOCACHE" "$GOTMPDIR"
go test ./cli/cmd/tetra ./tools/cmd/surface-runtime-smoke ./tools/validators/surface ./tools/scriptstest \
  -run 'Surface|surface' \
  -count=1
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-slice/product-gate-final
go run ./tools/cmd/validate-surface-claims \
  --root . \
  --report-dir reports/surface-product-slice/product-gate-final
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
go clean -cache
```

**Done when:** The flagship app, dev loop, packaging, claims, docs, manifest, artifact hashes, and
product-slice gate all pass or any remaining blocker is named with an exact command and evidence
path.

**Notes:** `DONE` requires end-to-end evidence. A single passing validator, runtime report,
screenshot, or package smoke is only local evidence.

## Recommended Execution Order

1. Task 0, because it freezes the truth boundary.
2. Task 1, because it prevents the flagship from becoming another demo.
3. Task 2, because product UI needs a recipe vocabulary before app polish.
4. Task 3, because the real app is the center of the milestone.
5. Task 4 and Task 5, because developer experience must orbit the flagship.
6. Task 6, because the app must become an artifact.
7. Task 7, because the competitive story must be honest before promotion.
8. Task 8 and Task 9, because repeatable gates and CI come after local proof.
9. Task 10, because final status needs full evidence, not one local pass.

## Open Questions Before Implementation

- Should the public flagship name be `Tetra Control Center`, `Tetra Studio`, or
  `Tetra Studio Shell`?
- Should the first product-slice gate extend `product-gate.sh` or live as a separate gate until the
  slice is stable?
- Is a new `tetra surface package` command required for this milestone, or is script-backed package
  evidence acceptable for the first product slice?
- Should the existing web-hosted Control Center remain as an integration demo, or should it be
  explicitly deprecated once the Surface-native flagship exists?
