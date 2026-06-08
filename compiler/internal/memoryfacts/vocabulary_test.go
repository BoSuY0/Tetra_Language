package memoryfacts

import (
	"strings"
	"testing"

	"tetra_language/compiler/memoryvocab"
)

func TestMemoryFactsValidationUsesSharedVocabulary(t *testing.T) {
	for _, stage := range memoryvocab.SourceStages() {
		if !knownSourceStage(SourceStage(stage)) {
			t.Fatalf("memoryfacts rejects registered source stage %q", stage)
		}
	}
	for _, class := range memoryvocab.CostClasses() {
		if !knownCostClass(CostClass(class)) {
			t.Fatalf("memoryfacts rejects registered cost class %q", class)
		}
	}
	for _, claim := range memoryvocab.ReportClaims() {
		if !knownReportClaim(claim) {
			t.Fatalf("memoryfacts rejects registered report claim %q", claim)
		}
	}
	for _, claim := range memoryvocab.ParentRequiredClaims() {
		if !claimRequiresParentFactID(claim) {
			t.Fatalf("memoryfacts does not require parent_fact_id for registered derived claim %q", claim)
		}
	}
	if !unsafeCheckedDisallowedClaim(ProvenanceUnsafeChecked, UnsafeChecked, memoryvocab.ClaimNoAlias) {
		t.Fatalf("memoryfacts should reject unsafe_checked no_alias through shared vocabulary")
	}
	if unsafeCheckedDisallowedClaim(ProvenanceUnsafeChecked, UnsafeChecked, memoryvocab.ClaimRawBoundsRuntimeCheckNormalBuild) {
		t.Fatalf("memoryfacts should allow checked raw bounds runtime evidence through shared vocabulary")
	}
}

func TestMemoryReportProjectionRejectsUnregisteredClaimVocabulary(t *testing.T) {
	graph := NewGraph("program")
	if _, err := graph.AddFact(Fact{
		ID:              "fact:unregistered-claim",
		FunctionID:      "main",
		SiteID:          "site",
		SourceStage:     StageValidation,
		Claim:           "new_zero_cost_magic_claim",
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
	}); err != nil {
		t.Fatalf("AddFact unregistered claim fixture: %v", err)
	}

	err := ValidateReportProjection(graph, BuildReportFromGraph(graph))
	if err == nil {
		t.Fatalf("expected projection validation to reject unregistered claim")
	}
	if got := err.Error(); !strings.Contains(got, "unknown memory report claim") || !strings.Contains(got, "new_zero_cost_magic_claim") {
		t.Fatalf("ValidateReportProjection error = %v, want unknown claim rejection", err)
	}
}

func TestMemoryFactsRejectIslandProofClaimWithoutIslandVerifier(t *testing.T) {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{{
			ProgramID:       "program",
			FunctionID:      "main",
			SiteID:          "site",
			SourceFactID:    "fact:island-proof",
			SourceStage:     StageValidation,
			Claim:           memoryvocab.ClaimIslandProofVerified,
			ClaimLevel:      ClaimValidated,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			ValidatorName:   "memory_report_validator",
			ValidatorStatus: ValidatorPass,
			CostClass:       CostInstrumentationOnly,
		}},
	}

	err := ValidateReport(report)
	if err == nil {
		t.Fatalf("expected memoryfacts report validation to reject island proof claim without island verifier")
	}
	if got := err.Error(); !strings.Contains(got, memoryvocab.ClaimIslandProofVerified) || !strings.Contains(got, "validate-island-proof") {
		t.Fatalf("ValidateReport error = %v, want validate-island-proof rejection", err)
	}
}
