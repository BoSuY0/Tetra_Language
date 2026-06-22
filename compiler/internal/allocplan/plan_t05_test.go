package allocplan

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

func TestBuildRequiresCanonicalAllocationEvidence(t *testing.T) {
	prog := t05SyntheticProgram("main", plir.EscapeNoEscape)
	graph := memoryfacts.NewGraph("program:test")
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Build(Input{Program: prog, Snapshot: snapshot})
	if err == nil || !strings.Contains(err.Error(), "allocation evidence") {
		t.Fatalf("Build error = %v, want missing allocation evidence", err)
	}
}

func TestBuildRejectsConflictingGraphAndPLIREvidence(t *testing.T) {
	prog := t05SyntheticProgram("main", plir.EscapeNoEscape)
	graph := memoryfacts.NewGraph("program:test")
	if _, err := graph.AddFact(memoryfacts.Fact{
		ID:               "fact:conflict",
		FunctionID:       "main",
		ValueID:          "alloc_intent:xs",
		SiteID:           "site:conflict",
		SourceStage:      memoryfacts.StagePLIR,
		TypeName:         "[]u16",
		ProvenanceClass:  memoryfacts.ProvenanceSafeOwned,
		UnsafeClass:      memoryfacts.UnsafeSafe,
		EscapeState:      memoryfacts.EscapeNoEscape,
		Claim:            memoryfacts.ClaimOwned,
		ValidationState:  memoryfacts.ValidationNotRun,
		NormalBuildCheck: true,
		LifetimeBirth:    "entry",
		LifetimeDeath:    "return",
		LifetimeOwner:    "xs",
		OwnerID:          "xs",
		AllocationSiteID: "core.make_u8",
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Build(Input{Program: prog, Snapshot: snapshot})
	if err == nil || !strings.Contains(err.Error(), "PLIR") {
		t.Fatalf("Build error = %v, want graph/PLIR conflict", err)
	}
}

func TestBuildUsesTypedEscapeEvidenceOnly(t *testing.T) {
	prog := t05SyntheticProgram("actor_named_call", plir.EscapeNoEscape)
	prog.Funcs[0].Ops = append(prog.Funcs[0].Ops, plir.Operation{
		ID:     "op:actor-name",
		Kind:   plir.OpCall,
		Inputs: []string{"xs"},
		Note:   "core.send_typed actor payload name only",
	})
	plan := t05BuildPlan(t, prog, Options{EnableStackLowering: true})
	alloc := findAllocation(t, plan, "actor_named_call", "xs")
	if alloc.Escape != EscapeNoEscape {
		t.Fatalf("string-named actor call changed escape: %+v", alloc)
	}
	if alloc.ActualLoweringStorage != StorageUnknownConservative || alloc.LoweringStatus != "pending" {
		t.Fatalf("planned allocation actual/lowering = %q/%q, want UnknownConservative/pending", alloc.ActualLoweringStorage, alloc.LoweringStatus)
	}
	if len(alloc.SourceFactIDs) == 0 || alloc.DecisionCode == "" || alloc.PlanDigest == "" {
		t.Fatalf("allocation missing evidence metadata: %+v", alloc)
	}
	if err := VerifyPlanned(plan); err != nil {
		t.Fatalf("VerifyPlanned: %v", err)
	}
	if err := VerifyLowered(plan); err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("VerifyLowered error = %v, want pending rejection", err)
	}
}

func TestBuildUsesTypedActorTaskAndUnsafeEvidence(t *testing.T) {
	for _, test := range []struct {
		name       string
		escape     memoryfacts.EscapeState
		wantEscape EscapeClass
		wantDomain runtimeabi.MemoryDomainKind
	}{
		{name: "actor", escape: memoryfacts.EscapeActor, wantEscape: EscapeActor, wantDomain: runtimeabi.DomainProcess},
		{name: "task", escape: memoryfacts.EscapeTask, wantEscape: EscapeTask, wantDomain: runtimeabi.DomainProcess},
		{name: "unsafe", escape: memoryfacts.EscapeUnsafe, wantEscape: EscapeUnsafe, wantDomain: runtimeabi.DomainExternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := t05SyntheticProgram(test.name, plir.EscapeNoEscape)
			snapshot := t05SnapshotWithEscape(t, prog, test.escape, memoryfacts.ProvenanceSafeOwned, memoryfacts.UnsafeSafe)
			plan, err := Build(Input{Program: prog, Snapshot: snapshot, Options: Options{EnableStackLowering: true}})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			alloc := findAllocation(t, plan, test.name, "xs")
			if alloc.Escape != test.wantEscape || alloc.Storage != StorageHeap {
				t.Fatalf("typed escape allocation = %+v, want %s/Heap", alloc, test.wantEscape)
			}
			if alloc.Domain == nil || alloc.Domain.Kind != test.wantDomain {
				t.Fatalf("domain = %+v, want %s", alloc.Domain, test.wantDomain)
			}
			if len(alloc.HeapReasonCodes) == 0 {
				t.Fatalf("heap allocation missing heap reason codes: %+v", alloc)
			}
		})
	}
}

