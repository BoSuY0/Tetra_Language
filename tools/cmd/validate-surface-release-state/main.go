package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

const surfaceArtifactHashSchema = "tetra.release-artifact-hashes.v1alpha1"
const surfaceProdStableScopedLinuxWeb = "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI"

type surfaceReleaseStateOptions struct {
	ReportDir      string
	ExpectedStatus string
	Scope          string
	ManifestPath   string
}

type surfaceReleaseArtifactHashManifest struct {
	Schema    string                       `json:"schema"`
	Root      string                       `json:"root"`
	Artifacts []surfaceReleaseHashArtifact `json:"artifacts"`
}

type surfaceReleaseHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type surfaceReleaseRuntimeEnvelope struct {
	Schema       string                     `json:"schema"`
	Status       string                     `json:"status"`
	Target       string                     `json:"target"`
	Source       string                     `json:"source"`
	HostEvidence surface.HostEvidenceReport `json:"host_evidence"`
}

type surfaceMorphGateSummary struct {
	Schema                  string   `json:"schema"`
	Status                  string   `json:"status"`
	ReleaseScope            string   `json:"release_scope"`
	Producer                string   `json:"producer"`
	Source                  string   `json:"source"`
	Module                  string   `json:"module"`
	SchemaUnderTest         string   `json:"schema_under_test"`
	DependencyGate          string   `json:"dependency_gate"`
	SameCommitValidated     bool     `json:"same_commit_validated"`
	HeadlessReport          string   `json:"headless_report"`
	TargetEvidence          []string `json:"target_evidence"`
	CorePrimitives          []string `json:"core_primitives"`
	ForbiddenCorePrimitives []string `json:"forbidden_core_primitives"`
	ArtifactHashesValidated bool     `json:"artifact_hashes_validated"`
}

type surfaceProdGateReport struct {
	Schema                  string                      `json:"schema"`
	Status                  string                      `json:"status"`
	Level                   string                      `json:"level"`
	Scope                   string                      `json:"scope"`
	ReleaseScope            string                      `json:"release_scope"`
	Producer                string                      `json:"producer"`
	GitHead                 string                      `json:"git_head"`
	SameCommit              bool                        `json:"same_commit"`
	CIJobs                  []surfaceProdCIJob          `json:"ci_jobs"`
	Gates                   []surfaceProdGateEvidence   `json:"gates"`
	Targets                 []surfaceProdTargetEvidence `json:"targets"`
	ArtifactHashesValidated bool                        `json:"artifact_hashes_validated"`
	NegativeGuards          surfaceProdNegativeGuards   `json:"negative_guards"`
	Cases                   []surfaceProdCaseReport     `json:"cases"`
}

type surfaceProdCIJob struct {
	Workflow        string `json:"workflow"`
	Job             string `json:"job"`
	Required        bool   `json:"required"`
	ContinueOnError bool   `json:"continue_on_error"`
	Command         string `json:"command"`
	ArtifactUpload  string `json:"artifact_upload"`
}

type surfaceProdGateEvidence struct {
	Name                 string `json:"name"`
	ReportDir            string `json:"report_dir"`
	Ran                  bool   `json:"ran"`
	Pass                 bool   `json:"pass"`
	Skipped              bool   `json:"skipped"`
	ArtifactHashManifest string `json:"artifact_hash_manifest"`
}

type surfaceProdTargetEvidence struct {
	Target  string `json:"target"`
	Tier    string `json:"tier"`
	Ran     bool   `json:"ran"`
	Pass    bool   `json:"pass"`
	Skipped bool   `json:"skipped"`
}

type surfaceProdNegativeGuards struct {
	MissingJobRejected                  bool `json:"missing_job_rejected"`
	ContinueOnErrorRejected             bool `json:"continue_on_error_rejected"`
	SkippedTargetAsPassRejected         bool `json:"skipped_target_as_pass_rejected"`
	MissingArtifactHashManifestRejected bool `json:"missing_artifact_hash_manifest_rejected"`
}

type surfaceProdCaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func main() {
	var opt surfaceReleaseStateOptions
	flag.StringVar(&opt.ReportDir, "report-dir", "", "Surface release report directory")
	flag.StringVar(&opt.ExpectedStatus, "expected-status", "current", "expected Surface release status")
	flag.StringVar(&opt.Scope, "scope", surface.ReleaseScopeSurfaceV1LinuxWeb, "expected Surface release scope")
	flag.StringVar(&opt.ManifestPath, "manifest", "docs/generated/manifest.json", "docs/generated manifest path")
	flag.Parse()
	if strings.TrimSpace(opt.ReportDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateSurfaceReleaseState(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceReleaseState(opt surfaceReleaseStateOptions) error {
	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	expectedStatus := strings.TrimSpace(opt.ExpectedStatus)
	if expectedStatus == "" {
		expectedStatus = "current"
	}
	scope := strings.TrimSpace(opt.Scope)
	if scope == "" {
		scope = surface.ReleaseScopeSurfaceV1LinuxWeb
	}
	var issues []string
	if expectedStatus != "current" {
		issues = append(issues, fmt.Sprintf("expected-status is %q, want current", expectedStatus))
	}
	prodClaimScope := scope == surfaceProdStableScopedLinuxWeb
	if scope != surface.ReleaseScopeSurfaceV1LinuxWeb && !prodClaimScope {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q or %q", scope, surface.ReleaseScopeSurfaceV1LinuxWeb, surfaceProdStableScopedLinuxWeb))
	}
	releaseScope := surface.ReleaseScopeSurfaceV1LinuxWeb
	issues = append(issues, validateReleaseSummaryFile(filepath.Join(reportDir, "surface-release-summary.json"), releaseScope, expectedStatus)...)
	issues = append(issues, validateReleaseTextInputFile(filepath.Join(reportDir, "surface-headless-release-text-input.json"))...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-wasm32-web-release-browser.json"), "wasm32-web")...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-linux-x64-release-window.json"), "linux-x64")...)
	issues = append(issues, validateReleaseMorphGateFile(filepath.Join(reportDir, "morph", "surface-morph-gate-summary.json"))...)
	issues = append(issues, validateReleaseMorphReportFile(filepath.Join(reportDir, "morph", "headless", "surface-headless-morph.json"))...)
	issues = append(issues, validateSurfaceArtifactHashes(filepath.Join(reportDir, "artifact-hashes.json"))...)
	issues = append(issues, validateSurfaceReleaseManifest(opt.ManifestPath, releaseScope, expectedStatus)...)
	if prodClaimScope {
		issues = append(issues, validateSurfaceProdGateReportFile(filepath.Join(reportDir, "surface-prod-gate-report.json"))...)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceProdGateReportFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var report surfaceProdGateReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: "tetra.surface.prod-gate-report.v1"},
		{field: "status", got: report.Status, want: "pass"},
		{field: "level", got: report.Level, want: "surface-production-ci-release-gate-v1"},
		{field: "scope", got: report.Scope, want: surfaceProdStableScopedLinuxWeb},
		{field: "release_scope", got: report.ReleaseScope, want: surface.ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/prod-gate.sh"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s %s is %q, want %q", filepath.Base(path), check.field, check.got, check.want))
		}
	}
	if !validSurfaceReleaseGitHead(report.GitHead) {
		issues = append(issues, fmt.Sprintf("%s git_head must be a 40-hex revision", filepath.Base(path)))
	}
	if !report.SameCommit {
		issues = append(issues, fmt.Sprintf("%s same_commit must be true", filepath.Base(path)))
	}
	if !report.ArtifactHashesValidated {
		issues = append(issues, fmt.Sprintf("%s artifact hashes must be validated", filepath.Base(path)))
	}
	issues = append(issues, validateSurfaceProdCIJobs(path, report.CIJobs)...)
	issues = append(issues, validateSurfaceProdGates(path, report.Gates)...)
	issues = append(issues, validateSurfaceProdTargets(path, report.Targets)...)
	issues = append(issues, validateSurfaceProdNegativeGuards(path, report.NegativeGuards)...)
	issues = append(issues, validateSurfaceProdCases(path, report.Cases)...)
	return issues
}

