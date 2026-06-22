package runtimeabi

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type MemoryDomainEventKind string

const (
	DomainEventRequest  MemoryDomainEventKind = "request"
	DomainEventReserve  MemoryDomainEventKind = "reserve"
	DomainEventCommit   MemoryDomainEventKind = "commit"
	DomainEventAllocate MemoryDomainEventKind = "allocate"
	DomainEventFree     MemoryDomainEventKind = "free"
	DomainEventDecommit MemoryDomainEventKind = "decommit"
	DomainEventRelease  MemoryDomainEventKind = "release"
	DomainEventTrim     MemoryDomainEventKind = "trim"
	DomainEventReset    MemoryDomainEventKind = "reset"
	DomainEventClose    MemoryDomainEventKind = "close"
	DomainEventCopy     MemoryDomainEventKind = "copy"
	DomainEventMove     MemoryDomainEventKind = "move"
)

type MemoryDomainEvent struct {
	Kind             MemoryDomainEventKind
	DomainID         string
	DestinationID    string
	Bytes            int64
	ReservationBytes int64
	CommitBytes      int64
	ReasonCode       string
}

type MemoryDomainLedger struct {
	mu      sync.RWMutex
	domains map[string]MemoryDomain
}

func NewMemoryDomainLedger(process MemoryDomain) (*MemoryDomainLedger, error) {
	process = normalizeLedgerDomain(process)
	if process.DomainID != "domain:process" || process.Kind != DomainProcess {
		return nil, fmt.Errorf("memory domain ledger: accounting invariant: process root is required")
	}
	if strings.TrimSpace(process.ParentDomainID) != "" {
		return nil, fmt.Errorf("memory domain ledger: accounting invariant: process root has parent")
	}
	if err := validateLedgerDomain(process); err != nil {
		return nil, err
	}
	return &MemoryDomainLedger{
		domains: map[string]MemoryDomain{process.DomainID: process},
	}, nil
}

func (l *MemoryDomainLedger) Register(domain MemoryDomain) error {
	if l == nil {
		return fmt.Errorf("memory domain ledger: accounting invariant: nil ledger")
	}
	domain = normalizeLedgerDomain(domain)
	if err := validateLedgerDomain(domain); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.domains[domain.DomainID]; exists {
		return fmt.Errorf("memory domain ledger: accounting invariant: duplicate domain %s", domain.DomainID)
	}
	l.domains[domain.DomainID] = domain
	return nil
}

func (l *MemoryDomainLedger) Apply(event MemoryDomainEvent) error {
	if l == nil {
		return fmt.Errorf("memory domain ledger: accounting invariant: nil ledger")
	}
	if err := validateEventShape(event); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	domainID := cleanEventDomainID(event.DomainID)
	domain, ok := l.domains[domainID]
	if !ok {
		return fmt.Errorf("memory domain ledger: missing parent: domain %s", domainID)
	}
	domain = normalizeLedgerDomain(domain)
	if domain.State == DomainStateClosed {
		return fmt.Errorf("memory domain ledger: domain closed: %s", domainID)
	}
	switch event.Kind {
	case DomainEventRequest:
		domain.RequestedBytes += event.Bytes
		return l.storeDomain(domain)
	case DomainEventReserve:
		domain.ReservedBytes += eventReservationBytes(event)
		return l.storeDomain(domain)
	case DomainEventCommit:
		domain.CommittedBytes += eventCommitBytes(event)
		return l.storeDomain(domain)
	case DomainEventAllocate:
		domain.CurrentBytes += event.Bytes
		if domain.CurrentBytes > domain.PeakBytes {
			domain.PeakBytes = domain.CurrentBytes
		}
		return l.storeDomain(domain)
	case DomainEventFree:
		if event.Bytes > domain.CurrentBytes {
			return fmt.Errorf("memory domain ledger: accounting invariant: free exceeds current bytes")
		}
		domain.CurrentBytes -= event.Bytes
		return l.storeDomain(domain)
	case DomainEventDecommit:
		amount := eventCommitBytes(event)
		if amount > domain.CommittedBytes-domain.CurrentBytes {
			return fmt.Errorf("memory domain ledger: accounting invariant: decommit exceeds idle committed bytes")
		}
		domain.CommittedBytes -= amount
		domain.DecommittedBytes += amount
		return l.storeDomain(domain)
	case DomainEventRelease, DomainEventTrim:
		amount := eventReservationBytes(event)
		if amount > domain.ReservedBytes-domain.CommittedBytes {
			return fmt.Errorf("memory domain ledger: accounting invariant: release exceeds idle reserved bytes")
		}
		domain.ReservedBytes -= amount
		domain.ReleasedBytes += amount
		return l.storeDomain(domain)
	case DomainEventReset:
		if domain.Kind != DomainIsland {
			return fmt.Errorf("memory domain ledger: accounting invariant: reset requires island domain")
		}
		domain.CurrentBytes = 0
		domain.Epoch++
		return l.storeDomain(domain)
	case DomainEventClose:
		if domain.CurrentBytes != 0 || domain.CommittedBytes != 0 || domain.ReservedBytes != 0 {
			return fmt.Errorf("memory domain ledger: accounting invariant: close requires zero live backend bytes")
		}
		domain.State = DomainStateClosed
		return l.storeDomain(domain)
	case DomainEventMove:
		return l.applyMove(domain, event)
	case DomainEventCopy:
		return l.applyCopy(domain, event)
	default:
		return fmt.Errorf("memory domain ledger: accounting invariant: unknown event kind %q", event.Kind)
	}
}

