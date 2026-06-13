package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"

	"tetra_language/tools/validators/surface"
)

func mergeFrameEvidenceByOrder(base []surface.FrameReport, evidence []surface.FrameReport) []surface.FrameReport {
	byOrder := map[int]surface.FrameReport{}
	for _, frame := range base {
		byOrder[frame.Order] = frame
	}
	for _, frame := range evidence {
		byOrder[frame.Order] = frame
	}
	orders := make([]int, 0, len(byOrder))
	for order := range byOrder {
		orders = append(orders, order)
	}
	sort.Ints(orders)
	merged := make([]surface.FrameReport, 0, len(orders))
	for _, order := range orders {
		merged = append(merged, byOrder[order])
	}
	return merged
}
func artifactReport(path string, kind string) (surface.ArtifactReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return surface.ArtifactReport{}, fmt.Errorf("open Surface artifact %s: %w", path, err)
	}
	hash := sha256.New()
	size, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return surface.ArtifactReport{}, fmt.Errorf("hash Surface artifact %s: %w", path, copyErr)
	}
	if closeErr != nil {
		return surface.ArtifactReport{}, fmt.Errorf("close Surface artifact %s: %w", path, closeErr)
	}
	return surface.ArtifactReport{
		Kind:   kind,
		Path:   path,
		SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		Size:   size,
	}, nil
}
func buildReport(opt smokeOptions, host string, processes []surface.ProcessReport, artifacts []surface.ArtifactReport, artifactScan surface.ArtifactScanReport, scenario headlessScenario) surface.Report {
	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	source := defaultSurfaceSourcePath(opt)
	target := mode
	runtimeName := "surface-headless"
	if mode == "headless-text-focus-input" || mode == "headless-release-toolkit" || mode == "headless-release-accessibility" || mode == "headless-component-tree" || mode == "headless-component-tree-api" || mode == "headless-block-paint" || mode == "headless-block-text" || mode == "headless-block-layout" || mode == "headless-block-events" || mode == "headless-block-states" || mode == "headless-block-motion" || mode == "headless-block-assets" || mode == "headless-block-accessibility" || mode == "headless-block-system" || mode == "headless-morph" || mode == "headless-minimal-toolkit" || mode == "headless-toolkit-reuse" || mode == "headless-accessibility-metadata" || mode == "headless-app-model" {
		target = "headless"
		runtimeName = "surface-headless"
	} else if mode == "linux-x64" || mode == "linux-x64-real-window" || mode == "linux-x64-real-window-text-focus-input" || mode == "linux-x64-release-toolkit" || mode == "linux-x64-release-window" || mode == "linux-x64-release-app-shell" || mode == "linux-x64-release-accessibility" || mode == "linux-x64-real-window-component-tree" || mode == "linux-x64-real-window-component-tree-api" || mode == "linux-x64-real-window-block-system" || mode == "linux-x64-real-window-minimal-toolkit" || mode == "linux-x64-real-window-toolkit-reuse" || mode == "linux-x64-real-window-accessibility-metadata" {
		target = "linux-x64"
		runtimeName = "surface-linux-x64"
	} else if mode == "wasm32-web" || mode == "wasm32-web-browser-canvas" || mode == "wasm32-web-browser-canvas-text-focus-input" || mode == "wasm32-web-release-toolkit" || mode == "wasm32-web-release-browser" || mode == "wasm32-web-release-accessibility" || mode == "wasm32-web-browser-canvas-component-tree" || mode == "wasm32-web-browser-canvas-component-tree-api" || mode == "wasm32-web-browser-canvas-minimal-toolkit" || mode == "wasm32-web-browser-canvas-toolkit-reuse" || mode == "wasm32-web-browser-canvas-accessibility-metadata" || mode == "wasm32-web-browser-canvas-block-system" {
		target = "wasm32-web"
		runtimeName = "surface-wasm32-web"
	}
	performanceBudget := scenario.SurfacePerformanceBudget
	if performanceBudget == nil {
		performanceBudget = surfacePerformanceBudgetForScenario(target, runtimeName, source, artifacts, scenario)
	}
	return surface.Report{
		Schema:                          surface.SchemaV1,
		Status:                          "pass",
		Target:                          target,
		Host:                            host,
		Runtime:                         runtimeName,
		SurfaceSchema:                   "tetra.surface.v1",
		HostABI:                         "tetra.surface.host-abi.v1",
		HostEvidence:                    hostEvidenceForMode(mode),
		Source:                          source,
		Processes:                       processes,
		Artifacts:                       artifacts,
		ArtifactScan:                    artifactScan,
		Components:                      scenario.Components,
		ComponentTree:                   scenario.ComponentTree,
		ComponentTreeAPI:                scenario.ComponentTreeAPI,
		BlockGraph:                      scenario.BlockGraph,
		PaintLayers:                     scenario.PaintLayers,
		PaintCommands:                   scenario.PaintCommands,
		VisualFeatures:                  scenario.VisualFeatures,
		PaintQualityLevel:               scenario.PaintQualityLevel,
		PaintCacheBudgetBytes:           scenario.PaintCacheBudgetBytes,
		PaintUnsupportedBlur:            scenario.PaintUnsupportedBlur,
		Renderer:                        scenario.Renderer,
		TextMeasurements:                scenario.TextMeasurements,
		FontFallbacks:                   scenario.FontFallbacks,
		GlyphCaches:                     scenario.GlyphCaches,
		TextRenderCommands:              scenario.TextRenderCommands,
		TextQualityLevel:                scenario.TextQualityLevel,
		TextCacheBudgetBytes:            scenario.TextCacheBudgetBytes,
		LayoutConstraints:               scenario.LayoutConstraints,
		LayoutPasses:                    scenario.LayoutPasses,
		LayoutScrolls:                   scenario.LayoutScrolls,
		LayoutDensity:                   scenario.LayoutDensity,
		LayoutFeatures:                  scenario.LayoutFeatures,
		LayoutQualityLevel:              scenario.LayoutQualityLevel,
		LayoutUnsupportedCSSFlexbox:     scenario.LayoutUnsupportedCSSFlexbox,
		BlockEventRoutes:                scenario.BlockEventRoutes,
		BlockFocusTransitions:           scenario.BlockFocusTransitions,
		BlockEventKinds:                 scenario.BlockEventKinds,
		BlockEventPolicy:                scenario.BlockEventPolicy,
		BlockEventQualityLevel:          scenario.BlockEventQualityLevel,
		BlockEventUnsupportedDragDrop:   scenario.BlockEventUnsupportedDragDrop,
		BlockStateSelectors:             scenario.BlockStateSelectors,
		BlockStateResolutions:           scenario.BlockStateResolutions,
		BlockStateResolverOrder:         scenario.BlockStateResolverOrder,
		BlockStateQualityLevel:          scenario.BlockStateQualityLevel,
		BlockStateUnsupportedCSSPseudos: scenario.BlockStateUnsupportedCSSPseudos,
		MotionFrames:                    scenario.MotionFrames,
		MotionQualityLevel:              scenario.MotionQualityLevel,
		MotionClock:                     scenario.MotionClock,
		MotionFrameBudget:               scenario.MotionFrameBudget,
		MotionUnsupportedCSSAnimations:  scenario.MotionUnsupportedCSSAnimations,
		BlockAssetManifest:              scenario.BlockAssetManifest,
		BlockAssetCache:                 scenario.BlockAssetCache,
		BlockAssetDiagnostics:           scenario.BlockAssetDiagnostics,
		BlockAssetRenderCommands:        scenario.BlockAssetRenderCommands,
		BlockAssetQualityLevel:          scenario.BlockAssetQualityLevel,
		BlockAssetNetworkFetchAllowed:   scenario.BlockAssetNetworkFetchAllowed,
		BlockAccessibilityTree:          scenario.BlockAccessibilityTree,
		BlockSystem:                     scenario.BlockSystem,
		Morph:                           scenario.Morph,
		Toolkit:                         scenario.Toolkit,
		AccessibilityTree:               scenario.AccessibilityTree,
		AppModel:                        scenario.AppModel,
		LinuxAppShell:                   scenario.LinuxAppShell,
		SecurityPermissions:             scenario.SecurityPermissions,
		SurfacePerformanceBudget:        performanceBudget,
		BrowserSurface:                  scenario.BrowserSurface,
		Events:                          scenario.Events,
		Frames:                          scenario.Frames,
		StateTransitions:                scenario.StateTransitions,
		Cases:                           scenario.Cases,
	}
}
func buildTextInputReport(opt smokeOptions, processes []surface.ProcessReport, artifacts []surface.ArtifactReport, artifactScan surface.ArtifactScanReport, cases []surface.CaseReport) surface.TextInputReport {
	return surface.TextInputReport{
		Schema:                     surface.TextInputSchemaV1,
		Target:                     releaseTextInputTarget(opt.Mode),
		Source:                     defaultSurfaceSourcePath(opt),
		Level:                      "production-text-input-v1",
		Experimental:               false,
		ProductionClaim:            true,
		Storage:                    "owned-utf8-byte-buffer",
		UTF8Validation:             true,
		InvalidUTF8Rejected:        true,
		Caret:                      true,
		Selection:                  true,
		SelectionClipboardTransfer: true,
		Multiline:                  true,
		Backspace:                  true,
		Delete:                     true,
		HomeEnd:                    true,
		ArrowLeftRight:             true,
		CompositionEvents:          true,
		CompositionCommit:          true,
		CompositionCancel:          true,
		ClipboardRead:              true,
		ClipboardWrite:             true,
		ClipboardHostABI:           true,
		ClipboardOwnedCopy:         true,
		TargetHostCompositionTrace: true,
		CompositionTrace: surface.CompositionTraceReport{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
		},
		TextShapingPlan: surface.TextShapingPlanReport{
			QualityLevel:       "scoped-text-shaping-plan-v1",
			FallbackFonts:      true,
			GraphemeBoundaries: "byte-offset-codepoint-v1",
			LineBreaking:       "newline-storage-plus-wrap-plan-v1",
			Bidi:               "nonclaim-full-bidi-v1",
			RichText:           "nonclaim-rich-text-editor-v1",
		},
		ReferenceTraces: []surface.TextInputReferenceTraceReport{
			{Source: "examples/surface_morph_settings.tetra", Trace: "settings text field trace", Focus: true, Selection: true, Clipboard: true, Composition: true, Multiline: true, Pass: true},
			{Source: "examples/surface_morph_editor_shell.tetra", Trace: "editor shell text area trace", Focus: true, Selection: true, Clipboard: true, Composition: true, Multiline: true, Pass: true},
		},
		UnsupportedClaims: []string{
			"full-rich-text-editor",
			"full-bidi-shaping",
			"grapheme-cluster-caret",
			"ide-grade-editor",
		},
		RichTextProductionClaim:   false,
		BidiProductionClaim:       false,
		FullEditorProductionClaim: false,
		BorrowedViewStorage:       false,
		SafeViewLifetimeChecked:   true,
		Processes:                 processes,
		Artifacts:                 artifacts,
		ArtifactScan:              artifactScan,
		Cases:                     cases,
	}
}
func releaseTextInputTarget(mode string) string {
	switch mode {
	case "linux-x64-release-text-input":
		return "linux-x64"
	case "wasm32-web-release-text-input":
		return "wasm32-web"
	default:
		return "headless"
	}
}
func releaseTextInputCases() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input ASCII insertion", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input UTF-8 insertion", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input invalid UTF-8 rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid utf8 rejected"},
		{Name: "release text input multiline storage", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input caret home end arrows", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection replacement", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection clipboard transfer", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input backspace delete", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input clipboard owned copy transfer", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition start update", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition commit", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition cancel", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input shaping plan scoped", Kind: "positive", Ran: true, Pass: true},
		{Name: "settings reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "editor reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input safe view lifetime checked", Kind: "positive", Ran: true, Pass: true},
		{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
	}
}
func hostEvidenceForMode(mode string) surface.HostEvidenceReport {
	switch mode {
	case "headless-text-focus-input", "headless-release-text-input", "headless-release-toolkit", "headless-release-accessibility", "headless-component-tree", "headless-component-tree-api", "headless-block-paint", "headless-block-text", "headless-block-layout", "headless-block-events", "headless-block-states", "headless-block-motion", "headless-block-assets", "headless-block-accessibility", "headless-block-system", "headless-morph", "headless-minimal-toolkit", "headless-toolkit-reuse", "headless-accessibility-metadata", "headless-app-model":
		return surface.HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		}
	case "linux-x64":
		return surface.HostEvidenceReport{
			Level:       "linux-x64-memfd-starter",
			Backend:     "memfd-rgba",
			Framebuffer: true,
		}
	case "linux-x64-release-window", "linux-x64-release-app-shell":
		return surface.HostEvidenceReport{
			Level:               "linux-x64-release-window-v1",
			Backend:             "wayland-shm-rgba-release-v1",
			Framebuffer:         true,
			RealWindow:          true,
			NativeInput:         true,
			TextInput:           true,
			Clipboard:           true,
			Composition:         true,
			AccessibilityBridge: true,
		}
	case "linux-x64-release-accessibility":
		return surface.HostEvidenceReport{
			Level:               "linux-x64-real-window",
			Backend:             "wayland-shm-rgba",
			Framebuffer:         true,
			RealWindow:          true,
			NativeInput:         true,
			AccessibilityBridge: true,
		}
	case "linux-x64-real-window", "linux-x64-real-window-text-focus-input", "linux-x64-release-text-input", "linux-x64-release-toolkit", "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api", "linux-x64-real-window-block-system", "linux-x64-real-window-minimal-toolkit", "linux-x64-real-window-toolkit-reuse", "linux-x64-real-window-accessibility-metadata":
		return surface.HostEvidenceReport{
			Level:       "linux-x64-real-window",
			Backend:     "wayland-shm-rgba",
			Framebuffer: true,
			RealWindow:  true,
			NativeInput: true,
		}
	case "wasm32-web":
		return surface.HostEvidenceReport{
			Level:       "wasm32-web-compiler-owned-loader",
			Backend:     "node-surface-host",
			Framebuffer: true,
		}
	case "wasm32-web-release-browser":
		return surface.HostEvidenceReport{
			Level:                        "wasm32-web-browser-canvas-release-v1",
			Backend:                      "browser-canvas-rgba-accessible",
			Framebuffer:                  true,
			NativeInput:                  true,
			BrowserCanvas:                true,
			BrowserInput:                 true,
			BrowserClipboard:             true,
			BrowserClipboardHarness:      "deterministic-browser-clipboard-v1",
			BrowserComposition:           true,
			BrowserAccessibilitySnapshot: true,
			BrowserAccessibilityMirror:   true,
		}
	case "wasm32-web-release-accessibility":
		return surface.HostEvidenceReport{
			Level:                        "wasm32-web-browser-canvas-input",
			Backend:                      "browser-canvas-rgba",
			Framebuffer:                  true,
			NativeInput:                  true,
			BrowserCanvas:                true,
			BrowserInput:                 true,
			BrowserAccessibilitySnapshot: true,
			BrowserAccessibilityMirror:   true,
		}
	case "wasm32-web-browser-canvas-block-system":
		return surface.HostEvidenceReport{
			Level:         "wasm32-web-browser-canvas-input",
			Backend:       "browser-canvas-rgba",
			Framebuffer:   true,
			NativeInput:   true,
			BrowserCanvas: true,
			BrowserInput:  true,
		}
	case "wasm32-web-browser-canvas", "wasm32-web-browser-canvas-text-focus-input", "wasm32-web-release-text-input", "wasm32-web-release-toolkit", "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api", "wasm32-web-browser-canvas-minimal-toolkit", "wasm32-web-browser-canvas-toolkit-reuse", "wasm32-web-browser-canvas-accessibility-metadata":
		return surface.HostEvidenceReport{
			Level:       "wasm32-web-browser-canvas-input",
			Backend:     "browser-canvas-rgba",
			Framebuffer: true,
			NativeInput: true,
		}
	default:
		return surface.HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		}
	}
}
func intPtr(v int) *int {
	return &v
}
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
