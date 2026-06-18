package x64core

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitRecursionBenchmarkRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, callPatches *[]x64obj.CallPatch, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	callKind, callInfo, ok := scalarCallABIFromBackendABI(abi)
	if !ok || callKind != scalarCallABISysV {
		return false, nil
	}
	if plan, ok, err := machine.RecursionFibPlanFromStackIRWithCallABI(fn, callInfo); err != nil || ok {
		if err != nil || !ok {
			return ok, err
		}
		return emitRecursionFibRegisterFunction(e, fn, abi, callKind, plan, callPatches, flush)
	}
	if plan, ok, err := machine.RecursionMainPlanFromStackIRWithCallABI(fn, callInfo); err != nil || ok {
		if err != nil || !ok {
			return ok, err
		}
		return emitRecursionMainRegisterFunction(e, fn, abi, callKind, plan, callPatches, flush)
	}
	return false, nil
}

func emitRecursionFibRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callKind scalarCallABIKind,
	plan machine.RecursionFibPlan,
	callPatches *[]x64obj.CallPatch,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize((fn.LocalSlots + 1) * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	paramOffset := scalarRegisterSlotOffset(plan.ParamLocal)
	scratchOffset := scalarRegisterSlotOffset(fn.LocalSlots)
	e.MovEaxFromRbpDisp(paramOffset)
	e.CmpEaxImm32(2)
	baseAt := e.JlRel32()

	e.MovEaxFromRbpDisp(paramOffset)
	e.SubEaxImm32(1)
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovMem64RbpDispRax(scratchOffset)

	e.MovEaxFromRbpDisp(paramOffset)
	e.SubEaxImm32(2)
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovEcxEax()
	e.MovEaxFromRbpDisp(scratchOffset)
	e.AddEaxEcx()
	doneAt := e.JmpRel32()

	baseTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, baseAt, baseTo); err != nil {
		return true, err
	}
	e.MovEaxFromRbpDisp(paramOffset)
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

func emitRecursionMainRegisterFunction(
	e *x64.Emitter,
	fn ir.IRFunc,
	abi x64abi.ABI,
	callKind scalarCallABIKind,
	plan machine.RecursionMainPlan,
	callPatches *[]x64obj.CallPatch,
	flush runtimeHeapTelemetryFlushFunc,
) (bool, error) {
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(fn.LocalSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)

	totalOffset := scalarRegisterSlotOffset(plan.TotalLocal)
	indexOffset := scalarRegisterSlotOffset(plan.IndexLocal)
	e.XorEcxEcx()
	e.XorEaxEax()

	loopStart := len(e.Buf)
	e.CmpRcxImm32(plan.LoopBound)
	exitAt := e.JgeRel32()
	e.MovMem64RbpDispRax(totalOffset)
	e.MovMem64RbpDispRcx(indexOffset)
	e.MovEaxImm32(uint32(plan.CallArg))
	emitMoveRaxToScalarCallArg(e, callKind, 0)
	if err := emitScalarLoopCall(e, callKind, plan.CallName, callPatches); err != nil {
		return true, err
	}
	e.MovEcxEax()
	e.MovEaxFromRbpDisp(totalOffset)
	e.AddEaxEcx()
	e.MovMem64RbpDispRax(totalOffset)
	e.MovRaxFromRbpDisp(indexOffset)
	e.MovEcxEax()
	e.AddEcxImm8(1)
	e.MovEaxFromRbpDisp(totalOffset)
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return true, err
	}

	exitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, exitAt, exitTo); err != nil {
		return true, err
	}
	e.MovMem64RbpDispRax(totalOffset)
	if err := flush.emit(); err != nil {
		return true, err
	}
	e.MovEaxFromRbpDisp(totalOffset)
	e.CmpEaxImm32(plan.SuccessTotal)
	failAt := e.JnzRel32()
	e.MovEaxImm32(uint32(plan.TrueReturnImm))
	doneAt := e.JmpRel32()
	failTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, failAt, failTo); err != nil {
		return true, err
	}
	e.MovEaxImm32(uint32(plan.FalseReturnImm))
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return true, err
	}
	e.Leave()
	e.Ret()
	return true, nil
}
