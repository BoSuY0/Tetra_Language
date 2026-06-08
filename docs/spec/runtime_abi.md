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

The compiler rejects non-runtime modules that export a reserved `__tetra_*`
name. Runtime override and link-object validation also treat TOBJ metadata as
part of the ABI: target mismatches are hard errors, duplicate link object paths
are hard errors, and link objects carrying a non-matching compiler version are
rejected before linking.

## User `@export` FFI boundary

Non-runtime `@export` functions are part of the native FFI surface, so target
ABI gaps must fail before object code is written. On native targets, exported
parameters or return types that require aggregate C ABI handling, such as
structs, arrays, slices, strings, enums, and optionals, are rejected with a
target-aware diagnostic until the target has verified aggregate C ABI lowering.
Export a scalar wrapper or provide a target-specific runtime object with a
verified ABI instead.

Scalar `@export` wrappers are build-verified in the x86/x64/x32 target suites:
the emitted TOBJ must contain the exported symbol with signature metadata for
the selected target and must not collect relocations from a different platform
ABI. The Linux family suites now cover canonical `ptr` wrappers plus
source-level `c_int`/`c_uint` wrappers, and the ILP32 Linux suites also cover
`usize`, `isize`, `size_t`, `ssize_t`, `native_int`, `native_uint`, `c_long`,
and `c_ulong` wrappers as 1-slot 32-bit source scalars on `linux-x86` and
`linux-x32`. The same suites also build
target-specific atomic object smoke tests so the scalar FFI surface is checked
alongside the target's lock-free atomic code generation contract.
The OS-specific x64 ABI suite cells also build object smoke tests for
`macos-x64` and `windows-x64`, keeping SysV Mach-O-style relocation evidence
separate from Win64 PE/COFF IAT relocation evidence.
`linux-x86` and `linux-x32` now build canonical `ptr`/`rawptr`/`nullable_ptr`/`ref`,
`c_int`/`c_uint`, and ILP32 native/libc scalar parameter/return `@export`
object smokes with target-specific symbol metadata, including a nullable pointer
null-return object smoke and a non-nullable `ref` null-return type diagnostic.
Function-pointer spellings (`fnptr`, `fn(...) -> ...`) remain rejected until
their C ABI wrappers are verified. This keeps x32's 32-bit pointer/libc ABI, i386's 32-bit cdecl
boundary, and the compiler-owned callable slot ABI from silently passing through
unverified target assumptions.

Internal runtime exports are separate: modules `__rt` and `__rt.*` may export
reserved `__tetra_*` symbols using the compiler-owned slot ABI documented
below. That exemption does not make arbitrary user aggregate FFI valid on native
targets.

## Calling convention per target

- `linux-x64`, `macos-x64`: SysV AMD64 ABI (first args in `rdi, rsi, rdx, rcx, r8, r9`)
- `windows-x64`: Windows x64 ABI (first args in `rcx, rdx, r8, r9`, plus 32-byte shadow space)
- `linux-x86`: i386 SysV ABI for no-runtime executable/object paths (stack
  arguments, caller cleanup, scalar returns in `eax`/`edx:eax` as applicable,
  including stdout write/string-literal executable and `core.net_write(2)`
  stderr fd runtime smoke coverage plus allocator success/failure, raw memory
  bounds, raw pointer-slot, and island/free executable ABI smoke coverage)
- `linux-x32`: x32 SysV ABI for no-runtime executable/object paths (x86_64
  registers with 32-bit pointer/native-integer ABI facts, including stdout
  write/string-literal executable and `core.net_write(2)` stderr fd runtime
  smoke coverage plus allocator success/failure, raw memory bounds, raw pointer-slot, and
  island/free executable ABI smoke coverage)

## Linux native promotion matrix

`linux-x64` is the production Linux native runtime ABI baseline. `linux-x86`
and `linux-x32` remain build-only/host-probed until their own ABI surfaces have
runtime, stdlib, FFI, linker, atomic, smoke, fuzz, brutal, artifact-hash, and
runner evidence. Promotion is target-specific: x86 cannot borrow x64 ABI facts,
and x32 cannot be represented as either LP64 x64 or i386.

| Target | Runtime ABI status | Pointer/native-int facts | Promotion requirement |
| --- | --- | --- | --- |
| `linux-x64` | Supported SysV AMD64 baseline with scalar and pointer `@export` object ABI regression evidence, filesystem+scheduler regression evidence, and networking runtime smokes | 64-bit pointers, 64-bit native ints, 64-bit registers | Keep ABI/runtime/stdlib/atomic/fuzz gates passing |
| `linux-x86` | Build-only i386 slices with canonical pointer/rawptr/nullable_ptr/ref, `c_int`/`c_uint`, and ILP32 native/libc scalar `@export` object evidence, no-runtime stdout/string-literal executable coverage, `core.net_write(2)` stderr fd runtime smoke coverage, allocator success/failure, raw memory bounds, raw pointer-slot, and island/free executable ABI smoke coverage, self-host logical time-only, bounded two-spawn actors/task/task-group, single-spawn typed-task/staged typed-task/typed task-group, actor-state, filesystem composition, and current `core.net` networking syscall smokes, plus explicit stdlib, function-pointer FFI, and remaining source target-layout scalar diagnostics | 32-bit pointers, native ints, and registers | Full i386 runtime startup, syscall bridge, stdlib, function-pointer FFI, aggregate/float ABI, atomics, executable smoke, and no-host-fallback evidence |
| `linux-x32` | Build-only x32 slices with canonical pointer/rawptr/nullable_ptr/ref, `c_int`/`c_uint`, and ILP32 native/libc scalar `@export` object evidence, no-runtime stdout/string-literal executable coverage, `core.net_write(2)` stderr fd runtime smoke coverage, allocator success/failure, raw memory bounds, raw pointer-slot, and island/free executable ABI smoke coverage, self-host time, bounded two-spawn actors/task/task-group, single-spawn typed-task/staged typed-task/typed task-group, actor-state, filesystem composition, `fs_exists` filesystem smokes, and current `core.net` x32-syscall smokes, plus explicit runtime/stdlib, function-pointer FFI, and remaining source target-layout scalar diagnostics | 32-bit pointers/native ints with x86_64 registers | Full x32 runtime policy, x32 syscalls, stdlib, function-pointer FFI, atomics, executable smoke, and no-host-fallback evidence |

