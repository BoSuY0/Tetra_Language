package actorsafety

import "fmt"

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
				return fmt.Errorf("actor sendability: %s cannot use moved region after send; %q was moved at %s", site(event.Site), value.Name, movedAt)
			}
			return fmt.Errorf("actor sendability: %s uses %q after it was moved at %s", site(event.Site), value.Name, movedAt)
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
		return fmt.Errorf("actor sendability: %s small scalar value %q must cross actor boundary by copy", site(event.Site), value.Name)
	case ValueOwned:
		if event.Mode == SendMove || event.Mode == SendCopy {
			return nil
		}
		return fmt.Errorf("actor sendability: %s owned value %q must be moved or explicitly copied before actor send", site(event.Site), value.Name)
	case ValueOwnedRegion:
		if event.Mode == SendMove {
			return nil
		}
		return fmt.Errorf("actor sendability: %s owned region %q must be moved to cross actor boundary", site(event.Site), value.Name)
	case ValueBorrowed:
		if event.Mode == SendCopy {
			return nil
		}
		return fmt.Errorf("actor sendability: %s cannot send borrowed view across actor boundary; use .copy() for %q", site(event.Site), value.Name)
	case ValueMutableAlias:
		return fmt.Errorf("actor sendability: %s mutable alias %q cannot cross actor boundary", site(event.Site), value.Name)
	case ValueUnsafePtr:
		if event.Mode == SendUnsafe && value.UnsafeSendContract {
			return nil
		}
		return fmt.Errorf("actor sendability: %s cannot send unknown unsafe provenance without audited contract for %q", site(event.Site), value.Name)
	default:
		return fmt.Errorf("actor sendability: %s value %q has unknown sendability kind %q", site(event.Site), value.Name, value.Kind)
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
