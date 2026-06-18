# Linux x86/x64/x32 Full Support Plan

**Goal:** promote the Linux native target family without fake support:
`linux-x64` remains the stable production baseline, while `linux-x86` and
`linux-x32` advance only when their runtime, stdlib, FFI, ABI, linker, atomic,
smoke, fuzz, and release-gate evidence is real.

**Current truth:** `linux-x64` is supported. `linux-x86` and `linux-x32` are
build-only, host-probed targets. `linux-x86` has a narrow self-host logical
time-runtime smoke for time-only programs and a bounded two-spawn i386 self-host
actors/task/task-group smoke, single-spawn typed-task/staged typed-task/typed task-group/actor-state
smoke, plus `fs_exists` filesystem runtime
and filesystem+scheduler composition smokes, plus current `core.net`
networking runtime smokes including epoll readiness, with no-runtime executable
stdout/string-literal, `core.net_write(2)` stderr fd, allocator
success/failure executable, raw memory bounds executable, raw pointer-slot
base/offset executables, and scoped island/free executable ABI smoke coverage.
`linux-x32` has self-host time, bounded two-spawn actors/task/task-group, single-spawn
typed-task/staged typed-task/typed task-group,
and actor-state runtime build evidence plus its own `fs_exists` filesystem runtime
smoke, x32 filesystem+scheduler composition smoke, current `core.net`
networking runtime smokes, and no-runtime executable stdout/string-literal plus
`core.net_write(2)` stderr fd plus allocator success/failure executable, raw
memory bounds executable, raw pointer-slot base/offset executables, and scoped
island/free executable ABI smoke coverage.
x86/x64/x32 ABI
reports now build canonical pointer plus `c_int`/`c_uint` `@export` FFI object smokes, and
`linux-x64` has filesystem+scheduler composition and networking runtime
regression smokes. Multi-spawn,
surface, distributed actor, full allocator/free parity, panic, and syscall/stdlib parity remain
unpromoted; x86/x32 Surface and distributed-actor calls are covered by explicit
ABI-report target-aware diagnostics rather than production runtime support.
x86/x32 staged typed-task and typed task-group composition
remain build-only/host-probed evidence. They may run source/test slices only on a host that can
execute the
exact target ABI, and unsupported surfaces must emit target-aware diagnostics
with no output artifact.

## Non-Negotiables

- Do not collapse `linux-x32` into either `linux-x64` or `linux-x86`.
  `linux-x32` uses x86_64 registers, 32-bit pointers/native ints, x32 SysV
  ABI, x32 syscall numbers, and ELFCLASS32/`EM_X86_64`.
- Do not let `linux-x86` borrow x64 assumptions. `linux-x86` is i386 SysV,
  ELFCLASS32/`EM_386`, 32-bit registers/pointers/native ints, cdecl stack
  arguments, caller cleanup, and an i386 syscall bridge.
- Keep `linux-x64` as the regression oracle for source semantics, runtime ABI
  symbols, stdlib behavior, atomics, linking, smoke, and performance. The
  oracle includes filesystem+scheduler composition and scheduler-restriction
  regression smokes so build-only target
  restrictions cannot become linux-x64 behavior.
- A target is not supported until the release validator accepts real runner or
  documented target-runner evidence. Build-only evidence, metadata-only
  reports, docs-only reports, skipped suites, fake runners, and host fallback
  are forbidden.

## Strict Promotion Matrix

