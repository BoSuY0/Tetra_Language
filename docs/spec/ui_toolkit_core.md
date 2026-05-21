# UI Toolkit Core

Status: current post-v0.4 production promotion for the platform-independent
`tetra.ui.toolkit.v1` contract/runtime core.

This document defines the internal toolkit core promoted after the bounded
`tetra.ui.v0.4.0` metadata/runtime surface. It is not a GTK, Qt, OS widget
backend, Windows/macOS GUI runtime, hosted UI service, or full cross-platform UI
claim. Platform backends must still provide their own runtime-backed reports
before they can claim production parity.

## Contract

The compiler keeps the compatibility `tetra.ui.v0.4.0` path and additionally
emits deterministic `<output>.ui.toolkit.json` artifacts with schema
`tetra.ui.toolkit.v1` whenever checked UI declarations are present.

The toolkit contract covers these selected model families:

- Widget model: window, root, panel/container, text, label, button, input,
  checkbox/toggle, select/dropdown, list, table/list-grid, dialog/modal,
  menu/menu item, spacer, and divider.
- Layout model: stack, row, column, grid, flex-like sizing constraints,
  padding/margin/gap metadata, min/max/preferred size metadata, stable bounds,
  and overflow/scroll metadata.
- Style model: color, background, border, text style, and deterministic
  resolution for enabled, disabled, visible, focused, selected, and error
  states. This is toolkit-core style metadata, not platform-native styling.
- Accessibility model: role, label, description, focus order, state metadata,
  and keyboard activation metadata.
- Event model: click, activate, focus, blur, input, change, select, submit,
  key, timer, async command completion, redraw/update, and error recovery.
- State/update model: scalar binding, list/table binding, two-way input binding,
  command operations, deterministic update order, and stable diagnostics for
  unsupported operations.

## Runtime Core

The production core is backend-independent. The smoke runtime constructs a real
widget tree, measures and places layouts, dispatches events, applies state
transitions, records widget updates, schedules timer/async/redraw behavior,
traverses focus metadata, projects accessibility metadata, and exercises command
failure and panic recovery paths.

Native shell sidecars and web UI evidence may support diagnostics, but the
toolkit-core claim requires `tetra.ui.toolkit.v1` runtime traces and cannot be
proved by metadata-only, preview-only, build-only, native-shell sidecar-only, or
web-only artifacts.

## Evidence

Fresh production evidence for this wave lives under `reports/ui-toolkit-core`.
The dedicated gate is:

`bash scripts/release/post_v0_4/ui-toolkit-core-production-gate.sh --report-dir reports/ui-toolkit-core`

The required smoke report is `reports/ui-toolkit-core/ui-toolkit-core.json` and
is validated by:

`go run ./tools/cmd/validate-ui-toolkit-core --report reports/ui-toolkit-core/ui-toolkit-core.json`

The validator rejects docs-only, metadata-only, preview-only, runtime-less,
native-shell sidecar-only, web-only, build-only, fake, mock, placeholder,
missing widget execution, missing event dispatch, missing state transition,
missing layout/focus/accessibility evidence, and missing runtime artifact paths.

## Boundaries

This promotion does not claim:

- GTK/Qt/OS platform backend production.
- Windows/macOS GUI production.
- Full cross-platform UI.
- Platform accessibility API integration.
- EcoNet or hosted TetraHub.

Those surfaces remain separate promotion waves with their own target-host
runtime evidence, validators, artifact hashes, and gates.
