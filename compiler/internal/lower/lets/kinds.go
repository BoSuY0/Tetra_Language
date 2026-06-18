package lets

import "tetra_language/compiler/internal/ir"

func StackSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	switch name {
	case "core.make_u8":
		return ir.IRStackSliceU8, true
	case "core.make_u16":
		return ir.IRStackSliceU16, true
	case "core.make_i32", "core.make_bool":
		return ir.IRStackSliceI32, true
	default:
		return 0, false
	}
}

func StackAllocationElementByBuiltin(name string) (string, bool) {
	switch name {
	case "core.make_u8":
		return "u8", true
	case "core.make_u16":
		return "u16", true
	case "core.make_i32":
		return "i32", true
	case "core.make_bool":
		return "bool", true
	default:
		return "", false
	}
}

func StackSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	switch elem {
	case "u8":
		return ir.IRStackSliceU8, true
	case "u16":
		return ir.IRStackSliceU16, true
	case "i32", "bool":
		return ir.IRStackSliceI32, true
	default:
		return 0, false
	}
}

func RegionSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	switch elem {
	case "u8":
		return ir.IRRegionMakeSliceU8, true
	case "u16":
		return ir.IRRegionMakeSliceU16, true
	case "i32", "bool":
		return ir.IRRegionMakeSliceI32, true
	default:
		return 0, false
	}
}

func IslandSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	switch name {
	case "core.island_make_u8":
		return ir.IRIslandMakeSliceU8, true
	case "core.island_make_u16":
		return ir.IRIslandMakeSliceU16, true
	case "core.island_make_i32", "core.island_make_bool":
		return ir.IRIslandMakeSliceI32, true
	default:
		return 0, false
	}
}

func AllocationElementSizeByBuiltin(name string) (int, bool) {
	switch name {
	case "core.make_u8", "core.island_make_u8":
		return 1, true
	case "core.make_u16", "core.island_make_u16":
		return 2, true
	case "core.make_i32", "core.island_make_i32", "core.make_bool", "core.island_make_bool":
		return 4, true
	default:
		return 0, false
	}
}
