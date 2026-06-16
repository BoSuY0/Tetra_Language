package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitVectorMapI32AddConstRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
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
