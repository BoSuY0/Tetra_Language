package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestValidateSurfaceTokenGraphAcceptsContractAndReport(t *testing.T) {
	root := t.TempDir()
	writeTokenGraphCLIFile(t, filepath.Join(root, "examples", "surface_morph_command_palette.tetra"), "import lib.core.morph as morph\nfunc main() -> Int:\n    return morph.schema_v1()\n")
	contractPath := filepath.Join(root, "surface-token-graph-contract.json")
	reportPath := filepath.Join(root, "surface-headless-morph.json")
	writeTokenGraphCLIFile(t, contractPath, string(validTokenGraphCLIContractRaw(t)))
	writeTokenGraphCLIFile(t, reportPath, string(validTokenGraphCLIReportRaw(t)))

	if err := validateSurfaceTokenGraph(tokenGraphCLIOptions{ContractPath: contractPath, ReportPath: reportPath, Root: root}); err != nil {
		t.Fatalf("validateSurfaceTokenGraph failed: %v", err)
	}
}

func TestValidateSurfaceTokenGraphRejectsMissingInputs(t *testing.T) {
	err := validateSurfaceTokenGraph(tokenGraphCLIOptions{})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "contract") {
		t.Fatalf("validateSurfaceTokenGraph err = %v, want contract path rejection", err)
	}
}

func validTokenGraphCLIContractRaw(t *testing.T) []byte {
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
	return raw
}

