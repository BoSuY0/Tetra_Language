package actorsrt

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/parallelrt"
)

// ---- linux_x64.go ----

const (
	schedActorsPtrOff                     = 0   // u64
	schedCapacityOff                      = 8   // u32
	schedCountOff                         = 12  // u32
	schedRspOff                           = 16  // u64
	schedCurrentIdxOff                    = 24  // u32
	schedMsgBaseOff                       = 32  // u64
	schedMsgBumpOff                       = 40  // u64
	schedMsgEndOff                        = 48  // u64
	schedPendingMsgOff                    = 56  // u64
	schedGroupCountOff                    = 64  // u32
	schedGroupState0Off                   = 68  // u32[maxTaskGroups]
	schedCloseGroupOff                    = 100 // u32
	schedCurrentGroupOff                  = 104 // u32
	schedSpawnGroupOff                    = 108 // u32
	schedTimeMsOff                        = 112 // u32
	schedNetFDOff                         = 116 // i32
	schedNodeIDOff                        = 120 // u32
	schedNetStatusOff                     = 124 // u32
	schedMsgFreeOff                       = 128 // u64
	schedRecvScratchOff                   = 136 // message-layout scratch for typed recv slots
	schedMsgPoolCapacityBytesOff          = 224 // u64
	schedMsgPoolLiveBytesOff              = 232 // u64
	schedMsgPoolReclaimedBytesOff         = 240 // u64
	schedMsgPoolAllocFailuresOff          = 248 // u64
	maxActors                             = 128
	schedActorGroup0Off                   = 256  // u32[maxActors]
	schedActorWakeAt0Off                  = 768  // u32[maxActors]
	schedActorWait0Off                    = 1280 // u32[maxActors]
	schedRemoteGeneration0Off             = 1792 // u32[maxActors]
	schedNodeEpoch0Off                    = 2304 // u32[maxActors], keyed by node id
	schedMonitorNextIDOff                 = 2816 // u32
	schedMonitorCountOff                  = 2820 // u32
	schedMonitorID0Off                    = 2824 // u32[maxActorMonitors]
	schedMonitorOwnerRef0Off              = 3336 // u32[maxActorMonitors], local ref low words
	schedMonitorTargetRef0Off             = 3848 // u32[maxActorMonitors], local ref low words
	schedMonitorTargetHigh0Off            = 4360 // u32[maxActorMonitors], v2 target high words
	schedSystemEventBaseOff               = 4872 // u64
	schedSystemEventBumpOff               = 4880 // u64
	schedSystemEventEndOff                = 4888 // u64
	schedSystemEventFreeOff               = 4896 // u64
	schedSystemEventCapacityBytesOff      = 4904 // u64
	schedSystemEventLiveBytesOff          = 4912 // u64
	schedSystemEventPeakBytesOff          = 4920 // u64
	schedSystemEventReclaimedBytesOff     = 4928 // u64
	schedSystemEventAllocFailuresOff      = 4936 // u64
	schedSystemEventReservedCreditsOff    = 4944 // u64
	schedRuntimeClosingOff                = 4952 // u32
	schedSize                             = 4960
	actorSizeShift                        = 9 // 512 bytes
	actorSize                             = 1 << actorSizeShift
	actorRspOff                           = 0   // u64
	actorStatusOff                        = 8   // u32
	actorEntryIDOff                       = 12  // u32
	actorMailboxHeadOff                   = 16  // u64
	actorMailboxTailOff                   = 24  // u64
	actorLastSenderOff                    = 32  // u32
	actorExitCodeOff                      = 36  // u32
	actorTaskCountOff                     = 40  // u32
	actorTaskSlot0Off                     = 44  // u32[8]
	actorTaskGroupOff                     = 76  // u32
	actorStateSlot0Off                    = 80  // i32[8]
	actorMailboxCountOff                  = 112 // u32
	actorMailboxBytesOff                  = 120 // u64
	actorMailboxPeakBytesOff              = 128 // u64
	actorReclaimedBytesOff                = 136 // u64
	actorBytesCopiedOff                   = 144 // u64
	actorCopyCountOff                     = 152 // u64
	actorByteBudgetOff                    = 160 // u64
	actorOverBudgetCountOff               = 168 // u64
	actorBackpressureEventsOff            = 176 // u64
	actorGenerationOff                    = 184 // u32
	actorTrapExitOff                      = 188 // u32
	actorStackInitRspOff                  = 192 // u64
	actorLinkCountOff                     = 200 // u32
	actorLinkRef0Off                      = 204 // u32[maxActorLinks], v2 ref low words
	maxActorStateSlots                    = 8
	maxActorLinks                         = 8
	actorLinkHigh0Off                     = actorLinkRef0Off + maxActorLinks*4 // u32[maxActorLinks], v2 ref high words
	actorSystemMailboxHeadOff             = 272                                // u64
	actorSystemMailboxTailOff             = 280                                // u64
	actorSystemMailboxCountOff            = 288                                // u32
	actorSystemMailboxReservedCreditsOff  = 292                                // u32
	actorSystemMailboxBytesOff            = 296                                // u64
	actorSystemMailboxPeakBytesOff        = 304                                // u64
	actorSystemMailboxReclaimedBytesOff   = 312                                // u64
	actorSystemMailboxOverflowAttemptsOff = 320                                // u64
	actorSystemRecvScratch0Off            = 328                                // u64[7]
	actorSystemRecvScratchCountOff        = 384                                // u32
	actorSystemRecvScratchStatusOff       = 388                                // u32
	actorWaitKindOff                      = 392                                // u32
	actorTerminalReasonKindOff            = 396                                // u32
	actorTerminalReasonCodeOff            = 400                                // u32
	maxActorMonitors                      = 128
	maxActorMailboxMsgs                   = 256
	maxActorMailboxBytes                  = maxActorMailboxMsgs * msgSize

	actorGenerationInitial = 1
	actorGenerationMax     = 0xFFFF
	actorRefLocalHighSlot  = 0x10000000
	actorRefRemoteHighMask = 0xF8000000
	actorRefRemoteHighBase = 0x18000000
	actorNodeEpochInitial  = 1

	statusFree        = 0
	statusReady       = 1
	statusBlocked     = 2
	statusDone        = 3
	statusSleeping    = 4
	statusWaiting     = 5
	statusRunning     = 6
	statusStarting    = 7
	statusStopping    = 8
	statusReclaimable = 9

	actorLifecycleStatusStarting     = 0
	actorLifecycleStatusReady        = 1
	actorLifecycleStatusRunning      = 2
	actorLifecycleStatusBlocked      = 3
	actorLifecycleStatusSleeping     = 4
	actorLifecycleStatusWaiting      = 5
	actorLifecycleStatusStopping     = 6
	actorLifecycleStatusExitedNormal = 7
	actorLifecycleStatusExitedError  = 8
	actorLifecycleStatusCanceled     = 9
	actorLifecycleStatusRestarting   = 10
	actorLifecycleStatusDead         = 11

	maxTaskGroups     = 8
	taskGroupFree     = 0
	taskGroupOpen     = 1
	taskGroupCanceled = 2
	taskGroupClosed   = 3

	actorWaitKindNone   = 0
	actorWaitKindUser   = 1
	actorWaitKindSystem = 2
)

const (
	stackSize                  = 64 * 1024
	maxRetainedDoneActorStacks = 8
	msgPoolSize                = 64 * 1024
	systemEventPoolSize        = 64 * 1024
	schedAllocSize             = schedSize + maxActors*actorSize
)

const (
	actorMemorySnapshotRecordSize                 = 104
	actorMemorySnapshotActorIDOffset        int32 = 0
	actorMemorySnapshotCurrentOffset        int32 = 8
	actorMemorySnapshotPeakOffset           int32 = 16
	actorMemorySnapshotBytesCopiedOffset    int32 = 24
	actorMemorySnapshotByteBudgetOffset     int32 = 32
	actorMemorySnapshotOverBudgetOffset     int32 = 40
	actorMemorySnapshotBackpressureOffset   int32 = 48
	actorMemorySnapshotMailboxCurrentOffset int32 = 56
	actorMemorySnapshotMailboxPeakOffset    int32 = 64
	actorMemorySnapshotStackLiveOffset      int32 = 72
	actorMemorySnapshotStackReservedOffset  int32 = 80
	actorMemorySnapshotStackRetainedOffset  int32 = 88
	actorMemorySnapshotStackReleasedOffset  int32 = 96
)

// BuildLinuxX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
// - __tetra_actor_send_msg / __tetra_actor_recv_msg
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in
// `core.spawn(...)`.
func BuildLinuxX64(entries []string) (*tobj.Object, error) {
	abi := x64abi.LinuxSysV()
	const linuxMapPrivateAnon = 0x22
	return buildSysVUnixX64(
		entries,
		abi.SysMmap,
		abi.SysMunmap,
		linuxMapPrivateAnon,
		true,
		SurfaceHostIPCOptions{},
	)
}

type SurfaceHostIPCOptions struct {
	SocketPath string
}

func (opt SurfaceHostIPCOptions) enabled() bool {
	return strings.TrimSpace(opt.SocketPath) != ""
}

func BuildLinuxX64WithSurfaceHostIPC(
	entries []string,
	opt SurfaceHostIPCOptions,
) (*tobj.Object, error) {
	if strings.TrimSpace(opt.SocketPath) == "" {
		return nil, fmt.Errorf("surface host IPC socket path is required")
	}
	abi := x64abi.LinuxSysV()
	const linuxMapPrivateAnon = 0x22
	return buildSysVUnixX64(entries, abi.SysMmap, abi.SysMunmap, linuxMapPrivateAnon, true, opt)
}

