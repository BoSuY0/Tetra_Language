package surfaceipc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsIPCProcessLifecycleEvidence(t *testing.T) {
	raw := mustMarshal(t, validIPCReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsSurfaceHandleActorBoundaryCrossing(t *testing.T) {
	report := validIPCReport()
	report.Messages = append(report.Messages, Message{
		Name:                  "bad-handle-transfer",
		Direction:             "background-to-ui",
		PayloadKind:           "surface-handle",
		OwnedData:             true,
		ContainsSurfaceHandle: true,
		DispatcherRouted:      true,
		Typed:                 true,
		Accepted:              true,
		Evidence:              "unsafe Surface handle crossed actor boundary",
	})
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "surface handle") {
		t.Fatalf("expected surface handle boundary rejection, got %v", err)
	}
}

func TestValidateReportRejectsSurfaceFrameActorMessage(t *testing.T) {
	report := validIPCReport()
	report.Messages[0].ContainsSurfaceFrame = true
	report.Messages[0].PayloadKind = "surface-frame"
	report.Messages[0].Accepted = true
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "surface frame") {
		t.Fatalf("expected surface frame actor-message rejection, got %v", err)
	}
}

func TestValidateReportRejectsSurfaceEventActorMessage(t *testing.T) {
	report := validIPCReport()
	report.Messages[0].ContainsSurfaceEvent = true
	report.Messages[0].PayloadKind = "surface-event"
	report.Messages[0].Accepted = true
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "surface event") {
		t.Fatalf("expected surface event actor-message rejection, got %v", err)
	}
}

func TestValidateReportRejectsBorrowedPayload(t *testing.T) {
	report := validIPCReport()
	report.Messages[0].OwnedData = false
	report.Messages[0].PayloadKind = "borrowed-view"
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "owned") {
		t.Fatalf("expected owned payload rejection, got %v", err)
	}
}

func TestValidateReportRejectsBackgroundUIMutationWithoutDispatcher(t *testing.T) {
	report := validIPCReport()
	report.UIUpdates = append(report.UIUpdates, UIUpdate{
		Name:             "unsafe-background-mutation",
		Source:           "background-task",
		Target:           "surface-state",
		MutatesUI:        true,
		DispatcherRouted: false,
		Allowed:          true,
		Evidence:         "background task wrote UI state directly",
	})
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "dispatcher") {
		t.Fatalf("expected dispatcher-only UI update rejection, got %v", err)
	}
}

func TestValidateReportRejectsMissingCrashIsolation(t *testing.T) {
	report := validIPCReport()
	report.CrashIsolation.BackgroundServiceRestart = false
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "crash") {
		t.Fatalf("expected crash isolation rejection, got %v", err)
	}
}

func validIPCReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelIPCLifecycleV1,
		Scope:        "surface-v1-scoped-linux-web-ipc-lifecycle",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		App: AppModel{
			Main:               "surface-app-main",
			UIIsolate:          "surface-ui-isolate",
			UIThreadPolicy:     "single-owner-ui-dispatcher-v1",
			BackgroundServices: []string{"asset-indexer", "settings-loader"},
			Lifecycle: []LifecycleStep{
				{Name: "launch", Phase: "start", UIThread: true, DispatcherRouted: true, Evidence: "app main starts UI isolate"},
				{Name: "suspend-background", Phase: "suspend", UIThread: false, DispatcherRouted: true, Evidence: "background services quiesced through dispatcher"},
				{Name: "shutdown", Phase: "stop", UIThread: true, DispatcherRouted: true, Evidence: "owned shutdown order"},
			},
		},
		Messages: []Message{
			{
				Name:             "settings-loaded",
				Direction:        "background-to-ui",
				PayloadKind:      "owned-snapshot",
				OwnedData:        true,
				DispatcherRouted: true,
				Typed:            true,
				Accepted:         true,
				Evidence:         "owned settings snapshot delivered through UI dispatcher",
			},
			{
				Name:                  "surface-handle-transfer-rejected",
				Direction:             "ui-to-background",
				PayloadKind:           "surface-handle",
				OwnedData:             false,
				ContainsSurfaceHandle: true,
				DispatcherRouted:      false,
				Typed:                 true,
				Accepted:              false,
				Evidence:              "Surface handle cannot leave UI isolate",
			},
		},
		UIUpdates: []UIUpdate{
			{Name: "apply-settings", Source: "background-task", Target: "settings-panel", MutatesUI: true, DispatcherRouted: true, Allowed: true, Evidence: "dispatcher applies owned snapshot"},
			{Name: "direct-background-mutation-rejected", Source: "background-task", Target: "settings-panel", MutatesUI: true, DispatcherRouted: false, Allowed: false, Evidence: "background direct UI mutation rejected"},
		},
		CrashIsolation: CrashIsolation{
			Strategy:                 "supervised-background-services-v1",
			UIStatePreserved:         true,
			BackgroundServiceRestart: true,
			CrashReport:              true,
			Evidence:                 "background service crash reported and restarted without transferring Surface handles",
		},
		Operations: []Operation{
			{Name: "app lifecycle validated", Kind: "lifecycle", Ran: true, Pass: true},
			{Name: "owned message passing validated", Kind: "ipc", Ran: true, Pass: true},
			{Name: "dispatcher UI updates validated", Kind: "dispatcher", Ran: true, Pass: true},
			{Name: "crash isolation strategy validated", Kind: "crash-isolation", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
			SurfaceHandleActorTransferRejected:            true,
			SurfaceFrameActorMessageRejected:              true,
			SurfaceEventActorMessageRejected:              true,
			BackgroundUIMutationWithoutDispatcherRejected: true,
			BorrowedPayloadRejected:                       true,
			UntypedChannelRejected:                        true,
			CrashIsolationRequired:                        true,
		},
		NonClaims: []string{
			"No unsafe shared Surface handles across actor/task boundaries.",
			"No Electron main/renderer parity claim.",
			"No process sandbox parity claim beyond the scoped Surface security report.",
			"No automatic crash recovery claim beyond supervised background services.",
		},
		Cases: []CaseReport{
			{Name: "owned background message dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "surface handle actor transfer rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "surface frame actor message rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "surface event actor message rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "background UI mutation without dispatcher rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "borrowed payload rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "untyped IPC channel rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
