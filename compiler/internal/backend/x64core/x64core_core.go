package x64core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path"
	"strconv"
	"strings"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	x64bounds "tetra_language/compiler/internal/backend/x64core/bounds"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/runtimeabi"
)

// ---- allocation_loop_register.go ----

func emitAllocationLoopRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.AllocationLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.BackingLocal + plan.BackingSlots - 1))
	e.MovR9Rax()
	e.MovR8dImm32(uint32(plan.SliceLength))
	e.MovR10dImm32(0)
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.CmpRcxImm32(plan.LoopBound)
	exitAt := e.JgeRel32()
	if err := emitAllocationLoopCheckedIndexZero(e, abi, importPatches, plan); err != nil {
		return true, err
	}
	e.MovRaxR9()
	e.MovMem32RaxPtrEcx()
	if err := emitAllocationLoopCheckedIndexZero(e, abi, importPatches, plan); err != nil {
		return true, err
	}
	e.MovRaxR9()
	e.MovEaxFromRaxPtr()
	e.MovEdxEax()
	emitAddR10dEdx(e)
	e.AddEcxImm8(byte(plan.Step))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}

	e.MovEaxR10d()
	e.CmpEaxImm32(0)
	successAt := e.JgRel32()
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitAllocationLoopCheckedIndexZero(
	e *x64.Emitter,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
	plan machine.AllocationLoopPlan,
) error {
	e.MovEdxImm32(uint32(plan.IndexConst))
	e.CmpEdxImm32(plan.SliceLength)
	failAt := e.JaeRel32()
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := abi.EmitExit(e, 1, 0, importPatches); err != nil {
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

// ---- region_island_allocation_main_register.go ----

func emitRegionIslandAllocationMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.RegionIslandAllocationMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.SlicePtrLocal))
	e.MovR9Rax()
	e.MovR10dImm32(0)
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.CmpRcxImm32(plan.LoopBound)
	exitAt := e.JgeRel32()
	e.MovRaxR9()
	e.MovMem32RaxPtrEcx()
	e.MovRaxR9()
	e.MovEaxFromRaxPtr()
	e.MovEdxEax()
	emitAddR10dEdx(e)
	e.AddEcxImm8(byte(plan.Step))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}

	e.MovEaxR10d()
	e.CmpEaxImm32(0)
	successAt := e.JgRel32()
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- call_register.go ----

type scalarCallABIKind int

const (
	scalarCallABISysV scalarCallABIKind = iota + 1
	scalarCallABIWin64
)

func scalarCallABIFromBackendABI(abi x64abi.ABI) (scalarCallABIKind, machine.CallABIInfo, bool) {
	switch abi.(type) {
	case *x64abi.SysVUnix:
		return scalarCallABISysV, machine.SysVCallABIInfo(), true
	case *x64abi.Win64:
		return scalarCallABIWin64, machine.Win64CallABIInfo(), true
	default:
		return 0, machine.CallABIInfo{}, false
	}
}

func irFuncHasCall(fn ir.IRFunc) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall {
			return true
		}
	}
	return false
}

func emitScalarRegisterCall(
	e *x64.Emitter,
	kind scalarCallABIKind,
	instr ir.IRInstr,
	depth *int,
	scratchOffset func(int) int32,
	callPatches *[]x64obj.CallPatch,
) error {
	if e == nil || depth == nil || scratchOffset == nil || callPatches == nil {
		return fmt.Errorf("x64 scalar register backend: missing call emission state")
	}
	if instr.Name == "" {
		return fmt.Errorf("x64 scalar register backend: call is missing target name")
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf(
			"x64 scalar register backend: call %q has negative ABI slots args=%d rets=%d",
			instr.Name,
			instr.ArgSlots,
			instr.RetSlots,
		)
	}
	if instr.RetSlots > 1 {
		return fmt.Errorf(
			"x64 scalar register backend: call %q has unsupported register return slots %d",
			instr.Name,
			instr.RetSlots,
		)
	}
	maxArgs := scalarRegisterCallMaxArgs(kind)
	if instr.ArgSlots > maxArgs {
		return fmt.Errorf(
			"x64 scalar register backend: call %q has unsupported register arg slots %d (max=%d)",
			instr.Name,
			instr.ArgSlots,
			maxArgs,
		)
	}
	if *depth < instr.ArgSlots {
		return fmt.Errorf("x64 scalar register backend: stack underflow in call to %q", instr.Name)
	}

	argBase := *depth - instr.ArgSlots
	for i := 0; i < instr.ArgSlots; i++ {
		e.MovRaxFromRbpDisp(scratchOffset(argBase + i))
		emitMoveRaxToScalarCallArg(e, kind, i)
	}
	*depth -= instr.ArgSlots
	emitScalarCallFramePrologue(e, kind)
	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: instr.Name})
	emitScalarCallFrameEpilogue(e, kind)
	if instr.RetSlots == 1 {
		e.MovMem64RbpDispRax(scratchOffset(*depth))
		*depth++
	}
	return nil
}

func scalarRegisterCallMaxArgs(kind scalarCallABIKind) int {
	switch kind {
	case scalarCallABIWin64:
		return 4
	default:
		return 6
	}
}

func emitMoveRaxToScalarCallArg(e *x64.Emitter, kind scalarCallABIKind, arg int) {
	if kind == scalarCallABIWin64 {
		switch arg {
		case 0:
			e.MovRcxRax()
		case 1:
			e.MovRdxRax()
		case 2:
			e.MovR8Rax()
		case 3:
			e.MovR9Rax()
		}
		return
	}
	switch arg {
	case 0:
		e.MovRdiRax()
	case 1:
		e.MovRsiRax()
	case 2:
		e.MovRdxRax()
	case 3:
		e.MovRcxRax()
	case 4:
		e.MovR8Rax()
	case 5:
		e.MovR9Rax()
	}
}

func emitScalarCallFramePrologue(e *x64.Emitter, kind scalarCallABIKind) {
	if kind == scalarCallABIWin64 {
		e.SubRspImm32(32)
	}
}

func emitScalarCallFrameEpilogue(e *x64.Emitter, kind scalarCallABIKind) {
	if kind == scalarCallABIWin64 {
		e.AddRspImm32(32)
	}
}

func emitScalarLoopCall(
	e *x64.Emitter,
	kind scalarCallABIKind,
	name string,
	callPatches *[]x64obj.CallPatch,
) error {
	if e == nil || callPatches == nil {
		return fmt.Errorf("x64 scalar call-loop backend: missing call emission state")
	}
	if name == "" {
		return fmt.Errorf("x64 scalar call-loop backend: call is missing target name")
	}
	emitScalarCallFramePrologue(e, kind)
	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: name})
	emitScalarCallFrameEpilogue(e, kind)
	return nil
}

// ---- emit.go ----

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
		case ir.IRRegionEnter,
			ir.IRRegionMakeSliceU8,
			ir.IRRegionMakeSliceU16,
			ir.IRRegionMakeSliceI32,
			ir.IRRegionReset:
			return true
		}
	}
	return false
}

