# Tetra Surface Electron Competitor Platform Plan

**Status:** planning document, not completion evidence.
**Date:** 2026-06-13.
**Owner:** Surface/Morph product slice.
**Related implementation plan:** `docs/plans/2026-06-13-surface-electron-competitor-product-slice.md`.

## Goal

Turn Tetra Surface from a strong UI/runtime evidence track into a developer
visible app platform that can credibly compete with Electron inside a bounded
scope.

The target is not API compatibility with Electron. The target is:

> A developer can create, run, inspect, package, and explain a real desktop-style
> Surface app without Electron, React, DOM-authored app UI, CSS runtime, user
> JavaScript app logic, or native widget UI, with evidence for every claimed
> feature and explicit nonclaims for everything outside scope.

## Why The Current Story Still Does Not Feel Like An Electron Competitor

Surface and Morph already have a lot of serious machinery: Block primitives,
Morph recipes, runtime smoke reports, app-shell evidence, templates, packaging
smokes, validators, claim scanners, and release gates. That is useful, but it
still reads like infrastructure unless it is wrapped around an app-platform
experience.

Electron feels competitive because it gives developers a full loop:

1. Start from a template.
2. Build an app-shaped UI quickly.
3. Run it locally.
4. Debug and inspect it.
5. Use desktop app-shell features.
6. Package artifacts.
7. Publish an honest support story.

Surface needs the same product loop, but with Tetra-native advantages:

- pure Tetra UI;
- small Surface Host ABI;
- Morph recipe authoring instead of web stack authoring;
- Block evidence instead of DOM tree reliance;
- claim-gated release artifacts;
- explicit nonclaims instead of vague parity language.

## Product Thesis

Surface should be positioned as a **bounded native app platform**, not an
Electron clone.

The first credible public claim should be:

> Tetra Surface ships a Linux/web desktop-style app slice through Tetra-native
> UI, app-shell evidence, packaging evidence, and repeatable gates. It is a
> bounded Electron alternative for Tetra apps, not an Electron API replacement.

That means:

- Green claim: Linux x64 real-window and wasm32-web browser-canvas product
  evidence for the flagship slice.
- Amber claim: app-shell features that are scoped, partial, or blocked-pass.
- Red claim: Electron API compatibility, all-platform support, native widgets,
  GPU renderer parity, production signing/notarization, automatic network
  updates, and unsupported platform hosts.

## Flagship Product

Use **Tetra Studio Shell** as the flagship product name.

It can grow out of the existing Control Center work, but the public product
story should be broader than a hardware/control dashboard. The flagship should
look like an app platform example:

- left navigation;
- project/status dashboard;
- command palette;
- settings form;
- logs/output panel;
- diagnostic or error surface;
- blocked-pass dialog for unsupported host features;
- app-shell status bar;
- packaged Linux and web artifacts.

`examples/projects/tetra_control_center` can remain the origin story, while the
Surface-native app source should be treated as the flagship product slice.

## Immediate Direction

The current near-term path is:

1. Finish the product-slice gate that proves the flagship app, developer loop,
   packaging, claims, docs, manifest, and artifact hashes in one fresh report
   directory.
2. Wire that gate into CI without claiming remote GitHub Actions success before
   it actually runs.
3. Use the resulting evidence to promote the story from "Surface primitives are
   promising" to "Surface can ship a bounded Electron-alternative app slice."
4. Start the next milestone from the product gaps below, not from more isolated
   primitives.

## Nonclaims

This plan must not produce or allow these claims unless separate evidence is
added later:

- Electron API compatibility.
- Full Electron API coverage.
- Surface supports production macOS or Windows UI.
- Surface uses or replaces Chromium as a general browser engine.
- Surface has DOM-authored app UI.
- Surface has React compatibility.
- Surface has CSS runtime compatibility.
- Surface has production GPU renderer parity.
- Surface has native platform widget parity.
- Surface has production signing, notarization, app store distribution, or
  automatic network updates.
