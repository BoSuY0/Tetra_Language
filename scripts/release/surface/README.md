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

- `release-gate.sh` runs the current Surface v1 `surface-v1-linux-web` release
  gate. It requires the headless release, Linux-x64 release-window, scoped
  Linux app-shell, Linux text/input/toolkit/accessibility, wasm32-web browser
  canvas, developer fast rebuild, static Surface inspector, project templates,
  reference app suite, package/update story, crash/error reporting,
  internationalization/localization,
  Block-system, Morph, artifact-hash, manifest, release-state, and
  claim-scanner evidence in one fresh report directory.

- `product-gate.sh` runs the scoped Surface product evidence gate. It executes
  `release-gate.sh` into the requested fresh report directory, revalidates the
  artifact hash manifest, runs the Surface claim scanner, validates
  `docs/generated/manifest.json` and docs, writes
  `surface-product-gate-summary.json`, and rewrites/validates artifact hashes
  after that summary is present. CI and release packaging use this as the
  mandatory Surface product evidence command; the final `PROD_STABLE_SCOPED`
  verdict and audit remain owned by P29.

- `surface-headless-smoke.sh` writes and validates `tetra.surface.runtime.v1`
  headless evidence for `examples/surface/runtime/surface_counter.tetra`. The gate builds and
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
  real-window evidence for `examples/surface/runtime/surface_window_counter.tetra`. The gate
  builds and runs the pure-Tetra counter/window app, then runs a Wayland shm
  probe that opens a real Linux window, sets a title, presents a Tetra-owned
  400x240 RGBA framebuffer, and exits `42`. Its report is marked
  `host_evidence.level = linux-x64-real-window`, records
  `backend = wayland-shm-rgba`, requires click, key, resize, text payload, and
  close event evidence, and keeps the same no-legacy-sidecar artifact scan.
  The validator rejects headless, memfd-only, docs-only, build-only,
  metadata-only, legacy `.ui.*`, DOM/web-only, fake, or stale evidence for this
  promotion level. This gate remains probe/release evidence for
  `surface-v1-linux-web`; it must not be cited as Native Surface Host v1 proof.

- `surface-linux-x64-native-host-smoke.sh` writes and validates the stricter
  Native Surface Host v1 report for
  `examples/surface/runtime/surface_window_counter.tetra`. It builds the
  official Wayland host, runs `tetra run --target linux-x64 --surface-host
  wayland` for the compiled Tetra app, records the host-side report, merges it
  into `tetra.surface.native-host.v1`, and validates with
  `--release linux-x64-native-host`. While the window is open, a real pointer
  event, key event, and close event must be delivered to the app. PNG/SVG/HTML,
  browser-canvas captures, ImageMagick/viewer windows, probe frames, and
  pre-rendered UI files are rejected substitutes. After strict runtime
  validation passes, the gate writes and validates `artifact-hashes.json` for
  the full native-host report directory.

- `surface-linux-x64-release-app-shell-smoke.sh` writes and validates the
  scoped Linux app-shell release subset for
  `examples/surface/toolkit/surface_linux_app_shell_notes.tetra`. The report uses
  `tetra.surface.linux-app-shell.v1` with
  `linux-app-shell-subset-v1` and proves target-host lifecycle open/close/
  reopen, two presented windows, resize/DPI/cursor traces, clipboard read/
  write, IME composition start/update/commit/cancel, accessibility platform
  bridge evidence, a scoped app-menu adapter, and
  `surface-security-permission-v1` default-deny filesystem/network policy.
  It also emits `surface-performance-budget-v1` local startup/frame/memory/
  cache/framebuffer/binary-size/CPU-proxy evidence. File dialog and notification
  remain `blocked_pass` nonclaims. The security and performance validators also
  check capability-scoped IPC/process boundaries, local hashed asset/font/image
  safety, bounded caches, mandatory peak RSS fields, and no unsupported
  faster-than-Electron claim. The validator rejects GTK/Qt/native widget UI,
  Electron/React runtimes, DOM UI, user JavaScript app logic, platform widgets,
  headless-only evidence, build-only evidence, docs-only evidence, and artifact
  claims without matching local hashes.

- `surface-inspector-smoke.sh` writes and validates
  `tetra.surface.inspector.v1` / `surface-inspector-v1` static tool evidence.
  It aggregates headless Block-system, Morph, app-model, accessibility, and
  event reports plus Morph rendered beauty evidence into
  `surface-inspector.json` plus optional `surface-inspector.html`. The report
  exposes Block tree, Morph tokens, recipe expansions, Block scene nodes,
  render commands, frame artifacts, golden diff result, layout, paint,
  accessibility, event route, focus, perf-counter, source location, input
  report coverage, and hidden-state scan evidence. It is not browser devtools,
  React devtools, DOM runtime UI, hidden app state, or target-host accessibility
  proof by itself.

