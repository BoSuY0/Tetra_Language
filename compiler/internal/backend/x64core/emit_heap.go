package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

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
