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

All functions in this document return values in `rax`/`eax` as usual for the platform ABI.

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

`--runtime=auto` selects the embedded self-host runtime when actor builtins are used. `--runtime=selfhost` forces that
path, and `--runtime=builtin` keeps the Go-emitted runtime available as a compatibility fallback.

Native execution is only supported when `host == target`; cross-target builds are build-verified but not run on
non-matching hosts.

## Additional linked objects

`--link-object path.tobj` appends an additional target-matching TOBJ library to the final link. The flag is repeatable.
Linked objects participate in the same symbol table as compiler-generated objects, so duplicate exported symbols and
unresolved relocations are reported by the linker.
