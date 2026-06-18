# P18.2 Per-Core Scheduler v1 Design

## Goal

Close P18.2 as an evidence-backed per-core scheduler v1 slice without
promoting the actor runtime beyond the P18.0 production boundary.

## Observed Context

- `compiler/internal/parallelrt/scheduler_model.go` already models per-core
  queues, two-core work stealing, bounded typed mailboxes, backpressure
  metadata, and owned-region transfer reports.
- `compiler/task_runtime_test.go` already has executable Linux-x64 evidence
  for timers, sleep/wake, deadline-aware waits, task groups, cancellation
  checkpoints, and task/actor mailbox handoff.
- `tools/validators/parallelprod/report.go` already validates the parallel
  production smoke report and requires lifecycle, backpressure, transfer,
  stress, and audit rows.
- `compiler/internal/actorsrt/production_boundary.go` keeps full production
  actor-runtime claims blocked until runtime integration, message-pool
  exhaustion, race-safety, cross-target distributed runtime, and broker evidence
  are separately proven.

## Design

Add a P18.2-specific coverage report in `compiler/internal/parallelrt`:

- schema `tetra.parallel.per_core_scheduler.v1`;
- one row for each master-plan feature and required test family;
- explicit evidence strings pointing at real model, runtime, validator, and
  smoke-test artifacts;
- non-claims preserving that P18.2 is not a full production actor runtime,
  not a non-Linux distributed actor-runtime promotion, and not an all-target
  race-detector claim.

The coverage validator rejects fake rows, missing facts, missing stress
evidence, actor-runtime promotion, runtime behavior-change claims, and
race-detector overclaims.

Add a narrow mailbox FIFO receive operation to the local model so mailbox
fairness has executable model evidence in addition to actor/self mailbox smoke
artifacts. This remains a model/report path and does not alter the built-in
actor runtime scheduler.

## Verification

Use TDD:

1. Add RED coverage/validator tests for the P18.2 rows and anti-overclaim
   behavior.
2. Add RED model evidence for FIFO mailbox fairness.
3. Implement the coverage API, validator, and mailbox receive helper.
4. Run focused `parallelrt` tests, feature/docs/manifest gates, broader
   parallel runtime validators, and `graphify update .`.

## Boundaries

- No full production actor runtime claim.
- No new distributed actor runtime target claim.
- No full race-safety proof.
- No scheduler performance claim.
- No public runtime mode or safe-semantics flag change.
