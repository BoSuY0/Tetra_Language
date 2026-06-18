package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceipc"
)

func TestValidateSurfaceIPCReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandIPCReport()
	reportPath := filepath.Join(dir, "surface-ipc-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceIPCReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface IPC/lifecycle report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceIPCReportCommandRejectsSurfaceFrameTransfer(t *testing.T) {
	dir := t.TempDir()
	report := commandIPCReport()
	report.Messages[0].ContainsSurfaceFrame = true
	report.Messages[0].PayloadKind = "surface-frame"
	report.Messages[0].Accepted = true
	reportPath := filepath.Join(dir, "surface-ipc-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceIPCReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "surface frame") {
		t.Fatalf("stderr = %q, want surface frame rejection", stderr.String())
	}
}

func commandIPCReport() surfaceipc.Report {
	return surfaceipc.Report{
		Schema:       surfaceipc.SchemaV1,
		Status:       "pass",
		Level:        surfaceipc.LevelIPCLifecycleV1,
		Scope:        "surface-v1-scoped-linux-web-ipc-lifecycle",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		App: surfaceipc.AppModel{
			Main:               "surface-app-main",
			UIIsolate:          "surface-ui-isolate",
			UIThreadPolicy:     "single-owner-ui-dispatcher-v1",
			BackgroundServices: []string{"asset-indexer", "settings-loader"},
			Lifecycle: []surfaceipc.LifecycleStep{
				{Name: "launch", Phase: "start", UIThread: true, DispatcherRouted: true, Evidence: "app main starts UI isolate"},
				{Name: "suspend-background", Phase: "suspend", UIThread: false, DispatcherRouted: true, Evidence: "background services quiesced through dispatcher"},
				{Name: "shutdown", Phase: "stop", UIThread: true, DispatcherRouted: true, Evidence: "owned shutdown order"},
			},
		},
		Messages: []surfaceipc.Message{
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
				ContainsSurfaceHandle: true,
				Typed:                 true,
				Accepted:              false,
				Evidence:              "Surface handle cannot leave UI isolate",
			},
		},
		UIUpdates: []surfaceipc.UIUpdate{
			{Name: "apply-settings", Source: "background-task", Target: "settings-panel", MutatesUI: true, DispatcherRouted: true, Allowed: true, Evidence: "dispatcher applies owned snapshot"},
			{Name: "direct-background-mutation-rejected", Source: "background-task", Target: "settings-panel", MutatesUI: true, DispatcherRouted: false, Allowed: false, Evidence: "background direct UI mutation rejected"},
		},
		CrashIsolation: surfaceipc.CrashIsolation{
			Strategy:                 "supervised-background-services-v1",
			UIStatePreserved:         true,
			BackgroundServiceRestart: true,
			CrashReport:              true,
			Evidence:                 "background service crash reported and restarted",
		},
		Operations: []surfaceipc.Operation{
			{Name: "app lifecycle validated", Kind: "lifecycle", Ran: true, Pass: true},
			{Name: "owned message passing validated", Kind: "ipc", Ran: true, Pass: true},
			{Name: "dispatcher UI updates validated", Kind: "dispatcher", Ran: true, Pass: true},
			{Name: "crash isolation strategy validated", Kind: "crash-isolation", Ran: true, Pass: true},
		},
		NegativeGuards: surfaceipc.NegativeGuards{
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
		Cases: []surfaceipc.CaseReport{
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

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
