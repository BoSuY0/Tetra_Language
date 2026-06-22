# Allocation Planner Lowering Readiness

Status: P2.0 readiness gate plus P2.1/P2.2/P2.3/P2.4/P2.5/P2.6 lowering and validation slices, with
P5.1-P15.2 runtime allocation evidence.

Allocation planning is an explanatory compiler stage before storage-aware lowering. It records what
storage the planner can justify today, and it records what the current backend actually lowers to.
Report flags expose this evidence only; they must not change safe semantics or silently select a
faster storage path.

## Allocation Records

Each allocation record carries:

- `id`: local allocation name used in PLIR and reports.
- `site_id`: stable allocation site id, currently `allocsite:<function>:<allocation>:<source>`.
- `value_id`: original PLIR allocation-intent value.
- `builtin`: constructor such as `core.make_u8` or `core.island_make_u8`.
- `element_type`, `element_size`, `length_expr`, and `length_status`.
- `zero_guard_status`, `negative_guard_status`, and `overflow_guard_status`.
- `escape`: planner escape class.
- `storage`: compatibility alias for the planned storage class.
- `planned_storage`: storage class justified by planner facts.
- `actual_lowering_storage`: storage class used by current lowering.
- `validation_status`: verifier conclusion for the planned storage.
- `lowering_status`: explanation of the actual lowering path.
- P5/P15 report hooks: `runtime_path`, `allocator_class`, `allocator_scope`,
  `allocator_reuse_policy`, `allocator_chunk_bytes`, `bytes_requested`, `bytes_reserved`,
  `region_id`, `lifetime`, and `debug_mode` where the hook applies to the row.

`planned_storage` and `actual_lowering_storage` are intentionally separate. For unsupported targets,
a fixed small no-escape allocation can be planned as `Stack` while the backend records
`actual_lowering_storage: Heap` with `lowering_status: conservative_heap_fallback`. For x64 native
targets in P2.1/P2.2/P2.3/P2.4/P2.5/P2.6, the same proven subset records concrete actual lowering
evidence such as `actual_lowering_storage: Stack` / `lowering_status: stack_lowering`,
`actual_lowering_storage: Eliminated` / `lowering_status: scalar_replacement` or
`lowering_status: eliminated_unused_copy`, or `actual_lowering_storage: ExplicitIsland` /
`lowering_status: explicit_island_lowering`.

## Validation Rules

The allocation-plan verifier is the P2.0 gate for future storage-changing work:

- `Stack`, `Register`, and `Eliminated` require `escape: NoEscape`.
- `Region` requires `escape: NoEscape`. In P5.3 this is limited to function-local temporary copies;
  returned values, actor sends, unknown calls, and task boundaries stay heap until an
  ownership/transfer region is modeled.
- `Eliminated` is allowed when the allocation is unused, scalar-replaced, or a valid empty
  allocation that needs no backing storage. Valid empty slice headers may escape because they carry
  `ptr=0,len=0` and no frame-backed payload.
- `ExplicitIsland` requires the island scope to dominate all supported uses. This is represented by
  island provenance plus `escape: NoEscape`; P2.4 also validates that lowering emits the matching
  `IRIslandMakeSlice*` instruction for every explicit island allocation record.
- `Heap` is the safe conservative fallback for returned allocations, unknown calls, actor crossings,
  unsafe exposure, unknown size, or unsupported proof states.
- Negative lengths and byte-size overflow remain rejected before storage selection. Valid empty
  allocations remain valid empty slices without allocator access on supported native ABI paths.

The verifier rejects missing stable site ids, missing builtins, missing planned/actual storage,
missing validation/lowering statuses, planned storage that diverges from the compatibility `storage`
field, and impossible escaping stack/register/eliminated/island plans.

P2.6 adds a lowered-IR dataflow check for stack-backed allocation records. The checker tags the
escape-sensitive pointer slot produced by each planned stack allocation, propagates that tag through
locals and safe view constructors, and rejects any path that returns the tagged pointer, passes it
to an unknown call, or stores it in a global. Returning or using a length slot alone does not extend
the stack lifetime. Normal in-function indexed loads/stores and `write` operations consume the
header without extending its lifetime and remain valid.

Allocation report generation also validates that the emitted report mirrors the exact allocation
plan totals, function allocation rows, and P5.4 summary. The schema v2 summary records allocation
count, planned-storage counts, actual-lowering counts, runtime-path counts, allocator-class counts,
allocator-scope counts, allocator-reuse-policy counts, total requested/reserved bytes, and
per-region summaries. A mismatched report is a compiler error, not a diagnostic artifact.

