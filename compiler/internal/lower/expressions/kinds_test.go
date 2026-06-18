package expressions

import (
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestIndexLoadKindAndUncheckedKind(t *testing.T) {
	types := map[string]*semantics.TypeInfo{
		"OneSlot": {Kind: semantics.TypeStruct, SlotCount: 1},
	}

	if kind, ok := IndexLoadKind("u8", types); !ok || kind != ir.IRIndexLoadU8 {
		t.Fatalf("u8 load kind = %v, %v", kind, ok)
	}
	if kind, ok := IndexLoadKind("OneSlot", types); !ok || kind != ir.IRIndexLoadI32 {
		t.Fatalf("OneSlot load kind = %v, %v", kind, ok)
	}
	if got := UncheckedIndexLoadKind(ir.IRIndexLoadU16); got != ir.IRIndexLoadU16Unchecked {
		t.Fatalf("unchecked u16 = %v", got)
	}
	if got := UncheckedIndexLoadKind(ir.IRCall); got != ir.IRCall {
		t.Fatalf("unexpected fallback kind = %v", got)
	}
}

func TestInt32LikeAndSlotCountHelpers(t *testing.T) {
	if !Int32LikeType("u16") || !Int32LikeType("c_uint") {
		t.Fatalf("expected int32-like scalar")
	}
	if Int32LikeType("str") {
		t.Fatalf("str should not be int32-like")
	}
	if slots := SlotCount("Pair", map[string]*semantics.TypeInfo{"Pair": {SlotCount: 2}}); slots != 2 {
		t.Fatalf("slot count = %d", slots)
	}
	if slots := SlotCount("Missing", nil); slots != 1 {
		t.Fatalf("missing slot count = %d", slots)
	}
}
