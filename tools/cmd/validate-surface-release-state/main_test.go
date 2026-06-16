package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateSurfaceReleaseStateAcceptsCurrentLinuxWebScope(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err != nil {
		t.Fatalf("validateSurfaceReleaseState failed: %v", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsManifestMissingSurfaceFeatureRegistry(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "surface_release": {
    "scope": "surface-v1-linux-web",
    "status": "current"
  },
  "docs": [
    "docs/spec/surface_v1.md",
    "docs/user/surface_guide.md",
    "docs/user/examples_index.md"
  ]
}`+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected manifest without Surface release feature IDs to fail")
	}
	if !strings.Contains(err.Error(), "ui.surface-core") {
		t.Fatalf("error = %v, want missing Surface release feature diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingLinuxReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	if err := os.Remove(filepath.Join(dir, "surface-linux-x64-release-window.json")); err != nil {
		t.Fatalf("remove linux report: %v", err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"surface_release":{"scope":"surface-v1-linux-web","status":"current"},"docs":["docs/spec/surface_v1.md","docs/user/surface_guide.md","docs/user/examples_index.md"]}`+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing linux release report to fail")
	}
	if !strings.Contains(err.Error(), "surface-linux-x64-release-window.json") {
		t.Fatalf("error = %v, want missing linux report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingLinuxAppShellReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	if err := os.Remove(filepath.Join(dir, "surface-linux-x64-release-app-shell.json")); err != nil {
		t.Fatalf("remove linux app-shell report: %v", err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing linux app-shell release report to fail")
	}
	if !strings.Contains(err.Error(), "surface-linux-x64-release-app-shell.json") {
		t.Fatalf("error = %v, want missing linux app-shell report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP16AppShellFeatureLedger(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	files["surface-linux-x64-release-app-shell.json"] = strings.Replace(surfaceReleaseStateLinuxAppShellReportJSONWithSecurityAndPerformance(), `,{"name":"error_report","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true}`, ``, 1)
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P16 app-shell feature ledger to fail")
	}
	if !strings.Contains(err.Error(), "error_report") {
		t.Fatalf("error = %v, want error_report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP17SecurityPermissions(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	files["surface-linux-x64-release-app-shell.json"] = strings.Replace(surfaceReleaseStateLinuxAppShellReportJSONWithSecurityAndPerformance(), `"security_permissions":`, `"security_permissions_removed":`, 1)
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P17 security permissions to fail")
	}
	if !strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP18PerformanceBudget(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	files["surface-linux-x64-release-app-shell.json"] = strings.Replace(surfaceReleaseStateLinuxAppShellReportJSONWithSecurityAndPerformance(), `"surface_performance_budget":`, `"surface_performance_budget_removed":`, 1)
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P18 performance budget to fail")
	}
	if !strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want surface_performance_budget diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP19DeveloperFastLoop(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-dev-workflow.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P19 developer fast loop report to fail")
	}
	if !strings.Contains(err.Error(), "surface-dev-workflow.json") {
		t.Fatalf("error = %v, want surface-dev-workflow.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP20Inspector(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-inspector.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P20 inspector report to fail")
	}
	if !strings.Contains(err.Error(), "surface-inspector.json") {
		t.Fatalf("error = %v, want surface-inspector.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP21ProjectTemplates(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-template-smoke.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P21 project template smoke report to fail")
	}
	if !strings.Contains(err.Error(), "surface-template-smoke.json") {
		t.Fatalf("error = %v, want surface-template-smoke.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP22ReferenceApps(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-reference-apps.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P22 reference app suite report to fail")
	}
	if !strings.Contains(err.Error(), "surface-reference-apps.json") {
		t.Fatalf("error = %v, want surface-reference-apps.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP23SurfacePackage(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-package.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P23 Surface package report to fail")
	}
	if !strings.Contains(err.Error(), "surface-package.json") {
		t.Fatalf("error = %v, want surface-package.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP24CrashReport(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-crash-report.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P24 crash report to fail")
	}
	if !strings.Contains(err.Error(), "surface-crash-report.json") {
		t.Fatalf("error = %v, want surface-crash-report.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsManifestMissingP24CrashReportingFeature(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest := strings.Replace(surfaceReleaseStateManifestJSON(), `    {"id":"ui.surface-crash-reporting-v1","status":"current"},
`, "", 1)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected manifest without P24 crash reporting feature to fail")
	}
	if !strings.Contains(err.Error(), "ui.surface-crash-reporting-v1") {
		t.Fatalf("error = %v, want ui.surface-crash-reporting-v1 diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP25I18nReport(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-i18n.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P25 i18n report to fail")
	}
	if !strings.Contains(err.Error(), "surface-i18n.json") {
		t.Fatalf("error = %v, want surface-i18n.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsManifestMissingP25I18nFeature(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest := strings.Replace(surfaceReleaseStateManifestJSON(), `    {"id":"ui.surface-i18n-v1","status":"current"},
`, "", 1)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected manifest without P25 i18n feature to fail")
	}
	if !strings.Contains(err.Error(), "ui.surface-i18n-v1") {
		t.Fatalf("error = %v, want ui.surface-i18n-v1 diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingP26WidgetMigrationReport(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-widget-migration.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing P26 widget migration report to fail")
	}
	if !strings.Contains(err.Error(), "surface-widget-migration.json") {
		t.Fatalf("error = %v, want surface-widget-migration.json diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsManifestMissingP26WidgetMigrationFeature(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest := strings.Replace(surfaceReleaseStateManifestJSON(), `    {"id":"ui.surface-widget-migration-v1","status":"current"},
`, "", 1)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected manifest without P26 widget migration feature to fail")
	}
	if !strings.Contains(err.Error(), "ui.surface-widget-migration-v1") {
		t.Fatalf("error = %v, want ui.surface-widget-migration-v1 diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingWindowsTargetHostStatusReport(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-windows-x64-target-host-status.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing Windows target-host status report to fail")
	}
	if !strings.Contains(err.Error(), "surface-windows-x64-target-host-status.json") {
		t.Fatalf("error = %v, want missing Windows target-host status diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsWindowsBuildOnlyPromotion(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	files["surface-windows-x64-target-host-status.json"] = strings.Replace(files["surface-windows-x64-target-host-status.json"], `"build_only_promotion": false`, `"build_only_promotion": true`, 1)
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected Windows build-only promotion to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "build-only") {
		t.Fatalf("error = %v, want build-only diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingMacOSTargetHostStatusReport(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	delete(files, "surface-macos-x64-target-host-status.json")
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing macOS target-host status report to fail")
	}
	if !strings.Contains(err.Error(), "surface-macos-x64-target-host-status.json") {
		t.Fatalf("error = %v, want missing macOS target-host status diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMacOSBuildOnlyPromotion(t *testing.T) {
	dir := t.TempDir()
	files := surfaceReleaseStateFixtureFiles()
	files["surface-macos-x64-target-host-status.json"] = strings.Replace(files["surface-macos-x64-target-host-status.json"], `"build_only_promotion": false`, `"build_only_promotion": true`, 1)
	writeSurfaceReleaseStateFixtureFiles(t, dir, files)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected macOS build-only promotion to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "build-only") {
		t.Fatalf("error = %v, want build-only diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingMorphReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	if err := os.Remove(filepath.Join(dir, "morph", "headless", "surface-headless-morph.json")); err != nil {
		t.Fatalf("remove morph report: %v", err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing morph report to fail")
	}
	if !strings.Contains(err.Error(), "surface-headless-morph.json") {
		t.Fatalf("error = %v, want missing morph report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMorphManifestPromotion(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest := strings.Replace(surfaceReleaseStateManifestJSON(), `{"id":"ui.surface-morph-capsule","status":"experimental"}`, `{"id":"ui.surface-morph-capsule","status":"current"}`, 1)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected Morph Capsule manifest promotion to fail")
	}
	if !strings.Contains(err.Error(), `ui.surface-morph-capsule status is "current", want "experimental"`) {
		t.Fatalf("error = %v, want Morph Capsule experimental diagnostic", err)
	}
}

func surfaceReleaseStateManifestJSON() string {
	return `{
  "surface_release": {
    "scope": "surface-v1-linux-web",
    "status": "current"
  },
  "docs": [
    "docs/spec/surface_v1.md",
    "docs/user/surface_guide.md",
    "docs/user/examples_index.md"
  ],
  "features": [
    {"id":"ui.surface-core","status":"current"},
    {"id":"ui.surface-headless","status":"current"},
    {"id":"ui.surface-linux-x64","status":"current"},
    {"id":"ui.surface-web-wasm","status":"current"},
    {"id":"ui.surface-component-model","status":"current"},
    {"id":"ui.surface-toolkit-v1","status":"current"},
    {"id":"ui.surface-text-input-v1","status":"current"},
    {"id":"ui.surface-accessibility-v1","status":"current"},
    {"id":"ui.surface-inspector-v1","status":"current"},
    {"id":"ui.surface-project-templates-v1","status":"current"},
    {"id":"ui.surface-reference-app-suite-v1","status":"current"},
    {"id":"ui.surface-packaging-v1","status":"current"},
    {"id":"ui.surface-crash-reporting-v1","status":"current"},
    {"id":"ui.surface-i18n-v1","status":"current"},
    {"id":"ui.surface-widget-migration-v1","status":"current"},
    {"id":"ui.surface-morph-capsule","status":"experimental"},
    {"id":"ui.surface-macos-x64","status":"unsupported"},
    {"id":"ui.surface-windows-x64","status":"unsupported"},
    {"id":"ui.surface-wasm32-wasi","status":"unsupported"}
  ]
}
`
}

func writeSurfaceReleaseStateFixture(t *testing.T, dir string) {
	t.Helper()
	writeSurfaceReleaseStateFixtureFiles(t, dir, surfaceReleaseStateFixtureFiles())
}

func surfaceReleaseStateFixtureFiles() map[string]string {
	return map[string]string{
		"surface-release-summary.json": `{
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
}`,
		"surface-macos-x64-target-host-status.json": `{
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
}`,
		"surface-windows-x64-target-host-status.json": `{
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
}`,
		"surface-dev-workflow.json":     surfaceReleaseStateDevWorkflowJSON(),
		"surface-inspector.json":        surfaceReleaseStateInspectorJSON(),
		"surface-template-smoke.json":   surfaceReleaseStateTemplateSmokeJSON(),
		"surface-reference-apps.json":   surfaceReleaseStateReferenceAppsJSON(),
		"surface-package.json":          surfaceReleaseStatePackageJSON(),
		"surface-crash-report.json":     surfaceReleaseStateCrashReportJSON(),
		"surface-i18n.json":             surfaceReleaseStateI18nJSON(),
		"surface-widget-migration.json": surfaceReleaseStateWidgetMigrationJSON(),
		"morph/surface-morph-gate-summary.json": `{
  "schema": "tetra.surface.morph.gate.v1",
  "status": "current",
  "release_scope": "surface-morph-experimental-linux-web",
  "producer": "scripts/release/surface/morph-gate.sh",
  "source": "examples/surface_morph_command_palette.tetra",
  "module": "lib.core.morph",
  "schema_under_test": "tetra.surface.morph.v1",
  "dependency_gate": "tetra.surface.block-system.gate.v1",
  "same_commit_validated": true,
  "headless_report": "headless/surface-headless-morph.json",
  "target_evidence": ["headless"],
  "core_primitives": ["Block"],
  "forbidden_core_primitives": ["Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"],
  "artifact_hashes_validated": true
}`,
		"morph/headless/surface-headless-morph.json": `{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "source": "examples/surface_morph_command_palette.tetra",
  "morph": {
    "schema": "tetra.surface.morph.v1",
    "quality_level": "deterministic-headless-morph-capsule-v1",
    "source": "examples/surface_morph_command_palette.tetra",
    "module": "lib.core.morph",
    "surface_scope": "surface-morph-experimental-linux-web",
    "production_claim": false
  }
}`,
		"surface-headless-release-text-input.json": `{
  "schema": "tetra.surface.text-input.v1",
  "target": "headless",
  "source": "examples/surface_release_text_input.tetra",
  "level": "production-text-input-v1",
  "experimental": false,
  "production_claim": true,
  "storage": "owned-utf8-byte-buffer",
  "utf8_validation": true,
  "invalid_utf8_rejected": true,
  "caret": true,
  "selection": true,
  "selection_clipboard_transfer": true,
  "multiline": true,
  "backspace": true,
  "delete": true,
  "home_end": true,
  "arrow_left_right": true,
  "composition_events": true,
  "composition_commit": true,
  "composition_cancel": true,
  "clipboard_read": true,
  "clipboard_write": true,
  "clipboard_host_abi": true,
  "clipboard_owned_copy": true,
  "target_host_composition_trace": true,
  "composition_trace": {"start":true,"update":true,"commit":true,"cancel":true},
  "text_shaping_plan": {
    "quality_level": "scoped-text-shaping-plan-v1",
    "fallback_fonts": true,
    "grapheme_boundaries": "byte-offset-codepoint-v1",
    "line_breaking": "newline-storage-plus-wrap-plan-v1",
    "bidi": "nonclaim-full-bidi-v1",
    "rich_text": "nonclaim-rich-text-editor-v1"
  },
  "reference_traces": [
    {"source":"examples/surface_morph_settings.tetra","trace":"settings text field trace","focus":true,"selection":true,"clipboard":true,"composition":true,"multiline":true,"pass":true},
    {"source":"examples/surface_morph_editor_shell.tetra","trace":"editor shell text area trace","focus":true,"selection":true,"clipboard":true,"composition":true,"multiline":true,"pass":true}
  ],
  "unsupported_claims": ["full-rich-text-editor","full-bidi-shaping","grapheme-cluster-caret","ide-grade-editor"],
  "rich_text_production_claim": false,
  "bidi_production_claim": false,
  "full_editor_production_claim": false,
  "borrowed_view_storage": false,
  "safe_view_lifetime_checked": true,
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-release-text-input","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":4096},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":2048}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "cases": [
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"release text input ASCII insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input UTF-8 insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input invalid UTF-8 rejected","kind":"negative","ran":true,"pass":true,"expected_error":"invalid utf8 rejected"},
    {"name":"release text input multiline storage","kind":"positive","ran":true,"pass":true},
    {"name":"release text input caret home end arrows","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection replacement","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection clipboard transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"release text input clipboard owned copy transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition start update","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition cancel","kind":"positive","ran":true,"pass":true},
    {"name":"release text input shaping plan scoped","kind":"positive","ran":true,"pass":true},
    {"name":"settings reference text input trace","kind":"positive","ran":true,"pass":true},
    {"name":"editor reference text input trace","kind":"positive","ran":true,"pass":true},
    {"name":"release text input safe view lifetime checked","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`,
		"surface-wasm32-web-release-browser.json":  `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"wasm32-web","host_evidence":{"level":"wasm32-web-browser-canvas-release-v1","backend":"browser-canvas-rgba-accessible","framebuffer":true,"browser_canvas":true,"browser_input":true,"browser_clipboard":true,"browser_clipboard_harness":"deterministic-browser-clipboard-v1","browser_composition":true,"browser_accessibility_snapshot":true,"browser_accessibility_mirror":true,"user_facing_platform_widgets":false},"source":"examples/surface_release_form.tetra","browser_surface":{"schema":"tetra.surface.browser-surface.v1","browser_surface_level":"browser-canvas-release-v1","dom_host_canvas_only":true,"negative_guards":{"no_dom_app_ui_tree":true,"no_user_js_app_logic":true,"no_node_only_promotion":true}}}`,
		"surface-linux-x64-release-window.json":    `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"linux-x64","host_evidence":{"level":"linux-x64-release-window-v1","backend":"wayland-shm-rgba-release-v1","framebuffer":true,"real_window":true,"native_input":true,"text_input":true,"clipboard":true,"composition":true,"accessibility_bridge":true,"user_facing_platform_widgets":false},"source":"examples/surface_release_form.tetra"}`,
		"surface-linux-x64-release-app-shell.json": surfaceReleaseStateLinuxAppShellReportJSONWithSecurityAndPerformance(),
	}
}

func surfaceReleaseStateLinuxAppShellReportJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"linux-x64","runtime":"surface-linux-x64","host_evidence":{"level":"linux-x64-release-window-v1","backend":"wayland-shm-rgba-release-v1","framebuffer":true,"real_window":true,"native_input":true,"text_input":true,"clipboard":true,"composition":true,"accessibility_bridge":true,"user_facing_platform_widgets":false},"source":"examples/surface_linux_app_shell_notes.tetra","linux_app_shell":{"schema":"tetra.surface.linux-app-shell.v1","app_shell_level":"linux-app-shell-subset-v1","shell_features":[{"name":"app_menu","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"window_lifecycle","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"multi_window","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"clipboard","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"ime","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"accessibility_bridge","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"crash_recovery","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"error_report","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},{"name":"dialog","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host dialog unavailable in CI","no_native_widget_ui":true,"pass":true},{"name":"file_dialog","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host file dialog unavailable in CI","no_native_widget_ui":true,"pass":true},{"name":"file_picker","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host file picker unavailable in CI","no_native_widget_ui":true,"pass":true},{"name":"notification","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host notification unavailable in CI","no_native_widget_ui":true,"pass":true},{"name":"tray","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host tray unavailable in CI","no_native_widget_ui":true,"pass":true},{"name":"deep_link","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host deep link unavailable in CI","no_native_widget_ui":true,"pass":true}],"negative_guards":{"no_gtk":true,"no_qt":true,"no_native_widgets":true}}}`
}

func surfaceReleaseStateLinuxAppShellReportJSONWithSecurity() string {
	return strings.TrimSuffix(surfaceReleaseStateLinuxAppShellReportJSON(), "}") + `,"security_permissions":` + surfaceReleaseStateSecurityPermissionsJSON() + `}`
}

func surfaceReleaseStateLinuxAppShellReportJSONWithSecurityAndPerformance() string {
	return strings.TrimSuffix(surfaceReleaseStateLinuxAppShellReportJSONWithSecurity(), "}") + `,"surface_performance_budget":` + surfaceReleaseStatePerformanceBudgetJSON() + `}`
}

func surfaceReleaseStateSecurityPermissionsJSON() string {
	return `{"schema":"tetra.surface.security-permission.v1","model":"surface-security-permission-v1","release_scope":"surface-v1-linux-web","source":"examples/surface_linux_app_shell_notes.tetra","app_shell_features":"electron-feature-ledger-v1","production_claim":true,"experimental":false,"default_deny":true,"shell_feature_policy_enforced":true,"capabilities":[{"name":"app_menu","source_feature":"app_menu","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"window_lifecycle","source_feature":"window_lifecycle","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"multi_window","source_feature":"multi_window","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"clipboard","source_feature":"clipboard","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"ime","source_feature":"ime","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"accessibility_bridge","source_feature":"accessibility_bridge","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"crash_recovery","source_feature":"crash_recovery","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"error_report","source_feature":"error_report","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},{"name":"dialog","source_feature":"dialog","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host dialog unavailable in CI","pass":true},{"name":"file_dialog","source_feature":"file_dialog","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host file dialog unavailable in CI","pass":true},{"name":"file_picker","source_feature":"file_picker","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host file picker unavailable in CI","pass":true},{"name":"notification","source_feature":"notification","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host notification unavailable in CI","pass":true},{"name":"tray","source_feature":"tray","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host tray unavailable in CI","pass":true},{"name":"deep_link","source_feature":"deep_link","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host deep link unavailable in CI","pass":true}],"permissions":[{"name":"filesystem","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"ambient filesystem denied in default template","evidence":"default-deny-policy","pass":true},{"name":"network","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"ambient network denied in default template","evidence":"default-deny-policy","pass":true},{"name":"clipboard","status":"allowed_with_policy","allowed":true,"capability_checked":true,"blocked_reason":"","evidence":"linux-app-shell-host-trace","pass":true},{"name":"notifications","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"notification target evidence absent","evidence":"blocked-pass-nonclaim","pass":true},{"name":"dialogs","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"dialog target evidence absent","evidence":"blocked-pass-nonclaim","pass":true},{"name":"shell_open_url","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"shell open-url denied in default template","evidence":"default-deny-policy","pass":true}],"process_boundaries":[{"name":"surface_app_to_host_abi","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true},{"name":"linux_app_shell_host_adapter","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true},{"name":"browser_canvas_host","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true}],"asset_safety":[{"kind":"font","local_only":true,"sha256_required":true,"size_limit_bytes":1048576,"network_fetch_allowed":false,"parser":"bounded-font-metadata-v1","bounds_checked":true,"pass":true},{"kind":"image","local_only":true,"sha256_required":true,"size_limit_bytes":2097152,"network_fetch_allowed":false,"parser":"bounded-image-header-v1","bounds_checked":true,"pass":true},{"kind":"icon","local_only":true,"sha256_required":true,"size_limit_bytes":262144,"network_fetch_allowed":false,"parser":"bounded-icon-header-v1","bounds_checked":true,"pass":true}],"unsupported_claims":["unrestricted-filesystem","unrestricted-network","native-permission-prompts","production-notifications","production-dialogs","remote-asset-fetch","electron-node-integration"],"negative_guards":{"no_ambient_filesystem":true,"no_ambient_network":true,"no_shell_feature_bypass":true,"no_permissionless_clipboard":true,"no_notification_dialog_without_target_evidence":true,"no_network_asset_fetch":true,"no_untrusted_font_image_decode":true,"no_electron_node_integration":true,"no_user_js_app_logic":true,"no_dom_app_ui_tree":true}}`
}

func surfaceReleaseStatePerformanceBudgetJSON() string {
	return `{"schema":"tetra.surface.performance-budget.v1","model":"surface-performance-budget-v1","release_scope":"surface-v1-linux-web","source":"examples/surface_linux_app_shell_notes.tetra","target":"linux-x64","runtime":"surface-linux-x64","production_claim":true,"experimental":false,"git_head":"0123456789abcdef0123456789abcdef01234567","performance_claim":"none","startup":{"launch_to_first_frame_ms":18,"budget_ms":250,"trace":"local-startup-trace-v1","pass":true},"frame":{"frame_count":3,"p50_build_ms":4,"p95_build_ms":7,"p50_present_ms":3,"p95_present_ms":6,"budget_ms":16,"idle_loop_count":24,"work_loop_count":6,"pass":true},"scene":{"block_count":3,"recipe_expansion_count":0,"paint_command_count":10,"layout_pass_count":4,"text_run_count":2},"memory":{"glyph_cache_bytes":4096,"asset_cache_bytes":5376,"layout_cache_bytes":4096,"paint_cache_bytes":10240,"framebuffer_peak_bytes":1555200,"framebuffer_total_bytes":2880000,"rss_measured":false,"peak_rss_bytes":0,"allocation_count":42,"allocation_bytes":2903808,"bounded_caches":true,"unbounded_cache_rejected":true,"pass":true},"binary":{"artifact_path":"/tmp/surface-artifacts/surface-linux-app-shell-notes","size_bytes":90001,"budget_bytes":16777216,"pass":true},"cpu_power_proxy":{"idle_loop_count":24,"work_loop_count":6,"idle_frame_count":2,"work_frame_count":1,"real_power_measured":false,"pass":true},"cache":{"glyph_cache_budget_bytes":65536,"asset_cache_budget_bytes":65536,"layout_cache_budget_bytes":65536,"paint_cache_budget_bytes":65536,"total_cache_bytes":23808,"total_cache_budget_bytes":262144,"eviction":"bounded-lru","pass":true},"methodology":{"kind":"local-deterministic-budget-v1","electron_comparison":"none","official_benchmark":false,"cross_machine":false,"fair_comparison_required_for_electron_claim":true},"unsupported_claims":["faster-than-electron","lower-power-than-electron","official-benchmark-result","cross-machine-benchmark","electron-parity-performance"],"negative_guards":{"bounded_caches":true,"unbounded_cache_rejected":true,"stale_report_rejected":true,"no_faster_than_electron_claim":true,"no_benchmark_parity_claim":true,"peak_memory_field_required":true,"no_official_benchmark_claim":true}}`
}

func surfaceReleaseStateDevWorkflowJSON() string {
	return `{"schema":"tetra.surface.dev-workflow.v1","model":"surface-dev-workflow-v1","release_scope":"surface-v1-linux-web","command":"tetra surface dev","source":"reports/surface-release-v1/dev-fixture/app/main.tetra","target":"linux-x64","mode":"fast-rebuild","reload_semantics":"fast-rebuild","process_restart_required":true,"hot_reload_claim":false,"watch":false,"supported_targets":["headless","linux-x64","wasm32-web"],"steps":[{"name":"initial build","kind":"initial","changed_path":"","output_path":"reports/surface-release-v1/dev-artifacts/initial/app","duration_ms":25,"compiled_modules":["app.main","design.recipes","design.tokens"],"cache_hits":[],"pass":true},{"name":"warm rebuild","kind":"warm-cache","changed_path":"","output_path":"reports/surface-release-v1/dev-artifacts/warm/app","duration_ms":3,"compiled_modules":[],"cache_hits":["app.main","design.recipes","design.tokens"],"pass":true},{"name":"token rebuild","kind":"token-change","changed_path":"reports/surface-release-v1/dev-fixture/design/tokens.tetra","output_path":"reports/surface-release-v1/dev-artifacts/token/app","duration_ms":8,"compiled_modules":["design.tokens"],"cache_hits":["app.main","design.recipes"],"pass":true},{"name":"recipe rebuild","kind":"recipe-change","changed_path":"reports/surface-release-v1/dev-fixture/design/recipes.tetra","output_path":"reports/surface-release-v1/dev-artifacts/recipe/app","duration_ms":7,"compiled_modules":["design.recipes"],"cache_hits":["app.main","design.tokens"],"pass":true},{"name":"source rebuild","kind":"source-change","changed_path":"reports/surface-release-v1/dev-fixture/app/main.tetra","output_path":"reports/surface-release-v1/dev-artifacts/source/app","duration_ms":9,"compiled_modules":["app.main"],"cache_hits":["design.recipes","design.tokens"],"pass":true}],"source_diagnostics":[{"kind":"token","path":"reports/surface-release-v1/dev-fixture/design/tokens.tetra","line":1,"column":1,"code":"SURFACE_DEV_TOKEN_PATH","message":"token file participates in Surface fast rebuild","severity":"info","pass":true},{"kind":"recipe","path":"reports/surface-release-v1/dev-fixture/design/recipes.tetra","line":1,"column":1,"code":"SURFACE_DEV_RECIPE_PATH","message":"recipe file participates in Surface fast rebuild","severity":"info","pass":true},{"kind":"source","path":"reports/surface-release-v1/dev-fixture/app/main.tetra","line":1,"column":1,"code":"SURFACE_DEV_SOURCE_PATH","message":"source file participates in Surface fast rebuild","severity":"info","pass":true}],"negative_guards":{"no_hot_reload_claim":true,"full_restart_documented_as_fast_rebuild":true,"no_electron_dev_server":true,"no_react_fast_refresh":true,"no_dom_hot_reload":true},"pass":true}`
}

func surfaceReleaseStateInspectorJSON() string {
	return `{"schema":"tetra.surface.inspector.v1","model":"surface-inspector-v1","release_scope":"surface-v1-linux-web","producer":"tools/cmd/surface-inspector","source":"examples/surface_block_system.tetra","target":"headless","mode":"static-tool-report","input_reports":[{"kind":"block","path":"inspector-inputs/surface-headless-block-system.json","schema":"tetra.surface.runtime.v1","source":"examples/surface_block_system.tetra","target":"headless","pass":true},{"kind":"morph","path":"inspector-inputs/surface-headless-morph.json","schema":"tetra.surface.runtime.v1","source":"examples/surface_morph_command_palette.tetra","target":"headless","pass":true},{"kind":"accessibility","path":"inspector-inputs/surface-headless-release-accessibility.json","schema":"tetra.surface.runtime.v1","source":"examples/surface_release_accessibility.tetra","target":"headless","pass":true},{"kind":"app-model","path":"inspector-inputs/surface-headless-app-model.json","schema":"tetra.surface.runtime.v1","source":"examples/surface_app_model.tetra","target":"headless","pass":true}],"source_locations":[{"kind":"block","path":"examples/surface_block_system.tetra","line":1,"column":1},{"kind":"morph","path":"examples/surface_morph_command_palette.tetra","line":1,"column":1},{"kind":"accessibility","path":"examples/surface_release_accessibility.tetra","line":1,"column":1},{"kind":"app-model","path":"examples/surface_app_model.tetra","line":1,"column":1}],"sections":{"block_tree":{"present":true,"count":6,"source":"block_graph.nodes"},"morph_tokens":{"present":true,"count":22,"source":"morph.token_graph.tokens"},"layout":{"present":true,"count":6,"source":"layout_passes"},"paint":{"present":true,"count":10,"source":"paint_commands"},"accessibility":{"present":true,"count":12,"source":"accessibility_tree.nodes"},"event_routes":{"present":true,"count":5,"source":"block_event_routes"},"focus":{"present":true,"count":3,"source":"block_focus_transitions"},"perf_counters":{"present":true,"count":4,"source":"surface_performance_budget"}},"static_artifacts":{"json":"surface-inspector.json","html":"surface-inspector.html","html_tool_report":true},"hidden_state":{"scanned":true,"findings":[]},"negative_guards":{"no_dom_runtime_dependency":true,"no_browser_devtools_dependency":true,"no_react_devtools_dependency":true,"static_html_tool_report_only":true,"no_hidden_state":true},"pass":true}`
}

func surfaceReleaseStateTemplateSmokeJSON() string {
	kinds := []string{"command-palette", "settings", "dashboard", "editor-shell", "studio-shell", "multi-window-notes", "web-canvas"}
	var templates []string
	for _, kind := range kinds {
		imports := `"imports":["lib.core.surface","lib.core.block","lib.core.morph"]`
		usesAppShell := "false"
		webCanvas := "false"
		if kind == "multi-window-notes" || kind == "studio-shell" {
			imports = `"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"]`
			usesAppShell = "true"
		}
		if kind == "web-canvas" {
			webCanvas = "true"
		}
		templates = append(templates, `{"kind":"`+kind+`","project_dir":"templates/`+kind+`","source":"templates/`+kind+`/src/main.tetra","capsule":"templates/`+kind+`/Capsule.t4","template_metadata":"templates/`+kind+`/surface-template.json","targets":["linux-x64","wasm32-web"],`+imports+`,"recipe_count":4,"block_morph_only":true,"uses_app_shell":`+usesAppShell+`,"web_canvas":`+webCanvas+`,"commands":[{"kind":"generate","command":"tetra new surface-app --template `+kind+`","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-`+kind+`.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}}`)
	}
	return `{"schema":"tetra.surface.template-smoke.v1","model":"surface-template-smoke-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-template-smoke.sh","command":"tetra new surface-app","template_count":7,"templates":[` + strings.Join(templates, ",") + `],"inspector_evidence":{"path":"surface-template-inspector.json","model":"surface-inspector-v1","pass":true},"visual_evidence":{"path":"template-visual/surface-visual-regression.json","schema":"tetra.surface.visual-regression.v1","pass":true},"morph_to_pixels":` + surfaceReleaseStateMorphToPixelsJSON("templates/studio-shell/src/main.tetra") + `,"package_evidence":[{"path":"template-packages/surface-template-command-palette.tar.gz","kind":"tar.gz","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","pass":true}],"negative_guards":{"no_react_import":true,"no_electron_import":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_core_widgets":true,"no_platform_widgets":true,"no_user_js_app_logic":true,"cookbook_uses_block_morph":true},"pass":true}`
}

func surfaceReleaseStateMorphToPixelsJSON(source string) string {
	slug := strings.NewReplacer("/", "-", ".", "-").Replace(source)
	return `{"chain_id":"sha256:0000000000000000000000000000000000000000000000000000000000000900","report_path":"reports/surface/morph-rendered-beauty/` + slug + `.json","schema":"tetra.surface.morph-rendered-beauty.v1","status":"pass","surface_scope":"surface-morph-rendered-beauty-linux-web","source":"` + source + `","source_sha256":"sha256:0000000000000000000000000000000000000000000000000000000000000001","target":"headless","scenario_name":"headless-morph:` + source + `","token_graph_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000003","token_count":6,"token_categories":["color","space","radius","typography","motion","assets"],"recipe_count":3,"recipe_expansion_count":4,"recipe_names":["studio_shell","hero_panel","toolbar"],"block_scene_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000005","block_scene_node_count":12,"render_command_stream_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000007","render_command_count":10,"renderer":"software-rgba-headless","frame_artifact":"reports/surface/morph-rendered-beauty/` + slug + `/frame.rgba","frame_artifact_sha256":"sha256:000000000000000000000000000000000000000000000000000000000000003c","frame_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003c","golden_artifact":"reports/surface/morph-rendered-beauty/` + slug + `/golden.rgba","golden_artifact_sha256":"sha256:000000000000000000000000000000000000000000000000000000000000003d","golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003d","diff_pixels":1,"diff_ratio_milli":0,"max_channel_delta":1,"product_claim":false,"final_signoff":false,"pass":true}`
}

func surfaceReleaseStateReferenceAppsJSON() string {
	apps := []struct {
		shape         string
		source        string
		compatibility bool
	}{
		{shape: "command-palette", source: "examples/surface_reference_command_palette.tetra"},
		{shape: "settings", source: "examples/surface_reference_settings.tetra"},
		{shape: "dashboard", source: "examples/surface_reference_dashboard.tetra"},
		{shape: "editor-shell", source: "examples/surface_reference_editor_shell.tetra"},
		{shape: "file-manager", source: "examples/surface_reference_file_manager.tetra"},
		{shape: "dialog-notification", source: "examples/surface_reference_dialog_notification.tetra"},
		{shape: "localized-form", source: "examples/surface_reference_localized_form.tetra"},
		{shape: "accessibility-form", source: "examples/surface_reference_accessibility_form.tetra"},
		{shape: "multi-window-notes", source: "examples/surface_reference_multi_window_notes.tetra"},
		{shape: "migration", source: "examples/surface_reference_migration.tetra", compatibility: true},
	}
	var entries []string
	for _, app := range apps {
		imports := `"imports":["lib.core.surface","lib.core.block","lib.core.morph"]`
		if app.shape == "multi-window-notes" {
			imports = `"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"]`
		}
		if app.compatibility {
			imports = `"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.widgets"]`
		}
		targets := []string{
			surfaceReleaseStateReferenceTargetJSON(app.shape, "headless"),
			surfaceReleaseStateReferenceTargetJSON(app.shape, "linux-x64-real-window"),
			surfaceReleaseStateReferenceTargetJSON(app.shape, "wasm32-web-browser-canvas"),
		}
		entry := `{"shape":"` + app.shape + `","source":"` + app.source + `","module":"examples.` + strings.TrimSuffix(strings.TrimPrefix(strings.ReplaceAll(app.source, "/", "."), "examples."), ".tetra") + `",` + imports + `,"recipes":["region.panel","field.text","control.action","command.item"],"beauty_coverage":` + surfaceReleaseStateReferenceBeautyCoverageJSON(app.shape) + `,"stable_morph_recipes":true,"resolves_to_block":true,"compiles":true,"runs":true,"exit_code":0,"token_theme_conformance":true,"layout_report":true,"interaction_trace":true,"accessibility_snapshot":true,"performance_budget":true,"artifact_hashes":true,"compatibility_widgets":` + releaseStateBoolJSON(app.compatibility)
		if app.compatibility {
			entry += `,"infrastructure_only":true,"non_product_reason":"legacy widget migration compatibility evidence only","targets":[` + strings.Join(targets, ",") + `]}`
		} else {
			entry += `,"infrastructure_only":false,"morph_to_pixels":` + surfaceReleaseStateMorphToPixelsJSON(app.source) + `,"targets":[` + strings.Join(targets, ",") + `]}`
		}
		entries = append(entries, entry)
	}
	return `{"schema":"tetra.surface.reference-app-suite.v1","model":"surface-reference-app-suite-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-reference-apps-smoke.sh","app_count":10,"required_targets":["headless","linux-x64-real-window","wasm32-web-browser-canvas"],"apps":[` + strings.Join(entries, ",") + `],"visual_evidence":{"path":"reference-visual/surface-visual-regression.json","schema":"tetra.surface.visual-regression.v1","app_count":10,"pass":true},"negative_guards":{"screenshot_only_rejected":true,"missing_interaction_rejected":true,"missing_accessibility_rejected":true,"missing_performance_rejected":true,"core_widget_usage_rejected":true,"migration_widgets_compatibility_only":true,"no_react_runtime":true,"no_electron_runtime":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_user_js_app_logic":true},"pass":true}`
}

func surfaceReleaseStateReferenceBeautyCoverageJSON(shape string) string {
	switch shape {
	case "command-palette":
		return `["command-palette","focus-state"]`
	case "settings":
		return `["settings","disabled-state"]`
	case "dashboard":
		return `["dashboard"]`
	case "editor-shell":
		return `["editor-shell"]`
	case "dialog-notification":
		return `["elevated-panel"]`
	case "migration":
		return `[]`
	default:
		return `["focus-state"]`
	}
}

func surfaceReleaseStateReferenceTargetJSON(shape string, target string) string {
	return `{"target":"` + target + `","runtime_report":"reference-runtime/` + shape + `-` + target + `.json","frame_checksum":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","visual_diff":true,"interaction_trace":true,"accessibility_snapshot":true,"performance_budget":true,"pass":true,"screenshot_only":false}`
}

func surfaceReleaseStatePackageJSON() string {
	return `{"schema":"tetra.surface.package.v1","model":"surface-package-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-package-smoke.sh","source":"examples/surface_reference_command_palette.tetra","reference_app":"command-palette","package_format":"surface-app-package-v1","format_version":1,"artifact_root":"surface-package-work","packages":[{"target":"linux-x64","kind":"linux-x64-tar.gz","path":"surface-packages/surface-command-palette-linux-x64.tar.gz","manifest_path":"surface-package-work/linux-x64/package-manifest.json","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","asset_manifest_sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","source_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","build_sha256":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","contains_executable":true,"contains_web_bundle":false,"local_only_assets":true,"pass":true},{"target":"wasm32-web","kind":"wasm32-web-tar.gz","path":"surface-packages/surface-command-palette-wasm32-web.tar.gz","manifest_path":"surface-package-work/wasm32-web/package-manifest.json","sha256":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","asset_manifest_sha256":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","source_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","build_sha256":"sha256:1111111111111111111111111111111111111111111111111111111111111111","contains_executable":false,"contains_web_bundle":true,"local_only_assets":true,"pass":true}],"assets":[{"path":"surface-package-work/assets/app-icon.txt","kind":"icon","sha256":"sha256:2222222222222222222222222222222222222222222222222222222222222222","size_bytes":32,"local_only":true,"network_fetch_allowed":false,"pass":true},{"path":"surface-package-work/assets/theme-manifest.json","kind":"theme","sha256":"sha256:3333333333333333333333333333333333333333333333333333333333333333","size_bytes":64,"local_only":true,"network_fetch_allowed":false,"pass":true}],"install_smokes":[{"target":"linux-x64","package_path":"surface-packages/surface-command-palette-linux-x64.tar.gz","install_dir":"surface-install/linux-x64","installed_binary":"surface-install/linux-x64/bin/surface-command-palette","command":"surface-install/linux-x64/bin/surface-command-palette","exit_code":0,"artifact_hash_verified":true,"package_manifest_verified":true,"app_run":true,"pass":true}],"web_bundles":[{"target":"wasm32-web","package_path":"surface-packages/surface-command-palette-wasm32-web.tar.gz","web_entry":"surface-package-work/wasm32-web/index.html","wasm_artifact":"surface-package-work/wasm32-web/surface-command-palette.wasm","loader_artifact":"surface-package-work/wasm32-web/surface-command-palette.mjs","browser_canvas_host":"surface-package-work/wasm32-web/surface-browser-canvas-host.mjs","command":"tetra build --target wasm32-web","artifact_hash_verified":true,"package_manifest_verified":true,"pass":true}],"update_strategy":{"strategy":"hash-pinned-channel-manifest-v1","manifest_format":"tetra.surface.update-channel.v1","channel_manifest":"surface-updates/channel.json","current_version":"p23.0.0","latest_version":"p23.0.0","latest_package_path":"surface-packages/surface-command-palette-linux-x64.tar.gz","latest_package_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","package_hash_pinned":true,"rollback_manifest":"surface-updates/rollback.json","signature_required_for_stable_promotion":true,"auto_update_runtime_claim":false,"network_update_claim":false,"pass":true},"signing":{"status":"nonclaim","signed":false,"notarized":false,"production_claim":false,"evidence":"","blocked_reason":"platform signing keys and CI signing evidence are not present in this release"},"notarization":{"status":"nonclaim","signed":false,"notarized":false,"production_claim":false,"evidence":"","blocked_reason":"macOS notarization evidence is unavailable because macOS Surface target host is unsupported"},"negative_guards":{"no_react_runtime":true,"no_electron_runtime":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_user_js_app_logic":true,"no_remote_asset_fetch":true,"no_unsigned_signing_claim":true,"no_notarization_without_platform_evidence":true,"no_auto_update_without_runtime_evidence":true,"no_docs_only_package_claim":true,"install_run_required":true,"web_bundle_required":true,"artifact_hashes_required":true},"pass":true}`
}

func surfaceReleaseStateCrashReportJSON() string {
	return `{"schema":"tetra.surface.crash-report.v1","model":"surface-crash-report-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-crash-report-smoke.sh","source":"examples/surface_reference_command_palette.tetra","reference_app":"command-palette","target":"linux-x64","diagnostic_schema":"tetra.surface.diagnostic.v1","scenarios":[{"name":"command failure boundary","kind":"command_failure","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"command.palette.missing","diagnostic_path":"surface-crash/command-failure.json","diagnostic_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","report_written":true,"command_boundary":true,"host_captured":false,"restarted":false,"contains_user_data":false,"pass":true},{"name":"host crash capture","kind":"host_crash","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"surface-host panic harness","diagnostic_path":"surface-crash/host-crash.json","diagnostic_sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","report_written":true,"command_boundary":false,"host_captured":true,"restarted":false,"contains_user_data":false,"pass":true},{"name":"restart after diagnostic","kind":"restart_recovery","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"restart after command failure report","diagnostic_path":"surface-crash/restart-recovery.json","diagnostic_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","report_written":true,"command_boundary":false,"host_captured":false,"restarted":true,"contains_user_data":false,"pass":true}],"diagnostics":[{"path":"surface-crash/command-failure.json","kind":"command_failure","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true},{"path":"surface-crash/host-crash.json","kind":"host_crash","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true},{"path":"surface-crash/restart-recovery.json","kind":"restart_recovery","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true}],"trace_collection":{"trace_path":"surface-crash/surface-app-trace.json","log_path":"surface-crash/surface-app.log","ring_buffer":true,"max_bytes":4096,"event_count":4,"bounded":true,"local_only":true,"pass":true},"restart_recovery":{"scope":"scoped-linux-x64-process-restart-v1","target":"linux-x64","restart_claim":true,"before_run":true,"failure_report_written":true,"after_run":true,"before_exit_code":0,"after_exit_code":0,"state_restored":"explicit-startup-state-v1","command":"surface-crash-work/surface-command-palette-linux-x64","pass":true},"privacy_policy":{"policy":"surface-non-user-data-diagnostics-v1","redaction_version":"surface-diagnostic-redaction-v1","user_data_redacted":true,"clipboard_payload_captured":false,"user_text_captured":false,"env_dumped":false,"home_path_captured":false,"network_upload":false,"local_only":true,"pass":true},"negative_guards":{"no_user_data_leak":true,"no_clipboard_payload":true,"no_user_text_payload":true,"no_env_dump":true,"no_home_path_leak":true,"no_network_upload":true,"no_restart_claim_without_evidence":true,"no_silent_failure":true,"no_docs_only_crash_claim":true,"no_electron_crash_reporter_dependency":true},"pass":true}`
}

func surfaceReleaseStateI18nJSON() string {
	return `{"schema":"tetra.surface.i18n.v1","model":"surface-i18n-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-i18n-smoke.sh","source":"examples/surface_reference_localized_form.tetra","reference_app":"localized-form","target":"linux-x64","string_tables":[{"locale":"en-US","entry_count":5,"checksum":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","primary":true,"fallback":false,"pass":true},{"locale":"uk-UA","entry_count":4,"checksum":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","primary":false,"fallback":true,"pass":true}],"locale_selection":{"requested_locale":"uk-UA","selected_locale":"uk-UA","fallback_locale":"en-US","fallback_used":true,"unsupported_locale_rejected":true,"pass":true},"lookups":[{"key":"form.title","locale":"uk-UA","resolved_locale":"uk-UA","source":"primary","missing_key":false,"fallback_used":false,"diagnostic_code":0,"pass":true},{"key":"form.secondary","locale":"uk-UA","resolved_locale":"en-US","source":"fallback","missing_key":false,"fallback_used":true,"diagnostic_code":0,"pass":true},{"key":"form.unknown","locale":"uk-UA","resolved_locale":"en-US","source":"missing","missing_key":true,"fallback_used":true,"diagnostic_code":2001,"pass":true}],"format_hooks":[{"kind":"date","locale":"uk-UA","input":"2026-06-12","output":"2026-06-12","deterministic":true,"icu_claim":false,"pass":true},{"kind":"number","locale":"uk-UA","input":"4200","output":"4200","deterministic":true,"icu_claim":false,"pass":true}],"text_direction":{"default_direction":"ltr","rtl_placeholder":true,"full_bidi_supported":false,"full_bidi_claim":false,"shaping_proof":false,"nonclaim":"rtl-placeholder-without-full-bidi-shaping-v1","pass":true},"localized_form":{"shape":"localized-form","source":"examples/surface_reference_localized_form.tetra","imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.i18n"],"compiles":true,"runs":true,"exit_code":0,"localized_strings":true,"fallback_evidence":true,"missing_key_diagnostic":true,"format_hook_evidence":true,"resolves_to_block":true,"pass":true},"negative_guards":{"no_full_icu_claim":true,"no_full_bidi_claim":true,"no_rtl_production_claim":true,"no_missing_key_silent_fallback":true,"no_docs_only_i18n_claim":true,"no_react_intl_runtime":true,"no_platform_locale_dependency":true},"pass":true}`
}

func surfaceReleaseStateWidgetMigrationJSON() string {
	return `{"schema":"tetra.surface.widget-migration.v1","model":"surface-widget-migration-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-widget-migration-smoke.sh","source":"examples/surface_reference_migration.tetra","reference_app":"migration","target":"linux-x64","compatibility_layer":{"module":"lib.core.widgets","supported_surface_v1":true,"current_api_preserved":true,"api_breaking_change":false,"migration_equivalence_helpers":true,"migration_docs":true,"pass":true},"release_widget_set":{"widgets":["Text","Label","StatusText","Button","TextBox","Row","Column","Panel","Checkbox","Stack","Scroll","Spacer"],"intact":true,"non_migration_widget_usage":false,"pass":true},"equivalence_rows":[{"legacy_widget":"Panel","legacy_function":"widgets.panel_init","morph_recipe":"recipe_region_panel","block_expander":"morph.expand_region_panel","block_kind":"Block","legacy_result":380,"block_result":380,"api_unchanged":true,"resolves_to_block":true,"pass":true},{"legacy_widget":"Button","legacy_function":"widgets.button_init","morph_recipe":"recipe_control_action","block_expander":"morph.expand_control_action","block_kind":"Block","legacy_result":1301,"block_result":1301,"api_unchanged":true,"resolves_to_block":true,"pass":true},{"legacy_widget":"TextBox","legacy_function":"widgets.textbox_init","morph_recipe":"recipe_field_text","block_expander":"morph.expand_field_text","block_kind":"Block","legacy_result":344,"block_result":344,"api_unchanged":true,"resolves_to_block":true,"pass":true}],"morph_recipe_migration":{"recipes":["recipe_region_panel","recipe_control_action","recipe_field_text"],"core_primitives":["Block"],"block_only_core_primitive":true,"widgets_promoted_to_core":false,"resolves_to_block":true,"pass":true},"migration_reference_app":{"shape":"migration","source":"examples/surface_reference_migration.tetra","imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.widgets"],"compiles":true,"runs":true,"exit_code":0,"uses_widgets_compat":true,"uses_morph_recipes":true,"resolves_to_block":true,"pass":true},"negative_guards":{"no_future_core_primitive_promotion":true,"no_widget_primary_future_core":true,"no_breaking_change":true,"no_docs_only":true,"no_platform_native_runtime_claims":true},"artifact_evidence":{"equivalence_rows_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","source_scan_sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"pass":true}`
}

func releaseStateBoolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func writeSurfaceReleaseStateFixtureFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, raw := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeSurfaceReleaseArtifactHashes(t, dir, files)
}

func writeSurfaceReleaseArtifactHashes(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	artifacts := make([]artifact, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s for hash manifest: %v", name, err)
		}
		sum := sha256.Sum256(raw)
		entry := artifact{
			Path:   filepath.ToSlash(name),
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
		}
		var envelope struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(raw, &envelope); err == nil {
			entry.Schema = envelope.Schema
		}
		artifacts = append(artifacts, entry)
	}
	manifest := struct {
		Schema    string     `json:"schema"`
		Root      string     `json:"root"`
		Artifacts []artifact `json:"artifacts"`
	}{
		Schema:    "tetra.release-artifact-hashes.v1alpha1",
		Root:      ".",
		Artifacts: artifacts,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal artifact hashes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "artifact-hashes.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write artifact hashes: %v", err)
	}
}
