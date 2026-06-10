# Tetra Surface v1 Release Contract

Status: current for surface-v1-linux-web scope after release gate passes.

This contract is the release-truth boundary for promoting Tetra Surface v1. It
does not promote every Surface experiment to production. A release/current claim
is valid only when release reports use `release_scope: surface-v1-linux-web`,
`status: current`, `experimental: false`, and `production_claim: true`, and the
release gate validates the required target evidence and artifact hashes.

## Supported

- pure-Tetra user UI code
- tiny Surface Host ABI
- headless release evidence target
- linux-x64 real-window runtime
- wasm32-web browser-canvas runtime
- software RGBA framebuffer presentation
- component tree helper API
- production widget toolkit v1 subset
- production text/input baseline
- clipboard baseline
- IME/composition baseline
- accessibility metadata plus platform bridge for supported targets
- release validators and artifact hashes

## Block System Status

`ui.surface-block-system` is experimental in this release contract. It records
the Block-first Surface architecture direction: Block as the core Surface
primitive for visual composition. Button/Card/TextField-like shapes are
recipes/compatibility rather than required core widget classes. It is not
current release support and is not production support. The current same-commit
evidence is scoped to `tetra.surface.block-system.gate.v1` reports for
headless, linux-x64 real-window, and wasm32-web browser-canvas targets,
validated artifact hashes, and `reports/surface-block/p18-budget`. That
evidence keeps the no production Block claim boundary.

The same Block-system reports now carry `tetra.surface.app-shell.v1` /
`production-app-shell-host-abi-v1` app-shell Host ABI evidence for windows,
lifecycle, menus/context menus, dialogs/file pickers, tray/status items,
notifications, cursors, drag/drop, permissions, clipboard, IME, DPI/scale, and
open URL/file requests. Menu support without target-host traces, notification
support without delivered host reports, and unsupported host features that
silently no-op are rejected. This is still app-shell ABI evidence, not broad
desktop production promotion.

The Block gate now requires each Block-system runtime report to include
`block_system.memory_budget` evidence. The budget is scoped to the reported
scene and records Block count, stress count, render/state loop counts,
framebuffer bytes, bounded paint/text/asset caches, and explicit nonclaims.
This budget evidence does not promote Block to production support and does not
stand in for an Electron comparison benchmark.

Block-system runtime reports may also include `app_model` evidence with schema
`tetra.surface.app-model.v1` and level `production-app-model-v1`. That evidence
records owned state stores, typed command dispatch, ordered Block event traces,
safe actor/task async command boundaries, navigation/focus scopes, scoped
shortcuts, propagated command errors, explicit redraw scheduling, and the
command palette/dashboard/settings/editor shell acceptance surfaces. The
validator rejects missing event traces, disabled dispatch, unfocused text
delivery, unsafe actor/task boundaries, and React runtime claims. This remains
experimental Block-system evidence.

Block-system runtime reports may also include `keyboard_ux` evidence with schema
`tetra.surface.keyboard-ux.v1` and level `production-keyboard-ux-v1`. That
evidence records graph focus order, overlay focus traps, roving focus, keyboard
activation, scoped shortcut conflict diagnostics, bounded undo/redo stacks, and
keyboard-only command palette/search/settings/editor scripts. The validator
rejects focusable elements without accessible names, overlay focus leaks,
shortcut conflicts that are not diagnosed, pointer-only actions, unknown
shortcuts, and undo/redo commands without a stack. This remains experimental
Block-system evidence.

Block asset reports for `examples/surface_block_assets.tetra` must include
`asset_pipeline` evidence with schema `tetra.surface.asset-pipeline.v1` and
level `production-asset-pipeline-v1`. The evidence records local-only
font/icon/image/vector manifests, sha256-before-decode, scoped safe decoders,
bounded-lru asset cache, missing fallback diagnostics, unsafe SVG rejection,
remote font rejection, network asset rejection, and oversized raster rejection.
This remains experimental Block-system evidence and does not claim network
assets, remote fonts, full SVG/CSS/SMIL, arbitrary image codecs, or production
Block support.

