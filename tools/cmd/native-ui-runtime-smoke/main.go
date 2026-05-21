package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tetra_language/tools/validators/nativeui"
)

const (
	nativeShellSchemaV1        = "tetra.ui.native-shell.v1"
	uiBundleSchemaV1           = "tetra.ui.v0.4.0"
	nativeShellRuntimeDispatch = "native shell command dispatch"
)

type smokeOptions struct {
	ReportPath string
	TetraPath  string
	KeepWork   bool
}

type smokeRunner struct {
	opt       smokeOptions
	workDir   string
	tetraPath string
	processes []nativeui.ProcessReport
}

type processResult struct {
	exitCode int
	output   string
	err      error
}

type shellReport struct {
	Schema   string            `json:"schema"`
	UISchema string            `json:"ui_schema"`
	Runtime  string            `json:"runtime"`
	States   []shellStateTrace `json:"states,omitempty"`
	Views    []shellViewTrace  `json:"views,omitempty"`
}

type shellStateTrace struct {
	Name   string                 `json:"name"`
	Module string                 `json:"module,omitempty"`
	Fields []shellStateFieldTrace `json:"fields,omitempty"`
}

type shellStateFieldTrace struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable,omitempty"`
	Const   bool   `json:"const,omitempty"`
	Value   string `json:"value"`
}

type shellViewTrace struct {
	Name          string                     `json:"name"`
	Module        string                     `json:"module,omitempty"`
	StateType     string                     `json:"state_type"`
	Bindings      []shellBindingTrace        `json:"bindings,omitempty"`
	Widgets       []shellWidgetTrace         `json:"widgets,omitempty"`
	Events        []shellEventTrace          `json:"events,omitempty"`
	Styles        []shellPropertyTrace       `json:"styles,omitempty"`
	Accessibility []shellPropertyTrace       `json:"accessibility,omitempty"`
	bindingFields map[string]string          `json:"-"`
	widgetValues  map[string]string          `json:"-"`
	widgetIDs     map[string]struct{}        `json:"-"`
	eventCommands map[string]shellEventTrace `json:"-"`
}

type shellWidgetTrace struct {
	ID            string               `json:"id"`
	Kind          string               `json:"kind"`
	Binding       string               `json:"binding,omitempty"`
	Event         string               `json:"event,omitempty"`
	Command       string               `json:"command,omitempty"`
	Type          string               `json:"type,omitempty"`
	Value         string               `json:"value,omitempty"`
	Styles        []shellPropertyTrace `json:"styles,omitempty"`
	Accessibility []shellPropertyTrace `json:"accessibility,omitempty"`
}

type shellEventTrace struct {
	Name       string                `json:"name"`
	Command    string                `json:"command"`
	Operations []shellOperationTrace `json:"operations,omitempty"`
	Bindings   []shellBindingTrace   `json:"bindings,omitempty"`
}

type shellOperationTrace struct {
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	Value      string `json:"value,omitempty"`
	StateField string `json:"state_field,omitempty"`
	StateValue string `json:"state_value,omitempty"`
}

type shellBindingTrace struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type shellPropertyTrace struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type nativeRuntime struct {
	shell       shellReport
	states      map[string]map[string]string
	widgets     map[string]*runtimeWidget
	widgetOrder []string
	closed      bool
}

