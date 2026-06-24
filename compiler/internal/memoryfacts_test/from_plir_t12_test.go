package memoryfacts_test

import (
	"testing"

	. "tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
)

func TestT12FromPLIREmitsTypedActorMoveDomainEvidence(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "sendOwned",
		Values: []plir.Value{{
			ID:     "alloc_intent:xs",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Source: "actor.tetra:4:13",
			Region: "allocation:xs",
			Alloc: &plir.AllocIntent{
				ElementType:      "u8",
				ElementSize:      1,
				LengthExpr:       "4",
				LengthConstKnown: true,
				LengthConst:      4,
				Builtin:          "core.make_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "xs"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "send", Owner: "xs"},
			Escape:     plir.EscapeNoEscape,
		}},
		Ops: []plir.Operation{{
			ID:     "op_send",
			Kind:   plir.OpActorSend,
			Source: "actor.tetra:5:5",
			Inputs: []string{"actor:worker", "xs"},
			Note:   "core.send_typed typed actor ownership transfer",
		}},
		Facts: []plir.Fact{{
			ID:      "f_owned",
			Kind:    plir.FactOwned,
			ValueID: "alloc_intent:xs",
			Source:  "actor.tetra:4:13",
			Reason:  "owned allocation",
		}, {
			ID:      "f_moved",
			Kind:    plir.FactMoved,
			ValueID: "alloc_intent:xs",
			Source:  "actor.tetra:5:5",
			Reason:  "typed actor ownership transfer moved payload",
		}},
	}}}

	graph, err := BuildGraphFromPLIRAndPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("BuildGraphFromPLIRAndPlan: %v", err)
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	evidence, err := snapshot.ResolveAllocation(ValueKey{
		FunctionID: "sendOwned",
		ValueID:    "alloc_intent:xs",
	})
	if err != nil {
		t.Fatalf("ResolveAllocation: %v", err)
	}
	if evidence.EscapeState != EscapeActor {
		t.Fatalf("escape = %q, want actor", evidence.EscapeState)
	}
	if evidence.DomainKind != DomainActor || evidence.DomainOwnerID != "actor:worker" {
		t.Fatalf("domain evidence = %s/%q, want actor/worker", evidence.DomainKind, evidence.DomainOwnerID)
	}
	if evidence.TransferKind != TransferMove || evidence.TransferProofID == "" ||
		!evidence.SourceConsumed || evidence.LiveBorrowCrossing ||
		!evidence.DestinationActive {
		t.Fatalf("transfer evidence = %+v, want proof-carrying consumed actor move", evidence)
	}
}
