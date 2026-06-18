# Tetra Memory Model vNext Current-State Audit

Status: current-state audit for
`docs/plans/2026-06/actors-memory/2026-06-13-tetra-memory-model-vnext.md` Task 1.

This audit records what the repository currently proves before the vNext memory work starts. It is
deliberately conservative: implemented behavior, report-only evidence, prototype evidence, planned
vocabulary, and nonclaims are separate categories.

Graphify context was consulted first through `graphify-out/GRAPH_REPORT.md` and `graphify query`,
but the graph was built from commit `95bfd4a8`; concrete file inspection is the authority for the
classifications below.

## Classification Key

- `implemented`: compiler/runtime/tooling behavior exists in current source and has direct tests or
  validators.
- `report-only`: report schemas, validators, or release artifacts exist, but the row is not itself a
  runtime behavior claim.
- `prototype-evidence`: executable or model evidence exists for a future shape, but the production
  runtime is not promoted by that evidence.
- `planned-vocabulary`: names, storage classes, blockers, or acceptance language exist before the
  full runtime path is proven.
- `nonclaim`: the repository explicitly rejects or withholds the broader claim.

## Current Implemented Surface

### Effects and policy

- Classification: `implemented`.
- Evidence:
  `docs/spec/runtime/effects_capabilities_privacy_v1.md` defines canonical effects,
  memory, policy, runtime groups, capability checks, unsafe policy, privacy, and local
  budget semantics.
- Boundary:
  budget is a deterministic local IR/cross-edge guardrail, not runtime-wide or
  distributed resource accounting.

### Ownership and lifetimes

- Classification: `implemented`.
- Evidence:
  `docs/spec/runtime/ownership_v1.md` documents `borrow`, `inout`, `consume`,
  resource lifetime tracking, actor/task transfer diagnostics, and conservative
  branch/loop joins.
- Boundary:
  it remains a local production slice; formal alias/race proofs and distributed actor
  safety are outside the current guarantee.

### Islands

- Classification: `implemented`.
- Evidence:
  `docs/spec/memory/islands.md` defines island handles, scoped islands, region typing,
  bump allocation, bulk free, native runtime paths, and a WASM compile-compatible
  fallback.
- Boundary:
  native island paths are in scope; WASM fallback maps to linear heap/no-op free
  behavior and is not equivalent native memory reclamation.

### Allocation planner

- Classification: `implemented`.
- Evidence:
  `compiler/internal/allocplan` records allocation intent, escape class, planned
  storage, actual lowering storage, runtime path, requested/reserved bytes, allocator
  metadata, and region summaries.
- Boundary:
  `planned_storage` is not a backend lowering claim; `actual_lowering_storage` is the
  current lowering truth.

### Runtime allocation contracts

- Classification: `implemented`.
- Evidence:
  `compiler/internal/runtimeabi/allocation_contract.go`, `small_heap.go`, and
  `region_allocator.go` define allocation paths, 16-byte alignment, small-heap size
  classes, 64 KiB chunks, per-core small-heap ABI, and region alignment.
- Boundary:
  the current ABI model does not yet expose a target-neutral `MemoryBackend` interface
  with reserve/commit/decommit/release/trim/footprint semantics.

### Small safe-slice allocator evidence

- Classification: `implemented` plus `report-only`.
- Evidence:
  `per_core_small_heap` metadata exists in runtime ABI and allocation reports.
  `tools/cmd/memory-production-smoke` classifies syscall reduction as
  `allocation_report_estimate`.
- Boundary:
  the small-heap benchmark is an allocation-report estimate, not RSS, pprof, MemStats,
  `time_v`, or `strace` runtime measurement.

### Function temp regions

- Classification: `implemented` when enabled.
- Evidence:
  `compiler/internal/allocplan` can report `FunctionTempRegion`, and x64 core has
  function-temp region lowering helpers. Tests cover actual lowering when
  `EnableRegionLowering` is enabled and heap fallback when disabled.
- Boundary:
  the planner must still preserve planned-vs-actual truth; not every temporary or
  actor-boundary copy can use a function temp region.

### Explicit islands in allocation reports

- Classification: `implemented`.
- Evidence:
  allocation reports preserve `ExplicitIsland`, region id, lifetime, debug mode, and
  byte alignment evidence. Island proof validators compare planned and actual storage.
- Boundary:
  explicit island evidence is scoped to proven island storage; it is not a generic
  arena allocator for all objects.

