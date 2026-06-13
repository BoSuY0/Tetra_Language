package surface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTokenGraphContractAcceptsP07Evidence(t *testing.T) {
	root := writeTokenGraphReferenceRoot(t, "import lib.core.morph as morph\nfunc main() -> Int:\n    return morph.capsule_default().token_graph_hash\n")
	if err := ValidateTokenGraphContract(validTokenGraphContractRaw(t, nil), validP07TokenGraphReportRaw(t, nil), TokenGraphValidationOptions{Root: root}); err != nil {
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
				graph["tokens"] = append(tokens, map[string]any{"id": "color.bg", "category": "color", "kind": "rgba", "value": "#000000ff", "source": "theme", "hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
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
				contract["forbidden_runtime_models"] = []any{"DOM style runtime", "React runtime", "Electron runtime"}
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
			root := writeTokenGraphReferenceRoot(t, "import lib.core.morph as morph\nfunc main() -> Int:\n    return morph.schema_v1()\n")
			contract := tokenGraphContractMap(t)
			report := p07TokenGraphReportMap(t)
			tc.mutate(contract, report)
			err := ValidateTokenGraphContract(mustJSON(t, contract), mustJSON(t, report), TokenGraphValidationOptions{Root: root})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("ValidateTokenGraphContract err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateTokenGraphContractRejectsRawLiteralsInReferenceSource(t *testing.T) {
	root := writeTokenGraphReferenceRoot(t, "import lib.core.surface as surface\nfunc main() -> Int:\n    let c: surface.Color = surface.Color(r: 1, g: 2, b: 3, a: 255)\n    return c.r\n")
	err := ValidateTokenGraphContract(validTokenGraphContractRaw(t, nil), validP07TokenGraphReportRaw(t, nil), TokenGraphValidationOptions{Root: root})
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
	raw := []byte(`{
  "schema": "tetra.surface.token-graph.contract.v1",
  "status": "current",
  "surface_scope": "surface-token-graph-linux-web",
  "source_of_truth": {
    "module": "lib.core.morph",
    "namespace": "tetra.surface.morph.app",
    "source": "capsule",
    "single_token_graph": true,
    "explicit_imports": true,
    "no_global_cascade": true,
    "multiple_color_sources": false
  },
  "required_categories": ["color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"],
  "required_tokens": ["color.bg", "color.surface", "color.surfaceAlpha", "color.accent", "color.muted", "color.warning", "space.3", "radius.sm", "radius.md", "radius.lg", "border.subtle", "border.glass", "elevation.2", "elevation.3", "opacity.disabled", "type.label", "motion.fast", "motion.soft", "z.base", "assets.gradient.vertical", "assets.icon.fallback", "density.1x"],
  "reference_sources": ["examples/surface_morph_command_palette.tetra"],
  "allowed_raw_literal_scopes": [
    {"path": "lib/core/morph.tetra", "reason": "canonical token graph source"},
    {"path": "lib/core/style.tetra", "reason": "legacy Surface v1 style compatibility"},
    {"path": "examples/surface_block_*.tetra", "reason": "experimental raw Block fixture until recipe migration"}
  ],
  "forbidden_runtime_models": ["CSS cascade runtime", "DOM style runtime", "React runtime", "Electron runtime", "platform-native widgets"],
  "override_order": ["base", "theme", "density", "variant", "state", "local"],
  "density_dpi": [
    {"target": "headless", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"},
    {"target": "linux-x64-real-window", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"},
    {"target": "wasm32-web-browser-canvas", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"}
  ],
  "diagnostics_required": ["alias_cycle", "missing_token", "duplicate_source", "raw_literal", "unresolved_fallback", "css_runtime", "multiple_color_sources", "override_order", "density_dpi"],
  "negative_guards": {
    "alias_cycle_rejected": true,
    "missing_token_rejected": true,
    "duplicate_source_rejected": true,
    "raw_literal_rejected": true,
    "unresolved_fallback_rejected": true,
    "css_runtime_rejected": true,
    "multiple_color_sources_rejected": true,
    "override_order_rejected": true,
    "density_dpi_rejected": true
  },
  "nonclaims": ["no CSS cascade runtime", "no React runtime", "no Electron runtime", "no DOM style runtime", "no platform-native widgets"]
}`)
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
		"schema":                       "tetra.surface.morph.token-graph.v1",
		"namespace":                    "tetra.surface.morph.app",
		"version":                      "1",
		"hash":                         "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"source_of_truth":              "capsule",
		"explicit_imports":             true,
		"no_global_cascade":            true,
		"fixed_override_order":         []any{"base", "theme", "density", "variant", "state", "local"},
		"categories":                   []any{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"},
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
		map[string]any{"id": "color.bg", "category": "color", "kind": "rgba", "value": "#0b0f14ff", "source": "capsule", "hash": hash("1")},
		map[string]any{"id": "color.surface", "category": "color", "kind": "rgba", "value": "#181f26ff", "source": "capsule", "hash": hash("2")},
		map[string]any{"id": "color.surfaceAlpha", "category": "color", "kind": "rgba", "value": "#181f26da", "source": "capsule", "hash": hash("3")},
		map[string]any{"id": "color.accent", "category": "color", "kind": "rgba", "value": "#60aef4ff", "source": "capsule", "hash": hash("4")},
		map[string]any{"id": "color.muted", "category": "color", "kind": "rgba", "value": "#7e90a3ff", "source": "capsule", "hash": hash("5")},
		map[string]any{"id": "color.warning", "category": "color", "kind": "rgba", "value": "#f4cd5cff", "source": "capsule", "hash": hash("6")},
		map[string]any{"id": "space.3", "category": "space", "kind": "px", "value": "12", "source": "capsule", "hash": hash("7")},
		map[string]any{"id": "radius.sm", "category": "radius", "kind": "px", "value": "8", "source": "capsule", "hash": hash("8")},
		map[string]any{"id": "radius.md", "category": "radius", "kind": "px", "value": "10", "source": "capsule", "hash": hash("9")},
		map[string]any{"id": "radius.lg", "category": "radius", "kind": "px", "value": "18", "source": "capsule", "hash": hash("a")},
		map[string]any{"id": "border.subtle", "category": "border", "kind": "px", "value": "1", "source": "capsule", "hash": hash("b")},
		map[string]any{"id": "border.glass", "category": "border", "kind": "px", "value": "1", "source": "capsule", "hash": hash("c")},
		map[string]any{"id": "elevation.2", "category": "elevation", "kind": "shadow", "value": "0 3 10 72", "source": "capsule", "hash": hash("d")},
		map[string]any{"id": "elevation.3", "category": "elevation", "kind": "shadow", "value": "0 10 24 128", "source": "capsule", "hash": hash("e")},
		map[string]any{"id": "opacity.disabled", "category": "opacity", "kind": "alpha", "value": "128", "source": "capsule", "hash": hash("f")},
		map[string]any{"id": "type.label", "category": "typography", "kind": "font", "value": "Tetra UI 13 600 18", "source": "capsule", "hash": hash("1")},
		map[string]any{"id": "motion.fast", "category": "motion", "kind": "transition", "value": "120 ease.out", "source": "capsule", "hash": hash("2")},
		map[string]any{"id": "motion.soft", "category": "motion", "kind": "transition", "value": "180 ease.inOut", "source": "capsule", "hash": hash("3")},
		map[string]any{"id": "z.base", "category": "z", "kind": "layer", "value": "0", "source": "capsule", "hash": hash("4")},
		map[string]any{"id": "assets.gradient.vertical", "category": "assets", "kind": "gradient", "value": "vertical", "source": "capsule", "hash": hash("5")},
		map[string]any{"id": "assets.icon.fallback", "category": "assets", "kind": "icon", "value": "fallback", "source": "capsule", "hash": hash("6")},
		map[string]any{"id": "density.1x", "category": "density", "kind": "dpi", "value": "96/1000", "source": "capsule", "hash": hash("7")},
	}
}

func p07MorphMaterials() []any {
	return []any{
		map[string]any{"name": "surface.base", "paint_stack": []any{"fill", "border", "radius"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "surface.elevated", "paint_stack": []any{"fill", "border", "radius", "shadow"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "elevation.2", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "control.primary", "paint_stack": []any{"fill", "radius"}, "fill": "color.accent", "border": "", "radius": "radius.sm", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "translucent.panel", "paint_stack": []any{"fill", "border", "radius", "shadow", "overlay"}, "fill": "color.surfaceAlpha", "border": "border.glass", "radius": "radius.lg", "shadow": "elevation.3", "overlay": "assets.gradient.vertical", "unsupported_blur": false, "unsupported_blur_rejected": true},
	}
}

func p07DensityMappings() []any {
	return []any{
		map[string]any{"target": "headless", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"},
		map[string]any{"target": "linux-x64-real-window", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"},
		map[string]any{"target": "wasm32-web-browser-canvas", "token": "density.1x", "target_dpi": 96, "scale_milli": 1000, "rounding_policy": "integer-half-up-v1"},
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

func writeTokenGraphReferenceRoot(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "examples", "surface_morph_command_palette.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
