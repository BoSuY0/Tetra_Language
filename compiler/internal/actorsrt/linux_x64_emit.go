package actorsrt

import (
	"fmt"
	"hash/fnv"

	"tetra_language/compiler/internal/backend/x64"
)

const (
	msgNextOff   = 0  // u64
	msgSenderOff = 8  // u32
	msgValueOff  = 12 // u32
	msgTagOff    = 16 // u32
	msgCountOff  = 20 // u32
	msgPayload0  = 24 // u32[8]
	msgSize      = 56
)

func fnv1a32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func emitMmapAnon(e *x64.Emitter, length int32, sysMmap uint32, mapFlags uint32) {
	// mmap(NULL, length, PROT_READ|PROT_WRITE, flags, -1, 0)
	e.MovEdiImm32(0)
	e.MovEaxImm32(uint32(length))
	e.MovRsiRax()
	e.MovEdxImm32(3)
	e.MovR10dImm32(mapFlags)
	e.MovR8dImm32(0xFFFFFFFF)
	e.MovR9dImm32(0)
	e.MovEaxImm32(sysMmap)
	e.Syscall()
}

func emitEntry(e *x64.Emitter, mainSymbol string, sysMmap uint32, mapFlags uint32, callPatches *[]callPatch, leaPatches *[]leaPatch) error {
	// Allocate scheduler + actor slots.
	emitMmapAnon(e, schedAllocSize, sysMmap, mapFlags)
	e.MovR15Rax()

	// sched.actorsPtr = sched + schedSize
	e.MovRdiR15()
	e.AddRdiImm32(schedSize)
	e.MovRaxRdi()
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedActorsPtrOff)

	// sched.capacity = maxActors, sched.count = 1, sched.currentIdx = 0
	e.MovMem32RdiDispImm32(schedCapacityOff, maxActors)
	e.MovMem32RdiDispImm32(schedCountOff, 1)
	e.MovMem32RdiDispImm32(schedCurrentIdxOff, 0)
	e.MovMem32RdiDispImm32(schedGroupCountOff, 0)
	e.MovMem32RdiDispImm32(schedCloseGroupOff, 0)
	e.MovMem32RdiDispImm32(schedCurrentGroupOff, 0)
	e.MovMem32RdiDispImm32(schedSpawnGroupOff, 0)
	e.MovMem32RdiDispImm32(schedTimeMsOff, 0)

	// Message pool
	emitMmapAnon(e, msgPoolSize, sysMmap, mapFlags)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgBaseOff)
	e.MovMem64RdiDispRax(schedMsgBumpOff)
	e.AddRaxImm32(msgPoolSize)
	e.MovMem64RdiDispRax(schedMsgEndOff)

	// actor0 = sched.actorsPtr + 0
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRdxRax() // actor0 ptr in rdx
	e.MovRdiRdx()

	// actor0.status = ready
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	// actor0.entryID = hash(main symbol)
	e.MovMem32RdiDispImm32(actorEntryIDOff, int32(fnv1a32(mainSymbol)))
	// actor0.mailbox = empty
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorLastSenderOff, 0)
	e.MovMem32RdiDispImm32(actorExitCodeOff, 0)
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	e.MovMem32RdiDispImm32(actorTaskGroupOff, 0)
	for i := 0; i < maxActorStateSlots; i++ {
		e.MovMem32RdiDispImm32(actorStateSlot0Off+int32(i*4), 0)
	}

	// Allocate actor0 stack and initialize its starting context. initRsp is in rcx.
	e.PushRdx()
	if err := emitInitActorStack(e, sysMmap, mapFlags, leaPatches); err != nil {
		return err
	}
	e.PopRdx()
	// Store actor0.rsp = initRsp
	e.MovRdiRdx()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorRspOff)

	// Switch to actor0 to start execution.
	e.MovRdiR15()
	e.AddRdiImm32(schedRspOff)
	e.MovRaxRdx()
	e.MovRsiRax()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_switch_to"})

	// Scheduler loop.
	loopStart := len(e.Buf)

	// Load count into ecx.
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	// candidate = currentIdx (eax)
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	// tries = count (edx)
	e.MovEdxEcx()
	e.TestEdxEdx()
	noReadyAt := e.JzRel32()

	tryLoop := len(e.Buf)
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	// candidate++
	e.AddEaxImm32(1)
	// if candidate == count => candidate = 0
	e.CmpEaxEcx()
	skipWrapAt := e.JnzRel32()
	e.MovEaxImm32(0)
	skipWrapTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipWrapAt, skipWrapTo); err != nil {
		return err
	}

	// Save candidate index.
	e.PushRax()

	// actorPtr = sched.actorsPtr + candidate<<actorSizeShift
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
	// status = actor.status
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusReady)
	readyAt := e.JzRel32()
	e.CmpEaxImm32(statusSleeping)
	sleepingAt := e.JzRel32()
	e.CmpEaxImm32(statusBlocked)
	blockedAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	waitingAt := e.JzRel32()
	notReadyAt := e.JmpRel32()

	// Sleeping actors become ready when their group is canceled.
	sleepingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sleepingAt, sleepingTo); err != nil {
		return err
	}
	e.PushRdi()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	hasGroupAt := e.JnzRel32()
	e.PopRdi()
	noGroupCheckWakeAt := e.JmpRel32()

	hasGroupTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, hasGroupAt, hasGroupTo); err != nil {
		return err
	}
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	canceledAt := e.JzRel32()
	e.PopRdi()
	notCanceledCheckWakeAt := e.JmpRel32()

	canceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, canceledAt, canceledTo); err != nil {
		return err
	}
	e.PopRdi()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	canceledReadyAt := e.JmpRel32()

	// Sleeping actors also become ready once the logical clock reaches wake_at.
	checkWakeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupCheckWakeAt, checkWakeTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notCanceledCheckWakeAt, checkWakeTo); err != nil {
		return err
	}
	e.PushRdi()
	e.MovEaxFromRspDisp(8)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(0)
	e.PopRdi()
	e.PushRdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.PopRdi()
	e.CmpEaxEcx()
	dueAt := e.JaeRel32()
	notDueAt := e.JmpRel32()

	dueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, dueAt, dueTo); err != nil {
		return err
	}
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	dueReadyAt := e.JmpRel32()

	// Timed receive waiters become ready once their deadline is due.
	blockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, blockedAt, blockedTo); err != nil {
		return err
	}
	blockedReadyAts, blockedNotReadyAts, err := emitBlockedDeadlineWakeCheck(e)
	if err != nil {
		return err
	}

	// Task join waiters become ready when the target is done.
	waitingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, waitingAt, waitingTo); err != nil {
		return err
	}
	waitReadyAts, waitNotReadyAts, err := emitWaitingTaskWakeCheck(e)
	if err != nil {
		return err
	}

	// Ready: restore candidate index and run it.
	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, canceledReadyAt, readyTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, dueReadyAt, readyTo); err != nil {
		return err
	}
	for _, at := range blockedReadyAts {
		if err := x64.PatchRel32(e.Buf, at, readyTo); err != nil {
			return err
		}
	}
	for _, at := range waitReadyAts {
		if err := x64.PatchRel32(e.Buf, at, readyTo); err != nil {
			return err
		}
	}
	e.PopRax()

	// sched.currentIdx = candidate
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCurrentIdxOff)

	// actorPtr = sched.actorsPtr + candidate<<actorSizeShift
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.PushRax()
	e.MovRdiRax()
	e.MovEdxFromRdiDisp(actorTaskGroupOff)
	e.MovEaxEdx()
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCurrentGroupOff)
	e.PopRax()
	e.PushRax()
	e.MovRdiRax()
	storeActorSavedGroupForActorPtrInRdiGroupInRdx(e)
	e.PopRax()

	// switch_to(&sched.rsp, &actor.rsp)
	e.MovRdiR15()
	e.AddRdiImm32(schedRspOff)
	e.MovRsiRax()
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_switch_to"})
	backAt := e.JmpRel32()

	// Not ready: restore candidate and continue loop.
	notReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notReadyAt, notReadyTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notDueAt, notReadyTo); err != nil {
		return err
	}
	for _, at := range blockedNotReadyAts {
		if err := x64.PatchRel32(e.Buf, at, notReadyTo); err != nil {
			return err
		}
	}
	for _, at := range waitNotReadyAts {
		if err := x64.PatchRel32(e.Buf, at, notReadyTo); err != nil {
			return err
		}
	}
	e.PopRax()
	e.AddEdxImm32(-1)
	e.TestEdxEdx()
	noReadyAt2 := e.JzRel32()
	jmpTry := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpTry, tryLoop); err != nil {
		return err
	}

	// No ready actors: advance logical time to the next sleeping actor.
	noReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noReadyAt, noReadyTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, noReadyAt2, noReadyTo); err != nil {
		return err
	}
	if err := emitAdvanceClockToNextSleepingWake(e, loopStart); err != nil {
		return err
	}

	// Patch loop-back.
	if err := x64.PatchRel32(e.Buf, backAt, loopStart); err != nil {
		return err
	}

	// Return actor0 exit code.
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.Ret()
	return nil
}