func NewEmitFunc(abi x64abi.ABI) x64obj.EmitFunc {
	smallHeapStateDataIndex := -1
	var runtimeHeapTelemetry *runtimeHeapTelemetryState
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
		ensureRuntimeHeapTelemetryState := func() (*runtimeHeapTelemetryState, error) {
			if runtimeHeapTelemetry != nil {
				return runtimeHeapTelemetry, nil
			}
			blob, layout, err := buildRuntimeHeapTelemetryBlob(opt)
			if err != nil {
				return nil, err
			}
			*dataBlobs = append(*dataBlobs, blob)
			runtimeHeapTelemetry = &runtimeHeapTelemetryState{
				dataIndex: len(*dataBlobs) - 1,
				layout:    layout,
			}
			return runtimeHeapTelemetry, nil
		}
		emitMainRuntimeHeapTelemetryFlush := runtimeHeapTelemetryFlushFunc(func() error {
			if !opt.EmitRuntimeHeapTelemetry || fn.Name != opt.RuntimeHeapTelemetryMain {
				return nil
			}
			telemetry, err := ensureRuntimeHeapTelemetryState()
			if err != nil {
				return err
			}
			return emitRuntimeHeapTelemetryFlush(e, abi, leaPatches, callPatches, telemetry)
		})

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
				return 0, fmt.Errorf(
					"x64 backend: local slot %d out of bounds in function '%s' (locals=%d)",
					slot,
					fn.Name,
					fn.LocalSlots,
				)
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
		atomicOps := emitAtomicInstrOps{
			pointerWidthBytes:                pointerWidthBytes,
			pop:                              pop,
			push:                             push,
			guardAllocationBaseRawAccess:     guardAllocationBaseRawAccess,
			emitPointerLoad:                  emitPointerLoad,
			emitAtomicPointerStore:           emitAtomicPointerStore,
			emitAtomicPointerExchange:        emitAtomicPointerExchange,
			emitAtomicPointerFetchAdd:        emitAtomicPointerFetchAdd,
			emitAtomicPointerFetchSub:        emitAtomicPointerFetchSub,
			emitAtomicPointerFetchCASLoop:    emitAtomicPointerFetchCASLoop,
			emitAtomicPointerCompareExchange: emitAtomicPointerCompareExchange,
			emitAtomicI32CompareExchange:     emitAtomicI32CompareExchange,
			emitAtomicI32FetchCASLoop:        emitAtomicI32FetchCASLoop,
			emitAtomicI64CompareExchange:     emitAtomicI64CompareExchange,
			emitAtomicI64FetchCASLoop:        emitAtomicI64FetchCASLoop,
			emitAtomicI8CompareExchange:      emitAtomicI8CompareExchange,
			emitAtomicI8FetchCASLoop:         emitAtomicI8FetchCASLoop,
			emitAtomicI16CompareExchange:     emitAtomicI16CompareExchange,
			emitAtomicI16FetchCASLoop:        emitAtomicI16FetchCASLoop,
		}

		if ok, err := emitMatrixMultiplyMainRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitRegionIslandAllocationMainRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitSliceSumMainRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitVectorSliceSumRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitVectorCopyU8RegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitVectorMapI32AddConstRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitVectorMemsetZeroU8RegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitScalarSliceSumRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := x64bounds.EmitRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitAllocationLoopRegisterFunction(
			e,
			fn,
			abi,
			importPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitRecursionBenchmarkRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitScalarCallLoopRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitParallelMapReduceMainRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitActorPingPongRuntimeCallRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitPostgreSQLFrameTypeAtRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitPostgreSQLInoutWriterRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitInoutWriterHelperSummaryRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitInoutWriterHelperSummaryCallerRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitPostgreSQLInoutWriterMainRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitHashTableMainRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitHashTableLookupRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitScalarConstModuloLoopRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitScalarLoopRegisterFunction(
			e,
			fn,
			abi,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
			return err
		}
		if ok, err := emitScalarRegisterFunction(
			e,
			fn,
			abi,
			callPatches,
			opt,
			emitMainRuntimeHeapTelemetryFlush,
		); ok ||
			err != nil {
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
				*leaPatches = append(
					*leaPatches,
					x64obj.LeaPatch{At: leaPos, DataIndex: len(*dataBlobs)},
				)
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
					return fmt.Errorf(
						"x64 backend: global slot %d out of bounds in function '%s'",
						instr.Local,
						fn.Name,
					)
				}
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovRaxFromRdiDisp(0)
				e.PushRax()
				push(1)
				*leaPatches = append(
					*leaPatches,
					x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local},
				)
			case ir.IRStoreGlobal:
				if instr.Local < 0 {
					return fmt.Errorf(
						"x64 backend: global slot %d out of bounds in function '%s'",
						instr.Local,
						fn.Name,
					)
				}
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovMem64RdiDispRax(0)
				*leaPatches = append(
					*leaPatches,
					x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local},
				)
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
					return fmt.Errorf(
						"x64 backend: negative label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
				}
				if _, exists := labelOffsets[instr.Label]; exists {
					return fmt.Errorf(
						"x64 backend: duplicate label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
				}
				labelOffsets[instr.Label] = len(e.Buf)
			case ir.IRJmp:
				if instr.Label < 0 {
					return fmt.Errorf(
						"x64 backend: negative label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
				}
				at := e.JmpRel32()
				patches = append(patches, labelPatch{at: at, label: instr.Label})
			case ir.IRJmpIfZero:
				if instr.Label < 0 {
					return fmt.Errorf(
						"x64 backend: negative label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
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
					return fmt.Errorf(
						"unsupported return slots %d in function %q",
						fn.ReturnSlots,
						fn.Name,
					)
				}
				if opt.EmitRuntimeHeapTelemetry && fn.Name == opt.RuntimeHeapTelemetryMain {
					telemetry, err := ensureRuntimeHeapTelemetryState()
					if err != nil {
						return err
					}
					if err := emitRuntimeHeapTelemetryFlush(
						e,
						abi,
						leaPatches,
						callPatches,
						telemetry,
					); err != nil {
						return err
					}
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
					var telemetry *runtimeHeapTelemetryState
					if opt.EmitRuntimeHeapTelemetry {
						var err error
						telemetry, err = ensureRuntimeHeapTelemetryState()
						if err != nil {
							return err
						}
					}
					if err := emitSmallHeapMakeSlice(
						e,
						instr.Kind,
						&stackDepth,
						abi,
						importPatches,
						&smallHeapCalls,
						stateIndex,
						telemetry,
						leaPatches,
					); err != nil {
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
				if err := emitFunctionTempRegionMakeSlice(
					e,
					abi,
					instr.Kind,
					&stackDepth,
					functionTempRegionBaseOffset,
					functionTempRegionSizeOffset,
					importPatches,
				); err != nil {
					return err
				}
			case ir.IRRegionReset:
				if functionTempRegionSlots == 0 {
					return fmt.Errorf("function-temp region reset without frame state")
				}
				if err := emitFunctionTempRegionReset(
					e,
					abi,
					&stackDepth,
					functionTempRegionBaseOffset,
					functionTempRegionSizeOffset,
					importPatches,
				); err != nil {
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
				if max := stackSliceMaxElements(instr.Kind); max != stackSliceMaxElements(
					ir.IRStackSliceU8,
				) {
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
				if err := emitSliceView(
					e,
					instr.Kind,
					byte(instr.Imm),
					pop,
					push,
					&stackDepth,
					abi,
					importPatches,
				); err != nil {
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
				checked := instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadU8 ||
					instr.Kind == ir.IRIndexLoadU16
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
			case ir.IRIslandReset:
				if err := abi.EmitIslandReset(e, &stackDepth, opt, importPatches); err != nil {
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
			case ir.IRAtomicLoadPtr, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
				ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr,
				ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr,
				ir.IRAtomicCompareExchangePtr,
				ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed,
				ir.IRAtomicFenceAcquire, ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel,
				ir.IRAtomicLoadI32, ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32,
				ir.IRAtomicCompareExchangeI32, ir.IRAtomicFetchAddI32,
				ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32,
				ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
				ir.IRAtomicLoadI64, ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
				ir.IRAtomicCompareExchangeI64, ir.IRAtomicFetchAddI64,
				ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
				ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64,
				ir.IRAtomicLoadI8, ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
				ir.IRAtomicCompareExchangeI8, ir.IRAtomicFetchAddI8,
				ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
				ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
				ir.IRAtomicLoadI16, ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16,
				ir.IRAtomicCompareExchangeI16, ir.IRAtomicFetchAddI16,
				ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16,
				ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16:
				if err := emitAtomicInstr(e, instr, atomicOps); err != nil {
					return err
				}
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
					return fmt.Errorf(
						"x64 backend: symbol address is missing name in function '%s'",
						fn.Name,
					)
				}
				leaPos := e.LeaRaxRipDisp()
				*callPatches = append(
					*callPatches,
					x64obj.CallPatch{At: leaPos, Name: instr.Name, Kind: x64obj.PatchFuncAddrRel32},
				)
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
			if err := emitSmallHeapAllocatorHelper(
				e,
				abi,
				stackDepth,
				importPatches,
				leaPatches,
				stateIndex,
			); err != nil {
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

// ---- emit_atomics.go ----

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

// ---- emit_heap.go ----

func emitSmallHeapMakeSliceEnabled(
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	pointerWidthBytes int32,
) bool {
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
	if max := functionTempRegionSliceMaxElements(kind); max != functionTempRegionSliceMaxElements(
		ir.IRRegionMakeSliceU8,
	) {
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
	telemetry *runtimeHeapTelemetryState,
	leaPatches *[]x64obj.LeaPatch,
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
	if err := emitRuntimeHeapTelemetryRecordAllocation(e, leaPatches, telemetry); err != nil {
		return err
	}
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

func emitSmallHeapMmapFailureGuard(
	e *x64.Emitter,
	abi x64abi.ABI,
	stackDepth int,
	importPatches *[]x64obj.ImportPatch,
) error {
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

// ---- emit_slice_views.go ----

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

func patchSliceViewFailure(
	e *x64.Emitter,
	failPatches []int,
	failStackDepth int,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
) error {
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

// ---- heap_telemetry.go ----

const (
	runtimeHeapTelemetrySchema = "tetra.runtime.heap_telemetry.v1"
	runtimeHeapTelemetryTarget = "linux-x64"
	runtimeHeapTelemetryMethod = "tetra_linux_x64_heap_telemetry_v1"

	runtimeHeapTelemetryActorSnapshotSymbol = "__tetra_actor_memory_snapshot"
	runtimeHeapTelemetryNumberWidth         = 20
	runtimeHeapTelemetryATFDCWD             = 0xffffff9c
	runtimeHeapTelemetryOpenAt              = 257
	runtimeHeapTelemetryClose               = 3
	runtimeHeapTelemetryOpenFlags           = 0x241
	runtimeHeapTelemetryOpenMode            = 0o644
)

const (
	runtimeHeapTelemetryCurrentOffset int32 = iota * 8
	runtimeHeapTelemetryPeakOffset
	runtimeHeapTelemetryTotalOffset
	runtimeHeapTelemetryCountOffset
	runtimeHeapTelemetryRequestedOffset
	runtimeHeapTelemetryReservedOffset
	runtimeHeapTelemetrySmallPathCountOffset
)

const (
	runtimeHeapTelemetryActorCountOffset              = runtimeHeapTelemetrySmallPathCountOffset + 8
	runtimeHeapTelemetryActorRecord0Offset            = runtimeHeapTelemetryActorCountOffset + 8
	runtimeHeapTelemetryActorRecordSize         int32 = 56
	runtimeHeapTelemetryActorMaxDomains               = 128
	runtimeHeapTelemetryActorRecordIDOffset           = 0
	runtimeHeapTelemetryActorCurrentOffset            = 8
	runtimeHeapTelemetryActorPeakOffset               = 16
	runtimeHeapTelemetryActorCopiedOffset             = 24
	runtimeHeapTelemetryActorByteBudgetOffset         = 32
	runtimeHeapTelemetryActorOverBudgetOffset         = 40
	runtimeHeapTelemetryActorBackpressureOffset       = 48
	runtimeHeapTelemetryHeaderSize                    = runtimeHeapTelemetryActorRecord0Offset + runtimeHeapTelemetryActorMaxDomains*runtimeHeapTelemetryActorRecordSize
)

type runtimeHeapTelemetryState struct {
	dataIndex int
	layout    runtimeHeapTelemetryLayout
}

type runtimeHeapTelemetryLayout struct {
	pathOffset          int32
	zeroJSONOffset      int32
	zeroJSONLength      uint32
	pathJSONOffset      int32
	pathJSONLength      uint32
	zeroNumbers         runtimeHeapTelemetryNumberOffsets
	pathNumbers         runtimeHeapTelemetryNumberOffsets
	zeroActorJSONOffset int32
	zeroActorJSONLength uint32
	zeroActorNumbers    runtimeHeapTelemetryNumberOffsets
	zeroActorTemplate   runtimeHeapTelemetryActorTemplate
	pathActorJSONOffset int32
	pathActorJSONLength uint32
	pathActorNumbers    runtimeHeapTelemetryNumberOffsets
	pathActorTemplate   runtimeHeapTelemetryActorTemplate
	actorDomains        bool
}

type runtimeHeapTelemetryNumberOffsets struct {
	current         int32
	peak            int32
	total           int32
	count           int32
	requested       int32
	reserved        int32
	domainRequested int32
	domainReserved  int32
	domainCurrent   int32
	domainPeak      int32
	smallPath       int32
}

type runtimeHeapTelemetryActorTemplate struct {
	entryOffset  int32
	entryLength  int32
	suffixOffset int32
	suffixLength int32
	numbers      []runtimeHeapTelemetryActorNumberOffsets
}

type runtimeHeapTelemetryActorNumberOffsets struct {
	current            int32
	peak               int32
	bytesCopied        int32
	mailboxCurrent     int32
	mailboxPeak        int32
	byteBudget         int32
	overBudgetCount    int32
	backpressureEvents int32
}

type runtimeHeapTelemetryNumberField struct {
	counterOffset int32
	fieldOffset   int32
}

type runtimeHeapTelemetryFlushFunc func() error

func (f runtimeHeapTelemetryFlushFunc) emit() error {
	if f == nil {
		return nil
	}
	return f()
}

func buildRuntimeHeapTelemetryBlob(
	opt x64.CodegenOptions,
) ([]byte, runtimeHeapTelemetryLayout, error) {
	program := strings.TrimSpace(opt.RuntimeHeapTelemetryProgram)
	if program == "" {
		return nil, runtimeHeapTelemetryLayout{}, fmt.Errorf(
			"runtime heap telemetry program is required",
		)
	}
	dir := strings.TrimSpace(opt.RuntimeHeapTelemetryDir)
	if dir == "" {
		return nil, runtimeHeapTelemetryLayout{}, fmt.Errorf(
			"runtime heap telemetry dir is required",
		)
	}
	sidecarPath := path.Join(strings.ReplaceAll(dir, "\\", "/"), program+".heap.json")

	zeroJSON, zeroNumbers := runtimeHeapTelemetryJSON(program, false)
	pathJSON, pathNumbers := runtimeHeapTelemetryJSON(program, true)

	blob := make([]byte, 0, 64+len(sidecarPath)+1+len(zeroJSON)+len(pathJSON))
	blob = append(blob, make([]byte, int(runtimeHeapTelemetryHeaderSize))...)
	layout := runtimeHeapTelemetryLayout{
		pathOffset: int32(len(blob)),
	}
	blob = append(blob, []byte(sidecarPath)...)
	blob = append(blob, 0)

	layout.zeroJSONOffset = int32(len(blob))
	layout.zeroJSONLength = uint32(len(zeroJSON))
	layout.zeroNumbers = zeroNumbers
	blob = append(blob, zeroJSON...)

	layout.pathJSONOffset = int32(len(blob))
	layout.pathJSONLength = uint32(len(pathJSON))
	layout.pathNumbers = pathNumbers
	blob = append(blob, pathJSON...)

	if opt.RuntimeHeapTelemetryActorDomains {
		zeroActorJSON, zeroActorNumbers, zeroActorTemplate := runtimeHeapTelemetryActorJSON(
			program,
			false,
		)
		pathActorJSON, pathActorNumbers, pathActorTemplate := runtimeHeapTelemetryActorJSON(
			program,
			true,
		)

		layout.zeroActorJSONOffset = int32(len(blob))
		layout.zeroActorJSONLength = uint32(len(zeroActorJSON))
		layout.zeroActorNumbers = zeroActorNumbers
		layout.zeroActorTemplate = zeroActorTemplate
		blob = append(blob, zeroActorJSON...)

		layout.pathActorJSONOffset = int32(len(blob))
		layout.pathActorJSONLength = uint32(len(pathActorJSON))
		layout.pathActorNumbers = pathActorNumbers
		layout.pathActorTemplate = pathActorTemplate
		blob = append(blob, pathActorJSON...)
		layout.actorDomains = true
	}

	return blob, layout, nil
}

func runtimeHeapTelemetryJSON(
	program string,
	includePaths bool,
) ([]byte, runtimeHeapTelemetryNumberOffsets) {
	var b bytes.Buffer
	numbers := runtimeHeapTelemetryNumberOffsets{smallPath: -1}
	placeholder := strings.Repeat(" ", runtimeHeapTelemetryNumberWidth)

	writeString := func(name string, value string, comma bool) {
		fmt.Fprintf(&b, "  %s: %s", strconv.Quote(name), strconv.Quote(value))
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	writeNumber := func(name string, comma bool) int32 {
		fmt.Fprintf(&b, "  %s: ", strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}
	writeIndentedNumber := func(indent string, name string, comma bool) int32 {
		fmt.Fprintf(&b, "%s%s: ", indent, strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}

	b.WriteString("{\n")
	writeString("schema", runtimeHeapTelemetrySchema, true)
	writeString("target", runtimeHeapTelemetryTarget, true)
	writeString("method", runtimeHeapTelemetryMethod, true)
	writeString("program", program, true)
	numbers.current = writeNumber("heap_current_bytes", true)
	numbers.peak = writeNumber("heap_peak_bytes", true)
	numbers.total = writeNumber("heap_total_alloc_bytes", true)
	numbers.count = writeNumber("heap_allocation_count", true)
	numbers.requested = writeNumber("bytes_requested", true)
	numbers.reserved = writeNumber("bytes_reserved", true)
	b.WriteString("  \"exit_status\": 0,\n")
	if includePaths {
		b.WriteString("  \"allocation_paths\": {\n")
		numbers.smallPath = writeIndentedNumber("    ", "small_heap_make_slice", false)
		b.WriteString("  },\n")
	}
	b.WriteString("  \"domain_bytes\": [\n")
	b.WriteString("    {\n")
	b.WriteString("      \"domain_id\": \"process\",\n")
	b.WriteString("      \"kind\": \"process\",\n")
	numbers.domainRequested = writeIndentedNumber("      ", "requested_bytes", true)
	numbers.domainReserved = writeIndentedNumber("      ", "reserved_bytes", true)
	numbers.domainCurrent = writeIndentedNumber("      ", "current_bytes", true)
	numbers.domainPeak = writeIndentedNumber("      ", "peak_bytes", false)
	b.WriteString("    }\n")
	b.WriteString("  ],\n")
	b.WriteString(
		"  \"notes\": [\"bytes_reserved is 0 because this sidecar counts Tetra heap allocation requests, not OS mmap reservations\"]\n",
	)
	b.WriteString("}\n")
	return b.Bytes(), numbers
}

func runtimeHeapTelemetryActorJSON(
	program string,
	includePaths bool,
) ([]byte, runtimeHeapTelemetryNumberOffsets, runtimeHeapTelemetryActorTemplate) {
	var b bytes.Buffer
	numbers := runtimeHeapTelemetryNumberOffsets{smallPath: -1}
	actorNumbers := make(
		[]runtimeHeapTelemetryActorNumberOffsets,
		runtimeHeapTelemetryActorMaxDomains,
	)
	placeholder := strings.Repeat(" ", runtimeHeapTelemetryNumberWidth)

	writeString := func(name string, value string, comma bool) {
		fmt.Fprintf(&b, "  %s: %s", strconv.Quote(name), strconv.Quote(value))
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	writeNumber := func(name string, comma bool) int32 {
		fmt.Fprintf(&b, "  %s: ", strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}
	writeIndentedNumber := func(indent string, name string, comma bool) int32 {
		fmt.Fprintf(&b, "%s%s: ", indent, strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}

	b.WriteString("{\n")
	writeString("schema", runtimeHeapTelemetrySchema, true)
	writeString("target", runtimeHeapTelemetryTarget, true)
	writeString("method", runtimeHeapTelemetryMethod, true)
	writeString("program", program, true)
	numbers.current = writeNumber("heap_current_bytes", true)
	numbers.peak = writeNumber("heap_peak_bytes", true)
	numbers.total = writeNumber("heap_total_alloc_bytes", true)
	numbers.count = writeNumber("heap_allocation_count", true)
	numbers.requested = writeNumber("bytes_requested", true)
	numbers.reserved = writeNumber("bytes_reserved", true)
	b.WriteString("  \"exit_status\": 0,\n")
	if includePaths {
		b.WriteString("  \"allocation_paths\": {\n")
		numbers.smallPath = writeIndentedNumber("    ", "small_heap_make_slice", false)
		b.WriteString("  },\n")
	}
	b.WriteString("  \"domain_bytes\": [\n")
	b.WriteString("    {\n")
	b.WriteString("      \"domain_id\": \"process\",\n")
	b.WriteString("      \"kind\": \"process\",\n")
	numbers.domainRequested = writeIndentedNumber("      ", "requested_bytes", true)
	numbers.domainReserved = writeIndentedNumber("      ", "reserved_bytes", true)
	numbers.domainCurrent = writeIndentedNumber("      ", "current_bytes", true)
	numbers.domainPeak = writeIndentedNumber("      ", "peak_bytes", false)
	b.WriteString("    }")

	template := runtimeHeapTelemetryActorTemplate{
		entryOffset: int32(b.Len()),
		numbers:     actorNumbers,
	}
	for i := 0; i < runtimeHeapTelemetryActorMaxDomains; i++ {
		entryStart := b.Len()
		b.WriteString(",\n")
		b.WriteString("    {\n")
		fmt.Fprintf(
			&b,
			"      \"domain_id\": %s,\n",
			strconv.Quote(fmt.Sprintf("domain:actor:%03d", i)),
		)
		b.WriteString("      \"kind\": \"actor\",\n")
		actorNumbers[i].current = writeIndentedNumber("      ", "current_bytes", true)
		actorNumbers[i].peak = writeIndentedNumber("      ", "peak_bytes", true)
		actorNumbers[i].bytesCopied = writeIndentedNumber("      ", "bytes_copied", true)
		actorNumbers[i].mailboxCurrent = writeIndentedNumber(
			"      ",
			"mailbox_current_bytes",
			true,
		)
		actorNumbers[i].mailboxPeak = writeIndentedNumber("      ", "mailbox_peak_bytes", true)
		actorNumbers[i].byteBudget = writeIndentedNumber("      ", "byte_budget", true)
		actorNumbers[i].overBudgetCount = writeIndentedNumber(
			"      ",
			"over_budget_count",
			true,
		)
		actorNumbers[i].backpressureEvents = writeIndentedNumber(
			"      ",
			"backpressure_events",
			false,
		)
		b.WriteString("    }")
		if i == 0 {
			template.entryLength = int32(b.Len() - entryStart)
		}
	}

	template.suffixOffset = int32(b.Len())
	b.WriteString("\n  ],\n")
	b.WriteString(
		"  \"notes\": [\"bytes_reserved is 0 because this sidecar counts Tetra heap allocation requests, not OS mmap reservations\"]\n",
	)
	b.WriteString("}\n")
	template.suffixLength = int32(b.Len()) - template.suffixOffset
	template.numbers = actorNumbers
	return b.Bytes(), numbers, template
}

func (o runtimeHeapTelemetryNumberOffsets) fields() []runtimeHeapTelemetryNumberField {
	fields := []runtimeHeapTelemetryNumberField{
		{counterOffset: runtimeHeapTelemetryCurrentOffset, fieldOffset: o.current},
		{counterOffset: runtimeHeapTelemetryPeakOffset, fieldOffset: o.peak},
		{counterOffset: runtimeHeapTelemetryTotalOffset, fieldOffset: o.total},
		{counterOffset: runtimeHeapTelemetryCountOffset, fieldOffset: o.count},
		{counterOffset: runtimeHeapTelemetryRequestedOffset, fieldOffset: o.requested},
		{counterOffset: runtimeHeapTelemetryReservedOffset, fieldOffset: o.reserved},
		{counterOffset: runtimeHeapTelemetryRequestedOffset, fieldOffset: o.domainRequested},
		{counterOffset: runtimeHeapTelemetryReservedOffset, fieldOffset: o.domainReserved},
		{counterOffset: runtimeHeapTelemetryCurrentOffset, fieldOffset: o.domainCurrent},
		{counterOffset: runtimeHeapTelemetryPeakOffset, fieldOffset: o.domainPeak},
	}
	if o.smallPath >= 0 {
		fields = append(fields, runtimeHeapTelemetryNumberField{
			counterOffset: runtimeHeapTelemetrySmallPathCountOffset,
			fieldOffset:   o.smallPath,
		})
	}
	return fields
}

func emitRuntimeHeapTelemetryRecordAllocation(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
) error {
	if state == nil {
		return nil
	}
	if leaPatches == nil {
		return fmt.Errorf("runtime heap telemetry: missing data patches")
	}
	e.PushRax()
	e.PushRdi()
	e.PushRdx()
	e.PushR8()
	e.PushRsi()

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()

	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCurrentOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryCurrentOffset)
	e.MovRdxFromRdiDisp(runtimeHeapTelemetryPeakOffset)
	e.CmpRdxRax()
	keepPeakAt := e.JaeRel32()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryPeakOffset)
	keepPeakOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepPeakAt, keepPeakOff); err != nil {
		return err
	}

	e.MovRaxFromRdiDisp(runtimeHeapTelemetryTotalOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryTotalOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryRequestedOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryRequestedOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCountOffset)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(runtimeHeapTelemetryCountOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetrySmallPathCountOffset)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(runtimeHeapTelemetrySmallPathCountOffset)

	e.PopRsi()
	e.PopR8()
	e.PopRdx()
	e.PopRdi()
	e.PopRax()
	return nil
}

func emitRuntimeHeapTelemetryFlush(
	e *x64.Emitter,
	abi x64abi.ABI,
	leaPatches *[]x64obj.LeaPatch,
	callPatches *[]x64obj.CallPatch,
	state *runtimeHeapTelemetryState,
) error {
	if state == nil {
		return nil
	}
	if leaPatches == nil {
		return fmt.Errorf("runtime heap telemetry: missing data patches")
	}
	sysv, ok := abi.(*x64abi.SysVUnix)
	if !ok || sysv.SysExit != 60 || sysv.SysWrite != 1 {
		return fmt.Errorf("runtime heap telemetry requires linux-x64 SysV ABI")
	}

	e.PushRax()
	if state.layout.actorDomains {
		if err := emitRuntimeHeapTelemetryCaptureActors(e, leaPatches, callPatches, state); err != nil {
			return err
		}
		if err := emitRuntimeHeapTelemetryFillTemplate(
			e,
			leaPatches,
			state,
			state.layout.zeroActorJSONOffset,
			state.layout.zeroActorNumbers,
		); err != nil {
			return err
		}
		if err := emitRuntimeHeapTelemetryFillActorTemplate(
			e,
			leaPatches,
			state,
			state.layout.zeroActorJSONOffset,
			state.layout.zeroActorTemplate,
		); err != nil {
			return err
		}
		if err := emitRuntimeHeapTelemetryFillTemplate(
			e,
			leaPatches,
			state,
			state.layout.pathActorJSONOffset,
			state.layout.pathActorNumbers,
		); err != nil {
			return err
		}
		if err := emitRuntimeHeapTelemetryFillActorTemplate(
			e,
			leaPatches,
			state,
			state.layout.pathActorJSONOffset,
			state.layout.pathActorTemplate,
		); err != nil {
			return err
		}
	} else {
		if err := emitRuntimeHeapTelemetryFillTemplate(
			e,
			leaPatches,
			state,
			state.layout.zeroJSONOffset,
			state.layout.zeroNumbers,
		); err != nil {
			return err
		}
		if err := emitRuntimeHeapTelemetryFillTemplate(
			e,
			leaPatches,
			state,
			state.layout.pathJSONOffset,
			state.layout.pathNumbers,
		); err != nil {
			return err
		}
	}

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCountOffset)
	e.TestRaxRax()
	useZeroAt := e.JzRel32()

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	if state.layout.actorDomains {
		e.AddRdxImm32(state.layout.pathActorJSONOffset)
	} else {
		e.AddRdxImm32(state.layout.pathJSONOffset)
	}
	e.MovR8Rdx()
	if state.layout.actorDomains {
		emitRuntimeHeapTelemetryActorJSONLength(
			e,
			leaPatches,
			state,
			state.layout.pathActorTemplate,
		)
	} else {
		e.MovR9dImm32(state.layout.pathJSONLength)
	}
	selectedAt := e.JmpRel32()

	zeroOff := len(e.Buf)
	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	if state.layout.actorDomains {
		e.AddRdxImm32(state.layout.zeroActorJSONOffset)
	} else {
		e.AddRdxImm32(state.layout.zeroJSONOffset)
	}
	e.MovR8Rdx()
	if state.layout.actorDomains {
		emitRuntimeHeapTelemetryActorJSONLength(
			e,
			leaPatches,
			state,
			state.layout.zeroActorTemplate,
		)
	} else {
		e.MovR9dImm32(state.layout.zeroJSONLength)
	}

	selectedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useZeroAt, zeroOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, selectedAt, selectedOff); err != nil {
		return err
	}

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.AddRdxImm32(state.layout.pathOffset)
	e.MovRsiRdx()
	e.MovEdiImm32(runtimeHeapTelemetryATFDCWD)
	e.MovEdxImm32(runtimeHeapTelemetryOpenFlags)
	e.MovR10dImm32(runtimeHeapTelemetryOpenMode)
	e.MovEaxImm32(runtimeHeapTelemetryOpenAt)
	e.Syscall()
	e.CmpRaxImm32(-4095)
	openFailedAt := e.JaeRel32()

	e.PushRax()
	e.MovRdiRax()
	e.MovRsiR8()
	e.MovRdxR9()
	e.MovEaxImm32(sysv.SysWrite)
	e.Syscall()
	e.PopRdi()
	e.MovEaxImm32(runtimeHeapTelemetryClose)
	e.Syscall()

	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, openFailedAt, doneOff); err != nil {
		return err
	}
	e.PopRax()
	return nil
}

func emitRuntimeHeapTelemetryFillTemplate(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
	jsonOffset int32,
	numbers runtimeHeapTelemetryNumberOffsets,
) error {
	for _, field := range numbers.fields() {
		emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
		e.MovRdiRdx()
		e.MovRaxFromRdiDisp(field.counterOffset)
		e.AddRdiImm32(jsonOffset + field.fieldOffset)
		if err := emitRuntimeHeapTelemetryWriteDecimal(e); err != nil {
			return err
		}
	}
	return nil
}

func emitRuntimeHeapTelemetryCaptureActors(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	callPatches *[]x64obj.CallPatch,
	state *runtimeHeapTelemetryState,
) error {
	if callPatches == nil {
		return fmt.Errorf("runtime heap telemetry: missing call patches")
	}
	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.AddRdiImm32(runtimeHeapTelemetryActorRecord0Offset)
	at := e.CallRel32()
	*callPatches = append(
		*callPatches,
		x64obj.CallPatch{At: at, Name: runtimeHeapTelemetryActorSnapshotSymbol},
	)

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryActorCountOffset)
	return nil
}

func emitRuntimeHeapTelemetryFillActorTemplate(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
	jsonOffset int32,
	template runtimeHeapTelemetryActorTemplate,
) error {
	for i, numbers := range template.numbers {
		recordOffset := runtimeHeapTelemetryActorRecord0Offset + int32(
			i,
		)*runtimeHeapTelemetryActorRecordSize
		fields := []runtimeHeapTelemetryNumberField{
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorCurrentOffset,
				fieldOffset:   numbers.current,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorPeakOffset,
				fieldOffset:   numbers.peak,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorCopiedOffset,
				fieldOffset:   numbers.bytesCopied,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorCurrentOffset,
				fieldOffset:   numbers.mailboxCurrent,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorPeakOffset,
				fieldOffset:   numbers.mailboxPeak,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorByteBudgetOffset,
				fieldOffset:   numbers.byteBudget,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorOverBudgetOffset,
				fieldOffset:   numbers.overBudgetCount,
			},
			{
				counterOffset: recordOffset + runtimeHeapTelemetryActorBackpressureOffset,
				fieldOffset:   numbers.backpressureEvents,
			},
		}
		for _, field := range fields {
			emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
			e.MovRdiRdx()
			e.MovRaxFromRdiDisp(field.counterOffset)
			e.AddRdiImm32(jsonOffset + field.fieldOffset)
			if err := emitRuntimeHeapTelemetryWriteDecimal(e); err != nil {
				return err
			}
		}
	}
	return emitRuntimeHeapTelemetryMoveActorSuffix(e, leaPatches, state, jsonOffset, template)
}

func emitRuntimeHeapTelemetryMoveActorSuffix(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
	jsonOffset int32,
	template runtimeHeapTelemetryActorTemplate,
) error {
	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryActorCountOffset)
	emitImulRaxImm32(e, template.entryLength)
	e.MovRdiRdx()
	e.AddRdiImm32(jsonOffset + template.entryOffset)
	emitAddRdiRax(e)
	e.MovRsiRdx()
	e.AddRsiImm32(jsonOffset + template.suffixOffset)

	e.XorEcxEcx()
	loopOff := len(e.Buf)
	e.CmpRcxImm32(template.suffixLength)
	doneAt := e.JaeRel32()
	e.MovzxEaxBytePtrRsiRcx()
	e.MovMem8RdiRcxPtrAl()
	e.AddEcxImm8(1)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, loopOff); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitRuntimeHeapTelemetryActorJSONLength(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
	template runtimeHeapTelemetryActorTemplate,
) {
	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryActorCountOffset)
	emitImulRaxImm32(e, template.entryLength)
	e.AddRaxImm32(template.entryOffset + template.suffixLength)
	e.MovR9Rax()
}

func emitImulRaxImm32(e *x64.Emitter, imm int32) {
	e.Emit(0x48, 0x69, 0xC0)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(imm))
	e.Emit(buf[:]...)
}

func emitAddRdiRax(e *x64.Emitter) {
	e.Emit(0x48, 0x01, 0xC7)
}

func emitRuntimeHeapTelemetryLoadBase(
	e *x64.Emitter,
	leaPatches *[]x64obj.LeaPatch,
	state *runtimeHeapTelemetryState,
) {
	leaPos := e.LeaRdxRipDisp()
	*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: state.dataIndex})
}

func emitRuntimeHeapTelemetryWriteDecimal(e *x64.Emitter) error {
	e.MovEcxImm32(runtimeHeapTelemetryNumberWidth)
	fillOff := len(e.Buf)
	e.MovMem8RdiRcxMinus1Imm8(' ')
	e.DecEcx()
	e.TestEcxEcx()
	fillAgainAt := e.JnzRel32()
	if err := x64.PatchRel32(e.Buf, fillAgainAt, fillOff); err != nil {
		return err
	}

	e.MovEcxImm32(runtimeHeapTelemetryNumberWidth)
	e.MovR9dImm32(10)
	e.TestRaxRax()
	nonZeroAt := e.JnzRel32()
	e.MovEdxImm32('0')
	e.MovMem8RdiDispDl(runtimeHeapTelemetryNumberWidth - 1)
	doneAt := e.JmpRel32()

	loopOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonZeroAt, loopOff); err != nil {
		return err
	}
	e.XorEdxEdx()
	e.DivR9()
	e.AddDlImm8('0')
	e.DecEcx()
	e.MovMem8RdiRcxDl()
	e.TestRaxRax()
	loopAgainAt := e.JnzRel32()
	if err := x64.PatchRel32(e.Buf, loopAgainAt, loopOff); err != nil {
		return err
	}

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

// ---- recursion_register.go ----

func emitRecursionBenchmarkRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, callInfo, ok := scalarCallABIFromBackendABI(abi)
	if !ok || callKind != scalarCallABISysV {
		return false, nil
	}
	if plan, ok, err := machine.RecursionFibPlanFromStackIRWithCallABI(fn, callInfo); err != nil ||
		ok {
		if err != nil || !ok {
			return ok, err
		}
		return emitRecursionFibRegisterFunction(e, fn, abi, callKind, plan, callPatches, flush)
	}
	if plan, ok, err := machine.RecursionMainPlanFromStackIRWithCallABI(fn, callInfo); err != nil ||
		ok {
		if err != nil || !ok {
			return ok, err
		}
		return emitRecursionMainRegisterFunction(e, fn, abi, callKind, plan, callPatches, flush)
	}
	return false, nil
}

func emitRecursionFibRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callKind scalarCallABIKind,
	plan machine.RecursionFibPlan,
	callPatches *[]x64obj.CallPatch,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize((fn.LocalSlots + 1) * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	paramOffset := scalarRegisterSlotOffset(plan.ParamLocal)
	scratchOffset := scalarRegisterSlotOffset(fn.LocalSlots)
	e.MovEaxFromRbpDisp(paramOffset)
	e.CmpEaxImm32(2)
	baseAt := e.JlRel32()

	e.MovEaxFromRbpDisp(paramOffset)
	e.SubEaxImm32(1)
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovMem64RbpDispRax(scratchOffset)

	e.MovEaxFromRbpDisp(paramOffset)
	e.SubEaxImm32(2)
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovEcxEax()
	e.MovEaxFromRbpDisp(scratchOffset)
	e.AddEaxEcx()
	doneAt := e.JmpRel32()

	baseTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, baseAt, baseTo); err != nil {
		return true, err
	}
	e.MovEaxFromRbpDisp(paramOffset)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitRecursionMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callKind scalarCallABIKind,
	plan machine.RecursionMainPlan,
	callPatches *[]x64obj.CallPatch,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	totalOffset := scalarRegisterSlotOffset(plan.TotalLocal)
	indexOffset := scalarRegisterSlotOffset(plan.IndexLocal)
	e.XorEcxEcx()
	e.XorEaxEax()

	loopStart := len(e.Buf)
	e.CmpRcxImm32(plan.LoopBound)
	exitAt := e.JgeRel32()
	e.MovMem64RbpDispRax(totalOffset)
	e.MovMem64RbpDispRcx(indexOffset)
	e.MovEaxImm32(uint32(plan.CallArg))
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovEcxEax()
	e.MovEaxFromRbpDisp(totalOffset)
	e.AddEaxEcx()
	e.MovMem64RbpDispRax(totalOffset)
	e.MovRaxFromRbpDisp(indexOffset)
	e.MovEcxEax()
	e.AddEcxImm8(1)
	e.MovEaxFromRbpDisp(totalOffset)
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}

	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovMem64RbpDispRax(totalOffset)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.MovEaxFromRbpDisp(totalOffset)
	e.CmpEaxImm32(plan.SuccessTotal)
	failAt := e.JnzRel32()
	e.MovEaxImm32(uint32(plan.TrueReturnImm))
	doneAt := e.JmpRel32()
	failTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, failAt, failTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.FalseReturnImm))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- scalar_call_loop_register.go ----

func emitScalarCallLoopRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, callInfo, ok := scalarCallABIFromBackendABI(abi)
	if !ok {
		return false, nil
	}
	plan, ok, err := machine.ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, callInfo)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	if plan.BoundLocal >= 0 {
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.BoundLocal))
		e.MovEdxEax()
	} else {
		e.MovEdxImm32(uint32(plan.BoundConst))
	}
	e.XorEaxEax()
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.CmpEdxEcx()
	bodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}

	e.MovMem64RbpDispRax(scalarRegisterSlotOffset(plan.TotalLocal))
	e.MovMem64RbpDispRcx(scalarRegisterSlotOffset(plan.IndexLocal))
	if plan.BoundLocal >= 0 {
		e.MovMem64RbpDispRdx(scalarRegisterSlotOffset(plan.BoundLocal))
	}
	for argIndex, localSlot := range plan.CallArgLocals {
		e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(localSlot))
		emitMoveRaxToScalarCallArg(e, callKind, argIndex)
	}
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovEcxEax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
	e.AddEaxEcx()
	e.MovMem64RbpDispRax(scalarRegisterSlotOffset(plan.TotalLocal))
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.IndexLocal))
	e.MovEcxEax()
	e.AddEcxImm8(1)
	if plan.BoundLocal >= 0 {
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.BoundLocal))
		e.MovEdxEax()
	} else {
		e.MovEdxImm32(uint32(plan.BoundConst))
	}
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	if plan.ReturnNonNegativeSuccess {
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
		e.CmpEaxImm32(0)
		successAt := e.JgeRel32()
		e.MovEaxImm32(1)
		doneAt := e.JmpRel32()
		successTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
			return true, err
		}
		e.XorEaxEax()
		doneTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
			return true, err
		}
	}
	if plan.ReturnOneIfTotalZero {
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
		e.CmpEaxImm32(0)
		equalAt := e.JzRel32()
		e.XorEaxEax()
		doneAt := e.JmpRel32()
		equalTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, equalAt, equalTo); err != nil {
			return true, err
		}
		e.MovEaxImm32(1)
		doneTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
			return true, err
		}
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitParallelMapReduceMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, _, ok := scalarCallABIFromBackendABI(abi)
	if !ok || callKind != scalarCallABISysV {
		return false, nil
	}
	plan, ok, err := machine.ParallelMapReduceMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	for _, spawn := range plan.Spawns {
		e.MovEdiImm32(uint32(spawn.EntryID))
		if err := emitScalarLoopCall(
			e,
			callKind,
			"__tetra_task_spawn_i32",
			callPatches,
		); err != nil {
			return true, err
		}
		e.MovMem64RbpDispRax(scalarRegisterSlotOffset(spawn.HandleLocal))
		e.MovMem64RbpDispRdx(scalarRegisterSlotOffset(spawn.StatusLocal))
	}

	e.XorEaxEax()
	e.MovMem64RbpDispRax(scalarRegisterSlotOffset(plan.TotalLocal))
	for _, join := range plan.Joins {
		e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(join.HandleLocal))
		e.MovRdiRax()
		e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(join.StatusLocal))
		e.MovRsiRax()
		if err := emitScalarLoopCall(
			e,
			callKind,
			"__tetra_task_join_i32",
			callPatches,
		); err != nil {
			return true, err
		}
		e.MovEcxEax()
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
		e.AddEaxEcx()
		e.MovMem64RbpDispRax(scalarRegisterSlotOffset(plan.TotalLocal))
	}

	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
	e.CmpEaxImm32(plan.ExpectedTotal)
	successAt := e.JzRel32()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.TotalLocal))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitActorPingPongRuntimeCallRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, callInfo, ok := scalarCallABIFromBackendABI(abi)
	if !ok || callKind != scalarCallABISysV {
		return false, nil
	}
	plan, ok, err := machine.ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(fn, callInfo)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	const actorPingPongScratchSlots = 2
	localSize := x64.AlignStackSize((fn.LocalSlots + actorPingPongScratchSlots) * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	depth := 0
	scratchOffset := func(stackIndex int) int32 {
		return scalarRegisterSlotOffset(fn.LocalSlots + stackIndex)
	}
	pushRAX := func() {
		e.MovMem64RbpDispRax(scratchOffset(depth))
		depth++
	}
	pushConst := func(value int32) {
		e.MovEaxImm32(uint32(value))
		pushRAX()
	}
	pushLocal := func(local int) {
		e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(local))
		pushRAX()
	}
	storeLocalFromTop := func(local int) error {
		if depth <= 0 {
			return fmt.Errorf("x64 actor ping-pong backend: %s stack underflow", fn.Name)
		}
		depth--
		e.MovRaxFromRbpDisp(scratchOffset(depth))
		e.MovMem64RbpDispRax(scalarRegisterSlotOffset(local))
		return nil
	}
	emitCall := func(name string, argSlots int, retSlots int) error {
		return emitScalarRegisterCall(
			e,
			callKind,
			ir.IRInstr{Kind: ir.IRCall, Name: name, ArgSlots: argSlots, RetSlots: retSlots},
			&depth,
			scratchOffset,
			callPatches,
		)
	}
	emitReturn := func(value int32) error {
		e.MovEaxImm32(uint32(value))
		if err := flush.emit(); err != nil {
			return err
		}
		e.Leave()
		e.Ret()
		return nil
	}

	switch plan.Path {
	case "machine-ir-actor-ping-pong-pong":
		if err := emitCall("__tetra_actor_recv", 0, 1); err != nil {
			return true, err
		}
		if err := storeLocalFromTop(plan.ValueLocal); err != nil {
			return true, err
		}
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.ValueLocal))
		e.CmpEaxImm32(41)
		failAt := e.JnzRel32()
		if err := emitCall("__tetra_actor_sender", 0, 1); err != nil {
			return true, err
		}
		pushConst(42)
		if err := emitCall("__tetra_actor_send", 2, 1); err != nil {
			return true, err
		}
		if err := storeLocalFromTop(plan.SentLocal); err != nil {
			return true, err
		}
		if err := emitReturn(0); err != nil {
			return true, err
		}
		failTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, failAt, failTo); err != nil {
			return true, err
		}
		if err := emitReturn(1); err != nil {
			return true, err
		}
	case "machine-ir-actor-ping-pong-main":
		pushConst(plan.SpawnEntryID)
		if err := emitCall("__tetra_actor_spawn", 1, 1); err != nil {
			return true, err
		}
		if err := storeLocalFromTop(plan.ActorLocal); err != nil {
			return true, err
		}
		pushLocal(plan.ActorLocal)
		pushConst(41)
		if err := emitCall("__tetra_actor_send", 2, 1); err != nil {
			return true, err
		}
		if err := storeLocalFromTop(plan.SentLocal); err != nil {
			return true, err
		}
		if err := emitCall("__tetra_actor_recv", 0, 1); err != nil {
			return true, err
		}
		if err := storeLocalFromTop(plan.ReplyLocal); err != nil {
			return true, err
		}
		e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.ReplyLocal))
		e.CmpEaxImm32(42)
		failAt := e.JnzRel32()
		if err := emitReturn(0); err != nil {
			return true, err
		}
		failTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, failAt, failTo); err != nil {
			return true, err
		}
		if err := emitReturn(1); err != nil {
			return true, err
		}
	default:
		return false, nil
	}
	return true, nil
}

