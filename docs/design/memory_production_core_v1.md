# Memory Production Core v1 Design

Status: Immediate MPC-0/MPC-1/MPC-2 design note.

Memory Production Core v1 moves memory evidence toward one compiler-owned fact
model:

```text
compiler stage facts -> MemoryFactGraph -> validators -> report projection
```

Reports are projections. They are useful for audits, release gates, and
debugging, but they must never reconstruct facts the compiler did not own.
Target capability rows are likewise bounded evidence: linux-x64
`production/host_runtime` memory reports do not become cross-target runtime
claims without a target-specific row in
`docs/audits/memory-target-capability-matrix.md`.

## Fact Ownership

`compiler/internal/memoryfacts` owns the v0 fact vocabulary:

- source ids: value, function, site, source span, parent fact, lowered artifact;
- classes: provenance, unsafe origin, borrow, escape, alias, storage;
- validation: source stage, validator name, validation status.

The graph rejects unsafe-to-safe promotion from `unsafe_unknown`, unsafe
unknown `no_alias` or bounds-proof claims, trusted stack/region-style storage
lowering for unsafe unknown rows, derived facts without parents, and validated
storage/lowering claims without a lowered artifact id.

MPC-3 also projects a compiler-owned representation namespace fact:
`safe_representation_metadata: not_user_assignable`. That fact is backed by the
semantics type model and assignment-target guard rather than by report-only
reconstruction.

## Projection

`BuildReportFromGraph` converts graph facts to `tetra.memory-report.v1` rows.
Validated facts become `claim_level: validated` only when their graph validation
state is `pass`. Rejected or invalidated facts become rejected rows. Unknown raw
pointers are conservative rows, not safe rows.

MPC-7 hardens unsafe fact classes. `unsafe_verified_root` may describe bounded
metadata from known Tetra allocation roots such as `core.alloc_bytes`, but it
remains unsafe-origin evidence. `unsafe_unknown` rows may explain checked or
conservative raw-pointer behavior, but they cannot authorize `no_alias`,
`index_in_range`, bounds-check elimination, or validated stack/region/island
lowering. PLIR records the same unsafe class on raw allocation roots, raw
pointer arithmetic, raw slice construction, and raw load/store gateway
operations so report projection does not recover unsafe origin from ad hoc
strings alone.

MPC-4 projects the supported borrow/copy/copy_into surface from PLIR and the
allocation plan. Borrowed views may emit `borrowed_imm`, `no_escape`,
`borrow_owner`, and `borrow_source_fact_id` rows when the source is a safe,
visible owner. `copy()` emits `copy_owned` and `copy_source_fact_id` evidence
for the new owned allocation provenance. `copy_into(dst)` emits
`copy_into_destination_fact_id` for the caller-owned destination relation and
does not create a storage/allocation claim. Borrowed values with
`unsafe_unknown` provenance stay conservative and do not derive safe
owner/source rows.

MPC-5 projects the conservative mutable alias/inout subset from PLIR. Supported
`inout` parameter facts may emit `no_alias` with `alias_state:
mutable_exclusive`, plus `mutable_exclusive`, `start_inout_exclusive`, and
`end_inout_exclusive` evidence rows. Unknown, maybe, or call-invalidated alias
state must not become a validated `no_alias` claim. The slice intentionally
does not claim universal generic wrapper tracking or a full Rust-like mutable
alias model.

MPC-6 projects bounded function provenance/resource summaries. PLIR carries a
checker-derived `FunctionSummary` with return ownership, supported region and
resource return summaries, throws-resource summaries, declared effects, and the
mutable-global touch bit. `compiler/internal/memoryfacts` also derives summary
facts from PLIR operations such as return, global store, actor send, task spawn,
closure capture, unknown external call, moved parameter, and inout mutation.
Safe summary rows remain `evidence_only`; unknown external calls, retained
pointers, unknown unsafe returns, and unknown resource returns remain
`conservative`. Allocation planning and proof/validation consumers keep their
existing conservative behavior unless a later validated narrow fact is added;
MPC-6 summary rows alone do not grant storage, alias, or safe-provenance
claims.

Memory Ideal Vertical Slice v2 adds narrow report projections for borrowed
views carried through function-typed values and callback parameters. These rows
are derived only from safe borrowed PLIR parent facts and known direct callback
targets; unknown callback targets stay conservative and do not emit trusted
borrow facts. Callback/reentrant `inout` evidence projects
`callback_inout_conservative` with `alias_state: invalidated_by_call` rather
than a validated noalias claim. This does not implement a full callable ABI,
captured or escaping closures, async/task/actor memory boundaries, raw pointer
expansion, target parity, broad noalias, or performance claims.