func emitSwitchTo(e *x64.Emitter, callPatches *[]callPatch) error {
	_ = callPatches
	// Signature:
	//   __tetra_switch_to(fromRspPtr: ptr, toRspPtr: ptr)
	//
	// Saves callee-saved regs by pushing them, stores rsp into *fromRspPtr,
	// then restores rsp from *toRspPtr and pops regs + ret.
	e.PushRbx()
	e.PushRbp()
	e.PushR12()
	e.PushR13()
	e.PushR14()
	e.PushR15()

	// *from = rsp
	e.MovMem64RdiDispRsp(0)
	// rsp = *to
	e.MovRdiRsi()
	e.MovRspFromRdiDisp(0)

	e.PopR15()
	e.PopR14()
	e.PopR13()
	e.PopR12()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return nil
}

func emitAdvanceClockToNextSleepingWake(e *x64.Emitter, loopStart int) error {
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	e.XorEaxEax()
	e.MovEdxImm32(0x7fffffff)

	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	scanDoneAt := e.JaeRel32()
	e.PushRax()
	e.PushRcx()

	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusSleeping)
	eligibleSleepingAt := e.JzRel32()
	e.CmpEaxImm32(statusBlocked)
	eligibleBlockedAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	eligibleWaitingAt := e.JzRel32()
	notEligibleAt := e.JmpRel32()

	eligibleTo := len(e.Buf)
	for _, at := range []int{eligibleSleepingAt, eligibleBlockedAt, eligibleWaitingAt} {
		if err := x64.PatchRel32(e.Buf, at, eligibleTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRspDisp(8)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(0)
	e.TestEaxEax()
	noWakeAt := e.JzRel32()
	e.CmpEaxEdx()
	notEarlierAt := e.JaeRel32()
	e.MovEdxEax()

	continueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notEligibleAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, noWakeAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notEarlierAt, continueTo); err != nil {
		return err
	}
	e.PopRcx()
	e.PopRax()
	e.AddEaxImm32(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		return err
	}

	scanDoneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, scanDoneAt, scanDoneTo); err != nil {
		return err
	}
	e.MovEaxEdx()
	e.CmpEaxImm32(0x7fffffff)
	noSleepingAt := e.JzRel32()
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedTimeMsOff)
	loopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, loopAt, loopStart); err != nil {
		return err
	}

	noSleepingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noSleepingAt, noSleepingTo); err != nil {
		return err
	}
	return nil
}

