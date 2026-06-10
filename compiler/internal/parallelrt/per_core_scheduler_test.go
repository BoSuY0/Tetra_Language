package parallelrt

import (
	"errors"
	"strings"
	"testing"
)

func TestPerCoreSchedulerCoverageCoversP18PlanList(t *testing.T) {
	report, err := PerCoreSchedulerCoverage()
	if err != nil {
		t.Fatalf("PerCoreSchedulerCoverage: %v", err)
	}
	if err := ValidatePerCoreSchedulerCoverage(report); err != nil {
		t.Fatalf("ValidatePerCoreSchedulerCoverage failed: %v", err)
	}
	if report.SchemaVersion != "tetra.parallel.per_core_scheduler.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionActorRuntimeClaimed {
		t.Fatalf("P18.2 must not claim a full production actor runtime")
	}
	if report.RaceDetectorAllTargetsClaimed {
		t.Fatalf("P18.2 must not claim race-detector coverage for every generated native target")
	}
	for _, want := range []string{
		"full production actor runtime is not claimed",
		"non-Linux distributed actor runtime targets are not promoted",
		"no scheduler performance claim",
	} {
		if !hasPerCoreText(report.NonClaims, want) {
			t.Fatalf("non-claims missing %q: %#v", want, report.NonClaims)
		}
	}

	byID := map[PerCoreSchedulerEvidenceID]PerCoreSchedulerEvidenceRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Status != PerCoreSchedulerImplementedNarrow {
			t.Fatalf("row %q status = %q, want %q", row.ID, row.Status, PerCoreSchedulerImplementedNarrow)
		}
		if row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing evidence or boundary: %#v", row.ID, row)
		}
	}
	expected := []PerCoreSchedulerEvidenceID{
		PerCoreSchedulerPerCoreQueues,
		PerCoreSchedulerWorkStealing,
		PerCoreSchedulerBoundedTypedMailboxes,
		PerCoreSchedulerBackpressure,
		PerCoreSchedulerTimersSleepWake,
		PerCoreSchedulerStructuredTaskGroups,
		PerCoreSchedulerCancellationCheckpoints,
		PerCoreSchedulerActorPingPong,
		PerCoreSchedulerFanoutFanin,
		PerCoreSchedulerTaskGroupCancel,
		PerCoreSchedulerBackpressureOverflow,
		PerCoreSchedulerMailboxFairness,
		PerCoreSchedulerStressRaceDetector,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P18.2 row %q", id)
		}
	}

	requirePerCoreFacts(t, byID[PerCoreSchedulerPerCoreQueues], "per-core queues", "single-core FIFO compatibility", "RunUntilIdle")
	requirePerCoreFacts(t, byID[PerCoreSchedulerWorkStealing], "work stealing", "Steals=1", "StepCore")
	requirePerCoreFacts(t, byID[PerCoreSchedulerBoundedTypedMailboxes], "bounded typed mailboxes", "TypedMailbox", "Capacity")
	requirePerCoreFacts(t, byID[PerCoreSchedulerBackpressure], "backpressure", "ErrMailboxFull", "blocking_recv_yield")
	requirePerCoreFacts(t, byID[PerCoreSchedulerTimersSleepWake], "timers", "sleep_ms", "wake in deadline order")
	requirePerCoreFacts(t, byID[PerCoreSchedulerStructuredTaskGroups], "structured task groups", "__tetra_task_group_open", "__tetra_task_group_close")
	requirePerCoreFacts(t, byID[PerCoreSchedulerCancellationCheckpoints], "cancellation checkpoints", "__tetra_task_checkpoint", "nested cancellation propagation")
	requirePerCoreFacts(t, byID[PerCoreSchedulerActorPingPong], "actor ping-pong", "actors_pingpong.tetra", "task actor mailbox handoff")
	requirePerCoreFacts(t, byID[PerCoreSchedulerFanoutFanin], "fanout/fanin", "actor fanout/fanin benchmark prep", "work stealing")
	requirePerCoreFacts(t, byID[PerCoreSchedulerTaskGroupCancel], "task group cancel", "cancel wakes deadline join", "cancellation storm")
	requirePerCoreFacts(t, byID[PerCoreSchedulerBackpressureOverflow], "backpressure overflow", "ErrMailboxFull", "TestActorRuntimeBuiltinCapacityLimitReturnsNoExtraActor")
	requirePerCoreFacts(t, byID[PerCoreSchedulerMailboxFairness], "mailbox fairness", "FIFO receive", "actor_self_mailbox_service")
	requirePerCoreFacts(t, byID[PerCoreSchedulerStressRaceDetector], "stress evidence", "many tasks stress", "race detector where applicable")
}

