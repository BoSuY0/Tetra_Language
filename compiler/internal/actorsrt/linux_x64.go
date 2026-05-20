package actorsrt

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/format/tobj"
)

const (
	schedActorsPtrOff    = 0   // u64
	schedCapacityOff     = 8   // u32
	schedCountOff        = 12  // u32
	schedRspOff          = 16  // u64
	schedCurrentIdxOff   = 24  // u32
	schedMsgBaseOff      = 32  // u64
	schedMsgBumpOff      = 40  // u64
	schedMsgEndOff       = 48  // u64
	schedPendingMsgOff   = 56  // u64
	schedGroupCountOff   = 64  // u32
	schedGroupState0Off  = 68  // u32[maxTaskGroups]
	schedCloseGroupOff   = 100 // u32
	schedCurrentGroupOff = 104 // u32
	schedSpawnGroupOff   = 108 // u32
	schedTimeMsOff       = 112 // u32
	schedNetFDOff        = 116 // i32
	schedNodeIDOff       = 120 // u32
	schedNetStatusOff    = 124 // u32
	maxActors            = 128
	schedActorGroup0Off  = 128  // u32[maxActors]
	schedActorWakeAt0Off = 640  // u32[maxActors]
	schedActorWait0Off   = 1152 // u32[maxActors]
	schedSize            = 1664
	actorSizeShift       = 7 // 128 bytes
	actorSize            = 1 << actorSizeShift
	actorRspOff          = 0  // u64
	actorStatusOff       = 8  // u32
	actorEntryIDOff      = 12 // u32
	actorMailboxHeadOff  = 16 // u64
	actorMailboxTailOff  = 24 // u64
	actorLastSenderOff   = 32 // u32
	actorExitCodeOff     = 36 // u32
	actorTaskCountOff    = 40 // u32
	actorTaskSlot0Off    = 44 // u32[8]
	actorTaskGroupOff    = 76 // u32
	actorStateSlot0Off   = 80 // i32[8]
	maxActorStateSlots   = 8

	statusFree     = 0
	statusReady    = 1
	statusBlocked  = 2
	statusDone     = 3
	statusSleeping = 4
	statusWaiting  = 5

	maxTaskGroups     = 8
	taskGroupFree     = 0
	taskGroupOpen     = 1
	taskGroupCanceled = 2
	taskGroupClosed   = 3
)

const (
	stackSize      = 64 * 1024
	msgPoolSize    = 64 * 1024
	schedAllocSize = schedSize + maxActors*actorSize
)

// BuildLinuxX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
// - __tetra_actor_send_msg / __tetra_actor_recv_msg
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in `core.spawn(...)`.
func BuildLinuxX64(entries []string) (*tobj.Object, error) {
	abi := x64abi.LinuxSysV()
	const linuxMapPrivateAnon = 0x22
	return buildSysVUnixX64(entries, abi.SysMmap, linuxMapPrivateAnon, true)
}

