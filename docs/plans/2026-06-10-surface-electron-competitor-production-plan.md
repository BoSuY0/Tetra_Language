# Tetra Surface Electron-Competitor Production Plan

Date: 2026-06-10  
Target repo/dump inspected: `tetra_language_dump_20260610_070301Z_*`  
Observed dump Git HEAD: `c0258b63a636775b114d69d31cb7832fc3991b05`  
Plan file: `docs/plans/2026-06-10-surface-electron-competitor-production-plan.md`

This file is a plan for Codex, not an implementation patch. It is written for a coding agent that must turn Tetra Surface from bounded Surface v1 + experimental Block/Morph evidence into a stable production UI platform that can compete with Electron inside explicitly supported scope.

No more vibes. No fake Electron replacement claim. The summit flag is allowed only after the gates in this file force the claim to be true.

---

## 1. Executive Verdict

The recommended route is **Block/Morph-first, Tetra-owned Surface Platform**: keep `Block` as the primitive, promote Morph from experimental evidence layer into a stable typed style/authoring graph, harden the renderer/compositor/text/layout/input/accessibility/runtime contracts, and add platform host adapters for app-shell services that Electron normally provides. Do **not** copy Electron, React, DOM, CSS cascade, Chromium shell, or platform-native widget trees. The repository is **not currently ready** for the final Electron-competitor claim: Surface v1 is scoped and current for bounded linux/web evidence; Block System and Morph are still experimental/non-production; Windows/macOS Surface production, full app shell, devtools, packaging/update, full accessibility, rich text, and competitive performance gates are missing or incomplete.

Recommended final first production tier:

```text
PROD_STABLE_SCOPED_LINUX_WEB_APP_UI
= Linux-x64 desktop app shell + wasm32-web browser-canvas output + headless tests
  with stable Block/Morph, Tetra-owned renderer, app shell, devtools basics,
  accessibility evidence, packaging story, security model, visual regression,
  performance budget, and same-commit clean release gates.
```

Broader Electron replacement language is allowed only after Windows and macOS reach the same gate standard and the claim text names the exact supported matrix.

---

## 2. Current Truth From The Repository

### 2.1 Repository/dump state inspected

The 2026-06-10 dump reports:

```text
Generated: 2026-06-10T07:03:02Z
Git HEAD: c0258b63a636775b114d69d31cb7832fc3991b05
Files listed: 2342
```

This is a dump, not a live `.git` checkout. The extracted local copy has no `.git`, so any final same-commit/clean-worktree claim must be regenerated inside the real repository checkout.

Graphify was requested by `AGENTS.md`, but `graphify-out/GRAPH_REPORT.md` was not present in the uploaded dump. Treat Graphify as unavailable in this analysis; Codex must read it in the real checkout if present.

Targeted validator check run in the extracted dump:

```bash
GOTELEMETRY=off \
GOCACHE=$PWD/.cache/go-build-plan-check \
GOTMPDIR=$PWD/.cache/go-tmp-plan-check \
go test -buildvcs=false \
  ./tools/cmd/validate-surface-morph-report \
  ./tools/cmd/validate-surface-block-report \
  ./tools/cmd/validate-surface-release-state \
  -count=1
```

Result:

```text
ok tetra_language/tools/cmd/validate-surface-morph-report
ok tetra_language/tools/cmd/validate-surface-block-report
ok tetra_language/tools/cmd/validate-surface-release-state
```

A broader extracted-dump semantics slice failed before test execution because `compiler/internal/pgrt` contained malformed/redacted dump fragments (`unexpected <`). This is treated as a dump extraction/redaction blocker, not evidence that the real checkout fails. Codex must rerun broad tests in the real checkout.

### 2.2 Observed files and systems

Observed key files/directories:

```text
AGENTS.md
GOAL.md
PLAN.md
compiler/features.go
docs/generated/manifest.json
docs/spec/current_supported_surface.md
docs/spec/surface_morph.md
docs/user/surface_guide.md
docs/user/examples_index.md
docs/release/surface_v1_release_contract.md
docs/release/surface_v1_release_audit.md
docs/plans/2026-05-22-full-platform-ui-runtime.md
lib/core/surface.tetra
lib/core/block.tetra
lib/core/morph.tetra
lib/core/widgets.tetra
tools/cmd/validate-surface-block-report
tools/cmd/validate-surface-morph-report
tools/cmd/validate-surface-release-state
tools/cmd/validate-surface-runtime
tools/validators/surface
scripts/release/surface
examples/surface_*.tetra
examples/core_*_smoke.tetra
.github/workflows/ci.yml
.github/workflows/full-platform-ui-runtime.yml
.github/workflows/release-packages.yml
```

### 2.3 What is proven now

| Area | Current truth | Evidence class | Notes |
|---|---|---|---|
| Surface v1 core | Current bounded `surface-v1-linux-web` support exists | CODE_PROVEN / VALIDATOR_PROVEN / RELEASE_GATE_PROVEN | Pure-Tetra UI over tiny Surface Host ABI; scoped to headless, linux-x64 real-window, wasm32-web browser-canvas. |
| Linux-x64 Surface host | Current Surface v1 target | CODE_PROVEN / RELEASE_GATE_PROVEN | Wayland shm RGBA real-window evidence is represented in feature registry and release gates. |
| wasm32-web Surface | Current browser-canvas Surface target | CODE_PROVEN / RELEASE_GATE_PROVEN | Compiler-owned browser boot and canvas RGBA evidence; not DOM UI / React / user JS app logic. |
| Headless Surface | Current evidence target | VALIDATOR_PROVEN | Deterministic runtime/report target, not an end-user platform. |
| Existing widgets | Compatibility/current toolkit subset | CODE_PROVEN / TEST_PROVEN | `Text`, `Label`, `StatusText`, `Button`, `TextBox`, `Checkbox`, `Row`, `Column`, `Panel`, `Stack`, `Scroll`, `Spacer`. |
| Block System | Implemented experimental track | CODE_PROVEN / VALIDATOR_PROVEN / RELEASE_GATE_PROVEN | `lib.core.block`, examples, block-system gate, target evidence. Still not production support. |
| Morph Capsule | Experimental authoring evidence layer | CODE_PROVEN / VALIDATOR_PROVEN | `lib.core.morph`, `surface_morph_command_palette`, headless morph gate. Not production support. |
| Surface validators | Strong fake-claim rejection exists | VALIDATOR_PROVEN | Validators reject unsupported target claims, DOM/user JS/platform widget claims, stale/malformed reports, fake Block/Morph claims. |
| CI integration | Surface release readiness includes Morph gate | CI_PROVEN_BY_CONFIG | `.github/workflows/ci.yml` has Surface Morph gate and Surface release gates. |
| Full-platform UI runtime workflow | Target-host smoke exists for Windows/macOS UI runtime | CI_PROVEN_BY_CONFIG | Does not override current docs that Surface v1 Windows/macOS production is unsupported. |

### 2.4 What is experimental now

| Area | Status | Why it is not final production |
|---|---|---|
| `ui.surface-block-system` | Experimental | Feature registry says no production Block claim and no Electron/React/DOM/CSS/runtime replacement claim. |
| `ui.surface-morph-capsule` | Experimental | Docs say Morph is not Surface v1 production support; current gate is deterministic headless Morph evidence. |
| Block-only polished scenes | Experimental evidence | Shows expressiveness, but not full app platform: no app shell, menus, packaging, devtools, platform accessibility parity. |
| Component tree/toolkit examples | Experimental or compatibility | Useful for migration, not final architecture. |
| Web UI DOM smoke artifacts | Separate/legacy web UI evidence | Must not be confused with Surface browser-canvas app UI. |
| Windows/macOS target-host UI runtime smoke | Early host evidence | Current supported-surface docs still mark Surface hosts unsupported for production. |

### 2.5 What is missing for Electron-competitor production

Missing or insufficient:

```text
stable Morph/style graph schema
stable renderer/compositor contract
full app-shell host ABI: windows, menus, dialogs, tray, notifications, cursors
production Windows Surface host
production macOS Surface host
app packaging/signing/notarization/update story
hot reload / fast rebuild loop
UI inspector / accessibility inspector / performance profiler
visual regression infrastructure with golden baselines
text shaping beyond deterministic fallback measurement
rich text/editing story
complete IME/composition per target
screen-reader target-host evidence per supported desktop target
security sandbox/permissions for Surface apps
IPC/process model comparable to Electron main/renderer separation
crash recovery and structured error reporting
internationalization/localization hooks
competitive performance and memory gates against realistic Electron baselines
claim governance for Electron/React/CSS replacement wording
```

### 2.6 Facts unavailable in this dump

- Clean worktree status of the real checkout.
- Actual final reports under `reports/` because `reports/` is ignored/excluded from the project dump.
- Graphify report, because `graphify-out/GRAPH_REPORT.md` was not in the dump.
- Live target-host Windows/macOS/Linux/Web execution results.
- Real packaged app artifacts, signatures, notarization, or update channels.

---

## 3. Final Claim Definition

### 3.1 What “replaces Electron/React/CSS” means here

Allowed final claim, only after gates pass:

```text
For apps within the supported Surface target matrix, Tetra Surface can replace
Electron/React/CSS as the production app UI platform: developers write Tetra
Block/Morph UI instead of React components/CSS; Surface owns rendering,
layout, style resolution, events, accessibility metadata, app shell integration,
packaging evidence, and performance budgets; no Electron/Chromium/DOM/CSS
runtime/user-JS app logic is required for production UI.
```

This does **not** mean:

```text
embedded arbitrary web pages
browser compatibility with all CSS/DOM APIs
React package ecosystem compatibility
Chromium DevTools parity
native platform widget wrapper
full screen-reader parity without target-host evidence
all-platform support without Windows/macOS/Linux/Web release gates
```

### 3.2 Minimum supported target matrix

Recommended phased matrix:

| Tier | Target matrix | Claim allowed | Required evidence |
|---|---|---|---|
| `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` | `headless`, `linux-x64`, `wasm32-web` | Electron-competitor for explicitly scoped Linux desktop + web-canvas apps | Clean same-commit release gates, app shell on Linux, browser-canvas web output, visual/a11y/perf/security evidence. |
| `BETA_TARGET_HOST_WINDOWS` | `windows-x64` | Beta target-host Surface app shell | Target-host real-window/event/input/accessibility/packaging smoke; no production claim. |
| `BETA_TARGET_HOST_MACOS` | `macos-x64` | Beta target-host Surface app shell | Target-host real-window/event/input/accessibility/signing/notarization smoke; no production claim. |
| `PROD_STABLE_CROSS_DESKTOP` | `linux-x64`, `windows-x64`, `macos-x64`, `headless` | Production desktop Electron replacement in supported desktop matrix | Same standard as Linux, plus target-host CI/release evidence on Windows/macOS. |
| `PROD_STABLE_CROSS_DESKTOP_WEB` | desktop matrix + `wasm32-web` | Production app UI platform across desktop and web-canvas | All target gates pass; web story clearly separated from DOM framework. |

### 3.3 Readiness names

Use only these:

```text
PROD_STABLE_SCOPED
BETA_TARGET_HOST
EXPERIMENTAL
UNSUPPORTED
NONCLAIM
```

Meaning:

