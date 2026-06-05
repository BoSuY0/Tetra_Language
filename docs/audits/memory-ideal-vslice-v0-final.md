# Memory Ideal Vertical Slice v0 Final Audit

Status: validated_narrow

This audit closes the immediate v0 slice from the Memory Ideal Track v2 plan.
It is not a "Memory 100%" claim. `MemoryFactGraph` remains the truth source;
reports remain projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-REP-001 | validated | `compiler/internal/semantics/representation_metadata.go` reserves `ptr`, `len`, `owner_id`, `region_id`, `provenance_id`, `borrow_source`, `storage_class`, and `unsafe_class`; `compiler/internal/memoryfacts/from_plir.go` emits `semantics:representation-metadata:not-user-assignable`; focused semantics and memoryfacts tests cover rejection/projection. | Safe metadata is compiler-owned and not user-assignable before lowering. |
| MEM-BORROW-001 | validated_narrow | `compiler/tests/semantics/borrow_copy_test.go` covers local struct/optional use, copied struct return, copied optional store, escape rejection, branch-owner rejection, and unsafe-unknown rejection; `compiler/internal/memoryfacts/from_plir_test.go` proves `aggregate_contains_borrow` and `optional_contains_borrow` derived rows with parent facts. | Only simple struct fields and optional payloads with one visible owner are in scope. |
| MEM-ALIAS-001 | validated_narrow | `compiler/tests/ownership/ownership_test.go` covers local and sequential inout plus alias rejection; `compiler/internal/plir/plir_test.go` keeps raw/callback/branch-joined cases conservative; `compiler/internal/memoryfacts/from_plir_test.go` proves narrow noalias report rows with `alias_interval_validator`. | Only unique local and sequential inout intervals are validated. |

## Active Inout Interval

For this slice, the exclusive interval starts after argument evaluation
succeeds and ends at normal call return. Sequential calls are valid because the
previous interval has ended. Alias reads/writes during an active interval are
rejected by existing ownership alias checks, and unknown/raw/callback/branch
cases stay conservative or unsupported rather than becoming noalias evidence.

## Minimal Report Projection

Projected v0 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `safe_representation_metadata: not_user_assignable` | semantics | `representation_namespace_validator` | MEM-REP-001 summary fact. |
| `aggregate_contains_borrow` | plir | `borrow_aggregate_escape_validator` | Derived from safe borrowed aggregate parent fact. |
| `optional_contains_borrow` | plir | `borrow_aggregate_escape_validator` | Derived from safe borrowed optional payload parent fact. |
| `no_alias_validated_narrow_unique_local` | plir | `alias_interval_validator` | Derived from narrow `FactNoAlias`. |
| `no_alias_validated_narrow_sequential_inout` | plir | `alias_interval_validator` | Derived from narrow `FactNoAlias`. |

Validators reject broad noalias rows, derived rows without `parent_fact_id`,
safe claims from `unsafe_unknown`, `copy_owned` without `safe_owned`
provenance, and validated noalias rows without `unique` or
`mutable_exclusive` alias state.

## Nonclaims

- No generic borrow closure.
- No enum payload borrow closure.
- No interface/callable borrow closure.
- No full mutable alias model.
- No raw pointer expansion.
- No actor/task expansion.
- No target parity.
- No performance claim.
- No named lifetimes, full Rust-like borrow checker, async/concurrency memory
  model, or perfect memory claim.
