# UI v0.4.0 Surface

Status: current `v0.4.0` metadata UI surface with separately gated post-v0.4
browser-backed Web UI and Linux-x64 native runtime smoke paths. This does not
claim GTK/Qt/OS widget backends, macOS/Windows native UI runtimes, or platform
accessibility integration.

This document defines the UI syntax and backend artifact contract that is in
scope for the `v0.4.0` metadata contract. It intentionally describes a
metadata-first UI surface: the compiler validates UI declarations, lowers them
to deterministic metadata, and emits deterministic runtime glue for web plus
native shell artifacts when the relevant gated paths are exercised.

## Syntax

UI source files may declare `state` and `view` at top level:

```tetra doctest
state CounterState:
    var count: Int = 0
    val title: String = "Counter"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    bind titleText: String = state.title
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment counter"

func main() -> Int:
    return 0
```

`state` fields may use `var`, `val`, or `const`. Every state field requires a
type and initializer.

`view` declarations bind to exactly one state type and support:

- `bind <name>: <type> = <expr>`
- `event <name> -> <command>`
- `command <name>:` followed by a statement block
- `style <name>: <type> = <expr>`
- `accessibility <name>: <type> = <expr>`

`a11y` is accepted as an alias for `accessibility`.

## Checking Rules

- Each view must declare at least one command.
- Events must reference commands declared in the same view.
- Binding values must match their declared type.
- Style and accessibility values support `Int`/`i32`, `Bool`/`bool`, and
  `String`/`str`.
- Commands may mutate `var` state fields.
- Commands must not mutate `val` or `const` state fields.
- Commands must not `return` or `throw`.

## Lowered Metadata

Checked UI declarations lower to a deterministic `tetra.ui.v0.4.0` JSON bundle
containing states, fields, views, bindings, events, commands, styles, and
accessibility metadata.

When a build contains a view:

- all targets emit `<output>.ui.json`;
- `wasm32-web` also emits `<output>.ui.web.mjs` and `<output>.ui.html`;
- native targets emit `<output>.ui.shell.txt`.

## Backend Status

`wasm32-web` is the bounded browser command-dispatch runtime backend. The
generated web module reads the UI JSON bundle, mounts a deterministic DOM
representation before running `tetra_main`, dispatches supported DOM events to
lowered command operations, and refreshes scalar state bindings. The current lowered scalar
operation set includes direct state assignment plus integer increment and
decrement patterns of the form `state.field = state.field +/- <integer>`.
The same integer delta operations are emitted for supported `+=` and `-=`
compound assignments.
String, boolean, and integer-like assignments are hydrated as scalar runtime
values rather than raw source literals, and same-state field assignments copy
the current source field value in command order.
The web runtime also mirrors supported style and accessibility metadata into
DOM preview attributes such as `data-tetra-style-*`,
`data-tetra-accessibility-*`, `role`, and `aria-label`; full styling/layout
engines and platform accessibility API integration remain outside this surface.
Passing web UI smoke evidence must carry the production runtime trace markers
for window/root/layout/text/button/input/list/panel/focus/input/change/select/
click/timer/async/redraw/error recovery plus
`ui-event-dispatch:web-command-dispatch`.

Native shell UI is a deterministic text-mode command-dispatch preview backend.
It renders the same validated state/view metadata into a sidecar, hydrates
scalar bindings from the lowered initial state, dispatches each declared event
through its lowered command operations, applies supported scalar state
updates, including direct assignment plus integer increment/decrement, and
records the resulting binding values. It also writes a machine-readable
`tetra.ui.native-shell.v1` JSON trace sidecar containing the same runtime,
event, operation, state-field, and post-dispatch binding evidence. The JSON
sidecar also includes a deterministic `widgets` array for each view: binding
widgets record hydrated display values plus style/accessibility metadata, and
event widgets record the action-to-command dispatch entrypoint. It is a
production artifact contract and smoke target for the native shell preview, not
a full platform widget toolkit.
Validate native shell JSON traces with
`go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json`.
The validator requires `tetra.ui.native-shell.v1`, `tetra.ui.v0.4.0`, native shell
command-dispatch runtime identity, state/view evidence, event operation traces,
post-dispatch bindings, and both binding/action widgets.

