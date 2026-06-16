# Tetra Studio Shell Surface Flagship Contract

Status: product-slice contract for
`docs/plans/2026-06-13-surface-electron-competitor-product-slice.md`.

## Product Name And Scope

The public flagship slice is **Tetra Studio Shell**. It reuses the existing
Tetra Control Center project as the first real app domain and
`examples/surface_migration_tetra_control_center.tetra` as the pure Surface
migration seed, but the product claim is broader than one laptop utility:

> Tetra Surface can ship a bounded Linux/web desktop-style app without Electron,
> Chromium as an app runtime, React, DOM-authored application UI, CSS runtime,
> user JavaScript app logic, or platform-native widget UI.

The existing Tetra Control Center remains an integration demo for hardware data,
safe helper boundaries, and web-hosted historical UI. The flagship claim is
only for the Surface-owned UI path described below.

## Surface-Owned Core UI

The flagship core UI must be authored in Tetra through Surface, Block, and
Morph:

- app shell frame with title/status regions;
- sidebar or navigation rail;
- dashboard/content panels;
- profile/project/action list;
- command palette surface;
- settings form;
- logs/output panel;
- diagnostic/error surface with retry or recovery action;
- status bar;
- modal/dialog surface or explicit blocked-pass dialog row;
- app-shell state for lifecycle, menu, multi-window notes, clipboard/IME where
  existing scoped evidence supports it.

The Surface-owned UI must not depend on `web/app.mjs`, DOM-authored application
views, CSS runtime layout, React, Electron, user JavaScript application logic,
Qt/GTK/native widgets, or the Python helper for rendering.

## Integration Plumbing

The existing sidecars stay allowed only as integration plumbing:

- `backend/tcc_backend.py` may provide local read-only or dry-run data for the
  hardware Control Center domain.
- `web/app.mjs` may remain as the historical integration demo and browser host
  for the existing project, but it is not evidence for the Surface flagship UI.
- Helper APIs must stay allowlisted and must not become arbitrary shell
  execution, remote asset fetching, or network update proof.

Any docs, reports, or comparison copy must label these boundaries explicitly.

## Required Screens

The first flagship implementation must expose these user-visible surfaces in
the Surface-owned path:

| Surface | Required content | Required interaction |
| --- | --- | --- |
| Dashboard | summary cards/panels, current state, health/status rows | navigation focus and refresh/update action |
| Profiles / Actions | selectable action rows such as Quiet/Balanced/Performance/Custom or project tasks | select action and stage/apply dry-run command |
| Command Palette | query field, command rows, primary action affordance | open/select command, update selected command |
| Settings | labeled fields/toggles and save/reset actions | edit/toggle, save/reset state |
| Logs / Output | ordered log rows, selected row, status or build output | select row, append or refresh output |
| Diagnostics / Error | unsupported/supported rows, local diagnostic state, retry/recovery action | retry or dismiss/recover |
| Status Bar | target, mode, dirty/build/report state | update after scripted interactions |

## App-Shell Feature Rows

The flagship contract inherits the scoped Linux app-shell ledger:

| Feature | Contract status |
| --- | --- |
| window lifecycle | scoped Linux evidence required |
| multi-window notes | scoped Linux evidence required or explicit notes surface |
| app menu | scoped adapter evidence required |
| clipboard | scoped text clipboard evidence where current runtime supports it |
| IME/composition | scoped baseline evidence where current runtime supports it |
| accessibility bridge/metadata | scoped evidence required |
| crash/error diagnostics | local redacted diagnostic evidence required |
| dialog/file picker/notification/tray/deep link | blocked-pass or nonclaim rows until target evidence exists |

The flagship must not turn blocked rows into claimed support without new target
host evidence and validator coverage.

## Target Evidence

The flagship must have current evidence for:

- headless deterministic Surface evidence;
- linux-x64 real-window evidence when a display host is available;
- wasm32-web browser-canvas evidence through compiler-owned canvas plumbing;
- `tetra.surface.dev-workflow.v1` fast rebuild evidence for token, recipe, and
  source changes;
- package evidence for linux-x64 and wasm32-web artifacts;
- claim scanner, docs, manifest, and artifact-hash evidence.

If a host dependency is unavailable, the report must be blocked or partial; do
not promote headless evidence to a real-window or browser-canvas claim.

## Morph / Block Authoring Boundary

Product UI should be written mostly through named Morph recipes that expand to
Block evidence. Required recipe families:

- app shell / region panel;
- sidebar or nav item;
- toolbar / action control;
- tabs;
- split pane;
- status bar;
- command item;
- settings form and field;
- log row;
- metric tile;
- toast/notification-like local surface;
- modal/dialog surface;
- empty state;
- error panel.

`lib.core.widgets` may remain compatibility evidence but must not become the
future core primitive set for this flagship slice.

## Nonclaims

This contract does not claim:

- full Electron API compatibility;
- no macOS or Windows Surface production support;
- GPU renderer parity;
- native platform widget parity;
- React compatibility;
- CSS cascade/runtime compatibility;
- DOM-authored application UI;
- user JavaScript application logic;
- full rich text, full bidi, or full screen-reader validation;
- signing, notarization, automatic network updates, or remote asset fetching.

## Verification Gates

Task 1 contract verification starts with:

```sh
./tetra check examples/surface_migration_tetra_control_center.tetra
bash examples/projects/tetra_control_center/scripts/smoke.sh
```

Later implementation tasks must add flagship-specific runtime, dev-loop,
package, product-gate, claim, manifest, docs, CI, and Graphify evidence before
the product slice can be marked complete.
