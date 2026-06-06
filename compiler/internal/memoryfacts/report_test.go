package memoryfacts

import (
	"strings"
	"testing"
)

func TestValidateMemoryReportRejectsValidatedClaimWithoutFact(t *testing.T) {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{{
			ProgramID:       "program",
			FunctionID:      "main",
			SiteID:          "site",
			Claim:           "borrowed view",
			ClaimLevel:      ClaimValidated,
			ProvenanceClass: ProvenanceSafeBorrowed,
			UnsafeClass:     UnsafeSafe,
			ValidatorName:   "memoryfacts",
			ValidatorStatus: ValidatorPass,
		}},
	}
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("ValidateReport error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].SourceFactID = "unsafe-root"
	report.Rows[0].ParentFactID = "unsafe-root"
	report.Rows[0].ProvenanceClass = ProvenanceSafeKnown
	report.Rows[0].UnsafeClass = UnsafeUnknown
	report.Rows[0].Claim = "unsafe_unknown became safe_known"

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "safe_known") {
		t.Fatalf("ValidateReport error = %v, want safe_known rejection", err)
	}
}

func TestValidateMemoryReportRejectsSafeBorrowedFromUnsafeUnknown(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "borrowed_imm"
	report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
	report.Rows[0].UnsafeClass = UnsafeUnknown

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "safe_borrowed") {
		t.Fatalf("ValidateReport error = %v, want safe_borrowed rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims(t *testing.T) {
	for _, tc := range []struct {
		name       string
		claim      string
		aliasState AliasState
	}{
		{name: "provenance known", claim: "provenance_known"},
		{name: "noalias", claim: "no_alias", aliasState: AliasMutableExclusive},
		{name: "bounds proof", claim: "index_in_range"},
		{name: "bounds check elimination", claim: "bounds_check_eliminated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := Report{
				SchemaVersion: ReportSchemaV1,
				Rows: []ReportRow{{
					ProgramID:       "program",
					FunctionID:      "main",
					SiteID:          "raw:1:1",
					SourceFactID:    FactID("fact:" + tc.name),
					SourceStage:     StagePLIR,
					Claim:           tc.claim,
					ClaimLevel:      ClaimConservative,
					ProvenanceClass: ProvenanceUnsafeUnknown,
					UnsafeClass:     UnsafeUnknown,
					AliasState:      tc.aliasState,
					ValidatorStatus: ValidatorNotApplicable,
					Reason:          "unsafe unknown cannot authorize optimization evidence",
				}},
			}
			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
				t.Fatalf("ValidateReport error = %v, want unsafe_unknown rejection", err)
			}
		})
	}
}

func TestMemoryReportRowsRequireCostClass(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].CostClass = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "cost_class") {
		t.Fatalf("ValidateReport error = %v, want cost_class rejection", err)
	}
}

func TestMemoryReportRejectsUnknownCostClass(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].CostClass = CostClass("mystery_cost")

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown cost_class") {
		t.Fatalf("ValidateReport error = %v, want unknown cost_class rejection", err)
	}
}

func TestMemoryReportRejectsDynamicOptimizationClaimWithoutNormalBuildCheck(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "bounds_check_eliminated"
	report.Rows[0].CostClass = CostDynamicCheckRequired
	report.Rows[0].NormalBuildCheck = false

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "dynamic_check_required") || !strings.Contains(err.Error(), "normal_build_check") {
		t.Fatalf("ValidateReport error = %v, want dynamic check rejection", err)
	}
}

