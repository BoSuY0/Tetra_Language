# Ownership Markers v1

Ownership markers are part of the checked function-call contract in the current production surface.
The local lifetime solver is SSA-like for branch, match, and loop joins: it snapshots
ownership/resource state, merges incoming edges, and reports maybe-consumed or maybe-finalized
diagnostics at the first unsafe use.

## Markers

- `borrow T`: read-only view. The parameter is immutable and values derived from the borrow cannot
  escape through returns, owned parameters, `inout` assignment, or mutable global assignment. The
  current checked escape surface covers region-backed borrowed values plus borrowed `ptr` parameters
  including same-module/cross-module scalar `ptr` `consume` and `inout` assignment,
  same-module/cross-module borrowed ptr-containing aggregate parameters including nested
  inout/global assignment paths, including same-module/cross-module whole-aggregate global
  assignment with stable `TETRA2102` JSON diagnostic evidence, same-module/cross-module
  ptr-containing enum whole-value global assignment with stable `TETRA2102` JSON diagnostic
  evidence, and stable same-module/cross-module global field target assignment with stable
  `TETRA2102` JSON diagnostic evidence, same-module/cross-module aggregate and nested-aggregate
  global field escapes with stable `TETRA2102` JSON diagnostic evidence, same-module/cross-module
  pattern-bound enum payload aliases and if-let/match optional payload aliases including scalar
  return, owned/consume/inout call, inout-assignment, global-assignment, and slice optional payload
  owned/consume/inout call plus `inout` assignment escapes, with stable TETRA2102 JSON diagnostic
  evidence for same-module/cross-module ptr enum-payload return/global/inout assignment escapes,
  same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable
  TETRA2102 JSON diagnostic evidence, and same-module/cross-module slice optional-payload
  inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence,
  same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with
  stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested slice struct
  return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, plus
  same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with
  stable TETRA2101 JSON diagnostic evidence, same-module/cross-module ptr enum-payload
  owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence,
  same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable
  TETRA2101 JSON diagnostic evidence, and same-module/cross-module slice optional-payload
  owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, and
  slice-containing struct literal, alias, nested struct, or nested enum-payload returns, struct
  `inout` assignment, and slice-containing enum direct or alias returns with stable
  same-module/cross-module `TETRA2102` CLI JSON evidence, their local aliases for scalar returns,
  slice-containing struct/enum owned/consume/inout call escapes with stable same-module/cross-module
  and imported direct `TETRA2101` CLI JSON evidence, ptr-containing aggregate literal or
  aggregate-alias returns across struct fields and enum payloads, direct owned/consume/inout
  parameter calls, including same-module/cross-module monomorphized generic aggregate parameters
  with ptr-containing struct/enum aggregate arguments and same-module/cross-module optional `ptr?`
  generic owned/consume/inout instantiations with stable `TETRA2101` CLI JSON evidence plus
  same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable
  `TETRA2102` CLI JSON evidence, imported direct owned/consume/inout calls with optional ptr,
  struct, enum-payload, and nested ptr-containing aggregate arguments, including imported direct
  ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON
  diagnostic evidence, function-typed callback value, struct-field, and enum-payload calls with
  ptr-containing struct/enum aggregate owned/consume/inout arguments and same-module/cross-module
  function-typed value, struct-field, or enum-payload callback calls with optional `ptr?`
  owned/consume/inout arguments and stable `TETRA2101` CLI JSON evidence, ptr-containing aggregate
  `inout` assignment, and mutable global assignment; those failures are reported as borrow escape
  diagnostics.
- `inout T`: exclusive mutable access to a mutable local. The argument must be a mutable local
  value, not a literal or expression.
- `consume T`: moves a local value, a struct-field projection, or an enum payload binding into the
  callee. A consumed whole local cannot be reused, reassigned, or consumed again after the call. A
  consumed struct field cannot be reused, and the enclosing struct cannot be consumed or copied as a
  whole while any field projection is consumed; sibling fields remain usable. Reassigning a mutable
  consumed field, or reassigning the mutable enclosing struct as a whole, reinitializes that
  ownership path and makes it usable again. A consumed enum payload binding marks the matched enum
  payload path unavailable, so the matched enum value cannot be used as a whole while sibling
  payload bindings remain usable. Reassigning the mutable matched enum value as a whole
  reinitializes its payload ownership paths.