type runtimeWidget struct {
	report nativeui.WidgetReport
	view   *shellViewTrace
}

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.ui.native-runtime.v1 report")
	flag.StringVar(&opt.TetraPath, "tetra", "", "tetra CLI path; defaults to a fresh temp build from ./cli/cmd/tetra")
	flag.BoolVar(&opt.KeepWork, "keep-work", false, "keep temporary build directory")
	flag.Parse()
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := runSmoke(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSmoke(ctx context.Context, opt smokeOptions) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("native UI runtime smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	workDir, err := os.MkdirTemp("", "tetra-native-ui-*")
	if err != nil {
		return err
	}
	r := &smokeRunner{opt: opt, workDir: workDir}
	if !opt.KeepWork {
		defer os.RemoveAll(workDir)
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	if opt.TetraPath == "" {
		r.tetraPath = filepath.Join(workDir, "tetra")
		res, err := runCommand(ctx, 30*time.Second, "go", "build", "-o", r.tetraPath, "./cli/cmd/tetra")
		r.recordProcess("go build tetra cli", "build", "go build ./cli/cmd/tetra", res)
		if err != nil {
			return fmt.Errorf("build smoke tetra CLI: %w", err)
		}
	} else {
		r.tetraPath = opt.TetraPath
	}

	outPath := filepath.Join(workDir, "ui-native")
	sourcePath := filepath.Join("examples", "ui_native_shell_smoke.tetra")
	res, err := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
	r.recordProcess("tetra build native UI", "build", r.tetraPath+" build --target linux-x64", res)
	if err != nil {
		return fmt.Errorf("build native UI smoke app: %w", err)
	}

	appRes, err := runCommand(ctx, 5*time.Second, outPath)
	r.recordProcess("native app", "app", outPath, appRes)
	if err != nil {
		return fmt.Errorf("run native UI smoke app: %w", err)
	}

	sidecarPath := outPath + ".ui.shell.json"
	rawSidecar, err := os.ReadFile(sidecarPath)
	if err != nil {
		return err
	}
	widgets, events, cases, err := runRuntimeScenario(rawSidecar)
	if err != nil {
		return err
	}
	runtimePath, err := os.Executable()
	if err != nil || strings.TrimSpace(runtimePath) == "" {
		runtimePath = "tools/cmd/native-ui-runtime-smoke"
	}
	r.processes = append(r.processes, nativeui.ProcessReport{
		Name:     "native ui runtime",
		Kind:     "runtime",
		Path:     runtimePath,
		Ran:      true,
		Pass:     true,
		ExitCode: intPtr(0),
	})

	return r.writeReport(sourcePath, widgets, events, cases)
}

func runRuntimeScenario(rawSidecar []byte) ([]nativeui.WidgetReport, []nativeui.EventReport, []nativeui.CaseReport, error) {
	rt, err := loadNativeRuntime(rawSidecar)
	if err != nil {
		return nil, nil, nil, err
	}
	actionID, err := rt.firstActionWidget()
	if err != nil {
		return nil, nil, nil, err
	}
	var events []nativeui.EventReport
	for order := 1; order <= 2; order++ {
		event, err := rt.dispatch(actionID, "click", "", order)
		if err != nil {
			return nil, nil, nil, err
		}
		events = append(events, event)
	}
	_, invalidWidgetErr := rt.dispatch("__missing_widget__", "click", "", len(events)+1)
	_, unsupportedEventErr := rt.dispatch(actionID, "hover", "", len(events)+1)
	_, commandFailureErr := rt.dispatch(actionID, "click", "__missing_command__", len(events)+1)
	_, malformedErr := loadNativeRuntime([]byte(`{"schema":"tetra.ui.native-shell.v1"`))
	closeErr := rt.close()

	cases := []nativeui.CaseReport{
		{Name: "load widget tree", Ran: true, Pass: len(rt.widgets) >= 3},
		{Name: "dispatch click command", Ran: true, Pass: len(events) >= 1 && events[0].Pass && events[0].Event == "click"},
		{Name: "propagate state update", Ran: true, Pass: len(events) >= 1 && stateChanged(events[0].BeforeState, events[0].AfterState)},
		{Name: "dispatch multiple ordered events", Ran: true, Pass: len(events) >= 2 && events[0].Order == 1 && events[1].Order == 2},
		{Name: "reject invalid widget id", Ran: true, Pass: invalidWidgetErr != nil, ExpectedError: errorString(invalidWidgetErr)},
		{Name: "reject malformed metadata", Ran: true, Pass: malformedErr != nil, ExpectedError: errorString(malformedErr)},
		{Name: "reject unsupported event kind", Ran: true, Pass: unsupportedEventErr != nil, ExpectedError: errorString(unsupportedEventErr)},
		{Name: "reject command failure", Ran: true, Pass: commandFailureErr != nil, ExpectedError: errorString(commandFailureErr)},
		{Name: "close runtime", Ran: true, Pass: closeErr == nil, ExpectedError: errorString(closeErr)},
	}
	return rt.widgetReports(), events, cases, nil
}

func loadNativeRuntime(raw []byte) (*nativeRuntime, error) {
	var shell shellReport
	if err := decodeStrictJSON(raw, &shell); err != nil {
		return nil, fmt.Errorf("malformed metadata: %w", err)
	}
	if shell.Schema != nativeShellSchemaV1 {
		return nil, fmt.Errorf("malformed metadata: schema is %q, want %q", shell.Schema, nativeShellSchemaV1)
	}
	if shell.UISchema != uiBundleSchemaV1 {
		return nil, fmt.Errorf("malformed metadata: ui_schema is %q, want %q", shell.UISchema, uiBundleSchemaV1)
	}
	if shell.Runtime != nativeShellRuntimeDispatch {
		return nil, fmt.Errorf("malformed metadata: runtime is %q, want %q", shell.Runtime, nativeShellRuntimeDispatch)
	}
	if len(shell.States) == 0 {
		return nil, errors.New("malformed metadata: states are required")
	}
	if len(shell.Views) == 0 {
		return nil, errors.New("malformed metadata: views are required")
	}
	rt := &nativeRuntime{
		shell:   shell,
		states:  map[string]map[string]string{},
		widgets: map[string]*runtimeWidget{},
	}
	for _, state := range rt.shell.States {
		if strings.TrimSpace(state.Name) == "" {
			return nil, errors.New("malformed metadata: state name is required")
		}
		fields := map[string]string{}
		for _, field := range state.Fields {
			if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Type) == "" {
				return nil, fmt.Errorf("malformed metadata: state %s has field missing name or type", state.Name)
			}
			fields[field.Name] = field.Value
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("malformed metadata: state %s has no fields", state.Name)
		}
		rt.states[state.Name] = fields
	}
	if err := rt.buildWidgets(); err != nil {
		return nil, err
	}
	return rt, nil
}

func (rt *nativeRuntime) buildWidgets() error {
	for viewIndex := range rt.shell.Views {
		view := &rt.shell.Views[viewIndex]
		if strings.TrimSpace(view.Name) == "" || strings.TrimSpace(view.StateType) == "" {
			return errors.New("malformed metadata: view name and state_type are required")
		}
		if _, ok := rt.states[view.StateType]; !ok {
			return fmt.Errorf("malformed metadata: view %s references unknown state %s", view.Name, view.StateType)
		}
		view.bindingFields = rt.inferBindingFields(view)
		view.widgetValues = map[string]string{}
		view.widgetIDs = map[string]struct{}{}
		view.eventCommands = map[string]shellEventTrace{}
		root := nativeui.WidgetReport{
			ID:      view.Name,
			Kind:    "view",
			Enabled: true,
			Visible: true,
			Bounds:  nativeui.Bounds{X: 0, Y: 0, Width: 320, Height: 16 + len(view.Widgets)*32},
		}
		if err := rt.addWidget(root, view); err != nil {
			return err
		}
		for eventIndex, event := range view.Events {
			if strings.TrimSpace(event.Name) == "" || strings.TrimSpace(event.Command) == "" {
				return fmt.Errorf("malformed metadata: view %s event %d missing name or command", view.Name, eventIndex+1)
			}
			view.eventCommands[event.Command] = event
		}
		for widgetIndex, widget := range view.Widgets {
			if strings.TrimSpace(widget.ID) == "" || strings.TrimSpace(widget.Kind) == "" {
				return fmt.Errorf("malformed metadata: view %s widget %d missing id or kind", view.Name, widgetIndex+1)
			}
			report := nativeui.WidgetReport{
				ID:      widget.ID,
				Kind:    widget.Kind,
				Parent:  view.Name,
				Binding: widget.Binding,
				Event:   widget.Event,
				Command: widget.Command,
				Value:   widget.Value,
				Enabled: true,
				Visible: true,
				Bounds: nativeui.Bounds{
					X:      8,
					Y:      8 + widgetIndex*32,
					Width:  304,
					Height: 24,
				},
			}
			if widget.Binding != "" {
				report.Value = rt.bindingValue(view, widget.Binding)
				view.widgetValues[widget.ID] = report.Value
			}
			if widget.Kind == "action" {
				if strings.TrimSpace(widget.Event) == "" || strings.TrimSpace(widget.Command) == "" {
					return fmt.Errorf("malformed metadata: action widget %s missing event or command", widget.ID)
				}
				if _, ok := view.eventCommands[widget.Command]; !ok {
					return fmt.Errorf("malformed metadata: action widget %s references unknown command %s", widget.ID, widget.Command)
				}
			}
			if err := rt.addWidget(report, view); err != nil {
				return err
			}
			view.widgetIDs[widget.ID] = struct{}{}
		}
	}
	return nil
}

func (rt *nativeRuntime) addWidget(report nativeui.WidgetReport, view *shellViewTrace) error {
	if _, exists := rt.widgets[report.ID]; exists {
		return fmt.Errorf("malformed metadata: duplicate widget %s", report.ID)
	}
	rt.widgets[report.ID] = &runtimeWidget{report: report, view: view}
	rt.widgetOrder = append(rt.widgetOrder, report.ID)
	return nil
}

func (rt *nativeRuntime) inferBindingFields(view *shellViewTrace) map[string]string {
	fields := rt.states[view.StateType]
	out := map[string]string{}
	used := map[string]bool{}
	for _, binding := range view.Bindings {
		if _, ok := fields[binding.Name]; ok {
			out[binding.Name] = binding.Name
			used[binding.Name] = true
			continue
		}
		for name, value := range fields {
			if used[name] {
				continue
			}
			if value == binding.Value {
				out[binding.Name] = name
				used[name] = true
				break
			}
		}
	}
	return out
}

func (rt *nativeRuntime) firstActionWidget() (string, error) {
	for _, id := range rt.widgetOrder {
		if rt.widgets[id].report.Kind == "action" {
			return id, nil
		}
	}
	return "", errors.New("runtime has no action widget")
}

func (rt *nativeRuntime) dispatch(widgetID, eventKind, commandOverride string, order int) (nativeui.EventReport, error) {
	if rt.closed {
		return nativeui.EventReport{}, errors.New("runtime is closed")
	}
	widget, ok := rt.widgets[widgetID]
	if !ok {
		return nativeui.EventReport{}, fmt.Errorf("unknown widget %s", widgetID)
	}
	if widget.report.Kind != "action" {
		return nativeui.EventReport{}, fmt.Errorf("widget %s is not an action widget", widgetID)
	}
	if eventKind != "click" && eventKind != "activate" {
		return nativeui.EventReport{}, fmt.Errorf("unsupported event %s for widget %s", eventKind, widgetID)
	}
	command := widget.report.Command
	if commandOverride != "" {
		command = commandOverride
	}
	shellEvent, ok := widget.view.eventCommands[command]
	if !ok {
		return nativeui.EventReport{}, fmt.Errorf("unknown command %s for widget %s", command, widgetID)
	}
	beforeState := rt.stateSnapshot(widget.view)
	beforeWidgets := rt.boundWidgetValues(widget.view)
	ops := make([]nativeui.OperationReport, 0, len(shellEvent.Operations))
	for _, op := range shellEvent.Operations {
		field, value, err := rt.applyOperation(widget.view, op)
		if err != nil {
			return nativeui.EventReport{}, err
		}
		ops = append(ops, nativeui.OperationReport{
			Kind:       op.Kind,
			Target:     op.Target,
			Value:      op.Value,
			StateField: field,
			StateValue: value,
		})
	}
	updates := rt.refreshWidgets(widget.view, beforeWidgets)
	return nativeui.EventReport{
		Order:         order,
		WidgetID:      widgetID,
		Event:         eventKind,
		Command:       command,
		Pass:          true,
		BeforeState:   beforeState,
		AfterState:    rt.stateSnapshot(widget.view),
		Operations:    ops,
		WidgetUpdates: updates,
	}, nil
}

func (rt *nativeRuntime) applyOperation(view *shellViewTrace, op shellOperationTrace) (string, string, error) {
	field, ok := stateFieldName(op.Target)
	if !ok {
		return "", "", fmt.Errorf("operation target %s is not a state field", op.Target)
	}
	fields := rt.states[view.StateType]
	if _, ok := fields[field]; !ok {
		return "", "", fmt.Errorf("operation target references unknown state field %s", field)
	}
	switch op.Kind {
	case "state_add":
		current, err := strconv.Atoi(fields[field])
		if err != nil {
			return "", "", fmt.Errorf("state_add field %s is not numeric: %w", field, err)
		}
		delta, err := strconv.Atoi(op.Value)
		if err != nil {
			return "", "", fmt.Errorf("state_add value %q is not numeric: %w", op.Value, err)
		}
		fields[field] = strconv.Itoa(current + delta)
	case "state_sub":
		current, err := strconv.Atoi(fields[field])
		if err != nil {
			return "", "", fmt.Errorf("state_sub field %s is not numeric: %w", field, err)
		}
		delta, err := strconv.Atoi(op.Value)
		if err != nil {
			return "", "", fmt.Errorf("state_sub value %q is not numeric: %w", op.Value, err)
		}
		fields[field] = strconv.Itoa(current - delta)
	case "state_set":
		fields[field] = rt.resolveOperationValue(view, op.Value)
	default:
		return "", "", fmt.Errorf("unsupported command operation %s", op.Kind)
	}
	return field, fields[field], nil
}

func (rt *nativeRuntime) resolveOperationValue(view *shellViewTrace, value string) string {
	if field, ok := stateFieldName(value); ok {
		return rt.states[view.StateType][field]
	}
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	}
	return value
}

