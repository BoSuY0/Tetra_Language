package uitoolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const SchemaV1 = "tetra.ui.toolkit.v1"
const TraceSchemaV1 = "tetra.ui.toolkit.trace.v1"

type Report struct {
	Schema           string                  `json:"schema"`
	Status           string                  `json:"status"`
	Target           string                  `json:"target"`
	Host             string                  `json:"host"`
	Runtime          string                  `json:"runtime"`
	UISchema         string                  `json:"ui_schema"`
	Source           string                  `json:"source"`
	Artifacts        []ArtifactReport        `json:"artifacts"`
	Processes        []ProcessReport         `json:"processes"`
	Contracts        []ContractReport        `json:"contracts"`
	Widgets          []WidgetReport          `json:"widgets"`
	Layouts          []LayoutReport          `json:"layouts"`
	Events           []EventReport           `json:"events"`
	StateTransitions []StateTransitionReport `json:"state_transitions"`
	Cases            []CaseReport            `json:"cases"`
	Audit            []AuditReport           `json:"audit"`
}

type ArtifactReport struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
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
	ID            string                `json:"id"`
	Kind          string                `json:"kind"`
	Parent        string                `json:"parent"`
	Binding       string                `json:"binding"`
	Event         string                `json:"event,omitempty"`
	Command       string                `json:"command,omitempty"`
	Value         string                `json:"value,omitempty"`
	Enabled       bool                  `json:"enabled"`
	Visible       bool                  `json:"visible"`
	Focusable     bool                  `json:"focusable"`
	Bounds        Bounds                `json:"bounds"`
	Layout        WidgetLayout          `json:"layout"`
	Style         WidgetStyle           `json:"style"`
	Accessibility AccessibilityMetadata `json:"accessibility"`
}

type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type WidgetLayout struct {
	Kind           string `json:"kind"`
	Order          int    `json:"order"`
	Gap            int    `json:"gap,omitempty"`
	MinWidth       int    `json:"min_width,omitempty"`
	MaxWidth       int    `json:"max_width,omitempty"`
	PreferredWidth int    `json:"preferred_width,omitempty"`
	Overflow       string `json:"overflow,omitempty"`
}

type WidgetStyle struct {
	Class      string `json:"class"`
	State      string `json:"state,omitempty"`
	Color      string `json:"color,omitempty"`
	Background string `json:"background,omitempty"`
	Border     string `json:"border,omitempty"`
	Text       string `json:"text,omitempty"`
}

type AccessibilityMetadata struct {
	Role               string   `json:"role"`
	Label              string   `json:"label"`
	Description        string   `json:"description"`
	FocusOrder         int      `json:"focus_order"`
	KeyboardActivation []string `json:"keyboard_activation"`
	State              string   `json:"state,omitempty"`
}

