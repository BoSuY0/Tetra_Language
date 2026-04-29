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

### `__tetra_actor_send_msg(to: actor, v: i32, tag: i32) -> i32`

Sends a tagged message to another actor. This is the runtime ABI for
`core.send_msg(to, value, tag)`.

- `linux-x64`, `macos-x64`: `to` in `edi`, `v` in `esi`, `tag` in `edx`
- `windows-x64`: `to` in `ecx`, `v` in `edx`, `tag` in `r8d`

Returns `v` in `eax` (MVP convenience).

### `__tetra_actor_send_begin(to: actor, tag: i32, slot_count: i32) -> i32`

Starts a multi-slot actor message send. The runtime records the destination,
tag, and payload slot count for the current send transaction.

### `__tetra_actor_send_slot(index: i32, value: i32) -> i32`

Writes one payload slot into the active send transaction.

### `__tetra_actor_send_commit() -> i32`

Commits the active multi-slot actor message send.

### `__tetra_actor_recv() -> i32`

Receives a message value from the current actor mailbox (blocking/yielding cooperatively until a message exists).

### `__tetra_actor_recv_msg() -> actor.msg`

Receives a tagged message from the current actor mailbox. This is the runtime
ABI for `core.recv_msg()`.

The return value uses the two-slot internal return convention: `value` in
`eax`/`rax` and `tag` in `edx`/`rdx`.

### `__tetra_actor_recv_poll() -> actor.recv_result_i32`

Performs a nonblocking receive from the current actor mailbox. If a message is
available, consumes it and returns `value` with `error = 0`. If the mailbox is
empty, returns `value = 0` and `error = 2` without blocking or yielding.

### `__tetra_actor_recv_until(deadline: i32) -> actor.recv_result_i32`

Receives a message value from the current actor mailbox before an absolute
logical deadline. If a message is available first, returns `value` with
`error = 0`. If the deadline is reached first, returns `value = 0` and
`error = 2`.

The return value uses the two-slot internal return convention: `value` in
`eax`/`rax` and `error` in `edx`/`rdx`.

### `__tetra_actor_recv_msg_until(deadline: i32) -> actor.recv_msg_result`

Receives a tagged message from the current actor mailbox before an absolute
logical deadline. Success returns `value`, `tag`, and `error = 0`. Timeout
returns `value = 0`, `tag = 0`, and `error = 2`.

The return value uses the three-slot internal return convention: `value` in
`eax`/`rax`, `tag` in `edx`/`rdx`, and `error` in `r8d`/`r8`.

### `__tetra_actor_recv_begin() -> i32`

Receives the next multi-slot message and returns its tag.

### `__tetra_actor_recv_slot(index: i32) -> i32`

Reads one payload slot from the most recently received multi-slot message.

### `__tetra_actor_recv_count() -> i32`

Returns the payload slot count for the most recently received multi-slot
message.

### `__tetra_actor_self() -> actor`

Returns the current actor handle in `eax`.

### `__tetra_actor_sender() -> actor`

Returns the sender of the most recently received message in `eax` (valid only after a successful recv).

### `__tetra_actor_yield_now() -> i32`

Cooperatively yields the current actor without changing its status, then returns
`0` when the scheduler resumes it. This is the runtime ABI for `core.yield()`.

The compiler validates these required runtime exports before linking:

- `__tetra_entry`
- `__tetra_actor_spawn`
- `__tetra_actor_send`
- `__tetra_actor_send_msg`
- `__tetra_actor_send_begin`
- `__tetra_actor_send_slot`
- `__tetra_actor_send_commit`
- `__tetra_actor_recv`
- `__tetra_actor_recv_msg`
- `__tetra_actor_recv_poll`
- `__tetra_actor_recv_until`
- `__tetra_actor_recv_msg_until`
- `__tetra_actor_recv_begin`
- `__tetra_actor_recv_slot`
- `__tetra_actor_recv_count`
- `__tetra_actor_self`
- `__tetra_actor_sender`
- `__tetra_actor_yield_now`

## Task runtime surface

