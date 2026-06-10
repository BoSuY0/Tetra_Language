package surfacemigration

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSurfaceMigrationValidateReportAcceptsCompatibilityEvidence(t *testing.T) {
	if err := ValidateReport(mustReportJSON(t, validMigrationReport())); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestSurfaceMigrationValidateReportRejectsWidgetsAsCoreFinalArchitecture(t *testing.T) {
	report := validMigrationReport()
	report.Policy.WidgetsCoreFinalArchitecture = true
	report.NegativeGuards.WidgetsCoreFinalRejected = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected widgets-as-core-final-architecture claim to be rejected")
	}
	if !strings.Contains(err.Error(), "core final architecture") {
		t.Fatalf("error = %q, want core final architecture rejection", err.Error())
	}
}

func TestSurfaceMigrationValidateReportRejectsBrokenV1Examples(t *testing.T) {
	report := validMigrationReport()
	report.Examples[0].Pass = false
	report.NegativeGuards.BreakingV1ExamplesRejected = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected broken v1 example to be rejected")
	}
	if !strings.Contains(err.Error(), "v1 example") {
		t.Fatalf("error = %q, want v1 example rejection", err.Error())
	}
}

func TestSurfaceMigrationValidateReportRejectsMissingBlockMorphRecommendation(t *testing.T) {
	report := validMigrationReport()
	report.Policy.DocsRecommendBlockMorph = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected missing Block/Morph recommendation to be rejected")
	}
	if !strings.Contains(err.Error(), "Block/Morph") {
		t.Fatalf("error = %q, want Block/Morph recommendation rejection", err.Error())
	}
}

func validMigrationReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSurfaceMigrationV1,
		Scope:        "surface-v1-widget-compat-to-block-morph",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: MigrationPolicy{
			CompatibilityLayer:            true,
			DocsRecommendBlockMorph:       true,
			WidgetsCoreFinalArchitecture:  false,
			DeprecationBeforeCoverage:     false,
			BreakV1ExamplesAllowed:        false,
			MigrationDiagnosticsAvailable: true,
		},
		Mappings: []WidgetMapping{
			{Widget: "Panel", ComponentKind: "panel", BlockLayout: "column", MorphRecipe: "region_panel", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true, Deprecated: false},
			{Widget: "Button", ComponentKind: "button", BlockLayout: "row", MorphRecipe: "control_action", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true, Deprecated: false},
			{Widget: "TextBox", ComponentKind: "textbox", BlockLayout: "row", MorphRecipe: "field_text", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true, Deprecated: false},
			{Widget: "StatusText", ComponentKind: "text", BlockLayout: "fixed", MorphRecipe: "status_message", CompatibilityLayer: true, BlockEquivalent: true, MorphRecommended: true, Deprecated: false},
		},
		Examples: []ExampleEvidence{
			{Path: "examples/surface_toolkit_form.tetra", Kind: "v1-widget", Ran: true, Pass: true, UsesWidgets: true, UsesBlock: false, UsesMorph: false},
			{Path: "examples/surface_migration_widgets_to_block.tetra", Kind: "migration", Ran: true, Pass: true, UsesWidgets: true, UsesBlock: true, UsesMorph: true},
		},
		Diagnostics: []DiagnosticEvidence{
			{Code: "surface.migration.use_block_morph", Message: "new production UI should prefer Block/Morph recipes", Emitted: true},
			{Code: "surface.migration.widgets_not_final_core", Message: "widgets remain compatibility helpers", Emitted: true},
		},
		NegativeGuards: NegativeGuards{
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
		Cases: []CaseReport{
			{Name: "existing Surface v1 widget examples still pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "widgets map to Block/Morph recipes", Kind: "positive", Ran: true, Pass: true},
			{Name: "widgets as core final architecture rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "breaking v1 examples without migration rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func mustReportJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
