package actorsrt

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
)

const (
	winImportVirtualAlloc = "kernel32.VirtualAlloc"
)

// BuildWindowsX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
// - __tetra_actor_send_msg / __tetra_actor_recv_msg
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in `core.spawn(...)`.
func BuildWindowsX64(entries []string) (*tobj.Object, error) {
	if err := validateRuntimeEntrySymbols(entries); err != nil {
		return nil, err
	}

	e := &x64.Emitter{}
	funcOffsets := make(map[string]int)
	var callPatches []callPatch
	var leaPatches []leaPatch
	var jmpPatches []callPatch
	var importPatches []importPatch

	emitFunc := func(name string, fn func() error) error {
		if _, exists := funcOffsets[name]; exists {
			return fmt.Errorf("duplicate runtime function '%s'", name)
		}
		funcOffsets[name] = len(e.Buf)
		return fn()
	}

	if err := emitFunc("__tetra_entry", func() error { return emitEntryWindowsX64(e, entries[0], &callPatches, &leaPatches, &importPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_switch_to", func() error { return emitSwitchToWindowsX64(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield", func() error { return emitActorYieldWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield_now_impl", func() error { return emitActorYieldNow(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_exit", func() error { return emitActorExitWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_trampoline", func() error { return emitActorTrampolineWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}

	if err := emitFunc("__tetra_actor_spawn_impl", func() error { return emitSpawnWindowsX64(e, &callPatches, &leaPatches, &importPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_impl", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg_impl", func() error { return emitSendMsg(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_begin_impl", func() error { return emitSendBegin(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_slot_impl", func() error { return emitSendSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_commit_impl", func() error { return emitSendCommit(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_net_pump", func() error { return emitActorNetPumpNoop(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_impl", func() error { return emitRecv(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg_impl", func() error { return emitRecvMsg(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_poll_impl", func() error { return emitRecvPoll(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_until_impl", func() error { return emitRecvUntil(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg_until_impl", func() error { return emitRecvMsgUntil(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_begin_impl", func() error { return emitRecvBegin(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_slot_impl", func() error { return emitRecvSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_count_impl", func() error { return emitRecvCount(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self_impl", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender_impl", func() error { return emitSender(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_load_impl", func() error { return emitActorStateLoad(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_store_impl", func() error { return emitActorStateStore(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_i32_impl", func() error { return emitTaskSpawnI32To(e, "__tetra_actor_spawn_impl", &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_open_impl", func() error { return emitTaskGroupOpen(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_close_impl", func() error { return emitTaskGroupClose(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_cancel_impl", func() error { return emitTaskGroupCancel(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_current_impl", func() error { return emitTaskGroupCurrent(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_status_impl", func() error { return emitTaskGroupStatus(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_is_canceled_impl", func() error { return emitTaskIsCanceled(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_checkpoint_impl", func() error { return emitTaskCheckpoint(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_time_now_ms_impl", func() error { return emitTimeNowMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_ms_impl", func() error { return emitSleepMs(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_until_ms_impl", func() error { return emitSleepUntilMs(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_deadline_ms_impl", func() error { return emitDeadlineMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_timer_ready_ms_impl", func() error { return emitTimerReadyMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_group_i32_impl", func() error {
		return emitTaskSpawnGroupI32(e, "__tetra_actor_spawn_impl", &callPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_i32_impl", func() error { return emitTaskJoinI32(e, false, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_result_i32_impl", func() error { return emitTaskJoinI32(e, true, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_until_i32_impl", func() error { return emitTaskJoinUntilI32(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_poll_i32_impl", func() error { return emitTaskPollI32(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_begin_impl", func() error { return emitTaskResultBegin(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_slot_impl", func() error { return emitTaskResultSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_get_impl", func() error { return emitTaskResultGet(e) }); err != nil {
		return nil, err
	}
	for slots := 2; slots <= 8; slots++ {
		name := fmt.Sprintf("__tetra_task_join_typed_%d_impl", slots)
		slotCount := slots
		if err := emitFunc(name, func() error { return emitTaskJoinTyped(e, slotCount, &callPatches) }); err != nil {
			return nil, err
		}
	}

	if err := emitFunc("__tetra_actor_spawn", func() error { return emitActorSpawnWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send", func() error { return emitActorSendWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg", func() error { return emitActorSendMsgWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_begin", func() error { return emitActorSendBeginWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_slot", func() error { return emitActorSendSlotWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_commit", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_send_commit_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_msg_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_poll", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_poll_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_until", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_actor_recv_until_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg_until", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_actor_recv_msg_until_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_begin", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_begin_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_slot", func() error { return emitActorOneArgWrapperWindowsX64(e, "__tetra_actor_recv_slot_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_count", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_count_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_self_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_sender_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_load", func() error { return emitActorOneArgWrapperWindowsX64(e, "__tetra_actor_state_load_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_store", func() error { return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_actor_state_store_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield_now", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_yield_now_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_i32", func() error { return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_spawn_i32_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_open", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_task_group_open_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_close", func() error { return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_group_close_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_cancel", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_group_cancel_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_current", func() error {
		return emitActorNoArgWrapperWindowsX64(e, "__tetra_task_group_current_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_status", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_group_status_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_is_canceled", func() error {
		return emitActorNoArgWrapperWindowsX64(e, "__tetra_task_is_canceled_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_checkpoint", func() error {
		return emitActorNoArgWrapperWindowsX64(e, "__tetra_task_checkpoint_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_time_now_ms", func() error {
		return emitActorNoArgWrapperWindowsX64(e, "__tetra_time_now_ms_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_ms", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_sleep_ms_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_until_ms", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_sleep_until_ms_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_deadline_ms", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_deadline_ms_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_timer_ready_ms", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_timer_ready_ms_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_group_i32", func() error {
		return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_task_spawn_group_i32_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_i32", func() error { return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_task_join_i32_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_result_i32", func() error {
		return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_task_join_result_i32_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_until_i32", func() error {
		return emitTaskThreeArgWrapperWindowsX64(e, "__tetra_task_join_until_i32_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_poll_i32", func() error {
		return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_task_poll_i32_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_begin", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_result_begin_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_slot", func() error { return emitTaskTwoArgWrapperWindowsX64(e, "__tetra_task_result_slot_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_get", func() error {
		return emitActorOneArgWrapperWindowsX64(e, "__tetra_task_result_get_impl", &jmpPatches)
	}); err != nil {
		return nil, err
	}
	for slots := 2; slots <= 8; slots++ {
		name := fmt.Sprintf("__tetra_task_join_typed_%d", slots)
		target := fmt.Sprintf("__tetra_task_join_typed_%d_impl", slots)
		slotCount := slots
		if err := emitFunc(name, func() error { return emitTaskJoinTypedWrapperWindowsX64(e, slotCount, target, &jmpPatches) }); err != nil {
			return nil, err
		}
	}

	code := e.Buf
	for _, patch := range leaPatches {
		target, ok := funcOffsets[patch.name]
		if !ok {
			return nil, fmt.Errorf("unknown lea target '%s'", patch.name)
		}
		if err := x64.PatchRel32(code, patch.at, target); err != nil {
			return nil, err
		}
	}

	var relocs []tobj.Reloc
	for _, patch := range callPatches {
		target, ok := funcOffsets[patch.name]
		if ok {
			if err := x64.PatchRel32(code, patch.at, target); err != nil {
				return nil, err
			}
			continue
		}
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocCallRel32, At: uint32(patch.at), Name: patch.name, Addend: 0})
	}
	for _, patch := range jmpPatches {
		target, ok := funcOffsets[patch.name]
		if !ok {
			return nil, fmt.Errorf("unknown jmp target '%s'", patch.name)
		}
		if err := x64.PatchRel32(code, patch.at, target); err != nil {
			return nil, err
		}
	}
	for _, patch := range importPatches {
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocIATDisp32, At: uint32(patch.at), Name: patch.name, Addend: 0})
	}

	names := make([]string, 0, len(funcOffsets))
	for name := range funcOffsets {
		names = append(names, name)
	}
	sort.Strings(names)
	symbols := make([]tobj.Symbol, 0, len(names))
	for _, name := range names {
		symbols = append(symbols, tobj.Symbol{Name: name, Offset: uint32(funcOffsets[name])})
	}

	return &tobj.Object{Code: code, Data: nil, Symbols: symbols, Relocs: relocs}, nil
}

type importPatch struct {
	at   int
	name string
}
