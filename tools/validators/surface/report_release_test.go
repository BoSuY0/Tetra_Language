package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSurfaceReleaseSummaryAcceptsScopedLinuxWebCurrent(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf("ValidateReleaseSummary failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceReleaseSummaryAcceptsBlockSystemAndMorphGateMetadata(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf("ValidateReleaseSummary failed with Block-system/Morph gate metadata: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresProjectTemplates(t *testing.T) {
	withTemplates := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withTemplates)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected project template evidence: %v\n%s", err, withTemplates)
	}

	missing := strings.Replace(withTemplates, `  "project_templates": "surface-template-smoke-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing project_templates to fail")
	}
	if !strings.Contains(err.Error(), "project_templates") {
		t.Fatalf("error = %v, want project_templates diagnostic", err)
	}

	wrong := strings.Replace(withTemplates, `"project_templates": "surface-template-smoke-v1"`, `"project_templates": "docs-only-template-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong project_templates to fail")
	}
	if !strings.Contains(err.Error(), "project_templates") {
		t.Fatalf("error = %v, want project_templates diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresReferenceApps(t *testing.T) {
	withReferenceApps := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withReferenceApps)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected reference app evidence: %v\n%s", err, withReferenceApps)
	}

	missing := strings.Replace(withReferenceApps, `  "reference_apps": "surface-reference-app-suite-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing reference_apps to fail")
	}
	if !strings.Contains(err.Error(), "reference_apps") {
		t.Fatalf("error = %v, want reference_apps diagnostic", err)
	}

	wrong := strings.Replace(withReferenceApps, `"reference_apps": "surface-reference-app-suite-v1"`, `"reference_apps": "docs-only-reference-app-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong reference_apps to fail")
	}
	if !strings.Contains(err.Error(), "reference_apps") {
		t.Fatalf("error = %v, want reference_apps diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresSurfacePackage(t *testing.T) {
	withPackage := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withPackage)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected Surface package evidence: %v\n%s", err, withPackage)
	}

	missing := strings.Replace(withPackage, `  "surface_package": "surface-package-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing surface_package to fail")
	}
	if !strings.Contains(err.Error(), "surface_package") {
		t.Fatalf("error = %v, want surface_package diagnostic", err)
	}

	wrong := strings.Replace(withPackage, `"surface_package": "surface-package-v1"`, `"surface_package": "docs-only-package-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong surface_package to fail")
	}
	if !strings.Contains(err.Error(), "surface_package") {
		t.Fatalf("error = %v, want surface_package diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresCrashReporting(t *testing.T) {
	withCrashReporting := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withCrashReporting)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected crash reporting evidence: %v\n%s", err, withCrashReporting)
	}

	missing := strings.Replace(withCrashReporting, `  "crash_reporting": "surface-crash-report-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing crash_reporting to fail")
	}
	if !strings.Contains(err.Error(), "crash_reporting") {
		t.Fatalf("error = %v, want crash_reporting diagnostic", err)
	}

	wrong := strings.Replace(withCrashReporting, `"crash_reporting": "surface-crash-report-v1"`, `"crash_reporting": "docs-only-crash-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong crash_reporting to fail")
	}
	if !strings.Contains(err.Error(), "crash_reporting") {
		t.Fatalf("error = %v, want crash_reporting diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresI18nLocalization(t *testing.T) {
	withI18n := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withI18n)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected i18n localization evidence: %v\n%s", err, withI18n)
	}

	missing := strings.Replace(withI18n, `  "i18n_localization": "surface-i18n-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing i18n_localization to fail")
	}
	if !strings.Contains(err.Error(), "i18n_localization") {
		t.Fatalf("error = %v, want i18n_localization diagnostic", err)
	}

	wrong := strings.Replace(withI18n, `"i18n_localization": "surface-i18n-v1"`, `"i18n_localization": "full-icu-bidi-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong i18n_localization to fail")
	}
	if !strings.Contains(err.Error(), "i18n_localization") {
		t.Fatalf("error = %v, want i18n_localization diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresWidgetMigration(t *testing.T) {
	withMigration := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withMigration)); err != nil {
		t.Fatalf("ValidateReleaseSummary rejected widget migration evidence: %v\n%s", err, withMigration)
	}

	missing := strings.Replace(withMigration, `  "widget_migration": "surface-widget-migration-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing widget_migration to fail")
	}
	if !strings.Contains(err.Error(), "widget_migration") {
		t.Fatalf("error = %v, want widget_migration diagnostic", err)
	}

	wrong := strings.Replace(withMigration, `"widget_migration": "surface-widget-migration-v1"`, `"widget_migration": "future-widget-core-claim"`, 1)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong widget_migration to fail")
	}
	if !strings.Contains(err.Error(), "widget_migration") {
		t.Fatalf("error = %v, want widget_migration diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRejectsFakePromotionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "missing unsupported targets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
`, ``, 1)
			},
			want: "unsupported_targets",
		},
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "unsupported target in supported targets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"supported_targets": ["headless", "linux-x64", "wasm32-web"]`, `"supported_targets": ["headless", "linux-x64", "wasm32-web", "macos-x64"]`, 1)
			},
			want: "supported_targets",
		},
		{
			name: "dom ui",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"dom_ui": false`, `"dom_ui": true`, 1)
			},
			want: "dom_ui",
		},
		{
			name: "platform widgets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"platform_widgets": false`, `"platform_widgets": true`, 1)
			},
			want: "platform_widgets",
		},
		{
			name: "missing block system",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "block_system": "block-system",
`, ``, 1)
			},
			want: "block_system",
		},
		{
			name: "wrong block system gate",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"block_system_gate": "tetra.surface.block-system.gate.v1"`, `"block_system_gate": "tetra.surface.block-system.fake"`, 1)
			},
			want: "block_system_gate",
		},
		{
			name: "missing morph",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "morph": "morph-capsule",
`, ``, 1)
			},
			want: "morph",
		},
		{
			name: "wrong morph gate",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"morph_gate": "tetra.surface.morph.gate.v1"`, `"morph_gate": "tetra.surface.morph.invalid"`, 1)
			},
			want: "morph_gate",
		},
		{
			name: "missing app model",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "app_model": "explicit-command-reducer-v1",
`, ``, 1)
			},
			want: "app_model",
		},
		{
			name: "wrong app model",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"app_model": "explicit-command-reducer-v1"`, `"app_model": "hidden-state-runtime"`, 1)
			},
			want: "app_model",
		},
		{
			name: "missing linux app shell",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "linux_app_shell": "linux-app-shell-subset-v1",
  "app_shell_features": "electron-feature-ledger-v1",
`, ``, 1)
			},
			want: "linux_app_shell",
		},
		{
			name: "wrong linux app shell",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"linux_app_shell": "linux-app-shell-subset-v1"`, `"linux_app_shell": "native-widget-shell"`, 1)
			},
			want: "linux_app_shell",
		},
		{
			name: "missing security permissions",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "security_permissions": "surface-security-permission-v1",
`, ``, 1)
			},
			want: "security_permissions",
		},
		{
			name: "wrong security permissions",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"security_permissions": "surface-security-permission-v1"`, `"security_permissions": "ambient-network-filesystem"`, 1)
			},
			want: "security_permissions",
		},
		{
			name: "missing performance budget",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "performance_budget": "surface-performance-budget-v1",
`, ``, 1)
			},
			want: "performance_budget",
		},
		{
			name: "wrong performance budget",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"performance_budget": "surface-performance-budget-v1"`, `"performance_budget": "faster-than-electron"`, 1)
			},
			want: "performance_budget",
		},
		{
			name: "missing developer fast loop",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "developer_fast_loop": "surface-dev-workflow-v1",
`, ``, 1)
			},
			want: "developer_fast_loop",
		},
		{
			name: "wrong developer fast loop",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"developer_fast_loop": "surface-dev-workflow-v1"`, `"developer_fast_loop": "hot-reload-claim"`, 1)
			},
			want: "developer_fast_loop",
		},
		{
			name: "missing inspector",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "inspector": "surface-inspector-v1",
`, ``, 1)
			},
			want: "inspector",
		},
		{
			name: "wrong inspector",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"inspector": "surface-inspector-v1"`, `"inspector": "browser-devtools-proxy"`, 1)
			},
			want: "inspector",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceReleaseSummaryJSON())))
			err := ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected release summary to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceReleaseSummaryRejectsStaleProducerMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing producer",
			mutate: func(report map[string]any) {
				delete(report, "producer")
			},
			want: "producer",
		},
		{
			name: "stale git head",
			mutate: func(report map[string]any) {
				report["git_head"] = "unknown"
			},
			want: "git_head",
		},
		{
			name: "missing command line",
			mutate: func(report map[string]any) {
				delete(report, "command_line")
			},
			want: "command_line",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var report map[string]any
			if err := json.Unmarshal(validSurfaceReleaseSummaryJSON(), &report); err != nil {
				t.Fatalf("decode release summary: %v", err)
			}
			tc.mutate(report)
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal release summary: %v", err)
			}
			err = ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected stale producer metadata to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceTargetHostStatusAcceptsWindowsUnsupportedNonclaim(t *testing.T) {
	raw := validSurfaceWindowsTargetHostStatusJSON()
	if err := ValidateTargetHostStatus(raw); err != nil {
		t.Fatalf("ValidateTargetHostStatus failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTargetHostStatusAcceptsMacOSUnsupportedNonclaim(t *testing.T) {
	raw := validSurfaceMacOSTargetHostStatusJSON()
	if err := ValidateTargetHostStatus(raw); err != nil {
		t.Fatalf("ValidateTargetHostStatus failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsFakeBuildOnlyPromotion(t *testing.T) {
	raw := strings.Replace(string(validSurfaceWindowsTargetHostStatusJSON()), `"build_only_promotion": false`, `"build_only_promotion": true`, 1)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected build-only promotion to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "build-only") {
		t.Fatalf("error = %v, want build-only diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsLinuxSubstitute(t *testing.T) {
	raw := strings.Replace(string(validSurfaceWindowsTargetHostStatusJSON()), `"linux_substitute": false`, `"linux_substitute": true`, 1)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected Linux substitute to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "linux substitute") {
		t.Fatalf("error = %v, want Linux substitute diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsProductionClaim(t *testing.T) {
	raw := strings.Replace(string(validSurfaceWindowsTargetHostStatusJSON()), `"production_claim": false`, `"production_claim": true`, 1)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected Windows production claim to fail without target-host evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "production_claim") {
		t.Fatalf("error = %v, want production_claim diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsBetaWithoutTargetHostEvidence(t *testing.T) {
	raw := strings.Replace(string(validSurfaceWindowsTargetHostStatusJSON()), `"status": "unsupported"`, `"status": "beta_target_host"`, 1)
	raw = strings.Replace(raw, `"tier": "UNSUPPORTED"`, `"tier": "BETA_TARGET_HOST"`, 1)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected beta target-host status without target-host evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "target-host") {
		t.Fatalf("error = %v, want target-host diagnostic", err)
	}
}
func validSurfaceReleaseSummaryJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "version": "tetra_language",
  "git_dirty": false,
  "host_os": "linux",
  "host_arch": "amd64",
  "generated_at_utc": "2026-06-08T16:00:00Z",
  "command_line": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
  "supported_targets": ["headless", "linux-x64", "wasm32-web"],
  "runtime_targets": ["linux-x64", "wasm32-web"],
  "test_targets": ["headless"],
  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "app_model": "explicit-command-reducer-v1",
  "linux_app_shell": "linux-app-shell-subset-v1",
  "app_shell_features": "electron-feature-ledger-v1",
  "security_permissions": "surface-security-permission-v1",
  "performance_budget": "surface-performance-budget-v1",
  "developer_fast_loop": "surface-dev-workflow-v1",
  "inspector": "surface-inspector-v1",
  "project_templates": "surface-template-smoke-v1",
  "reference_apps": "surface-reference-app-suite-v1",
  "surface_package": "surface-package-v1",
  "crash_reporting": "surface-crash-report-v1",
  "i18n_localization": "surface-i18n-v1",
  "widget_migration": "surface-widget-migration-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`)
}
func validSurfaceWindowsTargetHostStatusJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.target-host-status.v1",
  "target": "windows-x64",
  "status": "unsupported",
  "tier": "UNSUPPORTED",
  "release_scope": "surface-v1-linux-web",
  "source": "scripts/release/surface/release-gate.sh",
  "host_os": "linux",
  "host_arch": "amd64",
  "reason": "no Windows target-host Surface v1 runner evidence exists in this release",
  "production_claim": false,
  "experimental": false,
  "target_host_evidence": false,
  "build_only_evidence": false,
  "build_only_promotion": false,
  "linux_substitute": false,
  "ci_artifact_required": true,
  "required_evidence": {
    "real_window": false,
    "native_input": false,
    "clipboard": false,
    "dpi_scaling": false,
    "accessibility_snapshot": false,
    "app_shell": false
  },
  "unsupported_claims": [
    "windows-real-window-surface",
    "windows-production-surface-nonclaim",
    "windows-target-host-runtime",
    "build-only-windows-surface-runtime",
    "linux-substitute-windows-surface-runtime"
  ],
  "negative_guards": {
    "no_linux_substitute": true,
    "no_build_only_promotion": true,
    "no_production_claim": true,
    "no_docs_only_evidence": true,
    "no_copied_report": true,
    "ci_artifact_required": true
  }
}`)
}
func validSurfaceMacOSTargetHostStatusJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.target-host-status.v1",
  "target": "macos-x64",
  "status": "unsupported",
  "tier": "UNSUPPORTED",
  "release_scope": "surface-v1-linux-web",
  "source": "scripts/release/surface/release-gate.sh",
  "host_os": "linux",
  "host_arch": "amd64",
  "reason": "no macOS target-host Surface v1 runner evidence exists in this release",
  "production_claim": false,
  "experimental": false,
  "target_host_evidence": false,
  "build_only_evidence": false,
  "build_only_promotion": false,
  "linux_substitute": false,
  "ci_artifact_required": true,
  "required_evidence": {
    "real_window": false,
    "native_input": false,
    "clipboard": false,
    "dpi_scaling": false,
    "accessibility_snapshot": false,
    "app_shell": false
  },
  "unsupported_claims": [
    "macos-real-window-surface",
    "macos-production-surface-nonclaim",
    "macos-target-host-runtime",
    "build-only-macos-surface-runtime",
    "linux-substitute-macos-surface-runtime"
  ],
  "negative_guards": {
    "no_linux_substitute": true,
    "no_build_only_promotion": true,
    "no_production_claim": true,
    "no_docs_only_evidence": true,
    "no_copied_report": true,
    "ci_artifact_required": true
  }
}`)
}
