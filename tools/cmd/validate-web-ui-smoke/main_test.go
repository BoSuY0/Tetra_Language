package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWebUISmokeReportAcceptsPass(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	report := webUISmokeReport{
		Schema:             "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:        "2026-04-27T12:00:00Z",
		Target:             "wasm32-web",
		UIScopeActive:      true,
		Source:             "examples/projects/dogfood_web_ui/src/main.tetra",
		UsedFallbackSource: false,
		Automation:         "chromium --headless --dump-dom",
		Status:             "pass",
		Result:             "ok:0",
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v1",
		UIBundlePath:       uiBundlePath,
		UIModulePath:       uiModulePath,
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("validateWebUISmokeReport: %v", err)
	}
}

func TestValidateWebUISmokeReportAcceptsRootModuleUIBundle(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.ReplaceAll(string(raw), `"module":"main"`, `"module":""`))
	if err := os.WriteFile(uiBundlePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	report := webUISmokeReport{
		Schema:             "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:        "2026-04-27T12:00:00Z",
		Target:             "wasm32-web",
		UIScopeActive:      true,
		Source:             "examples/projects/dogfood_web_ui/src/main.tetra",
		UsedFallbackSource: false,
		Automation:         "chromium --headless --dump-dom",
		Status:             "pass",
		Result:             "ok:0:ui=1",
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v1",
		UIBundlePath:       uiBundlePath,
		UIModulePath:       uiModulePath,
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("root-module UI bundle should be valid evidence shape: %v", err)
	}
}

func TestValidateWebUISmokeReportAcceptsHostBlockedReport(t *testing.T) {
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "blocked",
		Blocker:       "headless chromium command failed",
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("blocked host report should be valid evidence shape: %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsFallbackPass(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	report := webUISmokeReport{
		Schema:             "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:        "2026-04-27T12:00:00Z",
		Target:             "wasm32-web",
		UIScopeActive:      false,
		Source:             "examples/flow_hello.tetra",
		UsedFallbackSource: true,
		Automation:         "chromium --headless --dump-dom",
		Status:             "pass",
		Result:             "ok:0",
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v1",
		UIBundlePath:       uiBundlePath,
		UIModulePath:       uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "fallback") {
		t.Fatalf("expected fallback pass rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.web-ui-smoke.v1alpha1",
  "generated_at": "2026-04-27T12:00:00Z",
  "target": "wasm32-web",
  "ui_scope_active": true,
  "source": "examples/ui_web_smoke.tetra",
  "used_fallback_source": false,
  "automation": "chromium --headless --dump-dom",
  "status": "pass",
  "result": "ok:0",
  "extra": true
}`)
	var report webUISmokeReport
	if err := decodeStrictJSON(raw, &report); err == nil {
		t.Fatalf("expected unknown field failure")
	}
}

func TestValidateWebUISmokeReportRejectsPassWithoutOKResult(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/ui_web_smoke.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "pending",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v1",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "ok:") {
		t.Fatalf("expected missing ok result rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithoutUISchemaEvidence(t *testing.T) {
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		DOMSnapshot:   "docs/generated/v1_0/web-ui-smoke.dom.html",
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "ui_schema") {
		t.Fatalf("expected ui_schema rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithMissingUISidecarArtifact(t *testing.T) {
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v1",
		UIBundlePath:  filepath.Join(t.TempDir(), "missing.ui.json"),
		UIModulePath:  filepath.Join(t.TempDir(), "missing.ui.web.mjs"),
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "existing artifact") {
		t.Fatalf("expected missing sidecar artifact rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithMissingDOMSnapshot(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		DOMSnapshot:   filepath.Join(t.TempDir(), "missing.dom.html"),
		UISchema:      "tetra.ui.v1",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "dom_snapshot") {
		t.Fatalf("expected missing dom_snapshot artifact rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithMissingUIAccessibilityRole(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `{"name":"role","type":"String","value":"\"button\""},`, "", 1))
	if err := os.WriteFile(uiBundlePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v1",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err = validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "accessibility role") {
		t.Fatalf("expected missing accessibility role rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithUnknownUIBundleFields(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"states":`, `"extra": true, "states":`, 1))
	if err := os.WriteFile(uiBundlePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v1",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err = validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected strict ui bundle rejection, got %v", err)
	}
}

func writeWebUISidecarArtifacts(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	uiBundlePath := filepath.Join(dir, "app.ui.json")
	uiModulePath := filepath.Join(dir, "app.ui.web.mjs")
	if err := os.WriteFile(uiBundlePath, []byte(`{
  "schema":"tetra.ui.v1",
  "states":[{"name":"CounterState","module":"main","fields":[]}],
  "views":[{
    "name":"CounterView",
    "module":"main",
    "state_type":"CounterState",
    "bindings":[],
    "events":[],
    "commands":[],
    "styles":[],
    "accessibility":[{"name":"role","type":"String","value":"\"button\""},{"name":"label","type":"String","value":"\"Increment counter\""}]
  }]
}`+"\n"), 0o644); err != nil {
		t.Fatalf("write ui bundle artifact: %v", err)
	}
	if err := os.WriteFile(uiModulePath, []byte("export function mountTetraUI() {}\n"), 0o644); err != nil {
		t.Fatalf("write ui module artifact: %v", err)
	}
	return uiBundlePath, uiModulePath
}

func writeWebUIDOMSnapshotArtifact(t *testing.T) string {
	t.Helper()
	domSnapshotPath := filepath.Join(t.TempDir(), "web-ui-smoke.dom.html")
	if err := os.WriteFile(domSnapshotPath, []byte("<!doctype html><main>Tetra UI</main>\n"), 0o644); err != nil {
		t.Fatalf("write DOM snapshot artifact: %v", err)
	}
	return domSnapshotPath
}
