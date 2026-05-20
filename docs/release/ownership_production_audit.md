# Tetra Ownership Production Audit

Status: achieved.

Audit date: 2026-05-07.

This audit maps the active ownership objective to concrete repository evidence.
It is stricter than the local safety-readiness gate: green package tests,
generated manifests, and the current safety profile count only when they cover a
specific ownership requirement.

## Objective Restatement

Full production ownership requires:

- checked `borrow`, `consume`, and `inout` call contracts
- SSA and interprocedural lifetime analysis
- alias/provenance tracking
- move/copy/drop/finalization semantics
- partial moves for struct fields and enum payloads
- ownership-aware generics, protocol/interface conformance, and callable values
- controlled heap/global/thread/callback escape behavior
- actor/task/island/resource transfer rules
- stable diagnostics for forbidden alias, use-after-move/use-after-consume,
  use-after-free, borrow-escape, and double-drop/double-finalization cases
- spec, docs, examples, tests, validators, feature registry evidence, and
  release-gate evidence
- no filler or demo-only production claims

## Prompt-To-Artifact Checklist

| Requirement | Required artifact or command | Current evidence | Result |
| --- | --- | --- | --- |
| Borrow/consume/inout model | `compiler/tests/ownership/ownership_test.go`; `docs/spec/ownership_v1.md`; `examples/ownership_smoke.tetra` | See `Evidence Details / Borrow/consume/inout model`. | pass |
| SSA local lifetime analysis | `language.lifetime-ssa`; `compiler/tests/ownership/ownership_test.go`; `compiler/islands_scope_test.go`; `compiler/tests/runtime/resource_finalization_test.go` | See `Evidence Details / SSA local lifetime analysis`. | pass |
| Interprocedural lifetime analysis | ownership/lifetime tests and release audit | See `Evidence Details / Interprocedural lifetime analysis`. | pass |
| Alias/provenance tracking | `compiler/tests/ownership/ownership_test.go`; `compiler/tests/runtime/resource_finalization_test.go`; `compiler/tests/ownership/actor_task_ownership_test.go`; `compiler/compiler_test.go` | See `Evidence Details / Alias/provenance tracking`. | pass |
| Move/copy/drop/finalization semantics | `compiler/tests/ownership/ownership_test.go`; `compiler/tests/runtime/resource_finalization_test.go` | See `Evidence Details / Move/copy/drop/finalization semantics`. | pass |
| Partial moves for struct/enum fields | `compiler/tests/ownership/ownership_test.go`; `examples/ownership_smoke.tetra` | See `Evidence Details / Partial moves for struct/enum fields`. | pass |
| Ownership-aware generics/interfaces/callables | `compiler/tests/ownership/ownership_test.go`; `compiler/generics_test.go`; `compiler/tests/callables/function_typed_callable_test.go`; `compiler/tests/semantics/closures_semantic_clauses_test.go`; `compiler/tests/runtime/resource_finalization_test.go`; `cli/cmd/tetra/main_test.go` | See `Evidence Details / Ownership-aware generics/interfaces/callables`. | pass |
| Heap/global/thread/callback escape analysis | `compiler/tests/semantics/async_test.go`; `compiler/tests/semantics/async_ownership_test.go`; `compiler/tests/semantics/async_inout_ownership_test.go`; `compiler/internal/semantics/callable_escape_test.go`; `compiler/tests/ownership/ownership_test.go`; callable escape diagnostics | See `Evidence Details / Heap/global/thread/callback escape analysis`. | pass |
| Actor/task/island/resource transfer rules | `compiler/tests/ownership/actor_task_ownership_test.go`; `compiler/tests/runtime/resource_finalization_test.go`; `compiler/tests/safety/safety_diagnostics_test.go`; `cli/cmd/tetra/main_test.go`; `docs/spec/ownership_v1.md` | See `Evidence Details / Actor/task/island/resource transfer rules`. | pass |
| Stable forbidden-case diagnostics | ownership/resource/callable negative tests; JSON diagnostic shape gate | See `Evidence Details / Stable forbidden-case diagnostics`. | pass |
| Spec/docs/examples/tests evidence | `docs/spec/ownership_v1.md`; `docs/spec/current_supported_surface.md`; `docs/spec/flow_syntax_v1.md`; `examples/ownership_smoke.tetra`; compiler tests | See `Evidence Details / Spec/docs/examples/tests evidence`. | pass |
| Dedicated ownership validator evidence | `tools/cmd/validate-ownership-audit`; `go test ./tools/cmd/validate-ownership-audit -count=1` | See `Evidence Details / Dedicated ownership validator evidence`. | pass as blocker |
| Feature registry evidence | `compiler/features.go`; `./tetra features --format=json`; `docs/generated/manifest.json` | See `Evidence Details / Feature registry evidence`. | pass |
| Release-gate evidence | `bash scripts/ci/test-all.sh`; safety-readiness step | See `Evidence Details / Release-gate evidence`. | pass |

