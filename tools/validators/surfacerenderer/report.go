package surfacerenderer

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
	SchemaV1                       = "tetra.surface.renderer-backend.v1"
	ScopeSurfaceProdScopedLinuxWeb = "surface-prod-scoped-linux-web"
)

type Report struct {
	Schema                 string                 `json:"schema"`
	Status                 string                 `json:"status"`
	Decision               string                 `json:"decision"`
	Scope                  string                 `json:"scope"`
	Producer               string                 `json:"producer"`
	GitHead                string                 `json:"git_head"`
	SoftwareBaseline       SoftwareBaselineReport `json:"software_baseline"`
	GPUCompositor          GPUCompositorReport    `json:"gpu_compositor"`
	TargetHostRequirements []string               `json:"target_host_requirements"`
	NonClaims              []string               `json:"nonclaims"`
	NegativeGuards         NegativeGuardsReport   `json:"negative_guards"`
	Cases                  []CaseReport           `json:"cases"`
}

type SoftwareBaselineReport struct {
	Backend        string `json:"backend"`
	ProductionPath bool   `json:"production_path"`
	EvidenceSchema string `json:"evidence_schema"`
	ReleaseGate    string `json:"release_gate"`
	Report         string `json:"report"`
}

type GPUCompositorReport struct {
	Status                   string                    `json:"status"`
	ProductionClaim          bool                      `json:"production_claim"`
	RequiredCapabilities     []string                  `json:"required_capabilities"`
	TargetHostBackendReports []TargetHostBackendReport `json:"target_host_backend_reports"`
	Fallback                 string                    `json:"fallback"`
	SameSceneEquivalence     bool                      `json:"same_scene_equivalence"`
}

type TargetHostBackendReport struct {
	Target     string `json:"target"`
	Backend    string `json:"backend"`
	Report     string `json:"report"`
	SameCommit bool   `json:"same_commit"`
}

