package actorsrt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/format/tobj"
)

// ---- actor_runtime_byte_counters_test.go ----

func TestActorRuntimeByteCounterLayoutIsStable(t *testing.T) {
	if msgSystemKindOff != msgCountOff+4 {
		t.Fatalf(
			"msgSystemKindOff=%d, want densely packed after msgCountOff=%d",
			msgSystemKindOff,
			msgCountOff,
		)
	}
	if msgPayload0 != msgSystemKindOff+8 {
		t.Fatalf(
			"msgPayload0=%d, want u64-aligned after msgSystemKindOff=%d",
			msgPayload0,
			msgSystemKindOff,
		)
	}
	if msgSize != msgPayload0+8*msgSlotSize {
		t.Fatalf(
			"msgSize=%d, want message payload ending at %d",
			msgSize,
			msgPayload0+8*msgSlotSize,
		)
	}
	if actorSize < actorBackpressureEventsOff+8 {
		t.Fatalf(
			"actorSize=%d does not cover backpressure events offset %d",
			actorSize,
			actorBackpressureEventsOff,
		)
	}
	if actorMailboxBytesOff%8 != 0 || actorMailboxPeakBytesOff%8 != 0 ||
		actorReclaimedBytesOff%8 != 0 ||
		actorBytesCopiedOff%8 != 0 ||
		actorCopyCountOff%8 != 0 ||
		actorByteBudgetOff%8 != 0 ||
		actorOverBudgetCountOff%8 != 0 ||
		actorBackpressureEventsOff%8 != 0 {
		t.Fatalf("actor byte counter offsets must remain u64-aligned")
	}
	if maxActorMailboxBytes != maxActorMailboxMsgs*msgSize {
		t.Fatalf(
			"maxActorMailboxBytes=%d, want maxActorMailboxMsgs*msgSize=%d",
			maxActorMailboxBytes,
			maxActorMailboxMsgs*msgSize,
		)
	}
	if schedMsgPoolAllocFailuresOff != schedMsgPoolReclaimedBytesOff+8 {
		t.Fatalf("scheduler message-pool counters must remain densely packed")
	}
}

func TestRecycleMessageNodeScrubsPayloadSlotsBeforeFreeList(t *testing.T) {
	e := &x64.Emitter{}
	emitRecycleMessageNodeInRax(e)

	for _, off := range []int32{
		msgSenderOff,
		msgValueOff,
		msgTagOff,
		msgCountOff,
		msgSystemKindOff,
	} {
		if !bytes.Contains(e.Buf, movMem32RdiDispEaxEncoding(off)) {
			t.Fatalf("recycle node missing zero store for message field offset %d", off)
		}
	}
	for slot := 0; slot < 8; slot++ {
		off := msgPayload0 + int32(slot*msgSlotSize)
		if !bytes.Contains(e.Buf, movMem64RdiDispRaxEncoding(off)) {
			t.Fatalf("recycle node missing zero store for payload slot %d offset %d", slot, off)
		}
	}
}

func TestActorRuntimeV2IdentityLayoutIsStable(t *testing.T) {
	if actorGenerationOff != actorBackpressureEventsOff+8 {
		t.Fatalf(
			"actorGenerationOff=%d, want densely packed after actorBackpressureEventsOff=%d",
			actorGenerationOff,
			actorBackpressureEventsOff,
		)
	}
	if actorSize < actorGenerationOff+4 {
		t.Fatalf("actorSize=%d does not cover generation offset %d", actorSize, actorGenerationOff)
	}
}

func TestActorRuntimeLinkLayoutIsStable(t *testing.T) {
	if actorLinkCountOff != actorStackInitRspOff+8 {
		t.Fatalf(
			"actorLinkCountOff=%d, want densely packed after actorStackInitRspOff=%d",
			actorLinkCountOff,
			actorStackInitRspOff,
		)
	}
	if actorLinkRef0Off != actorLinkCountOff+4 {
		t.Fatalf(
			"actorLinkRef0Off=%d, want densely packed after actorLinkCountOff=%d",
			actorLinkRef0Off,
			actorLinkCountOff,
		)
	}
	if actorLinkHigh0Off != actorLinkRef0Off+maxActorLinks*4 {
		t.Fatalf(
			"actorLinkHigh0Off=%d, want high lane after low refs ending at %d",
			actorLinkHigh0Off,
			actorLinkRef0Off+maxActorLinks*4,
		)
	}
	if actorSize < actorLinkHigh0Off+maxActorLinks*4 {
		t.Fatalf(
			"actorSize=%d does not cover link ref high lane ending at %d",
			actorSize,
			actorLinkHigh0Off+maxActorLinks*4,
		)
	}
}

func TestActorRuntimeRemoteGenerationLayoutIsStable(t *testing.T) {
	if schedRemoteGeneration0Off != schedActorWait0Off+maxActors*4 {
		t.Fatalf(
			"schedRemoteGeneration0Off=%d, want densely packed after schedActorWait0Off=%d",
			schedRemoteGeneration0Off,
			schedActorWait0Off,
		)
	}
	if schedNodeEpoch0Off != schedRemoteGeneration0Off+maxActors*4 {
		t.Fatalf(
			"schedNodeEpoch0Off=%d, want densely packed after remote generation table ending at %d",
			schedNodeEpoch0Off,
			schedRemoteGeneration0Off+maxActors*4,
		)
	}
	if schedMonitorNextIDOff != schedNodeEpoch0Off+maxActors*4 {
		t.Fatalf(
			"schedMonitorNextIDOff=%d, want densely packed after remote node epoch table ending at %d",
			schedMonitorNextIDOff,
			schedNodeEpoch0Off+maxActors*4,
		)
	}
}

func TestActorRuntimeMonitorLayoutIsStable(t *testing.T) {
	if schedMonitorCountOff != schedMonitorNextIDOff+4 {
		t.Fatalf(
			"schedMonitorCountOff=%d, want densely packed after schedMonitorNextIDOff=%d",
			schedMonitorCountOff,
			schedMonitorNextIDOff,
		)
	}
	if schedMonitorID0Off != schedMonitorCountOff+4 {
		t.Fatalf(
			"schedMonitorID0Off=%d, want densely packed after schedMonitorCountOff=%d",
			schedMonitorID0Off,
			schedMonitorCountOff,
		)
	}
	if schedMonitorOwnerRef0Off != schedMonitorID0Off+maxActorMonitors*4 {
		t.Fatalf(
			"schedMonitorOwnerRef0Off=%d, want densely packed after monitor ids ending at %d",
			schedMonitorOwnerRef0Off,
			schedMonitorID0Off+maxActorMonitors*4,
		)
	}
	if schedMonitorTargetRef0Off != schedMonitorOwnerRef0Off+maxActorMonitors*4 {
		t.Fatalf(
			"schedMonitorTargetRef0Off=%d, want densely packed after owner refs ending at %d",
			schedMonitorTargetRef0Off,
			schedMonitorOwnerRef0Off+maxActorMonitors*4,
		)
	}
	if schedMonitorTargetHigh0Off != schedMonitorTargetRef0Off+maxActorMonitors*4 {
		t.Fatalf(
			"schedMonitorTargetHigh0Off=%d, want densely packed after target low refs ending at %d",
			schedMonitorTargetHigh0Off,
			schedMonitorTargetRef0Off+maxActorMonitors*4,
		)
	}
	if schedSystemEventBaseOff != schedMonitorTargetHigh0Off+maxActorMonitors*4 {
		t.Fatalf(
			"schedSystemEventBaseOff=%d, want after monitor target high refs ending at %d",
			schedSystemEventBaseOff,
			schedMonitorTargetHigh0Off+maxActorMonitors*4,
		)
	}
}

func TestActorRuntimeSystemMessageQueueLayoutIsStable(t *testing.T) {
	if systemEventNextOff != 0 ||
		systemEventKindOff != 8 ||
		systemEventFlagsOff != 12 ||
		systemEventSubjectOff != 16 ||
		systemEventMonitorRefOff != 24 ||
		systemEventNodeIDOff != 32 ||
		systemEventNodeEpochOff != 40 ||
		systemEventReasonKindOff != 48 ||
		systemEventReasonCodeOff != 52 ||
		systemEventDedupKeyOff != 56 ||
		systemEventSize != 64 {
		t.Fatalf("system event layout drifted")
	}
	if systemEventSize == msgSize {
		t.Fatalf("system events must not share the ordinary user message node layout")
	}
	if actorSystemMailboxHeadOff != actorLinkHigh0Off+maxActorLinks*4+4 {
		t.Fatalf(
			"actorSystemMailboxHeadOff=%d, want u64-aligned after link high lane ending at %d",
			actorSystemMailboxHeadOff,
			actorLinkHigh0Off+maxActorLinks*4,
		)
	}
	if actorSystemMailboxHeadOff%8 != 0 {
		t.Fatalf("actorSystemMailboxHeadOff=%d must be u64-aligned", actorSystemMailboxHeadOff)
	}
	if actorSystemMailboxTailOff != actorSystemMailboxHeadOff+8 {
		t.Fatalf("actorSystemMailboxTailOff=%d, want after head", actorSystemMailboxTailOff)
	}
	if actorSystemMailboxCountOff != actorSystemMailboxTailOff+8 {
		t.Fatalf("actorSystemMailboxCountOff=%d, want after tail", actorSystemMailboxCountOff)
	}
	if actorSystemMailboxReservedCreditsOff != actorSystemMailboxCountOff+4 {
		t.Fatalf(
			"actorSystemMailboxReservedCreditsOff=%d, want after system count",
			actorSystemMailboxReservedCreditsOff,
		)
	}
	if actorSystemMailboxBytesOff != actorSystemMailboxReservedCreditsOff+4 {
		t.Fatalf("actorSystemMailboxBytesOff=%d, want u64-aligned after reserved credits", actorSystemMailboxBytesOff)
	}
	if actorSystemMailboxPeakBytesOff != actorSystemMailboxBytesOff+8 ||
		actorSystemMailboxReclaimedBytesOff != actorSystemMailboxPeakBytesOff+8 ||
		actorSystemMailboxOverflowAttemptsOff != actorSystemMailboxReclaimedBytesOff+8 {
		t.Fatalf("system mailbox byte counters must remain densely packed")
	}
	if actorSystemRecvScratch0Off != actorSystemMailboxOverflowAttemptsOff+8 {
		t.Fatalf("actorSystemRecvScratch0Off=%d, want after overflow attempts", actorSystemRecvScratch0Off)
	}
	if actorSystemRecvScratchCountOff != actorSystemRecvScratch0Off+7*8 {
		t.Fatalf("actorSystemRecvScratchCountOff=%d, want after seven raw slots", actorSystemRecvScratchCountOff)
	}
	if actorSystemRecvScratchStatusOff != actorSystemRecvScratchCountOff+4 {
		t.Fatalf("actorSystemRecvScratchStatusOff=%d, want after scratch count", actorSystemRecvScratchStatusOff)
	}
	if actorWaitKindOff != actorSystemRecvScratchStatusOff+4 {
		t.Fatalf("actorWaitKindOff=%d, want after system recv status", actorWaitKindOff)
	}
	if actorTerminalReasonKindOff != actorWaitKindOff+4 ||
		actorTerminalReasonCodeOff != actorTerminalReasonKindOff+4 {
		t.Fatalf("terminal reason fields must remain densely packed after wait kind")
	}
	if actorSize < actorTerminalReasonCodeOff+4 {
		t.Fatalf("actorSize=%d does not cover terminal reason code offset %d", actorSize, actorTerminalReasonCodeOff)
	}
	if schedSystemEventBaseOff != schedMonitorTargetHigh0Off+maxActorMonitors*4 {
		t.Fatalf(
			"schedSystemEventBaseOff=%d, want after monitor target high refs ending at %d",
			schedSystemEventBaseOff,
			schedMonitorTargetHigh0Off+maxActorMonitors*4,
		)
	}
	if schedSystemEventBumpOff != schedSystemEventBaseOff+8 ||
		schedSystemEventEndOff != schedSystemEventBumpOff+8 ||
		schedSystemEventFreeOff != schedSystemEventEndOff+8 ||
		schedSystemEventCapacityBytesOff != schedSystemEventFreeOff+8 ||
		schedSystemEventLiveBytesOff != schedSystemEventCapacityBytesOff+8 ||
		schedSystemEventPeakBytesOff != schedSystemEventLiveBytesOff+8 ||
		schedSystemEventReclaimedBytesOff != schedSystemEventPeakBytesOff+8 ||
		schedSystemEventAllocFailuresOff != schedSystemEventReclaimedBytesOff+8 ||
		schedSystemEventReservedCreditsOff != schedSystemEventAllocFailuresOff+8 {
		t.Fatalf("scheduler system-event pool fields must remain densely packed")
	}
	if schedRuntimeClosingOff != schedSystemEventReservedCreditsOff+8 {
		t.Fatalf("schedRuntimeClosingOff=%d, want after system-event reserved credits", schedRuntimeClosingOff)
	}
	if schedSize != schedRuntimeClosingOff+8 {
		t.Fatalf("schedSize=%d, want 8-byte-aligned size after runtime closing flag", schedSize)
	}
}