## Lowering Contract

P2.0 did not change runtime allocation storage. P2.1 enables the first real storage-changing path
for x64 native targets, P2.2 extends that path to borrowed local views and non-escaping copies of
fixed local views, P2.3 eliminates the backing allocation for tiny fixed slices with constant
in-range indexed uses, P2.4 hardens explicit island lowering evidence, P2.5 ties copy/copy_into
decisions to the same planner contract, and P2.6 validates the cross-stage storage evidence:

- `planned_storage` names the storage class the planner can prove.
- `actual_lowering_storage` names what lowering implements now.
- `backend_storage` and `backend_reason` remain compatibility notes when the actual backend is more
  conservative than the planner.
- `Stack` lowering is restricted to fixed-size, positive-length, no-escape local `make_*`
  allocations whose byte size fits the conservative stack threshold.
- Lowered IR must preserve the no-escape proof for stack-backed allocations. A forged plan that
  stack-lowers a value and then returns, calls, or stores its header globally is rejected even if
  the stack-machine verifier accepts the raw stack heights.
- Valid empty local `make_*` allocations lower to a zero pointer/zero length header without backing
  storage.
- `window`/`prefix`/`suffix` and `borrow()` over a stack-lowered local slice do not introduce
  allocation intents. They keep the view header stack-local and rely on the existing checked view
  construction and borrow lifetime rules.
- `copy()` over a fixed local view is an owned allocation intent. It can be stack-lowered only when
  the copy result does not escape the function. If the copy is returned or stored across an escape
  boundary, the planner keeps the conservative heap path.
- With P5.3 region planning enabled, a non-escaping temporary `copy()` that does not already fit the
  fixed-small stack lowering subset may report `planned_storage: Region` and
  `region_id: region:<function>:temp`. Planned-only rows must also report
  `actual_lowering_storage: Heap`, `backend_storage: Heap`, and
  `lowering_status: region_planned_heap_fallback`. When function-temp lowering is actually emitted,
  the runtime path is `scoped_single_mapping_v0` with `allocator_class: function_temp_region`; this
  is not a general arena.
- An unused `copy()` result may report `planned_storage: Eliminated`,
  `actual_lowering_storage: Eliminated`, and `lowering_status: eliminated_unused_copy`. Lowering
  still evaluates the source expression so safe view range checks are preserved, then stores a dummy
  empty local header without allocating or copying payload bytes.
- `copy_into(dst)` is not an allocation intent. PLIR records it as a checked call that copies into
  caller-owned destination storage, allocation reports do not create a fresh site for it, and
  lowering uses the checked destination prefix before the copy loop so `dst.len >= src.len` is
  proven or trapped before any write.
- Returning a borrow derived from a local stack allocation is rejected before lowering. Returning
  `copy()` is allowed because it creates owned storage.
- Invalid view ranges still trap or reject before the view header is consumed; stack lowering does
  not bypass those checks.
- Tiny fixed local `make_*` allocations may report `planned_storage: Eliminated`,
  `actual_lowering_storage: Eliminated`, and `lowering_status: scalar_replacement` when every
  indexed use is a constant in-range load/store. Lowering maps each element to a scalar local and
  emits no slice backing, no `make_*`, and no indexed memory operation for those accesses.
- Dynamic or out-of-range indices, view construction, unknown calls, returns, unsafe exposure, or
  other non-constant uses keep the allocation on the conservative stack/heap path. Scalar
  replacement does not invent bounds proofs; if proof-tagged BCE is needed, it must still come from
  PLIR proof facts and cross-stage validation.
- Explicit island allocation records report `planned_storage: ExplicitIsland`,
  `actual_lowering_storage: ExplicitIsland`, `validation_status: validated_explicit_island_scope`,
  and `lowering_status: explicit_island_lowering`. Cross-stage validation rejects a report that says
  explicit island while lowered IR emits a non-island `make_*` path or omits the matching
  `IRIslandMakeSliceU8/U16/I32` instruction.
- Borrowed views derived from island-backed slices remain no-escape views and cannot be returned,
  stored globally, or sent across ownership boundaries. Returning `copy()` from an island view is
  allowed only because the copy is a new owned allocation with non-island provenance and
  heap/stack/region planning of its own.
- `free(isl)` finalizes the island resource facts. Reusing the freed island for another
  `island_make_*` call, double-freeing it, or using aliases after free remains a safety diagnostic
  before lowering.

Future P2 slices may replace more conservative heap fallbacks with mmap, region, or broader
scalar-replaced lowering only after the verifier and reports expose enough evidence to review the
semantic effect.
