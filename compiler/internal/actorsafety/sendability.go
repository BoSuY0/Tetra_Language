package actorsafety

import (
	"fmt"

	"tetra_language/compiler/internal/islandkernel"
	"tetra_language/compiler/internal/runtimeabi"
)

type ValueKind string

const (
	ValueCopy         ValueKind = "copy"
	ValueOwned        ValueKind = "owned"
	ValueOwnedRegion  ValueKind = "owned_region"
	ValueBorrowed     ValueKind = "borrowed"
	ValueMutableAlias ValueKind = "mutable_alias"
	ValueUnsafePtr    ValueKind = "unsafe_ptr"
)

type SendMode string

const (
	SendMove     SendMode = "move"
	SendCopy     SendMode = "copy"
	SendUnsafe   SendMode = "unsafe_contract"
	SendBorrowed SendMode = "borrowed"
)

type Value struct {
	Name               string
	Type               string
	Kind               ValueKind
	UnsafeSendContract bool
}

type EventKind string

const (
	EventSend EventKind = "send"
	EventUse  EventKind = "use"
)

type Event struct {
	Kind  EventKind
	Value string
	Mode  SendMode
	Site  string
}

type OwnershipTransferRequest struct {
	Value               Value
	Mode                SendMode
	SourceDomainID      string
	DestinationDomainID string
	Bytes               int64
	Site                string
}

type OwnershipTransferDecision struct {
	Mode           SendMode
	Event          runtimeabi.MemoryDomainEvent
	SourceConsumed bool
	BytesCopied    int64
}

type Mailbox struct {
	Name         string
	Message      string
	Capacity     int
	Backpressure string
}

type Checker struct {
	values map[string]Value
	moved  map[string]string
}

func PlanOwnershipTransfer(req OwnershipTransferRequest) (OwnershipTransferDecision, error) {
	if req.Bytes <= 0 {
		return OwnershipTransferDecision{}, fmt.Errorf("actor sendability: transfer bytes must be positive")
	}
	checker := NewChecker([]Value{req.Value})
	if err := checker.Check([]Event{{
		Kind:  EventSend,
		Value: req.Value.Name,
		Mode:  req.Mode,
		Site:  req.Site,
	}}); err != nil {
		return OwnershipTransferDecision{}, err
	}
	decision := OwnershipTransferDecision{Mode: req.Mode}
	switch req.Mode {
	case SendMove:
		decision.SourceConsumed = true
		decision.Event = runtimeabi.MemoryDomainEvent{
			Kind:          runtimeabi.DomainEventMove,
			DomainID:      req.SourceDomainID,
			DestinationID: req.DestinationDomainID,
			Bytes:         req.Bytes,
			ReasonCode:    "domain.move.owned",
		}
	case SendCopy:
		decision.BytesCopied = req.Bytes
		decision.Event = runtimeabi.MemoryDomainEvent{
			Kind:          runtimeabi.DomainEventCopy,
			DomainID:      req.SourceDomainID,
			DestinationID: req.DestinationDomainID,
			Bytes:         req.Bytes,
			ReasonCode:    "domain.copy.serialized",
		}
	default:
		return OwnershipTransferDecision{}, fmt.Errorf(
			"actor sendability: %s unsupported transfer mode %q",
			site(req.Site),
			req.Mode,
		)
	}
	return decision, nil
}

func NewChecker(values []Value) Checker {
	c := Checker{values: map[string]Value{}, moved: map[string]string{}}
	for _, value := range values {
		c.values[value.Name] = value
	}
	return c
}

func (c Checker) Check(events []Event) error {
	for _, event := range events {
		value, ok := c.values[event.Value]
		if !ok {
			return fmt.Errorf("actor sendability: unknown value %q", event.Value)
		}
		if movedAt, ok := c.moved[value.Name]; ok && event.Kind != EventSend {
			if value.Kind == ValueOwnedRegion {
				return fmt.Errorf(
					"actor sendability: %s cannot use moved region after send; %q was moved at %s",
					site(event.Site),
					value.Name,
					movedAt,
				)
			}
			return fmt.Errorf(
				"actor sendability: %s uses %q after it was moved at %s",
				site(event.Site),
				value.Name,
				movedAt,
			)
		}
		switch event.Kind {
		case EventUse:
			continue
		case EventSend:
			if err := c.checkSend(value, event); err != nil {
				return err
			}
			if event.Mode == SendMove {
				c.moved[value.Name] = site(event.Site)
			}
		default:
			return fmt.Errorf("actor sendability: unknown event kind %q", event.Kind)
		}
	}
	return nil
}

