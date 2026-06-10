package surfacewindows

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const SchemaV1 = "tetra.surface.windows-target.v1"

type Report struct {
	Schema          string               `json:"schema"`
	Status          string               `json:"status"`
	Target          string               `json:"target"`
	Host            string               `json:"host"`
	SupportLevel    string               `json:"support_level"`
	EvidenceKind    string               `json:"evidence_kind"`
	SurfaceSchema   string               `json:"surface_schema"`
	AppShellABI     string               `json:"app_shell_abi"`
	ProductionClaim bool                 `json:"production_claim"`
	BetaClaim       bool                 `json:"beta_claim"`
	BlockedReason   string               `json:"blocked_reason"`
	Capabilities    CapabilityReport     `json:"capabilities"`
	Processes       []ProcessReport      `json:"processes"`
	NegativeGuards  NegativeGuardsReport `json:"negative_guards"`
	NonClaims       []string             `json:"nonclaims"`
	Cases           []CaseReport         `json:"cases"`
}

type CapabilityReport struct {
	NativeWindow        bool `json:"native_window"`
	NativeInput         bool `json:"native_input"`
	Clipboard           bool `json:"clipboard"`
	IME                 bool `json:"ime"`
	DPI                 bool `json:"dpi"`
	Menus               bool `json:"menus"`
	Dialogs             bool `json:"dialogs"`
	Notifications       bool `json:"notifications"`
	AccessibilityBridge bool `json:"accessibility_bridge"`
	AppShell            bool `json:"app_shell"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type NegativeGuardsReport struct {
	BuildOnlyRejected                bool `json:"build_only_rejected"`
	LinuxHostSyntheticRejected       bool `json:"linux_host_synthetic_rejected"`
	ProductionWithoutFullDoDRejected bool `json:"production_without_full_dod_rejected"`
	GenericUIRuntimeRejected         bool `json:"generic_ui_runtime_rejected"`
}

type CaseReport struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedError string `json:"expected_error,omitempty"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %s", report.Schema, SchemaV1))
	}
	if report.Target != "windows-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want windows-x64", report.Target))
	}
	if strings.TrimSpace(report.Host) == "" {
		issues = append(issues, "host is required")
	}
	if report.SurfaceSchema != "tetra.surface.v1" {
		issues = append(issues, fmt.Sprintf("surface_schema is %q, want tetra.surface.v1", report.SurfaceSchema))
	}
	if report.AppShellABI != "tetra.surface.app-shell.v1" {
		issues = append(issues, fmt.Sprintf("app_shell_abi is %q, want tetra.surface.app-shell.v1", report.AppShellABI))
	}
	issues = append(issues, validateStatusAndEvidence(report)...)
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
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

func validateStatusAndEvidence(report Report) []string {
	var issues []string
	switch report.Status {
	case "nonclaim":
		if report.SupportLevel != "unsupported" {
			issues = append(issues, fmt.Sprintf("nonclaim support_level is %q, want unsupported", report.SupportLevel))
		}
		if report.EvidenceKind != "nonclaim-boundary" {
			issues = append(issues, fmt.Sprintf("nonclaim evidence_kind is %q, want nonclaim-boundary", report.EvidenceKind))
		}
		if strings.TrimSpace(report.BlockedReason) == "" {
			issues = append(issues, "nonclaim Windows boundary requires blocked_reason")
		}
		if report.ProductionClaim || report.BetaClaim {
			issues = append(issues, "nonclaim Windows boundary cannot set production_claim or beta_claim")
		}
		if anyCapability(report.Capabilities) {
			issues = append(issues, "nonclaim Windows boundary cannot claim target-host capabilities")
		}
	case "beta":
		if report.SupportLevel != "beta-target-host" {
			issues = append(issues, fmt.Sprintf("beta support_level is %q, want beta-target-host", report.SupportLevel))
		}
		if report.EvidenceKind != "target-host-surface-beta" {
			issues = append(issues, fmt.Sprintf("beta evidence_kind is %q, want target-host-surface-beta", report.EvidenceKind))
		}
		if report.Host != "windows-x64" {
			issues = append(issues, fmt.Sprintf("windows beta target-host evidence host is %q, want windows-x64 target-host", report.Host))
		}
		if report.ProductionClaim {
			issues = append(issues, "Windows production claim is rejected until full Surface target-host DoD exists")
		}
		if !report.BetaClaim {
			issues = append(issues, "Windows beta target-host report requires beta_claim")
		}
		issues = append(issues, validateBetaCapabilities(report.Capabilities)...)
	default:
		issues = append(issues, fmt.Sprintf("status is %q, want nonclaim or beta", report.Status))
	}
	if report.SupportLevel == "production" || report.ProductionClaim {
		issues = append(issues, "Windows production is rejected until full target-host DoD, P20 accessibility, and P26 packaging evidence exist")
	}
	if strings.Contains(strings.ToLower(report.EvidenceKind), "build-only") {
		issues = append(issues, "build-only Windows target evidence cannot count as Surface UI runtime")
	}
	return issues
}

