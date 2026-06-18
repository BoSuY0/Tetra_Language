package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/ir"
)

type emitAtomicInstrOps struct {
	pointerWidthBytes                int32
	pop                              func(int) error
	push                             func(int)
	guardAllocationBaseRawAccess     func(int32) error
	emitPointerLoad                  func()
	emitAtomicPointerStore           func()
	emitAtomicPointerExchange        func()
	emitAtomicPointerFetchAdd        func()
	emitAtomicPointerFetchSub        func()
	emitAtomicPointerFetchCASLoop    func(func(), func()) error
	emitAtomicPointerCompareExchange func()
	emitAtomicI32CompareExchange     func()
	emitAtomicI32FetchCASLoop        func(func()) error
	emitAtomicI64CompareExchange     func()
	emitAtomicI64FetchCASLoop        func(func()) error
	emitAtomicI8CompareExchange      func()
	emitAtomicI8FetchCASLoop         func(func()) error
	emitAtomicI16CompareExchange     func()
	emitAtomicI16FetchCASLoop        func(func()) error
}

func emitAtomicInstr(e *x64.Emitter, instr ir.IRInstr, ops emitAtomicInstrOps) error {
	switch instr.Kind {
	case ir.IRAtomicLoadPtr:
		if err := ops.pop(2); err != nil {
			return err
		}
		e.PopRdx()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitPointerLoad()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicStorePtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitAtomicPointerStore()
		e.PushR9()
		ops.push(1)
	case ir.IRAtomicExchangePtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitAtomicPointerExchange()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAddPtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitAtomicPointerFetchAdd()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchSubPtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitAtomicPointerFetchSub()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAndPtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		if err := ops.emitAtomicPointerFetchCASLoop(e.AndR10dR8d, e.AndR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchOrPtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		if err := ops.emitAtomicPointerFetchCASLoop(e.OrR10dR8d, e.OrR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchXorPtr:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		if err := ops.emitAtomicPointerFetchCASLoop(e.XorR10dR8d, e.XorR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicCompareExchangePtr:
		if err := ops.pop(4); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopR9()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(ops.pointerWidthBytes); err != nil {
			return err
		}
		ops.emitAtomicPointerCompareExchange()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFenceSeqCst:
		e.Mfence()
	case ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		// x86-family TSO gives acquire/release fence semantics without
		// a hardware fence; seq_cst remains the explicit mfence case.
	case ir.IRAtomicLoadI32:
		if err := ops.pop(2); err != nil {
			return err
		}
		e.PopRdx()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		e.MovEaxFromRaxPtr()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicStoreI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		e.MovRdiRax()
		e.MovR9R8()
		e.XchgMem32RdiPtrR8d()
		e.PushR9()
		ops.push(1)
	case ir.IRAtomicExchangeI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		e.MovRdiRax()
		e.XchgMem32RdiPtrR8d()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicCompareExchangeI32:
		if err := ops.pop(4); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopR9()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		ops.emitAtomicI32CompareExchange()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchAddI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		e.MovRdiRax()
		e.LockXaddMem32RdiPtrR8d()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchSubI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		e.MovRdiRax()
		e.NegR8d()
		e.LockXaddMem32RdiPtrR8d()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAndI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		if err := ops.emitAtomicI32FetchCASLoop(e.AndR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchOrI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		if err := ops.emitAtomicI32FetchCASLoop(e.OrR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchXorI32:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(4); err != nil {
			return err
		}
		if err := ops.emitAtomicI32FetchCASLoop(e.XorR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicLoadI64:
		if err := ops.pop(2); err != nil {
			return err
		}
		e.PopRdx()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		e.MovRdiRax()
		e.MovRaxFromRdiDisp(0)
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicStoreI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		e.MovRdiRax()
		e.MovR9R8()
		e.XchgMem64RdiPtrR8()
		e.PushR9()
		ops.push(1)
	case ir.IRAtomicExchangeI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		e.MovRdiRax()
		e.XchgMem64RdiPtrR8()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicCompareExchangeI64:
		if err := ops.pop(4); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopR9()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		ops.emitAtomicI64CompareExchange()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchAddI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		e.MovRdiRax()
		e.LockXaddMem64RdiPtrR8()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchSubI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		e.MovRdiRax()
		e.NegR8()
		e.LockXaddMem64RdiPtrR8()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAndI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		if err := ops.emitAtomicI64FetchCASLoop(e.AndR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchOrI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		if err := ops.emitAtomicI64FetchCASLoop(e.OrR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchXorI64:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(8); err != nil {
			return err
		}
		if err := ops.emitAtomicI64FetchCASLoop(e.XorR10R8); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicLoadI8:
		if err := ops.pop(2); err != nil {
			return err
		}
		e.PopRdx()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		e.MovzxEaxBytePtrRax()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicStoreI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		e.MovRdiRax()
		e.MovzxR8dR8b()
		e.MovR9R8()
		e.XchgMem8RdiPtrR8b()
		e.PushR9()
		ops.push(1)
	case ir.IRAtomicExchangeI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		e.MovRdiRax()
		e.XchgMem8RdiPtrR8b()
		e.MovzxR8dR8b()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicCompareExchangeI8:
		if err := ops.pop(4); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopR9()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		ops.emitAtomicI8CompareExchange()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchAddI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		e.MovRdiRax()
		e.LockXaddMem8RdiPtrR8b()
		e.MovzxR8dR8b()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchSubI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		e.MovRdiRax()
		e.NegR8b()
		e.LockXaddMem8RdiPtrR8b()
		e.MovzxR8dR8b()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAndI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		if err := ops.emitAtomicI8FetchCASLoop(e.AndR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchOrI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		if err := ops.emitAtomicI8FetchCASLoop(e.OrR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchXorI8:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(1); err != nil {
			return err
		}
		if err := ops.emitAtomicI8FetchCASLoop(e.XorR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicLoadI16:
		if err := ops.pop(2); err != nil {
			return err
		}
		e.PopRdx()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		e.MovzxEaxWordPtrRax()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicStoreI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		e.MovRdiRax()
		e.MovzxR8dR8w()
		e.MovR9R8()
		e.XchgMem16RdiPtrR8w()
		e.PushR9()
		ops.push(1)
	case ir.IRAtomicExchangeI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		e.MovRdiRax()
		e.XchgMem16RdiPtrR8w()
		e.MovzxR8dR8w()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicCompareExchangeI16:
		if err := ops.pop(4); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopR9()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		ops.emitAtomicI16CompareExchange()
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchAddI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		e.MovRdiRax()
		e.LockXaddMem16RdiPtrR8w()
		e.MovzxR8dR8w()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchSubI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		e.MovRdiRax()
		e.NegR8w()
		e.LockXaddMem16RdiPtrR8w()
		e.MovzxR8dR8w()
		e.PushR8()
		ops.push(1)
	case ir.IRAtomicFetchAndI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		if err := ops.emitAtomicI16FetchCASLoop(e.AndR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchOrI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		if err := ops.emitAtomicI16FetchCASLoop(e.OrR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	case ir.IRAtomicFetchXorI16:
		if err := ops.pop(3); err != nil {
			return err
		}
		e.PopRdx()
		e.PopR8()
		e.PopRax()
		if err := ops.guardAllocationBaseRawAccess(2); err != nil {
			return err
		}
		if err := ops.emitAtomicI16FetchCASLoop(e.XorR10dR8d); err != nil {
			return err
		}
		e.PushRax()
		ops.push(1)
	default:
		return fmt.Errorf("x64 backend: unsupported atomic instruction %v", instr.Kind)
	}
	return nil
}
