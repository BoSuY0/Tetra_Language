package surface

import (
	"strings"
	"testing"
)

func TestValidateSurfaceDevWorkflowReportAcceptsFastRebuild(t *testing.T) {
	raw := []byte(validSurfaceDevWorkflowReportJSON())
	if err := ValidateDevWorkflowReport(raw); err != nil {
		t.Fatalf("ValidateDevWorkflowReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceDevWorkflowReportAcceptsMorphToPixelsChain(t *testing.T) {
	raw := []byte(validSurfaceDevWorkflowReportWithMorphToPixelsJSON())
	if err := ValidateDevWorkflowReport(raw); err != nil {
		t.Fatalf("ValidateDevWorkflowReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceDevWorkflowReportRejectsBrokenMorphToPixelsChain(t *testing.T) {
	raw := strings.Replace(validSurfaceDevWorkflowReportWithMorphToPixelsJSON(), `"golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003d"`, `"golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003c"`, 1)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected broken Morph-to-pixels chain to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "self-golden") {
		t.Fatalf("error = %v, want self-golden diagnostic", err)
	}
}

func TestValidateSurfaceDevWorkflowReportRejectsHotReloadClaimWithFullRestart(t *testing.T) {
	raw := strings.Replace(validSurfaceDevWorkflowReportJSON(), `"hot_reload_claim":false`, `"hot_reload_claim":true`, 1)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected hot reload claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "hot reload") {
		t.Fatalf("error = %v, want hot reload diagnostic", err)
	}
}

func TestValidateSurfaceDevWorkflowReportRejectsMissingChangedTokenRecipeSource(t *testing.T) {
	raw := strings.Replace(validSurfaceDevWorkflowReportJSON(), `,{"name":"recipe rebuild","kind":"recipe-change","changed_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/design/recipes.tetra","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/recipe/app","duration_ms":7,"compiled_modules":["design.recipes"],"cache_hits":["app.main","design.tokens"],"pass":true}`, ``, 1)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing recipe-change step to fail")
	}
	if !strings.Contains(err.Error(), "recipe-change") {
		t.Fatalf("error = %v, want recipe-change diagnostic", err)
	}
}

func validSurfaceDevWorkflowReportJSON() string {
	return `{"schema":"tetra.surface.dev-workflow.v1","model":"surface-dev-workflow-v1","release_scope":"surface-v1-linux-web","command":"tetra surface dev","source":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/app/main.tetra","target":"linux-x64","mode":"fast-rebuild","reload_semantics":"fast-rebuild","process_restart_required":true,"hot_reload_claim":false,"watch":false,"supported_targets":["headless","linux-x64","wasm32-web"],"steps":[{"name":"initial build","kind":"initial","changed_path":"","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/initial/app","duration_ms":25,"compiled_modules":["app.main","design.recipes","design.tokens"],"cache_hits":[],"pass":true},{"name":"warm rebuild","kind":"warm-cache","changed_path":"","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/warm/app","duration_ms":3,"compiled_modules":[],"cache_hits":["app.main","design.recipes","design.tokens"],"pass":true},{"name":"token rebuild","kind":"token-change","changed_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/design/tokens.tetra","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/token/app","duration_ms":8,"compiled_modules":["design.tokens"],"cache_hits":["app.main","design.recipes"],"pass":true},{"name":"recipe rebuild","kind":"recipe-change","changed_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/design/recipes.tetra","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/recipe/app","duration_ms":7,"compiled_modules":["design.recipes"],"cache_hits":["app.main","design.tokens"],"pass":true},{"name":"source rebuild","kind":"source-change","changed_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/app/main.tetra","output_path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-artifacts/source/app","duration_ms":9,"compiled_modules":["app.main"],"cache_hits":["design.recipes","design.tokens"],"pass":true}],"source_diagnostics":[{"kind":"token","path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/design/tokens.tetra","line":1,"column":1,"code":"SURFACE_DEV_TOKEN_PATH","message":"token file participates in Surface fast rebuild","severity":"info","pass":true},{"kind":"recipe","path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/design/recipes.tetra","line":1,"column":1,"code":"SURFACE_DEV_RECIPE_PATH","message":"recipe file participates in Surface fast rebuild","severity":"info","pass":true},{"kind":"source","path":"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/app/main.tetra","line":1,"column":1,"code":"SURFACE_DEV_SOURCE_PATH","message":"source file participates in Surface fast rebuild","severity":"info","pass":true}],"negative_guards":{"no_hot_reload_claim":true,"full_restart_documented_as_fast_rebuild":true,"no_electron_dev_server":true,"no_react_fast_refresh":true,"no_dom_hot_reload":true},"pass":true}` + "\n"
}

func validSurfaceDevWorkflowReportWithMorphToPixelsJSON() string {
	raw := strings.TrimSuffix(validSurfaceDevWorkflowReportJSON(), "\n")
	raw = strings.Replace(raw, `,"negative_guards":`, `,"morph_to_pixels":`+validMorphToPixelsChainJSON("reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/app/main.tetra")+`,"negative_guards":`, 1)
	return raw + "\n"
}

func validMorphToPixelsChainJSON(source string) string {
	return `{"chain_id":"sha256:0000000000000000000000000000000000000000000000000000000000000900","report_path":"reports/surface/morph-rendered-beauty.json","schema":"tetra.surface.morph-rendered-beauty.v1","status":"pass","surface_scope":"surface-morph-rendered-beauty-linux-web","source":"` + source + `","source_sha256":"sha256:0000000000000000000000000000000000000000000000000000000000000001","target":"headless","scenario_name":"headless-morph:` + source + `","git_head":"95bfd4a887bab5032437cb22494d034e82ae6d35","git_commit":"95bfd4a887bab5032437cb22494d034e82ae6d35","git_dirty":true,"token_graph_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000003","token_count":6,"token_categories":["color","space","radius","typography","motion","assets"],"recipe_count":3,"recipe_expansion_count":4,"recipe_names":["studio_shell","hero_panel","toolbar"],"block_scene_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000005","block_scene_node_count":12,"render_command_stream_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000007","render_command_count":10,"renderer":"software-rgba-headless","frame_artifact":"reports/surface/frame.rgba","frame_artifact_sha256":"sha256:000000000000000000000000000000000000000000000000000000000000003c","frame_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003c","golden_artifact":"reports/surface/golden.rgba","golden_artifact_sha256":"sha256:000000000000000000000000000000000000000000000000000000000000003d","golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003d","diff_pixels":1,"diff_ratio_milli":0,"max_channel_delta":1,"product_claim":false,"final_signoff":false,"pass":true}`
}
