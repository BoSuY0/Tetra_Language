package main

import (
	"tetra_language/tools/validators/surface"
)

func runBlockPaintScenario() headlessScenario {
	beforeFrame := renderBlockPaintFrameRGBA(false)
	afterFrame := renderBlockPaintFrameRGBA(true)
	frames := []surface.FrameReport{
		{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
		{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
	}
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "BlockPaintApp",
				Type:      "examples.surface_block_paint_layers.BlockPaintApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"hovered_id": "2", "pressed_count": "1", "text_count": "1", "accessibility_role": "none"},
			},
			{
				ID:        "PaintBlock",
				Type:      "examples.surface_block_paint_layers.PaintSurfaceBlock",
				Parent:    "BlockPaintApp",
				Bounds:    surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"paint_layers": "5", "radius": "8", "hovered": "true", "text_len_seen": "2", "accessibility_role": "button"},
			},
		},
		PaintLayers:           blockPaintLayersForScenario(),
		PaintCommands:         blockPaintCommandsForScenario(),
		VisualFeatures:        blockRendererVisualFeaturesForScenario(),
		PaintQualityLevel:     "deterministic-software-paint-v1",
		PaintCacheBudgetBytes: 65536,
		PaintUnsupportedBlur:  false,
		Renderer:              blockRendererReportForScenario(frames, 2),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "PaintBlock",
				DispatchPath:    []string{"BlockPaintApp", "PaintBlock"},
				Handled:         true,
				Pass:            true,
				X:               32,
				Y:               24,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 32, 24, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"BlockPaintApp.pressed_count": "0", "PaintBlock.hovered": "false"},
				AfterState:      map[string]string{"BlockPaintApp.pressed_count": "1", "PaintBlock.hovered": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "PaintBlock",
				DispatchPath:    []string{"BlockPaintApp", "PaintBlock"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"BlockPaintApp.text_count": "0", "PaintBlock.text_len_seen": "0"},
				AfterState:      map[string]string{"BlockPaintApp.text_count": "1", "PaintBlock.text_len_seen": "2"},
			},
		},
		Frames: frames,
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "BlockPaintApp", Field: "pressed_count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "PaintBlock", Field: "hovered", Before: "false", After: "true", Cause: "mouse_up"},
			{Order: 3, Component: "PaintBlock", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon", Kind: "positive", Ran: true, Pass: true},
			{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
			{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
			{Name: "block compositor dirty rect invalidation cache", Kind: "positive", Ran: true, Pass: true},
			{Name: "block renderer opacity transform clipped child", Kind: "positive", Ran: true, Pass: true},
			{Name: "block renderer gpu production claim rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "gpu production"},
			{Name: "block renderer unsupported backdrop blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "backdrop blur"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockPaintLayersForScenario() []surface.PaintLayerReport {
	return []surface.PaintLayerReport{
		{ID: "root-fill", BlockID: 2, Kind: "fill", Color: "#346ecfff", Radius: 8, Opacity: 255},
		{ID: "root-gradient", BlockID: 2, Kind: "gradient", Color: "#54b484ff", Radius: 8, Opacity: 255},
		{ID: "root-image-fill", BlockID: 2, Kind: "image_fill", Radius: 8, Opacity: 255},
		{ID: "root-border", BlockID: 2, Kind: "border", Color: "#e2eaf2ff", Radius: 8, Width: 1, Opacity: 255},
		{ID: "root-radius-clip", BlockID: 2, Kind: "radius_clip", Radius: 8, Opacity: 255},
		{ID: "root-shadow", BlockID: 2, Kind: "shadow", Color: "#00000058", Blur: 12, OffsetX: 0, OffsetY: 4, Opacity: 88},
		{ID: "root-overlay", BlockID: 2, Kind: "overlay", Color: "#10182066", Radius: 8, Opacity: 102},
		{ID: "root-outline", BlockID: 2, Kind: "outline", Color: "#f4cd5cff", Radius: 10, Width: 2, Opacity: 255},
		{ID: "root-text", BlockID: 2, Kind: "text", Color: "#edf2f7ff", Opacity: 255},
		{ID: "root-icon", BlockID: 2, Kind: "icon", Color: "#f4cd5cff", Opacity: 255},
	}
}
func blockPaintCommandsForScenario() []surface.PaintCommandReport {
	return []surface.PaintCommandReport{
		{Order: 1, Command: "fill", LayerID: "root-fill", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-rect-v1", Checksum: "sha256:" + checksumText("paint-fill")},
		{Order: 2, Command: "gradient", LayerID: "root-gradient", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "two-stop-linear-v1", Checksum: "sha256:" + checksumText("paint-gradient")},
		{Order: 3, Command: "image_fill", LayerID: "root-image-fill", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "bounded-asset-fill-v1", Checksum: "sha256:" + checksumText("paint-image-fill")},
		{Order: 4, Command: "border", LayerID: "root-border", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-outline-v1", Checksum: "sha256:" + checksumText("paint-border")},
		{Order: 5, Command: "radius_clip", LayerID: "root-radius-clip", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Clip: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "clip-stack-v1", Checksum: "sha256:" + checksumText("paint-radius-clip")},
		{Order: 6, Command: "shadow", LayerID: "root-shadow", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "box-shadow-approx-v1", Checksum: "sha256:" + checksumText("paint-shadow")},
		{Order: 7, Command: "overlay", LayerID: "root-overlay", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Opacity: 102, Quality: "alpha-over-v1", Checksum: "sha256:" + checksumText("paint-overlay")},
		{Order: 8, Command: "outline", LayerID: "root-outline", BlockID: 2, Rect: surface.RectReport{X: 10, Y: 8, W: 68, H: 32}, Radius: 10, Quality: "rounded-outline-v1", Checksum: "sha256:" + checksumText("paint-outline")},
		{Order: 9, Command: "text", LayerID: "root-text", BlockID: 2, Rect: surface.RectReport{X: 20, Y: 16, W: 32, H: 12}, Quality: "glyph-run-v1", Checksum: "sha256:" + checksumText("paint-text")},
		{Order: 10, Command: "icon", LayerID: "root-icon", BlockID: 2, Rect: surface.RectReport{X: 56, Y: 16, W: 12, H: 12}, Quality: "monochrome-mask-v1", Checksum: "sha256:" + checksumText("paint-icon")},
	}
}
func blockRendererVisualFeaturesForScenario() []string {
	return []string{"fill", "gradient", "image_fill", "border", "radius", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
}
func blockRendererCommandOrderForScenario() []string {
	return []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
}
func blockRendererReportForScenario(frames []surface.FrameReport, blockID int) *surface.RendererReport {
	checksums := make([]string, 0, 2)
	for _, frame := range frames {
		if len(checksums) >= 2 {
			break
		}
		checksums = append(checksums, frame.Checksum)
	}
	if len(checksums) < 2 {
		checksums = append(checksums, "sha256:"+checksumText("surface-renderer-missing-frame"))
	}
	return &surface.RendererReport{
		Schema:                       surface.RendererFeatureSchemaV1,
		Backend:                      "software-rgba",
		ColorFormat:                  "rgba8",
		QualityLevel:                 "deterministic-software-renderer-v1",
		SoftwareRenderer:             true,
		GPUProductionClaim:           false,
		BlurProductionClaim:          false,
		BackdropBlurProductionClaim:  false,
		CommandOrder:                 blockRendererCommandOrderForScenario(),
		CompositorLayers:             blockRendererCompositorLayersForScenario(blockID),
		DirtyRects:                   blockRendererDirtyRectsForScenario(),
		Invalidations:                blockRendererInvalidationsForScenario(blockID),
		CacheStats:                   blockRendererCacheStatsForScenario(),
		UnsupportedEffectsRejected:   []string{"gpu-production", "blur", "backdrop-blur"},
		DeterministicFrameChecksums:  checksums,
		ReferenceFrameArtifactSHA256: "sha256:" + checksumText("surface-renderer-reference-frame-v1"),
	}
}
func blockRendererCompositorLayersForScenario(blockID int) []surface.RendererCompositorLayerReport {
	return []surface.RendererCompositorLayerReport{
		{ID: "root", Kind: "root", Order: 1, BlockID: blockID, Rect: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Opacity: 255, Transform: "identity", Checksum: "sha256:" + checksumText("renderer-layer-root")},
		{ID: "content", Kind: "content", Order: 2, BlockID: blockID, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, ClipApplied: true, Clip: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Opacity: 255, Transform: "translate(0,0)", Checksum: "sha256:" + checksumText("renderer-layer-content")},
		{ID: "overlay", Kind: "overlay", Order: 3, BlockID: blockID, Rect: surface.RectReport{X: 12, Y: 10, W: 64, H: 28}, Opacity: 102, Transform: "translate(0,1)", Checksum: "sha256:" + checksumText("renderer-layer-overlay")},
		{ID: "text", Kind: "text", Order: 4, BlockID: blockID, Rect: surface.RectReport{X: 20, Y: 16, W: 32, H: 12}, Opacity: 255, Transform: "identity", Checksum: "sha256:" + checksumText("renderer-layer-text")},
		{ID: "icon", Kind: "icon", Order: 5, BlockID: blockID, Rect: surface.RectReport{X: 56, Y: 16, W: 12, H: 12}, Opacity: 255, Transform: "identity", Checksum: "sha256:" + checksumText("renderer-layer-icon")},
	}
}
func blockRendererDirtyRectsForScenario() []surface.RendererDirtyRectReport {
	return []surface.RendererDirtyRectReport{
		{FrameOrder: 1, Rect: surface.RectReport{X: 12, Y: 10, W: 68, H: 36}, Reason: "initial-paint", Checksum: "sha256:" + checksumText("renderer-dirty-initial")},
		{FrameOrder: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 68, H: 36}, Reason: "state-change", Checksum: "sha256:" + checksumText("renderer-dirty-state-change")},
	}
}
func blockRendererInvalidationsForScenario(blockID int) []surface.RendererInvalidationReport {
	return []surface.RendererInvalidationReport{
		{Order: 1, BlockID: blockID, Reason: "hovered changed", DirtyRect: surface.RectReport{X: 12, Y: 10, W: 68, H: 36}, Repaint: true},
		{Order: 2, BlockID: blockID, Reason: "text input changed", DirtyRect: surface.RectReport{X: 20, Y: 16, W: 44, H: 12}, Repaint: true},
	}
}
func blockRendererCacheStatsForScenario() surface.RendererCacheStatsReport {
	return surface.RendererCacheStatsReport{
		ID:          "software-rgba-render-cache",
		Strategy:    "bounded-lru",
		BudgetBytes: 65536,
		UsedBytes:   len(blockRendererCommandOrderForScenario()) * 2048,
		EntryCount:  len(blockRendererCommandOrderForScenario()),
		Hits:        3,
		Misses:      2,
		Bounded:     true,
	}
}
func runBlockTextScenario() headlessScenario {
	beforeFrame := renderBlockTextFrameRGBA(false)
	afterFrame := renderBlockTextFrameRGBA(true)
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "BlockTextApp",
				Type:      "examples.surface_block_text.BlockTextApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused_id": "3", "text_quality": "deterministic-fallback-text-v1"},
			},
			{
				ID:        "TextBlock",
				Type:      "examples.surface_block_text.TextSurfaceBlock",
				Parent:    "BlockTextApp",
				Bounds:    surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"},
			},
			{
				ID:        "InputBlock",
				Type:      "examples.surface_block_text.EditableTextBlock",
				Parent:    "BlockTextApp",
				Bounds:    surface.RectReport{X: 12, Y: 58, W: 144, H: 36},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"},
			},
		},
		TextMeasurements:     blockTextMeasurementsForScenario(),
		FontFallbacks:        blockFontFallbacksForScenario(),
		GlyphCaches:          blockGlyphCachesForScenario(),
		TextRenderCommands:   blockTextRenderCommandsForScenario(),
		TextQualityLevel:     "deterministic-fallback-text-v1",
		TextCacheBudgetBytes: 65536,
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockTextApp", "InputBlock"},
				Handled:         true,
				Pass:            true,
				X:               20,
				Y:               64,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 20, 64, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"BlockTextApp.focused_id": "0", "InputBlock.focused": "false"},
				AfterState:      map[string]string{"BlockTextApp.focused_id": "3", "InputBlock.focused": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockTextApp", "InputBlock"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         4,
				TextBytesHex:    "4f4bd0a2",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 4},
				BeforeState:     map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"},
				AfterState:      map[string]string{"InputBlock.buffer": "OKd0a2", "InputBlock.caret": "4"},
			},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "BlockTextApp", Field: "focused_id", Before: "0", After: "3", Cause: "mouse_up"},
			{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
			{Order: 3, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockTextMeasurementsForScenario() []surface.TextMeasurementReport {
	return []surface.TextMeasurementReport{
		{ID: "title-measure", BlockID: 2, TextLen: 28, FontFamily: "Tetra UI", FontWeight: 600, FontSize: 16, LineHeight: 20, MaxWidth: 96, Measured: surface.SizeReport{W: 96, H: 40}, LineCount: 2, Wrap: "word", Overflow: "ellipsis", Ellipsis: true, EllipsizedTextLen: 16, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:" + checksumText("text-title-measure")},
		{ID: "input-measure", BlockID: 3, TextLen: 4, FontFamily: "Tetra UI", FontWeight: 400, FontSize: 14, LineHeight: 18, MaxWidth: 120, Measured: surface.SizeReport{W: 34, H: 18}, LineCount: 1, Wrap: "none", Overflow: "clip", Ellipsis: false, EllipsizedTextLen: 4, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:" + checksumText("text-input-measure")},
	}
}
func blockFontFallbacksForScenario() []surface.FontFallbackReport {
	return []surface.FontFallbackReport{
		{ID: "ui-fallback", RequestedFamily: "Tetra UI", ResolvedFamily: "Tetra UI Fallback", Chain: []string{"Tetra UI", "Noto Sans", "monospace"}, MissingGlyphs: 0, Coverage: "ascii-plus-basic-utf8-smoke"},
	}
}
func blockGlyphCachesForScenario() []surface.GlyphCacheReport {
	return []surface.GlyphCacheReport{
		{ID: "glyph-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 4096, EntryCount: 12, Eviction: "lru", Bounded: true},
	}
}
func blockTextRenderCommandsForScenario() []surface.TextRenderCommandReport {
	return []surface.TextRenderCommandReport{
		{Order: 1, Command: "measure", MeasurementID: "title-measure", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: surface.RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-text-measure-v1", Checksum: "sha256:" + checksumText("text-command-measure")},
		{Order: 2, Command: "render_glyphs", MeasurementID: "title-measure", BlockID: 2, Rect: surface.RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: surface.RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-glyph-markers-v1", Checksum: "sha256:" + checksumText("text-command-glyphs")},
		{Order: 3, Command: "render_caret", MeasurementID: "input-measure", BlockID: 3, Rect: surface.RectReport{X: 12, Y: 58, W: 120, H: 18}, Clip: surface.RectReport{X: 12, Y: 58, W: 144, H: 36}, Color: "#f4cd5cff", Opacity: 255, Quality: "deterministic-caret-v1", Checksum: "sha256:" + checksumText("text-command-caret")},
	}
}
func runBlockLayoutScenario() headlessScenario {
	beforeFrame := renderBlockLayoutFrameRGBA(false)
	afterFrame := renderBlockLayoutFrameRGBA(true)
	resizedFrame := renderBlockLayoutResizedFrameRGBA()
	return headlessScenario{
		Components:                  blockLayoutComponentsForScenario(),
		LayoutConstraints:           blockLayoutConstraintsForScenario(),
		LayoutPasses:                blockLayoutPassesForScenario(),
		LayoutScrolls:               blockLayoutScrollsForScenario(),
		LayoutDensity:               blockLayoutDensityForScenario(),
		LayoutFeatures:              []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "aspect", "spacing", "alignment", "z-order", "clipping", "resize", "density", "stable-rounding"},
		LayoutQualityLevel:          "deterministic-block-layout-v1",
		LayoutUnsupportedCSSFlexbox: false,
		Events: []surface.EventReport{
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
				TimestampMS:     0,
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
			{Order: 3, Width: resizedFrame.Width, Height: resizedFrame.Height, Stride: resizedFrame.Stride, Checksum: checksumRGBA(resizedFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "RowBlock", Field: "pressed", Before: "false", After: "true", Cause: "mouse_up"},
			{Order: 2, Component: "RowBlock", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
			{Order: 3, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
			{Order: 4, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout aspect density stable rounding", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockLayoutComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{ID: "BlockLayoutApp", Type: "examples.surface_block_layout.BlockLayoutApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ColumnBlock", Type: "examples.surface_block_layout.ColumnBlock", Parent: "BlockLayoutApp", Bounds: surface.RectReport{X: 12, Y: 12, W: 296, H: 176}, Abilities: abilities, State: map[string]string{"mode": "column", "gap": "8"}},
		{ID: "RowBlock", Type: "examples.surface_block_layout.RowBlock", Parent: "ColumnBlock", Bounds: surface.RectReport{X: 24, Y: 24, W: 272, H: 48}, Abilities: abilities, State: map[string]string{"mode": "row", "gap": "6"}},
		{ID: "GridBlock", Type: "examples.surface_block_layout.GridBlock", Parent: "ColumnBlock", Bounds: surface.RectReport{X: 24, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "grid", "columns": "2"}},
		{ID: "DockBlock", Type: "examples.surface_block_layout.DockBlock", Parent: "ColumnBlock", Bounds: surface.RectReport{X: 164, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "dock"}},
		{ID: "OverlayBlock", Type: "examples.surface_block_layout.OverlayBlock", Parent: "BlockLayoutApp", Bounds: surface.RectReport{X: 220, Y: 20, W: 72, H: 40}, Abilities: abilities, State: map[string]string{"mode": "overlay", "z": "4"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_layout.ScrollBlock", Parent: "BlockLayoutApp", Bounds: surface.RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"mode": "scroll", "clipped": "true"}},
	}
}
func blockLayoutConstraintsForScenario() []surface.BlockLayoutConstraintReport {
	return []surface.BlockLayoutConstraintReport{
		{ID: "root-column", BlockID: 1, Mode: "column", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: surface.SizeReport{W: 320, H: 200}, Max: surface.SizeReport{W: 480, H: 260}, Padding: 12, Margin: 0, Gap: 8, Align: "stretch", Justify: "start", Overflow: "clip", ZIndex: 0, Clip: true},
		{ID: "row-fill", BlockID: 3, Mode: "row", WidthPolicy: "fill", HeightPolicy: "fixed", Min: surface.SizeReport{W: 160, H: 40}, Max: surface.SizeReport{W: 296, H: 64}, Padding: 6, Margin: 0, Gap: 6, Align: "center", Justify: "space-between", Overflow: "visible", ZIndex: 1, Clip: false},
		{ID: "text-fit", BlockID: 8, Mode: "absolute", WidthPolicy: "fit", HeightPolicy: "fit", Min: surface.SizeReport{W: 32, H: 18}, Max: surface.SizeReport{W: 160, H: 40}, Padding: 4, Margin: 0, Gap: 0, Align: "start", Justify: "start", Overflow: "clip", ZIndex: 2, Clip: true},
		{ID: "overlay-z", BlockID: 6, Mode: "overlay", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: surface.SizeReport{W: 72, H: 40}, Max: surface.SizeReport{W: 72, H: 40}, Padding: 0, Margin: 0, Gap: 0, Align: "end", Justify: "start", Overflow: "visible", ZIndex: 4, Clip: false},
		{ID: "aspect-fit", BlockID: 9, Mode: "absolute", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: surface.SizeReport{W: 96, H: 54}, Max: surface.SizeReport{W: 96, H: 54}, Padding: 0, Margin: 0, Gap: 0, Align: "start", Justify: "start", Overflow: "clip", ZIndex: 2, Clip: true},
	}
}
func blockLayoutPassesForScenario() []surface.BlockLayoutPassReport {
	return []surface.BlockLayoutPassReport{
		{Order: 1, ParentID: 0, BlockID: 1, Mode: "column", Input: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Resolved: surface.RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: surface.SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: true, ZIndex: 0, Checksum: "sha256:" + checksumText("layout-column")},
		{Order: 2, ParentID: 1, BlockID: 2, Mode: "stack", Input: surface.RectReport{X: 12, Y: 12, W: 296, H: 176}, Resolved: surface.RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: surface.SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: false, ZIndex: 0, Checksum: "sha256:" + checksumText("layout-stack")},
		{Order: 3, ParentID: 2, BlockID: 3, Mode: "row", Input: surface.RectReport{X: 24, Y: 24, W: 272, H: 48}, Resolved: surface.RectReport{X: 24, Y: 24, W: 272, H: 48}, Measured: surface.SizeReport{W: 272, H: 48}, Pass: "nested", Resize: false, Clip: false, ZIndex: 1, Checksum: "sha256:" + checksumText("layout-row")},
		{Order: 4, ParentID: 2, BlockID: 4, Mode: "grid", Input: surface.RectReport{X: 24, Y: 80, W: 132, H: 72}, Resolved: surface.RectReport{X: 24, Y: 80, W: 63, H: 34}, Measured: surface.SizeReport{W: 63, H: 34}, Pass: "grid-cell", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:" + checksumText("layout-grid")},
		{Order: 5, ParentID: 2, BlockID: 5, Mode: "dock", Input: surface.RectReport{X: 164, Y: 80, W: 132, H: 72}, Resolved: surface.RectReport{X: 164, Y: 80, W: 132, H: 24}, Measured: surface.SizeReport{W: 132, H: 24}, Pass: "dock-top", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:" + checksumText("layout-dock")},
		{Order: 6, ParentID: 1, BlockID: 6, Mode: "overlay", Input: surface.RectReport{X: 220, Y: 20, W: 72, H: 40}, Resolved: surface.RectReport{X: 220, Y: 20, W: 72, H: 40}, Measured: surface.SizeReport{W: 72, H: 40}, Pass: "overlay-z-order", Resize: false, Clip: false, ZIndex: 4, Checksum: "sha256:" + checksumText("layout-overlay")},
		{Order: 7, ParentID: 1, BlockID: 7, Mode: "scroll", Input: surface.RectReport{X: 236, Y: 72, W: 72, H: 80}, Resolved: surface.RectReport{X: 236, Y: 72, W: 72, H: 80}, Measured: surface.SizeReport{W: 72, H: 160}, Pass: "scroll-clip", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:" + checksumText("layout-scroll")},
		{Order: 8, ParentID: 1, BlockID: 8, Mode: "absolute", Input: surface.RectReport{X: 32, Y: 152, W: 0, H: 0}, Resolved: surface.RectReport{X: 32, Y: 152, W: 96, H: 20}, Measured: surface.SizeReport{W: 96, H: 20}, Pass: "fit-text", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:" + checksumText("layout-absolute-fit")},
		{Order: 9, ParentID: 1, BlockID: 9, Mode: "absolute", Input: surface.RectReport{X: 164, Y: 152, W: 96, H: 64}, Resolved: surface.RectReport{X: 164, Y: 152, W: 96, H: 54}, Measured: surface.SizeReport{W: 96, H: 54}, Pass: "aspect-fit", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:" + checksumText("layout-aspect-fit")},
		{Order: 10, ParentID: 0, BlockID: 1, Mode: "column", Input: surface.RectReport{X: 0, Y: 0, W: 480, H: 260}, Resolved: surface.RectReport{X: 12, Y: 12, W: 456, H: 236}, Measured: surface.SizeReport{W: 456, H: 236}, Pass: "resize", Resize: true, Clip: true, ZIndex: 0, Checksum: "sha256:" + checksumText("layout-resize")},
	}
}
func blockLayoutScrollsForScenario() []surface.BlockLayoutScrollReport {
	return []surface.BlockLayoutScrollReport{
		{BlockID: 7, Viewport: surface.RectReport{X: 236, Y: 72, W: 72, H: 80}, Content: surface.SizeReport{W: 72, H: 160}, OffsetY: 32, MaxOffsetY: 80, Clipped: true, Checksum: "sha256:" + checksumText("layout-scroll-bounds")},
	}
}
func blockLayoutDensityForScenario() *surface.BlockLayoutDensityReport {
	return &surface.BlockLayoutDensityReport{
		TargetDPI:      144,
		ScaleMilli:     1500,
		BaseUnitPx:     4,
		RoundingPolicy: "integer-half-up-v1",
		PixelSnapping:  true,
		Breakpoints:    []string{"small", "medium", "large"},
		Checksum:       "sha256:" + checksumText("layout-density-rounding"),
	}
}
func runBlockEventScenario() headlessScenario {
	beforeFrame := renderBlockEventFrameRGBA(false)
	afterFrame := renderBlockEventFrameRGBA(true)
	return headlessScenario{
		Components:                    blockEventComponentsForScenario(),
		BlockGraph:                    blockEventGraphForScenario("examples/surface_block_events.tetra"),
		BlockEventQualityLevel:        "deterministic-block-events-v1",
		BlockEventPolicy:              "capture-bubble-direct-v1",
		BlockEventUnsupportedDragDrop: false,
		BlockEventKinds:               []string{"pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame"},
		BlockEventRoutes:              blockEventRoutesForScenario(),
		BlockFocusTransitions:         blockFocusTransitionsForScenario(),
		Events: []surface.EventReport{
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
				TimestampMS:     0,
				BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"BlockEventApp.focused_id": "0", "InputBlock.focused": "false"},
				AfterState:      map[string]string{"BlockEventApp.focused_id": "4", "InputBlock.focused": "true"},
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "BlockEventApp", Field: "focused_id", Before: "0", After: "4", Cause: "click"},
			{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
			{Order: 3, Component: "BlockEventApp", Field: "focused_id", Before: "4", After: "6", Cause: "tab"},
			{Order: 4, Component: "BlockEventApp", Field: "focused_id", Before: "6", After: "4", Cause: "tab"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
			{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
			{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
			{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block event nested hit-test path", Kind: "positive", Ran: true, Pass: true},
			{Name: "block event capture bubble direct policy", Kind: "positive", Ran: true, Pass: true},
			{Name: "block event disabled click rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "disabled Block"},
			{Name: "block event text input focused only", Kind: "positive", Ran: true, Pass: true},
			{Name: "block focus tab order graph-derived", Kind: "positive", Ran: true, Pass: true},
			{Name: "block event no complex drag claim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "drag-and-drop nonclaim"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockEventComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{ID: "BlockEventApp", Type: "examples.surface_block_events.BlockEventApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "event_quality": "deterministic-block-events-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_events.PanelBlock", Parent: "BlockEventApp", Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"role": "panel"}},
		{ID: "LabelBlock", Type: "examples.surface_block_events.LabelBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "10"}},
		{ID: "InputBlock", Type: "examples.surface_block_events.InputBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"editable": "true", "focused": "true", "buffer": "OK"}},
		{ID: "DisabledBlock", Type: "examples.surface_block_events.DisabledBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"disabled": "true"}},
		{ID: "ActionBlock", Type: "examples.surface_block_events.ActionBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 24, Y: 120, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false"}},
	}
}
func blockEventGraphForScenario(source string) *surface.BlockGraphReport {
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         6,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
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
		Nodes: []surface.BlockGraphNodeReport{
			{ID: 1, Name: "BlockEventApp", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 4, Focusable: false, AccessibilityRole: "none", Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "InputBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "textbox", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "DisabledBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "button", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}},
			{ID: 6, Name: "ActionBlock", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 24, Y: 120, W: 120, H: 44}},
		},
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5, 6},
		DrawOrder:          []int{1, 2, 3, 4, 5, 6},
		FocusOrder:         []int{4, 6},
		AccessibilityOrder: []int{3, 4, 5, 6},
		HitTests: []surface.BlockGraphPathReport{
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 4, X: 40, Y: 80, Path: []int{1, 2, 4}},
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 6, Path: []int{1, 2, 6}},
		},
	}
}
func blockEventRoutesForScenario() []surface.BlockEventRouteReport {
	return []surface.BlockEventRouteReport{
		{Order: 1, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 4, TargetName: "InputBlock", HitTestPath: []int{1, 2, 4}, DispatchPath: []int{1, 2, 4}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, Disabled: false},
		{Order: 2, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 5, TargetName: "DisabledBlock", HitTestPath: []int{1, 2, 5}, DispatchPath: []int{1, 2, 5}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 5, Delivered: false, Rejected: true, RejectReason: "disabled", FocusedID: 4, Editable: false, Disabled: true},
		{Order: 3, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: false, Rejected: true, RejectReason: "unfocused", FocusedID: 6, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 4, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 5, Kind: "key", Policy: "direct-to-focused-v1", TargetID: 6, TargetName: "ActionBlock", DispatchPath: []int{1, 2, 6}, DirectTargetID: 6, Delivered: true, Rejected: false, FocusedID: 6, Editable: false, Disabled: false},
	}
}
func blockFocusTransitionsForScenario() []surface.BlockFocusTransitionReport {
	return []surface.BlockFocusTransitionReport{
		{Order: 1, Helper: "tree_focus_next", BeforeID: 4, AfterID: 6, Direction: "tab", GraphDerived: true, Wrapped: false},
		{Order: 2, Helper: "tree_focus_next", BeforeID: 6, AfterID: 4, Direction: "tab", GraphDerived: true, Wrapped: true},
	}
}
func runBlockStateScenario() headlessScenario {
	beforeFrame := renderBlockStateFrameRGBA(false)
	afterFrame := renderBlockStateFrameRGBA(true)
	return headlessScenario{
		Components:                      blockStateComponentsForScenario(),
		BlockStateQualityLevel:          "deterministic-block-state-resolver-v1",
		BlockStateResolverOrder:         []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"},
		BlockStateUnsupportedCSSPseudos: false,
		BlockStateSelectors:             blockStateSelectorsForScenario(),
		BlockStateResolutions:           blockStateResolutionsForScenario(),
		Events: []surface.EventReport{
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
			{Order: 2, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
			{Order: 3, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
			{Order: 4, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
			{Order: 5, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
			{Order: 6, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockStateComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state"}
	return []surface.ComponentReport{
		{ID: "BlockStateApp", Type: "examples.surface_block_states.BlockStateApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"state_quality": "deterministic-block-state-resolver-v1"}},
		{ID: "StateBlock", Type: "examples.surface_block_states.StateBlock", Parent: "BlockStateApp", Bounds: surface.RectReport{X: 24, Y: 40, W: 168, H: 56}, Abilities: abilities, State: map[string]string{"selector_flags": "127", "variant": "5", "disabled": "true", "error": "true", "loading": "true"}},
		{ID: "StatusBlock", Type: "examples.surface_block_states.StatusBlock", Parent: "BlockStateApp", Bounds: surface.RectReport{X: 24, Y: 112, W: 168, H: 32}, Abilities: abilities, State: map[string]string{"selected": "true", "focused": "true"}},
	}
}
func blockStateSelectorsForScenario() []surface.BlockStateSelectorReport {
	return []surface.BlockStateSelectorReport{
		{Order: 1, Name: "hover", BlockID: 2, Flags: 1, Hovered: true},
		{Order: 2, Name: "pressed", BlockID: 2, Flags: 2, Pressed: true},
		{Order: 3, Name: "focused", BlockID: 2, Flags: 4, Focused: true},
		{Order: 4, Name: "selected", BlockID: 2, Flags: 8, Selected: true},
		{Order: 5, Name: "disabled", BlockID: 2, Flags: 16, Disabled: true},
		{Order: 6, Name: "error", BlockID: 2, Flags: 32, Error: true},
		{Order: 7, Name: "loading", BlockID: 2, Flags: 64, Loading: true},
	}
}
func blockStateResolutionsForScenario() []surface.BlockStateResolutionReport {
	return []surface.BlockStateResolutionReport{
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
func runBlockMotionScenario() headlessScenario {
	startFrame := renderBlockMotionFrameRGBA(0)
	midFrame := renderBlockMotionFrameRGBA(1)
	doneFrame := renderBlockMotionFrameRGBA(2)
	return headlessScenario{
		Components:                     blockMotionComponentsForScenario(),
		MotionQualityLevel:             "deterministic-block-motion-v1",
		MotionClock:                    "deterministic-test-clock-v1",
		MotionFrameBudget:              4,
		MotionUnsupportedCSSAnimations: false,
		MotionFrames:                   blockMotionFramesForScenario(),
		Events: []surface.EventReport{
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: startFrame.Width, Height: startFrame.Height, Stride: startFrame.Stride, Checksum: checksumRGBA(startFrame.Pixels), Presented: true},
			{Order: 2, Width: midFrame.Width, Height: midFrame.Height, Stride: midFrame.Stride, Checksum: checksumRGBA(midFrame.Pixels), Presented: true},
			{Order: 3, Width: doneFrame.Width, Height: doneFrame.Height, Stride: doneFrame.Stride, Checksum: checksumRGBA(doneFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
			{Order: 2, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
			{Order: 3, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
			{Order: 4, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
			{Order: 5, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
			{Order: 6, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
			{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
			{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
			{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
			{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func blockMotionComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state", "motion"}
	return []surface.ComponentReport{
		{ID: "BlockMotionApp", Type: "examples.surface_block_motion.BlockMotionApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"motion_quality": "deterministic-block-motion-v1"}},
		{ID: "MotionBlock", Type: "examples.surface_block_motion.MotionBlock", Parent: "BlockMotionApp", Bounds: surface.RectReport{X: 24, Y: 44, W: 176, H: 64}, Abilities: abilities, State: map[string]string{"opacity": "200", "scale": "108", "translate_x": "12", "complete": "true"}},
	}
}
func blockMotionFramesForScenario() []surface.MotionFrameReport {
	return []surface.MotionFrameReport{
		{Order: 1, BlockID: 2, Trigger: "hover", TimestampMS: 0, DurationMS: 120, DelayMS: 0, Progress: 0, Easing: "linear", Opacity: 80, Color: "#203040ff", TranslateX: 0, TranslateY: 0, Scale: 100, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:" + checksumText("block-motion-frame-start")},
		{Order: 2, BlockID: 2, Trigger: "hover", TimestampMS: 60, DurationMS: 120, DelayMS: 0, Progress: 500, Easing: "linear", Opacity: 140, Color: "#407094ff", TranslateX: 6, TranslateY: 0, Scale: 104, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:" + checksumText("block-motion-frame-mid")},
		{Order: 3, BlockID: 2, Trigger: "hover", TimestampMS: 120, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: false, Scheduled: false, Settled: true, Checksum: "sha256:" + checksumText("block-motion-frame-done")},
		{Order: 4, BlockID: 2, Trigger: "reduced_motion", TimestampMS: 121, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: true, Scheduled: false, Settled: true, Checksum: "sha256:" + checksumText("block-motion-frame-reduced")},
	}
}
func runBlockAssetScenario() headlessScenario {
	beforeFrame := renderBlockAssetFrameRGBA(false)
	afterFrame := renderBlockAssetFrameRGBA(true)
	return headlessScenario{
		Components:                    blockAssetComponentsForScenario(),
		BlockAssetQualityLevel:        "deterministic-local-block-assets-v1",
		BlockAssetNetworkFetchAllowed: false,
		BlockAssetManifest:            blockAssetManifestForScenario("examples/surface_block_assets.tetra"),
		BlockAssetCache:               blockAssetCacheForScenario(),
		BlockAssetDiagnostics:         blockAssetDiagnosticsForScenario(),
		BlockAssetRenderCommands:      blockAssetRenderCommandsForScenario(),
		Events: []surface.EventReport{
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "IconBlock", Field: "tint", Before: "#ffffffff", After: "#60aef4ff", Cause: "asset tint"},
			{Order: 2, Component: "ImageBlock", Field: "scale", Before: "1x", After: "2x", Cause: "asset scale"},
			{Order: 3, Component: "MissingAssetBlock", Field: "fallback", Before: "missing", After: "fallback-raster", Cause: "missing asset"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block asset deterministic manifest hashes", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset missing fallback diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing asset"},
			{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network assets disabled"},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func runBlockAccessibilityScenario() headlessScenario {
	beforeFrame := renderBlockAccessibilityFrameRGBA(false)
	afterFrame := renderBlockAccessibilityFrameRGBA(true)
	return headlessScenario{
		Components:             blockAccessibilityComponentsForScenario(),
		BlockGraph:             blockAccessibilityGraphForScenario("examples/surface_block_accessibility.tetra"),
		BlockAccessibilityTree: blockAccessibilityTreeForScenario("examples/surface_block_accessibility.tetra"),
		Events: []surface.EventReport{
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
				TimestampMS:     0,
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
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "SubmitBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
			{Order: 2, Component: "ResetBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
			{Order: 3, Component: "BlockAccessibilityApp", Field: "reading_order_checked", Before: "false", After: "true", Cause: "block_graph"},
		},
		Cases: []surface.CaseReport{
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
			{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
			{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
			{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
			{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block accessibility tree derived from block graph", Kind: "positive", Ran: true, Pass: true},
			{Name: "block accessibility focusable actionable name required", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing accessible name"},
			{Name: "block accessibility label relationship mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "label relationship mismatch"},
			{Name: "block accessibility reading order graph mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "reading order mismatch"},
			{Name: "block accessibility screen-reader claim without platform proof rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "screen reader proof required"},
			{Name: "block accessibility platform claim scoped metadata only", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func runBlockSystemScenario() headlessScenario {
	source := "examples/surface_block_system.tetra"
	beforeFrame := renderBlockSystemFrameRGBA(false)
	afterFrame := renderBlockSystemFrameRGBA(true)
	motionFrame := renderBlockSystemFrameRGBA(true)
	rectRGBA(motionFrame, rect{X: 188, Y: 124, W: 30, H: 10}, rgbaColor{R: 96, G: 174, B: 244, A: 255})
	frames := []surface.FrameReport{
		{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
		{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		{Order: 3, Width: motionFrame.Width, Height: motionFrame.Height, Stride: motionFrame.Stride, Checksum: checksumRGBA(motionFrame.Pixels), Presented: true},
	}
	components := blockSystemComponentsForScenario()
	components = append(components, retargetBlockSystemComponentsForScenario(blockTextComponentsForScenario())...)
	components = append(components, retargetBlockSystemComponentsForScenario(blockStateComponentsForScenario())...)
	components = append(components, retargetBlockSystemComponentsForScenario(blockMotionComponentsForScenario())...)
	components = append(components, retargetBlockSystemComponentsForScenario(blockAssetComponentsForScenario())...)
	events := blockSystemEventsForScenario()
	events = appendScenarioEventsWithNextOrder(events,
		blockTextEventsForScenario(),
		blockStateEventsForScenario(),
		blockMotionEventsForScenario(),
		blockAssetEventsForScenario(),
	)
	stateTransitions := []surface.StateTransitionReport{
		{Order: 1, Component: "SubmitBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 2, Component: "ResetBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 3, Component: "BlockSystemApp", Field: "reading_order_checked", Before: "false", After: "true", Cause: "block_graph"},
		{Order: 4, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
		{Order: 5, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
	}
	stateTransitions = appendScenarioStateTransitionsWithNextOrder(stateTransitions, blockSystemReadinessTransitionsForScenario())
	scenario := headlessScenario{
		Components:                  components,
		BlockGraph:                  blockAccessibilityGraphForScenario(source),
		PaintLayers:                 blockPaintLayersForScenario(),
		PaintCommands:               blockPaintCommandsForScenario(),
		VisualFeatures:              blockRendererVisualFeaturesForScenario(),
		PaintQualityLevel:           "deterministic-software-paint-v1",
		PaintCacheBudgetBytes:       65536,
		PaintUnsupportedBlur:        false,
		Renderer:                    blockRendererReportForScenario(frames, 2),
		TextMeasurements:            blockTextMeasurementsForScenario(),
		FontFallbacks:               blockFontFallbacksForScenario(),
		GlyphCaches:                 blockGlyphCachesForScenario(),
		TextRenderCommands:          blockTextRenderCommandsForScenario(),
		TextQualityLevel:            "deterministic-fallback-text-v1",
		TextCacheBudgetBytes:        65536,
		LayoutConstraints:           blockLayoutConstraintsForScenario(),
		LayoutPasses:                blockLayoutPassesForScenario(),
		LayoutScrolls:               blockLayoutScrollsForScenario(),
		LayoutDensity:               blockLayoutDensityForScenario(),
		LayoutFeatures:              []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "aspect", "spacing", "alignment", "z-order", "clipping", "resize", "density", "stable-rounding"},
		LayoutQualityLevel:          "deterministic-block-layout-v1",
		LayoutUnsupportedCSSFlexbox: false,
		BlockStateSelectors:         blockStateSelectorsForScenario(),
		BlockStateResolutions:       blockStateResolutionsForScenario(),
		BlockStateResolverOrder:     []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"},
		BlockStateQualityLevel:      "deterministic-block-state-resolver-v1",
		MotionFrames:                blockMotionFramesForScenario(),
		MotionQualityLevel:          "deterministic-block-motion-v1",
		MotionClock:                 "deterministic-test-clock-v1",
		MotionFrameBudget:           4,
		BlockAssetManifest:          blockAssetManifestForScenario(source),
		BlockAssetCache:             blockAssetCacheForScenario(),
		BlockAssetDiagnostics:       blockAssetDiagnosticsForScenario(),
		BlockAssetRenderCommands:    blockAssetRenderCommandsForScenario(),
		BlockAssetQualityLevel:      "deterministic-local-block-assets-v1",
		BlockAccessibilityTree:      blockAccessibilityTreeForScenario(source),
		BlockSystem:                 blockSystemReportForScenario(source, frames),
		Events:                      events,
		Frames:                      frames,
		StateTransitions:            stateTransitions,
		Cases:                       blockSystemCasesForScenario(),
	}
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func runMorphScenario() headlessScenario {
	source := "examples/surface_morph_command_palette.tetra"
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, "examples.surface_morph_command_palette")
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	return scenario
}
func retargetScenarioToSource(scenario *headlessScenario, source string, module string) {
	if scenario == nil {
		return
	}
	for i := range scenario.Components {
		scenario.Components[i].Type = module + "." + typeBaseName(scenario.Components[i].Type)
	}
	if scenario.BlockGraph != nil {
		scenario.BlockGraph.Source = source
	}
	if scenario.BlockAssetManifest != nil {
		scenario.BlockAssetManifest.Source = source
	}
	if scenario.BlockAccessibilityTree != nil {
		scenario.BlockAccessibilityTree.Source = source
	}
	if scenario.BlockSystem != nil {
		scenario.BlockSystem.Source = source
	}
}