| Layer | `linux-x64` baseline | `linux-x86` required state | `linux-x32` required state | Promotion gate |
| --- | --- | --- | --- | --- |
| Target model | Supported LP64 SysV AMD64 ELF64 | i386 SysV ILP32 ELF32 `EM_386`; build-only until gates pass | x32 SysV x86_64 ISA with 32-bit ptr/native-int ELF32 `EM_X86_64`; build-only until gates pass | `tetra targets --format=json` plus `validate-targets` and `validate-linux-native-targets` |
| Byte layout | 64-bit ptr/native-int oracle | 32-bit ptr/native-int, i386 alignment, no x64 slot leaks | 32-bit ptr/native-int with 64-bit registers, no LP64 collapse | target layout tests and fuzz suite |
| ABI classifier | SysV AMD64 | i386 SysV scalar, aggregate, float, varargs, cdecl cleanup | x32 SysV scalar, aggregate, varargs, pointer/native C ABI | `tetra test --target <target> --abi` |
| Linker/object | ELF64 executable/object | ELF32 i386 executable/object, absolute i386 relocs where required | ELF32 x86_64/x32 executable/object, x32 relocs/syscall entry, x32 SysV `ctx_switch` object smoke | ABI smoke plus object/linker tests |
| Runtime startup | production runtime path | i386 startup, panic, allocator/free, full time/task/actor bridge; current evidence is self-host logical time-only plus bounded two-spawn actors/task/task-group, single-spawn typed-task/staged typed-task/typed task-group/actor-state, allocator success/failure, raw memory bounds, raw pointer-slot base/offset, and scoped island/free executable lowering, `fs_exists`, current `core.net` networking, and filesystem+scheduler composition smokes | x32 startup and builtin/self-host policy without silent x64 fallback; current evidence includes self-host time, bounded two-spawn actors/task/task-group, single-spawn typed-task/staged typed-task/typed task-group, actor-state, allocator success/failure, raw memory bounds, raw pointer-slot base/offset, and scoped island/free executable lowering, filesystem, and current `core.net` networking smokes | runnable runtime smoke |
| Syscall/stdlib | Linux-x64 runtime/stdlib baseline | current i386 bridge has `fs_exists`, the full current `core.net` runtime ABI, `core.net_write(2)` stderr fd, allocator executable mmap2 plus checked invalid-size and post-`mmap` error exit evidence, raw memory bounds executable evidence for `ptr_add` plus byte store/load, raw pointer-slot base/offset executable evidence for `store_ptr`/`load_ptr`, scoped island/free `munmap`/debug `mprotect` evidence, and filesystem+scheduler composition evidence; broader io/fs/runtime composition remains unpromoted | current x32 bridge has `fs_exists`, the full current `core.net` x32-syscall runtime ABI, `core.net_write(2)` stderr fd, allocator executable x32-syscall mmap plus checked invalid-size and post-`mmap` error exit evidence, raw memory bounds executable evidence for `ptr_add` plus byte store/load, raw pointer-slot base/offset executable evidence for `store_ptr`/`load_ptr`, scoped island/free x32-syscall `munmap`/debug `mprotect` evidence, and filesystem+scheduler composition evidence; broader io/fs/runtime composition remains unpromoted | stdlib smoke matrix |
| FFI boundary | verified scalar x64 boundary plus pointer-param and `c_int`/`c_uint` `@export` object regression evidence; broader LP64 source target-layout scalars remain gated | canonical i386 `ptr`/`rawptr`/`nullable_ptr`/`ref`, `c_int`/`c_uint`, and complete ILP32 native/libc scalar object evidence; function-pointer and aggregate/float ABI remain gated; wider/float target-layout scalar spellings stay in source diagnostics | canonical x32 `ptr`/`rawptr`/`nullable_ptr`/`ref`, `c_int`/`c_uint`, and complete ILP32 native/libc scalar object evidence; function-pointer wrappers remain gated; wider/float target-layout scalar spellings stay in source diagnostics | ABI + FFI diagnostics/object reports |
| Atomics | 8/16/32/64 and pointer-sized 64-bit oracle | 8/16/32 and pointer-sized 32-bit; no false 64-bit lock-free claim | pointer-sized 32-bit plus fixed `i64` 64-bit operations where supported | `--atomic-stress` |
| Test runner | host-native runnable | no host fallback; run only on i386-capable host/runner | no host fallback; run only on x32-capable host/runner | no-host-fallback diagnostics and real runner report |
| Release gate | supported | may become supported only after all required reports pass | may become supported only after all required reports pass | artifact hashes plus Linux native target smoke |

## Implementation Workstreams

1. **Docs and truth gates.**
   Keep `docs/spec/runtime_abi.md`, `docs/spec/current_supported_surface.md`,
   `docs/spec/cli_contracts.md`, `compiler/target`, generated manifests, and
   release scripts aligned with the matrix. Do not move x86/x32 out of
   `build_only` until the validator-backed promotion gate passes. Linux native
   target metadata must expose `runtime_status`, `stdlib_status`, `ffi_status`,
   `runner_probe_command`, `release_gate`, and `evidence_artifacts` so partial
   x86/x32 evidence remains machine-checkable rather than hidden in prose.

