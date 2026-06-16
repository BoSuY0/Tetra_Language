package surface

import (
	"strings"
	"testing"
)

func TestValidateReferenceAppsReportAcceptsCompleteSuite(t *testing.T) {
	raw := validReferenceAppsSuiteReportJSON()
	if err := ValidateReferenceAppsReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateReferenceAppsReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReferenceAppsReportRejectsMissingRequiredShape(t *testing.T) {
	raw := strings.Replace(validReferenceAppsSuiteReportJSON(), `,`+referenceAppJSON("migration", "examples/surface_reference_migration.tetra", true), "", 1)
	err := ValidateReferenceAppsReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing migration reference app to fail")
	}
	if !strings.Contains(err.Error(), "migration") {
		t.Fatalf("error = %v, want missing migration diagnostic", err)
	}
}

func TestValidateReferenceAppsReportRejectsScreenshotOnlyEvidence(t *testing.T) {
	raw := strings.Replace(validReferenceAppsSuiteReportJSON(), `"screenshot_only":false`, `"screenshot_only":true`, 1)
	err := ValidateReferenceAppsReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected screenshot-only evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "screenshot") {
		t.Fatalf("error = %v, want screenshot-only diagnostic", err)
	}
}

func TestValidateReferenceAppsReportRejectsProductAppWithoutMorphToPixels(t *testing.T) {
	raw := strings.Replace(validReferenceAppsSuiteReportJSON(), `,"morph_to_pixels":`+validMorphToPixelsChainJSON("examples/surface_reference_command_palette.tetra"), "", 1)
	err := ValidateReferenceAppsReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected product reference app without Morph-to-pixels evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "morph_to_pixels") {
		t.Fatalf("error = %v, want morph_to_pixels diagnostic", err)
	}
}

func validReferenceAppsSuiteReportJSON() string {
	apps := []string{
		referenceAppJSON("command-palette", "examples/surface_reference_command_palette.tetra", false),
		referenceAppJSON("settings", "examples/surface_reference_settings.tetra", false),
		referenceAppJSON("dashboard", "examples/surface_reference_dashboard.tetra", false),
		referenceAppJSON("editor-shell", "examples/surface_reference_editor_shell.tetra", false),
		referenceAppJSON("file-manager", "examples/surface_reference_file_manager.tetra", false),
		referenceAppJSON("dialog-notification", "examples/surface_reference_dialog_notification.tetra", false),
		referenceAppJSON("localized-form", "examples/surface_reference_localized_form.tetra", false),
		referenceAppJSON("accessibility-form", "examples/surface_reference_accessibility_form.tetra", false),
		referenceAppJSON("multi-window-notes", "examples/surface_reference_multi_window_notes.tetra", false),
		referenceAppJSON("migration", "examples/surface_reference_migration.tetra", true),
	}
	return `{"schema":"tetra.surface.reference-app-suite.v1","model":"surface-reference-app-suite-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-reference-apps-smoke.sh","app_count":10,"required_targets":["headless","linux-x64-real-window","wasm32-web-browser-canvas"],"apps":[` + strings.Join(apps, ",") + `],"visual_evidence":{"path":"reference-visual/surface-visual-regression.json","schema":"tetra.surface.visual-regression.v1","app_count":10,"pass":true},"negative_guards":{"screenshot_only_rejected":true,"missing_interaction_rejected":true,"missing_accessibility_rejected":true,"missing_performance_rejected":true,"core_widget_usage_rejected":true,"migration_widgets_compatibility_only":true,"no_react_runtime":true,"no_electron_runtime":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_user_js_app_logic":true},"pass":true}` + "\n"
}

func referenceAppJSON(shape string, source string, compatibility bool) string {
	targets := []string{
		referenceTargetJSON("headless"),
		referenceTargetJSON("linux-x64-real-window"),
		referenceTargetJSON("wasm32-web-browser-canvas"),
	}
	base := `{"shape":"` + shape + `","source":"` + source + `","module":"examples.` + strings.TrimSuffix(strings.TrimPrefix(strings.ReplaceAll(source, "/", "."), "examples."), ".tetra") + `","imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipes":["region.panel","field.text","control.action","command.item"],"beauty_coverage":` + referenceBeautyCoverageJSON(shape) + `,"stable_morph_recipes":true,"resolves_to_block":true,"compiles":true,"runs":true,"exit_code":0,"token_theme_conformance":true,"layout_report":true,"interaction_trace":true,"accessibility_snapshot":true,"performance_budget":true,"artifact_hashes":true,"compatibility_widgets":` + boolJSON(compatibility)
	if compatibility {
		return base + `,"infrastructure_only":true,"non_product_reason":"legacy widget migration compatibility evidence only","targets":[` + strings.Join(targets, ",") + `]}`
	}
	return base + `,"infrastructure_only":false,"morph_to_pixels":` + validMorphToPixelsChainJSON(source) + `,"targets":[` + strings.Join(targets, ",") + `]}`
}

func referenceBeautyCoverageJSON(shape string) string {
	switch shape {
	case "command-palette":
		return `["command-palette","focus-state"]`
	case "settings":
		return `["settings","disabled-state"]`
	case "dashboard":
		return `["dashboard"]`
	case "editor-shell":
		return `["editor-shell"]`
	case "dialog-notification":
		return `["elevated-panel"]`
	case "migration":
		return `[]`
	default:
		return `["focus-state"]`
	}
}

func referenceTargetJSON(target string) string {
	return `{"target":"` + target + `","runtime_report":"reference-runtime/` + target + `.json","frame_checksum":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","visual_diff":true,"interaction_trace":true,"accessibility_snapshot":true,"performance_budget":true,"pass":true,"screenshot_only":false}`
}

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
