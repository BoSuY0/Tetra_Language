package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tetra_language/compiler"
	"tetra_language/tools/validators/platformui"
)

type crossPlatformInputs struct {
	Linux   string
	Windows string
	MacOS   string
	Web     string
}

type validationOptions struct {
	ExpectedVersion string
	ExpectedGitHead string
}

type linuxReport struct {
	Schema   string `json:"schema"`
	Status   string `json:"status"`
	Target   string `json:"target"`
	Runtime  string `json:"runtime"`
	UISchema string `json:"ui_schema"`
	Blocker  string `json:"blocker"`
}

type webReport struct {
	Schema       string `json:"schema"`
	Status       string `json:"status"`
	Target       string `json:"target"`
	UISchema     string `json:"ui_schema"`
	UIBundlePath string `json:"ui_bundle_path"`
	UIModulePath string `json:"ui_module_path"`
	DOMSnapshot  string `json:"dom_snapshot"`
	RuntimeTrace string `json:"runtime_trace"`
	Blocker      string `json:"blocker"`
}

func main() {
	var inputs crossPlatformInputs
	var options validationOptions
	flag.StringVar(&inputs.Linux, "linux", "", "Linux UI production report")
	flag.StringVar(&inputs.Windows, "windows", "", "Windows UI runtime report")
	flag.StringVar(&inputs.MacOS, "macos", "", "macOS UI runtime report")
	flag.StringVar(&inputs.Web, "web", "", "Web UI smoke report")
	flag.StringVar(&options.ExpectedVersion, "expected-version", compiler.Version(), "expected compiler/runtime version for target-host reports")
	flag.StringVar(&options.ExpectedGitHead, "expected-git-head", "", "expected git HEAD for target-host reports; defaults to current repository HEAD")
	flag.Parse()
	if strings.TrimSpace(options.ExpectedGitHead) == "" {
		head, err := currentGitHead()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not determine expected git head: %v\n", err)
			os.Exit(2)
		}
		options.ExpectedGitHead = head
	}
	if err := validateCrossPlatformUIRuntimeWithOptions(inputs, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCrossPlatformUIRuntime(inputs crossPlatformInputs) error {
	return validateCrossPlatformUIRuntimeWithOptions(inputs, validationOptions{})
}

func validateCrossPlatformUIRuntimeWithOptions(inputs crossPlatformInputs, options validationOptions) error {
	var issues []string
	if err := validateLinux(inputs.Linux); err != nil {
		issues = append(issues, "linux: "+err.Error())
	}
	if err := validatePlatform(inputs.Windows, "windows-x64", options); err != nil {
		issues = append(issues, "windows: "+err.Error())
	}
	if err := validatePlatform(inputs.MacOS, "macos-x64", options); err != nil {
		issues = append(issues, "macos: "+err.Error())
	}
	if err := validateWeb(inputs.Web); err != nil {
		issues = append(issues, "web: "+err.Error())
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func currentGitHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func validateLinux(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report linuxReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != "tetra.ui.desktop-runtime.v1" {
		issues = append(issues, "schema mismatch")
	}
	if report.Status != "pass" {
		issues = append(issues, "status is "+report.Status)
	}
	if report.Target != "linux-x64" {
		issues = append(issues, "target mismatch")
	}
	if report.Runtime != "desktop-ui-linux-x64" {
		issues = append(issues, "runtime mismatch")
	}
	if !platformui.AcceptedUISchemas[report.UISchema] {
		issues = append(issues, "ui_schema mismatch")
	}
	if report.Blocker != "" {
		issues = append(issues, "blocker must be empty")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validatePlatform(path string, target string, options validationOptions) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return platformui.ValidateReportWithOptions(raw, platformui.ValidateOptions{
		ExpectedTarget:  target,
		ExpectedVersion: options.ExpectedVersion,
		ExpectedGitHead: options.ExpectedGitHead,
	})
}

func validateWeb(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report webReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != "tetra.web-ui-smoke.v1alpha1" || report.Status != "pass" || report.Target != "wasm32-web" {
		issues = append(issues, "web report header mismatch")
	}
	if !platformui.AcceptedUISchemas[report.UISchema] {
		issues = append(issues, "ui_schema mismatch")
	}
	if report.UIBundlePath == "" || report.UIModulePath == "" || report.DOMSnapshot == "" {
		issues = append(issues, "ui bundle paths and dom_snapshot are required")
	}
	for _, marker := range []string{"window-mount:ok", "root-mount:ok", "layout:ok", "text:ok", "button:ok", "input:ok", "list:ok", "panel:ok", "focus:ok", "input-event:ok", "change:ok", "select:ok", "click:ok", "timer:ok", "async-command:ok", "redraw-update:ok", "error-recovery:ok", "ui-event-dispatch:web-command-dispatch"} {
		if !strings.Contains(report.RuntimeTrace, marker) {
			issues = append(issues, "runtime_trace missing "+marker)
		}
	}
	if report.Blocker != "" {
		issues = append(issues, "blocker must be empty")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}
