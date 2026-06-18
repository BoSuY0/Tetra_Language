# x86/x64/x32 Target + Layout Implementation Plan

Date: 2026-05-20

Goal: implement production-grade x86, x64, and x32 support without collapsing x32
into either x86 or x64. This plan tracks the implementation slices needed before
the targets can be promoted from honest metadata to runnable production backends.

## Current Evidence

- `compiler/target` only has runnable x64 native targets and wasm32 targets.
- The x64 native backend assumes 64-bit pointer-sized slots in several places.
- The semantic and IR layers do not yet expose a full byte-level target layout,
  x32 pointer/register separation, or atomic operation surface.
- x86 still has no full runtime/stdlib/FFI path. x32 has TOBJ object codegen,
  ELF/linker primitives, no-runtime executable build output, self-host runtime
  build output, compiler-owned target suites, and host-probed source run/test
  execution when the Linux kernel supports the x32 ABI; full
  runtime/stdlib/FFI and builtin-runtime support remain blocked until they are
  fully wired and verified. Non-runtime `@export` aggregate signatures on native
  targets now fail with target-aware diagnostics instead of pretending the C ABI
  path is implemented.

## Slice 1: Canonical Target Model And Data Layout

- Add canonical triples and aliases:
  - x86: `linux-x86`, aliases `x86`, `i386`, `i686`, `linux-i386`,
    `linux-i686`, `i386-linux-gnu`, `i686-linux-gnu`,
    `i686-unknown-linux-gnu`, `i686-pc-linux-gnu`.
  - x64: existing `linux-x64`, plus aliases `x64`, `amd64`, `x86_64`,
    `linux-amd64`, `linux-x86_64`, `x86_64-linux-gnu`,
    `x86_64-unknown-linux-gnu`, `x86_64-pc-linux-gnu`, `amd64-linux-gnu`;
    OS-specific x64 aliases route to the OS ABI, including
    `x86_64-pc-windows-msvc`/`x86_64-pc-windows-gnu` -> `windows-x64`
    and `x86_64-apple-darwin` -> `macos-x64`.
  - x32: `linux-x32`, aliases `x32`, `x86_64-x32`, `linux-x86_64-x32`,
    `x86_64-linux-gnux32`, `x86_64-unknown-linux-gnux32`,
    `x86_64-pc-linux-gnux32`, `linux-x86_64-gnux32`.
- Track CPU arch, ABI, data model, pointer width, register width, native int
  width, endian, stack alignment, object format, and explicit unsupported
  reason.
- Add scalar and aggregate layout helpers that prove pointer width is not
  inferred from register width.
- Keep x86/x32 as `build_only` with `run_mode=host_probed`: executable
  run/test is allowed only when the current Linux host can execute the exact
  ABI, while broader backend, runtime, linker, FFI, and test limitations remain
  explicit in `unsupported_reason`.

## Slice 2: CLI And Manifest Diagnostics

- Surface x86/x32 as build-only targets with explicit runtime/backend missing
  diagnostics.
- Prevent host fallback for x86/x32 and every alias.
- Extend target JSON so tooling can inspect pointer/register/data-model facts.

## Slice 3: ABI And Codegen

- Add ABI classifiers for i386 SysV, SysV AMD64, Microsoft x64, and x32 SysV.
- Split machine register size from pointer/native integer size in IR lowering.
- Add x86 codegen or a dedicated rejection boundary until implemented.
- Add x32 x86_64 codegen support with 32-bit pointer operations, explicit
  extension/truncation, and x32 reloc/runtime rules.

## Slice 4: Runtime, FFI, Atomics, Tests

- Add per-target runtime startup, allocator, panic, libc/syscall bridge, and FFI
  contracts or explicit diagnostics.
- Implement atomic widths and memory orders per target, including pointer-sized
  atomics and unsupported-width/alignment diagnostics.
- Add the requested unit, golden, runtime, ABI torture, x32 edge, fuzz/property,
  differential, negative, and brutal stress tests.

## Promotion Rule

x86 or x32 can move from `build_only` to `supported` only after the required
commands for that target pass with real execution or a documented target runner:

- `test --target x86`
- `test --target x64`
- `test --target x32`
- `test --all-targets --brutal`
- `test --target x32 --abi`
- `test --target x32 --atomic-stress`
- `test --target x32 --fuzz`

## Next RED Batch

Add failing tests before the next implementation slice:

