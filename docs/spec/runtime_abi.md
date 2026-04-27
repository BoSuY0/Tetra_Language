# Runtime ABI (reserved `__tetra_*` symbols)

This document defines the ABI contract between a compiled Tetra program and a linked runtime object.

It is primarily used by:
- the embedded self-host actors runtime (`compiler/selfhostrt/actors_*.tetra`)
- the built-in actors runtime (`compiler/internal/actorsrt/*`)
- `tetra build --runtime-object <path.tobj>` (runtime override)
- `tetra build --link-object <path.tobj>` (additional TOBJ libraries)

## Reserved symbols

Symbols starting with `__tetra_` are reserved for the toolchain/runtime.

User code may only export reserved names from internal runtime modules (modules whose name starts with `__`) via
`@export("...")`. See `@export` rules in the language semantics.

## Calling convention per target

- `linux-x64`, `macos-x64`: SysV AMD64 ABI (first args in `rdi, rsi, rdx, rcx, r8, r9`)
- `windows-x64`: Windows x64 ABI (first args in `rcx, rdx, r8, r9`, plus 32-byte shadow space)

All functions in this document return the first slot in `rax`/`eax`. Two-slot internal returns use `rax` for the
low/first slot and `rdx` for the second slot. The backend spills incoming arguments into Tetra local slots in declaration
order before lowering the function body.

Stack arguments begin after the register argument window:

- SysV (`linux-x64`, `macos-x64`): argument 7 is read from `[rbp+16]`, argument 8 from `[rbp+24]`, and so on.
- Win64 (`windows-x64`): argument 5 is read from `[rbp+48]` after the return address and the 32-byte shadow space.

Calls preserve the platform alignment contract. SysV calls align `rsp` to 16 bytes before `call`; Win64 calls reserve
the mandatory 32-byte shadow space and aligns around additional stack arguments. Current ABI regression tests cover calls
with 0 through 8 arguments and return layouts with 0, 1, and 2 slots.

Unsupported ABI/runtime combinations are hard errors. For example, `ctx_switch` currently supports only the SysV Unix and
Win64 x64 ABIs; using another ABI for that instruction reports `ctx_switch: unsupported ABI`.

## Native executable format contracts

### `linux-x64`

The Linux backend emits a minimal ELF64 executable:

- ELF magic/version identify a little-endian x86-64 executable.
- The file has two `PT_LOAD` program headers: one RX segment for headers/text and one RW segment for data.
- The current writer intentionally does not emit an ELF section header table; release validation treats the program
  headers as the executable layout contract.
- String/data relocations point into the RW segment, not the RX text bytes.
- Output files are written executable.

### `macos-x64`

The macOS backend emits a build-verified Mach-O 64-bit x86-64 executable:

- Header magic, CPU type, and file type are checked structurally.
- The load command contract is `__TEXT`, `__DATA`, and `LC_MAIN`.
- `__TEXT,__text` contains executable code and `__DATA,__cstring` contains string data.
- Data relocations point to `__DATA,__cstring`.
- Cross-host execution is not attempted on non-macOS hosts; release evidence is build-only unless collected on macOS.

### `windows-x64`

The Windows backend emits a PE32+ x86-64 executable:

- Required sections are `.text`, `.rdata`, `.idata`, and `.reloc`.
- The entrypoint is inside `.text`.
- The import directory is inside `.idata`.
- The default import contract is `KERNEL32.dll` with `ExitProcess`, `GetStdHandle`, and `WriteFile`; runtime features add
  imports such as `VirtualAlloc`, `VirtualFree`, and `VirtualProtect`.
- PE output enables NX-compatible and dynamic-base characteristics and includes a relocation directory.
- Cross-host execution is not attempted on non-Windows hosts; release evidence is build-only unless collected on Windows.

## WASM target ABI contracts

### `wasm32-wasi`

The WASI backend emits a deterministic WebAssembly module with:

- WASM magic `\0asm` and version 1.
- Imports from `wasi_snapshot_preview1`: `fd_write` and `proc_exit`.
- Exports: `memory` and `_start`.
- Unsupported native runtime instructions are rejected at link/codegen time with an explicit `wasm backend` diagnostic.

`tetra smoke --target wasm32-wasi --run=false` is build-only release evidence. Runner evidence is produced separately by
`scripts/release_v1_0_wasi_smoke.sh`; when a runner fallback is used, the smoke report records that runner.

### `wasm32-web`

The web backend emits a deterministic WebAssembly module plus a JavaScript loader contract:

- Imports from `tetra_web_v1`: `console_log(ptr, len)` and `panic(code, ptr, len)`.
- Exports: `memory` and `tetra_main`.
- The loader fetches the `.wasm` module relative to `import.meta.url`, wires `tetra_web_v1`, and exposes
  `instantiateTetra()` plus `runTetra()`.
- Unsupported native runtime instructions are rejected at link/codegen time with an explicit `wasm backend` diagnostic.

`tetra smoke --target wasm32-web --run=false` is build-only release evidence. Full browser automation evidence is produced
by `scripts/release_v1_0_web_smoke.sh` and remains host/browser dependent.

## Actors runtime surface (MVP)

The actors MVP links a runtime object that exports:

### `__tetra_entry() -> i32`