func emitBlockedDeadlineWakeCheck(e *x64.Emitter) ([]int, []int, error) {
	// Candidate actor index is saved at rsp+0 by the scheduler scan.
	e.MovEaxFromRspDisp(0)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(0)
	e.MovEaxEcx()
	e.TestEaxEax()
	noDeadlineAt := e.JzRel32()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.CmpEaxEcx()
	dueAt := e.JaeRel32()
	notDueAt := e.JmpRel32()

	dueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, dueAt, dueTo); err != nil {
		return nil, nil, err
	}
	e.MovEaxFromRspDisp(0)
	actorPtrFromEaxToRdi(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	readyAt := e.JmpRel32()

	return []int{readyAt}, []int{noDeadlineAt, notDueAt}, nil
}

func emitWaitingTaskWakeCheck(e *x64.Emitter) ([]int, []int, error) {
	// Candidate actor index is saved at rsp+0 by the scheduler scan.
	e.MovEaxFromRspDisp(0)
	actorWaitTargetPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(0)
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	targetDoneAt := e.JzRel32()
	e.MovEaxFromRspDisp(0)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(0)
	e.MovEaxEcx()
	e.TestEaxEax()
	noDeadlineAt := e.JzRel32()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.CmpEaxEcx()
	deadlineDueAt := e.JaeRel32()
	notReadyAt := e.JmpRel32()

	doneReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetDoneAt, doneReadyTo); err != nil {
		return nil, nil, err
	}
	if err := x64.PatchRel32(e.Buf, deadlineDueAt, doneReadyTo); err != nil {
		return nil, nil, err
	}
	e.MovEaxFromRspDisp(0)
	actorPtrFromEaxToRdi(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	readyAt := e.JmpRel32()

	return []int{readyAt}, []int{noDeadlineAt, notReadyAt}, nil
}

func emitActorYield(e *x64.Emitter, callPatches *[]callPatch) error {
	// switch_to(&actor.rsp, &sched.rsp)
	// rdi = &actor.rsp (actorPtr)
	actorPtrInRax(e)
	e.MovRdiRax()
	// rsi = &sched.rsp
	e.MovRaxR15()
	e.MovRsiRax()
	e.AddRsiImm32(schedRspOff)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_switch_to"})
	e.Ret()
	return nil
}

func emitActorYieldNow(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitActorExit(e *x64.Emitter, callPatches *[]callPatch) error {
	// Argument: exitCode in edi.
	// actor.exitCode = edi; actor.status = done; yield.
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorExitCodeOff)
	e.MovMem32RdiDispImm32(actorStatusOff, statusDone)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	// Should never resume.
	e.MovEaxImm32(0)
	e.Ret()
	return nil
}

func emitActorTrampoline(e *x64.Emitter, callPatches *[]callPatch) error {
	// entryID := currentActor.entryID
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorEntryIDOff)
	e.MovEdiEax()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_dispatch"})
	// exit(code)
	e.MovEdiEax()
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_exit"})
	e.MovEaxImm32(0)
	e.Ret()
	return nil
}

