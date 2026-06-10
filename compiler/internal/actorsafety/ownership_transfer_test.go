package actorsafety

import (
	"strings"
	"testing"
)

func TestTypedActorOwnershipTransferCoverageCoversP18PlanList(t *testing.T) {
	report := TypedActorOwnershipTransferCoverage()
	if err := ValidateTypedActorOwnershipTransferCoverage(report); err != nil {
		t.Fatalf("ValidateTypedActorOwnershipTransferCoverage failed: %v", err)
	}
	if report.SchemaVersion != "tetra.actors.ownership_transfer.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.DistributedZeroCopyClaimed {
		t.Fatalf("P18.1 must not claim distributed zero-copy ownership transfer")
	}
	if report.RuntimeBehaviorChanged {
		t.Fatalf("P18.1 coverage must not claim new runtime behavior")
	}
	if !hasOwnershipTransferText(report.NonClaims, "distributed pointer or region zero-copy is not claimed") {
		t.Fatalf("non-claims = %#v, want distributed zero-copy non-claim", report.NonClaims)
	}

	byID := map[TypedActorOwnershipTransferID]TypedActorOwnershipTransferRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []TypedActorOwnershipTransferID{
		TypedActorOwnershipBorrowedViewCopyBoundary,
		TypedActorOwnershipOwnedRegionMove,
		TypedActorOwnershipSenderUseAfterMove,
		TypedActorOwnershipReceiverOwnsMovedRegion,
		TypedActorOwnershipExplicitCopyFallback,
		TypedActorOwnershipUnsafeSendContract,
		TypedActorOwnershipSemanticsChecker,
		TypedActorOwnershipPLIRMovedFacts,
		TypedActorOwnershipRuntimeMailboxRepresentation,
		TypedActorOwnershipActorTransferReport,
		TypedActorOwnershipStressDiagnostics,
		TypedActorOwnershipRaceSafetyRejectionMatrix,
	}
	for _, id := range expected {
		row, ok := byID[id]
		if !ok {
			t.Fatalf("missing P18.1 row %q", id)
		}
		if row.Status != TypedActorOwnershipImplementedNarrow {
			t.Fatalf("row %q status = %q, want %q", id, row.Status, TypedActorOwnershipImplementedNarrow)
		}
		if strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			t.Fatalf("row %q missing evidence/boundary: %#v", id, row)
		}
	}

	for _, want := range []string{"borrowed view rejected", "explicit copy accepted", ".copy()"} {
		if !hasOwnershipTransferText(byID[TypedActorOwnershipBorrowedViewCopyBoundary].RequiredFacts, want) {
			t.Fatalf("borrowed-view row missing fact %q: %#v", want, byID[TypedActorOwnershipBorrowedViewCopyBoundary].RequiredFacts)
		}
	}
	for _, want := range []string{"owned region can move", "bytes_copied=0", "zero_copy_move"} {
		if !hasOwnershipTransferText(byID[TypedActorOwnershipOwnedRegionMove].RequiredFacts, want) {
			t.Fatalf("owned-region row missing fact %q: %#v", want, byID[TypedActorOwnershipOwnedRegionMove].RequiredFacts)
		}
	}
	for _, want := range []string{"cannot use consumed value", "cannot use moved region after send"} {
		if !hasOwnershipTransferText(byID[TypedActorOwnershipSenderUseAfterMove].RequiredFacts, want) {
			t.Fatalf("sender-use row missing fact %q: %#v", want, byID[TypedActorOwnershipSenderUseAfterMove].RequiredFacts)
		}
	}
	for _, want := range []string{"receiver owns moved region", "RegionOwner(frame) = actor1", "recv_typed"} {
		if !hasOwnershipTransferText(byID[TypedActorOwnershipReceiverOwnsMovedRegion].RequiredFacts, want) {
			t.Fatalf("receiver row missing fact %q: %#v", want, byID[TypedActorOwnershipReceiverOwnsMovedRegion].RequiredFacts)
		}
	}
	for _, want := range []string{"FactMoved", "OpActorSend", "core.send_typed"} {
		if !hasOwnershipTransferText(byID[TypedActorOwnershipPLIRMovedFacts].RequiredFacts, want) {
			t.Fatalf("PLIR row missing fact %q: %#v", want, byID[TypedActorOwnershipPLIRMovedFacts].RequiredFacts)
		}
	}
}