func TestActorSystemLayoutReportCapturesRuntimeOwnedSystemLane(t *testing.T) {
	report := ActorSystemLayoutReport()
	if report.Schema != "tetra.actor.system_layout.v1" {
		t.Fatalf("schema = %q, want tetra.actor.system_layout.v1", report.Schema)
	}
	if report.Target != "linux-x64" || report.Runtime != "builtin-actor-runtime-v2" {
		t.Fatalf("target/runtime = %s/%s, want linux-x64/builtin-actor-runtime-v2", report.Target, report.Runtime)
	}
	if report.Actor.Size != actorSize || report.Actor.Alignment != 64 {
		t.Fatalf("actor size/alignment = %d/%d, want %d/64", report.Actor.Size, report.Actor.Alignment, actorSize)
	}
	for _, field := range []string{
		"system_mailbox_head",
		"system_mailbox_tail",
		"system_mailbox_count",
		"system_recv_scratch",
		"wait_kind",
	} {
		if !layoutReportHasField(report.Actor.Fields, field) {
			t.Fatalf("actor layout report missing field %s", field)
		}
	}
	for _, field := range []string{
		"system_event_base",
		"system_event_live_bytes",
		"system_event_reserved_credits",
		"runtime_closing",
	} {
		if !layoutReportHasField(report.Scheduler.Fields, field) {
			t.Fatalf("scheduler layout report missing field %s", field)
		}
	}
	if report.SystemEvent.Size != systemEventSize {
		t.Fatalf("system event size = %d, want %d", report.SystemEvent.Size, systemEventSize)
	}
	if !layoutReportHasField(report.SystemEvent.Fields, "node_epoch") {
		t.Fatalf("system event layout report missing node_epoch")
	}
	if !layoutReportHasOpaqueRawType(report.RawTypes, "actor.node", 2) {
		t.Fatalf("layout report missing two-slot actor.node raw type")
	}
	if !layoutReportHasOpaqueRawType(report.RawTypes, "actor.system_recv_raw", 8) {
		t.Fatalf("layout report missing eight-slot actor.system_recv_raw type")
	}
	for _, invariant := range []string{
		"actor_system_mailbox_within_actor",
		"scheduler_system_event_pool_fields_ordered",
		"system_event_layout_separate_from_user_message",
	} {
		if !layoutReportHasPassingInvariant(report.Invariants, invariant) {
			t.Fatalf("layout report missing passing invariant %s", invariant)
		}
	}
}

func layoutReportHasField(fields []LayoutField, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func layoutReportHasOpaqueRawType(types []RawTypeLayout, name string, slots int) bool {
	for _, typ := range types {
		if typ.Name == name && typ.Slots == slots && typ.RuntimeOwned && !typ.UserConstructible {
			return true
		}
	}
	return false
}

func layoutReportHasPassingInvariant(invariants []LayoutInvariant, name string) bool {
	for _, invariant := range invariants {
		if invariant.Name == name && invariant.Pass {
			return true
		}
	}
	return false
}

func TestLinuxRuntimeUserAndSystemMailboxWakeKindsAreSeparated(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	userWaitKindStore := movMem32RdiDispImm32Encoding(actorWaitKindOff, actorWaitKindUser)
	for _, name := range []string{
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !bytes.Contains(body, userWaitKindStore) {
			t.Fatalf("%s blocks on ordinary mailbox without actorWaitKindUser", name)
		}
	}

	userWakeFragments := [][]byte{
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(statusBlocked),
		movEaxFromRdiDispEncoding(actorWaitKindOff),
		cmpEaxImm32Encoding(actorWaitKindUser),
		movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
		movMem32RdiDispImm32Encoding(actorWaitKindOff, actorWaitKindNone),
	}
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !containsFragmentsInOrder(body, userWakeFragments...) {
			t.Fatalf("%s can wake a blocked actor without proving actorWaitKindUser", name)
		}
	}

	systemEnqueue, ok := symbolBody(obj, "__tetra_actor_exit")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_exit")
	}
	systemWakeFragments := [][]byte{
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(statusBlocked),
		movEaxFromRdiDispEncoding(actorWaitKindOff),
		cmpEaxImm32Encoding(actorWaitKindSystem),
		movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
		movMem32RdiDispImm32Encoding(actorWaitKindOff, actorWaitKindNone),
	}
	if !containsFragmentsInOrder(systemEnqueue, systemWakeFragments...) {
		t.Fatalf("system enqueue path does not prove actorWaitKindSystem before waking")
	}
}

func TestLinuxRuntimeSystemReceiveReturnsCanceledOnTaskGroupCancellation(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_recv_system_begin")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_recv_system_begin")
	}
	canceledFragments := [][]byte{
		cmpEaxImm32Encoding(taskGroupCanceled),
		movMem32RdiDispImm32Encoding(actorSystemRecvScratchStatusOff, actorSystemRecvStatusCanceled),
		movMem32RdiDispImm32Encoding(actorSystemRecvScratchCountOff, 0),
		movMem32RdiDispImm32Encoding(actorWaitKindOff, actorWaitKindNone),
		movEaxImm32Encoding(actorSystemRecvStatusCanceled),
	}
	if !containsFragmentsInOrder(body, canceledFragments...) {
		t.Fatalf("__tetra_actor_recv_system_begin missing canceled result path")
	}
}

func TestLinuxRuntimeSystemReceiveReturnsRuntimeClosedOnSchedulerShutdown(t *testing.T) {
	source := readActorsRTCoreSource(t)
	entrySource := functionSource(t, source, "emitEntry")
	for _, snippet := range []string{
		"schedRuntimeClosingOff",
		"emitMarkRuntimeClosingAndWakeSystemWaiters",
	} {
		if !strings.Contains(entrySource, snippet) {
			t.Fatalf("emitEntry missing scheduler runtime-close snippet %q", snippet)
		}
	}
	recvSystemSource := functionSource(t, source, "emitRecvSystemBegin")
	for _, snippet := range []string{
		"schedRuntimeClosingOff",
		"actorSystemRecvStatusRuntimeClosed",
	} {
		if !strings.Contains(recvSystemSource, snippet) {
			t.Fatalf("emitRecvSystemBegin missing runtime_closed snippet %q", snippet)
		}
	}

	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_recv_system_begin")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_recv_system_begin")
	}
	runtimeClosedFragments := [][]byte{
		movMem32RdiDispImm32Encoding(actorSystemRecvScratchStatusOff, actorSystemRecvStatusRuntimeClosed),
		movMem32RdiDispImm32Encoding(actorSystemRecvScratchCountOff, 0),
		movMem32RdiDispImm32Encoding(actorWaitKindOff, actorWaitKindNone),
		movEaxImm32Encoding(actorSystemRecvStatusRuntimeClosed),
	}
	if !containsFragmentsInOrder(body, runtimeClosedFragments...) {
		t.Fatalf("__tetra_actor_recv_system_begin missing runtime_closed result path")
	}
}

func TestLinuxRuntimeSystemEventReservationCreditsAreAccounted(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	monitor, ok := symbolBody(obj, "__tetra_actor_monitor")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_monitor")
	}
	demonitor, ok := symbolBody(obj, "__tetra_actor_demonitor")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_demonitor")
	}
	link, ok := symbolBody(obj, "__tetra_actor_link")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_link")
	}
	unlink, ok := symbolBody(obj, "__tetra_actor_unlink")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_unlink")
	}
	recvSystem, ok := symbolBody(obj, "__tetra_actor_recv_system_begin")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_recv_system_begin")
	}

	maxCredits := int32(systemEventPoolSize / systemEventSize)
	for label, body := range map[string][]byte{
		"__tetra_actor_monitor": monitor,
		"__tetra_actor_link":    link,
	} {
		if !containsFragmentsInOrder(
			body,
			movRaxFromRdiDispEncoding(schedSystemEventReservedCreditsOff),
			cmpRaxImm32Encoding(maxCredits),
			addRaxImm32Encoding(1),
			movMem64RdiDispRaxEncoding(schedSystemEventReservedCreditsOff),
		) {
			t.Fatalf("%s missing bounded scheduler system-event credit reservation", label)
		}
		if !containsFragmentsInOrder(
			body,
			movEaxFromRdiDispEncoding(actorSystemMailboxReservedCreditsOff),
			addEaxImm32Encoding(1),
			movMem32RdiDispEaxEncoding(actorSystemMailboxReservedCreditsOff),
		) {
			t.Fatalf("%s missing actor system mailbox reserved-credit increment", label)
		}
	}

	for label, body := range map[string][]byte{
		"__tetra_actor_demonitor":         demonitor,
		"__tetra_actor_unlink":            unlink,
		"__tetra_actor_recv_system_begin": recvSystem,
	} {
		if !containsFragmentsInOrder(
			body,
			movEaxFromRdiDispEncoding(actorSystemMailboxReservedCreditsOff),
			addEaxImm32Encoding(-1),
			movMem32RdiDispEaxEncoding(actorSystemMailboxReservedCreditsOff),
		) {
			t.Fatalf("%s missing actor system mailbox reserved-credit release", label)
		}
		if !containsFragmentsInOrder(
			body,
			movRaxFromRdiDispEncoding(schedSystemEventReservedCreditsOff),
			addRaxImm32Encoding(-1),
			movMem64RdiDispRaxEncoding(schedSystemEventReservedCreditsOff),
		) {
			t.Fatalf("%s missing scheduler system-event reserved-credit release", label)
		}
	}
	if !containsFragmentsInOrder(
		recvSystem,
		movRaxFromRdiDispEncoding(actorSystemMailboxBytesOff),
		addRaxImm32Encoding(-systemEventSize),
		movMem64RdiDispRaxEncoding(actorSystemMailboxBytesOff),
	) {
		t.Fatalf("__tetra_actor_recv_system_begin missing system mailbox byte return on consume")
	}
}

func TestLinuxRuntimeMonitorDownUsesUnforgeableSystemMessageLane(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	exit, ok := symbolBody(obj, "__tetra_actor_exit")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_exit")
	}

	for label, seq := range map[string][]byte{
		"monitor table count scan":     movEaxFromRdiDispEncoding(schedMonitorCountOff),
		"monitor target ref table":     addRaxImm32Encoding(schedMonitorTargetRef0Off),
		"monitor owner ref table":      addRaxImm32Encoding(schedMonitorOwnerRef0Off),
		"monitor id table":             addRaxImm32Encoding(schedMonitorID0Off),
		"system DOWN kind store":       movMem32RdiDispImm32Encoding(systemEventKindOff, actorSystemKindDown),
		"system queue tail load":       movRaxFromRdiDispEncoding(actorSystemMailboxTailOff),
		"system queue head store":      movMem64RdiDispRaxEncoding(actorSystemMailboxHeadOff),
		"system queue tail store":      movMem64RdiDispRaxEncoding(actorSystemMailboxTailOff),
		"monitor count decrement":      addEaxImm32Encoding(-1),
		"monitor count write-back":     movMem32RdiDispEaxEncoding(schedMonitorCountOff),
		"monitor entry compaction":     movMem32RaxPtrEcxEncoding(),
		"blocked monitor owner wakeup": movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
	} {
		if !bytes.Contains(exit, seq) {
			t.Fatalf("__tetra_actor_exit missing %s sequence % x", label, seq)
		}
	}

	userKindStore := movMem32RdiDispImm32Encoding(msgSystemKindOff, actorSystemKindUser)
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !bytes.Contains(body, userKindStore) {
			t.Fatalf("%s does not reset msg.system_kind to actorSystemKindUser", name)
		}
		if bytes.Contains(body, movMem32RdiDispImm32Encoding(msgSystemKindOff, actorSystemKindDown)) {
			t.Fatalf("%s can forge actorSystemKindDown on the ordinary user message path", name)
		}
	}
}