// ---- scalar_const_modulo_loop_register.go ----

func emitScalarConstModuloLoopRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.ScalarIntConstModuloLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.XorEcxEcx()
	e.MovR10dImm32(0)
	e.MovR8dImm32(uint32(plan.Modulus))

	loopStart := len(e.Buf)
	e.CmpRcxImm32(plan.Bound)
	exitAt := e.JgeRel32()
	e.MovEaxEcx()
	e.Cdq()
	emitIdivR8d(e)
	emitAddR10dEdx(e)
	e.AddEcxImm8(1)
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}

	e.MovEaxR10d()
	e.TestEaxEax()
	negativeAt := e.JlRel32()
	e.MovEaxImm32(uint32(plan.TrueReturnImm))
	doneAt := e.JmpRel32()
	negativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, negativeAt, negativeTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.FalseReturnImm))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitIdivR8d(e *x64.Emitter) {
	e.Emit(0x41, 0xF7, 0xF8)
}

func emitAddR10dEdx(e *x64.Emitter) {
	e.Emit(0x41, 0x01, 0xD2)
}

// ---- scalar_loop_register.go ----

func emitScalarLoopRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.ScalarIntLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.ParamLocal))
	e.MovEdxEax()
	e.XorEaxEax()
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.CmpEdxEcx()
	bodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.AddEaxEcx()
	e.AddEcxImm8(byte(plan.Step))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- scalar_register.go ----

func emitScalarRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	hasCall := irFuncHasCall(fn)
	var callKind scalarCallABIKind
	var callInfo machine.CallABIInfo
	if hasCall {
		var ok bool
		callKind, callInfo, ok = scalarCallABIFromBackendABI(abi)
		if !ok {
			return false, nil
		}
		if _, ok, err := machine.ScalarIntFunctionFromStackIRWithCallABI(fn, callInfo); err != nil ||
			!ok {
			return ok, err
		}
	} else {
		if _, ok, err := machine.ScalarIntFunctionFromStackIR(fn); err != nil || !ok {
			return ok, err
		}
	}
	maxStack, err := scalarRegisterMaxStack(fn)
	if err != nil {
		return true, err
	}
	frameSlots := fn.LocalSlots + maxStack
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(frameSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)
	for i := fn.ParamSlots; i < fn.LocalSlots; i++ {
		e.MovMem64RbpDispImm(scalarRegisterSlotOffset(i), 0)
	}

	depth := 0
	scratchOffset := func(stackIndex int) int32 {
		return scalarRegisterSlotOffset(fn.LocalSlots + stackIndex)
	}
	pushEAX := func() {
		e.MovMem64RbpDispRax(scratchOffset(depth))
		depth++
	}
	popToEAX := func() error {
		if depth <= 0 {
			return fmt.Errorf("x64 scalar register backend: %s stack underflow", fn.Name)
		}
		depth--
		e.MovRaxFromRbpDisp(scratchOffset(depth))
		return nil
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			e.MovEaxImm32(uint32(instr.Imm))
			pushEAX()
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return true, fmt.Errorf(
					"x64 scalar register backend: %s local %d out of bounds",
					fn.Name,
					instr.Local,
				)
			}
			e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(instr.Local))
			pushEAX()
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return true, fmt.Errorf(
					"x64 scalar register backend: %s local %d out of bounds",
					fn.Name,
					instr.Local,
				)
			}
			if err := popToEAX(); err != nil {
				return true, err
			}
			e.MovMem64RbpDispRax(scalarRegisterSlotOffset(instr.Local))
		case ir.IRAddI32,
			ir.IRSubI32,
			ir.IRMulI32,
			ir.IRDivI32,
			ir.IRModI32,
			ir.IRCmpEqI32,
			ir.IRCmpLtI32,
			ir.IRCmpGtI32,
			ir.IRCmpGeI32,
			ir.IRCmpLeI32,
			ir.IRCmpNeI32:
			if depth < 2 {
				return true, fmt.Errorf(
					"x64 scalar register backend: %s binary stack underflow",
					fn.Name,
				)
			}
			right := scratchOffset(depth - 1)
			left := scratchOffset(depth - 2)
			e.MovEaxFromRbpDisp(right)
			e.MovEcxEax()
			e.MovEaxFromRbpDisp(left)
			switch instr.Kind {
			case ir.IRAddI32:
				e.AddEaxEcx()
			case ir.IRSubI32:
				e.SubEaxEcx()
			case ir.IRMulI32:
				e.ImulEaxEcx()
			case ir.IRDivI32:
				e.Cdq()
				e.IdivEcx()
			case ir.IRModI32:
				e.Cdq()
				e.IdivEcx()
				e.MovMem64RbpDispRdx(scratchOffset(depth - 2))
				depth--
				continue
			case ir.IRCmpEqI32:
				e.CmpEaxEcx()
				e.SeteAl()
				e.MovzxEaxAl()
			case ir.IRCmpLtI32:
				e.CmpEaxEcx()
				e.SetlAl()
				e.MovzxEaxAl()
			case ir.IRCmpGtI32:
				e.CmpEaxEcx()
				e.SetgAl()
				e.MovzxEaxAl()
			case ir.IRCmpGeI32:
				e.CmpEaxEcx()
				e.SetgeAl()
				e.MovzxEaxAl()
			case ir.IRCmpLeI32:
				e.CmpEaxEcx()
				e.SetleAl()
				e.MovzxEaxAl()
			case ir.IRCmpNeI32:
				e.CmpEaxEcx()
				e.SetneAl()
				e.MovzxEaxAl()
			}
			depth--
			e.MovMem64RbpDispRax(scratchOffset(depth - 1))
		case ir.IRNegI32:
			if err := popToEAX(); err != nil {
				return true, err
			}
			e.NegEax()
			pushEAX()
		case ir.IRCall:
			if err := emitScalarRegisterCall(
				e,
				callKind,
				instr,
				&depth,
				scratchOffset,
				callPatches,
			); err != nil {
				return true, err
			}
		case ir.IRReturn:
			if err := popToEAX(); err != nil {
				return true, err
			}
			if depth != 0 {
				return true, fmt.Errorf(
					"x64 scalar register backend: %s return leaves %d extra values",
					fn.Name,
					depth,
				)
			}
			if err := flush.emit(); err != nil {
				return true, err
			}
			e.Leave()
			e.Ret()
		default:
			return false, nil
		}
	}
	return true, nil
}

