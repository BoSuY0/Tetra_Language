package x64abi

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

const (
	winImportExitProcess    = "kernel32.ExitProcess"
	winImportGetStdHandle   = "kernel32.GetStdHandle"
	winImportVirtualAlloc   = "kernel32.VirtualAlloc"
	winImportVirtualFree    = "kernel32.VirtualFree"
	winImportVirtualProtect = "kernel32.VirtualProtect"
	winImportWriteFile      = "kernel32.WriteFile"
)

type Win64 struct{}

func NewWin64() *Win64 { return &Win64{} }

func (a *Win64) SpillParams(e *x64.Emitter, fn ir.IRFunc) {
	for i := 0; i < fn.ParamSlots; i++ {
		off := -int32((i + 1) * 8)
		switch i {
		case 0:
			e.MovMem64RbpDispRcx(off)
		case 1:
			e.MovMem64RbpDispRdx(off)
		case 2:
			e.MovMem64RbpDispR8(off)
		case 3:
			e.MovMem64RbpDispR9(off)
		default:
			stackOff := int32(48 + 8*(i-4))
			e.MovRaxFromRbpDisp(stackOff)
			e.MovMem64RbpDispRax(off)
		}
	}
}

func (a *Win64) EmitCall(e *x64.Emitter, instr ir.IRInstr, stackDepth *int, callPatches *[]x64obj.CallPatch) error {
	if stackDepth == nil || callPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/callPatches")
	}
	if instr.Name == "" {
		return fmt.Errorf("call is missing target name")
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf("call %q has negative ABI slots args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots)
	}
	if instr.RetSlots > maxCallReturnSlots {
		return fmt.Errorf("call %q has unsupported return slots %d (max=%d)", instr.Name, instr.RetSlots, maxCallReturnSlots)
	}
	if *stackDepth < instr.ArgSlots {
		return fmt.Errorf("stack underflow in function '%s'", instr.Name)
	}
	stackBefore := *stackDepth
	*stackDepth -= instr.ArgSlots

	extra := 0
	if instr.ArgSlots > 4 {
		extra = instr.ArgSlots - 4
	}
	frameBytes := int32(32 + extra*8)
	if (stackBefore+extra)%2 != 0 {
		frameBytes += 8
	}
	if frameBytes > 0 {
		e.SubRspImm32(frameBytes)
	}

	srcBase := frameBytes
	for i := 0; i < instr.ArgSlots; i++ {
		srcOffset := srcBase + int32(8*(instr.ArgSlots-1-i))
		switch i {
		case 0:
			e.MovRcxFromRspDisp(srcOffset)
		case 1:
			e.MovRdxFromRspDisp(srcOffset)
		case 2:
			e.MovR8FromRspDisp(srcOffset)
		case 3:
			e.MovR9FromRspDisp(srcOffset)
		default:
			e.MovRaxFromRspDisp(srcOffset)
			dstOffset := int32(32 + 8*(i-4))
			e.MovMem64RspDispRax(dstOffset)
		}
	}

	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: instr.Name})

	if frameBytes > 0 {
		e.AddRspImm32(frameBytes)
	}
	if instr.ArgSlots > 0 {
		e.AddRspImm32(int32(instr.ArgSlots * 8))
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
		e.PushR15()
		*stackDepth++
	}
	return nil
}

func (a *Win64) EmitWriteStdout(e *x64.Emitter, stackDepth *int, importPatches *[]x64obj.ImportPatch) error {
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}

	{
		frameBytes := int32(32)
		if (*stackDepth)%2 != 0 {
			frameBytes += 8
		}
		e.SubRspImm32(frameBytes)
		e.MovEcxImm32(0xFFFFFFF5)
		at := e.CallRipDisp32()
		*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportGetStdHandle})
		e.AddRspImm32(frameBytes)
	}

	if *stackDepth < 2 {
		return fmt.Errorf("stack underflow in write")
	}
	*stackDepth -= 2
	e.PopR8()
	e.PopRdx()

	{
		extraArgs := 1
		localSlots := 1
		frameBytes := int32(32 + (extraArgs+localSlots)*8)
		if (*stackDepth+extraArgs+localSlots)%2 != 0 {
			frameBytes += 8
		}
		e.SubRspImm32(frameBytes)
		e.MovRcxRax()
		e.LeaR9RspDisp(32 + int32(extraArgs*8))
		e.MovEaxImm32(0)
		e.MovMem32RspDispEax(32)
		e.MovMem32RspDispEax(36)
		at := e.CallRipDisp32()
		*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportWriteFile})
		e.AddRspImm32(frameBytes)
	}

	return nil
}

func (a *Win64) EmitExit(e *x64.Emitter, code int32, stackSlots int, importPatches *[]x64obj.ImportPatch) error {
	if importPatches == nil {
		return fmt.Errorf("internal error: missing importPatches")
	}
	frameBytes := int32(32)
	if (stackSlots)%2 != 0 {
		frameBytes += 8
	}
	e.SubRspImm32(frameBytes)
	e.MovEcxImm32(uint32(code))
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportExitProcess})
	e.AddRspImm32(frameBytes)
	return nil
}

