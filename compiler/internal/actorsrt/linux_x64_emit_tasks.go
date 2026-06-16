package actorsrt

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
)

func emitTaskCanceledCheck(e *x64.Emitter, emitCanceledReturn func()) error {
	actorGroupPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(0)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	notCanceledAt := e.JnzRel32()
	emitCanceledReturn()

	continueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notCanceledAt, continueTo); err != nil {
		return err
	}
	return nil
}

func emitTaskJoinI32CanceledReturn(e *x64.Emitter, result bool) {
	e.XorEaxEax()
	if result {
		e.MovEdxImm32(1)
	}
	e.Ret()
}

func emitTaskJoinTypedCanceledReturn(e *x64.Emitter, slots int) {
	e.XorEaxEax()
	switch slots {
	case 2:
		e.MovEdxImm32(1)
	case 3:
		e.MovEdxImm32(0)
		e.MovR8dImm32(1)
	case 4:
		e.MovEdxImm32(0)
		e.MovR8dImm32(0)
		e.MovR9dImm32(1)
	}
	e.Ret()
}

func emitParkCurrentActorWaitingForTask(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWaitTargetPtrFromEaxToRdi(e)
	e.MovEaxR12d()
	e.MovMem32RdiDispEax(0)

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.XorEaxEax()
	e.MovMem32RdiDispEax(0)

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusWaiting)
}

func emitParkCurrentActorWaitingForTaskUntil(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWaitTargetPtrFromEaxToRdi(e)
	e.MovEaxR12d()
	e.MovMem32RdiDispEax(0)

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEaxR13d()
	e.MovMem32RdiDispEax(0)

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusWaiting)
}

func emitTaskJoinI32(e *x64.Emitter, result bool, callPatches *[]callPatch) error {
	// Arguments: rdi=actor handle, rsi=task error status.
	// task_join_i32 returns rax=value. task_join_result_i32 returns
	// rax=value, rdx=error status.
	e.MovEaxEsi()
	e.TestEaxEax()
	okAt := e.JzRel32()
	e.MovEaxImm32(0)
	if result {
		e.MovEdxEsi()
	}
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}

	e.MovRcxRdi()
	e.MovR12Rcx()
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinI32CanceledReturn(e, result) }); err != nil {
		return err
	}
	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinI32CanceledReturn(e, result) }); err != nil {
		return err
	}
	parkTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetWaitingAt, parkTo); err != nil {
		return err
	}

	emitParkCurrentActorWaitingForTask(e)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loop); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	if result {
		e.MovEdxImm32(0)
	}
	e.Ret()
	return nil
}

func emitTaskJoinUntilI32(e *x64.Emitter, callPatches *[]callPatch) error {
	// Arguments: rdi=actor handle, rsi=task error status, rdx=absolute deadline.
	// Returns task.result_i32: rax=value, rdx=error status. Error 2 means timeout.
	e.MovEaxEsi()
	e.TestEaxEax()
	okAt := e.JzRel32()
	e.XorEaxEax()
	e.MovEdxEsi()
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}

	e.MovRcxRdi()
	e.MovR12Rcx()
	if err := clampEdxNonNegativeIntoR13(e); err != nil {
		return err
	}
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinI32CanceledReturn(e, true) }); err != nil {
		return err
	}

	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinI32CanceledReturn(e, true) }); err != nil {
		return err
	}
	deadlineCheckTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetWaitingAt, deadlineCheckTo); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

	emitParkCurrentActorWaitingForTaskUntil(e)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loop); err != nil {
		return err
	}

	timeoutTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, timeoutAt, timeoutTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.MovEdxImm32(2)
	e.Ret()

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}

func emitTaskPollI32(e *x64.Emitter) error {
	// Arguments: rdi=actor handle, rsi=task error status.
	// Returns task.result_i32: rax=value, rdx=error. Error 2 means not ready.
	e.MovEaxEsi()
	e.TestEaxEax()
	okAt := e.JzRel32()
	e.XorEaxEax()
	e.MovEdxEsi()
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}

	e.MovRcxRdi()
	e.MovR12Rcx()
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(2)
	e.Ret()

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}

func emitTaskResultBegin(e *x64.Emitter) error {
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorTaskCountOff)
	e.XorEaxEax()
	for i := 0; i < 8; i++ {
		e.MovMem32RdiDispEax(actorTaskSlot0Off + int32(i*4))
	}
	e.Ret()
	return nil
}