- `surface-template-smoke.sh` writes and validates
  `tetra.surface.template-smoke.v1` / `surface-template-smoke-v1` onboarding
  evidence. It runs `tetra new surface-app` for command palette, settings,
  dashboard, editor shell, studio shell, multi-window notes, and web-canvas templates, then
  checks, builds, runs, inspects, visually tests, and packages the generated
  app paths. The report requires Block/Morph template source and rejects React,
  Electron, DOM-authored app UI trees, CSS runtime dependencies, core widget
  primitives, platform widgets, and user JavaScript app logic.

- `surface-reference-apps-smoke.sh` writes and validates
  `tetra.surface.reference-app-suite.v1` /
  `surface-reference-app-suite-v1` product-shape evidence. It checks, builds,
  and runs command palette, settings, dashboard, editor shell, file
  manager/list-detail, dialog/notification, localized form,
  accessibility-heavy form, multi-window notes, and migration reference apps.
  Each app uses stable Morph recipes that resolve to Block and records
  headless, linux-x64 real-window, and wasm32-web browser-canvas visual,
  interaction, accessibility, performance, token/theme, layout, and
  artifact-hash evidence. `lib.core.widgets` is accepted only for the migration
  compatibility example; screenshot-only and docs-only beauty claims are
  rejected.

- `surface-package-smoke.sh` writes and validates
  `tetra.surface.package.v1` / `surface-package-v1` packaging and update-story
  evidence. By default it builds the command-palette reference app for
  linux-x64 and wasm32-web; with `--source <path> --app-id <id>
  --app-title <title> --expected-exit-code <n>` it records the same evidence
  for an explicitly named product-slice app such as `studio-shell`. It creates
  tar.gz packages with `surface-app-package-v1` manifests, records local asset
  hashes, unpacks and runs the linux-x64 package, includes web
  HTML/wasm/compiler-owned loader output, and writes a hash-pinned update
  channel manifest. Signing, notarization, automatic runtime updates, network
  update fetching, React, Electron, CSS runtime, DOM-authored app UI trees,
  remote asset fetches, and user JavaScript app logic remain nonclaims.

- `surface-crash-report-smoke.sh` writes and validates
  `tetra.surface.crash-report.v1` / `surface-crash-report-v1` crash recovery
  and error-reporting evidence. It builds the command-palette reference app for
  linux-x64, records bounded command failure, host crash diagnostic capture,
  local redacted `tetra.surface.diagnostic.v1` artifacts, bounded trace/log
  collection, and `scoped-linux-x64-process-restart-v1` before/report/after
  restart proof. User data leaks, network upload, Electron crash reporter
  dependency, docs-only crash claims, and restart claims without evidence remain
  rejected.

- `surface-i18n-smoke.sh` writes and validates `tetra.surface.i18n.v1` /
  `surface-i18n-v1` internationalization and localization evidence. It builds
  the localized-form reference app for linux-x64, records bounded string
  tables, `uk-UA` locale selection, `en-US` fallback, missing-key diagnostics,
  deterministic formatting hooks, localized form execution, and an RTL
  placeholder nonclaim. Full ICU, full bidi shaping, RTL production text
  layout, third-party intl runtime, platform locale dependency, docs-only
  localization, and silent missing-key fallback remain rejected.

- `surface-widget-migration-smoke.sh` writes and validates
  `tetra.surface.widget-migration.v1` / `surface-widget-migration-v1`
  compatibility evidence. It builds the migration reference app for linux-x64,
  keeps `lib.core.widgets` supported for Surface v1, preserves the release
  widget set, records Panel/Button/TextBox equivalence rows against Morph
  recipes that resolve to Block, and rejects future core widget primitive
  promotion, breaking API changes, docs-only migration, and platform-native
  widget/runtime claims.

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
  `examples/surface/runtime/surface_browser_counter.tetra`. The gate builds the pure-Tetra app
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
  `examples/surface/runtime/surface_textbox_app.tetra`. The reports prove click focus, Tab
  focus transfer between `TextBox` and `SubmitButton`, focused keyboard
  routing, text insertion into component-owned storage, caret movement,
  backspace/delete, resize preserving focus, and visible RGBA framebuffer
  updates. The browser-canvas variant dispatches real browser pointer,
  `beforeinput`, ArrowLeft, Backspace, Delete, Tab, Space, and resize events
  through the compiler-owned host. These focus/input gates do not by
  themselves claim the stricter release text-input baseline; the
  `surface-*-release-text-input-smoke.sh` gates cover scoped clipboard and
  IME/composition traces. Full rich text, bidi shaping, platform accessibility
  tree, user JS, DOM UI, and legacy sidecar support remain nonclaims unless a
  later gate proves them.

- `surface-headless-component-tree-smoke.sh`,
  `surface-linux-x64-real-window-component-tree-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-component-tree-smoke.sh` write and
  validate the experimental Dynamic Component Tree milestone for
  `examples/surface/toolkit/surface_tree_app.tetra`. The reports prove a pure-Tetra
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
  `examples/surface/toolkit/surface_tree_app.tetra` source. Reports keep the
  `tetra.surface.component-tree.v1` tree block and add
  `component_tree_api.schema = tetra.surface.component-tree-api.v1` with
  `api_level = builder-layout-dispatch-v1` and
  `manual_bookkeeping:false`. The validator requires builder evidence from
  `tree_add_root`/`tree_add_child`, `tree_validate` invariant evidence,
  Column/Row layout helper evidence, helper-routed hit tests, focus wrap
  evidence, dispatch path helper output, matching source/host evidence, changed
  frame checksums, and no legacy sidecars.

