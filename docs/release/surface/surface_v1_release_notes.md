# Tetra Surface v1 Release Notes

Status: current for `surface-v1-linux-web` after the Surface v1 release gate
passes. Headless Surface is a release evidence target.

## Current

- Pure-Tetra Surface apps over the tiny Surface Host ABI.
- Headless deterministic release evidence target.
- Linux-x64 real-window Surface through the Wayland shm RGBA release path. This
  is `surface-v1-linux-web` release/probe evidence, not Native Surface Host v1
  completion.
- wasm32-web browser-canvas Surface through compiler-owned browser boot,
  canvas readback, input, clipboard/composition, and accessibility mirror
  evidence.
- `lib.core.widgets` production widget subset:
  Text, Label, StatusText, Button, TextBox, Checkbox, Row, Column, Panel,
  Stack, Scroll, and Spacer.
- Experimental `ui.surface-block-system` architecture direction: Block-first
  Surface composition where Block is the core Surface primitive. Same-commit
  `tetra.surface.block-system.gate.v1` reports under
  `reports/surface-block/p18-budget` cover headless, linux-x64 real-window, and
  wasm32-web browser-canvas Block evidence plus `block_system.memory_budget`.
  Current widget helper names remain recipes/compatibility over Block. This is
  not production support and carries no production Block claim in this release.
- Experimental `ui.surface-morph-capsule` evidence layer: `lib.core.morph`
  defines capsule tokens, materials, affordances, state lenses, motion presets,
  and recipes that expand into Block evidence for the five
  `examples/surface_morph_*.tetra` reference apps. The Morph gate is
  deterministic headless evidence only and carries no production Morph claim.
- Text/input baseline for UTF-8 byte storage, caret, selection,
  clipboard read/write, copy/paste, and IME/composition traces.
- Accessibility metadata plus platform bridge evidence for supported Linux
  and web targets.
- Scoped Linux app-shell security permission model:
  `surface-security-permission-v1` default-deny filesystem/network policy,
  capability-checked IPC/process boundaries, scoped clipboard policy, and
  local hashed asset/font/image safety.
- Scoped Linux app-shell performance budget model:
  `surface-performance-budget-v1` local startup/frame/memory/RSS/cache/
  framebuffer/binary-size/CPU-proxy evidence with bounded caches and mandatory
  `peak_rss_bytes`.
- Scoped developer fast loop: `tetra surface dev` writes
  `tetra.surface.dev-workflow.v1` / `surface-dev-workflow-v1` evidence for
  initial build, warm-cache rebuild, token/recipe/source changed rebuilds, and
  positioned source diagnostics. This is fast rebuild evidence, not a hot reload
  or React Fast Refresh claim.
- Static Surface inspector evidence:
  `tetra.surface.inspector.v1` / `surface-inspector-v1` reports expose Block
  tree, Morph tokens, layout, paint, accessibility, event routes, focus, perf
  counters, source locations, input report coverage, and hidden-state scan
  results. Optional HTML output is a static tool report, not browser devtools,
  React devtools, or DOM runtime UI.
- Surface project template evidence:
  `tetra.surface.template-smoke.v1` / `surface-template-smoke-v1` reports cover
  `tetra new surface-app` generation for command palette, settings, dashboard,
  editor shell, studio shell, multi-window notes, and web-canvas starts, including check,
  build, run, inspector, visual diff, and tar package evidence.
- Surface reference app suite evidence:
  `tetra.surface.reference-app-suite.v1` /
  `surface-reference-app-suite-v1` reports cover command palette, settings,
  dashboard, editor shell, file manager/list-detail, dialog/notification,
  localized form, accessibility-heavy form, multi-window notes, and migration
  apps. Each app checks, builds, runs, resolves Morph recipes to Block, and
  records visual, interaction, accessibility, performance, token/theme, layout,
  and artifact-hash evidence for headless, linux-x64 real-window, and
  wasm32-web browser-canvas targets.
- Surface packaging and update evidence:
  `tetra.surface.package.v1` / `surface-package-v1` reports cover linux-x64 and
  wasm32-web tar.gz packages for the command-palette reference app and the
  product-slice `studio-shell` flagship source, local asset hashes, package
  manifests, installed linux-x64 package execution, web bundle
  HTML/wasm/compiler-owned loader output, and a hash-pinned update channel
  manifest. The flagship install smoke records its current expected app-state
  exit code explicitly. Signing, notarization, automatic runtime updates, and
  network update fetching remain nonclaims without platform/runtime evidence.
- Surface/Electron comparison guidance:
  `docs/user/surface_electron_comparison.md` defines the bounded green/amber/red
  product-slice story and keeps Electron API compatibility, all-platform parity,
  GPU renderer parity, native widgets, signing/notarization, and automatic
  network updates as explicit nonclaims.
- Surface crash recovery and error-reporting evidence:
  `tetra.surface.crash-report.v1` / `surface-crash-report-v1` reports cover
  bounded linux-x64 command failure, host crash diagnostic capture, redacted
  local diagnostic artifacts, bounded trace/log collection, and scoped restart
  evidence for the command-palette reference app. User data leaks, network
  upload, Electron crash reporter dependency, docs-only crash claims, and
  restart claims without before/report/after evidence remain rejected.
