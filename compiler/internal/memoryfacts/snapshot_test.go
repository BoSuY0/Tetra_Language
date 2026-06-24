package memoryfacts

import (
	"strings"
	"sync"
	"testing"
)

func TestSnapshotImmutableIndexesAndDigest(t *testing.T) {
	graph := NewGraph("program")
	addSnapshotFact(t, graph, "fact:value", "main", "v", "alloc:v", "owner:v")
	parentID, err := graph.AddFact(Fact{
		ID:              "fact:parent",
		FunctionID:      "main",
		ValueID:         "parent",
		SiteID:          "site:parent",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimProvenanceKnown,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := graph.DeriveFact(parentID, Fact{
		ID:              "fact:child",
		FunctionID:      "main",
		ValueID:         "parent",
		SiteID:          "site:child",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimRegionAlive,
	}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	facts := snapshot.Facts()
	facts[0].Claim = "mutated"
	if fact, ok := snapshot.Fact("fact:value"); !ok || fact.Claim != ClaimOwned {
		t.Fatalf("snapshot fact mutated through returned slice: %#v ok=%v", fact, ok)
	}
	if err := graph.InvalidateFact("fact:value", "after snapshot"); err != nil {
		t.Fatal(err)
	}
	if fact, _ := snapshot.Fact("fact:value"); fact.ValidationState == ValidationInvalidated {
		t.Fatalf("snapshot changed after graph mutation: %#v", fact)
	}
	if got := snapshot.FactsForValue(ValueKey{FunctionID: "main", ValueID: "v"}); len(got) != 1 {
		t.Fatalf("FactsForValue returned %d facts, want 1", len(got))
	}
	if got := snapshot.FactsForAllocation(AllocationKey{FunctionID: "main", AllocationSiteID: "alloc:v"}); len(got) != 1 {
		t.Fatalf("FactsForAllocation returned %d facts, want 1", len(got))
	}
	if got := snapshot.DerivedFacts(parentID); len(got) != 1 || got[0].ID != "fact:child" {
		t.Fatalf("DerivedFacts returned %#v, want fact:child", got)
	}

	reordered := NewGraph("program")
	if _, err := reordered.AddFact(Fact{
		ID:              "fact:parent",
		FunctionID:      "main",
		ValueID:         "parent",
		SiteID:          "site:parent",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimProvenanceKnown,
	}); err != nil {
		t.Fatal(err)
	}
	addSnapshotFact(t, reordered, "fact:value", "main", "v", "alloc:v", "owner:v")
	if _, err := reordered.DeriveFact("fact:parent", Fact{
		ID:              "fact:child",
		FunctionID:      "main",
		ValueID:         "parent",
		SiteID:          "site:child",
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimRegionAlive,
	}); err != nil {
		t.Fatal(err)
	}
	reorderedSnapshot, err := reordered.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Digest() != reorderedSnapshot.Digest() {
		t.Fatalf("digest depends on insertion order: %s != %s", snapshot.Digest(), reorderedSnapshot.Digest())
	}
}

func TestSnapshotConcurrentReads(t *testing.T) {
	graph := NewGraph("program")
	addSnapshotFact(t, graph, "fact:value", "main", "v", "alloc:v", "owner:v")
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 128; j++ {
				_ = snapshot.Digest()
				_ = snapshot.Facts()
				_, _ = snapshot.ResolveAllocation(ValueKey{FunctionID: "main", ValueID: "v"})
			}
		}()
	}
	wg.Wait()
}

func TestResolveAllocationRejectsConflictingEvidence(t *testing.T) {
	graph := NewGraph("program")
	addSnapshotFact(t, graph, "fact:a", "main", "v", "alloc:v", "owner:a")
	addSnapshotFact(t, graph, "fact:b", "main", "v", "alloc:v", "owner:b")
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = snapshot.ResolveAllocation(ValueKey{FunctionID: "main", ValueID: "v"})
	if err == nil || !strings.Contains(err.Error(), "conflicting owner") {
		t.Fatalf("ResolveAllocation error = %v, want conflicting owner", err)
	}
}

func TestResolveProofFailClosed(t *testing.T) {
	graph := NewGraph("program")
	addProofFact(t, graph, Fact{
		ID:                 "proof:bounds",
		FunctionID:         "main",
		SiteID:             "site:proof",
		SourceStage:        StageValidation,
		ProvenanceClass:    ProvenanceSafeKnown,
		UnsafeClass:        UnsafeSafe,
		Claim:              ClaimBoundsProofID,
		ProofID:            "p1",
		ProofKind:          ProofBounds,
		ProofSubjectBaseID: "xs",
		ProofIndexValueID:  "i",
		ProofOperation:     "index_load",
		ProofRange:         "0 <= i < len(xs)",
		IslandID:           "island:xs",
		Epoch:              2,
		BaseID:             "xs",
		ValidationState:    ValidationPass,
		ValidatorName:      "bounds_proof_id_validator",
	})
	addProofFact(t, graph, Fact{
		ID:                 "proof:unsafe",
		FunctionID:         "main",
		SiteID:             "site:unsafe",
		SourceStage:        StageValidation,
		ProvenanceClass:    ProvenanceUnsafeUnknown,
		UnsafeClass:        UnsafeUnknown,
		Claim:              ClaimExternalUnknown,
		ProofID:            "unsafe",
		ProofKind:          ProofBounds,
		ProofSubjectBaseID: "raw",
		ProofOperation:     "index_load",
		ValidationState:    ValidationPass,
		ValidatorName:      "bounds_proof_id_validator",
		CostClass:          CostConservativeFallback,
	})
	addProofFact(t, graph, Fact{
		ID:                 "proof:unsafe-live",
		FunctionID:         "main",
		SiteID:             "site:unsafe-live",
		SourceStage:        StageValidation,
		ProvenanceClass:    ProvenanceUnsafeUnknown,
		UnsafeClass:        UnsafeUnknown,
		Claim:              ClaimExternalUnknown,
		ProofID:            "unsafe-live",
		ProofKind:          ProofBounds,
		ProofSubjectBaseID: "raw-live",
		ProofOperation:     "index_load",
		ValidationState:    ValidationPass,
		ValidatorName:      "bounds_proof_id_validator",
		CostClass:          CostConservativeFallback,
	})
	if err := graph.InvalidateFact("proof:unsafe", "stale"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.ResolveProof(ProofQuery{FunctionID: "main", ProofID: "p1", Kind: ProofBounds, SubjectBaseID: "xs", Operation: "index_load", IslandID: "island:xs", Epoch: 2}); !ok {
		t.Fatalf("ResolveProof did not find exact validated proof")
	}
	if got := snapshot.FactsForProof(ProofKey{FunctionID: "main", ProofID: "p1"}); len(got) != 1 || got[0].ID != "proof:bounds" {
		t.Fatalf("FactsForProof returned %#v, want proof:bounds", got)
	}
	for name, query := range map[string]ProofQuery{
		"stale epoch":        {FunctionID: "main", ProofID: "p1", Kind: ProofBounds, SubjectBaseID: "xs", Operation: "index_load", IslandID: "island:xs", Epoch: 1},
		"mismatched subject": {FunctionID: "main", ProofID: "p1", Kind: ProofBounds, SubjectBaseID: "ys", Operation: "index_load", IslandID: "island:xs", Epoch: 2},
		"invalidated proof":  {FunctionID: "main", ProofID: "unsafe", Kind: ProofBounds, SubjectBaseID: "raw", Operation: "index_load"},
		"unsafe proof":       {FunctionID: "main", ProofID: "unsafe-live", Kind: ProofBounds, SubjectBaseID: "raw-live", Operation: "index_load"},
	} {
		if proof, ok := snapshot.ResolveProof(query); ok {
			t.Fatalf("%s resolved unexpectedly: %#v", name, proof)
		}
	}
}

func TestGraphApplyAtomicAndStageRegression(t *testing.T) {
	graph := NewGraph("program")
	if err := graph.AdvanceTo(StagePLIR); err != nil {
		t.Fatal(err)
	}
	err := graph.Apply(Delta{
		Stage: StagePLIR,
		Add: []Fact{{
			ID:              "fact:ok",
			FunctionID:      "main",
			ValueID:         "v",
			SiteID:          "site:ok",
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			Claim:           ClaimProvenanceKnown,
		}, {
			ID:              "fact:bad",
			FunctionID:      "main",
			ValueID:         "v",
			SiteID:          "site:bad",
			SourceStage:     StageSemantics,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			Claim:           ClaimProvenanceKnown,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "stage regression") {
		t.Fatalf("Apply error = %v, want stage regression", err)
	}
	if facts := graph.Facts(); len(facts) != 0 {
		t.Fatalf("failed delta mutated graph: %#v", facts)
	}
	if err := graph.Apply(Delta{Stage: StageAllocPlan, Add: []Fact{{
		ID:              "fact:planned",
		FunctionID:      "main",
		ValueID:         "v",
		SiteID:          "site:planned",
		SourceStage:     StageAllocPlan,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimTrustedStorage,
	}}}); err != nil {
		t.Fatalf("valid Apply: %v", err)
	}
	if graph.CurrentStage() != StageAllocPlan {
		t.Fatalf("CurrentStage = %q, want %q", graph.CurrentStage(), StageAllocPlan)
	}
}

func addSnapshotFact(t *testing.T, graph *Graph, id FactID, fn string, value string, alloc string, owner string) {
	t.Helper()
	if _, err := graph.AddFact(Fact{
		ID:               id,
		FunctionID:       fn,
		ValueID:          value,
		SiteID:           "site:" + alloc,
		SourceStage:      StagePLIR,
		ProvenanceClass:  ProvenanceSafeOwned,
		UnsafeClass:      UnsafeSafe,
		AllocationSiteID: alloc,
		OwnerID:          owner,
		Claim:            ClaimOwned,
	}); err != nil {
		t.Fatal(err)
	}
}

func addProofFact(t *testing.T, graph *Graph, fact Fact) {
	t.Helper()
	if _, err := graph.AddFact(fact); err != nil {
		t.Fatal(err)
	}
}