func emitTaskSpawnI32(e *x64.Emitter, callPatches *[]callPatch) error {
	return emitTaskSpawnI32To(e, "__tetra_actor_spawn", callPatches)
}

func emitTaskSpawnI32To(e *x64.Emitter, actorSpawn string, callPatches *[]callPatch) error {
	// Argument: entryID in edi.
	// Returns task.i32 layout: rax=actor handle, rdx=error status.
	e.PushRdi()
	e.MovEaxR14d()
	e.TestEaxEax()
	noGroupAt := e.JzRel32()

	e.PushRax()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupOpen)
	openAt := e.JzRel32()
	e.PopRax()
	e.PopRdi()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	openTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, openAt, openTo); err != nil {
		return err
	}
	e.PopRdx()
	e.PopRdi()
	e.PushRdx()
	setPendingSpawnGroupFromEdx(e)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: actorSpawn})
	e.CmpEaxImm32(-1)
	groupSpawnedAt := e.JnzRel32()
	e.PopRdx()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	groupSpawnedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, groupSpawnedAt, groupSpawnedTo); err != nil {
		return err
	}
	e.PopRdx()
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.PushRax()
	e.MovRcxRax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
	storeActorSavedGroupForActorPtrInRdiGroupInRdx(e)
	e.PopRax()
	e.MovEdxImm32(0)
	e.Ret()

	noGroupTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, noGroupTo); err != nil {
		return err
	}
	e.PopRdi()
	e.MovEdxImm32(0)
	setPendingSpawnGroupFromEdx(e)
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: actorSpawn})
	e.CmpEaxImm32(-1)
	spawnedAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	spawnedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, spawnedAt, spawnedTo); err != nil {
		return err
	}
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}

func actorPtrFromR12ToRdi(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRbxR12()
	e.ShlRbxImm8(actorSizeShift)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func actorPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func actorGroupPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedActorGroup0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func actorGroupPtrFromR12ToRdi(e *x64.Emitter) {
	e.MovRbxR12()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedActorGroup0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func storeActorGroupForHandleInRaxGroupInRdx(e *x64.Emitter) {
	e.PushRax()
	actorGroupPtrFromEaxToRdi(e)
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(0)
	e.PopRax()
}

func storeActorSavedGroupForActorPtrInRdiGroupInRdx(e *x64.Emitter) {
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorTaskGroupOff)
	e.PushRdx()
	e.MovRaxFromRdiDisp(actorRspOff)
	e.MovRdiRax()
	e.PopRax()
	e.MovMem64RdiDispRax(8)
}

func setPendingSpawnGroupFromEdx(e *x64.Emitter) {
	e.MovEaxEdx()
	e.MovMem32R15DispEax(schedSpawnGroupOff)
}

func groupStatePtrFromEdi(e *x64.Emitter) {
	e.MovEaxEdi()
	e.AddEaxImm32(-1)
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedGroupState0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func emitTaskGroupOpen(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedGroupCountOff)
	e.CmpEaxImm32(maxTaskGroups)
	fullAt := e.JaeRel32()
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(schedGroupCountOff)
	e.PushRax()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovMem32RdiDispImm32(0, taskGroupOpen)
	e.PopRax()
	e.Ret()

	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitTaskGroupCancel(e *x64.Emitter) error {
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.PushRax()
	groupStatePtrFromEdi(e)
	e.MovMem32RdiDispImm32(0, taskGroupCanceled)
	e.PopRax()
	e.Ret()
	return nil
}

func emitTaskGroupCurrent(e *x64.Emitter) error {
	e.MovEaxR14d()
	e.Ret()
	return nil
}

func emitTaskGroupStatus(e *x64.Emitter) error {
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.Ret()
	return nil
}

func emitTaskIsCanceled(e *x64.Emitter) error {
	return emitTaskCancellationStatus(e)
}

func emitTaskCheckpoint(e *x64.Emitter) error {
	return emitTaskCancellationStatus(e)
}

func emitTaskCancellationStatus(e *x64.Emitter) error {
	e.MovEaxR14d()
	e.TestEaxEax()
	hasGroupAt := e.JnzRel32()
	e.Ret()

	hasGroupTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, hasGroupAt, hasGroupTo); err != nil {
		return err
	}
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	canceledAt := e.JzRel32()
	e.XorEaxEax()
	e.Ret()

	canceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, canceledAt, canceledTo); err != nil {
		return err
	}
	e.MovEaxImm32(1)
	e.Ret()
	return nil
}

func clampEdiNonNegativeIntoEcx(e *x64.Emitter) error {
	e.MovEcxEdi()
	e.CmpEdiImm32(0)
	nonNegativeAt := e.JgeRel32()
	e.MovEcxImm32(0)
	nonNegativeTo := len(e.Buf)
	return x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo)
}

