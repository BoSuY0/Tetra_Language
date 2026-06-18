package uiprod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaV1 = "tetra.ui.desktop-runtime.v1"
const UIBundleSchema = "tetra.ui.v0.4.0"

type Report struct {
	Schema    string           `json:"schema"`
	Status    string           `json:"status"`
	Target    string           `json:"target"`
	Host      string           `json:"host"`
	Runtime   string           `json:"runtime"`
	UISchema  string           `json:"ui_schema"`
	Source    string           `json:"source"`
	Processes []ProcessReport  `json:"processes"`
	Contracts []ContractReport `json:"contracts"`
	Widgets   []WidgetReport   `json:"widgets"`
	Events    []EventReport    `json:"events"`
	Cases     []CaseReport     `json:"cases"`
	Audit     []AuditReport    `json:"audit"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type ContractReport struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
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
	Kind          string `json:"kind"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedError string `json:"expected_error,omitempty"`
	Error         string `json:"error,omitempty"`
}

type AuditReport struct {
	Requirement string `json:"requirement"`
	Artifact    string `json:"artifact"`
	Evidence    string `json:"evidence"`
	Result      string `json:"result"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectPaperEvidence(report)...)
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "desktop-ui-linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf("runtime is %q, want desktop-ui-linux-x64", report.Runtime),
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
	issues = append(issues, validateContracts(report.Contracts)...)
	widgetIndex, widgetIssues := validateWidgets(report.Widgets)
	issues = append(issues, widgetIssues...)
	issues = append(issues, validateEvents(report.Events, widgetIndex)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateAudit(report.Audit)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(report Report) []string {
	text := strings.Join(reportEvidenceFields(report), "\n")
	lower := strings.ToLower(text)
	forbidden := []string{
		"metadata-only",
		"runtime-less",
		"build-only",
		"docs-only",
		"web-only",
		"sidecar-only",
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
					"report contains forbidden non-production UI evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	return issues
}

func reportEvidenceFields(report Report) []string {
	fields := []string{report.Source}
	for _, p := range report.Processes {
		fields = append(fields, p.Name, p.Kind, p.Path)
	}
	for _, c := range report.Contracts {
		fields = append(fields, c.Name, c.Evidence)
	}
	for _, w := range report.Widgets {
		fields = append(fields, w.ID, w.Kind, w.Parent, w.Binding, w.Event, w.Command, w.Value)
	}
	for _, event := range report.Events {
		fields = append(fields, event.WidgetID, event.Event, event.Command)
		for _, op := range event.Operations {
			fields = append(fields, op.Kind, op.Target, op.Value, op.StateField, op.StateValue)
		}
		for _, update := range event.WidgetUpdates {
			fields = append(fields, update.ID, update.Before, update.After)
		}
	}
	for _, c := range report.Cases {
		fields = append(fields, c.Name, c.Kind, c.ExpectedError, c.Error)
	}
	return fields
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 4 {
		issues = append(
			issues,
			fmt.Sprintf(
				"process evidence has %d entries, want build, app, runtime, and stress processes",
				len(processes),
			),
		)
	}
	requiredKinds := map[string]bool{
		"build":   false,
		"app":     false,
		"runtime": false,
		"stress":  false,
	}
	requiredProcesses := map[string]bool{
		"native shell runtime integration":  false,
		"native runtime evidence validator": false,
	}
	names := map[string]bool{}
	for _, p := range processes {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			issues = append(issues, "process name is required")
		} else if names[name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", name))
		}
		names[name] = true
		if _, ok := requiredProcesses[name]; ok {
			requiredProcesses[name] = true
		}
		if _, ok := requiredKinds[p.Kind]; ok {
			requiredKinds[p.Kind] = true
		} else {
			issues = append(
				issues,
				fmt.Sprintf("process %s kind is %q, want build, app, runtime, or stress", name, p.Kind),
			)
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", name))
		}
		if !p.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", name))
		}
		if !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", name))
		}
		if p.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", name))
		} else if *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want 0", name, *p.ExitCode))
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("process evidence missing %s process", kind))
		}
	}
	for name, seen := range requiredProcesses {
		if !seen {
			issues = append(issues, fmt.Sprintf("process evidence missing %s process", name))
		}
	}
	return issues
}

func validateContracts(contracts []ContractReport) []string {
	required := map[string]bool{
		"Linux-x64 desktop UI runtime":                  false,
		"window lifecycle":                              false,
		"layout system":                                 false,
		"buttons text input lists panels state binding": false,
		"event loop":                                    false,
		"async UI commands":                             false,
		"timers":                                        false,
		"redraw update model":                           false,
		"error crash handling":                          false,
		"real dogfood applications":                     false,
	}
	var issues []string
	for _, c := range contracts {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "contract name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Status != "pass" {
			issues = append(
				issues,
				fmt.Sprintf("contract %s status is %q, want pass", name, c.Status),
			)
		}
		if strings.TrimSpace(c.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("contract %s evidence is required", name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required UI contract %q", name))
		}
	}
	return issues
}

func validateWidgets(widgets []WidgetReport) (map[string]WidgetReport, []string) {
	var issues []string
	if len(widgets) < 6 {
		issues = append(
			issues,
			fmt.Sprintf(
				"widget evidence has %d entries, want window, panel, text, input, list, and button widgets",
				len(widgets),
			),
		)
	}
	requiredKinds := map[string]bool{
		"window": false,
		"panel":  false,
		"text":   false,
		"input":  false,
		"list":   false,
		"button": false,
	}
	index := map[string]WidgetReport{}
	for _, w := range widgets {
		id := strings.TrimSpace(w.ID)
		if id == "" {
			issues = append(issues, "widget id is required")
			continue
		}
		if _, exists := index[id]; exists {
			issues = append(issues, fmt.Sprintf("duplicate widget %s", id))
		}
		index[id] = w
		if _, ok := requiredKinds[w.Kind]; ok {
			requiredKinds[w.Kind] = true
		} else {
			issues = append(
				issues,
				fmt.Sprintf(("widget %s kind is %q, want window, panel, text, input, list, or "+
					"button"), id, w.Kind),
			)
		}
		if !w.Enabled {
			issues = append(
				issues,
				fmt.Sprintf(
					"widget %s must record enabled=true for passing production evidence",
					id,
				),
			)
		}
		if !w.Visible {
			issues = append(
				issues,
				fmt.Sprintf(
					"widget %s must record visible=true for passing production evidence",
					id,
				),
			)
		}
		if w.Bounds.Width <= 0 || w.Bounds.Height <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("widget %s bounds must have positive width and height", id),
			)
		}
		if w.Kind == "window" {
			if strings.TrimSpace(w.Parent) != "" {
				issues = append(issues, fmt.Sprintf("window widget %s must be a root widget", id))
			}
		} else if strings.TrimSpace(w.Parent) == "" {
			issues = append(issues, fmt.Sprintf("widget %s parent is required", id))
		}
		switch w.Kind {
		case "text", "input", "list", "button", "panel":
			if strings.TrimSpace(w.Binding) == "" {
				issues = append(issues, fmt.Sprintf("widget %s binding is required", id))
			}
		}
		if (w.Kind == "input" || w.Kind == "list" || w.Kind == "button") &&
			strings.TrimSpace(w.Event) == "" {
			issues = append(issues, fmt.Sprintf("widget %s event is required", id))
		}
		if w.Kind == "button" && strings.TrimSpace(w.Command) == "" {
			issues = append(issues, fmt.Sprintf("button widget %s command is required", id))
		}
	}
	for _, w := range widgets {
		if strings.TrimSpace(w.ID) == "" || strings.TrimSpace(w.Parent) == "" {
			continue
		}
		if _, ok := index[w.Parent]; !ok {
			issues = append(issues, fmt.Sprintf("widget %s parent %s is missing", w.ID, w.Parent))
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("widget evidence missing %s widget", kind))
		}
	}
	return index, issues
}

func validateEvents(events []EventReport, widgets map[string]WidgetReport) []string {
	var issues []string
	if len(events) < 6 {
		issues = append(
			issues,
			fmt.Sprintf(
				("event loop evidence has %d events, want focus, input, change, "+
					"selection, command, and timer tick events"),
				len(events),
			),
		)
	}
	lastOrder := 0
	seenAsync := false
	seenRedraw := false
	seenFocus := false
	seenChange := false
	seenInput := false
	seenSelection := false
	seenButton := false
	seenTimerTick := false
	seenTimerOperation := false
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
		}
		if exists && strings.TrimSpace(widget.Event) != "" &&
			!widgetAllowsEvent(widget, event.Event) {
			issues = append(
				issues,
				fmt.Sprintf(
					"event %d kind is %q, want widget event %q",
					event.Order,
					event.Event,
					widget.Event,
				),
			)
		}
		switch event.Event {
		case "click", "activate", "input", "change", "focus", "select", "close", "tick":
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"event %d kind is %q, want click, activate, input, change, focus, select, close, or tick",
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
		operationIssues, hasAsync, hasRedraw, hasTimerOperation := validateOperations(
			event.Order,
			event.Operations,
		)
		issues = append(issues, operationIssues...)
		seenAsync = seenAsync || hasAsync
		seenRedraw = seenRedraw || hasRedraw
		seenTimerOperation = seenTimerOperation || hasTimerOperation
		issues = append(issues, validateWidgetUpdates(event.Order, event.WidgetUpdates, widgets)...)
		if exists && widget.Kind == "input" && event.Event == "focus" {
			seenFocus = true
		}
		if exists && widget.Kind == "input" && event.Event == "input" {
			seenInput = true
		}
		if exists && widget.Kind == "input" && event.Event == "change" {
			seenChange = true
		}
		if exists && widget.Kind == "list" && event.Event == "select" {
			seenSelection = true
		}
		if exists && widget.Kind == "button" &&
			(event.Event == "click" || event.Event == "activate") {
			seenButton = true
		}
		if event.Event == "tick" {
			seenTimerTick = true
		}
	}
	if !seenInput {
		issues = append(issues, "event loop evidence missing input widget event")
	}
	if !seenFocus {
		issues = append(issues, "event loop evidence missing input focus event")
	}
	if !seenChange {
		issues = append(issues, "event loop evidence missing input change event")
	}
	if !seenSelection {
		issues = append(issues, "event loop evidence missing list selection event")
	}
	if !seenButton {
		issues = append(issues, "event loop evidence missing button command event")
	}
	if !seenTimerTick {
		issues = append(issues, "event loop evidence missing timer tick event")
	}
	if !seenAsync {
		issues = append(issues, "event loop evidence missing async UI command operation")
	}
	if !seenRedraw {
		issues = append(issues, "event loop evidence missing redraw operation")
	}
	if !seenTimerOperation {
		issues = append(issues, "event loop evidence missing timer tick operation")
	}
	return issues
}

func widgetAllowsEvent(widget WidgetReport, eventName string) bool {
	if widget.Event == eventName {
		return true
	}
	return widget.Kind == "input" && (eventName == "focus" || eventName == "change")
}

func validateOperations(order int, operations []OperationReport) ([]string, bool, bool, bool) {
	var issues []string
	seenAsync := false
	seenRedraw := false
	seenTimerTick := false
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
		if strings.TrimSpace(op.Value) == "" {
			issues = append(issues, label+" value is required")
		}
		if strings.TrimSpace(op.StateField) == "" {
			issues = append(issues, label+" state_field is required")
		}
		if strings.TrimSpace(op.StateValue) == "" {
			issues = append(issues, label+" state_value is required")
		}
		switch op.Kind {
		case "async_command":
			seenAsync = true
		case "redraw":
			seenRedraw = true
		case "timer_tick":
			seenTimerTick = true
		}
	}
	return issues, seenAsync, seenRedraw, seenTimerTick
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
	required := map[string]bool{
		"window lifecycle":                   false,
		"layout measure and place":           false,
		"button command dispatch":            false,
		"text render":                        false,
		"input focus traversal":              false,
		"input edit":                         false,
		"input change commit":                false,
		"list selection":                     false,
		"panel nesting":                      false,
		"state binding update":               false,
		"event loop dispatch":                false,
		"async UI command completion":        false,
		"timer scheduled redraw":             false,
		"redraw update lifecycle":            false,
		"compiler UI bundle runtime load":    false,
		"native shell runtime integration":   false,
		"native runtime sidecar consistency": false,
		"invalid widget diagnostic":          false,
		"command failure recovery":           false,
		"crash error handling":               false,
		"dogfood application smoke":          false,
		"widget tree stress":                 false,
	}
	negative := map[string]bool{
		"invalid widget diagnostic": true,
		"command failure recovery":  true,
		"crash error handling":      true,
	}
	var issues []string
	seenPositive := false
	seenNegative := false
	seenStress := false
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		switch c.Kind {
		case "positive":
			seenPositive = true
		case "negative":
			seenNegative = true
			if strings.TrimSpace(c.ExpectedError) == "" {
				issues = append(
					issues,
					fmt.Sprintf("negative case %s expected_error is required", name),
				)
			}
		case "stress":
			seenStress = true
		default:
			issues = append(
				issues,
				fmt.Sprintf("case %s kind is %q, want positive, negative, or stress", name, c.Kind),
			)
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", name))
		}
		if negative[name] && strings.TrimSpace(c.ExpectedError) == "" {
			issues = append(issues, fmt.Sprintf("case %s missing expected_error", name))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("case %s has unexpected error: %s", name, c.Error))
		}
	}
	if !seenPositive {
		issues = append(issues, "case evidence missing positive UI case")
	}
	if !seenNegative {
		issues = append(issues, "case evidence missing negative UI safety case")
	}
	if !seenStress {
		issues = append(issues, "case evidence missing UI stress case")
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required UI case %q", name))
		}
	}
	return issues
}

func validateAudit(audit []AuditReport) []string {
	required := map[string]bool{
		"Linux-x64 desktop UI runtime":                                false,
		"window lifecycle":                                            false,
		"layout system":                                               false,
		"buttons/text/input/lists/panels widgets":                     false,
		"state binding":                                               false,
		"event loop and redraw/update model":                          false,
		"async commands and timers":                                   false,
		"error/crash handling":                                        false,
		"real examples and dogfood applications":                      false,
		"compiler-emitted UI bundle/native-shell trace load evidence": false,
		"sidecar-driven native UI runtime integration":                false,
		"stable UI diagnostics":                                       false,
		"release-gate entrypoint rejecting runtime-less evidence":     false,
	}
	var issues []string
	if len(audit) == 0 {
		issues = append(issues, "completion audit is required")
	}
	seen := map[string]bool{}
	for _, row := range audit {
		requirement := strings.TrimSpace(row.Requirement)
		if requirement == "" {
			issues = append(issues, "completion audit row requirement is required")
			continue
		}
		if seen[requirement] {
			issues = append(
				issues,
				fmt.Sprintf("duplicate completion audit requirement %q", requirement),
			)
		}
		seen[requirement] = true
		if _, ok := required[requirement]; ok {
			required[requirement] = true
		}
		if strings.TrimSpace(row.Artifact) == "" {
			issues = append(
				issues,
				fmt.Sprintf("completion audit requirement %q artifact is required", requirement),
			)
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("completion audit requirement %q evidence is required", requirement),
			)
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(row.Result)), "pass") {
			issues = append(
				issues,
				fmt.Sprintf(
					"completion audit requirement %q result is %q, want pass",
					requirement,
					row.Result,
				),
			)
		}
	}
	for requirement, ok := range required {
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("completion audit missing required requirement %q", requirement),
			)
		}
	}
	return issues
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