The release validator entrypoint is
`tools/cmd/validate-linux-native-targets`. It rejects x86/x32 production claims
while they are still `build_only`, rejects x32 metadata that collapses to
LP64 x64, rejects x86 metadata that borrows x64 ABI facts, and rejects fake,
skipped, metadata-only, docs-only, mock, placeholder, or report-only evidence.
It also requires a same-run `artifact-hashes.json` manifest, validated by
`tools/cmd/validate-artifact-hashes`, so report artifacts cannot be changed
after suite validation without tripping the Linux native promotion gate.
When evidence includes all three Linux native targets, the validator also
requires the `tetra test --all-targets --brutal --format=json` report and
checks it for the per-target ABI, atomic, and fuzz results.
Passing per-target ABI, atomic, fuzz, and runner reports must carry the
matching top-level `target` identity, so x64 evidence cannot be reused as
x86/x32 evidence; blocked x86/x32 runner evidence must instead be a
target-runtime JSON diagnostic that mentions the target, host identity, exact
probe command, and no-host-fallback reason. The validator cross-checks that
runner evidence matches the same `targets.json`: passing runner reports require
`run_supported: true`, and no-host diagnostics require `run_supported: false`.
The current stdlib/runtime capability matrix for this target family is
`docs/spec/linux_native_target_stdlib_matrix.md`.

Target layout size checks follow the target native integer model rather than
the host compiler process. Fixed arrays whose byte size cannot be represented in
the target `usize` are rejected with an explicit diagnostic; in particular,
`linux-x86` and `linux-x32` reject layouts larger than `u32::MAX` bytes while
`linux-x64` keeps the LP64 limit.

The target layout names `usize`, `isize`, `size_t`, `ssize_t`, `native_int`,
`native_uint`, `c_long`, and `c_ulong` are source-level 1-slot 32-bit scalar
aliases only for the ILP32 Linux native targets (`linux-x86` and `linux-x32`).
LP64 targets still reject those spellings until their native-integer/c_long
width semantics and codegen are implemented. Other target-layout-only scalar
spellings such as `u32`, `u64`, `f32`, and `f64` continue to receive the
explicit target-layout-only diagnostic.

All functions in this document return the first slot in `rax`/`eax`. Linux x86
uses the i386 SysV C ABI at the external scalar boundary, but Tetra's internal
slot ABI for no-runtime x86 calls supports 0 through 3 direct register return
slots: slot 1 in `eax`, slot 2 in `edx`, and slot 3 in `ecx`. Wider x86
internal returns remain explicit backend errors until a hidden return-area
protocol is implemented.

Native x64-family internal returns currently support direct internal register
returns with 0 through 10 slots: slot 1 in `rax`/`eax`, slot 2 in `rdx`/`edx`, slot 3 in `r8`/`r8d`,
slot 4 in `r9`/`r9d`, slot 5 in `r10`/`r10d`, slot 6 in `r11`/`r11d`, slot 7 in `r12`/`r12d`, slot 8 in `r13`/`r13d`, slot 9 in `r14`/`r14d`, and slot 10 in `rbx`/`ebx`. The built-in actors scheduler owns
`r15` while runtime-backed code is executing, so internal return-slot expansion
must not use `r15` as a user return register. Runtime surfaces that need 3-slot returns, such as
`actor.recv_msg_result`, use that same register order; 4-slot direct returns are supported for the current typed runtime
envelopes, 9-slot direct returns support the current eight-environment-slot `fnptr` callable payload slice, and 10-slot direct returns
support the current enum tag plus nine-slot `fnptr` callable payload slice. Wider
runtime results continue to use an explicit staged buffer protocol instead of additional return registers. The backend spills incoming
arguments into Tetra local slots in declaration order before lowering the function body.

Stack arguments begin after the register argument window:

- SysV (`linux-x64`, `macos-x64`): argument 7 is read from `[rbp+16]`, argument 8 from `[rbp+24]`, and so on.
- Win64 (`windows-x64`): argument 5 is read from `[rbp+48]` after the return address and the 32-byte shadow space.

Calls preserve the platform alignment contract. SysV calls align `rsp` to 16 bytes before `call`; Win64 calls reserve
the mandatory 32-byte shadow space and aligns around additional stack arguments. Current ABI regression tests cover calls
with 0 through 8 arguments and return layouts with 0 through 10 slots.

Unsupported ABI/runtime combinations are hard errors. `ctx_switch` has
verified object-code emission for SysV Unix x64, Win64 x64, i386 `linux-x86`,
and x32 SysV. The x86 proof includes the i386 callee-saved frame (`ebx`,
`ebp`, `esi`, `edi`) used by the narrow i386 self-host actors runtime; the x32
proof keeps the SysV x86_64 callee-saved frame and explicitly rejects Win64
shadow-space leakage. These object smokes do not by themselves promote the full
x86 or x32 runtime surface beyond the current bounded two-spawn actor/task slices.
Any ABI without a verified context-switch lowering reports `ctx_switch:
unsupported ABI`.

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

## Linux-x64 Memory Production ABI

This is the narrow memory contract for the post-`v0.4.0` Memory Production Core
line. It applies only to native `linux-x64` promotion work. It is not a
cross-target guarantee and makes no cross-target memory production claim for
WASM, macOS, or Windows.

