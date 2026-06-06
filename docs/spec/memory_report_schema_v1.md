# Memory Report Schema v1

Status: Memory Production Core v1 schema contract.

`tetra.memory-report.v1` is a serialized projection of compiler-owned memory
facts. It is not a source of truth. The compiler creates and validates facts in
`compiler/internal/memoryfacts`, attaches lowered artifact ids when a storage or
lowering claim is made, then writes the report as reviewable evidence.

Report flags, including `--emit-memory-report`, are artifact-only. They must
not change safe-program semantics, enable optimizations, disable checks, or turn
unknown unsafe memory into safe memory.

## Top-Level Shape

```json
{
  "schema_version": "tetra.memory-report.v1",
  "rows": []
}
```

`rows` must be non-empty for a valid emitted report. Empty in-memory report
values are allowed only before projection or when a caller deliberately has no
program facts to emit.

## Row Fields

Each row describes one projected fact:

| Field | Required | Meaning |
| --- | --- | --- |
| `program_id` | optional | Stable report/program label. |
| `function_id` | optional | Function that owns the fact, when known. |
| `value_id` | optional | PLIR or graph value id the fact describes, when available. |
| `site_id` | yes | Stable source, IR, allocation, or unsafe gateway site id. |
| `source_span` | optional | Human-readable source location. |
| `source_fact_id` | yes | Compiler-owned `MemoryFactGraph` fact id. |
| `parent_fact_id` | optional | Parent fact for derived facts. |
| `lowered_artifact_id` | required for storage/lowering claims | Lowered IR/backend/report artifact that was validated. |
| `source_stage` | yes | Stage that produced or validated the fact. |
| `claim` | yes | Human-readable claim label. |
| `claim_level` | yes | Trust level for the claim. |
| `provenance_class` | yes | Provenance class owned by the compiler fact. |
| `owner_id` | optional | Visible owner/source/destination id for bounded borrow/copy facts. |
| `param_index` | optional | Zero-based parameter index for bounded function summary rows. |
| `param_path` | optional | Field/path suffix under the parameter owner for summary rows. |
| `borrow_state` | optional | Borrow state mirrored from the compiler fact. |
| `escape_state` | optional | Escape state mirrored from the compiler fact. |
| `alias_state` | optional | Alias state mirrored from compiler-owned alias/noalias evidence. |
| `unsafe_class` | yes | Unsafe-origin classification. |
| `allocation_site_id` | optional | Allocation or builtin site id for owned allocation facts. |
| `planned_storage` | required when storage is claimed | Storage selected by the planner. |
| `actual_lowering_storage` | required when storage is claimed | Storage actually emitted by lowering. |
| `validator_name` | optional unless claim is validated | Validator that checked the fact. |
| `validator_status` | yes | Validator result for this projected row. |
| `cost_class` | yes | Memory cost class projected from compiler-owned facts or conservative state. |
| `normal_build_check` | required for `dynamic_check_required` | Whether a required check remains in the normal build. |
| `reason` | optional; required for heap/conservative trusted-storage fallback rows | Reviewable explanation or conservative boundary. |

Known `source_stage` values:

```text
semantics
unsafe_gateway_lowering
plir
allocplan
lowering
validation
```

Known `claim_level` values:

```text
validated
evidence_only
conservative
rejected
future
```

Known `provenance_class` values:

```text
safe_known
safe_borrowed
safe_owned
unsafe_unknown
unsafe_checked
unsafe_verified_root
```

Known `unsafe_class` values:

```text
safe
unsafe_unknown
unsafe_checked
unsafe_verified_root
```

Known `validator_status` values:

```text
pass
fail
not_applicable
not_run
```

Known `cost_class` values:

```text
zero_cost_proven
dynamic_check_required
instrumentation_only
unsupported_rejected
conservative_fallback
```

Known `alias_state` values:

```text
unique
shared_readonly
mutable_exclusive
maybe_alias
unknown_alias
invalidated_by_call
```

## Storage Classes

The v1 validator accepts the bounded storage vocabulary currently implemented
by `compiler/internal/memoryfacts`:

```text
UnknownConservative
Eliminated
Register
Heap
Stack
Region
ExplicitIsland
FunctionTempRegion
TaskRegion
ActorMoveRegion
LargeMmap
External
```

Later MPC slices may add request-region or target-specific storage vocabulary
only with validator tests and manifest-visible schema notes.

