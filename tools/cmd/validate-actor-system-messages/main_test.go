package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateActorSystemMessagesAcceptsRepositoryState(t *testing.T) {
	if err := validateActorSystemMessages(repoRoot(t)); err != nil {
		t.Fatalf("validateActorSystemMessages: %v", err)
	}
}

func TestValidateActorSystemMessagesReportDirRejectsMissingReport(t *testing.T) {
	err := validateActorSystemMessagesReportDir(t.TempDir(), strings.Repeat("a", 40), false)
	if err == nil {
		t.Fatalf("expected missing report rejection")
	}
	if !strings.Contains(err.Error(), "actor-system-messages-linux-x64.json") {
		t.Fatalf("error = %v, want actor-system-messages-linux-x64.json", err)
	}
}

func TestValidateActorSystemMessagesRejectsOldSharedSystemKindStore(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"docs/design/actor_system_messages_v1.md": strings.Join([]string{
			"Owner approval: `TETRA-V1-ACTOR-SYSTEM-MESSAGES-B-2026-06-20`",
			"separate runtime-owned system queue",
			"actor.system_recv_raw",
			"eight slots",
			"`__tetra_actor_recv_system_count()` returns `7`",
			"actor.status_result_raw",
			"core.actor_status_raw",
			"pub enum ActorStatus:",
			"pub func status(target: actor) -> StatusResult uses actors",
			"pub func set_trap_exit(enabled: Bool) -> Bool uses actors",
		}, "\n"),
		"lib/core/actors/actors.tetra": strings.Join([]string{
			"module lib.core.actors",
			"pub enum ExitReason:",
			"pub enum NodeDownReason:",
			"pub enum SystemMessage:",
			"case down(actor.monitor, actor, ExitReason)",
			"case node_down(actor.node, NodeDownReason)",
			"pub enum SystemReceiveResult:",
			"case invalid_state(Int)",
			"pub enum ActorStatus:",
			"case exited_error(Int)",
			"pub enum StatusResult:",
			"case ok(ActorStatus)",
			"pub enum WaitResult:",
			"case exited(ExitReason)",
			"pub enum StopResult:",
			"case already_exited(ExitReason)",
			"pub enum LinkResult:",
			"case target_exited(ExitReason)",
			"pub enum MonitorResult:",
			"case monitoring(actor.monitor)",
			"pub func recv_system() -> SystemReceiveResult",
			"pub func poll_system() -> SystemReceiveResult",
			"pub func recv_system_until(deadline: Int) -> SystemReceiveResult",
			"pub func status(target: actor) -> StatusResult",
			"pub func wait(target: actor) -> WaitResult",
			"pub func wait_until(target: actor, deadline: Int) -> WaitResult",
			"pub func stop(target: actor, reason: ExitReason) -> StopResult",
			"pub func link(target: actor) -> LinkResult",
			"pub func unlink(target: actor) -> Bool",
			"pub func monitor(target: actor) -> MonitorResult",
			"pub func demonitor(reference: actor.monitor, flush: Bool) -> Bool",
			"pub func set_trap_exit(enabled: Bool) -> Bool",
			"core.actor_status_raw(target)",
			"let before: StatusResult = status(target)",
			"core.actor_recv_system()",
			"core.actor_recv_system_poll()",
			"core.actor_recv_system_until(deadline)",
		}, "\n"),
		"compiler/internal/formats/formats.go": strings.Join([]string{
			`case "actors":`,
			`return "actors"`,
		}, "\n"),
		"compiler/internal/formats/formats_test.go": strings.Join([]string{
			"TestModuleCandidateRelPathsIncludesActorsStdlibBucket",
			"lib/core/actors/actors.tetra",
		}, "\n"),
		"tools/validators/actorsystem/report.go": strings.Join([]string{
			`SchemaV1 = "tetra.actor.system_messages.v1"`,
			"ValidateReportDir",
			"ArtifactHashSchema",
			"ReleaseTestInjectorExported",
			"__tetra_test_actor_system_inject",
			`Producer != "test_hook"`,
		}, "\n"),
		"tools/validators/actorsystem/report_test.go": strings.Join([]string{
			"TestValidateReportAcceptsP01FixtureEvidence",
			"TestValidateReportRejectsExportedTestInjector",
			"TestValidateReportDirCrossChecksHashManifest",
		}, "\n"),
		"scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh": strings.Join([]string{
			"actor-system-messages-linux-x64.json",
			"actor-system-layout-linux-x64.json",
			"actor-system-layout-report",
			"artifact-hashes.json",
			"__tetra_test_actor_system_inject",
			"producer=test_hook",
			"validate-actor-system-messages",
		}, "\n"),
		"docs/contracts/actors/system-message-runtime-lane-linux-x64.release-contract.v1.json": strings.Join([]string{
			"actor-system-messages-linux-x64.json",
			"actor-system-layout-linux-x64.json",
			"artifact-hashes.json",
			"scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh",
		}, "\n"),
		"compiler/internal/semantics/semantics_core.go": strings.Join([]string{
			`"actor.node"`,
			`"actor.status_result_raw"`,
			`"actor.system_recv_raw"`,
			`"core.actor_status_raw"`,
			`"core.actor_recv_system"`,
			`"core.actor_recv_system_poll"`,
			`"core.actor_recv_system_until"`,
			"RuntimeOwned",
			"UserConstructible",
			"ActorSendable",
		}, "\n"),
		"compiler/internal/semantics/semantics_expressions.go": strings.Join([]string{
			"isRuntimeSystemMessageSurfaceType",
			"lib.core.actors.SystemMessage",
			"lib.core.actors.SystemReceiveResult",
			"runtime system messages cannot be sent through the ordinary actor mailbox",
		}, "\n"),
		"compiler/internal/semantics/semantics_checker.go": strings.Join([]string{
			"isFreshSystemReceiveResourceReturn",
			"bindFreshResourceTree",
			`"core.actor_recv_system_poll"`,
			`"lib.core.actors.poll_system"`,
		}, "\n"),
		"compiler/internal/lower/lower_expressions.go": strings.Join([]string{
			"__tetra_actor_recv_system_begin",
			"__tetra_actor_recv_system_slot",
			"__tetra_actor_recv_system_count",
			"__tetra_actor_status_raw",
		}, "\n"),
		"compiler/internal/runtimeabi/runtimeabi.go": strings.Join([]string{
			"RequiredActorSystemReceiveSymbols",
			`"__tetra_actor_status_raw"`,
			`"__tetra_actor_recv_system_begin"`,
			`"__tetra_actor_recv_system_slot"`,
			`"__tetra_actor_recv_system_count"`,
		}, "\n"),
		"compiler/internal/actorsrt/actorsrt_core.go": strings.Join([]string{
			"systemEventKindOff",
			"actorSystemMailboxHeadOff",
			"schedSystemEventBaseOff",
			"emitCheckedSystemEventPoolAlloc",
			"emitEnqueueSystemEventForActorPtrInRdiNodeInRdx",
			"maxSystemEventReservedCredits",
			"schedSystemEventReservedCreditsOff",
			"actorSystemMailboxReservedCreditsOff",
			"emitReserveSystemEventCreditForActorPtrInRdi",
			"emitReleaseSystemEventCreditForActorPtrInRdi",
			"emitRecvSystemBegin",
			"emitWakeActorIfBlockedOnUserMailboxInRdi",
			"emitCurrentTaskGroupCanceledCheck",
			"actorSystemRecvStatusCanceled",
			"schedRuntimeClosingOff",
			"emitMarkRuntimeClosingAndWakeSystemWaiters",
			"actorSystemRecvStatusRuntimeClosed",
			"actorWaitKindUser",
			"actorWaitKindSystem",
			"emitDecodeLocalActorRefInRdiRsiToEcxClassified",
			"emitActorStatusRaw",
			"msgSystemKindOff, actorSystemKindDown",
		}, "\n"),
		"compiler/internal/semantics/semantics_suite_test.go": strings.Join([]string{
			"TestActorSystemReceiveTypesAndBuiltinSignaturesUseRawContractSlots",
			"TestLibCoreActorsSystemReceiveSurfaceChecksAgainstRawBuiltins",
			"TestLibCoreActorsSystemReceiveResultPayloadsCanBePatternMatched",
			"TestLibCoreActorsLifecycleSurfaceChecksAgainstRawBuiltins",
			"core.actor_status_raw",
			"actor.status_result_raw",
			"TestLibCoreActorsSystemMessageCannotUseOrdinaryActorMailbox",
			"TestLibCoreActorsSystemMessageCannotUseOrdinaryTypedReceive",
			"core.recv_typed<actors.SystemMessage>()",
			"TestUserDefinedSystemMessageCanUseOrdinaryTypedActorMailbox",
		}, "\n"),
		"compiler/compiler_suite_test.go": strings.Join([]string{
			"TestLibCoreActorsLifecycleWrappersBuildAndRun",
			"TestLibCoreActorsStatusResultInvalidAndStaleBuildAndRun",
			"TestLibCoreActorsLifecycleWrappersInvalidAndStaleTaxonomyBuildAndRun",
			"TestLibCoreActorsWaitInvalidAndStaleTaxonomyBuildAndRun",
			"actors.status(core.self())",
			"actors.wait(normal)",
			"actors.wait_until(timed_peer, core.deadline_ms(1))",
			"actors.stop(stoppable, actors.ExitReason.normal)",
			"actors.link(linked_peer)",
			"actors.monitor(monitored_peer)",
			"actors.set_trap_exit(true)",
		}, "\n"),
		"compiler/internal/actorsrt/actorsrt_suite_test.go": strings.Join([]string{
			"TestActorRuntimeSystemMessageQueueLayoutIsStable",
			"TestLinuxRuntimeUserAndSystemMailboxWakeKindsAreSeparated",
			"TestLinuxRuntimeSystemReceiveReturnsCanceledOnTaskGroupCancellation",
			"TestLinuxRuntimeSystemReceiveReturnsRuntimeClosedOnSchedulerShutdown",
			"TestLinuxRuntimeSystemEventReservationCreditsAreAccounted",
			"TestLinuxRuntimeObjectExportsActorSystemReceiveSymbols",
			"TestLinuxRuntimeActorStatusRawDistinguishesInvalidAndStale",
			"TestLinuxRuntimeActorWaitInvalidAndStaleResultsUsePublicDeadStatus",
		}, "\n"),
	}
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := validateActorSystemMessages(root)
	if err == nil {
		t.Fatalf("expected forbidden old system kind store rejection")
	}
	if !strings.Contains(err.Error(), "msgSystemKindOff, actorSystemKindDown") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateActorSystemMessagesRequiresRecvTypedSystemMessageNegative(t *testing.T) {
	root := t.TempDir()
	copyActorSystemMessageValidatorInputs(t, root)
	semanticsPath := filepath.Join(
		root,
		filepath.FromSlash("compiler/internal/semantics/semantics_suite_test.go"),
	)
	raw, err := os.ReadFile(semanticsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(
		string(raw),
		"TestLibCoreActorsSystemMessageCannotUseOrdinaryTypedReceive",
		"",
	)
	text = strings.ReplaceAll(text, "core.recv_typed<actors.SystemMessage>()", "")
	if err := os.WriteFile(semanticsPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateActorSystemMessages(root)
	if err == nil {
		t.Fatalf("expected missing ordinary typed receive negative coverage rejection")
	}
	if !strings.Contains(err.Error(), "TestLibCoreActorsSystemMessageCannotUseOrdinaryTypedReceive") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateActorSystemMessagesRequiresLibCoreActorsLifecycleSurface(t *testing.T) {
	root := t.TempDir()
	copyActorSystemMessageValidatorInputs(t, root)
	libPath := filepath.Join(root, filepath.FromSlash("lib/core/actors/actors.tetra"))
	raw, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "pub enum ActorStatus:", "")
	text = strings.ReplaceAll(text, "pub func status(target: actor) -> StatusResult", "")
	if err := os.WriteFile(libPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateActorSystemMessages(root)
	if err == nil {
		t.Fatalf("expected missing lib.core.actors lifecycle API rejection")
	}
	if !strings.Contains(err.Error(), "pub enum ActorStatus:") ||
		!strings.Contains(err.Error(), "pub func status(target: actor) -> StatusResult") {
		t.Fatalf("error = %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func copyActorSystemMessageValidatorInputs(t *testing.T, root string) {
	t.Helper()
	sourceRoot := repoRoot(t)
	paths := []string{
		"docs/design/actor_system_messages_v1.md",
		"lib/core/actors/actors.tetra",
		"compiler/internal/formats/formats.go",
		"compiler/internal/formats/formats_test.go",
		"tools/validators/actorsystem/report.go",
		"tools/validators/actorsystem/report_test.go",
		"scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh",
		"docs/contracts/actors/system-message-runtime-lane-linux-x64.release-contract.v1.json",
		"compiler/internal/semantics/semantics_core.go",
		"compiler/internal/semantics/semantics_expressions.go",
		"compiler/internal/semantics/semantics_checker.go",
		"compiler/internal/lower/lower_expressions.go",
		"compiler/internal/runtimeabi/runtimeabi.go",
		"compiler/internal/actorsrt/actorsrt_core.go",
		"compiler/internal/semantics/semantics_suite_test.go",
		"compiler/internal/actorsrt/actorsrt_suite_test.go",
		"examples/actors/system_messages/system_user_queue_isolation.tetra",
		"examples/actors/system_messages/system_exit_trap.tetra",
		"examples/actors/system_messages/system_monitor_down.tetra",
		"examples/actors/system_messages/system_poll_timeout_cancel.tetra",
		"examples/actors/system_messages/system_sender_unchanged.tetra",
		"examples/actors/system_messages/system_forgery_negative.tetra",
	}
	for _, rel := range paths {
		source := filepath.Join(sourceRoot, filepath.FromSlash(rel))
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		dest := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
