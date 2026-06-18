package surfaceprod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	SchemaV1                             = "tetra.surface.prod-claim.v1"
	ClaimTierProdStableScopedLinuxWebApp = "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI"
	ScopeSurfaceProdScopedLinuxWeb       = "surface-prod-scoped-linux-web"
)

type ClaimReport struct {
	Schema                  string                  `json:"schema"`
	Status                  string                  `json:"status"`
	ClaimTier               string                  `json:"claim_tier"`
	Scope                   string                  `json:"scope"`
	Summary                 string                  `json:"summary"`
	Producer                string                  `json:"producer"`
	GitHead                 string                  `json:"git_head"`
	GitDirty                bool                    `json:"git_dirty"`
	RuntimeDependencyPolicy RuntimeDependencyPolicy `json:"runtime_dependency_policy"`
	Capabilities            CapabilityClaims        `json:"capabilities"`
	SupportedTargets        []TargetClaim           `json:"supported_targets"`
	UnsupportedTargets      []string                `json:"unsupported_targets"`
	NonClaims               []string                `json:"nonclaims"`
	TargetHostEvidence      []TargetHostEvidence    `json:"target_host_evidence"`
	GateEvidence            []GateEvidence          `json:"gate_evidence"`
	Cases                   []CaseReport            `json:"cases"`
}

type RuntimeDependencyPolicy struct {
	Electron             bool `json:"electron"`
	ChromiumDesktopShell bool `json:"chromium_desktop_shell"`
	ReactRuntime         bool `json:"react_runtime"`
	DOMUI                bool `json:"dom_ui"`
	CSSRuntime           bool `json:"css_runtime"`
	UserJSAppLogic       bool `json:"user_js_app_logic"`
	PlatformWidgets      bool `json:"platform_widgets"`
}

type CapabilityClaims struct {
	Renderer                   string `json:"renderer"`
	GPUProduction              bool   `json:"gpu_production"`
	CrossPlatformDesktopParity bool   `json:"cross_platform_desktop_parity"`
	AccessibilityLevel         string `json:"accessibility_level"`
	FullAccessibilityParity    bool   `json:"full_accessibility_parity"`
}

type TargetClaim struct {
	Target       string `json:"target"`
	SupportLevel string `json:"support_level"`
	Evidence     string `json:"evidence"`
}

type TargetHostEvidence struct {
	Target        string `json:"target"`
	Host          string `json:"host"`
	Level         string `json:"level"`
	RealWindow    bool   `json:"real_window"`
	NativeInput   bool   `json:"native_input"`
	BrowserCanvas bool   `json:"browser_canvas"`
	SameCommit    bool   `json:"same_commit"`
	Report        string `json:"report"`
}

