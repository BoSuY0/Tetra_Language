package surface

import (
	"errors"
	"fmt"
	"strings"
)

const WidgetMigrationSchemaV1 = "tetra.surface.widget-migration.v1"

type SurfaceWidgetMigrationReportV1 struct {
	Schema                string                                 `json:"schema"`
	Model                 string                                 `json:"model"`
	ReleaseScope          string                                 `json:"release_scope"`
	Producer              string                                 `json:"producer"`
	Source                string                                 `json:"source"`
	ReferenceApp          string                                 `json:"reference_app"`
	Target                string                                 `json:"target"`
	CompatibilityLayer    SurfaceWidgetMigrationCompatibility    `json:"compatibility_layer"`
	ReleaseWidgetSet      SurfaceWidgetMigrationReleaseSet       `json:"release_widget_set"`
	EquivalenceRows       []SurfaceWidgetMigrationEquivalence    `json:"equivalence_rows"`
	MorphRecipeMigration  SurfaceWidgetMigrationMorphRecipes     `json:"morph_recipe_migration"`
	MigrationReferenceApp SurfaceWidgetMigrationReferenceApp     `json:"migration_reference_app"`
	NegativeGuards        SurfaceWidgetMigrationNegativeGuards   `json:"negative_guards"`
	ArtifactEvidence      SurfaceWidgetMigrationArtifactEvidence `json:"artifact_evidence,omitempty"`
	Pass                  bool                                   `json:"pass"`
}

type SurfaceWidgetMigrationCompatibility struct {
	Module                      string `json:"module"`
	SupportedSurfaceV1          bool   `json:"supported_surface_v1"`
	CurrentAPIPreserved         bool   `json:"current_api_preserved"`
	APIBreakingChange           bool   `json:"api_breaking_change"`
	MigrationEquivalenceHelpers bool   `json:"migration_equivalence_helpers"`
	MigrationDocs               bool   `json:"migration_docs"`
	Pass                        bool   `json:"pass"`
}

type SurfaceWidgetMigrationReleaseSet struct {
	Widgets                 []string `json:"widgets"`
	Intact                  bool     `json:"intact"`
	NonMigrationWidgetUsage bool     `json:"non_migration_widget_usage"`
	Pass                    bool     `json:"pass"`
}

type SurfaceWidgetMigrationEquivalence struct {
	LegacyWidget    string `json:"legacy_widget"`
	LegacyFunction  string `json:"legacy_function"`
	MorphRecipe     string `json:"morph_recipe"`
	BlockExpander   string `json:"block_expander"`
	BlockKind       string `json:"block_kind"`
	LegacyResult    int    `json:"legacy_result"`
	BlockResult     int    `json:"block_result"`
	APIUnchanged    bool   `json:"api_unchanged"`
	ResolvesToBlock bool   `json:"resolves_to_block"`
	Pass            bool   `json:"pass"`
}

type SurfaceWidgetMigrationMorphRecipes struct {
	Recipes                []string `json:"recipes"`
	CorePrimitives         []string `json:"core_primitives"`
	BlockOnlyCorePrimitive bool     `json:"block_only_core_primitive"`
	WidgetsPromotedToCore  bool     `json:"widgets_promoted_to_core"`
	ResolvesToBlock        bool     `json:"resolves_to_block"`
	Pass                   bool     `json:"pass"`
}

type SurfaceWidgetMigrationReferenceApp struct {
	Shape             string   `json:"shape"`
	Source            string   `json:"source"`
	Imports           []string `json:"imports"`
	Compiles          bool     `json:"compiles"`
	Runs              bool     `json:"runs"`
	ExitCode          int      `json:"exit_code"`
	UsesWidgetsCompat bool     `json:"uses_widgets_compat"`
	UsesMorphRecipes  bool     `json:"uses_morph_recipes"`
	ResolvesToBlock   bool     `json:"resolves_to_block"`
	Pass              bool     `json:"pass"`
}

type SurfaceWidgetMigrationNegativeGuards struct {
	NoFutureCorePrimitivePromotion bool `json:"no_future_core_primitive_promotion"`
	NoWidgetPrimaryFutureCore      bool `json:"no_widget_primary_future_core"`
	NoBreakingChange               bool `json:"no_breaking_change"`
	NoDocsOnly                     bool `json:"no_docs_only"`
	NoPlatformNativeRuntimeClaims  bool `json:"no_platform_native_runtime_claims"`
}

type SurfaceWidgetMigrationArtifactEvidence struct {
	EquivalenceRowsSHA256 string `json:"equivalence_rows_sha256,omitempty"`
	SourceScanSHA256      string `json:"source_scan_sha256,omitempty"`
}

func ValidateWidgetMigrationReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != WidgetMigrationSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, WidgetMigrationSchemaV1)
	}
	var report SurfaceWidgetMigrationReportV1
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceWidgetMigrationReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceWidgetMigrationReport(report SurfaceWidgetMigrationReportV1) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: WidgetMigrationSchemaV1},
		{field: "model", got: report.Model, want: "surface-widget-migration-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-widget-migration-smoke.sh"},
		{field: "source", got: report.Source, want: "examples/surface_reference_migration.tetra"},
		{field: "reference_app", got: report.ReferenceApp, want: "migration"},
		{field: "target", got: report.Target, want: "linux-x64"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(issues, fmt.Sprintf("reference_app %q does not match source %q", report.ReferenceApp, report.Source))
	}
	issues = append(issues, validateSurfaceWidgetMigrationCompatibility(report.CompatibilityLayer)...)
	issues = append(issues, validateSurfaceWidgetMigrationReleaseSet(report.ReleaseWidgetSet)...)
	issues = append(issues, validateSurfaceWidgetMigrationEquivalence(report.EquivalenceRows)...)
	issues = append(issues, validateSurfaceWidgetMigrationMorphRecipes(report.MorphRecipeMigration)...)
	issues = append(issues, validateSurfaceWidgetMigrationReferenceApp(report.MigrationReferenceApp)...)
	issues = append(issues, validateSurfaceWidgetMigrationNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateSurfaceWidgetMigrationArtifactEvidence(report.ArtifactEvidence)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationCompatibility(layer SurfaceWidgetMigrationCompatibility) []string {
	var issues []string
	if layer.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("compatibility_layer module is %q, want lib.core.widgets", layer.Module))
	}
	if !layer.SupportedSurfaceV1 {
		issues = append(issues, "lib.core.widgets must remain supported for Surface v1")
	}
	if !layer.CurrentAPIPreserved {
		issues = append(issues, "lib.core.widgets current API must be preserved")
	}
	if layer.APIBreakingChange {
		issues = append(issues, "lib.core.widgets API breaking change must be false")
	}
	if !layer.MigrationEquivalenceHelpers {
		issues = append(issues, "compatibility_layer must record migration equivalence helpers")
	}
	if !layer.MigrationDocs {
		issues = append(issues, "compatibility_layer requires migration docs evidence")
	}
	if !layer.Pass {
		issues = append(issues, "compatibility_layer pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationReleaseSet(set SurfaceWidgetMigrationReleaseSet) []string {
	var issues []string
	issues = append(issues, validateExactStringList("release_widget_set.widgets", set.Widgets, surfaceWidgetMigrationReleaseWidgets())...)
	if !set.Intact {
		issues = append(issues, "release_widget_set intact must be true")
	}
	if set.NonMigrationWidgetUsage {
		issues = append(issues, "non_migration_widget_usage must be false")
	}
	if !set.Pass {
		issues = append(issues, "release_widget_set pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationEquivalence(rows []SurfaceWidgetMigrationEquivalence) []string {
	if len(rows) < 3 {
		return []string{"equivalence_rows require Panel, Button, and TextBox"}
	}
	required := map[string]struct {
		legacyFunction string
		recipe         string
		expander       string
	}{
		"Panel":   {legacyFunction: "widgets.panel_init", recipe: "recipe_region_panel", expander: "morph.expand_region_panel"},
		"Button":  {legacyFunction: "widgets.button_init", recipe: "recipe_control_action", expander: "morph.expand_control_action"},
		"TextBox": {legacyFunction: "widgets.textbox_init", recipe: "recipe_field_text", expander: "morph.expand_field_text"},
	}
	seen := map[string]SurfaceWidgetMigrationEquivalence{}
	var issues []string
	for _, row := range rows {
		widget := strings.TrimSpace(row.LegacyWidget)
		if widget == "" {
			issues = append(issues, "equivalence row legacy_widget is required")
			continue
		}
		if _, ok := seen[widget]; ok {
			issues = append(issues, fmt.Sprintf("duplicate equivalence row for %s", widget))
		}
		seen[widget] = row
		prefix := "equivalence row " + widget
		if row.BlockKind != "Block" {
			issues = append(issues, prefix+" block_kind must be Block")
		}
		if row.LegacyResult <= 0 || row.BlockResult <= 0 || row.LegacyResult != row.BlockResult {
			issues = append(issues, prefix+" legacy_result and block_result must match positive evidence")
		}
		if !row.APIUnchanged {
			issues = append(issues, prefix+" api_unchanged must be true")
		}
		if !row.ResolvesToBlock {
			issues = append(issues, prefix+" resolves_to_block must be true")
		}
		if !row.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	for widget, want := range required {
		row, ok := seen[widget]
		if !ok {
			issues = append(issues, fmt.Sprintf("equivalence_rows missing %s", widget))
			continue
		}
		if row.LegacyFunction != want.legacyFunction {
			issues = append(issues, fmt.Sprintf("%s legacy_function is %q, want %q", widget, row.LegacyFunction, want.legacyFunction))
		}
		if row.MorphRecipe != want.recipe {
			issues = append(issues, fmt.Sprintf("%s morph_recipe is %q, want %q", widget, row.MorphRecipe, want.recipe))
		}
		if row.BlockExpander != want.expander {
			issues = append(issues, fmt.Sprintf("%s block_expander is %q, want %q", widget, row.BlockExpander, want.expander))
		}
	}
	return issues
}

func validateSurfaceWidgetMigrationMorphRecipes(recipes SurfaceWidgetMigrationMorphRecipes) []string {
	var issues []string
	issues = append(issues, validateExactStringList("morph_recipe_migration.recipes", recipes.Recipes, []string{"recipe_region_panel", "recipe_control_action", "recipe_field_text"})...)
	issues = append(issues, validateExactStringList("morph_recipe_migration.core_primitives", recipes.CorePrimitives, []string{"Block"})...)
	if !recipes.BlockOnlyCorePrimitive {
		issues = append(issues, "Block must be the only core primitive; future widget core primitive promotion is rejected")
	}
	if recipes.WidgetsPromotedToCore {
		issues = append(issues, "widgets_promoted_to_core must be false")
	}
	if !recipes.ResolvesToBlock {
		issues = append(issues, "morph_recipe_migration resolves_to_block must be true")
	}
	if !recipes.Pass {
		issues = append(issues, "morph_recipe_migration pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationReferenceApp(app SurfaceWidgetMigrationReferenceApp) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "shape", got: app.Shape, want: "migration"},
		{field: "source", got: app.Source, want: "examples/surface_reference_migration.tetra"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("migration_reference_app %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !safeRelativeSourcePath(app.Source) {
		issues = append(issues, "migration_reference_app source must be a safe Tetra source path")
	}
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph", "lib.core.widgets"} {
		if !templateSmokeContainsString(app.Imports, required) {
			issues = append(issues, fmt.Sprintf("migration_reference_app imports missing %s", required))
		}
	}
	if !app.Compiles {
		issues = append(issues, "migration_reference_app compiles must be true")
	}
	if !app.Runs || app.ExitCode != 0 {
		issues = append(issues, "migration_reference_app must run with exit_code 0")
	}
	if !app.UsesWidgetsCompat {
		issues = append(issues, "migration_reference_app requires lib.core.widgets compatibility usage")
	}
	if !app.UsesMorphRecipes {
		issues = append(issues, "migration_reference_app requires Morph recipe usage")
	}
	if !app.ResolvesToBlock {
		issues = append(issues, "migration_reference_app resolves_to_block must be true")
	}
	if !app.Pass {
		issues = append(issues, "migration_reference_app pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationNegativeGuards(guards SurfaceWidgetMigrationNegativeGuards) []string {
	var issues []string
	for _, guard := range []struct {
		name string
		ok   bool
	}{
		{name: "no_future_core_primitive_promotion", ok: guards.NoFutureCorePrimitivePromotion},
		{name: "no_widget_primary_future_core", ok: guards.NoWidgetPrimaryFutureCore},
		{name: "no_breaking_change", ok: guards.NoBreakingChange},
		{name: "no_docs_only", ok: guards.NoDocsOnly},
		{name: "no_platform_native_runtime_claims", ok: guards.NoPlatformNativeRuntimeClaims},
	} {
		if !guard.ok {
			issues = append(issues, "negative guard "+guard.name+" must be true")
		}
	}
	return issues
}

func validateSurfaceWidgetMigrationArtifactEvidence(evidence SurfaceWidgetMigrationArtifactEvidence) []string {
	var issues []string
	if !validChecksumLike(evidence.EquivalenceRowsSHA256) {
		issues = append(issues, "artifact_evidence equivalence_rows_sha256 must be required sha256 evidence")
	}
	if !validChecksumLike(evidence.SourceScanSHA256) {
		issues = append(issues, "artifact_evidence source_scan_sha256 must be required sha256 evidence")
	}
	return issues
}

func surfaceWidgetMigrationReleaseWidgets() []string {
	return []string{"Text", "Label", "StatusText", "Button", "TextBox", "Row", "Column", "Panel", "Checkbox", "Stack", "Scroll", "Spacer"}
}
