package surfaceexamples

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSurfaceExampleSuiteValidateReportAcceptsProductionExamples(t *testing.T) {
	if err := ValidateReport(mustExampleSuiteJSON(t, validExampleSuiteReport())); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestSurfaceExampleSuiteValidateReportRejectsScreenshotOnlyEvidence(t *testing.T) {
	report := validExampleSuiteReport()
	report.Examples[0].ScreenshotOnly = true
	report.Examples[0].Executable = false
	report.NegativeGuards.ScreenshotOnlyRejected = false

	err := ValidateReport(mustExampleSuiteJSON(t, report))
	if err == nil {
		t.Fatal("expected screenshot-only evidence to be rejected")
	}
	if !strings.Contains(err.Error(), "screenshot") {
		t.Fatalf("error = %q, want screenshot rejection", err.Error())
	}
}

func TestSurfaceExampleSuiteValidateReportRejectsReactElectronDOMRuntime(t *testing.T) {
	report := validExampleSuiteReport()
	report.Examples[1].RequiresReact = true
	report.Examples[2].RequiresElectron = true
	report.Examples[3].RequiresDOMRuntime = true
	report.NegativeGuards.ReactElectronDOMRejected = false

	err := ValidateReport(mustExampleSuiteJSON(t, report))
	if err == nil {
		t.Fatal("expected React/Electron/DOM runtime dependency to be rejected")
	}
	if !strings.Contains(err.Error(), "React/Electron/DOM") {
		t.Fatalf("error = %q, want React/Electron/DOM rejection", err.Error())
	}
}

func TestSurfaceExampleSuiteValidateReportRejectsMissingSupportedTarget(t *testing.T) {
	report := validExampleSuiteReport()
	report.Targets = report.Targets[:2]
	report.NegativeGuards.MissingTargetCoverageRejected = false

	err := ValidateReport(mustExampleSuiteJSON(t, report))
	if err == nil {
		t.Fatal("expected missing supported target to be rejected")
	}
	if !strings.Contains(err.Error(), "target") {
		t.Fatalf("error = %q, want target coverage rejection", err.Error())
	}
}

func TestSurfaceExampleSuiteValidateReportRejectsWidgetsWhereBlockMorphRequired(t *testing.T) {
	report := validExampleSuiteReport()
	report.Examples[4].UsesWidgets = true
	report.Examples[4].UsesBlock = false
	report.NegativeGuards.WidgetsWhereBlockMorphRequiredRejected = false

	err := ValidateReport(mustExampleSuiteJSON(t, report))
	if err == nil {
		t.Fatal("expected widget-backed production example to be rejected")
	}
	if !strings.Contains(err.Error(), "Block/Morph") {
		t.Fatalf("error = %q, want Block/Morph rejection", err.Error())
	}
}

func validExampleSuiteReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSurfaceProductionExamplesV1,
		Scope:        "surface-prod-realistic-app-shapes-linux-web",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Examples: []ExampleEvidence{
			validExample("examples/surface_prod_command_palette.tetra", "command_palette"),
			validExample("examples/surface_prod_settings_app.tetra", "settings"),
			validExample("examples/surface_prod_project_dashboard.tetra", "project_dashboard"),
			validExample("examples/surface_prod_editor_shell.tetra", "editor_shell"),
			validExample("examples/surface_prod_file_manager_shell.tetra", "file_manager_shell"),
			validExample("examples/surface_prod_multi_window_notes.tetra", "multi_window_notes"),
			validExample("examples/surface_prod_system_tray_status.tetra", "system_tray_status"),
			validExample("examples/surface_prod_notification_dialog.tetra", "notification_dialog"),
			validLocalizedExample(),
			validAccessibilityExample(),
		},
		Targets: []TargetEvidence{
			{Target: "headless", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-headless.json"},
			{Target: "linux-x64", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-linux-x64.json"},
			{Target: "wasm32-web", ExampleCount: 10, Ran: true, Pass: true, Artifact: "surface-prod-examples-wasm32-web.json"},
		},
		Ecosystem: EcosystemSeed{
			TemplateCount:        6,
			PackageReportCount:   2,
			ExamplesIndexUpdated: true,
			SurfaceGuideUpdated:  true,
			ScaffoldSmokeRan:     true,
			PackageSmokeRan:      true,
		},
		NegativeGuards: NegativeGuards{
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
		Cases: []CaseReport{
			{Name: "ten realistic app shapes", Kind: "positive", Ran: true, Pass: true},
			{Name: "all scoped targets covered", Kind: "positive", Ran: true, Pass: true},
			{Name: "screenshot-only examples rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "React/Electron/DOM runtime examples rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "widgets where Block/Morph required rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func validExample(path string, shape string) ExampleEvidence {
	return ExampleEvidence{
		Path:                 path,
		Shape:                shape,
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
}

func validLocalizedExample() ExampleEvidence {
	example := validExample("examples/surface_prod_localized_form.tetra", "localized_form")
	example.HasLocalization = true
	return example
}

func validAccessibilityExample() ExampleEvidence {
	example := validExample("examples/surface_prod_accessibility_heavy_form.tetra", "accessibility_heavy_form")
	example.HasAccessibilityStress = true
	return example
}

func mustExampleSuiteJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
