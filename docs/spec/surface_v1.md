# Tetra Surface v1

Status: current for Surface v1 linux-x64 real-window, scoped Linux app-shell,
and wasm32-web browser-canvas release scope. Headless is a release evidence
target. macOS/Windows Surface and wasm32-wasi Surface UI are not
production-supported in this release.

This is the release contract for the Tetra Surface Object System in the
bounded `surface-v1-linux-web` scope: pure-Tetra user UI, tiny Surface Host ABI,
software RGBA framebuffer presentation, production widget subset, text/input
baseline, clipboard baseline, IME/composition baseline, accessibility metadata
plus platform bridge evidence for supported targets, strict validators, and
artifact hashes. The Linux app-shell claim is limited to the
`linux-app-shell-subset-v1` report schema for lifecycle, multi-window notes,
resize/DPI/cursor, clipboard, IME/composition, accessibility bridge, app-menu
adapter, `electron-feature-ledger-v1` rows, scoped crash/error reporting
adapters, `surface-crash-report-v1` local redacted diagnostic evidence,
`surface-i18n-v1` bounded localization evidence,
`surface-widget-migration-v1` widgets-to-Block/Morph compatibility evidence,
`surface-security-permission-v1` default-deny filesystem/network policy,
capability-checked process boundaries, local hashed asset/font/image safety,
`surface-performance-budget-v1` local startup/frame/memory/cache/
framebuffer/binary-size/CPU-proxy budget evidence, `surface-inspector-v1`
static inspector evidence, and blocked-pass dialog/file dialog/file picker/
notification/tray/deep-link evidence. It is not a claim
for GPU rendering, arbitrary native platform widgets, GTK/Qt widget UI,
DOM/React/user-JS application UI, unrestricted filesystem/network access,
remote asset fetching, official performance benchmarks, unsupported Electron
speed comparisons, dynamic trait-object widgets, witness-table component
dispatch, a full rich text editor, macOS/Windows Surface, or wasm32-wasi Surface
UI. macOS and Windows remain `UNSUPPORTED` through
`tetra.surface.target-host-status.v1`; build-only macOS/Windows artifacts do
not promote Surface runtime support.

Tetra Surface replaces the metadata-first UI direction for new work. Existing
`ui.metadata-v1` remains a legacy compatibility surface until Surface has
strict validator evidence.

## Claim Tier Boundary

Surface docs use the same claim-tier vocabulary as
`docs/spec/current_supported_surface.md`:

| Tier | Surface meaning |
| --- | --- |
| `PROD_STABLE_SCOPED` | guarded production vocabulary only for the named `surface-v1-linux-web` evidence scope after the product gate and final same-commit audit prove it; it is not an all-platform Electron/React/CSS replacement claim |
| `BETA_TARGET_HOST` | target-host evidence that may exist for a platform path but is not part of the current production claim |
| `EXPERIMENTAL` | Block, Morph, visual, recipe, or historical evidence tracks that must not be described as current production support |
| `UNSUPPORTED` | no current release support or no production target-host evidence |
| `NONCLAIM` | explicit boundary wording for Electron APIs, React APIs, CSS cascade/runtime compatibility, DOM-authored UI, Windows/macOS production, GPU rendering, full rich text, full bidi, and full screen-reader support |

The scoped product evidence gate is:

```sh
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-v1
```

That gate runs the Surface release evidence, claim scanner, manifest validator,
and docs verifier. It is not the P29 final audit and must not be used as a
standalone final `PROD_STABLE_SCOPED` signoff.

The Surface v1 docs/gate contract is mirrored by
`scripts/release/surface/contracts/surface-release-v1.json`; for an empty
report directory, the dry-run plan is
`go run ./tools/cmd/run-gate --contract scripts/release/surface/contracts/surface-release-v1.json --report-dir reports/surface-product-v1 --dry-run --json`.
The shell gates still produce evidence, and the dry-run path does not execute
the gate, prove remote CI execution, or promote macOS/Windows runtime support.

## Block-First Surface System Direction

The next Surface architecture track is the experimental
`ui.surface-block-system`: a Block-first Surface architecture where `Block` is
the core Surface primitive for layout, paint, text, images/assets,
input/events, state selectors, motion, and accessibility metadata. This remains
experimental and is not current Surface v1 production support. The current
evidence is scoped to same-commit `tetra.surface.block-system.gate.v1` reports,
validators, artifact hashes, and Block memory-budget evidence under
`reports/surface-block/p18-budget`; it does not create a production Block
claim.

In that model, a button-like, card-like, input-like, sidebar-like, or modal-like
control is a `Block` configuration with properties and behavior. `Button`,
`TextBox`, `Panel`, `Row`, `Column`, `Stack`, `Scroll`, `Checkbox`, and similar
helpers remain release-supported Surface v1 helpers today, but the Block System
requires them to become recipes/compatibility over Block instead of primary
architecture.

Block System beauty comes from primitive composition rather than imported
browser machinery: layered paint, rounded corners, borders, shadows, opacity,
clips, typography, local assets, hover/pressed/focus/selected/disabled/error
states, deterministic transitions, and accessibility metadata all resolve from
the same Block graph. The Block System still forbids Electron, Chromium, React,
DOM UI, a CSS runtime, user JavaScript app logic, Qt, GTK, Cocoa, WinUI, and
platform-native widgets as user-facing UI dependencies.

Current Block-system reports also carry a conservative
`block_system.memory_budget` section. That section is local release evidence
for the reported Block scene: Block count, stress Block count, render/state
loop counts, frame buffer bytes, paint/text/asset cache usage, cache budgets,
and an explicit performance nonclaim. RSS is optional host evidence and is not
required for this scoped budget. This is not an external benchmark against
Electron or any other desktop shell.

The strict Block gate is:

```sh
bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget
```

Passing that gate proves the scoped headless, linux-x64 real-window, and
wasm32-web browser-canvas Block reports for the same commit. It does not promote
Block to production support.

The Block ABI and renderer contract are frozen in
`docs/spec/surface_block_contract.md` with a machine-readable contract at
`docs/spec/surface_block_contract.json`. `validate-surface-block-contract`
checks that `Block` is the only core primitive, compatibility helpers stay out
of the primitive set, and any report with `block_graph` evidence also carries
paint command, layout pass, and Block accessibility tree evidence.

The Block layout hardening slice is validated through ordinary
`tetra.surface.runtime.v1` reports. Layout evidence must include constraints,
ordered layout passes, scroll bounds, overflow/clip policy, aspect sizing,
resize evidence, target density, pixel snapping, and the stable
`integer-half-up-v1` rounding policy. The current report quality level remains
`deterministic-block-layout-v1`; it is app-shell layout evidence, not CSS
flexbox/grid parity and not a general production layout claim.

The Block renderer/compositor baseline is also report-gated. Paint evidence now
uses the deterministic command order `fill`, `gradient`, `image_fill`,
`border`, `radius_clip`, `shadow`, `overlay`, `outline`, `text`, and `icon`.
The report-level `renderer` block uses schema
`tetra.surface.renderer-feature.v1` with the `software-rgba` backend, `rgba8`
color format, compositor layers, dirty rects, invalidations, bounded cache
stats, deterministic frame checksums, and explicit rejection of unsupported GPU,
blur, and backdrop-blur production claims. This is a production-baseline shape
for the current software renderer evidence; it is not a GPU renderer claim, not
a browser/CSS compositor claim, and not broad Electron renderer parity.

