# Memory Ideal Vertical Slice v1 Final Audit

Status: validated_narrow

This audit closes the enum payload and monomorphized generic wrapper borrow
carrier slice. It extends the v0 struct/optional pattern without claiming a
complete memory system. `MemoryFactGraph` remains the truth source; reports
remain projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-BORROW-002 | validated_narrow | `compiler/tests/semantics/borrow_copy_test.go` covers enum payload owned-return rejection, global-storage rejection, local use, and `.copy()` escape; `compiler/internal/memoryfacts/from_plir_test.go` proves `enum_payload_contains_borrow` derived rows with parent facts; `compiler/internal/memorymodel/mini_test.go` covers enum payload local, return, copy, mixed-owner, and unsafe-unknown cases. | Only direct enum payload carriers whose payload expression and owner are PLIR/checker visible are in scope. |
| MEM-BORROW-003 | validated_narrow | `compiler/tests/semantics/borrow_copy_test.go` covers monomorphized generic wrapper owned-return rejection, global-storage rejection, local use, and `.copy()` escape; `compiler/internal/memoryfacts/from_plir_test.go` proves `generic_wrapper_contains_borrow` derived rows with parent facts; `compiler/internal/memorymodel/mini_test.go` covers generic wrapper store, copy, and unsafe-unknown cases. | Only monomorphized generic struct wrappers with direct payload fields are in scope. |

## Minimal Report Projection

Projected v1 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `enum_payload_contains_borrow` | plir | `borrow_aggregate_escape_validator` | Derived from safe borrowed enum payload parent fact. |
| `generic_wrapper_contains_borrow` | plir | `borrow_aggregate_escape_validator` | Derived from safe borrowed monomorphized generic wrapper parent fact. |

Validators reject derived v1 rows without `parent_fact_id`, safe claims from
`unsafe_unknown`, and broad noalias wording. `unsafe_unknown` remains rejected
or conservative and cannot become a trusted borrowed source.

## Positive Coverage

- Local enum payload use stays inside the owner lifetime.
- Local generic wrapper use stays inside the owner lifetime.
- `.copy()` escapes safely as owned for enum payload and generic wrapper
  carriers.

## Negative Coverage

- Borrowed enum payload returned as owned is rejected.
- Borrowed enum payload stored globally is rejected.
- Borrowed generic wrapper returned as owned is rejected.
- Borrowed generic wrapper stored globally is rejected.
- Mixed branch owners are rejected by the borrowed return consistency model.
- `unsafe_unknown` remains rejected or conservative.

## Nonclaims

- No interface borrow closure.
- No function-typed value or callback borrow closure.
- No async, actor, or task boundary expansion.
- No raw pointer expansion.
- No target parity.
- No broad noalias.
- No performance claim.
- No named lifetimes, full Rust-like borrow checker, async/concurrency memory
  model, or perfect memory claim.