func (a *Win64) EmitAllocBytes(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error {
	_ = opt
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in alloc_bytes")
	}
	*stackDepth--
	e.PopRdx()
	frameBytes := int32(32)
	if (*stackDepth)%2 != 0 {
		frameBytes += 8
	}
	e.SubRspImm32(frameBytes)
	e.MovEcxImm32(0)
	e.MovR8dImm32(0x3000)
	e.MovR9dImm32(0x04)
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportVirtualAlloc})
	e.AddRspImm32(frameBytes)
	e.PushRax()
	*stackDepth++
	return nil
}

func (a *Win64) EmitMakeSlice(e *x64.Emitter, kind ir.IRInstrKind, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error {
	_ = opt
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in make_slice")
	}
	*stackDepth--
	e.PopRax()
	e.PushRax()
	*stackDepth++
	if kind == ir.IRMakeSliceI32 {
		e.ShlRaxImm8(2)
	} else if kind == ir.IRMakeSliceU16 {
		e.ShlRaxImm8(1)
	}
	e.MovRdxRax()
	frameBytes := int32(32)
	if (*stackDepth)%2 != 0 {
		frameBytes += 8
	}
	e.SubRspImm32(frameBytes)
	e.MovEcxImm32(0)
	e.MovR8dImm32(0x3000)
	e.MovR9dImm32(0x04)
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportVirtualAlloc})
	e.AddRspImm32(frameBytes)

	*stackDepth--
	e.PopRcx()
	e.PushRax()
	*stackDepth++
	e.PushRcx()
	*stackDepth++
	return nil
}

func (a *Win64) EmitIslandNew(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error {
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in island_new")
	}
	*stackDepth--
	e.PopRdx()
	headerSize := int32(16)
	if opt.IslandsDebug {
		headerSize = x64.IslandsDebugPageSize
	}
	e.AddEdxImm32(headerSize)
	e.PushRdx()
	*stackDepth++

	frameBytes := int32(32)
	if (*stackDepth)%2 != 0 {
		frameBytes += 8
	}
	e.SubRspImm32(frameBytes)
	e.MovEcxImm32(0)
	e.MovR8dImm32(0x3000)
	e.MovR9dImm32(0x04)
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportVirtualAlloc})
	e.AddRspImm32(frameBytes)

	*stackDepth--
	e.PopRcx()
	e.MovMem32RaxPtrImm32(0, headerSize)
	e.MovMem32Disp32RaxPtrEcx(4)
	e.MovMem32Disp32RaxPtrEcx(8)
	e.MovMem32RaxPtrImm32(12, 0)
	e.PushRax()
	*stackDepth++
	return nil
}

func (a *Win64) EmitIslandMakeSlice(e *x64.Emitter, kind ir.IRInstrKind, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error {
	_ = importPatches
	_ = opt
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}
	if *stackDepth < 2 {
		return fmt.Errorf("stack underflow in island_make_slice")
	}
	*stackDepth -= 2
	e.PopRcx()
	e.PopRax()
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

	failOff := len(e.Buf)
	if err := a.EmitExit(e, 1, *stackDepth, importPatches); err != nil {
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

func (a *Win64) EmitIslandFree(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error {
	if stackDepth == nil || importPatches == nil {
		return fmt.Errorf("internal error: missing stackDepth/importPatches")
	}
	if *stackDepth < 1 {
		return fmt.Errorf("stack underflow in island_free")
	}
	*stackDepth--
	e.PopRcx()
	if opt.IslandsDebug {
		e.MovRdiRcx()
		e.MovEaxFromRdiDisp(12)
		e.TestEaxEax()
		okAt := e.JzRel32()
		if err := a.EmitExit(e, 2, *stackDepth, importPatches); err != nil {
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
		e.MovRdxRax()
		e.AddRdiImm32(x64.IslandsDebugPageSize)
		e.MovRcxRdi()

		frameBytes := int32(32)
		if (*stackDepth)%2 != 0 {
			frameBytes += 8
		}
		e.SubRspImm32(frameBytes)
		e.MovR8dImm32(0x01)
		e.LeaR9RspDisp(0)
		at := e.CallRipDisp32()
		*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportVirtualProtect})
		e.AddRspImm32(frameBytes)
		return nil
	}

	frameBytes := int32(32)
	if (*stackDepth)%2 != 0 {
		frameBytes += 8
	}
	e.SubRspImm32(frameBytes)
	e.MovEdxImm32(0)
	e.MovR8dImm32(0x8000)
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, x64obj.ImportPatch{At: at, Name: winImportVirtualFree})
	e.AddRspImm32(frameBytes)
	return nil
}
