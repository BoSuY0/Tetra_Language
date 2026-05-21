package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const webUISmokeSchema = "tetra.web-ui-smoke.v1alpha1"
const uiBundleSchema = "tetra.ui.v1"
const uiEventDispatchBoundaryTrace = "ui-event-dispatch:web-command-dispatch"
const uiBundleSchemaArtifactID = "tetra.ui.v1.schema.json"
const defaultUIBundleSchemaArtifactPath = "docs/schemas/tetra.ui.v1.schema.json"

type webUISmokeReport struct {
	Schema             string `json:"schema"`
	GeneratedAt        string `json:"generated_at"`
	Target             string `json:"target"`
	UIScopeActive      bool   `json:"ui_scope_active"`
	Source             string `json:"source"`
	UsedFallbackSource bool   `json:"used_fallback_source"`
	Automation         string `json:"automation"`
	Status             string `json:"status"`
	Result             string `json:"result"`
	RuntimeTrace       string `json:"runtime_trace"`
	Blocker            string `json:"blocker"`
	DOMSnapshot        string `json:"dom_snapshot"`
	ChromiumStderr     string `json:"chromium_stderr"`
	UISchema           string `json:"ui_schema"`
	UIBundlePath       string `json:"ui_bundle_path"`
	UIModulePath       string `json:"ui_module_path"`
}

type uiBundleArtifact struct {
	Schema string          `json:"schema"`
	States []uiBundleState `json:"states"`
	Views  []uiBundleView  `json:"views"`
}

type uiBundleState struct {
	Name   string               `json:"name"`
	Module string               `json:"module"`
	Fields []uiBundleStateField `json:"fields"`
}

type uiBundleStateField struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable"`
	Const   bool   `json:"const"`
	Init    string `json:"init"`
}

type uiBundleView struct {
	Name          string                  `json:"name"`
	Module        string                  `json:"module"`
	StateType     string                  `json:"state_type"`
	Bindings      []uiBundleBinding       `json:"bindings"`
	Events        []uiBundleEvent         `json:"events"`
	Commands      []uiBundleCommand       `json:"commands"`
	Styles        []uiBundleStyle         `json:"styles"`
	Accessibility []uiBundleAccessibility `json:"accessibility"`
}

type uiBundleBinding struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type uiBundleEvent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type uiBundleCommand struct {
	Name           string                     `json:"name"`
	StatementCount int                        `json:"statement_count"`
	Operations     []uiBundleCommandOperation `json:"operations,omitempty"`
}

type uiBundleCommandOperation struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Value  string `json:"value,omitempty"`
}