type LayoutReport struct {
	Kind     string   `json:"kind"`
	Widgets  []string `json:"widgets"`
	Pass     bool     `json:"pass"`
	Evidence string   `json:"evidence"`
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

type StateTransitionReport struct {
	Name       string            `json:"name"`
	Before     map[string]string `json:"before"`
	After      map[string]string `json:"after"`
	Operations []string          `json:"operations"`
	Widgets    []string          `json:"widgets"`
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
	issues = append(issues, rejectPaperEvidence(raw)...)
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "toolkit-core" {
		issues = append(issues, fmt.Sprintf("target is %q, want toolkit-core", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "toolkit-core" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want toolkit-core", report.Runtime))
	}
	if report.UISchema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("ui_schema is %q, want %s", report.UISchema, SchemaV1))
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateArtifacts(report.Artifacts)...)
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateContracts(report.Contracts)...)
	widgets, widgetIssues := validateWidgets(report.Widgets)
	issues = append(issues, widgetIssues...)
	issues = append(issues, validateLayouts(report.Layouts, widgets)...)
	issues = append(issues, validateEvents(report.Events, widgets)...)
	issues = append(issues, validateStateTransitions(report.StateTransitions, widgets)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateAudit(report.Audit)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{
		"metadata-only",
		"preview-only",
		"runtime-less",
		"native-shell sidecar-only",
		"sidecar-only",
		"web-only",
		"docs-only",
		"build-only",
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
					"report contains forbidden non-production toolkit evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	return issues
}

func validateArtifacts(artifacts []ArtifactReport) []string {
	var issues []string
	if len(artifacts) < 2 {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact evidence has %d entries, want bundle and runtime trace artifacts",
				len(artifacts),
			),
		)
	}
	requiredKinds := map[string]string{
		"bundle": SchemaV1,
		"trace":  TraceSchemaV1,
	}
	seenKinds := map[string]bool{}
	seenPaths := map[string]bool{}
	for _, artifact := range artifacts {
		name := strings.TrimSpace(artifact.Name)
		if name == "" {
			issues = append(issues, "artifact name is required")
			name = "<unnamed>"
		}
		kind := strings.TrimSpace(artifact.Kind)
		if expectedSchema, ok := requiredKinds[kind]; ok {
			seenKinds[kind] = true
			if artifact.Schema != expectedSchema {
				issues = append(
					issues,
					fmt.Sprintf(
						"artifact %s schema is %q, want %q",
						name,
						artifact.Schema,
						expectedSchema,
					),
				)
			}
		} else {
			issues = append(
				issues,
				fmt.Sprintf("artifact %s kind is %q, want bundle or trace", name, artifact.Kind),
			)
		}
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			issues = append(issues, fmt.Sprintf("artifact %s path is required", name))
		} else {
			if seenPaths[path] {
				issues = append(issues, fmt.Sprintf("duplicate artifact path %s", path))
			}
			seenPaths[path] = true
			if info, err := os.Stat(path); err != nil {
				issues = append(issues, fmt.Sprintf("artifact %s path %s is not readable: %v", name, path, err))
			} else if info.IsDir() {
				issues = append(issues, fmt.Sprintf("artifact %s path %s is a directory", name, path))
			}
		}
		if err := validateSHA256(artifact.SHA256, name); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for kind := range requiredKinds {
		if !seenKinds[kind] {
			issues = append(issues, fmt.Sprintf("artifact evidence missing %s artifact", kind))
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(
			issues,
			fmt.Sprintf(
				"process evidence has %d entries, want runtime, stress, and validator processes",
				len(processes),
			),
		)
	}
	requiredKinds := map[string]bool{
		"runtime":   false,
		"stress":    false,
		"validator": false,
	}
	names := map[string]bool{}
	for _, p := range processes {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			issues = append(issues, "process name is required")
			name = "<unnamed>"
		} else if names[name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", name))
		}
		names[name] = true
		if _, ok := requiredKinds[p.Kind]; ok {
			requiredKinds[p.Kind] = true
		} else if p.Kind == "build" {
			issues = append(
				issues,
				fmt.Sprintf("process %s is build-only evidence; toolkit core requires runtime execution", name),
			)
		} else {
			issues = append(
				issues,
				fmt.Sprintf("process %s kind is %q, want runtime, stress, or validator", name, p.Kind),
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
	return issues
}

func validateContracts(contracts []ContractReport) []string {
	required := map[string]bool{
		"toolkit schema":      false,
		"widget model":        false,
		"layout model":        false,
		"style model":         false,
		"accessibility model": false,
		"event model":         false,
		"state binding model": false,
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
			issues = append(issues, fmt.Sprintf("missing required toolkit contract %q", name))
		}
	}
	return issues
}

func validateWidgets(widgets []WidgetReport) (map[string]WidgetReport, []string) {
	requiredKinds := map[string]bool{
		"window":    false,
		"root":      false,
		"panel":     false,
		"text":      false,
		"label":     false,
		"button":    false,
		"input":     false,
		"checkbox":  false,
		"select":    false,
		"list":      false,
		"table":     false,
		"dialog":    false,
		"menu":      false,
		"menu-item": false,
		"spacer":    false,
		"divider":   false,
	}
	var issues []string
	if len(widgets) < len(requiredKinds) {
		issues = append(
			issues,
			fmt.Sprintf(
				"widget evidence has %d entries, want all %d toolkit widget kinds",
				len(widgets),
				len(requiredKinds),
			),
		)
	}
	index := map[string]WidgetReport{}
	focusOrders := map[int]string{}
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
				fmt.Sprintf("widget %s kind is %q, want selected toolkit widget kind", id, w.Kind),
			)
		}
		if !w.Enabled {
			issues = append(
				issues,
				fmt.Sprintf("widget %s must record enabled=true for passing toolkit evidence", id),
			)
		}
		if !w.Visible {
			issues = append(
				issues,
				fmt.Sprintf("widget %s must record visible=true for passing toolkit evidence", id),
			)
		}
		if w.Bounds.Width <= 0 || w.Bounds.Height <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("widget %s bounds must have positive width and height", id),
			)
		}
		if strings.TrimSpace(w.Layout.Kind) == "" {
			issues = append(issues, fmt.Sprintf("widget %s layout.kind is required", id))
		}
		if strings.TrimSpace(w.Style.Class) == "" {
			issues = append(issues, fmt.Sprintf("widget %s style.class is required", id))
		}
		issues = append(issues, validateAccessibility(id, w)...)
		if strings.TrimSpace(w.Binding) == "" {
			issues = append(issues, fmt.Sprintf("widget %s binding is required", id))
		}
		switch w.Kind {
		case "button", "input", "checkbox", "select", "list", "table", "dialog", "menu-item":
			if strings.TrimSpace(w.Event) == "" {
				issues = append(issues, fmt.Sprintf("widget %s event is required", id))
			}
			if strings.TrimSpace(w.Command) == "" {
				issues = append(issues, fmt.Sprintf("widget %s command is required", id))
			}
		}
		if w.Focusable {
			if w.Accessibility.FocusOrder <= 0 {
				issues = append(
					issues,
					fmt.Sprintf("widget %s focus_order must be positive for focusable widgets", id),
				)
			} else if prior := focusOrders[w.Accessibility.FocusOrder]; prior != "" {
				issues = append(issues, fmt.Sprintf("widget %s focus_order duplicates widget %s", id, prior))
			} else {
				focusOrders[w.Accessibility.FocusOrder] = id
			}
			if len(w.Accessibility.KeyboardActivation) == 0 {
				issues = append(
					issues,
					fmt.Sprintf(
						"widget %s keyboard_activation is required for focusable widgets",
						id,
					),
				)
			}
		}
	}
	for _, w := range widgets {
		if strings.TrimSpace(w.ID) == "" {
			continue
		}
		parent := strings.TrimSpace(w.Parent)
		if w.Kind == "window" {
			if parent != "" {
				issues = append(issues, fmt.Sprintf("window widget %s must be a root widget", w.ID))
			}
			continue
		}
		if parent == "" {
			issues = append(issues, fmt.Sprintf("widget %s parent is required", w.ID))
			continue
		}
		if _, ok := index[parent]; !ok {
			issues = append(issues, fmt.Sprintf("widget %s parent %s is missing", w.ID, parent))
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("widget evidence missing %s widget", kind))
		}
	}
	return index, issues
}

