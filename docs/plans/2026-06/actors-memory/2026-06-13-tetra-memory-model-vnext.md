# Tetra Memory Model vNext Execution Plan

**Goal:** Extend the existing Tetra Memory Model with a target-neutral Memory Runtime Substrate and
Actor Memory Domains, without replacing the current ownership/islands/RAM-contract model.

**Context:** The current repository already has ownership markers, effects, islands/regions,
allocation planning, runtime allocation contracts, RAM contract reports, and actor transfer safety.
The vNext work must connect those layers into a measurable memory economy: heap, regions,
actor-owned memory, copy counts, and process footprint/RSS evidence.

**Execution:** Use `executing-plans` task-by-task for the first pass. Switch to
`subagent-driven-development` only after the audit/design tasks are merged and the implementation
tasks are small enough for per-task review.

## Current Ground Truth

- Effects and policy already model `alloc`, `mem`, `islands`, `budget`, and actor/runtime boundaries
  in `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Ownership/lifetime rules already cover `borrow`, `inout`, `consume`, resources, actor/task
  transfer, and conservative rejection in `docs/spec/runtime/ownership_v1.md`.
- Islands already provide arena/bump allocation and scoped lifetime rules in
  `docs/spec/memory/islands.md`.
- Allocation planning already records planned and actual storage in `compiler/internal/allocplan`,
  including `Stack`, `FunctionTempRegion`, `ExplicitIsland`, `TaskRegion`, `ActorMoveRegion`,
  `Heap`, and `LargeMmap`.
- Runtime allocation contracts already live in `compiler/internal/runtimeabi`, including
  `per_core_small_heap`, region alignment, explicit islands, and allocation report hooks.
- RAM contract reports already live in `compiler/internal/ramcontract` and explicitly do not claim
  zero heap, zero-copy for all programs, all-target RAM parity, or performance superiority.
- Actor transfer already supports narrow local owned-region/zero-copy evidence in
  `docs/spec/runtime/actors.md`, `docs/design/actor_region_transfer.md`, and
  `compiler/internal/parallelrt`, but current production actor runtime capacity remains fixed and
  bounded.

## Non-Goals

- Do not introduce a second competing memory model.
- Do not claim zero heap for all programs.
- Do not claim zero-copy for all actor transfers.
- Do not claim cross-target RSS parity before target adapters and gates exist.
- Do not promote the actor runtime to a full production multi-threaded actor system as part of this
  plan.
- Do not wire GitHub Actions until local gates and release scripts are proven and the CI scope is
  explicitly approved.

## Task 1 - Current-State Audit And Claim Boundary

**Goal:** Produce a single audit document that separates implemented behavior, prototype evidence,
planned vocabulary, and nonclaims.

**Files:**

- Inspect `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Inspect `docs/spec/runtime/ownership_v1.md`.
- Inspect `docs/spec/memory/islands.md`.
- Inspect `docs/spec/runtime/actors.md`.
- Inspect `docs/design/runtime_allocation_contract.md`.
- Inspect `docs/design/ram_contract_compiler.md`.
- Inspect `docs/design/actor_region_transfer.md`.
- Inspect `compiler/internal/allocplan`.
- Inspect `compiler/internal/runtimeabi`.
- Inspect `compiler/internal/ramcontract`.
- Add `docs/audits/memory/zero-heap-final/tetra-memory-model-vnext-current-state.md`.

**Approach:**

- Classify every memory surface as `implemented`, `report-only`, `prototype-evidence`,
  `planned-vocabulary`, or `nonclaim`.
- Record where `planned_storage` differs from `actual_lowering_storage`.
- Call out current RSS limits: `ram-measurement.json` is MemStats capture evidence and does not
  enforce hard RSS thresholds.
