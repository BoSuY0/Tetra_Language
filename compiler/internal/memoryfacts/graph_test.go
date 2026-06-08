package memoryfacts

import (
	"strings"
	"testing"
)

func TestMemoryFactsRejectsUnsafeUnknownToSafeKnown(t *testing.T) {
	graph := NewGraph("program")
	parentID, err := graph.AddFact(Fact{
		ID:              "unsafe-root",
		FunctionID:      "main",
		SiteID:          "main:1:1",
		SourceStage:     StageUnsafeGatewayLowering,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		Claim:           "external raw pointer",
	})
	if err != nil {
		t.Fatalf("AddFact unsafe root: %v", err)
	}

	_, err = graph.DeriveFact(parentID, Fact{
		ID:              "bad-safe-known",
		FunctionID:      "main",
		SiteID:          "main:1:2",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeChecked,
		Claim:           "unsafe pointer became safe",
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("DeriveFact error = %v, want unsafe_unknown rejection", err)
	}
}

func TestMemoryFactsRejectsDuplicateFactID(t *testing.T) {
	graph := NewGraph("program")
	if _, err := graph.AddFact(Fact{
		ID:              "fact:duplicate",
		FunctionID:      "main",
		SiteID:          "main:1:1",
		SourceStage:     StageSemantics,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		BorrowState:     BorrowImmutable,
		Claim:           "borrowed_imm",
	}); err != nil {
		t.Fatalf("AddFact first duplicate fixture: %v", err)
	}

	_, err := graph.AddFact(Fact{
		ID:              "fact:duplicate",
		FunctionID:      "main",
		SiteID:          "main:1:2",
		SourceStage:     StageSemantics,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		BorrowState:     BorrowImmutable,
		Claim:           "borrowed_imm",
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate fact_id") {
		t.Fatalf("AddFact duplicate error = %v, want duplicate fact_id rejection", err)
	}
}

func TestMemoryFactsRejectsDirectSafeBorrowedFromUnsafeUnknown(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "bad-safe-borrowed",
		FunctionID:      "main",
		SiteID:          "main:1:1",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeUnknown,
		BorrowState:     BorrowImmutable,
		Claim:           "borrowed_imm",
	})
	if err == nil || !strings.Contains(err.Error(), "safe provenance") {
		t.Fatalf("AddFact error = %v, want safe provenance rejection", err)
	}
}

func TestMemoryFactsRejectsDirectSafeOwnedFromUnsafeUnknown(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "bad-safe-owned",
		FunctionID:      "main",
		SiteID:          "main:1:2",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeOwned,
		UnsafeClass:     UnsafeUnknown,
		Claim:           "copy_owned",
	})
	if err == nil || !strings.Contains(err.Error(), "safe provenance") {
		t.Fatalf("AddFact error = %v, want safe provenance rejection", err)
	}
}

func TestMemoryFactsRejectsUnsafeUnknownNoAliasAndBoundsProofClaims(t *testing.T) {
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
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:              FactID("unsafe-" + tc.name),
				FunctionID:      "main",
				SiteID:          "main:1:1",
				SourceStage:     StagePLIR,
				ProvenanceClass: ProvenanceUnsafeUnknown,
				UnsafeClass:     UnsafeUnknown,
				AliasState:      tc.aliasState,
				Claim:           tc.claim,
			})
			if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
				t.Fatalf("AddFact error = %v, want unsafe_unknown rejection", err)
			}
		})
	}
}

func TestMemoryFactsRejectsValidatedBoundsProofWithoutTypedProofFields(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "fact:bounds:missing-typed-proof",
		FunctionID:      "sum",
		SiteID:          "bounds:sum:3",
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "bounds_check_removed_with_proof_id",
		ValidationState: ValidationPass,
		ValidatorName:   "bounds_proof_id_validator",
		CostClass:       CostZeroCostProven,
	})
	if err == nil || !strings.Contains(err.Error(), "typed proof fields") || !strings.Contains(err.Error(), "proof_kind") {
		t.Fatalf("AddFact error = %v, want typed proof field rejection", err)
	}
}

