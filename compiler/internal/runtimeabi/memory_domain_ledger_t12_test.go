package runtimeabi

import (
	"strings"
	"testing"
)

func TestT12TaskDomainCloseRequiresCancellationCleanupToZeroBytes(t *testing.T) {
	ledger := newLedgerForTest(t)
	task := TaskMemoryDomain("task:worker", "domain:process", "task:worker", 0)
	if err := ledger.Register(task); err != nil {
		t.Fatalf("Register(task): %v", err)
	}
	primeDomainForCurrentBytes(t, ledger, task.DomainID, 16)
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventClose, DomainID: task.DomainID}); err == nil ||
		!strings.Contains(err.Error(), "close requires zero live backend bytes") {
		t.Fatalf("close live task domain error = %v, want zero-live-byte rejection", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventFree, DomainID: task.DomainID, Bytes: 16}); err != nil {
		t.Fatalf("free task current bytes: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventDecommit, DomainID: task.DomainID, Bytes: 16}); err != nil {
		t.Fatalf("decommit task bytes: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventRelease, DomainID: task.DomainID, Bytes: 16}); err != nil {
		t.Fatalf("release task bytes: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventClose, DomainID: task.DomainID}); err != nil {
		t.Fatalf("close cleaned task domain: %v", err)
	}
}

func TestT12ActorCopyAccountingKeepsActorDomainIdentityStable(t *testing.T) {
	ledger := newLedgerForTest(t)
	task := TaskMemoryDomain("task:source", "domain:process", "task:source", 0)
	actor := ActorMemoryDomain("actor:mailbox", "domain:process", "actor:mailbox", 0)
	for _, domain := range []MemoryDomain{task, actor} {
		if err := ledger.Register(domain); err != nil {
			t.Fatalf("Register(%s): %v", domain.DomainID, err)
		}
	}
	primeDomainForCurrentBytes(t, ledger, task.DomainID, 12)
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventReserve, DomainID: actor.DomainID, Bytes: 5}); err != nil {
		t.Fatalf("reserve actor copy capacity: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventCommit, DomainID: actor.DomainID, Bytes: 5}); err != nil {
		t.Fatalf("commit actor copy capacity: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{
		Kind:          DomainEventCopy,
		DomainID:      task.DomainID,
		DestinationID: actor.DomainID,
		Bytes:         5,
	}); err != nil {
		t.Fatalf("actor message copy: %v", err)
	}
	byID := snapshotByID(ledger.Snapshot())
	got := byID[actor.DomainID]
	if got.DomainID != actor.DomainID || got.OwnerID != actor.OwnerID ||
		got.Kind != DomainActor {
		t.Fatalf("actor identity changed after copy: before %+v after %+v", actor, got)
	}
	if got.CopyCount != 1 || got.BytesCopied != 5 || got.CurrentBytes != 5 {
		t.Fatalf("actor copy counters = %+v, want one 5-byte destination copy", got)
	}
	if source := byID[task.DomainID]; source.CurrentBytes != 12 ||
		source.CopyCount != 0 || source.BytesCopied != 0 {
		t.Fatalf("source task changed by copy accounting: %+v", source)
	}
}
