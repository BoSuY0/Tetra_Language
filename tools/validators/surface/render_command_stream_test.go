package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateRenderCommandStreamAcceptsSourceLinkedBlockSceneCommands(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateRenderCommandStreamRejectsUnlinkedOrHandcraftedEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, func(stream map[string]any) {
		stream["source_linked"] = false
		stream["handcrafted_fixture"] = true
	})

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
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, func(stream map[string]any) {
		stream["block_scene_hash"] = "sha256:" + strings.Repeat("e", 64)
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected render command stream block_scene_hash mismatch to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block_scene_hash") {
		t.Fatalf("error = %v, want block_scene_hash diagnostic", err)
	}
}

func TestValidateRenderCommandStreamRejectsCommandMissingSourceRecipeLink(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, func(stream map[string]any) {
		commands := stream["commands"].([]any)
		first := commands[0].(map[string]any)
		first["source"] = "fixtures/precomputed/render.json"
		first["recipe"] = ""
	})

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
	raw := validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t, func(stream map[string]any) {
		commands := stream["commands"].([]any)
		text := commands[8].(map[string]any)
		icon := commands[9].(map[string]any)
		text["marker_only"] = true
		text["raster_hash"] = ""
		icon["marker_only"] = true
		icon["raster_hash"] = ""
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected marker-only text/icon raster stream to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "marker") || !strings.Contains(lower, "raster") {
		t.Fatalf("error = %v, want marker and raster diagnostics", err)
	}
}

func validHeadlessMorphSurfaceReportWithRenderCommandStreamJSON(t *testing.T, mutate func(map[string]any)) []byte {
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
	stream := renderCommandStreamMapForTest(source, snapshot["block_scene_hash"].(string), firstFrame["checksum"].(string))
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
			map[string]any{"kind": "border", "color": "#6eaef4ff", "width": 1, "radius": 8, "opacity": 255},
			map[string]any{"kind": "radius_clip", "radius": 8, "opacity": 255},
			map[string]any{"kind": "shadow", "color": "#00000040", "blur": 8, "offset_y": 2, "opacity": 64},
			map[string]any{"kind": "overlay", "color": "#10182066", "radius": 8, "opacity": 102},
			map[string]any{"kind": "outline", "color": "#6eaef4ff", "width": 1, "radius": 8, "opacity": 255},
		},
	}
}

func renderCommandStreamMapForTest(source string, blockSceneHash string, frameChecksum string) map[string]any {
	commands := []any{
		renderCommandMapForTest(1, "fill", source, "morph.search_input", "search-input-fill", 255),
		renderCommandMapForTest(2, "gradient", source, "morph.search_input", "search-input-gradient", 255),
		renderCommandMapForTest(3, "image_fill", source, "morph.search_input", "search-input-image-fill", 96),
		renderCommandMapForTest(4, "border", source, "morph.search_input", "search-input-border", 255),
		renderCommandMapForTest(5, "radius_clip", source, "morph.search_input", "search-input-radius-clip", 255),
		renderCommandMapForTest(6, "shadow", source, "morph.search_input", "search-input-shadow", 64),
		renderCommandMapForTest(7, "overlay", source, "morph.search_input", "search-input-overlay", 102),
		renderCommandMapForTest(8, "outline", source, "morph.search_input", "search-input-outline", 255),
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

func renderCommandMapForTest(order int, command string, source string, recipe string, layerID string, opacity int) map[string]any {
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