func anyCapability(cap CapabilityReport) bool {
	return cap.NativeWindow || cap.NativeInput || cap.Clipboard || cap.IME || cap.DPI ||
		cap.Menus || cap.Dialogs || cap.Notifications || cap.AccessibilityBridge || cap.AppShell
}

func validateBetaCapabilities(cap CapabilityReport) []string {
	var issues []string
	checks := []struct {
		name string
		ok   bool
	}{
		{name: "native_window", ok: cap.NativeWindow},
		{name: "native_input", ok: cap.NativeInput},
		{name: "clipboard", ok: cap.Clipboard},
		{name: "ime", ok: cap.IME},
		{name: "dpi", ok: cap.DPI},
		{name: "menus", ok: cap.Menus},
		{name: "dialogs", ok: cap.Dialogs},
		{name: "notifications", ok: cap.Notifications},
		{name: "accessibility_bridge", ok: cap.AccessibilityBridge},
		{name: "app_shell", ok: cap.AppShell},
	}
	for _, check := range checks {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("Windows beta target-host report requires %s capability evidence", check.name))
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	if len(processes) == 0 {
		return []string{"Windows Surface boundary report requires process evidence"}
	}
	var issues []string
	for _, process := range processes {
		if strings.TrimSpace(process.Name) == "" || strings.TrimSpace(process.Kind) == "" || strings.TrimSpace(process.Path) == "" {
			issues = append(issues, "process evidence requires name, kind, and path")
		}
		if !process.Ran || !process.Pass {
			issues = append(issues, fmt.Sprintf("process %q must run and pass", process.Name))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuardsReport) []string {
	var issues []string
	if !guards.BuildOnlyRejected {
		issues = append(issues, "Windows boundary must reject build-only UI runtime evidence")
	}
	if !guards.LinuxHostSyntheticRejected {
		issues = append(issues, "Windows boundary must reject linux-host synthetic target-host evidence")
	}
	if !guards.ProductionWithoutFullDoDRejected {
		issues = append(issues, "Windows boundary must reject production without full DoD")
	}
	if !guards.GenericUIRuntimeRejected {
		issues = append(issues, "Windows boundary must reject generic tetra.ui.v1 runtime as Surface production evidence")
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	var issues []string
	for _, required := range []string{
		"Windows Surface production target",
		"build-only Windows UI runtime",
		"linux-host synthetic Windows report",
		"generic tetra.ui.v1 platform UI runtime",
	} {
		if !containsFold(nonclaims, required) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q", required))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	var issues []string
	for _, required := range []string{
		"windows build-only target rejected as Surface UI runtime",
		"linux-host synthetic Windows report rejected",
	} {
		if !caseNameContains(cases, required) {
			issues = append(issues, fmt.Sprintf("Windows boundary report requires %s case", required))
		}
	}
	for _, c := range cases {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" || !c.Ran || !c.Pass {
			issues = append(issues, "cases require name, kind, ran, and pass")
		}
	}
	return issues
}

func containsFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), want) {
			return true
		}
	}
	return false
}

func caseNameContains(cases []CaseReport, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, c := range cases {
		if strings.Contains(strings.ToLower(strings.TrimSpace(c.Name)), want) {
			return true
		}
	}
	return false
}
