package actorsrt

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
)

func emitVirtualAllocAnon(e *x64.Emitter, length int32, importPatches *[]importPatch) error {
	// VirtualAlloc(NULL, length, MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE)
	//
	// Win64 ABI requires 32 bytes of shadow space, and 16-byte alignment at the call site.
	if importPatches == nil {
		return fmt.Errorf("missing importPatches")
	}
	e.SubRspImm32(40)
	e.MovEcxImm32(0)
	e.MovEdxImm32(uint32(length))
	e.MovR8dImm32(0x3000)
	e.MovR9dImm32(0x04)
	at := e.CallRipDisp32()
	*importPatches = append(*importPatches, importPatch{at: at, name: winImportVirtualAlloc})
	e.AddRspImm32(40)
	return nil
}

func emitInitActorStackWindowsX64(e *x64.Emitter, leaPatches *[]leaPatch, importPatches *[]importPatch) error {
	if leaPatches == nil {
		return fmt.Errorf("missing leaPatches")
	}
	if err := emitVirtualAllocAnon(e, stackSize, importPatches); err != nil {
		return err
	}

	// initRsp = base + stackSize - 80
	// (8 saved regs + return address + 8 bytes so trampoline entry rsp is 16n+8)
	e.AddRaxImm32(stackSize)
	e.AddRaxImm32(-80)
	e.MovRcxRax() // initRsp in rcx

	// Fill saved regs + return address.
	e.MovRdiRcx()
	// saved r15
	e.MovRaxR15()
	e.MovMem64RdiDispRax(0)
	// saved r14..r12 = 0
	e.XorEaxEax()
	e.MovMem64RdiDispRax(8)
	e.MovMem64RdiDispRax(16)
	e.MovMem64RdiDispRax(24)
	// saved rsi/rdi/rbp/rbx = 0
	e.MovMem64RdiDispRax(32)
	e.MovMem64RdiDispRax(40)
	e.MovMem64RdiDispRax(48)
	e.MovMem64RdiDispRax(56)

	// return address = __tetra_actor_trampoline
	leaAt := e.LeaRaxRipDisp()
	*leaPatches = append(*leaPatches, leaPatch{at: leaAt, name: "__tetra_actor_trampoline"})
	e.MovMem64RdiDispRax(64)
	return nil
}

func emitEntryWindowsX64(e *x64.Emitter, mainSymbol string, callPatches *[]callPatch, leaPatches *[]leaPatch, importPatches *[]importPatch) error {
	// Allocate scheduler + actor slots.
	if err := emitVirtualAllocAnon(e, schedAllocSize, importPatches); err != nil {
		return err
	}
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
	if err := emitVirtualAllocAnon(e, msgPoolSize, importPatches); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgBaseOff)
	e.MovMem64RdiDispRax(schedMsgBumpOff)
	e.AddRaxImm32(msgPoolSize)
	e.MovMem64RdiDispRax(schedMsgEndOff)

	// actor0 = sched.actorsPtr + 0
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRdiRax()

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
	if err := emitInitActorStackWindowsX64(e, leaPatches, importPatches); err != nil {
		return err
	}
	// Store actor0.rsp = initRsp
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRdiRax()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorRspOff)

	// Switch to actor0 to start execution.
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRsiRax()
	e.MovRdiR15()
	e.AddRdiImm32(schedRspOff)
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

func emitSwitchToWindowsX64(e *x64.Emitter) error {
	// Signature:
	//   __tetra_switch_to(fromRspPtr: ptr, toRspPtr: ptr)
	//
	// Windows x64: preserve all non-volatile regs we might use (includes rdi/rsi).
	e.PushRbx()
	e.PushRbp()
	e.PushRdi()
	e.PushRsi()
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
	e.PopRsi()
	e.PopRdi()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return nil
}

func emitActorYieldWindowsX64(e *x64.Emitter, callPatches *[]callPatch) error {
	return emitActorYield(e, callPatches)
}

func emitActorExitWindowsX64(e *x64.Emitter, callPatches *[]callPatch) error {
	return emitActorExit(e, callPatches)
}

func emitActorTrampolineWindowsX64(e *x64.Emitter, callPatches *[]callPatch) error {
	// entryID := currentActor.entryID
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorEntryIDOff)

	// Call external __tetra_actor_dispatch under Win64 ABI.
	e.MovEcxEax()
	e.SubRspImm32(40)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_dispatch"})
	e.AddRspImm32(40)

	// exit(code)
	e.MovEdiEax()
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_exit"})
	e.MovEaxImm32(0)
	e.Ret()
	return nil
}