- Surface has full rich text, full bidi, full screen-reader validation, or full
  accessibility parity.

## Workstream 0: Claim And Evidence Boundary

**Goal:** Freeze the exact support boundary before stronger product language is
published.

**Files to inspect or maintain:**

- `docs/spec/surface_v1.md`
- `docs/spec/current_supported_surface.md`
- `docs/user/surface_guide.md`
- `docs/user/surface_electron_comparison.md`
- `docs/release/surface_v1_release_contract.md`
- `docs/release/surface_v1_release_notes.md`
- `scripts/release/surface/product-gate.sh`
- `scripts/release/surface/surface-product-slice-gate.sh`
- `tools/cmd/validate-surface-claims/`
- `tools/cmd/validate-surface-product-summary/`
- `tools/cmd/validate-surface-product-slice/`

**Approach:**

Keep one canonical claim ladder:

- `PROD_STABLE_SCOPED` only for validated Linux/web Surface scope.
- `BETA_TARGET_HOST` for target-host evidence that is not production Surface UI.
- `EXPERIMENTAL` for Morph, visual, recipe, or future tracks.
- `NONCLAIM` for unsupported Electron/platform/runtime parity.

**Verification:**

```sh
go run ./tools/cmd/validate-surface-claims --root .
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:**

Every public doc uses the same bounded Electron-alternative language, and the
claim scanner rejects overclaims.

## Workstream 1: Flagship App Product Shape

**Goal:** Make Surface visibly app-shaped, not demo-shaped.

**Files to inspect or maintain:**

- `examples/surface_migration_tetra_control_center.tetra`
- `examples/projects/tetra_control_center/README.md`
- `examples/projects/tetra_control_center/docs/surface-flagship-contract.md`
- `examples/surface_morph_studio_shell.tetra`
- `lib/core/block.tetra`
- `lib/core/morph.tetra`
- `tools/cmd/surface-runtime-smoke/`
- `tools/validators/surface/`

**Approach:**

The flagship must include a product-grade shell:

- app frame and navigation;
- dashboard/content panels;
- command palette;
- settings editor;
- logs/output;
- status bar;
- modal or blocked-pass dialog;
- retry/error state;
- scripted user interactions;
- Morph recipe evidence that expands into Block evidence.

**Verification:**

```sh
./tetra check examples/surface_migration_tetra_control_center.tetra
./tetra check examples/surface_morph_studio_shell.tetra
go run ./tools/cmd/surface-runtime-smoke \
  --mode headless-block-system \
  --source examples/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/headless-block-system.json
go run ./tools/cmd/surface-runtime-smoke \
  --mode linux-x64-real-window-block-system \
  --source examples/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/linux-x64-real-window-block-system.json
go run ./tools/cmd/surface-runtime-smoke \
  --mode wasm32-web-browser-canvas-block-system \
  --source examples/surface_migration_tetra_control_center.tetra \
  --report reports/surface-product-slice/flagship/wasm32-web-browser-canvas-block-system.json
```

**Done when:**

The same flagship source has headless, Linux real-window, and wasm32-web
browser-canvas evidence, and the app reads as a real desktop-style tool.

## Workstream 2: Morph As The Authoring Layer

**Goal:** Make Morph feel like the Surface app authoring default.

**Files to inspect or maintain:**

- `lib/core/morph.tetra`
- `docs/spec/surface_morph.md`
- `docs/user/surface_morph_recipe_cookbook.md`
- `examples/surface_morph_*.tetra`
- `scripts/release/surface/morph-gate.sh`
- `tools/cmd/validate-surface-morph-report/`

**Approach:**

Build a practical recipe vocabulary for app products:

- app shell;
- sidebar;
- toolbar;
- split pane;
- tabs;
- command item;
- settings form;
- log row;
- metric tile;
- toast;
- modal/dialog;
- empty state;
- error panel;
- status bar.

Morph should remain a recipe layer that expands to Block. It should not become
a hidden widget framework or a new overclaimed production surface.

**Verification:**

```sh
bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-product-slice/morph
go run ./tools/cmd/validate-surface-morph-report \
  --report reports/surface-product-slice/morph/surface-headless-morph.json