func TestMemoryReportRejectsUnsafeUnknownZeroCostTrustedClaim(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].ProvenanceClass = ProvenanceUnsafeUnknown
	report.Rows[0].UnsafeClass = UnsafeUnknown
	report.Rows[0].Claim = "trusted_storage"
	report.Rows[0].CostClass = CostZeroCostProven

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") || !strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("ValidateReport error = %v, want unsafe zero-cost rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaims(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "provenance_known"
	report.Rows[0].ClaimLevel = ClaimEvidenceOnly
	report.Rows[0].ProvenanceClass = ProvenanceUnsafeVerifiedRoot
	report.Rows[0].UnsafeClass = UnsafeVerifiedRoot
	report.Rows[0].ValidatorStatus = ValidatorNotRun
	report.Rows[0].ValidatorName = ""
	report.Rows[0].LoweredArtifactID = ""
	report.Rows[0].PlannedStorage = ""
	report.Rows[0].ActualLoweringStorage = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsafe_verified_root") {
		t.Fatalf("ValidateReport error = %v, want unsafe_verified_root generic-claim rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedUnsafeUnknownTrustedStorage(t *testing.T) {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{{
			ProgramID:             "program",
			FunctionID:            "main",
			SiteID:                "raw:1:1",
			SourceFactID:          "fact:raw:storage",
			SourceStage:           StageAllocPlan,
			Claim:                 "storage_lowering",
			ClaimLevel:            ClaimValidated,
			ProvenanceClass:       ProvenanceUnsafeUnknown,
			UnsafeClass:           UnsafeUnknown,
			PlannedStorage:        StorageStack,
			ActualLoweringStorage: StorageStack,
			LoweredArtifactID:     "ir:main:unsafe",
			ValidatorName:         "allocation_lowering_validator",
			ValidatorStatus:       ValidatorPass,
			Reason:                "fixture should be rejected",
		}},
	}
	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("ValidateReport error = %v, want unsafe_unknown storage rejection", err)
	}
}

