package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceexamples"
)

func TestValidateSurfaceExampleSuiteCommandAcceptsValidReport(t *testing.T) {
	reportPath := writeExampleSuiteFixture(t, validExampleSuiteCommandReport())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runValidateSurfaceExampleSuite([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface example suite report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceExampleSuiteCommandRejectsScreenshotOnlyReport(t *testing.T) {
	report := validExampleSuiteCommandReport()
	report.Examples[0].ScreenshotOnly = true
	report.Examples[0].Executable = false
	report.NegativeGuards.ScreenshotOnlyRejected = false
	reportPath := writeExampleSuiteFixture(t, report)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runValidateSurfaceExampleSuite([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected screenshot-only report rejection")
	}
	if !strings.Contains(stderr.String(), "screenshot") {
		t.Fatalf("stderr = %q, want screenshot rejection", stderr.String())
	}
}

func writeExampleSuiteFixture(t *testing.T, report surfaceexamples.Report) string {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "surface-example-suite-report.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func validExampleSuiteCommandReport() surfaceexamples.Report {
	report := surfaceexamples.Report{
		Schema:       surfaceexamples.SchemaV1,
		Status:       "pass",
		Level:        surfaceexamples.LevelSurfaceProductionExamplesV1,
		Scope:        "surface-prod-realistic-app-shapes-linux-web",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Targets: []surfaceexamples.TargetEvidence{
			{Target: "headless", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-headless.json"},
			{Target: "linux-x64", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-linux-x64.json"},
			{Target: "wasm32-web", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-wasm32-web.json"},
		},
		Ecosystem: surfaceexamples.EcosystemSeed{
			TemplateCount:        6,
			PackageReportCount:   2,
			ExamplesIndexUpdated: true,
			SurfaceGuideUpdated:  true,
			ScaffoldSmokeRan:     true,
			PackageSmokeRan:      true,
		},
		NegativeGuards: surfaceexamples.NegativeGuards{
			ScreenshotOnlyRejected:                 true,
			ReactElectronDOMRejected:               true,
			WidgetsWhereBlockMorphRequiredRejected: true,
			MissingShapeRejected:                   true,
			MissingTargetCoverageRejected:          true,
			ToyVisualOnlyRejected:                  true,
		},
		NonClaims: []string{
			"Production examples do not claim broad cross-platform parity.",
			"Production examples do not require React, Electron, DOM runtime UI, external CSS, or platform widgets.",
			"Screenshot-only demos are not production example evidence.",
		},
		Cases: []surfaceexamples.CaseReport{
			{Name: "ten realistic app shapes", Kind: "positive", Ran: true, Pass: true},
			{Name: "all scoped targets covered", Kind: "positive", Ran: true, Pass: true},
			{Name: "screenshot-only examples rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "React/Electron/DOM runtime examples rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "widgets where Block/Morph required rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
	for _, item := range []struct {
		path  string
		shape string
	}{
		{"examples/surface_prod_command_palette.tetra", "command_palette"},
		{"examples/surface_prod_settings_app.tetra", "settings"},
		{"examples/surface_prod_project_dashboard.tetra", "project_dashboard"},
		{"examples/surface_prod_editor_shell.tetra", "editor_shell"},
		{"examples/surface_prod_file_manager_shell.tetra", "file_manager_shell"},
		{"examples/surface_prod_multi_window_notes.tetra", "multi_window_notes"},
		{"examples/surface_prod_system_tray_status.tetra", "system_tray_status"},
		{"examples/surface_prod_notification_dialog.tetra", "notification_dialog"},
		{"examples/surface_prod_localized_form.tetra", "localized_form"},
		{"examples/surface_prod_accessibility_heavy_form.tetra", "accessibility_heavy_form"},
	} {
		example := surfaceexamples.ExampleEvidence{
			Path:                 item.path,
			Shape:                item.shape,
			Ran:                  true,
			Pass:                 true,
			Executable:           true,
			UsesBlock:            true,
			UsesMorph:            true,
			HasEvents:            true,
			HasState:             true,
			HasAccessibility:     true,
			HasPerformanceBudget: true,
		}
		if item.shape == "localized_form" {
			example.HasLocalization = true
		}
		if item.shape == "accessibility_heavy_form" {
			example.HasAccessibilityStress = true
		}
		report.Examples = append(report.Examples, example)
	}
	return report
}
