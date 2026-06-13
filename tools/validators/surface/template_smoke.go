package surface

import (
	"errors"
	"fmt"
	"strings"
)

const TemplateSmokeSchemaV1 = "tetra.surface.template-smoke.v1"

type SurfaceTemplateSmokeReport struct {
	Schema            string                             `json:"schema"`
	Model             string                             `json:"model"`
	ReleaseScope      string                             `json:"release_scope"`
	Producer          string                             `json:"producer"`
	Command           string                             `json:"command"`
	TemplateCount     int                                `json:"template_count"`
	Templates         []SurfaceTemplateSmokeTemplate     `json:"templates"`
	InspectorEvidence SurfaceTemplateSmokeInspector      `json:"inspector_evidence"`
	VisualEvidence    SurfaceTemplateSmokeVisual         `json:"visual_evidence"`
	PackageEvidence   []SurfaceTemplateSmokePackage      `json:"package_evidence"`
	NegativeGuards    SurfaceTemplateSmokeNegativeGuards `json:"negative_guards"`
	Pass              bool                               `json:"pass"`
}

type SurfaceTemplateSmokeTemplate struct {
	Kind             string                         `json:"kind"`
	ProjectDir       string                         `json:"project_dir"`
	Source           string                         `json:"source"`
	Capsule          string                         `json:"capsule"`
	TemplateMetadata string                         `json:"template_metadata"`
	Targets          []string                       `json:"targets"`
	Imports          []string                       `json:"imports"`
	RecipeCount      int                            `json:"recipe_count"`
	BlockMorphOnly   bool                           `json:"block_morph_only"`
	UsesAppShell     bool                           `json:"uses_app_shell"`
	WebCanvas        bool                           `json:"web_canvas"`
	Commands         []SurfaceTemplateSmokeCommand  `json:"commands"`
	SourceScan       SurfaceTemplateSmokeSourceScan `json:"source_scan"`
}

type SurfaceTemplateSmokeCommand struct {
	Kind     string `json:"kind"`
	Command  string `json:"command"`
	Pass     bool   `json:"pass"`
	ExitCode int    `json:"exit_code"`
}

type SurfaceTemplateSmokeSourceScan struct {
	ReactImport     bool `json:"react_import"`
	ElectronImport  bool `json:"electron_import"`
	DOMAppUITree    bool `json:"dom_app_ui_tree"`
	CSSRuntime      bool `json:"css_runtime"`
	CoreWidgets     bool `json:"core_widgets"`
	PlatformWidgets bool `json:"platform_widgets"`
	UserJSAppLogic  bool `json:"user_js_app_logic"`
	Pass            bool `json:"pass"`
}

type SurfaceTemplateSmokeInspector struct {
	Path  string `json:"path"`
	Model string `json:"model"`
	Pass  bool   `json:"pass"`
}

type SurfaceTemplateSmokeVisual struct {
	Path   string `json:"path"`
	Schema string `json:"schema"`
	Pass   bool   `json:"pass"`
}

type SurfaceTemplateSmokePackage struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Pass   bool   `json:"pass"`
}

type SurfaceTemplateSmokeNegativeGuards struct {
	NoReactImport          bool `json:"no_react_import"`
	NoElectronImport       bool `json:"no_electron_import"`
	NoDOMAppUITree         bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime           bool `json:"no_css_runtime"`
	NoCoreWidgets          bool `json:"no_core_widgets"`
	NoPlatformWidgets      bool `json:"no_platform_widgets"`
	NoUserJSAppLogic       bool `json:"no_user_js_app_logic"`
	CookbookUsesBlockMorph bool `json:"cookbook_uses_block_morph"`
}

func ValidateTemplateSmokeReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TemplateSmokeSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TemplateSmokeSchemaV1)
	}

	var report SurfaceTemplateSmokeReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: TemplateSmokeSchemaV1},
		{field: "model", got: report.Model, want: "surface-template-smoke-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-template-smoke.sh"},
		{field: "command", got: report.Command, want: "tetra new surface-app"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	issues = append(issues, validateSurfaceTemplateSmokeTemplates(report.TemplateCount, report.Templates)...)
	issues = append(issues, validateSurfaceTemplateSmokeInspector(report.InspectorEvidence)...)
	issues = append(issues, validateSurfaceTemplateSmokeVisual(report.VisualEvidence)...)
	issues = append(issues, validateSurfaceTemplateSmokePackages(report.PackageEvidence)...)
	issues = append(issues, validateSurfaceTemplateSmokeNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceTemplateSmokeTemplates(templateCount int, templates []SurfaceTemplateSmokeTemplate) []string {
	required := []string{"command-palette", "settings", "dashboard", "editor-shell", "multi-window-notes", "web-canvas"}
	if templateCount != len(templates) {
		return []string{fmt.Sprintf("template_count = %d, want len(templates) %d", templateCount, len(templates))}
	}
	var issues []string
	if templateCount != len(required) {
		issues = append(issues, fmt.Sprintf("template_count = %d, want %d", templateCount, len(required)))
	}
	seen := map[string]bool{}
	for _, tmpl := range templates {
		kind := strings.TrimSpace(tmpl.Kind)
		if kind == "" {
			issues = append(issues, "templates kind is required")
			continue
		}
		if seen[kind] {
			issues = append(issues, fmt.Sprintf("duplicate template kind %s", kind))
		}
		seen[kind] = true
		if !safeRelativeReportPath(tmpl.ProjectDir) {
			issues = append(issues, fmt.Sprintf("%s project_dir is unsafe or empty", kind))
		}
		if !safeRelativeSourcePath(tmpl.Source) {
			issues = append(issues, fmt.Sprintf("%s source is unsafe or not a Tetra source", kind))
		}
		if !safeRelativeReportPath(tmpl.Capsule) || !strings.HasSuffix(tmpl.Capsule, "Capsule.t4") {
			issues = append(issues, fmt.Sprintf("%s capsule path must be safe Capsule.t4", kind))
		}
		if !safeRelativeReportPath(tmpl.TemplateMetadata) || !strings.HasSuffix(tmpl.TemplateMetadata, "surface-template.json") {
			issues = append(issues, fmt.Sprintf("%s template_metadata must be safe surface-template.json", kind))
		}
		issues = append(issues, validateSurfaceTemplateTargets(kind, tmpl.Targets)...)
		issues = append(issues, validateSurfaceTemplateImports(kind, tmpl.Imports)...)
		if tmpl.RecipeCount < 4 {
			issues = append(issues, fmt.Sprintf("%s recipe_count must be at least 4", kind))
		}
		if !tmpl.BlockMorphOnly {
			issues = append(issues, fmt.Sprintf("%s block_morph_only must be true", kind))
		}
		if kind == "multi-window-notes" && !tmpl.UsesAppShell {
			issues = append(issues, "multi-window-notes uses_app_shell must be true")
		}
		if kind == "web-canvas" && !tmpl.WebCanvas {
			issues = append(issues, "web-canvas web_canvas must be true")
		}
		issues = append(issues, validateSurfaceTemplateCommands(kind, tmpl.Commands)...)
		issues = append(issues, validateSurfaceTemplateSourceScan(kind, tmpl.SourceScan)...)
	}
	for _, kind := range required {
		if !seen[kind] {
			issues = append(issues, fmt.Sprintf("templates missing %s", kind))
		}
	}
	return issues
}

func validateSurfaceTemplateTargets(kind string, targets []string) []string {
	var issues []string
	if !templateSmokeContainsString(targets, "linux-x64") {
		issues = append(issues, fmt.Sprintf("%s targets missing linux-x64", kind))
	}
	if !templateSmokeContainsString(targets, "wasm32-web") {
		issues = append(issues, fmt.Sprintf("%s targets missing wasm32-web", kind))
	}
	return issues
}

func validateSurfaceTemplateImports(kind string, imports []string) []string {
	var issues []string
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph"} {
		if !templateSmokeContainsString(imports, required) {
			issues = append(issues, fmt.Sprintf("%s imports missing %s", kind, required))
		}
	}
	if kind == "multi-window-notes" && !templateSmokeContainsString(imports, "lib.core.surface_app_shell") {
		issues = append(issues, "multi-window-notes imports missing lib.core.surface_app_shell")
	}
	for _, imported := range imports {
		lower := strings.ToLower(imported)
		for _, forbidden := range []string{"react", "electron", "dom", "css", "javascript", "lib.core.widgets", "lib.core.component", "platform_widget", "native_widget"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("%s imports forbidden runtime or core widget primitive %q", kind, imported))
			}
		}
	}
	return issues
}

