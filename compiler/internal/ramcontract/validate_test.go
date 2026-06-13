package ramcontract

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
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

func TestRAMContractRejectsTrustedPlacementWithoutNoEscapeValidation(t *testing.T) {
	tests := []struct {
		name       string
		escape     EscapeStatus
		validation ValidationStatus
	}{
		{name: "escaped_stack", escape: EscapeReturn, validation: ValidationValidated},
		{name: "conservative_stack", escape: EscapeNoEscape, validation: ValidationConservative},
		{name: "unknown_region", escape: EscapeUnknown, validation: ValidationUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validReportForTest()
			report.Rows[0].EscapeStatus = test.escape
			report.Rows[0].ValidationStatus = test.validation
			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "trusted placement") {
				t.Fatalf("ValidateReport error = %v, want trusted placement no-escape proof rejection", err)
			}
		})
	}
}

func TestRAMContractAllowsEliminatedCopyWithoutTrustedProof(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].Intent = IntentCopyEliminated
	report.Rows[0].RequestedBytes = 0
	report.Rows[0].Placement = PlacementEliminated
	report.Rows[0].ProofIDs = nil
	report.Rows[0].CopyReason = "copy elided by lowering"
	report.Rows[0].ContractGrade = GradeM0
	report.Rows[0].ValidationStatus = ValidationConservative
	report.Proofs = nil
	report.Summary = SummarizeRows(report.Rows)
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport rejected eliminated copy without trusted proof: %v", err)
	}
}

func TestRAMContractRejectsRegionIslandPlacementWithoutScopedProof(t *testing.T) {
	tests := []struct {
		name      string
		placement Placement
		lifetime  string
	}{
		{name: "region", placement: PlacementRegion, lifetime: "region:main:temp"},
		{name: "island", placement: PlacementIsland, lifetime: "island:isl:scope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validReportForTest()
			report.Rows[0].Placement = test.placement
			report.Rows[0].Intent = IntentRegionAlloc
			report.Rows[0].Lifetime = test.lifetime
			report.Rows[0].FreePoint = test.lifetime + ":end"
			report.Rows[0].ContractGrade = GradeForPlacement(test.placement)
			report.Summary = SummarizeRows(report.Rows)
			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "scoped proof") {
				t.Fatalf("ValidateReport error = %v, want scoped proof rejection", err)
			}
		})
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

func TestValidateRAMContractReportRejectsForbiddenNonclaimText(t *testing.T) {
	report := validReportForTest()
	report.NonClaims = []string{"Memory 100%"}
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "forbidden broad claim") {
		t.Fatalf("ValidateReport error = %v, want forbidden broad claim rejection", err)
	}
}

func TestValidateRAMContractReportAllowsNegatedNonclaimText(t *testing.T) {
	report := validReportForTest()
	report.NonClaims = []string{
		"no Memory 100% claim",
		"not a full formal proof",
		"does not claim zero heap for all programs",
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport rejected negated nonclaims: %v", err)
	}
}

func TestValidatePipelineCoverageRejectsMissingBuildFileEntrypoint(t *testing.T) {
	report := validPipelineCoverageForTest()
	report.Entries = report.Entries[1:]
	err := ValidatePipelineCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "BuildFileWithStatsOpt") {
		t.Fatalf("ValidatePipelineCoverage error = %v, want missing BuildFileWithStatsOpt rejection", err)
	}
}