## Required Rejections

`tools/cmd/validate-memory-report` and
`compiler/internal/memoryfacts.ValidateReport` reject:

- missing or unknown `schema_version`;
- empty `rows`;
- trailing data after the first JSON report value;
- any row without `source_fact_id`, `site_id`, or `claim`;
- duplicate `source_fact_id` rows;
- unknown enum values;
- missing or unknown `cost_class`;
- `dynamic_check_required` rows without `normal_build_check`;
- negative `param_index` values;
- `claim_level: validated` with `validator_status` other than `pass`;
- `claim_level: validated` without `validator_name`;
- `safe_known`, `safe_borrowed`, or `safe_owned` provenance paired with
  `unsafe_unknown`;
- `unsafe_unknown` rows that claim `safe_known`, `safe_borrowed`,
  `safe_owned`, `no_alias`, `index_in_range`, `bounds_check_eliminated`,
  `provenance_known`, or that carry trusted alias states such as `unique` or
  `mutable_exclusive`;
- `unsafe_unknown` rows that claim `zero_cost_proven`;
- proof-id removed-check rows or raw-bounds normal-build rows without their
  `parent_fact_id` source/proof link;
- FFI/external derived rows such as `ffi_call_may_retain_borrow`,
  `ffi_noalias_invalidated_by_external_call`,
  `safe_wrapper_promotion_rejected_without_contract`, or
  `external_pointer_provenance_rejected` without their `parent_fact_id`
  source/proof link;
- `unsafe_verified_root` rows that claim generic provenance/lifetime facts
  instead of bounded raw metadata such as `allocation_base_metadata` or
  `unsafe_verified_root_allocation_base`;
- validated `unsafe_unknown` rows that claim trusted stack/region/island,
  register, or eliminated storage lowering;
- storage rows missing either `planned_storage` or `actual_lowering_storage`;
- storage or lowering claims without `lowered_artifact_id`;
- heap or conservative trusted-storage fallback rows without a reviewable
  `reason`; report rows still require `source_fact_id` so the fallback remains
  traceable to a compiler-owned `MemoryFactGraph` fact;
- validated rows whose trusted non-heap `planned_storage` lowers as `Heap`.
  This includes `Eliminated`, `Register`, `Stack`, `Region`,
  `ExplicitIsland`, `FunctionTempRegion`, `TaskRegion`, and
  `ActorMoveRegion`. For example, a validated row with
  `planned_storage: FunctionTempRegion` and `actual_lowering_storage: Heap`
  is invalid even when the planner recorded a function-local temporary-region
  candidate;
- validated `no_alias` rows whose `alias_state` is not `unique` or
  `mutable_exclusive`.
- Memory Ideal Vertical Slice v0 rejects broad noalias wording such as
  `broad_noalias`, `universal_noalias`, or `full_noalias_model`.
- Derived borrow/copy/inout rows such as `borrow_owner`,
  `borrow_source_fact_id`, `aggregate_contains_borrow`,
  `optional_contains_borrow`, `enum_payload_contains_borrow`,
  `generic_wrapper_contains_borrow`, `function_value_contains_borrow`,
  `callback_arg_contains_borrow`, `callback_inout_conservative`,
  `interface_value_contains_borrow`,
  `protocol_dispatch_borrow_conservative`,
  `protocol_dispatch_noalias_conservative`,
  `async_boundary_borrow_conservative`,
  `task_boundary_borrow_rejected`,
  `actor_boundary_borrow_rejected`,
  `boundary_noalias_conservative`,
  `pre_await_local_borrow_validated`,
  `post_await_borrow_conservative`,
  `cancellation_borrow_lifetime_invalidated`,
  `task_group_noalias_conservative`,
  `actor_reentrant_callback_conservative`,
  `dynamic_existential_borrow_conservative`,
  `static_witness_borrow_parent_validated`,
  `dynamic_protocol_noalias_rejected`,
  `witness_provenance_promotion_rejected`,
  `protocol_dispatch_report_integrity`,
  `unsafe_unknown_rejected_safe_facts`,
  `unsafe_verified_root_allocation_base`,
  `ffi_call_may_retain_borrow`,
  `ffi_noalias_invalidated_by_external_call`,
  `safe_wrapper_promotion_rejected_without_contract`,
  `external_pointer_provenance_rejected`,
  `copy_owned`, `copy_source_fact_id`,
  `mutable_exclusive`, `start_inout_exclusive`, `end_inout_exclusive`,
  `no_alias_validated_narrow_unique_local`, and
  `no_alias_validated_narrow_sequential_inout` require `parent_fact_id`.
