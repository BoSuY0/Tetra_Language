package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceprod"
)

func TestValidateSurfaceProdClaimAcceptsValidReportFile(t *testing.T) {
	path := writeSurfaceProdClaimFixture(t, validSurfaceProdClaimReport())
	if err := validateSurfaceProdClaim(path); err != nil {
		t.Fatalf("validateSurfaceProdClaim failed: %v", err)
	}
}

func TestValidateSurfaceProdClaimRejectsBroadElectronOverclaim(t *testing.T) {
	report := validSurfaceProdClaimReport()
	report.Summary = "Surface fully replaces Electron, React, and CSS for all production UI."
	path := writeSurfaceProdClaimFixture(t, report)

	err := validateSurfaceProdClaim(path)
	if err == nil {
		t.Fatalf("expected broad Electron overclaim to fail")
	}
	for _, want := range []string{"electron", "all production ui"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func writeSurfaceProdClaimFixture(t *testing.T, report surfaceprod.ClaimReport) string {
	t.Helper()
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "surface-prod-claim.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func validSurfaceProdClaimReport() surfaceprod.ClaimReport {
	return surfaceprod.ClaimReport{
		Schema:    surfaceprod.SchemaV1,
		Status:    "pass",
		ClaimTier: surfaceprod.ClaimTierProdStableScopedLinuxWebApp,
		Scope:     surfaceprod.ScopeSurfaceProdScopedLinuxWeb,
		Summary:   "Scoped Linux/web Surface production claim with explicit Electron, React, CSS, GPU, accessibility, and cross-platform nonclaims.",
		Producer:  "tools/cmd/validate-surface-prod-claim",
		GitHead:   "0123456789abcdef0123456789abcdef01234567",
		GitDirty:  false,
		RuntimeDependencyPolicy: surfaceprod.RuntimeDependencyPolicy{
			Electron:             false,
			ChromiumDesktopShell: false,
			ReactRuntime:         false,
			DOMUI:                false,
			CSSRuntime:           false,
			UserJSAppLogic:       false,
			PlatformWidgets:      false,
		},
		Capabilities: surfaceprod.CapabilityClaims{
			Renderer:                   "software-rgba",
			GPUProduction:              false,
			CrossPlatformDesktopParity: false,
			AccessibilityLevel:         "scoped-platform-bridge-v1",
			FullAccessibilityParity:    false,
		},
		SupportedTargets: []surfaceprod.TargetClaim{
			{Target: "headless", SupportLevel: "test-evidence", Evidence: "reports/surface-release-v1/surface-headless-release-smoke.json"},
			{Target: "linux-x64", SupportLevel: "production", Evidence: "reports/surface-release-v1/surface-linux-x64-release-window.json"},
			{Target: "wasm32-web", SupportLevel: "production", Evidence: "reports/surface-release-v1/surface-wasm32-web-release-browser.json"},
		},
		UnsupportedTargets: []string{"macos-x64", "windows-x64", "wasm32-wasi"},
		NonClaims: []string{
			"not a broad Electron replacement",
			"not cross-platform desktop parity",
			"not GPU production rendering",
			"not full accessibility parity",
			"not a CSS cascade runtime",
		},
		TargetHostEvidence: []surfaceprod.TargetHostEvidence{
			{Target: "linux-x64", Host: "linux-x64", Level: "target-host", RealWindow: true, NativeInput: true, BrowserCanvas: false, SameCommit: true, Report: "reports/surface-release-v1/surface-linux-x64-release-window.json"},
			{Target: "wasm32-web", Host: "chromium-linux", Level: "browser-canvas", RealWindow: false, NativeInput: false, BrowserCanvas: true, SameCommit: true, Report: "reports/surface-release-v1/surface-wasm32-web-release-browser.json"},
		},
		GateEvidence: []surfaceprod.GateEvidence{
			{Name: "surface release state", Status: "pass", Evidence: "scripts/release/surface/release-gate.sh"},
			{Name: "renderer backend decision gate", Status: "pass", Evidence: "tools/cmd/validate-surface-renderer-report"},
			{Name: "claim taxonomy negative fixtures", Status: "pass", Evidence: "tools/validators/surfaceprod/report_test.go"},
		},
		Cases: []surfaceprod.CaseReport{
			{Name: "fake electron/react/css replacement rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "fake cross-platform support rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "fake gpu production claim rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "gpu production without target-host backend reports rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "fake full accessibility parity rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing target-host evidence rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}
