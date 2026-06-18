package uiplatform

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateReportAcceptsWindowsTargetHostRuntimeEvidence(t *testing.T) {
	if err := ValidateReport([]byte(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
	), Options{
		Target:  "windows-x64",
		Host:    "windows-x64",
		Runtime: "platform-ui-windows-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}); err != nil {
		t.Fatalf("ValidateReport valid windows evidence: %v", err)
	}
}

func TestValidateReportAcceptsMacOSTargetHostRuntimeEvidence(t *testing.T) {
	if err := ValidateReport([]byte(
		validPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"),
	), Options{
		Target:  "macos-x64",
		Host:    "macos-x64",
		Runtime: "platform-ui-macos-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}); err != nil {
		t.Fatalf("ValidateReport valid macos evidence: %v", err)
	}
}

func TestValidateReportRejectsMissingOrStaleGeneratedAt(t *testing.T) {
	opts := Options{
		Target:  "windows-x64",
		Host:    "windows-x64",
		Runtime: "platform-ui-windows-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}
	raw := strings.Replace(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
		`  "generated_at": "2026-05-22T11:00:00Z",`+"\n",
		"",
		1,
	)
	err := ValidateReport([]byte(raw), opts)
	if err == nil || !strings.Contains(err.Error(), "generated_at is required") {
		t.Fatalf("expected missing generated_at rejection, got %v", err)
	}

	raw = strings.Replace(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
		`"generated_at": "2026-05-22T11:00:00Z"`,
		`"generated_at": "2026-05-20T11:00:00Z"`,
		1,
	)
	err = ValidateReport([]byte(raw), opts)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale generated_at rejection, got %v", err)
	}
}

func TestValidateReportRejectsBlockedOrBuildOnlyEvidence(t *testing.T) {
	raw := strings.Replace(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
		`"status": "pass"`,
		`"status": "blocked"`,
		1,
	)
	err := ValidateReport(
		[]byte(raw),
		Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"},
	)
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected blocked report rejection, got %v", err)
	}

	raw = strings.Replace(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
		`"evidence_kind": "target-host-runtime"`,
		`"evidence_kind": "build-only"`,
		1,
	)
	err = ValidateReport(
		[]byte(raw),
		Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"},
	)
	if err == nil || !strings.Contains(err.Error(), "build-only") {
		t.Fatalf("expected build-only report rejection, got %v", err)
	}
}

func TestValidateReportRejectsFakeOrRuntimeLessEvidence(t *testing.T) {
	raw := strings.Replace(
		validPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"),
		`"source": "examples/ui/ui_desktop_runtime_smoke.tetra"`,
		`"source": "docs-only fake runtime-less placeholder"`,
		1,
	)
	err := ValidateReport(
		[]byte(raw),
		Options{Target: "macos-x64", Host: "macos-x64", Runtime: "platform-ui-macos-x64"},
	)
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected fake/runtime-less rejection, got %v", err)
	}
}

func TestValidateReportRejectsFakeRuntimePathPrefix(t *testing.T) {
	raw := strings.Replace(
		validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"),
		`"source": "examples/ui/ui_desktop_runtime_smoke.tetra"`,
		`"source": "fake-runtime.exe"`,
		1,
	)
	err := ValidateReport(
		[]byte(raw),
		Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"},
	)
	if err == nil || !strings.Contains(err.Error(), "fake") {
		t.Fatalf("expected fake runtime path rejection, got %v", err)
	}
}

func mustJSON(v any) string {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(append(raw, '\n'))
}