- Surface internationalization and localization evidence:
  `tetra.surface.i18n.v1` / `surface-i18n-v1` reports cover bounded string
  tables, `uk-UA` locale selection, `en-US` fallback, missing-key diagnostics,
  deterministic formatting hooks, localized-form reference app execution, and
  an RTL placeholder nonclaim. Full ICU, full bidi shaping, RTL production text
  layout, third-party intl runtime, platform locale dependency, docs-only
  localization, and silent missing-key fallback remain rejected.
- Surface widget migration compatibility evidence:
  `tetra.surface.widget-migration.v1` / `surface-widget-migration-v1` reports
  keep `lib.core.widgets` supported for Surface v1, preserve the release widget
  set, prove Panel/Button/TextBox equivalence rows against Morph recipes that
  resolve to Block, and reject future core widget primitive promotion, breaking
  API changes, docs-only migration, and platform toolkit/runtime claims.
- Claim-governance vocabulary: `PROD_STABLE_SCOPED`, `BETA_TARGET_HOST`,
  `EXPERIMENTAL`, `UNSUPPORTED`, and `NONCLAIM`. These tiers keep current
  Linux/web support, beta/future target paths, experimental Block/Morph
  evidence, unsupported targets, and explicit nonclaims separate. The scoped
  product gate is required evidence, but it is not the P29 final
  `PROD_STABLE_SCOPED` verdict.
- Strict release validators, release-state validation, and artifact hashes.

## Unsupported

- macOS real-window Surface.
- Windows real-window Surface.
- wasm32-wasi Surface UI runtime.
- GPU renderer.
- Dynamic trait-object widgets and witness-table component dispatch.
- Arbitrary native platform widgets.
- DOM/React/user-JS application UI.
- Unrestricted filesystem/network access, native permission prompts, or remote
  network asset fetching from Surface app-shell defaults.
- External benchmark results or unsupported Electron speed comparison claims.
- Hot reload, Electron dev server, or React Fast Refresh claims for Surface
  apps.
- Full rich text editor or IDE-grade text editing.
- Full AT-SPI or screen-reader support without separate probe artifacts.

## Migration

Existing `ui.metadata-v1` apps remain on the legacy metadata compatibility
surface. New release-supported Surface apps should use ordinary Tetra structs,
`lib.core.component`, `lib.core.widgets`, `lib.core.text`,
`lib.core.accessibility`, and `lib.core.style`.
Future Block System apps should use Block configuration as the main visual
material. Current Block evidence is scoped and experimental; release-supported
Surface v1 apps still use the bounded `lib.core.widgets` subset.

## Evidence Commands

```sh
bash scripts/release/surface/release-gate.sh \
  --report-dir reports/surface-ui-production-final/surface-release-v1

bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-v1

bash scripts/release/surface/surface-dev-workflow-smoke.sh \
  --report-dir reports/surface-dev-workflow/gate

bash scripts/release/surface/surface-inspector-smoke.sh \
  --report-dir reports/surface-inspector/gate

bash scripts/release/surface/surface-template-smoke.sh \
  --report-dir reports/surface-templates/gate

bash scripts/release/surface/surface-reference-apps-smoke.sh \
  --report-dir reports/surface-reference-apps/gate

bash scripts/release/surface/surface-crash-report-smoke.sh \
  --report-dir reports/surface-crash-report/gate

bash scripts/release/surface/surface-i18n-smoke.sh \
  --report-dir reports/surface-i18n/gate

bash scripts/release/surface/surface-widget-migration-smoke.sh \
  --report-dir reports/surface-widget-migration/gate

bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget

bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate

bash scripts/release/surface/gate.sh \
  --report-dir reports/surface-ui-production-final/surface-experimental-regression

bash scripts/release/safe-view-lifetime/gate.sh \
  --report-dir reports/surface-ui-production-final/safe-view-lifetime
```

The release gate is the source of truth for release evidence and current
candidate scope. Reports remain evidence, not modes; unsupported target claims
remain invalid until target-specific evidence exists. P29 owns the final
same-commit verdict.

The machine-readable Surface v1 gate contract is
`scripts/release/surface/contracts/surface-release-v1.json`
(`schema:"tetra.gate-contract.v1"`, `id:"surface-release-v1"`). It mirrors the
validation/report/upload contract with 33 required reports, 33 CI artifacts, 41
ordered steps, 14 validators, claim ids `surface_release_required_reports`,
`crash_reporting`, `surface_release_summary`, `artifact_hash_integrity`,
`release_state_current`, and `unsupported_target_nonclaim_evidence`, plus
nonclaim ids `not_remote_ci_execution` and
`not_unsupported_target_runtime_support`. The shell release/product gates still
produce evidence; `run-gate` is currently a dry-run plan and does not prove
remote CI execution or macOS/Windows runtime support. Run it against an empty
report directory:

```sh
go run ./tools/cmd/run-gate --contract scripts/release/surface/contracts/surface-release-v1.json --report-dir reports/surface-product-v1 --dry-run --json
```

The living release audit is `docs/release/surface_v1_release_audit.md`; it
records which release checklist rows are proven now and which rows remain
pending for later sections.

## Known Limits

Surface v1 is a bounded release, not a general Qt/Flutter/browser framework
replacement. It proves the linux-x64 real-window and wasm32-web browser-canvas
release paths plus headless evidence. It does not prove
`tetra.surface.native-host.v1` until the strict native-host gate has a compiled
app, official Wayland host, real pointer/key/close evidence, app-produced
frames, and no pre-rendered delivery path. Broader platform targets, richer
text, GPU rendering, dynamic widget dispatch, and full platform accessibility
remain post-release work.