The P5 runtime allocation contract is the cross-stage contract for allocator
work. Its executable form is
`compiler/internal/runtimeabi.RuntimeAllocationContracts`; the design-level
contract is documented in `docs/design/runtime_allocation_contract.md`. The
contract freezes allocator API names, alignment guarantees, zero-size behavior,
negative/overflow guard behavior, stable failure behavior, debug
instrumentation hooks, and allocation-report hooks. P5.1 implements the first
`linux-x64` fast heap slice path: non-empty safe `make_*` requests up to 4096
bytes route through a shared 64 KiB bump chunk helper with 16-byte size-class
rounding, while larger safe-slice requests use the helper's `mmap` fallback.
P5.2 hardens explicit islands: `core.island_new` rejects negative or too-large
payload sizes before the host allocator, `core.island_make_*` rounds every bump
allocation to 16 bytes before the capacity commit, and debug island free keeps
double-free/use-after-free instrumentation where supported. P5.3 lets
allocation reports model function-local temporary regions for non-escaping
copies, but those rows still say heap fallback in `actual_lowering_storage`
until implicit region lowering exists. P5.4 allocation reports use schema v2
and include a validated runtime summary: allocation count, planned-storage
counts, actual-lowering counts, runtime-path counts, requested/reserved bytes,
and per-region summaries for rows with region ids. Unsafe raw
`core.alloc_bytes` remains conservative because its current raw memory bounds
checks depend on allocation-header metadata.

The current compiler surface exposes these unsafe builtins:

- `core.alloc_bytes(size: i32) -> ptr`
- `core.cap_mem() -> cap.mem`
- `core.ptr_add(ptr, offset: i32, mem: cap.mem) -> ptr`
- `core.load_i32(ptr, mem: cap.mem) -> i32`
- `core.store_i32(ptr, value: i32, mem: cap.mem) -> i32`
- `core.load_u8(ptr, mem: cap.mem) -> u8`
- `core.store_u8(ptr, value: u8, mem: cap.mem) -> u8`
- `core.load_ptr(ptr, mem: cap.mem) -> ptr`
- `core.store_ptr(ptr, value: ptr, mem: cap.mem) -> ptr`
- `core.raw_slice_u8_from_parts(ptr, len: i32, mem: cap.mem) -> []u8`
- `core.raw_slice_u16_from_parts(ptr, len: i32, mem: cap.mem) -> []u16`
- `core.raw_slice_i32_from_parts(ptr, len: i32, mem: cap.mem) -> []i32`
- `core.raw_slice_bool_from_parts(ptr, len: i32, mem: cap.mem) -> []bool`

All of these operations remain gated by `unsafe` and the required `uses`
effects documented in [unsafe.md](./unsafe.md). `core.cap_mem()` grants a
capability token for raw memory operations; it is permission, not pointer
provenance or bounds proof. On `linux-x64`, raw slice construction traps before
view construction for negative lengths and target byte-length overflow. Other
targets currently have build/lower/report evidence for the same unsafe gateway
shape, but do not claim direct runtime trap parity unless their target-specific
runtime smoke says so.

Production promotion for this ABI requires allocator failure semantics to be
deterministic on `linux-x64`. A production report must state whether
`core.alloc_bytes` succeeded, failed with a stable diagnostic/status, or was
blocked by a checked precondition. Silent wraparound, target-dependent crash
behavior, and metadata-only "allocated" claims are not acceptable production
evidence.

The invalid allocation sizes are checked before the host allocation request. The
current Linux-x64 slice rejects `core.alloc_bytes` sizes less than one with exit
code `2`; a zero-size allocation is not treated as a successful allocation.
Safe `make_*` zero-length allocations remain different from
`core.alloc_bytes(0)`: they produce canonical empty slices without allocator
access where the target implements the make-slice contract.

Outside this Linux-x64 production memory contract, the build-only `linux-x86`
and `linux-x32` ABI suites now build allocator and raw memory bounds
executables that exercise `core.alloc_bytes(4)` with
`core.store_i32`/`core.load_i32`, `core.alloc_bytes(0)` for the checked
invalid-size branch, `core.ptr_add` with byte `store_u8`/`load_u8` through
the allocation-header bounds helper, and raw pointer-slot programs that emit
target-width base and direct-`ptr_add` offset `core.store_ptr`/`core.load_ptr`.
The x86 smokes
require the i386 `mmap2` syscall number, the post-`mmap` `[-4095, -1]`
error-range guard, and precondition/failure `exit(2)` through `int 0x80`; the
x32 smokes require the x32 syscall-bit `mmap` number, the same error-range
guard, and precondition/failure `exit(2)` through `syscall` while rejecting
plain x64 syscall forms. These smokes prove target-specific lowering and ELF
identity only. They do not promote x86/x32 allocator/free/panic parity or the
full Memory Production Core.

The same build-only ABI suites now build scoped island/free executables in
normal and `--islands-debug` modes. The x86 smoke requires i386 `munmap` for
normal free and `mprotect(PROT_NONE)` through `int 0x80` for debug free
guarding; the x32 smoke requires the x32 syscall-bit `munmap` and `mprotect`
numbers through `syscall` and rejects i386 or plain x64 syscall forms. This is
target-specific lowering evidence for scoped islands and debug free guards, not
a complete allocator/free/panic runtime promotion. P5.2 additionally requires
island payload-size guards before allocator entry and 16-byte aligned bump
commit for island slice allocations.

The current native SysV allocator slice checks the Linux/macOS `mmap` error
range after the syscall. Values in `[-4095, -1]` are treated as allocation
failure and terminate with exit code `2` before the pointer is returned to Tetra
code. This is allocator failure semantics evidence only; it is not a
use-after-free, aliasing, or bounds proof.

Production promotion also requires runtime bounds diagnostics for raw byte and
word access. Until those diagnostics are implemented and covered by
`tetra.memory.production.v1` evidence, the current raw load/store helpers are
only capability-gated unsafe operations. They must not be described as a
complete production memory runtime by themselves.