func TestValidateMemoryReportRejectsStorageClaimWithoutArtifact(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "stack lowering claim"
	report.Rows[0].PlannedStorage = StorageStack
	report.Rows[0].ActualLoweringStorage = StorageStack
	report.Rows[0].LoweredArtifactID = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "lowered_artifact_id") {
		t.Fatalf("ValidateReport error = %v, want lowered_artifact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsPartialStorageFields(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].ActualLoweringStorage = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "planned_storage and actual_lowering_storage") {
		t.Fatalf("ValidateReport error = %v, want paired storage rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnknownStorageClass(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].PlannedStorage = StorageClass("Mystery")

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown planned_storage") {
		t.Fatalf("ValidateReport error = %v, want unknown storage rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedTrustedStorageHeapFallback(t *testing.T) {
	for _, planned := range []StorageClass{StorageStack, StorageRegion, StorageFunctionTempRegion} {
		t.Run(string(planned), func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = "storage_lowering"
			report.Rows[0].PlannedStorage = planned
			report.Rows[0].ActualLoweringStorage = StorageHeap
			report.Rows[0].LoweredArtifactID = "ir:main:xs:Heap"

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "validated "+string(planned)+" claim cannot lower as Heap") {
				t.Fatalf("ValidateReport error = %v, want trusted storage heap fallback rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsHeapFallbackWithoutReason(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "storage_lowering"
	report.Rows[0].ClaimLevel = ClaimEvidenceOnly
	report.Rows[0].PlannedStorage = StorageStack
	report.Rows[0].ActualLoweringStorage = StorageHeap
	report.Rows[0].LoweredArtifactID = "ir:main:xs:Heap"
	report.Rows[0].CostClass = CostConservativeFallback
	report.Rows[0].ValidatorName = ""
	report.Rows[0].ValidatorStatus = ValidatorNotRun
	report.Rows[0].Reason = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("ValidateReport error = %v, want heap fallback reason rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedClaimWithoutValidatorName(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].ValidatorName = ""

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "validator_name") {
		t.Fatalf("ValidateReport error = %v, want validator_name rejection", err)
	}
}

func TestValidateMemoryReportRejectsWhitespaceRequiredFields(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].SourceFactID = "   "
	report.Rows[0].SiteID = "   "
	report.Rows[0].Claim = "   "

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") || !strings.Contains(err.Error(), "site_id") || !strings.Contains(err.Error(), "claim") {
		t.Fatalf("ValidateReport error = %v, want whitespace required-field rejection", err)
	}
}

func TestValidateMemoryReportRejectsDuplicateSourceFactID(t *testing.T) {
	report := validMemoryReport()
	report.Rows = append(report.Rows, report.Rows[0])
	report.Rows[1].SiteID = "site-2"

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "duplicate source_fact_id") {
		t.Fatalf("ValidateReport error = %v, want duplicate source_fact_id rejection", err)
	}
}

func TestValidateMemoryReportJSONRejectsTrailingData(t *testing.T) {
	raw := []byte(`{"schema_version":"tetra.memory-report.v1","rows":[{"site_id":"site","source_fact_id":"fact:safe","source_stage":"validation","claim":"validated allocation metadata","claim_level":"validated","provenance_class":"safe_known","unsafe_class":"safe","lowered_artifact_id":"ir:main:0","planned_storage":"Heap","actual_lowering_storage":"Heap","validator_name":"memory_report_schema_v1","validator_status":"pass"}]} {"schema_version":"tetra.memory-report.v1","rows":[]}`)
	err := ValidateReportJSON(raw)
	if err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("ValidateReportJSON error = %v, want trailing data rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnknownAliasState(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].AliasState = AliasState("mystery_alias")

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown alias_state") {
		t.Fatalf("ValidateReport error = %v, want unknown alias_state rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "no_alias"
	report.Rows[0].AliasState = AliasUnknownConservative

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "validated no_alias") {
		t.Fatalf("ValidateReport error = %v, want validated no_alias rejection", err)
	}
}

func TestValidateMemoryReportAcceptsConservativeUnknownRawPointer(t *testing.T) {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{{
			ProgramID:       "program",
			FunctionID:      "main",
			SiteID:          "raw:1:1",
			SourceSpan:      "raw.tetra:1:1",
			SourceFactID:    "fact:raw:unknown",
			SourceStage:     StagePLIR,
			Claim:           "checked_external_unknown",
			ClaimLevel:      ClaimConservative,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			CostClass:       CostConservativeFallback,
			ValidatorName:   "memory_report_schema_v1",
			ValidatorStatus: ValidatorNotApplicable,
			Reason:          "unknown raw pointer remains conservative",
		}},
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport conservative unknown raw pointer: %v", err)
	}
}

func TestValidateMemoryReportRejectsV7DerivedFFIRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"ffi_call_may_retain_borrow",
		"ffi_noalias_invalidated_by_external_call",
		"safe_wrapper_promotion_rejected_without_contract",
		"external_pointer_provenance_rejected",
	} {
		t.Run(claim, func(t *testing.T) {
			report := Report{
				SchemaVersion: ReportSchemaV1,
				Rows: []ReportRow{{
					ProgramID:       "program",
					FunctionID:      "ffi",
					SiteID:          "ffi:call:1",
					SourceFactID:    FactID("fact:" + claim),
					SourceStage:     StagePLIR,
					Claim:           claim,
					ClaimLevel:      ClaimConservative,
					ProvenanceClass: ProvenanceUnsafeUnknown,
					UnsafeClass:     UnsafeUnknown,
					AliasState:      AliasInvalidatedByCall,
					CostClass:       CostConservativeFallback,
					ValidatorName:   "memory_report_schema_v1",
					ValidatorStatus: ValidatorNotApplicable,
					Reason:          "v7 derived FFI row without parent should be rejected",
				}},
			}
			if strings.Contains(claim, "rejected") {
				report.Rows[0].ClaimLevel = ClaimRejected
				report.Rows[0].ValidatorStatus = ValidatorFail
			}
			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsBroadNoAliasClaim(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "broad_noalias"
	report.Rows[0].AliasState = AliasMutableExclusive

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "broad noalias") {
		t.Fatalf("ValidateReport error = %v, want broad noalias rejection", err)
	}
}

func TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "broad_noalias"
	report.Rows[0].AliasState = AliasMutableExclusive
	report.Rows[0].Reason = "protocol/interface dispatch cannot produce broad noalias"

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "broad noalias") {
		t.Fatalf("ValidateReport error = %v, want broad noalias rejection", err)
	}
}

func TestValidateMemoryReportAcceptsCallbackInoutConservativeProjection(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "callback_inout_conservative"
	report.Rows[0].ParentFactID = "fact:inout"
	report.Rows[0].ProvenanceClass = ProvenanceUnsafeUnknown
	report.Rows[0].UnsafeClass = UnsafeUnknown
	report.Rows[0].AliasState = AliasInvalidatedByCall
	report.Rows[0].CostClass = CostConservativeFallback
	report.Rows[0].ClaimLevel = ClaimConservative
	report.Rows[0].ValidatorName = "callback_alias_conservative_validator"
	report.Rows[0].ValidatorStatus = ValidatorNotApplicable
	report.Rows[0].LoweredArtifactID = ""
	report.Rows[0].PlannedStorage = ""
	report.Rows[0].ActualLoweringStorage = ""

	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport callback inout conservative row: %v", err)
	}
}

func TestValidateMemoryReportRejectsDerivedBorrowRowWithoutParent(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "aggregate_contains_borrow"
	report.Rows[0].ParentFactID = ""
	report.Rows[0].OwnerID = "xs"
	report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
		t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsV1DerivedBorrowRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"enum_payload_contains_borrow", "generic_wrapper_contains_borrow"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].OwnerID = "xs"
			report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV2DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"function_value_contains_borrow", "callback_arg_contains_borrow", "callback_inout_conservative"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].OwnerID = "xs"
			report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
			report.Rows[0].UnsafeClass = UnsafeSafe

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV3DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"interface_value_contains_borrow", "protocol_dispatch_borrow_conservative", "protocol_dispatch_noalias_conservative"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].OwnerID = "xs"
			report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
			report.Rows[0].UnsafeClass = UnsafeSafe

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV11DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"dynamic_existential_borrow_conservative",
		"static_witness_borrow_parent_validated",
		"dynamic_protocol_noalias_rejected",
		"witness_provenance_promotion_rejected",
		"protocol_dispatch_report_integrity",
	} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].OwnerID = "xs"
			report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
			report.Rows[0].UnsafeClass = UnsafeSafe
			report.Rows[0].LoweredArtifactID = ""
			report.Rows[0].PlannedStorage = ""
			report.Rows[0].ActualLoweringStorage = ""
			if strings.Contains(claim, "rejected") {
				report.Rows[0].ClaimLevel = ClaimRejected
				report.Rows[0].ValidatorStatus = ValidatorFail
				report.Rows[0].CostClass = CostUnsupportedRejected
			}

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV11ProtocolDispatchIntegrityWithoutReportFields(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "protocol_dispatch_report_integrity"
	report.Rows[0].ParentFactID = "fact:parent"
	report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
	report.Rows[0].UnsafeClass = UnsafeSafe
	report.Rows[0].LoweredArtifactID = ""
	report.Rows[0].PlannedStorage = ""
	report.Rows[0].ActualLoweringStorage = ""
	report.Rows[0].ValidatorName = "protocol_dispatch_report_integrity_validator"
	report.Rows[0].ValidatorStatus = ValidatorPass
	report.Rows[0].CostClass = CostZeroCostProven
	report.Rows[0].NormalBuildCheck = false

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "cost_class") || !strings.Contains(err.Error(), "normal_build_check") {
		t.Fatalf("ValidateReport error = %v, want cost_class and normal_build_check rejection", err)
	}
}

