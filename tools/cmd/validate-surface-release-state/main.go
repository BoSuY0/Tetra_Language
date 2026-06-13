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
	Schema                   string                                  `json:"schema"`
	Status                   string                                  `json:"status"`
	Target                   string                                  `json:"target"`
	Source                   string                                  `json:"source"`
	HostEvidence             surface.HostEvidenceReport              `json:"host_evidence"`
	LinuxAppShell            *surface.LinuxAppShellReport            `json:"linux_app_shell,omitempty"`
	SecurityPermissions      *surface.SecurityPermissionReport       `json:"security_permissions,omitempty"`
	SurfacePerformanceBudget *surface.SurfacePerformanceBudgetReport `json:"surface_performance_budget,omitempty"`
	BrowserSurface           *surface.BrowserSurfaceReport           `json:"browser_surface,omitempty"`
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
	if scope != surface.ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q", scope, surface.ReleaseScopeSurfaceV1LinuxWeb))
	}
	issues = append(issues, validateReleaseSummaryFile(filepath.Join(reportDir, "surface-release-summary.json"), scope, expectedStatus)...)
	issues = append(issues, validateReleaseTextInputFile(filepath.Join(reportDir, "surface-headless-release-text-input.json"))...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-wasm32-web-release-browser.json"), "wasm32-web")...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-linux-x64-release-window.json"), "linux-x64")...)
	issues = append(issues, validateReleaseLinuxAppShellEnvelopeFile(filepath.Join(reportDir, "surface-linux-x64-release-app-shell.json"))...)
	issues = append(issues, validateReleaseDevWorkflowFile(filepath.Join(reportDir, "surface-dev-workflow.json"))...)
	issues = append(issues, validateReleaseInspectorFile(filepath.Join(reportDir, "surface-inspector.json"))...)
	issues = append(issues, validateReleaseTemplateSmokeFile(filepath.Join(reportDir, "surface-template-smoke.json"))...)
	issues = append(issues, validateReleaseReferenceAppsFile(filepath.Join(reportDir, "surface-reference-apps.json"))...)
	issues = append(issues, validateReleaseSurfacePackageFile(filepath.Join(reportDir, "surface-package.json"))...)
	issues = append(issues, validateReleaseCrashReportFile(filepath.Join(reportDir, "surface-crash-report.json"))...)
	issues = append(issues, validateReleaseI18nFile(filepath.Join(reportDir, "surface-i18n.json"))...)
	issues = append(issues, validateReleaseWidgetMigrationFile(filepath.Join(reportDir, "surface-widget-migration.json"))...)
	issues = append(issues, validateReleaseTargetHostStatusFile(filepath.Join(reportDir, "surface-macos-x64-target-host-status.json"), "macos-x64", "unsupported")...)
	issues = append(issues, validateReleaseTargetHostStatusFile(filepath.Join(reportDir, "surface-windows-x64-target-host-status.json"), "windows-x64", "unsupported")...)
	issues = append(issues, validateReleaseMorphGateFile(filepath.Join(reportDir, "morph", "surface-morph-gate-summary.json"))...)
	issues = append(issues, validateReleaseMorphReportFile(filepath.Join(reportDir, "morph", "headless", "surface-headless-morph.json"))...)
	issues = append(issues, validateSurfaceArtifactHashes(filepath.Join(reportDir, "artifact-hashes.json"))...)
	issues = append(issues, validateSurfaceReleaseManifest(opt.ManifestPath, scope, expectedStatus)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
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

func validateReleaseReferenceAppsFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateReferenceAppsReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseSurfacePackageFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidatePackageReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseCrashReportFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateCrashReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseI18nFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateI18nReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseWidgetMigrationFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateWidgetMigrationReport(raw); err != nil {
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
		if report.BrowserSurface == nil {
			issues = append(issues, fmt.Sprintf("%s requires browser_surface evidence", filepath.Base(path)))
			break
		}
		if report.BrowserSurface.Schema != surface.BrowserSurfaceSchemaV1 {
			issues = append(issues, fmt.Sprintf("%s browser_surface schema is %q, want %q", filepath.Base(path), report.BrowserSurface.Schema, surface.BrowserSurfaceSchemaV1))
		}
		if report.BrowserSurface.BrowserSurfaceLevel != "browser-canvas-release-v1" {
			issues = append(issues, fmt.Sprintf("%s browser_surface browser_surface_level is %q, want browser-canvas-release-v1", filepath.Base(path), report.BrowserSurface.BrowserSurfaceLevel))
		}
		if !report.BrowserSurface.DOMHostCanvasOnly {
			issues = append(issues, fmt.Sprintf("%s browser_surface dom_host_canvas_only must be true", filepath.Base(path)))
		}
		guards := report.BrowserSurface.NegativeGuards
		if !guards.NoDOMAppUITree || !guards.NoUserJSAppLogic || !guards.NoNodeOnlyPromotion {
			issues = append(issues, fmt.Sprintf("%s browser_surface must reject DOM-authored app UI tree, user JavaScript app logic, and Node-only promotion", filepath.Base(path)))
		}
	}
	return issues
}