func buildSysVUnixX64(
	entries []string,
	sysMmap uint32,
	sysMunmap uint32,
	mapFlags uint32,
	distributedActorNet bool,
	surfaceHost SurfaceHostIPCOptions,
) (*tobj.Object, error) {
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

	if err := emitFunc(
		"__tetra_entry",
		func() error { return emitEntry(e, entries[0], sysMmap, sysMunmap, mapFlags, &callPatches, &leaPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_switch_to",
		func() error { return emitSwitchTo(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_yield",
		func() error { return emitActorYield(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_yield_now",
		func() error { return emitActorYieldNow(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_memory_snapshot",
		func() error { return emitActorMemorySnapshot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_exit",
		func() error { return emitActorExit(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_trampoline",
		func() error { return emitActorTrampoline(e, &callPatches) },
	); err != nil {
		return nil, err
	}

	if err := emitFunc(
		"__tetra_actor_spawn",
		func() error { return emitSpawn(e, sysMmap, mapFlags, &callPatches, &leaPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg", func() error { return emitSendMsg(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_begin",
		func() error { return emitSendBegin(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_slot",
		func() error { return emitSendSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_commit",
		func() error { return emitSendCommit(e) },
	); err != nil {
		return nil, err
	}
	if distributedActorNet {
		if err := emitFunc(
			"__tetra_actor_net_pump",
			func() error { return emitActorNetPump(e) },
		); err != nil {
			return nil, err
		}
	} else {
		if err := emitFunc(
			"__tetra_actor_net_pump",
			func() error { return emitActorNetPumpNoop(e) },
		); err != nil {
			return nil, err
		}
	}
	if err := emitFunc(
		"__tetra_actor_recv",
		func() error { return emitRecv(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_msg",
		func() error { return emitRecvMsg(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_poll",
		func() error { return emitRecvPoll(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_until",
		func() error { return emitRecvUntil(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_msg_until",
		func() error { return emitRecvMsgUntil(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_begin",
		func() error { return emitRecvBegin(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_slot",
		func() error { return emitRecvSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_count",
		func() error { return emitRecvCount(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_system_begin",
		func() error { return emitRecvSystemBegin(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_system_slot",
		func() error { return emitRecvSystemSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_system_count",
		func() error { return emitRecvSystemCount(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender", func() error { return emitSender(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_status", func() error { return emitActorStatus(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_status_raw", func() error { return emitActorStatusRaw(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_wait", func() error { return emitActorWait(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_wait_until", func() error { return emitActorWaitUntil(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_stop", func() error { return emitActorStop(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_exit_reason", func() error { return emitActorExitReason(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_link", func() error { return emitActorLink(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_unlink", func() error { return emitActorUnlink(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_monitor", func() error { return emitActorMonitor(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_demonitor", func() error { return emitActorDemonitor(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_set_trap_exit", func() error { return emitActorSetTrapExit(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_load",
		func() error { return emitActorStateLoad(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_store",
		func() error { return emitActorStateStore(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_spawn_i32",
		func() error { return emitTaskSpawnI32(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_open",
		func() error { return emitTaskGroupOpen(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_close",
		func() error { return emitTaskGroupClose(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_cancel",
		func() error { return emitTaskGroupCancel(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_current",
		func() error { return emitTaskGroupCurrent(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_status",
		func() error { return emitTaskGroupStatus(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_is_canceled",
		func() error { return emitTaskIsCanceled(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_checkpoint",
		func() error { return emitTaskCheckpoint(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_time_now_ms", func() error { return emitTimeNowMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_sleep_ms",
		func() error { return emitSleepMs(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_sleep_until_ms",
		func() error { return emitSleepUntilMs(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_deadline_ms", func() error { return emitDeadlineMs(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_timer_ready_ms",
		func() error { return emitTimerReadyMs(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_fs_exists",
		func() error { return emitFilesystemExists(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_socket_tcp4",
		func() error { return emitNetSocketTCP4(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_bind_tcp4_loopback",
		func() error { return emitNetBindTCP4Loopback(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_connect_tcp4_loopback",
		func() error { return emitNetConnectTCP4Loopback(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_listen", func() error { return emitNetListen(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_accept4", func() error { return emitNetAccept4(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_read", func() error { return emitNetRead(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_recv", func() error { return emitNetRecv(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_write", func() error { return emitNetWrite(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_send", func() error { return emitNetSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_create",
		func() error { return emitNetEpollCreate(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_ctl_add_read",
		func() error { return emitNetEpollCtlAddRead(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_ctl_add_read_write",
		func() error { return emitNetEpollCtlAddReadWrite(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_ctl_mod_read",
		func() error { return emitNetEpollCtlModRead(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_ctl_mod_read_write",
		func() error { return emitNetEpollCtlModReadWrite(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_ctl_delete",
		func() error { return emitNetEpollCtlDelete(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_wait_one",
		func() error { return emitNetEpollWaitOne(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_epoll_wait_one_into",
		func() error { return emitNetEpollWaitOneInto(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_set_nonblocking",
		func() error { return emitNetSetNonblocking(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_set_reuseport",
		func() error { return emitNetSetReusePort(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_net_set_tcp_nodelay",
		func() error { return emitNetSetTCPNoDelay(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_net_close", func() error { return emitNetClose(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_open", func() error {
		if surfaceHost.enabled() {
			return emitSurfaceOpenHostIPC(e, surfaceHost)
		}
		return emitSurfaceOpen(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_close", func() error {
		if surfaceHost.enabled() {
			return emitSurfaceCloseHostIPC(e)
		}
		return emitSurfaceClose(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_kind",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollEventSlotHostIPC(e, 0)
			}
			return emitSurfaceConst(e, 5)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_x",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollEventSlotHostIPC(e, 1)
			}
			return emitSurfaceConst(e, 48)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_y",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollEventSlotHostIPC(e, 2)
			}
			return emitSurfaceConst(e, 96)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_button",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollEventSlotHostIPC(e, 3)
			}
			return emitSurfaceConst(e, 1)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_poll_event_into", func() error {
		if surfaceHost.enabled() {
			return emitSurfacePollEventIntoHostIPC(e)
		}
		return emitSurfacePollEventInto(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_text_len",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollEventTextLenHostIPC(e)
			}
			return emitSurfaceConst(e, 2)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_event_text_into",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePayloadIntoHostIPC(e, surfaceHostOpPollEventTextInto, 0)
			}
			return emitSurfacePollEventTextInto(e)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_clipboard_write_text",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfaceClipboardWriteTextHostIPC(e)
			}
			return emitSurfaceClipboardWriteText(e)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_clipboard_read_text_into",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePayloadIntoHostIPC(e, surfaceHostOpClipboardReadText, 0)
			}
			return emitSurfaceClipboardReadTextInto(e)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_surface_poll_composition_into",
		func() error {
			if surfaceHost.enabled() {
				return emitSurfacePollCompositionIntoHostIPC(e)
			}
			return emitSurfacePollCompositionInto(e)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_begin_frame", func() error {
		if surfaceHost.enabled() {
			return emitSurfaceOK(e)
		}
		return emitSurfaceOK(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_present_rgba", func() error {
		if surfaceHost.enabled() {
			return emitSurfacePresentRGBAHostIPC(e)
		}
		return emitSurfacePresentRGBA(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_now_ms", func() error {
		if surfaceHost.enabled() {
			return emitSurfaceNowMSHostIPC(e)
		}
		return emitSurfaceOK(e)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_surface_request_redraw", func() error {
		if surfaceHost.enabled() {
			return emitSurfaceSimpleHostIPC(e, surfaceHostOpRequestRedraw)
		}
		return emitSurfaceOK(e)
	}); err != nil {
		return nil, err
	}
	if distributedActorNet {
		if err := emitFunc(
			"__tetra_actor_node_connect",
			func() error { return emitActorNodeConnect(e) },
		); err != nil {
			return nil, err
		}
		if err := emitFunc(
			"__tetra_actor_spawn_remote",
			func() error { return emitActorSpawnRemote(e) },
		); err != nil {
			return nil, err
		}
		if err := emitFunc(
			"__tetra_actor_node_status",
			func() error { return emitActorNodeStatus(e) },
		); err != nil {
			return nil, err
		}
	}
	if err := emitFunc(
		"__tetra_task_spawn_group_i32",
		func() error { return emitTaskSpawnGroupI32(e, "__tetra_actor_spawn", &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_i32",
		func() error { return emitTaskJoinI32(e, false, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_result_i32",
		func() error { return emitTaskJoinI32(e, true, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_until_i32",
		func() error { return emitTaskJoinUntilI32(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_poll_i32",
		func() error { return emitTaskPollI32(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_begin",
		func() error { return emitTaskResultBegin(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_slot",
		func() error { return emitTaskResultSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_get",
		func() error { return emitTaskResultGet(e) },
	); err != nil {
		return nil, err
	}
	for slots := 2; slots <= 8; slots++ {
		name := fmt.Sprintf("__tetra_task_join_typed_%d", slots)
		slotCount := slots
		if err := emitFunc(
			name,
			func() error { return emitTaskJoinTyped(e, slotCount, &callPatches) },
		); err != nil {
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
		relocs = append(
			relocs,
			tobj.Reloc{
				Kind:   tobj.RelocCallRel32,
				At:     uint32(patch.at),
				Name:   patch.name,
				Addend: 0,
			},
		)
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

// ---- linux_x64_emit.go ----

const (
	msgNextOff       = 0  // u64
	msgSenderOff     = 8  // u32
	msgValueOff      = 12 // u32
	msgTagOff        = 16 // u32
	msgCountOff      = 20 // u32
	msgSystemKindOff = 24 // u32, runtime-only lane; ordinary send paths write actorSystemKindUser
	msgPayload0      = 32 // u64[8]
	msgSize          = 96
	msgSlotSize      = 8
)

const (
	systemEventNextOff       = 0  // u64
	systemEventKindOff       = 8  // u32
	systemEventFlagsOff      = 12 // u32
	systemEventSubjectOff    = 16 // u64, raw actor subject low word for P01-SM5
	systemEventMonitorRefOff = 24 // u64, actor.monitor-compatible raw slot
	systemEventNodeIDOff     = 32 // u64
	systemEventNodeEpochOff  = 40 // u64
	systemEventReasonKindOff = 48 // u32
	systemEventReasonCodeOff = 52 // u32
	systemEventDedupKeyOff   = 56 // u64
	systemEventSize          = 64

	maxSystemEventReservedCredits = systemEventPoolSize / systemEventSize
)

const (
	actorSystemKindUser     = 0
	actorSystemKindExit     = 1
	actorSystemKindDown     = 2
	actorSystemKindNodeDown = 3

	actorExitReasonKindNodeDown = 5

	actorSystemExitPayloadSlots     = 2
	actorSystemDownPayloadSlots     = 3
	actorSystemNodeDownPayloadSlots = 4

	actorSystemRecvStatusMessage       = 0
	actorSystemRecvStatusEmpty         = 1
	actorSystemRecvStatusTimeout       = 2
	actorSystemRecvStatusCanceled      = 3
	actorSystemRecvStatusRuntimeClosed = 4
	actorSystemRecvStatusInvalidState  = 5
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
	actorWireFrameSpawnAck  = 4
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

	surfaceHostMagic                 = 0x31534854
	surfaceHostRequestHeaderSize     = 32
	surfaceHostResponseSize          = 36
	surfaceHostEventPayloadSize      = 36
	surfaceHostEventSlots            = 9
	surfaceHostOpOpen                = 1
	surfaceHostOpClose               = 2
	surfaceHostOpBeginFrame          = 3
	surfaceHostOpPresentRGBA         = 4
	surfaceHostOpPollEventInto       = 5
	surfaceHostOpPollEventTextInto   = 6
	surfaceHostOpClipboardWriteText  = 7
	surfaceHostOpClipboardReadText   = 8
	surfaceHostOpPollCompositionInto = 9
	surfaceHostOpNowMS               = 10
	surfaceHostOpRequestRedraw       = 11
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

func emitMovMem8RspDispImm8(e *x64.Emitter, disp byte, val byte) {
	e.Emit(0xC6, 0x44, 0x24, disp, val)
}

func emitMovMem16RspDispAx(e *x64.Emitter, disp byte) {
	e.Emit(0x66, 0x89, 0x44, 0x24, disp)
}

func emitMovMem32RspDispEax(e *x64.Emitter, disp byte) {
	e.Emit(0x89, 0x44, 0x24, disp)
}

func emitMovMem32RspDispEdi(e *x64.Emitter, disp int32) {
	e.Emit(0x89, 0xBC, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem32RspDispEcx(e *x64.Emitter, disp int32) {
	e.Emit(0x89, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem32RspDispEdx(e *x64.Emitter, disp int32) {
	e.Emit(0x89, 0x94, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem32RspDispR8d(e *x64.Emitter, disp int32) {
	e.Emit(0x44, 0x89, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem32RspDispR9d(e *x64.Emitter, disp int32) {
	e.Emit(0x44, 0x89, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem64RspDispRdi(e *x64.Emitter, disp int32) {
	e.Emit(0x48, 0x89, 0xBC, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem64RspDispRsi(e *x64.Emitter, disp int32) {
	e.Emit(0x48, 0x89, 0xB4, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem64RspDispRdx(e *x64.Emitter, disp int32) {
	e.Emit(0x48, 0x89, 0x94, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovRdiFromRspDisp(e *x64.Emitter, disp int32) {
	e.Emit(0x48, 0x8B, 0xBC, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovRsiFromRspDisp(e *x64.Emitter, disp int32) {
	e.Emit(0x48, 0x8B, 0xB4, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
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
	emitAddSchedulerU64Counter(e, schedMsgPoolLiveBytesOff, msgSize)
	return overflowAt
}

func emitRecycleMessageNodeInRax(e *x64.Emitter) {
	e.PushRax()
	emitAddSchedulerU64Counter(e, schedMsgPoolLiveBytesOff, -msgSize)
	emitAddSchedulerU64Counter(e, schedMsgPoolReclaimedBytesOff, msgSize)
	e.PopRax()
	emitScrubMessageNodeForRecycle(e)
	e.MovRdiR15()
	e.MovR8FromRdiDisp(schedMsgFreeOff)
	e.MovRdiRax()
	e.MovMem64RdiDispR8(msgNextOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgFreeOff)
}

func emitScrubMessageNodeForRecycle(e *x64.Emitter) {
	e.PushRax()
	e.MovRdiRax()
	e.XorEaxEax()
	for _, off := range []int32{
		msgSenderOff,
		msgValueOff,
		msgTagOff,
		msgCountOff,
		msgSystemKindOff,
	} {
		e.MovMem32RdiDispEax(off)
	}
	for slot := 0; slot < 8; slot++ {
		e.MovMem64RdiDispRax(msgPayload0 + int32(slot*msgSlotSize))
	}
	e.PopRax()
}

func emitRecycleMessageNodeFromRspDisp(e *x64.Emitter, disp int32) {
	e.MovRaxFromRspDisp(disp)
	emitRecycleMessageNodeInRax(e)
}

func emitCheckedSystemEventPoolAlloc(e *x64.Emitter) int {
	// On success, rdx is the allocated system-event pointer. This pool is
	// intentionally distinct from ordinary user message nodes.
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedSystemEventFreeOff)
	e.TestRaxRax()
	bumpAt := e.JzRel32()

	e.MovRdxRax()
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(systemEventNextOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedSystemEventFreeOff)
	doneAt := e.JmpRel32()

	bumpTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bumpAt, bumpTo); err != nil {
		panic(err)
	}
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedSystemEventBumpOff)
	e.MovRdxRax()
	e.AddRaxImm32(systemEventSize)
	e.MovR8FromRdiDisp(schedSystemEventEndOff)
	e.CmpRaxR8()
	overflowAt := e.JaRel32()
	e.MovMem64RdiDispRax(schedSystemEventBumpOff)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
	emitAddSchedulerU64Counter(e, schedSystemEventLiveBytesOff, systemEventSize)
	emitUpdateSchedulerSystemEventPeak(e)
	return overflowAt
}

func emitUpdateSchedulerSystemEventPeak(e *x64.Emitter) {
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedSystemEventLiveBytesOff)
	e.MovR8FromRdiDisp(schedSystemEventPeakBytesOff)
	e.CmpRaxR8()
	updatePeakAt := e.JaRel32()
	afterPeakAt := e.JmpRel32()
	updatePeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, updatePeakAt, updatePeakTo); err != nil {
		panic(err)
	}
	e.MovMem64RdiDispRax(schedSystemEventPeakBytesOff)
	afterPeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterPeakAt, afterPeakTo); err != nil {
		panic(err)
	}
}

func emitRecycleSystemEventNodeInRax(e *x64.Emitter) {
	e.PushRax()
	emitAddSchedulerU64Counter(e, schedSystemEventLiveBytesOff, -systemEventSize)
	emitAddSchedulerU64Counter(e, schedSystemEventReclaimedBytesOff, systemEventSize)
	e.PopRax()
	emitScrubSystemEventNodeForRecycle(e)
	e.MovRdiR15()
	e.MovR8FromRdiDisp(schedSystemEventFreeOff)
	e.MovRdiRax()
	e.MovMem64RdiDispR8(systemEventNextOff)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedSystemEventFreeOff)
}

func emitScrubSystemEventNodeForRecycle(e *x64.Emitter) {
	e.PushRax()
	e.MovRdiRax()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(systemEventNextOff)
	e.MovMem32RdiDispEax(systemEventKindOff)
	e.MovMem32RdiDispEax(systemEventFlagsOff)
	e.MovMem64RdiDispRax(systemEventSubjectOff)
	e.MovMem64RdiDispRax(systemEventMonitorRefOff)
	e.MovMem64RdiDispRax(systemEventNodeIDOff)
	e.MovMem64RdiDispRax(systemEventNodeEpochOff)
	e.MovMem32RdiDispEax(systemEventReasonKindOff)
	e.MovMem32RdiDispEax(systemEventReasonCodeOff)
	e.MovMem64RdiDispRax(systemEventDedupKeyOff)
	e.PopRax()
}

func emitRecycleSystemEventNodeFromRspDisp(e *x64.Emitter, disp int32) {
	e.MovRaxFromRspDisp(disp)
	emitRecycleSystemEventNodeInRax(e)
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

func emitInitSchedulerMessagePoolCounters(e *x64.Emitter) {
	e.MovEaxImm32(msgPoolSize)
	e.MovMem64RdiDispRax(schedMsgPoolCapacityBytesOff)
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedMsgPoolLiveBytesOff)
	e.MovMem64RdiDispRax(schedMsgPoolReclaimedBytesOff)
	e.MovMem64RdiDispRax(schedMsgPoolAllocFailuresOff)
}

func emitInitSchedulerSystemEventPoolCounters(e *x64.Emitter) {
	e.MovEaxImm32(systemEventPoolSize)
	e.MovMem64RdiDispRax(schedSystemEventCapacityBytesOff)
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedSystemEventLiveBytesOff)
	e.MovMem64RdiDispRax(schedSystemEventPeakBytesOff)
	e.MovMem64RdiDispRax(schedSystemEventReclaimedBytesOff)
	e.MovMem64RdiDispRax(schedSystemEventAllocFailuresOff)
	e.MovMem64RdiDispRax(schedSystemEventReservedCreditsOff)
}

func emitInitActorByteCountersInRdi(e *x64.Emitter) {
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxBytesOff)
	e.MovMem64RdiDispRax(actorMailboxPeakBytesOff)
	e.MovMem64RdiDispRax(actorReclaimedBytesOff)
	e.MovMem64RdiDispRax(actorBytesCopiedOff)
	e.MovMem64RdiDispRax(actorCopyCountOff)
	e.MovMem64RdiDispRax(actorOverBudgetCountOff)
	e.MovMem64RdiDispRax(actorBackpressureEventsOff)
	e.MovEaxImm32(maxActorMailboxBytes)
	e.MovMem64RdiDispRax(actorByteBudgetOff)
}

func emitInitActorSystemMailboxInRdi(e *x64.Emitter) {
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorSystemMailboxHeadOff)
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)
	e.MovMem32RdiDispImm32(actorSystemMailboxCountOff, 0)
	e.MovMem32RdiDispImm32(actorSystemMailboxReservedCreditsOff, 0)
	e.MovMem64RdiDispRax(actorSystemMailboxBytesOff)
	e.MovMem64RdiDispRax(actorSystemMailboxPeakBytesOff)
	e.MovMem64RdiDispRax(actorSystemMailboxReclaimedBytesOff)
	e.MovMem64RdiDispRax(actorSystemMailboxOverflowAttemptsOff)
	for slot := 0; slot < 7; slot++ {
		e.MovMem64RdiDispRax(actorSystemRecvScratch0Off + int32(slot*8))
	}
	e.MovMem32RdiDispImm32(actorSystemRecvScratchCountOff, 0)
	e.MovMem32RdiDispImm32(actorSystemRecvScratchStatusOff, actorSystemRecvStatusEmpty)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	e.MovMem32RdiDispImm32(actorTerminalReasonKindOff, 0)
	e.MovMem32RdiDispImm32(actorTerminalReasonCodeOff, 0)
}

func emitAddSchedulerU64Counter(e *x64.Emitter, off int32, delta int32) {
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(off)
	e.AddRaxImm32(delta)
	e.MovMem64RdiDispRax(off)
}

func emitMessagePoolAllocationFailure(e *x64.Emitter) {
	emitAddSchedulerU64Counter(e, schedMsgPoolAllocFailuresOff, 1)
}

func emitSystemEventPoolAllocationFailure(e *x64.Emitter) {
	emitAddSchedulerU64Counter(e, schedSystemEventAllocFailuresOff, 1)
}

func emitReserveSystemEventCreditForActorPtrInRdi(e *x64.Emitter) {
	e.PushRdi()
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedSystemEventReservedCreditsOff)
	e.CmpRaxImm32(maxSystemEventReservedCredits)
	fullAt := e.JaeRel32()
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(schedSystemEventReservedCreditsOff)
	e.PopRdi()
	e.MovEaxFromRdiDisp(actorSystemMailboxReservedCreditsOff)
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(actorSystemMailboxReservedCreditsOff)
	e.XorEaxEax()
	doneAt := e.JmpRel32()

	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		panic(err)
	}
	e.PopRdi()
	e.MovEaxImm32(0xFFFFFFFE)

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
}

func emitReleaseSystemEventCreditForActorPtrInRdi(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorSystemMailboxReservedCreditsOff)
	e.TestEaxEax()
	doneAt := e.JzRel32()
	e.AddEaxImm32(-1)
	e.MovMem32RdiDispEax(actorSystemMailboxReservedCreditsOff)

	e.PushRdi()
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedSystemEventReservedCreditsOff)
	e.TestRaxRax()
	noSchedulerCreditAt := e.JzRel32()
	e.AddRaxImm32(-1)
	e.MovMem64RdiDispRax(schedSystemEventReservedCreditsOff)
	noSchedulerCreditTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noSchedulerCreditAt, noSchedulerCreditTo); err != nil {
		panic(err)
	}
	e.PopRdi()

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
}

func emitAccountMailboxEnqueueInRdi(e *x64.Emitter) {
	e.MovRaxFromRdiDisp(actorMailboxBytesOff)
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(actorMailboxBytesOff)

	e.MovR8FromRdiDisp(actorMailboxPeakBytesOff)
	e.CmpRaxR8()
	updatePeakAt := e.JaRel32()
	afterPeakAt := e.JmpRel32()
	updatePeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, updatePeakAt, updatePeakTo); err != nil {
		panic(err)
	}
	e.MovMem64RdiDispRax(actorMailboxPeakBytesOff)
	afterPeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterPeakAt, afterPeakTo); err != nil {
		panic(err)
	}

	e.MovRaxFromRdiDisp(actorBytesCopiedOff)
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(actorBytesCopiedOff)
	e.MovRaxFromRdiDisp(actorCopyCountOff)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(actorCopyCountOff)
}

func emitAccountActorByteBackpressureInRdi(e *x64.Emitter) {
	e.MovRaxFromRdiDisp(actorOverBudgetCountOff)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(actorOverBudgetCountOff)
	e.MovRaxFromRdiDisp(actorBackpressureEventsOff)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(actorBackpressureEventsOff)
}

func emitIncrementSystemMailboxCount(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorSystemMailboxCountOff)
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(actorSystemMailboxCountOff)
}

func emitDecrementSystemMailboxCount(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorSystemMailboxCountOff)
	e.AddEaxImm32(-1)
	e.MovMem32RdiDispEax(actorSystemMailboxCountOff)
}

func emitAccountSystemMailboxEnqueueInRdi(e *x64.Emitter) {
	e.MovRaxFromRdiDisp(actorSystemMailboxBytesOff)
	e.AddRaxImm32(systemEventSize)
	e.MovMem64RdiDispRax(actorSystemMailboxBytesOff)

	e.MovR8FromRdiDisp(actorSystemMailboxPeakBytesOff)
	e.CmpRaxR8()
	updatePeakAt := e.JaRel32()
	afterPeakAt := e.JmpRel32()
	updatePeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, updatePeakAt, updatePeakTo); err != nil {
		panic(err)
	}
	e.MovMem64RdiDispRax(actorSystemMailboxPeakBytesOff)
	afterPeakTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, afterPeakAt, afterPeakTo); err != nil {
		panic(err)
	}
}

func emitAccountSystemMailboxDequeueInRdi(e *x64.Emitter) {
	e.MovRaxFromRdiDisp(actorSystemMailboxBytesOff)
	e.AddRaxImm32(-systemEventSize)
	e.MovMem64RdiDispRax(actorSystemMailboxBytesOff)
	e.MovRaxFromRdiDisp(actorSystemMailboxReclaimedBytesOff)
	e.AddRaxImm32(systemEventSize)
	e.MovMem64RdiDispRax(actorSystemMailboxReclaimedBytesOff)
	emitReleaseSystemEventCreditForActorPtrInRdi(e)
}

func emitWakeActorIfBlockedOnSystemMailboxInRdi(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovEaxFromRdiDisp(actorWaitKindOff)
	e.CmpEaxImm32(actorWaitKindSystem)
	notSystemAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, doneTo); err != nil {
		panic(err)
	}
	if err := x64.PatchRel32(e.Buf, notSystemAt, doneTo); err != nil {
		panic(err)
	}
}

func emitWakeActorIfBlockedOnUserMailboxInRdi(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovEaxFromRdiDisp(actorWaitKindOff)
	e.CmpEaxImm32(actorWaitKindUser)
	notUserAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, doneTo); err != nil {
		panic(err)
	}
	if err := x64.PatchRel32(e.Buf, notUserAt, doneTo); err != nil {
		panic(err)
	}
}

func emitEnqueueSystemEventForActorPtrInRdiNodeInRdx(e *x64.Emitter) {
	e.PushRdi() // actor pointer
	e.PushRdx() // system-event node

	e.MovRaxFromRdiDisp(actorSystemMailboxTailOff)
	e.TestRaxRax()
	emptyAt := e.JzRel32()
	e.MovRdiRax()
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(systemEventNextOff)
	emitMovRdiFromRspDisp(e, 8)
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)
	appendedAt := e.JmpRel32()

	emptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, emptyAt, emptyTo); err != nil {
		panic(err)
	}
	emitMovRdiFromRspDisp(e, 8)
	e.MovRaxRdx()
	e.MovMem64RdiDispRax(actorSystemMailboxHeadOff)
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)

	appendedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, appendedAt, appendedTo); err != nil {
		panic(err)
	}
	emitIncrementSystemMailboxCount(e)
	emitAccountSystemMailboxEnqueueInRdi(e)
	emitWakeActorIfBlockedOnSystemMailboxInRdi(e)
	e.AddRspImm32(16)
}

func emitAccountMailboxDequeueInRdi(e *x64.Emitter) {
	e.MovRaxFromRdiDisp(actorMailboxBytesOff)
	e.AddRaxImm32(-msgSize)
	e.MovMem64RdiDispRax(actorMailboxBytesOff)
	e.MovRaxFromRdiDisp(actorReclaimedBytesOff)
	e.AddRaxImm32(msgSize)
	e.MovMem64RdiDispRax(actorReclaimedBytesOff)
}

func emitActorMemorySnapshot(e *x64.Emitter) error {
	// Argument: rdi points at records of u64 fields:
	// actor_id, current_bytes, peak_bytes, bytes_copied, byte_budget,
	// over_budget_count, backpressure_events, mailbox_current_bytes,
	// mailbox_peak_bytes, stack_live_bytes, stack_reserved_bytes,
	// stack_retained_bytes, stack_released_bytes.
	// Returns the number of actor snapshot records in eax and the number of
	// currently live actor slots in edx. Done/reusable slots may still have
	// retained/released stack accounting records, but they are not live actors.
	e.MovRsiRdi()
	e.XorEcxEcx()
	e.MovR9dImm32(0)

	loopOff := len(e.Buf)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	bodyAt := e.JaRel32()
	doneAt := e.JmpRel32()

	bodyOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, bodyAt, bodyOff); err != nil {
		return err
	}
	e.MovEdxEcx()
	e.Emit(0x48, 0x6B, 0xD1, byte(actorMemorySnapshotRecordSize)) // imul rdx, rcx, record size
	e.AddRdxRsi()
	e.MovRaxRcx()
	emitMovMem64RdxDispRax(e, actorMemorySnapshotActorIDOffset)

	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovR8Rdi()

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	notLiveDoneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	notLiveReclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusFree)
	notLiveFreeAt := e.JzRel32()
	e.AddR9Imm32(1)
	liveCountDoneOff := len(e.Buf)
	for _, at := range []int{notLiveDoneAt, notLiveReclaimableAt, notLiveFreeAt} {
		if err := x64.PatchRel32(e.Buf, at, liveCountDoneOff); err != nil {
			return err
		}
	}

	emitMovRaxFromR8Disp(e, actorMailboxBytesOff)
	emitMovMem64RdxDispRax(e, actorMemorySnapshotMailboxCurrentOffset)
	if err := emitStoreActorCounterPlusLiveStack(e, actorMailboxBytesOff, actorMemorySnapshotCurrentOffset); err != nil {
		return err
	}

	emitMovRaxFromR8Disp(e, actorMailboxPeakBytesOff)
	emitMovMem64RdxDispRax(e, actorMemorySnapshotMailboxPeakOffset)
	if err := emitStoreActorCounterPlusReservedStack(e, actorMailboxPeakBytesOff, actorMemorySnapshotPeakOffset); err != nil {
		return err
	}

	emitMovRaxFromR8Disp(e, actorBytesCopiedOff)
	emitMovMem64RdxDispRax(e, actorMemorySnapshotBytesCopiedOffset)
	if err := emitStoreActorCounterPlusReservedStack(e, actorByteBudgetOff, actorMemorySnapshotByteBudgetOffset); err != nil {
		return err
	}
	emitMovRaxFromR8Disp(e, actorOverBudgetCountOff)
	emitMovMem64RdxDispRax(e, actorMemorySnapshotOverBudgetOffset)
	emitMovRaxFromR8Disp(e, actorBackpressureEventsOff)
	emitMovMem64RdxDispRax(e, actorMemorySnapshotBackpressureOffset)
	if err := emitStoreActorLiveStackBytes(e, actorMemorySnapshotStackLiveOffset); err != nil {
		return err
	}
	if err := emitStoreActorReservedStackBytes(e, actorMemorySnapshotStackReservedOffset); err != nil {
		return err
	}
	if err := emitStoreActorRetainedStackBytes(e, actorMemorySnapshotStackRetainedOffset); err != nil {
		return err
	}
	if err := emitStoreActorReleasedStackBytes(e, actorMemorySnapshotStackReleasedOffset); err != nil {
		return err
	}

	e.AddEcxImm8(1)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, loopOff); err != nil {
		return err
	}

	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
		return err
	}
	e.MovRdxR9()
	e.MovEaxEcx()
	e.Ret()
	return nil
}

func emitStoreActorCounterPlusLiveStack(
	e *x64.Emitter,
	counterOff int32,
	recordOff int32,
) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	noStackAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	noStackReclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusFree)
	noStackFreeAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	noStackReleasedAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, counterOff)
	e.AddRaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	doneAt := e.JmpRel32()

	noStackOff := len(e.Buf)
	for _, at := range []int{noStackAt, noStackReclaimableAt, noStackFreeAt, noStackReleasedAt} {
		if err := x64.PatchRel32(e.Buf, at, noStackOff); err != nil {
			return err
		}
	}
	emitMovRaxFromR8Disp(e, counterOff)
	emitMovMem64RdxDispRax(e, recordOff)

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitStoreActorCounterPlusReservedStack(
	e *x64.Emitter,
	counterOff int32,
	recordOff int32,
) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusFree)
	noStackAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	noStackReleasedAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, counterOff)
	e.AddRaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	doneAt := e.JmpRel32()

	noStackOff := len(e.Buf)
	for _, at := range []int{noStackAt, noStackReleasedAt} {
		if err := x64.PatchRel32(e.Buf, at, noStackOff); err != nil {
			return err
		}
	}
	emitMovRaxFromR8Disp(e, counterOff)
	emitMovMem64RdxDispRax(e, recordOff)

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitStoreActorLiveStackBytes(e *x64.Emitter, recordOff int32) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	zeroDoneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	zeroReclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusFree)
	zeroFreeAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	zeroReleasedAt := e.JzRel32()
	e.MovEaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	doneAt := e.JmpRel32()

	zeroOff := len(e.Buf)
	for _, at := range []int{zeroDoneAt, zeroReclaimableAt, zeroFreeAt, zeroReleasedAt} {
		if err := x64.PatchRel32(e.Buf, at, zeroOff); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitStoreActorReservedStackBytes(e *x64.Emitter, recordOff int32) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusFree)
	zeroAt := e.JzRel32()
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	zeroReleasedAt := e.JzRel32()
	e.MovEaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	doneAt := e.JmpRel32()

	zeroOff := len(e.Buf)
	for _, at := range []int{zeroAt, zeroReleasedAt} {
		if err := x64.PatchRel32(e.Buf, at, zeroOff); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitStoreActorRetainedStackBytes(e *x64.Emitter, recordOff int32) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	retainedAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	retainedReclaimableAt := e.JzRel32()
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)
	doneAt := e.JmpRel32()

	retainedOff := len(e.Buf)
	for _, at := range []int{retainedAt, retainedReclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, retainedOff); err != nil {
			return err
		}
	}
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	zeroReleasedAt := e.JzRel32()
	e.MovEaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	retainedDoneAt := e.JmpRel32()

	zeroReleasedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, zeroReleasedAt, zeroReleasedOff); err != nil {
		return err
	}
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)

	doneOff := len(e.Buf)
	for _, at := range []int{doneAt, retainedDoneAt} {
		if err := x64.PatchRel32(e.Buf, at, doneOff); err != nil {
			return err
		}
	}
	return nil
}

func emitStoreActorReleasedStackBytes(e *x64.Emitter, recordOff int32) error {
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	nonDoneAt := e.JmpRel32()

	doneOff := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneOff); err != nil {
			return err
		}
	}
	emitMovRaxFromR8Disp(e, actorStackInitRspOff)
	e.TestRaxRax()
	releasedAt := e.JzRel32()
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)
	doneStoreAt := e.JmpRel32()

	releasedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, releasedAt, releasedOff); err != nil {
		return err
	}
	e.MovEaxImm32(stackSize)
	emitMovMem64RdxDispRax(e, recordOff)
	releasedDoneAt := e.JmpRel32()

	nonDoneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonDoneAt, nonDoneOff); err != nil {
		return err
	}
	e.XorEaxEax()
	emitMovMem64RdxDispRax(e, recordOff)

	finalDoneOff := len(e.Buf)
	for _, at := range []int{doneStoreAt, releasedDoneAt} {
		if err := x64.PatchRel32(e.Buf, at, finalDoneOff); err != nil {
			return err
		}
	}
	return nil
}

func emitMovRaxFromR8Disp(e *x64.Emitter, disp int32) {
	e.Emit(0x49, 0x8B, 0x80)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func emitMovMem64RdxDispRax(e *x64.Emitter, disp int32) {
	if disp == 0 {
		e.Emit(0x48, 0x89, 0x02)
		return
	}
	e.Emit(0x48, 0x89, 0x82)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
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

func emitMailboxByteBudgetCheckForReceiverInEcx(e *x64.Emitter) int {
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovRaxFromRdiDisp(actorMailboxBytesOff)
	e.AddRaxImm32(msgSize)
	e.MovR8FromRdiDisp(actorByteBudgetOff)
	e.CmpRaxR8()
	return e.JaRel32()
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

func emitDoneActorCheckForReceiverInEcx(e *x64.Emitter) []int {
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusStopping)
	stoppingAt := e.JzRel32()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.PushRcx()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	notCanceledAt := e.JnzRel32()
	e.PopRcx()
	canceledAt := e.JmpRel32()

	notCanceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notCanceledAt, notCanceledTo); err != nil {
		panic(err)
	}
	e.PopRcx()
	noGroupTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, noGroupTo); err != nil {
		panic(err)
	}
	return []int{doneAt, reclaimableAt, stoppingAt, canceledAt}
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

func emitTrimDoneActorStacks(e *x64.Emitter, sysMunmap uint32) error {
	e.XorEaxEax() // actor slot index
	e.XorEdxEdx() // retained done-stack count

	loopOff := len(e.Buf)
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	doneAt := e.JaeRel32()

	e.PushRax()
	e.PushRdx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneStackAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableStackAt := e.JzRel32()
	nextAts := []int{e.JmpRel32()}

	stackCandidateOff := len(e.Buf)
	for _, at := range []int{doneStackAt, reclaimableStackAt} {
		if err := x64.PatchRel32(e.Buf, at, stackCandidateOff); err != nil {
			return err
		}
	}

	e.MovRaxFromRdiDisp(actorStackInitRspOff)
	e.TestRaxRax()
	nextAts = append(nextAts, e.JzRel32())

	e.MovEdxFromRspDisp(0)
	e.CmpEdxImm32(maxRetainedDoneActorStacks)
	releaseAt := e.JaeRel32()
	e.AddEdxImm32(1)
	e.MovEaxEdx()
	e.MovMem32RspDispEax(0)
	retainedAt := e.JmpRel32()

	releaseOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, releaseAt, releaseOff); err != nil {
		return err
	}
	if err := emitReleaseDoneActorStack(e, sysMunmap); err != nil {
		return err
	}

	retainedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, retainedAt, retainedOff); err != nil {
		return err
	}
	for _, at := range nextAts {
		if err := x64.PatchRel32(e.Buf, at, retainedOff); err != nil {
			return err
		}
	}

	e.PopRdx()
	e.PopRax()
	e.AddEaxImm32(1)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, loopOff); err != nil {
		return err
	}

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitReleaseDoneActorStack(e *x64.Emitter, sysMunmap uint32) error {
	e.PushRdi()
	e.AddRaxImm32(-stackSize + 56)
	e.MovRdiRax()
	e.MovEaxImm32(stackSize)
	e.MovRsiRax()
	e.MovEaxImm32(sysMunmap)
	e.Syscall()
	e.TestRaxRax()
	failedAt := e.JnzRel32()

	e.PopRdi()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorRspOff)
	e.MovMem64RdiDispRax(actorStackInitRspOff)
	doneAt := e.JmpRel32()

	failedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, failedAt, failedOff); err != nil {
		return err
	}
	e.PopRdi()

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}

func emitEntry(
	e *x64.Emitter,
	mainSymbol string,
	sysMmap uint32,
	sysMunmap uint32,
	mapFlags uint32,
	callPatches *[]callPatch,
	leaPatches *[]leaPatch,
) error {
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
	e.MovMem32RdiDispImm32(schedRuntimeClosingOff, 0)
	e.MovMem32RdiDispImm32(schedMonitorNextIDOff, 1)
	e.MovMem32RdiDispImm32(schedMonitorCountOff, 0)

	// Message pool
	emitMmapAnon(e, msgPoolSize, sysMmap, mapFlags)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedMsgBaseOff)
	e.MovMem64RdiDispRax(schedMsgBumpOff)
	e.AddRaxImm32(msgPoolSize)
	e.MovMem64RdiDispRax(schedMsgEndOff)
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedMsgFreeOff)
	emitInitSchedulerMessagePoolCounters(e)

	// Runtime-owned system event pool, physically separate from user messages.
	emitMmapAnon(e, systemEventPoolSize, sysMmap, mapFlags)
	e.MovRdiR15()
	e.MovMem64RdiDispRax(schedSystemEventBaseOff)
	e.MovMem64RdiDispRax(schedSystemEventBumpOff)
	e.AddRaxImm32(systemEventPoolSize)
	e.MovMem64RdiDispRax(schedSystemEventEndOff)
	e.XorEaxEax()
	e.MovMem64RdiDispRax(schedSystemEventFreeOff)
	emitInitSchedulerSystemEventPoolCounters(e)

	// actor0 = sched.actorsPtr + 0
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.MovRdxRax() // actor0 ptr in rdx
	e.MovRdiRdx()

	// actor0.status = ready
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	// actor0.entryID = hash(main symbol)
	e.MovMem32RdiDispImm32(actorEntryIDOff, int32(fnv1a32(mainSymbol)))
	e.MovMem32RdiDispImm32(actorGenerationOff, actorGenerationInitial)
	e.MovMem32RdiDispImm32(actorTrapExitOff, 0)
	e.MovMem32RdiDispImm32(actorLinkCountOff, 0)
	// actor0.mailbox = empty
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorLastSenderOff, 0)
	e.MovMem32RdiDispImm32(actorExitCodeOff, 0)
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	e.MovMem32RdiDispImm32(actorTaskGroupOff, 0)
	e.MovMem32RdiDispImm32(actorMailboxCountOff, 0)
	emitInitActorByteCountersInRdi(e)
	emitInitActorSystemMailboxInRdi(e)
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
	e.MovMem64RdiDispRax(actorStackInitRspOff)
	e.MovMem32RdiDispImm32(actorStatusOff, statusRunning)

	// Switch to actor0 to start execution.
	e.MovRdiR15()
	e.AddRdiImm32(schedRspOff)
	e.MovRaxRdx()
	e.MovRsiRax()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_switch_to"})

	// Scheduler loop.
	loopStart := len(e.Buf)
	if err := emitTrimDoneActorStacks(e, sysMunmap); err != nil {
		return err
	}

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
	e.CmpEaxImm32(statusStarting)
	startingAt := e.JzRel32()
	e.CmpEaxImm32(statusStopping)
	stoppingAt := e.JzRel32()
	e.CmpEaxImm32(statusSleeping)
	sleepingAt := e.JzRel32()
	e.CmpEaxImm32(statusBlocked)
	blockedAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	waitingAt := e.JzRel32()
	notReadyAt := e.JmpRel32()

	// Stop requests become terminal during the scheduler scan, after one
	// observable stopping window for lifecycle status queries.
	stoppingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, stoppingAt, stoppingTo); err != nil {
		return err
	}
	e.MovMem32RdiDispImm32(actorStatusOff, statusDone)
	stoppingDoneAt := e.JmpRel32()

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
	if err := x64.PatchRel32(e.Buf, startingAt, readyTo); err != nil {
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
	e.PushRax()
	e.MovRdiRax()
	e.MovMem32RdiDispImm32(actorStatusOff, statusRunning)
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
	if err := x64.PatchRel32(e.Buf, stoppingDoneAt, notReadyTo); err != nil {
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
	if err := emitMarkRuntimeClosingAndWakeSystemWaiters(e, loopStart); err != nil {
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

func emitMarkRuntimeClosingAndWakeSystemWaiters(e *x64.Emitter, loopStart int) error {
	e.MovRdiR15()
	e.MovMem32RdiDispImm32(schedRuntimeClosingOff, 1)
	e.MovEcxFromRdiDisp(schedCountOff)
	e.XorEaxEax()
	e.XorEdxEdx()

	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	doneAt := e.JaeRel32()
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
	e.CmpEaxImm32(statusBlocked)
	notSystemWaiterAt := e.JnzRel32()
	e.MovEaxFromRdiDisp(actorWaitKindOff)
	e.CmpEaxImm32(actorWaitKindSystem)
	notSystemWaitKindAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	e.AddEdxImm32(1)

	continueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notSystemWaiterAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notSystemWaitKindAt, continueTo); err != nil {
		return err
	}
	e.PopRcx()
	e.PopRax()
	e.AddEaxImm32(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.TestEdxEdx()
	noSystemWaitersAt := e.JzRel32()
	loopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, loopAt, loopStart); err != nil {
		return err
	}
	noSystemWaitersTo := len(e.Buf)
	return x64.PatchRel32(e.Buf, noSystemWaitersAt, noSystemWaitersTo)
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
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusRunning)
	notRunningAt := e.JnzRel32()
	e.MovMem32RdiDispImm32(actorStatusOff, statusReady)
	notRunningTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notRunningAt, notRunningTo); err != nil {
		return err
	}
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
	// actor.exitCode = edi; drain pending mailbox nodes; actor.status = done; yield.
	e.MovEdxEdi()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorExitCodeOff)
	emitDrainActorMailboxInRdi(e)
	emitDrainActorSystemMailboxInRdi(e)
	e.MovEdxFromRdiDisp(actorExitCodeOff)
	if err := emitDeliverMonitorDownFromActorPtrInRdiExitCodeInEdx(e); err != nil {
		return err
	}
	e.MovEdxFromRdiDisp(actorExitCodeOff)
	emitPropagateLinkedAbnormalExitFromActorPtrInRdiExitCodeInEdx(e)
	e.MovMem32RdiDispImm32(actorStatusOff, statusDone)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	// Should never resume.
	e.MovEaxImm32(0)
	e.Ret()
	return nil
}

func emitDrainActorMailboxInRdi(e *x64.Emitter) {
	e.PushRdi() // keep actorPtr stable while recycling nodes through scheduler state
	loopStart := len(e.Buf)
	e.MovRaxFromRdiDisp(actorMailboxHeadOff)
	e.TestRaxRax()
	doneAt := e.JzRel32()

	e.PushRax() // nodePtr; saved actorPtr remains at rsp+8
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(msgNextOff)
	emitMovRdiFromRspDisp(e, 8)
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.PushRax() // next
	emitDecrementMailboxCount(e)
	emitAccountMailboxDequeueInRdi(e)
	e.PopRax()
	e.TestRaxRax()
	keepTailAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	keepTailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepTailAt, keepTailTo); err != nil {
		panic(err)
	}
	e.PopRax()
	emitRecycleMessageNodeInRax(e)
	emitMovRdiFromRspDisp(e, 0)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, loopStart); err != nil {
		panic(err)
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorMailboxCountOff, 0)
	e.MovMem64RdiDispRax(actorMailboxBytesOff)
	e.PopRdi()
}

func emitDrainActorSystemMailboxInRdi(e *x64.Emitter) {
	e.PushRdi()
	loopStart := len(e.Buf)
	e.MovRaxFromRdiDisp(actorSystemMailboxHeadOff)
	e.TestRaxRax()
	doneAt := e.JzRel32()

	e.PushRax()
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(systemEventNextOff)
	emitMovRdiFromRspDisp(e, 8)
	e.MovMem64RdiDispRax(actorSystemMailboxHeadOff)
	e.PushRax()
	emitDecrementSystemMailboxCount(e)
	emitAccountSystemMailboxDequeueInRdi(e)
	e.PopRax()
	e.TestRaxRax()
	keepTailAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)
	keepTailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepTailAt, keepTailTo); err != nil {
		panic(err)
	}
	e.PopRax()
	emitRecycleSystemEventNodeInRax(e)
	emitMovRdiFromRspDisp(e, 0)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, loopStart); err != nil {
		panic(err)
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)
	e.MovMem32RdiDispImm32(actorSystemMailboxCountOff, 0)
	e.MovMem64RdiDispRax(actorSystemMailboxBytesOff)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	e.PopRdi()
}

func emitPropagateLinkedAbnormalExitFromActorPtrInRdiExitCodeInEdx(e *x64.Emitter) {
	e.MovEaxEdx()
	e.TestEaxEax()
	normalExitAt := e.JzRel32()
	e.MovR8dEdx()
	for i := 0; i < maxActorLinks; i++ {
		e.MovEaxFromRdiDisp(actorLinkCountOff)
		e.CmpEaxImm32(int32(i + 1))
		skipAt := e.JlRel32()

		e.PushRdi()
		e.MovEaxFromRdiDisp(actorLinkRef0Off + int32(i*4))
		e.PushRax()
		e.MovEaxFromRdiDisp(actorLinkHigh0Off + int32(i*4))
		e.CmpEaxImm32(actorRefLocalHighSlot)
		localHighAt := e.JzRel32()
		e.PopRax()
		e.PopRdi()
		remoteLinkDoneAt := e.JmpRel32()

		localHighTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, localHighAt, localHighTo); err != nil {
			panic(err)
		}
		e.MovEaxFromRspDisp(0)
		emitAndEaxImm32(e, 0xFFFF)
		e.MovEdxEax()
		e.MovEaxFromRspDisp(0)
		emitShrEaxImm8(e, 16)
		emitAndEaxImm32(e, 0xFFFF)
		actorPtrFromEaxToRdi(e)
		e.MovEaxFromRdiDisp(actorGenerationOff)
		e.CmpEaxEdx()
		staleAt := e.JnzRel32()
		e.MovEaxFromRdiDisp(actorStatusOff)
		e.CmpEaxImm32(statusDone)
		doneAt := e.JzRel32()
		e.MovEaxFromRdiDisp(actorTrapExitOff)
		e.TestEaxEax()
		trapAt := e.JnzRel32()
		e.MovRdxR8()
		e.MovEaxEdx()
		e.MovMem32RdiDispEax(actorExitCodeOff)
		emitReleaseSystemEventCreditForActorPtrInRdi(e)
		e.MovMem32RdiDispImm32(actorStatusOff, statusDone)
		killedAt := e.JmpRel32()

		trapTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, trapAt, trapTo); err != nil {
			panic(err)
		}
		emitDeliverLinkedTrapExitFromActorPtrInRdiExitCodeInR8d(e)

		slotDoneTo := len(e.Buf)
		for _, at := range []int{staleAt, doneAt, killedAt} {
			if err := x64.PatchRel32(e.Buf, at, slotDoneTo); err != nil {
				panic(err)
			}
		}
		e.PopRax()
		e.PopRdi()

		skipTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, remoteLinkDoneAt, skipTo); err != nil {
			panic(err)
		}
		if err := x64.PatchRel32(e.Buf, skipAt, skipTo); err != nil {
			panic(err)
		}
	}
	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, normalExitAt, doneTo); err != nil {
		panic(err)
	}
}

func emitDeliverLinkedTrapExitFromActorPtrInRdiExitCodeInR8d(e *x64.Emitter) {
	// Linked trap-exit messages are runtime-owned system events on a mailbox
	// physically separate from ordinary user messages.
	e.PushRdi() // trapped actor pointer
	e.PushR8()  // exit reason

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	e.PushRax()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEdxFromRdiDisp(actorGenerationOff)
	e.PopRax()
	emitOrEaxEdx(e)
	e.PushRax() // source actor low ref

	overflowAt := emitCheckedSystemEventPoolAlloc(e)

	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(systemEventNextOff)
	e.MovMem32RdiDispEax(systemEventFlagsOff)
	e.MovMem64RdiDispRax(systemEventMonitorRefOff)
	e.MovMem64RdiDispRax(systemEventNodeIDOff)
	e.MovMem64RdiDispRax(systemEventNodeEpochOff)
	e.MovMem64RdiDispRax(systemEventDedupKeyOff)
	e.MovMem32RdiDispImm32(systemEventKindOff, actorSystemKindExit)
	e.MovEaxFromRspDisp(0)
	e.MovMem64RdiDispRax(systemEventSubjectOff)
	e.MovEaxFromRspDisp(8)
	e.MovMem32RdiDispEax(systemEventReasonCodeOff)
	e.MovMem32RdiDispImm32(systemEventReasonKindOff, 0)

	e.MovRaxFromRspDisp(16)
	e.MovRdiRax()
	emitEnqueueSystemEventForActorPtrInRdiNodeInRdx(e)

	doneAt := e.JmpRel32()
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		panic(err)
	}
	emitSystemEventPoolAllocationFailure(e)

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		panic(err)
	}
	e.MovR8dFromRspDisp(8)
	e.AddRspImm32(24)
}

func emitDeliverMonitorDownFromActorPtrInRdiExitCodeInEdx(e *x64.Emitter) error {
	// Monitor DOWN messages use a runtime-only system lane. Ordinary user send
	// paths cannot set msg.system_kind, so later system-message receivers can
	// distinguish these events from user tags with the same payload values.
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	e.PushRax()
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEdxFromRdiDisp(actorGenerationOff)
	e.PopRax()
	emitOrEaxEdx(e)
	e.PushRax()
	e.PopR8() // target actor low ref

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedMonitorCountOff)
	e.TestEaxEax()
	var doneAts []int
	doneAts = append(doneAts, e.JzRel32())
	e.XorEcxEcx()

	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	doneAts = append(doneAts, e.JzRel32())
	e.PushRax() // monitor count
	e.PushRcx() // scan index
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorTargetHigh0Off)
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localTargetAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		return err
	}

	localTargetTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localTargetAt, localTargetTo); err != nil {
		return err
	}
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorTargetRef0Off)
	e.PushR8()
	e.PopRdx()
	e.CmpEaxEdx()
	foundAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, scanStart); err != nil {
		return err
	}

	foundTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, foundAt, foundTo); err != nil {
		return err
	}
	e.PopRcx()
	e.PopRax()
	e.PushRax() // count
	e.PushRcx() // index
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorOwnerRef0Off)
	e.PushRax() // owner ref
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorID0Off)
	e.PushRax() // monitor ref/id
	e.PushR8()  // target ref
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.PushRax() // exit reason
	// Stack: exit, target ref, monitor id, owner ref, index, count.

	e.MovEaxFromRspDisp(24)
	emitAndEaxImm32(e, 0xFFFF)
	e.MovEdxEax()
	e.MovEaxFromRspDisp(24)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorGenerationOff)
	e.CmpEaxEdx()
	var skipDeliveryAts []int
	skipDeliveryAts = append(skipDeliveryAts, e.JnzRel32())
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	skipDeliveryAts = append(skipDeliveryAts, e.JzRel32())

	overflowAt := emitCheckedSystemEventPoolAlloc(e)
	skipDeliveryAts = append(skipDeliveryAts, overflowAt)
	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(systemEventNextOff)
	e.MovMem32RdiDispEax(systemEventFlagsOff)
	e.MovMem64RdiDispRax(systemEventNodeIDOff)
	e.MovMem64RdiDispRax(systemEventNodeEpochOff)
	e.MovMem64RdiDispRax(systemEventDedupKeyOff)
	e.MovMem32RdiDispImm32(systemEventKindOff, actorSystemKindDown)
	e.MovEaxFromRspDisp(8)
	e.MovMem64RdiDispRax(systemEventSubjectOff)
	e.MovEaxFromRspDisp(16)
	e.MovMem64RdiDispRax(systemEventMonitorRefOff)
	e.MovMem32RdiDispImm32(systemEventReasonKindOff, 0)
	e.MovEaxFromRspDisp(0)
	e.MovMem32RdiDispEax(systemEventReasonCodeOff)

	e.MovEaxFromRspDisp(24)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	actorPtrFromEaxToRdi(e)
	emitEnqueueSystemEventForActorPtrInRdiNodeInRdx(e)

	skipDeliveryTo := len(e.Buf)
	for _, at := range skipDeliveryAts {
		if err := x64.PatchRel32(e.Buf, at, skipDeliveryTo); err != nil {
			return err
		}
	}

	e.MovEaxFromRspDisp(40)
	e.MovEcxFromRspDisp(32)
	e.AddEaxImm32(-1)
	e.PushRax()
	e.CmpEaxEcx()
	skipCopyAt := e.JzRel32()
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorID0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorOwnerRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetHigh0Off)
	skipCopyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCopyAt, skipCopyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedMonitorCountOff)
	e.MovEcxFromRspDisp(32)
	e.AddRspImm32(48)
	recheckAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, recheckAt, scanStart); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	for _, at := range doneAts {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEdxFromRdiDisp(actorExitCodeOff)
	return nil
}

func emitDeliverRemoteMonitorNodeDownFromRspFrame(e *x64.Emitter) error {
	// Remote monitor NODE_DOWN messages use the same runtime-only system lane
	// as local DOWN/EXIT. The frame currently lives at rsp in
	// __tetra_actor_net_pump; every local push in this helper adjusts frame
	// offsets explicitly.
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedMonitorCountOff)
	e.TestEaxEax()
	var doneAts []int
	doneAts = append(doneAts, e.JzRel32())
	e.XorEcxEcx()

	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	doneAts = append(doneAts, e.JzRel32())
	e.PushRax() // monitor count
	e.PushRcx() // scan index
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorTargetHigh0Off)
	e.PushRax() // target high
	emitAndEaxImm32(e, actorRefRemoteHighMask)
	e.CmpEaxImm32(actorRefRemoteHighBase)
	remoteTargetAt := e.JzRel32()
	e.PopRax()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		return err
	}

	remoteTargetTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, remoteTargetAt, remoteTargetTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(0)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0x07FF)
	e.PushRax()
	emitMovzxEaxWordRspDisp(e, actorWireOffsetSrc+24)
	e.PopRdx()
	e.CmpEaxEdx()
	foundAt := e.JzRel32()
	e.PopRax()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	againAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, againAt, scanStart); err != nil {
		return err
	}

	foundTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, foundAt, foundTo); err != nil {
		return err
	}
	// Stack: target high, index, count.
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorTargetRef0Off)
	e.PushRax() // target low
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorOwnerRef0Off)
	e.PushRax() // owner ref
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorID0Off)
	e.PushRax() // monitor ref/id
	e.MovEaxFromRspDisp(48 + actorWireOffsetStatus)
	e.PushRax() // status/reason
	// Stack: status, monitor id, owner ref, target low, target high, index, count.

	e.MovEaxFromRspDisp(16)
	emitAndEaxImm32(e, 0xFFFF)
	e.MovEdxEax()
	e.MovEaxFromRspDisp(16)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorGenerationOff)
	e.CmpEaxEdx()
	var skipDeliveryAts []int
	skipDeliveryAts = append(skipDeliveryAts, e.JnzRel32())
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	skipDeliveryAts = append(skipDeliveryAts, e.JzRel32())

	overflowAt := emitCheckedSystemEventPoolAlloc(e)
	skipDeliveryAts = append(skipDeliveryAts, overflowAt)
	e.MovRdiRdx()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(systemEventNextOff)
	e.MovMem32RdiDispEax(systemEventFlagsOff)
	e.MovMem64RdiDispRax(systemEventDedupKeyOff)
	e.MovMem32RdiDispImm32(systemEventKindOff, actorSystemKindNodeDown)
	e.MovEaxFromRspDisp(24)
	e.MovMem64RdiDispRax(systemEventSubjectOff)
	e.MovEaxFromRspDisp(8)
	e.MovMem64RdiDispRax(systemEventMonitorRefOff)
	e.MovEaxFromRspDisp(32)
	e.MovMem64RdiDispRax(systemEventNodeIDOff)
	e.MovEaxFromRspDisp(32)
	e.MovMem64RdiDispRax(systemEventNodeEpochOff)
	e.MovMem32RdiDispImm32(systemEventReasonKindOff, 0)
	e.MovEaxFromRspDisp(0)
	e.MovMem32RdiDispEax(systemEventReasonCodeOff)

	e.MovEaxFromRspDisp(16)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	actorPtrFromEaxToRdi(e)
	emitEnqueueSystemEventForActorPtrInRdiNodeInRdx(e)

	skipDeliveryTo := len(e.Buf)
	for _, at := range skipDeliveryAts {
		if err := x64.PatchRel32(e.Buf, at, skipDeliveryTo); err != nil {
			return err
		}
	}

	e.MovEaxFromRspDisp(48)
	e.MovEcxFromRspDisp(40)
	e.AddEaxImm32(-1)
	e.PushRax()
	e.CmpEaxEcx()
	skipCopyAt := e.JzRel32()
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorID0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorOwnerRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetHigh0Off)
	skipCopyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCopyAt, skipCopyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedMonitorCountOff)
	e.MovEcxFromRspDisp(40)
	e.AddRspImm32(56)
	recheckAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, recheckAt, scanStart); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	for _, at := range doneAts {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	return nil
}

func emitDeliverRemoteLinkedNodeDownFromRspFrame(e *x64.Emitter) error {
	// Remote links use node_down as a terminal link signal in this P01 runtime
	// foundation. The authenticated network producer arrives in P10/P11; this
	// helper consumes the existing checked net-pump node_down frame.
	for actorSlot := 0; actorSlot < maxActors; actorSlot++ {
		e.MovEaxImm32(uint32(actorSlot))
		actorPtrFromEaxToRdi(e)
		e.MovEaxFromRdiDisp(actorStatusOff)
		e.CmpEaxImm32(statusDone)
		actorDoneAt := e.JzRel32()
		e.XorEcxEcx()

		scanStart := len(e.Buf)
		e.MovEaxFromRdiDisp(actorLinkCountOff)
		e.CmpEaxEcx()
		linksDoneAt := e.JzRel32()

		e.PushRdi() // actor pointer
		e.PushRcx() // link index
		emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkHigh0Off)
		e.PushRax() // remote actor high slot
		emitAndEaxImm32(e, actorRefRemoteHighMask)
		e.CmpEaxImm32(actorRefRemoteHighBase)
		remoteLinkAt := e.JzRel32()
		if err := emitRemoteLinkedNodeDownScanAdvance(e, scanStart); err != nil {
			return err
		}

		remoteLinkTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, remoteLinkAt, remoteLinkTo); err != nil {
			return err
		}
		e.MovEaxFromRspDisp(0)
		emitShrEaxImm8(e, 16)
		emitAndEaxImm32(e, 0x07FF)
		e.PushRax()
		emitMovzxEaxWordRspDisp(e, actorWireOffsetSrc+32)
		e.PopRdx()
		e.CmpEaxEdx()
		nodeMatchAt := e.JzRel32()
		if err := emitRemoteLinkedNodeDownScanAdvance(e, scanStart); err != nil {
			return err
		}

		nodeMatchTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, nodeMatchAt, nodeMatchTo); err != nil {
			return err
		}
		// Stack: target high, link index, actor pointer.
		e.MovEcxFromRspDisp(8)
		emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkRef0Off)
		e.PushRax() // target low
		// Stack: target low, target high, link index, actor pointer.
		e.MovRaxFromRspDisp(24)
		e.MovRdiRax()
		e.MovEaxFromRdiDisp(actorTrapExitOff)
		e.TestEaxEax()
		trapExitAt := e.JnzRel32()

		emitLoadNodeDownStatusFromRspFrameDispToEax(e, actorWireOffsetStatus+32)
		e.MovMem32RdiDispEax(actorExitCodeOff)
		e.MovMem32RdiDispImm32(actorTerminalReasonKindOff, actorExitReasonKindNodeDown)
		e.MovMem32RdiDispEax(actorTerminalReasonCodeOff)
		e.MovEdxFromRspDisp(0)
		e.MovR9dFromRspDisp(8)
		emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9d(e)
		e.MovMem32RdiDispImm32(actorStatusOff, statusDone)
		handledAt := e.JmpRel32()

		trapExitTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, trapExitAt, trapExitTo); err != nil {
			return err
		}
		overflowAt := emitCheckedSystemEventPoolAlloc(e)

		e.MovRdiRdx()
		e.XorEaxEax()
		e.MovMem64RdiDispRax(systemEventNextOff)
		e.MovMem32RdiDispEax(systemEventFlagsOff)
		e.MovMem64RdiDispRax(systemEventMonitorRefOff)
		e.MovMem64RdiDispRax(systemEventDedupKeyOff)
		e.MovMem32RdiDispImm32(systemEventKindOff, actorSystemKindExit)
		e.MovEaxFromRspDisp(0)
		e.MovMem64RdiDispRax(systemEventSubjectOff)
		e.MovEaxFromRspDisp(8)
		emitShrEaxImm8(e, 16)
		emitAndEaxImm32(e, 0x07FF)
		e.MovMem64RdiDispRax(systemEventNodeIDOff)
		e.MovEaxFromRspDisp(8)
		emitAndEaxImm32(e, 0xFFFF)
		e.MovMem64RdiDispRax(systemEventNodeEpochOff)
		e.MovMem32RdiDispImm32(systemEventReasonKindOff, actorExitReasonKindNodeDown)
		emitLoadNodeDownStatusFromRspFrameDispToEax(e, actorWireOffsetStatus+32)
		e.MovMem32RdiDispEax(systemEventReasonCodeOff)

		e.MovRaxFromRspDisp(24)
		e.MovRdiRax()
		emitEnqueueSystemEventForActorPtrInRdiNodeInRdx(e)

		e.MovRaxFromRspDisp(24)
		e.MovRdiRax()
		e.MovEdxFromRspDisp(0)
		e.MovR9dFromRspDisp(8)
		emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9dWithoutCreditRelease(e)
		trapHandledAt := e.JmpRel32()

		overflowTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
			return err
		}
		emitSystemEventPoolAllocationFailure(e)
		e.PopRax() // target low
		e.PopRax() // target high
		e.PopRcx() // link index
		e.PopRdi() // actor pointer
		e.AddEcxImm8(1)
		overflowNextAt := e.JmpRel32()
		if err := x64.PatchRel32(e.Buf, overflowNextAt, scanStart); err != nil {
			return err
		}

		handledTo := len(e.Buf)
		for _, at := range []int{handledAt, trapHandledAt} {
			if err := x64.PatchRel32(e.Buf, at, handledTo); err != nil {
				return err
			}
		}
		e.MovEcxFromRspDisp(16)
		e.MovRaxFromRspDisp(24)
		e.MovRdiRax()
		e.AddRspImm32(32)
		recheckAt := e.JmpRel32()
		if err := x64.PatchRel32(e.Buf, recheckAt, scanStart); err != nil {
			return err
		}

		actorDoneTo := len(e.Buf)
		for _, at := range []int{actorDoneAt, linksDoneAt} {
			if err := x64.PatchRel32(e.Buf, at, actorDoneTo); err != nil {
				return err
			}
		}
	}
	return nil
}

func emitRemoteLinkedNodeDownScanAdvance(e *x64.Emitter, scanStart int) error {
	e.PopRax() // remote actor high slot
	e.PopRcx() // link index
	e.PopRdi() // actor pointer
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	return x64.PatchRel32(e.Buf, nextAt, scanStart)
}

func emitLoadNodeDownStatusFromRspFrameDispToEax(e *x64.Emitter, statusDisp int32) {
	e.MovEaxFromRspDisp(statusDisp)
	e.TestEaxEax()
	statusReadyAt := e.JnzRel32()
	e.MovEaxImm32(actorWireStatusDown)
	statusReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, statusReadyAt, statusReadyTo); err != nil {
		panic(err)
	}
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
	emitShrEaxImm8(e, 16)
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
	emitShrEaxImm8(e, 16)
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

func remoteGenerationPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedRemoteGeneration0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func remoteNodeEpochPtrFromEaxToRdi(e *x64.Emitter) {
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedNodeEpoch0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
}

func emitLocalActorRefReturnFromSlotInEaxActorPtrInRdi(e *x64.Emitter) {
	e.MovEcxFromRdiDisp(actorGenerationOff)
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	emitOrEaxEcx(e)
	e.MovEdxImm32(actorRefLocalHighSlot)
}

func emitAndEaxImm32(e *x64.Emitter, v uint32) {
	e.Emit(0x25)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func emitOrEaxImm32(e *x64.Emitter, v uint32) {
	e.Emit(0x0D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func emitOrEdxImm32(e *x64.Emitter, v uint32) {
	e.Emit(0x81, 0xCA)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func emitShrEaxImm8(e *x64.Emitter, v byte) {
	e.Emit(0xC1, 0xE8, v)
}

func emitOrEaxEdx(e *x64.Emitter) {
	e.Emit(0x09, 0xD0)
}

func emitOrEaxEcx(e *x64.Emitter) {
	e.Emit(0x09, 0xC8)
}

func emitOrEdxEax(e *x64.Emitter) {
	e.Emit(0x09, 0xC2)
}

func emitMovEsiR8d(e *x64.Emitter) {
	e.Emit(0x44, 0x89, 0xC6)
}

func emitMovR10dEdx(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0xD2)
}

func emitMovR11dEcx(e *x64.Emitter) {
	e.Emit(0x41, 0x89, 0xCB)
}

func emitDecodeLocalActorRefInRdiRsiToEcx(e *x64.Emitter) ([]int, error) {
	var invalidAts []int

	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	highOKAt := e.JzRel32()
	invalidAts = append(invalidAts, e.JmpRel32())
	highOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, highOKAt, highOKTo); err != nil {
		return nil, err
	}

	e.MovEaxEdi()
	e.Emit(0xC1, 0xE8, 0x10) // shr eax, 16
	e.CmpEaxImm32(maxActors)
	invalidAts = append(invalidAts, e.JaeRel32())
	e.MovEcxEax()

	e.MovEaxEdi()
	emitAndEaxImm32(e, 0xFFFF)
	e.MovEdxEax()
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorGenerationOff)
	e.CmpEaxEdx()
	invalidAts = append(invalidAts, e.JnzRel32())
	return invalidAts, nil
}

func emitDecodeLocalActorRefInRdiRsiToEcxClassified(e *x64.Emitter) ([]int, []int, error) {
	var invalidAts []int
	var staleAts []int

	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	highOKAt := e.JzRel32()
	invalidAts = append(invalidAts, e.JmpRel32())
	highOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, highOKAt, highOKTo); err != nil {
		return nil, nil, err
	}

	e.MovEaxEdi()
	e.Emit(0xC1, 0xE8, 0x10) // shr eax, 16
	e.CmpEaxImm32(maxActors)
	invalidAts = append(invalidAts, e.JaeRel32())
	e.MovEcxEax()

	e.MovEaxEdi()
	emitAndEaxImm32(e, 0xFFFF)
	e.MovEdxEax()
	e.MovEaxEcx()
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorGenerationOff)
	e.CmpEaxEdx()
	staleAts = append(staleAts, e.JnzRel32())
	return invalidAts, staleAts, nil
}

func emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e *x64.Emitter) ([]int, error) {
	var invalidAts []int

	e.MovEaxEsi()
	emitAndEaxImm32(e, actorRefRemoteHighMask)
	e.CmpEaxImm32(actorRefRemoteHighBase)
	highOKAt := e.JzRel32()
	invalidAts = append(invalidAts, e.JmpRel32())
	highOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, highOKAt, highOKTo); err != nil {
		return nil, err
	}

	e.MovEaxEsi()
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0x07FF)
	e.CmpEaxImm32(maxActors - 1)
	invalidAts = append(invalidAts, e.JaRel32())

	e.MovEaxEsi()
	emitAndEaxImm32(e, 0xFFFF)
	e.TestEaxEax()
	invalidAts = append(invalidAts, e.JzRel32())
	e.PushRax()
	e.MovEaxEsi()
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0x07FF)
	e.PushRdi()
	remoteNodeEpochPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(0)
	e.PopRdi()
	e.PopRdx()
	e.CmpEaxEdx()
	invalidAts = append(invalidAts, e.JnzRel32())

	e.MovEaxEdi()
	emitAndEaxImm32(e, 0xFFFF)
	e.TestEaxEax()
	invalidAts = append(invalidAts, e.JzRel32())
	e.MovEdxEax()
	e.MovEaxEdi()
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	e.CmpEaxImm32(maxActors - 1)
	invalidAts = append(invalidAts, e.JaRel32())
	e.PushRdi()
	remoteGenerationPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(0)
	e.PopRdi()
	e.CmpEaxEdx()
	invalidAts = append(invalidAts, e.JnzRel32())

	e.MovEaxEsi()
	emitAndEaxImm32(e, 0x07FF0000)
	emitOrEaxImm32(e, 0x80000000)
	e.MovEdxEax()
	e.MovEaxEdi()
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	emitOrEaxEdx(e)
	e.MovEdiEax()
	return invalidAts, nil
}

func emitInvalidActorWaitResultReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFD)
	e.MovEdxImm32(actorLifecycleStatusDead)
	e.Ret()
}

func emitStaleActorWaitResultReturn(e *x64.Emitter) {
	e.MovEaxImm32(0xFFFFFFFB)
	e.MovEdxImm32(actorLifecycleStatusDead)
	e.Ret()
}

func emitActorWaitDoneResultReturn(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.TestEaxEax()
	errorAt := e.JnzRel32()
	e.MovEdxImm32(actorLifecycleStatusExitedNormal)
	e.Ret()

	errorTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, errorAt, errorTo); err != nil {
		panic(err)
	}
	e.MovEdxImm32(actorLifecycleStatusExitedError)
	e.Ret()
}

func patchInvalidActorHandleReturns(e *x64.Emitter, invalidAts []int) error {
	invalidTo := len(e.Buf)
	for _, invalidAt := range invalidAts {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorHandleReturn(e)
	return nil
}

func patchInvalidActorWaitResultReturns(e *x64.Emitter, invalidAts []int) error {
	invalidTo := len(e.Buf)
	for _, invalidAt := range invalidAts {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorWaitResultReturn(e)
	return nil
}

func patchStaleActorWaitResultReturns(e *x64.Emitter, staleAts []int) error {
	staleTo := len(e.Buf)
	for _, staleAt := range staleAts {
		if err := x64.PatchRel32(e.Buf, staleAt, staleTo); err != nil {
			return err
		}
	}
	emitStaleActorWaitResultReturn(e)
	return nil
}

func emitActorStatusRawReturn(e *x64.Emitter, statusCode uint32, result uint32) {
	e.MovEaxImm32(statusCode)
	e.MovEdxImm32(result)
	e.Ret()
}

func patchActorStatusRawResultReturns(e *x64.Emitter, patchAts []int, result uint32) error {
	target := len(e.Buf)
	for _, at := range patchAts {
		if err := x64.PatchRel32(e.Buf, at, target); err != nil {
			return err
		}
	}
	emitActorStatusRawReturn(e, actorLifecycleStatusDead, result)
	return nil
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

// ---- linux_x64_emit_net.go ----

func emitFilesystemExists(e *x64.Emitter) error {
	const (
		linuxSysAccess = 21
		maxPathLen     = 4095
		pathBufSize    = 4096
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(pathBufSize)

	// Arguments: rdi=path_ptr, rsi=path_len, rdx=cap.io token.
	e.Emit(0x48, 0x85, 0xff) // test rdi, rdi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x85, 0xf6) // test esi, esi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x81, 0xfe, 0xff, 0x0f, 0x00, 0x00) // cmp esi, 4095
	failJumps = append(failJumps, e.JaRel32())

	e.Emit(0x48, 0x89, 0xf1)       // mov rcx, rsi
	e.Emit(0x49, 0x89, 0xf8)       // mov r8, rdi
	e.Emit(0x4c, 0x8d, 0x0c, 0x24) // lea r9, [rsp]
	e.XorEaxEax()                  // rax = copy index

	copyLoop := len(e.Buf)
	e.Emit(0x48, 0x39, 0xc8) // cmp rax, rcx
	copiedAt := e.JaeRel32()
	e.Emit(0x41, 0x8a, 0x14, 0x00) // mov dl, byte ptr [r8+rax]
	e.Emit(0x84, 0xd2)             // test dl, dl
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x41, 0x88, 0x14, 0x01) // mov byte ptr [r9+rax], dl
	e.Emit(0x48, 0xff, 0xc0)       // inc rax
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, copyLoop); err != nil {
		return err
	}

	copiedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copiedAt, copiedTo); err != nil {
		return err
	}
	e.Emit(0x41, 0xc6, 0x04, 0x09, 0x00) // mov byte ptr [r9+rcx], 0
	e.Emit(0x4c, 0x89, 0xcf)             // mov rdi, r9
	e.Emit(0x31, 0xf6)                   // xor esi, esi (F_OK)
	e.MovEaxImm32(linuxSysAccess)
	e.Syscall()
	e.TestEaxEax()
	e.SeteAl()
	e.MovzxEaxAl()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSocketTCP4(e *x64.Emitter) error {
	// Arguments: rdi=cap.io token (ignored).
	e.MovEdiImm32(2)            // AF_INET
	e.Emit(0xBE, 0x01, 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)          // xor edx, edx
	e.MovR10dImm32(0)
	e.MovR8dImm32(0)
	e.MovR9dImm32(0)
	e.MovEaxImm32(linuxSysSocket)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetBindTCP4Loopback(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=port, rdx=cap.io token (ignored).
	failJumps, err := emitNetRejectInvalidTCPPort(e)
	if err != nil {
		return err
	}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x89, 0x7C, 0x24, 0x10) // mov [rsp+16], edi

	emitMovMem16RspDispImm16(e, 0, 2)          // AF_INET
	e.MovEaxEsi()                              // port
	e.Emit(0x86, 0xE0)                         // xchg al, ah
	emitMovMem16RspDispAx(e, 2)                // sin_port
	emitMovMem32RspDispImm32(e, 4, 0x0100007f) // 127.0.0.1 bytes
	emitMovMem32RspDispImm32(e, 8, 0)          // sin_zero
	emitMovMem32RspDispImm32(e, 12, 0)         // sin_zero
	e.Emit(0x8B, 0x7C, 0x24, 0x10)             // mov edi, [rsp+16]
	e.Emit(0x48, 0x8D, 0x34, 0x24)             // lea rsi, [rsp]
	e.MovEdxImm32(16)                          // sizeof(sockaddr_in)
	e.MovEaxImm32(linuxSysBind)
	e.Syscall()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetConnectTCP4Loopback(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=port, rdx=cap.io token (ignored).
	failJumps, err := emitNetRejectInvalidTCPPort(e)
	if err != nil {
		return err
	}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x89, 0x7C, 0x24, 0x10) // mov [rsp+16], edi

	emitMovMem16RspDispImm16(e, 0, 2)          // AF_INET
	e.MovEaxEsi()                              // port
	e.Emit(0x86, 0xE0)                         // xchg al, ah
	emitMovMem16RspDispAx(e, 2)                // sin_port
	emitMovMem32RspDispImm32(e, 4, 0x0100007f) // 127.0.0.1 bytes
	emitMovMem32RspDispImm32(e, 8, 0)          // sin_zero
	emitMovMem32RspDispImm32(e, 12, 0)         // sin_zero
	e.Emit(0x8B, 0x7C, 0x24, 0x10)             // mov edi, [rsp+16]
	e.Emit(0x48, 0x8D, 0x34, 0x24)             // lea rsi, [rsp]
	e.MovEdxImm32(16)                          // sizeof(sockaddr_in)
	e.MovEaxImm32(linuxSysConnect)
	e.Syscall()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetRejectInvalidTCPPort(e *x64.Emitter) ([]int, error) {
	var failJumps []int
	e.Emit(0x85, 0xF6) // test esi, esi
	nonNegativeAt := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return nil, err
	}
	e.Emit(0x81, 0xFE, 0xFF, 0xFF, 0x00, 0x00) // cmp esi, 65535
	failJumps = append(failJumps, e.JaRel32())
	return failJumps, nil
}

func emitNetListen(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=backlog, rdx=cap.io token (ignored).
	e.MovEaxImm32(linuxSysListen)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetAccept4(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=flags, rdx=cap.io token (ignored).
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x31, 0xF6)       // xor esi, esi (addr=NULL)
	e.Emit(0x31, 0xD2)       // xor edx, edx (addrlen=NULL)
	e.MovEaxImm32(linuxSysAccept4)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetRead(e *x64.Emitter) error {
	return emitNetReadWrite(e, linuxSysRead)
}

func emitNetRecv(e *x64.Emitter) error {
	return emitNetRecvSend(e, linuxSysRecvfrom)
}

func emitNetWrite(e *x64.Emitter) error {
	return emitNetReadWrite(e, linuxSysWrite)
}

func emitNetSend(e *x64.Emitter) error {
	return emitNetRecvSend(e, linuxSysSendto)
}

func emitNetReadWrite(e *x64.Emitter, syscall uint32) error {
	var failJumps []int

	// Arguments: rdi=fd, rsi=slice_ptr, rdx=slice_len, rcx=start, r8=count, r9=cap.io token
	// (ignored).
	e.Emit(0x85, 0xC9) // test ecx, ecx
	startOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startOK, startOKTo); err != nil {
		return err
	}
	e.Emit(0x45, 0x85, 0xC0) // test r8d, r8d
	countOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	countOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, countOK, countOKTo); err != nil {
		return err
	}
	e.Emit(0x39, 0xCA) // cmp edx, ecx
	startInRange := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startInRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startInRange, startInRangeTo); err != nil {
		return err
	}

	e.Emit(0x29, 0xCA)       // sub edx, ecx (available = len - start)
	e.Emit(0x44, 0x39, 0xC2) // cmp edx, r8d
	useRequestedCount := e.JgeRel32()
	e.Emit(0x41, 0x89, 0xD0) // mov r8d, edx
	useRequestedCountTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useRequestedCount, useRequestedCountTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetRecvSend(e *x64.Emitter, syscall uint32) error {
	var failJumps []int

	// Arguments: rdi=fd, rsi=slice_ptr, rdx=slice_len, rcx=start, r8=count, r9=cap.io token
	// (ignored).
	// Emits recvfrom/sendto with flags=0 and NULL address operands.
	e.Emit(0x85, 0xC9) // test ecx, ecx
	startOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startOK, startOKTo); err != nil {
		return err
	}
	e.Emit(0x45, 0x85, 0xC0) // test r8d, r8d
	countOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	countOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, countOK, countOKTo); err != nil {
		return err
	}
	e.Emit(0x39, 0xCA) // cmp edx, ecx
	startInRange := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startInRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startInRange, startInRangeTo); err != nil {
		return err
	}

	e.Emit(0x29, 0xCA)       // sub edx, ecx (available = len - start)
	e.Emit(0x44, 0x39, 0xC2) // cmp edx, r8d
	useRequestedCount := e.JgeRel32()
	e.Emit(0x41, 0x89, 0xD0) // mov r8d, edx
	useRequestedCountTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useRequestedCount, useRequestedCountTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovR10dImm32(0) // flags=0
	e.MovR8dImm32(0)  // addr=NULL
	e.MovR9dImm32(0)  // addrlen=NULL
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetEpollCreate(e *x64.Emitter) error {
	// Arguments: rdi=cap.io token (ignored).
	e.MovEdiImm32(0) // flags=0
	e.MovEaxImm32(linuxSysEpollCreate1)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetEpollCtlAddRead(e *x64.Emitter) error {
	const (
		epollCtlAdd = 1
		epollIn     = 1
	)
	return emitNetEpollCtl(e, epollCtlAdd, epollIn)
}

func emitNetEpollCtlAddReadWrite(e *x64.Emitter) error {
	const (
		epollCtlAdd = 1
		epollIn     = 1
		epollOut    = 4
	)
	return emitNetEpollCtl(e, epollCtlAdd, epollIn|epollOut)
}

func emitNetEpollCtlModRead(e *x64.Emitter) error {
	const (
		epollCtlMod = 3
		epollIn     = 1
	)
	return emitNetEpollCtl(e, epollCtlMod, epollIn)
}

func emitNetEpollCtlModReadWrite(e *x64.Emitter) error {
	const (
		epollCtlMod = 3
		epollIn     = 1
		epollOut    = 4
	)
	return emitNetEpollCtl(e, epollCtlMod, epollIn|epollOut)
}

func emitNetEpollCtlDelete(e *x64.Emitter) error {
	const epollCtlDel = 2
	return emitNetEpollCtl(e, epollCtlDel, 0)
}

func emitNetEpollCtl(e *x64.Emitter, op uint32, events uint32) error {
	// Arguments: rdi=epfd, rsi=fd, rdx=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	emitMovMem32RspDispImm32(e, 0, events)
	e.Emit(
		0x48,
		0x89,
		0x74,
		0x24,
		0x04,
	) // mov [rsp+4], rsi (event.data.u64)
	e.Emit(
		0x48,
		0x89,
		0xF2,
	) // mov rdx, rsi (fd)
	e.Emit(
		0xBE,
		byte(op&0xff),
		byte((op>>8)&0xff),
		byte((op>>16)&0xff),
		byte((op>>24)&0xff),
	) // mov esi, op
	e.Emit(
		0x49,
		0x89,
		0xE2,
	) // mov r10, rsp
	e.MovEaxImm32(linuxSysEpollCtl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetEpollWaitOne(e *x64.Emitter) error {
	// Arguments: rdi=epfd, rsi=timeout_ms, rdx=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x48, 0x89, 0xE6) // mov rsi, rsp
	e.MovEdxImm32(1)         // maxevents=1
	e.MovEaxImm32(linuxSysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return err
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	e.Emit(0x8B, 0x44, 0x24, 0x04) // mov eax, [rsp+4] (event.data lower i32)
	e.Leave()
	e.Ret()
	return nil
}

func emitNetEpollWaitOneInto(e *x64.Emitter) error {
	// Arguments: rdi=epfd, rsi=[]i32 ptr, rdx=[]i32 len, rcx=timeout_ms, r8=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x83, 0xFA, 0x02) // cmp edx, 2
	lenOKAt := e.JgeRel32()
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()

	lenOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, lenOKAt, lenOKTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x89, 0x74, 0x24, 0x10) // mov [rsp+16], rsi (out ptr)
	e.Emit(0x41, 0x89, 0xCA)             // mov r10d, ecx (timeout_ms)
	e.Emit(0x48, 0x89, 0xE6)             // mov rsi, rsp (events)
	e.MovEdxImm32(1)                     // maxevents=1
	e.MovEaxImm32(linuxSysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return err
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x8B, 0x54, 0x24, 0x10) // mov rdx, [rsp+16] (out ptr)
	e.MovEaxFromRspDisp(4)               // event.data lower i32
	e.Emit(0x89, 0x02)                   // mov [rdx], eax
	e.MovEaxFromRspDisp(0)               // event.events
	e.Emit(0x89, 0x42, 0x04)             // mov [rdx+4], eax
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSetNonblocking(e *x64.Emitter) error {
	const (
		linuxFGetFL    = 3
		linuxFSetFL    = 4
		linuxONonblock = 2048
	)

	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x89, 0x3C, 0x24) // mov [rsp], edi

	e.Emit(0xBE, byte(linuxFGetFL), 0, 0, 0) // mov esi, F_GETFL
	e.Emit(0x31, 0xD2)                       // xor edx, edx
	e.MovEaxImm32(linuxSysFcntl)
	e.Syscall()
	e.TestEaxEax()
	okAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}
	e.Emit(
		0x0D,
		byte(linuxONonblock&0xff),
		byte((linuxONonblock>>8)&0xff),
		byte((linuxONonblock>>16)&0xff),
		byte((linuxONonblock>>24)&0xff),
	) // or eax, O_NONBLOCK
	e.Emit(
		0x89,
		0xC2,
	) // mov edx, eax
	e.Emit(
		0x8B,
		0x3C,
		0x24,
	) // mov edi, [rsp]
	e.Emit(
		0xBE,
		byte(linuxFSetFL),
		0,
		0,
		0,
	) // mov esi, F_SETFL
	e.MovEaxImm32(linuxSysFcntl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSetReusePort(e *x64.Emitter) error {
	const (
		linuxSolSocket   = 1
		linuxSoReusePort = 15
	)
	return emitNetSetIntSockOpt(e, linuxSolSocket, linuxSoReusePort)
}

func emitNetSetTCPNoDelay(e *x64.Emitter) error {
	const (
		linuxIPProtoTCP = 6
		linuxTCPNoDelay = 1
	)
	return emitNetSetIntSockOpt(e, linuxIPProtoTCP, linuxTCPNoDelay)
}

func emitNetSetIntSockOpt(e *x64.Emitter, level uint32, optname uint32) error {
	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	emitMovMem32RspDispImm32(e, 0, 1)
	e.Emit(
		0xBE,
		byte(level&0xff),
		byte((level>>8)&0xff),
		byte((level>>16)&0xff),
		byte((level>>24)&0xff),
	) // mov esi, level
	e.MovEdxImm32(optname)
	e.Emit(0x49, 0x89, 0xE2) // mov r10, rsp (optval=&one)
	e.MovR8dImm32(4)         // optlen=sizeof(i32)
	e.MovR9dImm32(0)
	e.MovEaxImm32(linuxSysSetSockOpt)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetClose(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()
	e.Ret()
	return nil
}

func emitActorNodeConnect(e *x64.Emitter) error {
	var failReturnJumps []int
	var failCloseJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(128)
	e.Emit(0x89, 0x7C, 0x24, 0x70) // node id spill
	e.Emit(0x89, 0x74, 0x24, 0x74) // port spill

	e.CmpEdiImm32(1)
	nodeLowOK := e.JgeRel32()
	failReturnJumps = append(failReturnJumps, e.JmpRel32())
	nodeLowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeLowOK, nodeLowTo); err != nil {
		return err
	}
	e.CmpEdiImm32(maxActors - 1)
	failReturnJumps = append(failReturnJumps, e.JaRel32())

	e.MovEdiImm32(2)
	e.Emit(0xBE, 0x01, 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)          // xor edx, edx
	e.MovEaxImm32(linuxSysSocket)
	e.Syscall()
	e.TestEaxEax()
	socketOK := e.JgeRel32()
	failReturnJumps = append(failReturnJumps, e.JmpRel32())
	socketOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, socketOK, socketOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x78) // fd spill

	emitMovMem16RspDispImm16(e, 0, 2)
	e.MovEaxFromRspDisp(0x74)
	e.Emit(0x86, 0xE0) // xchg al, ah
	emitMovMem16RspDispAx(e, 2)
	emitMovMem32RspDispImm32(e, 4, 0x0100007f)
	e.Emit(0x48, 0xC7, 0x44, 0x24, 0x08, 0, 0, 0, 0)

	e.Emit(0x8B, 0x7C, 0x24, 0x78) // fd
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(16)
	e.MovEaxImm32(linuxSysConnect)
	e.Syscall()
	e.TestEaxEax()
	connectOK := e.JgeRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	connectOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, connectOK, connectOKTo); err != nil {
		return err
	}

	emitActorWireControlFrame(e, 0x20, actorWireFrameHello)
	e.MovEaxFromRspDisp(0x70)
	emitMovMem16RspDispAx(e, 0x20+actorWireOffsetSrc)
	emitMovMem16RspDispAx(e, 0x20+actorWireOffsetDest)
	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	emitLeaRsiRspDisp(e, 0x20)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	emitLeaRsiRspDisp(e, 0x20)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysRead)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	readOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	readOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readOK, readOKTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, 0x20+actorWireOffsetMagic)
	e.CmpEaxImm32(actorWireMagic)
	ackMagicOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackMagicOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackMagicOK, ackMagicOKTo); err != nil {
		return err
	}
	emitMovzxEaxWordRspDisp(e, 0x20+actorWireOffsetType)
	e.CmpEaxImm32(actorWireFrameHelloAck)
	ackTypeOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackTypeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackTypeOK, ackTypeOKTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, 0x20+actorWireOffsetStatus)
	e.TestEaxEax()
	ackStatusOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackStatusOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackStatusOK, ackStatusOKTo); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRspDisp(0x78)
	e.MovMem32RdiDispEax(schedNetFDOff)
	e.MovEaxFromRspDisp(0x70)
	e.MovMem32RdiDispEax(schedNodeIDOff)
	emitMovEaxRspDisp(e, 0x20+actorWireOffsetValue)
	emitAndEaxImm32(e, 0xFFFF)
	e.TestEaxEax()
	ackEpochOK := e.JnzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackEpochOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackEpochOK, ackEpochOKTo); err != nil {
		return err
	}
	e.PushRax()
	e.MovEaxFromRspDisp(0x78)
	remoteNodeEpochPtrFromEaxToRdi(e)
	e.PopRax()
	e.MovMem32RdiDispEax(0)
	e.MovRdiR15()
	e.MovMem32RdiDispImm32(schedNetStatusOff, 0)
	e.XorEaxEax()
	e.Leave()
	e.Ret()

	failCloseTo := len(e.Buf)
	for _, at := range failCloseJumps {
		if err := x64.PatchRel32(e.Buf, at, failCloseTo); err != nil {
			return err
		}
	}
	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()

	failReturnTo := len(e.Buf)
	for _, at := range failReturnJumps {
		if err := x64.PatchRel32(e.Buf, at, failReturnTo); err != nil {
			return err
		}
	}
	e.MovRdiR15()
	e.MovMem32RdiDispImm32(schedNetStatusOff, 1)
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitActorSpawnRemote(e *x64.Emitter) error {
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.Emit(0x48, 0x83, 0xEC, 0x70) // sub rsp, 112
	e.Emit(0x89, 0x7C, 0x24, 0x60) // remote node
	e.Emit(0x89, 0x74, 0x24, 0x64) // entry id

	e.CmpEdiImm32(1)
	nodeLowOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	nodeLowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeLowOK, nodeLowTo); err != nil {
		return err
	}
	e.CmpEdiImm32(maxActors - 1)
	failJumps = append(failJumps, e.JaRel32())

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x68) // fd

	emitActorWireControlFrame(e, 0, actorWireFrameSpawn)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNodeIDOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSrc)
	e.MovEaxFromRspDisp(0x60)
	emitMovMem16RspDispAx(e, actorWireOffsetDest)
	e.MovEaxFromRspDisp(0x64)
	e.Emit(0x89, 0x44, 0x24, actorWireOffsetTag)

	e.Emit(0x8B, 0x7C, 0x24, 0x68)
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

	e.Emit(0x8B, 0x7C, 0x24, 0x68)
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysRead)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	readOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	readOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readOK, readOKTo); err != nil {
		return err
	}

	emitMovEaxRspDisp(e, actorWireOffsetMagic)
	e.CmpEaxImm32(actorWireMagic)
	ackMagicOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	ackMagicOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackMagicOK, ackMagicOKTo); err != nil {
		return err
	}
	emitMovzxEaxWordRspDisp(e, actorWireOffsetType)
	e.CmpEaxImm32(actorWireFrameSpawnAck)
	ackTypeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	ackTypeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackTypeOK, ackTypeOKTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, actorWireOffsetStatus)
	e.TestEaxEax()
	ackStatusOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	ackStatusOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackStatusOK, ackStatusOKTo); err != nil {
		return err
	}

	emitMovzxEaxWordRspDisp(e, actorWireOffsetActor)
	e.CmpEaxImm32(maxActors - 1)
	failJumps = append(failJumps, e.JaRel32())
	e.PushRax()
	remoteGenerationPtrFromEaxToRdi(e)
	emitMovEaxRspDisp(e, actorWireOffsetValue+8)
	emitAndEaxImm32(e, 0xFFFF)
	e.TestEaxEax()
	ackGenerationOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	ackGenerationOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackGenerationOK, ackGenerationOKTo); err != nil {
		return err
	}
	e.MovMem32RdiDispEax(0)
	e.MovEcxEax()
	e.PopRax()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	emitOrEaxEcx(e)
	e.PushRax()
	emitMovEaxRspDisp(e, actorWireOffsetValue+4+8)
	emitAndEaxImm32(e, 0xFFFF)
	e.TestEaxEax()
	ackNodeEpochOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	ackNodeEpochOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackNodeEpochOK, ackNodeEpochOKTo); err != nil {
		return err
	}
	e.PushRax()
	e.MovEaxFromRspDisp(0x70)
	remoteNodeEpochPtrFromEaxToRdi(e)
	e.PopRax()
	e.MovMem32RdiDispEax(0)
	e.MovEdxFromRspDisp(0x68)
	e.Emit(0xC1, 0xE2, 0x10) // shl edx, 16
	emitOrEdxImm32(e, actorRefRemoteHighBase)
	e.MovEaxFromRspDisp(0x68)
	remoteNodeEpochPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(0)
	emitAndEaxImm32(e, 0xFFFF)
	emitOrEdxEax(e)
	e.PopRax()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.XorEdxEdx()
	e.Leave()
	e.Ret()
	return nil
}

func emitActorNodeStatus(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	connectedAt := e.JnzRel32()
	e.MovEaxImm32(1)
	e.Ret()
	connectedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, connectedAt, connectedTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(schedNetStatusOff)
	e.Ret()
	return nil
}

func emitActorWireControlFrame(e *x64.Emitter, base byte, frameType uint16) {
	emitMovMem32RspDispImm32(e, base+actorWireOffsetMagic, actorWireMagic)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetVer, actorWireVersion)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetType, frameType)
	emitMovMem32RspDispImm32(e, base+actorWireOffsetSeq, 0)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetActor, 0)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetSlots, 0)
	emitMovMem32RspDispImm32(e, base+actorWireOffsetTag, 0)
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
	e.CmpEaxImm32(statusReclaimable)
	reclaimableActorAt := e.JzRel32()

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
	if err := x64.PatchRel32(e.Buf, reclaimableActorAt, continueTo); err != nil {
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
	emitShrEaxImm8(e, 16)
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

// ---- linux_x64_emit_send.go ----

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

func emitSpawn(
	e *x64.Emitter,
	sysMmap uint32,
	mapFlags uint32,
	callPatches *[]callPatch,
	leaPatches *[]leaPatch,
) error {
	// Argument: entryID in edi.
	// Returns: actor ref in edx:eax.
	e.MovEdxEdi() // entryID -> edx

	e.MovEaxImm32(1)
	reuseScanStart := len(e.Buf)
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	noReusableAt := e.JaeRel32()
	e.PushRax() // candidate slot
	actorPtrFromEaxToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	skipCandidateAts := []int{e.JmpRel32()}

	reclaimableTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, reclaimableAt, reclaimableTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorGenerationOff)
	e.CmpEaxImm32(actorGenerationMax)
	skipCandidateAts = append(skipCandidateAts, e.JaeRel32())
	e.PopRax()
	e.PushRax()
	e.PushRdx()
	actorPtrFromEaxToRdi(e)
	if err := emitSpawnSlotInit(e, sysMmap, mapFlags, leaPatches, true); err != nil {
		return err
	}

	skipCandidateTo := len(e.Buf)
	for _, at := range skipCandidateAts {
		if err := x64.PatchRel32(e.Buf, at, skipCandidateTo); err != nil {
			return err
		}
	}
	e.PopRax()
	e.AddEaxImm32(1)
	retryReuseAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, retryReuseAt, reuseScanStart); err != nil {
		return err
	}

	noReusableTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noReusableAt, noReusableTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCapacityOff)
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	notFullAt := e.JnzRel32()
	// full -> return -1
	e.MovEaxImm32(uint32(^uint32(0)))
	e.XorEdxEdx()
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

	return emitSpawnSlotInit(e, sysMmap, mapFlags, leaPatches, false)
}

func emitSpawnSlotInit(
	e *x64.Emitter,
	sysMmap uint32,
	mapFlags uint32,
	leaPatches *[]leaPatch,
	reuseGeneration bool,
) error {
	// Precondition: rdi=actorPtr, stack top=entryID, next stack slot=actor index.
	e.MovMem32RdiDispImm32(actorStatusOff, statusStarting)

	// actor.entryID = entryID (saved)
	e.PopRdx()
	e.MovEaxEdx()
	e.MovMem32RdiDispEax(actorEntryIDOff)
	if reuseGeneration {
		e.MovEaxFromRdiDisp(actorGenerationOff)
		e.AddEaxImm32(1)
		e.MovMem32RdiDispEax(actorGenerationOff)
	} else {
		e.MovMem32RdiDispImm32(actorGenerationOff, actorGenerationInitial)
	}
	e.MovMem32RdiDispImm32(actorTrapExitOff, 0)
	e.MovMem32RdiDispImm32(actorLinkCountOff, 0)

	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorMailboxHeadOff)
	e.MovMem64RdiDispRax(actorMailboxTailOff)
	e.MovMem32RdiDispImm32(actorLastSenderOff, 0)
	e.MovMem32RdiDispImm32(actorExitCodeOff, 0)
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	e.MovMem32RdiDispImm32(actorTaskGroupOff, 0)
	e.MovMem32RdiDispImm32(actorMailboxCountOff, 0)
	emitInitActorByteCountersInRdi(e)
	emitInitActorSystemMailboxInRdi(e)

	// Save actorPtr across stack init.
	e.MovRaxRdi()
	e.PushRax()

	// Stack init (initRsp -> rcx).
	if reuseGeneration {
		e.MovRaxFromRdiDisp(actorStackInitRspOff)
		e.TestRaxRax()
		allocateStackAt := e.JzRel32()
		e.MovRcxRax()
		if err := emitFillActorStackFrameInRcx(e, leaPatches); err != nil {
			return err
		}
		stackReadyAt := e.JmpRel32()
		allocateStackTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, allocateStackAt, allocateStackTo); err != nil {
			return err
		}
		if err := emitInitActorStack(e, sysMmap, mapFlags, leaPatches); err != nil {
			return err
		}
		stackReadyTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, stackReadyAt, stackReadyTo); err != nil {
			return err
		}
	} else {
		if err := emitInitActorStack(e, sysMmap, mapFlags, leaPatches); err != nil {
			return err
		}
	}

	// Restore actorPtr.
	e.PopRax()
	e.MovRdiRax()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorRspOff)
	e.MovMem64RdiDispRax(actorStackInitRspOff)

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

	// return local actor ref for newIdx
	e.PopRax()
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.PopRax()
	emitLocalActorRefReturnFromSlotInEaxActorPtrInRdi(e)
	e.Ret()
	return nil
}

func emitSend(e *x64.Emitter) error {
	// Args: rdi=to.low, rsi=to.high, rdx=value (i32)
	// Returns: eax=value.

	e.MovR8dEdx()
	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	emitMovEsiR8d(e)
	if err := emitRemoteSendI32(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	emitMovEsiR8d(e)

	doneAts := emitDoneActorCheckForReceiverInEcx(e)
	budgetAt := emitMailboxByteBudgetCheckForReceiverInEcx(e)
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
	e.MovMem32RdiDispImm32(msgSystemKindOff, actorSystemKindUser)
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
	emitAccountMailboxEnqueueInRdi(e)

	emitWakeActorIfBlockedOnUserMailboxInRdi(e)

	e.MovEaxEsi()
	e.Ret()
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	emitMessagePoolAllocationFailure(e)
	emitMessagePoolExhaustedReturn(e)
	budgetTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, budgetAt, budgetTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	for _, invalidAt := range append(remoteInvalidAts, invalidAts...) {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	for _, doneAt := range doneAts {
		if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
			return err
		}
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
	// Args: rdi=to.low, rsi=to.high, rdx=value (i32), rcx=tag (i32)
	// Returns: eax=value.

	e.MovR8dEdx()
	e.MovR9Rcx()
	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	emitMovEsiR8d(e)
	e.PushR9()
	e.PopRdx()
	if err := emitRemoteSendMsg(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	emitMovEsiR8d(e)

	doneAts := emitDoneActorCheckForReceiverInEcx(e)
	budgetAt := emitMailboxByteBudgetCheckForReceiverInEcx(e)
	fullAt := emitMailboxFullCheckForReceiverInEcx(e)
	e.PushR9() // preserve tag across scheduler/actor pointer loads

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
	e.MovMem32RdiDispImm32(msgSystemKindOff, actorSystemKindUser)
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
	emitAccountMailboxEnqueueInRdi(e)

	emitWakeActorIfBlockedOnUserMailboxInRdi(e)

	e.MovEaxEsi()
	e.Ret()
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	e.PopRax()
	emitMessagePoolAllocationFailure(e)
	emitMessagePoolExhaustedReturn(e)
	budgetTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, budgetAt, budgetTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	for _, invalidAt := range append(remoteInvalidAts, invalidAts...) {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	for _, doneAt := range doneAts {
		if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
			return err
		}
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
	// Args: rdi=to.low, rsi=to.high, rdx=tag, rcx=payload slot count.
	emitMovR10dEdx(e)
	emitMovR11dEcx(e)
	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	e.PushR10()
	e.PopRsi()
	e.PushR11()
	e.PopRdx()
	if err := emitRemoteSendBegin(e); err != nil {
		return err
	}
	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}

	doneAts := emitDoneActorCheckForReceiverInEcx(e)
	budgetAt := emitMailboxByteBudgetCheckForReceiverInEcx(e)
	fullAt := emitMailboxFullCheckForReceiverInEcx(e)
	e.PushR10()
	e.PushR11()

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
	e.MovMem32RdiDispImm32(msgSystemKindOff, actorSystemKindUser)

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
	emitAccountMailboxEnqueueInRdi(e)

	emitWakeActorIfBlockedOnUserMailboxInRdi(e)

	e.XorEaxEax()
	e.Ret()
	overflowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, overflowAt, overflowTo); err != nil {
		return err
	}
	e.PopRax()
	e.PopRax()
	emitClearPendingMsg(e)
	emitMessagePoolAllocationFailure(e)
	emitMessagePoolExhaustedReturn(e)
	budgetTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, budgetAt, budgetTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	emitAccountActorByteBackpressureInRdi(e)
	emitMailboxFullReturn(e)
	invalidTo := len(e.Buf)
	for _, invalidAt := range append(remoteInvalidAts, invalidAts...) {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorHandleReturn(e)
	doneTo := len(e.Buf)
	for _, doneAt := range doneAts {
		if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
			return err
		}
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
	e.MovMem32RdiDispImm32(msgSystemKindOff, actorSystemKindUser)

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
	emitMessagePoolAllocationFailure(e)
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
	emitMovzxEaxWordRspDisp(e, actorWireOffsetVer)
	e.CmpEaxImm32(actorWireVersion)
	versionOKAt := e.JzRel32()
	retJumps = append(retJumps, e.JmpRel32())
	versionOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, versionOKAt, versionOKTo); err != nil {
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

	nodeDownTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeDownAt, nodeDownTo); err != nil {
		return err
	}
	if err := emitDeliverRemoteMonitorNodeDownFromRspFrame(e); err != nil {
		return err
	}
	if err := emitDeliverRemoteLinkedNodeDownFromRspFrame(e); err != nil {
		return err
	}

	failureStatusTo := len(e.Buf)
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
	e.MovMem32RdiDispImm32(msgSystemKindOff, actorSystemKindUser)

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
	emitAccountMailboxEnqueueInRdi(e)
	emitWakeActorIfBlockedOnUserMailboxInRdi(e)
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
	emitMessagePoolAllocationFailure(e)
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

// ---- linux_x64_emit_surface_recv.go ----

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

func emitSurfaceOpenHostIPC(e *x64.Emitter, opt SurfaceHostIPCOptions) error {
	// Host-required Surface mode starts from an AF_UNIX socket. The socket path
	// is validated and embedded by the builder before this runtime object is
	// linked; later requests speak tetra.surface.host-ipc.v1 over this fd.
	const (
		afUnix                  = 1
		sockStream              = 1
		surfaceHostStackLen     = 128
		surfaceHostHeaderOff    = 0
		surfaceHostResponseOff  = 40
		surfaceHostTitlePtrOff  = 88
		surfaceHostTitleLenOff  = 96
		surfaceHostOpenWidthOff = 104
		surfaceHostOpenHgtOff   = 108
		surfaceHostFDSpill      = 112
	)
	path := strings.TrimSpace(opt.SocketPath)
	if path == "" {
		return fmt.Errorf("surface host IPC socket path is required")
	}
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("surface host IPC socket path must not contain NUL")
	}
	pathBytes := []byte(path)
	if len(pathBytes) > 107 {
		return fmt.Errorf("surface host IPC socket path too long: %d bytes", len(pathBytes))
	}
	sockaddr := make([]byte, 2+len(pathBytes)+1)
	binary.LittleEndian.PutUint16(sockaddr[0:2], afUnix)
	copy(sockaddr[2:], pathBytes)

	var failReturnJumps []int
	var failCloseJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(surfaceHostStackLen)
	emitMovMem64RspDispRdi(e, surfaceHostTitlePtrOff)
	emitMovMem64RspDispRsi(e, surfaceHostTitleLenOff)
	emitMovMem32RspDispEdx(e, surfaceHostOpenWidthOff)
	emitMovMem32RspDispEcx(e, surfaceHostOpenHgtOff)

	e.MovEdiImm32(afUnix)                   // AF_UNIX
	e.Emit(0xBE, byte(sockStream), 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)                      // xor edx, edx
	e.MovEaxImm32(linuxSysSocket)
	e.Syscall()
	e.TestEaxEax()
	socketOK := e.JgeRel32()
	failReturnJumps = append(failReturnJumps, e.JmpRel32())
	socketOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, socketOK, socketOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, surfaceHostFDSpill) // fd spill

	e.Emit(0x8B, 0x7C, 0x24, surfaceHostFDSpill) // fd
	sockaddrLeaAt := e.LeaRsiRipDisp()
	e.MovEdxImm32(uint32(len(sockaddr)))
	e.MovEaxImm32(linuxSysConnect)
	e.Syscall()
	e.TestEaxEax()
	connectOK := e.JgeRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	connectOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, connectOK, connectOKTo); err != nil {
		return err
	}

	emitSurfaceHostInitRequestHeader(e, surfaceHostHeaderOff, surfaceHostOpOpen)
	e.MovEaxFromRspDisp(surfaceHostOpenWidthOff)
	e.MovMem32RspDispEax(surfaceHostHeaderOff + 16)
	e.MovEaxFromRspDisp(surfaceHostOpenHgtOff)
	e.MovMem32RspDispEax(surfaceHostHeaderOff + 20)
	e.MovEaxFromRspDisp(surfaceHostTitleLenOff)
	e.MovMem32RspDispEax(surfaceHostHeaderOff + 28)
	if err := emitSurfaceHostWriteStack(
		e,
		surfaceHostFDSpill,
		surfaceHostHeaderOff,
		surfaceHostRequestHeaderSize,
		&failCloseJumps,
	); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(surfaceHostFDSpill)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, surfaceHostTitlePtrOff)
	e.MovRdxFromRspDisp(surfaceHostTitleLenOff)
	if err := emitSurfaceHostIOFull(e, linuxSysWrite, &failCloseJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(
		e,
		surfaceHostFDSpill,
		surfaceHostResponseOff,
		&failCloseJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(
		e,
		surfaceHostResponseOff,
		&failCloseJumps,
	); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(surfaceHostFDSpill)
	e.Leave()
	e.Ret()

	failCloseTo := len(e.Buf)
	for _, at := range failCloseJumps {
		if err := x64.PatchRel32(e.Buf, at, failCloseTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRspDisp(surfaceHostFDSpill)
	e.MovRdiRax()
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()

	failReturnTo := len(e.Buf)
	for _, at := range failReturnJumps {
		if err := x64.PatchRel32(e.Buf, at, failReturnTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()

	sockaddrOffset := len(e.Buf)
	e.Emit(sockaddr...)
	if err := x64.PatchRel32(e.Buf, sockaddrLeaAt, sockaddrOffset); err != nil {
		return err
	}
	return nil
}

func emitSurfaceCloseHostIPC(e *x64.Emitter) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		stackLen    = 96
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpClose)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()
	e.XorEaxEax()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfaceSimpleHostIPC(e *x64.Emitter, op uint32) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		stackLen    = 96
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, op)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfaceNowMSHostIPC(e *x64.Emitter) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		stackLen    = 96
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpNowMS)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 16)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePresentRGBAHostIPC(e *x64.Emitter) error {
	const (
		headerOff    = 0
		responseOff  = 40
		fdOff        = 80
		pixelsPtrOff = 88
		pixelsLenOff = 96
		stackLen     = 128
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitMovMem64RspDispRsi(e, pixelsPtrOff)
	emitMovMem64RspDispRdx(e, pixelsLenOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpPresentRGBA)
	emitMovMem32RspDispEdi(e, headerOff+12)
	emitMovMem32RspDispEcx(e, headerOff+16)
	emitMovMem32RspDispR8d(e, headerOff+20)
	emitMovMem32RspDispR9d(e, headerOff+24)
	emitMovMem32RspDispEdx(e, headerOff+28)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, pixelsPtrOff)
	e.MovRdxFromRspDisp(pixelsLenOff)
	if err := emitSurfaceHostIOFull(e, linuxSysWrite, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePollEventSlotHostIPC(e *x64.Emitter, slot int32) error {
	if slot < 0 || slot >= surfaceHostEventSlots {
		return fmt.Errorf("surface host event slot out of range: %d", slot)
	}
	const (
		headerOff   = 0
		responseOff = 40
		eventOff    = 80
		fdOff       = 120
		stackLen    = 136
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpPollEventInto)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 32)
	e.CmpEaxImm32(surfaceHostEventPayloadSize)
	payloadSizeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	payloadSizeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, payloadSizeOK, payloadSizeOKTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitLeaRsiRspDisp(e, eventOff)
	e.MovEdxImm32(surfaceHostEventPayloadSize)
	if err := emitSurfaceHostIOFull(e, linuxSysRead, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(eventOff + slot*4)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePollEventIntoHostIPC(e *x64.Emitter) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		outPtrOff   = 88
		stackLen    = 112
	)
	var failJumps []int

	e.CmpEdxImm32(surfaceHostEventSlots)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitMovMem64RspDispRsi(e, outPtrOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpPollEventInto)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 32)
	e.CmpEaxImm32(surfaceHostEventPayloadSize)
	payloadSizeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	payloadSizeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, payloadSizeOK, payloadSizeOKTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, outPtrOff)
	e.MovEdxImm32(surfaceHostEventPayloadSize)
	if err := emitSurfaceHostIOFull(e, linuxSysRead, &failJumps); err != nil {
		return err
	}
	e.MovEaxImm32(surfaceHostEventSlots)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePollEventTextLenHostIPC(e *x64.Emitter) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		stackLen    = 96
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpPollEventTextInto)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 16)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePayloadIntoHostIPC(e *x64.Emitter, op uint32, minSlots int32) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		outPtrOff   = 88
		outLenOff   = 96
		stackLen    = 112
	)
	var failJumps []int

	if minSlots > 0 {
		e.CmpEdxImm32(minSlots)
		copyAt := e.JgeRel32()
		e.XorEaxEax()
		e.Ret()
		copyTo := len(e.Buf)
		if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
			return err
		}
	}

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitMovMem64RspDispRsi(e, outPtrOff)
	emitMovMem32RspDispEdx(e, outLenOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, op)
	emitMovMem32RspDispEdi(e, headerOff+12)
	emitMovMem32RspDispEdx(e, headerOff+16)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 32)
	e.MovEdxFromRspDisp(outLenOff)
	e.CmpEaxEdx()
	payloadFitsAt := e.JlRel32()
	payloadEqualAt := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	payloadFitsTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, payloadFitsAt, payloadFitsTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, payloadEqualAt, payloadFitsTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, outPtrOff)
	e.MovEdxFromRspDisp(responseOff + 32)
	if err := emitSurfaceHostIOFull(e, linuxSysRead, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 32)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfaceClipboardWriteTextHostIPC(e *x64.Emitter) error {
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		textPtrOff  = 88
		textLenOff  = 96
		stackLen    = 112
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitMovMem64RspDispRsi(e, textPtrOff)
	emitMovMem64RspDispRdx(e, textLenOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpClipboardWriteText)
	emitMovMem32RspDispEdi(e, headerOff+12)
	emitMovMem32RspDispEdx(e, headerOff+28)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, textPtrOff)
	e.MovRdxFromRspDisp(textLenOff)
	if err := emitSurfaceHostIOFull(e, linuxSysWrite, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 16)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfacePollCompositionIntoHostIPC(e *x64.Emitter) error {
	const surfaceHostCompositionPayloadSize = 16
	const (
		headerOff   = 0
		responseOff = 40
		fdOff       = 80
		outPtrOff   = 88
		stackLen    = 104
	)
	var failJumps []int

	e.CmpEdxImm32(4)
	copyAt := e.JgeRel32()
	e.XorEaxEax()
	e.Ret()
	copyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copyAt, copyTo); err != nil {
		return err
	}

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(stackLen)
	emitMovMem32RspDispEdi(e, fdOff)
	emitMovMem64RspDispRsi(e, outPtrOff)
	emitSurfaceHostInitRequestHeader(e, headerOff, surfaceHostOpPollCompositionInto)
	emitMovMem32RspDispEdi(e, headerOff+12)
	if err := emitSurfaceHostWriteStack(
		e,
		fdOff,
		headerOff,
		surfaceHostRequestHeaderSize,
		&failJumps,
	); err != nil {
		return err
	}
	if err := emitSurfaceHostReadResponse(e, fdOff, responseOff, &failJumps); err != nil {
		return err
	}
	if err := emitSurfaceHostRequireResponseOK(e, responseOff, &failJumps); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 32)
	e.CmpEaxImm32(surfaceHostCompositionPayloadSize)
	payloadSizeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	payloadSizeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, payloadSizeOK, payloadSizeOKTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitMovRsiFromRspDisp(e, outPtrOff)
	e.MovEdxImm32(surfaceHostCompositionPayloadSize)
	if err := emitSurfaceHostIOFull(e, linuxSysRead, &failJumps); err != nil {
		return err
	}
	e.MovEaxImm32(4)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitSurfaceHostInitRequestHeader(e *x64.Emitter, headerOff int32, op uint32) {
	emitMovMem32RspDispImm32(e, byte(headerOff+0), surfaceHostMagic)
	emitMovMem32RspDispImm32(e, byte(headerOff+4), op)
	emitMovMem32RspDispImm32(e, byte(headerOff+8), 0)
	emitMovMem32RspDispImm32(e, byte(headerOff+12), 0)
	emitMovMem32RspDispImm32(e, byte(headerOff+16), 0)
	emitMovMem32RspDispImm32(e, byte(headerOff+20), 0)
	emitMovMem32RspDispImm32(e, byte(headerOff+24), 0)
	emitMovMem32RspDispImm32(e, byte(headerOff+28), 0)
}

func emitSurfaceHostWriteStack(
	e *x64.Emitter,
	fdOff int32,
	bufOff int32,
	size uint32,
	failJumps *[]int,
) error {
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitLeaRsiRspDisp(e, byte(bufOff))
	e.MovEdxImm32(size)
	return emitSurfaceHostIOFull(e, linuxSysWrite, failJumps)
}

func emitSurfaceHostReadResponse(
	e *x64.Emitter,
	fdOff int32,
	responseOff int32,
	failJumps *[]int,
) error {
	e.MovEaxFromRspDisp(fdOff)
	e.MovRdiRax()
	emitLeaRsiRspDisp(e, byte(responseOff))
	e.MovEdxImm32(surfaceHostResponseSize)
	return emitSurfaceHostIOFull(e, linuxSysRead, failJumps)
}

func emitSurfaceHostIOFull(e *x64.Emitter, syscall uint32, failJumps *[]int) error {
	loopAt := len(e.Buf)
	e.MovRaxRdx()
	e.TestRaxRax()
	doneAt := e.JzRel32()
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.TestEaxEax()
	progressAt := e.JgRel32()
	e.CmpEaxImm32(-4)
	retryInterruptedAt := e.JzRel32()
	e.CmpEaxImm32(-11)
	retryAgainAt := e.JzRel32()
	*failJumps = append(*failJumps, e.JmpRel32())
	progressTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, progressAt, progressTo); err != nil {
		return err
	}
	emitAddRsiRax(e)
	emitSubRdxRax(e)
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loopAt); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, retryInterruptedAt, loopAt); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, retryAgainAt, loopAt); err != nil {
		return err
	}
	doneTo := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneTo)
}

func emitAddRsiRax(e *x64.Emitter) {
	e.Emit(0x48, 0x01, 0xC6)
}

func emitSubRdxRax(e *x64.Emitter) {
	e.Emit(0x48, 0x29, 0xC2)
}

func emitSurfaceHostRequireResponseOK(e *x64.Emitter, responseOff int32, failJumps *[]int) error {
	e.MovEaxFromRspDisp(responseOff + 0)
	e.CmpEaxImm32(surfaceHostMagic)
	magicOK := e.JzRel32()
	*failJumps = append(*failJumps, e.JmpRel32())
	magicOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, magicOK, magicOKTo); err != nil {
		return err
	}
	e.MovEaxFromRspDisp(responseOff + 12)
	e.TestEaxEax()
	statusOK := e.JzRel32()
	*failJumps = append(*failJumps, e.JmpRel32())
	statusOKTo := len(e.Buf)
	return x64.PatchRel32(e.Buf, statusOK, statusOKTo)
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

func emitSurfaceEventRecord(
	e *x64.Emitter,
	kind, x, y, button, key, width, height, timestamp, textLen uint32,
) {
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
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindUser)
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
	emitAccountMailboxDequeueInRdi(e)
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
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindUser)
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
	emitAccountMailboxDequeueInRdi(e)
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
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindUser)
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
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindUser)
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
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindUser)
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
	emitAccountMailboxDequeueInRdi(e)
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

func emitClearActorSystemRecvScratchInRdi(e *x64.Emitter, status int32) {
	e.MovMem32RdiDispImm32(actorSystemRecvScratchStatusOff, status)
	e.MovMem32RdiDispImm32(actorSystemRecvScratchCountOff, 0)
	e.XorEaxEax()
	for slot := 0; slot < 7; slot++ {
		e.MovMem64RdiDispRax(actorSystemRecvScratch0Off + int32(slot*8))
	}
}

func emitStoreSystemEventU64FieldToScratch(e *x64.Emitter, eventStackDisp int32, eventOff int32, scratchSlot int) {
	e.MovRaxFromRspDisp(eventStackDisp)
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(eventOff)
	e.MovRdiRdx()
	e.MovMem64RdiDispRax(actorSystemRecvScratch0Off + int32(scratchSlot*8))
}

func emitStoreSystemEventU32FieldToScratch(e *x64.Emitter, eventStackDisp int32, eventOff int32, scratchSlot int) {
	e.MovRaxFromRspDisp(eventStackDisp)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(eventOff)
	e.MovRdiRdx()
	e.MovMem64RdiDispRax(actorSystemRecvScratch0Off + int32(scratchSlot*8))
}

func emitCopySystemEventToActorRecvScratch(e *x64.Emitter, eventStackDisp int32) {
	// rdx must contain the receiving actor pointer. Slots mirror
	// actor.system_recv_raw after the status returned by begin().
	e.MovRdiRdx()
	e.MovMem32RdiDispImm32(actorSystemRecvScratchStatusOff, actorSystemRecvStatusMessage)
	e.MovMem32RdiDispImm32(actorSystemRecvScratchCountOff, 7)
	emitStoreSystemEventU32FieldToScratch(e, eventStackDisp, systemEventKindOff, 0)
	emitStoreSystemEventU64FieldToScratch(e, eventStackDisp, systemEventSubjectOff, 1)
	emitStoreSystemEventU64FieldToScratch(e, eventStackDisp, systemEventMonitorRefOff, 2)
	emitStoreSystemEventU64FieldToScratch(e, eventStackDisp, systemEventNodeIDOff, 3)
	emitStoreSystemEventU64FieldToScratch(e, eventStackDisp, systemEventNodeEpochOff, 4)
	emitStoreSystemEventU32FieldToScratch(e, eventStackDisp, systemEventReasonKindOff, 5)
	emitStoreSystemEventU32FieldToScratch(e, eventStackDisp, systemEventReasonCodeOff, 6)
}

func emitRecvSystemBegin(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	e.PushRdi() // mode
	e.PushRsi() // absolute deadline for mode 2

	loopStart := len(e.Buf)
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_net_pump"})
	actorPtrInRax(e)
	e.MovRdxRax()

	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(actorSystemMailboxHeadOff)
	e.TestRaxRax()
	haveEventAt := e.JnzRel32()

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedRuntimeClosingOff)
	e.TestEaxEax()
	notClosingAt := e.JzRel32()
	e.MovRdiRdx()
	emitClearActorSystemRecvScratchInRdi(e, actorSystemRecvStatusRuntimeClosed)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	e.AddRspImm32(16)
	e.MovEaxImm32(actorSystemRecvStatusRuntimeClosed)
	e.Ret()
	notClosingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notClosingAt, notClosingTo); err != nil {
		return err
	}

	if err := emitCurrentTaskGroupCanceledCheck(e, func() {
		e.MovRdiRdx()
		emitClearActorSystemRecvScratchInRdi(e, actorSystemRecvStatusCanceled)
		e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
		e.AddRspImm32(16)
		e.MovEaxImm32(actorSystemRecvStatusCanceled)
		e.Ret()
	}); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(8)
	e.CmpEaxImm32(1)
	pollEmptyAt := e.JzRel32()
	e.CmpEaxImm32(2)
	timedAt := e.JzRel32()
	e.CmpEaxImm32(0)
	blockingAt := e.JzRel32()

	e.MovRdiRdx()
	emitClearActorSystemRecvScratchInRdi(e, actorSystemRecvStatusInvalidState)
	e.AddRspImm32(16)
	e.MovEaxImm32(actorSystemRecvStatusInvalidState)
	e.Ret()

	pollEmptyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, pollEmptyAt, pollEmptyTo); err != nil {
		return err
	}
	e.MovRdiRdx()
	emitClearActorSystemRecvScratchInRdi(e, actorSystemRecvStatusEmpty)
	e.AddRspImm32(16)
	e.MovEaxImm32(actorSystemRecvStatusEmpty)
	e.Ret()

	timedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, timedAt, timedTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxFromRspDisp(0)
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()
	e.MovR13Rcx()
	setCurrentActorWakeAtFromR13(e)
	e.MovRdiRdx()
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindSystem)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	timedLoopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, timedLoopAt, loopStart); err != nil {
		return err
	}

	timeoutTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, timeoutAt, timeoutTo); err != nil {
		return err
	}
	e.MovRdiRdx()
	emitClearActorSystemRecvScratchInRdi(e, actorSystemRecvStatusTimeout)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	e.AddRspImm32(16)
	e.MovEaxImm32(actorSystemRecvStatusTimeout)
	e.Ret()

	blockingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, blockingAt, blockingTo); err != nil {
		return err
	}
	clearCurrentActorWakeAt(e)
	e.MovRdiRdx()
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindSystem)
	e.MovMem32RdiDispImm32(actorStatusOff, statusBlocked)
	at = e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	blockingLoopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, blockingLoopAt, loopStart); err != nil {
		return err
	}

	haveEventTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, haveEventAt, haveEventTo); err != nil {
		return err
	}
	e.PushRax()
	emitCopySystemEventToActorRecvScratch(e, 0)

	e.MovRaxFromRspDisp(0)
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(systemEventNextOff)
	e.MovRcxRax()

	e.MovRdiRdx()
	e.MovRaxRcx()
	e.MovMem64RdiDispRax(actorSystemMailboxHeadOff)
	e.TestRaxRax()
	keepTailAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovMem64RdiDispRax(actorSystemMailboxTailOff)
	keepTailTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepTailAt, keepTailTo); err != nil {
		return err
	}
	emitDecrementSystemMailboxCount(e)
	emitAccountSystemMailboxDequeueInRdi(e)
	e.MovMem32RdiDispImm32(actorWaitKindOff, actorWaitKindNone)
	emitRecycleSystemEventNodeFromRspDisp(e, 0)
	e.AddRspImm32(24)
	e.MovEaxImm32(actorSystemRecvStatusMessage)
	e.Ret()
	return nil
}

