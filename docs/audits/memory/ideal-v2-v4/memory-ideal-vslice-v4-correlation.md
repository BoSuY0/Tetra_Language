# Memory Ideal Vertical Slice v4 Correlation

Status: validated_narrow

This matrix intentionally has exactly four rows. It extends the Memory Ideal Vertical Slice
v0/v1/v2/v3 correlation pattern to the already-supported async/await, task, and actor boundary
surface, without claiming a full async lifetime system, full production actor runtime, structured
concurrency, cancellation model, distributed actor memory model, target parity, broad noalias, or
performance. `MemoryFactGraph` remains the source of truth; this document is only a projection/audit
correlation.

| requirement_id | claim                                                                                           | source_fact_id                                                                  | validator                             | report_row                         | negative_test                                    | target_level     | status           |
| -------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------- | ---------------------------------- | ------------------------------------------------ | ---------------- | ---------------- |
| MEM-BORROW-008 | borrowed view cannot cross async/await suspension boundary unless proven local and non-escaping | plir:borrowCarrierV4:f_async_boundary_borrow:async_boundary_borrow_conservative | async_boundary_borrow_validator       | async_boundary_borrow_conservative | TestMemoryIdealV4BorrowedAsyncResultRejected     | linux-x64:narrow | conservative     |
| MEM-BORROW-009 | borrowed view cannot cross task boundary without explicit copy                                  | plir:borrowCarrierV4:f_task_boundary_borrow:task_boundary_borrow_rejected       | task_boundary_borrow_validator        | task_boundary_borrow_rejected      | TestMemoryIdealV4BorrowedViewSentToTaskRejected  | linux-x64:narrow | validated_narrow |
| MEM-BORROW-010 | borrowed view cannot cross actor boundary without explicit copy                                 | plir:borrowCarrierV4:f_actor_boundary_borrow:actor_boundary_borrow_rejected     | actor_boundary_borrow_validator       | actor_boundary_borrow_rejected     | TestMemoryIdealV4BorrowedViewSentToActorRejected | linux-x64:narrow | validated_narrow |
| MEM-ALIAS-004  | task/actor boundary cannot produce broad noalias                                                | plir:borrowCarrierV4:f_boundary_noalias:boundary_noalias_conservative           | boundary_alias_conservative_validator | boundary_noalias_conservative      | TestMemoryIdealV4TaskActorBroadNoAliasRejected   | linux-x64:narrow | conservative     |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v4-correlation.md
```

The validator checks this v4 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`, `rejected`, `future`, or
  `explicit_non_goal`;
- the row set is exactly `MEM-BORROW-008`, `MEM-BORROW-009`, `MEM-BORROW-010`, and `MEM-ALIAS-004`.

## Update Policy

The rows are narrow by construction. Local async borrowed use before a suspension point may remain
valid when the checker keeps the view local and non-escaping. Borrowed views crossing an async
suspension remain conservative unless a later slice proves a narrower local lifetime. Task and actor
boundary transfers reject borrowed views without explicit copy. Unknown task/actor targets remain
rejected or conservative and must not emit trusted lifetime-safe borrow facts. Task/actor boundaries
must not emit broad noalias.

This document does not claim full production actor runtime, full async lifetime system, structured
concurrency, cancellation semantics, distributed actor memory model, zero-copy region move
expansion, raw pointer expansion, target parity, broad noalias, or performance.
