package x64abi

import (
	"tetra_language/compiler/internal/ir"
)

const allocationLengthTrapExitCode int32 = 2
const maxI32AllocationBytes int32 = 1<<31 - 1

func makeSliceMaxElements(kind ir.IRInstrKind) int32 {
	switch kind {
	case ir.IRMakeSliceU16, ir.IRIslandMakeSliceU16:
		return maxI32AllocationBytes / 2
	case ir.IRMakeSliceI32, ir.IRIslandMakeSliceI32:
		return maxI32AllocationBytes / 4
	default:
		return maxI32AllocationBytes
	}
}

func makeSliceNeedsOverflowGuard(kind ir.IRInstrKind) bool {
	return makeSliceMaxElements(kind) != maxI32AllocationBytes
}