func validateReleaseLinuxAppShellEnvelopeFile(path string) []string {
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
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("%s target is %q, want linux-x64", filepath.Base(path), report.Target))
	}
	if report.Source != "examples/surface_linux_app_shell_notes.tetra" {
		issues = append(issues, fmt.Sprintf("%s source is %q, want examples/surface_linux_app_shell_notes.tetra", filepath.Base(path), report.Source))
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" {
		issues = append(issues, fmt.Sprintf("%s host_evidence.level is %q, want linux-x64-release-window-v1", filepath.Base(path), report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
		issues = append(issues, fmt.Sprintf("%s host_evidence.backend is %q, want wayland-shm-rgba-release-v1", filepath.Base(path), report.HostEvidence.Backend))
	}
	if report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, fmt.Sprintf("%s must not claim GTK/Qt/native widget UI", filepath.Base(path)))
	}
	if report.LinuxAppShell == nil {
		return append(issues, fmt.Sprintf("%s requires linux_app_shell evidence", filepath.Base(path)))
	}
	if report.LinuxAppShell.Schema != surface.LinuxAppShellSchemaV1 {
		issues = append(issues, fmt.Sprintf("%s linux_app_shell schema is %q, want %q", filepath.Base(path), report.LinuxAppShell.Schema, surface.LinuxAppShellSchemaV1))
	}
	if report.LinuxAppShell.AppShellLevel != "linux-app-shell-subset-v1" {
		issues = append(issues, fmt.Sprintf("%s linux_app_shell app_shell_level is %q, want linux-app-shell-subset-v1", filepath.Base(path), report.LinuxAppShell.AppShellLevel))
	}
	if report.LinuxAppShell.NegativeGuards.NoGTK == false || report.LinuxAppShell.NegativeGuards.NoQT == false || report.LinuxAppShell.NegativeGuards.NoNativeWidgets == false {
		issues = append(issues, fmt.Sprintf("%s linux_app_shell must reject GTK/Qt/native widget UI", filepath.Base(path)))
	}
	issues = append(issues, validateReleaseLinuxAppShellFeatureLedger(filepath.Base(path), report.LinuxAppShell.ShellFeatures)...)
	if report.SecurityPermissions == nil {
		issues = append(issues, fmt.Sprintf("%s requires security_permissions evidence", filepath.Base(path)))
	} else if err := surface.ValidateSecurityPermissionReport(raw); err != nil {
		issues = append(issues, fmt.Sprintf("%s security_permissions invalid: %v", filepath.Base(path), err))
	}
	if report.SurfacePerformanceBudget == nil {
		issues = append(issues, fmt.Sprintf("%s requires surface_performance_budget evidence", filepath.Base(path)))
	} else if err := surface.ValidatePerformanceBudgetReport(raw); err != nil {
		issues = append(issues, fmt.Sprintf("%s surface_performance_budget invalid: %v", filepath.Base(path), err))
	}
	return issues
}

func validateReleaseDevWorkflowFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateDevWorkflowReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseInspectorFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateInspectorReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseTemplateSmokeFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateTemplateSmokeReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseLinuxAppShellFeatureLedger(base string, rows []surface.LinuxAppShellFeatureReport) []string {
	features := map[string]surface.LinuxAppShellFeatureReport{}
	for _, row := range rows {
		features[row.Name] = row
	}
	var issues []string
	for _, name := range []string{"window_lifecycle", "multi_window", "clipboard", "ime", "accessibility_bridge"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell shell_features missing %s", base, name))
			continue
		}
		if feature.Status != "target_evidenced" || !feature.Claimed {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell %s must be target_evidenced and claimed", base, name))
		}
	}
	for _, name := range []string{"app_menu", "crash_recovery", "error_report"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell shell_features missing %s", base, name))
			continue
		}
		if feature.Status != "scoped_adapter" || !feature.Claimed {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell %s must be scoped_adapter and claimed", base, name))
		}
	}
	for _, name := range []string{"dialog", "file_dialog", "file_picker", "notification", "tray", "deep_link"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell shell_features missing %s", base, name))
			continue
		}
		if feature.Status != "blocked_pass" || feature.Claimed || strings.TrimSpace(feature.BlockedReason) == "" {
			issues = append(issues, fmt.Sprintf("%s linux_app_shell %s must be blocked_pass nonclaim", base, name))
		}
	}
	return issues
}

func validateReleaseTargetHostStatusFile(path string, target string, status string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateTargetHostStatus(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	var report surface.TargetHostStatusReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.Target != target {
		issues = append(issues, fmt.Sprintf("%s target is %q, want %q", filepath.Base(path), report.Target, target))
	}
	if report.Status != status {
		issues = append(issues, fmt.Sprintf("%s status is %q, want %q", filepath.Base(path), report.Status, status))
	}
	if status == "unsupported" && report.Tier != "UNSUPPORTED" {
		issues = append(issues, fmt.Sprintf("%s tier is %q, want UNSUPPORTED", filepath.Base(path), report.Tier))
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
		"ui.surface-core":                   expectedStatus,
		"ui.surface-headless":               expectedStatus,
		"ui.surface-linux-x64":              expectedStatus,
		"ui.surface-web-wasm":               expectedStatus,
		"ui.surface-component-model":        expectedStatus,
		"ui.surface-toolkit-v1":             expectedStatus,
		"ui.surface-text-input-v1":          expectedStatus,
		"ui.surface-accessibility-v1":       expectedStatus,
		"ui.surface-inspector-v1":           expectedStatus,
		"ui.surface-project-templates-v1":   expectedStatus,
		"ui.surface-reference-app-suite-v1": expectedStatus,
		"ui.surface-packaging-v1":           expectedStatus,
		"ui.surface-crash-reporting-v1":     expectedStatus,
		"ui.surface-i18n-v1":                expectedStatus,
		"ui.surface-widget-migration-v1":    expectedStatus,
		"ui.surface-morph-capsule":          "experimental",
		"ui.surface-macos-x64":              "unsupported",
		"ui.surface-windows-x64":            "unsupported",
		"ui.surface-wasm32-wasi":            "unsupported",
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
