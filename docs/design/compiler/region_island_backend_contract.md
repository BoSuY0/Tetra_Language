# Region/Island Backend Contract

## Status

Status: design contract for future implementation packets.

This document pins the backend classification and native/register eligibility
rules for the Tier 1 row `region_island_allocation_tetra`. It is a contract
between Stack IR classification, Machine IR/register backend reporting, x64
island emission support, allocation/domain evidence, and Tier 1 report
ingestion.

This document does not implement backend behavior and does not authorize a
report-only promotion. A row may not become `backend_path="register"` unless a
fresh backend sidecar proves a real register/native path for every function
required by the row.

## Current Row Evidence

Fresh evidence comes from
`reports/benchmark-vnext-memory-baseline/tier1-after-matrix-multiply-main-native/`.
The current row state is:

- Tier 1 report row `region_island_allocation_tetra` is measured but still
  `backend_path="fallback"`.
- Its row blocker is `backend_blockers=["unsupported_effect_runtime_call"]`.
- `bounds_left=0`, `heap_allocations=0`, `heap_reason_codes=[]`, and
  `perf_blockers=[]`.
- Runtime feature evidence is separate from backend eligibility:
  `runtime_features_required`, `runtime_features_linked`, and
  `runtime_features_initialized` are all `["island_allocator"]`, with
  `runtime_feature_evidence.evidence_class="lowered_ir_static_plan"`.
- The runtime object plan is also separate:
  `runtime_object_plan.runtime_used=false`,
  `runtime_object_linked=false`, and `runtime_object_initialized=false`.

The backend sidecar
`artifacts/bin/region_island_allocation_tetra.backend.json` proves the current
backend classification, not native eligibility:

- `backend="stack"`;
- `summary.function_count=1`;
- `summary.register_path=0`;
- `summary.stack_fallback=1`;
- `summary.categories.unsupported_effect_runtime_call=1`;
- function `p25.region_island_allocation.main` has
  `backend_path="stack"`, `category="unsupported_effect_runtime_call"`,
  and `detail="ir_kind=49"`.

The IR enum evidence in `compiler/internal/ir/ir.go` identifies this first
reported blocker as `IRIslandNew` in the island IR block. The current
classifier in `compiler/internal/buildreports/backend.go` treats
`IRIslandNew`, `IRIslandMakeSliceU8`, `IRIslandMakeSliceU16`,
`IRIslandMakeSliceI32`, and `IRIslandFree` as generic effect-runtime IR kinds.
The same file separately maps island IR kinds to the `island_allocator`
runtime feature, including `IRIslandReset`.

The allocation sidecar
`artifacts/bin/region_island_allocation_tetra.alloc.json` proves explicit
island allocation, not heap allocation:

- `storage_classes.ExplicitIsland=1`;
- `actual_lowering_storage_classes.ExplicitIsland=1`;
- `runtime_paths.explicit_island=1`;
- `allocator_classes.region_bump_16=1`;
- `memory_backend_classes.region=1`;
- `bytes_requested=64`, `bytes_reserved=64`, `bytes_committed=64`, and
  `bytes_released=64`;
- function allocation `xs` has `builtin="core.island_make_i32"`,
  `storage="ExplicitIsland"`, `runtime_path="explicit_island"`,
  `allocator_class="region_bump_16"`, and
  `memory_backend.adapter="runtime.region_bump_v1"`;
- allocation-domain evidence includes `domain:island:isl` with island kind and
  64-byte budget/request/reserve/commit/release/current/peak accounting.

Domain bytes remain a separate evidence stream. The allocation sidecar exposes
`domain:island:isl`; the Tier 1 row `memory_evidence.domain_bytes` currently
shows the runtime-measured process domain. A later worker must not replace one
claim with the other.

RSS is also separate evidence. The current row records
`rss_current=20480` and `rss_peak=11522048` for iteration 01. These values are
observations only; they are not a reduction claim or a native/backend claim.

## Non-Claims

This contract makes these non-claims explicit:

- No row-level native/register support exists for
  `region_island_allocation_tetra` today.
- The current `backend_path="fallback"` row must not be relabeled to
  `backend_path="register"` without backend sidecar proof.
- Existing x64 island emitters prove that target emitters exist, but they do
  not by themselves prove a Machine IR/register backend path for this row.
- Heap evidence, bounds evidence, domain bytes, runtime feature evidence,
  runtime object evidence, and RSS evidence are distinct claims.
