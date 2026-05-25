package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"tetra_language/tools/validators/uiplatform"
)

type crossPlatformInputs struct {
	Linux   string
	Windows string
	MacOS   string
	Web     string
}

type webReport struct {
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

type linuxUIReport struct {
	Schema  string `json:"schema"`
	Status  string `json:"status"`
	Target  string `json:"target"`
	Host    string `json:"host"`
	Runtime string `json:"runtime"`
	Source  string `json:"source"`
}

const defaultMaxWebEvidenceAge = 7 * 24 * time.Hour

func main() {
	var inputs crossPlatformInputs
	flag.StringVar(&inputs.Linux, "linux", "", "path to linux-x64 UI production runtime report")
	flag.StringVar(&inputs.Windows, "windows", "", "path to windows-x64 UI runtime report")
	flag.StringVar(&inputs.MacOS, "macos", "", "path to macos-x64 UI runtime report")
	flag.StringVar(&inputs.Web, "web", "", "path to wasm32-web UI smoke report")
	flag.Parse()
	if inputs.Linux == "" || inputs.Windows == "" || inputs.MacOS == "" || inputs.Web == "" {
		fmt.Fprintln(os.Stderr, "error: --linux, --windows, --macos, and --web are required")
		os.Exit(2)
	}
	if err := validateCrossPlatformUIRuntime(inputs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCrossPlatformUIRuntime(inputs crossPlatformInputs) error {
	var issues []string
	if err := validateLinux(inputs.Linux); err != nil {
		issues = append(issues, fmt.Sprintf("linux UI runtime evidence invalid: %v", err))
	}
	if err := validatePlatform(inputs.Windows, uiplatform.Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64", Now: time.Now().UTC(), MaxAge: uiplatform.DefaultMaxEvidenceAge}); err != nil {
		issues = append(issues, fmt.Sprintf("windows UI runtime evidence invalid: %v", err))
	}
	if err := validatePlatform(inputs.MacOS, uiplatform.Options{Target: "macos-x64", Host: "macos-x64", Runtime: "platform-ui-macos-x64", Now: time.Now().UTC(), MaxAge: uiplatform.DefaultMaxEvidenceAge}); err != nil {
		issues = append(issues, fmt.Sprintf("macos UI runtime evidence invalid: %v", err))
	}
	if err := validateWeb(inputs.Web); err != nil {
		issues = append(issues, fmt.Sprintf("web UI runtime evidence invalid: %v", err))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateLinux(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report linuxUIReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	if report.Schema != "tetra.ui.desktop-runtime.v1" {
		return fmt.Errorf("schema is %q, want tetra.ui.desktop-runtime.v1", report.Schema)
	}
	if report.Status != "pass" {
		return fmt.Errorf("status is %q, want pass", report.Status)
	}
	if report.Target != "linux-x64" || report.Host != "linux-x64" {
		return fmt.Errorf("target/host is %q/%q, want linux-x64/linux-x64", report.Target, report.Host)
	}
	if report.Runtime != "desktop-ui-linux-x64" {
		return fmt.Errorf("runtime is %q, want desktop-ui-linux-x64", report.Runtime)
	}
	if strings.TrimSpace(report.Source) == "" {
		return fmt.Errorf("source is required")
	}
	return nil
}

func validatePlatform(path string, opts uiplatform.Options) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return uiplatform.ValidateReport(raw, opts)
}

func validateWeb(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report webReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return err
	}
	if err := rejectForbiddenText(raw); err != nil {
		return err
	}
	if report.Schema != "tetra.web-ui-smoke.v1alpha1" {
		return fmt.Errorf("schema is %q, want tetra.web-ui-smoke.v1alpha1", report.Schema)
	}
	generatedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil {
		return fmt.Errorf("generated_at is not RFC3339: %w", err)
	}
	now := time.Now().UTC()
	if generatedAt.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("generated_at %s is in the future relative to %s", generatedAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	if now.Sub(generatedAt) > defaultMaxWebEvidenceAge {
		return fmt.Errorf("generated_at %s is stale; max age is %s", generatedAt.Format(time.RFC3339), defaultMaxWebEvidenceAge)
	}
	if report.Target != "wasm32-web" {
		return fmt.Errorf("target is %q, want wasm32-web", report.Target)
	}
	if report.Status != "pass" {
		return fmt.Errorf("status is %q, want pass", report.Status)
	}
	if !report.UIScopeActive || report.UsedFallbackSource {
		return fmt.Errorf("web UI report must use active UI source without fallback")
	}
	if report.UISchema != "tetra.ui.v1" {
		return fmt.Errorf("ui_schema is %q, want tetra.ui.v1", report.UISchema)
	}
	if report.Blocker != "" {
		return fmt.Errorf("pass report cannot include blocker")
	}
	for _, path := range []struct {
		name  string
		value string
	}{
		{"ui_bundle_path", report.UIBundlePath},
		{"ui_module_path", report.UIModulePath},
		{"dom_snapshot", report.DOMSnapshot},
	} {
		if strings.TrimSpace(path.value) == "" {
			return fmt.Errorf("%s is required", path.name)
		}
		if info, err := os.Stat(path.value); err != nil {
			return fmt.Errorf("%s must point to an existing artifact: %w", path.name, err)
		} else if info.IsDir() || info.Size() == 0 {
			return fmt.Errorf("%s must point to a non-empty file", path.name)
		}
	}
	for _, marker := range requiredWebRuntimeMarkers() {
		if !strings.Contains(report.RuntimeTrace, marker) {
			return fmt.Errorf("runtime_trace missing %q", marker)
		}
	}
	return nil
}

func rejectForbiddenText(raw []byte) error {
	text := strings.ToLower(string(raw))
	for _, marker := range []string{"metadata-only", "runtime-less", "build-only", "docs-only", "sidecar-only", "startup_failure", " fake", "mock", "placeholder"} {
		if strings.Contains(text, marker) {
			return fmt.Errorf("report contains forbidden non-runtime evidence marker %q", strings.TrimSpace(marker))
		}
	}
	return nil
}

func requiredWebRuntimeMarkers() []string {
	return []string{
		"window/root mount:ok",
		"layout:ok",
		"text:ok",
		"button:ok",
		"input:ok",
		"list:ok",
		"panel:ok",
		"focus:ok",
		"change:ok",
		"select:ok",
		"click:ok",
		"timer:ok",
		"async command:ok",
		"redraw/update:ok",
		"error recovery:ok",
		"main-exit:ok",
		"stdout:ok",
		"nonzero-exit:ok",
		"failure-propagation:ok",
		"repeated-instantiation:ok",
		"ui-event-dispatch:web-command-dispatch",
	}
}

func decodeStrictJSON(raw []byte, out any) error {
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