func TestLinuxRuntimeTrapExitUsesUnforgeableSystemMessageLane(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	exit, ok := symbolBody(obj, "__tetra_actor_exit")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_exit")
	}

	for label, seq := range map[string][]byte{
		"linked trap-exit check":     movEaxFromRdiDispEncoding(actorTrapExitOff),
		"system EXIT kind store":     movMem32RdiDispImm32Encoding(systemEventKindOff, actorSystemKindExit),
		"system queue tail load":     movRaxFromRdiDispEncoding(actorSystemMailboxTailOff),
		"system queue head store":    movMem64RdiDispRaxEncoding(actorSystemMailboxHeadOff),
		"system queue tail store":    movMem64RdiDispRaxEncoding(actorSystemMailboxTailOff),
		"blocked linked actor wake":  movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
		"normal linked actor killed": movMem32RdiDispImm32Encoding(actorStatusOff, statusDone),
	} {
		if !bytes.Contains(exit, seq) {
			t.Fatalf("__tetra_actor_exit missing %s sequence % x", label, seq)
		}
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if bytes.Contains(body, movMem32RdiDispImm32Encoding(msgSystemKindOff, actorSystemKindExit)) {
			t.Fatalf("%s can forge actorSystemKindExit on the ordinary user message path", name)
		}
	}
}

func TestLinuxRuntimeNodeDownNotifiesRemoteMonitorOwners(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	netPump, ok := symbolBody(obj, "__tetra_actor_net_pump")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_net_pump")
	}

	for label, seq := range map[string][]byte{
		"node_down frame branch":      cmpEaxImm32Encoding(actorWireFrameNodeDown),
		"monitor count scan":          movEaxFromRdiDispEncoding(schedMonitorCountOff),
		"monitor target high table":   addRaxImm32Encoding(schedMonitorTargetHigh0Off),
		"remote target high mask":     andEaxImm32Encoding(actorRefRemoteHighMask),
		"remote target high base":     cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"monitor target low table":    addRaxImm32Encoding(schedMonitorTargetRef0Off),
		"monitor owner table":         addRaxImm32Encoding(schedMonitorOwnerRef0Off),
		"monitor id table":            addRaxImm32Encoding(schedMonitorID0Off),
		"system NODE_DOWN kind store": movMem32RdiDispImm32Encoding(systemEventKindOff, actorSystemKindNodeDown),
		"system queue tail load":      movRaxFromRdiDispEncoding(actorSystemMailboxTailOff),
		"system queue head store":     movMem64RdiDispRaxEncoding(actorSystemMailboxHeadOff),
		"system queue tail store":     movMem64RdiDispRaxEncoding(actorSystemMailboxTailOff),
		"monitor entry compaction":    movMem32RaxPtrEcxEncoding(),
		"blocked owner wakeup":        movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
	} {
		if !bytes.Contains(netPump, seq) {
			t.Fatalf("__tetra_actor_net_pump missing remote monitor node_down %s sequence % x", label, seq)
		}
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if bytes.Contains(body, movMem32RdiDispImm32Encoding(msgSystemKindOff, actorSystemKindNodeDown)) {
			t.Fatalf("%s can forge actorSystemKindNodeDown on the ordinary user message path", name)
		}
	}
}

func TestLinuxRuntimeNodeDownPropagatesRemoteLinkEntries(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(
			repoRootFromActorsRTTest(t),
			"compiler",
			"internal",
			"actorsrt",
			"actorsrt_core.go",
		),
	)
	if err != nil {
		t.Fatalf("read actorsrt_core.go: %v", err)
	}
	netPumpSource := functionSource(t, string(raw), "emitActorNetPump")
	if !strings.Contains(netPumpSource, "emitDeliverRemoteLinkedNodeDownFromRspFrame(e)") {
		t.Fatalf("__tetra_actor_net_pump does not route node_down frames through remote link propagation")
	}

	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	netPump, ok := symbolBody(obj, "__tetra_actor_net_pump")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_net_pump")
	}

	for label, seq := range map[string][]byte{
		"node_down frame branch":          cmpEaxImm32Encoding(actorWireFrameNodeDown),
		"remote link count scan":          movEaxFromRdiDispEncoding(actorLinkCountOff),
		"remote link low lane scan":       addRaxImm32Encoding(actorLinkRef0Off),
		"remote link high lane scan":      addRaxImm32Encoding(actorLinkHigh0Off),
		"remote link high mask":           andEaxImm32Encoding(actorRefRemoteHighMask),
		"remote link high base":           cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"remote linked trap-exit check":   movEaxFromRdiDispEncoding(actorTrapExitOff),
		"remote linked EXIT kind store":   movMem32RdiDispImm32Encoding(systemEventKindOff, actorSystemKindExit),
		"remote linked node_down reason":  movMem32RdiDispImm32Encoding(systemEventReasonKindOff, 5),
		"remote linked non-trap terminal": movMem32RdiDispImm32Encoding(actorStatusOff, statusDone),
		"remote link removal write-back":  movMem32RdiDispEaxEncoding(actorLinkCountOff),
	} {
		if !bytes.Contains(netPump, seq) {
			t.Fatalf("__tetra_actor_net_pump missing remote link node_down %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorMonitorUsesSchedulerMonitorTable(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	monitor, ok := symbolBody(obj, "__tetra_actor_monitor")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_monitor")
	}
	demonitor, ok := symbolBody(obj, "__tetra_actor_demonitor")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_demonitor")
	}

	for label, seq := range map[string][]byte{
		"monitor next id init": movMem32RdiDispImm32Encoding(schedMonitorNextIDOff, 1),
		"monitor count init":   movMem32RdiDispImm32Encoding(schedMonitorCountOff, 0),
	} {
		if !bytes.Contains(entry, seq) {
			t.Fatalf("__tetra_entry missing %s sequence % x", label, seq)
		}
	}

	for label, seq := range map[string][]byte{
		"monitor count load":       movEaxFromRdiDispEncoding(schedMonitorCountOff),
		"monitor next id load":     movEaxFromRdiDispEncoding(schedMonitorNextIDOff),
		"monitor next id write":    movMem32RdiDispEaxEncoding(schedMonitorNextIDOff),
		"monitor id table base":    addRaxImm32Encoding(schedMonitorID0Off),
		"monitor owner table base": addRaxImm32Encoding(schedMonitorOwnerRef0Off),
		"monitor target table base": addRaxImm32Encoding(
			schedMonitorTargetRef0Off,
		),
		"monitor target high table base": addRaxImm32Encoding(
			schedMonitorTargetHigh0Off,
		),
		"monitor table store":      movMem32RaxPtrEcxEncoding(),
		"monitor count write-back": movMem32RdiDispEaxEncoding(schedMonitorCountOff),
		"monitor local/remote split": append(
			movEaxEsiEncoding(),
			cmpEaxImm32Encoding(actorRefLocalHighSlot)...,
		),
		"remote monitor high mask":        andEaxImm32Encoding(actorRefRemoteHighMask),
		"remote monitor high base":        cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"remote monitor node epoch table": remoteNodeEpochPtrFromEaxToRdiEncoding(),
		"remote monitor generation table": remoteGenerationPtrFromEaxToRdiEncoding(),
	} {
		if !bytes.Contains(monitor, seq) {
			t.Fatalf("__tetra_actor_monitor missing %s sequence % x", label, seq)
		}
	}

	for label, seq := range map[string][]byte{
		"monitor count load":       movEaxFromRdiDispEncoding(schedMonitorCountOff),
		"monitor id table base":    addRaxImm32Encoding(schedMonitorID0Off),
		"monitor owner table base": addRaxImm32Encoding(schedMonitorOwnerRef0Off),
		"monitor target table base": addRaxImm32Encoding(
			schedMonitorTargetRef0Off,
		),
		"monitor target high table base": addRaxImm32Encoding(
			schedMonitorTargetHigh0Off,
		),
		"monitor table store":      movMem32RaxPtrEcxEncoding(),
		"monitor count decrement":  addEaxImm32Encoding(-1),
		"monitor count write-back": movMem32RdiDispEaxEncoding(schedMonitorCountOff),
	} {
		if !bytes.Contains(demonitor, seq) {
			t.Fatalf("__tetra_actor_demonitor missing %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorLinkUsesBidirectionalLinkTable(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}
	link, ok := symbolBody(obj, "__tetra_actor_link")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_link")
	}
	exit, ok := symbolBody(obj, "__tetra_actor_exit")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_exit")
	}

	linkCountInit := movMem32RdiDispImm32Encoding(actorLinkCountOff, 0)
	for name, body := range map[string][]byte{
		"__tetra_entry":       entry,
		"__tetra_actor_spawn": spawn,
	} {
		if !bytes.Contains(body, linkCountInit) {
			t.Fatalf("%s missing actor link count initialization at offset %d", name, actorLinkCountOff)
		}
	}

	for label, seq := range map[string][]byte{
		"link count load":              movEaxFromRdiDispEncoding(actorLinkCountOff),
		"link ref low table base":      addRaxImm32Encoding(actorLinkRef0Off),
		"link ref high table base":     addRaxImm32Encoding(actorLinkHigh0Off),
		"link ref table store":         movMem32RaxPtrEcxEncoding(),
		"link count increment":         addEaxImm32Encoding(1),
		"link count write-back":        movMem32RdiDispEaxEncoding(actorLinkCountOff),
		"local link high-slot compare": cmpEaxImm32Encoding(actorRefLocalHighSlot),
		"remote link high mask":        andEaxImm32Encoding(actorRefRemoteHighMask),
		"remote link high base":        cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"remote link node epoch table": remoteNodeEpochPtrFromEaxToRdiEncoding(),
		"remote link generation table": remoteGenerationPtrFromEaxToRdiEncoding(),
	} {
		if !bytes.Contains(link, seq) {
			t.Fatalf("__tetra_actor_link missing %s sequence % x", label, seq)
		}
	}

	for label, seq := range map[string][]byte{
		"abnormal exit link count scan": movEaxFromRdiDispEncoding(actorLinkCountOff),
		"linked ref high lane scan":     movEaxFromRdiDispEncoding(actorLinkHigh0Off),
		"linked ref local high guard":   cmpEaxImm32Encoding(actorRefLocalHighSlot),
		"linked ref generation load":    movEaxFromRdiDispEncoding(actorGenerationOff),
		"linked trap-exit check":        movEaxFromRdiDispEncoding(actorTrapExitOff),
		"linked exit code propagation":  movMem32RdiDispEaxEncoding(actorExitCodeOff),
		"linked status done":            movMem32RdiDispImm32Encoding(actorStatusOff, statusDone),
	} {
		if !bytes.Contains(exit, seq) {
			t.Fatalf("__tetra_actor_exit missing %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorUnlinkRemovesBidirectionalLinkTable(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	unlink, ok := symbolBody(obj, "__tetra_actor_unlink")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_unlink")
	}

	for label, seq := range map[string][]byte{
		"link count load":                movEaxFromRdiDispEncoding(actorLinkCountOff),
		"link ref low table base":        addRaxImm32Encoding(actorLinkRef0Off),
		"link ref high table base":       addRaxImm32Encoding(actorLinkHigh0Off),
		"link ref table store":           movMem32RaxPtrEcxEncoding(),
		"link count decrement":           addEaxImm32Encoding(-1),
		"link count write-back":          movMem32RdiDispEaxEncoding(actorLinkCountOff),
		"local unlink high-slot compare": cmpEaxImm32Encoding(actorRefLocalHighSlot),
		"remote unlink high mask":        andEaxImm32Encoding(actorRefRemoteHighMask),
		"remote unlink high base":        cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"remote unlink node epoch table": remoteNodeEpochPtrFromEaxToRdiEncoding(),
		"remote unlink generation table": remoteGenerationPtrFromEaxToRdiEncoding(),
	} {
		if !bytes.Contains(unlink, seq) {
			t.Fatalf("__tetra_actor_unlink missing %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeConstructsV2LocalActorRefs(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}
	self, ok := symbolBody(obj, "__tetra_actor_self")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_self")
	}
	sender, ok := symbolBody(obj, "__tetra_actor_sender")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_sender")
	}

	generationInit := movMem32RdiDispImm32Encoding(actorGenerationOff, actorGenerationInitial)
	for name, body := range map[string][]byte{
		"__tetra_entry":       entry,
		"__tetra_actor_spawn": spawn,
	} {
		if !bytes.Contains(body, generationInit) {
			t.Fatalf("%s missing actor generation initialization at offset %d", name, actorGenerationOff)
		}
	}

	localHigh := movEdxImm32Encoding(actorRefLocalHighSlot)
	generationLoad := movEcxFromRdiDispEncoding(actorGenerationOff)
	slotShift := []byte{0xC1, 0xE0, 0x10}
	slotGenerationJoin := []byte{0x09, 0xC8}
	for name, body := range map[string][]byte{
		"__tetra_actor_spawn":  spawn,
		"__tetra_actor_self":   self,
		"__tetra_actor_sender": sender,
	} {
		if !bytes.Contains(body, generationLoad) {
			t.Fatalf("%s does not load actor generation from offset %d", name, actorGenerationOff)
		}
		if !bytes.Contains(body, slotShift) {
			t.Fatalf("%s does not shift actor slot into the v2 ref low word", name)
		}
		if !bytes.Contains(body, slotGenerationJoin) {
			t.Fatalf("%s does not combine actor slot and generation in the v2 ref low word", name)
		}
		if !bytes.Contains(body, localHigh) {
			t.Fatalf("%s does not return the v2 local actor-ref high slot", name)
		}
	}
}

func TestLinuxRuntimeSpawnReusesReclaimableSlotsWithIncrementedGeneration(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}
	if !containsFragmentsInOrder(
		body,
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(statusReclaimable),
		movEaxFromRdiDispEncoding(actorGenerationOff),
		cmpEaxImm32Encoding(actorGenerationMax),
	) {
		t.Fatalf("__tetra_actor_spawn must gate slot reuse on reclaimable status before generation bump")
	}
	if !bytes.Contains(body, movEdxImm32Encoding(actorRefLocalHighSlot)) {
		t.Fatalf("__tetra_actor_spawn missing reused slot local high ref return")
	}
	generationBump := append([]byte{}, movEaxFromRdiDispEncoding(actorGenerationOff)...)
	generationBump = append(generationBump, addEaxImm32Encoding(1)...)
	generationBump = append(generationBump, movMem32RdiDispEaxEncoding(actorGenerationOff)...)
	if !bytes.Contains(body, generationBump) {
		t.Fatalf("__tetra_actor_spawn missing contiguous generation bump sequence % x", generationBump)
	}
}

func TestLinuxRuntimeSpawnReusedReclaimableSlotsReinitializeReleasedStacks(t *testing.T) {
	if actorStackInitRspOff%8 != 0 {
		t.Fatalf("actorStackInitRspOff=%d must be u64-aligned", actorStackInitRspOff)
	}
	if actorSize < actorStackInitRspOff+8 {
		t.Fatalf("actorSize=%d does not cover stack init rsp offset %d", actorSize, actorStackInitRspOff)
	}
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}
	storeInitRsp := movMem64RdiDispRaxEncoding(actorStackInitRspOff)
	if !bytes.Contains(entry, storeInitRsp) {
		t.Fatalf("__tetra_entry missing actor0 stored initial stack rsp at offset %d", actorStackInitRspOff)
	}
	if !bytes.Contains(spawn, storeInitRsp) {
		t.Fatalf("__tetra_actor_spawn missing new-slot stored initial stack rsp at offset %d", actorStackInitRspOff)
	}
	releasedStackCheck := append([]byte{}, movRaxFromRdiDispEncoding(actorStackInitRspOff)...)
	releasedStackCheck = append(releasedStackCheck, []byte{0x48, 0x85, 0xC0}...)
	if !bytes.Contains(spawn, releasedStackCheck) {
		t.Fatalf(
			"__tetra_actor_spawn missing reused-slot released stack check sequence % x",
			releasedStackCheck,
		)
	}
	if got := bytes.Count(spawn, movEaxImm32Encoding(9)); got < 2 {
		t.Fatalf("__tetra_actor_spawn mmap syscall count = %d, want new-slot and released-reuse stack allocation", got)
	}
}

