package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

type labelPatch struct {
	at    int
	label int
}

func stackSliceMaxElements(kind ir.IRInstrKind) int32 {
	const maxI32AllocationBytes int32 = 1<<31 - 1
	switch kind {
	case ir.IRStackSliceU16:
		return maxI32AllocationBytes / 2
	case ir.IRStackSliceI32:
		return maxI32AllocationBytes / 4
	default:
		return maxI32AllocationBytes
	}
}

func functionTempRegionSliceMaxElements(kind ir.IRInstrKind) int32 {
	const maxI32AllocationBytes int32 = 1<<31 - 1
	switch kind {
	case ir.IRRegionMakeSliceU16:
		return maxI32AllocationBytes / 2
	case ir.IRRegionMakeSliceI32:
		return maxI32AllocationBytes / 4
	default:
		return maxI32AllocationBytes
	}
}

func functionHasTempRegionIR(fn ir.IRFunc) bool {
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRRegionEnter, ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32, ir.IRRegionReset:
			return true
		}
	}
	return false
}

func NewEmitFunc(abi x64abi.ABI) x64obj.EmitFunc {
	smallHeapStateDataIndex := -1
	return func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]x64obj.LeaPatch,
		callPatches *[]x64obj.CallPatch,
		importPatches *[]x64obj.ImportPatch,
		opt x64.CodegenOptions,
	) error {
		if abi == nil {
			return fmt.Errorf("missing ABI")
		}
		if e == nil {
			return fmt.Errorf("missing emitter")
		}
		if dataBlobs == nil || leaPatches == nil || callPatches == nil {
			return fmt.Errorf("missing patches buffers")
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots || fn.ReturnSlots < 0 {
			return fmt.Errorf("x64 backend: function '%s' has invalid slots", fn.Name)
		}
		pointerWidthBytes, err := opt.PointerWidthBytes()
		if err != nil {
			return fmt.Errorf("x64 backend: %w", err)
		}
		registerWidthBytes, err := opt.RegisterWidthBytes()
		if err != nil {
			return fmt.Errorf("x64 backend: %w", err)
		}

		labelOffsets := make(map[int]int)
		var patches []labelPatch
		var smallHeapCalls []int
		stackDepth := 0
		nextInternalLabel := -1

		newInternalLabel := func() int {
			id := nextInternalLabel
			nextInternalLabel--
			return id
		}
		ensureSmallHeapState := func() int {
			if smallHeapStateDataIndex >= 0 {
				return smallHeapStateDataIndex
			}
			*dataBlobs = append(*dataBlobs, make([]byte, 16))
			smallHeapStateDataIndex = len(*dataBlobs) - 1
			return smallHeapStateDataIndex
		}

		pop := func(n int) error {
			if stackDepth < n {
				return fmt.Errorf("stack underflow in function '%s'", fn.Name)
			}
			stackDepth -= n
			return nil
		}
		push := func(n int) { stackDepth += n }
		localSlotOffset := func(slot int) (int32, error) {
			if slot < 0 || slot >= fn.LocalSlots {
				return 0, fmt.Errorf("x64 backend: local slot %d out of bounds in function '%s' (locals=%d)", slot, fn.Name, fn.LocalSlots)
			}
			return -int32((slot + 1) * 8), nil
		}
		guardAllocationOffsetRawAccess := func(width int32) error {
			e.CmpEdxImm32(0)
			okAt := e.JgeRel32()
			if err := abi.EmitExit(e, 2, stackDepth, importPatches); err != nil {
				return err
			}
			okOff := len(e.Buf)
			if err := x64.PatchRel32(e.Buf, okAt, okOff); err != nil {
				return err
			}
			e.MovRdiRax()
			e.AndRdiImm32(-4096)
			e.MovEcxFromRdiDisp(0)
			e.AddRdiImm32(8)
			e.SubRaxRdi()
			e.AddRdxRax()
			e.AddEdxImm32(width)
			e.CmpEdxEcx()
			failAt := e.JaRel32()
			e.AddEdxImm32(-width)
			e.MovsxdRdxEdx()
			e.MovRaxRdi()
			e.AddRaxRdx()
			doneAt := e.JmpRel32()
			failOff := len(e.Buf)
			if err := abi.EmitExit(e, 2, stackDepth, importPatches); err != nil {
				return err
			}
			doneOff := len(e.Buf)
			if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
				return err
			}
			if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
				return err
			}
			return nil
		}
		guardAllocationBaseRawAccess := func(width int32) error {
			e.MovEdxImm32(0)
			return guardAllocationOffsetRawAccess(width)
		}
		emitPointerLoad := func() {
			switch pointerWidthBytes {
			case 4:
				e.MovEaxFromRaxPtr()
			default:
				e.MovRdiRax()
				e.MovRaxFromRdiDisp(0)
			}
		}
		emitPointerStore := func() {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.MovR8dR8d()
				e.MovMem32RdiDispR8d(0)
			default:
				e.MovMem64RdiDispR8(0)
			}
		}
		emitArchPointerStore := func() {
			e.MovRdiRax()
			switch registerWidthBytes {
			case 4:
				e.MovMem32RdiDispR8d(0)
			default:
				e.MovMem64RdiDispR8(0)
			}
		}
		emitAtomicPointerExchange := func() {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.XchgMem32RdiPtrR8d()
			default:
				e.XchgMem64RdiPtrR8()
			}
		}
		emitAtomicPointerStore := func() {
			switch pointerWidthBytes {
			case 4:
				e.MovR9dR8d()
			default:
				e.MovR9R8()
			}
			emitAtomicPointerExchange()
		}
		emitAtomicPointerCompareExchange := func() {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.MovEaxR9d()
				e.LockCmpxchgMem32RdiPtrR8d()
			default:
				e.MovRaxR9()
				e.LockCmpxchgMem64RdiPtrR8()
			}
		}
		emitAtomicPointerFetchAdd := func() {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.LockXaddMem32RdiPtrR8d()
			default:
				e.LockXaddMem64RdiPtrR8()
			}
		}
		emitAtomicPointerFetchSub := func() {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.NegR8d()
				e.LockXaddMem32RdiPtrR8d()
			default:
				e.NegR8()
				e.LockXaddMem64RdiPtrR8()
			}
		}
		emitAtomicPointerFetchCASLoop := func(op32 func(), op64 func()) error {
			e.MovRdiRax()
			switch pointerWidthBytes {
			case 4:
				e.MovEaxFromRdiDisp(0)
			default:
				e.MovRaxFromRdiDisp(0)
			}
			retryOff := len(e.Buf)
			switch pointerWidthBytes {
			case 4:
				e.MovR10dEax()
				op32()
				e.LockCmpxchgMem32RdiPtrR10d()
			default:
				e.MovR10Rax()
				op64()
				e.LockCmpxchgMem64RdiPtrR10()
			}
			retryAt := e.JnzRel32()
			return x64.PatchRel32(e.Buf, retryAt, retryOff)
		}
		emitAtomicI32CompareExchange := func() {
			e.MovRdiRax()
			e.MovRaxR9()
			e.LockCmpxchgMem32RdiPtrR8d()
		}
		emitAtomicI32FetchCASLoop := func(op func()) error {
			e.MovRdiRax()
			e.MovEaxFromRdiDisp(0)
			retryOff := len(e.Buf)
			e.MovR10dEax()
			op()
			e.LockCmpxchgMem32RdiPtrR10d()
			retryAt := e.JnzRel32()
			return x64.PatchRel32(e.Buf, retryAt, retryOff)
		}
		emitAtomicI64CompareExchange := func() {
			e.MovRdiRax()
			e.MovRaxR9()
			e.LockCmpxchgMem64RdiPtrR8()
		}
		emitAtomicI64FetchCASLoop := func(op func()) error {
			e.MovRdiRax()
			e.MovRaxFromRdiDisp(0)
			retryOff := len(e.Buf)
			e.MovR10Rax()
			op()
			e.LockCmpxchgMem64RdiPtrR10()
			retryAt := e.JnzRel32()
			return x64.PatchRel32(e.Buf, retryAt, retryOff)
		}
		emitAtomicI8CompareExchange := func() {
			e.MovRdiRax()
			e.MovRaxR9()
			e.LockCmpxchgMem8RdiPtrR8b()
			e.MovzxEaxAl()
		}
		emitAtomicI16CompareExchange := func() {
			e.MovRdiRax()
			e.MovRaxR9()
			e.LockCmpxchgMem16RdiPtrR8w()
			e.MovzxEaxAx()
		}
		emitAtomicI8FetchCASLoop := func(op func()) error {
			e.MovRdiRax()
			e.MovzxEaxBytePtrRdi()
			retryOff := len(e.Buf)
			e.MovR10dEax()
			op()
			e.LockCmpxchgMem8RdiPtrR10b()
			retryAt := e.JnzRel32()
			if err := x64.PatchRel32(e.Buf, retryAt, retryOff); err != nil {
				return err
			}
			e.MovzxEaxAl()
			return nil
		}
		emitAtomicI16FetchCASLoop := func(op func()) error {
			e.MovRdiRax()
			e.MovzxEaxWordPtrRdi()
			retryOff := len(e.Buf)
			e.MovR10dEax()
			op()
			e.LockCmpxchgMem16RdiPtrR10w()
			retryAt := e.JnzRel32()
			if err := x64.PatchRel32(e.Buf, retryAt, retryOff); err != nil {
				return err
			}
			e.MovzxEaxAx()
			return nil
		}

		if ok, err := emitVectorSliceSumRegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitVectorCopyU8RegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitVectorMapI32AddConstRegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitVectorMemsetZeroU8RegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitScalarSliceSumRegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitScalarCallLoopRegisterFunction(e, fn, abi, callPatches, opt); ok || err != nil {
			return err
		}
		if ok, err := emitScalarLoopRegisterFunction(e, fn, abi, opt); ok || err != nil {
			return err
		}
		if ok, err := emitScalarRegisterFunction(e, fn, abi, callPatches, opt); ok || err != nil {
			return err
		}

		functionTempRegionSlots := 0
		var functionTempRegionBaseOffset int32
		var functionTempRegionSizeOffset int32
		if functionHasTempRegionIR(fn) {
			functionTempRegionSlots = 2
			functionTempRegionBaseOffset = -int32((fn.LocalSlots + 1) * 8)
			functionTempRegionSizeOffset = -int32((fn.LocalSlots + 2) * 8)
		}

		e.PushRbp()
		e.MovRbpRsp()
		localSize := x64.AlignStackSize((fn.LocalSlots + functionTempRegionSlots) * 8)
		if localSize > 0 {
			e.SubRspImm32(int32(localSize))
		}
		abi.SpillParams(e, fn)
		for i := fn.ParamSlots; i < fn.LocalSlots; i++ {
			off := -int32((i + 1) * 8)
			e.MovMem64RbpDispImm(off, 0)
		}
		if functionTempRegionSlots > 0 {
			e.MovMem64RbpDispImm(functionTempRegionBaseOffset, 0)
			e.MovMem64RbpDispImm(functionTempRegionSizeOffset, 0)
		}

		for _, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRWrite:
				if err := abi.EmitWriteStdout(e, &stackDepth, importPatches); err != nil {
					return err
				}
			case ir.IRStrLit:
				if len(instr.Str) == 0 {
					e.MovEaxImm32(0)
					e.PushRax()
					e.PushRax()
					push(2)
					continue
				}
				leaPos := e.LeaRaxRipDisp()
				e.PushRax()
				e.MovEaxImm32(uint32(len(instr.Str)))
				e.PushRax()
				push(2)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: len(*dataBlobs)})
				*dataBlobs = append(*dataBlobs, instr.Str)
			case ir.IRConstI32:
				e.MovEaxImm32(uint32(instr.Imm))
				e.PushRax()
				push(1)
			case ir.IRLoadLocal:
				off, err := localSlotOffset(instr.Local)
				if err != nil {
					return err
				}
				e.MovRaxFromRbpDisp(off)
				e.PushRax()
				push(1)
			case ir.IRStoreLocal:
				if err := pop(1); err != nil {
					return err
				}
				off, err := localSlotOffset(instr.Local)
				if err != nil {
					return err
				}
				e.PopRax()
				e.MovMem64RbpDispRax(off)
			case ir.IRLoadGlobal:
				if instr.Local < 0 {
					return fmt.Errorf("x64 backend: global slot %d out of bounds in function '%s'", instr.Local, fn.Name)
				}
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovRaxFromRdiDisp(0)
				e.PushRax()
				push(1)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local})
			case ir.IRStoreGlobal:
				if instr.Local < 0 {
					return fmt.Errorf("x64 backend: global slot %d out of bounds in function '%s'", instr.Local, fn.Name)
				}
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovMem64RdiDispRax(0)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local})
			case ir.IRAddI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.AddEaxEcx()
				e.PushRax()
				push(1)
			case ir.IRSubI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.SubEaxEcx()
				e.PushRax()
				push(1)
			case ir.IRNegI32:
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				e.NegEax()
				e.PushRax()
				push(1)
			case ir.IRCmpEqI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SeteAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCmpLtI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetlAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRMulI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.ImulEaxEcx()
				e.PushRax()
				push(1)
			case ir.IRDivI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.Cdq()
				e.IdivEcx()
				e.PushRax()
				push(1)
			case ir.IRModI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.Cdq()
				e.IdivEcx()
				e.PushRdx()
				push(1)
			case ir.IRCmpGtI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetgAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCmpGeI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetgeAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCmpLeI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetleAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCmpNeI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetneAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCall:
				if err := abi.EmitCall(e, instr, &stackDepth, callPatches); err != nil {
					return err
				}
			case ir.IRLabel:
				if instr.Label < 0 {
					return fmt.Errorf("x64 backend: negative label %d in function '%s'", instr.Label, fn.Name)
				}
				if _, exists := labelOffsets[instr.Label]; exists {
					return fmt.Errorf("x64 backend: duplicate label %d in function '%s'", instr.Label, fn.Name)
				}
				labelOffsets[instr.Label] = len(e.Buf)
			case ir.IRJmp:
				if instr.Label < 0 {
					return fmt.Errorf("x64 backend: negative label %d in function '%s'", instr.Label, fn.Name)
				}
				at := e.JmpRel32()
				patches = append(patches, labelPatch{at: at, label: instr.Label})
			case ir.IRJmpIfZero:
				if instr.Label < 0 {
					return fmt.Errorf("x64 backend: negative label %d in function '%s'", instr.Label, fn.Name)
				}
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				e.TestEaxEax()
				at := e.JzRel32()
				patches = append(patches, labelPatch{at: at, label: instr.Label})
			case ir.IRReturn:
				if err := pop(fn.ReturnSlots); err != nil {
					return err
				}
				switch fn.ReturnSlots {
				case 1:
					e.PopRax()
				case 2:
					e.PopRdx()
					e.PopRax()
				case 3:
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 4:
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 5:
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 6:
					e.PopR11()
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 7:
					e.PopR12()
					e.PopR11()
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 8:
					e.PopR13()
					e.PopR12()
					e.PopR11()
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 9:
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopR11()
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				case 10:
					e.PopRbx()
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopR11()
					e.PopR10()
					e.PopR9()
					e.PopR8()
					e.PopRdx()
					e.PopRax()
				default:
					return fmt.Errorf("unsupported return slots %d in function %q", fn.ReturnSlots, fn.Name)
				}
				e.Leave()
				e.Ret()
			case ir.IRAllocBytes:
				if err := abi.EmitAllocBytes(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
				if emitSmallHeapMakeSliceEnabled(abi, opt, pointerWidthBytes) {
					stateIndex := ensureSmallHeapState()
					if err := emitSmallHeapMakeSlice(e, instr.Kind, &stackDepth, abi, importPatches, &smallHeapCalls, stateIndex); err != nil {
						return err
					}
					continue
				}
				if err := abi.EmitMakeSlice(e, instr.Kind, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRRegionEnter:
				if functionTempRegionSlots == 0 {
					return fmt.Errorf("function-temp region enter without frame state")
				}
				e.MovMem64RbpDispImm(functionTempRegionBaseOffset, 0)
				e.MovMem64RbpDispImm(functionTempRegionSizeOffset, 0)
			case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
				if functionTempRegionSlots == 0 {
					return fmt.Errorf("function-temp region make_slice without frame state")
				}
				if err := emitFunctionTempRegionMakeSlice(e, abi, instr.Kind, &stackDepth, functionTempRegionBaseOffset, functionTempRegionSizeOffset, importPatches); err != nil {
					return err
				}
			case ir.IRRegionReset:
				if functionTempRegionSlots == 0 {
					return fmt.Errorf("function-temp region reset without frame state")
				}
				if err := emitFunctionTempRegionReset(e, abi, &stackDepth, functionTempRegionBaseOffset, functionTempRegionSizeOffset, importPatches); err != nil {
					return err
				}
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				stackAfterPop := stackDepth
				e.TestRaxRax()
				negativeAt := e.JlRel32()
				emptyAt := e.JzRel32()
				overflowAt := -1
				if max := stackSliceMaxElements(instr.Kind); max != stackSliceMaxElements(ir.IRStackSliceU8) {
					e.CmpRaxImm32(max)
					overflowAt = e.JgRel32()
				}
				e.MovRcxRax()
				if instr.ArgSlots == 0 {
					e.MovEaxImm32(0)
				} else {
					off, err := localSlotOffset(instr.Local + instr.ArgSlots - 1)
					if err != nil {
						return err
					}
					e.LeaRaxRbpDisp(off)
				}
				e.PushRax()
				push(1)
				e.PushRcx()
				push(1)
				doneAt := e.JmpRel32()
				lengthFailOff := len(e.Buf)
				if err := abi.EmitExit(e, 2, stackAfterPop, importPatches); err != nil {
					return err
				}
				emptyOff := len(e.Buf)
				e.MovEaxImm32(0)
				e.PushRax()
				e.PushRax()
				doneOff := len(e.Buf)
				if err := x64.PatchRel32(e.Buf, negativeAt, lengthFailOff); err != nil {
					return err
				}
				if overflowAt >= 0 {
					if err := x64.PatchRel32(e.Buf, overflowAt, lengthFailOff); err != nil {
						return err
					}
				}
				if err := x64.PatchRel32(e.Buf, emptyAt, emptyOff); err != nil {
					return err
				}
				if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
					return err
				}
			case ir.IRRawSliceFromParts:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				stackBeforePush := stackDepth
				e.TestEcxEcx()
				negativeAt := e.JlRel32()
				overflowAt := -1
				if max := rawSliceMaxElements(instr.Imm); max != rawSliceMaxElements(0) {
					e.CmpRcxImm32(max)
					overflowAt = e.JgRel32()
				}
				e.PushRax()
				e.PushRcx()
				push(2)
				doneAt := e.JmpRel32()
				lengthFailOff := len(e.Buf)
				if err := abi.EmitExit(e, 2, stackBeforePush, importPatches); err != nil {
					return err
				}
				doneOff := len(e.Buf)
				if err := x64.PatchRel32(e.Buf, negativeAt, lengthFailOff); err != nil {
					return err
				}
				if overflowAt >= 0 {
					if err := x64.PatchRel32(e.Buf, overflowAt, lengthFailOff); err != nil {
						return err
					}
				}
				if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
					return err
				}
			case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
				if err := emitSliceView(e, instr.Kind, byte(instr.Imm), pop, push, &stackDepth, abi, importPatches); err != nil {
					return err
				}
			case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
				ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				checked := instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadU8 || instr.Kind == ir.IRIndexLoadU16
				failAt := 0
				if checked {
					e.CmpEdxEcx()
					failAt = e.JaeRel32()
				}
				if instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadI32Unchecked {
					e.ShlRdxImm8(2)
				} else if instr.Kind == ir.IRIndexLoadU16 || instr.Kind == ir.IRIndexLoadU16Unchecked {
					e.ShlRdxImm8(1)
				}
				e.AddRaxRdx()
				if instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadI32Unchecked {
					e.MovEaxFromRaxPtr()
				} else if instr.Kind == ir.IRIndexLoadU16 || instr.Kind == ir.IRIndexLoadU16Unchecked {
					e.MovzxEaxWordPtrRax()
				} else {
					e.MovzxEaxBytePtrRax()
				}
				stackBeforePush := stackDepth
				e.PushRax()
				push(1)
				if checked {
					doneAt := e.JmpRel32()
					failOff := len(e.Buf)
					if err := abi.EmitExit(e, 1, stackBeforePush, importPatches); err != nil {
						return err
					}
					doneOff := len(e.Buf)
					if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
						return err
					}
					if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
						return err
					}
				}
			case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
				if err := pop(4); err != nil {
					return err
				}
				e.PopR8()
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.CmpEdxEcx()
				failAt := e.JaeRel32()
				if instr.Kind == ir.IRIndexStoreI32 {
					e.ShlRdxImm8(2)
				} else if instr.Kind == ir.IRIndexStoreU16 {
					e.ShlRdxImm8(1)
				}
				e.AddRaxRdx()
				if instr.Kind == ir.IRIndexStoreI32 {
					e.MovMem32RaxPtrR8d()
				} else if instr.Kind == ir.IRIndexStoreU16 {
					e.MovMem16RaxPtrR8w()
				} else {
					e.MovMem8RaxPtrR8b()
				}
				doneAt := e.JmpRel32()
				failOff := len(e.Buf)
				if err := abi.EmitExit(e, 1, stackDepth, importPatches); err != nil {
					return err
				}
				doneOff := len(e.Buf)
				if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
					return err
				}
				if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
					return err
				}
			case ir.IRIslandNew:
				if err := abi.EmitIslandNew(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
				if err := abi.EmitIslandMakeSlice(e, instr.Kind, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRIslandFree:
				if err := abi.EmitIslandFree(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRCapIO:
				e.MovEaxImm32(0xC10)
				e.PushRax()
				push(1)
			case ir.IRCapMem:
				e.MovEaxImm32(0xC11)
				e.PushRax()
				push(1)
			case ir.IRMemReadI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRMemWriteI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovMem32RdiDispR8d(0)
				e.PushR8()
				push(1)
			case ir.IRMemReadU8:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovzxEaxBytePtrRax()
				e.PushRax()
				push(1)
			case ir.IRMemWriteU8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovMem8RaxPtrR8b()
				e.PushR8()
				push(1)
			case ir.IRMemReadPtr:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitPointerLoad()
				e.PushRax()
				push(1)
			case ir.IRMemWritePtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitPointerStore()
				e.PushR8()
				push(1)
			case ir.IRMemWriteArchPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(registerWidthBytes); err != nil {
					return err
				}
				emitArchPointerStore()
				e.PushR8()
				push(1)
			case ir.IRAtomicLoadPtr:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitPointerLoad()
				e.PushRax()
				push(1)
			case ir.IRAtomicStorePtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitAtomicPointerStore()
				e.PushR9()
				push(1)
			case ir.IRAtomicExchangePtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitAtomicPointerExchange()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAddPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitAtomicPointerFetchAdd()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchSubPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitAtomicPointerFetchSub()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAndPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				if err := emitAtomicPointerFetchCASLoop(e.AndR10dR8d, e.AndR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchOrPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				if err := emitAtomicPointerFetchCASLoop(e.OrR10dR8d, e.OrR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchXorPtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				if err := emitAtomicPointerFetchCASLoop(e.XorR10dR8d, e.XorR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicCompareExchangePtr:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopR9()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitAtomicPointerCompareExchange()
				e.PushRax()
				push(1)
			case ir.IRAtomicFenceSeqCst:
				e.Mfence()
			case ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
				ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
				// x86-family TSO gives acquire/release fence semantics without
				// a hardware fence; seq_cst remains the explicit mfence case.
			case ir.IRAtomicLoadI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRAtomicStoreI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovR9R8()
				e.XchgMem32RdiPtrR8d()
				e.PushR9()
				push(1)
			case ir.IRAtomicExchangeI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.XchgMem32RdiPtrR8d()
				e.PushR8()
				push(1)
			case ir.IRAtomicCompareExchangeI32:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopR9()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				emitAtomicI32CompareExchange()
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchAddI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.LockXaddMem32RdiPtrR8d()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchSubI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.NegR8d()
				e.LockXaddMem32RdiPtrR8d()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAndI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				if err := emitAtomicI32FetchCASLoop(e.AndR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchOrI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				if err := emitAtomicI32FetchCASLoop(e.OrR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchXorI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(4); err != nil {
					return err
				}
				if err := emitAtomicI32FetchCASLoop(e.XorR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicLoadI64:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovRaxFromRdiDisp(0)
				e.PushRax()
				push(1)
			case ir.IRAtomicStoreI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovR9R8()
				e.XchgMem64RdiPtrR8()
				e.PushR9()
				push(1)
			case ir.IRAtomicExchangeI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				e.MovRdiRax()
				e.XchgMem64RdiPtrR8()
				e.PushR8()
				push(1)
			case ir.IRAtomicCompareExchangeI64:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopR9()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				emitAtomicI64CompareExchange()
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchAddI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				e.MovRdiRax()
				e.LockXaddMem64RdiPtrR8()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchSubI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				e.MovRdiRax()
				e.NegR8()
				e.LockXaddMem64RdiPtrR8()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAndI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				if err := emitAtomicI64FetchCASLoop(e.AndR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchOrI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				if err := emitAtomicI64FetchCASLoop(e.OrR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchXorI64:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(8); err != nil {
					return err
				}
				if err := emitAtomicI64FetchCASLoop(e.XorR10R8); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicLoadI8:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovzxEaxBytePtrRax()
				e.PushRax()
				push(1)
			case ir.IRAtomicStoreI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovzxR8dR8b()
				e.MovR9R8()
				e.XchgMem8RdiPtrR8b()
				e.PushR9()
				push(1)
			case ir.IRAtomicExchangeI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovRdiRax()
				e.XchgMem8RdiPtrR8b()
				e.MovzxR8dR8b()
				e.PushR8()
				push(1)
			case ir.IRAtomicCompareExchangeI8:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopR9()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				emitAtomicI8CompareExchange()
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchAddI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovRdiRax()
				e.LockXaddMem8RdiPtrR8b()
				e.MovzxR8dR8b()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchSubI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				e.MovRdiRax()
				e.NegR8b()
				e.LockXaddMem8RdiPtrR8b()
				e.MovzxR8dR8b()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAndI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				if err := emitAtomicI8FetchCASLoop(e.AndR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchOrI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				if err := emitAtomicI8FetchCASLoop(e.OrR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchXorI8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(1); err != nil {
					return err
				}
				if err := emitAtomicI8FetchCASLoop(e.XorR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicLoadI16:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				e.MovzxEaxWordPtrRax()
				e.PushRax()
				push(1)
			case ir.IRAtomicStoreI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovzxR8dR8w()
				e.MovR9R8()
				e.XchgMem16RdiPtrR8w()
				e.PushR9()
				push(1)
			case ir.IRAtomicExchangeI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				e.MovRdiRax()
				e.XchgMem16RdiPtrR8w()
				e.MovzxR8dR8w()
				e.PushR8()
				push(1)
			case ir.IRAtomicCompareExchangeI16:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopR9()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				emitAtomicI16CompareExchange()
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchAddI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				e.MovRdiRax()
				e.LockXaddMem16RdiPtrR8w()
				e.MovzxR8dR8w()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchSubI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				e.MovRdiRax()
				e.NegR8w()
				e.LockXaddMem16RdiPtrR8w()
				e.MovzxR8dR8w()
				e.PushR8()
				push(1)
			case ir.IRAtomicFetchAndI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				if err := emitAtomicI16FetchCASLoop(e.AndR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchOrI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				if err := emitAtomicI16FetchCASLoop(e.OrR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRAtomicFetchXorI16:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopR8()
				e.PopRax()
				if err := guardAllocationBaseRawAccess(2); err != nil {
					return err
				}
				if err := emitAtomicI16FetchCASLoop(e.XorR10dR8d); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRMemReadI32Offset:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(4); err != nil {
					return err
				}
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRMemWriteI32Offset:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRcx()
				e.PopR8()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(4); err != nil {
					return err
				}
				e.MovRdiRax()
				e.MovMem32RdiDispR8d(0)
				e.PushR8()
				push(1)
			case ir.IRMemReadU8Offset:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(1); err != nil {
					return err
				}
				e.MovzxEaxBytePtrRax()
				e.PushRax()
				push(1)
			case ir.IRMemWriteU8Offset:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRcx()
				e.PopR8()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(1); err != nil {
					return err
				}
				e.MovMem8RaxPtrR8b()
				e.PushR8()
				push(1)
			case ir.IRMemReadPtrOffset:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitPointerLoad()
				e.PushRax()
				push(1)
			case ir.IRMemWritePtrOffset:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRcx()
				e.PopR8()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(pointerWidthBytes); err != nil {
					return err
				}
				emitPointerStore()
				e.PushR8()
				push(1)
			case ir.IRMemWriteArchPtrOffset:
				if err := pop(4); err != nil {
					return err
				}
				e.PopRcx()
				e.PopR8()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(registerWidthBytes); err != nil {
					return err
				}
				emitArchPointerStore()
				e.PushR8()
				push(1)
			case ir.IRPtrAdd:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRdx()
				e.PopRax()
				if err := guardAllocationOffsetRawAccess(1); err != nil {
					return err
				}
				e.PushRax()
				push(1)
			case ir.IRMmioReadI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRMmioWriteI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.MovMem32RaxPtrEcx()
				e.PushRcx()
				push(1)
			case ir.IRSymAddr:
				if instr.Name == "" {
					return fmt.Errorf("x64 backend: symbol address is missing name in function '%s'", fn.Name)
				}
				leaPos := e.LeaRaxRipDisp()
				*callPatches = append(*callPatches, x64obj.CallPatch{At: leaPos, Name: instr.Name, Kind: x64obj.PatchFuncAddrRel32})
				e.PushRax()
				push(1)
			case ir.IRCtxSwitch:
				if err := pop(3); err != nil {
					return err
				}

				switch abi.(type) {
				case *x64abi.SysVUnix:
					e.PopR8()  // cap.mem (unused)
					e.PopRsi() // to_rsp_slot
					e.PopRdi() // from_rsp_slot
				case *x64abi.Win64:
					e.PopR8()  // cap.mem (unused)
					e.PopRdx() // to_rsp_slot
					e.PopRcx() // from_rsp_slot
				default:
					return fmt.Errorf("ctx_switch: unsupported ABI")
				}

				switchLabel := newInternalLabel()
				contLabel := newInternalLabel()

				if _, ok := abi.(*x64abi.Win64); ok {
					e.SubRspImm32(32)
				}
				callAt := e.CallRel32()
				patches = append(patches, labelPatch{at: callAt, label: switchLabel})

				if _, ok := abi.(*x64abi.Win64); ok {
					e.AddRspImm32(32)
				}
				e.XorEaxEax()
				e.PushRax()
				push(1)
				jmpAt := e.JmpRel32()
				patches = append(patches, labelPatch{at: jmpAt, label: contLabel})

				labelOffsets[switchLabel] = len(e.Buf)
				switch abi.(type) {
				case *x64abi.SysVUnix:
					e.PushRbx()
					e.PushRbp()
					e.PushR12()
					e.PushR13()
					e.PushR14()
					e.PushR15()
					e.MovMem64RdiDispRsp(0)
					e.MovRdiRsi()
					e.MovRspFromRdiDisp(0)
					e.PopR15()
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopRbp()
					e.PopRbx()
					e.Ret()
				case *x64abi.Win64:
					e.PushRbx()
					e.PushRbp()
					e.PushRdi()
					e.PushRsi()
					e.PushR12()
					e.PushR13()
					e.PushR14()
					e.PushR15()
					e.MovRdiRcx()
					e.MovMem64RdiDispRsp(0)
					e.MovRdiRdx()
					e.MovRspFromRdiDisp(0)
					e.PopR15()
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopRsi()
					e.PopRdi()
					e.PopRbp()
					e.PopRbx()
					e.Ret()
				}

				labelOffsets[contLabel] = len(e.Buf)
			default:
				return fmt.Errorf("unsupported IR instruction")
			}
		}

		if len(smallHeapCalls) > 0 {
			helperOff := len(e.Buf)
			stateIndex := ensureSmallHeapState()
			if err := emitSmallHeapAllocatorHelper(e, abi, stackDepth, importPatches, leaPatches, stateIndex); err != nil {
				return err
			}
			for _, at := range smallHeapCalls {
				if err := x64.PatchRel32(e.Buf, at, helperOff); err != nil {
					return err
				}
			}
		}

		for _, patch := range patches {
			target, ok := labelOffsets[patch.label]
			if !ok {
				return fmt.Errorf("unknown label %d", patch.label)
			}
			if err := x64.PatchRel32(e.Buf, patch.at, target); err != nil {
				return err
			}
		}

		return nil
	}
}

func emitSmallHeapMakeSliceEnabled(abi x64abi.ABI, opt x64.CodegenOptions, pointerWidthBytes int32) bool {
	if !opt.EnableSmallHeap || opt.DisableSmallHeap || pointerWidthBytes != 8 {
		return false
	}
	sysv, ok := abi.(*x64abi.SysVUnix)
	return ok && sysv.SysMmap == 9 && sysv.SysExit == 60
}

func emitFunctionTempRegionMakeSlice(
	e *x64.Emitter,
	abi x64abi.ABI,
	kind ir.IRInstrKind,
	stackDepth *int,
	baseOffset int32,
	sizeOffset int32,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = importPatches
	sysv, ok := abi.(*x64abi.SysVUnix)
	if !ok || sysv.SysMmap != 9 || sysv.SysMunmap != 11 || sysv.SysExit != 60 {
		return fmt.Errorf("function-temp region lowering: unsupported ABI")
	}
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in function-temp region make_slice")
	}
	*stackDepth--
	e.PopRax()
	e.TestRaxRax()
	negativeAt := e.JlRel32()
	emptyAt := e.JzRel32()
	overflowAt := -1
	if max := functionTempRegionSliceMaxElements(kind); max != functionTempRegionSliceMaxElements(ir.IRRegionMakeSliceU8) {
		e.CmpRaxImm32(max)
		overflowAt = e.JgRel32()
	}
	e.PushRax()
	*stackDepth++
	switch kind {
	case ir.IRRegionMakeSliceI32:
		e.ShlRaxImm8(2)
	case ir.IRRegionMakeSliceU16:
		e.ShlRaxImm8(1)
	}
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(false)
	e.CmpRaxImm32(cfg.MaxPayloadBytes)
	capacityAt := e.JgRel32()
	e.AddRaxImm32(cfg.HeaderBytes)
	e.PushRax()
	*stackDepth++
	e.MovRsiRax()
	e.MovEdiImm32(0)
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(sysv.SysMmap)
	e.Syscall()
	if err := emitSysVMmapFailureGuard(e, sysv, *stackDepth); err != nil {
		return err
	}
	*stackDepth--
	e.PopRcx()
	e.MovMem64RbpDispRcx(sizeOffset)
	e.MovMem64RbpDispRax(baseOffset)
	e.AddRaxImm32(cfg.HeaderBytes)
	*stackDepth--
	e.PopRcx()
	e.PushRax()
	*stackDepth++
	e.PushRcx()
	*stackDepth++
	doneAt := e.JmpRel32()

	lengthFailOff := len(e.Buf)
	if err := sysv.EmitExit(e, 2, *stackDepth, nil); err != nil {
		return err
	}
	emptyOff := len(e.Buf)
	e.MovEaxImm32(0)
	e.PushRax()
	e.PushRax()
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, negativeAt, lengthFailOff); err != nil {
		return err
	}
	if overflowAt >= 0 {
		if err := x64.PatchRel32(e.Buf, overflowAt, lengthFailOff); err != nil {
			return err
		}
	}
	if err := x64.PatchRel32(e.Buf, capacityAt, lengthFailOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func emitFunctionTempRegionReset(
	e *x64.Emitter,
	abi x64abi.ABI,
	stackDepth *int,
	baseOffset int32,
	sizeOffset int32,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = stackDepth
	_ = importPatches
	sysv, ok := abi.(*x64abi.SysVUnix)
	if !ok || sysv.SysMunmap != 11 || sysv.SysExit != 60 {
		return fmt.Errorf("function-temp region reset: unsupported ABI")
	}
	e.MovRaxFromRbpDisp(baseOffset)
	e.TestRaxRax()
	doneAt := e.JzRel32()
	e.MovRdiRax()
	e.MovRaxFromRbpDisp(sizeOffset)
	e.MovRsiRax()
	e.MovEaxImm32(sysv.SysMunmap)
	e.Syscall()
	e.MovMem64RbpDispImm(baseOffset, 0)
	e.MovMem64RbpDispImm(sizeOffset, 0)
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func emitSysVMmapFailureGuard(e *x64.Emitter, abi *x64abi.SysVUnix, stackSlots int) error {
	e.CmpRaxImm32(-4095)
	failAt := e.JaeRel32()
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := abi.EmitExit(e, 2, stackSlots, nil); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func emitSmallHeapMakeSlice(
	e *x64.Emitter,
	kind ir.IRInstrKind,
	stackDepth *int,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
	smallHeapCalls *[]int,
	stateIndex int,
) error {
	_ = stateIndex
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in make_slice")
	}
	*stackDepth--
	e.PopRax()
	e.TestRaxRax()
	negativeAt := e.JlRel32()
	emptyAt := e.JzRel32()
	overflowAt := -1
	if max := smallHeapMakeSliceMaxElements(kind); max != smallHeapMaxI32AllocationBytes {
		e.CmpRaxImm32(max)
		overflowAt = e.JgRel32()
	}
	e.PushRax()
	*stackDepth++
	if kind == ir.IRMakeSliceI32 {
		e.ShlRaxImm8(2)
	} else if kind == ir.IRMakeSliceU16 {
		e.ShlRaxImm8(1)
	}
	e.MovRsiRax()
	callAt := e.CallRel32()
	*smallHeapCalls = append(*smallHeapCalls, callAt)
	*stackDepth--
	e.PopRcx()
	e.PushRax()
	*stackDepth++
	e.PushRcx()
	*stackDepth++
	doneAt := e.JmpRel32()

	lengthFailOff := len(e.Buf)
	if err := abi.EmitExit(e, smallHeapAllocationLengthTrapExitCode, 0, importPatches); err != nil {
		return err
	}
	emptyOff := len(e.Buf)
	e.MovEaxImm32(0)
	e.PushRax()
	e.PushRax()
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, negativeAt, lengthFailOff); err != nil {
		return err
	}
	if overflowAt >= 0 {
		if err := x64.PatchRel32(e.Buf, overflowAt, lengthFailOff); err != nil {
			return err
		}
	}
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func emitSmallHeapAllocatorHelper(
	e *x64.Emitter,
	abi x64abi.ABI,
	stackDepth int,
	importPatches *[]x64obj.ImportPatch,
	leaPatches *[]x64obj.LeaPatch,
	stateIndex int,
) error {
	sysv, ok := abi.(*x64abi.SysVUnix)
	if !ok {
		return fmt.Errorf("small heap allocator requires SysV ABI")
	}
	e.MovRaxRsi()
	e.CmpRaxImm32(runtimeabi.SmallHeapMaxSmallBytes)
	largeAt := e.JgRel32()

	e.AddRsiImm32(runtimeabi.SmallHeapAlignment - 1)
	e.AndRsiImm32(-runtimeabi.SmallHeapAlignment)
	leaPos := e.LeaRaxRipDisp()
	*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: stateIndex})
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(0)
	e.MovR8FromRdiDisp(8)
	e.TestRaxRax()
	emptyAt := e.JzRel32()
	e.MovRdxRax()
	e.AddRdxRsi()
	e.CmpRdxR8()
	fullAt := e.JaRel32()
	e.MovMem64RdiDispRdx(0)
	e.Ret()

	refillOff := len(e.Buf)
	e.PushRsi()
	e.PushRdi()
	e.MovEdiImm32(0)
	e.MovEaxImm32(runtimeabi.SmallHeapChunkBytes)
	e.MovRsiRax()
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(sysv.SysMmap)
	e.Syscall()
	if err := emitSmallHeapMmapFailureGuard(e, abi, stackDepth, importPatches); err != nil {
		return err
	}
	e.PopRdi()
	e.PopRsi()
	e.MovRdxRax()
	e.AddRdxRsi()
	e.MovMem64RdiDispRdx(0)
	e.MovRdxRax()
	e.AddRdxImm32(runtimeabi.SmallHeapChunkBytes)
	e.MovMem64RdiDispRdx(8)
	e.Ret()

	largeOff := len(e.Buf)
	e.MovEdiImm32(0)
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(sysv.SysMmap)
	e.Syscall()
	if err := emitSmallHeapMmapFailureGuard(e, abi, stackDepth, importPatches); err != nil {
		return err
	}
	e.Ret()

	if err := x64.PatchRel32(e.Buf, largeAt, largeOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, emptyAt, refillOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, fullAt, refillOff); err != nil {
		return err
	}
	return nil
}

func emitSmallHeapMmapFailureGuard(e *x64.Emitter, abi x64abi.ABI, stackDepth int, importPatches *[]x64obj.ImportPatch) error {
	e.CmpRaxImm32(-4095)
	failAt := e.JaeRel32()
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := abi.EmitExit(e, 2, stackDepth, importPatches); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

const (
	smallHeapAllocationLengthTrapExitCode int32 = 2
	smallHeapMaxI32AllocationBytes        int32 = 1<<31 - 1
)

func smallHeapMakeSliceMaxElements(kind ir.IRInstrKind) int32 {
	switch kind {
	case ir.IRMakeSliceU16:
		return smallHeapMaxI32AllocationBytes / 2
	case ir.IRMakeSliceI32:
		return smallHeapMaxI32AllocationBytes / 4
	default:
		return smallHeapMaxI32AllocationBytes
	}
}

func rawSliceMaxElements(shift int32) int32 {
	if shift <= 0 {
		return smallHeapMaxI32AllocationBytes
	}
	if shift >= 30 {
		return 0
	}
	return smallHeapMaxI32AllocationBytes >> shift
}

func emitSliceView(
	e *x64.Emitter,
	kind ir.IRInstrKind,
	shift byte,
	pop func(int) error,
	push func(int),
	stackDepth *int,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
) error {
	failPatches := []int{}
	switch kind {
	case ir.IRSliceWindow:
		if err := pop(4); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRbx() // count
		e.PopRdx() // start
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEdxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEbxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEdxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.SubEcxEdx()
		e.CmpEbxEcx()
		failPatches = append(failPatches, e.JgRel32())
		if shift > 0 {
			e.ShlRdxImm8(shift)
		}
		e.AddRaxRdx()
		e.PushRax()
		e.PushRbx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	case ir.IRSlicePrefix:
		if err := pop(3); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRbx() // count
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEbxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEbxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.PushRax()
		e.PushRbx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	case ir.IRSliceSuffix:
		if err := pop(3); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRdx() // start
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEdxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEdxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.SubEcxEdx()
		if shift > 0 {
			e.ShlRdxImm8(shift)
		}
		e.AddRaxRdx()
		e.PushRax()
		e.PushRcx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	default:
		return fmt.Errorf("x64 backend: unsupported slice view kind %v", kind)
	}
}

func patchSliceViewFailure(e *x64.Emitter, failPatches []int, failStackDepth int, abi x64abi.ABI, importPatches *[]x64obj.ImportPatch) error {
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := abi.EmitExit(e, 1, failStackDepth, importPatches); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	for _, at := range failPatches {
		if err := x64.PatchRel32(e.Buf, at, failOff); err != nil {
			return err
		}
	}
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}