type NegativeGuardsReport struct {
	GPUProductionWithoutBackendReportsRejected bool `json:"gpu_production_without_backend_reports_rejected"`
	DocsGPUProductionRejected                  bool `json:"docs_gpu_production_rejected"`
	SoftwareOnlyProdStableAllowed              bool `json:"software_only_prod_stable_allowed"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validateSoftwareBaseline(report.SoftwareBaseline)...)
	issues = append(issues, validateGPUCompositor(report.GPUCompositor)...)
	issues = append(issues, validateTargetHostRequirements(report.TargetHostRequirements)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Decision != "software-only-prod-go-gpu-experimental" {
		issues = append(issues, fmt.Sprintf("decision is %q, want software-only-prod-go-gpu-experimental", report.Decision))
	}
	if report.Scope != ScopeSurfaceProdScopedLinuxWeb {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q", report.Scope, ScopeSurfaceProdScopedLinuxWeb))
	}
	if report.Producer != "tools/cmd/validate-surface-renderer-report" {
		issues = append(issues, fmt.Sprintf("producer is %q, want tools/cmd/validate-surface-renderer-report", report.Producer))
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	return issues
}

func validateSoftwareBaseline(baseline SoftwareBaselineReport) []string {
	var issues []string
	if baseline.Backend != "software-rgba" {
		issues = append(issues, fmt.Sprintf("software_baseline backend is %q, want software-rgba", baseline.Backend))
	}
	if !baseline.ProductionPath {
		issues = append(issues, "software_baseline production_path must be true")
	}
	if baseline.EvidenceSchema != "tetra.surface.software-renderer.v1" {
		issues = append(issues, fmt.Sprintf("software_baseline evidence_schema is %q, want tetra.surface.software-renderer.v1", baseline.EvidenceSchema))
	}
	if !strings.Contains(baseline.ReleaseGate, "scripts/release/surface/release-gate.sh") {
		issues = append(issues, "software_baseline release_gate must include scripts/release/surface/release-gate.sh")
	}
	if strings.TrimSpace(baseline.Report) == "" {
		issues = append(issues, "software_baseline report is required")
	}
	return issues
}

func validateGPUCompositor(gpu GPUCompositorReport) []string {
	var issues []string
	if gpu.Fallback != "software-rgba" {
		issues = append(issues, fmt.Sprintf("gpu_compositor fallback is %q, want software-rgba", gpu.Fallback))
	}
	for _, capability := range []string{"layer_compositing", "transforms", "clipping", "texture_atlas", "vsync_frame_timing"} {
		if !containsString(gpu.RequiredCapabilities, capability) {
			issues = append(issues, fmt.Sprintf("gpu_compositor required_capabilities missing %s", capability))
		}
	}
	if gpu.ProductionClaim {
		if gpu.Status != "production" {
			issues = append(issues, fmt.Sprintf("gpu_compositor status is %q, want production for GPU production claim", gpu.Status))
		}
		if len(gpu.TargetHostBackendReports) == 0 {
			issues = append(issues, "gpu production claim requires target-host backend reports")
		}
		if !gpu.SameSceneEquivalence {
			issues = append(issues, "gpu production claim requires same_scene_equivalence")
		}
	} else {
		if gpu.Status != "experimental" {
			issues = append(issues, fmt.Sprintf("gpu_compositor status is %q, want experimental without GPU production evidence", gpu.Status))
		}
		if len(gpu.TargetHostBackendReports) > 0 {
			issues = append(issues, "gpu_compositor target_host_backend_reports must be empty while GPU remains experimental")
		}
		if gpu.SameSceneEquivalence {
			issues = append(issues, "gpu_compositor same_scene_equivalence must be false while GPU remains experimental")
		}
	}
	for i, backend := range gpu.TargetHostBackendReports {
		if strings.TrimSpace(backend.Target) == "" {
			issues = append(issues, fmt.Sprintf("gpu_compositor target_host_backend_reports[%d] target is required", i))
		}
		if strings.TrimSpace(backend.Backend) == "" {
			issues = append(issues, fmt.Sprintf("gpu_compositor target_host_backend_reports[%d] backend is required", i))
		}
		if strings.TrimSpace(backend.Report) == "" {
			issues = append(issues, fmt.Sprintf("gpu_compositor target_host_backend_reports[%d] report is required", i))
		}
		if !backend.SameCommit {
			issues = append(issues, fmt.Sprintf("gpu_compositor target_host_backend_reports[%d] same_commit must be true", i))
		}
	}
	return issues
}

func validateTargetHostRequirements(requirements []string) []string {
	required := []string{
		"linux target-host GPU smoke",
		"web compositor/canvas evidence",
		"Windows/macOS target-host GPU evidence if claimed",
	}
	var issues []string
	for _, want := range required {
		if !containsString(requirements, want) {
			issues = append(issues, fmt.Sprintf("target_host_requirements missing %s", want))
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	var issues []string
	for _, want := range []string{"GPU renderer production", "GPU compositor production"} {
		if !containsSubstring(nonclaims, want) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %s", want))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuardsReport) []string {
	var missing []string
	if !guards.GPUProductionWithoutBackendReportsRejected {
		missing = append(missing, "gpu_production_without_backend_reports_rejected")
	}
	if !guards.DocsGPUProductionRejected {
		missing = append(missing, "docs_gpu_production_rejected")
	}
	if !guards.SoftwareOnlyProdStableAllowed {
		missing = append(missing, "software_only_prod_stable_allowed")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"software-only scoped production go decision":                 false,
		"gpu production without target-host backend reports rejected": false,
		"docs gpu renderer production overclaim rejected":             false,
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
		if c.Kind != "positive" && c.Kind != "negative" {
			issues = append(issues, fmt.Sprintf("case %s kind is %q, want positive or negative", name, c.Kind))
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
			issues = append(issues, fmt.Sprintf("missing required case %q", name))
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, want string) bool {
	want = strings.ToLower(want)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), want) {
			return true
		}
	}
	return false
}