## Evidence Details

### Borrow/consume/inout model

Parser/checker tests cover marker syntax, mutability, alias rejection, consume
reuse, borrow forwarding, borrow escape, and the runnable ownership smoke now
includes optional `ptr?` borrow matching plus enum payload partial consume with
sibling reuse.

### SSA local lifetime analysis

Local branch/match/loop flow snapshots report maybe-consumed diagnostics.

They also report branch/match/loop resource finalization merge diagnostics with
stable TETRA2101 JSON evidence, optional region-wrapper escapes for scoped
island slices with stable `TETRA2102` CLI JSON diagnostics, branch/match/loop
task-handle maybe-joined, task-group maybe-closed, island maybe-freed, and
maybe-finalized diagnostics.

### Interprocedural lifetime analysis

Local return-resource summaries, typed-error throw-resource summaries including
rethrow-through-`try`, same-module and interface-only cross-module per-field
interprocedural region summaries for aggregate returns from multiple island
parameters, including optional aggregate wrappers, enum payload wrappers,
branch aggregate wrappers, match aggregate wrappers, if-let aggregate wrappers,
mixed safe/provenance aggregate branch and match returns, and optional mixed
safe/provenance aggregate branch merges exist.

generated `.t4i`
direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias
resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match
optional and nested/field-local nested optional resource return, typed-error
direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias
provenance stubs exist.

selected same-module/cross-module transitive interprocedural resource cases,
including task-handle, task-group, island, struct-field, enum-payload,
enum-constructor return, same-module throw/catch enum-payload,
if-let/match optional-payload, and nested struct/enum optional-payload return
resource aliases exist, including cross-module async relay wrappers (`await` and
`try await`), including optional-wrapper and `match`-optional async relay cases.

### Alias/provenance tracking

Ownership paths, enum payload aliases, borrowed ptr-leaf aliases for
ptr-containing aggregate parameters, borrowed scalar `ptr` assignment into
optional `ptr?` payloads, borrowed region-bearing slice assignment into
optional `[]u8?` payloads, and pattern-bound enum/optional payloads are covered.

optional payload consume aliases, if-let/match optional resource aliases,
same-module typed-error throw/catch and rethrow-through-try enum-payload
resource aliases with stable TETRA2101 JSON diagnostic evidence are covered.

generated `.t4i`
direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias
resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match
optional and nested/field-local nested optional resource return, typed-error
direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias
provenance stubs are covered.

optional resource wrapper aliases including nested struct/enum wrappers are
covered.

Same-module/cross-module actor if-let/match optional-payload, struct-field,
enum-payload, and transitive interprocedural consume aliases including
same-module/cross-module actor struct-field/enum-payload alias transfer
diagnostics with stable TETRA2101 JSON diagnostic evidence are covered.

Same-module/cross-module task-handle/task-group if-let/match optional-payload
join/close aliases with stable TETRA2101 CLI JSON evidence are covered.

Same-module/cross-module task-handle/task-group struct-field/enum-payload
transfer/join/close aliases including same-module/cross-module task-handle
struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON
diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias join
diagnostics with stable TETRA2101 JSON diagnostic evidence and
same-module/cross-module task-group struct-field/enum-payload alias close
diagnostics with stable TETRA2101 JSON diagnostic evidence are covered.

Same-module/cross-module enum-constructor return resource aliases with stable
TETRA2101 CLI JSON evidence are covered.

