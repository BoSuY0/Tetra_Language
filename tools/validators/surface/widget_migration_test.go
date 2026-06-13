package surface

import (
	"strings"
	"testing"
)

func TestValidateWidgetMigrationReportAcceptsCompatibilityEvidence(t *testing.T) {
	raw := validWidgetMigrationReportJSON()
	if err := ValidateWidgetMigrationReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateWidgetMigrationReport failed: %v\n%s", err, raw)
	}
}

func TestValidateWidgetMigrationReportRejectsFutureCorePrimitivePromotion(t *testing.T) {
	raw := strings.Replace(validWidgetMigrationReportJSON(), `"block_only_core_primitive": true`, `"block_only_core_primitive": false`, 1)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected future core primitive promotion to fail")
	}
	if !strings.Contains(err.Error(), "core primitive") {
		t.Fatalf("error = %v, want core primitive diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsWidgetPromotedToCore(t *testing.T) {
	raw := strings.Replace(validWidgetMigrationReportJSON(), `"widgets_promoted_to_core": false`, `"widgets_promoted_to_core": true`, 1)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected promoted widget core primitive to fail")
	}
	if !strings.Contains(err.Error(), "widgets_promoted_to_core") {
		t.Fatalf("error = %v, want widgets_promoted_to_core diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsBreakingChange(t *testing.T) {
	raw := strings.Replace(validWidgetMigrationReportJSON(), `"api_breaking_change": false`, `"api_breaking_change": true`, 1)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected API breaking change to fail")
	}
	if !strings.Contains(err.Error(), "breaking") {
		t.Fatalf("error = %v, want breaking-change diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsDocsOnlyWithoutArtifactEvidence(t *testing.T) {
	raw := strings.Replace(validWidgetMigrationReportJSON(), `  "artifact_evidence": {
    "equivalence_rows_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "source_scan_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  },
`, "", 1)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing artifact evidence to fail")
	}
	if !strings.Contains(err.Error(), "artifact_evidence") {
		t.Fatalf("error = %v, want artifact_evidence diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsMissingTextboxEquivalence(t *testing.T) {
	raw := strings.Replace(validWidgetMigrationReportJSON(), `"legacy_widget":"TextBox"`, `"legacy_widget":"TextBoxLegacy"`, 1)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing TextBox equivalence to fail")
	}
	if !strings.Contains(err.Error(), "TextBox") {
		t.Fatalf("error = %v, want TextBox diagnostic", err)
	}
}

func validWidgetMigrationReportJSON() string {
	return `{
  "schema": "tetra.surface.widget-migration.v1",
  "model": "surface-widget-migration-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-widget-migration-smoke.sh",
  "source": "examples/surface_reference_migration.tetra",
  "reference_app": "migration",
  "target": "linux-x64",
  "compatibility_layer": {
    "module": "lib.core.widgets",
    "supported_surface_v1": true,
    "current_api_preserved": true,
    "api_breaking_change": false,
    "migration_equivalence_helpers": true,
    "migration_docs": true,
    "pass": true
  },
  "release_widget_set": {
    "widgets": ["Text","Label","StatusText","Button","TextBox","Row","Column","Panel","Checkbox","Stack","Scroll","Spacer"],
    "intact": true,
    "non_migration_widget_usage": false,
    "pass": true
  },
  "equivalence_rows": [
    {"legacy_widget":"Panel","legacy_function":"widgets.panel_init","morph_recipe":"recipe_region_panel","block_expander":"morph.expand_region_panel","block_kind":"Block","legacy_result":380,"block_result":380,"api_unchanged":true,"resolves_to_block":true,"pass":true},
    {"legacy_widget":"Button","legacy_function":"widgets.button_init","morph_recipe":"recipe_control_action","block_expander":"morph.expand_control_action","block_kind":"Block","legacy_result":1301,"block_result":1301,"api_unchanged":true,"resolves_to_block":true,"pass":true},
    {"legacy_widget":"TextBox","legacy_function":"widgets.textbox_init","morph_recipe":"recipe_field_text","block_expander":"morph.expand_field_text","block_kind":"Block","legacy_result":344,"block_result":344,"api_unchanged":true,"resolves_to_block":true,"pass":true}
  ],
  "morph_recipe_migration": {
    "recipes": ["recipe_region_panel","recipe_control_action","recipe_field_text"],
    "core_primitives": ["Block"],
    "block_only_core_primitive": true,
    "widgets_promoted_to_core": false,
    "resolves_to_block": true,
    "pass": true
  },
  "migration_reference_app": {
    "shape": "migration",
    "source": "examples/surface_reference_migration.tetra",
    "imports": ["lib.core.surface","lib.core.block","lib.core.morph","lib.core.widgets"],
    "compiles": true,
    "runs": true,
    "exit_code": 0,
    "uses_widgets_compat": true,
    "uses_morph_recipes": true,
    "resolves_to_block": true,
    "pass": true
  },
  "negative_guards": {
    "no_future_core_primitive_promotion": true,
    "no_widget_primary_future_core": true,
    "no_breaking_change": true,
    "no_docs_only": true,
    "no_platform_native_runtime_claims": true
  },
  "artifact_evidence": {
    "equivalence_rows_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "source_scan_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  },
  "pass": true
}
`
}
