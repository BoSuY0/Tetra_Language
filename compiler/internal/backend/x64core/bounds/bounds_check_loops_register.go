package bounds

import (
	"encoding/binary"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	machinebounds "tetra_language/compiler/internal/machine/bounds"
)

func EmitRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	opt x64.CodegenOptions,
	flush func() error,
) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	plan, ok, err := machinebounds.BoundsCheckLoopsPlanFromStackIR(fn)
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
	fillStart := len(e.Buf)
	e.CmpRcxImm32(plan.SliceLength)
	fillExitAt := e.JgeRel32()
	e.MovEaxEcx()
	e.MovR8dImm32(uint32(plan.FillModulus))
	e.Cdq()
	emitIdivR8d(e)
	e.MovR8dEdx()
	e.MovRaxRcx()
	e.ShlRaxImm8(2)
	e.MovRdxR9()
	e.AddRaxRdx()
	e.MovMem32RaxPtrR8d()
	e.AddEcxImm8(byte(plan.Step))
	fillBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, fillBackAt, fillStart); err != nil {
		return true, err
	}
	fillExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fillExitAt, fillExitTo); err != nil {
		return true, err
	}

	e.XorEcxEcx()
	e.MovR10dImm32(0)
	hotStart := len(e.Buf)
	e.CmpRcxImm32(plan.HotLoopBound)
	hotExitAt := e.JgeRel32()
	e.MovRaxRcx()
	emitImulRaxImm32(e, plan.IndexMultiplier)
	e.MovR8dImm32(uint32(plan.SliceLength))
	e.Cdq()
	emitIdivR8d(e)
	e.MovRaxRdx()
	e.ShlRaxImm8(2)
	e.MovRdxR9()
	e.AddRaxRdx()
	e.MovEaxFromRaxPtr()
	e.MovEdxEax()
	emitAddR10dEdx(e)
	e.AddEcxImm8(byte(plan.Step))
	hotBackAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, hotBackAt, hotStart); err != nil {
		return true, err
	}
	hotExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, hotExitAt, hotExitTo); err != nil {
		return true, err
	}

	e.MovEaxR10d()
	e.TestEaxEax()
	negativeAt := e.JlRel32()
	e.MovEaxImm32(uint32(plan.SuccessReturn))
	doneAt := e.JmpRel32()
	negativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, negativeAt, negativeTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.FailureReturn))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	if flush != nil {
		if err := flush(); err != nil {
			return true, err
		}
	}
	e.Leave()
	e.Ret()
	return true, nil
}

func scalarRegisterSlotOffset(slot int) int32 {
	return -int32((slot + 1) * 8)
}

func emitIdivR8d(e *x64.Emitter) {
	e.Emit(0x41, 0xF7, 0xF8)
}

func emitAddR10dEdx(e *x64.Emitter) {
	e.Emit(0x41, 0x01, 0xD2)
}

func emitImulRaxImm32(e *x64.Emitter, imm int32) {
	e.Emit(0x48, 0x69, 0xC0)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(imm))
	e.Emit(buf[:]...)
}