Task joins are cooperative scheduler waits, not busy loops. When
`__tetra_task_join_i32`, `__tetra_task_join_result_i32`,
`__tetra_task_join_until_i32`, or a typed task join sees that the target actor
is not done, the current actor records the target handle, enters
`waiting_task`, and yields. The scheduler wakes that actor when the target
reaches `done`. Deadline-aware joins also wake when the absolute deadline is
due. If the target task group is already canceled when a join begins,
result-style joins return the cancellation error immediately; tasks that observe
cancellation internally may still run checkpoint/defer code and finish with a
normal task value.

`core.select2_i32(task, deadline)` is the first wait-composition surface. It
uses the same runtime behavior as `__tetra_task_join_until_i32`: task completion
wins with `error = 0`; the timer wins with `error = 2`.

Typed task joins in the current MVP are emitted for slot counts `2..8`:
direct ABI returns for `2..4` (`__tetra_task_join_typed_2..4`) and staged
runtime-buffer joins for `5..8` (`__tetra_task_join_typed_5..8` plus
`__tetra_task_result_get`). One-slot typed handles reuse the existing
`task.i32` join path, and typed layouts above `8` are rejected during semantic
checking. Worker targets remain zero-argument synchronous `i32` functions; for
`2..4` they must throw the typed error enum, and for staged `5..8` they may be
either non-throwing or throw the same typed error enum.

### `__tetra_task_join_until_i32(handle: i32, error: i32, deadline: i32) -> task.result_i32`

Waits for a single-slot task until an absolute logical deadline. Completion
returns `task.result_i32(value: task_exit, error: 0)`. An invalid incoming task
handle propagates its incoming `error`, and timeout returns `value = 0` with
`error = 2`.

### `__tetra_task_poll_i32(handle: i32, error: i32) -> task.result_i32`

Checks a single-slot task without blocking. A completed task returns
`task.result_i32(value: task_exit, error: 0)`. A still-running task returns
`value = 0` with `error = 2`. An invalid incoming task handle propagates its
incoming `error`.

## Time runtime surface

Programs that call `core.time_now_ms`, `core.sleep_ms`, `core.sleep_until`,
`core.deadline_ms`, or `core.timer_ready` link the runtime object even when no
actor or task builtin is used. The current runtime clock is deterministic and
logical: it starts at `0` for each process, `sleep_ms` parks the current
actor/task until a non-negative relative logical deadline, `sleep_until` parks
until an absolute logical deadline, `deadline_ms` returns
`now + max(delta, 0)`, and `timer_ready` checks an absolute deadline without
parking.

The scheduler tracks actor/task states as `ready`, `blocked_recv`,
`waiting_task`, `sleeping`, `done`, and task-group `canceled`. `core.send`
wakes actors blocked in
`core.recv`; it does not wake sleeping actors. If no actor is ready and at least
one actor has a timed sleep, receive, or join deadline, the scheduler advances
the logical clock to the nearest deadline and wakes every actor due at that
time. Sleeping actors in a canceled task group become ready immediately so join
can observe the cancellation result.

### `__tetra_time_now_ms() -> i32`

Returns the current logical runtime time in milliseconds.

### `__tetra_sleep_ms(ms: i32) -> i32`

If `ms <= 0`, returns `0` immediately. Otherwise stores
`__tetra_time_now_ms() + ms` as the current actor/task wake deadline, marks it
sleeping, yields to the runtime scheduler, then returns `0` when the actor/task
is resumed. This is cooperative/deterministic; it does not claim wall-clock
sleeping.

### `__tetra_sleep_until_ms(deadline: i32) -> i32`

If `deadline <= __tetra_time_now_ms()`, returns `0` immediately. Otherwise
stores the absolute deadline as the current actor/task wake deadline, marks it
sleeping, yields to the runtime scheduler, then returns `0` when resumed.

### `__tetra_deadline_ms(delta: i32) -> i32`

Returns `__tetra_time_now_ms() + max(delta, 0)`.

### `__tetra_timer_ready_ms(deadline: i32) -> bool`

Returns whether `__tetra_time_now_ms() >= max(deadline, 0)` without yielding.

The compiler validates these time runtime exports when a program uses the time
builtins:

- `__tetra_time_now_ms`
- `__tetra_sleep_ms`
- `__tetra_sleep_until_ms`
- `__tetra_deadline_ms`
- `__tetra_timer_ready_ms`

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

Runtime override objects must also export every required runtime symbol set
used by the program: actor runtime symbols, actor-state symbols
(`__tetra_actor_state_load`, `__tetra_actor_state_store`) when actor state is
used, task/task-group/typed-task symbols when those builtins are used, and time
runtime symbols when the program calls `core.time_now_ms`, `core.sleep_ms`,
`core.sleep_until`, or `core.deadline_ms`. The compiler rejects missing
targets, target mismatches, and missing runtime exports before platform linking.

`--runtime=auto` currently selects the embedded self-host runtime only for the
mailbox-only actor surface, and switches to the built-in runtime when actor
state, task/task-group, typed-task, or time builtins are used. This remains
true even though self-host now exports parity symbols; auto mode is still
conservative. `--runtime=selfhost` forces the self-host path, and
`--runtime=builtin` keeps the Go-emitted runtime available as a compatibility
fallback.

Native execution is only supported when `host == target`; cross-target builds are build-verified but not run on
non-matching hosts.

## ABI compatibility policy

The v1 runtime ABI is source-stable for the reserved symbols listed in this
document and metadata-stable for TOBJ files that declare one of the supported
native triples. A compatible runtime object must:

- use the target's platform calling convention exactly as listed above;
- export all required actor runtime symbols, and any used time runtime symbols,
  with the reserved `__tetra_` prefix;
- set `target` to the final program target;
- avoid redefining program glue symbols or user symbols from linked libraries;
- preserve the meaning of actor handles, `i32` message values, and tagged
  `actor.msg` / `actor.recv_result_i32` two-slot returns.

The compiler rejects runtime objects with missing targets, target mismatches, or
missing required symbols before platform linking. It also build-verifies runtime
override objects for `linux-x64`, `macos-x64`, and `windows-x64`; real execution
evidence is only claimed on matching hosts.

## Additional linked objects

`--link-object path.tobj` appends an additional target-matching TOBJ library to the final link. The flag is repeatable.
Linked objects participate in the same symbol table as compiler-generated objects, so duplicate exported symbols and
unresolved relocations are reported by the linker.

When a program imports a module through `.t4i`, a regular native build may use
`--link-object` as that module's implementation provider. The provider object
must declare the same `module`, target, compiler-version-compatible metadata,
and public API hash as the interface file. The compiler rejects missing
providers, duplicate providers for the same interface module, public API hash
mismatches, and missing required function symbols before platform linking.

## TOBJ metadata contract

TOBJ objects carry enough metadata for target-safe linking:

- `target`: required target triple such as `linux-x64`, `macos-x64`, or
  `windows-x64`.
- `module`: producer module name, used for diagnostics and object identity.
- `compiler_version`: compiler version that produced the object, used for
  compatibility diagnostics.
- `public_api_hash`: deterministic hash of the module's generated `.t4i`
  public surface.
- `code`: raw text/code bytes for the target object fragment.
- `data`: raw data bytes for globals and constants.
- `symbols`: exported or internal symbol names with code/data offsets.
- `relocs`: relocation records naming the target symbol and relocation kind.

The linker accepts repeated `--link-object` flags when all objects match the
target and have non-conflicting symbols. Target mismatches, duplicate symbols,
and unresolved symbols are hard errors.

## Native x64 build-only and mismatch policy (Epic 09)

- `linux-x64`, `macos-x64`, and `windows-x64` native outputs are build-verified in the same matrix for ABI/object/link
  contracts; execution is still host-gated.
- Platform linker wrappers enforce target identity at link entry:
  - Linux linker accepts only `linux-x64` objects,
  - macOS linker accepts only `macos-x64` objects,
  - Windows linker accepts only `windows-x64` objects.
- Cross-target object usage through wrong linker path is a hard diagnostic (`linker target mismatch`).
- Compiler-level `--link-object`/`--runtime-object` target checks remain in force and fail before final image writing.
