package actorsrt

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
)

func emitSurfaceOpen(e *x64.Emitter) error {
	// memfd_create("", 0) gives the Surface host an owned kernel handle without
	// exposing toolkit widgets or filesystem sidecars to Tetra code.
	e.Emit(0x6A, 0x00) // push 0; empty C string on stack
	emitMovRdiRsp(e)
	emitXorEsiEsi(e)
	e.MovEaxImm32(linuxSysMemfdCreate)
	e.Syscall()
	e.AddRspImm32(8)
	e.Ret()
	return nil
}

func emitSurfaceClose(e *x64.Emitter) error {
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()
	e.Ret()
	return nil
}

func emitSurfacePresentRGBA(e *x64.Emitter) error {
	// ABI slots already match Linux write(fd, buf, count):
	// rdi=surface fd, rsi=pixels ptr, rdx=pixels len.
	e.PushRsi()
	e.PushRdx()
	emitXorEsiEsi(e)
	e.MovEdxImm32(1)
	e.MovEaxImm32(linuxSysLseek)
	e.Syscall()
	e.PushRax()
	emitXorEsiEsi(e)
	e.MovEdxImm32(0)
	e.MovEaxImm32(linuxSysLseek)
	e.Syscall()
	e.PopR8()
	e.PopRdx()
	e.PopRsi()
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.PushRax()
	e.PushR8()
	e.PopRsi()
	e.MovEdxImm32(0)
	e.MovEaxImm32(linuxSysLseek)
	e.Syscall()
	e.PopRax()
	e.TestRaxRax()
	okAt := e.JgeRel32()
	e.MovEaxImm32(1)
	e.Ret()
	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSurfacePollEventInto(e *x64.Emitter) error {
	e.CmpEdxImm32(9)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}

	e.PushRsi()
	emitXorEsiEsi(e)
	e.MovEdxImm32(1)
	e.MovEaxImm32(linuxSysLseek)
	e.Syscall()
	e.PopRsi()
	e.TestRaxRax()
	cursorOKAt := e.JgeRel32()
	e.XorEaxEax()
	cursorOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, cursorOKAt, cursorOKTo); err != nil {
		return err
	}

	e.PushRsi()
	e.PushRax()
	e.AddRaxImm32(1)
	e.MovRsiRax()
	e.MovEdxImm32(0)
	e.MovEaxImm32(linuxSysLseek)
	e.Syscall()
	e.PopRax()
	e.PopRsi()

	e.CmpRaxImm32(0)
	pointerAt := e.JzRel32()
	e.CmpRaxImm32(1)
	keyAt := e.JzRel32()
	e.CmpRaxImm32(2)
	resizeAt := e.JzRel32()
	e.CmpRaxImm32(3)
	textAt := e.JzRel32()
	e.CmpRaxImm32(4)
	closeAt := e.JzRel32()
	emitSurfaceEventRecord(e, 0, 0, 0, 0, 0, 400, 240, 5, 0)
	e.MovEaxImm32(9)
	e.Ret()

	pointerTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, pointerAt, pointerTo); err != nil {
		return err
	}
	emitSurfaceEventRecord(e, 5, 48, 96, 1, 0, 320, 200, 0, 0)
	e.MovEaxImm32(9)
	e.Ret()

	keyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keyAt, keyTo); err != nil {
		return err
	}
	emitSurfaceEventRecord(e, 6, 0, 0, 0, 32, 320, 200, 1, 0)
	e.MovEaxImm32(9)
	e.Ret()

	resizeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, resizeAt, resizeTo); err != nil {
		return err
	}
	emitSurfaceEventRecord(e, 2, 0, 0, 0, 0, 400, 240, 2, 0)
	e.MovEaxImm32(9)
	e.Ret()

	textTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, textAt, textTo); err != nil {
		return err
	}
	emitSurfaceEventRecord(e, 8, 0, 0, 0, 0, 400, 240, 3, 2)
	e.MovEaxImm32(9)
	e.Ret()

	closeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, closeAt, closeTo); err != nil {
		return err
	}
	emitSurfaceEventRecord(e, 1, 0, 0, 0, 0, 400, 240, 4, 0)
	e.MovEaxImm32(9)
	e.Ret()
	return nil
}

func emitSurfacePollEventTextInto(e *x64.Emitter) error {
	e.CmpEdxImm32(2)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}
	e.Emit(0xC6, 0x06, 'O')       // mov byte ptr [rsi], 'O'
	e.Emit(0xC6, 0x46, 0x01, 'K') // mov byte ptr [rsi+1], 'K'
	e.MovEaxImm32(2)
	e.Ret()
	return nil
}

func emitSurfaceClipboardWriteText(e *x64.Emitter) error {
	e.MovEaxEdx()
	e.Ret()
	return nil
}

func emitSurfaceClipboardReadTextInto(e *x64.Emitter) error {
	e.CmpEdxImm32(3)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}
	e.Emit(0xC6, 0x06, 'T')       // mov byte ptr [rsi], 'T'
	e.Emit(0xC6, 0x46, 0x01, 'e') // mov byte ptr [rsi+1], 'e'
	e.Emit(0xC6, 0x46, 0x02, 't') // mov byte ptr [rsi+2], 't'
	e.MovEaxImm32(3)
	e.Ret()
	return nil
}

