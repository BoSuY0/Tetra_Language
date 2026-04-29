package semantics

import (
	"strings"
	"testing"
)

func TestEnsureTypeInfoArraySupportedSubset(t *testing.T) {
	types := baseTypes()
	tests := []struct {
		name string
		elem string
		len  int
	}{
		{name: "[1]i32", elem: "i32", len: 1},
		{name: "[2]bool", elem: "bool", len: 2},
		{name: "[3]u8", elem: "u8", len: 3},
		{name: "[4]u16", elem: "u16", len: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ensureTypeInfo(tt.name, types)
			if err != nil {
				t.Fatalf("ensureTypeInfo(%q): %v", tt.name, err)
			}
			if info.Kind != TypeArray {
				t.Fatalf("kind = %v, want TypeArray", info.Kind)
			}
			if info.ElemType != tt.elem || info.ArrayLen != tt.len {
				t.Fatalf("array info = elem=%q len=%d, want elem=%q len=%d", info.ElemType, info.ArrayLen, tt.elem, tt.len)
			}
			if info.SlotCount != 2 {
				t.Fatalf("slot count = %d, want 2", info.SlotCount)
			}
		})
	}
}

func TestEnsureTypeInfoArrayRejectsUnsupportedSubset(t *testing.T) {
	types := baseTypes()

	if _, err := ensureTypeInfo("[0]i32", types); err == nil || !strings.Contains(err.Error(), "array size must be positive constant") {
		t.Fatalf("expected positive-size error, got: %v", err)
	}

	if _, err := ensureTypeInfo("[2]str", types); err == nil || !strings.Contains(err.Error(), "array element type 'str' is not supported") {
		t.Fatalf("expected unsupported-element error, got: %v", err)
	}
}
