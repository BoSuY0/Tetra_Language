# Tetra Surface Guide

Status: current for the bounded Surface v1 linux-x64 real-window and wasm32-web browser-canvas
release scope. Headless Surface is a release evidence target. macOS/Windows Surface and wasm32-wasi
Surface UI are unsupported in this release.

Tetra Surface is the future UI model for new Tetra applications. It is not a wrapper over React, the
HTML DOM, Qt, GTK, WinUI, Cocoa, or generated metadata sidecars. A Surface app is written as Tetra
structs and methods that measure, lay out, draw, and handle events. On `wasm32-web`, the browser DOM
document and canvas are compiler-owned host plumbing; app authors do not provide a DOM-authored UI
tree or user JavaScript application logic.

## Claim Tiers For Surface Authors

Surface docs use five claim tiers so examples and release notes do not outrun evidence:

| Tier                 | Author-facing meaning                                                                                                                                                                                           |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PROD_STABLE_SCOPED` | guarded vocabulary for the named Linux/web release scope only after the product gate and final audit; it is not a broad Electron, React, CSS, DOM, Windows, macOS, GPU, rich-text, bidi, or screen-reader claim |
| `BETA_TARGET_HOST`   | target-host path with evidence but not current production support                                                                                                                                               |
| `EXPERIMENTAL`       | Block/Morph/visual/recipe evidence that is useful for design direction but must not be described as production support                                                                                          |
| `UNSUPPORTED`        | feature or target with no current release support                                                                                                                                                               |
| `NONCLAIM`           | explicit wording for adjacent capabilities that are deliberately not claimed                                                                                                                                    |

The docs governance checks are `validate-manifest`, `verify-docs`, and `validate-surface-claims`.
The scoped product gate is:

```sh
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-v1
```

That gate is product evidence, not a final all-platform or API-compatibility claim.

## Block-First Direction

The next Surface authoring direction is the experimental Block System. Its goal is a Block-first
Surface architecture where `Block` is the core Surface primitive and polished UI shapes are ordinary
Block configurations. A button-like control is a Block with text, paint, state, click handling,
motion, and accessibility metadata; it is not a special core widget class.

This is an implementation track, not current release support. The current Block-system evidence is
scoped to the same-commit `tetra.surface.block-system.gate.v1` reports, artifact hashes, validators,
and `block_system.memory_budget` records under `reports/surface-block/p18-budget`.
`lib.core.widgets` remains the current release helper layer. Those helpers must move toward
recipes/compatibility over Block rather than becoming a larger built-in widget kit.

The first available slice is the `lib.core.block` data model:

```text
import lib.core.surface as surface
import lib.core.block as block

let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
let root_id: block.BlockID = block.id(1)
let props: block.BlockProps =
    block.props(block.layout_fixed(rect), block.paint_from_layer(block.paint_layer_fill(surface.Color(r: 24, g: 32, b: 40, a: 255))), block.text_label(18, surface.Color(r: 238, g: 242, b: 246, a: 255)), block.image_none(), block.input_clickable(), block.event_click(block.action_primary()), block.state_interactive(), block.motion_fast(), block.accessibility_button(18), block.asset_none())
let root: block.Block = block.make(root_id, block.id_none(), props)
```

That creates a Block model value only. The separate Block-system gate now proves scoped Block graph,
rendering, state/motion/input, accessibility, target runtime report, and memory-budget evidence for
the current scenes. It still does not promote Block to production support.

The current polished Block-only example set is release-gated by
`scripts/release/surface/surface-headless-block-system-smoke.sh`, which now also writes
`surface-block-examples.json`. These examples show visual grammar, not a new widget layer:

- `examples/surface/block_apps/surface_block_command_palette.tetra`: command-center overlay,
  editable query field, and command rows as configured Blocks.
- `examples/surface/block_apps/surface_block_project_dashboard.tetra`: sidebar-like navigation,
  metric panels, and action affordances as Block layout/paint/state recipes.
- `examples/surface/block_apps/surface_block_settings.tetra`: label relationships, editable fields,
  and action controls as Block input/accessibility metadata.
- `examples/surface/block_apps/surface_block_editor_shell.tetra`: editor shell, tabs, scrollable
  code area, and selected line treatment as Block composition.
- `examples/surface/block_apps/surface_block_glass_panel.tetra`: glass overlay/control-center
  treatment through layered paint, assets, state, and motion.

Each scene uses `lib.core.block` directly, includes dark/light theme tokens,
paint/layout/text/asset/accessibility/state/motion evidence, and keeps button-like, card-like,
input-like, command-item-like, and overlay-like shapes as Block configurations rather than core
`Button`, `Card`, `TextField`, `Sidebar`, or `Modal` abstractions.

Block-system runtime reports include bounded local memory/cache facts under
`block_system.memory_budget`: component count, deterministic stress count, render/state/motion/input
loop counts, framebuffer byte totals, paint/text/asset cache usage, cache budgets, and explicit
nonclaims. Treat that section as evidence that the current Block scene is budgeted and
cache-bounded. It is not an Electron comparison benchmark, and RSS is recorded only when host
evidence is available.

Run the complete Block-system gate with:

```sh
bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget
```

## Morph Capsule Layer

Morph Capsule is the next experimental authoring layer over Block. It gathers scoped design tokens,
materials, affordances, state lenses, motion presets, and recipes in `lib.core.morph`, then expands
those recipes into `Block` values. It does not add new core widget primitives or a separate runtime.

The current Morph examples are:

- `examples/surface/morph_core/surface_morph_command_palette.tetra`
- `examples/surface/morph_core/surface_morph_project_dashboard.tetra`
- `examples/surface/morph_core/surface_morph_settings.tetra`
- `examples/surface/morph_core/surface_morph_editor_shell.tetra`
- `examples/surface/morph_core/surface_morph_glass_panel.tetra`

They build command palette, dashboard, settings, editor shell, and glass-panel scenes through Morph
recipes before the recipes become `Block` graph evidence.

Run the Morph evidence gate with:

```sh
bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate
```

The gate writes a deterministic headless `tetra.surface.morph.v1` report and a
`tetra.surface.morph.gate.v1` summary. It is experimental evidence only and does not change the
Surface v1 production support boundary.

## What Changes

Legacy UI uses `state` and `view` declarations that lower to `tetra.ui.v1` metadata and
preview/runtime sidecars. That path remains supported as `ui.metadata-v1` compatibility.

Surface apps instead use ordinary Tetra objects:

```text
CounterApp
  state fields
  measure(max)
  layout(rect)
  draw(ctx)
  event(event)
  focus(focused)
  text_input(event)
  accessibility_role()