func clampEdxNonNegativeIntoR13(e *x64.Emitter) error {
	e.MovR13Rdx()
	e.CmpEdxImm32(0)
	nonNegativeAt := e.JgeRel32()
	e.XorEaxEax()
	e.MovR13Rax()
	nonNegativeTo := len(e.Buf)
	return x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo)
}

func emitTimeNowMs(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.Ret()
	return nil
}

func emitTimerReadyMs(e *x64.Emitter) error {
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.CmpEaxEcx()
	readyAt := e.JaeRel32()
	e.XorEaxEax()
	e.Ret()

	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	e.MovEaxImm32(1)
	e.Ret()
	return nil
}

func emitSleepMs(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.MovEaxEcx()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.XorEaxEax()
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.PushRcx()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.PopRcx()
	e.AddEaxEcx()
	e.PushRax()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.PopRax()
	e.MovMem32RdiDispEax(0)

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusSleeping)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSleepUntilMs(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.PushRcx()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.PopRcx()
	e.CmpEaxEcx()
	dueAt := e.JaeRel32()

	e.PushRcx()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.PopRax()
	e.MovMem32RdiDispEax(0)

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusSleeping)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	e.XorEaxEax()
	e.Ret()

	dueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, dueAt, dueTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitDeadlineMs(e *x64.Emitter) error {
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.AddEaxEcx()
	e.Ret()
	return nil
}

func emitTaskGroupClose(e *x64.Emitter, callPatches *[]callPatch) error {
	// Argument: rdi=task.group handle. Returns 0 on close, 1 for an invalid group.
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.MovEaxImm32(1)
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCloseGroupOff)
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupClosed)
	notClosedAt := e.JnzRel32()
	e.XorEaxEax()
	e.Ret()

	notClosedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notClosedAt, notClosedTo); err != nil {
		return err
	}
	loopStart := len(e.Buf)
	e.MovEaxImm32(1)

	scan := len(e.Buf)
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	doneAt := e.JaeRel32()
	e.PushRax()

	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()

	e.PushRdi()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.MovRdiR15()
	e.MovEdxFromRdiDisp(schedCloseGroupOff)
	e.CmpEaxEdx()
	e.PopRdi()
	notGroupAt := e.JnzRel32()
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneActorAt := e.JzRel32()

	e.PopRax()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backToLoopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backToLoopAt, loopStart); err != nil {
		return err
	}

	continueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notGroupAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneActorAt, continueTo); err != nil {
		return err
	}
	e.PopRax()
	e.AddEaxImm32(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scan); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCloseGroupOff)
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	keepCanceledAt := e.JzRel32()
	e.MovMem32RdiDispImm32(0, taskGroupClosed)
	keepCanceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepCanceledAt, keepCanceledTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitTaskSpawnGroupI32(e *x64.Emitter, actorSpawn string, callPatches *[]callPatch) error {
	// Arguments: rdi=task.group handle, rsi=entryID.
	// Returns task.i32 layout: rax=actor handle, rdx=error status.
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.PushRdi()
	e.PushRsi()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupOpen)
	openAt := e.JzRel32()
	e.PopRsi()
	e.PopRdi()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	openTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, openAt, openTo); err != nil {
		return err
	}
	e.PopRsi()
	e.PopRdi()
	e.PushRdi()
	e.MovEdxEdi()
	setPendingSpawnGroupFromEdx(e)
	e.MovRdiRsi()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: actorSpawn})

	e.CmpEaxImm32(-1)
	spawnedAt := e.JnzRel32()
	e.PopRdx()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	spawnedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, spawnedAt, spawnedTo); err != nil {
		return err
	}
	e.PopRdx()
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.PushRax()
	e.MovRcxRax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
	storeActorSavedGroupForActorPtrInRdiGroupInRdx(e)
	e.PopRax()
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}

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

func emitDispatch(e *x64.Emitter, entries []string, callPatches *[]callPatch) error {
	if len(entries) == 0 {
		return fmt.Errorf("missing dispatch entries")
	}
	type patch struct {
		at int
		to int
	}
	var patches []patch

	for _, name := range entries {
		id := int32(fnv1a32(name))
		e.CmpEdiImm32(id)
		jnzAt := e.JnzRel32()
		e.SubRspImm32(8)
		callAt := e.CallRel32()
		*callPatches = append(*callPatches, callPatch{at: callAt, name: name})
		e.AddRspImm32(8)
		e.Ret()
		patches = append(patches, patch{at: jnzAt, to: len(e.Buf)})
	}

	defStart := len(e.Buf)
	e.MovEaxImm32(1)
	e.Ret()

	for i := range patches {
		target := patches[i].to
		if i == len(patches)-1 {
			target = defStart
		}
		if err := x64.PatchRel32(e.Buf, patches[i].at, target); err != nil {
			return err
		}
	}
	return nil
}

