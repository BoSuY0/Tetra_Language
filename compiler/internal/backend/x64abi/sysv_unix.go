package x64abi

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

type SysVUnix struct {
	SysExit     uint32
	SysWrite    uint32
	SysMmap     uint32
	SysMunmap   uint32
	SysMprotect uint32
}

func LinuxSysV() *SysVUnix {
	return &SysVUnix{
		SysExit:     60,
		SysWrite:    1,
		SysMmap:     9,
		SysMunmap:   11,
		SysMprotect: 10,
	}
}

func LinuxX32SysV() *SysVUnix {
	const x32SyscallBit = 0x40000000
	return &SysVUnix{
		SysExit:     x32SyscallBit + 60,
		SysWrite:    x32SyscallBit + 1,
		SysMmap:     x32SyscallBit + 9,
		SysMunmap:   x32SyscallBit + 11,
		SysMprotect: x32SyscallBit + 10,
	}
}

func MacSysV() *SysVUnix {
	return &SysVUnix{
		SysExit:     0x2000001,
		SysWrite:    0x2000004,
		SysMmap:     0x20000C5,
		SysMunmap:   0x2000049,
		SysMprotect: 0x200000A,
	}
}

func (a *SysVUnix) SpillParams(e *x64.Emitter, fn ir.IRFunc) {
	for i := 0; i < fn.ParamSlots; i++ {
		off := -int32((i + 1) * 8)
		switch i {
		case 0:
			e.MovMem64RbpDispRdi(off)
		case 1:
			e.MovMem64RbpDispRsi(off)
		case 2:
			e.MovMem64RbpDispRdx(off)
		case 3:
			e.MovMem64RbpDispRcx(off)
		case 4:
			e.MovMem64RbpDispR8(off)
		case 5:
			e.MovMem64RbpDispR9(off)
		default:
			stackOff := int32(16 + 8*(i-6))
			e.MovRaxFromRbpDisp(stackOff)
			e.MovMem64RbpDispRax(off)
		}
	}
}

func (a *SysVUnix) EmitCall(
	e *x64.Emitter,
	instr ir.IRInstr,
	stackDepth *int,
	callPatches *[]x64obj.CallPatch,
) error {
	if stackDepth == nil || callPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/callPatches")
	}
	if instr.Name == "" {
		return fmt.Errorf("call is missing target name")
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf(
			"call %q has negative ABI slots args=%d rets=%d",
			instr.Name,
			instr.ArgSlots,
			instr.RetSlots,
		)
	}
	if instr.RetSlots > maxCallReturnSlots {
		return fmt.Errorf(
			"call %q has unsupported return slots %d (max=%d)",
			instr.Name,
			instr.RetSlots,
			maxCallReturnSlots,
		)
	}
	if *stackDepth < instr.ArgSlots {
		return fmt.Errorf("stack underflow in call to '%s'", instr.Name)
	}
	*stackDepth -= instr.ArgSlots

	extra := 0
	if instr.ArgSlots > 6 {
		extra = instr.ArgSlots - 6
	}
	if extra > 0 {
		tempSize := int32(extra * 8)
		e.SubRspImm32(tempSize)
		for i := 6; i < instr.ArgSlots; i++ {
			srcOffset := tempSize + int32(8*(instr.ArgSlots-1-i))
			dstOffset := int32(8 * (i - 6))
			e.MovRaxFromRspDisp(srcOffset)
			e.MovMem64RspDispRax(dstOffset)
		}
		e.AddRspImm32(tempSize)
	}
	for i := instr.ArgSlots - 1; i >= 0; i-- {
		switch i {
		case 0:
			e.PopRdi()
		case 1:
			e.PopRsi()
		case 2:
			e.PopRdx()
		case 3:
			e.PopRcx()
		case 4:
			e.PopR8()
		case 5:
			e.PopR9()
		default:
			e.PopRax()
		}
	}
	if extra > 0 {
		e.SubRspImm32(int32(extra * 8))
	}
	needAlign := (*stackDepth+extra)%2 != 0
	alignBytes := int32(0)
	if needAlign {
		e.SubRspImm32(8)
		alignBytes = 8
	}
	if extra > 0 {
		tempBaseOffset := -int32(instr.ArgSlots*8) + alignBytes
		for i := 6; i < instr.ArgSlots; i++ {
			tempOffset := tempBaseOffset + int32(8*(i-6))
			dstOffset := int32(8 * (i - 6))
			e.MovRaxFromRspDisp(tempOffset)
			e.MovMem64RspDispRax(dstOffset)
		}
	}
	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: instr.Name})
	if needAlign {
		e.AddRspImm32(8)
	}
	if extra > 0 {
		e.AddRspImm32(int32(extra * 8))
	}
	if instr.RetSlots > 0 {
		e.PushRax()
		*stackDepth++
	}
	if instr.RetSlots > 1 {
		e.PushRdx()
		*stackDepth++
	}
	if instr.RetSlots > 2 {
		e.PushR8()
		*stackDepth++
	}
	if instr.RetSlots > 3 {
		e.PushR9()
		*stackDepth++
	}
	if instr.RetSlots > 4 {
		e.PushR10()
		*stackDepth++
	}
	if instr.RetSlots > 5 {
		e.PushR11()
		*stackDepth++
	}
	if instr.RetSlots > 6 {
		e.PushR12()
		*stackDepth++
	}
	if instr.RetSlots > 7 {
		e.PushR13()
		*stackDepth++
	}
	if instr.RetSlots > 8 {
		e.PushR14()
		*stackDepth++
	}
	if instr.RetSlots > 9 {
		e.PushRbx()
		*stackDepth++
	}
	return nil
}