### RAM contract compiler

- Classification: `implemented` report/gate layer.
- Evidence:
  `compiler/internal/ramcontract` defines RAM contract reports, memory grades, proof
  summaries, validation pipeline coverage, blockers, placements, intents, escape
  statuses, and validation statuses.
- Boundary:
  the RAM contract projects compiler-owned facts; it does not reconstruct truth from
  JSON reports and does not claim zero heap for all programs.

### Memory production smoke

- Classification: `implemented` scoped gate.
- Evidence:
  `tools/cmd/memory-production-smoke`, `tools/cmd/validate-memory-production`, and
  `tools/validators/memoryprod` validate `tetra.memory.production.v1`, classified
  benchmark evidence, checked examples, fuzz-like memory cases, and
  `ram-measurement.json`.
- Boundary:
  `ram-measurement.json` currently validates MemStats snapshots or blocked status; it
  does not enforce hard RAM/RSS thresholds.

### Current actor runtime

- Classification: `implemented` bounded local runtime.
- Evidence:
  `docs/spec/runtime/actors.md` and `compiler/internal/actorsrt` document fixed actor
  table, mailbox depth, message pool, checked backpressure, message reclamation, done
  actor behavior, and single-thread cooperative scheduling.
- Boundary:
  it is not a full production multi-threaded actor runtime, supervision system,
  unbounded mailbox, or all-target distributed runtime.

### Actor/task memory boundary checks

- Classification: `implemented` narrow safety slice.
- Evidence:
  `compiler/internal/semantics/semantics_memory_resources.go`, ownership tests, and
  actor safety checks reject borrowed/unsafe payloads, stale islands, and use after
  transfer.
- Boundary:
  this is checker/runtime-boundary evidence, not a general distributed ownership
  protocol.

### Local owned-region typed actor move

- Classification: `implemented` narrow local path.
- Evidence:
  `docs/spec/runtime/actors.md`, `docs/design/actor_region_transfer.md`, and actor
  transfer tests record local `zero_copy_move` rows with `bytes_copied: 0` for owned
  island-backed slice moves.
- Boundary:
  the zero-copy guarantee is local typed mailbox owned-region evidence only; it is not
  cross-node pointer or region transfer.

## Report-Only And Gate Evidence

### RAM contract artifacts

- Classification: `report-only`.
- Evidence:
  `ram-contract-report.json`, `memory-grade-report.json`, `proof-store-summary.json`,
  `validation-pipeline-coverage.json`, `heap-blockers.json`, and `copy-blockers.json`
  are defined by `docs/design/ram_contract_compiler.md` and
  `compiler/internal/ramcontract`.
- Boundary:
  these artifacts explain and gate facts; they are not themselves proof of zero heap or
  performance.

### Memory grade and blockers

- Classification: `report-only`.
- Evidence:
  `M0..M6` grades and blocker rows identify heap/copy/unbounded reasons and suggested
  fixes.
- Boundary:
  a grade is a report classification, not a runtime footprint ceiling unless a later
  gate defines one.

### Allocation report estimate

- Classification: `report-only`.
- Evidence:
  the small heap syscall reduction row is explicitly `allocation_report_estimate` with
  method `allocation_report_summary`.
- Boundary:
  it cannot be reused as RSS or runtime measured footprint evidence.

### RAM measurement artifact

- Classification: `report-only` plus scoped runtime capture.
- Evidence:
  `tetra.memory.ram-measurement.v1` captures `runtime.MemStats` snapshots and accepts
  blocked measurement reports.
- Boundary:
  MemStats is a Go runtime snapshot for the smoke tool, not the target program RSS
  model or cross-target footprint contract.

### Actor runtime boundary audit

- Classification: `report-only`.
- Evidence:
  `compiler/internal/actorsrt/actorsrt_core.go` validates current limits, prototype
  features, acceptance requirements, and full-claim blockers.
- Boundary:
  the audit enforces nonclaims; it does not promote blocked future requirements.

### Release scripts

- Classification: `report-only` plus executable gate.
- Evidence:
  `scripts/release/post_v0_4/*` run memory, RAM contract, parallel, and actor
  foundation local gates.
- Boundary:
  GitHub Actions wiring is not part of vNext until explicitly approved.

## Prototype Evidence

### Scheduler model