The current raw pointer arithmetic slice rejects negative `core.ptr_add` offsets
at runtime before the adjusted pointer is returned. The diagnostic path exits
with code `2`, matching the allocator failure/precondition failure class. This
is a lower-bound check for pointer arithmetic; upper-bound checks for arbitrary
raw pointers still require allocation metadata and remain part of the unfinished
Memory Production Core.

The current allocator metadata slice stores the requested `core.alloc_bytes`
size in a runtime header immediately before the pointer returned to Tetra code.
For allocation-base pointers, allocation-base `core.ptr_add` upper bounds reject
offsets greater than or equal to that requested size with exit code `2`. This is
an allocation-base upper-bound check for helper loops and direct
`core.load_*`/`core.store_*` calls whose address is a visible
`core.ptr_add(base, offset, mem)`. It is not yet a complete derived-pointer
provenance table or a general raw-pointer upper-bound proof.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_i32` width bounds
reject a 4-byte store when the requested allocation size is smaller than 4
bytes, also with exit code `2`. The same allocation-base helper is shared by
the current `core.load_i32` path. This is a word-access width check for
allocation-base pointers only; derived-pointer width checks still require a
complete provenance table unless the backend can see a direct base+offset raw
access.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_ptr` width bounds
reject stores whose pointer width does not fit in the requested allocation. On
`linux-x64` that is an 8-byte pointer slot; the build-only `linux-x86` and
`linux-x32` ABI suites now require executable evidence that both allocation-base
and direct `core.ptr_add(base, offset, mem)` `core.store_ptr`/`core.load_ptr`
use a 4-byte pointer slot while preserving the target syscall bridge. The same
allocation-base helper is shared by the current `core.load_ptr` path. This is
pointer-slot width evidence for allocation-base and directly visible base+offset
raw access only, not a complete derived-pointer or arbitrary-address proof.
Direct `core.load_ptr`/`core.store_ptr` calls over a visible
`core.ptr_add(base, offset, mem)` use the same allocation header with an
offset+width check; stored arbitrary derived pointers still do not carry
provenance.

The stable `lib.core.memory` helper slice treats negative `memcpy_u8` and `memset_u8` lengths
as invalid helper preconditions and returns status `2` before entering the raw
byte loop. This is helper-level status evidence, not a process trap and not a
replacement for runtime bounds diagnostics on each raw access.

The Memory Production Core line must distinguish:

- compile-time diagnostics: missing `unsafe`, missing `uses`, missing
  `cap.mem`, borrow escape, use-after-consume/transfer, invalid actor/task
  transfer, and other statically visible ownership violations;
- runtime checked failures: allocator failure semantics, bounds diagnostics,
  double-free/use-after-free checks where the runtime owns enough metadata to
  detect them deterministically;
- forbidden evidence: mock, placeholder, docs-only, metadata-only, build-only,
  or sidecar-only reports.

## WASM target ABI contracts

### `wasm32-wasi`

The WASI backend emits a deterministic WebAssembly module with:

- WASM magic `\0asm` and version 1.
- Imports from `wasi_snapshot_preview1`: `fd_write` and `proc_exit`.
- Exports: `memory` and `_start`.
- Unsupported native runtime instructions are rejected at link/codegen time with an explicit `wasm backend` diagnostic.

`tetra smoke --target wasm32-wasi --run=false` is artifact/import preflight
evidence, not runtime proof.
`tetra run --target wasm32-wasi` is runtime-aware: it requires a discovered
WASI runner (`wasmtime`, or the Node WASI fallback) and reports a missing-runner
blocker when neither is available. Runner smoke evidence is also produced by
`scripts/release/v1_0/wasi-smoke.sh`; when a runner fallback is used, the smoke
report records that runner.

### `wasm32-web`

The web backend emits a deterministic WebAssembly module plus a JavaScript loader contract:

- Imports from `tetra_web_v0.4.0`: `console_log(ptr, len)` and `panic(code, ptr, len)`.
- Exports: `memory` and `tetra_main`.
- The loader fetches the `.wasm` module relative to `import.meta.url`, wires `tetra_web_v0.4.0`, and exposes
  `instantiateTetra()` plus `runTetra()`.
- Unsupported native runtime instructions are rejected at link/codegen time with an explicit `wasm backend` diagnostic.

`tetra smoke --target wasm32-web --run=false` is artifact/import preflight
evidence, not runtime proof.
`tetra run --target wasm32-web` uses the generated loader and a discovered
Chromium-compatible browser runner; `targets --format=json` reports
`run_supported` according to browser discovery. Full browser automation evidence
is produced by `scripts/release/v1_0/web-smoke.sh` and remains host/browser
dependent.

UI runtime target metadata is intentionally separate from general target
build/run support. `ui_runtime_status` is `production` for Linux-x64 native UI
runtime and wasm32-web browser UI runtime evidence, `requires_target_host_evidence`
for Windows/macOS until real target-host `tetra.ui.platform.v1` reports exist,
and `unsupported` for WASI/build-only targets that do not provide UI event
dispatch runtime behavior.

General Linux native target promotion metadata is separate again:
`runtime_status`, `stdlib_status`, `ffi_status`, `runner_probe_command`,
`release_gate`, and `evidence_artifacts` describe the linux-x64/x86/x32
runtime/stdlib/FFI gate state validated by the Linux native smoke reports. The
current x86/x32 values remain partial build-only evidence, not production
runtime ABI support.

Linux native target metadata also records the canonical syscall pack. `linux-x64`
uses the x86_64 `syscall` instruction and x86_64 syscall numbering with
`rax,rdi,rsi,rdx,r10,r8,r9`. `linux-x86` uses `int 0x80`, i386 syscall
numbering, and `eax,ebx,ecx,edx,esi,edi,ebp`. `linux-x32` uses the x86_64
`syscall` instruction and register pack with x32 syscall-bit numbering. All
three Linux packs use the Linux negative errno range `[-4095, -1]`.

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

