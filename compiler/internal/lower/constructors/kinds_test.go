package constructors

import (
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestIndexStoreKindSupportsScalarsAndSingleSlotStructs(t *testing.T) {
	types := map[string]*semantics.TypeInfo{
		"OneSlot": {Kind: semantics.TypeStruct, SlotCount: 1},
		"Wide":    {Kind: semantics.TypeStruct, SlotCount: 2},
	}

	for _, elem := range []string{"i32", "c_int", "usize", "bool"} {
		if kind, ok := IndexStoreKind(elem, types); !ok || kind != ir.IRIndexStoreI32 {
			t.Fatalf("IndexStoreKind(%q) = %v, %v", elem, kind, ok)
		}
	}
	if kind, ok := IndexStoreKind("u8", types); !ok || kind != ir.IRIndexStoreU8 {
		t.Fatalf("u8 kind = %v, %v", kind, ok)
	}
	if kind, ok := IndexStoreKind("u16", types); !ok || kind != ir.IRIndexStoreU16 {
		t.Fatalf("u16 kind = %v, %v", kind, ok)
	}
	if kind, ok := IndexStoreKind("OneSlot", types); !ok || kind != ir.IRIndexStoreI32 {
		t.Fatalf("OneSlot kind = %v, %v", kind, ok)
	}
	if _, ok := IndexStoreKind("Wide", types); ok {
		t.Fatalf("wide structs should not be index-store scalars")
	}
}