```

The host provides only a surface, caller-owned event buffer records, scalar compatibility event
helpers, minimal text payload copy, time, and presentation in the current starter slice. The app
owns the component tree, hit testing, state updates, drawing rules, and layout. The current
component-model evidence covers text input dispatch through `event_text_input`, `text_len`, and host
bytes copied into a caller-owned `[]u8`. The Linux-x64 event queue is deterministic for smoke
evidence: pointer, key, resize, text, then close.
`examples/surface/runtime/surface_textbox_app.tetra` proves the first pure-Tetra TextBox layer:
click focus, Tab focus changes, focused keyboard routing, byte-buffer insertion, caret movement,
backspace/delete, resize focus preservation, and visible framebuffer updates.
`examples/surface/release/surface_release_text_input.tetra` is the separate
`production-text-input-v1` baseline for invalid UTF-8 rejection, multiline byte storage, selection
copy/paste, host clipboard read/write, and IME/composition lifecycle traces. Full String-level IME
editing, rich text, bidi shaping, and grapheme-cluster caret movement remain nonclaims.
`examples/surface/toolkit/surface_tree_app.tetra` layers the next experimental milestone on top: a
Tetra-owned `ComponentTree`/`TreeNode` model built through `lib.core.component` helpers, with stable
IDs, helper-owned parent/child links, tree hit testing, root-to-leaf dispatch paths, focus order,
TextBox text routing, Button action routing, resize relayout, and changed RGBA frame evidence.
`examples/surface/toolkit/surface_toolkit_form.tetra` adds the first reusable toolkit layer on top
of that helper API. It imports `lib.core.widgets`, builds a
Panel/Column/Text/TextBox/Row/Button/StatusText form with ordinary Tetra structs, and proves click
focus, text editing, Submit/Reset actions, status updates, resize relayout, and changed frames.
It makes no DOM UI, no user JS, no platform-widget layer, and no production-toolkit claim.
`examples/surface/toolkit/surface_accessibility_settings.tetra` adds the first accessibility
metadata tree over the same toolkit. It imports `lib.core.accessibility` and `lib.core.widgets`,
builds a Panel/Column settings form with labels, TextBoxes, Save/Reset Buttons, and StatusText, and
proves roles, label relationships, values, states, bounds, focus order, reading order, actions,
status updates, snapshots, and resize-bound updates as metadata-only evidence. It is metadata-only
evidence; no platform accessibility host integration, no DOM/ARIA accessibility, no screen-reader
validation, and no production accessibility support are claimed.

## Target Order

1. Headless deterministic Surface for scripted events and checksums.
2. Linux-x64 Surface host behind the same ABI.
3. Linux-x64 real-window Surface evidence.
4. wasm32 web starter Surface with no user JS and no DOM UI.
5. wasm32-web browser canvas/input evidence through a compiler-owned host runner and a real browser
   canvas.

Linux-x64 real-window and wasm32-web browser-canvas are the current release targets for Surface v1.
macOS, Windows, and wasm32-wasi UI production claims require real target-host evidence. Build-only
or docs-only evidence does not promote Surface support. macOS and Windows stay `UNSUPPORTED` through
`tetra.surface.target-host-status.v1` until real target-host runners produce the required
window/input/clipboard/DPI/accessibility evidence.

## Developer Fast Loop

Use `tetra surface dev` for the current Surface developer loop. It runs a scoped fast rebuild
workflow and can write a `tetra.surface.dev-workflow.v1` report:

```text
tetra surface dev \
  --source path/to/project-or-entry \
  --target linux-x64 \
  --out-dir reports/surface-dev-workflow/dev-artifacts \
  --report reports/surface-dev-workflow/surface-dev-workflow.json \
  --morph-rendered-beauty-report reports/surface/morph-rendered-beauty.json \
  --change-file token:path/to/design/tokens.tetra \
  --change-file recipe:path/to/design/recipes.tetra \
  --change-file source:path/to/app/main.tetra
```

The report records the initial build, warm-cache rebuild, token/recipe/source changed rebuilds,
source diagnostics, artifact hashes, and, when a validated `tetra.surface.morph-rendered-beauty.v1`
report is attached, a `morph_to_pixels` chain for Morph tokens, recipe expansion, Block scene,
render command stream, frame artifact, golden artifact, and diff metrics. It is intentionally
described as fast rebuild evidence: it is not hot reload, not an Electron dev server, and not React
Fast Refresh.

## Surface Inspector

Use `tools/cmd/surface-inspector` to turn validated Surface runtime reports into a static
`tetra.surface.inspector.v1` report:

```text
go run ./tools/cmd/surface-inspector \
  --runtime-report block:path/to/surface-headless-block-system.json \
  --runtime-report morph:path/to/surface-headless-morph.json \
  --runtime-report morph-rendered-beauty:path/to/morph-rendered-beauty.json \
  --runtime-report app-model:path/to/surface-headless-app-model.json \
  --runtime-report accessibility:path/to/surface-headless-release-accessibility.json \
  --out reports/surface-inspector/surface-inspector.json \
  --html reports/surface-inspector/surface-inspector.html