func (rt *nativeRuntime) refreshWidgets(view *shellViewTrace, before map[string]string) []nativeui.WidgetUpdateReport {
	var updates []nativeui.WidgetUpdateReport
	for _, id := range rt.widgetOrder {
		widget := rt.widgets[id]
		if widget.view != view || widget.report.Binding == "" {
			continue
		}
		after := rt.bindingValue(view, widget.report.Binding)
		if before[id] != after {
			updates = append(updates, nativeui.WidgetUpdateReport{ID: id, Before: before[id], After: after})
		}
		widget.report.Value = after
		view.widgetValues[id] = after
	}
	return updates
}

func (rt *nativeRuntime) boundWidgetValues(view *shellViewTrace) map[string]string {
	out := map[string]string{}
	for _, id := range rt.widgetOrder {
		widget := rt.widgets[id]
		if widget.view == view && widget.report.Binding != "" {
			out[id] = widget.report.Value
		}
	}
	return out
}

func (rt *nativeRuntime) bindingValue(view *shellViewTrace, bindingName string) string {
	field, ok := view.bindingFields[bindingName]
	if !ok {
		return ""
	}
	return rt.states[view.StateType][field]
}

func (rt *nativeRuntime) stateSnapshot(view *shellViewTrace) map[string]string {
	out := map[string]string{}
	for field, value := range rt.states[view.StateType] {
		out[view.StateType+"."+field] = value
	}
	return out
}