Process entrypoint. Returns the program exit code.

The platform linker stubs call `__tetra_entry` and then terminate the process using the OS mechanism
(syscall on SysV Unix, `ExitProcess` on Windows).

### `__tetra_actor_spawn(entryID: i32) -> actor`

Spawns a new actor.

- `linux-x64`, `macos-x64`: `entryID` in `edi`
- `windows-x64`: `entryID` in `ecx`

Returns an `actor` handle in `eax` (`-1` for failure in the current MVP implementation).

### `__tetra_actor_send(to: actor, v: i32) -> i32`

Sends a message to another actor.

- `linux-x64`, `macos-x64`: `to` in `edi`, `v` in `esi`
- `windows-x64`: `to` in `ecx`, `v` in `edx`

Returns `v` in `eax` (MVP convenience).

### `__tetra_actor_recv() -> i32`

Receives a message value from the current actor mailbox (blocking/yielding cooperatively until a message exists).

### `__tetra_actor_self() -> actor`

Returns the current actor handle in `eax`.

### `__tetra_actor_sender() -> actor`

Returns the sender of the most recently received message in `eax` (valid only after a successful recv).

The compiler validates these required runtime exports before linking:

- `__tetra_entry`
- `__tetra_actor_spawn`
- `__tetra_actor_send`
- `__tetra_actor_recv`
- `__tetra_actor_self`
- `__tetra_actor_sender`

## Program-provided symbols

When actors are used, the compiler links (or generates) a small “glue” object that provides:

### `__tetra_actor_dispatch(entryID: i32) -> i32`

Dispatches an `entryID` to the corresponding actor entry function and returns its exit code.

This function is called by the runtime using the **platform ABI**:
- `linux-x64`, `macos-x64`: `entryID` in `edi`
- `windows-x64`: `entryID` in `ecx`

### `__tetra_actor_main_entry_id() -> i32`

Returns the FNV-1a 32-bit entry ID for the program main entry function (the same value as `FNV1a32(<main symbol name>)`).

This is provided so that alternate runtimes (including self-hosted ones) can spawn/run the main entry without hardcoding
the program symbol name.

## Actor entry IDs

`core.spawn(name: str)` is lowered by the compiler into a call to `__tetra_actor_spawn(entryID)` where `entryID` is the
FNV-1a 32-bit hash of the string literal used as `name`.

The runtime uses the same hash scheme to dispatch actor entrypoints.

## Internal runtime helpers

The toolchain may expose a small set of `core.*` builtins for internal runtime modules (modules whose name starts with `__`)
to call program-provided glue symbols without requiring explicit declarations:

- `core.actor_dispatch(entryID: i32) -> i32` (calls `__tetra_actor_dispatch`)
- `core.actor_main_entry_id() -> i32` (calls `__tetra_actor_main_entry_id`)

## Actor entry functions (user code)

Actor entry functions are regular Tetra functions with the shape:

- `fun <name>(): i32`

They are called by the runtime using the platform ABI for the current target.

## Runtime override and target matching

When using `--runtime-object`, the runtime `.tobj` must match the program target (for example, a `windows-x64` runtime
object must not be linked into a `linux-x64` executable).

Runtime override objects must also export every required actor runtime symbol.
The compiler rejects missing targets, target mismatches, and missing runtime
exports before platform linking.

`--runtime=auto` selects the embedded self-host runtime when actor builtins are used. `--runtime=selfhost` forces that
path, and `--runtime=builtin` keeps the Go-emitted runtime available as a compatibility fallback.

Native execution is only supported when `host == target`; cross-target builds are build-verified but not run on
non-matching hosts.

## ABI compatibility policy

The v1 runtime ABI is source-stable for the reserved symbols listed in this
document and metadata-stable for TOBJ files that declare one of the supported
native triples. A compatible runtime object must:

- use the target's platform calling convention exactly as listed above;
- export all required actor runtime symbols with the reserved `__tetra_` prefix;
- set `target` to the final program target;
- avoid redefining program glue symbols or user symbols from linked libraries;
- preserve the meaning of actor handles and `i32` message values.

The compiler rejects runtime objects with missing targets, target mismatches, or
missing required symbols before platform linking. It also build-verifies runtime
override objects for `linux-x64`, `macos-x64`, and `windows-x64`; real execution
evidence is only claimed on matching hosts.

## Additional linked objects

`--link-object path.tobj` appends an additional target-matching TOBJ library to the final link. The flag is repeatable.
Linked objects participate in the same symbol table as compiler-generated objects, so duplicate exported symbols and
unresolved relocations are reported by the linker.

## TOBJ metadata contract

TOBJ objects carry enough metadata for target-safe linking:

- `target`: required target triple such as `linux-x64`, `macos-x64`, or
  `windows-x64`.
- `module`: producer module name, used for diagnostics and object identity.
- `code`: raw text/code bytes for the target object fragment.
- `data`: raw data bytes for globals and constants.
- `symbols`: exported or internal symbol names with code/data offsets.
- `relocs`: relocation records naming the target symbol and relocation kind.

The linker accepts repeated `--link-object` flags when all objects match the
target and have non-conflicting symbols. Target mismatches, duplicate symbols,
and unresolved symbols are hard errors.