The experimental visual regression gate is:

```sh
bash scripts/release/surface/visual-gate.sh \
  --report-dir reports/surface-visual/gate
```

It runs the Block-system gate and writes
`tetra.surface.visual-regression.v1` evidence through
`tools/cmd/surface-visual-diff` plus
`tools/cmd/validate-surface-visual-report`. The report requires deterministic
frame/golden/diff rows, token/theme conformance evidence, layout evidence,
accessibility evidence, performance evidence, same-commit golden heads, and
negative guards for screenshot-only, stale-golden, major-drift, missing Block
graph, missing layout, missing accessibility, and missing performance evidence.
The current visual gate covers `examples/surface_block_system.tetra` and the
five polished Block-only examples from `surface-block-examples.json` across
headless, linux-x64 real-window, and wasm32-web browser-canvas targets. It is
visual infrastructure evidence, not a production beauty claim, not a GPU
renderer claim, and not broad Electron renderer parity.

Morph rendered beauty has its own experimental contract in
`docs/spec/surface_morph_rendered_beauty.md`, with a machine-readable contract at
`docs/spec/surface_morph_rendered_beauty_contract.json`. It requires source-
linked Morph capsule/token/recipe evidence, a resolved Morph scene, a Block scene
snapshot, render command stream evidence, real frame artifacts, separate pixel
goldens, and product-claim guards before Morph can be described as the rendered
beauty layer.

The Morph token graph contract is validated by
`tools/cmd/validate-surface-token-graph` against
`docs/spec/surface_token_graph_contract.json`. It requires a single capsule
source of truth, typed color/space/radius/border/elevation/opacity/typography/
motion/z/assets/density categories, explicit namespaces/imports, no global
cascade, fixed `base -> theme -> density -> variant -> state -> local`
resolution order, density/DPI mappings, and negative diagnostics for raw
literals, missing tokens, duplicate sources, alias cycles, CSS cascade/runtime
admission, and multiple color sources. This is token/style boundary evidence
for the Block/Morph track, not production Morph support and not a CSS cascade.

### Experimental Block Data Model

`lib.core.block` is the first Block System code slice. It is a copy-safe data
model only: `BlockID`, `Block`, `BlockProps`, `LayoutSpec`, `PaintSpec`,
`PaintLayer`, `TextSpec`, `ImageSpec`, `InputSpec`, `EventSpec`, `StateSpec`,
`StateSelector`, `MotionSpec`, `AccessibilitySpec`, and `AssetRef`. By itself,
this module remains a data model rather than a production widget toolkit. The
separate Block-system gate now proves scoped graph/runtime/renderer/report
evidence for the current Block scenes and targets, but that evidence is still
experimental and bounded to the reported release artifacts.

Builder-style authoring uses ordinary Tetra values:

```text
import lib.core.surface as surface
import lib.core.block as block

let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
let id: block.BlockID = block.id(1)
let paint: block.PaintSpec =
    block.paint_from_layer(block.paint_layer_fill(surface.Color(r: 24, g: 32, b: 40, a: 255)))
let props: block.BlockProps =
    block.props(block.layout_fixed(rect), paint, block.text_label(18, surface.Color(r: 238, g: 242, b: 246, a: 255)), block.image_none(), block.input_clickable(), block.event_click(block.action_primary()), block.state_interactive(), block.motion_fast(), block.accessibility_button(18), block.asset_none())
let root: block.Block = block.make(id, block.id_none(), props)
```

## Principles

- User UI code is pure Tetra. No user JavaScript, HTML UI, React, DOM widget
  model, Qt, GTK, WinUI, Cocoa, or platform-specific widget code is part of the
  user-facing model.
- Widgets are not magical built-ins. Any Tetra struct can become a component by
  implementing Surface abilities such as measure, layout, draw, event, focus,
  text, and accessibility.
- The only platform boundary is a tiny Surface Host ABI. Hosts open a surface,
  poll events, report time, and present a framebuffer or draw-command buffer.
  Layout, hit testing, state, rendering rules, and event dispatch are owned by
  Tetra.
- Headless deterministic Surface is a release evidence target. Linux-x64 has a
  current real-window Wayland shm release path; wasm32-web has a current real
  browser-canvas release path with compiler-owned boot, DOM host canvas only,
  and no user JavaScript app logic or DOM-authored app UI tree.
- Browser Surface permits only the compiler-owned DOM document/canvas host
  needed by the web platform. That boot layer may be reported only as boot and
  canvas plumbing, not as application UI, a DOM widget tree, or user
  application logic.

## Core Types

The first library surface is expected to live in `lib/core/surface.tetra` and
`lib/core/draw.tetra`.

```text
Size        width and height in pixels
Point       integer pixel position
Rect        integer pixel rectangle
Color       RGBA color
Surface     host surface handle
Frame       borrowed frame buffer for one present cycle
Event       close, resize, pointer, key, future text, frame, and none events
DrawContext draw access to a live Frame
```

`Surface`, `Frame`, and `DrawContext` are resource-like values. They must not be
double-closed, used after present, stored globally, transferred across
task/actor boundaries, or allowed to leak borrowed framebuffer/text lifetime
outside the active event or frame. `Surface` handles are tracked as resource
identities across local aliases: once `surface.close(surface)` consumes one
alias, the checker rejects using another alias to close or redraw the same
handle. `Frame.surface` is a non-owning reference to the host surface, so
`surface.present(frame)` consumes the frame without consuming or closing the
surface handle. The checker still records the `Surface` owner used to create a
local `Frame`, so presenting that frame after its owner has been closed is
rejected as a use-after-close of the owner handle. Manual
`surface.Frame(surface: win, ...)` construction also records `win` as the
frame owner, including when the `surface` field is itself a tracked
`Surface(handle: win.handle, ...)` alias. The same owner tracking is preserved
through the allowed `draw.DrawContext.frame` wrapper, so
`surface.present(ctx.frame)` is also rejected after the owner `Surface` closes.
If a mutable `DrawContext` updates its `frame` field, the owner tracked for
`ctx.frame` updates with that assignment. `DrawContext.frame.pixels` aliases
carry the same canonical frame path, so a local alias of `ctx.frame.pixels`
cannot be used after `surface.present(ctx.frame)`.
Direct Host ABI calls are still treated as host-boundary code, not component
authoring style. When a direct call such as `core.surface_close(win.handle)`
uses the `handle` field of a tracked `Surface`, the checker consumes the owning
`Surface` value so raw Host ABI access cannot bypass close/use-after-close
diagnostics. Local `Int` aliases initialized from a tracked `Surface.handle`
preserve that owner provenance for direct Host ABI calls as well: `close`
consumes the owner, and non-consuming handle calls such as `request_redraw` or
`poll_event_into` require the owner to still be live. Tooling host probes may
still use raw `Int` handles where no `Surface` owner exists. Likewise, direct
`core.surface_present_rgba(..., frame.pixels, ...)` calls that use pixels from
a tracked `Frame` must obey the same owner and use-after-present rules as
`surface.present(frame)`: the owner `Surface` must still be live, and aliases
of that frame's pixels become unusable after the raw present call. Raw tooling
probes may still present ordinary caller-owned `[]u8` buffers that are not
derived from `Frame.pixels`.
Manual `surface.Surface(handle: win.handle, ...)` construction is also treated
as an alias of `win` when the handle comes from a tracked `Surface`, so user
code cannot forge a second live owner around the same host handle. Constructing
a `Surface` from an ordinary raw `Int` remains fresh low-level host-boundary
code.