func (c Checker) checkSend(value Value, event Event) error {
	switch value.Kind {
	case ValueCopy:
		if event.Mode == SendCopy {
			return nil
		}
		return fmt.Errorf(
			"actor sendability: %s small scalar value %q must cross actor boundary by copy",
			site(event.Site),
			value.Name,
		)
	case ValueOwned:
		if event.Mode == SendMove || event.Mode == SendCopy {
			return nil
		}
		return fmt.Errorf(
			"actor sendability: %s owned value %q must be moved or explicitly copied before actor send",
			site(event.Site),
			value.Name,
		)
	case ValueOwnedRegion:
		if event.Mode == SendMove {
			if decision := islandkernel.CanMoveIsland(islandMoveTokenRequest(value)); decision.Decision != islandkernel.Accept {
				return fmt.Errorf(
					"actor sendability: %s owned region %q move rejected by islandkernel: %s",
					site(event.Site),
					value.Name,
					decision.Reason.Code,
				)
			}
			return nil
		}
		return fmt.Errorf(
			"actor sendability: %s owned region %q must be moved to cross actor boundary",
			site(event.Site),
			value.Name,
		)
	case ValueBorrowed:
		if event.Mode == SendCopy {
			return nil
		}
		actorDecision := islandkernel.CanSendToActor(islandBoundaryRequest(value))
		taskDecision := islandkernel.CanSendToTask(islandBoundaryRequest(value))
		if actorDecision.Decision == islandkernel.Reject ||
			taskDecision.Decision == islandkernel.Reject {
			return fmt.Errorf(
				"actor sendability: %s cannot send borrowed view across actor boundary; use .copy() for %q",
				site(event.Site),
				value.Name,
			)
		}
		return fmt.Errorf(
			"actor sendability: %s cannot send borrowed view across actor boundary; use .copy() for %q",
			site(event.Site),
			value.Name,
		)
	case ValueMutableAlias:
		return fmt.Errorf(
			"actor sendability: %s mutable alias %q cannot cross actor boundary",
			site(event.Site),
			value.Name,
		)
	case ValueUnsafePtr:
		if event.Mode == SendUnsafe && value.UnsafeSendContract {
			return nil
		}
		return fmt.Errorf(
			"actor sendability: %s cannot send unknown unsafe provenance without audited contract for %q",
			site(event.Site),
			value.Name,
		)
	default:
		return fmt.Errorf(
			"actor sendability: %s value %q has unknown sendability kind %q",
			site(event.Site),
			value.Name,
			value.Kind,
		)
	}
}

func islandMoveTokenRequest(value Value) islandkernel.TokenRequest {
	return islandkernel.TokenRequest{
		Token: islandkernel.Token{
			IslandID: "island:" + value.Name,
			Epoch:    0,
			OwnerID:  "actor:" + value.Name,
		},
	}
}

func islandBoundaryRequest(value Value) islandkernel.BoundaryRequest {
	return islandkernel.BoundaryRequest{
		Ref: islandkernel.MemoryRef{
			BaseID:     value.Name,
			IslandID:   "island:" + value.Name,
			Epoch:      0,
			OwnerID:    "actor:" + value.Name,
			Provenance: islandkernel.ProvenanceBorrowedView,
		},
		Transfer: islandkernel.TransferBorrowedView,
	}
}

func VerifyMailbox(m Mailbox) error {
	if m.Name == "" {
		return fmt.Errorf("actor mailbox: missing name")
	}
	if m.Message == "" {
		return fmt.Errorf("actor mailbox: %s missing message schema", m.Name)
	}
	if m.Capacity <= 0 {
		return fmt.Errorf("actor mailbox: %s capacity must be positive", m.Name)
	}
	if m.Backpressure == "" {
		return fmt.Errorf("actor mailbox: %s must name an explicit backpressure policy", m.Name)
	}
	return nil
}

func site(s string) string {
	if s == "" {
		return "<unknown>"
	}
	return s
}