Typed task joins in the current MVP are emitted for slot counts `2..8` on the
builtin runtime path: direct ABI returns for `2..4`
(`__tetra_task_join_typed_2..4`) and staged runtime-buffer joins for `5..8`
(`__tetra_task_join_typed_5..8` plus `__tetra_task_result_get`). One-slot typed
handles reuse the existing `task.i32` join path, and typed layouts above `8` are
rejected during semantic checking. Worker targets remain zero-argument
synchronous `i32` functions; for `2..4` they must throw the typed error enum,
and for staged `5..8` they may be either non-throwing or throw the same typed
error enum. The generic self-host runtime selector still rejects typed task
handles, but target-specific build-only slices may override that when their
self-host runtime exports prove the exact envelope: currently x86 and x32 cover
`2..8`, with staged runtime-buffer joins for wider typed errors.

The builtin x64 runtimes emit wrappers only for the supported `2..8` envelope.
Tests cover both SysV and Win64 typed-join wrapper bounds, including rejection
of slot counts below `2` and above `8`.

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

The runtime smoke boundary is native-host execution for `linux-x64` and
build-only evidence for non-host targets unless a platform runner is explicitly
available. The Linux native release runner report, when it is a passing
execution report rather than a no-host-fallback diagnostic, must include
arithmetic, allocator/raw-memory, filesystem, stderr fd, time, network socket
open/close, network options, and task-join smoke results for the target. The ABI/report path keeps canonical pointer `@export` object smokes
for `linux-x86`, `linux-x64`, and `linux-x32`; `linux-x64` also keeps explicit
filesystem+scheduler composition and scheduler-restriction regression smokes
so build-only target restrictions cannot become production Linux behavior.
`linux-x86` has a narrow self-host logical time runtime for
time-only programs, validated as an ELF32 `EM_386` smoke and run only when the
host can execute i386 binaries. When a program uses the supported actor/task
surface, `linux-x86` can also build and run the i386 self-host runtime for two
spawned actors/tasks/task-group workers, actor-state method, `task.i32`, typed-task handles through the
8-slot staged envelope, or typed task-group composition, including filesystem+scheduler
composition; `linux-x32` has matching
build-only ABI-report smokes for no-runtime stdout/string-literal executables,
self-host time, two spawned actors/tasks/task-group workers, actor-state
method, `task.i32`, typed-task handles through the 8-slot staged envelope, typed task-group composition, and filesystem+scheduler composition. A pure filesystem existence
probe can build and run through minimal target-specific `__tetra_fs_exists`
runtime objects on `linux-x86` and `linux-x32`. The x86 and x32 scheduler slices
can compose that filesystem symbol with their self-host schedulers. The same ABI
path also builds allocator executables for `core.alloc_bytes` success plus raw
`store_i32`/`load_i32`, checked invalid-size and post-`mmap` error exit
lowering, raw memory bounds executables for `ptr_add` plus byte store/load, raw
pointer-slot base/offset executables for `store_ptr`/`load_ptr`, and scoped
island/free executables using the target syscall bridge.
These slices do not
include actor fanout above 2, surface, distributed actors, full allocator/free
parity, panic, or broad syscall/stdlib parity, so those surfaces
remain target-aware diagnostics on `linux-x86` and `linux-x32`. The ABI suite
records Surface and distributed-actor diagnostics as explicit x86/x32 evidence
so those unsupported surfaces cannot be promoted by omission. x86/x32 staged
typed tasks and typed task-group composition remain build-only/host-probed evidence.
Distributed actors, networked mailboxes, and multi-process actor placement are
post-v1 and not part of this ABI.

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

## Surface runtime ABI

Tetra Surface programs call the host only through the tiny Surface runtime ABI.
The compiler selects a runtime object when a checked program uses `surface`
builtins, validates that the object exports every required Surface symbol, and
checks exported slot-count metadata when it is present. This starter slice is
not a production UI claim: the current linux-x64 built-in host exports
deterministic stub functions so pure Tetra Surface examples can link and run
while real headless, Linux window, and wasm Surface hosts remain gated by
separate evidence and validators.

String and slice values are scalarized at the slot boundary. `String` lowers to
`ptr,len`; `[]u8` lowers to `ptr,len`. Checked String byte views reuse the
slice-view guard shape with byte-width pointer adjustment: the range checks run
before the returned `ptr,len` header is constructed.

### `__tetra_surface_open(title_ptr: ptr, title_len: i32, width: i32, height: i32) -> i32`

Opens or creates a host surface and returns a positive handle on success. The
starter linux-x64 built-in runtime returns a kernel-backed handle.

### `__tetra_surface_close(surface_handle: i32) -> i32`

Closes a Surface handle and returns `0` on the deterministic starter host.

### `__tetra_surface_poll_event_kind(surface_handle: i32) -> i32`

Returns the next event kind as an integer. This is a scalar compatibility
helper; `__tetra_surface_poll_event_into` is the preferred starter event-record
path.

### `__tetra_surface_poll_event_x(surface_handle: i32) -> i32`

Returns the current pointer event x coordinate for the starter scalar event
ABI. The deterministic starter hosts report `48` for the scripted click.

### `__tetra_surface_poll_event_y(surface_handle: i32) -> i32`

Returns the current pointer event y coordinate for the starter scalar event
ABI. The deterministic starter hosts report `96` for the scripted click.

### `__tetra_surface_poll_event_button(surface_handle: i32) -> i32`

Returns the current pointer button for the starter scalar event ABI. The
deterministic starter hosts report `1` for the scripted click.

### `__tetra_surface_poll_event_into(surface_handle: i32, event_ptr: ptr, event_len: i32) -> i32`