- Call out current actor limits: fixed actor table, fixed mailbox depth, fixed message pool, and
  prototype-only scheduler/zero-copy benchmark rows.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check docs/audits/memory/zero-heap-final/tetra-memory-model-vnext-current-state.md docs/plans/2026-06/actors-memory/2026-06-13-tetra-memory-model-vnext.md
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-docs go clean -cache
```

**Done when:** The audit names every current layer and explicitly blocks overclaims about zero heap,
universal zero-copy, production actor runtime, and RSS thresholds.

**Notes:** If `verify-docs` requires manifest updates for the new audit, add an explicit
docs-manifest task before editing generated files.

## Task 2 - MemoryBackend Contract Design

**Goal:** Define a target-neutral backend contract for memory reservation, commitment, release,
trimming, and footprint accounting.

**Files:**

- Modify or extend `docs/design/runtime_allocation_contract.md`.
- Add `docs/spec/memory/memory_backend_vnext.md`.
- Inspect `compiler/internal/runtimeabi/allocation_contract.go`.
- Inspect `compiler/internal/runtimeabi/smallheap/small_heap.go`.
- Inspect `compiler/internal/runtimeabi/region_allocator.go`.
- Add implementation files under `compiler/internal/runtimeabi` only after the design doc is
  accepted.

**Approach:**

- Define the contract operations: `reserve`, `commit`, `decommit`, `release`, `trim`, and
  `footprint`.
- Define the difference between requested, reserved, committed, released, resident/current
  footprint, and peak footprint.
- Keep the contract target-neutral. Linux can be the first adapter, but the language/runtime
  contract must not expose Linux-specific names as the model.
- Add unsupported/blocked reporting for targets that cannot provide a metric.
- Preserve existing allocation paths: `heap`, `per_core_small_heap`, `large_mmap`,
  `explicit_island`, `region`, `stack_frame`, and `eliminated`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-runtimeabi go test ./compiler/internal/runtimeabi -run 'Allocation|SmallHeap|Region|MemoryBackend' -count=1
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-runtimeabi go clean -cache
```

**Done when:** The contract can describe Linux, Windows, macOS, WASM, and unknown targets without
changing allocator semantics per target.

**Notes:** The first implementation may support only Linux, but every report row must say whether a
metric is measured, estimated, unsupported, or blocked.

## Task 3 - MemoryDomain Data Model

**Goal:** Add a common domain model for process, task, actor, island, and request memory ownership.

**Files:**

- Add `docs/spec/memory/memory_domains_vnext.md`.
- Inspect and modify `compiler/internal/ramcontract/types.go`.
- Inspect and modify `compiler/internal/ramcontract/from_allocplan.go`.
- Inspect `compiler/internal/memoryfacts`.
- Inspect `compiler/internal/allocplan`.

**Approach:**

- Define domain kinds: `process`, `task`, `actor`, `island`, `request`, and `external`.
- Define stable fields: `domain_id`, `parent_domain_id`, `owner_kind`, `owner_id`, `lifetime`,
  `budget_bytes`, `requested_bytes`, `reserved_bytes`, `committed_bytes`, `released_bytes`,
  `current_bytes`, `peak_bytes`, `copy_count`, and `bytes_copied`.
