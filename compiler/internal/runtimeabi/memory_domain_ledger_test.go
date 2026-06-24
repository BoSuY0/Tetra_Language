package runtimeabi

import (
	"strings"
	"sync"
	"testing"
)

func TestMemoryDomainConstructorsStartActiveWithExplicitBudget(t *testing.T) {
	process := DefaultProcessMemoryDomain(17, 32)
	if process.DomainID != "domain:process" || process.ParentDomainID != "" ||
		process.Kind != DomainProcess || process.State != DomainStateActive {
		t.Fatalf("process domain identity/state = %+v, want active root process", process)
	}
	if process.BudgetBytes != 0 || process.RequestedBytes != 0 || process.ReservedBytes != 0 {
		t.Fatalf("process counters = %+v, want zero counters and explicit budget only", process)
	}

	task := TaskMemoryDomain("task:build", "domain:process", "task:build:lifetime", 4096)
	if task.DomainID != "domain:task:build" || task.ParentDomainID != "domain:process" ||
		task.Kind != DomainTask || task.OwnerKind != "task" || task.OwnerID != "build" ||
		task.Lifetime != "task:build:lifetime" || task.BudgetBytes != 4096 ||
		task.State != DomainStateActive {
		t.Fatalf("task domain = %+v, want active task child with explicit budget", task)
	}

	actor := ActorMemoryDomain("actor:mailbox", "domain:process", "actor:lifetime", 2048)
	if actor.DomainID != "domain:actor:mailbox" || actor.Kind != DomainActor ||
		actor.OwnerKind != "actor" || actor.OwnerID != "mailbox" || actor.BudgetBytes != 2048 {
		t.Fatalf("actor domain = %+v, want active actor child", actor)
	}

	request := RequestMemoryDomain("request:42", task.DomainID, "request:42", 512)
	if request.DomainID != "domain:request:42" || request.ParentDomainID != task.DomainID ||
		request.Kind != DomainRequest || request.OwnerKind != "request" ||
		request.OwnerID != "42" || request.BudgetBytes != 512 {
		t.Fatalf("request domain = %+v, want active request child", request)
	}
}

func TestMemoryDomainLedgerAccountingAndSnapshot(t *testing.T) {
	ledger := newLedgerForTest(t)
	task := TaskMemoryDomain("task:build", "domain:process", "task:build", 10)
	if err := ledger.Register(task); err != nil {
		t.Fatalf("Register(task): %v", err)
	}
	for _, event := range []MemoryDomainEvent{
		{Kind: DomainEventRequest, DomainID: task.DomainID, Bytes: 6},
		{Kind: DomainEventReserve, DomainID: task.DomainID, ReservationBytes: 8},
		{Kind: DomainEventCommit, DomainID: task.DomainID, CommitBytes: 6},
		{Kind: DomainEventAllocate, DomainID: task.DomainID, Bytes: 4},
	} {
		if err := ledger.Apply(event); err != nil {
			t.Fatalf("Apply(%s): %v", event.Kind, err)
		}
	}

	snapshot := ledger.Snapshot()
	if len(snapshot) != 2 || snapshot[0].DomainID != "domain:process" ||
		snapshot[1].DomainID != task.DomainID {
		t.Fatalf("snapshot = %+v, want sorted process/task domains", snapshot)
	}
	got := snapshot[1]
	if got.RequestedBytes != 6 || got.ReservedBytes != 8 || got.CommittedBytes != 6 ||
		got.CurrentBytes != 4 || got.PeakBytes != 4 || got.State != DomainStateActive {
		t.Fatalf("task accounting = %+v, want request/reserve/commit/current/peak", got)
	}

	snapshot[1].CurrentBytes = 99
	fresh := ledger.Snapshot()[1]
	if fresh.CurrentBytes != 4 {
		t.Fatalf("snapshot mutation changed ledger: %+v", fresh)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventAllocate, DomainID: task.DomainID, Bytes: 7}); err == nil ||
		!strings.Contains(err.Error(), "memory domain ledger: budget exceeded") {
		t.Fatalf("budget error = %v, want budget exceeded", err)
	}
}

