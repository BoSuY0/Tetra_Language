# P0-discovery Result

Status: accepted

## Observed Files

- `compiler/internal/memoryfacts/from_plir.go`: v0 derives
  `aggregate_contains_borrow` and `optional_contains_borrow` in
  `addBorrowAggregateV0Facts` through `memoryIdealBorrowWrapperClaim`.
- `compiler/internal/memoryfacts/validate.go` and
  `tools/cmd/validate-memory-report/main.go`: claim allowlists and
  `claimRequiresParentFactID` must include any new derived borrow projection.
- `compiler/internal/memoryfacts/report_test.go` and
  `compiler/internal/memoryfacts/from_plir_test.go`: existing validator and PLIR
  projection tests are the nearest pattern for new v1 RED tests.
- `compiler/internal/memorymodel/mini.go`: `WrapperKind` currently has
  `struct_field` and `optional_payload` only.
- `compiler/tests/semantics/borrow_copy_test.go`: current dirty checkout already
  rejects simple enum payload and generic wrapper owned-return escapes inside
  `TestBorrowedAggregateEscapeDiagnostics`; v1 still needs explicit local/global
  coverage, model/report facts, and docs.
- `tools/cmd/validate-memory-correlation/main.go`: validator is hard-coded to
  v0 rows and must support a v1 matrix with exactly `MEM-BORROW-002` and
  `MEM-BORROW-003` while preserving v0 validation.

## Decisions

- Add RED tests first for MiniMemoryModel v1 wrappers, memoryfacts v1
  projections, memory report validator parent requirements, correlation v1 row
  set, and semantics local/global coverage.
- Keep checker edits narrow. Existing `checkBorrowedAggregateEscape` already
  recurses through `TypeEnum`, `TypeOptional`, and `TypeStruct`; generic wrapper
  support may already arrive via monomorphized generic structs.

## Risks

- `memoryIdealBorrowWrapperClaim` currently checks optional/payload before
  enum/generic terms, so enum payload PLIR notes may incorrectly project as
  `optional_contains_borrow`.
- Current v0 semantics test includes v1-like cases; new tests should use
  `MemoryIdealV1` names to make gates/audit explicit rather than silently
  relying on v0 test names.