func validateAccessibility(id string, w WidgetReport) []string {
	var issues []string
	if strings.TrimSpace(w.Accessibility.Role) == "" {
		issues = append(issues, fmt.Sprintf("widget %s accessibility.role is required", id))
	}
	if strings.TrimSpace(w.Accessibility.Label) == "" {
		issues = append(issues, fmt.Sprintf("widget %s accessibility.label is required", id))
	}
	if strings.TrimSpace(w.Accessibility.Description) == "" {
		issues = append(issues, fmt.Sprintf("widget %s accessibility.description is required", id))
	}
	return issues
}

func validateLayouts(layouts []LayoutReport, widgets map[string]WidgetReport) []string {
	requiredKinds := map[string]bool{
		"stack":           false,
		"row":             false,
		"column":          false,
		"grid":            false,
		"flex":            false,
		"overflow-scroll": false,
	}
	var issues []string
	for _, layout := range layouts {
		kind := strings.TrimSpace(layout.Kind)
		if _, ok := requiredKinds[kind]; ok {
			requiredKinds[kind] = true
		} else {
			issues = append(
				issues,
				fmt.Sprintf(("layout kind is %q, want stack, row, column, grid, flex, or "+
					"overflow-scroll"), layout.Kind),
			)
		}
		if !layout.Pass {
			issues = append(issues, fmt.Sprintf("layout %s did not pass", kind))
		}
		if strings.TrimSpace(layout.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("layout %s evidence is required", kind))
		}
		if len(layout.Widgets) == 0 {
			issues = append(issues, fmt.Sprintf("layout %s widgets are required", kind))
		}
		for _, widgetID := range layout.Widgets {
			if _, ok := widgets[widgetID]; !ok {
				issues = append(
					issues,
					fmt.Sprintf("layout %s references missing widget %s", kind, widgetID),
				)
			}
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("layout evidence missing %s layout", kind))
		}
	}
	return issues
}