- [x] `compiler/target`: atomic width/alignment policy for 8/16/32/64-bit and
  pointer-sized atomics on x86, x64, and x32, with unsupported-width and
  misalignment diagnostics. Generic `linux-x86` currently advertises lock-free
  8/16/32-bit atomics only; 64-bit x86 atomics stay explicitly unsupported until
  there is a CPU-feature model for `cmpxchg8b`/i686-style guarantees.
- [x] `compiler/internal/backend/x64abi`: x32 SysV ABI classifier proving x32 is not
  SysV AMD64 and not i386 SysV for pointer args, aggregate returns, varargs, and
  register extension.
- [x] `compiler/internal/backend/x64core`: pointer load/store/cast tests that fail
  anywhere a slot, pointer, or `usize` is assumed to be 64-bit because the ISA
  register is 64-bit.
- [x] `cli/cmd/tetra`: target-suite harness flags for the requested brutal,
  ABI, atomic-stress, and fuzz entrypoints. x86, Linux x64, macOS x64,
  Windows x64, and x32 now have real per-target ABI classifier suites; the
  same target set has atomic stress suites and deterministic
  layout/object-signature property suites. `--all-targets` now runs the real
  ABI matrix, and `--all-targets --brutal` runs real ABI/atomic/fuzz checks
  across the expanded x64-by-OS matrix.
- [x] `compiler/target`: compound byte layouts for arrays, slice/string views,
  enum payload storage, nested structs, and packed structs across x86/x64/x32.

## 2026-05-21 Progress

- Target metadata now exposes atomic fixed widths, pointer-sized atomic width,
  alignment checks, and load/store/fetch/fence memory-order validation.
- Linux native target metadata also exposes promotion-gate fields
  (`runtime_status`, `stdlib_status`, `ffi_status`, `runner_probe_command`,
  `release_gate`, and `evidence_artifacts`) so x86/x32 partial build-only
  runtime/stdlib/FFI evidence is machine-checkable and cannot be mistaken for
  production support.
- The same metadata now exposes the canonical Linux syscall pack so x64,
  x86, and x32 cannot silently share a syscall instruction or numbering model:
  x64 uses x86_64 `syscall`, x86 uses i386 `int 0x80`, and x32 uses x86_64
  registers with x32 syscall-bit numbering.
- The x64 ABI package now has a target-driven classifier for SysV AMD64,
  Microsoft x64, and x32 SysV. x32 uses AMD64 registers while preserving 32-bit
  pointer/usize slot facts and zero/sign extension metadata for narrower
  integer-like arguments.
- The ABI classifier now uses target byte layouts for aggregate params and
  returns. Covered cases include x32 one-register ILP32 aggregates, LP64 vs x32
  register-count differences for the same source fields, mixed SSE/integer
  eightbytes, stack classification, and indirect large aggregate returns.
- A dedicated i386 SysV classifier now models x86 as its own ABI instead of a
  x64/x32 variant: caller stack cleanup, stack-only arguments, `eax` and
  `edx:eax` scalar returns, x87 `st0` float returns, aggregate stack copies, and
  hidden stack `sret` pointers.
- Variadic call metadata is now represented in the ABI classifiers: SysV/x32
  report the `%al` SSE-register upper bound, Win64 reports 32-byte shadow space
  plus floating-point GP-register mirrors, and i386 reports caller-cleaned
  stack-only varargs with no register save area.
- The x64 backend now receives target widths through native codegen options and
  emits 32-bit pointer load/store operations for x32 pointer memory accesses
  instead of assuming AMD64 registers imply 64-bit pointers.
- Function-address materialization now uses a distinct `RelocFuncAddrDisp32`
  relocation path through x64 emit/object/linking instead of reusing call
  relocations for `lea symbol(%rip)` patch sites.
- Linux x32 now has a real TOBJ object-codegen path: it uses x86_64 code
  emission with 32-bit pointer/native integer options, x32 SysV syscall numbers
  (`__X32_SYSCALL_BIT`), target `linux-x32`, and no Windows IAT imports.
  No-runtime executable build is now wired through the native pipeline; x32
  self-host runtime builds are allowed through the same native pipeline, while
  source `run`/`test` execution is guarded by an x32 host-execution probe and
  emits an explicit no-host-fallback diagnostic when the Linux kernel cannot
  execute x32 ABI binaries. Builtin runtime support remains explicitly blocked
  until the full x32 runtime surface is wired and verified.