## Aliasing Rules

Within a single call, the same ownership path cannot be passed as both `inout` and `borrow`, or as
both `inout` and `consume`. Parent/child paths alias, so `pair` aliases `pair.left`, and `msg`
aliases `msg.$case0.payload0` while a matching enum payload binding is live; sibling field or
payload paths do not alias. The same ownership path cannot satisfy two `consume` parameters in one
call.

## Resource Lifetime Production Slice

The current resource lifetime MVP conservatively tracks task handles, task groups, island handles,
region-backed slices, optional region wrappers, and structs containing those resources through local
scopes and common control-flow joins. It rejects double join/close/use, use-after-transfer,
ambiguous resource provenance on returns, and ambiguous lifetime merges. This is the current
`v0.4.0` local lifetime SSA join slice; richer interprocedural lifetime proofs, broad alias
modeling, race proofs, and full formal lifetime guarantees remain under full-v1 scope.

Branch and loop joins deliberately prefer conservative diagnostics over unsound acceptance. If one
branch assigns a borrowed or region-backed value into a variable and another branch leaves it
unbound or points it at a different region, the checker reports an ambiguous control-flow merge. If
a loop body closes, frees, joins, or consumes a handle, the value is treated as unavailable after
the loop unless the code rewrites ownership so the post-loop state is unambiguous.

## Memory Production Extension

The post-`v0.4.0` Memory Production Core extends the current local lifetime slice toward heap,
slices, structs, closures as one audited transfer and escape surface for native `linux-x64`. This
section is a contract for the work in progress; it does not turn the current raw memory helpers into
a complete production allocator by documentation alone.

The static side must reject borrow escape through returns, owned/consume/inout calls, mutable
globals, aggregate fields, enum/optional payloads, closure captures, and actor/task transfer when
the value carries a pointer, region, or runtime-owned memory handle that cannot be proven sendable.
The memory production extension treats heap, slices, structs, closures, borrow escape, and
actor/task transfer as one audited ownership boundary. The diagnostic policy is conservative rejection:
ambiguous provenance, hidden aliasing, or
unsupported transfer across task/actor/thread boundaries must be rejected with stable ownership
diagnostics instead of being accepted optimistically.

The expected diagnostic families remain `TETRA2101` for use-after-consume, double-use, and
transfer/resource lifetime violations, and `TETRA2102` for borrow escape and pointer/resource
capture escape. Runtime bounds diagnostics belong to the runtime ABI and production evidence; they
do not replace static borrow escape and actor/task transfer checks.

## Actor And Task Transfer

Actor/task transfer safety is a local production slice. It checks worker entrypoints, sendable
scalar and supported structural results, handle transfer, and actor/task use-after-transfer
diagnostics with stable `TETRA2101` CLI JSON evidence, branch/match/loop actor consume reuse
diagnostics with stable `TETRA2101` CLI JSON evidence, same-module/cross-module monomorphized
generic struct actor consume alias diagnostics with stable `TETRA2101` CLI JSON evidence,
same-module/cross-module transitive actor consume aliases with stable TETRA2101 CLI JSON evidence,
including same-module/cross-module actor/task if-let/match optional-payload aliases,
same-module/cross-module actor if-let/match optional-payload, struct-field, and enum-payload consume
aliases including same-module/cross-module actor struct-field/enum-payload alias transfer
diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module actor/task
if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic
evidence, same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics
with stable TETRA2101 JSON diagnostic evidence plus same-module/cross-module task-handle
struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence,
plus same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101
CLI JSON evidence. It does not claim distributed actor safety, full race-safety proofs, full
cancellation semantics, or structured concurrency.

Actor/task worker entrypoints must be zero-argument synchronous user functions returning `i32`.
Worker signatures that borrow, mutate, throw, await, or touch mutable global state are rejected.
Sendable result types are limited to scalar and recursively sendable structural values covered by
the current semantics checker.