func TestLinuxRuntimeSendEntrypointsDecodeV2LocalActorRefs(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	wantSequences := map[string][]byte{
		"local high-slot validation": movEaxEsiEncoding(),
		"local high-slot compare":    cmpEaxImm32Encoding(actorRefLocalHighSlot),
		"slot extraction":            []byte{0xC1, 0xE8, 0x10},
		"slot bounds check":          cmpEaxImm32Encoding(maxActors),
		"generation mask":            andEaxImm32Encoding(0xFFFF),
		"generation load":            movEaxFromRdiDispEncoding(actorGenerationOff),
		"generation compare":         cmpEaxEdxEncoding(),
	}
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for label, want := range wantSequences {
			if !bytes.Contains(body, want) {
				t.Fatalf("%s missing v2 actor-ref %s sequence % x", name, label, want)
			}
		}
	}
}

func TestLinuxRuntimeSendEntrypointsRejectStoppingActors(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !containsFragmentsInOrder(
			body,
			movEaxFromRdiDispEncoding(actorStatusOff),
			cmpEaxImm32Encoding(statusDone),
			cmpEaxImm32Encoding(statusReclaimable),
			cmpEaxImm32Encoding(statusStopping),
			movEaxImm32Encoding(0xFFFFFFFC),
		) {
			t.Fatalf("%s must reject reclaimable/stopping actors with the checked done-actor send failure", name)
		}
	}
}

func TestLinuxRuntimeSendEntrypointsRejectCanceledGroupActors(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !containsFragmentsInOrder(
			body,
			movEaxFromRdiDispEncoding(actorTaskGroupOff),
			cmpEaxImm32Encoding(taskGroupCanceled),
			movEaxImm32Encoding(0xFFFFFFFC),
		) {
			t.Fatalf("%s must reject canceled task-group actors with the checked done-actor send failure", name)
		}
	}
}

func TestLinuxRuntimeTaskGroupCloseTreatsReclaimableActorsAsDone(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_task_group_close")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_task_group_close")
	}
	if !containsFragmentsInOrder(
		body,
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(statusDone),
		cmpEaxImm32Encoding(statusReclaimable),
	) {
		t.Fatalf("__tetra_task_group_close must treat reclaimable group actors as terminal")
	}
}

func TestLinuxRuntimeSpawnRemoteConstructsV2RemoteActorRefs(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_spawn_remote")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn_remote")
	}

	want := map[string][]byte{
		"low-slot generation join": orEaxEcxEncoding(),
		"high-slot node shift":     []byte{0xC1, 0xE2, 0x10},
		"high-slot remote version": orEdxImm32Encoding(actorRefRemoteHighBase),
		"failure high-slot clear":  []byte{0x31, 0xD2},
	}
	for label, seq := range want {
		if !bytes.Contains(body, seq) {
			t.Fatalf("__tetra_actor_spawn_remote missing v2 remote ref %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeRemoteRefsUseSchedulerRemoteGenerationState(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	spawnRemote, ok := symbolBody(obj, "__tetra_actor_spawn_remote")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn_remote")
	}
	for label, seq := range map[string][]byte{
		"remote generation table pointer": remoteGenerationPtrFromEaxToRdiEncoding(),
		"remote generation ack load":      movEaxRspDispEncoding(actorWireOffsetValue + 8),
		"remote generation table store":   movMem32RdiDispEaxEncoding(0),
		"remote generation ref register":  movEcxEaxEncoding(),
		"low-slot generation join":        orEaxEcxEncoding(),
	} {
		if !bytes.Contains(spawnRemote, seq) {
			t.Fatalf("__tetra_actor_spawn_remote missing %s sequence % x", label, seq)
		}
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for label, seq := range map[string][]byte{
			"remote generation table pointer": remoteGenerationPtrFromEaxToRdiEncoding(),
			"remote generation table load":    movEaxFromRdiDispEncoding(0),
			"remote generation compare":       cmpEaxEdxEncoding(),
		} {
			if !bytes.Contains(body, seq) {
				t.Fatalf("%s missing %s sequence % x", name, label, seq)
			}
		}
	}
}

func TestLinuxRuntimeSpawnRemoteUsesSpawnAckSlotAndGeneration(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	body, ok := symbolBody(obj, "__tetra_actor_spawn_remote")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn_remote")
	}

	ackTypeCheck := append([]byte{}, movzxEaxWordRspDispEncoding(actorWireOffsetType)...)
	ackTypeCheck = append(ackTypeCheck, cmpEaxImm32Encoding(4)...)
	if !bytes.Contains(body, ackTypeCheck) {
		t.Fatalf("__tetra_actor_spawn_remote missing SpawnAck type check sequence % x", ackTypeCheck)
	}

	ackStatusCheck := append([]byte{}, movEaxRspDispEncoding(actorWireOffsetStatus)...)
	ackStatusCheck = append(ackStatusCheck, testEaxEaxEncoding()...)
	if !bytes.Contains(body, ackStatusCheck) {
		t.Fatalf("__tetra_actor_spawn_remote missing SpawnAck status check sequence % x", ackStatusCheck)
	}

	ackSlotLoad := movzxEaxWordRspDispEncoding(actorWireOffsetActor)
	if !bytes.Contains(body, ackSlotLoad) {
		t.Fatalf("__tetra_actor_spawn_remote missing SpawnAck actor slot load sequence % x", ackSlotLoad)
	}
	ackGenerationLoad := append([]byte{}, movEaxRspDispEncoding(actorWireOffsetValue+8)...)
	ackGenerationLoad = append(ackGenerationLoad, andEaxImm32Encoding(0xFFFF)...)
	ackGenerationLoad = append(ackGenerationLoad, testEaxEaxEncoding()...)
	if !bytes.Contains(body, ackGenerationLoad) {
		t.Fatalf(
			"__tetra_actor_spawn_remote missing SpawnAck generation validation sequence % x",
			ackGenerationLoad,
		)
	}
	ackNodeEpochLoad := append([]byte{}, movEaxRspDispEncoding(actorWireOffsetValue+4+8)...)
	ackNodeEpochLoad = append(ackNodeEpochLoad, andEaxImm32Encoding(0xFFFF)...)
	ackNodeEpochLoad = append(ackNodeEpochLoad, testEaxEaxEncoding()...)
	if !bytes.Contains(body, ackNodeEpochLoad) {
		t.Fatalf(
			"__tetra_actor_spawn_remote missing SpawnAck node epoch validation sequence % x",
			ackNodeEpochLoad,
		)
	}
	if bytes.Contains(body, movMem32RdiDispImm32Encoding(0, actorGenerationInitial)) {
		t.Fatalf("__tetra_actor_spawn_remote still stores synthetic generation %d", actorGenerationInitial)
	}
}

func TestLinuxRuntimeRemoteRefsUseSchedulerNodeEpochTable(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	nodeConnect, ok := symbolBody(obj, "__tetra_actor_node_connect")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_node_connect")
	}
	ackEpochValidation := append([]byte{}, movEaxRspDispEncoding(0x20+actorWireOffsetValue)...)
	ackEpochValidation = append(ackEpochValidation, andEaxImm32Encoding(0xFFFF)...)
	ackEpochValidation = append(ackEpochValidation, testEaxEaxEncoding()...)
	if !bytes.Contains(nodeConnect, ackEpochValidation) {
		t.Fatalf(
			"__tetra_actor_node_connect missing HelloAck node epoch validation sequence % x",
			ackEpochValidation,
		)
	}
	if !bytes.Contains(nodeConnect, remoteNodeEpochPtrFromEaxToRdiEncoding()) {
		t.Fatalf(
			"__tetra_actor_node_connect missing scheduler node epoch table pointer sequence % x",
			remoteNodeEpochPtrFromEaxToRdiEncoding(),
		)
	}
	if !bytes.Contains(nodeConnect, movMem32RdiDispEaxEncoding(0)) {
		t.Fatalf("__tetra_actor_node_connect missing acknowledged node epoch table store")
	}
	if bytes.Contains(nodeConnect, movMem32RdiDispImm32Encoding(0, actorNodeEpochInitial)) {
		t.Fatalf("__tetra_actor_node_connect still stores synthetic node epoch %d", actorNodeEpochInitial)
	}

	epochLoad := append([]byte{}, remoteNodeEpochPtrFromEaxToRdiEncoding()...)
	epochLoad = append(epochLoad, movEaxFromRdiDispEncoding(0)...)
	spawnRemote, ok := symbolBody(obj, "__tetra_actor_spawn_remote")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn_remote")
	}
	if !bytes.Contains(spawnRemote, epochLoad) {
		t.Fatalf("__tetra_actor_spawn_remote missing scheduler node epoch table load sequence % x", epochLoad)
	}
	if !bytes.Contains(spawnRemote, orEdxEaxEncoding()) {
		t.Fatalf("__tetra_actor_spawn_remote missing remote high-slot epoch join sequence % x", orEdxEaxEncoding())
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !bytes.Contains(body, epochLoad) {
			t.Fatalf("%s missing scheduler node epoch table load sequence % x", name, epochLoad)
		}
	}
}

func TestLinuxRuntimeSendEntrypointsDecodeV2RemoteActorRefs(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	want := map[string][]byte{
		"remote high-slot mask":    andEaxImm32Encoding(0xF8000000),
		"remote high-slot compare": cmpEaxImm32Encoding(actorRefRemoteHighBase),
		"remote epoch mask":        andEaxImm32Encoding(0xFFFF),
		"remote node mask":         andEaxImm32Encoding(0x07FF),
		"legacy wire node mask":    andEaxImm32Encoding(0x07FF0000),
		"legacy remote bit":        orEaxImm32Encoding(0x80000000),
		"legacy handle join":       []byte{0x09, 0xD0},
	}
	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for label, seq := range want {
			if !bytes.Contains(body, seq) {
				t.Fatalf("%s missing v2 remote actor-ref %s sequence % x", name, label, seq)
			}
		}
	}
}

