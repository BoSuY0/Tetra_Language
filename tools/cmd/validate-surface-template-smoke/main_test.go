package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesSurfaceTemplateSmokeReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-template-smoke.json")
	if err := os.WriteFile(reportPath, []byte(validSurfaceTemplateSmokeReportJSON()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := run([]string{"--report", reportPath}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRunRejectsSurfaceTemplateForbiddenRuntime(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-template-smoke.json")
	raw := strings.Replace(validSurfaceTemplateSmokeReportJSON(), `"no_react_import":true`, `"no_react_import":false`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := run([]string{"--report", reportPath})
	if err == nil {
		t.Fatalf("expected forbidden runtime guard to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "react") {
		t.Fatalf("error = %v, want React diagnostic", err)
	}
}

func validSurfaceTemplateSmokeReportJSON() string {
	kinds := []string{"command-palette", "settings", "dashboard", "editor-shell", "multi-window-notes", "web-canvas"}
	var templates []string
	for _, kind := range kinds {
		imports := `"imports":["lib.core.surface","lib.core.block","lib.core.morph"]`
		usesAppShell := "false"
		webCanvas := "false"
		if kind == "multi-window-notes" {
			imports = `"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"]`
			usesAppShell = "true"
		}
		if kind == "web-canvas" {
			webCanvas = "true"
		}
		templates = append(templates, `{"kind":"`+kind+`","project_dir":"templates/`+kind+`","source":"templates/`+kind+`/src/main.tetra","capsule":"templates/`+kind+`/Capsule.t4","template_metadata":"templates/`+kind+`/surface-template.json","targets":["linux-x64","wasm32-web"],`+imports+`,"recipe_count":4,"block_morph_only":true,"uses_app_shell":`+usesAppShell+`,"web_canvas":`+webCanvas+`,"commands":[{"kind":"generate","command":"tetra new surface-app --template `+kind+`","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-`+kind+`.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}}`)
	}
	return `{"schema":"tetra.surface.template-smoke.v1","model":"surface-template-smoke-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-template-smoke.sh","command":"tetra new surface-app","template_count":6,"templates":[` + strings.Join(templates, ",") + `],"inspector_evidence":{"path":"surface-inspector.json","model":"surface-inspector-v1","pass":true},"visual_evidence":{"path":"template-visual/surface-visual-regression.json","schema":"tetra.surface.visual-regression.v1","pass":true},"package_evidence":[{"path":"packages/surface-template-command-palette.tar.gz","kind":"tar.gz","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","pass":true}],"negative_guards":{"no_react_import":true,"no_electron_import":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_core_widgets":true,"no_platform_widgets":true,"no_user_js_app_logic":true,"cookbook_uses_block_morph":true},"pass":true}` + "\n"
}
