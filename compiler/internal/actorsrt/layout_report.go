package actorsrt

const actorSystemLayoutSchemaV1 = "tetra.actor.system_layout.v1"

type ActorSystemLayout struct {
	Schema      string            `json:"schema"`
	Target      string            `json:"target"`
	Runtime     string            `json:"runtime"`
	Actor       LayoutSection     `json:"actor"`
	Scheduler   LayoutSection     `json:"scheduler"`
	SystemEvent LayoutSection     `json:"system_event"`
	RawTypes    []RawTypeLayout   `json:"raw_types"`
	Invariants  []LayoutInvariant `json:"invariants"`
}

type LayoutSection struct {
	Name      string        `json:"name"`
	Size      int           `json:"size"`
	Alignment int           `json:"alignment"`
	Fields    []LayoutField `json:"fields"`
}

type LayoutField struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
	End    int    `json:"end"`
}

type RawTypeLayout struct {
	Name              string `json:"name"`
	Slots             int    `json:"slots"`
	RuntimeOwned      bool   `json:"runtime_owned"`
	UserConstructible bool   `json:"user_constructible"`
}

type LayoutInvariant struct {
	Name string `json:"name"`
	Pass bool   `json:"pass"`
}

func ActorSystemLayoutReport() ActorSystemLayout {
	return ActorSystemLayout{
		Schema:  actorSystemLayoutSchemaV1,
		Target:  "linux-x64",
		Runtime: "builtin-actor-runtime-v2",
		Actor: LayoutSection{
			Name:      "actor",
			Size:      actorSize,
			Alignment: 64,
			Fields: []LayoutField{
				layoutField("mailbox_head", actorMailboxHeadOff, 8),
				layoutField("mailbox_tail", actorMailboxTailOff, 8),
				layoutField("mailbox_count", actorMailboxCountOff, 4),
				layoutField("generation", actorGenerationOff, 4),
				layoutField("link_low_lane", actorLinkRef0Off, maxActorLinks*4),
				layoutField("link_high_lane", actorLinkHigh0Off, maxActorLinks*4),
				layoutField("system_mailbox_head", actorSystemMailboxHeadOff, 8),
				layoutField("system_mailbox_tail", actorSystemMailboxTailOff, 8),
				layoutField("system_mailbox_count", actorSystemMailboxCountOff, 4),
				layoutField("system_mailbox_reserved_credits", actorSystemMailboxReservedCreditsOff, 4),
				layoutField("system_mailbox_bytes", actorSystemMailboxBytesOff, 8),
				layoutField("system_mailbox_peak_bytes", actorSystemMailboxPeakBytesOff, 8),
				layoutField("system_mailbox_reclaimed_bytes", actorSystemMailboxReclaimedBytesOff, 8),
				layoutField("system_mailbox_overflow_attempts", actorSystemMailboxOverflowAttemptsOff, 8),
				layoutField("system_recv_scratch", actorSystemRecvScratch0Off, 7*8),
				layoutField("system_recv_scratch_count", actorSystemRecvScratchCountOff, 4),
				layoutField("system_recv_scratch_status", actorSystemRecvScratchStatusOff, 4),
				layoutField("wait_kind", actorWaitKindOff, 4),
				layoutField("terminal_reason_kind", actorTerminalReasonKindOff, 4),
				layoutField("terminal_reason_code", actorTerminalReasonCodeOff, 4),
			},
		},
		Scheduler: LayoutSection{
			Name:      "scheduler",
			Size:      schedSize,
			Alignment: 64,
			Fields: []LayoutField{
				layoutField("remote_generation_table", schedRemoteGeneration0Off, maxActors*4),
				layoutField("node_epoch_table", schedNodeEpoch0Off, maxActors*4),
				layoutField("monitor_id_table", schedMonitorID0Off, maxActorMonitors*4),
				layoutField("monitor_owner_ref_table", schedMonitorOwnerRef0Off, maxActorMonitors*4),
				layoutField("monitor_target_ref_table", schedMonitorTargetRef0Off, maxActorMonitors*4),
				layoutField("monitor_target_high_table", schedMonitorTargetHigh0Off, maxActorMonitors*4),
				layoutField("system_event_base", schedSystemEventBaseOff, 8),
				layoutField("system_event_bump", schedSystemEventBumpOff, 8),
				layoutField("system_event_end", schedSystemEventEndOff, 8),
				layoutField("system_event_free", schedSystemEventFreeOff, 8),
				layoutField("system_event_capacity_bytes", schedSystemEventCapacityBytesOff, 8),
				layoutField("system_event_live_bytes", schedSystemEventLiveBytesOff, 8),
				layoutField("system_event_peak_bytes", schedSystemEventPeakBytesOff, 8),
				layoutField("system_event_reclaimed_bytes", schedSystemEventReclaimedBytesOff, 8),
				layoutField("system_event_alloc_failures", schedSystemEventAllocFailuresOff, 8),
				layoutField("system_event_reserved_credits", schedSystemEventReservedCreditsOff, 8),
				layoutField("runtime_closing", schedRuntimeClosingOff, 4),
			},
		},
		SystemEvent: LayoutSection{
			Name:      "system_event",
			Size:      systemEventSize,
			Alignment: 8,
			Fields: []LayoutField{
				layoutField("next", systemEventNextOff, 8),
				layoutField("kind", systemEventKindOff, 4),
				layoutField("flags", systemEventFlagsOff, 4),
				layoutField("subject", systemEventSubjectOff, 8),
				layoutField("monitor_ref", systemEventMonitorRefOff, 8),
				layoutField("node_id", systemEventNodeIDOff, 8),
				layoutField("node_epoch", systemEventNodeEpochOff, 8),
				layoutField("reason_kind", systemEventReasonKindOff, 4),
				layoutField("reason_code", systemEventReasonCodeOff, 4),
				layoutField("dedup_key", systemEventDedupKeyOff, 8),
			},
		},
		RawTypes: []RawTypeLayout{
			{Name: "actor.monitor", Slots: 1, RuntimeOwned: true, UserConstructible: false},
			{Name: "actor.node", Slots: 2, RuntimeOwned: true, UserConstructible: false},
			{Name: "actor.system_recv_raw", Slots: 8, RuntimeOwned: true, UserConstructible: false},
		},
		Invariants: []LayoutInvariant{
			{
				Name: "actor_system_mailbox_within_actor",
				Pass: actorSystemMailboxHeadOff >= 0 &&
					actorTerminalReasonCodeOff+4 <= actorSize,
			},
			{
				Name: "scheduler_system_event_pool_fields_ordered",
				Pass: schedSystemEventBaseOff < schedSystemEventBumpOff &&
					schedSystemEventBumpOff < schedSystemEventEndOff &&
					schedSystemEventEndOff < schedSystemEventFreeOff &&
					schedSystemEventReservedCreditsOff < schedSize,
			},
			{
				Name: "system_event_layout_separate_from_user_message",
				Pass: systemEventSize != msgSize,
			},
		},
	}
}

func layoutField(name string, off int32, size int) LayoutField {
	offset := int(off)
	return LayoutField{
		Name:   name,
		Offset: offset,
		Size:   size,
		End:    offset + size,
	}
}