func TestPerCoreSchedulerCoverageRejectsFakeClaims(t *testing.T) {
	report, err := PerCoreSchedulerCoverage()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePerCoreSchedulerCoverage(report); err != nil {
		t.Fatalf("baseline report should validate: %v", err)
	}

	fakeActorRuntime := clonePerCoreSchedulerCoverage(report)
	fakeActorRuntime.FullProductionActorRuntimeClaimed = true
	if err := ValidatePerCoreSchedulerCoverage(fakeActorRuntime); err == nil || !strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake actor runtime claim error = %v", err)
	}

	fakeRace := clonePerCoreSchedulerCoverage(report)
	fakeRace.RaceDetectorAllTargetsClaimed = true
	if err := ValidatePerCoreSchedulerCoverage(fakeRace); err == nil || !strings.Contains(err.Error(), "race detector") {
		t.Fatalf("fake race-detector claim error = %v", err)
	}

	missingStress := clonePerCoreSchedulerCoverage(report)
	for i := range missingStress.Rows {
		if missingStress.Rows[i].ID == PerCoreSchedulerStressRaceDetector {
			missingStress.Rows[i].RequiredFacts = []string{"stress evidence"}
		}
	}
	if err := ValidatePerCoreSchedulerCoverage(missingStress); err == nil || !strings.Contains(err.Error(), "stress") {
		t.Fatalf("missing stress evidence error = %v", err)
	}

	fakeRuntimeChange := clonePerCoreSchedulerCoverage(report)
	for i := range fakeRuntimeChange.Rows {
		if fakeRuntimeChange.Rows[i].ID == PerCoreSchedulerWorkStealing {
			fakeRuntimeChange.Rows[i].ClaimsRuntimeBehaviorChange = true
		}
	}
	if err := ValidatePerCoreSchedulerCoverage(fakeRuntimeChange); err == nil || !strings.Contains(err.Error(), "runtime behavior") {
		t.Fatalf("fake runtime behavior change error = %v", err)
	}

	noNonClaim := clonePerCoreSchedulerCoverage(report)
	noNonClaim.NonClaims = nil
	if err := ValidatePerCoreSchedulerCoverage(noNonClaim); err == nil || !strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func TestTypedMailboxReceivePreservesFIFOAndBackpressureCapacity(t *testing.T) {
	box := NewTypedMailbox(MailboxConfig{
		Name:     "fair",
		Capacity: 2,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	for _, name := range []string{"first", "second"} {
		if _, err := box.Send(Message{Name: name}); err != nil {
			t.Fatalf("Send(%s): %v", name, err)
		}
	}
	if _, err := box.Send(Message{Name: "third"}); !errors.Is(err, ErrMailboxFull) {
		t.Fatalf("full Send error = %v, want ErrMailboxFull", err)
	}
	first, ok := box.Receive()
	if !ok || first.Name != "first" {
		t.Fatalf("first Receive = %#v, %v; want first", first, ok)
	}
	if _, err := box.Send(Message{Name: "third"}); err != nil {
		t.Fatalf("Send(third) after receive: %v", err)
	}
	second, ok := box.Receive()
	if !ok || second.Name != "second" {
		t.Fatalf("second Receive = %#v, %v; want second", second, ok)
	}
	third, ok := box.Receive()
	if !ok || third.Name != "third" {
		t.Fatalf("third Receive = %#v, %v; want third", third, ok)
	}
	_, ok = box.Receive()
	if ok {
		t.Fatalf("empty Receive returned ok")
	}
}

func requirePerCoreFacts(t *testing.T, row PerCoreSchedulerEvidenceRow, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !hasPerCoreText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func hasPerCoreText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func clonePerCoreSchedulerCoverage(report PerCoreSchedulerCoverageReport) PerCoreSchedulerCoverageReport {
	clone := report
	clone.Rows = append([]PerCoreSchedulerEvidenceRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
