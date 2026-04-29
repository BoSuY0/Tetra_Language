package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const webUISmokeSchema = "tetra.web-ui-smoke.v1alpha1"
const uiBundleSchema = "tetra.ui.v1"

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
	Name           string `json:"name"`
	StatementCount int    `json:"statement_count"`
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

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to web UI smoke JSON report")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
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
		if report.UISchema != uiBundleSchema {
			return fmt.Errorf("web UI smoke pass ui_schema = %q, want %q", report.UISchema, uiBundleSchema)
		}
		if report.UIBundlePath == "" || !strings.HasSuffix(report.UIBundlePath, ".ui.json") {
			return fmt.Errorf("web UI smoke pass must include ui_bundle_path ending with .ui.json")
		}
		if err := requireRegularFile(report.UIBundlePath, "ui_bundle_path"); err != nil {
			return err
		}
		if err := validateUIBundleArtifact(report.UIBundlePath); err != nil {
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

func validateUIBundleArtifact(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("web UI smoke pass cannot read ui_bundle_path: %w", err)
	}
	var bundle uiBundleArtifact
	if err := decodeStrictJSON(raw, &bundle); err != nil {
		return fmt.Errorf("web UI smoke ui_bundle_path is not strict tetra.ui.v1 metadata: %w", err)
	}
	if bundle.Schema != uiBundleSchema {
		return fmt.Errorf("web UI smoke ui bundle schema = %q, want %q", bundle.Schema, uiBundleSchema)
	}
	if len(bundle.Views) == 0 {
		return fmt.Errorf("web UI smoke ui bundle must include at least one view")
	}
	stateNames := map[string]bool{}
	for _, state := range bundle.States {
		if state.Name == "" {
			return fmt.Errorf("web UI smoke ui bundle state missing name")
		}
		if state.Module == "" && strings.Contains(state.Name, ".") {
			parts := strings.Split(state.Name, ".")
			state.Module = strings.Join(parts[:len(parts)-1], ".")
			state.Name = parts[len(parts)-1]
		}
		if stateNames[state.Name] {
			return fmt.Errorf("web UI smoke ui bundle duplicate state %s", state.Name)
		}
		stateNames[state.Name] = true
	}
	for _, view := range bundle.Views {
		if view.Name == "" {
			return fmt.Errorf("web UI smoke ui bundle view missing name")
		}
		if view.Module == "" && strings.Contains(view.Name, ".") {
			parts := strings.Split(view.Name, ".")
			view.Module = strings.Join(parts[:len(parts)-1], ".")
			view.Name = parts[len(parts)-1]
		}
		if view.StateType == "" {
			return fmt.Errorf("web UI smoke ui bundle view %s missing state_type", view.Name)
		}
		if strings.Contains(view.StateType, ".") {
			parts := strings.Split(view.StateType, ".")
			view.StateType = parts[len(parts)-1]
		}
		if len(stateNames) > 0 && !stateNames[view.StateType] {
			return fmt.Errorf("web UI smoke ui bundle view %s references unknown state_type %s", view.Name, view.StateType)
		}
		if len(view.Accessibility) == 0 {
			return fmt.Errorf("web UI smoke ui bundle view %s missing accessibility metadata", view.Name)
		}
		seenA11y := map[string]bool{}
		sawRole := false
		for _, a11y := range view.Accessibility {
			if a11y.Name == "" {
				return fmt.Errorf("web UI smoke ui bundle view %s has accessibility entry missing name", view.Name)
			}
			if a11y.Type == "" {
				return fmt.Errorf("web UI smoke ui bundle view %s accessibility %s missing type", view.Name, a11y.Name)
			}
			if a11y.Value == "" {
				return fmt.Errorf("web UI smoke ui bundle view %s accessibility %s missing value", view.Name, a11y.Name)
			}
			if seenA11y[a11y.Name] {
				return fmt.Errorf("web UI smoke ui bundle view %s duplicate accessibility metadata %s", view.Name, a11y.Name)
			}
			seenA11y[a11y.Name] = true
			if a11y.Name == "role" {
				sawRole = true
			}
		}
		if !sawRole {
			return fmt.Errorf("web UI smoke ui bundle view %s missing accessibility role", view.Name)
		}
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