Block motion reports for `examples/surface_block_motion.tetra` must include
`animation_scheduler` evidence with schema
`tetra.surface.animation-scheduler.v1` and level
`production-animation-scheduler-v1`. The evidence records deterministic frame
scheduling, stable motion timeline, dirty-Block invalidation, lifecycle
stop-after-settle behavior, reduced-motion instant settle, frame timing,
visual delta evidence, target smoke rows, and validator rejections for hidden
animation loops, missing reduced motion, missing frame timing, unbounded frame
schedules, unchanged visual frames, and CSS animation parity claims. This
remains experimental Block-system evidence and does not claim a CSS animation
runtime or requestAnimationFrame parity. GPU compositor timing is outside this
scheduler evidence. It also carries no production Block support claim.

Developer inspector snapshots use schema
`tetra.surface.inspector-snapshot.v1` and level
`surface-inspector-json-mvp-v1`. They are generated from valid Surface runtime
reports by `surface-inspect` or `tetra surface inspect` and validated by
`validate-surface-inspector-snapshot`. The snapshot records Block tree, Morph
style resolution or Block-only style diagnostic, layout boxes, paint layers,
events, focus, accessibility, performance counters, and source locations. This
is experimental Block-system developer evidence only: docs-only trees, missing
source locations, missing layout boxes, missing accessibility views, missing
performance counters, and browser devtools parity claims are rejected. It does
not claim interactive devtools UI, perfect source maps, production profiler, or
production Block support.

Surface dev-loop evidence uses schema `tetra.surface.dev-loop.v1` and level
`surface-fast-dev-loop-v1`. The supported deterministic path is
`tetra new surface-app`, `tetra surface dev --once`, an actual source edit, a
second `tetra surface dev --once`, and `validate-surface-dev-report`. The
report must include source hash reload traces, operation rows for check/run/
inspect/package, the six required templates (`surface-minimal`,
`surface-dashboard`, `surface-form`, `surface-editor-shell`,
`surface-tray-app`, `surface-web-canvas`), and
`schema-compatible-owned-state-only` preservation rules. Hot reload without a
source-change trace is rejected. This remains experimental Block-system
developer evidence and does not claim Electron dev-server parity, React Fast
Refresh, CSS HMR, DOM hot reload, browser devtools parity, incompatible state
preservation, production packaging/signing, or production Block support.

Surface visual regression evidence uses schema
`tetra.surface.visual-regression.v1` and level
`surface-visual-golden-v1`. The supported deterministic path is
`surface-golden`, `validate-surface-visual-report`, and
`scripts/release/surface/visual-gate.sh`. The report must include the five
required scenes (`command-palette`, `dashboard`, `settings`, `editor`,
`glass`), source scene hashes, target, software RGBA renderer version,
baseline/current/diff PNG artifacts, frame checksums, font manifest hashes, and
asset manifest hashes. Screenshot-only evidence without a scene hash, missing
baselines, tampered checksums, and changed goldens without
`surface-visual-review-approved` are rejected. This remains experimental
Block-system regression evidence and does not claim Electron/Chromium pixel
parity, CSS browser rendering parity, or GPU compositor parity. It does not
promote the Block System support level.

Surface package distribution evidence uses schema
`tetra.surface.package-report.v1` and level
`surface-package-distribution-v1`. The supported deterministic path is
`scripts/release/surface/package-gate.sh`, `surface-package-report`, and
`validate-surface-package-report`. The report must include a scoped
`surface-linux-tar-v1` Linux package, `.tdx` Surface app package file,
asset manifest, permissions manifest, host-adapter metadata, install smoke,
launcher smoke, artifact hashes, and explicit Windows/macOS/update nonclaims.
Unsigned macOS production packages, omitted package assets, and updater claims
without channel signature verification are rejected. This is not Windows/macOS
production packaging, auto-update production, or multi-target desktop installer
parity.