func emitRecvSystemSlot(e *x64.Emitter) error {
	e.MovRaxRdi()
	e.CmpRaxImm32(7)
	outOfRangeAt := e.JaeRel32()
	e.ShlRaxImm8(3)
	e.AddRaxImm32(actorSystemRecvScratch0Off)
	e.PushRax()
	actorPtrInRax(e)
	e.PopRdx()
	e.AddRaxRdx()
	e.MovRdiRax()
	e.MovRaxFromRdiDisp(0)
	e.Ret()

	outOfRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, outOfRangeAt, outOfRangeTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitRecvSystemCount(e *x64.Emitter) error {
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorSystemRecvScratchCountOff)
	e.Ret()
	return nil
}

func emitSelf(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.PopRax()
	emitLocalActorRefReturnFromSlotInEaxActorPtrInRdi(e)
	e.Ret()
	return nil
}

func emitSender(e *x64.Emitter) error {
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovEaxFromRdiDisp(actorLastSenderOff)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.PopRax()
	emitLocalActorRefReturnFromSlotInEaxActorPtrInRdi(e)
	e.Ret()
	return nil
}

func emitActorStatus(e *x64.Emitter) error {
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusStarting)
	notStartingAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusStarting)
	e.Ret()
	notStartingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notStartingAt, notStartingTo); err != nil {
		return err
	}

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusFree)
	notFreeAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusDead)
	e.Ret()
	notFreeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notFreeAt, notFreeTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusDone)
	skipCanceledForDoneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	skipCanceledForReclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusStopping)
	skipCanceledForStoppingAt := e.JzRel32()
	e.PushRdi()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	notCanceledAt := e.JnzRel32()
	e.PopRdi()
	e.MovEaxImm32(actorLifecycleStatusCanceled)
	e.Ret()

	restoreActorPtrTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, restoreActorPtrTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notCanceledAt, restoreActorPtrTo); err != nil {
		return err
	}
	e.PopRdi()
	skipCanceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCanceledForDoneAt, skipCanceledTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, skipCanceledForReclaimableAt, skipCanceledTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, skipCanceledForStoppingAt, skipCanceledTo); err != nil {
		return err
	}

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusReady)
	notReadyAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusReady)
	e.Ret()
	notReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notReadyAt, notReadyTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusRunning)
	notRunningAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusRunning)
	e.Ret()
	notRunningTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notRunningAt, notRunningTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusBlocked)
	e.Ret()
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusStopping)
	notStoppingAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusStopping)
	e.Ret()
	notStoppingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notStoppingAt, notStoppingTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	notDoneAt := e.JmpRel32()
	terminalTo := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, terminalTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.TestEaxEax()
	errorExitAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusExitedNormal)
	e.Ret()
	errorExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, errorExitAt, errorExitTo); err != nil {
		return err
	}
	e.MovEaxImm32(actorLifecycleStatusExitedError)
	e.Ret()
	notDoneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notDoneAt, notDoneTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusSleeping)
	notSleepingAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusSleeping)
	e.Ret()
	notSleepingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notSleepingAt, notSleepingTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusWaiting)
	notWaitingAt := e.JnzRel32()
	e.MovEaxImm32(actorLifecycleStatusWaiting)
	e.Ret()
	notWaitingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notWaitingAt, notWaitingTo); err != nil {
		return err
	}

	e.MovEaxImm32(actorLifecycleStatusDead)
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorStatusRaw(e *x64.Emitter) error {
	invalidAts, staleAts, err := emitDecodeLocalActorRefInRdiRsiToEcxClassified(e)
	if err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusStarting)
	notStartingAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusStarting, 0)
	notStartingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notStartingAt, notStartingTo); err != nil {
		return err
	}

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusFree)
	notFreeAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusDead, 0)
	notFreeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notFreeAt, notFreeTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusDone)
	skipCanceledForDoneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	skipCanceledForReclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusStopping)
	skipCanceledForStoppingAt := e.JzRel32()
	e.PushRdi()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.TestEaxEax()
	noGroupAt := e.JzRel32()
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	notCanceledAt := e.JnzRel32()
	e.PopRdi()
	emitActorStatusRawReturn(e, actorLifecycleStatusCanceled, 0)

	restoreActorPtrTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, noGroupAt, restoreActorPtrTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, notCanceledAt, restoreActorPtrTo); err != nil {
		return err
	}
	e.PopRdi()
	skipCanceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCanceledForDoneAt, skipCanceledTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, skipCanceledForReclaimableAt, skipCanceledTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, skipCanceledForStoppingAt, skipCanceledTo); err != nil {
		return err
	}

	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusReady)
	notReadyAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusReady, 0)
	notReadyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notReadyAt, notReadyTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusRunning)
	notRunningAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusRunning, 0)
	notRunningTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notRunningAt, notRunningTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusBlocked)
	notBlockedAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusBlocked, 0)
	notBlockedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notBlockedAt, notBlockedTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusStopping)
	notStoppingAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusStopping, 0)
	notStoppingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notStoppingAt, notStoppingTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	notDoneAt := e.JmpRel32()
	terminalTo := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, terminalTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.TestEaxEax()
	errorExitAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusExitedNormal, 0)
	errorExitTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, errorExitAt, errorExitTo); err != nil {
		return err
	}
	emitActorStatusRawReturn(e, actorLifecycleStatusExitedError, 0)
	notDoneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notDoneAt, notDoneTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusSleeping)
	notSleepingAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusSleeping, 0)
	notSleepingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notSleepingAt, notSleepingTo); err != nil {
		return err
	}

	e.CmpEaxImm32(statusWaiting)
	notWaitingAt := e.JnzRel32()
	emitActorStatusRawReturn(e, actorLifecycleStatusWaiting, 0)
	notWaitingTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notWaitingAt, notWaitingTo); err != nil {
		return err
	}

	e.MovEdxImm32(0)
	e.Ret()
	if err := patchActorStatusRawResultReturns(e, invalidAts, 0xFFFFFFFD); err != nil {
		return err
	}
	return patchActorStatusRawResultReturns(e, staleAts, 0xFFFFFFFB)
}

