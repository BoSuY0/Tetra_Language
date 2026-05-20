package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestClassifyCallableEscapeUsesFnptrForBoundedLocalSnapshot(t *testing.T) {
	kind, handle, err := classifyCallableEscape(callableBoundaryReturn, []frontend.ClosureCapture{
		{Name: "base", Type: frontend.TypeRef{Name: "i32"}},
	}, baseTypes())
	if err != nil {
		t.Fatalf("classifyCallableEscape: %v", err)
	}
	if kind != CallableEscapeLocalSnapshot || handle {
		t.Fatalf("classification = (%q, %v), want (%q, false)", kind, handle, CallableEscapeLocalSnapshot)
	}
}

func TestClassifyCallableEscapeUsesHandleForOversizedReturn(t *testing.T) {
	captures := make([]frontend.ClosureCapture, 0, FnPtrEnvSlotCount+1)
	for i := 0; i < FnPtrEnvSlotCount+1; i++ {
		captures = append(captures, frontend.ClosureCapture{
			Name: "capture",
			Type: frontend.TypeRef{Name: "i32"},
		})
	}

	kind, handle, err := classifyCallableEscape(callableBoundaryReturn, captures, baseTypes())
	if err != nil {
		t.Fatalf("classifyCallableEscape: %v", err)
	}
	if kind != CallableEscapeHeap || !handle {
		t.Fatalf("classification = (%q, %v), want (%q, true)", kind, handle, CallableEscapeHeap)
	}
}

func TestClassifyCallableEscapeRejectsMutableEscapingCapture(t *testing.T) {
	captures := make([]frontend.ClosureCapture, 0, FnPtrEnvSlotCount+1)
	for i := 0; i < FnPtrEnvSlotCount+1; i++ {
		captures = append(captures, frontend.ClosureCapture{
			Name:    "total",
			Type:    frontend.TypeRef{Name: "i32"},
			Mutable: i == 0,
		})
	}

	_, _, err := classifyCallableEscape(callableBoundaryGlobal, captures, baseTypes())
	if err == nil {
		t.Fatalf("expected mutable capture escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestClassifyCallableEscapeRejectsResourceCaptureAcrossThreadBoundary(t *testing.T) {
	_, _, err := classifyCallableEscape(callableBoundaryThread, []frontend.ClosureCapture{
		{Name: "raw", Type: frontend.TypeRef{Name: "ptr"}},
	}, baseTypes())
	if err == nil {
		t.Fatalf("expected resource capture escape diagnostic")
	}
	want := "escaped function value captures local 'raw' of type 'ptr'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestClassifyCallableEscapeRejectsMutableCaptureAcrossThreadBoundary(t *testing.T) {
	_, _, err := classifyCallableEscape(callableBoundaryThread, []frontend.ClosureCapture{
		{Name: "total", Type: frontend.TypeRef{Name: "i32"}, Mutable: true},
	}, baseTypes())
	if err == nil {
		t.Fatalf("expected mutable capture thread escape diagnostic")
	}
	want := "thread-escaped function value captures mutable local 'total'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}