func TestValidateMemoryReportRejectsV4DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"async_boundary_borrow_conservative", "task_boundary_borrow_rejected", "actor_boundary_borrow_rejected", "boundary_noalias_conservative"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].OwnerID = "xs"
			report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed
			report.Rows[0].UnsafeClass = UnsafeSafe

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV5DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"unsafe_unknown_rejected_safe_facts", "unsafe_verified_root_allocation_base"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].LoweredArtifactID = ""
			report.Rows[0].PlannedStorage = ""
			report.Rows[0].ActualLoweringStorage = ""
			report.Rows[0].ProvenanceClass = ProvenanceUnsafeUnknown
			report.Rows[0].UnsafeClass = UnsafeUnknown
			report.Rows[0].ClaimLevel = ClaimRejected
			report.Rows[0].ValidatorName = "unsafe_unknown_fact_validator"
			report.Rows[0].ValidatorStatus = ValidatorFail
			report.Rows[0].CostClass = CostUnsupportedRejected
			if claim == "unsafe_verified_root_allocation_base" {
				report.Rows[0].ProvenanceClass = ProvenanceUnsafeVerifiedRoot
				report.Rows[0].UnsafeClass = UnsafeVerifiedRoot
				report.Rows[0].ClaimLevel = ClaimValidated
				report.Rows[0].ValidatorName = "unsafe_verified_root_bounds_validator"
				report.Rows[0].ValidatorStatus = ValidatorPass
				report.Rows[0].CostClass = CostZeroCostProven
			}

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"bounds_check_removed_with_proof_id", "raw_bounds_runtime_check_normal_build"} {
		t.Run(claim, func(t *testing.T) {
			report := validMemoryReport()
			report.Rows[0].Claim = claim
			report.Rows[0].ParentFactID = ""
			report.Rows[0].SourceStage = StageValidation
			report.Rows[0].ProvenanceClass = ProvenanceSafeKnown
			report.Rows[0].UnsafeClass = UnsafeSafe
			report.Rows[0].ClaimLevel = ClaimValidated
			report.Rows[0].ValidatorName = "bounds_proof_id_validator"
			report.Rows[0].ValidatorStatus = ValidatorPass
			report.Rows[0].CostClass = CostZeroCostProven
			if claim == "raw_bounds_runtime_check_normal_build" {
				report.Rows[0].SourceStage = StagePLIR
				report.Rows[0].ProvenanceClass = ProvenanceUnsafeChecked
				report.Rows[0].UnsafeClass = UnsafeChecked
				report.Rows[0].ValidatorName = "raw_bounds_width_validator"
				report.Rows[0].CostClass = CostDynamicCheckRequired
				report.Rows[0].NormalBuildCheck = true
			}

			err := ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("ValidateReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportAcceptsMemoryIdealV5UnsafeContractRows(t *testing.T) {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{
			{
				ProgramID:       "program",
				FunctionID:      "main",
				SiteID:          "raw:main:1:1",
				SourceFactID:    "fact:unsafe:unknown:rejected",
				ParentFactID:    "fact:unsafe:unknown",
				SourceStage:     StagePLIR,
				Claim:           "unsafe_unknown_rejected_safe_facts",
				ClaimLevel:      ClaimRejected,
				ProvenanceClass: ProvenanceUnsafeUnknown,
				UnsafeClass:     UnsafeUnknown,
				CostClass:       CostUnsupportedRejected,
				ValidatorName:   "unsafe_unknown_fact_validator",
				ValidatorStatus: ValidatorFail,
				Reason:          "unsafe_unknown raw pointer cannot produce safe facts or noalias",
			},
			{
				ProgramID:       "program",
				FunctionID:      "main",
				SiteID:          "alloc:main:p",
				SourceFactID:    "fact:unsafe:verified:allocation-base",
				ParentFactID:    "fact:raw:root",
				SourceStage:     StageAllocPlan,
				Claim:           "unsafe_verified_root_allocation_base",
				ClaimLevel:      ClaimValidated,
				ProvenanceClass: ProvenanceUnsafeVerifiedRoot,
				UnsafeClass:     UnsafeVerifiedRoot,
				CostClass:       CostZeroCostProven,
				ValidatorName:   "unsafe_verified_root_bounds_validator",
				ValidatorStatus: ValidatorPass,
				Reason:          "core.alloc_bytes verified root may project bounded allocation-base metadata",
			},
			{
				ProgramID:        "program",
				FunctionID:       "main",
				SiteID:           "raw:main:2:1",
				SourceFactID:     "fact:unsafe:runtime-contract",
				SourceStage:      StagePLIR,
				Claim:            "unsafe_contract_runtime_checkable",
				ClaimLevel:       ClaimValidated,
				ProvenanceClass:  ProvenanceUnsafeChecked,
				UnsafeClass:      UnsafeChecked,
				CostClass:        CostDynamicCheckRequired,
				NormalBuildCheck: true,
				ValidatorName:    "unsafe_runtime_contract_validator",
				ValidatorStatus:  ValidatorPass,
				Reason:           "nonnull/alignment/length are runtime-checkable unsafe contracts",
			},
			{
				ProgramID:       "program",
				FunctionID:      "main",
				SiteID:          "raw:main:3:1",
				SourceFactID:    "fact:unsafe:static-contract",
				SourceStage:     StagePLIR,
				Claim:           "unsafe_contract_static_untrusted",
				ClaimLevel:      ClaimConservative,
				ProvenanceClass: ProvenanceUnsafeUnknown,
				UnsafeClass:     UnsafeUnknown,
				AliasState:      AliasInvalidatedByCall,
				CostClass:       CostConservativeFallback,
				ValidatorName:   "unsafe_static_contract_validator",
				ValidatorStatus: ValidatorNotApplicable,
				Reason:          "unsafe noalias/lifetime/region contracts remain static-untrusted",
			},
		},
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport v5 unsafe contract rows: %v", err)
	}
}

func TestValidateMemoryReportRejectsCopyOwnedWithoutOwnedProvenance(t *testing.T) {
	report := validMemoryReport()
	report.Rows[0].Claim = "copy_owned"
	report.Rows[0].ProvenanceClass = ProvenanceSafeBorrowed

	err := ValidateReport(report)
	if err == nil || !strings.Contains(err.Error(), "copy_owned") {
		t.Fatalf("ValidateReport error = %v, want copy_owned provenance rejection", err)
	}
}

func TestValidateReportProjectionAcceptsBuildReportFromGraph(t *testing.T) {
	graph := validProjectionGraph(t)
	report := BuildReportFromGraph(graph)

	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection exact report: %v", err)
	}
}