func (a *SysVUnix) EmitWriteStdout(
	e *x64.Emitter,
	stackDepth *int,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 2 {
		return fmt.Errorf("stack underflow in write")
	}
	*stackDepth -= 2
	e.PopRdx()
	e.PopRsi()
	e.MovEaxImm32(a.SysWrite)
	e.MovEdiImm32(1)
	e.Syscall()
	return nil
}

func (a *SysVUnix) EmitExit(
	e *x64.Emitter,
	code int32,
	stackSlots int,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = stackSlots
	_ = importPatches
	e.MovEdiImm32(uint32(code))
	e.MovEaxImm32(a.SysExit)
	e.Syscall()
	return nil
}

func (a *SysVUnix) EmitAllocBytes(
	e *x64.Emitter,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = opt
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in alloc_bytes")
	}
	*stackDepth--
	e.PopRsi()
	e.MovEaxEsi()
	e.CmpEaxImm32(1)
	sizeOKAt := e.JgeRel32()
	if err := a.EmitExit(e, 2, *stackDepth, nil); err != nil {
		return err
	}
	sizeOKOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sizeOKAt, sizeOKOff); err != nil {
		return err
	}
	e.PushRsi()
	e.AddRsiImm32(8)
	e.MovEdiImm32(0)
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(a.SysMmap)
	e.Syscall()
	if err := a.emitMmapFailureGuard(e, *stackDepth); err != nil {
		return err
	}
	e.PopRsi()
	e.MovMem32RaxPtrEsi()
	e.AddRaxImm32(8)
	e.PushRax()
	*stackDepth++
	return nil
}