```

**Done when:**

The flagship app is mostly authored through named Morph recipes, and the Morph
gate proves recipe-to-Block evidence.

## Workstream 3: Developer Experience

**Goal:** Make building a Surface app feel like application development, not
manual evidence production.

**Files to inspect or maintain:**

- `cli/cmd/tetra/surface_dev.go`
- `cli/cmd/tetra/surface_dev_test.go`
- `scripts/release/surface/surface-dev-workflow-smoke.sh`
- `tools/cmd/validate-surface-dev-workflow/`
- `docs/user/surface_guide.md`

**Approach:**

The developer loop should give a single documented path:

- run `tetra surface dev`;
- rebuild on token, recipe, and source changes;
- emit `tetra.surface.dev-workflow.v1`;
- explain fast rebuild honestly;
- avoid hot-reload language unless actual hot reload exists.

**Verification:**

```sh
go run ./cli/cmd/tetra surface dev \
  --source examples/surface_migration_tetra_control_center.tetra \
  --target linux-x64 \
  --out-dir reports/surface-product-slice/dev-workflow/dev-artifacts \
  --report reports/surface-product-slice/dev-workflow/surface-dev-workflow.json \
  --change-file token:lib/core/morph.tetra \
  --change-file recipe:lib/core/morph.tetra \
  --change-file source:examples/surface_migration_tetra_control_center.tetra
go run ./tools/cmd/validate-surface-dev-workflow \
  --report reports/surface-product-slice/dev-workflow/surface-dev-workflow.json
```

**Done when:**

A developer can start from the flagship source, run the dev loop, and get a
validated report proving the app development path.

## Workstream 4: Template And Onboarding

**Goal:** Make the first five minutes look like a platform.

**Files to inspect or maintain:**

- `cli/cmd/tetra/new_surface_app.go`
- `cli/cmd/tetra/new_surface_app_test.go`
- `scripts/release/surface/surface-template-smoke.sh`
- `tools/cmd/validate-surface-template-smoke/`
- `docs/user/surface_guide.md`

**Approach:**

Provide a product-shaped `studio-shell` template with:

- `Capsule.t4`;
- `src/main.tetra`;
- token and recipe files;
- README;
- check/build/run instructions;
- Linux and wasm target story;
- no Electron, React, DOM app UI, CSS runtime, platform-widget, or user-JS app
  logic vocabulary.

**Verification:**

```sh
go test ./cli/cmd/tetra -run 'NewSurfaceApp|SurfaceDev' -count=1
bash scripts/release/surface/surface-template-smoke.sh \
  --report-dir reports/surface-product-slice/template-smoke
go run ./tools/cmd/validate-surface-template-smoke \
  --report reports/surface-product-slice/template-smoke/surface-template-smoke.json
```

**Done when:**

`tetra new surface-app --template studio-shell` produces a checked, smoke-tested
Surface app that matches the flagship product story.

## Workstream 5: Packaging And Update Story

**Goal:** Make the app shippable as artifacts, not just runnable as examples.

**Files to inspect or maintain:**

- `scripts/release/surface/surface-package-smoke.sh`
- `tools/cmd/validate-surface-package/`
- `tools/validators/surface/package.go`
- `docs/user/surface_guide.md`
- `docs/release/surface_v1_release_contract.md`
- `scripts/release/surface/README.md`

**Approach:**

Package the flagship for:

- Linux x64 install/run proof;
- wasm32-web bundle proof;
- local asset manifest;
- hash-pinned update manifest;
- explicit nonclaims for signing, notarization, app-store distribution,
  automatic network updates, and remote asset fetching.

**Verification:**

```sh
bash scripts/release/surface/surface-package-smoke.sh \
  --report-dir reports/surface-product-slice/package \
  --source examples/surface_migration_tetra_control_center.tetra \
  --app-id studio-shell \
  --app-title "Tetra Studio Shell" \
  --expected-exit-code 5
