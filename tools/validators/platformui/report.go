package platformui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaV1 = "tetra.ui.platform-runtime.v1"

var AcceptedUISchemas = map[string]bool{
	"tetra.ui.v1":     true,
	"tetra.ui.v0.4.0": true,
}

type Report struct {
	Schema       string          `json:"schema"`
	Status       string          `json:"status"`
	Version      string          `json:"version,omitempty"`
	GitHead      string          `json:"git_head,omitempty"`
	Target       string          `json:"target"`
	Host         string          `json:"host"`
	Runtime      string          `json:"runtime"`
	RuntimeTrace string          `json:"runtime_trace,omitempty"`
	UISchema     string          `json:"ui_schema"`
	Source       string          `json:"source"`
	Runner       string          `json:"runner"`
	Blocker      string          `json:"blocker,omitempty"`
	Processes    []ProcessReport `json:"processes"`
	Widgets      []WidgetReport  `json:"widgets"`
	Events       []EventReport   `json:"events"`
	Cases        []CaseReport    `json:"cases"`
	Audit        []AuditReport   `json:"audit"`
}

type ValidateOptions struct {
	ExpectedTarget  string
	ExpectedVersion string
	ExpectedGitHead string
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
	Enabled bool   `json:"enabled"`
	Visible bool   `json:"visible"`
	Bounds  Bounds `json:"bounds"`
}

type Bounds struct {
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
	Kind string `json:"kind"`
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
}

type AuditReport struct {
	Requirement string `json:"requirement"`
	Artifact    string `json:"artifact"`
	Evidence    string `json:"evidence"`
	Result      string `json:"result"`
}

func ValidateReport(raw []byte, expectedTarget string) error {
	return ValidateReportWithOptions(raw, ValidateOptions{ExpectedTarget: expectedTarget})
}

func ValidateReportWithOptions(raw []byte, options ValidateOptions) error {
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
	if options.ExpectedVersion != "" && report.Version != options.ExpectedVersion {
		issues = append(issues, fmt.Sprintf("version is %q, want %q", report.Version, options.ExpectedVersion))
	}
	if options.ExpectedGitHead != "" && !gitHeadMatches(report.GitHead, options.ExpectedGitHead) {
		issues = append(issues, fmt.Sprintf("git_head is %q, want %q", report.GitHead, options.ExpectedGitHead))
	}
	if options.ExpectedTarget != "" && report.Target != options.ExpectedTarget {
		issues = append(issues, fmt.Sprintf("target is %q, want %s", report.Target, options.ExpectedTarget))
	}
	if report.Host != report.Target {
		issues = append(issues, fmt.Sprintf("host is %q, want matching target host %q", report.Host, report.Target))
	}
	if report.Runtime != "platform-ui-"+report.Target {
		issues = append(issues, fmt.Sprintf("runtime is %q, want platform-ui-%s", report.Runtime, report.Target))
	}
	if !AcceptedUISchemas[report.UISchema] {
		issues = append(issues, fmt.Sprintf("ui_schema is %q, want tetra.ui.v1 or tetra.ui.v0.4.0", report.UISchema))
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	if strings.TrimSpace(report.Runner) == "" {
		issues = append(issues, "runner is required")
	} else if report.Runner != "target-host-runtime-child" {
		issues = append(issues, fmt.Sprintf("runner is %q, want target-host-runtime-child", report.Runner))
	}
	if report.Blocker != "" {
		issues = append(issues, "blocker must be empty for production evidence")
	}
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateRuntimeTrace(report.RuntimeTrace)...)
	issues = append(issues, validateWidgets(report.Widgets)...)
	issues = append(issues, validateEvents(report.Events)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateAudit(report.Audit)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func gitHeadMatches(artifactHead string, currentHead string) bool {
	artifactHead = strings.TrimSpace(artifactHead)
	currentHead = strings.TrimSpace(currentHead)
	if artifactHead == "" || currentHead == "" {
		return false
	}
	if artifactHead == currentHead {
		return true
	}
	if len(artifactHead) >= 7 && len(currentHead) >= 7 {
		return strings.HasPrefix(artifactHead, currentHead) || strings.HasPrefix(currentHead, artifactHead)
	}
	return false
}

func rejectPaperEvidence(report Report) []string {
	fields := []string{report.Source, report.Runner, report.Blocker}
	for _, process := range report.Processes {
		fields = append(fields, process.Name, process.Kind, process.Path)
	}
	lower := strings.ToLower(strings.Join(fields, "\n"))
	for _, phrase := range []string{"reject runtime-less evidence", "rejects runtime-less evidence"} {
		lower = strings.ReplaceAll(lower, phrase, "rejects non-production evidence")
	}
	var issues []string
	for _, marker := range []string{"metadata-only", "runtime-less", "build-only", "docs-only", "sidecar-only", " fake", "fake/", "\"fake\"", " mock", "mock/", "\"mock\"", "placeholder"} {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("report contains forbidden non-production UI evidence marker %q", strings.Trim(marker, " /\"")))
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	required := map[string]bool{"build": false, "app": false, "runtime": false, "stress": false}
	var issues []string
	for _, process := range processes {
		if _, ok := required[process.Kind]; ok {
			required[process.Kind] = true
		}
		if strings.TrimSpace(process.Name) == "" || strings.TrimSpace(process.Path) == "" || !process.Ran || !process.Pass || process.ExitCode == nil || *process.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s must include name/path and pass with exit_code 0", process.Name))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("process evidence missing %s process", kind))
		}
	}
	return issues
}

func validateRuntimeTrace(trace string) []string {
	required := []string{
		"platform-process-spawn:ok",
		"platform-window-api:ok",
		"platform-widget-tree:ok",
		"platform-event-dispatch:ok",
		"platform-timer:ok",
		"platform-redraw:ok",
		"window-create:ok",
		"window-show:ok",
		"widget-tree-load:ok",
		"layout-measure:ok",
		"layout-place:ok",
		"event-loop-start:ok",
		"focus-dispatch:ok",
		"input-dispatch:ok",
		"select-dispatch:ok",
		"click-dispatch:ok",
		"state-update:ok",
		"async-command:ok",
		"timer-tick:ok",
		"redraw:ok",
		"error-recovery:ok",
		"window-close:ok",
	}
	var issues []string
	for _, marker := range required {
		if !strings.Contains(trace, marker) {
			issues = append(issues, "runtime_trace missing "+marker)
		}
	}
	return issues
}

func validateWidgets(widgets []WidgetReport) []string {
	required := map[string]bool{"window": false, "panel": false, "text": false, "button": false, "input": false, "list": false}
	var issues []string
	for _, widget := range widgets {
		if _, ok := required[widget.Kind]; ok {
			required[widget.Kind] = true
		}
		if strings.TrimSpace(widget.ID) == "" || !widget.Enabled || !widget.Visible || widget.Bounds.Width <= 0 || widget.Bounds.Height <= 0 {
			issues = append(issues, fmt.Sprintf("widget %s must include id, enabled/visible state, and positive bounds", widget.ID))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("widget evidence missing %s widget", kind))
		}
	}
	return issues
}

