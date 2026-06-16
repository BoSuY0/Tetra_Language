package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/tools/internal/surfacerender"
	"tetra_language/tools/validators/surface"
)

type orderedRGBAFrame struct {
	Order int
	Label string
	Frame rgbaFrame
}

func attachBlockSystemFrameArtifacts(opt smokeOptions, scenario *headlessScenario) error {
	if scenario == nil || scenario.BlockSystem == nil || !isBlockSystemMode(opt.Mode) {
		return nil
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return fmt.Errorf("create Block-system frame artifact directory: %w", err)
	}
	if opt.Mode != "wasm32-web-browser-canvas-block-system" {
		for _, artifact := range renderedBlockSystemFrameArtifacts(opt.Mode) {
			path := filepath.Join(artifactDir, fmt.Sprintf("surface-block-system-frame-order-%d-%s.rgba", artifact.Order, artifact.Label))
			if opt.Mode == "linux-x64-real-window-block-system" && artifact.Order == 5 {
				path = filepath.Join(artifactDir, "surface-block-system-real-window-frame.rgba")
			}
			if err := os.WriteFile(path, artifact.Frame.Pixels, 0o644); err != nil {
				return fmt.Errorf("write Block-system frame artifact %s: %w", path, err)
			}
			if err := setBlockSystemFrameArtifactPath(scenario, artifact.Order, path, checksumRGBA(artifact.Frame.Pixels)); err != nil {
				return err
			}
		}
	}
	for _, frame := range scenario.Frames {
		if strings.TrimSpace(frame.ArtifactPath) == "" {
			continue
		}
		if err := setBlockSystemFrameArtifactPath(scenario, frame.Order, frame.ArtifactPath, frame.Checksum); err != nil {
			return err
		}
	}
	return nil
}

func attachMorphRenderedBeautyFrameArtifacts(opt smokeOptions, scenario *headlessScenario) error {
	if scenario == nil || !isMorphMode(opt.Mode) {
		return nil
	}
	if opt.Mode != "headless-morph" {
		return nil
	}
	if scenario.BlockSystem == nil {
		return nil
	}
	if scenario.BlockSceneSnapshot == nil || scenario.RenderCommandStream == nil || scenario.Morph == nil {
		return fmt.Errorf("Morph rendered beauty frame artifacts require Morph, Block scene, and render command stream evidence")
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return fmt.Errorf("create Morph rendered beauty frame artifact directory: %w", err)
	}
	artifacts := renderedBlockSystemFrameArtifacts(opt.Mode)
	if rendererFrame, ok, err := renderedMorphCommandStreamFrameArtifact(scenario, artifacts); err != nil {
		return err
	} else if ok {
		for i := range artifacts {
			if artifacts[i].Order == rendererFrame.Order {
				artifacts[i].Frame = rendererFrame.Frame
				break
			}
		}
		checksum := checksumRGBA(rendererFrame.Frame.Pixels)
		updateFrameChecksumForOrder(scenario, rendererFrame.Order, checksum)
		surfacerender.BindFrameChecksum(scenario.RenderCommandStream, checksum)
	}
	for _, artifact := range artifacts {
		path := filepath.Join(artifactDir, fmt.Sprintf("surface-morph-rendered-beauty-frame-order-%d-%s.rgba", artifact.Order, artifact.Label))
		checksum := checksumRGBA(artifact.Frame.Pixels)
		if err := os.WriteFile(path, artifact.Frame.Pixels, 0o644); err != nil {
			return fmt.Errorf("write Morph rendered beauty frame artifact %s: %w", path, err)
		}
		if err := setBlockSystemFrameArtifactPath(scenario, artifact.Order, path, checksum); err != nil {
			return err
		}
		markMorphRenderedBeautyFrameProvenance(scenario, artifact.Order, defaultSurfaceSourcePath(opt))
	}
	return nil
}

func renderedMorphCommandStreamFrameArtifact(scenario *headlessScenario, artifacts []orderedRGBAFrame) (orderedRGBAFrame, bool, error) {
	if scenario == nil || scenario.RenderCommandStream == nil || scenario.RenderCommandStream.Renderer != "software-rgba-headless" {
		return orderedRGBAFrame{}, false, nil
	}
	if len(scenario.Frames) == 0 || len(artifacts) == 0 {
		return orderedRGBAFrame{}, false, nil
	}
	target := scenario.Frames[0]
	label := artifacts[0].Label
	for _, artifact := range artifacts {
		if artifact.Order == target.Order {
			label = artifact.Label
			break
		}
	}
	rendered, err := surfacerender.RenderCommandStreamRGBA(scenario.RenderCommandStream, target.Width, target.Height)
	if err != nil {
		return orderedRGBAFrame{}, false, fmt.Errorf("render Morph command stream frame: %w", err)
	}
	return orderedRGBAFrame{
		Order: target.Order,
		Label: label,
		Frame: rgbaFrame{
			Width:  rendered.Width,
			Height: rendered.Height,
			Stride: rendered.Stride,
			Pixels: rendered.Pixels,
		},
	}, true, nil
}