go run ./tools/cmd/validate-surface-package \
  --report reports/surface-product-slice/package/surface-package.json
go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/surface-product-slice/package/artifact-hashes.json
```

**Done when:**

The package report proves Linux and web artifacts for the flagship and names
all unsupported distribution features as nonclaims.

## Workstream 6: App-Shell Feature Ledger

**Goal:** Account for Electron-like app-shell expectations without claiming
Electron parity.

**Files to inspect or maintain:**

- `docs/plans/2026-06-12-surface-app-shell-electron-feature-ledger.md`
- `lib/core/surface_app_shell.tetra`
- `tools/cmd/surface-runtime-smoke/`
- `tools/validators/surface/linux_app_shell_validation.go`
- `scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh`
- `docs/user/surface_electron_comparison.md`

**Approach:**

Track each feature as supported, scoped, blocked-pass, or unsupported:

- window lifecycle;
- multi-window;
- app menu;
- clipboard;
- IME/composition;
- accessibility bridge metadata;
- crash/error report;
- file dialog;
- file picker;
- notification;
- tray;
- deep link.

**Verification:**

```sh
bash scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh \
  --report-dir reports/surface-product-slice/app-shell
go test ./tools/validators/surface ./tools/cmd/validate-surface-runtime \
  -run 'LinuxAppShell|AppShell|Electron|Claim' \
  -count=1
```

**Done when:**

Every Electron-like app-shell row is either evidenced or explicitly nonclaimed.

## Workstream 7: Inspector And Debuggability

**Goal:** Give developers a way to understand Surface app state and rendering
without reading raw JSON reports.

**Files to inspect or maintain:**

- `tools/cmd/surface-runtime-smoke/`
- `tools/cmd/validate-surface-runtime/`
- `scripts/release/surface/README.md`
- any existing Surface inspector docs or generated HTML reports found during
  implementation.

**Approach:**

The product story should expose:

- Block tree summary;
- Morph recipe expansion summary;
- event trace;
- draw trace;
- accessibility metadata trace;
- package artifact links;
- claim/nonclaim summary.

If an inspector already exists, wire the flagship into it. If it does not,
plan a minimal static HTML or JSON-to-summary artifact as a follow-up.

**Verification:**

```sh
rg -n "surface-inspector|inspector|Block tree|Morph tokens" \
  docs scripts tools examples