Linux-x64 native UI runtime evidence is a separate production gate from the
native shell sidecar. `scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh`
builds the current CLI, builds `examples/ui_native_shell_smoke.tetra` for
`linux-x64`, runs the native executable, loads the generated
`tetra.ui.native-shell.v1` sidecar into the native runtime smoke process, and
emits `reports/v0.4.0/native-ui-linux-x64.json` with schema
`tetra.ui.native-runtime.v1`. That report records:

- runtime widget instances with stable IDs, parent/child hierarchy, bounds,
  text/value state, enabled state, and visible state;
- click dispatch from an action widget to the lowered command operation path;
- ordered repeated events with before/after state maps and widget updates;
- negative invalid widget, malformed metadata, unsupported event, and command
  failure cases;
- runtime lifecycle close.

Validate the production native runtime report with:

```sh
go run ./tools/cmd/validate-native-ui-runtime --report reports/v0.4.0/native-ui-linux-x64.json
```

The native runtime validator rejects metadata-only, web-only, native-shell
sidecar-only, fake/mock/placeholder, missing event execution, and missing state
transition evidence. macOS/Windows native UI runtime claims require separate
host-native reports and are not promoted by the Linux-x64 report.

`wasm32-wasi` in this wave remains non-UI runtime: it may compile UI metadata
for artifact inspection, but it does not ship web/native UI preview sidecars
and does not provide UI event dispatch behavior.

Current smoke/dogfood expectation:

- `examples/projects/dogfood_web_ui/src/main.tetra` exercises browser-backed
  web runtime UI.
- `examples/ui_web_smoke.tetra` and `examples/ui_native_shell_smoke.tetra` stay
  as metadata-oriented UI source fixtures.
- `examples/projects/dogfood_wasi/src/main.tetra` stays intentionally non-UI for
  WASI runner and artifact/import preflight evidence.

## v0.4.0 Evidence Snapshot

The base `v0.4.0` release snapshot was limited to metadata and preview
artifacts. The post-v0.4 WASM/UI/GUI promotion requires fresh runtime-backed
gate evidence under `reports/wasm-ui-gui`; checked-in historical artifacts do
not count as production evidence. The table below records older evidence paths
only as provenance.

| Evidence field | Value |
| --- | --- |
| Web UI smoke report | `reports/plan250/backend/web-ui-smoke.json` |
| Web UI source | `examples/projects/dogfood_web_ui/src/main.tetra` |
| Web UI status/result | `pass`; `ok:0:ui=1` |
| UI schema | `tetra.ui.v0.4.0` |
| Native shell trace schema | `tetra.ui.native-shell.v1` |
| Native shell trace validator | `go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json` |
| Native Linux-x64 runtime report | `reports/v0.4.0/native-ui-linux-x64.json` |
| Native Linux-x64 runtime validator | `go run ./tools/cmd/validate-native-ui-runtime --report reports/v0.4.0/native-ui-linux-x64.json` |
| Native Linux-x64 runtime schema | `tetra.ui.native-runtime.v1` |
| UI bundle/module/DOM | `reports/plan250/backend/web-ui-smoke.ui.json`; `reports/plan250/backend/web-ui-smoke.ui.web.mjs`; `reports/plan250/backend/web-ui-smoke.dom.html` |
| Lowered metadata content | 1 state, 1 view, bindings `countValue`/`titleText`, event `click -> increment`, styles `width`/`theme`, accessibility `role`/`label` |
| WASI runner report | `reports/plan250/backend/wasi-smoke.json` |
| WASI runner status | target `wasm32-wasi`, runner `node-wasi`, total `5`, passed `5`, failed `0` |
| WASM artifact/import reports | `reports/plan250/backend/wasm32-wasi-artifact-smoke.json`; `reports/plan250/backend/wasm32-web-artifact-smoke.json` |

## Post-v0.4 Production Gate

The bounded WASM/UI/GUI promotion gate is:

```sh
bash scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh --report-dir reports/wasm-ui-gui
```

It collects fresh WASI/Web artifact and runtime smoke, validates WASM imports,
runs browser-backed Web UI smoke, validates `tetra.ui.v0.4.0` runtime evidence,
runs Linux-x64 native UI runtime smoke, validates `tetra.ui.native-runtime.v1`,
and writes artifact hashes. This does not promote macOS/Windows GUI or full
toolkit/runtime guarantees.

## Post-v1

GTK/Qt/OS widget toolkit backends, macOS/Windows native UI runtime reports,
richer event payloads, broad input/change/focus behavior, full styling/layout
systems, and accessibility integration with platform APIs remain post-v1 unless
promoted by a reviewed scope update.
