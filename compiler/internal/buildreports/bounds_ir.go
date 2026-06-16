package buildreports

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

func isUncheckedIndexLoad(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return true
	default:
		return false
	}
}

func isCheckedIndexAccess(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return true
	default:
		return false
	}
}

func irIndexKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
		return "i32.load"
	case ir.IRIndexLoadU8, ir.IRIndexLoadU8Unchecked:
		return "u8.load"
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
		return "u16.load"
	case ir.IRIndexStoreI32:
		return "i32.store"
	case ir.IRIndexStoreU8:
		return "u8.store"
	case ir.IRIndexStoreU16:
		return "u16.store"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}