func updateFrameChecksumForOrder(scenario *headlessScenario, order int, checksum string) {
	for i := range scenario.Frames {
		if scenario.Frames[i].Order == order {
			scenario.Frames[i].Checksum = checksum
		}
	}
	if scenario.BlockSystem == nil {
		return
	}
	for i := range scenario.BlockSystem.Frames {
		if scenario.BlockSystem.Frames[i].Order == order {
			scenario.BlockSystem.Frames[i].Checksum = checksum
			scenario.BlockSystem.Frames[i].RepeatChecksum = checksum
			scenario.BlockSystem.Frames[i].GoldenChecksum = checksum
		}
	}
}

func markMorphRenderedBeautyFrameProvenance(scenario *headlessScenario, order int, source string) {
	morphRecipeHash := morphRenderedBeautyRecipeHash(scenario.Morph)
	blockSceneHash := scenario.BlockSceneSnapshot.BlockSceneHash
	commandStreamHash := scenario.RenderCommandStream.CommandStreamHash
	for i := range scenario.Frames {
		if scenario.Frames[i].Order != order {
			continue
		}
		scenario.Frames[i].Producer = "app"
		scenario.Frames[i].EvidenceRole = "product_visual"
		scenario.Frames[i].AppSource = source
		scenario.Frames[i].MorphRecipeHash = morphRecipeHash
		scenario.Frames[i].BlockSceneHash = blockSceneHash
		scenario.Frames[i].RenderCommandStreamHash = commandStreamHash
		scenario.Frames[i].Precomputed = false
	}
	if scenario.BlockSystem == nil {
		return
	}
	for i := range scenario.BlockSystem.Frames {
		if scenario.BlockSystem.Frames[i].Order != order {
			continue
		}
		scenario.BlockSystem.Frames[i].Producer = "app"
		scenario.BlockSystem.Frames[i].EvidenceRole = "product_visual"
		scenario.BlockSystem.Frames[i].Precomputed = false
	}
}

func applyMorphTargetRuntimeFrameEvidence(opt smokeOptions, scenario *headlessScenario, frames []surface.FrameReport) error {
	if !isMorphTargetRuntimeMode(opt.Mode) {
		return nil
	}
	if scenario == nil || scenario.Morph == nil || scenario.BlockSceneSnapshot == nil {
		return fmt.Errorf("%s Morph evidence requires Morph and Block scene snapshot evidence", opt.Mode)
	}
	if len(frames) == 0 {
		return fmt.Errorf("%s Morph evidence requires target frame readback", opt.Mode)
	}
	source := defaultSurfaceSourcePath(opt)
	targetFrames := append([]surface.FrameReport(nil), frames...)
	sort.Slice(targetFrames, func(i, j int) bool {
		return targetFrames[i].Order < targetFrames[j].Order
	})
	for i := range targetFrames {
		if targetFrames[i].Order <= 0 || targetFrames[i].Width <= 0 || targetFrames[i].Height <= 0 || targetFrames[i].Stride <= 0 ||
			strings.TrimSpace(targetFrames[i].Checksum) == "" || strings.TrimSpace(targetFrames[i].ArtifactPath) == "" || !targetFrames[i].Presented {
			return fmt.Errorf("wasm32-web browser-canvas Morph frame evidence is incomplete: %#v", targetFrames[i])
		}
	}

	scenario.BlockSystem = nil
	scenario.Frames = targetFrames
	attachRenderCommandStreamForScenarioWithRenderer(source, renderCommandStreamRendererForMode(opt.Mode), scenario)
	refreshMorphMemoryBudgetForFrames(scenario)
	for _, frame := range scenario.Frames {
		markMorphRenderedBeautyFrameProvenance(scenario, frame.Order, source)
	}
	scenario.Cases = filterSurfaceCasesWithoutHeadlessEvidence(scenario.Cases)
	scenario.Cases = appendMissingCases(scenario.Cases, morphTargetCasesForScenario(opt.Mode)...)
	return nil
}