func TestTypedActorOwnershipTransferCoverageRejectsFakeClaims(t *testing.T) {
	report := TypedActorOwnershipTransferCoverage()

	distributed := cloneTypedActorOwnershipTransferReport(report)
	distributed.DistributedZeroCopyClaimed = true
	if err := ValidateTypedActorOwnershipTransferCoverage(distributed); err == nil || !strings.Contains(err.Error(), "distributed zero-copy") {
		t.Fatalf("distributed zero-copy fake claim error = %v", err)
	}

	runtimeChanged := cloneTypedActorOwnershipTransferReport(report)
	runtimeChanged.RuntimeBehaviorChanged = true
	if err := ValidateTypedActorOwnershipTransferCoverage(runtimeChanged); err == nil || !strings.Contains(err.Error(), "runtime behavior") {
		t.Fatalf("runtime behavior fake claim error = %v", err)
	}

	missingPLIR := cloneTypedActorOwnershipTransferReport(report)
	for i := range missingPLIR.Rows {
		if missingPLIR.Rows[i].ID == TypedActorOwnershipPLIRMovedFacts {
			missingPLIR.Rows[i].RequiredFacts = []string{"core.send_typed exists"}
		}
	}
	if err := ValidateTypedActorOwnershipTransferCoverage(missingPLIR); err == nil || !strings.Contains(err.Error(), "PLIR moved facts") {
		t.Fatalf("missing PLIR moved facts error = %v", err)
	}

	missingStress := cloneTypedActorOwnershipTransferReport(report)
	for i := range missingStress.Rows {
		if missingStress.Rows[i].ID == TypedActorOwnershipStressDiagnostics {
			missingStress.Rows[i].RequiredFacts = nil
		}
	}
	if err := ValidateTypedActorOwnershipTransferCoverage(missingStress); err == nil || !strings.Contains(err.Error(), "stress diagnostics") {
		t.Fatalf("missing stress diagnostics error = %v", err)
	}

	noNonClaim := cloneTypedActorOwnershipTransferReport(report)
	noNonClaim.NonClaims = nil
	if err := ValidateTypedActorOwnershipTransferCoverage(noNonClaim); err == nil || !strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func TestTypedActorOwnershipTransferCoverageRequiresRaceSafetyMatrix(t *testing.T) {
	report := TypedActorOwnershipTransferCoverage()
	var matrix *TypedActorOwnershipTransferRow
	for i := range report.Rows {
		if report.Rows[i].ID == TypedActorOwnershipTransferID("race_safety_rejection_matrix") {
			matrix = &report.Rows[i]
			break
		}
	}
	if matrix == nil {
		t.Fatalf("missing race_safety_rejection_matrix row")
	}
	for _, want := range []string{
		"local immutable payloads copy safely",
		"owned moved payloads consume sender access",
		"borrowed views rejected unless explicitly copied",
		"mutable global targets rejected across actor/task boundaries",
		"unsafe pointer payloads rejected without audited contract",
		"island region transfer deferred to actor/island proof",
	} {
		if !hasOwnershipTransferText(matrix.RequiredFacts, want) {
			t.Fatalf("race-safety matrix row missing fact %q: %#v", want, matrix.RequiredFacts)
		}
	}
	for _, want := range []string{
		"compiler/tests/ownership/actor_task_ownership_test.go",
		"compiler/internal/actorsafety/sendability_test.go",
		"compiler/internal/semantics/memory_boundary_handoff_test.go",
	} {
		if !strings.Contains(matrix.Evidence, want) {
			t.Fatalf("race-safety matrix evidence missing %q: %s", want, matrix.Evidence)
		}
	}
	if !strings.Contains(matrix.Boundary, "not a lock/atomic shared-memory model") {
		t.Fatalf("race-safety matrix boundary missing nonclaim: %s", matrix.Boundary)
	}
}

func hasOwnershipTransferText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneTypedActorOwnershipTransferReport(report TypedActorOwnershipTransferReport) TypedActorOwnershipTransferReport {
	clone := report
	clone.Rows = append([]TypedActorOwnershipTransferRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