func emitTaskResultSlot(e *x64.Emitter) error {
	// Args: rdi=index, rsi=value.
	e.PushRdi()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.PopRax()

	for i := 0; i < 7; i++ {
		e.CmpEaxImm32(int32(i))
		notI := e.JnzRel32()
		e.MovMem32RdiDispEsi(actorTaskSlot0Off + int32(i*4))
		e.XorEaxEax()
		e.Ret()
		if err := x64.PatchRel32(e.Buf, notI, len(e.Buf)); err != nil {
			return err
		}
	}

	e.MovMem32RdiDispEsi(actorTaskSlot0Off + 28)
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitTaskResultGet(e *x64.Emitter) error {
	// Args: rdi=index.
	// Returns: eax=current actor task result slot value (or 0 if out of range).
	e.MovEaxEdi()
	e.CmpEaxImm32(8)
	outOfRangeAt := e.JaeRel32()
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.AddRaxImm32(actorTaskSlot0Off)
	e.ShlRdxImm8(2)
	e.AddRaxRdx()
	e.MovEaxFromRaxPtr()
	e.Ret()

	outOfRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, outOfRangeAt, outOfRangeTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitTaskJoinTyped(e *x64.Emitter, slots int, callPatches *[]callPatch) error {
	if slots < 2 || slots > 8 {
		return fmt.Errorf("unsupported typed task join slot count %d", slots)
	}
	staged := slots > 4

	switch slots {
	case 2:
		e.MovEaxEsi()
	case 3:
		e.MovEaxEdx()
	case 4:
		e.MovRaxRcx()
	case 5:
		e.MovEaxR8d()
	case 6:
		e.MovEaxR9d()
	case 7:
		e.MovEaxFromRspDisp(8)
	case 8:
		e.MovEaxFromRspDisp(16)
	}
	e.TestEaxEax()
	okAt := e.JzRel32()
	if staged {
		e.MovEdxEax()
		actorPtrInRax(e)
		e.MovRdiRax()
		for slot := 0; slot < slots; slot++ {
			e.MovMem32RdiDispImm32(actorTaskSlot0Off+int32(slot*4), 0)
		}
		e.MovEaxEdx()
		e.MovMem32RdiDispEax(actorTaskSlot0Off + int32((slots-1)*4))
		e.MovEaxEdx()
		e.Ret()
	}
	e.XorEaxEax()
	switch slots {
	case 2:
		// status is already in rdx.
	case 3:
		e.MovR8Rdx()
		e.MovEdxImm32(0)
	case 4:
		e.MovR9Rcx()
		e.MovEdxImm32(0)
		e.MovR8dImm32(0)
	default:
		e.MovEdxImm32(0)
		e.MovR8dImm32(0)
		e.MovR9dImm32(0)
	}
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}

	e.MovRcxRdi()
	e.MovR12Rcx()
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinTypedCanceledReturn(e, slots) }); err != nil {
		return err
	}
	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	if err := emitTaskCanceledCheck(e, func() { emitTaskJoinTypedCanceledReturn(e, slots) }); err != nil {
		return err
	}
	parkTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetWaitingAt, parkTo); err != nil {
		return err
	}

	emitParkCurrentActorWaitingForTask(e)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loop); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	if staged {
		// rdi currently points to target actor. Copy staged slots to current actor.
		e.PushRdi() // rsp+8 target actor ptr
		actorPtrInRax(e)
		e.PushRax() // rsp+0 current actor ptr
		for slot := 0; slot < slots; slot++ {
			off := actorTaskSlot0Off + int32(slot*4)
			e.MovRaxFromRspDisp(8)
			e.MovRdiRax()
			e.MovEaxFromRdiDisp(off)
			e.MovEdxEax()
			e.MovRaxFromRspDisp(0)
			e.MovRdiRax()
			e.MovEaxEdx()
			e.MovMem32RdiDispEax(off)
		}
		e.MovRaxFromRspDisp(0)
		e.MovRdiRax()
		e.MovEaxFromRdiDisp(actorTaskSlot0Off + int32((slots-1)*4))
		e.AddRspImm32(16)
		e.Ret()
		return nil
	}
	e.MovEaxFromRdiDisp(actorTaskSlot0Off)
	if slots > 1 {
		e.MovEdxFromRdiDisp(actorTaskSlot0Off + 4)
	}
	if slots > 2 {
		e.MovR8dFromRdiDisp(actorTaskSlot0Off + 8)
	}
	if slots > 3 {
		e.MovR9dFromRdiDisp(actorTaskSlot0Off + 12)
	}
	e.Ret()
	return nil
}