- `copy_owned` requires `provenance_class: safe_owned`.

These checks keep reports from reconstructing truth that the compiler never
owned. A row that does not reference a source fact id is invalid even if the
human-readable claim looks plausible.

## Graph Projection Integrity

Memory Ideal Vertical Slice v8 adds an explicit graph/report projection gate.
`compiler/internal/memoryfacts.ValidateReportProjection` validates a
`tetra.memory-report.v1` value against the compiler-owned `MemoryFactGraph`
that produced it. The graph remains the source of truth; the report remains a
projection.

The projection validator rejects:

- a report row whose `source_fact_id` does not exist in the graph;
- a graph fact projected by `BuildReportFromGraph` that is missing from the
  report;
- altered source, parent, validator, claim-level, provenance, alias, borrow,
  escape, storage, artifact, or reason fields;
- changed `cost_class`;
- changed `normal_build_check`;
- v8 correlation rows that add, omit, or widen `MEM-REPORT-*` evidence;
- memory audit/release wording that turns conservative or rejected rows into a
  broad validated safety claim such as "Memory 100%".

This is an integrity check only. It does not add new memory semantics, enable
optimizations, prove target parity, or make unsafe/external pointers safe.

## Conservative Unknown Raw Pointer Row

Unknown external raw pointers must remain conservative and must not produce
`safe_known` facts:

```json
{
  "program_id": "program",
  "function_id": "main",
  "site_id": "raw:1:1",
  "source_fact_id": "fact:raw:unknown",
  "source_stage": "plir",
  "claim": "checked_external_unknown",
  "claim_level": "conservative",
  "provenance_class": "unsafe_unknown",
  "unsafe_class": "unsafe_unknown",
  "cost_class": "conservative_fallback",
  "validator_name": "memory_report_schema_v1",
  "validator_status": "not_applicable",
  "reason": "unknown raw pointer remains conservative"
}
```

## Verified `core.alloc_bytes` Root Row

Verified roots from known Tetra allocation mechanisms may carry bounded
metadata, but they remain unsafe-origin facts until a checked safe wrapper
converts them:

```json
{
  "program_id": "program",
  "function_id": "main",
  "site_id": "alloc:main:1:1",
  "source_fact_id": "fact:raw:root",
  "lowered_artifact_id": "ir:main:alloc_bytes:0",
  "source_stage": "validation",
  "claim": "allocation_base_metadata",
  "claim_level": "validated",
  "provenance_class": "unsafe_verified_root",
  "unsafe_class": "unsafe_verified_root",
  "planned_storage": "Heap",
  "actual_lowering_storage": "Heap",
  "cost_class": "zero_cost_proven",
  "validator_name": "raw_bounds_validator",
  "validator_status": "pass",
  "reason": "verified core.alloc_bytes root"
}
```

Memory Ideal v5 adds a derived allocation-base projection for the same
verified root:

```json
{
  "program_id": "program",
  "function_id": "main",
  "site_id": "alloc:main:1:1",
  "source_fact_id": "fact:raw:root:unsafe_verified_root_allocation_base",
  "parent_fact_id": "fact:raw:root",
  "source_stage": "allocplan",
  "claim": "unsafe_verified_root_allocation_base",
  "claim_level": "validated",
  "provenance_class": "unsafe_verified_root",
  "unsafe_class": "unsafe_verified_root",
  "cost_class": "zero_cost_proven",
  "validator_name": "unsafe_verified_root_bounds_validator",
  "validator_status": "pass",
  "reason": "core.alloc_bytes verified root may project bounded allocation-base metadata"
}
```

This row is still unsafe-origin evidence. It does not grant safe provenance,
safe wrappers, noalias, lifetime, region, or arbitrary external pointer safety.

## Safe Representation Metadata Row

MPC-3 adds a validated summary row for the supported representation namespace
invariant. This row states that slice/String representation metadata is not
user-assignable state:

```json
{
  "program_id": "program",
  "site_id": "semantics:representation-metadata",
  "source_fact_id": "semantics:representation-metadata:not-user-assignable",
  "source_stage": "semantics",
  "claim": "safe_representation_metadata: not_user_assignable",
  "claim_level": "validated",
  "provenance_class": "safe_known",
  "unsafe_class": "safe",
  "validator_name": "representation_namespace_validator",
  "validator_status": "pass",
  "reason": "slice/String representation metadata is not user-assignable state"
}
```

