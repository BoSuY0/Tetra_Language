package memoryfacts_test

import (
	"strings"
	"testing"

	. "tetra_language/compiler/internal/memoryfacts"
)

func TestMemoryFactsValidationUsesSharedVocabulary(t *testing.T) {
	for _, stage := range SourceStages() {
		graph := NewGraph("program")
		if _, err := graph.AddFact(Fact{
			ID:              FactID("fact:stage:" + strings.ReplaceAll(stage, ":", "_")),
			FunctionID:      "main",
			SiteID:          "site:" + stage,
			SourceStage:     SourceStage(stage),
			Claim:           ClaimBorrowedImm,
			ProvenanceClass: ProvenanceSafeBorrowed,
			UnsafeClass:     UnsafeSafe,
			BorrowState:     BorrowImmutable,
		}); err != nil {
			t.Fatalf("memoryfacts rejects registered source stage %q: %v", stage, err)
		}
	}

	zeroCost := validMemoryReport().Rows[0]
	dynamicCost := vocabularyReportRow(
		"fact:cost:dynamic",
		ClaimRawBoundsRuntimeCheckNormalBuild,
		ClaimValidated,
		ProvenanceUnsafeChecked,
		UnsafeChecked,
		CostDynamicCheckRequired,
	)
	dynamicCost.ParentFactID = "fact:cost:dynamic:parent"
	dynamicCost.NormalBuildCheck = true
	dynamicCost.ValidatorName = "raw_bounds_width_validator"
	dynamicCost.ValidatorStatus = ValidatorPass
	instrumentationCost := vocabularyReportRow(
		"fact:cost:instrumentation",
		ClaimNormalBuildBoundsCheckGuard,
		ClaimEvidenceOnly,
		ProvenanceSafeKnown,
		UnsafeSafe,
		CostInstrumentationOnly,
	)
	instrumentationCost.NormalBuildCheck = true
	rejectedCost := vocabularyReportRow(
		"fact:cost:rejected",
		ClaimUnsafeUnknownRejectedSafeFacts,
		ClaimRejected,
		ProvenanceUnsafeUnknown,
		UnsafeUnknown,
		CostUnsupportedRejected,
	)
	rejectedCost.ParentFactID = "fact:cost:rejected:parent"
	rejectedCost.ValidatorName = "unsafe_unknown_fact_validator"
	rejectedCost.ValidatorStatus = ValidatorFail
	conservativeCost := vocabularyReportRow(
		"fact:cost:conservative",
		ClaimExternalUnknown,
		ClaimConservative,
		ProvenanceUnsafeUnknown,
		UnsafeUnknown,
		CostConservativeFallback,
	)
	conservativeCost.ValidatorStatus = ValidatorNotApplicable
	costRows := map[string]ReportRow{
		string(CostZeroCostProven):       zeroCost,
		string(CostDynamicCheckRequired): dynamicCost,
		string(CostInstrumentationOnly):  instrumentationCost,
		string(CostUnsupportedRejected):  rejectedCost,
		string(CostConservativeFallback): conservativeCost,
	}
	for _, class := range CostClasses() {
		row, ok := costRows[class]
		if !ok {
			t.Fatalf("missing public validation fixture for registered cost class %q", class)
		}
		if err := ValidateReport(
			Report{SchemaVersion: ReportSchemaV1, Rows: []ReportRow{row}},
		); err != nil {
			t.Fatalf("memoryfacts rejects registered cost class %q: %v", class, err)
		}
	}

	for _, claim := range ReportClaims() {
		row := vocabularyReportRow(
			"fact:claim:"+claim,
			claim,
			ClaimEvidenceOnly,
			ProvenanceSafeKnown,
			UnsafeSafe,
			CostInstrumentationOnly,
		)
		err := ValidateReport(Report{SchemaVersion: ReportSchemaV1, Rows: []ReportRow{row}})
		if err != nil && strings.Contains(err.Error(), "unknown memory report claim") {
			t.Fatalf("memoryfacts rejects registered report claim %q as unknown: %v", claim, err)
		}
	}

	for _, claim := range ParentRequiredClaims() {
		row := vocabularyReportRow(
			"fact:parent-required:"+claim,
			claim,
			ClaimConservative,
			ProvenanceUnsafeUnknown,
			UnsafeUnknown,
			CostConservativeFallback,
		)
		err := ValidateReport(Report{SchemaVersion: ReportSchemaV1, Rows: []ReportRow{row}})
		if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
			t.Fatalf(
				"memoryfacts parent requirement for %q = %v, want parent_fact_id rejection",
				claim,
				err,
			)
		}
	}

	report := Report{SchemaVersion: ReportSchemaV1, Rows: []ReportRow{
		vocabularyReportRow(
			"fact:unsafe-checked:noalias",
			ClaimNoAlias,
			ClaimEvidenceOnly,
			ProvenanceUnsafeChecked,
			UnsafeChecked,
			CostInstrumentationOnly,
		),
	}}
	if err := ValidateReport(report); err == nil ||
		!strings.Contains(err.Error(), "unsafe_checked") {
		t.Fatalf(
			"ValidateReport unsafe_checked no_alias error = %v, want unsafe_checked rejection",
			err,
		)
	}

	rawBounds := vocabularyReportRow(
		"fact:unsafe-checked:raw-bounds",
		ClaimRawBoundsRuntimeCheckNormalBuild,
		ClaimValidated,
		ProvenanceUnsafeChecked,
		UnsafeChecked,
		CostDynamicCheckRequired,
	)
	rawBounds.ParentFactID = "fact:unsafe-checked:raw-bounds:parent"
	rawBounds.NormalBuildCheck = true
	rawBounds.ValidatorName = "raw_bounds_width_validator"
	rawBounds.ValidatorStatus = ValidatorPass
	if err := ValidateReport(
		Report{SchemaVersion: ReportSchemaV1, Rows: []ReportRow{rawBounds}},
	); err != nil {
		t.Fatalf(
			"memoryfacts should allow checked raw bounds runtime evidence through shared vocabulary: %v",
			err,
		)
	}
}

func vocabularyReportRow(
	id string,
	claim string,
	level ClaimLevel,
	provenance ProvenanceClass,
	unsafeClass UnsafeClass,
	cost CostClass,
) ReportRow {
	status := ValidatorNotRun
	if level == ClaimConservative {
		status = ValidatorNotApplicable
	}
	if level == ClaimRejected {
		status = ValidatorFail
	}
	if level == ClaimValidated {
		status = ValidatorPass
	}
	return ReportRow{
		ProgramID:       "program",
		FunctionID:      "main",
		SiteID:          "site",
		SourceFactID:    FactID(id),
		SourceStage:     StageValidation,
		Claim:           claim,
		ClaimLevel:      level,
		ProvenanceClass: provenance,
		UnsafeClass:     unsafeClass,
		ValidatorStatus: status,
		CostClass:       cost,
		Reason:          "vocabulary fixture",
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
	if got := err.Error(); !strings.Contains(got, "unknown memory report claim") ||
		!strings.Contains(got, "new_zero_cost_magic_claim") {
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
			Claim:           ClaimIslandProofVerified,
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
		t.Fatalf(
			"expected memoryfacts report validation to reject island proof claim without island verifier",
		)
	}
	if got := err.Error(); !strings.Contains(got, ClaimIslandProofVerified) ||
		!strings.Contains(got, "validate-island-proof") {
		t.Fatalf("ValidateReport error = %v, want validate-island-proof rejection", err)
	}
}