func emitActorWait(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	invalidAts, staleAts, err := emitDecodeLocalActorRefInRdiRsiToEcxClassified(e)
	if err != nil {
		return err
	}
	e.MovEaxEcx()
	e.MovRcxRax()
	e.MovR12Rcx()

	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()

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
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, loop); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
	emitActorWaitDoneResultReturn(e)
	if err := patchInvalidActorWaitResultReturns(e, invalidAts); err != nil {
		return err
	}
	return patchStaleActorWaitResultReturns(e, staleAts)
}

func emitActorWaitUntil(e *x64.Emitter, callPatches *[]callPatch) error {
	if callPatches == nil {
		return fmt.Errorf("missing callPatches")
	}
	if err := clampEdxNonNegativeIntoR13(e); err != nil {
		return err
	}
	invalidAts, staleAts, err := emitDecodeLocalActorRefInRdiRsiToEcxClassified(e)
	if err != nil {
		return err
	}
	e.MovEaxEcx()
	e.MovRcxRax()
	e.MovR12Rcx()

	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	e.MovEdxEax()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedTimeMsOff)
	e.MovEcxR13d()
	e.CmpEaxEcx()
	timeoutAt := e.JaeRel32()

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
	e.Ret()

	doneTo := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
	emitActorWaitDoneResultReturn(e)
	if err := patchInvalidActorWaitResultReturns(e, invalidAts); err != nil {
		return err
	}
	return patchStaleActorWaitResultReturns(e, staleAts)
}