The current checker enforces the first Surface lifetime guard for
`Frame`, `Event`, and `DrawContext`: these values may be local variables and
call arguments, but user code cannot store them in globals, user struct fields,
or user enum payloads, return them from functions, throw them through typed-error
boundaries, assign them through `inout` outputs, or capture them in function-typed
closure values. `Surface` handles plus `Frame`, `Event`,
and `DrawContext` values also cannot cross task or actor transfer boundaries:
typed task error payloads and typed actor message payloads that contain them
are rejected before slot-count promotion checks. The only starter exceptions are the core constructors
`lib.core.surface.begin_frame` and
`lib.core.surface.poll_event`, which are allowed to return the fresh per-turn
values they create, plus the `lib.core.draw.DrawContext` wrapper that carries
a live `Frame` only as an active draw call argument. `Frame.pixels` is also treated as a borrowed per-frame
buffer: user code cannot return it, throw it, return or throw a local alias of
it, assign it through an `inout` output, or keep using a local `[]u8` alias
after the owning `Frame` has been consumed by `surface.present`, including when
that frame is reached through `ctx.frame`. Draw helpers may still mutate
`ctx.frame.pixels` inside the active frame before
`surface.present`.

The current starter `Event` shape is a fixed caller-owned buffer record:
`kind`, coordinates, button/key fields, size, timestamp, and `text_len`.
`lib.core.surface.poll_event` uses `poll_event_into` to copy the host event
record into a Tetra-owned `[]i32` before constructing the public `Event`
value. Text payload bytes are copied by the host into caller-owned `[]u8`
buffers through `poll_event_text_into`; no borrowed host text lifetime is
exposed to user code. The first editable TextBox milestone is pure Tetra:
focus routing, focused keyboard routing, component-owned byte-buffer insertion,
caret movement, backspace/delete, and redraw evidence are implemented in
`examples/surface_textbox_app.tetra`. The release text-input baseline in
`examples/surface_release_text_input.tetra` adds owned UTF-8 byte-buffer
editing, invalid UTF-8 rejection, multiline byte storage, selection copy/paste,
host clipboard read/write, and IME/composition lifecycle traces. Full
String-level IME editing, rich text, bidi shaping, grapheme-cluster caret
movement, and a String-level `Event.text_input(str)` model remain future work.

## Host ABI

The initial ABI boundary is intentionally small:

```text
__tetra_surface_open(title_ptr, title_len, width, height) -> i32
__tetra_surface_close(surface_handle) -> i32
__tetra_surface_poll_event_kind(surface_handle) -> i32
__tetra_surface_poll_event_x(surface_handle) -> i32
__tetra_surface_poll_event_y(surface_handle) -> i32
__tetra_surface_poll_event_button(surface_handle) -> i32
__tetra_surface_poll_event_into(surface_handle, event_ptr, event_len) -> i32
__tetra_surface_poll_event_text_len(surface_handle) -> i32
__tetra_surface_poll_event_text_into(surface_handle, text_ptr, text_len) -> i32
__tetra_surface_begin_frame(surface_handle) -> i32
__tetra_surface_present_rgba(surface_handle, pixels_ptr, pixels_len, width, height, stride) -> i32
__tetra_surface_now_ms() -> i32
__tetra_surface_request_redraw(surface_handle) -> i32
```

At the Tetra slot ABI level, `String` and `[]u8` values are lowered as pointer
plus length, so `surface_open` uses 4 parameter slots and
`surface_present_rgba` uses 6 parameter slots. `surface_poll_event_into` and
`surface_poll_event_text_into` use the same caller-owned slice convention as
other host calls: surface handle, buffer pointer, and buffer length.

This starter ABI reports a compact event buffer plus scalar compatibility
helpers. The current event buffer has nine `i32` slots:
`kind,x,y,button,key,width,height,timestamp_ms,text_len`. It also exposes a
minimal text payload copy path needed by the deterministic counter smoke. The
headless and Linux-x64 starter hosts must prove a deterministic caller-owned
buffer sequence: first a pointer `mouse_up` record, then a `text_input` record
with host text length, then `none` records when the scripted queue is drained.
A richer event protocol with more event fields remains future work until
validated.

Target hosts must not know about `Button`, `Input`, `List`, or any other
component type. They do not perform layout, hit testing, platform widget
creation, or text rendering as a platform widget service.

## Component Model

The first component model started static. A component is an ordinary Tetra
struct whose methods satisfy the abilities the app uses:

```text
measure(self, max) -> Size
layout(self, rect) -> i32
draw(self, ctx) -> i32
event(self, event) -> i32
focus(self, focused) -> i32
text_input(self, event) -> i32
accessibility_role(self) -> i32
```

Static parent/child hierarchy is part of the starter evidence: a component
report names each component's layout `bounds`, may name a `parent`, and each
event records a root-to-target `dispatch_path`. The runtime counter dispatches
the host pointer event through `CounterApp` to `CounterButton`, and the strict
validator rejects reports where the pointer does not hit the target component
bounds. `CounterApp` still owns the state transition. Static text ability
evidence is scalar only: the counter handles a Tetra
`event_text_input` value with `text_len > 0`, copies deterministic host text
bytes into a caller-owned `[]u8`, and records a state transition.
Dynamic trait-object component lists, witness-table dispatch, full text editing
and IME payload modeling, platform accessibility tree integration, GPU rendering, and Tetra
Player remain future work unless promoted by later evidence.

The starter helper module is `lib/core/component.tetra`. It contains plain
layout/measurement structs and helpers such as `clamp_size`, `inset_rect`, and
`center_rect`; it does not register magic widget kinds with the compiler. The
semantic fixture `examples/surface_component_counter.tetra` demonstrates nested
ordinary structs implementing `measure`, `layout`, `draw`, `event`, `focus`,
`text_input`, and `accessibility_role` as extension methods. The runtime Surface
counter report also records `CounterApp` with `measure`, `layout`, `draw`,
`event`, `focus`, `text`, and `accessibility` abilities, plus a `CounterButton`
child component with layout bounds and root-to-child dispatch paths, so the
starter runtime evidence is tied to the same ordinary-struct component model.
The `examples/surface_text_input.tetra` fixture adds a user-defined `TextBox`
that owns a `[]u8` text buffer, receives deterministic host text payload bytes
through `surface_poll_event_text_into`, accepts them in its `text_input`
method, and presents a frame without any built-in text widget.
The `examples/surface_textbox_app.tetra` runtime fixture is the first editable
pure-Tetra TextBox layer. It keeps focus in a Tetra `FocusManager`, routes
clicks to `TextBox`, routes Tab to `SubmitButton`, sends key events only to the
focused component, inserts host text bytes into component-owned storage, tracks
the caret, handles left/backspace/delete, preserves focused state across
resize, and presents a changed RGBA frame after editing/focus changes.
This is static hierarchy, bounds-checked child-target event dispatch, scalar
text dispatch with caller-owned byte buffers, static focus dispatch, and static
accessibility metadata inside Tetra component state. The separate
`production-text-input-v1` report covers scoped clipboard and composition
baseline evidence; it is still not dynamic trait-object children, a platform
focus manager, full String/IME editor model, rich text, bidi shaping,
grapheme-cluster caret movement, or a platform accessibility API claim.

## Component Tree Evidence

