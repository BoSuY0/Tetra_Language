package uiplatform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const SchemaV1 = "tetra.ui.platform.v1"
const DefaultMaxEvidenceAge = 7 * 24 * time.Hour

type Options struct {
	Target  string
	Host    string
	Runtime string
	Now     time.Time
	MaxAge  time.Duration
}

type Report struct {
	Schema       string           `json:"schema"`
	GeneratedAt  string           `json:"generated_at"`
	Status       string           `json:"status"`
	Target       string           `json:"target"`
	Host         string           `json:"host"`
	Platform     string           `json:"platform"`
	Runtime      string           `json:"runtime"`
	UISchema     string           `json:"ui_schema"`
	EvidenceKind string           `json:"evidence_kind"`
	Source       string           `json:"source"`
	Blocker      string           `json:"blocker,omitempty"`
	Processes    []ProcessReport  `json:"processes"`
	Contracts    []ContractReport `json:"contracts"`
	Widgets      []WidgetReport   `json:"widgets"`
	Events       []EventReport    `json:"events"`
	Cases        []CaseReport     `json:"cases"`
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

func ValidateReport(raw []byte, opts Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	issues = append(issues, rejectNonRuntimeEvidence(report)...)
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	issues = append(issues, validateGeneratedAt(report.GeneratedAt, opts)...)
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != opts.Target {
		issues = append(issues, fmt.Sprintf("target is %q, want %q", report.Target, opts.Target))
	}
	if report.Host != opts.Host {
		issues = append(issues, fmt.Sprintf("host is %q, want %q", report.Host, opts.Host))
	}
	if report.Runtime != opts.Runtime {
		issues = append(issues, fmt.Sprintf("runtime is %q, want %q", report.Runtime, opts.Runtime))
	}
	if report.Host != report.Target {
		issues = append(issues, "host must equal target for target-host UI runtime evidence")
	}
	if report.UISchema != "tetra.ui.v1" {
		issues = append(issues, fmt.Sprintf("ui_schema is %q, want tetra.ui.v1", report.UISchema))
	}
	if report.EvidenceKind != "target-host-runtime" {
		issues = append(
			issues,
			fmt.Sprintf("evidence_kind is %q, want target-host-runtime", report.EvidenceKind),
		)
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateContracts(report.Contracts)...)
	widgets, widgetIssues := validateWidgets(report.Widgets)
	issues = append(issues, widgetIssues...)
	issues = append(issues, validateEvents(report.Events, widgets)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("unexpected trailing JSON payload")
	}
	return nil
}

func rejectNonRuntimeEvidence(report Report) []string {
	text := strings.ToLower(strings.Join(reportEvidenceFields(report), "\n"))
	forbiddenPhrases := []string{
		"metadata-only",
		"runtime-less",
		"build-only",
		"docs-only",
		"sidecar-only",
		"startup_failure",
		"placeholder",
	}
	var issues []string
	for _, marker := range forbiddenPhrases {
		if strings.Contains(text, marker) {
			issues = append(
				issues,
				fmt.Sprintf(
					"report contains forbidden non-runtime evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	for _, token := range forbiddenEvidenceTokens(text) {
		issues = append(
			issues,
			fmt.Sprintf("report contains forbidden non-runtime evidence marker %q", token),
		)
	}
	return issues
}

func forbiddenEvidenceTokens(text string) []string {
	forbidden := map[string]bool{"fake": true, "mock": true}
	seen := map[string]bool{}
	var matches []string
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if forbidden[token] && !seen[token] {
			seen[token] = true
			matches = append(matches, token)
		}
	}
	return matches
}

func reportEvidenceFields(report Report) []string {
	fields := []string{
		report.Source,
		report.Blocker,
		report.EvidenceKind,
		report.Runtime,
		report.GeneratedAt,
	}
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
	}
	for _, c := range report.Cases {
		fields = append(fields, c.Name, c.Kind, c.ExpectedError, c.Error)
	}
	return fields
}

func validateGeneratedAt(value string, opts Options) []string {
	if strings.TrimSpace(value) == "" {
		return []string{"generated_at is required"}
	}
	generatedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return []string{fmt.Sprintf("generated_at is not RFC3339: %v", err)}
	}
	if opts.Now.IsZero() {
		return nil
	}
	now := opts.Now.UTC()
	maxAge := opts.MaxAge
	if maxAge == 0 {
		maxAge = DefaultMaxEvidenceAge
	}
	if generatedAt.After(now.Add(5 * time.Minute)) {
		return []string{
			fmt.Sprintf(
				"generated_at %s is in the future relative to %s",
				generatedAt.Format(time.RFC3339),
				now.Format(time.RFC3339),
			),
		}
	}
	if now.Sub(generatedAt) > maxAge {
		return []string{
			fmt.Sprintf(
				"generated_at %s is stale; max age is %s",
				generatedAt.Format(time.RFC3339),
				maxAge,
			),
		}
	}
	return nil
}

func validateProcesses(processes []ProcessReport) []string {
	requiredKinds := map[string]bool{
		"build":   false,
		"app":     false,
		"runtime": false,
		"stress":  false,
	}
	var issues []string
	for _, p := range processes {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			issues = append(issues, "process name is required")
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
		if !p.Ran || !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s must run and pass", name))
		}
		if p.ExitCode == nil || *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code must be 0", name))
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
		"window lifecycle":          false,
		"widget tree":               false,
		"layout":                    false,
		"event dispatch":            false,
		"state redraw async timers": false,
		"negative diagnostics":      false,
	}
	var issues []string
	for _, c := range contracts {
		name := strings.TrimSpace(c.Name)
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
			issues = append(issues, fmt.Sprintf("missing contract %s", name))
		}
	}
	return issues
}