func scalarRegisterSlotOffset(slot int) int32 {
	return -int32((slot + 1) * 8)
}

func scalarRegisterMaxStack(fn ir.IRFunc) (int, error) {
	depth := 0
	maxDepth := 0
	push := func(n int) {
		depth += n
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	pop := func(n int, kind ir.IRInstrKind) error {
		if depth < n {
			return fmt.Errorf(
				"x64 scalar register backend: %s stack underflow at ir.%d",
				fn.Name,
				kind,
			)
		}
		depth -= n
		return nil
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal:
			push(1)
		case ir.IRStoreLocal:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
		case ir.IRAddI32,
			ir.IRSubI32,
			ir.IRMulI32,
			ir.IRDivI32,
			ir.IRModI32,
			ir.IRCmpEqI32,
			ir.IRCmpLtI32,
			ir.IRCmpGtI32,
			ir.IRCmpGeI32,
			ir.IRCmpLeI32,
			ir.IRCmpNeI32:
			if err := pop(2, instr.Kind); err != nil {
				return 0, err
			}
			push(1)
		case ir.IRNegI32:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
			push(1)
		case ir.IRCall:
			if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.RetSlots > 1 {
				return 0, fmt.Errorf(
					"x64 scalar register backend: unsupported call ABI at ir.%d",
					instr.Kind,
				)
			}
			if err := pop(instr.ArgSlots, instr.Kind); err != nil {
				return 0, err
			}
			push(instr.RetSlots)
		case ir.IRReturn:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("x64 scalar register backend: unsupported ir.%d", instr.Kind)
		}
	}
	return maxDepth, nil
}

func emitHashTableLookupRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.HashTableLookupPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.KeysBaseLocal))
	e.MovR9Rax()
	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.ValuesBaseLocal))
	e.MovRdiRax()
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.BoundLocal))
	e.MovEdxEax()
	e.CmpEdxEcx()
	bodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}

	e.MovR8dFromR9RcxScale4()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.KeyLocal))
	emitCmpR8dEax(e)
	matchAt := e.JzRel32()
	e.AddEcxImm8(byte(plan.Step))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}

	matchTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, matchAt, matchTo); err != nil {
		return true, err
	}
	e.MovR8dFromRdiRcxScale4()
	e.MovEaxR8d()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()

	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.XorEaxEax()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitHashTableMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callPatches *[]x64obj.CallPatch,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, _, ok := scalarCallABIFromBackendABI(abi)
	if !ok || callKind != scalarCallABISysV {
		return false, nil
	}
	plan, ok, err := machine.HashTableMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	keysPtrOffset := scalarRegisterSlotOffset(plan.KeysPtrLocal)
	keysLenOffset := scalarRegisterSlotOffset(plan.KeysLenLocal)
	valuesPtrOffset := scalarRegisterSlotOffset(plan.ValuesPtrLocal)
	valuesLenOffset := scalarRegisterSlotOffset(plan.ValuesLenLocal)
	nOffset := scalarRegisterSlotOffset(plan.NLocal)
	indexOffset := scalarRegisterSlotOffset(plan.IndexLocal)
	checksumOffset := scalarRegisterSlotOffset(plan.ChecksumLocal)
	queryOffset := scalarRegisterSlotOffset(plan.QueryLocal)
	keyOffset := scalarRegisterSlotOffset(plan.KeyLocal)

	e.MovEaxImm32(uint32(plan.Length))
	e.MovMem64RbpDispRax(nOffset)
	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.KeysBackingLocal + plan.KeysBackingSlots - 1))
	e.MovMem64RbpDispRax(keysPtrOffset)
	e.MovMem64RbpDispImm(keysLenOffset, plan.Length)
	e.LeaRaxRbpDisp(
		scalarRegisterSlotOffset(plan.ValuesBackingLocal + plan.ValuesBackingSlots - 1),
	)
	e.MovMem64RbpDispRax(valuesPtrOffset)
	e.MovMem64RbpDispImm(valuesLenOffset, plan.Length)
	e.MovMem64RbpDispImm(indexOffset, 0)

	e.MovRaxFromRbpDisp(keysPtrOffset)
	e.MovR9Rax()
	e.MovRaxFromRbpDisp(valuesPtrOffset)
	e.MovR10Rax()
	e.XorEcxEcx()

	fillLoopStart := len(e.Buf)
	e.CmpRcxImm32(plan.Length)
	fillExitAt := e.JgeRel32()
	e.MovEaxEcx()
	emitAddEaxEax(e)
	emitAddEaxImm8(e, 1)
	emitMovMem32R9RcxScale4Eax(e)
	e.MovEaxEcx()
	emitAddEaxImm8(e, 7)
	emitMovMem32R10RcxScale4Eax(e)
	e.AddEcxImm8(byte(plan.Step))
	fillBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, fillBackAt, fillLoopStart); err != nil {
		return true, err
	}
	fillExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fillExitAt, fillExitTo); err != nil {
		return true, err
	}

	e.MovMem64RbpDispImm(checksumOffset, 0)
	e.MovMem64RbpDispImm(queryOffset, 0)

	queryLoopStart := len(e.Buf)
	e.MovEaxFromRbpDisp(queryOffset)
	e.MovEcxEax()
	e.CmpRcxImm32(plan.Length)
	queryExitAt := e.JgeRel32()
	e.MovEaxEcx()
	emitAddEaxEax(e)
	emitAddEaxImm8(e, 1)
	e.MovMem64RbpDispRax(keyOffset)

	e.MovRaxFromRbpDisp(keysPtrOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	e.MovRaxFromRbpDisp(keysLenOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 1)
	e.MovRaxFromRbpDisp(valuesPtrOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 2)
	e.MovRaxFromRbpDisp(valuesLenOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 3)
	e.MovRaxFromRbpDisp(nOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 4)
	e.MovRaxFromRbpDisp(keyOffset)
	emitMoveRaxToScalarCallArg(e, callKind, 5)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}

	e.MovEcxEax()
	e.MovEaxFromRbpDisp(checksumOffset)
	e.AddEaxEcx()
	e.MovMem64RbpDispRax(checksumOffset)
	e.MovEaxFromRbpDisp(queryOffset)
	e.MovEcxEax()
	e.AddEcxImm8(byte(plan.Step))
	e.MovMem64RbpDispRcx(queryOffset)
	queryBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, queryBackAt, queryLoopStart); err != nil {
		return true, err
	}
	queryExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, queryExitAt, queryExitTo); err != nil {
		return true, err
	}

	e.MovEaxFromRbpDisp(checksumOffset)
	e.CmpEaxImm32(0)
	successAt := e.JgRel32()
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitCmpR8dEax(e *x64.Emitter) {
	e.Emit(0x41, 0x39, 0xC0)
}

func emitPostgreSQLFrameTypeAtRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.PostgreSQLFrameTypeAtPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.OffsetLocal))
	e.MovRcxRax()
	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.SrcBaseLocal))
	e.MovRsiRax()
	e.MovzxEaxBytePtrRsiRcx()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitPostgreSQLInoutWriterRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.PostgreSQLInoutWriterPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	shifts := postgresqlInoutWriterByteShifts(plan.StoreCount)
	for i, offset := range plan.StoreOffsets {
		shift := byte(0)
		if i < len(shifts) {
			shift = shifts[i]
		}
		emitPostgreSQLInoutWriterStoreByte(e, plan, offset, shift)
	}

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.StartLocal))
	e.AddRaxImm32(plan.ReturnAddend)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitInoutWriterHelperSummaryRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.InoutWriterHelperSummaryPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}
	if plan.StoreCount != len(plan.StoreIndexes) ||
		plan.StoreCount != len(plan.StoreValues) {
		return true, fmt.Errorf(
			"x64 helper-summary writer: %s has incomplete store facts",
			plan.HelperName,
		)
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.DstBaseLocal))
	e.MovRdiRax()
	for i, index := range plan.StoreIndexes {
		value := plan.StoreValues[i]
		if value < 0 || value > 255 {
			return true, fmt.Errorf(
				"x64 helper-summary writer: %s store %d byte value %d out of u8 range",
				plan.HelperName,
				i,
				value,
			)
		}
		e.MovEdxImm32(uint32(value))
		e.MovMem8RdiDispDl(index)
	}

	e.MovEaxImm32(uint32(plan.ScalarReturnConst))
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitInoutWriterHelperSummaryCallerRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	if _, ok, err := machine.InoutWriterHelperSummaryCallerFunctionFromStackIR(fn); err != nil ||
		!ok {
		return ok, err
	}
	switch fn.Name {
	case "p25.json_parse_stringify.main", "p25.http_plaintext_json.main":
	default:
		return false, nil
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovEaxImm32(0)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitPostgreSQLInoutWriterMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	if _, ok, err := machine.PostgreSQLInoutWriterMainPlanFromStackIR(fn); err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)
	e.XorEaxEax()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func postgresqlInoutWriterByteShifts(storeCount int) []byte {
	switch storeCount {
	case 4:
		return []byte{24, 16, 8, 0}
	case 2:
		return []byte{8, 0}
	default:
		return nil
	}
}

func emitPostgreSQLInoutWriterStoreByte(
	e *x64.Emitter,
	plan machine.PostgreSQLInoutWriterPlan,
	offset int32,
	shift byte,
) {
	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.StartLocal))
	e.MovRdxRax()
	if offset != 0 {
		e.AddRdxImm32(offset)
	}
	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.DstBaseLocal))
	e.AddRaxRdx()
	e.MovRsiRax()

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.ValueLocal))
	e.MovRcxRax()
	if shift != 0 {
		emitShrEcxImm8(e, shift)
	}
	emitAndEcxImm32(e, 0xff)
	e.MovRaxRsi()
	e.MovMem8RaxPtrCl()
}

func emitShrEcxImm8(e *x64.Emitter, shift byte) {
	e.Emit(0xC1, 0xE9, shift)
}

func emitAndEcxImm32(e *x64.Emitter, mask uint32) {
	e.Emit(0x81, 0xE1)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], mask)
	e.Emit(buf[:]...)
}

// ---- slice_sum_main_register.go ----

func emitSliceSumMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.SliceSumMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.BackingLocal + plan.BackingSlots - 1))
	e.MovR9Rax()
	e.XorEcxEcx()
	e.MovR8dImm32(uint32(plan.FillModulus))

	fillLoopStart := len(e.Buf)
	e.CmpRcxImm32(plan.Length)
	fillExitAt := e.JgeRel32()
	e.MovEaxEcx()
	e.Cdq()
	emitIdivR8d(e)
	emitMovMem32R9RcxScale4Edx(e)
	e.AddEcxImm8(byte(plan.Step))
	fillBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, fillBackAt, fillLoopStart); err != nil {
		return true, err
	}
	fillExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fillExitAt, fillExitTo); err != nil {
		return true, err
	}

	e.MovR10dImm32(0)
	e.XorEcxEcx()
	outerLoopStart := len(e.Buf)
	e.CmpRcxImm32(plan.RepeatCount)
	outerExitAt := e.JgeRel32()
	e.XorEdxEdx()

	innerLoopStart := len(e.Buf)
	e.CmpEdxImm32(plan.Length)
	innerExitAt := e.JgeRel32()
	emitMovR8dFromR9RdxScale4(e)
	e.AddR10dR8d()
	e.AddEdxImm32(plan.Step)
	innerBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, innerBackAt, innerLoopStart); err != nil {
		return true, err
	}
	innerExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, innerExitAt, innerExitTo); err != nil {
		return true, err
	}
	e.AddEcxImm8(byte(plan.Step))
	outerBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, outerBackAt, outerLoopStart); err != nil {
		return true, err
	}
	outerExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, outerExitAt, outerExitTo); err != nil {
		return true, err
	}

	e.MovEaxR10d()
	e.CmpEaxImm32(0)
	successAt := e.JgRel32()
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitMovMem32R9RcxScale4Edx(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0x14, 0x89)
}

func emitMovR8dFromR9RdxScale4(e *x64.Emitter) {
	e.Emit(0x45, 0x8B, 0x04, 0x91)
}

// ---- matrix_multiply_main_register.go ----

func emitMatrixMultiplyMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.MatrixMultiplyMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.ABackingLocal + plan.BackingSlots - 1))
	e.MovR9Rax()
	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.BBackingLocal + plan.BackingSlots - 1))
	e.MovR10Rax()
	e.LeaRaxRbpDisp(scalarRegisterSlotOffset(plan.CBackingLocal + plan.BackingSlots - 1))
	emitMovR11Rax(e)

	e.XorEcxEcx()
	fillLoopStart := len(e.Buf)
	e.CmpRcxImm32(plan.SliceLength)
	fillExitAt := e.JgeRel32()
	e.MovEaxEcx()
	emitAddEaxImm8(e, byte(plan.Step))
	emitMovMem32R9RcxScale4Eax(e)
	e.MovEaxImm32(uint32(plan.SliceLength))
	e.SubEaxEcx()
	emitMovMem32R10RcxScale4Eax(e)
	e.XorEaxEax()
	emitMovMem32R11RcxScale4Eax(e)
	e.AddEcxImm8(byte(plan.Step))
	fillBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, fillBackAt, fillLoopStart); err != nil {
		return true, err
	}
	fillExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fillExitAt, fillExitTo); err != nil {
		return true, err
	}

	checksumOffset := scalarRegisterSlotOffset(plan.ChecksumLocal)
	totalOffset := scalarRegisterSlotOffset(plan.TotalLocal)
	e.MovMem32RbpDispImm(checksumOffset, 0)
	e.XorEcxEcx()
	repeatLoopStart := len(e.Buf)
	e.CmpRcxImm32(plan.RepeatCount)
	repeatExitAt := e.JgeRel32()
	e.XorEdxEdx()

	rowLoopStart := len(e.Buf)
	e.CmpEdxImm32(plan.Dimension)
	rowExitAt := e.JgeRel32()
	emitXorEsiEsi(e)

	colLoopStart := len(e.Buf)
	emitCmpEsiImm32(e, plan.Dimension)
	colExitAt := e.JgeRel32()
	emitXorEdiEdi(e)
	e.MovMem32RbpDispImm(totalOffset, 0)

	kLoopStart := len(e.Buf)
	e.CmpEdiImm32(plan.Dimension)
	kExitAt := e.JgeRel32()
	e.MovEaxEdx()
	emitImulEaxImm8(e, byte(plan.Dimension))
	emitAddEaxEdi(e)
	emitMovR8dFromR9RaxScale4(e)
	e.MovEaxEdi()
	emitImulEaxImm8(e, byte(plan.Dimension))
	emitAddEaxEsi(e)
	emitMovEaxFromR10RaxScale4(e)
	emitImulEaxR8d(e)
	emitMovR8dFromRbpDisp(e, totalOffset)
	emitAddR8dEax(e)
	e.MovMem32RbpDispR8d(totalOffset)
	emitAddEdiImm8(e, byte(plan.Step))
	kBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, kBackAt, kLoopStart); err != nil {
		return true, err
	}
	kExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, kExitAt, kExitTo); err != nil {
		return true, err
	}

	e.MovEaxEdx()
	emitImulEaxImm8(e, byte(plan.Dimension))
	emitAddEaxEsi(e)
	emitMovR8dFromRbpDisp(e, totalOffset)
	emitMovMem32R11RaxScale4R8d(e)
	emitAddEsiImm8(e, byte(plan.Step))
	colBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, colBackAt, colLoopStart); err != nil {
		return true, err
	}
	colExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, colExitAt, colExitTo); err != nil {
		return true, err
	}

	e.AddEdxImm32(plan.Step)
	rowBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, rowBackAt, rowLoopStart); err != nil {
		return true, err
	}
	rowExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, rowExitAt, rowExitTo); err != nil {
		return true, err
	}

	e.MovEaxEcx()
	e.Cdq()
	e.MovR8dImm32(uint32(plan.SliceLength))
	emitIdivR8d(e)
	emitMovEaxFromR11RdxScale4(e)
	emitMovR8dFromRbpDisp(e, checksumOffset)
	emitAddR8dEax(e)
	e.MovMem32RbpDispR8d(checksumOffset)
	e.AddEcxImm8(byte(plan.Step))
	repeatBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, repeatBackAt, repeatLoopStart); err != nil {
		return true, err
	}
	repeatExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, repeatExitAt, repeatExitTo); err != nil {
		return true, err
	}

	e.MovEaxFromRbpDisp(checksumOffset)
	e.CmpEaxImm32(0)
	successAt := e.JgRel32()
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneAt := e.JmpRel32()
	successTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, successAt, successTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func emitMovR11Rax(e *x64.Emitter) {
	e.Emit(0x49, 0x89, 0xC3)
}