- Classification: `prototype-evidence`.
- Evidence:
  `compiler/internal/parallelrt/scheduler_model.go` models per-core queues, two-core
  work stealing, bounded typed mailbox metadata, and zero-copy move stats.
- Boundary:
  it is design/release evidence, not production multi-threaded actor scheduling
  behavior.

### Actor benchmark prep rows

- Classification: `prototype-evidence`.
- Evidence:
  actor ping-pong, fanout/fanin, mailbox throughput, backpressure latency, and
  `zero_copy_move` local typed mailbox rows exist as Tier 0/Tier 1 prep.
- Boundary:
  they publish no measured throughput guarantee, no C++/Rust parity, and no official
  benchmark claim.

### Distributed actor Linux-x64 slice

- Classification: `prototype-evidence` plus scoped current slice.
- Evidence:
  Linux-x64 distributed actor runtime smoke covers loopback broker/node status/message
  frames.
- Boundary:
  non-Linux distributed actors, cluster membership, reconnect/retry production
  behavior, and distributed zero-copy remain nonclaims.

## Planned Vocabulary And Partial Concepts

### `ActorMoveRegion`

- Classification: `planned-vocabulary`.
- Evidence:
  `compiler/internal/allocplan` and `compiler/internal/ramcontract` know the storage
  class and validation strings.
- What vNext must prove:
  a real actor-domain runtime transfer must prove ownership movement and domain
  accounting before treating it as implemented.

### `TaskRegion`

- Classification: `planned-vocabulary`.
- Evidence:
  storage/report vocabulary exists and request/task region audits exist.
- What vNext must prove:
  task-domain memory needs stable domain ownership, lifetime, byte accounting, and
  validator coverage.

### Memory domains

- Classification: `planned-vocabulary`.
- Evidence:
  the plan names process/task/actor/island/request domains, but current
  `ramcontract.Row` has no first-class domain fields.
- What vNext must prove:
  add schema/types/projection fields and validators that preserve current RAM grades
  and blockers.

### Memory backend substrate

- Classification: `planned-vocabulary`.
- Evidence:
  allocation paths exist, but `reserve`, `commit`, `decommit`, `release`, `trim`, and
  `footprint` are not a target-neutral ABI.
- What vNext must prove:
  add a target-neutral contract and implementation model with
  measured/estimated/unsupported/blocked evidence classes.

### RSS/footprint thresholds

- Classification: `planned-vocabulary`.
- Evidence:
  some surface validators have optional RSS fields, and memory production has MemStats
  capture.
- What vNext must prove:
  vNext must define process footprint/RSS evidence separately from heap and allocation
  estimates before adding hard thresholds.

### Actor byte backpressure

- Classification: `planned-vocabulary`.
- Evidence:
  actor runtime has message-count backpressure and fixed message-pool exhaustion
  behavior.
- What vNext must prove:
  vNext must report byte limits, live/reclaimed pool bytes, and byte-aware status
  without breaking current checked errors.

## Explicit Nonclaims To Preserve

- No second competing memory model.
- No zero heap for all programs claim.
- No zero-copy for all programs claim.
- No all-target RAM/RSS parity claim.
- No production object memory or persistent memory claim.
- No performance superiority or official benchmark claim.
- No full production actor runtime claim.
- No distributed zero-copy pointer or region transfer claim.
- No non-Linux distributed actor runtime support claim.
- No cluster membership, reconnect/retry production, supervision, restart, or Erlang/OTP-style actor
  claim.

## Gaps Blocking vNext Completion

1. There is no first-class `MemoryBackend` contract for target-neutral
   reserve/commit/decommit/release/trim/footprint behavior.
2. There is no first-class `MemoryDomain` schema in RAM contract rows.
3. Actor memory is not yet represented as actor-owned memory domains with mailbox bytes, slab/pool
   bytes, owned-region bytes, and byte-aware backpressure.
4. Existing RSS/RAM evidence is capture/classification oriented; it does not yet define hard
   footprint thresholds or portable target adapters.
5. `ActorMoveRegion` and `TaskRegion` vocabulary must remain conservative until runtime ownership
   transfer and domain accounting are proven end-to-end.
6. Local release scripts and validators must reject fake vNext evidence before CI or package
   workflows claim it.

## Task 1 Verdict

Task 1 is satisfied when this audit is present, docs verification passes, and the vNext
implementation keeps these boundaries intact. The next implementation step is to add the
target-neutral `MemoryBackend` contract and tests without changing current allocator semantics.
