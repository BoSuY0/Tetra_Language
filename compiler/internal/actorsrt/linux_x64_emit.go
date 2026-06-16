package actorsrt

import (
	"encoding/binary"
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
	msgPayload0  = 24 // u64[8]
	msgSize      = 88
	msgSlotSize  = 8
)

const (
	linuxSysRead         = 0
	linuxSysWrite        = 1
	linuxSysClose        = 3
	linuxSysFcntl        = 72
	linuxSysPoll         = 7
	linuxSysLseek        = 8
	linuxSysSocket       = 41
	linuxSysConnect      = 42
	linuxSysSendto       = 44
	linuxSysRecvfrom     = 45
	linuxSysBind         = 49
	linuxSysListen       = 50
	linuxSysSetSockOpt   = 54
	linuxSysEpollCtl     = 233
	linuxSysEpollWait    = 232
	linuxSysAccept4      = 288
	linuxSysEpollCreate1 = 291
	linuxSysMemfdCreate  = 319

	actorWireMagic          = 0x52444154
	actorWireVersion        = 1
	actorWireFrameSize      = 60
	actorWireFrameHello     = 1
	actorWireFrameHelloAck  = 2
	actorWireFrameSpawn     = 3
	actorWireFrameSendI32   = 5
	actorWireFrameSendMsg   = 6
	actorWireFrameSendTyped = 7
	actorWireFrameNodeDown  = 8
	actorWireFrameError     = 9
	actorWireStatusDown     = 1
	actorWireOffsetMagic    = 0
	actorWireOffsetVer      = 4
	actorWireOffsetType     = 6
	actorWireOffsetSrc      = 8
	actorWireOffsetDest     = 10
	actorWireOffsetSeq      = 12
	actorWireOffsetActor    = 16
	actorWireOffsetSlots    = 18
	actorWireOffsetTag      = 20
	actorWireOffsetStatus   = 24
	actorWireOffsetValue    = 28
)

func fnv1a32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func emitMovMem32RspDispImm32(e *x64.Emitter, disp byte, val uint32) {
	e.Emit(0xC7, 0x44, 0x24, disp)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	e.Emit(buf[:]...)
}

func emitMovMem16RspDispImm16(e *x64.Emitter, disp byte, val uint16) {
	e.Emit(0x66, 0xC7, 0x44, 0x24, disp)
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], val)
	e.Emit(buf[:]...)
}

func emitMovMem16RspDispAx(e *x64.Emitter, disp byte) {
	e.Emit(0x66, 0x89, 0x44, 0x24, disp)
}

func emitMovMem32RspDispEax(e *x64.Emitter, disp byte) {
	e.Emit(0x89, 0x44, 0x24, disp)
}

func emitCheckedMessagePoolAlloc(e *x64.Emitter) int {
	// On success, rdx is the allocated message pointer. Reclaimed nodes are
	// reused before advancing sched.msg_bump. The returned jump targets a
	// caller-specific stack unwind path when no reclaimed or bump node exists.
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedMsgFreeOff)
	e.TestRaxRax()
	bumpAt := e.JzRel32()

	e.MovRdxRax()
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(msgNextOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgFreeOff)
	doneAt := e.JmpRel32()

	bumpTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bumpAt, bumpTo); err != nil {
		panic(err)
	}
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedMsgBumpOff)
	e.MovRdxRax()
	e.AddRaxImm32(msgSize)
	e.MovR8FromRdiDisp(schedMsgEndOff)
	e.CmpRaxR8()
	overflowAt := e.JaRel32()
	e.MovMem64RdiDispRax(schedMsgBumpOff)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
	return overflowAt
}

func emitRecycleMessageNodeInRax(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovR8FromRdiDisp(schedMsgFreeOff)
	e.MovRdiRax()
	e.MovMem64RdiDispR8(msgNextOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgFreeOff)
}

func emitRecycleMessageNodeFromRspDisp(e *x64.Emitter, disp int32) {
	e.MovRaxFromRspDisp(disp)
	emitRecycleMessageNodeInRax(e)
}