`examples/surface_tree_app.tetra` adds the experimental component tree
milestone. The current implementation remains intentionally small, but the app
now builds its tree through the reusable `lib.core.component` helper API
instead of manually assigning structural fields. Tetra code owns a
`ComponentTree` plus stable `TreeNode` identities; reports still expose stable
node IDs, kind names, parent IDs, child positions, bounds, focusability, and
dispatch paths as evidence. Components remain ordinary Tetra structs such as
`TextLabel`, `TextBox`, `Button`, `Column`, `Row`, and `TreeApp`; no compiler
magic widgets, no DOM widgets, no platform widgets, and no production toolkit
claims are made.

The strict `component_tree` report block uses schema
`tetra.surface.component-tree.v1` inside the existing
`tetra.surface.runtime.v1` report. It records `dynamic_level =
semi-dynamic-child-list`, stable node IDs, parent/child links, layout passes,
draw order, focus order, and pointer dispatch paths. Required paths are:

```text
TextBox      0 -> 1 -> 3
SubmitButton 0 -> 1 -> 4 -> 5
ResetButton  0 -> 1 -> 4 -> 6
```

The milestone proves hit testing through the tree, root-to-leaf dispatch path
recording, Tab focus traversal
`TextBox -> SubmitButton -> ResetButton -> TextBox`, keyboard and text routing
to the focused component, reset/submit button action routing through focused
root-to-leaf tree paths, resize relayout from 320x200 to 400x240 while
preserving focus, and changed RGBA frame checksums after tree state changes.

## Component Tree API Hardening

The component tree API is still experimental, but authoring is no longer
allowed to depend on app-side structural bookkeeping. `lib.core.component`
provides ordinary pure-Tetra helpers for tree initialization and reset,
`tree_add_root`, `tree_add_child`, `tree_set_bounds`, `tree_child_at`,
`tree_validate`, `tree_layout_column`, `tree_layout_row`, `tree_focus_next`,
`tree_focus_prev`, `tree_hit_test`, `tree_build_dispatch_path`, and
`tree_build_draw_order`. The helpers own parent/child invariants, child lookup,
focus traversal, hit testing, and dispatch path construction for this
milestone's tree shape.

The app code should not manually write structural fields such as `id`,
`parent_id`, `first_child`, `child_count`, or future child-index storage.
Source-level tests enforce that `examples/surface_tree_app.tetra` uses
`component.tree_add_root`, `component.tree_add_child`,
`component.tree_layout_column`, `component.tree_layout_row`,
`component.tree_hit_test`, and `component.tree_build_dispatch_path` while
rejecting manual writes to those structural fields outside
`lib/core/component.tetra`.

API milestone reports keep the existing `component_tree` block and add a
`component_tree_api` block:

```json
{
  "schema": "tetra.surface.component-tree-api.v1",
  "api_level": "builder-layout-dispatch-v1",
  "source": "examples/surface_tree_app.tetra",
  "manual_bookkeeping": false
}
```

The full block proves builder calls, `tree_validate` invariant checks,
Column/Row layout helper use, focus helper traversal including
`ResetButton -> TextBox`, helper-routed hit tests, and dispatch path helper
output for TextBox, SubmitButton, and ResetButton. This hardening milestone is
not a final trait-object ABI, not witness-table dispatch, not a reactive tree,
not virtual DOM, not a production widget toolkit, not the stricter
`production-text-input-v1` gate, not rich text, not bidi shaping, not a
platform accessibility tree, not a GPU renderer, not Windows/macOS Surface, and
not production Surface promotion.

The validator rejects missing or fake tree evidence, path claims that skip a
parent container, unknown IDs, non-leaf click targets, child bounds outside a
parent, sibling Row overlap, Column visual order that contradicts
`child_index`, shuffled focus order, missing ResetButton-to-TextBox Tab wrap,
TextBox mutation while a Button is focused, button actions without a focused
keyboard routed event, resize claims without changed bounds, unchanged frame
checksums, source mismatches, missing `component_tree_api` evidence for API
reports, `manual_bookkeeping:true`, fake helper names, builder node-count
mismatches, missing `tree_validate` success evidence, missing Column/Row layout
helper evidence, helper hit-test paths that skip a parent, Node-only
browser-canvas claims, DOM/user-JS evidence, and legacy `.ui.*` sidecars.

This is not yet a final dynamic trait-object ABI, witness-table component
dispatch, reactive component tree, accessibility tree, GPU renderer, full
widget toolkit, or production Surface toolkit.

## Minimal Toolkit Evidence

`lib/core/widgets.tetra` adds the first experimental reusable widget helper
layer over the hardened `lib.core.component` tree API. It defines ordinary
Tetra structs for `Text`, `Button`, `TextBox`, `Row`, `Column`, and `Panel`
plus helper functions such as `add_panel`, `add_column`, `add_text`,
`add_textbox`, `add_row`, `add_button`, layout helpers, `hit_test`,
`textbox_text_input`, and `button_key_event`. These helpers are library code,
not compiler-known widgets, platform widgets, DOM widgets, user JavaScript, or
a production toolkit claim.

`examples/surface_toolkit_form.tetra` proves reuse of that module with this
shape:

```text
ToolkitFormApp
  Panel
    Column
      NameLabel
      TextBox
      ButtonRow
        SubmitButton
        ResetButton
      StatusText
```

The strict report adds a `toolkit` block with schema
`tetra.surface.toolkit.v1`, `toolkit_level = minimal-widgets-v1`,
`module = lib.core.widgets`, `experimental:true`, `production_claim:false`,
`uses_component_tree_api:true`, `manual_bookkeeping:false`, and widget evidence
for Panel, Column, Text, TextBox, Row, and Button. The same report still carries
`component_tree` and `component_tree_api` evidence, now with
`dynamic_level = minimal-toolkit-widget-tree` and root-to-leaf paths through
Panel and Column:

```text
TextBox      0 -> 1 -> 2 -> 4
SubmitButton 0 -> 1 -> 2 -> 5 -> 6
ResetButton  0 -> 1 -> 2 -> 5 -> 7
```

The milestone proves click focus, `OK` text insertion, caret movement,
backspace/delete, Tab focus cycling
`TextBox -> SubmitButton -> ResetButton -> TextBox`, Submit and Reset actions
routed through focused root-to-leaf paths, Reset clearing the TextBox,
StatusText updates, resize relayout, and changed frame checksums on headless,
linux-x64 real-window, and wasm32-web browser-canvas targets. It remains
experimental minimal widget evidence; no IME, no clipboard, no rich text, no
platform accessibility integration, no reactive UI framework support, and no
production Surface toolkit support are claimed.

## Toolkit Hardening + Reuse v1

`examples/surface_toolkit_settings.tetra` extends the experimental toolkit
slice from a single form into a second app shape using the same
`lib.core.widgets` module. The toolkit remains pure Tetra library code over
`lib.core.component`; it is not compiler-known widgets, DOM UI, user
JavaScript app logic, platform widgets, a reactive framework, a final
trait-object component ABI, or production Surface toolkit support.

The settings example uses reusable Panel, Column, Text, TextBox, Row, and
Button helpers with this evidence shape:

```text
ToolkitSettingsApp
  Panel
    Column
      TitleText
      NameTextBox
      NameLabel
      EmailTextBox
      ButtonRow
        SaveButton
        ResetButton
      StatusText
```