type uiBundleStyle struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type uiBundleAccessibility struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type uiBundleSchemaArtifact struct {
	JSONSchema           string                     `json:"$schema"`
	ID                   string                     `json:"$id"`
	Title                string                     `json:"title"`
	Type                 string                     `json:"type"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Defs                 map[string]json.RawMessage `json:"$defs"`
}

func main() {
	var reportPath string
	var uiSchemaArtifactPath string
	flag.StringVar(&reportPath, "report", "", "path to web UI smoke JSON report")
	flag.StringVar(&uiSchemaArtifactPath, "ui-schema-artifact", defaultUIBundleSchemaArtifactPath, "path to tetra.ui.v1 JSON Schema artifact")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateUIBundleSchemaArtifact(resolveUIBundleSchemaArtifactPath(uiSchemaArtifactPath)); err != nil {
		fmt.Fprintf(os.Stderr, "web UI smoke UI metadata schema artifact invalid: %v\n", err)
		os.Exit(1)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var report webUISmokeReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateWebUISmokeReport(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateWebUISmokeReport(report webUISmokeReport) error {
	if report.Schema != webUISmokeSchema {
		return fmt.Errorf("web UI smoke schema = %q, want %q", report.Schema, webUISmokeSchema)
	}
	if report.GeneratedAt == "" {
		return fmt.Errorf("web UI smoke missing generated_at")
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		return fmt.Errorf("web UI smoke generated_at is not RFC3339: %w", err)
	}
	if report.Target != "wasm32-web" {
		return fmt.Errorf("web UI smoke target = %q, want wasm32-web", report.Target)
	}
	if report.Source == "" || !strings.HasSuffix(report.Source, ".tetra") {
		return fmt.Errorf("web UI smoke source must be a .tetra file")
	}
	if report.Automation == "" {
		return fmt.Errorf("web UI smoke missing automation")
	}
	if report.UISchema != "" && report.UISchema != uiBundleSchema {
		return fmt.Errorf("web UI smoke ui_schema = %q, want %q", report.UISchema, uiBundleSchema)
	}
	switch report.Status {
	case "pass":
		if report.UsedFallbackSource {
			return fmt.Errorf("web UI smoke pass cannot use fallback source")
		}
		if !report.UIScopeActive {
			return fmt.Errorf("web UI smoke pass cannot use inactive UI scope")
		}
		if !strings.HasPrefix(report.Result, "ok:") {
			return fmt.Errorf("web UI smoke pass result must start with ok:")
		}
		if err := validateRuntimeTrace(report.RuntimeTrace); err != nil {
			return err
		}
		if report.UISchema != uiBundleSchema {
			return fmt.Errorf("web UI smoke pass ui_schema = %q, want %q", report.UISchema, uiBundleSchema)
		}
		if report.UIBundlePath == "" || !strings.HasSuffix(report.UIBundlePath, ".ui.json") {
			return fmt.Errorf("web UI smoke pass must include ui_bundle_path ending with .ui.json")
		}
		if err := requireRegularFile(report.UIBundlePath, "ui_bundle_path"); err != nil {
			return err
		}
		uiBundle, err := validateUIBundleArtifact(report.UIBundlePath)
		if err != nil {
			return err
		}
		if report.UIModulePath == "" || !strings.HasSuffix(report.UIModulePath, ".ui.web.mjs") {
			return fmt.Errorf("web UI smoke pass must include ui_module_path ending with .ui.web.mjs")
		}
		if err := requireRegularFile(report.UIModulePath, "ui_module_path"); err != nil {
			return err
		}
		if report.DOMSnapshot == "" || !strings.HasSuffix(report.DOMSnapshot, ".html") {
			return fmt.Errorf("web UI smoke pass must include dom_snapshot ending with .html")
		}
		if err := requireRegularFile(report.DOMSnapshot, "dom_snapshot"); err != nil {
			return err
		}
		if err := validateDOMSnapshotArtifact(report.DOMSnapshot, uiBundle); err != nil {
			return err
		}
		if report.Blocker != "" {
			return fmt.Errorf("web UI smoke pass cannot include blocker")
		}
	case "blocked":
		if report.Blocker == "" {
			return fmt.Errorf("web UI smoke blocked report missing blocker")
		}
	case "fail":
		if report.Blocker == "" {
			return fmt.Errorf("web UI smoke failure missing blocker")
		}
	default:
		return fmt.Errorf("web UI smoke status = %q, want pass, blocked, or fail", report.Status)
	}
	return nil
}

func validateRuntimeTrace(trace string) error {
	if trace == "" {
		return fmt.Errorf("web UI smoke pass missing runtime_trace")
	}
	for _, marker := range []string{
		"window-mount:ok",
		"root-mount:ok",
		"layout:ok",
		"text:ok",
		"button:ok",
		"input:ok",
		"list:ok",
		"panel:ok",
		"focus:ok",
		"input-event:ok",
		"change:ok",
		"select:ok",
		"click:ok",
		"timer:ok",
		"async-command:ok",
		"redraw-update:ok",
		"error-recovery:ok",
		"main-exit:ok",
		"stdout:ok",
		"nonzero-exit:ok",
		"failure-propagation:ok",
		"repeated-instantiation:ok",
		uiEventDispatchBoundaryTrace,
	} {
		if !strings.Contains(trace, marker) {
			return fmt.Errorf("web UI smoke runtime_trace missing %q", marker)
		}
	}
	return nil
}

func resolveUIBundleSchemaArtifactPath(path string) string {
	if path != defaultUIBundleSchemaArtifactPath {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	dir, err := os.Getwd()
	if err != nil {
		return path
	}
	for {
		candidate := filepath.Join(dir, defaultUIBundleSchemaArtifactPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return path
		}
		dir = parent
	}
}

func validateUIBundleSchemaArtifact(path string) error {
	if path == "" {
		return fmt.Errorf("missing UI metadata schema artifact path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var schema uiBundleSchemaArtifact
	if err := decodeStrictJSON(raw, &schema); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if schema.JSONSchema == "" {
		return fmt.Errorf("%s missing $schema", path)
	}
	if schema.ID != uiBundleSchemaArtifactID {
		return fmt.Errorf("%s $id = %q, want %q", path, schema.ID, uiBundleSchemaArtifactID)
	}
	if schema.Type != "object" {
		return fmt.Errorf("%s root type = %q, want object", path, schema.Type)
	}
	if schema.AdditionalProperties {
		return fmt.Errorf("%s root additionalProperties must be false", path)
	}
	for _, required := range []string{"schema", "states", "views"} {
		if !containsString(schema.Required, required) {
			return fmt.Errorf("%s missing required field %q", path, required)
		}
		if _, ok := schema.Properties[required]; !ok {
			return fmt.Errorf("%s missing property contract for %q", path, required)
		}
	}
	for _, def := range []string{"state", "stateField", "view", "binding", "event", "command", "commandOperation", "typedValue"} {
		if _, ok := schema.Defs[def]; !ok {
			return fmt.Errorf("%s missing $defs.%s", path, def)
		}
	}
	return nil
}

func requireRegularFile(path string, field string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("web UI smoke pass %s must point to an existing artifact: %w", field, err)
	}
	if info.IsDir() {
		return fmt.Errorf("web UI smoke pass %s points to a directory, want file", field)
	}
	return nil
}

func validateUIBundleArtifact(path string) (uiBundleArtifact, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return uiBundleArtifact{}, fmt.Errorf("web UI smoke pass cannot read ui_bundle_path: %w", err)
	}
	var bundle uiBundleArtifact
	if err := decodeStrictJSON(raw, &bundle); err != nil {
		return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui_bundle_path is not strict tetra.ui.v1 metadata: %w", err)
	}
	if bundle.Schema != uiBundleSchema {
		return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle schema = %q, want %q", bundle.Schema, uiBundleSchema)
	}
	if len(bundle.Views) == 0 {
		return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle must include at least one view")
	}
	stateNames := map[string]bool{}
	for _, state := range bundle.States {
		if state.Name == "" {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle state missing name")
		}
		if state.Module == "" && strings.Contains(state.Name, ".") {
			parts := strings.Split(state.Name, ".")
			state.Module = strings.Join(parts[:len(parts)-1], ".")
			state.Name = parts[len(parts)-1]
		}
		if stateNames[state.Name] {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle duplicate state %s", state.Name)
		}
		for _, field := range state.Fields {
			if field.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle state %s has field missing name", state.Name)
			}
			if field.Type == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle state %s field %s missing type", state.Name, field.Name)
			}
			if field.Init == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle state %s field %s missing init", state.Name, field.Name)
			}
		}
		stateNames[state.Name] = true
	}
	for _, view := range bundle.Views {
		if view.Name == "" {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view missing name")
		}
		if view.Module == "" && strings.Contains(view.Name, ".") {
			parts := strings.Split(view.Name, ".")
			view.Module = strings.Join(parts[:len(parts)-1], ".")
			view.Name = parts[len(parts)-1]
		}
		if view.StateType == "" {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s missing state_type", view.Name)
		}
		if strings.Contains(view.StateType, ".") {
			parts := strings.Split(view.StateType, ".")
			view.StateType = parts[len(parts)-1]
		}
		if len(stateNames) > 0 && !stateNames[view.StateType] {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s references unknown state_type %s", view.Name, view.StateType)
		}
		seenBindings := map[string]bool{}
		for _, binding := range view.Bindings {
			if binding.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s has binding missing name", view.Name)
			}
			if binding.Type == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s binding %s missing type", view.Name, binding.Name)
			}
			if binding.Source == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s binding %s missing source", view.Name, binding.Name)
			}
			if seenBindings[binding.Name] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s duplicate binding %s", view.Name, binding.Name)
			}
			seenBindings[binding.Name] = true
		}
		seenEvents := map[string]bool{}
		for _, event := range view.Events {
			if event.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s has event missing name", view.Name)
			}
			if event.Command == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s event %s missing command", view.Name, event.Name)
			}
			if seenEvents[event.Name] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s duplicate event %s", view.Name, event.Name)
			}
			seenEvents[event.Name] = true
		}
		seenCommands := map[string]bool{}
		for _, command := range view.Commands {
			if command.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s has command missing name", view.Name)
			}
			if command.StatementCount < 0 {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s command %s has negative statement_count", view.Name, command.Name)
			}
			for _, op := range command.Operations {
				switch op.Kind {
				case "state_add":
					if op.Value == "" {
						return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s command %s state_add operation missing value", view.Name, command.Name)
					}
				case "state_set":
				default:
					return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s command %s has unsupported operation kind %q", view.Name, command.Name, op.Kind)
				}
				if !strings.HasPrefix(op.Target, "state.") || len(op.Target) == len("state.") {
					return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s command %s has invalid operation target %q", view.Name, command.Name, op.Target)
				}
			}
			if seenCommands[command.Name] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s duplicate command %s", view.Name, command.Name)
			}
			seenCommands[command.Name] = true
		}
		for _, event := range view.Events {
			if !seenCommands[event.Command] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s event %s references unknown command %s", view.Name, event.Name, event.Command)
			}
		}
		seenStyles := map[string]bool{}
		for _, style := range view.Styles {
			if style.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s has style missing name", view.Name)
			}
			if style.Type == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s style %s missing type", view.Name, style.Name)
			}
			if style.Value == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s style %s missing value", view.Name, style.Name)
			}
			if seenStyles[style.Name] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s duplicate style %s", view.Name, style.Name)
			}
			seenStyles[style.Name] = true
		}
		if len(view.Accessibility) == 0 {
			return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s missing accessibility metadata", view.Name)
		}
		seenA11y := map[string]bool{}
		for _, a11y := range view.Accessibility {
			if a11y.Name == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s has accessibility entry missing name", view.Name)
			}
			if a11y.Type == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s accessibility %s missing type", view.Name, a11y.Name)
			}
			if a11y.Value == "" {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s accessibility %s missing value", view.Name, a11y.Name)
			}
			if seenA11y[a11y.Name] {
				return uiBundleArtifact{}, fmt.Errorf("web UI smoke ui bundle view %s duplicate accessibility metadata %s", view.Name, a11y.Name)
			}
			seenA11y[a11y.Name] = true
		}
	}
	return bundle, nil
}

func validateDOMSnapshotArtifact(path string, bundle uiBundleArtifact) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("web UI smoke pass cannot read dom_snapshot: %w", err)
	}
	dom := string(raw)
	text := html.UnescapeString(dom)
	if !hasDOMAttribute(dom, "data-tetra-ui", "v1") {
		return fmt.Errorf(`web UI smoke dom_snapshot missing mounted UI marker data-tetra-ui="v1"`)
	}
	for _, marker := range []string{
		"Tetra UI Shell",
		"runtime: web command dispatch",
	} {
		if !strings.Contains(text, marker) {
			return fmt.Errorf("web UI smoke dom_snapshot missing mounted UI text %q", marker)
		}
	}
	for _, view := range bundle.Views {
		viewMarker := fmt.Sprintf("view %s (state: %s)", view.Name, view.StateType)
		if !strings.Contains(text, viewMarker) {
			return fmt.Errorf("web UI smoke dom_snapshot missing view/state marker %q", viewMarker)
		}
		for _, binding := range view.Bindings {
			if !hasDOMAttribute(dom, "data-tetra-binding", binding.Name) {
				return fmt.Errorf(`web UI smoke dom_snapshot missing binding attribute data-tetra-binding=%q`, binding.Name)
			}
			bindingMarker := fmt.Sprintf("bind %s: %s =", binding.Name, binding.Type)
			if !strings.Contains(text, bindingMarker) {
				return fmt.Errorf("web UI smoke dom_snapshot missing hydrated binding marker %q", bindingMarker)
			}
		}
		for _, event := range view.Events {
			eventMarker := fmt.Sprintf("event %s -> %s", event.Name, event.Command)
			if !strings.Contains(text, eventMarker) {
				return fmt.Errorf("web UI smoke dom_snapshot missing event marker %q", eventMarker)
			}
		}
	}
	return nil
}

func hasDOMAttribute(dom string, name string, value string) bool {
	for _, marker := range []string{
		name + `="` + value + `"`,
		name + `='` + value + `'`,
		name + `=` + value,
	} {
		if strings.Contains(dom, marker) {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON payload")
		}
		return err
	}
	return nil
}