Memory Ideal Vertical Slice v3 adds narrow report projections for borrowed
views carried through interface/protocol-like values on the already-supported
static-conformance surface. `interface_value_contains_borrow` is trusted only
when the checker/PLIR exposes a safe borrowed parent fact and a statically known
concrete target. Unknown dynamic protocol dispatch projects
`protocol_dispatch_borrow_conservative`, and protocol/interface dispatch alias
evidence projects `protocol_dispatch_noalias_conservative` with conservative
fallback rather than validated noalias. This does not implement runtime
protocol values, trait objects, existential containers, witness tables,
conformance-table lookup, full dynamic dispatch, async/task/actor memory
boundaries, raw pointer expansion, target parity, broad noalias, or performance
claims.

Memory Ideal Vertical Slice v4 adds narrow report projections for
async/task/actor boundary evidence on the already-supported async/await,
typed-task, and typed-actor surfaces. `async_boundary_borrow_conservative`
keeps borrowed views crossing an async suspension conservative unless proven
local and non-escaping. `task_boundary_borrow_rejected` and
`actor_boundary_borrow_rejected` reject borrowed views crossing task/actor
boundaries without explicit copy. `boundary_noalias_conservative` records that
task/actor boundary alias evidence cannot become broad noalias. This does not
implement a full async lifetime system, production actor runtime, structured
concurrency, cancellation model, distributed actor memory model, zero-copy
region move expansion, raw pointer expansion, target parity, broad noalias, or
performance claims.

Memory Ideal Vertical Slice v5 adds narrow raw-pointer unsafe contract
projections without making arbitrary unsafe memory safe.
`unsafe_unknown_rejected_safe_facts` records that unknown external raw pointers
cannot produce `safe_known`, `provenance_known`, or noalias facts.
`unsafe_verified_root_allocation_base` records validated bounded
`core.alloc_bytes` allocation-base metadata as `unsafe_verified_root`, not safe
provenance. `unsafe_contract_runtime_checkable` validates only runtime-checkable
nonnull, alignment, and length/bounds contracts. `unsafe_contract_static_untrusted`
keeps unsafe noalias, lifetime, and region contracts conservative unless a
separate proof exists. This does not implement arbitrary external pointer
safety, an FFI lifetime system, broad unsafe noalias, safe wrapper promotion,
actor/task/runtime expansion, target parity, or performance claims.

Memory Ideal Vertical Slice v6 adds narrow bounds-check proof-id projections.
`bounds_check_retained_dynamic` records normal-build checks when no
compiler-owned proof exists. `bounds_check_removed_with_proof_id` records only
proof-tagged removed checks whose proof ids trace back to PLIR proof guards or
equivalent compiler-owned proof metadata. Missing proof ids project
`bounds_check_removal_rejected_missing_proof_id`, and raw bounds
width/overflow uncertainty projects `raw_bounds_runtime_check_normal_build`
with `normal_build_check`. This does not implement a broad optimizer proof,
target parity, performance claims, arbitrary unsafe pointer arithmetic proof,
or a full theorem prover.

Memory Ideal Vertical Slice v7 adds narrow external pointer and FFI lifetime
quarantine projections. `ffi_pointer_external_unknown` keeps external pointer
provenance unsafe/external unknown. `ffi_call_may_retain_borrow` records that
external calls may retain borrowed pointers unless a compiler-owned contract
proves otherwise. `safe_wrapper_promotion_rejected_without_contract` rejects
safe wrapper promotion from raw/external pointers without proof.
`ffi_noalias_invalidated_by_external_call` keeps external-call alias evidence
conservative rather than promoting broad noalias. This does not implement
arbitrary external pointer safety, C/FFI lifetime safety, safe wrapper
promotion, broad unsafe noalias, target parity, performance, arbitrary external
allocator provenance, or full runtime/ABI proof.

Memory Ideal Vertical Slice v8 adds graph/report projection and claim-drift
integrity. `ValidateReportProjection` checks emitted memory reports against
the `MemoryFactGraph` that produced them, rejecting unknown `source_fact_id`
rows, missing projected facts, and altered projection fields such as
`cost_class`, `normal_build_check`, parent/source links, validator fields, and
claim levels. The correlation validator recognizes the exact five v8
`MEM-REPORT-*` rows and rejects missing, extra, widened, or broad-safety
claim-drift wording. This does not add new memory semantics, optimizer
behavior, target parity, performance evidence, FFI/runtime proof, arbitrary
external pointer safety, or a "Memory 100%" claim.

