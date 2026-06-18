# Linux-x64 Memory, Parallelism, And UI Production Design

Status: proposed design for the active post-`v0.4.0` production line.

Scope: Linux x64 first. Non-Linux runtime promotion, WASM/browser production UI,
EcoNet, and full v1.0 language guarantees remain separate future scope unless a
later decision explicitly promotes them.

## Objective

Finish Tetra's next production line in three ordered layers:

1. Memory Production Core.
2. Parallelism Production Core.
3. UI Production Runtime.

The ordering is intentional. UI state, event dispatch, async commands, and
runtime lifecycle rely on predictable memory ownership and scheduler behavior.
Parallelism safety also depends on the memory model before it can claim stable
actor/task/thread boundaries.

## Observed Repository Facts

- Current release truth is `v0.4.0` Linux-x64 scoped production, recorded in
  `docs/spec/current_supported_surface.md` and `docs/spec/v0_4_scope.md`.
- Current UI surface lives in `docs/spec/ui_v1.md`,
  `compiler/internal/semantics/ui.go`, `compiler/internal/lower/ui.go`,
  `compiler/internal/backend/native_shell/codegen.go`,
  `tools/cmd/native-ui-runtime-smoke/main.go`, and
  `tools/validators/nativeui/report.go`.
- Current UI evidence is a metadata/native-shell/native-runtime slice. It
  validates widget instances, click dispatch, state updates, negative cases,
  and close lifecycle, but does not yet claim a full desktop toolkit backend,
  rich layout, broad input, or platform accessibility integration.
- Current ownership and lifetime rules are centered around
  `compiler/internal/semantics/region.go`, documented in
  `docs/spec/ownership_v1.md`, and covered by ownership tests under
  `compiler/tests/ownership/` plus CLI diagnostic tests.
- Current memory helpers are capability-bound in `lib/core/memory.tetra`.
  They use `cap.mem`, `unsafe`, and raw pointer builtins for `memset_u8` and
  `memcpy_u8`; they do not by themselves define a complete allocator or broad
  runtime bounds contract.
- Current actor/task runtime ABI is documented in `docs/spec/runtime_abi.md`
  and `docs/spec/actors.md`, with Linux-x64 built-in runtime code in
  `compiler/internal/actorsrt/linux_x64.go` and
  `compiler/internal/actorsrt/linux_x64_emit.go`.
- Current parallelism is a cooperative single-thread scheduler with tasks,
  groups, deadlines, select/poll paths, actor mailboxes, and Linux-x64
  distributed actor smoke evidence. It explicitly does not claim
  multi-threaded scheduling, full structured concurrency, broad race-safety
  proofs, or complete backpressure/capacity handling.
- Current release evidence patterns use scripts and validators such as
  `scripts/release/v0_4_0/gate.sh`,
  `scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh`,
  `scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh`,
  `tools/cmd/validate-native-ui-runtime`, and
  `tools/cmd/validate-distributed-actor-runtime`.

## Design Principles

- Production means executable Linux-x64 behavior with tests, docs, release-gate
  evidence, and negative validators. Metadata-only, docs-only, mock, fake,
  placeholder, or build-only evidence is not enough.
- Compile-time safety should reject unsupported behavior with stable
  diagnostics instead of allowing unsound runtime behavior.
- Runtime checks should exist where compile-time proof is not available and the
  runtime can report a deterministic failure boundary.
- Capacity limits are allowed only when documented, checked or reported, and
  covered by negative evidence.
- Existing syntax and CLI workflows should remain backward-compatible unless a
  documented diagnostic migration is required for safety.

## Layer 1: Memory Production Core

### Target Contract

Memory production support means Tetra has a coherent Linux-x64 memory model for
safe code, unsafe code, and capability-bound raw memory helpers.

The promoted surface must cover:

- allocator/runtime memory model with deterministic allocation and failure
  semantics;
- ownership/borrow/consume escape rules for heap values, slices, structs,
  enum payloads, closures, and actor/task transfers;
- explicit `unsafe`, `cap.mem`, raw pointer, `memcpy_u8`, and `memset_u8`
  contracts;
- runtime bounds checks or stable diagnostics where static checking cannot
  prove safety;
- use-after-free, double-free, borrow escape, aliasing, and transfer stress
  coverage;
- user documentation for safe memory patterns and forbidden unsafe patterns.

### Architecture

- Extend the existing region/resource model in
  `compiler/internal/semantics/region.go` instead of introducing a separate
  checker path.
