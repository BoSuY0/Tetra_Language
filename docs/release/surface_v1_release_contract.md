# Tetra Surface v1 Release Contract

Status: current for surface-v1-linux-web scope after release gate passes.

This contract is the release-truth boundary for promoting Tetra Surface v1. It
does not promote every Surface experiment to production. A release/current claim
is valid only when release reports use `release_scope: surface-v1-linux-web`,
`status: current`, `experimental: false`, and `production_claim: true`, and the
release gate validates the required target evidence and artifact hashes.

## Claim Governance

Surface release docs and manifest rows use this tier vocabulary:

| Tier | Contract rule |
| --- | --- |
| `PROD_STABLE_SCOPED` | allowed only for the named `surface-v1-linux-web` scope after product-gate evidence and the final same-commit audit; it is not a broad Electron, React, CSS, DOM, Windows, macOS, GPU, rich-text, bidi, or screen-reader claim |
| `BETA_TARGET_HOST` | target-host evidence may exist outside the production scope, but it is not current Surface v1 production support |
| `EXPERIMENTAL` | Block System, Morph Capsule, visual infrastructure, and historical toolkit layers remain evidence tracks unless a later contract promotes them |
| `UNSUPPORTED` | target or feature has no current release support and no production target-host evidence |
| `NONCLAIM` | explicit release-language boundary for adjacent capabilities that validators must not allow to drift into support claims |

P28 docs governance is enforced by:

```sh
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-surface-claims --root .
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-v1
```

The product gate is mandatory product evidence. It is not the final
`PROD_STABLE_SCOPED` verdict; `docs/release/surface_product_readiness_audit.md`
and the final verdict are owned by P29.

## Machine-Readable Gate Contract

`scripts/release/surface/contracts/surface-release-v1.json` is the
machine-readable gate contract for the same Surface v1 docs/gate surface. It
uses `schema:"tetra.gate-contract.v1"` and `id:"surface-release-v1"`, lists 33
required reports, 33 CI artifacts, 41 ordered steps, and 14 validators, and
binds claim ids `surface_release_required_reports`, `crash_reporting`,
`surface_release_summary`, `artifact_hash_integrity`, `release_state_current`,
and `unsupported_target_nonclaim_evidence`. It also records nonclaim ids
`not_remote_ci_execution` and `not_unsupported_target_runtime_support`.

The shell release/product gates remain the evidence producers. The JSON
contract, dry-run plan, and validators prevent docs/manifest/claim drift; they
do not prove remote CI execution, do not promote macOS/Windows runtime support,
and do not mean `run-gate` executes the gate yet. Run the dry-run plan against
an empty report directory:

```sh
go run ./tools/cmd/run-gate --contract scripts/release/surface/contracts/surface-release-v1.json --report-dir reports/surface-product-v1 --dry-run --json
```

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
- scoped Linux app-shell permission model with default-deny filesystem/network
  policy, capability-checked IPC/process boundaries, and local hashed asset
  safety
- scoped Linux app-shell local performance budget evidence for startup, frame,
  memory/RSS/cache/framebuffer, binary size, and CPU/power proxy
- scoped developer fast rebuild evidence through `tetra surface dev`, with
  token/recipe/source changed rebuild steps and source diagnostics
- reference app suite evidence for ten Block/Morph product shapes across
  headless, linux-x64 real-window, and wasm32-web browser-canvas targets
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

The Block gate now requires each Block-system runtime report to include
`block_system.memory_budget` evidence. The budget is scoped to the reported
scene and records Block count, stress count, render/state loop counts,
framebuffer bytes, bounded paint/text/asset caches, and explicit nonclaims.
This budget evidence does not promote Block to production support and does not
stand in for an Electron comparison benchmark.

Gate command:

```sh
bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget
```

## Morph Capsule Status

`ui.surface-morph-capsule` is experimental in this release contract. It records
the Morph authoring layer over Block: scoped tokens, materials, affordances,
state lenses, motion presets, and recipes in `lib.core.morph` that expand into
Block graph evidence. It is not current release support and is not production
support.

The current gate is deterministic headless evidence only:

```sh
bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate
```