func TestLinuxRuntimeNetPumpChecksActorWireVersion(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_net_pump")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_net_pump")
	}

	want := append([]byte{}, movzxEaxWordRspDispEncoding(actorWireOffsetVer)...)
	want = append(want, cmpEaxImm32Encoding(actorWireVersion)...)
	if !bytes.Contains(body, want) {
		t.Fatalf("__tetra_actor_net_pump missing wire version guard sequence % x", want)
	}
}

func TestLinuxRuntimeActorWaitParksUntilTargetDone(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_wait")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_wait")
	}

	want := map[string][]byte{
		"target done check":          cmpEaxImm32Encoding(statusDone),
		"wait table target storage":  addRaxImm32Encoding(schedActorWait0Off),
		"park target slot from r12":  movEaxR12dEncoding(),
		"current actor waiting park": movMem32RdiDispImm32Encoding(actorStatusOff, statusWaiting),
		"yield call":                 []byte{0xE8},
	}
	for label, seq := range want {
		if !bytes.Contains(body, seq) {
			t.Fatalf("__tetra_actor_wait missing blocking wait %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorWaitUntilUsesDeadlineTimeout(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_wait_until")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_wait_until")
	}

	want := map[string][]byte{
		"scheduler time load":      movEaxFromRdiDispEncoding(schedTimeMsOff),
		"deadline compare operand": movEcxR13dEncoding(),
		"deadline wake-at storage": movEaxR13dEncoding(),
		"current actor waiting park": movMem32RdiDispImm32Encoding(
			actorStatusOff,
			statusWaiting,
		),
		"yield call": []byte{0xE8},
	}
	for label, seq := range want {
		if !bytes.Contains(body, seq) {
			t.Fatalf("__tetra_actor_wait_until missing timeout %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorWaitResultMapsDoneToPublicLifecycleStatus(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{"__tetra_actor_wait", "__tetra_actor_wait_until"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !containsFragmentsInOrder(
			body,
			movEaxFromRdiDispEncoding(actorExitCodeOff),
			testEaxEaxEncoding(),
			movEdxImm32Encoding(actorLifecycleStatusExitedNormal),
			movEdxImm32Encoding(actorLifecycleStatusExitedError),
		) {
			t.Fatalf("%s must map done wait results to public exited lifecycle statuses", name)
		}
	}
}

func TestLinuxRuntimeActorWaitMarksDoneActorReclaimable(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{"__tetra_actor_wait", "__tetra_actor_wait_until"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !containsFragmentsInOrder(
			body,
			cmpEaxImm32Encoding(statusDone),
			movMem32RdiDispImm32Encoding(actorStatusOff, statusReclaimable),
			movEaxFromRdiDispEncoding(actorExitCodeOff),
			movEdxImm32Encoding(actorLifecycleStatusExitedNormal),
			movEdxImm32Encoding(actorLifecycleStatusExitedError),
		) {
			t.Fatalf("%s must mark done actors reclaimable before returning wait result", name)
		}
	}
}

func TestLinuxRuntimeActorWaitInvalidAndStaleResultsUsePublicDeadStatus(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{"__tetra_actor_wait", "__tetra_actor_wait_until"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for label, reason := range map[string]uint32{
			"invalid": 0xFFFFFFFD,
			"stale":   0xFFFFFFFB,
		} {
			if !containsFragmentsInOrder(
				body,
				movEaxImm32Encoding(reason),
				movEdxImm32Encoding(actorLifecycleStatusDead),
			) {
				t.Fatalf("%s must return public dead status for %s wait refs", name, label)
			}
		}
	}
}

func TestLinuxRuntimeInitializesActorByteCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}

	for _, body := range [][]byte{entry, spawn} {
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorMailboxPeakBytesOff,
			actorReclaimedBytesOff,
			actorBytesCopiedOff,
			actorCopyCountOff,
			actorOverBudgetCountOff,
			actorBackpressureEventsOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("runtime initializer missing zeroed actor counter offset %d", off)
			}
		}
		if !bytes.Contains(body, movMem64RdiDispRaxEncoding(actorByteBudgetOff)) {
			t.Fatalf("runtime initializer missing actor byte budget offset %d", actorByteBudgetOff)
		}
	}
	for _, off := range []int32{
		schedMsgPoolCapacityBytesOff,
		schedMsgPoolLiveBytesOff,
		schedMsgPoolReclaimedBytesOff,
		schedMsgPoolAllocFailuresOff,
	} {
		if !bytes.Contains(entry, movMem64RdiDispRaxEncoding(off)) {
			t.Fatalf("__tetra_entry missing scheduler message-pool counter offset %d", off)
		}
	}
}

func TestLinuxRuntimeObjectExportsActorMemoryTelemetrySnapshot(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	buildruntime.AnnotateRuntimeObjectSignatures(obj)
	if err := buildruntime.ValidateActorTelemetryRuntimeObject(obj); err != nil {
		t.Fatalf("ValidateActorTelemetryRuntimeObject: %v", err)
	}
	for _, sym := range obj.Symbols {
		if sym.Name != "__tetra_actor_memory_snapshot" {
			continue
		}
		if !sym.HasSignature || sym.ParamSlots != 1 || sym.ReturnSlots != 1 {
			t.Fatalf(
				"actor memory snapshot signature = has:%v params:%d returns:%d, want 1 -> 1",
				sym.HasSignature,
				sym.ParamSlots,
				sym.ReturnSlots,
			)
		}
		return
	}
	t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
}

func TestLinuxRuntimeObjectExportsActorLifecycleSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	buildruntime.AnnotateRuntimeObjectSignatures(obj)
	if err := buildruntime.ValidateActorLifecycleRuntimeObject(obj); err != nil {
		t.Fatalf("ValidateActorLifecycleRuntimeObject: %v", err)
	}
}

func TestLinuxRuntimeObjectExportsActorSystemReceiveSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_recv_system_begin",
		"__tetra_actor_recv_system_slot",
		"__tetra_actor_recv_system_count",
	} {
		if _, ok := symbolBody(obj, name); !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeActorMemorySnapshotExposesBudgetBackpressureFields(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_memory_snapshot")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
	}
	for _, off := range []int32{
		actorMailboxBytesOff,
		actorMailboxPeakBytesOff,
		actorBytesCopiedOff,
		actorByteBudgetOff,
		actorOverBudgetCountOff,
		actorBackpressureEventsOff,
	} {
		if !bytes.Contains(body, movRaxFromR8DispEncoding(off)) {
			t.Fatalf("actor memory snapshot missing load for actor offset %d", off)
		}
	}
}

func TestLinuxRuntimeActorMemorySnapshotRecordsStackDomainFields(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_memory_snapshot")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
	}
	if !bytes.Contains(body, []byte{0x48, 0x6B, 0xD1, 104}) {
		t.Fatalf("actor memory snapshot record stride missing stack-aware 104-byte record")
	}
	for label, seq := range map[string][]byte{
		"status load":       movEaxFromRdiDispEncoding(actorStatusOff),
		"done-status check": cmpEaxImm32Encoding(statusDone),
		"stack-size charge": addRaxImm32Encoding(stackSize),
	} {
		if !bytes.Contains(body, seq) {
			t.Fatalf("actor memory snapshot missing %s sequence % x", label, seq)
		}
	}
	for label, off := range map[string]int32{
		"mailbox_current_bytes": 56,
		"mailbox_peak_bytes":    64,
		"stack_live_bytes":      72,
		"stack_reserved_bytes":  80,
		"stack_retained_bytes":  88,
		"stack_released_bytes":  96,
	} {
		if !bytes.Contains(body, movMem64RdxDispRaxEncoding(off)) {
			t.Fatalf("actor memory snapshot missing store for %s at record offset %d", label, off)
		}
	}
}

func TestLinuxRuntimeActorMemorySnapshotSeparatesLiveCountFromRecordCount(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_memory_snapshot")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
	}
	for label, seq := range map[string][]byte{
		"live count initialization": {0x41, 0xB9, 0x00, 0x00, 0x00, 0x00}, // mov r9d, 0
		"done actors excluded from live count": append(
			cmpEaxImm32Encoding(statusDone),
			[]byte{0x0F, 0x84}...,
		),
		"reclaimable actors excluded from live count": append(
			cmpEaxImm32Encoding(statusReclaimable),
			[]byte{0x0F, 0x84}...,
		),
		"free actors excluded from live count": append(
			cmpEaxImm32Encoding(statusFree),
			[]byte{0x0F, 0x84}...,
		),
		"live count increment":       {0x49, 0x81, 0xC1, 0x01, 0x00, 0x00, 0x00}, // add r9, 1
		"live count returned in rdx": {0x4C, 0x89, 0xCA},                         // mov rdx, r9
	} {
		if !bytes.Contains(body, seq) {
			t.Fatalf("actor memory snapshot missing %s sequence % x", label, seq)
		}
	}
}

func TestLinuxRuntimeActorStatusMapsInternalStatesToV1Enum(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_status")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_status")
	}
	const (
		internalStatusStarting  = 7
		internalStatusStopping  = 8
		internalStatusRunning   = 6
		actorStatusStarting     = 0
		actorStatusReady        = 1
		actorStatusRunning      = 2
		actorStatusBlocked      = 3
		actorStatusSleeping     = 4
		actorStatusWaiting      = 5
		actorStatusStopping     = 6
		actorStatusExitedNormal = 7
		actorStatusExitedError  = 8
		actorStatusCanceled     = 9
		actorStatusDead         = 11
	)
	for label, fragments := range map[string][][]byte{
		"starting maps to starting": {
			cmpEaxImm32Encoding(internalStatusStarting),
			movEaxImm32Encoding(actorStatusStarting),
		},
		"free maps to dead": {
			cmpEaxImm32Encoding(statusFree),
			movEaxImm32Encoding(actorStatusDead),
		},
		"ready maps to ready": {
			cmpEaxImm32Encoding(statusReady),
			movEaxImm32Encoding(actorStatusReady),
		},
		"running maps to running": {
			cmpEaxImm32Encoding(internalStatusRunning),
			movEaxImm32Encoding(actorStatusRunning),
		},
		"blocked maps to blocked": {
			cmpEaxImm32Encoding(statusBlocked),
			movEaxImm32Encoding(actorStatusBlocked),
		},
		"done maps by exit code": {
			cmpEaxImm32Encoding(statusDone),
			movEaxFromRdiDispEncoding(actorExitCodeOff),
			testEaxEaxEncoding(),
			movEaxImm32Encoding(actorStatusExitedNormal),
			movEaxImm32Encoding(actorStatusExitedError),
		},
		"reclaimable maps by exit code": {
			cmpEaxImm32Encoding(statusReclaimable),
			movEaxFromRdiDispEncoding(actorExitCodeOff),
			testEaxEaxEncoding(),
			movEaxImm32Encoding(actorStatusExitedNormal),
			movEaxImm32Encoding(actorStatusExitedError),
		},
		"sleeping maps to sleeping": {
			cmpEaxImm32Encoding(statusSleeping),
			movEaxImm32Encoding(actorStatusSleeping),
		},
		"waiting maps to waiting": {
			cmpEaxImm32Encoding(statusWaiting),
			movEaxImm32Encoding(actorStatusWaiting),
		},
		"stopping maps to stopping": {
			cmpEaxImm32Encoding(internalStatusStopping),
			movEaxImm32Encoding(actorStatusStopping),
		},
		"task-group canceled maps to canceled": {
			movEaxFromRdiDispEncoding(actorTaskGroupOff),
			cmpEaxImm32Encoding(taskGroupCanceled),
			movEaxImm32Encoding(actorStatusCanceled),
		},
	} {
		if !containsFragmentsInOrder(body, fragments...) {
			t.Fatalf("__tetra_actor_status missing %s mapping", label)
		}
	}
}

func TestLinuxRuntimeActorStatusRawDistinguishesInvalidAndStale(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_status_raw")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_status_raw")
	}
	const actorStatusDead = 11
	for label, fragments := range map[string][][]byte{
		"ok result": {
			movEdxImm32Encoding(0),
		},
		"invalid result": {
			movEaxImm32Encoding(actorStatusDead),
			movEdxImm32Encoding(0xFFFFFFFD),
		},
		"stale result": {
			movEaxImm32Encoding(actorStatusDead),
			movEdxImm32Encoding(0xFFFFFFFB),
		},
	} {
		if !containsFragmentsInOrder(body, fragments...) {
			t.Fatalf("__tetra_actor_status_raw missing %s fragments", label)
		}
	}
}

