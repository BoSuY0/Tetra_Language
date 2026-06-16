package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitVectorSliceSumRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
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
