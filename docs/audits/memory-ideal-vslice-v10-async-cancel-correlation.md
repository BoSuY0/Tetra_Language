# Memory Ideal Vertical Slice v10 Async Cancellation Correlation

Status: accepted narrow correlation target for `MEM-ASYNC-010`.

This table is intentionally exact. It adds only compiler-visible async
cancellation and structured boundary conservatism evidence. It does not add
production actor runtime proof, a distributed actor memory model, a full async
lifetime system, complete structured concurrency proof, target parity,
performance evidence, broad noalias, arbitrary FFI/runtime lifetime proof,
arbitrary external pointer safety, a clean-release claim, or a "Memory 100%"
claim.

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-ASYNC-001 | borrowed value may be used only before suspension when proven local and non-escaping | memorymodel:asyncV10:preawait:local_borrow_before_suspension | pre_await_local_borrow_validator | pre_await_local_borrow_validated | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts | linux-x64:narrow | validated_narrow |
| MEM-ASYNC-002 | borrowed value crossing await or suspend remains conservative or rejected | memorymodel:asyncV10:postawait:borrow_after_suspension_conservative | post_await_borrow_conservative_validator | post_await_borrow_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts | linux-x64:narrow | conservative |
| MEM-ASYNC-003 | cancellation path invalidates borrowed task-owned lifetime assumptions | memorymodel:asyncV10:cancel:borrow_lifetime_invalidated | cancellation_lifetime_invalidation_validator | cancellation_borrow_lifetime_invalidated | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts | linux-x64:narrow | rejected |
| MEM-ASYNC-004 | task group structured concurrency boundary cannot validate broad noalias | memorymodel:asyncV10:taskgroup:task_group_noalias_conservative | task_group_boundary_conservative_validator | task_group_noalias_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |
| MEM-ASYNC-005 | actor reentrant callback boundary keeps borrow and storage conservative unless separately proven | memorymodel:asyncV10:actor:actor_reentrant_callback_conservative | actor_reentrant_callback_boundary_validator | actor_reentrant_callback_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts | linux-x64:narrow | conservative |

## Notes

- `MemoryFactGraph` remains the truth source.
- `tetra.memory-report.v1` rows remain projections.
- `validate-memory-correlation` recognizes exactly the five `MEM-ASYNC-*`
  rows and rejects missing, extra, or widened v10 rows.
- `pre_await_local_borrow_validated` validates only a compiler-visible local
  non-escaping borrow before suspension.
- Post-await, cancellation, task-group, and actor reentrant callback rows stay
  conservative or rejected unless a later narrow proof exists.