func emitActorStop(e *x64.Emitter) error {
	e.MovR8dEdx()
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	e.MovMem32RdiDispR8d(actorExitCodeOff)
	e.MovMem32RdiDispImm32(actorStatusOff, statusStopping)
	e.XorEaxEax()
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorExitReason(e *x64.Emitter) error {
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorLifecycleAck(e *x64.Emitter) error {
	invalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorLink(e *x64.Emitter) error {
	var invalidAts []int

	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()

	e.MovR8Rdi()
	e.PushRsi()
	e.PopR10()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, remoteInvalidAts...)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovRdxR8()
	e.PushR10()
	e.PopR9()
	emitAddLinkRefToActorPtrInRdiRefLowInEdxHighInR9d(e)
	e.Ret()

	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	localInvalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, localInvalidAts...)

	e.PushRdi()
	e.MovEaxEcx()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	e.MovEdxFromRdiDisp(actorGenerationOff)
	emitOrEaxEdx(e)
	e.MovEdxEax()
	e.MovR8dEdx()
	e.PopRdi()

	e.PushRdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(actorGenerationOff)
	e.PopRax()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	emitOrEaxEcx(e)
	e.MovEdxEax()
	e.PopRdi()

	e.MovR9dImm32(actorRefLocalHighSlot)
	emitAddLinkRefToActorPtrInRdiRefLowInEdxHighInR9d(e)
	e.TestEaxEax()
	firstAddFailedAt := e.JnzRel32()

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovRdxR8()
	e.MovR9dImm32(actorRefLocalHighSlot)
	emitAddLinkRefToActorPtrInRdiRefLowInEdxHighInR9d(e)
	e.Ret()

	firstAddFailedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, firstAddFailedAt, firstAddFailedTo); err != nil {
		return err
	}
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorUnlink(e *x64.Emitter) error {
	var invalidAts []int

	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()

	e.MovR8Rdi()
	e.PushRsi()
	e.PopR10()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, remoteInvalidAts...)
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovRdxR8()
	e.PushR10()
	e.PopR9()
	emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9d(e)
	e.Ret()

	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	localInvalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, localInvalidAts...)

	e.PushRdi()
	e.MovEaxEcx()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	e.MovEdxFromRdiDisp(actorGenerationOff)
	emitOrEaxEdx(e)
	e.MovEdxEax()
	e.MovR8dEdx()
	e.PopRdi()

	e.PushRdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(actorGenerationOff)
	e.PopRax()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	emitOrEaxEcx(e)
	e.MovEdxEax()
	e.PopRdi()

	e.MovR9dImm32(actorRefLocalHighSlot)
	emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9d(e)

	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovRdxR8()
	e.MovR9dImm32(actorRefLocalHighSlot)
	emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9d(e)
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitAddLinkRefToActorPtrInRdiRefLowInEdxHighInR9d(e *x64.Emitter) {
	e.MovEaxFromRdiDisp(actorLinkCountOff)
	e.XorEcxEcx()
	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	insertAt := e.JzRel32()

	e.PushRax()
	e.PushRcx()
	emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkRef0Off)
	e.CmpEaxEdx()
	lowMatchAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		panic(err)
	}

	lowMatchTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, lowMatchAt, lowMatchTo); err != nil {
		panic(err)
	}
	emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkHigh0Off)
	emitCmpEaxR9d(e)
	duplicateAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAfterHighAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAfterHighAt, scanStart); err != nil {
		panic(err)
	}

	insertTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, insertAt, insertTo); err != nil {
		panic(err)
	}
	e.CmpEaxImm32(maxActorLinks)
	fullAt := e.JaeRel32()
	e.PushRax()
	emitReserveSystemEventCreditForActorPtrInRdi(e)
	e.TestEaxEax()
	reserveFailedAt := e.JnzRel32()
	e.PopRax()
	e.PushRax()
	e.MovEcxEax()
	emitStoreActorLinkTableI32AtIndexInEcxValueInEdx(e, actorLinkRef0Off)
	e.PushR9()
	e.PopRdx()
	emitStoreActorLinkTableI32AtIndexInEcxValueInEdx(e, actorLinkHigh0Off)
	e.PopRax()
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(actorLinkCountOff)
	e.XorEaxEax()
	insertDoneAt := e.JmpRel32()

	duplicateTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, duplicateAt, duplicateTo); err != nil {
		panic(err)
	}
	e.PopRcx()
	e.PopRax()
	e.XorEaxEax()
	duplicateDoneAt := e.JmpRel32()

	reserveFailedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, reserveFailedAt, reserveFailedTo); err != nil {
		panic(err)
	}
	e.PopRax()
	e.MovEaxImm32(0xFFFFFFFE)
	reserveFailedDoneAt := e.JmpRel32()

	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		panic(err)
	}
	e.MovEaxImm32(0xFFFFFFFE)

	doneTo := len(e.Buf)
	for _, at := range []int{insertDoneAt, duplicateDoneAt, reserveFailedDoneAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			panic(err)
		}
	}
}

func emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9d(e *x64.Emitter) {
	emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9dWithCredit(e, true)
}

func emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9dWithoutCreditRelease(e *x64.Emitter) {
	emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9dWithCredit(e, false)
}

func emitRemoveLinkRefFromActorPtrInRdiRefLowInEdxHighInR9dWithCredit(
	e *x64.Emitter,
	releaseCredit bool,
) {
	e.MovEaxFromRdiDisp(actorLinkCountOff)
	e.TestEaxEax()
	emptyAt := e.JzRel32()
	e.XorEcxEcx()
	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	notFoundAt := e.JzRel32()

	e.PushRax()
	e.PushRcx()
	emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkRef0Off)
	e.CmpEaxEdx()
	lowMatchAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		panic(err)
	}

	lowMatchTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, lowMatchAt, lowMatchTo); err != nil {
		panic(err)
	}
	emitLoadActorLinkTableI32AtIndexInEcxToEax(e, actorLinkHigh0Off)
	emitCmpEaxR9d(e)
	foundAt := e.JzRel32()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAfterHighAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAfterHighAt, scanStart); err != nil {
		panic(err)
	}

	foundTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, foundAt, foundTo); err != nil {
		panic(err)
	}
	e.PopRcx() // found index
	e.PopRax() // original count
	if releaseCredit {
		e.PushRax()
		e.PushRcx()
		emitReleaseSystemEventCreditForActorPtrInRdi(e)
		e.PopRcx()
		e.PopRax()
	}
	e.AddEaxImm32(-1)
	e.PushRax()
	e.CmpEaxEcx()
	skipCopyAt := e.JzRel32()
	emitCopyActorLinkTableI32LastIndexInEaxToIndexInEcx(e, actorLinkRef0Off)
	emitCopyActorLinkTableI32LastIndexInEaxToIndexInEcx(e, actorLinkHigh0Off)
	skipCopyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCopyAt, skipCopyTo); err != nil {
		panic(err)
	}
	e.PopRax()
	e.MovMem32RdiDispEax(actorLinkCountOff)
	foundDoneAt := e.JmpRel32()

	doneTo := len(e.Buf)
	for _, at := range []int{emptyAt, notFoundAt, foundDoneAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			panic(err)
		}
	}
	e.XorEaxEax()
}