- Linux x86 now has a real TOBJ library object-codegen path for the current
  i386 slice of the IR: it emits i386 frame setup/teardown, scalar i32 local
  load/store/arithmetic/return code, target `linux-x86`, sorted TOBJ symbols,
  and 8/16/32-bit plus pointer-sized atomic object bytes.
- Linux x86 now has an ELF32 i386 executable build/link path for no-runtime
  programs: `LinkLinuxX86` accepts only `linux-x86` TOBJ inputs, emits an i386
  `int 0x80` exit entry stub, patches data relocations against an ELF32 i386
  layout, and `WriteELF32LinuxX86` writes `ELFCLASS32`/`EM_386` executables.
  On i386-compatible Linux hosts, `tetra run --target x86` and source-file
  `tetra test --target x86 <file>` now execute no-runtime programs through the
  generated ELF32 binary instead of falling back to the host. The x86 backend
  also handles `IRLabel`, `IRJmp`, `IRJmpIfZero`, and no-argument internal
  `IRCall` relocations needed by synthetic no-runtime test runners.
- The x86 backend now emits i386 SysV caller-cleaned stack arguments for
  internal scalar `IRCall`s: argument values are copied into a cdecl call area
  as `arg0, arg1, ...`, calls use TOBJ `RelocCallRel32`, and the caller removes
  both the call area and the expression-stack arguments before pushing scalar
  returns. This enables real no-runtime x86 source programs and tests with
  simple parameterized functions.
- The x86 backend now emits scalar global load/store object code for no-runtime
  programs: `IRLoadGlobal` uses i386 `mov eax, moffs32`, `IRStoreGlobal` uses
  `mov moffs32, eax`, TOBJ records these as `RelocDataAbs32`, `linkcore` keeps
  them separate from RIP-relative `RelocDataDisp32`, and `LinkLinuxX86` patches
  absolute ELF32 `.data` virtual addresses. x64/x32 Linux linkers explicitly
  reject absolute data relocations instead of silently treating them as
  RIP-relative relocations.
- The x86 backend now materializes symbol-backed function addresses for the
  no-runtime fnptr/callback path: `IRSymAddr` emits `mov eax, imm32; push eax`,
  TOBJ records `RelocFuncAddrAbs32`, `linkcore` tracks it separately from
  x64/x32 `RelocFuncAddrDisp32`, and `LinkLinuxX86` patches absolute ELF32
  code virtual addresses. Linux x64/x32 reject this i386-only relocation
  explicitly. This enables source-level `tetra run --target x86` smoke coverage
  for direct symbol-backed callback arguments. Full runtime startup, stdlib, FFI/libc,
  aggregate/float call ABI, and broader argument/return torture coverage remain
  explicitly blocked until the remaining i386 runtime/ABI work lands.
- The x86 backend now has real heap-backed slice allocation/indexing for the
  no-runtime slice surface: `IRAllocBytes` and `IRMakeSliceU8/U16/I32` allocate
  anonymous writable memory through Linux i386 `mmap2` (`int 0x80`) with
  deterministic exit-code-2 failure diagnostics, `IRAllocBytes` records the
  requested byte size in the allocation header, and `IRIndexLoad*`/`IRIndexStore*`
  scale `u8`/`u16`/`i32` indices with unsigned bounds checks that exit 1 on
  invalid access. Source-level `tetra run --target x86` now covers a real
  `make_i32` store/load roundtrip and `core.alloc_bytes(0)` failure. Full
  allocator ownership/freeing, libc/FFI, aggregate/float ABI, runtime startup,
  and stdlib promotion remain explicitly blocked until the remaining i386
  runtime work lands.
- The x86 backend now emits raw memory pointer arithmetic and load/store code
  for the no-runtime unsafe memory surface: `IRPtrAdd`,
  `IRMemReadI32/U8/Ptr`, `IRMemWriteI32/U8/Ptr/ArchPtr`, and their offset
  variants use 32-bit pointer-width addressing, validate negative and
  upper-bound accesses against the allocation header, exit 2 on raw-memory
  violations, and preserve the stored scalar as the helper return value.
  Source-level `tetra run --target x86` now covers `core.store_i32`/`load_i32`,
  `core.ptr_add` with byte load/store, and an upper-bound failure. This still
  does not promote full x86 stdlib/runtime/FFI: ownership/freeing, libc bridge,
  volatile/MMIO hardening, aggregate/float ABI, and production allocator policy
  remain separate work.