Surface security/sandbox evidence uses schema
`tetra.surface.security-report.v1` and level `surface-security-sandbox-v1`.
The supported deterministic path is
`scripts/release/surface/security-gate.sh` and
`validate-surface-security-report`. The report must include
explicit-deny-by-default permissions, safe-local-assets-only asset sandboxing,
typed-host-abi-only IPC, capsule/package hash supply-chain checks, and
rejections for network/filesystem/clipboard host calls without permission,
unsafe SVG/font/image acceptance, user JavaScript, remote code execution,
packages without hashes, and untyped IPC. This is not browser plugin sandbox
parity, Node/Electron process sandbox parity, or arbitrary untrusted decoder
support.

Surface IPC/lifecycle evidence uses schema
`tetra.surface.ipc-lifecycle-report.v1` and level
`surface-ipc-lifecycle-v1`. The supported deterministic path is
`scripts/release/surface/ipc-lifecycle-gate.sh` and
`validate-surface-ipc-report`. The report must include app main,
single-owner UI isolate, supervised background services, owned message passing,
dispatcher-routed UI updates, Surface handle/frame/event actor transfer
rejection, borrowed payload rejection, untyped channel rejection, background UI
mutation without dispatcher rejection, and scoped crash-isolation policy. This
is not Electron main/renderer parity, process sandbox parity, or a broad crash
recovery claim.

Surface crash diagnostics evidence uses schema
`tetra.surface.crash-report.v1` and level `surface-crash-diagnostics-v1`.
The supported deterministic path is `scripts/release/surface/crash-gate.sh`
and `validate-surface-crash-report`. The report must include structured crash
diagnostics, source locations, sanitized diagnostic bundles, production error
hook, dev-only panic/error overlay, secret scrubbing, expected-negative/crash
separation, crash swallowed as pass rejection, secret leak rejection, missing
diagnostic bundle rejection, and unsurfaced error rejection. This is not
automatic crash recovery, telemetry upload, or Electron crash reporter
compatibility.

Surface i18n/localization evidence uses schema
`tetra.surface.i18n-report.v1` and level `surface-i18n-l10n-v1`. The supported
deterministic path is `scripts/release/surface/i18n-gate.sh` and
`validate-surface-i18n-report`. The report must include locale resources,
stable string IDs, number/date/plural formatting hooks, translation asset
packaging, LTR/RTL layout direction metadata, missing locale resource
rejection, silent fallback rejection, unsupported host localization rejection,
and full ICU/CLDR claim rejection. This is a full bidi shaping nonclaim, full
ICU/CLDR nonclaim, and not platform-native localization framework parity.

Surface performance/memory evidence uses schema
`tetra.surface.perf-report.v1` and level `surface-performance-memory-v1`. The
supported deterministic path is `scripts/release/surface/perf-gate.sh`,
`surface-perf-smoke`, and `validate-surface-perf-report`. The report must
include startup time, first frame time, steady frame p95, peak RSS, frame
allocations, layout/glyph/asset cache bytes, binary size, CPU idle power proxy,
input latency, animation frame jitter, baseline environment capture, bounded
cache evidence, and fair Electron comparison nonclaims. It must reject missing
baseline environment evidence, impossible performance numbers, unbounded
caches, unsupported faster-than-Electron claims, fastest UI framework claims,
and zero memory overhead claims.

Surface widget/component migration evidence uses schema
`tetra.surface.migration-report.v1` and level
`surface-widget-block-migration-v1`. The supported deterministic path is
`scripts/release/surface/migration-gate.sh` and
`validate-surface-migration-report`. The report must prove `lib.core.widgets`
stays a compatibility layer, Panel/Button/TextBox/StatusText map to Block/Morph
recipe equivalents, existing Surface v1 widget examples still pass, migration
diagnostics are available, and docs recommend Block/Morph for new production
UI. It must reject widgets declared as the core final architecture and breaking
Surface v1 examples without migration.