type GateEvidence struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateClaim(raw []byte) error {
	var report ClaimReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, rejectBroadSummaryOverclaims(report.Summary)...)
	issues = append(issues, validateRuntimeDependencyPolicy(report.RuntimeDependencyPolicy)...)
	issues = append(issues, validateCapabilities(report.Capabilities)...)
	issues = append(issues, validateTargets(report.SupportedTargets, report.UnsupportedTargets)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateTargetHostEvidence(report.TargetHostEvidence)...)
	issues = append(issues, validateGateEvidence(report.GateEvidence)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, rejectPaperEvidence(report)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateIdentity(report ClaimReport) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.ClaimTier != ClaimTierProdStableScopedLinuxWebApp {
		issues = append(issues, fmt.Sprintf("claim_tier is %q, want %q", report.ClaimTier, ClaimTierProdStableScopedLinuxWebApp))
	}
	if report.Scope != ScopeSurfaceProdScopedLinuxWeb {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q", report.Scope, ScopeSurfaceProdScopedLinuxWeb))
	}
	if strings.TrimSpace(report.Summary) == "" {
		issues = append(issues, "summary is required")
	}
	if strings.TrimSpace(report.Producer) == "" {
		issues = append(issues, "producer is required")
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	if report.GitDirty {
		issues = append(issues, "git_dirty must be false for production claim validation")
	}
	return issues
}

func rejectBroadSummaryOverclaims(summary string) []string {
	normalized := normalizeClaimText(summary)
	var issues []string
	for _, rule := range []struct {
		needle string
		issue  string
	}{
		{needle: "fully replaces electron", issue: "summary contains fake broad Electron replacement claim"},
		{needle: "complete electron replacement", issue: "summary contains fake broad Electron replacement claim"},
		{needle: "drop in electron replacement", issue: "summary contains fake broad Electron replacement claim"},
		{needle: "all electron apps", issue: "summary contains fake broad Electron replacement claim"},
		{needle: "all production ui", issue: "summary contains all production UI overclaim"},
		{needle: "cross platform production parity", issue: "summary contains cross-platform production parity overclaim"},
		{needle: "full accessibility parity", issue: "summary contains full accessibility parity overclaim"},
		{needle: "gpu production", issue: "summary contains GPU production overclaim"},
	} {
		if strings.Contains(normalized, rule.needle) {
			issues = append(issues, rule.issue)
		}
	}
	return issues
}

func validateRuntimeDependencyPolicy(policy RuntimeDependencyPolicy) []string {
	checks := []struct {
		name string
		used bool
	}{
		{name: "electron", used: policy.Electron},
		{name: "chromium_desktop_shell", used: policy.ChromiumDesktopShell},
		{name: "react_runtime", used: policy.ReactRuntime},
		{name: "dom_ui", used: policy.DOMUI},
		{name: "css_runtime", used: policy.CSSRuntime},
		{name: "user_js_app_logic", used: policy.UserJSAppLogic},
		{name: "platform_widgets", used: policy.PlatformWidgets},
	}
	var issues []string
	for _, check := range checks {
		if check.used {
			issues = append(issues, fmt.Sprintf("runtime dependency policy must forbid %s", check.name))
		}
	}
	return issues
}

func validateCapabilities(claims CapabilityClaims) []string {
	var issues []string
	if claims.Renderer != "software-rgba" {
		issues = append(issues, fmt.Sprintf("renderer is %q, want software-rgba until GPU production evidence exists", claims.Renderer))
	}
	if claims.GPUProduction {
		issues = append(issues, "gpu_production must be false without target-host GPU backend evidence")
	}
	if claims.CrossPlatformDesktopParity {
		issues = append(issues, "cross_platform_desktop_parity must be false for scoped Linux/web claim")
	}
	if claims.AccessibilityLevel != "scoped-platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_level is %q, want scoped-platform-bridge-v1", claims.AccessibilityLevel))
	}
	if claims.FullAccessibilityParity {
		issues = append(issues, "full_accessibility_parity must be false without full platform accessibility evidence")
	}
	return issues
}

func validateTargets(supported []TargetClaim, unsupported []string) []string {
	requiredProduction := map[string]bool{"linux-x64": false, "wasm32-web": false}
	requiredUnsupported := map[string]bool{"macos-x64": false, "windows-x64": false, "wasm32-wasi": false}
	unsupportedSet := map[string]bool{}
	for _, target := range unsupported {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		unsupportedSet[target] = true
		if _, ok := requiredUnsupported[target]; ok {
			requiredUnsupported[target] = true
		}
	}
	var issues []string
	for _, target := range supported {
		name := strings.TrimSpace(target.Target)
		if name == "" {
			issues = append(issues, "supported target name is required")
			continue
		}
		if unsupportedSet[name] {
			issues = append(issues, fmt.Sprintf("target %s is listed as both supported and unsupported", name))
		}
		if _, blocked := requiredUnsupported[name]; blocked {
			issues = append(issues, fmt.Sprintf("unsupported target %s cannot be claimed as supported production", name))
		}
		if strings.TrimSpace(target.SupportLevel) == "" {
			issues = append(issues, fmt.Sprintf("target %s support_level is required", name))
		}
		if strings.TrimSpace(target.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("target %s evidence is required", name))
		}
		if _, ok := requiredProduction[name]; ok {
			requiredProduction[name] = true
			if target.SupportLevel != "production" {
				issues = append(issues, fmt.Sprintf("target %s support_level is %q, want production", name, target.SupportLevel))
			}
		}
	}
	for target, seen := range requiredProduction {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required production target %s", target))
		}
	}
	for target, seen := range requiredUnsupported {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required unsupported target %s", target))
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	required := map[string]bool{
		"electron":       false,
		"cross-platform": false,
		"gpu":            false,
		"accessibility":  false,
		"css":            false,
	}
	for _, nonclaim := range nonclaims {
		lower := strings.ToLower(nonclaim)
		for key := range required {
			if strings.Contains(lower, key) {
				required[key] = true
			}
		}
	}
	var issues []string
	for key, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing %s nonclaim", key))
		}
	}
	return issues
}