The row does not claim named lifetimes, generic lifetime parameters, arbitrary
borrowed aggregate returns, or safety for arbitrary unsafe external pointers.

## Borrow/Copy Rows

MPC-4 adds bounded projection rows for the supported safe slice/String
byte-view surface. These rows mirror PLIR and allocation-plan facts; they do
not introduce report-only ownership truth:

```text
borrowed_imm
no_escape
borrow_owner
borrow_source_fact_id
copy_owned
copy_source_fact_id
copy_into_destination_fact_id
```

`borrow_owner` and `borrow_source_fact_id` are derived only from safe borrowed
facts. If a borrowed value has `unsafe_unknown` provenance, the report may keep
the `borrowed_imm` row as `claim_level: conservative`, but it must not emit
safe owner/source rows for that value. `copy_owned` records that `copy()`
created independently owned storage and new provenance. `copy_into_destination_fact_id`
records the caller-owned destination relation for `copy_into(dst)` and is not
an allocation or storage claim.

## Alias/Inout Rows

MPC-5 adds bounded projection rows for the conservative mutable alias/inout
subset:

```text
no_alias
mutable_exclusive
start_inout_exclusive
end_inout_exclusive
```

`no_alias` rows for supported `inout` parameters carry
`alias_state: mutable_exclusive` and remain evidence rows unless a later
validator explicitly proves them. `start_inout_exclusive` and
`end_inout_exclusive` document the call-duration exclusivity window for the
parameter. Unknown, maybe, or call-invalidated alias state must not be promoted
to a validated `no_alias` claim. This does not claim a universal mutable alias
model, generic wrapper completeness, or full Rust-like aliasing.

Memory Ideal Vertical Slice v0 adds only narrow validated noalias projection
names:

```text
no_alias_validated_narrow_unique_local
no_alias_validated_narrow_sequential_inout
```

These rows are derived from compiler-owned `FactNoAlias` facts, require
`parent_fact_id`, use `validator_name: alias_interval_validator`, and carry
`alias_state: mutable_exclusive` or `unique`. They do not claim disjoint slice
windows, callback/reentrant safety, raw-pointer interference safety, async
safety, concurrency safety, or a broad mutable alias model.

## Memory Ideal Vertical Slice v0 Rows

The v0 slice projects the minimal correlated rows needed for
`MEM-REP-001`, `MEM-BORROW-001`, and `MEM-ALIAS-001`:

```text
safe_representation_metadata: not_user_assignable
aggregate_contains_borrow
optional_contains_borrow
no_alias_validated_narrow_unique_local
no_alias_validated_narrow_sequential_inout
```

`aggregate_contains_borrow` and `optional_contains_borrow` are derived only
from safe borrowed PLIR facts, require a parent fact, require a visible owner,
use `validator_name: borrow_aggregate_escape_validator`, and remain limited to
simple struct fields and optional payloads. They do not claim enum payload,
generic wrapper, interface, callable, async, actor/task, raw pointer, or
target-parity borrow closure. Enum payload and generic wrapper closure are
represented by the separate v1 rows below.

## Memory Ideal Vertical Slice v1 Rows

The v1 slice projects only the two correlated rows needed for
`MEM-BORROW-002` and `MEM-BORROW-003`:

```text
enum_payload_contains_borrow
generic_wrapper_contains_borrow
```

`enum_payload_contains_borrow` and `generic_wrapper_contains_borrow` are derived
only from safe borrowed PLIR facts, require a parent fact, require a visible
owner, use `validator_name: borrow_aggregate_escape_validator`, and remain
limited to direct enum payload carriers and monomorphized generic struct
wrappers. They do not claim interface, callable, callback, async, actor/task,
raw pointer, target-parity, or broad noalias closure.

## Memory Ideal Vertical Slice v2 Rows

The v2 slice projects only the three correlated rows needed for
`MEM-BORROW-004`, `MEM-BORROW-005`, and `MEM-ALIAS-002`:

```text
function_value_contains_borrow
callback_arg_contains_borrow
callback_inout_conservative
```

