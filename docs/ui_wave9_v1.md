# Wave 9 UI v1 Syntax

Wave 9 introduces two top-level declarations:

```tetra
state CounterState:
    var count: Int = 0
    val title: String = "Counter"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"
```

## `state`

- Declares typed UI state.
- Field forms: `var`, `val`, `const`.
- Every field requires a type and initializer.

## `view`

- Declares a UI view bound to one `state` type.
- Sections supported in v1:
  - `bind <name>: <type> = <expr>`
  - `event <name> -> <command>`
  - `command <name>:` + statement block
  - `style <name>: <type> = <expr>`
  - `accessibility <name>: <type> = <expr>` (alias: `a11y`)

## v1 checking rules

- View must declare at least one command.
- Event targets must reference an existing command.
- Style and accessibility values are typed (`Int`/`Bool`/`String`).
- Commands cannot `return`/`throw`.
- Commands cannot write immutable state fields.

## Backend artifacts

When a build includes at least one `view`:

- All targets: `<output>.ui.json`
- `wasm32-web`: `<output>.ui.web.mjs`, `<output>.ui.html`
- Native (`linux-x64`, `macos-x64`, `windows-x64`): `<output>.ui.shell.txt`