func emitCopyMessageNodeToRecvScratchFromRspDisp(e *x64.Emitter, disp int32) {
	for off := int32(0); off < msgSize; off += 8 {
		e.MovRaxFromRspDisp(disp)
		e.MovRdiRax()
		e.MovRaxFromRdiDisp(off)
		e.MovRdiR15()
		e.MovMem64RdiDispRax(schedRecvScratchOff + off)
	}

	e.MovRaxR15()
	e.AddRaxImm32(schedRecvScratchOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedPendingMsgOff)
}

func emitMessagePoolExhaustedReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
}

func emitClearPendingMsg(e *x64.Emitter) {
	e.MovRdiR15()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedPendingMsgOff)
}

func emitMailboxFullCheckForReceiverInEcx(e *x64.Emitter) int {
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorMailboxCountOff)
	e.CmpEaxImm32(maxActorMailboxMsgs)
	return e.JaeRel32()
}

func emitMailboxFullReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFE)
	e.Ret()
}

func emitInvalidActorHandleReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFD)
	e.Ret()
}

func emitActorDoneReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFC)
	e.Ret()
}

func emitDoneActorCheckForReceiverInEcx(e *x64.Emitter) int {
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	return e.JzRel32()
}

func emitIncrementMailboxCount(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorMailboxCountOff)
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(actorMailboxCountOff)
}

func emitDecrementMailboxCount(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorMailboxCountOff)
	e.SubEaxImm32(1)
	e.MovMem32RdiDispEax(actorMailboxCountOff)
}

func emitMovzxEaxWordRspDisp(e *x64.Emitter, disp byte) {
	e.Emit(0x0F, 0xB7, 0x44, 0x24, disp)
}

func emitMovEaxRspDisp(e *x64.Emitter, disp byte) {
	e.Emit(0x8B, 0x44, 0x24, disp)
}

func emitLeaRdiRspDisp(e *x64.Emitter, disp byte) {
	if disp == 0 {
		e.Emit(0x48, 0x8D, 0x3C, 0x24)
		return
	}
	e.Emit(0x48, 0x8D, 0x7C, 0x24, disp)
}

func emitLeaRsiRspDisp(e *x64.Emitter, disp byte) {
	if disp == 0 {
		e.Emit(0x48, 0x8D, 0x34, 0x24)
		return
	}
	e.Emit(0x48, 0x8D, 0x74, 0x24, disp)
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
	e.MovMem32RdiDispImm32(schedNetFDOff, 0)
	e.MovMem32RdiDispImm32(schedNodeIDOff, 0)
	e.MovMem32RdiDispImm32(schedNetStatusOff, 1)

	// Message pool
	emitMmapAnon(e, msgPoolSize, sysMmap, mapFlags)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgBaseOff)
	e.MovMem64RdiDispRax(schedMsgBumpOff)
	e.AddRaxImm32(msgPoolSize)
	e.MovMem64RdiDispRax(schedMsgEndOff)
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedMsgFreeOff)

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
	e.MovMem32RdiDispImm32(actorMailboxCountOff, 0)
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
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	e.TestEaxEax()
	noNetworkWorkAt := e.JzRel32()
	networkLoopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, networkLoopAt, loopStart); err != nil {
		return err
	}
	noNetworkWorkTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noNetworkWorkAt, noNetworkWorkTo); err != nil {
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
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	canceledAt := e.JzRel32()

	deadlineCheckTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, deadlineCheckTo); err != nil {
		return nil, nil, err
	}
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
	if err := x64.PatchRel32(e.Buf, canceledAt, dueTo); err != nil {
		return nil, nil, err
	}
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
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	targetCanceledAt := e.JzRel32()

	deadlineCheckTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetWaitingAt, deadlineCheckTo); err != nil {
		return nil, nil, err
	}
	if err := x64.PatchRel32(e.Buf, noGroupAt, deadlineCheckTo); err != nil {
		return nil, nil, err
	}
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
	if err := x64.PatchRel32(e.Buf, targetCanceledAt, doneReadyTo); err != nil {
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

func emitCurrentTaskGroupCanceledCheck(e *x64.Emitter, emitCanceledReturn func()) error {
	e.MovEaxR14d()
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
	return x64.PatchRel32(e.Buf, notCanceledAt, continueTo)
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
