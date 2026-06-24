package actorsafety

import (
	"testing"

	"tetra_language/compiler/internal/runtimeabi"
)

func TestT12OwnershipTransferDecisionProducesLedgerMoveAndCopyEvents(t *testing.T) {
	move, err := PlanOwnershipTransfer(OwnershipTransferRequest{
		Value:               Value{Name: "owned_region", Type: "region", Kind: ValueOwnedRegion},
		Mode:                SendMove,
		SourceDomainID:      "domain:request:42",
		DestinationDomainID: "domain:actor:worker",
		Bytes:               32,
		Site:                "actor.tetra:8",
	})
	if err != nil {
		t.Fatalf("PlanOwnershipTransfer(move): %v", err)
	}
	if move.Event.Kind != runtimeabi.DomainEventMove || !move.SourceConsumed ||
		move.BytesCopied != 0 {
		t.Fatalf("move decision = %+v, want source-consuming zero-copy ledger move", move)
	}

	copyDecision, err := PlanOwnershipTransfer(OwnershipTransferRequest{
		Value:               Value{Name: "borrowed", Type: "[]u8", Kind: ValueBorrowed},
		Mode:                SendCopy,
		SourceDomainID:      "domain:request:42",
		DestinationDomainID: "domain:actor:worker",
		Bytes:               12,
		Site:                "actor.tetra:9",
	})
	if err != nil {
		t.Fatalf("PlanOwnershipTransfer(copy): %v", err)
	}
	if copyDecision.Event.Kind != runtimeabi.DomainEventCopy ||
		copyDecision.SourceConsumed || copyDecision.BytesCopied != 12 {
		t.Fatalf("copy decision = %+v, want destination copy accounting", copyDecision)
	}
}

func TestT12OwnershipTransferDecisionRejectsBorrowedBoundaryWithoutCopy(t *testing.T) {
	_, err := PlanOwnershipTransfer(OwnershipTransferRequest{
		Value:               Value{Name: "borrowed", Type: "[]u8", Kind: ValueBorrowed},
		Mode:                SendBorrowed,
		SourceDomainID:      "domain:request:42",
		DestinationDomainID: "domain:actor:worker",
		Bytes:               12,
		Site:                "actor.tetra:10",
	})
	if err == nil {
		t.Fatalf("borrowed transfer without copy unexpectedly accepted")
	}
}
