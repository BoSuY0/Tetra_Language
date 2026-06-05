package parallelrt

import (
	"fmt"
	"strings"
)

type PerCoreSchedulerEvidenceID string

const (
	PerCoreSchedulerPerCoreQueues           PerCoreSchedulerEvidenceID = "per_core_queues"
	PerCoreSchedulerWorkStealing            PerCoreSchedulerEvidenceID = "work_stealing"
	PerCoreSchedulerBoundedTypedMailboxes   PerCoreSchedulerEvidenceID = "bounded_typed_mailboxes"
	PerCoreSchedulerBackpressure            PerCoreSchedulerEvidenceID = "backpressure"
	PerCoreSchedulerTimersSleepWake         PerCoreSchedulerEvidenceID = "timers_sleep_wake"
	PerCoreSchedulerStructuredTaskGroups    PerCoreSchedulerEvidenceID = "structured_task_groups"
	PerCoreSchedulerCancellationCheckpoints PerCoreSchedulerEvidenceID = "cancellation_checkpoints"
	PerCoreSchedulerActorPingPong           PerCoreSchedulerEvidenceID = "actor_ping_pong"
	PerCoreSchedulerFanoutFanin             PerCoreSchedulerEvidenceID = "fanout_fanin"
	PerCoreSchedulerTaskGroupCancel         PerCoreSchedulerEvidenceID = "task_group_cancel"
	PerCoreSchedulerBackpressureOverflow    PerCoreSchedulerEvidenceID = "backpressure_overflow"
	PerCoreSchedulerMailboxFairness         PerCoreSchedulerEvidenceID = "mailbox_fairness"
	PerCoreSchedulerStressRaceDetector      PerCoreSchedulerEvidenceID = "stress_race_detector_where_applicable"
)

type PerCoreSchedulerEvidenceStatus string

const (
	PerCoreSchedulerImplementedNarrow PerCoreSchedulerEvidenceStatus = "implemented_narrow"
)

type PerCoreSchedulerCoverageReport struct {
	SchemaVersion                     string                        `json:"schema_version"`
	Rows                              []PerCoreSchedulerEvidenceRow `json:"rows"`
	NonClaims                         []string                      `json:"non_claims"`
	FullProductionActorRuntimeClaimed bool                          `json:"full_production_actor_runtime_claimed"`
	RaceDetectorAllTargetsClaimed     bool                          `json:"race_detector_all_targets_claimed"`
}

type PerCoreSchedulerEvidenceRow struct {
	ID                          PerCoreSchedulerEvidenceID     `json:"id"`
	Name                        string                         `json:"name"`
	Status                      PerCoreSchedulerEvidenceStatus `json:"status"`
	RequiredFacts               []string                       `json:"required_facts,omitempty"`
	Evidence                    string                         `json:"evidence"`
	Boundary                    string                         `json:"boundary"`
	ClaimsRuntimeBehaviorChange bool                           `json:"claims_runtime_behavior_change,omitempty"`
	ClaimsFullActorRuntime      bool                           `json:"claims_full_actor_runtime,omitempty"`
}

func PerCoreSchedulerCoverage() (PerCoreSchedulerCoverageReport, error) {
	benchmarks, err := PrototypeBenchmarks()
	if err != nil {
		return PerCoreSchedulerCoverageReport{}, err
	}
	if len(benchmarks) < 2 {
		return PerCoreSchedulerCoverageReport{}, fmt.Errorf("per-core scheduler coverage: missing scheduler prototype benchmark rows")
	}
	return PerCoreSchedulerCoverageReport{
		SchemaVersion: "tetra.parallel.per_core_scheduler.v1",
		Rows: []PerCoreSchedulerEvidenceRow{
			perCoreQueuesRow(),
			workStealingRow(),
			boundedTypedMailboxesRow(),
			backpressureRow(),
			timersSleepWakeRow(),
			structuredTaskGroupsRow(),
			cancellationCheckpointsRow(),
			actorPingPongRow(),
			fanoutFaninRow(benchmarks),
			taskGroupCancelRow(),
			backpressureOverflowRow(),
			mailboxFairnessRow(),
			stressRaceDetectorRow(),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			"non-Linux distributed actor runtime targets are not promoted",
			"race-detector coverage is limited to applicable Go/model packages and does not claim every generated native target",
			"no scheduler performance claim is made",
			"P18.2 coverage does not change safe-program semantics or public runtime modes",
		},
		FullProductionActorRuntimeClaimed: false,
		RaceDetectorAllTargetsClaimed:     false,
	}, nil
}