- Keep `lib/core/memory.tetra` as the stable public helper surface, but make
  its safety contract explicit in docs and tests.
- Add runtime ABI functions only after their symbols, signatures, and target
  behavior are documented in `docs/spec/runtime_abi.md`.
- Add validators for memory production evidence so release gates can reject
  report-only or incomplete memory claims.

### Key Risks

- Heap and closure escape rules can accidentally weaken the existing local
  lifetime solver.
- Raw memory bounds checks can become target-specific and drift from the
  documented ABI.
- Actor/task transfer safety can duplicate memory rules unless region/resource
  summaries stay the source of truth.

## Layer 2: Parallelism Production Core

### Target Contract

Parallelism production support means tasks and actors are stable enough for
Linux-x64 server-like and interactive programs under a documented scheduler and
safety boundary.

The promoted surface must cover:

- production task scheduler semantics;
- task join, cancel, deadline, select, poll, and task-group lifecycle;
- actor mailbox capacity, backpressure, failure handling, and invalid handle
  behavior;
- transfer rules across task, actor, and future thread boundaries;
- race-safety via a conservative compile-time model and explicit rejection of
  unsupported shared mutable behavior;
- stress evidence for many tasks, many actor messages, cancellation storms,
  timeouts, and failure paths;
- documentation that separates safe, unsafe, and forbidden parallel patterns.

### Architecture

- Build on the existing cooperative runtime in
  `compiler/internal/actorsrt/linux_x64.go` and
  `compiler/internal/actorsrt/linux_x64_emit.go`.
- Preserve the current actor/task builtins and add stricter diagnostics before
  changing runtime behavior.
- Treat mailbox and task-group capacity as first-class reportable behavior,
  not unspecified overflow.
- Keep distributed actor production evidence Linux-x64 only until separate
  non-Linux runtime reports exist.

### Key Risks

- Current fixed-capacity message pool behavior is documented as unspecified
  after overflow; production backpressure requires changing that boundary.
- Scheduler behavior must remain deterministic enough for smoke/stress gates.
- Race-safety can become too permissive if callable/global/mutable-state checks
  do not compose with task and actor entrypoint checks.

## Layer 3: UI Production Runtime

### Target Contract

UI production support means a real Linux-x64 desktop app can be written in
Tetra and run through a production runtime, not just metadata and sidecar
validation.

The promoted surface must cover:

- window lifecycle;
- layout system;
- buttons, text, input, lists, panels, and state binding;
- event loop;
- async UI commands;
- timers;
- redraw/update model;
- error and crash handling;
- dogfood applications that use the runtime directly;
- UI release gate that rejects mock, placeholder, metadata-only,
  native-shell-only, and runtime-less evidence.

### Architecture

- Keep `tetra.ui.v1` metadata as the compiler-to-runtime contract.
- Evolve `tools/cmd/native-ui-runtime-smoke` from a smoke harness into the
  acceptance model for the production runtime, while keeping validators strict.
- Add a Linux-x64 runtime boundary that can execute a real event loop and
  window lifecycle. The backend choice is still an implementation decision, but
  it must be deterministic in CI and not rely on fake widget reports.
- Extend UI lowering only after the runtime contract names how controls,
  bindings, and command results are represented.

### Key Risks

- A full desktop toolkit backend can make CI flaky unless the gate has a
  deterministic headless or controlled Linux path.
- Async UI commands can violate memory and parallelism safety unless they use
  the completed task/actor transfer rules.
- Layout and input can become too large; the first production surface should be
  narrow but real.

## Release Evidence Model

Each layer gets its own evidence artifacts before the combined gate can pass:

- `tetra.memory.production.v1`
- `tetra.parallel.production.v1`
- `tetra.ui.desktop-runtime.v1`

Each artifact must have:

- executable Linux-x64 positive cases;
- negative cases for unsupported behavior;
- strict validator rejection of fake, mock, placeholder, docs-only, and
  metadata-only evidence;
- source examples under `examples/` or `examples/projects/`;
- docs and release notes;
- artifact hashes and clean release-state evidence.

## Completion Criteria

The active goal is complete only when all three layers have:

- real implementation merged into the Linux-x64 path;
- stable diagnostics for unsupported or unsafe behavior;
- user-facing documentation;
- examples and dogfood projects;
- smoke, stress, and fuzz or fuzz-like deterministic regression tests;
- release gate evidence from a clean snapshot;
- completion audit mapping every objective requirement to concrete artifacts.

