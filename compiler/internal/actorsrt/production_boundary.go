package actorsrt

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/parallelrt"
)

type ActorRuntimeBoundaryID string

const (
	ActorRuntimeBoundaryCurrentLimits        ActorRuntimeBoundaryID = "current_actor_runtime_limits"
	ActorRuntimeBoundarySchedulerPrototype   ActorRuntimeBoundaryID = "scheduler_prototype_features"
	ActorRuntimeBoundaryProductionAcceptance ActorRuntimeBoundaryID = "production_runtime_acceptance"
	ActorRuntimeBoundaryFullClaimBlockers    ActorRuntimeBoundaryID = "full_claim_blockers"
)

type ActorRuntimeBoundaryStatus string

const (
	ActorRuntimeBoundaryDocumentedLimit    ActorRuntimeBoundaryStatus = "documented_limit"
	ActorRuntimeBoundaryPrototypeEvidence  ActorRuntimeBoundaryStatus = "prototype_evidence"
	ActorRuntimeBoundaryAcceptanceRequired ActorRuntimeBoundaryStatus = "acceptance_required"
	ActorRuntimeBoundaryBlocked            ActorRuntimeBoundaryStatus = "blocked"
)

type ActorRuntimeBoundaryReport struct {
	SchemaVersion         string                    `json:"schema_version"`
	Rows                  []ActorRuntimeBoundaryRow `json:"rows"`
	NonClaims             []string                  `json:"non_claims"`
	FullProductionClaimed bool                      `json:"full_production_claimed"`
}

type ActorRuntimeBoundaryRow struct {
	ID            ActorRuntimeBoundaryID     `json:"id"`
	Name          string                     `json:"name"`
	Status        ActorRuntimeBoundaryStatus `json:"status"`
	RequiredFacts []string                   `json:"required_facts,omitempty"`
	MissingFacts  []string                   `json:"missing_facts,omitempty"`
	Evidence      string                     `json:"evidence"`
	Boundary      string                     `json:"boundary"`
}

func ActorRuntimeProductionBoundaryAudit() (ActorRuntimeBoundaryReport, error) {
	benchmarks, err := parallelrt.PrototypeBenchmarks()
	if err != nil {
		return ActorRuntimeBoundaryReport{}, err
	}
	if len(benchmarks) < 2 {
		return ActorRuntimeBoundaryReport{}, fmt.Errorf("actor runtime boundary audit: scheduler prototype benchmark evidence is incomplete")
	}
	return ActorRuntimeBoundaryReport{
		SchemaVersion: "tetra.runtime.actor.production_boundary.v1",
		Rows: []ActorRuntimeBoundaryRow{
			currentActorRuntimeLimitsRow(),
			schedulerPrototypeFeaturesRow(benchmarks),
			productionRuntimeAcceptanceRow(),
			fullClaimBlockersRow(),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			"scheduler prototype evidence is not a production multi-threaded actor scheduler",
			"distributed actor runtime support remains bounded to Linux-x64 loopback TCP smoke evidence",
			"nonzero actor entry returns have no user-facing actor status, join, exit-code, supervision, or restart API",
			"missing-node node_down evidence does not claim automatic retry, restart, reconnect, or supervision",
		},
		FullProductionClaimed: false,
	}, nil
}

func ValidateActorRuntimeProductionBoundaryAudit(report ActorRuntimeBoundaryReport) error {
	if report.SchemaVersion != "tetra.runtime.actor.production_boundary.v1" {
		return fmt.Errorf("actor runtime boundary audit: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionClaimed {
		return fmt.Errorf("actor runtime boundary audit: full production actor runtime claim is forbidden for P18.0")
	}
	if !containsBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		return fmt.Errorf("actor runtime boundary audit: missing full production actor runtime non-claim")
	}
	expected := map[ActorRuntimeBoundaryID]bool{
		ActorRuntimeBoundaryCurrentLimits:        false,
		ActorRuntimeBoundarySchedulerPrototype:   false,
		ActorRuntimeBoundaryProductionAcceptance: false,
		ActorRuntimeBoundaryFullClaimBlockers:    false,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf("actor runtime boundary audit: row count = %d, want %d", len(report.Rows), len(expected))
	}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("actor runtime boundary audit: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("actor runtime boundary audit: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("actor runtime boundary audit: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("actor runtime boundary audit: row %q missing evidence or boundary", row.ID)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("actor runtime boundary audit: missing row %q", id)
		}
	}
	rows := rowsByID(report.Rows)
	if err := validateCurrentLimitsRow(rows[ActorRuntimeBoundaryCurrentLimits]); err != nil {
		return err
	}
	if err := validateSchedulerPrototypeRow(rows[ActorRuntimeBoundarySchedulerPrototype]); err != nil {
		return err
	}
	if err := validateProductionAcceptanceRow(rows[ActorRuntimeBoundaryProductionAcceptance]); err != nil {
		return err
	}
	if err := validateFullClaimBlockersRow(rows[ActorRuntimeBoundaryFullClaimBlockers]); err != nil {
		return err
	}
	return nil
}

func currentActorRuntimeLimitsRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryCurrentLimits,
		Name:   "Current actor runtime limits",
		Status: ActorRuntimeBoundaryDocumentedLimit,
		RequiredFacts: []string{
			fmt.Sprintf("maxActors=%d", maxActors),
			fmt.Sprintf("msgPoolSize=%d", msgPoolSize),
			fmt.Sprintf("maxActorMailboxMsgs=%d", maxActorMailboxMsgs),
			fmt.Sprintf("actor_state_slots=%d", maxActorStateSlots),
			"single-thread cooperative scheduler documented for current actor runtime",
			"round-robin runnable actor fairness has bounded yield-progress evidence",
			"timed sleeping actors wake in deterministic deadline order",
			"linux-x64 distributed runtime only; non-Linux-x64 targets keep distributed actor symbols out of the built-in runtime",
			"non-linux actor net pump is no-op",
			"mailbox full returns checked -2 backpressure without allocating a message",
			"mailbox backpressure recovers after drain for local legacy, tagged, and typed sends",
			"typed mailbox backpressure does not enqueue a partial payload",
			"message pool exhaustion returns checked -1 without enqueueing an overflow message",
			"drained message pool entries are reclaimed after receive and can be reused",
			"invalid actor handle sends return checked -3 without allocating a message",
			"done actor sends return checked -4 without allocating a message",
			"nonzero actor entry return is exposed only as done-state send failure for later local sends",
			"no actor status, actor join, or actor exit-code API is exposed for done actors",
			"messages already queued in another actor mailbox remain receivable after the sender is done",
			"done actors are not restarted and pending mailbox entries are not drained by a shutdown phase",
			"blocked actors continue to depend on normal message, deadline, timer, or task-wait readiness when another actor exits",
			"missing-node node_down remains checked distributed status evidence",
			"no automatic retry, restart, reconnect, or supervision is claimed for local actor failure or distributed node_down status",
			"task-group cancellation wakes recv_until and recv_msg_until waiters with checked error 1",
			"task-group cancellation wakes actors already waiting on task_join_result_i32, task_join_until_i32, and select2_i32 with checked error 1",
			"task_join_i32 wakes on task-group cancellation with raw zero value; checked status requires result or timed join APIs",
			"non-timed actor receives do not expose a cancellation result in the current profile",
			"typed actor message payloads are capped at 8 value slots",
		},
		Evidence: "compiler/internal/actorsrt/linux_x64.go::BuildLinuxX64; compiler/internal/actorsrt/linux_x64_emit.go::emitMailboxFullCheckForReceiverInEcx; compiler/internal/actorsrt/linux_x64_emit.go::emitCheckedMessagePoolAlloc; compiler/internal/actorsrt/linux_x64_emit.go::emitRecycleMessageNodeInRax; compiler/internal/actorsrt/linux_x64_emit.go::emitInvalidActorHandleReturn; compiler/internal/actorsrt/linux_x64_emit.go::emitActorDoneReturn; compiler/internal/actorsrt/linux_x64_emit.go::emitBlockedDeadlineWakeCheck; compiler/internal/actorsrt/linux_x64_emit.go::emitWaitingTaskWakeCheck; compiler/internal/actorsrt/linux_x64_emit.go::emitCurrentTaskGroupCanceledCheck; compiler/actors_test.go::TestActorMailboxFullReturnsCheckedBackpressure; compiler/actors_test.go::TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRun; compiler/actors_test.go::TestActorTaggedMailboxBackpressureRecoversAfterSelfDrainBuildAndRun; compiler/actors_test.go::TestActorTypedMailboxBackpressureRecoversWithoutPartialPayloadBuildAndRun; compiler/actors_test.go::TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun; compiler/actors_test.go::TestActorMessagePoolExhaustionReturnsCheckedFailure; compiler/actors_test.go::TestActorInvalidHandleSendReturnsCheckedFailure; compiler/actors_test.go::TestActorSendToDoneActorReturnsCheckedFailure; compiler/actors_test.go::TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAndRun; compiler/actors_test.go::TestActorLifecycleReceivesPendingMessageFromDoneSenderBuildAndRun; compiler/actors_test.go::TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlockedActorsBuildAndRun; compiler/actors_test.go::TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuildAndRun; compiler/actors_test.go::TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndRun; cli/internal/actornet/broker_test.go::TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect; cli/internal/actornet/runtime_integration_test.go::TestLinuxRuntimePumpsNodeDownIntoNodeStatus; compiler/task_runtime_test.go::TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun; compiler/internal/actorsrt/actor_state_symbols_test.go::TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump; compiler/internal/actorsrt/actor_state_symbols_test.go::TestNonLinuxRuntimesDoNotExportDistributedActorSymbols; docs/spec/actors.md::Lifecycle Matrix; docs/spec/actors.md::Runtime Capacity Limits; docs/spec/actors.md::Distributed Runtime Promotion Surface; docs/spec/actors.md::Scheduling semantics",
		Boundary: "current evidence covers fixed-capacity x64 built-in actor runtime behavior, cooperative round-robin bounded progress for yielding runnable actors, deterministic deadline-order wake for sleeping actors, recoverable checked per-actor mailbox backpressure for local legacy/tagged/typed sends, no partial typed payload after failed backpressure, reusable drained message nodes with checked bounded message-pool exhaustion for live overload, checked invalid-handle and done-actor send failures, narrow done-state lifecycle semantics where zero and nonzero actor returns are user-visible only as done for later sends, scoped cooperative task-group cancellation wake/error behavior for timed actor receive and task join waiters, Linux-x64 distributed node_down status evidence for missing-node cases, Linux-x64 distributed actor runtime symbols, and documented capacity limits; it does not provide an unbounded mailbox, automatic retry/reconnect, actor close/shutdown API, actor status/join/exit-code API, cancellation results for non-timed actor receives, supervision/restart/linking/OTP lifecycle behavior, preemptive or production multi-threaded scheduling, non-Linux distributed runtime support, a full structured-concurrency model, or a full production actor runtime claim",
	}
}