```

The report exposes Block tree, resolved Morph tokens, layout, paint, accessibility, event route,
focus, and perf-counter state. With a `morph-rendered-beauty` input, it also exposes recipe
expansions, Block scene nodes, render commands, frame artifacts, golden diff result, and the
source-linked `morph_to_pixels` hash chain. The report also includes source locations, input report
coverage, hidden-state scan results, and optional static HTML. The HTML is a tool report only. It is
not browser devtools, React devtools, DOM runtime UI, hidden app state, hot reload, or a replacement
for target-host accessibility evidence.

## Project Templates

Use `tetra new surface-app` to start a Block/Morph Surface app:

```text
tetra new surface-app --template command-palette my-palette
tetra new surface-app --template settings my-settings
tetra new surface-app --template dashboard my-dashboard
tetra new surface-app --template editor-shell my-editor
tetra new surface-app --template studio-shell my-studio
tetra new surface-app --template multi-window-notes my-notes
tetra new surface-app --template web-canvas my-web-canvas
```

Generated projects include `Capsule.t4`, `src/main.tetra`, `surface-template.json`, design
token/recipe marker files, and a README. They target linux-x64 plus wasm32-web, use Block/Morph
recipe expansion, and avoid core widget primitives in template source. Validate the full onboarding
path with:

```text
bash scripts/release/surface/surface-template-smoke.sh \
  --report-dir reports/surface-templates/gate
```

The smoke writes `tetra.surface.template-smoke.v1` / `surface-template-smoke-v1` evidence for
generation, check, build, run, inspector, visual diff, and package artifacts.

## Reference App Suite

Use the reference app suite to check real product shapes rather than isolated rectangles or one-off
screenshots:

```text
bash scripts/release/surface/surface-reference-apps-smoke.sh \
  --report-dir reports/surface-reference-apps/gate
```

The suite covers command palette, settings, dashboard, editor shell, file manager/list-detail,
dialog/notification, localized form, accessibility-heavy form, multi-window notes, and migration
examples. It writes `tetra.surface.reference-app-suite.v1` / `surface-reference-app-suite-v1`
evidence plus a `tetra.surface.visual-regression.v1` report for headless, linux-x64 real-window, and
wasm32-web browser-canvas targets. Each app uses stable Morph recipes that resolve to Block and
records visual, interaction, accessibility, performance, token/theme, layout, and artifact-hash
evidence. The migration app is the only reference app allowed to import `lib.core.widgets`; that
import is compatibility evidence, not a new core primitive.

## Packaging And Updates

Use the package smoke to build installable Surface app artifacts for the current Linux/web scope:

```text
bash scripts/release/surface/surface-package-smoke.sh \
  --report-dir reports/surface-packages/gate
```

For the product-slice flagship app, run the same verified script with an explicit source and app id:

```text
bash scripts/release/surface/surface-package-smoke.sh \
  --report-dir reports/surface-product-slice/package \
  --source examples/surface/migration/surface_migration_tetra_control_center.tetra \
  --app-id studio-shell \
  --app-title "Tetra Studio Shell" \
  --expected-exit-code 5
```

The smoke writes `tetra.surface.package.v1` / `surface-package-v1` evidence for the default
command-palette reference app or an explicitly named app such as the `studio-shell` flagship. It
creates a linux-x64 tar.gz package, a wasm32-web tar.gz bundle with HTML, wasm, and the
compiler-owned loader, local asset manifests with sha256 hashes, an installed linux-x64 package run,
and a hash-pinned update channel manifest. The flagship report records `expected_exit_code: 5`
because the current Surface app-state smoke returns `5` as its deterministic success value; this is
install/run evidence, not a process-manager success-code claim. The report keeps signing,
notarization, automatic runtime updates, network update fetching, Electron runtime, React runtime,
CSS cascade runtime, DOM-authored app UI trees, remote asset fetching, and user JavaScript app logic
as nonclaims unless later platform/runtime evidence promotes them.

For the competitive product-slice boundary, see `docs/user/surface/surface_electron_comparison.md`.

Use the crash report smoke to prove bounded diagnostic behavior:

```text
bash scripts/release/surface/surface-crash-report-smoke.sh \
  --report-dir reports/surface-crash-report/gate
```

The smoke writes `tetra.surface.crash-report.v1` / `surface-crash-report-v1` evidence for the
command-palette reference app. It records linux-x64 `command_failure`, `host_crash`, and
`restart_recovery` scenarios, local redacted `tetra.surface.diagnostic.v1` artifacts, bounded
trace/log collection, a non-user-data diagnostics policy, and scoped before/report/after restart
proof. It does not claim universal crash recovery and rejects diagnostics that leak user data,
upload over the network, depend on Electron crash reporting, or claim restart without evidence.

Use the i18n smoke to prove bounded localization behavior:

```text
bash scripts/release/surface/surface-i18n-smoke.sh \
  --report-dir reports/surface-i18n/gate
```

The smoke writes `tetra.surface.i18n.v1` / `surface-i18n-v1` evidence for the localized-form
reference app. It records bounded string tables, `uk-UA` locale selection, `en-US` fallback,
missing-key diagnostics, deterministic formatting hooks, localized form execution, and an RTL
placeholder nonclaim. It does not claim full ICU, full bidi shaping, RTL production text layout,
third-party intl runtime, platform locale dependency, or silent missing-key fallback.

Use the widget migration smoke to prove that existing Surface v1 widget-helper apps remain
compatible while new architecture moves through Morph recipes to Block:

```text
bash scripts/release/surface/surface-widget-migration-smoke.sh \
  --report-dir reports/surface-widget-migration/gate