Worker call graphs are also checked at the declared worker effect boundary. The current conservative
rule allows actor/runtime scheduling effects needed by the MVP scheduler surface, but rejects worker
entrypoints whose transitive declared effect surface includes raw memory allocation/access,
capability acquisition, MMIO, islands, linker/control, or privacy effects. IO and budget remain
covered by their existing effect and budget-context checkers. This is an explicit diagnostic
boundary, not a full interprocedural race-safety proof.

Typed actor messages use the same P6.1 sendability rule across module boundaries. Imported enum
payloads are checked after resolution: small scalar payloads copy, owned buffer or owned `String`
payloads may copy or move, borrowed slice/String views are rejected unless the payload expression
uses `.copy()`, owned regions/islands move and consume the source, and unknown unsafe provenance is
rejected unless an audited unsafe send contract is present. P6.1 also supports a local zero-copy
move for a region-backed slice when the slice is known to come from `core.island_make_*` and the
same typed actor payload carries the owning `island`; the sender loses both values after send and
`--explain` records actor-transfer evidence. P6.2 evidence includes typed mailbox
schema/capacity/backpressure metadata plus per-payload ownership rows for scalar copy, explicit view
copy, island move, and zero-copy region-slice move behavior. Raw pointers, actor/task handles,
distributed pointer transfer, and other unsupported runtime handles remain stable rejections even
when the enum is declared in another module.

## Current Limits

The checker is intentionally conservative. It tracks region-backed slices, island handles, task
handles, task groups, actor handles, and structs containing them across local scopes and common
control-flow merges. Ambiguous region/resource/lifetime merges are reported as diagnostics and must
be resolved by rewriting the code. Broader interprocedural proofs, formal alias proofs, distributed
actor safety, and synchronization-aware race proofs remain outside this local production slice.

## Epic 06 coverage

Ownership coverage is release-blocking in the focused safety slice:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

