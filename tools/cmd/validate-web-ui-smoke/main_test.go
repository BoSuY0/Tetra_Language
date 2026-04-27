package main

import (
	"strings"
	"testing"
)

func TestValidateWebUISmokeReportAcceptsPass(t *testing.T) {
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
	}
	if err := validateWebUISmokeReport(report); err != nil {
		t.Fatalf("validateWebUISmokeReport: %v", err)
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
	report := webUISmokeReport{
		Schema:        "tetra.web-ui-smoke.v1alpha1",
		GeneratedAt:   "2026-04-27T12:00:00Z",
		Target:        "wasm32-web",
		UIScopeActive: true,
		Source:        "examples/ui_web_smoke.tetra",
		Automation:    "chromium --headless --dump-dom",
		Status:        "pass",
		Result:        "pending",
	}
	err := validateWebUISmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "ok:") {
		t.Fatalf("expected missing ok result rejection, got %v", err)
	}
}