func validateEvents(events []EventReport, widgets map[string]WidgetReport) []string {
	requiredEvents := map[string]bool{
		"click":          false,
		"activate":       false,
		"focus":          false,
		"blur":           false,
		"input":          false,
		"change":         false,
		"select":         false,
		"submit":         false,
		"key":            false,
		"timer":          false,
		"error_recovery": false,
	}
	requiredOperations := map[string]bool{
		"async_command":  false,
		"redraw":         false,
		"two_way_bind":   false,
		"key_activate":   false,
		"timer_tick":     false,
		"error_recovery": false,
		"state_set":      false,
		"focus":          false,
		"blur":           false,
	}
	var issues []string
	if len(events) < len(requiredEvents) {
		issues = append(
			issues,
			fmt.Sprintf(
				"event evidence has %d events, want all %d toolkit events",
				len(events),
				len(requiredEvents),
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
		if _, ok := requiredEvents[event.Event]; ok {
			requiredEvents[event.Event] = true
		} else {
			issues = append(
				issues,
				fmt.Sprintf("event %d kind is %q, want selected toolkit event", event.Order, event.Event),
			)
		}
		if strings.TrimSpace(event.WidgetID) == "" {
			issues = append(issues, fmt.Sprintf("event %d widget_id is required", event.Order))
		} else if _, ok := widgets[event.WidgetID]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("event %d widget_id %s is not in widget tree", event.Order, event.WidgetID),
			)
		}
		if strings.TrimSpace(event.Command) == "" {
			issues = append(issues, fmt.Sprintf("event %d command is required", event.Order))
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
		for i, op := range event.Operations {
			label := fmt.Sprintf("event %d operation %d", event.Order, i+1)
			if _, ok := requiredOperations[op.Kind]; ok {
				requiredOperations[op.Kind] = true
			} else {
				issues = append(
					issues,
					fmt.Sprintf("%s kind is %q, want selected toolkit operation", label, op.Kind),
				)
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
		}
		if len(event.Operations) == 0 {
			issues = append(issues, fmt.Sprintf("event %d operations are required", event.Order))
		}
		issues = append(issues, validateWidgetUpdates(event, widgets)...)
	}
	for event, seen := range requiredEvents {
		if !seen {
			issues = append(issues, fmt.Sprintf("event evidence missing %s event", event))
		}
	}
	for op, seen := range requiredOperations {
		if !seen {
			issues = append(issues, fmt.Sprintf("event evidence missing %s operation", op))
		}
	}
	return issues
}

func validateWidgetUpdates(event EventReport, widgets map[string]WidgetReport) []string {
	var issues []string
	if len(event.WidgetUpdates) == 0 {
		issues = append(issues, fmt.Sprintf("event %d widget_updates are required", event.Order))
	}
	for i, update := range event.WidgetUpdates {
		label := fmt.Sprintf("event %d widget_update %d", event.Order, i+1)
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

func validateStateTransitions(
	transitions []StateTransitionReport,
	widgets map[string]WidgetReport,
) []string {
	required := map[string]bool{
		"scalar binding update":      false,
		"list selection binding":     false,
		"table selection binding":    false,
		"two-way input binding":      false,
		"deterministic update order": false,
	}
	var issues []string
	for _, transition := range transitions {
		name := strings.TrimSpace(transition.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		} else {
			issues = append(
				issues,
				fmt.Sprintf("state transition %q is not selected toolkit evidence", transition.Name),
			)
		}
		if len(transition.Before) == 0 || len(transition.After) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("state transition %s must include before and after state", name),
			)
		} else if !stateChanged(transition.Before, transition.After) {
			issues = append(issues, fmt.Sprintf("state transition %s has no observable state change", name))
		}
		if len(transition.Operations) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("state transition %s operations are required", name),
			)
		}
		if len(transition.Widgets) == 0 {
			issues = append(issues, fmt.Sprintf("state transition %s widgets are required", name))
		}
		for _, widgetID := range transition.Widgets {
			if _, ok := widgets[widgetID]; !ok {
				issues = append(
					issues,
					fmt.Sprintf("state transition %s references missing widget %s", name, widgetID),
				)
			}
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("state transition evidence missing %s", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"positive widget tree":             false,
		"layout stress":                    false,
		"event dispatch":                   false,
		"state binding update":             false,
		"input focus select key":           false,
		"timer async redraw":               false,
		"dialog menu":                      false,
		"table list binding":               false,
		"accessibility metadata":           false,
		"unsupported widget diagnostic":    false,
		"unsupported operation diagnostic": false,
		"malformed metadata":               false,
		"command failure recovery":         false,
		"crash error recovery":             false,
	}
	negative := map[string]bool{
		"unsupported widget diagnostic":    true,
		"unsupported operation diagnostic": true,
		"malformed metadata":               true,
		"command failure recovery":         true,
		"crash error recovery":             true,
	}
	var issues []string
	seenPositive := false
	seenNegative := false
	seenStress := false
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		} else {
			issues = append(issues, fmt.Sprintf("case %q is not selected toolkit evidence", c.Name))
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
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("case evidence missing %s", name))
		}
	}
	if !seenPositive {
		issues = append(issues, "case evidence missing positive case")
	}
	if !seenNegative {
		issues = append(issues, "case evidence missing negative case")
	}
	if !seenStress {
		issues = append(issues, "case evidence missing stress case")
	}
	return issues
}

func validateAudit(audits []AuditReport) []string {
	required := map[string]bool{
		"toolkit core contract":      false,
		"real runtime evidence":      false,
		"widget model":               false,
		"layout focus accessibility": false,
		"event state update model":   false,
		"negative diagnostics":       false,
	}
	var issues []string
	for _, audit := range audits {
		name := strings.TrimSpace(audit.Requirement)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if name == "" {
			issues = append(issues, "audit requirement is required")
		}
		if strings.TrimSpace(audit.Artifact) == "" {
			issues = append(issues, fmt.Sprintf("audit %s artifact is required", name))
		}
		if strings.TrimSpace(audit.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("audit %s evidence is required", name))
		}
		if audit.Result != "pass" {
			issues = append(
				issues,
				fmt.Sprintf("audit %s result is %q, want pass", name, audit.Result),
			)
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("audit evidence missing %s", name))
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

func validateSHA256(value string, name string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("artifact %s has invalid sha256 format %q", name, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact %s sha256 must contain 64 hex chars", name)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("artifact %s sha256 has non-hex characters", name)
		}
	}
	return nil
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