- The x86/x32 ABI suites now promote that existing raw-memory lowering evidence
  into release-gated executable smoke names: each target builds a
  `core.alloc_bytes` program that checks allocation-header bounds for
  `store_i32`/`load_i32`, `core.ptr_add`, and byte `store_u8`/`load_u8`, while
  preserving the target syscall bridge requirements. This is still build-only
  ABI evidence, not full runtime allocator/free/panic parity.
- The x86/x32 ABI suites also require raw pointer-slot executable smokes for
  allocation-base and direct `core.ptr_add(base, offset, mem)`
  `core.store_ptr`/`core.load_ptr` over 4-byte pointer slots, so ILP32 and x32
  pointer memory cannot silently widen to LP64 while the targets remain
  build-only/host-probed.
- The x86 backend now supports no-runtime stdout output and string literal data:
  `IRStrLit` appends the literal to the TOBJ data section, materializes its
  ELF32 absolute data address through `RelocDataAbs32`, pushes the usual
  `(ptr,len)` pair, and `IRWrite` lowers to Linux i386 `write(1, ptr, len)` via
  syscall 4 and `int 0x80`. Source-level `tetra run --target x86` now verifies
  both `print("...")` and `print([]u8)` paths. This is still a no-runtime syscall
  slice, not a full libc/stdio/runtime promotion.
- The x86 backend now supports the non-debug scoped-island allocation path for
  no-runtime programs: `IRIslandNew` allocates a 16-byte-header island with
  Linux i386 `mmap2`, `IRIslandMakeSliceU8/U16/I32` uses the same
  `[next, capacity, mmap_len, freed]` bump-pointer header contract as x64 and
  exits 1 on overflow, and `IRIslandFree` lowers to Linux i386 `munmap`
  syscall 91. Source-level `tetra run --target x86` now verifies scoped island
  allocation/load/store/free and overflow diagnostics.
- The x86 backend now honors `--islands-debug` / `BuildOptions.IslandsDebug`
  instead of silently using the non-debug island path: debug `IRIslandNew`
  reserves a 4096-byte protected-header page, initializes `[next, capacity,
  mmap_len, freed]` with `next=4096`, and debug `IRIslandFree` checks the
  freed marker, exits 2 on double-free, sets the marker, and protects the
  payload with Linux i386 `mprotect(PROT_NONE)` syscall 125. Backend tests cover
  the exact object bytes and `tetra run --target x86 --islands-debug` verifies
  the source-level scoped-island path.
- Linux x86 time-runtime use now fails early with a target-aware diagnostic
  instead of falling through to the generic missing actors runtime path:
  `core.time_now_ms`, `core.sleep_ms`, `core.sleep_until`,
  `core.deadline_ms`, and `core.timer_ready` on `linux-x86` report
  `time runtime not supported on linux-x86` with linux-x64 guidance and do not
  write an executable. Linux x32 keeps its existing auto self-host time-runtime
  build path.
- Linux x86 task-runtime use now follows the same target-aware diagnostic
  contract: `core.task_spawn_*`, task join/poll/select builtins, task groups,
  typed task handles, and cancellation checkpoints are detected with source
  positions before runtime selection, report `task runtime not supported on
  linux-x86`, and do not fall through to the generic missing actors-runtime
  error or write an executable.
- Linux x86 actors-runtime use now fails at the same explicit target-runtime
  boundary instead of attempting the SysV self-host runtime and reporting the
  plain `self-host runtime not available` error: actor spawn/send/recv/self/
  sender/yield builtins on `linux-x86` report `actors runtime not supported on
  linux-x86` with source position and linux-x64 guidance. Linux x32 retains the
  existing SysV self-host actor-runtime build path.
- Linux x86 actor state now uses the same target-runtime boundary even when no
  actor builtin is called directly: methods with actor state slots require the
  actor-state load/store runtime, report `actors runtime not supported on
  linux-x86` at the actor method position, preserve linux-x64 guidance, and do
  not write an executable. This keeps actor state from falling through to the
  generic missing-runtime linker path.
- The x86 backend now supports the MMIO primitive pair used by the unsafe
  capability surface: `IRMmioReadI32` and `IRMmioWriteI32` use direct 32-bit
  loads/stores through the address guarded by `cap.io` at the semantic layer,
  and `mmio_write_i32` returns the stored scalar value. Source-level
  `tetra run --target x86` now verifies `core.mmio_write_i32` followed by
  `core.mmio_read_i32`. Broader volatile ordering/device-barrier semantics
  remain a separate hardening item.