- `heap_allocations=0` does not prove native/register backend support.
- `bounds_left=0` does not prove native/register backend support.
- `runtime_features_*=["island_allocator"]` does not prove native/register
  backend support.
- Explicit-island allocation evidence does not prove RSS reduction.
- This row does not prove actor transfer, actor zero-copy, JSON/HTTP/hash,
  matrix, slice, or broad region/island native support.
- This contract does not approve generic support for all `IRRegion*` or
  `IRIsland*` operations.

## Backend Classification Contract

Backend classification must split region/island work into three cases.

1. Generic runtime-effect call.

Use the existing generic `unsupported_effect_runtime_call` bucket only for
real runtime-effect calls or effect IR that has not been identified as a
region/island domain primitive. Examples include unknown `IRCall` targets whose
names look like `__tetra_*`, `runtime.*`, or `core.*`, and non-island effect
IR with no accepted backend contract.

2. Precise island/domain fallback.

Island/domain primitives that are not eligible for the exact native path must
not remain hidden behind the generic runtime-effect bucket. A later classifier
slice should use precise categories such as:

- `unsupported_island_domain_primitive` for an island/domain primitive whose
  allocation/domain semantics are known but whose backend lowering path is not
  accepted;
- `unsupported_island_runtime_effect` for a true island runtime operation,
  unknown island runtime call, missing required runtime evidence, or accepted
  island API used outside the proven shape.

The detail must name the IR kind or runtime symbol and must preserve runtime
feature evidence. A precise fallback remains fallback; it is not a register
path.

3. Exact explicit-island native path.

An explicit-island native path is allowed only for a fully recognized function
shape. The first candidate is the current
`p25.region_island_allocation.main` shape identified by P106: an explicit
island scope using `IRIslandNew`, `IRIslandMakeSliceI32`, and island-scope
cleanup, with one `core.island_make_i32` allocation of length 16, zero heap,
zero bounds left, and no unrelated runtime calls.

Recognition must be structural, not name-only. The recognizer must validate
the accepted Stack IR shape, the allocation intent, the island/domain evidence,
the control-flow shape used by the benchmark, and the required x64 emission
support. Any extra island operation, unknown island API, dynamic unsupported
shape, missing proof, missing allocation-domain evidence, or mixed accepted and
unaccepted runtime/effect operation keeps the function on precise fallback.

## Native/Register Eligibility

`region_island_allocation_tetra` may become `backend_path="register"` only
when all of these conditions hold in fresh evidence:

- The backend report is produced by real backend selection, not by changing the
  Tier 1 report label.
- The backend sidecar has `function_count=1`, `register_path=1`,
  `stack_fallback=0`, and no fallback category for
  `p25.region_island_allocation.main`.
- The function sidecar row has `backend_path="register"` and a detail that
  identifies the accepted explicit-island lowering path, for example an exact
  `machine-ir-region-island-allocation-main` style recognizer detail.
- The row-level `backend_path` in `report.json` is derived from that sidecar
  evidence, and `backend_blockers` no longer contains
  `unsupported_effect_runtime_call`.
- Tests prove that a report-only `backend_path="register"` change without
  matching sidecar evidence is rejected.
- Runtime feature evidence remains visible:
  `runtime_features_required`, `runtime_features_linked`, and
  `runtime_features_initialized` must still include `island_allocator`, and
  lazy-init blockers must stay explicit.
- Runtime object evidence remains honest. If the explicit-island path continues
  to use compile-time allocation/domain evidence rather than a linked runtime
  object, the sidecar must keep `runtime_object_plan` fields truthful instead
  of implying runtime object linkage.
- Allocation evidence remains explicit-island evidence:
  `ExplicitIsland=1`, `runtime_path="explicit_island"`,
  `allocator_class="region_bump_16"`, region backend operations, and 64-byte
  request/reserve/commit/release accounting stay present.
- Bounds and heap remain independently green:
  `bounds_left=0`, `heap_allocations=0`, and allocation-sidecar `heap=0`.
- Domain bytes remain visible as domain/accounting evidence and are not folded
  into heap, backend, runtime feature, or RSS claims.
- RSS is reported only as measured RSS evidence unless a separate RSS policy
  and validation gate explicitly claims reduction.

## Worker-Ready Implementation Slice

Preferred next implementation slice: a precision-first backend classification
packet, not a register-promotion packet.

That worker may:

- add RED tests proving the current island row must not be classified as the
  generic `unsupported_effect_runtime_call` bucket once the contract is
  implemented;
- add precise island/domain fallback categories for recognized-but-not-native
  island/domain primitives;