func emitSurfacePollCompositionInto(e *x64.Emitter) error {
	e.CmpEdxImm32(4)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}
	emitMovMem32RsiDispImm(e, 0, 1)
	emitMovMem32RsiDispImm(e, 4, 1)
	emitMovMem32RsiDispImm(e, 8, 1)
	emitMovMem32RsiDispImm(e, 12, 1)
	e.MovEaxImm32(4)
	e.Ret()
	return nil
}

func emitMovMem32RsiDispImm(e *x64.Emitter, disp byte, imm uint32) {
	if disp == 0 {
		e.Emit(0xC7, 0x06)
	} else {
		e.Emit(0xC7, 0x46, disp)
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], imm)
	e.Emit(buf[:]...)
}

func emitSurfaceEventRecord(e *x64.Emitter, kind, x, y, button, key, width, height, timestamp, textLen uint32) {
	emitMovMem32RsiDispImm(e, 0, kind)
	emitMovMem32RsiDispImm(e, 4, x)
	emitMovMem32RsiDispImm(e, 8, y)
	emitMovMem32RsiDispImm(e, 12, button)
	emitMovMem32RsiDispImm(e, 16, key)
	emitMovMem32RsiDispImm(e, 20, width)
	emitMovMem32RsiDispImm(e, 24, height)
	emitMovMem32RsiDispImm(e, 28, timestamp)
	emitMovMem32RsiDispImm(e, 32, textLen)
}

func emitSurfaceOK(e *x64.Emitter) error {
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitSurfaceConst(e *x64.Emitter, value uint32) error {
	e.MovEaxImm32(value)
	e.Ret()
	return nil
}

func emitMovRdiRsp(e *x64.Emitter) {
	e.Emit(0x48, 0x89, 0xE7)
}

func emitXorEsiEsi(e *x64.Emitter) {
	e.Emit(0x31, 0xF6)
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
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax() // actorPtr in rdx

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff) // nodePtr in rax
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	// Empty: block and yield.
	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
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

	// Preserve node fields before unlinking the mailbox entry.
	e.PushRax() // nodePtr
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgValueOff)
	e.PushRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.PushRax()

	// next = node.next
	e.MovRaxFromRspDisp(16)
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
	emitDecrementMailboxCount(e)
	emitRecycleMessageNodeFromRspDisp(e, 16)

	e.PopRax()
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	e.PopRax()
	e.AddRspImm32(8)
	e.Ret()
	return nil
}

func emitRecvMsg(e *x64.Emitter, callPatches *[]callPatch) error {
	loopStart := len(e.Buf)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax() // actorPtr in rdx

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff) // nodePtr in rax
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	// Empty: block and yield.
	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
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

	// Preserve node fields before unlinking the mailbox entry.
	e.PushRax() // nodePtr
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(msgTagOff)
	e.PushRax()
	e.MovEaxFromRdiDisp(msgValueOff)
	e.PushRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.PushRax()

	// next = node.next
	e.MovRaxFromRspDisp(24)
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
	emitDecrementMailboxCount(e)
	emitRecycleMessageNodeFromRspDisp(e, 24)

	e.PopRax()
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	// value/tag
	e.PopRax()
	e.PopRdx()
	e.AddRspImm32(8)
	e.Ret()
	return nil
}

func emitRecvPoll(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})

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
	at = e.CallRel32()
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
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	if err := emitCurrentTaskGroupCanceledCheck(e, func() {
		e.XorEaxEax()
		e.MovEdxImm32(1)
		e.Ret()
	}); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

	setCurrentActorWakeAtFromR13(e)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
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
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	if err := emitCurrentTaskGroupCanceledCheck(e, func() {
		e.XorEaxEax()
		e.MovEdxImm32(0)
		e.MovR8dImm32(1)
		e.Ret()
	}); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

	setCurrentActorWakeAtFromR13(e)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
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
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	haveMsgAt := e.JnzRel32()

	clearCurrentActorWakeAt(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
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
	e.MovEaxFromRdiDisp(msgTagOff)
	e.PushRax()
	e.MovEaxFromRdiDisp(msgSenderOff)
	e.PushRax()
	emitCopyMessageNodeToRecvScratchFromRspDisp(e, 16)

	e.MovRaxFromRspDisp(16)
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
	emitDecrementMailboxCount(e)
	emitRecycleMessageNodeFromRspDisp(e, 16)

	e.PopRax()
	e.MovRdiRdx()
	e.MovMem32RdiDispEax(actorLastSenderOff)

	e.PopRax()
	e.AddRspImm32(8)
	e.Ret()
	return nil
}

func emitRecvSlot(e *x64.Emitter) error {
	// Args: rdi=index.
	e.MovRaxRdi()
	e.ShlRaxImm8(3)
	e.AddRaxImm32(msgPayload0)
	e.MovRdiR15()
	e.MovRdxRax()
	e.MovRaxFromRdiDisp(schedPendingMsgOff)
	e.AddRaxRdx()
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(0)
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
