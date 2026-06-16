package surface

import (
	"strings"
	"testing"
)

func TestValidateTemplateSmokeReportAcceptsP21TemplateEvidence(t *testing.T) {
	raw := validTemplateSmokeReportJSON()
	if err := ValidateTemplateSmokeReport(raw); err != nil {
		t.Fatalf("ValidateTemplateSmokeReport failed: %v\n%s", err, raw)
	}
}

func TestValidateTemplateSmokeReportRejectsMissingTemplateKind(t *testing.T) {
	raw := strings.Replace(string(validTemplateSmokeReportJSON()), `"kind":"web-canvas"`, `"kind":"browser-widget"`, 1)
	err := ValidateTemplateSmokeReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing web-canvas template to fail")
	}
	if !strings.Contains(err.Error(), "web-canvas") {
		t.Fatalf("error = %v, want web-canvas diagnostic", err)
	}
}

func TestValidateTemplateSmokeReportRejectsForbiddenRuntimeImports(t *testing.T) {
	raw := strings.Replace(string(validTemplateSmokeReportJSON()), `"imports":["lib.core.surface","lib.core.block","lib.core.morph"]`, `"imports":["lib.core.surface","React","Electron"]`, 1)
	err := ValidateTemplateSmokeReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected forbidden runtime imports to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "react") || !strings.Contains(strings.ToLower(err.Error()), "electron") {
		t.Fatalf("error = %v, want React/Electron diagnostic", err)
	}
}

func TestValidateTemplateSmokeReportRejectsMissingMorphToPixels(t *testing.T) {
	raw := strings.Replace(string(validTemplateSmokeReportJSON()), `
  "morph_to_pixels": `+validMorphToPixelsChainJSON("templates/studio-shell/src/main.tetra")+`,
`, "", 1)
	err := ValidateTemplateSmokeReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing Morph-to-pixels evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "morph_to_pixels") {
		t.Fatalf("error = %v, want morph_to_pixels diagnostic", err)
	}
}

func validTemplateSmokeReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.template-smoke.v1",
  "model": "surface-template-smoke-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-template-smoke.sh",
  "command": "tetra new surface-app",
  "template_count": 7,
  "templates": [
    {"kind":"command-palette","project_dir":"templates/command-palette","source":"templates/command-palette/src/main.tetra","capsule":"templates/command-palette/Capsule.t4","template_metadata":"templates/command-palette/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":false,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template command-palette","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-command-palette.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"settings","project_dir":"templates/settings","source":"templates/settings/src/main.tetra","capsule":"templates/settings/Capsule.t4","template_metadata":"templates/settings/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":false,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template settings","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-settings.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"dashboard","project_dir":"templates/dashboard","source":"templates/dashboard/src/main.tetra","capsule":"templates/dashboard/Capsule.t4","template_metadata":"templates/dashboard/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":false,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template dashboard","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-dashboard.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"editor-shell","project_dir":"templates/editor-shell","source":"templates/editor-shell/src/main.tetra","capsule":"templates/editor-shell/Capsule.t4","template_metadata":"templates/editor-shell/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":false,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template editor-shell","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-editor-shell.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"studio-shell","project_dir":"templates/studio-shell","source":"templates/studio-shell/src/main.tetra","capsule":"templates/studio-shell/Capsule.t4","template_metadata":"templates/studio-shell/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":true,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template studio-shell","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-studio-shell.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"multi-window-notes","project_dir":"templates/multi-window-notes","source":"templates/multi-window-notes/src/main.tetra","capsule":"templates/multi-window-notes/Capsule.t4","template_metadata":"templates/multi-window-notes/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":true,"web_canvas":false,"commands":[{"kind":"generate","command":"tetra new surface-app --template multi-window-notes","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-multi-window-notes.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}},
    {"kind":"web-canvas","project_dir":"templates/web-canvas","source":"templates/web-canvas/src/main.tetra","capsule":"templates/web-canvas/Capsule.t4","template_metadata":"templates/web-canvas/surface-template.json","targets":["linux-x64","wasm32-web"],"imports":["lib.core.surface","lib.core.block","lib.core.morph"],"recipe_count":4,"block_morph_only":true,"uses_app_shell":false,"web_canvas":true,"commands":[{"kind":"generate","command":"tetra new surface-app --template web-canvas","pass":true,"exit_code":0},{"kind":"check","command":"tetra check","pass":true,"exit_code":0},{"kind":"build","command":"tetra build --target linux-x64","pass":true,"exit_code":0},{"kind":"run","command":"tetra run --target linux-x64","pass":true,"exit_code":0},{"kind":"inspect","command":"surface-inspector","pass":true,"exit_code":0},{"kind":"visual","command":"surface-visual-diff","pass":true,"exit_code":0},{"kind":"package","command":"tar surface-template-web-canvas.tar.gz","pass":true,"exit_code":0}],"source_scan":{"react_import":false,"electron_import":false,"dom_app_ui_tree":false,"css_runtime":false,"core_widgets":false,"platform_widgets":false,"user_js_app_logic":false,"pass":true}}
  ],
  "inspector_evidence": {"path":"surface-inspector.json","model":"surface-inspector-v1","pass":true},
  "visual_evidence": {"path":"template-visual/surface-visual-regression.json","schema":"tetra.surface.visual-regression.v1","pass":true},
  "morph_to_pixels": ` + validMorphToPixelsChainJSON("templates/studio-shell/src/main.tetra") + `,
  "package_evidence": [{"path":"packages/surface-template-command-palette.tar.gz","kind":"tar.gz","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","pass":true}],
  "negative_guards": {"no_react_import":true,"no_electron_import":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_core_widgets":true,"no_platform_widgets":true,"no_user_js_app_logic":true,"cookbook_uses_block_morph":true},
  "pass": true
}` + "\n")
}