- Source-level Linux x86 runtime smoke coverage now includes the broader
  no-runtime matrix expected by the target goal: recursion, while loops,
  stack-passed function calls, structs, enum payload matches, `[]u16`, `[]bool`,
  strings/stdout, globals, callbacks, heap slices, raw pointer memory, scoped
  islands, debug islands, and MMIO all execute through `tetra run --target x86`
  when the host kernel supports i386-compatible binaries. Runtime/std/FFI-heavy
  surfaces still stay behind explicit target diagnostics until implemented.
- Linux x32 now has an internal ELF/linker primitive: `WriteELF32LinuxX32`
  writes ELFCLASS32 objects with `EM_X86_64` per AMD64 ILP32 psABI, and
  `LinkLinuxX32` accepts only `linux-x32` TOBJ inputs, emits the x32
  `__X32_SYSCALL_BIT | exit` entry stub, and patches data relocations using the
  ELF32 x32 layout rather than the x64 ELF64 header size.
- Linux x32 executable builds now use the x32 codegen/link/write path end to end
  for no-runtime programs and self-host runtime programs, producing
  ELFCLASS32/`EM_X86_64` executables with x32 exit syscalls. Source-level
  `tetra run --target x32` and `tetra test --target x32` now route through an
  x32 host-execution probe: they execute only when the Linux host kernel can run
  x32 ABI binaries and otherwise fail with an explicit no-host-fallback
  diagnostic. `RuntimeAuto` now selects the self-host runtime on x32 when the
  requested runtime surface is within the self-host contract, while explicit
  builtin-runtime selection remains an explicit diagnostic instead of falling
  back to the host. x32 supports the single-spawn self-host actor/task slice, but
  multi-spawn actor/task programs, task groups, and typed task handles now fail
  even earlier with target-runtime diagnostics (`multi-spawn actors runtime not
  supported on linux-x32`, `task group runtime not supported on linux-x32`,
  `typed task runtime not supported on linux-x32`) so unsupported runtime
  surfaces do not collapse into a generic builtin-runtime error.
- The unsafe raw-memory surface now separates ABI pointer stores from
  architectural context-frame stores: `core.store_ptr` stays pointer-width
  (4 bytes on x32), while `core.store_arch_ptr` stores register-width addresses
  (8 bytes for x32/x64). The SysV self-host runtime now uses this for saved
  stack pointers, return addresses, and callee-saved register slots so x32 does
  not depend on the false invariant that pointer width equals register width.
  x32 pointer stores also zero-extend their pointer-width return value into the
  64-bit Tetra machine slot so stale high register bits cannot leak through a
  raw store expression.
- The IR and x64 backend now include the first real atomic codegen primitive:
  pointer-sized atomic exchange plus a seq_cst fence. x64 emits qword `xchg`
  and x32 emits dword `xchg`, preserving the pointer-width/register-width split;
  the fence emits a real `mfence`.
- Pointer-sized compare-exchange is now also represented in IR and x64 codegen:
  x64 emits `lock cmpxchg` on qwords, while x32 emits the dword form and returns
  the observed old value from the accumulator. The x32 path loads the expected
  value through `eax` rather than `rax`, so a successful CAS still returns a
  pointer-width, zero-extended accumulator instead of leaking stale high
  register bits.
- Pointer-sized atomic load/store are now represented in IR and x64 codegen.
  Loads use the target pointer width, while stores use pointer-width `xchg` so
  the primitive has seq_cst-safe codegen on x86-family targets; x32 continues to
  use the dword form rather than accidentally touching qwords. x32 atomic
  pointer stores copy the returned stored value through a 32-bit register move,
  keeping the return slot zero-extended to pointer width.
- Pointer-sized atomic fetch-add now has a real IR and x64 backend primitive:
  x64 emits qword `lock xadd`, x32 emits dword `lock xadd`, and both return the
  old memory value without widening x32 pointer-sized atomics to 64 bits.
- Pointer-sized atomic fetch-sub/and/or/xor now have real IR and x64 backend
  primitives. Fetch-sub uses pointer-width `neg` plus `lock xadd`; logical
  fetch ops use a pointer-width `lock cmpxchg` retry loop, with x32 staying on
  dword loads/ops/CAS while x64 uses qwords.
- Fixed 32-bit atomics now have real IR and x64 backend primitives for
  load/store/exchange/CAS/fetch-add/fetch-sub/fetch-and/fetch-or/fetch-xor.
  They always use dword guards and dword atomic instructions on both x32 and
  x64, so `i32` atomics no longer inherit pointer-sized behavior by accident.