func validateSurfaceProdCIJobs(path string, jobs []surfaceProdCIJob) []string {
	if len(jobs) == 0 {
		return []string{fmt.Sprintf("%s missing production gate CI job", filepath.Base(path))}
	}
	var issues []string
	foundRequired := false
	for _, job := range jobs {
		if strings.TrimSpace(job.Workflow) == "" || strings.TrimSpace(job.Job) == "" {
			issues = append(issues, fmt.Sprintf("%s CI job workflow and job are required", filepath.Base(path)))
		}
		if job.ContinueOnError {
			issues = append(issues, fmt.Sprintf("%s CI job %s uses continue-on-error", filepath.Base(path), job.Job))
		}
		if strings.Contains(job.Command, "prod-gate.sh") && job.Required && strings.Contains(job.Workflow, "release-packages.yml") {
			foundRequired = true
		}
		if strings.Contains(job.Command, "prod-gate.sh") && !strings.Contains(job.ArtifactUpload, "surface-production-final") {
			issues = append(issues, fmt.Sprintf("%s production gate CI job must upload surface-production-final artifacts", filepath.Base(path)))
		}
	}
	if !foundRequired {
		issues = append(issues, fmt.Sprintf("%s missing production gate CI job for release-packages.yml", filepath.Base(path)))
	}
	return issues
}

func validateSurfaceProdGates(path string, gates []surfaceProdGateEvidence) []string {
	required := map[string]bool{
		"surface-release":     false,
		"block-system":        false,
		"morph":               false,
		"visual":              false,
		"package":             false,
		"security":            false,
		"ipc-lifecycle":       false,
		"crash-diagnostics":   false,
		"i18n-localization":   false,
		"performance-memory":  false,
		"widget-migration":    false,
		"example-suite":       false,
		"api-stability":       false,
		"electron-comparison": false,
		"prod-claim":          false,
	}
	if len(gates) == 0 {
		return []string{fmt.Sprintf("%s production gates are required", filepath.Base(path))}
	}
	var issues []string
	for _, gate := range gates {
		name := strings.TrimSpace(gate.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if !gate.Ran || !gate.Pass {
			issues = append(issues, fmt.Sprintf("%s gate %s must run and pass", filepath.Base(path), name))
		}
		if gate.Skipped && gate.Pass {
			issues = append(issues, fmt.Sprintf("%s skipped gate %s counted as pass", filepath.Base(path), name))
		}
		if strings.TrimSpace(gate.ReportDir) == "" {
			issues = append(issues, fmt.Sprintf("%s gate %s report_dir is required", filepath.Base(path), name))
		}
		if strings.TrimSpace(gate.ArtifactHashManifest) == "" {
			issues = append(issues, fmt.Sprintf("%s gate %s artifact hash manifest is required", filepath.Base(path), name))
		}
	}
	for name, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("%s missing production gate %s", filepath.Base(path), name))
		}
	}
	return issues
}

func validateSurfaceProdTargets(path string, targets []surfaceProdTargetEvidence) []string {
	required := map[string]string{
		"linux-x64":   "prod",
		"wasm32-web":  "prod",
		"windows-x64": "beta",
		"macos-x64":   "beta",
	}
	if len(targets) == 0 {
		return []string{fmt.Sprintf("%s target tier evidence is required", filepath.Base(path))}
	}
	var issues []string
	seen := map[string]bool{}
	for _, target := range targets {
		name := strings.TrimSpace(target.Target)
		seen[name] = true
		wantTier, ok := required[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s unsupported target %s", filepath.Base(path), name))
			continue
		}
		if target.Tier != wantTier {
			issues = append(issues, fmt.Sprintf("%s target %s tier is %q, want %q", filepath.Base(path), name, target.Tier, wantTier))
		}
		if target.Skipped && target.Pass {
			issues = append(issues, fmt.Sprintf("%s skipped target %s counted as pass", filepath.Base(path), name))
		}
		if wantTier == "prod" && (!target.Ran || !target.Pass || target.Skipped) {
			issues = append(issues, fmt.Sprintf("%s production target %s must run and pass without skip", filepath.Base(path), name))
		}
	}
	for name := range required {
		if !seen[name] {
			issues = append(issues, fmt.Sprintf("%s missing target tier %s", filepath.Base(path), name))
		}
	}
	return issues
}