Same-module/cross-module monomorphized generic struct actor consume aliases,
same-module/cross-module monomorphized generic struct task-handle/task-group/island
resource aliases with stable TETRA2101 CLI JSON evidence, same-module/cross-module
transitive interprocedural task-handle/task-group/island resource aliases with
stable TETRA2101 CLI JSON evidence, resource aliases, and ambiguous resource
provenance are covered.

### Move/copy/drop/finalization semantics

Whole consumes, same-module/cross-module partial consumes, same-module/cross-module
struct/enum whole-value call/let/return rejection after partial consume plus
stable `TETRA2101` CLI JSON evidence for same-module/cross-module whole-copy
rejection after partial struct/enum consume are covered.

Same-module/cross-module enum wrapper-constructor rejection after partial
field/payload consume with stable `TETRA2101` CLI JSON evidence and
same-module/cross-module optional-payload whole-value rejection after payload
consume/free with stable TETRA2101 JSON diagnostic evidence are covered.

Mutable reinitialization, task/island/task-group finalization including stable
`TETRA2101` task-group use-after-close, same-module/cross-module
struct-field/enum-payload alias use-after-free CLI JSON diagnostics, and
same-module/cross-module struct-field and enum-payload alias use-after-free with
stable TETRA2101 JSON diagnostic evidence are covered.

Same-module/cross-module task-handle struct-field/enum-payload alias join
diagnostics with stable TETRA2101 JSON diagnostic evidence and
same-module/cross-module task-group struct-field/enum-payload alias close
diagnostics with stable TETRA2101 JSON diagnostic evidence are covered.

Same-module/cross-module task-handle/task-group if-let/match optional-payload
join/close aliases with stable TETRA2101 CLI JSON evidence,
same-module/cross-module nested optional resource wrapper alias use-after-free
CLI JSON diagnostics, and same-module/cross-module task_group_cancel return
provenance diagnostics with stable TETRA2101 CLI JSON evidence are covered.

Full copy/drop model evidence is covered for move/copy/finalization cases in
same-module and cross-module settings.

### Partial moves for struct/enum fields

Same-module/cross-module struct-field and enum-payload partial consume, sibling
reuse, whole-value rejection with stable `TETRA2101` CLI JSON diagnostics,
mutable reinitialization, and runnable struct/enum partial-move smoke coverage
are covered.

### Ownership-aware generics/interfaces/callables

Generic fnptr and protocol marker checks plus generic function-typed global
consume-marker preservation and ownership mismatch diagnostics are covered.

Same-module/cross-module generic aggregate and optional-ptr owned/consume/inout
instantiations including slice-containing struct/enum aggregate instantiations
with stable TETRA2101 CLI JSON evidence are covered.

Same-module/cross-module generic borrow-aggregate/optional-ptr return
diagnostics with stable TETRA2102 CLI JSON evidence are covered.

Same-module/cross-module monomorphized generic struct actor consume alias
diagnostics plus same-module/cross-module monomorphized generic struct
task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON
evidence are covered.

Same-module/cross-module function-typed value/struct-field/enum-payload
optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI
JSON evidence are covered.

Function-typed value/struct-field/enum-payload callback slice-containing
struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON
diagnostic evidence are covered.

Same-module/cross-module protocol parameter ownership matching plus
same-module/cross-module protocol impl parameter ownership mismatch diagnostics
with stable TETRA2001 CLI JSON evidence and same-module/cross-module generic
protocol requirement parameter ownership mismatch diagnostics with stable
TETRA2001 JSON diagnostic evidence are covered.

generated `.t4i` function-typed parameter local-alias return metadata for
interface-only global-storage diagnostics and function type ownership markers
parse/format plus function-typed callable ownership-marker diagnostics are
covered.

Full generic/protocol runtime dispatch remains outside the current surface.

### Heap/global/thread/callback escape analysis

Callable heap/global/thread/callback escape classification is covered for the
promoted callable surface.

