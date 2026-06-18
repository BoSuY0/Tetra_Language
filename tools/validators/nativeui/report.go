package nativeui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaV1 = "tetra.ui.native-runtime.v1"
const UIBundleSchema = "tetra.ui.v0.4.0"

type Report struct {
	Schema    string          `json:"schema"`
	Status    string          `json:"status"`
	Target    string          `json:"target"`
	Host      string          `json:"host"`
	Runtime   string          `json:"runtime"`
	UISchema  string          `json:"ui_schema"`
	Source    string          `json:"source"`
	Processes []ProcessReport `json:"processes"`
	Widgets   []WidgetReport  `json:"widgets"`
	Events    []EventReport   `json:"events"`
	Cases     []CaseReport    `json:"cases"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type WidgetReport struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Parent  string `json:"parent"`
	Binding string `json:"binding,omitempty"`
	Event   string `json:"event,omitempty"`
	Command string `json:"command,omitempty"`
	Value   string `json:"value,omitempty"`
	Enabled bool   `json:"enabled"`
	Visible bool   `json:"visible"`
	Bounds  Bounds `json:"bounds"`
}

type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type EventReport struct {
	Order         int                  `json:"order"`
	WidgetID      string               `json:"widget_id"`
	Event         string               `json:"event"`
	Command       string               `json:"command"`
	Pass          bool                 `json:"pass"`
	BeforeState   map[string]string    `json:"before_state"`
	AfterState    map[string]string    `json:"after_state"`
	Operations    []OperationReport    `json:"operations"`
	WidgetUpdates []WidgetUpdateReport `json:"widget_updates"`
}

type OperationReport struct {
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	Value      string `json:"value"`
	StateField string `json:"state_field"`
	StateValue string `json:"state_value"`
}

type WidgetUpdateReport struct {
	ID     string `json:"id"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type CaseReport struct {
	Name          string `json:"name"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedError string `json:"expected_error,omitempty"`
	Error         string `json:"error,omitempty"`
}

func ValidateReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != SchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, SchemaV1)
	}

	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectPaperEvidence(raw)...)
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "native-ui-linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf("runtime is %q, want native-ui-linux-x64", report.Runtime),
		)
	}
	if report.UISchema != UIBundleSchema {
		issues = append(
			issues,
			fmt.Sprintf("ui_schema is %q, want %s", report.UISchema, UIBundleSchema),
		)
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Processes)...)
	widgetIndex, actionIDs, widgetIssues := validateWidgets(report.Widgets)
	issues = append(issues, widgetIssues...)
	issues = append(issues, validateEvents(report.Events, widgetIndex, actionIDs)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{
		"metadata-only",
		"web-only",
		"sidecar-only",
		"docs-only",
		" fake",
		"fake/",
		"\"fake\"",
		" mock",
		"mock/",
		"\"mock\"",
		"placeholder",
	}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(
				issues,
				fmt.Sprintf(
					"report contains forbidden non-runtime evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 2 {
		issues = append(
			issues,
			fmt.Sprintf(
				"process evidence has %d entries, want app and runtime processes",
				len(processes),
			),
		)
	}
	seenApp := false
	seenRuntime := false
	seenBuild := false
	names := map[string]bool{}
	for _, p := range processes {
		if strings.TrimSpace(p.Name) == "" {
			issues = append(issues, "process name is required")
		} else if names[p.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", p.Name))
		}
		names[p.Name] = true
		switch p.Kind {
		case "build":
			seenBuild = true
		case "app":
			seenApp = true
		case "runtime":
			seenRuntime = true
		default:
			issues = append(
				issues,
				fmt.Sprintf("process %s kind is %q, want build, app, or runtime", p.Name, p.Kind),
			)
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", p.Name))
		}
		if !p.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", p.Name))
		}
		if !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", p.Name))
		}
		if p.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", p.Name))
		} else if *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want 0", p.Name, *p.ExitCode))
		}
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable native app process")
	}
	if !seenRuntime {
		issues = append(issues, "process evidence missing native runtime process")
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	return issues
}

func validateWidgets(widgets []WidgetReport) (map[string]WidgetReport, map[string]bool, []string) {
	var issues []string
	if len(widgets) < 3 {
		issues = append(
			issues,
			fmt.Sprintf(
				"widget evidence has %d entries, want view, state binding, and action widgets",
				len(widgets),
			),
		)
	}
	index := map[string]WidgetReport{}
	actions := map[string]bool{}
	seenRoot := false
	seenBinding := false
	for _, w := range widgets {
		if strings.TrimSpace(w.ID) == "" {
			issues = append(issues, "widget id is required")
			continue
		}
		if _, exists := index[w.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate widget %s", w.ID))
		}
		index[w.ID] = w
		if strings.TrimSpace(w.Kind) == "" {
			issues = append(issues, fmt.Sprintf("widget %s kind is required", w.ID))
		}
		if !w.Enabled {
			issues = append(
				issues,
				fmt.Sprintf(
					"widget %s must record enabled=true for passing runtime evidence",
					w.ID,
				),
			)
		}
		if !w.Visible {
			issues = append(
				issues,
				fmt.Sprintf(
					"widget %s must record visible=true for passing runtime evidence",
					w.ID,
				),
			)
		}
		if w.Bounds.Width <= 0 || w.Bounds.Height <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("widget %s bounds must have positive width and height", w.ID),
			)
		}
		switch w.Kind {
		case "view":
			if strings.TrimSpace(w.Parent) == "" {
				seenRoot = true
			}
		case "value", "text", "input":
			seenBinding = true
			if strings.TrimSpace(w.Binding) == "" {
				issues = append(issues, fmt.Sprintf("widget %s binding is required", w.ID))
			}
		case "action":
			actions[w.ID] = true
			if strings.TrimSpace(w.Event) == "" {
				issues = append(issues, fmt.Sprintf("action widget %s event is required", w.ID))
			}
			if strings.TrimSpace(w.Command) == "" {
				issues = append(issues, fmt.Sprintf("action widget %s command is required", w.ID))
			}
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"widget %s kind is %q, want view, value, text, input, or action",
					w.ID,
					w.Kind,
				),
			)
		}
	}
	for _, w := range widgets {
		if w.ID == "" || w.Parent == "" {
			continue
		}
		if _, ok := index[w.Parent]; !ok {
			issues = append(issues, fmt.Sprintf("widget %s parent %s is missing", w.ID, w.Parent))
		}
	}
	if !seenRoot {
		issues = append(issues, "widget evidence missing root view widget")
	}
	if !seenBinding {
		issues = append(issues, "widget evidence missing state-bound value/text/input widget")
	}
	if len(actions) == 0 {
		issues = append(issues, "widget evidence missing action widget")
	}
	return index, actions, issues
}

func validateEvents(
	events []EventReport,
	widgets map[string]WidgetReport,
	actionIDs map[string]bool,
) []string {
	var issues []string
	if len(events) < 2 {
		issues = append(
			issues,
			fmt.Sprintf(
				"event state transition evidence has %d events, want at least 2",
				len(events),
			),
		)
	}
	lastOrder := 0
	for _, event := range events {
		if event.Order <= lastOrder {
			issues = append(
				issues,
				fmt.Sprintf(
					"event order %d is not strictly greater than previous order %d",
					event.Order,
					lastOrder,
				),
			)
		}
		lastOrder = event.Order
		widget, exists := widgets[event.WidgetID]
		if strings.TrimSpace(event.WidgetID) == "" {
			issues = append(issues, "event widget_id is required")
		} else if !exists {
			issues = append(issues, fmt.Sprintf("event widget_id %s is not in widget tree", event.WidgetID))
		} else if !actionIDs[event.WidgetID] {
			issues = append(
				issues,
				fmt.Sprintf("event widget_id %s must reference an action widget", event.WidgetID),
			)
		}
		switch event.Event {
		case "click", "activate", "input", "change", "focus":
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"event %d kind is %q, want click, activate, input, change, or focus",
					event.Order,
					event.Event,
				),
			)
		}
		if strings.TrimSpace(event.Command) == "" {
			issues = append(issues, fmt.Sprintf("event %d command is required", event.Order))
		} else if exists && widget.Command != "" && event.Command != widget.Command {
			issues = append(
				issues,
				fmt.Sprintf("event %d command is %q, want widget command %q", event.Order, event.Command, widget.Command),
			)
		}
		if !event.Pass {
			issues = append(issues, fmt.Sprintf("event %d did not pass", event.Order))
		}
		if len(event.BeforeState) == 0 || len(event.AfterState) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("event %d must include before_state and after_state", event.Order),
			)
		} else if !stateChanged(event.BeforeState, event.AfterState) {
			issues = append(issues, fmt.Sprintf("event %d has no observable state change", event.Order))
		}
		issues = append(issues, validateOperations(event.Order, event.Operations)...)
		issues = append(issues, validateWidgetUpdates(event.Order, event.WidgetUpdates, widgets)...)
	}
	return issues
}

func validateOperations(order int, operations []OperationReport) []string {
	var issues []string
	if len(operations) == 0 {
		issues = append(issues, fmt.Sprintf("event %d operations are required", order))
	}
	for i, op := range operations {
		label := fmt.Sprintf("event %d operation %d", order, i+1)
		if strings.TrimSpace(op.Kind) == "" {
			issues = append(issues, label+" kind is required")
		}
		if strings.TrimSpace(op.Target) == "" {
			issues = append(issues, label+" target is required")
		}
		if strings.TrimSpace(op.StateField) == "" {
			issues = append(issues, label+" state_field is required")
		}
		if strings.TrimSpace(op.StateValue) == "" {
			issues = append(issues, label+" state_value is required")
		}
	}
	return issues
}

func validateWidgetUpdates(
	order int,
	updates []WidgetUpdateReport,
	widgets map[string]WidgetReport,
) []string {
	var issues []string
	if len(updates) == 0 {
		issues = append(issues, fmt.Sprintf("event %d widget_updates are required", order))
	}
	for i, update := range updates {
		label := fmt.Sprintf("event %d widget_update %d", order, i+1)
		if strings.TrimSpace(update.ID) == "" {
			issues = append(issues, label+" id is required")
		} else if _, ok := widgets[update.ID]; !ok {
			issues = append(issues, fmt.Sprintf("%s id %s is not in widget tree", label, update.ID))
		}
		if update.Before == update.After {
			issues = append(issues, label+" must record a value change")
		}
	}
	return issues
}

func stateChanged(before, after map[string]string) bool {
	for key, beforeValue := range before {
		if afterValue, ok := after[key]; ok && afterValue != beforeValue {
			return true
		}
	}
	for key := range after {
		if _, ok := before[key]; !ok {
			return true
		}
	}
	return false
}

func validateCases(cases []CaseReport) []string {
	var issues []string
	required := map[string]bool{
		"load widget tree":                 false,
		"dispatch click command":           false,
		"propagate state update":           false,
		"dispatch multiple ordered events": false,
		"reject invalid widget id":         false,
		"reject malformed metadata":        false,
		"reject unsupported event kind":    false,
		"reject command failure":           false,
		"close runtime":                    false,
	}
	negative := map[string]bool{
		"reject invalid widget id":      true,
		"reject malformed metadata":     true,
		"reject unsupported event kind": true,
		"reject command failure":        true,
	}
	for _, c := range cases {
		if strings.TrimSpace(c.Name) == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", c.Name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", c.Name))
		}
		if negative[c.Name] && strings.TrimSpace(c.ExpectedError) == "" {
			issues = append(issues, fmt.Sprintf("case %s missing expected_error", c.Name))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("case %s has error text", c.Name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required case %s", name))
		}
	}
	return issues
}

func decodeSchema(raw []byte) (string, error) {
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", err
	}
	return envelope.Schema, nil
}

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
