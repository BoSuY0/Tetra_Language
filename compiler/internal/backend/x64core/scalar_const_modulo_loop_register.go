package x64core

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitScalarConstModuloLoopRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
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