func emitSpawn(e *x64.Emitter, sysMmap uint32, mapFlags uint32, callPatches *[]callPatch, leaPatches *[]leaPatch) error {
	// Argument: entryID in edi.
	// Returns: actor handle in eax.
	e.MovEdxEdi() // entryID -> edx

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCapacityOff)
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	notFullAt := e.JnzRel32()
	// full -> return -1
	e.MovEaxImm32(uint32(^uint32(0)))
	e.Ret()
	notFullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notFullAt, notFullTo); err != nil {
		return err
	}

	// newIdx = count (ecx)
	e.MovEaxEcx()
	e.PushRax() // save newIdx
	e.PushRdx() // save entryID

	// sched.count = count+1
	e.AddEaxImm32(1)
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCountOff)

	// actorPtr = sched.actorsPtr + (newIdx << shift)
	e.MovEaxEcx()
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax() // actorPtr

	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)

	// actor.entryID = entryID (saved)
	e.PopRdx()
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorEntryIDOff)

	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorLastSenderOff, 0)
	e.MovMem32RdiDispImm32(actorExitCodeOff, 0)
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	e.MovMem32RdiDispImm32(actorTaskGroupOff, 0)

	// Save actorPtr across stack init.
	e.MovRaxRdi()
	e.PushRax()

	// Stack init (initRsp -> rcx).
	if err := emitInitActorStack(e, sysMmap, mapFlags, leaPatches); err != nil {
		return err
	}

	// Restore actorPtr.
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorRspOff)

	e.PushRdi()
	e.MovRdiR15()
	e.MovEdxFromRdiDisp(schedSpawnGroupOff)
	e.TestEdxEdx()
	haveSpawnGroupAt := e.JnzRel32()
	e.MovEdxR14d()
	haveSpawnGroupTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveSpawnGroupAt, haveSpawnGroupTo); err != nil {
		return err
	}
	e.MovMem32RdiDispImm32(schedSpawnGroupOff, 0)
	e.PopRdi()
	storeActorSavedGroupForActorPtrInRdiGroupInRdx(e)

	// return newIdx
	e.PopRax()
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.Ret()
	return nil
}

func emitSend(e *x64.Emitter) error {
	// Args: rdi=to (actor handle), rsi=value (i32)
	// Returns: eax=value.

	e.MovEcxEdi() // save receiver idx in ecx

	// msgPtr = bump; bump += msgSize
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedMsgBumpOff)
	e.MovRdxRax() // msgPtr in rdx
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(schedMsgBumpOff)

	// msg.next = 0
	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(msgNextOff)

	// msg.sender = sched.currentIdx
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(msgSenderOff)

	// msg.value = esi
	e.MovMem32RdiDispEsi(msgValueOff)
	// msg.tag = 0 (legacy i32 channel)
	e.MovMem32RdiDispImm32(msgTagOff, 0)
	e.MovMem32RdiDispImm32(msgCountOff, 1)
	e.MovMem32RdiDispEsi(msgPayload0)

	// actorPtr = sched.actorsPtr + (to<<shift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.AddRaxRbx()
	e.PushRax() // save actorPtr
	e.MovRdiRax()

	// tail = actor.mailboxTail
	e.MovRaxFromRdiDisp(actorMailboxTailOff)
	e.TestRaxRax()
	emptyAt := e.JzRel32()

	// non-empty: tail.next = msgPtr
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(msgNextOff)
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	afterAppendAt := e.JmpRel32()

	// empty: head=tail=msgPtr
	emptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)

	afterAppendTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterAppendAt, afterAppendTo); err != nil {
		return err
	}

	// If receiver blocked -> ready
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}

	e.MovEaxEsi()
	e.Ret()
	return nil
}

func emitSendMsg(e *x64.Emitter) error {
	// Args: rdi=to (actor handle), rsi=value (i32), rdx=tag (i32)
	// Returns: eax=value.

	e.MovEcxEdi() // save receiver idx in ecx
	e.PushRdx()   // preserve tag across scheduler/actor pointer loads

	// msgPtr = bump; bump += msgSize
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedMsgBumpOff)
	e.MovRdxRax() // msgPtr in rdx
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(schedMsgBumpOff)

	// msg.next = 0
	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(msgNextOff)

	// msg.sender = sched.currentIdx
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(msgSenderOff)

	// msg.value = esi; msg.tag = preserved stack value
	e.MovMem32RdiDispEsi(msgValueOff)
	e.MovMem32RdiDispImm32(msgCountOff, 1)
	e.MovMem32RdiDispEsi(msgPayload0)
	e.PopRax()
	e.MovMem32RdiDispEax(msgTagOff)

	// actorPtr = sched.actorsPtr + (to<<shift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.AddRaxRbx()
	e.PushRax() // save actorPtr
	e.MovRdiRax()

	// tail = actor.mailboxTail
	e.MovRaxFromRdiDisp(actorMailboxTailOff)
	e.TestRaxRax()
	emptyAt := e.JzRel32()

	// non-empty: tail.next = msgPtr
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(msgNextOff)
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	afterAppendAt := e.JmpRel32()

	// empty: head=tail=msgPtr
	emptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)

	afterAppendTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterAppendAt, afterAppendTo); err != nil {
		return err
	}

	// If receiver blocked -> ready
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}

	e.MovEaxEsi()
	e.Ret()
	return nil
}