func TestRAMContractFromAllocPlanTracksRowsAndBlockers(t *testing.T) {
	plan := &allocplan.Plan{
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				SiteID:                "site:main:heap",
				ValueID:               "heap",
				Source:                "fixtures/main.tetra:7:13",
				Escape:                allocplan.EscapeUnknown,
				Storage:               allocplan.StorageStack,
				PlannedStorage:        allocplan.StorageStack,
				ActualLoweringStorage: allocplan.StorageHeap,
				Reason:                "backend conservative heap fallback",
				ValidationStatus:      "validated_conservative",
			}},
		}},
	}
	report := BuildReportFromAllocPlan(plan, "linux-x64", "e2c19b8ee276158f8eb2c54cf61e11bd84952893", "test")
	if len(report.Rows) != 1 {
		t.Fatalf("report rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.SiteID != "site:main:heap" || row.Intent != IntentHeapFallback || row.Placement != PlacementHeapUnbounded {
		t.Fatalf("row = %#v, want heap fallback row from alloc plan", row)
	}
	if len(row.Blockers) == 0 {
		t.Fatalf("row blockers = %#v, want heap blocker explanation", row.Blockers)
	}
	if report.Summary.HeapRows != 1 || report.Summary.UnboundedRows != 1 || report.Summary.ArtifactGrade != GradeM5 {
		t.Fatalf("summary = %#v, want heap/unbounded M5 summary", report.Summary)
	}
}

func TestRAMContractHeapBlockerReportCarriesActionableSourceMetadata(t *testing.T) {
	plan := &allocplan.Plan{
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				SiteID:                "site:main:heap",
				ValueID:               "heap",
				Source:                "fixtures/main.tetra:7:13",
				Escape:                allocplan.EscapeUnknown,
				Storage:               allocplan.StorageStack,
				PlannedStorage:        allocplan.StorageStack,
				ActualLoweringStorage: allocplan.StorageHeap,
				Reason:                "backend conservative heap fallback",
				ValidationStatus:      "validated_conservative",
			}},
		}},
	}
	report := BuildReportFromAllocPlan(plan, "linux-x64", "e2c19b8ee276158f8eb2c54cf61e11bd84952893", "test")
	blockers := BuildHeapBlockerReport(report)
	if err := ValidateBlockerReport(blockers, "heap"); err != nil {
		t.Fatalf("ValidateBlockerReport: %v", err)
	}
	if len(blockers.Rows) != 1 {
		t.Fatalf("heap blocker rows = %d, want 1", len(blockers.Rows))
	}
	row := blockers.Rows[0]
	if row.File != "fixtures/main.tetra" || row.Line != 7 || row.SourceLocationStatus != "available" {
		t.Fatalf("source metadata = file %q line %d status %q, want fixtures/main.tetra:7 available", row.File, row.Line, row.SourceLocationStatus)
	}
	if row.Symbol != "main" || row.Severity == "" || row.Reason == "" || row.SuggestedFix == "" || row.EvidenceID == "" {
		t.Fatalf("row missing actionable metadata: %#v", row)
	}
	if row.SafeToOptimize {
		t.Fatalf("heap blocker safe_to_optimize = true, want conservative false")
	}
}

func TestRAMContractCopyBlockerReportClassifiesCopySafety(t *testing.T) {
	report := validReportForTest()
	report.Rows = []Row{{
		SiteID:           "site:main:copy",
		ValueID:          "copy",
		Function:         "main",
		SourceSpan:       "fixtures/copy.tetra:9:17",
		Intent:           IntentCopyHeapBounded,
		RequestedBytes:   64,
		Bounded:          true,
		Owner:            "function:main",
		Lifetime:         "function:main",
		EscapeStatus:     EscapeNoEscape,
		Placement:        PlacementHeapBounded,
		ProofIDs:         nil,
		Blockers:         []string{"backend_conservative_heap_fallback"},
		CopyReason:       "copy_requires_bounded_heap_fallback",
		ContractGrade:    GradeM4,
		ValidationStatus: ValidationConservative,
		SourceFactID:     "fact:ram:site:main:copy",
	}}
	report.Summary = SummarizeRows(report.Rows)
	report.Functions = SummarizeFunctions(report.Rows)
	blockers := BuildCopyBlockerReport(report)
	if err := ValidateBlockerReport(blockers, "copy"); err != nil {
		t.Fatalf("ValidateBlockerReport: %v", err)
	}
	if len(blockers.Rows) != 1 {
		t.Fatalf("copy blocker rows = %d, want 1", len(blockers.Rows))
	}
	row := blockers.Rows[0]
	if row.CopyKind != "HOT_PATH_COPY" || row.SourceValue != "copy" || row.DestinationValue != "heap_bounded" || row.BytesEstimate != 64 {
		t.Fatalf("copy classification = %#v, want HOT_PATH_COPY copy -> heap_bounded bytes 64", row)
	}
	if row.SafetyReason == "" || row.SuggestedFix == "" || row.SafeToOptimize {
		t.Fatalf("copy safety metadata = safety_reason %q suggested_fix %q safe %v, want conservative action guidance", row.SafetyReason, row.SuggestedFix, row.SafeToOptimize)
	}
}

