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
