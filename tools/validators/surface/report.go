package surface

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != SchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, SchemaV1)
	}

	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectNonRuntimeEvidence(raw)...)
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "headless" && report.Target != "linux-x64" && report.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "surface-headless" && report.Runtime != "surface-linux-x64" && report.Runtime != "surface-wasm32-web" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want surface-headless, surface-linux-x64, or surface-wasm32-web", report.Runtime))
	}
	if report.SurfaceSchema != "tetra.surface.v1" {
		issues = append(issues, fmt.Sprintf("surface_schema is %q, want tetra.surface.v1", report.SurfaceSchema))
	}
	if report.HostABI != "tetra.surface.host-abi.v1" {
		issues = append(issues, fmt.Sprintf("host_abi is %q, want tetra.surface.host-abi.v1", report.HostABI))
	}
	issues = append(issues, validateHostEvidence(report)...)
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Source, report.Processes)...)
	issues = append(issues, validateArtifacts(report.Target, report.Source, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	componentIndex, componentIssues := validateComponents(report.Components)
	issues = append(issues, componentIssues...)
	issues = append(issues, validateSourceComponentModel(report.Source, report.Components)...)
	issues = append(issues, validateEvents(report.Events, componentIndex)...)
	issues = append(issues, validateFrames(report.Frames)...)
	issues = append(issues, validateStateTransitions(report.StateTransitions, componentIndex)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateTargetRuntimeEvidence(report)...)
	issues = append(issues, validateTextFocusInputEvidence(report, componentIndex)...)
	issues = append(issues, validateComponentTreeEvidence(report)...)
	issues = append(issues, validateBlockGraphEvidence(report)...)
	issues = append(issues, validateBlockCorePrimitiveEvidence(report)...)
	issues = append(issues, validateBlockPaintEvidence(report)...)
	issues = append(issues, validateBlockTextEvidence(report)...)
	issues = append(issues, validateBlockLayoutEvidence(report)...)
	issues = append(issues, validateBlockEventFocusEvidence(report)...)
	issues = append(issues, validateBlockStateEvidence(report)...)
	issues = append(issues, validateBlockMotionEvidence(report)...)
	issues = append(issues, validateBlockAssetEvidence(report)...)
	issues = append(issues, validateBlockAccessibilityEvidence(report)...)
	issues = append(issues, validateBlockSystemEvidence(report)...)
	issues = append(issues, validateMorphEvidence(report)...)
	issues = append(issues, validateProductionToolkitEvidence(report)...)
	issues = append(issues, validateBrowserReleaseEvidence(report)...)
	issues = append(issues, validateBrowserSurfaceEvidence(report)...)
	issues = append(issues, validateLinuxReleaseWindowEvidence(report)...)
	issues = append(issues, validateMinimalToolkitEvidence(report)...)
	issues = append(issues, validateAccessibilityTreeEvidence(report)...)
	issues = append(issues, validateAppModelEvidence(report)...)
	issues = append(issues, validateLinuxAppShellEvidence(report)...)
	issues = append(issues, validateSecurityPermissionEvidence(report)...)
	issues = append(issues, validateSurfacePerformanceBudgetEvidence(report)...)
	if report.SurfacePerformanceBudget != nil && !performanceBudgetPeakRSSFieldPresent(raw, true) {
		issues = append(issues, "surface_performance_budget memory peak_rss_bytes field is required")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReleaseSummary(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != ReleaseSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, ReleaseSchemaV1)
	}

	var report ReleaseSummaryReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectNonRuntimeEvidence(raw)...)
	if report.Schema != ReleaseSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, ReleaseSchemaV1))
	}
	if report.ReleaseScope != ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want %q", report.ReleaseScope, ReleaseScopeSurfaceV1LinuxWeb))
	}
	if report.Status != "current" {
		issues = append(issues, fmt.Sprintf("status is %q, want current", report.Status))
	}
	if !report.ProductionClaim {
		issues = append(issues, "production_claim must be true for Surface v1 release summaries")
	}
	if report.Experimental {
		issues = append(issues, "experimental must be false for Surface v1 release summaries")
	}
	if report.Producer != "scripts/release/surface/release-gate.sh" {
		issues = append(issues, fmt.Sprintf("producer is %q, want scripts/release/surface/release-gate.sh", report.Producer))
	}
	if !isGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character hex commit")
	}
	if strings.TrimSpace(report.Version) == "" {
		issues = append(issues, "version is required")
	}
	if strings.TrimSpace(report.HostOS) == "" {
		issues = append(issues, "host_os is required")
	}
	if strings.TrimSpace(report.HostArch) == "" {
		issues = append(issues, "host_arch is required")
	}
	if strings.TrimSpace(report.GeneratedAtUTC) == "" || !strings.HasSuffix(report.GeneratedAtUTC, "Z") || !strings.Contains(report.GeneratedAtUTC, "T") {
		issues = append(issues, "generated_at_utc must be an RFC3339 UTC timestamp")
	}
	if !strings.Contains(report.CommandLine, "scripts/release/surface/release-gate.sh") {
		issues = append(issues, "command_line must include scripts/release/surface/release-gate.sh")
	}
	issues = append(issues, validateExactStringList("supported_targets", report.SupportedTargets, []string{"headless", "linux-x64", "wasm32-web"})...)
	issues = append(issues, validateExactStringList("runtime_targets", report.RuntimeTargets, []string{"linux-x64", "wasm32-web"})...)
	issues = append(issues, validateExactStringList("test_targets", report.TestTargets, []string{"headless"})...)
	issues = append(issues, validateExactStringList("unsupported_targets", report.UnsupportedTargets, []string{"macos-x64", "windows-x64", "wasm32-wasi"})...)
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "host_abi", got: report.HostABI, want: "tetra.surface.host.v1"},
		{field: "toolkit", got: report.Toolkit, want: "production-widgets-v1"},
		{field: "text_input", got: report.TextInput, want: "production-text-input-v1"},
		{field: "clipboard", got: report.Clipboard, want: "clipboard-text-v1"},
		{field: "ime", got: report.IME, want: "composition-baseline-v1"},
		{field: "accessibility", got: report.Accessibility, want: "platform-bridge-v1"},
		{field: "app_model", got: report.AppModel, want: "explicit-command-reducer-v1"},
		{field: "linux_app_shell", got: report.LinuxAppShell, want: "linux-app-shell-subset-v1"},
		{field: "app_shell_features", got: report.AppShellFeatures, want: "electron-feature-ledger-v1"},
		{field: "security_permissions", got: report.SecurityPermissions, want: "surface-security-permission-v1"},
		{field: "performance_budget", got: report.PerformanceBudget, want: "surface-performance-budget-v1"},
		{field: "developer_fast_loop", got: report.DeveloperFastLoop, want: "surface-dev-workflow-v1"},
		{field: "inspector", got: report.Inspector, want: "surface-inspector-v1"},
		{field: "project_templates", got: report.ProjectTemplates, want: "surface-template-smoke-v1"},
		{field: "reference_apps", got: report.ReferenceApps, want: "surface-reference-app-suite-v1"},
		{field: "surface_package", got: report.SurfacePackage, want: "surface-package-v1"},
		{field: "crash_reporting", got: report.CrashReporting, want: "surface-crash-report-v1"},
		{field: "i18n_localization", got: report.I18nLocalization, want: "surface-i18n-v1"},
		{field: "widget_migration", got: report.WidgetMigration, want: "surface-widget-migration-v1"},
		{field: "browser_surface", got: report.BrowserSurface, want: "browser-canvas-release-v1"},
		{field: "linux_surface", got: report.LinuxSurface, want: "linux-x64-release-window-v1"},
		{field: "block_system", got: report.BlockSystem, want: "block-system"},
		{field: "block_system_gate", got: report.BlockSystemGate, want: "tetra.surface.block-system.gate.v1"},
		{field: "morph", got: report.Morph, want: "morph-capsule"},
		{field: "morph_gate", got: report.MorphGate, want: "tetra.surface.morph.gate.v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !report.ArtifactHashesValidated {
		issues = append(issues, "artifact_hashes_validated must be true")
	}
	if report.LegacySidecars {
		issues = append(issues, "legacy_sidecars must be false")
	}
	if report.DOMUI {
		issues = append(issues, "dom_ui must be false")
	}
	if report.UserJS {
		issues = append(issues, "user_js must be false")
	}
	if report.PlatformWidgets {
		issues = append(issues, "platform_widgets must be false")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}
