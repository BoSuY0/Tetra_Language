package netrt

import (
	"strings"
	"testing"
)

func TestIOReactorCoverageCoversP18PlanList(t *testing.T) {
	report, err := IOReactorCoverage()
	if err != nil {
		t.Fatalf("IOReactorCoverage: %v", err)
	}
	if err := ValidateIOReactorCoverage(report); err != nil {
		t.Fatalf("ValidateIOReactorCoverage failed: %v", err)
	}
	if report.SchemaVersion != "tetra.runtime.io_reactor.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed {
		t.Fatalf("P18.3 must not claim a full production web stack")
	}
	if report.CrossPlatformParityClaimed {
		t.Fatalf("P18.3 must not claim cross-platform reactor parity")
	}
	if report.IOUringClaimed {
		t.Fatalf("P18.3 must not claim io_uring support")
	}
	if report.RuntimeBehaviorChanged {
		t.Fatalf("P18.3 coverage must be report-only and not change runtime behavior")
	}
	for _, want := range []string{
		"full production web stack is not claimed",
		"cross-platform reactor parity is not claimed",
		"io_uring is not implemented",
		"runtime behavior is unchanged",
	} {
		if !hasIOReactorText(report.NonClaims, want) {
			t.Fatalf("non-claims missing %q: %#v", want, report.NonClaims)
		}
	}

	byID := map[IOReactorEvidenceID]IOReactorEvidenceRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing evidence or boundary: %#v", row.ID, row)
		}
	}
	expected := []IOReactorEvidenceID{
		IOReactorLinuxEpollV1,
		IOReactorIOUringFuture,
		IOReactorKqueueBoundary,
		IOReactorIOCPBoundary,
		IOReactorWASIWebBoundary,
		IOReactorNonblockingAcceptReadWrite,
		IOReactorReadinessPolling,
		IOReactorTaskWakeupsFromIO,
		IOReactorTimerIntegration,
		IOReactorCancellation,
		IOReactorBackpressure,
		IOReactorReportRows,
		IOReactorHTTPSmoke,
		IOReactorDBSmoke,
		IOReactorStressEvidence,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P18.3 row %q", id)
		}
	}

	requireIOReactorFacts(t, byID[IOReactorLinuxEpollV1], "epoll v1", "NewPoller", "EPOLLIN", "EPOLLOUT")
	requireIOReactorFacts(t, byID[IOReactorIOUringFuture], "io_uring later after epoll stable", "not implemented")
	requireIOReactorFacts(t, byID[IOReactorKqueueBoundary], "kqueue macOS", "not implemented")
	requireIOReactorFacts(t, byID[IOReactorIOCPBoundary], "IOCP Windows", "not implemented")
	requireIOReactorFacts(t, byID[IOReactorWASIWebBoundary], "WASI/web event adapters", "not implemented")
	requireIOReactorFacts(t, byID[IOReactorNonblockingAcceptReadWrite], "Accept4", "SOCK_NONBLOCK", "Read", "Write")
	requireIOReactorFacts(t, byID[IOReactorReadinessPolling], "Poller.Wait", "wait-one readiness", "TestNetRuntimeEpollReadiness")
	requireIOReactorFacts(t, byID[IOReactorTaskWakeupsFromIO], "I/O readiness wakes", "poller.Wait", "core.task_spawn_i32")
	requireIOReactorFacts(t, byID[IOReactorTimerIntegration], "50ms poll timeout", "sleep_ms", "wake in deadline order")
	requireIOReactorFacts(t, byID[IOReactorCancellation], "context cancellation", "Close", "task_group_cancel")
	requireIOReactorFacts(t, byID[IOReactorBackpressure], "EPOLLOUT", "ErrPoolExhausted", "output buffer")
	requireIOReactorFacts(t, byID[IOReactorReportRows], "tetra.techempower.benchmark.v1", "ValidateReport")
	requireIOReactorFacts(t, byID[IOReactorHTTPSmoke], "plaintext", "json", "pipelining")
	requireIOReactorFacts(t, byID[IOReactorDBSmoke], "PostgreSQL", "prepared statement", "pool")
	requireIOReactorFacts(t, byID[IOReactorStressEvidence], "go test -race ./compiler/internal/netrt", "many readiness waits")

	if byID[IOReactorLinuxEpollV1].Status != IOReactorImplementedNarrow {
		t.Fatalf("linux epoll status = %q, want %q", byID[IOReactorLinuxEpollV1].Status, IOReactorImplementedNarrow)
	}
	for _, id := range []IOReactorEvidenceID{IOReactorIOUringFuture, IOReactorKqueueBoundary, IOReactorIOCPBoundary, IOReactorWASIWebBoundary} {
		if byID[id].Status != IOReactorBoundaryDocumented {
			t.Fatalf("platform boundary row %q status = %q, want %q", id, byID[id].Status, IOReactorBoundaryDocumented)
		}
	}
}