func TestLinuxRuntimeActorStopMarksStoppingAndSchedulerFinalizes(t *testing.T) {
	const internalStatusStopping = 8
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	stop, ok := symbolBody(obj, "__tetra_actor_stop")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_stop")
	}
	if !bytes.Contains(stop, movMem32RdiDispImm32Encoding(actorStatusOff, internalStatusStopping)) {
		t.Fatalf("__tetra_actor_stop must mark target as stopping before scheduler finalization")
	}
	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	if !containsFragmentsInOrder(
		entry,
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(internalStatusStopping),
		movMem32RdiDispImm32Encoding(actorStatusOff, statusDone),
	) {
		t.Fatalf("__tetra_entry must finalize stopping actors to done during scheduler scan")
	}
}

func TestLinuxRuntimeSpawnedActorsStartInStartingState(t *testing.T) {
	const internalStatusStarting = 7
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}
	if !bytes.Contains(spawn, movMem32RdiDispImm32Encoding(actorStatusOff, internalStatusStarting)) {
		t.Fatalf("__tetra_actor_spawn missing starting status initialization")
	}
	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	if !containsFragmentsInOrder(
		entry,
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(statusReady),
		cmpEaxImm32Encoding(internalStatusStarting),
	) {
		t.Fatalf("__tetra_entry must treat starting actors as runnable before dispatch")
	}
}

func TestLinuxRuntimeMarksRunningActorsAndRestoresReadyOnYield(t *testing.T) {
	const internalStatusRunning = 6
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	if !bytes.Contains(entry, movMem32RdiDispImm32Encoding(actorStatusOff, internalStatusRunning)) {
		t.Fatalf("__tetra_entry missing running status store before actor dispatch")
	}
	yield, ok := symbolBody(obj, "__tetra_actor_yield")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_yield")
	}
	if !containsFragmentsInOrder(
		yield,
		movEaxFromRdiDispEncoding(actorStatusOff),
		cmpEaxImm32Encoding(internalStatusRunning),
		movMem32RdiDispImm32Encoding(actorStatusOff, statusReady),
	) {
		t.Fatalf("__tetra_actor_yield must restore only running actors to ready")
	}
}

func TestLinuxSurfaceHostIPCRuntimeDoesNotUseMemfdSurfaceOpen(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_open")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_open")
	}
	if bytes.Contains(body, movEaxImm32Encoding(linuxSysMemfdCreate)) {
		t.Fatalf("__tetra_surface_open still uses memfd_create in host-required runtime")
	}
	if !bytes.Contains(body, movEaxImm32Encoding(linuxSysSocket)) {
		t.Fatalf("__tetra_surface_open does not attempt AF_UNIX socket creation")
	}
	if !bytes.Contains(body, movEaxImm32Encoding(linuxSysConnect)) {
		t.Fatalf("__tetra_surface_open does not connect to the Surface host socket")
	}
	if !bytes.Contains(body, []byte("/tmp/tetra-surface-host.sock")) {
		t.Fatalf("__tetra_surface_open does not embed the Surface host socket path")
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForPresentAndPoll(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{"__tetra_surface_present_rgba", "__tetra_surface_poll_event_into"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if bytes.Contains(body, movEaxImm32Encoding(linuxSysLseek)) {
			t.Fatalf("%s still uses lseek/memfd cursor behavior in host-required runtime", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForScalarEventAccessors(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForTextClipboardAndComposition(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCBeginFrameStaysLocal(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_begin_frame")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_begin_frame")
	}
	if bytes.Contains(body, uint32LE(surfaceHostMagic)) {
		t.Fatalf(
			"__tetra_surface_begin_frame should stay local; " +
				"host evidence is produced by open/poll/present/close",
		)
	}
	if !bytes.Contains(body, []byte{0x31, 0xC0, 0xC3}) {
		t.Fatalf("__tetra_surface_begin_frame should return local no-op success")
	}
}

func TestLinuxSurfaceHostIPCPresentReturnsOneOnSuccess(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_present_rgba")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_present_rgba")
	}
	successEpilogue := append(movEaxImm32Encoding(1), 0xC9, 0xC3)
	failureEpilogue := []byte{0x31, 0xC0, 0xC9, 0xC3}
	successAt := bytes.Index(body, successEpilogue)
	failureAt := bytes.LastIndex(body, failureEpilogue)
	if successAt < 0 {
		t.Fatalf("__tetra_surface_present_rgba missing success return 1 epilogue")
	}
	if failureAt < 0 {
		t.Fatalf("__tetra_surface_present_rgba missing failure return 0 epilogue")
	}
	if successAt > failureAt {
		t.Fatalf(
			"__tetra_surface_present_rgba returns 0 on success and 1 on failure; "+
				"success epilogue at %d, failure epilogue at %d",
			successAt,
			failureAt,
		)
	}
}

func TestLinuxSurfaceHostIPCPresentUsesStreamWriteLoop(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_present_rgba")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_present_rgba")
	}
	if bytes.Contains(body, []byte{0x48, 0x39, 0xC2}) {
		t.Fatalf(
			"__tetra_surface_present_rgba still uses single-write cmp rdx,rax instead of stream write loop",
		)
	}
	if !bytes.Contains(body, []byte{0x48, 0x01, 0xC6}) ||
		!bytes.Contains(body, []byte{0x48, 0x29, 0xC2}) {
		t.Fatalf("__tetra_surface_present_rgba missing stream pointer/remaining update loop")
	}
}

func TestLinuxRuntimeAccountsMailboxBytesOnSendAndReceive(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorMailboxPeakBytesOff,
			actorBytesCopiedOff,
			actorCopyCountOff,
			schedMsgPoolLiveBytesOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing enqueue accounting for offset %d", name, off)
			}
		}
	}

	for _, name := range []string{
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_begin",
		"__tetra_actor_exit",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorReclaimedBytesOff,
			schedMsgPoolLiveBytesOff,
			schedMsgPoolReclaimedBytesOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing receive/reclaim accounting for offset %d", name, off)
			}
		}
	}
}

func TestLinuxRuntimeAccountsMessagePoolFailuresWithoutMailboxCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !bytes.Contains(body, movMem64RdiDispRaxEncoding(schedMsgPoolAllocFailuresOff)) {
			t.Fatalf("%s missing message-pool allocation failure counter", name)
		}
	}
}

func TestLinuxRuntimeBackpressurePathsAccountBudgetCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{actorOverBudgetCountOff, actorBackpressureEventsOff} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing backpressure budget counter write for offset %d", name, off)
			}
		}
	}
}

func TestLinuxRuntimeSendFailurePathsDoNotEmitMailboxAccounting(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(
			repoRootFromActorsRTTest(t),
			"compiler",
			"internal",
			"actorsrt",
			"actorsrt_core.go",
		),
	)
	if err != nil {
		t.Fatalf("read actorsrt_core.go: %v", err)
	}
	source := string(raw)
	for _, name := range []string{"emitSend", "emitSendMsg", "emitSendBegin"} {
		body := functionSource(t, source, name)
		accountAt := strings.Index(body, "emitAccountMailboxEnqueueInRdi(e)")
		if accountAt < 0 {
			t.Fatalf("%s missing successful enqueue accounting call", name)
		}
		for _, marker := range []string{
			"overflowTo := len(e.Buf)",
			"fullTo := len(e.Buf)",
			"invalidTo := len(e.Buf)",
			"doneTo := len(e.Buf)",
		} {
			markerAt := strings.Index(body, marker)
			if markerAt < 0 {
				t.Fatalf("%s missing failure marker %q", name, marker)
			}
			if markerAt < accountAt {
				t.Fatalf(
					"%s failure marker %q appears before successful enqueue accounting",
					name,
					marker,
				)
			}
			if strings.Contains(body[markerAt:], "emitAccountMailboxEnqueueInRdi(e)") {
				t.Fatalf(
					"%s failure marker %q must not flow through mailbox accounting",
					name,
					marker,
				)
			}
		}
	}
}

func functionSource(t *testing.T, source string, name string) string {
	t.Helper()
	start := strings.Index(source, "func "+name+"(")
	if start < 0 {
		t.Fatalf("missing function %s", name)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("missing function body for %s", name)
	}
	open += start
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open+1 : i]
			}
		}
	}
	t.Fatalf("unterminated function body for %s", name)
	return ""
}

func readActorsRTCoreSource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(
		filepath.Join(
			repoRootFromActorsRTTest(t),
			"compiler",
			"internal",
			"actorsrt",
			"actorsrt_core.go",
		),
	)
	if err != nil {
		t.Fatalf("read actorsrt_core.go: %v", err)
	}
	return string(raw)
}

func movMem64RdiDispRaxEncoding(off int32) []byte {
	var out [7]byte
	out[0] = 0x48
	out[1] = 0x89
	out[2] = 0x87
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movMem32RdiDispImm32Encoding(off int32, imm int32) []byte {
	e := &x64.Emitter{}
	e.MovMem32RdiDispImm32(off, imm)
	return append([]byte(nil), e.Buf...)
}

func containsFragmentsInOrder(body []byte, fragments ...[]byte) bool {
	cursor := 0
	for _, fragment := range fragments {
		idx := bytes.Index(body[cursor:], fragment)
		if idx < 0 {
			return false
		}
		cursor += idx + len(fragment)
	}
	return true
}

func movMem32RdiDispEaxEncoding(off int32) []byte {
	e := &x64.Emitter{}
	e.MovMem32RdiDispEax(off)
	return append([]byte(nil), e.Buf...)
}

func movMem32RaxPtrEcxEncoding() []byte {
	e := &x64.Emitter{}
	e.MovMem32RaxPtrEcx()
	return append([]byte(nil), e.Buf...)
}

func movRaxFromR8DispEncoding(off int32) []byte {
	var out [7]byte
	out[0] = 0x49
	out[1] = 0x8B
	out[2] = 0x80
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movMem64RdxDispRaxEncoding(off int32) []byte {
	if off == 0 {
		return []byte{0x48, 0x89, 0x02}
	}
	var out [7]byte
	out[0] = 0x48
	out[1] = 0x89
	out[2] = 0x82
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movRaxFromRdiDispEncoding(off int32) []byte {
	var out [7]byte
	out[0] = 0x48
	out[1] = 0x8B
	out[2] = 0x87
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movEcxFromRdiDispEncoding(off int32) []byte {
	e := &x64.Emitter{}
	e.MovEcxFromRdiDisp(off)
	return append([]byte(nil), e.Buf...)
}

func movEaxFromRdiDispEncoding(off int32) []byte {
	e := &x64.Emitter{}
	e.MovEaxFromRdiDisp(off)
	return append([]byte(nil), e.Buf...)
}

func movEaxFromRspDispEncoding(disp byte) []byte {
	e := &x64.Emitter{}
	e.MovEaxFromRspDisp(int32(disp))
	return append([]byte(nil), e.Buf...)
}

func movEaxRspDispEncoding(disp byte) []byte {
	return []byte{0x8B, 0x44, 0x24, disp}
}

func movEaxEsiEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEaxEsi()
	return append([]byte(nil), e.Buf...)
}

func movEcxEaxEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEcxEax()
	return append([]byte(nil), e.Buf...)
}

func movzxEaxWordRspDispEncoding(disp byte) []byte {
	return []byte{0x0F, 0xB7, 0x44, 0x24, disp}
}

func movEaxImm32Encoding(value uint32) []byte {
	var out [5]byte
	out[0] = 0xB8
	binary.LittleEndian.PutUint32(out[1:], value)
	return out[:]
}

func addEaxImm32Encoding(value int32) []byte {
	e := &x64.Emitter{}
	e.AddEaxImm32(value)
	return append([]byte(nil), e.Buf...)
}

func addRaxImm32Encoding(value int32) []byte {
	e := &x64.Emitter{}
	e.AddRaxImm32(value)
	return append([]byte(nil), e.Buf...)
}

func movEaxR12dEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEaxR12d()
	return append([]byte(nil), e.Buf...)
}

func movEaxR13dEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEaxR13d()
	return append([]byte(nil), e.Buf...)
}

func movEcxR13dEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEcxR13d()
	return append([]byte(nil), e.Buf...)
}

func cmpEaxImm32Encoding(value uint32) []byte {
	e := &x64.Emitter{}
	e.CmpEaxImm32(int32(value))
	return append([]byte(nil), e.Buf...)
}