func validateEvents(events []EventReport) []string {
	requiredEvents := map[string]bool{"focus": false, "input": false, "select": false, "click": false, "tick": false}
	requiredOps := map[string]bool{"state_set": false, "async_command": false, "timer_tick": false, "redraw": false}
	var issues []string
	for _, event := range events {
		if _, ok := requiredEvents[event.Event]; ok {
			requiredEvents[event.Event] = true
		}
		if event.Order <= 0 || strings.TrimSpace(event.WidgetID) == "" || !event.Pass || len(event.BeforeState) == 0 || len(event.AfterState) == 0 || len(event.WidgetUpdates) == 0 {
			issues = append(issues, fmt.Sprintf("event %s must include order/widget/state/update/pass evidence", event.Event))
		}
		for _, op := range event.Operations {
			if _, ok := requiredOps[op.Kind]; ok {
				requiredOps[op.Kind] = true
			}
		}
	}
	for name, seen := range requiredEvents {
		if !seen {
			issues = append(issues, fmt.Sprintf("event evidence missing %s event", name))
		}
	}
	for name, seen := range requiredOps {
		if !seen {
			issues = append(issues, fmt.Sprintf("event evidence missing %s operation", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"window lifecycle":            false,
		"layout measure and place":    false,
		"widget tree load":            false,
		"event loop dispatch":         false,
		"state binding update":        false,
		"redraw update lifecycle":     false,
		"async UI command completion": false,
		"timer scheduled redraw":      false,
		"invalid widget diagnostic":   false,
		"command failure recovery":    false,
		"crash error handling":        false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", c.Name))
		}
		if c.Kind == "negative" && strings.TrimSpace(c.ExpectedError) == "" {
			issues = append(issues, fmt.Sprintf("negative case %s missing expected_error", c.Name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("case evidence missing %s", name))
		}
	}
	return issues
}

func validateAudit(audit []AuditReport) []string {
	required := map[string]bool{"real platform runtime evidence": false, "reject runtime-less evidence": false}
	var issues []string
	for _, entry := range audit {
		if _, ok := required[entry.Requirement]; ok {
			required[entry.Requirement] = entry.Result == "pass"
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("audit missing passing %s", name))
		}
	}
	return issues
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