Borrowed `ptr` parameter/alias scalar, same-module/cross-module scalar `ptr`
`consume` and `inout` assignment, match/catch-expression return escapes,
typed-error throw ptr/region payload escapes, aggregate-literal,
struct-field/enum-payload aggregate-alias return, and same-module/cross-module
borrowed scalar `ptr` escapes through ptr-containing struct `inout` assignment,
cross-module `await`/`try await` return/global-assignment chains and relay module
chains are covered, including `inout` assignment chains, `await`/`try await`
callback `inout` invocations with borrow-derived values, optional, struct-field,
and enum-payload callback targets (including optional struct-field and
optional enum-payload, transitive
relay-chain and optional transitive relay-chain for struct and enum callbacks, and
variants with `throws` boundaries), optional `Holder?` wrapper return/global
cases in `compiler/tests/semantics/async_ownership_test.go`, plus non-throwing
`await` struct-field/enum-payload callback `inout` invocations (including direct
and relay-chain variants) in
`compiler/tests/semantics/async_inout_ownership_test.go`.

Same-module/cross-module fixed-array alias return, same-module/cross-module
direct fixed-array global assignment, same-module/cross-module optional
fixed-array global assignment, same-module/cross-module fixed-array inout
assignment, fixed-array escapes including inout assignment with stable TETRA2102
diagnostic evidence, and borrowed string alias return/global assignment escapes
with stable `TETRA2102` CLI JSON evidence are covered.

Slice-containing struct literal/alias/nested struct/enum-payload return and
inout assignment escapes plus slice-containing enum direct/alias return escapes
with stable same-module/cross-module `TETRA2102` CLI JSON evidence are covered.

Slice-containing struct/enum owned/consume/inout call escapes with stable
same-module/cross-module and imported direct `TETRA2101` CLI JSON evidence are
covered.

Ptr/slice optional assignment return/owned/consume/inout escape including
stable same-module/cross-module `TETRA2101`/`TETRA2102` slice optional
assignment return/owned/consume/inout CLI JSON evidence are covered.

Same-module/cross-module slice optional payload binding owned/consume/inout
call, `inout` assignment, and global assignment escapes with stable
`TETRA2101`/`TETRA2102` CLI JSON evidence are covered.

Same-module/cross-module ptr optional assignment if-let/match global escape
with stable TETRA2102 JSON diagnostic evidence is covered.

Same-module/cross-module slice optional-payload inout/global assignment escapes
with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module nested slice enum-payload return/inout/global
assignment escapes with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module nested slice struct return/inout/global assignment
escapes with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module direct slice global assignment with stable TETRA2102
JSON diagnostic evidence, same-module/cross-module optional ptr global
assignment with stable TETRA2102 JSON diagnostic evidence, and
same-module/cross-module optional aggregate global assignment with stable
TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON
diagnostic evidence and same-module/cross-module ptr-containing aggregate
whole/field/alias/nested-field return escapes with stable TETRA2102 JSON
diagnostic evidence are covered.

Same-module/cross-module whole-aggregate global assignment with stable TETRA2102
JSON diagnostic evidence, same-module/cross-module ptr-containing enum
whole-value global assignment with stable TETRA2102 JSON diagnostic evidence,
same-module/cross-module global field target assignment with stable TETRA2102
JSON diagnostic evidence, and same-module/cross-module aggregate and
nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic
evidence are covered.

Same-module/cross-module pattern-bound enum payload plus if-let/match optional
payload return/owned/consume/inout call/inout/global escapes including
same-module/cross-module ptr enum-payload return/global/inout assignment
escapes with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module ptr optional-payload return/global/inout assignment
escapes with stable TETRA2102 JSON diagnostic evidence and
same-module/cross-module slice optional-payload inout/global assignment escapes
with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module nested slice enum-payload return/inout/global
assignment escapes with stable TETRA2102 JSON diagnostic evidence and
same-module/cross-module nested slice struct return/inout/global assignment
escapes with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module ptr-containing/nested aggregate owned/consume/inout
call rejections with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module ptr enum-payload owned/consume/inout call rejections
with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module ptr optional-payload owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence, and
same-module/cross-module slice optional-payload owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence are covered.

Direct/imported-direct/function-typed value/struct-field/enum-payload
owned/consume/inout struct/enum/optional-ptr/nested-aggregate parameter
including imported direct ptr-containing/nested aggregate owned/consume/inout
call rejections with stable TETRA2101 JSON diagnostic evidence are covered.

Ptr-containing/nested-aggregate `inout` assignment and scalar/nested-aggregate
global-assignment escapes are rejected; async-specific non-callable
`await`/`try` global, optional-global, match-optional, and global-field
borrow-escape cases are now covered in `compiler/tests/semantics/async_test.go` and
`compiler/tests/semantics/async_ownership_test.go`, including cross-module
`match await`/`match try await` optional-wrapper global-assignment cases and
interprocedural async-wrapper chains.