Surface production example-suite evidence uses schema
`tetra.surface.example-suite-report.v1` and level
`surface-production-example-suite-v1`. The supported deterministic path is
`scripts/release/surface/example-suite-gate.sh` and
`validate-surface-example-suite`. The report must prove ten realistic Surface
app shapes, executable `examples/surface_prod_*.tetra` examples,
Block/Morph-only production examples, scoped headless/linux-x64/wasm32-web
target coverage, event/state/accessibility/performance-budget evidence, and
ecosystem seed metadata. It must reject screenshot-only examples,
React/Electron/DOM runtime dependencies, widgets where Block/Morph is required,
missing app shapes, missing scoped target coverage, and toy visual-only
examples.

Surface production CI/release gate evidence uses schema
`tetra.surface.prod-gate-report.v1` and level
`surface-production-ci-release-gate-v1`. The supported deterministic path is
`scripts/release/surface/prod-gate.sh` and
`validate-surface-release-state --scope
PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`. The gate aggregates Surface release,
Block/Morph, visual, package, security, IPC/lifecycle, crash diagnostics,
i18n/localization, performance/memory, widget migration, production
example-suite, API stability, prod-claim governance, and artifact hashes. The
release workflow must upload `surface-production-final/**` and must not use
`continue-on-error` for production evidence. Missing production CI job, skipped
Linux/web production target counted as pass, and missing artifact hash manifest
are rejected. Windows and macOS remain beta/unsupported target-host boundaries
until their own production evidence exists.

Surface final same-commit audit evidence uses schema
`tetra.surface.prod-audit.v1` and level
`surface-prod-final-same-commit-audit-v1`. The public claim source is
`docs/release/surface_prod_release_audit.md`, and the validator is
`validate-surface-prod-audit`. The final verdict enum is
`PROD_STABLE_SCOPED/NEAR_READY_WITH_BLOCKERS/BETA_ONLY/EXPERIMENTAL_ONLY/FAIL`.
Promotion requires clean checkout rejection, report from different git head
rejection, missing unsupported-target nonclaims rejection, and public claim
generated from audit result.

Surface-vs-Electron public positioning evidence uses schema
`tetra.surface.electron-comparison-report.v1` and level
`surface-electron-comparison-method-v1`. The supported deterministic path is
`scripts/release/surface/electron-comparison-gate.sh`,
`surface-electron-comparison`, and
`validate-surface-electron-comparison-report`. Release notes may say
competitive with Electron in the supported scope, but public benchmark-
superiority claim rejection, cherry-picked hardware rejection, missing variance
rejection, unfair app shape rejection, missing environment rejection, and single-smoke
faster-than-Electron claim rejection are required.

Gate command:

```sh
bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget

bash scripts/release/surface/dev-loop-gate.sh \
  --report-dir reports/surface-prod/P24-dev-loop-gate

bash scripts/release/surface/visual-gate.sh \
  --report-dir reports/surface-prod/P25-visual-gate

bash scripts/release/surface/package-gate.sh \
  --report-dir reports/surface-prod/P26-package-gate

bash scripts/release/surface/security-gate.sh \
  --report-dir reports/surface-prod/P27-security-gate

bash scripts/release/surface/ipc-lifecycle-gate.sh \
  --report-dir reports/surface-prod/P28-ipc-lifecycle-gate

bash scripts/release/surface/crash-gate.sh \
  --report-dir reports/surface-prod/P29-crash-gate

bash scripts/release/surface/i18n-gate.sh \
  --report-dir reports/surface-prod/P30-i18n-gate

bash scripts/release/surface/perf-gate.sh \
  --report-dir reports/surface-prod/P31-perf-gate

bash scripts/release/surface/migration-gate.sh \
  --report-dir reports/surface-prod/P32-migration-gate

bash scripts/release/surface/example-suite-gate.sh \
  --report-dir reports/surface-prod/P33-example-suite-gate

bash scripts/release/surface/prod-gate.sh \
  --report-dir reports/surface-prod/final

go run -buildvcs=false ./tools/cmd/surface-electron-comparison \
  --out reports/surface-prod/final/surface-electron-comparison/surface-electron-comparison-report.json

go run -buildvcs=false ./tools/cmd/validate-surface-electron-comparison-report \
  --report reports/surface-prod/final/surface-electron-comparison/surface-electron-comparison-report.json

go run -buildvcs=false ./tools/cmd/validate-surface-prod-audit \
  --audit docs/release/surface_prod_release_audit.md \
  --expected-status PROD_STABLE_SCOPED
```