- Connect allocation rows to domains without changing safety semantics.
- Preserve the existing RAM grades and blocker reports.
- Keep stack/register/eliminated storage visible as allocation-plan facts, not process RSS claims.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-domain go test ./compiler/internal/ramcontract ./compiler/internal/memoryfacts ./compiler/internal/allocplan -run 'RAMContract|MemoryFact|AllocPlan|Domain|Budget|Blocker' -count=1
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-domain go clean -cache
```

**Done when:** RAM contract rows can be grouped by domain and still validate against allocation-plan
facts and existing blocker semantics.

**Notes:** If `compiler/internal/memoryfacts` does not yet expose enough domain facts, add an
investigation subtask before changing report schemas.

## Task 4 - Allocator Integration With Domains

**Goal:** Make allocator evidence domain-aware while preserving current planner/runtime behavior.

**Files:**

- Inspect and modify `compiler/internal/allocplan/plan.go`.
- Inspect and modify `compiler/internal/runtimeabi/smallheap/small_heap.go`.
- Inspect and modify `compiler/internal/runtimeabi/region_allocator.go`.
- Inspect allocation report validators under `tools/cmd/validate-memory-report` and memory
  production validators under `tools/validators/memoryprod`.

**Approach:**

- Add domain metadata to allocation report rows only after the RAM contract data model is accepted.
- Attach heap and small-heap allocations to the process/default runtime domain first.
- Attach explicit islands to island domains.
- Keep `FunctionTempRegion` and `ActorMoveRegion` conservative until actual lowering/runtime
  ownership transfer is proven.
- Keep estimated allocator evidence separate from runtime-measured memory.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-alloc go test ./compiler/internal/allocplan ./compiler/internal/runtimeabi ./tools/validators/memoryprod -run 'Alloc|Runtime|SmallHeap|Region|Domain|Report|Evidence' -count=1
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-alloc go clean -cache
```

**Done when:** Allocation reports can show domain ownership without converting planned/prototype
storage into unsupported runtime claims.

**Notes:** Do not make `ActorMoveRegion` mean real actor-domain transfer until Task 5 proves the
actor-side report and runtime path.

## Task 5 - ActorMemoryDomain

**Goal:** Model actor-owned memory as domains: mailbox pool, message slabs, owned regions,
byte-aware backpressure, and zero-copy owner transfer.

**Files:**

- Modify `docs/spec/runtime/actors.md`.
- Modify `docs/design/actor_region_transfer.md`.
- Inspect and modify `compiler/internal/actorsrt`.
- Inspect and modify `compiler/internal/parallelrt`.
- Inspect and modify `compiler/internal/actorsafety`.
- Investigate the current producer of `<output>.actor-transfer.json` before editing actor-transfer
  report schema.

**Approach:**

- Define `ActorMemoryDomain` as a domain owned by an actor handle/runtime actor id.
- Track mailbox bytes separately from mailbox message count.
- Track message slab/pool capacity, live bytes, reclaimed bytes, and backpressure status.
- Treat local zero-copy region transfer as owner movement: sender actor domain -> receiver actor
  domain.
- Keep distributed actor frames copy/serialization-based unless a future audited transport contract
  proves otherwise.
- Add byte-aware backpressure without removing the existing checked `-2` mailbox-full behavior.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-actors go test ./compiler/internal/actorsrt ./compiler/internal/parallelrt ./compiler/internal/actorsafety -count=1
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-actors go test ./compiler -run 'Actor|Mailbox|Backpressure|MessagePool|Typed|ZeroCopy|Transfer' -count=1
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-actors go clean -cache
```

**Done when:** Actor reports can explain message count limits and byte limits, and a local
owned-region transfer updates domain ownership without claiming distributed zero-copy.

**Notes:** Keep production actor-runtime promotion out of scope. This task extends memory accounting
and safety evidence only.

## Task 6 - RAM/RSS Measurement And Gates

**Goal:** Add a reliable measurement/reporting layer for heap, allocator bytes, domain bytes, and
process footprint/RSS evidence.

**Files:**

- Modify `tools/cmd/memory-production-smoke/report.go`.
- Modify `tools/cmd/validate-memory-production/main.go`.
- Modify `tools/validators/memoryprod`.
- Modify `docs/design/runtime_allocation_contract.md`.
- Modify `docs/release/production/post_v0_4_linux_x64_memory_parallel_ui_scope.md` only after the
  schema and local validators pass.

**Approach:**

- Extend `tetra.memory.ram-measurement.v1` or add a vNext artifact only after a schema decision is
  written.
- Report these fields separately: `heap_alloc_bytes`, `bytes_requested`, `bytes_reserved`,
  `bytes_committed`, `bytes_copied`, `rss_current`, `rss_peak`, and `per_actor_domain_bytes`.
- Keep `MemStats` as one method, not the whole truth.
- Add RSS methods as measured, unsupported, or blocked. Linux may read process footprint first, but
  the schema must allow other target adapters later.
- Start with observation thresholds. Add hard fail thresholds only when baseline evidence exists and
  the threshold policy is accepted.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-measure go test ./tools/cmd/memory-production-smoke ./tools/cmd/validate-memory-production ./tools/validators/memoryprod -run 'RAM|RSS|Measurement|MemStats|Footprint|Evidence|Benchmark' -count=1
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/memory-vnext-local
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-measure go clean -cache
```

