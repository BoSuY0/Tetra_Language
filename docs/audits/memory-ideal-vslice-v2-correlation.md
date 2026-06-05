# Memory Ideal Vertical Slice v2 Correlation

Status: validated_narrow

This matrix is intentionally minimal. It exists only to correlate the three
Memory Ideal Vertical Slice v2 rows for function-typed values, callback
parameters, and callback/reentrant `inout` conservatism. It is not a universal
callable memory model and does not replace `MemoryFactGraph` as the source of
truth.

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BORROW-004 | borrowed view passed through function-typed value cannot escape owner | plir:borrowCarrierV2:f_function_value_borrow:function_value_contains_borrow | function_value_borrow_escape_validator | function_value_contains_borrow | TestMemoryIdealV2BorrowedCallbackReturnAsOwnedRejected | linux-x64:narrow | validated_narrow |
| MEM-BORROW-005 | borrowed view passed through callback parameter cannot escape owner | plir:borrowCarrierV2:f_callback_arg_borrow:callback_arg_contains_borrow | callback_borrow_escape_validator | callback_arg_contains_borrow | TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected | linux-x64:narrow | validated_narrow |
| MEM-ALIAS-002 | callback/reentrant inout cannot produce broad noalias | plir:borrowCarrierV2:f_callback_inout:callback_inout_conservative | callback_alias_conservative_validator | callback_inout_conservative | TestMemoryIdealV2CallbackAliasesInoutArgumentRejected | linux-x64:narrow | conservative |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v2-correlation.md
```

The validator checks this v2 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`,
  `rejected`, `future`, or `explicit_non_goal`;
- the row set is exactly `MEM-BORROW-004`, `MEM-BORROW-005`, and
  `MEM-ALIAS-002`.

## Update Policy

The rows are narrow by construction. They do not claim full callable ABI,
captured closures, escaping closures, interfaces/protocol values, async,
actor/task boundaries, raw pointer expansion, target parity, broad noalias, or
performance.
