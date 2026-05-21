package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validWebUIRuntimeTrace = "window-mount:ok;root-mount:ok;layout:ok;text:ok;button:ok;input:ok;list:ok;panel:ok;focus:ok;input-event:ok;change:ok;select:ok;click:ok;timer:ok;async-command:ok;redraw-update:ok;error-recovery:ok;ui-event-dispatch:web-command-dispatch;main-exit:ok;stdout:ok;nonzero-exit:ok;failure-propagation:ok;repeated-instantiation:ok;main-instantiation:ok"

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
		RuntimeTrace:       validWebUIRuntimeTrace,
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v0.4.0",
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
		RuntimeTrace:       validWebUIRuntimeTrace,
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v0.4.0",
		UIBundlePath:       uiBundlePath,
		UIModulePath:       uiModulePath,
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("root-module UI bundle should be valid evidence shape: %v", err)
	}
}

func TestValidateWebUISmokeReportAcceptsCommandOperations(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifact(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"commands":[{"name":"increment","statement_count":1}]`, `"commands":[{"name":"increment","statement_count":1,"operations":[{"kind":"state_add","target":"state.count","value":"1"}]}]`, 1))
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
		RuntimeTrace:       validWebUIRuntimeTrace,
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v0.4.0",
		UIBundlePath:       uiBundlePath,
		UIModulePath:       uiModulePath,
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("command operations should be valid tetra.ui.v0.4.0 evidence: %v", err)
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
		RuntimeTrace:       validWebUIRuntimeTrace,
		DOMSnapshot:        domSnapshotPath,
		UISchema:           "tetra.ui.v0.4.0",
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

func TestValidateWebUISmokeReportRejectsTrailingJSONPayload(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.web-ui-smoke.v1alpha1",
  "generated_at": "2026-04-27T12:00:00Z",
  "target": "wasm32-web",
  "ui_scope_active": true,
  "source": "examples/ui_web_smoke.tetra",
  "used_fallback_source": false,
  "automation": "chromium --headless --dump-dom",
  "status": "blocked",
  "blocker": "host browser unavailable"
}
{"schema":"tetra.web-ui-smoke.v1alpha1"}`)
	var report webUISmokeReport
	if err := decodeStrictJSON(raw, &report); err == nil {
		t.Fatalf("expected trailing report payload failure")
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
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
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
		RuntimeTrace:  validWebUIRuntimeTrace,
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
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
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
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   filepath.Join(t.TempDir(), "missing.dom.html"),
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "dom_snapshot") {
		t.Fatalf("expected missing dom_snapshot artifact rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithMissingMountedDOMMarker(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifactWithHTML(t, "<!doctype html><main>Tetra UI</main>\n")
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), `data-tetra-ui="v1"`) {
		t.Fatalf("expected missing mounted DOM marker rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithMissingDOMBindingMarker(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
	domSnapshotPath := writeWebUIDOMSnapshotArtifactWithHTML(t, `<!doctype html>
<main>
  <section data-tetra-ui="v1" data-tetra-runtime="web-production" data-tetra-widget="window">
    <div data-tetra-widget="root">
      <div data-tetra-widget="layout"></div>
      <div data-tetra-widget="panel">
        <span data-tetra-widget="text">Counter</span>
        <input data-tetra-widget="input" value="tetra">
        <select data-tetra-widget="list"><option>item-1</option></select>
        <button data-tetra-widget="button" type="button">Save</button>
      </div>
    </div>
    <div>Tetra UI Shell</div>
    <div>runtime: web command dispatch</div>
    <div>view CounterView (state: CounterState)</div>
    <div>  event click -> increment</div>
  </section>
</main>
`)
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/projects/dogfood_web_ui/src/main.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "ok:0",
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "data-tetra-binding") {
		t.Fatalf("expected missing DOM binding marker rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportAcceptsPassWithoutUIAccessibilityRole(t *testing.T) {
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
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err = validateWebUISmokeReport(report)
	if err != nil {
		t.Fatalf("expected pass without mandatory accessibility role, got %v", err)
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
		RuntimeTrace:  validWebUIRuntimeTrace,
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err = validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected strict ui bundle rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithoutRuntimeTrace(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
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
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "runtime_trace") {
		t.Fatalf("expected runtime_trace rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithIncompleteRuntimeTrace(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
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
		RuntimeTrace:  "main-exit:ok;stdout:ok;nonzero-exit:ok;repeated-instantiation:ok",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "failure-propagation:ok") {
		t.Fatalf("expected missing runtime marker rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithoutUIEventDispatchBoundaryTrace(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
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
		RuntimeTrace:  "main-exit:ok;stdout:ok;nonzero-exit:ok;failure-propagation:ok;repeated-instantiation:ok",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "ui-event-dispatch:web-command-dispatch") {
		t.Fatalf("expected missing UI event dispatch boundary rejection, got %v", err)
	}
}

func TestValidateWebUISmokeReportRejectsPassWithoutProductionUITrace(t *testing.T) {
	uiBundlePath, uiModulePath := writeWebUISidecarArtifacts(t)
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
		RuntimeTrace:  "main-exit:ok;stdout:ok;nonzero-exit:ok;failure-propagation:ok;repeated-instantiation:ok;ui-event-dispatch:web-command-dispatch",
		DOMSnapshot:   domSnapshotPath,
		UISchema:      "tetra.ui.v0.4.0",
		UIBundlePath:  uiBundlePath,
		UIModulePath:  uiModulePath,
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "window-mount:ok") {
		t.Fatalf("expected missing production UI trace rejection, got %v", err)
	}
}

func TestValidateUIBundleSchemaArtifactAcceptsCheckedInSchema(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "..", "docs", "schemas", "tetra.ui.v0.4.0.schema.json")
	if err := validateUIBundleSchemaArtifact(schemaPath); err != nil {
		t.Fatalf("checked-in UI metadata schema artifact should be valid: %v", err)
	}
}

func TestValidateUIBundleSchemaArtifactRejectsInvalidID(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "..", "docs", "schemas", "tetra.ui.v0.4.0.schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"tetra.ui.v0.4.0.schema.json"`, `"tetra.ui.v2.schema.json"`, 1))
	invalidPath := filepath.Join(t.TempDir(), "tetra.ui.v0.4.0.schema.json")
	if err := os.WriteFile(invalidPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateUIBundleSchemaArtifact(invalidPath)
	if err == nil || !strings.Contains(err.Error(), "$id") {
		t.Fatalf("expected schema artifact $id rejection, got %v", err)
	}
}

func TestValidateUIBundleSchemaArtifactRejectsTrailingJSONPayload(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "..", "docs", "schemas", "tetra.ui.v0.4.0.schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	trailingPath := filepath.Join(t.TempDir(), "tetra.ui.v0.4.0.schema.json")
	if err := os.WriteFile(trailingPath, append(raw, []byte(`{"$id":"tetra.ui.v0.4.0.schema.json"}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateUIBundleSchemaArtifact(trailingPath)
	if err == nil {
		t.Fatalf("expected trailing schema artifact payload failure")
	}
}

func TestValidateUIBundleArtifactRejectsMissingRequiredNestedField(t *testing.T) {
	uiBundlePath, _ := writeWebUISidecarArtifacts(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"source":"state.count"`, `"source":""`, 1))
	if err := os.WriteFile(uiBundlePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = validateUIBundleArtifact(uiBundlePath)
	if err == nil || !strings.Contains(err.Error(), "binding countValue missing source") {
		t.Fatalf("expected missing binding source rejection, got %v", err)
	}
}

func TestValidateUIBundleArtifactRejectsTrailingJSONPayload(t *testing.T) {
	uiBundlePath, _ := writeWebUISidecarArtifacts(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uiBundlePath, append(raw, []byte(`{"schema":"tetra.ui.v0.4.0"}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = validateUIBundleArtifact(uiBundlePath)
	if err == nil {
		t.Fatalf("expected trailing ui bundle payload failure")
	}
}

func TestValidateUIBundleArtifactRejectsUnsupportedCommandOperation(t *testing.T) {
	uiBundlePath, _ := writeWebUISidecarArtifacts(t)
	raw, err := os.ReadFile(uiBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"commands":[{"name":"increment","statement_count":1}]`, `"commands":[{"name":"increment","statement_count":1,"operations":[{"kind":"network_fetch","target":"state.count"}]}]`, 1))
	if err := os.WriteFile(uiBundlePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = validateUIBundleArtifact(uiBundlePath)
	if err == nil || !strings.Contains(err.Error(), "unsupported operation kind") {
		t.Fatalf("expected unsupported command operation rejection, got %v", err)
	}
}

func writeWebUISidecarArtifacts(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	uiBundlePath := filepath.Join(dir, "app.ui.json")
	uiModulePath := filepath.Join(dir, "app.ui.web.mjs")
	if err := os.WriteFile(uiBundlePath, []byte(`{
  "schema":"tetra.ui.v0.4.0",
  "states":[{"name":"CounterState","module":"main","fields":[{"name":"count","type":"i32","mutable":true,"const":false,"init":"0"}]}],
  "views":[{
    "name":"CounterView",
    "module":"main",
    "state_type":"CounterState",
    "bindings":[{"name":"countValue","type":"i32","source":"state.count"}],
    "events":[{"name":"click","command":"increment"}],
    "commands":[{"name":"increment","statement_count":1}],
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
	return writeWebUIDOMSnapshotArtifactWithHTML(t, `<!doctype html>
<main>
  <section data-tetra-ui="v1" data-tetra-runtime="web-production" data-tetra-widget="window">
    <div>Tetra UI Shell</div>
    <div>runtime: web command dispatch</div>
    <div>view CounterView (state: CounterState)</div>
    <div data-tetra-widget="root">
      <div data-tetra-widget="layout"></div>
      <div data-tetra-widget="panel">
        <span data-tetra-widget="text">Counter</span>
        <input data-tetra-widget="input" value="tetra">
        <select data-tetra-widget="list"><option>item-1</option></select>
        <button data-tetra-widget="button" type="button">Save</button>
      </div>
    </div>
    <div data-tetra-binding="countValue">  bind countValue: i32 = 0</div>
    <div>  event click -&gt; increment</div>
  </section>
</main>
`)
}

func writeWebUIDOMSnapshotArtifactWithHTML(t *testing.T, html string) string {
	t.Helper()
	domSnapshotPath := filepath.Join(t.TempDir(), "web-ui-smoke.dom.html")
	if err := os.WriteFile(domSnapshotPath, []byte(html), 0o644); err != nil {
		t.Fatalf("write DOM snapshot artifact: %v", err)
	}
	return domSnapshotPath
}
