package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

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
			map[string]any{"name": "control.action@1", "output": "Block", "slots": []any{"label", "icon"}, "inputs": []any{"text", "action", "variant"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
			map[string]any{"name": "field.text@1", "output": "Block", "slots": []any{"label", "control"}, "inputs": []any{"value", "on_text"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
			map[string]any{"name": "command.item@1", "output": "Block", "slots": []any{"icon", "title", "subtitle"}, "inputs": []any{"title", "subtitle", "icon", "selected"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
			map[string]any{"name": "region.panel@1", "output": "Block", "slots": []any{"header", "body", "actions"}, "inputs": []any{"title"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
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
		report.HostEvidence = HostEvidenceReport{Level: "deterministic-headless", Backend: "software-rgba", Framebuffer: true}
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
		t.Fatalf("expected linux-x64 Block system report without real-window frame presentation to fail")
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
				report.HostEvidence = HostEvidenceReport{Level: "wasm32-web-compiler-owned-loader", Backend: "node-surface-host", Framebuffer: true}
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
				report.BlockSystem.Frames[1].RepeatChecksum = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
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
				report.BlockSystem.Frames[0].GoldenChecksum = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
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
				report.Components[3].Type = "examples.surface_block_system.Button"
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
	report.Source = "examples/surface_block_system.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockSystemComponentsForTest()
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockTextComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockStateComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockMotionComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockAssetComponentsForTest())...)
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.PaintQualityLevel = "deterministic-software-paint-v1"
	report.PaintCacheBudgetBytes = 65536
	report.PaintUnsupportedBlur = false
	report.PaintLayers = blockPaintLayersForTest()
	report.PaintCommands = blockPaintCommandsForTest()
	report.VisualFeatures = []string{"fill", "gradient", "image_fill", "border", "radius", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
	report.Renderer = rendererReportForTest()
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "aspect", "spacing", "alignment", "z-order", "clipping", "resize", "density", "stable-rounding"}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutDensity = blockLayoutDensityForTest()
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateResolverOrder = []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
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
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "SubmitBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 2, Component: "ResetBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 3, Component: "BlockSystemApp", Field: "reading_order_checked", Before: "false", After: "true", Cause: "block_graph"},
		{Order: 4, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
		{Order: 5, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
	}
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, blockSystemReadinessTransitionsForTest())
	report.BlockSystem = &BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "deterministic-headless-block-system-v1",
		Source:       report.Source,
		Renderer:     "software-rgba-headless",
		GoldenSet:    "surface-block-system-golden-v1",
		FrameCount:   3,
		GoldenHash:   "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Frames: []BlockSystemFrameReport{
			{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
			{Order: 2, Label: "focused", Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", RepeatChecksum: "2222222222222222222222222222222222222222222222222222222222222222", GoldenChecksum: "2222222222222222222222222222222222222222222222222222222222222222", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
			{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
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
		"schema":            "tetra.surface.morph.v1",
		"quality_level":     "deterministic-headless-morph-capsule-v1",
		"source":            "examples/surface_morph_command_palette.tetra",
		"module":            "lib.core.morph",
		"surface_scope":     "surface-morph-experimental-linux-web",
		"experimental":      true,
		"production_claim":  false,
		"git_head":          "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"git_dirty":         true,
		"capsule_hash":      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"token_graph_hash":  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"capsule":           validMorphCapsuleMap(),
		"token_graph":       validMorphTokenGraphMap(),
		"materials":         validMorphMaterials(),
		"layout_modes":      []any{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"},
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
		"nonclaims":         []any{"DOM runtime absent", "React runtime absent", "Electron claim absent", "platform-native widgets absent", "full screen-reader production absent", "CSS cascade absent"},
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
		"schema":                       "tetra.surface.morph.token-graph.v1",
		"namespace":                    "tetra.surface.morph.app",
		"version":                      "1",
		"hash":                         "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"categories":                   []any{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"},
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
		map[string]any{"id": "color.bg", "category": "color", "kind": "rgba", "value": "#0b0f14ff", "source": "capsule", "hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		map[string]any{"id": "space.3", "category": "space", "kind": "px", "value": "12", "source": "capsule", "hash": "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		map[string]any{"id": "radius.md", "category": "radius", "kind": "px", "value": "10", "source": "capsule", "hash": "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		map[string]any{"id": "type.label", "category": "typography", "kind": "font", "value": "Tetra UI 13 600 18", "source": "capsule", "hash": "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		map[string]any{"id": "motion.fast", "category": "motion", "kind": "transition", "value": "120 ease.out", "source": "capsule", "hash": "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
	}
}
func validMorphMaterials() []any {
	return []any{
		map[string]any{"name": "surface.base", "paint_stack": []any{"fill", "border", "radius"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "surface.elevated", "paint_stack": []any{"fill", "border", "radius", "shadow"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "elevation.2", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "control.primary", "paint_stack": []any{"fill", "radius"}, "fill": "color.accent", "border": "", "radius": "radius.sm", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "translucent.panel", "paint_stack": []any{"fill", "border", "radius", "shadow", "overlay"}, "fill": "color.surfaceAlpha", "border": "border.glass", "radius": "radius.lg", "shadow": "elevation.3", "overlay": "gradient.vertical", "unsupported_blur": false, "unsupported_blur_rejected": true},
	}
}
func validMorphAssetRefs() []any {
	return []any{
		map[string]any{"id": "project.new", "kind": "icon", "sha256": "sha256:6666666666666666666666666666666666666666666666666666666666666666", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.accent"},
		map[string]any{"id": "command.search", "kind": "icon", "sha256": "sha256:7777777777777777777777777777777777777777777777777777777777777777", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.muted"},
		map[string]any{"id": "status.warning", "kind": "icon", "sha256": "sha256:8888888888888888888888888888888888888888888888888888888888888888", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.warning"},
	}
}
func validMorphAffordances() []any {
	return []any{
		map[string]any{"name": "action", "role": "button", "focusable": true, "action": "activate", "input": "", "projects_accessibility": true},
		map[string]any{"name": "field.text", "role": "textbox", "focusable": true, "action": "edit", "input": "editable_text", "projects_accessibility": true},
		map[string]any{"name": "toggle", "role": "checkbox", "focusable": true, "action": "toggle", "input": "toggle", "projects_accessibility": true},
		map[string]any{"name": "navigation", "role": "navigation", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
		map[string]any{"name": "region", "role": "region", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
		map[string]any{"name": "overlay", "role": "dialog", "focusable": true, "action": "dismiss", "input": "focus_trap", "projects_accessibility": true},
		map[string]any{"name": "status", "role": "status", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
	}
}
func validMorphStateLenses() []any {
	return []any{
		map[string]any{"selector": "hover", "property": "paint.overlay", "deterministic": true},
		map[string]any{"selector": "pressed", "property": "transform.scale", "deterministic": true},
		map[string]any{"selector": "focusVisible", "property": "paint.outline", "deterministic": true},
		map[string]any{"selector": "selected", "property": "accessibility.selected", "deterministic": true},
		map[string]any{"selector": "disabled", "property": "input.disabled", "deterministic": true},
		map[string]any{"selector": "error", "property": "paint.outline_color", "deterministic": true},
		map[string]any{"selector": "loading", "property": "text.content", "deterministic": true},
	}
}
func validMorphMotionPresets() []any {
	return []any{
		map[string]any{"name": "motion.fast", "duration_ms": 120, "curve": "ease.out", "properties": []any{"fill", "opacity", "transform"}, "reduced_motion": true, "deterministic_time": true},
		map[string]any{"name": "motion.soft", "duration_ms": 180, "curve": "ease.inOut", "properties": []any{"fill", "opacity", "transform"}, "reduced_motion": true, "deterministic_time": true},
	}
}
func validMorphRecipes() []any {
	return []any{
		map[string]any{"name": "control.action@1", "output": "Block", "slots": []any{"label", "icon"}, "inputs": []any{"text", "action", "variant"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "field.text@1", "output": "Block", "slots": []any{"label", "control"}, "inputs": []any{"value", "on_text"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "command.item@1", "output": "Block", "slots": []any{"icon", "title", "subtitle"}, "inputs": []any{"title", "subtitle", "icon", "selected"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "region.panel@1", "output": "Block", "slots": []any{"header", "body", "actions"}, "inputs": []any{"title"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "form.field@1", "output": "Block", "slots": []any{"label", "control", "hint", "error"}, "inputs": []any{"label", "value", "validation"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "nav.item@1", "output": "Block", "slots": []any{"icon", "label", "badge"}, "inputs": []any{"label", "destination", "selected"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "metric.tile@1", "output": "Block", "slots": []any{"label", "value", "trend"}, "inputs": []any{"label", "value", "trend"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "dialog.panel@1", "output": "Block", "slots": []any{"title", "body", "actions"}, "inputs": []any{"title", "open", "dismiss"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "toast.notification@1", "output": "Block", "slots": []any{"icon", "message", "action"}, "inputs": []any{"message", "severity", "timeout"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "tab.item@1", "output": "Block", "slots": []any{"label", "indicator"}, "inputs": []any{"label", "selected", "target"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
		map[string]any{"name": "list.row@1", "output": "Block", "slots": []any{"leading", "title", "meta", "action"}, "inputs": []any{"title", "subtitle", "selected"}, "expands_to_block_graph": true, "hidden_app_state": false, "platform_widgets": false, "core_primitive_promotion": false},
	}
}
func validMorphRecipeExpansions() []any {
	return []any{
		map[string]any{"recipe": "control.action@1", "block_ids": []any{4}, "slot_bindings": []any{"label", "icon"}, "variant": "primary", "reported": true},
		map[string]any{"recipe": "field.text@1", "block_ids": []any{3}, "slot_bindings": []any{"label", "control"}, "variant": "default", "reported": true},
		map[string]any{"recipe": "command.item@1", "block_ids": []any{4, 5}, "slot_bindings": []any{"icon", "title", "subtitle"}, "variant": "selected", "reported": true},
		map[string]any{"recipe": "region.panel@1", "block_ids": []any{2}, "slot_bindings": []any{"header", "body", "actions"}, "variant": "elevated", "reported": true},
		map[string]any{"recipe": "form.field@1", "block_ids": []any{3, 4}, "slot_bindings": []any{"label", "control", "hint", "error"}, "variant": "validated", "reported": true},
		map[string]any{"recipe": "nav.item@1", "block_ids": []any{5}, "slot_bindings": []any{"icon", "label", "badge"}, "variant": "selected", "reported": true},
		map[string]any{"recipe": "metric.tile@1", "block_ids": []any{2, 5}, "slot_bindings": []any{"label", "value", "trend"}, "variant": "compact", "reported": true},
		map[string]any{"recipe": "dialog.panel@1", "block_ids": []any{2, 4}, "slot_bindings": []any{"title", "body", "actions"}, "variant": "modal", "reported": true},
		map[string]any{"recipe": "toast.notification@1", "block_ids": []any{5}, "slot_bindings": []any{"icon", "message", "action"}, "variant": "warning", "reported": true},
		map[string]any{"recipe": "tab.item@1", "block_ids": []any{4}, "slot_bindings": []any{"label", "indicator"}, "variant": "active", "reported": true},
		map[string]any{"recipe": "list.row@1", "block_ids": []any{4, 5}, "slot_bindings": []any{"leading", "title", "meta", "action"}, "variant": "interactive", "reported": true},
	}
}
func validMorphRecipeApps() []any {
	return []any{
		map[string]any{"source": "examples/surface_morph_command_palette.tetra", "module": "examples.surface_morph_command_palette", "recipes": []any{"control.action@1", "field.text@1", "command.item@1", "region.panel@1"}, "expands_to_block_graph": true, "block_count": 7, "accessibility_projection": true, "hidden_app_state": false, "react_runtime": false, "electron_runtime": false, "dom_runtime": false, "platform_widgets": false, "output_primitives": []any{"Block"}},
		map[string]any{"source": "examples/surface_morph_project_dashboard.tetra", "module": "examples.surface_morph_project_dashboard", "recipes": []any{"region.panel@1", "metric.tile@1", "list.row@1", "toast.notification@1"}, "expands_to_block_graph": true, "block_count": 7, "accessibility_projection": true, "hidden_app_state": false, "react_runtime": false, "electron_runtime": false, "dom_runtime": false, "platform_widgets": false, "output_primitives": []any{"Block"}},
		map[string]any{"source": "examples/surface_morph_settings.tetra", "module": "examples.surface_morph_settings", "recipes": []any{"form.field@1", "field.text@1", "tab.item@1", "control.action@1"}, "expands_to_block_graph": true, "block_count": 7, "accessibility_projection": true, "hidden_app_state": false, "react_runtime": false, "electron_runtime": false, "dom_runtime": false, "platform_widgets": false, "output_primitives": []any{"Block"}},
		map[string]any{"source": "examples/surface_morph_editor_shell.tetra", "module": "examples.surface_morph_editor_shell", "recipes": []any{"nav.item@1", "tab.item@1", "command.item@1", "region.panel@1"}, "expands_to_block_graph": true, "block_count": 7, "accessibility_projection": true, "hidden_app_state": false, "react_runtime": false, "electron_runtime": false, "dom_runtime": false, "platform_widgets": false, "output_primitives": []any{"Block"}},
		map[string]any{"source": "examples/surface_morph_glass_panel.tetra", "module": "examples.surface_morph_glass_panel", "recipes": []any{"dialog.panel@1", "toast.notification@1", "control.action@1", "region.panel@1"}, "expands_to_block_graph": true, "block_count": 7, "accessibility_projection": true, "hidden_app_state": false, "react_runtime": false, "electron_runtime": false, "dom_runtime": false, "platform_widgets": false, "output_primitives": []any{"Block"}},
	}
}
func validMorphAccessibilityProjectionMap() map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.morph.accessibility-projection.v1",
		"derived_from_block_graph": true,
		"safety_overrides_win":     true,
		"snapshot_evidence":        true,
		"required_fields":          []any{"role", "name", "description", "action", "state", "bounds", "focus_order", "reading_order", "labelled_by", "label_for"},
		"roles":                    []any{"button", "textbox", "checkbox", "navigation", "region", "dialog", "status"},
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
		"expanded_recipe_count":    11,
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
func validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
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
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system-real-window-probe", Ran: true, Pass: true, ExitCode: intPtrForTest(42), ExpectedExitCode: intPtrForTest(42)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 1, ForbiddenPaths: nil, Pass: true}
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
		{Order: 5, Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", Presented: true},
	}
	report.BlockSystem.QualityLevel = "linux-x64-real-window-block-system-v1"
	report.BlockSystem.Renderer = "wayland-shm-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-linux-x64-real-window-v1"
	report.BlockSystem.FrameCount = 4
	report.BlockSystem.GoldenHash = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 2, Label: "focused", Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", RepeatChecksum: "2222222222222222222222222222222222222222222222222222222222222222", GoldenChecksum: "2222222222222222222222222222222222222222222222222222222222222222", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 5, Label: "real-window-focused", Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", RepeatChecksum: "5555555555555555555555555555555555555555555555555555555555555555", GoldenChecksum: "5555555555555555555555555555555555555555555555555555555555555555", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
		{Kind: "close", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 5, BufferSlots: []int{1, 0, 0, 0, 0, 400, 240, 5, 0}, BeforeState: map[string]string{"BlockSystemApp.closed": "false"}, AfterState: map[string]string{"BlockSystemApp.closed": "true"}},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, []StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
		{Component: "BlockSystemApp", Field: "closed", Before: "false", After: "true", Cause: "close"},
	})
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
func validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
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
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=block-system wasm=/tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0), ExpectedExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium Block-system fixture", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom <surface-browser-canvas-file-runner scenario=block-system>", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-block-system.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 3, ForbiddenPaths: nil, Pass: true}
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
		{Order: 5, Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", Presented: true},
	}
	report.BlockSystem.QualityLevel = "wasm32-web-browser-canvas-block-system-v1"
	report.BlockSystem.Renderer = "browser-canvas-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-wasm32-web-browser-canvas-v1"
	report.BlockSystem.FrameCount = 3
	report.BlockSystem.GoldenHash = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 5, Label: "browser-canvas-focused", Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", RepeatChecksum: "5555555555555555555555555555555555555555555555555555555555555555", GoldenChecksum: "5555555555555555555555555555555555555555555555555555555555555555", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, []StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
	})
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
		{ID: "BlockSystemApp", Type: "examples.surface_block_system.BlockSystemApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "quality": "deterministic-headless-block-system-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_system.PanelBlock", Parent: "BlockSystemApp", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"paint_layers": "5"}},
		{ID: "LabelBlock", Type: "examples.surface_block_system.LabelBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "4", "label_for": "4"}},
		{ID: "SubmitBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "true", "action": "submit"}},
		{ID: "ResetBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false", "action": "reset"}},
		{ID: "BlockLayoutApp", Type: "examples.surface_block_system.BlockLayoutApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"width": "480", "layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_system.ScrollBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"scroll_y": "32"}},
	}
}
func blockSystemEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"SubmitBlock.focused": "false"}, AfterState: map[string]string{"SubmitBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"SubmitBlock.value_len": "0"}, AfterState: map[string]string{"SubmitBlock.value_len": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 13, 320, 200, 2, 0}, BeforeState: map[string]string{"SubmitBlock.pressed": "false"}, AfterState: map[string]string{"SubmitBlock.pressed": "true"}},
		{Order: 4, Kind: "scroll", TargetComponent: "ScrollBlock", DispatchPath: []string{"BlockLayoutApp", "ScrollBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{7, 0, 0, 0, 0, 320, 200, 3, 0}, BeforeState: map[string]string{"ScrollBlock.scroll_y": "0"}, AfterState: map[string]string{"ScrollBlock.scroll_y": "32"}},
	}
}
func retargetBlockSystemComponentsForTest(components []ComponentReport) []ComponentReport {
	retargeted := make([]ComponentReport, len(components))
	for i, component := range components {
		component.Type = "examples.surface_block_system." + typeBaseName(component.Type)
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
func appendEventReportsWithNextOrder(events []EventReport, additions ...[]EventReport) []EventReport {
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
func appendStateTransitionReportsWithNextOrder(transitions []StateTransitionReport, additions ...[]StateTransitionReport) []StateTransitionReport {
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
		{Order: 1, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
		{Order: 2, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
		{Order: 3, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
		{Order: 4, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
		{Order: 5, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
		{Order: 6, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
		{Order: 7, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
		{Order: 8, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
		{Order: 9, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
		{Order: 10, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
		{Order: 11, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
		{Order: 12, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
		{Order: 13, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
		{Order: 14, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
		{Order: 15, Component: "IconBlock", Field: "tint", Before: "#ffffffff", After: "#60aef4ff", Cause: "asset tint"},
		{Order: 16, Component: "ImageBlock", Field: "scale", Before: "1x", After: "2x", Cause: "asset scale"},
		{Order: 17, Component: "MissingAssetBlock", Field: "fallback", Before: "missing", After: "fallback-raster", Cause: "missing asset"},
	}
}
func blockSystemCasesForTest() []CaseReport {
	return []CaseReport{
		{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
		{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
		{Name: "block compositor dirty rect invalidation cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block renderer opacity transform clipped child", Kind: "positive", Ran: true, Pass: true},
		{Name: "block renderer gpu production claim rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "gpu production"},
		{Name: "block renderer unsupported backdrop blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "backdrop blur"},
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
		{Name: "block layout aspect density stable rounding", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
		{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
		{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
		{Name: "block asset deterministic manifest hashes", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset missing fallback diagnostic", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network asset rejected"},
		{Name: "block accessibility tree derived from block graph", Kind: "positive", Ran: true, Pass: true},
		{Name: "block accessibility focusable actionable name required", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing accessible name"},
		{Name: "block accessibility label relationship mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "label relationship mismatch"},
		{Name: "block accessibility reading order graph mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "reading order mismatch"},
		{Name: "block accessibility screen-reader claim without platform proof rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "screen reader proof required"},
		{Name: "block accessibility platform claim scoped metadata only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system headless golden checksums", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system deterministic repeat checksum", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system missing frame checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "frame checksum required"},
		{Name: "block system nondeterministic checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "repeat checksum mismatch"},
		{Name: "block system missing paint evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "paint evidence required"},
		{Name: "block system missing layout evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "layout evidence required"},
		{Name: "block system missing accessibility evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "accessibility evidence required"},
		{Name: "block system bounded memory budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system stress render loop budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system performance nonclaim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "Electron comparison benchmark not claimed"},
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
	cases = append(cases,
		CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 real-window frame presentation", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 native input state transition", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 real-window checksum", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system missing real-window presentation rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "real-window presentation required"},
		CaseReport{Name: "block system missing native input state transition rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "native input required"},
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
	cases = append(cases,
		CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas frame readback", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas native input state transition", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas checksum", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system browser-canvas node runtime substitution rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "browser evidence required"},
		CaseReport{Name: "block system browser-canvas missing RGBA readback rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "RGBA readback required"},
		CaseReport{Name: "block system browser-canvas script sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "script artifact rejected"},
		CaseReport{Name: "block system browser-canvas html visual sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "html artifact rejected"},
	)
	return cases
}
func blockMemoryBudgetForTest(report *Report) *BlockMemoryBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotals(report.Frames)
	cacheUsedBytes := len(report.PaintCommands)*2048 + 4096 + report.BlockAssetCache.UsedBytes
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
		TotalCacheBudgetBytes:    report.PaintCacheBudgetBytes + report.TextCacheBudgetBytes + report.BlockAssetCache.BudgetBytes,
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
