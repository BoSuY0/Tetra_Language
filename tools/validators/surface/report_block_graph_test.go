package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

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
				report.PaintCommands[0], report.PaintCommands[1] = report.PaintCommands[1], report.PaintCommands[0]
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
				report.LayoutFeatures = removeString(removeString(report.LayoutFeatures, "density"), "stable-rounding")
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
				report.BlockStateResolverOrder = []string{"base", "hover", "variant", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
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
					if report.BlockStateResolutions[i].Selector == "pressed" && report.BlockStateResolutions[i].Property == "layout.scale" {
						report.BlockStateResolutions[i].Applied = false
						report.BlockStateResolutions[i].After = report.BlockStateResolutions[i].Before
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
	if err := json.Unmarshal(validHeadlessComponentTreeSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode component tree report: %v", err)
	}
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.Cases = append(report.Cases,
		CaseReport{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		CaseReport{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		CaseReport{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
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
	report.VisualFeatures = []string{"fill", "gradient", "image_fill", "border", "radius", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
	report.Renderer = rendererReportForTest()
	report.Cases = append(report.Cases,
		CaseReport{Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
		CaseReport{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block compositor dirty rect invalidation cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block renderer opacity transform clipped child", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block renderer gpu production claim rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "gpu production"},
		CaseReport{Name: "block renderer unsupported backdrop blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "backdrop blur"},
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
	report.Source = "examples/surface_block_text.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_text.tetra -o /tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-text", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockTextComponentsForTest()
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.Events = blockTextEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "BlockTextApp", Field: "focused_id", Before: "0", After: "3", Cause: "mouse_up"},
		{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
		{Order: 3, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
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
	report.Source = "examples/surface_block_layout.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_layout.tetra -o /tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-layout", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockLayoutComponentsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "aspect", "spacing", "alignment", "z-order", "clipping", "resize", "density", "stable-rounding"}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutDensity = blockLayoutDensityForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 480, Height: 260, Stride: 1920, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "RowBlock", Field: "pressed", Before: "false", After: "true", Cause: "mouse_up"},
		{Order: 2, Component: "RowBlock", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
		{Order: 3, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
		{Order: 4, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
	}
	report.Events = []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "RowBlock", DispatchPath: []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"}, Handled: true, Pass: true, X: 32, Y: 32, Width: 320, Height: 200, BufferSlots: []int{5, 32, 32, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"RowBlock.pressed": "false"}, AfterState: map[string]string{"RowBlock.pressed": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "RowBlock", DispatchPath: []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"RowBlock.text_len_seen": "0"}, AfterState: map[string]string{"RowBlock.text_len_seen": "2"}},
		{Order: 3, Kind: "resize", TargetComponent: "BlockLayoutApp", DispatchPath: []string{"BlockLayoutApp"}, Handled: true, Pass: true, Width: 480, Height: 260, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 0, 480, 260, 2, 0}, BeforeState: map[string]string{"BlockLayoutApp.width": "320"}, AfterState: map[string]string{"BlockLayoutApp.width": "480"}},
		{Order: 4, Kind: "scroll", TargetComponent: "ScrollBlock", DispatchPath: []string{"BlockLayoutApp", "ScrollBlock"}, Handled: true, Pass: true, X: 260, Y: 80, Width: 480, Height: 260, TimestampMS: 3, BufferSlots: []int{7, 260, 80, 0, 0, 480, 260, 3, 0}, BeforeState: map[string]string{"ScrollBlock.scroll_y": "0"}, AfterState: map[string]string{"ScrollBlock.scroll_y": "32"}},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout aspect density stable rounding", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
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
	report.Source = "examples/surface_block_events.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_events.tetra -o /tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-events", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockEventComponentsForTest()
	report.BlockGraph = blockEventGraphReportForTest(report.Source)
	report.BlockEventQualityLevel = "deterministic-block-events-v1"
	report.BlockEventPolicy = "capture-bubble-direct-v1"
	report.BlockEventUnsupportedDragDrop = false
	report.BlockEventKinds = []string{"pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame"}
	report.BlockEventRoutes = blockEventRoutesForTest()
	report.BlockFocusTransitions = blockFocusTransitionsForTest()
	report.Events = blockEventRuntimeEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "BlockEventApp", Field: "focused_id", Before: "0", After: "4", Cause: "click"},
		{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
		{Order: 3, Component: "BlockEventApp", Field: "focused_id", Before: "4", After: "6", Cause: "tab"},
		{Order: 4, Component: "BlockEventApp", Field: "focused_id", Before: "6", After: "4", Cause: "tab"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		CaseReport{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		CaseReport{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event nested hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event capture bubble direct policy", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event disabled click rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "disabled Block"},
		CaseReport{Name: "block event text input focused only", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block focus tab order graph-derived", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event no complex drag claim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "drag-and-drop nonclaim"},
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
	report.Source = "examples/surface_block_states.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_states.tetra -o /tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-states", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockStateComponentsForTest()
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateResolverOrder = []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateSelectors = blockStateSelectorsForTest()
	report.BlockStateResolutions = blockStateResolutionsForTest()
	report.Events = blockStateEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
		{Order: 2, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
		{Order: 3, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
		{Order: 4, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
		{Order: 5, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
		{Order: 6, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
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
	report.Source = "examples/surface_block_motion.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_motion.tetra -o /tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-motion", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockMotionComponentsForTest()
	report.MotionQualityLevel = "deterministic-block-motion-v1"
	report.MotionClock = "deterministic-test-clock-v1"
	report.MotionFrameBudget = 4
	report.MotionUnsupportedCSSAnimations = false
	report.MotionFrames = blockMotionFramesForTest()
	report.Events = blockMotionEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
		{Order: 2, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
		{Order: 3, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
		{Order: 4, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
		{Order: 5, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
		{Order: 6, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
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
		{ID: "BlockTextApp", Type: "examples.surface_block_text.BlockTextApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "3", "text_quality": "deterministic-fallback-text-v1"}},
		{ID: "TextBlock", Type: "examples.surface_block_text.TextSurfaceBlock", Parent: "BlockTextApp", Bounds: RectReport{X: 12, Y: 10, W: 96, H: 40}, Abilities: abilities, State: map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"}},
		{ID: "InputBlock", Type: "examples.surface_block_text.EditableTextBlock", Parent: "BlockTextApp", Bounds: RectReport{X: 12, Y: 58, W: 144, H: 36}, Abilities: abilities, State: map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"}},
	}
}
func blockEventComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockEventApp", Type: "examples.surface_block_events.BlockEventApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "event_quality": "deterministic-block-events-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_events.PanelBlock", Parent: "BlockEventApp", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"role": "panel"}},
		{ID: "LabelBlock", Type: "examples.surface_block_events.LabelBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "10"}},
		{ID: "InputBlock", Type: "examples.surface_block_events.InputBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"editable": "true", "focused": "true", "buffer": "OK"}},
		{ID: "DisabledBlock", Type: "examples.surface_block_events.DisabledBlock", Parent: "PanelBlock", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"disabled": "true"}},
		{ID: "ActionBlock", Type: "examples.surface_block_events.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 120, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false"}},
	}
}
func blockStateComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state"}
	return []ComponentReport{
		{ID: "BlockStateApp", Type: "examples.surface_block_states.BlockStateApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"state_quality": "deterministic-block-state-resolver-v1"}},
		{ID: "StateBlock", Type: "examples.surface_block_states.StateBlock", Parent: "BlockStateApp", Bounds: RectReport{X: 24, Y: 40, W: 168, H: 56}, Abilities: abilities, State: map[string]string{"selector_flags": "127", "variant": "2", "disabled": "true", "error": "true", "loading": "true"}},
		{ID: "StatusBlock", Type: "examples.surface_block_states.StatusBlock", Parent: "BlockStateApp", Bounds: RectReport{X: 24, Y: 112, W: 168, H: 32}, Abilities: abilities, State: map[string]string{"selected": "true", "focused": "true"}},
	}
}
func blockMotionComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state", "motion"}
	return []ComponentReport{
		{ID: "BlockMotionApp", Type: "examples.surface_block_motion.BlockMotionApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"motion_quality": "deterministic-block-motion-v1"}},
		{ID: "MotionBlock", Type: "examples.surface_block_motion.MotionBlock", Parent: "BlockMotionApp", Bounds: RectReport{X: 24, Y: 44, W: 176, H: 64}, Abilities: abilities, State: map[string]string{"opacity": "200", "scale": "108", "translate_x": "12", "complete": "true"}},
	}
}
func blockLayoutComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockLayoutApp", Type: "examples.surface_block_layout.BlockLayoutApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ColumnBlock", Type: "examples.surface_block_layout.ColumnBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 12, Y: 12, W: 296, H: 176}, Abilities: abilities, State: map[string]string{"mode": "column", "gap": "8"}},
		{ID: "RowBlock", Type: "examples.surface_block_layout.RowBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 24, Y: 24, W: 272, H: 48}, Abilities: abilities, State: map[string]string{"mode": "row", "gap": "6"}},
		{ID: "GridBlock", Type: "examples.surface_block_layout.GridBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 24, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "grid", "columns": "2"}},
		{ID: "DockBlock", Type: "examples.surface_block_layout.DockBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 164, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "dock"}},
		{ID: "OverlayBlock", Type: "examples.surface_block_layout.OverlayBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 220, Y: 20, W: 72, H: 40}, Abilities: abilities, State: map[string]string{"mode": "overlay", "z": "4"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_layout.ScrollBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"mode": "scroll", "clipped": "true"}},
	}
}
func blockMotionFramesForTest() []MotionFrameReport {
	return []MotionFrameReport{
		{Order: 1, BlockID: 2, Trigger: "hover", TimestampMS: 0, DurationMS: 120, DelayMS: 0, Progress: 0, Easing: "linear", Opacity: 80, Color: "#203040ff", TranslateX: 0, TranslateY: 0, Scale: 100, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, BlockID: 2, Trigger: "hover", TimestampMS: 60, DurationMS: 120, DelayMS: 0, Progress: 500, Easing: "linear", Opacity: 140, Color: "#407094ff", TranslateX: 6, TranslateY: 0, Scale: 104, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, BlockID: 2, Trigger: "hover", TimestampMS: 120, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: false, Scheduled: false, Settled: true, Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, BlockID: 2, Trigger: "reduced_motion", TimestampMS: 121, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: true, Scheduled: false, Settled: true, Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
	}
}
func blockMotionEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, X: 48, Y: 72, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 48, 72, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"MotionBlock.hovered": "false"}, AfterState: map[string]string{"MotionBlock.hovered": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"MotionBlock.buffer": ""}, AfterState: map[string]string{"MotionBlock.buffer": "OK"}},
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
		{Order: 1, BlockID: 2, Selector: "hover", ResolverStep: "hover", Property: "paint.fill", Before: "#20262eff", After: "#2d9bf0ff", Applied: true},
		{Order: 2, BlockID: 2, Selector: "pressed", ResolverStep: "pressed", Property: "layout.scale", Before: "100", After: "97", Applied: true},
		{Order: 3, BlockID: 2, Selector: "focused", ResolverStep: "focused", Property: "paint.outline", Before: "none", After: "focus-ring", Applied: true},
		{Order: 4, BlockID: 2, Selector: "selected", ResolverStep: "selected", Property: "accessibility.selected", Before: "false", After: "true", Applied: true},
		{Order: 5, BlockID: 2, Selector: "disabled", ResolverStep: "disabled", Property: "input.disabled", Before: "false", After: "true", Applied: true},
		{Order: 6, BlockID: 2, Selector: "disabled", ResolverStep: "disabled", Property: "text.opacity", Before: "255", After: "112", Applied: true},
		{Order: 7, BlockID: 2, Selector: "error", ResolverStep: "error", Property: "paint.outline_color", Before: "#7aa2f7ff", After: "#ff5f57ff", Applied: true},
		{Order: 8, BlockID: 2, Selector: "loading", ResolverStep: "loading", Property: "text.content", Before: "Run", After: "Loading", Applied: true},
		{Order: 9, BlockID: 2, Selector: "motion", ResolverStep: "motion", Property: "motion.transition_ms", Before: "0", After: "120", Applied: true},
	}
}
func blockStateEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 40, 56, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"StateBlock.selected": "false"}, AfterState: map[string]string{"StateBlock.selected": "true"}},
		{Order: 2, Kind: "mouse_move", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 1, BufferSlots: []int{2, 40, 56, 0, 0, 320, 200, 1, 0}, BeforeState: map[string]string{"StateBlock.hovered": "false"}, AfterState: map[string]string{"StateBlock.hovered": "true"}},
		{Order: 3, Kind: "mouse_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{4, 40, 56, 1, 0, 320, 200, 2, 0}, BeforeState: map[string]string{"StateBlock.pressed": "false"}, AfterState: map[string]string{"StateBlock.pressed": "true"}},
		{Order: 4, Kind: "text_input", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 3, 2}, BeforeState: map[string]string{"StateBlock.buffer": ""}, AfterState: map[string]string{"StateBlock.buffer": "OK"}},
		{Order: 5, Kind: "key_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 4, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 4, 0}, BeforeState: map[string]string{"StateBlock.focused": "false"}, AfterState: map[string]string{"StateBlock.focused": "true"}},
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
			{ID: 1, Name: "BlockEventApp", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 4, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "InputBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "textbox", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "DisabledBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "button", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}},
			{ID: 6, Name: "ActionBlock", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 24, Y: 120, W: 120, H: 44}},
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
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 4, X: 40, Y: 80, Path: []int{1, 2, 4}},
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
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
		{ID: "root-column", BlockID: 1, Mode: "column", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: SizeReport{W: 320, H: 200}, Max: SizeReport{W: 480, H: 260}, Padding: 12, Margin: 0, Gap: 8, Align: "stretch", Justify: "start", Overflow: "clip", ZIndex: 0, Clip: true},
		{ID: "row-fill", BlockID: 3, Mode: "row", WidthPolicy: "fill", HeightPolicy: "fixed", Min: SizeReport{W: 160, H: 40}, Max: SizeReport{W: 296, H: 64}, Padding: 6, Margin: 0, Gap: 6, Align: "center", Justify: "space-between", Overflow: "visible", ZIndex: 1, Clip: false},
		{ID: "text-fit", BlockID: 8, Mode: "absolute", WidthPolicy: "fit", HeightPolicy: "fit", Min: SizeReport{W: 32, H: 18}, Max: SizeReport{W: 160, H: 40}, Padding: 4, Margin: 0, Gap: 0, Align: "start", Justify: "start", Overflow: "clip", ZIndex: 2, Clip: true},
		{ID: "overlay-z", BlockID: 6, Mode: "overlay", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: SizeReport{W: 72, H: 40}, Max: SizeReport{W: 72, H: 40}, Padding: 0, Margin: 0, Gap: 0, Align: "end", Justify: "start", Overflow: "visible", ZIndex: 4, Clip: false},
		{ID: "aspect-fit", BlockID: 9, Mode: "absolute", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: SizeReport{W: 96, H: 54}, Max: SizeReport{W: 96, H: 54}, Padding: 0, Margin: 0, Gap: 0, Align: "start", Justify: "start", Overflow: "clip", ZIndex: 2, Clip: true},
	}
}
func blockLayoutPassesForTest() []BlockLayoutPassReport {
	return []BlockLayoutPassReport{
		{Order: 1, ParentID: 0, BlockID: 1, Mode: "column", Input: RectReport{X: 0, Y: 0, W: 320, H: 200}, Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: true, ZIndex: 0, Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, ParentID: 1, BlockID: 2, Mode: "stack", Input: RectReport{X: 12, Y: 12, W: 296, H: 176}, Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: false, ZIndex: 0, Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, ParentID: 2, BlockID: 3, Mode: "row", Input: RectReport{X: 24, Y: 24, W: 272, H: 48}, Resolved: RectReport{X: 24, Y: 24, W: 272, H: 48}, Measured: SizeReport{W: 272, H: 48}, Pass: "nested", Resize: false, Clip: false, ZIndex: 1, Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, ParentID: 2, BlockID: 4, Mode: "grid", Input: RectReport{X: 24, Y: 80, W: 132, H: 72}, Resolved: RectReport{X: 24, Y: 80, W: 63, H: 34}, Measured: SizeReport{W: 63, H: 34}, Pass: "grid-cell", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Order: 5, ParentID: 2, BlockID: 5, Mode: "dock", Input: RectReport{X: 164, Y: 80, W: 132, H: 72}, Resolved: RectReport{X: 164, Y: 80, W: 132, H: 24}, Measured: SizeReport{W: 132, H: 24}, Pass: "dock-top", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
		{Order: 6, ParentID: 1, BlockID: 6, Mode: "overlay", Input: RectReport{X: 220, Y: 20, W: 72, H: 40}, Resolved: RectReport{X: 220, Y: 20, W: 72, H: 40}, Measured: SizeReport{W: 72, H: 40}, Pass: "overlay-z-order", Resize: false, Clip: false, ZIndex: 4, Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{Order: 7, ParentID: 1, BlockID: 7, Mode: "scroll", Input: RectReport{X: 236, Y: 72, W: 72, H: 80}, Resolved: RectReport{X: 236, Y: 72, W: 72, H: 80}, Measured: SizeReport{W: 72, H: 160}, Pass: "scroll-clip", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
		{Order: 8, ParentID: 1, BlockID: 8, Mode: "absolute", Input: RectReport{X: 32, Y: 152, W: 0, H: 0}, Resolved: RectReport{X: 32, Y: 152, W: 96, H: 20}, Measured: SizeReport{W: 96, H: 20}, Pass: "fit-text", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 9, ParentID: 1, BlockID: 9, Mode: "absolute", Input: RectReport{X: 164, Y: 152, W: 96, H: 64}, Resolved: RectReport{X: 164, Y: 152, W: 96, H: 54}, Measured: SizeReport{W: 96, H: 54}, Pass: "aspect-fit", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		{Order: 10, ParentID: 0, BlockID: 1, Mode: "column", Input: RectReport{X: 0, Y: 0, W: 480, H: 260}, Resolved: RectReport{X: 12, Y: 12, W: 456, H: 236}, Measured: SizeReport{W: 456, H: 236}, Pass: "resize", Resize: true, Clip: true, ZIndex: 0, Checksum: "sha256:1010101010101010101010101010101010101010101010101010101010101010"},
	}
}
func blockLayoutScrollsForTest() []BlockLayoutScrollReport {
	return []BlockLayoutScrollReport{
		{BlockID: 7, Viewport: RectReport{X: 236, Y: 72, W: 72, H: 80}, Content: SizeReport{W: 72, H: 160}, OffsetY: 32, MaxOffsetY: 80, Clipped: true, Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
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
		{Order: 1, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 4, TargetName: "InputBlock", HitTestPath: []int{1, 2, 4}, DispatchPath: []int{1, 2, 4}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, Disabled: false},
		{Order: 2, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 5, TargetName: "DisabledBlock", HitTestPath: []int{1, 2, 5}, DispatchPath: []int{1, 2, 5}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 5, Delivered: false, Rejected: true, RejectReason: "disabled", FocusedID: 4, Editable: false, Disabled: true},
		{Order: 3, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: false, Rejected: true, RejectReason: "unfocused", FocusedID: 6, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 4, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 5, Kind: "key", Policy: "direct-to-focused-v1", TargetID: 6, TargetName: "ActionBlock", DispatchPath: []int{1, 2, 6}, DirectTargetID: 6, Delivered: true, Rejected: false, FocusedID: 6, Editable: false, Disabled: false},
	}
}
func blockFocusTransitionsForTest() []BlockFocusTransitionReport {
	return []BlockFocusTransitionReport{
		{Order: 1, Helper: "tree_focus_next", BeforeID: 4, AfterID: 6, Direction: "tab", GraphDerived: true, Wrapped: false},
		{Order: 2, Helper: "tree_focus_next", BeforeID: 6, AfterID: 4, Direction: "tab", GraphDerived: true, Wrapped: true},
	}
}
func blockTextEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, X: 20, Y: 64, Width: 320, Height: 200, BufferSlots: []int{5, 20, 64, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"BlockTextApp.focused_id": "0", "InputBlock.focused": "false"}, AfterState: map[string]string{"BlockTextApp.focused_id": "3", "InputBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 4, TextBytesHex: "4f4bd0a2", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 4}, BeforeState: map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"}, AfterState: map[string]string{"InputBlock.buffer": "OKd0a2", "InputBlock.caret": "4"}},
	}
}
func blockEventRuntimeEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "InputBlock", DispatchPath: []string{"BlockEventApp", "PanelBlock", "InputBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "0", "InputBlock.focused": "false"}, AfterState: map[string]string{"BlockEventApp.focused_id": "4", "InputBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "InputBlock", DispatchPath: []string{"BlockEventApp", "PanelBlock", "InputBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"}, AfterState: map[string]string{"InputBlock.buffer": "OK", "InputBlock.caret": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "BlockEventApp", DispatchPath: []string{"BlockEventApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 2, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "4"}, AfterState: map[string]string{"BlockEventApp.focused_id": "6"}},
		{Order: 4, Kind: "key_down", TargetComponent: "BlockEventApp", DispatchPath: []string{"BlockEventApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 3, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "6"}, AfterState: map[string]string{"BlockEventApp.focused_id": "4"}},
	}
}
func blockPaintLayersForTest() []PaintLayerReport {
	return []PaintLayerReport{
		{ID: "root-fill", BlockID: 1, Kind: "fill", Color: "#346ecfff", Radius: 8, Opacity: 255},
		{ID: "root-gradient", BlockID: 1, Kind: "gradient", Color: "#54b484ff", Radius: 8, Opacity: 255},
		{ID: "root-image-fill", BlockID: 1, Kind: "image_fill", Radius: 8, Opacity: 255},
		{ID: "root-border", BlockID: 1, Kind: "border", Color: "#e2eaf2ff", Radius: 8, Width: 1, Opacity: 255},
		{ID: "root-radius-clip", BlockID: 1, Kind: "radius_clip", Radius: 8, Opacity: 255},
		{ID: "root-shadow", BlockID: 1, Kind: "shadow", Color: "#00000058", Blur: 12, OffsetX: 0, OffsetY: 4, Opacity: 88},
		{ID: "root-overlay", BlockID: 1, Kind: "overlay", Color: "#10182066", Radius: 8, Opacity: 102},
		{ID: "root-outline", BlockID: 1, Kind: "outline", Color: "#f4cd5cff", Radius: 10, Width: 2, Opacity: 255},
		{ID: "root-text", BlockID: 1, Kind: "text", Color: "#edf2f7ff", Opacity: 255},
		{ID: "root-icon", BlockID: 1, Kind: "icon", Color: "#f4cd5cff", Opacity: 255},
	}
}
func blockPaintCommandsForTest() []PaintCommandReport {
	return []PaintCommandReport{
		{Order: 1, Command: "fill", LayerID: "root-fill", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-rect-v1", Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, Command: "gradient", LayerID: "root-gradient", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "two-stop-linear-v1", Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, Command: "image_fill", LayerID: "root-image-fill", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "bounded-asset-fill-v1", Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, Command: "border", LayerID: "root-border", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-outline-v1", Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Order: 5, Command: "radius_clip", LayerID: "root-radius-clip", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "clip-stack-v1", Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
		{Order: 6, Command: "shadow", LayerID: "root-shadow", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "box-shadow-approx-v1", Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{Order: 7, Command: "overlay", LayerID: "root-overlay", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "alpha-over-v1", Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
		{Order: 8, Command: "outline", LayerID: "root-outline", BlockID: 1, Rect: RectReport{X: 10, Y: 8, W: 68, H: 32}, Radius: 10, Quality: "rounded-outline-v1", Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 9, Command: "text", LayerID: "root-text", BlockID: 1, Rect: RectReport{X: 20, Y: 16, W: 32, H: 12}, Quality: "glyph-run-v1", Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		{Order: 10, Command: "icon", LayerID: "root-icon", BlockID: 1, Rect: RectReport{X: 56, Y: 16, W: 12, H: 12}, Quality: "monochrome-mask-v1", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
}
func rendererReportForTest() *RendererReport {
	return &RendererReport{
		Schema:                       "tetra.surface.renderer-feature.v1",
		Backend:                      "software-rgba",
		ColorFormat:                  "rgba8",
		QualityLevel:                 "deterministic-software-renderer-v1",
		SoftwareRenderer:             true,
		GPUProductionClaim:           false,
		BlurProductionClaim:          false,
		BackdropBlurProductionClaim:  false,
		CommandOrder:                 []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"},
		CompositorLayers:             rendererCompositorLayersForTest(),
		DirtyRects:                   rendererDirtyRectsForTest(),
		Invalidations:                rendererInvalidationsForTest(),
		CacheStats:                   rendererCacheStatsForTest(),
		UnsupportedEffectsRejected:   []string{"gpu-production", "blur", "backdrop-blur"},
		DeterministicFrameChecksums:  []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		ReferenceFrameArtifactSHA256: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	}
}
func rendererCompositorLayersForTest() []RendererCompositorLayerReport {
	return []RendererCompositorLayerReport{
		{ID: "root", Kind: "root", Order: 1, BlockID: 1, Rect: RectReport{X: 0, Y: 0, W: 320, H: 200}, Opacity: 255, Transform: "identity", Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "content", Kind: "content", Order: 2, BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Opacity: 255, Transform: "translate(0,0)", ClipApplied: true, Clip: RectReport{X: 12, Y: 10, W: 64, H: 28}, Checksum: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "overlay", Kind: "overlay", Order: 3, BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Opacity: 102, Transform: "translate(0,1)", Checksum: "sha256:1212121212121212121212121212121212121212121212121212121212121212"},
		{ID: "text", Kind: "text", Order: 4, BlockID: 1, Rect: RectReport{X: 20, Y: 16, W: 32, H: 12}, Opacity: 255, Transform: "identity", Checksum: "sha256:1313131313131313131313131313131313131313131313131313131313131313"},
		{ID: "icon", Kind: "icon", Order: 5, BlockID: 1, Rect: RectReport{X: 56, Y: 16, W: 12, H: 12}, Opacity: 255, Transform: "identity", Checksum: "sha256:1414141414141414141414141414141414141414141414141414141414141414"},
	}
}
func rendererDirtyRectsForTest() []RendererDirtyRectReport {
	return []RendererDirtyRectReport{
		{FrameOrder: 1, Rect: RectReport{X: 12, Y: 10, W: 68, H: 36}, Reason: "initial-paint", Checksum: "sha256:1515151515151515151515151515151515151515151515151515151515151515"},
		{FrameOrder: 2, Rect: RectReport{X: 12, Y: 10, W: 68, H: 36}, Reason: "state-change", Checksum: "sha256:1616161616161616161616161616161616161616161616161616161616161616"},
	}
}
func rendererInvalidationsForTest() []RendererInvalidationReport {
	return []RendererInvalidationReport{
		{Order: 1, BlockID: 1, Reason: "hovered changed", DirtyRect: RectReport{X: 12, Y: 10, W: 68, H: 36}, Repaint: true},
		{Order: 2, BlockID: 1, Reason: "text input changed", DirtyRect: RectReport{X: 20, Y: 16, W: 44, H: 12}, Repaint: true},
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
		{ID: "title-measure", BlockID: 2, TextLen: 28, FontFamily: "Tetra UI", FontWeight: 600, FontSize: 16, LineHeight: 20, MaxWidth: 96, Measured: SizeReport{W: 96, H: 40}, LineCount: 2, Wrap: "word", Overflow: "ellipsis", Ellipsis: true, EllipsizedTextLen: 16, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{ID: "input-measure", BlockID: 6, TextLen: 4, FontFamily: "Tetra UI", FontWeight: 400, FontSize: 14, LineHeight: 18, MaxWidth: 120, Measured: SizeReport{W: 34, H: 18}, LineCount: 1, Wrap: "none", Overflow: "clip", Ellipsis: false, EllipsizedTextLen: 4, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
	}
}
func blockFontFallbacksForTest() []FontFallbackReport {
	return []FontFallbackReport{
		{ID: "ui-fallback", RequestedFamily: "Tetra UI", ResolvedFamily: "Tetra UI Fallback", Chain: []string{"Tetra UI", "Noto Sans", "monospace"}, MissingGlyphs: 0, Coverage: "ascii-plus-basic-utf8-smoke"},
	}
}
func blockGlyphCachesForTest() []GlyphCacheReport {
	return []GlyphCacheReport{
		{ID: "glyph-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 4096, EntryCount: 12, Eviction: "lru", Bounded: true},
	}
}
func blockTextRenderCommandsForTest() []TextRenderCommandReport {
	return []TextRenderCommandReport{
		{Order: 1, Command: "measure", MeasurementID: "title-measure", BlockID: 2, Rect: RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-text-measure-v1", Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 2, Command: "render_glyphs", MeasurementID: "title-measure", BlockID: 2, Rect: RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-glyph-markers-v1", Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		{Order: 3, Command: "render_caret", MeasurementID: "input-measure", BlockID: 6, Rect: RectReport{X: 12, Y: 48, W: 120, H: 18}, Clip: RectReport{X: 12, Y: 48, W: 144, H: 36}, Color: "#f4cd5cff", Opacity: 255, Quality: "deterministic-caret-v1", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
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
func removeBlockLayoutPassMode(passes []BlockLayoutPassReport, mode string) []BlockLayoutPassReport {
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
			{ID: 1, Name: "RootBlock", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 3, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "SubmitBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "ResetBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}},
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
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
		},
		DispatchPaths: []BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
		},
	}
}