func ValidatePerCoreSchedulerCoverage(report PerCoreSchedulerCoverageReport) error {
	if report.SchemaVersion != "tetra.parallel.per_core_scheduler.v1" {
		return fmt.Errorf("per-core scheduler coverage: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionActorRuntimeClaimed {
		return fmt.Errorf("per-core scheduler coverage: full production actor runtime claim is forbidden for P18.2")
	}
	if report.RaceDetectorAllTargetsClaimed {
		return fmt.Errorf("per-core scheduler coverage: race detector all-targets claim is forbidden for P18.2")
	}
	for _, want := range []string{
		"full production actor runtime is not claimed",
		"non-Linux distributed actor runtime targets are not promoted",
		"race-detector coverage is limited",
		"no scheduler performance claim",
	} {
		if !containsPerCoreSchedulerText(report.NonClaims, want) {
			return fmt.Errorf("per-core scheduler coverage: missing non-claim %q", want)
		}
	}

	expected := map[PerCoreSchedulerEvidenceID]bool{
		PerCoreSchedulerPerCoreQueues:           false,
		PerCoreSchedulerWorkStealing:            false,
		PerCoreSchedulerBoundedTypedMailboxes:   false,
		PerCoreSchedulerBackpressure:            false,
		PerCoreSchedulerTimersSleepWake:         false,
		PerCoreSchedulerStructuredTaskGroups:    false,
		PerCoreSchedulerCancellationCheckpoints: false,
		PerCoreSchedulerActorPingPong:           false,
		PerCoreSchedulerFanoutFanin:             false,
		PerCoreSchedulerTaskGroupCancel:         false,
		PerCoreSchedulerBackpressureOverflow:    false,
		PerCoreSchedulerMailboxFairness:         false,
		PerCoreSchedulerStressRaceDetector:      false,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf("per-core scheduler coverage: row count = %d, want %d", len(report.Rows), len(expected))
	}
	rows := map[PerCoreSchedulerEvidenceID]PerCoreSchedulerEvidenceRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("per-core scheduler coverage: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("per-core scheduler coverage: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("per-core scheduler coverage: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		rows[row.ID] = row
		if row.Status != PerCoreSchedulerImplementedNarrow {
			return fmt.Errorf("per-core scheduler coverage: row %q status = %q", row.ID, row.Status)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("per-core scheduler coverage: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("per-core scheduler coverage: row %q missing required facts", row.ID)
		}
		if row.ClaimsRuntimeBehaviorChange {
			return fmt.Errorf("per-core scheduler coverage: row %q claims runtime behavior change", row.ID)
		}
		if row.ClaimsFullActorRuntime {
			return fmt.Errorf("per-core scheduler coverage: row %q claims full production actor runtime", row.ID)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("per-core scheduler coverage: missing row %q", id)
		}
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerPerCoreQueues], "per-core queues", "single-core FIFO compatibility", "RunUntilIdle"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerWorkStealing], "work stealing", "Steals=1", "StepCore"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerBoundedTypedMailboxes], "bounded typed mailboxes", "TypedMailbox", "Capacity"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerBackpressure], "backpressure", "ErrMailboxFull", "blocking_recv_yield"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerTimersSleepWake], "timers", "sleep_ms", "wake in deadline order"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerStructuredTaskGroups], "structured task groups", "__tetra_task_group_open", "__tetra_task_group_close"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerCancellationCheckpoints], "cancellation checkpoints", "__tetra_task_checkpoint", "nested cancellation propagation"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerActorPingPong], "actor ping-pong", "actors_pingpong.tetra", "task actor mailbox handoff"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerFanoutFanin], "fanout/fanin", "actor ping-pong fanout scheduler prototype", "work stealing"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerTaskGroupCancel], "task group cancel", "cancel wakes deadline join", "cancellation storm"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerBackpressureOverflow], "backpressure overflow", "ErrMailboxFull", "TestActorRuntimeBuiltinCapacityLimitReturnsNoExtraActor"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerMailboxFairness], "mailbox fairness", "FIFO receive", "actor_self_mailbox_service"); err != nil {
		return err
	}
	if err := requirePerCoreSchedulerFacts(rows[PerCoreSchedulerStressRaceDetector], "stress evidence", "many tasks stress", "many actor messages stress", "cancellation storm", "timeouts stress", "race detector where applicable"); err != nil {
		return fmt.Errorf("per-core scheduler coverage: stress row invalid: %w", err)
	}
	return nil
}

func perCoreQueuesRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerPerCoreQueues,
		Name:   "Per-core queues",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"per-core queues are allocated per configured core",
			"single-core FIFO compatibility is covered by RunUntilIdle",
			"RunUntilIdle walks every core until no work remains",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::NewSchedulerModel; compiler/internal/parallelrt/scheduler_model.go::RunUntilIdle; compiler/internal/parallelrt/scheduler_model_test.go::TestSchedulerModelRunsSingleCoreFIFO",
		Boundary: "per-core queue evidence covers the checked scheduler model and release parallel runtime gates; it does not promote the built-in actor runtime to a full production actor runtime",
	}
}

func workStealingRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerWorkStealing,
		Name:   "Work stealing",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"work stealing lets an idle core take work from another core",
			"StepCore records stolen work from another queue",
			"Steals=1 in the two-core scheduler model test",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::StepCore; compiler/internal/parallelrt/scheduler_model_test.go::TestSchedulerModelStealsWorkAcrossTwoCores",
		Boundary: "work-stealing evidence is scheduler-model and parallel production gate evidence; this change does not alter runtime scheduling behavior",
	}
}

func boundedTypedMailboxesRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerBoundedTypedMailboxes,
		Name:   "Bounded typed mailboxes",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"bounded typed mailboxes are represented by TypedMailbox",
			"TypedMailbox Capacity is stable and positive",
			"typed mailbox ownership metadata is preserved",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::TypedMailbox; compiler/internal/parallelrt/scheduler_model_test.go::TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata; compiler/actors_test.go::TestActorsTypedMailboxExplainReportIncludesMetadataAndCopyMove",
		Boundary: "bounded mailbox evidence covers typed mailbox model/report paths and existing actor mailbox smokes; it does not add a new distributed mailbox protocol",
	}
}

func backpressureRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerBackpressure,
		Name:   "Backpressure",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"backpressure uses blocking_recv_yield metadata",
			"ErrMailboxFull is returned when bounded model capacity is exceeded",
			"parallel production validator requires actor mailbox backpressure evidence",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::TypedMailbox.Send; compiler/internal/parallelrt/scheduler_model_test.go::TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata; tools/validators/parallelprod/report.go::validateCases",
		Boundary: "backpressure evidence is checked model/report plus existing runtime smoke evidence; built-in actor message-pool checked exhaustion remains a P18.0 blocker",
	}
}

func timersSleepWakeRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerTimersSleepWake,
		Name:   "Timers and sleep/wake",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"timers use deadline_ms and timer_ready runtime symbols",
			"sleep_ms and sleep_until advance the logical runtime clock",
			"wake in deadline order is covered by task sleep timers",
		},
		Evidence: "compiler/task_runtime_test.go::TestTaskSleepTimersWakeInDeadlineOrderBuildAndRun; compiler/task_runtime_test.go::TestTimerReadyAndSelect2TaskTimerBuildAndRun; compiler/task_runtime_test.go::TestSleepUntilUsesAbsoluteDeadlineBuildAndRun",
		Boundary: "timer evidence is executable Linux-x64 runtime behavior and self-host parity evidence; it does not claim wall-clock performance or every target runtime",
	}
}

func structuredTaskGroupsRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerStructuredTaskGroups,
		Name:   "Structured task groups",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"structured task groups lower to __tetra_task_group_open",
			"structured task groups lower to __tetra_task_group_close",
			"task_group_current and task_group_status expose group lifecycle state",
		},
		Evidence: "compiler/task_runtime_test.go::TestRequiredTaskGroupRuntimeSymbolsIncludeCancellationABI; compiler/task_runtime_test.go::TestTaskGroupLowersToRuntimeCalls; compiler/task_runtime_test.go::TestTaskGroupCurrentVisibleInGroupTaskBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCloseMarksOpenGroupClosedBuildAndRun",
		Boundary: "task-group evidence covers current cooperative structured task groups; full structured concurrency guarantees across actors and distributed runtime remain outside this row",
	}
}

func cancellationCheckpointsRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerCancellationCheckpoints,
		Name:   "Cancellation checkpoints",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"cancellation checkpoints lower to __tetra_task_checkpoint",
			"task cancellation status lowers to __tetra_task_is_canceled",
			"nested cancellation propagation reaches child tasks",
		},
		Evidence: "compiler/task_runtime_test.go::TestTaskCancellationCheckpointLowersToRuntimeCalls; compiler/task_runtime_test.go::TestTaskCancellationCheckpointSeesSelfCanceledGroupBuildAndRun; compiler/task_runtime_test.go::TestTaskCancellationCheckpointInheritedByNestedChildBuildAndRun",
		Boundary: "checkpoint evidence covers cooperative task cancellation in the current runtime; it is not a full preemptive cancellation model",
	}
}

func actorPingPongRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerActorPingPong,
		Name:   "Actor ping-pong",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"actor ping-pong runs examples/actors_pingpong.tetra",
			"task actor mailbox handoff is executable with recv_until",
			"actor ping-pong remains bounded to current actor runtime limits",
		},
		Evidence: "compiler/actors_test.go::TestActorsPingPongBuildAndRun; examples/actors_pingpong.tetra; compiler/task_runtime_test.go::TestTaskSpawnsActorAndReceivesMailboxReplyBuildAndRun",
		Boundary: "actor ping-pong evidence is local actor/task mailbox behavior and does not claim distributed actor runtime promotion",
	}
}

