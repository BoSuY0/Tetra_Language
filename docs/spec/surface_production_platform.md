# Tetra Surface Production Platform

Status: planned scoped production contract.

This document defines the final Surface production claim target:

`PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`

The machine-readable scope name is:

`surface-prod-scoped-linux-web`

This is the mountain the production plan is climbing. It is not a claim that
the current checkout already replaces Electron for every desktop application.
The claim can become true only when the Surface production gates validate the
same commit with target-host evidence, report schemas, artifact hashes, and
explicit nonclaims.

## Meaning Of The Claim

Within the scoped target matrix, Surface is intended to replace Electron,
React, and CSS runtime dependencies as the user-facing UI layer for pure-Tetra
applications. In that scope:

- application UI is authored in Tetra;
- the UI tree, layout, style resolution, input routing, state updates, text,
  accessibility metadata, and rendering evidence are owned by Surface;
- the scoped text path uses `tetra.surface.text-pipeline.v1` evidence for
  Tier 1 Latin/UTF-8 glyph runs, font manifest hashes, fallback chain, bounded
  glyph cache, Unicode scalar and cluster boundaries, deterministic
  measurement, wrap/ellipsis/alignment/baseline, caret/selection rectangles,
  and IME composition spans, while unsupported complex scripts, bidi shaping,
  platform widget text controls, and full Unicode editor semantics remain
  explicit nonclaims;
- the scoped editing path uses `tetra.surface.text-editing.v1` evidence for
  owned editable TextBox storage, forms and command palette search safety,
  target IME traces, clipboard owned-copy transfers, selection replacement,
  undo unit boundaries, validation diagnostics, and nonclaims for rich text,
  full editor-grade text semantics, and native platform text controls;
- the Block layout path uses `tetra.surface.layout-engine.v1` evidence for
  row/column/stack/grid/dock/absolute/overlay/scroll modes, min/max/fit/fill/
  fixed constraints, DPI/density-independent layout, explicit overflow/clip/
  scroll rules, resize/scroll invalidation traces, bounded layout cache
  evidence, app shell/settings forms/dashboards/editor shells resize stability,
  and rejection of CSS flexbox/grid parity, accidental overflow-hidden behavior,
  unbounded layout cache evidence, and missing DPI/density evidence;
- the Block app-model path uses `tetra.surface.app-model.v1` evidence at
  `production-app-model-v1` level for `owned-state-store-v1`,
  `typed-command-dispatch-v1`, `block-event-trace-v1`,
  `actor-task-safe-boundary-v1`, navigation/focus scopes, scoped shortcuts,
  command error propagation, explicit redraw scheduling, and the required
  command palette/dashboard/settings/editor shell surfaces, while rejecting
  missing event traces, disabled dispatch, unfocused text input, unsafe
  actor/task boundaries, and React runtime claims;
- the host boundary is the tiny Surface Host ABI, not Electron, Chromium
  desktop shell APIs, React runtime, DOM UI, CSS cascade runtime, user
  JavaScript application logic, Qt, GTK, Cocoa, WinUI, or platform-native
  widgets;
- software RGBA rendering is the production rendering baseline, with
  `tetra.surface.software-renderer.v1` evidence required for deterministic
  raster output, alpha blending, clipping, frame/golden checksums, resize/scale/
  DPI behavior, use-after-present rejection, and frame-alias rejection, until a
  separate `tetra.surface.renderer-backend.v1` GPU/compositor gate proves
  target-host backend reports, same-scene equivalence, fallback behavior, and
  the required compositor capabilities;
- every promoted claim must point to a report schema, validator, release script,
  and same-commit target evidence.

The phrase "Electron replacement" therefore means "replace the Electron UI
runtime dependency inside the scoped Surface target matrix." It does not mean
drop-in compatibility with arbitrary Electron apps, full Chromium platform
coverage, Node/Electron APIs, or all desktop targets.

## Current / Experimental / Unsupported / Future

Current release-scoped Surface v1 support remains the truth boundary recorded
in `docs/spec/surface_v1.md` and `docs/spec/current_supported_surface.md`.
The current release scope is `surface-v1-linux-web`, covering headless release
evidence, linux-x64 real-window evidence, and wasm32-web browser-canvas
evidence.

Experimental tracks:

- Surface Block System remains experimental until its production contract,
  renderer contract, visual regression, accessibility, performance, and final
  production gates promote it.
- Morph Capsule remains experimental; its `typed-style-graph-candidate-v1`
  boundary may replace CSS cascade semantics only inside scoped Surface
  evidence, and later production packets must still promote target evidence,
  app model, accessibility, packaging, performance, and final gates.