`function_value_contains_borrow` and `callback_arg_contains_borrow` are derived
only from safe borrowed PLIR facts, require a parent fact, require a visible
owner, and use `validator_name: function_value_borrow_escape_validator` or
`callback_borrow_escape_validator`. They remain limited to function-typed local
values, function-typed struct fields, already-supported function-typed enum
payloads, callback parameters, and known direct callback targets. Unknown
callback targets remain conservative and must not emit trusted borrow facts.

`callback_inout_conservative` is derived only from callback/reentrant `inout`
alias evidence, requires a parent fact, uses `validator_name:
callback_alias_conservative_validator`, carries `alias_state:
invalidated_by_call`, and projects `cost_class: conservative_fallback`. It does
not grant validated `no_alias`, broad noalias, callback/reentrant noalias,
raw-pointer interference safety, async safety, concurrency safety, or a full
callable memory model.

## Memory Ideal Vertical Slice v3 Rows

The v3 slice projects only the three correlated rows needed for
`MEM-BORROW-006`, `MEM-BORROW-007`, and `MEM-ALIAS-003`:

```text
interface_value_contains_borrow
protocol_dispatch_borrow_conservative
protocol_dispatch_noalias_conservative
```

`interface_value_contains_borrow` is derived only from safe borrowed PLIR facts,
requires a parent fact, requires a visible owner, and uses `validator_name:
interface_borrow_escape_validator`. It remains limited to checker/PLIR-visible
interface/protocol-like values with statically known concrete targets and does
not claim runtime protocol values or full existential containers.

`protocol_dispatch_borrow_conservative` is derived for unknown dynamic protocol
dispatch borrow evidence, requires a parent fact, uses `validator_name:
protocol_dispatch_borrow_validator`, and projects `cost_class:
conservative_fallback`. It must not become a trusted lifetime-safe borrow fact
unless the target is statically known.

`protocol_dispatch_noalias_conservative` is derived for protocol/interface
dispatch alias evidence, requires a parent fact, uses `validator_name:
protocol_dispatch_alias_conservative_validator`, carries `alias_state:
invalidated_by_call`, and projects `cost_class: conservative_fallback`. It does
not grant validated `no_alias`, broad noalias, runtime dispatch noalias, raw
pointer interference safety, async safety, concurrency safety, or a full
interface/protocol memory model.

## Memory Ideal Vertical Slice v4 Rows

The v4 slice projects only the four correlated rows needed for
`MEM-BORROW-008`, `MEM-BORROW-009`, `MEM-BORROW-010`, and `MEM-ALIAS-004`:

```text
async_boundary_borrow_conservative
task_boundary_borrow_rejected
actor_boundary_borrow_rejected
boundary_noalias_conservative
```

`async_boundary_borrow_conservative` is derived for borrowed views that may
cross an async/await suspension boundary, requires a parent fact, uses
`validator_name: async_boundary_borrow_validator`, and projects `cost_class:
conservative_fallback`. It must not become a trusted lifetime-safe borrow fact
unless a later checker path proves the view stays local and non-escaping before
suspension.

`task_boundary_borrow_rejected` and `actor_boundary_borrow_rejected` are derived
for borrowed views crossing task or actor boundaries without explicit copy,
require parent facts, use `validator_name: task_boundary_borrow_validator` or
`actor_boundary_borrow_validator`, and project `cost_class:
unsupported_rejected` with rejected validator status. Explicit `.copy()` creates
owned provenance through the existing `copy_owned` path.

`boundary_noalias_conservative` is derived for task/actor boundary alias
evidence, requires a parent fact, uses `validator_name:
boundary_alias_conservative_validator`, carries `alias_state:
invalidated_by_call`, and projects `cost_class: conservative_fallback`. It does
not grant validated `no_alias`, broad task/actor noalias, async/task/actor
runtime safety, distributed actor memory safety, target parity, or performance.

## Memory Ideal Vertical Slice v5 Rows

The v5 slice projects only the four correlated rows needed for
`MEM-UNSAFE-001`, `MEM-UNSAFE-002`, `MEM-UNSAFE-003`, and
`MEM-UNSAFE-004`:

```text
unsafe_unknown_rejected_safe_facts
unsafe_verified_root_allocation_base
unsafe_contract_runtime_checkable
unsafe_contract_static_untrusted
```

`unsafe_unknown_rejected_safe_facts` is derived from an `unsafe_unknown`
raw-pointer parent fact, requires `parent_fact_id`, uses `validator_name:
unsafe_unknown_fact_validator`, and projects `claim_level: rejected` with
`cost_class: unsupported_rejected`. It records that unknown raw pointers cannot
produce `safe_known`, `provenance_known`, or `no_alias` facts.

