package surface

import (
	"errors"
	"fmt"
	"strings"
)

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
