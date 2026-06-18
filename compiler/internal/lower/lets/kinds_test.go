package lets

import (
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestStackRegionAndIslandKindMappings(t *testing.T) {
	if kind, ok := StackSliceKindByBuiltin("core.make_u16"); !ok || kind != ir.IRStackSliceU16 {
		t.Fatalf("stack builtin kind = %v, %v", kind, ok)
	}
	if elem, ok := StackAllocationElementByBuiltin("core.make_bool"); !ok || elem != "bool" {
		t.Fatalf("stack elem = %q, %v", elem, ok)
	}
	if kind, ok := StackSliceKindByElement("i32"); !ok || kind != ir.IRStackSliceI32 {
		t.Fatalf("stack elem kind = %v, %v", kind, ok)
	}
	if kind, ok := RegionSliceKindByElement("u8"); !ok || kind != ir.IRRegionMakeSliceU8 {
		t.Fatalf("region elem kind = %v, %v", kind, ok)
	}
	if kind, ok := IslandSliceKindByBuiltin("core.island_make_bool"); !ok ||
		kind != ir.IRIslandMakeSliceI32 {
		t.Fatalf("island builtin kind = %v, %v", kind, ok)
	}
	if _, ok := StackSliceKindByBuiltin("core.make_str"); ok {
		t.Fatalf("unexpected stack kind for str")
	}
}

func TestAllocationElementSizeByBuiltin(t *testing.T) {
	if size, ok := AllocationElementSizeByBuiltin("core.make_u8"); !ok || size != 1 {
		t.Fatalf("u8 size = %d, %v", size, ok)
	}
	if size, ok := AllocationElementSizeByBuiltin("core.island_make_bool"); !ok || size != 4 {
		t.Fatalf("bool size = %d, %v", size, ok)
	}
	if _, ok := AllocationElementSizeByBuiltin("core.make_str"); ok {
		t.Fatalf("unexpected allocation element size for str")
	}
}
