package x64core

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitScalarLoopRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
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