The slice checks allowed borrow forwarding, distinct `borrow`/`inout` locals,
same-module/cross-module struct-field partial consume with sibling-field reuse, and enum payload
binding partial consume with sibling-payload reuse plus whole-value call/let/return rejection with
stable `TETRA2101` CLI JSON diagnostics, stable same-module/cross-module whole-copy CLI JSON
rejection after partial struct/enum consume, and same-module/cross-module enum wrapper-constructor
rejection after partial consume with stable `TETRA2101` CLI JSON evidence; permits mutable
struct-field or whole-struct or whole-enum reinitialization after a partial field/payload consume;
rejects reuse after `consume`, whole-struct or whole-enum use after a child ownership path has been
consumed, same-module/cross-module optional-payload whole-value rejection after payload consume/free
with stable TETRA2101 JSON diagnostic evidence, field writes through a consumed parent, borrowed
values escaping through returns, owned/consume/inout parameters including imported direct call
boundaries, or mutable global `ptr` assignment, double use of closed/joined resources, resource
use-after-free, double-join, ambiguous-provenance, same-module/cross-module struct-field and
enum-payload alias use-after-free, same-module/cross-module struct-field and enum-payload alias
use-after-free with stable `TETRA2101` JSON diagnostic evidence, and island transfer
non-local-payload rejection with stable `TETRA2101` CLI JSON evidence, callable mutable-capture
global-escape and heap-escape CLI JSON diagnostics with the stable lifetime safety code `TETRA2102`,
callable pointer/resource capture escape CLI JSON diagnostics with `TETRA2102`, callable
captured-value or function-typed-parameter global-storage escape CLI JSON diagnostics with
`TETRA2102`, unsupported function-value escape outside the fnptr ABI with `TETRA2102`, and capturing
closure raw-ptr escape with `TETRA2102`, function-typed storage/return unsupported capture rejection
with `TETRA2102`, captured closure explicit type-arg rejection with `TETRA2102`, function-typed
explicit type-arg rejection with `TETRA2102`, unsupported function-value call with `TETRA2102`, plus
generic closure capture and generic callback-closure capture rejection and generic closure
pointer/direct-call rejection with `TETRA2102`, and imported mutable function-typed global boundary
diagnostics with `TETRA2102`, generated `.t4i` function-typed parameter local-alias return metadata
for interface-only global-storage diagnostics, same-module/cross-module borrowed scalar `ptr`
escapes through ptr-containing struct `inout` assignment, same-module/cross-module fixed-array alias
return plus direct global assignment, optional global assignment, and inout-assignment escapes with
stable `TETRA2102` diagnostic evidence and borrowed string alias return/global assignment escapes
with stable `TETRA2102` CLI JSON evidence, slice-containing struct literal/alias/nested
struct/enum-payload return, struct `inout` assignment, match/catch-expression return escapes,
typed-error throw ptr/region payload escapes, and enum direct/alias return escapes with stable
same-module/cross-module `TETRA2102` CLI JSON evidence, slice-containing struct/enum
owned/consume/inout call escapes with stable same-module/cross-module and imported direct
`TETRA2101` CLI JSON evidence, ptr/slice optional assignment function-typed
value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence, return/owned/consume/inout escape,
including stable same-module/cross-module `TETRA2101`/`TETRA2102` CLI JSON evidence for slice
optional assignment owned/consume/inout and return escapes, same-module/cross-module slice optional
payload binding owned/consume/inout call, `inout` assignment, and global assignment escapes with
stable `TETRA2101`/`TETRA2102` CLI JSON evidence, same-module/cross-module direct slice global
assignment with stable `TETRA2102` JSON diagnostic evidence, same-module/cross-module optional ptr
global assignment with stable `TETRA2102` JSON diagnostic evidence, and same-module/cross-module
optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence,
same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102
JSON diagnostic evidence, same-module/cross-module slice optional-payload inout/global assignment
escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested slice
enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence,
same-module/cross-module nested slice struct return/inout/global assignment escapes with stable
TETRA2102 JSON diagnostic evidence, scoped island optional region-wrapper escape with `TETRA2102`,
same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic
evidence, and same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field
return escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module
whole-aggregate global assignment with stable `TETRA2102` JSON diagnostic evidence,
same-module/cross-module generic slice-containing struct/enum aggregate owned/consume/inout
instantiations with stable `TETRA2101` CLI JSON evidence, branch/match/loop task-handle
maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics, branch/match/loop
resource finalization merge diagnostics with stable `TETRA2101` JSON evidence, stable `TETRA2101`
task-group use-after-close CLI JSON diagnostics, same-module/cross-module struct-field and
enum-payload alias use-after-free CLI JSON diagnostics, same-module and interface-only cross-module
per-field interprocedural region summaries for aggregate returns from multiple island parameters,
including optional aggregate wrappers, enum payload wrappers, branch aggregate wrappers, match
aggregate wrappers, if-let aggregate wrappers, mixed safe/provenance aggregate branch and match
returns, and optional mixed safe/provenance aggregate branch merges, same-module/cross-module
transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI
JSON evidence, same-module/cross-module enum-constructor return resource aliases with stable
TETRA2101 CLI JSON evidence, same-module typed-error throw/catch and rethrow-through-try
enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence, generated `.t4i`
direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource
return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and
nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and
rethrow-through-`try` direct/field-local-alias provenance stubs, enum-payload and if-let/match
optional-payload return resource alias double-free, including nested struct-field and enum-payload
optional resource wrappers with stable same-module/cross-module `TETRA2101` CLI JSON evidence,
same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases
with stable TETRA2101 CLI JSON evidence, same-module/cross-module task-handle/task-group
struct-field/enum-payload transfer/join/close aliases including same-module/cross-module task-handle
struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable
TETRA2101 JSON diagnostic evidence, and same-module/cross-module task-group
struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases
with stable TETRA2101 CLI JSON evidence, and same-module/cross-module monomorphized generic struct
actor consume aliases with stable TETRA2101 CLI JSON evidence, same-module/cross-module protocol
impl parameter ownership matching plus same-module/cross-module protocol impl parameter ownership
mismatch diagnostics with stable TETRA2001 CLI JSON evidence, and same-module/cross-module generic
protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic
evidence, and ambiguous resource provenance; and verifies actor/task handles cannot be used after
ownership transfer.

Plan250 Epic 04 coverage additionally exercises branch-merge region diagnostics, looped
use-after-consume/resource-use diagnostics, cross-module typed actor message sendability,
conservative worker effect-boundary race diagnostics, and deterministic budget guard lowering:

```sh
go test ./compiler -run "Plan250Safety|Plan250Runtime|Plan250Link" -count=1
```