- `surface-headless-block-system-smoke.sh` writes and validates the
  experimental headless Block-system golden/checksum milestone for
  `examples/surface/block_core/surface_block_system.tetra`. Reports keep Block graph, software
  paint, deterministic layout, and Block-derived accessibility evidence in the
  same `tetra.surface.runtime.v1` envelope and add
  `block_system.schema = tetra.surface.block-system.v1` with deterministic
  software RGBA frame goldens, repeat checksum equality, and explicit negative
  guards for missing frame checksum, nondeterministic checksum, missing paint,
  missing layout, and missing accessibility evidence. It remains headless-only
  evidence, not host display/browser promotion. The same gate also runs
  `validate-surface-block-examples` and writes `surface-block-examples.json`
  for the five polished Block-only examples:
  `examples/surface/block_apps/surface_block_command_palette.tetra`,
  `examples/surface/block_apps/surface_block_project_dashboard.tetra`,
  `examples/surface/block_apps/surface_block_settings.tetra`,
  `examples/surface/block_apps/surface_block_editor_shell.tetra`, and
  `examples/surface/block_apps/surface_block_glass_panel.tetra`.

- `surface-headless-morph-smoke.sh` writes and validates the experimental
  headless Morph Capsule milestone for
  `examples/surface/morph_core/surface_morph_command_palette.tetra`. Reports keep the normal
  `tetra.surface.runtime.v1` envelope, include Block System evidence, and add
  `morph.schema = tetra.surface.morph.v1` with
  `quality_level = deterministic-headless-morph-capsule-v1`. The smoke proves
  scoped capsule tokens, materials, affordances, state lenses, motion presets,
  recipes, recipe expansions, accessibility projection, memory-budget evidence,
  and negative guards while keeping Morph as a recipe layer that expands into
  Block.

- `morph-gate.sh` runs the strict experimental Morph Capsule evidence gate. It
  requires deterministic headless Morph evidence, same-commit report
  validation through `validate-surface-morph-report`, Block System dependency
  evidence in the same runtime envelope, P07 token graph validation through
  `validate-surface-token-graph` and
  `docs/spec/surface/surface_token_graph_contract.json`, final artifact hash integrity,
  and a `tetra.surface.morph.gate.v1` summary. It is headless experimental
  evidence, not Surface v1 production support.

- `visual-gate.sh` runs the experimental Surface visual regression evidence
  gate. It first runs `block-system-gate.sh`, then uses
  `surface-visual-diff` and `validate-surface-visual-report` to produce and
  validate a `tetra.surface.visual-regression.v1` report across headless,
  linux-x64 real-window, and wasm32-web browser-canvas Block System reports.
  The report records deterministic frame/golden/diff, token/theme, layout,
  accessibility, performance, and screenshot-only rejection evidence for
  `examples/surface/block_core/surface_block_system.tetra` plus the five polished Block-only
  examples from `surface-block-examples.json`. This is visual infrastructure
  evidence, not a production beauty claim or broad Electron renderer parity
  claim.

- `surface-linux-x64-real-window-block-system-smoke.sh` writes and validates
  the experimental linux-x64 real-window Block-system milestone for
  `examples/surface/block_core/surface_block_system.tetra`. It uses the existing Wayland shm RGBA
  real-window path, validates `block_system.schema =
  tetra.surface.block-system.v1` through `validate-surface-block-report`,
  records order-5 presented frame checksum evidence and native input/state
  cases, and writes an explicit blocked status artifact when `WAYLAND_DISPLAY`
  is unavailable instead of promoting headless evidence.

- `surface-wasm32-web-browser-canvas-block-system-smoke.sh` writes and
  validates the experimental wasm32-web browser-canvas Block-system milestone
  for `examples/surface/block_core/surface_block_system.tetra`. It requires a Chromium-compatible
  browser runner, builds the wasm app with the compiler-owned loader, validates
  wasm imports, reads back browser canvas RGBA pixels, records browser input
  cases, rejects Node-only promotion, and keeps `no user JavaScript app logic`
  plus `no DOM-authored app UI tree` sidecar claims enforced through
  `validate-surface-block-report`. The browser DOM document and canvas are
  compiler-owned host plumbing, not the app UI model.

- `surface-headless-minimal-toolkit-smoke.sh`,
  `surface-linux-x64-real-window-minimal-toolkit-smoke.sh`, and
  `surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh` write and
  validate the experimental minimal reusable widget toolkit milestone for
  `examples/surface/toolkit/surface_toolkit_form.tetra`. Reports keep `component_tree` and
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
  `examples/surface/toolkit/surface_toolkit_settings.tetra`. Reports keep `component_tree` and
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
  `examples/surface/toolkit/surface_accessibility_settings.tetra`. Reports keep
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