func fanoutFaninRow(benchmarks []PrototypeBenchmark) PerCoreSchedulerEvidenceRow {
	names := make([]string, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		if benchmark.Ran && benchmark.Pass {
			names = append(names, benchmark.Name)
		}
	}
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerFanoutFanin,
		Name:   "Fanout/fanin",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"fanout/fanin evidence includes actor ping-pong fanout scheduler prototype",
			"work stealing reduces max_queue_depth in the prototype comparison",
			"prototype benchmark rows passed: " + strings.Join(names, "; "),
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::PrototypeBenchmarks; compiler/internal/parallelrt/scheduler_model_test.go::TestPrototypeBenchmarksReportFanoutAndZeroCopyRows; tools/cmd/parallel-production-smoke/main.go::runSchedulerPrototypeEvidence",
		Boundary: "fanout/fanin evidence is a bounded model benchmark and release report row, not a broad throughput or scheduler-performance claim",
	}
}

func taskGroupCancelRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerTaskGroupCancel,
		Name:   "Task group cancel",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"task group cancel wakes deadline join",
			"task group cancel returns canceled task error",
			"cancellation storm stress exercises repeated cancel/join cycles",
		},
		Evidence: "compiler/task_runtime_test.go::TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun; compiler/task_runtime_test.go::TestTaskGroupCancelAfterSpawnBeforeJoinBuildAndRun; tools/cmd/parallel-production-smoke/main.go::cancellationStormSource",
		Boundary: "task-group cancel evidence covers cooperative current-runtime behavior and stress smokes; it is not a full actor-runtime structured-concurrency proof",
	}
}

func backpressureOverflowRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerBackpressureOverflow,
		Name:   "Backpressure overflow",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"backpressure overflow returns ErrMailboxFull in the typed mailbox model",
			"blocking_recv_yield backpressure metadata is preserved",
			"TestActorRuntimeBuiltinCapacityLimitReturnsNoExtraActor covers built-in actor capacity failure boundary",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model_test.go::TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata; compiler/actors_test.go::TestActorRuntimeBuiltinCapacityLimitReturnsNoExtraActor; tools/validators/parallelprod/report.go::validateCases",
		Boundary: "overflow evidence covers typed mailbox model failure plus current actor capacity smoke; checked message-pool reclamation remains outside this row",
	}
}

func mailboxFairnessRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerMailboxFairness,
		Name:   "Mailbox fairness",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"mailbox fairness uses FIFO receive order in the typed mailbox model",
			"FIFO receive frees capacity before the next send",
			"actor_self_mailbox_service smoke records current actor mailbox ordering evidence",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::TypedMailbox.Receive; compiler/internal/parallelrt/per_core_scheduler_test.go::TestTypedMailboxReceivePreservesFIFOAndBackpressureCapacity; examples/microservices/actor_self_mailbox_service.tetra",
		Boundary: "mailbox fairness evidence covers FIFO order for the bounded model and existing smoke examples; it is not a full fairness proof for distributed actor mailboxes",
	}
}

func stressRaceDetectorRow() PerCoreSchedulerEvidenceRow {
	return PerCoreSchedulerEvidenceRow{
		ID:     PerCoreSchedulerStressRaceDetector,
		Name:   "Stress and race-detector scope",
		Status: PerCoreSchedulerImplementedNarrow,
		RequiredFacts: []string{
			"stress evidence covers many tasks stress",
			"stress evidence covers many actor messages stress",
			"stress evidence covers cancellation storm",
			"stress evidence covers timeouts stress",
			"race detector where applicable is limited to Go/model package gates such as go test -race ./compiler/internal/parallelrt",
		},
		Evidence: "tools/cmd/parallel-production-smoke/main.go::requiredPassingCases; tools/validators/parallelprod/report.go::validateCases; compiler/internal/parallelrt/per_core_scheduler_test.go::TestPerCoreSchedulerCoverageCoversP18PlanList",
		Boundary: "stress evidence is bounded to existing smokes and applicable Go/model race-detector gates; generated native binaries are not claimed to run under Go's race detector",
	}
}

func requirePerCoreSchedulerFacts(row PerCoreSchedulerEvidenceRow, wants ...string) error {
	for _, want := range wants {
		if !containsPerCoreSchedulerText(row.RequiredFacts, want) {
			return fmt.Errorf("row %q missing fact %q", row.ID, want)
		}
	}
	return nil
}

func containsPerCoreSchedulerText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