func TestMemoryFactsRejectsValidatedConservativeNoAliasBoundary(t *testing.T) {
	graph := NewGraph("program")
	parentID, err := graph.AddFact(Fact{
		ID:              "borrowed-mut-parent",
		FunctionID:      "callbackBoundary",
		ValueID:         "param:dst",
		SiteID:          "test.tetra:4:19",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		BorrowState:     BorrowMutable,
		Claim:           "borrowed_mut",
	})
	if err != nil {
		t.Fatalf("AddFact parent: %v", err)
	}
	_, err = graph.DeriveFact(parentID, Fact{
		ID:              "bad-callback-conservative-validated",
		FunctionID:      "callbackBoundary",
		ValueID:         "param:dst",
		SiteID:          "test.tetra:5:5",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		AliasState:      AliasInvalidatedByCall,
		Claim:           "callback_inout_conservative",
		ValidationState: ValidationPass,
		ValidatorName:   "callback_alias_conservative_validator",
		CostClass:       CostConservativeFallback,
	})
	if err == nil || !strings.Contains(err.Error(), "conservative") {
		t.Fatalf("AddFact error = %v, want conservative noalias boundary validation rejection", err)
	}
}

func TestMemoryFactsRejectsUnsafeCheckedGenericPromotions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		claim      string
		provenance ProvenanceClass
		aliasState AliasState
		want       string
	}{
		{name: "safe known", claim: "safe_known", provenance: ProvenanceSafeKnown, want: "unsafe_checked"},
		{name: "provenance known", claim: "provenance_known", provenance: ProvenanceUnsafeChecked, want: "unsafe_checked"},
		{name: "noalias", claim: "no_alias", provenance: ProvenanceUnsafeChecked, aliasState: AliasUnique, want: "unsafe_checked"},
		{name: "bounds check elimination", claim: "bounds_check_eliminated", provenance: ProvenanceUnsafeChecked, want: "proof id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:              FactID("unsafe-checked-" + strings.ReplaceAll(tc.name, " ", "-")),
				FunctionID:      "main",
				SiteID:          "main:1:1",
				SourceStage:     StagePLIR,
				ProvenanceClass: tc.provenance,
				UnsafeClass:     UnsafeChecked,
				AliasState:      tc.aliasState,
				Claim:           tc.claim,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AddFact error = %v, want %q rejection", err, tc.want)
			}
		})
	}
}

func TestMemoryFactsRejectsDynamicRawOffsetZeroCostPromotion(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "bad-dynamic-raw-offset-zero-cost",
		FunctionID:      "read_at",
		ValueID:         "q",
		SiteID:          "test.tetra:6:17",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeChecked,
		UnsafeClass:     UnsafeChecked,
		Claim:           "derived_allocation_offset",
		CostClass:       CostZeroCostProven,
		Reason:          "core.ptr_add raw_pointer_bounds: checked_external_unknown base:p offset:n",
	})
	if err == nil || !strings.Contains(err.Error(), "dynamic raw") || !strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("AddFact error = %v, want dynamic raw zero-cost rejection", err)
	}
}

func TestMemoryFactsRejectsUnvalidatedStorageLoweringZeroCost(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:                    "storage-lowering-zero-cost-without-proof",
		FunctionID:            "main",
		SiteID:                "alloc:main:stack",
		SourceStage:           StageAllocPlan,
		ProvenanceClass:       ProvenanceSafeOwned,
		UnsafeClass:           UnsafeSafe,
		StoragePlan:           StorageStack,
		ActualLoweringStorage: StorageStack,
		Claim:                 "storage_lowering",
		CostClass:             CostZeroCostProven,
		Reason:                "forged storage report without allocation lowering validation",
	})
	if err == nil || !strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("AddFact error = %v, want zero_cost_proven proof rejection", err)
	}
}

func TestMemoryFactsRejectsValidatedCapMemAsProof(t *testing.T) {
	for _, tc := range []struct {
		name       string
		claim      string
		aliasState AliasState
		want       string
	}{
		{name: "safe provenance", claim: "provenance_known", want: "cap.mem"},
		{name: "noalias", claim: "no_alias", aliasState: AliasMutableExclusive, want: "cap.mem"},
		{name: "bounds proof", claim: "index_in_range", want: "cap.mem"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:              FactID("cap-mem-" + strings.ReplaceAll(tc.name, " ", "-")),
				FunctionID:      "main",
				SiteID:          "main:cap.mem",
				SourceStage:     StagePLIR,
				ProvenanceClass: ProvenanceSafeKnown,
				UnsafeClass:     UnsafeSafe,
				AliasState:      tc.aliasState,
				Claim:           tc.claim,
				ValidationState: ValidationPass,
				ValidatorName:   "cap_mem_authorization_validator",
				CostClass:       CostZeroCostProven,
				Reason:          "cap.mem authorized raw helper call",
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AddFact error = %v, want %q rejection", err, tc.want)
			}
		})
	}
}

