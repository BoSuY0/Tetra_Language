package validation

import "tetra_language/compiler/internal/ir"

func validationStackEffect(instr ir.IRInstr) (pop int, push int, known bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRAllocBytes, ir.IRIslandNew:
		return 1, 1, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		return 1, 2, true
	case ir.IRRegionEnter, ir.IRRegionReset:
		return 0, 0, true
	case ir.IRRawSliceFromParts:
		return 3, 2, true
	case ir.IRSliceWindow:
		return 4, 2, true
	case ir.IRSlicePrefix, ir.IRSliceSuffix:
		return 3, 2, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandReset:
		return 1, 1, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr, ir.IRMmioReadI32,
		ir.IRAtomicLoadPtr, ir.IRAtomicLoadI32, ir.IRAtomicLoadI64,
		ir.IRAtomicLoadI8, ir.IRAtomicLoadI16:
		return 2, 1, true
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr, ir.IRPtrAdd,
		ir.IRMmioWriteI32, ir.IRCtxSwitch, ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr, ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32, ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
		ir.IRAtomicFetchAddI64, ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8, ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8, ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return 3, 1, true
	case ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32, ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicCompareExchangeI8, ir.IRAtomicCompareExchangeI16:
		return 4, 1, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset:
		return 4, 1, true
	case ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		return 0, 0, true
	default:
		return 0, 0, false
	}
}