- keep `region_island_allocation_tetra` as fallback if the worker only lands
  precise classification;
- preserve runtime feature evidence, runtime object evidence, allocation
  evidence, heap evidence, bounds evidence, domain bytes, and RSS as separate
  fields;
- prepare a later exact-native recognizer test fixture for
  `p25.region_island_allocation.main`.

That worker must not:

- set row-level `backend_path="register"` without a backend sidecar with
  `register_path=1` and `stack_fallback=0`;
- use row name, benchmark name, or allocation sidecar evidence alone as native
  eligibility;
- make all `IRIsland*` or `IRRegion*` operations native by allow-list;
- hide island/domain fallback behind a generic runtime-effect category after
  the precise classifier is available;
- alter heap, bounds, RSS, actor, JSON/HTTP/hash/matrix, or benchmark-report
  semantics as a shortcut.

A register-promotion worker is not ready until the precision-first classifier
tests exist and a separate packet scopes an exact structural recognizer for
the current explicit-island row.

## Required Tests

Any later code packet that changes region/island backend behavior must include
targeted tests before implementation and keep them fresh after implementation:

- backend classification tests for island/domain primitives that distinguish
  generic runtime-effect calls, precise island/domain fallback, and the exact
  explicit-island native candidate;
- negative tests proving unknown island APIs, extra island operations, missing
  allocation/domain evidence, mixed accepted/unaccepted effects, or unsupported
  runtime calls stay fallback;
- a test that prevents row-level `backend_path="register"` unless the backend
  sidecar proves register/native support;
- if native promotion is attempted, a Machine IR/backend recognizer test for
  the exact `p25.region_island_allocation.main` shape;
- if native promotion is attempted, x64 emission tests or reuse of existing
  x64 island emitter tests proving `IRIslandNew`, `IRIslandMakeSliceI32`, and
  island cleanup preserve the allocation contract and guard order;
- allocation contract tests preserving `core.island_make_i32` explicit-island
  behavior: zero length valid without metadata access, negative/overflow reject
  before metadata access, `runtime_path="explicit_island"`, and
  `allocator_class="region_bump_16"`;
- Tier 1 metadata tests preserving `heap_allocations=0`, `bounds_left=0`,
  allocation-domain visibility, runtime feature evidence, and RSS separation.

## Fresh Tier 1 Gate

Every later region/island backend code packet must run a fresh Tier 1 evidence
gate after the change.

For a classifier-only packet, the fresh gate must show:

- `region_island_allocation_tetra` may remain `backend_path="fallback"`;
- the blocker is no longer the vague generic bucket if the classifier split is
  in scope;
- `heap_allocations=0`;
- `bounds_left=0`;
- allocation sidecar explicit-island evidence is unchanged or deliberately
  improved with evidence;
- runtime feature evidence still reports `island_allocator`;
- RSS values are only measured observations unless an RSS packet is in scope.

For a native/register packet, the fresh gate must additionally show:

- row-level `backend_path="register"`;
- backend sidecar `function_count=1`, `register_path=1`, and
  `stack_fallback=0`;
- function-level `backend_path="register"` for
  `p25.region_island_allocation.main`;
- no row blocker for `unsupported_effect_runtime_call`;
- heap, bounds, explicit-island allocation, domain bytes, runtime feature
  evidence, runtime object evidence, and RSS evidence remain separately
  inspectable.

## Open Risks

- The current backend sidecar reports `runtime_features_*=["island_allocator"]`
  while `runtime_object_plan.runtime_used=false`. The contract treats this as
  descriptive static-plan evidence, not a backend blocker, but later native
  work must keep the distinction explicit.
- Existing x64 island emitters are necessary evidence, not sufficient evidence,
  for Machine IR/register backend eligibility.
- Tier 1 row `memory_evidence.domain_bytes` currently exposes the runtime
  process domain, while the allocation sidecar exposes `domain:island:isl`.
  Later ingestion work must decide whether and how to surface island-domain
  bytes in the row without overwriting runtime-measured process evidence.
- `IRIslandReset` is runtime-feature mapped and emitted by x64, while the
  current generic effect classifier evidence observed by P106 centers on
  `IRIslandNew`, `IRIslandMakeSlice*`, and `IRIslandFree`. Tests should pin
  desired reset classification before reset is used as evidence for this row.
- Debug-island behavior and syscall-backed reserve/commit/release behavior may
  add target-specific obligations; they are not broad native eligibility by
  themselves.
