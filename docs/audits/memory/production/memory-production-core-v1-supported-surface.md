# Memory Production Core v1 Supported Surface

Status: MPC-0 supported-surface declaration.

The supported surface is intentionally narrower than the long-term memory vision. It covers the safe
features and unsafe boundaries that have current compiler evidence, plus the schema-v1 report slice
introduced by this goal.

## Supported Safe Surface

- Safe slice and String byte views: `window`, `prefix`, `suffix`, `borrow`, `copy`, and `copy_into`.
- Safe slice/String representation metadata (`ptr` and `len`) is readable where the current surface
  already allows metadata reads, but it is not writable user state. Assignment attempts are rejected
  before lowering, including nested and index-through-metadata assignment targets.
- Borrowed returns for documented safe slice/String byte-view paths.
- Conservative ownership, consume, borrow, inout, and resource diagnostics for the current local and
  cross-module surface.
- Conservative mutable alias evidence for supported `inout` parameters: `no_alias`,
  `mutable_exclusive`, `start_inout_exclusive`, and `end_inout_exclusive` rows are projected only
  from compiler-owned PLIR facts. Unknown, maybe, or call-invalidated alias state is not a validated
  noalias claim.
- Bounded function provenance/resource summaries for PLIR-visible return, global-store, actor/task
  escape, closure capture, pointer-retention, consume-param, inout mutation, effects, and capability
  evidence. Unknown external calls and unknown unsafe/resource returns remain conservative.
- Allocation length contract for `make_*` and `island_make_*`.
- Allocation planner reports that keep planned storage separate from actual lowering storage.
- Validated storage-truth rows for the narrow MPC-10 surface: stack storage is validated only when
  the lowered slice does not return, global-store, actor/task transfer, closure-capture,
  unknown-call pass, or aggregate-escape; explicit island storage is validated only with a lowered
  named island slice, region id, lifetime, active handle, and no supported return, use-after-free,
  or double-free violation.
- Validated `FunctionTempRegion` rows for the narrow MPC-11 linux-x64 surface: function-local
  temporary copy buffers are valid only when the allocation plan records
  `planned_storage: FunctionTempRegion`, `actual_lowering_storage: FunctionTempRegion`, a
  single-function lifetime, and the lowered IR contains matching `IRRegionEnter`,
  `IRRegionMakeSlice*`, and `IRRegionReset` evidence. The supported backend path maps one active
  temporary allocation per function with reset cleanup; broader bump-region reuse, multiple active
  temp-region allocations in one function, arbitrary control-flow cleanup, cross-function retention,
  actor/task transfer, and target parity remain outside this claim.
- Planned Stack, Region, ExplicitIsland, Register, Eliminated, FunctionTempRegion, TaskRegion, or
  ActorMoveRegion storage that lowers as Heap remains a heap fallback and is not promoted to a
  validated optimization claim.
- Actor/task/request boundary rules for the current supported surface: borrowed slice/String actor
  payloads are rejected unless the source expression explicitly uses `.copy()`, owned-region actor
  moves are allowed only for the narrow local typed-mailbox contract, and actor zero-copy move
  report rows are `claim_level: evidence_only` with `production_runtime_validated: false`. Typed
  task spawn has no payload expression today; task String/slice payload transfer remains
  conservative rather than a validated cross-task copy path. Request/task region views are explicit
  entry-scope evidence and must reset before escape.
- Bounds proof and PLIR evidence for supported loop/view patterns.

## Supported Unsafe Boundary

- `core.alloc_bytes` is unsafe and may create `unsafe_verified_root` metadata when runtime/IR
  evidence proves the allocation base and size.
- `core.ptr_add`, raw loads, and raw stores remain unsafe and must respect negative offset,
  upper-bound, and access-width checks for verified roots.
- Unknown external raw pointers stay `unsafe_unknown` or `checked_external_unknown`.
- `unsafe_unknown` facts and rows cannot authorize safe provenance, noalias, bounds-check
  elimination, or trusted stack/region/island lowering.
- PLIR records explicit unsafe classes for `core.alloc_bytes`, `core.ptr_add`, raw load/store, and
  raw slice construction; reports project only bounded unsafe-origin gateway rows from those facts.
- `raw_slice_from_parts` is unsafe-only. Unknown raw parts remain external/unknown; verified
  allocation roots may emit bounded `raw_slice_verified_allocation_root` evidence only for proven
  in-bounds constant lengths.