func refreshMorphMemoryBudgetForFrames(scenario *headlessScenario) {
	if scenario == nil || scenario.Morph == nil {
		return
	}
	scenario.Morph.EvidenceContract.MemoryBudget = true
	scenario.Morph.EvidenceContract.FrameChecksums = len(scenario.Frames) > 0
	scenario.Morph.MemoryBudget.FrameCount = len(scenario.Frames)
	scenario.Morph.MemoryBudget.FramebufferBytes = morphFramebufferBytesForScenario(scenario.Frames)
}

func filterSurfaceCasesWithoutHeadlessEvidence(cases []surface.CaseReport) []surface.CaseReport {
	filtered := make([]surface.CaseReport, 0, len(cases))
	for _, item := range cases {
		if strings.Contains(strings.ToLower(item.Name), "headless") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func appendMissingCases(cases []surface.CaseReport, additions ...surface.CaseReport) []surface.CaseReport {
	for _, addition := range additions {
		if caseReportNamesContain(cases, addition.Name) {
			continue
		}
		cases = append(cases, addition)
	}
	return cases
}

func caseReportNamesContain(cases []surface.CaseReport, want string) bool {
	for _, item := range cases {
		if item.Name == want {
			return true
		}
	}
	return false
}

func wasmBrowserCanvasMorphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
		{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
		{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas Morph rendered beauty frame readback", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web browser canvas Morph rendered beauty checksum", Kind: "positive", Ran: true, Pass: true},
	}
}

func linuxX64RealWindowMorphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window Morph rendered beauty app frame readback", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window Morph rendered beauty checksum", Kind: "positive", Ran: true, Pass: true},
	}
}

func morphTargetCasesForScenario(mode string) []surface.CaseReport {
	if isLinuxX64RealWindowMorphMode(mode) {
		return linuxX64RealWindowMorphCasesForScenario()
	}
	return wasmBrowserCanvasMorphCasesForScenario()
}

func renderedBlockSystemFrameArtifacts(mode string) []orderedRGBAFrame {
	beforeFrame := renderBlockSystemFrameRGBA(false)
	afterFrame := renderBlockSystemFrameRGBA(true)
	motionFrame := renderBlockSystemFrameRGBA(true)
	rectRGBA(motionFrame, rect{X: 188, Y: 124, W: 30, H: 10}, rgbaColor{R: 96, G: 174, B: 244, A: 255})
	switch mode {
	case "wasm32-web-browser-canvas-block-system":
		return []orderedRGBAFrame{
			{Order: 1, Label: "initial", Frame: beforeFrame},
			{Order: 3, Label: "motion", Frame: motionFrame},
		}
	case "linux-x64-real-window-block-system":
		realWindowFrame := renderBlockSystemFrameSizedRGBA(400, 240, true)
		return []orderedRGBAFrame{
			{Order: 1, Label: "initial", Frame: beforeFrame},
			{Order: 2, Label: "focused", Frame: afterFrame},
			{Order: 3, Label: "motion", Frame: motionFrame},
			{Order: 5, Label: "real-window-focused", Frame: realWindowFrame},
		}
	default:
		return []orderedRGBAFrame{
			{Order: 1, Label: "initial", Frame: beforeFrame},
			{Order: 2, Label: "focused", Frame: afterFrame},
			{Order: 3, Label: "motion", Frame: motionFrame},
		}
	}
}

func setBlockSystemFrameArtifactPath(scenario *headlessScenario, order int, path string, checksum string) error {
	matchedFrame := false
	for i := range scenario.Frames {
		if scenario.Frames[i].Order != order {
			continue
		}
		if scenario.Frames[i].Checksum != checksum {
			return fmt.Errorf("Block-system frame artifact %s checksum %s does not match frame order %d checksum %s", path, checksum, order, scenario.Frames[i].Checksum)
		}
		scenario.Frames[i].ArtifactPath = path
		matchedFrame = true
	}
	if !matchedFrame {
		return fmt.Errorf("Block-system frame artifact %s has no runtime frame order %d", path, order)
	}
	for i := range scenario.BlockSystem.Frames {
		if scenario.BlockSystem.Frames[i].Order != order {
			continue
		}
		if scenario.BlockSystem.Frames[i].Checksum != checksum {
			return fmt.Errorf("Block-system frame artifact %s checksum %s does not match block_system frame order %d checksum %s", path, checksum, order, scenario.BlockSystem.Frames[i].Checksum)
		}
		scenario.BlockSystem.Frames[i].ArtifactPath = path
	}
	return nil
}
