package surface

import (
	"errors"
	"fmt"
	"strings"
)

const ReferenceAppsSchemaV1 = "tetra.surface.reference-app-suite.v1"

type SurfaceReferenceAppsReport struct {
	Schema          string                             `json:"schema"`
	Model           string                             `json:"model"`
	ReleaseScope    string                             `json:"release_scope"`
	Producer        string                             `json:"producer"`
	AppCount        int                                `json:"app_count"`
	RequiredTargets []string                           `json:"required_targets"`
	Apps            []SurfaceReferenceAppReport        `json:"apps"`
	VisualEvidence  SurfaceReferenceAppsVisualEvidence `json:"visual_evidence"`
	NegativeGuards  SurfaceReferenceAppsNegativeGuards `json:"negative_guards"`
	Pass            bool                               `json:"pass"`
}

type SurfaceReferenceAppReport struct {
	Shape                 string                            `json:"shape"`
	Source                string                            `json:"source"`
	Module                string                            `json:"module"`
	Imports               []string                          `json:"imports"`
	Recipes               []string                          `json:"recipes"`
	StableMorphRecipes    bool                              `json:"stable_morph_recipes"`
	ResolvesToBlock       bool                              `json:"resolves_to_block"`
	Compiles              bool                              `json:"compiles"`
	Runs                  bool                              `json:"runs"`
	ExitCode              int                               `json:"exit_code"`
	TokenThemeConformance bool                              `json:"token_theme_conformance"`
	LayoutReport          bool                              `json:"layout_report"`
	InteractionTrace      bool                              `json:"interaction_trace"`
	AccessibilitySnapshot bool                              `json:"accessibility_snapshot"`
	PerformanceBudget     bool                              `json:"performance_budget"`
	ArtifactHashes        bool                              `json:"artifact_hashes"`
	CompatibilityWidgets  bool                              `json:"compatibility_widgets"`
	Targets               []SurfaceReferenceAppTargetReport `json:"targets"`
}

type SurfaceReferenceAppTargetReport struct {
	Target                string `json:"target"`
	RuntimeReport         string `json:"runtime_report"`
	FrameChecksum         string `json:"frame_checksum"`
	VisualDiff            bool   `json:"visual_diff"`
	InteractionTrace      bool   `json:"interaction_trace"`
	AccessibilitySnapshot bool   `json:"accessibility_snapshot"`
	PerformanceBudget     bool   `json:"performance_budget"`
	Pass                  bool   `json:"pass"`
	ScreenshotOnly        bool   `json:"screenshot_only"`
}

type SurfaceReferenceAppsVisualEvidence struct {
	Path     string `json:"path"`
	Schema   string `json:"schema"`
	AppCount int    `json:"app_count"`
	Pass     bool   `json:"pass"`
}

type SurfaceReferenceAppsNegativeGuards struct {
	ScreenshotOnlyRejected            bool `json:"screenshot_only_rejected"`
	MissingInteractionRejected        bool `json:"missing_interaction_rejected"`
	MissingAccessibilityRejected      bool `json:"missing_accessibility_rejected"`
	MissingPerformanceRejected        bool `json:"missing_performance_rejected"`
	CoreWidgetUsageRejected           bool `json:"core_widget_usage_rejected"`
	MigrationWidgetsCompatibilityOnly bool `json:"migration_widgets_compatibility_only"`
	NoReactRuntime                    bool `json:"no_react_runtime"`
	NoElectronRuntime                 bool `json:"no_electron_runtime"`
	NoDOMAppUITree                    bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime                      bool `json:"no_css_runtime"`
	NoUserJSAppLogic                  bool `json:"no_user_js_app_logic"`
}

func ValidateReferenceAppsReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != ReferenceAppsSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, ReferenceAppsSchemaV1)
	}
	var report SurfaceReferenceAppsReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceReferenceAppsReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceReferenceAppsReport(report SurfaceReferenceAppsReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: ReferenceAppsSchemaV1},
		{field: "model", got: report.Model, want: "surface-reference-app-suite-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-reference-apps-smoke.sh"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if report.AppCount != len(report.Apps) {
		issues = append(issues, fmt.Sprintf("app_count = %d, want len(apps) %d", report.AppCount, len(report.Apps)))
	}
	issues = append(issues, validateSurfaceReferenceTargets(report.RequiredTargets)...)
	issues = append(issues, validateSurfaceReferenceApps(report.Apps, report.RequiredTargets)...)
	issues = append(issues, validateSurfaceReferenceVisualEvidence(report.VisualEvidence, len(requiredSurfaceReferenceApps()))...)
	issues = append(issues, validateSurfaceReferenceNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceReferenceTargets(targets []string) []string {
	var issues []string
	for _, target := range requiredSurfaceReferenceTargets() {
		if !templateSmokeContainsString(targets, target) {
			issues = append(issues, fmt.Sprintf("required_targets missing %s", target))
		}
	}
	return issues
}

func validateSurfaceReferenceApps(apps []SurfaceReferenceAppReport, requiredTargets []string) []string {
	required := requiredSurfaceReferenceApps()
	seen := map[string]SurfaceReferenceAppReport{}
	var issues []string
	for _, app := range apps {
		shape := strings.TrimSpace(app.Shape)
		if shape == "" {
			issues = append(issues, "apps shape is required")
			continue
		}
		if _, ok := seen[shape]; ok {
			issues = append(issues, fmt.Sprintf("duplicate reference app shape %s", shape))
		}
		seen[shape] = app
		if wantSource, ok := required[shape]; ok && normalizeEvidencePath(app.Source) != wantSource {
			issues = append(issues, fmt.Sprintf("%s source is %q, want %s", shape, app.Source, wantSource))
		}
		issues = append(issues, validateSurfaceReferenceApp(app, requiredTargets)...)
	}
	for shape := range required {
		if _, ok := seen[shape]; !ok {
			issues = append(issues, fmt.Sprintf("reference suite missing %s", shape))
		}
	}
	if len(apps) != len(required) {
		issues = append(issues, fmt.Sprintf("apps length = %d, want %d", len(apps), len(required)))
	}
	return issues
}

func validateSurfaceReferenceApp(app SurfaceReferenceAppReport, requiredTargets []string) []string {
	shape := strings.TrimSpace(app.Shape)
	prefix := "reference app " + shape
	var issues []string
	if !safeRelativeSourcePath(app.Source) {
		issues = append(issues, prefix+" source must be a safe Tetra source path")
	}
	if strings.TrimSpace(app.Module) == "" {
		issues = append(issues, prefix+" module is required")
	}
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph"} {
		if !templateSmokeContainsString(app.Imports, required) {
			issues = append(issues, fmt.Sprintf("%s imports missing %s", prefix, required))
		}
	}
	if app.CompatibilityWidgets && shape != "migration" {
		issues = append(issues, fmt.Sprintf("%s compatibility_widgets may be true only for migration", prefix))
	}
	if shape == "migration" && !app.CompatibilityWidgets {
		issues = append(issues, "reference app migration requires compatibility_widgets evidence")
	}
	for _, imported := range app.Imports {
		lower := strings.ToLower(imported)
		if strings.Contains(lower, "lib.core.widgets") && shape != "migration" {
			issues = append(issues, fmt.Sprintf("%s imports widgets outside migration compatibility example", prefix))
		}
		for _, forbidden := range []string{"react", "electron", "dom", "css", "javascript", "platform_widget", "native_widget"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("%s imports forbidden runtime %q", prefix, imported))
			}
		}
	}
	if len(app.Recipes) < 4 || !app.StableMorphRecipes || !app.ResolvesToBlock {
		issues = append(issues, prefix+" requires at least four stable Morph recipes that resolve to Block")
	}
	if !app.Compiles || !app.Runs || app.ExitCode != 0 {
		issues = append(issues, fmt.Sprintf("%s compile/run evidence must pass with exit 0", prefix))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "token_theme_conformance", ok: app.TokenThemeConformance},
		{name: "layout_report", ok: app.LayoutReport},
		{name: "interaction_trace", ok: app.InteractionTrace},
		{name: "accessibility_snapshot", ok: app.AccessibilitySnapshot},
		{name: "performance_budget", ok: app.PerformanceBudget},
		{name: "artifact_hashes", ok: app.ArtifactHashes},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("%s %s must be true", prefix, check.name))
		}
	}
	issues = append(issues, validateSurfaceReferenceAppTargets(prefix, app.Targets, requiredTargets)...)
	return issues
}

