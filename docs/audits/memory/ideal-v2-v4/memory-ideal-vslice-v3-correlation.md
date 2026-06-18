# Memory Ideal Vertical Slice v3 Correlation

Status: validated_narrow

This matrix intentionally has exactly three rows. It extends the Memory Ideal Vertical Slice
v0/v1/v2 correlation pattern to the already-supported interface/protocol/static-conformance surface,
without claiming full dynamic dispatch or full existential container semantics. `MemoryFactGraph`
remains the source of truth; this document is only a projection/audit correlation.

| requirement_id | claim                                                                                                | source_fact_id                                                                          | validator                                      | report_row                             | negative_test                                               | target_level     | status           |
| -------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ---------------------------------------------- | -------------------------------------- | ----------------------------------------------------------- | ---------------- | ---------------- |
| MEM-BORROW-006 | borrowed view through interface/protocol value cannot escape owner                                   | plir:borrowCarrierV3:f_interface_borrow:interface_value_contains_borrow                 | interface_borrow_escape_validator              | interface_value_contains_borrow        | TestMemoryIdealV3BorrowedInterfaceReturnAsOwnedRejected     | linux-x64:narrow | validated_narrow |
| MEM-BORROW-007 | borrowed view passed through dynamic dispatch remains conservative unless target is statically known | plir:borrowCarrierV3:f_protocol_dispatch_borrow:protocol_dispatch_borrow_conservative   | protocol_dispatch_borrow_validator             | protocol_dispatch_borrow_conservative  | TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected | linux-x64:narrow | conservative     |
| MEM-ALIAS-003  | interface/protocol dispatch cannot produce broad noalias                                             | plir:borrowCarrierV3:f_protocol_dispatch_noalias:protocol_dispatch_noalias_conservative | protocol_dispatch_alias_conservative_validator | protocol_dispatch_noalias_conservative | TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected       | linux-x64:narrow | conservative     |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v3-correlation.md
```

The validator checks this v3 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`, `rejected`, `future`, or
  `explicit_non_goal`;
- the row set is exactly `MEM-BORROW-006`, `MEM-BORROW-007`, and `MEM-ALIAS-003`.

## Update Policy

The rows are narrow by construction. Known/static protocol targets may receive narrow facts only
when the checker/PLIR already exposes concrete target and borrow provenance. Unknown dynamic
dispatch remains conservative and must not emit trusted lifetime-safe borrow facts.
Interface/protocol dispatch must not emit broad noalias. This document does not claim full trait
objects, protocol existentials, witness tables, conformance-table lookup, async/task/actor
expansion, raw pointer expansion, target parity, broad noalias, or performance.