- Fixed 64-bit atomics now have real IR and x64 backend primitives for the same
  operation set. They always use qword guards and qword atomic instructions,
  including on x32, so fixed `i64` atomics remain distinct from x32 pointer-sized
  dword atomics.
- Fixed 8-bit and 16-bit atomics now have real IR and x64 backend primitives for
  load/store/exchange/CAS/fetch-add/fetch-sub/fetch-and/fetch-or/fetch-xor.
  They use byte/word guards and byte/word atomic memory instructions with
  explicit zero-extension of returned narrow values, including stored values
  returned by store operations.
- The i386 backend now covers the same supported 8/16-bit logical fetch
  surface for x86's declared atomic widths: `fetch_and`, `fetch_or`, and
  `fetch_xor` lower to byte/word `lock cmpxchg` retry loops and return the
  observed old value zero-extended to the Tetra scalar slot. The x86
  `--atomic-stress` object matrix now checks these source-level operations
  instead of only the 32-bit logical fetch path.
- The i386 backend now keeps narrow atomic store semantics distinct from
  exchange: `atomic_store_u8` and `atomic_store_u16` still lower through
  byte/word `xchg` for atomicity, but preserve and return the width-truncated,
  zero-extended stored value instead of the old memory value. Exchange/fetch/CAS
  operations continue returning the observed old value.
- Atomic fences now model memory-order variants in IR. On x86-family codegen,
  relaxed/acquire/release/acq_rel fences are explicit no-ops under TSO, while
  seq_cst remains a real `mfence`; the lower layer now has a tested
  `MemoryOrder`-to-IR mapping with explicit diagnostics for unknown orders.
- The lower layer now has a tested fixed-width atomic op mapper for
  8/16/32/64-bit values covering load/store/exchange/strong-CAS/weak-CAS/fetch
  ops. Fence lowering is kept on the memory-order helper, unsupported widths
  report explicit diagnostics, and weak CAS lowers to the same single
  `cmpxchg` IR as strong CAS on x86-family targets. This is a valid
  non-spurious weak-CAS implementation; the atomic stress suite now also
  exercises retry-loop callers and randomized yielding through the compiler-owned
  concurrency oracle described below.
- Pointer-sized atomic lowering now has a separate tested op mapper to keep
  pointer atomics distinct from fixed-width integer atomics; this preserves the
  x32 rule that pointer-sized operations are 32-bit even though the machine
  register file is x86_64.
- Source-level unsafe `core.atomic_*` builtins now expose the implemented atomic
  IR for `u8`, `u16`, `i32`, `i64`, and `ptr` values with compile-time
  memory-order suffixes. The builtins require `cap.mem`, report `mem` effects,
  lower to fixed-width or pointer-sized atomic IR, and include fence lowering
  through the memory-order helper. The compiler now has a first-class one-slot
  `i64` scalar sufficient for these atomic load/store/exchange/CAS/fetch value
  flows; broader `i64` arithmetic is still outside this slice.
- Source-level `core.atomic_compare_exchange_weak_*` now lowers for fixed-width
  and pointer-sized atomics. The x64-family backend emits the same strong
  `cmpxchg` sequence, which is permitted for weak CAS because it simply never
  fails spuriously.
- Invalid source-level atomic builtin forms now report targeted diagnostics for
  disallowed memory orders, unsupported source widths, unknown memory-order
  suffixes, and unknown atomic operations instead of falling through to a
  generic unknown-function error.
- `tetra test --target x32 --abi` now runs a real x32 ABI/object suite without
  host execution fallback. It validates the x32 target/data model, pointer and
  native-int byte layouts, x32 SysV classifier behavior with x86_64 registers
  but 32-bit pointer/usize/isize ABI slots, sign/zero-extension metadata,
  vararg `%al` metadata, packed/unpacked aggregate pass/return behavior,
  pointer-sized and fixed 64-bit atomic layout, a pointer-only atomic ABI-width
  object check, Linux x32 TOBJ output, x32 syscall numbering, and
  qword-vs-dword weak-CAS object code. It also builds a source-level x32
  executable matrix for recursion, global/string length metadata, direct
  function callbacks, control flow, stack-passed calls, structs, enum payload
  matches, heap slices, `[]u8`, `[]u16`, `[]bool`, raw pointer add/load/store,
  pointer load/store, scoped island allocation/free, and MMIO read/write. The
  harness verifies the outputs are ELFCLASS32/`EM_X86_64` executables using x32
  syscall numbers rather than plain x64 syscalls or i386 `int 0x80`. These
  checks do not execute on the host, so there is no host fallback or fake
  runtime pass.
