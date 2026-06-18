# Tetra Surface And Electron: Bounded Product-Slice Comparison

This document explains what the current Tetra Surface product slice can and cannot claim against an
Electron-style desktop app stack.

The claim is intentionally narrow:

> Tetra Surface can ship a bounded Linux/web desktop-style app slice without Electron, Chromium as
> an app runtime, React, DOM-authored application UI, CSS runtime, user JavaScript app logic, or
> platform-native widget UI.

This is not an Electron API compatibility claim, not an all-platform desktop claim, and not a
native-widget parity claim.

## Evidence Boundary

Primary product-slice evidence:

- Flagship source: `examples/surface/migration/surface_migration_tetra_control_center.tetra`
- Flagship contract: `examples/projects/tetra_control_center/docs/surface-flagship-contract.md`
- Headless runtime: `reports/surface-product-slice/flagship/headless-block-system.json`
- Linux real-window runtime:
  `reports/surface-product-slice/flagship/linux-x64-real-window-block-system.json`
- Web browser-canvas runtime:
  `reports/surface-product-slice/flagship/wasm32-web-browser-canvas-block-system.json`
- Developer loop: `reports/surface-product-slice/dev-workflow-flagship/surface-dev-workflow.json`
- Package and update story:
  `reports/surface-product-slice/package-flagship-task6/surface-package.json`
- Template onboarding:
  `reports/surface-product-slice/template-smoke-task5/surface-template-smoke.json`

Release-level Surface evidence and nonclaims remain governed by
`docs/release/surface/surface_product_readiness_audit.md`, `docs/spec/surface/surface_v1.md`, and
`docs/spec/core/current_supported_surface.md`.

## Matrix

### App UI Authoring

- Surface status: Green.
- Evidence: flagship source uses Surface, Block, and Morph directly.
- Boundary: no DOM-authored app UI tree, React runtime, CSS runtime, or user
  JavaScript app logic.

### Linux Windowed App Slice

- Surface status: Green.
- Evidence: Linux real-window flagship report and app-shell release evidence.
- Boundary: scoped Linux evidence only; no macOS or Windows production
  target-host claim.

### Web Delivery

- Surface status: Green.
- Evidence: `wasm32-web` browser-canvas flagship report and package web bundle.
- Boundary: browser canvas target only; no DOM application UI ownership claim.

### Morph Recipe Evidence

- Surface status: Amber.
- Evidence: Morph gate and flagship recipe evidence cover app shell, toolbar,
  split pane, status bar, command item, settings form, log row, empty state,
  and error panel.
- Boundary: Morph remains bounded evidence over Block, not a promoted
  production widget toolkit.

### Product-Shaped Flagship

- Surface status: Green.
- Evidence: Tetra Studio Shell source covers navigation, content panels,
  command palette, settings, logs/output, status bar, dialog row, error/retry
  surface, and app-shell state.
- Boundary: helper or web-hosted Control Center sidecars are integration
  plumbing, not the core Surface UI claim.

### Developer Loop

- Surface status: Green.
- Evidence: `tetra.surface.dev-workflow.v1` records initial, warm-cache,
  token-change, recipe-change, and source-change rebuild steps.
- Boundary: fast rebuild evidence only; no hot reload, React Fast Refresh, or
  DOM hot-reload claim.

### Packaging

- Surface status: Green.
- Evidence: `tetra.surface.package.v1` records linux-x64 and wasm32-web
  flagship packages, local assets, install/run evidence, web bundle evidence,
  artifact hashes, and a hash-pinned update-channel manifest.
- Boundary: signing, notarization, automatic runtime updates, and network
  update fetching remain nonclaims.

### Templates

- Surface status: Green.
- Evidence: `studio-shell` template is generated, checked, built, run,
  inspected, visually tested, and packaged by template smoke.
- Boundary: template packaging is local smoke evidence, not an app-store
  distribution claim.

### Inspector And Static Evidence

- Surface status: Green.
- Evidence: Surface inspector and product gates validate
  Block/Morph/layout/paint/accessibility/event/focus/perf evidence.
- Boundary: static inspector output is not browser devtools or React devtools.

### Accessibility Metadata And Bridge

- Surface status: Amber.
- Evidence: existing Surface gates record accessibility metadata and scoped
  bridge evidence for supported targets.
- Boundary: full screen-reader validation remains a nonclaim.

### Clipboard And IME

- Surface status: Amber.
- Evidence: Linux app-shell ledger carries scoped clipboard and IME evidence.
- Boundary: cross-platform clipboard/IME parity is outside this slice.

### Dialogs, File Picker, Notifications, Tray, Deep Links

- Surface status: Amber/Red.
- Evidence: app-shell ledger uses blocked-pass or nonclaim rows where target
  evidence is absent.
- Boundary: these rows must not be described as supported until target
  evidence exists.

### Crash And Error Surfaces

- Surface status: Amber.
- Evidence: flagship UI has an error/retry surface, and release smoke records
  bounded crash/error diagnostics for a reference app.
- Boundary: no Electron crash reporter dependency, network upload, or full
  crash service claim.

### Rich Text And Bidi

- Surface status: Red NONCLAIM.
- Evidence: no product-slice evidence.
- Boundary: no full rich text or full bidi claim.

### Native Platform Widgets

- Surface status: Red NONCLAIM.
- Evidence: Surface uses Block/Morph and compatibility widgets only where
  documented.
- Boundary: no native widget parity claim.

### GPU Renderer Parity

- Surface status: Red NONCLAIM.
- Evidence: no product-slice evidence.
- Boundary: no GPU renderer parity claim.

### Electron APIs

- Surface status: Red NONCLAIM.
- Evidence: no product-slice evidence for Electron API compatibility.
- Boundary: no Electron API compatibility claim.

### macOS Surface App

- Surface status: Red NONCLAIM.
- Evidence: no product-slice target-host evidence.
- Boundary: no macOS Surface production support is claimed for this slice.

### Windows Surface App

- Surface status: Red NONCLAIM.
- Evidence: no product-slice target-host evidence.
- Boundary: no Windows Surface production support is claimed for this slice.

### Production Signing And Notarization

- Surface status: Red NONCLAIM.
- Evidence: package report records signing and notarization as nonclaims.
- Boundary: no signing or notarization claim without platform evidence.

### Automatic Network Updates

- Surface status: Red NONCLAIM.
- Evidence: package report records a hash-pinned local update-channel manifest
  only.
- Boundary: no automatic network updater claim.

## Practical Reading

Use Surface for this slice when the app can live inside the current Surface-owned UI model:
Block/Morph composition, local package assets, linux-x64 and wasm32-web targets, and explicit
nonclaims for unsupported platform services.

Use Electron or another mature desktop shell when the product needs broad Electron API
compatibility, existing Node/Electron plugin ecosystems, mature cross-platform native integration,
production signing/notarization workflows, automatic network updates, native widget parity, full
rich text, full bidi, or full screen-reader validation today.

## Guardrails

Public wording should keep these constraints:

- Say "bounded Linux/web app slice" or "bounded Electron alternative".
- Do not say "Electron replacement", "Electron parity", or "Electron API compatible".
- Do not say "React replacement" or "CSS replacement"; say the flagship source does not use those
  runtimes.
- Do not promote Morph to production support; describe Morph as bounded recipe evidence over Block.
- Keep macOS, Windows, GPU renderer parity, native widgets, rich text, full bidi, full screen-reader
  validation, signing/notarization, and automatic network updates as `NONCLAIM` rows until new
  target evidence exists.
