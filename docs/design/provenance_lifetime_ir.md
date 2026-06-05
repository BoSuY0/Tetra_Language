# Provenance/Lifetime IR

Status: PLIR v0.

`compiler/internal/plir` records a small fact-bearing IR before ordinary stack
IR lowering. The first supported vertical slice is slice-loop proof:

- function-local values
- memory provenance roots
- function region liveness
- immutable/mutable borrow facts
- allocation intent values for `make_*` and `island_make_*`
- safe slice view values for `xs.window`, `xs.prefix`, and `xs.suffix`
- simple local alias facts for safe views and memory-backed slice/String locals
- `for x in xs` loop index range facts
- basic blocks, CFG edges, and dominance rows for proof checking
- proof guard/use metadata for range proofs that justify removed bounds checks
- simple integer range facts for `while i < xs.len` and
  `while i <= xs.len - 1`
- function summary metadata for return ownership, supported return-region and
  resource provenance, thrown resources, declared effects, and mutable-global
  access

Minimum fact vocabulary implemented in Go:

```text
len_stable
index_in_range
region_alive
no_escape
no_alias
non_null
maybe_null
aligned
provenance_known
provenance_unknown
owned
borrowed_imm
borrowed_mut
moved
pure_call
no_heap_allocation
no_mem_write
no_actor_send
no_unknown_escape
derived_window
```

The PLIR verifier rejects contradictory or incomplete facts, including
`len_stable` on unknown provenance and `index_in_range` without a proof id.
It also rejects proof uses whose guard block does not dominate the use block,
unknown proof ids, inverted constant ranges, and range facts without source
metadata. `lower.Lower` runs PLIR verification automatically before stack IR is
emitted.

The v0 dump is available through `compiler.FormatPLIR` and build reports.

Allocation intent values now carry element type, element size, length
expression, optional constant length, source, builtin metadata, and the
zero/negative/overflow guard status required by the allocation-length contract.
`compiler/internal/allocplan` consumes those values to classify escape, storage,
and length status without changing safe semantics. Its report distinguishes
valid empty allocations, normal allocations, rejected negative lengths, rejected
byte-size overflows, and runtime-guarded dynamic lengths. P2.1 lowers the
fixed small no-escape local `make_*` subset to x64 stack-frame backing when the
allocation report records `actual_lowering_storage: Stack`; unsupported targets
continue to lower through the conservative path. P2.2 keeps borrowed views over
those stack-backed locals allocation-free, and allows a fixed local view
`copy()` to become a separate stack-backed owned allocation when it does not
escape. P2.3 adds scalar-replacement evidence for tiny fixed local slices: PLIR
records both `index_load` and `index_store`, and the allocation planner only
selects eliminated storage when every indexed use is a constant in-range access.
P2.4 preserves explicit island provenance through safe view and borrow values,
keeps `copy()` as a new owned allocation provenance root, and validates that
allocation reports claiming `ExplicitIsland` correspond to lowered island slice
IR instead of a conservative heap constructor. P2.5 completes the first copy
storage integration slice: unused `copy()` allocation intents may be actually
eliminated, local fixed copies may be stack-backed, escaping copies remain
owned heap fallbacks, and `copy_into(dst)` remains a no-allocation operation
over caller-owned destination storage. P2.6 adds lowered-IR cross-stage
validation for stack-backed allocation evidence: stack slice pointer tags may
flow through locals and safe view constructors, but any return, call, or global
store path for that pointer is rejected. Returning or inspecting a length value
alone does not escape storage. Allocation reports are also checked against the
exact plan before emission.

Loops over statically invalid constructor expressions do not receive
`index_in_range` proof facts. This keeps bounds reports from claiming a removed
check for an iterable that must trap before it can produce slice metadata.

While-loop bounds-check removal is intentionally narrow. The supported v0
patterns require `i` to be initialized to zero, a loop guard of `i < xs.len` or
`i <= xs.len - 1`, and a unit increment `i = i + 1` in the loop body. Matching
`xs[i]` loads receive a proof id only while the dominating loop guard is live.
Simple aliases such as `let ys = xs` may carry the range proof, but aliases of
raw external views, unknown provenance, or statically invalid constructors stay
checked and do not receive `provenance_known` or `index_in_range` facts.
Control-flow joins merge proof metadata conservatively: external/invalid
provenance is sticky across any incoming path, and zero-initialization is kept
only when every incoming path proves it. Cross-stage validation checks that
every unchecked lowered load references a live PLIR proof guard.

