package actorsrt

import (
	"strings"
	"testing"
)

func TestActorRuntimeProductionBoundaryAuditCoversP18PlanList(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(report); err != nil {
		t.Fatalf("ValidateActorRuntimeProductionBoundaryAudit failed: %v", err)
	}
	if report.SchemaVersion != "tetra.runtime.actor.production_boundary.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionClaimed {
		t.Fatalf("P18.0 audit must not claim a full production actor runtime")
	}
	if !hasActorBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		t.Fatalf("non-claims = %#v, want full production actor runtime non-claim", report.NonClaims)
	}

	byID := map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []ActorRuntimeBoundaryID{
		ActorRuntimeBoundaryCurrentLimits,
		ActorRuntimeBoundarySchedulerPrototype,
		ActorRuntimeBoundaryProductionAcceptance,
		ActorRuntimeBoundaryFullClaimBlockers,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P18.0 audit row %q", id)
		}
	}

	limits := byID[ActorRuntimeBoundaryCurrentLimits]
	if limits.Status != ActorRuntimeBoundaryDocumentedLimit {
		t.Fatalf("current limits status = %q, want %q", limits.Status, ActorRuntimeBoundaryDocumentedLimit)
	}
	for _, want := range []string{"maxActors=128", "msgPoolSize=65536", "maxActorMailboxMsgs=256", "actor_state_slots=8", "single-thread cooperative scheduler", "linux-x64 distributed runtime only", "non-linux actor net pump is no-op", "mailbox full returns checked -2", "message pool exhaustion returns checked -1", "invalid actor handle sends return checked -3", "done actor sends return checked -4", "task-group cancellation wakes timed actor receive waiters", "message pool entries are not reclaimed"} {
		if !hasActorBoundaryText(limits.RequiredFacts, want) {
			t.Fatalf("current limits row missing fact %q: %#v", want, limits.RequiredFacts)
		}
	}
	for _, want := range []string{"compiler/internal/actorsrt/linux_x64.go", "emitMailboxFullCheckForReceiverInEcx", "emitCheckedMessagePoolAlloc", "emitInvalidActorHandleReturn", "emitActorDoneReturn", "emitBlockedDeadlineWakeCheck", "emitCurrentTaskGroupCanceledCheck", "TestActorMailboxFullReturnsCheckedBackpressure", "TestActorMessagePoolExhaustionReturnsCheckedFailure", "TestActorInvalidHandleSendReturnsCheckedFailure", "TestActorSendToDoneActorReturnsCheckedFailure", "TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun", "docs/spec/actors.md", "TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump"} {
		if !strings.Contains(limits.Evidence, want) {
			t.Fatalf("current limits evidence missing %q: %s", want, limits.Evidence)
		}
	}

	prototype := byID[ActorRuntimeBoundarySchedulerPrototype]
	if prototype.Status != ActorRuntimeBoundaryPrototypeEvidence {
		t.Fatalf("scheduler prototype status = %q, want %q", prototype.Status, ActorRuntimeBoundaryPrototypeEvidence)
	}
	for _, want := range []string{"single-core FIFO compatibility", "two-core work stealing", "bounded typed mailbox", "zero_copy_move", "bytes_copied=0"} {
		if !hasActorBoundaryText(prototype.RequiredFacts, want) {
			t.Fatalf("scheduler prototype row missing fact %q: %#v", want, prototype.RequiredFacts)
		}
	}
	if !strings.Contains(prototype.Boundary, "not a production multi-threaded actor scheduler") {
		t.Fatalf("scheduler prototype boundary = %q", prototype.Boundary)
	}

	acceptance := byID[ActorRuntimeBoundaryProductionAcceptance]
	if acceptance.Status != ActorRuntimeBoundaryAcceptanceRequired {
		t.Fatalf("production acceptance status = %q, want %q", acceptance.Status, ActorRuntimeBoundaryAcceptanceRequired)
	}
	for _, want := range []string{"production task scheduler", "bounded mailbox backpressure", "message reclamation", "race-safety model", "cross-target distributed runtime gates", "structured concurrency"} {
		if !hasActorBoundaryText(acceptance.RequiredFacts, want) {
			t.Fatalf("production acceptance row missing fact %q: %#v", want, acceptance.RequiredFacts)
		}
	}

	blockers := byID[ActorRuntimeBoundaryFullClaimBlockers]
	if blockers.Status != ActorRuntimeBoundaryBlocked {
		t.Fatalf("blockers status = %q, want %q", blockers.Status, ActorRuntimeBoundaryBlocked)
	}
	for _, want := range []string{"production multi-threaded actor scheduler", "non-Linux-x64 distributed actor runtime", "full cancellation and structured concurrency", "full race-safety proof"} {
		if !hasActorBoundaryText(blockers.MissingFacts, want) {
			t.Fatalf("blockers row missing fact %q: %#v", want, blockers.MissingFacts)
		}
	}
}

func TestActorRuntimeProductionBoundaryAuditRejectsFakeFullProductionClaim(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}

	fakeClaim := report
	fakeClaim.FullProductionClaimed = true
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakeClaim); err == nil || !strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake full-production claim error = %v", err)
	}

	missingBlockers := cloneActorRuntimeBoundaryReport(report)
	for i := range missingBlockers.Rows {
		if missingBlockers.Rows[i].ID == ActorRuntimeBoundaryFullClaimBlockers {
			missingBlockers.Rows[i].MissingFacts = nil
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(missingBlockers); err == nil || !strings.Contains(err.Error(), "blockers") {
		t.Fatalf("missing blocker facts error = %v", err)
	}

	fakePromotion := cloneActorRuntimeBoundaryReport(report)
	for i := range fakePromotion.Rows {
		if fakePromotion.Rows[i].ID == ActorRuntimeBoundarySchedulerPrototype {
			fakePromotion.Rows[i].Status = ActorRuntimeBoundaryStatus("production_ready")
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakePromotion); err == nil || !strings.Contains(err.Error(), "scheduler prototype") {
		t.Fatalf("fake scheduler promotion error = %v", err)
	}

	noNonClaim := cloneActorRuntimeBoundaryReport(report)
	noNonClaim.NonClaims = nil
	if err := ValidateActorRuntimeProductionBoundaryAudit(noNonClaim); err == nil || !strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func hasActorBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneActorRuntimeBoundaryReport(report ActorRuntimeBoundaryReport) ActorRuntimeBoundaryReport {
	clone := report
	clone.Rows = append([]ActorRuntimeBoundaryRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
