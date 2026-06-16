package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

func collectHeadlessRunnerTraceEvidence(sourcePath string, artifactDir string, scenario headlessScenario) (surface.ArtifactReport, surface.ArtifactScanReport, error) {
	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	if err := writeHeadlessSurfaceTrace(tracePath, sourcePath, scenario); err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	traceFrames, err := readHeadlessSurfaceTrace(tracePath)
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	if !sameFrameEvidence(traceFrames, scenario.Frames) {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, fmt.Errorf("headless Surface runner trace frames = %#v, want scenario frames %#v", traceFrames, scenario.Frames)
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir)
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	return traceArtifact, sidecarScan, nil
}
func writeHeadlessSurfaceTrace(path string, sourcePath string, scenario headlessScenario) error {
	trace := headlessSurfaceRunnerTrace{
		Schema:                          "tetra.surface.headless-runner-trace.v1",
		Source:                          sourcePath,
		Frames:                          scenario.Frames,
		Events:                          scenario.Events,
		StateTransitions:                scenario.StateTransitions,
		Components:                      scenario.Components,
		ComponentTree:                   scenario.ComponentTree,
		ComponentTreeAPI:                scenario.ComponentTreeAPI,
		BlockGraph:                      scenario.BlockGraph,
		BlockSceneSnapshot:              scenario.BlockSceneSnapshot,
		RenderCommandStream:             scenario.RenderCommandStream,
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
		Cases:                           scenario.Cases,
	}
	raw, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return fmt.Errorf("encode headless Surface runner trace: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("write headless Surface runner trace %s: %w", path, err)
	}
	return nil
}
func readHeadlessSurfaceTrace(path string) ([]surface.FrameReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read headless Surface runner trace %s: %w", path, err)
	}
	var trace headlessSurfaceRunnerTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, fmt.Errorf("decode headless Surface runner trace %s: %w", path, err)
	}
	if trace.Schema != "tetra.surface.headless-runner-trace.v1" {
		return nil, fmt.Errorf("headless Surface runner trace schema is %q, want tetra.surface.headless-runner-trace.v1", trace.Schema)
	}
	if strings.TrimSpace(trace.Source) == "" {
		return nil, fmt.Errorf("headless Surface runner trace source is required")
	}
	if len(trace.Frames) < 2 {
		return nil, fmt.Errorf("headless Surface runner trace has %d frames, want pre/post presented frames", len(trace.Frames))
	}
	for _, frame := range trace.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 || strings.TrimSpace(frame.Checksum) == "" || !frame.Presented {
			return nil, fmt.Errorf("headless Surface runner trace frame %d has incomplete presented frame evidence", frame.Order)
		}
	}
	return trace.Frames, nil
}
func sameFrameEvidence(a []surface.FrameReport, b []surface.FrameReport) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Order != b[i].Order ||
			a[i].Width != b[i].Width ||
			a[i].Height != b[i].Height ||
			a[i].Stride != b[i].Stride ||
			a[i].Checksum != b[i].Checksum ||
			a[i].Presented != b[i].Presented {
			return false
		}
	}
	return true
}
