package x64core

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitScalarCallLoopRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, callPatches *[]x64obj.CallPatch, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
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