func TestRAMContractFromAllocPlanDoesNotProveUnknownCallTrustedLowering(t *testing.T) {
	plan := &allocplan.Plan{
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				SiteID:                "site:main:unknown_call",
				ValueID:               "unknown_call",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthStatus:          allocplan.LengthStatusNormal,
				ByteSize:              16,
				BytesRequested:        16,
				BytesReserved:         16,
				Escape:                allocplan.EscapeCallUnknown,
				Storage:               allocplan.StorageHeap,
				PlannedStorage:        allocplan.StorageHeap,
				ActualLoweringStorage: allocplan.StorageStack,
				ValidationStatus:      "validated_heap_fallback",
				Reason:                "unknown call may retain allocation",
			}},
		}},
	}
	report := BuildReportFromAllocPlan(plan, "linux-x64", "e2c19b8ee276158f8eb2c54cf61e11bd84952893", "test")
	if len(report.Rows) != 1 {
		t.Fatalf("report rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.EscapeStatus != EscapeCall || row.Placement != PlacementStack {
		t.Fatalf("row escape/placement = %q/%q, want escapes_call/stack test fixture", row.EscapeStatus, row.Placement)
	}
	if row.ValidationStatus == ValidationValidated {
		t.Fatalf("unknown-call trusted lowering validation_status = %q, want non-validated", row.ValidationStatus)
	}
	if len(row.ProofIDs) != 0 || len(report.Proofs) != 0 {
		t.Fatalf("unknown-call trusted lowering proof_ids/proofs = %v/%v, want no proven trusted proof", row.ProofIDs, report.Proofs)
	}
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "trusted placement") {
		t.Fatalf("ValidateReport error = %v, want trusted placement rejection", err)
	}
}

func TestRAMContractFromAllocPlanEmitsScopedRegionIslandProofs(t *testing.T) {
	plan := &allocplan.Plan{
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{
				{
					SiteID:                "site:main:region",
					ValueID:               "region",
					Builtin:               "core.slice_copy_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              16,
					BytesRequested:        16,
					BytesReserved:         16,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageFunctionTempRegion,
					PlannedStorage:        allocplan.StorageFunctionTempRegion,
					ActualLoweringStorage: allocplan.StorageFunctionTempRegion,
					ValidationStatus:      "validated_function_temp_region_scope",
					Lifetime:              "function:main",
					RegionID:              "region:main:temp",
					Reason:                "function-temp region scope proof",
				},
				{
					SiteID:                "site:main:island",
					ValueID:               "island",
					Builtin:               "core.island_make_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              16,
					BytesRequested:        16,
					BytesReserved:         16,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageExplicitIsland,
					PlannedStorage:        allocplan.StorageExplicitIsland,
					ActualLoweringStorage: allocplan.StorageExplicitIsland,
					ValidationStatus:      "validated_explicit_island_scope",
					Lifetime:              "island:isl:scope",
					RegionID:              "island:isl",
					Reason:                "explicit island scope proof",
				},
			},
		}},
	}
	report := BuildReportFromAllocPlan(plan, "linux-x64", "e2c19b8ee276158f8eb2c54cf61e11bd84952893", "test")
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport rejected generated scoped proof report: %v", err)
	}
	proofs := map[string]ProofSummary{}
	for _, proof := range report.Proofs {
		proofs[proof.ProofID] = proof
	}
	for _, row := range report.Rows {
		if row.Placement != PlacementRegion && row.Placement != PlacementIsland {
			continue
		}
		if len(row.ProofIDs) != 1 {
			t.Fatalf("row %#v proof_ids = %v, want one scoped proof", row.SiteID, row.ProofIDs)
		}
		proof := proofs[row.ProofIDs[0]]
		wantKind := "region_lifetime_placement"
		if row.Placement == PlacementIsland {
			wantKind = "island_lifetime_placement"
		}
		if proof.Kind != wantKind {
			t.Fatalf("row %s proof kind = %q, want %q", row.SiteID, proof.Kind, wantKind)
		}
	}
}

