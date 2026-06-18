# Storage Classes

Status: P2.0 report schema with P2.1/P2.2 stack-backed slices, P2.3 scalar replacement, P2.4
explicit island validation, P2.5 copy/copy_into integration, P2.6 cross-stage allocation validation,
the P5.0 runtime allocation contract, and P15.2 `linux-x64` per-core small-heap safe-slice runtime
evidence.

Allocation storage classes are planner facts first and backend-lowering facts second. Reports
therefore expose both `planned_storage` and `actual_lowering_storage`.

## Required Classes

`Eliminated` : No backing storage is needed. Used for valid empty allocations or allocation intents
whose result is unused or scalar-replaced. In P2.3, tiny fixed local slices with only constant
in-range indexed uses lower to scalar locals and report `lowering_status: scalar_replacement`. In
P2.5, unused `copy()` results lower through `lowering_status: eliminated_unused_copy`: source view
checks are still evaluated, but no fresh destination allocation is emitted.

`Stack` : The planner proved the allocation is fixed-size, small enough for the current threshold,
and does not escape. In P2.1 x64 native builds lower the supported local `make_*` subset to
frame-backed storage. In P2.2, borrowed local views over those stack-backed slices remain
no-allocation views, and a fixed local view `copy()` may also be stack-backed when the copy result
does not escape. P2.6 validates the lowered IR with a tag dataflow pass: returning the stack slice
header, passing it to a call, or storing it globally is rejected even if the raw stack-machine shape
is otherwise valid. Unsupported targets may still use the conservative heap path while reporting
that actual fallback.

`ExplicitIsland` : User-written island scope owns the allocation. The island must dominate the
supported uses, and the allocation must not escape the island scope. P2.4 validates this storage
class against lowered `IRIslandMakeSlice*` evidence: a plan cannot report
`actual_lowering_storage: ExplicitIsland` while lowering emits a heap `make_*` path or no island
slice constructor. Borrowed island views stay no-allocation/no-escape; an escaping `copy()` from an
island view is a fresh owned allocation and is planned independently from the island backing
storage.

`Heap` : Conservative runtime allocation fallback. Returned allocations, unknown calls, unsafe
exposure, actor crossings, unknown sizes, and unsupported proof states use this class. P5.0 freezes
the heap runtime path contract: 16-byte minimum alignment, guarded invalid sizes before allocator
access, stable trap/status failure behavior, and report hooks for runtime path and bytes. P15.2
keeps the storage class as `Heap` while exposing the concrete `linux-x64` runtime path for constant
safe-slice allocations: `runtime_path: per_core_small_heap` with an `allocator_class` such as
`small_32`, `allocator_scope: core:0`, and
`allocator_reuse_policy: same_core_same_size_class_free_list`, or `runtime_path: large_mmap` for
requests beyond the 4096-byte small class.

`LargeMmap` : Reserved for future large allocation lowering. P2.0 defines the class but does not yet
select it for runtime lowering.

`UnknownConservative` : Explicit unknown state. The planner could not prove a narrower storage
contract, so lowering must remain conservative.

## Forward-Compatible Classes

The v0 enum also includes `Register`, `TaskRegion`, `ActorMoveRegion`, and `External` for planned
follow-up work. They are report schema values, not permission to bypass validation.

P5.3 enables the first `Region` planner selection for function-local temporary copies that do not
already fit the fixed-small stack lowering subset. It remains a modeled region, not an implemented
implicit backend region: the row must name the region id, lifetime, runtime path, and heap fallback
evidence.

## Report Reading Rules

- `storage` is retained as the compatibility planned-storage field.
- `planned_storage` is the authoritative planner decision.
- `actual_lowering_storage` is the current backend path.
- When the two differ, `backend_storage`, `backend_reason`, and `lowering_status` explain the
  conservative fallback.
- For x64 native P2.1 stack-lowered sites, `planned_storage` and `actual_lowering_storage` are both
  `Stack`.
- For x64 native P2.2 borrowed views, no allocation record is created for the borrow itself. A
  non-escaping `copy()` has its own allocation record and may report `Stack`; an escaping `copy()`
  remains `Heap`.
- For P2.3 scalar-replaced slices, the allocation record reports `Eliminated`, but validation does
  not expect an `IRStackSlice*` instruction: the lowered evidence is the absence of slice backing
  plus scalar local loads/stores for the constant element accesses.
- For P2.4 explicit island slices, the allocation record reports `ExplicitIsland` for both planned
  and actual storage, and validation expects the corresponding `IRIslandMakeSliceU8`,
  `IRIslandMakeSliceU16`, or `IRIslandMakeSliceI32` instruction in the lowered function. `bool`
  island slices use the same I32-width lowered representation as ordinary bool slices.
- For P5.2 explicit island rows with constant byte sizes, reports also expose
  `runtime_path: explicit_island`, `allocator_class: region_bump_16`, `bytes_requested`,
  16-byte-rounded `bytes_reserved`, `region_id`, `lifetime`, and `debug_mode`. These hooks describe
  the aligned bump allocator and bulk island free; they do not imply compiler-selected implicit
  regions yet.
- For P5.3 function-local temporary region rows, reports expose `planned_storage: Region`,
  `runtime_path: region`, `allocator_class: function_temp_region`,
  `region_id: region:<function>:temp`, `lifetime: function:<function>`, and
  `debug_mode: region_reset_when_enabled`. Until implicit region lowering is implemented, those same
  rows must keep `actual_lowering_storage: Heap` and
  `lowering_status: region_planned_heap_fallback`.
- For P2.5 `copy_into(dst)`, no allocation record is created. The proof/PLIR report names the
  no-allocation copy-into operation, while the allocation report continues to show only the
  source/destination allocations that already exist.
- For P2.6, allocation reports are checked against the exact plan before they are written. A report
  with mismatched totals or allocation rows is rejected instead of being emitted as misleading
  evidence.
- For P15.2 `linux-x64` safe-slice heap rows with constant byte sizes, reports expose
  `runtime_path`, `allocator_class`, `bytes_requested`, and `bytes_reserved`, plus
  `allocator_scope`, `allocator_reuse_policy`, and `allocator_chunk_bytes`. P15.3 `core.alloc_bytes`
  rows also expose `raw_pointer_bounds`, `raw_pointer_base`, `raw_pointer_base_bytes`, and
  `raw_slice_policy` so allocator optimizations can see verified allocation roots without trusting
  arbitrary unsafe pointers.
- P5.4 allocation reports use schema v2. In addition to per-allocation hooks, they expose a
  `summary` with allocation count, planned-storage counts, actual-lowering counts, runtime-path
  counts, total requested/reserved bytes, allocator-class counts, allocator-scope counts,
  allocator-reuse-policy counts, raw-pointer bounds status counts, raw-slice policy counts, and
  per-region counts for rows with `region_id`. Stack-lowered rows report runtime path `stack_frame`;
  eliminated rows report runtime path `eliminated`; heap fallback rows report a heap/runtime path.
- Report flags such as `--explain` and `--emit-alloc-report` never change the selected storage or
  safe semantics.