func validateSurfaceProdNegativeGuards(path string, guards surfaceProdNegativeGuards) []string {
	required := map[string]bool{
		"missing job":                    guards.MissingJobRejected,
		"continue-on-error":              guards.ContinueOnErrorRejected,
		"skipped target as pass":         guards.SkippedTargetAsPassRejected,
		"missing artifact hash manifest": guards.MissingArtifactHashManifestRejected,
	}
	var issues []string
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("%s %s rejection guard is required", filepath.Base(path), name))
		}
	}
	return issues
}

func validateSurfaceProdCases(path string, cases []surfaceProdCaseReport) []string {
	required := map[string]bool{
		"release-packages production gate job required": false,
		"no continue-on-error production jobs":          false,
		"skipped target counted as pass rejected":       false,
		"artifact hash manifest missing rejected":       false,
	}
	var issues []string
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if name == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, fmt.Sprintf("%s case name and kind are required", filepath.Base(path)))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("%s case %s must run and pass", filepath.Base(path), name))
		}
	}
	for name, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("%s missing case %s", filepath.Base(path), name))
		}
	}
	return issues
}

func validateReleaseSummaryFile(path string, scope string, expectedStatus string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateReleaseSummary(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	var report surface.ReleaseSummaryReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.ReleaseScope != scope {
		issues = append(issues, fmt.Sprintf("%s release_scope is %q, want %q", filepath.Base(path), report.ReleaseScope, scope))
	}
	if report.Status != expectedStatus {
		issues = append(issues, fmt.Sprintf("%s status is %q, want %q", filepath.Base(path), report.Status, expectedStatus))
	}
	return issues
}

func validateReleaseMorphGateFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var report surfaceMorphGateSummary
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: "tetra.surface.morph.gate.v1"},
		{field: "status", got: report.Status, want: "current"},
		{field: "release_scope", got: report.ReleaseScope, want: "surface-morph-experimental-linux-web"},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/morph-gate.sh"},
		{field: "source", got: report.Source, want: "examples/surface_morph_command_palette.tetra"},
		{field: "module", got: report.Module, want: "lib.core.morph"},
		{field: "schema_under_test", got: report.SchemaUnderTest, want: "tetra.surface.morph.v1"},
		{field: "dependency_gate", got: report.DependencyGate, want: "tetra.surface.block-system.gate.v1"},
		{field: "headless_report", got: report.HeadlessReport, want: "headless/surface-headless-morph.json"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s %s is %q, want %q", filepath.Base(path), check.field, check.got, check.want))
		}
	}
	if !report.SameCommitValidated {
		issues = append(issues, fmt.Sprintf("%s same_commit_validated must be true", filepath.Base(path)))
	}
	if !report.ArtifactHashesValidated {
		issues = append(issues, fmt.Sprintf("%s artifact_hashes_validated must be true", filepath.Base(path)))
	}
	if !stringListEqual(report.TargetEvidence, []string{"headless"}) {
		issues = append(issues, fmt.Sprintf("%s target_evidence must be [headless]", filepath.Base(path)))
	}
	if !stringListEqual(report.CorePrimitives, []string{"Block"}) {
		issues = append(issues, fmt.Sprintf("%s core_primitives must be [Block]", filepath.Base(path)))
	}
	if len(report.ForbiddenCorePrimitives) == 0 {
		issues = append(issues, fmt.Sprintf("%s forbidden_core_primitives must not be empty", filepath.Base(path)))
	}
	return issues
}

func validateReleaseMorphReportFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.Schema != surface.SchemaV1 {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", filepath.Base(path), report.Schema, surface.SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("%s status is %q, want pass", filepath.Base(path), report.Status))
	}
	if report.Target != "headless" {
		issues = append(issues, fmt.Sprintf("%s target is %q, want headless", filepath.Base(path), report.Target))
	}
	if report.Source != "examples/surface_morph_command_palette.tetra" {
		issues = append(issues, fmt.Sprintf("%s source is %q, want examples/surface_morph_command_palette.tetra", filepath.Base(path), report.Source))
	}
	if report.Morph == nil {
		issues = append(issues, fmt.Sprintf("%s requires morph evidence", filepath.Base(path)))
		return issues
	}
	if report.Morph.Schema != "tetra.surface.morph.v1" {
		issues = append(issues, fmt.Sprintf("%s morph schema is %q, want tetra.surface.morph.v1", filepath.Base(path), report.Morph.Schema))
	}
	if report.Morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		issues = append(issues, fmt.Sprintf("%s morph quality_level is %q, want deterministic-headless-morph-capsule-v1", filepath.Base(path), report.Morph.QualityLevel))
	}
	if report.Morph.Module != "lib.core.morph" {
		issues = append(issues, fmt.Sprintf("%s morph module is %q, want lib.core.morph", filepath.Base(path), report.Morph.Module))
	}
	if report.Morph.SurfaceScope != "surface-morph-experimental-linux-web" {
		issues = append(issues, fmt.Sprintf("%s morph surface_scope is %q, want surface-morph-experimental-linux-web", filepath.Base(path), report.Morph.SurfaceScope))
	}
	if report.Morph.ProductionClaim {
		issues = append(issues, fmt.Sprintf("%s morph production_claim must be false", filepath.Base(path)))
	}
	return issues
}

func validateReleaseTextInputFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateTextInputReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseRuntimeEnvelopeFile(path string, target string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var report surfaceReleaseRuntimeEnvelope
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.Schema != surface.SchemaV1 {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", filepath.Base(path), report.Schema, surface.SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("%s status is %q, want pass", filepath.Base(path), report.Status))
	}
	if report.Target != target {
		issues = append(issues, fmt.Sprintf("%s target is %q, want %q", filepath.Base(path), report.Target, target))
	}
	if report.Source != "examples/surface_release_form.tetra" {
		issues = append(issues, fmt.Sprintf("%s source is %q, want examples/surface_release_form.tetra", filepath.Base(path), report.Source))
	}
	if report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, fmt.Sprintf("%s must not claim user-facing platform widgets", filepath.Base(path)))
	}
	switch target {
	case "linux-x64":
		if report.HostEvidence.Level != "linux-x64-release-window-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.level is %q, want linux-x64-release-window-v1", filepath.Base(path), report.HostEvidence.Level))
		}
		if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.backend is %q, want wayland-shm-rgba-release-v1", filepath.Base(path), report.HostEvidence.Backend))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "real_window", ok: report.HostEvidence.RealWindow},
			{name: "native_input", ok: report.HostEvidence.NativeInput},
			{name: "text_input", ok: report.HostEvidence.TextInput},
			{name: "clipboard", ok: report.HostEvidence.Clipboard},
			{name: "composition", ok: report.HostEvidence.Composition},
			{name: "accessibility_bridge", ok: report.HostEvidence.AccessibilityBridge},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s host_evidence.%s must be true", filepath.Base(path), check.name))
			}
		}
	case "wasm32-web":
		if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", filepath.Base(path), report.HostEvidence.Level))
		}
		if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.backend is %q, want browser-canvas-rgba-accessible", filepath.Base(path), report.HostEvidence.Backend))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "browser_canvas", ok: report.HostEvidence.BrowserCanvas},
			{name: "browser_input", ok: report.HostEvidence.BrowserInput},
			{name: "browser_clipboard", ok: report.HostEvidence.BrowserClipboard},
			{name: "browser_composition", ok: report.HostEvidence.BrowserComposition},
			{name: "browser_accessibility_snapshot", ok: report.HostEvidence.BrowserAccessibilitySnapshot},
			{name: "browser_accessibility_mirror", ok: report.HostEvidence.BrowserAccessibilityMirror},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s host_evidence.%s must be true", filepath.Base(path), check.name))
			}
		}
	}
	return issues
}

