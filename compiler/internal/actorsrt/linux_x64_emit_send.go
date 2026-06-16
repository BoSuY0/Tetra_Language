package actorsrt

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
)

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
	e.MovMem32RdiDispImm32(actorMailboxCountOff, 0)

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

	e.CmpEdiImm32(-1)
	invalidAt := e.JzRel32()
	e.Emit(0xF7, 0xC7, 0x00, 0x00, 0x00, 0x80) // test edi, remote-handle high bit
	localAt := e.JzRel32()
	if err := emitRemoteSendI32(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}

	e.MovEcxEdi() // save receiver idx in ecx
	doneAt := emitDoneActorCheckForReceiverInEcx(e)
	fullAt := emitMailboxFullCheckForReceiverInEcx(e)

	overflowAt := emitCheckedMessagePoolAlloc(e)

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
	emitIncrementMailboxCount(e)

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
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	emitMessagePoolExhaustedReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
		return err
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	emitActorDoneReturn(e)
	return nil
}

func emitRemoteSendI32(e *x64.Emitter) error {
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.Emit(0x48, 0x83, 0xEC, 0x60) // sub rsp, 96
	e.Emit(0x89, 0x7C, 0x24, 0x40) // handle
	e.Emit(0x89, 0x74, 0x24, 0x44) // value

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x48) // fd

	emitActorWireControlFrame(e, 0, actorWireFrameSendI32)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNodeIDOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSrc)
	e.MovEaxFromRspDisp(0x40)
	e.Emit(0xC1, 0xE8, 0x10) // shr eax, 16
	e.Emit(0x83, 0xE0, 0x7F) // and eax, 0x7f
	emitMovMem16RspDispAx(e, actorWireOffsetDest)
	e.MovEaxFromRspDisp(0x40)
	emitMovMem16RspDispAx(e, actorWireOffsetActor)
	emitMovMem16RspDispImm16(e, actorWireOffsetSlots, 1)
	e.MovEaxFromRspDisp(0x44)
	emitMovMem32RspDispEax(e, actorWireOffsetValue)

	e.Emit(0x8B, 0x7C, 0x24, 0x48)
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(0x44)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()
	return nil
}

func emitSendMsg(e *x64.Emitter) error {
	// Args: rdi=to (actor handle), rsi=value (i32), rdx=tag (i32)
	// Returns: eax=value.

	e.CmpEdiImm32(-1)
	invalidAt := e.JzRel32()
	e.Emit(0xF7, 0xC7, 0x00, 0x00, 0x00, 0x80) // test edi, remote-handle high bit
	localAt := e.JzRel32()
	if err := emitRemoteSendMsg(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}

	e.MovEcxEdi() // save receiver idx in ecx
	doneAt := emitDoneActorCheckForReceiverInEcx(e)
	fullAt := emitMailboxFullCheckForReceiverInEcx(e)
	e.PushRdx() // preserve tag across scheduler/actor pointer loads

	overflowAt := emitCheckedMessagePoolAlloc(e)

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
	emitIncrementMailboxCount(e)

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
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	e.PopRax()
	emitMessagePoolExhaustedReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
		return err
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	emitActorDoneReturn(e)
	return nil
}

func emitRemoteSendMsg(e *x64.Emitter) error {
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.Emit(0x48, 0x83, 0xEC, 0x60) // sub rsp, 96
	e.Emit(0x89, 0x7C, 0x24, 0x40) // handle
	e.Emit(0x89, 0x74, 0x24, 0x44) // value
	e.Emit(0x89, 0x54, 0x24, 0x48) // tag

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x4C) // fd

	emitActorWireControlFrame(e, 0, actorWireFrameSendMsg)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNodeIDOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSrc)
	e.MovEaxFromRspDisp(0x40)
	e.Emit(0xC1, 0xE8, 0x10)
	e.Emit(0x83, 0xE0, 0x7F)
	emitMovMem16RspDispAx(e, actorWireOffsetDest)
	e.MovEaxFromRspDisp(0x40)
	emitMovMem16RspDispAx(e, actorWireOffsetActor)
	emitMovMem16RspDispImm16(e, actorWireOffsetSlots, 1)
	e.MovEaxFromRspDisp(0x48)
	emitMovMem32RspDispEax(e, actorWireOffsetTag)
	e.MovEaxFromRspDisp(0x44)
	emitMovMem32RspDispEax(e, actorWireOffsetValue)

	e.Emit(0x8B, 0x7C, 0x24, 0x4C)
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(0x44)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()
	return nil
}

