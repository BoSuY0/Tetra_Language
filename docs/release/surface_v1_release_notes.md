# Tetra Surface v1 Release Notes

Status: current for `surface-v1-linux-web` after the Surface v1 release gate
passes. Headless Surface is a release evidence target.

## Current

- Pure-Tetra Surface apps over the tiny Surface Host ABI.
- Headless deterministic release evidence target.
- Linux-x64 real-window Surface through the Wayland shm RGBA release path.
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
  and recipes that expand into Block evidence for
  `examples/surface_morph_command_palette.tetra`. The Morph gate is
  deterministic headless evidence only and carries no production Morph claim.
- Text/input baseline for UTF-8 byte storage, caret, selection,
  clipboard read/write, copy/paste, and IME/composition traces.
- Accessibility metadata plus platform bridge evidence for supported Linux
  and web targets.
- Strict release validators, release-state validation, and artifact hashes.

## Unsupported

- macOS real-window Surface.
- Windows real-window Surface.
- wasm32-wasi Surface UI runtime.
- GPU renderer.
- Dynamic trait-object widgets and witness-table component dispatch.
- Arbitrary native platform widgets.
- DOM/React/user-JS application UI.
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

bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget

bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate

bash scripts/release/surface/gate.sh \
  --report-dir reports/surface-ui-production-final/surface-experimental-regression

bash scripts/release/safe-view-lifetime/gate.sh \
  --report-dir reports/surface-ui-production-final/safe-view-lifetime
```

The release gate is the source of truth for the final current claim. Reports
remain evidence, not modes; unsupported target claims remain invalid until
target-specific evidence exists.

The living release audit is `docs/release/surface_v1_release_audit.md`; it
records which release checklist rows are proven now and which rows remain
pending for later sections.

## Known Limits

Surface v1 is a bounded release, not a general Qt/Flutter/browser framework
replacement. It proves the linux-x64 real-window and wasm32-web browser-canvas
release paths plus headless evidence. Broader platform targets, richer text,
GPU rendering, dynamic widget dispatch, and full platform accessibility remain
post-release work.