func (a *SysVUnix) EmitMakeSlice(
	e *x64.Emitter,
	kind ir.IRInstrKind,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = opt
	_ = importPatches
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
	if makeSliceNeedsOverflowGuard(kind) {
		e.CmpRaxImm32(makeSliceMaxElements(kind))
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
	e.MovEdiImm32(0)
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(a.SysMmap)
	e.Syscall()
	if err := a.emitMmapFailureGuard(e, *stackDepth); err != nil {
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
	if err := a.EmitExit(e, allocationLengthTrapExitCode, 0, nil); err != nil {
		return err
	}
	emptyOff := len(e.Buf)
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

func (a *SysVUnix) EmitIslandNew(
	e *x64.Emitter,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in island_new")
	}
	*stackDepth--
	e.PopRax()
	failStackDepth := *stackDepth
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(opt.IslandsDebug)
	headerSize := cfg.HeaderBytes
	e.TestRaxRax()
	negativeAt := e.JlRel32()
	e.CmpRaxImm32(cfg.MaxPayloadBytes)
	overflowAt := e.JgRel32()
	if opt.IslandsDebug && headerSize != x64.IslandsDebugPageSize {
		return fmt.Errorf("internal error: island debug header size mismatch")
	}
	e.AddEaxImm32(headerSize)
	e.MovRsiRax()
	e.PushRax()
	*stackDepth++
	e.MovEdiImm32(0)
	e.MovEdxImm32(3)
	e.MovR10dImm32(0x22)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(a.SysMmap)
	e.Syscall()
	if err := a.emitMmapFailureGuard(e, *stackDepth); err != nil {
		return err
	}
	*stackDepth--
	e.PopRcx()
	e.MovMem32RaxPtrImm32(0, headerSize)
	e.MovMem32Disp32RaxPtrEcx(4)
	e.MovMem32Disp32RaxPtrEcx(8)
	e.MovMem32RaxPtrImm32(12, 0)
	e.PushRax()
	*stackDepth++
	doneAt := e.JmpRel32()
	lengthFailOff := len(e.Buf)
	if err := a.EmitExit(e, allocationLengthTrapExitCode, failStackDepth, nil); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, negativeAt, lengthFailOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, overflowAt, lengthFailOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func (a *SysVUnix) EmitIslandMakeSlice(
	e *x64.Emitter,
	kind ir.IRInstrKind,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = opt
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 2 {
		return fmt.Errorf("stack underflow in island_make_slice")
	}
	*stackDepth -= 2
	e.PopRcx()
	e.PopRax()
	e.TestRcxRcx()
	negativeAt := e.JlRel32()
	emptyAt := e.JzRel32()
	overflowAt := -1
	if makeSliceNeedsOverflowGuard(kind) {
		e.CmpRcxImm32(makeSliceMaxElements(kind))
		overflowAt = e.JgRel32()
	}
	e.PushRax()
	*stackDepth++
	e.PushRcx()
	*stackDepth++
	e.MovRsiRcx()
	if kind == ir.IRIslandMakeSliceI32 {
		e.ShlRsiImm8(2)
	} else if kind == ir.IRIslandMakeSliceU16 {
		e.ShlRsiImm8(1)
	}
	e.MovEdxFromRaxPtrDisp0()
	e.MovR8dFromRaxPtrDisp4()
	e.MovR9Rdx()
	e.AddR9Rsi()
	e.AddR9Imm32(runtimeabi.RegionAllocatorAlignmentBytes - 1)
	e.AndR9Imm32(-runtimeabi.RegionAllocatorAlignmentBytes)
	e.CmpR9R8()
	failAt := e.JaRel32()
	e.AddRdxRax()
	e.MovMem32RaxPtrFromR9d()

	*stackDepth -= 2
	e.PopRcx()
	e.PopRax()
	e.PushRdx()
	*stackDepth++
	e.PushRcx()
	*stackDepth++
	doneAt := e.JmpRel32()

	lengthFailOff := len(e.Buf)
	if err := a.EmitExit(e, allocationLengthTrapExitCode, 0, nil); err != nil {
		return err
	}
	capacityFailOff := len(e.Buf)
	if err := a.EmitExit(e, 1, *stackDepth, nil); err != nil {
		return err
	}
	emptyOff := len(e.Buf)
	e.MovEaxImm32(0)
	e.PushRax()
	e.PushRcx()
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
	if err := x64.PatchRel32(e.Buf, failAt, capacityFailOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	return nil
}

func (a *SysVUnix) emitMmapFailureGuard(e *x64.Emitter, stackSlots int) error {
	e.CmpRaxImm32(-4095)
	failAt := e.JaeRel32()
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := a.EmitExit(e, 2, stackSlots, nil); err != nil {
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

func (a *SysVUnix) EmitIslandFree(
	e *x64.Emitter,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in island_free")
	}
	*stackDepth--
	e.PopRdi()
	if opt.IslandsDebug {
		e.MovEaxFromRdiDisp(12)
		e.TestEaxEax()
		okAt := e.JzRel32()
		if err := a.EmitExit(e, 2, *stackDepth, nil); err != nil {
			return err
		}
		okOff := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, okAt, okOff); err != nil {
			return err
		}
		e.MovRaxRdi()
		e.MovMem32RaxPtrImm32(12, 1)
		e.MovEaxFromRdiDisp(8)
		e.SubEaxImm32(x64.IslandsDebugPageSize)
		e.MovRsiRax()
		e.AddRdiImm32(x64.IslandsDebugPageSize)
		e.MovEdxImm32(0)
		e.MovEaxImm32(a.SysMprotect)
		e.Syscall()
		return nil
	}
	e.MovEsiFromRdiDisp(8)
	e.MovEaxImm32(a.SysMunmap)
	e.Syscall()
	return nil
}

func (a *SysVUnix) EmitIslandReset(
	e *x64.Emitter,
	stackDepth *int,
	opt x64.CodegenOptions,
	importPatches *[]x64obj.ImportPatch,
) error {
	_ = importPatches
	if stackDepth == nil {
		return fmt.Errorf("internal error: missing stackDepth")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in island_reset")
	}
	*stackDepth--
	e.PopRdi()
	if opt.IslandsDebug {
		e.MovEaxFromRdiDisp(12)
		e.TestEaxEax()
		okAt := e.JzRel32()
		if err := a.EmitExit(e, 2, *stackDepth, nil); err != nil {
			return err
		}
		okOff := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, okAt, okOff); err != nil {
			return err
		}
	}
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(opt.IslandsDebug)
	e.MovMem32RdiDispImm32(0, cfg.HeaderBytes)
	e.PushRdi()
	*stackDepth++
	return nil
}