func TestIOReactorCoverageRejectsFakeClaims(t *testing.T) {
	report, err := IOReactorCoverage()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateIOReactorCoverage(report); err != nil {
		t.Fatalf("baseline report should validate: %v", err)
	}

	fakeWebStack := cloneIOReactorCoverage(report)
	fakeWebStack.FullProductionWebStackClaimed = true
	if err := ValidateIOReactorCoverage(fakeWebStack); err == nil || !strings.Contains(err.Error(), "full production web stack") {
		t.Fatalf("fake web-stack claim error = %v", err)
	}

	fakeParity := cloneIOReactorCoverage(report)
	fakeParity.CrossPlatformParityClaimed = true
	if err := ValidateIOReactorCoverage(fakeParity); err == nil || !strings.Contains(err.Error(), "cross-platform") {
		t.Fatalf("fake cross-platform parity claim error = %v", err)
	}

	fakeIOUring := cloneIOReactorCoverage(report)
	fakeIOUring.IOUringClaimed = true
	if err := ValidateIOReactorCoverage(fakeIOUring); err == nil || !strings.Contains(err.Error(), "io_uring") {
		t.Fatalf("fake io_uring claim error = %v", err)
	}

	fakeRuntimeChange := cloneIOReactorCoverage(report)
	fakeRuntimeChange.RuntimeBehaviorChanged = true
	if err := ValidateIOReactorCoverage(fakeRuntimeChange); err == nil || !strings.Contains(err.Error(), "runtime behavior") {
		t.Fatalf("fake runtime behavior claim error = %v", err)
	}

	fakeKqueue := cloneIOReactorCoverage(report)
	for i := range fakeKqueue.Rows {
		if fakeKqueue.Rows[i].ID == IOReactorKqueueBoundary {
			fakeKqueue.Rows[i].Status = IOReactorImplementedNarrow
		}
	}
	if err := ValidateIOReactorCoverage(fakeKqueue); err == nil || !strings.Contains(err.Error(), "kqueue") {
		t.Fatalf("fake kqueue promotion error = %v", err)
	}

	missingStress := cloneIOReactorCoverage(report)
	for i := range missingStress.Rows {
		if missingStress.Rows[i].ID == IOReactorStressEvidence {
			missingStress.Rows[i].RequiredFacts = []string{"stress evidence"}
		}
	}
	if err := ValidateIOReactorCoverage(missingStress); err == nil || !strings.Contains(err.Error(), "stress") {
		t.Fatalf("missing stress evidence error = %v", err)
	}

	missingHTTP := cloneIOReactorCoverage(report)
	for i := range missingHTTP.Rows {
		if missingHTTP.Rows[i].ID == IOReactorHTTPSmoke {
			missingHTTP.Rows[i].RequiredFacts = []string{"HTTP smoke"}
		}
	}
	if err := ValidateIOReactorCoverage(missingHTTP); err == nil || !strings.Contains(err.Error(), "HTTP") {
		t.Fatalf("missing HTTP smoke evidence error = %v", err)
	}

	noNonClaim := cloneIOReactorCoverage(report)
	noNonClaim.NonClaims = nil
	if err := ValidateIOReactorCoverage(noNonClaim); err == nil || !strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func requireIOReactorFacts(t *testing.T, row IOReactorEvidenceRow, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !hasIOReactorText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func hasIOReactorText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneIOReactorCoverage(report IOReactorCoverageReport) IOReactorCoverageReport {
	clone := report
	clone.Rows = append([]IOReactorEvidenceRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