2. **Shared contracts and per-ABI lowering.**
   Keep shared target-independent contracts for layout, ABI facts, runtime
   symbol requirements, object metadata, stdlib bridge selection, FFI gating,
   and test reports. Keep lowering/backend/linker/syscall/runtime decisions
   per ABI. The target metadata exposes the Linux syscall pack explicitly:
   x64 is x86_64 `syscall`, x86 is i386 `int 0x80`, and x32 is x86_64
   `syscall` with x32 syscall-bit numbering.

3. **Finish `linux-x32`.**
   Complete runtime ABI coverage, builtin/self-host runtime policy,
   filesystem/networking/io/time/task/actor parity where intended, x32
   pointer/native C ABI wrappers, stdlib gates, executable run/test evidence,
   and explicit no-host-fallback diagnostics.

4. **Finish `linux-x86`.**
   Complete i386 codegen parity for the supported IR, broaden the current
   time-only, allocator success/failure executable, raw memory bounds executable,
   raw pointer-slot base/offset executable, scoped island/free executable, and bounded two-spawn
   self-host smokes into runtime startup/allocator/free/panic/time/task/actor/syscall bridge,
   filesystem/networking/stdout/stderr, i386 FFI, aggregate/float ABI, atomic
   contract, and source run/test matrix.

5. **Protect `linux-x64`.**
   Add regression tests that prove x64 keeps LP64/SysV/ELF64/runtime/stdlib
   behavior and does not inherit x86/x32 restrictions.

6. **Release scripts and validators.**
   Keep `scripts/release/post_v0_4/linux-native-targets-smoke.sh`,
   `linux-x86-smoke.sh`, `linux-x32-smoke.sh`,
   `tools/cmd/validate-linux-native-targets`, `validate-targets`, and artifact
   hash validation in the release path.

7. **Final promotion.**
   Only after real evidence passes, update `compiler/target` status and the
   generated docs/feature manifests. The promotion patch must include the
   successful target reports and no-host-fallback behavior for unsupported
   runner environments.

## Required Evidence Commands

```sh
./tetra targets --format=json
go run ./tools/cmd/validate-targets --report <targets.json>
go run ./tools/cmd/validate-artifact-hashes --write --root <report-dir> --out <report-dir>/artifact-hashes.json
go run ./tools/cmd/validate-artifact-hashes --manifest <report-dir>/artifact-hashes.json
go run ./tools/cmd/validate-linux-native-targets --targets <targets.json> --artifact-hashes <report-dir>/artifact-hashes.json --target <triple>:<abi.json>:<atomic.json>:<fuzz.json> --brutal <brutal.json>
./tetra test --target x64 --abi --report=json
./tetra test --target x64 --atomic-stress --report=json
./tetra test --target x64 --fuzz --report=json
./tetra test --target x86 --abi --report=json
./tetra test --target x86 --atomic-stress --report=json
./tetra test --target x86 --fuzz --report=json
./tetra test --target x32 --abi --report=json
./tetra test --target x32 --atomic-stress --report=json
./tetra test --target x32 --fuzz --report=json
./tetra test --all-targets --brutal --format=json
bash scripts/release/post_v0_4/linux-native-targets-smoke.sh --report-dir <dir>
```

The Linux native smoke scripts also write `*-runner.json` reports. On a host
that can execute the target ABI, those are normal `tetra test --format=json`
reports containing the required release runner smokes: `runner arithmetic`,
`runner alloc memory`, `runner filesystem`, `runner stderr fd`, `runner time`,
`runner network socket`, `runner network options`, and `runner task join`. On an unsupported x86/x32
runner host, they must be JSON
diagnostics that include the target, host identity, exact probe command, and
no-host-fallback reason; linux-x64 is not allowed to use a blocked runner
diagnostic. The runner evidence kind must match the same `targets.json`
metadata: passing runner reports require `run_supported: true`, and no-host
diagnostics require `run_supported: false`.

## Done When

- `tetra targets --format=json` truthfully reports `linux-x64`,
  `linux-x86`, and `linux-x32`.
- `linux-x86` and `linux-x32` stay build-only unless every required runtime,
  stdlib, FFI, ABI, atomic, smoke, fuzz, brutal, artifact-hash, and runner gate
  passes.
- Unsupported runner environments produce explicit no-host-fallback diagnostics.
- No docs, feature registry entry, manifest, release note, or script claims
  production support without validator-backed evidence.