func emitSendBegin(e *x64.Emitter) error {
	// Args: rdi=to, rsi=tag, rdx=payload slot count.
	e.CmpEdiImm32(-1)
	invalidAt := e.JzRel32()
	e.Emit(0xF7, 0xC7, 0x00, 0x00, 0x00, 0x80) // test edi, remote-handle high bit
	localAt := e.JzRel32()
	if err := emitRemoteSendBegin(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}

	e.MovEcxEdi()
	doneAt := emitDoneActorCheckForReceiverInEcx(e)
	fullAt := emitMailboxFullCheckForReceiverInEcx(e)
	e.PushRsi()
	e.PushRdx()

	overflowAt := emitCheckedMessagePoolAlloc(e)

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
	emitIncrementMailboxCount(e)

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
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	e.PopRax()
	e.PopRax()
	emitClearPendingMsg(e)
	emitMessagePoolExhaustedReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
		return err
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	emitActorDoneReturn(e)
	return nil
}

func emitRemoteSendBegin(e *x64.Emitter) error {
	// Args: rdi=remote actor handle, rsi=enum tag, rdx=payload slot count.
	// The pending message is not enqueued locally; send_commit serializes it as
	// actorwire FrameSendTyped.
	e.PushRdi()
	e.PushRsi()
	e.PushRdx()

	overflowAt := emitCheckedMessagePoolAlloc(e)

	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(msgNextOff)

	e.PopRax()
	e.MovMem32RdiDispEax(msgCountOff)
	e.PopRax()
	e.MovMem32RdiDispEax(msgTagOff)
	e.PopRax()
	e.MovMem32RdiDispEax(msgSenderOff)
	e.MovMem32RdiDispImm32(msgValueOff, 0)

	e.MovRdiR15()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(schedPendingMsgOff)
	e.XorEaxEax()
	e.Ret()
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	e.PopRax()
	e.PopRax()
	e.PopRax()
	emitClearPendingMsg(e)
	emitMessagePoolExhaustedReturn(e)
	return nil
}

func emitSendSlot(e *x64.Emitter) error {
	// Args: rdi=index, rsi=value.
	e.MovRaxRdi()
	e.ShlRaxImm8(3)
	e.AddRaxImm32(msgPayload0)
	e.MovRdiR15()
	e.MovRdxRax()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.TestRaxRax()
	havePendingAt := e.JnzRel32()
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	havePendingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, havePendingAt, havePendingTo); err != nil {
		return err
	}
	e.AddRaxRdx()
	e.MovRdiRax()
	e.MovRaxRsi()
	e.MovMem64RdiDispRax(0)
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSendCommit(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.TestRaxRax()
	havePendingAt := e.JnzRel32()
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	havePendingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, havePendingAt, havePendingTo); err != nil {
		return err
	}

	e.MovRdxRax()
	e.MovRdiRdx()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.Emit(0xA9, 0x00, 0x00, 0x00, 0x80) // test eax, remote-handle high bit
	remoteAt := e.JnzRel32()
	emitClearPendingMsg(e)
	e.Ret()
	remoteTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, remoteAt, remoteTo); err != nil {
		return err
	}
	return emitRemoteSendCommit(e)
}