Reports keep `tetra.surface.runtime.v1`,
`tetra.surface.component-tree.v1`, and
`tetra.surface.component-tree-api.v1`, and extend
`tetra.surface.toolkit.v1` with `toolkit_level = toolkit-reuse-v1`,
`reuse_level = multi-form-widget-reuse-v1`, `example_count = 2`, sources for
both `examples/surface_toolkit_form.tetra` and
`examples/surface_toolkit_settings.tetra`, `text_box_count = 2`,
`button_count = 2`, `multi_textbox_evidence:true`, and
`multi_form_evidence:true`. The validator requires
`module = lib.core.widgets`, `experimental:true`, `production_claim:false`,
`uses_component_tree_api:true`, `manual_bookkeeping:false`,
`demo_specific_widget_structs:false`, `no_dom_ui:true`, and `no_user_js:true`.

The reuse scenario proves click focus on `NameTextBox`, text insertion into the
focused TextBox only, Tab traversal
`NameTextBox -> EmailTextBox -> SaveButton -> ResetButton -> NameTextBox`,
keyboard-routed Save and Reset actions through root-to-leaf paths, StatusText
updates after Save and Reset, Reset clearing both TextBoxes, resize relayout
from 320x240 to 480x320 while preserving focus, and changed frame checksums on
headless, linux-x64 real-window, and wasm32-web browser-canvas targets.

The strict validator rejects single-example reuse claims, missing
`lib.core.widgets` module evidence, production claims, demo-local widget
structs, manual tree bookkeeping, missing second-TextBox routing, unfocused
TextBox mutation, missing StatusText updates, resize claims without changed
bounds, unchanged frame checksums, Node-only browser claims, DOM/user-JS
claims, and missing artifact scans.

This milestone is still experimental. It does not include the later text-input
release gate for scoped clipboard/composition evidence. It also does not
include rich text, Unicode grapheme editing, bidi shaping, platform
accessibility host trees, GPU rendering, a virtual DOM, dynamic trait-object
widgets, witness-table component dispatch, or toolkit promotion.

## Accessibility Metadata Tree v1

`lib/core/accessibility.tetra` adds the first experimental pure-Tetra
accessibility metadata layer over `lib.core.component` and
`lib.core.widgets`. The layer records metadata for a Tetra-owned widget tree:
roles, names/labels, values, state flags, bounds, parent-child relationships,
label relationships, focus order, reading order, actions, status updates, and
snapshots. It is metadata only. It does not export to Linux AT-SPI, macOS AX,
Windows UI Automation, browser DOM/ARIA accessibility, screen readers, or
platform widget accessibility hosts.

`examples/surface_accessibility_settings.tetra` isolates the milestone with
this shape:

```text
AccessibilitySettingsApp
  Panel
    Column
      TitleText
      NameLabel
      NameTextBox
      EmailLabel
      EmailTextBox
      ButtonRow
        SaveButton
        ResetButton
      StatusText
```

The strict reports keep `tetra.surface.runtime.v1`,
`tetra.surface.component-tree.v1`, `tetra.surface.component-tree-api.v1`, and
`tetra.surface.toolkit.v1`, then add
`accessibility_tree.schema = tetra.surface.accessibility-tree.v1` with
`accessibility_level = metadata-tree-v1`, `module = lib.core.accessibility`,
`widget_module = lib.core.widgets`, `experimental:true`,
`production_claim:false`, `platform_host_integration:false`,
`dom_aria_integration:false`, `screen_reader_evidence:false`,
`derived_from_component_tree:true`, `uses_component_tree_api:true`,
`uses_widget_toolkit:true`, `manual_bookkeeping:false`, `no_dom_ui:true`,
`no_user_js:true`, `no_platform_widgets:true`, and `no_legacy_sidecars:true`.

The required tree contains 12 aligned component/accessibility nodes with one
root, Panel, Column, TitleText, NameLabel, NameTextBox, EmailLabel,
EmailTextBox, ButtonRow, SaveButton, ResetButton, and StatusText. Required
relationships are `NameLabel -> NameTextBox`, `EmailLabel -> EmailTextBox`,
and the matching `labelled_by` edges. Focus order is
`NameTextBox -> EmailTextBox -> SaveButton -> ResetButton -> NameTextBox`;
reading order is `TitleText, NameLabel, NameTextBox, EmailLabel,
EmailTextBox, SaveButton, ResetButton, StatusText`.

The metadata snapshots cover initial state, NameTextBox focus/text,
EmailTextBox focus/text, Save, Reset, and resize to 480x320. The validator
requires value changes, status changes, metadata checksum changes, bounds
checksum changes after resize, stable reading/focus order across resize, and
changed frame checksums for UI-changing events. It rejects missing trees,
wrong schemas, unknown roles, duplicate or unknown component IDs, bounds
mismatches, missing labels, shuffled focus or reading order, multiple focused
nodes, static-only snapshots, unchanged checksums, Node-only browser claims,
DOM/ARIA or user-JS evidence, legacy `.ui.*` sidecars, platform accessibility
host claims, no screen-reader claims, and no production accessibility claims;
those claims are rejected, not promoted.

The evidence runs on headless, linux-x64 real-window, and wasm32-web
browser-canvas/input targets. It remains experimental metadata evidence;
production accessibility support, platform accessibility integration, browser
ARIA/DOM accessibility, and screen-reader validation are not claimed.

## Evidence

Surface promotion requires `tetra.surface.runtime.v1` reports with executable
process evidence, explicit `host_evidence`, frames, events, state transitions,
checksums, and strict validator rejection for:

- docs-only reports;
- metadata-only `tetra.ui.v1` bundles;
- old native-shell sidecar-only evidence;
- web-only DOM evidence;
- build-only evidence;
- fake, mock, placeholder, stale, or startup-failure reports;
- legacy `.ui.html`, `.ui.web.mjs`, `.ui.json`, `tetra.ui.v1`, DOM UI, HTML
  UI, user JavaScript, or user JS markers;
- missing frame, event, state transition, checksum, or executable process
  evidence;
- process evidence without a build command tied to the reported Tetra source
  path, or without an executable Surface component app process with the
  expected app exit;
- missing `component-app` artifact hash evidence linked to the executable
  Surface component app process;
- wasm32-web reports without a `compiler-owned-loader` `.mjs` artifact hash,
  or reports that list generated HTML/JavaScript UI artifacts;
- missing `artifact_scan` evidence proving the artifact directory containing
  the reported artifacts was scanned and had zero forbidden legacy UI/HTML/JS
  sidecar paths;
- component type evidence that does not match the reported Tetra source module
  path;
- missing `host_evidence` or starter evidence that claims real-window/native
  input promotion;
- missing positive `no legacy UI sidecar artifacts` evidence;
- for `examples/surface_textbox_app.tetra`, missing click focus, Tab focus
  routing, keyboard routing to the focused component, text insertion into
  component-owned storage, caret movement, backspace/delete, resize preserving
  focus, or visible framebuffer update evidence.
- for `examples/surface_tree_app.tetra`, missing `component_tree` evidence,
  node count, parent/child links, layout bounds, draw traversal, root-to-leaf
  dispatch paths, focus traversal, focused TextBox text routing, Button action
  routing, resize relayout, or visible framebuffer update evidence; for API
  hardening reports, missing `component_tree_api` schema
  `tetra.surface.component-tree-api.v1`,
  `api_level = builder-layout-dispatch-v1`, `manual_bookkeeping:false`, builder,
  invariant, layout, focus, hit-test, dispatch-path helper evidence, or matching
  source evidence.
