package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/tools/validators/uiprod"
)

func TestBuildReportProducesValidUIProductionRuntimeEvidence(t *testing.T) {
	widgets, events, cases, err := runDesktopRuntimeScenario()
	if err != nil {
		t.Fatalf("runDesktopRuntimeScenario failed: %v", err)
	}
	cases = append(cases, uiprod.CaseReport{Name: "dogfood application smoke", Kind: "positive", Ran: true, Pass: true})
	cases = append(cases, uiprod.CaseReport{Name: "widget tree stress", Kind: "stress", Ran: true, Pass: true})
	cases = append(cases, uiprod.CaseReport{Name: "compiler UI bundle runtime load", Kind: "positive", Ran: true, Pass: true})
	cases = append(cases, uiprod.CaseReport{Name: "native shell runtime integration", Kind: "positive", Ran: true, Pass: true})
	cases = append(cases, uiprod.CaseReport{Name: "native runtime sidecar consistency", Kind: "positive", Ran: true, Pass: true})
	report := buildReport("tools/cmd/ui-production-runtime-smoke", []uiprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "desktop UI app", Kind: "app", Path: "/tmp/ui-desktop", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "desktop UI runtime", Kind: "runtime", Path: "tools/cmd/ui-production-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "native shell runtime integration", Kind: "runtime", Path: "go run ./tools/cmd/native-ui-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "native runtime evidence validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-native-ui-runtime", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "desktop UI widget stress", Kind: "stress", Path: "tools/cmd/ui-production-runtime-smoke --internal-check stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, widgets, events, cases)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := uiprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(events) != 6 {
		t.Fatalf("events = %d, want focus, input, change, select, click, and timer tick events", len(events))
	}
	if !hasEvent(events, "focus") {
		t.Fatalf("events missing input focus transition")
	}
	if !hasEvent(events, "change") {
		t.Fatalf("events missing input change transition")
	}
	if !hasCase(cases, "input focus traversal") {
		t.Fatalf("cases missing input focus traversal")
	}
	if !hasCase(cases, "input change commit") {
		t.Fatalf("cases missing input change commit")
	}
	if !hasCase(cases, "native runtime sidecar consistency") {
		t.Fatalf("cases missing native runtime sidecar consistency")
	}
	saveEvent := events[4]
	if saveEvent.Operations[0].Kind != "async_command" {
		t.Fatalf("save event first operation = %q, want async_command", saveEvent.Operations[0].Kind)
	}
	if saveEvent.Operations[1].Kind != "redraw" {
		t.Fatalf("save event second operation = %q, want redraw", saveEvent.Operations[1].Kind)
	}
	timerEvent := events[5]
	if timerEvent.Event != "tick" {
		t.Fatalf("timer event kind = %q, want tick", timerEvent.Event)
	}
	if timerEvent.Operations[0].Kind != "timer_tick" {
		t.Fatalf("timer event first operation = %q, want timer_tick", timerEvent.Operations[0].Kind)
	}
}

func hasEvent(events []uiprod.EventReport, kind string) bool {
	for _, event := range events {
		if event.Event == kind {
			return true
		}
	}
	return false
}

func hasCase(cases []uiprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}

func TestDesktopRuntimeScenarioRejectsInvalidAndCrashPaths(t *testing.T) {
	rt := newDesktopRuntime()
	if _, err := rt.dispatch("__missing__", "click", "", 1); err == nil || !strings.Contains(err.Error(), "unknown widget") {
		t.Fatalf("invalid widget error = %v, want unknown widget", err)
	}
	if _, err := rt.dispatch("SaveButton", "click", "__missing__", 1); err == nil || !strings.Contains(err.Error(), "command failed") {
		t.Fatalf("unknown command error = %v, want command failed", err)
	}
	if err := rt.recoverCrash(func() { panic("runtime panic recovered") }); err == nil || !strings.Contains(err.Error(), "runtime panic recovered") {
		t.Fatalf("crash recovery error = %v, want runtime panic recovered", err)
	}
}
