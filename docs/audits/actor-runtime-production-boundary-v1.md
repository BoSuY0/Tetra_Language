# Actor Runtime Production Boundary Audit v1

Status: P18.0 audit for the Ideal Master Plan.

## Summary

`compiler/internal/actorsrt.ActorRuntimeProductionBoundaryAudit()` emits schema
`tetra.runtime.actor.production_boundary.v1`. The report separates four things:
current actor runtime limits, scheduler prototype features, production runtime
acceptance, and the facts that keep a no full production actor runtime claim
boundary explicit.

This slice does not implement a new scheduler, transport, mailbox reclamation
policy, or production actor-runtime mode. Its purpose is to keep bounded runtime
evidence from being promoted into a broader runtime claim.

## Rows

| Row | Status | Evidence | Boundary |
| --- | --- | --- | --- |
| current actor runtime limits | `documented_limit` | `compiler/internal/actorsrt/linux_x64.go`, `emitMailboxFullCheckForReceiverInEcx`, `emitCheckedMessagePoolAlloc`, `emitRecycleMessageNodeInRax`, `emitInvalidActorHandleReturn`, `emitActorDoneReturn`, `emitBlockedDeadlineWakeCheck`, `emitWaitingTaskWakeCheck`, `emitCurrentTaskGroupCanceledCheck`, `TestActorMailboxFullReturnsCheckedBackpressure`, `TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRun`, `TestActorTypedMailboxBackpressureRecoversWithoutPartialPayloadBuildAndRun`, `TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun`, `TestActorMessagePoolExhaustionReturnsCheckedFailure`, `TestActorInvalidHandleSendReturnsCheckedFailure`, `TestActorSendToDoneActorReturnsCheckedFailure`, `TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAndRun`, `TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuildAndRun`, `TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndRun`, `TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect`, `TestLinuxRuntimePumpsNodeDownIntoNodeStatus`, `TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun`, `TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun`, `TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun`, `TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun`, `TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun`, `TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun`, `TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump`, `TestNonLinuxRuntimesDoNotExportDistributedActorSymbols`, `docs/spec/actors.md` | Fixed-capacity x64 actor runtime evidence includes `maxActors=128`, `maxActorMailboxMsgs=256`, cooperative round-robin bounded progress for yielding runnable actors, deterministic deadline-order wake for sleeping actors, recoverable checked `-2` mailbox-full backpressure, `msgPoolSize=65536`, checked `-1` live message-pool exhaustion without overflow enqueue, reclaimed drained message nodes, checked `-3` invalid-handle sends, checked `-4` done-actor sends, nonzero actor entry returns exposed only as the same user-visible done-state send failure, missing-node `node_down` status evidence with no automatic retry/reconnect/restart/supervision claim, scoped cooperative task-group cancellation wake/error behavior for timed actor receive and task join waiters, eight actor-state slots, typed payload slot caps, and Linux-x64 distributed symbols. It does not provide preemptive or production multi-threaded scheduling, actor status/join/exit-code APIs, cancellation results for non-timed actor receives, supervision, restart semantics, retry/reconnect production behavior, or a full structured-concurrency model. |
| scheduler prototype features | `prototype_evidence` | `compiler/internal/parallelrt`, `TestSchedulerModelRunsSingleCoreFIFO`, `TestSchedulerModelStealsWorkAcrossTwoCores`, `TestPrototypeBenchmarksReportFanoutAndZeroCopyRows`, `tools/cmd/parallel-production-smoke` | The per-core scheduler work is a checked model and benchmark row. It covers single-core FIFO compatibility, two-core work stealing, bounded typed mailboxes, and `zero_copy_move` transfer rows, but it is not the production actor scheduler. |
| production runtime acceptance | `acceptance_required` | `tools/validators/parallelprod`, `tools/validators/actordist`, `docs/spec/actors.md`, `docs/user/async_actors_guide.md` | Future production claims must prove scheduler fairness, wake, deadline, starvation/progress-bound, and stress gates; mailbox backpressure; message exhaustion/reclamation; race-safety; actor/island boundary proof rows; cross-target distributed runtime gates; a blocking-primitive-by-cancellation-source matrix; structured concurrency; and fake-evidence rejection. |
| full claim blockers | `blocked` | `docs/spec/actors.md`, `docs/user/async_actors_guide.md`, `docs/design/actor_region_transfer.md` | Missing facts still block a full production actor-runtime claim: integrated production multi-threaded actor scheduling, non-Linux-x64 distributed actor runtime gates, full cancellation/structured concurrency guarantees, full race-safety proof, and production broker deployment evidence. |

## Validator Guards

`ValidateActorRuntimeProductionBoundaryAudit()` rejects:

- `FullProductionClaimed=true`;
- missing or duplicate P18.0 rows;
- rows missing evidence or boundary text;
- a scheduler prototype row marked as production-ready;
- a blockers row without missing facts;
- a missing full-production-runtime non-claim.

## Memory Boundary Handoff

`MEMISL-P10` adds a separate
`compiler/internal/semantics.MemoryBoundaryHandoffAudit()` with schema
`tetra.memory.boundary_handoff.v1`. That audit is the Memory/Islands handoff
row for actor/task/request work: actor borrowed payloads require explicit
`.copy()`, typed task reference-shaped error payloads reject, request/task
regions are scoped and reset, raw unsafe payloads cannot become safe typed
actor messages, stale island handles after `core.island_reset` reject before
actor send, and island payloads remain linear across `core.send_typed`.

This handoff does not change the actor runtime production status above. It
proves boundary preconditions for a later actor-runtime plan; it does not start
or complete that actor-runtime implementation.

The actor-runtime foundation P08 slice consumes those preconditions as an
additive Linux-x64 release evidence row named `actor island boundary proof`.
`parallel-production-smoke` emits that case and `parallelprod` rejects reports
that omit it. This remains scoped to actor/task/island boundary evidence and
does not promote request-island semantics.

`MEMISL-P20` records the final Memory/Islands handoff in
`docs/audits/memory-islands-final-production-handoff.md`. That document permits
starting a separate Actor Runtime Production Foundation plan from the scoped
Memory/Islands baseline; no actor production gate passed claim is made by that
handoff.

## Actor Runtime Foundation Gate

Actor runtime foundation scoped release truth is
`tetra.actor.production_foundation.v1`. The authoritative gate is
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`, and its
final evidence lives under `reports/actor-runtime-foundation/final/`:
`actor-runtime-foundation-manifest.json`, `artifact-hashes.json`,
`distributed-actors-linux-x64/distributed-actors-linux-x64.json`, and
`parallel-production-linux-x64/parallel-production-linux-x64.json`.

The gate is also wired into `.github/workflows/ci.yml` and
`.github/workflows/release-packages.yml` so package publishing cannot make an
actor foundation claim before the gate and uploaded artifacts exist.

## Non-Claims

- no full Erlang/OTP actor runtime claim.
- no cluster membership or reconnect/retry production claim.
- no non-Linux distributed actor runtime support claim.
- no distributed zero-copy pointer or region transfer claim.
- no formal race proof claim.
- no benchmark superiority, no C++/Rust parity, and no official benchmark
  claim.
- no full production actor runtime claim.
- Scheduler prototype evidence is not a production multi-threaded actor
  scheduler.
- Distributed actor runtime support remains bounded to Linux-x64 loopback TCP
  smoke evidence.
