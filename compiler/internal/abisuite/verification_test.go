package abisuite

import (
	"strings"
	"testing"
)

func TestBuildP21VerificationReportCoversTargetsTasksAndNonClaims(t *testing.T) {
	report := BuildP21VerificationReport()
	if report.Schema != VerificationSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.Schema, VerificationSchemaV1)
	}
	if report.Scope != VerificationScopeP211 {
		t.Fatalf("scope = %q, want %q", report.Scope, VerificationScopeP211)
	}
	if err := ValidateP21VerificationReport(report); err != nil {
		t.Fatalf("ValidateP21VerificationReport: %v", err)
	}
	if len(report.Targets) != len(P21VerificationTargets()) {
		t.Fatalf("targets len = %d, want %d", len(report.Targets), len(P21VerificationTargets()))
	}
	if len(report.Tasks) != len(P21VerificationTaskIDs()) {
		t.Fatalf("tasks len = %d, want %d", len(report.Tasks), len(P21VerificationTaskIDs()))
	}
	for _, nonClaim := range P21VerificationNonClaims() {
		if !stringSliceHas(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q", nonClaim)
		}
	}
}

func TestValidateP21VerificationReportRejectsFakeClaims(t *testing.T) {
	report := BuildP21VerificationReport()
	report.Claims = append(report.Claims, "full runtime execution verified")
	err := ValidateP21VerificationReport(report)
	if err == nil || !strings.Contains(err.Error(), "runtime execution") {
		t.Fatalf("ValidateP21VerificationReport fake claim err = %v", err)
	}
}