func emitSendBegin(e *x64.Emitter) error {
	// Args: rdi=to, rsi=tag, rdx=payload slot count.
	e.MovEcxEdi()
	e.PushRsi()
	e.PushRdx()

	// msgPtr = bump; bump += fixed typed-message node size.
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedMsgBumpOff)
	e.MovRdxRax()
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(schedMsgBumpOff)

	// msg.next = 0
	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(msgNextOff)

	// msg.sender = sched.currentIdx
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(msgSenderOff)

	// msg.tag and msg.count
	e.PopRax()
	e.MovMem32RdiDispEax(msgCountOff)
	e.PopRax()
	e.MovMem32RdiDispEax(msgTagOff)
	e.MovMem32RdiDispImm32(msgValueOff, 0)

	// sched.pendingMsg = msgPtr for send_slot calls.
	e.MovRdiR15()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(schedPendingMsgOff)

	// actorPtr = sched.actorsPtr + (to<<shift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.AddRaxRbx()
	e.PushRax()
	e.MovRdiRax()

	// tail = actor.mailboxTail
	e.MovRaxFromRdiDisp(actorMailboxTailOff)
	e.TestRaxRax()
	emptyAt := e.JzRel32()

	// non-empty: tail.next = msgPtr
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(msgNextOff)
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	afterAppendAt := e.JmpRel32()

	emptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)

	afterAppendTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterAppendAt, afterAppendTo); err != nil {
		return err
	}

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}

	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSendSlot(e *x64.Emitter) error {
	// Args: rdi=index, rsi=value.
	e.MovRaxRdi()
	e.ShlRaxImm8(2)
	e.AddRaxImm32(msgPayload0)
	e.MovRdiR15()
	e.MovRdxRax()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.AddRaxRdx()
	e.MovRdiRax()
	e.MovMem32RdiDispEsi(0)
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSendCommit(e *x64.Emitter) error {
	e.XorEaxEax()
	e.Ret()
	return nil
}

func clearCurrentActorWakeAt(e *x64.Emitter) {
	e.PushRdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.XorEaxEax()
	e.MovMem32RdiDispEax(0)
	e.PopRdi()
}

func setCurrentActorWakeAtFromR13(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorWakeAtPtrFromEaxToRdi(e)
	e.MovEaxR13d()
	e.MovMem32RdiDispEax(0)
}

func emitRecv(e *x64.Emitter, callPatches *[]callPatch) error {
	loopStart := len(e.Buf)
	actorPtrInRax(e)
	e.MovRdxRax() // actorPtr in rdx

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff) // nodePtr in rax
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	// Empty: block and yield.
	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	jmpAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpAt, loopStart); err != nil {
		return err
	}

	// haveMsg:
	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}

	// Preserve nodePtr.
	e.PushRax()

	// next = node.next
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(msgNextOff)
	e.MovRcxRax() // next in rcx

	// actor.mailboxHead = next
	e.MovRdiRdx()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.TestRaxRax()
	skipClearAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	skipClearTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipClearAt, skipClearTo); err != nil {
		return err
	}

	// nodePtr back -> rcx
	e.PopRax()
	e.MovRcxRax()

	// sender = node.sender
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	// value = node.value
	e.MovRdiRcx()
	e.MovEaxFromRdiDisp(msgValueOff)
	e.Ret()
	return nil
}

func emitRecvMsg(e *x64.Emitter, callPatches *[]callPatch) error {
	loopStart := len(e.Buf)
	actorPtrInRax(e)
	e.MovRdxRax() // actorPtr in rdx

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff) // nodePtr in rax
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	// Empty: block and yield.
	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	jmpAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpAt, loopStart); err != nil {
		return err
	}

	// haveMsg:
	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}

	// Preserve nodePtr.
	e.PushRax()

	// next = node.next
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(msgNextOff)
	e.MovRcxRax() // next in rcx

	// actor.mailboxHead = next
	e.MovRdiRdx()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.TestRaxRax()
	skipClearAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	skipClearTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipClearAt, skipClearTo); err != nil {
		return err
	}

	// nodePtr back -> rcx
	e.PopRax()
	e.MovRcxRax()

	// sender = node.sender
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	// value/tag
	e.MovRdiRcx()
	e.MovEaxFromRdiDisp(msgTagOff)
	e.PushRax()
	e.MovEaxFromRdiDisp(msgValueOff)
	e.PopRdx()
	e.Ret()
	return nil
}

func emitRecvPoll(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(2)
	e.Ret()

	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_recv"})
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}

