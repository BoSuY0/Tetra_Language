package allocplan

import (
	"testing"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

func TestT12ProofCarryingActorAndTaskMovesPlanBoundaryDomains(t *testing.T) {
	for _, test := range []struct {
		name        string
		domainKind  memoryfacts.DomainKind
		transfer    memoryfacts.TransferKind
		escape      memoryfacts.EscapeState
		wantStorage StorageClass
		wantDomain  runtimeabi.MemoryDomainKind
		wantOwner   string
	}{
		{
			name:        "actor",
			domainKind:  memoryfacts.DomainActor,
			transfer:    memoryfacts.TransferMove,
			escape:      memoryfacts.EscapeActor,
			wantStorage: StorageActorMoveRegion,
			wantDomain:  runtimeabi.DomainActor,
			wantOwner:   "worker",
		},
		{
			name:        "task",
			domainKind:  memoryfacts.DomainTask,
			transfer:    memoryfacts.TransferMove,
			escape:      memoryfacts.EscapeTask,
			wantStorage: StorageTaskRegion,
			wantDomain:  runtimeabi.DomainTask,
			wantOwner:   "worker",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := t05SyntheticProgram(test.name, plir.EscapeNoEscape)
			snapshot := t12SnapshotWithDomainTransfer(t, prog, t12TransferFixture{
				DomainKind:         test.domainKind,
				DomainOwnerID:      test.name + ":worker",
				TransferKind:       test.transfer,
				Escape:             test.escape,
				TransferProofID:    "proof:domain_move:" + test.name + ":xs",
				SourceConsumed:     true,
				LiveBorrowCrossing: false,
				DestinationActive:  true,
			})

			plan, err := Build(Input{
				Program:  prog,
				Snapshot: snapshot,
				Options:  Options{EnableStackLowering: true},
			})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			alloc := findAllocation(t, plan, test.name, "xs")
			if alloc.Storage != test.wantStorage || alloc.PlannedStorage != test.wantStorage {
				t.Fatalf("boundary storage = %s/%s, want %s: %+v", alloc.Storage, alloc.PlannedStorage, test.wantStorage, alloc)
			}
			if alloc.ActualLoweringStorage != StorageUnknownConservative ||
				alloc.LoweringStatus != "pending" {
				t.Fatalf("actual/lowering = %s/%s, want pending split: %+v", alloc.ActualLoweringStorage, alloc.LoweringStatus, alloc)
			}
			if alloc.Domain == nil || alloc.Domain.Kind != test.wantDomain ||
				alloc.Domain.OwnerID != test.wantOwner {
				t.Fatalf("domain = %+v, want %s owner %q", alloc.Domain, test.wantDomain, test.wantOwner)
			}
			if !containsString(alloc.ProofIDs, "proof:domain_move:"+test.name+":xs") {
				t.Fatalf("proof ids = %v, want domain move proof", alloc.ProofIDs)
			}
			if len(alloc.HeapReasonCodes) != 0 {
				t.Fatalf("trusted boundary storage should not carry heap reason codes: %+v", alloc)
			}
		})
	}
}

