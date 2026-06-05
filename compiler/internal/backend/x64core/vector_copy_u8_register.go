package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitVectorCopyU8RegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, opt x64.CodegenOptions) (bool, error) {
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
	e.Leave()
	e.Ret()
	return true, nil
}
