package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/actorsystem"
)

const okVerdict = "TETRA_ACTOR_SYSTEM_MESSAGES_V1_OK"

func main() {
	root := flag.String("root", ".", "repository root")
	reportDir := flag.String("report-dir", "", "optional actor system-message report directory")
	currentGitHead := flag.String(
		"current-git-head",
		"",
		"optional git head expected in the actor system-message report",
	)
	final := flag.Bool("final", false, "require clean-git report evidence for final validation")
	flag.Parse()
	if err := validateActorSystemMessages(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *reportDir != "" {
		if err := validateActorSystemMessagesReportDir(*reportDir, *currentGitHead, *final); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println(okVerdict)
}

func validateActorSystemMessagesReportDir(
	reportDir string,
	currentGitHead string,
	requireCleanGit bool,
) error {
	return actorsystem.ValidateReportDir(reportDir, actorsystem.Options{
		CurrentGitHead:  strings.TrimSpace(currentGitHead),
		RequireCleanGit: requireCleanGit,
	})
}

func validateActorSystemMessages(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	checks := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: "docs/design/actor_system_messages_v1.md",
			required: []string{
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
			},
			forbidden: []string{"actorRefSlots + 7"},
		},
		{
			path: "lib/core/actors/actors.tetra",
			required: []string{
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
			},
			forbidden: []string{"SystemMessage.unknown"},
		},
		{
			path: "compiler/internal/formats/formats.go",
			required: []string{
				`case "actors":`,
				`return "actors"`,
			},
		},
		{
			path: "compiler/internal/formats/formats_test.go",
			required: []string{
				"TestModuleCandidateRelPathsIncludesActorsStdlibBucket",
				"lib/core/actors/actors.tetra",
			},
		},
		{
			path: "tools/validators/actorsystem/report.go",
			required: []string{
				`SchemaV1 = "tetra.actor.system_messages.v1"`,
				"ValidateReportDir",
				"ArtifactHashSchema",
				"ReleaseTestInjectorExported",
				"__tetra_test_actor_system_inject",
				`Producer != "test_hook"`,
			},
		},
		{
			path: "tools/validators/actorsystem/report_test.go",
			required: []string{
				"TestValidateReportAcceptsP01FixtureEvidence",
				"TestValidateReportRejectsExportedTestInjector",
				"TestValidateReportDirCrossChecksHashManifest",
			},
		},
		{
			path: "scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh",
			required: []string{
				"actor-system-messages-linux-x64.json",
				"actor-system-layout-linux-x64.json",
				"actor-system-layout-report",
				"artifact-hashes.json",
				"__tetra_test_actor_system_inject",
				"producer=test_hook",
				"validate-actor-system-messages",
			},
		},
		{
			path: "docs/contracts/actors/system-message-runtime-lane-linux-x64.release-contract.v1.json",
			required: []string{
				"actor-system-messages-linux-x64.json",
				"actor-system-layout-linux-x64.json",
				"artifact-hashes.json",
				"scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh",
			},
		},
		{
			path: "compiler/internal/semantics/semantics_core.go",
			required: []string{
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
			},
		},
		{
			path: "compiler/internal/semantics/semantics_expressions.go",
			required: []string{
				"isRuntimeSystemMessageSurfaceType",
				"lib.core.actors.SystemMessage",
				"lib.core.actors.SystemReceiveResult",
				"runtime system messages cannot be sent through the ordinary actor mailbox",
			},
		},
		{
			path: "compiler/internal/semantics/semantics_checker.go",
			required: []string{
				"isFreshSystemReceiveResourceReturn",
				"bindFreshResourceTree",
				`"core.actor_recv_system_poll"`,
				`"lib.core.actors.poll_system"`,
			},
		},
		{
			path: "compiler/internal/lower/lower_expressions.go",
			required: []string{
				"__tetra_actor_recv_system_begin",
				"__tetra_actor_recv_system_slot",
				"__tetra_actor_recv_system_count",
				"__tetra_actor_status_raw",
			},
		},
		{
			path: "compiler/internal/runtimeabi/runtimeabi.go",
			required: []string{
				"RequiredActorSystemReceiveSymbols",
				`"__tetra_actor_status_raw"`,
				`"__tetra_actor_recv_system_begin"`,
				`"__tetra_actor_recv_system_slot"`,
				`"__tetra_actor_recv_system_count"`,
			},
		},
		{
			path: "compiler/internal/actorsrt/actorsrt_core.go",
			required: []string{
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
			},
			forbidden: []string{
				"msgSystemKindOff, actorSystemKindExit",
				"msgSystemKindOff, actorSystemKindDown",
				"msgSystemKindOff, actorSystemKindNodeDown",
			},
		},
		{
			path: "compiler/internal/semantics/semantics_suite_test.go",
			required: []string{
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
			},
		},
		{
			path: "compiler/compiler_suite_test.go",
			required: []string{
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
			},
		},
		{
			path: "compiler/internal/actorsrt/actorsrt_suite_test.go",
			required: []string{
				"TestActorRuntimeSystemMessageQueueLayoutIsStable",
				"TestLinuxRuntimeUserAndSystemMailboxWakeKindsAreSeparated",
				"TestLinuxRuntimeSystemReceiveReturnsCanceledOnTaskGroupCancellation",
				"TestLinuxRuntimeSystemReceiveReturnsRuntimeClosedOnSchedulerShutdown",
				"TestLinuxRuntimeSystemEventReservationCreditsAreAccounted",
				"TestLinuxRuntimeObjectExportsActorSystemReceiveSymbols",
				"TestLinuxRuntimeActorStatusRawDistinguishesInvalidAndStale",
				"TestLinuxRuntimeActorWaitInvalidAndStaleResultsUsePublicDeadStatus",
			},
		},
		{
			path: "examples/actors/system_messages/system_user_queue_isolation.tetra",
			required: []string{
				"actors.poll_system()",
				"SystemReceiveResult.empty",
				"core.recv_msg()",
			},
		},
		{
			path: "examples/actors/system_messages/system_exit_trap.tetra",
			required: []string{
				"SystemMessage.exit",
				"ExitReason.node_down",
				"actors.poll_system()",
			},
		},
		{
			path: "examples/actors/system_messages/system_monitor_down.tetra",
			required: []string{
				"SystemMessage.down",
				"actor.monitor",
				"actors.poll_system()",
			},
		},
		{
			path: "examples/actors/system_messages/system_poll_timeout_cancel.tetra",
			required: []string{
				"actors.poll_system()",
				"actors.recv_system_until(0)",
				"SystemReceiveResult.timeout",
				"SystemReceiveResult.empty",
			},
		},
		{
			path: "examples/actors/system_messages/system_sender_unchanged.tetra",
			required: []string{
				"core.sender()",
				"actors.poll_system()",
				"SystemReceiveResult.empty",
			},
		},
		{
			path: "examples/actors/system_messages/system_forgery_negative.tetra",
			required: []string{
				"SystemMessage.exit",
				"runtime system messages cannot be sent through the ordinary actor mailbox",
				"NEGATIVE",
			},
		},
	}
	var issues []string
	for _, check := range checks {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(check.path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", check.path, err))
			continue
		}
		text := string(raw)
		for _, snippet := range check.required {
			if !strings.Contains(text, snippet) {
				issues = append(issues, fmt.Sprintf("%s: missing required snippet %q", check.path, snippet))
			}
		}
		for _, snippet := range check.forbidden {
			if strings.Contains(text, snippet) {
				issues = append(issues, fmt.Sprintf("%s: forbidden snippet present %q", check.path, snippet))
			}
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "\n"))
	}
	return nil
}