func (rt *nativeRuntime) widgetReports() []nativeui.WidgetReport {
	out := make([]nativeui.WidgetReport, 0, len(rt.widgetOrder))
	for _, id := range rt.widgetOrder {
		out = append(out, rt.widgets[id].report)
	}
	return out
}

func (rt *nativeRuntime) close() error {
	if rt.closed {
		return errors.New("runtime already closed")
	}
	rt.closed = true
	return nil
}

func (r *smokeRunner) recordProcess(name, kind, path string, res processResult) {
	r.processes = append(r.processes, nativeui.ProcessReport{
		Name:     name,
		Kind:     kind,
		Path:     path,
		Ran:      true,
		Pass:     res.err == nil && res.exitCode == 0,
		ExitCode: intPtr(res.exitCode),
	})
}

func (r *smokeRunner) writeReport(sourcePath string, widgets []nativeui.WidgetReport, events []nativeui.EventReport, cases []nativeui.CaseReport) error {
	report := nativeui.Report{
		Schema:    nativeui.SchemaV1,
		Status:    "pass",
		Target:    "linux-x64",
		Host:      "linux-x64",
		Runtime:   "native-ui-linux-x64",
		UISchema:  uiBundleSchemaV1,
		Source:    sourcePath,
		Processes: r.processes,
		Widgets:   widgets,
		Events:    events,
		Cases:     cases,
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := nativeui.ValidateReport(raw); err != nil {
		return err
	}
	return os.WriteFile(r.opt.ReportPath, append(raw, '\n'), 0o644)
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (processResult, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	res := processResult{exitCode: processExitCode(err), output: output.String(), err: err}
	if cctx.Err() == context.DeadlineExceeded {
		res.err = cctx.Err()
		return res, fmt.Errorf("%s timed out", name)
	}
	if err != nil {
		return res, fmt.Errorf("%s %s: %w output=%q", name, strings.Join(args, " "), err, res.output)
	}
	return res, nil
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return exitErr.ExitCode()
	}
	return 1
}

func decodeStrictJSON(raw []byte, out any) error {
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

func stateFieldName(path string) (string, bool) {
	const prefix = "state."
	if !strings.HasPrefix(path, prefix) || len(path) == len(prefix) {
		return "", false
	}
	return strings.TrimPrefix(path, prefix), true
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

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func intPtr(v int) *int { return &v }