`unsafe_verified_root_allocation_base` is derived from validated
`core.alloc_bytes` allocation-base metadata, requires `parent_fact_id`, uses
`validator_name: unsafe_verified_root_bounds_validator`, and projects
`claim_level: validated`. It is still `unsafe_verified_root` evidence and does
not promote the pointer to safe provenance.

`unsafe_contract_runtime_checkable` may validate only runtime-checkable
nonnull, alignment, and length/bounds contracts. It uses `validator_name:
unsafe_runtime_contract_validator`, projects `cost_class:
dynamic_check_required`, and requires `normal_build_check` when the check
remains in a normal build.

`unsafe_contract_static_untrusted` records unsafe noalias, lifetime, or region
contracts that cannot be runtime-validated in this slice. It uses
`validator_name: unsafe_static_contract_validator`, carries `alias_state:
invalidated_by_call` when noalias is involved, and remains conservative rather
than validated. It does not grant safe lifetime, safe region, validated
noalias, broad unsafe noalias, FFI lifetime safety, safe wrapper promotion,
target parity, or performance.

## Memory Ideal Vertical Slice v6 Bounds Rows

The v6 slice projects only the four correlated rows needed for
`MEM-BOUNDS-001`, `MEM-BOUNDS-002`, `MEM-BOUNDS-003`, and
`MEM-BOUNDS-004`:

```text
bounds_check_retained_dynamic
bounds_check_removed_with_proof_id
bounds_check_removal_rejected_missing_proof_id
raw_bounds_runtime_check_normal_build
```

`bounds_check_retained_dynamic` records that no compiler-owned proof was
available for a bounds check, so the check remains in the normal build. It uses
`validator_name: normal_build_bounds_check_validator`, projects
`cost_class: dynamic_check_required`, and requires `normal_build_check`.

`bounds_check_removed_with_proof_id` is derived from compiler-owned proof-id
evidence, requires `parent_fact_id`, uses `validator_name:
bounds_proof_id_validator`, and projects `claim_level: validated` with
`cost_class: zero_cost_proven`. The parent fact links the lowered removed
check back to PLIR proof guards or equivalent compiler-owned proof metadata.

`bounds_check_removal_rejected_missing_proof_id` records that a removed bounds
check without a compiler-owned proof id is rejected. It uses `validator_name:
bounds_proof_id_validator`, `validator_status: fail`, and `cost_class:
unsupported_rejected`.

`raw_bounds_runtime_check_normal_build` is derived from unsafe-checked raw
bounds gateway evidence, requires `parent_fact_id`, uses `validator_name:
raw_bounds_width_validator`, and keeps `normal_build_check` with
`cost_class: dynamic_check_required`. It is not a proof of arbitrary unsafe
pointer arithmetic, target parity, or performance.

## Memory Ideal Vertical Slice v7 FFI Rows

The v7 slice projects only the four correlated rows needed for `MEM-FFI-001`,
`MEM-FFI-002`, `MEM-FFI-003`, and `MEM-FFI-004`:

```text
ffi_pointer_external_unknown
ffi_call_may_retain_borrow
safe_wrapper_promotion_rejected_without_contract
ffi_noalias_invalidated_by_external_call
```

The supporting rejected row
`external_pointer_provenance_rejected` may be emitted to make the provenance
promotion rejection explicit, but it is not an additional correlation row.

`ffi_pointer_external_unknown` records that a raw or external pointer at an FFI
boundary remains `unsafe_unknown` / external-unknown evidence. It uses
`validator_name: external_pointer_provenance_validator` and projects
`claim_level: conservative` with `cost_class: conservative_fallback`.

`ffi_call_may_retain_borrow` is derived from a borrowed parent fact passed to
an external call, requires `parent_fact_id`, uses `validator_name:
ffi_lifetime_conservative_validator`, and projects `claim_level:
conservative` with `cost_class: conservative_fallback`.

`safe_wrapper_promotion_rejected_without_contract` is derived from raw or
external unsafe parent evidence, requires `parent_fact_id`, uses
`validator_name: safe_wrapper_promotion_validator`, and projects
`claim_level: rejected` with `cost_class: unsupported_rejected`.