func emitLoadActorLinkTableI32AtIndexInEcxToEax(e *x64.Emitter, baseOff int32) {
	e.PushRcx()
	e.MovRaxRdi()
	e.AddRaxImm32(baseOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.AddRaxRbx()
	e.MovEaxFromRaxPtr()
	e.PopRcx()
}

func emitStoreActorLinkTableI32AtIndexInEcxValueInEdx(e *x64.Emitter, baseOff int32) {
	e.PushRcx()
	e.MovRaxRdi()
	e.AddRaxImm32(baseOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.AddRaxRbx()
	e.PushRdx()
	e.PopRcx()
	e.MovMem32RaxPtrEcx()
	e.PopRcx()
}

func emitCopyActorLinkTableI32LastIndexInEaxToIndexInEcx(e *x64.Emitter, baseOff int32) {
	e.PushRax()
	e.PushRcx()
	e.PushRax()
	e.PopRbx()
	e.ShlRbxImm8(2)
	e.MovRaxRdi()
	e.AddRaxImm32(baseOff)
	e.AddRaxRbx()
	e.MovEaxFromRaxPtr()
	e.MovEdxEax()
	e.PopRcx()
	e.PushRcx()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxRdi()
	e.AddRaxImm32(baseOff)
	e.AddRaxRbx()
	e.PushRdx()
	e.PopRcx()
	e.MovMem32RaxPtrEcx()
	e.PopRcx()
	e.PopRax()
}

func emitCmpEaxR9d(e *x64.Emitter) {
	e.Emit(0x44, 0x39, 0xC8)
}

func emitActorMonitor(e *x64.Emitter) error {
	var invalidAts []int

	e.MovEaxEsi()
	e.CmpEaxImm32(actorRefLocalHighSlot)
	localAt := e.JzRel32()

	e.MovR8Rdi()
	e.PushRsi()
	e.PopR10()
	remoteInvalidAts, err := emitDecodeRemoteActorRefInRdiRsiToLegacyHandleInEdi(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, remoteInvalidAts...)
	targetDecodedAt := e.JmpRel32()

	localTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, localAt, localTo); err != nil {
		return err
	}
	localInvalidAts, err := emitDecodeLocalActorRefInRdiRsiToEcx(e)
	if err != nil {
		return err
	}
	invalidAts = append(invalidAts, localInvalidAts...)
	e.PushRdi()
	e.MovEaxEcx()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	e.MovEdxFromRdiDisp(actorGenerationOff)
	emitOrEaxEdx(e)
	e.MovEdxEax()
	e.MovR8dEdx()
	e.MovR10dImm32(actorRefLocalHighSlot)
	e.PopRdi()

	targetDecodedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, targetDecodedAt, targetDecodedTo); err != nil {
		return err
	}

	e.PushRdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	e.PushRax()
	actorPtrFromEaxToRdi(e)
	e.MovEcxFromRdiDisp(actorGenerationOff)
	e.PopRax()
	e.Emit(0xC1, 0xE0, 0x10) // shl eax, 16
	emitOrEaxEcx(e)
	e.MovEdxEax()
	e.PushRdx()
	e.PopR9()
	e.PopRdi()

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedMonitorCountOff)
	e.CmpEaxImm32(maxActorMonitors)
	fullAt := e.JaeRel32()
	e.MovEcxEax()
	e.PushRcx()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCurrentIdxOff)
	actorPtrFromEaxToRdi(e)
	emitReserveSystemEventCreditForActorPtrInRdi(e)
	e.TestEaxEax()
	reserveFailedAt := e.JnzRel32()
	e.PopRcx()
	e.MovRdiR15()

	e.MovEaxFromRdiDisp(schedMonitorNextIDOff)
	e.PushRax()
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(schedMonitorNextIDOff)
	e.PopRax()
	e.PushRax()

	e.MovEdxEax()
	emitStoreSchedulerTableI32AtIndexInEcxValueInEdx(e, schedMonitorID0Off)
	e.MovRdxR9()
	emitStoreSchedulerTableI32AtIndexInEcxValueInEdx(e, schedMonitorOwnerRef0Off)
	e.MovRdxR8()
	emitStoreSchedulerTableI32AtIndexInEcxValueInEdx(e, schedMonitorTargetRef0Off)
	e.PushR10()
	e.PopRdx()
	emitStoreSchedulerTableI32AtIndexInEcxValueInEdx(e, schedMonitorTargetHigh0Off)

	e.MovEaxEcx()
	e.AddEaxImm32(1)
	e.MovMem32RdiDispEax(schedMonitorCountOff)
	e.PopRax()
	e.Ret()

	reserveFailedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, reserveFailedAt, reserveFailedTo); err != nil {
		return err
	}
	e.PopRcx()
	e.MovEaxImm32(0xFFFFFFFE)
	e.Ret()

	fullTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fullAt, fullTo); err != nil {
		return err
	}
	e.MovEaxImm32(0xFFFFFFFE)
	e.Ret()
	return patchInvalidActorHandleReturns(e, invalidAts)
}

func emitActorDemonitor(e *x64.Emitter) error {
	var invalidAts []int

	e.CmpEdiImm32(1)
	validAt := e.JgeRel32()
	invalidAts = append(invalidAts, e.JmpRel32())
	validTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, validAt, validTo); err != nil {
		return err
	}

	e.MovEdxEdi()
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedMonitorCountOff)
	e.TestEaxEax()
	invalidAts = append(invalidAts, e.JzRel32())
	e.XorEcxEcx()
	scanStart := len(e.Buf)
	e.CmpEaxEcx()
	invalidAts = append(invalidAts, e.JzRel32())

	e.PushRax()
	e.PushRcx()
	e.MovRaxR15()
	e.AddRaxImm32(schedMonitorID0Off)
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.AddRaxRbx()
	e.PushRax()
	e.MovEaxFromRaxPtr()
	e.CmpEaxEdx()
	foundAt := e.JzRel32()
	e.PopRax()
	e.PopRcx()
	e.PopRax()
	e.AddEcxImm8(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scanStart); err != nil {
		return err
	}

	foundTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, foundAt, foundTo); err != nil {
		return err
	}
	e.PopR8()
	e.PopRcx()
	e.PopRax()
	e.PushRax()
	e.PushRcx()
	emitLoadSchedulerTableI32AtIndexInEcxToEax(e, schedMonitorOwnerRef0Off)
	emitShrEaxImm8(e, 16)
	emitAndEaxImm32(e, 0xFFFF)
	actorPtrFromEaxToRdi(e)
	emitReleaseSystemEventCreditForActorPtrInRdi(e)
	e.PopRcx()
	e.PopRax()
	e.AddEaxImm32(-1)
	e.PushRax()
	e.CmpEaxEcx()
	skipCopyAt := e.JzRel32()
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorID0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorOwnerRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetRef0Off)
	emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e, schedMonitorTargetHigh0Off)
	skipCopyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, skipCopyAt, skipCopyTo); err != nil {
		return err
	}
	e.PopRax()
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedMonitorCountOff)
	e.XorEaxEax()
	e.Ret()

	invalidTo := len(e.Buf)
	for _, invalidAt := range invalidAts {
		if err := x64.PatchRel32(e.Buf, invalidAt, invalidTo); err != nil {
			return err
		}
	}
	emitInvalidActorHandleReturn(e)
	return nil
}

func emitStoreSchedulerTableI32AtIndexInEcxValueInEdx(e *x64.Emitter, baseOff int32) {
	e.PushRcx()
	e.MovRaxR15()
	e.AddRaxImm32(baseOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.AddRaxRbx()
	e.PushRdx()
	e.PopRcx()
	e.MovMem32RaxPtrEcx()
	e.PopRcx()
}

func emitLoadSchedulerTableI32AtIndexInEcxToEax(e *x64.Emitter, baseOff int32) {
	e.PushRcx()
	e.MovRaxR15()
	e.AddRaxImm32(baseOff)
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.AddRaxRbx()
	e.MovEaxFromRaxPtr()
	e.PopRcx()
}

func emitCopySchedulerTableI32LastIndexInEaxToIndexInEcx(e *x64.Emitter, baseOff int32) {
	e.PushRax()
	e.PushRcx()
	e.PushRax()
	e.PopRbx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(baseOff)
	e.AddRaxRbx()
	e.MovEaxFromRaxPtr()
	e.MovEdxEax()
	e.PopRcx()
	e.PushRcx()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(baseOff)
	e.AddRaxRbx()
	e.PushRdx()
	e.PopRcx()
	e.MovMem32RaxPtrEcx()
	e.PopRcx()
	e.PopRax()
}

func emitActorSetTrapExit(e *x64.Emitter) error {
	actorPtrInRax(e)
	e.MovRdiRax()
	e.MovMem32RdiDispEdi(actorTrapExitOff)
	e.XorEaxEax()
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

func emitInitActorStack(
	e *x64.Emitter,
	sysMmap uint32,
	mapFlags uint32,
	leaPatches *[]leaPatch,
) error {
	// Stack mapping.
	emitMmapAnon(e, stackSize, sysMmap, mapFlags)
	// initRsp = base + stackSize - 56
	e.AddRaxImm32(stackSize)
	e.AddRaxImm32(-56)
	e.MovRcxRax() // initRsp in rcx

	return emitFillActorStackFrameInRcx(e, leaPatches)
}

func emitFillActorStackFrameInRcx(e *x64.Emitter, leaPatches *[]leaPatch) error {
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

// ---- linux_x64_emit_tasks.go ----

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
	if err := emitTaskCanceledCheck(
		e,
		func() { emitTaskJoinI32CanceledReturn(e, result) },
	); err != nil {
		return err
	}
	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	if err := emitTaskCanceledCheck(
		e,
		func() { emitTaskJoinI32CanceledReturn(e, result) },
	); err != nil {
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
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
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
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
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
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
	}
	e.MovEaxFromRdiDisp(actorExitCodeOff)
	e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
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
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(2)
	e.Ret()

	doneTo := len(e.Buf)
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
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
	e.PushRax()
	e.MovEaxFromRaxPtr()
	e.PopRdi()
	e.MovMem32RdiDispImm32(0, 0)
	e.Ret()

	outOfRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, outOfRangeAt, outOfRangeTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitClearJoinedTaskResultSlots(e *x64.Emitter, slots int) {
	e.MovMem32RdiDispImm32(actorTaskCountOff, 0)
	for slot := 0; slot < slots; slot++ {
		e.MovMem32RdiDispImm32(actorTaskSlot0Off+int32(slot*4), 0)
	}
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
	if err := emitTaskCanceledCheck(
		e,
		func() { emitTaskJoinTypedCanceledReturn(e, slots) },
	); err != nil {
		return err
	}
	loop := len(e.Buf)
	actorPtrFromR12ToRdi(e)
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneAt := e.JzRel32()
	e.CmpEaxImm32(statusReclaimable)
	reclaimableAt := e.JzRel32()
	e.CmpEaxImm32(statusWaiting)
	targetWaitingAt := e.JzRel32()
	if err := emitTaskCanceledCheck(
		e,
		func() { emitTaskJoinTypedCanceledReturn(e, slots) },
	); err != nil {
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
	for _, at := range []int{doneAt, reclaimableAt} {
		if err := x64.PatchRel32(e.Buf, at, doneTo); err != nil {
			return err
		}
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
		e.MovRaxFromRspDisp(8)
		e.MovRdiRax()
		emitClearJoinedTaskResultSlots(e, slots)
		e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
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
	emitClearJoinedTaskResultSlots(e, slots)
	e.MovMem32RdiDispImm32(actorStatusOff, statusReclaimable)
	e.Ret()
	return nil
}

// ---- macos_x64.go ----

func BuildMacOSX64(entries []string) (*tobj.Object, error) {
	abi := x64abi.MacSysV()
	const macMapPrivateAnon = 0x1002
	return buildSysVUnixX64(
		entries,
		abi.SysMmap,
		abi.SysMunmap,
		macMapPrivateAnon,
		false,
		SurfaceHostIPCOptions{},
	)
}

// ---- production_boundary.go ----

type ActorRuntimeBoundaryID string

const (
	ActorRuntimeBoundaryCurrentLimits        ActorRuntimeBoundaryID = "current_actor_runtime_limits"
	ActorRuntimeBoundarySchedulerPrototype   ActorRuntimeBoundaryID = "scheduler_prototype_features"
	ActorRuntimeBoundaryProductionAcceptance ActorRuntimeBoundaryID = "production_runtime_acceptance"
	ActorRuntimeBoundaryFullClaimBlockers    ActorRuntimeBoundaryID = "full_claim_blockers"
)

type ActorRuntimeBoundaryStatus string

const (
	ActorRuntimeBoundaryDocumentedLimit    ActorRuntimeBoundaryStatus = "documented_limit"
	ActorRuntimeBoundaryPrototypeEvidence  ActorRuntimeBoundaryStatus = "prototype_evidence"
	ActorRuntimeBoundaryAcceptanceRequired ActorRuntimeBoundaryStatus = "acceptance_required"
	ActorRuntimeBoundaryBlocked            ActorRuntimeBoundaryStatus = "blocked"
)

type ActorRuntimeBoundaryReport struct {
	SchemaVersion         string                    `json:"schema_version"`
	Rows                  []ActorRuntimeBoundaryRow `json:"rows"`
	NonClaims             []string                  `json:"non_claims"`
	FullProductionClaimed bool                      `json:"full_production_claimed"`
}

type ActorRuntimeBoundaryRow struct {
	ID            ActorRuntimeBoundaryID     `json:"id"`
	Name          string                     `json:"name"`
	Status        ActorRuntimeBoundaryStatus `json:"status"`
	RequiredFacts []string                   `json:"required_facts,omitempty"`
	MissingFacts  []string                   `json:"missing_facts,omitempty"`
	Evidence      string                     `json:"evidence"`
	Boundary      string                     `json:"boundary"`
}

func ActorRuntimeProductionBoundaryAudit() (ActorRuntimeBoundaryReport, error) {
	benchmarks, err := parallelrt.PrototypeBenchmarks()
	if err != nil {
		return ActorRuntimeBoundaryReport{}, err
	}
	if len(benchmarks) < 2 {
		return ActorRuntimeBoundaryReport{}, fmt.Errorf(
			"actor runtime boundary audit: scheduler prototype benchmark evidence is incomplete",
		)
	}
	return ActorRuntimeBoundaryReport{
		SchemaVersion: "tetra.runtime.actor.production_boundary.v1",
		Rows: []ActorRuntimeBoundaryRow{
			currentActorRuntimeLimitsRow(),
			schedulerPrototypeFeaturesRow(benchmarks),
			productionRuntimeAcceptanceRow(),
			fullClaimBlockersRow(),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			"scheduler prototype evidence is not a production multi-threaded actor scheduler",
			"distributed actor runtime support remains bounded to Linux-x64 loopback TCP smoke evidence",
			("nonzero actor entry returns have no user-facing actor " +
				"status, join, exit-code, supervision, or restart API"),
			("missing-node node_down evidence does not claim automatic " +
				"retry, restart, reconnect, or supervision"),
		},
		FullProductionClaimed: false,
	}, nil
}

func ValidateActorRuntimeProductionBoundaryAudit(report ActorRuntimeBoundaryReport) error {
	if report.SchemaVersion != "tetra.runtime.actor.production_boundary.v1" {
		return fmt.Errorf("actor runtime boundary audit: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionClaimed {
		return fmt.Errorf(
			"actor runtime boundary audit: full production actor runtime claim is forbidden for P18.0",
		)
	}
	if !containsBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		return fmt.Errorf(
			"actor runtime boundary audit: missing full production actor runtime non-claim",
		)
	}
	expected := map[ActorRuntimeBoundaryID]bool{
		ActorRuntimeBoundaryCurrentLimits:        false,
		ActorRuntimeBoundarySchedulerPrototype:   false,
		ActorRuntimeBoundaryProductionAcceptance: false,
		ActorRuntimeBoundaryFullClaimBlockers:    false,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf(
			"actor runtime boundary audit: row count = %d, want %d",
			len(report.Rows),
			len(expected),
		)
	}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("actor runtime boundary audit: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("actor runtime boundary audit: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("actor runtime boundary audit: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf(
				"actor runtime boundary audit: row %q missing evidence or boundary",
				row.ID,
			)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("actor runtime boundary audit: missing row %q", id)
		}
	}
	rows := rowsByID(report.Rows)
	if err := validateCurrentLimitsRow(rows[ActorRuntimeBoundaryCurrentLimits]); err != nil {
		return err
	}
	if err := validateSchedulerPrototypeRow(rows[ActorRuntimeBoundarySchedulerPrototype]); err != nil {
		return err
	}
	if err := validateProductionAcceptanceRow(
		rows[ActorRuntimeBoundaryProductionAcceptance],
	); err != nil {
		return err
	}
	if err := validateFullClaimBlockersRow(rows[ActorRuntimeBoundaryFullClaimBlockers]); err != nil {
		return err
	}
	return nil
}

func currentActorRuntimeLimitsRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryCurrentLimits,
		Name:   "Current actor runtime limits",
		Status: ActorRuntimeBoundaryDocumentedLimit,
		RequiredFacts: []string{
			fmt.Sprintf("maxActors=%d", maxActors),
			fmt.Sprintf("msgPoolSize=%d", msgPoolSize),
			fmt.Sprintf("maxActorMailboxMsgs=%d", maxActorMailboxMsgs),
			fmt.Sprintf("actor_state_slots=%d", maxActorStateSlots),
			"single-thread cooperative scheduler documented for current actor runtime",
			"round-robin runnable actor fairness has bounded yield-progress evidence",
			"timed sleeping actors wake in deterministic deadline order",
			("linux-x64 distributed runtime only; non-Linux-x64 targets " +
				"keep distributed actor symbols out of the built-in runtime"),
			"non-linux actor net pump is no-op",
			"mailbox full returns checked -2 backpressure without allocating a message",
			"mailbox backpressure recovers after drain for local legacy, tagged, and typed sends",
			"typed mailbox backpressure does not enqueue a partial payload",
			"message pool exhaustion returns checked -1 without enqueueing an overflow message",
			"drained message pool entries are reclaimed after receive or actor exit and can be reused",
			"recycled message nodes scrub system kind and payload slots before free-list reuse",
			"invalid actor handle sends return checked -3 without allocating a message",
			"done actor sends return checked -4 without allocating a message",
			"nonzero actor entry return is exposed only as done-state send failure for later local sends",
			"local linked abnormal exits propagate the exit reason to non-trapping linked actors",
			"local actor_unlink removes bounded local abnormal-exit propagation",
			"no actor status, actor join, or actor exit-code API is exposed for done actors",
			"waited reclaimable actor slots reuse retained stack frames and reinitialize released stacks",
			"more than 10000 lifetime spawns work under the concurrent actor cap",
			"messages already queued in another actor mailbox remain receivable after the sender is done",
			"done actors are not restarted and pending mailbox nodes drain into the message-pool free list on actor exit",
			("blocked actors continue to depend on normal message, " +
				"deadline, timer, or task-wait readiness when another actor " +
				"exits"),
			"missing-node node_down remains checked distributed status evidence",
			("no automatic retry, restart, reconnect, or supervision is " +
				"claimed for local actor failure or distributed node_down " +
				"status"),
			"task-group cancellation wakes recv_until and recv_msg_until waiters with checked error 1",
			("task-group cancellation wakes actors already waiting on " +
				"task_join_result_i32, task_join_until_i32, and select2_i32 " +
				"with checked error 1"),
			("task_join_i32 wakes on task-group cancellation with raw " +
				"zero value; checked status requires result or timed join " +
				"APIs"),
			("task join, timed join, poll, and typed task join observers " +
				"treat reclaimable target slots as terminal result states"),
			("successful consuming task joins mark completed task actor " +
				"slots reclaimable after result read"),
			("more than 10000 joined task lifetime spawns work under the " +
				"concurrent actor cap"),
			("task_group_close treats joined reclaimable task actors as " +
				"terminal for group-close completion"),
			"typed task joins and result getters clear target/current result slots after use",
			"non-timed actor receives do not expose a cancellation result in the current profile",
			"typed actor message payloads are capped at 8 value slots",
		},
		Evidence: ("compiler/internal/actorsrt/actorsrt_core.go::BuildLinuxX64; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitMailboxFullCheckForReceiverInEcx; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitCheckedMessagePoolAlloc; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitRecycleMessageNodeInRax; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitInvalidActorHandleReturn; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitActorDoneReturn; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitBlockedDeadlineWakeCheck; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitWaitingTaskWakeCheck; " +
			"compiler/internal/actorsrt/actorsrt_core.go::" +
			"emitCurrentTaskGroupCanceledCheck; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorMailboxFullReturnsCheckedBackpressure; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRu" +
			"n; compiler/compiler_suite_test.go::" +
			"TestActorTaggedMailboxBackpressureRecoversAfterSelfDrainBuil" +
			"dAndRun; compiler/compiler_suite_test.go::" +
			"TestActorTypedMailboxBackpressureRecoversWithoutPartialPaylo" +
			"adBuildAndRun; compiler/compiler_suite_test.go::" +
			"TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorMessagePoolExhaustionReturnsCheckedFailure; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestRecycleMessageNodeScrubsPayloadSlotsBeforeFreeList; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorInvalidHandleSendReturnsCheckedFailure; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorSendToDoneActorReturnsCheckedFailure; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAnd" +
			"Run; compiler/compiler_suite_test.go::" +
			"TestActorLinkPropagatesAbnormalExitBuildAndRun; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorUnlinkStopsAbnormalExitPropagationBuildAndRun; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestLinuxRuntimeActorLinkUsesBidirectionalLinkTable; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestLinuxRuntimeActorUnlinkRemovesBidirectionalLinkTable; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorLifecycleReceivesPendingMessageFromDoneSenderBuildA" +
			"ndRun; compiler/compiler_suite_test.go::" +
			"TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlo" +
			"ckedActorsBuildAndRun; compiler/compiler_suite_test.go::" +
			"TestActorLifecycleDoneActorDrainsPendingMailboxIntoMessagePo" +
			"olBuildAndRun; compiler/compiler_suite_test.go::" +
			"TestActorLifetimeSpawnsExceedTenThousandUnderConcurrentCapB" +
			"uildAndRun; compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestLinuxRuntimeSpawnReusedReclaimableSlotsReinitializeReleased" +
			"Stacks; compiler/compiler_suite_test.go::" +
			"TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuild" +
			"AndRun; compiler/compiler_suite_test.go::" +
			"TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndR" +
			"un; cli/internal/actornet/broker_test.go::" +
			"TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect;" +
			" cli/internal/actornet/runtime_integration_test.go::" +
			"TestLinuxRuntimePumpsNodeDownIntoNodeStatus; " +
			"compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAnd" +
			"Run; compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuild" +
			"AndRun; compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuild" +
			"AndRun; compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValu" +
			"eBuildAndRun; compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun; " +
			"compiler/compiler_suite_test.go::" +
			"TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestEmitTaskJoinEntrypointsTreatReclaimableTargetsAsDone; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestEmitTaskJoinEntrypointsMarkDoneTargetsReclaimableAfter" +
			"ResultRead; compiler/compiler_suite_test.go::" +
			"TestTaskLifetimeSpawnsExceedTenThousandUnderConcurrentCap" +
			"BuildAndRun; compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestLinuxRuntimeTaskGroupCloseTreatsReclaimableActorsAsDone; " +
			"compiler/compiler_suite_test.go::" +
			"TestActorSendToCanceledActorReturnsCheckedFailure; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestEmitTaskJoinTypedClearsTargetResultSlotsAfterJoin; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestEmitTaskResultGetClearsCurrentStagedSlotAfterRead; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump; " +
			"compiler/internal/actorsrt/actorsrt_suite_test.go::" +
			"TestNonLinuxRuntimesDoNotExportDistributedActorSymbols; " +
			"docs/spec/runtime/actors.md::Lifecycle Matrix; " +
			"docs/spec/runtime/actors.md::Runtime Capacity Limits; " +
			"docs/spec/runtime/actors.md::Distributed Runtime Promotion " +
			"Surface; docs/spec/runtime/actors.md::Scheduling semantics"),
		Boundary: ("current evidence covers fixed-capacity x64 built-in actor " +
			"runtime behavior, cooperative round-robin bounded progress " +
			"for yielding runnable actors, deterministic deadline-order " +
			"wake for sleeping actors, recoverable checked per-actor " +
			"mailbox backpressure for local legacy/tagged/typed sends, " +
			"no partial typed payload after failed backpressure, " +
			"reusable drained message nodes with checked bounded " +
			"message-pool exhaustion for live overload, payload-slot " +
			"scrubbing before free-list reuse, stored stack " +
			"frame reset for waited reclaimable actor slots, more than 10000 " +
			"lifetime spawns under the concurrent cap, checked " +
			"invalid-handle and done-actor send failures, narrow " +
			"done-state lifecycle semantics where zero and nonzero actor " +
			"returns are user-visible only as done for later sends, " +
			"local linked abnormal-exit propagation and local unlink " +
			"removal of that propagation link, " +
			"consuming task joins marking completed task actors reclaimable " +
			"after result read, task_group_close treating joined " +
			"reclaimable task actors as terminal, task join/poll observers treating " +
			"reclaimable targets as terminal done-like result states, " +
			"typed task join/getter result-slot cleanup after successful use, " +
			"scoped cooperative task-group cancellation wake/error " +
			"behavior for timed actor receive and task join waiters, " +
			"Linux-x64 distributed node_down status evidence for " +
			"missing-node cases, Linux-x64 distributed actor runtime " +
			"symbols, and documented capacity limits; it does not " +
			"provide an unbounded mailbox, automatic retry/reconnect, " +
			"actor close/shutdown API, actor status/join/exit-code API, " +
			"cancellation results for non-timed actor receives, " +
			"monitor DOWN delivery, trap-exit message delivery, " +
			"supervision/restart/OTP lifecycle behavior, " +
			"preemptive or production multi-threaded scheduling, " +
			"non-Linux distributed runtime support, a full " +
			"structured-concurrency model, or a full production actor " +
			"runtime claim"),
	}
}

func schedulerPrototypeFeaturesRow(
	benchmarks []parallelrt.PrototypeBenchmark,
) ActorRuntimeBoundaryRow {
	var names []string
	for _, benchmark := range benchmarks {
		if benchmark.Ran && benchmark.Pass {
			names = append(names, benchmark.Name)
		}
	}
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundarySchedulerPrototype,
		Name:   "Scheduler prototype features",
		Status: ActorRuntimeBoundaryPrototypeEvidence,
		RequiredFacts: []string{
			"single-core FIFO compatibility",
			"two-core work stealing",
			"bounded typed mailbox with blocking_recv_yield backpressure metadata",
			"zero_copy_move owned-region transfer benchmark",
			"bytes_copied=0 for owned-region prototype transfer",
			"prototype benchmarks: " + strings.Join(names, "; "),
		},
		Evidence: ("compiler/internal/parallelrt/scheduler_model.go::" +
			"NewSchedulerModel; " +
			"compiler/internal/parallelrt/scheduler_model_test.go::" +
			"TestSchedulerModelRunsSingleCoreFIFO; " +
			"compiler/internal/parallelrt/scheduler_model_test.go::" +
			"TestSchedulerModelStealsWorkAcrossTwoCores; " +
			"compiler/internal/parallelrt/scheduler_model_test.go::" +
			"TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequire" +
			"sCopy; compiler/internal/parallelrt/scheduler_model_test.go:" +
			":TestPrototypeBenchmarksReportFanoutAndZeroCopyRows; " +
			"tools/cmd/parallel-production-smoke/main.go::" +
			"runSchedulerPrototypeEvidence"),
		Boundary: ("scheduler evidence is a checked model and release benchmark " +
			"row; it is not a production multi-threaded actor scheduler, " +
			"does not change compiler/runtime scheduling behavior, and " +
			"does not promote the built-in actor runtime beyond its " +
			"documented cooperative runtime boundary"),
	}
}

func productionRuntimeAcceptanceRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryProductionAcceptance,
		Name:   "Production runtime acceptance",
		Status: ActorRuntimeBoundaryAcceptanceRequired,
		RequiredFacts: []string{
			("production task scheduler evidence with executable fairness," +
				" wake, deadline, actor scheduler starvation/progress bound, " +
				"and stress gates"),
			"bounded mailbox backpressure with checked recoverable failure behavior",
			"message reclamation or checked exhaustion semantics for runtime message pools",
			"race-safety model or conservative rejection evidence across task/actor/thread boundaries",
			"cross-target distributed runtime gates for every claimed target",
			"blocking primitive by cancellation-source matrix covering wake and checked-error behavior",
			("structured concurrency and cancellation semantics beyond " +
				"the current cooperative task group handles"),
			("artifact-hash and validator gates that reject fake, " +
				"docs-only, metadata-only, and transport-only evidence"),
		},
		Evidence: ("tools/validators/parallelprod/report.go::validateContracts; " +
			"tools/validators/parallelprod/report.go::validateCases; " +
			"tools/validators/parallelprod/report.go::validateAudit; " +
			"tools/validators/actordist/report.go::ValidateReport; " +
			"docs/spec/runtime/actors.md::Distributed Runtime Promotion " +
			"Surface; docs/user/platform/async_actors_guide.md::Actors"),
		Boundary: ("acceptance criteria describe what a future production actor " +
			"runtime claim must prove; P18.0 records the criteria only " +
			"and does not mark those criteria satisfied for a full actor " +
			"runtime"),
	}
}