func TestT12UnprovenBoundaryAndRequestOwnershipFallBackWithExactReasons(t *testing.T) {
	tests := []struct {
		name       string
		fixture    t12TransferFixture
		wantReason string
	}{
		{
			name: "actor_move_unproven",
			fixture: t12TransferFixture{
				DomainKind:         memoryfacts.DomainActor,
				DomainOwnerID:      "actor:worker",
				TransferKind:       memoryfacts.TransferMove,
				Escape:             memoryfacts.EscapeActor,
				SourceConsumed:     true,
				LiveBorrowCrossing: false,
				DestinationActive:  true,
			},
			wantReason: HeapReasonActorMoveUnproven,
		},
		{
			name: "task_move_unproven",
			fixture: t12TransferFixture{
				DomainKind:         memoryfacts.DomainTask,
				DomainOwnerID:      "task:worker",
				TransferKind:       memoryfacts.TransferMove,
				Escape:             memoryfacts.EscapeTask,
				SourceConsumed:     true,
				LiveBorrowCrossing: false,
				DestinationActive:  true,
			},
			wantReason: HeapReasonTaskMoveUnproven,
		},
		{
			name: "request_owner_unproven",
			fixture: t12TransferFixture{
				DomainKind:         memoryfacts.DomainRequest,
				DomainOwnerID:      "request:42",
				TransferKind:       memoryfacts.TransferMove,
				Escape:             memoryfacts.EscapeNoEscape,
				SourceConsumed:     true,
				LiveBorrowCrossing: false,
				DestinationActive:  true,
			},
			wantReason: HeapReasonRequestOwnerUnproven,
		},
		{
			name: "borrowed_actor_crossing",
			fixture: t12TransferFixture{
				DomainKind:         memoryfacts.DomainActor,
				DomainOwnerID:      "actor:worker",
				TransferKind:       memoryfacts.TransferBorrowed,
				Escape:             memoryfacts.EscapeActor,
				TransferProofID:    "proof:domain_move:actor:xs",
				SourceConsumed:     false,
				LiveBorrowCrossing: true,
				DestinationActive:  true,
			},
			wantReason: HeapReasonActorMoveUnproven,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := t05SyntheticProgram(test.name, plir.EscapeNoEscape)
			snapshot := t12SnapshotWithDomainTransfer(t, prog, test.fixture)

			plan, err := Build(Input{
				Program:  prog,
				Snapshot: snapshot,
				Options:  Options{EnableStackLowering: true},
			})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			alloc := findAllocation(t, plan, test.name, "xs")
			assertPlannedPending(t, alloc, StorageHeap)
			assertHeapReasonCode(t, alloc, test.wantReason)
			if alloc.Domain != nil && alloc.Domain.Kind != runtimeabi.DomainProcess {
				t.Fatalf("unproven boundary domain = %+v, want process/conservative", alloc.Domain)
			}
		})
	}
}

type t12TransferFixture struct {
	DomainKind         memoryfacts.DomainKind
	DomainOwnerID      string
	TransferKind       memoryfacts.TransferKind
	Escape             memoryfacts.EscapeState
	TransferProofID    string
	SourceConsumed     bool
	LiveBorrowCrossing bool
	DestinationActive  bool
}

func t12SnapshotWithDomainTransfer(
	t *testing.T,
	prog *plir.Program,
	fixture t12TransferFixture,
) memoryfacts.Snapshot {
	t.Helper()
	graph := memoryfacts.NewGraph("program:test")
	if err := graph.AdvanceTo(memoryfacts.StagePLIR); err != nil {
		t.Fatal(err)
	}
	fact := memoryfacts.Fact{
		ID:                 memoryfacts.FactID("fact:t12:" + prog.Funcs[0].Name),
		FunctionID:         prog.Funcs[0].Name,
		ValueID:            "alloc_intent:xs",
		SiteID:             "site:t12:" + prog.Funcs[0].Name,
		SourceStage:        memoryfacts.StagePLIR,
		TypeName:           "[]u8",
		ProvenanceClass:    memoryfacts.ProvenanceSafeOwned,
		UnsafeClass:        memoryfacts.UnsafeSafe,
		EscapeState:        fixture.Escape,
		AllocationSiteID:   "core.make_u8",
		Claim:              memoryfacts.ClaimOwned,
		LifetimeBirth:      "entry",
		LifetimeDeath:      "return",
		LifetimeOwner:      "xs",
		OwnerID:            "xs",
		DomainKind:         fixture.DomainKind,
		DomainOwnerID:      fixture.DomainOwnerID,
		TransferKind:       fixture.TransferKind,
		TransferProofID:    fixture.TransferProofID,
		SourceConsumed:     fixture.SourceConsumed,
		LiveBorrowCrossing: fixture.LiveBorrowCrossing,
		DestinationActive:  fixture.DestinationActive,
	}
	if fixture.TransferProofID != "" {
		fact.ProofID = fixture.TransferProofID
		fact.ProofKind = memoryfacts.ProofDomainMove
		fact.ProofSubjectBaseID = "alloc_intent:xs"
		fact.ProofOperation = "domain_move"
	}
	if _, err := graph.AddFact(fact); err != nil {
		t.Fatal(err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
