# Memory Ideal Vertical Slice v1 Correlation

Status: validated_narrow

This matrix is intentionally minimal. It exists only to correlate the two Memory Ideal Vertical
Slice v1 rows for enum payload and monomorphized generic wrapper borrow closure. It is not a
universal release correlation engine and does not replace `MemoryFactGraph` as the source of truth.

| requirement_id | claim                                                                   | source_fact_id                                                        | validator                         | report_row                      | negative_test                                                                                      | target_level     | status           |
| -------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------------------- | ---------------- | ---------------- |
| MEM-BORROW-002 | borrowed view through enum payload cannot escape owner                  | plir:borrowCarrierV1:f_enum_borrow:enum_payload_contains_borrow       | borrow_aggregate_escape_validator | enum_payload_contains_borrow    | TestMemoryIdealV1BorrowEnumPayloadGlobalStorageRejected, TestBorrowedAggregateEscapeDiagnostics    | linux-x64:narrow | validated_narrow |
| MEM-BORROW-003 | borrowed view through monomorphized generic wrapper cannot escape owner | plir:borrowCarrierV1:f_generic_borrow:generic_wrapper_contains_borrow | borrow_aggregate_escape_validator | generic_wrapper_contains_borrow | TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected, TestBorrowedAggregateEscapeDiagnostics | linux-x64:narrow | validated_narrow |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v1-correlation.md
```

The validator checks this v1 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`, `rejected`, `future`, or
  `explicit_non_goal`;
- the row set is exactly `MEM-BORROW-002` and `MEM-BORROW-003`.

## Update Policy

The rows are narrow by construction. They do not claim interfaces, function-typed values, callbacks,
async, actor/task boundaries, raw pointer semantics, target parity, or broad noalias.