```

**Done when:**

The flagship evidence has a human-readable inspection path, not only validator
output.

## Workstream 8: Accessibility, Text, And Localization Reality

**Goal:** Keep app-platform credibility without overstating accessibility/text
support.

**Files to inspect or maintain:**

- `docs/spec/surface_v1.md`
- `docs/user/surface_guide.md`
- `tools/validators/surface/accessibility_validation.go`
- `tools/validators/surface/text_input_validation.go`
- `scripts/release/surface/*accessibility*`
- `scripts/release/surface/*i18n*`

**Approach:**

The flagship should include:

- accessibility metadata evidence;
- text input evidence where scoped;
- i18n/localization smoke where scoped;
- explicit nonclaims for full screen-reader validation, full rich text, full
  bidi shaping, full ICU, and all-platform accessibility parity.

**Verification:**

```sh
rg -n "accessibility|text input|i18n|bidi|screen-reader|rich text" \
  docs scripts tools examples
go test ./tools/validators/surface -run 'Accessibility|Text|I18n|Bidi' -count=1
```

**Done when:**

The Electron comparison has honest accessibility and text rows backed by local
evidence.

## Workstream 9: Performance And Memory Story

**Goal:** Make the competitive angle measurable without making premature
"faster than Electron" claims.

**Files to inspect or maintain:**

- `docs/spec/surface_v1.md`
- `docs/release/surface_v1_release_notes.md`
- `scripts/release/surface/README.md`
- Surface runtime smoke reports under `tools/cmd/surface-runtime-smoke/`
- benchmark plans under `docs/plans/`

**Approach:**

Use bounded local measurements:

- startup/build/run timings where already emitted;
- artifact sizes;
- memory-footprint evidence if a stable local metric exists;
- no marketing claim unless the benchmark is repeatable and documented.

**Verification:**

```sh
rg -n "performance|memory|benchmark|faster|Electron" \
  docs scripts tools reports
```

**Done when:**

Docs say what is measured, what is not measured, and what must not be claimed.

## Workstream 10: Security And Permission Model

**Goal:** Avoid shipping an app-platform story that ignores Electron's security
conversation.

**Files to inspect or maintain:**

- `docs/plans/2026-06-12-surface-security-permission-model.md`
- `docs/spec/surface_v1.md`
- `docs/user/surface_electron_comparison.md`
- Surface package and app-shell validators.

**Approach:**

Document:

- local-only assets;
- no remote asset fetch claim;
- no crash upload service claim;
- no automatic update download claim;
- host capability boundaries;
- future permission prompts or manifests if needed.

**Verification:**

```sh
rg -n "permission|security|remote asset|network update|crash upload|capability" \
  docs scripts tools
```

**Done when:**

The comparison doc and release docs explain the security boundary as a product
feature, not as a missing footnote.

## Workstream 11: Product Docs And Demo Narrative

**Goal:** Make the repo explain Surface as a product platform at first read.

**Files to inspect or maintain:**

- `docs/user/surface_guide.md`
- `docs/user/surface_electron_comparison.md`
- `docs/user/surface_morph_recipe_cookbook.md`
- `docs/release/surface_v1_release_notes.md`
- `examples/projects/tetra_control_center/README.md`
- `scripts/release/surface/README.md`

**Approach:**

The docs should answer these questions quickly:

- What is Tetra Surface?
- What is the first flagship app?
- How do I create one?
- How do I run it?
- How do I inspect it?
- How do I package it?
- Which Electron-like features are supported?
- Which features are not claimed?
- Which commands prove the claims?

**Verification:**

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-surface-claims --root .
```

**Done when:**

A new developer can understand the Surface product story without reading
validator internals first.

## Workstream 12: Product-Slice Gate

**Goal:** Make the Electron-competitor story repeatable and fake-resistant.

**Files to inspect or maintain:**

- `scripts/release/surface/surface-product-slice-gate.sh`
- `tools/cmd/validate-surface-product-slice/`
- `tools/scriptstest/surface_product_gate_test.go`
- `scripts/release/surface/README.md`

**Approach:**

The gate should require a fresh report directory and collect:

- flagship runtime evidence;
- developer loop evidence;
- template/onboarding evidence;
- package/update evidence;
- claim scanner evidence;
- docs/manifest evidence;
- artifact hashes;
- category summaries;
- final `tetra.surface.product-slice-summary.v1`.

**Verification:**

```sh
bash scripts/release/surface/surface-product-slice-gate.sh \
  --report-dir reports/surface-product-slice/product-gate
go run ./tools/cmd/validate-surface-product-slice \
  --report-dir reports/surface-product-slice/product-gate
go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/surface-product-slice/product-gate/artifact-hashes.json
```

**Done when:**

The gate fails if any flagship evidence, docs, manifest, claims, or artifact
hashes are missing or stale.

## Workstream 13: CI And GitHub Actions

**Goal:** Make the product slice visible in automation.

**Files to inspect or maintain:**

- `.github/workflows/ci.yml`
- `.github/workflows/full-platform-ui-runtime.yml`
- `tools/scriptstest/ci_workflow_test.go`
- `tools/scriptstest/surface_product_gate_test.go`

**Approach:**

Add a CI job that can run the product-slice gate without weakening existing
release gates. Keep GitHub Actions status honest:

- local lint and script tests prove wiring;
- remote run proves GitHub Actions success;
- if remote execution is unavailable, report `PARTIAL`.

**Verification:**

```sh
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
go test ./tools/scriptstest -run 'Workflow|Surface|Product|Release' -count=1
```

**Done when:**

CI can run the product-slice job and the workflow syntax plus script tests pass.

## Workstream 14: Release Decision

**Goal:** Decide whether the milestone is `DONE`, `PARTIAL`, or `BLOCKED` from
evidence.

**Files to inspect or maintain:**

- `GOAL.md`
- `docs/plans/2026-06-13-surface-electron-competitor-product-slice.md`
- `docs/plan/2026-06-13-surface-electron-competitor-platform-plan.md`
- final reports under `reports/surface-product-slice/`

**Approach:**

Do not call the work complete because one validator, one app, or one report
passes. Final status needs the full product loop:

- app;
- Morph authoring;
- dev loop;
- template;
- packaging;
- app-shell ledger;
- claims;
- docs;
- manifest;
- artifact hashes;
- CI wiring;
- clean or explicitly scoped checkout status.

**Verification:**

```sh
export GOCACHE="$(pwd)/.cache/go-build-surface-product-slice"
export GOTMPDIR="$(pwd)/.cache/go-tmp-surface-product-slice"
mkdir -p "$GOCACHE" "$GOTMPDIR"
go test ./cli/cmd/tetra ./tools/cmd/surface-runtime-smoke ./tools/validators/surface ./tools/scriptstest \
  -run 'Surface|surface' \
  -count=1
bash scripts/release/surface/surface-product-slice-gate.sh \
  --report-dir reports/surface-product-slice/product-gate-final
go run ./tools/cmd/validate-surface-claims \
  --root . \
  --report-dir reports/surface-product-slice/product-gate-final
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
GOCACHE="$GOCACHE" go clean -cache
```

**Done when:**

All acceptance criteria are evidenced, remaining nonclaims are explicit, and
the final report can honestly say `DONE`. If remote CI or final clean-checkout
evidence is missing, status must be `PARTIAL`.

## Acceptance Criteria

This platform plan is complete when:

1. A flagship app exists and reads as a real app, not a widget demo.
2. The app is authored primarily through Surface/Block/Morph.
3. The developer loop is documented and validated.
4. The onboarding template creates a product-shaped app.
5. Linux and wasm package artifacts are produced and hash-verified.
6. App-shell features are accounted for in an Electron-style ledger.
7. Unsupported Electron/platform features are explicit nonclaims.
8. Docs explain the product story before exposing validator internals.
9. The product-slice gate proves the whole loop.
10. CI can run the product-slice gate or the missing remote execution is named
    as a blocker.

## Recommended Execution Order

1. Finish Workstream 12 locally, because the gate becomes the truth source.
2. Finish Workstream 13, because automation turns the slice from local proof
   into project behavior.
3. Audit Workstreams 0 and 11, because claim language decides whether the story
   is credible.
4. Fill Workstreams 7, 8, 9, and 10 as the next product-quality wave.
5. Re-run Workstream 14 and report `DONE`, `PARTIAL`, or `BLOCKED`.

## Next Product Milestone After This Slice

After the current product-slice work lands, the next milestone should be:

**Surface App Platform v0: Studio Shell + Inspector + Packager**

That milestone should prioritize:

- a polished flagship experience;
- a human-readable inspector artifact;
- a template that matches the flagship;
- a package command or documented package script with stable UX;
- screenshot or visual evidence where possible;
- a sharper Electron comparison page with links to evidence.

This is the point where Surface begins to feel like a product platform rather
than a technically impressive runtime subsystem.
