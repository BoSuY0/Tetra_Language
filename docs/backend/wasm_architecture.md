# WASM Object and Runtime Architecture

Status: accepted for TODO 660 (2026-04-26)

Related v1 scope-freeze decisions for unresolved backend/UI rollout items:
`../plans/v1_scope_freeze_backend_stdlib_ui.md`.

This document defines the concrete WASM backend architecture that must be used before changing target metadata in `compiler/target/target.go`.

## Decision Summary

- Compilation unit model: one Tetra build unit produces one WebAssembly module; v1 has no separate relocatable `.o` stage and no native-style linker pass.
- Internal object model: the backend builds a deterministic in-memory WASM object (`WOBJ`) and serializes it directly to `.wasm`.
- Runtime model: single-threaded, single linear memory, explicit host imports only, and target-specific entry wrappers (`_start` for WASI, JS-called export for web).
- Packaging: `wasm32-wasi` emits one `.wasm`; `wasm32-web` emits `.wasm` plus a deterministic JS loader module.
- Host bindings: WASI imports only from `wasi_snapshot_preview1`; web imports only from `tetra_web_v1`.
- Release gates: WASM support is blocked until the gate commands in this document are real and green.
- UI boundary in this architecture wave: metadata preview only. `wasm32-web` may mount `tetra.ui.v1` metadata in a preview shell, while runtime event dispatch/layout engines remain post-v1.

## Concrete Object Model

The backend object is `WOBJ` (in-memory only in v1) with stable ordering rules:

- `types`: deduplicated function signatures in first-use order.
- `imports`: target-specific imports in lexical order by `(module,name)`.
- `functions`: internal functions in deterministic symbol order; each contains locals and instruction stream.
- `memory`: exactly one linear memory declaration.
- `data`: deterministic data segments for literals/readonly blobs, sorted by symbol.
- `exports`: deterministic export list.
- `names` (custom section): emitted only in debug builds, sorted by function index.

Relocation/link policy:

- Calls are resolved by function index assignment inside the same module.
- Data addresses are resolved during final layout as absolute linear-memory offsets.
- Unresolved symbols are allowed only for configured host imports; everything else is a compile error.

## Runtime Model

Execution model:

- Single-threaded runtime for v1. No threads, shared memory, or atomics.
- No GC runtime in v1; values are lowered to scalar/register-local WASM forms plus explicit linear-memory data.
- Trap/panic path is deterministic and target-specific through explicit host bindings.

Linear memory contract:

- Exactly one memory named/exported as `memory`.
- Static data starts at offset `0x1000`.
- Heap base global `__tetra_heap_base` is set to the first 16-byte-aligned offset after static data.
- Offset range `0x0000..0x0FFF` is reserved for null/sentinel/trap-adjacent checks and never used for program data.

Entry contract:

- `wasm32-wasi`: module exports `_start`; `_start` calls lowered Tetra entry and returns via WASI process semantics.
- `wasm32-web`: module exports `tetra_main`; JS loader instantiates the module and invokes `tetra_main`.

## Packaging Model

Given output base path `<out>/<name>`:

- `wasm32-wasi` output:
  - `<out>/<name>.wasm`
- `wasm32-web` outputs:
  - `<out>/<name>.wasm`
  - `<out>/<name>.mjs` (deterministic loader/import adapter)

Packaging constraints:

- No bundler requirement in compiler output.
- No host-specific executable container (no ELF/Mach-O/PE analog) for WASM targets.
- Build determinism is measured on emitted `.wasm` bytes and loader text for web target.

## Host Binding Contract

### `wasm32-wasi`

Allowed imports (v1):

- `wasi_snapshot_preview1.fd_write`
- `wasi_snapshot_preview1.proc_exit`

Policy:

- Any additional WASI import requires explicit architecture-doc update before implementation.
- Host access must remain effect-gated; code without the required effect must fail before backend emission.

### `wasm32-web`

Allowed imports (v1), module `tetra_web_v1`:

- `console_log(ptr:i32, len:i32) -> void`
- `panic(code:i32, ptr:i32, len:i32) -> void`

Policy:

- Browser APIs are accessed only through these imports in v1.
- DOM/event-loop expansion is deferred until UI MVP runtime slices are approved.
- UI sidecar behavior for v0.2.0: `.ui.web.mjs` and `.ui.html` are metadata preview artifacts and must validate `tetra.ui.v1` schema before rendering.

### UI Sidecar Boundary

- `wasm32-wasi` must not emit web/native UI runtime sidecars (`.ui.web.mjs`, `.ui.html`, `.ui.shell.txt`).
- `wasm32-web` mounts metadata and reports preview output, but does not execute UI event dispatch semantics in runtime.
- Native targets may emit `.ui.shell.txt` as deterministic metadata text only.

## Gate Commands (Must Stay Mandatory)

These commands define the architecture gate for enabling real WASM targets:

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-targets
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
./tetra smoke --target wasm32-wasi --run=true
bash scripts/release_v1_0_gate.sh
```

TODO 660 is considered resolved when this architecture remains the referenced contract and target/backends changes follow it without reintroducing planned-target placeholders.