func TestMemoryFactsRejectsBareBoundsCheckEliminatedWithoutProofID(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "bad-bounds-eliminated",
		FunctionID:      "main",
		SiteID:          "bounds:main:1",
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "bounds_check_eliminated",
		CostClass:       CostZeroCostProven,
	})
	if err == nil || !strings.Contains(err.Error(), "proof id") {
		t.Fatalf("AddFact error = %v, want proof id rejection", err)
	}
}

func TestMemoryFactsRejectsUnsafeVerifiedRootGenericClaims(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.AddFact(Fact{
		ID:              "raw-root-provenance-known",
		FunctionID:      "main",
		SiteID:          "main:1:1",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeVerifiedRoot,
		UnsafeClass:     UnsafeVerifiedRoot,
		Claim:           "provenance_known",
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe_verified_root") {
		t.Fatalf("AddFact error = %v, want unsafe_verified_root generic-claim rejection", err)
	}
}

func TestMemoryFactsRejectsValidatedUnsafeUnknownTrustedStorage(t *testing.T) {
	for _, storage := range []StorageClass{StorageStack, StorageRegion, StorageFunctionTempRegion} {
		t.Run(string(storage), func(t *testing.T) {
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:                    FactID("unsafe-storage-" + storage),
				FunctionID:            "main",
				SiteID:                "main:1:1",
				SourceStage:           StageAllocPlan,
				ProvenanceClass:       ProvenanceUnsafeUnknown,
				UnsafeClass:           UnsafeUnknown,
				StoragePlan:           storage,
				ActualLoweringStorage: storage,
				ValidationState:       ValidationPass,
				ValidatorName:         "allocation_lowering_validator",
				LoweredArtifactID:     "ir:main:unsafe",
				Claim:                 "storage_lowering",
			})
			if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
				t.Fatalf("AddFact error = %v, want unsafe_unknown storage rejection", err)
			}
		})
	}
}

func TestMemoryFactsRejectsValidatedTaskActorRegionStorageWithoutRuntimeProof(t *testing.T) {
	for _, storage := range []StorageClass{StorageTaskRegion, StorageActorMoveRegion} {
		t.Run(string(storage), func(t *testing.T) {
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:                    FactID("boundary-storage-" + storage),
				FunctionID:            "main",
				SiteID:                "alloc:main:boundary",
				SourceStage:           StageAllocPlan,
				ProvenanceClass:       ProvenanceSafeOwned,
				UnsafeClass:           UnsafeSafe,
				StoragePlan:           storage,
				ActualLoweringStorage: storage,
				ValidationState:       ValidationPass,
				ValidatorName:         "allocation_lowering_validator",
				LoweredArtifactID:     "ir:main:boundary:" + string(storage),
				Claim:                 "storage_lowering",
			})
			if err == nil || !strings.Contains(err.Error(), "runtime") {
				t.Fatalf("AddFact error = %v, want runtime proof rejection", err)
			}
		})
	}
}

func TestMemoryFactsRejectsValidatedNoAliasFromUnknownAlias(t *testing.T) {
	graph := NewGraph("program")
	factID, err := graph.AddFact(Fact{
		ID:              "unknown-noalias",
		FunctionID:      "main",
		SiteID:          "main:1:3",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		AliasState:      AliasUnknownConservative,
		Claim:           "no_alias",
	})
	if err != nil {
		t.Fatalf("AddFact unknown noalias: %v", err)
	}
	err = graph.MarkValidated(factID, "alias_validator")
	if err == nil || !strings.Contains(err.Error(), "unknown alias") {
		t.Fatalf("MarkValidated error = %v, want unknown alias no_alias rejection", err)
	}
}

func TestMemoryFactsRejectsDerivedFactWithoutParent(t *testing.T) {
	graph := NewGraph("program")
	_, err := graph.DeriveFact("", Fact{
		ID:              "derived",
		FunctionID:      "main",
		SiteID:          "main:2:1",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		Claim:           "borrowed view",
	})
	if err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("DeriveFact error = %v, want missing parent rejection", err)
	}
}

func TestMemoryFactsRejectsValidatedLoweringWithoutArtifact(t *testing.T) {
	graph := NewGraph("program")
	factID, err := graph.AddFact(Fact{
		ID:                    "stack-plan",
		FunctionID:            "main",
		SiteID:                "alloc:main:1:1",
		SourceStage:           StageAllocPlan,
		ProvenanceClass:       ProvenanceSafeOwned,
		UnsafeClass:           UnsafeSafe,
		StoragePlan:           StorageStack,
		ActualLoweringStorage: StorageStack,
		Claim:                 "stack lowering claim",
	})
	if err != nil {
		t.Fatalf("AddFact stack plan: %v", err)
	}

	err = graph.MarkValidated(factID, "allocation_lowering_validator")
	if err == nil || !strings.Contains(err.Error(), "lowered_artifact_id") {
		t.Fatalf("MarkValidated error = %v, want lowered_artifact_id rejection", err)
	}
}