func TestValidateReportProjectionRejectsUnknownSourceFactID(t *testing.T) {
	graph := validProjectionGraph(t)
	report := BuildReportFromGraph(graph)
	report.Rows[0].SourceFactID = "fact:projection:fake"

	err := ValidateReportProjection(graph, report)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("ValidateReportProjection error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateReportProjectionRejectsMissingProjectedGraphFact(t *testing.T) {
	graph := validProjectionGraph(t)
	report := BuildReportFromGraph(graph)
	report.Rows = report.Rows[:1]

	err := ValidateReportProjection(graph, report)
	if err == nil || !strings.Contains(err.Error(), "missing report row") {
		t.Fatalf("ValidateReportProjection error = %v, want missing projected fact rejection", err)
	}
}

func TestValidateReportProjectionRejectsAlteredCostClass(t *testing.T) {
	graph := validProjectionGraph(t)
	report := BuildReportFromGraph(graph)
	report.Rows[0].CostClass = CostConservativeFallback

	err := ValidateReportProjection(graph, report)
	if err == nil || !strings.Contains(err.Error(), "cost_class") {
		t.Fatalf("ValidateReportProjection error = %v, want cost_class preservation rejection", err)
	}
}

func TestValidateReportProjectionRejectsDroppedNormalBuildCheck(t *testing.T) {
	graph := validProjectionGraph(t)
	report := BuildReportFromGraph(graph)
	for i := range report.Rows {
		if report.Rows[i].SourceFactID == "fact:projection:instrumented-check" {
			report.Rows[i].NormalBuildCheck = false
			break
		}
	}

	err := ValidateReportProjection(graph, report)
	if err == nil || !strings.Contains(err.Error(), "normal_build_check") {
		t.Fatalf("ValidateReportProjection error = %v, want normal_build_check preservation rejection", err)
	}
}

func validMemoryReport() Report {
	return Report{
		SchemaVersion: ReportSchemaV1,
		Rows: []ReportRow{{
			ProgramID:             "program",
			FunctionID:            "main",
			SiteID:                "site",
			SourceSpan:            "main.tetra:1:1",
			SourceFactID:          "fact:safe",
			SourceStage:           StageValidation,
			Claim:                 "validated allocation metadata",
			ClaimLevel:            ClaimValidated,
			ProvenanceClass:       ProvenanceSafeKnown,
			UnsafeClass:           UnsafeSafe,
			LoweredArtifactID:     "ir:main:0",
			PlannedStorage:        StorageHeap,
			ActualLoweringStorage: StorageHeap,
			CostClass:             CostZeroCostProven,
			ValidatorName:         "memory_report_schema_v1",
			ValidatorStatus:       ValidatorPass,
			Reason:                "fixture",
		}},
	}
}

func validProjectionGraph(t *testing.T) *Graph {
	t.Helper()
	graph := NewGraph("program")
	if _, err := graph.AddFact(Fact{
		ID:              "fact:projection:borrow",
		FunctionID:      "main",
		SiteID:          "projection:borrow:1",
		SourceStage:     StageSemantics,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		BorrowState:     BorrowImmutable,
		Claim:           "borrowed_imm",
	}); err != nil {
		t.Fatalf("AddFact borrowed projection fixture: %v", err)
	}
	if _, err := graph.AddFact(Fact{
		ID:               "fact:projection:instrumented-check",
		FunctionID:       "main",
		SiteID:           "projection:check:1",
		SourceStage:      StageValidation,
		ProvenanceClass:  ProvenanceSafeKnown,
		UnsafeClass:      UnsafeSafe,
		Claim:            "instrumented_memory_check",
		CostClass:        CostInstrumentationOnly,
		NormalBuildCheck: true,
	}); err != nil {
		t.Fatalf("AddFact instrumented projection fixture: %v", err)
	}
	return graph
}
