package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/uiprod"
)

func TestValidateUIProductionRuntimeReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-production.json")
	if err := os.WriteFile(path, []byte(validUIProductionRuntimeReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateUIProductionRuntimeReport(path); err != nil {
		t.Fatalf("validateUIProductionRuntimeReport failed: %v", err)
	}
}

func TestValidateUIProductionRuntimeReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-production.json")
	raw := strings.Replace(
		validUIProductionRuntimeReport(),
		`"schema": "tetra.ui.desktop-runtime.v1"`,
		`"schema": "tetra.ui.desktop-fake.v1"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateUIProductionRuntimeReport(path)
	if err == nil {
		t.Fatalf("expected invalid UI production runtime report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.ui.desktop-runtime.v1") {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func validUIProductionRuntimeReport() string {
	raw, err := json.MarshalIndent(validUIProductionReport(), "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

func validUIProductionReport() uiprod.Report {
	return uiprod.Report{
		Schema:    uiprod.SchemaV1,
		Status:    "pass",
		Target:    "linux-x64",
		Host:      "linux-x64",
		Runtime:   "desktop-ui-linux-x64",
		UISchema:  uiprod.UIBundleSchema,
		Source:    "tools/cmd/ui-production-runtime-smoke",
		Processes: uiProductionProcesses(),
		Contracts: uiProductionContracts(),
		Widgets:   uiProductionWidgets(),
		Events:    uiProductionEvents(),
		Cases:     uiProductionCases(),
		Audit:     uiProductionAudit(),
	}
}

func uiProductionProcesses() []uiprod.ProcessReport {
	return []uiprod.ProcessReport{
		uiProcess("tetra build", "build", "/tmp/tetra"),
		uiProcess("desktop UI app", "app", "/tmp/ui-desktop"),
		uiProcess("desktop UI runtime", "runtime", "tools/cmd/ui-production-runtime-smoke"),
		uiProcess(
			"native shell runtime integration",
			"runtime",
			"go run ./tools/cmd/native-ui-runtime-smoke",
		),
		uiProcess(
			"native runtime evidence validator",
			"runtime",
			"go run ./tools/cmd/validate-native-ui-runtime",
		),
		uiProcess("desktop UI widget stress", "stress", "/tmp/ui-widget-stress"),
	}
}

func uiProcess(name, kind, path string) uiprod.ProcessReport {
	return uiprod.ProcessReport{
		Name:     name,
		Kind:     kind,
		Path:     path,
		Ran:      true,
		Pass:     true,
		ExitCode: intPtr(0),
	}
}

func intPtr(value int) *int {
	return &value
}

func uiProductionContracts() []uiprod.ContractReport {
	return []uiprod.ContractReport{
		uiContract(
			"Linux-x64 desktop UI runtime",
			"desktop UI and sidecar-driven native runtime process evidence ran on linux-x64",
		),
		uiContract("window lifecycle", "window create, show, close, and teardown are covered"),
		uiContract("layout system", "layout measure/place and panel nesting cases ran"),
		uiContract(
			"buttons text input lists panels state binding",
			"button, text, input, focus/change, list, panel, and bound state widgets are present",
		),
		uiContract(
			"event loop",
			"focus, input, change, select, click, and timer events ran through the runtime",
		),
		uiContract(
			"async UI commands",
			"async command completion case runs through the UI runtime",
		),
		uiContract(
			"timers",
			"timer scheduled redraw case records a real timer tick event and timer_tick operation",
		),
		uiContract(
			"redraw update model",
			"redraw/update lifecycle case records dirty state to redraw",
		),
		uiContract(
			"error crash handling",
			"invalid widget, command failure recovery, and crash handling cases are required",
		),
		uiContract(
			"real dogfood applications",
			"dogfood application smoke case uses real Tetra UI source",
		),
	}
}

func uiContract(name, evidence string) uiprod.ContractReport {
	return uiprod.ContractReport{Name: name, Status: "pass", Evidence: evidence}
}

func uiProductionWidgets() []uiprod.WidgetReport {
	return []uiprod.WidgetReport{
		uiWidget("AppWindow", "window", "", "app.open", "", "", "", 0, 0, 640, 480),
		uiWidget("RootPanel", "panel", "AppWindow", "layout.root", "", "", "", 0, 0, 640, 480),
		uiWidget(
			"TitleText",
			"text",
			"RootPanel",
			"state.title",
			"",
			"",
			"Saved after timer",
			16,
			16,
			608,
			32,
		),
		uiWidget(
			"NameInput",
			"input",
			"RootPanel",
			"state.name",
			"input",
			"",
			"tetra-prod",
			16,
			64,
			608,
			32,
		),
		uiWidget(
			"ItemList",
			"list",
			"RootPanel",
			"state.items",
			"select",
			"",
			"item-1",
			16,
			112,
			608,
			240,
		),
		uiWidget(
			"SaveButton",
			"button",
			"RootPanel",
			"state.saved",
			"click",
			"saveAsync",
			"",
			16,
			368,
			200,
			44,
		),
	}
}

func uiWidget(
	id, kind, parent, binding, event, command, value string,
	x, y, width, height int,
) uiprod.WidgetReport {
	return uiprod.WidgetReport{
		ID:      id,
		Kind:    kind,
		Parent:  parent,
		Binding: binding,
		Event:   event,
		Command: command,
		Value:   value,
		Enabled: true,
		Visible: true,
		Bounds:  uiprod.Bounds{X: x, Y: y, Width: width, Height: height},
	}
}

func uiProductionEvents() []uiprod.EventReport {
	return []uiprod.EventReport{
		uiEvent(
			1,
			"NameInput",
			"focus",
			"focusName",
			map[string]string{"AppState.focused": "none"},
			map[string]string{"AppState.focused": "NameInput"},
			[]uiprod.OperationReport{
				uiOperation("focus", "widget.NameInput", "focused", "focused", "NameInput"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("TitleText", "Ready", "Editing name")},
		),
		uiEvent(
			2,
			"NameInput",
			"input",
			"setName",
			map[string]string{"AppState.name": "tetra"},
			map[string]string{"AppState.name": "tetra-lang"},
			[]uiprod.OperationReport{
				uiOperation("state_set", "state.name", "tetra-lang", "name", "tetra-lang"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("NameInput", "tetra", "tetra-lang")},
		),
		uiEvent(
			3,
			"NameInput",
			"change",
			"commitName",
			map[string]string{
				"AppState.name":    "tetra-lang",
				"AppState.changed": "false",
			},
			map[string]string{
				"AppState.name":    "tetra-prod",
				"AppState.changed": "true",
			},
			[]uiprod.OperationReport{
				uiOperation("change", "state.name", "tetra-prod", "name", "tetra-prod"),
				uiOperation("state_set", "state.changed", "true", "changed", "true"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("NameInput", "tetra-lang", "tetra-prod")},
		),
		uiEvent(
			4,
			"ItemList",
			"select",
			"selectItem",
			map[string]string{"AppState.selected": "item-1"},
			map[string]string{"AppState.selected": "item-2"},
			[]uiprod.OperationReport{
				uiOperation("state_set", "state.selected", "item-2", "selected", "item-2"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("ItemList", "item-1", "item-2")},
		),
		uiEvent(
			5,
			"SaveButton",
			"click",
			"saveAsync",
			map[string]string{"AppState.saved": "false"},
			map[string]string{"AppState.saved": "true"},
			[]uiprod.OperationReport{
				uiOperation("async_command", "command.saveAsync", "completed", "saved", "true"),
				uiOperation("redraw", "AppWindow", "scheduled", "dirty", "false"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("TitleText", "Editing name", "Saved")},
		),
		uiEvent(
			6,
			"AppWindow",
			"tick",
			"timerTick",
			map[string]string{"AppState.dirty": "true"},
			map[string]string{"AppState.dirty": "false"},
			[]uiprod.OperationReport{
				uiOperation("timer_tick", "timer.redraw", "fired", "dirty", "false"),
				uiOperation("redraw", "AppWindow", "completed", "dirty", "false"),
			},
			[]uiprod.WidgetUpdateReport{uiUpdate("TitleText", "Saved", "Saved after timer")},
		),
	}
}

func uiEvent(
	order int,
	widgetID, event, command string,
	beforeState map[string]string,
	afterState map[string]string,
	operations []uiprod.OperationReport,
	updates []uiprod.WidgetUpdateReport,
) uiprod.EventReport {
	return uiprod.EventReport{
		Order:         order,
		WidgetID:      widgetID,
		Event:         event,
		Command:       command,
		Pass:          true,
		BeforeState:   beforeState,
		AfterState:    afterState,
		Operations:    operations,
		WidgetUpdates: updates,
	}
}

func uiOperation(kind, target, value, field, stateValue string) uiprod.OperationReport {
	return uiprod.OperationReport{
		Kind:       kind,
		Target:     target,
		Value:      value,
		StateField: field,
		StateValue: stateValue,
	}
}

func uiUpdate(id, before, after string) uiprod.WidgetUpdateReport {
	return uiprod.WidgetUpdateReport{ID: id, Before: before, After: after}
}

func uiProductionCases() []uiprod.CaseReport {
	return []uiprod.CaseReport{
		uiCase("window lifecycle", "positive"),
		uiCase("layout measure and place", "positive"),
		uiCase("button command dispatch", "positive"),
		uiCase("text render", "positive"),
		uiCase("input focus traversal", "positive"),
		uiCase("input edit", "positive"),
		uiCase("input change commit", "positive"),
		uiCase("list selection", "positive"),
		uiCase("panel nesting", "positive"),
		uiCase("state binding update", "positive"),
		uiCase("event loop dispatch", "positive"),
		uiCase("async UI command completion", "positive"),
		uiCase("timer scheduled redraw", "positive"),
		uiCase("redraw update lifecycle", "positive"),
		uiCase("compiler UI bundle runtime load", "positive"),
		uiCase("native shell runtime integration", "positive"),
		uiCase("native runtime sidecar consistency", "positive"),
		uiErrorCase("invalid widget diagnostic", "unknown widget"),
		uiErrorCase("command failure recovery", "command failed"),
		uiErrorCase("crash error handling", "runtime panic recovered"),
		uiCase("dogfood application smoke", "positive"),
		uiCase("widget tree stress", "stress"),
	}
}

func uiCase(name, kind string) uiprod.CaseReport {
	return uiprod.CaseReport{Name: name, Kind: kind, Ran: true, Pass: true}
}

func uiErrorCase(name, expected string) uiprod.CaseReport {
	return uiprod.CaseReport{
		Name:          name,
		Kind:          "negative",
		Ran:           true,
		Pass:          true,
		ExpectedError: expected,
	}
}

func uiProductionAudit() []uiprod.AuditReport {
	return []uiprod.AuditReport{
		uiAudit(
			"Linux-x64 desktop UI runtime",
			"tools/cmd/ui-production-runtime-smoke; compiler/internal/backend/native_shell",
			"build, app, desktop runtime, native runtime, stress, and compiler-emitted "+
				"UI bundle load evidence ran on linux-x64",
		),
		uiAudit(
			"window lifecycle",
			"examples/ui/ui_desktop_runtime_smoke.tetra",
			"window create, show, close, and teardown cases are required",
		),
		uiAudit(
			"layout system",
			"compiler/internal/lower/lower_runtime_ui.go; docs/spec/ui/ui_v0.4.0.md",
			"layout measure/place and panel nesting cases are required",
		),
		uiAudit(
			"buttons/text/input/lists/panels widgets",
			"examples/ui/ui_desktop_runtime_smoke.tetra",
			"widget tree must include button, text, input, list, and panel widgets",
		),
		uiAudit(
			"state binding",
			"tools/validators/uiprod",
			"state binding update plus input focus/change widget update evidence are required",
		),
		uiAudit(
			"event loop and redraw/update model",
			"tools/cmd/ui-production-runtime-smoke",
			"focus, input, change, select, click, timer, and redraw/update lifecycle "+
				"cases are required",
		),
		uiAudit(
			"async commands and timers",
			"tools/cmd/ui-production-runtime-smoke",
			"async UI command completion, timer tick event evidence, and timer "+
				"scheduled redraw cases are required",
		),
		uiAudit(
			"error/crash handling",
			"tools/validators/uiprod",
			"invalid widget diagnostic, command failure recovery, and crash error "+
				"handling cases are required",
		),
		uiAudit(
			"real examples and dogfood applications",
			"examples/ui/ui_desktop_runtime_smoke.tetra; "+
				"examples/ui/ui_native_shell_smoke.tetra",
			"dogfood application smoke, compiler-emitted UI bundle/runtime trace load, "+
				"and native runtime integration cases are required",
		),
		uiAudit(
			"compiler-emitted UI bundle/native-shell trace load evidence",
			"examples/ui/ui_desktop_runtime_smoke.tetra; <output>.ui.json; "+
				"<output>.ui.shell.json",
			"UI production smoke loads compiler-emitted tetra.ui.v0.4.0 and "+
				"tetra.ui.native-shell.v1 artifacts before accepting runtime evidence",
		),
		uiAudit(
			"sidecar-driven native UI runtime integration",
			"tools/cmd/native-ui-runtime-smoke; tools/cmd/validate-native-ui-runtime; "+
				"native-ui-runtime-linux-x64.integration.json",
			"UI production smoke runs the sidecar-driven native UI runtime and validates "+
				"tetra.ui.native-runtime.v1 consistency before accepting the release gate",
		),
		uiAudit(
			"stable UI diagnostics",
			"tools/cmd/ui-production-runtime-smoke; tools/validators/uiprod",
			"negative UI cases require stable expected_error evidence for invalid widget "+
				"diagnostics, command failure recovery, and crash error handling",
		),
		uiAudit(
			"release-gate entrypoint rejecting runtime-less evidence",
			"scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh",
			"validator rejects metadata-only, runtime-less, fake, mock, placeholder, "+
				"docs-only, and build-only evidence and requires compiler UI bundle plus "+
				"native runtime integration evidence",
		),
	}
}

func uiAudit(requirement, artifact, evidence string) uiprod.AuditReport {
	return uiprod.AuditReport{
		Requirement: requirement,
		Artifact:    artifact,
		Evidence:    evidence,
		Result:      "pass",
	}
}