func cmpRaxImm32Encoding(value int32) []byte {
	e := &x64.Emitter{}
	e.CmpRaxImm32(value)
	return append([]byte(nil), e.Buf...)
}

func andEaxImm32Encoding(value uint32) []byte {
	var out [5]byte
	out[0] = 0x25
	binary.LittleEndian.PutUint32(out[1:], value)
	return out[:]
}

func orEaxImm32Encoding(value uint32) []byte {
	var out [5]byte
	out[0] = 0x0D
	binary.LittleEndian.PutUint32(out[1:], value)
	return out[:]
}

func orEdxImm32Encoding(value uint32) []byte {
	var out [6]byte
	out[0] = 0x81
	out[1] = 0xCA
	binary.LittleEndian.PutUint32(out[2:], value)
	return out[:]
}

func orEdxEaxEncoding() []byte {
	return []byte{0x09, 0xC2}
}

func orEaxEcxEncoding() []byte {
	return []byte{0x09, 0xC8}
}

func cmpEaxEdxEncoding() []byte {
	e := &x64.Emitter{}
	e.CmpEaxEdx()
	return append([]byte(nil), e.Buf...)
}

func testEaxEaxEncoding() []byte {
	e := &x64.Emitter{}
	e.TestEaxEax()
	return append([]byte(nil), e.Buf...)
}

func movEdxImm32Encoding(value uint32) []byte {
	e := &x64.Emitter{}
	e.MovEdxImm32(value)
	return append([]byte(nil), e.Buf...)
}

func remoteGenerationPtrFromEaxToRdiEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedRemoteGeneration0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
	return append([]byte(nil), e.Buf...)
}

func remoteNodeEpochPtrFromEaxToRdiEncoding() []byte {
	e := &x64.Emitter{}
	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(2)
	e.MovRaxR15()
	e.AddRaxImm32(schedNodeEpoch0Off)
	e.AddRaxRbx()
	e.MovRdiRax()
	return append([]byte(nil), e.Buf...)
}

func uint32LE(value uint32) []byte {
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], value)
	return out[:]
}

// ---- actor_state_symbols_test.go ----

func TestBuiltinRuntimeExportsActorStateSymbols(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_load") {
				t.Fatalf("runtime missing __tetra_actor_state_load")
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_store") {
				t.Fatalf("runtime missing __tetra_actor_state_store")
			}
		})
	}
}

func TestLinuxRuntimeExportsFilesystemSymbol(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if !hasSymbol(obj.Symbols, "__tetra_fs_exists") {
		t.Fatalf("linux runtime missing __tetra_fs_exists")
	}
}

func TestLinuxRuntimeExportsNetSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeExportsSurfaceSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeExportsDistributedActorSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name       string
		build      func([]string) (*tobj.Object, error)
		wantNoop   bool
		wantActive bool
	}{
		{name: "linux-x64", build: BuildLinuxX64, wantActive: true},
		{name: "macos-x64", build: BuildMacOSX64, wantNoop: true},
		{name: "windows-x64", build: BuildWindowsX64, wantNoop: true},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			body, ok := symbolBody(obj, "__tetra_actor_net_pump")
			if !ok {
				t.Fatalf("runtime missing __tetra_actor_net_pump")
			}
			isNoop := len(body) >= 3 && body[0] == 0x31 && body[1] == 0xC0 && body[2] == 0xC3
			if tt.wantNoop && !isNoop {
				t.Fatalf(
					"%s __tetra_actor_net_pump must be a no-op on non-Linux targets, body prefix=% x",
					tt.name,
					bodyPrefix(body, 8),
				)
			}
			if tt.wantActive && isNoop {
				t.Fatalf("%s __tetra_actor_net_pump must be active, got no-op body", tt.name)
			}
		})
	}
}

func TestLinuxDistributedRuntimeUsesWideStackSubFor128ByteFrames(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	badSignedImm8Sub := []byte{0x48, 0x83, 0xEC, 0x80}
	goodImm32Sub := []byte{0x48, 0x81, 0xEC, 0x80, 0x00, 0x00, 0x00}
	for _, name := range []string{"__tetra_actor_node_connect", "__tetra_actor_net_pump"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("runtime missing %s", name)
		}
		if bytes.Contains(body, badSignedImm8Sub) {
			t.Fatalf("%s uses signed imm8 stack subtraction for 128-byte frame", name)
		}
		if !bytes.Contains(body, goodImm32Sub) {
			t.Fatalf(
				"%s missing imm32 stack subtraction for 128-byte frame, prefix=% x",
				name,
				bodyPrefix(body, 16),
			)
		}
	}
}

func TestNonLinuxRuntimesDoNotExportDistributedActorSymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build([]string{"main", "worker"})
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			for _, name := range []string{
				"__tetra_actor_node_connect",
				"__tetra_actor_spawn_remote",
				"__tetra_actor_node_status",
			} {
				if hasSymbol(obj.Symbols, name) {
					t.Fatalf(
						"%s runtime must not export Linux distributed actor symbol %s",
						tt.name,
						name,
					)
				}
			}
		})
	}
}

func TestRuntimeBuildersRejectInvalidEntrySymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	cases := []struct {
		name    string
		entries []string
		want    string
	}{
		{
			name:    "missing_main",
			entries: nil,
			want:    "missing entry symbols",
		},
		{
			name:    "empty_main",
			entries: []string{""},
			want:    "missing entry symbols",
		},
		{
			name:    "empty_spawn_entry",
			entries: []string{"main", ""},
			want:    "empty runtime entry symbol at index 1",
		},
		{
			name:    "duplicate_entry",
			entries: []string{"main", "worker", "worker"},
			want:    "duplicate runtime entry symbol 'worker'",
		},
	}

	for _, builder := range builders {
		for _, tc := range cases {
			t.Run(builder.name+"/"+tc.name, func(t *testing.T) {
				_, err := builder.build(tc.entries)
				if err == nil {
					t.Fatalf("expected invalid entry symbol error")
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want substring %q", err, tc.want)
				}
			})
		}
	}
}

func hasSymbol(symbols []tobj.Symbol, want string) bool {
	for _, sym := range symbols {
		if sym.Name == want {
			return true
		}
	}
	return false
}

func symbolBody(obj *tobj.Object, want string) ([]byte, bool) {
	start := -1
	end := len(obj.Code)
	for _, sym := range obj.Symbols {
		offset := int(sym.Offset)
		if sym.Name == want {
			start = offset
			continue
		}
		if start >= 0 && offset > start && offset < end {
			end = offset
		}
	}
	if start < 0 || start > len(obj.Code) || end < start {
		return nil, false
	}
	return obj.Code[start:end], true
}

func bodyPrefix(body []byte, n int) []byte {
	if len(body) < n {
		return body
	}
	return body[:n]
}

// ---- production_boundary_test.go ----

func TestActorRuntimeProductionBoundaryAuditCoversP18PlanList(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(report); err != nil {
		t.Fatalf("ValidateActorRuntimeProductionBoundaryAudit failed: %v", err)
	}
	if report.SchemaVersion != "tetra.runtime.actor.production_boundary.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionClaimed {
		t.Fatalf("P18.0 audit must not claim a full production actor runtime")
	}
	if !hasActorBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		t.Fatalf("non-claims = %#v, want full production actor runtime non-claim", report.NonClaims)
	}

	byID := map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []ActorRuntimeBoundaryID{
		ActorRuntimeBoundaryCurrentLimits,
		ActorRuntimeBoundarySchedulerPrototype,
		ActorRuntimeBoundaryProductionAcceptance,
		ActorRuntimeBoundaryFullClaimBlockers,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P18.0 audit row %q", id)
		}
	}

	limits := byID[ActorRuntimeBoundaryCurrentLimits]
	if limits.Status != ActorRuntimeBoundaryDocumentedLimit {
		t.Fatalf(
			"current limits status = %q, want %q",
			limits.Status,
			ActorRuntimeBoundaryDocumentedLimit,
		)
	}
	for _, want := range []string{
		"maxActors=128",
		"msgPoolSize=65536",
		"maxActorMailboxMsgs=256",
		"actor_state_slots=8",
		"single-thread cooperative scheduler",
		"round-robin runnable actor fairness has bounded yield-progress evidence",
		"timed sleeping actors wake in deterministic deadline order",
		"linux-x64 distributed runtime only",
		"non-linux actor net pump is no-op",
		"mailbox full returns checked -2",
		"mailbox backpressure recovers after drain",
		"typed mailbox backpressure does not enqueue a partial payload",
		"message pool exhaustion returns checked -1",
		"drained message pool entries are reclaimed",
		"recycled message nodes scrub system kind and payload slots",
		"invalid actor handle sends return checked -3",
		"done actor sends return checked -4",
		"nonzero actor entry return is exposed only as done-state send failure",
		"local linked abnormal exits propagate",
		"local actor_unlink removes bounded local abnormal-exit propagation",
		"no actor status, actor join, or actor exit-code API",
		"waited reclaimable actor slots reuse retained stack frames and reinitialize released stacks",
		"more than 10000 lifetime spawns work under the concurrent actor cap",
		"messages already queued in another actor mailbox remain receivable",
		"done actors are not restarted",
		"blocked actors continue to depend on normal message",
		"missing-node node_down remains checked distributed status evidence",
		"no automatic retry, restart, reconnect, or supervision",
		"task-group cancellation wakes recv_until",
		"task-group cancellation wakes actors already waiting on task_join_result_i32",
		"task_join_i32 wakes on task-group cancellation with raw zero value",
		"task join, timed join, poll, and typed task join observers treat reclaimable target slots",
		"successful consuming task joins mark completed task actor slots reclaimable after result read",
		"more than 10000 joined task lifetime spawns work under the concurrent actor cap",
		"task_group_close treats joined reclaimable task actors as terminal",
		"typed task joins and result getters clear target/current result slots",
		"non-timed actor receives do not expose a cancellation result",
	} {
		if !hasActorBoundaryText(limits.RequiredFacts, want) {
			t.Fatalf("current limits row missing fact %q: %#v", want, limits.RequiredFacts)
		}
	}
	for _, want := range []string{
		"compiler/internal/actorsrt/actorsrt_core.go",
		"emitMailboxFullCheckForReceiverInEcx",
		"emitCheckedMessagePoolAlloc",
		"emitRecycleMessageNodeInRax",
		"emitInvalidActorHandleReturn",
		"emitActorDoneReturn",
		"emitBlockedDeadlineWakeCheck",
		"emitWaitingTaskWakeCheck",
		"emitCurrentTaskGroupCanceledCheck",
		"TestActorMailboxFullReturnsCheckedBackpressure",
		"TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRun",
		"TestActorTaggedMailboxBackpressureRecoversAfterSelfDrainBuildAndRun",
		"TestActorTypedMailboxBackpressureRecoversWithoutPartialPayloadBuildAndRun",
		"TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun",
		"TestActorMessagePoolExhaustionReturnsCheckedFailure",
		"TestRecycleMessageNodeScrubsPayloadSlotsBeforeFreeList",
		"TestActorInvalidHandleSendReturnsCheckedFailure",
		"TestActorSendToDoneActorReturnsCheckedFailure",
		"TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAndRun",
		"TestActorLinkPropagatesAbnormalExitBuildAndRun",
		"TestActorUnlinkStopsAbnormalExitPropagationBuildAndRun",
		"TestLinuxRuntimeActorLinkUsesBidirectionalLinkTable",
		"TestLinuxRuntimeActorUnlinkRemovesBidirectionalLinkTable",
		"TestActorLifecycleReceivesPendingMessageFromDoneSenderBuildAndRun",
		"TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlockedActorsBuildAndRun",
		"TestActorLifecycleDoneActorDrainsPendingMailboxIntoMessagePoolBuildAndRun",
		"TestActorLifetimeSpawnsExceedTenThousandUnderConcurrentCapBuildAndRun",
		"TestLinuxRuntimeSpawnReusedReclaimableSlotsReinitializeReleasedStacks",
		"TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuildAndRun",
		"TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndRun",
		"TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect",
		"TestLinuxRuntimePumpsNodeDownIntoNodeStatus",
		"TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun",
		"TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun",
		"TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun",
		"TestEmitTaskJoinEntrypointsTreatReclaimableTargetsAsDone",
		"TestEmitTaskJoinEntrypointsMarkDoneTargetsReclaimableAfterResultRead",
		"TestTaskLifetimeSpawnsExceedTenThousandUnderConcurrentCapBuildAndRun",
		"TestLinuxRuntimeTaskGroupCloseTreatsReclaimableActorsAsDone",
		"TestActorSendToCanceledActorReturnsCheckedFailure",
		"TestEmitTaskJoinTypedClearsTargetResultSlotsAfterJoin",
		"TestEmitTaskResultGetClearsCurrentStagedSlotAfterRead",
		"docs/spec/runtime/actors.md",
		"TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump",
	} {
		if !strings.Contains(limits.Evidence, want) {
			t.Fatalf("current limits evidence missing %q: %s", want, limits.Evidence)
		}
	}

	prototype := byID[ActorRuntimeBoundarySchedulerPrototype]
	if prototype.Status != ActorRuntimeBoundaryPrototypeEvidence {
		t.Fatalf(
			"scheduler prototype status = %q, want %q",
			prototype.Status,
			ActorRuntimeBoundaryPrototypeEvidence,
		)
	}
	for _, want := range []string{
		"single-core FIFO compatibility",
		"two-core work stealing",
		"bounded typed mailbox",
		"zero_copy_move",
		"bytes_copied=0",
	} {
		if !hasActorBoundaryText(prototype.RequiredFacts, want) {
			t.Fatalf("scheduler prototype row missing fact %q: %#v", want, prototype.RequiredFacts)
		}
	}
	if !strings.Contains(prototype.Boundary, "not a production multi-threaded actor scheduler") {
		t.Fatalf("scheduler prototype boundary = %q", prototype.Boundary)
	}

	acceptance := byID[ActorRuntimeBoundaryProductionAcceptance]
	if acceptance.Status != ActorRuntimeBoundaryAcceptanceRequired {
		t.Fatalf(
			"production acceptance status = %q, want %q",
			acceptance.Status,
			ActorRuntimeBoundaryAcceptanceRequired,
		)
	}
	for _, want := range []string{
		"production task scheduler",
		"actor scheduler starvation/progress bound",
		"bounded mailbox backpressure",
		"message reclamation",
		"race-safety model",
		"cross-target distributed runtime gates",
		"blocking primitive by cancellation-source matrix",
		"structured concurrency",
	} {
		if !hasActorBoundaryText(acceptance.RequiredFacts, want) {
			t.Fatalf(
				"production acceptance row missing fact %q: %#v",
				want,
				acceptance.RequiredFacts,
			)
		}
	}

	blockers := byID[ActorRuntimeBoundaryFullClaimBlockers]
	if blockers.Status != ActorRuntimeBoundaryBlocked {
		t.Fatalf("blockers status = %q, want %q", blockers.Status, ActorRuntimeBoundaryBlocked)
	}
	for _, want := range []string{
		"production multi-threaded actor scheduler",
		"non-Linux-x64 distributed actor runtime",
		"full cancellation and structured concurrency",
		"full race-safety proof",
	} {
		if !hasActorBoundaryText(blockers.MissingFacts, want) {
			t.Fatalf("blockers row missing fact %q: %#v", want, blockers.MissingFacts)
		}
	}
}