- GPU/compositor backend remains experimental/nonclaim. The current renderer
  backend decision is `software-only-prod-go-gpu-experimental`; software RGBA
  can carry the first scoped production claim without a GPU backend.

Unsupported targets for this planned production claim:

- macOS Surface host production UI;
- Windows Surface host production UI;
- wasm32-wasi Surface UI runtime;
- GPU production renderer/compositor;
- full accessibility parity across all platform screen readers;
- rich native text editor parity;
- drop-in Electron, React, DOM, CSS, or Node API compatibility.

Future work must stay future in docs, manifests, and release reports until the
corresponding target-host evidence and validators pass.

## Target Matrix

| Target | Claim role | Required evidence |
|---|---|---|
| `headless` | release evidence target | deterministic runtime, layout, style, text/input, `tetra.surface.accessibility-target.v1` deterministic inspector evidence, `tetra.surface.asset-pipeline.v1` local asset pipeline evidence, visual, perf, security, and artifact reports |
| `linux-x64` | production target candidate | real-window target-host evidence, native input, clipboard, IME/composition, `tetra.surface.accessibility-target.v1` with Linux accessibility host bridge/probe evidence, `tetra.surface.asset-pipeline.v1` local asset pipeline evidence, packaging, diagnostics, perf/memory, and security reports |
| `wasm32-web` | production target candidate | `tetra.surface.browser-canvas-target.v1` / `wasm32-web-first-class-browser-canvas-target-v1`, browser-canvas target-host evidence, compiler-owned boot, browser input, clipboard/composition, `tetra.surface.accessibility-target.v1` with accessibility snapshot/mirror evidence, `tetra.surface.asset-pipeline.v1` local asset pipeline evidence, visual, perf/memory, and security reports |
| `macos-x64` | unsupported until proven | `tetra.surface.macos-target.v1` nonclaim/beta boundary; explicit unsupported/nonclaim status unless real macOS target-host Surface reports exist |
| `windows-x64` | unsupported until proven | `tetra.surface.windows-target.v1` nonclaim/beta boundary; explicit unsupported/nonclaim status unless real Windows target-host Surface reports exist |
| `wasm32-wasi` | unsupported UI target | explicit unsupported/nonclaim status |

For Linux, the current production-host-adapter evidence object is
`tetra.surface.linux-host-adapter.v1` at
`linux-x64-production-host-adapter-v1`. It is scoped to the
`linux-x64-release-window-v1` Wayland shm RGBA path, app-shell ABI evidence, and
`linux-x64-unpacked-binary-v1` packaging scope. Blocked display reports and
offscreen-only evidence cannot promote a Linux production claim.

## Required Claim Governance

The first claim gate is `tetra.surface.prod-claim.v1`, validated by
`validate-surface-prod-claim`.

The command is:

`go run -buildvcs=false ./tools/cmd/validate-surface-prod-claim --report <report>`

The release-script wrapper is:

`bash scripts/release/surface/prod-claim-gate.sh --report <report>`

The claim validator rejects:

- fake Electron/React/CSS replacement claims;
- fake cross-platform production support;
- fake GPU production support;
- fake full accessibility parity;
- missing target-host evidence;
- forbidden runtime dependencies;
- dirty-checkout production claims;
- paper evidence such as docs-only, mock, fake, or placeholder reports.

Later production gates must aggregate this claim gate with the Surface v1
release gate, Block System gate, Morph gate, renderer backend decision gate,
visual regression gate, performance gate, security gate, IPC/lifecycle gate,
crash diagnostics gate, i18n/localization gate, migration gate, packaging gate,
example-suite gate, API stability gate, and final same-commit audit.

The performance gate is `scripts/release/surface/perf-gate.sh`. It must produce
`tetra.surface.perf-report.v1` / `surface-performance-memory-v1` evidence with
baseline environment capture, bounded cache evidence, startup time, first frame
time, steady frame p95, peak RSS, frame allocations, layout/glyph/asset cache
bytes, binary size, CPU idle power proxy, input latency, animation frame jitter,
and fake-claim rejection for unsupported faster-than-Electron claims, fastest
UI framework claims, and zero memory overhead claims.

The migration gate is `scripts/release/surface/migration-gate.sh`. It must
produce `tetra.surface.migration-report.v1` /
`surface-widget-block-migration-v1` evidence that existing Surface v1 widget
examples still pass, `lib.core.widgets` remains a compatibility layer,
widget-to-Block/Morph mappings exist, docs recommend Block/Morph for new
production UI, and fake claims that widgets are the core final architecture are
rejected.