## Morph Capsule Status

`ui.surface-morph-capsule` is experimental in this release contract. It records
the Morph authoring layer over Block: scoped tokens, materials, affordances,
state lenses, motion presets, and recipes in `lib.core.morph` that expand into
Block graph evidence. Its `typed-style-graph-candidate-v1` style graph and
`tetra.surface.morph.authoring.v1` `production-recipe-authoring-v1` evidence
record 11 stable recipe families with declared inputs/slots/state/a11y
projections and reported Block-only expansions. These are scoped experimental
boundaries for replacing CSS cascade semantics inside Surface reports. Morph is
not current release support and is not production support.

The current gate is deterministic headless evidence only:

```sh
bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate
```

The gate writes `tetra.surface.morph.v1` and
`tetra.surface.morph.gate.v1` evidence for
`examples/surface_morph_command_palette.tetra`, validates same-commit report
state, and checks artifact hashes. The final Surface release gate records this
Morph gate as an experimental evidence dependency without promoting Morph to a
production Surface API.

## Unsupported

- macOS real-window Surface
- Windows real-window Surface
- wasm32-wasi Surface UI
- GPU renderer
- dynamic trait-object component ABI
- witness-table widget dispatch
- full rich text editor
- arbitrary native platform widgets
- React/DOM UI/user-JS app logic

## Release Target Matrix

| Target | Release status | Required evidence |
|---|---|---|
| `headless` | release-test-supported | deterministic runtime/text/toolkit plus `tetra.surface.accessibility-target.v1` deterministic inspector evidence |
| `linux-x64` | current | `linux-x64-release-window-v1` real Wayland shm window, native event pump, text/clipboard/IME, `tetra.surface.linux-host-adapter.v1`, app-shell ABI, toolkit, `tetra.surface.accessibility-target.v1` with Linux accessibility host bridge/probe evidence, `linux-x64-unpacked-binary-v1` packaging scope |
| `wasm32-web` | current | real browser canvas, `tetra.surface.browser-canvas-target.v1`, browser input, clipboard/IME, toolkit, `tetra.surface.accessibility-target.v1` with accessibility snapshot/mirror evidence |
| `macos-x64` | unsupported for Surface v1 | must not claim production; `tetra.surface.macos-target.v1` / `validate-surface-macos-target` records the nonclaim or beta target-host boundary |
| `windows-x64` | unsupported for Surface v1 | must not claim production; `tetra.surface.windows-target.v1` / `validate-surface-windows-target` records the nonclaim or beta target-host boundary |
| `wasm32-wasi` | unsupported for Surface UI | must not claim UI runtime |

## Release Status Vocabulary

Feature and report status models may use these lifecycle labels:

- `experimental`
- `release_candidate`
- `current`
- `unsupported`
- `legacy_compatibility`

Historical and non-Surface registries may still use existing future-planning
labels such as `planned` and `post-v1`; those labels do not constitute Surface
release evidence.

## Final Release Report Rules

Final Surface release summaries must include:

```json
{
  "status": "current",
  "experimental": false,
  "production_claim": true,
  "release_scope": "surface-v1-linux-web"
}
```

Unsupported target entries must remain non-current and non-production:

```json
{
  "status": "unsupported",
  "production_claim": false,
  "reason": "no real target-host Surface v1 evidence in this release"
}
```

The release validator must reject any production/current claim for
`macos-x64`, `windows-x64`, or `wasm32-wasi` until a future release contract
adds real target-host evidence for that target.
