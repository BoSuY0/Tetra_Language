# scripts/release/surface

Surface release gates live here while Tetra Surface remains experimental.

Current entrypoints:

- `gate.sh` runs the complete experimental Surface release gate. It executes
  the headless, Linux-x64 starter, Linux-x64 real-window, wasm32-web starter,
  wasm32-web browser canvas/input, TextBox focus/text input, component-tree,
  component-tree API hardening, minimal toolkit, toolkit reuse, and
  accessibility metadata smoke scripts into the same report directory,
  revalidates all
  `tetra.surface.runtime.v1` reports, and writes plus validates the final
  artifact hash manifest. This is the shortest command for checking the current
  cross-target Surface evidence set.

- `surface-headless-smoke.sh` writes and validates `tetra.surface.runtime.v1`
  headless evidence for `examples/surface_counter.tetra`. The gate builds and
  runs the Surface component app, stores the executable under the report
  directory, writes a hashed compiler-owned `surface-runner-trace.json`, and
  records process, event, state-transition, and SHA-256 RGBA framebuffer
  checksum evidence for distinct pre-event and post-event frames
  plus a positive `no legacy UI sidecar artifacts` case after scanning the
  artifact directory. The report also records `host-provided pointer event
  dispatch` and `pre/post event frame sequence`, proving the counter consumes a
  Surface Host event and redraws changed state rather than using a
  self-constructed click.

- `surface-linux-x64-smoke.sh` writes and validates Linux-x64 starter evidence
  for the same pure-Tetra counter app, then builds and runs a pure-Tetra host
  probe that must exit `42` after kernel-backed Surface Host ABI
  open/present/close behavior. The current host uses `memfd_create`, `write`,
  and `close`; it is non-stub executable evidence, not a full real-window
  desktop/event-pump promotion gate. It also runs a pure-Tetra 2x2 RGBA
  presentation probe, reads the memfd bytes through `/proc/<pid>/fd/*`, and
  records the app-presented frame checksum as a third frame. It then runs a
  long-lived pure-Tetra counter presentation probe, reads the CounterApp /
  CounterButton after-event 320x200 RGBA bytes from the host memfd, and records
  that checksum as frame order 4. It uses the same no-legacy-sidecar artifact
  scan, host-provided pointer event case, and pre/post event frame sequence
  case before accepting the report. The report is explicitly marked with
  `host_evidence.level = linux-x64-memfd-starter`; the validator rejects this
  starter level if it claims real-window or native-input promotion.

- `surface-linux-x64-real-window-smoke.sh` writes and validates Linux-x64
  real-window evidence for `examples/surface_window_counter.tetra`. The gate
  builds and runs the pure-Tetra counter/window app, then runs a Wayland shm
  probe that opens a real Linux window, sets a title, presents a Tetra-owned
  400x240 RGBA framebuffer, and exits `42`. Its report is marked
  `host_evidence.level = linux-x64-real-window`, records
  `backend = wayland-shm-rgba`, requires click, key, resize, text payload, and
  close event evidence, and keeps the same no-legacy-sidecar artifact scan.
  The validator rejects headless, memfd-only, docs-only, build-only,
  metadata-only, legacy `.ui.*`, DOM/web-only, fake, or stale evidence for this
  promotion level.

- `surface-wasm32-web-smoke.sh` writes and validates wasm32-web starter
  evidence for the same pure-Tetra counter app. The gate builds `.wasm` plus
  the compiler-owned wasm Surface loader, validates the exact
  `tetra_surface_host_v1.__tetra_surface_*` import allowlist, runs the module
  through `scripts/tools/web_run_module.mjs --surface-trace`, records a
  `surface-wasm32-web` runtime report with pre/post event frame evidence plus
  actual wasm `present_rgba` frame checksums read from wasm memory, and rejects
  legacy UI sidecars while allowing only the compiler-owned loader paired with
  the `.wasm` artifact and hashed runner-trace JSON artifact. It is not a
  production browser canvas/input promotion gate.

- `surface-wasm32-web-browser-canvas-smoke.sh` writes and validates
  wasm32-web browser canvas/input evidence for
  `examples/surface_browser_counter.tetra`. The gate builds the pure-Tetra app
  as `.wasm`, validates the exact Surface Host ABI imports, launches a real
  Chromium-compatible browser, opens an `HTMLCanvasElement`, presents
  Tetra-owned RGBA framebuffer bytes, reads the canvas pixels back, dispatches
  pointer/key/resize/text input through the tiny Surface Host ABI, and records
  a `tetra.surface.browser-canvas-trace.v1` artifact with matching source and
  canvas checksums. It is still experimental Surface evidence, not DOM UI,
  React, user JavaScript app logic, legacy `.ui.*` sidecar playback, or
  Node-only promotion evidence.

- `surface-headless-text-focus-input-smoke.sh`,
  `surface-linux-x64-real-window-text-focus-input-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh` write and
  validate TextBox focus/text input evidence for
  `examples/surface_textbox_app.tetra`. The reports prove click focus, Tab
  focus transfer between `TextBox` and `SubmitButton`, focused keyboard
  routing, text insertion into component-owned storage, caret movement,
  backspace/delete, resize preserving focus, and visible RGBA framebuffer
  updates. The browser-canvas variant dispatches real browser pointer,
  `beforeinput`, ArrowLeft, Backspace, Delete, Tab, Space, and resize events
  through the compiler-owned host. These gates do not claim IME, clipboard,
  rich text, platform accessibility tree, production widget toolkit, user JS,
  DOM UI, or legacy sidecar support.