func emitDispatchWindowsX64(e *x64.Emitter, entries []string, callPatches *[]callPatch) error {
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

		// Call external entry function under Win64 ABI:
		// - reserve 32 bytes shadow space
		// - align stack at call site
		e.SubRspImm32(40)
		callAt := e.CallRel32()
		*callPatches = append(*callPatches, callPatch{at: callAt, name: name})
		e.AddRspImm32(40)
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

func emitSpawnWindowsX64(e *x64.Emitter, callPatches *[]callPatch, leaPatches *[]leaPatch, importPatches *[]importPatch) error {
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
	// sched.count = count+1
	e.MovEaxEcx()
	e.AddEaxImm32(1)
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCountOff)

	// actorPtr = sched.actorsPtr + (newIdx << shift)
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax() // actorPtr

	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)

	// actor.entryID = entryID (edx)
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorEntryIDOff)

	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorLastSenderOff, 0)
	e.MovMem32RdiDispImm32(actorExitCodeOff, 0)
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	e.MovMem32RdiDispImm32(actorTaskGroupOff, 0)

	e.MovRaxRdi()
	e.PushRax()

	// Stack init (initRsp -> rcx).
	if err := emitInitActorStackWindowsX64(e, leaPatches, importPatches); err != nil {
		return err
	}

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

	// return newIdx (= sched.count - 1)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCountOff)
	e.AddEaxImm32(-1)
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.Ret()
	return nil
}

func emitActorSpawnWrapperWindowsX64(e *x64.Emitter, jmpPatches *[]callPatch) error {
	// Win64: entryID in rcx -> internal: entryID in edi.
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	e.MovRdiRcx()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: "__tetra_actor_spawn_impl"})
	return nil
}

func emitActorSendWrapperWindowsX64(e *x64.Emitter, jmpPatches *[]callPatch) error {
	// Win64: rcx=to, rdx=value -> internal: rdi=to, rsi=value.
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: "__tetra_actor_send_impl"})
	return nil
}

func emitActorSendMsgWrapperWindowsX64(e *x64.Emitter, jmpPatches *[]callPatch) error {
	// Win64: rcx=to, rdx=value, r8=tag -> internal: rdi=to, rsi=value, rdx=tag.
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	e.MovRdxR8()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: "__tetra_actor_send_msg_impl"})
	return nil
}

func emitActorSendBeginWrapperWindowsX64(e *x64.Emitter, jmpPatches *[]callPatch) error {
	// Win64: rcx=to, rdx=tag, r8=count -> internal: rdi=to, rsi=tag, rdx=count.
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	e.MovRdxR8()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: "__tetra_actor_send_begin_impl"})
	return nil
}

func emitActorSendSlotWrapperWindowsX64(e *x64.Emitter, jmpPatches *[]callPatch) error {
	// Win64: rcx=index, rdx=value -> internal: rdi=index, rsi=value.
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: "__tetra_actor_send_slot_impl"})
	return nil
}

func emitActorOneArgWrapperWindowsX64(e *x64.Emitter, target string, jmpPatches *[]callPatch) error {
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	if target == "" {
		return fmt.Errorf("missing wrapper target")
	}
	e.MovRdiRcx()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: target})
	return nil
}

func emitTaskTwoArgWrapperWindowsX64(e *x64.Emitter, target string, jmpPatches *[]callPatch) error {
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	if target == "" {
		return fmt.Errorf("missing wrapper target")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: target})
	return nil
}

func emitTaskThreeArgWrapperWindowsX64(e *x64.Emitter, target string, jmpPatches *[]callPatch) error {
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	if target == "" {
		return fmt.Errorf("missing wrapper target")
	}
	e.MovRdiRcx()
	e.MovRaxRdx()
	e.MovRsiRax()
	e.MovRdxR8()
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: target})
	return nil
}

func emitTaskJoinTypedWrapperWindowsX64(e *x64.Emitter, slots int, target string, jmpPatches *[]callPatch) error {
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	if target == "" {
		return fmt.Errorf("missing wrapper target")
	}
	switch slots {
	case 2:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
	case 3:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
	case 4:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
		e.MovRcxR9()
	case 5:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
		e.MovRcxR9()
		e.MovR8FromRspDisp(40)
	case 6:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
		e.MovRcxR9()
		e.MovR8FromRspDisp(40)
		e.MovR9FromRspDisp(48)
	case 7:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
		e.MovRcxR9()
		e.MovR8FromRspDisp(40)
		e.MovR9FromRspDisp(48)
		e.MovRaxFromRspDisp(56)
		e.MovMem64RspDispRax(8)
	case 8:
		e.MovRdiRcx()
		e.MovRaxRdx()
		e.MovRsiRax()
		e.MovRdxR8()
		e.MovRcxR9()
		e.MovR8FromRspDisp(40)
		e.MovR9FromRspDisp(48)
		e.MovRaxFromRspDisp(56)
		e.MovMem64RspDispRax(8)
		e.MovRaxFromRspDisp(64)
		e.MovMem64RspDispRax(16)
	default:
		return fmt.Errorf("unsupported typed task join wrapper slots %d", slots)
	}
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: target})
	return nil
}

func emitActorNoArgWrapperWindowsX64(e *x64.Emitter, target string, jmpPatches *[]callPatch) error {
	if jmpPatches == nil {
		return fmt.Errorf("missing jmpPatches")
	}
	if target == "" {
		return fmt.Errorf("missing wrapper target")
	}
	at := e.JmpRel32()
	*jmpPatches = append(*jmpPatches, callPatch{at: at, name: target})
	return nil
}