func validTokenGraphCLIReportRaw(t *testing.T) []byte {
	t.Helper()
	hash := func(seed string) string { return "sha256:" + strings.Repeat(seed, 64) }
	report := surface.Report{
		Schema: "tetra.surface.runtime.v1",
		Source: "examples/surface_morph_command_palette.tetra",
		Morph: &surface.MorphReport{
			Module:         "lib.core.morph",
			TokenGraphHash: hash("b"),
			Capsule: surface.MorphCapsuleReport{
				Namespace:       "tetra.surface.morph.app",
				ExplicitImports: true,
				NoGlobalCascade: true,
			},
			TokenGraph: &surface.MorphTokenGraphReport{
				Schema:                     "tetra.surface.morph.token-graph.v1",
				Namespace:                  "tetra.surface.morph.app",
				Version:                    "1",
				Hash:                       hash("b"),
				SourceOfTruth:              "capsule",
				ExplicitImports:            true,
				NoGlobalCascade:            true,
				FixedOverrideOrder:         []string{"base", "theme", "density", "variant", "state", "local"},
				Categories:                 []string{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"},
				Tokens:                     validTokenGraphCLITokens(),
				DensityDPI:                 validTokenGraphCLIDensityDPI(),
				Diagnostics:                validTokenGraphCLIDiagnostics(),
				AliasCycleRejected:         true,
				DuplicateSourceRejected:    true,
				RawLiteralsInAppCode:       false,
				UnresolvedFallbackRejected: true,
				FallbackToRandomDefault:    false,
			},
			Materials: validTokenGraphCLIMaterials(),
			AssetRefs: []surface.MorphAssetRefReport{
				{ID: "command.search", TintToken: "color.accent", FallbackID: "icon.fallback"},
			},
		},
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return raw
}

func validTokenGraphCLITokens() []surface.MorphTokenReport {
	hash := func(seed string) string { return "sha256:" + strings.Repeat(seed, 64) }
	return []surface.MorphTokenReport{
		{ID: "color.bg", Category: "color", Kind: "rgba", Value: "#0b0f14ff", Source: "capsule", Hash: hash("1")},
		{ID: "color.surface", Category: "color", Kind: "rgba", Value: "#181f26ff", Source: "capsule", Hash: hash("2")},
		{ID: "color.surfaceAlpha", Category: "color", Kind: "rgba", Value: "#181f26da", Source: "capsule", Hash: hash("3")},
		{ID: "color.accent", Category: "color", Kind: "rgba", Value: "#60aef4ff", Source: "capsule", Hash: hash("4")},
		{ID: "color.muted", Category: "color", Kind: "rgba", Value: "#7e90a3ff", Source: "capsule", Hash: hash("5")},
		{ID: "color.warning", Category: "color", Kind: "rgba", Value: "#f4cd5cff", Source: "capsule", Hash: hash("6")},
		{ID: "space.3", Category: "space", Kind: "px", Value: "12", Source: "capsule", Hash: hash("7")},
		{ID: "radius.sm", Category: "radius", Kind: "px", Value: "8", Source: "capsule", Hash: hash("8")},
		{ID: "radius.md", Category: "radius", Kind: "px", Value: "10", Source: "capsule", Hash: hash("9")},
		{ID: "radius.lg", Category: "radius", Kind: "px", Value: "18", Source: "capsule", Hash: hash("a")},
		{ID: "border.subtle", Category: "border", Kind: "px", Value: "1", Source: "capsule", Hash: hash("b")},
		{ID: "border.glass", Category: "border", Kind: "px", Value: "1", Source: "capsule", Hash: hash("c")},
		{ID: "elevation.2", Category: "elevation", Kind: "shadow", Value: "0 3 10 72", Source: "capsule", Hash: hash("d")},
		{ID: "elevation.3", Category: "elevation", Kind: "shadow", Value: "0 10 24 128", Source: "capsule", Hash: hash("e")},
		{ID: "opacity.disabled", Category: "opacity", Kind: "alpha", Value: "128", Source: "capsule", Hash: hash("f")},
		{ID: "type.label", Category: "typography", Kind: "font", Value: "Tetra UI 13 600 18", Source: "capsule", Hash: hash("1")},
		{ID: "motion.fast", Category: "motion", Kind: "transition", Value: "120 ease.out", Source: "capsule", Hash: hash("2")},
		{ID: "motion.soft", Category: "motion", Kind: "transition", Value: "180 ease.inOut", Source: "capsule", Hash: hash("3")},
		{ID: "z.base", Category: "z", Kind: "layer", Value: "0", Source: "capsule", Hash: hash("4")},
		{ID: "assets.gradient.vertical", Category: "assets", Kind: "gradient", Value: "vertical", Source: "capsule", Hash: hash("5")},
		{ID: "assets.icon.fallback", Category: "assets", Kind: "icon", Value: "fallback", Source: "capsule", Hash: hash("6")},
		{ID: "density.1x", Category: "density", Kind: "dpi", Value: "96/1000", Source: "capsule", Hash: hash("7")},
	}
}

func validTokenGraphCLIDensityDPI() []surface.MorphDensityDPIReport {
	return []surface.MorphDensityDPIReport{
		{Target: "headless", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
		{Target: "linux-x64-real-window", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
		{Target: "wasm32-web-browser-canvas", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
	}
}

func validTokenGraphCLIDiagnostics() surface.MorphTokenGraphDiagnosticsReport {
	return surface.MorphTokenGraphDiagnosticsReport{
		AliasCycleRejected:           true,
		MissingTokenRejected:         true,
		DuplicateSourceRejected:      true,
		RawLiteralRejected:           true,
		UnresolvedFallbackRejected:   true,
		CSSRuntimeRejected:           true,
		MultipleColorSourcesRejected: true,
		OverrideOrderRejected:        true,
		DensityDPIRejected:           true,
	}
}

func validTokenGraphCLIMaterials() []surface.MorphMaterialReport {
	return []surface.MorphMaterialReport{
		{Name: "surface.base", PaintStack: []string{"fill", "border", "radius"}, Fill: "color.surface", Border: "border.subtle", Radius: "radius.md", UnsupportedBlurRejected: true},
		{Name: "translucent.panel", PaintStack: []string{"fill", "border", "radius", "shadow", "overlay"}, Fill: "color.surfaceAlpha", Border: "border.glass", Radius: "radius.lg", Shadow: "elevation.3", Overlay: "assets.gradient.vertical", UnsupportedBlurRejected: true},
	}
}

func writeTokenGraphCLIFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