Copies the current event record into caller-owned Tetra `[]i32` memory and
returns the number of slots copied. The starter record has nine slots:
`kind`, `x`, `y`, `button`, `key`, `width`, `height`, `timestamp_ms`, and
`text_len`. Deterministic starter hosts copy `[5, 48, 96, 1, 0, 320, 200, 0,
0]` for the scripted pointer event. The host must not retain the pointer after
this call.

### `__tetra_surface_poll_event_text_len(surface_handle: i32) -> i32`

Returns the byte length of the current text-input payload. The deterministic
starter hosts report `2` for the scripted `"OK"` payload.

### `__tetra_surface_poll_event_text_into(surface_handle: i32, text_ptr: ptr, text_len: i32) -> i32`

Copies the current text-input payload into caller-owned Tetra memory and
returns the number of bytes copied. The starter hosts copy `OK` when the buffer
is at least two bytes. The host must not retain the pointer after this call.

### `__tetra_surface_clipboard_write_text(surface_handle: i32, text_ptr: ptr, text_len: i32) -> i32`

Copies caller-owned UTF-8 bytes from Tetra memory into the host clipboard
boundary and returns the number of bytes accepted. Safe code must pass owned or
copied storage; borrowed slice/String views are rejected before crossing this
host boundary. The host must not retain `text_ptr` after the call.

### `__tetra_surface_clipboard_read_text_into(surface_handle: i32, text_ptr: ptr, text_len: i32) -> i32`

Copies host clipboard text into caller-owned Tetra `[]u8` memory and returns
the number of bytes copied. The host may truncate to the destination capacity,
and it must not retain `text_ptr` after the call.

### `__tetra_surface_poll_composition_into(surface_handle: i32, event_ptr: ptr, event_len: i32) -> i32`

Copies the current deterministic composition trace into caller-owned Tetra
`[]i32` memory and returns the number of slots copied. The Surface v1 release
baseline uses four boolean-like slots for `start`, `update`, `commit`, and
`cancel`; missing slots must not be accepted as production IME/composition
evidence. The host must not retain `event_ptr` after the call.

### `__tetra_surface_begin_frame(surface_handle: i32) -> i32`

Begins a frame on the host. The current pure Tetra helper allocates the RGBA
framebuffer in Tetra-owned memory after this call returns.

### `__tetra_surface_present_rgba(surface_handle: i32, pixels_ptr: ptr, pixels_len: i32, width: i32, height: i32, stride: i32) -> i32`

Presents a Tetra-owned RGBA framebuffer to the host. Runtime hosts must treat
the framebuffer as valid only for the call and must not retain the pointer
after presentation without a future explicit ownership protocol.

### `__tetra_surface_now_ms() -> i32`

Returns the Surface host time in milliseconds. The starter linux-x64 built-in
runtime returns deterministic `0`.

### `__tetra_surface_request_redraw(surface_handle: i32) -> i32`

Requests another frame and returns `0` on the deterministic starter host.

The compiler validates these Surface runtime exports when a program uses
Surface host builtins:

- `__tetra_surface_open`
- `__tetra_surface_close`
- `__tetra_surface_poll_event_kind`
- `__tetra_surface_poll_event_x`
- `__tetra_surface_poll_event_y`
- `__tetra_surface_poll_event_button`
- `__tetra_surface_poll_event_into`
- `__tetra_surface_poll_event_text_len`
- `__tetra_surface_poll_event_text_into`
- `__tetra_surface_clipboard_write_text`
- `__tetra_surface_clipboard_read_text_into`
- `__tetra_surface_poll_composition_into`
- `__tetra_surface_begin_frame`
- `__tetra_surface_present_rgba`
- `__tetra_surface_now_ms`
- `__tetra_surface_request_redraw`

## Filesystem runtime ABI

Filesystem host builtins use explicit `ptr,len` strings; runtime exports must
not treat path arguments as NUL-terminated input.

### `__tetra_fs_exists(path_ptr: ptr, path_len: i32, io_cap: cap.io) -> bool`

Returns `1` when the host path exists and `0` when it does not exist. Invalid
or unsupported paths return `0`. The third slot is the `cap.io` token required
by the semantic builtin and is reserved for future runtime-side capability
validation.

The compiler validates these filesystem runtime exports when a program uses
filesystem host builtins:

- `__tetra_fs_exists`

`linux-x86` and `linux-x32` can append their target-specific filesystem objects
to the self-host scheduler runtime for ABI-report filesystem+scheduler
composition smokes. The Linux native ABI suites keep pointer plus `c_int`/`c_uint`
`@export` FFI object smokes, and the linux-x64 ABI suite keeps the
filesystem+scheduler composition path as a regression smoke.

## Networking runtime ABI

The current networking runtime ABI is a Linux TCP socket client/server I/O
slice with one-event epoll readiness helpers. `linux-x64`, `linux-x86`, and
`linux-x32` cover socket, bind/connect/listen/accept4, read/recv/write/send,
epoll create/control/wait, nonblocking, `SO_REUSEPORT`, `TCP_NODELAY`, and
close. The i386 bridge uses `socketcall` where Linux i386 requires it plus
direct `read`, `write`, `fcntl`, `epoll_ctl`, `epoll_wait`, `epoll_create1`,
and `close` `int 0x80` syscalls. The x32 bridge uses x32 syscall-bit numbers,
including the x32-specific `recvfrom`, `setsockopt`, and epoll entries rather
than plain x64 syscall numbers. The slice is
intentionally smaller than a production HTTP server transport: full event-loop
abstractions, io_uring, per-core workers, socket options beyond the listed
helpers, DNS, TLS, and PostgreSQL/database protocols remain outside this ABI
slice.

### `__tetra_net_socket_tcp4(io_cap: cap.io) -> i32`

Opens an IPv4 TCP stream socket and returns the fd, or the negative Linux
syscall result on failure. The `cap.io` slot is required by the semantic
builtin and reserved for future runtime-side capability validation.