- `cap.mem` authorizes raw operation use; it is not proof of pointer validity, bounds, ownership, or
  safe provenance.

## Report Surface

`--emit-memory-report` writes a sibling `.memory.json` artifact using `tetra.memory-report.v1`. The
report may include:

- a validated `safe_representation_metadata: not_user_assignable` summary row;
- borrowed-view and allocation-intent evidence when mirrored from PLIR or the allocation plan,
  including `borrowed_imm`, `no_escape`, `borrow_owner`, `borrow_source_fact_id`, `copy_owned`,
  `copy_source_fact_id`, and `copy_into_destination_fact_id` for the supported borrow/copy/copy_into
  surface;
- alias/inout rows for the conservative supported subset, including `no_alias`, `mutable_exclusive`,
  `start_inout_exclusive`, and `end_inout_exclusive`;
- function summary rows for the supported MPC-6 vocabulary, including
  `returns_owned_new_allocation`, `returns_borrow_from_param`, `returns_unknown_unsafe`,
  `may_store_global`, `may_escape_to_actor`, `may_escape_to_task`, `may_capture_in_closure`,
  `may_retain_pointer`, `may_return_region`, `may_return_resource`, `may_throw_resource`,
  `may_consume_param`, `may_mutate_inout`, `requires_effects`, `requires_capabilities`, and
  `unknown_external_call_conservative`;
- `allocation_base_metadata` rows for verified `core.alloc_bytes` roots;
- conservative `checked_external_unknown` or `external_unknown` rows for unknown raw pointers and
  raw slices; unsafe unknown borrowed values remain conservative and do not get safe owner/source
  rows;
- raw gateway rows including `derived_allocation_offset`, `raw_memory_access_checked`,
  `raw_memory_access_unknown`, `rejected_negative_offset`, `rejected_upper_bound`, and
  `rejected_access_width_overflow`, plus raw-slice `raw_slice_verified_allocation_root`,
  `rejected_negative_length`, and `rejected_length_overflow` evidence;
- validator rejections for `unsafe_unknown` rows that attempt noalias, `index_in_range`,
  bounds-check-elimination, or trusted storage-lowering claims;
- storage/lowering rows only when a lowered artifact id is attached, with `planned_storage` and
  `actual_lowering_storage` preserved separately.
- memory cost model rows with `cost_class` set to `zero_cost_proven`, `dynamic_check_required`,
  `instrumentation_only`, `unsupported_rejected`, or `conservative_fallback`; dynamic checks must
  keep `normal_build_check` when the normal build still needs the check.
- memory fuzz oracle evidence using `tetra.memory-fuzz.oracle.v1`, Tier 1 short CI smoke artifacts
  under `reports/memory-fuzz-short/...`, and explicit categories for checker reject expected,
  runtime trap expected, compiled output equals interpreter/reference expected, compiler crash is
  bug, miscompile is bug, unsafe_unknown optimized as safe is bug, and report validation failure is
  bug.

The report is not a source of truth and does not change safe-program behavior.

## Explicit Non-Goals

- Full Rust-like borrow checker parity.
- Full Rust-like mutable alias model.
- Named lifetimes or generic lifetime parameters.
- Arbitrary borrowed aggregate returns.
- Universal generic wrapper alias tracking.
- Universal interprocedural storage/lifetime proof; MPC-10 tracks direct explicit-island handle
  identity returns only for the supported lowered IR shape.
- Production FFI lifetime contracts.
- Safety for arbitrary unsafe external pointers.
- Cross-target runtime memory parity without target evidence.
- Target memory production claim parity. See
  `docs/audits/memory/islands/memory-target-capability-matrix.md`; linux-x64
  `production/host_runtime` evidence does not promote linux-x86, linux-x32, macOS, Windows,
  wasm32-wasi, or wasm32-web.
- A performance or fastest-language claim.
- Full production actor runtime, distributed pointer/region zero-copy, or general task/request
  lifetime transfer without explicit runtime/lowering evidence.
- A flag that disables required safe checks.

## Validation Commands

```bash
go test ./compiler/internal/memoryfacts -count=1
go test ./tools/cmd/validate-memory-report -count=1
go test ./compiler -run 'Memory|Raw|Unsafe|Bounds|Report' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