Memory Ideal Vertical Slice v9 adds narrow escape-aware storage/lowering
integrity. `allocplan.VerifyPlan` rejects escaped values whose planned,
reported, or actual lowering storage uses trusted stack/register/region,
function-temp region, explicit island, task-region, actor-move-region, or
non-empty eliminated storage without compiler-owned no-escape proof. Heap or
conservative trusted-storage fallback rows must preserve their
`source_fact_id` and carry a reviewable `reason`. Task, actor, FFI, and
unknown-call storage boundaries remain conservative unless a later narrow
proof exists. This does not implement full region inference, optimizer-wide
allocation correctness, target parity, performance evidence, production actor
runtime proof, full async lifetime proof, arbitrary FFI lifetime proof,
arbitrary external pointer safety, or a "Memory 100%" claim.

## Compiler Integration

`BuildOptions.EmitMemoryReport` and `--emit-memory-report` request a report
artifact only. The flag does not change parsing, checking, PLIR construction,
allocation planning, lowering, bounds checks, runtime traps, or optimization
eligibility.

The initial adapter mirrors bounded PLIR/allocation-plan facts rather than
rewriting every compiler stage at once. That keeps the first slice auditable and
lets later MPC milestones promote additional stages into the graph with tests.

## Raw-Bounds Closure

For this slice, verified `core.alloc_bytes` roots may project
`allocation_base_metadata` and the v5
`unsafe_verified_root_allocation_base` row with `unsafe_verified_root`. Unknown
external raw pointers, raw memory accesses, and raw slices remain conservative.
Verified-root
`ptr_add` and raw load/store gateways now distinguish
`rejected_negative_offset`, `rejected_upper_bound`, and
`rejected_access_width_overflow` rows when constant allocation metadata is
available. These are `unsafe_checked` diagnostics/evidence only; they are not
safe provenance and do not make arbitrary raw pointers safe. Pointer-width raw
access evidence in MPC-8 is linux-x64 runtime evidence; non-x64 targets remain
build/lower scoped until target-aware raw pointer metadata is added.
MPC-9 extends the same conservative boundary to raw slice construction:
verified allocation roots can report `raw_slice_verified_allocation_root` only
when constant length bytes fit inside allocation metadata, while unknown raw
parts remain `external_unknown`. Linux-x64 raw slice construction traps negative
lengths and element-size byte overflow before creating a view; other targets are
currently build/lower/report scoped, and target-overflow metadata remains a
future target-aware extension.
Memory Ideal v6 additionally projects `raw_bounds_runtime_check_normal_build`
from unsafe-checked raw bounds evidence so overflow/width uncertainty remains a
normal-build check or trap instead of a zero-cost elimination claim.
Memory Ideal v7 additionally projects FFI/external-boundary evidence so
external pointers remain unknown/conservative, borrowed arguments may be
retained by external calls, safe-wrapper promotion without compiler-owned proof
is rejected, and external calls invalidate broad noalias.
Memory Ideal v8 additionally validates report projection integrity so audit
artifacts cannot add unknown source facts, drop graph facts, alter
`cost_class`, drop `normal_build_check`, or widen conservative/rejected
evidence into broad safety claims.
Memory Ideal v9 additionally validates escape-aware storage and lowering
integrity so escaped or boundary-crossing values cannot be projected as
trusted stack/region/task/actor storage without compiler-owned no-escape
evidence, and heap fallback rows remain traceable and reasoned.

## Memory Cost Model

MPC-14 adds `docs/design/memory_cost_model.md` as the cost vocabulary for memory
reports and performance blocker reports. Memory report rows project
`cost_class` from compiler-owned facts: `zero_cost_proven`,
`dynamic_check_required`, `instrumentation_only`, `unsupported_rejected`, or
`conservative_fallback`. `dynamic_check_required` rows must keep
`normal_build_check` when the check remains in a normal build. `unsafe_unknown`
may be checked, trapped, or conservative, but never optimized as trusted.

## Memory Fuzz Oracle

MPC-15 uses `tetra.memory-fuzz.oracle.v1` as the bounded fuzz/property/stress
oracle for memory evidence. Tier 1 short CI smoke writes
`reports/memory-fuzz-short/...` artifacts. Tier 2 nightly fuzz and Tier 3
release-blocking focused memory fuzz are boundary-recorded separately. The
oracle treats checker reject expected, runtime trap expected, and compiled
output equals interpreter/reference expected as passing categories when the
expected observation occurs. Compiler crash is bug, miscompile is bug,
unsafe_unknown optimized as safe is bug, and report validation failure is bug.

## Representation Invariant

Slice/String layout fields remain available to compiler internals and existing
safe read paths, but their `FieldInfo` entries are not `UserAssignable`.
`resolveAssignTarget` and the assignment collection pass reject attempts to
write `ptr`, `len`, or reserved representation metadata names before lowering.
This protects direct assignments such as `xs.len = 1`, nested assignments such
as `box.value.len = 1`, inout assignments, and index-through-metadata targets
such as `xs.ptr[0] = 1`.