**Done when:** The memory production report can distinguish runtime measured RSS/footprint evidence
from allocation-report estimates and blocked/unsupported measurement methods.

**Notes:** Use a fresh report directory. Do not write evidence commands that set `GOCACHE` under
`/tmp`.

## Task 7 - Release Gate Integration

**Goal:** Wire the vNext memory evidence into local release gates without overclaiming CI or
all-target support.

**Files:**

- Inspect `scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh`.
- Inspect `scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh`.
- Inspect `scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`.
- Inspect `.github/workflows/ci.yml`.
- Inspect `.github/workflows/release-packages.yml`.
- Modify release scripts only after local validators pass.
- Modify workflows only after explicit CI wiring approval.

**Approach:**

- Add local release-script checks for new memory-domain and footprint artifacts.
- Keep CI workflow changes separate from local implementation.
- Make validators reject docs-only, metadata-only, stale, fake, and overclaiming reports.
- Preserve the existing ordering: memory before parallelism/actor promotion, actor foundation before
  broader runtime claims.

**Verification:**

```sh
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/memory-vnext-local-final
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-release go run ./tools/cmd/validate-memory-production --report reports/memory-vnext-local-final/memory-production-linux-x64.json --manifest reports/memory-vnext-local-final/memory-release-manifest.json --report-dir reports/memory-vnext-local-final
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-release go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-memory-vnext-release go clean -cache
git diff --check
```

**Done when:** A fresh local release directory validates with the new artifacts, and CI remains
unchanged unless the CI step was explicitly approved.

**Notes:** If this plan is executed in a dirty worktree, commit only the files created or modified
for this vNext scope.

## Rollout Order

1. Complete Task 1 and review the audit.
2. Complete Task 2 and review the target-neutral backend contract.
3. Complete Task 3 and review the domain schema before touching actor runtime.
4. Complete Task 4 for allocator/report integration.
5. Complete Task 5 for actor domains.
6. Complete Task 6 for RAM/RSS measurement.
7. Complete Task 7 for local release gate integration.

## Acceptance Criteria

- The existing Tetra Memory Model remains the base model.
- vNext adds a target-neutral memory substrate instead of Linux-specific public semantics.
- Allocation/RAM reports can group memory by domain.
- Actor memory can be reported by actor domain without claiming a full production actor runtime.
- Zero-copy actor transfer is represented as ownership movement of a proven local owned region, not
  as arbitrary pointer sharing.
- Runtime-measured memory, RSS/footprint, and allocation-report estimates are visibly different
  evidence classes.
- Local validators reject fake or overclaiming reports.
- Documentation and reports keep explicit nonclaims for universal zero heap, universal zero-copy,
  all-target parity, and benchmark/performance superiority.

## Open Decisions

- Whether to evolve `tetra.memory.ram-measurement.v1` in place or add a vNext footprint artifact.
- Whether hard RSS thresholds should be global, per target, per benchmark, or per memory grade.
- Whether actor byte backpressure should return a new checked status or reuse the existing
  backpressure surface with richer diagnostics.
- Where the first `MemoryDomain` facts should be born: PLIR, MemoryFactGraph, AllocPlan, or RAM
  contract projection.
- Whether `ActorMoveRegion` should remain report vocabulary until a runtime actor-domain transfer
  gate proves it end-to-end.
