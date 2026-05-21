package compiler

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	ctarget "tetra_language/compiler/target"
)

func validateTargetAtomicIR(funcs []IRFunc, tgt ctarget.Target) error {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			info, ok := atomicIRTargetInfo(instr.Kind, tgt)
			if !ok {
				continue
			}
			if info.op == ctarget.AtomicFence {
				if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, info.order); err != nil {
					return targetAtomicDiagnostic(instr.Pos, tgt.Triple, info.op, 0, err)
				}
				continue
			}
			if _, err := tgt.AtomicLayout(info.widthBits); err != nil {
				return targetAtomicDiagnostic(instr.Pos, tgt.Triple, info.op, info.widthBits, err)
			}
		}
	}
	return nil
}

type atomicIRInfo struct {
	op        ctarget.AtomicOp
	widthBits int
	order     ctarget.MemoryOrder
}

func atomicIRTargetInfo(kind ir.IRInstrKind, tgt ctarget.Target) (atomicIRInfo, bool) {
	ptrWidth := tgt.PointerWidthBits
	switch kind {
	case ir.IRAtomicLoadPtr:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: ptrWidth}, true
	case ir.IRAtomicStorePtr:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: ptrWidth}, true
	case ir.IRAtomicExchangePtr:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchAddPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchSubPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchAndPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchOrPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchXorPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: ptrWidth}, true
	case ir.IRAtomicCompareExchangePtr:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: ptrWidth}, true
	case ir.IRAtomicFenceSeqCst:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderSeqCst}, true
	case ir.IRAtomicFenceRelaxed:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderRelaxed}, true
	case ir.IRAtomicFenceAcquire:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderAcquire}, true
	case ir.IRAtomicFenceRelease:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderRelease}, true
	case ir.IRAtomicFenceAcqRel:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderAcqRel}, true
	case ir.IRAtomicLoadI8:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 8}, true
	case ir.IRAtomicStoreI8:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 8}, true
	case ir.IRAtomicExchangeI8:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 8}, true
	case ir.IRAtomicCompareExchangeI8:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 8}, true
	case ir.IRAtomicFetchAddI8:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 8}, true
	case ir.IRAtomicFetchSubI8:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 8}, true
	case ir.IRAtomicFetchAndI8:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 8}, true
	case ir.IRAtomicFetchOrI8:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 8}, true
	case ir.IRAtomicFetchXorI8:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 8}, true
	case ir.IRAtomicLoadI16:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 16}, true
	case ir.IRAtomicStoreI16:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 16}, true
	case ir.IRAtomicExchangeI16:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 16}, true
	case ir.IRAtomicCompareExchangeI16:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 16}, true
	case ir.IRAtomicFetchAddI16:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 16}, true
	case ir.IRAtomicFetchSubI16:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 16}, true
	case ir.IRAtomicFetchAndI16:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 16}, true
	case ir.IRAtomicFetchOrI16:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 16}, true
	case ir.IRAtomicFetchXorI16:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 16}, true
	case ir.IRAtomicLoadI32:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 32}, true
	case ir.IRAtomicStoreI32:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 32}, true
	case ir.IRAtomicExchangeI32:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 32}, true
	case ir.IRAtomicCompareExchangeI32:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 32}, true
	case ir.IRAtomicFetchAddI32:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 32}, true
	case ir.IRAtomicFetchSubI32:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 32}, true
	case ir.IRAtomicFetchAndI32:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 32}, true
	case ir.IRAtomicFetchOrI32:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 32}, true
	case ir.IRAtomicFetchXorI32:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 32}, true
	case ir.IRAtomicLoadI64:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 64}, true
	case ir.IRAtomicStoreI64:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 64}, true
	case ir.IRAtomicExchangeI64:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 64}, true
	case ir.IRAtomicCompareExchangeI64:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 64}, true
	case ir.IRAtomicFetchAddI64:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 64}, true
	case ir.IRAtomicFetchSubI64:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 64}, true
	case ir.IRAtomicFetchAndI64:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 64}, true
	case ir.IRAtomicFetchOrI64:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 64}, true
	case ir.IRAtomicFetchXorI64:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 64}, true
	default:
		return atomicIRInfo{}, false
	}
}

func targetAtomicDiagnostic(pos frontend.Position, target string, op ctarget.AtomicOp, widthBits int, cause error) error {
	width := "pointer-sized"
	if widthBits > 0 {
		width = fmt.Sprintf("%d-bit", widthBits)
	}
	hint := fmt.Sprintf("Use an atomic width supported by %s, or build this source for a target whose atomic model supports %s operations.", target, width)
	if target == "linux-x86" {
		hint = "Use 8/16/32-bit or pointer atomics on linux-x86, or build for linux-x64/linux-x32 when 64-bit lock-free atomics are required."
	}
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("%s atomic %s requires unsupported %s width: %v", target, op, width, cause),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}