func validateSurfaceArtifactHashes(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var manifest surfaceReleaseArtifactHashManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if manifest.Schema != surfaceArtifactHashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", filepath.Base(path), manifest.Schema, surfaceArtifactHashSchema))
	}
	if strings.TrimSpace(manifest.Root) == "" || filepath.IsAbs(manifest.Root) || strings.Contains(manifest.Root, "..") {
		issues = append(issues, fmt.Sprintf("%s root is unsafe or empty", filepath.Base(path)))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, fmt.Sprintf("%s artifacts must not be empty", filepath.Base(path)))
	}
	root := filepath.Join(filepath.Dir(path), filepath.FromSlash(manifest.Root))
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") {
			issues = append(issues, fmt.Sprintf("%s contains unsafe artifact path %q", filepath.Base(path), artifact.Path))
			continue
		}
		size, digest, err := hashFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s artifact %s read failed: %v", filepath.Base(path), artifact.Path, err))
			continue
		}
		if size != artifact.Size {
			issues = append(issues, fmt.Sprintf("%s artifact %s size = %d, want %d", filepath.Base(path), artifact.Path, size, artifact.Size))
		}
		if digest != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("%s artifact %s sha256 = %s, want %s", filepath.Base(path), artifact.Path, digest, artifact.SHA256))
		}
	}
	return issues
}

func validateSurfaceReleaseManifest(path string, scope string, expectedStatus string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("manifest %s read failed: %v", path, err)}
	}
	text := string(raw)
	var issues []string
	for _, want := range []string{
		scope,
		expectedStatus,
		"docs/spec/surface_v1.md",
		"docs/user/surface_guide.md",
		"docs/user/examples_index.md",
	} {
		if !strings.Contains(text, want) {
			issues = append(issues, fmt.Sprintf("manifest %s missing %q", path, want))
		}
	}
	var manifest struct {
		Features []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		issues = append(issues, fmt.Sprintf("manifest %s decode failed: %v", path, err))
		return issues
	}
	requiredSurfaceFeatures := map[string]string{
		"ui.surface-core":             expectedStatus,
		"ui.surface-block-system":     "experimental",
		"ui.surface-morph-capsule":    "experimental",
		"ui.surface-gpu":              "experimental",
		"ui.surface-headless":         expectedStatus,
		"ui.surface-linux-x64":        expectedStatus,
		"ui.surface-web-wasm":         expectedStatus,
		"ui.surface-component-model":  expectedStatus,
		"ui.surface-toolkit-v1":       expectedStatus,
		"ui.surface-text-input-v1":    expectedStatus,
		"ui.surface-accessibility-v1": expectedStatus,
		"ui.surface-macos-x64":        "unsupported",
		"ui.surface-windows-x64":      "unsupported",
		"ui.surface-wasm32-wasi":      "unsupported",
	}
	seen := map[string]string{}
	for _, feature := range manifest.Features {
		if _, ok := requiredSurfaceFeatures[feature.ID]; ok {
			seen[feature.ID] = feature.Status
		}
	}
	for id, wantStatus := range requiredSurfaceFeatures {
		if gotStatus, ok := seen[id]; !ok {
			issues = append(issues, fmt.Sprintf("manifest %s missing Surface release feature %s", path, id))
		} else if gotStatus != wantStatus {
			issues = append(issues, fmt.Sprintf("manifest %s Surface release feature %s status is %q, want %q", path, id, gotStatus, wantStatus))
		}
	}
	return issues
}

func stringListEqual(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func validSurfaceReleaseGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func hashFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return 0, "", err
	}
	return size, "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
