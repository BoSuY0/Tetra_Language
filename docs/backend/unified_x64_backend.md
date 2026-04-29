# Unified x64 Backend (Linux / macOS / Windows)

This repository uses an “hourglass” x64 backend: CPU-only logic is implemented once, while OS/ABI differences are
constrained to a small adapter layer.

Status: the unified backend refactor described below is **implemented** (see `x64core`, `x64abi`, `x64obj`), and the
platform backends are now thin wrappers.

## What is OS-specific vs CPU-specific

### CPU-only (shared x86_64 logic)

These IR instructions are pure CPU logic and can be shared across Linux/macOS/Windows:

- locals: `IRLoadLocal`, `IRStoreLocal`
- stack-machine arithmetic: `IRAddI32`, `IRSubI32`, `IRNegI32`, `IRCmpEqI32`, `IRCmpLtI32`
- control flow: `IRLabel`, `IRJmp`, `IRJmpIfZero`, `IRReturn`
- calls: `IRCall` (argument packing, stack alignment rules are ABI-dependent, but the IR-level behavior is shared)
- constants/literals: `IRConstI32`, `IRStrLit` (data placement differs by platform/linker)
- slices/indexing: `IRMakeSlice*`, `IRIndexLoad*`, `IRIndexStore*`
- Islands model: most checks/manipulation is shared; the OS interactions are not
- capabilities/mmio: semantics are shared (backend emits load/store)

### OS/ABI-specific

These parts depend on either the calling convention or OS services:

- ABI register order + stack rules:
  - SysV (Linux/macOS): args in `RDI, RSI, RDX, RCX, R8, R9`
  - Win64: args in `RCX, RDX, R8, R9`, plus 32-byte shadow space
- process services:
  - exit: Linux `sys_exit=60`, macOS `sys_exit=0x2000001`, Windows `ExitProcess` import
  - write stdout:
    - Linux/macOS `sys_write` (Linux `1`, macOS `0x2000004`)
    - Windows `GetStdHandle` + `WriteFile` imports
- memory mapping used by IR/runtime features:
  - Linux: `mmap=9`, `mprotect=10`, `munmap=11`
  - macOS: `mmap=0x20000C5`, `mprotect=0x200000A`, `munmap=0x2000049`
  - Windows: `VirtualAlloc`, `VirtualFree`, `VirtualProtect` imports

## Historical duplication points (resolved)

### TOBJ object builder is duplicated

Each platform backend used to repeat the pattern:

- emit all funcs into one `x64.Emitter` buffer
- collect:
  - `callPatches` (internal calls patched locally, external become `RelocCallRel32`)
  - `leaPatches` + `dataBlobs` for string data addressing
  - (Windows) IAT patches for imports → `RelocIATDisp32`
- build `tobj.Object{Code, Data, Symbols, Relocs}`

### Data relocation strategy differs by platform linker

- Linux, macOS and Windows linkers all have distinct data sections and support `RelocDataDisp32` to patch
  RIP-relative `lea` to point into `.data` / cstring / rdata as appropriate.

## Refactor direction (“hourglass”)

Implemented:
- Shared TOBJ builder: `compiler/internal/backend/x64obj` (data blobs + call/import patches + relocs).
- Shared IR emission switch: `compiler/internal/backend/x64core` (single IR → x64 switch).
- ABI/OS services layer: `compiler/internal/backend/x64abi` (SysV Unix vs Win64).

Platform backends are now thin:
- `linux_x64`: ELF executable path + TOBJ object path use `x64core.NewEmitFunc(x64abi.LinuxSysV())`.
- `macos_x64`: TOBJ object path uses `x64core.NewEmitFunc(x64abi.MacSysV())`.
- `windows_x64`: TOBJ object path uses `x64core.NewEmitFunc(x64abi.NewWin64())` with import collection enabled.

## Native x64 verification matrix (v0.2.0 / Epic 09)

### Object-level contracts

- shared emitter/x64 core emits deterministic rel32 patch points and validates rel32 bounds.
- TOBJ writer/reader preserves `target/module/code/data/symbols/relocs` and rejects invalid magic/version/header strings.
- x64 object builder keeps deterministic symbol ordering and emits only valid reloc kinds:
  - `RelocCallRel32` for unresolved calls,
  - `RelocDataDisp32` for literal/data LEA fixups,
  - `RelocIATDisp32` only on Windows import paths.

### ABI edge cases

- SysV and Win64 argument spilling is validated for 0..10 parameters.
- Call lowering validates stack/register argument boundaries (including stack-argument cases above register windows).
- Return-slot layouts (`0/1/2`) are validated on both ABIs.
- Shared `ctx_switch` lowering has explicit SysV/Win64 emission checks.

### Mismatch diagnostics

- shared emission fails early on missing ABI/emitter buffers and unsupported return-slot layouts.
- ABI helpers fail with explicit diagnostics for missing context pointers and stack underflow.
- native linkers reject cross-target objects with a clear `linker target mismatch` diagnostic before format writing.

### Build-only target expectations

- native build smoke is verified for `linux-x64`, `macos-x64`, `windows-x64` on object/executable paths without requiring
  cross-host execution.
- execution evidence remains host-bound (`host == target`); non-host targets are build-verified only.

## Remaining OS-specific areas

- **ABI/OS services**: syscalls vs imports, stack alignment, and calling conventions are handled in `x64abi`.
- **Executable format writers**: ELF/Mach-O/PE are format-specific by nature (they sit on top of `linkcore`).
- **Runtime objects**: some runtime components (for example actors) are currently emitted as TOBJ objects and still have
  low-level, ABI-aware code paths that should stay thin and well-documented.

## Notes on future architectures (e.g. ARM64)

This “hourglass” split makes it much easier to share *OS-agnostic* compiler logic across targets, but adding a new CPU
architecture is still substantial work:

- a new IR → ISA emitter (analogous to `x64core`)
- an ABI/OS services adapter for that ISA (analogous to `x64abi`)
- relocation rules and tests for the new ISA
- (optionally) object builder/writer abstractions if the ISA has different patch/reloc needs

In other words: the codebase is architecturally prepared for additional architectures, but they are not “trivial”.