### `__tetra_net_bind_tcp4_loopback(fd: i32, port: i32, io_cap: cap.io) -> i32`

Binds `fd` to `127.0.0.1:port` using a runtime-constructed `sockaddr_in` and
returns the Linux `bind` syscall status. Passing `0` asks the kernel to choose
an ephemeral port.

### `__tetra_net_connect_tcp4_loopback(fd: i32, port: i32, io_cap: cap.io) -> i32`

Connects caller-owned `fd` to `127.0.0.1:port` using a runtime-constructed
`sockaddr_in` and returns the Linux `connect` syscall status.

### `__tetra_net_listen(fd: i32, backlog: i32, io_cap: cap.io) -> i32`

Calls Linux `listen(fd, backlog)` and returns the syscall status.

### `__tetra_net_accept4(fd: i32, flags: i32, io_cap: cap.io) -> i32`

Calls Linux `accept4(fd, NULL, NULL, flags)` and returns the accepted fd or the
negative syscall result.

### `__tetra_net_read(fd: i32, dst_ptr: ptr, dst_len: i32, start: i32, count: i32, io_cap: cap.io) -> i32`

Reads from `fd` into `dst_ptr + start`, after rejecting negative `start` or
`count` and rejecting `start > dst_len`. The runtime clamps `count` to the
remaining slice length and returns the Linux `read` syscall result.

### `__tetra_net_recv(fd: i32, dst_ptr: ptr, dst_len: i32, start: i32, count: i32, io_cap: cap.io) -> i32`

Receives from `fd` into `dst_ptr + start` via Linux `recvfrom` with flags `0`
and `NULL` address operands, after rejecting negative `start` or `count` and
rejecting `start > dst_len`. The runtime clamps `count` to the remaining slice
length and returns the Linux syscall result.

### `__tetra_net_write(fd: i32, src_ptr: ptr, src_len: i32, start: i32, count: i32, io_cap: cap.io) -> i32`

Writes to `fd` from `src_ptr + start`, after rejecting negative `start` or
`count` and rejecting `start > src_len`. The runtime clamps `count` to the
remaining slice length and returns the Linux `write` syscall result.

### `__tetra_net_send(fd: i32, src_ptr: ptr, src_len: i32, start: i32, count: i32, io_cap: cap.io) -> i32`

Sends from `fd` using `src_ptr + start` via Linux `sendto` with flags `0` and
`NULL` address operands, after rejecting negative `start` or `count` and
rejecting `start > src_len`. The runtime clamps `count` to the remaining slice
length and returns the Linux syscall result.

### `__tetra_net_epoll_create(io_cap: cap.io) -> i32`

Calls `epoll_create1(0)` and returns the epoll fd or the negative syscall
result.

### `__tetra_net_epoll_ctl_add_read(epfd: i32, fd: i32, io_cap: cap.io) -> i32`

Registers `fd` with `epfd` for `EPOLLIN` readiness using `event.data.u64 = fd`
and returns the Linux `epoll_ctl` syscall status.

### `__tetra_net_epoll_ctl_add_read_write(epfd: i32, fd: i32, io_cap: cap.io) -> i32`

Registers `fd` with `epfd` for `EPOLLIN | EPOLLOUT` readiness using
`event.data.u64 = fd` and returns the Linux `epoll_ctl` syscall status.

### `__tetra_net_epoll_ctl_mod_read(epfd: i32, fd: i32, io_cap: cap.io) -> i32`

Modifies an existing epoll registration to `EPOLLIN` readiness using
`event.data.u64 = fd` and returns the Linux `epoll_ctl` syscall status.

### `__tetra_net_epoll_ctl_mod_read_write(epfd: i32, fd: i32, io_cap: cap.io) -> i32`

Modifies an existing epoll registration to `EPOLLIN | EPOLLOUT` readiness using
`event.data.u64 = fd` and returns the Linux `epoll_ctl` syscall status.

### `__tetra_net_epoll_ctl_delete(epfd: i32, fd: i32, io_cap: cap.io) -> i32`

Removes `fd` from `epfd` and returns the Linux `epoll_ctl` syscall status.

### `__tetra_net_epoll_wait_one(epfd: i32, timeout_ms: i32, io_cap: cap.io) -> i32`

Calls `epoll_wait` for one event. It returns the ready fd from event data when
one event is available, `0` on timeout, or the negative syscall result.

### `__tetra_net_epoll_wait_one_into(epfd: i32, event_ptr: ptr, event_len: i32, timeout_ms: i32, io_cap: cap.io) -> i32`

Calls `epoll_wait` for one event after requiring `event_len >= 2`. When one
event is available, it writes the ready fd to `event[0]`, writes the Linux
`epoll_event.events` flag word to `event[1]`, and returns `1`. It returns `0`
on timeout or the negative syscall result on error.

On `linux-x86`, epoll helpers use i386 syscall numbers `255` (`epoll_ctl`),
`256` (`epoll_wait`), and `329` (`epoll_create1`). On `linux-x32`, they use
x32 syscall-bit numbers `0x400000e9`, `0x400000e8`, and `0x40000123`.

### `__tetra_net_set_nonblocking(fd: i32, io_cap: cap.io) -> i32`

Reads the current fd flags with `fcntl(F_GETFL)`, sets `O_NONBLOCK` with
`fcntl(F_SETFL)`, and returns the syscall status. Negative syscall results are
returned unchanged.

### `__tetra_net_set_reuseport(fd: i32, io_cap: cap.io) -> i32`

Enables `SO_REUSEPORT` with `setsockopt(fd, SOL_SOCKET, SO_REUSEPORT, &one, 4)`
and returns the Linux syscall status.

### `__tetra_net_set_tcp_nodelay(fd: i32, io_cap: cap.io) -> i32`

Enables `TCP_NODELAY` with `setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &one, 4)`
and returns the Linux syscall status.