- for `examples/surface_toolkit_form.tetra`, missing `toolkit` schema
  `tetra.surface.toolkit.v1`, `toolkit_level = minimal-widgets-v1`,
  `production_claim:false`, reusable widget evidence for Panel, Column, Text,
  TextBox, Row, Button, and StatusText, `uses_component_tree_api:true`,
  `manual_bookkeeping:false`, root-to-leaf Button dispatch paths, TextBox edit
  routing, StatusText transitions, resize relayout, changed frame checksums, or
  rejection of DOM/user-JS/platform-widget/magic-widget claims.
- for `examples/surface_toolkit_settings.tetra`, missing
  `toolkit_level = toolkit-reuse-v1`, missing
  `reuse_level = multi-form-widget-reuse-v1`, sources for both toolkit
  examples, two TextBox widgets, two Button widgets, focused-only text routing,
  Save/Reset action evidence, StatusText transitions, resize relayout for both
  TextBoxes, changed frame checksums, or rejection of single-example,
  demo-local-widget, manual-bookkeeping, Node-only browser, DOM/user-JS, or
  missing-artifact-scan evidence.
- for `examples/surface_accessibility_settings.tetra`, missing
  `accessibility_tree` schema `tetra.surface.accessibility-tree.v1`,
  `accessibility_level = metadata-tree-v1`, `module = lib.core.accessibility`,
  `widget_module = lib.core.widgets`, the exact 12-node settings tree, label
  and labelled-by relationships, NameTextBox/EmailTextBox/SaveButton/
  ResetButton focus order, reading order, edit/press/save/reset actions,
  snapshots for text, focus, save, reset, and resize, metadata and bounds
  checksum changes, or rejection of production/platform-host/DOM/ARIA/
  screen-reader/user-JS/Node-only/legacy-sidecar/manual-bookkeeping evidence.

`host_evidence` names the evidence level and backend instead of relying only on
target names:

```json
{"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}
{"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}
{"level":"linux-x64-real-window","backend":"wayland-shm-rgba","framebuffer":true,"real_window":true,"native_input":true,"user_facing_platform_widgets":false}
{"level":"wasm32-web-compiler-owned-loader","backend":"node-surface-host","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}
{"level":"wasm32-web-browser-canvas-input","backend":"browser-canvas-rgba","framebuffer":true,"real_window":false,"native_input":true,"user_facing_platform_widgets":false}
```

The validator rejects Linux-x64 memfd starter reports that claim
`real_window:true` or `native_input:true`. A Linux-x64 real-window report must
use `level:"linux-x64-real-window"` with `framebuffer:true`,
`real_window:true`, and `native_input:true`, and it must use a backend that is
not the memfd starter. It must also include executable evidence that cannot be
satisfied by the memfd starter: an app process named like
`surface linux-x64 real-window probe` that exits `42`, positive
`linux-x64 real-window surface`, `linux-x64 native input event pump`,
`linux-x64 real-window resize event`, and `linux-x64 real-window close event`
case evidence, plus a presented 400x240 frame checksum.

The scoped Linux app-shell release report is
`surface-linux-x64-release-app-shell.json` with
`linux_app_shell.schema:"tetra.surface.linux-app-shell.v1"` and
`app_shell_level:"linux-app-shell-subset-v1"`. It must use
`examples/surface_linux_app_shell_notes.tetra`, real Linux host evidence, two
presented windows (`notes-main` and `notes-inspector`), lifecycle open/close/
reopen rows, resize and DPI rows, pointer/text/resize cursor transitions,
clipboard read/write, IME composition start/update/commit/cancel,
accessibility platform bridge evidence, and host trace artifacts for the
app-shell and accessibility paths. `app_shell_features:"electron-feature-ledger-v1"`
records supported/scoped rows for `app_menu`,
`window_lifecycle`, `multi_window`, `clipboard`, `ime`,
`accessibility_bridge`, `crash_recovery`, and `error_report`. `dialog`,
`file_dialog`, `file_picker`, `notification`, `tray`, and `deep_link` are
accepted only as `blocked_pass` nonclaims until target-host evidence exists.
The same report must include `security_permissions` with
`schema:"tetra.surface.security-permission.v1"` and
`model:"surface-security-permission-v1"`. That section is default-deny for
ambient filesystem and network access, maps every `electron-feature-ledger-v1`
row to a capability policy decision, checks Surface app-to-host IPC/process
boundaries, permits clipboard only through the scoped host policy, keeps
notifications/dialogs/shell-open-url denied until target evidence exists, and
requires local-only hashed font/image/icon assets with bounded parsers. It
does not grant unrestricted filesystem/network access, native permission
prompts, remote asset fetching, or Electron Node integration.
The same report must include `surface_performance_budget` with
`schema:"tetra.surface.performance-budget.v1"` and
`model:"surface-performance-budget-v1"`. That section records local deterministic
startup-to-first-frame, p50/p95 frame build/present, scene counts, memory/RSS/
cache/framebuffer, binary size, CPU/power proxy, bounded-cache, stale-report
rejection, and mandatory `peak_rss_bytes` evidence. It is not an official
benchmark and cannot claim unsupported Electron speed superiority.
The release gate must also include `surface-dev-workflow.json` with
`schema:"tetra.surface.dev-workflow.v1"` and
`model:"surface-dev-workflow-v1"`. That report is produced by
`tetra surface dev` and records the developer fast rebuild loop: initial build,
warm-cache rebuild with cache hits, token-change, recipe-change, source-change,
positioned source diagnostics, and artifact hashes. It is fast rebuild evidence
only; it does not claim hot reload, an Electron dev server, React Fast Refresh,
or a browser-DOM application UI reload loop.
The release gate must also include `surface-inspector.json` with
`schema:"tetra.surface.inspector.v1"` and `model:"surface-inspector-v1"`.
That report is produced by `tools/cmd/surface-inspector` from validated Surface
runtime reports and exposes Block tree, resolved Morph tokens, layout, paint,
accessibility, event route, focus, and perf-counter state. It also records
source locations, input report coverage, hidden-state scan results, JSON output,
and an optional static HTML tool report. It is not browser devtools, React
devtools, DOM runtime UI, hidden app state, an interactive runtime inspector, or
a replacement for target-host accessibility evidence.
The release gate must also include `surface-template-smoke.json` with
`schema:"tetra.surface.template-smoke.v1"` and
`model:"surface-template-smoke-v1"` for generated Block/Morph onboarding
projects.
The release gate must also include `surface-reference-apps.json` with
`schema:"tetra.surface.reference-app-suite.v1"` and
`model:"surface-reference-app-suite-v1"`. That report covers ten product
shapes: command palette, settings, dashboard, editor shell, file
manager/list-detail, dialog/notification, localized form,
accessibility-heavy form, multi-window notes, and migration. Each source must
check, build, run, use stable Morph recipes that resolve to Block, and carry
headless/linux/web visual, interaction, accessibility, performance,
token/theme, layout, and artifact-hash evidence. `lib.core.widgets` is allowed
only in the migration compatibility app.
The release gate must also include `surface-package.json` with
`schema:"tetra.surface.package.v1"` and `model:"surface-package-v1"`. That
report records `surface-app-package-v1` package manifests, local asset hashes,
linux-x64 and wasm32-web tar.gz packages, installed linux-x64 package execution
for the default reference app or explicitly named product-slice app,
web bundle HTML/wasm/compiler-owned loader output, and a hash-pinned
`tetra.surface.update-channel.v1` manifest. The `studio-shell` flagship package
uses an explicit expected app-state exit code in its install smoke. Signing,
notarization, automatic runtime updates, and network update fetching remain
nonclaims until platform signing/notarization evidence and runtime updater
evidence exist.
The release gate must also include `surface-crash-report.json` with
`schema:"tetra.surface.crash-report.v1"` and
`model:"surface-crash-report-v1"`. That report records bounded linux-x64
`command_failure`, `host_crash`, and `restart_recovery` scenarios for the
command-palette reference app, local `tetra.surface.diagnostic.v1` artifacts,
bounded ring-buffer trace/log collection, `surface-non-user-data-diagnostics-v1`
privacy policy evidence, and `scoped-linux-x64-process-restart-v1`
before/report/after restart proof. It is not a claim that all crashes recover,
and it must reject user data leaks, clipboard/user-text/env/home capture,
network upload, docs-only crash claims, Electron crash reporter dependency, and
restart claims without evidence.

