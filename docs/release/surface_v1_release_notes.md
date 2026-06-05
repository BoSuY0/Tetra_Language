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

## Evidence Commands

```sh
bash scripts/release/surface/release-gate.sh \
  --report-dir /tmp/tetra-surface-release-v1-current

bash scripts/release/surface/gate.sh \
  --report-dir /tmp/tetra-surface-experimental-regression-current

bash scripts/release/safe-view-lifetime/gate.sh \
  --report-dir /tmp/tetra-safe-view-lifetime-surface-release-current
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