### `__tetra_net_close(fd: i32, io_cap: cap.io) -> i32`

Closes a caller-owned fd and returns the Linux `close` syscall status. On
`linux-x86`, this is emitted as an i386 `int 0x80` close syscall that preserves
callee-saved `ebx`. On `linux-x32`, this is emitted as the x32 syscall-bit
`close` number with the x86_64 `syscall` instruction.

The compiler validates these networking runtime exports when a program uses
networking host builtins:

- `__tetra_net_socket_tcp4`
- `__tetra_net_bind_tcp4_loopback`
- `__tetra_net_connect_tcp4_loopback`
- `__tetra_net_listen`
- `__tetra_net_accept4`
- `__tetra_net_read`
- `__tetra_net_recv`
- `__tetra_net_write`
- `__tetra_net_send`
- `__tetra_net_epoll_create`
- `__tetra_net_epoll_ctl_add_read`
- `__tetra_net_epoll_ctl_add_read_write`
- `__tetra_net_epoll_ctl_mod_read`
- `__tetra_net_epoll_ctl_mod_read_write`
- `__tetra_net_epoll_ctl_delete`
- `__tetra_net_epoll_wait_one`
- `__tetra_net_epoll_wait_one_into`
- `__tetra_net_set_nonblocking`
- `__tetra_net_set_reuseport`
- `__tetra_net_set_tcp_nodelay`
- `__tetra_net_close`

For `linux-x86` and `linux-x32`, all listed networking symbols are currently
target-supported by their target-specific runtime objects. The compiler still
validates only the used symbol subset for runtime override objects.

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
`core.sleep_until`, or `core.deadline_ms`, and filesystem runtime symbols when
the program calls `core.fs_exists`, and networking runtime symbols when the
program calls `core.net_socket_tcp4`, `core.net_bind_tcp4_loopback`,
`core.net_connect_tcp4_loopback`, `core.net_listen`, `core.net_accept4`, `core.net_read`, `core.net_recv`, `core.net_write`, `core.net_send`,
`core.net_epoll_create`, `core.net_epoll_ctl_add_read`,
`core.net_epoll_ctl_add_read_write`, `core.net_epoll_ctl_mod_read`,
`core.net_epoll_ctl_mod_read_write`, `core.net_epoll_ctl_delete`,
`core.net_epoll_wait_one`, `core.net_epoll_wait_one_into`,
`core.net_set_nonblocking`,
`core.net_set_reuseport`, `core.net_set_tcp_nodelay`, or `core.net_close`.
For `linux-x86` and `linux-x32`, the currently listed `core.net` builtins are
target-supported, and the target capability table requires the full current
networking symbol set when any of those builtins are used.
The compiler rejects missing targets,
target mismatches, missing runtime exports, and runtime export signature
metadata whose slot counts do not match the ABI before platform linking.
Runtime objects without per-symbol signature metadata remain name-validated for
compatibility with earlier TOBJ producers.

`--runtime=auto` currently selects the embedded self-host runtime for
mailbox-only actors and for the bounded two-spawn actor/task/task-group slices on
build-only targets that do not have a builtin runtime. On `linux-x86` and
`linux-x32`, a pure `core.fs_exists` or target-supported `core.net`
networking program uses the minimal target-specific filesystem/networking
runtime object.
On `linux-x86`,
`--runtime=auto` and explicit
`--runtime=selfhost` can also use the i386 self-host runtime for supported
typed-task and typed task-group envelopes, including staged typed-task joins.
On `linux-x32`, the same modes can use the
SysV self-host runtime for supported typed-task and typed
task-group envelopes; both targets can append the target-supported networking
object to those supported self-host runtime compositions. Other
targets switch to the built-in runtime
when task groups, typed-task, networking, surface, or distributed actor
builtins require a broader supported runtime surface. `--runtime=builtin` keeps
the Go-emitted runtime available as a compatibility fallback on targets with a
builtin runtime and remains an explicit diagnostic on linux-x32.

Linux x86 and x32 are stricter because builtin runtimes are not yet available
for those ABIs. `--runtime=auto` may use the i386 or SysV self-host runtime for
surfaces that fit each contract, including up to two spawned actors/tasks/task-group workers,
target-supported typed-task handles, x86/x32 typed task-group
composition, current `core.net` networking, and actor state where the target capability
table permits it.
Actor/task programs with fanout above 2 fail before runtime selection with target-aware
diagnostics (`actor fanout above 2 runtime not supported on linux-x86`/`linux-x32`)
instead of falling through to a generic builtin-runtime error.

Native execution is only supported when `host == target`; cross-target builds are build-verified but not run on
non-matching hosts.

## ABI compatibility policy

The v1 runtime ABI is source-stable for the reserved symbols listed in this
document and metadata-stable for TOBJ files that declare one of the supported
native triples. A compatible runtime object must:

- use the target's platform calling convention exactly as listed above;
- export all required actor, actor-state, task, task-group, and typed-task
  runtime symbols, plus any used time runtime symbols, with the reserved
  `__tetra_` prefix;
- set `target` to the final program target;
- avoid redefining program glue symbols or user symbols from linked libraries;
- preserve the meaning of actor handles, `i32` message values, and tagged
  `actor.msg` / `actor.recv_result_i32` two-slot returns.

The compiler rejects runtime objects with missing targets, target mismatches, or
missing required symbols before platform linking. When TOBJ symbol metadata
declares runtime export parameter and return slot counts, the compiler also
rejects ABI signature mismatches before platform linking. It also
build-verifies runtime override objects for `linux-x64`, `macos-x64`, and
`windows-x64`; real execution evidence is only claimed on matching hosts.

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
- `symbols`: exported or internal symbol names with code/data offsets, and
  optional per-function ABI slot metadata (`param_slots`, `return_slots`) for
  producers that can provide it.
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