The release gate must include `surface-i18n.json` with
`schema:"tetra.surface.i18n.v1"` and `model:"surface-i18n-v1"`. That report
records bounded string tables, `uk-UA` locale selection with `en-US` fallback,
missing-key diagnostic evidence, deterministic date/number formatting hooks,
localized-form reference app execution, and
`rtl-placeholder-without-full-bidi-shaping-v1`. It is not a full ICU, full bidi
shaping, RTL production text-layout, platform locale dependency, or third-party intl
runtime claim, and validators reject docs-only localization claims and silent
missing-key fallback.

The release gate must include `surface-widget-migration.json` with
`schema:"tetra.surface.widget-migration.v1"` and
`model:"surface-widget-migration-v1"`. That report keeps `lib.core.widgets`
supported as a Surface v1 compatibility layer, preserves the current release
widget set, proves Panel/Button/TextBox equivalence rows against Morph recipes
that resolve to Block, runs the migration reference app, and records Block as
the only core primitive. It is not a future core widget primitive promotion, a
breaking API change, a docs-only migration claim, or a platform toolkit/runtime
claim.

Validators reject
GTK/Qt/native widget UI, Electron/React runtime use, DOM UI, user JavaScript
application logic, platform widgets, build-only evidence, docs-only evidence,
and copied reports without matching local artifacts. This is a bounded Linux
Surface app-shell claim, not broad desktop-shell parity.

The first required scripts are:

```text
scripts/release/surface/gate.sh
scripts/release/surface/surface-headless-smoke.sh
scripts/release/surface/surface-linux-x64-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-smoke.sh
scripts/release/surface/surface-wasm32-web-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh
scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh
scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh
scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh
scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh
scripts/release/surface/surface-package-smoke.sh
```

`scripts/release/surface/gate.sh` is the aggregate experimental Surface gate:
it runs the headless, Linux-x64 starter, Linux-x64 real-window, wasm32-web
starter, wasm32-web browser canvas/input, the three TextBox focus/text input
smoke scripts, the three component-tree scripts, the three component-tree API
hardening scripts, the three minimal toolkit scripts, the three toolkit reuse
scripts, and the three accessibility metadata scripts into one report
directory, revalidates every
`tetra.surface.runtime.v1` report, then writes and validates the final artifact
hash manifest.

```text
scripts/release/surface/surface-headless-text-focus-input-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-text-focus-input-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh
scripts/release/surface/surface-headless-component-tree-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-component-tree-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-smoke.sh
scripts/release/surface/surface-headless-component-tree-api-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-component-tree-api-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh
scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh
scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh
scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh
scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh
scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh
```

The headless entrypoint is:

```text
go run ./tools/cmd/surface-runtime-smoke --mode headless --report reports/surface/surface-headless.json
go run ./tools/cmd/validate-surface-runtime --report reports/surface/surface-headless.json
```

It emits `tetra.surface.runtime.v1` evidence for the pure-Tetra
`examples/surface_counter.tetra` component app, including executable process
evidence, deterministic host-provided pointer event dispatch, a state
transition, component layout bounds, root-to-child dispatch paths, and distinct
pre-event and post-event RGBA frame checksums. Those
checksums are SHA-256 over deterministic headless framebuffer bytes before and
after the Surface event updates state, not hashes of metadata or prose. The
headless gate builds and runs the Surface app binary before accepting the
report, scans the artifact directory for legacy UI/HTML/JS sidecars, records a
positive `no legacy UI sidecar artifacts` case, records a positive
`headless actual runner trace` case backed by a hashed
`surface-runner-trace.json` artifact with schema
`tetra.surface.headless-runner-trace.v1`; `validate-surface-runtime` checks
that trace `source` matches the reported source and that trace frames match the
reported Surface frame evidence. The gate records a positive
`host-provided pointer event dispatch` case, records a positive
`host event buffer poll_event` case for the pointer/text event-buffer sequence,
records a positive
`pre/post event frame sequence` case, records a positive
`component hierarchy dispatch` case with bounds-checked `dispatch_path`
evidence, records a positive
`component text input scalar dispatch` case, records a positive
`host text payload buffer` case, records a positive
`component focus dispatch` case, records a positive
`component accessibility metadata` case, and the validator rejects
source/metadata paths as executable app process evidence. The validator also
requires a build process path that references the reported source and a
`surface component app` process with the expected application exit, requires a
`component-app` artifact entry with `sha256:<hex>` and size evidence linked to
that process path, then derives the source module from the reported
`.tetra`/`.t4` path and rejects reports whose component types are not from that
source module. That keeps component evidence tied to the app that was actually
built and run.

For wasm32-web, the report must additionally contain a
`compiler-owned-loader` `.mjs` artifact hash. This artifact is the compiler
boot/runtime bridge for wasm instantiation and Surface Host ABI imports; it is
not user application logic. Surface reports reject generated HTML artifacts,
legacy `.ui.*` sidecars, and non-loader JavaScript artifacts.

Every report also carries an `artifact_scan` record with the scanned root,
number of checked files, empty `forbidden_paths`, and `pass: true`. The checked
file count must be at least the number of reported artifact records, so the
positive `no legacy UI sidecar artifacts` case is backed by the actual artifact
directory scan, not only by a case label. Each reported artifact path must live
under that scanned root, so a report cannot hash artifacts from one directory
while scanning a different clean directory.

The Linux-x64 starter gate now runs the same pure-Tetra counter app through the
native `linux-x64` target and also builds a pure-Tetra host probe that succeeds
only when `surface_open` returns a kernel-backed handle, `surface_present_rgba`
can present RGBA bytes through that handle, and `surface_close` really closes
it. The gate also builds a pure-Tetra event-sequence probe that calls
`surface_poll_event_into` three times through the Linux host ABI and must see
the deterministic pointer, key, then resize records before exiting `42`. The
starter Linux host is deliberately tiny: it uses `memfd_create`, `lseek`,
`write`, and `close` behind the Surface Host ABI. This is executable non-stub
host evidence without GTK/Qt/OS widget exposure or metadata sidecar playback,
and with the same no-legacy-sidecar artifact scan. The gate also runs a
long-lived pure-Tetra 2x2 RGBA probe, reads the kernel-backed memfd through
`/proc/<pid>/fd/*`, and records a third frame checksum plus positive
`linux-x64 host event sequence` and
`linux-x64 app-presented RGBA checksum` cases. It also builds a long-lived
pure-Tetra counter presentation probe, verifies the CounterApp/CounterButton
after-event 320x200 RGBA frame through the same memfd readback path, records
that checksum as frame order 4, and requires a positive
`linux-x64 counter component app-presented frame` case. It is not yet a full
real-window desktop Surface or native event pump, and its report is marked
`host_evidence.level:"linux-x64-memfd-starter"` rather than real-window
evidence.