- `PROD_STABLE_SCOPED`: clean same-commit target-host + headless gates, docs, validators, perf/a11y/security/report evidence all pass.
- `BETA_TARGET_HOST`: real target-host smoke exists but gaps remain; no production replacement claim.
- `EXPERIMENTAL`: code/report can demonstrate a slice; no app platform claim.
- `UNSUPPORTED`: feature/target is intentionally rejected or unavailable.
- `NONCLAIM`: not asserted publicly until evidence changes.

### 3.4 Prohibited claims until proven

Forbidden wording until specific gates exist:

```text
full Electron replacement
full React replacement
full CSS replacement
all desktop platforms supported
cross-platform production UI
full screen-reader support
full rich text editor
GPU production renderer
native platform widgets
Chromium replacement
DOM compatibility
zero memory overhead
faster than Electron
official benchmark superiority
```

Allowed scoped wording after final gates:

```text
Tetra Surface is production-stable for [exact target matrix] as a Tetra-owned
Block/Morph UI platform with no Electron/React/CSS runtime dependency.
```

---

## 4. Electron/React/CSS Parity Matrix

| Capability area | Electron/React/CSS provides | Current Surface/Morph | Required Surface for final claim | Current tier |
|---|---|---|---|---|
| Desktop app shell | windows, lifecycle, menus, tray, dialogs, notifications | Surface host window basics on linux-x64; Windows/macOS unsupported for Surface v1 | Stable cross-platform host ABI and target-host gates for shell services | PARTIAL / MISSING |
| Renderer/compositor | Chromium compositor/GPU/text/image stack | Software RGBA Surface renderer, Block paint layers, no GPU production claim | Hardened software renderer + optional gated GPU/compositor path | PARTIAL |
| Layout | CSS flex/grid/positioning | Block layout modes: row/column/grid/dock/overlay/scroll APIs exist | Stable constraint engine, responsive/density/DPI, visual regressions | PARTIAL |
| Style system | CSS cascade, variables, classes, pseudo-states | Morph tokens/materials/lenses experimental | Stable Morph graph replacing CSS runtime without cascade chaos | EXPERIMENTAL |
| Authoring model | React components/hooks/state | Surface widgets current; Block/Morph examples experimental | Recipes/affordances/state model without component zoo or React runtime | EXPERIMENTAL |
| State/events | React state/events, DOM events | Surface events, Block event/focus APIs, text input baseline | Stable app state/commands/navigation/focus/async/undo model | PARTIAL |
| Text/input | Browser text shaping/editing/IME/clipboard | TextBuffer, caret/selection, clipboard/IME baseline | HarfBuzz-class shaping or equivalent, bidi, IME per target, editing suite | PARTIAL |
| Accessibility | Browser accessibility tree/platform bridge | scoped metadata + Linux/web evidence, no full screen-reader claim | Screen-reader target-host evidence and inspectors per production target | PARTIAL |
| Assets/fonts/images | browser decoders/fonts/SVG/caching | Block asset refs/cache diagnostics; not full decode pipeline | fonts/icons/SVG/raster decode/cache/hash/security gates | PARTIAL |
| Animation | CSS/Web Animations/RAF | Motion presets deterministic headless | frame timing, compositor scheduling, reduced motion, visual gates | PARTIAL |
| Devtools | Chromium DevTools, React devtools | mostly missing | Surface inspector, tree/style/layout/a11y/perf views, screenshots | MISSING |
| Hot reload | bundlers/dev server | missing or not Surface-specific | fast rebuild/reload loop with state preservation rules | MISSING |
| Packaging | electron-builder/forge, signing/updater | Tetra release packaging for CLI exists | app packaging/signing/notarization/updates per target | MISSING |
| Security | Chromium sandbox plus Electron risks | Tetra capability/effects exist, Surface-specific app sandbox not final | permissions, asset sandbox, IPC hardening, supply chain rules | PARTIAL |
| IPC/process | main/renderer split | Actor/task runtime exists, not Surface app platform IPC | stable app/service/process model with crash isolation | PARTIAL/MISSING |
| Performance/memory | heavy but optimized Chromium | lower-overhead intent; Block memory budget evidence exists | measured startup/RSS/frame/binary/power budgets vs Electron baselines | PARTIAL |
| Ecosystem | npm/React/CSS ecosystem | Tetra stdlib + examples | templates, recipes, docs, migration tools, package/capsule story | MISSING/PARTIAL |
| Web target | Chromium/web app | wasm32-web browser-canvas current; DOM UI nonclaim | first-class web-canvas output with accessibility and packaging story | PARTIAL |

---

## 5. Recommended Architecture

### 5.1 Final Surface stack

```text
Tetra app source
  ↓
Surface App Model
  ├─ state / commands / async / navigation / undo
  ├─ permissions / capabilities
  └─ app shell declarations
  ↓
Block Authoring Graph
  ├─ raw Block primitive
  ├─ stable Block ABI / scene contract
  └─ event/focus/accessibility identity graph
  ↓
Stable Morph Graph
  ├─ typed tokens
  ├─ materials
  ├─ typography
  ├─ assets/icons/fonts
  ├─ state lenses
  ├─ motion presets
  ├─ affordances
  └─ recipes that expand to Block only
  ↓
Resolved Scene
  ├─ resolved layout
  ├─ resolved style/paint layers
  ├─ resolved text runs/glyph runs
  ├─ resolved images/icons
  ├─ resolved event routes
  ├─ resolved accessibility tree
  └─ resolved motion timelines
  ↓
Renderer / Compositor
  ├─ software RGBA baseline
  ├─ optional GPU/compositor backend behind gates
  ├─ frame scheduler
  ├─ visual regression hooks
  └─ performance counters
  ↓
Surface Host ABI
  ├─ window/frame/input/clipboard/IME
  ├─ accessibility platform bridge
  ├─ menus/dialogs/tray/notifications/cursors/drag-drop
  └─ app lifecycle / packaging hooks
  ↓
Targets
  ├─ headless
  ├─ linux-x64
  ├─ wasm32-web
  ├─ windows-x64
  └─ macos-x64
  ↓
Validators / Reports / Release Gates
```

### 5.2 Stable vs experimental vs internal

| Layer | Final status target | Rule |
|---|---|---|
| `Block` primitive | Stable public | Core UI primitive. No `Button/Card/TextField/Modal` core promotion. |
| Block ABI / scene report | Stable public contract | Versioned and backward-compatible within major release. |
| Morph token/style graph | Stable public | Replaces CSS runtime for supported Surface apps. |
| Morph recipes | Stable recipe API | Recipes output Blocks; recipe library can evolve by version. |
| Renderer software backend | Stable internal/public report | Internal implementation, public evidence/report contract. |
| GPU/compositor backend | Experimental until gated | Cannot be mentioned in production claims until target-host evidence. |
| Platform host ABI | Stable public/internal boundary | Stable enough for adapters; target-specific implementation can vary. |
| Devtools inspector | Stable developer tool | Not required at runtime for shipped apps. |
| Validators/reports | Stable release boundary | Claims depend on validators, not docs. |
| Compatibility widgets | Supported compatibility | Not final architecture; migration path to recipes. |

### 5.3 Architectural invariants

```text
Block is the primitive.
Morph is the stable style/authoring graph.
Recipes are not components.
Platform adapters provide host services, not UI widgets.
Renderer is Tetra-owned.
CSS/React/Electron are migration targets, not runtime dependencies.
Every production claim has a report, validator, artifact hash, and same-commit clean gate.
```

---

## 6. Architectural Alternatives Considered

### Alternative A — Copy Electron

Use Chromium shell, JS/DOM renderer, Node-like main process, web app UI.

Pros:

- fastest path to feature parity;
- existing mental model;
- mature renderer/accessibility/devtools.

Cons:

- violates Tetra goal;
- inherits Electron memory/runtime/security costs;
- UI truth lives in DOM/CSS/JS, not Tetra;
- no reason for Tetra Surface to exist.

Verdict: rejected.

### Alternative B — Copy React/CSS without Electron

Build a React-style component runtime and CSS-like cascade over a Tetra renderer.

Pros:

- familiar developer model;
- easier migration from frontend apps.

Cons:

- creates component zoo;
- recreates CSS specificity/global conflict problems;
- risks hidden virtual-DOM-like runtime;
- undermines Block-first model.

Verdict: rejected as primary architecture. Limited import/migration tooling is acceptable later.

### Alternative C — Native widget wrappers

Map Tetra UI to GTK/Cocoa/WinUI/Qt/native controls.

Pros:

- native accessibility and platform behavior;
- less custom rendering.

Cons:

- violates user-facing widget-layer constraint;
- cross-platform visual/runtime drift;
- app authoring becomes platform abstraction over native widgets;
- hard to provide deterministic headless/block visual evidence.

Verdict: rejected for user-facing UI. Platform APIs are allowed only as adapters for window/input/accessibility/app-shell services.

### Alternative D — Raw Block only

Expose Block with manual properties; developers build everything themselves.

Pros:

- pure architecture;
- easy to validate;
- no recipe bloat.

Cons:

- too much boilerplate;
- apps will visually drift;
- not competitive with Electron/React developer speed.

Verdict: good primitive, insufficient platform.

### Alternative E — Block/Morph-first Tetra-owned stack

Keep Block primitive, promote Morph into stable typed style/authoring graph, harden renderer/host/app-shell/devtools/security/packaging.

Pros:

- preserves Tetra architecture;
- avoids CSS/React/Electron runtime baggage;
- gives validators concrete artifacts;
- supports ergonomic authoring without component zoo;
- can compete in supported scope.

Cons:

- highest engineering burden;
- platform host work is nontrivial;
- text/accessibility/devtools require serious investment.

Verdict: recommended.

---

## 7. Implementation Packets

Each packet must be independently reviewable. Codex must not skip packet validation. If a packet cannot be completed, it must mark the dependent claims as false.

### SURFACE-PROD-P00 — Baseline truth audit and repo-state capture

Goal: establish non-fake starting truth.

Files/directories likely affected:

```text
docs/plans/2026-06-10-surface-electron-competitor-production-plan.md
reports/surface-prod/P00-baseline/
GOAL.md / PLAN.md only if this becomes active goal
```

Implementation notes:

- Re-read `AGENTS.md`, `GOAL.md`, current plan, and Graphify if available.
- Record `git rev-parse HEAD`, `git status --short`, target OS/arch, Go version.
- Discover all Surface/Morph/Block/host/devtools/packaging files.
- Classify each capability as `PROD_STABLE_SCOPED`, `BETA_TARGET_HOST`, `EXPERIMENTAL`, `UNSUPPORTED`, or `NONCLAIM`.

Tests:

```bash
rg -n "Surface|surface|Block|block|Morph|morph|renderer|canvas|Wayland|accessibility|IME|clipboard|widget|component|Electron|React|DOM|CSS" .
```

Validators/gates:

```bash
go test -buildvcs=false ./tools/cmd/validate-surface-morph-report ./tools/cmd/validate-surface-block-report ./tools/cmd/validate-surface-release-state -count=1
bash -n scripts/release/surface/*.sh
```

Acceptance criteria:

- Baseline report exists.
- All current claims mapped to files/gates.
- No final Electron replacement claim appears in baseline.

Fake-claim rejection cases:

- current repo marked Electron replacement;
- dirty checkout accepted as final;
- docs used as sole proof.

Dependencies: none.

Risk: stale dump or missing Graphify. Mitigation: rerun in real checkout.