func emitRemoteSendCommit(e *x64.Emitter) error {
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.Emit(0x48, 0x83, 0xEC, 0x60)       // sub rsp, 96
	e.Emit(0x48, 0x89, 0x54, 0x24, 0x40) // msg ptr
	emitMovMem32RspDispEax(e, 0x48)      // target handle

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x4C) // fd

	emitActorWireControlFrame(e, 0, actorWireFrameSendTyped)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNodeIDOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSrc)
	e.MovEaxFromRspDisp(0x48)
	e.Emit(0xC1, 0xE8, 0x10)
	e.Emit(0x83, 0xE0, 0x7F)
	emitMovMem16RspDispAx(e, actorWireOffsetDest)
	e.MovEaxFromRspDisp(0x48)
	emitMovMem16RspDispAx(e, actorWireOffsetActor)

	e.Emit(0x48, 0x8B, 0x7C, 0x24, 0x40) // msg ptr
	e.MovEaxFromRdiDisp(msgCountOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSlots)
	e.MovEaxFromRdiDisp(msgTagOff)
	emitMovMem32RspDispEax(e, actorWireOffsetTag)
	for slot := 0; slot < 8; slot++ {
		e.MovEaxFromRdiDisp(msgPayload0 + int32(slot*msgSlotSize))
		emitMovMem32RspDispEax(e, byte(actorWireOffsetValue+slot*4))
	}

	e.Emit(0x8B, 0x7C, 0x24, 0x4C)
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	emitRecycleMessageNodeFromRspDisp(e, 0x40)
	e.MovRdiR15()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedPendingMsgOff)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	emitRecycleMessageNodeFromRspDisp(e, 0x40)
	e.MovRdiR15()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedPendingMsgOff)
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()
	return nil
}