func emitAddEaxImm8(e *x64.Emitter, v byte) {
	e.Emit(0x83, 0xC0, v)
}

func emitAddEaxEax(e *x64.Emitter) {
	e.Emit(0x01, 0xC0)
}

func emitMovMem32R9RcxScale4Eax(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0x04, 0x89)
}

func emitMovMem32R10RcxScale4Eax(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0x04, 0x8A)
}

func emitMovMem32R11RcxScale4Eax(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0x04, 0x8B)
}

func emitXorEsiEsi(e *x64.Emitter) {
	e.Emit(0x31, 0xF6)
}

func emitXorEdiEdi(e *x64.Emitter) {
	e.Emit(0x31, 0xFF)
}

func emitCmpEsiImm32(e *x64.Emitter, v int32) {
	e.Emit(0x81, 0xFE)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func emitImulEaxImm8(e *x64.Emitter, v byte) {
	e.Emit(0x6B, 0xC0, v)
}

func emitAddEaxEdi(e *x64.Emitter) {
	e.Emit(0x01, 0xF8)
}

func emitAddEaxEsi(e *x64.Emitter) {
	e.Emit(0x01, 0xF0)
}

func emitMovR8dFromR9RaxScale4(e *x64.Emitter) {
	e.Emit(0x45, 0x8B, 0x04, 0x81)
}

func emitMovEaxFromR10RaxScale4(e *x64.Emitter) {
	e.Emit(0x41, 0x8B, 0x04, 0x82)
}

func emitImulEaxR8d(e *x64.Emitter) {
	e.Emit(0x41, 0x0F, 0xAF, 0xC0)
}

func emitMovR8dFromRbpDisp(e *x64.Emitter, disp int32) {
	e.Emit(0x44, 0x8B, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitAddR8dEax(e *x64.Emitter) {
	e.Emit(0x41, 0x01, 0xC0)
}

func emitAddEdiImm8(e *x64.Emitter, v byte) {
	e.Emit(0x83, 0xC7, v)
}

func emitMovMem32R11RaxScale4R8d(e *x64.Emitter) {
	e.Emit(0x45, 0x89, 0x04, 0x83)
}

func emitAddEsiImm8(e *x64.Emitter, v byte) {
	e.Emit(0x83, 0xC6, v)
}

func emitMovEaxFromR11RdxScale4(e *x64.Emitter) {
	e.Emit(0x41, 0x8B, 0x04, 0x93)
}

// ---- scalar_slice_sum_register.go ----

func emitScalarSliceSumRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.ScalarI32SliceSumLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(plan.BaseLocal))
	e.MovR9Rax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(plan.LenLocal))
	e.MovEdxEax()
	e.MovR10dImm32(0)
	e.XorEcxEcx()

	loopStart := len(e.Buf)
	e.CmpEdxEcx()
	bodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.MovR8dFromR9RcxScale4()
	e.AddR10dR8d()
	e.AddEcxImm8(byte(plan.Step))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovEaxR10d()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- vector_copy_u8_register.go ----

func emitVectorCopyU8RegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.VectorU8x16CopyLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned {
		return true, fmt.Errorf("x64 vector copy_u8: unsupported plan shape")
	}
	scalar := plan.ScalarPlan

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(scalar.DstBaseLocal))
	e.MovRdiRax()
	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(scalar.SrcBaseLocal))
	e.MovRsiRax()
	e.MovR9Rax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(scalar.LenLocal))
	e.MovEdxEax()
	e.XorEcxEcx()

	vectorLoopStart := len(e.Buf)
	e.MovR8dEdx()
	e.SubR8dEcx()
	e.CmpR8dImm8(byte(plan.LaneCount))
	bodyAt := e.JgeRel32()
	tailAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.MovdquXmm0FromR9Rcx()
	e.MovdquRdiRcxFromXmm0()
	e.AddEcxImm8(byte(plan.LaneCount))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, vectorLoopStart); err != nil {
		return true, err
	}

	tailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailAt, tailTo); err != nil {
		return true, err
	}
	tailLoopStart := len(e.Buf)
	e.CmpEdxEcx()
	tailBodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	tailBodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailBodyAt, tailBodyTo); err != nil {
		return true, err
	}
	e.MovzxEaxBytePtrRsiRcx()
	e.MovMem8RdiRcxPtrAl()
	e.AddEcxImm8(1)
	tailBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, tailBackAt, tailLoopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(0)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- vector_map_i32_register.go ----

func emitVectorMapI32AddConstRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.VectorI32x4MapAddConstPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}
	if plan.LaneCount != 4 || !plan.SafeUnaligned || plan.Addend != 1 {
		return true, fmt.Errorf("x64 vector map_i32 add-const: unsupported plan shape")
	}
	scalar := plan.ScalarPlan

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(scalar.BaseLocal))
	e.MovR9Rax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(scalar.LenLocal))
	e.MovEdxEax()
	e.XorEcxEcx()
	e.MovEaxImm32(uint32(plan.Addend))
	e.MovdXmm1Eax()
	e.PshufdXmm1Xmm1Imm8(0)

	vectorLoopStart := len(e.Buf)
	e.MovR8dEdx()
	e.SubR8dEcx()
	e.CmpR8dImm8(byte(plan.LaneCount))
	bodyAt := e.JgeRel32()
	tailAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.MovdquXmm0FromR9RcxScale4()
	e.PadddXmm0Xmm1()
	e.MovdquR9RcxScale4FromXmm0()
	e.AddEcxImm8(byte(plan.LaneCount))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, vectorLoopStart); err != nil {
		return true, err
	}

	tailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailAt, tailTo); err != nil {
		return true, err
	}
	tailLoopStart := len(e.Buf)
	e.CmpEdxEcx()
	tailBodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	tailBodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailBodyAt, tailBodyTo); err != nil {
		return true, err
	}
	e.AddMem32R9RcxScale4Imm8(byte(plan.Addend))
	e.AddEcxImm8(1)
	tailBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, tailBackAt, tailLoopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(0)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- vector_memset_u8_register.go ----

func emitVectorMemsetZeroU8RegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.VectorU8x16MemsetZeroPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned || plan.FillValue != 0 {
		return true, fmt.Errorf("x64 vector memset_zero_u8: unsupported plan shape")
	}
	scalar := plan.ScalarPlan

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(scalar.BaseLocal))
	e.MovRdiRax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(scalar.LenLocal))
	e.MovEdxEax()
	e.XorEcxEcx()
	e.PxorXmm0Xmm0()

	vectorLoopStart := len(e.Buf)
	e.MovR8dEdx()
	e.SubR8dEcx()
	e.CmpR8dImm8(byte(plan.LaneCount))
	bodyAt := e.JgeRel32()
	tailAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.MovdquRdiRcxFromXmm0()
	e.AddEcxImm8(byte(plan.LaneCount))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, vectorLoopStart); err != nil {
		return true, err
	}

	tailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailAt, tailTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(0)
	tailLoopStart := len(e.Buf)
	e.CmpEdxEcx()
	tailBodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	tailBodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailBodyAt, tailBodyTo); err != nil {
		return true, err
	}
	e.MovMem8RdiRcxPtrAl()
	e.AddEcxImm8(1)
	tailBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, tailBackAt, tailLoopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(0)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}

// ---- vector_slice_sum_register.go ----

func emitVectorSliceSumRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machine.VectorI32x4SliceSumLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return ok, err
	}
	if plan.LaneCount != 4 || !plan.SafeUnaligned || plan.ScalarPlan.Step != 1 {
		return true, fmt.Errorf("x64 vector slice sum: unsupported plan shape")
	}
	scalar := plan.ScalarPlan

	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(scalar.BaseLocal))
	e.MovR9Rax()
	e.MovEaxFromRbpDisp(scalarRegisterSlotOffset(scalar.LenLocal))
	e.MovEdxEax()
	e.XorEcxEcx()
	e.PxorXmm1Xmm1()

	vectorLoopStart := len(e.Buf)
	e.MovR8dEdx()
	e.SubR8dEcx()
	e.CmpR8dImm8(byte(plan.LaneCount))
	bodyAt := e.JgeRel32()
	tailAt := e.JmpRel32()
	bodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyTo); err != nil {
		return true, err
	}
	e.MovdquXmm0FromR9RcxScale4()
	e.PadddXmm1Xmm0()
	e.AddEcxImm8(byte(plan.LaneCount))
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, vectorLoopStart); err != nil {
		return true, err
	}

	tailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailAt, tailTo); err != nil {
		return true, err
	}
	e.PshufdXmm0Xmm1Imm8(0x4E)
	e.PadddXmm1Xmm0()
	e.PshufdXmm0Xmm1Imm8(0xB1)
	e.PadddXmm1Xmm0()
	e.MovdEaxXmm1()
	e.MovR10dEax()

	tailLoopStart := len(e.Buf)
	e.CmpEdxEcx()
	tailBodyAt := e.JgRel32()
	exitAt := e.JmpRel32()
	tailBodyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, tailBodyAt, tailBodyTo); err != nil {
		return true, err
	}
	e.MovR8dFromR9RcxScale4()
	e.AddR10dR8d()
	e.AddEcxImm8(byte(scalar.Step))
	tailBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, tailBackAt, tailLoopStart); err != nil {
		return true, err
	}
	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovEaxR10d()
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}