func TestBuildKeepsUnsafeUnknownConservative(t *testing.T) {
	prog := t05SyntheticProgram("unsafe_unknown", plir.EscapeNoEscape)
	snapshot := t05SnapshotWithEscape(
		t,
		prog,
		memoryfacts.EscapeNoEscape,
		memoryfacts.ProvenanceUnsafeUnknown,
		memoryfacts.UnsafeUnknown,
	)
	plan, err := Build(Input{Program: prog, Snapshot: snapshot, Options: Options{EnableStackLowering: true}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	alloc := findAllocation(t, plan, "unsafe_unknown", "xs")
	if alloc.Storage != StorageHeap || alloc.Escape != EscapeUnknown {
		t.Fatalf("unsafe_unknown allocation = %+v, want conservative heap/unknown", alloc)
	}
}

func TestBuildRejectsStaleIslandEpochForTrustedIslandPlan(t *testing.T) {
	prog := t05SyntheticProgram("stale_island", plir.EscapeNoEscape)
	value := &prog.Funcs[0].Values[0]
	value.Provenance = plir.Provenance{Kind: plir.ProvenanceIsland, Root: "isl"}
	value.Region = "island:isl"
	for i := range prog.Funcs[0].Facts {
		prog.Funcs[0].Facts[i].IslandID = "island:isl"
		prog.Funcs[0].Facts[i].Epoch = 2
		prog.Funcs[0].Facts[i].BaseID = "alloc_intent:xs"
	}
	graph := memoryfacts.NewGraph("program:test")
	if err := graph.AdvanceTo(memoryfacts.StagePLIR); err != nil {
		t.Fatal(err)
	}
	if _, err := graph.AddFact(memoryfacts.Fact{
		ID:               "fact:stale-island",
		FunctionID:       "stale_island",
		ValueID:          "alloc_intent:xs",
		IslandID:         "island:isl",
		Epoch:            1,
		BaseID:           "alloc_intent:xs",
		SiteID:           "site:stale-island",
		SourceStage:      memoryfacts.StagePLIR,
		TypeName:         "[]u8",
		ProvenanceClass:  memoryfacts.ProvenanceSafeOwned,
		UnsafeClass:      memoryfacts.UnsafeSafe,
		EscapeState:      memoryfacts.EscapeNoEscape,
		AllocationSiteID: "core.make_u8",
		Claim:            memoryfacts.ClaimOwned,
		LifetimeBirth:    "entry",
		LifetimeDeath:    "return",
		LifetimeOwner:    "xs",
		OwnerID:          "isl",
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Build(Input{Program: prog, Snapshot: snapshot, Options: Options{EnableStackLowering: true}})
	if err == nil || !strings.Contains(err.Error(), "epoch") {
		t.Fatalf("Build error = %v, want stale island epoch rejection", err)
	}
}

func TestBuildPlanDigestDeterministicAndOptionSensitive(t *testing.T) {
	prog := t05SyntheticProgram("digest", plir.EscapeNoEscape)
	plan1 := t05BuildPlan(t, prog, Options{EnableStackLowering: true})
	plan2 := t05BuildPlan(t, prog, Options{EnableStackLowering: true})
	digest1 := findAllocation(t, plan1, "digest", "xs").PlanDigest
	digest2 := findAllocation(t, plan2, "digest", "xs").PlanDigest
	if digest1 == "" || digest1 != digest2 {
		t.Fatalf("PlanDigest = %q/%q, want stable non-empty digest", digest1, digest2)
	}
	changed := t05BuildPlan(t, prog, Options{EnableStackLowering: false})
	changedDigest := findAllocation(t, changed, "digest", "xs").PlanDigest
	if changedDigest == "" || changedDigest == digest1 {
		t.Fatalf("PlanDigest ignored options: %q vs %q", digest1, changedDigest)
	}
}

func t05BuildPlan(t *testing.T, prog *plir.Program, opt Options) *Plan {
	t.Helper()
	graph, err := fromplir.Build("program:test", prog)
	if err != nil {
		t.Fatalf("fromplir.Build: %v", err)
	}
	if err := graph.AdvanceTo(memoryfacts.StagePLIR); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Build(Input{Program: prog, Snapshot: snapshot, Options: opt})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return plan
}

func t05SnapshotWithEscape(
	t *testing.T,
	prog *plir.Program,
	escape memoryfacts.EscapeState,
	provenance memoryfacts.ProvenanceClass,
	unsafe memoryfacts.UnsafeClass,
) memoryfacts.Snapshot {
	t.Helper()
	graph := memoryfacts.NewGraph("program:test")
	if err := graph.AdvanceTo(memoryfacts.StagePLIR); err != nil {
		t.Fatal(err)
	}
	if _, err := graph.AddFact(memoryfacts.Fact{
		ID:               memoryfacts.FactID("fact:typed:" + string(escape)),
		FunctionID:       prog.Funcs[0].Name,
		ValueID:          "alloc_intent:xs",
		SiteID:           "site:typed:" + string(escape),
		SourceStage:      memoryfacts.StagePLIR,
		TypeName:         "[]u8",
		ProvenanceClass:  provenance,
		UnsafeClass:      unsafe,
		EscapeState:      escape,
		AllocationSiteID: "core.make_u8",
		Claim:            memoryfacts.ClaimOwned,
		LifetimeBirth:    "entry",
		LifetimeDeath:    "return",
		LifetimeOwner:    "xs",
		OwnerID:          "xs",
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func t05SyntheticProgram(name string, escape plir.EscapeState) *plir.Program {
	return &plir.Program{Funcs: []plir.Function{{
		Name: name,
		Values: []plir.Value{{
			ID:     "alloc_intent:xs",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Source: "test:1",
			Region: "fn:" + name,
			Alloc: &plir.AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "4",
				LengthConstKnown:    true,
				LengthConst:         4,
				ZeroGuardStatus:     "valid_empty_no_allocator",
				NegativeGuardStatus: "reject_before_allocation",
				OverflowGuardStatus: "reject_before_allocation",
				Builtin:             "core.make_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "xs"},
			Lifetime: plir.Lifetime{
				Birth: "entry",
				Death: "return",
				Owner: "xs",
			},
			Escape: escape,
		}},
		Facts: []plir.Fact{{
			ID:      "fact:owned",
			Kind:    plir.FactOwned,
			ValueID: "alloc_intent:xs",
			Source:  "test:1",
		}, {
			ID:      "fact:noescape",
			Kind:    plir.FactNoEscape,
			ValueID: "alloc_intent:xs",
			Source:  "test:1",
		}},
		Blocks: []plir.BasicBlock{{
			ID:    "entry",
			Kind:  "entry",
			Entry: true,
			Exit:  true,
		}},
	}}}
}