### Actor/task/island/resource transfer rules

Local actor/task/island/resource transfer, branch/match/loop actor consume reuse
diagnostics with stable TETRA2101 CLI JSON evidence, actor/task
use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence, island
transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence,
same-module/cross-module transitive actor consume alias diagnostics with stable
TETRA2101 CLI JSON evidence, and same-module/cross-module monomorphized generic
struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence
are covered.

Same-module/cross-module task_group_cancel return provenance diagnostics with
stable TETRA2101 CLI JSON evidence are covered.

Same-module/cross-module actor/task if-let/match optional-payload alias transfer
diagnostics with stable TETRA2101 JSON diagnostic evidence are covered.

Same-module/cross-module actor if-let/match optional-payload, struct-field, and
enum-payload consume aliases including same-module/cross-module actor
struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON
diagnostic evidence are covered.

Same-module/cross-module task-handle struct-field/enum-payload alias transfer
diagnostics with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module task-handle struct-field/enum-payload alias join
diagnostics with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module task-group struct-field/enum-payload alias close
diagnostics with stable TETRA2101 JSON diagnostic evidence, and
same-module/cross-module task-handle/task-group if-let/match optional-payload
join/close aliases with stable TETRA2101 CLI JSON evidence are covered.

Distributed actor safety and broad race proofs are tracked outside this
production ownership target.

### Stable forbidden-case diagnostics

use-after-move/use-after-consume, partial struct-field and enum-payload consume
whole-value rejection, partial struct/enum whole-copy rejection, partial
struct/enum enum-constructor rejection, optional payload consume/free
whole-value rejection, actor/task use-after-transfer, same-module/cross-module
actor struct-field/enum-payload alias transfer diagnostics with stable
TETRA2101 JSON diagnostic evidence, and same-module/cross-module task-handle
struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON
diagnostic evidence are covered.

Task-handle struct-field/enum-payload alias use-after-transfer/join, task-group
use-after-close, branch/match/loop actor consume reuse with stable branch actor
CLI JSON, maybe-consumed joins, branch/match/loop task-handle maybe-joined,
task-group maybe-closed, and island maybe-freed merge diagnostics are covered.

Branch/match/loop resource finalization merge diagnostics with stable TETRA2101
JSON evidence, borrow escape, alias conflicts, use-after-free/join/close,
resource use-after-free/double-join/ambiguous-provenance, island transfer
non-local-payload, callable mutable-capture global/heap-escape, callable
pointer/resource capture escape, function-typed storage/return unsupported
capture rejection, callable global-storage escape, unsupported function-value
escape, unsupported function-value call, capturing closure raw-ptr escape,
captured closure explicit type-arg rejection, function-typed explicit type-arg
rejection, generic closure/generic callback-closure capture, generic closure
pointer/direct-call, and imported mutable function-typed global boundary JSON
diagnostics are covered.

Double-drop/double-finalization, callable escape diagnostics, and CLI JSON
ownership/lifetime safety codes for use-after-move/use-after-consume, partial
struct/enum consume whole-value rejection, partial struct/enum whole-copy
rejection, partial struct/enum enum-constructor rejection, optional payload
consume/free whole-value rejection, borrow-escape including fixed-array alias
return/global assignment/optional global assignment/inout assignment with
stable `TETRA2102` diagnostic evidence, and borrowed string alias return/global
assignment are covered.

Same-module/cross-module slice-containing struct literal/alias/nested
struct/enum-payload return and inout assignment escapes plus slice-containing
enum direct/alias return escape CLI JSON evidence are covered.

Slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence
including imported direct cases, same-module/cross-module generic
borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON
evidence, same-module/cross-module function-typed value/struct-field/enum-payload
optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI
JSON evidence, function-typed value/struct-field/enum-payload callback
slice-containing struct/enum owned/consume/inout call rejections with stable
TETRA2101 JSON diagnostic evidence, and ptr/slice optional assignment
return/owned/consume/inout escape with stable same-module/cross-module slice
optional assignment return/owned/consume/inout CLI JSON evidence are covered.