- `surface-headless-component-tree-smoke.sh`,
  `surface-linux-x64-real-window-component-tree-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-component-tree-smoke.sh` write and
  validate the experimental Dynamic Component Tree milestone for
  `examples/surface_tree_app.tetra`. The reports prove a pure-Tetra
  semi-dynamic `ComponentTree`/`TreeNode` child list with stable node IDs,
  parent IDs, child indices, layout bounds, draw order, exact
  `TextBox -> SubmitButton -> ResetButton -> TextBox` focus cycling,
  root-to-leaf dispatch paths for TextBox/Submit/Reset, keyboard-routed
  Submit/Reset actions, TextBox text routing only while focused, ignored text
  while a Button is focused, resize relayout, changed frame checksums, and
  strict rejection of hardcoded, metadata-only, Node-only, DOM/user-JS, wrong
  host, and legacy sidecar evidence. The browser-canvas variant also validates
  wasm imports and records real Chromium-compatible canvas/input evidence.

- `surface-headless-component-tree-api-smoke.sh`,
  `surface-linux-x64-real-window-component-tree-api-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh` write and
  validate the Component Tree API Hardening milestone for the same
  `examples/surface_tree_app.tetra` source. Reports keep the
  `tetra.surface.component-tree.v1` tree block and add
  `component_tree_api.schema = tetra.surface.component-tree-api.v1` with
  `api_level = builder-layout-dispatch-v1` and
  `manual_bookkeeping:false`. The validator requires builder evidence from
  `tree_add_root`/`tree_add_child`, `tree_validate` invariant evidence,
  Column/Row layout helper evidence, helper-routed hit tests, focus wrap
  evidence, dispatch path helper output, matching source/host evidence, changed
  frame checksums, and no legacy sidecars.

- `surface-headless-minimal-toolkit-smoke.sh`,
  `surface-linux-x64-real-window-minimal-toolkit-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh` write and
  validate the experimental minimal reusable widget toolkit milestone for
  `examples/surface_toolkit_form.tetra`. Reports keep `component_tree` and
  `component_tree_api` evidence and add
  `toolkit.schema = tetra.surface.toolkit.v1` with
  `toolkit_level = minimal-widgets-v1`, `module = lib.core.widgets`,
  `experimental:true`, `production_claim:false`,
  `uses_component_tree_api:true`, and `manual_bookkeeping:false`. The gates
  prove reusable Panel/Column/Text/TextBox/Row/Button helpers, TextBox focus and
  byte-buffer editing, caret/backspace/delete, Submit/Reset keyboard routing,
  StatusText updates, resize relayout, changed frame checksums, no user JS, no
  DOM UI, no platform widgets, and no legacy sidecars. The browser-canvas
  variant also validates wasm imports and records real Chromium-compatible
  canvas/input evidence.

- `surface-headless-toolkit-reuse-smoke.sh`,
  `surface-linux-x64-real-window-toolkit-reuse-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh` write and
  validate the experimental toolkit-reuse-v1 milestone for
  `examples/surface_toolkit_settings.tetra`. Reports keep `component_tree` and
  `component_tree_api` evidence and extend `toolkit.schema =
  tetra.surface.toolkit.v1` with `toolkit_level = toolkit-reuse-v1`,
  `reuse_level = multi-form-widget-reuse-v1`, both toolkit example sources,
  `text_box_count = 2`, `button_count = 2`, `multi_textbox_evidence:true`,
  and `multi_form_evidence:true`. The gates prove reusable `lib.core.widgets`
  Panel/Column/Text/TextBox/Row/Button helpers across two examples,
  focused-only NameTextBox/EmailTextBox routing, Save/Reset keyboard actions,
  StatusText updates, resize relayout to 480x320, changed frame checksums, no
  demo-local widget structs, no manual tree structural writes, no Node-only
  browser promotion, no user JS, no DOM UI, no platform widgets, and no legacy
  sidecars. The browser-canvas variant also validates wasm imports and records
  real Chromium-compatible canvas/input evidence.

- `surface-headless-accessibility-metadata-smoke.sh`,
  `surface-linux-x64-real-window-accessibility-metadata-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh` write and
  validate the experimental accessibility metadata tree milestone for
  `examples/surface_accessibility_settings.tetra`. Reports keep
  `component_tree`, `component_tree_api`, and `toolkit` evidence and add
  `accessibility_tree.schema = tetra.surface.accessibility-tree.v1` with
  `accessibility_level = metadata-tree-v1`, `module = lib.core.accessibility`,
  `widget_module = lib.core.widgets`, `experimental:true`,
  `production_claim:false`, `platform_host_integration:false`,
  `dom_aria_integration:false`, `screen_reader_evidence:false`,
  `manual_bookkeeping:false`, `no_dom_ui:true`, `no_user_js:true`, and
  `no_legacy_sidecars:true`. The gates prove roles, labels, label relations,
  values, states, bounds, focus order, reading order, edit/press/save/reset
  actions, snapshots, status updates, metadata checksum changes, bounds
  checksum changes after 480x320 resize, changed frame checksums, no DOM/ARIA
  evidence, no user JS, no platform accessibility host, no screen-reader
  claim, no production accessibility claim, and no legacy sidecars. The
  browser-canvas variant also validates wasm imports and records real
  Chromium-compatible canvas/input evidence.