The gate writes `tetra.surface.morph.v1` and
`tetra.surface.morph.gate.v1` evidence for
`examples/surface_morph_command_palette.tetra`,
`examples/surface_morph_project_dashboard.tetra`,
`examples/surface_morph_settings.tetra`,
`examples/surface_morph_editor_shell.tetra`, and
`examples/surface_morph_glass_panel.tetra`, validates same-commit report state,
recipe expansions, token graph evidence, and artifact hashes. The final Surface
release gate records this Morph gate as an experimental evidence dependency
without promoting Morph to a production Surface API.

## Unsupported

- macOS real-window Surface
- Windows real-window Surface
- wasm32-wasi Surface UI
- GPU renderer
- dynamic trait-object component ABI
- witness-table widget dispatch
- full rich text editor
- arbitrary native platform widgets
- React, DOM-authored app UI trees, or user-JS app logic
- unrestricted filesystem or network access
- native permission prompts
- remote network asset fetching

## Release Target Matrix

| Target | Release status | Required evidence |
|---|---|---|
| `headless` | release-test-supported | deterministic runtime/text/toolkit/accessibility evidence |
| `linux-x64` | current | real Wayland shm window, native event pump, text/clipboard/IME, toolkit, accessibility bridge, `linux-app-shell-subset-v1`, `electron-feature-ledger-v1`, `surface-security-permission-v1`, `surface-performance-budget-v1`, and `surface-dev-workflow-v1` |
| `wasm32-web` | current | real browser canvas, browser input, clipboard/IME, toolkit, accessibility snapshot/mirror, `tetra.surface.browser-surface.v1` evidence with DOM host canvas only |
| `macos-x64` | unsupported for Surface v1 | `tetra.surface.target-host-status.v1` records `UNSUPPORTED` nonclaim evidence; build-only artifacts must not promote Surface runtime support |
| `windows-x64` | unsupported for Surface v1 | `tetra.surface.target-host-status.v1` records `UNSUPPORTED` nonclaim evidence; build-only artifacts must not promote Surface runtime support |
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
  "release_scope": "surface-v1-linux-web",
  "performance_budget": "surface-performance-budget-v1",
  "developer_fast_loop": "surface-dev-workflow-v1",
  "inspector": "surface-inspector-v1",
  "project_templates": "surface-template-smoke-v1",
  "reference_apps": "surface-reference-app-suite-v1",
  "surface_package": "surface-package-v1",
  "crash_reporting": "surface-crash-report-v1",
  "i18n_localization": "surface-i18n-v1",
  "widget_migration": "surface-widget-migration-v1"
}
```

`surface-performance-budget-v1` is local deterministic budget evidence, not an
external benchmark result or unsupported Electron speed comparison. The app-shell
runtime report must include startup-to-first-frame, p50/p95 frame build/present,
memory/RSS/cache/framebuffer, binary size, CPU/power proxy, bounded-cache, and
mandatory `peak_rss_bytes` fields.

`surface-dev-workflow-v1` is developer workflow evidence, not a hot reload
claim. The release gate must include `surface-dev-workflow.json` with
`tetra.surface.dev-workflow.v1`, `command:"tetra surface dev"`,
`mode:"fast-rebuild"`, warm-cache rebuild evidence, token/recipe/source changed
rebuild steps, source diagnostics, and negative guards against Electron dev
server or React Fast Refresh claims.

`surface-inspector-v1` is static tool evidence, not a browser devtools or DOM
runtime claim. The release gate must include `surface-inspector.json` with
`tetra.surface.inspector.v1`, source locations, input report coverage, sections
for Block tree, Morph tokens, layout, paint, accessibility, event routes, focus,
and perf counters, hidden-state scanning, and negative guards against DOM
runtime, browser devtools, React devtools, and hidden state dependencies.

`surface-template-smoke-v1` is onboarding evidence for generated Surface
projects. The release gate must include `surface-template-smoke.json` with
`tetra.surface.template-smoke.v1`, all seven template kinds, generated app
check/build/run rows, inspector evidence, visual diff evidence, package
artifacts, and negative guards against React/Electron/runtime imports,
DOM-authored app UI trees, CSS runtime dependencies, core widget primitives,
platform widgets, and user JavaScript app logic.

`surface-reference-app-suite-v1` is product-shape evidence for ten polished
Block/Morph reference apps: command palette, settings, dashboard, editor shell,
file manager/list-detail, dialog/notification, localized form,
accessibility-heavy form, multi-window notes, and migration. The release gate
must include `surface-reference-apps.json` with
`tetra.surface.reference-app-suite.v1`, stable Morph recipe expansion to Block,
check/build/run evidence for every source, headless/linux/web target entries,
visual diff evidence, token/theme conformance, layout evidence, interaction
traces, accessibility snapshots, performance budgets, and artifact hashes.
Negative guards must reject screenshot-only evidence, missing interaction,
missing accessibility, missing performance, React/Electron/runtime imports,
DOM-authored app UI trees, CSS runtime dependencies, and widget usage outside
the migration compatibility example.

`surface-package-v1` is packaging and update-story evidence for the bounded
Linux/web release scope. The release gate must include `surface-package.json`
with `tetra.surface.package.v1`, `surface-app-package-v1` manifests,
linux-x64 and wasm32-web tar.gz artifacts, local asset sha256 hashes, installed
linux-x64 package execution for the default reference app or an explicitly
named product-slice app such as `studio-shell`, web bundle HTML/wasm/
compiler-owned loader output, explicit expected app-state exit code when it is
nonzero, and a hash-pinned
`tetra.surface.update-channel.v1` manifest. Negative guards must reject
docs-only package claims, remote asset fetching, signing or notarization claims
without platform evidence, and automatic runtime or network update claims
without updater evidence.

`surface-crash-report-v1` is crash recovery and error-reporting evidence for
the bounded Linux/web release scope. The release gate must include
`surface-crash-report.json` with `tetra.surface.crash-report.v1`, linux-x64
command failure, host crash diagnostic capture, restart/recovery scenarios,
redacted `tetra.surface.diagnostic.v1` artifacts, bounded local trace/log
collection, `surface-non-user-data-diagnostics-v1` privacy policy evidence, and
`scoped-linux-x64-process-restart-v1` before/report/after proof. Negative
guards must reject user data leaks, clipboard/user-text/env/home capture,
network upload, Electron crash reporter dependency, docs-only crash claims, and
restart claims without evidence.

`surface-i18n-v1` is internationalization and localization evidence for the
bounded Linux/web release scope. The release gate must include
`surface-i18n.json` with `tetra.surface.i18n.v1`, bounded string tables,
`uk-UA` locale selection with `en-US` fallback, missing-key diagnostics,
deterministic formatting hooks, localized-form reference app execution, and an
RTL placeholder nonclaim. Validators must reject full ICU, full bidi shaping,
RTL production text-layout, third-party intl runtime, platform locale dependency,
docs-only localization, and silent missing-key fallback claims.

`surface-widget-migration-v1` is widget migration compatibility evidence for
the bounded Linux/web release scope. The release gate must include
`surface-widget-migration.json` with
`tetra.surface.widget-migration.v1`, the exact `lib.core.widgets` Surface v1
release widget set, Panel/Button/TextBox equivalence rows, Morph recipes that
resolve to Block, migration reference app execution, and negative guards
against future core widget primitive promotion, primary future-widget-core
claims, breaking API changes, docs-only migration, and platform toolkit/runtime
claims.

Unsupported target entries must remain non-current and non-production:

```json
{
  "schema": "tetra.surface.target-host-status.v1",
  "target": "macos-x64 or windows-x64",
  "status": "unsupported",
  "tier": "UNSUPPORTED",
  "production_claim": false,
  "target_host_evidence": false,
  "build_only_promotion": false,
  "reason": "no real target-host Surface v1 evidence in this release"
}
```

The release validator must reject any production/current claim for
`macos-x64`, `windows-x64`, or `wasm32-wasi` until a future release contract
adds real target-host evidence for that target. For `macos-x64` and
`windows-x64`, the release state also requires target-host status JSON artifacts
and rejects Linux substitute evidence or build-only promotion.