func (l *MemoryDomainLedger) Snapshot() []MemoryDomain {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	keys := make([]string, 0, len(l.domains))
	for key := range l.domains {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]MemoryDomain, 0, len(keys))
	for _, key := range keys {
		out = append(out, l.domains[key])
	}
	return out
}

func (l *MemoryDomainLedger) Validate() error {
	if l == nil {
		return fmt.Errorf("memory domain ledger: accounting invariant: nil ledger")
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.validateLocked()
}

func (l *MemoryDomainLedger) storeDomain(domain MemoryDomain) error {
	if err := validateLedgerDomain(domain); err != nil {
		return err
	}
	l.domains[domain.DomainID] = domain
	return nil
}

func (l *MemoryDomainLedger) applyMove(source MemoryDomain, event MemoryDomainEvent) error {
	destination, err := l.activeDestination(event.DestinationID)
	if err != nil {
		return err
	}
	if event.Bytes > source.CurrentBytes {
		return fmt.Errorf("memory domain ledger: accounting invariant: move exceeds source current bytes")
	}
	source.CurrentBytes -= event.Bytes
	destination.CurrentBytes += event.Bytes
	if destination.CurrentBytes > destination.PeakBytes {
		destination.PeakBytes = destination.CurrentBytes
	}
	if err := validateLedgerDomain(source); err != nil {
		return err
	}
	if err := validateLedgerDomain(destination); err != nil {
		return err
	}
	l.domains[source.DomainID] = source
	l.domains[destination.DomainID] = destination
	return nil
}

func (l *MemoryDomainLedger) applyCopy(source MemoryDomain, event MemoryDomainEvent) error {
	destination, err := l.activeDestination(event.DestinationID)
	if err != nil {
		return err
	}
	if event.Bytes > source.CurrentBytes {
		return fmt.Errorf("memory domain ledger: accounting invariant: copy exceeds source current bytes")
	}
	destination.RequestedBytes += event.Bytes
	destination.CurrentBytes += event.Bytes
	if destination.CurrentBytes > destination.PeakBytes {
		destination.PeakBytes = destination.CurrentBytes
	}
	destination.CopyCount++
	destination.BytesCopied += event.Bytes
	if err := validateLedgerDomain(source); err != nil {
		return err
	}
	if err := validateLedgerDomain(destination); err != nil {
		return err
	}
	l.domains[source.DomainID] = source
	l.domains[destination.DomainID] = destination
	return nil
}

func (l *MemoryDomainLedger) activeDestination(rawID string) (MemoryDomain, error) {
	destinationID := cleanEventDomainID(rawID)
	destination, ok := l.domains[destinationID]
	if !ok {
		return MemoryDomain{}, fmt.Errorf("memory domain ledger: missing parent: destination %s", destinationID)
	}
	destination = normalizeLedgerDomain(destination)
	if destination.State == DomainStateClosed {
		return MemoryDomain{}, fmt.Errorf("memory domain ledger: domain closed: %s", destinationID)
	}
	return destination, nil
}

func (l *MemoryDomainLedger) validateLocked() error {
	if _, ok := l.domains["domain:process"]; !ok {
		return fmt.Errorf("memory domain ledger: missing parent: domain:process")
	}
	for _, domain := range l.domains {
		domain = normalizeLedgerDomain(domain)
		if err := validateLedgerDomain(domain); err != nil {
			return err
		}
		if domain.Kind == DomainProcess || domain.DomainID == "domain:process" {
			if strings.TrimSpace(domain.ParentDomainID) != "" {
				return fmt.Errorf("memory domain ledger: accounting invariant: process root has parent")
			}
			continue
		}
		if strings.TrimSpace(domain.ParentDomainID) == "" {
			return fmt.Errorf("memory domain ledger: missing parent: %s", domain.DomainID)
		}
		if _, ok := l.domains[domain.ParentDomainID]; !ok {
			return fmt.Errorf("memory domain ledger: missing parent: %s -> %s", domain.DomainID, domain.ParentDomainID)
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("memory domain ledger: cycle: %s", id)
		}
		visiting[id] = true
		parent := l.domains[id].ParentDomainID
		if parent != "" {
			if err := visit(parent); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	keys := make([]string, 0, len(l.domains))
	for id := range l.domains {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func normalizeLedgerDomain(domain MemoryDomain) MemoryDomain {
	domain.DomainID = cleanEventDomainID(domain.DomainID)
	if strings.TrimSpace(domain.ParentDomainID) != "" {
		domain.ParentDomainID = cleanEventDomainID(domain.ParentDomainID)
	}
	if domain.State == "" {
		domain.State = DomainStateActive
	}
	return domain
}

func validateLedgerDomain(domain MemoryDomain) error {
	if err := ValidateMemoryDomain(domain); err != nil {
		if strings.Contains(err.Error(), "budget exceeded") {
			return fmt.Errorf("memory domain ledger: budget exceeded: %w", err)
		}
		return fmt.Errorf("memory domain ledger: accounting invariant: %w", err)
	}
	if domain.State == DomainStateClosed &&
		(domain.CurrentBytes != 0 || domain.CommittedBytes != 0 || domain.ReservedBytes != 0) {
		return fmt.Errorf("memory domain ledger: accounting invariant: closed domain has live backend bytes")
	}
	return nil
}

func validateEventShape(event MemoryDomainEvent) error {
	if !knownMemoryDomainEventKind(event.Kind) {
		return fmt.Errorf("memory domain ledger: accounting invariant: unknown event kind %q", event.Kind)
	}
	if strings.TrimSpace(event.DomainID) == "" {
		return fmt.Errorf("memory domain ledger: missing parent: domain_id is required")
	}
	if event.Bytes < 0 || event.ReservationBytes < 0 || event.CommitBytes < 0 {
		return fmt.Errorf("memory domain ledger: accounting invariant: event bytes must not be negative")
	}
	if event.Kind != DomainEventClose && event.Kind != DomainEventReset && eventAmount(event) <= 0 {
		return fmt.Errorf("memory domain ledger: accounting invariant: event bytes are required")
	}
	return nil
}

func knownMemoryDomainEventKind(kind MemoryDomainEventKind) bool {
	switch kind {
	case DomainEventRequest,
		DomainEventReserve,
		DomainEventCommit,
		DomainEventAllocate,
		DomainEventFree,
		DomainEventDecommit,
		DomainEventRelease,
		DomainEventTrim,
		DomainEventReset,
		DomainEventClose,
		DomainEventCopy,
		DomainEventMove:
		return true
	default:
		return false
	}
}

func eventAmount(event MemoryDomainEvent) int64 {
	switch event.Kind {
	case DomainEventReserve, DomainEventRelease, DomainEventTrim:
		return eventReservationBytes(event)
	case DomainEventCommit, DomainEventDecommit:
		return eventCommitBytes(event)
	case DomainEventClose, DomainEventReset:
		return 0
	default:
		return event.Bytes
	}
}

func eventReservationBytes(event MemoryDomainEvent) int64 {
	if event.ReservationBytes > 0 {
		return event.ReservationBytes
	}
	return event.Bytes
}

func eventCommitBytes(event MemoryDomainEvent) int64 {
	if event.CommitBytes > 0 {
		return event.CommitBytes
	}
	return event.Bytes
}

func cleanEventDomainID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return cleanDomainID(value)
}