- `tetra test --target x32 --atomic-stress` now runs a real x32 object/target
  atomic suite. It checks the complete target memory-order validation matrix for
  8/16/32/64-bit atomics, rejects misalignment/unsupported widths, compiles an
  x32 TOBJ covering load/store/exchange/CAS/weak-CAS/fetch/fence builtins across
  `u8`, `u16`, `i32`, `i64`, and `ptr`, and verifies explicit diagnostics for
  invalid atomic builtin forms. The suite also builds a pointer-only atomic
  object and rejects opposite-width atomic instruction fingerprints, so x32
  `ptr` atomics must stay dword even though fixed `i64` atomics in the same
  backend are qword. It also runs a deterministic compiler-owned concurrency
  oracle over the target pointer width, covering contended CAS loops,
  release/acquire message passing, seq_cst ordering, ABA-stamped pointer cases,
  false-sharing counters, injected weak-CAS spurious failures, randomized
  yields, and masked 8/16-bit CAS loops. `TETRA_ATOMIC_STRESS_ITERS` can raise
  or lower the oracle iteration count without changing the target matrix.
- `tetra test --target x64 --atomic-stress` now uses the same compiler-owned
  stress contract for the LP64 x64 backend: target memory-order validation,
  misalignment/unsupported-width rejection, source-level atomic diagnostics, and
  real x64 TOBJ object code checks for fixed-width and pointer-sized atomic
  builtins, plus the same target-width concurrency oracle. `tetra test --target
  x86 --atomic-stress` now also passes the
  compiler-owned validation, object byte, and diagnostic checks for x86’s
  supported 8/16/32-bit and pointer-sized atomic contract, including a
  source-level rejection for unsupported `i64` atomics before backend codegen
  and the 32-bit pointer-width concurrency oracle. x86, Linux x64, macOS x64,
  Windows x64, and x32 all include a pointer-only atomic object-width check and
  the concurrency oracle in the brutal matrix.
- `tetra test --target x86 --fuzz`, `tetra test --target x64 --fuzz`, and
  `tetra test --target x32 --fuzz` now run deterministic compiler-owned
  property tests for layout and object signatures. They fuzz scalar aggregate
  and array layout against an independent oracle, prove pointer-sensitive x86,
  x32, and x64 layouts/register models do not collapse, reject x86/x32 fixed
  arrays whose byte size exceeds the 32-bit target `usize` limit while allowing
  the corresponding x64 layout, build randomized TOBJ libraries with mixed
  `i32`/`i64`/`ptr` signatures, and fuzz x86/x64/x32 target aliases plus
  invalid aliases.
- `tetra test --target x86 --abi`, `tetra test --target x64 --abi`,
  `tetra test --target macos-x64 --abi`, and
  `tetra test --target windows-x64 --abi` now run compiler-owned ABI classifier
  suites instead of CLI-only placeholder checks. The x86 suite validates the
  i386 SysV target model, stack-only caller-cleaned arguments, `eax`/`edx:eax`
  scalar returns, x87 float returns, hidden stack `sret`, stack varargs, a real
  linux-x86 scalar `@export` TOBJ smoke, a linux-x86 atomic object smoke, a
  pointer-only atomic ABI-width object check, and a 32-bit ELF executable matrix
  covering recursion, global/string length metadata, direct function callbacks,
  control flow, local memory, and aggregate enum return lowering through the
  internal `eax`/`edx`/`ecx` slot ABI. The x64 suites validate SysV AMD64 and
  Microsoft x64 register assignment, data models, vararg metadata (`%al` SSE
  upper bound or Win64 shadow space/float mirrors), aggregate pass/return
  behavior, pointer-only atomic ABI-width object checks for Linux/macOS/Windows
  x64, a Linux-x64 scalar `@export` TOBJ smoke, a Linux-x64 atomic object smoke,
  and a 64-bit ELF executable matrix. The x32 suite includes an x32 scalar
  `@export` TOBJ smoke, x32 atomic object smoke, and the expanded x32 executable
  matrix.
