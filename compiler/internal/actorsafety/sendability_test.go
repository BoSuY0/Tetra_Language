package actorsafety

import (
	"strings"
	"testing"
)

func TestSmallScalarMustSendByCopy(t *testing.T) {
	checker := NewChecker([]Value{{Name: "count", Type: "i32", Kind: ValueCopy}})
	err := checker.Check([]Event{{Kind: EventSend, Value: "count", Mode: SendMove, Site: "app.tetra:3"}})
	if err == nil || !strings.Contains(err.Error(), "small scalar") || !strings.Contains(err.Error(), "by copy") {
		t.Fatalf("scalar move send error = %v, want copy-only contract", err)
	}
	checker = NewChecker([]Value{{Name: "count", Type: "i32", Kind: ValueCopy}})
	if err := checker.Check([]Event{{Kind: EventSend, Value: "count", Mode: SendCopy, Site: "app.tetra:4"}}); err != nil {
		t.Fatalf("scalar copy send should pass: %v", err)
	}
}

func TestBorrowedSliceAcrossActorBoundaryRejectsUnlessCopied(t *testing.T) {
	checker := NewChecker([]Value{{Name: "xs", Type: "[]u8", Kind: ValueBorrowed}})
	err := checker.Check([]Event{{Kind: EventSend, Value: "xs", Mode: SendBorrowed, Site: "app.tetra:7"}})
	if err == nil || !strings.Contains(err.Error(), "cannot send borrowed view across actor boundary; use .copy()") {
		t.Fatalf("borrowed send error = %v, want .copy() rejection", err)
	}
	checker = NewChecker([]Value{{Name: "xs", Type: "[]u8", Kind: ValueBorrowed}})
	if err := checker.Check([]Event{{Kind: EventSend, Value: "xs", Mode: SendCopy, Site: "app.tetra:8"}}); err != nil {
		t.Fatalf("copied borrowed send should pass: %v", err)
	}
}

func TestOwnedStringCanCopyOrMoveDependingOwnership(t *testing.T) {
	checker := NewChecker([]Value{{Name: "text", Type: "String", Kind: ValueOwned}})
	if err := checker.Check([]Event{{Kind: EventSend, Value: "text", Mode: SendCopy, Site: "app.tetra:9"}}); err != nil {
		t.Fatalf("owned String copy send should pass: %v", err)
	}
	checker = NewChecker([]Value{{Name: "text", Type: "String", Kind: ValueOwned}})
	if err := checker.Check([]Event{{Kind: EventSend, Value: "text", Mode: SendMove, Site: "app.tetra:10"}}); err != nil {
		t.Fatalf("owned String move send should pass: %v", err)
	}
	checker = NewChecker([]Value{{Name: "text", Type: "String", Kind: ValueOwned}})
	err := checker.Check([]Event{{Kind: EventSend, Value: "text", Mode: SendBorrowed, Site: "app.tetra:11"}})
	if err == nil || !strings.Contains(err.Error(), "must be moved or explicitly copied") {
		t.Fatalf("owned String borrowed-mode error = %v, want move/copy requirement", err)
	}
}

func TestOwnedRegionMustMoveAndSenderUseAfterMoveRejects(t *testing.T) {
	checker := NewChecker([]Value{{Name: "request_region", Type: "region", Kind: ValueOwnedRegion}})
	err := checker.Check([]Event{{Kind: EventSend, Value: "request_region", Mode: SendCopy, Site: "app.tetra:10"}})
	if err == nil || !strings.Contains(err.Error(), "must be moved") {
		t.Fatalf("copying owned region error = %v, want move requirement", err)
	}
	checker = NewChecker([]Value{{Name: "request_region", Type: "region", Kind: ValueOwnedRegion}})
	err = checker.Check([]Event{
		{Kind: EventSend, Value: "request_region", Mode: SendMove, Site: "app.tetra:12"},
		{Kind: EventUse, Value: "request_region", Site: "app.tetra:13"},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot use moved region after send") {
		t.Fatalf("use-after-move error = %v, want moved region rejection", err)
	}
}

func TestActorSendabilityStableDiagnosticsMatrix(t *testing.T) {
	cases := []struct {
		name   string
		values []Value
		events []Event
		want   []string
	}{
		{
			name:   "borrowed typed payload requires copy",
			values: []Value{{Name: "payload", Type: "[]u8", Kind: ValueBorrowed}},
			events: []Event{{Kind: EventSend, Value: "payload", Mode: SendBorrowed, Site: "worker.t4:7"}},
			want: []string{
				"actor sendability",
				"worker.t4:7",
				"cannot send borrowed view across actor boundary; use .copy() for \"payload\"",
			},
		},
		{
			name:   "sender use-after-move names send and use sites",
			values: []Value{{Name: "request_region", Type: "region", Kind: ValueOwnedRegion}},
			events: []Event{
				{Kind: EventSend, Value: "request_region", Mode: SendMove, Site: "worker.t4:12"},
				{Kind: EventUse, Value: "request_region", Site: "worker.t4:13"},
			},
			want: []string{
				"actor sendability",
				"worker.t4:13",
				"cannot use moved region after send",
				"\"request_region\" was moved at worker.t4:12",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(tt.values)
			err := checker.Check(tt.events)
			if err == nil {
				t.Fatalf("expected sendability diagnostic")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want text %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "MVP") {
				t.Fatalf("error = %v, want stable non-versioned diagnostic", err)
			}
		})
	}
}

func TestUnsafePointerRequiresExplicitUnsafeSendContract(t *testing.T) {
	checker := NewChecker([]Value{{Name: "raw", Type: "ptr", Kind: ValueUnsafePtr}})
	err := checker.Check([]Event{{Kind: EventSend, Value: "raw", Mode: SendUnsafe, Site: "app.tetra:20"}})
	if err == nil || !strings.Contains(err.Error(), "cannot send unknown unsafe provenance without audited contract") {
		t.Fatalf("unsafe pointer send error = %v, want audited contract rejection", err)
	}
	checker = NewChecker([]Value{{Name: "raw", Type: "ptr", Kind: ValueUnsafePtr, UnsafeSendContract: true}})
	if err := checker.Check([]Event{{Kind: EventSend, Value: "raw", Mode: SendUnsafe, Site: "app.tetra:21"}}); err != nil {
		t.Fatalf("unsafe contract send should pass: %v", err)
	}
}

func TestTypedMailboxRequiresCapacityAndBackpressure(t *testing.T) {
	if err := VerifyMailbox(Mailbox{Name: "worker", Message: "Frame", Capacity: 32, Backpressure: "block"}); err != nil {
		t.Fatalf("VerifyMailbox valid: %v", err)
	}
	err := VerifyMailbox(Mailbox{Name: "worker", Message: "Frame"})
	if err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("VerifyMailbox error = %v, want capacity rejection", err)
	}
}
