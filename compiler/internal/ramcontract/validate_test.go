package ramcontract

import (
	"strings"
	"testing"
)

func TestValidateRAMContractReportRejectsUnknownGrade(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].ContractGrade = "MX"
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown contract_grade") {
		t.Fatalf("ValidateReport error = %v, want unknown grade", err)
	}
}

func TestValidateRAMContractReportRejectsMissingSiteID(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].SiteID = ""
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "site_id") {
		t.Fatalf("ValidateReport error = %v, want site_id rejection", err)
	}
}

func TestValidateRAMContractReportRejectsHeapFallbackWithoutBlocker(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].Placement = PlacementHeapBounded
	report.Rows[0].Intent = IntentHeapFallback
	report.Rows[0].ContractGrade = GradeM4
	report.Rows[0].ProofIDs = nil
	report.Rows[0].Blockers = nil
	report.Summary = SummarizeRows(report.Rows)
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "heap") || !strings.Contains(err.Error(), "blocker") {
		t.Fatalf("ValidateReport error = %v, want heap blocker rejection", err)
	}
}

func TestRAMContractRejectsTrustedPlacementWithoutProof(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].ProofIDs = nil
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "proof_ids") {
		t.Fatalf("ValidateReport error = %v, want proof_ids rejection", err)
	}
}

func TestValidateMemoryGradeRejectsContradictorySummary(t *testing.T) {
	report := validReportForTest()
	report.Summary.ArtifactGrade = GradeM0
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "artifact_grade") {
		t.Fatalf("ValidateReport error = %v, want contradictory grade rejection", err)
	}
}

func validReportForTest() Report {
	proof := ProofSummary{
		ProofID:    "proof:ram:main:alloc0",
		Kind:       "allocation_placement",
		Subject:    "main/alloc0",
		StableHash: "sha256:test",
		Status:     "proven",
	}
	rows := []Row{{
		SiteID:           "site:main:alloc0",
		ValueID:          "alloc0",
		Function:         "main",
		Intent:           IntentAllocation,
		RequestedBytes:   16,
		Bounded:          true,
		Owner:            "function:main",
		Lifetime:         "function:main",
		EscapeStatus:     EscapeNoEscape,
		Placement:        PlacementStack,
		ProofIDs:         []string{proof.ProofID},
		ContractGrade:    GradeM1,
		ValidationStatus: ValidationValidated,
	}}
	return Report{
		SchemaVersion: ReportSchemaV1,
		GitHead:       "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows:          rows,
		Proofs:        []ProofSummary{proof},
		Summary:       SummarizeRows(rows),
		NonClaims:     DefaultNonClaims(),
	}
}
