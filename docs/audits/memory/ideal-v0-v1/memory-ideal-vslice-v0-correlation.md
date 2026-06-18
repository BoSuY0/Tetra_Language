# Memory Ideal Vertical Slice v0 Correlation

Status: validated

This A1-lite matrix is intentionally minimal. It exists only to correlate the three Memory Ideal
Vertical Slice v0 rows. It is not a universal release correlation engine and does not replace
`MemoryFactGraph` as the source of truth.

| requirement_id | claim                                                                                             | source_fact_id                                                    | validator                          | report_row                                                                         | negative_test                                                                                            | target_level     | status    |
| -------------- | ------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ---------------------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ---------------- | --------- |
| MEM-REP-001    | safe representation metadata cannot be assigned by user code                                      | semantics:representation-metadata:not-user-assignable             | representation_namespace_validator | safe_representation_metadata:not_user_assignable                                   | metadata_assignment_rejected                                                                             | all:semantics    | validated |
| MEM-BORROW-001 | borrowed slice/String byte view through struct or optional cannot escape its source owner         | plir:borrowAggregate:f_struct_borrow:aggregate_contains_borrow    | borrow_aggregate_escape_validator  | aggregate_contains_borrow, optional_contains_borrow                                | TestBorrowedAggregateEscapeDiagnostics, TestOwnershipRejectsBorrowedSliceOptionalPayloadGlobalAssignment | linux-x64:narrow | validated |
| MEM-ALIAS-001  | unique local and sequential inout are narrow-exclusive; alias use during active inout is rejected | plir:mutate:f_no_alias:no_alias_validated_narrow_sequential_inout | alias_interval_validator           | no_alias_validated_narrow_unique_local, no_alias_validated_narrow_sequential_inout | TestOwnershipRejectsBorrowInoutAlias, TestOwnershipRejectsOverlappingMutableInoutSliceBorrow             | linux-x64:narrow | validated |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-correlation.md
```

The validator only checks the v0 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `conservative`, `rejected`, `future`, or `explicit_non_goal`;
- the row set is exactly `MEM-REP-001`, `MEM-BORROW-001`, and `MEM-ALIAS-001`.

## Update Policy

The final audit uses narrower prose such as `validated_narrow`, but this matrix intentionally uses
the limited v0 validator status vocabulary from the plan. The B2a and B3a rows are validated only
for the explicit v0 surface and remain bounded by the final audit nonclaims.