func validateSurfaceReferenceAppTargets(prefix string, targets []SurfaceReferenceAppTargetReport, requiredTargets []string) []string {
	seen := map[string]SurfaceReferenceAppTargetReport{}
	var issues []string
	for _, target := range targets {
		name := strings.TrimSpace(target.Target)
		if name == "" {
			issues = append(issues, prefix+" target name is required")
			continue
		}
		seen[name] = target
		if !safeRelativeReportPath(target.RuntimeReport) {
			issues = append(issues, fmt.Sprintf("%s %s runtime_report is unsafe or empty", prefix, name))
		}
		if !validChecksumLike(target.FrameChecksum) {
			issues = append(issues, fmt.Sprintf("%s %s frame_checksum must be sha256 evidence", prefix, name))
		}
		if target.ScreenshotOnly {
			issues = append(issues, fmt.Sprintf("%s %s screenshot-only evidence is not sufficient", prefix, name))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "visual_diff", ok: target.VisualDiff},
			{name: "interaction_trace", ok: target.InteractionTrace},
			{name: "accessibility_snapshot", ok: target.AccessibilitySnapshot},
			{name: "performance_budget", ok: target.PerformanceBudget},
			{name: "pass", ok: target.Pass},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s %s %s must be true", prefix, name, check.name))
			}
		}
	}
	for _, target := range requiredTargets {
		if !templateSmokeContainsString(requiredSurfaceReferenceTargets(), target) {
			continue
		}
		if _, ok := seen[target]; !ok {
			issues = append(issues, fmt.Sprintf("%s missing required target %s", prefix, target))
		}
	}
	return issues
}

func validateSurfaceReferenceVisualEvidence(evidence SurfaceReferenceAppsVisualEvidence, appCount int) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "visual_evidence.path is unsafe or empty")
	}
	if evidence.Schema != VisualRegressionSchemaV1 {
		issues = append(issues, fmt.Sprintf("visual_evidence.schema is %q, want %s", evidence.Schema, VisualRegressionSchemaV1))
	}
	if evidence.AppCount != appCount {
		issues = append(issues, fmt.Sprintf("visual_evidence.app_count = %d, want %d", evidence.AppCount, appCount))
	}
	if !evidence.Pass {
		issues = append(issues, "visual_evidence pass must be true")
	}
	return issues
}

func validateSurfaceReferenceNegativeGuards(guards SurfaceReferenceAppsNegativeGuards) []string {
	var missing []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "screenshot_only_rejected", ok: guards.ScreenshotOnlyRejected},
		{name: "missing_interaction_rejected", ok: guards.MissingInteractionRejected},
		{name: "missing_accessibility_rejected", ok: guards.MissingAccessibilityRejected},
		{name: "missing_performance_rejected", ok: guards.MissingPerformanceRejected},
		{name: "core_widget_usage_rejected", ok: guards.CoreWidgetUsageRejected},
		{name: "migration_widgets_compatibility_only", ok: guards.MigrationWidgetsCompatibilityOnly},
		{name: "no_react_runtime", ok: guards.NoReactRuntime},
		{name: "no_electron_runtime", ok: guards.NoElectronRuntime},
		{name: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
		{name: "no_css_runtime", ok: guards.NoCSSRuntime},
		{name: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
	} {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func requiredSurfaceReferenceApps() map[string]string {
	return map[string]string{
		"command-palette":     "examples/surface_reference_command_palette.tetra",
		"settings":            "examples/surface_reference_settings.tetra",
		"dashboard":           "examples/surface_reference_dashboard.tetra",
		"editor-shell":        "examples/surface_reference_editor_shell.tetra",
		"file-manager":        "examples/surface_reference_file_manager.tetra",
		"dialog-notification": "examples/surface_reference_dialog_notification.tetra",
		"localized-form":      "examples/surface_reference_localized_form.tetra",
		"accessibility-form":  "examples/surface_reference_accessibility_form.tetra",
		"multi-window-notes":  "examples/surface_reference_multi_window_notes.tetra",
		"migration":           "examples/surface_reference_migration.tetra",
	}
}

func requiredSurfaceReferenceTargets() []string {
	return []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}
}