`ffi_noalias_invalidated_by_external_call` is derived from noalias parent
evidence at an external call, requires `parent_fact_id`, uses
`validator_name: ffi_noalias_conservative_validator`, carries `alias_state:
invalidated_by_call`, and remains conservative rather than validated.

`external_pointer_provenance_rejected` is derived from external/unknown
provenance evidence, requires `parent_fact_id`, uses `validator_name:
external_pointer_provenance_validator`, and projects `claim_level: rejected`.
These rows do not grant arbitrary external pointer safety, C/FFI lifetime
safety, safe wrapper promotion, broad unsafe noalias, target parity,
performance, or arbitrary external allocator provenance.

## Memory Ideal Vertical Slice v10 Async Cancellation Rows

The v10 slice projects only the five correlated rows needed for
`MEM-ASYNC-001`, `MEM-ASYNC-002`, `MEM-ASYNC-003`, `MEM-ASYNC-004`, and
`MEM-ASYNC-005`:

```text
pre_await_local_borrow_validated
post_await_borrow_conservative
cancellation_borrow_lifetime_invalidated
task_group_noalias_conservative
actor_reentrant_callback_conservative
```

`pre_await_local_borrow_validated` is derived from a safe borrowed parent fact
only when compiler-visible evidence shows the borrow is local, non-escaping,
and used before suspension. It requires `parent_fact_id`, uses
`validator_name: pre_await_local_borrow_validator`, and projects
`claim_level: validated` with `cost_class: zero_cost_proven`.

`post_await_borrow_conservative` is derived from a borrowed parent fact that
crosses or is used after await/suspend. It requires `parent_fact_id`, uses
`validator_name: post_await_borrow_conservative_validator`, and remains
conservative rather than validated.

`cancellation_borrow_lifetime_invalidated` is derived from task-owned borrowed
lifetime evidence on a cancellation path. It requires `parent_fact_id`, uses
`validator_name: cancellation_lifetime_invalidation_validator`, and projects
`claim_level: rejected` with `cost_class: unsupported_rejected`.

`task_group_noalias_conservative` is derived from noalias evidence at a task
group or structured concurrency boundary. It requires `parent_fact_id`, uses
`validator_name: task_group_boundary_conservative_validator`, carries
`alias_state: invalidated_by_call`, and remains conservative.

`actor_reentrant_callback_conservative` is derived from actor reentrant
callback borrow evidence. It requires `parent_fact_id`, uses
`validator_name: actor_reentrant_callback_boundary_validator`, and keeps the
borrow/storage relationship conservative unless a later narrow proof exists.

These rows do not grant production actor runtime proof, distributed actor
memory safety, a full async lifetime system, complete structured concurrency
proof, target parity, performance, broad noalias, arbitrary FFI/runtime
lifetime proof, arbitrary external pointer safety, or "Memory 100%".

## Memory Ideal Vertical Slice v11 Dynamic Protocol Rows

The v11 slice projects only the five correlated rows needed for
`MEM-DYNPROTO-001`, `MEM-DYNPROTO-002`, `MEM-DYNPROTO-003`,
`MEM-DYNPROTO-004`, and `MEM-DYNPROTO-005`:

```text
dynamic_existential_borrow_conservative
static_witness_borrow_parent_validated
dynamic_protocol_noalias_rejected
witness_provenance_promotion_rejected
protocol_dispatch_report_integrity
```

`dynamic_existential_borrow_conservative` is derived from safe borrowed parent
evidence carried by a dynamic existential or protocol surface. It requires
`parent_fact_id`, uses `validator_name:
dynamic_existential_borrow_conservative_validator`, and remains conservative
unless a compiler-owned static resolution proof exists.

`static_witness_borrow_parent_validated` is derived only from a safe borrowed
parent fact and static witness/conformance proof. It requires
`parent_fact_id`, uses `validator_name: static_witness_parent_fact_validator`,
and projects `claim_level: validated` with `cost_class: zero_cost_proven`.

`dynamic_protocol_noalias_rejected` is derived from dynamic protocol dispatch
noalias evidence. It requires `parent_fact_id`, uses `validator_name:
dynamic_protocol_noalias_rejection_validator`, carries `alias_state:
invalidated_by_call`, and projects `claim_level: rejected` with `cost_class:
unsupported_rejected`.

`witness_provenance_promotion_rejected` is derived from witness/conformance
lookup evidence with unsafe or unknown provenance. It requires
`parent_fact_id`, uses `validator_name:
witness_provenance_promotion_validator`, and rejects promotion to `safe_known`.