func validPlatformReport(target string, platform string, runtime string) string {
	exitCode := 0
	return mustJSON(Report{
		Schema:       "tetra.ui.platform.v1",
		GeneratedAt:  "2026-05-22T11:00:00Z",
		Status:       "pass",
		Target:       target,
		Host:         target,
		Platform:     platform,
		Runtime:      runtime,
		UISchema:     "tetra.ui.v1",
		EvidenceKind: "target-host-runtime",
		Source:       "examples/ui/ui_desktop_runtime_smoke.tetra",
		Processes: []ProcessReport{
			process("tetra build", "build", "./tetra", &exitCode),
			process("platform UI app", "app", "./ui-app", &exitCode),
			process("platform UI runtime", "runtime", "./ui-runtime", &exitCode),
			process("platform UI stress", "stress", "./ui-stress", &exitCode),
		},
		Contracts: []ContractReport{
			contract("window lifecycle", "target host process created, showed, closed"),
			contract("widget tree", "target host runtime instantiated widgets"),
			contract("layout", "target host runtime measured and placed widgets"),
			contract("event dispatch", "target host runtime dispatched UI events"),
			contract("state redraw async timers", "target host runtime updated state"),
			contract("negative diagnostics", "target host runtime rejected unsupported UI"),
		},
		Widgets: []WidgetReport{
			widget("AppWindow", "window", "", "app.open", "", "", "open", bounds(0, 0, 640, 480)),
			widget(
				"RootPanel",
				"panel",
				"AppWindow",
				"layout.root",
				"",
				"",
				"",
				bounds(0, 0, 640, 480),
			),
			widget(
				"TitleText",
				"text",
				"RootPanel",
				"state.title",
				"",
				"",
				"Ready",
				bounds(16, 16, 608, 32),
			),
			widget(
				"NameInput",
				"input",
				"RootPanel",
				"state.name",
				"input",
				"",
				"tetra",
				bounds(16, 64, 608, 32),
			),
			widget(
				"ItemList",
				"list",
				"RootPanel",
				"state.items",
				"select",
				"",
				"item-1",
				bounds(16, 112, 608, 240),
			),
			widget(
				"SaveButton",
				"button",
				"RootPanel",
				"state.saved",
				"click",
				"saveAsync",
				"",
				bounds(16, 368, 200, 44),
			),
		},
		Events: []EventReport{
			event(1, "NameInput", "focus", "focusName", "focused", "none", "NameInput"),
			event(2, "NameInput", "input", "setName", "name", "tetra", "tetra-lang"),
			event(3, "NameInput", "change", "commitName", "changed", "false", "true"),
			event(4, "ItemList", "select", "selectItem", "selected", "item-1", "item-2"),
			event(5, "SaveButton", "click", "saveAsync", "saved", "false", "true"),
			event(6, "AppWindow", "tick", "timerTick", "dirty", "true", "false"),
		},
		Cases: []CaseReport{
			testCase("window lifecycle", "positive", ""),
			testCase("widget tree", "positive", ""),
			testCase("layout measure and place", "positive", ""),
			testCase("event dispatch", "positive", ""),
			testCase("state update redraw", "positive", ""),
			testCase("async command completion", "positive", ""),
			testCase("timer tick", "positive", ""),
			testCase("unsupported feature diagnostic", "negative", "unsupported UI feature"),
			testCase("invalid widget diagnostic", "negative", "unknown widget"),
			testCase("stress deterministic event loop", "stress", ""),
		},
	})
}

func process(name string, kind string, path string, exitCode *int) ProcessReport {
	return ProcessReport{Name: name, Kind: kind, Path: path, Ran: true, Pass: true, ExitCode: exitCode}
}

func contract(name string, evidence string) ContractReport {
	return ContractReport{Name: name, Status: "pass", Evidence: evidence}
}

func widget(
	id string,
	kind string,
	parent string,
	binding string,
	eventName string,
	command string,
	value string,
	b Bounds,
) WidgetReport {
	return WidgetReport{
		ID:      id,
		Kind:    kind,
		Parent:  parent,
		Binding: binding,
		Event:   eventName,
		Command: command,
		Value:   value,
		Enabled: true,
		Visible: true,
		Bounds:  b,
	}
}

func bounds(x int, y int, width int, height int) Bounds {
	return Bounds{X: x, Y: y, Width: width, Height: height}
}

func event(
	order int,
	widgetID string,
	eventName string,
	command string,
	field string,
	before string,
	after string,
) EventReport {
	ops := []OperationReport{
		operation(eventName, "state."+field, after, field, after),
	}
	switch eventName {
	case "input", "select":
		ops = []OperationReport{
			operation("state_set", "state."+field, after, field, after),
		}
	case "click":
		ops = []OperationReport{
			operation("async_command", "command."+command, "completed", field, after),
			operation("redraw", "AppWindow", "scheduled", "dirty", "true"),
		}
	case "tick":
		ops = []OperationReport{
			operation("timer_tick", "timer.redraw", "fired", field, after),
			operation("redraw", "AppWindow", "completed", field, after),
		}
	}
	return EventReport{
		Order:       order,
		WidgetID:    widgetID,
		Event:       eventName,
		Command:     command,
		Pass:        true,
		BeforeState: map[string]string{field: before},
		AfterState:  map[string]string{field: after},
		Operations:  ops,
		WidgetUpdates: []WidgetUpdateReport{
			{ID: widgetID, Before: before, After: after},
		},
	}
}

func operation(
	kind string,
	target string,
	value string,
	stateField string,
	stateValue string,
) OperationReport {
	return OperationReport{
		Kind:       kind,
		Target:     target,
		Value:      value,
		StateField: stateField,
		StateValue: stateValue,
	}
}

func testCase(name string, kind string, expectedError string) CaseReport {
	return CaseReport{
		Name:          name,
		Kind:          kind,
		Ran:           true,
		Pass:          true,
		ExpectedError: expectedError,
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
