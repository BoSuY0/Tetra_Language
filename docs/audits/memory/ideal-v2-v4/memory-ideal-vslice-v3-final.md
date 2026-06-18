# Memory Ideal Vertical Slice v3 Final Audit

Status: validated_narrow

This audit closes the interface/protocol/existential-like borrow-boundary slice for the current
static-conformance surface. It extends the v0/v1/v2 memory correlation pattern without implementing
full dynamic dispatch or full existential container semantics. `MemoryFactGraph` remains the truth
source; reports remain projections.

## Row Classifications

| requirement_id | classification   | evidence                                                                                                                                                                                                                                                                                                                                                                                                                                   | boundary                                                                                                                                            |
| -------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| MEM-BORROW-006 | validated_narrow | `compiler/tests/semantics/semantics_async_ownership_test.go` covers known/static concrete protocol target local borrowed use and owned-return/global-storage rejection; `compiler/internal/memoryfacts_test/from_plir_test.go` proves `interface_value_contains_borrow` with parent facts; `compiler/internal/memorymodel/mini_test.go` covers interface/protocol local use and escape rejection.                                          | Only checker/PLIR-visible interface/protocol-like values with statically known concrete target and safe borrowed parent facts are trusted.          |
| MEM-BORROW-007 | conservative     | `compiler/tests/semantics/semantics_async_ownership_test.go` keeps generic-bound protocol requirement calls and runtime protocol values rejected; `compiler/internal/memoryfacts_test/from_plir_test.go` proves `protocol_dispatch_borrow_conservative` and verifies unknown dynamic dispatch does not emit `interface_value_contains_borrow`; `compiler/internal/memorymodel/mini_test.go` covers unknown protocol dispatch conservatism. | Unknown dynamic protocol dispatch remains conservative unless a target is statically known. No trusted borrow fact is emitted for unknown dispatch. |
| MEM-ALIAS-003  | conservative     | `compiler/internal/memoryfacts_test/from_plir_test.go` proves `protocol_dispatch_noalias_conservative` with conservative fallback; `compiler/internal/memoryfacts_test/report_test.go` rejects broad protocol/interface noalias wording; `tools/cmd/validate-memory-correlation/main_test.go` rejects missing and mixed v3 row sets; `compiler/internal/memorymodel/mini_test.go` covers protocol dispatch noalias rejection.              | Interface/protocol dispatch never grants broad noalias in this slice. Only future narrower static proof may refine a non-dispatch local case.       |

## Minimal Report Projection

Projected v3 claims:

| claim                                    | source stage | validator                                        | notes                                                                                                                                         |
| ---------------------------------------- | ------------ | ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `interface_value_contains_borrow`        | plir         | `interface_borrow_escape_validator`              | Derived from a safe borrowed parent fact carried through a checker/PLIR-visible interface/protocol-like value with a statically known target. |
| `protocol_dispatch_borrow_conservative`  | plir         | `protocol_dispatch_borrow_validator`             | Derived for unknown dynamic protocol dispatch and projected as conservative fallback, not as a trusted borrow fact.                           |
| `protocol_dispatch_noalias_conservative` | plir         | `protocol_dispatch_alias_conservative_validator` | Derived for protocol/interface dispatch alias evidence and projected as conservative fallback rather than validated noalias.                  |

Validators reject derived v3 rows without `parent_fact_id`, safe claims from `unsafe_unknown`,
trusted interface borrow facts for unknown dynamic dispatch, and broad protocol/interface noalias
wording. `unsafe_unknown` remains rejected or conservative and cannot become a trusted borrowed
source.

## Positive Coverage

- Known/static concrete protocol target local use keeps the borrowed view inside the owner lifetime.
- The positive semantics path uses `borrow BorrowView` self and a direct `BorrowView.len` call, so
  no runtime protocol value is required.

## Negative Coverage

- Borrowed view returned as owned through an interface/protocol-like aggregate is rejected.
- Borrowed view stored globally through an interface/protocol-like aggregate is rejected.
- Unknown dynamic protocol dispatch remains rejected or conservative and emits no trusted
  `interface_value_contains_borrow` fact.
- Broad noalias through interface/protocol dispatch is rejected or conservative.

## Nonclaims

- No full trait-object or protocol existential runtime.
- No runtime protocol values.
- No witness tables or conformance-table lookup.
- No full dynamic dispatch.
- No async, actor, or task boundary expansion.
- No raw pointer expansion.
- No target parity.
- No broad noalias.
- No performance claim.