The example-suite gate is `scripts/release/surface/example-suite-gate.sh`. It
must produce `tetra.surface.example-suite-report.v1` /
`surface-production-example-suite-v1` evidence for ten realistic Surface app
shapes, executable `examples/surface_prod_*.tetra` examples, Block/Morph-only
production examples, headless/linux-x64/wasm32-web target coverage,
event/state/accessibility/performance-budget evidence, and ecosystem seed
metadata. It must reject screenshot-only examples, React/Electron/DOM runtime
example dependencies, widgets where Block/Morph is required, missing app
shapes, missing scoped target coverage, and toy visual-only examples.

The final CI/release aggregator is
`scripts/release/surface/prod-gate.sh`. It must produce
`tetra.surface.prod-gate-report.v1` /
`surface-production-ci-release-gate-v1` evidence, validate the release state
with `validate-surface-release-state --scope
PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`, upload `surface-production-final/**` from
`.github/workflows/release-packages.yml`, and run without
`continue-on-error`. The aggregator must prove Surface release, Block/Morph,
visual, package, security, IPC/lifecycle, crash diagnostics, i18n/localization,
performance, widget migration, example-suite, API stability, prod-claim
governance, and artifact hash manifests on the same commit. It must reject a
missing production CI job, a skipped Linux/web production target counted as
pass, and a missing artifact hash manifest.

The final same-commit audit is
`docs/release/surface_prod_release_audit.md`. It embeds
`tetra.surface.prod-audit.v1` /
`surface-prod-final-same-commit-audit-v1` evidence and is validated by
`validate-surface-prod-audit`. The final verdict enum is
`PROD_STABLE_SCOPED/NEAR_READY_WITH_BLOCKERS/BETA_ONLY/EXPERIMENTAL_ONLY/FAIL`.
The audit must reject clean checkout violations, a report from different git
head, missing unsupported-target nonclaims, and any public claim not generated
from the audit result.

The Surface-vs-Electron public positioning gate is
`scripts/release/surface/electron-comparison-gate.sh`. It produces
`tetra.surface.electron-comparison-report.v1` /
`surface-electron-comparison-method-v1` evidence through
`surface-electron-comparison` and
`validate-surface-electron-comparison-report`. Public wording may say
competitive with Electron in the supported scope only when the report records
equivalent app shapes, sample/variance data, environment capture, and method
artifacts. Public benchmark-superiority claim rejection, cherry-picked hardware
rejection, missing variance rejection, unfair app shape rejection, missing
environment rejection, and single-smoke faster-than-Electron claim rejection
are mandatory.

## Exact Nonclaims

Until every required production packet passes, Surface does not claim:

- broad Electron replacement;
- drop-in compatibility with arbitrary Electron applications;
- React runtime compatibility;
- CSS cascade runtime compatibility;
- DOM UI or user JavaScript application UI;
- Chromium desktop shell dependency compatibility;
- cross-platform desktop parity;
- macOS or Windows production Surface UI;
- wasm32-wasi Surface UI;
- GPU production rendering;
- full accessibility parity;
- full rich text editor parity;
- full bidi shaping, full ICU/CLDR localization, or platform-native
  localization framework parity;
- fastest UI framework status, zero memory overhead, broad Electron
  performance replacement, or cross-platform desktop performance parity;
- public benchmark-superiority, faster-than-Electron from one local smoke,
  cherry-picked benchmark hardware, or unfair app-shape comparisons;
- widgets as the core final architecture or breaking Surface v1 widget examples
  without migration;
- screenshot-only example proof, examples requiring React/Electron/DOM runtime,
  or broad cross-platform parity from the example suite alone;
- production packaging, signing, auto-update, crash recovery, or app shell
  parity before their dedicated gates pass.

## Promotion Rule

`PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` can be promoted only when:

- `validate-surface-prod-claim` accepts the claim report for the same commit;
- `prod-gate.sh` produces `tetra.surface.prod-gate-report.v1` and
  `surface-production-ci-release-gate-v1` evidence in
  `surface-production-final/**`;
- `validate-surface-prod-audit` accepts
  `docs/release/surface_prod_release_audit.md` as
  `PROD_STABLE_SCOPED` from a clean checkout with same-commit reports and a
  public claim generated from audit result;
- Surface release, Block, Morph/style, renderer, visual, accessibility,
  app-shell, security, IPC/lifecycle, crash diagnostics, i18n/localization,
  migration, example-suite, packaging, performance, API, and documentation
  gates all pass for the scoped target matrix;
- unsupported targets remain explicit unsupported entries unless real
  target-host evidence promotes them;
- public docs keep the exact nonclaims above.

If a report, docs page, release note, or manifest entry uses broader wording
than this document allows, the production claim must fail.