func fullClaimBlockersRow() ActorRuntimeBoundaryRow {
	return ActorRuntimeBoundaryRow{
		ID:     ActorRuntimeBoundaryFullClaimBlockers,
		Name:   "Full production actor runtime blockers",
		Status: ActorRuntimeBoundaryBlocked,
		MissingFacts: []string{
			"production multi-threaded actor scheduler integrated into the runtime",
			"non-Linux-x64 distributed actor runtime executable smoke and validator gates",
			"full cancellation and structured concurrency guarantees beyond cooperative task group handles",
			("full race-safety proof or audited conservative rejection " +
				"matrix for shared mutable actor/task/thread boundaries"),
			("production broker deployment, reconnect, ordering, retry, " +
				"and cluster membership evidence beyond loopback TCP smoke"),
		},
		Evidence: ("docs/spec/runtime/actors.md::Non-goals; " +
			"docs/spec/runtime/actors.md::Runtime Capacity Limits; " +
			"docs/user/platform/async_actors_guide.md::Actors; " +
			"docs/design/actor_region_transfer.md::P6.3 adds a checked " +
			"scheduler prototype model"),
		Boundary: ("these blockers keep the current evidence from becoming a " +
			"full production actor runtime claim; existing distributed " +
			"Linux-x64 and parallel production reports remain bounded " +
			"slices rather than proof of general actor-runtime " +
			"production completeness"),
	}
}

func validateCurrentLimitsRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryDocumentedLimit {
		return fmt.Errorf("actor runtime boundary audit: current limits status = %q", row.Status)
	}
	for _, fact := range []string{"maxActors=128", "msgPoolSize=65536", "maxActorMailboxMsgs=256", "actor_state_slots=8", "single-thread cooperative scheduler", (("round-robin runnable " +
		"actor fairness has bounded ") +
		"yield-progress evidence"), "timed sleeping actors wake in deterministic deadline order", "linux-x64 distributed runtime only", "non-linux actor net pump is no-op", "mailbox full returns checked -2", ("mailbox backpressure " +
		"recovers after drain"), (("typed mailbox " +
		"backpressure does not enqueue a partial ") +
		"payload"), "message pool exhaustion returns checked -1", ("drained message pool " +
		"entries are reclaimed"), ("invalid actor handle " +
		"sends return checked -3"), "done actor sends return checked -4", (("nonzero actor entry " +
		"return is exposed only as done-state ") +
		"send failure"), "no actor status, actor join, or actor exit-code API", (("messages already queued " +
		"in another actor mailbox remain ") +
		"receivable"), "local linked abnormal exits propagate", "done actors are not restarted", "waited reclaimable actor slots reuse retained stack frames", ("blocked actors continue to depend on " +
		"normal message"), (("missing-node node_down " +
		"remains checked distributed status ") +
		"evidence"), "no automatic retry, restart, reconnect, or supervision", ("task-group cancellation " +
		"wakes recv_until"), (("task-group cancellation " +
		"wakes actors already waiting on ") +
		"task_join_result_i32"), ("task_join_i32 wakes on task-group cancellation with raw " +
		"zero value"), "task join, timed join, poll, and typed task join observers treat reclaimable target slots", "non-timed actor receives do not expose a cancellation result"} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: current limits missing fact %q", fact)
		}
	}
	return nil
}

func validateSchedulerPrototypeRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryPrototypeEvidence {
		return fmt.Errorf(
			"actor runtime boundary audit: scheduler prototype status = %q, want prototype_evidence",
			row.Status,
		)
	}
	if strings.Contains(strings.ToLower(string(row.Status)), "production") {
		return fmt.Errorf(
			"actor runtime boundary audit: scheduler prototype must not be production-ready",
		)
	}
	for _, fact := range []string{
		"single-core FIFO compatibility",
		"two-core work stealing",
		"bounded typed mailbox",
		"zero_copy_move",
		"bytes_copied=0",
	} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"actor runtime boundary audit: scheduler prototype missing fact %q",
				fact,
			)
		}
	}
	if !strings.Contains(row.Boundary, "not a production multi-threaded actor scheduler") {
		return fmt.Errorf(
			"actor runtime boundary audit: scheduler prototype boundary must preserve production non-claim",
		)
	}
	return nil
}

func validateProductionAcceptanceRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryAcceptanceRequired {
		return fmt.Errorf(
			"actor runtime boundary audit: production acceptance status = %q",
			row.Status,
		)
	}
	for _, fact := range []string{
		"production task scheduler",
		"actor scheduler starvation/progress bound",
		"bounded mailbox backpressure",
		"message reclamation",
		"race-safety model",
		"cross-target distributed runtime gates",
		"blocking primitive by cancellation-source matrix",
		"structured concurrency",
	} {
		if !containsBoundaryText(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"actor runtime boundary audit: production acceptance missing fact %q",
				fact,
			)
		}
	}
	return nil
}

func validateFullClaimBlockersRow(row ActorRuntimeBoundaryRow) error {
	if row.Status != ActorRuntimeBoundaryBlocked {
		return fmt.Errorf("actor runtime boundary audit: blockers status = %q", row.Status)
	}
	if len(row.MissingFacts) == 0 {
		return fmt.Errorf("actor runtime boundary audit: blockers row must record missing facts")
	}
	for _, fact := range []string{
		"production multi-threaded actor scheduler",
		"non-Linux-x64 distributed actor runtime",
		"full cancellation and structured concurrency",
		"full race-safety proof",
	} {
		if !containsBoundaryText(row.MissingFacts, fact) {
			return fmt.Errorf("actor runtime boundary audit: blockers missing fact %q", fact)
		}
	}
	return nil
}

func rowsByID(rows []ActorRuntimeBoundaryRow) map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow {
	out := make(map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func containsBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

// ---- windows_x64.go ----

const (
	winImportVirtualAlloc = "kernel32.VirtualAlloc"
)

// BuildWindowsX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
// - __tetra_actor_send_msg / __tetra_actor_recv_msg
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in
// `core.spawn(...)`.
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

	if err := emitFunc(
		"__tetra_entry",
		func() error {
			return emitEntryWindowsX64(
				e,
				entries[0],
				&callPatches,
				&leaPatches,
				&importPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_switch_to",
		func() error { return emitSwitchToWindowsX64(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_yield",
		func() error { return emitActorYieldWindowsX64(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_yield_now_impl",
		func() error { return emitActorYieldNow(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_exit",
		func() error { return emitActorExitWindowsX64(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_trampoline",
		func() error { return emitActorTrampolineWindowsX64(e, &callPatches) },
	); err != nil {
		return nil, err
	}

	if err := emitFunc(
		"__tetra_actor_spawn_impl",
		func() error { return emitSpawnWindowsX64(e, &callPatches, &leaPatches, &importPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_impl", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_msg_impl",
		func() error { return emitSendMsg(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_begin_impl",
		func() error { return emitSendBegin(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_slot_impl",
		func() error { return emitSendSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_commit_impl",
		func() error { return emitSendCommit(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_net_pump",
		func() error { return emitActorNetPumpNoop(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_impl",
		func() error { return emitRecv(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_msg_impl",
		func() error { return emitRecvMsg(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_poll_impl",
		func() error { return emitRecvPoll(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_until_impl",
		func() error { return emitRecvUntil(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_msg_until_impl",
		func() error { return emitRecvMsgUntil(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_begin_impl",
		func() error { return emitRecvBegin(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_slot_impl",
		func() error { return emitRecvSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_count_impl",
		func() error { return emitRecvCount(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self_impl", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_sender_impl",
		func() error { return emitSender(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_load_impl",
		func() error { return emitActorStateLoad(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_store_impl",
		func() error { return emitActorStateStore(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_spawn_i32_impl",
		func() error { return emitTaskSpawnI32To(e, "__tetra_actor_spawn_impl", &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_open_impl",
		func() error { return emitTaskGroupOpen(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_close_impl",
		func() error { return emitTaskGroupClose(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_cancel_impl",
		func() error { return emitTaskGroupCancel(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_current_impl",
		func() error { return emitTaskGroupCurrent(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_status_impl",
		func() error { return emitTaskGroupStatus(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_is_canceled_impl",
		func() error { return emitTaskIsCanceled(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_checkpoint_impl",
		func() error { return emitTaskCheckpoint(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_time_now_ms_impl",
		func() error { return emitTimeNowMs(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_sleep_ms_impl",
		func() error { return emitSleepMs(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_sleep_until_ms_impl",
		func() error { return emitSleepUntilMs(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_deadline_ms_impl",
		func() error { return emitDeadlineMs(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_timer_ready_ms_impl",
		func() error { return emitTimerReadyMs(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_task_spawn_group_i32_impl", func() error {
		return emitTaskSpawnGroupI32(e, "__tetra_actor_spawn_impl", &callPatches)
	}); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_i32_impl",
		func() error { return emitTaskJoinI32(e, false, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_result_i32_impl",
		func() error { return emitTaskJoinI32(e, true, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_join_until_i32_impl",
		func() error { return emitTaskJoinUntilI32(e, &callPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_poll_i32_impl",
		func() error { return emitTaskPollI32(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_begin_impl",
		func() error { return emitTaskResultBegin(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_slot_impl",
		func() error { return emitTaskResultSlot(e) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_result_get_impl",
		func() error { return emitTaskResultGet(e) },
	); err != nil {
		return nil, err
	}
	for slots := 2; slots <= 8; slots++ {
		name := fmt.Sprintf("__tetra_task_join_typed_%d_impl", slots)
		slotCount := slots
		if err := emitFunc(
			name,
			func() error { return emitTaskJoinTyped(e, slotCount, &callPatches) },
		); err != nil {
			return nil, err
		}
	}

	if err := emitFunc(
		"__tetra_actor_spawn",
		func() error { return emitActorSpawnWrapperWindowsX64(e, &jmpPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send",
		func() error { return emitActorSendWrapperWindowsX64(e, &jmpPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_msg",
		func() error { return emitActorSendMsgWrapperWindowsX64(e, &jmpPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_begin",
		func() error { return emitActorSendBeginWrapperWindowsX64(e, &jmpPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_slot",
		func() error { return emitActorSendSlotWrapperWindowsX64(e, &jmpPatches) },
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_send_commit",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_send_commit_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_msg",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_msg_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_poll",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_poll_impl",
				&jmpPatches,
			)
		},
	); err != nil {
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
	if err := emitFunc(
		"__tetra_actor_recv_begin",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_begin_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_slot",
		func() error {
			return emitActorOneArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_slot_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_recv_count",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_recv_count_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_self",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_self_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_sender",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_sender_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_load",
		func() error {
			return emitActorOneArgWrapperWindowsX64(
				e,
				"__tetra_actor_state_load_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_state_store",
		func() error {
			return emitTaskTwoArgWrapperWindowsX64(
				e,
				"__tetra_actor_state_store_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_actor_yield_now",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_actor_yield_now_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_spawn_i32",
		func() error {
			return emitActorOneArgWrapperWindowsX64(
				e,
				"__tetra_task_spawn_i32_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_open",
		func() error {
			return emitActorNoArgWrapperWindowsX64(
				e,
				"__tetra_task_group_open_impl",
				&jmpPatches,
			)
		},
	); err != nil {
		return nil, err
	}
	if err := emitFunc(
		"__tetra_task_group_close",
		func() error {
			return emitActorOneArgWrapperWindowsX64(
				e,
				"__tetra_task_group_close_impl",
				&jmpPatches,
			)
		},
	); err != nil {
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
	if err := emitFunc(
		"__tetra_task_join_i32",
		func() error {
			return emitTaskTwoArgWrapperWindowsX64(
				e,
				"__tetra_task_join_i32_impl",
				&jmpPatches,
			)
		},
	); err != nil {
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
	if err := emitFunc(
		"__tetra_task_result_slot",
		func() error {
			return emitTaskTwoArgWrapperWindowsX64(
				e,
				"__tetra_task_result_slot_impl",
				&jmpPatches,
			)
		},
	); err != nil {
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
		if err := emitFunc(
			name,
			func() error { return emitTaskJoinTypedWrapperWindowsX64(e, slotCount, target, &jmpPatches) },
		); err != nil {
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
		relocs = append(
			relocs,
			tobj.Reloc{
				Kind:   tobj.RelocCallRel32,
				At:     uint32(patch.at),
				Name:   patch.name,
				Addend: 0,
			},
		)
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
		relocs = append(
			relocs,
			tobj.Reloc{
				Kind:   tobj.RelocIATDisp32,
				At:     uint32(patch.at),
				Name:   patch.name,
				Addend: 0,
			},
		)
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

// ---- windows_x64_emit.go ----

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

func emitInitActorStackWindowsX64(
	e *x64.Emitter,
	leaPatches *[]leaPatch,
	importPatches *[]importPatch,
) error {
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

func emitEntryWindowsX64(
	e *x64.Emitter,
	mainSymbol string,
	callPatches *[]callPatch,
	leaPatches *[]leaPatch,
	importPatches *[]importPatch,
) error {
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

func emitSpawnWindowsX64(
	e *x64.Emitter,
	callPatches *[]callPatch,
	leaPatches *[]leaPatch,
	importPatches *[]importPatch,
) error {
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

func emitActorOneArgWrapperWindowsX64(
	e *x64.Emitter,
	target string,
	jmpPatches *[]callPatch,
) error {
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

func emitTaskThreeArgWrapperWindowsX64(
	e *x64.Emitter,
	target string,
	jmpPatches *[]callPatch,
) error {
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

func emitTaskJoinTypedWrapperWindowsX64(
	e *x64.Emitter,
	slots int,
	target string,
	jmpPatches *[]callPatch,
) error {
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