func schedulerPrototypeFeaturesRow(benchmarks []parallelrt.PrototypeBenchmark) ActorRuntimeBoundaryRow {
	var names []string
	for _, benchmark := range benchmarks {
		if benchmark.Ran && benchmark.Pass {
			names = append(names, benchmark.Name)
		}
	}
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundarySchedulerPrototype,
		Name:   "Scheduler prototype features",
		Status: ActorRuntimeBoundaryPrototypeEvidence,
		RequiredFacts: []string{
			"single-core FIFO compatibility",
			"two-core work stealing",
			"bounded typed mailbox with blocking_recv_yield backpressure metadata",
			"zero_copy_move owned-region transfer benchmark",
			"bytes_copied=0 for owned-region prototype transfer",
			"prototype benchmarks: " + strings.Join(names, "; "),
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::NewSchedulerModel; compiler/internal/parallelrt/scheduler_model_test.go::TestSchedulerModelRunsSingleCoreFIFO; compiler/internal/parallelrt/scheduler_model_test.go::TestSchedulerModelStealsWorkAcrossTwoCores; compiler/internal/parallelrt/scheduler_model_test.go::TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy; compiler/internal/parallelrt/scheduler_model_test.go::TestPrototypeBenchmarksReportFanoutAndZeroCopyRows; tools/cmd/parallel-production-smoke/main.go::runSchedulerPrototypeEvidence",
		Boundary: "scheduler evidence is a checked model and release benchmark row; it is not a production multi-threaded actor scheduler, does not change compiler/runtime scheduling behavior, and does not promote the built-in actor runtime beyond its documented cooperative runtime boundary",
	}
}

func productionRuntimeAcceptanceRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryProductionAcceptance,
		Name:   "Production runtime acceptance",
		Status: ActorRuntimeBoundaryAcceptanceRequired,
		RequiredFacts: []string{
			"production task scheduler evidence with executable fairness, wake, deadline, actor scheduler starvation/progress bound, and stress gates",
			"bounded mailbox backpressure with checked recoverable failure behavior",
			"message reclamation or checked exhaustion semantics for runtime message pools",
			"race-safety model or conservative rejection evidence across task/actor/thread boundaries",
			"cross-target distributed runtime gates for every claimed target",
			"blocking primitive by cancellation-source matrix covering wake and checked-error behavior",
			"structured concurrency and cancellation semantics beyond the current cooperative task group handles",
			"artifact-hash and validator gates that reject fake, docs-only, metadata-only, and transport-only evidence",
		},
		Evidence: "tools/validators/parallelprod/report.go::validateContracts; tools/validators/parallelprod/report.go::validateCases; tools/validators/parallelprod/report.go::validateAudit; tools/validators/actordist/report.go::ValidateReport; docs/spec/actors.md::Distributed Runtime Promotion Surface; docs/user/async_actors_guide.md::Actors",
		Boundary: "acceptance criteria describe what a future production actor runtime claim must prove; P18.0 records the criteria only and does not mark those criteria satisfied for a full actor runtime",
	}
}

func fullClaimBlockersRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryFullClaimBlockers,
		Name:   "Full production actor runtime blockers",
		Status: ActorRuntimeBoundaryBlocked,
		MissingFacts: []string{
			"production multi-threaded actor scheduler integrated into the runtime",
			"non-Linux-x64 distributed actor runtime executable smoke and validator gates",
			"full cancellation and structured concurrency guarantees beyond cooperative task group handles",
			"full race-safety proof or audited conservative rejection matrix for shared mutable actor/task/thread boundaries",
			"production broker deployment, reconnect, ordering, retry, and cluster membership evidence beyond loopback TCP smoke",
		},
		Evidence: "docs/spec/actors.md::Non-goals; docs/spec/actors.md::Runtime Capacity Limits; docs/user/async_actors_guide.md::Actors; docs/design/actor_region_transfer.md::P6.3 adds a checked scheduler prototype model",
		Boundary: "these blockers keep the current evidence from becoming a full production actor runtime claim; existing distributed Linux-x64 and parallel production reports remain bounded slices rather than proof of general actor-runtime production completeness",
	}
}

func validateCurrentLimitsRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryDocumentedLimit {
		return fmt.Errorf("actor runtime boundary audit: current limits status = %q", row.Status)
	}
	for _, fact := range []string{"maxActors=128", "msgPoolSize=65536", "maxActorMailboxMsgs=256", "actor_state_slots=8", "single-thread cooperative scheduler", "round-robin runnable actor fairness has bounded yield-progress evidence", "timed sleeping actors wake in deterministic deadline order", "linux-x64 distributed runtime only", "non-linux actor net pump is no-op", "mailbox full returns checked -2", "mailbox backpressure recovers after drain", "typed mailbox backpressure does not enqueue a partial payload", "message pool exhaustion returns checked -1", "drained message pool entries are reclaimed", "invalid actor handle sends return checked -3", "done actor sends return checked -4", "nonzero actor entry return is exposed only as done-state send failure", "no actor status, actor join, or actor exit-code API", "messages already queued in another actor mailbox remain receivable", "done actors are not restarted", "blocked actors continue to depend on normal message", "missing-node node_down remains checked distributed status evidence", "no automatic retry, restart, reconnect, or supervision", "task-group cancellation wakes recv_until", "task-group cancellation wakes actors already waiting on task_join_result_i32", "task_join_i32 wakes on task-group cancellation with raw zero value", "non-timed actor receives do not expose a cancellation result"} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: current limits missing fact %q", fact)
		}
	}
	return nil
}

func validateSchedulerPrototypeRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryPrototypeEvidence {
		return fmt.Errorf("actor runtime boundary audit: scheduler prototype status = %q, want prototype_evidence", row.Status)
	}
	if strings.Contains(strings.ToLower(string(row.Status)), "production") {
		return fmt.Errorf("actor runtime boundary audit: scheduler prototype must not be production-ready")
	}
	for _, fact := range []string{"single-core FIFO compatibility", "two-core work stealing", "bounded typed mailbox", "zero_copy_move", "bytes_copied=0"} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: scheduler prototype missing fact %q", fact)
		}
	}
	if !strings.Contains(row.Boundary, "not a production multi-threaded actor scheduler") {
		return fmt.Errorf("actor runtime boundary audit: scheduler prototype boundary must preserve production non-claim")
	}
	return nil
}

func validateProductionAcceptanceRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryAcceptanceRequired {
		return fmt.Errorf("actor runtime boundary audit: production acceptance status = %q", row.Status)
	}
	for _, fact := range []string{"production task scheduler", "actor scheduler starvation/progress bound", "bounded mailbox backpressure", "message reclamation", "race-safety model", "cross-target distributed runtime gates", "blocking primitive by cancellation-source matrix", "structured concurrency"} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: production acceptance missing fact %q", fact)
		}
	}
	return nil
}

func validateFullClaimBlockersRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryBlocked {
		return fmt.Errorf("actor runtime boundary audit: blockers status = %q", row.Status)
	}
	if len(row.MissingFacts) == 0 {
		return fmt.Errorf("actor runtime boundary audit: blockers row must record missing facts")
	}
	for _, fact := range []string{"production multi-threaded actor scheduler", "non-Linux-x64 distributed actor runtime", "full cancellation and structured concurrency", "full race-safety proof"} {
		if !containsBoundaryText(row.MissingFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: blockers missing fact %q", fact)
		}
	}
	return nil
}

func rowsByID(rows []ActorRuntimeBoundaryRow) map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow {
	out := make(map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func containsBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