func validateWidgets(widgets []WidgetReport) (map[string]WidgetReport, []string) {
	requiredKinds := map[string]bool{
		"window": false,
		"panel":  false,
		"text":   false,
		"button": false,
		"input":  false,
		"list":   false,
	}
	index := map[string]WidgetReport{}
	var issues []string
	for _, w := range widgets {
		if strings.TrimSpace(w.ID) == "" {
			issues = append(issues, "widget id is required")
			continue
		}
		if _, exists := index[w.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate widget %s", w.ID))
		}
		index[w.ID] = w
		if _, ok := requiredKinds[w.Kind]; ok {
			requiredKinds[w.Kind] = true
		}
		if !w.Enabled || !w.Visible {
			issues = append(issues, fmt.Sprintf("widget %s must be enabled and visible", w.ID))
		}
		if w.Bounds.Width <= 0 || w.Bounds.Height <= 0 {
			issues = append(issues, fmt.Sprintf("widget %s bounds must be non-zero", w.ID))
		}
		if w.Kind != "window" && strings.TrimSpace(w.Parent) == "" {
			issues = append(issues, fmt.Sprintf("widget %s parent is required", w.ID))
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("widget tree missing %s widget", kind))
		}
	}
	return index, issues
}

func validateEvents(events []EventReport, widgets map[string]WidgetReport) []string {
	requiredEvents := map[string]bool{
		"focus":  false,
		"input":  false,
		"change": false,
		"select": false,
		"click":  false,
		"tick":   false,
	}
	requiredOps := map[string]bool{
		"focus":         false,
		"state_set":     false,
		"change":        false,
		"async_command": false,
		"timer_tick":    false,
		"redraw":        false,
	}
	var issues []string
	for _, event := range events {
		if _, ok := widgets[event.WidgetID]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("event %d references unknown widget %s", event.Order, event.WidgetID),
			)
		}
		if _, ok := requiredEvents[event.Event]; ok {
			requiredEvents[event.Event] = true
		}
		if !event.Pass {
			issues = append(issues, fmt.Sprintf("event %d did not pass", event.Order))
		}
		if len(event.BeforeState) == 0 || len(event.AfterState) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("event %d must include before/after state", event.Order),
			)
		}
		if len(event.WidgetUpdates) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("event %d must include widget update evidence", event.Order),
			)
		}
		for _, op := range event.Operations {
			if _, ok := requiredOps[op.Kind]; ok {
				requiredOps[op.Kind] = true
			}
		}
	}
	for name, seen := range requiredEvents {
		if !seen {
			issues = append(issues, fmt.Sprintf("runtime event evidence missing %s", name))
		}
	}
	for name, seen := range requiredOps {
		if !seen {
			issues = append(issues, fmt.Sprintf("runtime operation evidence missing %s", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	requiredKinds := map[string]bool{"positive": false, "negative": false, "stress": false}
	requiredNames := map[string]bool{
		"window lifecycle":               false,
		"widget tree":                    false,
		"layout measure and place":       false,
		"event dispatch":                 false,
		"state update redraw":            false,
		"async command completion":       false,
		"timer tick":                     false,
		"unsupported feature diagnostic": false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := requiredKinds[c.Kind]; ok {
			requiredKinds[c.Kind] = true
		}
		if _, ok := requiredNames[c.Name]; ok {
			requiredNames[c.Name] = true
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s must run and pass", c.Name))
		}
		if c.Kind == "negative" && strings.TrimSpace(c.ExpectedError) == "" {
			issues = append(issues, fmt.Sprintf("negative case %s missing expected_error", c.Name))
		}
	}
	for kind, seen := range requiredKinds {
		if !seen {
			issues = append(issues, fmt.Sprintf("case evidence missing %s case", kind))
		}
	}
	for name, seen := range requiredNames {
		if !seen {
			issues = append(issues, fmt.Sprintf("case evidence missing %s", name))
		}
	}
	return issues
}