```

The smoke writes `tetra.surface.widget-migration.v1` / `surface-widget-migration-v1` evidence for
the migration reference app. It keeps `lib.core.widgets` supported as a compatibility layer,
preserves the release widget set, proves Panel/Button/TextBox equivalence rows against Morph recipes
that resolve to Block, and rejects future core widget primitive promotion, breaking API changes,
docs-only migration, and platform toolkit/runtime claims.

## Troubleshooting Release Evidence

- Linux-x64 real-window release evidence requires a display host. Set `WAYLAND_DISPLAY` or `DISPLAY`
  before running the release-window gates. If no display host is available, the gate must write a
  blocked report instead of promoting headless or memfd starter evidence.
- Browser-canvas release evidence requires a Chromium-compatible browser and working browser
  dependencies. If the browser cannot launch or canvas readback cannot be collected, the gate must
  write a blocked report instead of promoting Node-only starter evidence.
- Keep current release evidence in fresh repo-local report directories such as
  `reports/surface-ui-production-final/surface-release-v1`. Do not use host temp directory paths,
  copied reports, non-empty report dirs, or stale report dirs as current proof.
- Starter evidence remains useful regression coverage: the linux memfd starter and Node wasm loader
  are not linux-x64 real-window or wasm32-web browser-canvas release proof.

## Release-Supported Surface App Shape

Release-supported apps use ordinary Tetra structs plus `lib.core.component`, `lib.core.widgets`,
`lib.core.text`, `lib.core.accessibility`, and `lib.core.style`. The scoped Linux app-shell subset
also uses `lib.core.surface_app_shell` for explicit window/shell state helpers. The app owns layout,
hit testing, focus, state, text buffers, clipboard/IME state, accessibility metadata, and app-shell
lifecycle state. The host boundary only opens a surface, copies events/text into caller-owned
buffers, reports time, handles clipboard/composition bridge calls, mirrors accessibility for
supported targets, records scoped app-shell traces, checks app-shell capabilities through
`surface-security-permission-v1`, records local `surface-performance-budget-v1`
startup/frame/memory/RSS/cache/framebuffer/binary-size/CPU-proxy evidence, records
`surface-dev-workflow-v1` fast rebuild evidence through `tetra surface dev`, records
`surface-inspector-v1` static inspector evidence for
Block/Morph/layout/paint/accessibility/event/focus/perf state, records
`surface-reference-app-suite-v1` evidence for ten Block/Morph product shapes across headless,
linux-x64 real-window, and wasm32-web browser-canvas targets, records `surface-package-v1`
package/install/web-bundle/update-channel evidence, records `surface-crash-report-v1` local
diagnostic/restart evidence, records `surface-i18n-v1` bounded localization evidence, and presents
RGBA frames. Ambient filesystem and network access are denied by default in the release app-shell
template; clipboard is allowed only through the scoped host policy, while notifications, dialogs,
shell open-url, remote asset fetching, signing, notarization, automatic runtime updates, network
update fetching, and unsupported Electron speed comparisons remain nonclaims until target evidence
exists.

Use `examples/surface/release/surface_release_counter.tetra`,
`examples/surface/release/surface_release_form.tetra`,
`examples/surface/release/surface_release_text_input.tetra`, and
`examples/surface/release/surface_release_accessibility.tetra` as the release-supported examples.
Use `examples/surface/toolkit/surface_linux_app_shell_notes.tetra` only for the scoped Linux
app-shell subset. Older `surface_toolkit_*` and `surface_accessibility_settings` examples remain
experimental regression evidence.

The current headless evidence entrypoint is:

```text
bash scripts/release/surface/surface-headless-smoke.sh
```

The aggregate experimental Surface gate is:

```text
bash scripts/release/surface/gate.sh
```

It runs the headless, Linux-x64 starter, Linux-x64 real-window, wasm32-web starter, wasm32-web
browser canvas/input, three TextBox focus/text input Surface smoke gates, three component-tree smoke
gates, and three component-tree API hardening gates, plus minimal toolkit, toolkit reuse, and
accessibility metadata gates into one report directory, revalidates every report, and writes plus
validates the final artifact hash manifest.

It writes a `tetra.surface.runtime.v1` report for `examples/surface/runtime/surface_counter.tetra`
and validates that the report has executable process evidence, deterministic host-provided pointer
event handling, a `count` state transition, distinct pre-event and post-event RGBA frame checksums,
and positive `host-provided pointer event dispatch`, `pre/post event frame sequence`,
`component hierarchy dispatch`, `component text input scalar dispatch`, `component focus dispatch`,
`component accessibility metadata`, and `no legacy UI sidecar artifacts` cases. The validator also
checks that process evidence includes a build command for the reported source, an executable Surface
component app process with the expected app exit, `component-app` artifact hash/size evidence linked
to that process, and component type names from the reported `.tetra`/`.t4` source module. The
`validate-surface-runtime` CLI recomputes local artifact file sizes and SHA-256 digests, so a report
cannot claim an artifact hash without the matching file. A report cannot pair an unrelated source
path with copied component evidence. For wasm32-web, the report must also include a
`compiler-owned-loader` `.mjs` artifact hash. HTML artifacts, legacy `.ui.*` sidecars, and
non-loader JavaScript artifacts are rejected as Surface evidence. The report includes an
`artifact_scan` record with the scanned root, checked file count, no forbidden paths, and
`pass: true`; the checked-file count must cover at least every reported artifact, and every reported
artifact must be under that root, so the no-sidecar case is backed by the same concrete directory
scan that covers the hashed runtime artifacts. The checksums are SHA-256 over deterministic headless
RGBA framebuffer bytes, not metadata or descriptor hashes. The gate builds and runs the Surface
component app, scans the report artifact directory for legacy `.ui.*`, HTML, and JS sidecars, writes
a compiler-owned `surface-runner-trace.json` with the deterministic headless frame/event trace, and
stores both the executable and trace under the report directory before hashing the artifacts. The
runtime validator also checks that the trace schema is the headless schema and that the trace source
and frames match the reported Surface source and frames. This proves the headless starter slice.

The Linux-x64 starter evidence entrypoint is:

```text
bash scripts/release/surface/surface-linux-x64-smoke.sh
```

That gate builds and runs the Surface counter through the native `linux-x64` target and also runs a
pure-Tetra host probe that requires `surface_open` to return a kernel-backed handle,
`surface_present_rgba` to write RGBA bytes through that handle, and `surface_close` to close it. The
current Linux host is `memfd_create`/`lseek`/`write`/`close` behind the Surface Host ABI. It is
non-stub executable evidence. The gate also runs a pure-Tetra event-sequence probe that calls
`surface_poll_event_into` and must observe pointer, key, then resize records through the Linux host
ABI. Finally, it runs a pure-Tetra 2x2 RGBA presentation probe, reads the presented bytes back from
the kernel-backed memfd through `/proc/<pid>/fd/*`, records an app-presented frame checksum as the
third frame after the counter's pre/post event frames, then runs a long-lived pure-Tetra counter
presentation probe and records the CounterApp/CounterButton after-event 320x200 presented frame as
frame order 4. The report requires the counter app to consume the starter host-provided pointer
event instead of a self-constructed click and requires positive
`linux-x64 counter component app-presented frame` evidence. It is not yet a full real-window desktop
Surface, native input pump, text-input host, or accessibility host.

The Linux-x64 real-window evidence entrypoint is:

```text
bash scripts/release/surface/surface-linux-x64-real-window-smoke.sh
```

That gate builds and runs `examples/surface/runtime/surface_window_counter.tetra`, a pure-Tetra
counter/button app that opens a Surface, draws into a framebuffer, consumes click and key events,
handles resize without breaking layout, consumes a small host text payload, presents an updated
frame, and exits through close. The gate also opens a real Wayland shm Linux window through the
Surface smoke probe, presents a 400x240 RGBA frame, records
`host_evidence.level = linux-x64-real-window`, and rejects headless, memfd-only, docs-only,
metadata-only, legacy `.ui.*`, DOM/web-only, fake, or stale evidence for that promotion level.

The Linux-x64 app-shell subset entrypoint is:

```text
bash scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh
```

That gate builds `examples/surface/toolkit/surface_linux_app_shell_notes.tetra`, opens the target
Linux Surface path, writes `surface-linux-x64-release-app-shell.json`, and validates
`tetra.surface.linux-app-shell.v1` evidence. The accepted subset is lifecycle open/close/reopen, two
presented notes windows, resize/DPI/cursor traces, clipboard read/write, IME composition
start/update/commit/cancel, accessibility platform bridge evidence, a scoped app-menu adapter, and
`electron-feature-ledger-v1` rows for scoped crash/error reporting plus blocked-pass nonclaims for
dialog, file dialog, file picker, notification, tray, and deep link. The same report carries
`surface-security-permission-v1` permission evidence: default-deny filesystem and network policy,
capability-checked IPC/process boundaries, scoped clipboard policy, denied
notification/dialog/shell-open-url rows, and local hashed font/image/icon asset safety. It also
carries `surface-performance-budget-v1` budget evidence for startup-to-first-frame, p50/p95 frame
build/present, scene counts, memory/RSS/cache/framebuffer, binary size, CPU/power proxy, bounded
caches, and mandatory `peak_rss_bytes`. It rejects GTK/Qt/native widget UI, Electron/React runtimes,
DOM UI, user JavaScript app logic, platform widgets, unrestricted filesystem/network access, remote
asset fetches, unsupported Electron speed comparisons, headless-only evidence, build-only evidence,
docs-only evidence, and artifact claims without matching local hashes.

Surface runtime gates reject reports that mention legacy `.ui.html`, `.ui.web.mjs`, `.ui.json`,
`tetra.ui.v1`, DOM UI, HTML UI, user JavaScript, or user JS evidence instead of pure-Tetra Surface
runtime evidence.

The wasm32-web starter evidence entrypoint is:

```text
bash scripts/release/surface/surface-wasm32-web-smoke.sh
```

That gate builds `examples/surface/runtime/surface_counter.tetra` as pure Tetra for `wasm32-web`,
validates the exact `tetra_surface_host_v1.__tetra_surface_*` import allowlist, checks the
compiler-owned `.mjs` loader, runs the module through `scripts/tools/web_run_module.mjs`, emits a
`surface-wasm32-web` `tetra.surface.runtime.v1` report, and accepts the compiler-owned loader while
rejecting legacy `.ui.json`, `.ui.web.mjs`, `.ui.html`, user JS, and DOM UI evidence. The Node
runner supplies the same starter scalar pointer event as the native starter host and writes a
compiler-owned Surface trace containing the actual `present_rgba` frame checksums read from wasm
memory. The validator requires the web runner-trace schema, checks that trace `wasm_path` matches
the reported `.wasm` component artifact, maps trace frame orders back to the reported Surface
frames, and requires the order-4 320x200 actual presented frame trace evidence plus a hashed
`runner-trace` artifact. It proves the starter wasm Host ABI path and Node web runner path.

The wasm32-web browser canvas/input evidence entrypoint is:

```text
bash scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh
```

That gate builds `examples/surface/runtime/surface_browser_counter.tetra` as pure Tetra for
`wasm32-web`, validates the `tetra_surface_host_v1.__tetra_surface_*` import allowlist, launches a
real Chromium-compatible browser, opens an `HTMLCanvasElement`, presents Tetra-owned RGBA
framebuffer bytes, reads the canvas pixels back, and dispatches pointer, key, resize, and text
events through the tiny Surface Host ABI. The report uses
`host_evidence.level = wasm32-web-browser-canvas-input` and `backend = browser-canvas-rgba`, records
browser-native input evidence, state updates, frame order 5 at 400x240, and a
`tetra.surface.browser-canvas-trace.v1` runner trace whose source/canvas checksums must match. This
is real browser canvas/input evidence, not DOM UI, React, user JavaScript app logic, legacy `.ui.*`
sidecars, or Node-only promotion evidence.

The TextBox focus/text input evidence entrypoints are:

```text
bash scripts/release/surface/surface-headless-text-focus-input-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-text-focus-input-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh
```

Those gates build `examples/surface/runtime/surface_textbox_app.tetra`. The browser-canvas gate
dispatches real browser pointer, `beforeinput`, ArrowLeft, Backspace, Delete, Tab, Space, and resize
events through the compiler-owned Surface host; the headless and linux real-window reports carry the
same strict TextBox focus/text/caret/edit evidence.

The component-tree evidence entrypoints are:

```text
bash scripts/release/surface/surface-headless-component-tree-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-component-tree-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-smoke.sh
bash scripts/release/surface/surface-headless-component-tree-api-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-component-tree-api-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh
```

Those gates build `examples/surface/toolkit/surface_tree_app.tetra`. The app is intentionally small:
`TreeApp -> Column -> TextLabel/TextBox/Row -> SubmitButton/ResetButton`. The Tetra app owns the
component tree and ordinary component structs; the host only delivers pointer/key/text/resize events
and presents RGBA bytes. The component-tree reports record tree node IDs, parent IDs, child
positions, layout bounds, draw order, focus order, and click dispatch paths. The component-tree API
reports add `tetra.surface.component-tree-api.v1` evidence showing `tree_add_root`,
`tree_add_child`, `tree_validate`, `tree_layout_column`, `tree_layout_row`, `tree_hit_test`,
`tree_focus_next`, and `tree_build_dispatch_path` helper use with `manual_bookkeeping:false`. Tab
moves focus `TextBox -> SubmitButton -> ResetButton -> TextBox`; text bytes insert only while the
TextBox owns focus; Submit/Reset actions are keyboard-routed through the focused root-to-leaf tree
path; the reset button clears the TextBox through a routed tree event; resize relayout widens the
TextBox from 288 to 368 pixels and preserves the focused node. This remains experimental
semi-dynamic child-list evidence with a hardened helper API. It is not a production widget toolkit,
not a final trait-object child list, and not the stricter `production-text-input-v1` gate. It makes
no rich text, no bidi shaping, and no platform accessibility tree claim.

The minimal toolkit evidence entrypoints are:

```text
bash scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh
```

Those gates build `examples/surface/toolkit/surface_toolkit_form.tetra`. The app imports
`lib.core.widgets` and uses reusable helpers for Panel, Column, Text, TextBox, Row, and Button
construction instead of defining demo-local widgets or manually writing tree structural fields.
Reports include `tetra.surface.toolkit.v1`, `toolkit_level = minimal-widgets-v1`,
`module = lib.core.widgets`, `experimental:true`, `production_claim:false`,
`uses_component_tree_api:true`, and `manual_bookkeeping:false`. The runtime evidence covers TextBox
focus and byte-buffer editing, caret/backspace/delete, Tab focus cycling through Submit/Reset,
keyboard-routed Submit and Reset, StatusText updates, resize relayout, and changed RGBA frames on
headless, linux-x64 real-window, and wasm32-web browser-canvas targets.

The toolkit reuse evidence entrypoints are:

```text
bash scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh
```

Those gates build `examples/surface/toolkit/surface_toolkit_settings.tetra`. The app uses the same
`lib.core.widgets` helpers across a second shape with `NameTextBox`, `EmailTextBox`, `SaveButton`,
`ResetButton`, and `StatusText`. Reports use `toolkit_level = toolkit-reuse-v1` and prove
multi-TextBox focus traversal, focused-only byte-buffer text routing, Save/Reset keyboard actions,
StatusText changes, resize relayout to 480x320, changed frame checksums, no demo-local widget
structs, no manual tree structural writes, no DOM UI, no user JavaScript app logic, and no
production toolkit claim.

The accessibility metadata tree evidence entrypoints are:

```text
bash scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh
bash scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh
bash scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh
```

Those gates build `examples/surface/toolkit/surface_accessibility_settings.tetra`. The app uses
`lib.core.widgets` for the ComponentTree shape and `lib.core.accessibility` for stable metadata
helpers. Reports add `accessibility_tree.schema = tetra.surface.accessibility-tree.v1` with
`accessibility_level = metadata-tree-v1`, `module = lib.core.accessibility`,
`widget_module = lib.core.widgets`, `experimental:true`, `production_claim:false`,
`platform_host_integration:false`, `dom_aria_integration:false`, `screen_reader_evidence:false`,
`manual_bookkeeping:false`, `no_dom_ui:true`, `no_user_js:true`, and `no_legacy_sidecars:true`. The
evidence proves the exact 12-node settings tree, NameLabel/EmailLabel label relationships,
NameTextBox to EmailTextBox to SaveButton to ResetButton focus order, reading order,
edit/press/save/reset actions, status updates, metadata checksum changes, bounds checksum changes
after 480x320 resize, and changed frame checksums. It is metadata-only accessibility evidence; no
Linux AT-SPI, no macOS AX, no Windows UI Automation, no browser DOM/ARIA accessibility, no
screen-reader validation, and no production Surface accessibility are claimed.

## Using lib.core.widgets

Surface apps that use the experimental toolkit should keep app state in the app and route structure
through helpers:

```tetra
import lib.core.component as component
import lib.core.widgets as widgets

var tree: component.ComponentTree = component.tree_init_api(20)
let root_id: Int = component.tree_add_root(tree, component.kind_root(), bounds)
let panel_id: Int = widgets.add_panel(tree, root_id, bounds)
let column_id: Int = widgets.add_column(tree, panel_id, bounds)
let name_id: Int = widgets.add_textbox(tree, column_id, bounds)
let row_id: Int = widgets.add_row(tree, column_id, bounds)
let save_id: Int = widgets.add_button(tree, row_id, bounds)
```

Use `component.tree_validate`, `widgets.column_layout`, `widgets.row_layout`, `widgets.hit_test`,
`component.tree_build_dispatch_path`, `widgets.textbox_text_input`, and `widgets.button_key_event`
instead of writing `TreeNode` structural fields directly. TextBox storage is caller-owned `[]u8`
storage copied from host text input; do not store borrowed String or slice views inside widget
state. The host only provides events and RGBA presentation.

## Using lib.core.accessibility

The accessibility metadata slice uses stable integer roles, values, and action codes so reports can
be validated without storing borrowed text views in persistent state:

```text
import lib.core.accessibility as accessibility
import lib.core.widgets as widgets

let name_box = widgets.add_accessible_textbox(tree, column_id, name_rect, name_label)
let save = widgets.add_accessible_button(tree, row_id, save_rect, widgets.action_save())
let status = widgets.add_accessible_status(tree, column_id, status_rect)
let meta = accessibility.textbox_metadata(name_label, accessibility.value_name(), name_len, 0, name_box)
```

Build the ComponentTree through `lib.core.component` and `lib.core.widgets`, then build metadata
through `lib.core.accessibility` and widget accessibility helpers. TextBox labels should use
`label_for` and `labelled_by` relationships, Buttons should expose focus and press semantics, and
StatusText should expose a status value without becoming focusable. Do not store borrowed `String`
or `[]u8` views inside accessibility state; use stable codes or copied/owned storage.

## Authoring Rules

- Write UI behavior in Tetra.
- Define components as structs.
- Implement only the abilities a component needs.
- Use `lib.core.component` helpers for static measurement and layout; do not treat them as magic
  compiler-known widgets.
- Keep host-specific code below the Surface Host ABI.
- Prefer the `lib.core.surface` wrappers over direct `core.surface_*` calls in app code. If
  low-level code closes `core.surface_close(win.handle)`, the checker treats `win` as consumed so
  later wrapper calls cannot reuse that surface handle. A local `Int` alias such as
  `let handle: Int = win.handle` keeps the same owner provenance for direct handle calls, so
  non-consuming host calls like `core.surface_request_redraw(handle)` also require `win` to still be
  live. If low-level code presents `core.surface_present_rgba(..., frame.pixels, ...)`, the checker
  treats the tracked frame pixels like `surface.present(frame)`: the frame owner must still be live,
  and local aliases of those pixels cannot be used after the raw present call. If code manually
  constructs `surface.Surface(handle: win.handle, ...)`, the new value is still an alias of `win`;
  closing either owner makes the other unusable.
- Keep `surface.Frame`, `surface.Event`, and `draw.DrawContext` local to the active Surface turn.
  They can be passed to draw/event helpers, but not stored in globals, user struct fields, or user
  enum payloads, returned from user functions, thrown through typed-error boundaries, assigned out
  through `inout`, or captured by function-typed closure values. The core `draw.DrawContext` wrapper
  is only for active draw call arguments.
- Keep `surface.Surface` handles on the Surface owner side of concurrency boundaries.
  `surface.Surface`, `surface.Frame`, `surface.Event`, and `draw.DrawContext` cannot be carried
  through typed task errors or typed actor messages without a future explicit transfer contract. A
  copied local `surface.Surface` handle is still an alias of the same host surface: after
  `surface.close(win)`, the checker rejects `surface.close(alias)` and
  `surface.request_redraw(alias)`.
- Treat `frame.pixels` the same way: it is a per-frame buffer. Mutate it while drawing, but do not
  return it or hand it out through `inout`, including via a local `[]u8` alias. Once
  `surface.present(frame)` consumes the frame, any local alias to `frame.pixels` is also no longer
  usable. The same rule applies to aliases of `ctx.frame.pixels` after `surface.present(ctx.frame)`.
  Present the frame before closing the `surface.Surface` that created it; a local frame cannot be
  presented after its owner surface handle has been closed, including when it was manually
  constructed as `surface.Frame(surface: win, ...)` or reached through `draw.DrawContext.frame`.
  Reassigning `ctx.frame` updates the tracked owner for later `surface.present(ctx.frame)` checks.
- Do not rely on generated `.ui.web.mjs`, `.ui.html`, DOM widgets, or native-shell sidecar playback
  for Surface apps.
- On `wasm32-web`, rely only on the compiler-owned Surface loader/host ABI. User JavaScript and
  generated DOM UI remain outside the Surface authoring model.
- Build component trees through `lib.core.component` helpers. Use `tree_add_root`, `tree_add_child`,
  `tree_set_bounds`, `tree_layout_column`, `tree_layout_row`, `tree_hit_test`, `tree_focus_next`,
  `tree_build_dispatch_path`, and `tree_build_draw_order`; do not manually write structural fields
  such as `id`, `parent_id`, `first_child`, or `child_count` in app code.
- Build accessibility metadata through `lib.core.accessibility` and `lib.core.widgets` helpers. Keep
  accessibility labels and values metadata-only and owned by Tetra; do not use DOM/ARIA, user
  JavaScript, platform widgets, screen-reader claims, or platform accessibility hosts as Surface
  accessibility evidence.

Minimal component-tree authoring shape:

```tetra
var tree: component.ComponentTree = component.tree_init_api(16)
let root: Int = component.tree_add_root(tree, component.kind_root(), root_rect)
let column: Int = component.tree_add_child(tree, root, component.kind_column(), false, root_rect)
let textbox: Int = component.tree_add_child(tree, column, component.kind_textbox(), true, textbox_rect)
let ok: Int = component.tree_layout_column(tree, column, root_rect, 16, 8)
let target: Int = component.tree_hit_test(tree, root_node, column_node, label_node, textbox_node, row_node, submit_node, reset_node, x, y)
let path_len: Int = component.tree_build_dispatch_path(tree, target, path_slots)
```

The current helper API is an experimental foundation for a future toolkit. It now has a minimal
experimental reusable `lib.core.widgets` layer, but it still does not provide production `Button` or
`TextBox` widgets, trait-object child lists, or witness-table dispatch. It is not the stricter
`production-text-input-v1` gate and makes no rich text, no bidi shaping, and no platform
accessibility integration claim.

The static component fixture is `examples/surface/runtime/surface_component_counter.tetra`. It
composes `CounterApp` and `CounterButton` as ordinary structs with `measure`, `layout`, `draw`,
`event`, `focus`, `text_input`, and `accessibility_role` methods, then uses `lib.core.component`
helpers for rectangle layout. The main `examples/surface/runtime/surface_counter.tetra` runtime
smoke now uses the same static ability shape for `CounterApp` and its `CounterButton` child, and
`tetra.surface.runtime.v1` reports record `measure`, `layout`, `draw`, `event`, `focus`, `text`,
`accessibility`, parent/child hierarchy, measured component bounds, root-to-child `dispatch_path`
entries, child-target event evidence, scalar text-input state evidence, and host text payload bytes.
The validator now rejects child dispatch evidence where the reported pointer event does not hit the
target component bounds.

`examples/surface/runtime/surface_text_input.tetra` adds a smaller TextBox fixture. It is still pure
Tetra: the host copies deterministic text payload bytes into the component-owned `[]u8` buffer, and
the user-defined `TextBox` accepts those bytes through its `text_input` method before drawing a
Surface frame. This is byte-buffer text input evidence; the stricter release text-input example
covers the scoped clipboard/composition baseline, while full String-level IME editing, rich text,
bidi shaping, and grapheme-cluster caret movement remain nonclaims.
`examples/surface/runtime/surface_textbox_app.tetra` is the editable milestone fixture: a Tetra
`FocusManager` routes click/Tab focus between `TextBox` and `SubmitButton`, keyboard input goes only
to the focused component, text bytes insert into component-owned storage, caret/backspace/delete
mutate the buffer, and resize preserves the focused component before the app redraws.
`examples/surface/toolkit/surface_tree_app.tetra` is the component-tree milestone fixture. It uses
`ComponentTree` helper calls for root/child construction, focus state, Column/Row layout, hit
testing, root-to-leaf pointer dispatch paths, exact
`TextBox -> SubmitButton -> ResetButton -> TextBox` focus cycling, focused TextBox text insertion,
keyboard-routed Submit/Reset Button actions, and resize relayout. Source tests reject app-side
writes to structural tree fields, and API reports prove `manual_bookkeeping:false`. This is
experimental component-tree helper evidence, not a full dynamic trait-object list and not full
String/IME editor evidence. It makes no rich text, no bidi shaping, no platform accessibility
integration, and no production browser evidence claim.
Linux-x64 real-window counter evidence is covered separately by
`examples/surface/runtime/surface_window_counter.tetra`; wasm32-web browser canvas/input counter
evidence is covered separately by `examples/surface/runtime/surface_browser_counter.tetra`.
`examples/surface/toolkit/surface_toolkit_form.tetra` is the minimal toolkit milestone fixture: it
uses `lib.core.widgets` helpers over `ComponentTree` for a form tree, records
`tetra.surface.toolkit.v1` evidence, and remains experimental minimal-widget evidence rather than a
production toolkit claim. `examples/surface/toolkit/surface_accessibility_settings.tetra` is the
accessibility metadata tree milestone fixture: it uses `lib.core.widgets` and
`lib.core.accessibility` helpers for the settings form metadata tree, records
`tetra.surface.accessibility-tree.v1` evidence, and remains experimental metadata-only accessibility
evidence rather than platform accessibility or production accessibility support.

## Migration

Existing `ui_web_smoke`, `ui_native_shell_smoke`, `dogfood_web_ui`, and `tetra_control_center`
examples stay available as legacy fixtures while Surface migration fixtures prove the pure-Tetra
shape in parallel.

Current migration examples:

- `examples/surface/migration/surface_migration_ui_web_smoke.tetra`
- `examples/surface/migration/surface_migration_ui_native_shell_smoke.tetra`
- `examples/surface/migration/surface_migration_dogfood_web_ui.tetra`
- `examples/surface/migration/surface_migration_tetra_control_center.tetra`

These examples replace `state`/`view` metadata with ordinary Tetra structs, `draw` methods, `event`
methods, local `surface.Event` values, and `draw.DrawContext` frame rendering. They are part of the
native smoke matrix and currently exit `2`, `11`, `3`, and `5` respectively through the Linux-x64
Surface Host ABI; they are not yet Linux-x64 real-window or wasm Surface promotion evidence. New
experimental examples should prefer Surface now that
`examples/surface/runtime/surface_counter.tetra`, the headless smoke path, the Linux-x64 starter
smoke path, the Linux-x64 real-window smoke path, and the wasm32-web starter plus browser
canvas/input smoke paths are available, with TextBox focus/text input gates layered on top.