func buildSysVUnixX64(entries []string, sysMmap uint32, mapFlags uint32, distributedActorNet bool) (*tobj.Object, error) {
	if err := validateRuntimeEntrySymbols(entries); err != nil {
		return nil, err
	}

	e := &x64.Emitter{}
	funcOffsets := make(map[string]int)
	var callPatches []callPatch
	var leaPatches []leaPatch

	emitFunc := func(name string, fn func() error) error {
		if _, exists := funcOffsets[name]; exists {
			return fmt.Errorf("duplicate runtime function '%s'", name)
		}
		funcOffsets[name] = len(e.Buf)
		return fn()
	}

	if err := emitFunc("__tetra_entry", func() error { return emitEntry(e, entries[0], sysMmap, mapFlags, &callPatches, &leaPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_switch_to", func() error { return emitSwitchTo(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield", func() error { return emitActorYield(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield_now", func() error { return emitActorYieldNow(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_exit", func() error { return emitActorExit(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_trampoline", func() error { return emitActorTrampoline(e, &callPatches) }); err != nil {
		return nil, err
	}

	if err := emitFunc("__tetra_actor_spawn", func() error { return emitSpawn(e, sysMmap, mapFlags, &callPatches, &leaPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg", func() error { return emitSendMsg(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_begin", func() error { return emitSendBegin(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_slot", func() error { return emitSendSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_commit", func() error { return emitSendCommit(e) }); err != nil {
		return nil, err
	}
	if distributedActorNet {
		if err := emitFunc("__tetra_actor_net_pump", func() error { return emitActorNetPump(e) }); err != nil {
			return nil, err
		}
	} else {
		if err := emitFunc("__tetra_actor_net_pump", func() error { return emitActorNetPumpNoop(e) }); err != nil {
			return nil, err
		}
	}
	if err := emitFunc("__tetra_actor_recv", func() error { return emitRecv(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg", func() error { return emitRecvMsg(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_poll", func() error { return emitRecvPoll(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_until", func() error { return emitRecvUntil(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg_until", func() error { return emitRecvMsgUntil(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_begin", func() error { return emitRecvBegin(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_slot", func() error { return emitRecvSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_count", func() error { return emitRecvCount(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender", func() error { return emitSender(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_load", func() error { return emitActorStateLoad(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_state_store", func() error { return emitActorStateStore(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_i32", func() error { return emitTaskSpawnI32(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_open", func() error { return emitTaskGroupOpen(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_close", func() error { return emitTaskGroupClose(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_cancel", func() error { return emitTaskGroupCancel(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_current", func() error { return emitTaskGroupCurrent(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_group_status", func() error { return emitTaskGroupStatus(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_is_canceled", func() error { return emitTaskIsCanceled(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_checkpoint", func() error { return emitTaskCheckpoint(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_time_now_ms", func() error { return emitTimeNowMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_ms", func() error { return emitSleepMs(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_sleep_until_ms", func() error { return emitSleepUntilMs(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_deadline_ms", func() error { return emitDeadlineMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_timer_ready_ms", func() error { return emitTimerReadyMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_fs_exists", func() error { return emitFilesystemExists(e) }); err != nil {
		return nil, err
	}
	if distributedActorNet {
		if err := emitFunc("__tetra_actor_node_connect", func() error { return emitActorNodeConnect(e) }); err != nil {
			return nil, err
		}
		if err := emitFunc("__tetra_actor_spawn_remote", func() error { return emitActorSpawnRemote(e) }); err != nil {
			return nil, err
		}
		if err := emitFunc("__tetra_actor_node_status", func() error { return emitActorNodeStatus(e) }); err != nil {
			return nil, err
		}
	}
	if err := emitFunc("__tetra_task_spawn_group_i32", func() error { return emitTaskSpawnGroupI32(e, "__tetra_actor_spawn", &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_i32", func() error { return emitTaskJoinI32(e, false, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_result_i32", func() error { return emitTaskJoinI32(e, true, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_join_until_i32", func() error { return emitTaskJoinUntilI32(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_poll_i32", func() error { return emitTaskPollI32(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_begin", func() error { return emitTaskResultBegin(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_slot", func() error { return emitTaskResultSlot(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_result_get", func() error { return emitTaskResultGet(e) }); err != nil {
		return nil, err
	}
	for slots := 2; slots <= 8; slots++ {
		name := fmt.Sprintf("__tetra_task_join_typed_%d", slots)
		slotCount := slots
		if err := emitFunc(name, func() error { return emitTaskJoinTyped(e, slotCount, &callPatches) }); err != nil {
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

func validateRuntimeEntrySymbols(entries []string) error {
	if len(entries) == 0 || entries[0] == "" {
		return fmt.Errorf("missing entry symbols (need main at index 0)")
	}
	seen := make(map[string]struct{}, len(entries))
	for i, name := range entries {
		if name == "" {
			return fmt.Errorf("empty runtime entry symbol at index %d", i)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate runtime entry symbol '%s'", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

type callPatch struct {
	at   int
	name string
}

type leaPatch struct {
	at   int
	name string
}
