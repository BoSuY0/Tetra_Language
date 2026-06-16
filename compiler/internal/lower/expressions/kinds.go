package expressions

import (
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func IndexLoadKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	switch elemType {
	case "i32", "c_int", "c_uint",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return ir.IRIndexLoadI32, true
	case "bool":
		return ir.IRIndexLoadI32, true
	case "u8":
		return ir.IRIndexLoadU8, true
	case "u16":
		return ir.IRIndexLoadU16, true
	}
	info, ok := types[elemType]
	if !ok {
		return 0, false
	}
	if info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return ir.IRIndexLoadI32, true
	}
	return 0, false
}

func UncheckedIndexLoadKind(kind ir.IRInstrKind) ir.IRInstrKind {
	switch kind {
	case ir.IRIndexLoadI32:
		return ir.IRIndexLoadI32Unchecked
	case ir.IRIndexLoadU8:
		return ir.IRIndexLoadU8Unchecked
	case ir.IRIndexLoadU16:
		return ir.IRIndexLoadU16Unchecked
	default:
		return kind
	}
}

func Int32LikeType(typeName string) bool {
	switch typeName {
	case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
		return true
	default:
		return semantics.IsILP32NativeScalarType(typeName)
	}
}

func SlotCount(typeName string, types map[string]*semantics.TypeInfo) int {
	if info, ok := types[typeName]; ok {
		return info.SlotCount
	}
	return 1
}
