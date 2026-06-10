package semantics

import (
	"strings"
	"testing"
)

func TestMemoryBoundaryHandoffAuditCoversP10PlanRows(t *testing.T) {
	report := MemoryBoundaryHandoffAudit()
	if err := ValidateMemoryBoundaryHandoffAudit(report); err != nil {
		t.Fatalf("ValidateMemoryBoundaryHandoffAudit failed: %v", err)
	}
	if report.SchemaVersion != "tetra.memory.boundary_handoff.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullActorRuntimeClaimed {
		t.Fatalf("P10 audit must not claim full actor runtime production")
	}
	if !hasBoundaryHandoffText(report.NonClaims, "full production actor runtime is not claimed") {
		t.Fatalf("nonclaims = %#v, want actor runtime nonclaim", report.NonClaims)
	}

	byID := map[MemoryBoundaryHandoffID]MemoryBoundaryHandoffRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []MemoryBoundaryHandoffID{
		MemoryBoundaryActorBorrowRejected,
		MemoryBoundaryTaskBorrowRejected,
		MemoryBoundaryRequestRegionScoped,
		MemoryBoundaryUnsafeSafeMessageRejected,
		MemoryBoundaryStaleEpochRejected,
		MemoryBoundaryIslandMoveLinear,
		MemoryBoundaryActorRuntimeNonClaim,
	}
	for _, id := range expected {
		row, ok := byID[id]
		if !ok {
			t.Fatalf("missing P10 boundary handoff row %q", id)
		}
		if row.Status != MemoryBoundaryImplementedNarrow {
			t.Fatalf("row %q status = %q, want %q", id, row.Status, MemoryBoundaryImplementedNarrow)
		}
		if row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing evidence/boundary: %#v", id, row)
		}
	}

	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryActorBorrowRejected], "cannot send borrowed view across actor boundary", ".copy()")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryTaskBorrowRejected], "typed task error payload must be sendable across task boundary")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryRequestRegionScoped], "RequestRegionScope", "TaskRegionScope", "reset")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryUnsafeSafeMessageRejected], "ptr", "cap.mem", "typed actor message payload must be value-only")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryStaleEpochRejected], "core.island_reset", "cannot use consumed value")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryIslandMoveLinear], "core.send_typed", "cannot use consumed value", "island")
	requireBoundaryHandoffFacts(t, byID[MemoryBoundaryActorRuntimeNonClaim], "not a production actor runtime")
}

func TestMemoryBoundaryHandoffAuditRejectsFakeScopeClaims(t *testing.T) {
	report := MemoryBoundaryHandoffAudit()

	fakeClaim := report
	fakeClaim.FullActorRuntimeClaimed = true
	if err := ValidateMemoryBoundaryHandoffAudit(fakeClaim); err == nil || !strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake full actor runtime claim error = %v", err)
	}

	missingStale := cloneMemoryBoundaryHandoffReport(report)
	var rows []MemoryBoundaryHandoffRow
	for _, row := range missingStale.Rows {
		if row.ID != MemoryBoundaryStaleEpochRejected {
			rows = append(rows, row)
		}
	}
	missingStale.Rows = rows
	if err := ValidateMemoryBoundaryHandoffAudit(missingStale); err == nil || !strings.Contains(err.Error(), "stale_epoch_rejected") {
		t.Fatalf("missing stale row error = %v", err)
	}

	noNonclaim := cloneMemoryBoundaryHandoffReport(report)
	noNonclaim.NonClaims = nil
	if err := ValidateMemoryBoundaryHandoffAudit(noNonclaim); err == nil || !strings.Contains(err.Error(), "nonclaim") {
		t.Fatalf("missing nonclaim error = %v", err)
	}
}

func requireBoundaryHandoffFacts(t *testing.T, row MemoryBoundaryHandoffRow, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !hasBoundaryHandoffText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func hasBoundaryHandoffText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneMemoryBoundaryHandoffReport(report MemoryBoundaryHandoffReport) MemoryBoundaryHandoffReport {
	clone := report
	clone.Rows = append([]MemoryBoundaryHandoffRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