The Linux-x64 real-window gate builds
`examples/surface_window_counter.tetra` and emits
`surface-linux-x64-real-window.json`. The pure-Tetra app opens a Surface,
presents a counter/button frame, consumes click and key events to update state,
handles resize by updating layout width, consumes a host text payload, presents
an updated frame, then consumes close and exits cleanly. The companion Wayland
shm probe opens a real Linux window, sets a title/app id, presents a Tetra-owned
400x240 RGBA framebuffer, and exits `42`. The strict report uses
`host_evidence.level:"linux-x64-real-window"` and
`backend:"wayland-shm-rgba"`, records click/key/resize/text/close events,
records the real-window frame as order 5, and is rejected if the evidence is
headless, memfd-only, docs-only, build-only, metadata-only, legacy `.ui.*`,
DOM/web-only, fake, or stale.
The companion
`surface-linux-x64-real-window-text-focus-input-smoke.sh` builds
`examples/surface_textbox_app.tetra` and emits
`surface-linux-x64-real-window-text-focus-input.json` with the same real-window
promotion level plus TextBox focus/text/caret/backspace/delete evidence.

The Linux-x64 app-shell release gate builds
`examples/surface_linux_app_shell_notes.tetra` and emits
`surface-linux-x64-release-app-shell.json`. The report proves scoped
multi-window notes app-shell behavior on the target host: open/close/reopen
lifecycle, two presented windows, resize/DPI/cursor traces, clipboard and
IME/composition bridge traces, accessibility platform bridge evidence,
app-menu adapter evidence, `electron-feature-ledger-v1`, scoped crash/error
reporting adapters, local redacted `surface-crash-report-v1` evidence,
bounded `surface-i18n-v1` localization evidence, and
bounded `surface-widget-migration-v1` compatibility evidence, and
blocked-pass nonclaims for dialog, file dialog, file picker, notification, tray,
and deep link. The strict validator requires local
artifact hashes plus `surface-security-permission-v1` evidence for default-deny
filesystem/network permissions, capability-checked IPC/process boundaries, and
local hashed asset/font/image safety, plus `surface-performance-budget-v1`
startup/frame/memory/RSS/cache/framebuffer/binary-size/CPU-proxy evidence. It
rejects
GTK/Qt/native widget UI, Electron/React runtimes, DOM UI, user JavaScript app
logic, platform widgets, unrestricted filesystem/network access, remote asset
fetching, unsupported Electron speed comparisons, headless-only evidence,
build-only evidence, and docs-only evidence for this subset.

The starter `wasm32-web` Surface slice now builds
`examples/surface_counter.tetra` as pure Tetra into `.wasm` plus a
compiler-owned `.mjs` bootloader. The wasm module imports only the ordinary
`tetra_web_v1` console/panic helpers and the strict
`tetra_surface_host_v1.__tetra_surface_*` Host ABI allowlist; the legacy
`.ui.json`, `.ui.web.mjs`, and `.ui.html` sidecars are not emitted for the
Surface counter. The Node web runner provides the same tiny host ABI for
runtime smoke execution, including the starter scalar pointer event payload,
minimal text payload copy into caller-owned memory, and an optional
compiler-owned `tetra.surface.web-runner-trace.v1` file that records actual
`__tetra_surface_present_rgba` frame dimensions, stride, pixel length, and
SHA-256 checksum from wasm memory without printing to stdout. The runtime
validator requires that web trace schema for wasm32-web reports, requires the
trace `wasm_path` to match the reported `.wasm` component artifact, and maps
its runner frame orders back to the reported Surface frames. The
`tools/cmd/validate-wasm-imports` rejects imports outside that allowlist.
The `surface-wasm32-web-smoke.sh` gate emits a strict
`tetra.surface.runtime.v1` report with process, pre/post frame,
host-provided event, state-transition, compiler-owned loader, import-allowlist,
actual presented frame trace, and no-legacy-sidecar evidence. This is still not
full production browser Surface promotion: it proves the starter Node web
runner path, and the compiler-owned JavaScript boot is not user application
logic.
The report artifacts include the `.wasm`, compiler-owned `.mjs` loader, and
`runner-trace` JSON file hashes so `validate-surface-runtime` can recompute
their local SHA-256 and size before accepting the evidence.

The wasm32-web browser canvas/input gate builds
`examples/surface_browser_counter.tetra` and runs it in a real Chromium-
compatible browser canvas through
`scripts/tools/surface_browser_canvas_host.mjs`, a compiler/smoke-owned host
runner rather than user JavaScript application logic. The pure-Tetra app opens
a Surface, presents Tetra-owned RGBA framebuffer bytes to a real
`HTMLCanvasElement`, reads the canvas pixels back, consumes pointer, key,
resize, and text-input browser events through
`tetra_surface_host_v1.__tetra_surface_*`, updates Tetra-owned
count/key/layout/text state, then presents a 400x240 updated frame. The trace
schema is `tetra.surface.browser-canvas-trace.v1`; it records browser-native
event types, the `.wasm` path, canvas open/readback evidence, app exit code,
and per-frame source/canvas SHA-256 checksums. The strict validator accepts
this only with
`host_evidence.level:"wasm32-web-browser-canvas-input"`,
`backend:"browser-canvas-rgba"`, a Chromium-compatible app process, frame
order 5 at 400x240, pointer/key/resize/text report events, the exact Host ABI
import allowlist, hashed `.wasm`, compiler-owned loader, and runner-trace
artifacts, plus the same no-legacy-sidecar scan. Starter Node evidence,
DOM-only/user-JS evidence, metadata-only evidence, build-only evidence, fake,
stale, and legacy `.ui.*` sidecars do not satisfy this evidence level.
The companion
`surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh` builds
`examples/surface_textbox_app.tetra`, dispatches real browser pointer,
`beforeinput`, ArrowLeft, Backspace, Delete, Tab, Space, and resize events
through the compiler-owned browser canvas host, and emits
`surface-wasm32-web-browser-canvas-text-focus-input.json`.

Surface migration fixtures now exist for the legacy metadata examples:

- `examples/surface_migration_ui_web_smoke.tetra`
- `examples/surface_migration_ui_native_shell_smoke.tetra`
- `examples/surface_migration_dogfood_web_ui.tetra`
- `examples/surface_migration_tetra_control_center.tetra`

These are migration fixtures for the pure-Tetra object model. They show ordinary
struct state, `draw`/`event` abilities, synthetic scalar events, and local
frame presentation without metadata sidecars. The native smoke matrix now builds
and runs them through the Linux-x64 Surface Host ABI with deterministic exits
`2`, `11`, `3`, and `5`. They do not by themselves promote Linux-x64
real-window or production browser Surface support.

WASM Surface uses no user JavaScript app logic and no DOM-authored app UI tree.
The current browser boot is compiler-owned JavaScript because the web platform
still needs a loader and a document-owned canvas; that boot must be reported as
boot/canvas plumbing, not as user application logic or DOM widgets. Absolute
no-JS browser launch remains future/blocked unless browsers expose direct wasm
boot and surface/event integration without JavaScript.