func TestMemoryFactsRejectsValidatedTrustedStorageHeapFallback(t *testing.T) {
	for _, planned := range []StorageClass{StorageStack, StorageRegion, StorageFunctionTempRegion} {
		t.Run(string(planned), func(t *testing.T) {
			graph := NewGraph("program")
			_, err := graph.AddFact(Fact{
				ID:                    FactID("fallback-" + planned),
				FunctionID:            "main",
				SiteID:                "alloc:main:1:1",
				SourceStage:           StageAllocPlan,
				ProvenanceClass:       ProvenanceSafeOwned,
				UnsafeClass:           UnsafeSafe,
				StoragePlan:           planned,
				ActualLoweringStorage: StorageHeap,
				ValidationState:       ValidationPass,
				ValidatorName:         "allocation_lowering_validator",
				LoweredArtifactID:     "ir:main:xs:Heap",
				Claim:                 "storage_lowering",
			})
			if err == nil || !strings.Contains(err.Error(), "validated "+string(planned)+" claim cannot lower as Heap") {
				t.Fatalf("AddFact error = %v, want trusted storage heap fallback rejection", err)
			}
		})
	}
}

func TestMemoryFactsAcceptsSafeBorrowedView(t *testing.T) {
	graph := NewGraph("program")
	if _, err := graph.AddFact(Fact{
		ID:              "borrow-xs-window",
		FunctionID:      "main",
		SiteID:          "main:3:1",
		SourceStage:     StageSemantics,
		TypeName:        "[]u8",
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		BorrowState:     BorrowImmutable,
		OwnerID:         "xs",
		Claim:           "borrowed_imm",
	}); err != nil {
		t.Fatalf("AddFact safe borrowed view: %v", err)
	}
	if err := graph.Validate(); err != nil {
		t.Fatalf("Validate safe borrowed graph: %v", err)
	}
}

func TestMemoryFactsAcceptsUnsafeVerifiedAllocRoot(t *testing.T) {
	graph := NewGraph("program")
	factID, err := graph.AddFact(Fact{
		ID:               "raw-alloc-root",
		FunctionID:       "main",
		SiteID:           "alloc_bytes:main:4:1",
		SourceStage:      StageUnsafeGatewayLowering,
		TypeName:         "ptr",
		ProvenanceClass:  ProvenanceUnsafeVerifiedRoot,
		UnsafeClass:      UnsafeVerifiedRoot,
		AllocationSiteID: "core.alloc_bytes",
		Claim:            "allocation_base_metadata",
	})
	if err != nil {
		t.Fatalf("AddFact unsafe verified root: %v", err)
	}
	if err := graph.AttachLoweredArtifact(factID, "ir:main:alloc_bytes:0"); err != nil {
		t.Fatalf("AttachLoweredArtifact: %v", err)
	}
	if err := graph.MarkValidated(factID, "raw_bounds_validator"); err != nil {
		t.Fatalf("MarkValidated unsafe verified root: %v", err)
	}
	if err := graph.Validate(); err != nil {
		t.Fatalf("Validate unsafe verified root graph: %v", err)
	}
}

func TestMemoryFactsKeepsRawSliceUnknownConservative(t *testing.T) {
	graph := NewGraph("program")
	if _, err := graph.AddFact(Fact{
		ID:              "raw-slice-unknown",
		FunctionID:      "main",
		SiteID:          "raw_slice:main:5:1",
		SourceStage:     StageUnsafeGatewayLowering,
		TypeName:        "[]u8",
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		Claim:           "external_unknown",
		Reason:          "raw_slice_from_parts over unknown external pointer",
	}); err != nil {
		t.Fatalf("AddFact raw slice unknown: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if len(report.Rows) != 1 {
		t.Fatalf("report rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.ClaimLevel != ClaimConservative || row.ValidatorStatus != ValidatorNotApplicable {
		t.Fatalf("raw slice unknown report row = %+v, want conservative/not_applicable", row)
	}
	if row.ProvenanceClass == ProvenanceSafeKnown {
		t.Fatalf("raw slice unknown became safe_known: %+v", row)
	}
}