- Non-runtime `@export` functions on native targets now reject struct/aggregate
  parameter and return signatures before lowering/codegen with an explicit
  aggregate C ABI diagnostic. Linux x86 and x32 build canonical `ptr`,
  `rawptr`, `nullable_ptr`, and `ref` object smokes, including a nullable
  null-return smoke and a non-nullable `ref` null-return type diagnostic, while
  still rejecting unverified function-pointer `@export` parameters and returns
  until those C ABI wrappers are verified; this avoids
  silently using an internal slot/callable ABI as if it were a verified C ABI.
  Wider/float target-layout scalar spellings remain covered by the explicit
  native-scalar diagnostic below. This keeps FFI honest while runtime
  `__rt`/`__tetra_*` exports continue to use the compiler-owned internal slot
  ABI for self-host runtime objects.
- The x86/x32 ABI suites now also prove source-level ILP32 native/libc scalar
  spellings `usize`, `isize`, `size_t`, `ssize_t`, `native_int`,
  `native_uint`, `c_long`, and `c_ulong` lower as 1-slot 32-bit `@export`
  object wrappers with target-specific symbol metadata.
  Linux x64 still rejects those source-level native/libc aliases, and x86/x32
  still reject wider/float target-layout spellings with the explicit
  target-layout-scalar diagnostic and no output object.
- The x86 and x32 ABI/default target suites now include compiler-owned
  stdlib/runtime boundary diagnostics: small `core.fs_exists` and
  `core.net_socket_tcp4` programs must fail on those targets with `TETRA3003`,
  target-specific `filesystem`/`networking` messages, linux-x64 migration
  guidance, and no emitted output. They also include target-runtime boundary
  diagnostics: x86 verifies `time`, `task`, actor spawn/mailbox, and actor-state
  surfaces reject with target-specific `TETRA3003`; x32 verifies multi-spawn
  actor/task, task-group, and typed-task surfaces reject with x32-specific
  diagnostics while the supported single-spawn self-host slice remains separate.
  This keeps currently unsupported stdlib/runtime bridges explicit instead of
  becoming host fallback.
- `tetra test --all-targets` now runs the x86, Linux x64, macOS x64,
  Windows x64, and x32 ABI matrix and currently passes 102/102 checks.
  `tetra test --all-targets --brutal` now runs a real aggregate harness
  instead of a blanket unsupported diagnostic: ABI checks, atomic stress
  checks, and fuzz checks execute for real across that matrix and currently
  report 142/142 passing checks. The macOS x64 and Windows x64 ABI cells now
  include real object smoke checks: macOS verifies SysV-style object data
  relocations without Windows IAT imports, while Windows verifies Win64
  `kernel32` IAT relocations. Combined per-target suite flags and per-target
  runners outside the implemented cells still return explicit diagnostics
  rather than fake skipped tests or host fallback.
- The all-targets brutal JSON report now keeps target-specific evidence files
  for OS-specific x64 variants (`tetra:x64-*`, `tetra:macos-x64-*`,
  `tetra:windows-x64-*`) so SysV Linux/macOS and Win64 checks do not collapse
  into one ambiguous `x64` report group. Suite-generated results now also use
  validator-compatible `__tetra_test_` function names and sorted
  `filename,index` result ordering, so the brutal matrix JSON can be checked by
  `tools/cmd/validate-test-report`.
- `compiler/target` now has tested byte-layout helpers for fixed arrays,
  `ptr+i32` slice/string views, enum tag+payload storage, nested structs, and
  packed structs. These helpers preserve the x32 invariant that pointer/native
  integer layout is 32-bit while the ISA register model remains x86_64. Fixed
  arrays now use checked layout multiplication and explicit target-native size
  diagnostics, so x86/x32 reject layouts larger than `u32::MAX` bytes instead
  of silently relying on host integer width.
- Build-only x86/x32 target metadata now names the verified ABI/atomic evidence
  in `unsupported_reason` instead of only listing older no-runtime slices. The
  canonical metadata and CLI JSON both require the i386/x32 ABI classifiers,
  explicit filesystem/networking stdlib plus target-runtime boundary diagnostics,
  x86/x32 `rawptr`/`nullable_ptr`/`ref` plus ILP32 native/libc scalar `@export` object smokes,
  function-pointer `@export` diagnostics, and remaining source
  target-layout scalar diagnostics,
  pointer-only atomic ABI-width object checks, x32 dword pointer atomics, x32
  syscall numbering, and source-level atomic diagnostics to remain visible
  while full runtime/stdlib/FFI support stays explicitly blocked.
