# UI v1 Surface

Status: v1.0 required metadata UI surface.

This document defines the UI syntax and backend artifact contract that is in
scope for v1.0. It intentionally describes a metadata-first UI surface: the
compiler validates UI declarations, lowers them to deterministic metadata, and
emits preview artifacts for web and native shell targets.

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

Checked UI declarations lower to a deterministic `tetra.ui.v1` JSON bundle
containing states, fields, views, bindings, events, commands, styles, and
accessibility metadata.

When a build contains a view:

- all targets emit `<output>.ui.json`;
- `wasm32-web` also emits `<output>.ui.web.mjs` and `<output>.ui.html`;
- native targets emit `<output>.ui.shell.txt`.

## Backend Status

`wasm32-web` is the v1 browser preview backend. The generated web module reads
the UI JSON bundle and mounts a simple DOM representation before running
`tetra_main`.

Native shell UI is a v1 metadata preview backend. It renders the same validated
state/view metadata into a deterministic text sidecar. It is release-supported
as an artifact contract and smoke target, not as a full native widget toolkit.

`wasm32-wasi` in this wave remains non-UI runtime: it may compile UI metadata
for artifact inspection, but it does not ship web/native UI preview sidecars
and does not provide UI event dispatch behavior.

Current smoke/dogfood expectation:

- `examples/projects/dogfood_web_ui/src/main.tetra` exercises web metadata UI.
- `examples/ui_web_smoke.tetra` and `examples/ui_native_shell_smoke.tetra` stay
  as metadata-oriented UI source fixtures.
- `examples/projects/dogfood_wasi/src/main.tetra` stays intentionally non-UI for
  WASI runner/build-only evidence.

## Post-v1

Native widgets, layout engines, command dispatch at runtime, richer event
payloads, styling systems, and accessibility integration with platform APIs are
post-v1 unless promoted by a reviewed scope update.