`protocol_dispatch_report_integrity` is derived from protocol/existential
dispatch report integrity evidence. It requires `parent_fact_id`, uses
`validator_name: protocol_dispatch_report_integrity_validator`, projects
`cost_class: dynamic_check_required`, and requires `normal_build_check` so the
projection cannot drop reviewable runtime/check evidence.

These rows do not grant full trait-object/existential runtime proof, complete
witness-table ABI safety proof, production dynamic dispatch runtime safety,
target parity, performance, broad noalias, arbitrary unsafe/external pointer
promotion, or "Memory 100%".

## Function Summary Rows

MPC-6 adds bounded provenance/resource summary rows for PLIR-visible function
evidence and checker-provided summary metadata:

```text
returns_owned_new_allocation
returns_borrow_from_param
returns_unknown_unsafe
may_store_global
may_escape_to_actor
may_escape_to_task
may_capture_in_closure
may_retain_pointer
may_return_region
may_return_resource
may_throw_resource
may_consume_param
may_mutate_inout
requires_effects
requires_capabilities
unknown_external_call_conservative
```

These rows are compiler-owned facts, but they are not validation proofs by
themselves. Safe summary rows project as `claim_level: evidence_only` unless a
later validator marks a narrower fact validated. Unknown external calls,
unknown returned resources, unknown unsafe returns, and retained pointer
summaries project with `provenance_class: unsafe_unknown`,
`unsafe_class: unsafe_unknown`, and `claim_level: conservative`.

`returns_borrow_from_param` is emitted only when the returned PLIR value has
safe parameter provenance or supported checker return-summary metadata. It may
include `param_index` and `param_path` when that owner/path is available from
compiler metadata. `may_return_region`, `may_return_resource`, and
`may_throw_resource` rows use the same bounded parameter/path fields where the
checker summary exposes them. `returns_owned_new_allocation` is emitted for
returned owned allocation intents such as supported `copy()` and `make_*`
paths. The report must not use these rows to promote unknown unsafe memory,
arbitrary FFI lifetimes, or universal generic lifetime behavior.

## Unsafe Gateway Rows

MPC-7 keeps unsafe gateway classes explicit. Supported raw gateway projection
uses these bounded claims:

```text
allocation_base_metadata
derived_allocation_offset
rejected_negative_offset
rejected_upper_bound
rejected_access_width_overflow
rejected_negative_length
rejected_length_overflow
checked_external_unknown
external_unknown
raw_memory_access_checked
raw_memory_access_unknown
raw_slice_verified_allocation_root
unsafe_unknown_rejected_safe_facts
unsafe_verified_root_allocation_base
unsafe_contract_runtime_checkable
unsafe_contract_static_untrusted
raw_bounds_runtime_check_normal_build
ffi_pointer_external_unknown
ffi_call_may_retain_borrow
safe_wrapper_promotion_rejected_without_contract
ffi_noalias_invalidated_by_external_call
external_pointer_provenance_rejected
```

`allocation_base_metadata` and `unsafe_verified_root_allocation_base` are the
only `unsafe_verified_root` rows in schema-v1 reports. `derived_allocation_offset`,
`raw_memory_access_checked`,
`rejected_negative_offset`, `rejected_upper_bound`, and
`rejected_access_width_overflow`, `rejected_negative_length`,
`rejected_length_overflow`, and `raw_slice_verified_allocation_root` remain
`unsafe_checked` evidence, not safe provenance. `checked_external_unknown`,
`external_unknown`, and `raw_memory_access_unknown` remain conservative
`unsafe_unknown` rows. `unsafe_unknown_rejected_safe_facts` is rejected evidence
that prevents safe/noalias promotion from those unknown rows.
`raw_bounds_runtime_check_normal_build` is a v6 dynamic-check row derived only
from unsafe-checked raw bounds evidence; it must keep `normal_build_check` and
does not authorize zero-cost check elimination.
The v7 FFI rows keep external pointers, borrow retention, safe-wrapper
promotion, and noalias evidence conservative or rejected at external-call
boundaries unless a compiler-owned proof exists.

## Validation Command

```bash
go run ./tools/cmd/validate-memory-report --report path/to/file.memory.json
```

For compiler integration, use `tetra build --emit-memory-report` with the
normal build target and inspect the sibling `.memory.json` artifact.
