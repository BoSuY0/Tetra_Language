package fromlowering

import (
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/loweringevidence"
	"tetra_language/compiler/internal/memoryfacts"
)

func TestDeltaAddsEvidenceOnlyLoweringFact(t *testing.T) {
	graph := memoryfacts.NewGraph("program:test")
	parentID := memoryfacts.FactID("allocplan:main:xs")
	if err := graph.Apply(memoryfacts.Delta{
		Stage: memoryfacts.StageAllocPlan,
		Add: []memoryfacts.Fact{{
			ID:                    parentID,
			FunctionID:            "main",
			ValueID:               "xs",
			AllocationSiteID:      "xs",
			ProvenanceClass:       memoryfacts.ProvenanceSafeKnown,
			UnsafeClass:           memoryfacts.UnsafeSafe,
			StoragePlan:           memoryfacts.StorageStack,
			ActualLoweringStorage: memoryfacts.StorageUnknownConservative,
			Claim:                 memoryfacts.ClaimTrustedStorage,
			SourceStage:           memoryfacts.StageAllocPlan,
			CostClass:             memoryfacts.CostInstrumentationOnly,
		}},
	}); err != nil {
		t.Fatalf("seed allocplan fact: %v", err)
	}

	evidence := loweringevidence.Evidence{Allocations: []loweringevidence.Allocation{{
		Function:       "main",
		AllocationID:   "xs",
		ValueID:        "xs",
		PlannedStorage: allocplan.StorageStack,
		ActualStorage:  allocplan.StorageStack,
		ArtifactID:     "ir:main:3:3:xs",
		DecisionCode:   "lowering:emitted:Stack",
		SourceFactIDs:  []string{string(parentID)},
	}}}

	if err := AddFacts(graph, evidence); err != nil {
		t.Fatalf("AddFacts: %v", err)
	}
	fact, ok := graph.Fact(memoryfacts.FactID("lowering:main:xs"))
	if !ok {
		t.Fatalf("missing lowering fact")
	}
	if fact.Claim != memoryfacts.ClaimStorageLowering {
		t.Fatalf("claim = %q, want storage_lowering", fact.Claim)
	}
	if fact.ValidationState != memoryfacts.ValidationNotRun || fact.ValidatorName != "" {
		t.Fatalf("validation = %q/%q, want not_run without validator", fact.ValidationState, fact.ValidatorName)
	}
	if fact.LoweredArtifactID != "ir:main:3:3:xs" {
		t.Fatalf("artifact = %q", fact.LoweredArtifactID)
	}
	if fact.ParentFactID != parentID {
		t.Fatalf("parent = %q, want %q", fact.ParentFactID, parentID)
	}
}

func TestDeltaRejectsLoweringFactWithoutParent(t *testing.T) {
	_, err := Delta(loweringevidence.Evidence{Allocations: []loweringevidence.Allocation{{
		Function:       "main",
		AllocationID:   "xs",
		PlannedStorage: allocplan.StorageStack,
		ActualStorage:  allocplan.StorageStack,
		ArtifactID:     "ir:main:3:3:xs",
		DecisionCode:   "lowering:emitted:Stack",
	}}})
	if err == nil {
		t.Fatalf("Delta succeeded without source fact parent")
	}
}