func TestActorRuntimeProductionBoundaryAuditRejectsFakeFullProductionClaim(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}

	fakeClaim := report
	fakeClaim.FullProductionClaimed = true
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakeClaim); err == nil ||
		!strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake full-production claim error = %v", err)
	}

	missingBlockers := cloneActorRuntimeBoundaryReport(report)
	for i := range missingBlockers.Rows {
		if missingBlockers.Rows[i].ID == ActorRuntimeBoundaryFullClaimBlockers {
			missingBlockers.Rows[i].MissingFacts = nil
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(missingBlockers); err == nil ||
		!strings.Contains(err.Error(), "blockers") {
		t.Fatalf("missing blocker facts error = %v", err)
	}

	fakePromotion := cloneActorRuntimeBoundaryReport(report)
	for i := range fakePromotion.Rows {
		if fakePromotion.Rows[i].ID == ActorRuntimeBoundarySchedulerPrototype {
			fakePromotion.Rows[i].Status = ActorRuntimeBoundaryStatus("production_ready")
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakePromotion); err == nil ||
		!strings.Contains(err.Error(), "scheduler prototype") {
		t.Fatalf("fake scheduler promotion error = %v", err)
	}

	noNonClaim := cloneActorRuntimeBoundaryReport(report)
	noNonClaim.NonClaims = nil
	if err := ValidateActorRuntimeProductionBoundaryAudit(noNonClaim); err == nil ||
		!strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func hasActorBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneActorRuntimeBoundaryReport(report ActorRuntimeBoundaryReport) ActorRuntimeBoundaryReport {
	clone := report
	clone.Rows = append([]ActorRuntimeBoundaryRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}

// ---- runtime_source_parity_test.go ----

func TestSelfhostActorRuntimeSourcesMatchCanonicalRT(t *testing.T) {
	root := repoRootFromActorsRTTest(t)
	canonicalDir := filepath.Join(root, "__rt")
	selfhostDir := filepath.Join(root, "compiler", "selfhostrt")

	canonical, err := filepath.Glob(filepath.Join(canonicalDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob canonical actor runtime files: %v", err)
	}
	if len(canonical) == 0 {
		t.Fatalf("no canonical actor runtime files found under %s", canonicalDir)
	}
	sort.Strings(canonical)

	for _, canonicalPath := range canonical {
		name := filepath.Base(canonicalPath)
		selfhostPath := filepath.Join(selfhostDir, name)
		t.Run(name, func(t *testing.T) {
			canonicalRaw, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatalf("read canonical runtime source: %v", err)
			}
			selfhostRaw, err := os.ReadFile(selfhostPath)
			if err != nil {
				t.Fatalf("read selfhost runtime source: %v", err)
			}
			if !bytes.Equal(canonicalRaw, selfhostRaw) {
				canonicalSum := sha256.Sum256(canonicalRaw)
				selfhostSum := sha256.Sum256(selfhostRaw)
				t.Fatalf(
					"selfhost actor runtime source drift for %s: __rt sha256=%x selfhostrt sha256=%x",
					name,
					canonicalSum,
					selfhostSum,
				)
			}
		})
	}

	selfhost, err := filepath.Glob(filepath.Join(selfhostDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob selfhost actor runtime files: %v", err)
	}
	canonicalNames := map[string]bool{}
	for _, path := range canonical {
		canonicalNames[filepath.Base(path)] = true
	}
	for _, path := range selfhost {
		name := filepath.Base(path)
		if !canonicalNames[name] {
			t.Fatalf("selfhost actor runtime source %s has no canonical __rt peer", name)
		}
	}
}

func TestActorRuntimePOCSourcesRemainHistoricalReferences(t *testing.T) {
	root := repoRootFromActorsRTTest(t)
	historical := []string{
		filepath.Join("__rt", "actors_poc_sysv.tetra"),
		filepath.Join("__rt", "actors_poc_win64.tetra"),
		filepath.Join("compiler", "selfhostrt", "actors_poc_sysv.tetra"),
		filepath.Join("compiler", "selfhostrt", "actors_poc_win64.tetra"),
	}
	for _, rel := range historical {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("read historical PoC runtime source: %v", err)
			}
			if !bytes.Contains(raw, []byte("actors_poc")) {
				t.Fatalf("%s does not look like a historical actors_poc module", rel)
			}
		})
	}

	productionSelectionFiles := []string{
		filepath.Join("compiler", "compiler_build_runtime.go"),
		filepath.Join("compiler", "internal", "actorsrt", "actorsrt_core.go"),
	}
	for _, rel := range productionSelectionFiles {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("read production runtime selection file: %v", err)
			}
			if bytes.Contains(raw, []byte("actors_poc")) {
				t.Fatalf("%s promotes historical actors_poc runtime into production selection", rel)
			}
		})
	}
}

func repoRootFromActorsRTTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "__rt")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "compiler", "selfhostrt")); err == nil {
				return dir
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", file)
		}
		if strings.TrimSpace(parent) == "" {
			t.Fatalf("invalid parent while walking from %s", file)
		}
		dir = parent
	}
}

// ---- typed_task_slots_test.go ----

func TestEmitTaskJoinEntrypointsTreatReclaimableTargetsAsDone(t *testing.T) {
	tests := []struct {
		name string
		emit func(e *x64.Emitter, patches *[]callPatch) error
		want []byte
	}{
		{
			name: "join_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinI32(e, false, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "join_result_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinI32(e, true, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "join_until_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinUntilI32(e, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "poll_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskPollI32(e)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "typed_compact",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinTyped(e, 4, patches)
			},
			want: movMem32RdiDispImm32Encoding(actorTaskCountOff, 0),
		},
		{
			name: "typed_staged",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinTyped(e, 5, patches)
			},
			want: movMem32RdiDispImm32Encoding(actorTaskCountOff, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			if err := tt.emit(e, &patches); err != nil {
				t.Fatalf("%s emit: %v", tt.name, err)
			}
			if !containsFragmentsInOrder(
				e.Buf,
				movEaxFromRdiDispEncoding(actorStatusOff),
				cmpEaxImm32Encoding(statusDone),
				cmpEaxImm32Encoding(statusReclaimable),
				tt.want,
			) {
				t.Fatalf("%s must treat reclaimable task targets as done terminal targets", tt.name)
			}
		})
	}
}

func TestEmitTaskJoinEntrypointsMarkDoneTargetsReclaimableAfterResultRead(t *testing.T) {
	reclaimStore := movMem32RdiDispImm32Encoding(actorStatusOff, statusReclaimable)
	tests := []struct {
		name string
		emit func(e *x64.Emitter, patches *[]callPatch) error
		want []byte
	}{
		{
			name: "join_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinI32(e, false, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "join_result_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinI32(e, true, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "join_until_i32",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinUntilI32(e, patches)
			},
			want: movEaxFromRdiDispEncoding(actorExitCodeOff),
		},
		{
			name: "typed_compact",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinTyped(e, 4, patches)
			},
			want: movMem32RdiDispImm32Encoding(actorTaskCountOff, 0),
		},
		{
			name: "typed_staged",
			emit: func(e *x64.Emitter, patches *[]callPatch) error {
				return emitTaskJoinTyped(e, 5, patches)
			},
			want: movMem32RdiDispImm32Encoding(actorTaskCountOff, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			if err := tt.emit(e, &patches); err != nil {
				t.Fatalf("%s emit: %v", tt.name, err)
			}
			if !containsFragmentsInOrder(
				e.Buf,
				movEaxFromRdiDispEncoding(actorStatusOff),
				cmpEaxImm32Encoding(statusDone),
				cmpEaxImm32Encoding(statusReclaimable),
				tt.want,
				reclaimStore,
			) {
				t.Fatalf(
					"%s must mark joined done task targets reclaimable after reading result",
					tt.name,
				)
			}
		})
	}
}

func TestEmitTaskJoinTypedSlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTyped(e, tt.slots, &patches)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTyped(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join slot count") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestEmitTaskJoinTypedClearsTargetResultSlotsAfterJoin(t *testing.T) {
	tests := []struct {
		name             string
		slots            int
		minSlotZeroCount int
	}{
		{name: "compact", slots: 4, minSlotZeroCount: 1},
		{name: "staged", slots: 5, minSlotZeroCount: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			if err := emitTaskJoinTyped(e, tt.slots, &patches); err != nil {
				t.Fatalf("emitTaskJoinTyped(%d): %v", tt.slots, err)
			}

			countClear := movMem32RdiDispImm32Encoding(actorTaskCountOff, 0)
			if got := bytes.Count(e.Buf, countClear); got < 1 {
				t.Fatalf(
					"emitTaskJoinTyped(%d) task count clears = %d, want target result count clear",
					tt.slots,
					got,
				)
			}
			for slot := 0; slot < tt.slots; slot++ {
				slotClear := movMem32RdiDispImm32Encoding(
					actorTaskSlot0Off+int32(slot*4),
					0,
				)
				if got := bytes.Count(e.Buf, slotClear); got < tt.minSlotZeroCount {
					t.Fatalf(
						"emitTaskJoinTyped(%d) slot %d clears = %d, want at least %d",
						tt.slots,
						slot,
						got,
						tt.minSlotZeroCount,
					)
				}
			}
		})
	}
}

func TestEmitTaskResultGetClearsCurrentStagedSlotAfterRead(t *testing.T) {
	e := &x64.Emitter{}
	if err := emitTaskResultGet(e); err != nil {
		t.Fatalf("emitTaskResultGet: %v", err)
	}
	slotClear := movMem32RdiDispImm32Encoding(0, 0)
	if got := bytes.Count(e.Buf, slotClear); got < 1 {
		t.Fatalf("emitTaskResultGet slot clears = %d, want current staged slot clear", got)
	}
}

func TestEmitTaskJoinTypedWrapperWindowsX64SlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTypedWrapperWindowsX64(
				e,
				tt.slots,
				"__tetra_task_join_typed_impl",
				&patches,
			)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTypedWrapperWindowsX64(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join wrapper slots") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
