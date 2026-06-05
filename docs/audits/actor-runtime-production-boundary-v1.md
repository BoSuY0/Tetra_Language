# Actor Runtime Production Boundary Audit v1

Status: P18.0 audit for the Ideal Master Plan.

## Summary

`compiler/internal/actorsrt.ActorRuntimeProductionBoundaryAudit()` emits schema
`tetra.runtime.actor.production_boundary.v1`. The report separates four things:
current actor runtime limits, scheduler prototype features, production runtime
acceptance, and blockers for a full production actor-runtime claim.

This slice does not implement a new scheduler, transport, mailbox capacity
policy, or production actor-runtime mode. Its purpose is to keep prototype
evidence from being promoted into a broader runtime claim.

## Rows

| Row | Status | Evidence | Boundary |
| --- | --- | --- | --- |
| current actor runtime limits | `documented_limit` | `compiler/internal/actorsrt/linux_x64.go`, `TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump`, `TestNonLinuxRuntimesDoNotExportDistributedActorSymbols`, `docs/spec/actors.md` | Fixed-capacity x64 actor runtime evidence includes `maxActors=128`, `msgPoolSize=65536`, eight actor-state slots, typed payload slot caps, and Linux-x64 distributed symbols. It does not provide checked recoverable message-pool exhaustion or production multi-threaded scheduling. |
| scheduler prototype features | `prototype_evidence` | `compiler/internal/parallelrt`, `TestSchedulerModelRunsSingleCoreFIFO`, `TestSchedulerModelStealsWorkAcrossTwoCores`, `TestPrototypeBenchmarksReportFanoutAndZeroCopyRows`, `tools/cmd/parallel-production-smoke` | The per-core scheduler work is a checked model and benchmark row. It covers single-core FIFO compatibility, two-core work stealing, bounded typed mailboxes, and `zero_copy_move` transfer rows, but it is not the production actor scheduler. |
| production runtime acceptance | `acceptance_required` | `tools/validators/parallelprod`, `tools/validators/actordist`, `docs/spec/actors.md`, `docs/user/async_actors_guide.md` | Future production claims must prove scheduler, mailbox backpressure, message exhaustion/reclamation, race-safety, cross-target distributed runtime gates, structured concurrency, and fake-evidence rejection. |
| full claim blockers | `blocked` | `docs/spec/actors.md`, `docs/user/async_actors_guide.md`, `docs/design/actor_region_transfer.md` | Missing facts still block a full production actor-runtime claim: integrated production multi-threaded actor scheduling, checked message-pool exhaustion or reclamation, non-Linux-x64 distributed actor runtime gates, full cancellation/structured concurrency guarantees, full race-safety proof, and production broker deployment evidence. |

## Validator Guards

`ValidateActorRuntimeProductionBoundaryAudit()` rejects:

- `FullProductionClaimed=true`;
- missing or duplicate P18.0 rows;
- rows missing evidence or boundary text;
- a scheduler prototype row marked as production-ready;
- a blockers row without missing facts;
- a missing full-production-runtime non-claim.

## Non-Claims

- A full production actor runtime is not claimed.
- Scheduler prototype evidence is not a production multi-threaded actor
  scheduler.
- Distributed actor runtime support remains bounded to Linux-x64 loopback TCP
  smoke evidence.