func emitRecvUntil(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.MovR13Rcx()

	loopStart := len(e.Buf)
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

	setCurrentActorWakeAtFromR13(e)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	jmpAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpAt, loopStart); err != nil {
		return err
	}

	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_recv"})
	e.MovEdxImm32(0)
	e.Ret()

	timeoutTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, timeoutAt, timeoutTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.MovEdxImm32(2)
	e.Ret()
	return nil
}

func emitRecvMsgUntil(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	if err := clampEdiNonNegativeIntoEcx(e); err != nil {
		return err
	}
	e.MovR13Rcx()

	loopStart := len(e.Buf)
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

	setCurrentActorWakeAtFromR13(e)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	jmpAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpAt, loopStart); err != nil {
		return err
	}

	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_recv_msg"})
	e.MovR8dImm32(0)
	e.Ret()

	timeoutTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, timeoutAt, timeoutTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.MovEdxImm32(0)
	e.MovR8dImm32(2)
	e.Ret()
	return nil
}

func emitRecvBegin(e *x64.Emitter, callPatches *[]callPatch) error {
	loopStart := len(e.Buf)
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	jmpAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, jmpAt, loopStart); err != nil {
		return err
	}

	haveMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveMsgAt, haveMsgTo); err != nil {
		return err
	}

	e.PushRax()
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(msgNextOff)
	e.MovRcxRax()

	e.MovRdiRdx()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.TestRaxRax()
	skipClearAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	skipClearTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipClearAt, skipClearTo); err != nil {
		return err
	}

	e.PopRax()
	e.MovRcxRax()

	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	e.MovRdiR15()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(schedPendingMsgOff)

	e.MovRdiRcx()
	e.MovEaxFromRdiDisp(msgTagOff)
	e.Ret()
	return nil
}

func emitRecvSlot(e *x64.Emitter) error {
	// Args: rdi=index.
	e.MovRaxRdi()
	e.ShlRaxImm8(2)
	e.AddRaxImm32(msgPayload0)
	e.MovRdiR15()
	e.MovRdxRax()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.AddRaxRdx()
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(0)
	e.Ret()
	return nil
}

func emitRecvCount(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgCountOff)
	e.Ret()
	return nil
}

func emitSelf(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.Ret()
	return nil
}

func emitSender(e *x64.Emitter) error {
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorLastSenderOff)
	e.Ret()
	return nil
}

func emitActorStateLoad(e *x64.Emitter) error {
	// Args: rdi=slot
	// Returns: eax=value (or 0 when slot is out of bounds)
	e.MovEaxEdi()
	e.CmpEaxImm32(maxActorStateSlots)
	outOfRangeAt := e.JaeRel32()
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.AddRaxImm32(actorStateSlot0Off)
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

func emitActorStateStore(e *x64.Emitter) error {
	// Args: rdi=slot, rsi=value
	// Returns: eax=value (or 0 when slot is out of bounds)
	e.MovEaxEdi()
	e.CmpEaxImm32(maxActorStateSlots)
	outOfRangeAt := e.JaeRel32()
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.AddRaxImm32(actorStateSlot0Off)
	e.ShlRdxImm8(2)
	e.AddRaxRdx()
	e.MovMem32RaxPtrEsi()
	e.MovEaxEsi()
	e.Ret()

	outOfRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, outOfRangeAt, outOfRangeTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func actorPtrInRax(e *x64.Emitter) {
	// rax = sched.actorsPtr + (sched.currentIdx << actorSizeShift)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
}

func actorWakeAtPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedActorWakeAt0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func actorWaitTargetPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedActorWait0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func emitInitActorStack(e *x64.Emitter, sysMmap uint32, mapFlags uint32, leaPatches *[]leaPatch) error {
	// Stack mapping.
	emitMmapAnon(e, stackSize, sysMmap, mapFlags)
	// initRsp = base + stackSize - 56
	e.AddRaxImm32(stackSize)
	e.AddRaxImm32(-56)
	e.MovRcxRax() // initRsp in rcx

	// Fill saved regs + return address.
	e.MovRdiRcx()
	// saved r15
	e.MovRaxR15()
	e.MovMem64RdiDispRax(0)
	// saved r14..rbx = 0
	e.XorEaxEax()
	e.MovMem64RdiDispRax(8)
	e.MovMem64RdiDispRax(16)
	e.MovMem64RdiDispRax(24)
	e.MovMem64RdiDispRax(32)
	e.MovMem64RdiDispRax(40)

	// return address = __tetra_actor_trampoline
	if leaPatches == nil {
		return fmt.Errorf("missing leaPatches")
	}
	leaAt := e.LeaRaxRipDisp()
	*leaPatches = append(*leaPatches, leaPatch{at: leaAt, name: "__tetra_actor_trampoline"})
	e.MovMem64RdiDispRax(48)
	return nil
}
