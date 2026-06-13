package main

import (
	"tetra_language/tools/validators/surface"
)

type smokeOptions struct {
	ReportPath       string
	Mode             string
	SourcePath       string
	RealWindowProbe  bool
	ProbeTitle       string
	ProbeFramePath   string
	ProbeFrameWidth  int
	ProbeFrameHeight int
	ProbeFrameStride int
}
type headlessScenario struct {
	Components                      []surface.ComponentReport
	ComponentTree                   *surface.ComponentTreeReport
	ComponentTreeAPI                *surface.ComponentTreeAPIReport
	BlockGraph                      *surface.BlockGraphReport
	PaintLayers                     []surface.PaintLayerReport
	PaintCommands                   []surface.PaintCommandReport
	VisualFeatures                  []string
	PaintQualityLevel               string
	PaintCacheBudgetBytes           int
	PaintUnsupportedBlur            bool
	Renderer                        *surface.RendererReport
	TextMeasurements                []surface.TextMeasurementReport
	FontFallbacks                   []surface.FontFallbackReport
	GlyphCaches                     []surface.GlyphCacheReport
	TextRenderCommands              []surface.TextRenderCommandReport
	TextQualityLevel                string
	TextCacheBudgetBytes            int
	LayoutConstraints               []surface.BlockLayoutConstraintReport
	LayoutPasses                    []surface.BlockLayoutPassReport
	LayoutScrolls                   []surface.BlockLayoutScrollReport
	LayoutDensity                   *surface.BlockLayoutDensityReport
	LayoutFeatures                  []string
	LayoutQualityLevel              string
	LayoutUnsupportedCSSFlexbox     bool
	BlockEventRoutes                []surface.BlockEventRouteReport
	BlockFocusTransitions           []surface.BlockFocusTransitionReport
	BlockEventKinds                 []string
	BlockEventPolicy                string
	BlockEventQualityLevel          string
	BlockEventUnsupportedDragDrop   bool
	BlockStateSelectors             []surface.BlockStateSelectorReport
	BlockStateResolutions           []surface.BlockStateResolutionReport
	BlockStateResolverOrder         []string
	BlockStateQualityLevel          string
	BlockStateUnsupportedCSSPseudos bool
	MotionFrames                    []surface.MotionFrameReport
	MotionQualityLevel              string
	MotionClock                     string
	MotionFrameBudget               int
	MotionUnsupportedCSSAnimations  bool
	BlockAssetManifest              *surface.BlockAssetManifestReport
	BlockAssetCache                 surface.BlockAssetCacheReport
	BlockAssetDiagnostics           []surface.BlockAssetDiagnosticReport
	BlockAssetRenderCommands        []surface.BlockAssetRenderCommandReport
	BlockAssetQualityLevel          string
	BlockAssetNetworkFetchAllowed   bool
	BlockAccessibilityTree          *surface.BlockAccessibilityTreeReport
	BlockSystem                     *surface.BlockSystemReport
	Morph                           *surface.MorphReport
	Toolkit                         *surface.ToolkitReport
	AccessibilityTree               *surface.AccessibilityTreeReport
	AppModel                        *surface.AppModelReport
	LinuxAppShell                   *surface.LinuxAppShellReport
	SecurityPermissions             *surface.SecurityPermissionReport
	SurfacePerformanceBudget        *surface.SurfacePerformanceBudgetReport
	BrowserSurface                  *surface.BrowserSurfaceReport
	Events                          []surface.EventReport
	Frames                          []surface.FrameReport
	StateTransitions                []surface.StateTransitionReport
	Cases                           []surface.CaseReport
}
type surfaceProcessEvidence struct {
	Processes    []surface.ProcessReport
	Artifacts    []surface.ArtifactReport
	ArtifactScan surface.ArtifactScanReport
	Frames       []surface.FrameReport
}
type headlessSurfaceRunnerTrace struct {
	Schema                          string                                  `json:"schema"`
	Source                          string                                  `json:"source"`
	Frames                          []surface.FrameReport                   `json:"frames"`
	Events                          []surface.EventReport                   `json:"events"`
	StateTransitions                []surface.StateTransitionReport         `json:"state_transitions"`
	Components                      []surface.ComponentReport               `json:"components"`
	ComponentTree                   *surface.ComponentTreeReport            `json:"component_tree,omitempty"`
	ComponentTreeAPI                *surface.ComponentTreeAPIReport         `json:"component_tree_api,omitempty"`
	BlockGraph                      *surface.BlockGraphReport               `json:"block_graph,omitempty"`
	PaintLayers                     []surface.PaintLayerReport              `json:"paint_layers,omitempty"`
	PaintCommands                   []surface.PaintCommandReport            `json:"paint_commands,omitempty"`
	VisualFeatures                  []string                                `json:"visual_features,omitempty"`
	PaintQualityLevel               string                                  `json:"paint_quality_level,omitempty"`
	PaintCacheBudgetBytes           int                                     `json:"paint_cache_budget_bytes,omitempty"`
	PaintUnsupportedBlur            bool                                    `json:"paint_unsupported_blur,omitempty"`
	Renderer                        *surface.RendererReport                 `json:"renderer,omitempty"`
	TextMeasurements                []surface.TextMeasurementReport         `json:"text_measurements,omitempty"`
	FontFallbacks                   []surface.FontFallbackReport            `json:"font_fallbacks,omitempty"`
	GlyphCaches                     []surface.GlyphCacheReport              `json:"glyph_caches,omitempty"`
	TextRenderCommands              []surface.TextRenderCommandReport       `json:"text_render_commands,omitempty"`
	TextQualityLevel                string                                  `json:"text_quality_level,omitempty"`
	TextCacheBudgetBytes            int                                     `json:"text_cache_budget_bytes,omitempty"`
	LayoutConstraints               []surface.BlockLayoutConstraintReport   `json:"layout_constraints,omitempty"`
	LayoutPasses                    []surface.BlockLayoutPassReport         `json:"layout_passes,omitempty"`
	LayoutScrolls                   []surface.BlockLayoutScrollReport       `json:"layout_scrolls,omitempty"`
	LayoutDensity                   *surface.BlockLayoutDensityReport       `json:"layout_density,omitempty"`
	LayoutFeatures                  []string                                `json:"layout_features,omitempty"`
	LayoutQualityLevel              string                                  `json:"layout_quality_level,omitempty"`
	LayoutUnsupportedCSSFlexbox     bool                                    `json:"layout_unsupported_css_flexbox,omitempty"`
	BlockEventRoutes                []surface.BlockEventRouteReport         `json:"block_event_routes,omitempty"`
	BlockFocusTransitions           []surface.BlockFocusTransitionReport    `json:"block_focus_transitions,omitempty"`
	BlockEventKinds                 []string                                `json:"block_event_kinds,omitempty"`
	BlockEventPolicy                string                                  `json:"block_event_policy,omitempty"`
	BlockEventQualityLevel          string                                  `json:"block_event_quality_level,omitempty"`
	BlockEventUnsupportedDragDrop   bool                                    `json:"block_event_unsupported_drag_drop,omitempty"`
	BlockStateSelectors             []surface.BlockStateSelectorReport      `json:"block_state_selectors,omitempty"`
	BlockStateResolutions           []surface.BlockStateResolutionReport    `json:"block_state_resolutions,omitempty"`
	BlockStateResolverOrder         []string                                `json:"block_state_resolver_order,omitempty"`
	BlockStateQualityLevel          string                                  `json:"block_state_quality_level,omitempty"`
	BlockStateUnsupportedCSSPseudos bool                                    `json:"block_state_unsupported_css_pseudos,omitempty"`
	MotionFrames                    []surface.MotionFrameReport             `json:"motion_frames,omitempty"`
	MotionQualityLevel              string                                  `json:"motion_quality_level,omitempty"`
	MotionClock                     string                                  `json:"motion_clock,omitempty"`
	MotionFrameBudget               int                                     `json:"motion_frame_budget,omitempty"`
	MotionUnsupportedCSSAnimations  bool                                    `json:"motion_unsupported_css_animations,omitempty"`
	BlockAssetManifest              *surface.BlockAssetManifestReport       `json:"block_asset_manifest,omitempty"`
	BlockAssetCache                 surface.BlockAssetCacheReport           `json:"block_asset_cache,omitempty"`
	BlockAssetDiagnostics           []surface.BlockAssetDiagnosticReport    `json:"block_asset_diagnostics,omitempty"`
	BlockAssetRenderCommands        []surface.BlockAssetRenderCommandReport `json:"block_asset_render_commands,omitempty"`
	BlockAssetQualityLevel          string                                  `json:"block_asset_quality_level,omitempty"`
	BlockAssetNetworkFetchAllowed   bool                                    `json:"block_asset_network_fetch_allowed,omitempty"`
	BlockAccessibilityTree          *surface.BlockAccessibilityTreeReport   `json:"block_accessibility_tree,omitempty"`
	BlockSystem                     *surface.BlockSystemReport              `json:"block_system,omitempty"`
	Morph                           *surface.MorphReport                    `json:"morph,omitempty"`
	Toolkit                         *surface.ToolkitReport                  `json:"toolkit,omitempty"`
	AccessibilityTree               *surface.AccessibilityTreeReport        `json:"accessibility_tree,omitempty"`
	AppModel                        *surface.AppModelReport                 `json:"app_model,omitempty"`
	Cases                           []surface.CaseReport                    `json:"cases"`
}
type wasmSurfaceRunnerTrace struct {
	Schema string                        `json:"schema"`
	WASM   string                        `json:"wasm_path"`
	Frames []wasmSurfaceRunnerTraceFrame `json:"frames"`
}
type wasmSurfaceRunnerTraceFrame struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	PixelsLen int    `json:"pixels_len"`
	Checksum  string `json:"checksum"`
}
type browserCanvasTrace struct {
	Schema               string                          `json:"schema"`
	WASM                 string                          `json:"wasm_path"`
	Canvas               browserCanvasTraceCanvas        `json:"canvas"`
	BrowserEvents        []browserCanvasTraceEvent       `json:"browser_events"`
	BrowserClipboard     browserCanvasTraceClipboard     `json:"browser_clipboard"`
	BrowserComposition   browserCanvasTraceComposition   `json:"browser_composition"`
	BrowserAccessibility browserCanvasTraceAccessibility `json:"browser_accessibility"`
	Frames               []browserCanvasTraceFrame       `json:"frames"`
	AppExitCode          int                             `json:"app_exit_code"`
	Error                string                          `json:"error,omitempty"`
}
type browserCanvasTraceCanvas struct {
	Opened   bool `json:"opened"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	Readback bool `json:"readback"`
}
type browserCanvasTraceEvent struct {
	Order      int    `json:"order"`
	NativeType string `json:"native_type"`
	Kind       int    `json:"kind"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Key        int    `json:"key"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	TextLen    int    `json:"text_len"`
}
type browserCanvasTraceClipboard struct {
	Harness   string `json:"harness"`
	Read      bool   `json:"read"`
	Write     bool   `json:"write"`
	OwnedCopy bool   `json:"owned_copy"`
	Bytes     int    `json:"bytes"`
}
type browserCanvasTraceComposition struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
}
type browserCanvasTraceAccessibility struct {
	Snapshot      bool     `json:"snapshot"`
	Mirror        bool     `json:"mirror"`
	CompilerOwned bool     `json:"compiler_owned"`
	Roles         []string `json:"roles"`
	Bounds        bool     `json:"bounds"`
	Focus         bool     `json:"focus"`
	DOMVisualUI   bool     `json:"dom_visual_ui"`
	UserJS        bool     `json:"user_js"`
}
type browserCanvasTraceFrame struct {
	Order           int    `json:"order"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Stride          int    `json:"stride"`
	PixelsLen       int    `json:"pixels_len"`
	SourcePixelsB64 string `json:"source_pixels_b64"`
	CanvasPixelsB64 string `json:"canvas_pixels_b64"`
}
type sidecarScanOptions struct {
	AllowCompilerOwnedWASMLoader bool
}
type rgbaFrame struct {
	Width  int
	Height int
	Stride int
	Pixels []byte
}
type rgbaColor struct {
	R byte
	G byte
	B byte
	A byte
}
type rect struct {
	X int
	Y int
	W int
	H int
}