func validateSurfaceTemplateCommands(kind string, commands []SurfaceTemplateSmokeCommand) []string {
	required := []string{"generate", "check", "build", "run", "inspect", "visual", "package"}
	var issues []string
	seen := map[string]SurfaceTemplateSmokeCommand{}
	for _, command := range commands {
		if strings.TrimSpace(command.Kind) == "" {
			issues = append(issues, fmt.Sprintf("%s command kind is required", kind))
			continue
		}
		seen[command.Kind] = command
		if strings.TrimSpace(command.Command) == "" {
			issues = append(issues, fmt.Sprintf("%s %s command is required", kind, command.Kind))
		}
		if !command.Pass {
			issues = append(issues, fmt.Sprintf("%s %s command must pass", kind, command.Kind))
		}
		if command.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("%s %s exit_code = %d, want 0", kind, command.Kind, command.ExitCode))
		}
	}
	for _, commandKind := range required {
		command, ok := seen[commandKind]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s commands missing %s", kind, commandKind))
			continue
		}
		switch commandKind {
		case "generate":
			if !strings.Contains(command.Command, "tetra new surface-app") || !strings.Contains(command.Command, "--template "+kind) {
				issues = append(issues, fmt.Sprintf("%s generate command must run tetra new surface-app --template %s", kind, kind))
			}
		case "check":
			if !strings.Contains(command.Command, "tetra check") {
				issues = append(issues, fmt.Sprintf("%s check command must run tetra check", kind))
			}
		case "build":
			if !strings.Contains(command.Command, "tetra build") || !strings.Contains(command.Command, "linux-x64") {
				issues = append(issues, fmt.Sprintf("%s build command must run tetra build --target linux-x64", kind))
			}
		case "run":
			if !strings.Contains(command.Command, "tetra run") || !strings.Contains(command.Command, "linux-x64") {
				issues = append(issues, fmt.Sprintf("%s run command must run tetra run --target linux-x64", kind))
			}
		case "inspect":
			if !strings.Contains(command.Command, "surface-inspector") {
				issues = append(issues, fmt.Sprintf("%s inspect command must run surface-inspector", kind))
			}
		case "visual":
			if !strings.Contains(command.Command, "surface-visual-diff") {
				issues = append(issues, fmt.Sprintf("%s visual command must run surface-visual-diff", kind))
			}
		case "package":
			if !strings.Contains(command.Command, "tar") {
				issues = append(issues, fmt.Sprintf("%s package command must create tar package evidence", kind))
			}
		}
	}
	return issues
}

func validateSurfaceTemplateSourceScan(kind string, scan SurfaceTemplateSmokeSourceScan) []string {
	var issues []string
	for _, check := range []struct {
		name string
		bad  bool
	}{
		{name: "react_import", bad: scan.ReactImport},
		{name: "electron_import", bad: scan.ElectronImport},
		{name: "dom_app_ui_tree", bad: scan.DOMAppUITree},
		{name: "css_runtime", bad: scan.CSSRuntime},
		{name: "core_widgets", bad: scan.CoreWidgets},
		{name: "platform_widgets", bad: scan.PlatformWidgets},
		{name: "user_js_app_logic", bad: scan.UserJSAppLogic},
	} {
		if check.bad {
			issues = append(issues, fmt.Sprintf("%s source_scan %s must be false", kind, check.name))
		}
	}
	if !scan.Pass {
		issues = append(issues, fmt.Sprintf("%s source_scan pass must be true", kind))
	}
	return issues
}

func validateSurfaceTemplateSmokeInspector(evidence SurfaceTemplateSmokeInspector) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "inspector_evidence.path is unsafe or empty")
	}
	if evidence.Model != "surface-inspector-v1" {
		issues = append(issues, fmt.Sprintf("inspector_evidence.model is %q, want surface-inspector-v1", evidence.Model))
	}
	if !evidence.Pass {
		issues = append(issues, "inspector_evidence pass must be true")
	}
	return issues
}

func validateSurfaceTemplateSmokeVisual(evidence SurfaceTemplateSmokeVisual) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "visual_evidence.path is unsafe or empty")
	}
	if evidence.Schema != VisualRegressionSchemaV1 {
		issues = append(issues, fmt.Sprintf("visual_evidence.schema is %q, want %s", evidence.Schema, VisualRegressionSchemaV1))
	}
	if !evidence.Pass {
		issues = append(issues, "visual_evidence pass must be true")
	}
	return issues
}

func validateSurfaceTemplateSmokePackages(packages []SurfaceTemplateSmokePackage) []string {
	if len(packages) == 0 {
		return []string{"package_evidence is required"}
	}
	var issues []string
	for _, pkg := range packages {
		if !safeRelativeReportPath(pkg.Path) {
			issues = append(issues, "package_evidence path is unsafe or empty")
		}
		if pkg.Kind != "tar.gz" {
			issues = append(issues, fmt.Sprintf("package_evidence kind is %q, want tar.gz", pkg.Kind))
		}
		if !strings.HasPrefix(pkg.SHA256, "sha256:") || len(pkg.SHA256) != len("sha256:")+64 {
			issues = append(issues, "package_evidence sha256 must be sha256 digest")
		}
		if !pkg.Pass {
			issues = append(issues, "package_evidence pass must be true")
		}
	}
	return issues
}

func validateSurfaceTemplateSmokeNegativeGuards(guards SurfaceTemplateSmokeNegativeGuards) []string {
	if guards.NoReactImport &&
		guards.NoElectronImport &&
		guards.NoDOMAppUITree &&
		guards.NoCSSRuntime &&
		guards.NoCoreWidgets &&
		guards.NoPlatformWidgets &&
		guards.NoUserJSAppLogic &&
		guards.CookbookUsesBlockMorph {
		return nil
	}
	return []string{"negative_guards must reject React, Electron, DOM app UI tree, CSS runtime, core widgets, platform widgets, user JS app logic, and require Block/Morph cookbook recipes"}
}

func templateSmokeContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
