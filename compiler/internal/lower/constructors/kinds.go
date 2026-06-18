package constructors

import (
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func IndexStoreKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	switch elemType {
	case "i32", "c_int", "c_uint",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return ir.IRIndexStoreI32, true
	case "bool":
		return ir.IRIndexStoreI32, true
	case "u8":
		return ir.IRIndexStoreU8, true
	case "u16":
		return ir.IRIndexStoreU16, true
	}
	info, ok := types[elemType]
	if !ok {
		return 0, false
	}
	if info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return ir.IRIndexStoreI32, true
	}
	return 0, false
}
