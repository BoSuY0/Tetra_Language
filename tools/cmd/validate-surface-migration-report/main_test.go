package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacemigration"
)

func TestSurfaceMigrationReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandMigrationReport()
	reportPath := filepath.Join(dir, "surface-migration-report.json")
	writeMigrationJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceMigrationReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface migration report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSurfaceMigrationReportCommandRejectsBrokenV1Example(t *testing.T) {
	dir := t.TempDir()
	report := commandMigrationReport()
	report.Examples[0].Pass = false
	reportPath := filepath.Join(dir, "surface-migration-report.json")
	writeMigrationJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceMigrationReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "v1 example") {
		t.Fatalf("stderr = %q, want v1 example rejection", stderr.String())
	}
}

func commandMigrationReport() surfacemigration.Report {
	return surfacemigration.Report{
		Schema:       surfacemigration.SchemaV1,
		Status:       "pass",
		Level:        surfacemigration.LevelSurfaceMigrationV1,
		Scope:        "surface-v1-widget-compat-to-block-morph",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: surfacemigration.MigrationPolicy{
			CompatibilityLayer:            true,
			DocsRecommendBlockMorph:       true,
			WidgetsCoreFinalArchitecture:  false,
			DeprecationBeforeCoverage:     false,
			BreakV1ExamplesAllowed:        false,
			MigrationDiagnosticsAvailable: true,
		},
		Mappings: []surfacemigration.WidgetMapping{
			{Widget: "Panel", ComponentKind: "panel", BlockLayout: "column", MorphRecipe: "region_panel", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true},
			{Widget: "Button", ComponentKind: "button", BlockLayout: "row", MorphRecipe: "control_action", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true},
			{Widget: "TextBox", ComponentKind: "textbox", BlockLayout: "row", MorphRecipe: "field_text", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true},
			{Widget: "StatusText", ComponentKind: "text", BlockLayout: "fixed", MorphRecipe: "status_message", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true},
		},
		Examples: []surfacemigration.ExampleEvidence{
			{Path: "examples/surface_toolkit_form.tetra", Kind: "v1-widget", Ran: true, Pass: true, UsesWidgets: true},
			{Path: "examples/surface_migration_widgets_to_block.tetra", Kind: "migration", Ran: true, Pass: true, UsesWidgets: true, UsesBlock: true, UsesMorph: true},
		},
		Diagnostics: []surfacemigration.DiagnosticEvidence{
			{Code: "surface.migration.use_block_morph", Message: "new production UI should prefer Block/Morph recipes", Emitted: true},
			{Code: "surface.migration.widgets_not_final_core", Message: "widgets remain compatibility helpers", Emitted: true},
		},
		NegativeGuards: surfacemigration.NegativeGuards{
			WidgetsCoreFinalRejected:      true,
			BreakingV1ExamplesRejected:    true,
			MissingMappingRejected:        true,
			DeprecationBeforeCoverage:     true,
			MissingBlockMorphDocsRejected: true,
		},
		NonClaims: []string{
			"Widgets are not the core final architecture.",
			"No deprecation before production examples and gates cover replacement.",
			"No breaking Surface v1 examples without migration.",
			"New production UI should prefer Block/Morph recipes.",
		},
		Cases: []surfacemigration.CaseReport{
			{Name: "existing Surface v1 widget examples still pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "widgets map to Block/Morph recipes", Kind: "positive", Ran: true, Pass: true},
			{Name: "widgets as core final architecture rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "breaking v1 examples without migration rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func writeMigrationJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
