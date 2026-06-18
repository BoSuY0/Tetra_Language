package surface

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ---- block_scene_snapshot_test.go ----

func TestValidateBlockSceneSnapshotAcceptsRichVisualSpecs(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsCompactPropsOnlyEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(
		t,
		func(snapshot map[string]any) {
			snapshot["compact_props_only"] = true
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected compact-only Block scene snapshot to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "compact") {
		t.Fatalf("error = %v, want compact diagnostic", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsNonBlockCorePrimitive(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(
		t,
		func(snapshot map[string]any) {
			snapshot["core_primitives"] = []any{"Block", "Button"}
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected non-Block core primitive to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "button") {
		t.Fatalf("error = %v, want Button diagnostic", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsMissingRichSpecCoverage(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(
		t,
		func(snapshot map[string]any) {
			coverage := snapshot["spec_coverage"].(map[string]any)
			coverage["motion"] = false
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing motion spec coverage to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "motion") {
		t.Fatalf("error = %v, want motion diagnostic", err)
	}
}

func validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessMorphSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode Morph surface report: %v", err)
	}
	snapshot := blockSceneSnapshotMapForTest(report["source"].(string))
	if mutate != nil {
		mutate(snapshot)
	}
	report["block_scene_snapshot"] = snapshot
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block scene snapshot report: %v", err)
	}
	return raw
}

func blockSceneSnapshotMapForTest(source string) map[string]any {
	return map[string]any{
		"schema":                 "tetra.surface.block-scene-snapshot.v1",
		"source":                 source,
		"surface_scope":          "surface-morph-rendered-beauty-linux-web",
		"producer":               "surface-runtime-smoke",
		"quality_level":          "rich-renderable-block-scene-v1",
		"core_primitives":        []any{"Block"},
		"compact_props_only":     false,
		"recipe_expansion_count": 3,
		"node_count":             2,
		"rich_spec_hash":         "sha256:" + strings.Repeat("b", 64),
		"block_scene_hash":       "sha256:" + strings.Repeat("c", 64),
		"spec_coverage": map[string]any{
			"layout":        true,
			"paint":         true,
			"text":          true,
			"image":         true,
			"input":         true,
			"event":         true,
			"state":         true,
			"motion":        true,
			"accessibility": true,
		},
		"nodes": []any{
			map[string]any{
				"block_id":  1,
				"parent_id": -1,
				"recipe":    "morph.surface",
				"name":      "CommandPaletteRoot",
				"layout": map[string]any{
					"mode": "column",
					"x":    0,
					"y":    0,
					"w":    320,
					"h":    200,
				},
				"paint": map[string]any{
					"layer_count": 2,
					"layers": []any{
						map[string]any{
							"kind":    "fill",
							"color":   "#10151dff",
							"radius":  12,
							"opacity": 255,
						},
						map[string]any{
							"kind":    "border",
							"color":   "#6eaef4ff",
							"width":   1,
							"radius":  12,
							"opacity": 255,
						},
					},
				},
				"text": map[string]any{
					"text_len": 0,
					"color":    "#f4f7fbff",
					"size":     14,
					"weight":   500,
				},
				"image": map[string]any{"asset_id": "none", "mode": "none", "opacity": 0},
				"input": map[string]any{
					"kind":      "none",
					"focusable": false,
					"editable":  false,
				},
				"event": map[string]any{"pointer_action": "none", "key_action": "none"},
				"state": map[string]any{"variant": "surface", "enabled": true},
				"motion": map[string]any{
					"duration_ms":         120,
					"easing":              "standard",
					"reduced_motion_safe": true,
				},
				"accessibility": map[string]any{
					"role":          "group",
					"label_len":     15,
					"reading_index": 1,
				},
			},
			map[string]any{
				"block_id":  2,
				"parent_id": 1,
				"recipe":    "morph.search_input",
				"name":      "SearchInput",
				"layout": map[string]any{
					"mode": "row",
					"x":    16,
					"y":    16,
					"w":    288,
					"h":    168,
				},
				"paint": map[string]any{
					"layer_count": 3,
					"layers": []any{
						map[string]any{
							"kind":    "fill",
							"color":   "#202733ff",
							"radius":  8,
							"opacity": 255,
						},
						map[string]any{
							"kind":     "shadow",
							"color":    "#00000040",
							"blur":     8,
							"offset_y": 2,
							"opacity":  64,
						},
						map[string]any{
							"kind":    "outline",
							"color":   "#6eaef4ff",
							"width":   1,
							"radius":  8,
							"opacity": 255,
						},
					},
				},
				"text": map[string]any{
					"text_len": 12,
					"hint_len": 10,
					"color":    "#f4f7fbff",
					"size":     14,
					"weight":   500,
				},
				"image": map[string]any{
					"asset_id": "search-icon",
					"mode":     "template",
					"tint":     "#b7c4d6ff",
					"opacity":  255,
				},
				"input": map[string]any{
					"kind":      "text",
					"focusable": true,
					"editable":  true,
				},
				"event": map[string]any{"pointer_action": "focus", "key_action": "edit"},
				"state": map[string]any{
					"variant": "focused",
					"enabled": true,
					"focused": true,
				},
				"motion": map[string]any{
					"duration_ms":         140,
					"easing":              "standard",
					"reduced_motion_safe": true,
				},
				"accessibility": map[string]any{
					"role":          "textbox",
					"label_len":     13,
					"focus_index":   1,
					"reading_index": 2,
					"actions":       []any{"focus", "edit"},
				},
			},
		},
	}
}

// ---- claims_test.go ----

func TestSurfaceClaimTierVocabulary(t *testing.T) {
	want := []ClaimTier{
		ClaimTierProdStableScoped,
		ClaimTierBetaTargetHost,
		ClaimTierExperimental,
		ClaimTierUnsupported,
		ClaimTierNonClaim,
	}

	got := SurfaceClaimTiers()
	if len(got) != len(want) {
		t.Fatalf("SurfaceClaimTiers len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SurfaceClaimTiers[%d] = %q, want %q", i, got[i], want[i])
		}
		if !ValidSurfaceClaimTier(string(got[i])) {
			t.Fatalf("ValidSurfaceClaimTier(%q) = false, want true", got[i])
		}
	}
	if ValidSurfaceClaimTier("PRODUCTION_EVERYWHERE") {
		t.Fatalf("ValidSurfaceClaimTier accepted unknown tier")
	}
}

// ---- crash_report_test.go ----

func TestValidateCrashReportAcceptsCompleteScopedEvidence(t *testing.T) {
	raw := validCrashReportJSON()
	if err := ValidateCrashReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateCrashReport failed: %v\n%s", err, raw)
	}
}

func TestValidateCrashReportRejectsRestartClaimWithoutEvidence(t *testing.T) {
	raw := strings.Replace(validCrashReportJSON(), `"after_run":true`, `"after_run":false`, 1)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing restart evidence to fail")
	}
	if !strings.Contains(err.Error(), "restart_recovery") {
		t.Fatalf("error = %v, want restart_recovery diagnostic", err)
	}
}

func TestValidateCrashReportRejectsUserDataLeak(t *testing.T) {
	raw := strings.Replace(
		validCrashReportJSON(),
		`"user_text_captured":false`,
		`"user_text_captured":true`,
		1,
	)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected user text leak to fail")
	}
	if !strings.Contains(err.Error(), "user_text_captured") {
		t.Fatalf("error = %v, want user_text_captured diagnostic", err)
	}
}

func TestValidateCrashReportRejectsMissingHostCrashScenario(t *testing.T) {
	raw := strings.Replace(validCrashReportJSON(), crashScenarioJSON("host_crash"), "", 1)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing host crash scenario to fail")
	}
	if !strings.Contains(err.Error(), "host_crash") {
		t.Fatalf("error = %v, want host_crash diagnostic", err)
	}
}

func validCrashReportJSON() string {
	return ("{\"schema\":\"tetra.surface.crash-report.v1\",\"model\":" +
		"\"surface-crash-report-v1\",\"release_scope\":" +
		"\"surface-v1-linux-web\",\"producer\":" +
		"\"scripts/release/surface/surface-crash-report-smoke.sh\"," +
		"\"source\":" +
		"\"examples/surface/reference_core/surface_reference_command_p" +
		"alette.tetra\",\"reference_app\":\"command-palette\",\"target\":" +
		"\"linux-x64\",\"diagnostic_schema\":" +
		"\"tetra.surface.diagnostic.v1\",\"scenarios\":[") + crashScenarioJSON(
		"command_failure",
	) + crashScenarioJSON(
		"host_crash",
	) + crashScenarioJSON(
		"restart_recovery",
	) + ("],\"diagnostics\":[{\"path\":" +
		"\"surface-crash/command-failure.json\",\"kind\":" +
		"\"command_failure\",\"schema\":\"tetra.surface.diagnostic.v1\"," +
		"\"sha256\":\"sha256:" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaa\",\"size_bytes\":256,\"redacted\":true,\"contains_user_data\":" +
		"false,\"pass\":true},{\"path\":\"surface-crash/host-crash.json\"," +
		"\"kind\":\"host_crash\",\"schema\":\"tetra.surface.diagnostic.v1\"," +
		"\"sha256\":\"sha256:" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" +
		"bbbb\",\"size_bytes\":256,\"redacted\":true,\"contains_user_data\":" +
		"false,\"pass\":true},{\"path\":" +
		"\"surface-crash/restart-recovery.json\",\"kind\":" +
		"\"restart_recovery\",\"schema\":\"tetra.surface.diagnostic.v1\"," +
		"\"sha256\":\"sha256:" +
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" +
		"cccc\",\"size_bytes\":256,\"redacted\":true,\"contains_user_data\":" +
		"false,\"pass\":true}],\"trace_collection\":{\"trace_path\":" +
		"\"surface-crash/surface-app-trace.json\",\"log_path\":" +
		"\"surface-crash/surface-app.log\",\"ring_buffer\":true," +
		"\"max_bytes\":4096,\"event_count\":4,\"bounded\":true," +
		"\"local_only\":true,\"pass\":true},\"restart_recovery\":{\"scope\":" +
		"\"scoped-linux-x64-process-restart-v1\",\"target\":\"linux-x64\"," +
		"\"restart_claim\":true,\"before_run\":true," +
		"\"failure_report_written\":true,\"after_run\":true," +
		"\"before_exit_code\":0,\"after_exit_code\":0,\"state_restored\":" +
		"\"explicit-startup-state-v1\",\"command\":" +
		"\"surface-crash-work/surface-command-palette-linux-x64\"," +
		"\"pass\":true},\"privacy_policy\":{\"policy\":" +
		"\"surface-non-user-data-diagnostics-v1\",\"redaction_version\":" +
		"\"surface-diagnostic-redaction-v1\",\"user_data_redacted\":true," +
		"\"clipboard_payload_captured\":false,\"user_text_captured\":" +
		"false,\"env_dumped\":false,\"home_path_captured\":false," +
		"\"network_upload\":false,\"local_only\":true,\"pass\":true}," +
		"\"negative_guards\":{\"no_user_data_leak\":true," +
		"\"no_clipboard_payload\":true,\"no_user_text_payload\":true," +
		"\"no_env_dump\":true,\"no_home_path_leak\":true," +
		"\"no_network_upload\":true," +
		"\"no_restart_claim_without_evidence\":true," +
		"\"no_silent_failure\":true,\"no_docs_only_crash_claim\":true," +
		"\"no_electron_crash_reporter_dependency\":true},\"pass\":true}") + "\n"
}

func crashScenarioJSON(kind string) string {
	switch kind {
	case "command_failure":
		return "" +
			"{\"name\":\"command failure boundary\",\"kind\":\"command_failure\"," +
			"\"target\":\"linux-x64\",\"source\":" +
			"\"examples/surface/reference_core/surface_reference_command_p" +
			"alette.tetra\",\"trigger\":\"command.palette.missing\"," +
			"\"diagnostic_path\":\"surface-crash/command-failure.json\"," +
			"\"diagnostic_sha256\":\"sha256:" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaa\",\"report_written\":true,\"command_boundary\":true," +
			"\"host_captured\":false,\"restarted\":false," +
			"\"contains_user_data\":false,\"pass\":true},"
	case "host_crash":
		return "" +
			"{\"name\":\"host crash capture\",\"kind\":\"host_crash\",\"target\":" +
			"\"linux-x64\",\"source\":" +
			"\"examples/surface/reference_core/surface_reference_command_p" +
			"alette.tetra\",\"trigger\":\"surface-host panic harness\"," +
			"\"diagnostic_path\":\"surface-crash/host-crash.json\"," +
			"\"diagnostic_sha256\":\"sha256:" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" +
			"bbbb\",\"report_written\":true,\"command_boundary\":false," +
			"\"host_captured\":true,\"restarted\":false,\"contains_user_data\":" +
			"false,\"pass\":true},"
	case "restart_recovery":
		return "" +
			"{\"name\":\"restart after diagnostic\",\"kind\":" +
			"\"restart_recovery\",\"target\":\"linux-x64\",\"source\":" +
			"\"examples/surface/reference_core/surface_reference_command_p" +
			"alette.tetra\",\"trigger\":\"restart after command failure " +
			"report\",\"diagnostic_path\":" +
			"\"surface-crash/restart-recovery.json\",\"diagnostic_sha256\":" +
			"\"sha256:" +
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" +
			"cccc\",\"report_written\":true,\"command_boundary\":false," +
			"\"host_captured\":false,\"restarted\":true,\"contains_user_data\":" +
			"false,\"pass\":true}"
	default:
		return ""
	}
}

// ---- dev_workflow_test.go ----

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
	raw := strings.Replace(
		validSurfaceDevWorkflowReportWithMorphToPixelsJSON(),
		`"golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003d"`,
		`"golden_checksum":"sha256:000000000000000000000000000000000000000000000000000000000000003c"`,
		1,
	)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected broken Morph-to-pixels chain to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "self-golden") {
		t.Fatalf("error = %v, want self-golden diagnostic", err)
	}
}

func TestValidateSurfaceDevWorkflowReportRejectsHotReloadClaimWithFullRestart(t *testing.T) {
	raw := strings.Replace(
		validSurfaceDevWorkflowReportJSON(),
		`"hot_reload_claim":false`,
		`"hot_reload_claim":true`,
		1,
	)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected hot reload claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "hot reload") {
		t.Fatalf("error = %v, want hot reload diagnostic", err)
	}
}

func TestValidateSurfaceDevWorkflowReportRejectsMissingChangedTokenRecipeSource(t *testing.T) {
	raw := strings.Replace(
		validSurfaceDevWorkflowReportJSON(),
		""+
			",{\"name\":\"recipe rebuild\",\"kind\":\"recipe-change\","+
			"\"changed_path\":"+
			"\"reports/surface-electron-react-beauty-production/P19/dev-wo"+
			"rkflow/dev-fixture/design/recipes.tetra\",\"output_path\":"+
			"\"reports/surface-electron-react-beauty-production/P19/dev-wo"+
			"rkflow/dev-artifacts/recipe/app\",\"duration_ms\":7,"+
			"\"compiled_modules\":[\"design.recipes\"],\"cache_hits\":"+
			"[\"app.main\",\"design.tokens\"],\"pass\":true}",
		``,
		1,
	)
	err := ValidateDevWorkflowReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing recipe-change step to fail")
	}
	if !strings.Contains(err.Error(), "recipe-change") {
		t.Fatalf("error = %v, want recipe-change diagnostic", err)
	}
}

func validSurfaceDevWorkflowReportJSON() string {
	return ("{\"schema\":\"tetra.surface.dev-workflow.v1\",\"model\":" +
		"\"surface-dev-workflow-v1\",\"release_scope\":" +
		"\"surface-v1-linux-web\",\"command\":\"tetra surface dev\"," +
		"\"source\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/app/main.tetra\",\"target\":\"linux-x64\"," +
		"\"mode\":\"fast-rebuild\",\"reload_semantics\":\"fast-rebuild\"," +
		"\"process_restart_required\":true,\"hot_reload_claim\":false," +
		"\"watch\":false,\"supported_targets\":[\"headless\",\"linux-x64\"," +
		"\"wasm32-web\"],\"steps\":[{\"name\":\"initial build\",\"kind\":" +
		"\"initial\",\"changed_path\":\"\",\"output_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-artifacts/initial/app\",\"duration_ms\":25," +
		"\"compiled_modules\":[\"app.main\",\"design.recipes\"," +
		"\"design.tokens\"],\"cache_hits\":[],\"pass\":true},{\"name\":\"warm " +
		"rebuild\",\"kind\":\"warm-cache\",\"changed_path\":\"\"," +
		"\"output_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-artifacts/warm/app\",\"duration_ms\":3," +
		"\"compiled_modules\":[],\"cache_hits\":[\"app.main\"," +
		"\"design.recipes\",\"design.tokens\"],\"pass\":true},{\"name\":" +
		"\"token rebuild\",\"kind\":\"token-change\",\"changed_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/design/tokens.tetra\",\"output_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-artifacts/token/app\",\"duration_ms\":8," +
		"\"compiled_modules\":[\"design.tokens\"],\"cache_hits\":" +
		"[\"app.main\",\"design.recipes\"],\"pass\":true},{\"name\":\"recipe " +
		"rebuild\",\"kind\":\"recipe-change\",\"changed_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/design/recipes.tetra\",\"output_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-artifacts/recipe/app\",\"duration_ms\":7," +
		"\"compiled_modules\":[\"design.recipes\"],\"cache_hits\":" +
		"[\"app.main\",\"design.tokens\"],\"pass\":true},{\"name\":\"source " +
		"rebuild\",\"kind\":\"source-change\",\"changed_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/app/main.tetra\",\"output_path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-artifacts/source/app\",\"duration_ms\":9," +
		"\"compiled_modules\":[\"app.main\"],\"cache_hits\":" +
		"[\"design.recipes\",\"design.tokens\"],\"pass\":true}]," +
		"\"source_diagnostics\":[{\"kind\":\"token\",\"path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/design/tokens.tetra\",\"line\":1,\"column\":1," +
		"\"code\":\"SURFACE_DEV_TOKEN_PATH\",\"message\":\"token file " +
		"participates in Surface fast rebuild\",\"severity\":\"info\"," +
		"\"pass\":true},{\"kind\":\"recipe\",\"path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/design/recipes.tetra\",\"line\":1,\"column\":" +
		"1,\"code\":\"SURFACE_DEV_RECIPE_PATH\",\"message\":\"recipe file " +
		"participates in Surface fast rebuild\",\"severity\":\"info\"," +
		"\"pass\":true},{\"kind\":\"source\",\"path\":" +
		"\"reports/surface-electron-react-beauty-production/P19/dev-wo" +
		"rkflow/dev-fixture/app/main.tetra\",\"line\":1,\"column\":1," +
		"\"code\":\"SURFACE_DEV_SOURCE_PATH\",\"message\":\"source file " +
		"participates in Surface fast rebuild\",\"severity\":\"info\"," +
		"\"pass\":true}],\"negative_guards\":{\"no_hot_reload_claim\":true," +
		"\"full_restart_documented_as_fast_rebuild\":true," +
		"\"no_electron_dev_server\":true,\"no_react_fast_refresh\":true," +
		"\"no_dom_hot_reload\":true},\"pass\":true}") + "\n"
}

func validSurfaceDevWorkflowReportWithMorphToPixelsJSON() string {
	raw := strings.TrimSuffix(validSurfaceDevWorkflowReportJSON(), "\n")
	raw = strings.Replace(
		raw,
		`,"negative_guards":`,
		`,"morph_to_pixels":`+validMorphToPixelsChainJSON(
			"reports/surface-electron-react-beauty-production/P19/dev-workflow/dev-fixture/app/main.tetra",
		)+`,"negative_guards":`,
		1,
	)
	return raw + "\n"
}

func validMorphToPixelsChainJSON(source string) string {
	return ("{\"chain_id\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0900\",\"report_path\":" +
		"\"reports/surface/morph-rendered-beauty.json\",\"schema\":" +
		"\"tetra.surface.morph-rendered-beauty.v1\",\"status\":\"pass\"," +
		"\"surface_scope\":\"surface-morph-rendered-beauty-linux-web\"," +
		"\"source\":\"") + source + ("\",\"source_sha256\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0001\",\"target\":\"headless\",\"scenario_name\":\"headless-morph:") + source + (("\",\"git_" +
		"head\":\"95bfd4a887bab5032437cb22494d034e82ae6d35\",") +
		"\"git_commit\":\"95bfd4a887bab5032437cb22494d034e82ae6d35\"," +
		"\"git_dirty\":true,\"token_graph_hash\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0003\",\"token_count\":6,\"token_categories\":[\"color\",\"space\"," +
		"\"radius\",\"typography\",\"motion\",\"assets\"],\"recipe_count\":3," +
		"\"recipe_expansion_count\":4,\"recipe_names\":[\"studio_shell\"," +
		"\"hero_panel\",\"toolbar\"],\"block_scene_hash\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0005\",\"block_scene_node_count\":12," +
		"\"render_command_stream_hash\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0007\",\"render_command_count\":10,\"renderer\":" +
		"\"software-rgba-headless\",\"frame_artifact\":" +
		"\"reports/surface/frame.rgba\",\"frame_artifact_sha256\":" +
		"\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"003c\",\"frame_checksum\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"003c\",\"golden_artifact\":\"reports/surface/golden.rgba\"," +
		"\"golden_artifact_sha256\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"003d\",\"golden_checksum\":\"sha256:" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"003d\",\"diff_pixels\":1,\"diff_ratio_milli\":0," +
		"\"max_channel_delta\":1,\"product_claim\":false,\"final_signoff\":" +
		"false,\"pass\":true}")
}

// ---- frame_provenance_test.go ----

func TestValidateReportRejectsPrecomputedProductVisualFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "surface-runtime-smoke"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].Precomputed = true
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected precomputed product visual frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "precomputed") ||
		!strings.Contains(strings.ToLower(err.Error()), "product visual") {
		t.Fatalf("error = %v, want precomputed product visual diagnostic", err)
	}
}

func TestValidateReportRejectsProductVisualFrameWithoutSourceLinks(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "app"
		report.Frames[3].EvidenceRole = "product_visual"
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unlinked product visual frame to fail")
	}
	for _, want := range []string{
		"app_source",
		"morph_recipe_hash",
		"block_scene_hash",
		"render_command_stream_hash",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %s diagnostic", err, want)
		}
	}
}

func TestValidateReportAcceptsSourceLinkedProductVisualFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "app"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
	})

	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateReportAcceptsPrecomputedHostProbeOnlyFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Frames[3].Producer = "host_probe"
		report.Frames[3].EvidenceRole = "host_probe_only"
		report.Frames[3].Precomputed = true
	})

	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestBlockSystemRejectsProductVisualFrameFromPrecomputedRenderer(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "host_probe"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].Precomputed = true
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
		report.BlockSystem.Frames[3].Producer = "host_probe"
		report.BlockSystem.Frames[3].EvidenceRole = "product_visual"
		report.BlockSystem.Frames[3].Precomputed = true
	})

	var decoded Report
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected precomputed Block-system product visual frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block_system") ||
		!strings.Contains(strings.ToLower(err.Error()), "product visual") {
		t.Fatalf("error = %v, want Block-system product visual diagnostic", err)
	}
}

func attachProductVisualChainForTest(report *Report) {
	nodes := make([]BlockSceneNodeReport, 0, len(report.BlockGraph.Nodes))
	for _, graphNode := range report.BlockGraph.Nodes {
		nodes = append(nodes, BlockSceneNodeReport{
			BlockID:  graphNode.ID,
			ParentID: graphNode.ParentID,
			Recipe:   "morph.product_visual@1",
			Name:     graphNode.Name,
			Layout: &BlockSceneLayoutSpecReport{
				Mode: "absolute",
				X:    graphNode.Bounds.X,
				Y:    graphNode.Bounds.Y,
				W:    graphNode.Bounds.W,
				H:    graphNode.Bounds.H,
			},
			Paint: &BlockScenePaintSpecReport{
				LayerCount: 1,
				Layers: []BlockScenePaintLayerSpecReport{
					{Kind: "fill", Color: "#202733ff", Radius: 4, Opacity: 255},
				},
			},
			Text: &BlockSceneTextSpecReport{TextLen: 4, Color: "#f4f7fbff", Size: 14, Weight: 500},
			Image: &BlockSceneImageSpecReport{
				AssetID: "search-icon",
				Mode:    "template",
				Tint:    "#b7c4d6ff",
				Opacity: 255,
			},
			Input: &BlockSceneInputSpecReport{
				Kind:      "button",
				Focusable: graphNode.Focusable,
				Editable:  false,
			},
			Event: &BlockSceneEventSpecReport{
				PointerAction: "press",
				KeyAction:     "activate",
			},
			State: &BlockSceneStateSpecReport{
				Variant: "focused",
				Enabled: true,
				Focused: graphNode.Focusable,
			},
			Motion: &BlockSceneMotionSpecReport{
				DurationMS:        120,
				Easing:            "standard",
				ReducedMotionSafe: true,
			},
			Accessibility: &BlockSceneAccessibilitySpecReport{
				Role:         graphNode.AccessibilityRole,
				LabelLen:     4,
				ReadingIndex: graphNode.ID,
			},
		})
	}
	blockSceneHash := "sha256:" + strings.Repeat("c", 64)
	report.BlockSceneSnapshot = &BlockSceneSnapshotReport{
		Schema:               "tetra.surface.block-scene-snapshot.v1",
		Source:               report.Source,
		SurfaceScope:         "surface-morph-rendered-beauty-linux-web",
		Producer:             "surface-runtime-smoke",
		QualityLevel:         "rich-renderable-block-scene-v1",
		CorePrimitives:       []string{"Block"},
		CompactPropsOnly:     false,
		RecipeExpansionCount: len(nodes),
		NodeCount:            len(nodes),
		RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
		BlockSceneHash:       blockSceneHash,
		SpecCoverage: BlockSceneSpecCoverageReport{
			Layout:        true,
			Paint:         true,
			Text:          true,
			Image:         true,
			Input:         true,
			Event:         true,
			State:         true,
			Motion:        true,
			Accessibility: true,
		},
		Nodes: nodes,
	}
	frame := report.Frames[len(report.Frames)-1]
	report.RenderCommandStream = &RenderCommandStreamReport{
		Schema:                        "tetra.surface.render-command-stream.v1",
		Source:                        report.Source,
		SurfaceScope:                  "surface-morph-rendered-beauty-linux-web",
		Producer:                      "surface-runtime-smoke",
		QualityLevel:                  "deterministic-render-command-stream-v1",
		Renderer:                      "software-rgba-headless",
		DerivedFromBlockSceneSnapshot: true,
		BlockSceneHash:                blockSceneHash,
		FrameChecksum:                 frame.Checksum,
		CommandStreamHash:             "sha256:" + strings.Repeat("d", 64),
		CommandCount:                  10,
		SourceLinked:                  true,
		HandcraftedFixture:            false,
		Commands: []RenderCommandReport{
			productVisualRenderCommandForTest(1, "fill"),
			productVisualRenderCommandForTest(2, "gradient"),
			productVisualRenderCommandForTest(3, "image_fill"),
			productVisualRenderCommandForTest(4, "border"),
			productVisualRenderCommandForTest(5, "radius_clip"),
			productVisualRenderCommandForTest(6, "shadow"),
			productVisualRenderCommandForTest(7, "overlay"),
			productVisualRenderCommandForTest(8, "outline"),
			productVisualRenderCommandForTest(9, "text"),
			productVisualRenderCommandForTest(10, "icon"),
		},
	}
}

func productVisualRenderCommandForTest(order int, command string) RenderCommandReport {
	item := RenderCommandReport{
		Order:        order,
		Command:      command,
		Source:       "examples/surface/block_core/surface_block_system.tetra",
		SourceNodeID: "block:2",
		Recipe:       "morph.product_visual@1",
		LayerID:      command + "-layer",
		BlockID:      2,
		Rect:         RectReport{X: 12, Y: 12, W: 296, H: 176},
		Clip:         RectReport{X: 12, Y: 12, W: 296, H: 176},
		Radius:       8,
		Opacity:      255,
		Quality:      "source-linked-block-render-command-v1",
		AssetID:      "search-icon",
		TextLen:      12,
		Checksum:     "sha256:" + strings.Repeat("e", 64),
	}
	if command != "radius_clip" {
		item.Color = productVisualRenderCommandColorForTest(command)
	}
	if command == "border" || command == "outline" {
		item.Width = 1
	}
	if command == "shadow" {
		item.Blur = 8
		item.OffsetY = 2
	}
	if command == "text" {
		item.RasterFormat = "builtin-5x7-alpha-mask-v1"
		item.RasterHash = "sha256:" + strings.Repeat("f", 64)
		item.RasterWidth = 296
		item.RasterHeight = 176
		item.RasterCoverage = 204
	}
	if command == "icon" {
		item.RasterFormat = "builtin-icon-mask-raster-v1"
		item.RasterHash = "sha256:" + strings.Repeat("a", 64)
		item.RasterWidth = 296
		item.RasterHeight = 176
		item.RasterCoverage = 1024
	}
	return item
}

func productVisualRenderCommandColorForTest(command string) string {
	switch command {
	case "fill":
		return "#202733ff"
	case "gradient":
		return "#2c3848ff"
	case "image_fill":
		return "#ffffff22"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#6eaef4ff"
	}
}

// ---- i18n_test.go ----

func TestValidateI18nReportAcceptsLocalizedFormEvidence(t *testing.T) {
	raw := validI18nReportJSON()
	if err := ValidateI18nReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateI18nReport failed: %v\n%s", err, raw)
	}
}

func TestValidateI18nReportRejectsMissingKeyWithoutDiagnostic(t *testing.T) {
	raw := strings.Replace(
		validI18nReportJSON(),
		`"missing_key_diagnostic":true`,
		`"missing_key_diagnostic":false`,
		1,
	)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing key diagnostic absence to fail")
	}
	if !strings.Contains(err.Error(), "missing_key") {
		t.Fatalf("error = %v, want missing_key diagnostic", err)
	}
}

func TestValidateI18nReportRejectsMissingFallbackLanguage(t *testing.T) {
	raw := strings.Replace(
		validI18nReportJSON(),
		`"fallback_locale":"en-US"`,
		`"fallback_locale":""`,
		1,
	)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing fallback language to fail")
	}
	if !strings.Contains(err.Error(), "fallback") {
		t.Fatalf("error = %v, want fallback diagnostic", err)
	}
}

func TestValidateI18nReportRejectsFullBidiClaimWithoutShapingProof(t *testing.T) {
	raw := strings.Replace(
		validI18nReportJSON(),
		`"full_bidi_claim":false`,
		`"full_bidi_claim":true`,
		1,
	)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected full bidi claim without shaping proof to fail")
	}
	if !strings.Contains(err.Error(), "bidi") {
		t.Fatalf("error = %v, want bidi diagnostic", err)
	}
}

func validI18nReportJSON() string {
	return strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.i18n.v1\",",
		"  \"model\": \"surface-i18n-v1\",",
		"  \"release_scope\": \"surface-v1-linux-web\",",
		"  \"producer\": \"scripts/release/surface/surface-i18n-smoke.sh\",",
		("  \"source\": \"examples/surface/reference_forms/surface_reference_loca" +
			"lized_form.tetra\","),
		"  \"reference_app\": \"localized-form\",",
		"  \"target\": \"linux-x64\",",
		"  \"string_tables\": [",
		("    {\"locale\":\"en-US\",\"entry_count\":5,\"checksum\":\"sha256:aaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"primary\":" +
			"true,\"fallback\":false,\"pass\":true},"),
		("    {\"locale\":\"uk-UA\",\"entry_count\":4,\"checksum\":\"sha256:bbbbbbbb" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"primary\":" +
			"false,\"fallback\":true,\"pass\":true}"),
		"  ],",
		("  \"locale_selection\": {\"requested_locale\":\"uk-UA\",\"selected_locale" +
			"\":\"uk-UA\",\"fallback_locale\":\"en-US\",\"fallback_used\":true,\"unsupp" +
			"orted_locale_rejected\":true,\"pass\":true},"),
		"  \"lookups\": [",
		("    {\"key\":\"form.title\",\"locale\":\"uk-UA\",\"resolved_locale\":\"uk-" +
			"UA\",\"source\":\"primary\",\"missing_key\":false,\"fallback_used\":false," +
			"\"diagnostic_code\":0,\"pass\":true},"),
		("    {\"key\":\"form.secondary\",\"locale\":\"uk-UA\",\"resolved_locale\":" +
			"\"en-US\",\"source\":\"fallback\",\"missing_key\":false,\"fallback_used\":" +
			"true,\"diagnostic_code\":0,\"pass\":true},"),
		("    {\"key\":\"form.unknown\",\"locale\":\"uk-UA\",\"resolved_locale\":\"e" +
			"n-US\",\"source\":\"missing\",\"missing_key\":true,\"fallback_used\":true," +
			"\"diagnostic_code\":2001,\"pass\":true}"),
		"  ],",
		"  \"format_hooks\": [",
		("    {\"kind\":\"date\",\"locale\":\"uk-UA\",\"input\":\"2026-06-12\",\"out" +
			"put\":\"2026-06-12\",\"deterministic\":true,\"icu_claim\":false,\"pass\":t" +
			"rue},"),
		("    {\"kind\":\"number\",\"locale\":\"uk-UA\",\"input\":\"4200\",\"output" +
			"\":\"4200\",\"deterministic\":true,\"icu_claim\":false,\"pass\":true}"),
		"  ],",
		("  \"text_direction\": {\"default_direction\":\"ltr\",\"rtl_placeholder\":t" +
			"rue,\"full_bidi_supported\":false,\"full_bidi_claim\":false,\"shaping_pro" +
			"of\":false,\"nonclaim\":\"rtl-placeholder-without-full-bidi-shaping-v1\"," +
			"\"pass\":true},"),
		("  \"localized_form\": {\"shape\":\"localized-form\",\"source\":\"examples/" +
			"surface/reference_forms/surface_reference_localized_form.tetra\",\"imp" +
			"orts\":[\"lib.core.surface\",\"lib.core.block\",\"lib.core.morph\",\"lib.c" +
			"ore.i18n\"],\"compiles\":true,\"runs\":true,\"exit_code\":0,\"localized_st" +
			"rings\":true,\"fallback_evidence\":true,\"missing_key_diagnostic\":true,\"" +
			"format_hook_evidence\":true,\"resolves_to_block\":true,\"pass\":true},"),
		("  \"negative_guards\": {\"no_full_icu_claim\":true,\"no_full_bidi_claim\":" +
			"true,\"no_rtl_production_claim\":true,\"no_missing_key_silent_fallback\"" +
			":true,\"no_docs_only_i18n_claim\":true,\"no_react_intl_runtime\":true,\"n" +
			"o_platform_locale_dependency\":true},"),
		"  \"pass\": true",
		"}",
		"",
	}, "\n")
}

// ---- inspector_test.go ----

func TestValidateSurfaceInspectorReportAcceptsCompleteEvidence(t *testing.T) {
	raw := []byte(validSurfaceInspectorReportJSON())
	if err := ValidateInspectorReport(raw); err != nil {
		t.Fatalf("ValidateInspectorReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceInspectorReportAcceptsMorphToPixelsChain(t *testing.T) {
	raw := []byte(validSurfaceInspectorReportWithMorphToPixelsJSON())
	if err := ValidateInspectorReport(raw); err != nil {
		t.Fatalf("ValidateInspectorReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceInspectorReportRejectsIncompleteMorphToPixelsSections(t *testing.T) {
	raw := strings.Replace(
		validSurfaceInspectorReportWithMorphToPixelsJSON(),
		`"golden_diff":{"present":true`,
		`"golden_diff":{"present":false`,
		1,
	)
	err := ValidateInspectorReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected incomplete Morph-to-pixels inspector sections to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "golden_diff") {
		t.Fatalf("error = %v, want golden_diff diagnostic", err)
	}
}

func TestValidateSurfaceInspectorReportRejectsInvalidSourceLocations(t *testing.T) {
	raw := strings.Replace(
		validSurfaceInspectorReportJSON(),
		`"path":"examples/surface/block_core/surface_block_system.tetra","line":1,"column":1`,
		`"path":"../outside.tetra","line":0,"column":1`,
		1,
	)
	err := ValidateInspectorReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected invalid source location to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "source") ||
		!strings.Contains(strings.ToLower(err.Error()), "location") {
		t.Fatalf("error = %v, want source location diagnostic", err)
	}
}

func TestValidateSurfaceInspectorReportRejectsHiddenStateFindings(t *testing.T) {
	raw := strings.Replace(
		validSurfaceInspectorReportJSON(),
		`"findings":[]`,
		("\"findings\":[{\"path\":" +
			"\"examples/surface/block_core/surface_block_system.tetra\"," +
			"\"kind\":\"hidden-state\",\"message\":\"secret state escaped " +
			"inspector\"}]"),
		1,
	)
	err := ValidateInspectorReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected hidden state finding to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "hidden") {
		t.Fatalf("error = %v, want hidden state diagnostic", err)
	}
}

func TestValidateSurfaceInspectorReportRejectsDOMRuntimeDependency(t *testing.T) {
	raw := strings.Replace(
		validSurfaceInspectorReportJSON(),
		`"no_dom_runtime_dependency":true`,
		`"no_dom_runtime_dependency":false`,
		1,
	)
	err := ValidateInspectorReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected DOM runtime dependency guard to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dom") {
		t.Fatalf("error = %v, want DOM dependency diagnostic", err)
	}
}

func validSurfaceInspectorReportJSON() string {
	return ("{\"schema\":\"tetra.surface.inspector.v1\",\"model\":" +
		"\"surface-inspector-v1\",\"release_scope\":" +
		"\"surface-v1-linux-web\",\"producer\":" +
		"\"tools/cmd/surface-inspector\",\"source\":" +
		"\"examples/surface/block_core/surface_block_system.tetra\"," +
		"\"target\":\"headless\",\"mode\":\"static-tool-report\"," +
		"\"input_reports\":[{\"kind\":\"block\",\"path\":" +
		"\"reports/surface-inspector/inputs/surface-headless-block-sys" +
		"tem.json\",\"schema\":\"tetra.surface.runtime.v1\",\"source\":" +
		"\"examples/surface/block_core/surface_block_system.tetra\"," +
		"\"target\":\"headless\",\"pass\":true},{\"kind\":\"morph\",\"path\":" +
		"\"reports/surface-inspector/inputs/surface-headless-morph.jso" +
		"n\",\"schema\":\"tetra.surface.runtime.v1\",\"source\":" +
		"\"examples/surface/morph_core/surface_morph_command_palette.t" +
		"etra\",\"target\":\"headless\",\"pass\":true},{\"kind\":" +
		"\"accessibility\",\"path\":" +
		"\"reports/surface-inspector/inputs/surface-headless-release-a" +
		"ccessibility.json\",\"schema\":\"tetra.surface.runtime.v1\"," +
		"\"source\":" +
		"\"examples/surface/release/surface_release_accessibility.tetr" +
		"a\",\"target\":\"headless\",\"pass\":true},{\"kind\":\"app-model\"," +
		"\"path\":" +
		"\"reports/surface-inspector/inputs/surface-headless-app-model" +
		".json\",\"schema\":\"tetra.surface.runtime.v1\",\"source\":" +
		"\"examples/surface/toolkit/surface_app_model.tetra\",\"target\":" +
		"\"headless\",\"pass\":true}],\"source_locations\":[{\"kind\":" +
		"\"block\",\"path\":" +
		"\"examples/surface/block_core/surface_block_system.tetra\"," +
		"\"line\":1,\"column\":1},{\"kind\":\"morph\",\"path\":" +
		"\"examples/surface/morph_core/surface_morph_command_palette.t" +
		"etra\",\"line\":1,\"column\":1},{\"kind\":\"accessibility\",\"path\":" +
		"\"examples/surface/release/surface_release_accessibility.tetr" +
		"a\",\"line\":1,\"column\":1},{\"kind\":\"app-model\",\"path\":" +
		"\"examples/surface/toolkit/surface_app_model.tetra\",\"line\":1," +
		"\"column\":1}],\"sections\":{\"block_tree\":{\"present\":true," +
		"\"count\":6,\"source\":\"block_graph.nodes\"},\"morph_tokens\":" +
		"{\"present\":true,\"count\":22,\"source\":" +
		"\"morph.token_graph.tokens\"},\"layout\":{\"present\":true," +
		"\"count\":6,\"source\":\"layout_passes\"},\"paint\":{\"present\":true," +
		"\"count\":10,\"source\":\"paint_commands\"},\"accessibility\":" +
		"{\"present\":true,\"count\":12,\"source\":" +
		"\"accessibility_tree.nodes\"},\"event_routes\":{\"present\":true," +
		"\"count\":5,\"source\":\"block_event_routes\"},\"focus\":{\"present\":" +
		"true,\"count\":3,\"source\":\"block_focus_transitions\"}," +
		"\"perf_counters\":{\"present\":true,\"count\":4,\"source\":" +
		"\"surface_performance_budget\"}},\"static_artifacts\":{\"json\":" +
		"\"reports/surface-inspector/surface-inspector.json\",\"html\":" +
		"\"reports/surface-inspector/surface-inspector.html\"," +
		"\"html_tool_report\":true},\"hidden_state\":{\"scanned\":true," +
		"\"findings\":[]},\"negative_guards\":" +
		"{\"no_dom_runtime_dependency\":true," +
		"\"no_browser_devtools_dependency\":true," +
		"\"no_react_devtools_dependency\":true," +
		"\"static_html_tool_report_only\":true,\"no_hidden_state\":true}," +
		"\"pass\":true}") + "\n"
}

func validSurfaceInspectorReportWithMorphToPixelsJSON() string {
	source := "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	raw := strings.TrimSuffix(validSurfaceInspectorReportJSON(), "\n")
	raw = strings.Replace(
		raw,
		`}],"source_locations":`,
		("},{\"kind\":\"morph-rendered-beauty\",\"path\":"+
			"\"reports/surface-inspector/inputs/morph-rendered-beauty.json"+
			"\",\"schema\":\"tetra.surface.morph-rendered-beauty.v1\","+
			"\"source\":\"")+source+`","target":"headless","pass":true}],"source_locations":`,
		1,
	)
	raw = strings.Replace(
		raw,
		`}],"sections":`,
		`},{"kind":"morph-rendered-beauty","path":"`+source+`","line":1,"column":1}],"sections":`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"perf_counters":{"present":true,"count":4,"source":"surface_performance_budget"}}`,
		("\"perf_counters\":{\"present\":true,\"count\":4,\"source\":" +
			"\"surface_performance_budget\"},\"recipe_expansions\":" +
			"{\"present\":true,\"count\":4,\"source\":" +
			"\"morph_to_pixels.recipe_expansion_count\"}," +
			"\"block_scene_nodes\":{\"present\":true,\"count\":12,\"source\":" +
			"\"morph_to_pixels.block_scene_node_count\"},\"render_commands\":" +
			"{\"present\":true,\"count\":10,\"source\":" +
			"\"morph_to_pixels.render_command_count\"},\"frame_artifacts\":" +
			"{\"present\":true,\"count\":1,\"source\":" +
			"\"morph_to_pixels.frame_artifact\"},\"golden_diff\":{\"present\":" +
			"true,\"count\":1,\"source\":\"morph_to_pixels.golden_artifact + " +
			"diff metrics\"}}"),
		1,
	)
	raw = strings.Replace(
		raw,
		`,"static_artifacts":`,
		`,"morph_to_pixels":`+validMorphToPixelsChainJSON(source)+`,"static_artifacts":`,
		1,
	)
	return raw + "\n"
}

// ---- morph_rendered_beauty_validation_test.go ----

func TestValidateMorphRenderedBeautyReportAcceptsFirstClassReport(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()

	if err := ValidateMorphRenderedBeautyReportValue(report); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingScenarioName(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.ScenarioName = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing scenario_name to fail")
	}
	if !strings.Contains(err.Error(), "scenario_name") {
		t.Fatalf("error = %v, want scenario_name diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingPixelLink(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.PixelEvidence.RenderCommandStreamHash = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing render command stream pixel link to fail")
	}
	if !strings.Contains(err.Error(), "pixel_evidence.render_command_stream_hash") {
		t.Fatalf("error = %v, want pixel render_command_stream_hash diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingGitCommitAlias(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.GitCommit = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing git_commit to fail")
	}
	if !strings.Contains(err.Error(), "git_commit") {
		t.Fatalf("error = %v, want git_commit diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMismatchedGitCommit(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.GitCommit = strings.Repeat("a", 40)

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected mismatched git_commit to fail")
	}
	if !strings.Contains(err.Error(), "git_commit must match git_head") {
		t.Fatalf("error = %v, want git_commit mismatch diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsProductClaimWithoutRendererOwnedStableProof(
	t *testing.T,
) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.ProductClaim = true
	report.RendererStableProof.RendererOwned = false
	report.RendererStableProof.PixelOwner = "morph-evidence-bridge"

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected product_claim without renderer-owned stable proof to fail")
	}
	if !strings.Contains(err.Error(), "renderer_owned stable proof") {
		t.Fatalf("error = %v, want renderer-owned stable proof diagnostic", err)
	}
}

func validMorphRenderedBeautySurfaceReportFixture() MorphRenderedBeautyReport {
	source := "examples/surface/morph_core/surface_morph_command_palette.tetra"
	blockHash := "sha256:" + strings.Repeat("5", 64)
	streamHash := "sha256:" + strings.Repeat("6", 64)
	frameChecksum := "sha256:" + strings.Repeat("8", 64)
	gitCommit := "95bfd4a887bab5032437cb22494d034e82ae6d35"
	return MorphRenderedBeautyReport{
		Schema:         MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   MorphRenderedBeautyScope,
		Target:         "headless",
		ScenarioName:   "headless-morph",
		GitHead:        gitCommit,
		GitCommit:      gitCommit,
		CorePrimitives: []string{"Block"},
		MorphEvidence: MorphRenderedBeautyMorphEvidence{
			Source:         source,
			SourceSHA256:   "sha256:" + strings.Repeat("1", 64),
			CapsuleHash:    "sha256:" + strings.Repeat("2", 64),
			TokenGraphHash: "sha256:" + strings.Repeat("3", 64),
			TokenCount:     22,
			TokenCategories: []string{
				"color",
				"space",
				"radius",
				"typography",
				"motion",
				"assets",
			},
			RecipeCount:            3,
			RecipeExpansionCount:   19,
			RecipeNames:            []string{"control.action@1", "field.text@1", "command.item@1"},
			ResolvedMorphSceneHash: "sha256:" + strings.Repeat("4", 64),
			BlockSceneSnapshotHash: blockHash,
		},
		BlockSceneSnapshot: MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope:         MorphRenderedBeautyScope,
			Source:               source,
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			RecipeExpansionCount: 19,
			NodeCount:            5,
			RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
			BlockSceneHash:       blockHash,
			SpecCoverage: MorphRenderedBeautyBlockSceneSpecCoverage{
				Layout:        true,
				Paint:         true,
				Text:          true,
				Image:         true,
				Input:         true,
				Event:         true,
				State:         true,
				Motion:        true,
				Accessibility: true,
			},
		},
		RenderEvidence: MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: streamHash,
			CommandCount:      10,
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: MorphRenderedBeautyRendererStableProof{
			Schema:                         "tetra.surface.renderer-stable-proof.v1",
			PixelOwner:                     "surface-renderer",
			RendererOwned:                  true,
			BridgeOwnedPixels:              false,
			BlockFirst:                     true,
			DerivedFromRenderCommandStream: true,
			RenderCommandStreamHash:        streamHash,
			BlockSceneHash:                 blockHash,
			FrameChecksum:                  frameChecksum,
			StablePromotionEligible:        true,
		},
		RenderCommandStream: MorphRenderedBeautyRenderCommandStream{
			Schema:                        "tetra.surface.render-command-stream.v1",
			Source:                        source,
			SurfaceScope:                  MorphRenderedBeautyScope,
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                blockHash,
			FrameChecksum:                 frameChecksum,
			CommandStreamHash:             streamHash,
			CommandCount:                  10,
			SourceLinked:                  true,
			Commands: []MorphRenderedBeautyRenderCommand{
				morphRenderedBeautySurfaceCommandForTest(source, 1, "fill"),
				morphRenderedBeautySurfaceCommandForTest(source, 2, "gradient"),
				morphRenderedBeautySurfaceCommandForTest(source, 3, "image_fill"),
				morphRenderedBeautySurfaceCommandForTest(source, 4, "border"),
				morphRenderedBeautySurfaceCommandForTest(source, 5, "radius_clip"),
				morphRenderedBeautySurfaceCommandForTest(source, 6, "shadow"),
				morphRenderedBeautySurfaceCommandForTest(source, 7, "overlay"),
				morphRenderedBeautySurfaceCommandForTest(source, 8, "outline"),
				morphRenderedBeautySurfaceCommandForTest(source, 9, "text"),
				morphRenderedBeautySurfaceCommandForTest(source, 10, "icon"),
			},
		},
		PixelEvidence: MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           "reports/surface/morph/headless/current.rgba",
			FrameArtifactSHA256:     "sha256:" + strings.Repeat("7", 64),
			FrameChecksum:           frameChecksum,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         "sha256:" + strings.Repeat("c", 64),
			BlockSceneHash:          blockHash,
			RenderCommandStreamHash: streamHash,
			GoldenArtifact:          "reports/surface/morph/headless/golden.rgba",
			GoldenArtifactSHA256:    "sha256:" + strings.Repeat("9", 64),
			GoldenChecksum:          "sha256:" + strings.Repeat("a", 64),
		},
		NegativeGuards: MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected:             true,
			SelfGoldenRejected:               true,
			PrecomputedFrameRejected:         true,
			MissingFrameArtifactRejected:     true,
			NoDOMUI:                          true,
			NoCSSRuntime:                     true,
			NoReactRuntime:                   true,
			NoElectronRuntime:                true,
			NoNativeWidgets:                  true,
			NoHiddenAppState:                 true,
			NonBlockOutputRejected:           true,
			DirtyCheckoutProductionRejected:  true,
			UnsupportedTargetRejected:        true,
			RendererOwnedStableProofRequired: true,
		},
		NonClaims: []string{
			"no Electron runtime claim",
			"no React runtime claim",
			"no CSS runtime claim",
			"no DOM-authored UI claim",
			"no GPU renderer production claim",
			"no macOS production claim",
			"no Windows production claim",
		},
	}
}

func morphRenderedBeautySurfaceCommandForTest(
	source string,
	order int,
	command string,
) MorphRenderedBeautyRenderCommand {
	item := MorphRenderedBeautyRenderCommand{
		Order:        order,
		Command:      command,
		Source:       source,
		SourceNodeID: "block:2",
		Recipe:       "morph.recipe",
		LayerID:      "block-2-layer-" + command,
		BlockID:      2,
		Quality:      "source-linked-block-render-command-v1",
		Checksum:     "sha256:" + strings.Repeat("d", 64),
	}
	if command != "radius_clip" {
		item.Color = morphRenderedBeautyCommandColorForTest(command)
	}
	if command == "border" || command == "outline" {
		item.Width = 1
	}
	if command == "shadow" {
		item.Blur = 8
		item.OffsetY = 2
	}
	if command == "text" {
		item.RasterFormat = "builtin-5x7-alpha-mask-v1"
		item.RasterHash = "sha256:" + strings.Repeat("e", 64)
		item.RasterWidth = 16
		item.RasterHeight = 16
		item.RasterCoverage = 24
	}
	if command == "icon" {
		item.RasterFormat = "builtin-icon-mask-raster-v1"
		item.RasterHash = "sha256:" + strings.Repeat("f", 64)
		item.RasterWidth = 16
		item.RasterHeight = 16
		item.RasterCoverage = 48
	}
	return item
}

func morphRenderedBeautyCommandColorForTest(command string) string {
	switch command {
	case "fill":
		return "#202733ff"
	case "gradient":
		return "#2c3848ff"
	case "image_fill":
		return "#ffffff22"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#6eaef4ff"
	}
}

// ---- package_test.go ----

func TestValidatePackageReportAcceptsCompletePackageStory(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	if err := ValidatePackageReport([]byte(raw)); err != nil {
		t.Fatalf("ValidatePackageReport failed: %v\n%s", err, raw)
	}
}

func TestValidatePackageReportAcceptsFlagshipPackageStory(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	raw = strings.Replace(
		raw,
		`"source":"examples/surface/reference_core/surface_reference_command_palette.tetra"`,
		`"source":"examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"reference_app":"command-palette"`,
		`"reference_app":"studio-shell"`,
		1,
	)
	if err := ValidatePackageReport([]byte(raw)); err != nil {
		t.Fatalf("ValidatePackageReport failed for flagship package story: %v\n%s", err, raw)
	}
}

func TestValidatePackageReportRejectsUnexpectedFlagshipInstallExitCode(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	raw = strings.Replace(
		raw,
		`"source":"examples/surface/reference_core/surface_reference_command_palette.tetra"`,
		`"source":"examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"reference_app":"command-palette"`,
		`"reference_app":"studio-shell"`,
		1,
	)
	raw = strings.Replace(raw, `"exit_code":0`, `"exit_code":1`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected unexpected flagship install exit code to fail")
	}
	if !strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("error = %v, want expected_exit_code diagnostic", err)
	}
}

func TestValidatePackageReportRejectsNonzeroExpectedExitForReferenceApp(t *testing.T) {
	raw := strings.Replace(
		validSurfacePackageReportJSON(),
		`"exit_code":0`,
		`"exit_code":1,"expected_exit_code":1`,
		1,
	)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected nonzero expected exit code for reference app to fail")
	}
	if !strings.Contains(err.Error(), "want 0") {
		t.Fatalf("error = %v, want zero-exit diagnostic", err)
	}
}

func TestValidatePackageReportRejectsSigningClaimWithoutEvidence(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"signed":false`, `"signed":true`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected signed package claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "signed") {
		t.Fatalf("error = %v, want signed diagnostic", err)
	}
}

func TestValidatePackageReportRejectsMissingLinuxInstallRun(t *testing.T) {
	raw := strings.Replace(
		validSurfacePackageReportJSON(),
		("\"install_smokes\":[{\"target\":\"linux-x64\",\"package_path\":" +
			"\"surface-packages/surface-command-palette-linux-x64.tar.gz\"," +
			"\"install_dir\":\"surface-install/linux-x64\"," +
			"\"installed_binary\":" +
			"\"surface-install/linux-x64/bin/surface-command-palette\"," +
			"\"command\":" +
			"\"surface-install/linux-x64/bin/surface-command-palette\"," +
			"\"exit_code\":0,\"artifact_hash_verified\":true," +
			"\"package_manifest_verified\":true,\"app_run\":true,\"pass\":" +
			"true}]"),
		`"install_smokes":[]`,
		1,
	)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing linux install/run smoke to fail")
	}
	if !strings.Contains(err.Error(), "install_smokes") {
		t.Fatalf("error = %v, want install_smokes diagnostic", err)
	}
}

func TestValidatePackageReportRejectsRemoteAssetFetch(t *testing.T) {
	raw := strings.Replace(
		validSurfacePackageReportJSON(),
		`"network_fetch_allowed":false`,
		`"network_fetch_allowed":true`,
		1,
	)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected remote asset fetch to fail")
	}
	if !strings.Contains(err.Error(), "network_fetch_allowed") {
		t.Fatalf("error = %v, want network_fetch_allowed diagnostic", err)
	}
}

func TestValidatePackageReportRejectsAutoUpdateClaimWithoutRuntimeEvidence(t *testing.T) {
	raw := strings.Replace(
		validSurfacePackageReportJSON(),
		`"auto_update_runtime_claim":false`,
		`"auto_update_runtime_claim":true`,
		1,
	)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected auto-update runtime claim to fail")
	}
	if !strings.Contains(err.Error(), "auto_update_runtime_claim") {
		t.Fatalf("error = %v, want auto_update_runtime_claim diagnostic", err)
	}
}

func validSurfacePackageReportJSON() string {
	return ("{\"schema\":\"tetra.surface.package.v1\",\"model\":" +
		"\"surface-package-v1\",\"release_scope\":\"surface-v1-linux-web\"," +
		"\"producer\":" +
		"\"scripts/release/surface/surface-package-smoke.sh\",\"source\":" +
		"\"examples/surface/reference_core/surface_reference_command_p" +
		"alette.tetra\",\"reference_app\":\"command-palette\"," +
		"\"package_format\":\"surface-app-package-v1\",\"format_version\":" +
		"1,\"artifact_root\":\"surface-package-work\",\"packages\":" +
		"[{\"target\":\"linux-x64\",\"kind\":\"linux-x64-tar.gz\",\"path\":" +
		"\"surface-packages/surface-command-palette-linux-x64.tar.gz\"," +
		"\"manifest_path\":" +
		"\"surface-package-work/linux-x64/package-manifest.json\"," +
		"\"sha256\":\"sha256:" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaa\",\"asset_manifest_sha256\":\"sha256:" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" +
		"bbbb\",\"source_sha256\":\"sha256:" +
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" +
		"cccc\",\"build_sha256\":\"sha256:" +
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" +
		"dddd\",\"contains_executable\":true,\"contains_web_bundle\":" +
		"false,\"local_only_assets\":true,\"pass\":true},{\"target\":" +
		"\"wasm32-web\",\"kind\":\"wasm32-web-tar.gz\",\"path\":" +
		"\"surface-packages/surface-command-palette-wasm32-web.tar.gz\"" +
		",\"manifest_path\":" +
		"\"surface-package-work/wasm32-web/package-manifest.json\"," +
		"\"sha256\":\"sha256:" +
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" +
		"eeee\",\"asset_manifest_sha256\":\"sha256:" +
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" +
		"ffff\",\"source_sha256\":\"sha256:" +
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" +
		"cccc\",\"build_sha256\":\"sha256:" +
		"111111111111111111111111111111111111111111111111111111111111" +
		"1111\",\"contains_executable\":false,\"contains_web_bundle\":" +
		"true,\"local_only_assets\":true,\"pass\":true}],\"assets\":" +
		"[{\"path\":\"surface-package-work/assets/app-icon.txt\",\"kind\":" +
		"\"icon\",\"sha256\":\"sha256:" +
		"222222222222222222222222222222222222222222222222222222222222" +
		"2222\",\"size_bytes\":32,\"local_only\":true," +
		"\"network_fetch_allowed\":false,\"pass\":true},{\"path\":" +
		"\"surface-package-work/assets/theme-manifest.json\",\"kind\":" +
		"\"theme\",\"sha256\":\"sha256:" +
		"333333333333333333333333333333333333333333333333333333333333" +
		"3333\",\"size_bytes\":64,\"local_only\":true," +
		"\"network_fetch_allowed\":false,\"pass\":true}]," +
		"\"install_smokes\":[{\"target\":\"linux-x64\",\"package_path\":" +
		"\"surface-packages/surface-command-palette-linux-x64.tar.gz\"," +
		"\"install_dir\":\"surface-install/linux-x64\"," +
		"\"installed_binary\":" +
		"\"surface-install/linux-x64/bin/surface-command-palette\"," +
		"\"command\":" +
		"\"surface-install/linux-x64/bin/surface-command-palette\"," +
		"\"exit_code\":0,\"artifact_hash_verified\":true," +
		"\"package_manifest_verified\":true,\"app_run\":true,\"pass\":" +
		"true}],\"web_bundles\":[{\"target\":\"wasm32-web\",\"package_path\":" +
		"\"surface-packages/surface-command-palette-wasm32-web.tar.gz\"" +
		",\"web_entry\":\"surface-package-work/wasm32-web/index.html\"," +
		"\"wasm_artifact\":" +
		"\"surface-package-work/wasm32-web/surface-command-palette.was" +
		"m\",\"loader_artifact\":" +
		"\"surface-package-work/wasm32-web/surface-command-palette.mjs" +
		"\",\"browser_canvas_host\":" +
		"\"surface-package-work/wasm32-web/surface-browser-canvas-host" +
		".mjs\",\"command\":\"tetra build --target wasm32-web\"," +
		"\"artifact_hash_verified\":true,\"package_manifest_verified\":" +
		"true,\"pass\":true}],\"update_strategy\":{\"strategy\":" +
		"\"hash-pinned-channel-manifest-v1\",\"manifest_format\":" +
		"\"tetra.surface.update-channel.v1\",\"channel_manifest\":" +
		"\"surface-updates/channel.json\",\"current_version\":\"p23.0.0\"," +
		"\"latest_version\":\"p23.0.0\",\"latest_package_path\":" +
		"\"surface-packages/surface-command-palette-linux-x64.tar.gz\"," +
		"\"latest_package_sha256\":\"sha256:" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaa\",\"package_hash_pinned\":true,\"rollback_manifest\":" +
		"\"surface-updates/rollback.json\"," +
		"\"signature_required_for_stable_promotion\":true," +
		"\"auto_update_runtime_claim\":false,\"network_update_claim\":" +
		"false,\"pass\":true},\"signing\":{\"status\":\"nonclaim\",\"signed\":" +
		"false,\"notarized\":false,\"production_claim\":false,\"evidence\":" +
		"\"\",\"blocked_reason\":\"platform signing keys and CI signing " +
		"evidence are not present in this release\"},\"notarization\":" +
		"{\"status\":\"nonclaim\",\"signed\":false,\"notarized\":false," +
		"\"production_claim\":false,\"evidence\":\"\",\"blocked_reason\":" +
		"\"macOS notarization evidence is unavailable because macOS " +
		"Surface target host is unsupported\"},\"negative_guards\":" +
		"{\"no_react_runtime\":true,\"no_electron_runtime\":true," +
		"\"no_dom_app_ui_tree\":true,\"no_css_runtime\":true," +
		"\"no_user_js_app_logic\":true,\"no_remote_asset_fetch\":true," +
		"\"no_unsigned_signing_claim\":true," +
		"\"no_notarization_without_platform_evidence\":true," +
		"\"no_auto_update_without_runtime_evidence\":true," +
		"\"no_docs_only_package_claim\":true,\"install_run_required\":" +
		"true,\"web_bundle_required\":true,\"artifact_hashes_required\":" +
		"true},\"pass\":true}") + "\n"
}

// ---- reference_apps_test.go ----

func TestValidateReferenceAppsReportAcceptsCompleteSuite(t *testing.T) {
	raw := validReferenceAppsSuiteReportJSON()
	if err := ValidateReferenceAppsReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateReferenceAppsReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReferenceAppsReportRejectsMissingRequiredShape(t *testing.T) {
	raw := strings.Replace(
		validReferenceAppsSuiteReportJSON(),
		`,`+referenceAppJSON(
			"migration",
			"examples/surface/reference_forms/surface_reference_migration.tetra",
			true,
		),
		"",
		1,
	)
	err := ValidateReferenceAppsReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing migration reference app to fail")
	}
	if !strings.Contains(err.Error(), "migration") {
		t.Fatalf("error = %v, want missing migration diagnostic", err)
	}
}

func TestValidateReferenceAppsReportRejectsScreenshotOnlyEvidence(t *testing.T) {
	raw := strings.Replace(
		validReferenceAppsSuiteReportJSON(),
		`"screenshot_only":false`,
		`"screenshot_only":true`,
		1,
	)
	err := ValidateReferenceAppsReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected screenshot-only evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "screenshot") {
		t.Fatalf("error = %v, want screenshot-only diagnostic", err)
	}
}

func TestValidateReferenceAppsReportRejectsProductAppWithoutMorphToPixels(t *testing.T) {
	raw := strings.Replace(
		validReferenceAppsSuiteReportJSON(),
		`,"morph_to_pixels":`+validMorphToPixelsChainJSON(
			"examples/surface/reference_core/surface_reference_command_palette.tetra",
		),
		"",
		1,
	)
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
		referenceAppJSON(
			"command-palette",
			"examples/surface/reference_core/surface_reference_command_palette.tetra",
			false,
		),
		referenceAppJSON(
			"settings",
			"examples/surface/reference_core/surface_reference_settings.tetra",
			false,
		),
		referenceAppJSON(
			"dashboard",
			"examples/surface/reference_core/surface_reference_dashboard.tetra",
			false,
		),
		referenceAppJSON(
			"editor-shell",
			"examples/surface/reference_core/surface_reference_editor_shell.tetra",
			false,
		),
		referenceAppJSON(
			"file-manager",
			"examples/surface/reference_core/surface_reference_file_manager.tetra",
			false,
		),
		referenceAppJSON(
			"dialog-notification",
			"examples/surface/reference_core/surface_reference_dialog_notification.tetra",
			false,
		),
		referenceAppJSON(
			"localized-form",
			"examples/surface/reference_forms/surface_reference_localized_form.tetra",
			false,
		),
		referenceAppJSON(
			"accessibility-form",
			"examples/surface/reference_forms/surface_reference_accessibility_form.tetra",
			false,
		),
		referenceAppJSON(
			"multi-window-notes",
			"examples/surface/reference_forms/surface_reference_multi_window_notes.tetra",
			false,
		),
		referenceAppJSON(
			"migration",
			"examples/surface/reference_forms/surface_reference_migration.tetra",
			true,
		),
	}
	return ("{\"schema\":\"tetra.surface.reference-app-suite.v1\",\"model\":" +
		"\"surface-reference-app-suite-v1\",\"release_scope\":" +
		"\"surface-v1-linux-web\",\"producer\":" +
		"\"scripts/release/surface/surface-reference-apps-smoke.sh\"," +
		"\"app_count\":10,\"required_targets\":[\"headless\"," +
		"\"linux-x64-real-window\",\"wasm32-web-browser-canvas\"],\"apps\":" +
		"[") + strings.Join(
		apps,
		",",
	) + ("],\"visual_evidence\":{\"path\":" +
		"\"reference-visual/surface-visual-regression.json\",\"schema\":" +
		"\"tetra.surface.visual-regression.v1\",\"app_count\":10,\"pass\":" +
		"true},\"negative_guards\":{\"screenshot_only_rejected\":true," +
		"\"missing_interaction_rejected\":true," +
		"\"missing_accessibility_rejected\":true," +
		"\"missing_performance_rejected\":true," +
		"\"core_widget_usage_rejected\":true," +
		"\"migration_widgets_compatibility_only\":true," +
		"\"no_react_runtime\":true,\"no_electron_runtime\":true," +
		"\"no_dom_app_ui_tree\":true,\"no_css_runtime\":true," +
		"\"no_user_js_app_logic\":true},\"pass\":true}") + "\n"
}

func referenceAppJSON(shape string, source string, compatibility bool) string {
	targets := []string{
		referenceTargetJSON("headless"),
		referenceTargetJSON("linux-x64-real-window"),
		referenceTargetJSON("wasm32-web-browser-canvas"),
	}
	module := strings.TrimSuffix(
		strings.TrimPrefix(strings.ReplaceAll(source, "/", "."), "examples."),
		".tetra",
	)
	base := `{"shape":"` + shape + `","source":"` + source +
		`","module":"examples.` + module +
		("\",\"imports\":[\"lib.core.surface\",\"lib.core.block\"," +
			"\"lib.core.morph\"],\"recipes\":[\"region.panel\",\"field.text\"," +
			"\"control.action\",\"command.item\"],\"beauty_coverage\":") + referenceBeautyCoverageJSON(
		shape,
	) + (",\"stable_morph_recipes\":true,\"resolves_to_block\":true," +
		"\"compiles\":true,\"runs\":true,\"exit_code\":0," +
		"\"token_theme_conformance\":true,\"layout_report\":true," +
		"\"interaction_trace\":true,\"accessibility_snapshot\":true," +
		"\"performance_budget\":true,\"artifact_hashes\":true," +
		"\"compatibility_widgets\":") + boolJSON(
		compatibility,
	)
	if compatibility {
		return base + (",\"infrastructure_only\":true,\"non_product_reason\":\"legacy " +
			"widget migration compatibility evidence only\",\"targets\":[") + strings.Join(
			targets,
			",",
		) + `]}`
	}
	return base + `,"infrastructure_only":false,"morph_to_pixels":` + validMorphToPixelsChainJSON(
		source,
	) + `,"targets":[` + strings.Join(
		targets,
		",",
	) + `]}`
}

func referenceBeautyCoverageJSON(shape string) string {
	switch shape {
	case "command-palette":
		return strings.Join([]string{
			"[\"command-palette\",\"focus-state\"]",
		}, "\n")
	case "settings":
		return strings.Join([]string{
			"[\"settings\",\"disabled-state\"]",
		}, "\n")
	case "dashboard":
		return strings.Join([]string{
			"[\"dashboard\"]",
		}, "\n")
	case "editor-shell":
		return strings.Join([]string{
			"[\"editor-shell\"]",
		}, "\n")
	case "dialog-notification":
		return strings.Join([]string{
			"[\"elevated-panel\"]",
		}, "\n")
	case "migration":
		return strings.Join([]string{
			"[]",
		}, "\n")
	default:
		return strings.Join([]string{
			"[\"focus-state\"]",
		}, "\n")
	}
}

func referenceTargetJSON(target string) string {
	runtimeReport := "reference-runtime/" + target + ".json"
	return `{"target":"` + target + `","runtime_report":"` + runtimeReport +
		("\",\"frame_checksum\":\"sha256:" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaa\",\"visual_diff\":true,\"interaction_trace\":true," +
			"\"accessibility_snapshot\":true,\"performance_budget\":true," +
			"\"pass\":true,\"screenshot_only\":false}")
}

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// ---- release_negative_test.go ----

type releaseNegativeFixture struct {
	Validator     string `json:"validator"`
	Mutation      string `json:"mutation"`
	ExpectedError string `json:"expected_error"`
}

func TestSurfaceReleaseNegativeFixturesRejectFakeClaims(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "release_negative")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read release negative fixtures: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	wantNames := []string{
		"browser_dom_ui_claims_surface.json",
		"browser_node_only_claims_release.json",
		"browser_user_js_claims_surface.json",
		"clipboard_claim_without_transfer.json",
		"legacy_sidecars.json",
		"linux_memfd_claims_release_window.json",
		"linux_real_window_missing_accessibility.json",
		"linux_real_window_missing_clipboard.json",
		"metadata_only_accessibility_claims_production.json",
		"platform_claim_without_probe.json",
		"release_report_experimental_true.json",
		"release_report_production_false.json",
		"release_summary_missing_command_line.json",
		"release_summary_missing_producer.json",
		"release_summary_missing_unsupported_targets.json",
		"release_summary_stale_git_head.json",
		"screen_reader_claim_without_artifact.json",
		"stale_artifact_hash.json",
		"text_input_missing_composition_trace.json",
		"text_input_missing_utf8_validation.json",
		"toolkit_manual_bookkeeping_true.json",
		"toolkit_missing_checkbox.json",
		"toolkit_missing_scroll.json",
		"toolkit_single_example_claims_production.json",
	}
	if strings.Join(names, "\n") != strings.Join(wantNames, "\n") {
		t.Fatalf("release negative fixture names = %#v, want %#v", names, wantNames)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var fixture releaseNegativeFixture
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			err = runReleaseNegativeFixture(t, fixture)
			if err == nil {
				t.Fatalf("fixture %s unexpectedly passed", name)
			}
			if !strings.Contains(
				strings.ToLower(err.Error()),
				strings.ToLower(fixture.ExpectedError),
			) {
				t.Fatalf("fixture %s error = %v, want %q", name, err, fixture.ExpectedError)
			}
		})
	}
}

func TestSurfaceReleaseRejectsExperimentalTrue(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_report_experimental_true.json")
}

func TestSurfaceReleaseRejectsProductionClaimFalse(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_report_production_false.json")
}

func TestSurfaceReleaseRejectsMetadataOnlyAccessibility(t *testing.T) {
	requireReleaseNegativeFixture(t, "metadata_only_accessibility_claims_production.json")
}

func TestSurfaceReleaseRejectsPlatformClaimWithoutProbe(t *testing.T) {
	requireReleaseNegativeFixture(t, "platform_claim_without_probe.json")
}

func TestSurfaceReleaseRejectsNodeOnlyBrowserRelease(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_node_only_claims_release.json")
}

func TestSurfaceReleaseRejectsMemfdLinuxRelease(t *testing.T) {
	requireReleaseNegativeFixture(t, "linux_memfd_claims_release_window.json")
}

func TestSurfaceReleaseRejectsMissingClipboard(t *testing.T) {
	requireReleaseNegativeFixture(t, "linux_real_window_missing_clipboard.json")
}

func TestSurfaceReleaseRejectsMissingComposition(t *testing.T) {
	requireReleaseNegativeFixture(t, "text_input_missing_composition_trace.json")
}

func TestSurfaceReleaseRejectsToolkitWithoutScroll(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_missing_scroll.json")
}

func TestSurfaceReleaseRejectsToolkitWithoutCheckbox(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_missing_checkbox.json")
}

func TestSurfaceReleaseRejectsToolkitSingleExampleClaim(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_single_example_claims_production.json")
}

func TestSurfaceReleaseRejectsUserJS(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_user_js_claims_surface.json")
}

func TestSurfaceReleaseRejectsDOMUI(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_dom_ui_claims_surface.json")
}

func TestSurfaceReleaseRejectsLegacySidecars(t *testing.T) {
	requireReleaseNegativeFixture(t, "legacy_sidecars.json")
}

func TestSurfaceReleaseRejectsCopiedMissingProducerMetadata(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_summary_missing_producer.json")
}

func TestSurfaceReleaseRejectsStaleGitHead(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_summary_stale_git_head.json")
}

func TestSurfaceReleaseRejectsMissingCommandLine(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_summary_missing_command_line.json")
}

func requireReleaseNegativeFixture(t *testing.T, name string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "release_negative", name))
	if err != nil {
		t.Fatalf("read release negative fixture %s: %v", name, err)
	}
	var fixture releaseNegativeFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode release negative fixture %s: %v", name, err)
	}
	err = runReleaseNegativeFixture(t, fixture)
	if err == nil {
		t.Fatalf("fixture %s unexpectedly passed", name)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fixture.ExpectedError)) {
		t.Fatalf("fixture %s error = %v, want %q", name, err, fixture.ExpectedError)
	}
}

func runReleaseNegativeFixture(t *testing.T, fixture releaseNegativeFixture) error {
	t.Helper()
	switch fixture.Validator {
	case "release-summary":
		return ValidateReleaseSummary(mutateReleaseSummaryFixture(t, fixture.Mutation))
	case "text-input":
		return ValidateTextInputReport(mutateTextInputFixture(t, fixture.Mutation))
	case "linux-release-window":
		return ValidateReport(
			validLinuxReleaseWindowSurfaceReportJSON(t, func(report map[string]any) {
				mutateLinuxReleaseWindowFixture(t, report, fixture.Mutation)
			}),
		)
	case "browser-release":
		return ValidateReport(
			validWASM32WebReleaseBrowserSurfaceReportJSON(t, func(report map[string]any) {
				mutateBrowserReleaseFixture(t, report, fixture.Mutation)
			}),
		)
	case "accessibility-release":
		return ValidateReport(
			validLinuxReleaseAccessibilitySurfaceReportJSON(t, func(report map[string]any) {
				mutateAccessibilityReleaseFixture(t, report, fixture.Mutation)
			}),
		)
	case "accessibility-metadata":
		return ValidateReport(
			validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
				tree := report["accessibility_tree"].(map[string]any)
				tree["experimental"] = false
				tree["production_claim"] = true
			}),
		)
	case "toolkit":
		return ValidateReport(
			validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
				mutateToolkitFixture(t, report, fixture.Mutation)
			}),
		)
	default:
		t.Fatalf("unknown fixture validator %q", fixture.Validator)
	}
	return nil
}

func mutateReleaseSummaryFixture(t *testing.T, mutation string) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validSurfaceReleaseSummaryJSON(), &report); err != nil {
		t.Fatalf("decode release summary fixture: %v", err)
	}
	switch mutation {
	case "missing_unsupported_targets":
		report["unsupported_targets"] = []any{"macos-x64", "windows-x64"}
	case "experimental_true":
		report["experimental"] = true
	case "production_false":
		report["production_claim"] = false
	case "stale_artifact_hash":
		report["artifact_hashes_validated"] = false
	case "missing_producer":
		delete(report, "producer")
	case "stale_git_head":
		report["git_head"] = "unknown"
	case "missing_command_line":
		delete(report, "command_line")
	default:
		t.Fatalf("unknown release summary mutation %q", mutation)
	}
	return mustMarshalFixture(t, report)
}

func mutateTextInputFixture(t *testing.T, mutation string) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validSurfaceTextInputReportJSON(), &report); err != nil {
		t.Fatalf("decode text input fixture: %v", err)
	}
	switch mutation {
	case "missing_utf8_validation":
		report["utf8_validation"] = false
	case "missing_composition_trace":
		report["composition_trace"] = map[string]any{
			"start":  true,
			"update": true,
			"commit": false,
			"cancel": true,
		}
	case "clipboard_without_transfer":
		report["clipboard_owned_copy"] = false
	default:
		t.Fatalf("unknown text input mutation %q", mutation)
	}
	return mustMarshalFixture(t, report)
}

func mutateLinuxReleaseWindowFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	host := report["host_evidence"].(map[string]any)
	switch mutation {
	case "memfd_claims_release":
		host["level"] = "linux-x64-memfd-starter"
	case "missing_clipboard":
		host["clipboard"] = false
	case "missing_accessibility":
		host["accessibility_bridge"] = false
	default:
		t.Fatalf("unknown linux release window mutation %q", mutation)
	}
}

func mutateBrowserReleaseFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	host := report["host_evidence"].(map[string]any)
	switch mutation {
	case "node_only_claims_release":
		processes := report["processes"].([]any)
		app := processes[1].(map[string]any)
		app["path"] = "node scripts/tools/web_run_module.mjs surface-release-form.wasm"
	case "dom_ui_claims_surface":
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_dom_ui"] = false
	case "user_js_claims_surface":
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_user_js"] = false
	case "legacy_sidecars":
		artifacts := report["artifacts"].([]any)
		report["artifacts"] = append(artifacts, map[string]any{
			"kind":   "legacy-ui-sidecar",
			"path":   "/tmp/surface-artifacts/surface-release-form.ui.json",
			"sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"size":   float64(128),
		})
		report["artifact_scan"].(map[string]any)["files_checked"] = float64(4)
	default:
		t.Fatalf("unknown browser release mutation %q", mutation)
	}
	host["level"] = "wasm32-web-browser-canvas-release-v1"
}

func mutateAccessibilityReleaseFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	tree := report["accessibility_tree"].(map[string]any)
	switch mutation {
	case "platform_claim_without_probe":
		tree["linux_platform_probe"] = false
	case "screen_reader_without_artifact":
		tree["linux_probe_artifact"] = ""
	default:
		t.Fatalf("unknown accessibility release mutation %q", mutation)
	}
}

func mutateToolkitFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	toolkit := report["toolkit"].(map[string]any)
	switch mutation {
	case "single_example_claims_production":
		toolkit["example_count"] = float64(0)
		toolkit["sources"] = []any{"examples/surface/toolkit/surface_toolkit_form.tetra"}
	case "missing_scroll":
		toolkit["widget_set"] = removeStringAny(toolkit["widget_set"].([]any), "Scroll")
	case "missing_checkbox":
		toolkit["widget_set"] = removeStringAny(toolkit["widget_set"].([]any), "Checkbox")
	case "manual_bookkeeping_true":
		toolkit["manual_bookkeeping"] = true
	default:
		t.Fatalf("unknown toolkit mutation %q", mutation)
	}
}

func removeStringAny(values []any, remove string) []any {
	var out []any
	for _, value := range values {
		if value == remove {
			continue
		}
		out = append(out, value)
	}
	return out
}

func mustMarshalFixture(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

// ---- render_command_stream_test.go ----

func TestValidateRenderCommandStreamAcceptsSourceLinkedBlockSceneCommands(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateRenderCommandStreamRejectsUnlinkedOrHandcraftedEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(
		t,
		func(stream map[string]any) {
			stream["source_linked"] = false
			stream["handcrafted_fixture"] = true
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unlinked handcrafted render command stream to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "source_linked") || !strings.Contains(lower, "handcrafted") {
		t.Fatalf("error = %v, want source_linked and handcrafted diagnostics", err)
	}
}

func TestValidateRenderCommandStreamRejectsBlockSceneHashMismatch(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(
		t,
		func(stream map[string]any) {
			stream["block_scene_hash"] = "sha256:" + strings.Repeat("e", 64)
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected render command stream block_scene_hash mismatch to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block_scene_hash") {
		t.Fatalf("error = %v, want block_scene_hash diagnostic", err)
	}
}

func TestValidateRenderCommandStreamRejectsCommandMissingSourceRecipeLink(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(
		t,
		func(stream map[string]any) {
			commands := stream["commands"].([]any)
			first := commands[0].(map[string]any)
			first["source"] = "fixtures/precomputed/render.json"
			first["recipe"] = ""
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected render command without source/recipe link to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "source") || !strings.Contains(lower, "recipe") {
		t.Fatalf("error = %v, want source and recipe diagnostics", err)
	}
}

func TestValidateRenderCommandStreamRejectsMarkerOnlyTextIconRaster(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(
		t,
		func(stream map[string]any) {
			commands := stream["commands"].([]any)
			text := commands[8].(map[string]any)
			icon := commands[9].(map[string]any)
			text["marker_only"] = true
			text["raster_hash"] = ""
			icon["marker_only"] = true
			icon["raster_hash"] = ""
		},
	)

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected marker-only text/icon raster stream to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "marker") || !strings.Contains(lower, "raster") {
		t.Fatalf("error = %v, want marker and raster diagnostics", err)
	}
}

func validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessMorphSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode Morph surface report: %v", err)
	}
	source := report["source"].(string)
	snapshot := blockSceneSnapshotMapForTest(source)
	makeBlockSceneSnapshotRenderCommandRich(snapshot)
	report["block_scene_snapshot"] = snapshot
	frames := report["frames"].([]any)
	firstFrame := frames[0].(map[string]any)
	stream := renderCommandStreamMapForTest(
		source,
		snapshot["block_scene_hash"].(string),
		firstFrame["checksum"].(string),
	)
	if mutate != nil {
		mutate(stream)
	}
	report["render_command_stream"] = stream
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal render command stream report: %v", err)
	}
	return raw
}

func makeBlockSceneSnapshotRenderCommandRich(snapshot map[string]any) {
	nodes := snapshot["nodes"].([]any)
	node := nodes[1].(map[string]any)
	node["paint"] = map[string]any{
		"layer_count": 8,
		"layers": []any{
			map[string]any{"kind": "fill", "color": "#202733ff", "radius": 8, "opacity": 255},
			map[string]any{"kind": "gradient", "color": "#2c3848ff", "radius": 8, "opacity": 255},
			map[string]any{"kind": "image_fill", "color": "#ffffff22", "radius": 8, "opacity": 96},
			map[string]any{
				"kind":    "border",
				"color":   "#6eaef4ff",
				"width":   1,
				"radius":  8,
				"opacity": 255,
			},
			map[string]any{"kind": "radius_clip", "radius": 8, "opacity": 255},
			map[string]any{
				"kind":     "shadow",
				"color":    "#00000040",
				"blur":     8,
				"offset_y": 2,
				"opacity":  64,
			},
			map[string]any{"kind": "overlay", "color": "#10182066", "radius": 8, "opacity": 102},
			map[string]any{
				"kind":    "outline",
				"color":   "#6eaef4ff",
				"width":   1,
				"radius":  8,
				"opacity": 255,
			},
		},
	}
}

func renderCommandStreamMapForTest(
	source string,
	blockSceneHash string,
	frameChecksum string,
) map[string]any {
	commands := []any{
		renderCommandMapForTest(1, "fill", source, "morph.search_input", "search-input-fill", 255),
		renderCommandMapForTest(
			2,
			"gradient",
			source,
			"morph.search_input",
			"search-input-gradient",
			255,
		),
		renderCommandMapForTest(
			3,
			"image_fill",
			source,
			"morph.search_input",
			"search-input-image-fill",
			96,
		),
		renderCommandMapForTest(
			4,
			"border",
			source,
			"morph.search_input",
			"search-input-border",
			255,
		),
		renderCommandMapForTest(
			5,
			"radius_clip",
			source,
			"morph.search_input",
			"search-input-radius-clip",
			255,
		),
		renderCommandMapForTest(
			6,
			"shadow",
			source,
			"morph.search_input",
			"search-input-shadow",
			64,
		),
		renderCommandMapForTest(
			7,
			"overlay",
			source,
			"morph.search_input",
			"search-input-overlay",
			102,
		),
		renderCommandMapForTest(
			8,
			"outline",
			source,
			"morph.search_input",
			"search-input-outline",
			255,
		),
		renderCommandMapForTest(9, "text", source, "morph.search_input", "search-input-text", 255),
		renderCommandMapForTest(10, "icon", source, "morph.search_input", "search-input-icon", 255),
	}
	return map[string]any{
		"schema":                            "tetra.surface.render-command-stream.v1",
		"source":                            source,
		"surface_scope":                     "surface-morph-rendered-beauty-linux-web",
		"producer":                          "surface-runtime-smoke",
		"quality_level":                     "deterministic-render-command-stream-v1",
		"renderer":                          "software-rgba-headless",
		"derived_from_block_scene_snapshot": true,
		"block_scene_hash":                  blockSceneHash,
		"frame_checksum":                    frameChecksum,
		"command_stream_hash":               "sha256:" + strings.Repeat("d", 64),
		"command_count":                     len(commands),
		"source_linked":                     true,
		"handcrafted_fixture":               false,
		"commands":                          commands,
	}
}

func renderCommandMapForTest(
	order int,
	command string,
	source string,
	recipe string,
	layerID string,
	opacity int,
) map[string]any {
	item := map[string]any{
		"order":          order,
		"command":        command,
		"source":         source,
		"source_node_id": "block:2",
		"recipe":         recipe,
		"layer_id":       layerID,
		"block_id":       2,
		"rect":           map[string]any{"x": 16, "y": 16, "w": 288, "h": 168},
		"clip":           map[string]any{"x": 16, "y": 16, "w": 288, "h": 168},
		"radius":         8,
		"opacity":        opacity,
		"quality":        "source-linked-block-render-command-v1",
		"asset_id":       "search-icon",
		"text_len":       12,
		"checksum":       renderCommandChecksumForOrder(order),
	}
	if command != "radius_clip" {
		item["color"] = renderCommandColorForTest(command)
	}
	if command == "border" || command == "outline" {
		item["width"] = 1
	}
	if command == "shadow" {
		item["blur"] = 8
		item["offset_y"] = 2
	}
	if command == "text" {
		item["raster_format"] = "builtin-5x7-alpha-mask-v1"
		item["raster_hash"] = "sha256:" + strings.Repeat("b", 64)
		item["raster_width"] = 288
		item["raster_height"] = 168
		item["raster_coverage"] = 204
		item["marker_only"] = false
	}
	if command == "icon" {
		item["raster_format"] = "builtin-icon-mask-raster-v1"
		item["raster_hash"] = "sha256:" + strings.Repeat("c", 64)
		item["raster_width"] = 288
		item["raster_height"] = 168
		item["raster_coverage"] = 16128
		item["marker_only"] = false
	}
	return item
}

func renderCommandColorForTest(command string) string {
	switch command {
	case "fill":
		return "#202733ff"
	case "gradient":
		return "#2c3848ff"
	case "image_fill":
		return "#ffffff22"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#6eaef4ff"
	}
}

func renderCommandChecksumForOrder(order int) string {
	digits := "abcdef"
	index := order % len(digits)
	return "sha256:" + strings.Repeat(digits[index:index+1], 64)
}

// ---- report_accessibility_test.go ----

func TestValidateSurfaceAccessibilityMetadataTreeReport(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceAccessibilityRejectsMissingTree(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "accessibility_tree")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected accessibility report without accessibility_tree to fail")
	}
	if !strings.Contains(err.Error(), "accessibility_tree") {
		t.Fatalf("error = %v, want accessibility_tree diagnostic", err)
	}
}
func TestValidateSurfaceAccessibilityRejectsClaimsAndManualBookkeeping(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		want  string
	}{
		{name: "production", field: "production_claim", want: "production"},
		{name: "platform", field: "platform_host_integration", want: "platform_host_integration"},
		{name: "dom", field: "dom_aria_integration", want: "dom_aria_integration"},
		{name: "manual", field: "manual_bookkeeping", want: "manual_bookkeeping"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(
				t,
				func(report map[string]any) {
					a11y := report["accessibility_tree"].(map[string]any)
					a11y[tc.field] = true
				},
			)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility report with %s=true to fail", tc.field)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceReleaseAccessibilityPlatformBridgeReport(t *testing.T) {
	raw := validLinuxReleaseAccessibilitySurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceReleaseAccessibilityRejectsMissingBridgeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "metadata tree false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["metadata_tree"] = false
			},
			want: "metadata_tree",
		},
		{
			name: "platform export false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["platform_export"] = false
			},
			want: "platform_export",
		},
		{
			name: "linux probe false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["linux_platform_probe"] = false
			},
			want: "linux_platform_probe",
		},
		{
			name: "missing bridge evidence name",
			mutate: func(report map[string]any) {
				a11y := report["accessibility_tree"].(map[string]any)
				a11y["screen_reader_evidence"] = "full-screen-reader-support"
			},
			want: "screen_reader_evidence",
		},
		{
			name: "node only browser evidence",
			mutate: func(report map[string]any) {
				report["target"] = "wasm32-web"
				report["runtime"] = "surface-wasm32-web"
				report["host_evidence"].(map[string]any)["level"] = "wasm32-web-compiler-owned-loader"
			},
			want: "browser",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxReleaseAccessibilitySurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected release accessibility report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsNodeRelationshipAndOrderMismatches(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "unknown role",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["role"] = "slider"
			},
			want: "unknown role",
		},
		{
			name: "duplicate node id",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[7].(map[string]any)["id"] = 5
			},
			want: "duplicate",
		},
		{
			name: "unknown component",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["component_id"] = 99
			},
			want: "component_id",
		},
		{
			name: "bounds mismatch",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["bounds"] = rectMap(RectReport{X: 1, Y: 2, W: 3, H: 4})
			},
			want: "bounds",
		},
		{
			name: "missing label",
			mutate: func(report map[string]any) {
				a11y := report["accessibility_tree"].(map[string]any)
				a11y["relationships"] = []any{
					map[string]any{"kind": "label_for", "from": "NameLabel", "to": "NameTextBox"},
					map[string]any{"kind": "labelled_by", "from": "NameTextBox", "to": "NameLabel"},
				}
			},
			want: "EmailLabel",
		},
		{
			name: "focus order",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["focus_order"] = []any{
					"NameTextBox",
					"EmailTextBox",
					"SaveButton",
				}
			},
			want: "focus_order",
		},
		{
			name: "reading order",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["reading_order"] = []any{
					"TitleText",
					"NameTextBox",
					"NameLabel",
					"EmailLabel",
					"EmailTextBox",
					"SaveButton",
					"ResetButton",
					"StatusText",
				}
			},
			want: "reading_order",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility %s mismatch to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsSnapshotMismatches(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "two focused nodes",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["focused"] = true
				nodes[7].(map[string]any)["focused"] = true
			},
			want: "focused",
		},
		{
			name: "email value while wrong focus",
			mutate: func(report map[string]any) {
				for _, rawSnapshot := range report["accessibility_tree"].(map[string]any)["snapshots"].([]any) {
					snapshot := rawSnapshot.(map[string]any)
					if snapshot["name"] == "after_email_text" {
						snapshot["focused"] = "NameTextBox"
					}
				}
			},
			want: "after_email_text",
		},
		{
			name: "status unchanged after save",
			mutate: func(report map[string]any) {
				for _, rawSnapshot := range report["accessibility_tree"].(map[string]any)["snapshots"].([]any) {
					snapshot := rawSnapshot.(map[string]any)
					if snapshot["name"] == "after_save" {
						snapshot["status_value"] = "idle"
					}
				}
			},
			want: "after_save",
		},
		{
			name: "metadata checksum unchanged",
			mutate: func(report map[string]any) {
				snapshots := report["accessibility_tree"].(map[string]any)["snapshots"].([]any)
				snapshots[2].(map[string]any)["metadata_checksum"] =
					snapshots[1].(map[string]any)["metadata_checksum"]
			},
			want: "metadata_checksum",
		},
		{
			name: "bounds checksum unchanged",
			mutate: func(report map[string]any) {
				snapshots := report["accessibility_tree"].(map[string]any)["snapshots"].([]any)
				snapshots[len(
					snapshots,
				)-1].(map[string]any)["bounds_checksum"] = snapshots[len(
					snapshots,
				)-2].(map[string]any)["bounds_checksum"]
			},
			want: "bounds_checksum",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility %s mismatch to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsNodeOnlyBrowserAndLegacySidecarEvidence(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
		report["target"] = "wasm32-web"
		report["host"] = "node"
		report["host_evidence"] = map[string]any{
			"level":                        "wasm32-web-compiler-owned-loader",
			"backend":                      "node-surface-host",
			"framebuffer":                  true,
			"real_window":                  false,
			"native_input":                 false,
			"user_facing_platform_widgets": false,
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected accessibility Node-only browser evidence to fail")
	}
	if !strings.Contains(err.Error(), "browser") && !strings.Contains(err.Error(), "Node") {
		t.Fatalf("error = %v, want browser/Node diagnostic", err)
	}

	raw = validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
		artifacts := report["artifacts"].([]any)
		report["artifacts"] = append(
			artifacts,
			map[string]any{
				"kind":   "legacy-ui-sidecar",
				"path":   "/tmp/surface-artifacts/accessibility.ui.json",
				"sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
				"size":   1,
			},
		)
	})
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected accessibility legacy sidecar evidence to fail")
	}
	if !strings.Contains(err.Error(), ".ui.json") && !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("error = %v, want legacy sidecar diagnostic", err)
	}
}

func validHeadlessAccessibilityMetadataSurfaceReportJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessToolkitReuseSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base accessibility report: %v", err)
	}
	report["source"] = "examples/surface/toolkit/surface_accessibility_settings.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_accessibility_settings.tetr" +
				"a -o /tmp/surface-artifacts/surface-accessibility-settings"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-accessibility-settings",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":      "surface headless runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-accessibility-settings",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   90001,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   24000,
		},
	}
	report["components"] = []any{
		componentMap(
			"AccessibilitySettingsApp",
			"examples.surface.toolkit.surface_accessibility_settings.AccessibilitySettingsApp",
			"",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			map[string]string{
				"focused_id":         "5",
				"save_count":         "1",
				"reset_count":        "1",
				"status_code":        "2",
				"width":              "480",
				"height":             "320",
				"accessibility_role": "root",
			},
		),
		componentMap(
			"Panel",
			"lib.core.widgets.Panel",
			"AccessibilitySettingsApp",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			map[string]string{"padding": "12", "accessibility_role": "panel"},
		),
		componentMap(
			"Column",
			"lib.core.widgets.Column",
			"Panel",
			RectReport{X: 12, Y: 12, W: 456, H: 296},
			map[string]string{"child_count": "7", "accessibility_role": "column"},
		),
		componentMap(
			"TitleText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 20, W: 440, H: 24},
			map[string]string{"role": "text", "text_len": "8", "accessibility_role": "text"},
		),
		componentMap(
			"NameLabel",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 52, W: 440, H: 24},
			map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"},
		),
		componentMap(
			"NameTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 20, Y: 84, W: 440, H: 44},
			map[string]string{
				"focused":            "true",
				"buffer":             "",
				"text_len":           "0",
				"caret":              "0",
				"accessibility_role": "textbox",
			},
		),
		componentMap(
			"EmailLabel",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 136, W: 440, H: 24},
			map[string]string{"role": "label", "text_len": "5", "accessibility_role": "label"},
		),
		componentMap(
			"EmailTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 20, Y: 168, W: 440, H: 44},
			map[string]string{
				"focused":            "false",
				"buffer":             "",
				"text_len":           "0",
				"caret":              "0",
				"accessibility_role": "textbox",
			},
		),
		componentMap(
			"ButtonRow",
			"lib.core.widgets.Row",
			"Column",
			RectReport{X: 20, Y: 224, W: 440, H: 44},
			map[string]string{"child_count": "2", "accessibility_role": "row"},
		),
		componentMap(
			"SaveButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 20, Y: 224, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "save",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"ResetButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 164, Y: 224, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "reset",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"StatusText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 280, W: 440, H: 24},
			map[string]string{"role": "status", "status_code": "2", "accessibility_role": "status"},
		),
	}
	report["component_tree"] = accessibilityComponentTreeMap(
		"accessibility-metadata-tree-v1",
		"AccessibilitySettingsApp",
	)
	report["component_tree_api"] = accessibilityComponentTreeAPIMap()
	report["toolkit"] = accessibilityToolkitMap()
	report["accessibility_tree"] = accessibilityTreeMap()
	report["events"] = []any{
		eventMap(
			1,
			"mouse_up",
			"NameTextBox",
			[]any{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"},
			40,
			100,
			0,
			320,
			240,
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "-1",
				"NameTextBox.focused":                 "false",
			},
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "5",
				"NameTextBox.focused":                 "true",
			},
		),
		textEventMap(
			2,
			"NameTextBox",
			[]any{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"},
			3,
			"416461",
			320,
			240,
			map[string]string{
				"NameTextBox.buffer":  "",
				"NameTextBox.caret":   "0",
				"EmailTextBox.buffer": "",
			},
			map[string]string{
				"NameTextBox.buffer":  "Ada",
				"NameTextBox.caret":   "3",
				"EmailTextBox.buffer": "",
			},
		),
		keyEventMap(
			3,
			"AccessibilitySettingsApp",
			[]any{"AccessibilitySettingsApp"},
			9,
			320,
			240,
			map[string]string{"AccessibilitySettingsApp.focused_id": "5"},
			map[string]string{"AccessibilitySettingsApp.focused_id": "7"},
		),
		textEventMap(
			4,
			"EmailTextBox",
			[]any{"AccessibilitySettingsApp", "Panel", "Column", "EmailTextBox"},
			5,
			"7465747261",
			320,
			240,
			map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"},
			map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"},
		),
		keyEventMap(
			5,
			"AccessibilitySettingsApp",
			[]any{"AccessibilitySettingsApp"},
			9,
			320,
			240,
			map[string]string{"AccessibilitySettingsApp.focused_id": "7"},
			map[string]string{"AccessibilitySettingsApp.focused_id": "9"},
		),
		keyEventMap(
			6,
			"SaveButton",
			[]any{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"},
			32,
			320,
			240,
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "9",
				"AccessibilitySettingsApp.save_count": "0",
				"StatusText.status_code":              "0",
			},
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "9",
				"AccessibilitySettingsApp.save_count": "1",
				"StatusText.status_code":              "1",
			},
		),
		keyEventMap(
			7,
			"AccessibilitySettingsApp",
			[]any{"AccessibilitySettingsApp"},
			9,
			320,
			240,
			map[string]string{"AccessibilitySettingsApp.focused_id": "9"},
			map[string]string{"AccessibilitySettingsApp.focused_id": "10"},
		),
		keyEventMap(
			8,
			"ResetButton",
			[]any{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"},
			13,
			320,
			240,
			map[string]string{
				"AccessibilitySettingsApp.focused_id":  "10",
				"AccessibilitySettingsApp.reset_count": "0",
				"StatusText.status_code":               "1",
				"NameTextBox.buffer":                   "Ada",
				"EmailTextBox.buffer":                  "tetra",
			},
			map[string]string{
				"AccessibilitySettingsApp.focused_id":  "10",
				"AccessibilitySettingsApp.reset_count": "1",
				"StatusText.status_code":               "2",
				"NameTextBox.buffer":                   "",
				"EmailTextBox.buffer":                  "",
			},
		),
		keyEventMap(
			9,
			"AccessibilitySettingsApp",
			[]any{"AccessibilitySettingsApp"},
			9,
			320,
			240,
			map[string]string{"AccessibilitySettingsApp.focused_id": "10"},
			map[string]string{"AccessibilitySettingsApp.focused_id": "5"},
		),
		resizeEventMap(
			10,
			"AccessibilitySettingsApp",
			[]any{"AccessibilitySettingsApp"},
			480,
			320,
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "5",
				"NameTextBox.bounds.w":                "280",
				"EmailTextBox.bounds.w":               "280",
			},
			map[string]string{
				"AccessibilitySettingsApp.focused_id": "5",
				"NameTextBox.bounds.w":                "440",
				"EmailTextBox.bounds.w":               "440",
			},
		),
	}
	report["frames"] = []any{
		map[string]any{
			"order":     1,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"presented": true,
		},
		map[string]any{
			"order":     2,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"presented": true,
		},
		map[string]any{
			"order":     3,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"presented": true,
		},
		map[string]any{
			"order":     4,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"presented": true,
		},
		map[string]any{
			"order":     5,
			"width":     480,
			"height":    320,
			"stride":    1920,
			"checksum":  "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"presented": true,
		},
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "AccessibilitySettingsApp",
			"field":     "focused_id",
			"before":    "-1",
			"after":     "5",
			"cause":     "mouse_up",
		},
		map[string]any{
			"order":     2,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "Ada",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     3,
			"component": "AccessibilitySettingsApp",
			"field":     "focused_id",
			"before":    "5",
			"after":     "7",
			"cause":     "tab",
		},
		map[string]any{
			"order":     4,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "tetra",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     5,
			"component": "AccessibilitySettingsApp",
			"field":     "focused_id",
			"before":    "7",
			"after":     "9",
			"cause":     "tab",
		},
		map[string]any{
			"order":     6,
			"component": "AccessibilitySettingsApp",
			"field":     "save_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     7,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "0",
			"after":     "1",
			"cause":     "save",
		},
		map[string]any{
			"order":     8,
			"component": "AccessibilitySettingsApp",
			"field":     "focused_id",
			"before":    "9",
			"after":     "10",
			"cause":     "tab",
		},
		map[string]any{
			"order":     9,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "Ada",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     10,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "tetra",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     11,
			"component": "AccessibilitySettingsApp",
			"field":     "reset_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     12,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "1",
			"after":     "2",
			"cause":     "reset",
		},
		map[string]any{
			"order":     13,
			"component": "AccessibilitySettingsApp",
			"field":     "focused_id",
			"before":    "10",
			"after":     "5",
			"cause":     "tab",
		},
		map[string]any{
			"order":     14,
			"component": "AccessibilitySettingsApp",
			"field":     "NameTextBox.bounds.w",
			"before":    "280",
			"after":     "440",
			"cause":     "resize",
		},
		map[string]any{
			"order":     15,
			"component": "AccessibilitySettingsApp",
			"field":     "EmailTextBox.bounds.w",
			"before":    "280",
			"after":     "440",
			"cause":     "resize",
		},
	}
	report["cases"] = append(
		report["cases"].([]any),
		map[string]any{
			"name": "accessibility metadata tree schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata roles labels values states",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata component tree alignment",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata focus order alignment",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata reading order",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata snapshots update",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility metadata no DOM ARIA platform host claim",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal accessibility metadata report: %v", err)
	}
	return raw
}

func validLinuxReleaseAccessibilitySurfaceReportJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(
		validHeadlessAccessibilityMetadataSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode base release accessibility report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface/release/surface_release_accessibility.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/release/surface_release_accessibility.tetra" +
				" -o /tmp/surface-artifacts/surface-release-accessibility"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-release-accessibility",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":               "surface linux-x64 real-window probe",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-accessibility-real-window-probe",
			"ran":                true,
			"pass":               true,
			"exit_code":          42,
			"expected_exit_code": 42,
		},
		map[string]any{
			"name":      "surface linux-x64 runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux accessibility host bridge",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux accessibility platform probe",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-release-accessibility",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   90001,
		},
		map[string]any{
			"kind":   "linux-accessibility-host-bridge",
			"path":   "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size":   4096,
		},
		map[string]any{
			"kind":   "linux-accessibility-platform-probe",
			"path":   "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   4096,
		},
	}
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	report["host_evidence"].(map[string]any)["level"] = "linux-x64-real-window"
	report["host_evidence"].(map[string]any)["backend"] = "wayland-shm-rgba"
	report["host_evidence"].(map[string]any)["real_window"] = true
	report["host_evidence"].(map[string]any)["native_input"] = true
	report["host_evidence"].(map[string]any)["accessibility_bridge"] = true
	report["components"].([]any)[0].(map[string]any)["type"] = (("examples.surface.release.su" +
		"rface_release_accessibility.Acces") +
		"sibilitySettingsApp")
	tree := report["accessibility_tree"].(map[string]any)
	tree["accessibility_level"] = "platform-bridge-v1"
	tree["release_scope"] = "surface-v1-linux-web"
	tree["source"] = "examples/surface/release/surface_release_accessibility.tetra"
	tree["experimental"] = false
	tree["production_claim"] = true
	tree["platform_host_integration"] = true
	tree["metadata_tree"] = true
	tree["platform_export"] = true
	tree["platform_bridge"] = "linux_accessibility_host_bridge_v1"
	tree["linux_platform_probe"] = true
	tree["linux_probe_artifact"] = "/tmp/surface-artifacts/surface-linux-accessibility-probe.json"
	tree["browser_accessibility_snapshot"] = false
	tree["browser_accessibility_mirror"] = false
	tree["screen_reader_evidence"] = "linux_accessibility_host_bridge_v1"
	report["component_tree"].(map[string]any)["dynamic_level"] = "platform-bridge-v1"
	report["component_tree_api"].(map[string]any)["source"] = ("examples/surface/release/" +
		"surface_release_accessibility.tetra")
	toolkit := report["toolkit"].(map[string]any)
	toolkit["source"] = "examples/surface/release/surface_release_accessibility.tetra"
	toolkit["sources"] = append(
		toolkit["sources"].([]any),
		"examples/surface/release/surface_release_accessibility.tetra",
	)
	cases := []any{}
	for _, item := range report["cases"].([]any) {
		name, _ := item.(map[string]any)["name"].(string)
		if !strings.Contains(strings.ToLower(name), "headless") {
			cases = append(cases, item)
		}
	}
	report["cases"] = append(
		cases,
		map[string]any{
			"name": "linux-x64 real-window surface",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 native input event pump",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window resize event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window close event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility platform bridge v1 schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux accessibility host bridge export",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux accessibility platform probe roles labels values states bounds",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux accessibility probe focus order labels status resize",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility release honest screen reader evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal release accessibility report: %v", err)
	}
	return raw
}
func releaseWindowAccessibilityTreeMap() map[string]any {
	return map[string]any{
		"schema":                      "tetra.surface.accessibility-tree.v1",
		"accessibility_level":         "platform-bridge-v1",
		"release_scope":               "surface-v1-linux-web",
		"source":                      "examples/surface/release/surface_release_form.tetra",
		"module":                      "lib.core.accessibility",
		"widget_module":               "lib.core.widgets",
		"experimental":                false,
		"production_claim":            true,
		"platform_host_integration":   true,
		"dom_aria_integration":        false,
		"screen_reader_evidence":      "linux_accessibility_host_bridge_v1",
		"metadata_tree":               true,
		"platform_export":             true,
		"platform_bridge":             "linux_accessibility_host_bridge_v1",
		"linux_platform_probe":        true,
		"linux_probe_artifact":        "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
		"derived_from_component_tree": true,
		"uses_component_tree_api":     true,
		"uses_widget_toolkit":         true,
		"manual_bookkeeping":          false,
		"no_dom_ui":                   true,
		"no_user_js":                  true,
		"no_platform_widgets":         true,
		"no_legacy_sidecars":          true,
		"component_tree_schema":       "tetra.surface.component-tree.v1",
		"component_tree_api_schema":   "tetra.surface.component-tree-api.v1",
		"toolkit_schema":              "tetra.surface.toolkit.v1",
		"node_count":                  18,
		"focusable_count":             5,
		"label_count":                 2,
		"textbox_count":               2,
		"button_count":                2,
		"status_count":                1,
		"roles_present": []any{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"checkbox",
			"row",
			"button",
			"status",
		},
		"focus_order": []any{
			"NameTextBox",
			"EmailTextBox",
			"SubscribeCheckbox",
			"SaveButton",
			"ResetButton",
		},
		"reading_order": []any{
			"TitleText",
			"DescriptionText",
			"NameLabel",
			"NameTextBox",
			"EmailLabel",
			"EmailTextBox",
			"SubscribeCheckbox",
			"TermsText",
			"SaveButton",
			"ResetButton",
			"StatusText",
		},
		"nodes":         []any{},
		"relationships": []any{},
		"actions":       []any{},
		"snapshots":     []any{},
		"negative_guards": map[string]any{
			"no_borrowed_view_storage":       true,
			"component_id_alignment_checked": true,
			"bounds_alignment_checked":       true,
			"focus_order_alignment_checked":  true,
			"reading_order_checked":          true,
			"label_relationships_checked":    true,
			"state_updates_checked":          true,
			"artifact_scan_checked":          true,
		},
	}
}
func accessibilityComponentTreeMap(dynamicLevel string, rootName string) map[string]any {
	return map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": dynamicLevel,
		"root_id":       0,
		"node_count":    12,
		"focused_id":    5,
		"nodes": []any{
			treeNodeMap(
				0,
				rootName,
				"root",
				-1,
				0,
				1,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 480, H: 320},
			),
			treeNodeMap(
				1,
				"Panel",
				"panel",
				0,
				0,
				2,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 480, H: 320},
			),
			treeNodeMap(
				2,
				"Column",
				"column",
				1,
				0,
				3,
				7,
				false,
				RectReport{X: 12, Y: 12, W: 456, H: 296},
			),
			treeNodeMap(
				3,
				"TitleText",
				"text",
				2,
				0,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 20, W: 440, H: 24},
			),
			treeNodeMap(
				4,
				"NameLabel",
				"text",
				2,
				1,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 52, W: 440, H: 24},
			),
			treeNodeMap(
				5,
				"NameTextBox",
				"textbox",
				2,
				2,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 84, W: 440, H: 44},
			),
			treeNodeMap(
				6,
				"EmailLabel",
				"text",
				2,
				3,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 136, W: 440, H: 24},
			),
			treeNodeMap(
				7,
				"EmailTextBox",
				"textbox",
				2,
				4,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 168, W: 440, H: 44},
			),
			treeNodeMap(
				8,
				"ButtonRow",
				"row",
				2,
				5,
				9,
				2,
				false,
				RectReport{X: 20, Y: 224, W: 440, H: 44},
			),
			treeNodeMap(
				9,
				"SaveButton",
				"button",
				8,
				0,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 224, W: 132, H: 44},
			),
			treeNodeMap(
				10,
				"ResetButton",
				"button",
				8,
				1,
				-1,
				0,
				true,
				RectReport{X: 164, Y: 224, W: 132, H: 44},
			),
			treeNodeMap(
				11,
				"StatusText",
				"text",
				2,
				6,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 280, W: 440, H: 24},
			),
		},
		"layout_passes": []any{
			map[string]any{
				"component_id": 5,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 20, Y: 84, W: 280, H: 44}),
				"measured":     map[string]any{"w": 280, "h": 44},
			},
			map[string]any{
				"component_id": 7,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 20, Y: 168, W: 280, H: 44}),
				"measured":     map[string]any{"w": 280, "h": 44},
			},
			map[string]any{
				"component_id": 5,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 20, Y: 84, W: 440, H: 44}),
				"measured":     map[string]any{"w": 440, "h": 44},
			},
			map[string]any{
				"component_id": 7,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 20, Y: 168, W: 440, H: 44}),
				"measured":     map[string]any{"w": 440, "h": 44},
			},
			map[string]any{
				"component_id": 11,
				"pass":         "status-update",
				"bounds":       rectMap(RectReport{X: 20, Y: 280, W: 440, H: 24}),
				"measured":     map[string]any{"w": 440, "h": 24},
			},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		"focus_order": []any{5, 7, 9, 10},
		"dispatch_paths": []any{
			map[string]any{
				"event":     "click",
				"target_id": 5,
				"x":         40,
				"y":         100,
				"path":      []any{0, 1, 2, 5},
			},
			map[string]any{
				"event":     "click",
				"target_id": 7,
				"x":         40,
				"y":         184,
				"path":      []any{0, 1, 2, 7},
			},
			map[string]any{
				"event":     "key",
				"target_id": 9,
				"x":         40,
				"y":         240,
				"path":      []any{0, 1, 2, 8, 9},
			},
			map[string]any{
				"event":     "key",
				"target_id": 10,
				"x":         180,
				"y":         240,
				"path":      []any{0, 1, 2, 8, 10},
			},
		},
	}
}
func accessibilityComponentTreeAPIMap() map[string]any {
	return map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface/toolkit/surface_accessibility_settings.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          12,
			"capacity":            24,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{
				"helper":         "widgets.panel_content_rect",
				"target":         "Panel",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.row_layout",
				"target":         "ButtonRow",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "resize",
				"changed_bounds": true,
			},
		},
		"focus_helpers": []any{
			map[string]any{
				"helper": "tree_focus_next",
				"before": "NameTextBox",
				"after":  "EmailTextBox",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "EmailTextBox",
				"after":  "SaveButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "SaveButton",
				"after":  "ResetButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "ResetButton",
				"after":  "NameTextBox",
			},
		},
		"hit_tests": []any{
			map[string]any{
				"helper": "widgets.hit_test_accessibility_settings",
				"x":      40,
				"y":      100,
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 5},
			},
			map[string]any{
				"helper": "widgets.hit_test_accessibility_settings",
				"x":      40,
				"y":      184,
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 7},
			},
			map[string]any{
				"helper": "widgets.hit_test_accessibility_settings",
				"x":      180,
				"y":      240,
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 8, 10},
			},
		},
		"dispatch_paths": []any{
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 5},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 7},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "SaveButton",
				"path":   []any{0, 1, 2, 8, 9},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 8, 10},
			},
		},
	}
}
func accessibilityToolkitMap() map[string]any {
	return map[string]any{
		"schema":        "tetra.surface.toolkit.v1",
		"toolkit_level": "toolkit-reuse-v1",
		"reuse_level":   "multi-form-widget-reuse-v1",
		"source":        "examples/surface/toolkit/surface_accessibility_settings.tetra",
		"sources": []any{
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
			"examples/surface/toolkit/surface_accessibility_settings.tetra",
		},
		"module":                       "lib.core.widgets",
		"experimental":                 true,
		"production_claim":             false,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"example_count":                3,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("TitleText", "Text", 3, "text", true),
			toolkitWidgetMap("NameLabel", "Text", 4, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 5, "", true),
			toolkitWidgetMap("EmailLabel", "Text", 6, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 7, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 8, "", true),
			toolkitWidgetMap("SaveButton", "Button", 9, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 10, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 11, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:add_accessible_textbox",
			"lib/core/widgets/widgets.tetra:add_accessible_button",
			"lib/core/widgets/widgets.tetra:add_accessible_status",
		},
	}
}
func accessibilityTreeMap() map[string]any {
	return map[string]any{
		"schema":                      "tetra.surface.accessibility-tree.v1",
		"accessibility_level":         "metadata-tree-v1",
		"source":                      "examples/surface/toolkit/surface_accessibility_settings.tetra",
		"module":                      "lib.core.accessibility",
		"widget_module":               "lib.core.widgets",
		"experimental":                true,
		"production_claim":            false,
		"platform_host_integration":   false,
		"dom_aria_integration":        false,
		"screen_reader_evidence":      false,
		"derived_from_component_tree": true,
		"uses_component_tree_api":     true,
		"uses_widget_toolkit":         true,
		"manual_bookkeeping":          false,
		"no_dom_ui":                   true,
		"no_user_js":                  true,
		"no_platform_widgets":         true,
		"no_legacy_sidecars":          true,
		"component_tree_schema":       "tetra.surface.component-tree.v1",
		"component_tree_api_schema":   "tetra.surface.component-tree-api.v1",
		"toolkit_schema":              "tetra.surface.toolkit.v1",
		"node_count":                  12,
		"focusable_count":             4,
		"label_count":                 2,
		"textbox_count":               2,
		"button_count":                2,
		"status_count":                1,
		"roles_present": []any{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"row",
			"button",
			"status",
		},
		"nodes":         accessibilityNodes(),
		"relationships": accessibilityRelationships(),
		"focus_order": []any{
			"NameTextBox",
			"EmailTextBox",
			"SaveButton",
			"ResetButton",
		},
		"reading_order": []any{
			"TitleText",
			"NameLabel",
			"NameTextBox",
			"EmailLabel",
			"EmailTextBox",
			"SaveButton",
			"ResetButton",
			"StatusText",
		},
		"actions": []any{
			map[string]any{"target": "NameTextBox", "action": "edit", "semantic": "text-input"},
			map[string]any{"target": "EmailTextBox", "action": "edit", "semantic": "text-input"},
			map[string]any{"target": "SaveButton", "action": "press", "semantic": "save"},
			map[string]any{"target": "ResetButton", "action": "press", "semantic": "reset"},
		},
		"snapshots": accessibilitySnapshots(),
		"negative_guards": map[string]any{
			"no_borrowed_view_storage":       true,
			"component_id_alignment_checked": true,
			"bounds_alignment_checked":       true,
			"focus_order_alignment_checked":  true,
			"reading_order_checked":          true,
			"label_relationships_checked":    true,
			"state_updates_checked":          true,
			"artifact_scan_checked":          true,
		},
	}
}
func accessibilityNodes() []any {
	return []any{
		accessibilityNodeMap(
			0,
			0,
			-1,
			"AccessibilitySettingsApp",
			"root",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			false,
			false,
			false,
			"",
			"",
			"",
			0,
			nil,
			-1,
			0,
		),
		accessibilityNodeMap(
			1,
			1,
			0,
			"Panel",
			"panel",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			false,
			false,
			false,
			"",
			"",
			"",
			0,
			nil,
			-1,
			1,
		),
		accessibilityNodeMap(
			2,
			2,
			1,
			"Column",
			"column",
			RectReport{X: 12, Y: 12, W: 456, H: 296},
			false,
			false,
			false,
			"",
			"",
			"",
			0,
			nil,
			-1,
			2,
		),
		accessibilityNodeMap(
			3,
			3,
			2,
			"TitleText",
			"text",
			RectReport{X: 20, Y: 20, W: 440, H: 24},
			false,
			false,
			false,
			"",
			"",
			"title",
			0,
			nil,
			-1,
			3,
		),
		accessibilityNodeMap(
			4,
			4,
			2,
			"NameLabel",
			"label",
			RectReport{X: 20, Y: 52, W: 440, H: 24},
			false,
			false,
			false,
			"NameTextBox",
			"",
			"name",
			0,
			nil,
			-1,
			4,
		),
		accessibilityNodeMap(
			5,
			5,
			2,
			"NameTextBox",
			"textbox",
			RectReport{X: 20, Y: 84, W: 440, H: 44},
			true,
			true,
			true,
			"",
			"NameLabel",
			"name-present",
			0,
			[]any{"focus", "edit"},
			0,
			5,
		),
		accessibilityNodeMap(
			6,
			6,
			2,
			"EmailLabel",
			"label",
			RectReport{X: 20, Y: 136, W: 440, H: 24},
			false,
			false,
			false,
			"EmailTextBox",
			"",
			"email",
			0,
			nil,
			-1,
			6,
		),
		accessibilityNodeMap(
			7,
			7,
			2,
			"EmailTextBox",
			"textbox",
			RectReport{X: 20, Y: 168, W: 440, H: 44},
			true,
			true,
			false,
			"",
			"EmailLabel",
			"email-present",
			0,
			[]any{"focus", "edit"},
			1,
			7,
		),
		accessibilityNodeMap(
			8,
			8,
			2,
			"ButtonRow",
			"row",
			RectReport{X: 20, Y: 224, W: 440, H: 44},
			false,
			false,
			false,
			"",
			"",
			"",
			0,
			nil,
			-1,
			8,
		),
		accessibilityNodeMap(
			9,
			9,
			8,
			"SaveButton",
			"button",
			RectReport{X: 20, Y: 224, W: 132, H: 44},
			true,
			false,
			false,
			"",
			"",
			"save",
			0,
			[]any{"focus", "press", "save"},
			2,
			9,
		),
		accessibilityNodeMap(
			10,
			10,
			8,
			"ResetButton",
			"button",
			RectReport{X: 164, Y: 224, W: 132, H: 44},
			true,
			false,
			false,
			"",
			"",
			"reset",
			0,
			[]any{"focus", "press", "reset"},
			3,
			10,
		),
		accessibilityNodeMap(
			11,
			11,
			2,
			"StatusText",
			"status",
			RectReport{X: 20, Y: 280, W: 440, H: 24},
			false,
			false,
			false,
			"",
			"",
			"reset",
			0,
			nil,
			-1,
			11,
		),
	}
}

func accessibilityNodeMap(
	id int,
	componentID int,
	parentID int,
	name string,
	role string,
	bounds RectReport,
	focusable bool,
	editable bool,
	focused bool,
	labelFor string,
	labelledBy string,
	valueKind string,
	valueLen int,
	actions []any,
	focusIndex int,
	readingIndex int,
) map[string]any {
	value := map[string]any{
		"id":            id,
		"component_id":  componentID,
		"parent_id":     parentID,
		"name":          name,
		"role":          role,
		"bounds":        rectMap(bounds),
		"visible":       true,
		"enabled":       true,
		"focusable":     focusable,
		"focused":       focused,
		"editable":      editable,
		"readonly":      false,
		"required":      false,
		"pressed":       false,
		"invalid":       false,
		"focus_index":   focusIndex,
		"reading_index": readingIndex,
	}
	if labelFor != "" {
		value["label_for"] = labelFor
	}
	if labelledBy != "" {
		value["labelled_by"] = labelledBy
	}
	if valueKind != "" {
		value["value_kind"] = valueKind
	}
	if valueLen > 0 {
		value["value_len"] = valueLen
	}
	if actions != nil {
		value["actions"] = actions
	}
	return value
}
func accessibilityRelationships() []any {
	return []any{
		map[string]any{"kind": "label_for", "from": "NameLabel", "to": "NameTextBox"},
		map[string]any{"kind": "labelled_by", "from": "NameTextBox", "to": "NameLabel"},
		map[string]any{"kind": "label_for", "from": "EmailLabel", "to": "EmailTextBox"},
		map[string]any{"kind": "labelled_by", "from": "EmailTextBox", "to": "EmailLabel"},
	}
}
func accessibilitySnapshots() []any {
	return []any{
		accessibilitySnapshotMap(
			"initial",
			1,
			"",
			-1,
			-1,
			0,
			0,
			"idle",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"1111111111111111111111111111111111111111111111111111111111111111",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		),
		accessibilitySnapshotMap(
			"after_name_focus",
			2,
			"NameTextBox",
			5,
			5,
			0,
			0,
			"idle",
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"2222222222222222222222222222222222222222222222222222222222222222",
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		),
		accessibilitySnapshotMap(
			"after_name_text",
			3,
			"NameTextBox",
			5,
			5,
			3,
			0,
			"idle",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"3333333333333333333333333333333333333333333333333333333333333333",
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		),
		accessibilitySnapshotMap(
			"after_email_focus",
			4,
			"EmailTextBox",
			7,
			7,
			3,
			0,
			"idle",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"4444444444444444444444444444444444444444444444444444444444444444",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		),
		accessibilitySnapshotMap(
			"after_email_text",
			5,
			"EmailTextBox",
			7,
			7,
			3,
			5,
			"idle",
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"5555555555555555555555555555555555555555555555555555555555555555",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		),
		accessibilitySnapshotMap(
			"after_save",
			6,
			"SaveButton",
			9,
			9,
			3,
			5,
			"saved",
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"6666666666666666666666666666666666666666666666666666666666666666",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		),
		accessibilitySnapshotMap(
			"after_reset",
			7,
			"ResetButton",
			10,
			10,
			0,
			0,
			"reset",
			"9999999999999999999999999999999999999999999999999999999999999999",
			"7777777777777777777777777777777777777777777777777777777777777777",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		),
		accessibilitySnapshotMap(
			"after_resize",
			8,
			"NameTextBox",
			5,
			5,
			0,
			0,
			"reset",
			"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"8888888888888888888888888888888888888888888888888888888888888888",
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		),
	}
}

func accessibilitySnapshotMap(
	name string,
	generation int,
	focused string,
	focusedComponentID int,
	focusedAccessibilityNodeID int,
	nameLen int,
	emailLen int,
	status string,
	boundsChecksum string,
	metadataChecksum string,
	frameChecksum string,
) map[string]any {
	return map[string]any{
		"name":                          name,
		"generation":                    generation,
		"focused":                       focused,
		"focused_component_id":          focusedComponentID,
		"focused_accessibility_node_id": focusedAccessibilityNodeID,
		"name_value_len":                nameLen,
		"email_value_len":               emailLen,
		"status_value":                  status,
		"bounds_checksum":               boundsChecksum,
		"metadata_checksum":             metadataChecksum,
		"frame_checksum":                frameChecksum,
	}
}

// ---- report_app_model_test.go ----

func TestValidateSurfaceAppModelReport(t *testing.T) {
	raw := validHeadlessAppModelSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceAppModelRejectsIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "hidden app state",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["hidden_app_state"] = true
			},
			want: "hidden app state",
		},
		{
			name: "React runtime",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["react_runtime"] = true
			},
			want: "React",
		},
		{
			name: "DOM event model",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["dom_event_model"] = true
			},
			want: "DOM event model",
		},
		{
			name: "command without binding",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				bindings := app["event_bindings"].([]any)
				app["event_bindings"] = bindings[:len(bindings)-1]
			},
			want: "no explicit event binding",
		},
		{
			name: "async complete without start",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				tasks := app["async_tasks"].([]any)
				app["async_tasks"] = tasks[1:]
			},
			want: "completed without matching start",
		},
		{
			name: "async cancel mutates state",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				tasks := app["async_tasks"].([]any)
				cancel := tasks[2].(map[string]any)
				cancel["after_state"] = map[string]any{"pending_task": "0", "save_count": "2"}
			},
			want: "canceled command must not mutate app state",
		},
		{
			name: "navigation underflow drift",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				nav := app["navigation_transitions"].([]any)
				underflow := nav[2].(map[string]any)
				underflow["after_route"] = "settings"
			},
			want: "underflow rejection must preserve route and stack",
		},
		{
			name: "focus scope escape",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				focus := app["focus_scope_transitions"].([]any)
				modal := focus[1].(map[string]any)
				modal["escaped"] = true
			},
			want: "escaped active scope",
		},
		{
			name: "undo redo without history",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				history := app["undo_redo_transitions"].([]any)
				undo := history[1].(map[string]any)
				undo["matched_history_entry"] = false
			},
			want: "matched applied history entry",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAppModelSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected app_model %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessAppModelSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode base headless report: %v", err)
	}
	report["source"] = "examples/surface/toolkit/surface_app_model.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_app_model.tetra -o " +
				"/tmp/surface-artifacts/surface-app-model"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-app-model",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":      "surface headless runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-app-model",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   98234,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   26000,
		},
	}
	report["components"] = []any{
		componentMap(
			"AppModelApp",
			"examples.surface.toolkit.surface_app_model.AppModelApp",
			"",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			map[string]string{
				"route":              "settings",
				"focused":            "NameField",
				"save_count":         "1",
				"pending_task":       "0",
				"history_depth":      "1",
				"redo_depth":         "0",
				"accessibility_role": "none",
			},
		),
		componentMap(
			"NameField",
			"examples.surface.toolkit.surface_app_model.NameField",
			"AppModelApp",
			RectReport{X: 32, Y: 80, W: 240, H: 44},
			map[string]string{
				"focused":            "true",
				"buffer":             "Ada",
				"caret":              "3",
				"accessibility_role": "textbox",
			},
		),
		componentMap(
			"SaveButton",
			"examples.surface.toolkit.surface_app_model.SaveButton",
			"AppModelApp",
			RectReport{X: 32, Y: 144, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "save",
				"accessibility_role": "button",
			},
		),
	}
	report["events"] = []any{
		eventMap(
			1,
			"mouse_up",
			"NameField",
			[]any{"AppModelApp", "NameField"},
			48,
			96,
			0,
			480,
			320,
			map[string]string{"AppModelApp.focused": ""},
			map[string]string{"AppModelApp.focused": "NameField"},
		),
		textEventMap(
			2,
			"NameField",
			[]any{"AppModelApp", "NameField"},
			3,
			"416461",
			480,
			320,
			map[string]string{"NameField.buffer": ""},
			map[string]string{"NameField.buffer": "Ada"},
		),
		keyEventMap(
			3,
			"SaveButton",
			[]any{"AppModelApp", "SaveButton"},
			13,
			480,
			320,
			map[string]string{"AppModelApp.save_count": "0"},
			map[string]string{"AppModelApp.save_count": "1"},
		),
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "AppModelApp",
			"field":     "focused",
			"before":    "",
			"after":     "NameField",
			"cause":     "focus",
		},
		map[string]any{
			"order":     2,
			"component": "NameField",
			"field":     "buffer",
			"before":    "",
			"after":     "Ada",
			"cause":     "command.insert_text",
		},
		map[string]any{
			"order":     3,
			"component": "AppModelApp",
			"field":     "route",
			"before":    "home",
			"after":     "settings",
			"cause":     "command.navigate",
		},
		map[string]any{
			"order":     4,
			"component": "AppModelApp",
			"field":     "pending_task",
			"before":    "1",
			"after":     "0",
			"cause":     "command.async_complete",
		},
		map[string]any{
			"order":     5,
			"component": "AppModelApp",
			"field":     "history_depth",
			"before":    "0",
			"after":     "1",
			"cause":     "command.undoable",
		},
		map[string]any{
			"order":     6,
			"component": "AppModelApp",
			"field":     "save_count",
			"before":    "0",
			"after":     "1",
			"cause":     "command.save",
		},
	}
	report["app_model"] = map[string]any{
		"schema":                  "tetra.surface.app-model.v1",
		"app_model_level":         "explicit-command-reducer-v1",
		"release_scope":           "surface-v1-linux-web",
		"source":                  "examples/surface/toolkit/surface_app_model.tetra",
		"module":                  "lib.core.surface_app",
		"uses_component_tree_api": true,
		"caller_owned_state":      true,
		"explicit_event_bindings": true,
		"deterministic_reducer":   true,
		"hidden_app_state":        false,
		"react_runtime":           false,
		"electron_runtime":        false,
		"dom_runtime":             false,
		"dom_event_model":         false,
		"user_js":                 false,
		"platform_widgets":        false,
		"state_fields": []any{
			"route",
			"focused",
			"name_buffer",
			"save_count",
			"pending_task",
			"history_depth",
			"redo_depth",
		},
		"command_registry": []any{
			"focus.name",
			"text.insert",
			"nav.push.settings",
			"nav.back",
			"async.save.start",
			"async.save.complete",
			"async.save.cancel",
			"history.undo",
			"history.redo",
		},
		"event_bindings":          validAppModelEventBindings(),
		"command_dispatches":      validAppModelCommandDispatches(),
		"navigation_transitions":  validAppModelNavigationTransitions(),
		"focus_scope_transitions": validAppModelFocusScopeTransitions(),
		"async_tasks":             validAppModelAsyncTasks(),
		"undo_redo_transitions":   validAppModelUndoRedoTransitions(),
		"negative_guards":         validAppModelNegativeGuards(),
	}
	report["cases"] = append(
		report["cases"].([]any),
		map[string]any{
			"name": "app model explicit event-to-command binding",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model deterministic command reducer",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model navigation stack",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model focus scope modal trap",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model async completion cancellation boundary",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model undo redo history",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "app model no React hooks DOM event model hidden JS state",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal app model report: %v", err)
	}
	return raw
}
func validAppModelEventBindings() []any {
	return []any{
		map[string]any{
			"order":         1,
			"event_order":   1,
			"event_kind":    "mouse_up",
			"target":        "NameField",
			"dispatch_path": []any{"AppModelApp", "NameField"},
			"command":       "focus.name",
			"explicit":      true,
		},
		map[string]any{
			"order":         2,
			"event_order":   2,
			"event_kind":    "text_input",
			"target":        "NameField",
			"dispatch_path": []any{"AppModelApp", "NameField"},
			"command":       "text.insert",
			"explicit":      true,
		},
		map[string]any{
			"order":         3,
			"event_order":   3,
			"event_kind":    "key_down",
			"target":        "SaveButton",
			"dispatch_path": []any{"AppModelApp", "SaveButton"},
			"command":       "async.save.start",
			"explicit":      true,
		},
	}
}
func validAppModelCommandDispatches() []any {
	return []any{
		map[string]any{
			"order":        1,
			"event_order":  1,
			"command":      "focus.name",
			"kind":         "focus",
			"target":       "NameField",
			"handled":      true,
			"before_state": map[string]any{"focused": ""},
			"after_state":  map[string]any{"focused": "NameField"},
		},
		map[string]any{
			"order":         2,
			"event_order":   2,
			"command":       "text.insert",
			"kind":          "edit",
			"target":        "NameField",
			"handled":       true,
			"reversible":    true,
			"history_index": 1,
			"before_state":  map[string]any{"name_buffer": ""},
			"after_state":   map[string]any{"name_buffer": "Ada"},
		},
		map[string]any{
			"order":         3,
			"event_order":   3,
			"command":       "async.save.start",
			"kind":          "async_start",
			"target":        "SaveButton",
			"handled":       true,
			"async_task_id": "save-1",
			"before_state":  map[string]any{"pending_task": "0"},
			"after_state":   map[string]any{"pending_task": "1"},
		},
		map[string]any{
			"order":         4,
			"event_order":   0,
			"command":       "async.save.complete",
			"kind":          "async_complete",
			"target":        "AppModelApp",
			"handled":       true,
			"async_task_id": "save-1",
			"before_state":  map[string]any{"pending_task": "1", "save_count": "0"},
			"after_state":   map[string]any{"pending_task": "0", "save_count": "1"},
		},
	}
}
func validAppModelNavigationTransitions() []any {
	return []any{
		map[string]any{
			"order":              1,
			"command":            "nav.push.settings",
			"operation":          "push",
			"before_route":       "home",
			"after_route":        "settings",
			"stack_before":       []any{"home"},
			"stack_after":        []any{"home", "settings"},
			"underflow_rejected": false,
		},
		map[string]any{
			"order":              2,
			"command":            "nav.back",
			"operation":          "back",
			"before_route":       "settings",
			"after_route":        "home",
			"stack_before":       []any{"home", "settings"},
			"stack_after":        []any{"home"},
			"underflow_rejected": false,
		},
		map[string]any{
			"order":              3,
			"command":            "nav.back",
			"operation":          "back",
			"before_route":       "home",
			"after_route":        "home",
			"stack_before":       []any{"home"},
			"stack_after":        []any{"home"},
			"underflow_rejected": true,
		},
	}
}
func validAppModelFocusScopeTransitions() []any {
	return []any{
		map[string]any{
			"order":        1,
			"scope":        "main",
			"before_focus": "",
			"after_focus":  "NameField",
			"wrapped":      false,
			"modal_trap":   false,
			"escaped":      false,
		},
		map[string]any{
			"order":        2,
			"scope":        "dialog",
			"before_focus": "DialogCancel",
			"after_focus":  "DialogConfirm",
			"wrapped":      true,
			"modal_trap":   true,
			"escaped":      false,
		},
	}
}
func validAppModelAsyncTasks() []any {
	return []any{
		map[string]any{
			"id":               "save-1",
			"command":          "async.save.start",
			"operation":        "start",
			"status":           "pending",
			"before_state":     map[string]any{"pending_task": "0"},
			"after_state":      map[string]any{"pending_task": "1"},
			"completion_order": 0,
			"canceled":         false,
		},
		map[string]any{
			"id":               "save-1",
			"command":          "async.save.complete",
			"operation":        "complete",
			"status":           "completed",
			"before_state":     map[string]any{"pending_task": "1"},
			"after_state":      map[string]any{"pending_task": "0"},
			"completion_order": 4,
			"canceled":         false,
		},
		map[string]any{
			"id":               "save-2",
			"command":          "async.save.cancel",
			"operation":        "cancel",
			"status":           "canceled",
			"before_state":     map[string]any{"pending_task": "1", "save_count": "1"},
			"after_state":      map[string]any{"pending_task": "0", "save_count": "1"},
			"completion_order": 0,
			"canceled":         true,
		},
	}
}
func validAppModelUndoRedoTransitions() []any {
	return []any{
		map[string]any{
			"order":                 1,
			"command":               "text.insert",
			"history_index":         1,
			"operation":             "record",
			"before":                "",
			"after":                 "Ada",
			"matched_history_entry": true,
			"applied":               true,
		},
		map[string]any{
			"order":                 2,
			"command":               "history.undo",
			"history_index":         1,
			"operation":             "undo",
			"before":                "Ada",
			"after":                 "",
			"matched_history_entry": true,
			"applied":               true,
		},
		map[string]any{
			"order":                 3,
			"command":               "history.redo",
			"history_index":         1,
			"operation":             "redo",
			"before":                "",
			"after":                 "Ada",
			"matched_history_entry": true,
			"applied":               true,
		},
	}
}
func validAppModelNegativeGuards() map[string]any {
	return map[string]any{
		"no_hidden_app_state":              true,
		"no_react_hooks":                   true,
		"no_dom_event_model":               true,
		"no_user_js":                       true,
		"no_platform_widgets":              true,
		"async_cancel_no_mutation":         true,
		"navigation_underflow_rejected":    true,
		"focus_scope_escape_rejected":      true,
		"undo_redo_requires_history":       true,
		"command_without_binding_rejected": true,
	}
}

// ---- report_block_asset_accessibility_test.go ----

func TestValidateReportAcceptsBlockAssetEvidence(t *testing.T) {
	raw := validHeadlessBlockAssetSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsBlockAccessibilityEvidence(t *testing.T) {
	raw := validHeadlessBlockAccessibilitySurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockAccessibilityEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing name for actionable focusable block",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[1].Name = ""
			},
			want: "name",
		},
		{
			name: "label relationship mismatch",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[1].LabelledBy = "WrongLabel"
			},
			want: "label",
		},
		{
			name: "reading order not from block graph",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.ReadingOrder = []int{4, 3, 5}
			},
			want: "reading",
		},
		{
			name: "fake screen-reader claim",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.ScreenReaderEvidence = true
			},
			want: "screen_reader",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockAccessibilitySurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block accessibility %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteBlockAssetEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing asset hashes",
			mutate: func(report *Report) {
				report.BlockAssetManifest.Assets[1].SHA256 = ""
			},
			want: "sha256",
		},
		{
			name: "missing diagnostic",
			mutate: func(report *Report) {
				report.BlockAssetDiagnostics = nil
			},
			want: "diagnostic",
		},
		{
			name: "unbounded cache",
			mutate: func(report *Report) {
				report.BlockAssetCache.Bounded = false
				report.BlockAssetCache.BudgetBytes = 0
			},
			want: "cache",
		},
		{
			name: "network asset url",
			mutate: func(report *Report) {
				report.BlockAssetManifest.Assets[0].Path = "https://assets.example.test/tetra-ui.woff2"
				report.BlockAssetManifest.Assets[0].Local = false
				report.BlockAssetManifest.RemoteCount = 1
			},
			want: "network",
		},
		{
			name: "missing tint command",
			mutate: func(report *Report) {
				report.BlockAssetRenderCommands = removeBlockAssetRenderCommand(
					report.BlockAssetRenderCommands,
					"tint_icon",
				)
			},
			want: "tint",
		},
		{
			name: "marker only icon raster",
			mutate: func(report *Report) {
				report.BlockAssetRenderCommands[1].MarkerOnly = true
				report.BlockAssetRenderCommands[1].RasterHash = ""
			},
			want: "marker",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockAssetSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block asset %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessBlockAssetSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_render/surface_block_assets.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_assets.tetra -o " +
				"/tmp/surface-artifacts/surface-block-assets"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-assets",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-assets",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-assets",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockAssetComponentsForTest()
	report.BlockAssetQualityLevel = "deterministic-local-block-assets-v1"
	report.BlockAssetNetworkFetchAllowed = false
	report.BlockAssetManifest = blockAssetManifestForTest(report.Source)
	report.BlockAssetCache = blockAssetCacheForTest()
	report.BlockAssetDiagnostics = blockAssetDiagnosticsForTest()
	report.BlockAssetRenderCommands = blockAssetRenderCommandsForTest()
	report.Events = blockAssetEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "IconBlock",
			Field:     "tint",
			Before:    "#ffffffff",
			After:     "#60aef4ff",
			Cause:     "asset tint",
		},
		{
			Order:     2,
			Component: "ImageBlock",
			Field:     "scale",
			Before:    "1x",
			After:     "2x",
			Cause:     "asset scale",
		},
		{
			Order:     3,
			Component: "MissingAssetBlock",
			Field:     "fallback",
			Before:    "missing",
			After:     "fallback-raster",
			Cause:     "missing asset",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name: "block asset deterministic manifest hashes",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block asset local embedded only",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block asset image scale evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block asset missing fallback diagnostic",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing asset",
		},
		CaseReport{
			Name:          "block asset network url rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "network assets disabled",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block asset report: %v", err)
	}
	return raw
}
func validHeadlessBlockAccessibilitySurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_render/surface_block_accessibility.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_accessibility.te" +
				"tra -o /tmp/surface-artifacts/surface-block-accessibility"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-accessibility",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-accessibility",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-accessibility",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockAccessibilityComponentsForTest()
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.BlockAccessibilityTree = blockAccessibilityTreeForTest(report.Source)
	report.Events = blockAccessibilityEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "SubmitBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     2,
			Component: "ResetBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     3,
			Component: "BlockAccessibilityApp",
			Field:     "reading_order_checked",
			Before:    "false",
			After:     "true",
			Cause:     "block_graph",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name:          "block graph duplicate id rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "duplicate Block ID",
		},
		CaseReport{
			Name:          "block graph missing parent rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing parent",
		},
		CaseReport{
			Name:          "block graph cycle rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "cycle",
		},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block graph accessibility order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block accessibility tree derived from block graph",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block accessibility focusable actionable name required",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing accessible name",
		},
		CaseReport{
			Name:          "block accessibility label relationship mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "label relationship mismatch",
		},
		CaseReport{
			Name:          "block accessibility reading order graph mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "reading order mismatch",
		},
		CaseReport{
			Name:          "block accessibility screen-reader claim without platform proof rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "screen reader proof required",
		},
		CaseReport{
			Name: "block accessibility platform claim scoped metadata only",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block accessibility report: %v", err)
	}
	return raw
}
func blockAccessibilityComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{
			ID:        "BlockAccessibilityApp",
			Type:      "examples.surface.block_render.surface_block_accessibility.BlockAccessibilityApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":   "4",
				"a11y_quality": "block-derived-accessibility-metadata-v1",
			},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.LabelBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "4", "label_for": "4"},
		},
		{
			ID:        "SubmitBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.ActionBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "true", "action": "submit"},
		},
		{
			ID:        "ResetBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.ActionBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false", "action": "reset"},
		},
	}
}
func blockAccessibilityTreeForTest(source string) *BlockAccessibilityTreeReport {
	return &BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               3,
		FocusableCount:          2,
		RolesPresent:            []string{"text", "button"},
		FocusOrder:              []int{4, 5},
		ReadingOrder:            []int{3, 4, 5},
		Nodes: []BlockAccessibilityNodeReport{
			{
				ID:            3,
				BlockID:       3,
				ParentBlockID: 2,
				Name:          "LabelBlock",
				Role:          "text",
				Bounds:        RectReport{X: 24, Y: 24, W: 200, H: 24},
				Visible:       true,
				Enabled:       true,
				Focusable:     false,
				LabelFor:      "SubmitBlock",
				FocusIndex:    -1,
				ReadingIndex:  0,
			},
			{
				ID:            4,
				BlockID:       4,
				ParentBlockID: 2,
				Name:          "SubmitBlock",
				Role:          "button",
				Description:   "primary action",
				Bounds:        RectReport{X: 24, Y: 64, W: 120, H: 44},
				Visible:       true,
				Enabled:       true,
				Focusable:     true,
				Focused:       true,
				LabelledBy:    "LabelBlock",
				Actions:       []string{"focus", "press", "submit"},
				FocusIndex:    0,
				ReadingIndex:  1,
			},
			{
				ID:            5,
				BlockID:       5,
				ParentBlockID: 2,
				Name:          "ResetBlock",
				Role:          "button",
				Description:   "secondary action",
				Bounds:        RectReport{X: 152, Y: 64, W: 120, H: 44},
				Visible:       true,
				Enabled:       true,
				Focusable:     true,
				Actions:       []string{"focus", "press", "reset"},
				FocusIndex:    1,
				ReadingIndex:  2,
			},
		},
		Relationships: []AccessibilityRelationshipReport{
			{Kind: "label_for", From: "LabelBlock", To: "SubmitBlock"},
			{Kind: "labelled_by", From: "SubmitBlock", To: "LabelBlock"},
		},
		Actions: []AccessibilityActionReport{
			{Target: "SubmitBlock", Action: "press", Semantic: "submit"},
			{Target: "ResetBlock", Action: "press", Semantic: "reset"},
		},
		NegativeGuards: BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}
func blockAccessibilityEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               80,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"SubmitBlock.focused": "false"},
			AfterState:      map[string]string{"SubmitBlock.focused": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"SubmitBlock.value_len": "0"},
			AfterState:      map[string]string{"SubmitBlock.value_len": "2"},
		},
		{
			Order:           3,
			Kind:            "key_down",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Key:             13,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{3, 0, 0, 0, 13, 320, 200, 2, 0},
			BeforeState:     map[string]string{"SubmitBlock.pressed": "false"},
			AfterState:      map[string]string{"SubmitBlock.pressed": "true"},
		},
	}
}
func blockAssetComponentsForTest() []ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"asset",
	}
	return []ComponentReport{
		{
			ID:        "BlockAssetApp",
			Type:      "examples.surface.block_render.surface_block_assets.BlockAssetApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"asset_quality": "deterministic-local-block-assets-v1"},
		},
		{
			ID:        "IconBlock",
			Type:      "examples.surface.block_render.surface_block_assets.IconBlock",
			Parent:    "BlockAssetApp",
			Bounds:    RectReport{X: 24, Y: 36, W: 32, H: 32},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "icon-settings", "tint": "#60aef4ff"},
		},
		{
			ID:        "ImageBlock",
			Type:      "examples.surface.block_render.surface_block_assets.ImageBlock",
			Parent:    "BlockAssetApp",
			Bounds:    RectReport{X: 72, Y: 32, W: 96, H: 64},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "image-hero", "scale": "2x"},
		},
		{
			ID:        "MissingAssetBlock",
			Type:      "examples.surface.block_render.surface_block_assets.MissingAssetBlock",
			Parent:    "BlockAssetApp",
			Bounds:    RectReport{X: 24, Y: 112, W: 96, H: 32},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "missing-logo", "fallback": "fallback-raster"},
		},
	}
}
func blockAssetManifestForTest(source string) *BlockAssetManifestReport {
	return &BlockAssetManifestReport{
		Schema:        "tetra.surface.block-assets.v1",
		Source:        source,
		Quality:       "deterministic-local-block-assets-v1",
		HashAlgorithm: "sha256",
		ManifestHash:  "sha256:9999999999999999999999999999999999999999999999999999999999999999",
		LocalOnly:     true,
		FontCount:     1,
		IconCount:     1,
		ImageCount:    1,
		EmbeddedCount: 3,
		RemoteCount:   0,
		Assets: []BlockAssetReport{
			{
				ID:       "font-ui",
				Kind:     "font",
				Path:     "embedded://surface/font-ui",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:     2048,
				Family:   "Tetra UI",
				CacheKey: "font-ui",
			},
			{
				ID:       "icon-settings",
				Kind:     "icon",
				Path:     "embedded://surface/icon-settings",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:     256,
				Width:    16,
				Height:   16,
				CacheKey: "icon-settings",
			},
			{
				ID:       "image-hero",
				Kind:     "image",
				Path:     "embedded://surface/image-hero",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
				Size:     1024,
				Width:    48,
				Height:   32,
				CacheKey: "image-hero",
			},
		},
	}
}
func blockAssetCacheForTest() BlockAssetCacheReport {
	return BlockAssetCacheReport{
		ID:            "asset-cache",
		Strategy:      "bounded-lru",
		BudgetBytes:   65536,
		UsedBytes:     5376,
		EntryCount:    3,
		MaxEntries:    16,
		RepeatedLoads: 6,
		Eviction:      "lru",
		Bounded:       true,
	}
}
func blockAssetDiagnosticsForTest() []BlockAssetDiagnosticReport {
	return []BlockAssetDiagnosticReport{
		{
			Order:      1,
			AssetID:    "missing-logo",
			Kind:       "image",
			Code:       "missing_asset_fallback",
			Message:    "missing local asset resolved to fallback raster",
			FallbackID: "fallback-raster-image",
			Pass:       true,
		},
		{
			Order:       2,
			AssetID:     "https://assets.example.test/logo.png",
			Kind:        "image",
			Code:        "network_asset_rejected",
			Message:     "network assets are disabled for Surface Block v1",
			RejectedURL: "https://assets.example.test/logo.png",
			Pass:        true,
		},
	}
}
func blockAssetRenderCommandsForTest() []BlockAssetRenderCommandReport {
	return []BlockAssetRenderCommandReport{
		{
			Order:    1,
			Command:  "load_font",
			AssetID:  "font-ui",
			BlockID:  1,
			Rect:     RectReport{X: 0, Y: 0, W: 320, H: 200},
			Quality:  "font-manifest-metadata-v1",
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
		{
			Order:          2,
			Command:        "tint_icon",
			AssetID:        "icon-settings",
			BlockID:        2,
			Rect:           RectReport{X: 24, Y: 36, W: 32, H: 32},
			Tint:           "#60aef4ff",
			Scale:          1,
			Quality:        "icon-mask-raster-v1",
			RasterFormat:   "builtin-icon-mask-raster-v1",
			RasterHash:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			RasterWidth:    32,
			RasterHeight:   32,
			RasterCoverage: 341,
			Checksum:       "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			Order:    3,
			Command:  "scale_image",
			AssetID:  "image-hero",
			BlockID:  3,
			Rect:     RectReport{X: 72, Y: 32, W: 96, H: 64},
			Scale:    2,
			Quality:  "nearest-scale-v1",
			Checksum: "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			Order:    4,
			Command:  "fallback_missing",
			AssetID:  "missing-logo",
			BlockID:  4,
			Rect:     RectReport{X: 24, Y: 112, W: 96, H: 32},
			Quality:  "fallback-raster-v1",
			Checksum: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
	}
}
func blockAssetEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "IconBlock",
			DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
			Handled:         true,
			Pass:            true,
			X:               32,
			Y:               44,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 32, 44, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"IconBlock.tint": "#ffffffff"},
			AfterState:      map[string]string{"IconBlock.tint": "#60aef4ff"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "IconBlock",
			DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"IconBlock.label": ""},
			AfterState:      map[string]string{"IconBlock.label": "OK"},
		},
	}
}

func removeBlockAssetRenderCommand(
	commands []BlockAssetRenderCommandReport,
	command string,
) []BlockAssetRenderCommandReport {
	filtered := commands[:0]
	for _, current := range commands {
		if current.Command == command {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

// ---- report_block_graph_test.go ----

func TestValidateReportAcceptsBlockGraphEvidence(t *testing.T) {
	raw := validHeadlessBlockGraphSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockGraphEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "manual bookkeeping",
			mutate: func(report *Report) {
				report.BlockGraph.ManualBookkeeping = true
			},
			want: "manual_bookkeeping",
		},
		{
			name: "missing duplicate guard",
			mutate: func(report *Report) {
				report.BlockGraph.Invariants.DuplicateIDRejected = false
			},
			want: "duplicate_id",
		},
		{
			name: "missing parent",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[4].ParentID = 99
			},
			want: "parent_id",
		},
		{
			name: "cycle",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[1].ParentID = 5
			},
			want: "cycle",
		},
		{
			name: "child order",
			mutate: func(report *Report) {
				report.BlockGraph.ChildOrders[1].Children = []int{3, 5, 4}
			},
			want: "child_orders",
		},
		{
			name: "focus order",
			mutate: func(report *Report) {
				report.BlockGraph.FocusOrder = []int{5, 4}
			},
			want: "focus_order",
		},
		{
			name: "hit path",
			mutate: func(report *Report) {
				report.BlockGraph.HitTests[0].Path = []int{1, 5}
			},
			want: "hit_tests",
		},
		{
			name: "accessibility order",
			mutate: func(report *Report) {
				report.BlockGraph.AccessibilityOrder = []int{4, 5}
			},
			want: "accessibility_order",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockGraphSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block graph %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockPaintEvidence(t *testing.T) {
	raw := validHeadlessBlockPaintSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockPaintEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing fill",
			mutate: func(report *Report) {
				report.VisualFeatures = removeString(report.VisualFeatures, "fill")
			},
			want: "fill",
		},
		{
			name: "missing renderer report",
			mutate: func(report *Report) {
				report.Renderer = nil
			},
			want: "renderer",
		},
		{
			name: "missing border",
			mutate: func(report *Report) {
				report.PaintLayers = removePaintLayerKind(report.PaintLayers, "border")
			},
			want: "border",
		},
		{
			name: "missing image fill command",
			mutate: func(report *Report) {
				report.PaintCommands = removePaintCommand(report.PaintCommands, "image_fill")
			},
			want: "image_fill",
		},
		{
			name: "missing radius",
			mutate: func(report *Report) {
				for i := range report.PaintLayers {
					report.PaintLayers[i].Radius = 0
				}
				for i := range report.PaintCommands {
					report.PaintCommands[i].Radius = 0
				}
			},
			want: "radius",
		},
		{
			name: "missing shadow",
			mutate: func(report *Report) {
				report.PaintCommands = removePaintCommand(report.PaintCommands, "shadow")
			},
			want: "shadow",
		},
		{
			name: "missing outline",
			mutate: func(report *Report) {
				report.VisualFeatures = removeString(report.VisualFeatures, "outline")
			},
			want: "outline",
		},
		{
			name: "unsupported blur",
			mutate: func(report *Report) {
				report.PaintUnsupportedBlur = true
				report.VisualFeatures = append(report.VisualFeatures, "blur")
			},
			want: "unsupported blur",
		},
		{
			name: "gpu production claim",
			mutate: func(report *Report) {
				report.Renderer.GPUProductionClaim = true
			},
			want: "gpu production",
		},
		{
			name: "backdrop blur production claim",
			mutate: func(report *Report) {
				report.Renderer.BackdropBlurProductionClaim = true
			},
			want: "backdrop blur",
		},
		{
			name: "missing dirty rects",
			mutate: func(report *Report) {
				report.Renderer.DirtyRects = nil
			},
			want: "dirty_rects",
		},
		{
			name: "missing invalidations",
			mutate: func(report *Report) {
				report.Renderer.Invalidations = nil
			},
			want: "invalidations",
		},
		{
			name: "unbounded renderer cache",
			mutate: func(report *Report) {
				report.Renderer.CacheStats.Bounded = false
			},
			want: "renderer cache",
		},
		{
			name: "command order",
			mutate: func(report *Report) {
				commands := report.PaintCommands
				commands[0], commands[1] = commands[1], commands[0]
			},
			want: "paint_commands",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "paint frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockPaintSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block paint %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockTextEvidence(t *testing.T) {
	raw := validHeadlessBlockTextSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockTextEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing measurement",
			mutate: func(report *Report) {
				report.TextMeasurements = nil
			},
			want: "text_measurements",
		},
		{
			name: "wrap ellipsis mismatch",
			mutate: func(report *Report) {
				report.TextMeasurements[0].EllipsizedTextLen = report.TextMeasurements[0].TextLen
			},
			want: "ellipsis",
		},
		{
			name: "missing fallback chain",
			mutate: func(report *Report) {
				report.FontFallbacks = nil
			},
			want: "font_fallback",
		},
		{
			name: "unbounded glyph cache",
			mutate: func(report *Report) {
				report.GlyphCaches[0].Bounded = false
			},
			want: "glyph cache",
		},
		{
			name: "missing render command",
			mutate: func(report *Report) {
				report.TextRenderCommands = nil
			},
			want: "text render",
		},
		{
			name: "marker only glyph raster",
			mutate: func(report *Report) {
				report.TextRenderCommands[1].Quality = "deterministic-glyph-markers-v1"
				report.TextRenderCommands[1].MarkerOnly = true
				report.TextRenderCommands[1].RasterHash = ""
			},
			want: "marker",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "text frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockTextSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block text %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockLayoutEvidence(t *testing.T) {
	raw := validHeadlessBlockLayoutSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockLayoutEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing grid",
			mutate: func(report *Report) {
				report.LayoutPasses = removeBlockLayoutPassMode(report.LayoutPasses, "grid")
				report.LayoutFeatures = removeString(report.LayoutFeatures, "grid")
			},
			want: "grid",
		},
		{
			name: "missing dock",
			mutate: func(report *Report) {
				report.LayoutPasses = removeBlockLayoutPassMode(report.LayoutPasses, "dock")
				report.LayoutFeatures = removeString(report.LayoutFeatures, "dock")
			},
			want: "dock",
		},
		{
			name: "missing scroll",
			mutate: func(report *Report) {
				report.LayoutScrolls = nil
				report.LayoutFeatures = removeString(report.LayoutFeatures, "scroll")
			},
			want: "scroll",
		},
		{
			name: "missing resize",
			mutate: func(report *Report) {
				for i := range report.LayoutPasses {
					report.LayoutPasses[i].Resize = false
				}
				report.LayoutFeatures = removeString(report.LayoutFeatures, "resize")
			},
			want: "resize",
		},
		{
			name: "missing density stable rounding",
			mutate: func(report *Report) {
				report.LayoutFeatures = removeString(
					removeString(report.LayoutFeatures, "density"),
					"stable-rounding",
				)
			},
			want: "density",
		},
		{
			name: "missing aspect",
			mutate: func(report *Report) {
				report.LayoutFeatures = removeString(report.LayoutFeatures, "aspect")
			},
			want: "aspect",
		},
		{
			name: "unsupported css flexbox",
			mutate: func(report *Report) {
				report.LayoutUnsupportedCSSFlexbox = true
			},
			want: "CSS flexbox",
		},
		{
			name: "missing min max",
			mutate: func(report *Report) {
				report.LayoutConstraints[0].Min = SizeReport{}
				report.LayoutConstraints[0].Max = SizeReport{}
				report.LayoutFeatures = removeString(removeString(report.LayoutFeatures, "min"), "max")
			},
			want: "min",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "layout frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockLayoutSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block layout %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockEventFocusEvidence(t *testing.T) {
	raw := validHeadlessBlockEventSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockEventFocusEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing nested hit path",
			mutate: func(report *Report) {
				report.BlockEventRoutes[0].HitTestPath = []int{1, 4}
			},
			want: "hit_test_path",
		},
		{
			name: "disabled click delivered",
			mutate: func(report *Report) {
				report.BlockEventRoutes[1].Delivered = true
				report.BlockEventRoutes[1].Rejected = false
				report.BlockEventRoutes[1].RejectReason = ""
			},
			want: "disabled",
		},
		{
			name: "unfocused text accepted",
			mutate: func(report *Report) {
				report.BlockEventRoutes[2].Delivered = true
				report.BlockEventRoutes[2].Rejected = false
				report.BlockEventRoutes[2].FocusedID = 5
				report.BlockEventRoutes[2].RejectReason = ""
			},
			want: "unfocused",
		},
		{
			name: "missing tab wrap",
			mutate: func(report *Report) {
				report.BlockFocusTransitions[1].Wrapped = false
			},
			want: "wrap",
		},
		{
			name: "unsupported drag drop",
			mutate: func(report *Report) {
				report.BlockEventUnsupportedDragDrop = true
			},
			want: "drag",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockEventSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block event/focus %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockStateSelectorEvidence(t *testing.T) {
	raw := validHeadlessBlockStateSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockStateSelectorEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "wrong resolver order",
			mutate: func(report *Report) {
				report.BlockStateResolverOrder = []string{
					"base",
					"hover",
					"variant",
					"pressed",
					"focused",
					"selected",
					"disabled",
					"error",
					"loading",
					"motion",
				}
			},
			want: "resolver order",
		},
		{
			name: "missing hover selector",
			mutate: func(report *Report) {
				report.BlockStateSelectors = report.BlockStateSelectors[1:]
			},
			want: "hover",
		},
		{
			name: "pressed scale not applied",
			mutate: func(report *Report) {
				for i := range report.BlockStateResolutions {
					resolution := &report.BlockStateResolutions[i]
					if resolution.Selector == "pressed" &&
						resolution.Property == "layout.scale" {
						resolution.Applied = false
						resolution.After = resolution.Before
					}
				}
			},
			want: "pressed",
		},
		{
			name: "disabled transition missing",
			mutate: func(report *Report) {
				filtered := report.StateTransitions[:0]
				for _, transition := range report.StateTransitions {
					if transition.Component != "StateBlock" || transition.Field != "disabled" {
						filtered = append(filtered, transition)
					}
				}
				report.StateTransitions = filtered
			},
			want: "disabled",
		},
		{
			name: "unsupported css pseudo claim",
			mutate: func(report *Report) {
				report.BlockStateUnsupportedCSSPseudos = true
			},
			want: "css",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockStateSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block state %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockMotionEvidence(t *testing.T) {
	raw := validHeadlessBlockMotionSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockMotionEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing motion frames",
			mutate: func(report *Report) {
				report.MotionFrames = nil
			},
			want: "motion_frames",
		},
		{
			name: "reduced motion keeps scheduling",
			mutate: func(report *Report) {
				for i := range report.MotionFrames {
					if report.MotionFrames[i].ReducedMotion {
						report.MotionFrames[i].Scheduled = true
						report.MotionFrames[i].Settled = false
					}
				}
			},
			want: "reduced",
		},
		{
			name: "completion keeps scheduling",
			mutate: func(report *Report) {
				report.MotionFrames[len(report.MotionFrames)-2].Scheduled = true
				report.MotionFrames[len(report.MotionFrames)-2].Settled = false
			},
			want: "settled",
		},
		{
			name: "opacity not interpolated",
			mutate: func(report *Report) {
				for i := range report.MotionFrames {
					report.MotionFrames[i].Opacity = 80
				}
			},
			want: "opacity",
		},
		{
			name: "unsupported css animations",
			mutate: func(report *Report) {
				report.MotionUnsupportedCSSAnimations = true
			},
			want: "css",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockMotionSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block motion %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessBlockGraphSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(
		validHeadlessComponentTreeSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode component tree report: %v", err)
	}
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name:          "block graph duplicate id rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "duplicate Block ID",
		},
		CaseReport{
			Name:          "block graph missing parent rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing parent",
		},
		CaseReport{
			Name:          "block graph cycle rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "cycle",
		},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block graph accessibility order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block graph report: %v", err)
	}
	return raw
}
func validHeadlessBlockPaintSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockGraphSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode block graph report: %v", err)
	}
	report.PaintQualityLevel = "deterministic-software-paint-v1"
	report.PaintCacheBudgetBytes = 65536
	report.PaintUnsupportedBlur = false
	report.PaintLayers = blockPaintLayersForTest()
	report.PaintCommands = blockPaintCommandsForTest()
	report.VisualFeatures = []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
	report.Renderer = rendererReportForTest()
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block paint deterministic command order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block paint frame checksum changed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block paint unsupported blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "unsupported blur",
		},
		CaseReport{
			Name: "block renderer software rgba contract",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block compositor dirty rect invalidation cache",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block renderer opacity transform clipped child",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block renderer gpu production claim rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "gpu production",
		},
		CaseReport{
			Name:          "block renderer unsupported backdrop blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "backdrop blur",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block paint report: %v", err)
	}
	return raw
}
func validHeadlessBlockTextSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_render/surface_block_text.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_text.tetra -o " +
				"/tmp/surface-artifacts/surface-block-text"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-text",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-text",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-text",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockTextComponentsForTest()
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.Events = blockTextEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "BlockTextApp",
			Field:     "focused_id",
			Before:    "0",
			After:     "3",
			Cause:     "mouse_up",
		},
		{
			Order:     2,
			Component: "InputBlock",
			Field:     "buffer",
			Before:    "",
			After:     "OKd0a2",
			Cause:     "text_input",
		},
		{
			Order:     3,
			Component: "InputBlock",
			Field:     "caret",
			Before:    "0",
			After:     "4",
			Cause:     "text_input",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name: "block text deterministic measurement",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block text wrap ellipsis layout",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block text render command evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block text report: %v", err)
	}
	return raw
}
func validHeadlessBlockLayoutSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_core/surface_block_layout.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_layout.tetra -o " +
				"/tmp/surface-artifacts/surface-block-layout"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-layout",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-layout",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-layout",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockLayoutComponentsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{
		"stack",
		"row",
		"column",
		"absolute",
		"overlay",
		"grid",
		"dock",
		"scroll",
		"fit",
		"fill",
		"fixed",
		"min",
		"max",
		"aspect",
		"spacing",
		"alignment",
		"z-order",
		"clipping",
		"resize",
		"density",
		"stable-rounding",
	}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutDensity = blockLayoutDensityForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
		{
			Order:     3,
			Width:     480,
			Height:    260,
			Stride:    1920,
			Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "RowBlock",
			Field:     "pressed",
			Before:    "false",
			After:     "true",
			Cause:     "mouse_up",
		},
		{
			Order:     2,
			Component: "RowBlock",
			Field:     "text_len_seen",
			Before:    "0",
			After:     "2",
			Cause:     "text_input",
		},
		{
			Order:     3,
			Component: "BlockLayoutApp",
			Field:     "width",
			Before:    "320",
			After:     "480",
			Cause:     "resize",
		},
		{
			Order:     4,
			Component: "ScrollBlock",
			Field:     "scroll_y",
			Before:    "0",
			After:     "32",
			Cause:     "scroll",
		},
	}
	report.Events = []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "RowBlock",
			DispatchPath:    []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"},
			Handled:         true,
			Pass:            true,
			X:               32,
			Y:               32,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 32, 32, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"RowBlock.pressed": "false"},
			AfterState:      map[string]string{"RowBlock.pressed": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "RowBlock",
			DispatchPath:    []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"RowBlock.text_len_seen": "0"},
			AfterState:      map[string]string{"RowBlock.text_len_seen": "2"},
		},
		{
			Order:           3,
			Kind:            "resize",
			TargetComponent: "BlockLayoutApp",
			DispatchPath:    []string{"BlockLayoutApp"},
			Handled:         true,
			Pass:            true,
			Width:           480,
			Height:          260,
			TimestampMS:     2,
			BufferSlots:     []int{6, 0, 0, 0, 0, 480, 260, 2, 0},
			BeforeState:     map[string]string{"BlockLayoutApp.width": "320"},
			AfterState:      map[string]string{"BlockLayoutApp.width": "480"},
		},
		{
			Order:           4,
			Kind:            "scroll",
			TargetComponent: "ScrollBlock",
			DispatchPath:    []string{"BlockLayoutApp", "ScrollBlock"},
			Handled:         true,
			Pass:            true,
			X:               260,
			Y:               80,
			Width:           480,
			Height:          260,
			TimestampMS:     3,
			BufferSlots:     []int{7, 260, 80, 0, 0, 480, 260, 3, 0},
			BeforeState:     map[string]string{"ScrollBlock.scroll_y": "0"},
			AfterState:      map[string]string{"ScrollBlock.scroll_y": "32"},
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block layout fit fill fixed min max",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block layout grid dock overlay scroll",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block layout resize constraints",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block layout aspect density stable rounding",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block layout no css flexbox parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "CSS flexbox parity nonclaim",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block layout report: %v", err)
	}
	return raw
}
func validHeadlessBlockEventSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_core/surface_block_events.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_events.tetra -o " +
				"/tmp/surface-artifacts/surface-block-events"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-events",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-events",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-events",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockEventComponentsForTest()
	report.BlockGraph = blockEventGraphReportForTest(report.Source)
	report.BlockEventQualityLevel = "deterministic-block-events-v1"
	report.BlockEventPolicy = "capture-bubble-direct-v1"
	report.BlockEventUnsupportedDragDrop = false
	report.BlockEventKinds = []string{
		"pointer_enter",
		"pointer_leave",
		"pointer_move",
		"pointer_down",
		"pointer_up",
		"click",
		"double_click",
		"key",
		"text",
		"focus",
		"blur",
		"scroll",
		"resize",
		"close",
		"frame",
	}
	report.BlockEventRoutes = blockEventRoutesForTest()
	report.BlockFocusTransitions = blockFocusTransitionsForTest()
	report.Events = blockEventRuntimeEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "BlockEventApp",
			Field:     "focused_id",
			Before:    "0",
			After:     "4",
			Cause:     "click",
		},
		{
			Order:     2,
			Component: "InputBlock",
			Field:     "buffer",
			Before:    "",
			After:     "OK",
			Cause:     "text_input",
		},
		{
			Order:     3,
			Component: "BlockEventApp",
			Field:     "focused_id",
			Before:    "4",
			After:     "6",
			Cause:     "tab",
		},
		{
			Order:     4,
			Component: "BlockEventApp",
			Field:     "focused_id",
			Before:    "6",
			After:     "4",
			Cause:     "tab",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name:          "block graph duplicate id rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "duplicate Block ID",
		},
		CaseReport{
			Name:          "block graph missing parent rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing parent",
		},
		CaseReport{
			Name:          "block graph cycle rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "cycle",
		},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "block graph accessibility order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block event nested hit-test path",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block event capture bubble direct policy",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block event disabled click rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "disabled Block",
		},
		CaseReport{
			Name: "block event text input focused only",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block focus tab order graph-derived",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block event no complex drag claim",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "drag-and-drop nonclaim",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block event report: %v", err)
	}
	return raw
}
func validHeadlessBlockStateSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_core/surface_block_states.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_states.tetra -o " +
				"/tmp/surface-artifacts/surface-block-states"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-states",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-states",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-states",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockStateComponentsForTest()
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateResolverOrder = []string{
		"base",
		"variant",
		"hover",
		"pressed",
		"focused",
		"selected",
		"disabled",
		"error",
		"loading",
		"motion",
	}
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateSelectors = blockStateSelectorsForTest()
	report.BlockStateResolutions = blockStateResolutionsForTest()
	report.Events = blockStateEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "StateBlock",
			Field:     "selector_flags",
			Before:    "0",
			After:     "127",
			Cause:     "pointer/key/state input",
		},
		{
			Order:     2,
			Component: "StateBlock",
			Field:     "resolved_fill",
			Before:    "#20262eff",
			After:     "#2d9bf0ff",
			Cause:     "hover",
		},
		{
			Order:     3,
			Component: "StateBlock",
			Field:     "resolved_scale",
			Before:    "100",
			After:     "97",
			Cause:     "pressed",
		},
		{
			Order:     4,
			Component: "StateBlock",
			Field:     "disabled",
			Before:    "false",
			After:     "true",
			Cause:     "disabled selector",
		},
		{
			Order:     5,
			Component: "StateBlock",
			Field:     "error",
			Before:    "false",
			After:     "true",
			Cause:     "error selector",
		},
		{
			Order:     6,
			Component: "StateBlock",
			Field:     "loading",
			Before:    "false",
			After:     "true",
			Cause:     "loading selector",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name: "block state selector resolver order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block state hover fill override",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block state pressed scale override",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block state focus selected metadata",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block state disabled error loading overrides",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block state frame checksum changed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block state no css pseudo parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css pseudo nonclaim",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block state report: %v", err)
	}
	return raw
}
func validHeadlessBlockMotionSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_core/surface_block_motion.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_motion.tetra -o " +
				"/tmp/surface-artifacts/surface-block-motion"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-motion",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-motion",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-motion",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockMotionComponentsForTest()
	report.MotionQualityLevel = "deterministic-block-motion-v1"
	report.MotionClock = "deterministic-test-clock-v1"
	report.MotionFrameBudget = 4
	report.MotionUnsupportedCSSAnimations = false
	report.MotionFrames = blockMotionFramesForTest()
	report.Events = blockMotionEventsForTest()
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
		{
			Order:     3,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "MotionBlock",
			Field:     "opacity",
			Before:    "80",
			After:     "200",
			Cause:     "motion frame",
		},
		{
			Order:     2,
			Component: "MotionBlock",
			Field:     "color",
			Before:    "#203040ff",
			After:     "#60aef4ff",
			Cause:     "motion frame",
		},
		{
			Order:     3,
			Component: "MotionBlock",
			Field:     "scale",
			Before:    "100",
			After:     "108",
			Cause:     "motion frame",
		},
		{
			Order:     4,
			Component: "MotionBlock",
			Field:     "translate_x",
			Before:    "0",
			After:     "12",
			Cause:     "motion frame",
		},
		{
			Order:     5,
			Component: "MotionBlock",
			Field:     "motion_complete",
			Before:    "false",
			After:     "true",
			Cause:     "duration elapsed",
		},
		{
			Order:     6,
			Component: "MotionBlock",
			Field:     "reduced_motion",
			Before:    "false",
			After:     "true",
			Cause:     "accessibility setting",
		},
	}
	report.Cases = append(
		report.Cases,
		CaseReport{
			Name: "block motion deterministic test clock",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block motion opacity color transform frames",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block motion reduced motion instant settle",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block motion completion stops scheduling",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block motion frame checksum changed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block motion no css animation parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css animation nonclaim",
		},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block motion report: %v", err)
	}
	return raw
}
func blockTextComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{
			ID:        "BlockTextApp",
			Type:      "examples.surface.block_render.surface_block_text.BlockTextApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":   "3",
				"text_quality": "deterministic-fallback-text-v1",
			},
		},
		{
			ID:        "TextBlock",
			Type:      "examples.surface.block_render.surface_block_text.TextSurfaceBlock",
			Parent:    "BlockTextApp",
			Bounds:    RectReport{X: 12, Y: 10, W: 96, H: 40},
			Abilities: abilities,
			State:     map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"},
		},
		{
			ID:        "InputBlock",
			Type:      "examples.surface.block_render.surface_block_text.EditableTextBlock",
			Parent:    "BlockTextApp",
			Bounds:    RectReport{X: 12, Y: 58, W: 144, H: 36},
			Abilities: abilities,
			State:     map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"},
		},
	}
}
func blockEventComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{
			ID:        "BlockEventApp",
			Type:      "examples.surface.block_core.surface_block_events.BlockEventApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":    "4",
				"event_quality": "deterministic-block-events-v1",
			},
		},
		{
			ID:        "PanelBlock",
			Type:      "examples.surface.block_core.surface_block_events.PanelBlock",
			Parent:    "BlockEventApp",
			Bounds:    RectReport{X: 16, Y: 16, W: 288, H: 168},
			Abilities: abilities,
			State:     map[string]string{"role": "panel"},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_core.surface_block_events.LabelBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "10"},
		},
		{
			ID:        "InputBlock",
			Type:      "examples.surface.block_core.surface_block_events.InputBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"editable": "true", "focused": "true", "buffer": "OK"},
		},
		{
			ID:        "DisabledBlock",
			Type:      "examples.surface.block_core.surface_block_events.DisabledBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"disabled": "true"},
		},
		{
			ID:        "ActionBlock",
			Type:      "examples.surface.block_core.surface_block_events.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 24, Y: 120, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false"},
		},
	}
}
func blockStateComponentsForTest() []ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
	}
	return []ComponentReport{
		{
			ID:        "BlockStateApp",
			Type:      "examples.surface.block_core.surface_block_states.BlockStateApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"state_quality": "deterministic-block-state-resolver-v1"},
		},
		{
			ID:        "StateBlock",
			Type:      "examples.surface.block_core.surface_block_states.StateBlock",
			Parent:    "BlockStateApp",
			Bounds:    RectReport{X: 24, Y: 40, W: 168, H: 56},
			Abilities: abilities,
			State: map[string]string{
				"selector_flags": "127",
				"variant":        "2",
				"disabled":       "true",
				"error":          "true",
				"loading":        "true",
			},
		},
		{
			ID:        "StatusBlock",
			Type:      "examples.surface.block_core.surface_block_states.StatusBlock",
			Parent:    "BlockStateApp",
			Bounds:    RectReport{X: 24, Y: 112, W: 168, H: 32},
			Abilities: abilities,
			State:     map[string]string{"selected": "true", "focused": "true"},
		},
	}
}
func blockMotionComponentsForTest() []ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
		"motion",
	}
	return []ComponentReport{
		{
			ID:        "BlockMotionApp",
			Type:      "examples.surface.block_core.surface_block_motion.BlockMotionApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"motion_quality": "deterministic-block-motion-v1"},
		},
		{
			ID:        "MotionBlock",
			Type:      "examples.surface.block_core.surface_block_motion.MotionBlock",
			Parent:    "BlockMotionApp",
			Bounds:    RectReport{X: 24, Y: 44, W: 176, H: 64},
			Abilities: abilities,
			State: map[string]string{
				"opacity":     "200",
				"scale":       "108",
				"translate_x": "12",
				"complete":    "true",
			},
		},
	}
}
func blockLayoutComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{
			ID:        "BlockLayoutApp",
			Type:      "examples.surface.block_core.surface_block_layout.BlockLayoutApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"layout_quality": "deterministic-block-layout-v1"},
		},
		{
			ID:        "ColumnBlock",
			Type:      "examples.surface.block_core.surface_block_layout.ColumnBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    RectReport{X: 12, Y: 12, W: 296, H: 176},
			Abilities: abilities,
			State:     map[string]string{"mode": "column", "gap": "8"},
		},
		{
			ID:        "RowBlock",
			Type:      "examples.surface.block_core.surface_block_layout.RowBlock",
			Parent:    "ColumnBlock",
			Bounds:    RectReport{X: 24, Y: 24, W: 272, H: 48},
			Abilities: abilities,
			State:     map[string]string{"mode": "row", "gap": "6"},
		},
		{
			ID:        "GridBlock",
			Type:      "examples.surface.block_core.surface_block_layout.GridBlock",
			Parent:    "ColumnBlock",
			Bounds:    RectReport{X: 24, Y: 80, W: 132, H: 72},
			Abilities: abilities,
			State:     map[string]string{"mode": "grid", "columns": "2"},
		},
		{
			ID:        "DockBlock",
			Type:      "examples.surface.block_core.surface_block_layout.DockBlock",
			Parent:    "ColumnBlock",
			Bounds:    RectReport{X: 164, Y: 80, W: 132, H: 72},
			Abilities: abilities,
			State:     map[string]string{"mode": "dock"},
		},
		{
			ID:        "OverlayBlock",
			Type:      "examples.surface.block_core.surface_block_layout.OverlayBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    RectReport{X: 220, Y: 20, W: 72, H: 40},
			Abilities: abilities,
			State:     map[string]string{"mode": "overlay", "z": "4"},
		},
		{
			ID:        "ScrollBlock",
			Type:      "examples.surface.block_core.surface_block_layout.ScrollBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    RectReport{X: 236, Y: 72, W: 72, H: 80},
			Abilities: abilities,
			State:     map[string]string{"mode": "scroll", "clipped": "true"},
		},
	}
}
func blockMotionFramesForTest() []MotionFrameReport {
	return []MotionFrameReport{
		{
			Order:         1,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   0,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      0,
			Easing:        "linear",
			Opacity:       80,
			Color:         "#203040ff",
			TranslateX:    0,
			TranslateY:    0,
			Scale:         100,
			ReducedMotion: false,
			Scheduled:     true,
			Settled:       false,
			Checksum:      "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},
		{
			Order:         2,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   60,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      500,
			Easing:        "linear",
			Opacity:       140,
			Color:         "#407094ff",
			TranslateX:    6,
			TranslateY:    0,
			Scale:         104,
			ReducedMotion: false,
			Scheduled:     true,
			Settled:       false,
			Checksum:      "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		},
		{
			Order:         3,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   120,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      1000,
			Easing:        "linear",
			Opacity:       200,
			Color:         "#60aef4ff",
			TranslateX:    12,
			TranslateY:    0,
			Scale:         108,
			ReducedMotion: false,
			Scheduled:     false,
			Settled:       true,
			Checksum:      "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
		{
			Order:         4,
			BlockID:       2,
			Trigger:       "reduced_motion",
			TimestampMS:   121,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      1000,
			Easing:        "linear",
			Opacity:       200,
			Color:         "#60aef4ff",
			TranslateX:    12,
			TranslateY:    0,
			Scale:         108,
			ReducedMotion: true,
			Scheduled:     false,
			Settled:       true,
			Checksum:      "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		},
	}
}
func blockMotionEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "MotionBlock",
			DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
			Handled:         true,
			Pass:            true,
			X:               48,
			Y:               72,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 48, 72, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"MotionBlock.hovered": "false"},
			AfterState:      map[string]string{"MotionBlock.hovered": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "MotionBlock",
			DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"MotionBlock.buffer": ""},
			AfterState:      map[string]string{"MotionBlock.buffer": "OK"},
		},
	}
}
func blockStateSelectorsForTest() []BlockStateSelectorReport {
	return []BlockStateSelectorReport{
		{Order: 1, Name: "hover", BlockID: 2, Flags: 1, Hovered: true},
		{Order: 2, Name: "pressed", BlockID: 2, Flags: 2, Pressed: true},
		{Order: 3, Name: "focused", BlockID: 2, Flags: 4, Focused: true},
		{Order: 4, Name: "selected", BlockID: 2, Flags: 8, Selected: true},
		{Order: 5, Name: "disabled", BlockID: 2, Flags: 16, Disabled: true},
		{Order: 6, Name: "error", BlockID: 2, Flags: 32, Error: true},
		{Order: 7, Name: "loading", BlockID: 2, Flags: 64, Loading: true},
	}
}
func blockStateResolutionsForTest() []BlockStateResolutionReport {
	return []BlockStateResolutionReport{
		{
			Order:        1,
			BlockID:      2,
			Selector:     "hover",
			ResolverStep: "hover",
			Property:     "paint.fill",
			Before:       "#20262eff",
			After:        "#2d9bf0ff",
			Applied:      true,
		},
		{
			Order:        2,
			BlockID:      2,
			Selector:     "pressed",
			ResolverStep: "pressed",
			Property:     "layout.scale",
			Before:       "100",
			After:        "97",
			Applied:      true,
		},
		{
			Order:        3,
			BlockID:      2,
			Selector:     "focused",
			ResolverStep: "focused",
			Property:     "paint.outline",
			Before:       "none",
			After:        "focus-ring",
			Applied:      true,
		},
		{
			Order:        4,
			BlockID:      2,
			Selector:     "selected",
			ResolverStep: "selected",
			Property:     "accessibility.selected",
			Before:       "false",
			After:        "true",
			Applied:      true,
		},
		{
			Order:        5,
			BlockID:      2,
			Selector:     "disabled",
			ResolverStep: "disabled",
			Property:     "input.disabled",
			Before:       "false",
			After:        "true",
			Applied:      true,
		},
		{
			Order:        6,
			BlockID:      2,
			Selector:     "disabled",
			ResolverStep: "disabled",
			Property:     "text.opacity",
			Before:       "255",
			After:        "112",
			Applied:      true,
		},
		{
			Order:        7,
			BlockID:      2,
			Selector:     "error",
			ResolverStep: "error",
			Property:     "paint.outline_color",
			Before:       "#7aa2f7ff",
			After:        "#ff5f57ff",
			Applied:      true,
		},
		{
			Order:        8,
			BlockID:      2,
			Selector:     "loading",
			ResolverStep: "loading",
			Property:     "text.content",
			Before:       "Run",
			After:        "Loading",
			Applied:      true,
		},
		{
			Order:        9,
			BlockID:      2,
			Selector:     "motion",
			ResolverStep: "motion",
			Property:     "motion.transition_ms",
			Before:       "0",
			After:        "120",
			Applied:      true,
		},
	}
}
func blockStateEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 40, 56, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"StateBlock.selected": "false"},
			AfterState:      map[string]string{"StateBlock.selected": "true"},
		},
		{
			Order:           2,
			Kind:            "mouse_move",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			BufferSlots:     []int{2, 40, 56, 0, 0, 320, 200, 1, 0},
			BeforeState:     map[string]string{"StateBlock.hovered": "false"},
			AfterState:      map[string]string{"StateBlock.hovered": "true"},
		},
		{
			Order:           3,
			Kind:            "mouse_down",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{4, 40, 56, 1, 0, 320, 200, 2, 0},
			BeforeState:     map[string]string{"StateBlock.pressed": "false"},
			AfterState:      map[string]string{"StateBlock.pressed": "true"},
		},
		{
			Order:           4,
			Kind:            "text_input",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     3,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 3, 2},
			BeforeState:     map[string]string{"StateBlock.buffer": ""},
			AfterState:      map[string]string{"StateBlock.buffer": "OK"},
		},
		{
			Order:           5,
			Kind:            "key_down",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			Key:             9,
			Width:           320,
			Height:          200,
			TimestampMS:     4,
			BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 4, 0},
			BeforeState:     map[string]string{"StateBlock.focused": "false"},
			AfterState:      map[string]string{"StateBlock.focused": "true"},
		},
	}
}
func blockEventGraphReportForTest(source string) *BlockGraphReport {
	return &BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         6,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 6,
		Nodes: []BlockGraphNodeReport{
			{
				ID:                1,
				Name:              "BlockEventApp",
				ParentID:          -1,
				ChildIndex:        0,
				FirstChild:        2,
				ChildCount:        1,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            RectReport{X: 0, Y: 0, W: 320, H: 200},
			},
			{
				ID:                2,
				Name:              "PanelBlock",
				ParentID:          1,
				ChildIndex:        0,
				FirstChild:        3,
				ChildCount:        4,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            RectReport{X: 16, Y: 16, W: 288, H: 168},
			},
			{
				ID:                3,
				Name:              "LabelBlock",
				ParentID:          2,
				ChildIndex:        0,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "text",
				Bounds:            RectReport{X: 24, Y: 24, W: 200, H: 24},
			},
			{
				ID:                4,
				Name:              "InputBlock",
				ParentID:          2,
				ChildIndex:        1,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "textbox",
				Bounds:            RectReport{X: 24, Y: 64, W: 120, H: 44},
			},
			{
				ID:                5,
				Name:              "DisabledBlock",
				ParentID:          2,
				ChildIndex:        2,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "button",
				Bounds:            RectReport{X: 152, Y: 64, W: 120, H: 44},
			},
			{
				ID:                6,
				Name:              "ActionBlock",
				ParentID:          2,
				ChildIndex:        3,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            RectReport{X: 24, Y: 120, W: 120, H: 44},
			},
		},
		ChildOrders: []BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5, 6},
		DrawOrder:          []int{1, 2, 3, 4, 5, 6},
		FocusOrder:         []int{4, 6},
		AccessibilityOrder: []int{3, 4, 5, 6},
		HitTests: []BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 4,
				X:        40,
				Y:        80,
				Path:     []int{1, 2, 4},
			},
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 5,
				X:        180,
				Y:        80,
				Path:     []int{1, 2, 5},
			},
		},
		DispatchPaths: []BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 6, Path: []int{1, 2, 6}},
		},
	}
}
func blockLayoutConstraintsForTest() []BlockLayoutConstraintReport {
	return []BlockLayoutConstraintReport{
		{
			ID:           "root-column",
			BlockID:      1,
			Mode:         "column",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          SizeReport{W: 320, H: 200},
			Max:          SizeReport{W: 480, H: 260},
			Padding:      12,
			Margin:       0,
			Gap:          8,
			Align:        "stretch",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       0,
			Clip:         true,
		},
		{
			ID:           "row-fill",
			BlockID:      3,
			Mode:         "row",
			WidthPolicy:  "fill",
			HeightPolicy: "fixed",
			Min:          SizeReport{W: 160, H: 40},
			Max:          SizeReport{W: 296, H: 64},
			Padding:      6,
			Margin:       0,
			Gap:          6,
			Align:        "center",
			Justify:      "space-between",
			Overflow:     "visible",
			ZIndex:       1,
			Clip:         false,
		},
		{
			ID:           "text-fit",
			BlockID:      8,
			Mode:         "absolute",
			WidthPolicy:  "fit",
			HeightPolicy: "fit",
			Min:          SizeReport{W: 32, H: 18},
			Max:          SizeReport{W: 160, H: 40},
			Padding:      4,
			Margin:       0,
			Gap:          0,
			Align:        "start",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       2,
			Clip:         true,
		},
		{
			ID:           "overlay-z",
			BlockID:      6,
			Mode:         "overlay",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          SizeReport{W: 72, H: 40},
			Max:          SizeReport{W: 72, H: 40},
			Padding:      0,
			Margin:       0,
			Gap:          0,
			Align:        "end",
			Justify:      "start",
			Overflow:     "visible",
			ZIndex:       4,
			Clip:         false,
		},
		{
			ID:           "aspect-fit",
			BlockID:      9,
			Mode:         "absolute",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          SizeReport{W: 96, H: 54},
			Max:          SizeReport{W: 96, H: 54},
			Padding:      0,
			Margin:       0,
			Gap:          0,
			Align:        "start",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       2,
			Clip:         true,
		},
	}
}
func blockLayoutPassesForTest() []BlockLayoutPassReport {
	return []BlockLayoutPassReport{
		{
			Order:    1,
			ParentID: 0,
			BlockID:  1,
			Mode:     "column",
			Input:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176},
			Measured: SizeReport{W: 296, H: 176},
			Pass:     "initial",
			Resize:   false,
			Clip:     true,
			ZIndex:   0,
			Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},
		{
			Order:    2,
			ParentID: 1,
			BlockID:  2,
			Mode:     "stack",
			Input:    RectReport{X: 12, Y: 12, W: 296, H: 176},
			Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176},
			Measured: SizeReport{W: 296, H: 176},
			Pass:     "initial",
			Resize:   false,
			Clip:     false,
			ZIndex:   0,
			Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		},
		{
			Order:    3,
			ParentID: 2,
			BlockID:  3,
			Mode:     "row",
			Input:    RectReport{X: 24, Y: 24, W: 272, H: 48},
			Resolved: RectReport{X: 24, Y: 24, W: 272, H: 48},
			Measured: SizeReport{W: 272, H: 48},
			Pass:     "nested",
			Resize:   false,
			Clip:     false,
			ZIndex:   1,
			Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
		{
			Order:    4,
			ParentID: 2,
			BlockID:  4,
			Mode:     "grid",
			Input:    RectReport{X: 24, Y: 80, W: 132, H: 72},
			Resolved: RectReport{X: 24, Y: 80, W: 63, H: 34},
			Measured: SizeReport{W: 63, H: 34},
			Pass:     "grid-cell",
			Resize:   false,
			Clip:     true,
			ZIndex:   1,
			Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		},
		{
			Order:    5,
			ParentID: 2,
			BlockID:  5,
			Mode:     "dock",
			Input:    RectReport{X: 164, Y: 80, W: 132, H: 72},
			Resolved: RectReport{X: 164, Y: 80, W: 132, H: 24},
			Measured: SizeReport{W: 132, H: 24},
			Pass:     "dock-top",
			Resize:   false,
			Clip:     true,
			ZIndex:   1,
			Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555",
		},
		{
			Order:    6,
			ParentID: 1,
			BlockID:  6,
			Mode:     "overlay",
			Input:    RectReport{X: 220, Y: 20, W: 72, H: 40},
			Resolved: RectReport{X: 220, Y: 20, W: 72, H: 40},
			Measured: SizeReport{W: 72, H: 40},
			Pass:     "overlay-z-order",
			Resize:   false,
			Clip:     false,
			ZIndex:   4,
			Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666",
		},
		{
			Order:    7,
			ParentID: 1,
			BlockID:  7,
			Mode:     "scroll",
			Input:    RectReport{X: 236, Y: 72, W: 72, H: 80},
			Resolved: RectReport{X: 236, Y: 72, W: 72, H: 80},
			Measured: SizeReport{W: 72, H: 160},
			Pass:     "scroll-clip",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777",
		},
		{
			Order:    8,
			ParentID: 1,
			BlockID:  8,
			Mode:     "absolute",
			Input:    RectReport{X: 32, Y: 152, W: 0, H: 0},
			Resolved: RectReport{X: 32, Y: 152, W: 96, H: 20},
			Measured: SizeReport{W: 96, H: 20},
			Pass:     "fit-text",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888",
		},
		{
			Order:    9,
			ParentID: 1,
			BlockID:  9,
			Mode:     "absolute",
			Input:    RectReport{X: 164, Y: 152, W: 96, H: 64},
			Resolved: RectReport{X: 164, Y: 152, W: 96, H: 54},
			Measured: SizeReport{W: 96, H: 54},
			Pass:     "aspect-fit",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999",
		},
		{
			Order:    10,
			ParentID: 0,
			BlockID:  1,
			Mode:     "column",
			Input:    RectReport{X: 0, Y: 0, W: 480, H: 260},
			Resolved: RectReport{X: 12, Y: 12, W: 456, H: 236},
			Measured: SizeReport{W: 456, H: 236},
			Pass:     "resize",
			Resize:   true,
			Clip:     true,
			ZIndex:   0,
			Checksum: "sha256:1010101010101010101010101010101010101010101010101010101010101010",
		},
	}
}
func blockLayoutScrollsForTest() []BlockLayoutScrollReport {
	return []BlockLayoutScrollReport{
		{
			BlockID:    7,
			Viewport:   RectReport{X: 236, Y: 72, W: 72, H: 80},
			Content:    SizeReport{W: 72, H: 160},
			OffsetY:    32,
			MaxOffsetY: 80,
			Clipped:    true,
			Checksum:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
}
func blockLayoutDensityForTest() *BlockLayoutDensityReport {
	return &BlockLayoutDensityReport{
		TargetDPI:      144,
		ScaleMilli:     1500,
		BaseUnitPx:     4,
		RoundingPolicy: "integer-half-up-v1",
		PixelSnapping:  true,
		Breakpoints:    []string{"small", "medium", "large"},
		Checksum:       "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
}
func blockEventRoutesForTest() []BlockEventRouteReport {
	return []BlockEventRouteReport{
		{
			Order:          1,
			Kind:           "click",
			Policy:         "capture-bubble-direct-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			HitTestPath:    []int{1, 2, 4},
			DispatchPath:   []int{1, 2, 4},
			CapturePath:    []int{1, 2},
			BubblePath:     []int{2, 1},
			DirectTargetID: 4,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      4,
			Editable:       true,
			Disabled:       false,
		},
		{
			Order:          2,
			Kind:           "click",
			Policy:         "capture-bubble-direct-v1",
			TargetID:       5,
			TargetName:     "DisabledBlock",
			HitTestPath:    []int{1, 2, 5},
			DispatchPath:   []int{1, 2, 5},
			CapturePath:    []int{1, 2},
			BubblePath:     []int{2, 1},
			DirectTargetID: 5,
			Delivered:      false,
			Rejected:       true,
			RejectReason:   "disabled",
			FocusedID:      4,
			Editable:       false,
			Disabled:       true,
		},
		{
			Order:          3,
			Kind:           "text",
			Policy:         "direct-to-focused-editable-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			DispatchPath:   []int{1, 2, 4},
			DirectTargetID: 4,
			Delivered:      false,
			Rejected:       true,
			RejectReason:   "unfocused",
			FocusedID:      6,
			Editable:       true,
			TextLen:        2,
			TextBytesHex:   "4f4b",
		},
		{
			Order:          4,
			Kind:           "text",
			Policy:         "direct-to-focused-editable-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			DispatchPath:   []int{1, 2, 4},
			DirectTargetID: 4,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      4,
			Editable:       true,
			TextLen:        2,
			TextBytesHex:   "4f4b",
		},
		{
			Order:          5,
			Kind:           "key",
			Policy:         "direct-to-focused-v1",
			TargetID:       6,
			TargetName:     "ActionBlock",
			DispatchPath:   []int{1, 2, 6},
			DirectTargetID: 6,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      6,
			Editable:       false,
			Disabled:       false,
		},
	}
}
func blockFocusTransitionsForTest() []BlockFocusTransitionReport {
	return []BlockFocusTransitionReport{
		{
			Order:        1,
			Helper:       "tree_focus_next",
			BeforeID:     4,
			AfterID:      6,
			Direction:    "tab",
			GraphDerived: true,
			Wrapped:      false,
		},
		{
			Order:        2,
			Helper:       "tree_focus_next",
			BeforeID:     6,
			AfterID:      4,
			Direction:    "tab",
			GraphDerived: true,
			Wrapped:      true,
		},
	}
}
func blockTextEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockTextApp", "InputBlock"},
			Handled:         true,
			Pass:            true,
			X:               20,
			Y:               64,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 20, 64, 1, 0, 320, 200, 0, 0},
			BeforeState: map[string]string{
				"BlockTextApp.focused_id": "0",
				"InputBlock.focused":      "false",
			},
			AfterState: map[string]string{
				"BlockTextApp.focused_id": "3",
				"InputBlock.focused":      "true",
			},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockTextApp", "InputBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         4,
			TextBytesHex:    "4f4bd0a2",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 4},
			BeforeState:     map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"},
			AfterState: map[string]string{
				"InputBlock.buffer": "OKd0a2",
				"InputBlock.caret":  "4",
			},
		},
	}
}
func blockEventRuntimeEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockEventApp", "PanelBlock", "InputBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               80,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
			BeforeState: map[string]string{
				"BlockEventApp.focused_id": "0",
				"InputBlock.focused":       "false",
			},
			AfterState: map[string]string{
				"BlockEventApp.focused_id": "4",
				"InputBlock.focused":       "true",
			},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockEventApp", "PanelBlock", "InputBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"},
			AfterState:      map[string]string{"InputBlock.buffer": "OK", "InputBlock.caret": "2"},
		},
		{
			Order:           3,
			Kind:            "key_down",
			TargetComponent: "BlockEventApp",
			DispatchPath:    []string{"BlockEventApp"},
			Handled:         true,
			Pass:            true,
			Key:             9,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 2, 0},
			BeforeState:     map[string]string{"BlockEventApp.focused_id": "4"},
			AfterState:      map[string]string{"BlockEventApp.focused_id": "6"},
		},
		{
			Order:           4,
			Kind:            "key_down",
			TargetComponent: "BlockEventApp",
			DispatchPath:    []string{"BlockEventApp"},
			Handled:         true,
			Pass:            true,
			Key:             9,
			Width:           320,
			Height:          200,
			TimestampMS:     3,
			BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 3, 0},
			BeforeState:     map[string]string{"BlockEventApp.focused_id": "6"},
			AfterState:      map[string]string{"BlockEventApp.focused_id": "4"},
		},
	}
}
func blockPaintLayersForTest() []PaintLayerReport {
	return []PaintLayerReport{
		{ID: "root-fill", BlockID: 1, Kind: "fill", Color: "#346ecfff", Radius: 8, Opacity: 255},
		{
			ID:      "root-gradient",
			BlockID: 1,
			Kind:    "gradient",
			Color:   "#54b484ff",
			Radius:  8,
			Opacity: 255,
		},
		{ID: "root-image-fill", BlockID: 1, Kind: "image_fill", Radius: 8, Opacity: 255},
		{
			ID:      "root-border",
			BlockID: 1,
			Kind:    "border",
			Color:   "#e2eaf2ff",
			Radius:  8,
			Width:   1,
			Opacity: 255,
		},
		{ID: "root-radius-clip", BlockID: 1, Kind: "radius_clip", Radius: 8, Opacity: 255},
		{
			ID:      "root-shadow",
			BlockID: 1,
			Kind:    "shadow",
			Color:   "#00000058",
			Blur:    12,
			OffsetX: 0,
			OffsetY: 4,
			Opacity: 88,
		},
		{
			ID:      "root-overlay",
			BlockID: 1,
			Kind:    "overlay",
			Color:   "#10182066",
			Radius:  8,
			Opacity: 102,
		},
		{
			ID:      "root-outline",
			BlockID: 1,
			Kind:    "outline",
			Color:   "#f4cd5cff",
			Radius:  10,
			Width:   2,
			Opacity: 255,
		},
		{ID: "root-text", BlockID: 1, Kind: "text", Color: "#edf2f7ff", Opacity: 255},
		{ID: "root-icon", BlockID: 1, Kind: "icon", Color: "#f4cd5cff", Opacity: 255},
	}
}
func blockPaintCommandsForTest() []PaintCommandReport {
	return []PaintCommandReport{
		{
			Order:    1,
			Command:  "fill",
			LayerID:  "root-fill",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "rounded-rect-v1",
			Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},
		{
			Order:    2,
			Command:  "gradient",
			LayerID:  "root-gradient",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "two-stop-linear-v1",
			Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		},
		{
			Order:    3,
			Command:  "image_fill",
			LayerID:  "root-image-fill",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "bounded-asset-fill-v1",
			Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
		{
			Order:    4,
			Command:  "border",
			LayerID:  "root-border",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "rounded-outline-v1",
			Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		},
		{
			Order:    5,
			Command:  "radius_clip",
			LayerID:  "root-radius-clip",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "clip-stack-v1",
			Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555",
		},
		{
			Order:    6,
			Command:  "shadow",
			LayerID:  "root-shadow",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "box-shadow-approx-v1",
			Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666",
		},
		{
			Order:    7,
			Command:  "overlay",
			LayerID:  "root-overlay",
			BlockID:  1,
			Rect:     RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "alpha-over-v1",
			Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777",
		},
		{
			Order:    8,
			Command:  "outline",
			LayerID:  "root-outline",
			BlockID:  1,
			Rect:     RectReport{X: 10, Y: 8, W: 68, H: 32},
			Radius:   10,
			Quality:  "rounded-outline-v1",
			Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888",
		},
		{
			Order:    9,
			Command:  "text",
			LayerID:  "root-text",
			BlockID:  1,
			Rect:     RectReport{X: 20, Y: 16, W: 32, H: 12},
			Quality:  "glyph-run-v1",
			Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999",
		},
		{
			Order:    10,
			Command:  "icon",
			LayerID:  "root-icon",
			BlockID:  1,
			Rect:     RectReport{X: 56, Y: 16, W: 12, H: 12},
			Quality:  "monochrome-mask-raster-v1",
			Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
}
func rendererReportForTest() *RendererReport {
	return &RendererReport{
		Schema:                      "tetra.surface.renderer-feature.v1",
		Backend:                     "software-rgba",
		ColorFormat:                 "rgba8",
		QualityLevel:                "deterministic-software-renderer-v1",
		SoftwareRenderer:            true,
		GPUProductionClaim:          false,
		BlurProductionClaim:         false,
		BackdropBlurProductionClaim: false,
		CommandOrder: []string{
			"fill",
			"gradient",
			"image_fill",
			"border",
			"radius_clip",
			"shadow",
			"overlay",
			"outline",
			"text",
			"icon",
		},
		CompositorLayers:           rendererCompositorLayersForTest(),
		DirtyRects:                 rendererDirtyRectsForTest(),
		Invalidations:              rendererInvalidationsForTest(),
		CacheStats:                 rendererCacheStatsForTest(),
		UnsupportedEffectsRejected: []string{"gpu-production", "blur", "backdrop-blur"},
		DeterministicFrameChecksums: []string{
			"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
		ReferenceFrameArtifactSHA256: ("sha256:" +
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" +
			"dddd"),
	}
}
func rendererCompositorLayersForTest() []RendererCompositorLayerReport {
	return []RendererCompositorLayerReport{
		{
			ID:        "root",
			Kind:      "root",
			Order:     1,
			BlockID:   1,
			Rect:      RectReport{X: 0, Y: 0, W: 320, H: 200},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
		{
			ID:          "content",
			Kind:        "content",
			Order:       2,
			BlockID:     1,
			Rect:        RectReport{X: 12, Y: 10, W: 64, H: 28},
			Opacity:     255,
			Transform:   "translate(0,0)",
			ClipApplied: true,
			Clip:        RectReport{X: 12, Y: 10, W: 64, H: 28},
			Checksum:    "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			ID:        "overlay",
			Kind:      "overlay",
			Order:     3,
			BlockID:   1,
			Rect:      RectReport{X: 12, Y: 10, W: 64, H: 28},
			Opacity:   102,
			Transform: "translate(0,1)",
			Checksum:  "sha256:1212121212121212121212121212121212121212121212121212121212121212",
		},
		{
			ID:        "text",
			Kind:      "text",
			Order:     4,
			BlockID:   1,
			Rect:      RectReport{X: 20, Y: 16, W: 32, H: 12},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:1313131313131313131313131313131313131313131313131313131313131313",
		},
		{
			ID:        "icon",
			Kind:      "icon",
			Order:     5,
			BlockID:   1,
			Rect:      RectReport{X: 56, Y: 16, W: 12, H: 12},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:1414141414141414141414141414141414141414141414141414141414141414",
		},
	}
}
func rendererDirtyRectsForTest() []RendererDirtyRectReport {
	return []RendererDirtyRectReport{
		{
			FrameOrder: 1,
			Rect:       RectReport{X: 12, Y: 10, W: 68, H: 36},
			Reason:     "initial-paint",
			Checksum:   "sha256:1515151515151515151515151515151515151515151515151515151515151515",
		},
		{
			FrameOrder: 2,
			Rect:       RectReport{X: 12, Y: 10, W: 68, H: 36},
			Reason:     "state-change",
			Checksum:   "sha256:1616161616161616161616161616161616161616161616161616161616161616",
		},
	}
}
func rendererInvalidationsForTest() []RendererInvalidationReport {
	return []RendererInvalidationReport{
		{
			Order:     1,
			BlockID:   1,
			Reason:    "hovered changed",
			DirtyRect: RectReport{X: 12, Y: 10, W: 68, H: 36},
			Repaint:   true,
		},
		{
			Order:     2,
			BlockID:   1,
			Reason:    "text input changed",
			DirtyRect: RectReport{X: 20, Y: 16, W: 44, H: 12},
			Repaint:   true,
		},
	}
}
func rendererCacheStatsForTest() RendererCacheStatsReport {
	return RendererCacheStatsReport{
		ID:          "software-rgba-render-cache",
		Strategy:    "bounded-lru",
		BudgetBytes: 65536,
		UsedBytes:   20480,
		EntryCount:  10,
		Hits:        3,
		Misses:      2,
		Bounded:     true,
	}
}
func blockTextMeasurementsForTest() []TextMeasurementReport {
	return []TextMeasurementReport{
		{
			ID:                "title-measure",
			BlockID:           2,
			TextLen:           28,
			FontFamily:        "Tetra UI",
			FontWeight:        600,
			FontSize:          16,
			LineHeight:        20,
			MaxWidth:          96,
			Measured:          SizeReport{W: 96, H: 40},
			LineCount:         2,
			Wrap:              "word",
			Overflow:          "ellipsis",
			Ellipsis:          true,
			EllipsizedTextLen: 16,
			Align:             "start",
			Quality:           "deterministic-metrics-v1",
			Checksum:          "sha256:6666666666666666666666666666666666666666666666666666666666666666",
		},
		{
			ID:                "input-measure",
			BlockID:           6,
			TextLen:           4,
			FontFamily:        "Tetra UI",
			FontWeight:        400,
			FontSize:          14,
			LineHeight:        18,
			MaxWidth:          120,
			Measured:          SizeReport{W: 34, H: 18},
			LineCount:         1,
			Wrap:              "none",
			Overflow:          "clip",
			Ellipsis:          false,
			EllipsizedTextLen: 4,
			Align:             "start",
			Quality:           "deterministic-metrics-v1",
			Checksum:          "sha256:7777777777777777777777777777777777777777777777777777777777777777",
		},
	}
}
func blockFontFallbacksForTest() []FontFallbackReport {
	return []FontFallbackReport{
		{
			ID:              "ui-fallback",
			RequestedFamily: "Tetra UI",
			ResolvedFamily:  "Tetra UI Fallback",
			Chain:           []string{"Tetra UI", "Noto Sans", "monospace"},
			MissingGlyphs:   0,
			Coverage:        "ascii-plus-basic-utf8-smoke",
		},
	}
}
func blockGlyphCachesForTest() []GlyphCacheReport {
	return []GlyphCacheReport{
		{
			ID:          "glyph-cache",
			Strategy:    "bounded-lru",
			BudgetBytes: 65536,
			UsedBytes:   4096,
			EntryCount:  12,
			Eviction:    "lru",
			Bounded:     true,
		},
	}
}
func blockTextRenderCommandsForTest() []TextRenderCommandReport {
	return []TextRenderCommandReport{
		{
			Order:         1,
			Command:       "measure",
			MeasurementID: "title-measure",
			BlockID:       2,
			Rect:          RectReport{X: 12, Y: 10, W: 96, H: 40},
			Clip:          RectReport{X: 12, Y: 10, W: 96, H: 40},
			Color:         "#edf2f7ff",
			Opacity:       255,
			Quality:       "deterministic-text-measure-v1",
			Checksum:      "sha256:8888888888888888888888888888888888888888888888888888888888888888",
		},
		{
			Order:          2,
			Command:        "render_glyphs",
			MeasurementID:  "title-measure",
			BlockID:        2,
			Rect:           RectReport{X: 12, Y: 10, W: 96, H: 40},
			Clip:           RectReport{X: 12, Y: 10, W: 96, H: 40},
			Color:          "#edf2f7ff",
			Opacity:        255,
			Quality:        "deterministic-glyph-raster-v1",
			RasterFormat:   "builtin-5x7-alpha-mask-v1",
			RasterHash:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			RasterWidth:    96,
			RasterHeight:   40,
			RasterCoverage: 476,
			Checksum:       "sha256:9999999999999999999999999999999999999999999999999999999999999999",
		},
		{
			Order:         3,
			Command:       "render_caret",
			MeasurementID: "input-measure",
			BlockID:       6,
			Rect:          RectReport{X: 12, Y: 48, W: 120, H: 18},
			Clip:          RectReport{X: 12, Y: 48, W: 144, H: 36},
			Color:         "#f4cd5cff",
			Opacity:       255,
			Quality:       "deterministic-caret-v1",
			Checksum:      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
}
func removeString(values []string, value string) []string {
	filtered := values[:0]
	for _, current := range values {
		if current == value {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}
func removePaintLayerKind(layers []PaintLayerReport, kind string) []PaintLayerReport {
	filtered := layers[:0]
	for _, layer := range layers {
		if layer.Kind == kind {
			continue
		}
		filtered = append(filtered, layer)
	}
	return filtered
}
func removePaintCommand(commands []PaintCommandReport, command string) []PaintCommandReport {
	filtered := commands[:0]
	for _, current := range commands {
		if current.Command == command {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func removeBlockLayoutPassMode(
	passes []BlockLayoutPassReport,
	mode string,
) []BlockLayoutPassReport {
	filtered := passes[:0]
	for _, current := range passes {
		if normalizeLayoutToken(current.Mode) == normalizeLayoutToken(mode) {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}
func blockGraphReportForTest(source string) *BlockGraphReport {
	return &BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         5,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 5,
		Nodes: []BlockGraphNodeReport{
			{
				ID:                1,
				Name:              "RootBlock",
				ParentID:          -1,
				ChildIndex:        0,
				FirstChild:        2,
				ChildCount:        1,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            RectReport{X: 0, Y: 0, W: 320, H: 200},
			},
			{
				ID:                2,
				Name:              "PanelBlock",
				ParentID:          1,
				ChildIndex:        0,
				FirstChild:        3,
				ChildCount:        3,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            RectReport{X: 16, Y: 16, W: 288, H: 168},
			},
			{
				ID:                3,
				Name:              "LabelBlock",
				ParentID:          2,
				ChildIndex:        0,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "text",
				Bounds:            RectReport{X: 24, Y: 24, W: 200, H: 24},
			},
			{
				ID:                4,
				Name:              "SubmitBlock",
				ParentID:          2,
				ChildIndex:        1,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            RectReport{X: 24, Y: 64, W: 120, H: 44},
			},
			{
				ID:                5,
				Name:              "ResetBlock",
				ParentID:          2,
				ChildIndex:        2,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            RectReport{X: 152, Y: 64, W: 120, H: 44},
			},
		},
		ChildOrders: []BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5},
		DrawOrder:          []int{1, 2, 3, 4, 5},
		FocusOrder:         []int{4, 5},
		AccessibilityOrder: []int{3, 4, 5},
		HitTests: []BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 5,
				X:        180,
				Y:        80,
				Path:     []int{1, 2, 5},
			},
		},
		DispatchPaths: []BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
		},
	}
}

// ---- report_block_system_test.go ----

func TestValidateReportAcceptsHeadlessBlockSystemGoldenChecksumEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsMorphCapsuleEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed with Morph evidence: %v\n%s", err, raw)
	}
}
func TestValidateReportRequiresP08RecipeAuthoringSuite(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportJSON(t, func(morph map[string]any) {
		morph["recipes"] = []any{
			map[string]any{
				"name":                     "control.action@1",
				"output":                   "Block",
				"slots":                    []any{"label", "icon"},
				"inputs":                   []any{"text", "action", "variant"},
				"expands_to_block_graph":   true,
				"hidden_app_state":         false,
				"platform_widgets":         false,
				"core_primitive_promotion": false,
			},
			map[string]any{
				"name":                     "field.text@1",
				"output":                   "Block",
				"slots":                    []any{"label", "control"},
				"inputs":                   []any{"value", "on_text"},
				"expands_to_block_graph":   true,
				"hidden_app_state":         false,
				"platform_widgets":         false,
				"core_primitive_promotion": false,
			},
			map[string]any{
				"name":                     "command.item@1",
				"output":                   "Block",
				"slots":                    []any{"icon", "title", "subtitle"},
				"inputs":                   []any{"title", "subtitle", "icon", "selected"},
				"expands_to_block_graph":   true,
				"hidden_app_state":         false,
				"platform_widgets":         false,
				"core_primitive_promotion": false,
			},
			map[string]any{
				"name":                     "region.panel@1",
				"output":                   "Block",
				"slots":                    []any{"header", "body", "actions"},
				"inputs":                   []any{"title"},
				"expands_to_block_graph":   true,
				"hidden_app_state":         false,
				"platform_widgets":         false,
				"core_primitive_promotion": false,
			},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected P08 Morph recipe suite to reject the legacy four-recipe set")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "form.field@1") {
		t.Fatalf("error = %v, want missing form.field@1 diagnostic", err)
	}
}
func TestValidateReportRejectsIncompleteMorphRecipeApps(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing recipe apps",
			mutate: func(morph map[string]any) {
				delete(morph, "recipe_apps")
			},
			want: "recipe_apps",
		},
		{
			name: "hidden app state",
			mutate: func(morph map[string]any) {
				apps := morph["recipe_apps"].([]any)
				app := apps[0].(map[string]any)
				app["hidden_app_state"] = true
			},
			want: "hidden app state",
		},
		{
			name: "React runtime",
			mutate: func(morph map[string]any) {
				apps := morph["recipe_apps"].([]any)
				app := apps[0].(map[string]any)
				app["react_runtime"] = true
			},
			want: "React runtime",
		},
		{
			name: "Button primitive",
			mutate: func(morph map[string]any) {
				apps := morph["recipe_apps"].([]any)
				app := apps[0].(map[string]any)
				app["output_primitives"] = []any{"Block", "Button"}
			},
			want: "Button",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessMorphSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete Morph recipe app evidence to fail")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteMorphCapsuleEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing token graph",
			mutate: func(morph map[string]any) {
				delete(morph, "token_graph")
			},
			want: "token_graph",
		},
		{
			name: "fake core primitive recipe",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				recipe["output"] = "Button"
				recipe["core_primitive_promotion"] = true
			},
			want: "Button",
		},
		{
			name: "dirty production signoff",
			mutate: func(morph map[string]any) {
				morph["production_claim"] = true
				morph["git_dirty"] = true
			},
			want: "dirty checkout",
		},
		{
			name: "missing recipe expansion",
			mutate: func(morph map[string]any) {
				morph["recipe_expansions"] = []any{}
			},
			want: "recipe_expansions",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessMorphSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete Morph evidence to fail")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsLinuxX64RealWindowBlockSystemEvidence(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsWASM32WebBrowserCanvasBlockSystemEvidence(t *testing.T) {
	raw := validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsLinuxX64BlockSystemHeadlessPromotion(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Target = "headless"
		report.Runtime = "surface-headless"
		report.HostEvidence = HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 Block system report promoted from headless evidence to fail")
	}
	for _, want := range []string{"linux-x64", "real-window"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsLinuxX64BlockSystemMissingRealWindowPresentation(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Frames = nil
		report.BlockSystem.Frames[0].Order = 2
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf(
			"expected linux-x64 Block system report without real-window frame presentation to fail",
		)
	}
	for _, want := range []string{"real-window", "frame"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebBlockSystemFakeBrowserClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "node-only browser promotion",
			mutate: func(report *Report) {
				report.HostEvidence = HostEvidenceReport{
					Level:       "wasm32-web-compiler-owned-loader",
					Backend:     "node-surface-host",
					Framebuffer: true,
				}
			},
			want: "browser-canvas",
		},
		{
			name: "missing browser canvas RGBA readback",
			mutate: func(report *Report) {
				report.Frames = report.Frames[:1]
				report.BlockSystem.Frames = report.BlockSystem.Frames[:1]
				report.BlockSystem.FrameCount = 1
				filtered := report.Cases[:0]
				for _, tc := range report.Cases {
					if tc.Name == "wasm32-web browser canvas RGBA readback" {
						continue
					}
					filtered = append(filtered, tc)
				}
				report.Cases = filtered
			},
			want: "RGBA readback",
		},
		{
			name: "user JS artifact",
			mutate: func(report *Report) {
				report.Artifacts = append(report.Artifacts, ArtifactReport{
					Kind:   "user-js",
					Path:   "/tmp/surface-artifacts/surface-block-system.user.js",
					SHA256: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					Size:   128,
				})
				report.ArtifactScan.FilesChecked++
			},
			want: "user JS",
		},
		{
			name: "DOM UI artifact",
			mutate: func(report *Report) {
				report.Artifacts = append(report.Artifacts, ArtifactReport{
					Kind:   "dom-ui",
					Path:   "/tmp/surface-artifacts/surface-block-system.dom.html",
					SHA256: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
					Size:   256,
				})
				report.ArtifactScan.FilesChecked++
			},
			want: "DOM UI",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected wasm32-web browser-canvas Block system fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteHeadlessBlockSystemGoldenChecksumEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing frame checksum",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[0].Checksum = ""
			},
			want: "checksum",
		},
		{
			name: "nondeterministic repeat checksum",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[1].RepeatChecksum = ("sha256:" +
					"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" +
					"ffff")
			},
			want: "nondeterministic",
		},
		{
			name: "missing paint evidence",
			mutate: func(report *Report) {
				report.PaintLayers = nil
				report.PaintCommands = nil
				report.BlockSystem.Frames[0].PaintEvidence = false
			},
			want: "paint",
		},
		{
			name: "missing layout evidence",
			mutate: func(report *Report) {
				report.LayoutPasses = nil
				report.LayoutConstraints = nil
				report.BlockSystem.Frames[0].LayoutEvidence = false
			},
			want: "layout",
		},
		{
			name: "missing accessibility evidence",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree = nil
				report.BlockSystem.Frames[0].AccessibilityEvidence = false
			},
			want: "accessibility",
		},
		{
			name: "golden mismatch",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[0].GoldenChecksum = ("sha256:" +
					"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" +
					"eeee")
			},
			want: "golden",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected headless Block system %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteBlockSystemReadinessEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing text measurement",
			mutate: func(report *Report) {
				report.TextMeasurements = nil
				report.FontFallbacks = nil
				report.GlyphCaches = nil
				report.TextRenderCommands = nil
				report.TextQualityLevel = ""
				report.TextCacheBudgetBytes = 0
			},
			want: "text",
		},
		{
			name: "missing state selector",
			mutate: func(report *Report) {
				report.BlockStateSelectors = nil
				report.BlockStateResolutions = nil
				report.BlockStateResolverOrder = nil
				report.BlockStateQualityLevel = ""
			},
			want: "state",
		},
		{
			name: "missing motion frames",
			mutate: func(report *Report) {
				report.MotionFrames = nil
				report.MotionQualityLevel = ""
				report.MotionClock = ""
				report.MotionFrameBudget = 0
			},
			want: "motion",
		},
		{
			name: "missing asset cache",
			mutate: func(report *Report) {
				report.BlockAssetManifest = nil
				report.BlockAssetQualityLevel = ""
				report.BlockAssetCache = BlockAssetCacheReport{}
				report.BlockAssetDiagnostics = nil
				report.BlockAssetRenderCommands = nil
			},
			want: "asset",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected Block system %s to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteBlockSystemMemoryBudget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing memory budget",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = nil
			},
			want: "block_system memory_budget is required",
		},
		{
			name: "unbounded caches",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.BoundedCaches = false
			},
			want: "bounded_caches",
		},
		{
			name: "mismatched framebuffer total",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.TotalFramebufferBytes++
			},
			want: "total_framebuffer_bytes",
		},
		{
			name: "broad electron claim",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.PerformanceClaim = "faster than " + "Electron"
			},
			want: "Electron",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete Block memory budget report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportAcceptsBlockSystemMemoryBudgetEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
	})
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed with Block memory budget evidence: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsFakeBlockCorePrimitiveClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "Button component type",
			mutate: func(report *Report) {
				report.Components[3].Type = "examples.surface.block_core.surface_block_system.Button"
			},
			want: "Button",
		},
		{
			name: "Card block graph node",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[1].Name = "Card"
			},
			want: "Card",
		},
		{
			name: "TextField accessibility node",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[0].Name = "TextField"
			},
			want: "TextField",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected fake Block core primitive claim %s to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface/block_core/surface_block_system.tetra"
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-system",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-system",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   409,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   2,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Components = blockSystemComponentsForTest()
	report.Components = append(
		report.Components,
		retargetBlockSystemComponentsForTest(blockTextComponentsForTest())...)
	report.Components = append(
		report.Components,
		retargetBlockSystemComponentsForTest(blockStateComponentsForTest())...)
	report.Components = append(
		report.Components,
		retargetBlockSystemComponentsForTest(blockMotionComponentsForTest())...)
	report.Components = append(
		report.Components,
		retargetBlockSystemComponentsForTest(blockAssetComponentsForTest())...)
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.PaintQualityLevel = "deterministic-software-paint-v1"
	report.PaintCacheBudgetBytes = 65536
	report.PaintUnsupportedBlur = false
	report.PaintLayers = blockPaintLayersForTest()
	report.PaintCommands = blockPaintCommandsForTest()
	report.VisualFeatures = []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
	report.Renderer = rendererReportForTest()
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{
		"stack",
		"row",
		"column",
		"absolute",
		"overlay",
		"grid",
		"dock",
		"scroll",
		"fit",
		"fill",
		"fixed",
		"min",
		"max",
		"aspect",
		"spacing",
		"alignment",
		"z-order",
		"clipping",
		"resize",
		"density",
		"stable-rounding",
	}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutDensity = blockLayoutDensityForTest()
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateResolverOrder = []string{
		"base",
		"variant",
		"hover",
		"pressed",
		"focused",
		"selected",
		"disabled",
		"error",
		"loading",
		"motion",
	}
	report.BlockStateSelectors = blockStateSelectorsForTest()
	report.BlockStateResolutions = blockStateResolutionsForTest()
	report.MotionQualityLevel = "deterministic-block-motion-v1"
	report.MotionClock = "deterministic-test-clock-v1"
	report.MotionFrameBudget = 4
	report.MotionUnsupportedCSSAnimations = false
	report.MotionFrames = blockMotionFramesForTest()
	report.BlockAssetQualityLevel = "deterministic-local-block-assets-v1"
	report.BlockAssetNetworkFetchAllowed = false
	report.BlockAssetManifest = blockAssetManifestForTest(report.Source)
	report.BlockAssetCache = blockAssetCacheForTest()
	report.BlockAssetDiagnostics = blockAssetDiagnosticsForTest()
	report.BlockAssetRenderCommands = blockAssetRenderCommandsForTest()
	report.BlockAccessibilityTree = blockAccessibilityTreeForTest(report.Source)
	report.Events = blockSystemEventsForTest()
	report.Events = appendEventReportsWithNextOrder(report.Events,
		blockTextEventsForTest(),
		blockStateEventsForTest(),
		blockMotionEventsForTest(),
		blockAssetEventsForTest(),
	)
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
		{
			Order:     3,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
			Presented: true,
		},
	}
	report.StateTransitions = []StateTransitionReport{
		{
			Order:     1,
			Component: "SubmitBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     2,
			Component: "ResetBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     3,
			Component: "BlockSystemApp",
			Field:     "reading_order_checked",
			Before:    "false",
			After:     "true",
			Cause:     "block_graph",
		},
		{
			Order:     4,
			Component: "BlockLayoutApp",
			Field:     "width",
			Before:    "320",
			After:     "480",
			Cause:     "resize",
		},
		{
			Order:     5,
			Component: "ScrollBlock",
			Field:     "scroll_y",
			Before:    "0",
			After:     "32",
			Cause:     "scroll",
		},
	}
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(
		report.StateTransitions,
		blockSystemReadinessTransitionsForTest(),
	)
	report.BlockSystem = &BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "deterministic-headless-block-system-v1",
		Source:       report.Source,
		Renderer:     "software-rgba-headless",
		GoldenSet:    "surface-block-system-golden-v1",
		FrameCount:   3,
		GoldenHash:   "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Frames: []BlockSystemFrameReport{
			{
				Order:                 1,
				Label:                 "initial",
				Width:                 320,
				Height:                200,
				Stride:                1280,
				Checksum:              "1111111111111111111111111111111111111111111111111111111111111111",
				RepeatChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
				GoldenChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
				PaintEvidence:         true,
				LayoutEvidence:        true,
				AccessibilityEvidence: true,
			},
			{
				Order:                 2,
				Label:                 "focused",
				Width:                 320,
				Height:                200,
				Stride:                1280,
				Checksum:              "2222222222222222222222222222222222222222222222222222222222222222",
				RepeatChecksum:        "2222222222222222222222222222222222222222222222222222222222222222",
				GoldenChecksum:        "2222222222222222222222222222222222222222222222222222222222222222",
				PaintEvidence:         true,
				LayoutEvidence:        true,
				AccessibilityEvidence: true,
			},
			{
				Order:                 3,
				Label:                 "motion",
				Width:                 320,
				Height:                200,
				Stride:                1280,
				Checksum:              "3333333333333333333333333333333333333333333333333333333333333333",
				RepeatChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
				GoldenChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
				PaintEvidence:         true,
				LayoutEvidence:        true,
				AccessibilityEvidence: true,
			},
		},
		NegativeGuards: BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	report.Cases = append(report.Cases, blockSystemCasesForTest()...)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block system report: %v", err)
	}
	return raw
}
func validHeadlessMorphSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	morph := validMorphEvidenceMap()
	if mutate != nil {
		mutate(morph)
	}
	report["morph"] = morph
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Morph report: %v", err)
	}
	return raw
}
func validMorphEvidenceMap() map[string]any {
	return map[string]any{
		"schema":           "tetra.surface.morph.v1",
		"quality_level":    "deterministic-headless-morph-capsule-v1",
		"source":           "examples/surface/morph_core/surface_morph_command_palette.tetra",
		"module":           "lib.core.morph",
		"surface_scope":    "surface-morph-experimental-linux-web",
		"experimental":     true,
		"production_claim": false,
		"git_head":         "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"git_dirty":        true,
		"capsule_hash":     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"token_graph_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"capsule":          validMorphCapsuleMap(),
		"token_graph":      validMorphTokenGraphMap(),
		"materials":        validMorphMaterials(),
		"layout_modes": []any{
			"row",
			"column",
			"stack",
			"grid",
			"dock",
			"absolute",
			"overlay",
			"scroll",
		},
		"typography_roles":  []any{"title", "body", "label", "code"},
		"asset_refs":        validMorphAssetRefs(),
		"affordances":       validMorphAffordances(),
		"state_lenses":      validMorphStateLenses(),
		"motion_presets":    validMorphMotionPresets(),
		"recipes":           validMorphRecipes(),
		"recipe_expansions": validMorphRecipeExpansions(),
		"recipe_apps":       validMorphRecipeApps(),
		"accessibility":     validMorphAccessibilityProjectionMap(),
		"evidence_contract": validMorphEvidenceContractMap(),
		"memory_budget":     validMorphMemoryBudgetMap(),
		"negative_guards":   validMorphNegativeGuardsMap(),
		"nonclaims": []any{
			"DOM runtime absent",
			"React runtime absent",
			"Electron claim absent",
			"platform-native widgets absent",
			"full screen-reader production absent",
			"CSS cascade absent",
		},
	}
}
func validMorphCapsuleMap() map[string]any {
	return map[string]any{
		"namespace":         "tetra.surface.morph.app",
		"version":           "1",
		"capsule_hash":      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"imports":           []any{"lib.core.block", "lib.core.morph"},
		"explicit_imports":  true,
		"no_global_cascade": true,
	}
}
func validMorphTokenGraphMap() map[string]any {
	return map[string]any{
		"schema":    "tetra.surface.morph.token-graph.v1",
		"namespace": "tetra.surface.morph.app",
		"version":   "1",
		"hash":      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"categories": []any{
			"color",
			"space",
			"radius",
			"border",
			"elevation",
			"opacity",
			"typography",
			"motion",
			"z",
			"assets",
			"density",
		},
		"tokens":                       validMorphTokens(),
		"alias_cycle_rejected":         true,
		"duplicate_source_rejected":    true,
		"raw_literals_in_app_code":     false,
		"unresolved_fallback_rejected": true,
		"fallback_to_random_default":   false,
	}
}
func validMorphTokens() []any {
	return []any{
		map[string]any{
			"id":       "color.bg",
			"category": "color",
			"kind":     "rgba",
			"value":    "#0b0f14ff",
			"source":   "capsule",
			"hash":     "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},
		map[string]any{
			"id":       "space.3",
			"category": "space",
			"kind":     "px",
			"value":    "12",
			"source":   "capsule",
			"hash":     "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		},
		map[string]any{
			"id":       "radius.md",
			"category": "radius",
			"kind":     "px",
			"value":    "10",
			"source":   "capsule",
			"hash":     "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
		map[string]any{
			"id":       "type.label",
			"category": "typography",
			"kind":     "font",
			"value":    "Tetra UI 13 600 18",
			"source":   "capsule",
			"hash":     "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		},
		map[string]any{
			"id":       "motion.fast",
			"category": "motion",
			"kind":     "transition",
			"value":    "120 ease.out",
			"source":   "capsule",
			"hash":     "sha256:5555555555555555555555555555555555555555555555555555555555555555",
		},
	}
}
func validMorphMaterials() []any {
	return []any{
		map[string]any{
			"name":                      "surface.base",
			"paint_stack":               []any{"fill", "border", "radius"},
			"fill":                      "color.surface",
			"border":                    "border.subtle",
			"radius":                    "radius.md",
			"shadow":                    "",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "surface.elevated",
			"paint_stack":               []any{"fill", "border", "radius", "shadow"},
			"fill":                      "color.surface",
			"border":                    "border.subtle",
			"radius":                    "radius.md",
			"shadow":                    "elevation.2",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "control.primary",
			"paint_stack":               []any{"fill", "radius"},
			"fill":                      "color.accent",
			"border":                    "",
			"radius":                    "radius.sm",
			"shadow":                    "",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "translucent.panel",
			"paint_stack":               []any{"fill", "border", "radius", "shadow", "overlay"},
			"fill":                      "color.surfaceAlpha",
			"border":                    "border.glass",
			"radius":                    "radius.lg",
			"shadow":                    "elevation.3",
			"overlay":                   "gradient.vertical",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
	}
}
func validMorphAssetRefs() []any {
	return []any{
		map[string]any{
			"id":          "project.new",
			"kind":        "icon",
			"sha256":      "sha256:6666666666666666666666666666666666666666666666666666666666666666",
			"local":       true,
			"fallback_id": "icon.fallback",
			"tint_token":  "color.accent",
		},
		map[string]any{
			"id":          "command.search",
			"kind":        "icon",
			"sha256":      "sha256:7777777777777777777777777777777777777777777777777777777777777777",
			"local":       true,
			"fallback_id": "icon.fallback",
			"tint_token":  "color.muted",
		},
		map[string]any{
			"id":          "status.warning",
			"kind":        "icon",
			"sha256":      "sha256:8888888888888888888888888888888888888888888888888888888888888888",
			"local":       true,
			"fallback_id": "icon.fallback",
			"tint_token":  "color.warning",
		},
	}
}
func validMorphAffordances() []any {
	return []any{
		map[string]any{
			"name":                   "action",
			"role":                   "button",
			"focusable":              true,
			"action":                 "activate",
			"input":                  "",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "field.text",
			"role":                   "textbox",
			"focusable":              true,
			"action":                 "edit",
			"input":                  "editable_text",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "toggle",
			"role":                   "checkbox",
			"focusable":              true,
			"action":                 "toggle",
			"input":                  "toggle",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "navigation",
			"role":                   "navigation",
			"focusable":              false,
			"action":                 "",
			"input":                  "",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "region",
			"role":                   "region",
			"focusable":              false,
			"action":                 "",
			"input":                  "",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "overlay",
			"role":                   "dialog",
			"focusable":              true,
			"action":                 "dismiss",
			"input":                  "focus_trap",
			"projects_accessibility": true,
		},
		map[string]any{
			"name":                   "status",
			"role":                   "status",
			"focusable":              false,
			"action":                 "",
			"input":                  "",
			"projects_accessibility": true,
		},
	}
}
func validMorphStateLenses() []any {
	return []any{
		map[string]any{"selector": "hover", "property": "paint.overlay", "deterministic": true},
		map[string]any{"selector": "pressed", "property": "transform.scale", "deterministic": true},
		map[string]any{
			"selector":      "focusVisible",
			"property":      "paint.outline",
			"deterministic": true,
		},
		map[string]any{
			"selector":      "selected",
			"property":      "accessibility.selected",
			"deterministic": true,
		},
		map[string]any{"selector": "disabled", "property": "input.disabled", "deterministic": true},
		map[string]any{
			"selector":      "error",
			"property":      "paint.outline_color",
			"deterministic": true,
		},
		map[string]any{"selector": "loading", "property": "text.content", "deterministic": true},
	}
}
func validMorphMotionPresets() []any {
	return []any{
		map[string]any{
			"name":               "motion.fast",
			"duration_ms":        120,
			"curve":              "ease.out",
			"properties":         []any{"fill", "opacity", "transform"},
			"reduced_motion":     true,
			"deterministic_time": true,
		},
		map[string]any{
			"name":               "motion.soft",
			"duration_ms":        180,
			"curve":              "ease.inOut",
			"properties":         []any{"fill", "opacity", "transform"},
			"reduced_motion":     true,
			"deterministic_time": true,
		},
	}
}
func validMorphRecipes() []any {
	return []any{
		map[string]any{
			"name":                     "control.action@1",
			"output":                   "Block",
			"slots":                    []any{"label", "icon"},
			"inputs":                   []any{"text", "action", "variant"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "field.text@1",
			"output":                   "Block",
			"slots":                    []any{"label", "control"},
			"inputs":                   []any{"value", "on_text"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "command.item@1",
			"output":                   "Block",
			"slots":                    []any{"icon", "title", "subtitle"},
			"inputs":                   []any{"title", "subtitle", "icon", "selected"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "region.panel@1",
			"output":                   "Block",
			"slots":                    []any{"header", "body", "actions"},
			"inputs":                   []any{"title"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "form.field@1",
			"output":                   "Block",
			"slots":                    []any{"label", "control", "hint", "error"},
			"inputs":                   []any{"label", "value", "validation"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "nav.item@1",
			"output":                   "Block",
			"slots":                    []any{"icon", "label", "badge"},
			"inputs":                   []any{"label", "destination", "selected"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "metric.tile@1",
			"output":                   "Block",
			"slots":                    []any{"label", "value", "trend"},
			"inputs":                   []any{"label", "value", "trend"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "dialog.panel@1",
			"output":                   "Block",
			"slots":                    []any{"title", "body", "actions"},
			"inputs":                   []any{"title", "open", "dismiss"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "toast.notification@1",
			"output":                   "Block",
			"slots":                    []any{"icon", "message", "action"},
			"inputs":                   []any{"message", "severity", "timeout"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "tab.item@1",
			"output":                   "Block",
			"slots":                    []any{"label", "indicator"},
			"inputs":                   []any{"label", "selected", "target"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "list.row@1",
			"output":                   "Block",
			"slots":                    []any{"leading", "title", "meta", "action"},
			"inputs":                   []any{"title", "subtitle", "selected"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "app.shell@1",
			"output":                   "Block",
			"slots":                    []any{"nav", "toolbar", "content", "status"},
			"inputs":                   []any{"title", "target", "mode"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "toolbar@1",
			"output":                   "Block",
			"slots":                    []any{"leading", "actions", "search"},
			"inputs":                   []any{"title", "commands", "density"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "split.pane@1",
			"output":                   "Block",
			"slots":                    []any{"primary", "secondary", "divider"},
			"inputs":                   []any{"ratio", "orientation", "resize"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "status.bar@1",
			"output":                   "Block",
			"slots":                    []any{"target", "state", "progress"},
			"inputs":                   []any{"target", "dirty", "message"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "settings.form@1",
			"output":                   "Block",
			"slots":                    []any{"section", "fields", "actions"},
			"inputs":                   []any{"profile", "validation", "save"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "log.row@1",
			"output":                   "Block",
			"slots":                    []any{"level", "message", "timestamp"},
			"inputs":                   []any{"level", "message", "selected"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "empty.state@1",
			"output":                   "Block",
			"slots":                    []any{"title", "body", "action"},
			"inputs":                   []any{"reason", "action", "illustration"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
		map[string]any{
			"name":                     "error.panel@1",
			"output":                   "Block",
			"slots":                    []any{"title", "body", "retry"},
			"inputs":                   []any{"code", "message", "recover"},
			"expands_to_block_graph":   true,
			"hidden_app_state":         false,
			"platform_widgets":         false,
			"core_primitive_promotion": false,
		},
	}
}
func validMorphRecipeExpansions() []any {
	return []any{
		map[string]any{
			"recipe":        "control.action@1",
			"block_ids":     []any{4},
			"slot_bindings": []any{"label", "icon"},
			"variant":       "primary",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "field.text@1",
			"block_ids":     []any{3},
			"slot_bindings": []any{"label", "control"},
			"variant":       "default",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "command.item@1",
			"block_ids":     []any{4, 5},
			"slot_bindings": []any{"icon", "title", "subtitle"},
			"variant":       "selected",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "region.panel@1",
			"block_ids":     []any{2},
			"slot_bindings": []any{"header", "body", "actions"},
			"variant":       "elevated",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "form.field@1",
			"block_ids":     []any{3, 4},
			"slot_bindings": []any{"label", "control", "hint", "error"},
			"variant":       "validated",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "nav.item@1",
			"block_ids":     []any{5},
			"slot_bindings": []any{"icon", "label", "badge"},
			"variant":       "selected",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "metric.tile@1",
			"block_ids":     []any{2, 5},
			"slot_bindings": []any{"label", "value", "trend"},
			"variant":       "compact",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "dialog.panel@1",
			"block_ids":     []any{2, 4},
			"slot_bindings": []any{"title", "body", "actions"},
			"variant":       "modal",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "toast.notification@1",
			"block_ids":     []any{5},
			"slot_bindings": []any{"icon", "message", "action"},
			"variant":       "warning",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "tab.item@1",
			"block_ids":     []any{4},
			"slot_bindings": []any{"label", "indicator"},
			"variant":       "active",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "list.row@1",
			"block_ids":     []any{4, 5},
			"slot_bindings": []any{"leading", "title", "meta", "action"},
			"variant":       "interactive",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "app.shell@1",
			"block_ids":     []any{1, 2, 5},
			"slot_bindings": []any{"nav", "toolbar", "content", "status"},
			"variant":       "studio",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "toolbar@1",
			"block_ids":     []any{2, 4},
			"slot_bindings": []any{"leading", "actions", "search"},
			"variant":       "compact",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "split.pane@1",
			"block_ids":     []any{2, 3, 4},
			"slot_bindings": []any{"primary", "secondary", "divider"},
			"variant":       "horizontal",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "status.bar@1",
			"block_ids":     []any{5},
			"slot_bindings": []any{"target", "state", "progress"},
			"variant":       "reporting",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "settings.form@1",
			"block_ids":     []any{3, 4},
			"slot_bindings": []any{"section", "fields", "actions"},
			"variant":       "validated",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "log.row@1",
			"block_ids":     []any{4, 5},
			"slot_bindings": []any{"level", "message", "timestamp"},
			"variant":       "selected",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "empty.state@1",
			"block_ids":     []any{3},
			"slot_bindings": []any{"title", "body", "action"},
			"variant":       "onboarding",
			"reported":      true,
		},
		map[string]any{
			"recipe":        "error.panel@1",
			"block_ids":     []any{2, 5},
			"slot_bindings": []any{"title", "body", "retry"},
			"variant":       "recoverable",
			"reported":      true,
		},
	}
}
func validMorphRecipeApps() []any {
	return []any{
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_command_palette.tetra",
			"module": "examples.surface.morph_core.surface_morph_command_palette",
			"recipes": []any{
				"control.action@1",
				"field.text@1",
				"command.item@1",
				"region.panel@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              7,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_project_dashboard.tetra",
			"module": "examples.surface.morph_core.surface_morph_project_dashboard",
			"recipes": []any{
				"region.panel@1",
				"metric.tile@1",
				"list.row@1",
				"toast.notification@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              7,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_settings.tetra",
			"module": "examples.surface.morph_core.surface_morph_settings",
			"recipes": []any{
				"form.field@1",
				"field.text@1",
				"tab.item@1",
				"control.action@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              7,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_editor_shell.tetra",
			"module": "examples.surface.morph_core.surface_morph_editor_shell",
			"recipes": []any{
				"nav.item@1",
				"tab.item@1",
				"command.item@1",
				"region.panel@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              7,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_glass_panel.tetra",
			"module": "examples.surface.morph_core.surface_morph_glass_panel",
			"recipes": []any{
				"dialog.panel@1",
				"toast.notification@1",
				"control.action@1",
				"region.panel@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              7,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_core/surface_morph_studio_shell.tetra",
			"module": "examples.surface.morph_core.surface_morph_studio_shell",
			"recipes": []any{
				"app.shell@1",
				"toolbar@1",
				"split.pane@1",
				"status.bar@1",
				"settings.form@1",
				"log.row@1",
				"empty.state@1",
				"error.panel@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              12,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
		map[string]any{
			"source": "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
			"module": "examples.surface.morph_flagship.surface_morph_rendered_studio_shell",
			"recipes": []any{
				"app.shell@1",
				"nav.item@1",
				"toolbar@1",
				"tab.item@1",
				"split.pane@1",
				"status.bar@1",
				"command.item@1",
				"settings.form@1",
				"log.row@1",
				"metric.tile@1",
				"toast.notification@1",
				"dialog.panel@1",
				"empty.state@1",
				"error.panel@1",
				"control.action@1",
				"field.text@1",
			},
			"expands_to_block_graph":   true,
			"block_count":              18,
			"accessibility_projection": true,
			"hidden_app_state":         false,
			"react_runtime":            false,
			"electron_runtime":         false,
			"dom_runtime":              false,
			"platform_widgets":         false,
			"output_primitives":        []any{"Block"},
		},
	}
}
func validMorphAccessibilityProjectionMap() map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.morph.accessibility-projection.v1",
		"derived_from_block_graph": true,
		"safety_overrides_win":     true,
		"snapshot_evidence":        true,
		"required_fields": []any{
			"role",
			"name",
			"description",
			"action",
			"state",
			"bounds",
			"focus_order",
			"reading_order",
			"labelled_by",
			"label_for",
		},
		"roles": []any{
			"button",
			"textbox",
			"checkbox",
			"navigation",
			"region",
			"dialog",
			"status",
		},
	}
}
func validMorphEvidenceContractMap() map[string]any {
	return map[string]any{
		"capsule_hash":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"token_graph_hash":   "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"recipe_expansions":  true,
		"block_tree":         true,
		"resolved_layout":    true,
		"paint_layers":       true,
		"text_runs":          true,
		"motion_frames":      true,
		"asset_hashes":       true,
		"accessibility_tree": true,
		"memory_budget":      true,
		"frame_checksums":    true,
		"artifact_hashes":    true,
	}
}
func validMorphMemoryBudgetMap() map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.morph-memory-budget.v1",
		"expanded_recipe_count":    19,
		"block_count":              24,
		"paint_command_count":      6,
		"layout_pass_count":        8,
		"text_run_count":           3,
		"motion_active_count":      1,
		"glyph_cache_bytes":        4096,
		"asset_cache_bytes":        5376,
		"layout_cache_bytes":       8192,
		"framebuffer_bytes":        256000,
		"peak_rss_bytes":           0,
		"alloc_count":              0,
		"frame_count":              3,
		"bounded_caches":           true,
		"unbounded_cache_rejected": true,
	}
}
func validMorphNegativeGuardsMap() map[string]any {
	return map[string]any{
		"no_core_widget_primitives":          true,
		"no_dom_ui":                          true,
		"no_react":                           true,
		"no_electron":                        true,
		"no_user_js":                         true,
		"no_platform_widgets":                true,
		"missing_token_rejected":             true,
		"alias_cycle_rejected":               true,
		"duplicate_token_source_rejected":    true,
		"duplicate_recipe_name_rejected":     true,
		"missing_recipe_expansion_rejected":  true,
		"unresolved_token_rejected":          true,
		"missing_asset_rejected":             true,
		"unbounded_cache_rejected":           true,
		"fake_motion_rejected":               true,
		"fake_accessibility_rejected":        true,
		"unsupported_target_rejected":        true,
		"dirty_checkout_production_rejected": true,
	}
}

func validLinuxX64RealWindowBlockSystemSurfaceReportJSON(
	t *testing.T,
	mutate func(*Report),
) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	report.Target = "linux-x64"
	report.Runtime = "surface-linux-x64"
	report.HostEvidence = HostEvidenceReport{
		Level:       "linux-x64-real-window",
		Backend:     "wayland-shm-rgba",
		Framebuffer: true,
		RealWindow:  true,
		NativeInput: true,
	}
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(1),
			ExpectedExitCode: intPtrForTest(1),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system-real-window-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(42),
			ExpectedExitCode: intPtrForTest(42),
		},
		{
			Name:     "surface linux-x64 runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-system",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   1,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     2,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
			Presented: true,
		},
		{
			Order:     3,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
			Presented: true,
		},
		{
			Order:     5,
			Width:     400,
			Height:    240,
			Stride:    1600,
			Checksum:  "5555555555555555555555555555555555555555555555555555555555555555",
			Presented: true,
		},
	}
	report.BlockSystem.QualityLevel = "linux-x64-real-window-block-system-v1"
	report.BlockSystem.Renderer = "wayland-shm-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-linux-x64-real-window-v1"
	report.BlockSystem.FrameCount = 4
	report.BlockSystem.GoldenHash = ("sha256:" +
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" +
		"dddd")
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{
			Order:                 1,
			Label:                 "initial",
			Width:                 320,
			Height:                200,
			Stride:                1280,
			Checksum:              "1111111111111111111111111111111111111111111111111111111111111111",
			RepeatChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
			GoldenChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
		{
			Order:                 2,
			Label:                 "focused",
			Width:                 320,
			Height:                200,
			Stride:                1280,
			Checksum:              "2222222222222222222222222222222222222222222222222222222222222222",
			RepeatChecksum:        "2222222222222222222222222222222222222222222222222222222222222222",
			GoldenChecksum:        "2222222222222222222222222222222222222222222222222222222222222222",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
		{
			Order:                 3,
			Label:                 "motion",
			Width:                 320,
			Height:                200,
			Stride:                1280,
			Checksum:              "3333333333333333333333333333333333333333333333333333333333333333",
			RepeatChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
			GoldenChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
		{
			Order:                 5,
			Label:                 "real-window-focused",
			Width:                 400,
			Height:                240,
			Stride:                1600,
			Checksum:              "5555555555555555555555555555555555555555555555555555555555555555",
			RepeatChecksum:        "5555555555555555555555555555555555555555555555555555555555555555",
			GoldenChecksum:        "5555555555555555555555555555555555555555555555555555555555555555",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{
			Kind:            "resize",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     4,
			BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 4, 0},
			BeforeState:     map[string]string{"BlockSystemApp.width": "320"},
			AfterState:      map[string]string{"BlockSystemApp.width": "400"},
		},
		{
			Kind:            "close",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     5,
			BufferSlots:     []int{1, 0, 0, 0, 0, 400, 240, 5, 0},
			BeforeState:     map[string]string{"BlockSystemApp.closed": "false"},
			AfterState:      map[string]string{"BlockSystemApp.closed": "true"},
		},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(
		report.StateTransitions,
		[]StateTransitionReport{
			{
				Component: "SubmitBlock",
				Field:     "pressed",
				Before:    "false",
				After:     "true",
				Cause:     "key_down",
			},
			{
				Component: "BlockSystemApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
			{
				Component: "BlockSystemApp",
				Field:     "closed",
				Before:    "false",
				After:     "true",
				Cause:     "close",
			},
		},
	)
	report.Cases = blockSystemLinuxX64RealWindowCasesForTest()
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux-x64 real-window Block system report: %v", err)
	}
	return raw
}

func validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(
	t *testing.T,
	mutate func(*Report),
) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	report.Target = "wasm32-web"
	report.Runtime = "surface-wasm32-web"
	report.HostEvidence = HostEvidenceReport{
		Level:         "wasm32-web-browser-canvas-input",
		Backend:       "browser-canvas-rgba",
		Framebuffer:   true,
		NativeInput:   true,
		BrowserCanvas: true,
		BrowserInput:  true,
	}
	report.Processes = []ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> scenario=block-system " +
				"wasm=/tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtrForTest(0),
			ExpectedExitCode: intPtrForTest(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web /tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     "Chromium Block-system fixture",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
		{
			Name: "surface wasm32-web browser canvas trace",
			Kind: "runtime",
			Path: ("/usr/bin/chromium --headless --dump-dom " +
				"<surface-browser-canvas-file-runner scenario=block-system>"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtrForTest(0),
		},
	}
	report.Artifacts = []ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-system.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   8604,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-block-system.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4939,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   1184,
		},
	}
	report.ArtifactScan = ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   3,
		ForbiddenPaths: nil,
		Pass:           true,
	}
	report.Frames = []FrameReport{
		{
			Order:     1,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
			Presented: true,
		},
		{
			Order:     3,
			Width:     320,
			Height:    200,
			Stride:    1280,
			Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
			Presented: true,
		},
		{
			Order:     5,
			Width:     400,
			Height:    240,
			Stride:    1600,
			Checksum:  "5555555555555555555555555555555555555555555555555555555555555555",
			Presented: true,
		},
	}
	report.BlockSystem.QualityLevel = "wasm32-web-browser-canvas-block-system-v1"
	report.BlockSystem.Renderer = "browser-canvas-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-wasm32-web-browser-canvas-v1"
	report.BlockSystem.FrameCount = 3
	report.BlockSystem.GoldenHash = ("sha256:" +
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" +
		"dddd")
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{
			Order:                 1,
			Label:                 "initial",
			Width:                 320,
			Height:                200,
			Stride:                1280,
			Checksum:              "1111111111111111111111111111111111111111111111111111111111111111",
			RepeatChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
			GoldenChecksum:        "1111111111111111111111111111111111111111111111111111111111111111",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
		{
			Order:                 3,
			Label:                 "motion",
			Width:                 320,
			Height:                200,
			Stride:                1280,
			Checksum:              "3333333333333333333333333333333333333333333333333333333333333333",
			RepeatChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
			GoldenChecksum:        "3333333333333333333333333333333333333333333333333333333333333333",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
		{
			Order:                 5,
			Label:                 "browser-canvas-focused",
			Width:                 400,
			Height:                240,
			Stride:                1600,
			Checksum:              "5555555555555555555555555555555555555555555555555555555555555555",
			RepeatChecksum:        "5555555555555555555555555555555555555555555555555555555555555555",
			GoldenChecksum:        "5555555555555555555555555555555555555555555555555555555555555555",
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{
			Kind:            "resize",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     4,
			BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 4, 0},
			BeforeState:     map[string]string{"BlockSystemApp.width": "320"},
			AfterState:      map[string]string{"BlockSystemApp.width": "400"},
		},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(
		report.StateTransitions,
		[]StateTransitionReport{
			{
				Component: "SubmitBlock",
				Field:     "pressed",
				Before:    "false",
				After:     "true",
				Cause:     "key_down",
			},
			{
				Component: "BlockSystemApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
		},
	)
	report.Cases = blockSystemWASM32WebBrowserCanvasCasesForTest()
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal wasm32-web browser-canvas Block system report: %v", err)
	}
	return raw
}
func blockSystemComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{
			ID:        "BlockSystemApp",
			Type:      "examples.surface.block_core.surface_block_system.BlockSystemApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id": "4",
				"quality":    "deterministic-headless-block-system-v1",
			},
		},
		{
			ID:        "PanelBlock",
			Type:      "examples.surface.block_core.surface_block_system.PanelBlock",
			Parent:    "BlockSystemApp",
			Bounds:    RectReport{X: 16, Y: 16, W: 288, H: 168},
			Abilities: abilities,
			State:     map[string]string{"paint_layers": "5"},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_core.surface_block_system.LabelBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "4", "label_for": "4"},
		},
		{
			ID:        "SubmitBlock",
			Type:      "examples.surface.block_core.surface_block_system.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "true", "action": "submit"},
		},
		{
			ID:        "ResetBlock",
			Type:      "examples.surface.block_core.surface_block_system.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false", "action": "reset"},
		},
		{
			ID:        "BlockLayoutApp",
			Type:      "examples.surface.block_core.surface_block_system.BlockLayoutApp",
			Bounds:    RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"width":          "480",
				"layout_quality": "deterministic-block-layout-v1",
			},
		},
		{
			ID:        "ScrollBlock",
			Type:      "examples.surface.block_core.surface_block_system.ScrollBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    RectReport{X: 236, Y: 72, W: 72, H: 80},
			Abilities: abilities,
			State:     map[string]string{"scroll_y": "32"},
		},
	}
}
func blockSystemEventsForTest() []EventReport {
	return []EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               80,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"SubmitBlock.focused": "false"},
			AfterState:      map[string]string{"SubmitBlock.focused": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"SubmitBlock.value_len": "0"},
			AfterState:      map[string]string{"SubmitBlock.value_len": "2"},
		},
		{
			Order:           3,
			Kind:            "key_down",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Key:             13,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{3, 0, 0, 0, 13, 320, 200, 2, 0},
			BeforeState:     map[string]string{"SubmitBlock.pressed": "false"},
			AfterState:      map[string]string{"SubmitBlock.pressed": "true"},
		},
		{
			Order:           4,
			Kind:            "scroll",
			TargetComponent: "ScrollBlock",
			DispatchPath:    []string{"BlockLayoutApp", "ScrollBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     3,
			BufferSlots:     []int{7, 0, 0, 0, 0, 320, 200, 3, 0},
			BeforeState:     map[string]string{"ScrollBlock.scroll_y": "0"},
			AfterState:      map[string]string{"ScrollBlock.scroll_y": "32"},
		},
	}
}
func retargetBlockSystemComponentsForTest(components []ComponentReport) []ComponentReport {
	retargeted := make([]ComponentReport, len(components))
	for i, component := range components {
		component.Type = "examples.surface.block_core.surface_block_system." + typeBaseName(
			component.Type,
		)
		retargeted[i] = component
	}
	return retargeted
}
func typeBaseName(value string) string {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return value
	}
	return value[index+1:]
}

func appendEventReportsWithNextOrder(
	events []EventReport,
	additions ...[]EventReport,
) []EventReport {
	nextOrder := 0
	if len(events) > 0 {
		nextOrder = events[len(events)-1].Order
	}
	for _, group := range additions {
		for _, event := range group {
			nextOrder++
			event.Order = nextOrder
			events = append(events, event)
		}
	}
	return events
}

func appendStateTransitionReportsWithNextOrder(
	transitions []StateTransitionReport,
	additions ...[]StateTransitionReport,
) []StateTransitionReport {
	nextOrder := 0
	if len(transitions) > 0 {
		nextOrder = transitions[len(transitions)-1].Order
	}
	for _, group := range additions {
		for _, transition := range group {
			nextOrder++
			transition.Order = nextOrder
			transitions = append(transitions, transition)
		}
	}
	return transitions
}
func blockSystemReadinessTransitionsForTest() []StateTransitionReport {
	return []StateTransitionReport{
		{
			Order:     1,
			Component: "InputBlock",
			Field:     "buffer",
			Before:    "",
			After:     "OKd0a2",
			Cause:     "text_input",
		},
		{
			Order:     2,
			Component: "InputBlock",
			Field:     "caret",
			Before:    "0",
			After:     "4",
			Cause:     "text_input",
		},
		{
			Order:     3,
			Component: "StateBlock",
			Field:     "selector_flags",
			Before:    "0",
			After:     "127",
			Cause:     "pointer/key/state input",
		},
		{
			Order:     4,
			Component: "StateBlock",
			Field:     "resolved_fill",
			Before:    "#20262eff",
			After:     "#2d9bf0ff",
			Cause:     "hover",
		},
		{
			Order:     5,
			Component: "StateBlock",
			Field:     "resolved_scale",
			Before:    "100",
			After:     "97",
			Cause:     "pressed",
		},
		{
			Order:     6,
			Component: "StateBlock",
			Field:     "disabled",
			Before:    "false",
			After:     "true",
			Cause:     "disabled selector",
		},
		{
			Order:     7,
			Component: "StateBlock",
			Field:     "error",
			Before:    "false",
			After:     "true",
			Cause:     "error selector",
		},
		{
			Order:     8,
			Component: "StateBlock",
			Field:     "loading",
			Before:    "false",
			After:     "true",
			Cause:     "loading selector",
		},
		{
			Order:     9,
			Component: "MotionBlock",
			Field:     "opacity",
			Before:    "80",
			After:     "200",
			Cause:     "motion frame",
		},
		{
			Order:     10,
			Component: "MotionBlock",
			Field:     "color",
			Before:    "#203040ff",
			After:     "#60aef4ff",
			Cause:     "motion frame",
		},
		{
			Order:     11,
			Component: "MotionBlock",
			Field:     "scale",
			Before:    "100",
			After:     "108",
			Cause:     "motion frame",
		},
		{
			Order:     12,
			Component: "MotionBlock",
			Field:     "translate_x",
			Before:    "0",
			After:     "12",
			Cause:     "motion frame",
		},
		{
			Order:     13,
			Component: "MotionBlock",
			Field:     "motion_complete",
			Before:    "false",
			After:     "true",
			Cause:     "duration elapsed",
		},
		{
			Order:     14,
			Component: "MotionBlock",
			Field:     "reduced_motion",
			Before:    "false",
			After:     "true",
			Cause:     "accessibility setting",
		},
		{
			Order:     15,
			Component: "IconBlock",
			Field:     "tint",
			Before:    "#ffffffff",
			After:     "#60aef4ff",
			Cause:     "asset tint",
		},
		{
			Order:     16,
			Component: "ImageBlock",
			Field:     "scale",
			Before:    "1x",
			After:     "2x",
			Cause:     "asset scale",
		},
		{
			Order:     17,
			Component: "MissingAssetBlock",
			Field:     "fallback",
			Before:    "missing",
			After:     "fallback-raster",
			Cause:     "missing asset",
		},
	}
}
func blockSystemCasesForTest() []CaseReport {
	return []CaseReport{
		{
			Name:          "block graph duplicate id rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "duplicate Block ID",
		},
		{
			Name:          "block graph missing parent rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing parent",
		},
		{
			Name:          "block graph cycle rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "cycle",
		},
		{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block paint unsupported blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "unsupported blur",
		},
		{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block compositor dirty rect invalidation cache",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "block renderer opacity transform clipped child",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block renderer gpu production claim rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "gpu production",
		},
		{
			Name:          "block renderer unsupported backdrop blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "backdrop blur",
		},
		{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block layout aspect density stable rounding",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block layout no css flexbox parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "CSS flexbox parity nonclaim",
		},
		{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block state disabled error loading overrides",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block state no css pseudo parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css pseudo nonclaim",
		},
		{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block motion opacity color transform frames",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "block motion reduced motion instant settle",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block motion no css animation parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css animation nonclaim",
		},
		{
			Name: "block asset deterministic manifest hashes",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset missing fallback diagnostic", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block asset network url rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "network asset rejected",
		},
		{
			Name: "block accessibility tree derived from block graph",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block accessibility focusable actionable name required",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing accessible name",
		},
		{
			Name:          "block accessibility label relationship mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "label relationship mismatch",
		},
		{
			Name:          "block accessibility reading order graph mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "reading order mismatch",
		},
		{
			Name:          "block accessibility screen-reader claim without platform proof rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "screen reader proof required",
		},
		{
			Name: "block accessibility platform claim scoped metadata only",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block system headless golden checksums", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block system deterministic repeat checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block system missing frame checksum rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "frame checksum required",
		},
		{
			Name:          "block system nondeterministic checksum rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "repeat checksum mismatch",
		},
		{
			Name:          "block system missing paint evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "paint evidence required",
		},
		{
			Name:          "block system missing layout evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "layout evidence required",
		},
		{
			Name:          "block system missing accessibility evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "accessibility evidence required",
		},
		{Name: "block system bounded memory budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system stress render loop budget", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block system performance nonclaim",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "Electron comparison benchmark not claimed",
		},
	}
}
func blockSystemLinuxX64RealWindowCasesForTest() []CaseReport {
	cases := []CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
	}
	for _, tc := range blockSystemCasesForTest() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(
		cases,
		CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		CaseReport{
			Name: "linux-x64 native input event pump",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "linux-x64 real-window resize event",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "linux-x64 real-window close event",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system linux-x64 real-window frame presentation",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system linux-x64 native input state transition",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system linux-x64 real-window checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block system missing real-window presentation rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "real-window presentation required",
		},
		CaseReport{
			Name:          "block system missing native input state transition rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "native input required",
		},
	)
	return cases
}
func blockSystemWASM32WebBrowserCanvasCasesForTest() []CaseReport {
	cases := []CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
	}
	for _, tc := range blockSystemCasesForTest() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(
		cases,
		CaseReport{
			Name: "wasm32-web browser canvas surface",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web browser canvas RGBA readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web browser canvas pointer input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web browser canvas keyboard input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web browser canvas resize input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web browser canvas text input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "wasm32-web Surface Host ABI imports",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "compiler-owned wasm Surface loader",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "compiler-owned browser canvas Surface host",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system wasm32-web browser-canvas frame readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system wasm32-web browser-canvas native input state transition",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name: "block system wasm32-web browser-canvas checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		CaseReport{
			Name:          "block system browser-canvas node runtime substitution rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "browser evidence required",
		},
		CaseReport{
			Name:          "block system browser-canvas missing RGBA readback rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "RGBA readback required",
		},
		CaseReport{
			Name:          "block system browser-canvas script sidecar artifact rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "script artifact rejected",
		},
		CaseReport{
			Name:          "block system browser-canvas html visual sidecar artifact rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "html artifact rejected",
		},
	)
	return cases
}
func blockMemoryBudgetForTest(report *Report) *BlockMemoryBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotals(report.Frames)
	cacheUsedBytes := len(report.PaintCommands)*2048 + 4096 + report.BlockAssetCache.UsedBytes
	totalCacheBudgetBytes := report.PaintCacheBudgetBytes +
		report.TextCacheBudgetBytes +
		report.BlockAssetCache.BudgetBytes
	return &BlockMemoryBudgetReport{
		Schema:                   "tetra.surface.block-memory-budget.v1",
		Scope:                    "surface-block-system-local-budget-v1",
		BlockCount:               len(report.Components),
		StressBlockCount:         128,
		RenderLoopCount:          32,
		StateLoopCount:           len(report.StateTransitions),
		MotionFrameCount:         len(report.MotionFrames),
		InputEventCount:          len(report.Events),
		PaintCommandCount:        len(report.PaintCommands),
		TextRenderCommandCount:   len(report.TextRenderCommands),
		AssetRenderCommandCount:  len(report.BlockAssetRenderCommands),
		PeakFramebufferBytes:     peakFramebufferBytes,
		TotalFramebufferBytes:    totalFramebufferBytes,
		FramebufferBudgetBytes:   1048576,
		PaintCacheUsedBytes:      len(report.PaintCommands) * 2048,
		PaintCacheBudgetBytes:    report.PaintCacheBudgetBytes,
		TextCacheUsedBytes:       4096,
		TextCacheBudgetBytes:     report.TextCacheBudgetBytes,
		AssetCacheUsedBytes:      report.BlockAssetCache.UsedBytes,
		AssetCacheBudgetBytes:    report.BlockAssetCache.BudgetBytes,
		TotalCacheUsedBytes:      cacheUsedBytes,
		TotalCacheBudgetBytes:    totalCacheBudgetBytes,
		EstimatedAllocationBytes: totalFramebufferBytes + cacheUsedBytes,
		RSSMeasured:              false,
		PeakRSSBytes:             0,
		BoundedCaches:            true,
		UnboundedCacheRejected:   true,
		StressScene:              "deterministic-block-stress-128",
		PerformanceClaim:         "none",
		NonClaims: []string{
			"no Electron comparison benchmark",
			"no broad performance superiority claim",
			"RSS is optional host evidence and not required for this local budget",
		},
	}
}

// ---- report_component_tree_test.go ----

func TestValidateReportAcceptsHeadlessComponentTreeSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsComponentTreeMissingAPIEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without component_tree_api to fail")
	}
	if !strings.Contains(err.Error(), "component_tree_api") {
		t.Fatalf("error = %v, want component_tree_api diagnostic", err)
	}
}
func TestValidateReportRejectsComponentTreeManualBookkeepingAPIEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.ManualBookkeeping = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report with manual bookkeeping to fail")
	}
	for _, want := range []string{"component_tree_api", "manual_bookkeeping"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPINodeCountMismatch(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Builder.NodeCount = 6
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report with builder node count mismatch to fail")
	}
	for _, want := range []string{"builder", "node_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingTreeValidate(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Invariants.TreeValidateRan = false
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without tree_validate evidence to fail")
	}
	for _, want := range []string{"tree_validate", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingRowLayoutHelper(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.LayoutHelpers = []ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without tree_layout_row evidence to fail")
	}
	for _, want := range []string{"tree_layout_row", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingFocusWrap(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.FocusHelpers = []ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf(
			"expected component tree API report without ResetButton -> TextBox helper evidence to fail",
		)
	}
	for _, want := range []string{"ResetButton", "TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIHitTestPathSkippingRow(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTreeAPI.HitTests {
			if report.ComponentTreeAPI.HitTests[i].Target == "ResetButton" {
				report.ComponentTreeAPI.HitTests[i].Path = []int{0, 1, 6}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API hit-test path skipping Row to fail")
	}
	for _, want := range []string{"hit", "path"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPISourceMismatch(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Source = "examples/surface/runtime/surface_counter.tetra"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API source mismatch to fail")
	}
	for _, want := range []string{"source", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeMissingEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report without component_tree to fail")
	}
	if !strings.Contains(err.Error(), "component_tree") {
		t.Fatalf("error = %v, want component_tree diagnostic", err)
	}
}
func TestValidateReportRejectsHardcodedTreeClickEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree.DispatchPaths = nil
		report.Events = []EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "-1"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3"},
			},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected hardcoded component click evidence without root-to-leaf path to fail")
	}
	for _, want := range []string{"dispatch path", "parent"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeDispatchPathSkippingRow(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.DispatchPaths {
			if report.ComponentTree.DispatchPaths[i].TargetID == 6 {
				report.ComponentTree.DispatchPaths[i].Path = []int{0, 1, 6}
			}
		}
		for i := range report.Events {
			if report.Events[i].TargetComponent == "ResetButton" {
				report.Events[i].DispatchPath = []string{"TreeApp", "Column", "ResetButton"}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree dispatch path skipping Row to fail")
	}
	for _, want := range []string{"dispatch path", "parent"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeTextMutationWhileButtonFocused(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			if report.Events[i].Order == 6 {
				report.Events[i].AfterState["TextBox.buffer"] = "BAD"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected TextBox mutation while Button focused to fail")
	}
	for _, want := range []string{"TextBox", "Button focused"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeResizeWithoutLayoutChange(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			if report.Events[i].Kind == "resize" {
				report.Events[i].AfterState["TextBox.bounds.w"] = "288"
			}
		}
		for i := range report.StateTransitions {
			if report.StateTransitions[i].Field == "TextBox.bounds.w" {
				report.StateTransitions[i].After = "288"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree resize without changed layout bounds to fail")
	}
	for _, want := range []string{"resize", "bounds"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeFocusOrderNotTreeOrder(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree.FocusOrder = []int{3, 6, 5}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with shuffled focus_order to fail")
	}
	for _, want := range []string{"focus_order", "TextBox -> SubmitButton -> ResetButton"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeMissingFocusWrapEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		var events []EventReport
		for _, event := range report.Events {
			if event.Kind == "key_down" && event.Key == 9 &&
				event.BeforeState["TreeApp.focused_id"] == "6" &&
				event.AfterState["TreeApp.focused_id"] == "3" {
				continue
			}
			events = append(events, event)
		}
		report.Events = events
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report without ResetButton -> TextBox Tab wrap to fail")
	}
	for _, want := range []string{"Tab focus traversal", "ResetButton -> TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeButtonActionWithoutFocusedKeyRoute(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			event := &report.Events[i]
			if event.Kind != "key_down" {
				continue
			}
			if event.TargetComponent == "SubmitButton" ||
				event.TargetComponent == "ResetButton" {
				event.Kind = "mouse_up"
				event.Key = 0
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf(
			"expected component tree report without focused keyboard button action route to fail",
		)
	}
	for _, want := range []string{"button action", "keyboard"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeRowChildrenOverlap(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.Nodes {
			if report.ComponentTree.Nodes[i].Name == "ResetButton" {
				report.ComponentTree.Nodes[i].Bounds.X = 100
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with overlapping Row children to fail")
	}
	for _, want := range []string{"Row", "overlap"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeColumnChildrenOutOfOrder(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.Nodes {
			if report.ComponentTree.Nodes[i].Name == "NameLabel" {
				report.ComponentTree.Nodes[i].Bounds.Y = 40
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with Column children out of visual order to fail")
	}
	for _, want := range []string{"Column", "child_index"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func validHeadlessComponentTreeSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	report := Report{
		Schema:        SchemaV1,
		Status:        "pass",
		Target:        "headless",
		Host:          "linux-x64",
		Runtime:       "surface-headless",
		SurfaceSchema: "tetra.surface.v1",
		HostABI:       "tetra.surface.host-abi.v1",
		HostEvidence: HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		},
		Source: "examples/surface/toolkit/surface_tree_app.tetra",
		Processes: []ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/toolkit/surface_tree_app.tetra -o " +
					"/tmp/surface-artifacts/surface-tree-app"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtrForTest(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-tree-app",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtrForTest(1),
				ExpectedExitCode: intPtrForTest(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtrForTest(0),
			},
		},
		Artifacts: []ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-tree-app",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   81234,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Size:   22000,
			},
		},
		ArtifactScan: ArtifactScanReport{
			Root:           "/tmp/surface-artifacts",
			FilesChecked:   2,
			ForbiddenPaths: []string{},
			Pass:           true,
		},
		Components: []ComponentReport{
			treeComponent(
				"TreeApp",
				"examples.surface.toolkit.surface_tree_app.TreeApp",
				"",
				RectReport{X: 0, Y: 0, W: 400, H: 240},
				map[string]string{
					"focused_id":         "6",
					"submitted_count":    "1",
					"reset_count":        "1",
					"width":              "400",
					"height":             "240",
					"accessibility_role": "none",
				},
			),
			treeComponent(
				"Column",
				"examples.surface.toolkit.surface_tree_app.Column",
				"TreeApp",
				RectReport{X: 0, Y: 0, W: 400, H: 240},
				map[string]string{"child_count": "3", "accessibility_role": "none"},
			),
			treeComponent(
				"NameLabel",
				"examples.surface.toolkit.surface_tree_app.TextLabel",
				"Column",
				RectReport{X: 16, Y: 16, W: 288, H: 24},
				map[string]string{"text": "Name", "accessibility_role": "label"},
			),
			treeComponent(
				"TextBox",
				"examples.surface.toolkit.surface_tree_app.TextBox",
				"Column",
				RectReport{X: 16, Y: 48, W: 368, H: 44},
				map[string]string{
					"focused":            "false",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			),
			treeComponent(
				"ButtonRow",
				"examples.surface.toolkit.surface_tree_app.Row",
				"Column",
				RectReport{X: 16, Y: 104, W: 368, H: 44},
				map[string]string{"child_count": "2", "accessibility_role": "none"},
			),
			treeComponent(
				"SubmitButton",
				"examples.surface.toolkit.surface_tree_app.Button",
				"ButtonRow",
				RectReport{X: 16, Y: 104, W: 132, H: 44},
				map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"accessibility_role": "button",
				},
			),
			treeComponent(
				"ResetButton",
				"examples.surface.toolkit.surface_tree_app.Button",
				"ButtonRow",
				RectReport{X: 160, Y: 104, W: 132, H: 44},
				map[string]string{
					"focused":            "true",
					"press_count":        "1",
					"accessibility_role": "button",
				},
			),
		},
		ComponentTree: &ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "semi-dynamic-child-list",
			RootID:       0,
			NodeCount:    7,
			FocusedID:    6,
			Nodes: []ComponentTreeNodeReport{
				{
					ID:         0,
					Name:       "TreeApp",
					Kind:       "root",
					ParentID:   -1,
					ChildIndex: 0,
					FirstChild: 1,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         1,
					Name:       "Column",
					Kind:       "column",
					ParentID:   0,
					ChildIndex: 0,
					FirstChild: 2,
					ChildCount: 3,
					Focusable:  false,
					Bounds:     RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         2,
					Name:       "NameLabel",
					Kind:       "text",
					ParentID:   1,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     RectReport{X: 16, Y: 16, W: 288, H: 24},
				},
				{
					ID:         3,
					Name:       "TextBox",
					Kind:       "textbox",
					ParentID:   1,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     RectReport{X: 16, Y: 48, W: 368, H: 44},
				},
				{
					ID:         4,
					Name:       "ButtonRow",
					Kind:       "row",
					ParentID:   1,
					ChildIndex: 2,
					FirstChild: 5,
					ChildCount: 2,
					Focusable:  false,
					Bounds:     RectReport{X: 16, Y: 104, W: 368, H: 44},
				},
				{
					ID:         5,
					Name:       "SubmitButton",
					Kind:       "button",
					ParentID:   4,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     RectReport{X: 16, Y: 104, W: 132, H: 44},
				},
				{
					ID:         6,
					Name:       "ResetButton",
					Kind:       "button",
					ParentID:   4,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     RectReport{X: 160, Y: 104, W: 132, H: 44},
				},
			},
			LayoutPasses: []ComponentTreeLayoutPassReport{
				{
					ComponentID: 3,
					Pass:        "initial",
					Bounds:      RectReport{X: 16, Y: 48, W: 288, H: 44},
					Measured:    SizeReport{W: 288, H: 44},
				},
				{
					ComponentID: 3,
					Pass:        "resize",
					Bounds:      RectReport{X: 16, Y: 48, W: 368, H: 44},
					Measured:    SizeReport{W: 368, H: 44},
				},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6},
			FocusOrder: []int{3, 5, 6},
			DispatchPaths: []ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 3, X: 40, Y: 72, Path: []int{0, 1, 3}},
				{Event: "click", TargetID: 5, X: 32, Y: 120, Path: []int{0, 1, 4, 5}},
				{Event: "click", TargetID: 6, X: 176, Y: 120, Path: []int{0, 1, 4, 6}},
			},
		},
		ComponentTreeAPI: componentTreeAPIReportForTest(),
		Events: []EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "-1",
					"TextBox.focused":    "false",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "3",
					"TextBox.focused":    "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "SubmitButton"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 3, 0},
				BeforeState: map[string]string{
					"TreeApp.submitted_count": "0",
					"TreeApp.focused_id":      "5",
				},
				AfterState: map[string]string{
					"TreeApp.submitted_count": "1",
					"TreeApp.focused_id":      "5",
				},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         false,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.buffer":     "OK",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.buffer":     "OK",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 13, 320, 200, 6, 0},
				BeforeState: map[string]string{
					"TreeApp.reset_count": "0",
					"TextBox.buffer":      "OK",
					"TreeApp.focused_id":  "6",
				},
				AfterState: map[string]string{
					"TreeApp.reset_count": "1",
					"TextBox.buffer":      "",
					"TreeApp.focused_id":  "6",
				},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 7, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "6"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3"},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 8, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           10,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     9,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 9, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           11,
				Kind:            "resize",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     10,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 10, 0},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.bounds.w":   "288",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.bounds.w":   "368",
				},
			},
		},
		Frames: []FrameReport{
			{
				Order:     1,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
				Presented: true,
			},
			{
				Order:     2,
				Width:     400,
				Height:    240,
				Stride:    1600,
				Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
				Presented: true,
			},
		},
		StateTransitions: []StateTransitionReport{
			{
				Order:     1,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "3",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "OK",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "3",
				After:     "5",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "TreeApp",
				Field:     "submitted_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     5,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "5",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     6,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "OK",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     7,
				Component: "TreeApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     8,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "6",
				After:     "3",
				Cause:     "tab",
			},
			{
				Order:     9,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "3",
				After:     "5",
				Cause:     "tab",
			},
			{
				Order:     10,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "5",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     11,
				Component: "TreeApp",
				Field:     "TextBox.bounds.w",
				Before:    "288",
				After:     "368",
				Cause:     "resize",
			},
		},
		Cases: []CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree text routed to focused TextBox",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree button action dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree api builder node creation",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api parent child invariants",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api layout helper dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree api focus helper traversal",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api dispatch path helper",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api no manual bookkeeping",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal component tree report: %v", err)
	}
	return raw
}
func componentTreeAPIReportForTest() *ComponentTreeAPIReport {
	return &ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/toolkit/surface_tree_app.tetra",
		ManualBookkeeping: false,
		Builder: ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         7,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_row", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []ComponentTreeAPIHitTestReport{
			{Helper: "tree_hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 3}},
			{
				Helper: "tree_hit_test",
				X:      176,
				Y:      120,
				Target: "ResetButton",
				Path:   []int{0, 1, 4, 6},
			},
		},
		DispatchPaths: []ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_build_dispatch_path", Target: "SubmitButton", Path: []int{0, 1, 4, 5}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
	}
}

// ---- report_fixture_test.go ----

func componentMap(
	id string,
	typ string,
	parent string,
	bounds RectReport,
	state map[string]string,
) map[string]any {
	value := map[string]any{
		"id":        id,
		"type":      typ,
		"bounds":    rectMap(bounds),
		"abilities": []any{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		"state":     stringMapAny(state),
	}
	if parent != "" {
		value["parent"] = parent
	}
	return value
}

func treeNodeMap(
	id int,
	name string,
	kind string,
	parentID int,
	childIndex int,
	firstChild int,
	childCount int,
	focusable bool,
	bounds RectReport,
) map[string]any {
	return map[string]any{
		"id":          id,
		"name":        name,
		"kind":        kind,
		"parent_id":   parentID,
		"child_index": childIndex,
		"first_child": firstChild,
		"child_count": childCount,
		"focusable":   focusable,
		"bounds":      rectMap(bounds),
	}
}
func rectMap(rect RectReport) map[string]any {
	return map[string]any{"x": rect.X, "y": rect.Y, "w": rect.W, "h": rect.H}
}

func toolkitWidgetMap(
	name string,
	kind string,
	nodeID int,
	role string,
	reusable bool,
) map[string]any {
	value := map[string]any{
		"name":                  name,
		"kind":                  kind,
		"node_id":               nodeID,
		"reusable":              reusable,
		"ordinary_tetra_struct": true,
	}
	if role != "" {
		if kind == "Button" {
			value["action"] = role
		} else {
			value["role"] = role
		}
	}
	if kind == "TextBox" {
		value["editable"] = true
	}
	return value
}

func eventMap(
	order int,
	kind string,
	target string,
	path []any,
	x int,
	y int,
	key int,
	width int,
	height int,
	before map[string]string,
	after map[string]string,
) map[string]any {
	return map[string]any{
		"order": order, "kind": kind, "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": x, "y": y, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{5, x, y, 1, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func keyEventMap(
	order int,
	target string,
	path []any,
	key int,
	width int,
	height int,
	before map[string]string,
	after map[string]string,
) map[string]any {
	return map[string]any{
		"order": order, "kind": "key_down", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{6, 0, 0, 0, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func textEventMap(
	order int,
	target string,
	path []any,
	textLen int,
	textHex string,
	width int,
	height int,
	before map[string]string,
	after map[string]string,
) map[string]any {
	return map[string]any{
		"order": order, "kind": "text_input", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "text_len": textLen, "text_bytes_hex": textHex,
		"buffer_slots": []any{8, 0, 0, 0, 0, width, height, order - 1, textLen},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func resizeEventMap(
	order int,
	target string,
	path []any,
	width int,
	height int,
	before map[string]string,
	after map[string]string,
) map[string]any {
	return map[string]any{
		"order": order, "kind": "resize", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{2, 0, 0, 0, 0, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}
func stringMapAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func treeComponent(
	id string,
	typ string,
	parent string,
	bounds RectReport,
	state map[string]string,
) ComponentReport {
	return ComponentReport{
		ID:        id,
		Type:      typ,
		Parent:    parent,
		Bounds:    bounds,
		Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		State:     state,
	}
}
func intPtrForTest(v int) *int {
	return &v
}

// ---- report_platform_release_test.go ----

func TestValidateSurfaceBrowserReleaseReport(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceBrowserReleaseRequiresFirstClassBrowserSurfaceEvidence(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "browser_surface")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected browser release without first-class browser_surface evidence to fail")
	}
	if !strings.Contains(err.Error(), "browser_surface") {
		t.Fatalf("error = %v, want browser_surface diagnostic", err)
	}
}
func TestValidateSurfaceBrowserReleaseRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "starter loader level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "wasm32-web-compiler-owned-loader"
				host["backend"] = "node-surface-host"
				host["native_input"] = false
			},
			want: "browser release host_evidence.level",
		},
		{
			name: "missing browser clipboard",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_clipboard"] = false
			},
			want: "browser_clipboard",
		},
		{
			name: "missing composition trace",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_composition"] = false
			},
			want: "browser_composition",
		},
		{
			name: "missing accessibility snapshot",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_accessibility_snapshot"] = false
			},
			want: "browser_accessibility_snapshot",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected browser release fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceLinuxReleaseWindowReport(t *testing.T) {
	raw := validLinuxReleaseWindowSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceLinuxReleaseWindowRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "memfd starter level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-memfd-starter"
				host["backend"] = "memfd-rgba"
				host["real_window"] = false
				host["native_input"] = false
			},
			want: "linux release host_evidence.level",
		},
		{
			name: "old real window level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-real-window"
				host["backend"] = "wayland-shm-rgba"
			},
			want: "linux release host_evidence.level",
		},
		{
			name: "missing clipboard",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["clipboard"] = false
			},
			want: "clipboard",
		},
		{
			name: "missing composition",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["composition"] = false
			},
			want: "composition",
		},
		{
			name: "missing accessibility bridge",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["accessibility_bridge"] = false
			},
			want: "accessibility_bridge",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxReleaseWindowSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected linux release fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceLinuxAppShellReport(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP17SecurityPermissions(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "security_permissions")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P17 security permissions to fail")
	}
	if !strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsPermissionBypassForBlockedFeatures(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		security := report["security_permissions"].(map[string]any)
		capabilities := security["capabilities"].([]any)
		for _, capability := range capabilities {
			row := capability.(map[string]any)
			if row["name"] == "notification" {
				row["status"] = "allowed_with_policy"
				row["allowed"] = true
				row["blocked_reason"] = ""
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected permission bypass for blocked notification feature to fail")
	}
	if !strings.Contains(err.Error(), "notification") ||
		!strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions notification diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP18PerformanceBudget(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "surface_performance_budget")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P18 performance budget to fail")
	}
	if !strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want surface_performance_budget diagnostic", err)
	}
}
func TestValidateSurfacePerformanceBudgetRejectsFasterThanElectronClaim(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		budget := report["surface_performance_budget"].(map[string]any)
		budget["performance_claim"] = "faster than Electron"
		budget["methodology"].(map[string]any)["electron_comparison"] = "faster than Electron"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected fake faster-than-Electron claim to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "faster than electron") &&
		!strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want faster than Electron performance diagnostic", err)
	}
}
func TestValidateSurfacePerformanceBudgetRejectsMissingPeakMemoryField(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		budget := report["surface_performance_budget"].(map[string]any)
		memory := budget["memory"].(map[string]any)
		delete(memory, "peak_rss_bytes")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing peak memory field to fail")
	}
	if !strings.Contains(err.Error(), "peak_rss_bytes") {
		t.Fatalf("error = %v, want peak_rss_bytes diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP16FeatureLedgerRows(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		appShell["shell_features"] = withoutLinuxAppShellFeature(
			p16LinuxAppShellFeaturesForTest(),
			"tray",
		)
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P16 tray ledger row to fail")
	}
	if !strings.Contains(err.Error(), "tray") {
		t.Fatalf("error = %v, want tray diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsTrayClaimWithoutTargetEvidence(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		features := p16LinuxAppShellFeaturesForTest()
		for _, feature := range features {
			row := feature.(map[string]any)
			if row["name"] == "tray" {
				row["status"] = "scoped_adapter"
				row["claimed"] = true
				row["blocked_reason"] = ""
			}
		}
		appShell["shell_features"] = features
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected tray claim without target evidence to fail")
	}
	if !strings.Contains(err.Error(), "tray") || !strings.Contains(err.Error(), "target evidence") {
		t.Fatalf("error = %v, want tray target evidence diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "memfd starter host level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-memfd-starter"
				host["backend"] = "memfd-rgba"
				host["real_window"] = false
				host["native_input"] = false
			},
			want: "linux app-shell host_evidence.level",
		},
		{
			name: "missing reopen lifecycle",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				appShell["window_lifecycle"] = []any{
					map[string]any{
						"order":      1,
						"window_id":  "notes-main",
						"operation":  "open",
						"host_trace": true,
						"pass":       true,
					},
					map[string]any{
						"order":      2,
						"window_id":  "notes-main",
						"operation":  "close",
						"host_trace": true,
						"pass":       true,
					},
				}
			},
			want: "reopen",
		},
		{
			name: "native widget UI substitute",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				appShell["negative_guards"].(map[string]any)["no_gtk"] = false
			},
			want: "GTK/Qt/native widget UI",
		},
		{
			name: "file dialog claimed without blocked pass",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				features := appShell["shell_features"].([]any)
				for _, feature := range features {
					row := feature.(map[string]any)
					if row["name"] == "file_dialog" {
						row["status"] = "claimed-native-dialog"
						row["claimed"] = true
					}
				}
			},
			want: "file_dialog",
		},
		{
			name: "missing host trace artifact",
			mutate: func(report map[string]any) {
				artifacts := report["artifacts"].([]any)
				report["artifacts"] = artifacts[:1]
			},
			want: "linux-app-shell-host-trace",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxAppShellSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected linux app-shell fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func validWASM32WebReleaseBrowserSurfaceReportJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(
		validHeadlessProductionToolkitSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode base production toolkit report: %v", err)
	}
	report["target"] = "wasm32-web"
	report["runtime"] = "surface-wasm32-web"
	report["source"] = "examples/surface/release/surface_release_form.tetra"
	report["host_evidence"] = map[string]any{
		"level":                          "wasm32-web-browser-canvas-release-v1",
		"backend":                        "browser-canvas-rgba-accessible",
		"framebuffer":                    true,
		"real_window":                    false,
		"native_input":                   true,
		"browser_canvas":                 true,
		"browser_input":                  true,
		"browser_clipboard":              true,
		"browser_clipboard_harness":      "deterministic-browser-clipboard-v1",
		"browser_composition":            true,
		"browser_accessibility_snapshot": true,
		"browser_accessibility_mirror":   true,
		"user_facing_platform_widgets":   false,
	}
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target wasm32-web " +
				"examples/surface/release/surface_release_form.tetra -o " +
				"/tmp/surface-artifacts/surface-release-form.wasm"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name": "surface wasm32-web browser canvas component app",
			"kind": "app",
			"path": ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> scenario=release-browser " +
				"wasm=/tmp/surface-artifacts/surface-release-form.wasm"),
			"ran":                true,
			"pass":               true,
			"exit_code":          0,
			"expected_exit_code": 0,
		},
		map[string]any{
			"name": "surface wasm32-web import validator",
			"kind": "runtime",
			"path": ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web /tmp/surface-artifacts/surface-release-form.wasm"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface wasm32-web browser canvas runtime",
			"kind":      "runtime",
			"path":      "Chromium release browser fixture",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name": "surface wasm32-web browser canvas trace",
			"kind": "runtime",
			"path": ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
				"1/surface-browser-canvas-runner?scenario=release-browser"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-release-form.wasm",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   9604,
		},
		map[string]any{
			"kind":   "compiler-owned-loader",
			"path":   "/tmp/surface-artifacts/surface-release-form.mjs",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size":   4939,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   4096,
		},
	}
	report["browser_surface"] = validBrowserSurfaceEvidenceMap()
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	cases := make([]any, 0, len(report["cases"].([]any)))
	for _, item := range report["cases"].([]any) {
		row, ok := item.(map[string]any)
		if !ok {
			cases = append(cases, item)
			continue
		}
		name, _ := row["name"].(string)
		if strings.Contains(strings.ToLower(name), "headless") {
			continue
		}
		cases = append(cases, item)
	}
	report["cases"] = append(
		cases,
		map[string]any{
			"name": "wasm32-web Surface Host ABI imports",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "compiler-owned wasm Surface loader",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas surface",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas RGBA readback",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas pointer input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas keyboard input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas resize input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "wasm32-web browser canvas text input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "compiler-owned browser canvas Surface host",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release Surface v1 schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release Chromium canvas readback",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release native pointer keyboard text resize",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release deterministic clipboard harness",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release composition trace",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "browser release accessibility snapshot mirror",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name":           "browser release forbidden web sidecar rejection",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "forbidden web sidecar rejected",
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal browser release report: %v", err)
	}
	return raw
}
func validBrowserSurfaceEvidenceMap() map[string]any {
	return map[string]any{
		"schema":                "tetra.surface.browser-surface.v1",
		"browser_surface_level": "browser-canvas-release-v1",
		"release_scope":         "surface-v1-linux-web",
		"source":                "examples/surface/release/surface_release_form.tetra",
		"host_adapter":          "compiler-owned-browser-canvas-host",
		"production_claim":      true,
		"experimental":          false,
		"compiler_owned_boot":   true,
		"dom_host_canvas_only":  true,
		"canvas": map[string]any{
			"opened":        true,
			"readback":      true,
			"width":         560,
			"height":        420,
			"frame_order":   5,
			"artifact_kind": "runner-trace",
			"pass":          true,
		},
		"input": map[string]any{
			"pointer":       true,
			"keyboard":      true,
			"text":          true,
			"resize":        true,
			"host_trace":    true,
			"native_events": []any{"pointerup", "keydown", "beforeinput", "resize"},
			"pass":          true,
		},
		"clipboard": map[string]any{
			"harness":    "deterministic-browser-clipboard-v1",
			"read":       true,
			"write":      true,
			"owned_copy": true,
			"bytes":      13,
			"pass":       true,
		},
		"composition": map[string]any{
			"start":  true,
			"update": true,
			"commit": true,
			"cancel": true,
			"pass":   true,
		},
		"accessibility": map[string]any{
			"snapshot":       true,
			"mirror":         true,
			"compiler_owned": true,
			"bounds":         true,
			"focus":          true,
			"roles":          []any{"root", "textbox", "checkbox", "button", "status"},
			"dom_visual_ui":  false,
			"user_js":        false,
			"pass":           true,
		},
		"host_traces": []any{
			map[string]any{
				"name":          "browser-canvas",
				"artifact_kind": "runner-trace",
				"path":          "/tmp/surface-artifacts/surface-runner-trace.json",
				"pass":          true,
			},
		},
		"negative_guards": map[string]any{
			"no_dom_app_ui_tree":     true,
			"no_user_js_app_logic":   true,
			"no_node_only_promotion": true,
			"no_legacy_sidecars":     true,
			"no_react_runtime":       true,
			"no_platform_widgets":    true,
		},
	}
}
func validLinuxReleaseWindowSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(
		validHeadlessProductionToolkitSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode base production toolkit report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface/release/surface_release_form.tetra"
	report["host_evidence"] = map[string]any{
		"level":                        "linux-x64-release-window-v1",
		"backend":                      "wayland-shm-rgba-release-v1",
		"framebuffer":                  true,
		"real_window":                  true,
		"native_input":                 true,
		"text_input":                   true,
		"clipboard":                    true,
		"composition":                  true,
		"accessibility_bridge":         true,
		"user_facing_platform_widgets": false,
	}
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/release/surface_release_form.tetra -o " +
				"/tmp/surface-artifacts/surface-release-form"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-release-form",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":               "surface linux-x64 real-window probe",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-release-window-probe",
			"ran":                true,
			"pass":               true,
			"exit_code":          42,
			"expected_exit_code": 42,
		},
		map[string]any{
			"name":      "surface linux-x64 release clipboard harness",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-clipboard-harness.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 release composition harness",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-composition-harness.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux accessibility host bridge",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux accessibility platform probe",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke --mode linux-x64-release-window",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-release-form",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   90001,
		},
		map[string]any{
			"kind":   "linux-accessibility-host-bridge",
			"path":   "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size":   4096,
		},
		map[string]any{
			"kind":   "linux-accessibility-platform-probe",
			"path":   "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   4096,
		},
	}
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	events := report["events"].([]any)
	report["events"] = append(events, map[string]any{
		"order": 14, "kind": "close", "target_component": "SurfaceReleaseFormApp",
		"dispatch_path": []any{"SurfaceReleaseFormApp"}, "handled": true, "pass": true,
		"width": 560, "height": 420, "timestamp_ms": 13,
		"buffer_slots": []any{9, 0, 0, 0, 0, 560, 420, 13, 0},
		"before_state": map[string]any{"SurfaceReleaseFormApp.open": "true"},
		"after_state":  map[string]any{"SurfaceReleaseFormApp.open": "false"},
	})
	cases := make([]any, 0, len(report["cases"].([]any)))
	for _, item := range report["cases"].([]any) {
		row, ok := item.(map[string]any)
		if !ok {
			cases = append(cases, item)
			continue
		}
		name, _ := row["name"].(string)
		if strings.Contains(strings.ToLower(name), "headless") {
			continue
		}
		cases = append(cases, item)
	}
	report["cases"] = append(
		cases,
		map[string]any{
			"name": "linux-x64 real-window surface",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 native input event pump",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window resize event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window close event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility platform bridge v1 schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux accessibility host bridge export",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "accessibility release honest screen reader evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release window v1 schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release real window presented frame",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release native pointer key text resize close",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release clipboard harness",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release composition harness",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release accessibility bridge probe",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name":           "linux release forbids memfd starter promotion",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "memfd starter rejected",
		},
	)
	report["accessibility_tree"] = releaseWindowAccessibilityTreeMap()
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux release window report: %v", err)
	}
	return raw
}
func validLinuxAppShellSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode base headless report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface/toolkit/surface_linux_app_shell_notes.tetra"
	report["host_evidence"] = map[string]any{
		"level":                        "linux-x64-release-window-v1",
		"backend":                      "wayland-shm-rgba-release-v1",
		"framebuffer":                  true,
		"real_window":                  true,
		"native_input":                 true,
		"text_input":                   true,
		"clipboard":                    true,
		"composition":                  true,
		"accessibility_bridge":         true,
		"user_facing_platform_widgets": false,
	}
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_linux_app_shell_notes.tetra" +
				" -o /tmp/surface-artifacts/surface-linux-app-shell-notes"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-linux-app-shell-notes",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":               "surface linux-x64 real-window probe",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-linux-app-shell-window-probe",
			"ran":                true,
			"pass":               true,
			"exit_code":          42,
			"expected_exit_code": 42,
		},
		map[string]any{
			"name":      "surface linux app-shell host trace",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux app-shell window trace",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 release clipboard harness",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-clipboard-harness.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 release composition harness",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-composition-harness.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux accessibility platform probe",
			"kind":      "runtime",
			"path":      "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-linux-app-shell-notes",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   90001,
		},
		map[string]any{
			"kind":   "linux-app-shell-host-trace",
			"path":   "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size":   4096,
		},
		map[string]any{
			"kind":   "linux-app-shell-window-trace",
			"path":   "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   4096,
		},
		map[string]any{
			"kind":   "linux-accessibility-platform-probe",
			"path":   "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			"sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"size":   4096,
		},
	}
	report["artifact_scan"] = map[string]any{
		"root":            "/tmp/surface-artifacts",
		"files_checked":   4,
		"forbidden_paths": []any{},
		"pass":            true,
	}
	report["components"] = []any{
		componentMap(
			"NotesShellApp",
			"examples.surface.toolkit.surface_linux_app_shell_notes.NotesShellApp",
			"",
			RectReport{X: 0, Y: 0, W: 720, H: 540},
			map[string]string{
				"open_windows":       "2",
				"focused_window":     "notes-main",
				"accessibility_role": "application",
			},
		),
		componentMap(
			"NotesMainWindow",
			"examples.surface.toolkit.surface_linux_app_shell_notes.NotesMainWindow",
			"NotesShellApp",
			RectReport{X: 0, Y: 0, W: 560, H: 420},
			map[string]string{
				"title":              "Notes",
				"lifecycle":          "reopened",
				"dpi_scale_milli":    "1250",
				"cursor":             "text",
				"accessibility_role": "document",
			},
		),
		componentMap(
			"NotesInspectorWindow",
			"examples.surface.toolkit.surface_linux_app_shell_notes.NotesInspectorWindow",
			"NotesShellApp",
			RectReport{X: 24, Y: 24, W: 320, H: 240},
			map[string]string{
				"title":              "Inspector",
				"lifecycle":          "open",
				"dpi_scale_milli":    "1000",
				"cursor":             "pointer",
				"accessibility_role": "panel",
			},
		),
	}
	report["events"] = []any{
		map[string]any{
			"order":            1,
			"kind":             "mouse_up",
			"target_component": "NotesMainWindow",
			"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
			"handled":          true,
			"pass":             true,
			"x":                40,
			"y":                72,
			"key":              0,
			"width":            560,
			"height":           420,
			"timestamp_ms":     0,
			"buffer_slots":     []any{5, 40, 72, 1, 0, 560, 420, 0, 0},
			"before_state":     map[string]any{"NotesShellApp.focused_window": ""},
			"after_state":      map[string]any{"NotesShellApp.focused_window": "notes-main"},
		},
		map[string]any{
			"order":            2,
			"kind":             "key_down",
			"target_component": "NotesMainWindow",
			"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
			"handled":          true,
			"pass":             true,
			"x":                0,
			"y":                0,
			"key":              78,
			"width":            560,
			"height":           420,
			"timestamp_ms":     2,
			"buffer_slots":     []any{6, 0, 0, 1, 78, 560, 420, 2, 0},
			"before_state":     map[string]any{"NotesMainWindow.shortcut": ""},
			"after_state":      map[string]any{"NotesMainWindow.shortcut": "new-note"},
		},
		map[string]any{
			"order":            3,
			"kind":             "text_input",
			"target_component": "NotesMainWindow",
			"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
			"handled":          true,
			"pass":             true,
			"x":                0,
			"y":                0,
			"key":              0,
			"width":            560,
			"height":           420,
			"timestamp_ms":     3,
			"text_len":         5,
			"text_bytes_hex":   "4e6f746573",
			"buffer_slots":     []any{8, 0, 0, 0, 0, 560, 420, 3, 5},
			"before_state":     map[string]any{"NotesMainWindow.buffer": ""},
			"after_state":      map[string]any{"NotesMainWindow.buffer": "Notes"},
		},
		map[string]any{
			"order":            4,
			"kind":             "resize",
			"target_component": "NotesMainWindow",
			"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
			"handled":          true,
			"pass":             true,
			"width":            720,
			"height":           540,
			"timestamp_ms":     4,
			"buffer_slots":     []any{7, 0, 0, 0, 0, 720, 540, 4, 0},
			"before_state": map[string]any{
				"NotesMainWindow.size": "560x420",
				"NotesMainWindow.dpi":  "1000",
			},
			"after_state": map[string]any{
				"NotesMainWindow.size": "720x540",
				"NotesMainWindow.dpi":  "1250",
			},
		},
		map[string]any{
			"order":            5,
			"kind":             "close",
			"target_component": "NotesInspectorWindow",
			"dispatch_path":    []any{"NotesShellApp", "NotesInspectorWindow"},
			"handled":          true,
			"pass":             true,
			"width":            320,
			"height":           240,
			"timestamp_ms":     5,
			"buffer_slots":     []any{9, 0, 0, 0, 0, 320, 240, 5, 0},
			"before_state":     map[string]any{"NotesInspectorWindow.open": "true"},
			"after_state":      map[string]any{"NotesInspectorWindow.open": "false"},
		},
	}
	report["frames"] = []any{
		map[string]any{
			"order":     1,
			"width":     400,
			"height":    240,
			"stride":    1600,
			"checksum":  "1111111111111111111111111111111111111111111111111111111111111111",
			"presented": true,
		},
		map[string]any{
			"order":     5,
			"width":     560,
			"height":    420,
			"stride":    2240,
			"checksum":  "2222222222222222222222222222222222222222222222222222222222222222",
			"presented": true,
		},
		map[string]any{
			"order":     6,
			"width":     720,
			"height":    540,
			"stride":    2880,
			"checksum":  "3333333333333333333333333333333333333333333333333333333333333333",
			"presented": true,
		},
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "NotesShellApp",
			"field":     "focused_window",
			"before":    "",
			"after":     "notes-main",
			"cause":     "lifecycle.open",
		},
		map[string]any{
			"order":     2,
			"component": "NotesInspectorWindow",
			"field":     "open",
			"before":    "true",
			"after":     "false",
			"cause":     "lifecycle.close",
		},
		map[string]any{
			"order":     3,
			"component": "NotesMainWindow",
			"field":     "size",
			"before":    "560x420",
			"after":     "720x540",
			"cause":     "resize",
		},
	}
	report["cases"] = []any{
		map[string]any{
			"name": "pure Tetra component app",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "host-provided pointer event dispatch",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "host event buffer poll_event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "pre/post event frame sequence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "component hierarchy dispatch",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "component text input scalar dispatch",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "host text payload buffer",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "component focus dispatch",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "component accessibility metadata",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "no legacy UI sidecar artifacts",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{"name": "state transition", "kind": "positive", "ran": true, "pass": true},
		map[string]any{
			"name":           "reject legacy UI evidence",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "legacy UI evidence rejected",
		},
		map[string]any{
			"name": "linux-x64 real-window surface",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 native input event pump",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window resize event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux-x64 real-window close event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release real window presented frame",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux release accessibility bridge probe",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell v1 schema",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell lifecycle open close reopen",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell multi-window notes reference",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell resize dpi cursor trace",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell clipboard ime accessibility adapters",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell file dialog notification blocked-pass",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell electron feature ledger",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell dialog file picker tray blocked-pass",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "linux app-shell crash error report scoped adapters",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name":           "linux app-shell rejects GTK Qt native widget UI",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "native widget UI rejected",
		},
		map[string]any{
			"name":           "linux app-shell no Electron React DOM application scripting",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "runtime substitute rejected",
		},
	}
	report["linux_app_shell"] = validLinuxAppShellEvidenceMap()
	report["security_permissions"] = validSurfaceSecurityPermissionsMap(
		p16LinuxAppShellFeaturesForTest(),
	)
	report["surface_performance_budget"] = validSurfacePerformanceBudgetMap(
		"linux-x64",
		"surface-linux-x64",
		"examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux app-shell report: %v", err)
	}
	return raw
}

func validSurfacePerformanceBudgetMap(
	target string,
	runtimeName string,
	source string,
) map[string]any {
	return map[string]any{
		"schema":            "tetra.surface.performance-budget.v1",
		"model":             "surface-performance-budget-v1",
		"release_scope":     "surface-v1-linux-web",
		"source":            source,
		"target":            target,
		"runtime":           runtimeName,
		"production_claim":  true,
		"experimental":      false,
		"git_head":          "0123456789abcdef0123456789abcdef01234567",
		"performance_claim": "none",
		"startup": map[string]any{
			"launch_to_first_frame_ms": 18,
			"budget_ms":                250,
			"trace":                    "local-startup-trace-v1",
			"pass":                     true,
		},
		"frame": map[string]any{
			"frame_count":     3,
			"p50_build_ms":    4,
			"p95_build_ms":    7,
			"p50_present_ms":  3,
			"p95_present_ms":  6,
			"budget_ms":       16,
			"idle_loop_count": 24,
			"work_loop_count": 6,
			"pass":            true,
		},
		"scene": map[string]any{
			"block_count":            3,
			"recipe_expansion_count": 0,
			"paint_command_count":    10,
			"layout_pass_count":      4,
			"text_run_count":         2,
		},
		"memory": map[string]any{
			"glyph_cache_bytes":        4096,
			"asset_cache_bytes":        5376,
			"layout_cache_bytes":       4096,
			"paint_cache_bytes":        10240,
			"framebuffer_peak_bytes":   1555200,
			"framebuffer_total_bytes":  2880000,
			"rss_measured":             false,
			"peak_rss_bytes":           0,
			"allocation_count":         42,
			"allocation_bytes":         2903808,
			"bounded_caches":           true,
			"unbounded_cache_rejected": true,
			"pass":                     true,
		},
		"binary": map[string]any{
			"artifact_path": "/tmp/surface-artifacts/surface-linux-app-shell-notes",
			"size_bytes":    90001,
			"budget_bytes":  16777216,
			"pass":          true,
		},
		"cpu_power_proxy": map[string]any{
			"idle_loop_count":     24,
			"work_loop_count":     6,
			"idle_frame_count":    2,
			"work_frame_count":    1,
			"real_power_measured": false,
			"pass":                true,
		},
		"cache": map[string]any{
			"glyph_cache_budget_bytes":  65536,
			"asset_cache_budget_bytes":  65536,
			"layout_cache_budget_bytes": 65536,
			"paint_cache_budget_bytes":  65536,
			"total_cache_bytes":         23808,
			"total_cache_budget_bytes":  262144,
			"eviction":                  "bounded-lru",
			"pass":                      true,
		},
		"methodology": map[string]any{
			"kind":                "local-deterministic-budget-v1",
			"electron_comparison": "none",
			"official_benchmark":  false,
			"cross_machine":       false,
			"fair_comparison_required_for_electron_claim": true,
		},
		"unsupported_claims": []any{
			"faster-than-electron",
			"lower-power-than-electron",
			"official-benchmark-result",
			"cross-machine-benchmark",
			"electron-parity-performance",
		},
		"negative_guards": map[string]any{
			"bounded_caches":                true,
			"unbounded_cache_rejected":      true,
			"stale_report_rejected":         true,
			"no_faster_than_electron_claim": true,
			"no_benchmark_parity_claim":     true,
			"peak_memory_field_required":    true,
			"no_official_benchmark_claim":   true,
		},
	}
}
func validLinuxAppShellEvidenceMap() map[string]any {
	return map[string]any{
		"schema":           "tetra.surface.linux-app-shell.v1",
		"app_shell_level":  "linux-app-shell-subset-v1",
		"release_scope":    "surface-v1-linux-web",
		"source":           "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"module":           "lib.core.surface_app_shell",
		"host_adapter":     "wayland-shm-rgba-release-v1",
		"production_claim": true,
		"experimental":     false,
		"window_lifecycle": []any{
			map[string]any{
				"order":      1,
				"window_id":  "notes-main",
				"operation":  "open",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      2,
				"window_id":  "notes-inspector",
				"operation":  "open",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      3,
				"window_id":  "notes-inspector",
				"operation":  "close",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      4,
				"window_id":  "notes-inspector",
				"operation":  "reopen",
				"host_trace": true,
				"pass":       true,
			},
		},
		"windows": []any{
			map[string]any{
				"id":              "notes-main",
				"title":           "Notes",
				"role":            "primary",
				"block_root":      "NotesMainWindow",
				"real_window":     true,
				"presented":       true,
				"width":           720,
				"height":          540,
				"dpi_scale_milli": 1250,
			},
			map[string]any{
				"id":              "notes-inspector",
				"title":           "Inspector",
				"role":            "secondary",
				"block_root":      "NotesInspectorWindow",
				"real_window":     true,
				"presented":       true,
				"width":           320,
				"height":          240,
				"dpi_scale_milli": 1000,
			},
		},
		"resize_dpi": []any{
			map[string]any{
				"window_id":       "notes-main",
				"operation":       "resize",
				"before_width":    560,
				"before_height":   420,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
				"host_trace":      true,
				"pass":            true,
			},
			map[string]any{
				"window_id":       "notes-main",
				"operation":       "dpi_scale",
				"before_width":    720,
				"before_height":   540,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
				"host_trace":      true,
				"pass":            true,
			},
		},
		"cursor_transitions": []any{
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "pointer",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "text",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "resize",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
		},
		"clipboard": map[string]any{
			"level":         "clipboard-text-v1",
			"host_trace":    true,
			"artifact_kind": "linux-app-shell-host-trace",
			"read":          true,
			"write":         true,
			"pass":          true,
		},
		"ime": map[string]any{
			"level":         "composition-baseline-v1",
			"host_trace":    true,
			"artifact_kind": "linux-app-shell-host-trace",
			"start":         true,
			"update":        true,
			"commit":        true,
			"cancel":        true,
			"pass":          true,
		},
		"accessibility": map[string]any{
			"level":           "platform-bridge-v1",
			"host_trace":      true,
			"artifact_kind":   "linux-accessibility-platform-probe",
			"metadata_tree":   true,
			"platform_export": true,
			"pass":            true,
		},
		"shell_features": p16LinuxAppShellFeaturesForTest(),
		"host_traces": []any{
			map[string]any{
				"name":          "lifecycle",
				"artifact_kind": "linux-app-shell-host-trace",
				"path":          "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json",
				"pass":          true,
			},
			map[string]any{
				"name":          "windows",
				"artifact_kind": "linux-app-shell-window-trace",
				"path":          "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json",
				"pass":          true,
			},
			map[string]any{
				"name":          "accessibility",
				"artifact_kind": "linux-accessibility-platform-probe",
				"path":          "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
				"pass":          true,
			},
		},
		"negative_guards": map[string]any{
			"no_gtk":              true,
			"no_qt":               true,
			"no_native_widgets":   true,
			"no_electron_runtime": true,
			"no_react_runtime":    true,
			"no_dom_ui":           true,
			"no_user_js":          true,
			"no_platform_widgets": true,
		},
	}
}
func p16LinuxAppShellFeaturesForTest() []any {
	return []any{
		map[string]any{
			"name":                "app_menu",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "window_lifecycle",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "multi_window",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "clipboard",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "ime",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "accessibility_bridge",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "crash_recovery",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "error_report",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "dialog",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host dialog unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "file_dialog",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host file dialog unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "file_picker",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host file picker unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "notification",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host notification unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "tray",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host tray unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "deep_link",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host deep link unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
	}
}
func validSurfaceSecurityPermissionsMap(features []any) map[string]any {
	capabilities := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		name := row["name"].(string)
		status, allowed := mapSecurityCapabilityStatus(row["status"].(string))
		blockedReason := ""
		if value, ok := row["blocked_reason"].(string); ok {
			blockedReason = value
		}
		capabilities = append(capabilities, map[string]any{
			"name":               name,
			"source_feature":     name,
			"status":             status,
			"allowed":            allowed,
			"capability_checked": true,
			"host_trace":         true,
			"policy":             "surface-app-shell-capability-policy-v1",
			"evidence":           "linux-app-shell-host-trace",
			"blocked_reason":     blockedReason,
			"pass":               true,
		})
	}
	return map[string]any{
		"schema":                        "tetra.surface.security-permission.v1",
		"model":                         "surface-security-permission-v1",
		"release_scope":                 "surface-v1-linux-web",
		"source":                        "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"app_shell_features":            "electron-feature-ledger-v1",
		"production_claim":              true,
		"experimental":                  false,
		"default_deny":                  true,
		"shell_feature_policy_enforced": true,
		"capabilities":                  capabilities,
		"permissions": []any{
			map[string]any{
				"name":               "filesystem",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "ambient filesystem denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
			map[string]any{
				"name":               "network",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "ambient network denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
			map[string]any{
				"name":               "clipboard",
				"status":             "allowed_with_policy",
				"allowed":            true,
				"capability_checked": true,
				"blocked_reason":     "",
				"evidence":           "linux-app-shell-host-trace",
				"pass":               true,
			},
			map[string]any{
				"name":               "notifications",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "notification target evidence absent",
				"evidence":           "blocked-pass-nonclaim",
				"pass":               true,
			},
			map[string]any{
				"name":               "dialogs",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "dialog target evidence absent",
				"evidence":           "blocked-pass-nonclaim",
				"pass":               true,
			},
			map[string]any{
				"name":               "shell_open_url",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "shell open-url denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
		},
		"process_boundaries": []any{
			map[string]any{
				"name":               "surface_app_to_host_abi",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
			map[string]any{
				"name":               "linux_app_shell_host_adapter",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
			map[string]any{
				"name":               "browser_canvas_host",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
		},
		"asset_safety": []any{
			map[string]any{
				"kind":                  "font",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      1048576,
				"network_fetch_allowed": false,
				"parser":                "bounded-font-metadata-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
			map[string]any{
				"kind":                  "image",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      2097152,
				"network_fetch_allowed": false,
				"parser":                "bounded-image-header-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
			map[string]any{
				"kind":                  "icon",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      262144,
				"network_fetch_allowed": false,
				"parser":                "bounded-icon-header-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
		},
		"unsupported_claims": []any{
			"unrestricted-filesystem",
			"unrestricted-network",
			"native-permission-prompts",
			"production-notifications",
			"production-dialogs",
			"remote-asset-fetch",
			"electron-node-integration",
		},
		"negative_guards": map[string]any{
			"no_ambient_filesystem":                          true,
			"no_ambient_network":                             true,
			"no_shell_feature_bypass":                        true,
			"no_permissionless_clipboard":                    true,
			"no_notification_dialog_without_target_evidence": true,
			"no_network_asset_fetch":                         true,
			"no_untrusted_font_image_decode":                 true,
			"no_electron_node_integration":                   true,
			"no_user_js_app_logic":                           true,
			"no_dom_app_ui_tree":                             true,
		},
	}
}
func mapSecurityCapabilityStatus(featureStatus string) (string, bool) {
	switch featureStatus {
	case "target_evidenced", "scoped_adapter":
		return "allowed_with_policy", true
	case "blocked_pass":
		return "blocked_nonclaim", false
	default:
		return "unknown", false
	}
}
func withoutLinuxAppShellFeature(features []any, name string) []any {
	filtered := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		if row["name"] == name {
			continue
		}
		filtered = append(filtered, feature)
	}
	return filtered
}

// ---- report_release_test.go ----

func TestValidateSurfaceReleaseSummaryAcceptsScopedLinuxWebCurrent(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf("ValidateReleaseSummary failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceReleaseSummaryAcceptsBlockSystemAndMorphGateMetadata(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary failed with Block-system/Morph gate metadata: %v\n%s",
			err,
			raw,
		)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresProjectTemplates(t *testing.T) {
	withTemplates := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withTemplates)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected project template evidence: %v\n%s",
			err,
			withTemplates,
		)
	}

	missing := strings.Replace(withTemplates, `  "project_templates": "surface-template-smoke-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing project_templates to fail")
	}
	if !strings.Contains(err.Error(), "project_templates") {
		t.Fatalf("error = %v, want project_templates diagnostic", err)
	}

	wrong := strings.Replace(
		withTemplates,
		`"project_templates": "surface-template-smoke-v1"`,
		`"project_templates": "docs-only-template-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong project_templates to fail")
	}
	if !strings.Contains(err.Error(), "project_templates") {
		t.Fatalf("error = %v, want project_templates diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresReferenceApps(t *testing.T) {
	withReferenceApps := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withReferenceApps)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected reference app evidence: %v\n%s",
			err,
			withReferenceApps,
		)
	}

	missing := strings.Replace(
		withReferenceApps,
		`  "reference_apps": "surface-reference-app-suite-v1",
`,
		"",
		1,
	)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing reference_apps to fail")
	}
	if !strings.Contains(err.Error(), "reference_apps") {
		t.Fatalf("error = %v, want reference_apps diagnostic", err)
	}

	wrong := strings.Replace(
		withReferenceApps,
		`"reference_apps": "surface-reference-app-suite-v1"`,
		`"reference_apps": "docs-only-reference-app-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong reference_apps to fail")
	}
	if !strings.Contains(err.Error(), "reference_apps") {
		t.Fatalf("error = %v, want reference_apps diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresSurfacePackage(t *testing.T) {
	withPackage := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withPackage)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected Surface package evidence: %v\n%s",
			err,
			withPackage,
		)
	}

	missing := strings.Replace(withPackage, `  "surface_package": "surface-package-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing surface_package to fail")
	}
	if !strings.Contains(err.Error(), "surface_package") {
		t.Fatalf("error = %v, want surface_package diagnostic", err)
	}

	wrong := strings.Replace(
		withPackage,
		`"surface_package": "surface-package-v1"`,
		`"surface_package": "docs-only-package-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong surface_package to fail")
	}
	if !strings.Contains(err.Error(), "surface_package") {
		t.Fatalf("error = %v, want surface_package diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresCrashReporting(t *testing.T) {
	withCrashReporting := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withCrashReporting)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected crash reporting evidence: %v\n%s",
			err,
			withCrashReporting,
		)
	}

	missing := strings.Replace(withCrashReporting, `  "crash_reporting": "surface-crash-report-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing crash_reporting to fail")
	}
	if !strings.Contains(err.Error(), "crash_reporting") {
		t.Fatalf("error = %v, want crash_reporting diagnostic", err)
	}

	wrong := strings.Replace(
		withCrashReporting,
		`"crash_reporting": "surface-crash-report-v1"`,
		`"crash_reporting": "docs-only-crash-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong crash_reporting to fail")
	}
	if !strings.Contains(err.Error(), "crash_reporting") {
		t.Fatalf("error = %v, want crash_reporting diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresI18nLocalization(t *testing.T) {
	withI18n := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withI18n)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected i18n localization evidence: %v\n%s",
			err,
			withI18n,
		)
	}

	missing := strings.Replace(withI18n, `  "i18n_localization": "surface-i18n-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing i18n_localization to fail")
	}
	if !strings.Contains(err.Error(), "i18n_localization") {
		t.Fatalf("error = %v, want i18n_localization diagnostic", err)
	}

	wrong := strings.Replace(
		withI18n,
		`"i18n_localization": "surface-i18n-v1"`,
		`"i18n_localization": "full-icu-bidi-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong i18n_localization to fail")
	}
	if !strings.Contains(err.Error(), "i18n_localization") {
		t.Fatalf("error = %v, want i18n_localization diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRequiresWidgetMigration(t *testing.T) {
	withMigration := string(validSurfaceReleaseSummaryJSON())
	if err := ValidateReleaseSummary([]byte(withMigration)); err != nil {
		t.Fatalf(
			"ValidateReleaseSummary rejected widget migration evidence: %v\n%s",
			err,
			withMigration,
		)
	}

	missing := strings.Replace(withMigration, `  "widget_migration": "surface-widget-migration-v1",
`, "", 1)
	err := ValidateReleaseSummary([]byte(missing))
	if err == nil {
		t.Fatalf("expected missing widget_migration to fail")
	}
	if !strings.Contains(err.Error(), "widget_migration") {
		t.Fatalf("error = %v, want widget_migration diagnostic", err)
	}

	wrong := strings.Replace(
		withMigration,
		`"widget_migration": "surface-widget-migration-v1"`,
		`"widget_migration": "future-widget-core-claim"`,
		1,
	)
	err = ValidateReleaseSummary([]byte(wrong))
	if err == nil {
		t.Fatalf("expected wrong widget_migration to fail")
	}
	if !strings.Contains(err.Error(), "widget_migration") {
		t.Fatalf("error = %v, want widget_migration diagnostic", err)
	}
}
func TestValidateSurfaceReleaseSummaryRejectsFakePromotionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "missing unsupported targets",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`  "unsupported_targets": ["macos-x64","windows-x64","wasm32-wasi"],`+"\n",
					``,
					1,
				)
			},
			want: "unsupported_targets",
		},
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "unsupported target in supported targets",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"supported_targets": ["headless","linux-x64","wasm32-web"]`,
					`"supported_targets": ["headless","linux-x64","wasm32-web","macos-x64"]`,
					1,
				)
			},
			want: "supported_targets",
		},
		{
			name: "dom ui",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"dom_ui": false`, `"dom_ui": true`, 1)
			},
			want: "dom_ui",
		},
		{
			name: "platform widgets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"platform_widgets": false`, `"platform_widgets": true`, 1)
			},
			want: "platform_widgets",
		},
		{
			name: "missing block system",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "block_system": "block-system",
`, ``, 1)
			},
			want: "block_system",
		},
		{
			name: "wrong block system gate",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"block_system_gate": "tetra.surface.block-system.gate.v1"`,
					`"block_system_gate": "tetra.surface.block-system.fake"`,
					1,
				)
			},
			want: "block_system_gate",
		},
		{
			name: "missing morph",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "morph": "morph-capsule",
`, ``, 1)
			},
			want: "morph",
		},
		{
			name: "wrong morph gate",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"morph_gate": "tetra.surface.morph.gate.v1"`,
					`"morph_gate": "tetra.surface.morph.invalid"`,
					1,
				)
			},
			want: "morph_gate",
		},
		{
			name: "missing app model",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "app_model": "explicit-command-reducer-v1",
`, ``, 1)
			},
			want: "app_model",
		},
		{
			name: "wrong app model",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"app_model": "explicit-command-reducer-v1"`,
					`"app_model": "hidden-state-runtime"`,
					1,
				)
			},
			want: "app_model",
		},
		{
			name: "missing linux app shell",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "linux_app_shell": "linux-app-shell-subset-v1",
  "app_shell_features": "electron-feature-ledger-v1",
`, ``, 1)
			},
			want: "linux_app_shell",
		},
		{
			name: "wrong linux app shell",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"linux_app_shell": "linux-app-shell-subset-v1"`,
					`"linux_app_shell": "native-widget-shell"`,
					1,
				)
			},
			want: "linux_app_shell",
		},
		{
			name: "missing security permissions",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "security_permissions": "surface-security-permission-v1",
`, ``, 1)
			},
			want: "security_permissions",
		},
		{
			name: "wrong security permissions",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"security_permissions": "surface-security-permission-v1"`,
					`"security_permissions": "ambient-network-filesystem"`,
					1,
				)
			},
			want: "security_permissions",
		},
		{
			name: "missing performance budget",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "performance_budget": "surface-performance-budget-v1",
`, ``, 1)
			},
			want: "performance_budget",
		},
		{
			name: "wrong performance budget",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"performance_budget": "surface-performance-budget-v1"`,
					`"performance_budget": "faster-than-electron"`,
					1,
				)
			},
			want: "performance_budget",
		},
		{
			name: "missing developer fast loop",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "developer_fast_loop": "surface-dev-workflow-v1",
`, ``, 1)
			},
			want: "developer_fast_loop",
		},
		{
			name: "wrong developer fast loop",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"developer_fast_loop": "surface-dev-workflow-v1"`,
					`"developer_fast_loop": "hot-reload-claim"`,
					1,
				)
			},
			want: "developer_fast_loop",
		},
		{
			name: "missing inspector",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "inspector": "surface-inspector-v1",
`, ``, 1)
			},
			want: "inspector",
		},
		{
			name: "wrong inspector",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"inspector": "surface-inspector-v1"`,
					`"inspector": "browser-devtools-proxy"`,
					1,
				)
			},
			want: "inspector",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceReleaseSummaryJSON())))
			err := ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected release summary to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceReleaseSummaryRejectsStaleProducerMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing producer",
			mutate: func(report map[string]any) {
				delete(report, "producer")
			},
			want: "producer",
		},
		{
			name: "stale git head",
			mutate: func(report map[string]any) {
				report["git_head"] = "unknown"
			},
			want: "git_head",
		},
		{
			name: "missing command line",
			mutate: func(report map[string]any) {
				delete(report, "command_line")
			},
			want: "command_line",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var report map[string]any
			if err := json.Unmarshal(validSurfaceReleaseSummaryJSON(), &report); err != nil {
				t.Fatalf("decode release summary: %v", err)
			}
			tc.mutate(report)
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal release summary: %v", err)
			}
			err = ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected stale producer metadata to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceTargetHostStatusAcceptsWindowsUnsupportedNonclaim(t *testing.T) {
	raw := validSurfaceWindowsTargetHostStatusJSON()
	if err := ValidateTargetHostStatus(raw); err != nil {
		t.Fatalf("ValidateTargetHostStatus failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTargetHostStatusAcceptsMacOSUnsupportedNonclaim(t *testing.T) {
	raw := validSurfaceMacOSTargetHostStatusJSON()
	if err := ValidateTargetHostStatus(raw); err != nil {
		t.Fatalf("ValidateTargetHostStatus failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsFakeBuildOnlyPromotion(t *testing.T) {
	raw := strings.Replace(
		string(validSurfaceWindowsTargetHostStatusJSON()),
		`"build_only_promotion": false`,
		`"build_only_promotion": true`,
		1,
	)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected build-only promotion to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "build-only") {
		t.Fatalf("error = %v, want build-only diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsLinuxSubstitute(t *testing.T) {
	raw := strings.Replace(
		string(validSurfaceWindowsTargetHostStatusJSON()),
		`"linux_substitute": false`,
		`"linux_substitute": true`,
		1,
	)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected Linux substitute to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "linux substitute") {
		t.Fatalf("error = %v, want Linux substitute diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsProductionClaim(t *testing.T) {
	raw := strings.Replace(
		string(validSurfaceWindowsTargetHostStatusJSON()),
		`"production_claim": false`,
		`"production_claim": true`,
		1,
	)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected Windows production claim to fail without target-host evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "production_claim") {
		t.Fatalf("error = %v, want production_claim diagnostic", err)
	}
}
func TestValidateSurfaceTargetHostStatusRejectsBetaWithoutTargetHostEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validSurfaceWindowsTargetHostStatusJSON()),
		`"status": "unsupported"`,
		`"status": "beta_target_host"`,
		1,
	)
	raw = strings.Replace(raw, `"tier": "UNSUPPORTED"`, `"tier": "BETA_TARGET_HOST"`, 1)
	err := ValidateTargetHostStatus([]byte(raw))
	if err == nil {
		t.Fatalf("expected beta target-host status without target-host evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "target-host") {
		t.Fatalf("error = %v, want target-host diagnostic", err)
	}
}
func validSurfaceReleaseSummaryJSON() []byte {
	return []byte(strings.Join([]string{
		"{",
		"  \"schema\": \"tetra.surface.release.v1\",",
		"  \"release_scope\": \"surface-v1-linux-web\",",
		"  \"status\": \"current\",",
		"  \"production_claim\": true,",
		"  \"experimental\": false,",
		"  \"producer\": \"scripts/release/surface/release-gate.sh\",",
		"  \"git_head\": \"0123456789abcdef0123456789abcdef01234567\",",
		"  \"version\": \"tetra_language\",",
		"  \"git_dirty\": false,",
		"  \"host_os\": \"linux\",",
		"  \"host_arch\": \"amd64\",",
		"  \"generated_at_utc\": \"2026-06-08T16:00:00Z\",",
		("  \"command_line\": \"bash scripts/release/surface/release-gate.sh --re" +
			"port-dir reports/surface-release-v1\","),
		"  \"supported_targets\": [\"headless\",\"linux-x64\",\"wasm32-web\"],",
		"  \"runtime_targets\": [\"linux-x64\",\"wasm32-web\"],",
		"  \"test_targets\": [\"headless\"],",
		("  \"unsupported_targets\": [\"macos-x64\",\"windows-x64\",\"wasm32-wasi\"]" +
			","),
		"  \"host_abi\": \"tetra.surface.host.v1\",",
		"  \"toolkit\": \"production-widgets-v1\",",
		"  \"text_input\": \"production-text-input-v1\",",
		"  \"clipboard\": \"clipboard-text-v1\",",
		"  \"ime\": \"composition-baseline-v1\",",
		"  \"accessibility\": \"platform-bridge-v1\",",
		"  \"app_model\": \"explicit-command-reducer-v1\",",
		"  \"linux_app_shell\": \"linux-app-shell-subset-v1\",",
		"  \"app_shell_features\": \"electron-feature-ledger-v1\",",
		"  \"security_permissions\": \"surface-security-permission-v1\",",
		"  \"performance_budget\": \"surface-performance-budget-v1\",",
		"  \"developer_fast_loop\": \"surface-dev-workflow-v1\",",
		"  \"inspector\": \"surface-inspector-v1\",",
		"  \"project_templates\": \"surface-template-smoke-v1\",",
		"  \"reference_apps\": \"surface-reference-app-suite-v1\",",
		"  \"surface_package\": \"surface-package-v1\",",
		"  \"crash_reporting\": \"surface-crash-report-v1\",",
		"  \"i18n_localization\": \"surface-i18n-v1\",",
		"  \"widget_migration\": \"surface-widget-migration-v1\",",
		"  \"browser_surface\": \"browser-canvas-release-v1\",",
		"  \"linux_surface\": \"linux-x64-release-window-v1\",",
		"  \"block_system\": \"block-system\",",
		"  \"block_system_gate\": \"tetra.surface.block-system.gate.v1\",",
		"  \"morph\": \"morph-capsule\",",
		"  \"morph_gate\": \"tetra.surface.morph.gate.v1\",",
		"  \"artifact_hashes_validated\": true,",
		"  \"legacy_sidecars\": false,",
		"  \"dom_ui\": false,",
		"  \"user_js\": false,",
		"  \"platform_widgets\": false",
		"}",
	}, "\n"))
}
func validSurfaceWindowsTargetHostStatusJSON() []byte {
	return []byte(strings.Join([]string{
		"{",
		"  \"schema\": \"tetra.surface.target-host-status.v1\",",
		"  \"target\": \"windows-x64\",",
		"  \"status\": \"unsupported\",",
		"  \"tier\": \"UNSUPPORTED\",",
		"  \"release_scope\": \"surface-v1-linux-web\",",
		"  \"source\": \"scripts/release/surface/release-gate.sh\",",
		"  \"host_os\": \"linux\",",
		"  \"host_arch\": \"amd64\",",
		("  \"reason\": \"no Windows target-host Surface v1 runner evidence exist" +
			"s in this release\","),
		"  \"production_claim\": false,",
		"  \"experimental\": false,",
		"  \"target_host_evidence\": false,",
		"  \"build_only_evidence\": false,",
		"  \"build_only_promotion\": false,",
		"  \"linux_substitute\": false,",
		"  \"ci_artifact_required\": true,",
		("  \"required_evidence\": {\"real_window\":false,\"native_input\":false,\"c" +
			"lipboard\":false,\"dpi_scaling\":false,\"accessibility_snapshot\":false,\"" +
			"app_shell\":false},"),
		("  \"unsupported_claims\": [\"windows-real-window-surface\",\"windows-prod" +
			"uction-surface-nonclaim\",\"windows-target-host-runtime\",\"build-only-w" +
			"indows-surface-runtime\",\"linux-substitute-windows-surface-runtime\"],"),
		("  \"negative_guards\": {\"no_linux_substitute\":true,\"no_build_only_prom" +
			"otion\":true,\"no_production_claim\":true,\"no_docs_only_evidence\":true," +
			"\"no_copied_report\":true,\"ci_artifact_required\":true}"),
		"}",
	}, "\n"))
}
func validSurfaceMacOSTargetHostStatusJSON() []byte {
	return []byte(strings.Join([]string{
		"{",
		"  \"schema\": \"tetra.surface.target-host-status.v1\",",
		"  \"target\": \"macos-x64\",",
		"  \"status\": \"unsupported\",",
		"  \"tier\": \"UNSUPPORTED\",",
		"  \"release_scope\": \"surface-v1-linux-web\",",
		"  \"source\": \"scripts/release/surface/release-gate.sh\",",
		"  \"host_os\": \"linux\",",
		"  \"host_arch\": \"amd64\",",
		("  \"reason\": \"no macOS target-host Surface v1 runner evidence exists " +
			"in this release\","),
		"  \"production_claim\": false,",
		"  \"experimental\": false,",
		"  \"target_host_evidence\": false,",
		"  \"build_only_evidence\": false,",
		"  \"build_only_promotion\": false,",
		"  \"linux_substitute\": false,",
		"  \"ci_artifact_required\": true,",
		("  \"required_evidence\": {\"real_window\":false,\"native_input\":false,\"c" +
			"lipboard\":false,\"dpi_scaling\":false,\"accessibility_snapshot\":false,\"" +
			"app_shell\":false},"),
		("  \"unsupported_claims\": [\"macos-real-window-surface\",\"macos-producti" +
			"on-surface-nonclaim\",\"macos-target-host-runtime\",\"build-only-macos-s" +
			"urface-runtime\",\"linux-substitute-macos-surface-runtime\"],"),
		("  \"negative_guards\": {\"no_linux_substitute\":true,\"no_build_only_prom" +
			"otion\":true,\"no_production_claim\":true,\"no_docs_only_evidence\":true," +
			"\"no_copied_report\":true,\"ci_artifact_required\":true}"),
		"}",
	}, "\n"))
}

// ---- report_test.go ----

func TestValidateReportAcceptsHeadlessSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsLinuxX64SurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64SurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsLinuxX64RealWindowSurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64RealWindowSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsLinuxX64NativeSurfaceHostRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsNativeSurfaceHostMissingNativeEvidence(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		delete(report, "native_surface_host")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected native Surface host report without native_surface_host evidence to fail")
	}
	if !strings.Contains(err.Error(), "native_surface_host") {
		t.Fatalf("error = %v, want native_surface_host diagnostic", err)
	}
}
func TestValidateReportRejectsNativeSurfaceHostProbeSubstitution(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		processes := report["processes"].([]any)
		processes = append(processes, map[string]any{
			"name":               "surface linux-x64 real-window probe",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-real-window-probe --probe-frame",
			"ran":                true,
			"pass":               true,
			"exit_code":          42,
			"expected_exit_code": 42,
		})
		report["processes"] = processes
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected native Surface host report with probe substitution to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "probe") {
		t.Fatalf("error = %v, want probe substitution diagnostic", err)
	}
}
func TestValidateReportRejectsNativeSurfaceHostPreRenderedFrame(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		host := report["native_surface_host"].(map[string]any)
		host["pre_rendered_frame_source"] = true
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		first["artifact_path"] = "/tmp/surface-artifacts/surface-window-counter.svg"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected native Surface host report with pre-rendered frame evidence to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "pre-rendered") && !strings.Contains(lower, "pre_rendered") {
		t.Fatalf("error = %v, want pre-rendered frame diagnostic", err)
	}
}
func TestValidateReportRejectsNativeSurfaceHostViewerSubstitution(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		processes := report["processes"].([]any)
		processes = append(processes, map[string]any{
			"name":      "surface linux-x64 guest viewer",
			"kind":      "runtime",
			"path":      "display -title Tetra /tmp/surface-artifacts/surface-window-counter.png",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		})
		report["processes"] = processes
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected native Surface host report with viewer substitution to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "viewer") && !strings.Contains(lower, "imagemagick") {
		t.Fatalf("error = %v, want viewer substitution diagnostic", err)
	}
}
func TestValidateReportAcceptsWASM32WebSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsWASM32WebBrowserCanvasSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebBrowserCanvasSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestHeadlessReleaseRequiresBuiltBinary(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra " +
			"build --target linux-x64 " +
			"examples/surface/runtime/surface_counter.tetra -o " +
			"/tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":" +
			"true,\"exit_code\":0},\n"),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without build process to fail")
	}
	if !strings.Contains(err.Error(), "build process") {
		t.Fatalf("error = %v, want build process diagnostic", err)
	}
}
func TestHeadlessRunnerTraceMatchesReport(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		second := frames[1].(map[string]any)
		second["checksum"] = first["checksum"]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unchanged pre/post headless frame checksum evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post") {
		t.Fatalf("error = %v, want pre/post frame diagnostic", err)
	}
}
func TestHeadlessRejectsMetadataOnlyFrame(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		first["checksum"] = ""
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected metadata-only headless frame to fail")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error = %v, want checksum diagnostic", err)
	}
}
func TestHeadlessNoLegacySidecars(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()),
		`    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
`,
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-legacy sidecar case to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar") {
		t.Fatalf("error = %v, want no legacy sidecar diagnostic", err)
	}
}
func mutateHeadlessSurfaceReport(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	mutate(report)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal headless report: %v", err)
	}
	return raw
}
func validLinuxX64NativeSurfaceHostReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validLinuxX64RealWindowSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode linux-x64 real-window report: %v", err)
	}
	source := "examples/surface/runtime/surface_window_counter.tetra"
	report["host_evidence"] = map[string]any{
		"level":                        "linux-x64-native-surface-host-v1",
		"backend":                      "wayland-surface-host-v1",
		"framebuffer":                  true,
		"real_window":                  true,
		"native_input":                 true,
		"user_facing_platform_widgets": false,
	}
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build native surface host app",
			"kind": "build",
			"path": "tetra build --target linux-x64 " + source + (" -o /tmp/surface-artifacts/" +
				"surface-window-counter"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-window-counter --surface-host wayland",
			"ran":                true,
			"pass":               true,
			"exit_code":          0,
			"expected_exit_code": 0,
		},
		map[string]any{
			"name":      "surface linux-x64 native surface host wayland",
			"kind":      "runtime",
			"path":      "tetra-surface-host-wayland --socket /run/user/1000/tetra-surface-host.sock",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-window-counter",
			"sha256": "sha256:" + strings.Repeat("a", 64),
			"size":   49172,
		},
		map[string]any{
			"kind":   "surface-host",
			"path":   "/tmp/surface-artifacts/tetra-surface-host-wayland",
			"sha256": "sha256:" + strings.Repeat("b", 64),
			"size":   32768,
		},
	}
	report["artifact_scan"] = map[string]any{
		"root":            "/tmp/surface-artifacts",
		"files_checked":   2,
		"forbidden_paths": []any{},
		"pass":            true,
	}
	frames := report["frames"].([]any)
	for _, item := range frames {
		frame := item.(map[string]any)
		frame["producer"] = "app"
		frame["evidence_role"] = "runtime_smoke"
		frame["app_source"] = source
		frame["precomputed"] = false
	}
	report["native_surface_host"] = map[string]any{
		"schema":                    "tetra.surface.native-host.v1",
		"host":                      "wayland",
		"protocol":                  "tetra.surface.host-ipc.v1",
		"app_process_kind":          "compiled-linux-x64-tetra-app",
		"host_process_kind":         "tetra-surface-host-wayland",
		"app_pid":                   4242,
		"host_pid":                  4243,
		"surface_open_from_app":     true,
		"poll_event_from_host":      true,
		"present_from_app_rgba":     true,
		"app_loop_observed":         true,
		"real_window":               true,
		"real_close_event":          true,
		"real_pointer_event_count":  1,
		"real_key_event_count":      1,
		"presented_frame_count":     len(frames),
		"pre_rendered_frame_source": false,
		"delivery_path":             "compiled-tetra-app-to-wayland-surface",
	}
	cases := report["cases"].([]any)
	cases = append(
		cases,
		map[string]any{
			"name": "native Surface host Wayland live window",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "native Surface host app loop observed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "native Surface host close event",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "native Surface host pointer input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "native Surface host keyboard input",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "native Surface host frame presented by running app",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name":           "native Surface host rejects pre-rendered frame source",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "pre-rendered frame source rejected",
		},
		map[string]any{
			"name":           "native Surface host rejects viewer substitution",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "viewer substitution rejected",
		},
		map[string]any{
			"name":           "native Surface host rejects probe-frame substitution",
			"kind":           "negative",
			"ran":            true,
			"pass":           true,
			"expected_error": "probe-frame substitution rejected",
		},
	)
	report["cases"] = cases
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal native Surface host report: %v", err)
	}
	return raw
}
func TestValidateReportRejectsMissingHostEvidence(t *testing.T) {
	raw := []byte(
		strings.Replace(
			string(validHeadlessSurfaceReportJSON()),
			("  \"host_evidence\": {\"level\":\"deterministic-headless\"," +
				"\"backend\":\"software-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false},\n"),
			``,
			1,
		),
	)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected report without explicit host_evidence to fail")
	}
	if !strings.Contains(err.Error(), "host_evidence") {
		t.Fatalf("error = %v, want host_evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(
		strings.Replace(
			string(validHeadlessSurfaceReportJSON()),
			`"target": "headless"`,
			`"target": "linux-x64"`,
			1,
		),
	)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "linux-x64") ||
		!strings.Contains(err.Error(), "surface-linux-x64") {
		t.Fatalf("error = %v, want linux-x64 runtime evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64MemfdStarterClaimingRealWindow(t *testing.T) {
	raw := strings.Replace(
		string(validLinuxX64SurfaceReportJSON()),
		`"real_window":false,"native_input":false`,
		`"real_window":true,"native_input":true`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 memfd starter real-window claim to fail")
	}
	if !strings.Contains(err.Error(), "memfd starter") ||
		!strings.Contains(err.Error(), "real_window") {
		t.Fatalf("error = %v, want memfd starter real_window diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64RealWindowWithoutRealWindowProbeEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validLinuxX64SurfaceReportJSON()),
		("\"host_evidence\": {\"level\":\"linux-x64-memfd-starter\"," +
			"\"backend\":\"memfd-rgba\",\"framebuffer\":true,\"real_window\":" +
			"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
			"false}"),
		("\"host_evidence\": {\"level\":\"linux-x64-real-window\",\"backend\":" +
			"\"x11-rgba\",\"framebuffer\":true,\"real_window\":true," +
			"\"native_input\":true,\"user_facing_platform_widgets\":false}"),
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf(
			"expected linux-x64 real-window claim without real-window process/case evidence to fail",
		)
	}
	for _, want := range []string{"real-window", "native input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(
		strings.Replace(
			string(validHeadlessSurfaceReportJSON()),
			`"target": "headless"`,
			`"target": "wasm32-web"`,
			1,
		),
	)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected wasm32-web report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "wasm32-web") ||
		!strings.Contains(err.Error(), "surface-wasm32-web") {
		t.Fatalf("error = %v, want wasm32-web runtime evidence diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebReportMissingCompilerOwnedLoaderEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"name": "compiler-owned wasm Surface loader",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader evidence to fail")
	}
	if !strings.Contains(err.Error(), "compiler-owned wasm Surface loader") {
		t.Fatalf("error = %v, want compiler-owned loader evidence diagnostic", err)
	}
}

func TestValidateReportRejectsWASM32WebReportMissingActualPresentedFrameTraceEvidence(
	t *testing.T,
) {
	raw := strings.Replace(
		string(validWASM32WebSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"order": 4,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "4444444444444444444444444444444444444444444444444444444444444444",
			"presented": true
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "wasm32-web actual presented frame trace",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without actual presented frame trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "actual presented frame trace") {
		t.Fatalf("error = %v, want actual presented frame trace evidence diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebReportMissingImportValidatorProcess(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"name": "surface wasm32-web import validator",
			"kind": "runtime",
			"path": "`+("go run ./tools/cmd/validate-wasm-imports --target "+
			"wasm32-web /tmp/surface-artifacts/surface-counter.wasm")+`",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf(
			"expected wasm32-web report without validate-wasm-imports process evidence to fail",
		)
	}
	for _, want := range []string{"wasm32-web", "validate-wasm-imports"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebBrowserCanvasWithoutBrowserProcess(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebBrowserCanvasSurfaceReportJSON()),
		("/usr/bin/chromium --headless " +
			"<surface-browser-canvas-runner> " +
			"wasm=/tmp/surface-artifacts/surface-browser-counter.wasm"),
		`node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-browser-counter.wasm`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without Chromium process evidence to fail")
	}
	if !strings.Contains(err.Error(), "Chromium-compatible browser") {
		t.Fatalf("error = %v, want Chromium-compatible browser process diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebBrowserCanvasMissingInputEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebBrowserCanvasSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"order": 2,
			"kind": "key_down",
			"target_component": "CounterApp",
			"dispatch_path": ["CounterApp"],
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"key": 32,
			"width": 320,
			"height": 200,
			"timestamp_ms": 1,
			"buffer_slots": [6,0,0,0,32,320,200,1,0],
			"before_state": {"CounterApp.count": "1", "CounterApp.key_count": "0"},
			"after_state": {"CounterApp.count": "2", "CounterApp.key_count": "1"}
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "wasm32-web browser canvas keyboard input",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without keyboard input evidence to fail")
	}
	for _, want := range []string{"keyboard input", "key_down"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebReportMissingRunnerTraceArtifact(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"kind": "runner-trace",
			"path": "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size": 413
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without runner trace artifact to fail")
	}
	if !strings.Contains(err.Error(), "runner trace artifact") {
		t.Fatalf("error = %v, want runner trace artifact diagnostic", err)
	}
}
func TestValidateReportRejectsHeadlessReportMissingRunnerTraceEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"kind": "runner-trace",
			"path": "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size": 409
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "headless actual runner trace",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without runner trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "headless actual runner trace") {
		t.Fatalf("error = %v, want headless runner trace evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportMissingAppPresentedFrameEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validLinuxX64SurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"order": 3,
			"width": 2,
			"height": 2,
			"stride": 8,
			"checksum": "3333333333333333333333333333333333333333333333333333333333333333",
			"presented": true
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "linux-x64 app-presented RGBA checksum",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without app-presented frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "app-presented RGBA checksum") {
		t.Fatalf("error = %v, want app-presented frame evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64ReportMissingCounterComponentAppPresentedFrameEvidence(
	t *testing.T,
) {
	raw := strings.Replace(
		string(validLinuxX64SurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"order": 4,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "4444444444444444444444444444444444444444444444444444444444444444",
			"presented": true
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "surface linux-x64 counter app presented frame probe",
			"kind": "app",
			"path": "/tmp/surface-artifacts/surface-counter-present-probe",
			"ran": true,
			"pass": true,
			"exit_code": -1,
			"expected_exit_code": -1
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "linux-x64 counter component app-presented frame",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf(
			"expected linux-x64 report without counter component app-presented frame evidence to fail",
		)
	}
	if !strings.Contains(err.Error(), "counter component app-presented frame") {
		t.Fatalf("error = %v, want counter component app-presented frame evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportMissingEventSequenceProbeEvidence(t *testing.T) {
	processes := compactJSONArrayLine(t, `{
		"name": "surface linux-x64 event sequence probe build",
		"kind": "build",
		"path": "/tmp/tetra build event sequence probe",
		"ran": true,
		"pass": true,
		"exit_code": 0
	}`) + compactJSONArrayLine(t, `{
		"name": "surface linux-x64 event sequence probe",
		"kind": "app",
		"path": "/tmp/surface-event-sequence-probe",
		"ran": true,
		"pass": true,
		"exit_code": 42,
		"expected_exit_code": 42
	}`)
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), processes, ``, 1)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "linux-x64 host event sequence",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without event sequence probe evidence to fail")
	}
	if !strings.Contains(err.Error(), "event sequence") {
		t.Fatalf("error = %v, want event sequence probe evidence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingPrePostEventFrameSequence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"order": 2,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "2222222222222222222222222222222222222222222222222222222222222222",
			"presented": true
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "pre/post event frame sequence",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without pre/post frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post event frame sequence") {
		t.Fatalf("error = %v, want pre/post frame sequence diagnostic", err)
	}
}
func TestValidateReportRejectsLegacyMetadataEvidence(t *testing.T) {
	raw := []byte(
		strings.Join([]string{
			"{",
			"  \"schema\": \"tetra.ui.v1\",",
			"  \"status\": \"pass\",",
			"  \"source\": \"examples/ui/ui_web_smoke.tetra\"",
			"}",
		}, "\n"),
	)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected legacy metadata report to fail")
	}
	if !strings.Contains(err.Error(), SchemaV1) {
		t.Fatalf("error = %v, want Surface runtime schema rejection", err)
	}
}
func TestValidateReportRejectsDocsOnlyMarkers(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"source": "examples/surface/runtime/surface_counter.tetra"`,
		`"source": "docs-only surface note"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected docs-only marker to fail")
	}
	if !strings.Contains(err.Error(), "docs-only") {
		t.Fatalf("error = %v, want docs-only rejection", err)
	}
}
func TestValidateReportRejectsForbiddenEvidenceMarkers(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   string
	}{
		{source: "web-only", want: "web-only"},
		{source: "metadata-only", want: "metadata-only"},
		{source: "node-only", want: "node-only"},
		{source: "dom-only", want: "dom-only"},
		{source: "build-only", want: "build-only"},
		{source: "surface fake evidence", want: "fake"},
		{source: "surface stale evidence", want: "stale"},
		{source: "surface mock evidence", want: "mock"},
		{source: "placeholder", want: "placeholder"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			raw := strings.Replace(
				string(validHeadlessSurfaceReportJSON()),
				`"source": "examples/surface/runtime/surface_counter.tetra"`,
				`"source": "`+tc.source+`"`,
				1,
			)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected marker %q to fail", tc.source)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsLegacyUISidecarMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "generated .ui.html sidecar", want: ".ui.html"},
		{name: "generated .ui.web.mjs sidecar", want: ".ui.web.mjs"},
		{name: "generated .ui.json sidecar", want: ".ui.json"},
		{name: "DOM UI surface", want: "dom ui"},
		{name: "user JavaScript bridge", want: "user javascript"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(
				string(validHeadlessSurfaceReportJSON()),
				strings.Join([]string{
					"\"pure Tetra component app\"",
				}, "\n"),
				`"`+tc.name+`"`,
				1,
			)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected legacy UI marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsUserFacingPlatformWidgetMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "React component surface", want: "react"},
		{name: "GTK widget surface", want: "gtk widget"},
		{name: "Qt widget surface", want: "qt widget"},
		{name: "WinUI widget surface", want: "winui"},
		{name: "Cocoa widget surface", want: "cocoa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(
				string(validHeadlessSurfaceReportJSON()),
				strings.Join([]string{
					"\"pure Tetra component app\"",
				}, "\n"),
				`"`+tc.name+`"`,
				1,
			)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsMissingNoLegacyUISidecarEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"name": "no legacy UI sidecar artifacts",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-sidecar evidence to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar artifacts") {
		t.Fatalf("error = %v, want no legacy UI sidecar evidence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingArtifactScanEvidence(t *testing.T) {
	artifactScan := ",\n  \"artifact_scan\": " + compactJSONSnippet(t, `{
		"root": "/tmp/surface-artifacts",
		"files_checked": 2,
		"forbidden_paths": [],
		"pass": true
	}`)
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), artifactScan, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without artifact_scan evidence to fail")
	}
	for _, want := range []string{"artifact_scan"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsArtifactOutsideArtifactScanRoot(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"artifact_scan": {"root":"/tmp/surface-artifacts"`,
		`"artifact_scan": {"root":"/tmp/other-surface-artifacts"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report whose artifacts are outside artifact_scan.root to fail")
	}
	for _, want := range []string{"artifact_scan.root", "outside"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsArtifactScanCheckingFewerFilesThanReportedArtifacts(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"files_checked":2`,
		`"files_checked":1`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf(
			"expected report whose artifact_scan checked fewer files than reported artifacts to fail",
		)
	}
	for _, want := range []string{"artifact_scan.files_checked", "reported artifacts"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostProvidedPointerEventEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"name": "host-provided pointer event dispatch",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing host-provided pointer event evidence to fail")
	}
	if !strings.Contains(err.Error(), "host-provided pointer event dispatch") {
		t.Fatalf("error = %v, want host-provided pointer event evidence diagnostic", err)
	}
}
func TestValidateReportRejectsComponentMissingMeasureLayoutAbilities(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"abilities":["measure","layout","draw","event","focus","text","accessibility"]`,
		`"abilities":["draw","event","focus","text","accessibility"]`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected component without measure/layout abilities to fail")
	}
	for _, want := range []string{"measure ability", "layout ability"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingFocusAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"abilities":["measure","layout","draw","event","focus","text","accessibility"]`,
		`"abilities":["measure","layout","draw","event","text","accessibility"]`,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without focus ability and case evidence to fail")
	}
	for _, want := range []string{"focus ability", "component focus dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingAccessibilityAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"abilities":["measure","layout","draw","event","focus","text","accessibility"]`,
		`"abilities":["measure","layout","draw","event","focus","text"]`,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without accessibility ability and case evidence to fail")
	}
	for _, want := range []string{"accessibility ability", "component accessibility metadata"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingTextAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"abilities":["measure","layout","draw","event","focus","text","accessibility"]`,
		`"abilities":["measure","layout","draw","event","focus","accessibility"]`,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"order": 3,
			"kind": "text_input",
			"target_component": "CounterButton",
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"text_len": 2,
			"text_bytes_hex": "4f4b",
			"before_state": {
				"CounterApp.text_count": "0",
				"CounterButton.text_len_seen": "0"
			},
			"after_state": {
				"CounterApp.text_count": "1",
				"CounterButton.text_len_seen": "2"
			}
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"order": 2,
			"component": "CounterApp",
			"field": "text_count",
			"before": "0",
			"after": "1",
			"cause": "text_input"
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "component text input scalar dispatch",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without text ability and scalar text-input evidence to fail")
	}
	for _, want := range []string{"text ability", "component text input scalar dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostTextPayloadBufferEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"text_len":2,"text_bytes_hex":"4f4b",`,
		``,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host text payload buffer evidence to fail")
	}
	for _, want := range []string{"text payload", "host text payload buffer"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostEventBufferEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],`,
		``,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer evidence to fail")
	}
	for _, want := range []string{"event buffer", "host event buffer poll_event"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostEventBufferSequenceEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2]`,
		`"timestamp_ms":0,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,0,2]`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer pointer/text sequence to fail")
	}
	if !strings.Contains(err.Error(), "event buffer pointer/text sequence") {
		t.Fatalf("error = %v, want host event buffer pointer/text sequence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingComponentHierarchyEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"id": "CounterButton",
			"type": "examples.surface.runtime.surface_counter.CounterButton",
			"parent": "CounterApp",
			"bounds": {"x":32,"y":80,"w":160,"h":48},
			"abilities": ["measure","layout","draw","event","focus","text","accessibility"],
			"state": {
				"pressed": "false",
				"focused": "true",
				"text_len_seen": "2",
				"accessibility_role": "button"
			}
		}`),
		``,
		1,
	)
	raw = strings.Replace(
		raw,
		`"target_component":"CounterButton"`,
		`"target_component":"CounterApp"`,
		1,
	)
	raw = strings.Replace(
		raw,
		compactJSONArrayLine(t, `{
			"name": "component hierarchy dispatch",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component hierarchy evidence to fail")
	}
	for _, want := range []string{"component hierarchy", "component hierarchy dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingComponentLayoutBoundsEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"bounds":{"x":32,"y":80,"w":160,"h":48},`,
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component bounds evidence to fail")
	}
	if !strings.Contains(err.Error(), "layout bounds") {
		t.Fatalf("error = %v, want layout bounds diagnostic", err)
	}
}
func TestValidateReportRejectsMissingEventDispatchPathEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"dispatch_path":["CounterApp","CounterButton"],`,
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child dispatch_path evidence to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") {
		t.Fatalf("error = %v, want dispatch_path diagnostic", err)
	}
}
func TestValidateReportRejectsDispatchPathSkippingParent(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"dispatch_path":["CounterApp","CounterButton"]`,
		`"dispatch_path":["CounterButton"]`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report with dispatch_path skipping parent to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("error = %v, want dispatch_path parent diagnostic", err)
	}
}
func TestValidateReportRejectsPointerDispatchOutsideTargetBounds(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		("\"x\":48,\"y\":96,\"key\":0,\"width\":320,\"height\":200," +
			"\"timestamp_ms\":0,\"buffer_slots\":[5,48,96,1,0,320,200,0,0]"),
		("\"x\":4,\"y\":4,\"key\":0,\"width\":320,\"height\":200,\"timestamp_ms\":" +
			"0,\"buffer_slots\":[5,4,4,1,0,320,200,0,0]"),
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected pointer dispatch outside target bounds to fail")
	}
	if !strings.Contains(err.Error(), "target bounds") {
		t.Fatalf("error = %v, want target bounds diagnostic", err)
	}
}
func TestValidateReportRejectsSourcePathAsExecutableAppProcess(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"kind":"app","path":"/tmp/surface-artifacts/surface-counter"`,
		`"kind":"app","path":"examples/surface/runtime/surface_counter.tetra"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected app process source path to fail")
	}
	if !strings.Contains(err.Error(), "executable Surface app process") {
		t.Fatalf("error = %v, want executable app path diagnostic", err)
	}
}
func TestValidateReportRejectsBuildProcessMissingReportedSource(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		("\"path\":\"tetra build --target linux-x64 " +
			"examples/surface/runtime/surface_counter.tetra -o " +
			"/tmp/surface-artifacts/surface-counter\""),
		("\"path\":\"tetra build --target linux-x64 " +
			"examples/other_surface.tetra -o " +
			"/tmp/surface-artifacts/surface-counter\""),
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected build process without reported source to fail")
	}
	for _, want := range []string{"build process", "source"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingSurfaceComponentAppProcess(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"name":"surface component app"`,
		`"name":"surface auxiliary app"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing Surface component app process to fail")
	}
	if !strings.Contains(err.Error(), "Surface component app process") {
		t.Fatalf("error = %v, want Surface component app process diagnostic", err)
	}
}
func TestValidateReportRejectsMissingComponentAppArtifactEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		("    {\"kind\":\"component-app\",\"path\":" +
			"\"/tmp/surface-artifacts/surface-counter\",\"sha256\":\"sha256:" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaa\",\"size\":49172},\n"),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without Surface component app artifact hash evidence to fail")
	}
	for _, want := range []string{"artifact", "Surface component app"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestSurfaceProjectTemplateSourceAcceptsFinalProductReportPath(t *testing.T) {
	source := "reports/surface-product-v1/templates/command-palette/src/main.tetra"
	if !isSurfaceProjectTemplateSource(source) {
		t.Fatalf("final product report template source was rejected: %s", source)
	}
	if !isSurfaceBlockAccessibilitySource(source) {
		t.Fatalf(
			"final product report template source was rejected for Block accessibility evidence: %s",
			source,
		)
	}
}
func TestValidateReportRejectsWASM32WebMissingCompilerOwnedLoaderArtifact(t *testing.T) {
	raw := strings.Replace(
		string(validWASM32WebSurfaceReportJSON()),
		compactJSONArrayLine(t, `{
			"kind": "compiler-owned-loader",
			"path": "/tmp/surface-artifacts/surface-counter.mjs",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size": 4931
		}`),
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader artifact to fail")
	}
	for _, want := range []string{"compiler-owned loader artifact", "wasm32-web"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsGeneratedHTMLArtifactEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter"`,
		`{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.html"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected generated HTML artifact evidence to fail")
	}
	if !strings.Contains(err.Error(), "generated HTML UI") {
		t.Fatalf("error = %v, want generated HTML UI diagnostic", err)
	}
}
func TestValidateReportRejectsPlatformWidgetArtifactEvidence(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		want   string
	}{
		{suffix: ".jsx", want: "react"},
		{suffix: ".tsx", want: "react"},
		{suffix: ".qml", want: "qt"},
		{suffix: ".xaml", want: "winui"},
		{suffix: ".xib", want: "cocoa"},
		{suffix: ".storyboard", want: "cocoa"},
		{suffix: ".glade", want: "gtk"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			raw := strings.ReplaceAll(
				string(validHeadlessSurfaceReportJSON()),
				`/tmp/surface-artifacts/surface-counter`,
				`/tmp/surface-artifacts/surface-counter`+tc.suffix,
			)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget artifact suffix %q to fail", tc.suffix)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want platform artifact rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsSourceComponentModuleMismatch(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessSurfaceReportJSON()),
		`"source": "examples/surface/runtime/surface_counter.tetra"`,
		`"source": "examples/other_surface.tetra"`,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected source/component module mismatch to fail")
	}
	for _, want := range []string{"source module", "component type"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestSurfaceProjectTemplateSourceAcceptsExitZeroComponentAppProcess(t *testing.T) {
	for _, source := range []string{
		("reports/surface-electron-react-beauty-production/P21/templat" +
			"e-smoke/templates/command-palette/src/main.tetra"),
		("reports/surface-electron-react-beauty-production/P21/release" +
			"-gate/templates/command-palette/src/main.tetra"),
		"reports/surface/mrb10-template-smoke/templates/studio-shell/src/main.tetra",
		"reports/surface-product-v1-final-clean-20260613-0926/templates/command-palette/src/main.tetra",
		("reports/contract-refactor-pr-hardening/surface-release-final" +
			"/templates/studio-shell/src/main.tetra"),
	} {
		t.Run(source, func(t *testing.T) {
			exit := 0
			expected := 0
			process := ProcessReport{
				Name: "surface component app",
				Kind: "app",
				Path: ("reports/surface-electron-react-beauty-production/P21/templat" +
					"e-smoke/template-runtime/command-palette-linux-x64"),
				Ran:              true,
				Pass:             true,
				ExitCode:         &exit,
				ExpectedExitCode: &expected,
			}
			if !isSurfaceComponentAppProcess(source, process) {
				t.Fatalf(
					"generated Surface template source should accept exit-zero component app process",
				)
			}
			if !isSurfaceBlockAccessibilitySource(source) {
				t.Fatalf(
					"generated Surface template source should accept Block accessibility evidence",
				)
			}
			if !isSurfaceMorphReportSource(source) {
				t.Fatalf("generated Surface template source should accept Morph evidence")
			}
		})
	}
}

func TestSurfaceProjectTemplateSourceRejectsNonReportMainTetra(t *testing.T) {
	for _, source := range []string{
		"templates/studio-shell/src/main.tetra",
		"examples/studio-shell/src/main.tetra",
		"src/main.tetra",
	} {
		t.Run(source, func(t *testing.T) {
			if isSurfaceProjectTemplateSource(source) {
				t.Fatalf("non-report main.tetra source was accepted: %s", source)
			}
		})
	}
}

func TestSurfaceReferenceAppSourceAcceptsExitZeroMorphEvidence(t *testing.T) {
	exit := 0
	expected := 0
	process := ProcessReport{
		Name: "surface component app",
		Kind: "app",
		Path: ("reports/surface/reference-apps/reference-runtime/surface-hea" +
			"dless-morph-artifacts/surface-morph-command-palette"),
		Ran:              true,
		Pass:             true,
		ExitCode:         &exit,
		ExpectedExitCode: &expected,
	}
	for _, source := range []string{
		"examples/surface/reference_core/surface_reference_command_palette.tetra",
		"/repo/examples/surface/reference_core/surface_reference_settings.tetra",
	} {
		t.Run(source, func(t *testing.T) {
			if !isSurfaceReferenceAppSource(source) {
				t.Fatalf("reference Surface app source was rejected: %s", source)
			}
			if !isSurfaceComponentAppProcess(source, process) {
				t.Fatalf(
					"reference Surface app source should accept exit-zero component app process",
				)
			}
			if !isSurfaceBlockAccessibilitySource(source) {
				t.Fatalf("reference Surface app source should accept Block accessibility evidence")
			}
			if !isSurfaceMorphReportSource(source) {
				t.Fatalf("reference Surface app source should accept Morph evidence")
			}
		})
	}
}
func TestSurfaceFlagshipControlCenterSourceAcceptsAppStateExitFive(t *testing.T) {
	exit := 5
	expected := 5
	process := ProcessReport{
		Name: "surface component app",
		Kind: "app",
		Path: ("reports/surface-product-slice/product-gate/flagship/surface-" +
			"headless-block-system-artifacts/surface-block-system"),
		Ran:              true,
		Pass:             true,
		ExitCode:         &exit,
		ExpectedExitCode: &expected,
	}
	for _, source := range []string{
		"examples/surface/migration/surface_migration_tetra_control_center.tetra",
		"/repo/examples/surface/migration/surface_migration_tetra_control_center.tetra",
	} {
		if !isSurfaceComponentAppProcess(source, process) {
			t.Fatalf("flagship Surface source should accept app-state exit 5 for %s", source)
		}
	}
}
func TestValidateReportRejectsMissingFrameChecksumAndStateTransition(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.runtime.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"headless\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"surface-headless\",",
		"  \"surface_schema\": \"tetra.surface.v1\",",
		"  \"host_abi\": \"tetra.surface.host-abi.v1\",",
		("  \"host_evidence\": {\"level\":\"deterministic-headless\",\"backend\":\"s" +
			"oftware-rgba\",\"framebuffer\":true,\"real_window\":false,\"native_input\"" +
			":false,\"user_facing_platform_widgets\":false},"),
		"  \"source\": \"examples/surface/runtime/surface_counter.tetra\",",
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra build --" +
			"target linux-x64 examples/surface/runtime/surface_counter.tetra -o /" +
			"tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":true,\"exit_" +
			"code\":0},"),
		("    {\"name\":\"surface component app\",\"kind\":\"app\",\"path\":\"/tmp/s" +
			"urface-artifacts/surface-counter\",\"ran\":true,\"pass\":true,\"exit_code" +
			"\":1,\"expected_exit_code\":1},"),
		("    {\"name\":\"surface headless runtime\",\"kind\":\"runtime\",\"path\":" +
			"\"tools/cmd/surface-runtime-smoke\",\"ran\":true,\"pass\":true,\"exit_code" +
			"\":0}"),
		"  ],",
		"  \"artifacts\": [",
		("    {\"kind\":\"component-app\",\"path\":\"/tmp/surface-artifacts/surface-" +
			"counter\",\"sha256\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaa\",\"size\":49172},"),
		("    {\"kind\":\"runner-trace\",\"path\":\"/tmp/surface-artifacts/surface-r" +
			"unner-trace.json\",\"sha256\":\"sha256:ccccccccccccccccccccccccccccccccc" +
			"ccccccccccccccccccccccccccccccc\",\"size\":409}"),
		"  ],",
		("  \"artifact_scan\": {\"root\":\"/tmp/surface-artifacts\",\"files_checked" +
			"\":2,\"forbidden_paths\":[],\"pass\":true},"),
		"  \"components\": [",
		("    {\"id\":\"CounterApp\",\"type\":\"examples.surface.runtime.surface_cou" +
			"nter.CounterApp\",\"bounds\":{\"x\":0,\"y\":0,\"w\":320,\"h\":200},\"abili" +
			"ties\":[\"measure\",\"layout\",\"draw\",\"event\",\"focus\",\"text\",\"acc" +
			"essibility\"],\"state\":{\"count\":\"1\",\"text_count\":\"1\",\"accessibil" +
			"ity_role\":\"button\"}},"),
		("    {\"id\":\"CounterButton\",\"type\":\"examples.surface.runtime.surface_" +
			"counter.CounterButton\",\"parent\":\"CounterApp\",\"bounds\":{\"x\":32,\"y" +
			"\":80,\"w\":160,\"h\":48},\"abilities\":[\"measure\",\"layout\",\"draw\"," +
			"\"event\",\"focus\",\"text\",\"accessibility\"],\"state\":{\"pressed\":\"f" +
			"alse\",\"focused\":\"true\",\"text_len_seen\":\"2\",\"accessibility_role\"" +
			":\"button\"}}"),
		"  ],",
		"  \"events\": [",
		("    {\"order\":1,\"kind\":\"mouse_up\",\"target_component\":\"CounterApp\"" +
			",\"dispatch_path\":[\"CounterApp\"],\"handled\":true,\"pass\":true,\"x\":4" +
			"8,\"y\":96,\"before_state\":{\"CounterApp.count\":\"0\"},\"after_state\":{" +
			"\"CounterApp.count\":\"1\"}}"),
		"  ],",
		"  \"frames\": [",
		("    {\"order\":1,\"width\":320,\"height\":200,\"stride\":1280,\"checksum\"" +
			":\"\",\"presented\":true}"),
		"  ],",
		"  \"cases\": [",
		("    {\"name\":\"pure Tetra component app\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"headless event dispatch\",\"kind\":\"positive\",\"ran\":tr" +
			"ue,\"pass\":true},"),
		("    {\"name\":\"headless framebuffer checksum\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"headless actual runner trace\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"host-provided pointer event dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"component hierarchy dispatch\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"component focus dispatch\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component accessibility metadata\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"no legacy UI sidecar artifacts\",\"kind\":\"positive\",\"r" +
			"an\":true,\"pass\":true},"),
		("    {\"name\":\"state transition\",\"kind\":\"positive\",\"ran\":true,\"pa" +
			"ss\":true},"),
		("    {\"name\":\"reject legacy UI evidence\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"legacy UI evidence rejected\"}"),
		"  ]",
		"}",
		"",
	}, "\n"))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing checksum and transition to fail")
	}
	for _, want := range []string{"checksum", "state transition"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}
func validHeadlessSurfaceReportJSON() []byte {
	return []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.runtime.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"headless\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"surface-headless\",",
		"  \"surface_schema\": \"tetra.surface.v1\",",
		"  \"host_abi\": \"tetra.surface.host-abi.v1\",",
		("  \"host_evidence\": {\"level\":\"deterministic-headless\",\"backend\":\"s" +
			"oftware-rgba\",\"framebuffer\":true,\"real_window\":false,\"native_input\"" +
			":false,\"user_facing_platform_widgets\":false},"),
		"  \"source\": \"examples/surface/runtime/surface_counter.tetra\",",
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra build --" +
			"target linux-x64 examples/surface/runtime/surface_counter.tetra -o /" +
			"tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":true,\"exit_" +
			"code\":0},"),
		("    {\"name\":\"surface component app\",\"kind\":\"app\",\"path\":\"/tmp/s" +
			"urface-artifacts/surface-counter\",\"ran\":true,\"pass\":true,\"exit_code" +
			"\":1,\"expected_exit_code\":1},"),
		("    {\"name\":\"surface headless runtime\",\"kind\":\"runtime\",\"path\":" +
			"\"tools/cmd/surface-runtime-smoke\",\"ran\":true,\"pass\":true,\"exit_code" +
			"\":0}"),
		"  ],",
		"  \"artifacts\": [",
		("    {\"kind\":\"component-app\",\"path\":\"/tmp/surface-artifacts/surface-" +
			"counter\",\"sha256\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaa\",\"size\":49172},"),
		("    {\"kind\":\"runner-trace\",\"path\":\"/tmp/surface-artifacts/surface-r" +
			"unner-trace.json\",\"sha256\":\"sha256:ccccccccccccccccccccccccccccccccc" +
			"ccccccccccccccccccccccccccccccc\",\"size\":409}"),
		"  ],",
		("  \"artifact_scan\": {\"root\":\"/tmp/surface-artifacts\",\"files_checked" +
			"\":2,\"forbidden_paths\":[],\"pass\":true},"),
		"  \"components\": [",
		("    {\"id\":\"CounterApp\",\"type\":\"examples.surface.runtime.surface_cou" +
			"nter.CounterApp\",\"bounds\":{\"x\":0,\"y\":0,\"w\":320,\"h\":200},\"abili" +
			"ties\":[\"measure\",\"layout\",\"draw\",\"event\",\"focus\",\"text\",\"acc" +
			"essibility\"],\"state\":{\"count\":\"1\",\"text_count\":\"1\",\"accessibil" +
			"ity_role\":\"button\"}},"),
		("    {\"id\":\"CounterButton\",\"type\":\"examples.surface.runtime.surface_" +
			"counter.CounterButton\",\"parent\":\"CounterApp\",\"bounds\":{\"x\":32,\"y" +
			"\":80,\"w\":160,\"h\":48},\"abilities\":[\"measure\",\"layout\",\"draw\"," +
			"\"event\",\"focus\",\"text\",\"accessibility\"],\"state\":{\"pressed\":\"f" +
			"alse\",\"focused\":\"true\",\"text_len_seen\":\"2\",\"accessibility_role\"" +
			":\"button\"}}"),
		"  ],",
		"  \"events\": [",
		("    {\"order\":1,\"kind\":\"none\",\"target_component\":\"CounterApp\",\"d" +
			"ispatch_path\":[\"CounterApp\"],\"handled\":false,\"pass\":true,\"x\":0,\"" +
			"y\":0,\"before_state\":{\"CounterApp.count\":\"0\"},\"after_state\":{\"Cou" +
			"nterApp.count\":\"0\"}},"),
		("    {\"order\":2,\"kind\":\"mouse_up\",\"target_component\":\"CounterButto" +
			"n\",\"dispatch_path\":[\"CounterApp\",\"CounterButton\"],\"handled\":true," +
			"\"pass\":true,\"x\":48,\"y\":96,\"key\":0,\"width\":320,\"height\":200,\"t" +
			"imestamp_ms\":0,\"buffer_slots\":[5,48,96,1,0,320,200,0,0],\"before_stat" +
			"e\":{\"CounterApp.count\":\"0\",\"CounterButton.pressed\":\"false\"},\"aft" +
			"er_state\":{\"CounterApp.count\":\"1\",\"CounterButton.pressed\":\"false\"" +
			"}},"),
		("    {\"order\":3,\"kind\":\"text_input\",\"target_component\":\"CounterBut" +
			"ton\",\"dispatch_path\":[\"CounterApp\",\"CounterButton\"],\"handled\":tru" +
			"e,\"pass\":true,\"x\":0,\"y\":0,\"key\":0,\"width\":320,\"height\":200,\"t" +
			"imestamp_ms\":1,\"text_len\":2,\"text_bytes_hex\":\"4f4b\",\"buffer_slots" +
			"\":[8,0,0,0,0,320,200,1,2],\"before_state\":{\"CounterApp.text_count\":\"0" +
			"\",\"CounterButton.text_len_seen\":\"0\"},\"after_state\":{\"CounterApp.te" +
			"xt_count\":\"1\",\"CounterButton.text_len_seen\":\"2\"}}"),
		"  ],",
		"  \"frames\": [",
		("    {\"order\":1,\"width\":320,\"height\":200,\"stride\":1280,\"checksum\"" +
			":\"1111111111111111111111111111111111111111111111111111111111111111\"," +
			"\"presented\":true},"),
		("    {\"order\":2,\"width\":320,\"height\":200,\"stride\":1280,\"checksum\"" +
			":\"2222222222222222222222222222222222222222222222222222222222222222\"," +
			"\"presented\":true}"),
		"  ],",
		"  \"state_transitions\": [",
		("    {\"order\":1,\"component\":\"CounterApp\",\"field\":\"count\",\"before" +
			"\":\"0\",\"after\":\"1\",\"cause\":\"mouse_up\"},"),
		("    {\"order\":2,\"component\":\"CounterApp\",\"field\":\"text_count\",\"b" +
			"efore\":\"0\",\"after\":\"1\",\"cause\":\"text_input\"}"),
		"  ],",
		"  \"cases\": [",
		("    {\"name\":\"pure Tetra component app\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"headless event dispatch\",\"kind\":\"positive\",\"ran\":tr" +
			"ue,\"pass\":true},"),
		("    {\"name\":\"headless framebuffer checksum\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"headless actual runner trace\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"host-provided pointer event dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host event buffer poll_event\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"pre/post event frame sequence\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"component hierarchy dispatch\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"component text input scalar dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host text payload buffer\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component focus dispatch\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component accessibility metadata\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"no legacy UI sidecar artifacts\",\"kind\":\"positive\",\"r" +
			"an\":true,\"pass\":true},"),
		("    {\"name\":\"state transition\",\"kind\":\"positive\",\"ran\":true,\"pa" +
			"ss\":true},"),
		("    {\"name\":\"reject legacy UI evidence\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"legacy UI evidence rejected\"}"),
		"  ]",
		"}",
		"",
	}, "\n"))
}
func validLinuxX64SurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "linux-x64"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-linux-x64"`},
		{
			old: ("\"host_evidence\": {\"level\":\"deterministic-headless\"," +
				"\"backend\":\"software-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false}"),
			new: ("\"host_evidence\": {\"level\":\"linux-x64-memfd-starter\"," +
				"\"backend\":\"memfd-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false}"),
		},
		{
			old: ("{\"name\":\"surface component app\",\"kind\":\"app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":" +
				"true,\"exit_code\":1,\"expected_exit_code\":1}"),
			new: ("{\"name\":\"surface component app\",\"kind\":\"app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":" +
				"true,\"exit_code\":1,\"expected_exit_code\":1},\n    {\"name\":" +
				"\"surface linux-x64 host probe build\",\"kind\":\"build\",\"path\":" +
				"\"/tmp/tetra build probe\",\"ran\":true,\"pass\":true,\"exit_code\":" +
				"0},\n    {\"name\":\"surface linux-x64 host probe\",\"kind\":\"app\"," +
				"\"path\":\"/tmp/surface-host-probe\",\"ran\":true,\"pass\":true," +
				"\"exit_code\":42,\"expected_exit_code\":42},\n    {\"name\":" +
				"\"surface linux-x64 event sequence probe build\",\"kind\":" +
				"\"build\",\"path\":\"/tmp/tetra build event sequence probe\"," +
				"\"ran\":true,\"pass\":true,\"exit_code\":0},\n    {\"name\":\"surface " +
				"linux-x64 event sequence probe\",\"kind\":\"app\",\"path\":" +
				"\"/tmp/surface-event-sequence-probe\",\"ran\":true,\"pass\":true," +
				"\"exit_code\":42,\"expected_exit_code\":42},\n    {\"name\":" +
				"\"surface linux-x64 counter app presented frame probe\"," +
				"\"kind\":\"app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter-present-probe\"," +
				"\"ran\":true,\"pass\":true,\"exit_code\":-1,\"expected_exit_code\":" +
				"-1}"),
		},
		{old: strings.Join([]string{
			"\"surface headless runtime\"",
		}, "\n"), new: strings.Join([]string{
			"\"surface linux-x64 runtime\"",
		}, "\n")},
		{old: strings.Join([]string{
			"\"headless event dispatch\"",
		}, "\n"), new: strings.Join([]string{
			"\"linux-x64 Surface Host ABI open/present/close\"",
		}, "\n")},
		{old: strings.Join([]string{
			"\"headless framebuffer checksum\"",
		}, "\n"), new: strings.Join([]string{
			"\"linux-x64 framebuffer present evidence\"",
		}, "\n")},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(
		raw,
		compactJSONArrayLineRaw(`{
			"kind": "runner-trace",
			"path": "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size": 409
		}`),
		``,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(
		raw,
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"22222222222222222222222222222222222222222222222222222222222" +
			"22222\",\"presented\":true}"),
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"22222222222222222222222222222222222222222222222222222222222" +
			"22222\",\"presented\":true},\n    {\"order\":3,\"width\":2,\"height\":" +
			"2,\"stride\":8,\"checksum\":" +
			"\"33333333333333333333333333333333333333333333333333333333333" +
			"33333\",\"presented\":true},\n    {\"order\":4,\"width\":320," +
			"\"height\":200,\"stride\":1280,\"checksum\":" +
			"\"44444444444444444444444444444444444444444444444444444444444" +
			"44444\",\"presented\":true}"),
		1,
	)
	framebufferEvidence := mustCompactJSONSnippet(`{
		"name": "linux-x64 framebuffer present evidence",
		"kind": "positive",
		"ran": true,
		"pass": true
	}`)
	linuxEvidenceCases := framebufferEvidence +
		compactJSONArrayLineRaw(`{
			"name": "linux-x64 host event sequence",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`) +
		compactJSONArrayLineRaw(`{
			"name": "linux-x64 app-presented RGBA checksum",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`) +
		compactJSONArrayLineRaw(`{
			"name": "linux-x64 counter component app-presented frame",
			"kind": "positive",
			"ran": true,
			"pass": true
		}`)
	raw = strings.Replace(raw, framebufferEvidence, linuxEvidenceCases, 1)
	return []byte(raw)
}
func validLinuxX64RealWindowSurfaceReportJSON() []byte {
	raw := string(validLinuxX64SurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{
			old: `"source": "examples/surface/runtime/surface_counter.tetra"`,
			new: `"source": "examples/surface/runtime/surface_window_counter.tetra"`,
		},
		{
			old: ("\"host_evidence\": {\"level\":\"linux-x64-memfd-starter\"," +
				"\"backend\":\"memfd-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false}"),
			new: ("\"host_evidence\": {\"level\":\"linux-x64-real-window\",\"backend\":" +
				"\"wayland-shm-rgba\",\"framebuffer\":true,\"real_window\":true," +
				"\"native_input\":true,\"user_facing_platform_widgets\":false}"),
		},
		{
			old: `examples/surface/runtime/surface_counter.tetra`,
			new: `examples/surface/runtime/surface_window_counter.tetra`,
		},
		{
			old: `/tmp/surface-artifacts/surface-counter`,
			new: `/tmp/surface-artifacts/surface-window-counter`,
		},
		{
			old: `examples.surface.runtime.surface_counter.CounterApp`,
			new: `examples.surface.runtime.surface_window_counter.CounterApp`,
		},
		{
			old: `examples.surface.runtime.surface_counter.CounterButton`,
			new: `examples.surface.runtime.surface_window_counter.CounterButton`,
		},
	}
	for _, repl := range replacements {
		raw = strings.ReplaceAll(raw, repl.old, repl.new)
	}
	counterProbeProcesses := compactJSONArrayLineRaw(`{
		"name": "surface linux-x64 event sequence probe build",
		"kind": "build",
		"path": "/tmp/tetra build event sequence probe",
		"ran": true,
		"pass": true,
		"exit_code": 0
	}`) + compactJSONArrayLineRaw(`{
		"name": "surface linux-x64 event sequence probe",
		"kind": "app",
		"path": "/tmp/surface-event-sequence-probe",
		"ran": true,
		"pass": true,
		"exit_code": 42,
		"expected_exit_code": 42
	}`) + compactJSONArrayLineRaw(`{
		"name": "surface linux-x64 counter app presented frame probe",
		"kind": "app",
		"path": "/tmp/surface-artifacts/surface-window-counter-present-probe",
		"ran": true,
		"pass": true,
		"exit_code": -1,
		"expected_exit_code": -1
	}`)
	realWindowProbe := compactJSONArrayLineRaw(`{
		"name": "surface linux-x64 real-window probe",
		"kind": "app",
		"path": "/tmp/surface-artifacts/surface-real-window-probe",
		"ran": true,
		"pass": true,
		"exit_code": 42,
		"expected_exit_code": 42
	}`)
	raw = strings.Replace(raw, counterProbeProcesses, realWindowProbe, 1)
	raw = strings.Replace(
		raw,
		("{\"order\":4,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"44444444444444444444444444444444444444444444444444444444444" +
			"44444\",\"presented\":true}"),
		("{\"order\":4,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"44444444444444444444444444444444444444444444444444444444444" +
			"44444\",\"presented\":true},\n    {\"order\":5,\"width\":400," +
			"\"height\":240,\"stride\":1600,\"checksum\":" +
			"\"55555555555555555555555555555555555555555555555555555555555" +
			"55555\",\"presented\":true}"),
		1,
	)
	raw = strings.Replace(
		raw,
		("{\"order\":3,\"kind\":\"text_input\",\"target_component\":" +
			"\"CounterButton\",\"dispatch_path\":[\"CounterApp\"," +
			"\"CounterButton\"],\"handled\":true,\"pass\":true,\"x\":0,\"y\":0," +
			"\"key\":0,\"width\":320,\"height\":200,\"timestamp_ms\":1," +
			"\"text_len\":2,\"text_bytes_hex\":\"4f4b\",\"buffer_slots\":[8,0,0," +
			"0,0,320,200,1,2],\"before_state\":{\"CounterApp.text_count\":" +
			"\"0\",\"CounterButton.text_len_seen\":\"0\"},\"after_state\":" +
			"{\"CounterApp.text_count\":\"1\",\"CounterButton.text_len_seen\":" +
			"\"2\"}}"),
		("{\"order\":3,\"kind\":\"key_down\",\"target_component\":" +
			"\"CounterApp\",\"dispatch_path\":[\"CounterApp\"],\"handled\":true," +
			"\"pass\":true,\"x\":0,\"y\":0,\"key\":32,\"width\":320,\"height\":200," +
			"\"timestamp_ms\":1,\"buffer_slots\":[6,0,0,0,32,320,200,1,0]," +
			"\"before_state\":{\"CounterApp.key_count\":\"0\"},\"after_state\":" +
			"{\"CounterApp.key_count\":\"1\"}},\n    {\"order\":4,\"kind\":" +
			"\"resize\",\"target_component\":\"CounterApp\",\"dispatch_path\":" +
			"[\"CounterApp\"],\"handled\":true,\"pass\":true,\"x\":0,\"y\":0,\"key\":" +
			"0,\"width\":400,\"height\":240,\"timestamp_ms\":2,\"buffer_slots\":" +
			"[2,0,0,0,0,400,240,2,0],\"before_state\":{\"CounterApp.width\":" +
			"\"320\",\"CounterApp.height\":\"200\"},\"after_state\":" +
			"{\"CounterApp.width\":\"400\",\"CounterApp.height\":\"240\"}},\n    " +
			"{\"order\":5,\"kind\":\"text_input\",\"target_component\":" +
			"\"CounterButton\",\"dispatch_path\":[\"CounterApp\"," +
			"\"CounterButton\"],\"handled\":true,\"pass\":true,\"x\":0,\"y\":0," +
			"\"key\":0,\"width\":400,\"height\":240,\"timestamp_ms\":3," +
			"\"text_len\":2,\"text_bytes_hex\":\"4f4b\",\"buffer_slots\":[8,0,0," +
			"0,0,400,240,3,2],\"before_state\":{\"CounterApp.text_count\":" +
			"\"0\",\"CounterButton.text_len_seen\":\"0\"},\"after_state\":" +
			"{\"CounterApp.text_count\":\"1\",\"CounterButton.text_len_seen\":" +
			"\"2\"}},\n    {\"order\":6,\"kind\":\"close\",\"target_component\":" +
			"\"CounterApp\",\"dispatch_path\":[\"CounterApp\"],\"handled\":true," +
			"\"pass\":true,\"x\":0,\"y\":0,\"key\":0,\"width\":400,\"height\":240," +
			"\"timestamp_ms\":4,\"buffer_slots\":[1,0,0,0,0,400,240,4,0]," +
			"\"before_state\":{\"CounterApp.closed\":\"false\"},\"after_state\":" +
			"{\"CounterApp.closed\":\"true\"}}"),
		1,
	)
	raw = strings.Replace(
		raw,
		("{\"order\":2,\"component\":\"CounterApp\",\"field\":\"text_count\"," +
			"\"before\":\"0\",\"after\":\"1\",\"cause\":\"text_input\"}"),
		("{\"order\":2,\"component\":\"CounterApp\",\"field\":\"key_count\"," +
			"\"before\":\"0\",\"after\":\"1\",\"cause\":\"key_down\"},\n    {\"order\":" +
			"3,\"component\":\"CounterApp\",\"field\":\"width\",\"before\":\"320\"," +
			"\"after\":\"400\",\"cause\":\"resize\"},\n    {\"order\":4,\"component\":" +
			"\"CounterApp\",\"field\":\"text_count\",\"before\":\"0\",\"after\":\"1\"," +
			"\"cause\":\"text_input\"},\n    {\"order\":5,\"component\":" +
			"\"CounterApp\",\"field\":\"closed\",\"before\":\"false\",\"after\":" +
			"\"true\",\"cause\":\"close\"}"),
		1,
	)
	raw = strings.Replace(
		raw,
		("{\"name\":\"linux-x64 counter component app-presented frame\"," +
			"\"kind\":\"positive\",\"ran\":true,\"pass\":true}"),
		("{\"name\":\"linux-x64 counter component app-presented frame\"," +
			"\"kind\":\"positive\",\"ran\":true,\"pass\":true},\n    {\"name\":" +
			"\"linux-x64 real-window surface\",\"kind\":\"positive\",\"ran\":" +
			"true,\"pass\":true},\n    {\"name\":\"linux-x64 native input " +
			"event pump\",\"kind\":\"positive\",\"ran\":true,\"pass\":true},\n    " +
			"{\"name\":\"linux-x64 real-window resize event\",\"kind\":" +
			"\"positive\",\"ran\":true,\"pass\":true},\n    {\"name\":\"linux-x64 " +
			"real-window close event\",\"kind\":\"positive\",\"ran\":true," +
			"\"pass\":true}"),
		1,
	)
	return []byte(raw)
}
func validWASM32WebSurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "wasm32-web"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-wasm32-web"`},
		{
			old: ("\"host_evidence\": {\"level\":\"deterministic-headless\"," +
				"\"backend\":\"software-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false}"),
			new: ("\"host_evidence\": {\"level\":" +
				"\"wasm32-web-compiler-owned-loader\",\"backend\":" +
				"\"node-surface-host\",\"framebuffer\":true,\"real_window\":false," +
				"\"native_input\":false,\"user_facing_platform_widgets\":false}"),
		},
		{
			old: ("{\"name\":\"surface component app\",\"kind\":\"app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter\",\"ran\":true,\"pass\":" +
				"true,\"exit_code\":1,\"expected_exit_code\":1}"),
			new: ("{\"name\":\"surface wasm32-web component app\",\"kind\":\"app\"," +
				"\"path\":\"node scripts/tools/web_run_module.mjs " +
				"/tmp/surface-artifacts/surface-counter.wasm\",\"ran\":true," +
				"\"pass\":true,\"exit_code\":1,\"expected_exit_code\":1},\n    " +
				"{\"name\":\"surface wasm32-web import validator\",\"kind\":" +
				"\"runtime\",\"path\":\"go run ./tools/cmd/validate-wasm-imports " +
				"--target wasm32-web " +
				"/tmp/surface-artifacts/surface-counter.wasm\",\"ran\":true," +
				"\"pass\":true,\"exit_code\":0}"),
		},
		{
			old: ("{\"kind\":\"component-app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter\",\"sha256\":\"sha256:" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaa\",\"size\":49172}"),
			new: ("{\"kind\":\"component-app\",\"path\":" +
				"\"/tmp/surface-artifacts/surface-counter.wasm\",\"sha256\":" +
				"\"sha256:" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaa\",\"size\":7502}"),
		},
		{old: strings.Join([]string{
			"\"surface headless runtime\"",
		}, "\n"), new: strings.Join([]string{
			"\"surface wasm32-web runtime\"",
		}, "\n")},
		{old: strings.Join([]string{
			"\"headless event dispatch\"",
		}, "\n"), new: strings.Join([]string{
			"\"wasm32-web Surface Host ABI imports\"",
		}, "\n")},
		{old: strings.Join([]string{
			"\"headless framebuffer checksum\"",
		}, "\n"), new: strings.Join([]string{
			"\"wasm32-web framebuffer checksum evidence\"",
		}, "\n")},
		{
			old: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2`,
			new: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":3`,
		},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(
		raw,
		compactJSONArrayLineRaw(`{
			"kind": "runner-trace",
			"path": "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size": 409
		}`),
		``,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	componentAppArtifact := mustCompactJSONSnippet(`{
		"kind": "component-app",
		"path": "/tmp/surface-artifacts/surface-counter.wasm",
		"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"size": 7502
	}`)
	oldArtifacts := "\"artifacts\": [\n    " + componentAppArtifact + "\n  ]"
	newArtifacts := "\"artifacts\": [\n    " + componentAppArtifact +
		compactJSONArrayLineRaw(`{
			"kind": "compiler-owned-loader",
			"path": "/tmp/surface-artifacts/surface-counter.mjs",
			"sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"size": 4931
		}`) +
		compactJSONArrayLineRaw(`{
			"kind": "runner-trace",
			"path": "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size": 413
		}`) + "\n  ]"
	raw = strings.Replace(raw, oldArtifacts, newArtifacts, 1)
	raw = strings.Replace(
		raw,
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"22222222222222222222222222222222222222222222222222222222222" +
			"22222\",\"presented\":true}"),
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"22222222222222222222222222222222222222222222222222222222222" +
			"22222\",\"presented\":true},\n    {\"order\":3,\"width\":320," +
			"\"height\":200,\"stride\":1280,\"checksum\":" +
			"\"33333333333333333333333333333333333333333333333333333333333" +
			"33333\",\"presented\":true},\n    {\"order\":4,\"width\":320," +
			"\"height\":200,\"stride\":1280,\"checksum\":" +
			"\"44444444444444444444444444444444444444444444444444444444444" +
			"44444\",\"presented\":true}"),
		1,
	)
	raw = strings.Replace(
		raw,
		("{\"name\":\"wasm32-web framebuffer checksum evidence\"," +
			"\"kind\":\"positive\",\"ran\":true,\"pass\":true}"),
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`,
		1,
	)
	return []byte(raw)
}
func validWASM32WebBrowserCanvasSurfaceReportJSON() []byte {
	return []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.runtime.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"wasm32-web\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"surface-wasm32-web\",",
		"  \"surface_schema\": \"tetra.surface.v1\",",
		"  \"host_abi\": \"tetra.surface.host-abi.v1\",",
		("  \"host_evidence\": {\"level\":\"wasm32-web-browser-canvas-input\",\"back" +
			"end\":\"browser-canvas-rgba\",\"framebuffer\":true,\"real_window\":false," +
			"\"native_input\":true,\"user_facing_platform_widgets\":false},"),
		("  \"source\": \"examples/surface/runtime/surface_browser_counter.tetra\"" +
			","),
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra build --" +
			"target wasm32-web examples/surface/runtime/surface_browser_counter.t" +
			"etra -o /tmp/surface-artifacts/surface-browser-counter.wasm\",\"ran\":t" +
			"rue,\"pass\":true,\"exit_code\":0},"),
		("    {\"name\":\"surface wasm32-web browser canvas component app\",\"kind\"" +
			":\"app\",\"path\":\"/usr/bin/chromium --headless <surface-browser-canvas-" +
			"runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm\",\"r" +
			"an\":true,\"pass\":true,\"exit_code\":0,\"expected_exit_code\":0},"),
		("    {\"name\":\"surface wasm32-web import validator\",\"kind\":\"runtime\"" +
			",\"path\":\"go run ./tools/cmd/validate-wasm-imports --target wasm32-we" +
			"b /tmp/surface-artifacts/surface-browser-counter.wasm\",\"ran\":true,\"p" +
			"ass\":true,\"exit_code\":0},"),
		("    {\"name\":\"surface wasm32-web browser canvas runtime\",\"kind\":\"run" +
			"time\",\"path\":\"Chromium fixture\",\"ran\":true,\"pass\":true,\"exit_cod" +
			"e\":0}"),
		"  ],",
		"  \"artifacts\": [",
		("    {\"kind\":\"component-app\",\"path\":\"/tmp/surface-artifacts/surface-" +
			"browser-counter.wasm\",\"sha256\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"size\":8604},"),
		("    {\"kind\":\"compiler-owned-loader\",\"path\":\"/tmp/surface-artifacts/" +
			"surface-browser-counter.mjs\",\"sha256\":\"sha256:bbbbbbbbbbbbbbbbbbbbbb" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"size\":4939},"),
		("    {\"kind\":\"runner-trace\",\"path\":\"/tmp/surface-artifacts/surface-r" +
			"unner-trace.json\",\"sha256\":\"sha256:ccccccccccccccccccccccccccccccccc" +
			"ccccccccccccccccccccccccccccccc\",\"size\":1184}"),
		"  ],",
		("  \"artifact_scan\": {\"root\":\"/tmp/surface-artifacts\",\"files_checked" +
			"\":3,\"forbidden_paths\":[],\"pass\":true},"),
		"  \"components\": [",
		("    {\"id\":\"CounterApp\",\"type\":\"examples.surface.runtime.surface_bro" +
			"wser_counter.CounterApp\",\"bounds\":{\"x\":0,\"y\":0,\"w\":400,\"h\":240}" +
			",\"abilities\":[\"measure\",\"layout\",\"draw\",\"event\",\"focus\",\"text" +
			"\",\"accessibility\"],\"state\":{\"count\":\"2\",\"key_count\":\"1\",\"wid" +
			"th\":\"400\",\"accessibility_role\":\"button\"}},"),
		("    {\"id\":\"CounterButton\",\"type\":\"examples.surface.runtime.surface_" +
			"browser_counter.CounterButton\",\"parent\":\"CounterApp\",\"bounds\":{\"x" +
			"\":32,\"y\":88,\"w\":160,\"h\":48},\"abilities\":[\"measure\",\"layout\"," +
			"\"draw\",\"event\",\"focus\",\"text\",\"accessibility\"],\"state\":{\"focu" +
			"sed\":\"true\",\"text_len_seen\":\"2\"}}"),
		"  ],",
		"  \"events\": [",
		("    {\"order\":1,\"kind\":\"mouse_up\",\"target_component\":\"CounterButto" +
			"n\",\"dispatch_path\":[\"CounterApp\",\"CounterButton\"],\"handled\":true," +
			"\"pass\":true,\"x\":48,\"y\":96,\"key\":0,\"width\":320,\"height\":200,\"t" +
			"imestamp_ms\":0,\"buffer_slots\":[5,48,96,1,0,320,200,0,0],\"before_stat" +
			"e\":{\"CounterApp.count\":\"0\"},\"after_state\":{\"CounterApp.count\":\"1" +
			"\"}},"),
		("    {\"order\":2,\"kind\":\"key_down\",\"target_component\":\"CounterApp\"" +
			",\"dispatch_path\":[\"CounterApp\"],\"handled\":true,\"pass\":true,\"x\":0" +
			",\"y\":0,\"key\":32,\"width\":320,\"height\":200,\"timestamp_ms\":1,\"buff" +
			"er_slots\":[6,0,0,0,32,320,200,1,0],\"before_state\":{\"CounterApp.count" +
			"\":\"1\",\"CounterApp.key_count\":\"0\"},\"after_state\":{\"CounterApp.cou" +
			"nt\":\"2\",\"CounterApp.key_count\":\"1\"}},"),
		("    {\"order\":3,\"kind\":\"resize\",\"target_component\":\"CounterApp\"," +
			"\"dispatch_path\":[\"CounterApp\"],\"handled\":true,\"pass\":true,\"x\":0," +
			"\"y\":0,\"key\":0,\"width\":400,\"height\":240,\"timestamp_ms\":2,\"buffer" +
			"_slots\":[2,0,0,0,0,400,240,2,0],\"before_state\":{\"CounterApp.width\":\"" +
			"320\"},\"after_state\":{\"CounterApp.width\":\"400\"}},"),
		("    {\"order\":4,\"kind\":\"text_input\",\"target_component\":\"CounterBut" +
			"ton\",\"dispatch_path\":[\"CounterApp\",\"CounterButton\"],\"handled\":tru" +
			"e,\"pass\":true,\"x\":0,\"y\":0,\"key\":0,\"width\":400,\"height\":240,\"t" +
			"imestamp_ms\":3,\"text_len\":2,\"text_bytes_hex\":\"4f4b\",\"buffer_slots" +
			"\":[8,0,0,0,0,400,240,3,2],\"before_state\":{\"CounterButton.text_len_se" +
			"en\":\"0\"},\"after_state\":{\"CounterButton.text_len_seen\":\"2\"}}"),
		"  ],",
		"  \"frames\": [",
		("    {\"order\":1,\"width\":320,\"height\":200,\"stride\":1280,\"checksum\"" +
			":\"1111111111111111111111111111111111111111111111111111111111111111\"," +
			"\"presented\":true},"),
		("    {\"order\":5,\"width\":400,\"height\":240,\"stride\":1600,\"checksum\"" +
			":\"5555555555555555555555555555555555555555555555555555555555555555\"," +
			"\"presented\":true}"),
		"  ],",
		"  \"state_transitions\": [",
		("    {\"order\":1,\"component\":\"CounterApp\",\"field\":\"count\",\"before" +
			"\":\"0\",\"after\":\"1\",\"cause\":\"mouse_up\"},"),
		("    {\"order\":2,\"component\":\"CounterApp\",\"field\":\"key_count\",\"be" +
			"fore\":\"0\",\"after\":\"1\",\"cause\":\"key_down\"},"),
		("    {\"order\":3,\"component\":\"CounterApp\",\"field\":\"width\",\"before" +
			"\":\"320\",\"after\":\"400\",\"cause\":\"resize\"},"),
		("    {\"order\":4,\"component\":\"CounterButton\",\"field\":\"text_len_seen" +
			"\",\"before\":\"0\",\"after\":\"2\",\"cause\":\"text_input\"}"),
		"  ],",
		"  \"cases\": [",
		("    {\"name\":\"pure Tetra component app\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"wasm32-web Surface Host ABI imports\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"compiler-owned wasm Surface loader\",\"kind\":\"positive\"" +
			",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas surface\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas RGBA readback\",\"kind\":\"posit" +
			"ive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas pointer input\",\"kind\":\"posit" +
			"ive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas keyboard input\",\"kind\":\"posi" +
			"tive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas resize input\",\"kind\":\"positi" +
			"ve\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"wasm32-web browser canvas text input\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"compiler-owned browser canvas Surface host\",\"kind\":\"po" +
			"sitive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host-provided pointer event dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host event buffer poll_event\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"pre/post event frame sequence\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"component hierarchy dispatch\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"component text input scalar dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host text payload buffer\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component focus dispatch\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component accessibility metadata\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"no legacy UI sidecar artifacts\",\"kind\":\"positive\",\"r" +
			"an\":true,\"pass\":true},"),
		("    {\"name\":\"state transition\",\"kind\":\"positive\",\"ran\":true,\"pa" +
			"ss\":true},"),
		("    {\"name\":\"reject legacy UI evidence\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"legacy UI evidence rejected\"}"),
		"  ]",
		"}",
		"",
	}, "\n"))
}

// ---- report_text_input_test.go ----

func TestValidateReportAcceptsHeadlessTextFocusInputSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessTextFocusInputSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTextInputReportAcceptsProductionBaseline(t *testing.T) {
	raw := validSurfaceTextInputReportJSON()
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTextInputReportRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "missing utf8 validation",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"utf8_validation": true`, `"utf8_validation": false`, 1)
			},
			want: "utf8_validation",
		},
		{
			name: "missing invalid utf8 rejection",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"invalid_utf8_rejected": true`,
					`"invalid_utf8_rejected": false`,
					1,
				)
			},
			want: "invalid_utf8_rejected",
		},
		{
			name: "missing multiline storage",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"multiline": true`, `"multiline": false`, 1)
			},
			want: "multiline",
		},
		{
			name: "missing composition commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"composition_commit": true`, `"composition_commit": false`, 1)
			},
			want: "composition_commit",
		},
		{
			name: "missing clipboard write",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_write": true`, `"clipboard_write": false`, 1)
			},
			want: "clipboard_write",
		},
		{
			name: "missing clipboard host abi",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_host_abi": true`, `"clipboard_host_abi": false`, 1)
			},
			want: "clipboard_host_abi",
		},
		{
			name: "missing composition trace commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"commit":true`, `"commit":false`, 1)
			},
			want: "composition_trace.commit",
		},
		{
			name: "missing clipboard owned copy",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_owned_copy": true`, `"clipboard_owned_copy": false`, 1)
			},
			want: "clipboard_owned_copy",
		},
		{
			name: "borrowed view storage",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"borrowed_view_storage": false`,
					`"borrowed_view_storage": true`,
					1,
				)
			},
			want: "borrowed_view_storage",
		},
		{
			name: "missing safe view lifetime",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"safe_view_lifetime_checked": true`,
					`"safe_view_lifetime_checked": false`,
					1,
				)
			},
			want: "safe_view_lifetime_checked",
		},
		{
			name: "missing target host composition trace",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"target_host_composition_trace": true`,
					`"target_host_composition_trace": false`,
					1,
				)
			},
			want: "target_host_composition_trace",
		},
		{
			name: "rich text production claim",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"rich_text_production_claim": false`,
					`"rich_text_production_claim": true`,
					1,
				)
			},
			want: "rich_text_production_claim",
		},
		{
			name: "bidi production claim",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"bidi_production_claim": false`,
					`"bidi_production_claim": true`,
					1,
				)
			},
			want: "bidi_production_claim",
		},
		{
			name: "missing settings reference trace",
			mutate: func(raw string) string {
				return strings.Replace(raw, ("    {\"source\":" +
					"\"examples/surface/morph_core/surface_morph_settings.tetra\"," +
					"\"trace\":\"settings text field trace\",\"focus\":true," +
					"\"selection\":true,\"clipboard\":true,\"composition\":true," +
					"\"multiline\":true,\"pass\":true},\n"), "", 1)
			},
			want: "examples/surface/morph_core/surface_morph_settings.tetra",
		},
		{
			name: "shaping plan claims bidi",
			mutate: func(raw string) string {
				return strings.Replace(
					raw,
					`"bidi":"nonclaim-full-bidi-v1"`,
					`"bidi":"full-bidi-production-v1"`,
					1,
				)
			},
			want: "text_shaping_plan.bidi",
		},
		{
			name: "missing target evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "target": "headless",`+"\n", "", 1)
			},
			want: "target",
		},
		{
			name: "missing process evidence",
			mutate: func(raw string) string {
				processes := "\"processes\": [\n    " + compactJSONSnippet(t, `{
					"name": "tetra build",
					"kind": "build",
					"path": "`+("tetra build --target linux-x64 "+
					"examples/surface/release/surface_release_text_input.tetra "+
					"-o /tmp/surface-artifacts/surface-release-text-input")+`",
					"ran": true,
					"pass": true,
					"exit_code": 0
				}`) + compactJSONArrayLine(t, `{
					"name": "surface component app",
					"kind": "app",
					"path": "/tmp/surface-artifacts/surface-release-text-input",
					"ran": true,
					"pass": true,
					"exit_code": 1,
					"expected_exit_code": 1
				}`) + compactJSONArrayLine(t, `{
					"name": "surface headless runtime",
					"kind": "runtime",
					"path": "tools/cmd/surface-runtime-smoke --mode headless-release-text-input",
					"ran": true,
					"pass": true,
					"exit_code": 0
				}`) + "\n  ]"
				return strings.Replace(raw, processes, `"processes": []`, 1)
			},
			want: "process evidence",
		},
		{
			name: "missing composition case evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, (("    {\"name\":\"release text input composition commit\"," +
					"\"kind\":") +
					"\"positive\",\"ran\":true,\"pass\":true},\n"), "", 1)
			},
			want: "composition commit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceTextInputReportJSON())))
			err := ValidateTextInputReport(raw)
			if err == nil {
				t.Fatalf("expected text-input report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsTextFocusInputMissingCaretAndDeleteEvidence(t *testing.T) {
	raw := strings.Replace(
		string(validHeadlessTextFocusInputSurfaceReportJSON()),
		`"caret":"1",`,
		``,
		1,
	)
	raw = strings.Replace(raw, `,
    {"name":"text focus input backspace delete","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without caret/delete evidence to fail")
	}
	for _, want := range []string{"caret", "backspace delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsTextFocusInputMissingTabRoutingEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessTextFocusInputSurfaceReportJSON()), `,
    {"name":"text focus input Tab changes focus","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(
		raw,
		("{\"order\":7,\"kind\":\"key_down\",\"target_component\":" +
			"\"TextInputApp\",\"dispatch_path\":[\"TextInputApp\"],\"handled\":" +
			"true,\"pass\":true,\"x\":0,\"y\":0,\"key\":9"),
		("{\"order\":7,\"kind\":\"key_down\",\"target_component\":" +
			"\"TextInputApp\",\"dispatch_path\":[\"TextInputApp\"],\"handled\":" +
			"true,\"pass\":true,\"x\":0,\"y\":0,\"key\":0"),
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without Tab routing evidence to fail")
	}
	for _, want := range []string{"Tab", "focus routing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func validSurfaceTextInputReportJSON() []byte {
	return []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.text-input.v1\",",
		"  \"target\": \"headless\",",
		("  \"source\": \"examples/surface/release/surface_release_text_input.tet" +
			"ra\","),
		"  \"level\": \"production-text-input-v1\",",
		"  \"experimental\": false,",
		"  \"production_claim\": true,",
		"  \"storage\": \"owned-utf8-byte-buffer\",",
		"  \"utf8_validation\": true,",
		"  \"invalid_utf8_rejected\": true,",
		"  \"caret\": true,",
		"  \"selection\": true,",
		"  \"selection_clipboard_transfer\": true,",
		"  \"multiline\": true,",
		"  \"backspace\": true,",
		"  \"delete\": true,",
		"  \"home_end\": true,",
		"  \"arrow_left_right\": true,",
		"  \"composition_events\": true,",
		"  \"composition_commit\": true,",
		"  \"composition_cancel\": true,",
		"  \"clipboard_read\": true,",
		"  \"clipboard_write\": true,",
		"  \"clipboard_host_abi\": true,",
		"  \"clipboard_owned_copy\": true,",
		"  \"target_host_composition_trace\": true,",
		("  \"composition_trace\": {\"start\":true,\"update\":true,\"commit\":true," +
			"\"cancel\":true},"),
		("  \"text_shaping_plan\": {\"quality_level\":\"scoped-text-shaping-plan-v1" +
			"\",\"fallback_fonts\":true,\"grapheme_boundaries\":\"byte-offset-codepoint" +
			"-v1\",\"line_breaking\":\"newline-storage-plus-wrap-plan-v1\",\"bidi\":\"n" +
			"onclaim-full-bidi-v1\",\"rich_text\":\"nonclaim-rich-text-editor-v1\"},"),
		"  \"reference_traces\": [",
		("    {\"source\":\"examples/surface/morph_core/surface_morph_settings.te" +
			"tra\",\"trace\":\"settings text field trace\",\"focus\":true,\"selection\"" +
			":true,\"clipboard\":true,\"composition\":true,\"multiline\":true,\"pass\":" +
			"true},"),
		("    {\"source\":\"examples/surface/morph_core/surface_morph_editor_shel" +
			"l.tetra\",\"trace\":\"editor shell text area trace\",\"focus\":true,\"sele" +
			"ction\":true,\"clipboard\":true,\"composition\":true,\"multiline\":true,\"" +
			"pass\":true}"),
		"  ],",
		("  \"unsupported_claims\": [\"full-rich-text-editor\",\"full-bidi-shaping\"" +
			",\"grapheme-cluster-caret\",\"ide-grade-editor\"],"),
		"  \"rich_text_production_claim\": false,",
		"  \"bidi_production_claim\": false,",
		"  \"full_editor_production_claim\": false,",
		"  \"borrowed_view_storage\": false,",
		"  \"safe_view_lifetime_checked\": true,",
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra build --" +
			"target linux-x64 examples/surface/release/surface_release_text_input" +
			".tetra -o /tmp/surface-artifacts/surface-release-text-input\",\"ran\":t" +
			"rue,\"pass\":true,\"exit_code\":0},"),
		("    {\"name\":\"surface component app\",\"kind\":\"app\",\"path\":\"/tmp/s" +
			"urface-artifacts/surface-release-text-input\",\"ran\":true,\"pass\":true," +
			"\"exit_code\":1,\"expected_exit_code\":1},"),
		("    {\"name\":\"surface headless runtime\",\"kind\":\"runtime\",\"path\":" +
			"\"tools/cmd/surface-runtime-smoke --mode headless-release-text-input\"" +
			",\"ran\":true,\"pass\":true,\"exit_code\":0}"),
		"  ],",
		"  \"artifacts\": [",
		("    {\"kind\":\"component-app\",\"path\":\"/tmp/surface-artifacts/surface-" +
			"release-text-input\",\"sha256\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"size\":4096},"),
		("    {\"kind\":\"runner-trace\",\"path\":\"/tmp/surface-artifacts/surface-r" +
			"unner-trace.json\",\"sha256\":\"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"size\":2048}"),
		"  ],",
		("  \"artifact_scan\": {\"root\":\"/tmp/surface-artifacts\",\"files_checked" +
			"\":2,\"forbidden_paths\":[],\"pass\":true},"),
		"  \"cases\": [",
		("    {\"name\":\"host-provided pointer event dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host event buffer poll_event\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"pre/post event frame sequence\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"component hierarchy dispatch\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"component text input scalar dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host text payload buffer\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component focus dispatch\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component accessibility metadata\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"no legacy UI sidecar artifacts\",\"kind\":\"positive\",\"r" +
			"an\":true,\"pass\":true},"),
		("    {\"name\":\"release text input ASCII insertion\",\"kind\":\"positive\"" +
			",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input UTF-8 insertion\",\"kind\":\"positive\"" +
			",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input invalid UTF-8 rejected\",\"kind\":\"neg" +
			"ative\",\"ran\":true,\"pass\":true,\"expected_error\":\"invalid utf8 rejec" +
			"ted\"},"),
		("    {\"name\":\"release text input multiline storage\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input caret home end arrows\",\"kind\":\"posi" +
			"tive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input selection replacement\",\"kind\":\"posi" +
			"tive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input selection clipboard transfer\",\"kind\"" +
			":\"positive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input backspace delete\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input clipboard owned copy transfer\",\"kind" +
			"\":\"positive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input composition start update\",\"kind\":\"p" +
			"ositive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input composition commit\",\"kind\":\"positiv" +
			"e\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input composition cancel\",\"kind\":\"positiv" +
			"e\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input shaping plan scoped\",\"kind\":\"positi" +
			"ve\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"settings reference text input trace\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"editor reference text input trace\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"release text input safe view lifetime checked\",\"kind\":" +
			"\"positive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"reject legacy UI evidence\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"legacy UI evidence rejected\"}"),
		"  ]",
		"}",
		"",
	}, "\n"))
}
func validHeadlessTextFocusInputSurfaceReportJSON() []byte {
	return []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.runtime.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"headless\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"surface-headless\",",
		"  \"surface_schema\": \"tetra.surface.v1\",",
		"  \"host_abi\": \"tetra.surface.host-abi.v1\",",
		("  \"host_evidence\": {\"level\":\"deterministic-headless\",\"backend\":\"s" +
			"oftware-rgba\",\"framebuffer\":true,\"real_window\":false,\"native_input\"" +
			":false,\"user_facing_platform_widgets\":false},"),
		"  \"source\": \"examples/surface/runtime/surface_textbox_app.tetra\",",
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"tetra build --" +
			"target linux-x64 examples/surface/runtime/surface_textbox_app.tetra " +
			"-o /tmp/surface-artifacts/surface-textbox-app\",\"ran\":true,\"pass\":tru" +
			"e,\"exit_code\":0},"),
		("    {\"name\":\"surface component app\",\"kind\":\"app\",\"path\":\"/tmp/s" +
			"urface-artifacts/surface-textbox-app\",\"ran\":true,\"pass\":true,\"exit_c" +
			"ode\":1,\"expected_exit_code\":1},"),
		("    {\"name\":\"surface headless runtime\",\"kind\":\"runtime\",\"path\":" +
			"\"tools/cmd/surface-runtime-smoke\",\"ran\":true,\"pass\":true,\"exit_code" +
			"\":0}"),
		"  ],",
		"  \"artifacts\": [",
		("    {\"kind\":\"component-app\",\"path\":\"/tmp/surface-artifacts/surface-" +
			"textbox-app\",\"sha256\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaa\",\"size\":69657},"),
		("    {\"kind\":\"runner-trace\",\"path\":\"/tmp/surface-artifacts/surface-r" +
			"unner-trace.json\",\"sha256\":\"sha256:ccccccccccccccccccccccccccccccccc" +
			"ccccccccccccccccccccccccccccccc\",\"size\":13015}"),
		"  ],",
		("  \"artifact_scan\": {\"root\":\"/tmp/surface-artifacts\",\"files_checked" +
			"\":2,\"forbidden_paths\":[],\"pass\":true},"),
		"  \"components\": [",
		("    {\"id\":\"TextInputApp\",\"type\":\"examples.surface.runtime.surface_t" +
			"extbox_app.TextInputApp\",\"bounds\":{\"x\":0,\"y\":0,\"w\":400,\"h\":240}" +
			",\"abilities\":[\"measure\",\"layout\",\"draw\",\"event\",\"focus\",\"text" +
			"\",\"accessibility\"],\"state\":{\"focused_component\":\"SubmitButton\",\"" +
			"width\":\"400\",\"height\":\"240\",\"resize_count\":\"1\",\"accessibility_" +
			"role\":\"none\"}},"),
		("    {\"id\":\"TextBox\",\"type\":\"examples.surface.runtime.surface_textbo" +
			"x_app.TextBox\",\"parent\":\"TextInputApp\",\"bounds\":{\"x\":32,\"y\":64," +
			"\"w\":224,\"h\":44},\"abilities\":[\"measure\",\"layout\",\"draw\",\"event" +
			"\",\"focus\",\"text\",\"accessibility\"],\"state\":{\"focused\":\"false\"," +
			"\"buffer\":\"Z\",\"caret\":\"1\",\"text_len\":\"1\",\"backspace_count\":\"" +
			"1\",\"delete_count\":\"1\",\"accessibility_role\":\"label\"}},"),
		("    {\"id\":\"SubmitButton\",\"type\":\"examples.surface.runtime.surface_t" +
			"extbox_app.ActionButton\",\"parent\":\"TextInputApp\",\"bounds\":{\"x\":32" +
			",\"y\":128,\"w\":128,\"h\":44},\"abilities\":[\"measure\",\"layout\",\"dra" +
			"w\",\"event\",\"focus\",\"text\",\"accessibility\"],\"state\":{\"focused\"" +
			":\"true\",\"press_count\":\"1\",\"key_count\":\"1\",\"accessibility_role\"" +
			":\"button\"}}"),
		"  ],",
		"  \"events\": [",
		("    {\"order\":1,\"kind\":\"mouse_up\",\"target_component\":\"TextBox\",\"" +
			"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\":t" +
			"rue,\"x\":48,\"y\":96,\"key\":0,\"width\":320,\"height\":200,\"timestamp_m" +
			"s\":0,\"buffer_slots\":[5,48,96,1,0,320,200,0,0],\"before_state\":{\"TextI" +
			"nputApp.focused_component\":\"none\",\"TextBox.focused\":\"false\"},\"afte" +
			"r_state\":{\"TextInputApp.focused_component\":\"TextBox\",\"TextBox.focuse" +
			"d\":\"true\"}},"),
		("    {\"order\":2,\"kind\":\"text_input\",\"target_component\":\"TextBox\"," +
			"\"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\"" +
			":true,\"x\":0,\"y\":0,\"key\":0,\"width\":320,\"height\":200,\"timestamp_m" +
			"s\":1,\"text_len\":2,\"text_bytes_hex\":\"4f4b\",\"buffer_slots\":[8,0,0,0" +
			",0,320,200,1,2],\"before_state\":{\"TextBox.buffer\":\"\",\"TextBox.caret" +
			"\":\"0\",\"TextBox.text_len\":\"0\"},\"after_state\":{\"TextBox.buffer\":" +
			"\"OK\",\"TextBox.caret\":\"2\",\"TextBox.text_len\":\"2\"}},"),
		("    {\"order\":3,\"kind\":\"key_down\",\"target_component\":\"TextBox\",\"" +
			"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\":t" +
			"rue,\"x\":0,\"y\":0,\"key\":37,\"width\":320,\"height\":200,\"timestamp_ms" +
			"\":2,\"buffer_slots\":[6,0,0,0,37,320,200,2,0],\"before_state\":{\"TextBox" +
			".buffer\":\"OK\",\"TextBox.caret\":\"2\"},\"after_state\":{\"TextBox.buffe" +
			"r\":\"OK\",\"TextBox.caret\":\"1\"}},"),
		("    {\"order\":4,\"kind\":\"key_down\",\"target_component\":\"TextBox\",\"" +
			"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\":t" +
			"rue,\"x\":0,\"y\":0,\"key\":8,\"width\":320,\"height\":200,\"timestamp_ms" +
			"\":3,\"buffer_slots\":[6,0,0,0,8,320,200,3,0],\"before_state\":{\"TextBox." +
			"buffer\":\"OK\",\"TextBox.caret\":\"1\"},\"after_state\":{\"TextBox.buffer" +
			"\":\"K\",\"TextBox.caret\":\"0\"}},"),
		("    {\"order\":5,\"kind\":\"key_down\",\"target_component\":\"TextBox\",\"" +
			"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\":t" +
			"rue,\"x\":0,\"y\":0,\"key\":46,\"width\":320,\"height\":200,\"timestamp_ms" +
			"\":4,\"buffer_slots\":[6,0,0,0,46,320,200,4,0],\"before_state\":{\"TextBox" +
			".buffer\":\"K\",\"TextBox.caret\":\"0\"},\"after_state\":{\"TextBox.buffer" +
			"\":\"\",\"TextBox.caret\":\"0\"}},"),
		("    {\"order\":6,\"kind\":\"text_input\",\"target_component\":\"TextBox\"," +
			"\"dispatch_path\":[\"TextInputApp\",\"TextBox\"],\"handled\":true,\"pass\"" +
			":true,\"x\":0,\"y\":0,\"key\":0,\"width\":320,\"height\":200,\"timestamp_m" +
			"s\":5,\"text_len\":1,\"text_bytes_hex\":\"5a\",\"buffer_slots\":[8,0,0,0,0" +
			",320,200,5,1],\"before_state\":{\"TextBox.buffer\":\"\",\"TextBox.caret\":" +
			"\"0\",\"TextBox.text_len\":\"0\"},\"after_state\":{\"TextBox.buffer\":\"Z" +
			"\",\"TextBox.caret\":\"1\",\"TextBox.text_len\":\"1\"}},"),
		("    {\"order\":7,\"kind\":\"key_down\",\"target_component\":\"TextInputApp" +
			"\",\"dispatch_path\":[\"TextInputApp\"],\"handled\":true,\"pass\":true,\"x" +
			"\":0,\"y\":0,\"key\":9,\"width\":320,\"height\":200,\"timestamp_ms\":6,\"b" +
			"uffer_slots\":[6,0,0,0,9,320,200,6,0],\"before_state\":{\"TextInputApp.f" +
			"ocused_component\":\"TextBox\",\"TextBox.focused\":\"true\",\"SubmitButton" +
			".focused\":\"false\"},\"after_state\":{\"TextInputApp.focused_component\":" +
			"\"SubmitButton\",\"TextBox.focused\":\"false\",\"SubmitButton.focused\":\"" +
			"true\"}},"),
		("    {\"order\":8,\"kind\":\"key_down\",\"target_component\":\"SubmitButton" +
			"\",\"dispatch_path\":[\"TextInputApp\",\"SubmitButton\"],\"handled\":true," +
			"\"pass\":true,\"x\":0,\"y\":0,\"key\":32,\"width\":320,\"height\":200,\"ti" +
			"mestamp_ms\":7,\"buffer_slots\":[6,0,0,0,32,320,200,7,0],\"before_state\"" +
			":{\"SubmitButton.press_count\":\"0\",\"TextBox.buffer\":\"Z\"},\"after_sta" +
			"te\":{\"SubmitButton.press_count\":\"1\",\"TextBox.buffer\":\"Z\"}},"),
		("    {\"order\":9,\"kind\":\"resize\",\"target_component\":\"TextInputApp\"" +
			",\"dispatch_path\":[\"TextInputApp\"],\"handled\":true,\"pass\":true,\"x\"" +
			":0,\"y\":0,\"key\":0,\"width\":400,\"height\":240,\"timestamp_ms\":8,\"buf" +
			"fer_slots\":[2,0,0,0,0,400,240,8,0],\"before_state\":{\"TextInputApp.wid" +
			"th\":\"320\",\"TextInputApp.focused_component\":\"SubmitButton\"},\"after_" +
			"state\":{\"TextInputApp.width\":\"400\",\"TextInputApp.focused_component\"" +
			":\"SubmitButton\"}}"),
		"  ],",
		"  \"frames\": [",
		("    {\"order\":1,\"width\":320,\"height\":200,\"stride\":1280,\"checksum\"" +
			":\"1111111111111111111111111111111111111111111111111111111111111111\"," +
			"\"presented\":true},"),
		("    {\"order\":2,\"width\":400,\"height\":240,\"stride\":1600,\"checksum\"" +
			":\"2222222222222222222222222222222222222222222222222222222222222222\"," +
			"\"presented\":true}"),
		"  ],",
		"  \"state_transitions\": [",
		("    {\"order\":1,\"component\":\"TextInputApp\",\"field\":\"focused_compon" +
			"ent\",\"before\":\"none\",\"after\":\"TextBox\",\"cause\":\"mouse_up\"},"),
		("    {\"order\":2,\"component\":\"TextBox\",\"field\":\"buffer\",\"before\"" +
			":\"\",\"after\":\"OK\",\"cause\":\"text_input\"},"),
		("    {\"order\":3,\"component\":\"TextBox\",\"field\":\"caret\",\"before\":" +
			"\"2\",\"after\":\"1\",\"cause\":\"key_down\"},"),
		("    {\"order\":4,\"component\":\"TextBox\",\"field\":\"buffer\",\"before\"" +
			":\"OK\",\"after\":\"K\",\"cause\":\"backspace\"},"),
		("    {\"order\":5,\"component\":\"TextBox\",\"field\":\"buffer\",\"before\"" +
			":\"K\",\"after\":\"\",\"cause\":\"delete\"},"),
		("    {\"order\":6,\"component\":\"TextBox\",\"field\":\"buffer\",\"before\"" +
			":\"\",\"after\":\"Z\",\"cause\":\"text_input\"},"),
		("    {\"order\":7,\"component\":\"TextInputApp\",\"field\":\"focused_compon" +
			"ent\",\"before\":\"TextBox\",\"after\":\"SubmitButton\",\"cause\":\"tab\"}" +
			","),
		("    {\"order\":8,\"component\":\"SubmitButton\",\"field\":\"press_count\"," +
			"\"before\":\"0\",\"after\":\"1\",\"cause\":\"key_down\"},"),
		("    {\"order\":9,\"component\":\"TextInputApp\",\"field\":\"width\",\"befo" +
			"re\":\"320\",\"after\":\"400\",\"cause\":\"resize\"}"),
		"  ],",
		"  \"cases\": [",
		("    {\"name\":\"pure Tetra component app\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"text focus input click focuses TextBox\",\"kind\":\"positi" +
			"ve\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input Tab changes focus\",\"kind\":\"positive\"" +
			",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input keyboard routes only focused component" +
			"\",\"kind\":\"positive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input text insertion\",\"kind\":\"positive\",\"" +
			"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input caret movement\",\"kind\":\"positive\",\"" +
			"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input backspace delete\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input resize preserves focus\",\"kind\":\"posit" +
			"ive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"text focus input rendered frame update\",\"kind\":\"positi" +
			"ve\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host-provided pointer event dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host event buffer poll_event\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"pre/post event frame sequence\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"component hierarchy dispatch\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"component text input scalar dispatch\",\"kind\":\"positive" +
			"\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"host text payload buffer\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component focus dispatch\",\"kind\":\"positive\",\"ran\":t" +
			"rue,\"pass\":true},"),
		("    {\"name\":\"component accessibility metadata\",\"kind\":\"positive\"," +
			"\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"no legacy UI sidecar artifacts\",\"kind\":\"positive\",\"r" +
			"an\":true,\"pass\":true},"),
		("    {\"name\":\"state transition\",\"kind\":\"positive\",\"ran\":true,\"pa" +
			"ss\":true},"),
		("    {\"name\":\"reject legacy UI evidence\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"legacy UI evidence rejected\"},"),
		("    {\"name\":\"headless event dispatch\",\"kind\":\"positive\",\"ran\":tr" +
			"ue,\"pass\":true},"),
		("    {\"name\":\"headless framebuffer checksum\",\"kind\":\"positive\",\"ra" +
			"n\":true,\"pass\":true},"),
		("    {\"name\":\"headless actual runner trace\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true}"),
		"  ]",
		"}",
		"",
	}, "\n"))
}

// ---- report_toolkit_test.go ----

func TestValidateReportAcceptsHeadlessMinimalToolkitSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceToolkitReuseReport(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceProductionToolkitReport(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceProductionToolkitRejectsMissingReleaseWidget(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["widget_set"] = []any{
			"Text",
			"Label",
			"StatusText",
			"Button",
			"TextBox",
			"Row",
			"Column",
			"Panel",
			"Stack",
			"Scroll",
			"Spacer",
		}
		widgets := toolkit["widgets"].([]any)
		filtered := make([]any, 0, len(widgets))
		for _, rawWidget := range widgets {
			widget := rawWidget.(map[string]any)
			if widget["kind"] == "Checkbox" {
				continue
			}
			filtered = append(filtered, widget)
		}
		toolkit["widgets"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "Checkbox") {
		t.Fatalf("ValidateReport error = %v, want missing Checkbox rejection", err)
	}
}
func TestValidateSurfaceProductionToolkitRejectsSingleExampleClaim(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["example_count"] = 1
		toolkit["sources"] = []any{"examples/surface/release/surface_release_form.tetra"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected production toolkit single-example claim to fail")
	}
	for _, want := range []string{"production toolkit", "example_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsSingleExampleReuseClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["example_count"] = 1
		toolkit["sources"] = []any{"examples/surface/toolkit/surface_toolkit_settings.tetra"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report with one example to fail")
	}
	for _, want := range []string{"toolkit", "example_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingWidgetsModule(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["module"] = "examples.local.widgets"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report with wrong module to fail")
	}
	if !strings.Contains(err.Error(), "module") {
		t.Fatalf("error = %v, want module diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsProductionClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["production_claim"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse production claim to fail")
	}
	for _, want := range []string{"toolkit", "production"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsDemoLocalWidgetStructs(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["demo_specific_widget_structs"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse demo-local widget structs to fail")
	}
	if !strings.Contains(err.Error(), "demo_specific_widget_structs") {
		t.Fatalf("error = %v, want demo_specific_widget_structs diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsManualTreeBookkeeping(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["manual_bookkeeping"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse manual bookkeeping to fail")
	}
	if !strings.Contains(err.Error(), "manual_bookkeeping") {
		t.Fatalf("error = %v, want manual_bookkeeping diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsMissingSecondTextBoxRouting(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		var filtered []any
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "EmailTextBox" {
				continue
			}
			filtered = append(filtered, event)
		}
		report["events"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing second TextBox routing to fail")
	}
	for _, want := range []string{"EmailTextBox", "routing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsUnfocusedTextBoxMutation(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "EmailTextBox" && event["kind"] == "text_input" {
				after := event["after_state"].(map[string]any)
				after["NameTextBox.buffer"] = "AdaX"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse unfocused TextBox mutation to fail")
	}
	for _, want := range []string{"unfocused", "TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingStatusUpdate(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		var filtered []any
		for _, rawTransition := range report["state_transitions"].([]any) {
			transition := rawTransition.(map[string]any)
			if transition["component"] == "StatusText" {
				continue
			}
			filtered = append(filtered, transition)
		}
		report["state_transitions"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing StatusText update to fail")
	}
	if !strings.Contains(err.Error(), "StatusText") {
		t.Fatalf("error = %v, want StatusText diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsMissingResizeRelayout(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		var filtered []any
		for _, rawEvent := range report["events"].([]any) {
			event := rawEvent.(map[string]any)
			if event["kind"] == "resize" {
				continue
			}
			filtered = append(filtered, event)
		}
		report["events"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing resize relayout to fail")
	}
	if !strings.Contains(err.Error(), "resize") {
		t.Fatalf("error = %v, want resize diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsUnchangedFrameChecksum(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		last := frames[len(frames)-1].(map[string]any)
		last["checksum"] = first["checksum"]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse unchanged final frame to fail")
	}
	if !strings.Contains(err.Error(), "frame") {
		t.Fatalf("error = %v, want frame diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsDOMOrUserJSClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_dom_ui"] = false
		toolkit["no_user_js"] = false
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse DOM/user JS claim to fail")
	}
	for _, want := range []string{"no_dom_ui", "no_user_js"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsNodeOnlyBrowserClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		report["target"] = "wasm32-web"
		report["host"] = "node"
		report["host_evidence"] = map[string]any{
			"level":                        "wasm32-web-compiler-owned-loader",
			"backend":                      "node-surface-host",
			"framebuffer":                  true,
			"real_window":                  false,
			"native_input":                 false,
			"user_facing_platform_widgets": false,
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse browser evidence downgraded to Node-only to fail")
	}
	for _, want := range []string{"browser", "Node"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingArtifactScan(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "artifact_scan")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report without artifact scan to fail")
	}
	if !strings.Contains(err.Error(), "artifact_scan") {
		t.Fatalf("error = %v, want artifact_scan diagnostic", err)
	}
}
func TestValidateReportRejectsMinimalToolkitMissingToolkitBlock(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "toolkit")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit report without toolkit block to fail")
	}
	if !strings.Contains(err.Error(), "toolkit") {
		t.Fatalf("error = %v, want toolkit diagnostic", err)
	}
}
func TestValidateReportRejectsMinimalToolkitProductionClaim(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["production_claim"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit production claim to fail")
	}
	for _, want := range []string{"toolkit", "production"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitMissingWidgetEvidence(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		widgets := toolkit["widgets"].([]any)
		toolkit["widgets"] = widgets[:len(widgets)-1]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit report without StatusText evidence to fail")
	}
	for _, want := range []string{"toolkit", "StatusText"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitButtonActionWithoutFocusedDispatch(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "SubmitButton" && event["kind"] == "key_down" {
				event["dispatch_path"] = []any{"ToolkitFormApp", "Panel", "Column", "SubmitButton"}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit bad Submit dispatch path to fail")
	}
	for _, want := range []string{"SubmitButton", "dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitTextMutationWhileButtonFocused(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "ResetButton" && event["kind"] == "text_input" {
				after := event["after_state"].(map[string]any)
				after["TextBox.buffer"] = "BAD"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit TextBox mutation while Button focused to fail")
	}
	for _, want := range []string{"TextBox", "Button focused"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func validHeadlessMinimalToolkitSurfaceReportJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(
		validHeadlessComponentTreeSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode base component tree report: %v", err)
	}
	report["source"] = "examples/surface/toolkit/surface_toolkit_form.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_toolkit_form.tetra -o " +
				"/tmp/surface-artifacts/surface-toolkit-form"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-toolkit-form",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":      "surface headless runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-toolkit-form",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   81234,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   22000,
		},
	}
	report["components"] = []any{
		componentMap(
			"ToolkitFormApp",
			"examples.surface.toolkit.surface_toolkit_form.ToolkitFormApp",
			"",
			RectReport{X: 0, Y: 0, W: 400, H: 240},
			map[string]string{
				"focused_id":         "7",
				"submit_count":       "1",
				"reset_count":        "1",
				"status_code":        "2",
				"width":              "400",
				"height":             "240",
				"accessibility_role": "none",
			},
		),
		componentMap(
			"Panel",
			"lib.core.widgets.Panel",
			"ToolkitFormApp",
			RectReport{X: 0, Y: 0, W: 400, H: 240},
			map[string]string{"padding": "12", "accessibility_role": "none"},
		),
		componentMap(
			"Column",
			"lib.core.widgets.Column",
			"Panel",
			RectReport{X: 12, Y: 12, W: 376, H: 216},
			map[string]string{"child_count": "4", "accessibility_role": "none"},
		),
		componentMap(
			"NameLabel",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 20, W: 360, H: 24},
			map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"},
		),
		componentMap(
			"TextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 20, Y: 52, W: 360, H: 44},
			map[string]string{
				"focused":            "false",
				"buffer":             "",
				"text_len":           "0",
				"caret":              "0",
				"backspace_count":    "1",
				"delete_count":       "1",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"ButtonRow",
			"lib.core.widgets.Row",
			"Column",
			RectReport{X: 20, Y: 108, W: 360, H: 44},
			map[string]string{"child_count": "2", "accessibility_role": "none"},
		),
		componentMap(
			"SubmitButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 20, Y: 108, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "submit",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"ResetButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 164, Y: 108, W: 132, H: 44},
			map[string]string{
				"focused":            "true",
				"press_count":        "1",
				"action":             "reset",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"StatusText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 160, W: 360, H: 24},
			map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"},
		),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "minimal-toolkit-widget-tree",
		"root_id":       0,
		"node_count":    9,
		"focused_id":    7,
		"nodes": []any{
			treeNodeMap(
				0,
				"ToolkitFormApp",
				"root",
				-1,
				0,
				1,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 400, H: 240},
			),
			treeNodeMap(
				1,
				"Panel",
				"panel",
				0,
				0,
				2,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 400, H: 240},
			),
			treeNodeMap(
				2,
				"Column",
				"column",
				1,
				0,
				3,
				4,
				false,
				RectReport{X: 12, Y: 12, W: 376, H: 216},
			),
			treeNodeMap(
				3,
				"NameLabel",
				"text",
				2,
				0,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 20, W: 360, H: 24},
			),
			treeNodeMap(
				4,
				"TextBox",
				"textbox",
				2,
				1,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 52, W: 360, H: 44},
			),
			treeNodeMap(
				5,
				"ButtonRow",
				"row",
				2,
				2,
				6,
				2,
				false,
				RectReport{X: 20, Y: 108, W: 360, H: 44},
			),
			treeNodeMap(
				6,
				"SubmitButton",
				"button",
				5,
				0,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 108, W: 132, H: 44},
			),
			treeNodeMap(
				7,
				"ResetButton",
				"button",
				5,
				1,
				-1,
				0,
				true,
				RectReport{X: 164, Y: 108, W: 132, H: 44},
			),
			treeNodeMap(
				8,
				"StatusText",
				"text",
				2,
				3,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 160, W: 360, H: 24},
			),
		},
		"layout_passes": []any{
			map[string]any{
				"component_id": 4,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 20, Y: 52, W: 280, H: 44}),
				"measured":     map[string]any{"w": 280, "h": 44},
			},
			map[string]any{
				"component_id": 4,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 20, Y: 52, W: 360, H: 44}),
				"measured":     map[string]any{"w": 360, "h": 44},
			},
			map[string]any{
				"component_id": 8,
				"pass":         "status-update",
				"bounds":       rectMap(RectReport{X: 20, Y: 160, W: 360, H: 24}),
				"measured":     map[string]any{"w": 360, "h": 24},
			},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8},
		"focus_order": []any{4, 6, 7},
		"dispatch_paths": []any{
			map[string]any{
				"event":     "click",
				"target_id": 4,
				"x":         40,
				"y":         72,
				"path":      []any{0, 1, 2, 4},
			},
			map[string]any{
				"event":     "click",
				"target_id": 6,
				"x":         40,
				"y":         124,
				"path":      []any{0, 1, 2, 5, 6},
			},
			map[string]any{
				"event":     "click",
				"target_id": 7,
				"x":         180,
				"y":         124,
				"path":      []any{0, 1, 2, 5, 7},
			},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface/toolkit/surface_toolkit_form.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          9,
			"capacity":            16,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{
				"helper":         "widgets.panel_content_rect",
				"target":         "Panel",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.row_layout",
				"target":         "ButtonRow",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "resize",
				"changed_bounds": true,
			},
		},
		"focus_helpers": []any{
			map[string]any{
				"helper": "tree_focus_next",
				"before": "TextBox",
				"after":  "SubmitButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "SubmitButton",
				"after":  "ResetButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "ResetButton",
				"after":  "TextBox",
			},
		},
		"hit_tests": []any{
			map[string]any{
				"helper": "widgets.hit_test",
				"x":      40,
				"y":      72,
				"target": "TextBox",
				"path":   []any{0, 1, 2, 4},
			},
			map[string]any{
				"helper": "widgets.hit_test",
				"x":      180,
				"y":      124,
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 5, 7},
			},
		},
		"dispatch_paths": []any{
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "TextBox",
				"path":   []any{0, 1, 2, 4},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "SubmitButton",
				"path":   []any{0, 1, 2, 5, 6},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 5, 7},
			},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":                       "tetra.surface.toolkit.v1",
		"toolkit_level":                "minimal-widgets-v1",
		"source":                       "examples/surface/toolkit/surface_toolkit_form.tetra",
		"module":                       "lib.core.widgets",
		"experimental":                 true,
		"production_claim":             false,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("NameLabel", "Text", 3, "label", true),
			toolkitWidgetMap("TextBox", "TextBox", 4, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 5, "", true),
			toolkitWidgetMap("SubmitButton", "Button", 6, "submit", true),
			toolkitWidgetMap("ResetButton", "Button", 7, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 8, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
		},
	}
	report["events"] = []any{
		eventMap(
			1,
			"mouse_up",
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			40,
			72,
			0,
			320,
			200,
			map[string]string{"ToolkitFormApp.focused_id": "-1", "TextBox.focused": "false"},
			map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.focused": "true"},
		),
		textEventMap(
			2,
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			2,
			"4f4b",
			320,
			200,
			map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"},
			map[string]string{
				"TextBox.buffer":   "OK",
				"TextBox.caret":    "2",
				"TextBox.text_len": "2",
			},
		),
		keyEventMap(
			3,
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			37,
			320,
			200,
			map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"},
			map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
		),
		keyEventMap(
			4,
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			8,
			320,
			200,
			map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
			map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
		),
		keyEventMap(
			5,
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			46,
			320,
			200,
			map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
			map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
		),
		textEventMap(
			6,
			"TextBox",
			[]any{"ToolkitFormApp", "Panel", "Column", "TextBox"},
			1,
			"5a",
			320,
			200,
			map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"},
			map[string]string{"TextBox.buffer": "Z", "TextBox.caret": "1", "TextBox.text_len": "1"},
		),
		keyEventMap(
			7,
			"ToolkitFormApp",
			[]any{"ToolkitFormApp"},
			9,
			320,
			200,
			map[string]string{"ToolkitFormApp.focused_id": "4"},
			map[string]string{"ToolkitFormApp.focused_id": "6"},
		),
		keyEventMap(
			8,
			"SubmitButton",
			[]any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "SubmitButton"},
			32,
			320,
			200,
			map[string]string{
				"ToolkitFormApp.focused_id":   "6",
				"ToolkitFormApp.submit_count": "0",
				"StatusText.status_code":      "0",
				"TextBox.buffer":              "Z",
			},
			map[string]string{
				"ToolkitFormApp.focused_id":   "6",
				"ToolkitFormApp.submit_count": "1",
				"StatusText.status_code":      "1",
				"TextBox.buffer":              "Z",
			},
		),
		keyEventMap(
			9,
			"ToolkitFormApp",
			[]any{"ToolkitFormApp"},
			9,
			320,
			200,
			map[string]string{"ToolkitFormApp.focused_id": "6"},
			map[string]string{"ToolkitFormApp.focused_id": "7"},
		),
		textEventMap(
			10,
			"ResetButton",
			[]any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"},
			1,
			"58",
			320,
			200,
			map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"},
			map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"},
		),
		keyEventMap(
			11,
			"ResetButton",
			[]any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"},
			13,
			320,
			200,
			map[string]string{
				"ToolkitFormApp.focused_id":  "7",
				"ToolkitFormApp.reset_count": "0",
				"StatusText.status_code":     "1",
				"TextBox.buffer":             "Z",
			},
			map[string]string{
				"ToolkitFormApp.focused_id":  "7",
				"ToolkitFormApp.reset_count": "1",
				"StatusText.status_code":     "2",
				"TextBox.buffer":             "",
			},
		),
		keyEventMap(
			12,
			"ToolkitFormApp",
			[]any{"ToolkitFormApp"},
			9,
			320,
			200,
			map[string]string{"ToolkitFormApp.focused_id": "7"},
			map[string]string{"ToolkitFormApp.focused_id": "4"},
		),
		resizeEventMap(
			13,
			"ToolkitFormApp",
			[]any{"ToolkitFormApp"},
			400,
			240,
			map[string]string{
				"ToolkitFormApp.focused_id": "4",
				"TextBox.bounds.w":          "280",
				"TextBox.buffer":            "",
			},
			map[string]string{
				"ToolkitFormApp.focused_id": "4",
				"TextBox.bounds.w":          "360",
				"TextBox.buffer":            "",
			},
		),
	}
	report["frames"] = []any{
		map[string]any{
			"order":     1,
			"width":     320,
			"height":    200,
			"stride":    1280,
			"checksum":  "1111111111111111111111111111111111111111111111111111111111111111",
			"presented": true,
		},
		map[string]any{
			"order":     2,
			"width":     320,
			"height":    200,
			"stride":    1280,
			"checksum":  "2222222222222222222222222222222222222222222222222222222222222222",
			"presented": true,
		},
		map[string]any{
			"order":     3,
			"width":     320,
			"height":    200,
			"stride":    1280,
			"checksum":  "3333333333333333333333333333333333333333333333333333333333333333",
			"presented": true,
		},
		map[string]any{
			"order":     4,
			"width":     400,
			"height":    240,
			"stride":    1600,
			"checksum":  "4444444444444444444444444444444444444444444444444444444444444444",
			"presented": true,
		},
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "ToolkitFormApp",
			"field":     "focused_id",
			"before":    "-1",
			"after":     "4",
			"cause":     "mouse_up",
		},
		map[string]any{
			"order":     2,
			"component": "TextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "OK",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     3,
			"component": "TextBox",
			"field":     "caret",
			"before":    "2",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     4,
			"component": "TextBox",
			"field":     "buffer",
			"before":    "OK",
			"after":     "K",
			"cause":     "backspace",
		},
		map[string]any{
			"order":     5,
			"component": "TextBox",
			"field":     "buffer",
			"before":    "K",
			"after":     "",
			"cause":     "delete",
		},
		map[string]any{
			"order":     6,
			"component": "TextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "Z",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     7,
			"component": "ToolkitFormApp",
			"field":     "focused_id",
			"before":    "4",
			"after":     "6",
			"cause":     "tab",
		},
		map[string]any{
			"order":     8,
			"component": "ToolkitFormApp",
			"field":     "submit_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     9,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "0",
			"after":     "1",
			"cause":     "submit",
		},
		map[string]any{
			"order":     10,
			"component": "ToolkitFormApp",
			"field":     "focused_id",
			"before":    "6",
			"after":     "7",
			"cause":     "tab",
		},
		map[string]any{
			"order":     11,
			"component": "TextBox",
			"field":     "buffer",
			"before":    "Z",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     12,
			"component": "ToolkitFormApp",
			"field":     "reset_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     13,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "1",
			"after":     "2",
			"cause":     "reset",
		},
		map[string]any{
			"order":     14,
			"component": "ToolkitFormApp",
			"field":     "focused_id",
			"before":    "7",
			"after":     "4",
			"cause":     "tab",
		},
		map[string]any{
			"order":     15,
			"component": "ToolkitFormApp",
			"field":     "TextBox.bounds.w",
			"before":    "280",
			"after":     "360",
			"cause":     "resize",
		},
	}
	report["cases"] = append(
		report["cases"].([]any),
		map[string]any{
			"name": "minimal toolkit reusable widgets",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit Text widget evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit Button widget evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit TextBox widget evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit Row Column Panel layout",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit tree api reuse",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit TextBox focus input editing",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit Submit action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit Reset action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit status text update",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit resize relayout",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "minimal toolkit rendered frame update",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal minimal toolkit report: %v", err)
	}
	return raw
}
func validHeadlessToolkitReuseSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(
		validHeadlessMinimalToolkitSurfaceReportJSON(t, nil),
		&report,
	); err != nil {
		t.Fatalf("decode base toolkit report: %v", err)
	}
	report["source"] = "examples/surface/toolkit/surface_toolkit_settings.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_toolkit_settings.tetra -o " +
				"/tmp/surface-artifacts/surface-toolkit-settings"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-toolkit-settings",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":      "surface headless runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-toolkit-settings",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   81234,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   22000,
		},
	}
	report["components"] = []any{
		componentMap(
			"ToolkitSettingsApp",
			"examples.surface.toolkit.surface_toolkit_settings.ToolkitSettingsApp",
			"",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			map[string]string{
				"focused_id":         "4",
				"save_count":         "1",
				"reset_count":        "1",
				"status_code":        "2",
				"width":              "480",
				"height":             "320",
				"accessibility_role": "none",
			},
		),
		componentMap(
			"Panel",
			"lib.core.widgets.Panel",
			"ToolkitSettingsApp",
			RectReport{X: 0, Y: 0, W: 480, H: 320},
			map[string]string{"padding": "12", "accessibility_role": "none"},
		),
		componentMap(
			"Column",
			"lib.core.widgets.Column",
			"Panel",
			RectReport{X: 12, Y: 12, W: 456, H: 296},
			map[string]string{"child_count": "6", "accessibility_role": "none"},
		),
		componentMap(
			"TitleText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 20, W: 440, H: 24},
			map[string]string{"role": "label", "text_len": "8", "accessibility_role": "label"},
		),
		componentMap(
			"NameTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 20, Y: 52, W: 440, H: 44},
			map[string]string{
				"focused":            "true",
				"buffer":             "",
				"text_len":           "0",
				"caret":              "0",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"NameLabel",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 104, W: 440, H: 24},
			map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"},
		),
		componentMap(
			"EmailTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 20, Y: 136, W: 440, H: 44},
			map[string]string{
				"focused":            "false",
				"buffer":             "",
				"text_len":           "0",
				"caret":              "0",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"ButtonRow",
			"lib.core.widgets.Row",
			"Column",
			RectReport{X: 20, Y: 192, W: 440, H: 44},
			map[string]string{"child_count": "2", "accessibility_role": "none"},
		),
		componentMap(
			"SaveButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 20, Y: 192, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "save",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"ResetButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 164, Y: 192, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "reset",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"StatusText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 20, Y: 248, W: 440, H: 24},
			map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"},
		),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "toolkit-reuse-widget-tree",
		"root_id":       0,
		"node_count":    11,
		"focused_id":    4,
		"nodes": []any{
			treeNodeMap(
				0,
				"ToolkitSettingsApp",
				"root",
				-1,
				0,
				1,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 480, H: 320},
			),
			treeNodeMap(
				1,
				"Panel",
				"panel",
				0,
				0,
				2,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 480, H: 320},
			),
			treeNodeMap(
				2,
				"Column",
				"column",
				1,
				0,
				3,
				6,
				false,
				RectReport{X: 12, Y: 12, W: 456, H: 296},
			),
			treeNodeMap(
				3,
				"TitleText",
				"text",
				2,
				0,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 20, W: 440, H: 24},
			),
			treeNodeMap(
				4,
				"NameTextBox",
				"textbox",
				2,
				1,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 52, W: 440, H: 44},
			),
			treeNodeMap(
				5,
				"NameLabel",
				"text",
				2,
				2,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 104, W: 440, H: 24},
			),
			treeNodeMap(
				6,
				"EmailTextBox",
				"textbox",
				2,
				3,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 136, W: 440, H: 44},
			),
			treeNodeMap(
				7,
				"ButtonRow",
				"row",
				2,
				4,
				8,
				2,
				false,
				RectReport{X: 20, Y: 192, W: 440, H: 44},
			),
			treeNodeMap(
				8,
				"SaveButton",
				"button",
				7,
				0,
				-1,
				0,
				true,
				RectReport{X: 20, Y: 192, W: 132, H: 44},
			),
			treeNodeMap(
				9,
				"ResetButton",
				"button",
				7,
				1,
				-1,
				0,
				true,
				RectReport{X: 164, Y: 192, W: 132, H: 44},
			),
			treeNodeMap(
				10,
				"StatusText",
				"text",
				2,
				5,
				-1,
				0,
				false,
				RectReport{X: 20, Y: 248, W: 440, H: 24},
			),
		},
		"layout_passes": []any{
			map[string]any{
				"component_id": 4,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 20, Y: 52, W: 280, H: 44}),
				"measured":     map[string]any{"w": 280, "h": 44},
			},
			map[string]any{
				"component_id": 6,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 20, Y: 136, W: 280, H: 44}),
				"measured":     map[string]any{"w": 280, "h": 44},
			},
			map[string]any{
				"component_id": 4,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 20, Y: 52, W: 440, H: 44}),
				"measured":     map[string]any{"w": 440, "h": 44},
			},
			map[string]any{
				"component_id": 6,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 20, Y: 136, W: 440, H: 44}),
				"measured":     map[string]any{"w": 440, "h": 44},
			},
			map[string]any{
				"component_id": 10,
				"pass":         "status-update",
				"bounds":       rectMap(RectReport{X: 20, Y: 248, W: 440, H: 24}),
				"measured":     map[string]any{"w": 440, "h": 24},
			},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"focus_order": []any{4, 6, 8, 9},
		"dispatch_paths": []any{
			map[string]any{
				"event":     "click",
				"target_id": 4,
				"x":         40,
				"y":         72,
				"path":      []any{0, 1, 2, 4},
			},
			map[string]any{
				"event":     "click",
				"target_id": 6,
				"x":         40,
				"y":         156,
				"path":      []any{0, 1, 2, 6},
			},
			map[string]any{
				"event":     "key",
				"target_id": 8,
				"x":         40,
				"y":         208,
				"path":      []any{0, 1, 2, 7, 8},
			},
			map[string]any{
				"event":     "key",
				"target_id": 9,
				"x":         180,
				"y":         208,
				"path":      []any{0, 1, 2, 7, 9},
			},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface/toolkit/surface_toolkit_settings.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          11,
			"capacity":            20,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{
				"helper":         "widgets.panel_content_rect",
				"target":         "Panel",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.row_layout",
				"target":         "ButtonRow",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "resize",
				"changed_bounds": true,
			},
		},
		"focus_helpers": []any{
			map[string]any{
				"helper": "tree_focus_next",
				"before": "NameTextBox",
				"after":  "EmailTextBox",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "EmailTextBox",
				"after":  "SaveButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "SaveButton",
				"after":  "ResetButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "ResetButton",
				"after":  "NameTextBox",
			},
		},
		"hit_tests": []any{
			map[string]any{
				"helper": "widgets.hit_test",
				"x":      40,
				"y":      72,
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 4},
			},
			map[string]any{
				"helper": "widgets.hit_test",
				"x":      40,
				"y":      156,
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 6},
			},
			map[string]any{
				"helper": "widgets.hit_test",
				"x":      180,
				"y":      208,
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 7, 9},
			},
		},
		"dispatch_paths": []any{
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 4},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 6},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "SaveButton",
				"path":   []any{0, 1, 2, 7, 8},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 7, 9},
			},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":        "tetra.surface.toolkit.v1",
		"toolkit_level": "toolkit-reuse-v1",
		"reuse_level":   "multi-form-widget-reuse-v1",
		"source":        "examples/surface/toolkit/surface_toolkit_settings.tetra",
		"sources": []any{
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
		},
		"module":                       "lib.core.widgets",
		"experimental":                 true,
		"production_claim":             false,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"example_count":                2,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("TitleText", "Text", 3, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 4, "", true),
			toolkitWidgetMap("NameLabel", "Text", 5, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 6, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 7, "", true),
			toolkitWidgetMap("SaveButton", "Button", 8, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 9, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 10, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:hit_test",
			"lib/core/widgets/widgets.tetra:textbox_text_input",
			"lib/core/widgets/widgets.tetra:button_key_event",
		},
	}
	report["events"] = []any{
		eventMap(
			1,
			"mouse_up",
			"NameTextBox",
			[]any{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"},
			40,
			72,
			0,
			320,
			240,
			map[string]string{
				"ToolkitSettingsApp.focused_id": "-1",
				"NameTextBox.focused":           "false",
			},
			map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.focused": "true"},
		),
		textEventMap(
			2,
			"NameTextBox",
			[]any{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"},
			3,
			"416461",
			320,
			240,
			map[string]string{
				"NameTextBox.buffer":  "",
				"NameTextBox.caret":   "0",
				"EmailTextBox.buffer": "",
			},
			map[string]string{
				"NameTextBox.buffer":  "Ada",
				"NameTextBox.caret":   "3",
				"EmailTextBox.buffer": "",
			},
		),
		keyEventMap(
			3,
			"ToolkitSettingsApp",
			[]any{"ToolkitSettingsApp"},
			9,
			320,
			240,
			map[string]string{"ToolkitSettingsApp.focused_id": "4"},
			map[string]string{"ToolkitSettingsApp.focused_id": "6"},
		),
		textEventMap(
			4,
			"EmailTextBox",
			[]any{"ToolkitSettingsApp", "Panel", "Column", "EmailTextBox"},
			5,
			"7465747261",
			320,
			240,
			map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"},
			map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"},
		),
		keyEventMap(
			5,
			"ToolkitSettingsApp",
			[]any{"ToolkitSettingsApp"},
			9,
			320,
			240,
			map[string]string{"ToolkitSettingsApp.focused_id": "6"},
			map[string]string{"ToolkitSettingsApp.focused_id": "8"},
		),
		keyEventMap(
			6,
			"SaveButton",
			[]any{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"},
			32,
			320,
			240,
			map[string]string{
				"ToolkitSettingsApp.focused_id": "8",
				"ToolkitSettingsApp.save_count": "0",
				"StatusText.status_code":        "0",
			},
			map[string]string{
				"ToolkitSettingsApp.focused_id": "8",
				"ToolkitSettingsApp.save_count": "1",
				"StatusText.status_code":        "1",
			},
		),
		keyEventMap(
			7,
			"ToolkitSettingsApp",
			[]any{"ToolkitSettingsApp"},
			9,
			320,
			240,
			map[string]string{"ToolkitSettingsApp.focused_id": "8"},
			map[string]string{"ToolkitSettingsApp.focused_id": "9"},
		),
		keyEventMap(
			8,
			"ResetButton",
			[]any{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"},
			13,
			320,
			240,
			map[string]string{
				"ToolkitSettingsApp.focused_id":  "9",
				"ToolkitSettingsApp.reset_count": "0",
				"StatusText.status_code":         "1",
				"NameTextBox.buffer":             "Ada",
				"EmailTextBox.buffer":            "tetra",
			},
			map[string]string{
				"ToolkitSettingsApp.focused_id":  "9",
				"ToolkitSettingsApp.reset_count": "1",
				"StatusText.status_code":         "2",
				"NameTextBox.buffer":             "",
				"EmailTextBox.buffer":            "",
			},
		),
		keyEventMap(
			9,
			"ToolkitSettingsApp",
			[]any{"ToolkitSettingsApp"},
			9,
			320,
			240,
			map[string]string{"ToolkitSettingsApp.focused_id": "9"},
			map[string]string{"ToolkitSettingsApp.focused_id": "4"},
		),
		resizeEventMap(
			10,
			"ToolkitSettingsApp",
			[]any{"ToolkitSettingsApp"},
			480,
			320,
			map[string]string{
				"ToolkitSettingsApp.focused_id": "4",
				"NameTextBox.bounds.w":          "280",
				"EmailTextBox.bounds.w":         "280",
			},
			map[string]string{
				"ToolkitSettingsApp.focused_id": "4",
				"NameTextBox.bounds.w":          "440",
				"EmailTextBox.bounds.w":         "440",
			},
		),
	}
	report["frames"] = []any{
		map[string]any{
			"order":     1,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"presented": true,
		},
		map[string]any{
			"order":     2,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"presented": true,
		},
		map[string]any{
			"order":     3,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"presented": true,
		},
		map[string]any{
			"order":     4,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"presented": true,
		},
		map[string]any{
			"order":     5,
			"width":     480,
			"height":    320,
			"stride":    1920,
			"checksum":  "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"presented": true,
		},
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "ToolkitSettingsApp",
			"field":     "focused_id",
			"before":    "-1",
			"after":     "4",
			"cause":     "mouse_up",
		},
		map[string]any{
			"order":     2,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "Ada",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     3,
			"component": "ToolkitSettingsApp",
			"field":     "focused_id",
			"before":    "4",
			"after":     "6",
			"cause":     "tab",
		},
		map[string]any{
			"order":     4,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "tetra",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     5,
			"component": "ToolkitSettingsApp",
			"field":     "focused_id",
			"before":    "6",
			"after":     "8",
			"cause":     "tab",
		},
		map[string]any{
			"order":     6,
			"component": "ToolkitSettingsApp",
			"field":     "save_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     7,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "0",
			"after":     "1",
			"cause":     "save",
		},
		map[string]any{
			"order":     8,
			"component": "ToolkitSettingsApp",
			"field":     "focused_id",
			"before":    "8",
			"after":     "9",
			"cause":     "tab",
		},
		map[string]any{
			"order":     9,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "Ada",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     10,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "tetra",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     11,
			"component": "ToolkitSettingsApp",
			"field":     "reset_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     12,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "1",
			"after":     "2",
			"cause":     "reset",
		},
		map[string]any{
			"order":     13,
			"component": "ToolkitSettingsApp",
			"field":     "focused_id",
			"before":    "9",
			"after":     "4",
			"cause":     "tab",
		},
		map[string]any{
			"order":     14,
			"component": "ToolkitSettingsApp",
			"field":     "NameTextBox.bounds.w",
			"before":    "280",
			"after":     "440",
			"cause":     "resize",
		},
		map[string]any{
			"order":     15,
			"component": "ToolkitSettingsApp",
			"field":     "EmailTextBox.bounds.w",
			"before":    "280",
			"after":     "440",
			"cause":     "resize",
		},
	}
	report["cases"] = append(
		report["cases"].([]any),
		map[string]any{
			"name": "toolkit reuse second example evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse widgets module evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse multi TextBox routing",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse focused TextBox only mutates",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse Save action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse Reset action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse StatusText updates",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse resize relayout",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse changed frame checksums",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "toolkit reuse no demo-local widget structs",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal toolkit reuse report: %v", err)
	}
	return raw
}

func validHeadlessProductionToolkitSurfaceReportJSON(
	t *testing.T,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessToolkitReuseSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base toolkit reuse report: %v", err)
	}
	report["source"] = "examples/surface/release/surface_release_form.tetra"
	report["processes"] = []any{
		map[string]any{
			"name": "tetra build",
			"kind": "build",
			"path": ("tetra build --target linux-x64 " +
				"examples/surface/release/surface_release_form.tetra -o " +
				"/tmp/surface-artifacts/surface-release-form"),
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
		map[string]any{
			"name":               "surface component app",
			"kind":               "app",
			"path":               "/tmp/surface-artifacts/surface-release-form",
			"ran":                true,
			"pass":               true,
			"exit_code":          1,
			"expected_exit_code": 1,
		},
		map[string]any{
			"name":      "surface headless runtime",
			"kind":      "runtime",
			"path":      "tools/cmd/surface-runtime-smoke",
			"ran":       true,
			"pass":      true,
			"exit_code": 0,
		},
	}
	report["artifacts"] = []any{
		map[string]any{
			"kind":   "component-app",
			"path":   "/tmp/surface-artifacts/surface-release-form",
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   98234,
		},
		map[string]any{
			"kind":   "runner-trace",
			"path":   "/tmp/surface-artifacts/surface-runner-trace.json",
			"sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"size":   32000,
		},
	}
	report["components"] = []any{
		componentMap(
			"SurfaceReleaseFormApp",
			"examples.surface.release.surface_release_form.SurfaceReleaseFormApp",
			"",
			RectReport{X: 0, Y: 0, W: 560, H: 420},
			map[string]string{
				"focused_id":         "7",
				"save_count":         "1",
				"reset_count":        "1",
				"status_code":        "2",
				"width":              "560",
				"height":             "420",
				"accessibility_role": "none",
			},
		),
		componentMap(
			"Panel",
			"lib.core.widgets.Panel",
			"SurfaceReleaseFormApp",
			RectReport{X: 0, Y: 0, W: 560, H: 420},
			map[string]string{"padding": "16", "accessibility_role": "none"},
		),
		componentMap(
			"Stack",
			"lib.core.widgets.Stack",
			"Panel",
			RectReport{X: 16, Y: 16, W: 528, H: 396},
			map[string]string{"child_count": "1", "accessibility_role": "none"},
		),
		componentMap(
			"Column",
			"lib.core.widgets.Column",
			"Stack",
			RectReport{X: 24, Y: 24, W: 512, H: 388},
			map[string]string{"child_count": "9", "accessibility_role": "none"},
		),
		componentMap(
			"TitleText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 32, Y: 32, W: 496, H: 28},
			map[string]string{"role": "title", "text_len": "18", "accessibility_role": "label"},
		),
		componentMap(
			"DescriptionText",
			"lib.core.widgets.Text",
			"Column",
			RectReport{X: 32, Y: 68, W: 496, H: 28},
			map[string]string{
				"role":               "description",
				"text_len":           "24",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"NameLabel",
			"lib.core.widgets.Label",
			"Column",
			RectReport{X: 32, Y: 104, W: 496, H: 24},
			map[string]string{
				"role":               "label",
				"text_len":           "4",
				"labelled_for":       "7",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"NameTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 32, Y: 132, W: 496, H: 44},
			map[string]string{
				"focused":            "true",
				"buffer":             "Ada",
				"text_len":           "3",
				"caret":              "3",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"EmailLabel",
			"lib.core.widgets.Label",
			"Column",
			RectReport{X: 32, Y: 184, W: 496, H: 24},
			map[string]string{
				"role":               "label",
				"text_len":           "5",
				"labelled_for":       "9",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"EmailTextBox",
			"lib.core.widgets.TextBox",
			"Column",
			RectReport{X: 32, Y: 212, W: 496, H: 44},
			map[string]string{
				"focused":            "false",
				"buffer":             "tetra",
				"text_len":           "5",
				"caret":              "5",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"SubscribeCheckbox",
			"lib.core.widgets.Checkbox",
			"Column",
			RectReport{X: 32, Y: 264, W: 496, H: 32},
			map[string]string{
				"focused":            "false",
				"checked":            "true",
				"toggle_count":       "1",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"TermsScroll",
			"lib.core.widgets.Scroll",
			"Column",
			RectReport{X: 32, Y: 304, W: 496, H: 48},
			map[string]string{"offset_y": "16", "content_h": "120", "accessibility_role": "none"},
		),
		componentMap(
			"TermsText",
			"lib.core.widgets.Text",
			"TermsScroll",
			RectReport{X: 36, Y: 308, W: 488, H: 24},
			map[string]string{
				"role":               "description",
				"text_len":           "48",
				"accessibility_role": "label",
			},
		),
		componentMap(
			"ButtonRow",
			"lib.core.widgets.Row",
			"Column",
			RectReport{X: 32, Y: 360, W: 496, H: 44},
			map[string]string{"child_count": "4", "accessibility_role": "none"},
		),
		componentMap(
			"SaveButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 32, Y: 360, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "save",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"ResetButton",
			"lib.core.widgets.Button",
			"ButtonRow",
			RectReport{X: 176, Y: 360, W: 132, H: 44},
			map[string]string{
				"focused":            "false",
				"press_count":        "1",
				"action":             "reset",
				"accessibility_role": "button",
			},
		),
		componentMap(
			"Spacer",
			"lib.core.widgets.Spacer",
			"ButtonRow",
			RectReport{X: 320, Y: 360, W: 16, H: 44},
			map[string]string{"min_w": "16", "min_h": "44", "accessibility_role": "none"},
		),
		componentMap(
			"StatusText",
			"lib.core.widgets.StatusText",
			"ButtonRow",
			RectReport{X: 344, Y: 360, W: 184, H: 44},
			map[string]string{
				"role":               "status",
				"status_code":        "2",
				"text_len":           "6",
				"accessibility_role": "label",
			},
		),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "production-widgets-v1",
		"root_id":       0,
		"node_count":    18,
		"focused_id":    7,
		"nodes": []any{
			treeNodeMap(
				0,
				"SurfaceReleaseFormApp",
				"root",
				-1,
				0,
				1,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 560, H: 420},
			),
			treeNodeMap(
				1,
				"Panel",
				"panel",
				0,
				0,
				2,
				1,
				false,
				RectReport{X: 0, Y: 0, W: 560, H: 420},
			),
			treeNodeMap(
				2,
				"Stack",
				"stack",
				1,
				0,
				3,
				1,
				false,
				RectReport{X: 16, Y: 16, W: 528, H: 396},
			),
			treeNodeMap(
				3,
				"Column",
				"column",
				2,
				0,
				4,
				9,
				false,
				RectReport{X: 24, Y: 24, W: 512, H: 388},
			),
			treeNodeMap(
				4,
				"TitleText",
				"text",
				3,
				0,
				-1,
				0,
				false,
				RectReport{X: 32, Y: 32, W: 496, H: 28},
			),
			treeNodeMap(
				5,
				"DescriptionText",
				"text",
				3,
				1,
				-1,
				0,
				false,
				RectReport{X: 32, Y: 68, W: 496, H: 28},
			),
			treeNodeMap(
				6,
				"NameLabel",
				"label",
				3,
				2,
				-1,
				0,
				false,
				RectReport{X: 32, Y: 104, W: 496, H: 24},
			),
			treeNodeMap(
				7,
				"NameTextBox",
				"textbox",
				3,
				3,
				-1,
				0,
				true,
				RectReport{X: 32, Y: 132, W: 496, H: 44},
			),
			treeNodeMap(
				8,
				"EmailLabel",
				"label",
				3,
				4,
				-1,
				0,
				false,
				RectReport{X: 32, Y: 184, W: 496, H: 24},
			),
			treeNodeMap(
				9,
				"EmailTextBox",
				"textbox",
				3,
				5,
				-1,
				0,
				true,
				RectReport{X: 32, Y: 212, W: 496, H: 44},
			),
			treeNodeMap(
				10,
				"SubscribeCheckbox",
				"checkbox",
				3,
				6,
				-1,
				0,
				true,
				RectReport{X: 32, Y: 264, W: 496, H: 32},
			),
			treeNodeMap(
				11,
				"TermsScroll",
				"scroll",
				3,
				7,
				12,
				1,
				false,
				RectReport{X: 32, Y: 304, W: 496, H: 48},
			),
			treeNodeMap(
				12,
				"TermsText",
				"text",
				11,
				0,
				-1,
				0,
				false,
				RectReport{X: 36, Y: 308, W: 488, H: 24},
			),
			treeNodeMap(
				13,
				"ButtonRow",
				"row",
				3,
				8,
				14,
				4,
				false,
				RectReport{X: 32, Y: 360, W: 496, H: 44},
			),
			treeNodeMap(
				14,
				"SaveButton",
				"button",
				13,
				0,
				-1,
				0,
				true,
				RectReport{X: 32, Y: 360, W: 132, H: 44},
			),
			treeNodeMap(
				15,
				"ResetButton",
				"button",
				13,
				1,
				-1,
				0,
				true,
				RectReport{X: 176, Y: 360, W: 132, H: 44},
			),
			treeNodeMap(
				16,
				"Spacer",
				"spacer",
				13,
				2,
				-1,
				0,
				false,
				RectReport{X: 320, Y: 360, W: 16, H: 44},
			),
			treeNodeMap(
				17,
				"StatusText",
				"status",
				13,
				3,
				-1,
				0,
				false,
				RectReport{X: 344, Y: 360, W: 184, H: 44},
			),
		},
		"layout_passes": []any{
			map[string]any{
				"component_id": 7,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 32, Y: 132, W: 320, H: 44}),
				"measured":     map[string]any{"w": 320, "h": 44},
			},
			map[string]any{
				"component_id": 9,
				"pass":         "initial",
				"bounds":       rectMap(RectReport{X: 32, Y: 212, W: 320, H: 44}),
				"measured":     map[string]any{"w": 320, "h": 44},
			},
			map[string]any{
				"component_id": 11,
				"pass":         "scroll",
				"bounds":       rectMap(RectReport{X: 32, Y: 304, W: 496, H: 48}),
				"measured":     map[string]any{"w": 496, "h": 120},
			},
			map[string]any{
				"component_id": 7,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 32, Y: 132, W: 496, H: 44}),
				"measured":     map[string]any{"w": 496, "h": 44},
			},
			map[string]any{
				"component_id": 9,
				"pass":         "resize",
				"bounds":       rectMap(RectReport{X: 32, Y: 212, W: 496, H: 44}),
				"measured":     map[string]any{"w": 496, "h": 44},
			},
			map[string]any{
				"component_id": 17,
				"pass":         "status-update",
				"bounds":       rectMap(RectReport{X: 344, Y: 360, W: 184, H: 44}),
				"measured":     map[string]any{"w": 184, "h": 44},
			},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"focus_order": []any{7, 9, 10, 14, 15},
		"dispatch_paths": []any{
			map[string]any{
				"event":     "click",
				"target_id": 7,
				"x":         48,
				"y":         148,
				"path":      []any{0, 1, 2, 3, 7},
			},
			map[string]any{
				"event":     "click",
				"target_id": 9,
				"x":         48,
				"y":         228,
				"path":      []any{0, 1, 2, 3, 9},
			},
			map[string]any{
				"event":     "click",
				"target_id": 10,
				"x":         48,
				"y":         280,
				"path":      []any{0, 1, 2, 3, 10},
			},
			map[string]any{
				"event":     "key",
				"target_id": 14,
				"x":         48,
				"y":         376,
				"path":      []any{0, 1, 2, 3, 13, 14},
			},
			map[string]any{
				"event":     "key",
				"target_id": 15,
				"x":         192,
				"y":         376,
				"path":      []any{0, 1, 2, 3, 13, 15},
			},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface/release/surface_release_form.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          18,
			"capacity":            32,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{
				"helper":         "widgets.panel_content_rect",
				"target":         "Panel",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.stack_layout",
				"target":         "Stack",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.scroll_set_offset",
				"target":         "TermsScroll",
				"pass":           "scroll",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.row_layout",
				"target":         "ButtonRow",
				"pass":           "initial",
				"changed_bounds": true,
			},
			map[string]any{
				"helper":         "widgets.column_layout",
				"target":         "Column",
				"pass":           "resize",
				"changed_bounds": true,
			},
		},
		"focus_helpers": []any{
			map[string]any{
				"helper": "tree_focus_next",
				"before": "NameTextBox",
				"after":  "EmailTextBox",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "EmailTextBox",
				"after":  "SubscribeCheckbox",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "SubscribeCheckbox",
				"after":  "SaveButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "SaveButton",
				"after":  "ResetButton",
			},
			map[string]any{
				"helper": "tree_focus_next",
				"before": "ResetButton",
				"after":  "NameTextBox",
			},
		},
		"hit_tests": []any{
			map[string]any{
				"helper": "widgets.hit_test_release_form",
				"x":      48,
				"y":      148,
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 3, 7},
			},
			map[string]any{
				"helper": "widgets.hit_test_release_form",
				"x":      48,
				"y":      228,
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 3, 9},
			},
			map[string]any{
				"helper": "widgets.hit_test_release_form",
				"x":      48,
				"y":      280,
				"target": "SubscribeCheckbox",
				"path":   []any{0, 1, 2, 3, 10},
			},
			map[string]any{
				"helper": "widgets.hit_test_release_form",
				"x":      48,
				"y":      320,
				"target": "TermsScroll",
				"path":   []any{0, 1, 2, 3, 11},
			},
			map[string]any{
				"helper": "widgets.hit_test_release_form",
				"x":      192,
				"y":      376,
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 3, 13, 15},
			},
		},
		"dispatch_paths": []any{
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "NameTextBox",
				"path":   []any{0, 1, 2, 3, 7},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "EmailTextBox",
				"path":   []any{0, 1, 2, 3, 9},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "SubscribeCheckbox",
				"path":   []any{0, 1, 2, 3, 10},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "TermsScroll",
				"path":   []any{0, 1, 2, 3, 11},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "SaveButton",
				"path":   []any{0, 1, 2, 3, 13, 14},
			},
			map[string]any{
				"helper": "tree_build_dispatch_path",
				"target": "ResetButton",
				"path":   []any{0, 1, 2, 3, 13, 15},
			},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":        "tetra.surface.toolkit.v1",
		"toolkit_level": "production-widgets-v1",
		"release_scope": "surface-v1-linux-web",
		"source":        "examples/surface/release/surface_release_form.tetra",
		"sources": []any{
			"examples/surface/release/surface_release_form.tetra",
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
		},
		"module":                       "lib.core.widgets",
		"style_module":                 "lib.core.style",
		"experimental":                 false,
		"production_claim":             true,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"example_count":                3,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widget_set": []any{
			"Text",
			"Label",
			"StatusText",
			"Button",
			"TextBox",
			"Checkbox",
			"Row",
			"Column",
			"Panel",
			"Stack",
			"Scroll",
			"Spacer",
		},
		"state_set": []any{
			"normal",
			"focused",
			"hovered",
			"pressed",
			"disabled",
			"error",
		},
		"layout_features": []any{
			"padding",
			"margin",
			"spacing",
			"min_size",
			"max_size",
			"fill",
			"scroll_offset",
		},
		"theme":             true,
		"safe_text_storage": true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Stack", "Stack", 2, "", true),
			toolkitWidgetMap("Column", "Column", 3, "", true),
			toolkitWidgetMap("TitleText", "Text", 4, "label", true),
			toolkitWidgetMap("DescriptionText", "Text", 5, "description", true),
			toolkitWidgetMap("NameLabel", "Label", 6, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 7, "", true),
			toolkitWidgetMap("EmailLabel", "Label", 8, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 9, "", true),
			toolkitWidgetMap("SubscribeCheckbox", "Checkbox", 10, "", true),
			toolkitWidgetMap("TermsScroll", "Scroll", 11, "", true),
			toolkitWidgetMap("TermsText", "Text", 12, "description", true),
			toolkitWidgetMap("ButtonRow", "Row", 13, "", true),
			toolkitWidgetMap("SaveButton", "Button", 14, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 15, "reset", true),
			toolkitWidgetMap("Spacer", "Spacer", 16, "", true),
			toolkitWidgetMap("StatusText", "StatusText", 17, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:label_init",
			"lib/core/widgets/widgets.tetra:status_text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:checkbox_init",
			"lib/core/widgets/widgets.tetra:checkbox_toggle",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:stack_init",
			"lib/core/widgets/widgets.tetra:scroll_init",
			"lib/core/widgets/widgets.tetra:scroll_set_offset",
			"lib/core/widgets/widgets.tetra:spacer_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:hit_test_release_form",
			"lib/core/widgets/style.tetra:default_theme",
			"lib/core/widgets/style.tetra:style_for_state",
		},
	}
	report["events"] = []any{
		eventMap(
			1,
			"mouse_up",
			"NameTextBox",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"},
			48,
			148,
			0,
			560,
			420,
			map[string]string{
				"SurfaceReleaseFormApp.focused_id": "-1",
				"NameTextBox.focused":              "false",
			},
			map[string]string{
				"SurfaceReleaseFormApp.focused_id": "7",
				"NameTextBox.focused":              "true",
			},
		),
		textEventMap(
			2,
			"NameTextBox",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"},
			3,
			"416461",
			560,
			420,
			map[string]string{"NameTextBox.buffer": "", "EmailTextBox.buffer": ""},
			map[string]string{"NameTextBox.buffer": "Ada", "EmailTextBox.buffer": ""},
		),
		keyEventMap(
			3,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			9,
			560,
			420,
			map[string]string{"SurfaceReleaseFormApp.focused_id": "7"},
			map[string]string{"SurfaceReleaseFormApp.focused_id": "9"},
		),
		textEventMap(
			4,
			"EmailTextBox",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "EmailTextBox"},
			5,
			"7465747261",
			560,
			420,
			map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"},
			map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"},
		),
		keyEventMap(
			5,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			9,
			560,
			420,
			map[string]string{"SurfaceReleaseFormApp.focused_id": "9"},
			map[string]string{"SurfaceReleaseFormApp.focused_id": "10"},
		),
		keyEventMap(
			6,
			"SubscribeCheckbox",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "SubscribeCheckbox"},
			32,
			560,
			420,
			map[string]string{
				"SubscribeCheckbox.checked":      "false",
				"SubscribeCheckbox.toggle_count": "0",
			},
			map[string]string{
				"SubscribeCheckbox.checked":      "true",
				"SubscribeCheckbox.toggle_count": "1",
			},
		),
		eventMap(
			7,
			"scroll",
			"TermsScroll",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "TermsScroll"},
			48,
			320,
			0,
			560,
			420,
			map[string]string{"TermsScroll.offset_y": "0"},
			map[string]string{"TermsScroll.offset_y": "16"},
		),
		keyEventMap(
			8,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			9,
			560,
			420,
			map[string]string{"SurfaceReleaseFormApp.focused_id": "10"},
			map[string]string{"SurfaceReleaseFormApp.focused_id": "14"},
		),
		keyEventMap(
			9,
			"SaveButton",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "SaveButton"},
			32,
			560,
			420,
			map[string]string{
				"SurfaceReleaseFormApp.save_count": "0",
				"StatusText.status_code":           "0",
			},
			map[string]string{
				"SurfaceReleaseFormApp.save_count": "1",
				"StatusText.status_code":           "1",
			},
		),
		keyEventMap(
			10,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			9,
			560,
			420,
			map[string]string{"SurfaceReleaseFormApp.focused_id": "14"},
			map[string]string{"SurfaceReleaseFormApp.focused_id": "15"},
		),
		keyEventMap(
			11,
			"ResetButton",
			[]any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "ResetButton"},
			13,
			560,
			420,
			map[string]string{
				"SurfaceReleaseFormApp.reset_count": "0",
				"StatusText.status_code":            "1",
				"NameTextBox.buffer":                "Ada",
				"EmailTextBox.buffer":               "tetra",
			},
			map[string]string{
				"SurfaceReleaseFormApp.reset_count": "1",
				"StatusText.status_code":            "2",
				"NameTextBox.buffer":                "",
				"EmailTextBox.buffer":               "",
			},
		),
		keyEventMap(
			12,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			9,
			560,
			420,
			map[string]string{"SurfaceReleaseFormApp.focused_id": "15"},
			map[string]string{"SurfaceReleaseFormApp.focused_id": "7"},
		),
		resizeEventMap(
			13,
			"SurfaceReleaseFormApp",
			[]any{"SurfaceReleaseFormApp"},
			560,
			420,
			map[string]string{
				"SurfaceReleaseFormApp.focused_id": "7",
				"NameTextBox.bounds.w":             "320",
				"EmailTextBox.bounds.w":            "320",
			},
			map[string]string{
				"SurfaceReleaseFormApp.focused_id": "7",
				"NameTextBox.bounds.w":             "496",
				"EmailTextBox.bounds.w":            "496",
			},
		),
	}
	report["frames"] = []any{
		map[string]any{
			"order":     1,
			"width":     320,
			"height":    240,
			"stride":    1280,
			"checksum":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"presented": true,
		},
		map[string]any{
			"order":     2,
			"width":     560,
			"height":    420,
			"stride":    2240,
			"checksum":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"presented": true,
		},
		map[string]any{
			"order":     3,
			"width":     560,
			"height":    420,
			"stride":    2240,
			"checksum":  "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"presented": true,
		},
		map[string]any{
			"order":     4,
			"width":     560,
			"height":    420,
			"stride":    2240,
			"checksum":  "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"presented": true,
		},
		map[string]any{
			"order":     5,
			"width":     560,
			"height":    420,
			"stride":    2240,
			"checksum":  "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"presented": true,
		},
	}
	report["state_transitions"] = []any{
		map[string]any{
			"order":     1,
			"component": "SurfaceReleaseFormApp",
			"field":     "focused_id",
			"before":    "-1",
			"after":     "7",
			"cause":     "mouse_up",
		},
		map[string]any{
			"order":     2,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "Ada",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     3,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "",
			"after":     "tetra",
			"cause":     "text_input",
		},
		map[string]any{
			"order":     4,
			"component": "SubscribeCheckbox",
			"field":     "checked",
			"before":    "false",
			"after":     "true",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     5,
			"component": "TermsScroll",
			"field":     "offset_y",
			"before":    "0",
			"after":     "16",
			"cause":     "scroll",
		},
		map[string]any{
			"order":     6,
			"component": "SurfaceReleaseFormApp",
			"field":     "save_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     7,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "0",
			"after":     "1",
			"cause":     "save",
		},
		map[string]any{
			"order":     8,
			"component": "NameTextBox",
			"field":     "buffer",
			"before":    "Ada",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     9,
			"component": "EmailTextBox",
			"field":     "buffer",
			"before":    "tetra",
			"after":     "",
			"cause":     "reset",
		},
		map[string]any{
			"order":     10,
			"component": "SurfaceReleaseFormApp",
			"field":     "reset_count",
			"before":    "0",
			"after":     "1",
			"cause":     "key_down",
		},
		map[string]any{
			"order":     11,
			"component": "StatusText",
			"field":     "status_code",
			"before":    "1",
			"after":     "2",
			"cause":     "reset",
		},
		map[string]any{
			"order":     12,
			"component": "SurfaceReleaseFormApp",
			"field":     "focused_id",
			"before":    "15",
			"after":     "7",
			"cause":     "tab",
		},
		map[string]any{
			"order":     13,
			"component": "SurfaceReleaseFormApp",
			"field":     "NameTextBox.bounds.w",
			"before":    "320",
			"after":     "496",
			"cause":     "resize",
		},
		map[string]any{
			"order":     14,
			"component": "SurfaceReleaseFormApp",
			"field":     "EmailTextBox.bounds.w",
			"before":    "320",
			"after":     "496",
			"cause":     "resize",
		},
	}
	report["cases"] = append(
		report["cases"].([]any),
		map[string]any{
			"name": "production toolkit required widget set",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit style module default theme",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit style states normal focused hovered pressed disabled error",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Text Label StatusText evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Button TextBox Checkbox evidence",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Row Column Panel Stack Scroll Spacer layout",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit component tree api reuse",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit TextBox focus input editing",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Checkbox toggle routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Scroll offset routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Save action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit Reset action routed",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit StatusText updates",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit safe text storage",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit no demo-local widget structs",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit browser host separation",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
		map[string]any{
			"name": "production toolkit rendered frame update",
			"kind": "positive",
			"ran":  true,
			"pass": true,
		},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal production toolkit report: %v", err)
	}
	return raw
}

// ---- template_smoke_test.go ----

func TestValidateTemplateSmokeReportAcceptsP21TemplateEvidence(t *testing.T) {
	raw := validTemplateSmokeReportJSON()
	if err := ValidateTemplateSmokeReport(raw); err != nil {
		t.Fatalf("ValidateTemplateSmokeReport failed: %v\n%s", err, raw)
	}
}

func TestValidateTemplateSmokeReportRejectsMissingTemplateKind(t *testing.T) {
	raw := strings.Replace(
		string(validTemplateSmokeReportJSON()),
		`"kind":"web-canvas"`,
		`"kind":"browser-widget"`,
		1,
	)
	err := ValidateTemplateSmokeReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing web-canvas template to fail")
	}
	if !strings.Contains(err.Error(), "web-canvas") {
		t.Fatalf("error = %v, want web-canvas diagnostic", err)
	}
}

func TestValidateTemplateSmokeReportRejectsForbiddenRuntimeImports(t *testing.T) {
	raw := strings.Replace(
		string(validTemplateSmokeReportJSON()),
		`"imports":["lib.core.surface","lib.core.block","lib.core.morph"]`,
		`"imports":["lib.core.surface","React","Electron"]`,
		1,
	)
	err := ValidateTemplateSmokeReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected forbidden runtime imports to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "react") ||
		!strings.Contains(strings.ToLower(err.Error()), "electron") {
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
	templateLines := []string{
		"    " + templateSmokeTemplateJSON("command-palette", false, false),
		"    " + templateSmokeTemplateJSON("settings", false, false),
		"    " + templateSmokeTemplateJSON("dashboard", false, false),
		"    " + templateSmokeTemplateJSON("editor-shell", false, false),
		"    " + templateSmokeTemplateJSON("studio-shell", true, false),
		"    " + templateSmokeTemplateJSON("multi-window-notes", true, false),
		"    " + templateSmokeTemplateJSON("web-canvas", false, true),
	}
	return []byte(strings.Join([]string{
		"{",
		`  "schema": "tetra.surface.template-smoke.v1",`,
		`  "model": "surface-template-smoke-v1",`,
		`  "release_scope": "surface-v1-linux-web",`,
		`  "producer": "scripts/release/surface/surface-template-smoke.sh",`,
		`  "command": "tetra new surface-app",`,
		`  "template_count": 7,`,
		`  "templates": [`,
		strings.Join(templateLines, ",\n"),
		`  ],`,
		`  "inspector_evidence": ` + templateSmokeInspectorEvidenceJSON() + `,`,
		`  "visual_evidence": ` + templateSmokeVisualEvidenceJSON() + `,`,
		`  "morph_to_pixels": ` +
			validMorphToPixelsChainJSON("templates/studio-shell/src/main.tetra") + `,`,
		`  "package_evidence": [` + templateSmokePackageEvidenceJSON() + `],`,
		`  "negative_guards": ` + templateSmokeNegativeGuardsJSON() + `,`,
		`  "pass": true`,
		"}",
		"",
	}, "\n"))
}

func templateSmokeTemplateJSON(kind string, usesAppShell bool, webCanvas bool) string {
	dir := "templates/" + kind
	imports := []string{"lib.core.surface", "lib.core.block", "lib.core.morph"}
	if usesAppShell {
		imports = append(imports, "lib.core.surface_app_shell")
	}
	return mustCompactJSONValue(map[string]any{
		"kind":              kind,
		"project_dir":       dir,
		"source":            dir + "/src/main.tetra",
		"capsule":           dir + "/Capsule.t4",
		"template_metadata": dir + "/surface-template.json",
		"targets":           []string{"linux-x64", "wasm32-web"},
		"imports":           imports,
		"recipe_count":      4,
		"block_morph_only":  true,
		"uses_app_shell":    usesAppShell,
		"web_canvas":        webCanvas,
		"commands":          templateSmokeCommands(kind),
		"source_scan":       templateSmokeSourceScan(),
	})
}

func templateSmokeCommands(kind string) []map[string]any {
	return []map[string]any{
		templateSmokeCommand("generate", "tetra new surface-app --template "+kind),
		templateSmokeCommand("check", "tetra check"),
		templateSmokeCommand("build", "tetra build --target linux-x64"),
		templateSmokeCommand("run", "tetra run --target linux-x64"),
		templateSmokeCommand("inspect", "surface-inspector"),
		templateSmokeCommand("visual", "surface-visual-diff"),
		templateSmokeCommand("package", "tar surface-template-"+kind+".tar.gz"),
	}
}

func templateSmokeCommand(kind string, command string) map[string]any {
	return map[string]any{
		"kind":      kind,
		"command":   command,
		"pass":      true,
		"exit_code": 0,
	}
}

func templateSmokeSourceScan() map[string]any {
	return map[string]any{
		"react_import":      false,
		"electron_import":   false,
		"dom_app_ui_tree":   false,
		"css_runtime":       false,
		"core_widgets":      false,
		"platform_widgets":  false,
		"user_js_app_logic": false,
		"pass":              true,
	}
}

func templateSmokeInspectorEvidenceJSON() string {
	return mustCompactJSONValue(map[string]any{
		"path":  "surface-inspector.json",
		"model": "surface-inspector-v1",
		"pass":  true,
	})
}

func templateSmokeVisualEvidenceJSON() string {
	return mustCompactJSONValue(map[string]any{
		"path":   "template-visual/surface-visual-regression.json",
		"schema": "tetra.surface.visual-regression.v1",
		"pass":   true,
	})
}

func templateSmokePackageEvidenceJSON() string {
	return mustCompactJSONValue(map[string]any{
		"path": "packages/surface-template-command-palette.tar.gz",
		"kind": "tar.gz",
		"sha256": ("sha256:" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
			"aaaa"),
		"pass": true,
	})
}

func templateSmokeNegativeGuardsJSON() string {
	return mustCompactJSONValue(map[string]any{
		"no_react_import":           true,
		"no_electron_import":        true,
		"no_dom_app_ui_tree":        true,
		"no_css_runtime":            true,
		"no_core_widgets":           true,
		"no_platform_widgets":       true,
		"no_user_js_app_logic":      true,
		"cookbook_uses_block_morph": true,
	})
}

// ---- token_graph_test.go ----

func TestValidateTokenGraphContractAcceptsP07Evidence(t *testing.T) {
	root := writeTokenGraphReferenceRoot(
		t,
		("import lib.core.morph as morph\nfunc main() -> Int:\n    " +
			"return morph.capsule_default().token_graph_hash\n"),
	)
	if err := ValidateTokenGraphContract(
		validTokenGraphContractRaw(t, nil),
		validP07TokenGraphReportRaw(t, nil),
		TokenGraphValidationOptions{Root: root},
	); err != nil {
		t.Fatalf("ValidateTokenGraphContract failed: %v", err)
	}
}

func TestValidateTokenGraphContractRejectsP07DiagnosticsGaps(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any, map[string]any)
		want   string
	}{
		{
			name: "missing token graph",
			mutate: func(_ map[string]any, report map[string]any) {
				morph := report["morph"].(map[string]any)
				delete(morph, "token_graph")
			},
			want: "token_graph",
		},
		{
			name: "alias cycle guard disabled",
			mutate: func(_ map[string]any, report map[string]any) {
				graph := tokenGraphFromReport(report)
				graph["alias_cycle_rejected"] = false
			},
			want: "alias_cycle",
		},
		{
			name: "duplicate source token",
			mutate: func(_ map[string]any, report map[string]any) {
				graph := tokenGraphFromReport(report)
				tokens := graph["tokens"].([]any)
				graph["tokens"] = append(tokens, map[string]any{
					"id":       "color.bg",
					"category": "color",
					"kind":     "rgba",
					"value":    "#000000ff",
					"source":   "theme",
					"hash": ("sha256:" +
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
						"aaaa"),
				})
			},
			want: "duplicate",
		},
		{
			name: "material missing token",
			mutate: func(_ map[string]any, report map[string]any) {
				morph := report["morph"].(map[string]any)
				materials := morph["materials"].([]any)
				material := materials[0].(map[string]any)
				material["fill"] = "color.not_declared"
			},
			want: "missing token",
		},
		{
			name: "css runtime admitted",
			mutate: func(contract map[string]any, _ map[string]any) {
				contract["forbidden_runtime_models"] = []any{
					"DOM style runtime",
					"React runtime",
					"Electron runtime",
				}
			},
			want: "CSS cascade runtime",
		},
		{
			name: "multiple color sources",
			mutate: func(contract map[string]any, _ map[string]any) {
				source := contract["source_of_truth"].(map[string]any)
				source["multiple_color_sources"] = true
			},
			want: "multiple_color_sources",
		},
		{
			name: "override order drift",
			mutate: func(contract map[string]any, _ map[string]any) {
				contract["override_order"] = []any{"base", "state", "theme", "density", "variant", "local"}
			},
			want: "override_order",
		},
		{
			name: "density dpi mismatch",
			mutate: func(_ map[string]any, report map[string]any) {
				graph := tokenGraphFromReport(report)
				density := graph["density_dpi"].([]any)
				first := density[0].(map[string]any)
				first["target_dpi"] = 72
			},
			want: "density",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTokenGraphReferenceRoot(
				t,
				"import lib.core.morph as morph\nfunc main() -> Int:\n    return morph.schema_v1()\n",
			)
			contract := tokenGraphContractMap(t)
			report := p07TokenGraphReportMap(t)
			tc.mutate(contract, report)
			err := ValidateTokenGraphContract(
				mustJSON(t, contract),
				mustJSON(t, report),
				TokenGraphValidationOptions{Root: root},
			)
			if err == nil ||
				!strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("ValidateTokenGraphContract err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateTokenGraphContractRejectsRawLiteralsInReferenceSource(t *testing.T) {
	root := writeTokenGraphReferenceRoot(
		t,
		("import lib.core.surface as surface\nfunc main() -> Int:\n    " +
			"let c: surface.Color = surface.Color(r: 1, g: 2, b: 3, a: " +
			"255)\n    return c.r\n"),
	)
	err := ValidateTokenGraphContract(
		validTokenGraphContractRaw(t, nil),
		validP07TokenGraphReportRaw(t, nil),
		TokenGraphValidationOptions{Root: root},
	)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "raw literal") {
		t.Fatalf("ValidateTokenGraphContract err = %v, want raw literal rejection", err)
	}
}

func validTokenGraphContractRaw(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	contract := tokenGraphContractMap(t)
	if mutate != nil {
		mutate(contract)
	}
	return mustJSON(t, contract)
}

func validP07TokenGraphReportRaw(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	report := p07TokenGraphReportMap(t)
	if mutate != nil {
		mutate(report)
	}
	return mustJSON(t, report)
}

func tokenGraphContractMap(t *testing.T) map[string]any {
	t.Helper()
	raw := []byte(strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.token-graph.contract.v1\",",
		"  \"status\": \"current\",",
		"  \"surface_scope\": \"surface-token-graph-linux-web\",",
		("  \"source_of_truth\": {\"module\":\"lib.core.morph\",\"namespace\":\"tetr" +
			"a.surface.morph.app\",\"source\":\"capsule\",\"single_token_graph\":true," +
			"\"explicit_imports\":true,\"no_global_cascade\":true,\"multiple_color_sou" +
			"rces\":false},"),
		("  \"required_categories\": [\"color\",\"space\",\"radius\",\"border\",\"el" +
			"evation\",\"opacity\",\"typography\",\"motion\",\"z\",\"assets\",\"density" +
			"\"],"),
		("  \"required_tokens\": [\"color.bg\",\"color.surface\",\"color.surfaceAlph" +
			"a\",\"color.accent\",\"color.muted\",\"color.warning\",\"space.3\",\"radiu" +
			"s.sm\",\"radius.md\",\"radius.lg\",\"border.subtle\",\"border.glass\",\"el" +
			"evation.2\",\"elevation.3\",\"opacity.disabled\",\"type.label\",\"motion.f" +
			"ast\",\"motion.soft\",\"z.base\",\"assets.gradient.vertical\",\"assets.ico" +
			"n.fallback\",\"density.1x\"],"),
		("  \"reference_sources\": [\"examples/surface/morph_core/surface_morph_c" +
			"ommand_palette.tetra\"],"),
		"  \"allowed_raw_literal_scopes\": [",
		("    {\"path\":\"lib/core/morph/morph.tetra\",\"reason\":\"canonical token " +
			"graph source\"},"),
		("    {\"path\":\"lib/core/widgets/style.tetra\",\"reason\":\"legacy Surface" +
			" v1 style compatibility\"},"),
		("    {\"path\":\"examples/surface/block_*/surface_block_*.tetra\",\"reason" +
			"\":\"experimental raw Block fixture until recipe migration\"}"),
		"  ],",
		("  \"forbidden_runtime_models\": [\"CSS cascade runtime\",\"DOM style runt" +
			"ime\",\"React runtime\",\"Electron runtime\",\"platform-native widgets\"],"),
		("  \"override_order\": [\"base\",\"theme\",\"density\",\"variant\",\"state" +
			"\",\"local\"],"),
		"  \"density_dpi\": [",
		("    {\"target\":\"headless\",\"token\":\"density.1x\",\"target_dpi\":96,\"" +
			"scale_milli\":1000,\"rounding_policy\":\"integer-half-up-v1\"},"),
		("    {\"target\":\"linux-x64-real-window\",\"token\":\"density.1x\",\"targe" +
			"t_dpi\":96,\"scale_milli\":1000,\"rounding_policy\":\"integer-half-up-v1\"" +
			"},"),
		("    {\"target\":\"wasm32-web-browser-canvas\",\"token\":\"density.1x\",\"t" +
			"arget_dpi\":96,\"scale_milli\":1000,\"rounding_policy\":\"integer-half-up-" +
			"v1\"}"),
		"  ],",
		("  \"diagnostics_required\": [\"alias_cycle\",\"missing_token\",\"duplicate" +
			"_source\",\"raw_literal\",\"unresolved_fallback\",\"css_runtime\",\"multip" +
			"le_color_sources\",\"override_order\",\"density_dpi\"],"),
		("  \"negative_guards\": {\"alias_cycle_rejected\":true,\"missing_token_rej" +
			"ected\":true,\"duplicate_source_rejected\":true,\"raw_literal_rejected\":" +
			"true,\"unresolved_fallback_rejected\":true,\"css_runtime_rejected\":true" +
			",\"multiple_color_sources_rejected\":true,\"override_order_rejected\":tr" +
			"ue,\"density_dpi_rejected\":true},"),
		("  \"nonclaims\": [\"no CSS cascade runtime\",\"no React runtime\",\"no Ele" +
			"ctron runtime\",\"no DOM style runtime\",\"no platform-native widgets\"]"),
		"}",
		"",
	}, "\n"))
	var contract map[string]any
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode token graph contract fixture: %v", err)
	}
	return contract
}

func p07TokenGraphReportMap(t *testing.T) map[string]any {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessMorphSurfaceReportJSON(t, func(morph map[string]any) {
		morph["token_graph"] = p07MorphTokenGraphMap()
		morph["materials"] = p07MorphMaterials()
	}), &report); err != nil {
		t.Fatalf("decode P07 Morph report fixture: %v", err)
	}
	return report
}

func p07MorphTokenGraphMap() map[string]any {
	return map[string]any{
		"schema":            "tetra.surface.morph.token-graph.v1",
		"namespace":         "tetra.surface.morph.app",
		"version":           "1",
		"hash":              "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"source_of_truth":   "capsule",
		"explicit_imports":  true,
		"no_global_cascade": true,
		"fixed_override_order": []any{
			"base",
			"theme",
			"density",
			"variant",
			"state",
			"local",
		},
		"categories": []any{
			"color",
			"space",
			"radius",
			"border",
			"elevation",
			"opacity",
			"typography",
			"motion",
			"z",
			"assets",
			"density",
		},
		"tokens":                       p07MorphTokens(),
		"density_dpi":                  p07DensityMappings(),
		"diagnostics":                  p07TokenGraphDiagnostics(),
		"alias_cycle_rejected":         true,
		"duplicate_source_rejected":    true,
		"raw_literals_in_app_code":     false,
		"unresolved_fallback_rejected": true,
		"fallback_to_random_default":   false,
	}
}

func p07MorphTokens() []any {
	hash := func(seed string) string {
		return "sha256:" + strings.Repeat(seed, 64)
	}
	return []any{
		map[string]any{
			"id":       "color.bg",
			"category": "color",
			"kind":     "rgba",
			"value":    "#0b0f14ff",
			"source":   "capsule",
			"hash":     hash("1"),
		},
		map[string]any{
			"id":       "color.surface",
			"category": "color",
			"kind":     "rgba",
			"value":    "#181f26ff",
			"source":   "capsule",
			"hash":     hash("2"),
		},
		map[string]any{
			"id":       "color.surfaceAlpha",
			"category": "color",
			"kind":     "rgba",
			"value":    "#181f26da",
			"source":   "capsule",
			"hash":     hash("3"),
		},
		map[string]any{
			"id":       "color.accent",
			"category": "color",
			"kind":     "rgba",
			"value":    "#60aef4ff",
			"source":   "capsule",
			"hash":     hash("4"),
		},
		map[string]any{
			"id":       "color.muted",
			"category": "color",
			"kind":     "rgba",
			"value":    "#7e90a3ff",
			"source":   "capsule",
			"hash":     hash("5"),
		},
		map[string]any{
			"id":       "color.warning",
			"category": "color",
			"kind":     "rgba",
			"value":    "#f4cd5cff",
			"source":   "capsule",
			"hash":     hash("6"),
		},
		map[string]any{
			"id":       "space.3",
			"category": "space",
			"kind":     "px",
			"value":    "12",
			"source":   "capsule",
			"hash":     hash("7"),
		},
		map[string]any{
			"id":       "radius.sm",
			"category": "radius",
			"kind":     "px",
			"value":    "8",
			"source":   "capsule",
			"hash":     hash("8"),
		},
		map[string]any{
			"id":       "radius.md",
			"category": "radius",
			"kind":     "px",
			"value":    "10",
			"source":   "capsule",
			"hash":     hash("9"),
		},
		map[string]any{
			"id":       "radius.lg",
			"category": "radius",
			"kind":     "px",
			"value":    "18",
			"source":   "capsule",
			"hash":     hash("a"),
		},
		map[string]any{
			"id":       "border.subtle",
			"category": "border",
			"kind":     "px",
			"value":    "1",
			"source":   "capsule",
			"hash":     hash("b"),
		},
		map[string]any{
			"id":       "border.glass",
			"category": "border",
			"kind":     "px",
			"value":    "1",
			"source":   "capsule",
			"hash":     hash("c"),
		},
		map[string]any{
			"id":       "elevation.2",
			"category": "elevation",
			"kind":     "shadow",
			"value":    "0 3 10 72",
			"source":   "capsule",
			"hash":     hash("d"),
		},
		map[string]any{
			"id":       "elevation.3",
			"category": "elevation",
			"kind":     "shadow",
			"value":    "0 10 24 128",
			"source":   "capsule",
			"hash":     hash("e"),
		},
		map[string]any{
			"id":       "opacity.disabled",
			"category": "opacity",
			"kind":     "alpha",
			"value":    "128",
			"source":   "capsule",
			"hash":     hash("f"),
		},
		map[string]any{
			"id":       "type.label",
			"category": "typography",
			"kind":     "font",
			"value":    "Tetra UI 13 600 18",
			"source":   "capsule",
			"hash":     hash("1"),
		},
		map[string]any{
			"id":       "motion.fast",
			"category": "motion",
			"kind":     "transition",
			"value":    "120 ease.out",
			"source":   "capsule",
			"hash":     hash("2"),
		},
		map[string]any{
			"id":       "motion.soft",
			"category": "motion",
			"kind":     "transition",
			"value":    "180 ease.inOut",
			"source":   "capsule",
			"hash":     hash("3"),
		},
		map[string]any{
			"id":       "z.base",
			"category": "z",
			"kind":     "layer",
			"value":    "0",
			"source":   "capsule",
			"hash":     hash("4"),
		},
		map[string]any{
			"id":       "assets.gradient.vertical",
			"category": "assets",
			"kind":     "gradient",
			"value":    "vertical",
			"source":   "capsule",
			"hash":     hash("5"),
		},
		map[string]any{
			"id":       "assets.icon.fallback",
			"category": "assets",
			"kind":     "icon",
			"value":    "fallback",
			"source":   "capsule",
			"hash":     hash("6"),
		},
		map[string]any{
			"id":       "density.1x",
			"category": "density",
			"kind":     "dpi",
			"value":    "96/1000",
			"source":   "capsule",
			"hash":     hash("7"),
		},
	}
}

func p07MorphMaterials() []any {
	return []any{
		map[string]any{
			"name":                      "surface.base",
			"paint_stack":               []any{"fill", "border", "radius"},
			"fill":                      "color.surface",
			"border":                    "border.subtle",
			"radius":                    "radius.md",
			"shadow":                    "",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "surface.elevated",
			"paint_stack":               []any{"fill", "border", "radius", "shadow"},
			"fill":                      "color.surface",
			"border":                    "border.subtle",
			"radius":                    "radius.md",
			"shadow":                    "elevation.2",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "control.primary",
			"paint_stack":               []any{"fill", "radius"},
			"fill":                      "color.accent",
			"border":                    "",
			"radius":                    "radius.sm",
			"shadow":                    "",
			"overlay":                   "",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
		map[string]any{
			"name":                      "translucent.panel",
			"paint_stack":               []any{"fill", "border", "radius", "shadow", "overlay"},
			"fill":                      "color.surfaceAlpha",
			"border":                    "border.glass",
			"radius":                    "radius.lg",
			"shadow":                    "elevation.3",
			"overlay":                   "assets.gradient.vertical",
			"unsupported_blur":          false,
			"unsupported_blur_rejected": true,
		},
	}
}

func p07DensityMappings() []any {
	return []any{
		map[string]any{
			"target":          "headless",
			"token":           "density.1x",
			"target_dpi":      96,
			"scale_milli":     1000,
			"rounding_policy": "integer-half-up-v1",
		},
		map[string]any{
			"target":          "linux-x64-real-window",
			"token":           "density.1x",
			"target_dpi":      96,
			"scale_milli":     1000,
			"rounding_policy": "integer-half-up-v1",
		},
		map[string]any{
			"target":          "wasm32-web-browser-canvas",
			"token":           "density.1x",
			"target_dpi":      96,
			"scale_milli":     1000,
			"rounding_policy": "integer-half-up-v1",
		},
	}
}

func p07TokenGraphDiagnostics() map[string]any {
	return map[string]any{
		"alias_cycle_rejected":            true,
		"missing_token_rejected":          true,
		"duplicate_source_rejected":       true,
		"raw_literal_rejected":            true,
		"unresolved_fallback_rejected":    true,
		"css_runtime_rejected":            true,
		"multiple_color_sources_rejected": true,
		"override_order_rejected":         true,
		"density_dpi_rejected":            true,
	}
}

func tokenGraphFromReport(report map[string]any) map[string]any {
	morph := report["morph"].(map[string]any)
	return morph["token_graph"].(map[string]any)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON fixture: %v", err)
	}
	return raw
}

func compactJSONSnippet(t *testing.T, raw string) string {
	t.Helper()
	return mustCompactJSONSnippet(raw)
}

func mustCompactJSONSnippet(raw string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(raw)); err != nil {
		panic("compact JSON snippet fixture: " + err.Error())
	}
	return buf.String()
}

func compactJSONValue(t *testing.T, value any) string {
	t.Helper()
	return mustCompactJSONValue(value)
}

func mustCompactJSONValue(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("marshal JSON value fixture: " + err.Error())
	}
	return string(encoded)
}

func compactJSONArrayLine(t *testing.T, raw string) string {
	t.Helper()
	return ",\n    " + compactJSONSnippet(t, raw)
}

func compactJSONArrayLineRaw(raw string) string {
	return ",\n    " + mustCompactJSONSnippet(raw)
}

func compactJSONArrayValueLine(t *testing.T, value any) string {
	t.Helper()
	return ",\n    " + compactJSONValue(t, value)
}

func writeTokenGraphReferenceRoot(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(
		root,
		"examples",
		"surface",
		"morph_core",
		"surface_morph_command_palette.tetra",
	)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// ---- visual_test.go ----

func TestValidateVisualReportAcceptsCompleteEvidence(t *testing.T) {
	raw := validSurfaceVisualReportJSON(t, nil)
	if err := ValidateVisualReport(raw); err != nil {
		t.Fatalf("ValidateVisualReport failed: %v\n%s", err, raw)
	}
}

func TestValidateVisualReportRejectsIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*VisualRegressionReport)
		want   string
	}{
		{
			name: "screenshot only",
			mutate: func(report *VisualRegressionReport) {
				target := &report.Apps[0].Targets[0]
				target.ScreenshotOnly = true
				target.BlockGraphEvidence = false
				target.LayoutEvidence = false
				target.AccessibilityEvidence = false
				target.PerformanceEvidence = false
			},
			want: "screenshot-only",
		},
		{
			name: "stale golden",
			mutate: func(report *VisualRegressionReport) {
				report.Apps[0].Targets[0].GoldenGitHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
			want: "stale golden",
		},
		{
			name: "major drift",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.DiffPixels = 4096
				frame.DiffRatioMilli = 640
				frame.MaxChannelDelta = 64
				frame.Pass = false
			},
			want: "visual drift",
		},
		{
			name: "missing token conformance",
			mutate: func(report *VisualRegressionReport) {
				report.Apps[0].Targets[0].TokenThemeEvidence = false
			},
			want: "token/theme",
		},
		{
			name: "missing required target",
			mutate: func(report *VisualRegressionReport) {
				report.RequiredTargets = append(report.RequiredTargets, "linux-x64-real-window")
			},
			want: "required target",
		},
		{
			name: "missing required source",
			mutate: func(report *VisualRegressionReport) {
				report.RequiredSources = append(
					report.RequiredSources,
					"examples/surface/block_apps/surface_block_settings.tetra",
				)
			},
			want: "required source",
		},
		{
			name: "negative guard missing",
			mutate: func(report *VisualRegressionReport) {
				report.NegativeGuards.ScreenshotOnlyRejected = false
			},
			want: "negative_guards",
		},
		{
			name: "self golden artifact",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.GoldenArtifactPath = frame.ArtifactPath
			},
			want: "self-golden",
		},
		{
			name: "metadata checksum only",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.ArtifactPath = ""
				frame.ArtifactSHA256 = ""
				frame.GoldenArtifactPath = ""
				frame.GoldenArtifactSHA256 = ""
			},
			want: "artifact_path",
		},
		{
			name: "fixture frame only",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.ArtifactPath = "tools/validators/surface/testdata/fixture-frame.rgba"
				frame.GoldenArtifactPath = "tools/validators/surface/testdata/fixture-frame.golden.rgba"
			},
			want: "fixture frame",
		},
		{
			name: "missing png or rgba artifact",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.ArtifactPath = "reports/surface-visual/headless/frame.json"
				frame.GoldenArtifactPath = "reports/surface/goldens/headless/frame.json"
				frame.ArtifactFormat = "json"
			},
			want: "png or rgba",
		},
		{
			name: "artifact checksum mismatch",
			mutate: func(report *VisualRegressionReport) {
				report.Apps[0].Targets[0].Frames[0].ArtifactSHA256 = ("sha256:" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaa")
			},
			want: "artifact_sha256",
		},
		{
			name: "MRB05 guard missing",
			mutate: func(report *VisualRegressionReport) {
				report.NegativeGuards.SelfGoldenRejected = false
			},
			want: "negative_guards",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validSurfaceVisualReportJSON(t, tc.mutate)
			err := ValidateVisualReport(raw)
			if err == nil {
				t.Fatalf("expected visual report %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func validSurfaceVisualReportJSON(t *testing.T, mutate func(*VisualRegressionReport)) []byte {
	t.Helper()
	report := VisualRegressionReport{
		Schema:          VisualRegressionSchemaV1,
		Status:          "pass",
		GitHead:         "c0258b63a636775b114d69d31cb7832fc3991b05",
		GoldenSet:       "surface-visual-regression-v1",
		GoldenHash:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RequiredTargets: []string{"headless"},
		RequiredSources: []string{"examples/surface/block_core/surface_block_system.tetra"},
		Apps: []VisualRegressionAppReport{
			{
				Name:         "surface-block-system",
				Source:       "examples/surface/block_core/surface_block_system.tetra",
				ReferenceApp: true,
				Targets: []VisualRegressionTargetReport{
					{
						Target:                "headless",
						RuntimeReport:         "reports/surface-visual/headless/surface-headless-block-system.json",
						RuntimeSchema:         SchemaV1,
						GitHead:               "c0258b63a636775b114d69d31cb7832fc3991b05",
						GoldenGitHead:         "c0258b63a636775b114d69d31cb7832fc3991b05",
						Renderer:              "software-rgba",
						ScreenshotOnly:        false,
						BlockGraphEvidence:    true,
						TokenThemeEvidence:    true,
						LayoutEvidence:        true,
						AccessibilityEvidence: true,
						PerformanceEvidence:   true,
						Frames: []VisualRegressionFrameReport{
							{
								Order:  1,
								Label:  "initial",
								Width:  320,
								Height: 200,
								Stride: 1280,
								Checksum: ("sha256:" +
									"111111111111111111111111111111111111111111111111111111111111" +
									"1111"),
								GoldenChecksum: ("sha256:" +
									"111111111111111111111111111111111111111111111111111111111111" +
									"1111"),
								ArtifactPath: "reports/surface-visual/headless/frames/initial.rgba",
								ArtifactSHA256: ("sha256:" +
									"111111111111111111111111111111111111111111111111111111111111" +
									"1111"),
								ArtifactFormat:     "rgba",
								GoldenArtifactPath: "reports/surface/goldens/headless/initial.rgba",
								GoldenArtifactSHA256: ("sha256:" +
									"111111111111111111111111111111111111111111111111111111111111" +
									"1111"),
								DiffPixels:            0,
								DiffRatioMilli:        0,
								MaxChannelDelta:       0,
								TolerancePixels:       4,
								ToleranceRatioMilli:   1,
								ToleranceChannelDelta: 1,
								Pass:                  true,
							},
						},
					},
				},
			},
		},
		NegativeGuards: VisualRegressionNegativeGuardsReport{
			ScreenshotOnlyRejected:           true,
			StaleGoldenRejected:              true,
			MajorDriftRejected:               true,
			MissingBlockGraphRejected:        true,
			MissingLayoutRejected:            true,
			MissingAccessibilityRejected:     true,
			MissingPerformanceRejected:       true,
			SelfGoldenRejected:               true,
			MetadataChecksumRejected:         true,
			FixtureFrameOnlyRejected:         true,
			MissingPNGOrRGBAArtifactRejected: true,
		},
	}
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal visual report: %v", err)
	}
	return raw
}

// ---- widget_migration_test.go ----

func TestValidateWidgetMigrationReportAcceptsCompatibilityEvidence(t *testing.T) {
	raw := validWidgetMigrationReportJSON()
	if err := ValidateWidgetMigrationReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateWidgetMigrationReport failed: %v\n%s", err, raw)
	}
}

func TestValidateWidgetMigrationReportRejectsFutureCorePrimitivePromotion(t *testing.T) {
	raw := strings.Replace(
		validWidgetMigrationReportJSON(),
		`"block_only_core_primitive":true`,
		`"block_only_core_primitive":false`,
		1,
	)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected future core primitive promotion to fail")
	}
	if !strings.Contains(err.Error(), "core primitive") {
		t.Fatalf("error = %v, want core primitive diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsWidgetPromotedToCore(t *testing.T) {
	raw := strings.Replace(
		validWidgetMigrationReportJSON(),
		`"widgets_promoted_to_core":false`,
		`"widgets_promoted_to_core":true`,
		1,
	)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected promoted widget core primitive to fail")
	}
	if !strings.Contains(err.Error(), "widgets_promoted_to_core") {
		t.Fatalf("error = %v, want widgets_promoted_to_core diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsBreakingChange(t *testing.T) {
	raw := strings.Replace(
		validWidgetMigrationReportJSON(),
		`"api_breaking_change":false`,
		`"api_breaking_change":true`,
		1,
	)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected API breaking change to fail")
	}
	if !strings.Contains(err.Error(), "breaking") {
		t.Fatalf("error = %v, want breaking-change diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsDocsOnlyWithoutArtifactEvidence(t *testing.T) {
	raw := strings.Replace(
		validWidgetMigrationReportJSON(),
		("  \"artifact_evidence\": {\"equivalence_rows_sha256\":" +
			"\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"," +
			"\"source_scan_sha256\":\"sha256:" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"},\n"),
		"",
		1,
	)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing artifact evidence to fail")
	}
	if !strings.Contains(err.Error(), "artifact_evidence") {
		t.Fatalf("error = %v, want artifact_evidence diagnostic", err)
	}
}

func TestValidateWidgetMigrationReportRejectsMissingTextboxEquivalence(t *testing.T) {
	raw := strings.Replace(
		validWidgetMigrationReportJSON(),
		`"legacy_widget":"TextBox"`,
		`"legacy_widget":"TextBoxLegacy"`,
		1,
	)
	err := ValidateWidgetMigrationReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing TextBox equivalence to fail")
	}
	if !strings.Contains(err.Error(), "TextBox") {
		t.Fatalf("error = %v, want TextBox diagnostic", err)
	}
}

func validWidgetMigrationReportJSON() string {
	return strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.surface.widget-migration.v1\",",
		"  \"model\": \"surface-widget-migration-v1\",",
		"  \"release_scope\": \"surface-v1-linux-web\",",
		("  \"producer\": \"scripts/release/surface/surface-widget-migration-smok" +
			"e.sh\","),
		("  \"source\": \"examples/surface/reference_forms/surface_reference_migr" +
			"ation.tetra\","),
		"  \"reference_app\": \"migration\",",
		"  \"target\": \"linux-x64\",",
		("  \"compatibility_layer\": {\"module\":\"lib.core.widgets\",\"supported_su" +
			"rface_v1\":true,\"current_api_preserved\":true,\"api_breaking_change\":fa" +
			"lse,\"migration_equivalence_helpers\":true,\"migration_docs\":true,\"pass" +
			"\":true},"),
		("  \"release_widget_set\": {\"widgets\":[\"Text\",\"Label\",\"StatusText\"," +
			"\"Button\",\"TextBox\",\"Row\",\"Column\",\"Panel\",\"Checkbox\",\"Stack\"" +
			",\"Scroll\",\"Spacer\"],\"intact\":true,\"non_migration_widget_usage\":fal" +
			"se,\"pass\":true},"),
		"  \"equivalence_rows\": [",
		("    {\"legacy_widget\":\"Panel\",\"legacy_function\":\"widgets.panel_init" +
			"\",\"morph_recipe\":\"recipe_region_panel\",\"block_expander\":\"morph.exp" +
			"and_region_panel\",\"block_kind\":\"Block\",\"legacy_result\":380,\"block_" +
			"result\":380,\"api_unchanged\":true,\"resolves_to_block\":true,\"pass\":tr" +
			"ue},"),
		("    {\"legacy_widget\":\"Button\",\"legacy_function\":\"widgets.button_ini" +
			"t\",\"morph_recipe\":\"recipe_control_action\",\"block_expander\":\"morph." +
			"expand_control_action\",\"block_kind\":\"Block\",\"legacy_result\":1301,\"" +
			"block_result\":1301,\"api_unchanged\":true,\"resolves_to_block\":true,\"pa" +
			"ss\":true},"),
		("    {\"legacy_widget\":\"TextBox\",\"legacy_function\":\"widgets.textbox_i" +
			"nit\",\"morph_recipe\":\"recipe_field_text\",\"block_expander\":\"morph.ex" +
			"pand_field_text\",\"block_kind\":\"Block\",\"legacy_result\":344,\"block_r" +
			"esult\":344,\"api_unchanged\":true,\"resolves_to_block\":true,\"pass\":tru" +
			"e}"),
		"  ],",
		("  \"morph_recipe_migration\": {\"recipes\":[\"recipe_region_panel\",\"reci" +
			"pe_control_action\",\"recipe_field_text\"],\"core_primitives\":[\"Block\"]" +
			",\"block_only_core_primitive\":true,\"widgets_promoted_to_core\":false,\"" +
			"resolves_to_block\":true,\"pass\":true},"),
		("  \"migration_reference_app\": {\"shape\":\"migration\",\"source\":\"examp" +
			"les/surface/reference_forms/surface_reference_migration.tetra\",\"impo" +
			"rts\":[\"lib.core.surface\",\"lib.core.block\",\"lib.core.morph\",\"lib.co" +
			"re.widgets\"],\"compiles\":true,\"runs\":true,\"exit_code\":0,\"uses_widge" +
			"ts_compat\":true,\"uses_morph_recipes\":true,\"resolves_to_block\":true,\"" +
			"pass\":true},"),
		("  \"negative_guards\": {\"no_future_core_primitive_promotion\":true,\"no_" +
			"widget_primary_future_core\":true,\"no_breaking_change\":true,\"no_docs_" +
			"only\":true,\"no_platform_native_runtime_claims\":true},"),
		("  \"artifact_evidence\": {\"equivalence_rows_sha256\":\"sha256:aaaaaaaaaa" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"source_scan" +
			"_sha256\":\"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" +
			"bbbbbbbbbbbbb\"},"),
		"  \"pass\": true",
		"}",
		"",
	}, "\n")
}