Safe slice view constructors record the source provenance and derived range.
For example `xs.window(1, 2)` records a `derived_window` fact with a range like
`xs[1..1+2]`; loops over the resulting view still receive `index_in_range`
proof facts for bounds-check reporting. A borrow or alias of that view preserves
the derived window range; `copy()` resets provenance to a new owned allocation
and breaks dependence on the source lifetime. When the copied source is a
direct fixed `window`/`prefix`/borrowed view, PLIR records the copy length as a
constant so the allocation planner can decide whether that owned copy can use
stack storage.

For island-backed slices, the same rule is explicit: `island_make_*` creates an
allocation intent with `ProvenanceIsland`, safe `window`/`prefix`/`suffix` and
`borrow()` values preserve that island provenance and stay `no_escape`, while
`copy()` produces `ProvenanceAllocation` with `owned` evidence. The semantic
resource tracker invalidates the island handle after `free(isl)`, so a later
`island_make_*` through the freed handle is rejected before PLIR/lowering.
P5.2 ties island allocation intent facts to the island handle in PLIR: the
allocation value region is `island:<handle>`, the alloc operation records the
handle and length inputs, and an `aligned` fact records the 16-byte region bump
contract consumed by allocation reports.

## Safe View Lifetime Contracts v1

Safe view lifetime checking now uses the same PLIR vocabulary for user-visible
borrow/copy behavior:

- borrowed views emit `borrowed_imm`, `no_escape`, `derived_window` when
  derived through `window`/`prefix`/`suffix`, `provenance_known` when the source
  provenance is known, and `no_heap_allocation`. Memory reports may project the
  safe owner/source relation as `borrow_owner` and `borrow_source_fact_id`.
- borrowed returns are represented as no-escape borrowed values tied to the
  caller-visible source relation; report artifacts preserve deterministic
  borrow facts instead of inventing an allocation site.
- `copy()` emits an owned result with new known provenance and allocation
  intent; the source may still appear in provenance/report relations, but the
  copied value is not borrowed. Memory reports project this relation as
  `copy_owned` and `copy_source_fact_id`.
- `copy_into(dst)` records the checked transfer into existing destination
  storage and must not appear as a fresh allocation. Lowering performs the
  destination prefix check before the copy loop, preserving `dst.len >= src.len`
  as a runtime guard/proof point before any write. Memory reports project the
  destination relation as `copy_into_destination_fact_id`, not as an allocation.

The verifier is intentionally stricter than the current optimizer needs. It
rejects contradictory `owned` plus borrowed facts, borrowed non-parameter facts
without `no_escape`, `derived_window` facts without a source, and copy
allocation intents that do not have matching `owned` evidence. These checks
protect the "reports explain, flags do not change behavior" rule before stack
IR lowering.

## Function Summaries v2

PLIR `FunctionSummary` is populated from the semantics `FuncSig` after the
checker has resolved same-module metadata, plus the supported cross-module and
monomorphized generic metadata that is already present in current interface
summaries. Unsupported cross-module resource shapes and generic lifetime shapes
stay conservative. `FunctionSummary` records only bounded facts: parameter
names/ownership, return ownership/type, throws type, declared effects,
mutable-global access, supported return-region/resource summaries, and thrown
resource summaries.

`compiler/internal/memoryfacts` projects those summaries plus PLIR operation
evidence into memory report rows such as `returns_borrow_from_param`,
`returns_owned_new_allocation`, `may_store_global`, `may_escape_to_actor`,
`may_escape_to_task`, `may_capture_in_closure`, `may_retain_pointer`,
`may_return_region`, `may_return_resource`, `may_throw_resource`,
`may_consume_param`, `may_mutate_inout`, `requires_effects`, and
`requires_capabilities`. Unknown external calls and unsafe/unknown return,
resource, or pointer summaries remain conservative. The summary is an audit
input; it is not a replacement for allocation-lowering validation, proof
checking, or unsafe-boundary validation.