func emitActorNetPump(e *x64.Emitter) error {
	var retJumps []int
	var netFailureJumps []int
	var successJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(128)

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	retJumps = append(retJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	emitMovMem32RspDispEax(e, 0x40)      // pollfd.fd
	emitMovMem16RspDispImm16(e, 0x44, 1) // POLLIN
	emitMovMem16RspDispImm16(e, 0x46, 0) // revents
	emitLeaRdiRspDisp(e, 0x40)           // pollfd*
	e.Emit(0xBE, 0x01, 0x00, 0x00, 0x00) // nfds = 1
	e.Emit(0x31, 0xD2)                   // timeout = 0
	e.MovEaxImm32(linuxSysPoll)
	e.Syscall()
	e.TestEaxEax()
	pollNonNegativeAt := e.JgeRel32()
	retJumps = append(retJumps, e.JmpRel32())
	pollNonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, pollNonNegativeAt, pollNonNegativeTo); err != nil {
		return err
	}
	e.TestEaxEax()
	pollReadyAt := e.JnzRel32()
	retJumps = append(retJumps, e.JmpRel32())
	pollReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, pollReadyAt, pollReadyTo); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.MovEdiEax()
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysRead)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	frameReadAt := e.JzRel32()
	retJumps = append(retJumps, e.JmpRel32())
	frameReadTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, frameReadAt, frameReadTo); err != nil {
		return err
	}

	emitMovEaxRspDisp(e, actorWireOffsetMagic)
	e.CmpEaxImm32(actorWireMagic)
	magicOKAt := e.JzRel32()
	retJumps = append(retJumps, e.JmpRel32())
	magicOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, magicOKAt, magicOKTo); err != nil {
		return err
	}
	emitMovzxEaxWordRspDisp(e, actorWireOffsetType)
	e.CmpEaxImm32(actorWireFrameSendI32)
	sendI32At := e.JzRel32()
	e.CmpEaxImm32(actorWireFrameSendMsg)
	sendMsgAt := e.JzRel32()
	e.CmpEaxImm32(actorWireFrameSendTyped)
	sendTypedAt := e.JzRel32()
	e.CmpEaxImm32(actorWireFrameNodeDown)
	nodeDownAt := e.JzRel32()
	e.CmpEaxImm32(actorWireFrameError)
	errorAt := e.JzRel32()
	retJumps = append(retJumps, e.JmpRel32())

	failureStatusTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeDownAt, failureStatusTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, errorAt, failureStatusTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, actorWireOffsetStatus)
	e.TestEaxEax()
	statusNonZeroAt := e.JnzRel32()
	e.MovEaxImm32(actorWireStatusDown)
	statusReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, statusNonZeroAt, statusReadyTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedNetStatusOff)
	successJumps = append(successJumps, e.JmpRel32())

	sendI32To := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sendI32At, sendI32To); err != nil {
		return err
	}
	e.XorEaxEax()
	emitMovMem32RspDispEax(e, 0x4C) // normalized tag
	e.MovEaxImm32(1)
	emitMovMem32RspDispEax(e, 0x50) // normalized payload slot count
	sendI32NormalizedAt := e.JmpRel32()

	sendMsgTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sendMsgAt, sendMsgTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, actorWireOffsetTag)
	emitMovMem32RspDispEax(e, 0x4C)
	e.MovEaxImm32(1)
	emitMovMem32RspDispEax(e, 0x50)
	sendMsgNormalizedAt := e.JmpRel32()

	sendTypedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sendTypedAt, sendTypedTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, actorWireOffsetTag)
	emitMovMem32RspDispEax(e, 0x4C)
	emitMovzxEaxWordRspDisp(e, actorWireOffsetSlots)
	emitMovMem32RspDispEax(e, 0x50)

	normalizedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, sendI32NormalizedAt, normalizedTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, sendMsgNormalizedAt, normalizedTo); err != nil {
		return err
	}

	emitMovzxEaxWordRspDisp(e, actorWireOffsetActor)
	e.CmpEaxImm32(maxActors - 1)
	retJumps = append(retJumps, e.JaRel32())
	emitMovMem32RspDispEax(e, 0x48) // actor id

	overflowAt := emitCheckedMessagePoolAlloc(e)
	netFailureJumps = append(netFailureJumps, overflowAt)

	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(msgNextOff)
	emitMovzxEaxWordRspDisp(e, actorWireOffsetSrc)
	e.Emit(0xC1, 0xE0, 0x10)             // shl eax, 16
	e.Emit(0x0D, 0x00, 0x00, 0x00, 0x80) // or eax, remote handle bit
	e.MovMem32RdiDispEax(msgSenderOff)
	emitMovEaxRspDisp(e, actorWireOffsetValue)
	e.MovMem32RdiDispEax(msgValueOff)
	for slot := 0; slot < 8; slot++ {
		emitMovEaxRspDisp(e, byte(actorWireOffsetValue+slot*4))
		e.MovMem32RdiDispEax(msgPayload0 + int32(slot*msgSlotSize))
	}
	emitMovEaxRspDisp(e, 0x4C)
	e.MovMem32RdiDispEax(msgTagOff)
	emitMovEaxRspDisp(e, 0x50)
	e.MovMem32RdiDispEax(msgCountOff)

	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	emitMovEaxRspDisp(e, 0x48)
	e.Emit(0x48, 0x89, 0xC3) // mov rbx, rax
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.PushRax()
	e.MovRdiRax()

	e.MovRaxFromRdiDisp(actorMailboxTailOff)
	e.TestRaxRax()
	emptyAt := e.JzRel32()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(msgNextOff)
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	appendedAt := e.JmpRel32()

	emptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)

	appendedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, appendedAt, appendedTo); err != nil {
		return err
	}
	emitIncrementMailboxCount(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}
	e.MovRdiRdx()
	emitMovEaxRspDisp(e, 0x4C)
	e.MovMem32RdiDispEax(msgTagOff)
	emitMovEaxRspDisp(e, 0x50)
	e.MovMem32RdiDispEax(msgCountOff)
	successJumps = append(successJumps, e.JmpRel32())

	netFailureTo := len(e.Buf)
	for _, at := range netFailureJumps {
		if err := x64.PatchRel32(e.Buf, at, netFailureTo); err != nil {
			return err
		}
	}
	e.MovRdiR15()
	e.MovMem32RdiDispImm32(schedNetStatusOff, actorWireStatusDown)
	successJumps = append(successJumps, e.JmpRel32())

	retTo := len(e.Buf)
	for _, at := range retJumps {
		if err := x64.PatchRel32(e.Buf, at, retTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()

	successTo := len(e.Buf)
	for _, at := range successJumps {
		if err := x64.PatchRel32(e.Buf, at, successTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitActorNetPumpNoop(e *x64.Emitter) error {
	e.XorEaxEax()
	e.Ret()
	return nil
}
