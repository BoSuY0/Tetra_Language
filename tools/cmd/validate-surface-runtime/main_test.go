package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestValidateSurfaceRuntimeReportAcceptsHeadlessEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsProductionTextInputSchema(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-text-input.json")
	raw := []byte(`{
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
  "text_shaping_plan": {"quality_level":"scoped-text-shaping-plan-v1","fallback_fonts":true,"grapheme_boundaries":"byte-offset-codepoint-v1","line_breaking":"newline-storage-plus-wrap-plan-v1","bidi":"nonclaim-full-bidi-v1","rich_text":"nonclaim-rich-text-editor-v1"},
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
}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestTextInputReleaseValidatorAcceptsProductionTextInputReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-text-input.json")
	if err := os.WriteFile(reportPath, validProductionTextInputReportJSON(t), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "text-input"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions text-input failed: %v", err)
	}
}

func TestAppModelReleaseValidatorAcceptsHeadlessCommandReducerReport(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-app-model", []byte("surface app-model fixture\n"), 0o755)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixtureWithSourceAndFrames(t, artifactDir, "examples/surface_app_model.tetra", []surfaceTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	})
	reportPath := filepath.Join(dir, "surface-headless-app-model.json")
	raw := validAppModelReleaseRuntimeReportJSON(t, artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "app-model"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions app-model failed: %v\n%s", err, raw)
	}
}

func TestAppModelReleaseValidatorRejectsHiddenStateRuntimeSubstitute(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-app-model", []byte("surface app-model fixture\n"), 0o755)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless-app-model-hidden-state.json")
	var report surface.Report
	if err := json.Unmarshal(validAppModelReleaseRuntimeReportJSON(t, artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), &report); err != nil {
		t.Fatalf("decode app-model report: %v", err)
	}
	report.AppModel.HiddenAppState = true
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal mutated app-model report: %v", err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err = validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "app-model"})
	if err == nil {
		t.Fatalf("expected hidden app state app-model substitute to fail")
	}
	if !strings.Contains(err.Error(), "app_model") || !strings.Contains(err.Error(), "hidden app state") {
		t.Fatalf("error = %v, want app_model hidden app state diagnostic", err)
	}
}

func TestLinuxAppShellReleaseValidatorAcceptsTargetHostReport(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	reportPath := filepath.Join(dir, "surface-linux-x64-release-app-shell.json")
	raw := validLinuxAppShellReleaseRuntimeReportJSON(t, artifactDir, nil)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "linux-app-shell"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions linux-app-shell failed: %v\n%s", err, raw)
	}
}

func TestLinuxAppShellReleaseValidatorRejectsNativeWidgetSubstitute(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	reportPath := filepath.Join(dir, "surface-linux-x64-release-app-shell-native-widget.json")
	raw := validLinuxAppShellReleaseRuntimeReportJSON(t, artifactDir, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		appShell["negative_guards"].(map[string]any)["no_qt"] = false
	})
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "linux-app-shell"})
	if err == nil {
		t.Fatalf("expected native widget app-shell substitute to fail")
	}
	if !strings.Contains(err.Error(), "GTK/Qt/native widget UI") {
		t.Fatalf("error = %v, want native widget diagnostic", err)
	}
}

func TestLinuxAppShellReleaseValidatorRejectsMissingP16FeatureLedger(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	reportPath := filepath.Join(dir, "surface-linux-x64-release-app-shell-missing-error-report.json")
	raw := validLinuxAppShellReleaseRuntimeReportJSON(t, artifactDir, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		appShell["shell_features"] = withoutLinuxAppShellRuntimeFeature(p16LinuxAppShellRuntimeFeaturesForTest(), "error_report")
	})
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "linux-app-shell"})
	if err == nil {
		t.Fatalf("expected missing P16 error_report ledger row to fail")
	}
	if !strings.Contains(err.Error(), "error_report") {
		t.Fatalf("error = %v, want error_report diagnostic", err)
	}
}

func TestLinuxAppShellReleaseValidatorRejectsMissingP17SecurityPermissions(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	reportPath := filepath.Join(dir, "surface-linux-x64-release-app-shell-missing-security.json")
	raw := validLinuxAppShellReleaseRuntimeReportJSON(t, artifactDir, func(report map[string]any) {
		delete(report, "security_permissions")
	})
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "linux-app-shell"})
	if err == nil {
		t.Fatalf("expected missing P17 security permissions to fail")
	}
	if !strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions diagnostic", err)
	}
}

func TestLinuxAppShellReleaseValidatorRejectsMissingP18PerformanceBudget(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	reportPath := filepath.Join(dir, "surface-linux-x64-release-app-shell-missing-performance.json")
	raw := validLinuxAppShellReleaseRuntimeReportJSON(t, artifactDir, func(report map[string]any) {
		delete(report, "surface_performance_budget")
	})
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "linux-app-shell"})
	if err == nil {
		t.Fatalf("expected missing P18 performance budget to fail")
	}
	if !strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want surface_performance_budget diagnostic", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsReleaseSummarySchema(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-release-summary.json")
	raw := []byte(`{
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
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeAcceptsReleaseSummary(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-release-summary.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReleaseSummaryJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "surface-v1"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions failed: %v", err)
	}
}

func TestHeadlessReleaseValidatorAcceptsHeadlessRuntimeReport(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless-release.json")
	raw := headlessReleaseRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "headless"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions(headless) failed: %v\n%s", err, raw)
	}
}

func TestHeadlessReleaseValidatorRejectsLinuxOrBrowserSubstitute(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-linux-x64.json")
	raw := strings.Replace(
		string(headlessReleaseRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)),
		`"target": "headless"`,
		`"target": "linux-x64"`,
		1,
	)
	raw = strings.Replace(raw, `"runtime": "surface-headless"`, `"runtime": "surface-linux-x64"`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "headless"})
	if err == nil {
		t.Fatalf("expected linux/browser substitute to fail headless release validation")
	}
	if !strings.Contains(err.Error(), "headless") {
		t.Fatalf("error = %v, want headless diagnostic", err)
	}
}

func TestBrowserReleaseRequiresChromium(t *testing.T) {
	dir := t.TempDir()
	reportPath := writeWASM32WebBrowserReleaseRuntimeReport(t, dir)
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	processes := report["processes"].([]any)
	for _, item := range processes {
		process := item.(map[string]any)
		if process["name"] == "surface wasm32-web browser canvas component app" {
			process["path"] = "node scripts/tools/web_run_module.mjs surface-release-form.wasm"
		}
	}
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err = validateWASM32WebBrowserReleaseEnvelope(surface.SchemaV1, raw)
	if err == nil {
		t.Fatalf("expected Node-only browser release substitute to fail")
	}
	if !strings.Contains(err.Error(), "Chromium-compatible browser") {
		t.Fatalf("error = %v, want Chromium-compatible browser diagnostic", err)
	}
}
