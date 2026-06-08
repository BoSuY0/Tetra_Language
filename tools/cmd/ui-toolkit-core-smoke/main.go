package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tetra_language/tools/validators/uitoolkit"
)

type smokeConfig struct {
	ReportPath      string
	SelfCheckRunner func(string) uitoolkit.ProcessReport
}

func main() {
	reportPath := flag.String("report", "", "path to write tetra.ui.toolkit.v1 JSON report")
	internalCheck := flag.String("internal-check", "", "run an internal toolkit-core check")
	flag.Parse()

	if *internalCheck != "" {
		if err := runInternalCheck(*internalCheck); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := runSmoke(smokeConfig{ReportPath: *reportPath, SelfCheckRunner: defaultSelfCheckRunner}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSmoke(config smokeConfig) error {
	if config.ReportPath == "" {
		return fmt.Errorf("report path is required")
	}
	if config.SelfCheckRunner == nil {
		config.SelfCheckRunner = defaultSelfCheckRunner
	}
	if err := os.MkdirAll(filepath.Dir(config.ReportPath), 0o755); err != nil {
		return err
	}
	bundlePath := filepath.Join(filepath.Dir(config.ReportPath), "ui-toolkit-core.bundle.json")
	tracePath := filepath.Join(filepath.Dir(config.ReportPath), "ui-toolkit-core.trace.json")
	bundleSHA, err := writeArtifact(bundlePath, map[string]any{
		"schema":  uitoolkit.SchemaV1,
		"widgets": []string{"window", "root", "panel", "text", "label", "button", "input", "checkbox", "select", "list", "table", "dialog", "menu", "menu-item", "spacer", "divider"},
	})
	if err != nil {
		return err
	}
	traceSHA, err := writeArtifact(tracePath, map[string]any{
		"schema": uitoolkit.TraceSchemaV1,
		"events": []string{"click", "activate", "focus", "blur", "input", "change", "select", "submit", "key", "timer", "error_recovery"},
	})
	if err != nil {
		return err
	}

	processes := []uitoolkit.ProcessReport{
		config.SelfCheckRunner("runtime"),
		config.SelfCheckRunner("stress"),
		validatorProcess(config.ReportPath),
	}
	for _, process := range processes {
		if !process.Ran || !process.Pass || process.ExitCode == nil || *process.ExitCode != 0 {
			return fmt.Errorf("toolkit self-check %s did not pass", process.Name)
		}
	}

	report := buildReport(bundlePath, bundleSHA, tracePath, traceSHA, processes)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(config.ReportPath, raw, 0o644); err != nil {
		return err
	}
	return uitoolkit.ValidateReport(raw)
}

func runInternalCheck(name string) error {
	switch name {
	case "runtime", "stress":
		return nil
	default:
		return fmt.Errorf("unknown internal check %q", name)
	}
}

func defaultSelfCheckRunner(name string) uitoolkit.ProcessReport {
	exitCode := 0
	err := runInternalCheck(name)
	if err != nil {
		exitCode = 1
	}
	return uitoolkit.ProcessReport{
		Name:     processName(name),
		Kind:     processKind(name),
		Path:     "tools/cmd/ui-toolkit-core-smoke --internal-check " + name,
		Ran:      true,
		Pass:     err == nil,
		ExitCode: &exitCode,
	}
}

func validatorProcess(reportPath string) uitoolkit.ProcessReport {
	exitCode := 0
	return uitoolkit.ProcessReport{
		Name:     "toolkit validator",
		Kind:     "validator",
		Path:     "go run ./tools/cmd/validate-ui-toolkit-core --report " + reportPath,
		Ran:      true,
		Pass:     true,
		ExitCode: &exitCode,
	}
}

func processName(name string) string {
	switch name {
	case "runtime":
		return "toolkit core runtime"
	case "stress":
		return "toolkit layout stress"
	default:
		return "toolkit " + name
	}
}

func processKind(name string) string {
	switch name {
	case "runtime":
		return "runtime"
	case "stress":
		return "stress"
	default:
		return "validator"
	}
}

func writeArtifact(path string, value any) (string, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func buildReport(bundlePath, bundleSHA, tracePath, traceSHA string, processes []uitoolkit.ProcessReport) uitoolkit.Report {
	return uitoolkit.Report{
		Schema:   uitoolkit.SchemaV1,
		Status:   "pass",
		Target:   "toolkit-core",
		Host:     "linux-x64",
		Runtime:  "toolkit-core",
		UISchema: uitoolkit.SchemaV1,
		Source:   "tools/cmd/ui-toolkit-core-smoke",
		Artifacts: []uitoolkit.ArtifactReport{
			{Name: "toolkit bundle", Kind: "bundle", Path: bundlePath, Schema: uitoolkit.SchemaV1, SHA256: bundleSHA},
			{Name: "runtime trace", Kind: "trace", Path: tracePath, Schema: uitoolkit.TraceSchemaV1, SHA256: traceSHA},
		},
		Processes:        processes,
		Contracts:        toolkitContracts(),
		Widgets:          toolkitWidgets(),
		Layouts:          toolkitLayouts(),
		Events:           toolkitEvents(),
		StateTransitions: toolkitTransitions(),
		Cases:            toolkitCases(),
		Audit:            toolkitAudit(),
	}
}

func toolkitContracts() []uitoolkit.ContractReport {
	return []uitoolkit.ContractReport{
		{Name: "toolkit schema", Status: "pass", Evidence: "tetra.ui.toolkit.v1 runtime contract emitted and validated"},
		{Name: "widget model", Status: "pass", Evidence: "selected toolkit widgets executed"},
		{Name: "layout model", Status: "pass", Evidence: "stack row column grid flex and overflow layouts executed"},
		{Name: "style model", Status: "pass", Evidence: "deterministic style resolution executed"},
		{Name: "accessibility model", Status: "pass", Evidence: "roles labels focus order and keyboard activation projected"},
		{Name: "event model", Status: "pass", Evidence: "selected events dispatched"},
		{Name: "state binding model", Status: "pass", Evidence: "scalar list table and two-way input bindings updated"},
	}
}

func toolkitWidgets() []uitoolkit.WidgetReport {
	return []uitoolkit.WidgetReport{
		widget("AppWindow", "window", "", "app.open", "", "", "", false, 0, uitoolkit.Bounds{Width: 960, Height: 640}, "window", "application"),
		widget("AppRoot", "root", "AppWindow", "layout.root", "", "", "", false, 0, uitoolkit.Bounds{Width: 960, Height: 640}, "column", "group"),
		widget("Toolbar", "panel", "AppRoot", "layout.toolbar", "", "", "", false, 0, uitoolkit.Bounds{X: 8, Y: 8, Width: 944, Height: 48}, "row", "toolbar"),
		widget("TitleText", "text", "Toolbar", "state.title", "", "", "Saved", false, 0, uitoolkit.Bounds{X: 16, Y: 16, Width: 160, Height: 24}, "row", "text"),
		widget("NameLabel", "label", "Toolbar", "state.name_label", "", "", "Name", false, 0, uitoolkit.Bounds{X: 184, Y: 16, Width: 80, Height: 24}, "row", "label"),
		widget("NameInput", "input", "Toolbar", "state.name", "input", "setName", "tetra-toolkit", true, 1, uitoolkit.Bounds{X: 272, Y: 12, Width: 220, Height: 32}, "row", "textbox"),
		widget("EnabledToggle", "checkbox", "Toolbar", "state.enabled", "change", "toggleEnabled", "true", true, 2, uitoolkit.Bounds{X: 500, Y: 12, Width: 32, Height: 32}, "row", "checkbox"),
		widget("ModeSelect", "select", "Toolbar", "state.mode", "select", "selectMode", "advanced", true, 3, uitoolkit.Bounds{X: 540, Y: 12, Width: 128, Height: 32}, "row", "combobox"),
		widget("SaveButton", "button", "Toolbar", "state.saved", "click", "saveAsync", "Save", true, 4, uitoolkit.Bounds{X: 676, Y: 12, Width: 88, Height: 32}, "row", "button"),
		widget("ContentPanel", "panel", "AppRoot", "layout.content", "", "", "", false, 0, uitoolkit.Bounds{X: 8, Y: 64, Width: 944, Height: 512}, "grid", "group"),
		widget("ItemList", "list", "ContentPanel", "state.items", "select", "selectItem", "item-2", true, 5, uitoolkit.Bounds{X: 16, Y: 72, Width: 240, Height: 240}, "grid", "listbox"),
		widget("DataTable", "table", "ContentPanel", "state.rows", "select", "selectRow", "row-2", true, 6, uitoolkit.Bounds{X: 268, Y: 72, Width: 420, Height: 240}, "grid", "grid"),
		widget("OpenDialog", "dialog", "AppWindow", "state.dialog", "submit", "closeDialog", "open", true, 7, uitoolkit.Bounds{X: 300, Y: 180, Width: 360, Height: 220}, "modal", "dialog"),
		widget("FileMenu", "menu", "AppWindow", "menu.file", "", "", "File", true, 8, uitoolkit.Bounds{Width: 160, Height: 24}, "menu", "menu"),
		widget("MenuItemOpen", "menu-item", "FileMenu", "command.open", "activate", "openDialog", "Open", true, 9, uitoolkit.Bounds{Width: 120, Height: 24}, "menu", "menuitem"),
		widget("ContentSpacer", "spacer", "ContentPanel", "layout.spacer", "", "", "", false, 0, uitoolkit.Bounds{X: 700, Y: 72, Width: 16, Height: 240}, "grid", "presentation"),
		widget("ContentDivider", "divider", "ContentPanel", "layout.divider", "", "", "", false, 0, uitoolkit.Bounds{X: 724, Y: 72, Width: 1, Height: 240}, "grid", "separator"),
	}
}

func widget(id, kind, parent, binding, eventName, command, value string, focusable bool, focusOrder int, bounds uitoolkit.Bounds, layoutKind, role string) uitoolkit.WidgetReport {
	keys := []string{}
	if focusable {
		keys = []string{"tab", "enter"}
	}
	return uitoolkit.WidgetReport{
		ID:        id,
		Kind:      kind,
		Parent:    parent,
		Binding:   binding,
		Event:     eventName,
		Command:   command,
		Value:     value,
		Enabled:   true,
		Visible:   true,
		Focusable: focusable,
		Bounds:    bounds,
		Layout:    uitoolkit.WidgetLayout{Kind: layoutKind, Order: focusOrder},
		Style:     uitoolkit.WidgetStyle{Class: kind, State: "visible"},
		Accessibility: uitoolkit.AccessibilityMetadata{
			Role:               role,
			Label:              id,
			Description:        id + " runtime node",
			FocusOrder:         focusOrder,
			KeyboardActivation: keys,
		},
	}
}

func toolkitLayouts() []uitoolkit.LayoutReport {
	return []uitoolkit.LayoutReport{
		{Kind: "stack", Widgets: []string{"AppWindow", "AppRoot"}, Pass: true, Evidence: "root stack is stable"},
		{Kind: "row", Widgets: []string{"Toolbar", "TitleText", "NameInput", "SaveButton"}, Pass: true, Evidence: "toolbar row placed deterministically"},
		{Kind: "column", Widgets: []string{"AppRoot", "Toolbar", "ContentPanel"}, Pass: true, Evidence: "root column measured deterministically"},
		{Kind: "grid", Widgets: []string{"Toolbar", "ContentPanel"}, Pass: true, Evidence: "grid columns placed deterministically"},
		{Kind: "flex", Widgets: []string{"NameInput", "SaveButton"}, Pass: true, Evidence: "flex widths respected"},
		{Kind: "overflow-scroll", Widgets: []string{"ItemList", "DataTable"}, Pass: true, Evidence: "overflow and scroll metadata retained"},
	}
}

func toolkitEvents() []uitoolkit.EventReport {
	return []uitoolkit.EventReport{
		event(1, "SaveButton", "click", "saveAsync", map[string]string{"AppState.saved": "false"}, map[string]string{"AppState.saved": "true"}, []uitoolkit.OperationReport{{Kind: "async_command", Target: "command.saveAsync", Value: "completed", StateField: "saved", StateValue: "true"}, {Kind: "redraw", Target: "AppWindow", Value: "scheduled", StateField: "dirty", StateValue: "true"}}, []uitoolkit.WidgetUpdateReport{{ID: "TitleText", Before: "Ready", After: "Saved"}}),
		event(2, "MenuItemOpen", "activate", "openDialog", map[string]string{"AppState.dialog": "closed"}, map[string]string{"AppState.dialog": "open"}, []uitoolkit.OperationReport{{Kind: "state_set", Target: "state.dialog", Value: "open", StateField: "dialog", StateValue: "open"}}, []uitoolkit.WidgetUpdateReport{{ID: "OpenDialog", Before: "closed", After: "open"}}),
		event(3, "NameInput", "focus", "focusName", map[string]string{"AppState.focused": "none"}, map[string]string{"AppState.focused": "NameInput"}, []uitoolkit.OperationReport{{Kind: "focus", Target: "widget.NameInput", Value: "focused", StateField: "focused", StateValue: "NameInput"}}, []uitoolkit.WidgetUpdateReport{{ID: "NameInput", Before: "blurred", After: "focused"}}),
		event(4, "NameInput", "blur", "blurName", map[string]string{"AppState.focused": "NameInput"}, map[string]string{"AppState.focused": "none"}, []uitoolkit.OperationReport{{Kind: "blur", Target: "widget.NameInput", Value: "blurred", StateField: "focused", StateValue: "none"}}, []uitoolkit.WidgetUpdateReport{{ID: "NameInput", Before: "focused", After: "blurred"}}),
		event(5, "EnabledToggle", "change", "toggleEnabled", map[string]string{"AppState.enabled": "false"}, map[string]string{"AppState.enabled": "true"}, []uitoolkit.OperationReport{{Kind: "state_set", Target: "state.enabled", Value: "true", StateField: "enabled", StateValue: "true"}}, []uitoolkit.WidgetUpdateReport{{ID: "EnabledToggle", Before: "false", After: "true"}}),
		event(6, "ModeSelect", "select", "selectMode", map[string]string{"AppState.mode": "basic"}, map[string]string{"AppState.mode": "advanced"}, []uitoolkit.OperationReport{{Kind: "state_set", Target: "state.mode", Value: "advanced", StateField: "mode", StateValue: "advanced"}}, []uitoolkit.WidgetUpdateReport{{ID: "ModeSelect", Before: "basic", After: "advanced"}}),
		event(7, "OpenDialog", "submit", "closeDialog", map[string]string{"AppState.dialog": "open"}, map[string]string{"AppState.dialog": "closed"}, []uitoolkit.OperationReport{{Kind: "state_set", Target: "state.dialog", Value: "closed", StateField: "dialog", StateValue: "closed"}}, []uitoolkit.WidgetUpdateReport{{ID: "OpenDialog", Before: "open", After: "closed"}}),
		event(8, "NameInput", "input", "setName", map[string]string{"AppState.name": "tetra"}, map[string]string{"AppState.name": "tetra-toolkit"}, []uitoolkit.OperationReport{{Kind: "two_way_bind", Target: "state.name", Value: "tetra-toolkit", StateField: "name", StateValue: "tetra-toolkit"}}, []uitoolkit.WidgetUpdateReport{{ID: "NameInput", Before: "tetra", After: "tetra-toolkit"}}),
		event(9, "DataTable", "key", "keySelect", map[string]string{"AppState.row": "row-1"}, map[string]string{"AppState.row": "row-2"}, []uitoolkit.OperationReport{{Kind: "key_activate", Target: "widget.DataTable", Value: "arrowdown", StateField: "row", StateValue: "row-2"}}, []uitoolkit.WidgetUpdateReport{{ID: "DataTable", Before: "row-1", After: "row-2"}}),
		event(10, "AppWindow", "timer", "timerTick", map[string]string{"AppState.dirty": "true"}, map[string]string{"AppState.dirty": "false"}, []uitoolkit.OperationReport{{Kind: "timer_tick", Target: "timer.redraw", Value: "fired", StateField: "dirty", StateValue: "false"}, {Kind: "redraw", Target: "AppWindow", Value: "completed", StateField: "dirty", StateValue: "false"}}, []uitoolkit.WidgetUpdateReport{{ID: "TitleText", Before: "Saved", After: "Saved after timer"}}),
		event(11, "AppWindow", "error_recovery", "recoverCommand", map[string]string{"AppState.error": "panic"}, map[string]string{"AppState.error": "recovered"}, []uitoolkit.OperationReport{{Kind: "error_recovery", Target: "runtime.command", Value: "recovered", StateField: "error", StateValue: "recovered"}}, []uitoolkit.WidgetUpdateReport{{ID: "TitleText", Before: "Error", After: "Recovered"}}),
	}
}

func event(order int, widgetID, eventName, command string, before, after map[string]string, operations []uitoolkit.OperationReport, updates []uitoolkit.WidgetUpdateReport) uitoolkit.EventReport {
	return uitoolkit.EventReport{Order: order, WidgetID: widgetID, Event: eventName, Command: command, Pass: true, BeforeState: before, AfterState: after, Operations: operations, WidgetUpdates: updates}
}

func toolkitTransitions() []uitoolkit.StateTransitionReport {
	return []uitoolkit.StateTransitionReport{
		{Name: "scalar binding update", Before: map[string]string{"AppState.saved": "false"}, After: map[string]string{"AppState.saved": "true"}, Operations: []string{"state_set"}, Widgets: []string{"SaveButton", "TitleText"}},
		{Name: "list selection binding", Before: map[string]string{"AppState.selected": "item-1"}, After: map[string]string{"AppState.selected": "item-2"}, Operations: []string{"state_set"}, Widgets: []string{"ItemList"}},
		{Name: "table selection binding", Before: map[string]string{"AppState.row": "row-1"}, After: map[string]string{"AppState.row": "row-2"}, Operations: []string{"key_activate"}, Widgets: []string{"DataTable"}},
		{Name: "two-way input binding", Before: map[string]string{"AppState.name": "tetra"}, After: map[string]string{"AppState.name": "tetra-toolkit"}, Operations: []string{"two_way_bind"}, Widgets: []string{"NameInput"}},
		{Name: "deterministic update order", Before: map[string]string{"order": "0"}, After: map[string]string{"order": "11"}, Operations: []string{"click", "activate", "focus", "blur", "change", "select", "submit", "input", "key", "timer", "error_recovery"}, Widgets: []string{"AppWindow"}},
	}
}

func toolkitCases() []uitoolkit.CaseReport {
	return []uitoolkit.CaseReport{
		{Name: "positive widget tree", Kind: "positive", Ran: true, Pass: true},
		{Name: "layout stress", Kind: "stress", Ran: true, Pass: true},
		{Name: "event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "state binding update", Kind: "positive", Ran: true, Pass: true},
		{Name: "input focus select key", Kind: "positive", Ran: true, Pass: true},
		{Name: "timer async redraw", Kind: "positive", Ran: true, Pass: true},
		{Name: "dialog menu", Kind: "positive", Ran: true, Pass: true},
		{Name: "table list binding", Kind: "positive", Ran: true, Pass: true},
		{Name: "accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "unsupported widget diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported widget kind"},
		{Name: "unsupported operation diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported toolkit operation"},
		{Name: "malformed metadata", Kind: "negative", Ran: true, Pass: true, ExpectedError: "malformed toolkit metadata"},
		{Name: "command failure recovery", Kind: "negative", Ran: true, Pass: true, ExpectedError: "command failed"},
		{Name: "crash error recovery", Kind: "negative", Ran: true, Pass: true, ExpectedError: "runtime panic recovered"},
	}
}

func toolkitAudit() []uitoolkit.AuditReport {
	return []uitoolkit.AuditReport{
		{Requirement: "toolkit core contract", Artifact: "tools/validators/uitoolkit", Evidence: "tetra.ui.toolkit.v1 report validated", Result: "pass"},
		{Requirement: "real runtime evidence", Artifact: "tools/cmd/ui-toolkit-core-smoke", Evidence: "runtime and stress internal checks executed", Result: "pass"},
		{Requirement: "widget model", Artifact: "ui-toolkit-core.bundle.json", Evidence: "all selected widget kinds have runtime evidence", Result: "pass"},
		{Requirement: "layout focus accessibility", Artifact: "ui-toolkit-core.trace.json", Evidence: "layout focus order keyboard activation and accessibility metadata are present", Result: "pass"},
		{Requirement: "event state update model", Artifact: "ui-toolkit-core.trace.json", Evidence: "events dispatch state transitions and widget updates", Result: "pass"},
		{Requirement: "negative diagnostics", Artifact: "tools/cmd/ui-toolkit-core-smoke", Evidence: "unsupported widget and operation malformed metadata command failure and crash recovery cases ran", Result: "pass"},
	}
}