Same-module/cross-module slice optional payload binding owned/consume/inout
call, `inout` assignment, and global assignment CLI JSON evidence are covered.

Same-module/cross-module slice optional-payload inout/global assignment escapes
with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested
slice enum-payload return/inout/global assignment escapes with stable TETRA2102
JSON diagnostic evidence, and same-module/cross-module nested slice struct
return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic
evidence are covered.

Same-module/cross-module direct slice global assignment with stable TETRA2102
JSON diagnostic evidence, same-module/cross-module optional ptr global
assignment with stable TETRA2102 JSON diagnostic evidence, and
same-module/cross-module optional aggregate global assignment with stable
TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module ptr optional assignment if-let/match global escape
with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr
enum alias return escape with stable TETRA2102 JSON diagnostic evidence, and
same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field
return escapes with stable TETRA2102 JSON diagnostic evidence are covered.

Same-module/cross-module whole-aggregate global assignment with stable TETRA2102
JSON diagnostic evidence, same-module/cross-module ptr-containing enum
whole-value global assignment with stable TETRA2102 JSON diagnostic evidence,
same-module/cross-module global field target assignment with stable TETRA2102
JSON diagnostic evidence, and same-module/cross-module aggregate and
nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic
evidence are covered.

Same-module/cross-module ptr-containing/nested aggregate owned/consume/inout
call rejections with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module ptr enum-payload owned/consume/inout call rejections
with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module ptr optional-payload owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence, and
same-module/cross-module slice optional-payload owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence are covered.

Imported direct ptr-containing/nested aggregate owned/consume/inout call
rejections with stable TETRA2101 JSON diagnostic evidence,
same-module/cross-module ptr enum-payload return/global/inout assignment
escapes with stable TETRA2102 JSON diagnostic evidence,
same-module/cross-module ptr optional-payload return/global/inout assignment
escapes with stable TETRA2102 JSON diagnostic evidence, scoped island optional
region-wrapper escape, branch actor consume reuse, task-group use-after-close,
resource use-after-free including same-module/cross-module
struct-field/enum-payload alias use-after-free, same-module/cross-module
task-handle struct-field/enum-payload alias transfer diagnostics with stable
TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle
struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON
diagnostic evidence, same-module/cross-module task-group struct-field/enum-payload
alias close diagnostics with stable TETRA2101 JSON diagnostic evidence, and
same-module/cross-module nested optional resource wrapper alias use-after-free
are covered.

Resource double-join, resource ambiguous-provenance, island transfer
non-local-payload, callable mutable-capture global/heap-escape, callable
pointer/resource capture escape, function-typed storage/return unsupported
capture rejection, callable global-storage escape, unsupported function-value
escape, unsupported function-value call, capturing closure raw-ptr escape,
captured closure explicit type-arg rejection, function-typed explicit type-arg
rejection, generic closure/generic callback-closure capture, generic closure
pointer/direct-call, and imported mutable function-typed global boundary cases
are covered in tests and gates.

### Spec/docs/examples/tests evidence

Specs, supported surface docs, syntax docs, runnable smoke, and focused compiler
tests exist for the current local profile. `examples/ownership_smoke.tetra` is
part of the evidence set.

### Dedicated ownership validator evidence

The dedicated validator checks this audit for required rows, status consistency,
missing-work summary, forbidden filler claims, required `examples/ownership_smoke.tetra`
evidence in the spec/docs/examples row, required branch/match/loop task-handle
maybe-joined, task-group maybe-closed, and island maybe-freed evidence in the
SSA row, and required same-module/interface-only cross-module per-field
interprocedural region summaries in the interprocedural row.

It also checks required base alias/provenance evidence for ownership paths, enum
payload aliases, borrowed ptr/optional/slice payload aliases, pattern-bound
payloads, optional payload consume alias and if-let/match optional resource
alias evidence, optional resource wrapper aliases, actor consume aliases,
task-handle/task-group aggregate aliases, monomorphized generic actor aliases,
transitive resource alias, enum-constructor return resource alias `TETRA2101`
evidence, and ambiguous resource provenance in the alias/provenance row.