func validateTargetHostEvidence(evidence []TargetHostEvidence) []string {
	byTarget := map[string]TargetHostEvidence{}
	for _, item := range evidence {
		if strings.TrimSpace(item.Target) != "" {
			byTarget[item.Target] = item
		}
	}
	var issues []string
	linux, ok := byTarget["linux-x64"]
	if !ok {
		issues = append(issues, "missing target-host evidence for linux-x64")
	} else {
		issues = append(issues, validateLinuxHostEvidence(linux)...)
	}
	web, ok := byTarget["wasm32-web"]
	if !ok {
		issues = append(issues, "missing target-host evidence for wasm32-web")
	} else {
		issues = append(issues, validateWebHostEvidence(web)...)
	}
	return issues
}

func validateLinuxHostEvidence(evidence TargetHostEvidence) []string {
	var issues []string
	if evidence.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("linux-x64 target-host host is %q, want linux-x64", evidence.Host))
	}
	if evidence.Level != "target-host" {
		issues = append(issues, fmt.Sprintf("linux-x64 target-host level is %q, want target-host", evidence.Level))
	}
	if !evidence.RealWindow {
		issues = append(issues, "linux-x64 target-host evidence requires real_window")
	}
	if !evidence.NativeInput {
		issues = append(issues, "linux-x64 target-host evidence requires native_input")
	}
	if !evidence.SameCommit {
		issues = append(issues, "linux-x64 target-host evidence requires same_commit")
	}
	if strings.TrimSpace(evidence.Report) == "" {
		issues = append(issues, "linux-x64 target-host evidence report is required")
	}
	return issues
}

func validateWebHostEvidence(evidence TargetHostEvidence) []string {
	var issues []string
	if evidence.Level != "browser-canvas" {
		issues = append(issues, fmt.Sprintf("wasm32-web target-host level is %q, want browser-canvas", evidence.Level))
	}
	if !evidence.BrowserCanvas {
		issues = append(issues, "wasm32-web target-host evidence requires browser_canvas")
	}
	if !evidence.SameCommit {
		issues = append(issues, "wasm32-web target-host evidence requires same_commit")
	}
	if strings.TrimSpace(evidence.Report) == "" {
		issues = append(issues, "wasm32-web target-host evidence report is required")
	}
	return issues
}

func validateGateEvidence(gates []GateEvidence) []string {
	required := map[string]bool{
		"surface release state":            false,
		"renderer backend decision gate":   false,
		"claim taxonomy negative fixtures": false,
	}
	var issues []string
	for _, gate := range gates {
		name := strings.TrimSpace(gate.Name)
		if name == "" {
			issues = append(issues, "gate evidence name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if gate.Status != "pass" {
			issues = append(issues, fmt.Sprintf("gate evidence %s status is %q, want pass", name, gate.Status))
		}
		if strings.TrimSpace(gate.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("gate evidence %s evidence is required", name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing gate evidence %q", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"fake electron/react/css replacement rejected":                false,
		"fake cross-platform support rejected":                        false,
		"fake gpu production claim rejected":                          false,
		"gpu production without target-host backend reports rejected": false,
		"fake full accessibility parity rejected":                     false,
		"missing target-host evidence rejected":                       false,
	}
	var issues []string
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Kind != "negative" && c.Kind != "positive" && c.Kind != "stress" {
			issues = append(issues, fmt.Sprintf("case %s kind is %q, want negative, positive, or stress", name, c.Kind))
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required negative case %q", name))
		}
	}
	return issues
}

func rejectPaperEvidence(report ClaimReport) []string {
	fields := []string{report.Producer}
	for _, target := range report.SupportedTargets {
		fields = append(fields, target.Evidence)
	}
	for _, evidence := range report.TargetHostEvidence {
		fields = append(fields, evidence.Report)
	}
	for _, gate := range report.GateEvidence {
		fields = append(fields, gate.Evidence)
	}
	text := strings.ToLower(strings.Join(fields, "\n"))
	var issues []string
	for _, marker := range []string{"docs-only", "mock", "fake/", "\"fake\"", "placeholder"} {
		if strings.Contains(text, marker) {
			issues = append(issues, fmt.Sprintf("claim evidence contains forbidden paper evidence marker %q", strings.Trim(marker, "\"/")))
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
		return fmt.Errorf("unexpected trailing JSON content")
	}
	return nil
}

func normalizeClaimText(text string) string {
	lower := strings.ToLower(text)
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "-", " ", "_", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(lower)), " ")
}
