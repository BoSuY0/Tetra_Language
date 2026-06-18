package model

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestModelTypesCarryFrontendReferences(t *testing.T) {
	field := FieldInfo{
		Name:            "callback",
		FunctionTypeRef: frontend.TypeRef{Kind: frontend.TypeRefFunction},
	}
	info := TypeInfo{
		Name:     "Widget",
		Kind:     TypeStruct,
		Fields:   []FieldInfo{field},
		FieldMap: map[string]FieldInfo{field.Name: field},
	}
	if info.FieldMap["callback"].FunctionTypeRef.Kind != frontend.TypeRefFunction {
		t.Fatalf("function type ref was not preserved: %#v", info.FieldMap["callback"])
	}
}

func TestModelCallableAndRuntimeConstants(t *testing.T) {
	if FnPtrSlotCount != 1+FnPtrEnvSlotCount {
		t.Fatalf("FnPtrSlotCount = %d, want %d", FnPtrSlotCount, 1+FnPtrEnvSlotCount)
	}
	if CallableHandleSlotCount != 4 {
		t.Fatalf("CallableHandleSlotCount = %d, want 4", CallableHandleSlotCount)
	}
	if MaxActorStateSlots != 8 {
		t.Fatalf("MaxActorStateSlots = %d, want 8", MaxActorStateSlots)
	}
	if CallableEscapeHeap != CallableEscapeKind("heap") {
		t.Fatalf("CallableEscapeHeap = %q", CallableEscapeHeap)
	}
}