func TestMemoryDomainLedgerLifecycleMoveCopyAndClose(t *testing.T) {
	ledger := newLedgerForTest(t)
	island := IslandMemoryDomain("island:isl", "island:isl:scope", 0, 0)
	island.ParentDomainID = "domain:process"
	island.BudgetBytes = 64
	task := TaskMemoryDomain("task:dst", "domain:process", "task:dst", 64)
	for _, domain := range []MemoryDomain{island, task} {
		if err := ledger.Register(domain); err != nil {
			t.Fatalf("Register(%s): %v", domain.DomainID, err)
		}
	}
	primeDomainForCurrentBytes(t, ledger, island.DomainID, 16)
	primeDomainForCurrentBytes(t, ledger, task.DomainID, 16)
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventReserve, DomainID: task.DomainID, Bytes: 4}); err != nil {
		t.Fatalf("reserve task move capacity: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventCommit, DomainID: task.DomainID, Bytes: 4}); err != nil {
		t.Fatalf("commit task move capacity: %v", err)
	}

	if err := ledger.Apply(MemoryDomainEvent{
		Kind:          DomainEventMove,
		DomainID:      island.DomainID,
		DestinationID: task.DomainID,
		Bytes:         4,
	}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{
		Kind:          DomainEventCopy,
		DomainID:      task.DomainID,
		DestinationID: island.DomainID,
		Bytes:         3,
	}); err != nil {
		t.Fatalf("copy: %v", err)
	}
	byID := snapshotByID(ledger.Snapshot())
	if byID[island.DomainID].CurrentBytes != 15 || byID[task.DomainID].CurrentBytes != 20 {
		t.Fatalf("move/copy current bytes = island %+v task %+v", byID[island.DomainID], byID[task.DomainID])
	}
	if byID[island.DomainID].CopyCount != 1 || byID[island.DomainID].BytesCopied != 3 {
		t.Fatalf("copy counters = %+v, want destination-side copy accounting", byID[island.DomainID])
	}
	if byID[task.DomainID].CopyCount != 0 || byID[task.DomainID].BytesCopied != 0 {
		t.Fatalf("source copy counters changed on move/copy: %+v", byID[task.DomainID])
	}

	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventReset, DomainID: island.DomainID}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	afterReset := snapshotByID(ledger.Snapshot())[island.DomainID]
	if afterReset.CurrentBytes != 0 || afterReset.Epoch != 1 ||
		afterReset.ReservedBytes == 0 || afterReset.CommittedBytes == 0 {
		t.Fatalf("reset domain = %+v, want current zero, epoch increment, backend bytes preserved", afterReset)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventClose, DomainID: island.DomainID}); err == nil ||
		!strings.Contains(err.Error(), "memory domain ledger: accounting invariant") {
		t.Fatalf("close with reserved bytes error = %v, want accounting invariant", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventDecommit, DomainID: island.DomainID, Bytes: afterReset.CommittedBytes}); err != nil {
		t.Fatalf("decommit: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventRelease, DomainID: island.DomainID, Bytes: afterReset.ReservedBytes}); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventClose, DomainID: island.DomainID}); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventClose, DomainID: island.DomainID}); err == nil ||
		!strings.Contains(err.Error(), "memory domain ledger: domain closed") {
		t.Fatalf("second close error = %v, want domain closed", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{
		Kind:          DomainEventCopy,
		DomainID:      task.DomainID,
		DestinationID: island.DomainID,
		Bytes:         1,
	}); err == nil || !strings.Contains(err.Error(), "memory domain ledger: domain closed") {
		t.Fatalf("copy into closed domain error = %v, want domain closed", err)
	}
}

func TestMemoryDomainLedgerRejectsInvalidEventsAndGraphs(t *testing.T) {
	ledger := newLedgerForTest(t)
	missingParent := TaskMemoryDomain("task:orphan", "domain:missing", "task:orphan", 0)
	if err := ledger.Register(missingParent); err != nil {
		t.Fatalf("Register(orphan): %v", err)
	}
	if err := ledger.Validate(); err == nil ||
		!strings.Contains(err.Error(), "memory domain ledger: missing parent") {
		t.Fatalf("missing parent validation = %v, want missing parent", err)
	}

	cycleLedger := newLedgerForTest(t)
	a := TaskMemoryDomain("task:a", "domain:task:b", "task:a", 0)
	b := TaskMemoryDomain("task:b", a.DomainID, "task:b", 0)
	if err := cycleLedger.Register(a); err != nil {
		t.Fatalf("Register(a): %v", err)
	}
	if err := cycleLedger.Register(b); err != nil {
		t.Fatalf("Register(b): %v", err)
	}
	if err := cycleLedger.Validate(); err == nil ||
		!strings.Contains(err.Error(), "memory domain ledger: cycle") {
		t.Fatalf("cycle validation = %v, want cycle", err)
	}

	activeLedger := newLedgerForTest(t)
	task := TaskMemoryDomain("task:active", "domain:process", "task:active", 0)
	if err := activeLedger.Register(task); err != nil {
		t.Fatalf("Register(task): %v", err)
	}
	for _, test := range []struct {
		name  string
		event MemoryDomainEvent
		want  string
	}{
		{name: "unknown", event: MemoryDomainEvent{Kind: "unknown", DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "negative", event: MemoryDomainEvent{Kind: DomainEventRequest, DomainID: task.DomainID, Bytes: -1}, want: "memory domain ledger: accounting invariant"},
		{name: "zero request", event: MemoryDomainEvent{Kind: DomainEventRequest, DomainID: task.DomainID}, want: "memory domain ledger: accounting invariant"},
		{name: "commit without reserve", event: MemoryDomainEvent{Kind: DomainEventCommit, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "allocate without commit", event: MemoryDomainEvent{Kind: DomainEventAllocate, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "free too much", event: MemoryDomainEvent{Kind: DomainEventFree, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "decommit too much", event: MemoryDomainEvent{Kind: DomainEventDecommit, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "release too much", event: MemoryDomainEvent{Kind: DomainEventRelease, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "missing destination", event: MemoryDomainEvent{Kind: DomainEventMove, DomainID: task.DomainID, Bytes: 1}, want: "memory domain ledger: missing parent"},
		{name: "move too much", event: MemoryDomainEvent{Kind: DomainEventMove, DomainID: task.DomainID, DestinationID: "domain:process", Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "copy too much", event: MemoryDomainEvent{Kind: DomainEventCopy, DomainID: task.DomainID, DestinationID: "domain:process", Bytes: 1}, want: "memory domain ledger: accounting invariant"},
		{name: "reset non island", event: MemoryDomainEvent{Kind: DomainEventReset, DomainID: task.DomainID}, want: "memory domain ledger: accounting invariant"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := activeLedger.Apply(test.event); err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf("Apply(%s) error = %v, want %q", test.name, err, test.want)
			}
		})
	}
}

func TestMemoryDomainLedgerConcurrentSnapshotAndEvents(t *testing.T) {
	ledger := newLedgerForTest(t)
	task := TaskMemoryDomain("task:concurrent", "domain:process", "task:concurrent", 0)
	if err := ledger.Register(task); err != nil {
		t.Fatalf("Register(task): %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventReserve, DomainID: task.DomainID, Bytes: 1024}); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventCommit, DomainID: task.DomainID, Bytes: 1024}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				if err := ledger.Apply(MemoryDomainEvent{Kind: DomainEventAllocate, DomainID: task.DomainID, Bytes: 1}); err != nil {
					t.Errorf("allocate: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				_ = ledger.Snapshot()
			}
		}()
	}
	wg.Wait()
	if err := ledger.Validate(); err != nil {
		t.Fatalf("Validate after concurrent access: %v", err)
	}
}

func newLedgerForTest(t *testing.T) *MemoryDomainLedger {
	t.Helper()
	ledger, err := NewMemoryDomainLedger(DefaultProcessMemoryDomain(0, 0))
	if err != nil {
		t.Fatalf("NewMemoryDomainLedger: %v", err)
	}
	return ledger
}

func primeDomainForCurrentBytes(t *testing.T, ledger *MemoryDomainLedger, domainID string, bytes int64) {
	t.Helper()
	for _, event := range []MemoryDomainEvent{
		{Kind: DomainEventRequest, DomainID: domainID, Bytes: bytes},
		{Kind: DomainEventReserve, DomainID: domainID, Bytes: bytes},
		{Kind: DomainEventCommit, DomainID: domainID, Bytes: bytes},
		{Kind: DomainEventAllocate, DomainID: domainID, Bytes: bytes},
	} {
		if err := ledger.Apply(event); err != nil {
			t.Fatalf("prime %s with %s: %v", domainID, event.Kind, err)
		}
	}
}

func snapshotByID(domains []MemoryDomain) map[string]MemoryDomain {
	out := map[string]MemoryDomain{}
	for _, domain := range domains {
		out[domain.DomainID] = domain
	}
	return out
}
