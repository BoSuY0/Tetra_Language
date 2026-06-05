# Memory Ideal Vertical Slice v2 Final Audit

Status: validated_narrow

This audit closes the function-typed value and callback-boundary borrow carrier
slice. It extends the v0/v1 memory correlation pattern without claiming full
first-class callable memory semantics. `MemoryFactGraph` remains the truth
source; reports remain projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-BORROW-004 | validated_narrow | `compiler/tests/semantics/borrow_copy_test.go` covers function-typed field local borrowed use and owned-return escape rejection; `compiler/internal/memoryfacts/from_plir_test.go` proves `function_value_contains_borrow` rows with parent facts; `compiler/internal/memorymodel/mini_test.go` covers function value local use and borrowed escape rejection. | Only checker/PLIR-visible function-typed local values, struct fields, and already-supported enum payload carriers are in scope. |
| MEM-BORROW-005 | validated_narrow | `compiler/tests/semantics/borrow_copy_test.go` covers known local callback borrowed use, `.copy()` before callback escape, non-borrow callback parameter rejection, callback returned-owned rejection, global-storage rejection, consumed argument rejection, unknown target conservatism, and captured callback conservatism; `compiler/internal/memoryfacts/from_plir_test.go` proves `callback_arg_contains_borrow` rows with parent facts and no trusted facts for unknown callback targets; `compiler/internal/memorymodel/mini_test.go` covers known callback local use, copied payload escape, and unknown callback conservative handling. | Only callback parameters with known direct targets and PLIR-visible borrowed arguments are trusted. Unknown callback targets stay conservative. |
| MEM-ALIAS-002 | conservative | `compiler/tests/semantics/borrow_copy_test.go` covers callback `inout` rejection and callback aliasing of `inout` rejection; `compiler/internal/memoryfacts/from_plir_test.go` proves `callback_inout_conservative` with `alias_state: invalidated_by_call`; `compiler/internal/memorymodel/mini_test.go` covers callback reentrant `inout` conservative/reject behavior; `tools/cmd/validate-memory-correlation/main_test.go` rejects broad/mixed row sets. | Callback or reentrant `inout` never grants broad noalias in this slice. Only future local non-reentrant proof may refine it. |

## Minimal Report Projection

Projected v2 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `function_value_contains_borrow` | plir | `function_value_borrow_escape_validator` | Derived from a safe borrowed parent fact carried through a function-typed value. |
| `callback_arg_contains_borrow` | plir | `callback_borrow_escape_validator` | Derived from a safe borrowed parent fact at a known direct callback boundary. |
| `callback_inout_conservative` | plir | `callback_alias_conservative_validator` | Derived from callback/reentrant `inout` alias evidence and projected as conservative with `alias_state: invalidated_by_call`. |

Validators reject derived v2 rows without `parent_fact_id`, safe claims from
`unsafe_unknown`, trusted borrow facts for unknown callback targets, and broad
noalias wording. `unsafe_unknown` remains rejected or conservative and cannot
become a trusted borrowed source.

## Positive Coverage

- Known local callback use keeps the borrowed view inside the owner lifetime.
- Known function-typed field calls use borrowed views locally.
- `.copy()` before callback escape is accepted as owned.

## Negative Coverage

- Borrowed view passed to a non-borrow callback parameter is rejected.
- Borrowed view returned as owned through a callback is rejected.
- Borrowed view stored globally through a callback is rejected.
- Borrowed view consumed by a callback is rejected.
- Borrowed view passed as `inout` is rejected.
- Callback aliasing of an `inout` argument is rejected or conservative.
- Unknown callback targets do not emit trusted facts.
- Capturing callbacks remain rejected or conservative.
- Broad noalias rows are rejected or conservative.

## Nonclaims

- No full callable ABI.
- No captured or escaping closure memory model.
- No interface or protocol value borrow closure.
- No async, actor, or task boundary expansion.
- No raw pointer expansion.
- No target parity.
- No broad noalias.
- No performance claim.
