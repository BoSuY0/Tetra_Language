# P18.2 Per-Core Scheduler v1 Audit

P18.2 records per-core scheduler v1 evidence without converting the current actor runtime into a
full production actor-runtime claim.

## Evidence

- `compiler/internal/parallelrt/per_core_scheduler.go` emits `tetra.parallel.per_core_scheduler.v1`.
- Rows cover per-core queues, work stealing, bounded typed mailboxes, backpressure, timers and
  sleep/wake, structured task groups, cancellation checkpoints, actor ping-pong, fanout/fanin, task
  group cancel, backpressure overflow, mailbox fairness, and stress/race-detector scope.
- `compiler/internal/parallelrt/per_core_scheduler_test.go` validates the P18.2 matrix and rejects
  fake actor-runtime, runtime-behavior-change, race-detector, missing stress, and missing non-claim
  evidence.
- `compiler/internal/parallelrt/scheduler_model.go::TypedMailbox.Receive` adds FIFO model evidence
  for mailbox fairness and capacity recovery.

## Runtime Evidence Used

- Per-core queues, work stealing, mailbox capacity, backpressure, and owned region transfer evidence
  come from `compiler/internal/parallelrt`.
- Timers, sleep/wake, task groups, cancellation checkpoints, task/actor mailbox handoff, and
  cancel-before-deadline evidence come from `compiler/compiler_suite_test.go`.
- Parallel production report validation comes from `tools/validators/parallelprod/report.go` and
  `tools/cmd/parallel-production-smoke/main.go`.

## Non-Claims

- No full production actor runtime is claimed.
- No non-Linux distributed actor runtime target is promoted.
- No all-target race-detector claim is made.
- No scheduler performance or throughput claim is made.
- No safe-program semantics or public runtime mode changes are made.