func TestRAMContractRejectsMissingBlockerExplanation(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].Placement = PlacementHeapUnbounded
	report.Rows[0].Intent = IntentHeapFallback
	report.Rows[0].ContractGrade = GradeM5
	report.Rows[0].Bounded = false
	report.Rows[0].ProofIDs = nil
	report.Rows[0].Blockers = nil
	report.Summary = SummarizeRows(report.Rows)
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "heap") || !strings.Contains(err.Error(), "blocker") {
		t.Fatalf("ValidateReport error = %v, want missing blocker explanation rejection", err)
	}
}

func TestRAMContractEnforcementFailsForHeap(t *testing.T) {
	report := validReportForTest()
	report.Rows[0].Placement = PlacementHeapUnbounded
	report.Rows[0].Intent = IntentHeapFallback
	report.Rows[0].ContractGrade = GradeM5
	report.Rows[0].Bounded = false
	report.Rows[0].ProofIDs = nil
	report.Rows[0].Blockers = []string{"unknown_size"}
	report.Rows[0].ValidationStatus = ValidationConservative
	report.Summary = SummarizeRows(report.Rows)
	err := Enforce(report, EnforcementOptions{FailIfHeap: true})
	if err == nil || !strings.Contains(err.Error(), "RAM_CONTRACT_HEAP") {
		t.Fatalf("Enforce error = %v, want fail-if-heap rejection", err)
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

func validPipelineCoverageForTest() PipelineCoverageReport {
	return PipelineCoverageReport{
		SchemaVersion: PipelineCoverageSchemaV1,
		GitHead:       "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Entries: []PipelineEntry{
			{Entrypoint: "BuildFileWithStatsOpt", ArtifactPath: "app", Status: "validated_by_pipeline", Validators: []string{"ramcontract.ValidateReport"}},
			{Entrypoint: "buildObjectFileWithStatsOpt", Status: "formal_exemption_with_reason", Exemption: "not exercised by this linux-x64 RAM release fixture; object builds must carry their own RAM coverage evidence"},
			{Entrypoint: "buildLibraryObjectWithStatsOpt", Status: "formal_exemption_with_reason", Exemption: "not exercised by this linux-x64 RAM release fixture; library builds must carry their own RAM coverage evidence"},
			{Entrypoint: "InterfaceOnly", Status: "formal_exemption_with_reason", Exemption: "interface-only mode does not produce a RAM artifact in this release fixture"},
			{Entrypoint: "wasm32-wasi-build", Status: "formal_exemption_with_reason", Exemption: "wasm32-wasi RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
			{Entrypoint: "wasm32-web-build", Status: "formal_exemption_with_reason", Exemption: "wasm32-web RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
			{Entrypoint: "explain-report-path", Status: "formal_exemption_with_reason", Exemption: "explain report path is not artifact-producing in this release fixture"},
		},
		NonClaims: DefaultNonClaims(),
	}
}
