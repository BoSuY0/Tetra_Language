package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateBlockSceneSnapshotAcceptsRichVisualSpecs(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsCompactPropsOnlyEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t, func(snapshot map[string]any) {
		snapshot["compact_props_only"] = true
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected compact-only Block scene snapshot to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "compact") {
		t.Fatalf("error = %v, want compact diagnostic", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsNonBlockCorePrimitive(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t, func(snapshot map[string]any) {
		snapshot["core_primitives"] = []any{"Block", "Button"}
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected non-Block core primitive to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "button") {
		t.Fatalf("error = %v, want Button diagnostic", err)
	}
}

func TestValidateBlockSceneSnapshotRejectsMissingRichSpecCoverage(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t, func(snapshot map[string]any) {
		coverage := snapshot["spec_coverage"].(map[string]any)
		coverage["motion"] = false
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing motion spec coverage to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "motion") {
		t.Fatalf("error = %v, want motion diagnostic", err)
	}
}

func validHeadlessMorphSurfaceReportWithBlockSceneSnapshotJSON(t *testing.T, mutate func(map[string]any)) []byte {
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
						map[string]any{"kind": "fill", "color": "#10151dff", "radius": 12, "opacity": 255},
						map[string]any{"kind": "border", "color": "#6eaef4ff", "width": 1, "radius": 12, "opacity": 255},
					},
				},
				"text":          map[string]any{"text_len": 0, "color": "#f4f7fbff", "size": 14, "weight": 500},
				"image":         map[string]any{"asset_id": "none", "mode": "none", "opacity": 0},
				"input":         map[string]any{"kind": "none", "focusable": false, "editable": false},
				"event":         map[string]any{"pointer_action": "none", "key_action": "none"},
				"state":         map[string]any{"variant": "surface", "enabled": true},
				"motion":        map[string]any{"duration_ms": 120, "easing": "standard", "reduced_motion_safe": true},
				"accessibility": map[string]any{"role": "group", "label_len": 15, "reading_index": 1},
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
						map[string]any{"kind": "fill", "color": "#202733ff", "radius": 8, "opacity": 255},
						map[string]any{"kind": "shadow", "color": "#00000040", "blur": 8, "offset_y": 2, "opacity": 64},
						map[string]any{"kind": "outline", "color": "#6eaef4ff", "width": 1, "radius": 8, "opacity": 255},
					},
				},
				"text":          map[string]any{"text_len": 12, "hint_len": 10, "color": "#f4f7fbff", "size": 14, "weight": 500},
				"image":         map[string]any{"asset_id": "search-icon", "mode": "template", "tint": "#b7c4d6ff", "opacity": 255},
				"input":         map[string]any{"kind": "text", "focusable": true, "editable": true},
				"event":         map[string]any{"pointer_action": "focus", "key_action": "edit"},
				"state":         map[string]any{"variant": "focused", "enabled": true, "focused": true},
				"motion":        map[string]any{"duration_ms": 140, "easing": "standard", "reduced_motion_safe": true},
				"accessibility": map[string]any{"role": "textbox", "label_len": 13, "focus_index": 1, "reading_index": 2, "actions": []any{"focus", "edit"}},
			},
		},
	}
}