### SURFACE-PROD-P01 — Claim taxonomy and overclaim validator

Goal: make claim tiers machine-enforced.

Files/directories likely affected:

```text
tools/cmd/validate-surface-prod-claim/
tools/validators/surfaceprod/
docs/spec/current_supported_surface.md
docs/release/surface_v1_release_contract.md
docs/generated/manifest.json
scripts/release/surface/prod-claim-gate.sh
```

Implementation notes:

- Add schema `tetra.surface.prod-claim.v1`.
- Encode allowed claim tiers.
- Require target matrix, git head, dirty state, evidence artifacts, nonclaims.
- Reject Electron/React/CSS replacement wording unless tier and gates match.

Tests:

```bash
go test -buildvcs=false ./tools/cmd/validate-surface-prod-claim ./tools/validators/surfaceprod -count=1
```

Validators/gates:

- negative fixtures: fake Electron replacement, fake cross-platform support, fake GPU, fake full screen-reader, missing target-host evidence.

Acceptance criteria:

- Claim validator rejects every forbidden final claim when evidence is missing.
- Release docs cannot pass if they overclaim.

Fake-claim rejection cases:

```text
"Surface replaces Electron" with only Morph headless evidence
"Windows production" without Windows target-host report
"CSS parity" with Morph only
"full accessibility" without screen-reader evidence
```

Dependencies: P00.

Risk: validator becomes wording-only. Mitigation: require artifact graph and target report references.

### SURFACE-PROD-P02 — Final Surface production scope spec

Goal: write the contract that defines the mountain.

Files/directories likely affected:

```text
docs/spec/surface_production_platform.md
docs/spec/current_supported_surface.md
docs/spec/surface_morph.md
compiler/features.go
docs/generated/manifest.json
```

Implementation notes:

- Define `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`.
- Keep Morph experimental until later packets promote it.
- Separate `current`, `experimental`, `unsupported`, and `future`.
- Add exact nonclaims.

Tests:

```bash
go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Validators/gates:

- overclaim scanner in P01.

Acceptance criteria:

- User-facing docs define what Electron replacement means and does not mean.

Fake-claim rejection cases:

- docs say “full Electron replacement” without target matrix.

Dependencies: P01.

Risk: spec too vague. Mitigation: every claim has required report schema.

### SURFACE-PROD-P03 — Stable Morph/style graph promotion plan and schema

Goal: promote Morph from experimental headless capsule to stable style graph candidate.

Files/directories likely affected:

```text
lib/core/morph.tetra
docs/spec/surface_morph.md
tools/cmd/validate-surface-morph-report
tools/validators/surface/report.go
examples/surface_morph_*.tetra
```

Implementation notes:

- Freeze Morph vocabulary: tokens, materials, affordances, recipes, state lenses, motion presets.
- Add versioned namespace/capsule import rules.
- Make override order explicit.
- Add diagnostics for token alias cycles, duplicate recipes, raw literals, unsupported CSS cascade imports.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*Morph|SurfaceBlock.*Morph' -count=1
go test -buildvcs=false ./tools/cmd/validate-surface-morph-report -count=1
```

Validators/gates:

```bash
bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-morph-gate
```

Acceptance criteria:

- Morph reports cover token graph, recipes, expansion into Blocks, accessibility projection, memory budget.
- Morph still does not claim production until P34/P35.

Fake-claim rejection cases:

- recipe outputs platform widget;
- `Button/Card/TextField/Modal` as core primitive;
- CSS cascade enabled;
- DOM/React/Electron runtime required.

Dependencies: P01, P02.

Risk: Morph remains symbolic metadata only. Mitigation: require resolved Block graph and visual frame checksums.

### SURFACE-PROD-P04 — Block ABI and renderer contract freeze

Goal: make Block scene output stable enough for a production renderer.

Files/directories likely affected:

```text
lib/core/block.tetra
lib/core/surface.tetra
compiler/features.go
tools/cmd/validate-surface-block-report
tools/validators/surface/report.go
docs/spec/surface_block_system.md
```

Implementation notes:

- Define stable `Block`, `BlockTree`, `BlockProps`, `ResolvedBlock`, `ResolvedScene` report fields.
- Freeze parent/child order, draw order, hit-test order, focus order, accessibility order.
- Add schema version and compatibility rules.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlock.*Tree|SurfaceBlock.*API|SurfaceBlock.*System' -count=1
go test -buildvcs=false ./tools/cmd/validate-surface-block-report -count=1
```

Validators/gates:

```bash
bash scripts/release/surface/block-system-gate.sh --report-dir reports/surface-block-gate
```

Acceptance criteria:

- Block ABI/schema can be validated independently of docs.
- Renderer accepts only validated resolved scenes.

Fake-claim rejection cases:

- missing Block tree;
- manual undocumented structural mutation;
- component/widget tree pretending to be Block scene.

Dependencies: P00.

Risk: public API freezes too early. Mitigation: use `v1alpha` until all renderer/app-shell packets pass.

### SURFACE-PROD-P05 — Renderer scene graph and paint command contract

Goal: build a deterministic bridge from Block/Morph to paint commands.

Files/directories likely affected:

```text
lib/core/draw.tetra
lib/core/block.tetra
compiler/internal/surface*
tools/validators/surface/report.go
scripts/release/surface/*block-system-smoke.sh
```

Implementation notes:

- Define resolved paint command schema: fill, gradient, border, radius, shadow, outline, image, text, clip, transform.
- Record paint command order and checksums.
- Keep blur/backdrop unsupported unless implemented and gated.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlockPaint|SurfaceBlockSystem' -count=1
go test -buildvcs=false ./tools/... -run 'Surface.*Paint|Block.*Paint' -count=1
```

Validators/gates:

- block report validator requires paint command evidence.

Acceptance criteria:

- same scene yields deterministic command sequence on headless.
- unsupported paint features fail loudly.

Fake-claim rejection cases:

- `backdropBlur` claim with no backend support;
- screenshot-only proof;
- command count missing.

Dependencies: P04.

Risk: visual quality remains primitive. Mitigation: connect to P06/P08/P21.

### SURFACE-PROD-P06 — Software renderer production hardening

Goal: make software RGBA renderer reliable enough for scoped production.

Files/directories likely affected:

```text
compiler/internal/surfacert/
compiler/internal/runtimeabi/
lib/core/draw.tetra
scripts/release/surface/*headless*smoke.sh
scripts/release/surface/*linux-x64*smoke.sh
scripts/release/surface/*wasm32-web*smoke.sh
```

Implementation notes:

- Deterministic raster for headless.
- Consistent alpha blending and clipping.
- Pixel checksum and golden image export.
- Stable resize/scale/DPI behavior.

Tests:

```bash
go test -buildvcs=false ./compiler/... ./tools/... -run 'Surface|Renderer|Draw|RGBA|Frame|Checksum' -count=1
```

Validators/gates:

```bash
bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1
```

Acceptance criteria:

- headless/linux/web frame checksums and visual feature summaries pass.
- no use-after-present or frame alias violation.

Fake-claim rejection cases:

- metadata-only frame;
- unchanged checksum after draw;
- Node-only browser promotion.

Dependencies: P05.

Risk: no anti-aliased text parity. Mitigation: P08.

### SURFACE-PROD-P07 — GPU/compositor path decision gate

Goal: decide whether GPU is required for Electron-competitive claims.

Files/directories likely affected:

```text
docs/spec/surface_renderer_backend.md
compiler/features.go
tools/cmd/validate-surface-renderer-report/
```

Implementation notes:

- Do not implement GPU just to claim it.
- Define required capabilities: layer compositing, transforms, clipping, texture atlas, vsync/frame timing.
- Keep GPU `EXPERIMENTAL` unless target-host gates pass.

Tests:

```bash
go test -buildvcs=false ./tools/cmd/validate-surface-renderer-report -count=1
```

Validators/gates:

- reject GPU production claim without target-host backend reports.

Acceptance criteria:

- Clear go/no-go for software-only `PROD_STABLE_SCOPED`.
- GPU claim forbidden until implementation exists.

Fake-claim rejection cases:

- docs say GPU renderer production while backend absent.

Dependencies: P06.

Risk: overbuilding GPU delays platform. Mitigation: software baseline remains production path.

### SURFACE-PROD-P08 — Text shaping and glyph pipeline

Goal: replace toy text drawing with production-grade text pipeline.

Files/directories likely affected:

```text
lib/core/text.tetra
lib/core/block.tetra
lib/core/draw.tetra
compiler/internal/surfacert/
tools/validators/surface/report.go
```

Implementation notes:

- Add font loading/fallback report.
- Add glyph run model and cache budget.
- Add Unicode scalar/cluster boundaries; define shaping scope.
- Decide integration with system libraries or embedded shaping engine; no platform widget text controls.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*Text|String|IME|Clipboard' -count=1
go test -buildvcs=false ./tools/... -run 'Surface.*Text|Glyph|Font' -count=1
```

Validators/gates:

- text report must include font/glyph cache bytes, fallback chain, shaping scope.

Acceptance criteria:

- clear baseline: Latin, UTF-8, fallback, wrap, ellipsis, selection metrics.
- nonclaims for unsupported scripts until shaping evidence exists.

Fake-claim rejection cases:

- full Unicode editor semantics claimed without tests;
- missing font fallback diagnostics;
- unbounded glyph cache.

Dependencies: P06.

Risk: text complexity explodes. Mitigation: tier shaping scope.

### SURFACE-PROD-P09 — Text editing, selections, IME, clipboard production path

Goal: reach app-grade input forms and editor basics.

Files/directories likely affected:

```text
lib/core/text.tetra
lib/core/surface.tetra
lib/core/block.tetra
scripts/release/surface/*text-input*smoke.sh
```

Implementation notes:

- Define editable text Block behavior.
- Per-target IME/composition traces.
- Clipboard read/write ownership/copy-safe boundaries.
- Selection/caret movement, undo unit boundaries, validation diagnostics.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*TextInput|Clipboard|Composition|Editable|SafeView' -count=1
```

Validators/gates:

- release text-input reports for headless/linux/web.

Acceptance criteria:

- forms and command palette search are production-safe.
- rich text remains nonclaim unless implemented later.

Fake-claim rejection cases:

- IME claimed but target report lacks composition events;
- borrowed text buffer crosses host boundary.

Dependencies: P08.

Risk: input bugs are product-killing. Mitigation: fuzz editing operations and run target-host smokes.

### SURFACE-PROD-P10 — Layout engine hardening

Goal: production responsive layout without CSS runtime.

Files/directories likely affected:

```text
lib/core/block.tetra
compiler/tests/semantics/surface_stdlib_test.go
tools/validators/surface/report.go
```

Implementation notes:

- Harden row, column, stack, grid, dock, absolute, overlay, scroll.
- Add min/max/fit/fill constraints, DPI/density, overflow/clip rules.
- Add invalidation and cache budget.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlockLayout|Surface.*Resize|Layout' -count=1
```

Validators/gates:

- block report requires layout mode evidence and rejects CSS flexbox parity claim.

Acceptance criteria:

- app shell, settings forms, dashboards, editor shells are layout-stable under resize.

Fake-claim rejection cases:

- CSS flexbox/grid parity claim;
- overflow hidden by accident;
- layout cache unbounded.

Dependencies: P04.

Risk: layout bugs become visual regressions. Mitigation: P25 visual tests.

### SURFACE-PROD-P11 — Stable Morph style/token graph as CSS replacement

Goal: replace CSS runtime with a deterministic typed style graph.

Files/directories likely affected:

```text
lib/core/morph.tetra
lib/core/block.tetra
docs/spec/surface_morph.md
tools/cmd/validate-surface-morph-report
```

Implementation notes:

- Token categories: color, spacing, radius, type, motion, elevation, z, density, assets.
- No global cascade; explicit imports only.
- Fixed override order.
- Public diagnostics for conflicts and missing tokens.

Tests:

```bash
go test -buildvcs=false ./tools/cmd/validate-surface-morph-report -run 'Token|Capsule|Recipe|Fake|Claim' -count=1
```

Validators/gates:

- reject token alias cycles, duplicate source of truth, unresolved fallback.

Acceptance criteria:

- Morph becomes candidate stable style graph.
- Developers can build polished UI without raw 80-field Blocks.

Fake-claim rejection cases:

- global style leak;
- specificity-like override ambiguity;
- raw CSS import as runtime dependency.

Dependencies: P03, P10.

Risk: Morph becomes CSS with different syntax. Mitigation: no selector engine/cascade.

### SURFACE-PROD-P12 — State, events, commands, and app model

Goal: replace React app-state/event ergonomics without React runtime.

Files/directories likely affected:

```text
lib/core/block.tetra
lib/core/surface.tetra
compiler/tests/semantics/ui_semantics_test.go
compiler/tests/semantics/surface_stdlib_test.go
```

Implementation notes:

- Define state stores, commands, event routing, async command policy.
- Include navigation, focus, shortcut scopes, error propagation, redraw scheduling.
- Integrate actor/task runtime only through safe app model boundaries.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'UI|SurfaceBlockEvents|SurfaceBlockState|Actor|Task' -count=1
```

Validators/gates:

- event trace in Surface production report.

Acceptance criteria:

- command palette, dashboard, settings, editor shell can run with real state/event flow.

Fake-claim rejection cases:

- event trace missing;
- disabled control dispatches;
- text input goes to unfocused Block.

Dependencies: P04, P09.

Risk: React-style component sprawl sneaks in. Mitigation: recipes output Blocks only.

### SURFACE-PROD-P13 — Focus, keyboard, shortcuts, navigation, undo/redo

Goal: production keyboard UX.

Files/directories likely affected:

```text
lib/core/block.tetra
lib/core/surface.tetra
examples/surface_morph_*.tetra
```

Implementation notes:

- Focus order, focus trap, roving focus, modal/overlay focus policy.
- Keyboard activation and shortcut scopes.
- Undo/redo model for text/forms/editor shell.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Focus|Keyboard|Shortcut|Undo|SurfaceBlockEvents' -count=1
```

Validators/gates:

- accessibility report requires focus/reading order.

Acceptance criteria:

- app can be keyboard-operated.
- command palette/search/settings forms pass keyboard script.

Fake-claim rejection cases:

- focusable element without accessible name;
- overlay allows focus leak;
- shortcut conflict not diagnosed.

Dependencies: P12, P20.

Risk: inaccessible apps. Mitigation: accessibility gate blocks.

### SURFACE-PROD-P14 — Recipe authoring library without component bloat

Goal: give React-like speed without React-like components.

Files/directories likely affected:

```text
lib/core/morph.tetra
lib/core/block.tetra
examples/surface_morph_*.tetra
docs/user/surface_guide.md
```

Implementation notes:

- Stable recipe families: action, field, toggle, command item, nav item, panel, dialog overlay, tabs, list, table-lite, status.
- Every recipe declares inputs/slots/state/a11y projection.
- Recipe expansion is recorded in reports.

Tests:

```bash
rg -n "widgets\.|Button\(|Card\(|TextField\(|Modal\(" examples/surface_morph_*.tetra && exit 1 || true
go test -buildvcs=false ./tools/cmd/validate-surface-morph-report -run 'Recipe|Forbidden|Expansion' -count=1
```

Validators/gates:

- Morph validator rejects core primitive promotion.

Acceptance criteria:

- production example suite uses recipes but resolves to Blocks only.

Fake-claim rejection cases:

- hidden app state in recipe;
- platform widget recipe output;
- unreported expansion.

Dependencies: P11, P12.

Risk: recipe zoo. Mitigation: public recipes limited and versioned.

### SURFACE-PROD-P15 — App shell host ABI

Goal: provide Electron-like shell capabilities without Electron.

Files/directories likely affected:

```text
lib/core/surface.tetra
compiler/internal/runtimeabi/
compiler/internal/surfacert/
docs/spec/surface_host_abi.md
```

Implementation notes:

Define host ABI for:

```text
window create/close/show/hide/focus
resize/min/max/fullscreen
title/icon/cursor
menus/context menus
dialogs/file pickers
notifications
tray/status item
clipboard
IME
DPI/scale
open URL/file
app lifecycle
```

Tests:

```bash
go test -buildvcs=false ./compiler/... ./tools/... -run 'SurfaceHost|Window|Menu|Dialog|Tray|Notification|Cursor|DPI' -count=1
```

Validators/gates:

- app-shell report schema `tetra.surface.app-shell.v1`.

Acceptance criteria:

- Linux app shell can prove minimum Electron-equivalent shell features.
- Unsupported host features return diagnostics, not silent no-op.

Fake-claim rejection cases:

- menu support claimed with no target-host action trace;
- notification support claimed with no host report.

Dependencies: P06, P12.

Risk: host ABI grows uncontrolled. Mitigation: versioned shell capabilities matrix.

### SURFACE-PROD-P16 — Linux-x64 production host adapter

Goal: make Linux the first real production desktop target.

Files/directories likely affected:

```text
compiler/internal/surfacert/linux*
scripts/release/surface/*linux-x64*
tools/cmd/validate-surface-linux-prod-report/
```

Implementation notes:

- Harden Wayland shm RGBA path or add X11 fallback only with evidence.
- Event pump, DPI, cursor, menus/dialogs/notifications if in scope.
- Accessibility bridge target-host validation.

Tests:

```bash
bash scripts/release/surface/surface-linux-x64-real-window-smoke.sh --report-dir reports/surface-prod/linux
bash scripts/release/surface/surface-linux-x64-real-window-block-system-smoke.sh --report-dir reports/surface-prod/linux-block
```

Validators/gates:

- Linux production host validator.

Acceptance criteria:

- Linux target can ship real app shell.

Fake-claim rejection cases:

- headless-only evidence promoted to Linux production;
- blocked display counted as pass.

Dependencies: P15.

Risk: CI runners lack Wayland. Mitigation: strict blocked state prevents release claim; use target-host runner.

### SURFACE-PROD-P17 — Windows target-host adapter

Goal: move Windows from unsupported/BETA to production only with real evidence.

Files/directories likely affected:

```text
compiler/internal/surfacert/windows*
tools/cmd/platform-ui-runtime-smoke
tools/cmd/validate-windows-ui-runtime
.github/workflows/full-platform-ui-runtime.yml
```

Implementation notes:

- Native window, input, clipboard, IME, DPI, menus/dialogs/notifications.
- DWM/bitmap/compositor path or equivalent.
- Accessibility bridge evidence.

Tests:

```bash
go run ./tools/cmd/platform-ui-runtime-smoke --target windows-x64 --report windows-ui-runtime.json
go run ./tools/cmd/validate-windows-ui-runtime --report windows-ui-runtime.json --expected-version "$version" --expected-git-head "$git_head"
```

Validators/gates:

- Windows Surface production validator separate from generic UI runtime.

Acceptance criteria:

- `BETA_TARGET_HOST_WINDOWS` first; production only after full DoD.

Fake-claim rejection cases:

- build-only Windows target counted as UI runtime;
- linux-host synthetic report counted as Windows target-host.

Dependencies: P15, P20, P26.

Risk: Windows host ABI complexity. Mitigation: beta tier until all target-host evidence exists.

### SURFACE-PROD-P18 — macOS target-host adapter

Goal: move macOS from unsupported/BETA to production only with real evidence.

Files/directories likely affected:

```text
compiler/internal/surfacert/macos*
tools/cmd/platform-ui-runtime-smoke
tools/cmd/validate-macos-ui-runtime
.github/workflows/full-platform-ui-runtime.yml
```

Implementation notes:

- Native window/input/DPI/menu bar/dialogs/notifications.
- Accessibility bridge evidence.
- Signing/notarization path in P26.

Tests:

```bash
go run ./tools/cmd/platform-ui-runtime-smoke --target macos-x64 --report macos-ui-runtime.json
go run ./tools/cmd/validate-macos-ui-runtime --report macos-ui-runtime.json --expected-version "$version" --expected-git-head "$git_head"
```

Validators/gates:

- macOS Surface production validator separate from generic UI runtime.

Acceptance criteria:

- `BETA_TARGET_HOST_MACOS` first; production only after full DoD.

Fake-claim rejection cases:

- non-notarized package treated as production distribution;
- no screen-reader bridge but full a11y claim.

Dependencies: P15, P20, P26.

Risk: macOS runner/env limitations. Mitigation: dedicated target-host CI.

### SURFACE-PROD-P19 — wasm32-web first-class browser-canvas target

Goal: keep web output first-class without turning Surface into DOM/React.

Files/directories likely affected:

```text
compiler/internal/backend/wasm*
scripts/release/surface/*wasm32-web*
tools/cmd/validate-surface-runtime
```

Implementation notes:

- Browser-canvas Surface boot remains compiler-owned.
- No user JS app logic.
- Accessibility snapshot/mirror evidence remains explicit.
- Separate legacy DOM UI smoke from Surface web-canvas claim.

Tests:

```bash
bash scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh --report-dir reports/surface-prod/wasm-web
bash scripts/release/surface/surface-wasm32-web-browser-canvas-block-system-smoke.sh --report-dir reports/surface-prod/wasm-web-block
```

Validators/gates:

- reject Node-only, DOM UI, metadata-only sidecar.

Acceptance criteria:

- same Block/Morph scenes run in web canvas with frame checksums and events.

Fake-claim rejection cases:

- DOM snapshot counted as Surface renderer;
- user JS command dispatch counted as Tetra app logic.

Dependencies: P06, P12.

Risk: web accessibility without DOM is hard. Mitigation: scoped snapshot/mirror until platform evidence exists.

### SURFACE-PROD-P20 — Accessibility production path per target

Goal: accessibility cannot be marketing text.

Files/directories likely affected:

```text
lib/core/accessibility.tetra
lib/core/block.tetra
lib/core/morph.tetra
compiler/internal/surfacert/*accessibility*
tools/validators/surface/report.go
```

Implementation notes:

- Stable a11y tree: role, name, description, value, state, relationships, actions, bounds, focus order, reading order.
- Target bridges: Linux first, Windows/macOS beta, web snapshot/mirror.
- Add screen-reader smoke protocol only when real AT integration exists.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*Accessibility|Accessibility' -count=1
go test -buildvcs=false ./tools/... -run 'Surface.*Accessibility|A11y' -count=1
```

Validators/gates:

- accessibility validator rejects metadata-only platform claim.

Acceptance criteria:

- Production target requires target-host bridge evidence.
- Full screen-reader claim requires screen-reader smoke.

Fake-claim rejection cases:

- focusable unnamed Block;
- ARIA/DOM evidence used for desktop bridge;
- full AT-SPI claim with no screen-reader validation.

Dependencies: P13, P16/P17/P18/P19.

Risk: accessibility becomes permanently partial. Mitigation: block production claim or scope it honestly.

### SURFACE-PROD-P21 — Assets, fonts, icons, image/vector pipeline

Goal: app-quality visuals with safe asset handling.

Files/directories likely affected:

```text
lib/core/block.tetra
lib/core/morph.tetra
compiler/internal/surfacert/assets*
tools/validators/surface/report.go
```

Implementation notes:

- Font embedding/loading/fallback.
- SVG/raster decode policy.
- Icon tinting, image scale/fit, atlas/cache.
- Asset manifest hashes and no network fetch in release tests.
- Untrusted image/font validation.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlockAssets|Asset|Font|Icon' -count=1
go test -buildvcs=false ./tools/... -run 'Surface.*Asset|Image|Font|Icon' -count=1
```

Validators/gates:

- asset report rejects missing hashes, network fetch, unbounded cache.

Acceptance criteria:

- production examples ship local asset manifest.

Fake-claim rejection cases:

- missing asset silently rendered;
- remote font used in release test;
- SVG parser accepts unsafe payload.

Dependencies: P08, P11.

Risk: image decode security. Mitigation: sandbox/limited codecs/fuzz.

### SURFACE-PROD-P22 — Animation, transitions, frame scheduling

Goal: Electron-class smoothness without CSS animations.

Files/directories likely affected:

```text
lib/core/block.tetra
lib/core/morph.tetra
compiler/internal/surfacert/frame*
tools/validators/surface/report.go
```

Implementation notes:

- Stable motion timeline and reduced-motion policy.
- Frame scheduler, invalidation, animation lifecycle.
- Transition properties: opacity, color, transform, maybe layout.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlockMotion|Motion|Frame' -count=1
go test -buildvcs=false ./tools/... -run 'Surface.*Motion|FrameTiming' -count=1
```

Validators/gates:

- motion report must include frame count, time source, reduced-motion evidence.

Acceptance criteria:

- deterministic headless motion frames and target-host smoke.

Fake-claim rejection cases:

- CSS animation parity claim;
- infinite hidden animation loop;
- missing reduced-motion path.

Dependencies: P06, P11.

Risk: performance regressions. Mitigation: P31 budget gates.

### SURFACE-PROD-P23 — Developer inspector

Goal: give Surface the minimum devtools needed to replace Electron development workflow.

Files/directories likely affected:

```text
cli/cmd/tetra/*surface*
tools/cmd/surface-inspect/
docs/user/surface_guide.md
```

Implementation notes:

- Inspect Block tree, Morph resolution, layout boxes, paint layers, events, focus, accessibility, performance counters.
- Export JSON snapshots and optionally a Tetra Surface inspector UI.

Tests:

```bash
go test -buildvcs=false ./cli/... ./tools/... -run 'Surface.*Inspect|Inspector|Snapshot' -count=1
```

Validators/gates:

- inspector snapshot schema validator.

Acceptance criteria:

- developer can diagnose why a Block looks/behaves wrong.

Fake-claim rejection cases:

- inspector shows docs-only tree;
- snapshot missing source locations.

Dependencies: P04, P11, P20, P31.

Risk: Devtools become huge. Mitigation: JSON-first MVP.

### SURFACE-PROD-P24 — Hot reload, templates, fast dev loop

Goal: compete with Electron/React DX.

Files/directories likely affected:

```text
cli/cmd/tetra/new*
cli/cmd/tetra/surface*
examples/templates/surface*
docs/user/surface_guide.md
```

Implementation notes:

- `tetra new surface-app`.
- `tetra surface dev` or equivalent fast rebuild/reload.
- State preservation rules optional and scoped.
- Error overlays as Surface inspector data, not DOM dev server.

Tests:

```bash
go test -buildvcs=false ./cli/... ./tools/... -run 'Surface.*Dev|Template|Reload|New' -count=1
```

Validators/gates:

- template smoke builds and runs.

Acceptance criteria:

- new app can be created, checked, run, inspected, packaged.

Fake-claim rejection cases:

- hot reload claim without end-to-end file change trace.

Dependencies: P23.

Risk: dev loop becomes flaky. Mitigation: deterministic reload smoke.

### SURFACE-PROD-P25 — Visual regression infrastructure

Goal: prevent pixel/UI regressions from hiding.

Files/directories likely affected:

```text
tools/cmd/surface-golden/
tools/cmd/validate-surface-visual-report/
testdata/surface-golden/
scripts/release/surface/visual-gate.sh
```

Implementation notes:

- Headless golden PNG/RGBA/checksum export.
- Per-target visual summary and tolerances.
- Store source scene, target, renderer version, fonts/assets hashes.

Tests:

```bash
go test -buildvcs=false ./tools/... -run 'Surface.*Golden|Visual|Screenshot|Checksum' -count=1
```

Validators/gates:

- visual gate fails on missing/changed baseline unless approved.

Acceptance criteria:

- command palette/dashboard/settings/editor/glass scenes have baselines.

Fake-claim rejection cases:

- screenshot only, no scene hash;
- golden updated without review marker.

Dependencies: P06, P21.

Risk: brittle golden tests. Mitigation: semantic visual reports + pixel thresholds.

### SURFACE-PROD-P26 — Packaging, signing, notarization, update story

Goal: app distribution, not only rendering pixels.

Files/directories likely affected:

```text
scripts/release/packages/
cli/cmd/tetra/package*
tools/cmd/validate-surface-package-report/
.github/workflows/release-packages.yml
```

Implementation notes:

- Package Surface app with assets, permissions, host adapter metadata.
- Linux AppImage/deb/tar or scoped package format.
- Windows installer/signing path.
- macOS bundle/sign/notarize path.
- Auto-update strategy as explicit separate tier.

Tests:

```bash
go test -buildvcs=false ./cli/... ./tools/... -run 'Package|Installer|SurfacePackage|Signing|Update' -count=1
```

Validators/gates:

- package report with file hashes, asset manifest, signature/notarization state.

Acceptance criteria:

- Linux production package can install/run.
- Windows/macOS remain beta/nonclaim until signing evidence.

Fake-claim rejection cases:

- unsigned macOS app called production;
- asset omitted from package;
- updater claim without channel/sig verification.

Dependencies: P15-P19, P21.

Risk: platform-specific release complexity. Mitigation: target-tier claims.

### SURFACE-PROD-P27 — Security and sandbox model

Goal: avoid Electron-class attack surfaces and define Surface-specific ones.

Files/directories likely affected:

```text
compiler/features.go
lib/core/surface.tetra
lib/core/filesystem.tetra
lib/core/networking.tetra
docs/spec/surface_security.md
tools/cmd/validate-surface-security-report/
```

Implementation notes:

- Permissions for filesystem/network/clipboard/window/open-url/notifications.
- Asset/font/image sandbox.
- IPC hardening.
- Capability audit for app shell.
- No remote code execution by default.

Tests:

```bash
go test -buildvcs=false ./compiler/... ./tools/... -run 'Surface.*Security|Permission|Capability|Sandbox|Asset' -count=1
```

Validators/gates:

- security report rejects unbounded/unpermissioned host calls.

Acceptance criteria:

- production app declares permissions and validators enforce them.

Fake-claim rejection cases:

- network/filesystem access without permission;
- untrusted SVG/font parsed unsafely;
- user JS introduced.

Dependencies: P15, P21, P28.

Risk: sandbox too weak or too restrictive. Mitigation: explicit permissions model and negative tests.

### SURFACE-PROD-P28 — IPC/process/app lifecycle model

Goal: cover what Electron main/renderer split normally solves.

Files/directories likely affected:

```text
compiler/internal/actorsrt/
compiler/internal/parallelrt/
lib/core/actor*.tetra
lib/core/surface.tetra
docs/spec/surface_app_model.md
```

Implementation notes:

- Define app main, UI isolate/thread, background tasks/services.
- Message passing with owned data only.
- Crash isolation strategy.
- No unsafe Surface handles across actor/task boundaries unless proven.

Tests:

```bash
go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -run 'Actor|Task|Surface.*Boundary|IPC|Lifecycle' -count=1
```

Validators/gates:

- report rejects Surface handle/event/frame crossing unsafe boundaries.

Acceptance criteria:

- background work can update UI safely.

Fake-claim rejection cases:

- Surface frame/event stored in actor message;
- background task mutates UI state without dispatcher.

Dependencies: P12, P27.

Risk: actor-runtime active goal may be in flux. Mitigation: isolate Surface requirements and run actor gates when touched.

### SURFACE-PROD-P29 — Crash recovery, diagnostics, error reporting

Goal: production app behavior when things fail.

Files/directories likely affected:

```text
cli/cmd/tetra/*
compiler/internal/surfacert/
tools/cmd/validate-surface-crash-report/
docs/user/surface_guide.md
```

Implementation notes:

- Structured error reports.
- Crash-safe app restart/recovery policy.
- Surface panic/error overlay for dev only.
- Production error hook without leaking secrets.

Tests:

```bash
go test -buildvcs=false ./cli/... ./tools/... ./compiler/... -run 'Surface.*Crash|Error|Recovery|Panic' -count=1
```

Validators/gates:

- crash report schema, no secrets, source locations.

Acceptance criteria:

- failing app produces useful diagnostics.

Fake-claim rejection cases:

- crash swallowed as pass;
- error report includes secrets.

Dependencies: P24, P27.

Risk: error handling hides real bugs. Mitigation: release gate distinguishes expected negative cases from crashes.

### SURFACE-PROD-P30 — Internationalization and localization

Goal: make production apps possible outside English-only demos.

Files/directories likely affected:

```text
lib/core/text.tetra
lib/core/i18n.tetra
docs/spec/surface_i18n.md
examples/surface_i18n_*.tetra
```

Implementation notes:

- Locale resources, string IDs, formatting hooks.
- Bidi/text shaping scope.
- Layout direction support if implemented.
- Translation asset packaging.

Tests:

```bash
go test -buildvcs=false ./compiler/... ./tools/... -run 'I18n|Localization|Bidi|Locale|Surface.*Text' -count=1
```

Validators/gates:

- localization manifest validator.

Acceptance criteria:

- basic localized Surface app builds/renders.

Fake-claim rejection cases:

- full bidi claim without shaping evidence;
- missing locale resources silently fallback.

Dependencies: P08, P21, P26.

Risk: Unicode/localization scope too broad. Mitigation: tiered support matrix.

### SURFACE-PROD-P31 — Performance and memory gates

Goal: compete with Electron without fake speed claims.

Files/directories likely affected:

```text
tools/cmd/surface-perf-smoke/
tools/cmd/validate-surface-perf-report/
scripts/release/surface/perf-gate.sh
reports/surface-prod/perf/
```

Implementation notes:

Measure:

```text
startup time
first frame time
steady frame time
peak RSS
frame allocations
layout/cache/glyph/asset bytes
binary size
CPU idle/power proxy
input latency
animation frame jitter
```

Benchmark against Electron fairly:

- same app shape;
- same OS/target;
- same cold/warm states;
- report hardware/environment;
- no “faster than Electron” claim unless statistically supported.

Tests:

```bash
go test -buildvcs=false ./tools/... -run 'Surface.*Perf|Memory|Budget|RSS|Frame' -count=1
```

Validators/gates:

- performance report rejects missing baselines, impossible numbers, unbounded cache.

Acceptance criteria:

- production budget thresholds defined and enforced.

Fake-claim rejection cases:

- zero memory overhead;
- fastest UI framework;
- no baseline environment.

Dependencies: P06, P08, P10, P21, P22.

Risk: perf gates flaky. Mitigation: strict environment capture and tolerance windows.

### SURFACE-PROD-P32 — Migration from widgets/component tree to Block/Morph recipes

Goal: avoid breaking current users while moving architecture forward.

Files/directories likely affected:

```text
lib/core/widgets.tetra
lib/core/component.tetra
lib/core/block.tetra
lib/core/morph.tetra
docs/user/surface_guide.md
docs/user/examples_index.md
examples/surface_migration_*.tetra
```

Implementation notes:

- Keep current widgets as compatibility layer.
- Add mappings to recipe/Block equivalents.
- Emit migration diagnostics where useful.
- Do not deprecate until production examples and gates cover replacement.

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*Widget|Surface.*Migration|Surface.*Toolkit' -count=1
```

Validators/gates:

- migration report proves compatibility and Block equivalence.

Acceptance criteria:

- existing Surface v1 examples still pass.
- new docs recommend Block/Morph for new production UI.

Fake-claim rejection cases:

- widgets declared core final architecture;
- breaking v1 examples without migration.

Dependencies: P14.

Risk: dual systems cause confusion. Mitigation: claim tiers and docs.

### SURFACE-PROD-P33 — Production example suite and ecosystem seed

Goal: prove realistic app shapes, not toy demos.

Files/directories likely affected:

```text
examples/surface_prod_*.tetra
docs/user/examples_index.md
docs/user/surface_guide.md
```

Implementation notes:

Create examples:

```text
command palette app
settings app
project dashboard
editor shell
file manager shell
multi-window notes app
system tray/status app
notification/dialog demo
localized form
accessibility-heavy form
```

Tests:

```bash
go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*Prod|Surface.*Example|SurfaceBlockExamples' -count=1
```

Validators/gates:

- example validator rejects widgets where Block/Morph is required and rejects fake claims.

Acceptance criteria:

- examples run on every supported target for the claim tier.

Fake-claim rejection cases:

- screenshots without executable examples;
- examples require React/Electron/DOM runtime.

Dependencies: P14-P26.

Risk: examples become curated illusions. Mitigation: validator checks events/state/a11y/perf, not only visuals.

### SURFACE-PROD-P34 — CI and release gate integration

Goal: make production readiness impossible to fake locally.

Files/directories likely affected:

```text
.github/workflows/ci.yml
.github/workflows/full-platform-ui-runtime.yml
.github/workflows/release-packages.yml
scripts/release/surface/prod-gate.sh
scripts/release/surface/release-gate.sh
```

Implementation notes:

- Add `surface-prod-gate.sh` aggregating Block/Morph/app-shell/visual/a11y/perf/package/security.
- No `continue-on-error` for production claim job.
- Artifact upload includes reports and hashes.
- Separate Linux/Web prod from Windows/macOS beta.

Tests:

```bash
bash -n scripts/release/surface/prod-gate.sh
bash scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final
```

Validators/gates:

- release-state validator must require Surface prod reports for prod claim.

Acceptance criteria:

- CI can produce or block the exact claim tier.

Fake-claim rejection cases:

- missing job but docs claim production;
- skipped target counted as pass;
- artifact hash manifest missing.

Dependencies: P01-P33.

Risk: gate too slow. Mitigation: split fast PR gates and release gates.

### SURFACE-PROD-P35 — Final same-commit audit and claim governance

Goal: declare readiness only if true.

Files/directories likely affected:

```text
docs/release/surface_prod_release_audit.md
reports/surface-prod/final/
tools/cmd/validate-surface-prod-audit/
```

Implementation notes:

- Audit every requirement.
- Include git head, clean status, command list, target reports, artifact hashes.
- Produce final verdict enum:

```text
PROD_STABLE_SCOPED
NEAR_READY_WITH_BLOCKERS
BETA_ONLY
EXPERIMENTAL_ONLY
FAIL
```

Tests:

```bash
go test -buildvcs=false ./tools/cmd/validate-surface-prod-audit -count=1
```

Validators/gates:

- audit validator rejects missing clean checkout or stale artifacts.

Acceptance criteria:

- final public claim is generated from audit result, not hand-written optimism.

Fake-claim rejection cases:

- dirty checkout promoted;
- report from different git head;
- missing unsupported-target nonclaims.

Dependencies: P34.

Risk: pressure to ship early. Mitigation: validator owns verdict.

### SURFACE-PROD-P36 — Electron comparison benchmark and public positioning

Goal: make the competitor claim precise and defensible.

Files/directories likely affected:

```text
docs/benchmarks/surface_vs_electron.md
tools/cmd/surface-electron-comparison/
tools/cmd/validate-surface-electron-comparison-report/
```

Implementation notes:

- Build equivalent app in Electron and Surface.
- Measure startup, RSS, first frame, input latency, idle CPU, package size.
- Keep React/CSS/Electron compatibility/migration narrative honest.

Tests:

```bash
go test -buildvcs=false ./tools/... -run 'ElectronComparison|Surface.*Benchmark' -count=1
```

Validators/gates:

- reject official benchmark claim, cherry-picked hardware, missing variance.

Acceptance criteria:

- public docs can say “competitive with Electron in supported scope” with evidence.

Fake-claim rejection cases:

- “faster than Electron” from one local smoke;
- unfair app shape;
- missing environment.

Dependencies: P31, P35.

Risk: benchmark politics. Mitigation: publish method, not hype.

---

## 8. Renderer And Compositor Plan

### 8.1 Software renderer baseline

Software RGBA remains the first production path. It must support:

```text
clear
rect / rounded rect
stroke / border
gradient
shadow approximation
outline / focus ring
clip / overflow
image blit / scale
icon tint
text glyph runs
opacity
transform for translate/scale where supported
```

Required evidence:

```text
frame width/height/stride
paint command count
paint feature flags
frame checksum before/after
changed pixel count or semantic visual summary
cache bytes
allocation count if available
```

### 8.2 GPU/compositor path

GPU is optional for initial scoped production. It becomes required only if performance gates show software cannot meet app-grade budgets.

GPU production claim requires:

```text
backend report schema
linux target-host GPU smoke
web compositor/canvas evidence
Windows/macOS target-host GPU evidence if claimed
fallback behavior
same scene equivalence against software baseline
```

Until then:

```text
GPU renderer = EXPERIMENTAL / NONCLAIM
```

### 8.3 Text/glyph pipeline

Production text requires:

```text
font manifest and hashes
font fallback chain
glyph cache budget and eviction
text shaping scope
text measurement consistency
wrap/ellipsis/alignment/baseline
selection/caret rectangles
IME composition spans
```

If full shaping is not implemented, docs must state exact script/language limitations.

### 8.4 Paint model

Paint is layered and deterministic:

```text
background fill
material layers
content layers
state overlay
focus ring / outline
motion transform
accessibility bounds projection
```

Unsupported features must be explicit diagnostics:

```text
blur/backdropBlur
CSS filters
arbitrary shaders
native effects
```

### 8.5 Image/vector/icon pipeline

Required:

```text
local asset manifest
asset hashes
image dimensions
decode result
safe format whitelist
icon tinting
atlas/cache limits
missing asset fallback diagnostic
```

Network assets are forbidden in release tests unless the security model explicitly scopes and validates them.

### 8.6 Frame scheduling

Required:

```text
request_redraw
frame begin/present lifetime safety
animation time source
input-to-frame trace
frame budget report
idle behavior
reduced motion
```

### 8.7 Visual regression strategy

Use both semantic and pixel evidence:

```text
scene hash
resolved Block tree hash
resolved Morph graph hash
paint command hash
font/asset hash
frame checksum
optional golden PNG/RGBA
approved baseline metadata
```

### 8.8 Performance budgets

Initial proposed budgets, to be calibrated by measurement:

| Metric | Initial scoped budget | Notes |
|---|---:|---|
| first app window | ≤ 500 ms on reference Linux host | Must record host. |
| steady frame | ≤ 16.7 ms for simple app, ≤ 33 ms fallback | 60 FPS target where feasible. |
| idle CPU | near-zero when no animation/input | Must avoid hidden loops. |
| peak RSS simple app | ≤ 80 MB initial target | Not a public claim until measured. |
| package size simple app | define per target | Compare against Electron fairly. |
| input-to-frame | ≤ 50 ms | Headless + target-host. |
| cache growth | bounded | glyph/asset/layout caches reported. |

---

## 9. Layout And Style Plan

### 9.1 Block layout model

Required stable modes:

```text
row
column
stack
grid
dock
absolute
overlay
scroll
```

Required constraints:

```text
fixed
fit
fill
min/max width/height
padding
margin
gap
align
justify
z-index
clip/overflow
DPI/density scaling
```

### 9.2 Stable Morph/style/token graph

Morph replaces CSS runtime with:

```text
typed tokens
materials
typography roles
state lenses
motion presets
affordances
recipes
explicit imports
fixed override order
```

### 9.3 CSS replacement boundaries

Surface should replace CSS for supported app UI by providing:

```text
layout rules
paint/materials
typography
state styling
responsive constraints
theme variants
motion presets
```

It should **not** provide:

```text
CSS cascade
selector specificity wars
arbitrary global selectors
runtime DOM styling
browser CSS compatibility
```

### 9.4 Conflict/override rules

Mandatory order:

```text
1. default token graph
2. imported capsule tokens
3. material defaults
4. recipe defaults
5. recipe variant
6. state lens
7. local explicit override
8. accessibility safety override
```

A validator must reject ambiguous sources and duplicate token authority.

### 9.5 Theming, variants, responsive constraints, density, DPI

Production Morph must include:

```text
light/dark/high-contrast themes
size/density scale
responsive breakpoints or constraint variants
DPI scaling policy
target-specific font metrics adjustments with reports
```

### 9.6 Prevent CSS-like chaos

Rules:

- no global cascade;
- no implicit imports;
- no undeclared recipe slots;
- no name collisions inside capsule namespace;
- no local raw values in production examples unless marked and allowed;
- no hidden style fallback.

---

## 10. Application Model Plan

### 10.1 State/events/commands

Final model:

```text
state declarations
computed bindings
commands
event routing
async tasks
navigation state
redraw requests
error propagation
```

Events must be Block events, not DOM events.

### 10.2 Navigation/focus/undo

Required:

```text
keyboard navigation
focus scopes
focus trap for overlays/dialogs
shortcut scopes
undo/redo action stack for text/form/editor scope
```

### 10.3 Component/recipe authoring without React runtime

Use:

```text
Block
Morph recipe
Morph material
Morph affordance
state command
```

Do not use:

```text
React component lifecycle
virtual DOM
hooks runtime
CSS-in-JS
```

### 10.4 App shell integration

App source should declare shell needs:

```tetra
surface app ProjectApp:
  window title "Project"
  window size 1200 800
  permission filesystem.read user_selected
  menu app_menu
  tray optional
```

Implementation syntax may differ, but the model must exist.

### 10.5 IPC/process model

Use actor/task/service model where appropriate:

```text
UI actor owns Surface tree
background tasks send owned messages
Surface handles/events/frames do not cross unsafe boundaries
host bridge is capability-checked
```

---

## 11. Platform Host Plan

### 11.1 Linux

First production target.

Required:

```text
real window
input pump
wheel
DPI/scale
cursor
clipboard
IME/composition
menus/dialogs/notifications if in app-shell scope
accessibility bridge
packaging
```

### 11.2 Windows

Start as `BETA_TARGET_HOST_WINDOWS`.

Required before production:

```text
Win32/DirectComposition or equivalent host path
window/input/clipboard/IME/DPI/menu/dialog/notification/tray
accessibility bridge
installer/signing
same-commit target-host CI
```

### 11.3 macOS

Start as `BETA_TARGET_HOST_MACOS`.

Required before production:

```text
Cocoa/AppKit host adapter only as shell/input/accessibility provider
not as widget layer
menu bar/dialogs/notifications/DPI/IME/accessibility
bundle/sign/notarize
same-commit target-host CI
```

### 11.4 Web

Keep wasm32-web as browser-canvas Surface target.

Required:

```text
compiler-owned boot
canvas renderer
browser input/clipboard/composition
a11y snapshot/mirror
no user JS app logic
no DOM UI tree for Surface app
```

### 11.5 Unsupported target diagnostics

For every unsupported target:

```text
clear compiler/runtime diagnostic
no output artifact if unsafe
validator rejects production claim
feature manifest says unsupported/nonclaim
```

---

## 12. Developer Experience Plan

Required CLI/dev tools:

```text
tetra surface check
tetra surface run
tetra surface dev
tetra surface inspect
tetra surface screenshot
tetra surface test-golden
tetra surface profile
tetra surface package
tetra new surface-app
```

### 12.1 Hot reload / fast rebuild

- Rebuild on file change.
- Reopen/update Surface window.
- Preserve state only when safe and schema-compatible.
- Report source diagnostics in terminal and inspector.

### 12.2 UI inspector

Must show:

```text
Block tree
Morph expansion
resolved tokens/materials
layout boxes
paint commands
hit-test/focus paths
event trace
accessibility tree
performance counters
source locations
```

### 12.3 Screenshot/golden tests

- Generate deterministic headless frames.
- Store semantic and pixel artifacts.
- Validate asset/font hashes.

### 12.4 Accessibility inspector

- Show role/name/state/action/bounds.
- Show focus order and reading order.
- Flag unnamed focusable Blocks.

### 12.5 Performance profiler

- Frame time.
- Layout time.
- Paint time.
- Text shaping time.
- Glyph/asset/layout cache bytes.
- Event latency.

### 12.6 Project templates

Templates:

```text
surface-minimal
surface-dashboard
surface-form
surface-editor-shell
surface-tray-app
surface-web-canvas
```

---

## 13. Security And Sandbox Plan

### 13.1 Permissions

Surface app permissions:

```text
filesystem read/write
network
clipboard
notifications
open-url
camera/microphone if ever added
tray/system integration
process spawn
```

No permission means no host action.

### 13.2 Filesystem/network boundaries

- User-selected files default.
- App-specific storage directory.
- No arbitrary filesystem access unless declared.
- Network must be declared and target-gated.

### 13.3 IPC hardening

- Typed messages.
- Owned payloads.
- No Surface handles/events/frames crossing worker boundary unless specifically modeled.
- Crash isolation.

### 13.4 Untrusted assets/fonts/images

- Whitelist formats.
- Size limits.
- Decode error diagnostics.
- Fuzz parsers.
- No network font/image in release test path.

### 13.5 Supply chain/capsules

- Capsule lockfile.
- Asset hashes.
- Recipe namespace and version.
- No hidden runtime dependencies.

### 13.6 Electron-class attack surfaces Surface must avoid

Avoid:

```text
Node integration in renderer
remote code execution through HTML/JS
untrusted preload scripts
webview as core app UI
unrestricted filesystem from UI context
```

Surface-specific risks must be tested:

```text
host ABI misuse
asset decoder bugs
permission bypass
IPC message forgery
malicious recipe/capsule imports
```

---

## 14. Performance And Memory Plan

### 14.1 Startup budgets

Record:

```text
compiler/build time for dev loop
app process launch time
window creation time
first frame time
asset/font load time
```

### 14.2 Frame budgets

Record:

```text
layout ms
style/Morph resolve ms
text/glyph ms
paint ms
present ms
input-to-frame ms
animation jitter
```

### 14.3 Memory budgets

Record:

```text
peak RSS
steady RSS
Block tree bytes
resolved scene bytes
layout cache bytes
glyph cache bytes
asset cache bytes
framebuffer bytes
paint command buffer bytes
allocations per frame
```

### 14.4 Binary size budgets

Record:

```text
app binary size
runtime/host adapter size
asset bundle size
installer/package size
```

### 14.5 Power/CPU budgets

Record:

```text
idle CPU
animation CPU
input burst CPU
background task CPU
```

### 14.6 Benchmark methodology against Electron

Rules:

- equivalent app shapes;
- same host machine;
- cold and warm runs;
- at least 20 runs for public numbers;
- report variance;
- no official benchmark claim;
- no faster-than-Electron claim unless validated by benchmark report.

---

## 15. Accessibility And Internationalization Plan

### 15.1 Accessibility graph

Required fields:

```text
role
name
description
value
state
actions
label_for
labelled_by
focusable
tab index
reading order
bounds
live/status
keyboard action
```

### 15.2 Screen-reader evidence

Production desktop target requires:

```text
platform accessibility bridge report
screen-reader smoke or equivalent target-host assistive-tech validation
manual audit artifact if automation unavailable
nonclaim if not proven
```

### 15.3 Keyboard navigation

- Every action reachable by keyboard.
- Focus visible cannot be styled away.
- Dialogs trap focus.
- Lists/tabs/support roving focus where needed.

### 15.4 IME/composition

Per target:

```text
composition start/update/commit/cancel
candidate/preedit handling where exposed
text insertion trace
clipboard boundary copy safety
```

### 15.5 Text shaping and bidi

Tier support:

```text
Tier 1: UTF-8 storage, Latin shaping/fallback, forms
Tier 2: common Unicode scripts, bidi, combining marks
Tier 3: full editor-grade shaping/selection
```

Do not claim higher tier without tests.

### 15.6 Localization hooks

- String catalogs.
- Plural/date/number formatting strategy.
- Locale resource packaging.
- Missing localization diagnostics.

---

## 16. Testing And Release Gates

### 16.1 Test categories

Required:

```text
unit tests
semantic tests
Block graph tests
Morph graph tests
renderer tests
layout tests
text/input tests
event/focus tests
accessibility tests
platform target-host tests
visual regression tests
security/permission tests
packaging tests
performance tests
fuzz/property tests
stress/soak tests
same-commit artifact validation
```

### 16.2 New validators to add

```text
validate-surface-prod-claim
validate-surface-prod-audit
validate-surface-app-shell-report
validate-surface-visual-report
validate-surface-perf-report
validate-surface-security-report
validate-surface-package-report
validate-surface-accessibility-target-report
validate-surface-inspector-snapshot
validate-surface-electron-comparison-report
```

### 16.3 New release scripts to add

```text
scripts/release/surface/prod-gate.sh
scripts/release/surface/visual-gate.sh
scripts/release/surface/perf-gate.sh
scripts/release/surface/security-gate.sh
scripts/release/surface/package-gate.sh
scripts/release/surface/linux-prod-gate.sh
scripts/release/surface/windows-beta-gate.sh
scripts/release/surface/macos-beta-gate.sh
scripts/release/surface/web-prod-gate.sh
```

### 16.4 Required broad commands

Final broad candidate commands:

```bash
git rev-parse HEAD
git status --short

GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-surface-prod" GOTMPDIR="$PWD/.cache/go-tmp-surface-prod" \
go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1

bash scripts/ci/test.sh

bash scripts/release/surface/block-system-gate.sh --report-dir reports/surface-prod/final/block-system
bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-prod/final/morph
bash scripts/release/surface/release-gate.sh --report-dir reports/surface-prod/final/surface-v1
bash scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final/prod

bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/surface-prod/final/safe-view-lifetime
bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-prod/final/api-stability

go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest reports/surface-prod/final/artifact-hashes.json

git diff --check
git diff --exit-code -- docs/generated/manifest.json
```

Target-host commands for broader tiers:

```bash
go run ./tools/cmd/platform-ui-runtime-smoke --target windows-x64 --report windows-ui-runtime.json
go run ./tools/cmd/validate-windows-ui-runtime --report windows-ui-runtime.json --expected-version "$version" --expected-git-head "$git_head"

go run ./tools/cmd/platform-ui-runtime-smoke --target macos-x64 --report macos-ui-runtime.json
go run ./tools/cmd/validate-macos-ui-runtime --report macos-ui-runtime.json --expected-version "$version" --expected-git-head "$git_head"
```

### 16.5 Artifact hash and same-commit evidence

Every report must include:

```text
git_head
git_dirty
command_line
generated_at_utc
host_os/arch
target
artifact hashes
schema version
producer
validator provenance
```

Final production claim requires `git_dirty: false`.

---

## 17. Documentation And Claim Governance

### 17.1 Docs to update

```text
docs/spec/current_supported_surface.md
docs/spec/surface_morph.md
docs/spec/surface_production_platform.md
docs/user/surface_guide.md
docs/user/examples_index.md
docs/release/surface_v1_release_contract.md
docs/release/surface_prod_release_audit.md
README.md
compiler/features.go
docs/generated/manifest.json
```

### 17.2 Manifest feature statuses

Feature IDs must separate:

```text
ui.surface-core current scoped
ui.surface-block-system experimental/current candidate depending gate
ui.surface-morph-capsule experimental -> stable only after prod gate
ui.surface-prod-linux-web new scoped production
ui.surface-windows-x64 beta/unsupported until target-host prod
ui.surface-macos-x64 beta/unsupported until target-host prod
ui.surface-gpu experimental/nonclaim
ui.surface-devtools current/experimental depending gate
```

### 17.3 Claim-tier language

Required wording:

```text
production-stable in [target matrix]
Block/Morph-first
Tetra-owned renderer
no Electron/React/CSS runtime dependency
unsupported targets listed explicitly
```

Forbidden wording unless validated:

```text
full Electron replacement
full CSS parity
full React compatibility
all-platform desktop UI
full screen-reader support
GPU renderer production
```

### 17.4 Overclaim scanners

Add scanner command:

```bash
rg -n "Electron replacement|React replacement|CSS parity|full UI|cross-platform UI|GPU production|screen-reader|Windows Surface production|macOS Surface production|100%|complete" docs README.md compiler lib examples scripts tools .github
```

Validator must distinguish forbidden claims from explicit nonclaims.

### 17.5 Nonclaims that must remain explicit

Until evidence changes:

```text
no full DOM/CSS/browser compatibility
no React runtime compatibility
no Electron/Chromium shell
no user JavaScript UI logic
no native platform widgets as UI layer
no Windows/macOS production Surface without target-host gates
no full rich text editor
no full screen-reader support
no GPU production renderer
no arbitrary webview
no official benchmark superiority
```

---

## 18. Migration Plan

### 18.1 Existing `lib.core.widgets`

Keep as compatibility. Map to Block/Morph recipes:

| Existing helper | Target recipe/Block shape |
|---|---|
| `Text` | `Block + text role` |
| `Label` | `Block + text + label_for` |
| `StatusText` | `Block + status/live affordance` |
| `Button` | `Block + action affordance + control material + interactive lens` |
| `TextBox` | `Block + text field affordance + editable input + focus/error lens` |
| `Checkbox` | `Block + toggle affordance + checked state lens` |
| `Row` | layout recipe |
| `Column` | layout recipe |
| `Panel` | material/layout recipe |
| `Stack` | layout recipe |
| `Scroll` | scroll recipe |
| `Spacer` | layout Block/min-size helper |

### 18.2 Existing Block examples

Promote only after:

- examples use stable Block ABI;
- reports cover headless/linux/web;
- visual regression baselines exist;
- a11y/perf/security evidence exists.

### 18.3 Existing Morph examples

`examples/surface_morph_command_palette.tetra` is the seed. Expand to:

```text
surface_morph_dashboard.tetra
surface_morph_settings.tetra
surface_morph_editor_shell.tetra
surface_morph_file_manager.tetra
surface_morph_multi_window.tetra
```

### 18.4 App migration without breakage

Migration stages:

1. Keep Surface v1 widgets working.
2. Add Block equivalents.
3. Add Morph recipes.
4. Add migration guide and compatibility shims.
5. Mark new production examples as Morph-first.
6. Only deprecate old helpers after two release cycles and gates prove replacements.

### 18.5 Compatibility/deprecation policy

Do not deprecate current v1 widgets yet. Label them:

```text
supported compatibility toolkit for bounded Surface v1 apps
not final architecture for new production Surface apps
```

---

## 19. Risk Register

| # | Risk | Severity | Detection | Mitigation | Blocking gate |
|---:|---|---|---|---|---|
| 1 | Fake Electron replacement claim | Critical | overclaim validator | claim tiers + required target matrix | P01/P35 |
| 2 | Morph becomes CSS chaos | High | token/style conflict tests | no cascade, fixed override order | P11 |
| 3 | Recipe zoo replaces Block truth | High | recipe validator | recipes output Blocks only | P14 |
| 4 | Renderer visual quality too low | High | visual regression and user examples | text/glyph/paint hardening | P06/P08/P25 |
| 5 | Windows/macOS hidden unsupported | Critical | target-host validators | beta tiers until evidence | P17/P18 |
| 6 | Accessibility overclaim | Critical | a11y report validator | screen-reader nonclaim unless tested | P20 |
| 7 | Text shaping insufficient | High | text tests across scripts | scoped shaping tiers | P08/P30 |
| 8 | IME broken | High | target-host input tests | composition trace per target | P09 |
| 9 | App shell too shallow | High | app-shell report | menus/dialogs/tray/window gates | P15 |
| 10 | Packaging absent | High | package report validator | platform package gates | P26 |
| 11 | Security sandbox weak | Critical | security negative tests | permissions/capabilities | P27 |
| 12 | Asset decoder vulnerability | High | fuzz/security tests | whitelist + sandbox | P21/P27 |
| 13 | Performance worse than Electron | Medium/High | perf benchmarks | budgets and profiling | P31/P36 |
| 14 | Memory leaks/caches unbounded | High | memory report | cache caps/eviction | P31 |
| 15 | Devtools missing hurts adoption | Medium | DX checklist | inspector/hot reload MVP | P23/P24 |
| 16 | Visual golden tests flaky | Medium | CI instability | semantic + pixel baselines | P25 |
| 17 | Actor/task integration unsafe | High | boundary tests | owned messages only | P28 |
| 18 | Dirty checkout signoff | Critical | audit validator | `git_dirty:false` required | P35 |
| 19 | Docs drift from code | High | manifest/docs validators | generated manifest gate | P34/P35 |
| 20 | Web target confused with DOM UI | High | web validator | separate canvas Surface claim | P19 |
| 21 | GPU scope creep | Medium | renderer report | software first, GPU experimental | P07 |
| 22 | Compatibility widgets break | High | migration tests | keep compatibility layer | P32 |
| 23 | Cross-platform shell API too large | High | host capability matrix | phased shell capabilities | P15-P18 |
| 24 | Benchmark hype | Medium | benchmark validator | method-first reporting | P36 |
| 25 | Internationalization ignored | Medium | i18n tests | staged locale/text support | P30 |

---

## 20. Final Definition Of Done

### 20.1 Exact final command list for `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`

Run in a clean real checkout:

```bash
git rev-parse HEAD
git status --short
```

`git status --short` must be empty before final audit.

Then:

```bash
export GOTELEMETRY=off
export GOCACHE="$PWD/.cache/go-build-surface-prod-final"
export GOTMPDIR="$PWD/.cache/go-tmp-surface-prod-final"
mkdir -p "$GOCACHE" "$GOTMPDIR"

go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test.sh

bash scripts/release/surface/block-system-gate.sh --report-dir reports/surface-prod/final/block-system
bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-prod/final/morph
bash scripts/release/surface/release-gate.sh --report-dir reports/surface-prod/final/surface-v1
bash scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final/prod
bash scripts/release/surface/visual-gate.sh --report-dir reports/surface-prod/final/visual
bash scripts/release/surface/perf-gate.sh --report-dir reports/surface-prod/final/perf
bash scripts/release/surface/security-gate.sh --report-dir reports/surface-prod/final/security
bash scripts/release/surface/package-gate.sh --report-dir reports/surface-prod/final/package
bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/surface-prod/final/safe-view-lifetime
bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-prod/final/api-stability

go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-surface-prod-audit --audit docs/release/surface_prod_release_audit.md --expected-status PROD_STABLE_SCOPED

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest reports/surface-prod/final/artifact-hashes.json

git diff --check
git diff --exit-code -- docs/generated/manifest.json
git status --short
```

### 20.2 Required reports

```text
reports/surface-prod/final/prod/surface-prod-summary.json
reports/surface-prod/final/block-system/surface-block-system-gate-summary.json
reports/surface-prod/final/morph/surface-morph-gate-summary.json
reports/surface-prod/final/surface-v1/surface-release-summary.json
reports/surface-prod/final/visual/surface-visual-report.json
reports/surface-prod/final/perf/surface-perf-report.json
reports/surface-prod/final/security/surface-security-report.json
reports/surface-prod/final/package/surface-package-report.json
reports/surface-prod/final/safe-view-lifetime/*
reports/surface-prod/final/api-stability/*
reports/surface-prod/final/artifact-hashes.json
```

### 20.3 Required target-host evidence

For scoped Linux/Web:

```text
headless deterministic
linux-x64 real-window
wasm32-web browser-canvas
```

For broader desktop claim:

```text
windows-x64 target-host real-window
macos-x64 target-host real-window
platform accessibility target-host reports
packaging/signing target reports
```

### 20.4 Exact conditions for `PROD_STABLE_SCOPED`

All must be true:

1. Clean same-commit checkout.
2. Stable Block ABI report passes.
3. Stable Morph/style graph report passes.
4. Renderer/compositor software production report passes.
5. Layout/style/event/app model reports pass.
6. Text/input/IME/clipboard scoped reports pass.
7. Accessibility target reports pass for supported targets.
8. App shell report passes for Linux and scoped web story.
9. Visual regression report passes.
10. Performance/memory budgets pass.
11. Security/sandbox report passes.
12. Packaging report passes for supported desktop scope.
13. Docs/manifest match code.
14. CI/release gates pass without `continue-on-error` bypass.
15. Nonclaims list unsupported targets/features.
16. Claim validator accepts final wording and rejects fake variants.

### 20.5 Exact conditions for broader “Electron replacement” claim

Allowed only when all `PROD_STABLE_SCOPED` conditions pass for:

```text
linux-x64
windows-x64
macos-x64
headless
```

And, if claim mentions web:

```text
wasm32-web browser-canvas
```

Plus:

```text
packaging/signing/notarization/update path per platform
screen-reader/assistive-tech evidence per desktop platform
app-shell features per platform
performance comparison report against Electron app shapes
```

Claim text must include:

```text
in the supported target matrix
for Block/Morph Tetra app UI
without Electron/React/CSS runtime dependency
```

---

## 21. Recommended First 10 Commits

1. **Add Surface production claim taxonomy and validator skeleton**  
   Files: `tools/cmd/validate-surface-prod-claim`, `tools/validators/surfaceprod`, fixtures.  
   Test: fake Electron replacement rejected.

2. **Add `docs/spec/surface_production_platform.md` and manifest feature IDs**  
   Files: docs + `compiler/features.go`.  
   Test: docs/manifest validators.

3. **Add `surface-prod` baseline audit script/report**  
   Files: `scripts/release/surface/prod-baseline.sh`.  
   Test: baseline report generated in `reports/surface-prod/P00-baseline`.

4. **Harden Morph validator for stable style graph fields**  
   Files: `tools/cmd/validate-surface-morph-report`, `tools/validators/surface/report.go`.  
   Test: token alias/duplicate recipe/unreported expansion negative cases.

5. **Add Block ABI/scene contract report schema**  
   Files: `tools/cmd/validate-surface-block-report`, docs.  
   Test: missing resolved scene rejected.

6. **Add visual regression MVP**  
   Files: `tools/cmd/surface-golden`, `tools/cmd/validate-surface-visual-report`, `scripts/release/surface/visual-gate.sh`.  
   Test: one command palette baseline.

7. **Add app-shell capability schema**  
   Files: `docs/spec/surface_host_abi.md`, `tools/cmd/validate-surface-app-shell-report`.  
   Test: missing menu/dialog/window capability rejected when claimed.

8. **Add performance/memory report schema and thresholds**  
   Files: `tools/cmd/validate-surface-perf-report`, `scripts/release/surface/perf-gate.sh`.  
   Test: unbounded cache/fake zero memory rejected.

9. **Add security/permission Surface report**  
   Files: `docs/spec/surface_security.md`, `tools/cmd/validate-surface-security-report`.  
   Test: filesystem/network/clipboard action without permission rejected.

10. **Create aggregate `scripts/release/surface/prod-gate.sh`**  
    Files: release script + CI draft.  
    Test: gate fails until required reports exist, then passes only with all artifacts/hashes.

---

## 22. Prompt For Codex Goal

```text
/goal Реалізуй packet-by-packet план `docs/plans/2026-06-10-surface-electron-competitor-production-plan.md`.

Місія: довести Tetra Surface до чесного `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` як Block/Morph-first production UI platform без Electron, React, DOM/CSS runtime або user JavaScript app logic. Не роби broad Electron replacement claim, доки Windows/macOS/Web/desktop target-host gates не доведені.

Почни з `SURFACE-PROD-P00`, створи baseline truth audit, потім `SURFACE-PROD-P01` claim taxonomy/overclaim validator. Працюй packet-by-packet. Кожен packet має RED tests/negative fixtures, implementation, validators/gates, artifact hashes, and docs/manifest updates if public claims change.

Done_when:
- all packets required for `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` pass;
- final clean same-commit checkout exists;
- `scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final/prod` passes;
- Block/Morph/app-shell/renderer/text/layout/accessibility/security/perf/package/visual reports exist and validate;
- docs/manifest match actual evidence;
- unsupported targets/features remain explicit nonclaims;
- final audit validator returns `PROD_STABLE_SCOPED`.
```