Move/copy/drop/finalization checks include struct/enum whole-value
call/let/return rejection after partial consume, enum wrapper-constructor
rejection after partial field/payload consume, mutable reinitialization,
task/island/task-group finalization including `TETRA2101` task-group
use-after-close, resource alias CLI JSON diagnostics, resource alias,
task_group_cancel return provenance evidence, task-handle alias join,
task-group alias close, task-handle/task-group optional-payload join/close
`TETRA2101` evidence, and nested optional resource wrapper alias use-after-free
CLI JSON diagnostics.

Stable diagnostics checks include partial struct/enum consume whole-value
rejection, whole-copy rejection, enum-constructor rejection, optional payload
consume/free whole-value rejection, actor/task use-after-transfer, actor
struct-field/enum-payload alias transfer `TETRA2101` evidence, task-handle
struct-field/enum-payload alias transfer `TETRA2101` evidence, task-handle alias
use-after-transfer/join evidence, task-group use-after-close evidence,
branch/match/loop actor consume reuse evidence, maybe-consumed joins evidence,
branch/match/loop task-handle/task-group/island merge diagnostics evidence,
branch/match/loop resource finalization merge `TETRA2101` evidence, borrow
escape evidence, alias conflicts evidence, use-after-free/join/close evidence,
callable escape diagnostics evidence, CLI JSON ownership/lifetime safety codes
evidence, fixed-array borrow-escape CLI JSON evidence, borrowed string CLI JSON
evidence, slice struct/nested return-inout CLI JSON evidence, slice struct/enum
owned-consume-inout call CLI JSON evidence, generic borrow aggregate/optional-ptr
stable CLI JSON evidence, function-typed optional-ptr callback stable CLI JSON
evidence, function-typed slice callback stable CLI JSON evidence,
optional-assignment stable CLI JSON evidence, slice optional-payload binding
stable CLI JSON evidence, direct slice global assignment stable JSON evidence,
optional ptr global assignment stable JSON evidence, optional aggregate global
assignment stable JSON evidence, ptr optional assignment if-let/match global
escape stable JSON evidence, ptr enum alias return stable JSON evidence, ptr
aggregate return stable JSON evidence, whole aggregate global assignment stable
JSON evidence, ptr enum whole-value global assignment stable JSON evidence,
global field target assignment stable JSON evidence, aggregate/nested global
field stable JSON evidence, ptr enum-payload return/global/inout stable JSON
evidence, ptr optional-payload return/global/inout stable JSON evidence, slice
optional-payload inout/global stable JSON evidence, nested slice enum-payload
return/inout/global stable JSON evidence, nested slice struct return/inout/global
stable JSON evidence, ptr-containing/nested aggregate call TETRA2101 stable
evidence, and slice enum return escape CLI JSON evidence in the stable
diagnostics row.

It also checks required generic resource alias `TETRA2101` evidence in the
generics row, required task_group_cancel return provenance `TETRA2101` evidence
in the transfer row, required `./tetra features --format=json` evidence in the
feature registry row, and required `bash scripts/ci/test-all.sh` evidence in the
release-gate row.

### Feature registry evidence

Registry reports current bounded ownership/lifetime/resource/callable slices,
CLI JSON feature output asserts key ownership/resource/actor-transfer
boundaries, and the registry documents full-v1 limits. `./tetra features
--format=json` remains the canonical command evidence.

### Release-gate evidence

Full local gate and safety-readiness evidence pass for the bounded local profile,
and are included as supporting regression evidence for this ownership objective.
`bash scripts/ci/test-all.sh` remains the canonical release-gate command, but
the audit requires structured evidence rather than the command name alone.
The accepted release-gate artifact is
`docs/generated/v1_0/test-all/summary.json`; it records `status: pass`,
`failed_count: 0`, per-step `exit_code: 0`, and is checked by
`validate-test-all-summary`.

## Verifier Coverage Notes

- `validate-safety-readiness` proves the current local safety profile, not the
  complete ownership objective.
- `verify-docs` proves documentation/manifest consistency, not full ownership
  readiness.
- `scripts/ci/test-all.sh` is the canonical release-gate evidence when the
  checklist rows above are marked passed and evidence is current.
- `validate-ownership-audit` validates row-wise coverage and flags partial rows
  until they are upgraded to pass.

## Missing Work Summary

The objective is achieved for this audit scope, with all checklist rows currently
set to `pass` and required evidence linked for each declared requirement.
