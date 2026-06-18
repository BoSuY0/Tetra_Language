# Tetra Projects Bug Hunt

Status: closure pass complete on 2026-05-19; historical discovery evidence
remains below for traceability.
Scope: create and run small Tetra microservice-shaped programs to find language,
runtime, edge-case, and security bugs in the Tetra toolchain.

## Working Rules

- Keep repro code small and self-contained.
- Record only evidence-backed findings as confirmed bugs.
- Include the command, observed behavior, expected behavior, and risk.
- Scratch probes may live under `/tmp`; this file is the durable project log.
- The repository worktree was already dirty when this log was created, so this
  audit only claims changes made to this file unless stated otherwise.

## Environment Snapshot

- Date: 2026-05-18
- Repository: `/home/tetra/Desktop/Projects/Tetra_Language`
- Graphify report: `graphify-out/GRAPH_REPORT.md`
- Graphify graph commit and `git rev-parse HEAD`: `b8846534066cd9400ab1b3fc902973fc2ef7fc57`
- Tetra CLI: `./tetra version` returned `v0.4.0`
- Initial log state: `TetraProjects_Bugs.md` was missing and has been created
  for this continuous bug-hunt thread.

## Confirmed Bugs

## Closure Pass - 2026-05-19

Status: all confirmed `BUG-001` through `BUG-075` are closed against the
current source tree and local `./tetra` entrypoint.

Closure evidence:

```sh
GOCACHE=/tmp/tetra-language-go-cache go test ./compiler/... ./cli/... ./tools/...
GOCACHE=/tmp/tetra-language-go-cache go test ./compiler/tests/semantics -run 'TestDeferRejectsLaterConsumeOfCapturedDescendant|TestDeferAllowsSiblingCaptureAfterDescendantConsume' -count=1
GOCACHE=/tmp/tetra-language-go-cache go test ./compiler/tests/semantics -run 'TestSmallInt|TestOptional|TestConst|TestDefer|TestCompound|TestArray|TestSlice|TestString' -count=1
GOCACHE=/tmp/tetra-language-go-cache go test ./compiler/internal/semantics -run 'TestCheckNoConsumedDescendantsCanonicalizesAliasPaths|TestCheckNotConsumedCanonicalizesAliasPaths|TestMergeFlowWithLabelsIntersectsOwnershipAliases' -count=1
GOCACHE=/tmp/tetra-language-go-cache go build -o tetra ./cli/cmd/tetra
./tetra version
./tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.tetra
./tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_sibling_after_field_consume_control.tetra
```

Results:

- Full submodule test suite passed for `compiler`, `cli`, and `tools`.
- Local `./tetra` was rebuilt from current source and reports `v0.4.0`, closing
  the stale-entrypoint condition recorded as `BUG-002`.
- `BUG-063` now rejects the field/whole-struct/whole-enum/whole-optional
  deferred cleanup repros with `defer cleanup captures value ... before cleanup
  ran`, while the sibling-field and sibling-payload controls still pass.
- The root command `go test ./...` is intentionally not the closure command for
  this workspace: the repository uses `go.work` submodules, so the evidence
  command is `go test ./compiler/... ./cli/... ./tools/...`.

Cluster status:

- `BUG-001`, `BUG-004`, `BUG-005`, `BUG-009`, `BUG-012`, `BUG-013`: closed by
  the stdlib/runtime regression coverage in the full compiler/CLI/tools suite.
- `BUG-002`, `BUG-007`, `BUG-008`: closed by rebuilding `./tetra` plus the CLI
  and eco packaging tests in the full suite.
- `BUG-003`, `BUG-006`, `BUG-014`, `BUG-016`, `BUG-017`, `BUG-032`,
  `BUG-033`, `BUG-037`, `BUG-065`: closed by actor/task transport and compiler
  safety regression coverage in the full suite.
- `BUG-010`, `BUG-011`, `BUG-015`, `BUG-018`, `BUG-019`, `BUG-020`,
  `BUG-023`, `BUG-024`, `BUG-025`, `BUG-026`, `BUG-027`, `BUG-028`,
  `BUG-029`, `BUG-030`, `BUG-031`, `BUG-056`: closed by semantics, lowering,
  backend, runtime, and compiler regression coverage in the full suite.
- `BUG-021`, `BUG-022`, `BUG-034`, `BUG-035`, `BUG-036`, `BUG-038`,
  `BUG-039`, `BUG-040`, `BUG-041`, `BUG-042`, `BUG-043`, `BUG-063`,
  `BUG-064`: closed by flow/defer/privacy/budget semantics and safety
  regression coverage in the full suite, with `BUG-063` additionally verified
  by the focused RED/GREEN commands above.
- `BUG-044` through `BUG-062`, and `BUG-066` through `BUG-072`: closed by
  export/ABI/object metadata, malformed symbol, and TOBJ regression coverage in
  the full suite.
- `BUG-073`, `BUG-074`, `BUG-075`: closed by `wasm32-wasi` and `wasm32-web`
  backend regression coverage in the full suite.

### BUG-001 - `lib.core.networking.retry_backoff_ms` overflows before applying `max_ms`

- Area: stable stdlib networking policy helper.
- Severity: medium; security/availability edge for service retry policies.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/bughunt/net_backoff_cap_bypass.tetra`
- Command:

```sh
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-001/bughunt/net_backoff_cap_bypass.tetra
```

- Observed: command printed `exit status 43`. The repro returns `43` when
  `networking.retry_backoff_ms(31, 2, 1000)` returns `0`.
- Expected: the helper should not wrap below the cap. Under the documented
  contract, it starts at `base_ms`, doubles per attempt, and caps at non-negative
  `max_ms`; this input should return `1000` or otherwise reject/saturate safely.
- Why it matters: a microservice using this helper for capped retry delays can
  turn a large attempt count into a zero-delay retry loop, bypassing the intended
  cap and creating a denial-of-service style busy retry edge.
- Supporting evidence:
  - `go run ./cli/cmd/tetra check .../net_backoff_cap_bypass.tetra` succeeded.
  - `docs/user/platform/standard_library_guide.md` documents capping at non-negative
    `max_ms`.
  - `lib/core/io/networking.tetra` multiplies `value = value * 2` in a loop before
    comparing with `max_ms`, so overflow can occur before the cap check.

### BUG-002 - Repository `./tetra` binary is stale relative to current source/runtime checks

- Area: local toolchain entrypoint artifact.
- Severity: high for this workspace's CLI trust; source-level runtime guard is
  present, but the checked local entrypoint does not enforce it.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/bughunt/mem_oob_store.tetra`
- Commands:

```sh
./tetra run /tmp/tetra-bug-hunt/session-001/bughunt/mem_oob_store.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-001/bughunt/mem_oob_store.tetra
go test ./compiler -run TestBuildRawStoreI32AllocationBaseWidthDiagnostic -count=1
```

- Observed:
  - `./tetra run .../mem_oob_store.tetra` exited `42`, meaning a 4-byte
    `core.store_i32` into `core.alloc_bytes(1)` completed.
  - `go run ./cli/cmd/tetra run .../mem_oob_store.tetra` printed
    `exit status 2`, which is the expected runtime bounds diagnostic.
  - `go test ./compiler -run TestBuildRawStoreI32AllocationBaseWidthDiagnostic -count=1`
    passed.
- Expected: the documented local entrypoint from README should match current
  source behavior for security/runtime diagnostics after source changes, or the
  repo should force rebuild detection before relying on it.
- Why it matters: following README's `./tetra run` path in this workspace can
  run a stale compiler/runtime that misses a raw-memory bounds diagnostic already
  covered by current source tests.

### BUG-003 - `actor-net` routes spoofed frames from unregistered TCP connections

- Area: distributed actor transport broker (`cli/internal/actornet`).
- Severity: high; transport authentication/authorization boundary.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/actornet_spoof_repro.go`
- Command:

```sh
go run /tmp/tetra-bug-hunt/session-001/actornet_spoof_repro.go
```

- Observed: command printed
  `SPOOF_ROUTED source=1 dest=2 seq=99 payload=12345`.
- Expected: a connection that has not completed a `FrameHello` registration
  should not be able to route `FrameSendI32` or any other payload frame. A
  registered connection should also only be able to send frames whose
  `SourceNodeID` matches its registered node.
- Evidence path:
  - The scratch client started `go run ./cli/cmd/tetra actor-net`, registered
    only node 2 with a valid HELLO, then opened a second TCP connection without
    HELLO and sent `FrameSendI32{SourceNodeID: 1, DestNodeID: 2}`.
  - Node 2 received the spoofed payload.
  - `cli/internal/actornet/broker.go` `handleConn` calls `routeFrame(frame)` for
    non-HELLO frames even when `registeredNode == 0`.
  - `routeFrame` routes by `frame.SourceNodeID`/`frame.DestNodeID` and does not
    verify that the sending connection owns `frame.SourceNodeID`.
- Why it matters: any local process that can connect to the loopback broker can
  inject messages as another node, confuse remote actor identity, and spoof
  failure/status traffic unless higher layers independently authenticate every
  frame.

### BUG-004 - `filesystem.exists` accepts embedded-NUL paths and checks only the prefix

- Area: stable filesystem stdlib/runtime ABI (`lib.core.filesystem.exists` /
  `__tetra_fs_exists`).
- Severity: high for any service that validates or authorizes paths with Tetra
  `String` values before host filesystem access.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/bughunt/fs_nul_truncation.tetra`
- Setup and commands:

```sh
python -c 'from pathlib import Path; Path("/tmp/tetra-bug-hunt/session-001/nul_prefix").write_text("exists", encoding="utf-8")'
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-001/bughunt/fs_nul_truncation.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-001/bughunt/fs_nul_truncation.tetra
```

- Observed:
  - `check` succeeded.
  - `run` printed `exit status 42`, the repro's sentinel for
    `filesystem.exists("/tmp/.../nul_prefix\0_suffix", cap)` returning true.
  - A byte check of the source confirmed one real NUL byte in the string
    literal.
- Expected: a filesystem helper that accepts Tetra `String` as pointer+length
  should reject embedded NUL bytes or return false for such paths. It should not
  silently ask Linux `access(2)` about the prefix before the NUL.
- Evidence path:
  - `compiler/internal/actorsrt/actorsrt_core.go` `emitFilesystemExists`
    copies `path_len` bytes to a stack buffer and then appends a NUL terminator
    before calling Linux `access`.
  - The copy loop does not reject NUL bytes already present in the input.
- Why it matters: a microservice can validate/log/compare a full Tetra path
  string while the runtime checks a shorter host path. That creates path
  confusion and policy-bypass risk wherever `exists` is used as an authorization
  or routing predicate.

### BUG-005 - `crypto.mix_seed` can return a negative value on `Int` minimum overflow

- Area: stable crypto interface helper (`lib.core.crypto.mix_seed`).
- Severity: low to medium; deterministic helper edge that can surprise callers
  using it as a non-negative seed/mixer.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/bughunt/crypto_mix_seed_minint.tetra`
- Command:

```sh
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-001/bughunt/crypto_mix_seed_minint.tetra
```

- Observed: command printed `exit status 42`, the repro's sentinel for
  `crypto.mix_seed(65075262, 2) < 0`.
- Expected: `mix_seed` should not return a negative value after entering its
  `if mixed < 0` normalization branch. It should either safely saturate, apply
  modulo in a way that preserves a non-negative result, or document that signed
  overflow can escape as a negative seed.
- Evidence path:
  - `lib/core/data/crypto.tetra` computes `mixed = seed * 33 + value`.
  - For the repro input this overflows to the minimum signed `Int` value.
  - The branch `return 0 - mixed` overflows again and remains negative.
- Why it matters: even though this module is explicitly not cryptographic,
  deterministic service examples may use `mix_seed` to pick retry, shard, or
  fixture values; negative output can break array indexing, modulo assumptions,
  or policy branches.

### BUG-006 - `actor-net` lets one TCP connection own multiple node identities

- Area: distributed actor transport broker (`cli/internal/actornet`).
- Severity: high; identity lifecycle and message routing isolation.
- Reproducer: `/tmp/tetra-bug-hunt/session-001/actornet_multi_hello_repro.go`
- Command:

```sh
go run /tmp/tetra-bug-hunt/session-001/actornet_multi_hello_repro.go
```

- Observed: command printed
  `MULTI_HELLO_STALE_ROUTE source=3 dest=1 payload=77`.
- Expected: a broker connection should complete exactly one HELLO identity, or
  a later HELLO should replace the previous identity atomically and remove the
  old node mapping. It should not leave the same TCP connection registered for
  multiple node IDs.
- Evidence path:
  - The scratch client sent HELLO for node 1, then HELLO for node 2 on the same
    TCP connection.
  - A second registered node sent a frame to `DestNodeID: 1`.
  - The first connection, whose latest HELLO was node 2, still received the
    frame addressed to node 1.
  - `cli/internal/actornet/broker.go` overwrites the local `registeredNode`
    variable on each HELLO but does not unregister the previous node mapping for
    the same connection.
- Why it matters: a single client can occupy multiple node IDs, receive traffic
  for stale identities, and leave stale map entries after disconnect. This
  compounds BUG-003's source-spoofing issue and can break node availability and
  routing isolation.

### BUG-007 - `eco unpack` follows pre-existing symlinks in `-C` and writes outside the output directory

- Area: Eco package unpack/materialization path safety.
- Severity: high; archive extraction can overwrite/create files outside the
  requested destination when the destination tree already contains a symlink.
- Reproducer: `/tmp/tetra-bug-hunt/session-002/project-t4/Tetra.capsule`
- Commands:

```sh
go run ./cli/cmd/tetra eco pack --project /tmp/tetra-bug-hunt/session-002/project-t4/Tetra.capsule -o /tmp/tetra-bug-hunt/session-002/demo-t4.todex
ln -s /tmp/tetra-bug-hunt/session-002/escape-t4 /tmp/tetra-bug-hunt/session-002/out-symlink-t4/src
go run ./cli/cmd/tetra eco unpack /tmp/tetra-bug-hunt/session-002/demo-t4.todex -C /tmp/tetra-bug-hunt/session-002/out-symlink-t4
find /tmp/tetra-bug-hunt/session-002/escape-t4 -maxdepth 3 -printf '%y %p -> %l\n'
```

- Observed:
  - `eco unpack` exited 0 and printed
    `Unpacked: /tmp/tetra-bug-hunt/session-002/out-symlink-t4`.
  - The destination still had `out-symlink-t4/src` as a symlink to
    `/tmp/tetra-bug-hunt/session-002/escape-t4`.
  - The package payload was written to
    `/tmp/tetra-bug-hunt/session-002/escape-t4/main.t4`, outside the requested
    `-C` directory tree.
- Expected: unpack should reject symlinked path components in an existing output
  tree, remove/replace them only under an explicit safe policy, or open files
  with no-follow/inside-root guarantees.
- Evidence path:
  - `cli/cmd/tetra/tetra_eco.go` normalizes archive entry names in
    `unpackCapsule`, then writes `filepath.Join(outDir, name)`.
  - The write path uses `os.MkdirAll(filepath.Dir(outPath), 0o755)` followed by
    `os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)`, which
    follows pre-existing symlink path components.
  - `go run ./tools/cmd/validate-eco-unpack --dir .../out-symlink-t4` later
    fails with `missing T4 sources under src`, but the write outside `-C` has
    already happened.
- Why it matters: any workflow that unpacks or materializes a trusted package
  into a reusable workspace can be turned into an arbitrary file write within
  the privileges of the CLI process if an attacker can pre-place a symlink.

### BUG-008 - Capsule source-root validation silently ignores unsafe roots instead of rejecting them

- Area: Capsule project loading and unpack validation.
- Severity: medium; manifest validation can pass a package that declares unsafe
  source roots, making source-root policy and audit output misleading.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-002/project-only-unsafe-source/Tetra.capsule`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-002/project-only-unsafe-source/Tetra.capsule
go run ./cli/cmd/tetra eco pack --project /tmp/tetra-bug-hunt/session-002/project-only-unsafe-source/Tetra.capsule -o /tmp/tetra-bug-hunt/session-002/only-unsafe-source.todex
go run ./cli/cmd/tetra eco unpack /tmp/tetra-bug-hunt/session-002/only-unsafe-source.todex -C /tmp/tetra-bug-hunt/session-002/out-only-unsafe-source
go run ./tools/cmd/validate-eco-unpack --dir /tmp/tetra-bug-hunt/session-002/out-only-unsafe-source
```

- Observed:
  - The Capsule declares only `source "../outside"` plus `entry "src/main.t4"`.
  - `tetra check` exited 0 and printed
    `Checked: /tmp/tetra-bug-hunt/session-002/project-only-unsafe-source/src/main.t4`.
  - `eco pack`, `eco unpack`, and `validate-eco-unpack` all exited 0.
- Expected: `source "../outside"` should be rejected as an unsafe source root
  in both project loading and post-unpack validation. A validator should not
  silently discard the only declared source root and then validate a default
  root instead.
- Evidence path:
  - `compiler/internal/module/loader.go` `cleanSourceRoots` skips roots where
    `strings.HasPrefix(root, "..") || filepath.IsAbs(root)`.
  - `tools/cmd/validate-eco-unpack/main.go` `appendUnpackSourceRoot` similarly
    returns the previous list for `..`, `../*`, empty, or absolute roots.
  - `validateEcoUnpack` then defaults an empty source-root list to `src`, so an
    unsafe-only manifest can still pass validation if `src/main.t4` exists.
- Why it matters: malicious or malformed packages can preserve a misleading
  manifest while validators report success. Downstream tooling that relies on
  source roots for audit, dependency scanning, or sandbox policy can be tricked
  into auditing a different root set than the manifest actually declared.

### BUG-009 - `lib.core.time` duration helpers overflow into negative or zero values for positive inputs

- Area: stable stdlib time policy helpers.
- Severity: medium; timeout and deadline arithmetic can wrap across the
  non-negative duration boundary.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-003/bughunt/time_millis_positive_overflow.tetra`
  - `/tmp/tetra-bug-hunt/session-003/bughunt/time_add_positive_overflow_zero.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-003/bughunt/time_millis_positive_overflow.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-003/bughunt/time_add_positive_overflow_zero.tetra
```

- Observed:
  - `time.millis_from_seconds(2147484)` produced a negative value; the repro
    printed `exit status 42`.
  - `time.add_duration_ms(2147483640, 10)` returned `0`; the repro printed
    `exit status 42`.
- Expected: positive duration arithmetic should not wrap across the
  non-negative duration boundary. These helpers should saturate, reject
  overflowing inputs, or document overflow behavior explicitly.
- Evidence path:
  - `docs/spec/standard_library/stdlib.md` labels `lib.core.time` as stable duration arithmetic
    and documents negative-input/result clamping.
  - `docs/user/platform/standard_library_guide.md` says `millis_from_seconds` converts
    seconds by multiplying by `1000` and negative seconds clamp to `0`; it says
    `add_duration_ms` returns `0` if the result would be negative.
  - `lib/core/base/time.tetra` returns `seconds * 1000` directly after checking only
    the input sign.
  - `lib/core/base/time.tetra` computes `let next: Int = base + delta` and treats an
    overflowed negative `next` as a real negative result.
- Why it matters: microservices commonly compute retry, timeout, lease, and
  deadline durations from configuration. A large but positive value can become
  negative or zero, turning a long wait into an immediate timeout or busy retry.

### BUG-010 - Heap slice constructors reject zero-length slices despite empty-slice contracts

- Area: native Linux-x64 heap slice allocation (`core.make_i32`,
  `core.make_u8`, `core.make_u16`, `core.make_bool`).
- Severity: medium; empty payloads and zero-result collection scans cannot be
  represented with the normal heap slice constructors.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-004/bughunt/collections_empty_make_i32.tetra`
  - `/tmp/tetra-bug-hunt/session-004/bughunt/slices_empty_make_u8.tetra`
  - `/tmp/tetra-bug-hunt/session-010/bughunt/slice_u16_empty_heap.tetra`
  - `/tmp/tetra-bug-hunt/session-010/bughunt/slice_bool_empty_heap.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-004/bughunt/collections_empty_make_i32.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-004/bughunt/slices_empty_make_u8.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-010/bughunt/slice_u16_empty_heap.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-010/bughunt/slice_bool_empty_heap.tetra
```

- Observed:
  - All listed heap zero-length programs pass `tetra check`.
  - All listed heap zero-length `run` commands print `exit status 2` before the
    program can use or return the empty slice.
  - The same empty `[]u8` scan works when the slice is created with
    `core.island_make_u8(isl, 0)`.
  - Session 010 also confirmed `core.island_make_u16(isl, 0)` and
    `core.island_make_bool(isl, 0)` return normally, so the regression is on
    the heap constructor path rather than every slice constructor.
- Expected: `core.make_i32(0)`, `core.make_u8(0)`, `core.make_u16(0)`, and
  `core.make_bool(0)` should create valid empty slices, or the language/runtime
  docs should state that heap slice length must be positive and provide another
  way to construct empty slices.
- Evidence path:
  - `docs/spec/standard_library/stdlib.md` says `collections.first_or_i32` returns the fallback
    for an empty slice.
  - `docs/user/platform/standard_library_guide.md` says `collections.first_or_i32`
    returns the fallback when the slice is empty and says a byte checksum over
    an empty slice returns `0`.
  - `compiler/internal/backend/x64abi/sysv_unix.go` `EmitMakeSlice` forwards
    the requested byte length directly to `mmap`.
  - Linux `mmap` rejects length `0`, and `emitMmapFailureGuard` converts that
    allocator failure into process exit code `2`.
- Why it matters: microservices often represent empty request bodies, empty
  recipient lists, or cache misses as empty slices. The documented empty-scan
  behavior is unreachable through the ordinary heap constructors.

### BUG-011 - Heap slice constructors accept negative lengths as huge slices

- Area: native Linux-x64 heap slice allocation and bounds metadata
  (`core.make_u8`, `core.make_i32`, `core.make_u16`, `core.make_bool`).
- Severity: high; allocation precondition and slice bounds bypass for
  attacker-controlled lengths.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-004/bughunt/heap_negative_make_u8.tetra`
  - `/tmp/tetra-bug-hunt/session-004/bughunt/heap_negative_make_i32.tetra`
  - `/tmp/tetra-bug-hunt/session-010/bughunt/slice_u16_negative_heap_store.tetra`
  - `/tmp/tetra-bug-hunt/session-010/bughunt/slice_bool_negative_heap_store.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-004/bughunt/heap_negative_make_u8.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-004/bughunt/heap_negative_make_i32.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-010/bughunt/slice_u16_negative_heap_store.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-010/bughunt/slice_bool_negative_heap_store.tetra
```

- Observed:
  - All listed negative-length programs pass `tetra check`.
  - All listed `run` commands print `exit status 42`.
  - Each repro calls `core.make_*(0 - 1)`, writes element `0`, reads it back,
    and reaches the success sentinel.
- Expected: negative slice lengths should be rejected before any host
  allocation request or slice metadata is returned to Tetra code.
- Evidence path:
  - `compiler/internal/backend/x64abi/sysv_unix.go` `EmitMakeSlice` pops the
    requested length, keeps it as the returned slice length, shifts only for
    element width, and passes the resulting byte count to `mmap`.
  - There is no explicit `len < 0` guard before the syscall.
  - On this Linux environment, the huge overcommitted mapping succeeds, so the
    returned slice carries a huge effective length and index `0` passes bounds
    checks.
- Why it matters: any service that builds slices from decoded request sizes can
  turn `-1` into a massive virtual allocation and a slice whose bounds do not
  reflect the caller's intended validation. This can mask input validation
  failures and expose denial-of-service or memory safety edges.

### BUG-012 - Runtime logical deadlines overflow into negative absolute times

- Area: runtime logical clock ABI (`core.deadline_ms`, `core.timer_ready`).
- Severity: medium to high; long service deadlines can become immediately ready.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-005/bughunt/runtime_deadline_positive_overflow.tetra`
  - `/tmp/tetra-bug-hunt/session-005/bughunt/runtime_timer_ready_overflow_deadline.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-005/bughunt/runtime_deadline_positive_overflow.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-005/bughunt/runtime_deadline_positive_overflow.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-005/bughunt/runtime_timer_ready_overflow_deadline.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-005/bughunt/runtime_timer_ready_overflow_deadline.tetra
```

- Observed:
  - All four commands printed `exit status 42`.
  - After advancing logical time to `10`, `core.deadline_ms(2147483640)`
    returned a negative absolute deadline.
  - `core.timer_ready(deadline)` treated that overflowed deadline as ready
    immediately.
- Expected: `deadline_ms` should not wrap a positive relative delay into a
  negative absolute deadline. It should saturate, reject overflow, or document a
  bounded deadline range and return an error/status for out-of-range values.
- Evidence path:
  - `docs/spec/runtime/runtime_abi.md` says `deadline_ms` returns
    `now + max(delta, 0)` and `timer_ready` checks an absolute deadline.
  - `compiler/internal/actorsrt/actorsrt_core.go` `emitDeadlineMs` clamps the
    delta to non-negative and then performs `schedTimeMs + delta` with a plain
    32-bit add.
  - The same repro triggers under `-runtime builtin` and `-runtime selfhost`.
- Why it matters: a microservice using logical deadlines for joins, receives,
  or timeouts can turn a very long timeout into an already-expired deadline,
  bypassing intended waiting/cancellation windows.

### BUG-013 - Builtin runtime `sleep_ms` overflow can terminate the process with success before `main` returns

- Area: builtin runtime scheduler sleep handling.
- Severity: high; a positive service sleep can silently skip remaining code and
  report process success.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-005/bughunt/runtime_sleep_overflow_exits_zero.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-005/bughunt/runtime_sleep_overflow_exits_zero.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-005/bughunt/runtime_sleep_overflow_exits_zero.tetra
```

- Observed:
  - Under `-runtime builtin`, the command exited 0 with no `exit status ...`
    output, even though the program's only explicit post-sleep return is `42`.
  - Under `-runtime selfhost`, the same program printed `exit status 42`.
- Expected: `core.sleep_ms(2147483640)` after logical time `10` should not
  sleep the current task on an overflowed wake deadline and let the scheduler
  end the process as success before control returns to Tetra code.
- Evidence path:
  - The repro first calls `core.sleep_ms(10)`, then
    `core.sleep_ms(2147483640)`, then `return 42`.
  - `compiler/internal/actorsrt/actorsrt_core.go` `emitSleepMs` clamps only
    negative `ms`, computes `schedTimeMs + ms` with a plain 32-bit add, stores
    that as the actor wake deadline, marks the actor sleeping, and yields.
  - The builtin scheduler does not resume the only task after that overflowed
    wake deadline, so the process exits successfully without reaching
    `return 42`.
- Why it matters: an attacker-controlled timeout or retry delay can skip
  cleanup, status reporting, or authorization code after a sleep while the
  process reports success. This is worse than a normal timeout error because it
  is silent.

### BUG-014 - Public brace literals can forge `task.i32` handles accepted by task poll/join runtimes

- Area: task handle opacity and runtime actor-handle validation.
- Severity: high; user code can manufacture task handles that were never
  returned by `core.task_spawn_i32`.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_poll_fake_nonzero.tetra`
  - `/tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_join_until_fake_nonzero.tetra`
  - `/tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_error_short_circuit.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_poll_fake_nonzero.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_poll_fake_nonzero.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_poll_fake_nonzero.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_join_until_fake_nonzero.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_join_until_fake_nonzero.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_error_short_circuit.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_brace_literal_error_short_circuit.tetra
```

- Observed:
  - `check` accepted `let fake: task.i32 = task.i32{value: 999, error: 0}`.
  - Builtin runtime runs of fake `poll` and `join_until` printed
    `exit status 255`.
  - Selfhost runtime runs of fake `poll` and `join_until` exited 0 with no
    `exit status ...`, treating the forged nonzero handle as a real task path.
  - The `error: 1` control repro printed `exit status 42` under both runtimes,
    confirming the invalid actor slot is trusted only when the task error slot
    is zero.
- Expected: `task.i32` should be opaque/unforgeable from ordinary Tetra code, or
  every task poll/join runtime entry should validate the actor handle range and
  lifecycle before reading actor state.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` defines `task.i32` as a public struct
    with public `value: i32` and `error: task.error` fields.
  - `compiler/internal/actorsrt/actorsrt_core.go` `emitTaskPollI32`,
    `emitTaskJoinI32`, and `emitTaskJoinUntilI32` trust the first slot as an
    actor index after checking only the error slot.
  - `compiler/selfhostrt/actors_sysv.tetra` maps any nonzero actor index through
    `actor_status`/`actor_slot` to the pong actor state, so fake handle `999`
    aliases that runtime state.
- Why it matters: task handles are resource-like values with ownership
  diagnostics, but direct struct construction bypasses provenance entirely.
  Services can accidentally or maliciously poll/join arbitrary actor slots,
  producing runtime divergence and exposing scheduler state assumptions.

### BUG-015 - Call-style dotted built-in struct constructors pass `check` but fail `build`/`run`

- Area: checker/build pipeline consistency for built-in dotted struct
  constructor calls.
- Severity: medium; `tetra check` can approve source that the same CLI cannot
  build or run.
- Reproducers:
  `/tmp/tetra-bug-hunt/session-006/bughunt/task_handle_struct_literal_error_short_circuit.tetra`
  and `/tmp/tetra-bug-hunt/session-027/bughunt/{actor_msg_call_constructor.tetra,actor_recv_result_call_constructor.tetra,task_result_call_constructor.tetra}`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_struct_literal_error_short_circuit.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-006/bughunt/error_short_circuit.bin /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_struct_literal_error_short_circuit.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-006/bughunt/task_handle_struct_literal_error_short_circuit.tetra
```

- Observed:
  - `check` printed `Checked: .../task_handle_struct_literal_error_short_circuit.tetra`.
  - `build` and `run` failed with `missing signature for 'task.i32'` and
    `exit status 1`.
  - Session 027 confirmed the same check/run split for other built-in dotted
    structs: `actor.msg(...)`, `actor.recv_result_i32(...)`, and
    `task.result_i32(...)` all pass `check` and fail `run` with
    `missing signature for '<type>'`.
  - Nearby brace-style `actor.msg { value: 20, tag: 22 }` ran successfully and
    printed `exit status 42`, so the failure is specific to call-style
    constructor lowering/dependency handling.
- Expected: either `check` should reject call-style construction of
  built-in dotted structs, or build/run dependency collection should
  honor the checker's constructor resolution and not require a function
  signature for the struct type name.
- Evidence path:
  - `compiler/internal/semantics/semantics_expressions.go`
    `checkStructConstructorCallWithEffects` accepts labeled calls whose name
    resolves to a struct type and rewrites the call to the resolved type.
  - The build/cache path still reports `task.i32` as a missing call signature,
    so a source file can pass semantic checking and fail before codegen.
- Why it matters: this breaks the usual `check` as a reliable preflight for
  build/run in exactly the area that carries task resource handles. It also
  hides the forged-handle issue behind a separate build-stage failure for the
  modern call-style struct literal syntax.

### BUG-016 - `-runtime selfhost` lacks public task-group runtime symbols

- Area: selfhost runtime parity for public `core.task_group_*` builtins.
- Severity: medium; task-group programs pass `check` and run under builtin, but
  fail under an advertised runtime mode.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-006/bughunt/task_group_selfhost_missing_symbol.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-006/bughunt/task_group_selfhost_missing_symbol.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-006/bughunt/task_group_selfhost_missing_symbol.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-006/bughunt/task_group_selfhost_missing_symbol.tetra
```

- Observed:
  - `check` succeeded.
  - Builtin runtime printed `exit status 42` from the success sentinel.
  - Selfhost runtime failed with
    `runtime object missing required symbol '__tetra_task_group_open'` and
    `exit status 1`.
- Expected: `-runtime selfhost` should either implement the checked
  `core.task_group_*` ABI surface or reject/diagnose task-group use before
  trying to link a runtime object missing required symbols.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` exposes `core.task_group_open`,
    `core.task_group_close`, `core.task_group_cancel`,
    `core.task_group_current`, and `core.task_group_status`.
  - `compiler/selfhostrt/actors_sysv.tetra` implements task spawn/join/time
    helpers but has no exported `__tetra_task_group_*` functions.
- Why it matters: a service that selects the selfhost runtime for parity testing
  or deployment can pass normal semantic checks and then fail at runtime-object
  validation for a public task-group API.

### BUG-017 - Raw actor `send_msg` can spoof `recv_typed` enum messages, including resource payload cases

- Area: actor typed-message runtime envelope (`core.send_msg`,
  `core.recv_typed`, `core.send_typed`).
- Severity: high; untyped actor sends bypass typed actor message validation and
  can materialize typed enum cases that the sender could not construct via
  `send_typed`.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_inc.tetra`
  - `/tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_reset.tetra`
  - `/tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_island_payload.tetra`
  - `/tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_invalid_typed_tag.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_inc.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_inc.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_inc.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_island_payload.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_spoofs_typed_island_payload.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_invalid_typed_tag.tetra
go run ./cli/cmd/tetra run -runtime selfhost /tmp/tetra-bug-hunt/session-007/bughunt/actor_raw_send_msg_invalid_typed_tag.tetra
```

- Observed:
  - All reproducers pass `check`.
  - Baseline typed send of `CounterMsg.inc(20, 22)` returns `exit status 42`
    under both builtin and selfhost.
  - Raw `core.send_msg(peer, 7, 0)` is received by
    `core.recv_typed<CounterMsg>()` as `CounterMsg.inc(7, 0)`; both runtimes
    print `exit status 42`.
  - Raw `core.send_msg(peer, 123, 1)` is received as the zero-payload
    `CounterMsg.reset` case; both runtimes print `exit status 42`.
  - Raw `core.send_msg(peer, 0, 0)` is received as `MoveMsg.take(island)`;
    both runtimes print `exit status 42`, proving a typed enum case carrying an
    `island` payload can be materialized from a raw integer send.
  - Raw tag `99`, which is not a valid `CounterMsg` case, still enters
    `recv_typed<CounterMsg>()`; the worker falls through the `match` and both
    runtimes print `exit status 42`.
- Expected: `recv_typed<Enum>()` should only accept messages sent through the
  typed message path with a valid enum tag and payload slot count for `Enum`, or
  it should return/throw a typed-message decode error instead of manufacturing
  an enum value from raw `send_msg` frames.
- Evidence path:
  - `compiler/internal/lower/lower_core.go` lowers `core.recv_typed` by calling
    `__tetra_actor_recv_begin` for the tag and then blindly reading
    `Enum.SlotCount-1` payload slots via `__tetra_actor_recv_slot`.
  - `compiler/internal/actorsrt/actorsrt_core.go` `emitRecvBegin` stores the
    message as `schedPendingMsg` and returns `msgTag`; `emitRecvSlot` reads
    payload slots without checking `msgCount`.
  - `core.send_msg` creates the same mailbox message shape with caller-chosen
    tag and only a raw integer payload, so it can impersonate typed enum tags.
- Why it matters: typed actor messages are used to enforce enum shape and
  resource movement. A raw sender can bypass the `send_typed` checker, spoof
  typed control messages, trigger impossible enum tags, and create resource
  payload cases such as `MoveMsg.take(island)` without owning an island.

### BUG-018 - `secret.i32{}` passes `check` but fails `build` with a slot mismatch

- Area: checker/lowerer consistency for built-in secret wrapper type.
- Severity: medium; `tetra check` accepts a privacy-sensitive value literal
  that cannot be lowered.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-007/bughunt/secret_empty_brace_literal_check_build_split.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-007/bughunt/secret_empty_brace_literal_check_build_split.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-007/bughunt/secret_empty.bin /tmp/tetra-bug-hunt/session-007/bughunt/secret_empty_brace_literal_check_build_split.tetra
go run ./cli/cmd/tetra run -runtime builtin /tmp/tetra-bug-hunt/session-007/bughunt/secret_empty_brace_literal_check_build_split.tetra
```

- Observed:
  - `check` printed `Checked: .../secret_empty_brace_literal_check_build_split.tetra`.
  - `build` and `run` failed with
    `.../secret_empty_brace_literal_check_build_split.tetra:11:5: slot mismatch for 'fake'`
    and `exit status 1`.
- Expected: `secret.i32{}` should be rejected during semantic checking, or the
  type definition/lowering should agree on how a 1-slot secret wrapper with no
  public fields is constructed.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` defines `secret.i32` as a public
    `TypeStruct` with `SlotCount: 1` but no fields.
  - `compiler/internal/semantics/semantics_expressions.go` accepts a brace struct literal when
    all declared fields are present; for `secret.i32`, there are zero declared
    fields.
  - `compiler/internal/lower/lower_core.go` then lowers the literal to zero slots and
    rejects the `let fake: secret.i32 = secret.i32{}` assignment as a slot
    mismatch.
- Why it matters: privacy/secret wrappers are part of the safety surface, and
  `check` should be a reliable preflight for privacy code. This also exposes
  that `secret.i32` is modeled as a constructible public struct even though its
  only valid creator should be `core.secret_seal_i32`.

### BUG-019 - Runtime integer divide/modulo edge cases crash native programs with SIGFPE

- Area: native x64 arithmetic lowering for `IRDivI32` and `IRModI32`.
- Severity: high; an unchecked service denominator can abort the whole process.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.tetra`
  - `/tmp/tetra-bug-hunt/session-008/bughunt/runtime_mod_zero.tetra`
  - `/tmp/tetra-bug-hunt/session-008/bughunt/runtime_min_div_neg_one.tetra`
  - `/tmp/tetra-bug-hunt/session-008/bughunt/runtime_min_mod_neg_one.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.bin /tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.tetra
timeout 5s /tmp/tetra-bug-hunt/session-008/bughunt/runtime_div_zero.bin
```

- Observed:
  - Runtime `42 / denom()` and `42 % denom()` pass `check` when `denom()`
    returns `0`.
  - `tetra run` prints `exit status 255` for the runtime division, modulo, and
    `minInt / -1` / `minInt % -1` reproducers.
  - Directly executing the built native binaries exits with code `136` and
    prints `timeout: the monitored command dumped core`, confirming SIGFPE
    rather than a controlled Tetra diagnostic.
  - As a control, `const bad: Int = 42 / 0` is rejected with
    `division by zero in global const expression`.
- Expected: generated native code should guard runtime division/modulo by zero
  and the signed `minInt / -1` overflow edge, returning a deterministic Tetra
  runtime diagnostic/status instead of letting the process receive SIGFPE. If
  unchecked arithmetic is intentional, this process-abort behavior needs to be
  explicitly documented as part of the language/runtime contract.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` only diagnoses division/modulo by
    zero inside global constant expressions.
  - `compiler/internal/backend/x64core/x64core_core.go` lowers both `IRDivI32` and
    `IRModI32` to `cdq; idiv ecx` without checking whether `ecx == 0` or
    whether the operands are `-2147483648 / -1`.
  - The WASI/web backends similarly emit raw `i32.div_s` / `i32.rem_s` opcodes,
    which indicates no cross-backend arithmetic guard policy is modeled at IR
    lowering time.
- Why it matters: request handlers commonly divide by parsed limits, rates, or
  bucket sizes. A zero or `-1` edge value can convert a checked Tetra program
  into a native process crash instead of an application-level error path.

### BUG-020 - Out-of-range positive integer literals silently wrap to `i32`

- Area: frontend lexer/parser numeric literal handling.
- Severity: high; source constants can silently become negative or otherwise
  attacker-controlled wrap values.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_positive_overflow.tetra`
  - `/tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_large_wrap_minus_one.tetra`
  - `/tmp/tetra-bug-hunt/session-008/bughunt/global_const_i32_positive_overflow.tetra`
  - `/tmp/tetra-bug-hunt/session-059/bughunt/budget_uint32_wrap_zero_repro.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_positive_overflow.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_positive_overflow.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_large_wrap_minus_one.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-008/bughunt/literal_i32_large_wrap_minus_one.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-008/bughunt/global_const_i32_positive_overflow.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-008/bughunt/global_const_i32_positive_overflow.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-059/bughunt/budget_uint32_wrap_zero_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-059/bughunt/budget_uint32_wrap_zero_repro.tetra
```

- Observed:
  - `2147483647` works as the positive max control and returns `exit status 42`.
  - `let x: Int = 2147483648` passes `check` and returns `exit status 42`
    from a branch proving `x < 0`.
  - `let x: Int = 4294967295` passes `check` and returns `exit status 42`
    from a branch proving `x == -1`.
  - `const wrapped: Int = 2147483648` also passes `check` and returns
    `exit status 42` from a branch proving the global const is negative.
  - `budget(4294967296)` passes `check` and `run` prints `exit status 42`,
    showing semantic-clause numeric arguments can also silently wrap rather
    than being rejected as out-of-range positive constants.
- Expected: because `Int`/integer literals default to canonical `i32`, positive
  literals outside `0..2147483647` should be rejected before lowering, or the
  language should specify a wider literal type and require explicit narrowing.
- Evidence path:
  - `compiler/internal/frontend/frontend_core.go` parses number tokens with
    `strconv.ParseInt(..., 64)`.
  - `compiler/internal/frontend/frontend_core.go` stores both normal expression and
    match-pattern numbers as `NumberExpr{Value: int32(tok.num)}`.
  - That unchecked `int64` to `int32` cast is enough to turn `2147483648` into
    `-2147483648` and `4294967295` into `-1` before semantic checking sees the
    expression.
  - `compiler/internal/semantics/semantics_checker.go` validates `budget(...)` through
    the already-wrapped `NumberExpr` value, so `4294967296` arrives as `0` and
    satisfies the non-negative check.
- Why it matters: config-like constants, quota limits, retry thresholds, and
  authorization sentinels can be written as apparently large positive numbers
  but compile as negative or sentinel values. That is a silent source/semantic
  mismatch rather than an explicit overflow diagnostic.

### BUG-021 - `try` error propagation skips active `defer` cleanups

- Area: typed-error lowering and deferred cleanup execution.
- Severity: high; service cleanup/rollback/audit code can be skipped on a
  propagated typed error.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-009/bughunt/defer_try_error_skips_cleanup.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-009/bughunt/defer_try_error_skips_cleanup.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-009/bughunt/defer_try_error_skips_cleanup.bin /tmp/tetra-bug-hunt/session-009/bughunt/defer_try_error_skips_cleanup.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-009/bughunt/defer_try_error_skips_cleanup.tetra
```

- Observed:
  - `check` and `build` succeed.
  - The program returns `exit status 42` from a sentinel branch proving
    `cleaned == 0` after `handler()` propagates `try fail()`.
  - Control reproducers show cleanup does run for an explicit `throw`
    statement and for a successful `try ok()` path:
    `defer_throw_statement_control.tetra` and
    `defer_try_success_control.tetra` both return `exit status 42` from
    branches proving `cleaned == 1`.
- Expected: active `defer` cleanups should run before any function exit caused
  by typed-error propagation, just as they run before explicit `throw` and
  `return`.
- Evidence path:
  - `compiler/internal/lower/lower_core.go` return/throw statement lowering calls
    `emitDeferredFramesSince(0, ...)` before cleanup and `IRReturn`.
  - `TryExpr` error-path lowering emits the converted error status and then
    calls `emitCleanup(...)` plus `IRReturn`, but does not emit active defer
    frames first.
- Why it matters: request handlers often use `defer` for releasing reservations,
  closing resources, rollback flags, or audit markers. A typed error returned
  through `try` can bypass that cleanup while still being a normal checked
  language feature.

### BUG-022 - Propagating `try` inside a `defer` body passes `check` but fails IR verification

- Area: semantic control-flow validation for defer bodies versus lowering.
- Severity: medium; `tetra check` accepts cleanup code that cannot be built.
- Reproducer:
  `/tmp/tetra-bug-hunt/session-009/bughunt/defer_body_try_can_throw.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-009/bughunt/defer_body_try_can_throw.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-009/bughunt/defer_body_try_can_throw.bin /tmp/tetra-bug-hunt/session-009/bughunt/defer_body_try_can_throw.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-009/bughunt/defer_body_try_can_throw.tetra
```

- Observed:
  - `check` succeeds.
  - `build` and `run` fail with
    `ir verifier: handler instr 7: return expects 2 stack slots, have 4`.
  - The same semantic guard does reject an explicit `throw` statement inside a
    `defer` body, so the gap is specifically propagating `try` expression
    control flow.
- Expected: a `defer` body should reject propagating `try` expressions during
  semantic checking, or lowering should model them in a way that preserves
  cleanup/control-flow invariants and passes IR verification.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `validateDeferBodyControl`
    rejects `ReturnStmt`, `ThrowStmt`, nested `DeferStmt`, and nonlocal
    `break`/`continue`, but it does not inspect expressions for `TryExpr`.
  - `compiler/internal/lower/lower_core.go` lowers `TryExpr` error paths by emitting
    an `IRReturn`; when that expression appears inside a replayed defer body,
    the generated stack shape is inconsistent and the verifier rejects it.
- Why it matters: cleanup blocks should have constrained, predictable control
  flow. Accepting a cleanup-local `try` makes `check` unreliable and exposes a
  compiler-stage crash instead of a stable diagnostic.

### BUG-023 - Small `u8`/`u16` boundaries accept out-of-range integers

- Area: numeric subtype/range validation for local values, function
  returns/arguments, typed-error throws, struct fields, enum payloads, and
  slice/raw-memory byte stores.
- Severity: high; byte/word payload types do not enforce their advertised
  ranges at service boundaries.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-011/bughunt/local_u8_out_of_range_256.tetra`
  - `/tmp/tetra-bug-hunt/session-011/bughunt/local_u8_negative_literal.tetra`
  - `/tmp/tetra-bug-hunt/session-011/bughunt/local_u16_out_of_range_70000.tetra`
  - `/tmp/tetra-bug-hunt/session-011/bughunt/store_u8_out_of_range_truncates.tetra`
  - `/tmp/tetra-bug-hunt/session-011/bughunt/slice_u8_store_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-012/bughunt/arg_u8_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-012/bughunt/arg_u16_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-012/bughunt/slice_u16_store_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-016/bughunt/enum_u8_payload_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-016/bughunt/enum_u16_payload_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-018/bughunt/return_u8_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-018/bughunt/return_u16_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_constructor_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_call_constructor_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-019/bughunt/struct_u16_constructor_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_field_assignment_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-020/bughunt/throw_u8_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-020/bughunt/throw_u16_out_of_range.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-011/bughunt/local_u8_out_of_range_256.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-011/bughunt/local_u8_out_of_range_256.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-011/bughunt/local_u16_out_of_range_70000.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-011/bughunt/local_u16_out_of_range_70000.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-011/bughunt/store_u8_out_of_range_truncates.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-011/bughunt/store_u8_out_of_range_truncates.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-011/bughunt/slice_u8_store_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-011/bughunt/slice_u8_store_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/arg_u8_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/arg_u8_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/arg_u16_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/arg_u16_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/slice_u16_store_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/slice_u16_store_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-016/bughunt/enum_u8_payload_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-016/bughunt/enum_u8_payload_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-016/bughunt/enum_u16_payload_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-016/bughunt/enum_u16_payload_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-018/bughunt/return_u8_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-018/bughunt/return_u8_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-018/bughunt/return_u16_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-018/bughunt/return_u16_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_constructor_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_constructor_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_call_constructor_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_call_constructor_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-019/bughunt/struct_u16_constructor_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-019/bughunt/struct_u16_constructor_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_field_assignment_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-019/bughunt/struct_u8_field_assignment_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-020/bughunt/throw_u8_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-020/bughunt/throw_u8_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-020/bughunt/throw_u16_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-020/bughunt/throw_u16_out_of_range.tetra
```

- Observed:
  - `let b: u8 = 256` passes `check` and returns `exit status 42` from a
    branch proving `b == 256`.
  - `let b: u8 = 0 - 1` passes `check` and returns `exit status 42` from a
    branch proving `b < 0`.
  - `let w: u16 = 70000` passes `check` and returns `exit status 42` from a
    branch proving `w == 70000`.
  - `core.store_u8(p, 300, mem)` passes `check`; reading the byte returns `44`,
    and the repro returns `exit status 42`.
  - `xs[0] = 300` for `[]u8` passes `check`; reading `xs[0]` returns `44`, and
    the repro returns `exit status 42`.
  - Passing `256` to a `u8` parameter and `70000` to a `u16` parameter both
    pass `check`; each repro returns `exit status 42` from a branch proving the
    callee received the out-of-range value unchanged.
  - `xs[0] = 70000` for `[]u16` passes `check`; reading `xs[0]` returns `4464`,
    and the repro returns `exit status 42`.
  - `Packet.byte(300)` for enum payload `u8` and `Packet.word(70000)` for enum
    payload `u16` both pass `check`; matching the enum binds payload values
    `300` and `70000` unchanged, and both repros return `exit status 42`.
  - Returning `300` from a `-> UInt8` function and `70000` from a `-> UInt16`
    function both pass `check`; callers that bind the return value as
    `UInt8`/`UInt16` see the full out-of-range values unchanged, and both
    repros return `exit status 42`.
  - `Header{byte: 300}` and call-style `Header(byte: 300)` both pass `check`
    for a `UInt8` struct field; reading `h.byte` returns the full invalid
    value `300`, and both repros return `exit status 42`.
  - `Header{word: 70000}` similarly passes `check` for a `UInt16` struct field
    and reads back `70000`.
  - Assigning `h.byte = 300` to a mutable struct value passes `check`, reads
    back `300`, and returns `exit status 42`.
  - Throwing `300` from a `throws UInt8` function and `70000` from a
    `throws UInt16` function both pass `check`; `catch` literal patterns
    `case 300` and `case 70000` match those invalid error values at runtime,
    and both repros return `exit status 42`.
  - Max-value controls for `255` and `65535` also pass and return
    `exit status 42`.
- Expected: assigning or passing a value to `u8` should reject values outside
  `0..255`, and assigning or passing a value to `u16` should reject values
  outside `0..65535`, unless the language explicitly requires a checked or
  explicit wrapping conversion.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` has `validateGlobalIntLikeRange`
    for global `u8`/`u16` initializers, but the local declaration path checks
    only `typesCompatibleWithNullPtr`.
  - `compiler/internal/semantics/semantics_core.go` treats any `isInt32Like` expected
    and actual type as compatible, so `i32` literals are accepted for `u8` and
    `u16` without range validation.
  - `compiler/internal/semantics/semantics_expressions.go` validates enum case constructor
    payloads with the same `typesCompatibleWithNullPtr` compatibility check,
    so enum/message payload constructors inherit the missing `u8`/`u16` range
    validation.
  - `compiler/internal/semantics/semantics_checker.go` validates `ReturnStmt` values with
    `typesCompatibleWithNullPtr(returnType, tname, s.Value)`, so `i32`
    expressions are accepted for `u8`/`u16` returns without range validation.
  - `compiler/internal/semantics/semantics_expressions.go` validates both brace-style struct
    literals and call-style labeled struct constructors with
    `typesCompatibleWithNullPtr(field.TypeName, valType, value)`.
  - `compiler/internal/semantics/semantics_checker.go` validates struct field assignment
    through `resolveAssignTarget` followed by
    `typesCompatibleWithNullPtr(targetType, valType, s.Value)`.
  - `compiler/internal/semantics/semantics_checker.go` validates `ThrowStmt` values with
    `typesCompatibleWithNullPtr(state.throwType, tname, s.Value)`, so typed
    error codes declared as `u8`/`u16` inherit the same range hole.
  - `compiler/internal/lower/lower_core.go` lowers `core.store_u8` directly to
    `IRMemWriteU8`, and x64 emission stores only the low byte.
  - `[]u8` index assignment similarly lowers to `IRIndexStoreU8`; the x64
    emitter writes `r8b`, silently truncating the out-of-range value.
- Why it matters: microservices routinely decode byte fields, opcodes, ports,
  protocol versions, and packed payload bytes into small integer types. Tetra's
  checker currently lets invalid values cross the typed boundary, and different
  storage paths disagree between keeping the invalid full integer and silently
  truncating it.

### BUG-024 - Zeroed fixed-array fields compile but trap on any index access

- Area: fixed-array runtime representation for zeroed globals and struct
  fields.
- Severity: high; a documented/build-tested fixed-array field shape is accepted
  by `check` but unusable at runtime.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_read_traps.tetra`
  - `/tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_store.tetra`
  - `/tmp/tetra-bug-hunt/session-012/bughunt/array_int_local_param_store.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_read_traps.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_read_traps.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_store.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/array_int_global_field_store.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-012/bughunt/array_int_local_param_store.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-012/bughunt/array_int_local_param_store.tetra
```

- Observed:
  - `struct IntBox: items: [1]Int` plus `var box: IntBox` passes `check`.
  - Reading `box.items[0]` immediately prints `exit status 1`.
  - Assigning `box.items[0] = 42` also prints `exit status 1`, before the
    program can read the value back.
  - Passing `box.items` into a function as `[1]Int`, assigning a local copy, and
    reading `xs[0]` also prints `exit status 1`.
- Expected: a `[1]Int` field should have length one and usable backing storage
  after zero initialization, so index `0` should be valid and should initially
  read the element default value.
- Evidence path:
  - `compiler/tests/semantics/semantics_core_language_test.go` has a build smoke test named
    `TestArrayMVPBuildSupportsZeroedFixedArrayFieldGlobal`, so this surface is
    intentionally accepted.
  - `compiler/internal/semantics/semantics_core.go` models `TypeArray` with slice-like
    `ptr` and `len` fields and `SlotCount: 2`.
  - `compiler/internal/lower/lower_core.go` lowers both fixed arrays and slices
    through the same `IRIndexLoad*` / `IRIndexStore*` path, which expects a
    runtime pointer and length on the stack.
  - Zeroed global struct fields do not initialize those runtime `ptr`/`len`
    slots to a valid fixed-array backing store, so index `0` bounds-checks
    against length zero and exits with status `1`.
- Why it matters: fixed arrays are the natural shape for protocol headers,
  small IDs, static slots, and embedded service state. Code that passes
  `check` and build smoke tests cannot safely read or write even index zero,
  which makes the feature unusable for microservice state.

### BUG-025 - Global struct field assignment writes to local slots instead of globals

- Area: lowering for assignment targets resolved through global struct fields.
- Severity: critical; accepted code can either fail IR verification or silently
  corrupt local variables while leaving global service state unchanged.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_ignored.tetra`
  - `/tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_corrupts_local.tetra`
  - `/tmp/tetra-bug-hunt/session-013/bughunt/global_struct_second_field_assignment_corrupts_second_local.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-013/bughunt/local_struct_field_assignment_control.tetra`
  - `/tmp/tetra-bug-hunt/session-013/bughunt/global_struct_whole_assignment_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-013/bughunt/local_struct_field_assignment_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-013/bughunt/local_struct_field_assignment_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_whole_assignment_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_whole_assignment_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_ignored.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_ignored.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_corrupts_local.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_field_assignment_corrupts_local.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_second_field_assignment_corrupts_second_local.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-013/bughunt/global_struct_second_field_assignment_corrupts_second_local.tetra
```

- Observed:
  - Local struct field assignment passes `check` and returns `exit status 42`.
  - Whole-global assignment `box = Box{value: 42}` passes `check` and returns
    `exit status 42`.
  - Direct global field assignment `box.value = 42` passes `check`, but with no
    locals `run` fails IR verification:
    `local slot 0 out of bounds (locals=0)`.
  - Adding a local `marker` makes the same assignment build and run; the repro
    returns `exit status 42` from a branch proving `marker == 42` while
    `box.value == 0`.
  - Assigning the second global field similarly corrupts the second local slot:
    `box.value = 42` with `value` at field offset `1` leaves `first == 11`,
    changes `second == 42`, and leaves `box.value == 0`.
- Expected: assignments to fields of global structs should emit `IRStoreGlobal`
  to the resolved global data slot, just as whole-global assignment already
  does.
- Evidence path:
  - `compiler/internal/lower/lower_core.go` `resolveLValue` returns `Global: true`
    for global field targets and computes the base from `g.DataIndex`.
  - The general `AssignStmt` field-assignment path then ignores `target.Global`
    and always emits `IRStoreLocal` for `target.Base + i`.
  - Whole-global identifier assignment has a separate branch that correctly
    emits `IRStoreGlobal`, which is why the whole-assignment control passes.
- Why it matters: service state often lives in global structs for counters,
  routing tables, cached config, and task/actor handles. A normal-looking field
  update can mutate a local request variable instead of durable global state,
  breaking invariants and potentially leaking data across control paths.

### BUG-026 - Compound index assignment evaluates side-effecting targets twice

- Area: parser/lowering semantics for compound assignments such as `+=` on
  indexed targets.
- Severity: high; bucket/counter updates can touch the wrong element or trap at
  runtime even when the single-evaluation form is valid.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_double_eval.tetra`
  - `/tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_oob.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-014/bughunt/compound_index_no_side_effect_control.tetra`
  - `/tmp/tetra-bug-hunt/session-014/bughunt/compound_index_explicit_temp_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_no_side_effect_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_no_side_effect_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_explicit_temp_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_explicit_temp_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_double_eval.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_double_eval.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_oob.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-014/bughunt/compound_index_side_effect_oob.tetra
```

- Observed:
  - `xs[0] += 2` passes `check` and returns `exit status 42`.
  - The explicit-temp form `idx = next(); xs[idx] = xs[idx] + 2` passes
    `check` and returns `exit status 42`, proving `next()` ran once and updated
    bucket `0` from `40` to `42`.
  - `xs[next()] += 2` passes `check` but returns `exit status 42` from a branch
    proving `next()` ran twice (`cursor == 2`) and stored `xs[1] + 2` into
    `xs[0]`, producing `xs[0] == 102` instead of `42`.
  - The same pattern over a length-one slice passes `check` but `run` prints
    `exit status 1`, because the cloned RHS target evaluates `next()` a second
    time and tries to load `xs[1]`.
- Expected: compound assignment should evaluate the assignment target once, or
  the checker should reject side-effecting compound targets until lowering can
  preserve single-evaluation semantics.
- Evidence path:
  - `compiler/internal/frontend/frontend_core.go` parses `a += b` by constructing a
    `BinaryExpr` whose left side is `cloneCompoundTarget(expr)`.
  - `cloneCompoundTarget` recursively clones `IndexExpr`, including its index
    expression, rather than staging the target result in a temporary.
  - `compiler/internal/lower/lower_core.go` lowers index assignment by evaluating the
    store target base/index first, then evaluating `s.Value`; for compound
    assignments `s.Value` contains the cloned target and therefore evaluates the
    index expression again.
- Why it matters: request routers, rate limiters, sharded counters, and cache
  buckets often use helper functions to choose the next slot. A natural
  `buckets[next()] += 1` can double-advance the cursor, update the wrong bucket,
  or become an unexpected bounds trap.

### BUG-027 - Global const-expression overflow wraps before range checks

- Area: semantic checking of global `const`/`var` initializer expressions.
- Severity: high; global service limits, ports, quotas, and protocol constants
  can silently become small wrapped values while passing `check`.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-017/bughunt/global_u8_wrapped_const_expr.tetra`
  - `/tmp/tetra-bug-hunt/session-017/bughunt/global_u16_wrapped_const_expr.tetra`
  - `/tmp/tetra-bug-hunt/session-017/bughunt/global_i32_wrapped_const_expr.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-017/bughunt/global_u8_direct_out_of_range_control.tetra`
  - `/tmp/tetra-bug-hunt/session-017/bughunt/global_u8_simple_expr_out_of_range_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-017/bughunt/global_u8_direct_out_of_range_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-017/bughunt/global_u8_simple_expr_out_of_range_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-017/bughunt/global_u8_wrapped_const_expr.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-017/bughunt/global_u8_wrapped_const_expr.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-017/bughunt/global_u16_wrapped_const_expr.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-017/bughunt/global_u16_wrapped_const_expr.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-017/bughunt/global_i32_wrapped_const_expr.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-017/bughunt/global_i32_wrapped_const_expr.tetra
```

- Observed:
  - Direct `var b: UInt8 = 256` is rejected with the expected `0..255`
    diagnostic.
  - Simple `var b: UInt8 = 255 + 1` is also rejected, showing ordinary
    out-of-range constant expressions are covered.
  - `var b: UInt8 = 65536 * 65536` passes `check` and `run` returns
    `exit status 42` from a branch proving the stored value is `0`.
  - `var w: UInt16 = 65536 * 65536` behaves the same way and stores `0`.
  - `const wrapped: Int = 65536 * 65536` passes `check` and evaluates to `0`,
    proving the root issue is unchecked `int32` constant folding, not just the
    small-integer range gate.
- Expected: global constant evaluation should detect arithmetic overflow, or
  evaluate in a wider/exact integer domain and only narrow after validating the
  declared target type.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `evalGlobalConstI32` evaluates
    `+`, `-`, `*`, `/`, and `%` directly in `int32`.
  - `validateGlobalIntLikeRange` receives the already-wrapped value, so a
    mathematically out-of-range expression like `65536 * 65536` becomes `0`
    before `u8`/`u16` validation.
- Why it matters: microservice-style globals often encode max payload sizes,
  retry budgets, rate limits, and protocol IDs. An overflowing expression can
  pass review as a large intended value but execute as zero or another wrapped
  value.

### BUG-028 - Optional small-int literal payloads pass check but fail lowering

- Area: checker/lowering contract for optional `UInt8?` and `UInt16?`
  payload initialization/assignment.
- Severity: high; accepted optional byte/word fields in config, protocol, or
  error DTOs fail after `check`, including valid in-range payload literals.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_max_control.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u16_out_of_range.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_assignment_out_of_range.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_typed_payload_control.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u16_typed_payload_control.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_typed_assignment_control.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_i32_literal_control.tetra`
  - `/tmp/tetra-bug-hunt/session-021/bughunt/optional_bool_int_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_max_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_max_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_u16_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_u16_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_assignment_out_of_range.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_assignment_out_of_range.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_typed_payload_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_u8_typed_payload_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_i32_literal_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-021/bughunt/optional_i32_literal_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-021/bughunt/optional_bool_int_rejected_control.tetra
```

- Observed:
  - `let maybe: UInt8? = 255` passes `check`, but `run` fails with
    `slot mismatch for 'maybe'`.
  - `let maybe: UInt8? = 300` and `let maybe: UInt16? = 70000` also pass
    `check`, then fail with the same local slot mismatch.
  - `var maybe: UInt8? = none; maybe = 300` passes `check`, then `run` fails
    with `slot mismatch for assignment`.
  - Typed payload controls work: `let b: UInt8 = 255; let maybe: UInt8? = b`,
    `let w: UInt16 = 65535; let maybe: UInt16? = w`, and typed assignment
    from a `UInt8` local all pass `check` and return `exit status 42`.
  - `let maybe: Int? = 42` passes `check` and returns `exit status 42`,
    showing optional literal wrapping works for exact `i32`.
  - `let maybe: Bool? = 1` is rejected with
    `type mismatch: expected 'bool?', got 'i32'`.
- Expected: either the checker should reject `i32` literals for `UInt8?` and
  `UInt16?` unless a checked conversion exists, or lowering should perform the
  same accepted small-int narrowing/wrapping and emit the optional payload plus
  tag slots consistently. In-range `255` for `UInt8?` should not pass `check`
  and then fail lowering.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` accepts optional payloads when
    `typesCompatible(elem, actual)` is true, and `typesCompatible("u8", "i32")`
    / `typesCompatible("u16", "i32")` are true through `isInt32Like`.
  - `compiler/internal/lower/lower_core.go` `lowerExprAs` wraps an optional payload
    only when `actualType == expectedInfo.ElemType`; numeric literals infer as
    `i32`, so `UInt8?`/`UInt16?` literals fall through to plain one-slot
    lowering.
  - The later local/assignment store still expects the optional wrapper slot
    shape, producing `slot mismatch for 'maybe'` or `slot mismatch for
    assignment`.
- Why it matters: optional byte/word values are natural for nullable headers,
  optional status codes, parsed config, and partial RPC payloads. The current
  compiler accepts such code at semantic-check time but fails during run/build,
  so users get a later internal lowering error instead of a stable diagnostic
  or a working optional value.

### BUG-029 - Nested optional literal payloads pass check but fail lowering

- Area: checker/lowering contract for implicit optional lifting across more
  than one `?` layer.
- Severity: high; accepted nested nullable DTO/config fields, assignments, and
  returns fail after `check` with internal slot-mismatch errors.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_literal_payload.tetra`
  - `/tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_assignment_literal_payload.tetra`
  - `/tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_return_literal_payload.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_none_control.tetra`
  - `/tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_inner_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_literal_payload.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_literal_payload.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_assignment_literal_payload.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_assignment_literal_payload.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_return_literal_payload.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_return_literal_payload.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_none_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_none_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_inner_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-022/bughunt/nested_optional_inner_control.tetra
```

- Observed:
  - `let nested: Int?? = 42` passes `check`, then `run` fails with
    `slot mismatch for 'nested'`.
  - `var nested: Int?? = none; nested = 42` passes `check`, then `run` fails
    with `slot mismatch for assignment`.
  - `func make_nested() -> Int??: return 42` passes `check`, then `run` fails
    with `return slot mismatch`.
  - Controls work: `let nested: Int?? = none` and
    `let inner: Int? = 42; let nested: Int?? = inner` both pass `check` and
    return `exit status 42`.
- Expected: either recursive implicit optional lifting should lower all
  required payload/tag slots consistently, or the checker should reject direct
  `Int` payloads for `Int??` and require an explicitly typed `Int?` payload.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` `typesCompatible` accepts optional
    payloads recursively via `typesCompatible(elem, actual)`, so
    `typesCompatible("i32??", "i32")` succeeds through the inner `i32?`.
  - `compiler/internal/lower/lower_core.go` `lowerExprAs` only wraps one optional
    layer when `actualType == expectedInfo.ElemType`; direct `42` infers as
    `i32`, not `i32?`, so it falls through to one-slot lowering.
  - Local initialization, assignment, and return lowering still expect the
    three-slot `Int??` shape, producing the observed slot mismatches.
- Why it matters: nested optionals are a common representation for
  "missing field" versus "present null" in service/config payloads. The current
  contract accepts the code at check time but fails later in build/run, which
  turns a user-facing type decision into an internal lowering error.

### BUG-030 - Mutable slice `ptr`/`len` fields let safe code bypass bounds checks

- Area: slice metadata mutability, safe-code memory safety, and native index
  bounds checks.
- Severity: critical; safe Tetra code can enlarge or mismatch slice metadata
  after allocation, then write outside the allocation without `unsafe` or
  `cap.mem`.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_oob_write.tetra`
  - `/tmp/tetra-bug-hunt/session-023/bughunt/slice_ptr_len_mismatch_oob_write.tetra`
  - `/tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_blocks_valid_index.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-023/bughunt/slice_oob_without_len_mutation_control.tetra`
  - `/tmp/tetra-bug-hunt/session-023/bughunt/slice_len_immutable_control.tetra`
  - `/tmp/tetra-bug-hunt/session-023/bughunt/fixed_array_len_assignment_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_oob_write.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_oob_write.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/slice_ptr_len_mismatch_oob_write.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-023/bughunt/slice_ptr_len_mismatch_oob_write.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_blocks_valid_index.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-023/bughunt/slice_len_mutation_blocks_valid_index.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/slice_oob_without_len_mutation_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-023/bughunt/slice_oob_without_len_mutation_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/slice_len_immutable_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-023/bughunt/fixed_array_len_assignment_rejected_control.tetra
```

- Observed:
  - `var bytes: []u8 = core.make_u8(1); bytes.len = 64; bytes[50] = 42`
    passes `check` and returns `exit status 42`, proving index `50` was allowed
    through a one-byte slice after safe metadata mutation.
  - `var tiny: []u8 = core.make_u8(1); var wide: []u8 = core.make_u8(64);
    wide.ptr = tiny.ptr; wide[50] = 42` passes `check` and returns
    `exit status 42`, proving safe code can mismatch pointer and length fields.
  - `bytes.len = 0` after a valid one-byte allocation makes `bytes[0] = 42`
    fail with `exit status 1`, proving the runtime bounds check trusts the
    mutable metadata.
  - Without metadata mutation, `bytes[1] = 42` on a one-byte slice fails with
    `exit status 1`.
  - `let bytes: []u8 = ...; bytes.len = 64` is rejected as assignment to an
    immutable `val`, so the hole is specifically mutable slice metadata.
  - The analogous fixed-array field assignment `box.items.len = 64` is rejected
    with `cannot assign to fixed-array internals ('ptr'/'len')`.
- Expected: safe code should not be able to assign to slice `ptr` or `len`.
  Slice metadata should be immutable/opaque like fixed-array internals, or any
  metadata mutation should be confined to an audited unsafe boundary that
  preserves allocation provenance and length invariants.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` models `TypeSlice` with public
    `ptr` and `len` fields.
  - `compiler/internal/semantics/semantics_core.go` `resolveAssignTarget` permits
    field assignment after `resolveFieldChain`.
  - `rejectFixedArrayInternalAssignment` rejects `ptr`/`len` only when the
    current type is `TypeArray`; it does not reject `TypeSlice` internals.
  - Native x64 `IRIndexLoad*` / `IRIndexStore*` checks compare the index against
    the slice length slot and then address from the slice pointer slot, so
    forged metadata directly changes the bounds/provenance used by safe index
    operations.
- Why it matters: services commonly keep slices for request bodies, byte
  buffers, frame payloads, and packet fields. Allowing ordinary safe code to
  rewrite slice metadata turns a one-byte allocation into an apparently
  64-byte buffer and bypasses the safe index bounds contract.

### BUG-031 - Mutable `String` `ptr`/`len` fields let safe code read outside string bounds

- Area: string view metadata mutability, safe-code memory safety, and native
  string indexing / collection iteration.
- Severity: high; safe Tetra code can enlarge or mismatch a `String` view and
  read bytes beyond the original string literal without `unsafe`, `cap.mem`, or
  raw pointer APIs.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_oob_read.tetra`
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_ptr_len_mismatch_oob_read.tetra`
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_for_len_mutation_count.tetra`
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_blocks_valid_read.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_oob_without_len_mutation_control.tetra`
  - `/tmp/tetra-bug-hunt/session-024/bughunt/string_len_immutable_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_oob_read.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_oob_read.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_ptr_len_mismatch_oob_read.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-024/bughunt/string_ptr_len_mismatch_oob_read.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_for_len_mutation_count.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-024/bughunt/string_for_len_mutation_count.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_blocks_valid_read.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-024/bughunt/string_len_mutation_blocks_valid_read.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_oob_without_len_mutation_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-024/bughunt/string_oob_without_len_mutation_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-024/bughunt/string_len_immutable_control.tetra
```

- Observed:
  - `var text: String = "*"; text.len = 2; let second: UInt8 = text[1]`
    passes `check` and returns `exit status 42`, proving index `1` was allowed
    through a one-byte string literal after safe metadata mutation.
  - `var tiny: String = "*"; var wide: String = "AB"; wide.ptr = tiny.ptr;
    let second: UInt8 = wide[1]` passes `check` and returns `exit status 42`,
    proving safe code can mismatch a longer string length with a shorter string
    pointer.
  - `text.len = 2` also makes `for ch in text` execute two iterations over a
    one-byte literal; the repro returns `exit status 42` after counting two
    loaded elements.
  - `text.len = 0` after `var text: String = "*"` makes `text[0]` fail with
    `exit status 1`, proving string indexing trusts the mutable length slot.
  - Without metadata mutation, `text[1]` on `"*"` fails with `exit status 1`.
  - `let text: String = "*"; text.len = 2` is rejected as assignment to an
    immutable `val`, so the hole is specifically mutable `String` metadata.
- Expected: safe code should not be able to assign to `String.ptr` or
  `String.len`. String views should be immutable/opaque; changing their
  pointer/length pair should require an audited unsafe operation that preserves
  provenance and bounds.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` builds `str` by calling
    `makeSliceTypeInfo("str", "u8")`, then changing the kind to `TypeStr`,
    so `String` exposes the same `ptr` and `len` fields as slices.
  - `compiler/internal/semantics/semantics_core.go` `resolveAssignTarget` permits
    field assignment after `resolveFieldChain`, and
    `rejectFixedArrayInternalAssignment` only rejects `ptr`/`len` for
    `TypeArray`, not `TypeStr`.
  - `compiler/internal/lower/lower_core.go` treats `TypeStr` as indexable with
    element type `u8`, and native `IRIndexLoadU8` compares against the string
    length slot and addresses through the string pointer slot.
- Why it matters: strings carry request paths, hostnames, headers, and protocol
  tokens. Letting safe code forge string view metadata can expose adjacent
  bytes and make parsers/validators operate on data outside the original
  string value.

### BUG-032 - Typed task error enums can carry non-sendable `String`/`ptr` payloads across task boundaries

- Area: typed task transfer safety, typed-error payload validation, and
  task-boundary sendability.
- Severity: high; task workers can throw enum payloads containing string views
  or raw pointers across `core.task_join_i32_typed`, even though actor typed
  messages reject the same payload categories and task-boundary checks claim
  sendable-value coverage.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-028/bughunt/typed_task_string_error_payload.tetra`
  - `/tmp/tetra-bug-hunt/session-028/bughunt/typed_task_ptr_error_payload_null.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-028/bughunt/typed_actor_string_payload_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-028/bughunt/typed_task_string_return_control.tetra`
- Additional probes:
  - `/tmp/tetra-bug-hunt/session-029/bughunt/typed_task_error_actor_payload.tetra`
  - `/tmp/tetra-bug-hunt/session-029/bughunt/typed_task_error_task_handle_payload_match_only.tetra`
  - `/tmp/tetra-bug-hunt/session-029/bughunt/typed_task_error_island_payload_type_only.tetra`
  - `/tmp/tetra-bug-hunt/session-029/bughunt/typed_task_error_cap_payload_type_only.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-028/bughunt/typed_task_string_error_payload.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-028/bughunt/typed_task_string_error_payload.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-028/bughunt/typed_task_ptr_error_payload_null.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-028/bughunt/typed_task_ptr_error_payload_null.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-028/bughunt/typed_actor_string_payload_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-028/bughunt/typed_task_string_return_control.tetra
```

- Observed:
  - `enum TaskErr: case boom(String)` plus a worker that throws
    `TaskErr.boom("*")` passes `check`; catching
    `core.task_join_i32_typed<TaskErr>(task)` in `main` reads `text[0]` and
    returns `exit status 42`.
  - `enum TaskErr: case bad(ptr)` plus a worker that throws
    `TaskErr.bad(0)` also passes `check` and returns `exit status 42` through
    the catch arm.
  - The nearby actor typed-message control rejects `Msg.text(String)` with
    `typed actor message payload must be value-only, got string view 'str'`.
  - The nearby task worker return control rejects `func worker() -> String`
    before spawn with `task_spawn_i32 target must have shape func worker() -> i32`.
  - Session 029 showed the same unchecked `ThrowsType` gate accepts an
    `actor` payload and a `task.i32` payload that both match through
    `task_join_i32_typed`; attempting to join the caught `task.i32` is stopped
    later by `ambiguous resource provenance`, but the payload itself crosses
    the typed-error channel.
  - Session 029 also showed `TaskErr.moved(island)` and
    `TaskErr.leaked(cap.mem)` are accepted as typed task error arguments and
    run normally when the worker returns successfully, proving `E` is not
    rejected up front even when an error case contains non-sendable
    island/capability payloads.
- Expected: typed task error enum payloads should be validated with the same
  task-boundary sendability policy as worker params/returns, or
  `task_spawn_i32_typed<E>` should reject `E` when any error payload contains
  `String`, `ptr`, capabilities, slices, arrays, actors, or other non-sendable
  handles/views.
- Evidence path:
  - `docs/spec/flow/v1_scope.md` describes Actor/task transfer safety as covering
    worker entrypoints, sendable results, handle transfer, and local
    actor/task safety.
  - `docs/spec/core/current_supported_surface.md` advertises an Actor/task transfer
    safety MVP for local worker entrypoints, sendable scalar and supported
    structural results, handle transfer, and related diagnostics.
  - `compiler/internal/semantics/semantics_core.go`
    `funcSigActorTaskTransferUnsafeReason` validates only parameter and return
    types with `typeActorTaskSendable`; it does not inspect `sig.ThrowsType`.
  - `compiler/internal/semantics/semantics_expressions.go` `validateTypedTaskErrorType` only
    checks that the typed task error argument is an enum.
  - `compiler/internal/semantics/semantics_expressions.go` `checkTypedTaskBuiltin` calls
    `validateTypedTaskErrorType` and then `funcSigActorTaskTransferSafe`, so
    non-sendable error payloads bypass the task-boundary sendability check.
  - `validateTypedActorMessageType` already rejects `TypeStr`, `TypePtr`,
    `TypeCap`, slices, arrays, optionals, and actors for typed actor messages,
    showing the stricter value-only policy exists on the adjacent boundary.
- Why it matters: typed errors are part of the task result channel. Letting
  string views or raw pointers cross a task boundary can move lifetime- or
  address-sensitive data through a path that is advertised as transfer-safe,
  undermining the worker boundary checks used by service code.

### BUG-033 - Typed task group spawn bypasses closed `task.group` ownership checks

- Area: typed task groups, resource finalization, task group ownership, and
  specialized builtin checking.
- Severity: high; safe code can reuse a closed `task.group` with
  `core.task_spawn_group_i32_typed<E>` even though the untyped group spawn and
  ordinary task-group APIs reject the same use-after-close.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-031/bughunt/typed_group_spawn_after_close_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_after_maybe_close_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_optional_alias_after_close_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_enum_alias_after_close_repro.tetra`
- Control:
  - `/tmp/tetra-bug-hunt/session-031/bughunt/untyped_group_spawn_after_close_control.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_after_maybe_close_control.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_optional_alias_after_close_control.tetra`
  - `/tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_enum_alias_after_close_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-031/bughunt/untyped_group_spawn_after_close_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-031/bughunt/typed_group_spawn_after_close_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-031/bughunt/typed_group_spawn_after_close_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_after_maybe_close_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_after_maybe_close_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_after_maybe_close_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_optional_alias_after_close_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_optional_alias_after_close_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_optional_alias_after_close_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/untyped_group_spawn_enum_alias_after_close_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_enum_alias_after_close_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-032/bughunt/typed_group_spawn_enum_alias_after_close_repro.tetra
```

- Observed:
  - The untyped control is rejected during `check` with
    `cannot use closed resource 'group'`, pointing back to the
    `core.task_group_close(group)` line.
  - The typed version closes the same `group` and then calls
    `core.task_spawn_group_i32_typed<GroupErr>(group, "worker")`; `check`
    passes.
  - Running the typed version returns `exit status 5` through the
    `GroupErr.stopped` catch arm, proving the closed group reaches the typed
    task group runtime path instead of being stopped by the ownership checker.
  - Session 032 showed the same bypass after a control-flow merge:
    the untyped spawn reports `resource may have been closed after
    control-flow merge`, while the typed spawn passes `check` and returns
    `exit status 5`.
  - Session 032 also showed optional-payload and enum-payload aliases of a
    closed group are rejected by untyped group spawn as `cannot use closed
    resource 'other'`, but both aliases pass the typed group spawn checker and
    return `exit status 5`.
- Expected: `core.task_spawn_group_i32_typed<E>(group, "worker")` should enforce
  the same `task.group` resource-finalization rule as
  `core.task_spawn_group_i32(group, "worker")` and reject closed or maybe-closed
  groups before lowering.
- Evidence path:
  - `compiler/tests/runtime/resource_finalization_test.go`
    `TestTaskGroupFinalizationRejectsSpawnAfterClose` expects the untyped group
    spawn to reject a closed `task.group`.
  - `compiler/internal/semantics/semantics_expressions.go` generic builtin checking calls
    `checkResourceCallArg` for each argument and then
    `markCallFinalizedResources`, which is the path that catches ordinary
    task-group use-after-close.
  - `compiler/internal/semantics/semantics_expressions.go` `checkTypedTaskBuiltin` handles
    `core.task_spawn_group_i32_typed` in a specialized branch; for the first
    argument it only calls `checkExprWithEffects` and checks that the type is
    `task.group`.
  - Graphify `get_neighbors(checkTypedTaskBuiltin(), calls)` showed no call to
    `checkResourceCallArg`, while the generic call path has that resource
    validation.
- Why it matters: task groups are affine runtime resources. Letting a typed
  task spawn bypass the closed-group diagnostic weakens the resource
  finalization model exactly on the structured-concurrency API surface.

### BUG-034 - Secret-tainted values can leak through thrown enum payloads from `@export` functions

- Area: privacy taint analysis, `throw` statements, exported function
  boundaries, and enum payloads.
- Severity: high; the checker blocks returning an unsealed secret-tainted
  `Int` from an `@export` function, but the same value can be thrown inside a
  plain enum payload and caught by the caller.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-036/bughunt/export_secret_throw_payload_repro.tetra`
- Control:
  - `/tmp/tetra-bug-hunt/session-036/bughunt/export_secret_return_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-036/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-036/bughunt/export_secret_throw_payload_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-036/bughunt/export_secret_throw_payload_repro.tetra
```

- Observed:
  - The return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The throw repro defines `@export("leak_throw") func leak(...) -> Int throws
    LeakErr`, unseals `secret.i32` into `raw: Int`, and throws
    `LeakErr.raw(raw)`.
  - `check` accepts the throw repro, and `run` prints `exit status 42` after
    `main` catches `LeakErr.raw(value)` and returns the payload.
- Expected: secret-tainted values should not be able to cross an exported
  boundary through thrown enum payloads. `ThrowStmt` should apply equivalent
  `exprSecretTainted` policy to the thrown expression, at least for exported
  functions and for non-privacy throw sites.
- Evidence path:
  - `compiler/tests/safety/effects/effects_test.go` covers the return-side policy:
    exported functions cannot return unsealed secret-tainted locals.
  - `compiler/internal/semantics/semantics_checker.go` checks `exprSecretTainted` for
    `ReturnStmt` and emits the `@export` diagnostic when needed.
  - The adjacent `ThrowStmt` checker validates throw type compatibility,
    borrowed escapes, region scope, and resource payload summaries, but does
    not call `exprSecretTainted` for the thrown value.
  - `exprSecretTainted` already treats `core.secret_unseal_i32` as tainted, so
    the missing hook is in throw-statement policy rather than taint detection.
- Why it matters: thrown enum payloads are an observable function boundary just
  like returns. A service API that bans raw secret returns can still expose the
  raw value through its error channel.

### BUG-035 - Secret-tainted byte buffers can be emitted through `print`

- Area: privacy taint analysis, `PrintStmt`, IO sinks, and printable byte
  buffers.
- Severity: high; an unsealed secret-tainted `Int` can be written into a local
  `[]UInt8` buffer and printed to stdout even though the checker blocks the
  same value from returning across an exported boundary.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-040/bughunt/secret_print_u8_buffer_assignment_probe.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-040/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-040/bughunt/print_plain_u8_buffer_control.tetra`
  - `/tmp/tetra-bug-hunt/session-040/bughunt/export_secret_local_field_assignment_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-040/bughunt/export_secret_local_index_assignment_repro.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-040/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-040/bughunt/secret_print_u8_buffer_assignment_probe.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-040/bughunt/secret_print_u8_buffer_assignment_probe.tetra
```

- Observed:
  - The exported return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - Local field and index assignment controls are also rejected on exported
    return, proving the local container taint is tracked when the tainted value
    is read back.
  - The print repro unseals `secret.i32(42)` into `raw`, writes `raw` into
    `bytes[0]`, writes newline into `bytes[1]`, and calls `print(bytes)`.
  - `check` accepts the print repro, and `run` prints `*`, the byte value 42,
    to stdout.
- Expected: `print` should apply the same secret-taint policy to its printable
  expression as other externally observable sinks, or require an explicit
  declassification primitive/policy before emitting tainted bytes.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `PrintStmt` checking only requires
    the `io` effect, checks the expression type, and validates
    `isPrintableType`; it does not call `exprSecretTainted`.
  - `ReturnStmt` in the same checker path calls `exprSecretTainted` and emits
    the exported-return privacy diagnostic.
  - `AssignStmt` marks local containers tainted after secret-tainted field or
    index writes; the Session 040 field/index return controls were rejected.
  - Graphify navigation (`query_graph`, `get_neighbors(PrintStmt)`, and
    `shortest_path(PrintStmt, exprSecretTainted())`) led back to the
    `checkStmts`/`PrintStmt` path, then source inspection confirmed the missing
    sink check.
- Why it matters: stdout is an observable service boundary. A program that
  cannot return or store a raw secret can still disclose it by packing it into a
  printable buffer and using `print`.

### BUG-036 - Secret-tainted conditions can leak through public branch outputs

- Area: privacy taint analysis, implicit flows, `IfStmt` conditions, exported
  returns, and global assignment sinks.
- Severity: high; the checker blocks directly returning an unsealed secret, but
  accepts a branch whose condition depends on that secret and whose public
  constants reveal which branch was taken.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_false_branch_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-041/bughunt/global_secret_if_condition_assignment_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-041/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-041/bughunt/export_public_branch_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-041/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_false_branch_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-041/bughunt/export_secret_if_condition_return_false_branch_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-041/bughunt/global_secret_if_condition_assignment_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-041/bughunt/global_secret_if_condition_assignment_repro.tetra
```

- Observed:
  - The direct return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The exported branch repro unseals `secret.i32` into `raw`, branches on
    `raw == 1`, and returns public constants `42` or `7`.
  - `check` accepts both true-branch and false-branch repros.
  - With `secret_seal_i32(1, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(0, token)`, `run` prints `exit status 7`.
  - The global-assignment variant also passes `check` and prints
    `exit status 42`, proving the same implicit flow can write public constants
    into a global based on a secret condition.
  - Session 042 extended the same root cause to `match` expressions, `match`
    statements, and `while` conditions: all accepted secret-controlled public
    constants, and the `while` false-branch twin returned `exit status 7`.
- Expected: either secret-tainted conditions should taint returns, global
  writes, and other outward effects in their control-dependent blocks, or the
  privacy model should explicitly reject/require declassification for branches
  whose condition is secret-tainted.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `ReturnStmt` calls
    `exprSecretTainted` on the returned expression, which catches direct raw
    secret returns.
  - `IfStmt` checking calls `checkExprWithEffects` for the condition and only
    validates that it is a condition type; it does not call
    `exprSecretTainted` on the condition or propagate a secret-control context
    into branch `ReturnStmt`/`AssignStmt` checks.
  - `MatchExpr` taint checking inspects case result expressions but not the
    tainted scrutinee as an implicit control dependency; `MatchStmt` records
    scrutinee taint for bindings but does not taint public case outputs.
  - `AssignStmt` rejects explicit secret-tainted values assigned to globals,
    but public constants assigned under a secret-tainted condition are not
    marked tainted.
  - Graphify navigation (`query_graph` for `IfStmt`/`ReturnStmt` privacy,
    `get_neighbors(checkStmts())`, and
    `shortest_path(IfStmt, exprSecretTainted())`) pointed to the same
    `checkStmts` paths; source inspection confirmed the missing
    control-dependence taint hook.
- Why it matters: an exported service API can leak arbitrary bits of a secret
  by branching on the secret and returning or storing public sentinel values,
  even when explicit raw secret returns and explicit secret global writes are
  blocked.

### BUG-037 - Secret-tainted values can be laundered through actor mailboxes

- Area: privacy taint analysis, actor runtime mailboxes, `core.send`,
  `core.recv`, statement calls, and exported returns.
- Severity: high; an unsealed secret-tainted `Int` can be sent through an actor
  mailbox and received back as an apparently clean `Int`, bypassing exported
  return taint checks.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_false_value_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-043/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-043/bughunt/actor_public_self_send_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-043/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-043/bughunt/actor_public_self_send_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-043/bughunt/actor_public_self_send_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_false_value_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-043/bughunt/export_secret_actor_mailbox_launder_false_value_repro.tetra
```

- Observed:
  - The direct return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The public actor self-send control passes `check` and `run` prints
    `exit status 42`, confirming the mailbox round trip.
  - The secret repro unseals `secret.i32`, sends `raw` to `core.self()` via
    `core.send`, then returns `core.recv()` from an `@export` function.
  - `check` accepts the secret mailbox repro.
  - With `secret_seal_i32(42, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(7, token)`, `run` prints `exit status 7`.
  - Session 044 extended the same mailbox taint-loss root cause to
    `core.send_typed`/`core.recv_typed<LeakMsg>()` enum payloads and to tagged
    `core.send_msg`/`core.recv_msg().value`; both variants passed `check` and
    returned `exit status 42` or `exit status 7` depending on the sealed value.
- Expected: sending a secret-tainted scalar into an actor mailbox should either
  be rejected, require an explicit declassification policy, or taint the
  receive side so exported returns of mailbox data remain blocked.
- Evidence path:
  - `docs/spec/runtime/actors.md` documents `core.send(to: actor, v: i32) -> i32` as
    sending a message to another actor and `core.recv() -> i32` as receiving
    from the current mailbox.
  - `compiler/internal/semantics/semantics_checker.go` `ExprStmt` calls
    `exprSecretTainted` on call statements but discards the returned `true`
    value unless an error is produced.
  - `exprSecretTainted(CallExpr)` treats calls with tainted arguments to
    `core.*` as tainted results rather than rejecting effectful sink calls.
  - `core.recv()` has no argument and is not modeled as returning tainted data
    from a mailbox that previously received a tainted value.
  - `core.send_typed` validates enum/value-only payload shape but not
    secret-tainted payload values; `core.recv_typed` returns the enum without
    mailbox taint provenance. Tagged `actor.msg` receives have the same gap on
    the `.value` field.
  - Graphify navigation for actor mailbox calls and `ExprStmt`/`CallExpr`
    pointed to `checkStmts`, `checkCallExprWithEffects`, and runtime actor
    build/run tests; source inspection confirmed the missing mailbox taint
    boundary.
- Why it matters: actor mailboxes are service boundaries. A program can move
  an unsealed secret through a runtime queue and return it from an exported API
  even though the checker blocks direct raw secret returns.

### BUG-038 - Secret-tainted values can be laundered through raw memory load/store

- Area: privacy taint analysis, raw memory builtins, `unsafe` blocks,
  `cap.mem`, and exported returns.
- Severity: high; inside an explicit `unsafe:` block, an unsealed
  secret-tainted `Int` can be written through `core.store_i32` and read back
  through `core.load_i32` as an apparently clean `Int`, bypassing exported
  return taint checks.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_false_value_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-045/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-045/bughunt/public_raw_memory_roundtrip_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-045/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-045/bughunt/public_raw_memory_roundtrip_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-045/bughunt/public_raw_memory_roundtrip_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_false_value_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-045/bughunt/export_secret_raw_memory_launder_false_value_repro.tetra
```

- Observed:
  - The direct return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The public raw-memory control passes `check` and `run` prints
    `exit status 42`, confirming the `cap.mem` round trip.
  - The secret repro unseals `secret.i32`, enters `unsafe:`, obtains
    `cap.mem`, writes `raw` to a freshly allocated pointer with
    `core.store_i32`, then returns `core.load_i32` from the same pointer.
  - `check` accepts both secret raw-memory repros.
  - With `secret_seal_i32(42, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(7, token)`, `run` prints `exit status 7`.
- Expected: raw memory may require `unsafe` and `cap.mem`, but it should not
  silently declassify privacy-tainted data. Either storing secret-tainted values
  through raw memory should be rejected/require declassification, or loads from
  memory that may contain tainted data should preserve taint.
- Evidence path:
  - `docs/spec/runtime/effects_capabilities_privacy_v1.md` states the v1 privacy MVP is
    static auditing and call-shape enforcement, while `cap.mem` belongs to a
    separate unsafe capability boundary.
  - `docs/spec/runtime/unsafe.md` documents `core.load_i32`/`core.store_i32` as unsafe
    raw-memory operations requiring `mem` and a `cap.mem` argument; it does not
    state that raw memory declassifies privacy-tainted values.
  - `compiler/internal/semantics/semantics_checker.go` `ReturnStmt` checks
    `exprSecretTainted`, which blocks the direct raw secret return control.
  - `exprSecretTainted(CallExpr)` treats a `core.*` call with tainted arguments
    as a tainted result, so `core.store_i32(p, raw, mem)` can mark only the
    ignored store result/let binding as tainted.
  - `core.load_i32(p, mem)` has no tainted argument, so its return value is not
    connected to the previous store's tainted payload.
  - Graphify/source navigation for raw-memory builtins led to the builtin
    signatures, effect/unsafe policy, and privacy return checking paths;
    source inspection confirmed there is no memory-cell taint provenance.
- Why it matters: raw memory is a powerful but still typed language boundary.
  If `unsafe` also implicitly disables privacy taint, exported APIs can
  disclose unsealed secrets with a one-cell memory round trip while all direct
  privacy checks appear to pass.

### BUG-039 - Secret-tainted sleep durations can leak through runtime logical time

- Area: privacy taint analysis, runtime time builtins, temporal side channels,
  `core.sleep_ms`, `core.time_now_ms`, and exported returns.
- Severity: high; the checker blocks returning a runtime value directly derived
  from a secret-tainted argument, but accepts sleeping for a secret-tainted
  duration and then returning the public logical clock value.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_false_value_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-046/bughunt/public_sleep_time_control.tetra`
  - `/tmp/tetra-bug-hunt/session-046/bughunt/export_secret_deadline_return_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-046/bughunt/public_sleep_time_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-046/bughunt/public_sleep_time_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-046/bughunt/export_secret_deadline_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_false_value_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-046/bughunt/export_secret_sleep_time_launder_false_value_repro.tetra
```

- Observed:
  - The public control passes `check`; `run` prints `exit status 42` after
    `core.sleep_ms(42)` and `core.time_now_ms()`.
  - The direct `return core.deadline_ms(raw)` control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`,
    proving direct tainted runtime-derived values are caught.
  - The repro unseals `secret.i32` into `raw`, calls `core.sleep_ms(raw)`, then
    returns `core.time_now_ms()` from an exported function.
  - `check` accepts both secret sleep/time repros.
  - With `secret_seal_i32(42, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(7, token)`, `run` prints `exit status 7`.
- Expected: a secret-tainted duration should not be allowed to mutate
  externally observable runtime time without preserving taint or requiring an
  explicit declassification policy. If `core.sleep_ms(raw)` is accepted, the
  runtime clock value derived from that sleep should be considered tainted for
  outward sinks such as exported returns.
- Evidence path:
  - `docs/spec/runtime/runtime_abi.md` documents that the runtime clock is
    deterministic/logical, starts at `0`, and that `sleep_ms` parks until
    `time_now_ms() + ms`.
  - `compiler/internal/semantics/semantics_checker.go` `ReturnStmt` calls
    `exprSecretTainted`, which catches direct `core.deadline_ms(raw)` returns.
  - `exprSecretTainted(CallExpr)` treats a `core.*` call with tainted arguments
    as a tainted result, so the local `_slept` binding becomes tainted, but the
    global/logical clock side effect is not modeled.
  - `core.time_now_ms()` has no tainted argument, so returning it after a
    secret-controlled sleep is treated as clean.
  - Graphify/source navigation for runtime time builtins led to the builtin
    signatures, runtime ABI docs, and privacy return checking paths; source
    inspection confirmed there is no temporal-state taint provenance.
- Why it matters: logical time is an observable service output. A Tetra service
  can encode a secret in its runtime delay and then return the public clock,
  bypassing direct secret-return diagnostics while still revealing the payload.

### BUG-040 - Secret-tainted values can be written through observable MMIO and read back clean

- Area: privacy taint analysis, MMIO builtins, `unsafe` blocks, `cap.io`,
  `io`/`mmio` effects, and exported returns.
- Severity: high; inside an explicit `unsafe:` block, an unsealed
  secret-tainted `Int` can be passed to `core.mmio_write_i32`, which is an
  observable MMIO operation by language contract, and then read back with
  `core.mmio_read_i32` as an apparently clean public value.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_false_value_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-047/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-047/bughunt/public_mmio_roundtrip_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-047/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-047/bughunt/public_mmio_roundtrip_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-047/bughunt/public_mmio_roundtrip_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_false_value_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-047/bughunt/export_secret_mmio_launder_false_value_repro.tetra
```

- Observed:
  - The direct return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The public MMIO control passes `check` and `run` prints `exit status 42`,
    confirming the `cap.io` MMIO round trip.
  - The secret repro unseals `secret.i32` into `raw`, enters `unsafe:`,
    obtains `cap.io`, writes `raw` with `core.mmio_write_i32`, then returns
    `core.mmio_read_i32` from the same pointer.
  - `check` accepts both secret MMIO repros.
  - With `secret_seal_i32(42, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(7, token)`, `run` prints `exit status 7`.
- Expected: MMIO may require `unsafe`, `cap.io`, and `io`/`mmio` effects, but
  those gates should not silently declassify privacy-tainted data. A
  secret-tainted MMIO write should be rejected, require explicit
  declassification, or taint subsequent reads from the affected location.
- Evidence path:
  - `docs/spec/runtime/capabilities.md` states that MMIO read/write currently lower to
    normal memory operations, but the language contract treats MMIO operations
    as observable and orders them as MMIO.
  - `compiler/internal/semantics/semantics_core.go` defines `core.mmio_write_i32` as
    `(ptr, i32, cap.io) -> i32` and `core.mmio_read_i32` as
    `(ptr, cap.io) -> i32`.
  - `compiler/internal/semantics/semantics_checker.go` `ReturnStmt` calls
    `exprSecretTainted`, which blocks the direct raw secret return control.
  - `exprSecretTainted(CallExpr)` treats a `core.*` call with tainted arguments
    as a tainted result, so `core.mmio_write_i32(p, raw, io_cap)` can taint
    only the ignored write result/local; it does not reject the observable
    MMIO sink or mark the target location.
  - `core.mmio_read_i32(p, io_cap)` has no tainted argument, so its return is
    not connected to the previous secret-tainted MMIO write.
  - Graphify/source navigation for MMIO privacy taint led to the builtin
    signatures, MMIO capability docs, and privacy return checking paths;
    source inspection confirmed there is no MMIO-location taint provenance.
- Why it matters: MMIO is explicitly observable by contract. A Tetra service
  can disclose unsealed secrets through an I/O-shaped boundary while the
  direct exported-return diagnostic still appears to enforce privacy.

### BUG-041 - Secret-tainted values can be laundered through closure captures

- Area: privacy taint analysis, closure captures, function-typed locals,
  callable invocation, and exported returns.
- Severity: high; an `@export` function can unseal `secret.i32` into a
  secret-tainted `Int`, capture it in a local `fn() -> Int` closure, call the
  closure, and return the result as if it were a clean public value.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_false_value_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-049/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-049/bughunt/public_closure_capture_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-049/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-049/bughunt/public_closure_capture_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-049/bughunt/public_closure_capture_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_false_value_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-049/bughunt/export_secret_closure_capture_launder_false_value_repro.tetra
```

- Observed:
  - The direct return control is rejected with
    `secret-tainted value cannot be returned from @export function 'leak'`.
  - The public closure-capture control passes `check` and `run` prints
    `exit status 42`, confirming the supported immutable `Int` capture path.
  - The secret repro unseals `secret.i32` into `raw`, binds
    `let f: fn() -> Int = fn() -> Int: return raw`, then returns `f()` from
    the exported function.
  - `check` accepts both secret closure-capture repros.
  - With `secret_seal_i32(42, token)`, `run` prints `exit status 42`.
  - With `secret_seal_i32(7, token)`, `run` prints `exit status 7`.
- Expected: closure capture metadata should preserve privacy provenance. A
  closure that captures a secret-tainted local should either be rejected at the
  exported boundary, require explicit declassification, or mark calls through
  that function-typed value as secret-tainted.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `ReturnStmt` calls
    `exprSecretTainted`, which blocks the direct raw secret return control.
  - `exprSecretTainted(CallExpr)` propagates taint from tainted call arguments
    and from known `funcReturnSecretTaint`, but a zero-argument function-typed
    local call has no tainted argument.
  - The same `exprSecretTainted` switch has no `ClosureExpr`/capture case that
    marks a function value tainted when its environment captures a tainted
    local.
  - Closure/callable source inspection shows rich capture metadata for
    ownership and escape checks, but the privacy taint path is not connected to
    that capture metadata.
  - Graphify/source navigation for closure/function-typed returns led to the
    closure capture machinery and the `exprSecretTainted` call path; source
    inspection confirmed the missing capture-taint link.
- Why it matters: closure values are a normal language boundary for delayed
  computation. Without capture-aware privacy taint, a service can wrap a raw
  secret in a zero-argument callback and immediately call it to bypass direct
  exported-return diagnostics.

### BUG-042 - Function-typed calls bypass budget context guardrails

- Area: budget semantic clauses, function-typed locals/globals, callbacks,
  callable invocation, and static budget context validation.
- Severity: medium-high; code with `budget(5)` can call a known `budget(6)`
  target by first routing it through a function-typed value or callback, even
  though the same direct call is rejected by the budget context checker.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_underbudget_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_global_underbudget_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-058/bughunt/budget_callback_underbudget_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-058/bughunt/budget_direct_underbudget_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_covered_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-058/bughunt/budget_direct_underbudget_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_underbudget_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_underbudget_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_global_underbudget_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_global_underbudget_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-058/bughunt/budget_callback_underbudget_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-058/bughunt/budget_callback_underbudget_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_covered_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-058/bughunt/budget_function_typed_local_covered_control.tetra
```

- Observed:
  - The direct control `return callee(41)` from a `budget(5)` caller to a
    `budget(6)` callee is rejected with
    `budget context for call to 'callee' requires caller budget at least 6, got 5`.
  - The local function-typed repro binds
    `let f: fn(Int) -> Int uses budget = callee` inside the same `budget(5)`
    caller, then returns `f(41)`.
  - The global function-typed repro initializes
    `var cb: fn(Int) -> Int uses budget = callee`, then returns `cb(41)` from
    a `budget(5)` caller.
  - The callback repro passes `callee` into
    `apply(x: Int, cb: fn(Int) -> Int uses budget)`, where `apply` itself has
    `budget(5)` and calls `cb(x)`.
  - All three underbudget function-typed repros pass `check`.
  - All three underbudget function-typed repros run and print
    `exit status 42`, proving the known `budget(6)` target executed.
  - The covered local control with caller `budget(6)` also passes and prints
    `exit status 42`.
- Expected: when a function-typed value has a known target set containing a
  `budget(N)` function, calling it from a lower-budget context should be
  rejected the same way as the direct call, or the function type should carry
  and enforce a required budget contract rather than only the `budget` effect.
- Evidence path:
  - `docs/spec/core/current_supported_surface.md` states direct calls into
    `budget(N)` functions require a caller budget context of at least `N`.
  - `compiler/internal/semantics/semantics_checker.go` `validateBudgetContextsInExpr`
    checks named `CallExpr` targets found directly in `funcs` and string-based
    actor/task spawn targets.
  - That budget-context pass does not resolve function-typed local/global
    targets, struct or enum callable metadata, or callback argument target sets.
  - `compiler/internal/semantics/semantics_expressions.go`
    `validateCallAgainstSemanticClauseTarget` enforces `realtime`, `noalloc`,
    and `noblock`, but has no budget clause case.
  - `hasStrictSemanticCallClauses` similarly excludes `HasBudget`, so existing
    callable/callback semantic-clause validation paths do not compensate for
    the separate budget-context pass.
  - Function types record `uses budget` as an effect, but the type surface has
    no `budget(N)` amount to compare against the caller.
- Why it matters: APIs commonly accept callbacks and store function-typed
  handlers. A service can accidentally or intentionally route a budgeted target
  through those call surfaces and bypass the compiler's advertised static
  guardrail for underbudget calls.

### BUG-043 - Function-typed signatures hide nested secret types from consent checks

- Area: privacy/consent signature validation, function-typed parameters and
  returns, function-typed struct fields, enum payloads, and `fnptr` type
  resolution.
- Severity: medium-high; APIs can expose callback/function surfaces that accept
  or return `secret.i32` without the enclosing function or aggregate being
  treated as secret-bearing by the consent checker.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_param_missing_consent_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_return_missing_consent_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/function_returning_secret_callable_missing_consent_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/struct_function_typed_secret_param_missing_consent_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/enum_function_typed_secret_return_missing_consent_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-060/bughunt/direct_secret_param_missing_consent_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/struct_direct_secret_field_missing_consent_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/closure_secret_param_missing_consent_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-060/bughunt/closure_secret_return_missing_consent_repro.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/direct_secret_param_missing_consent_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_param_missing_consent_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_param_missing_consent_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_return_missing_consent_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-060/bughunt/function_typed_secret_return_missing_consent_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/function_returning_secret_callable_missing_consent_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-060/bughunt/function_returning_secret_callable_missing_consent_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/struct_function_typed_secret_param_missing_consent_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-060/bughunt/struct_function_typed_secret_param_missing_consent_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-060/bughunt/enum_function_typed_secret_return_missing_consent_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-060/bughunt/enum_function_typed_secret_return_missing_consent_repro.tetra
```

- Observed:
  - The direct control `func inspect(value: secret.i32) -> Int` is rejected
    with `secret types in function signature require semantic clause consent(<token>)`.
  - The direct aggregate control `func inspect(box: SecretBox) -> Int`, where
    `SecretBox.value: secret.i32`, is rejected with the same consent diagnostic.
  - A function parameter typed `fn(secret.i32) -> Int` passes `check` without
    `uses privacy`, `privacy`, or `consent(...)`; `run` exits successfully.
  - A function parameter typed `fn() -> secret.i32` also passes `check` and
    `run` without the enclosing function declaring consent.
  - A function returning `fn(consent.token) -> secret.i32 uses privacy` passes
    `check` and `run` without the enclosing function being treated as
    secret-bearing.
  - A struct field `cb: fn(secret.i32) -> Int` and enum payload
    `case some(fn() -> secret.i32)` both pass through functions that accept the
    aggregate without consent; both programs check and run successfully.
  - Closure controls with direct `secret.i32` parameter or return type are
    rejected with the expected `consent(<token>)` diagnostic, so the bug is not
    that all synthetic callables skip policy checks.
- Expected: `typeUsesSecret` should treat function-typed signatures as
  secret-bearing when any parameter, return, or throws type uses `secret.*`.
  Enclosing functions and aggregates that expose such function types should
  require the same privacy/consent policy as direct secret signatures.
- Evidence path:
  - `docs/spec/core/current_supported_surface.md` says secret-bearing signatures
    require `consent(<token>)`.
  - `compiler/internal/semantics/semantics_core.go` resolves `TypeRefFunction` to
    the opaque type name `fnptr` after resolving its nested parameter/return
    refs.
  - `compiler/internal/semantics/semantics_checker.go` `validateFunctionPolicyClauses`
    decides whether a function signature has secrets by calling
    `typeUsesSecret` on the resolved return, throws, and parameter type names.
  - `typeUsesSecret` recursively examines struct/enum/optional/slice/array
    type names, but it does not inspect `FunctionParamTypes`,
    `FunctionReturnType`, enum payload function metadata, or function-typed
    field metadata once the public type name is `fnptr`.
  - Function-typed aggregate metadata is stored separately for callability, but
    the consent policy path is not connected to that metadata.
- Why it matters: callback surfaces are API signatures. A service can publish
  or accept handlers that are allowed to consume or produce secret values while
  the enclosing API is documented and checked as if it had no secret-bearing
  signature at all.

### BUG-044 - Exported functions can expose opaque capability tokens as forgeable ABI slots

- Area: `@export` ABI boundary, opaque `cap.io` / `cap.mem` tokens, filesystem
  and raw-memory capability checks.
- Severity: high; safe Tetra code cannot pass integer literals as capability
  tokens, but an exported symbol can accept `cap.io` or `cap.mem` parameters as
  ordinary ABI slots. A native caller can therefore supply arbitrary integer
  values at the FFI boundary while the lowered runtime operations do not inspect
  the token value.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-061/bughunt/cap_io_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-061/bughunt/consent_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-061/bughunt/export_consent_token_param_privacy_repro.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tobj /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tetra
rg -a -n "ffi_forged_fs_exists|forged_fs_exists" /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_io_param_fs_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tobj /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tetra
rg -a -n "ffi_forged_mem_load|forged_mem_load" /tmp/tetra-bug-hunt/session-061/bughunt/export_cap_mem_param_load_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-061/bughunt/cap_io_literal_param_rejected_control.tetra
```

- Observed:
  - Internal literal forge control `forged_fs_exists(0)` is rejected with
    `type mismatch for 'forged_fs_exists' arg 1`, confirming ordinary Tetra
    source cannot pass an `i32` as `cap.io`.
  - `@export("ffi_forged_fs_exists") func forged_fs_exists(io_cap: cap.io)`
    passes `check`.
  - The filesystem repro runs with a legitimate internally acquired token and
    prints `exit status 42`, confirming the exported function body is runnable.
  - `build -emit object` succeeds and the produced `.tobj` contains both
    `ffi_forged_fs_exists` and `forged_fs_exists` symbols.
  - `@export("ffi_forged_mem_load") func forged_mem_load(p: ptr, mem_cap: cap.mem)`
    passes `check`, runs with a legitimate internally acquired token and prints
    `exit status 42`.
  - `build -emit object` succeeds for the raw-memory repro and the `.tobj`
    contains both `ffi_forged_mem_load` and `forged_mem_load` symbols.
  - A direct `consent.token` literal control is still rejected, and a direct
    exported privacy round-trip is blocked by the existing exported-return
    secret-taint diagnostic. The confirmed issue here is the `cap.io`/`cap.mem`
    exported ABI surface.
- Expected: exported ABI signatures should reject opaque capability token
  parameters and returns, or lower exported wrappers that validate token
  provenance before allowing capability-gated operations. Host-callable symbols
  should not be able to manufacture a capability merely by placing an integer
  in the ABI slot.
- Evidence path:
  - `docs/spec/runtime/effects_capabilities_privacy_v1.md` says `cap.io` and
    `cap.mem` are opaque tokens that can only be obtained inside `unsafe`
    through `core.cap_io()` / `core.cap_mem()`.
  - `compiler/internal/semantics/semantics_checker.go` validates reserved export names
    and duplicate export aliases, but there is no exported-signature filter for
    `TypeCap` parameters or returns.
  - `compiler/internal/lower/lower_core.go` lowers `core.fs_exists` as a call with
    three argument slots and lowers raw memory operations to `IRMemRead*` /
    `IRMemWrite*`; capability token values are consumed for call shape, not
    validated.
  - `compiler/internal/actorsrt/actorsrt_core.go` documents
    `rdi=path_ptr, rsi=path_len, rdx=cap.io token` for `__tetra_fs_exists`,
    but the emitted helper never tests or compares `rdx`.
- Why it matters: the typechecker protects capability tokens inside Tetra
  source, but `@export` creates a native ABI where those tokens are just
  machine values. Any service exposing such a function can have its filesystem,
  MMIO, or raw-memory capability boundary bypassed by an external caller.

### BUG-045 - Exported functions can expose actor and task-group handles as forgeable ABI slots

- Area: `@export` ABI boundary, actor handles, task-group resource handles,
  and task handle ABI shape.
- Severity: high; ordinary Tetra source rejects integer literals passed as
  `actor` or `task.group`, but exported functions can publish those same
  handle types as plain ABI slots. A native caller can therefore supply
  arbitrary handle integers. Local actor send and task/task-group runtime paths
  derive scheduler table pointers directly from incoming handle slots; the
  `task.i32` exported-wrapper probe also demonstrates the same ABI exposure for
  the task handle shape already tracked by BUG-014.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-062/bughunt/export_task_i32_param_join_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-062/bughunt/actor_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-062/bughunt/task_group_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-062/bughunt/task_i32_literal_param_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-062/bughunt/actor_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tobj /tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tetra
rg -a -n "ffi_send_actor|send_actor" /tmp/tetra-bug-hunt/session-062/bughunt/export_actor_param_send_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-062/bughunt/task_group_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tobj /tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tetra
rg -a -n "ffi_close_group|close_group" /tmp/tetra-bug-hunt/session-062/bughunt/export_task_group_param_close_repro.tobj
```

- Observed:
  - Internal controls `send_actor(0, 42)`, `close_group(0)`, and
    `join_task(0)` are rejected with type mismatch diagnostics for argument 1.
  - `@export("ffi_send_actor") func send_actor(peer: actor, value: Int)`
    passes `check`, runs through a valid internal actor handle and prints
    `exit status 42`.
  - `build -emit object` succeeds for the actor repro and the produced `.tobj`
    contains both `ffi_send_actor` and `send_actor` symbols.
  - `@export("ffi_close_group") func close_group(group: task.group)` passes
    `check`, runs through a valid internal group and prints `exit status 42`.
  - `build -emit object` succeeds for the task-group repro and the `.tobj`
    contains both `ffi_close_group` and `close_group` symbols.
  - `@export("ffi_join_task") func join_task(task: task.i32)` also passes
    `check`, runs and builds, with `ffi_join_task` present in the object. This
    is an exported-ABI extension of the task-handle opacity issue in BUG-014.
- Expected: exported ABI signatures should reject opaque/resource handle
  parameters and returns, or generate host-callable wrappers that validate
  handle provenance and bounds before reaching actor/task runtime operations.
  A native caller should not be able to manufacture an `actor`, `task.group`,
  or task handle by placing an integer in the ABI slot.
- Evidence path:
  - `docs/spec/runtime/actors.md` defines `actor` as an opaque handle and states that
    sending to an invalid handle is outside the current guarantee.
  - `docs/spec/core/current_supported_surface.md` and ownership tests treat
    task-group handles as resource-lifetime handles rather than plain integers.
  - `compiler/internal/semantics/semantics_checker.go` validates reserved export names
    and duplicate export aliases, but does not filter exported signatures by
    `TypeActor`, `task.group`, or task-handle resource types.
  - `compiler/internal/lower/lower_core.go` lowers `core.task_group_close` with
    `ArgSlots: 1`, `core.task_join_i32` with `ArgSlots: 2`, and `core.send`
    with `ArgSlots: 2`, preserving raw handle slots at the runtime ABI.
  - `compiler/internal/actorsrt/actorsrt_core.go` computes actor and
    task-group pointers by shifting incoming handle values into scheduler table
    offsets. The local `emitSend` path does not bounds-check the actor handle
    before appending to the target mailbox.
- Why it matters: `@export` turns internal handle types into ambient native
  authority. Host code can target arbitrary scheduler actor slots, close or
  inspect task groups by raw index, or feed task-join paths handles that did
  not come from the corresponding runtime constructor.

### BUG-046 - Exported functions can expose island handles as forgeable arena pointers

- Area: `@export` ABI boundary, `island` handles, island arena allocation, and
  explicit island cleanup.
- Severity: high; ordinary Tetra source rejects integer literals passed as an
  `island`, but exported functions can accept `island` parameters as a raw
  one-slot native ABI value. On x64 an `island` handle is the arena base
  pointer, and `core.island_make_*` / `free(isl)` read allocator header fields
  directly from that pointer.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-063/bughunt/island_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-063/bughunt/island_free_literal_param_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-063/bughunt/island_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tobj /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tetra
rg -a -n "ffi_island_byte_roundtrip|island_byte_roundtrip" /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_slice_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-063/bughunt/island_free_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tobj /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tetra
rg -a -n "ffi_free_island|free_island" /tmp/tetra-bug-hunt/session-063/bughunt/export_island_param_free_repro.tobj
```

- Observed:
  - Internal controls `island_byte_roundtrip(0)` and `free_island(0)` are
    rejected with type mismatch diagnostics for argument 1.
  - `@export("ffi_island_byte_roundtrip") func island_byte_roundtrip(isl: island)`
    passes `check`, runs with a legitimate scoped island handle, and prints
    `exit status 42`.
  - `build -emit object` succeeds for the island allocation repro and the
    `.tobj` contains both `ffi_island_byte_roundtrip` and
    `island_byte_roundtrip`.
  - `@export("ffi_free_island") func free_island(isl: island)` passes `check`,
    runs with a legitimate manually allocated island, and prints
    `exit status 42`.
  - `build -emit object` succeeds for the island free repro and the `.tobj`
    contains both `ffi_free_island` and `free_island`.
- Expected: exported ABI signatures should reject opaque resource handles such
  as `island`, or generate wrappers that validate handle provenance and arena
  header bounds before entering island allocator/free code. Host-callable
  symbols should not be able to manufacture an island by passing an integer or
  arbitrary pointer.
- Evidence path:
  - `docs/spec/memory/islands.md` says `island` is an opaque handle pointing to the
    island base address, with a header containing `bump`, `end`, `total`, and
    `flags`.
  - `docs/spec/standard_library/stdlib.md` says opaque handles such as `ptr`, `island`,
    `actor`, `cap.io`, `cap.mem`, and `task.*` are not interchangeable even
    when they occupy one slot.
  - `compiler/internal/semantics/semantics_memory_resources.go` classifies `island` as a resource
    handle type alongside `actor`, `task.group`, and `task.i32`.
  - `compiler/internal/semantics/semantics_checker.go` validates reserved export names
    and duplicate export aliases, but does not reject exported signatures that
    contain `island`.
  - `compiler/internal/lower/lower_core.go` lowers `core.island_make_u8` to
    `IRIslandMakeSliceU8` with the incoming island slot preserved.
  - `compiler/internal/backend/x64abi/sysv_unix.go` emits `island_make_slice`
    by reading `bump` and `end` directly from the handle pointer, and emits
    `free` by reading `total` from the same pointer before `munmap`.
- Why it matters: island handles are allocator capabilities. Exposing them as
  unvalidated native ABI slots lets external callers steer arena allocation or
  cleanup toward arbitrary addresses if a Tetra service exports such a wrapper.

### BUG-047 - Exported functions can return opaque capability/resource handles as raw ABI slots

- Area: `@export` ABI boundary, opaque handle return types, capability
  minting, actor/task runtime handles, and island arena handles.
- Severity: high; ordinary Tetra source rejects returning integer literals as
  `cap.io`, `cap.mem`, `island`, `actor`, `task.group`, or `task.i32`, but
  exported functions can return those same opaque types as raw native ABI
  slots. Native callers can therefore receive capability tokens or resource
  handles that the Tetra type system treats as non-interchangeable authority.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_island_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_task_group_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/export_task_i32_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-064/bughunt/cap_io_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/cap_mem_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/island_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/actor_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/task_group_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-064/bughunt/task_i32_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-064/bughunt/cap_io_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tobj /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tetra
rg -a -n "ffi_mint_io_cap|mint_io_cap" /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_io_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-064/bughunt/cap_mem_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tobj /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tetra
rg -a -n "ffi_mint_mem_cap|mint_mem_cap" /tmp/tetra-bug-hunt/session-064/bughunt/export_cap_mem_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tobj /tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tetra
rg -a -n "ffi_spawn_peer|spawn_peer" /tmp/tetra-bug-hunt/session-064/bughunt/export_actor_return_repro.tobj
```

- Observed:
  - Literal-return controls for `cap.io`, `cap.mem`, `island`, `actor`,
    `task.group`, and `task.i32` are rejected with `return type mismatch:
    expected '<handle>', got 'i32'`.
  - `@export("ffi_mint_io_cap") func mint_io_cap() -> cap.io` passes `check`,
    runs through `core.fs_exists` with the returned token, and prints
    `exit status 42`.
  - `@export("ffi_mint_mem_cap") func mint_mem_cap() -> cap.mem` passes
    `check`, runs through raw memory store/load with the returned token, and
    prints `exit status 42`.
  - `@export("ffi_mint_island") func mint_island() -> island` passes `check`,
    runs through island allocation/use/free with the returned handle, and
    prints `exit status 42`.
  - `@export("ffi_spawn_peer") func spawn_peer() -> actor` passes `check`,
    runs through actor send/receive with the returned actor handle, and prints
    `exit status 42`.
  - `@export("ffi_open_group") func open_group() -> task.group` passes
    `check`, runs through task-group status/cancel/close, and prints
    `exit status 42`.
  - `@export("ffi_spawn_task") func spawn_task() -> task.i32` passes `check`,
    returns a two-slot task handle, joins successfully, and prints
    `exit status 42`.
  - Object builds succeed for all six repros and each `.tobj` contains the
    corresponding `ffi_*` export alias plus the internal function symbol.
- Expected: exported ABI signatures should reject opaque capability/resource
  handle returns, or expose only explicit safe wrapper APIs whose returned
  values are meaningful to native callers without becoming forgeable authority.
  A host-callable function should not be able to mint `cap.*`, `island`,
  `actor`, `task.group`, or `task.i32` as untyped ABI machine values.
- Evidence path:
  - `docs/spec/standard_library/stdlib.md` states opaque handles such as `ptr`, `island`,
    `actor`, `cap.io`, `cap.mem`, and `task.*` are not interchangeable even
    when they occupy one slot.
  - `docs/spec/memory/islands.md` defines `island` as an opaque arena base handle, and
    `docs/spec/runtime/effects_capabilities_privacy_v1.md` defines `cap.io`/`cap.mem`
    as opaque tokens obtained only inside `unsafe`.
  - `compiler/internal/semantics/semantics_checker.go` validates `@export` names and
    duplicate aliases, but does not filter exported return types by `TypeCap`,
    `TypeActor`, `TypeIsland`, or task resource handles.
  - `compiler/internal/lower/lower_core.go` copies `fn.ReturnSlots` directly into
    `ir.IRFunc.ReturnSlots` while preserving `ExportName`.
  - `compiler/internal/backend/x64obj/builder.go` assigns the same compiled
    function body and signature to both the internal symbol and export alias.
- Why it matters: even if parameter validation were added, exported return
  handles would still publish raw authority-bearing runtime values to native
  code. Those values can later be replayed into other exported functions,
  stored outside the Tetra lifetime model, or used to couple native code to
  scheduler/allocator internals.

### BUG-048 - Exported aggregate signatures hide opaque handles from ABI boundary checks

- Area: `@export` ABI boundary, aggregate signature validation, capability
  tokens, actor/task handles, optionals, structs, and enum payloads.
- Severity: high; direct opaque handle parameters and returns are already
  unsafe across native ABI boundaries, and aggregate wrappers make the same
  authority-bearing values easier to miss. Ordinary Tetra source rejects
  integer literals in `cap.io`, `actor`, `task.group?`, and `IoBox(cap.io)`
  positions, but `@export` accepts struct, enum, optional, and return
  signatures that contain those opaque handles recursively.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-065/bughunt/struct_cap_io_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/enum_actor_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/optional_task_group_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-065/bughunt/struct_cap_io_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-065/bughunt/struct_cap_io_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tobj /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tetra
rg -a -n "ffi_struct_fs_exists|struct_fs_exists" /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tobj /tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tetra
rg -a -n "ffi_send_enveloped_actor|send_enveloped_actor" /tmp/tetra-bug-hunt/session-065/bughunt/export_enum_actor_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tobj /tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tetra
rg -a -n "ffi_optional_group_status|optional_group_status" /tmp/tetra-bug-hunt/session-065/bughunt/export_optional_task_group_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tobj /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tetra
rg -a -n "ffi_mint_io_box|mint_io_box" /tmp/tetra-bug-hunt/session-065/bughunt/export_struct_cap_io_return_repro.tobj
```

- Observed:
  - `IoBox(io: 0)` is rejected with `type mismatch for field 'io'`, but
    `@export("ffi_struct_fs_exists") func struct_fs_exists(box: IoBox) -> Int`
    passes `check`, runs through `core.fs_exists` with the boxed capability,
    prints `exit status 42`, and builds an object containing the export alias.
  - `ActorEnvelope.peer(0)` is rejected because the enum payload expects
    `actor`, but
    `@export("ffi_send_enveloped_actor") func send_enveloped_actor(msg: ActorEnvelope, value: Int) -> Int`
    passes `check`, sends to the actor contained in the enum payload, prints
    `exit status 42`, and emits the host-callable alias.
  - Passing `0` to a `task.group?` parameter is rejected, but
    `@export("ffi_optional_group_status") func optional_group_status(maybe: task.group?) -> Int`
    passes `check`, reads task-group status from the optional payload, prints
    `exit status 42`, and emits the host-callable alias.
  - Returning `IoBox(io: 0)` is rejected with `type mismatch for field 'io'`,
    but `@export("ffi_mint_io_box") func mint_io_box() -> IoBox` passes
    `check`, returns a boxed `cap.io`, runs through `core.fs_exists`, prints
    `exit status 42`, and emits the host-callable alias.
  - Struct-return repros for boxed `island` and boxed `actor` were blocked by
    existing resource provenance diagnostics after reading the returned field;
    those shapes are not counted as BUG-048 reproducers.
- Expected: exported ABI validation should recursively reject aggregate
  signatures containing opaque capability/resource handles, or require an
  explicit FFI-safe representation that cannot smuggle scheduler, allocator,
  task, or capability authority through native call slots.
- Evidence path:
  - `docs/spec/standard_library/stdlib.md` states opaque handles such as `ptr`, `island`,
    `actor`, `cap.io`, `cap.mem`, and `task.*` are not interchangeable even
    when they occupy one slot.
  - `compiler/internal/semantics/semantics_memory_resources.go` already has recursive
    `typeContainsResourceHandle` logic for structs, enum payloads, arrays, and
    optionals containing `actor`, `island`, and `task.*` handles.
  - `compiler/internal/semantics/semantics_checker.go` validates export namespace,
    reserved `__tetra_` aliases, and duplicate aliases, but does not apply
    recursive handle filtering to exported parameter or return signatures and
    does not cover `TypeCap` inside aggregates.
  - Lowering/object emission preserves the `ExportName` and compiled function
    body, so the aggregate signature is published as a host-callable object
    symbol without an opaque-handle boundary check.
- Why it matters: a code review can easily miss `cap.io`, `actor`, or
  `task.group` when it is nested inside a "plain" wrapper type. Native callers
  still see a raw ABI surface, while Tetra code treats the nested value as
  authority that should only be produced and transported under the language's
  own capability and resource rules.

### BUG-049 - Exported function-typed signatures expose raw fnptr capture slots

- Area: `@export` ABI boundary, function-typed values, `fnptr`, callback
  parameters, function-typed returns, and captured closure environments.
- Severity: high; function-typed values are represented as a 9-slot `fnptr`
  payload. Ordinary Tetra source rejects integer literals as callback/function
  values, but `@export` accepts parameters and returns whose ABI metadata
  publishes those 9-slot payloads directly to native callers. For captured
  callbacks, lowering may pin the callable target to a known Tetra closure
  symbol, but it still reads hidden captured environment slots from the
  incoming `fnptr` value.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-066/bughunt/fnptr_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-066/bughunt/fnptr_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-066/bughunt/fnptr_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-066/bughunt/fnptr_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tobj /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tetra
rg -a -n "ffi_apply_captured_callback|apply_captured_callback" /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tobj /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tetra
rg -a -n "ffi_make_captured_callback|make_captured_callback" /tmp/tetra-bug-hunt/session-066/bughunt/export_fnptr_captured_return_repro.tobj
```

- Observed:
  - Passing `0` as a callback argument is rejected with
    `callback argument for 'apply_callback' must be a supported fnptr source`.
  - Returning `0` from a function-typed return is rejected with
    `function-typed return must use a supported fnptr source`.
  - `@export("ffi_apply_captured_callback") func apply_captured_callback(cb: fn(Int) -> Int, value: Int) -> Int`
    passes `check`, calls a captured closure through the incoming callback,
    prints `exit status 42`, builds an object, and emits `ffi_apply_captured_callback`
    with signature metadata `params=10 returns=1`.
  - `@export("ffi_make_captured_callback") func make_captured_callback() -> fn(Int) -> Int`
    passes `check`, returns a captured closure, prints `exit status 42`,
    builds an object, and emits `ffi_make_captured_callback` with signature
    metadata `params=0 returns=9`.
  - Non-capturing variants also pass and publish `ffi_apply_callback` as
    `params=10 returns=1` and `ffi_make_callback` as `params=0 returns=9`.
- Expected: exported native ABI signatures should reject function-typed values
  or require an explicit, validated FFI callback wrapper whose target and
  captured environment cannot be forged as ordinary integer/register slots.
  At minimum, captured callback parameters need a boundary check that incoming
  environment slots correspond to a compiler-created closure payload.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not reject `TypeRefFunction`
    in exported parameters or return types.
  - `compiler/internal/semantics/semantics_core.go` defines `fnptr` as a public pointer
    kind with `FnPtrSlotCount` slots.
  - `docs/spec/runtime/runtime_abi.md` documents 9-slot direct returns for the current
    `fnptr` callable payload slice, and
    `docs/spec/core/current_supported_surface.md` describes captured closure
    lifetime/ABI evidence through the `fnptr` fast path.
  - `compiler/internal/lower/lower_callables.go` lowers function-typed parameter
    calls by loading hidden capture slots from `local.Base + 1 + slot`; when
    there is a single known target it calls that target directly, but the
    captured environment slots still come from the incoming parameter.
  - TOBJ metadata for the repros confirms the exported aliases carry raw
    `fnptr` shapes: `ffi_apply_captured_callback` has 10 parameter slots, and
    `ffi_make_captured_callback` has 9 return slots.
- Why it matters: Tetra's closure capture checks prove that only supported,
  compiler-created values flow into function-typed storage and calls. An
  exported native boundary can bypass that construction path by supplying or
  retaining raw `fnptr` slots, including captured environment values that Tetra
  code then uses as if they came from a checked closure.

### BUG-050 - Exported aggregate signatures hide raw fnptr payloads in struct and enum slots

- Area: `@export` ABI boundary, aggregate signature validation, function-typed
  struct fields, function-typed enum payloads, function-typed aggregate
  returns, and captured closure environments.
- Severity: high; BUG-049 shows direct exported `fn(...)` signatures publish
  raw 9-slot `fnptr` payloads. Aggregate wrappers make the same surface easier
  to miss: ordinary Tetra source rejects integer literals in function-typed
  struct fields and enum payloads, but `@export` accepts structs/enums/returns
  that contain those `fnptr` slots recursively.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-067/bughunt/struct_fnptr_field_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-067/bughunt/enum_fnptr_payload_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-067/bughunt/struct_fnptr_field_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/struct_fnptr_field_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/enum_fnptr_payload_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/struct_fnptr_field_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tobj /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tetra
rg -a -n "ffi_boxed_callback_apply|boxed_callback_apply" /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tobj /tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tetra
rg -a -n "ffi_enveloped_callback_apply|enveloped_callback_apply" /tmp/tetra-bug-hunt/session-067/bughunt/export_enum_fnptr_payload_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tobj /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tetra
rg -a -n "ffi_make_callback_box|make_callback_box" /tmp/tetra-bug-hunt/session-067/bughunt/export_struct_fnptr_field_return_repro.tobj
```

- Observed:
  - `CallbackBox(cb: 0)` is rejected with
    `function-typed struct field 'box.cb' initializer must be a supported fnptr source`.
  - `CallbackEnvelope.call(0)` is rejected with
    `function-typed enum payload 'CallbackEnvelope.call[1]' initializer must be a supported fnptr source`.
  - Returning `CallbackBox(cb: 0)` is rejected with the same supported-`fnptr`
    source diagnostic for `CallbackBox.cb`.
  - `@export("ffi_boxed_callback_apply") func boxed_callback_apply(box: CallbackBox, value: Int) -> Int`
    passes `check`, calls the captured callback stored in the struct field,
    prints `exit status 42`, builds an object, and emits an export alias with
    TOBJ metadata `params=10 returns=1`.
  - `@export("ffi_enveloped_callback_apply") func enveloped_callback_apply(env: CallbackEnvelope, value: Int) -> Int`
    passes `check`, calls the captured callback stored in the enum payload,
    prints `exit status 42`, builds an object, and emits an export alias with
    TOBJ metadata `params=11 returns=1`.
  - `@export("ffi_make_callback_box") func make_callback_box() -> CallbackBox`
    passes `check`, returns a struct containing a captured callback, prints
    `exit status 42`, builds an object, and emits an export alias with TOBJ
    metadata `params=0 returns=9`.
- Expected: exported ABI validation should recursively reject aggregate
  signatures containing function-typed values, or require a dedicated FFI
  callback representation with validated target/environment provenance. The
  rule needs to inspect struct fields, enum payloads, and aggregate returns,
  not only direct `fn(...)` parameter and return positions.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not reject
    aggregate-contained `TypeRefFunction` values in exported signatures.
  - `compiler/internal/semantics/semantics_expressions.go` has dedicated metadata
    paths for function-typed struct fields and enum payloads, proving those
    fields are semantically special even though they lower into `fnptr` slots.
  - `compiler/internal/lower/lower_callables.go` lowers stored function calls by
    reading hidden capture slots from `fnptrBase + 1 + slot`, so a native
    caller that supplies the aggregate supplies the callback environment slots.
  - TOBJ metadata confirms the exported aggregate aliases carry the flattened
    raw `fnptr` payload shape: 10 parameter slots for a boxed callback plus an
    `Int`, 11 parameter slots for enum tag plus callback plus `Int`, and 9
    return slots for a boxed callback.
- Why it matters: aggregate types look like normal service request/response
  DTOs, but here they contain executable callback state. A native caller can
  replay or forge the flattened closure environment through a struct or enum
  boundary, bypassing the source-level supported-`fnptr` construction rules.

### BUG-051 - Exported String and slice signatures expose forgeable ptr,len views

- Area: `@export` ABI boundary, `String`/`str`, slices, two-slot `ptr,len`
  metadata, indexed loads/stores, and lifetime/bounds provenance.
- Severity: high; ordinary Tetra source rejects integer literals where a
  `String` or `[]u8` value is required, but exported signatures flatten those
  views into raw native ABI slots. A native caller can therefore supply an
  arbitrary pointer and length to Tetra code that indexes the value as a checked
  `String`/slice, while exported returns publish internal string/slice views
  back to the host as the same raw slot pair.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-068/bughunt/string_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/slice_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/string_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-068/bughunt/slice_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/string_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/slice_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/string_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/slice_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tobj /tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tetra
rg -a -n "ffi_string_first_byte|string_first_byte" /tmp/tetra-bug-hunt/session-068/bughunt/export_string_param_index_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tobj /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tetra
rg -a -n "ffi_slice_first_byte|slice_first_byte" /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_param_index_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tobj /tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tetra
rg -a -n "ffi_make_string|make_string" /tmp/tetra-bug-hunt/session-068/bughunt/export_string_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tobj /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tetra
rg -a -n "ffi_make_slice|make_slice" /tmp/tetra-bug-hunt/session-068/bughunt/export_slice_return_repro.tobj
```

- Observed:
  - Passing `0` as a `String` parameter is rejected with
    `type mismatch for 'string_first_byte' arg 1`.
  - Passing `0` as a `[]u8` parameter is rejected with
    `type mismatch for 'slice_first_byte' arg 1`.
  - Returning `0` from `String` and `[]u8` functions is rejected with
    `return type mismatch: expected 'str', got 'i32'` and
    `return type mismatch: expected '[]u8', got 'i32'`.
  - `@export("ffi_string_first_byte") func string_first_byte(text: String) -> Int`
    passes `check`, indexes `text[0]`, prints `exit status 42`, builds an
    object, and emits TOBJ metadata `params=2 returns=1`.
  - `@export("ffi_slice_first_byte") func slice_first_byte(bytes: []u8) -> Int`
    passes `check`, indexes `bytes[0]`, prints `exit status 42`, builds an
    object, and emits TOBJ metadata `params=2 returns=1`.
  - `@export("ffi_make_string") func make_string() -> String` passes `check`,
    returns a string literal view, prints `exit status 42`, builds an object,
    and emits TOBJ metadata `params=0 returns=2`.
  - `@export("ffi_make_slice") func make_slice() -> []u8` passes `check`,
    returns a heap slice view, prints `exit status 42`, builds an object, and
    emits TOBJ metadata `params=0 returns=2`.
- Expected: exported ABI validation should reject raw view types such as
  `str`/`String` and slices, or require explicit FFI buffer types whose pointer,
  length, ownership, mutability, and lifetime are validated or copied at the
  boundary. Return paths should similarly require an explicit borrowed/owned
  buffer contract rather than publishing internal `ptr,len` views.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not reject `TypeStr`,
    `TypeSlice`, or aggregate two-slot view types in exported signatures.
  - `compiler/internal/semantics/semantics_core.go` builds slices with public `ptr` and
    `len` fields and `SlotCount: 2`; `String`/`str` reuses the same slice shape
    over `u8`.
  - `docs/spec/standard_library/stdlib.md` documents `String`/`str` as a two-slot UTF-8
    string/slice shape and says `str`, `[]u8`, `[]u16`, `[]i32`, and `[]bool`
    are two-slot values (`ptr`, `len`).
  - `docs/spec/runtime/runtime_abi.md` already distinguishes explicit host `ptr,len`
    filesystem paths from language-level string handling, which shows this
    boundary needs a deliberate ABI contract rather than accidental flattening.
  - TOBJ metadata confirms the exported aliases carry raw two-slot shapes:
    `ffi_string_first_byte` and `ffi_slice_first_byte` have two parameter slots;
    `ffi_make_string` and `ffi_make_slice` have two return slots.
- Why it matters: BUG-030 and BUG-031 show that forged or mutated view metadata
  is memory-sensitive once inside Tetra. This exported boundary gives native
  callers that same `ptr,len` authority directly, bypassing the source-level
  construction and type checks that normally prevent integers from becoming
  `String` or slice values.

### BUG-052 - Exported aggregate signatures hide String and slice ptr,len views

- Area: `@export` ABI boundary, aggregate signature validation, `String`/`str`,
  slices, struct fields, enum payloads, optionals, and aggregate returns.
- Severity: high; BUG-051 shows direct exported `String` and slice signatures
  publish raw two-slot `ptr,len` views. Aggregate wrappers make the same ABI
  surface easier to miss: ordinary Tetra source rejects integer literals in
  `String`/slice fields and enum payloads, but `@export` accepts structs,
  enums, optionals, and returns that contain those view slots recursively.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-069/bughunt/struct_string_field_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/struct_slice_field_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/enum_string_payload_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/optional_string_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/struct_string_field_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-069/bughunt/struct_slice_field_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/struct_string_field_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/struct_slice_field_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/enum_string_payload_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/optional_string_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/struct_string_field_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/struct_slice_field_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tetra
rg -a -n "ffi_boxed_string_first_byte|boxed_string_first_byte" /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tetra
rg -a -n "ffi_boxed_slice_first_byte|boxed_slice_first_byte" /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tetra
rg -a -n "ffi_enveloped_string_first_byte|enveloped_string_first_byte" /tmp/tetra-bug-hunt/session-069/bughunt/export_enum_string_payload_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tetra
rg -a -n "ffi_optional_string_first_byte|optional_string_first_byte" /tmp/tetra-bug-hunt/session-069/bughunt/export_optional_string_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tetra
rg -a -n "ffi_make_string_box|make_string_box" /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_string_field_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tobj /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tetra
rg -a -n "ffi_make_slice_box|make_slice_box" /tmp/tetra-bug-hunt/session-069/bughunt/export_struct_slice_field_return_repro.tobj
```

- Observed:
  - `TextBox(text: 0)` and `BytesBox(bytes: 0)` are rejected with
    `type mismatch for field 'text'` and `type mismatch for field 'bytes'`.
  - `TextEnvelope.text(0)` is rejected with
    `enum case 'TextEnvelope.text' payload 1 expects 'str', got 'i32'`.
  - Passing `0` as a `String?` parameter is rejected with
    `type mismatch for 'optional_string_first_byte' arg 1`.
  - Returning `TextBox(text: 0)` and `BytesBox(bytes: 0)` is rejected with the
    same field mismatch diagnostics.
  - Exported struct parameters containing `String` and `[]u8` pass `check`,
    print `exit status 42`, build objects, and emit `ffi_boxed_string_first_byte`
    and `ffi_boxed_slice_first_byte` with TOBJ metadata `params=2 returns=1`.
  - The exported enum payload case passes `check`, prints `exit status 42`,
    builds an object, and emits `ffi_enveloped_string_first_byte` with metadata
    `params=3 returns=1` (enum tag plus `ptr,len` string payload).
  - The exported optional `String?` parameter passes `check`, prints
    `exit status 42`, builds an object, and emits
    `ffi_optional_string_first_byte` with metadata `params=3 returns=1`
    (presence tag plus `ptr,len` string payload).
  - Exported struct returns containing `String` and `[]u8` pass `check`, print
    `exit status 42`, build objects, and emit `ffi_make_string_box` and
    `ffi_make_slice_box` with TOBJ metadata `params=0 returns=2`.
- Expected: exported ABI validation should recursively reject aggregate
  signatures containing raw view types, or require explicit FFI-safe buffer
  wrappers whose pointer, length, ownership, mutability, and lifetime contracts
  are visible at the boundary. The rule needs to inspect struct fields, enum
  payloads, optionals, and aggregate returns, not only direct parameter and
  return positions.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not recursively inspect
    exported signatures for `TypeStr`, `TypeSlice`, or optional/aggregate view
    payloads.
  - `compiler/internal/semantics/semantics_core.go` builds slices with public `ptr` and
    `len` fields and `SlotCount: 2`, then computes struct slot offsets by
    summing field `SlotCount`, so a `String`/slice field contributes two raw
    slots to the aggregate ABI.
  - `docs/spec/standard_library/stdlib.md` documents `String`/`str` and slices as two-slot
    values (`ptr`, `len`) and says `T?` adds one presence tag slot to the
    payload slots.
  - TOBJ metadata confirms the flattened raw slot shapes for struct fields,
    enum payloads, optionals, and aggregate returns.
- Why it matters: DTO-shaped service boundaries are where developers naturally
  place request bodies, byte buffers, and response payloads. A code review can
  miss a `String` or `[]u8` nested inside a wrapper type, but native callers
  still get the same authority to supply or retain unchecked `ptr,len` views
  that Tetra code treats as language-constructed, bounds-checked data.

### BUG-053 - Exported fixed-array signatures expose raw ptr,len metadata slots

- Area: `@export` ABI boundary, fixed arrays (`[N]T`), `TypeArray`, `ptr`/`len`
  metadata, and source-level fixed-array internal-field protections.
- Severity: high; Tetra source rejects integer literals where `[1]Int` is
  expected and explicitly rejects assignment to fixed-array `ptr`/`len`, but
  exported signatures still flatten fixed arrays into two native ABI slots. A
  host caller can therefore supply arbitrary pointer and length metadata for a
  value that Tetra code treats as a fixed-size array.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_return_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_internal_len_assignment_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/fixed_array_internal_len_assignment_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tobj /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tetra
rg -a -n "ffi_fixed_array_len|fixed_array_len" /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_len_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tobj /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tetra
rg -a -n "ffi_fixed_array_first|fixed_array_first" /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_param_index_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tobj /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tetra
rg -a -n "ffi_echo_fixed_array|echo_fixed_array" /tmp/tetra-bug-hunt/session-070/bughunt/export_fixed_array_echo_return_repro.tobj
```

- Observed:
  - Passing `0` as a `[1]Int` parameter is rejected with
    `type mismatch for 'fixed_array_len' arg 1`.
  - Returning `0` from a `[1]Int` function is rejected with
    `return type mismatch: expected '[1]i32', got 'i32'`.
  - Assigning `xs.len = 2` in Tetra source is rejected with
    `cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead`.
  - `@export("ffi_fixed_array_len") func fixed_array_len(xs: [1]Int) -> Int`
    passes `check`, builds an object, and emits TOBJ metadata
    `params=2 returns=1`. The function body reads `xs.len`, proving the
    incoming length slot is part of the callable value shape.
  - `@export("ffi_fixed_array_first") func fixed_array_first(xs: [1]Int) -> Int`
    passes `check`, builds an object, and emits TOBJ metadata
    `params=2 returns=1`. The function body indexes `xs[0]`, so native-supplied
    array metadata feeds the normal fixed-array index path.
  - `@export("ffi_echo_fixed_array") func echo_fixed_array(xs: [1]Int) -> [1]Int`
    passes `check`, builds an object, and emits TOBJ metadata
    `params=2 returns=2`, publishing the same raw view back to native callers.
  - The repro `main` functions intentionally return `0` without constructing a
    fixed array internally. BUG-024 already tracks that zeroed fixed-array fields
    trap on index access, so this evidence focuses on the exported ABI surface
    rather than conflating it with the existing fixed-array storage bug.
- Expected: exported ABI validation should reject fixed-array parameters and
  returns until `[N]T` has a stable FFI representation, or require an explicit
  FFI array/buffer contract that does not expose mutable `ptr`/`len` metadata.
  In particular, a fixed-size `[1]Int` boundary should not trust a host-provided
  length slot that can disagree with the static array length.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` models `TypeArray` with public `ptr`
    and `len` fields and `SlotCount: 2`.
  - `compiler/internal/semantics/semantics_core.go` has a targeted
    `rejectFixedArrayInternalAssignment` guard for `ptr`/`len`, showing those
    metadata slots are sensitive enough to protect from Tetra assignment.
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not reject `TypeArray` in
    exported signatures.
  - `compiler/internal/semantics/semantics_expressions.go` and `compiler/internal/lower/lower_core.go`
    both route `TypeArray` through the ordinary index element path, so an
    exported function that indexes a fixed array consumes the incoming view
    metadata.
  - TOBJ metadata confirms exported fixed-array aliases carry raw two-slot
    parameter and return shapes.
- Why it matters: fixed arrays are meant to model static protocol headers and
  small in-memory records. If exposed over FFI as a forgeable `ptr,len` pair, a
  native caller can violate the fixed length invariant before Tetra code reads
  the `len` field or indexes the array, bypassing the source-level protections
  that block direct metadata mutation.

### BUG-054 - Exported aggregate signatures hide fixed-array ptr,len metadata

- Area: `@export` ABI boundary, aggregate signature validation, fixed arrays
  (`[N]T`), `TypeArray`, struct fields, enum payloads, optionals, and aggregate
  returns.
- Severity: high; BUG-053 shows direct exported `[N]T` signatures publish raw
  two-slot `ptr,len` array views. Aggregate wrappers hide the same unsafe shape
  inside DTO-looking values: ordinary Tetra source rejects integer literals in
  fixed-array fields, enum payloads, and optional arguments, but `@export`
  accepts the aggregate signatures and flattens the nested arrays into native
  slots.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-071/bughunt/struct_fixed_array_field_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/enum_fixed_array_payload_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/optional_fixed_array_literal_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-071/bughunt/struct_fixed_array_field_return_literal_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/struct_fixed_array_field_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/enum_fixed_array_payload_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/optional_fixed_array_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/struct_fixed_array_field_return_literal_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tetra
rg -a -n "ffi_boxed_fixed_array_len|boxed_fixed_array_len" /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tetra
rg -a -n "ffi_boxed_fixed_array_first|boxed_fixed_array_first" /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_index_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tetra
rg -a -n "ffi_enveloped_fixed_array_len|enveloped_fixed_array_len" /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tetra
rg -a -n "ffi_optional_fixed_array_len|optional_fixed_array_len" /tmp/tetra-bug-hunt/session-071/bughunt/export_optional_fixed_array_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tetra
rg -a -n "ffi_make_fixed_array_box|make_fixed_array_box" /tmp/tetra-bug-hunt/session-071/bughunt/export_struct_fixed_array_field_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tobj /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tetra
rg -a -n "ffi_wrap_fixed_array_envelope|wrap_fixed_array_envelope" /tmp/tetra-bug-hunt/session-071/bughunt/export_enum_fixed_array_payload_return_repro.tobj
```

- Observed:
  - `ArrayBox(items: 0)` is rejected with `type mismatch for field 'items'`.
  - `ArrayEnvelope.items(0)` is rejected with
    `enum case 'ArrayEnvelope.items' payload 1 expects '[1]i32', got 'i32'`.
  - Passing `0` as a `[1]Int?` parameter is rejected with
    `type mismatch for 'optional_fixed_array_len' arg 1`.
  - Returning `ArrayBox(items: 0)` is rejected with
    `type mismatch for field 'items'`.
  - Exported struct parameters containing `[1]Int` pass `check`, build objects,
    and emit `ffi_boxed_fixed_array_len` and `ffi_boxed_fixed_array_first` with
    TOBJ metadata `params=2 returns=1`. The second function indexes
    `box.items[0]`, so the nested host-supplied array view reaches the normal
    fixed-array index path.
  - The exported enum payload parameter passes `check`, builds an object, and
    emits `ffi_enveloped_fixed_array_len` with metadata `params=3 returns=1`
    (enum tag plus array `ptr,len`).
  - The exported optional fixed-array parameter passes `check`, builds an
    object, and emits `ffi_optional_fixed_array_len` with metadata
    `params=3 returns=1` (presence tag plus array `ptr,len`).
  - Exported struct and enum returns containing `[1]Int` pass `check`, build
    objects, and emit `ffi_make_fixed_array_box` as `params=0 returns=2` and
    `ffi_wrap_fixed_array_envelope` as `params=2 returns=3`.
  - Each repro `main` intentionally exits `0` without constructing or indexing
    a fixed array internally, so this finding stays separate from BUG-024's
    zeroed fixed-array storage trap.
- Expected: exported ABI validation should recursively reject aggregate
  signatures containing fixed arrays until `[N]T` has a stable FFI
  representation, or require explicit FFI array/buffer wrappers that do not
  expose mutable `ptr,len` metadata. The rule needs to inspect struct fields,
  enum payloads, optionals, and aggregate returns.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` models `TypeArray` with public `ptr`
    and `len` fields and `SlotCount: 2`, then computes struct slot offsets by
    summing field slot counts.
  - `compiler/tests/semantics/semantics_core_language_test.go` has build smoke coverage for
    fixed arrays inside structs and optionals, so these aggregate surfaces are
    intentionally accepted by the compiler.
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not recursively inspect
    exported signatures for nested `TypeArray`.
  - TOBJ metadata confirms the flattened raw slot shapes for struct fields,
    enum payloads, optionals, and aggregate returns.
- Why it matters: fixed arrays are supposed to express static size and are often
  embedded in protocol/header DTOs. Once a native caller can smuggle a fixed
  array through a wrapper type as `ptr,len`, Tetra code may read `len` or index
  the array while trusting metadata that the language itself prevents Tetra code
  from forging.

### BUG-055 - Exported Bool signatures expose unnormalized truth slots

- Area: `@export` ABI boundary, `Bool`/`bool`, one-slot scalar ABI metadata,
  struct fields, enum payloads, optionals, and truthiness in control flow.
- Severity: medium; ordinary Tetra source rejects integer literals where `Bool`
  is required, but exported signatures publish `Bool` positions as ordinary
  one-slot ABI values. Native callers can supply any nonzero integer for a
  boolean gate, and lowering branches with `test eax,eax`, so non-canonical
  truth values are treated as true even though Tetra code cannot construct them
  as `Bool`.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-072/bughunt/bool_int_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/bool_return_int_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/struct_bool_field_int_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/enum_bool_payload_int_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-072/bughunt/optional_bool_int_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/bool_int_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/bool_return_int_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/struct_bool_field_int_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/enum_bool_payload_int_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/optional_bool_int_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tobj /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tetra
rg -a -n "ffi_bool_gate|bool_gate" /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tobj /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tetra
rg -a -n "ffi_is_ready|is_ready" /tmp/tetra-bug-hunt/session-072/bughunt/export_bool_return_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tobj /tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tetra
rg -a -n "ffi_boxed_bool_gate|boxed_bool_gate" /tmp/tetra-bug-hunt/session-072/bughunt/export_struct_bool_field_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tobj /tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tetra
rg -a -n "ffi_enveloped_bool_gate|enveloped_bool_gate" /tmp/tetra-bug-hunt/session-072/bughunt/export_enum_bool_payload_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tobj /tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tetra
rg -a -n "ffi_optional_bool_gate|optional_bool_gate" /tmp/tetra-bug-hunt/session-072/bughunt/export_optional_bool_param_repro.tobj
```

- Observed:
  - Passing `1` as a direct `Bool` parameter is rejected with
    `type mismatch for 'bool_gate' arg 1`.
  - Returning `1` from a `Bool` function is rejected with
    `return type mismatch: expected 'bool', got 'i32'`.
  - `Gate(allow: 1)`, `GateMsg.allow(1)`, and passing `1` as `Bool?` are
    rejected with field, enum payload, and optional-argument type diagnostics.
  - `@export("ffi_bool_gate") func bool_gate(flag: Bool) -> Int` passes `check`,
    runs with a valid internal `true` and prints `exit status 42`, builds an
    object, and emits TOBJ metadata `params=1 returns=1`.
  - `@export("ffi_is_ready") func is_ready() -> Bool` passes `check`, returns a
    valid `true`, prints `exit status 42`, builds an object, and emits TOBJ
    metadata `params=0 returns=1`.
  - Exported struct-field, enum-payload, and optional `Bool` parameters pass
    `check`, print `exit status 42`, build objects, and emit metadata
    `params=1 returns=1` for the struct field and `params=2 returns=1` for the
    tag-plus-`Bool` enum/optional shapes.
- Expected: exported ABI metadata should preserve scalar type information or
  validate/normalize boolean inputs at the boundary. A `Bool` parameter should
  not be indistinguishable from an arbitrary one-slot integer in the exported
  object record, especially for gate/authorization-style service APIs.
- Evidence path:
  - `compiler/internal/semantics/semantics_core.go` defines `bool` as `TypeBool` with
    `SlotCount: 1`, while `typesCompatible` and the controls reject integer
    literals in `Bool` positions.
  - `compiler/internal/semantics/semantics_checker.go` validates only export namespace,
    reserved aliases, and duplicate aliases; it does not validate or annotate
    exported `Bool` signatures.
  - `compiler/internal/lower/lower_core.go` lowers `if flag` to a one-slot condition,
    and `compiler/internal/backend/x64core/x64core_core.go` implements conditional jumps
    with `test eax,eax`, so any nonzero incoming slot behaves as true.
  - `compiler/internal/format/tobj/object.go` stores exported symbol signatures
    as only `ParamSlots` and `ReturnSlots`, not scalar types, ranges, or boolean
    normalization requirements.
- Why it matters: services often expose boolean gates such as `allow`, `is_admin`,
  `enabled`, or `confirmed`. Tetra source preserves a strict `Bool` invariant,
  but the exported ABI erases that invariant and lets host callers drive boolean
  control flow with arbitrary integer payloads.

### BUG-056 - Guarded default arms make `match`/`catch` expressions falsely exhaustive

- Area: expression exhaustiveness, guarded `case _`, `match` expressions,
  `catch` expressions, typed-error fallback handling, and result-local
  initialization.
- Severity: high; `tetra check` correctly rejects a guarded concrete case as
  non-exhaustive, but accepts `case _ if false` as an exhaustive default in
  both `match` and `catch` expressions. Lowering then lets the guard fail and
  jumps to the expression end without storing the result local, so checked code
  can observe the zero/default slot instead of receiving an exhaustiveness
  diagnostic.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_default_false_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_false_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_case_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_case_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-073/bughunt/match_unguarded_default_control.tetra`
  - `/tmp/tetra-bug-hunt/session-073/bughunt/catch_unguarded_default_control.tetra`
  - `/tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_true_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_case_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_case_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/match_unguarded_default_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-073/bughunt/match_unguarded_default_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/catch_unguarded_default_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-073/bughunt/catch_unguarded_default_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_true_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_true_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_default_false_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_default_false_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_default_false_repro.bin /tmp/tetra-bug-hunt/session-073/bughunt/match_guarded_default_false_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_false_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_false_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_false_repro.bin /tmp/tetra-bug-hunt/session-073/bughunt/catch_guarded_default_false_repro.tetra
```

- Observed:
  - `match 7: case 7 if true: ...` is rejected with
    `match expression must be exhaustive`, proving guarded concrete cases do not
    count as exhaustive.
  - `catch read(): case ReadError.denied(code) if code > 0: ...` is rejected
    with `catch expression must be exhaustive`, proving the same control path
    for typed-error payload cases.
  - Unguarded `case _:` controls for both `match` and `catch` pass `check` and
    print `exit status 42`.
  - `catch read(): case _ if true: ...` passes `check` and prints
    `exit status 42`, showing guarded defaults are accepted at all.
  - `match 7: case _ if false: 99` passes `check`, builds, and prints
    `exit status 42` only because the caller checks that the expression result
    became `0` after no arm stored `99`.
  - `catch read(): case _ if false: 99` passes `check`, builds, and prints
    `exit status 42` through the same zero-result path after the thrown
    `ReadError.denied(7)` skips the guarded default body.
- Expected: a guarded default arm must not satisfy expression exhaustiveness.
  Either `case _ if ...` should require another unguarded fallback arm, or the
  lowering must define a checked failure path instead of exposing an unstored
  result local.
- Evidence path:
  - `compiler/internal/semantics/semantics_expressions.go` checks `match` expression
    exhaustiveness by calling the complete optional/enum helpers, then falls
    back to a `hasDefault` loop that treats any default arm as exhaustive without
    checking `c.Guard == nil`.
  - The same file's `checkCatchExpr` uses the same fallback shape for `catch`
    expression exhaustiveness.
  - The dedicated complete-pattern helpers skip guarded cases, and
    `compiler/internal/semantics/semantics_checker.go` uses a stricter `matchHasDefault`
    helper for match statements, so the hole is specific to expression fallback
    default detection.
  - `compiler/internal/lower/lower_core.go` lowers guarded expression arms by storing
    into the result local only after the guard passes; when a guarded default
    guard fails, control reaches the shared end label and reloads the result
    local without any arm assignment.
- Why it matters: `match`/`catch` expressions are natural service routing and
  typed-error recovery constructs. A guarded fallback that is false for a
  particular request/error should be treated as non-exhaustive; silently
  producing `0` can bypass error handling, route to a default success value, or
  mask missing authorization/error cases.

### BUG-057 - Exported enum signatures expose forgeable discriminant slots

- Area: `@export` ABI boundary, enum discriminants, enum payload tags,
  exhaustive `match` statements, and TOBJ slot-only signatures.
- Severity: high; ordinary Tetra source rejects integer literals where an enum
  is required, but exported enum parameters are published as raw one-slot
  discriminants. A native caller can pass an impossible tag value that no Tetra
  enum constructor can produce. Exhaustive `match` statements over such a value
  have no default arm, so lowering falls through to the shared end label without
  running any case body.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-074/bughunt/enum_int_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-074/bughunt/enum_int_return_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-074/bughunt/enum_int_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-074/bughunt/enum_int_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tobj /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tetra
rg -a -n "ffi_route_decision|route_decision" /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_tag_param_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tobj /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tetra
rg -a -n "ffi_request_decision|request_decision" /tmp/tetra-bug-hunt/session-074/bughunt/export_enum_payload_tag_param_repro.tobj
```

- Observed:
  - Passing `99` to a `Route` parameter is rejected with
    `type mismatch for 'route_decision' arg 1`.
  - Returning `99` from a `-> Route` function is rejected with
    `return type mismatch: expected 'Route', got 'i32'`.
  - `@export("ffi_route_decision") func route_decision(route: Route) -> Int`
    passes `check`, prints `exit status 42` for a valid internal
    `Route.admin` call, builds an object, and emits `ffi_route_decision` as
    `params=1 returns=1`.
  - `@export("ffi_request_decision") func request_decision(req: Request) -> Int`
    for an enum with `read(Int)` and `admin(Int)` payload cases passes
    `check`, prints `exit status 42` for a valid internal call, builds an
    object, and emits `ffi_request_decision` as `params=2 returns=1` (raw tag
    plus one payload slot).
  - The exported repros initialize `decision` to `42` before an exhaustive
    `match`. From the lowering source, an impossible host-supplied tag would
    fail every generated tag comparison and jump to the match end label, leaving
    the pre-match value unchanged.
- Expected: exported enum parameters should either be rejected at the raw ABI
  boundary, carry type metadata that a host binding must enforce, or emit an
  entry validation guard that rejects tags outside the declared enum cases before
  Tetra control flow sees them.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates `@export` names for
    namespace/reserved/duplicate rules only; it does not reject enum parameter
    or return types and does not attach discriminant validation metadata.
  - `compiler/internal/format/tobj/object.go` stores exported symbol signatures
    as `ParamSlots` and `ReturnSlots` only, so `Route` is indistinguishable from
    an arbitrary one-slot integer to a native caller.
  - `compiler/internal/semantics/semantics_checker.go` and `exprs.go` treat enum matches
    as exhaustive when every declared case appears, so no source-level default
    is required for invalid runtime tags.
  - `compiler/internal/lower/lower_core.go` lowers enum `match` by comparing the raw
    tag slot to each case ordinal. If no case matches and there is no default,
    lowering emits a jump to the end label.
- Why it matters: exported functions are service/FFI boundaries. Route,
  command, state, and permission enums are commonly used as closed sets; erasing
  the discriminant invariant lets an untrusted host caller inject states that
  checked Tetra code assumes are impossible, bypassing exhaustive routing and
  leaving pre-existing state or default-initialized results in place.

### BUG-058 - Exported optional signatures expose forgeable presence tags

- Area: `@export` ABI boundary, optional `T?` presence tags, `match`/`if let`
  unwrapping, and TOBJ slot-only signatures.
- Severity: high; ordinary Tetra source constructs `T?` values only through
  `none` or implicit `some` packing, and lowering emits presence tags `0` or
  `1`. Exported optional parameters publish the optional as raw slots, so a
  native caller can supply a non-canonical presence tag such as `2`. Optional
  `some` checks treat any nonzero tag as present, so Tetra code sees a forged
  optional state that the source language cannot construct.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-075/bughunt/optional_none_control.tetra`
  - `/tmp/tetra-bug-hunt/session-075/bughunt/optional_implicit_some_control.tetra`
  - `/tmp/tetra-bug-hunt/session-075/bughunt/optional_tag_field_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-075/bughunt/optional_none_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-075/bughunt/optional_none_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-075/bughunt/optional_implicit_some_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-075/bughunt/optional_implicit_some_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-075/bughunt/optional_tag_field_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tobj /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tetra
rg -a -n "ffi_optional_status|optional_status" /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_match_repro.tobj
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tobj /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tetra
rg -a -n "ffi_optional_iflet|optional_iflet" /tmp/tetra-bug-hunt/session-075/bughunt/export_optional_int_iflet_repro.tobj
```

- Observed:
  - `optional_status(none)` passes `check` and prints `exit status 42` through
    the `none` arm.
  - `optional_status(42)` passes `check` and prints `exit status 42`; lowering
    uses the ordinary implicit-`some` path, which appends presence tag `1`.
  - `var maybe: Int? = 42; maybe.tag = 2` is rejected with
    `'i32?' is not a struct`, so source code cannot mutate a public optional
    tag field the way it can mutate slice/string metadata.
  - `@export("ffi_optional_status") func optional_status(code: Int?) -> Int`
    passes `check`, prints `exit status 42` for a valid internal call, builds an
    object, and emits `ffi_optional_status` as `params=2 returns=1`.
  - The `if let some(value) = code` exported variant also passes `check`, prints
    `exit status 42`, builds an object, and emits `ffi_optional_iflet` as
    `params=2 returns=1`.
- Expected: exported optional parameters should reject raw ABI exposure, carry
  type/tag metadata for host bindings, or validate the presence tag at the entry
  boundary so only `0` and `1` are accepted.
- Evidence path:
  - `docs/spec/flow/flow_syntax_v1.md` defines optional layout as a presence tag
    followed by payload slots.
  - `compiler/internal/lower/lower_core.go` `lowerExprAs` emits `0` for `none` and
    `1` for implicit optional payload packing.
  - The same file lowers optional `match` and `if let some(...)` checks with
    `IRJmpIfZero`, so any nonzero presence tag is treated as `some`.
  - `compiler/internal/semantics/semantics_checker.go` validates `@export` names but does
    not reject optional parameter types or add presence-tag validation metadata.
  - `compiler/internal/format/tobj/object.go` records only parameter and return
    slot counts, making `Int?` indistinguishable from two arbitrary integer
    slots to a native caller.
- Why it matters: optional parameters are common for nullable account IDs,
  feature flags, quotas, and partial request fields. At an FFI/service boundary,
  non-canonical optional tags can force Tetra code down a `some` branch with an
  attacker-controlled payload even though checked Tetra source only creates
  canonical `none`/`some` values.

### BUG-059 - Exported consent-token signatures expose forgeable policy slots

- Area: `@export` ABI boundary, `consent.token`, privacy policy guards, and
  secret-bearing exported signatures.
- Severity: high; ordinary Tetra source rejects integer literals passed as
  `consent.token`, but an exported function can publish a `consent.token`
  parameter as a one-slot native ABI value. Lowering validates the parameter by
  comparing it to a fixed sentinel embedded in the generated code, and TOBJ
  metadata records only slot counts. A native caller can therefore satisfy the
  consent guard by supplying the sentinel slot value, bypassing source-level
  token provenance at the FFI boundary. Existing secret-taint diagnostics still
  block direct exported secret returns, throws, branch-selected public returns,
  and global stores; the confirmed issue is the policy-token ABI boundary.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_consent_token_guard_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_ignore_probe.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_discard_probe.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-076/bughunt/consent_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/secret_literal_param_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_return_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_branch_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_global_store_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/consent_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/secret_literal_param_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_consent_token_guard_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-076/bughunt/export_consent_token_guard_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-076/bughunt/export_consent_token_guard_repro.tobj /tmp/tetra-bug-hunt/session-076/bughunt/export_consent_token_guard_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_ignore_probe.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_ignore_probe.tobj /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_ignore_probe.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_discard_probe.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_discard_probe.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_discard_probe.tobj /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_discard_probe.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_unseal_return_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_branch_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-076/bughunt/export_secret_param_global_store_rejected_control.tetra
```

- Observed:
  - `require_consent(1)` is rejected with `type mismatch for
    'require_consent' arg 1`, confirming ordinary Tetra source cannot pass an
    integer as `consent.token`.
  - `consume(token, 7)` where the second parameter is `secret.i32` is rejected
    with `type mismatch for 'consume' arg 2`, confirming ordinary source cannot
    forge a secret wrapper from an integer.
  - `@export("ffi_require_consent") func require_consent(token:
    consent.token) -> Int uses privacy privacy consent(token)` passes `check`,
    runs with an internally minted token and prints `exit status 42`.
  - The object build succeeds and the TOBJ symbol table records
    `ffi_require_consent` and `require_consent` as `params=1 returns=1`; the
    generated code contains the consent sentinel bytes at offsets `27` and
    `93`.
  - `@export("ffi_accept_secret") func accept_secret(token: consent.token,
    value: secret.i32) -> Int` passes `check` and emits both symbols as
    `params=2 returns=1`, showing secret-bearing exported signatures also cross
    the ABI as plain slots when the body does not leak the value.
  - `@export("ffi_unseal_discard")` can unseal and discard a `secret.i32`
    parameter, passes `check`, prints `exit status 42`, and emits both symbols
    as `params=2 returns=1`; this confirms the exported policy guard is the only
    runtime boundary before a host-supplied secret slot reaches the unseal path.
  - Exported secret returns, unsealed public returns, branch-selected public
    returns, and global stores are rejected by existing diagnostics:
    `secret-tainted value cannot be returned from @export function ...` and
    `secret-tainted value cannot be stored in global 'leaked'`.
- Expected: exported ABI signatures should reject `consent.token` parameters
  and returns, or exported wrappers should carry a non-forgeable token
  provenance mechanism instead of accepting a raw slot compared against an
  embedded constant. Secret-bearing exported signatures should not rely on a
  host-supplied one-slot consent value as their only policy entry guard.
- Evidence path:
  - `docs/spec/runtime/effects_capabilities_privacy_v1.md` says `core.consent_token()`
    lowers to an opaque runtime sentinel and consent clauses validate exact
    sentinel equality; it also states the surface is static auditing and
    call-shape/lowering-shape enforcement, not cryptographic isolation.
  - `compiler/internal/semantics/semantics_core.go` models `consent.token` as a public
    one-slot `TypeCap`, while `secret.i32` is a public one-slot struct.
  - `compiler/internal/semantics/semantics_checker.go` requires `privacy` and
    `consent(<token>)` for secret-bearing signatures, but `@export` validation
    only checks reserved/duplicate names and does not reject `consent.token`
    parameters at the native boundary.
  - `compiler/internal/lower/lower_core.go` emits `IRLoadLocal`, sentinel
    `IRConstI32`, `IRCmpEqI32`, and `IRJmpIfZero` for consent clauses; the
    sentinel is the compile-time constant `-0x43544f4b`.
  - `compiler/internal/format/tobj/object.go` stores symbol signatures as
    `ParamSlots` / `ReturnSlots` only, so host callers see no capability
    provenance metadata.
- Why it matters: privacy-gated service functions can be exported as native
  symbols. Even though secret-taint checks prevent many direct exfiltration
  paths, the consent gate itself is reduced to a forgeable ABI integer slot at
  the external boundary, undermining the source-level guarantee that consent
  tokens are obtained through `core.consent_token()`.

### BUG-060 - `@export` on generic functions passes but emits no export symbol

- Area: `@export` validation, generic monomorphization, TOBJ symbol emission,
  and native ABI contracts.
- Severity: medium-high; a user can write `@export("ffi_generic_id")` on a
  generic function and both `check` and `build -emit object` succeed, but the
  requested `ffi_generic_id` symbol is not emitted. If the generic is unused,
  only `main` appears in the object. If the generic is used, the monomorphized
  specialization such as `id__T_i32` appears, but `cloneGenericFunc()` strips
  `ExportName`, so the requested export alias still disappears. This silently
  ships an object that does not satisfy its source-level export declaration.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-077/bughunt/export_generic_unused_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-077/bughunt/export_generic_used_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-077/bughunt/export_plain_control.tetra`
  - `/tmp/tetra-bug-hunt/session-077/bughunt/generic_plain_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_unused_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_unused_repro.tobj /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_unused_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_used_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_used_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_used_repro.tobj /tmp/tetra-bug-hunt/session-077/bughunt/export_generic_used_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-077/bughunt/export_plain_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-077/bughunt/export_plain_control.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-077/bughunt/export_plain_control.tobj /tmp/tetra-bug-hunt/session-077/bughunt/export_plain_control.tetra
```

- Observed:
  - `@export("ffi_generic_id") func id<T>(x: T) -> T` passes `check` when
    unused, and object build succeeds, but the TOBJ symbol table contains only
    `main`; there is no `ffi_generic_id` or concrete `id` symbol.
  - The same exported generic passes `check` when `main` calls `id(42)`, runs
    and prints `exit status 42`, and object build succeeds, but the TOBJ symbol
    table contains `id__T_i32` and `main`; there is still no `ffi_generic_id`
    export alias.
  - The non-generic control `@export("ffi_plain_id") func plain_id(x: Int) ->
    Int` passes `check`, runs with `exit status 42`, builds an object, and the
    TOBJ symbol table contains both `ffi_plain_id` and `plain_id` as
    `params=1 returns=1`.
  - The plain generic control without `@export` behaves as expected: a used
    generic emits only the specialization `id__T_i32`.
- Expected: generic functions should either be rejected when annotated with
  `@export`, or the language should require an explicit concrete exported
  wrapper/specialization so each native symbol has a stable, concrete ABI.
  Accepting an export declaration and silently omitting the requested symbol is
  not a safe build artifact.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` validates `@export` names before
    the generic-function branch records a `Generic` signature with zero slots,
    but the later body-checking/lowering path skips functions whose
    `TypeParams` are still present.
  - `compiler/internal/semantics/semantics_expressions.go` `cloneGenericFunc()` sets
    `out.ExportName = ""`, so monomorphized specializations are never emitted
    under the requested export alias.
  - `compiler/internal/backend/x64obj/builder.go` emits export aliases from
    lowered `IRFunc.ExportName`; since the generic declaration is skipped and
    clones have no export name, no `ffi_generic_id` symbol reaches the TOBJ.
- Why it matters: `@export` is a native/service contract. A successful build
  that drops the declared symbol will fail only at link/load/integration time,
  and the missing symbol is especially easy to miss when `main` uses a
  monomorphized specialization successfully.

### BUG-061 - Exported throwing functions erase typed-error metadata into raw return slots

- Area: `@export` ABI boundary, typed errors, `throws E`, TOBJ symbol metadata,
  and native binding contracts.
- Severity: medium-high; ordinary Tetra source rejects bare calls to throwing
  functions and forces `try`/`catch`, preserving the typed-error control-flow
  contract. `@export` accepts throwing functions, but object metadata records
  only `ParamSlots` and `ReturnSlots`; it does not record that the function
  throws, the error type, the status slot, or which slots are success vs error
  payloads. Compact `Int throws OneSlotError` exports as `returns=2`, the same
  slot count as an ordinary two-field struct return, while a payload-bearing
  error enum exports as raw success/error/status slots.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_compact_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_payload_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-078/bughunt/throwing_bare_call_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-078/bughunt/throwing_catch_control.tetra`
  - `/tmp/tetra-bug-hunt/session-078/bughunt/export_struct_two_slot_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-078/bughunt/throwing_bare_call_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-078/bughunt/throwing_catch_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-078/bughunt/throwing_catch_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_compact_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_compact_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_compact_repro.tobj /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_compact_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_payload_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_payload_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_payload_repro.tobj /tmp/tetra-bug-hunt/session-078/bughunt/export_throwing_payload_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-078/bughunt/export_struct_two_slot_control.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-078/bughunt/export_struct_two_slot_control.tobj /tmp/tetra-bug-hunt/session-078/bughunt/export_struct_two_slot_control.tetra
```

- Observed:
  - A bare source call `return read(true)` where `read` is `-> Int throws
    ReadError` is rejected with `call to throwing function 'read' requires
    try`, confirming ordinary Tetra source preserves the typed-error boundary.
  - The `catch read(...)` control passes `check` and prints `exit status 42`,
    confirming the normal local recovery path.
  - `@export("ffi_read_compact") func read_compact(flag: Bool) -> Int throws
    ReadError` passes `check`, prints `exit status 42` through internal
    `catch` calls, builds an object, and emits both `ffi_read_compact` and
    `read_compact` as `params=1 returns=2`.
  - `@export("ffi_read_payload") func read_payload(flag: Bool) -> Int throws
    ServiceError` where `ServiceError.denied(Int)` has an enum payload passes
    `check`, prints `exit status 42`, builds an object, and emits both
    `ffi_read_payload` and `read_payload` as `params=1 returns=4`.
  - The non-throwing control `@export("ffi_pair") func pair() -> Pair` also
    emits `returns=2`, showing the compact throwing ABI is indistinguishable
    from an ordinary two-slot return in TOBJ metadata.
- Expected: exported throwing functions should either be rejected, require an
  explicit non-throwing result wrapper type, or TOBJ/interface metadata should
  record `throws`, the error type, status slot semantics, and success/error
  payload layout so native bindings cannot silently treat typed-error control
  flow as an untyped tuple.
- Evidence path:
  - `docs/spec/flow/flow_syntax_v1.md` says typed errors return success tag plus
    success/error payload slots and source bare calls to throwing functions are
    rejected in favor of `try`/`catch`.
  - `compiler/internal/semantics/semantics_checker.go` computes throwing return slot
    counts but `@export` validation only checks names/duplicates, not whether
    a function has `HasThrows`.
  - `compiler/internal/lower/lower_core.go` lowers throwing returns into compact
    two-slot or expanded success/error/status layouts and then copies only
    `ReturnSlots` into `IRFunc`.
  - `compiler/internal/format/tobj/object.go` writes `HasSignature`,
    `ParamSlots`, and `ReturnSlots` for each symbol; there is no field for
    `ThrowsType` or typed-error layout.
- Why it matters: exported functions are service/FFI contracts. A generated or
  handwritten native caller cannot discover from the object whether `returns=2`
  is a `Pair`, an optional-like result, or a compact typed-error ABI, and it
  cannot recover the error enum shape for larger throwing returns. The language
  checker protects Tetra callers, but the exported artifact drops that contract.

### BUG-062 - Exported ownership-marked signatures erase borrow/consume/inout contracts

- Area: `@export` ABI boundary, ownership markers, `borrow`, `consume`,
  `inout`, TOBJ symbol metadata, and native binding contracts.
- Severity: medium-high; ordinary Tetra source treats ownership markers as
  call-site contracts and rejects borrowed values passed to owned or `inout`
  parameters. `@export` accepts ownership-marked signatures, but object metadata
  records only `ParamSlots` and `ReturnSlots`. As a result, `borrow []UInt8`
  and owned `[]UInt8` exports are both `params=2 returns=1`, `consume Int` and
  owned `Int` exports are both `params=1 returns=1`, and `inout []UInt8`
  exports carry no mutability/borrowing marker beyond the same raw slice slots.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-079/bughunt/export_borrow_slice_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-079/bughunt/export_consume_int_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-079/bughunt/export_inout_slice_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-079/bughunt/export_owned_slice_control.tetra`
  - `/tmp/tetra-bug-hunt/session-079/bughunt/export_owned_int_control.tetra`
  - `/tmp/tetra-bug-hunt/session-079/bughunt/borrow_to_owned_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-079/bughunt/inout_from_borrow_rejected_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/borrow_to_owned_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/inout_from_borrow_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/export_borrow_slice_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-079/bughunt/export_borrow_slice_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-079/bughunt/export_borrow_slice_repro.tobj /tmp/tetra-bug-hunt/session-079/bughunt/export_borrow_slice_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_slice_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_slice_control.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_slice_control.tobj /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_slice_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/export_consume_int_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-079/bughunt/export_consume_int_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-079/bughunt/export_consume_int_repro.tobj /tmp/tetra-bug-hunt/session-079/bughunt/export_consume_int_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_int_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_int_control.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_int_control.tobj /tmp/tetra-bug-hunt/session-079/bughunt/export_owned_int_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-079/bughunt/export_inout_slice_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-079/bughunt/export_inout_slice_repro.tetra
go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-079/bughunt/export_inout_slice_repro.tobj /tmp/tetra-bug-hunt/session-079/bughunt/export_inout_slice_repro.tetra
```

- Observed:
  - `borrow_to_owned_rejected_control.tetra` is rejected with `borrowed value
    derived from 'buf' cannot be passed to non-borrow parameter 1 of
    'owned_first'`.
  - `inout_from_borrow_rejected_control.tetra` is rejected with `borrowed value
    derived from 'buf' cannot be passed as inout to 'fill_first'`.
  - `@export("ffi_borrow_first") func borrow_first(buf: borrow []UInt8) -> Int`
    passes `check`, prints `exit status 42`, builds an object, and emits both
    `ffi_borrow_first` and `borrow_first` as `params=2 returns=1`.
  - The owned slice control `@export("ffi_owned_first") func owned_first(buf:
    []UInt8) -> Int` also emits `params=2 returns=1`.
  - `@export("ffi_take_int") func take_int(value: consume Int) -> Int` and the
    owned `Int` control both emit `params=1 returns=1`.
  - `@export("ffi_fill_first") func fill_first(buf: inout []UInt8) -> Int`
    passes `check`, prints `exit status 42`, builds an object, and emits both
    `ffi_fill_first` and `fill_first` as `params=2 returns=1`.
- Expected: exported functions should either reject ownership-marked parameters
  at the native boundary, require explicit FFI-safe wrapper functions, or record
  per-parameter ownership/mutability metadata so generated bindings and native
  callers cannot silently treat `borrow`, `consume`, and `inout` as ordinary
  owned slots.
- Evidence path:
  - `docs/spec/core/current_supported_surface.md` defines `borrow`, `inout`, and
    `consume` as call-site contracts.
  - `compiler/internal/semantics/semantics_core.go`,
    `compiler/internal/semantics/semantics_expressions.go`, and
    `compiler/internal/semantics/semantics_checker.go` use `ParamOwnership` to reject
    unsafe local calls and actor/task transfers.
  - `compiler/internal/backend/x64obj/builder.go` copies only lowered slot
    counts into `tobj.Symbol`.
  - `compiler/internal/format/tobj/object.go` stores `HasSignature`,
    `ParamSlots`, and `ReturnSlots`; there is no per-parameter ownership field.
- Why it matters: ownership markers are part of the semantic contract Tetra
  uses to distinguish borrowed, consumed, and mutable parameters. Native callers
  and generated bindings see only raw slot counts, so the artifact cannot
  communicate when a parameter must not be retained, must be treated as moved,
  or is intended to be mutated through an `inout` boundary.

### BUG-063 - Deferred cleanup captures miss consumed descendant ownership paths

- Area: `defer`, ownership-path tracking, partial struct-field, enum-payload,
  optional-payload `consume`, cleanup capture validation, and service/resource
  cleanup safety.
- Severity: medium-high; ordinary source rejects whole-value use after a child
  ownership path is consumed, and deferred cleanup correctly rejects a captured
  whole local when that same local is consumed before cleanup runs. However, a
  `defer` body that captures `pair.left`, whole `pair`, whole `msg`, or whole
  `maybe` is accepted when only a descendant path such as `pair.left`,
  `msg.$case0.payload0`, or `maybe.$elem` is consumed after the `defer` is
  registered. The cleanup then runs and reads the consumed descendant path.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_field_consumed_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_whole_then_payload_consumed_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_optional_whole_then_payload_consumed_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-080/bughunt/immediate_whole_after_field_consume_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_whole_consumed_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_sibling_after_field_consume_control.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/immediate_enum_whole_after_payload_consume_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/immediate_optional_whole_after_payload_consume_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_payload_alias_then_payload_consumed_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_sibling_payload_control.tetra`
- Commands:

```sh
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-080/bughunt/immediate_whole_after_field_consume_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_whole_consumed_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.bin /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_field_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_field_consumed_repro.bin /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_whole_then_field_consumed_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_sibling_after_field_consume_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-080/bughunt/defer_captures_sibling_after_field_consume_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/immediate_enum_whole_after_payload_consume_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/immediate_optional_whole_after_payload_consume_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_payload_alias_then_payload_consumed_rejected_control.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_whole_then_payload_consumed_repro.bin /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_optional_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_optional_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_optional_whole_then_payload_consumed_repro.bin /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_optional_whole_then_payload_consumed_repro.tetra
go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_sibling_payload_control.tetra
go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-081/bughunt/defer_captures_enum_sibling_payload_control.tetra
```

- Observed:
  - The immediate control `let moved: Int = take(pair.left); return use(pair)`
    is rejected with `cannot use consumed value 'pair.left'`, confirming
    normal whole-struct use after partial field consume is protected.
  - The whole-consume control after deferred capture is rejected with `defer
    cleanup captures value 'pair' ... but it was consumed ... before cleanup
    ran`, confirming direct whole-local capture/consume tracking works.
  - `defer_captures_field_then_field_consumed_repro.tetra` passes `check`,
    builds, and `run` prints `fieldexit status 42`, proving cleanup read
    `pair.left` after `take(pair.left)` consumed that ownership path.
  - `defer_captures_whole_then_field_consumed_repro.tetra` passes `check`,
    builds, and `run` prints `wholeexit status 42`, proving cleanup can use the
    whole `pair` after one of its fields has been consumed.
  - `defer_captures_sibling_after_field_consume_control.tetra` passes and
    prints `siblingexit status 42`; sibling-field cleanup after consuming
    `pair.left` is the intended allowed case and shows the desired fix needs
    path-aware captures rather than simply rejecting every base-local capture.
  - The immediate enum and optional controls are rejected with `cannot use
    consumed value 'msg.$case0.payload0'` and `cannot use consumed value
    'maybe.$elem'`, confirming ordinary whole-value use after payload consume is
    protected.
  - A deferred cleanup that captures the direct enum payload alias `left` is
    rejected with `defer cleanup captures value 'left' ... but it was consumed
    ... before cleanup ran`, confirming direct alias tracking works.
  - `defer_captures_enum_whole_then_payload_consumed_repro.tetra` passes
    `check`, builds, and `run` prints `enumexit status 42`, proving cleanup can
    use whole `msg` after one payload binding has been consumed.
  - `defer_captures_optional_whole_then_payload_consumed_repro.tetra` passes
    `check`, builds, and `run` prints `optionalexit status 42`, proving cleanup
    can use whole `maybe` after its `some` payload has been consumed.
  - `defer_captures_enum_sibling_payload_control.tetra` passes and prints
    `siblingexit status 42`, confirming sibling-payload cleanup remains valid.
- Expected: pending deferred-cleanup validation should reject a cleanup that
  captures a consumed descendant or its enclosing whole value if the descendant
  is consumed before cleanup runs, while still allowing cleanup that captures
  only unaffected sibling paths such as `pair.right` or the second enum payload.
- Evidence path:
  - `docs/spec/core/current_supported_surface.md` and `docs/spec/runtime/ownership_v1.md`
    state that partial struct-field, enum-payload, and optional-payload consumes
    reject whole-value use while allowing sibling-path reuse.
  - `compiler/internal/semantics/semantics_memory_resources.go` records a field access by
    collecting only the base identifier, so `pair.left` becomes capture `pair`.
  - `compiler/internal/semantics/semantics_memory_resources.go` `checkPendingDeferCaptures()`
    checks only `consumedAt(name)` for each captured base; it does not call the
    descendant-aware `checkNoConsumedDescendants()` path used by ordinary
    whole-value checks.
- Why it matters: service handlers commonly use `defer` for cleanup after
  staged request parsing, ownership transfer, or resource handoff. This gap lets
  cleanup code observe or act on a moved field path even though the same
  ownership rule rejects equivalent immediate code.

### BUG-064 - Deferred cleanup reuses stale privacy taint after captured locals become secret

- Area: `defer`, privacy/secret-taint analysis, global-store privacy sinks,
  `secret.i32`, and cleanup validation.
- Severity: high; direct secret-tainted stores into globals are rejected, and a
  deferred cleanup that captures an already secret-tainted local is rejected.
  However, if the cleanup is registered while a captured local is still public,
  and the local is assigned a secret-unsealed value later, the checker accepts
  the function. Runtime cleanup then writes the secret-derived value into a
  global.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-082/bughunt/defer_late_secret_global_store_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-082/bughunt/direct_secret_global_store_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-082/bughunt/defer_already_secret_global_store_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-082/bughunt/defer_public_global_store_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-082/bughunt/direct_secret_global_store_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-082/bughunt/defer_already_secret_global_store_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-082/bughunt/defer_late_secret_global_store_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-082/bughunt/defer_late_secret_global_store_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-082/defer_late_secret_global_store_repro /tmp/tetra-bug-hunt/session-082/bughunt/defer_late_secret_global_store_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-082/bughunt/defer_public_global_store_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-082/bughunt/defer_public_global_store_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-082/defer_public_global_store_control /tmp/tetra-bug-hunt/session-082/bughunt/defer_public_global_store_control.tetra
```

- Observed:
  - The direct control is rejected with `secret-tainted value cannot be stored
    in global 'leaked'`.
  - The already-tainted deferred control is rejected with `secret-tainted value
    cannot be stored in global 'leaked'`, proving the defer-body checker can see
    taint when it is present at registration time.
  - `defer_late_secret_global_store_repro.tetra` passes `check`, builds, and
    `run` prints `exit status 42`. The sentinel returns `42` only after
    deferred cleanup stores the later `core.secret_unseal_i32(...)` result into
    global `leaked`.
  - `defer_public_global_store_control.tetra` passes `check`, builds, and
    `run` prints `exit status 42`, confirming ordinary public deferred global
    stores still work.
- Expected: deferred cleanup should be revalidated against the taint state that
  can hold when cleanup runs, or registering a `defer` should track captured
  locals as taint dependencies and reject later secret-tainting assignments that
  would make the cleanup body write secret-derived data to public/global sinks.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `checkDeferBody()` snapshots
    `analysis.secretTaint`, checks the cleanup body immediately, then restores
    the saved taint map.
  - `LetStmt` and `AssignStmt` update local secret taint after
    `core.secret_unseal_i32(...)`.
  - Global assignment rejects `secretTainted` only when the checker sees the
    tainted expression during statement checking.
  - `compiler/internal/lower/lower_core.go` stores deferred bodies and lowers them at
    scope exit, so the cleanup reads the later local value rather than the value
    from registration time.
- Why it matters: cleanup code in request handlers and microservices often
  writes metrics, audit state, cache entries, or status globals after the main
  body mutates locals. This gap lets a value become secret-tainted after cleanup
  registration and then escape through a sink that equivalent direct code
  rejects.

### BUG-065 - Actor/task boundary misses constant writes to mutable globals

- Area: actor/task worker boundary checks, mutable global state analysis,
  `core.task_spawn_i32`, `core.spawn`, and race/isolation safety.
- Severity: high; workers that read or read-modify-write a mutable global are
  rejected before crossing task/actor boundaries, but workers that only assign a
  non-global value such as `leaked = 42` pass `check`, build, and mutate shared
  global state at runtime.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-083/bughunt/task_worker_constant_global_write_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_constant_global_write_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-083/bughunt/task_worker_read_write_global_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_read_write_global_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-083/bughunt/task_worker_public_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-083/task_worker_constant_global_write_repro /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_read_write_global_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o /tmp/tetra-bug-hunt/session-083/actor_worker_constant_global_write_repro /tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_constant_global_write_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-083/bughunt/actor_worker_read_write_global_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_public_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-083/bughunt/task_worker_public_control.tetra
```

- Observed:
  - `task_worker_constant_global_write_repro.tetra` passes `check`, builds, and
    `run` prints `exit status 42`. The sentinel returns `42` only after the task
    worker writes `leaked = 42` and `main` observes the changed global after
    `core.task_join_i32`.
  - `actor_worker_constant_global_write_repro.tetra` passes `check`, builds,
    and `run` prints `exit status 42` after the spawned actor receives a
    message, writes `leaked = 42`, replies, and `main` observes the global.
  - The read-modify-write controls are rejected with
    `task_spawn_i32 target 'worker' touches mutable global state and cannot
    cross task boundary` and `spawn target 'worker' touches mutable global state
    and cannot cross actor boundary`.
  - The public task control passes `check` and `run` prints `exit status 42`,
    confirming the task harness itself is valid.
- Expected: any assignment to a mutable global should mark the function as
  touching mutable global state, independent of whether the RHS reads a global.
  Task and actor spawn checks should reject such workers before they cross the
  concurrency boundary.
- Evidence path:
  - `compiler/internal/semantics/semantics_expressions.go` rejects task and actor spawn targets
    when `targetSig.TouchesMutableGlobals` is true.
  - `compiler/internal/semantics/semantics_checker.go` copies
    `analysis.touchesMutableGlobals` into each function signature.
  - `checkExprWithEffects()` marks mutable global reads by setting
    `analysis.touchesMutableGlobals = true`.
  - The ordinary global-assignment path in `checkStmts()` validates mutability,
    type, ownership, and secret taint, but only sets `touchesMutableGlobals` for
    function-typed globals. A plain assignment such as `leaked = 42` therefore
    leaves the worker signature boundary-safe.
- Why it matters: task and actor boundaries are intended to prevent shared
  mutable global state from crossing concurrency isolation. A service worker can
  mutate global routing tables, counters, cache state, or security flags from a
  spawned task/actor as long as it writes a constant, local, parameter, or other
  non-global RHS.

### BUG-066 - Exported async functions are emitted as synchronous native symbols

- Area: `@export` ABI validation, async function signatures, TOBJ symbol
  metadata, and native/service boundary contracts.
- Severity: medium-high; ordinary Tetra source rejects bare calls to async
  functions and rejects async task targets, but `@export` accepts an async
  function and emits a native object symbol with only ordinary parameter/return
  slot counts. The object has no async marker, so host bindings see a synchronous
  `params=0 returns=1` function-shaped ABI.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-084/bughunt/export_async_function_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-084/bughunt/async_bare_call_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-084/bughunt/task_spawn_async_target_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-084/bughunt/export_sync_function_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-084/bughunt/export_async_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-084/bughunt/export_async_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-084/export_async_function_repro.tobj /tmp/tetra-bug-hunt/session-084/bughunt/export_async_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-084/bughunt/async_bare_call_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-084/bughunt/task_spawn_async_target_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-084/bughunt/export_sync_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-084/bughunt/export_sync_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-084/export_sync_function_control.tobj /tmp/tetra-bug-hunt/session-084/bughunt/export_sync_function_control.tetra
rg -a -n "ffi_async_answer|async_answer|ffi_sync_answer|sync_answer" /tmp/tetra-bug-hunt/session-084/export_async_function_repro.tobj /tmp/tetra-bug-hunt/session-084/export_sync_function_control.tobj
```

- Observed:
  - `export_async_function_repro.tetra` passes `check`, runs with
    `exit status 42`, and builds a TOBJ object.
  - The TOBJ object contains both `async_answer` and exported alias
    `ffi_async_answer`. The raw symbol record shows the same `params=0,
    returns=1` slot shape as the synchronous export control.
  - `async_bare_call_rejected_control.tetra` is rejected with `call to async
    function 'async_answer' requires await`.
  - `task_spawn_async_target_rejected_control.tetra` is rejected with
    `task_spawn_i32 target must be synchronous`.
  - The synchronous export control passes, runs with `exit status 42`, builds a
    TOBJ object, and emits `ffi_sync_answer` / `sync_answer` symbols.
- Expected: `@export` should reject async functions unless the native object
  format and generated bindings carry an explicit async ABI contract. A
  successful exported object should not erase the source-level async call
  requirement into a plain synchronous slot signature.
- Evidence path:
  - `compiler/internal/frontend/frontend_core.go` permits `@export` attributes before
    `async func`.
  - `compiler/internal/semantics/semantics_checker.go` stores `FuncSig.Async` and uses it
    to reject bare async calls, async `main`, and async task/actor targets, but
    `validateExportedOpaqueABISignature()` does not reject `fn.Async`.
  - `compiler/internal/lower/lower_core.go` copies `fn.Decl.ExportName` into
    `ir.IRFunc` while not carrying async metadata into the exported symbol.
  - `compiler/internal/format/tobj/object.go` `Symbol` stores only `Name`,
    `Offset`, `HasSignature`, `ParamSlots`, and `ReturnSlots`; there is no async
    field.
- Why it matters: `@export` is the service/native ABI boundary. Host code can
  discover and call `ffi_async_answer` as if it were an ordinary synchronous
  function, despite Tetra source requiring `await` for the same callee and
  rejecting it from synchronous task/actor spawn paths.

### BUG-067 - Exported budgeted functions erase caller-budget context into plain native symbols

- Area: `@export` ABI validation, budget semantic clauses, caller-context
  checks, TOBJ symbol metadata, and native/service boundary contracts.
- Severity: medium-high; ordinary Tetra source rejects calls into
  `budget(N)` functions unless the caller declares a sufficient budget, but
  `@export` accepts the same budgeted function and emits an ordinary native
  symbol. The object symbol has only parameter/return slot counts, so host
  bindings cannot discover or enforce the caller-budget requirement.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-085/bughunt/export_budgeted_function_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-085/bughunt/direct_missing_budget_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-085/bughunt/direct_underbudget_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-085/bughunt/direct_budgeted_call_control.tetra`
  - `/tmp/tetra-bug-hunt/session-085/bughunt/export_plain_function_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-085/bughunt/export_budgeted_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-085/bughunt/export_budgeted_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-085/export_budgeted_function_repro.tobj /tmp/tetra-bug-hunt/session-085/bughunt/export_budgeted_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-085/bughunt/direct_missing_budget_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-085/bughunt/direct_underbudget_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-085/bughunt/direct_budgeted_call_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-085/bughunt/direct_budgeted_call_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-085/bughunt/export_plain_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-085/bughunt/export_plain_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-085/export_plain_function_control.tobj /tmp/tetra-bug-hunt/session-085/bughunt/export_plain_function_control.tetra
rg -a -n "ffi_budgeted_answer|budgeted_answer|ffi_plain_answer|plain_answer" /tmp/tetra-bug-hunt/session-085/export_budgeted_function_repro.tobj /tmp/tetra-bug-hunt/session-085/export_plain_function_control.tobj
```

- Observed:
  - `export_budgeted_function_repro.tetra` passes `check`, runs with
    `exit status 42`, and builds a TOBJ object containing both
    `ffi_budgeted_answer` and `budgeted_answer`.
  - `direct_missing_budget_rejected_control.tetra` is rejected with
    `budget context for call to 'budgeted_answer' requires caller budget at
    least 5`.
  - `direct_underbudget_rejected_control.tetra` is rejected with
    `budget context for call to 'budgeted_answer' requires caller budget at
    least 6, got 5`.
  - The direct covered call control passes and runs with `exit status 42`.
  - The plain export control passes, runs with `exit status 42`, builds a TOBJ
    object, and emits `ffi_plain_answer` / `plain_answer` symbols.
- Expected: `@export` should reject functions whose ABI depends on
  caller-budget context unless the native object format and generated bindings
  carry explicit budget/effect metadata. A successful exported object should not
  erase a source-level `budget(N)` call precondition into a plain slot
  signature.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` `validateBudgetContextEdge()`
    enforces caller budget for direct calls into budgeted functions.
  - `compiler/internal/ir/ir.go` `IRPolicy` carries `HasBudget` and `Budget`
    internally for lowered function policy.
  - `compiler/internal/format/tobj/object.go` `Symbol` stores only `Name`,
    `Offset`, `HasSignature`, `ParamSlots`, and `ReturnSlots`; there is no
    budget or effect field.
  - The exported budgeted repro is accepted by the checker and object builder
    despite that missing exported metadata.
- Why it matters: `@export` is the service/native ABI boundary. A host can call
  `ffi_budgeted_answer` through the raw symbol ABI without declaring any budget,
  while equivalent Tetra source calls are statically rejected unless their caller
  budget is present and sufficient.

### BUG-068 - Exported effectful functions erase `uses` effects into plain native symbols

- Area: `@export` ABI validation, `uses` effect propagation, TOBJ symbol
  metadata, and native/service boundary contracts.
- Severity: medium-high; ordinary Tetra calls propagate callee effects and
  reject callers that omit required `uses` declarations, but `@export` accepts
  an effectful `uses io` function and emits a native object symbol whose
  metadata contains only parameter/return slot counts. Host bindings therefore
  cannot discover or audit that calling the exported symbol performs IO.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-086/bughunt/export_effectful_io_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-086/bughunt/direct_missing_uses_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-086/bughunt/direct_with_uses_control.tetra`
  - `/tmp/tetra-bug-hunt/session-086/bughunt/export_pure_function_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-086/bughunt/export_effectful_io_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-086/bughunt/export_effectful_io_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-086/export_effectful_io_repro.tobj /tmp/tetra-bug-hunt/session-086/bughunt/export_effectful_io_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-086/bughunt/direct_missing_uses_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-086/bughunt/direct_with_uses_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-086/bughunt/direct_with_uses_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-086/bughunt/export_pure_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-086/bughunt/export_pure_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-086/export_pure_function_control.tobj /tmp/tetra-bug-hunt/session-086/bughunt/export_pure_function_control.tetra
rg -a -n "ffi_log_answer|log_answer|ffi_plain_answer|plain_answer" /tmp/tetra-bug-hunt/session-086/export_effectful_io_repro.tobj /tmp/tetra-bug-hunt/session-086/export_pure_function_control.tobj
```

- Observed:
  - `export_effectful_io_repro.tetra` defines
    `@export("ffi_log_answer") func log_answer() -> Int uses io` and calls
    `print`. It passes `check`, runs with `exit status 42`, and builds a TOBJ
    object containing both `ffi_log_answer` and `log_answer`.
  - `direct_missing_uses_rejected_control.tetra` calls the same effectful
    function from a caller without `uses io` and is rejected with
    `function 'main' uses effect 'io' but does not declare it`.
  - `direct_with_uses_control.tetra` declares `uses io`, passes `check`, prints
    `direct`, and exits with sentinel `exit status 42`.
  - The pure export control passes, runs with `exit status 42`, builds a TOBJ
    object, and emits `ffi_plain_answer` / `plain_answer`.
- Expected: `@export` should reject effectful functions unless the native object
  format and generated bindings carry explicit effect metadata. A successful
  exported object should not erase a source-level `uses io` call requirement
  into a plain slot signature.
- Evidence path:
  - `docs/spec/runtime/effects_capabilities_privacy_v1.md` states that function calls
    propagate callee effects transitively and missing `uses` declarations are
    diagnostics.
  - `compiler/internal/semantics/semantics_memory_resources.go` `effectContext.require()` emits the
    missing-effect diagnostic for ordinary source callers.
  - `compiler/internal/semantics/semantics_checker.go` stores normalized effects in
    `FuncSig.Effects`, but `validateExportedOpaqueABISignature()` only rejects
    selected opaque ABI value shapes and does not reject or preserve
    `fn.Uses`.
  - `compiler/internal/backend/x64obj/builder.go` emits exported aliases into
    the TOBJ symbol table using only `ParamSlots` and `ReturnSlots`.
  - `compiler/internal/format/tobj/object.go` `Symbol` has no effect metadata
    field.
- Why it matters: `uses` declarations are the source-language effect audit
  boundary. Exported service/native callers can invoke `ffi_log_answer` and
  trigger IO through the raw symbol ABI without any equivalent static
  declaration, reviewable metadata, or generated binding contract.

### BUG-069 - Export aliases that collide with later function names are silently rebound

- Area: `@export` name validation, object symbol table construction, TOBJ
  symbol offsets, and native/service ABI contracts.
- Severity: high; a source file can declare `@export("target")` on
  `exported_answer`, then later define an ordinary function named `target`.
  `check`, `run`, and object build all succeed, but the emitted TOBJ symbol
  named `target` points at the later ordinary function instead of the exported
  function. Reversing declaration order fails at object build with
  `duplicate exported symbol 'target'`, so the behavior is order-dependent and
  not caught at the semantic boundary.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_later_function_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_prior_function_control.tetra`
  - `/tmp/tetra-bug-hunt/session-087/bughunt/non_colliding_export_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_later_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_later_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-087/export_name_collides_later_function_repro.tobj /tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_later_function_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_prior_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-087/export_name_collides_prior_function_control.tobj /tmp/tetra-bug-hunt/session-087/bughunt/export_name_collides_prior_function_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-087/bughunt/non_colliding_export_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run /tmp/tetra-bug-hunt/session-087/bughunt/non_colliding_export_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-087/non_colliding_export_control.tobj /tmp/tetra-bug-hunt/session-087/bughunt/non_colliding_export_control.tetra
python -c '<TOBJ symbol parser>' /tmp/tetra-bug-hunt/session-087/export_name_collides_later_function_repro.tobj /tmp/tetra-bug-hunt/session-087/non_colliding_export_control.tobj
```

- Observed:
  - The later-collision repro passes `check`, runs with `exit status 42`, and
    builds a TOBJ object.
  - Parsed TOBJ symbols for the repro:
    - `exported_answer offset=0 params=0 returns=1`
    - `target offset=13 params=0 returns=1`
    - `main offset=26 params=0 returns=1`
  - Since `@export("target")` was attached to `exported_answer`, the exported
    alias should point at offset `0`; instead the only `target` symbol points
    at the later ordinary `target` function's offset `13`.
  - The non-colliding control emits `ffi_exported_answer offset=0` alongside
    `exported_answer offset=0`, proving alias offsets normally match the
    exported function.
  - The prior-collision control passes `check`, but object build fails with
    `duplicate exported symbol 'target'`.
- Expected: semantic export validation should reject any `@export` name that
  collides with any emitted internal function symbol, independent of declaration
  order. The object builder should also treat function-name symbol collisions as
  hard errors instead of overwriting existing symbol table entries.
- Evidence path:
  - `compiler/internal/semantics/semantics_checker.go` tracks duplicate `@export` names
    only in `exportedSymbols`; it does not compare export aliases against all
    function names in the module/world.
  - `compiler/internal/backend/x64obj/builder.go` writes
    `symbolOffsets[fn.Name] = len(e.Buf)` without first checking whether that
    symbol name already exists as an export alias.
  - The same builder checks `fn.ExportName` only against symbols already seen,
    making collisions with earlier internal names fail but collisions with later
    internal names silently rebind the symbol.
- Why it matters: `@export` names are native/service ABI promises. A published
  object can claim to export one function while the symbol table actually binds
  the promised name to a different later function, producing silent host-side
  misrouting rather than a deterministic compiler diagnostic.

### BUG-070 - Export names accept whitespace and control characters as native symbols

- Area: `@export` name validation, native symbol grammar, TOBJ symbol table
  encoding, generated bindings, and service ABI hygiene.
- Severity: medium-high; the parser rejects `@export("")`, but accepts
  whitespace/control-character names such as `@export("ffi log")`,
  `@export("ffi\nlog")`, and `@export("ffi\tlog")`. These names pass semantic
  checking and object emission, and the TOBJ symbol table preserves the
  malformed bytes as exported symbols.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-088/bughunt/export_space_name_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-088/bughunt/export_newline_name_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-088/bughunt/export_tab_name_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-088/bughunt/export_empty_name_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-088/bughunt/export_identifier_name_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-088/bughunt/export_space_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-088/export_space_name_repro.tobj /tmp/tetra-bug-hunt/session-088/bughunt/export_space_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-088/bughunt/export_newline_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-088/export_newline_name_repro.tobj /tmp/tetra-bug-hunt/session-088/bughunt/export_newline_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-088/bughunt/export_tab_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-088/export_tab_name_repro.tobj /tmp/tetra-bug-hunt/session-088/bughunt/export_tab_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-088/bughunt/export_empty_name_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-088/bughunt/export_identifier_name_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-088/export_identifier_name_control.tobj /tmp/tetra-bug-hunt/session-088/bughunt/export_identifier_name_control.tetra
python -c '<TOBJ symbol parser with repr(name)>' /tmp/tetra-bug-hunt/session-088/export_space_name_repro.tobj /tmp/tetra-bug-hunt/session-088/export_newline_name_repro.tobj /tmp/tetra-bug-hunt/session-088/export_tab_name_repro.tobj /tmp/tetra-bug-hunt/session-088/export_identifier_name_control.tobj
```

- Observed:
  - Space, newline, and tab export-name repros all pass `check` and build TOBJ
    objects.
  - Parsed TOBJ symbols include:
    - `name='ffi log' offset=0 params=0 returns=1`
    - `name='ffi\nlog' offset=0 params=0 returns=1`
    - `name='ffi\tlog' offset=0 params=0 returns=1`
  - The empty-name control is rejected at parse/check time with
    `@export name must not be empty`.
  - The identifier-name control emits `name='ffi_log' offset=0 params=0
    returns=1`.
- Expected: `@export` should enforce a stable native/service symbol grammar
  such as `[A-Za-z_][A-Za-z0-9_]*` or a documented superset, and reject
  whitespace, control characters, and other binding-hostile bytes before object
  emission.
- Evidence path:
  - `compiler/internal/frontend/frontend_core.go` rejects only an empty export string
    during attribute parsing.
  - `compiler/internal/semantics/semantics_checker.go` checks `core.*`, reserved
    `__tetra_*`, and duplicate export names, but does not validate a general
    symbol-name grammar.
  - `compiler/internal/format/tobj/object.go` `validateSymbolRecord()` rejects
    only empty symbol names and out-of-range metadata, so whitespace/control
    names are serialized into the TOBJ symbol table.
- Why it matters: exported symbols are consumed by native linkers, generated
  bindings, service manifests, and review tooling. Allowing spaces and control
  characters makes symbol lookup ambiguous, log/report output misleading, and
  downstream binding generation vulnerable to malformed identifiers.

### BUG-071 - core.sym_addr accepts malformed names into call relocations

- Area: `core.sym_addr`, unsafe/link boundary, native symbol grammar,
  `IRSymAddr`, TOBJ relocation records, and native linker hygiene.
- Severity: medium-high; `core.sym_addr("")` is rejected, but
  `core.sym_addr("ffi log")`, `core.sym_addr("ffi\nlog")`, and
  `core.sym_addr("ffi\tlog")` pass `check`, lower to `IRSymAddr`, build as
  TOBJ objects, and preserve the malformed bytes as `RelocCallRel32` names.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_space_name_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_newline_name_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_tab_name_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_empty_name_rejected_control.tetra`
  - `/tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_identifier_name_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_space_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-089/sym_addr_space_name_repro.tobj /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_space_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_newline_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-089/sym_addr_newline_name_repro.tobj /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_newline_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_tab_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-089/sym_addr_tab_name_repro.tobj /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_tab_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_empty_name_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-089/sym_addr_empty_name_rejected_control.tobj /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_empty_name_rejected_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_identifier_name_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-089/sym_addr_identifier_name_control.tobj /tmp/tetra-bug-hunt/session-089/bughunt/sym_addr_identifier_name_control.tetra
python -c '<TOBJ relocation parser with repr(name)>' /tmp/tetra-bug-hunt/session-089/sym_addr_space_name_repro.tobj /tmp/tetra-bug-hunt/session-089/sym_addr_newline_name_repro.tobj /tmp/tetra-bug-hunt/session-089/sym_addr_tab_name_repro.tobj /tmp/tetra-bug-hunt/session-089/sym_addr_identifier_name_control.tobj
```

- Observed:
  - Space, newline, and tab symbol-name repros all pass `check` and build TOBJ
    objects.
  - Parsed TOBJ relocations include:
    - `reloc kind=1 at=25 name='ffi log' addend=0`
    - `reloc kind=1 at=25 name='ffi\nlog' addend=0`
    - `reloc kind=1 at=25 name='ffi\tlog' addend=0`
  - The empty-name control is rejected with
    `sym_addr expects a non-empty symbol name`.
  - The identifier-name control emits
    `reloc kind=1 at=25 name='ffi_target' addend=0`.
- Expected: `core.sym_addr` should enforce the same documented native/service
  symbol grammar as exported symbols and reject whitespace, control characters,
  and other binding-hostile bytes before lowering or object emission.
- Evidence path:
  - `docs/spec/runtime/unsafe.md:33` classifies `core.sym_addr` as always unsafe with
    the `link` effect.
  - `compiler/internal/semantics/semantics_expressions.go:2012` to `:2022` validates only arity,
    string-literal form, and non-empty bytes.
  - `compiler/internal/lower/lower_core.go:3291` to `:3303` repeats the same
    non-empty check and emits `IRSymAddr` with the raw string.
  - `compiler/internal/backend/x64core/x64core_core.go:724` to `:729` turns `IRSymAddr`
    into a `CallPatch` name.
  - `compiler/internal/backend/x64obj/builder.go:124` to `:130` serializes
    unresolved call patches as `RelocCallRel32` without symbol grammar checks.
  - `compiler/internal/format/tobj/object.go:324` to `:331` rejects only empty
    call-relocation names and non-zero addends.
- Why it matters: `core.sym_addr` is a deliberate escape hatch into native
  linking. Letting whitespace/control-character names reach relocations can
  make linker diagnostics, generated manifests, review logs, and binding
  tooling ambiguous, while a deterministic syntax diagnostic would catch the
  issue at the Tetra boundary.

### BUG-072 - Native symbol names preserve embedded NUL bytes

- Area: string literal lexing, `@export`, `core.sym_addr`, TOBJ symbol/reloc
  serialization, native service ABI, and C-compatible host tooling.
- Severity: high; a raw `0x00` byte inside a string literal is accepted inside
  native symbol names. `@export("ffi<0x00>log")` passes `check`, builds a TOBJ
  object, and emits an exported symbol whose bytes are `66 66 69 00 6c 6f 67`.
  `core.sym_addr("ffi<0x00>log")` likewise passes `check`, builds an object,
  and emits a `RelocCallRel32` with the same embedded NUL bytes.
- Reproducers:
  - `/tmp/tetra-bug-hunt/session-090/bughunt/export_nul_name_repro.tetra`
  - `/tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_nul_name_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-090/bughunt/export_identifier_name_control.tetra`
  - `/tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_identifier_name_control.tetra`
- Commands:

```sh
python -c '<write repro files with raw 0x00 inside the two symbol string literals>'
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-090/bughunt/export_nul_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_nul_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-090/export_nul_name_repro.tobj /tmp/tetra-bug-hunt/session-090/bughunt/export_nul_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-090/sym_addr_nul_name_repro.tobj /tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_nul_name_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-090/export_identifier_name_control.tobj /tmp/tetra-bug-hunt/session-090/bughunt/export_identifier_name_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-090/sym_addr_identifier_name_control.tobj /tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_identifier_name_control.tetra
python -c '<TOBJ parser printing repr(name) and raw hex>' /tmp/tetra-bug-hunt/session-090/export_nul_name_repro.tobj /tmp/tetra-bug-hunt/session-090/sym_addr_nul_name_repro.tobj /tmp/tetra-bug-hunt/session-090/export_identifier_name_control.tobj /tmp/tetra-bug-hunt/session-090/sym_addr_identifier_name_control.tobj
```

- Observed:
  - Both raw-NUL repros pass `tetra check`.
  - Both raw-NUL repros build TOBJ objects.
  - Parsed TOBJ artifact evidence:
    - `symbol name='ffi\x00log' hex=666669006c6f67 offset=0 params=0 returns=1`
    - `reloc kind=1 at=25 name='ffi\x00log' hex=666669006c6f67 addend=0`
  - Identifier controls emit `ffi_log` and `ffi_target` without embedded NUL
    bytes.
- Expected: native symbol-name boundaries should reject embedded NUL bytes
  before parser output is stored as `ExportName` or lowered as `IRSymAddr`.
  A NUL-containing string may be a valid byte string in some languages, but it
  must not become a native ABI symbol name.
- Evidence path:
  - `compiler/internal/frontend/frontend_core.go:337` to `:343` treats a source file as
    acceptable when it is valid UTF-8; `0x00` satisfies that check.
  - `compiler/internal/frontend/frontend_core.go:355` to `:391` appends any
    non-quote/non-escape byte directly to `TokenString.str`.
  - `compiler/internal/frontend/frontend_core.go:1127` to `:1139` copies
    `TokenString.str` into `ExportName` and rejects only the empty string.
  - `compiler/internal/semantics/semantics_expressions.go:2012` to `:2022` validates
    `core.sym_addr` only as a non-empty string literal.
  - `compiler/internal/lower/lower_core.go:3291` to `:3303` emits `IRSymAddr` with
    the raw string bytes.
  - `compiler/internal/format/tobj/object.go:308` to `:331` rejects only empty
    symbol/relocation names.
  - `compiler/internal/format/tobj/object.go:469` to `:477` serializes strings
    as byte length plus raw bytes, preserving embedded NUL in the custom object
    format.
- Why it matters: native linkers, PE/ELF/Mach-O adapters, C FFI loaders,
  generated bindings, logs, and service manifests often treat symbol names as
  NUL-terminated strings. A Tetra artifact can promise `ffi\x00log` while
  downstream tooling sees or reports only `ffi`, creating truncation,
  collision, audit-log ambiguity, and host binding confusion.

### BUG-073 - WASM pure services import host I/O functions without effects

- Area: `wasm32-wasi` and `wasm32-web` backends, host import policy,
  effect-gated host access, microservice least privilege, and
  `validate-wasm-imports`.
- Severity: medium-high; a pure Tetra service with no `uses io`, no `print`,
  and body `return 42` builds successfully for both WASM targets but still
  declares host I/O imports. The WASI artifact imports
  `wasi_snapshot_preview1.fd_write` even though the code never calls it. The
  Web artifact imports `tetra_web_v1.console_log` and `tetra_web_v1.panic`
  even though the pure module has no host calls at all.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-091/bughunt/wasm_pure_service_repro.tetra`
- Control:
  - `/tmp/tetra-bug-hunt/session-091/bughunt/wasm_print_service_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-091/bughunt/wasm_pure_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.wasi.wasm /tmp/tetra-bug-hunt/session-091/bughunt/wasm_pure_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.web.wasm /tmp/tetra-bug-hunt/session-091/bughunt/wasm_pure_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o /tmp/tetra-bug-hunt/session-091/wasm_print_service_control.wasi.wasm /tmp/tetra-bug-hunt/session-091/bughunt/wasm_print_service_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o /tmp/tetra-bug-hunt/session-091/wasm_print_service_control.web.wasm /tmp/tetra-bug-hunt/session-091/bughunt/wasm_print_service_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.wasi.wasm
GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.web.wasm
python -c '<WASM import/export/call-index parser>' /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.wasi.wasm /tmp/tetra-bug-hunt/session-091/wasm_pure_service_repro.web.wasm /tmp/tetra-bug-hunt/session-091/wasm_print_service_control.wasi.wasm /tmp/tetra-bug-hunt/session-091/wasm_print_service_control.web.wasm
```

- Observed:
  - Pure service source:

```tetra
func main() -> Int:
    return 42
```

  - Pure WASI artifact imports both allowlisted functions:
    - `import wasi_snapshot_preview1.fd_write kind=0 type=0`
    - `import wasi_snapshot_preview1.proc_exit kind=0 type=1`
    - `call_indexes=2,1`, meaning it calls the internal `main` wrapper and
      `proc_exit`, but not imported `fd_write` at index `0`.
  - Pure Web artifact imports both allowlisted functions:
    - `import tetra_web_v1.console_log kind=0 type=0`
    - `import tetra_web_v1.panic kind=0 type=1`
    - `call_indexes=` is empty, so neither host import is called.
  - The print control imports the same surfaces, but its call indexes include
    imported log functions:
    - WASI control `call_indexes=0,2,1`
    - Web control `call_indexes=0`
  - `validate-wasm-imports` exits 0 for the pure WASI and pure Web artifacts,
    so the current gate does not detect the expanded import surface.
- Expected: WASM import sections should be demand-driven by lowered code and
  effects. A pure service may need `proc_exit` for WASI process completion, but
  it should not import `fd_write` unless `print`/`uses io` actually lowers to
  stdout. A pure Web service should not import `console_log` when no output
  path exists. The validator should either enforce minimal imports or the
  backend should emit only imports that are actually referenced.
- Evidence path:
  - `docs/backend/wasm_architecture.md:93` to `:108` says compiled `.wasm`
    artifacts are the production host boundary and safe code must not gain host
    access by lowering around effects/capabilities.
  - `docs/backend/wasm_architecture.md:117` to `:120` says WASI host access
    must remain effect-gated.
  - `compiler/internal/backend/wasm32_wasi/codegen.go:220` to `:231`
    unconditionally writes both `fd_write` and `proc_exit` into every WASI
    import section.
  - `compiler/internal/backend/wasm32_web/codegen.go:209` to `:220`
    unconditionally writes both `console_log` and `panic` into every Web import
    section.
  - `tools/cmd/validate-wasm-imports/main.go:124` to `:131` checks only that
    imports are functions and appear in the target allowlist.
  - `tools/cmd/validate-wasm-imports/main.go:135` to `:146` allowlists the
    full target import set, so it cannot distinguish a pure artifact from an
    effectful one.
- Why it matters: WASM import sections are the visible capability contract for
  sandbox hosts, deployment manifests, and service review tooling. Importing
  unused host I/O functions for pure services violates least privilege, makes
  generated service manifests look more privileged than the Tetra source, and
  lets release gates pass artifacts whose host boundary is broader than their
  declared effects.

### BUG-074 - WASM builds silently drop `@export` service endpoints

- Area: `@export`, `wasm32-wasi`, `wasm32-web`, WASM export sections, service
  endpoint ABI, and generated host bindings.
- Severity: high; a Tetra service function marked
  `@export("service_answer")` passes `check` and builds for both WASM targets,
  but the emitted `.wasm` artifacts export only the target entrypoint and
  memory. The promised `service_answer` endpoint is absent. Native TOBJ output
  from the same source does contain `service_answer`, so the attribute is parsed
  and semantically accepted but lost at the WASM artifact boundary.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-092/bughunt/wasm_exported_service_repro.tetra`
- Control:
  - `/tmp/tetra-bug-hunt/session-092/bughunt/wasm_entry_only_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-092/bughunt/wasm_exported_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.wasi.wasm /tmp/tetra-bug-hunt/session-092/bughunt/wasm_exported_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.web.wasm /tmp/tetra-bug-hunt/session-092/bughunt/wasm_exported_service_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o /tmp/tetra-bug-hunt/session-092/wasm_entry_only_control.wasi.wasm /tmp/tetra-bug-hunt/session-092/bughunt/wasm_entry_only_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o /tmp/tetra-bug-hunt/session-092/wasm_entry_only_control.web.wasm /tmp/tetra-bug-hunt/session-092/bughunt/wasm_entry_only_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.native.tobj /tmp/tetra-bug-hunt/session-092/bughunt/wasm_exported_service_repro.tetra
python -c '<WASM export-section parser and TOBJ symbol parser>' /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.wasi.wasm /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.web.wasm /tmp/tetra-bug-hunt/session-092/wasm_entry_only_control.wasi.wasm /tmp/tetra-bug-hunt/session-092/wasm_entry_only_control.web.wasm /tmp/tetra-bug-hunt/session-092/wasm_exported_service_repro.native.tobj
```

- Observed:
  - Repro source:

```tetra
@export("service_answer")
func answer() -> Int:
    return 7

func main() -> Int:
    return answer()
```

  - The repro passes `tetra check`.
  - `wasm32-wasi` repro exports:
    - `export memory kind=2 index=0`
    - `export _start kind=0 index=4`
  - `wasm32-web` repro exports:
    - `export memory kind=2 index=0`
    - `export tetra_main kind=0 index=3`
  - Entry-only controls export exactly the same names.
  - Native TOBJ control from the same repro source includes:
    - `symbol 'service_answer' offset=0 params=0 returns=1`
- Expected: a target that accepts `@export` should either emit the requested
  WASM export or reject the source with a clear target diagnostic such as
  `@export is not supported for wasm32-wasi/wasm32-web entry artifacts`.
  Silent omission is the unsafe option because build success implies the
  service endpoint exists.
- Evidence path:
  - `docs/backend/wasm_architecture.md:25` to `:33` defines a deterministic
    WOBJ `exports` list.
  - `docs/backend/wasm_architecture.md:70` to `:73` defines only target
    entrypoint exports, but the frontend/checker still accept ordinary
    `@export` attributes for WASM builds.
  - `compiler/internal/backend/wasm32_wasi/codegen.go:265` to `:274`
    hard-codes the WASI export section to exactly `memory` and `_start`.
  - `compiler/internal/backend/wasm32_web/codegen.go:254` to `:263`
    hard-codes the Web export section to exactly `memory` and `tetra_main`.
  - `compiler/internal/backend/wasm32_wasi/codegen_test.go:73` to `:82` and
    `compiler/internal/backend/wasm32_web/codegen_test.go:73` to `:82`
    assert only the fixed entry exports, with no coverage for accepted
    `IRFunc.ExportName` values.
  - Native object emission from the same source proves the `@export` attribute
    is present before the WASM backend boundary.
- Why it matters: WASM exports are the service endpoint surface consumed by
  hosts, JS loaders, WASI adapters, and deployment manifests. A successful build
  that drops `@export` creates a broken service artifact: callers cannot invoke
  the promised endpoint, generated bindings will disagree with source intent,
  and review tooling sees a green build instead of a target-unsupported ABI
  diagnostic.

### BUG-075 - WASM `core.sym_addr` lowers unresolved symbols to anonymous hash constants

- Area: `core.sym_addr`, `wasm32-wasi`, `wasm32-web`, unsafe `link`,
  unresolved symbol policy, symbol-token lowering, and generated host bindings.
- Severity: high; `core.sym_addr("missing_external")` passes `check` and
  builds for both WASM targets even though `missing_external` is neither a local
  function nor a configured host import/export. The emitted `.wasm` contains no
  raw symbol name, import, export, relocation, or manifest-visible dependency;
  it contains only the FNV-1a `i32.const` token `2313925087`
  (`-1981042209` signed). Native object output from the same source preserves a
  relocation named `missing_external`, and native executable linking rejects the
  unresolved symbol.
- Reproducer:
  - `/tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra`
- Controls:
  - `/tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_internal_control.tetra`
  - `/tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_empty_control.tetra`
  - `/tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_none_control.tetra`
- Commands:

```sh
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_empty_control.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.wasi.wasm /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.web.wasm /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra
python /tmp/tetra-bug-hunt/session-093/bughunt/inspect_wasm.py /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.wasi.wasm /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.web.wasm
GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.wasi.wasm
GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.web.wasm
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.native.tobj /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra
python /tmp/tetra-bug-hunt/session-093/bughunt/inspect_tobj.py /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.native.tobj
GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-bug-hunt/session-093/bughunt/out/missing_external.native.exe /tmp/tetra-bug-hunt/session-093/bughunt/wasm_sym_addr_missing_external_repro.tetra
```

- Observed:
  - Repro source:

```tetra
func main() -> Int
uses link:
    unsafe:
        var _p: ptr = core.sym_addr("missing_external")
    return 42
```

  - The repro passes `tetra check`; the empty-name control is still rejected
    with `sym_addr expects a non-empty symbol name`.
  - `wasm32-wasi` build succeeds. The artifact contains no raw
    `missing_external` bytes, imports only the fixed WASI functions
    `fd_write`/`proc_exit`, exports only `memory`/`_start`, and has
    `i32.const -1981042209/2313925087` in the function body.
  - `wasm32-web` build succeeds. The artifact contains no raw
    `missing_external` bytes, imports only the fixed Web functions
    `console_log`/`panic`, exports only `memory`/`tetra_main`, and has the same
    `i32.const -1981042209/2313925087` token.
  - `validate-wasm-imports` exits 0 for both WASI and Web artifacts because
    the unresolved symbol is not represented as an import.
  - Native TOBJ output from the same source preserves the unresolved boundary:
    `reloc kind=1 at=25 name='missing_external' addend=0`.
  - Native executable linking rejects the same source with
    `unresolved symbol 'missing_external'`.
- Expected: if WASM cannot link a `core.sym_addr` name, the build should reject
  unresolved names unless they are configured host imports, matching the
  documented relocation/link policy. If token lowering is intentional, the
  artifact must emit an explicit symbol-token table or host-binding metadata
  that keeps the original symbol auditable and resolvable; it must not replace
  a `link` dependency with a bare integer constant.
- Evidence path:
  - `docs/backend/wasm_architecture.md:35` to `:39` says unresolved symbols
    are allowed only for configured host imports and everything else is a
    compile error.
  - `docs/backend/wasm_architecture.md:60` to `:63` lists
    `core.sym_addr` token lowering as currently allowed WASM IR.
  - `docs/spec/runtime/unsafe.md:33` marks `core.sym_addr` as always unsafe with the
    `link` effect.
  - `compiler/internal/semantics/semantics_expressions.go:2012` to `:2023` and
    `compiler/internal/lower/lower_core.go:3291` to `:3304` validate only arity,
    string-literal shape, and non-empty names before emitting `IRSymAddr`.
  - `compiler/internal/backend/wasm32_wasi/codegen.go:514` to `:516` and
    `compiler/internal/backend/wasm32_web/codegen.go:581` to `:583` lower
    `IRSymAddr` directly to `writeI32Const(wasmSymbolToken(name))`.
  - `compiler/internal/backend/wasm32_wasi/codegen.go:1548` to `:1585` and
    `compiler/internal/backend/wasm32_web/codegen.go:1570` to `:1607` check
    only for empty names and in-module FNV token collisions.
  - `compiler/internal/backend/x64core/x64core_core.go:724` to `:730` shows the native
    path preserves `IRSymAddr` as a named relocation instead of erasing it.
- Why it matters: `core.sym_addr` is explicitly a `link` boundary. Erasing the
  original symbol into an unaudited 32-bit token means service manifests,
  sandbox hosts, JS loaders, WASI adapters, and review tools cannot see or
  resolve the dependency. It also makes external collisions and stale host
  mappings hard to diagnose, while a green build suggests the link dependency
  was handled.

## Probe Sessions

### 2026-05-18 - Session 001 - Microservice Edge/Security Harness

Planned focus:

- Networking policy helpers: port validation, fallback behavior, retry backoff
  overflow/negative attempts.
- Capability-gated filesystem checks: path edge cases and missing-file behavior.
- Memory/capability helpers: negative lengths and raw-pointer safety diagnostics.
- Actor/runtime-shaped service boundaries where supported by the current
  `v0.4.0` surface.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S001-001 | Harness setup | `mkdir -p /tmp/tetra-bug-hunt/session-001`; scratch files under `bughunt/`; symlinked repo `lib/` | Complete after correcting module-path and import-root assumptions | Keep scratch outside repo |
| S001-002 | Networking normal edges | `go run ./cli/cmd/tetra run .../net_choose_port_edges.tetra`; `.../net_backoff_negative_base.tetra` | Both printed `exit status 42` from success sentinels | No bug |
| S001-003 | Networking capped backoff overflow | `go run ./cli/cmd/tetra run .../net_backoff_cap_bypass.tetra` | Printed `exit status 43`; confirmed BUG-001 | Needs regression test/fix later |
| S001-004 | Filesystem health-check | `go run ./cli/cmd/tetra run .../fs_healthcheck.tetra` | Printed `exit status 42` from success sentinel | No bug |
| S001-005 | Memory negative copy length | `go run ./cli/cmd/tetra run .../mem_negative_copy.tetra` | Printed `exit status 42` from success sentinel | No bug |
| S001-006 | Memory OOB base store through local binary vs source | `./tetra run .../mem_oob_store.tetra`; `go run ./cli/cmd/tetra run .../mem_oob_store.tetra` | Local binary exited `42`; current source printed `exit status 2`; confirmed BUG-002 | Rebuild or stale-entrypoint guard needed |
| S001-007 | Actor-net source spoof from unregistered connection | `go run /tmp/tetra-bug-hunt/session-001/actornet_spoof_repro.go` | Printed `SPOOF_ROUTED source=1 dest=2 seq=99 payload=12345`; confirmed BUG-003 | Broker must require HELLO and source ownership |
| S001-008 | Filesystem embedded-NUL truncation | `go run ./cli/cmd/tetra check .../fs_nul_truncation.tetra`; `go run ./cli/cmd/tetra run .../fs_nul_truncation.tetra` | `check` passed; `run` printed `exit status 42`; confirmed BUG-004 | Reject NUL before host syscall |
| S001-009 | Memory offset+width OOB checks | `go run ./cli/cmd/tetra run .../mem_offset_width_oob.tetra`; `.../mem_offset_width_direct.tetra` | Both printed `exit status 2` | No bug in current source |
| S001-010 | Serialization clamp/unpack edges | `go run ./cli/cmd/tetra run .../serialization_edges.tetra` | Printed `exit status 42` from success sentinel | No bug |
| S001-011 | Crypto mix seed min-int overflow | `go run ./cli/cmd/tetra run .../crypto_mix_seed_minint.tetra` | Printed `exit status 42`; confirmed BUG-005 | Needs saturation/non-negative modulo or doc |
| S001-012 | Privacy consent literal forging | `go run ./cli/cmd/tetra check .../privacy_forge_consent_literal.tetra`; `go run ./cli/cmd/tetra run .../privacy_consent_smoke.tetra`; `go test ./compiler/... -run 'Privacy|Consent' -count=1` | Literal forge rejected with `type mismatch`; valid token smoke printed `exit status 55`; focused tests passed | No bug |
| S001-013 | Actor-net repeated HELLO identity lifecycle | `go run /tmp/tetra-bug-hunt/session-001/actornet_multi_hello_repro.go` | Printed `MULTI_HELLO_STALE_ROUTE source=3 dest=1 payload=77`; confirmed BUG-006 | Broker must bind one node per connection |

### 2026-05-18 - Session 002 - Eco Package Path and Manifest Surface

Planned focus:

- Project bundle pack/unpack path safety around existing output directories.
- Symlink and path traversal behavior before and after archive path
  normalization.
- Capsule source-root validation consistency between project loading, package
  generation, unpacking, and `validate-eco-unpack`.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S002-001 | Clean T4 project bundle baseline | `go run ./cli/cmd/tetra check .../project-t4/Tetra.capsule`; `go run ./cli/cmd/tetra eco pack --project ... -o .../demo-t4.todex`; `go run ./cli/cmd/tetra eco unpack .../demo-t4.todex -C .../out-clean-t4`; `go run ./tools/cmd/validate-eco-unpack --dir .../out-clean-t4` | All exited 0; validator produced no stderr | Baseline valid |
| S002-002 | Pre-existing symlink source root in unpack output | `ln -s .../escape-t4 .../out-symlink-t4/src`; `go run ./cli/cmd/tetra eco unpack .../demo-t4.todex -C .../out-symlink-t4`; `find .../escape-t4 -maxdepth 3 -printf '%y %p -> %l\n'` | `eco unpack` exited 0 and wrote `f .../escape-t4/main.t4`; confirmed BUG-007 | Unpack needs no-follow/inside-root writes |
| S002-003 | Validator after symlink escape | `go run ./tools/cmd/validate-eco-unpack --dir .../out-symlink-t4` | Failed with `missing T4 sources under src` after the out-of-root write had already happened | Validator cannot be relied on to prevent unpack write |
| S002-004 | Unsafe-only Capsule source root | `go run ./cli/cmd/tetra check .../project-only-unsafe-source/Tetra.capsule`; `go run ./cli/cmd/tetra eco pack --project ... -o .../only-unsafe-source.todex`; `go run ./cli/cmd/tetra eco unpack ... -C .../out-only-unsafe-source`; `go run ./tools/cmd/validate-eco-unpack --dir .../out-only-unsafe-source` | All exited 0 even though the manifest declares only `source "../outside"`; confirmed BUG-008 | Project loader and validator should reject unsafe roots |

### 2026-05-18 - Session 003 - Core Stdlib Duration Policy Edges

Planned focus:

- Stable `lib.core.time` helpers used by retry/deadline/timeout-style
  microservice examples.
- Positive overflow behavior at the `Int` boundary.
- Baseline normal duration arithmetic to separate harness failures from helper
  bugs.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S003-001 | Time helper normal edges | `go run ./cli/cmd/tetra check .../time_normal_edges.tetra`; `go run ./cli/cmd/tetra run .../time_normal_edges.tetra` | `check` exited 0; `run` printed `exit status 42` from success sentinel | No bug |
| S003-002 | `millis_from_seconds` positive overflow | `go run ./cli/cmd/tetra check .../time_millis_positive_overflow.tetra`; `go run ./cli/cmd/tetra run .../time_millis_positive_overflow.tetra` | `check` exited 0; `run` printed `exit status 42` because the positive seconds value became negative milliseconds; confirmed BUG-009 | Needs saturation/rejection/doc |
| S003-003 | `add_duration_ms` positive overflow to zero | `go run ./cli/cmd/tetra check .../time_add_positive_overflow_zero.tetra`; `go run ./cli/cmd/tetra run .../time_add_positive_overflow_zero.tetra` | `check` exited 0; `run` printed `exit status 42` because a mathematically positive sum wrapped negative and was clamped to `0`; confirmed BUG-009 | Needs overflow-aware add |

### 2026-05-18 - Session 004 - Heap Slice Constructor Edge Cases

Planned focus:

- Heap slice constructors used by microservice payload buffers.
- Empty slice behavior needed by collection and checksum helpers.
- Negative length handling before allocation and index bounds.
- Nearby island allocator and negative-index controls to separate root causes.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S004-001 | Positive heap slice baseline | `go run ./cli/cmd/tetra check .../slice_positive_baseline.tetra`; `go run ./cli/cmd/tetra run .../slice_positive_baseline.tetra` | `check` exited 0; `run` printed `exit status 42` from success sentinel | No bug |
| S004-002 | Empty heap `[]i32` collection scan | `go run ./cli/cmd/tetra check .../collections_empty_make_i32.tetra`; `go run ./cli/cmd/tetra run .../collections_empty_make_i32.tetra` | `check` exited 0; `run` printed `exit status 2`; confirmed BUG-010 | Heap make needs empty-slice path |
| S004-003 | Empty heap `[]u8` slice scan | `go run ./cli/cmd/tetra check .../slices_empty_make_u8.tetra`; `go run ./cli/cmd/tetra run .../slices_empty_make_u8.tetra` | `check` exited 0; `run` printed `exit status 2`; confirmed BUG-010 | Heap make needs empty-slice path |
| S004-004 | Negative heap `[]u8` length | `go run ./cli/cmd/tetra check .../heap_negative_make_u8.tetra`; `go run ./cli/cmd/tetra run .../heap_negative_make_u8.tetra` | `check` exited 0; `run` printed `exit status 42` after writing and reading `xs[0]`; confirmed BUG-011 | Reject negative lengths before `mmap` |
| S004-005 | Negative heap `[]i32` length | `go run ./cli/cmd/tetra check .../heap_negative_make_i32.tetra`; `go run ./cli/cmd/tetra run .../heap_negative_make_i32.tetra` | `check` exited 0; `run` printed `exit status 42` after writing and reading `xs[0]`; confirmed BUG-011 | Reject negative lengths before width scaling |
| S004-006 | Negative heap index store | `go run ./cli/cmd/tetra check .../slice_negative_index_store.tetra`; `go run ./cli/cmd/tetra run .../slice_negative_index_store.tetra` | `check` exited 0; `run` printed `exit status 1` from bounds failure | No bug |
| S004-007 | Empty island `[]u8` scan | `go run ./cli/cmd/tetra check .../island_empty_make_u8.tetra`; `go run ./cli/cmd/tetra run .../island_empty_make_u8.tetra` | `check` exited 0; `run` printed `exit status 42` from success sentinel | No bug; contrasts BUG-010 |
| S004-008 | Negative island `[]u8` length with store | `go run ./cli/cmd/tetra check .../island_negative_make_u8_store.tetra`; `go run ./cli/cmd/tetra run .../island_negative_make_u8_store.tetra` | `check` exited 0; `run` printed `exit status 1` | No heap-style bypass observed |

### 2026-05-18 - Session 005 - Runtime Logical Clock and Scheduler Edges

Planned focus:

- Deadline and sleep behavior used by task/actor microservices.
- Negative/zero delay baselines versus positive overflow.
- Builtin versus selfhost runtime parity for timing behavior.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S005-001 | Runtime time baseline | `go run ./cli/cmd/tetra check .../runtime_time_baseline.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../runtime_time_baseline.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../runtime_time_baseline.tetra` | `check` exited 0; both runtimes printed `exit status 42` from success sentinel | No bug |
| S005-002 | Positive deadline overflow | `go run ./cli/cmd/tetra check .../runtime_deadline_positive_overflow.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../runtime_deadline_positive_overflow.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../runtime_deadline_positive_overflow.tetra` | `check` exited 0; both runtimes printed `exit status 42` because `deadline_ms` returned a negative absolute deadline; confirmed BUG-012 | Needs overflow-aware deadline arithmetic |
| S005-003 | Timer-ready on overflowed deadline | `go run ./cli/cmd/tetra check .../runtime_timer_ready_overflow_deadline.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../runtime_timer_ready_overflow_deadline.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../runtime_timer_ready_overflow_deadline.tetra` | `check` exited 0; both runtimes printed `exit status 42` because the overflowed deadline was immediately ready; confirmed BUG-012 | Needs deadline range handling |
| S005-004 | Builtin sleep overflow skips return | `go run ./cli/cmd/tetra check .../runtime_sleep_overflow_exits_zero.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../runtime_sleep_overflow_exits_zero.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../runtime_sleep_overflow_exits_zero.tetra` | `check` exited 0; builtin runtime exited 0 with no `exit status`; selfhost printed `exit status 42`; confirmed BUG-013 | Builtin scheduler must not terminate sleeping only task on overflowed wake deadline |

### 2026-05-18 - Session 006 - Task Handle and Task Group Forgery/Parity Edges

Planned focus:

- Task handle opacity: whether ordinary code can forge `task.i32` values.
- Runtime bounds/lifecycle checks when task poll/join receives invalid handles.
- Task-group scalar handle construction and runtime parity between builtin and
  selfhost.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S006-001 | Graphify/source navigation for task resources | `mcp__graphify__.query_graph ... task group task handle runtime invalid forged stale handle ...`; `rg -n "task\\.group|task\\.i32|task_group_|task_join"`; source reads in `compiler/internal/semantics` and `compiler/internal/actorsrt` | Found ownership diagnostics for normal aliases, `task.i32` public struct fields, and runtime poll/join paths that trust actor handle slots | Probe constructor paths directly |
| S006-002 | Direct integer `task.group` forgery | `go run ./cli/cmd/tetra check .../task_group_assign_int_rejected.tetra` | Rejected with `type mismatch: expected 'task.group', got 'i32'` | No direct scalar group forge |
| S006-003 | Call-style forged `task.i32(...)` | `go run ./cli/cmd/tetra check .../task_handle_struct_literal_error_short_circuit.tetra`; `go run ./cli/cmd/tetra build -o .../error_short_circuit.bin .../task_handle_struct_literal_error_short_circuit.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../task_handle_struct_literal_error_short_circuit.tetra` | `check` succeeded; `build`/`run` failed with `missing signature for 'task.i32'`; confirmed BUG-015 | Align check and build dependency resolution |
| S006-004 | Brace-literal forged `task.i32` with invalid nonzero actor handle | `go run ./cli/cmd/tetra check .../task_handle_brace_literal_poll_fake_nonzero.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../task_handle_brace_literal_poll_fake_nonzero.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../task_handle_brace_literal_poll_fake_nonzero.tetra`; repeated for `.../task_handle_brace_literal_join_until_fake_nonzero.tetra` | `check` succeeded; builtin printed `exit status 255`; selfhost exited 0 with no `exit status`; confirmed BUG-014 | Make task handles opaque or validate runtime handles |
| S006-005 | Forged `task.i32` error-slot short-circuit control | `go run ./cli/cmd/tetra run -runtime builtin .../task_handle_brace_literal_error_short_circuit.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../task_handle_brace_literal_error_short_circuit.tetra` | Both printed `exit status 42`, proving fake actor slot is trusted only when `error == 0` | Supports BUG-014 |
| S006-006 | Task-group open over capacity | `go run ./cli/cmd/tetra check .../task_group_open_over_capacity_zero.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../task_group_open_over_capacity_zero.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../task_group_open_over_capacity_zero.tetra` | `check` succeeded; builtin printed `exit status 42` for ninth group returning/statusing as zero; selfhost failed on missing task-group symbol | Capacity behavior needs policy review; selfhost gap tracked as BUG-016 |
| S006-007 | Minimal task-group selfhost parity | `go run ./cli/cmd/tetra check .../task_group_selfhost_missing_symbol.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../task_group_selfhost_missing_symbol.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../task_group_selfhost_missing_symbol.tetra` | `check` succeeded; builtin printed `exit status 42`; selfhost failed with `runtime object missing required symbol '__tetra_task_group_open'`; confirmed BUG-016 | Implement or reject selfhost task groups |

### 2026-05-18 - Session 007 - Actor Typed Message and Privacy Wrapper Edges

Planned focus:

- Whether raw actor messages can cross into typed enum receives.
- Whether typed actor message slot counts and tags are validated at receive
  time.
- Whether secret/privacy wrapper values can be directly constructed.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S007-001 | Graphify/source navigation for actor/privacy surfaces | `mcp__graphify__.query_graph ... actor handle consent.token secret.i32 ...`; `rg -n "secret\\.i32|consent\\.token|send_typed|recv_typed|send_msg"`; source reads in `compiler/internal/lower/lower_core.go`, `compiler/internal/actorsrt/actorsrt_core.go`, and `compiler/selfhostrt/actors_sysv.tetra` | Found typed actor receive lowering reads tag and payload slots from shared mailbox without count validation; found `secret.i32` public struct with slot count 1 and no fields | Create raw-to-typed and secret literal probes |
| S007-002 | Typed actor baseline | `go run ./cli/cmd/tetra check .../actor_typed_send_inc_baseline.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../actor_typed_send_inc_baseline.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../actor_typed_send_inc_baseline.tetra` | `check` succeeded; both runtimes printed `exit status 42` for `CounterMsg.inc(20, 22)` | Baseline valid |
| S007-003 | Raw `send_msg` spoof of typed payload case | `go run ./cli/cmd/tetra check .../actor_raw_send_msg_spoofs_typed_inc.tetra`; `go run ./cli/cmd/tetra run -runtime builtin .../actor_raw_send_msg_spoofs_typed_inc.tetra`; `go run ./cli/cmd/tetra run -runtime selfhost .../actor_raw_send_msg_spoofs_typed_inc.tetra` | `check` succeeded; both runtimes printed `exit status 42`, proving raw tag 0 decoded as `CounterMsg.inc(7, 0)` | Confirmed BUG-017 |
| S007-004 | Raw `send_msg` spoof of typed zero-payload case | `go run ./cli/cmd/tetra check .../actor_raw_send_msg_spoofs_typed_reset.tetra`; `go run ./cli/cmd/tetra run -runtime builtin ...`; `go run ./cli/cmd/tetra run -runtime selfhost ...` | `check` succeeded; both runtimes printed `exit status 42`, proving raw tag 1 decoded as `CounterMsg.reset` | Confirms BUG-017 |
| S007-005 | Raw `send_msg` spoof of typed island payload | `go run ./cli/cmd/tetra check .../actor_raw_send_msg_spoofs_typed_island_payload.tetra`; `go run ./cli/cmd/tetra run -runtime builtin ...`; `go run ./cli/cmd/tetra run -runtime selfhost ...` | `check` succeeded; both runtimes printed `exit status 42`, proving raw integer send entered `MoveMsg.take(island)` | Confirms BUG-017 security/resource angle |
| S007-006 | Raw invalid typed enum tag | `go run ./cli/cmd/tetra check .../actor_raw_send_msg_invalid_typed_tag.tetra`; `go run ./cli/cmd/tetra run -runtime builtin ...`; `go run ./cli/cmd/tetra run -runtime selfhost ...` | `check` succeeded; both runtimes printed `exit status 42` after tag 99 fell through the typed `match` | Confirms missing tag validation in BUG-017 |
| S007-007 | Empty `secret.i32{}` literal | `go run ./cli/cmd/tetra check .../secret_empty_brace_literal_check_build_split.tetra`; `go run ./cli/cmd/tetra build -o .../secret_empty.bin ...`; `go run ./cli/cmd/tetra run -runtime builtin ...` | `check` succeeded; build/run failed with `slot mismatch for 'fake'`; confirmed BUG-018 | Checker should reject or model secret wrapper consistently |

### 2026-05-18 - Session 008 - Runtime Arithmetic Crash Edges

Planned focus:

- Runtime `Int` arithmetic used by request parsing, rate limits, and quota
  calculations.
- Division/modulo by a value only known at runtime, contrasted with global const
  diagnostics.
- Native crash behavior for x64 `idiv` processor traps.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S008-001 | Graphify/source navigation for stdlib and arithmetic surfaces | `mcp__graphify__.query_graph ... serialization json strings payload helpers ...`; `rg -n "division|divide|mod|zero|IRDivI32|IRModI32"`; source reads in `lib/core/*`, `compiler/internal/semantics/semantics_checker.go`, and backend emitters | Found const-only division/modulo diagnostics and direct backend `idiv`/`i32.div_s` emission without arithmetic guards | Probe runtime expressions directly |
| S008-002 | Compile-time divide-by-zero control | `go run ./cli/cmd/tetra check .../const_div_zero_control.tetra` | Rejected with `division by zero in global const expression` | No bug; confirms existing const guard |
| S008-003 | Runtime divide by zero | `go run ./cli/cmd/tetra check .../runtime_div_zero.tetra`; `go run ./cli/cmd/tetra run .../runtime_div_zero.tetra`; `go run ./cli/cmd/tetra build -o .../runtime_div_zero.bin ...`; `timeout 5s .../runtime_div_zero.bin` | `check`/`build` succeeded; `run` printed `exit status 255`; direct binary exited `136` and dumped core; confirmed BUG-019 | Add runtime guard or documented trap policy |
| S008-004 | Runtime modulo by zero | `go run ./cli/cmd/tetra check .../runtime_mod_zero.tetra`; `go run ./cli/cmd/tetra run .../runtime_mod_zero.tetra`; `go run ./cli/cmd/tetra build -o .../runtime_mod_zero.bin ...`; `timeout 5s .../runtime_mod_zero.bin` | `check`/`build` succeeded; `run` printed `exit status 255`; direct binary exited `136` and dumped core; confirmed BUG-019 | Same guard as division |
| S008-005 | Runtime signed division overflow edge | `go run ./cli/cmd/tetra check .../runtime_min_div_neg_one.tetra`; `go run ./cli/cmd/tetra run .../runtime_min_div_neg_one.tetra`; `go run ./cli/cmd/tetra build -o .../runtime_min_div_neg_one.bin ...`; `timeout 5s .../runtime_min_div_neg_one.bin`; repeated for `.../runtime_min_mod_neg_one.tetra` | Both min-int reproducers passed `check`/`build`, `run` printed `exit status 255`, and direct binaries exited `136` with core dumps | Confirms BUG-019 beyond zero divisor |
| S008-006 | Positive max integer literal control | `go run ./cli/cmd/tetra check .../literal_i32_max_control.tetra`; `go run ./cli/cmd/tetra run .../literal_i32_max_control.tetra` | `check` succeeded; `run` printed `exit status 42` for `2147483647 > 0` | Baseline valid |
| S008-007 | Out-of-range positive `Int` literals | `go run ./cli/cmd/tetra check .../literal_i32_positive_overflow.tetra`; `go run ./cli/cmd/tetra run ...`; repeated for `.../literal_i32_large_wrap_minus_one.tetra` and `.../global_const_i32_positive_overflow.tetra` | All passed `check`; `2147483648` behaved as negative, `4294967295` behaved as `-1`, and the global const variant also wrapped; confirmed BUG-020 | Reject out-of-range literals before `int32` cast |

### 2026-05-18 - Session 009 - Defer and Typed Error Cleanup Edges

Planned focus:

- Cleanup behavior for service handlers that use `defer` plus typed errors.
- Difference between explicit `throw`, successful `try`, and propagated
  `try` failure.
- Whether `defer` body validation catches indirect control transfer through
  expressions.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S009-001 | Graphify/source navigation for defer/throws cleanup | `mcp__graphify__.query_graph ... defer throws cleanup semantics ...`; `mcp__graphify__.get_neighbors("TestDeferRunsOnNestedReturnBeforeOuterCleanup()")`; source reads in `compiler/tests/semantics/semantics_core_language_test.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/semantics/semantics_checker.go` | Found tests for explicit return/throw cleanup, and found `TryExpr` error-path lowering emits `IRReturn` without `emitDeferredFramesSince` | Probe `try` propagation |
| S009-002 | Explicit throw cleanup control | `go run ./cli/cmd/tetra check .../defer_throw_statement_control.tetra`; `go run ./cli/cmd/tetra run .../defer_throw_statement_control.tetra` | `check` succeeded; `run` printed `exit status 42` from branch proving `defer` set `cleaned = 1` before explicit `throw` propagated | Baseline valid |
| S009-003 | Successful try cleanup control | `go run ./cli/cmd/tetra check .../defer_try_success_control.tetra`; `go run ./cli/cmd/tetra run .../defer_try_success_control.tetra` | `check` succeeded; `run` printed `exit status 42` from branch proving `try ok()` returned value and `defer` set `cleaned = 1` on normal return | Baseline valid |
| S009-004 | Propagated try skips active defer | `go run ./cli/cmd/tetra check .../defer_try_error_skips_cleanup.tetra`; `go run ./cli/cmd/tetra build -o .../defer_try_error_skips_cleanup.bin ...`; `go run ./cli/cmd/tetra run .../defer_try_error_skips_cleanup.tetra` | `check`/`build` succeeded; `run` printed `exit status 42` from branch proving `cleaned == 0` after `try fail()` propagated; confirmed BUG-021 | `TryExpr` error path must emit active defer frames |
| S009-005 | Propagating try inside defer body | `go run ./cli/cmd/tetra check .../defer_body_try_can_throw.tetra`; `go run ./cli/cmd/tetra build -o .../defer_body_try_can_throw.bin ...`; `go run ./cli/cmd/tetra run .../defer_body_try_can_throw.tetra` | `check` succeeded; build/run failed with `ir verifier: handler instr 7: return expects 2 stack slots, have 4`; confirmed BUG-022 | Reject/handle `TryExpr` in defer-body control validation |

### 2026-05-18 - Session 010 - Heap Slice Constructor Type Coverage

Planned focus:

- Extend heap slice constructor edge coverage beyond `[]u8` and `[]i32`.
- Check native-first `[]u16` and `[]bool` heap constructors for the same
  zero/negative length behavior.
- Use island constructors as controls for empty slices.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S010-001 | Graphify/source navigation for u16/bool slice constructors | `mcp__graphify__.query_graph ... make_u16 make_bool slice constructor ...`; `rg -n "make_u16|make_bool|IRMakeSliceU16|\\[\\]bool"`; source reads in `compiler/internal/lower/lower_core.go`, `compiler/internal/backend/x64abi/sysv_unix.go`, and `compiler/tests/semantics/semantics_memory_surface_test.go` | Found `make_u16` lowers to `IRMakeSliceU16`, `make_bool` lowers through `IRMakeSliceI32`, and both reach the same heap `mmap` path with no length guard | Probe zero/negative heap lengths |
| S010-002 | Positive `[]u16` and `[]bool` heap baselines | `go run ./cli/cmd/tetra check .../slice_u16_positive_baseline.tetra`; `go run ./cli/cmd/tetra run .../slice_u16_positive_baseline.tetra`; repeated for `.../slice_bool_positive_baseline.tetra` | Both passed `check`; both `run` commands printed `exit status 42` from success sentinels | Baseline valid |
| S010-003 | Empty heap `[]u16` and `[]bool` constructors | `go run ./cli/cmd/tetra check .../slice_u16_empty_heap.tetra`; `go run ./cli/cmd/tetra run .../slice_u16_empty_heap.tetra`; repeated for `.../slice_bool_empty_heap.tetra` | Both passed `check`; both `run` commands printed `exit status 2`; extends BUG-010 to `make_u16(0)` and `make_bool(0)` | Heap make needs empty-slice path for all supported element types |
| S010-004 | Negative heap `[]u16` and `[]bool` lengths | `go run ./cli/cmd/tetra check .../slice_u16_negative_heap_store.tetra`; `go run ./cli/cmd/tetra run .../slice_u16_negative_heap_store.tetra`; repeated for `.../slice_bool_negative_heap_store.tetra` | Both passed `check`; both `run` commands printed `exit status 42` after writing and reading element `0`; extends BUG-011 to `make_u16(-1)` and `make_bool(-1)` | Reject negative heap lengths for all supported element types |
| S010-005 | Empty island `[]u16` and `[]bool` controls | `go run ./cli/cmd/tetra check .../island_u16_empty_control.tetra`; `go run ./cli/cmd/tetra run .../island_u16_empty_control.tetra`; repeated for `.../island_bool_empty_control.tetra` | Both passed `check`; both `run` commands printed `exit status 42` | Confirms heap-specific zero-length regression |

### 2026-05-18 - Session 011 - Small Integer Payload Range Edges

Planned focus:

- Local and argument range validation for `u8`/`u16` request-payload fields.
- Raw byte stores and `[]u8` index stores with out-of-range integer values.
- Contrast local behavior with existing global initializer range diagnostics.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S011-001 | Graphify/source navigation for small integer range validation | `mcp__graphify__.query_graph ... u8 u16 local literal range validation ...`; `rg -n "validate.*Range|within 0..255|typesCompatibleWithNullPtr"`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/backend/x64core/x64core_core.go` | Found global-only `u8`/`u16` range validation, broad `isInt32Like` compatibility for locals/args, and byte store lowering that writes low-byte registers | Probe local and store behavior |
| S011-002 | Valid local max controls | `go run ./cli/cmd/tetra check .../local_u8_max_control.tetra`; `go run ./cli/cmd/tetra run .../local_u8_max_control.tetra`; repeated for `.../local_u16_max_control.tetra` | Both passed `check`; both `run` commands printed `exit status 42` from max-value success sentinels | Baseline valid |
| S011-003 | Out-of-range local `u8` values | `go run ./cli/cmd/tetra check .../local_u8_out_of_range_256.tetra`; `go run ./cli/cmd/tetra run .../local_u8_out_of_range_256.tetra`; repeated for `.../local_u8_negative_literal.tetra` | Both passed `check`; `run` printed `exit status 42` proving `u8` local held `256` and `-1`; confirmed BUG-023 | Add local/assignment range validation |
| S011-004 | Out-of-range local `u16` value | `go run ./cli/cmd/tetra check .../local_u16_out_of_range_70000.tetra`; `go run ./cli/cmd/tetra run .../local_u16_out_of_range_70000.tetra` | Passed `check`; `run` printed `exit status 42` proving `u16` local held `70000`; confirmed BUG-023 | Add `u16` local/assignment range validation |
| S011-005 | Raw `store_u8` out-of-range argument | `go run ./cli/cmd/tetra check .../store_u8_out_of_range_truncates.tetra`; `go run ./cli/cmd/tetra run .../store_u8_out_of_range_truncates.tetra` | Passed `check`; `run` printed `exit status 42` after storing `300` and reading back `44`; confirmed BUG-023 | Reject or require explicit wrapping conversion |
| S011-006 | `[]u8` index store out-of-range assignment | `go run ./cli/cmd/tetra check .../slice_u8_store_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../slice_u8_store_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42` after assigning `300` and reading back `44`; confirmed BUG-023 | Enforce element range at assignment/checker boundary |

### 2026-05-18 - Session 012 - Small Integer Args and Fixed Array Runtime Edges

Planned focus:

- Extend BUG-023 coverage from local values and `[]u8` stores to function
  arguments and `[]u16` stores.
- Probe fixed-array element assignment through accepted zeroed global struct
  fields.
- Separate small-integer range behavior from fixed-array runtime
  representation bugs.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S012-001 | Graphify/source navigation for fixed arrays and index store lowering | `mcp__graphify__.query_graph ... fixed arrays u8 u16 index store lowering ...`; `rg -n "TypeArray|IRIndexLoad|IRIndexStore|lowerIndexStoreKind"`; source reads in `compiler/tests/semantics/semantics_core_language_test.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/backend/x64core/x64core_core.go` | Found fixed-array build smoke tests, `TypeArray` modeled as `ptr`/`len` with `SlotCount: 2`, and common slice/fixed-array index lowering | Probe accepted runtime paths |
| S012-002 | Out-of-range `u8` and `u16` function arguments | `go run ./cli/cmd/tetra check .../arg_u8_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../arg_u8_out_of_range.tetra`; repeated for `.../arg_u16_out_of_range.tetra` | Both passed `check`; both `run` commands printed `exit status 42`, proving the callee received `256` as `u8` and `70000` as `u16`; extends BUG-023 | Validate parameter binding/conversion ranges |
| S012-003 | `[]u16` index store out-of-range assignment | `go run ./cli/cmd/tetra check .../slice_u16_store_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../slice_u16_store_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42` after assigning `70000` and reading back `4464`; extends BUG-023 | Enforce `u16` element range at assignment/checker boundary |
| S012-004 | Fixed-array direct local zero init attempt | `go run ./cli/cmd/tetra check .../array_u8_max_control.tetra` before rewriting through a global field | Rejected `var xs: [1]u8 = 0` with `type mismatch: expected '[1]u8', got 'i32'` | Use accepted global-field construction from existing tests |
| S012-005 | Fixed-array global field read/index access | `go run ./cli/cmd/tetra check .../array_int_global_field_read_traps.tetra`; `go run ./cli/cmd/tetra run .../array_int_global_field_read_traps.tetra` | Passed `check`; `run` printed `exit status 1` for reading `box.items[0]` from `var box: IntBox`; confirmed BUG-024 | Initialize fixed-array backing storage/length or reject unsupported field shape |
| S012-006 | Fixed-array global/local assignment controls | `go run ./cli/cmd/tetra check .../array_int_global_field_store.tetra`; `go run ./cli/cmd/tetra run .../array_int_global_field_store.tetra`; `go run ./cli/cmd/tetra check .../array_int_local_param_store.tetra`; `go run ./cli/cmd/tetra run .../array_int_local_param_store.tetra` | Both passed `check`; both `run` commands printed `exit status 1` at index assignment/read, confirming BUG-024 beyond read-only access | Add runtime regression tests for accepted fixed-array MVP surface |

### 2026-05-18 - Session 013 - Global Struct Field Assignment Corruption

Planned focus:

- Verify whether non-index global field assignments use global or local storage
  in lowering.
- Compare direct global field writes with local field writes and whole-global
  assignment controls.
- Check whether the issue is verifier-only or can silently corrupt locals when
  local slots are present.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S013-001 | Graphify/source navigation for global field assignment lowering | `mcp__graphify__.query_graph ... global struct field assignment lowering IRStoreLocal IRStoreGlobal ...`; `rg -n "AssignStmt|resolveLValue|IRStoreLocal|IRStoreGlobal"`; source reads in `compiler/internal/lower/lower_core.go` and nearby tests | Found `resolveLValue` marks global targets, whole-global assignment emits `IRStoreGlobal`, but general field assignment always emits `IRStoreLocal` | Probe runtime behavior |
| S013-002 | Local field assignment and whole-global controls | `go run ./cli/cmd/tetra check .../local_struct_field_assignment_control.tetra`; `go run ./cli/cmd/tetra run .../local_struct_field_assignment_control.tetra`; repeated for `.../global_struct_whole_assignment_control.tetra` | Both controls passed `check`; both `run` commands printed `exit status 42` | Confirms struct fields and whole-global assignment work outside the suspect path |
| S013-003 | Global field assignment with no locals | `go run ./cli/cmd/tetra check .../global_struct_field_assignment_ignored.tetra`; `go run ./cli/cmd/tetra run .../global_struct_field_assignment_ignored.tetra`; repeated for nested `.../global_nested_struct_field_assignment_ignored.tetra` | Both passed `check`; both `run` commands failed IR verification with `local slot 0 out of bounds (locals=0)` | Confirmed BUG-025 verifier-failure mode |
| S013-004 | Global first-field assignment corrupts local slot 0 | `go run ./cli/cmd/tetra check .../global_struct_field_assignment_corrupts_local.tetra`; `go run ./cli/cmd/tetra run .../global_struct_field_assignment_corrupts_local.tetra` | Passed `check`; `run` printed `exit status 42` from branch proving `marker == 42` and `box.value == 0` after `box.value = 42` | Confirmed BUG-025 silent-corruption mode |
| S013-005 | Global second-field assignment corrupts local slot 1 | `go run ./cli/cmd/tetra check .../global_struct_second_field_assignment_corrupts_second_local.tetra`; `go run ./cli/cmd/tetra run .../global_struct_second_field_assignment_corrupts_second_local.tetra` | Passed `check`; `run` printed `exit status 42` from branch proving field offset `1` wrote `second == 42` while global `box.value` stayed `0` | Store global fields with `IRStoreGlobal` when `target.Global` is true |

### 2026-05-18 - Session 014 - Compound Indexed Counter Side Effects

Planned focus:

- Test compound assignment semantics on indexed service counters.
- Compare simple `xs[0] += n` and explicit-temp update with side-effecting
  index helper `next()`.
- Determine whether the issue is just value skew or can become a bounds trap.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S014-001 | Graphify/source navigation for compound assignment desugaring | `mcp__graphify__.query_graph ... compound assignment += lowering AssignStmt ...`; `rg -n "CompoundValue|cloneCompoundTarget|IRIndexStore"`; source reads in `compiler/internal/frontend/frontend_core.go`, `compiler/internal/lower/lower_core.go`, and `compiler/tests/semantics/semantics_core_language_test.go` | Found parser desugars `+=` into an assignment whose RHS contains a cloned target; tests cover field/index smoke without side-effecting targets | Probe target evaluation count |
| S014-002 | Non-side-effecting and explicit-temp controls | `go run ./cli/cmd/tetra check .../compound_index_no_side_effect_control.tetra`; `go run ./cli/cmd/tetra run .../compound_index_no_side_effect_control.tetra`; repeated for `.../compound_index_explicit_temp_control.tetra` | Both passed `check`; both `run` commands printed `exit status 42` | Baseline valid |
| S014-003 | Side-effecting index compound assignment | `go run ./cli/cmd/tetra check .../compound_index_side_effect_double_eval.tetra`; `go run ./cli/cmd/tetra run .../compound_index_side_effect_double_eval.tetra` | Passed `check`; `run` printed `exit status 42` from branch proving `next()` executed twice and wrote `xs[1] + 2` into `xs[0]`; confirmed BUG-026 | Stage compound assignment targets once |
| S014-004 | Side-effecting index compound assignment to length-one slice | `go run ./cli/cmd/tetra check .../compound_index_side_effect_oob.tetra`; `go run ./cli/cmd/tetra run .../compound_index_side_effect_oob.tetra` | Passed `check`; `run` printed `exit status 1` because the cloned RHS target evaluated `next()` again and loaded index `1` from a length-one slice; confirmed BUG-026 bounds-trap mode | Add semantic or lowering guard for side-effecting compound targets |

### 2026-05-18 - Session 015 - Bool Payload Controls

Planned focus:

- Check whether `Bool` shares the same over-broad integer compatibility as
  `u8`/`u16`.
- Validate local, function argument, and `[]bool` assignment boundaries.
- Contrast legacy integer conditions with strict `Bool` payload storage.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S015-001 | Graphify/source navigation for `Bool` compatibility | `mcp__graphify__.query_graph ... Bool type range validation ...`; `rg -n "isInt32Like|Bool|make_bool|type mismatch: expected 'bool'"`; source reads in `compiler/internal/semantics/semantics_core.go` and `compiler/tests/semantics/semantics_memory_surface_test.go` | Found `isInt32Like` excludes `bool`, and existing tests reject `Bool = 1` / `[]bool = 1` | Use as control, not a bug |
| S015-002 | Local/argument/slice `Bool` integer rejection | `go run ./cli/cmd/tetra check .../bool_local_int_rejected.tetra`; repeated for `.../bool_arg_int_rejected.tetra` and `.../bool_slice_int_rejected.tetra` | All rejected with type mismatch diagnostics; no BUG entry | Bool payload boundary does not share BUG-023 |
| S015-003 | Integer condition control | `go run ./cli/cmd/tetra check .../int_condition_control.tetra`; `go run ./cli/cmd/tetra run .../int_condition_control.tetra` | Passed `check`; `run` printed `exit status 42`; this matches `isConditionType` allowing `i32` conditions | No bug logged |

### 2026-05-18 - Session 016 - Enum Payload Small Integer Range Edges

Planned focus:

- Check enum/message constructors as another service-boundary path for
  `u8`/`u16` payloads.
- Contrast `u8`/`u16` payload acceptance with `Bool` payload rejection.
- Extend BUG-023 only if enum match bindings preserve out-of-range payloads.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S016-001 | Graphify/source navigation for enum payload checking/lowering | `mcp__graphify__.query_graph ... enum value construction ...`; `rg -n "resolveEnumCaseConstructorCall|PayloadTypes|typesCompatibleWithNullPtr"`; source reads in `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/internal/lower/lower_core.go` | Found enum case constructor payload validation uses `typesCompatibleWithNullPtr`, and lowering stores payload slots as-is | Probe payload range boundaries |
| S016-002 | Enum `u8` max control | `go run ./cli/cmd/tetra check .../enum_u8_payload_max_control.tetra`; `go run ./cli/cmd/tetra run .../enum_u8_payload_max_control.tetra` | Passed `check`; `run` printed `exit status 42` for `Packet.byte(255)` | Baseline valid |
| S016-003 | Enum `u8` out-of-range payload | `go run ./cli/cmd/tetra check .../enum_u8_payload_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../enum_u8_payload_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving match binding saw `b == 300`; extends BUG-023 | Validate enum payload ranges |
| S016-004 | Enum `u16` out-of-range payload | `go run ./cli/cmd/tetra check .../enum_u16_payload_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../enum_u16_payload_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving match binding saw `w == 70000`; extends BUG-023 | Validate enum payload ranges |
| S016-005 | Enum `Bool` payload integer control | `go run ./cli/cmd/tetra check .../enum_bool_payload_int_rejected.tetra` | Rejected with `enum case 'Packet.flag' payload 1 expects 'bool', got 'i32'` | Confirms bug is small-int compatibility, not all enum payloads |

### 2026-05-18 - Session 017 - Global Const Overflow Range Gate

Planned focus:

- Check whether global `u8`/`u16` const-expression range validation happens
  before or after `int32` constant folding.
- Compare direct literal and simple out-of-range expression controls with an
  overflowing expression that mathematically lands outside every small type.
- Determine whether the issue is small-type-only or affects general `Int`
  global constants.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S017-001 | Graphify/source navigation for global const/range validation | `mcp__graphify__.query_graph ... validateGlobalIntLikeRange u8 u16 global const initializer ...`; `rg -n "validateGlobalIntLikeRange|evalGlobalConstI32|constI32"`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/tests/semantics/semantics_core_language_test.go` | Found global `u8`/`u16` validation exists, but `evalGlobalConstI32` computes binary arithmetic directly in `int32` before `validateGlobalIntLikeRange` sees the value | Probe overflow-wrapped expressions |
| S017-002 | Direct and simple-expression out-of-range controls | `go run ./cli/cmd/tetra check .../global_u8_direct_out_of_range_control.tetra`; `go run ./cli/cmd/tetra check .../global_u8_simple_expr_out_of_range_control.tetra` | Both rejected with `global var 'b' initializer must be within 0..255 for type u8` | Confirms ordinary global range checks are active |
| S017-003 | Global `UInt8` overflow-wrapped expression | `go run ./cli/cmd/tetra check .../global_u8_wrapped_const_expr.tetra`; `go run ./cli/cmd/tetra run .../global_u8_wrapped_const_expr.tetra` | Passed `check`; `run` printed `exit status 42`, proving `65536 * 65536` was folded/stored as `0`; confirmed BUG-027 | Detect const-expression overflow before narrowing |
| S017-004 | Global `UInt16` overflow-wrapped expression | `go run ./cli/cmd/tetra check .../global_u16_wrapped_const_expr.tetra`; `go run ./cli/cmd/tetra run .../global_u16_wrapped_const_expr.tetra` | Passed `check`; `run` printed `exit status 42`, proving the same wrapped-zero path for `UInt16`; confirmed BUG-027 | Add small-int global regression coverage |
| S017-005 | Global `Int` overflow-wrapped expression | `go run ./cli/cmd/tetra check .../global_i32_wrapped_const_expr.tetra`; `go run ./cli/cmd/tetra run .../global_i32_wrapped_const_expr.tetra` | Passed `check`; `run` printed `exit status 42`, proving unchecked `int32` constant folding is the root issue | Consider exact/wider const arithmetic for all global const expressions |

### 2026-05-18 - Session 018 - Function Return Small Integer Range Edges

Planned focus:

- Check whether `-> UInt8`/`-> UInt16` return statements validate numeric
  ranges or only structural type compatibility.
- Compare max-value and `Bool` return controls with out-of-range small integer
  returns.
- Confirm whether callers observe the full invalid value or a truncated one.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S018-001 | Graphify/source navigation for return type checking | `mcp__graphify__.query_graph ... function return type checking u8 u16 ...`; `rg -n "ReturnStmt|typesCompatibleWithNullPtr"`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, and `compiler/internal/semantics/semantics_core.go` | Found `ReturnStmt` validation eventually uses `typesCompatibleWithNullPtr(returnType, tname, s.Value)` and `isInt32Like` compatibility covers `i32`, `u8`, and `u16` without range checks | Probe runtime return values |
| S018-002 | `UInt8` max return control | `go run ./cli/cmd/tetra check .../return_u8_max_control.tetra`; `go run ./cli/cmd/tetra run .../return_u8_max_control.tetra` | Passed `check`; `run` printed `exit status 42` for `return 255` from `-> UInt8` | Baseline valid |
| S018-003 | Out-of-range `UInt8` return | `go run ./cli/cmd/tetra check .../return_u8_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../return_u8_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving the caller saw returned `UInt8` value `300`; extends BUG-023 | Validate return expression ranges |
| S018-004 | Out-of-range `UInt16` return | `go run ./cli/cmd/tetra check .../return_u16_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../return_u16_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving the caller saw returned `UInt16` value `70000`; extends BUG-023 | Validate return expression ranges |
| S018-005 | `Bool` integer return control | `go run ./cli/cmd/tetra check .../return_bool_int_rejected_control.tetra` | Rejected with `return type mismatch: expected 'bool', got 'i32'` | Confirms this follows small-int compatibility rather than all return types |

### 2026-05-18 - Session 019 - Struct Field Small Integer Range Edges

Planned focus:

- Check struct DTO/state fields as another boundary for `UInt8`/`UInt16`
  payloads.
- Compare brace-style and call-style struct constructors, plus mutable field
  assignment.
- Use a `Bool` field control to confirm this is the small-int compatibility
  hole rather than all field types accepting integers.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S019-001 | Graphify/source navigation for struct field validation | `mcp__graphify__.query_graph ... struct constructor field payload type checking ...`; source reads in `compiler/internal/semantics/semantics_expressions.go` and `compiler/internal/semantics/semantics_checker.go` | Found brace-style struct literals, call-style struct constructors, and field assignments all rely on `typesCompatibleWithNullPtr` for field/target type compatibility | Probe runtime field values |
| S019-002 | `UInt8` struct constructor max control | `go run ./cli/cmd/tetra check .../struct_u8_constructor_max_control.tetra`; `go run ./cli/cmd/tetra run .../struct_u8_constructor_max_control.tetra` | Passed `check`; `run` printed `exit status 42` for `Header{byte: 255}` | Baseline valid |
| S019-003 | Brace-style `UInt8`/`UInt16` out-of-range field initializers | `go run ./cli/cmd/tetra check .../struct_u8_constructor_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../struct_u8_constructor_out_of_range.tetra`; repeated for `.../struct_u16_constructor_out_of_range.tetra` | Both passed `check`; both `run` commands printed `exit status 42`, proving struct fields held `300` and `70000` unchanged; extends BUG-023 | Validate struct field initializer ranges |
| S019-004 | Call-style `UInt8` out-of-range field initializer | `go run ./cli/cmd/tetra check .../struct_u8_call_constructor_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../struct_u8_call_constructor_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving `Header(byte: 300)` follows the same path | Validate call-style constructor field ranges |
| S019-005 | Mutable `UInt8` struct field assignment | `go run ./cli/cmd/tetra check .../struct_u8_field_assignment_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../struct_u8_field_assignment_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving `h.byte = 300` stored/read the invalid value unchanged; extends BUG-023 | Validate assignment ranges for field targets |
| S019-006 | `Bool` struct field integer control | `go run ./cli/cmd/tetra check .../struct_bool_constructor_int_rejected_control.tetra` | Rejected with `type mismatch for field 'flag'` | Confirms this is limited to small-int compatibility |

### 2026-05-18 - Session 020 - Typed Error Small Integer Throw Edges

Planned focus:

- Check `throws UInt8`/`throws UInt16` as a typed service error-code boundary.
- Use `catch` literal patterns to prove which error value is delivered at
  runtime.
- Contrast with `throws Bool` to avoid confusing small-int compatibility with
  every typed-error payload.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S020-001 | Graphify/source navigation for typed-error throw validation | `mcp__graphify__.query_graph ... typed errors throw statement UInt8 UInt16 ...`; source reads in `compiler/tests/semantics/semantics_types_protocols_test.go`, `compiler/internal/semantics/semantics_checker.go`, and `compiler/internal/semantics/semantics_expressions.go` | Found `ThrowStmt` checks values with `typesCompatibleWithNullPtr(state.throwType, tname, s.Value)`, and scalar catch patterns can match integer literals with a default fallback | Probe runtime error-code delivery |
| S020-002 | `UInt8` throw max control | `go run ./cli/cmd/tetra check .../throw_u8_max_control.tetra`; `go run ./cli/cmd/tetra run .../throw_u8_max_control.tetra` | Passed `check`; `run` printed `exit status 42` for `throw 255` matched by `case 255` | Baseline valid |
| S020-003 | Out-of-range `UInt8` typed-error throw | `go run ./cli/cmd/tetra check .../throw_u8_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../throw_u8_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving `throw 300` from `throws UInt8` matched `case 300`; extends BUG-023 | Validate throw expression ranges |
| S020-004 | Out-of-range `UInt16` typed-error throw | `go run ./cli/cmd/tetra check .../throw_u16_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../throw_u16_out_of_range.tetra` | Passed `check`; `run` printed `exit status 42`, proving `throw 70000` from `throws UInt16` matched `case 70000`; extends BUG-023 | Validate throw expression ranges |
| S020-005 | `Bool` typed-error integer control | `go run ./cli/cmd/tetra check .../throw_bool_int_rejected_control.tetra` | Rejected with `throw type mismatch: expected 'bool', got 'i32'` | Confirms typed-error issue follows small-int compatibility |

### 2026-05-18 - Session 021 - Optional Small Integer Payload Edges

Planned focus:

- Check optional `UInt8?`/`UInt16?` payload creation and assignment as nullable
  service field boundaries.
- Compare direct numeric literals with already-typed `UInt8`/`UInt16` locals and
  exact `Int?` literals.
- Determine whether the bug is another invalid-value preservation path or a
  checker/lowering contract mismatch.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S021-001 | Graphify/source navigation for optional payload validation/lowering | `mcp__graphify__.query_graph ... optional some payload UInt8 UInt16 ...`; source reads in `compiler/tests/semantics/semantics_types_protocols_test.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, and `compiler/internal/lower/lower_core.go` | Found optional checker compatibility accepts `UInt8?`/`UInt16?` from `i32` via `typesCompatible(elem, actual)`, while `lowerExprAs` only wraps optional payloads when the inferred actual type exactly equals the optional element type | Probe check/run behavior |
| S021-002 | Direct `UInt8?` in-range literal payload | `go run ./cli/cmd/tetra check .../optional_u8_max_control.tetra`; `go run ./cli/cmd/tetra run .../optional_u8_max_control.tetra` | Passed `check`; `run` failed with `slot mismatch for 'maybe'`; confirmed BUG-028 even for valid `255` | Align checker/lowering or reject literal payload |
| S021-003 | Direct `UInt8?`/`UInt16?` out-of-range literal payloads | `go run ./cli/cmd/tetra check .../optional_u8_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../optional_u8_out_of_range.tetra`; repeated for `.../optional_u16_out_of_range.tetra` | Both passed `check`; both `run` commands failed with `slot mismatch for 'maybe'` before payload could be matched | Reject out-of-range optional payloads during check |
| S021-004 | `UInt8?` assignment from out-of-range literal | `go run ./cli/cmd/tetra check .../optional_u8_assignment_out_of_range.tetra`; `go run ./cli/cmd/tetra run .../optional_u8_assignment_out_of_range.tetra` | Passed `check`; `run` failed with `slot mismatch for assignment` | Align assignment lowering with checker or reject |
| S021-005 | Typed small-int optional payload controls | `go run ./cli/cmd/tetra check .../optional_u8_typed_payload_control.tetra`; `go run ./cli/cmd/tetra run .../optional_u8_typed_payload_control.tetra`; repeated for `.../optional_u16_typed_payload_control.tetra` and `.../optional_u8_typed_assignment_control.tetra` | All passed `check`; all `run` commands printed `exit status 42`, proving optional small-int wrapping works when the payload expression already has the exact element type | Use as nearby working behavior |
| S021-006 | Exact `Int?` and `Bool?` controls | `go run ./cli/cmd/tetra check .../optional_i32_literal_control.tetra`; `go run ./cli/cmd/tetra run .../optional_i32_literal_control.tetra`; `go run ./cli/cmd/tetra check .../optional_bool_int_rejected_control.tetra` | `Int? = 42` passed and returned `exit status 42`; `Bool? = 1` was rejected with `type mismatch: expected 'bool?', got 'i32'` | Confirms bug is small-int compatibility plus optional lowering |

### 2026-05-18 - Session 022 - Collection Loop and Nested Optional Mismatch Edges

Planned focus:

- First check whether collection `for` loops accept multi-slot fixed-array
  element types that lowering cannot load.
- Avoid recording known fixed-array zero-initialization failures as a new bug.
- If the collection path is already rejected or duplicates BUG-024, continue
  into nearby checker/lowering mismatches around optional lifting.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S022-001 | Graphify/source navigation for collection loop lowering | `mcp__graphify__.query_graph ... collectionElementType lowerIndexLoadKind ...`; `mcp__graphify__.get_neighbors ForRangeStmt`; `mcp__graphify__.shortest_path ForRangeStmt lowerIndexLoadKind` | Found `collectionElementType` accepts string/slice/array element types, while `lowerIndexLoadKind` only supports `i32`, `bool`, `u8`, `u16`, and single-slot structs | Probe whether unsupported array element types pass `check` |
| S022-002 | Collection-loop controls | `go run ./cli/cmd/tetra check .../string_for_control.tetra`; `go run ./cli/cmd/tetra check .../u8_slice_for_control.tetra` | Both passed `check`, confirming standard string and `[]u8` collection loops are accepted | Baselines only |
| S022-003 | Fixed array `String` collection loop hypothesis | `go run ./cli/cmd/tetra check .../array_string_for_field.tetra` | Rejected during `check` with `array element type 'str' is not supported`; no checker/lowering bug | Do not log as new |
| S022-004 | Fixed array `Int` collection loop duplicate check | `go run ./cli/cmd/tetra check .../array_i32_for_field_control.tetra` | Passed `check`, but this path depends on the already-known zeroed fixed-array storage bug if executed | Treat as BUG-024 territory |
| S022-005 | Nested optional source navigation | `mcp__graphify__.query_graph ... typesCompatible lowerExprAs ...`; source reads in `compiler/internal/semantics/semantics_core.go` and `compiler/internal/lower/lower_core.go` | Found `typesCompatible` recursively accepts optional payload lifting, but `lowerExprAs` only emits one wrapper layer when the actual type exactly matches the optional element type | Probe `Int??` direct literals |
| S022-006 | Nested optional controls | `go run ./cli/cmd/tetra check .../nested_optional_none_control.tetra`; `go run ./cli/cmd/tetra run .../nested_optional_none_control.tetra`; repeated for `.../nested_optional_inner_control.tetra` | Both passed `check`; both `run` commands printed `exit status 42`, proving `none` and already-typed `Int?` payloads lower correctly | Working nearby behavior |
| S022-007 | Direct nested optional literal payload | `go run ./cli/cmd/tetra check .../nested_optional_literal_payload.tetra`; `go run ./cli/cmd/tetra run .../nested_optional_literal_payload.tetra` | Passed `check`; `run` failed with `slot mismatch for 'nested'`; confirmed BUG-029 | Align recursive optional lifting or reject direct payload |
| S022-008 | Nested optional assignment and return payloads | `go run ./cli/cmd/tetra check .../nested_optional_assignment_literal_payload.tetra`; `go run ./cli/cmd/tetra run .../nested_optional_assignment_literal_payload.tetra`; repeated for `.../nested_optional_return_literal_payload.tetra` | Both passed `check`; assignment failed with `slot mismatch for assignment`; return failed with `return slot mismatch`; extends BUG-029 across assignment/return surfaces | Add regression coverage for all three contexts |

### 2026-05-18 - Session 023 - Slice Metadata Bounds Bypass

Planned focus:

- Check whether the same `ptr`/`len` protection added for fixed arrays also
  applies to slices.
- Prove whether native bounds checks rely on mutable slice metadata.
- Contrast mutable slices, immutable slice bindings, and fixed-array internals
  to avoid recording a false positive.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S023-001 | Graphify/source navigation for slice metadata assignment | `mcp__graphify__.query_graph ... slice ptr len field assignment ...`; `mcp__graphify__.get_neighbors rejectFixedArrayInternalAssignment()`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/internal/backend/x64core/x64core_core.go` | Found `TypeSlice` exposes `ptr`/`len`; `resolveAssignTarget` rejects `ptr`/`len` only for `TypeArray`; x64 index checks compare against the slice length slot and address via the pointer slot | Probe safe metadata mutation |
| S023-002 | Enlarge slice length and write out of allocation | `go run ./cli/cmd/tetra check .../slice_len_mutation_oob_write.tetra`; `go run ./cli/cmd/tetra run .../slice_len_mutation_oob_write.tetra` | Passed `check`; `run` printed `exit status 42`, proving `bytes.len = 64` lets `bytes[50]` pass on a one-byte slice | Confirmed BUG-030 |
| S023-003 | Pointer/length mismatch across slices | `go run ./cli/cmd/tetra check .../slice_ptr_len_mismatch_oob_write.tetra`; `go run ./cli/cmd/tetra run .../slice_ptr_len_mismatch_oob_write.tetra` | Passed `check`; `run` printed `exit status 42`, proving `wide.ptr = tiny.ptr` keeps the wide length and writes through the tiny allocation pointer | Include `ptr` in BUG-030 |
| S023-004 | Shrink length and block valid allocation index | `go run ./cli/cmd/tetra check .../slice_len_mutation_blocks_valid_index.tetra`; `go run ./cli/cmd/tetra run .../slice_len_mutation_blocks_valid_index.tetra` | Passed `check`; `run` printed `exit status 1` after `bytes.len = 0; bytes[0] = 42`, proving bounds checks trust mutated metadata | Supports root cause |
| S023-005 | Normal out-of-bounds control | `go run ./cli/cmd/tetra check .../slice_oob_without_len_mutation_control.tetra`; `go run ./cli/cmd/tetra run .../slice_oob_without_len_mutation_control.tetra` | Passed `check`; `run` printed `exit status 1`, proving index checks fire when metadata is not forged | Baseline |
| S023-006 | Immutable slice binding control | `go run ./cli/cmd/tetra check .../slice_len_immutable_control.tetra` | Rejected with `cannot assign to val 'bytes'` | Hole requires mutable slice binding |
| S023-007 | Fixed-array internals control | `go run ./cli/cmd/tetra check .../fixed_array_len_assignment_rejected_control.tetra` | Rejected with `cannot assign to fixed-array internals ('ptr'/'len')` | Apply same protection to slices |

### 2026-05-18 - Session 024 - String Metadata Bounds Bypass

Planned focus:

- Check whether `String` has the same public `ptr`/`len` metadata surface as
  slices.
- Prove whether string indexing and collection iteration trust mutable string
  metadata.
- Keep this distinct from BUG-030 by using only `String` literals and string
  operations, not `[]u8` slices.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S024-001 | Graphify/source navigation for string metadata assignment | `mcp__graphify__.query_graph ... String TypeStr ptr len ...`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/backend/x64core/x64core_core.go` | Found `str` is `makeSliceTypeInfo("str", "u8")` with `Kind = TypeStr`; field assignment rejection only protects `TypeArray`; string index lowering uses `IRIndexLoadU8` with the view's pointer/length slots | Probe safe string metadata mutation |
| S024-002 | Enlarge string length and read past literal | `go run ./cli/cmd/tetra check .../string_len_mutation_oob_read.tetra`; `go run ./cli/cmd/tetra run .../string_len_mutation_oob_read.tetra` | Passed `check`; `run` printed `exit status 42`, proving `text.len = 2` lets `text[1]` pass on `"*"` | Confirmed BUG-031 |
| S024-003 | Pointer/length mismatch across strings | `go run ./cli/cmd/tetra check .../string_ptr_len_mismatch_oob_read.tetra`; `go run ./cli/cmd/tetra run .../string_ptr_len_mismatch_oob_read.tetra` | Passed `check`; `run` printed `exit status 42`, proving `wide.ptr = tiny.ptr` keeps the wide length and reads through the tiny string pointer | Include `ptr` in BUG-031 |
| S024-004 | String collection iteration trusts forged length | `go run ./cli/cmd/tetra check .../string_for_len_mutation_count.tetra`; `go run ./cli/cmd/tetra run .../string_for_len_mutation_count.tetra` | Passed `check`; `run` printed `exit status 42` after counting two `for ch in text` iterations over a one-byte literal | Cover collection loop path |
| S024-005 | Shrink string length and block valid read | `go run ./cli/cmd/tetra check .../string_len_mutation_blocks_valid_read.tetra`; `go run ./cli/cmd/tetra run .../string_len_mutation_blocks_valid_read.tetra` | Passed `check`; `run` printed `exit status 1` after `text.len = 0; text[0]`, proving string bounds checks trust mutated metadata | Supports root cause |
| S024-006 | Normal string out-of-bounds control | `go run ./cli/cmd/tetra check .../string_oob_without_len_mutation_control.tetra`; `go run ./cli/cmd/tetra run .../string_oob_without_len_mutation_control.tetra` | Passed `check`; `run` printed `exit status 1`, proving out-of-bounds string indexing fails when metadata is not forged | Baseline |
| S024-007 | Immutable string binding control | `go run ./cli/cmd/tetra check .../string_len_immutable_control.tetra` | Rejected with `cannot assign to val 'text'` | Hole requires mutable `String` binding |

### 2026-05-18 - Session 027 - Dotted Built-in Call-Style Constructors

Planned focus:

- Check whether BUG-015 is limited to `task.i32(...)` or affects other
  built-in dotted structs.
- Use brace-style construction as a nearby control to separate constructor
  lowering from type visibility or field layout problems.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S027-001 | Graphify/source navigation for call-style struct constructors | `mcp__graphify__.query_graph ... call-style constructors for dotted builtin structs ...`; `mcp__graphify__.get_neighbors checkCallExprWithEffects()`; `mcp__graphify__.shortest_path call-style constructor lowerExpr`; source reads in `compiler/internal/semantics/semantics_expressions.go` and `compiler/internal/lower/lower_core.go` | Found `checkStructConstructorCallWithEffects` accepts labeled calls that resolve to struct types, while lowering/build still has paths that can demand function signatures for dotted type names | Probe concrete built-in structs |
| S027-002 | Call-style `actor.msg(...)` constructor | `go run ./cli/cmd/tetra check .../actor_msg_call_constructor.tetra`; `go run ./cli/cmd/tetra run .../actor_msg_call_constructor.tetra` | Passed `check`; `run` failed with `missing signature for 'actor.msg'`; extends BUG-015 | Align call-style constructor lowering/dependency handling |
| S027-003 | Call-style `actor.recv_result_i32(...)` constructor | `go run ./cli/cmd/tetra check .../actor_recv_result_call_constructor.tetra`; `go run ./cli/cmd/tetra run .../actor_recv_result_call_constructor.tetra` | Passed `check`; `run` failed with `missing signature for 'actor.recv_result_i32'`; extends BUG-015 | Align call-style constructor lowering/dependency handling |
| S027-004 | Call-style `task.result_i32(...)` constructor | `go run ./cli/cmd/tetra check .../task_result_call_constructor.tetra`; `go run ./cli/cmd/tetra run .../task_result_call_constructor.tetra` | Passed `check`; `run` failed with `missing signature for 'task.result_i32'`; extends BUG-015 | Align call-style constructor lowering/dependency handling |
| S027-005 | Brace-style `actor.msg` constructor control | `go run ./cli/cmd/tetra check .../actor_msg_brace_constructor_control.tetra`; `go run ./cli/cmd/tetra run .../actor_msg_brace_constructor_control.tetra` | Passed `check`; `run` printed `exit status 42` for the expected field sum | Confirms brace-style construction remains a working baseline |

### 2026-05-18 - Session 028 - Typed Task Error Payload Sendability

Planned focus:

- Compare typed actor payload sendability with typed task error payload
  sendability.
- Probe whether `String`/`ptr` payloads can cross the task result channel even
  though they are not accepted as typed actor messages.
- Keep capability-token construction probes separate unless a concrete
  capability transfer path is reproduced.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S028-001 | Graphify/source navigation for typed task error payload validation | `mcp__graphify__.query_graph ... typed task error enum payload sendable validation ...`; `mcp__graphify__.get_neighbors validateTypedActorMessageType()`; `mcp__graphify__.get_neighbors funcSigActorTaskTransferSafe()`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_expressions.go`, `docs/spec/flow/v1_scope.md`, and `docs/spec/core/current_supported_surface.md` | Found task worker sendability checks inspect params/return but not `ThrowsType`; typed task error validation only requires enum; typed actor payload validation rejects string/pointer/capability handles | Probe task error payloads |
| S028-002 | Typed task error carries `String` payload across join | `go run ./cli/cmd/tetra check .../typed_task_string_error_payload.tetra`; `go run ./cli/cmd/tetra run .../typed_task_string_error_payload.tetra` | Passed `check`; `run` printed `exit status 42` after catch arm read `text[0]` from the task-thrown `String` payload; confirmed BUG-032 | Validate typed task error payload sendability |
| S028-003 | Typed task error carries `ptr` payload across join | `go run ./cli/cmd/tetra check .../typed_task_ptr_error_payload_null.tetra`; `go run ./cli/cmd/tetra run .../typed_task_ptr_error_payload_null.tetra` | Passed `check`; `run` printed `exit status 42` through the `TaskErr.bad(ptr)` catch arm; extends BUG-032 | Reject pointer payloads in typed task errors |
| S028-004 | Typed actor `String` payload control | `go run ./cli/cmd/tetra check .../typed_actor_string_payload_rejected_control.tetra` | Rejected with `typed actor message payload must be value-only, got string view 'str'` | Use analogous value-only policy for typed task errors |
| S028-005 | Task worker `String` return control | `go run ./cli/cmd/tetra check .../typed_task_string_return_control.tetra` | Rejected with `task_spawn_i32 target must have shape func worker() -> i32` | Confirms the worker result path does not accept a `String` return |

### 2026-05-18 - Session 029 - Typed Task Error Resource Payload Edges

Planned focus:

- Extend BUG-032 probes from raw views (`String`/`ptr`) into handle/resource
  payloads.
- Distinguish allowed or provenance-checked handle transfer from genuinely
  missing `ThrowsType` payload validation.
- Avoid logging a new bug unless the behavior has a distinct root cause.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S029-001 | Graphify/source navigation for resource payloads in typed task errors | `mcp__graphify__.query_graph ... typed task error enum payload resource handles ...`; `mcp__graphify__.get_neighbors typeContainsResourceHandle()`; `mcp__graphify__.get_neighbors funcSigActorTaskTransferUnsafeReason()`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_memory_resources.go`, and `compiler/internal/semantics/semantics_expressions.go` | Found resource tracking knows `actor`/`island`/`task.i32`, but typed-task spawn still validates only params/return and the error type's enum-ness | Probe handles and non-sendable error cases |
| S029-002 | `task.i32` payload matched through typed task error | `go run ./cli/cmd/tetra check .../typed_task_error_task_handle_payload_match_only.tetra`; `go run ./cli/cmd/tetra run .../typed_task_error_task_handle_payload_match_only.tetra` | Passed `check`; `run` printed `exit status 42`, proving `TaskErr.moved(task.i32)` can be matched after join; using it as a join handle in the stronger variant was rejected with `ambiguous resource provenance` | Keep as handle-transfer/provenance probe under BUG-032 root |
| S029-003 | `actor` payload matched through typed task error | `go run ./cli/cmd/tetra check .../typed_task_error_actor_payload.tetra`; `go run ./cli/cmd/tetra run .../typed_task_error_actor_payload.tetra` | Passed `check`; `run` printed `exit status 42`, proving `TaskErr.who(actor)` crosses the typed task error channel | Clarify whether actor handles are intended sendable error payloads |
| S029-004 | `island` payload accepted in typed task error type | `go run ./cli/cmd/tetra check .../typed_task_error_island_payload_type_only.tetra`; `go run ./cli/cmd/tetra run .../typed_task_error_island_payload_type_only.tetra` | Passed `check`; `run` printed `exit status 42` on the success path, proving `task_spawn_i32_typed<TaskErr>` accepts `E` with an island payload case | Extend BUG-032 validation gap |
| S029-005 | `cap.mem` payload accepted in typed task error type | `go run ./cli/cmd/tetra check .../typed_task_error_cap_payload_type_only.tetra`; `go run ./cli/cmd/tetra run .../typed_task_error_cap_payload_type_only.tetra` | Passed `check`; `run` printed `exit status 42` on the success path, proving `task_spawn_i32_typed<TaskErr>` accepts `E` with a capability payload case | Extend BUG-032 validation gap |

### 2026-05-18 - Session 031 - Typed Task Group Closed-Resource Bypass

Planned focus:

- Compare ordinary task-group spawn with typed task-group spawn after a
  `task_group_close`.
- Verify whether the typed task specialized checker enforces the same
  resource-finalization rule as the generic builtin path.
- Record only if a closed `task.group` reaches `check` on the typed path.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S031-001 | Graphify/source navigation for typed task group resource checks | `mcp__graphify__.query_graph ... task_join_until_i32 task_select2_i32 typed task join ...`; `mcp__graphify__.get_neighbors checkTypedTaskBuiltin()`; source reads in `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/tests/runtime/resource_finalization_test.go` | Found the generic call path invokes `checkResourceCallArg`, while `checkTypedTaskBuiltin` validates the group argument with only `checkExprWithEffects` plus a `task.group` type check | Probe direct closed-group spawn |
| S031-002 | Untyped closed-group spawn control | `go run ./cli/cmd/tetra check .../untyped_group_spawn_after_close_control.tetra` | Rejected with `cannot use closed resource 'group'` at `core.task_spawn_group_i32(group, "worker")` | Confirms baseline resource finalization rule |
| S031-003 | Typed closed-group spawn repro | `go run ./cli/cmd/tetra check .../typed_group_spawn_after_close_repro.tetra`; `go run ./cli/cmd/tetra run .../typed_group_spawn_after_close_repro.tetra` | Passed `check`; `run` printed `exit status 5` via `GroupErr.stopped`, proving the closed group reaches the typed task group runtime path | Confirmed BUG-033 |

### 2026-05-18 - Session 032 - Typed Task Group Alias and Maybe-Closed Bypass

Planned focus:

- Extend BUG-033 beyond direct closed local variables into control-flow merge
  and aggregate alias forms.
- Compare each typed repro with an untyped group-spawn control.
- Keep the finding under BUG-033 unless a distinct root cause appears.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S032-001 | Graphify/source navigation for alias/maybe-closed resource checks | `mcp__graphify__.query_graph ... typed task group spawn closed task.group alias maybe closed ...`; source reads in `graphify-out/GRAPH_REPORT.md` and `compiler/tests/runtime/resource_finalization_test.go` | Found existing untyped coverage for maybe-closed groups and optional/enum payload aliases after close | Build typed equivalents |
| S032-002 | Maybe-closed group control and typed repro | `go run ./cli/cmd/tetra check .../untyped_group_spawn_after_maybe_close_control.tetra`; `go run ./cli/cmd/tetra check .../typed_group_spawn_after_maybe_close_repro.tetra`; `go run ./cli/cmd/tetra run .../typed_group_spawn_after_maybe_close_repro.tetra` | Untyped rejected with `resource may have been closed after control-flow merge`; typed passed `check` and `run` printed `exit status 5` | Extends BUG-033 to merge state |
| S032-003 | Optional alias control and typed repro | `go run ./cli/cmd/tetra check .../untyped_group_spawn_optional_alias_after_close_control.tetra`; `go run ./cli/cmd/tetra check .../typed_group_spawn_optional_alias_after_close_repro.tetra`; `go run ./cli/cmd/tetra run .../typed_group_spawn_optional_alias_after_close_repro.tetra` | Untyped rejected with `cannot use closed resource 'other'`; typed passed `check` and `run` printed `exit status 5` | Extends BUG-033 to optional payload aliases |
| S032-004 | Enum payload alias control and typed repro | `go run ./cli/cmd/tetra check .../untyped_group_spawn_enum_alias_after_close_control.tetra`; `go run ./cli/cmd/tetra check .../typed_group_spawn_enum_alias_after_close_repro.tetra`; `go run ./cli/cmd/tetra run .../typed_group_spawn_enum_alias_after_close_repro.tetra` | Untyped rejected with `cannot use closed resource 'other'`; typed passed `check` and `run` printed `exit status 5` | Extends BUG-033 to enum payload aliases |

### 2026-05-18 - Session 036 - Secret Taint Through Throw Payloads

Planned focus:

- Check whether privacy taint analysis applies to `ThrowStmt` payloads.
- Use an `@export` return control because exported raw secret returns are
  already explicitly rejected.
- Record only if the same unsealed value can cross through an error enum.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S036-001 | Graphify/source navigation for secret-tainted throw policy | `mcp__graphify__.query_graph ... secret tainted value throw enum payload ...`; source reads in `compiler/internal/semantics/semantics_checker.go` and `compiler/tests/safety/effects/effects_test.go` | Found `ReturnStmt` calls `exprSecretTainted`, while `ThrowStmt` does not; existing tests cover exported secret-tainted returns but not throw payloads | Build export return/throw pair |
| S036-002 | Exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline privacy boundary |
| S036-003 | Exported throw payload repro | `go run ./cli/cmd/tetra check .../export_secret_throw_payload_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_throw_payload_repro.tetra` | Passed `check`; `run` printed `exit status 42` after catching `LeakErr.raw(value)` | Confirmed BUG-034 |

### 2026-05-18 - Session 040 - Secret Taint Through Printable Buffers

Planned focus:

- Check whether `PrintStmt` treats printable byte buffers as an external
  privacy sink.
- Distinguish missing output-sink checks from local field/index taint
  propagation failures.
- Record only if an unsealed secret reaches stdout after `check` succeeds.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S040-001 | Graphify/source navigation for `PrintStmt` secret-taint policy | `mcp__graphify__.query_graph ... PrintStmt secret tainted ...`; `mcp__graphify__.get_neighbors PrintStmt`; `mcp__graphify__.shortest_path PrintStmt exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go` and `compiler/tests/safety/effects/effects_test.go` | Found `PrintStmt` only requires `io` and printable type; `ReturnStmt` checks `exprSecretTainted`; assignment code can mark local containers tainted after secret writes | Build print and local-container controls |
| S040-002 | Exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline privacy boundary |
| S040-003 | Plain printable byte buffer control | `go run ./cli/cmd/tetra check .../print_plain_u8_buffer_control.tetra`; `go run ./cli/cmd/tetra run .../print_plain_u8_buffer_control.tetra` | Passed `check`; `run` printed `*`, confirming the `[]UInt8` print syntax/runtime path | Baseline printable sink |
| S040-004 | Local field/index assignment taint controls | `go run ./cli/cmd/tetra check .../export_secret_local_field_assignment_repro.tetra`; `go run ./cli/cmd/tetra check .../export_secret_local_index_assignment_repro.tetra` | Both rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Container taint is preserved when read back |
| S040-005 | Secret-tainted printable buffer repro | `go run ./cli/cmd/tetra check .../secret_print_u8_buffer_assignment_probe.tetra`; `go run ./cli/cmd/tetra run .../secret_print_u8_buffer_assignment_probe.tetra` | Passed `check`; `run` printed `*` from secret value 42 written into `bytes[0]` | Confirmed BUG-035 |

### 2026-05-18 - Session 041 - Secret Taint Through Control Flow

Planned focus:

- Check whether secret-tainted conditions propagate taint to branch outputs.
- Compare against direct exported secret return and a public branch control.
- Include a global-assignment variant to test outward effects beyond
  `ReturnStmt`.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S041-001 | Graphify/source navigation for secret-tainted control flow | `mcp__graphify__.query_graph ... IfStmt secret tainted ...`; `mcp__graphify__.get_neighbors checkStmts()`; `mcp__graphify__.shortest_path IfStmt exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/tests/safety/effects/effects_test.go`, and privacy docs | Found `IfStmt` checks condition type but does not call `exprSecretTainted` for the condition; `ReturnStmt` and explicit global writes do check expression taint | Build implicit-flow controls |
| S041-002 | Exported raw return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S041-003 | Public branch control | `go run ./cli/cmd/tetra check .../export_public_branch_control.tetra`; `go run ./cli/cmd/tetra run .../export_public_branch_control.tetra` | Passed `check`; `run` printed `exit status 42` | Baseline ordinary branch behavior |
| S041-004 | Secret condition exported return, true branch | `go run ./cli/cmd/tetra check .../export_secret_if_condition_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_if_condition_return_repro.tetra` | Passed `check`; `run` printed `exit status 42` for `secret_seal_i32(1, token)` | Confirmed implicit branch leak |
| S041-005 | Secret condition exported return, false branch | `go run ./cli/cmd/tetra check .../export_secret_if_condition_return_false_branch_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_if_condition_return_false_branch_repro.tetra` | Passed `check`; `run` printed `exit status 7` for `secret_seal_i32(0, token)` | Confirms output depends on secret |
| S041-006 | Secret condition global assignment | `go run ./cli/cmd/tetra check .../global_secret_if_condition_assignment_repro.tetra`; `go run ./cli/cmd/tetra run .../global_secret_if_condition_assignment_repro.tetra` | Passed `check`; `run` printed `exit status 42` after storing a public constant selected by the secret condition | Extends BUG-036 beyond returns |

### 2026-05-18 - Session 042 - Secret Taint Through Match and Loop Control Flow

Planned focus:

- Extend BUG-036 beyond `IfStmt` into `match` expression, `match` statement,
  and `while` condition control flow.
- Avoid a new bug number unless a distinct checker path/root cause appears.
- Keep outputs as public constants so the leak is purely control-dependent.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S042-001 | Graphify/source navigation for `match`/loop secret control flow | `mcp__graphify__.query_graph ... MatchStmt MatchExpr WhileStmt secret ...`; `mcp__graphify__.get_neighbors MatchStmt`; `mcp__graphify__.shortest_path MatchStmt exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go` and `docs/spec/flow/flow_syntax_v1.md` | Found `MatchExpr` result taint ignores the scrutinee control dependency; `MatchStmt` records scrutinee taint for bindings but not case outputs; `WhileStmt` follows the same condition-only type check shape as `IfStmt` | Build extension probes |
| S042-002 | Secret `match` expression exported return | `go run ./cli/cmd/tetra check .../export_secret_match_expr_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_match_expr_return_repro.tetra` | Passed `check`; `run` printed `exit status 42` for `secret_seal_i32(1, token)` | Extends BUG-036 to `MatchExpr` |
| S042-003 | Secret `match` statement exported return | `go run ./cli/cmd/tetra check .../export_secret_match_stmt_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_match_stmt_return_repro.tetra` | Passed `check`; `run` printed `exit status 42` for `secret_seal_i32(1, token)` | Extends BUG-036 to `MatchStmt` |
| S042-004 | Secret `while` condition exported return, true branch | `go run ./cli/cmd/tetra check .../export_secret_while_condition_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_while_condition_return_repro.tetra` | Passed `check`; `run` printed `exit status 42` for `secret_seal_i32(1, token)` | Extends BUG-036 to loop conditions |
| S042-005 | Secret `while` condition exported return, false branch | `go run ./cli/cmd/tetra check .../export_secret_while_condition_false_branch_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_while_condition_false_branch_repro.tetra` | Passed `check`; `run` printed `exit status 7` for `secret_seal_i32(0, token)` | Confirms loop output depends on secret |

### 2026-05-18 - Session 043 - Secret Taint Through Actor Mailboxes

Planned focus:

- Check whether effectful `core.send` actor mailbox calls are treated as
  privacy sinks for secret-tainted scalar values.
- Compare against a direct exported secret return and a public mailbox
  round-trip control.
- Use two secret payloads to prove the received exported return carries the
  mailbox value.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S043-001 | Graphify/source navigation for actor mailbox taint | `mcp__graphify__.query_graph ... core.send core.recv secret taint ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/semantics/semantics_core.go`, and `docs/spec/runtime/actors.md` | Found `ExprStmt` ignores a tainted call result; `exprSecretTainted` marks tainted `core.*` calls as tainted rather than rejecting side-effect sinks; `core.recv` return provenance is not tied to previous sends | Build mailbox laundering probes |
| S043-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S043-003 | Public actor self-send control | `go run ./cli/cmd/tetra check .../actor_public_self_send_control.tetra`; `go run ./cli/cmd/tetra run .../actor_public_self_send_control.tetra` | Passed `check`; `run` printed `exit status 42` after `core.send(core.self(), 42)` and `core.recv()` | Baseline actor mailbox round trip |
| S043-004 | Secret actor mailbox laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_actor_mailbox_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_actor_mailbox_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after sending unsealed `raw` through the mailbox and returning `core.recv()` | Confirmed BUG-037 |
| S043-005 | Secret actor mailbox laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_actor_mailbox_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_actor_mailbox_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms returned value tracks secret payload |

### 2026-05-18 - Session 044 - Secret Taint Through Typed and Tagged Actor Mailboxes

Planned focus:

- Extend BUG-037 from plain `core.send`/`core.recv` to typed enum actor
  messages and raw tagged actor messages.
- Avoid duplicating BUG-017 raw-to-typed spoofing; this session is about
  privacy taint provenance, not tag validation.
- Use two sealed values (`42` and `7`) for each mailbox path.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S044-001 | Graphify/source navigation for typed/tagged actor mailbox taint | `mcp__graphify__.query_graph ... send_typed recv_typed send_msg recv_msg secret taint ...`; source reads in `compiler/internal/semantics/semantics_expressions.go`, `compiler/compiler_suite_test.go`, and `docs/spec/runtime/actors.md` | Found `send_typed` validates enum/value-only payload shape and transfer payloads, but not secret-tainted payload values; `recv_typed`/`recv_msg` return mailbox data without taint provenance | Build typed/tagged mailbox probes |
| S044-002 | Typed actor mailbox laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_typed_actor_mailbox_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_typed_actor_mailbox_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after `LeakMsg.raw(raw)` round-tripped through `send_typed`/`recv_typed` | Extends BUG-037 to typed actor payloads |
| S044-003 | Typed actor mailbox laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_typed_actor_mailbox_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_typed_actor_mailbox_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms typed payload tracks secret |
| S044-004 | Tagged actor mailbox laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_tagged_actor_mailbox_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_tagged_actor_mailbox_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after `core.send_msg(self, raw, 99)` and `core.recv_msg().value` | Extends BUG-037 to tagged actor messages |
| S044-005 | Tagged actor mailbox laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_tagged_actor_mailbox_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_tagged_actor_mailbox_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms tagged payload tracks secret |

### 2026-05-18 - Session 045 - Secret Taint Through Raw Memory

Planned focus:

- Check whether raw memory load/store preserves privacy taint provenance.
- Keep the repro inside an explicit `unsafe:` block with a real `cap.mem`
  token, so the finding is about privacy declassification rather than missing
  unsafe gating.
- Compare against a direct exported return control and a public raw-memory
  round-trip control.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S045-001 | Graphify/source navigation for raw-memory taint | `mcp__graphify__.query_graph ... core.store_i32 core.load_i32 secret taint ...`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, `docs/spec/runtime/unsafe.md`, and `docs/spec/runtime/effects_capabilities_privacy_v1.md` | Found raw memory is gated by `unsafe`/`cap.mem`, but load/store calls do not model privacy taint provenance across memory cells | Build memory laundering probes |
| S045-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S045-003 | Public raw-memory round trip | `go run ./cli/cmd/tetra check .../public_raw_memory_roundtrip_control.tetra`; `go run ./cli/cmd/tetra run .../public_raw_memory_roundtrip_control.tetra` | Passed `check`; `run` printed `exit status 42` after `core.store_i32`/`core.load_i32` | Baseline unsafe memory path |
| S045-004 | Secret raw-memory laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_raw_memory_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_raw_memory_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after storing unsealed `raw` and returning the loaded value | Confirmed BUG-038 |
| S045-005 | Secret raw-memory laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_raw_memory_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_raw_memory_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms loaded value tracks secret payload |

### 2026-05-18 - Session 046 - Secret Taint Through Runtime Logical Time

Planned focus:

- Check whether secret-tainted runtime delays can affect public time reads.
- Contrast with a direct `core.deadline_ms(raw)` return control, because direct
  tainted `core.*` return values should already be rejected.
- Use two sealed values (`42` and `7`) to prove the logical clock output tracks
  the secret.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S046-001 | Graphify/source navigation for runtime timing taint | `mcp__graphify__.query_graph ... sleep_ms time_now_ms secret taint ...`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, and `docs/spec/runtime/runtime_abi.md` | Found `sleep_ms` mutates deterministic logical runtime time, while privacy taint tracks call return values but not runtime temporal state | Build timing side-channel probes |
| S046-002 | Public sleep/time control | `go run ./cli/cmd/tetra check .../public_sleep_time_control.tetra`; `go run ./cli/cmd/tetra run .../public_sleep_time_control.tetra` | Passed `check`; `run` printed `exit status 42` after `core.sleep_ms(42)` then `core.time_now_ms()` | Baseline runtime clock behavior |
| S046-003 | Direct tainted deadline return control | `go run ./cli/cmd/tetra check .../export_secret_deadline_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Direct tainted runtime value is caught |
| S046-004 | Secret sleep/time laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_sleep_time_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_sleep_time_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after sleeping for unsealed `raw` and returning `time_now_ms()` | Confirmed BUG-039 |
| S046-005 | Secret sleep/time laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_sleep_time_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_sleep_time_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms time output tracks secret payload |

### 2026-05-18 - Session 047 - Secret Taint Through Observable MMIO

Planned focus:

- Check whether MMIO write/read preserves privacy taint provenance.
- Keep the repro inside an explicit `unsafe:` block with a real `cap.io`
  token, so the finding is about privacy declassification rather than missing
  unsafe or capability gating.
- Distinguish from raw memory by relying on the documented MMIO observable
  operation contract, even though the current backend lowers MMIO to memory.
- Compare against a direct exported return control and a public MMIO
  round-trip control.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S047-001 | Graphify/source navigation for MMIO privacy taint | `mcp__graphify__.query_graph ... core.mmio_write_i32 mmio_read_i32 secret taint ...`; `mcp__graphify__.get_neighbors core.mmio_write_i32`; `mcp__graphify__.shortest_path exprSecretTainted core.mmio_write_i32`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `docs/spec/runtime/capabilities.md`, and `examples/memory/raw/mmio_smoke.tetra` | Graphify had no direct `core.mmio_write_i32` node, but source/docs show MMIO is `unsafe`/`cap.io` gated and documented as observable; privacy taint tracks call return values, not MMIO locations or sinks | Build MMIO laundering probes |
| S047-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S047-003 | Public MMIO round trip | `go run ./cli/cmd/tetra check .../public_mmio_roundtrip_control.tetra`; `go run ./cli/cmd/tetra run .../public_mmio_roundtrip_control.tetra` | Passed `check`; `run` printed `exit status 42` after `core.mmio_write_i32`/`core.mmio_read_i32` | Baseline unsafe MMIO path |
| S047-004 | Secret MMIO laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_mmio_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_mmio_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after writing unsealed `raw` through MMIO and returning the read value | Confirmed BUG-040 |
| S047-005 | Secret MMIO laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_mmio_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_mmio_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms MMIO read output tracks secret payload |

### 2026-05-18 - Session 048 - Task Result Privacy Boundary Control

Planned focus:

- Check whether a privacy-tainted worker return can be laundered through
  `core.task_spawn_i32` and `core.task_join_i32`.
- Compare against a direct exported call to the same worker, which should be
  rejected if `funcReturnSecretTaint` is propagated.
- Treat a privacy-effect worker spawn rejection as a valid no-bug control.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S048-001 | Graphify/source navigation for task result taint | `mcp__graphify__.query_graph ... task_spawn_i32 task_join_i32 task.result_i32 secret taint ...`; `mcp__graphify__.get_neighbors core.task_spawn_i32`; `mcp__graphify__.shortest_path task.result_i32 exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, and `examples/tasks/task_join_wait_smoke.tetra` | Graphify had no direct task builtin nodes; source shows normal calls propagate `funcReturnSecretTaint`, while task spawn has separate boundary validation | Build direct-call and task-join probes |
| S048-002 | Public task join control | `go run ./cli/cmd/tetra check .../public_task_join_control.tetra`; `go run ./cli/cmd/tetra run .../public_task_join_control.tetra` | Passed `check`; `run` printed `exit status 42` | Baseline task join path |
| S048-003 | Direct exported worker-return control | `go run ./cli/cmd/tetra check .../export_secret_direct_worker_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Confirms direct function-return taint propagation |
| S048-004 | Secret-tainted task worker return, value 42 | `go run ./cli/cmd/tetra check .../export_secret_task_join_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_task_join_launder_repro.tetra` | Both rejected at `core.task_spawn_i32("worker")` with `target 'worker' uses effect 'privacy' and cannot cross task boundary` | No new bug in this path |
| S048-005 | Secret-tainted task worker return, value 7 | `go run ./cli/cmd/tetra check .../export_secret_task_join_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_task_join_launder_false_value_repro.tetra` | Both rejected with the same privacy-effect task boundary diagnostic | No new bug in this path |

### 2026-05-18 - Session 049 - Secret Taint Through Closure Captures

Planned focus:

- Check whether function-typed local calls preserve privacy taint from captured
  locals.
- Use a direct exported raw return as the privacy baseline and a public
  closure-capture program as the callable baseline.
- Use two sealed values (`42` and `7`) to prove the callable result tracks the
  secret payload.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S049-001 | Graphify/source navigation for closure capture taint | `mcp__graphify__.query_graph ... closure captures function-typed returns exprSecretTainted secret taint ...`; `mcp__graphify__.get_neighbors ClosureExpr`; `mcp__graphify__.shortest_path ClosureExpr exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/tests/semantics/semantics_callables_closures_test.go`, and `compiler/internal/lower/lower_suite_test.go` | Found closure/callable capture metadata is rich for ownership/escape, but `exprSecretTainted` has no closure-capture case and zero-arg function-typed local calls have no tainted argument to propagate | Build closure-capture laundering probes |
| S049-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S049-003 | Public closure capture control | `go run ./cli/cmd/tetra check .../public_closure_capture_control.tetra`; `go run ./cli/cmd/tetra run .../public_closure_capture_control.tetra` | Passed `check`; `run` printed `exit status 42` after a local immutable `Int` capture | Baseline function-typed local call path |
| S049-004 | Secret closure-capture laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_closure_capture_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_closure_capture_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` after `raw` was captured by `fn() -> Int` and returned via `f()` | Confirmed BUG-041 |
| S049-005 | Secret closure-capture laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_closure_capture_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_closure_capture_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms closure call output tracks secret payload |

### 2026-05-18 - Session 050 - Catch Payload Privacy Boundary Control

Planned focus:

- Check whether a non-exported helper can throw a secret-tainted enum payload,
  then an exported function can catch and return the payload clean.
- Compare against the known direct exported raw return rejection and a public
  throw/catch payload control.
- Treat rejection at the exported catch return as a no-bug result for this
  exact path, distinct from BUG-034's exported thrown payload escaping to an
  external caller.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S050-001 | Graphify/source navigation for catch payload taint | `mcp__graphify__.query_graph ... ThrowStmt CatchExpr catch bindings enum payloads exprSecretTainted secret taint ...`; `mcp__graphify__.get_neighbors CatchExpr`; `mcp__graphify__.shortest_path CatchExpr exprSecretTainted`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, `docs/spec/flow/flow_syntax_v1.md`, and Session 036 repros | Found `checkCatchExpr` binds ownership/region/resource payload locals, and `exprSecretTainted(CatchExpr)` rechecks the catch call/case values; build repro to see whether exported return catches the thrown payload | Build local catch probes |
| S050-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S050-003 | Public catch payload control | `go run ./cli/cmd/tetra check .../public_catch_payload_control.tetra`; `go run ./cli/cmd/tetra run .../public_catch_payload_control.tetra` | Passed `check`; `run` printed `exit status 42` after `LeakErr.raw(42)` was caught and returned | Baseline catch payload path |
| S050-004 | Secret catch payload laundering attempt, value 42 | `go run ./cli/cmd/tetra check .../export_secret_catch_payload_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_catch_payload_launder_repro.tetra` | Both rejected at the exported `return catch helper(token, value):` with `secret-tainted value cannot be returned from @export function 'leak'` | No new bug in this path |
| S050-005 | Secret catch payload laundering attempt, value 5 | `go run ./cli/cmd/tetra check .../export_secret_catch_payload_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_catch_payload_launder_false_value_repro.tetra` | Both rejected with the same exported-return privacy diagnostic | No new bug in this path |

### 2026-05-18 - Session 051 - Capability Token Forge Controls

Planned focus:

- Check whether opaque `cap.io` and `cap.mem` tokens can be manufactured with
  brace literals instead of being obtained through `core.cap_io()` /
  `core.cap_mem()` inside `unsafe`.
- Compare against the documented unsafe-only capability acquisition control.
- Treat `cap.io{}` / `cap.mem{}` rejection as a no-bug result for this exact
  forge syntax.

Evidence to add as probes run:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S051-001 | Graphify/source navigation for opaque capability tokens | `mcp__graphify__.query_graph ... cap.io cap.mem opaque capability tokens core.cap_io core.cap_mem fs_exists load_i32 store_i32 ...`; `mcp__graphify__.get_neighbors cap.io`; `mcp__graphify__.shortest_path cap.io StructLitExpr`; source reads in `compiler/internal/semantics/semantics_core.go`, `docs/spec/runtime/effects_capabilities_privacy_v1.md`, `docs/spec/runtime/capabilities.md`, and `compiler/internal/actorsrt/actorsrt_core.go` | Docs say `cap.io`/`cap.mem` are opaque tokens only acquired inside `unsafe`; source marks them as `TypeCap` and backend filesystem code does not inspect the token value, so typechecker opacity is the relevant barrier | Build capability forge probes |
| S051-002 | `core.cap_io()` outside `unsafe` control | `go run ./cli/cmd/tetra check .../cap_io_outside_unsafe_rejected_control.tetra` | Rejected with `'core.cap_io' is only allowed in unsafe blocks` | Baseline unsafe-only acquisition check |
| S051-003 | Valid `core.cap_io()` filesystem control | `go run ./cli/cmd/tetra check .../valid_cap_io_fs_control.tetra`; `go run ./cli/cmd/tetra run .../valid_cap_io_fs_control.tetra` | Passed `check`; `run` printed `exit status 42` when `README.md` existed | Baseline legitimate capability path |
| S051-004 | `cap.io{}` filesystem forge attempt | `go run ./cli/cmd/tetra check .../cap_io_brace_forge_fs_repro.tetra` | Rejected with `'cap.io' is not a struct` | No new bug for brace-literal `cap.io` forge |
| S051-005 | `cap.mem{}` raw-memory forge attempt | `go run ./cli/cmd/tetra check .../cap_mem_brace_forge_raw_memory_repro.tetra` | Rejected with `'cap.mem' is not a struct` | No new bug for brace-literal `cap.mem` forge |

### 2026-05-18 - Session 052 - Match Expression Control-Flow Confirmation

Planned focus:

- Re-check secret-tainted `match` expression control flow with two concrete
  payloads.
- Classify this as duplicate evidence for BUG-036 / Session 042, not a new
  bug number.
- Compare against direct exported secret return rejection and a public match
  expression control.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S052-001 | Graphify/source navigation for match-expression privacy taint | `mcp__graphify__.query_graph ... MatchExpr secret taint exported return ...`; source context from prior Session 042 and current repros | Existing evidence already extended BUG-036 to `MatchExpr`, `MatchStmt`, and `WhileStmt` | Build two-value duplicate confirmation probes |
| S052-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S052-003 | Public match expression control | `go run ./cli/cmd/tetra check .../public_match_expr_control.tetra`; `go run ./cli/cmd/tetra run .../public_match_expr_control.tetra` | Passed `check`; `run` printed `exit status 42` | Baseline match-expression return path |
| S052-004 | Secret match-expression control-flow laundering, value 42 | `go run ./cli/cmd/tetra check .../export_secret_match_expr_control_launder_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_match_expr_control_launder_repro.tetra` | Passed `check`; `run` printed `exit status 42` | Duplicate confirmation of BUG-036 through `MatchExpr` |
| S052-005 | Secret match-expression control-flow laundering, value 7 | `go run ./cli/cmd/tetra check .../export_secret_match_expr_control_launder_false_value_repro.tetra`; `go run ./cli/cmd/tetra run .../export_secret_match_expr_control_launder_false_value_repro.tetra` | Passed `check`; `run` printed `exit status 7` | Confirms the public result tracks the secret branch value |

### 2026-05-18 - Session 053 - Generic Identity Privacy Boundary Control

Planned focus:

- Check whether monomorphized generic identity calls preserve secret taint.
- Compare against a public generic identity control and direct exported secret
  return rejection.
- Treat exported-return rejection after `id[Int](raw)` as a no-bug result.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S053-001 | Generic identity taint probe setup | Source/repro inspection around generic function calls and exported privacy boundary | Generic call result is still considered secret-tainted when returning from an exported function | Build generic identity controls |
| S053-002 | Direct exported return control | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | Baseline explicit-flow rejection |
| S053-003 | Public generic identity control | `go run ./cli/cmd/tetra check .../public_generic_identity_control.tetra`; `go run ./cli/cmd/tetra run .../public_generic_identity_control.tetra` | Passed `check`; `run` printed `exit status 42` | Baseline generic call path |
| S053-004 | Secret generic identity attempt, value 42 | `go run ./cli/cmd/tetra check .../export_secret_generic_identity_launder_repro.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'leak'` | No bug in this path |
| S053-005 | Secret generic identity attempt, value 7 | `go run ./cli/cmd/tetra check .../export_secret_generic_identity_false_value_repro.tetra` | Rejected with the same exported-return privacy diagnostic | Confirms generic identity preserves the taint boundary |

### 2026-05-18 - Session 054 - Global Field Privacy Storage Controls

Planned focus:

- Check whether secret-tainted values can be stored through global struct or
  array fields and then read back as public data.
- Keep any global struct-field runtime anomaly classified separately under
  existing BUG-025.
- Treat privacy storage rejection as a no-bug result for these paths.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S054-001 | Global field privacy probe setup | Source/repro inspection around global storage and field assignment | Global privacy checks reject secret-tainted stores to the global aggregate | Build struct and array field controls |
| S054-002 | Public global struct field control | `go run ./cli/cmd/tetra check .../public_global_struct_field_control.tetra`; `go run ./cli/cmd/tetra run .../public_global_struct_field_control.tetra` | Passed `check`; `run` exited 0 without `exit status 42` because the public global struct field assignment shows existing BUG-025 behavior | Not a new privacy bug |
| S054-003 | Secret global struct field store attempt | `go run ./cli/cmd/tetra check .../export_secret_global_struct_field_repro.tetra` | Rejected with `secret-tainted value cannot be stored in global 'leaked'` | No bug in this path |
| S054-004 | Secret global array field store attempt | `go run ./cli/cmd/tetra check .../export_secret_global_array_field_repro.tetra` | Rejected with `secret-tainted value cannot be stored in global 'leaked'` | No bug in this path |

### 2026-05-18 - Session 055 - Task Group Integer Forge Controls

Planned focus:

- Check whether `task.group` can be forged from integer literals, since the
  runtime stores group identifiers as integer-like values.
- Compare a valid `core.task_group_open()` / `core.task_group_cancel()` path
  with direct integer assignment attempts.
- Treat type mismatch rejection as a no-bug result.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S055-001 | Task group representation probe setup | Source/repro inspection around `task.group` and task group builtins | `task.group` is represented with integer storage internally, so the typechecker is the forge barrier | Build literal forge probes |
| S055-002 | Valid task group open/cancel control | `go run ./cli/cmd/tetra check .../valid_task_group_open_cancel_control.tetra` | Passed `check` | Baseline legitimate task group path |
| S055-003 | Zero literal task group assignment | `go run ./cli/cmd/tetra check .../task_group_zero_literal_control.tetra` | Rejected with `type mismatch: expected 'task.group', got 'i32'` | No bug for zero literal assignment |
| S055-004 | Integer literal cancel forge attempt | `go run ./cli/cmd/tetra check .../task_group_int_literal_cancel_forge_repro.tetra` | Rejected with `type mismatch: expected 'task.group', got 'i32'` | No bug for nonzero literal assignment |
| S055-005 | Large integer literal check forge attempt | `go run ./cli/cmd/tetra check .../task_group_large_int_literal_check_repro.tetra` | Rejected with `type mismatch: expected 'task.group', got 'i32'` | No bug for large literal assignment |

### 2026-05-18 - Session 056 - Zeroed Capability Token Controls

Planned focus:

- Check whether zero-initialized globals or local no-init syntax can create
  usable `cap.io` / `cap.mem` tokens.
- Complement Session 051's brace-literal capability forge controls.
- Treat unsupported global types and syntax rejection as a no-bug result.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S056-001 | Zeroed capability token probe setup | Source/repro inspection around global type allow-lists and local declaration syntax | Capability token globals are outside the supported global type set, and local declarations require initialization syntax | Build zero-token controls |
| S056-002 | Zeroed global `cap.io` filesystem attempt | `go run ./cli/cmd/tetra check .../global_zero_cap_io_fs_repro.tetra` | Rejected with `global 'io_cap' has unsupported type 'cap.io' (allowed: i32, bool, ptr, str, u8, u16, task.error)` | No bug for global zeroed `cap.io` |
| S056-003 | Zeroed global `cap.mem` raw-memory attempt | `go run ./cli/cmd/tetra check .../global_zero_cap_mem_raw_memory_repro.tetra` | Rejected with `global 'mem_cap' has unsupported type 'cap.mem' (allowed: i32, bool, ptr, str, u8, u16, task.error)` | No bug for global zeroed `cap.mem` |
| S056-004 | Local no-init `cap.io` syntax attempt | `go run ./cli/cmd/tetra check .../local_zero_cap_io_syntax_probe.tetra` | Rejected with `expected =, got if` | No bug for this local no-init syntax |

### 2026-05-18 - Session 057 - Zeroed Secret Global Boundary Control

Planned focus:

- Check whether a zero-initialized `secret.i32` global can leak through an
  exported unseal/read boundary.
- Distinguish internal existence of a zeroed secret value from an externally
  observable privacy leak.
- Treat exported-return rejection as a no-bug result for this path.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S057-001 | Zeroed secret global probe setup | Source/repro inspection around global secrets and exported privacy boundary | A non-exported function can unseal the zeroed secret global locally, but exported return checks still enforce privacy taint | Build exported boundary control |
| S057-002 | Non-exported zeroed secret global unseal | `go run ./cli/cmd/tetra check .../global_zero_secret_i32_unseal_probe.tetra`; `go run ./cli/cmd/tetra run .../global_zero_secret_i32_unseal_probe.tetra` | Passed `check`; `run` exited 0 with no output | Internal zero value observed, no exported leak |
| S057-003 | Exported zeroed secret global return attempt | `go run ./cli/cmd/tetra check .../global_zero_secret_i32_export_return_probe.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'read'` | No new external privacy leak in this path |

### 2026-05-18 - Session 058 - Function-Typed Budget Context Bypass

Planned focus:

- Check whether static `budget(N)` caller-context guardrails apply when the
  callee is known but reached through a function-typed local, global, or
  callback parameter.
- Use a direct underbudget call as the rejection control.
- Use `budget(5)` callers and a `budget(6)` target so the runtime has enough
  local budget to complete while still violating the static edge requirement.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S058-001 | Graphify/source navigation for budget contexts | `mcp__graphify__.query_graph ... budget semantic clause lowering guard negative overflow direct calls task spawn loops enforcement`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_core.go`, `compiler/tests/safety/plan250_safety_runtime_test.go`, and `docs/spec/core/current_supported_surface.md` | Found budget context validation is a separate AST pass over named calls/spawn strings, while callable semantic-clause checks enforce `realtime`/`noalloc`/`noblock` but not `budget(N)` | Build function-typed underbudget probes |
| S058-002 | Direct underbudget call control | `go run ./cli/cmd/tetra check .../budget_direct_underbudget_rejected_control.tetra` | Rejected with `budget context for call to 'callee' requires caller budget at least 6, got 5` | Baseline static budget context rejection |
| S058-003 | Local function-typed underbudget call | `go run ./cli/cmd/tetra check .../budget_function_typed_local_underbudget_repro.tetra`; `go run ./cli/cmd/tetra run .../budget_function_typed_local_underbudget_repro.tetra` | Passed `check`; `run` printed `exit status 42` after `let f: fn(Int) -> Int uses budget = callee` and `f(41)` | Confirmed BUG-042 |
| S058-004 | Global function-typed underbudget call | `go run ./cli/cmd/tetra check .../budget_function_typed_global_underbudget_repro.tetra`; `go run ./cli/cmd/tetra run .../budget_function_typed_global_underbudget_repro.tetra` | Passed `check`; `run` printed `exit status 42` after `var cb: fn(Int) -> Int uses budget = callee` and `cb(41)` | Extends BUG-042 to function-typed globals |
| S058-005 | Callback underbudget call | `go run ./cli/cmd/tetra check .../budget_callback_underbudget_repro.tetra`; `go run ./cli/cmd/tetra run .../budget_callback_underbudget_repro.tetra` | Passed `check`; `run` printed `exit status 42` after passing `callee` to `apply(..., cb)` where both caller and apply have only `budget(5)` | Extends BUG-042 to callback parameters |
| S058-006 | Covered function-typed local control | `go run ./cli/cmd/tetra check .../budget_function_typed_local_covered_control.tetra`; `go run ./cli/cmd/tetra run .../budget_function_typed_local_covered_control.tetra` | Passed `check`; `run` printed `exit status 42` with caller `budget(6)` | Baseline covered callable path |

### 2026-05-18 - Session 059 - Budget Clause Literal Overflow Controls

Planned focus:

- Check whether `budget(...)` semantic-clause arguments reject negative and
  out-of-range positive numeric literals.
- Classify positive wrap behavior as an extension of BUG-020, because the root
  cause is the same unchecked `int64` to `int32` literal cast.
- Keep `2147483647` as a maximum valid control.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S059-001 | Source navigation for budget literal constants | Source reads in `compiler/internal/frontend/frontend_core.go`, `compiler/internal/frontend/frontend_core.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/internal/semantics/semantics_checker.go` | Lexer parses decimal tokens as `int64`, parser stores `NumberExpr.Value` as `int32`, and budget validation consumes the already-wrapped value via `constI32` | Build boundary literal probes |
| S059-002 | Negative budget control | `go run ./cli/cmd/tetra check .../budget_negative_rejected_control.tetra` | Rejected with `semantic clause 'budget' requires a non-negative value` | Baseline negative rejection |
| S059-003 | `2147483648` budget control | `go run ./cli/cmd/tetra check .../budget_int32_plus_one_rejected_control.tetra` | Rejected with `semantic clause 'budget' requires a non-negative value` after wrapping to negative | Rejected, but diagnostic exposes wrapped interpretation |
| S059-004 | `4294967296` budget wrap repro | `go run ./cli/cmd/tetra check .../budget_uint32_wrap_zero_repro.tetra`; `go run ./cli/cmd/tetra run .../budget_uint32_wrap_zero_repro.tetra` | Passed `check`; `run` printed `exit status 42` | Extends BUG-020 to `budget(...)` semantic-clause arguments |
| S059-005 | `2147483647` max valid budget control | `go run ./cli/cmd/tetra check .../budget_int32_max_valid_control.tetra`; `go run ./cli/cmd/tetra run .../budget_int32_max_valid_control.tetra` | Passed `check`; `run` printed `exit status 42` | Baseline max valid literal |

### 2026-05-18 - Session 060 - Nested Function-Type Consent Checks

Planned focus:

- Check whether `secret.i32` inside function-typed parameters, returns,
  struct fields, and enum payloads is visible to the privacy/consent signature
  checker.
- Compare against direct secret parameter and direct secret aggregate controls.
- Bound the result with closure controls, because synthetic closures with
  direct secret signatures should still be rejected if normal function policy
  validation runs.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S060-001 | Graphify/source navigation for consent and function-typed signatures | `mcp__graphify__.query_graph ... consent semantic clause consent(token) callback function typed callable validation privacy consent.token bypass`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_core.go`, `docs/spec/core/current_supported_surface.md`, and consent tests | Found `TypeRefFunction` resolves to `fnptr`; nested callable signature metadata is tracked separately for callability but not inspected by `typeUsesSecret` | Build nested function-type consent probes |
| S060-002 | Direct secret parameter missing consent control | `go run ./cli/cmd/tetra check .../direct_secret_param_missing_consent_rejected_control.tetra` | Rejected with `secret types in function signature require semantic clause consent(<token>)` | Baseline direct secret signature rejection |
| S060-003 | Function-typed secret parameter without consent | `go run ./cli/cmd/tetra check .../function_typed_secret_param_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra run .../function_typed_secret_param_missing_consent_repro.tetra` | Passed `check`; `run` exited 0 with no consent clause on the enclosing function | Confirmed BUG-043 |
| S060-004 | Function-typed secret return without consent | `go run ./cli/cmd/tetra check .../function_typed_secret_return_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra run .../function_typed_secret_return_missing_consent_repro.tetra` | Passed `check`; `run` exited 0 with no consent clause on the enclosing function | Extends BUG-043 to callback return types |
| S060-005 | Function returning secret-producing callable without consent | `go run ./cli/cmd/tetra check .../function_returning_secret_callable_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra run .../function_returning_secret_callable_missing_consent_repro.tetra` | Passed `check`; `run` exited 0 for `fn(consent.token) -> secret.i32 uses privacy` return type without enclosing consent | Extends BUG-043 to function-typed returns |
| S060-006 | Function-typed secret parameter with consent control | `go run ./cli/cmd/tetra check .../function_typed_secret_param_with_consent_control.tetra`; `go run ./cli/cmd/tetra run .../function_typed_secret_param_with_consent_control.tetra` | Passed `check`; `run` exited 0 when the enclosing function explicitly declared `uses privacy`, `privacy`, and `consent(token)` | Baseline explicit consent path |
| S060-007 | Closure direct secret signature controls | `go run ./cli/cmd/tetra check .../closure_secret_param_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra check .../closure_secret_return_missing_consent_repro.tetra` | Both rejected with `secret types in function signature require semantic clause consent(<token>)` after adding the required `privacy` semantic clause | Confirms ordinary closure FuncDecl policy still sees direct secret signatures |
| S060-008 | Struct and enum wrappers with function-typed secret signatures | `go run ./cli/cmd/tetra check .../struct_function_typed_secret_param_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra run .../struct_function_typed_secret_param_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra check .../enum_function_typed_secret_return_missing_consent_repro.tetra`; `go run ./cli/cmd/tetra run .../enum_function_typed_secret_return_missing_consent_repro.tetra` | Both wrappers passed `check`; both `run` commands exited 0, while a direct `SecretBox.value: secret.i32` aggregate control was rejected with the consent diagnostic | Extends BUG-043 to aggregate-contained function-typed signatures |

### 2026-05-18 - Session 061 - Exported Capability Token ABI Boundary

Planned focus:

- Check whether `@export` can expose opaque capability token parameters as
  host-callable ABI slots.
- Compare against ordinary Tetra source attempts to pass integer literals as
  capability/consent tokens.
- Distinguish `cap.io`/`cap.mem` export exposure from consent-token privacy
  paths that are still blocked by exported-return taint diagnostics.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S061-001 | Graphify/source navigation for exported capability parameters | `mcp__graphify__.query_graph ... @export function cap.io cap.mem consent.token parameter ABI exported capability token forge ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/actorsrt/actorsrt_core.go`, and capability/privacy specs | Found export validation checks names/duplicates but not `TypeCap` signatures; filesystem/raw-memory lowering consumes token slots without validating token values | Build exported capability probes |
| S061-002 | Internal literal token controls | `go run ./cli/cmd/tetra check .../cap_io_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../consent_literal_param_rejected_control.tetra` | Both rejected with type mismatch diagnostics for argument 1 | Baseline internal source cannot pass integer literals as tokens |
| S061-003 | Exported `cap.io` filesystem capability parameter | `go run ./cli/cmd/tetra check .../export_cap_io_param_fs_repro.tetra`; `go run ./cli/cmd/tetra run .../export_cap_io_param_fs_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_cap_io_param_fs_repro.tobj .../export_cap_io_param_fs_repro.tetra`; `rg -a -n "ffi_forged_fs_exists|forged_fs_exists" .../export_cap_io_param_fs_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Confirmed BUG-044 for `cap.io` |
| S061-004 | Exported `cap.mem` raw-memory capability parameter | `go run ./cli/cmd/tetra check .../export_cap_mem_param_load_repro.tetra`; `go run ./cli/cmd/tetra run .../export_cap_mem_param_load_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_cap_mem_param_load_repro.tobj .../export_cap_mem_param_load_repro.tetra`; `rg -a -n "ffi_forged_mem_load|forged_mem_load" .../export_cap_mem_param_load_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Extends BUG-044 to `cap.mem` |
| S061-005 | Exported consent-token privacy round trip | `go run ./cli/cmd/tetra check .../export_consent_token_param_privacy_repro.tetra` | Rejected with `secret-tainted value cannot be returned from @export function 'forged_privacy_roundtrip'` | Not part of BUG-044; exported-return privacy taint blocks this direct consent path |

### 2026-05-18 - Session 062 - Exported Actor/Task Handle ABI Boundary

Planned focus:

- Check whether `@export` can expose `task.group`, `task.i32`, and `actor`
  handles as host-callable ABI slots.
- Compare against ordinary Tetra source attempts to pass integer literals to
  the same wrapper functions.
- Separate new `actor` / `task.group` ABI exposure from the already known
  task-handle forge behavior tracked by BUG-014.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S062-001 | Graphify/source navigation for exported runtime handles | `mcp__graphify__.query_graph ... @export task.group task.i32 actor handle parameter ABI forge ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/actorsrt/actorsrt_core.go`, and actor/runtime specs | Found export validation checks names/duplicates only; task/actor builtins lower handles as raw slots; local actor send and join paths derive scheduler pointers from incoming handle integers | Build exported handle probes |
| S062-002 | Internal literal handle controls | `go run ./cli/cmd/tetra check .../task_group_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../task_i32_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../actor_literal_param_rejected_control.tetra` | All three rejected with type mismatch diagnostics for argument 1 | Baseline internal source cannot pass integer literals as these handle types |
| S062-003 | Exported `task.group` close parameter | `go run ./cli/cmd/tetra check .../export_task_group_param_close_repro.tetra`; `go run ./cli/cmd/tetra run .../export_task_group_param_close_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_task_group_param_close_repro.tobj .../export_task_group_param_close_repro.tetra`; `rg -a -n "ffi_close_group|close_group" .../export_task_group_param_close_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Confirmed BUG-045 for `task.group` |
| S062-004 | Exported `task.i32` join parameter | `go run ./cli/cmd/tetra check .../export_task_i32_param_join_repro.tetra`; `go run ./cli/cmd/tetra run .../export_task_i32_param_join_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_task_i32_param_join_repro.tobj .../export_task_i32_param_join_repro.tetra`; `rg -a -n "ffi_join_task|join_task" .../export_task_i32_param_join_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Exported-ABI extension of BUG-014 and supporting evidence for BUG-045 |
| S062-005 | Exported `actor` send parameter | `go run ./cli/cmd/tetra check .../export_actor_param_send_repro.tetra`; `go run ./cli/cmd/tetra run .../export_actor_param_send_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_actor_param_send_repro.tobj .../export_actor_param_send_repro.tetra`; `rg -a -n "ffi_send_actor|send_actor" .../export_actor_param_send_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Confirmed BUG-045 for `actor` |

### 2026-05-18 - Session 063 - Exported Island Handle ABI Boundary

Planned focus:

- Check whether `@export` can expose `island` handles as host-callable ABI
  slots.
- Compare against ordinary Tetra source attempts to pass integer literals to
  the same island wrapper functions.
- Distinguish the island pointer/header risk from the actor/task handle class
  already recorded in BUG-045.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S063-001 | Graphify/source navigation for exported island handles | `mcp__graphify__.query_graph ... @export island handle parameter ABI forge core.island_alloc core.island_make_u8 ...`; source reads in `docs/spec/memory/islands.md`, `docs/spec/standard_library/stdlib.md`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_memory_resources.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/backend/x64abi/sysv_unix.go` | Found `island` is an opaque resource handle/base pointer; export validation does not filter it; island make/free code reads allocator header fields from the incoming pointer | Build exported island probes |
| S063-002 | Internal literal island controls | `go run ./cli/cmd/tetra check .../island_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../island_free_literal_param_rejected_control.tetra` | Both rejected with type mismatch diagnostics for argument 1 | Baseline internal source cannot pass integer literals as `island` |
| S063-003 | Exported island slice allocation parameter | `go run ./cli/cmd/tetra check .../export_island_param_slice_repro.tetra`; `go run ./cli/cmd/tetra run .../export_island_param_slice_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_island_param_slice_repro.tobj .../export_island_param_slice_repro.tetra`; `rg -a -n "ffi_island_byte_roundtrip|island_byte_roundtrip" .../export_island_param_slice_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Confirmed BUG-046 for `core.island_make_u8` |
| S063-004 | Exported island free parameter | `go run ./cli/cmd/tetra check .../export_island_param_free_repro.tetra`; `go run ./cli/cmd/tetra run .../export_island_param_free_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_island_param_free_repro.tobj .../export_island_param_free_repro.tetra`; `rg -a -n "ffi_free_island|free_island" .../export_island_param_free_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Extends BUG-046 to explicit island cleanup |

### 2026-05-18 - Session 064 - Exported Opaque Handle Return ABI Boundary

Planned focus:

- Check whether `@export` return types can expose opaque capabilities and
  resource handles as raw native ABI return slots.
- Compare against ordinary Tetra source attempts to return integer literals as
  the same opaque handle types.
- Cover both one-slot handles and the two-slot `task.i32` return shape.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S064-001 | Graphify/source navigation for exported handle returns | `mcp__graphify__.query_graph ... @export function return type island cap.io cap.mem actor task.group task.i32 ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/backend/x64obj/builder.go`, and handle docs | Found export validation checks names/duplicates only; lowering/backend preserve return slots and export aliases without opaque-handle filtering | Build return-handle probes |
| S064-002 | Internal literal return controls | `go run ./cli/cmd/tetra check .../cap_io_return_literal_rejected_control.tetra`; repeated for `cap.mem`, `island`, `actor`, `task.group`, and `task.i32` controls | All rejected with `return type mismatch: expected '<handle>', got 'i32'` | Baseline internal source cannot return integer literals as opaque handles |
| S064-003 | Exported capability return tokens | `go run ./cli/cmd/tetra check .../export_cap_io_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_cap_io_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_cap_io_return_repro.tobj ...`; `rg -a -n "ffi_mint_io_cap|mint_io_cap" ...`; repeated for `export_cap_mem_return_repro.tetra` and `ffi_mint_mem_cap` | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; objects contain export aliases and internal symbols | Confirmed BUG-047 for `cap.io` and `cap.mem` returns |
| S064-004 | Exported island/actor return handles | `go run ./cli/cmd/tetra check .../export_island_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_island_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_island_return_repro.tobj ...`; `rg -a -n "ffi_mint_island|mint_island" ...`; repeated for `export_actor_return_repro.tetra` and `ffi_spawn_peer` | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; objects contain export aliases and internal symbols | Extends BUG-047 to `island` and `actor` returns |
| S064-005 | Exported task handle returns | `go run ./cli/cmd/tetra check .../export_task_group_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_task_group_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_task_group_return_repro.tobj ...`; `rg -a -n "ffi_open_group|open_group" ...`; repeated for `export_task_i32_return_repro.tetra` and `ffi_spawn_task` | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; objects contain export aliases and internal symbols, including a two-slot `task.i32` export | Extends BUG-047 to `task.group` and `task.i32` returns |

### 2026-05-18 - Session 065 - Exported Aggregate Handle ABI Boundary

Planned focus:

- Check whether `@export` recursively validates structs, enums, optionals, and
  aggregate returns that contain opaque capability/resource handles.
- Compare against ordinary Tetra source attempts to forge the same nested
  handles with integer literals.
- Keep boxed `island`/`actor` return experiments separate if existing
  resource-provenance checks block the path.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S065-001 | Graphify/source navigation for aggregate handle signatures | `mcp__graphify__.query_graph ... @export aggregate struct enum optional contains cap.io cap.mem island actor task.group task.i32 ...`; source reads in `compiler/internal/semantics/semantics_memory_resources.go`, `compiler/internal/semantics/semantics_checker.go`, `docs/spec/standard_library/stdlib.md`, and `docs/spec/runtime/ownership_v1.md` | Found recursive `typeContainsResourceHandle` support for structs, enum payloads, arrays, and optionals, but `@export` validation only checks names/aliases and does not use that recursive filter or cover `TypeCap` aggregate fields | Build aggregate export probes |
| S065-002 | Internal nested literal controls | `go run ./cli/cmd/tetra check .../struct_cap_io_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_actor_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../optional_task_group_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_cap_io_return_literal_rejected_control.tetra` | All four rejected: `type mismatch for field 'io'`, enum payload expects `actor`, optional parameter type mismatch, and return field `io` mismatch | Baseline internal source cannot forge nested opaque handles |
| S065-003 | Exported struct parameter containing `cap.io` | `go run ./cli/cmd/tetra check .../export_struct_cap_io_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_cap_io_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_cap_io_param_repro.tobj ...`; `rg -a -n "ffi_struct_fs_exists|struct_fs_exists" .../export_struct_cap_io_param_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Confirms BUG-048 for struct-wrapped capability parameters |
| S065-004 | Exported enum and optional resource parameters | `go run ./cli/cmd/tetra check .../export_enum_actor_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_actor_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_actor_param_repro.tobj ...`; `rg -a -n "ffi_send_enveloped_actor|send_enveloped_actor" ...`; repeated for `export_optional_task_group_param_repro.tetra` and `ffi_optional_group_status` | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; objects contain export aliases and internal symbols | Extends BUG-048 to enum payloads and optional resource payloads |
| S065-005 | Exported struct return containing `cap.io` | `go run ./cli/cmd/tetra check .../export_struct_cap_io_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_cap_io_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_cap_io_return_repro.tobj ...`; `rg -a -n "ffi_mint_io_box|mint_io_box" .../export_struct_cap_io_return_repro.tobj` | Passed `check`; `run` printed `exit status 42`; object build succeeded; object contains the export alias and function symbol | Extends BUG-048 to aggregate returns containing capability tokens |
| S065-006 | Boxed `island`/`actor` return experiments | `go run ./cli/cmd/tetra check .../export_struct_island_return_repro.tetra`; `go run ./cli/cmd/tetra check .../export_struct_actor_return_repro.tetra` | Both rejected by existing resource provenance diagnostics after reading the returned field: `ambiguous resource provenance for 'box.handle'` and `ambiguous resource provenance for 'box.peer'` | Not counted as BUG-048 reproducers; useful boundary for future fixes |

### 2026-05-18 - Session 066 - Exported Function-Typed fnptr ABI Boundary

Planned focus:

- Check whether `@export` accepts function-typed parameters and returns as raw
  native ABI slots.
- Compare against ordinary Tetra source attempts to pass or return integer
  literals as function-typed values.
- Distinguish non-capturing callback surfaces from captured closure
  environment slots, because a single-target dispatch can still trust the
  closure target while trusting forged capture slots.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S066-001 | Graphify/source navigation for exported `fnptr` signatures | `mcp__graphify__.query_graph ... @export function typed parameter return fnptr callable ABI signature validation ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/lower/lower_callables.go`, `docs/spec/runtime/runtime_abi.md`, and `docs/spec/core/current_supported_surface.md` | Found `@export` validation checks names/aliases only; `fnptr` is a public 9-slot type; docs describe 9-slot callable payloads; function-typed parameter calls load hidden capture slots from incoming local slots | Build fnptr export probes |
| S066-002 | Internal literal function-typed controls | `go run ./cli/cmd/tetra check .../fnptr_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../fnptr_return_literal_rejected_control.tetra` | Both rejected: callback argument literal must be a supported `fnptr` source, and function-typed return must use a supported `fnptr` source | Baseline internal source cannot forge function-typed values with integers |
| S066-003 | Exported non-capturing function-typed parameter and return | `go run ./cli/cmd/tetra check .../export_fnptr_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fnptr_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fnptr_param_repro.tobj ...`; repeated for `export_fnptr_return_repro.tetra`; TOBJ metadata reader over `compiler/internal/format/tobj/object.go` layout | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; `ffi_apply_callback` metadata is `params=10 returns=1`; `ffi_make_callback` metadata is `params=0 returns=9` | Confirms raw fnptr ABI exposure for non-capturing function-typed values |
| S066-004 | Exported captured callback parameter | `go run ./cli/cmd/tetra check .../export_fnptr_captured_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fnptr_captured_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fnptr_captured_param_repro.tobj ...`; `rg -a -n "ffi_apply_captured_callback|apply_captured_callback" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=10 returns=1` | Confirms BUG-049 for incoming captured callback environments |
| S066-005 | Exported captured callback return | `go run ./cli/cmd/tetra check .../export_fnptr_captured_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fnptr_captured_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fnptr_captured_return_repro.tobj ...`; `rg -a -n "ffi_make_captured_callback|make_captured_callback" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=0 returns=9` | Extends BUG-049 to function-typed returns that publish captured environments |

### 2026-05-18 - Session 067 - Exported Aggregate Function-Typed ABI Boundary

Planned focus:

- Check whether `@export` recursively validates function-typed values hidden in
  struct fields, enum payloads, and aggregate returns.
- Compare against ordinary Tetra source attempts to initialize the same
  aggregate function-typed positions with integer literals.
- Capture TOBJ slot metadata so the result is about native ABI exposure, not
  just semantic acceptance.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S067-001 | Graphify/source navigation for aggregate `fnptr` metadata | `mcp__graphify__.query_graph ... @export aggregate struct enum optional contains function-typed fnptr ...`; source reads in `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_callables.go`, and existing callable tests | Found function-typed struct fields and enum payloads carry special metadata and stored-call lowering reads hidden capture slots from aggregate `fnptr` bases | Build aggregate fnptr export probes |
| S067-002 | Internal aggregate literal controls | `go run ./cli/cmd/tetra check .../struct_fnptr_field_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_fnptr_payload_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_fnptr_field_return_literal_rejected_control.tetra` | All rejected with supported-`fnptr` source diagnostics for the struct field, enum payload, and returned struct field | Baseline internal source cannot forge aggregate-contained function-typed values |
| S067-003 | Exported struct field callback parameter | `go run ./cli/cmd/tetra check .../export_struct_fnptr_field_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_fnptr_field_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_fnptr_field_param_repro.tobj ...`; `rg -a -n "ffi_boxed_callback_apply|boxed_callback_apply" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=10 returns=1` | Confirms BUG-050 for struct field callback parameters |
| S067-004 | Exported enum payload callback parameter | `go run ./cli/cmd/tetra check .../export_enum_fnptr_payload_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_fnptr_payload_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_fnptr_payload_param_repro.tobj ...`; `rg -a -n "ffi_enveloped_callback_apply|enveloped_callback_apply" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=11 returns=1` | Extends BUG-050 to enum payload callback parameters |
| S067-005 | Exported struct return containing callback field | `go run ./cli/cmd/tetra check .../export_struct_fnptr_field_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_fnptr_field_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_fnptr_field_return_repro.tobj ...`; `rg -a -n "ffi_make_callback_box|make_callback_box" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=0 returns=9` | Extends BUG-050 to aggregate returns containing function-typed fields |

### 2026-05-18 - Session 068 - Exported String/Slice ptr,len ABI Boundary

Planned focus:

- Check whether `@export` accepts `String`/`str` and `[]u8` parameters as raw
  native ABI slot pairs.
- Compare against ordinary Tetra source attempts to pass or return integer
  literals where `String`/slice values are expected.
- Capture TOBJ metadata for direct parameters and returns so the result is about
  exported ABI exposure, not only local semantic acceptance.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S068-001 | Graphify/source navigation for exported string/slice views | `mcp__graphify__.query_graph ... @export String str slice []u8 parameter return ptr len ABI signature validation ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `docs/spec/standard_library/stdlib.md`, and `docs/spec/runtime/runtime_abi.md` | Found export validation checks names/aliases only; `str` and slices are public two-slot `ptr,len` values; docs describe explicit host `ptr,len` strings as a boundary contract | Build direct String/slice export probes |
| S068-002 | Internal literal String/slice controls | `go run ./cli/cmd/tetra check .../string_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../slice_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../string_return_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../slice_return_literal_rejected_control.tetra` | All rejected: String and slice argument controls report type mismatches; return controls report expected `str`/`[]u8`, got `i32` | Baseline internal source cannot forge String/slice views from integers |
| S068-003 | Exported String parameter view | `go run ./cli/cmd/tetra check .../export_string_param_index_repro.tetra`; `go run ./cli/cmd/tetra run .../export_string_param_index_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_string_param_index_repro.tobj ...`; `rg -a -n "ffi_string_first_byte|string_first_byte" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=2 returns=1` | Confirms BUG-051 for incoming `String` `ptr,len` |
| S068-004 | Exported slice parameter view | `go run ./cli/cmd/tetra check .../export_slice_param_index_repro.tetra`; `go run ./cli/cmd/tetra run .../export_slice_param_index_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_slice_param_index_repro.tobj ...`; `rg -a -n "ffi_slice_first_byte|slice_first_byte" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=2 returns=1` | Confirms BUG-051 for incoming `[]u8` `ptr,len` |
| S068-005 | Exported String and slice returns | `go run ./cli/cmd/tetra check .../export_string_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_string_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_string_return_repro.tobj ...`; repeated for `export_slice_return_repro.tetra`; `rg -a -n "ffi_make_string|make_string" ...`; `rg -a -n "ffi_make_slice|make_slice" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; aliases exist; metadata is `params=0 returns=2` for both aliases | Extends BUG-051 to returned internal string/slice views |

### 2026-05-18 - Session 069 - Exported Aggregate String/Slice ABI Boundary

Planned focus:

- Check whether `@export` recursively validates `String`/`str` and `[]u8`
  values hidden in structs, enum payloads, optionals, and aggregate returns.
- Compare against ordinary Tetra source attempts to initialize those nested
  positions with integer literals.
- Capture TOBJ slot metadata so the result is about native ABI exposure, not
  just semantic acceptance.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S069-001 | Graphify/source navigation for aggregate string/slice views | `mcp__graphify__.query_graph ... @export aggregate struct enum optional String str []u8 slice ptr len ABI signature validation ...`; `mcp__graphify__.get_neighbors CheckWorldOpt() relation_filter=call`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `docs/spec/standard_library/stdlib.md`, and prior aggregate fnptr repros | Found export validation checks names/aliases only; struct slot layout sums field slot counts; `String`/slices are two-slot values and optionals add one tag slot | Build aggregate String/slice export probes |
| S069-002 | Internal nested literal controls | `go run ./cli/cmd/tetra check .../struct_string_field_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_slice_field_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_string_payload_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../optional_string_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_string_field_return_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_slice_field_return_literal_rejected_control.tetra` | All rejected: field mismatches for `text`/`bytes`, enum payload expected `str`, optional parameter mismatch, and return field mismatches | Baseline internal source cannot forge nested String/slice views from integers |
| S069-003 | Exported struct field String/slice parameters | `go run ./cli/cmd/tetra check .../export_struct_string_field_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_string_field_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_string_field_param_repro.tobj ...`; repeated for `export_struct_slice_field_param_repro.tetra`; `rg -a -n "ffi_boxed_string_first_byte|boxed_string_first_byte" ...`; `rg -a -n "ffi_boxed_slice_first_byte|boxed_slice_first_byte" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; aliases exist; metadata is `params=2 returns=1` for both aliases | Confirms BUG-052 for struct-contained raw `ptr,len` parameters |
| S069-004 | Exported enum and optional String payload parameters | `go run ./cli/cmd/tetra check .../export_enum_string_payload_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_string_payload_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_string_payload_param_repro.tobj ...`; repeated for `export_optional_string_param_repro.tetra`; `rg -a -n "ffi_enveloped_string_first_byte|enveloped_string_first_byte" ...`; `rg -a -n "ffi_optional_string_first_byte|optional_string_first_byte" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; aliases exist; both metadata entries are `params=3 returns=1` | Extends BUG-052 to tag-plus-`ptr,len` enum/optional payloads |
| S069-005 | Exported struct returns containing String/slice fields | `go run ./cli/cmd/tetra check .../export_struct_string_field_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_string_field_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_string_field_return_repro.tobj ...`; repeated for `export_struct_slice_field_return_repro.tetra`; `rg -a -n "ffi_make_string_box|make_string_box" ...`; `rg -a -n "ffi_make_slice_box|make_slice_box" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; aliases exist; metadata is `params=0 returns=2` for both aliases | Extends BUG-052 to returned aggregate views |

### 2026-05-18 - Session 070 - Exported Fixed-Array ABI Boundary

Planned focus:

- Check whether `@export` accepts fixed-array (`[N]T`) parameters and returns as
  raw native ABI slot pairs.
- Compare against ordinary Tetra source attempts to pass or return integer
  literals as `[1]Int`, and against the source guard that rejects fixed-array
  `ptr`/`len` assignment.
- Keep this distinct from BUG-024, which already covers zeroed fixed-array
  fields trapping at runtime.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S070-001 | Graphify/source navigation for fixed-array ABI | `mcp__graphify__.query_graph ... @export fixed array TypeArray [1]u8 ptr len ABI signature validation ...`; `mcp__graphify__.get_neighbors makeArrayTypeInfo`; `mcp__graphify__.get_neighbors resolveAssignTarget`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, and `compiler/internal/lower/lower_core.go` | Found `TypeArray` is a two-slot `ptr,len` type; fixed-array `ptr`/`len` assignment is rejected in source; export validation checks names/aliases only | Build direct fixed-array export probes |
| S070-002 | Internal fixed-array controls | `go run ./cli/cmd/tetra check .../fixed_array_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../fixed_array_return_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../fixed_array_internal_len_assignment_rejected_control.tetra` | All rejected: param literal mismatch, return literal mismatch, and fixed-array internals assignment diagnostic | Baseline source cannot forge `[1]Int` from an integer or mutate fixed-array `len` |
| S070-003 | Exported fixed-array parameter length reader | `go run ./cli/cmd/tetra check .../export_fixed_array_param_len_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fixed_array_param_len_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fixed_array_param_len_repro.tobj ...`; `rg -a -n "ffi_fixed_array_len|fixed_array_len" ...`; TOBJ metadata reader | Passed `check`; `run` exited 0; object build succeeded; alias exists; metadata is `params=2 returns=1` | Confirms BUG-053 for incoming fixed-array `ptr,len` metadata |
| S070-004 | Exported fixed-array index parameter | `go run ./cli/cmd/tetra check .../export_fixed_array_param_index_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fixed_array_param_index_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fixed_array_param_index_repro.tobj ...`; `rg -a -n "ffi_fixed_array_first|fixed_array_first" ...`; TOBJ metadata reader | Passed `check`; `run` exited 0; object build succeeded; alias exists; metadata is `params=2 returns=1` | Confirms BUG-053 reaches the normal fixed-array index path |
| S070-005 | Exported fixed-array echo return | `go run ./cli/cmd/tetra check .../export_fixed_array_echo_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_fixed_array_echo_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_fixed_array_echo_return_repro.tobj ...`; `rg -a -n "ffi_echo_fixed_array|echo_fixed_array" ...`; TOBJ metadata reader | Passed `check`; `run` exited 0; object build succeeded; alias exists; metadata is `params=2 returns=2` | Extends BUG-053 to returned fixed-array views |

### 2026-05-18 - Session 071 - Exported Aggregate Fixed-Array ABI Boundary

Planned focus:

- Check whether `@export` recursively validates `[N]T` values hidden in structs,
  enum payloads, optionals, and aggregate returns.
- Compare against ordinary Tetra source attempts to initialize those nested
  positions with integer literals.
- Capture TOBJ slot metadata while keeping runtime execution separate from
  BUG-024's zeroed fixed-array storage trap.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S071-001 | Graphify/source navigation for aggregate fixed arrays | `mcp__graphify__.query_graph ... @export aggregate struct enum optional fixed array TypeArray [1]Int ptr len ABI signature validation ...`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/tests/semantics/semantics_core_language_test.go`, `docs/spec/standard_library/stdlib.md`, and BUG-053 | Found `TypeArray` is two-slot `ptr,len`; struct layout sums slot counts; fixed arrays in structs/optionals have build-smoke coverage; export validation checks names/aliases only | Build aggregate fixed-array export probes |
| S071-002 | Internal aggregate literal controls | `go run ./cli/cmd/tetra check .../struct_fixed_array_field_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_fixed_array_payload_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../optional_fixed_array_literal_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_fixed_array_field_return_literal_rejected_control.tetra` | All rejected: struct field mismatch, enum payload expected `[1]i32`, optional parameter mismatch, and returned struct field mismatch | Baseline source cannot forge nested fixed-array views from integers |
| S071-003 | Exported struct field fixed-array parameters | `go run ./cli/cmd/tetra check .../export_struct_fixed_array_field_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_fixed_array_field_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_fixed_array_field_param_repro.tobj ...`; repeated for `export_struct_fixed_array_field_index_repro.tetra`; `rg -a -n "ffi_boxed_fixed_array_len|boxed_fixed_array_len" ...`; `rg -a -n "ffi_boxed_fixed_array_first|boxed_fixed_array_first" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands exited 0; object builds succeeded; aliases exist; metadata is `params=2 returns=1` for both aliases | Confirms BUG-054 for struct-contained fixed-array metadata, including index path |
| S071-004 | Exported enum and optional fixed-array payload parameters | `go run ./cli/cmd/tetra check .../export_enum_fixed_array_payload_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_fixed_array_payload_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_fixed_array_payload_param_repro.tobj ...`; repeated for `export_optional_fixed_array_param_repro.tetra`; `rg -a -n "ffi_enveloped_fixed_array_len|enveloped_fixed_array_len" ...`; `rg -a -n "ffi_optional_fixed_array_len|optional_fixed_array_len" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands exited 0; object builds succeeded; aliases exist; both metadata entries are `params=3 returns=1` | Extends BUG-054 to tag-plus-`ptr,len` enum/optional payloads |
| S071-005 | Exported aggregate returns containing fixed arrays | `go run ./cli/cmd/tetra check .../export_struct_fixed_array_field_return_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_fixed_array_field_return_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_fixed_array_field_return_repro.tobj ...`; repeated for `export_enum_fixed_array_payload_return_repro.tetra`; `rg -a -n "ffi_make_fixed_array_box|make_fixed_array_box" ...`; `rg -a -n "ffi_wrap_fixed_array_envelope|wrap_fixed_array_envelope" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands exited 0; object builds succeeded; aliases exist; struct return metadata is `params=0 returns=2`; enum return metadata is `params=2 returns=3` | Extends BUG-054 to aggregate return shapes |

### 2026-05-18 - Session 072 - Exported Bool Scalar ABI Boundary

Planned focus:

- Check whether exported `Bool` parameters, returns, and nested aggregate
  payloads preserve the source-level boolean invariant.
- Compare against ordinary Tetra source attempts to pass, return, or store
  integer literals in `Bool` positions.
- Capture TOBJ metadata and lowering/backend evidence for raw one-slot truth
  handling.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S072-001 | Graphify/source navigation for Bool ABI | `mcp__graphify__.query_graph ... @export Bool bool parameter return ABI signature validation ...`; `mcp__graphify__.get_neighbors baseTypes`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/backend/x64core/x64core_core.go`, `compiler/internal/format/tobj/object.go`, and prior Session 015 controls | Found `bool` is a one-slot `TypeBool`; source rejects integer-as-Bool; export validation checks names/aliases only; TOBJ symbol signatures store only slot counts; branch lowering tests one slot for zero/nonzero | Build exported Bool boundary probes |
| S072-002 | Internal Bool integer controls | `go run ./cli/cmd/tetra check .../bool_int_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../bool_return_int_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../struct_bool_field_int_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_bool_payload_int_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../optional_bool_int_rejected_control.tetra` | All rejected: direct param mismatch, return mismatch, struct field mismatch, enum payload expected `bool`, and optional parameter mismatch | Baseline source cannot forge Bool values from integers |
| S072-003 | Exported direct Bool parameter and return | `go run ./cli/cmd/tetra check .../export_bool_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_bool_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_bool_param_repro.tobj ...`; repeated for `export_bool_return_repro.tetra`; `rg -a -n "ffi_bool_gate|bool_gate" ...`; `rg -a -n "ffi_is_ready|is_ready" ...`; TOBJ metadata reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; aliases exist; `ffi_bool_gate` metadata is `params=1 returns=1`; `ffi_is_ready` metadata is `params=0 returns=1` | Confirms BUG-055 for direct Bool slots |
| S072-004 | Exported aggregate Bool parameters | `go run ./cli/cmd/tetra check .../export_struct_bool_field_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_struct_bool_field_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_bool_field_param_repro.tobj ...`; repeated for enum and optional Bool payload repros; `rg -a -n "ffi_boxed_bool_gate|boxed_bool_gate" ...`; `rg -a -n "ffi_enveloped_bool_gate|enveloped_bool_gate" ...`; `rg -a -n "ffi_optional_bool_gate|optional_bool_gate" ...`; TOBJ metadata reader | All three passed `check`; all three `run` commands printed `exit status 42`; object builds succeeded; aliases exist; struct metadata is `params=1 returns=1`; enum and optional metadata are `params=2 returns=1` | Extends BUG-055 to aggregate/tagged Bool payloads |

### 2026-05-18 - Session 073 - Guarded Default Expression Exhaustiveness

Planned focus:

- Check whether guarded default arms in expression-form `match` and `catch`
  count as exhaustive.
- Compare against guarded concrete arms, which should remain non-exhaustive, and
  unguarded defaults, which should be accepted.
- Verify runtime behavior when the guarded default's guard is false.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S073-001 | Graphify/source navigation for guarded defaults | `mcp__graphify__.query_graph ... catch expression default guard exhaustive ...`; source reads in `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/semantics/semantics_checker.go`, and `compiler/internal/lower/lower_core.go` | Found `match`/`catch` expression fallback exhaustiveness loops count any default arm, while complete-pattern helpers and statement `matchHasDefault` skip guarded arms | Build guarded-default expression probes |
| S073-002 | Guarded concrete case controls | `go run ./cli/cmd/tetra check .../match_guarded_case_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../catch_guarded_case_rejected_control.tetra` | Both rejected with `match expression must be exhaustive` and `catch expression must be exhaustive` | Baseline guarded non-default arms do not count as exhaustive |
| S073-003 | Unguarded and true-guard default controls | `go run ./cli/cmd/tetra check .../match_unguarded_default_control.tetra`; `go run ./cli/cmd/tetra run .../match_unguarded_default_control.tetra`; repeated for `catch_unguarded_default_control.tetra` and `catch_guarded_default_true_control.tetra` | All passed `check`; all run commands printed `exit status 42`; object builds for catch controls succeeded | Confirms normal default behavior and that guarded defaults are accepted |
| S073-004 | False-guarded `match` default repro | `go run ./cli/cmd/tetra check .../match_guarded_default_false_repro.tetra`; `go run ./cli/cmd/tetra run .../match_guarded_default_false_repro.tetra`; `go run ./cli/cmd/tetra build -o .../match_guarded_default_false_repro.bin ...` | Passed `check`; `run` printed `exit status 42` from `value == 0` after no guarded arm stored `99`; build succeeded | Confirms BUG-056 for `match` expressions |
| S073-005 | False-guarded `catch` default repro | `go run ./cli/cmd/tetra check .../catch_guarded_default_false_repro.tetra`; `go run ./cli/cmd/tetra run .../catch_guarded_default_false_repro.tetra`; `go run ./cli/cmd/tetra build -o .../catch_guarded_default_false_repro.bin ...` | Passed `check`; `run` printed `exit status 42` from `value == 0` after the thrown error skipped the false guarded default body; build succeeded | Confirms BUG-056 for typed-error `catch` expressions |

### 2026-05-18 - Session 074 - Exported Enum Discriminant ABI Boundary

Planned focus:

- Re-check the collection-loop/lowering element-type hypothesis without
  duplicating Session 022.
- Probe whether exported enum parameters preserve the source-level closed-set
  invariant for discriminant tags.
- Compare direct enum parameters and enum-with-payload parameters against
  ordinary source controls that try to pass or return raw integers.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S074-001 | Collection loop source/navigation follow-up | `mcp__graphify__.get_neighbors collectionElementType`; `mcp__graphify__.get_neighbors lowerIndexLoadKind`; source reads in `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, and prior Session 022 notes | Found `ensureTypeInfo` already rejects unsupported `[]T`/`[N]T` elements such as `str`, `ptr`, slices, arrays, and multi-slot structs; no new collection-loop bug beyond the fixed-array storage territory already covered by BUG-024/Session 022 | Pivot to ABI boundaries |
| S074-002 | Internal enum integer controls | `go run ./cli/cmd/tetra check .../enum_int_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../enum_int_return_rejected_control.tetra` | Rejected with `type mismatch for 'route_decision' arg 1` and `return type mismatch: expected 'Route', got 'i32'` | Baseline source cannot forge enum discriminants from integers |
| S074-003 | Exported direct enum tag parameter | `go run ./cli/cmd/tetra check .../export_enum_tag_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_tag_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_tag_param_repro.tobj ...`; `rg -a -n "ffi_route_decision|route_decision" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=1 returns=1` for both `ffi_route_decision` and `route_decision` | Confirms BUG-057 for direct enum discriminant slots |
| S074-004 | Exported enum payload tag parameter | `go run ./cli/cmd/tetra check .../export_enum_payload_tag_param_repro.tetra`; `go run ./cli/cmd/tetra run .../export_enum_payload_tag_param_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_enum_payload_tag_param_repro.tobj ...`; `rg -a -n "ffi_request_decision|request_decision" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=2 returns=1` for tag plus payload slot | Extends BUG-057 to payload-bearing enums |

### 2026-05-18 - Session 075 - Exported Optional Presence Tag ABI Boundary

Planned focus:

- Check whether `@export` preserves the canonical `0/1` optional presence tag
  invariant for `Int?`.
- Compare source-level `none`, implicit `some`, and attempted tag-field
  mutation controls.
- Capture TOBJ slot metadata and lowering evidence for `match` and `if let`
  optional unwrapping.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S075-001 | Graphify/source navigation for optional ABI | `mcp__graphify__.query_graph ... optional presence tag exported ABI ...`; `mcp__graphify__.get_neighbors matchExprHasCompleteOptionalPatterns`; `mcp__graphify__.get_neighbors lowerMatchExpr`; source reads in `docs/spec/flow/flow_syntax_v1.md`, `compiler/internal/lower/lower_core.go`, `compiler/internal/semantics/semantics_core.go`, and `compiler/internal/semantics/semantics_checker.go` | Found spec layout is presence tag plus payload slots; lowering emits canonical `0/1` for source values but checks `some` via `IRJmpIfZero`; export validation checks names only | Build optional tag boundary probes |
| S075-002 | Source optional controls | `go run ./cli/cmd/tetra check .../optional_none_control.tetra`; `go run ./cli/cmd/tetra run .../optional_none_control.tetra`; repeated for `optional_implicit_some_control.tetra`; `go run ./cli/cmd/tetra check .../optional_tag_field_rejected_control.tetra` | `none` and implicit `some` controls passed and printed `exit status 42`; `maybe.tag = 2` was rejected with `'i32?' is not a struct` | Baseline source creates canonical optionals and cannot mutate a tag field |
| S075-003 | Exported optional `match` parameter | `go run ./cli/cmd/tetra check .../export_optional_int_match_repro.tetra`; `go run ./cli/cmd/tetra run .../export_optional_int_match_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_optional_int_match_repro.tobj ...`; `rg -a -n "ffi_optional_status|optional_status" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=2 returns=1` for both `ffi_optional_status` and `optional_status` | Confirms BUG-058 for `match` optional unwrapping |
| S075-004 | Exported optional `if let` parameter | `go run ./cli/cmd/tetra check .../export_optional_int_iflet_repro.tetra`; `go run ./cli/cmd/tetra run .../export_optional_int_iflet_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_optional_int_iflet_repro.tobj ...`; `rg -a -n "ffi_optional_iflet|optional_iflet" ...`; TOBJ metadata reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; alias exists; metadata is `params=2 returns=1` for both `ffi_optional_iflet` and `optional_iflet` | Extends BUG-058 to `if let some(...)` |

### 2026-05-18 - Session 076 - Exported Consent Token ABI Boundary

Planned focus:

- Check whether `@export` can expose `consent.token` and `secret.i32`
  signatures as raw native ABI slots.
- Separate consent-token ABI exposure from existing secret-taint leak classes by
  testing return, branch, and global-store controls.
- Capture TOBJ slot metadata and lowered sentinel evidence.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S076-001 | Graphify/source navigation for consent and secret signatures | `mcp__graphify__.query_graph ... secret.i32 @export function signature consent privacy ...`; `mcp__graphify__.get_neighbors typeUsesSecret`; `mcp__graphify__.get_neighbors validateFunctionPolicyClauses`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/lower/lower_suite_test.go`, `compiler/internal/format/tobj/object.go`, and privacy docs | Found `typeUsesSecret()` drives privacy/consent policy checks and taint tracking; export validation checks names/duplicates only; lowering emits an exact sentinel guard for consent clauses; TOBJ signatures store only slot counts | Build consent/secret exported boundary probes |
| S076-002 | Internal consent and secret literal controls | `go run ./cli/cmd/tetra check .../consent_literal_param_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../secret_literal_param_rejected_control.tetra` | Rejected with `type mismatch for 'require_consent' arg 1` and `type mismatch for 'consume' arg 2` | Baseline source cannot pass raw integers as `consent.token` or `secret.i32` |
| S076-003 | Exported consent-token guard | `go run ./cli/cmd/tetra check .../export_consent_token_guard_repro.tetra`; `go run ./cli/cmd/tetra run .../export_consent_token_guard_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_consent_token_guard_repro.tobj ...`; TOBJ metadata/sentinel reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; `ffi_require_consent` and `require_consent` are `params=1 returns=1`; generated code contains sentinel bytes at offsets `27` and `93` | Confirms BUG-059 for the exported consent policy slot |
| S076-004 | Exported secret-bearing signatures without direct leak | `go run ./cli/cmd/tetra check .../export_secret_param_ignore_probe.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_secret_param_ignore_probe.tobj ...`; `go run ./cli/cmd/tetra check .../export_secret_param_unseal_discard_probe.tetra`; `go run ./cli/cmd/tetra run .../export_secret_param_unseal_discard_probe.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_secret_param_unseal_discard_probe.tobj ...`; TOBJ metadata/sentinel reader | Both passed `check`; unseal-discard printed `exit status 42`; object builds succeeded; exported/internal symbols are `params=2 returns=1`; generated code contains consent sentinel bytes | Shows secret-bearing exports can accept host-supplied consent/secret slots even when taint checks prevent value exfiltration |
| S076-005 | Exported secret leak controls | `go run ./cli/cmd/tetra check .../export_secret_return_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../export_secret_param_unseal_return_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../export_secret_param_branch_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../export_secret_param_global_store_rejected_control.tetra` | Rejected with exported-return secret-taint diagnostics for secret return, raw unseal return, and branch-selected return; global store rejected with `secret-tainted value cannot be stored in global 'leaked'` | Confirms BUG-059 is consent-token ABI exposure, not a new direct secret-taint leak |

### 2026-05-18 - Session 077 - Generic `@export` Symbol Drop

Planned focus:

- Check whether generic functions can be annotated with `@export` despite not
  having a single concrete native ABI.
- Compare unused and used generic exports against a normal non-generic export
  and a non-exported generic specialization.
- Inspect TOBJ symbols directly so the result is about emitted ABI artifacts,
  not only semantic acceptance.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S077-001 | Graphify/source navigation for exported generics | `mcp__graphify__.query_graph ... @export generic function monomorphization exported symbol ...`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, and generic tests | Found `@export` name validation runs on the generic declaration, generic bodies are skipped in the later concrete checking path, and `cloneGenericFunc()` clears `ExportName` on specializations | Build generic export probes |
| S077-002 | Unused exported generic | `go run ./cli/cmd/tetra check .../export_generic_unused_repro.tetra`; `go run ./cli/cmd/tetra run .../export_generic_unused_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_generic_unused_repro.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; TOBJ symbol table contains only `main`, with no `ffi_generic_id` | Confirms `@export` declaration can disappear entirely |
| S077-003 | Used exported generic | `go run ./cli/cmd/tetra check .../export_generic_used_repro.tetra`; `go run ./cli/cmd/tetra run .../export_generic_used_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_generic_used_repro.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; TOBJ symbol table contains `id__T_i32` and `main`, with no `ffi_generic_id` alias | Confirms BUG-060 even when monomorphization creates a concrete specialization |
| S077-004 | Non-generic export control | `go run ./cli/cmd/tetra check .../export_plain_control.tetra`; `go run ./cli/cmd/tetra run .../export_plain_control.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_plain_control.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; TOBJ includes `ffi_plain_id` and `plain_id`, both `params=1 returns=1` | Baseline export alias emission works for concrete functions |
| S077-005 | Plain generic control | `go run ./cli/cmd/tetra check .../generic_plain_control.tetra`; `go run ./cli/cmd/tetra run .../generic_plain_control.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../generic_plain_control.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; TOBJ includes `id__T_i32` and `main` | Baseline generic monomorphization works when no export contract is promised |

### 2026-05-18 - Session 078 - Exported Typed-Error ABI Metadata

Planned focus:

- Check whether `@export` on `throws` functions preserves typed-error metadata
  at the native object boundary.
- Compare exported throwing functions against source-level bare-call rejection
  and normal `catch` recovery.
- Compare compact throwing `returns=2` against a non-throwing two-slot struct
  return to test TOBJ ambiguity.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S078-001 | Graphify/source navigation for typed-error export ABI | `mcp__graphify__.query_graph ... @export typed throws function ABI error payload trap status ReturnSlots TOBJ ...`; source reads in `docs/spec/flow/flow_syntax_v1.md`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/format/tobj/object.go` | Found source typed errors carry success/error/status slot layout; checker computes throwing return slot count; lowering emits only return slots into `IRFunc`; TOBJ records only slot counts | Build throwing export probes |
| S078-002 | Source throwing controls | `go run ./cli/cmd/tetra check .../throwing_bare_call_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../throwing_catch_control.tetra`; `go run ./cli/cmd/tetra run .../throwing_catch_control.tetra` | Bare call rejected with `call to throwing function 'read' requires try`; catch control passed and printed `exit status 42` | Baseline source preserves typed-error control flow |
| S078-003 | Exported compact throwing function | `go run ./cli/cmd/tetra check .../export_throwing_compact_repro.tetra`; `go run ./cli/cmd/tetra run .../export_throwing_compact_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_throwing_compact_repro.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; `ffi_read_compact` and `read_compact` are `params=1 returns=2` | Confirms BUG-061 for compact typed-error export metadata |
| S078-004 | Exported payload throwing function | `go run ./cli/cmd/tetra check .../export_throwing_payload_repro.tetra`; `go run ./cli/cmd/tetra run .../export_throwing_payload_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_throwing_payload_repro.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; `ffi_read_payload` and `read_payload` are `params=1 returns=4` | Extends BUG-061 to payload-bearing error enums |
| S078-005 | Non-throwing two-slot control | `go run ./cli/cmd/tetra check .../export_struct_two_slot_control.tetra`; `go run ./cli/cmd/tetra run .../export_struct_two_slot_control.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_struct_two_slot_control.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; `ffi_pair` and `pair` are `params=0 returns=2` | Shows compact throwing metadata is slot-count indistinguishable from ordinary two-slot returns |

### 2026-05-18 - Session 079 - Exported Ownership-Marker ABI Metadata

Planned focus:

- Check whether `@export` preserves `borrow`, `consume`, and `inout`
  parameter contracts in native object metadata.
- Compare ownership-marked exports against owned-parameter controls with the
  same slot shape.
- Verify that ordinary Tetra source still enforces ownership markers, so the
  probe is about the exported ABI boundary rather than decorative syntax.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S079-001 | Graphify/source navigation for ownership export ABI | `mcp__graphify__.query_graph ... @export borrow consume inout parameter ownership markers ABI TOBJ ...`; `mcp__graphify__.get_neighbors ParamOwnership`; source reads in `docs/spec/core/current_supported_surface.md`, `compiler/internal/semantics/semantics_core.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/backend/x64obj/builder.go`, and `compiler/internal/format/tobj/object.go` | Found ownership markers are source contracts carried in `ParamOwnership`; checker rejects borrow/inout misuse and actor/task transfers; TOBJ symbols store only slot counts | Build ownership-marked export probes |
| S079-002 | Source ownership controls | `go run ./cli/cmd/tetra check .../borrow_to_owned_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../inout_from_borrow_rejected_control.tetra` | Rejected with `borrowed value derived from 'buf' cannot be passed to non-borrow parameter 1 of 'owned_first'` and `borrowed value derived from 'buf' cannot be passed as inout to 'fill_first'` | Baseline source preserves ownership marker semantics |
| S079-003 | Exported `borrow` slice vs owned slice metadata | `go run ./cli/cmd/tetra check .../export_borrow_slice_repro.tetra`; `go run ./cli/cmd/tetra run .../export_borrow_slice_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_borrow_slice_repro.tobj ...`; repeated `check`, `run`, and object build for `export_owned_slice_control.tetra`; TOBJ symbol reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; `ffi_borrow_first`/`borrow_first` and `ffi_owned_first`/`owned_first` are all `params=2 returns=1` | Confirms BUG-062 for borrowed slice exports |
| S079-004 | Exported `consume` Int vs owned Int metadata | `go run ./cli/cmd/tetra check .../export_consume_int_repro.tetra`; `go run ./cli/cmd/tetra run .../export_consume_int_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_consume_int_repro.tobj ...`; repeated for `export_owned_int_control.tetra`; TOBJ symbol reader | Both passed `check`; both `run` commands printed `exit status 42`; object builds succeeded; `ffi_take_int`/`take_int` and `ffi_owned_int`/`owned_int` are all `params=1 returns=1` | Extends BUG-062 to consumed scalar exports |
| S079-005 | Exported `inout` slice metadata | `go run ./cli/cmd/tetra check .../export_inout_slice_repro.tetra`; `go run ./cli/cmd/tetra run .../export_inout_slice_repro.tetra`; `go run ./cli/cmd/tetra build -emit object -o .../export_inout_slice_repro.tobj ...`; TOBJ symbol reader | Passed `check`; `run` printed `exit status 42`; object build succeeded; `ffi_fill_first` and `fill_first` are `params=2 returns=1` | Extends BUG-062 to mutable `inout` exports |

### 2026-05-18 - Session 080 - Deferred Cleanup Partial-Consume Captures

Planned focus:

- Check whether deferred cleanup capture tracking is path-aware for partial
  struct-field consumes.
- Compare deferred cleanup behavior against ordinary immediate whole-value use
  after field consume.
- Preserve the sibling-field case as a control, since consuming `pair.left`
  should not poison cleanup that only needs `pair.right`.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S080-001 | Graphify/source navigation for defer ownership captures | `mcp__graphify__.query_graph ... try catch defer cleanup error propagation lowering semantics ...`; `mcp__graphify__.get_neighbors ParamOwnership`; source reads in `compiler/tests/semantics/semantics_core_language_test.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_memory_resources.go`, `compiler/internal/semantics/semantics_memory_resources.go`, `docs/spec/core/current_supported_surface.md`, and `docs/spec/runtime/ownership_v1.md` | Found tests for simple deferred capture after whole consume, but pending defer validation checks captured base names via `consumedAt(name)` while ordinary whole-value checks have descendant-aware consumed-path logic | Build partial field-consume cleanup probes |
| S080-002 | Immediate and whole-consume controls | `go run ./cli/cmd/tetra check .../immediate_whole_after_field_consume_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../defer_captures_whole_then_whole_consumed_rejected_control.tetra` | Rejected with `cannot use consumed value 'pair.left'` and `defer cleanup captures value 'pair' ... but it was consumed ... before cleanup ran` | Baseline immediate ownership and direct deferred whole-local tracking work |
| S080-003 | Deferred field capture after same field consume | `go run ./cli/cmd/tetra check .../defer_captures_field_then_field_consumed_repro.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_field_then_field_consumed_repro.tetra`; `go run ./cli/cmd/tetra build -o .../defer_captures_field_then_field_consumed_repro.bin ...` | Passed `check`; `run` printed `fieldexit status 42`; build succeeded | Confirms BUG-063 for direct field cleanup capture |
| S080-004 | Deferred whole-value capture after child field consume | `go run ./cli/cmd/tetra check .../defer_captures_whole_then_field_consumed_repro.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_whole_then_field_consumed_repro.tetra`; `go run ./cli/cmd/tetra build -o .../defer_captures_whole_then_field_consumed_repro.bin ...` | Passed `check`; `run` printed `wholeexit status 42`; build succeeded | Extends BUG-063 to whole-value cleanup capture with consumed descendant |
| S080-005 | Deferred sibling-field capture control | `go run ./cli/cmd/tetra check .../defer_captures_sibling_after_field_consume_control.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_sibling_after_field_consume_control.tetra` | Passed `check`; `run` printed `siblingexit status 42` | Confirms sibling-path reuse remains a valid case |

### 2026-05-18 - Session 081 - Deferred Cleanup Enum/Optional Payload Captures

Planned focus:

- Check whether BUG-063 also affects enum-payload and optional-payload child
  ownership paths.
- Compare deferred whole-value cleanup captures against ordinary whole-value
  use after payload consume.
- Preserve direct payload-alias and sibling-payload controls to separate the
  bug from intended alias and sibling-path behavior.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S081-001 | Graphify/source navigation for payload cleanup captures | `mcp__graphify__.query_graph ... defer cleanup captures enum payload optional payload consume ownership paths ...`; source reads in `compiler/tests/safety/diagnostics/core/safety_diagnostics_test.go`, `compiler/tests/ownership/ownership_test.go`, `docs/spec/runtime/ownership_v1.md`, and `docs/spec/core/current_supported_surface.md` | Found stable ordinary diagnostics for enum/optional payload consume and whole-value rejection, plus the same defer capture base-name mechanism from Session 080 | Build payload cleanup probes |
| S081-002 | Immediate payload whole-value controls | `go run ./cli/cmd/tetra check .../immediate_enum_whole_after_payload_consume_rejected_control.tetra`; `go run ./cli/cmd/tetra check .../immediate_optional_whole_after_payload_consume_rejected_control.tetra` | Rejected with `cannot use consumed value 'msg.$case0.payload0'` and `cannot use consumed value 'maybe.$elem'` | Baseline ordinary payload ownership works |
| S081-003 | Direct payload-alias deferred capture control | `go run ./cli/cmd/tetra check .../defer_captures_enum_payload_alias_then_payload_consumed_rejected_control.tetra` | Rejected with `defer cleanup captures value 'left' ... but it was consumed ... before cleanup ran` | Direct alias capture is protected |
| S081-004 | Deferred whole enum after payload consume | `go run ./cli/cmd/tetra check .../defer_captures_enum_whole_then_payload_consumed_repro.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_enum_whole_then_payload_consumed_repro.tetra`; `go run ./cli/cmd/tetra build -o .../defer_captures_enum_whole_then_payload_consumed_repro.bin ...` | Passed `check`; `run` printed `enumexit status 42`; build succeeded | Extends BUG-063 to enum payload descendants |
| S081-005 | Deferred whole optional after payload consume | `go run ./cli/cmd/tetra check .../defer_captures_optional_whole_then_payload_consumed_repro.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_optional_whole_then_payload_consumed_repro.tetra`; `go run ./cli/cmd/tetra build -o .../defer_captures_optional_whole_then_payload_consumed_repro.bin ...` | Passed `check`; `run` printed `optionalexit status 42`; build succeeded | Extends BUG-063 to optional payload descendants |
| S081-006 | Deferred sibling-payload control | `go run ./cli/cmd/tetra check .../defer_captures_enum_sibling_payload_control.tetra`; `go run ./cli/cmd/tetra run .../defer_captures_enum_sibling_payload_control.tetra` | Passed `check`; `run` printed `siblingexit status 42` | Confirms sibling-payload cleanup remains valid |

### 2026-05-18 - Session 082 - Deferred Cleanup Late Privacy Taint

Planned focus:

- Check whether deferred cleanup validates secret/privacy taint only at
  registration time.
- Compare late-tainted captured locals against direct global-store privacy
  sinks and already-tainted deferred captures.
- Verify runtime behavior to distinguish live cleanup reads from registration
  snapshots.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S082-001 | Graphify/source navigation for defer privacy taint | `mcp__graphify__.query_graph ... defer body privacy secret taint checkDeferBody secretTaint AssignStmt PrintStmt exported return secret.i32 unseal privacy analysis restoreSecretTaint`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, and existing privacy bug notes | Found `checkDeferBody()` snapshots and restores `analysis.secretTaint`, `AssignStmt` can taint a local later, global stores reject secret-tainted values only when seen during statement checking, and lowering emits stored defer bodies at scope exit | Build late-taint cleanup probes |
| S082-002 | Direct and already-tainted global-store controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_secret_global_store_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../defer_already_secret_global_store_rejected_control.tetra` | Both rejected with `secret-tainted value cannot be stored in global 'leaked'` | Baseline privacy sink and defer-body registration checks work when taint is visible |
| S082-003 | Late-tainted deferred global-store repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../defer_late_secret_global_store_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../defer_late_secret_global_store_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o .../defer_late_secret_global_store_repro .../defer_late_secret_global_store_repro.tetra` | Passed `check`; `run` printed `exit status 42`; build succeeded | Confirms BUG-064: cleanup uses stale registration-time taint and writes later secret-derived value to global |
| S082-004 | Public deferred global-store control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../defer_public_global_store_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../defer_public_global_store_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o .../defer_public_global_store_control .../defer_public_global_store_control.tetra` | Passed `check`; `run` printed `exit status 42`; build succeeded | Confirms ordinary public deferred global stores remain valid |

### 2026-05-18 - Session 083 - Actor/Task Constant Global Writes

Planned focus:

- Check whether actor/task worker boundary diagnostics treat plain mutable
  global writes the same as read-modify-write global access.
- Compare constant writes against existing read-write controls that should be
  rejected by `TouchesMutableGlobals`.
- Verify runtime mutation for both task and actor workers.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S083-001 | Graphify/source navigation for mutable-global worker boundaries | `mcp__graphify__.query_graph ... mutable global state task_spawn_i32 worker assignment global constant touchesMutableGlobals analysis.touchesMutableGlobals AssignStmt IdentExpr global write`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/tests/ownership/actor_task/actor_task_ownership_test.go`, and existing actor/task bug notes | Found spawn checks use `targetSig.TouchesMutableGlobals`; reads mark `analysis.touchesMutableGlobals`; plain global assignment does not set it except for function-typed globals | Build constant-write probes |
| S083-002 | Task worker constant global write | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../task_worker_constant_global_write_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../task_worker_constant_global_write_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o .../task_worker_constant_global_write_repro .../task_worker_constant_global_write_repro.tetra` | Passed `check`; `run` printed `exit status 42`; build succeeded | Confirms BUG-065 for `core.task_spawn_i32` |
| S083-003 | Task controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../task_worker_read_write_global_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../task_worker_public_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../task_worker_public_control.tetra` | Read-write control rejected with `task_spawn_i32 target 'worker' touches mutable global state`; public control passed and printed `exit status 42` | Baseline task boundary and harness are valid |
| S083-004 | Actor worker constant global write | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../actor_worker_constant_global_write_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../actor_worker_constant_global_write_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -o .../actor_worker_constant_global_write_repro .../actor_worker_constant_global_write_repro.tetra` | Passed `check`; `run` printed `exit status 42`; build succeeded | Extends BUG-065 to `core.spawn` |
| S083-005 | Actor read-write control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../actor_worker_read_write_global_rejected_control.tetra` | Rejected with `spawn target 'worker' touches mutable global state and cannot cross actor boundary` | Confirms actor boundary catches the read path but misses write-only path |

### 2026-05-18 - Session 084 - Exported Async ABI Boundary

Planned focus:

- Check whether `@export` accepts async functions despite source-level async
  call restrictions.
- Compare against bare async call and task-spawn async-target rejection.
- Inspect object symbols to see whether asyncness survives the native boundary.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S084-001 | Graphify/source navigation for exported async functions | `mcp__graphify__.query_graph ... @export async function ABI async FuncSig Async export validation lowering symbol native boundary`; source reads in `compiler/internal/frontend/frontend_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/format/tobj/object.go`, and async/export tests | Found parser allows `@export` before `async func`; checker stores and enforces `Async` for source calls/spawns but export validation does not reject async; TOBJ symbols have only slot metadata | Build exported async probes |
| S084-002 | Exported async function repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_async_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_async_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_async_function_repro.tobj .../export_async_function_repro.tetra` | Passed `check`; `run` printed `exit status 42`; object build succeeded | Confirms BUG-066 acceptance and artifact emission |
| S084-003 | Async source controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../async_bare_call_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../task_spawn_async_target_rejected_control.tetra` | Rejected with `call to async function 'async_answer' requires await` and `task_spawn_i32 target must be synchronous` | Baseline source-level async restrictions work |
| S084-004 | Sync export control and symbol comparison | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_sync_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_sync_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_sync_function_control.tobj .../export_sync_function_control.tetra`; `rg -a -n "ffi_async_answer|async_answer|ffi_sync_answer|sync_answer" .../export_async_function_repro.tobj .../export_sync_function_control.tobj` | Sync control passed, ran with `exit status 42`, and built; object grep shows `ffi_async_answer`/`async_answer` and `ffi_sync_answer`/`sync_answer`, with both exported symbols represented as ordinary slot signatures | Shows async export is indistinguishable from sync export at TOBJ metadata level |

### 2026-05-18 - Session 085 - Exported Budget Caller-Context ABI

Planned focus:

- Check whether `@export` preserves `budget(N)` caller-context requirements.
- Compare direct missing/underbudget call diagnostics against exported object
  emission.
- Inspect TOBJ symbol metadata for budget/effect fields.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S085-001 | Graphify/source navigation for exported budgeted functions | `mcp__graphify__.query_graph ... @export effect clauses uses io runtime budget realtime noalloc noblock semantic clauses ABI metadata TOBJ Effects FuncSig`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/ir/ir.go`, and `compiler/internal/format/tobj/object.go` | Found `validateBudgetContextEdge()` rejects direct calls into budgeted functions without sufficient caller budget; `IRPolicy` carries budget internally; TOBJ `Symbol` has only name/offset/signature slot counts | Build exported budget probes |
| S085-002 | Direct budget caller controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_missing_budget_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_underbudget_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_budgeted_call_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../direct_budgeted_call_control.tetra` | Missing-budget and underbudget controls were rejected with required-budget diagnostics; covered caller passed and ran with `exit status 42` | Baseline source-level budget caller checks work |
| S085-003 | Exported budgeted function repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_budgeted_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_budgeted_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_budgeted_function_repro.tobj .../export_budgeted_function_repro.tetra`; `rg -a -n "ffi_budgeted_answer|budgeted_answer|ffi_plain_answer|plain_answer" ...` | Exported budgeted function passed `check`, ran with `exit status 42`, built a TOBJ object, and emitted `ffi_budgeted_answer` / `budgeted_answer` as ordinary symbols | Confirms BUG-067 acceptance and artifact emission |
| S085-004 | Plain export control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_plain_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_plain_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_plain_function_control.tobj .../export_plain_function_control.tetra` | Plain export passed, ran with `exit status 42`, built a TOBJ object, and emitted `ffi_plain_answer` / `plain_answer` | Control shows the budgeted export uses the same exported-symbol surface as an ordinary function |

### 2026-05-18 - Session 086 - Exported Effect Metadata ABI

Planned focus:

- Check whether `@export` preserves ordinary `uses` effect requirements.
- Compare a direct missing-`uses io` caller diagnostic against exported object
  emission.
- Inspect TOBJ symbol metadata for effect fields.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S086-001 | Graphify/source navigation for exported effects | `mcp__graphify__.query_graph ... @export semantic clauses noalloc noblock realtime uses io network fs unsafe ABI metadata FuncSig Effects validateExportedOpaqueABISignature TOBJ Symbol`; source reads in `docs/spec/runtime/effects_capabilities_privacy_v1.md`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/semantics/semantics_memory_resources.go`, `compiler/internal/backend/x64obj/builder.go`, and `compiler/internal/format/tobj/object.go` | Found source calls propagate `FuncSig.Effects` through `effectContext.require()`, export validation does not reject/preserve `fn.Uses`, and TOBJ `Symbol` has only name/offset/signature slot counts | Build exported effect probes |
| S086-002 | Direct effect caller controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_missing_uses_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../direct_with_uses_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../direct_with_uses_control.tetra` | Missing-uses caller was rejected with `function 'main' uses effect 'io' but does not declare it`; covered caller passed, printed `direct`, and exited with `exit status 42` | Baseline source-level effect propagation works |
| S086-003 | Exported `uses io` function repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_effectful_io_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_effectful_io_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_effectful_io_repro.tobj .../export_effectful_io_repro.tetra`; `rg -a -n "ffi_log_answer|log_answer|ffi_plain_answer|plain_answer" ...` | Exported effectful function passed `check`, ran with `exit status 42`, built a TOBJ object, and emitted `ffi_log_answer` / `log_answer` as ordinary symbols | Confirms BUG-068 acceptance and artifact emission |
| S086-004 | Pure export control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_pure_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_pure_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_pure_function_control.tobj .../export_pure_function_control.tetra` | Pure export passed, ran with `exit status 42`, built a TOBJ object, and emitted `ffi_plain_answer` / `plain_answer` | Control shows the effectful export has no distinct exported-symbol metadata surface |

### 2026-05-18 - Session 087 - Export Name Collision Symbol Rebinding

Planned focus:

- Check whether `@export` names are validated against ordinary emitted function
  symbols, not only other export aliases.
- Compare declaration-order behavior for alias/internal-name collisions.
- Parse TOBJ symbol offsets to verify which function the published name points
  to.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S087-001 | Graphify/source navigation for export-name collisions | `mcp__graphify__.query_graph ... @export duplicate exported symbol internal function name ExportName duplicate exported symbol validate export name builder symbolOffsets main TOBJ check build fails`; source reads in `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/backend/x64obj/builder.go`, `compiler/internal/format/tobj/object.go`, and `compiler/internal/backend/x64obj/builder_test.go` | Found checker tracks duplicate export aliases only; object builder checks export aliases against already-seen symbols but writes function-name symbols without collision checks | Build declaration-order probes |
| S087-002 | Later function name collision repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_name_collides_later_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../export_name_collides_later_function_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_name_collides_later_function_repro.tobj ...` | Passed `check`, ran with `exit status 42`, and built a TOBJ object | Confirms the collision is not rejected when the ordinary function appears later |
| S087-003 | Prior function name collision control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_name_collides_prior_function_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_name_collides_prior_function_control.tobj ...` | `check` passed, but object build failed with `duplicate exported symbol 'target'` | Shows the validation gap is order-dependent and deferred past semantics |
| S087-004 | Non-colliding export control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../non_colliding_export_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra run .../non_colliding_export_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../non_colliding_export_control.tobj ...` | Passed `check`, ran with `exit status 42`, and built a TOBJ object | Baseline alias emission works when the alias is unique |
| S087-005 | TOBJ symbol offset comparison | `python -c '<TOBJ symbol parser>' .../export_name_collides_later_function_repro.tobj .../non_colliding_export_control.tobj` | Repro symbols: `exported_answer offset=0`, `target offset=13`, `main offset=26`; control symbols: `exported_answer offset=0`, `ffi_exported_answer offset=0`, `target offset=13`, `main offset=26` | Confirms BUG-069: colliding export alias `target` was silently rebound to the later ordinary `target` function |

### 2026-05-18 - Session 088 - Export Symbol Name Grammar

Planned focus:

- Check whether `@export` validates native symbol names beyond non-empty and
  reserved namespaces.
- Compare whitespace/control-character names against the empty-name control.
- Parse TOBJ symbols using `repr(name)` so invisible characters are visible in
  evidence.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S088-001 | Graphify/source navigation for export symbol grammar | `mcp__graphify__.query_graph ... @export export name validation native symbol grammar empty string whitespace newline nul object symbol validateSymbolRecord writeString linker symbol name`; source reads in `compiler/internal/frontend/frontend_core.go`, `compiler/internal/semantics/semantics_checker.go`, `compiler/internal/format/tobj/object.go`, and export/object tests | Found parser rejects only empty export names; checker handles reserved namespaces and duplicate aliases; TOBJ validates only non-empty names | Build malformed-name probes |
| S088-002 | Space export-name repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_space_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_space_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `name='ffi log' offset=0 params=0 returns=1` | Confirms BUG-070 for whitespace export names |
| S088-003 | Newline export-name repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_newline_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_newline_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `name='ffi\nlog' offset=0 params=0 returns=1` | Extends BUG-070 to control-character export names |
| S088-004 | Tab export-name repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_tab_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_tab_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `name='ffi\tlog' offset=0 params=0 returns=1` | Extends BUG-070 to tab export names |
| S088-005 | Empty and valid identifier controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_empty_name_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_identifier_name_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_identifier_name_control.tobj ...` | Empty name rejected with `@export name must not be empty`; valid identifier passed and emitted `name='ffi_log' offset=0 params=0 returns=1` | Confirms existing validation is present but too narrow |

### 2026-05-18 - Session 089 - core.sym_addr Symbol Name Grammar

Planned focus:

- Check whether `core.sym_addr` validates native symbol names beyond non-empty.
- Compare malformed whitespace/control names against empty and identifier
  controls.
- Parse TOBJ relocations using `repr(name)` so invisible characters are visible
  in evidence.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S089-001 | Graphify/source navigation for `core.sym_addr` native names | `mcp__graphify__.query_graph ... core.sym_addr symbol name validation empty reserved __tetra internal runtime whitespace newline native symbol unsafe link effect checker lower builtin sym_addr`; source reads in `docs/spec/runtime/unsafe.md`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/backend/x64core/x64core_core.go`, `compiler/internal/backend/x64obj/builder.go`, and `compiler/internal/format/tobj/object.go` | Found `core.sym_addr` is always unsafe with `link`, checker/lower validate only non-empty strings, x64 emits `CallPatch`, and TOBJ reloc validation rejects only empty relocation names | Build malformed relocation-name probes |
| S089-002 | Space `core.sym_addr` repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_space_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_space_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `reloc kind=1 at=25 name='ffi log' addend=0` | Confirms BUG-071 for whitespace relocation names |
| S089-003 | Newline `core.sym_addr` repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_newline_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_newline_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `reloc kind=1 at=25 name='ffi\nlog' addend=0` | Extends BUG-071 to control-character relocation names |
| S089-004 | Tab `core.sym_addr` repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_tab_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_tab_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `reloc kind=1 at=25 name='ffi\tlog' addend=0` | Extends BUG-071 to tab relocation names |
| S089-005 | Empty and valid identifier controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_empty_name_rejected_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_empty_name_rejected_control.tobj ...`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_identifier_name_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_identifier_name_control.tobj ...` | Empty name rejected with `sym_addr expects a non-empty symbol name`; valid identifier passed and emitted `reloc kind=1 at=25 name='ffi_target' addend=0` | Confirms existing validation is present but too narrow |

### 2026-05-18 - Session 090 - Embedded NUL Native Symbol Names

Planned focus:

- Check whether raw `0x00` bytes can enter native symbol names separately from
  printable whitespace/control-character probes.
- Cover both `@export` symbol table entries and `core.sym_addr` relocation
  entries.
- Parse TOBJ artifacts with both `repr(name)` and raw hex output.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S090-001 | Graphify/source navigation for string literal and symbol serialization | `mcp__graphify__.query_graph ... lexer string literal raw NUL byte export name sym_addr TOBJ symbol relocation native C string truncation validation`; `mcp__graphify__.get_neighbors readString`; source reads in `compiler/internal/frontend/frontend_core.go`, `compiler/internal/frontend/frontend_core.go`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_core.go`, and `compiler/internal/format/tobj/object.go` | Found lexer accepts valid UTF-8 source and appends raw non-escape bytes to string literals; export/sym_addr/object validation reject only empty names | Build raw-NUL repros |
| S090-002 | Create raw-NUL repro files | `python -c '<write repro files with raw 0x00 inside @export and core.sym_addr string literals>'` | Created `/tmp/tetra-bug-hunt/session-090/bughunt/export_nul_name_repro.tetra` and `/tmp/tetra-bug-hunt/session-090/bughunt/sym_addr_nul_name_repro.tetra`; Python reported `contains_nul=True` for both | Run check/build |
| S090-003 | Raw-NUL `@export` repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_nul_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../export_nul_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `symbol name='ffi\x00log' hex=666669006c6f67 offset=0 params=0 returns=1` | Confirms BUG-072 for exported symbols |
| S090-004 | Raw-NUL `core.sym_addr` repro | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_nul_name_repro.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../sym_addr_nul_name_repro.tobj ...` | Passed `check` and built; TOBJ parser reported `reloc kind=1 at=25 name='ffi\x00log' hex=666669006c6f67 addend=0` | Confirms BUG-072 for relocation names |
| S090-005 | Identifier controls | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../export_identifier_name_control.tetra`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../sym_addr_identifier_name_control.tetra`; object builds and TOBJ parser over both control objects | Both controls passed and built; parser reported `ffi_log` and `ffi_target` names without embedded `00` bytes | Confirms artifact parser is showing a real byte-level difference |

### 2026-05-18 - Session 091 - WASM Pure Service Import Surface

Planned focus:

- Check whether a pure WASM-targeted microservice imports host I/O functions
  without `uses io` or `print`.
- Compare pure service artifacts against a print control that legitimately
  needs host output.
- Verify whether `validate-wasm-imports` detects only target allowlists or also
  catches expanded effect/import surfaces.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S091-001 | Graphify/source navigation for WASM host import policy | `mcp__graphify__.query_graph ... wasm32-wasi wasm32-web imports Allowed imports validate-wasm-imports tetra_web_v1 wasi_snapshot_preview1 proc_exit fd_write console_log panic`; source reads in `docs/backend/wasm_architecture.md`, `compiler/internal/backend/wasm32_wasi/codegen.go`, `compiler/internal/backend/wasm32_web/codegen.go`, and `tools/cmd/validate-wasm-imports/main.go` | Found policy requires host-boundary validation and effect-gated host access; both backends appear to write fixed import sections, while validator checks allowlist membership only | Build pure/control WASM artifacts |
| S091-002 | Pure service and print control setup | Scratch files under `/tmp/tetra-bug-hunt/session-091/bughunt`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../wasm_pure_service_repro.tetra`; `... check .../wasm_print_service_control.tetra` | Pure service has only `func main() -> Int: return 42`; both pure and print-control sources passed `check` | Build both targets |
| S091-003 | Pure WASI import surface | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o .../wasm_pure_service_repro.wasi.wasm ...`; `GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi .../wasm_pure_service_repro.wasi.wasm`; Python WASM parser | Build passed and validator exited 0; parser reported `import wasi_snapshot_preview1.fd_write`, `import wasi_snapshot_preview1.proc_exit`, and `call_indexes=2,1`, so unused `fd_write` is still in the pure artifact import section | Confirms BUG-073 on WASI |
| S091-004 | Pure Web import surface | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o .../wasm_pure_service_repro.web.wasm ...`; `GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-web .../wasm_pure_service_repro.web.wasm`; Python WASM parser | Build passed and validator exited 0; parser reported `import tetra_web_v1.console_log`, `import tetra_web_v1.panic`, and empty `call_indexes=`, so both host imports are unused in the pure artifact | Confirms BUG-073 on Web |
| S091-005 | Print control comparison | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o .../wasm_print_service_control.wasi.wasm ...`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o .../wasm_print_service_control.web.wasm ...`; Python WASM parser | Print control imports the same surfaces, but call indexes include the output imports: WASI `call_indexes=0,2,1`; Web `call_indexes=0` | Shows the parser distinguishes legitimately used host output from the pure-service import leak |

### 2026-05-18 - Session 092 - WASM `@export` Service Endpoint Surface

Planned focus:

- Check whether `@export` on a service function survives into WASM export
  sections or is rejected for unsupported WASM entry artifacts.
- Compare `@export` repro artifacts against entry-only controls.
- Compare WASM output against native TOBJ emission from the same source.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S092-001 | Graphify/source navigation for WASM export sections | `mcp__graphify__.query_graph ... wasm32 web wasi export section @export ExportName tetra service exported functions wasm exports _start tetra_main memory backend codegen`; source reads in `docs/backend/wasm_architecture.md`, `compiler/internal/backend/wasm32_wasi/codegen.go`, `compiler/internal/backend/wasm32_web/codegen.go`, and import/export shape tests | Found docs define deterministic WOBJ exports and fixed entry exports; WASI/Web backends hard-code `memory` plus `_start`/`tetra_main`; tests assert only fixed entry exports | Build `@export` repros |
| S092-002 | `@export` repro setup and check | Scratch files under `/tmp/tetra-bug-hunt/session-092/bughunt`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../wasm_exported_service_repro.tetra`; `... check .../wasm_entry_only_control.tetra` | `@export("service_answer") func answer() -> Int` passed `check`; entry-only control also passed | Build artifacts |
| S092-003 | WASI `@export` artifact | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o .../wasm_exported_service_repro.wasi.wasm ...`; Python WASM export parser | Build passed; exports were only `memory` and `_start`, with no `service_answer`; entry-only WASI control had the same export list | Confirms BUG-074 on WASI |
| S092-004 | Web `@export` artifact | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o .../wasm_exported_service_repro.web.wasm ...`; Python WASM export parser | Build passed; exports were only `memory` and `tetra_main`, with no `service_answer`; entry-only Web control had the same export list | Confirms BUG-074 on Web |
| S092-005 | Native object contrast | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../wasm_exported_service_repro.native.tobj ...`; Python TOBJ symbol parser | Native object emitted `symbol 'service_answer' offset=0 params=0 returns=1` | Shows the attribute exists before the WASM backend drops it |

### 2026-05-18 - Session 093 - WASM `core.sym_addr` Link Token Boundary

Planned focus:

- Check whether `core.sym_addr("missing_external")` is rejected on WASM or
  represented as an explicit import/export/link dependency.
- Compare missing external symbols against an internal function control and the
  empty-name validation control.
- Contrast WASM token lowering with native TOBJ relocation and native
  executable linking.

Evidence:

| ID | Probe | Command | Result | Follow-up |
| --- | --- | --- | --- | --- |
| S093-001 | Graphify/source navigation for WASM `core.sym_addr` | `mcp__graphify__.query_graph ... WASM core.sym_addr IRSymAddr symbol address lowering unsafe link effect wasm32-wasi wasm32-web unsupported policy function index external symbols`; `mcp__graphify__.get_neighbors IRSymAddr`; `mcp__graphify__.shortest_path IRSymAddr wasm32_web`; source reads in `docs/backend/wasm_architecture.md`, `docs/spec/runtime/unsafe.md`, `compiler/internal/semantics/semantics_expressions.go`, `compiler/internal/lower/lower_core.go`, `compiler/internal/backend/wasm32_wasi/codegen.go`, `compiler/internal/backend/wasm32_web/codegen.go`, and `compiler/internal/backend/x64core/x64core_core.go` | Found docs say unresolved symbols are compile errors except configured host imports; `core.sym_addr` is unsafe with `link`; semantics/lower validate only non-empty string literals; WASM backends lower `IRSymAddr` to FNV token `i32.const`; native x64 preserves named relocations | Build missing-external repros |
| S093-002 | Repro/control setup and `check` | Scratch files under `/tmp/tetra-bug-hunt/session-093/bughunt`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra check .../wasm_sym_addr_missing_external_repro.tetra`; checks for internal, empty, and no-sym controls | Missing-external repro passed `check`; internal and no-sym controls passed; empty control rejected with `sym_addr expects a non-empty symbol name` | Confirms the issue is not empty-name validation |
| S093-003 | WASM missing-external artifacts | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o .../out/missing_external.wasi.wasm ...`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o .../out/missing_external.web.wasm ...`; `python .../inspect_wasm.py ...`; `GOCACHE=/tmp/tetra-go-cache go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi ...`; `... --target wasm32-web ...` | Both builds passed and both validators exited 0. Parser found no raw `missing_external` bytes, no import/export for that symbol, and `i32.const -1981042209/2313925087` in each artifact | Confirms BUG-075 on WASI and Web |
| S093-004 | Internal-function token control | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-wasi -o .../out/internal_control.wasi.wasm ...`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target wasm32-web -o .../out/internal_control.web.wasm ...`; `python .../inspect_wasm.py ...` | Both internal controls built; parser found no raw `callback` bytes and emitted token `-2014301178/2280666118` for `core.sym_addr("callback")` while the function itself returned `7` normally | Shows token lowering is the general WASM mechanism, not a parser artifact |
| S093-005 | Native object and executable contrast | `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build -emit object -o .../out/missing_external.native.tobj ...`; `python .../inspect_tobj.py ...`; `GOCACHE=/tmp/tetra-go-cache go run ./cli/cmd/tetra build --target linux-x64 -o .../out/missing_external.native.exe ...` | Native object emitted `reloc kind=1 at=25 name='missing_external' addend=0`; native executable build failed with `unresolved symbol 'missing_external'` | Confirms WASM uniquely erases the unresolved link dependency into an anonymous token |
