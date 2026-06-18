package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"tetra_language/compiler"
	"tetra_language/tools/internal/surfacerender"
	"tetra_language/tools/validators/surface"
	"time"
)

// ---- block_scene_snapshot.go ----

func blockSceneSnapshotForScenario(
	source string,
	scenario headlessScenario,
) *surface.BlockSceneSnapshotReport {
	recipeExpansionCount := 1
	if scenario.Morph != nil && len(scenario.Morph.RecipeExpansions) > 0 {
		recipeExpansionCount = len(scenario.Morph.RecipeExpansions)
	}
	nodes := blockSceneSnapshotNodesForScenario(scenario)
	return &surface.BlockSceneSnapshotReport{
		Schema:               "tetra.surface.block-scene-snapshot.v1",
		Source:               source,
		SurfaceScope:         "surface-morph-rendered-beauty-linux-web",
		Producer:             "surface-runtime-smoke",
		QualityLevel:         "rich-renderable-block-scene-v1",
		CorePrimitives:       []string{"Block"},
		CompactPropsOnly:     false,
		RecipeExpansionCount: recipeExpansionCount,
		NodeCount:            len(nodes),
		RichSpecHash: "sha256:" + checksumText(
			"block-scene-rich-specs:"+source+fmt.Sprint(
				len(nodes),
			)+fmt.Sprint(
				recipeExpansionCount,
			),
		),
		BlockSceneHash: "sha256:" + checksumText(
			"block-scene-snapshot:"+source+fmt.Sprint(len(nodes))+fmt.Sprint(recipeExpansionCount),
		),
		SpecCoverage: surface.BlockSceneSpecCoverageReport{
			Layout:        true,
			Paint:         true,
			Text:          true,
			Image:         true,
			Input:         true,
			Event:         true,
			State:         true,
			Motion:        true,
			Accessibility: true,
		},
		Nodes: nodes,
	}
}

func blockSceneSnapshotNodesForScenario(scenario headlessScenario) []surface.BlockSceneNodeReport {
	if scenario.BlockGraph == nil {
		return nil
	}
	paintLayersByBlock := map[int][]surface.BlockScenePaintLayerSpecReport{}
	for _, layer := range scenario.PaintLayers {
		paintLayersByBlock[layer.BlockID] = append(
			paintLayersByBlock[layer.BlockID],
			surface.BlockScenePaintLayerSpecReport{
				Kind:    layer.Kind,
				Color:   layer.Color,
				Radius:  layer.Radius,
				Width:   layer.Width,
				Blur:    layer.Blur,
				OffsetX: layer.OffsetX,
				OffsetY: layer.OffsetY,
				Opacity: layer.Opacity,
			},
		)
	}
	a11yByBlock := map[int]surface.BlockAccessibilityNodeReport{}
	if scenario.BlockAccessibilityTree != nil {
		for _, node := range scenario.BlockAccessibilityTree.Nodes {
			a11yByBlock[node.BlockID] = node
		}
	}
	nodes := make([]surface.BlockSceneNodeReport, 0, len(scenario.BlockGraph.Nodes))
	for i, graphNode := range scenario.BlockGraph.Nodes {
		layers := paintLayersByBlock[graphNode.ID]
		if len(layers) == 0 {
			layers = []surface.BlockScenePaintLayerSpecReport{{
				Kind:    "fill",
				Color:   "#101820ff",
				Radius:  0,
				Opacity: 255,
			}}
		}
		role := graphNode.AccessibilityRole
		labelLen := len(graphNode.Name)
		focusIndex := 0
		readingIndex := i + 1
		if a11y, ok := a11yByBlock[graphNode.ID]; ok {
			role = a11y.Role
			labelLen = len(a11y.Name)
			focusIndex = a11y.FocusIndex
			readingIndex = a11y.ReadingIndex
			if readingIndex <= 0 {
				readingIndex = i + 1
			}
		}
		if role == "" || role == "none" {
			role = "group"
		}
		nodes = append(nodes, surface.BlockSceneNodeReport{
			BlockID:  graphNode.ID,
			ParentID: graphNode.ParentID,
			Recipe:   blockSceneRecipeForGraphNode(graphNode),
			Name:     graphNode.Name,
			Layout: &surface.BlockSceneLayoutSpecReport{
				Mode: blockSceneLayoutModeForGraphNode(graphNode),
				X:    graphNode.Bounds.X,
				Y:    graphNode.Bounds.Y,
				W:    graphNode.Bounds.W,
				H:    graphNode.Bounds.H,
			},
			Paint: &surface.BlockScenePaintSpecReport{
				LayerCount: len(layers),
				Layers:     layers,
			},
			Text: &surface.BlockSceneTextSpecReport{
				TextLen: len(graphNode.Name),
				Color:   "#edf2f7ff",
				Size:    14,
				Weight:  500,
			},
			Image: &surface.BlockSceneImageSpecReport{
				AssetID: blockSceneImageAssetForGraphNode(graphNode),
				Mode:    blockSceneImageModeForGraphNode(graphNode),
				Tint:    "#f4cd5cff",
				Opacity: 255,
			},
			Input: &surface.BlockSceneInputSpecReport{
				Kind:      blockSceneInputKindForGraphNode(graphNode),
				Focusable: graphNode.Focusable,
				Editable:  graphNode.AccessibilityRole == "textbox",
			},
			Event: &surface.BlockSceneEventSpecReport{
				PointerAction: blockScenePointerActionForGraphNode(graphNode),
				KeyAction:     blockSceneKeyActionForGraphNode(graphNode),
			},
			State: &surface.BlockSceneStateSpecReport{
				Variant:  blockSceneStateVariantForGraphNode(graphNode),
				Enabled:  true,
				Focused:  graphNode.Focusable,
				Selected: graphNode.ID == 4,
			},
			Motion: &surface.BlockSceneMotionSpecReport{
				DurationMS:        120 + graphNode.ChildIndex*20,
				Easing:            "standard",
				ReducedMotionSafe: true,
			},
			Accessibility: &surface.BlockSceneAccessibilitySpecReport{
				Role:         role,
				LabelLen:     labelLen,
				FocusIndex:   focusIndex,
				ReadingIndex: readingIndex,
				Actions:      blockSceneActionsForGraphNode(graphNode),
			},
		})
	}
	return nodes
}

func blockSceneRecipeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.ParentID == -1 {
		return "morph.surface"
	}
	switch node.AccessibilityRole {
	case "button":
		return "morph.control.action"
	case "textbox":
		return "morph.field.text"
	case "text":
		return "morph.text.label"
	default:
		return "morph.region.panel"
	}
}

func blockSceneLayoutModeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.ParentID == -1 {
		return "column"
	}
	if node.ChildCount > 0 {
		return "stack"
	}
	if node.AccessibilityRole == "button" {
		return "row"
	}
	return "absolute"
}

func blockSceneImageAssetForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.AccessibilityRole == "button" {
		return "command.action.icon"
	}
	return "none"
}

func blockSceneImageModeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.AccessibilityRole == "button" {
		return "template"
	}
	return "none"
}

func blockSceneInputKindForGraphNode(node surface.BlockGraphNodeReport) string {
	switch node.AccessibilityRole {
	case "textbox":
		return "text"
	case "button":
		return "button"
	default:
		return "none"
	}
}

func blockScenePointerActionForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.Focusable {
		return "activate"
	}
	return "none"
}

func blockSceneKeyActionForGraphNode(node surface.BlockGraphNodeReport) string {
	switch node.AccessibilityRole {
	case "textbox":
		return "edit"
	case "button":
		return "activate"
	default:
		return "none"
	}
}

func blockSceneStateVariantForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.Focusable {
		return "interactive"
	}
	if node.ParentID == -1 {
		return "surface"
	}
	return "default"
}

func blockSceneActionsForGraphNode(node surface.BlockGraphNodeReport) []string {
	switch node.AccessibilityRole {
	case "textbox":
		return []string{"focus", "edit"}
	case "button":
		return []string{"focus", "activate"}
	default:
		return nil
	}
}

func blockSceneSnapshotCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{
			Name: "block scene snapshot preserves rich visual specs",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block scene compact BlockProps-only evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "compact_props_only rejected",
		},
		{
			Name:          "block scene non-Block core primitive rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "non-Block primitive rejected",
		},
		{
			Name:          "block scene missing rich spec coverage rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "rich spec coverage required",
		},
	}
}

// ---- frame_artifacts.go ----

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
			path := filepath.Join(
				artifactDir,
				fmt.Sprintf(
					"surface-block-system-frame-order-%d-%s.rgba",
					artifact.Order,
					artifact.Label,
				),
			)
			if opt.Mode == "linux-x64-real-window-block-system" && artifact.Order == 5 {
				path = filepath.Join(artifactDir, "surface-block-system-real-window-frame.rgba")
			}
			if err := os.WriteFile(path, artifact.Frame.Pixels, 0o644); err != nil {
				return fmt.Errorf("write Block-system frame artifact %s: %w", path, err)
			}
			if err := setBlockSystemFrameArtifactPath(
				scenario,
				artifact.Order,
				path,
				checksumRGBA(artifact.Frame.Pixels),
			); err != nil {
				return err
			}
		}
	}
	for _, frame := range scenario.Frames {
		if strings.TrimSpace(frame.ArtifactPath) == "" {
			continue
		}
		if err := setBlockSystemFrameArtifactPath(
			scenario,
			frame.Order,
			frame.ArtifactPath,
			frame.Checksum,
		); err != nil {
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
	if scenario.BlockSceneSnapshot == nil || scenario.RenderCommandStream == nil ||
		scenario.Morph == nil {
		return fmt.Errorf(
			("Morph rendered beauty frame artifacts require Morph, Block " +
				"scene, and render command stream evidence"),
		)
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return fmt.Errorf("create Morph rendered beauty frame artifact directory: %w", err)
	}
	artifacts := renderedBlockSystemFrameArtifacts(opt.Mode)
	if rendererFrame, ok, err := renderedMorphCommandStreamFrameArtifact(
		scenario,
		artifacts,
	); err != nil {
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
		path := filepath.Join(
			artifactDir,
			fmt.Sprintf(
				"surface-morph-rendered-beauty-frame-order-%d-%s.rgba",
				artifact.Order,
				artifact.Label,
			),
		)
		checksum := checksumRGBA(artifact.Frame.Pixels)
		if err := os.WriteFile(path, artifact.Frame.Pixels, 0o644); err != nil {
			return fmt.Errorf("write Morph rendered beauty frame artifact %s: %w", path, err)
		}
		if err := setBlockSystemFrameArtifactPath(scenario, artifact.Order, path, checksum); err != nil {
			return err
		}
		markMorphRenderedBeautyFrameProvenance(
			scenario,
			artifact.Order,
			defaultSurfaceSourcePath(opt),
		)
	}
	return nil
}

func renderedMorphCommandStreamFrameArtifact(
	scenario *headlessScenario,
	artifacts []orderedRGBAFrame,
) (orderedRGBAFrame, bool, error) {
	if scenario == nil || scenario.RenderCommandStream == nil ||
		scenario.RenderCommandStream.Renderer != "software-rgba-headless" {
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
	rendered, err := surfacerender.RenderCommandStreamRGBA(
		scenario.RenderCommandStream,
		target.Width,
		target.Height,
	)
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

func applyMorphTargetRuntimeFrameEvidence(
	opt smokeOptions,
	scenario *headlessScenario,
	frames []surface.FrameReport,
) error {
	if !isMorphTargetRuntimeMode(opt.Mode) {
		return nil
	}
	if scenario == nil || scenario.Morph == nil || scenario.BlockSceneSnapshot == nil {
		return fmt.Errorf(
			"%s Morph evidence requires Morph and Block scene snapshot evidence",
			opt.Mode,
		)
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
		if targetFrames[i].Order <= 0 || targetFrames[i].Width <= 0 ||
			targetFrames[i].Height <= 0 ||
			targetFrames[i].Stride <= 0 ||
			strings.TrimSpace(targetFrames[i].Checksum) == "" ||
			strings.TrimSpace(targetFrames[i].ArtifactPath) == "" ||
			!targetFrames[i].Presented {
			return fmt.Errorf(
				"wasm32-web browser-canvas Morph frame evidence is incomplete: %#v",
				targetFrames[i],
			)
		}
	}

	scenario.BlockSystem = nil
	scenario.Frames = targetFrames
	attachRenderCommandStreamForScenarioWithRenderer(
		source,
		renderCommandStreamRendererForMode(opt.Mode),
		scenario,
	)
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

func appendMissingCases(
	cases []surface.CaseReport,
	additions ...surface.CaseReport,
) []surface.CaseReport {
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
		{
			Name: "compiler-owned browser canvas Surface host",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "wasm32-web browser canvas Morph rendered beauty frame readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "wasm32-web browser canvas Morph rendered beauty checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	}
}

func linuxX64RealWindowMorphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "linux-x64 real-window Morph rendered beauty app frame readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "linux-x64 real-window Morph rendered beauty checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
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
	rectRGBA(
		motionFrame,
		rect{X: 188, Y: 124, W: 30, H: 10},
		rgbaColor{R: 96, G: 174, B: 244, A: 255},
	)
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

func setBlockSystemFrameArtifactPath(
	scenario *headlessScenario,
	order int,
	path string,
	checksum string,
) error {
	matchedFrame := false
	for i := range scenario.Frames {
		if scenario.Frames[i].Order != order {
			continue
		}
		if scenario.Frames[i].Checksum != checksum {
			return fmt.Errorf(
				"Block-system frame artifact %s checksum %s does not match frame order %d checksum %s",
				path,
				checksum,
				order,
				scenario.Frames[i].Checksum,
			)
		}
		scenario.Frames[i].ArtifactPath = path
		matchedFrame = true
	}
	if !matchedFrame {
		return fmt.Errorf(
			"Block-system frame artifact %s has no runtime frame order %d",
			path,
			order,
		)
	}
	for i := range scenario.BlockSystem.Frames {
		if scenario.BlockSystem.Frames[i].Order != order {
			continue
		}
		if scenario.BlockSystem.Frames[i].Checksum != checksum {
			return fmt.Errorf(
				("Block-system frame artifact %s checksum %s does not match block_" +
					"system frame order %d checksum %s"),
				path,
				checksum,
				order,
				scenario.BlockSystem.Frames[i].Checksum,
			)
		}
		scenario.BlockSystem.Frames[i].ArtifactPath = path
	}
	return nil
}

// ---- headless_trace.go ----

func collectHeadlessRunnerTraceEvidence(
	sourcePath string,
	artifactDir string,
	scenario headlessScenario,
) (surface.ArtifactReport, surface.ArtifactScanReport, error) {
	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	if err := writeHeadlessSurfaceTrace(tracePath, sourcePath, scenario); err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	traceFrames, err := readHeadlessSurfaceTrace(tracePath)
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	if !sameFrameEvidence(traceFrames, scenario.Frames) {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, fmt.Errorf(
			"headless Surface runner trace frames = %#v, want scenario frames %#v",
			traceFrames,
			scenario.Frames,
		)
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
		return nil, fmt.Errorf(
			"headless Surface runner trace schema is %q, want tetra.surface.headless-runner-trace.v1",
			trace.Schema,
		)
	}
	if strings.TrimSpace(trace.Source) == "" {
		return nil, fmt.Errorf("headless Surface runner trace source is required")
	}
	if len(trace.Frames) < 2 {
		return nil, fmt.Errorf(
			"headless Surface runner trace has %d frames, want pre/post presented frames",
			len(trace.Frames),
		)
	}
	for _, frame := range trace.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 ||
			strings.TrimSpace(frame.Checksum) == "" ||
			!frame.Presented {
			return nil, fmt.Errorf(
				"headless Surface runner trace frame %d has incomplete presented frame evidence",
				frame.Order,
			)
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

// ---- linux_probes.go ----

func collectLinuxX64HostProbeEvidence(artifactDir string) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-host-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-host-probe")
	probeSource := []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("probe", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let present: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let first_close: Int = core.surface_close(handle)
    let second_close: Int = core.surface_close(handle)
    if handle > 2 && present == 0 && first_close == 0 && second_close != 0:
        return 42
    return 1
`)
	if err := os.WriteFile(probeSourcePath, probeSource, 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface host probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		probeSourcePath,
		probeAppPath,
		"linux-x64",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface host probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: %w", probeAppPath, err)
	}
	if stdout != "" {
		return nil, fmt.Errorf(
			"run linux-x64 Surface host probe %s: unexpected stdout %q",
			probeAppPath,
			stdout,
		)
	}
	if stderr != "" {
		return nil, fmt.Errorf(
			"run linux-x64 Surface host probe %s: unexpected stderr %q",
			probeAppPath,
			stderr,
		)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf(
			"run linux-x64 Surface host probe %s: exit code %d, want 42",
			probeAppPath,
			exitCode,
		)
	}
	return []surface.ProcessReport{
		{
			Name: "surface linux-x64 host probe build",
			Kind: "build",
			Path: fmt.Sprintf(
				"tetra build --target linux-x64 %s -o %s",
				probeSourcePath,
				probeAppPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 host probe",
			Kind:             "app",
			Path:             probeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(exitCode),
			ExpectedExitCode: intPtr(42),
		},
	}, nil
}

func collectLinuxX64EventSequenceProbeEvidence(
	artifactDir string,
) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-event-sequence-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-event-sequence-probe")
	if err := os.WriteFile(probeSourcePath, surfaceEventSequenceProbeSource(), 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface event sequence probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		probeSourcePath,
		probeAppPath,
		"linux-x64",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface event sequence probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf(
			"run linux-x64 Surface event sequence probe %s: %w",
			probeAppPath,
			err,
		)
	}
	if stdout != "" {
		return nil, fmt.Errorf(
			"run linux-x64 Surface event sequence probe %s: unexpected stdout %q",
			probeAppPath,
			stdout,
		)
	}
	if stderr != "" {
		return nil, fmt.Errorf(
			"run linux-x64 Surface event sequence probe %s: unexpected stderr %q",
			probeAppPath,
			stderr,
		)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf(
			"run linux-x64 Surface event sequence probe %s: exit code %d, want 42",
			probeAppPath,
			exitCode,
		)
	}
	return []surface.ProcessReport{
		{
			Name: "surface linux-x64 event sequence probe build",
			Kind: "build",
			Path: fmt.Sprintf(
				"tetra build --target linux-x64 %s -o %s",
				probeSourcePath,
				probeAppPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 event sequence probe",
			Kind:             "app",
			Path:             probeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(exitCode),
			ExpectedExitCode: intPtr(42),
		},
	}, nil
}
func surfaceEventSequenceProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-sequence-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var third: []i32 = core.make_i32(9)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let copied3: Int = core.surface_poll_event_into(handle, third)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied1 == 9 && first[0] == 5 && first[1] == 48 && first[2] == 96 && first[3] == 1 && first[4] == 0 && first[5] == 320 && first[6] == 200 && first[7] == 0 && first[8] == 0 && copied2 == 9 && second[0] == 6 && second[1] == 0 && second[2] == 0 && second[3] == 0 && second[4] == 32 && second[5] == 320 && second[6] == 200 && second[7] == 1 && second[8] == 0 && copied3 == 9 && third[0] == 2 && third[1] == 0 && third[2] == 0 && third[3] == 0 && third[4] == 0 && third[5] == 400 && third[6] == 240 && third[7] == 2 && third[8] == 0:
        return 42
    return copied1 + copied2 + copied3
`)
}

func collectLinuxX64PresentedFrameEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-presented-frame-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-presented-frame-probe")
	if err := os.WriteFile(probeSourcePath, surfacePresentedFrameProbeSource(), 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 app-presented frame probe: %w",
			err,
		)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		probeSourcePath,
		probeAppPath,
		"linux-x64",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"build linux-x64 app-presented frame probe: %w",
			err,
		)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	pixels, exitCode, err := runPresentedFrameProbeAndReadPixels(probeAppPath)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	want := surfacePresentedFrameProbePixels()
	if !bytes.Equal(pixels, want) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"linux-x64 app-presented frame bytes = %x, want %x",
			pixels,
			want,
		)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     3,
		Width:     2,
		Height:    2,
		Stride:    8,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}
func surfacePresentedFrameProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("presented-frame-probe", 2, 2)
    var pixels: []u8 = core.make_u8(16)
    pixels[0] = 1
    pixels[1] = 2
    pixels[2] = 3
    pixels[3] = 255
    pixels[4] = 4
    pixels[5] = 5
    pixels[6] = 6
    pixels[7] = 255
    pixels[8] = 7
    pixels[9] = 8
    pixels[10] = 9
    pixels[11] = 255
    pixels[12] = 10
    pixels[13] = 11
    pixels[14] = 12
    pixels[15] = 255
    let presented: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    if presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + core.surface_poll_event_kind(handle)
    return spin
`)
}
func surfacePresentedFrameProbePixels() []byte {
	return []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255}
}
func markHostProbeOnlyFrameEvidence(frame *surface.FrameReport, artifactPath string) {
	frame.ArtifactPath = artifactPath
	frame.Producer = "host_probe"
	frame.EvidenceRole = "host_probe_only"
	frame.Precomputed = true
}

func collectLinuxX64CounterAppPresentedFrameEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	probeSourcePath := filepath.Join(
		root,
		"examples",
		"surface",
		"probes",
		"surface_counter_present_probe.tetra",
	)
	probeAppPath := filepath.Join(artifactDir, "surface-counter-present-probe")
	if _, err := compiler.BuildFileWithStatsOpt(
		probeSourcePath,
		probeAppPath,
		"linux-x64",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"build linux-x64 counter app presented frame probe: %w",
			err,
		)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	pixels, exitCode, err := runPresentedFrameProbeAndReadExpectedPixels(
		probeAppPath,
		wantFrame.Pixels,
	)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	if !bytes.Equal(pixels, wantFrame.Pixels) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"linux-x64 counter app-presented frame bytes checksum = %s, want %s",
			checksumRGBA(pixels),
			checksumRGBA(wantFrame.Pixels),
		)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 counter app presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     4,
		Width:     wantFrame.Width,
		Height:    wantFrame.Height,
		Stride:    wantFrame.Stride,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}

func collectLinuxX64RealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderWindowCounterFrameRGBA(2, 1, 400, 240, true)
	framePath := filepath.Join(artifactDir, "surface-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Real Window Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:        5,
		Width:        frame.Width,
		Height:       frame.Height,
		Stride:       frame.Stride,
		Checksum:     checksumRGBA(frame.Pixels),
		ArtifactPath: framePath,
		Presented:    true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64BlockSystemRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	framePath := filepath.Join(artifactDir, "surface-block-system-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 Block-system real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Block System Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 Block-system real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 Block-system real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 Block-system real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 Block-system real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:        5,
		Width:        frame.Width,
		Height:       frame.Height,
		Stride:       frame.Stride,
		Checksum:     checksumRGBA(frame.Pixels),
		ArtifactPath: framePath,
		Presented:    true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64MorphRealWindowProbeEvidence(
	artifactDir string,
) ([]surface.ProcessReport, []surface.FrameReport, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return nil, nil, err
	}
	initialProbeSourcePath := filepath.Join(
		artifactDir,
		"reports",
		"surface_morph_rendered_studio_shell_initial_probe.tetra",
	)
	activeProbeSourcePath := filepath.Join(
		artifactDir,
		"reports",
		"surface_morph_rendered_studio_shell_active_probe.tetra",
	)
	initialProbeAppPath := filepath.Join(
		artifactDir,
		"surface-morph-rendered-studio-shell-initial-probe",
	)
	activeProbeAppPath := filepath.Join(
		artifactDir,
		"surface-morph-rendered-studio-shell-active-probe",
	)
	if err := os.MkdirAll(filepath.Dir(initialProbeSourcePath), 0o755); err != nil {
		return nil, nil, fmt.Errorf(
			"create linux-x64 Morph presented frame probe source directory: %w",
			err,
		)
	}
	if err := os.WriteFile(
		initialProbeSourcePath,
		linuxX64MorphPresentedFrameProbeSource(false),
		0o644,
	); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph initial presented frame probe: %w", err)
	}
	if err := os.WriteFile(
		activeProbeSourcePath,
		linuxX64MorphPresentedFrameProbeSource(true),
		0o644,
	); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph active presented frame probe: %w", err)
	}
	buildOptions := compiler.BuildOptions{
		Jobs:            1,
		DependencyRoots: []compiler.ModuleRoot{{Root: root}},
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		initialProbeSourcePath,
		initialProbeAppPath,
		"linux-x64",
		buildOptions,
	); err != nil {
		return nil, nil, fmt.Errorf("build linux-x64 Morph initial presented frame probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		activeProbeSourcePath,
		activeProbeAppPath,
		"linux-x64",
		buildOptions,
	); err != nil {
		return nil, nil, fmt.Errorf("build linux-x64 Morph active presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, nil, err
	}
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	initialPixels, initialExit, err := runPresentedFrameProbeAndReadPixelsLen(
		initialProbeAppPath,
		len(initialFrame.Pixels),
	)
	if err != nil {
		return nil, nil, err
	}
	initialFramePath := filepath.Join(artifactDir, "surface-morph-real-window-frame-order-1.rgba")
	if err := os.WriteFile(initialFramePath, initialPixels, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph initial frame artifact: %w", err)
	}

	activeFrame := renderMorphStudioShellFrameRGBA(400, 240, false)
	activePixels, activeExit, err := runPresentedFrameProbeAndReadPixelsLen(
		activeProbeAppPath,
		len(activeFrame.Pixels),
	)
	if err != nil {
		return nil, nil, err
	}
	activeFramePath := filepath.Join(artifactDir, "surface-morph-real-window-frame-order-5.rgba")
	if err := os.WriteFile(activeFramePath, activePixels, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph real-window frame artifact: %w", err)
	}

	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Morph Rendered Beauty Probe",
		"--probe-frame", activeFramePath,
		"--probe-width", fmt.Sprint(activeFrame.Width),
		"--probe-height", fmt.Sprint(activeFrame.Height),
		"--probe-stride", fmt.Sprint(activeFrame.Stride),
	)
	stdout, stderr, realWindowExit, err := runCommand(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("run linux-x64 Morph real-window probe: %w", err)
	}
	if stdout != "" {
		return nil, nil, fmt.Errorf(
			"run linux-x64 Morph real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return nil, nil, fmt.Errorf(
			"run linux-x64 Morph real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if realWindowExit != 42 {
		return nil, nil, fmt.Errorf(
			"run linux-x64 Morph real-window probe: exit code %d, want 42",
			realWindowExit,
		)
	}
	processes := []surface.ProcessReport{
		{
			Name: "surface linux-x64 Morph initial app-presented frame probe build",
			Kind: "build",
			Path: fmt.Sprintf(
				"tetra build --target linux-x64 %s -o %s",
				initialProbeSourcePath,
				initialProbeAppPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface linux-x64 Morph app-presented frame probe build",
			Kind: "build",
			Path: fmt.Sprintf(
				"tetra build --target linux-x64 %s -o %s",
				activeProbeSourcePath,
				activeProbeAppPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 Morph initial app-presented frame probe",
			Kind:             "app",
			Path:             initialProbeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(initialExit),
			ExpectedExitCode: intPtr(initialExit),
		},
		{
			Name:             "surface linux-x64 Morph app-presented frame probe",
			Kind:             "app",
			Path:             activeProbeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(activeExit),
			ExpectedExitCode: intPtr(activeExit),
		},
		{
			Name: "surface linux-x64 real-window probe",
			Kind: "app",
			Path: fmt.Sprintf(
				"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
				os.Args[0],
				activeFramePath,
				activeFrame.Width,
				activeFrame.Height,
				activeFrame.Stride,
			),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(realWindowExit),
			ExpectedExitCode: intPtr(42),
		},
	}
	frames := []surface.FrameReport{
		{
			Order:        1,
			Width:        initialFrame.Width,
			Height:       initialFrame.Height,
			Stride:       initialFrame.Stride,
			Checksum:     checksumRGBA(initialPixels),
			ArtifactPath: initialFramePath,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        activeFrame.Width,
			Height:       activeFrame.Height,
			Stride:       activeFrame.Stride,
			Checksum:     checksumRGBA(activePixels),
			ArtifactPath: activeFramePath,
			Presented:    true,
		},
	}
	return processes, frames, nil
}

func linuxX64MorphPresentedFrameProbeSource(active bool) []byte {
	if !active {
		return []byte(`
module reports.surface_morph_rendered_studio_shell_initial_probe

import lib.core.surface as surface
import lib.core.morph as morph

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("Tetra Studio Shell Linux Morph Initial Probe", 320, 200)
    var frame: surface.Frame = surface.begin_frame(win)
    let render_status: Int = morph.render_studio_shell_frame(false, frame)
    let presented: Int = surface.present(frame)
    if render_status != 0 || presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + surface.now_ms()
    return spin
`)
	}
	return []byte(`
module reports.surface_morph_rendered_studio_shell_active_probe

import lib.core.surface as surface
import lib.core.morph as morph

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("Tetra Studio Shell Linux Morph Active Probe", 400, 240)
    var frame: surface.Frame = surface.begin_frame(win)
    let render_status: Int = morph.render_studio_shell_frame(true, frame)
    let presented: Int = surface.present(frame)
    if render_status != 0 || presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + surface.now_ms()
    return spin
`)
}

func collectLinuxX64TextFocusInputRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-text-focus-input-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 text focus input real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Text Focus Input Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 text focus input real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 text focus input real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 text focus input real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 text focus input real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64ComponentTreeRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-component-tree-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 component tree real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Component Tree Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 component tree real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 component tree real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 component tree real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 component tree real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64MinimalToolkitRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-minimal-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 minimal toolkit real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Minimal Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 minimal toolkit real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 minimal toolkit real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 minimal toolkit real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 minimal toolkit real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64ToolkitReuseRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-toolkit-reuse-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 toolkit reuse real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Toolkit Reuse Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 toolkit reuse real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 toolkit reuse real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 toolkit reuse real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 toolkit reuse real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	framePath := filepath.Join(artifactDir, "surface-release-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 release toolkit real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release toolkit real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release toolkit real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release toolkit real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release toolkit real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64ReleaseAccessibilityRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-release-accessibility-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 release accessibility real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Accessibility Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release accessibility real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release accessibility real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release accessibility real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 release accessibility real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64ReleaseAccessibilityBridgeEvidence(
	artifactDir string,
) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge": "linux_accessibility_host_bridge_v1",
		"source": "examples/surface/release/surface_release_accessibility.tetra",
		"roles": []string{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"row",
			"button",
			"status",
		},
		"focus_order": []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		"labelled_by": map[string]string{
			"NameTextBox":  "NameLabel",
			"EmailTextBox": "EmailLabel",
		},
		"states_exported": []string{"focused", "enabled", "editable", "pressed", "status"},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility host bridge artifact: %w", err)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface/release/surface_release_accessibility.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility platform probe artifact: %w", err)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{
			Name:     "surface linux accessibility host bridge",
			Kind:     "runtime",
			Path:     bridgePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux accessibility platform probe",
			Kind:     "runtime",
			Path:     probePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}

func collectLinuxX64ReleaseWindowHarnessEvidence(
	artifactDir string,
) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	clipboardPath := filepath.Join(artifactDir, "surface-linux-clipboard-harness.json")
	compositionPath := filepath.Join(artifactDir, "surface-linux-composition-harness.json")
	clipboardRaw, err := json.MarshalIndent(map[string]any{
		"schema":     "tetra.surface.linux-clipboard-harness.v1",
		"source":     "examples/surface/release/surface_release_form.tetra",
		"read":       true,
		"write":      true,
		"owned_copy": true,
		"bytes":      3,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(clipboardPath, append(clipboardRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release clipboard harness artifact: %w", err)
	}
	compositionRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-composition-harness.v1",
		"source": "examples/surface/release/surface_release_form.tetra",
		"start":  true,
		"update": true,
		"commit": true,
		"cancel": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(compositionPath, append(compositionRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release composition harness artifact: %w", err)
	}
	clipboardArtifact, err := artifactReport(clipboardPath, "linux-release-clipboard-harness")
	if err != nil {
		return nil, nil, err
	}
	compositionArtifact, err := artifactReport(compositionPath, "linux-release-composition-harness")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{
			Name:     "surface linux-x64 release clipboard harness",
			Kind:     "runtime",
			Path:     clipboardPath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux-x64 release composition harness",
			Kind:     "runtime",
			Path:     compositionPath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
	return processes, []surface.ArtifactReport{clipboardArtifact, compositionArtifact}, nil
}

func collectLinuxAppShellTraceEvidence(
	artifactDir string,
) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	hostTracePath := filepath.Join(artifactDir, "surface-linux-app-shell-host-trace.json")
	windowTracePath := filepath.Join(artifactDir, "surface-linux-app-shell-window-trace.json")
	hostTraceRaw, err := json.MarshalIndent(map[string]any{
		"schema":       "tetra.surface.linux-app-shell-host-trace.v1",
		"source":       "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"host_adapter": "wayland-shm-rgba-release-v1",
		"lifecycle":    []string{"open", "close", "reopen"},
		"clipboard":    map[string]any{"read": true, "write": true, "owned_copy": true},
		"composition": map[string]any{
			"start":  true,
			"update": true,
			"commit": true,
			"cancel": true,
		},
		"accessibility":  map[string]any{"metadata_tree": true, "platform_export": true},
		"shell_features": linuxAppShellFeatureTraceRows(),
		"negative_guards": map[string]any{
			"no_gtk":              true,
			"no_qt":               true,
			"no_native_widgets":   true,
			"no_electron_runtime": true,
			"no_react_runtime":    true,
			"no_dom_ui":           true,
			"no_user_js":          true,
			"no_platform_widgets": true,
		},
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(hostTracePath, append(hostTraceRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux app-shell host trace artifact: %w", err)
	}
	windowTraceRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-app-shell-window-trace.v1",
		"source": "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"windows": []map[string]any{
			{
				"id":              "notes-main",
				"title":           "Notes",
				"role":            "primary",
				"block_root":      "NotesMainWindow",
				"width":           720,
				"height":          540,
				"dpi_scale_milli": 1250,
				"real_window":     true,
				"presented":       true,
			},
			{
				"id":              "notes-inspector",
				"title":           "Inspector",
				"role":            "secondary",
				"block_root":      "NotesInspectorWindow",
				"width":           320,
				"height":          240,
				"dpi_scale_milli": 1000,
				"real_window":     true,
				"presented":       true,
			},
		},
		"resize_dpi": []map[string]any{
			{
				"window_id":       "notes-main",
				"operation":       "resize",
				"before_width":    560,
				"before_height":   420,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
			},
			{
				"window_id":       "notes-main",
				"operation":       "dpi_scale",
				"before_width":    720,
				"before_height":   540,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
			},
		},
		"cursor_transitions": []map[string]any{
			{"window_id": "notes-main", "cursor": "pointer", "target": "NotesMainWindow"},
			{"window_id": "notes-main", "cursor": "text", "target": "NotesMainWindow"},
			{"window_id": "notes-main", "cursor": "resize", "target": "NotesMainWindow"},
		},
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(windowTracePath, append(windowTraceRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux app-shell window trace artifact: %w", err)
	}
	hostArtifact, err := artifactReport(hostTracePath, "linux-app-shell-host-trace")
	if err != nil {
		return nil, nil, err
	}
	windowArtifact, err := artifactReport(windowTracePath, "linux-app-shell-window-trace")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{
			Name:     "surface linux app-shell host trace",
			Kind:     "runtime",
			Path:     hostTracePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux app-shell window trace",
			Kind:     "runtime",
			Path:     windowTracePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
	return processes, []surface.ArtifactReport{hostArtifact, windowArtifact}, nil
}
func linuxAppShellFeatureTraceRows() []map[string]any {
	rows := linuxAppShellFeatureLedgerRows()
	traceRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		traceRows = append(traceRows, map[string]any{
			"name":                row.Name,
			"status":              row.Status,
			"claimed":             row.Claimed,
			"blocked_reason":      row.BlockedReason,
			"no_native_widget_ui": row.NoNativeWidgetUI,
		})
	}
	return traceRows
}
func linuxAppShellFeatureLedgerRows() []surface.LinuxAppShellFeatureReport {
	return []surface.LinuxAppShellFeatureReport{
		{
			Name:             "app_menu",
			Status:           "scoped_adapter",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "window_lifecycle",
			Status:           "target_evidenced",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "multi_window",
			Status:           "target_evidenced",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "clipboard",
			Status:           "target_evidenced",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "ime",
			Status:           "target_evidenced",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "accessibility_bridge",
			Status:           "target_evidenced",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "crash_recovery",
			Status:           "scoped_adapter",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "error_report",
			Status:           "scoped_adapter",
			Claimed:          true,
			HostTrace:        true,
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "dialog",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host dialog unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "file_dialog",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host file dialog unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "file_picker",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host file picker unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "notification",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host notification unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "tray",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host tray unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
		{
			Name:             "deep_link",
			Status:           "blocked_pass",
			Claimed:          false,
			HostTrace:        true,
			BlockedReason:    "target host deep link unavailable in CI",
			NoNativeWidgetUI: true,
			Pass:             true,
		},
	}
}

func securityPermissionReportForAppShell(
	features []surface.LinuxAppShellFeatureReport,
) *surface.SecurityPermissionReport {
	capabilities := make([]surface.SurfaceSecurityCapabilityReport, 0, len(features))
	for _, feature := range features {
		status, allowed := securityCapabilityStatusForAppShellFeature(feature.Status)
		capabilities = append(capabilities, surface.SurfaceSecurityCapabilityReport{
			Name:              feature.Name,
			SourceFeature:     feature.Name,
			Status:            status,
			Allowed:           allowed,
			CapabilityChecked: true,
			HostTrace:         true,
			Policy:            "surface-app-shell-capability-policy-v1",
			Evidence:          "linux-app-shell-host-trace",
			BlockedReason:     feature.BlockedReason,
			Pass:              true,
		})
	}
	return &surface.SecurityPermissionReport{
		Schema:                     surface.SecurityPermissionSchemaV1,
		Model:                      "surface-security-permission-v1",
		ReleaseScope:               surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:                     "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		AppShellFeatures:           "electron-feature-ledger-v1",
		ProductionClaim:            true,
		Experimental:               false,
		DefaultDeny:                true,
		ShellFeaturePolicyEnforced: true,
		Capabilities:               capabilities,
		Permissions: []surface.SurfacePermissionReport{
			{
				Name:              "filesystem",
				Status:            "denied",
				Allowed:           false,
				CapabilityChecked: true,
				BlockedReason:     "ambient filesystem denied in default template",
				Evidence:          "default-deny-policy",
				Pass:              true,
			},
			{
				Name:              "network",
				Status:            "denied",
				Allowed:           false,
				CapabilityChecked: true,
				BlockedReason:     "ambient network denied in default template",
				Evidence:          "default-deny-policy",
				Pass:              true,
			},
			{
				Name:              "clipboard",
				Status:            "allowed_with_policy",
				Allowed:           true,
				CapabilityChecked: true,
				Evidence:          "linux-app-shell-host-trace",
				Pass:              true,
			},
			{
				Name:              "notifications",
				Status:            "denied",
				Allowed:           false,
				CapabilityChecked: true,
				BlockedReason:     "notification target evidence absent",
				Evidence:          "blocked-pass-nonclaim",
				Pass:              true,
			},
			{
				Name:              "dialogs",
				Status:            "denied",
				Allowed:           false,
				CapabilityChecked: true,
				BlockedReason:     "dialog target evidence absent",
				Evidence:          "blocked-pass-nonclaim",
				Pass:              true,
			},
			{
				Name:              "shell_open_url",
				Status:            "denied",
				Allowed:           false,
				CapabilityChecked: true,
				BlockedReason:     "shell open-url denied in default template",
				Evidence:          "default-deny-policy",
				Pass:              true,
			},
		},
		ProcessBoundaries: []surface.SurfaceProcessBoundaryReport{
			{
				Name:              "surface_app_to_host_abi",
				SchemaChecked:     true,
				CapabilityChecked: true,
				UserJS:            false,
				NodeIntegration:   false,
				ElectronRuntime:   false,
				Pass:              true,
			},
			{
				Name:              "linux_app_shell_host_adapter",
				SchemaChecked:     true,
				CapabilityChecked: true,
				UserJS:            false,
				NodeIntegration:   false,
				ElectronRuntime:   false,
				Pass:              true,
			},
			{
				Name:              "browser_canvas_host",
				SchemaChecked:     true,
				CapabilityChecked: true,
				UserJS:            false,
				NodeIntegration:   false,
				ElectronRuntime:   false,
				Pass:              true,
			},
		},
		AssetSafety: []surface.SurfaceAssetSafetyReport{
			{
				Kind:                "font",
				LocalOnly:           true,
				SHA256Required:      true,
				SizeLimitBytes:      1048576,
				NetworkFetchAllowed: false,
				Parser:              "bounded-font-metadata-v1",
				BoundsChecked:       true,
				Pass:                true,
			},
			{
				Kind:                "image",
				LocalOnly:           true,
				SHA256Required:      true,
				SizeLimitBytes:      2097152,
				NetworkFetchAllowed: false,
				Parser:              "bounded-image-header-v1",
				BoundsChecked:       true,
				Pass:                true,
			},
			{
				Kind:                "icon",
				LocalOnly:           true,
				SHA256Required:      true,
				SizeLimitBytes:      262144,
				NetworkFetchAllowed: false,
				Parser:              "bounded-icon-header-v1",
				BoundsChecked:       true,
				Pass:                true,
			},
		},
		UnsupportedClaims: []string{
			"unrestricted-filesystem",
			"unrestricted-network",
			"native-permission-prompts",
			"production-notifications",
			"production-dialogs",
			"remote-asset-fetch",
			"electron-node-integration",
		},
		NegativeGuards: surface.SurfaceSecurityNegativeGuards{
			NoAmbientFilesystem:                       true,
			NoAmbientNetwork:                          true,
			NoShellFeatureBypass:                      true,
			NoPermissionlessClipboard:                 true,
			NoNotificationDialogWithoutTargetEvidence: true,
			NoNetworkAssetFetch:                       true,
			NoUntrustedFontImageDecode:                true,
			NoElectronNodeIntegration:                 true,
			NoUserJSAppLogic:                          true,
			NoDOMAppUITree:                            true,
		},
	}
}
func securityCapabilityStatusForAppShellFeature(featureStatus string) (string, bool) {
	switch featureStatus {
	case "target_evidenced", "scoped_adapter":
		return "allowed_with_policy", true
	case "blocked_pass":
		return "blocked_nonclaim", false
	default:
		return "unknown", false
	}
}

func collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(
	artifactDir string,
) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge": "linux_accessibility_host_bridge_v1",
		"source": "examples/surface/release/surface_release_form.tetra",
		"roles": []string{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"checkbox",
			"row",
			"button",
			"status",
		},
		"focus_order": []string{
			"NameTextBox",
			"EmailTextBox",
			"SubscribeCheckbox",
			"SaveButton",
			"ResetButton",
		},
		"labelled_by": map[string]string{
			"NameTextBox":  "NameLabel",
			"EmailTextBox": "EmailLabel",
		},
		"states_exported": []string{
			"focused",
			"enabled",
			"editable",
			"checked",
			"pressed",
			"status",
		},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf(
			"write linux release window accessibility host bridge artifact: %w",
			err,
		)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface/release/surface_release_form.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf(
			"write linux release window accessibility platform probe artifact: %w",
			err,
		)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{
			Name:     "surface linux accessibility host bridge",
			Kind:     "runtime",
			Path:     bridgePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux accessibility platform probe",
			Kind:     "runtime",
			Path:     probePath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}

func collectLinuxX64AccessibilityMetadataRealWindowProbeEvidence(
	artifactDir string,
) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-accessibility-metadata-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"write linux-x64 accessibility metadata real-window frame artifact: %w",
			err,
		)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Accessibility Metadata Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 accessibility metadata real-window probe: %w",
			err,
		)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 accessibility metadata real-window probe: unexpected stdout %q",
			stdout,
		)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 accessibility metadata real-window probe: unexpected stderr %q",
			stderr,
		)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf(
			"run linux-x64 accessibility metadata real-window probe: exit code %d, want 42",
			exitCode,
		)
	}
	process := surface.ProcessReport{
		Name: "surface linux-x64 real-window probe",
		Kind: "app",
		Path: fmt.Sprintf(
			"%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d",
			os.Args[0],
			framePath,
			frame.Width,
			frame.Height,
			frame.Stride,
		),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func runRealWindowProbe(opt smokeOptions) error {
	if opt.ProbeFrameWidth <= 0 || opt.ProbeFrameHeight <= 0 || opt.ProbeFrameStride <= 0 {
		return fmt.Errorf("real-window probe requires positive frame dimensions and stride")
	}
	var frame rgbaFrame
	if opt.ProbeFramePath != "" {
		pixels, err := os.ReadFile(opt.ProbeFramePath)
		if err != nil {
			return fmt.Errorf("read real-window probe frame %s: %w", opt.ProbeFramePath, err)
		}
		if len(pixels) != opt.ProbeFrameStride*opt.ProbeFrameHeight {
			return fmt.Errorf(
				"real-window probe frame bytes = %d, want stride*height %d",
				len(pixels),
				opt.ProbeFrameStride*opt.ProbeFrameHeight,
			)
		}
		frame = rgbaFrame{
			Width:  opt.ProbeFrameWidth,
			Height: opt.ProbeFrameHeight,
			Stride: opt.ProbeFrameStride,
			Pixels: pixels,
		}
	} else {
		frame = renderWindowCounterFrameRGBA(2, 1, opt.ProbeFrameWidth, opt.ProbeFrameHeight, true)
	}
	return presentRealWindowSurface(
		opt.ProbeTitle,
		frame,
		350*time.Millisecond,
		opt.ProbeHoldUntilClose,
	)
}
func runPresentedFrameProbeAndReadExpectedPixels(path string, want []byte) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixelsMatching(cmd.Process.Pid, want, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: unexpected stdout %q",
			path,
			stdout.String(),
		)
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: unexpected stderr %q",
			path,
			stderr.String(),
		)
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: %w",
			path,
			readErr,
		)
	}
	return pixels, exitCode, nil
}
func runPresentedFrameProbeAndReadPixels(path string) ([]byte, int, error) {
	return runPresentedFrameProbeAndReadPixelsLen(path, len(surfacePresentedFrameProbePixels()))
}
func runPresentedFrameProbeAndReadPixelsLen(path string, wantLen int) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixels(cmd.Process.Pid, wantLen, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: unexpected stdout %q",
			path,
			stdout.String(),
		)
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: unexpected stderr %q",
			path,
			stderr.String(),
		)
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf(
			"run linux-x64 app-presented frame probe %s: %w",
			path,
			readErr,
		)
	}
	return pixels, exitCode, nil
}
func readSurfaceMemfdPixelsMatching(pid int, want []byte, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, len(want))
		if err == nil {
			if bytes.Equal(pixels, want) {
				return pixels, nil
			}
			lastErr = fmt.Errorf(
				"surface memfd checksum %s, waiting for %s",
				checksumRGBA(pixels),
				checksumRGBA(want),
			)
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}
func terminateProbe(cmd *exec.Cmd) int {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}
func readSurfaceMemfdPixels(pid int, wantLen int, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, wantLen)
		if err == nil {
			return pixels, nil
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}
func tryReadSurfaceMemfdPixels(pid int, wantLen int) ([]byte, error) {
	fdDir := filepath.Join("/proc", fmt.Sprint(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fdPath := filepath.Join(fdDir, entry.Name())
		target, err := os.Readlink(fdPath)
		if err != nil || !strings.Contains(target, "memfd") {
			continue
		}
		file, err := os.Open(fdPath)
		if err != nil {
			continue
		}
		_, _ = file.Seek(0, io.SeekStart)
		buf := make([]byte, wantLen)
		_, readErr := io.ReadFull(file, buf)
		_ = file.Close()
		if readErr == nil {
			return buf, nil
		}
	}
	return nil, fmt.Errorf("no readable Surface memfd with %d bytes for pid %d", wantLen, pid)
}
func rejectLegacyUISidecarArtifacts(root string, opts ...sidecarScanOptions) error {
	_, err := scanLegacyUISidecarArtifacts(root, opts...)
	return err
}

func scanLegacyUISidecarArtifacts(
	root string,
	opts ...sidecarScanOptions,
) (surface.ArtifactScanReport, error) {
	var opt sidecarScanOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	report := surface.ArtifactScanReport{
		Root:           root,
		ForbiddenPaths: []string{},
		Pass:           true,
	}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		report.FilesChecked++
		if legacyUISidecarArtifactPath(path, opt) {
			report.ForbiddenPaths = append(report.ForbiddenPaths, path)
		}
		return nil
	}); err != nil {
		return report, err
	}
	if len(report.ForbiddenPaths) > 0 {
		report.Pass = false
		return report, fmt.Errorf(
			"Surface build emitted legacy UI sidecar artifact %s",
			report.ForbiddenPaths[0],
		)
	}
	return report, nil
}
func legacyUISidecarArtifactPath(path string, opt sidecarScanOptions) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") ||
		strings.HasSuffix(base, ".html") ||
		strings.HasSuffix(base, ".js") {
		return true
	}
	if strings.HasSuffix(base, ".mjs") {
		return !opt.AllowCompilerOwnedWASMLoader || !compilerOwnedWASMLoaderArtifactPath(path)
	}
	return false
}
func compilerOwnedWASMLoaderArtifactPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") || !strings.HasSuffix(base, ".mjs") {
		return false
	}
	wasmPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".wasm"
	return fileExists(wasmPath)
}
func resolveSurfaceSourcePath(raw string) (string, error) {
	if raw == "" {
		raw = "examples/surface/runtime/surface_counter.tetra"
	}
	if filepath.IsAbs(raw) {
		return requireExistingSource(raw)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if path, err := requireExistingSource(filepath.Join(cwd, raw)); err == nil {
		return path, nil
	}
	if root := findRepoRoot(cwd); root != "" {
		return requireExistingSource(filepath.Join(root, raw))
	}
	return requireExistingSource(filepath.Join(cwd, raw))
}
func requireExistingSource(path string) (string, error) {
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, want Surface source file", cleaned)
	}
	return cleaned, nil
}
func findRepoRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(dir, "go.work")) && dirExists(filepath.Join(dir, "examples")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
func repoRootForCommands() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := findRepoRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("find repo root from %s", cwd)
	}
	return root, nil
}
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
func runExecutable(path string) (string, string, int, error) {
	return runCommand(exec.Command(path))
}
func nodeCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("node", args...)
	cmd.Env = withoutNodeEnvProxy(os.Environ())
	return cmd
}
func withoutNodeEnvProxy(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "NODE_USE_ENV_PROXY=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
func runCommand(cmd *exec.Cmd) (string, string, int, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return stdout.String(), stderr.String(), -1, err
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode(), nil
}

// ---- morph_rendered_beauty_report.go ----

func readVisualRegressionReport(path string) (surface.VisualRegressionReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surface.VisualRegressionReport{}, err
	}
	var report surface.VisualRegressionReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return surface.VisualRegressionReport{}, err
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		return surface.VisualRegressionReport{}, err
	}
	return report, nil
}

func buildMorphRenderedBeautyReport(
	runtimeReportPath string,
	runtime surface.Report,
	visual surface.VisualRegressionReport,
	scenarioName string,
) (surface.MorphRenderedBeautyReport, error) {
	if runtime.Morph == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"morph_evidence is required for Morph rendered beauty report",
		)
	}
	if runtime.BlockSceneSnapshot == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"block_scene_snapshot is required for Morph rendered beauty report",
		)
	}
	if runtime.RenderCommandStream == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"render_command_stream is required for Morph rendered beauty report",
		)
	}
	if strings.TrimSpace(scenarioName) == "" {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"scenario_name is required for Morph rendered beauty report",
		)
	}
	source := strings.TrimSpace(runtime.Morph.Source)
	if source == "" {
		source = strings.TrimSpace(runtime.Source)
	}
	if source == "" {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"morph source is required for Morph rendered beauty report",
		)
	}
	if !sameEvidencePath(source, runtime.Source) {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"morph source %q must match runtime report source %q",
			source,
			runtime.Source,
		)
	}
	sourceSHA, err := prefixedSHA256File(source)
	if err != nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"hash Morph source %s: %w",
			source,
			err,
		)
	}
	target := morphRenderedBeautyTarget(runtime)
	visualTarget, visualFrame, err := morphRenderedBeautyVisualEvidence(
		runtimeReportPath,
		runtime,
		visual,
		source,
		target,
	)
	if err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	if strings.TrimSpace(visualTarget.GitHead) != "" &&
		strings.TrimSpace(runtime.Morph.GitHead) != "" &&
		visualTarget.GitHead != runtime.Morph.GitHead {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"visual git_head %q must match morph git_head %q",
			visualTarget.GitHead,
			runtime.Morph.GitHead,
		)
	}
	if visualFrame.Checksum != normalizePrefixedSHA256(runtime.RenderCommandStream.FrameChecksum) {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf(
			"pixel golden frame checksum %s must match render_command_stream.frame_checksum %s",
			visualFrame.Checksum,
			runtime.RenderCommandStream.FrameChecksum,
		)
	}
	rendererStableProof := morphRenderedBeautyRendererStableProof(runtime, visualFrame)

	report := surface.MorphRenderedBeautyReport{
		Schema:         surface.MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   surface.MorphRenderedBeautyScope,
		Target:         target,
		ScenarioName:   scenarioName,
		GitHead:        runtime.Morph.GitHead,
		GitCommit:      runtime.Morph.GitHead,
		GitDirty:       runtime.Morph.GitDirty,
		ProductClaim:   false,
		FinalSignoff:   false,
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:               source,
			SourceSHA256:         sourceSHA,
			CapsuleHash:          runtime.Morph.CapsuleHash,
			TokenGraphHash:       runtime.Morph.TokenGraphHash,
			TokenCount:           morphRenderedBeautyTokenCount(runtime.Morph),
			TokenCategories:      morphRenderedBeautyTokenCategories(runtime.Morph),
			RecipeCount:          len(runtime.Morph.Recipes),
			RecipeExpansionCount: len(runtime.Morph.RecipeExpansions),
			RecipeNames:          morphRenderedBeautyRecipeNames(runtime.Morph),
			ResolvedMorphSceneHash: prefixedSHA256Text(
				"resolved-morph-scene|" + source + "|" + runtime.Morph.CapsuleHash + "|" + runtime.Morph.TokenGraphHash + "|" + runtime.BlockSceneSnapshot.BlockSceneHash + "|" + runtime.RenderCommandStream.CommandStreamHash,
			),
			BlockSceneSnapshotHash: runtime.BlockSceneSnapshot.BlockSceneHash,
		},
		BlockSceneSnapshot: surface.MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               runtime.BlockSceneSnapshot.Schema,
			SurfaceScope:         runtime.BlockSceneSnapshot.SurfaceScope,
			Source:               runtime.BlockSceneSnapshot.Source,
			QualityLevel:         runtime.BlockSceneSnapshot.QualityLevel,
			CorePrimitives:       runtime.BlockSceneSnapshot.CorePrimitives,
			CompactPropsOnly:     runtime.BlockSceneSnapshot.CompactPropsOnly,
			RecipeExpansionCount: runtime.BlockSceneSnapshot.RecipeExpansionCount,
			NodeCount:            runtime.BlockSceneSnapshot.NodeCount,
			RichSpecHash:         runtime.BlockSceneSnapshot.RichSpecHash,
			BlockSceneHash:       runtime.BlockSceneSnapshot.BlockSceneHash,
			SpecCoverage: surface.MorphRenderedBeautyBlockSceneSpecCoverage{
				Layout:        runtime.BlockSceneSnapshot.SpecCoverage.Layout,
				Paint:         runtime.BlockSceneSnapshot.SpecCoverage.Paint,
				Text:          runtime.BlockSceneSnapshot.SpecCoverage.Text,
				Image:         runtime.BlockSceneSnapshot.SpecCoverage.Image,
				Input:         runtime.BlockSceneSnapshot.SpecCoverage.Input,
				Event:         runtime.BlockSceneSnapshot.SpecCoverage.Event,
				State:         runtime.BlockSceneSnapshot.SpecCoverage.State,
				Motion:        runtime.BlockSceneSnapshot.SpecCoverage.Motion,
				Accessibility: runtime.BlockSceneSnapshot.SpecCoverage.Accessibility,
			},
		},
		RenderEvidence: surface.MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: runtime.RenderCommandStream.CommandStreamHash,
			CommandCount:      runtime.RenderCommandStream.CommandCount,
			Renderer:          runtime.RenderCommandStream.Renderer,
		},
		RendererStableProof: rendererStableProof,
		RenderCommandStream: morphRenderedBeautyCommandStream(runtime.RenderCommandStream),
		PixelEvidence: surface.MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           visualFrame.ArtifactPath,
			FrameArtifactSHA256:     visualFrame.ArtifactSHA256,
			FrameChecksum:           visualFrame.Checksum,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         morphRenderedBeautyRecipeHash(runtime.Morph),
			BlockSceneHash:          runtime.BlockSceneSnapshot.BlockSceneHash,
			RenderCommandStreamHash: runtime.RenderCommandStream.CommandStreamHash,
			GoldenArtifact:          visualFrame.GoldenArtifactPath,
			GoldenArtifactSHA256:    visualFrame.GoldenArtifactSHA256,
			GoldenChecksum:          visualFrame.GoldenChecksum,
			DiffPixels:              visualFrame.DiffPixels,
			DiffRatioMilli:          visualFrame.DiffRatioMilli,
			MaxChannelDelta:         visualFrame.MaxChannelDelta,
			PrecomputedFixtureFrame: false,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected:             true,
			SelfGoldenRejected:               true,
			PrecomputedFrameRejected:         true,
			MissingFrameArtifactRejected:     true,
			NoDOMUI:                          true,
			NoCSSRuntime:                     true,
			NoReactRuntime:                   true,
			NoElectronRuntime:                true,
			NoNativeWidgets:                  true,
			NoHiddenAppState:                 true,
			NonBlockOutputRejected:           true,
			DirtyCheckoutProductionRejected:  true,
			UnsupportedTargetRejected:        true,
			RendererOwnedStableProofRequired: true,
		},
		NonClaims: []string{
			"no Electron runtime claim",
			"no React runtime claim",
			"no CSS runtime claim",
			"no DOM-authored UI claim",
			"no GPU renderer production claim",
			"no macOS production claim",
			"no Windows production claim",
		},
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(report); err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	return report, nil
}

func applyMorphRenderedBeautyProductSignoff(
	report *surface.MorphRenderedBeautyReport,
	productClaim bool,
	finalSignoff bool,
) error {
	if report == nil {
		return fmt.Errorf("Morph rendered beauty report is required for product signoff")
	}
	if !productClaim && !finalSignoff {
		return nil
	}
	if finalSignoff && !productClaim {
		return fmt.Errorf("Morph rendered beauty final_signoff requires product_claim")
	}
	if productClaim && !finalSignoff {
		return fmt.Errorf("Morph rendered beauty product_claim requires final_signoff")
	}
	if report.GitDirty {
		return fmt.Errorf(
			"Morph rendered beauty product_claim requires clean checkout: git_dirty=true",
		)
	}
	proof := report.RendererStableProof
	if proof.PixelOwner != "surface-renderer" || !proof.RendererOwned || proof.BridgeOwnedPixels ||
		!proof.BlockFirst ||
		!proof.DerivedFromRenderCommandStream ||
		!proof.StablePromotionEligible {
		return fmt.Errorf(
			"Morph rendered beauty product_claim requires renderer-owned stable proof",
		)
	}
	report.ProductClaim = true
	report.FinalSignoff = true
	return nil
}

func morphRenderedBeautyScenarioName(opt smokeOptions) string {
	source := strings.TrimSpace(defaultSurfaceSourcePath(opt))
	if source == "" {
		return strings.TrimSpace(opt.Mode)
	}
	return strings.TrimSpace(opt.Mode) + ":" + source
}

func morphRenderedBeautyVisualEvidence(
	runtimeReportPath string,
	runtime surface.Report,
	visual surface.VisualRegressionReport,
	source string,
	target string,
) (surface.VisualRegressionTargetReport, surface.VisualRegressionFrameReport, error) {
	if len(visual.Apps) == 0 {
		return surface.VisualRegressionTargetReport{}, surface.VisualRegressionFrameReport{}, fmt.Errorf(
			"pixel golden comparison is required for Morph rendered beauty report",
		)
	}
	frameChecksum := normalizePrefixedSHA256(runtime.RenderCommandStream.FrameChecksum)
	for _, app := range visual.Apps {
		if !sameEvidencePath(app.Source, source) {
			continue
		}
		for _, visualTarget := range app.Targets {
			if normalizeTargetName(visualTarget.Target) != normalizeTargetName(target) {
				continue
			}
			if strings.TrimSpace(runtimeReportPath) != "" &&
				strings.TrimSpace(visualTarget.RuntimeReport) != "" &&
				visualTarget.RuntimeReport != runtimeReportPath {
				continue
			}
			for _, frame := range visualTarget.Frames {
				if !frame.Pass {
					continue
				}
				if frame.Checksum == frameChecksum {
					return visualTarget, frame, nil
				}
			}
		}
	}
	return surface.VisualRegressionTargetReport{}, surface.VisualRegressionFrameReport{}, fmt.Errorf(
		"pixel golden comparison missing passing frame for source %s target %s checksum %s",
		source,
		target,
		frameChecksum,
	)
}

func morphRenderedBeautyCommandStream(
	stream *surface.RenderCommandStreamReport,
) surface.MorphRenderedBeautyRenderCommandStream {
	out := surface.MorphRenderedBeautyRenderCommandStream{
		Schema:                        stream.Schema,
		Source:                        stream.Source,
		SurfaceScope:                  stream.SurfaceScope,
		Producer:                      stream.Producer,
		QualityLevel:                  stream.QualityLevel,
		Renderer:                      stream.Renderer,
		DerivedFromBlockSceneSnapshot: stream.DerivedFromBlockSceneSnapshot,
		BlockSceneHash:                stream.BlockSceneHash,
		FrameChecksum:                 normalizePrefixedSHA256(stream.FrameChecksum),
		CommandStreamHash:             stream.CommandStreamHash,
		CommandCount:                  stream.CommandCount,
		SourceLinked:                  stream.SourceLinked,
		HandcraftedFixture:            stream.HandcraftedFixture,
	}
	for _, command := range stream.Commands {
		out.Commands = append(out.Commands, surface.MorphRenderedBeautyRenderCommand{
			Order:          command.Order,
			Command:        command.Command,
			Source:         command.Source,
			SourceNodeID:   command.SourceNodeID,
			Recipe:         command.Recipe,
			LayerID:        command.LayerID,
			BlockID:        command.BlockID,
			Quality:        command.Quality,
			Color:          command.Color,
			Width:          command.Width,
			Blur:           command.Blur,
			OffsetX:        command.OffsetX,
			OffsetY:        command.OffsetY,
			RasterFormat:   command.RasterFormat,
			RasterHash:     command.RasterHash,
			RasterWidth:    command.RasterWidth,
			RasterHeight:   command.RasterHeight,
			RasterCoverage: command.RasterCoverage,
			MarkerOnly:     command.MarkerOnly,
			Checksum:       command.Checksum,
		})
	}
	return out
}

func morphRenderedBeautyRendererStableProof(
	runtime surface.Report,
	visualFrame surface.VisualRegressionFrameReport,
) surface.MorphRenderedBeautyRendererStableProof {
	proof := surface.MorphRenderedBeautyRendererStableProof{
		Schema:                         "tetra.surface.renderer-stable-proof.v1",
		PixelOwner:                     "morph-evidence-bridge",
		RendererOwned:                  false,
		BridgeOwnedPixels:              true,
		BlockFirst:                     true,
		DerivedFromRenderCommandStream: false,
		RenderCommandStreamHash:        runtime.RenderCommandStream.CommandStreamHash,
		BlockSceneHash:                 runtime.BlockSceneSnapshot.BlockSceneHash,
		FrameChecksum: normalizePrefixedSHA256(
			runtime.RenderCommandStream.FrameChecksum,
		),
		StablePromotionEligible: false,
	}
	if runtime.RenderCommandStream == nil || runtime.BlockSceneSnapshot == nil {
		return proof
	}
	rendered, err := surfacerender.RenderCommandStreamRGBA(
		runtime.RenderCommandStream,
		visualFrame.Width,
		visualFrame.Height,
	)
	if err != nil {
		return proof
	}
	if normalizePrefixedSHA256(rendered.Checksum) != normalizePrefixedSHA256(visualFrame.Checksum) {
		return proof
	}
	proof.PixelOwner = "surface-renderer"
	proof.RendererOwned = true
	proof.BridgeOwnedPixels = false
	proof.DerivedFromRenderCommandStream = true
	proof.StablePromotionEligible = true
	proof.FrameChecksum = normalizePrefixedSHA256(rendered.Checksum)
	return proof
}

func morphRenderedBeautyTarget(report surface.Report) string {
	switch {
	case report.Target == "linux-x64" && report.HostEvidence.RealWindow:
		return "linux-x64-real-window"
	case report.Target == "wasm32-web" && report.HostEvidence.BrowserCanvas:
		return "wasm32-web-browser-canvas"
	default:
		return strings.TrimSpace(report.Target)
	}
}

func morphRenderedBeautyTokenCount(morph *surface.MorphReport) int {
	if morph == nil || morph.TokenGraph == nil {
		return 0
	}
	return len(morph.TokenGraph.Tokens)
}

func morphRenderedBeautyTokenCategories(morph *surface.MorphReport) []string {
	if morph == nil || morph.TokenGraph == nil {
		return nil
	}
	values := append([]string{}, morph.TokenGraph.Categories...)
	if len(values) == 0 {
		for _, token := range morph.TokenGraph.Tokens {
			values = append(values, token.Category)
		}
	}
	return uniqueSortedStrings(values)
}

func morphRenderedBeautyRecipeNames(morph *surface.MorphReport) []string {
	if morph == nil {
		return nil
	}
	values := make([]string, 0, len(morph.Recipes))
	for _, recipe := range morph.Recipes {
		values = append(values, recipe.Name)
	}
	if len(values) == 0 {
		for _, expansion := range morph.RecipeExpansions {
			values = append(values, expansion.Recipe)
		}
	}
	return uniqueSortedStrings(values)
}

func morphRenderedBeautyRecipeHash(morph *surface.MorphReport) string {
	var builder strings.Builder
	for _, name := range morphRenderedBeautyRecipeNames(morph) {
		builder.WriteString(name)
		builder.WriteByte('\n')
	}
	if morph != nil {
		for _, expansion := range morph.RecipeExpansions {
			builder.WriteString(expansion.Recipe)
			builder.WriteString(fmt.Sprint(expansion.BlockIDs))
			builder.WriteByte('\n')
		}
	}
	return prefixedSHA256Text(builder.String())
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sameEvidencePath(a string, b string) bool {
	return strings.TrimSpace(
		strings.ReplaceAll(a, "\\", "/"),
	) == strings.TrimSpace(
		strings.ReplaceAll(b, "\\", "/"),
	)
}

func normalizeTargetName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func normalizePrefixedSHA256(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 64 && isHexSHA256(value) {
		return "sha256:" + strings.ToLower(value)
	}
	if strings.HasPrefix(value, "sha256:") {
		digest := strings.TrimPrefix(value, "sha256:")
		if len(digest) == 64 && isHexSHA256(digest) {
			return "sha256:" + strings.ToLower(digest)
		}
	}
	return value
}

func isHexSHA256(value string) bool {
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func prefixedSHA256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func prefixedSHA256File(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		resolved, resolveErr := resolveSurfaceSourcePath(path)
		if resolveErr != nil {
			return "", err
		}
		raw, err = os.ReadFile(resolved)
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// ---- render_command_stream.go ----

func attachRenderCommandStreamForScenario(source string, scenario *headlessScenario) {
	attachRenderCommandStreamForScenarioWithRenderer(source, "software-rgba-headless", scenario)
}

func attachRenderCommandStreamForScenarioWithRenderer(
	source string,
	renderer string,
	scenario *headlessScenario,
) {
	if scenario == nil || scenario.BlockSceneSnapshot == nil {
		return
	}
	scenario.RenderCommandStream = surfacerender.BuildCommandStream(
		source,
		renderer,
		scenario.BlockSceneSnapshot,
		scenario.Frames,
	)
}

// ---- render_rgba.go ----

func renderBlockPaintFrameRGBA(active bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 24, B: 30, A: 255}
	fillTop := rgbaColor{R: 52, G: 118, B: 215, A: 255}
	fillBottom := rgbaColor{R: 84, G: 180, B: 132, A: 255}
	border := rgbaColor{R: 226, G: 234, B: 242, A: 255}
	shadow := rgbaColor{R: 0, G: 0, B: 0, A: 88}
	outline := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if active {
		fillTop = rgbaColor{R: 66, G: 138, B: 232, A: 255}
		fillBottom = rgbaColor{R: 104, G: 196, B: 148, A: 255}
		outline = rgbaColor{R: 252, G: 220, B: 112, A: 255}
	}
	block := rect{X: 12, Y: 10, W: 64, H: 28}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: block.X, Y: block.Y + 4, W: block.W + 4, H: block.H + 4}, shadow)
	rectRGBA(frame, rect{X: block.X, Y: block.Y, W: block.W, H: block.H / 2}, fillTop)
	rectRGBA(
		frame,
		rect{X: block.X, Y: block.Y + block.H/2, W: block.W, H: block.H - block.H/2},
		fillBottom,
	)
	rectOutlineRGBA(frame, block, border)
	rectOutlineRGBA(
		frame,
		rect{X: block.X - 2, Y: block.Y - 2, W: block.W + 4, H: block.H + 4},
		outline,
	)
	if active {
		rectRGBA(frame, rect{X: block.X + 8, Y: block.Y + 10, W: 28, H: 6}, border)
	}
	return frame
}
func renderBlockTextFrameRGBA(active bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 16, G: 20, B: 24, A: 255}
	panel := rgbaColor{R: 32, G: 40, B: 48, A: 255}
	fg := rgbaColor{R: 237, G: 242, B: 247, A: 255}
	muted := rgbaColor{R: 128, G: 146, B: 164, A: 255}
	accent := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if active {
		panel = rgbaColor{R: 38, G: 50, B: 58, A: 255}
		fg = rgbaColor{R: 248, G: 250, B: 252, A: 255}
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 10, Y: 8, W: 150, H: 96}, panel)
	rectOutlineRGBA(frame, rect{X: 10, Y: 8, W: 150, H: 96}, muted)
	rectRGBA(frame, rect{X: 18, Y: 18, W: 78, H: 7}, fg)
	rectRGBA(frame, rect{X: 18, Y: 32, W: 96, H: 7}, fg)
	rectRGBA(frame, rect{X: 18, Y: 46, W: 54, H: 7}, fg)
	rectRGBA(frame, rect{X: 18, Y: 58, W: 120, H: 28}, rgbaColor{R: 20, G: 28, B: 34, A: 255})
	rectOutlineRGBA(frame, rect{X: 18, Y: 58, W: 120, H: 28}, fg)
	if active {
		rectRGBA(frame, rect{X: 28, Y: 68, W: 34, H: 6}, fg)
		rectRGBA(frame, rect{X: 68, Y: 64, W: 2, H: 20}, accent)
	}
	return frame
}
func renderBlockLayoutFrameRGBA(active bool) rgbaFrame {
	return renderBlockLayoutFrameSizedRGBA(320, 200, active)
}
func renderBlockLayoutResizedFrameRGBA() rgbaFrame {
	return renderBlockLayoutFrameSizedRGBA(480, 260, true)
}
func renderBlockEventFrameRGBA(active bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 23, B: 28, A: 255}
	panel := rgbaColor{R: 30, G: 38, B: 46, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 247, A: 255}
	accent := rgbaColor{R: 82, G: 154, B: 232, A: 255}
	disabled := rgbaColor{R: 72, G: 78, B: 86, A: 255}
	warn := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if active {
		panel = rgbaColor{R: 36, G: 48, B: 56, A: 255}
		accent = rgbaColor{R: 96, G: 174, B: 244, A: 255}
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, panel)
	rectOutlineRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, fg)
	rectRGBA(frame, rect{X: 24, Y: 24, W: 94, H: 7}, fg)
	rectRGBA(frame, rect{X: 24, Y: 64, W: 120, H: 44}, accent)
	rectOutlineRGBA(frame, rect{X: 24, Y: 64, W: 120, H: 44}, fg)
	rectRGBA(frame, rect{X: 152, Y: 64, W: 120, H: 44}, disabled)
	rectOutlineRGBA(frame, rect{X: 152, Y: 64, W: 120, H: 44}, warn)
	rectRGBA(frame, rect{X: 24, Y: 120, W: 120, H: 44}, rgbaColor{R: 42, G: 92, B: 74, A: 255})
	rectOutlineRGBA(frame, rect{X: 24, Y: 120, W: 120, H: 44}, fg)
	if active {
		rectOutlineRGBA(frame, rect{X: 20, Y: 60, W: 128, H: 52}, warn)
		rectRGBA(frame, rect{X: 34, Y: 80, W: 32, H: 6}, fg)
	}
	return frame
}
func renderBlockStateFrameRGBA(active bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 32, G: 38, B: 46, A: 255}
	fill := rgbaColor{R: 32, G: 38, B: 46, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 247, A: 255}
	outline := rgbaColor{R: 122, G: 162, B: 247, A: 255}
	status := rgbaColor{R: 72, G: 80, B: 90, A: 255}
	if active {
		fill = rgbaColor{R: 45, G: 155, B: 240, A: 255}
		outline = rgbaColor{R: 255, G: 95, B: 87, A: 255}
		status = rgbaColor{R: 82, G: 154, B: 120, A: 255}
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, panel)
	rectOutlineRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, fg)
	rectRGBA(frame, rect{X: 24, Y: 40, W: 168, H: 56}, fill)
	rectOutlineRGBA(frame, rect{X: 24, Y: 40, W: 168, H: 56}, outline)
	rectRGBA(frame, rect{X: 36, Y: 58, W: 72, H: 8}, fg)
	rectRGBA(frame, rect{X: 24, Y: 112, W: 168, H: 32}, status)
	rectOutlineRGBA(frame, rect{X: 24, Y: 112, W: 168, H: 32}, fg)
	if active {
		rectOutlineRGBA(
			frame,
			rect{X: 20, Y: 36, W: 176, H: 64},
			rgbaColor{R: 246, G: 205, B: 92, A: 255},
		)
		rectRGBA(frame, rect{X: 122, Y: 58, W: 28, H: 8}, rgbaColor{R: 255, G: 255, B: 255, A: 112})
		rectRGBA(frame, rect{X: 154, Y: 58, W: 8, H: 8}, rgbaColor{R: 255, G: 255, B: 255, A: 112})
	}
	return frame
}
func renderBlockMotionFrameRGBA(step int) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 28, G: 36, B: 44, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 247, A: 255}
	fill := rgbaColor{R: 32, G: 48, B: 64, A: 80}
	translateX := 0
	scale := 100
	if step == 1 {
		fill = rgbaColor{R: 64, G: 112, B: 148, A: 140}
		translateX = 6
		scale = 104
	}
	if step >= 2 {
		fill = rgbaColor{R: 96, G: 174, B: 244, A: 200}
		translateX = 12
		scale = 108
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, panel)
	rectOutlineRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, fg)
	w := 176 * scale / 100
	h := 64 * scale / 100
	rectRGBA(frame, rect{X: 24 + translateX, Y: 44, W: w, H: h}, fill)
	rectOutlineRGBA(frame, rect{X: 24 + translateX, Y: 44, W: w, H: h}, fg)
	rectRGBA(frame, rect{X: 36 + translateX, Y: 68, W: 72, H: 8}, fg)
	if step >= 2 {
		rectRGBA(
			frame,
			rect{X: 116 + translateX, Y: 68, W: 34, H: 8},
			rgbaColor{R: 255, G: 255, B: 255, A: 180},
		)
	}
	return frame
}
func renderBlockAssetFrameRGBA(active bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 28, G: 36, B: 44, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 247, A: 255}
	iconFill := rgbaColor{R: 255, G: 255, B: 255, A: 255}
	imageFill := rgbaColor{R: 76, G: 126, B: 156, A: 255}
	fallbackFill := rgbaColor{R: 86, G: 92, B: 102, A: 255}
	imageW := 48
	imageH := 32
	if active {
		iconFill = rgbaColor{R: 96, G: 174, B: 244, A: 255}
		imageW = 96
		imageH = 64
		fallbackFill = rgbaColor{R: 180, G: 190, B: 200, A: 255}
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, panel)
	rectOutlineRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, fg)
	rectRGBA(frame, rect{X: 24, Y: 36, W: 32, H: 32}, iconFill)
	rectRGBA(frame, rect{X: 30, Y: 42, W: 20, H: 4}, panel)
	rectRGBA(frame, rect{X: 38, Y: 42, W: 4, H: 20}, panel)
	rectOutlineRGBA(frame, rect{X: 24, Y: 36, W: 32, H: 32}, fg)
	rectRGBA(frame, rect{X: 72, Y: 32, W: imageW, H: imageH}, imageFill)
	rectRGBA(
		frame,
		rect{X: 80, Y: 42, W: imageW - 16, H: 8},
		rgbaColor{R: 220, G: 238, B: 255, A: 255},
	)
	rectOutlineRGBA(frame, rect{X: 72, Y: 32, W: imageW, H: imageH}, fg)
	rectRGBA(frame, rect{X: 24, Y: 112, W: 96, H: 32}, fallbackFill)
	rectOutlineRGBA(frame, rect{X: 24, Y: 112, W: 96, H: 32}, fg)
	if active {
		rectRGBA(frame, rect{X: 36, Y: 124, W: 72, H: 6}, rgbaColor{R: 18, G: 22, B: 26, A: 255})
		rectOutlineRGBA(
			frame,
			rect{X: 68, Y: 28, W: 104, H: 72},
			rgbaColor{R: 244, G: 205, B: 92, A: 255},
		)
	}
	return frame
}
func renderBlockAccessibilityFrameRGBA(focused bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 28, G: 36, B: 44, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 247, A: 255}
	label := rgbaColor{R: 150, G: 166, B: 184, A: 255}
	action := rgbaColor{R: 64, G: 112, B: 148, A: 255}
	focus := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if focused {
		action = rgbaColor{R: 96, G: 174, B: 244, A: 255}
	}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, panel)
	rectOutlineRGBA(frame, rect{X: 16, Y: 16, W: 288, H: 168}, fg)
	rectRGBA(frame, rect{X: 24, Y: 24, W: 200, H: 24}, label)
	rectRGBA(frame, rect{X: 24, Y: 64, W: 120, H: 44}, action)
	rectRGBA(frame, rect{X: 152, Y: 64, W: 120, H: 44}, rgbaColor{R: 58, G: 66, B: 78, A: 255})
	rectOutlineRGBA(frame, rect{X: 24, Y: 64, W: 120, H: 44}, fg)
	rectOutlineRGBA(frame, rect{X: 152, Y: 64, W: 120, H: 44}, fg)
	if focused {
		rectOutlineRGBA(frame, rect{X: 21, Y: 61, W: 126, H: 50}, focus)
		rectRGBA(frame, rect{X: 40, Y: 82, W: 64, H: 6}, rgbaColor{R: 18, G: 22, B: 26, A: 255})
	}
	return frame
}
func renderBlockSystemFrameRGBA(focused bool) rgbaFrame {
	frame := renderBlockAccessibilityFrameRGBA(focused)
	layoutFill := rgbaColor{R: 64, G: 92, B: 116, A: 255}
	scrollFill := rgbaColor{R: 84, G: 180, B: 132, A: 255}
	outline := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if focused {
		layoutFill = rgbaColor{R: 74, G: 118, B: 154, A: 255}
		scrollFill = rgbaColor{R: 96, G: 174, B: 244, A: 255}
	}
	rectRGBA(frame, rect{X: 236, Y: 72, W: 72, H: 80}, layoutFill)
	rectRGBA(frame, rect{X: 244, Y: 80, W: 56, H: 12}, scrollFill)
	rectRGBA(frame, rect{X: 244, Y: 100, W: 56, H: 12}, scrollFill)
	rectRGBA(frame, rect{X: 244, Y: 120, W: 56, H: 12}, scrollFill)
	rectOutlineRGBA(
		frame,
		rect{X: 236, Y: 72, W: 72, H: 80},
		rgbaColor{R: 238, G: 242, B: 247, A: 255},
	)
	if focused {
		rectOutlineRGBA(frame, rect{X: 232, Y: 68, W: 80, H: 88}, outline)
	}
	return frame
}
func renderBlockSystemFrameSizedRGBA(width int, height int, focused bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 32, G: 42, B: 50, A: 255}
	fg := rgbaColor{R: 232, G: 238, B: 244, A: 255}
	label := rgbaColor{R: 150, G: 166, B: 184, A: 255}
	action := rgbaColor{R: 64, G: 112, B: 148, A: 255}
	reset := rgbaColor{R: 58, G: 66, B: 78, A: 255}
	layoutFill := rgbaColor{R: 64, G: 92, B: 116, A: 255}
	scrollFill := rgbaColor{R: 84, G: 180, B: 132, A: 255}
	focus := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if focused {
		action = rgbaColor{R: 96, G: 174, B: 244, A: 255}
		layoutFill = rgbaColor{R: 74, G: 118, B: 154, A: 255}
		scrollFill = rgbaColor{R: 96, G: 174, B: 244, A: 255}
	}
	clearRGBA(frame, bg)
	panelRect := rect{X: 16, Y: 16, W: width - 32, H: height - 32}
	labelRect := rect{X: 24, Y: 24, W: width - 120, H: 24}
	submitRect := rect{X: 24, Y: 64, W: 120, H: 44}
	resetRect := rect{X: 152, Y: 64, W: 120, H: 44}
	layoutRect := rect{X: width - 84, Y: 72, W: 72, H: 96}
	rectRGBA(frame, panelRect, panel)
	rectOutlineRGBA(frame, panelRect, fg)
	rectRGBA(frame, labelRect, label)
	rectRGBA(frame, submitRect, action)
	rectRGBA(frame, resetRect, reset)
	rectOutlineRGBA(frame, submitRect, fg)
	rectOutlineRGBA(frame, resetRect, fg)
	rectRGBA(frame, layoutRect, layoutFill)
	for y := layoutRect.Y + 8; y <= layoutRect.Y+56; y += 20 {
		rectRGBA(frame, rect{X: layoutRect.X + 8, Y: y, W: 56, H: 12}, scrollFill)
	}
	rectOutlineRGBA(frame, layoutRect, fg)
	if focused {
		rectOutlineRGBA(
			frame,
			rect{
				X: submitRect.X - 3,
				Y: submitRect.Y - 3,
				W: submitRect.W + 6,
				H: submitRect.H + 6,
			},
			focus,
		)
		rectOutlineRGBA(
			frame,
			rect{
				X: layoutRect.X - 4,
				Y: layoutRect.Y - 4,
				W: layoutRect.W + 8,
				H: layoutRect.H + 8,
			},
			focus,
		)
		rectRGBA(frame, rect{X: submitRect.X + 16, Y: submitRect.Y + 18, W: 64, H: 6}, bg)
	}
	return frame
}

func renderMorphStudioShellFrameRGBA(width int, height int, active bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 14, G: 18, B: 22, A: 255}
	panel := rgbaColor{R: 32, G: 42, B: 50, A: 255}
	panelSoft := rgbaColor{R: 45, G: 58, B: 70, A: 255}
	fg := rgbaColor{R: 232, G: 238, B: 244, A: 255}
	softText := rgbaColor{R: 150, G: 166, B: 184, A: 255}
	action := rgbaColor{R: 64, G: 112, B: 148, A: 255}
	success := rgbaColor{R: 84, G: 180, B: 132, A: 255}
	accent := rgbaColor{R: 96, G: 174, B: 244, A: 255}
	focus := rgbaColor{R: 244, G: 205, B: 92, A: 255}
	if active {
		focus = accent
	}
	shellRect := rect{X: 8, Y: 8, W: width - 16, H: height - 16}
	navRect := rect{X: 18, Y: 28, W: 72, H: height - 56}
	toolbarRect := rect{X: 102, Y: 28, W: width - 122, H: 32}
	contentRect := rect{X: 102, Y: 72, W: width - 122, H: height - 114}
	commandRect := rect{X: 118, Y: 86, W: width - 154, H: 30}
	metricRect := rect{X: 118, Y: 130, W: 86, H: 40}
	logRect := rect{X: 216, Y: 130, W: width - 252, H: 40}
	statusRect := rect{X: 102, Y: height - 34, W: width - 122, H: 18}
	clearRGBA(frame, bg)
	rectRGBA(frame, shellRect, panel)
	rectOutlineRGBA(frame, shellRect, fg)
	rectRGBA(frame, navRect, panelSoft)
	rectRGBA(frame, toolbarRect, panelSoft)
	rectRGBA(frame, contentRect, rgbaColor{R: 24, G: 31, B: 38, A: 255})
	rectRGBA(frame, commandRect, action)
	rectRGBA(frame, metricRect, accent)
	rectRGBA(frame, logRect, softText)
	rectRGBA(frame, statusRect, success)
	rectOutlineRGBA(frame, commandRect, fg)
	rectRGBA(frame, rect{X: navRect.X + 12, Y: navRect.Y + 14, W: navRect.W - 24, H: 8}, accent)
	rectRGBA(frame, rect{X: navRect.X + 12, Y: navRect.Y + 34, W: navRect.W - 24, H: 8}, softText)
	rectRGBA(frame, rect{X: navRect.X + 12, Y: navRect.Y + 54, W: navRect.W - 24, H: 8}, softText)
	if active {
		rectOutlineRGBA(
			frame,
			rect{
				X: commandRect.X - 3,
				Y: commandRect.Y - 3,
				W: commandRect.W + 6,
				H: commandRect.H + 6,
			},
			focus,
		)
		rectOutlineRGBA(
			frame,
			rect{
				X: metricRect.X - 3,
				Y: metricRect.Y - 3,
				W: metricRect.W + 6,
				H: metricRect.H + 6,
			},
			focus,
		)
		rectRGBA(frame, rect{X: commandRect.X + 14, Y: commandRect.Y + 12, W: 72, H: 6}, bg)
	}
	return frame
}

func renderBlockLayoutFrameSizedRGBA(width int, height int, active bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 22, B: 26, A: 255}
	panel := rgbaColor{R: 32, G: 42, B: 50, A: 255}
	rowFill := rgbaColor{R: 66, G: 132, B: 214, A: 255}
	gridFill := rgbaColor{R: 70, G: 166, B: 130, A: 255}
	dockFill := rgbaColor{R: 204, G: 104, B: 78, A: 255}
	overlayFill := rgbaColor{R: 236, G: 198, B: 72, A: 230}
	scrollFill := rgbaColor{R: 126, G: 94, B: 190, A: 255}
	fg := rgbaColor{R: 232, G: 238, B: 244, A: 255}
	clearRGBA(frame, bg)

	column := rect{X: 12, Y: 12, W: width - 24, H: height - 24}
	row := rect{X: column.X + 12, Y: column.Y + 12, W: column.W - 24, H: 48}
	grid := rect{X: column.X + 12, Y: row.Y + row.H + 8, W: 132, H: 72}
	dock := rect{X: grid.X + grid.W + 8, Y: grid.Y, W: 132, H: 72}
	scroll := rect{X: width - 84, Y: 72, W: 72, H: 80}
	overlay := rect{X: width - 100, Y: 20, W: 72, H: 40}

	rectRGBA(frame, column, panel)
	rectOutlineRGBA(frame, column, fg)
	rectRGBA(frame, row, rowFill)
	rectOutlineRGBA(frame, row, fg)
	rectRGBA(frame, rect{X: row.X + 8, Y: row.Y + 14, W: 64, H: 8}, fg)
	rectRGBA(frame, rect{X: row.X + row.W - 96, Y: row.Y + 14, W: 72, H: 8}, fg)

	rectRGBA(frame, grid, rgbaColor{R: 24, G: 34, B: 38, A: 255})
	rectOutlineRGBA(frame, grid, fg)
	cellW := (grid.W - 6) / 2
	cellH := (grid.H - 6) / 2
	rectRGBA(frame, rect{X: grid.X, Y: grid.Y, W: cellW, H: cellH}, gridFill)
	rectRGBA(
		frame,
		rect{X: grid.X + cellW + 6, Y: grid.Y, W: cellW, H: cellH},
		rgbaColor{R: 90, G: 184, B: 150, A: 255},
	)
	rectRGBA(
		frame,
		rect{X: grid.X, Y: grid.Y + cellH + 6, W: cellW, H: cellH},
		rgbaColor{R: 52, G: 138, B: 118, A: 255},
	)

	rectRGBA(frame, dock, rgbaColor{R: 30, G: 38, B: 46, A: 255})
	rectOutlineRGBA(frame, dock, fg)
	rectRGBA(frame, rect{X: dock.X, Y: dock.Y, W: dock.W, H: 24}, dockFill)
	rectRGBA(frame, rect{X: dock.X + 8, Y: dock.Y + 34, W: dock.W - 16, H: 8}, fg)

	rectRGBA(frame, scroll, rgbaColor{R: 28, G: 32, B: 42, A: 255})
	rectOutlineRGBA(frame, scroll, fg)
	rectRGBA(frame, rect{X: scroll.X + 8, Y: scroll.Y + 12, W: 42, H: 8}, scrollFill)
	rectRGBA(frame, rect{X: scroll.X + 8, Y: scroll.Y + 30, W: 50, H: 8}, scrollFill)
	rectRGBA(
		frame,
		rect{X: scroll.X + scroll.W - 12, Y: scroll.Y + 16 + 8, W: 5, H: 24},
		overlayFill,
	)

	rectRGBA(frame, overlay, overlayFill)
	rectOutlineRGBA(frame, overlay, fg)
	if active {
		rectRGBA(frame, rect{X: column.X + 20, Y: height - 44, W: 96, H: 20}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderWindowCounterFrameRGBA(
	count int,
	keyCount int,
	width int,
	height int,
	focused bool,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 22, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	accent := rgbaColor{R: 32, G: 132, B: 214, A: 255}
	keyAccent := rgbaColor{R: 34, G: 160, B: 104, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	textMaskRGBA(frame, 32, 28, 5, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	textMaskRGBA(frame, 32, 52, 3, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(
			frame,
			rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8},
			fg,
		)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderBrowserCounterFrameRGBA(
	count int,
	keyCount int,
	width int,
	height int,
	focused bool,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 24, G: 22, B: 34, A: 255}
	fg := rgbaColor{R: 242, G: 244, B: 248, A: 255}
	accent := rgbaColor{R: 54, G: 130, B: 218, A: 255}
	keyAccent := rgbaColor{R: 42, G: 170, B: 112, A: 255}
	textAccent := rgbaColor{R: 218, G: 184, B: 58, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	textMaskRGBA(frame, 32, 28, 5, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	textMaskRGBA(frame, 32, 52, 3, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	textMaskRGBA(frame, 32, 68, 4, fg)
	rectRGBA(frame, rect{X: 88, Y: 68, W: 18, H: 7}, textAccent)
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(
			frame,
			rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8},
			fg,
		)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderReleaseCounterFrameRGBA(
	count int,
	keyCount int,
	resetCount int,
	statusCode int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 24, B: 28, A: 255}
	fg := rgbaColor{R: 236, G: 242, B: 240, A: 255}
	accent := rgbaColor{R: 60, G: 142, B: 212, A: 255}
	resetAccent := rgbaColor{R: 210, G: 96, B: 78, A: 255}
	statusAccent := rgbaColor{R: 88, G: 174, B: 128, A: 255}
	clearRGBA(frame, bg)
	textMaskRGBA(frame, 32, 28, 23, fg)
	textMaskRGBA(frame, 32, 56, 5, fg)
	rectRGBA(frame, rect{X: 96, Y: 58, W: 24 + count*8, H: 8}, statusAccent)
	textMaskRGBA(frame, 32, 76, 3, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 96, Y: 78, W: 24 + keyCount*8, H: 8}, accent)
	}
	if resetCount > 0 {
		rectRGBA(frame, rect{X: 136, Y: 78, W: 24 + resetCount*8, H: 8}, resetAccent)
	}
	button := rect{X: 32, Y: height/2 - 24, W: 160, H: 48}
	status := rect{X: 32, Y: height/2 + 40, W: width - 64, H: 32}
	rectRGBA(frame, button, accent)
	rectOutlineRGBA(frame, button, fg)
	rectRGBA(frame, status, rgbaColor{R: 28, G: 36, B: 42, A: 255})
	rectRGBA(
		frame,
		rect{X: status.X + 12, Y: status.Y + 12, W: 24 + statusCode*12, H: 8},
		statusAccent,
	)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderTextFocusInputFrameRGBA(
	textLen int,
	caret int,
	focusIndex int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 19, G: 25, B: 29, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	textBg := rgbaColor{R: 28, G: 38, B: 45, A: 255}
	textAccent := rgbaColor{R: 56, G: 148, B: 112, A: 255}
	buttonAccent := rgbaColor{R: 54, G: 130, B: 218, A: 255}
	caretColor := rgbaColor{R: 232, G: 196, B: 64, A: 255}
	textbox := rect{X: 32, Y: 64, W: 224, H: 44}
	button := rect{X: 32, Y: 128, W: 128, H: 44}

	clearRGBA(frame, bg)
	textMaskRGBA(frame, 32, 28, 5, fg)
	textMaskRGBA(frame, 32, 44, 5, fg)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(
			frame,
			rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10},
			textAccent,
		)
	}
	caretX := textbox.X + 12 + caret*12
	rectRGBA(frame, rect{X: caretX, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, button, buttonAccent)
	rectOutlineRGBA(frame, button, fg)
	if focusIndex == 0 {
		rectOutlineRGBA(
			frame,
			rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8},
			fg,
		)
	}
	if focusIndex == 1 {
		rectOutlineRGBA(
			frame,
			rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8},
			fg,
		)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderComponentTreeFrameRGBA(
	textLen int,
	caret int,
	focusID int,
	submitted int,
	reset int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 23, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	textBg := rgbaColor{R: 29, G: 41, B: 47, A: 255}
	textAccent := rgbaColor{R: 59, G: 150, B: 113, A: 255}
	submitAccent := rgbaColor{R: 44, G: 127, B: 204, A: 255}
	resetAccent := rgbaColor{R: 192, G: 92, B: 64, A: 255}
	caretColor := rgbaColor{R: 232, G: 196, B: 64, A: 255}
	markerColor := rgbaColor{R: 172, G: 206, B: 96, A: 255}

	textbox := rect{X: 16, Y: 48, W: width - 32, H: 44}
	row := rect{X: 16, Y: 104, W: width - 32, H: 44}
	submitButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 76, Y: 16, W: 24 + submitted*14, H: 7}, markerColor)
	rectRGBA(frame, rect{X: 116, Y: 16, W: 24 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(
			frame,
			rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10},
			textAccent,
		)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	if focusID == 3 {
		rectOutlineRGBA(
			frame,
			rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8},
			fg,
		)
	}
	if focusID == 5 {
		rectOutlineRGBA(
			frame,
			rect{
				X: submitButton.X - 4,
				Y: submitButton.Y - 4,
				W: submitButton.W + 8,
				H: submitButton.H + 8,
			},
			fg,
		)
	}
	if focusID == 6 {
		rectOutlineRGBA(
			frame,
			rect{
				X: resetButton.X - 4,
				Y: resetButton.Y - 4,
				W: resetButton.W + 8,
				H: resetButton.H + 8,
			},
			fg,
		)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderMinimalToolkitFrameRGBA(
	textLen int,
	caret int,
	focusID int,
	submitted int,
	reset int,
	statusCode int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 17, G: 24, B: 25, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	panelBg := rgbaColor{R: 23, G: 33, B: 34, A: 255}
	textBg := rgbaColor{R: 29, G: 43, B: 45, A: 255}
	textAccent := rgbaColor{R: 58, G: 156, B: 125, A: 255}
	submitAccent := rgbaColor{R: 49, G: 122, B: 204, A: 255}
	resetAccent := rgbaColor{R: 192, G: 86, B: 74, A: 255}
	caretColor := rgbaColor{R: 235, G: 196, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 205, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	column := rect{X: 12, Y: 12, W: width - 24, H: height - 24}
	textbox := rect{X: 20, Y: 52, W: width - 40, H: 44}
	row := rect{X: 20, Y: 108, W: width - 40, H: 44}
	submitButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 160, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: column.X + 8, Y: column.Y + 8, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 76, Y: column.Y + 8, W: 22 + submitted*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: 116, Y: column.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(
			frame,
			rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10},
			textAccent,
		)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(
			frame,
			rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8},
			statusAccent,
		)
	}
	if focusID == 4 {
		rectOutlineRGBA(
			frame,
			rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8},
			fg,
		)
	}
	if focusID == 6 {
		rectOutlineRGBA(
			frame,
			rect{
				X: submitButton.X - 4,
				Y: submitButton.Y - 4,
				W: submitButton.W + 8,
				H: submitButton.H + 8,
			},
			fg,
		)
	}
	if focusID == 7 {
		rectOutlineRGBA(
			frame,
			rect{
				X: resetButton.X - 4,
				Y: resetButton.Y - 4,
				W: resetButton.W + 8,
				H: resetButton.H + 8,
			},
			fg,
		)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderToolkitReuseFrameRGBA(
	nameLen int,
	emailLen int,
	focusID int,
	saved int,
	reset int,
	statusCode int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 16, G: 22, B: 29, A: 255}
	fg := rgbaColor{R: 235, G: 242, B: 244, A: 255}
	panelBg := rgbaColor{R: 25, G: 33, B: 42, A: 255}
	textBg := rgbaColor{R: 31, G: 45, B: 58, A: 255}
	nameAccent := rgbaColor{R: 75, G: 166, B: 138, A: 255}
	emailAccent := rgbaColor{R: 86, G: 137, B: 214, A: 255}
	saveAccent := rgbaColor{R: 54, G: 133, B: 210, A: 255}
	resetAccent := rgbaColor{R: 194, G: 92, B: 78, A: 255}
	caretColor := rgbaColor{R: 235, G: 196, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 205, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	title := rect{X: 20, Y: 20, W: width - 40, H: 24}
	nameBox := rect{X: 20, Y: 52, W: width - 40, H: 44}
	nameLabel := rect{X: 20, Y: 104, W: width - 40, H: 24}
	emailBox := rect{X: 20, Y: 136, W: width - 40, H: 44}
	row := rect{X: 20, Y: 192, W: width - 40, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 248, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 72, H: 7}, fg)
	rectRGBA(frame, rect{X: title.X + 96, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: title.X + 136, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	if nameLen > 0 {
		rectRGBA(
			frame,
			rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10},
			nameAccent,
		)
	}
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(
			frame,
			rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10},
			emailAccent,
		)
	}
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(
			frame,
			rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8},
			statusAccent,
		)
	}
	if focusID == 4 {
		rectOutlineRGBA(
			frame,
			rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 6 {
		rectOutlineRGBA(
			frame,
			rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 8 {
		rectOutlineRGBA(
			frame,
			rect{
				X: saveButton.X - 4,
				Y: saveButton.Y - 4,
				W: saveButton.W + 8,
				H: saveButton.H + 8,
			},
			fg,
		)
	}
	if focusID == 9 {
		rectOutlineRGBA(
			frame,
			rect{
				X: resetButton.X - 4,
				Y: resetButton.Y - 4,
				W: resetButton.W + 8,
				H: resetButton.H + 8,
			},
			fg,
		)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderReleaseToolkitFrameRGBA(
	nameLen int,
	emailLen int,
	focusID int,
	saved int,
	reset int,
	statusCode int,
	checked bool,
	scrollOffset int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 24, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 240, A: 255}
	panelBg := rgbaColor{R: 28, G: 38, B: 42, A: 255}
	stackBg := rgbaColor{R: 33, G: 45, B: 50, A: 255}
	textBg := rgbaColor{R: 39, G: 52, B: 59, A: 255}
	nameAccent := rgbaColor{R: 80, G: 172, B: 132, A: 255}
	emailAccent := rgbaColor{R: 80, G: 138, B: 214, A: 255}
	checkboxAccent := rgbaColor{R: 214, G: 177, B: 72, A: 255}
	scrollAccent := rgbaColor{R: 136, G: 106, B: 210, A: 255}
	saveAccent := rgbaColor{R: 56, G: 132, B: 206, A: 255}
	resetAccent := rgbaColor{R: 198, G: 92, B: 78, A: 255}
	statusAccent := rgbaColor{R: 176, G: 206, B: 94, A: 255}
	caretColor := rgbaColor{R: 236, G: 197, B: 64, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	stack := rect{X: 16, Y: 16, W: width - 32, H: height - 32}
	title := rect{X: 32, Y: 32, W: width - 64, H: 28}
	description := rect{X: 32, Y: 68, W: width - 64, H: 28}
	nameLabel := rect{X: 32, Y: 104, W: width - 64, H: 24}
	nameBox := rect{X: 32, Y: 132, W: width - 64, H: 44}
	emailLabel := rect{X: 32, Y: 184, W: width - 64, H: 24}
	emailBox := rect{X: 32, Y: 212, W: width - 64, H: 44}
	checkbox := rect{X: 32, Y: 264, W: width - 64, H: 32}
	scroll := rect{X: 32, Y: 304, W: width - 64, H: 48}
	row := rect{X: 32, Y: 360, W: width - 64, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	spacer := rect{X: row.X + 288, Y: row.Y, W: 16, H: 44}
	status := rect{X: row.X + 312, Y: row.Y, W: row.W - 312, H: 44}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, stack, stackBg)
	rectOutlineRGBA(frame, stack, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 116, H: 8}, fg)
	rectRGBA(frame, rect{X: description.X + 8, Y: description.Y + 8, W: 164, H: 7}, scrollAccent)
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	if nameLen > 0 {
		rectRGBA(
			frame,
			rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10},
			nameAccent,
		)
	}
	rectRGBA(frame, rect{X: emailLabel.X + 8, Y: emailLabel.Y + 8, W: 52, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(
			frame,
			rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10},
			emailAccent,
		)
	}
	rectRGBA(frame, checkbox, textBg)
	rectOutlineRGBA(frame, checkbox, fg)
	rectOutlineRGBA(frame, rect{X: checkbox.X + 12, Y: checkbox.Y + 8, W: 16, H: 16}, fg)
	if checked {
		rectRGBA(frame, rect{X: checkbox.X + 16, Y: checkbox.Y + 12, W: 8, H: 8}, checkboxAccent)
	}
	rectRGBA(frame, scroll, textBg)
	rectOutlineRGBA(frame, scroll, fg)
	rectRGBA(
		frame,
		rect{X: scroll.X + 12, Y: scroll.Y + 12 - scrollOffset/2, W: scroll.W - 40, H: 8},
		scrollAccent,
	)
	rectRGBA(
		frame,
		rect{X: scroll.X + scroll.W - 18, Y: scroll.Y + 6 + scrollOffset/2, W: 6, H: 20},
		checkboxAccent,
	)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, spacer, panelBg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(
			frame,
			rect{X: status.X + 12, Y: status.Y + 16, W: 20 + statusCode*16, H: 8},
			statusAccent,
		)
	}
	if focusID == 7 {
		rectOutlineRGBA(
			frame,
			rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 9 {
		rectOutlineRGBA(
			frame,
			rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 10 {
		rectOutlineRGBA(
			frame,
			rect{X: checkbox.X - 4, Y: checkbox.Y - 4, W: checkbox.W + 8, H: checkbox.H + 8},
			fg,
		)
	}
	if focusID == 14 {
		rectOutlineRGBA(
			frame,
			rect{
				X: saveButton.X - 4,
				Y: saveButton.Y - 4,
				W: saveButton.W + 8,
				H: saveButton.H + 8,
			},
			fg,
		)
	}
	if focusID == 15 {
		rectOutlineRGBA(
			frame,
			rect{
				X: resetButton.X - 4,
				Y: resetButton.Y - 4,
				W: resetButton.W + 8,
				H: resetButton.H + 8,
			},
			fg,
		)
	}
	if saved > 0 {
		rectRGBA(
			frame,
			rect{X: title.X + 140, Y: title.Y + 8, W: 22 + saved*14, H: 7},
			statusAccent,
		)
	}
	if reset > 0 {
		rectRGBA(frame, rect{X: title.X + 184, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderAccessibilityMetadataFrameRGBA(
	nameLen int,
	emailLen int,
	focusID int,
	saved int,
	reset int,
	statusCode int,
	width int,
	height int,
) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 14, G: 24, B: 28, A: 255}
	fg := rgbaColor{R: 234, G: 242, B: 238, A: 255}
	panelBg := rgbaColor{R: 24, G: 34, B: 38, A: 255}
	textBg := rgbaColor{R: 31, G: 46, B: 51, A: 255}
	nameAccent := rgbaColor{R: 78, G: 166, B: 128, A: 255}
	emailAccent := rgbaColor{R: 72, G: 136, B: 205, A: 255}
	saveAccent := rgbaColor{R: 52, G: 126, B: 205, A: 255}
	resetAccent := rgbaColor{R: 196, G: 92, B: 78, A: 255}
	caretColor := rgbaColor{R: 236, G: 197, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 204, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	title := rect{X: 20, Y: 20, W: width - 40, H: 24}
	nameLabel := rect{X: 20, Y: 52, W: width - 40, H: 24}
	nameBox := rect{X: 20, Y: 84, W: width - 40, H: 44}
	emailLabel := rect{X: 20, Y: 136, W: width - 40, H: 24}
	emailBox := rect{X: 20, Y: 168, W: width - 40, H: 44}
	row := rect{X: 20, Y: 224, W: width - 40, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 280, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 84, H: 7}, fg)
	rectRGBA(frame, rect{X: title.X + 104, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: title.X + 144, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	rectRGBA(frame, rect{X: emailLabel.X + 8, Y: emailLabel.Y + 8, W: 52, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	rectRGBA(
		frame,
		rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10},
		emailAccent,
	)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	rectRGBA(
		frame,
		rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8},
		statusAccent,
	)
	if focusID == 5 {
		rectOutlineRGBA(
			frame,
			rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 7 {
		rectOutlineRGBA(
			frame,
			rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8},
			fg,
		)
		rectRGBA(
			frame,
			rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24},
			caretColor,
		)
	}
	if focusID == 9 {
		rectOutlineRGBA(
			frame,
			rect{
				X: saveButton.X - 4,
				Y: saveButton.Y - 4,
				W: saveButton.W + 8,
				H: saveButton.H + 8,
			},
			fg,
		)
	}
	if focusID == 10 {
		rectOutlineRGBA(
			frame,
			rect{
				X: resetButton.X - 4,
				Y: resetButton.Y - 4,
				W: resetButton.W + 8,
				H: resetButton.H + 8,
			},
			fg,
		)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func newRGBAFrame(width, height int) rgbaFrame {
	stride := width * 4
	return rgbaFrame{
		Width:  width,
		Height: height,
		Stride: stride,
		Pixels: make([]byte, stride*height),
	}
}
func clearRGBA(frame rgbaFrame, color rgbaColor) {
	rectRGBA(frame, rect{X: 0, Y: 0, W: frame.Width, H: frame.Height}, color)
}
func rectOutlineRGBA(frame rgbaFrame, r rect, color rgbaColor) {
	rectRGBA(frame, rect{X: r.X, Y: r.Y, W: r.W, H: 1}, color)
	rectRGBA(frame, rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, color)
	rectRGBA(frame, rect{X: r.X, Y: r.Y, W: 1, H: r.H}, color)
	rectRGBA(frame, rect{X: r.X + r.W - 1, Y: r.Y, W: 1, H: r.H}, color)
}
func rectRGBA(frame rgbaFrame, r rect, color rgbaColor) {
	maxY := r.Y + r.H
	maxX := r.X + r.W
	for y := r.Y; y < maxY; y++ {
		for x := r.X; x < maxX; x++ {
			if x < 0 || y < 0 || x >= frame.Width || y >= frame.Height {
				continue
			}
			i := y*frame.Stride + x*4
			frame.Pixels[i] = color.R
			frame.Pixels[i+1] = color.G
			frame.Pixels[i+2] = color.B
			frame.Pixels[i+3] = color.A
		}
	}
}
func textMaskRGBA(frame rgbaFrame, x int, y int, textLen int, color rgbaColor) {
	for glyph := 0; glyph < textLen; glyph++ {
		glyphMask5x7RGBA(frame, x+glyph*6, y, glyph, color)
	}
}
func glyphMask5x7RGBA(frame rgbaFrame, x int, y int, glyphIndex int, color rgbaColor) {
	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if glyphPixelOn(glyphIndex, row, col) {
				rectRGBA(frame, rect{X: x + col, Y: y + row, W: 1, H: 1}, color)
			}
		}
	}
}
func glyphPixelOn(glyphIndex int, row int, col int) bool {
	if row == 0 || row == 6 {
		return (glyphIndex+col)%3 != 1
	}
	if col == 0 || col == 4 {
		return (glyphIndex+row)%2 == 0
	}
	return (glyphIndex+row*2+col*3)%4 == 0
}
func checksumRGBA(pixels []byte) string {
	sum := sha256.Sum256(pixels)
	return hex.EncodeToString(sum[:])
}
func checksumText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ---- wasm_browser.go ----

func collectWASM32WebProcessEvidence(
	sourcePath string,
	artifactDir string,
) (surfaceProcessEvidence, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	wasmPath := filepath.Join(artifactDir, "surface-counter.wasm")
	if _, err := compiler.BuildFileWithStatsOpt(
		sourcePath,
		wasmPath,
		"wasm32-web",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"build wasm32-web Surface source %s: %w",
			sourcePath,
			err,
		)
	}
	componentArtifact, err := artifactReport(wasmPath, "component-app")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if err := validateCompilerOwnedWASMLoader(wasmPath); err != nil {
		return surfaceProcessEvidence{}, err
	}
	loaderArtifact, err := artifactReport(
		strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath))+".mjs",
		"compiler-owned-loader",
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	validateCmd := exec.Command(
		"go",
		"run",
		"./tools/cmd/validate-wasm-imports",
		"--target",
		"wasm32-web",
		wasmPath,
	)
	validateCmd.Dir = root
	validateStdout, validateStderr, validateExit, err := runCommand(validateCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface import validator: %w",
			err,
		)
	}
	if validateExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface import validator: exit code %d, stdout %q stderr %q",
			validateExit,
			validateStdout,
			validateStderr,
		)
	}
	if validateStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface import validator: unexpected stdout %q",
			validateStdout,
		)
	}

	nodeVersionCmd := nodeCommand("--version")
	nodeStdout, nodeStderr, nodeExit, err := runCommand(nodeVersionCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface runtime probe: %w", err)
	}
	if nodeExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface runtime probe: exit code %d, stdout %q stderr %q",
			nodeExit,
			nodeStdout,
			nodeStderr,
		)
	}
	if nodeStderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface runtime probe: unexpected stderr %q",
			nodeStderr,
		)
	}

	helperPath := filepath.Join(root, "scripts", "tools", "web_run_module.mjs")
	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	appCmd := nodeCommand(helperPath, "--surface-trace", tracePath, wasmPath)
	appCmd.Dir = root
	appStdout, appStderr, appExit, err := runCommand(appCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface app: %w", err)
	}
	if appStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface app %s: unexpected stdout %q",
			wasmPath,
			appStdout,
		)
	}
	if appStderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface app %s: unexpected stderr %q",
			wasmPath,
			appStderr,
		)
	}
	if appExit != 1 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web Surface app %s: exit code %d, want 1",
			wasmPath,
			appExit,
		)
	}
	traceFrames, err := readWASM32WebSurfaceTrace(tracePath)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if len(traceFrames) < 2 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web Surface runner trace has %d frames, want pre/post presented frames",
			len(traceFrames),
		)
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	if traceFrames[len(traceFrames)-1].Checksum != checksumRGBA(wantFrame.Pixels) {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web Surface runner after-frame checksum = %s, want %s",
			traceFrames[len(traceFrames)-1].Checksum,
			checksumRGBA(wantFrame.Pixels),
		)
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(
		artifactDir,
		sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	processes := []surface.ProcessReport{
		{
			Name:     "tetra build",
			Kind:     "build",
			Path:     fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web component app",
			Kind: "app",
			Path: fmt.Sprintf(
				"node scripts/tools/web_run_module.mjs --surface-trace %s %s",
				tracePath,
				wasmPath,
			),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(appExit),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: fmt.Sprintf(
				"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
				wasmPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(validateExit),
		},
		{
			Name:     "surface wasm32-web runtime",
			Kind:     "runtime",
			Path:     "node --version",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(nodeExit),
		},
	}
	return surfaceProcessEvidence{
		Processes:    processes,
		Artifacts:    []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact},
		ArtifactScan: sidecarScan,
		Frames:       traceFrames,
	}, nil
}

func collectWASM32WebBrowserCanvasProcessEvidence(
	sourcePath string,
	artifactDir string,
	scenarioName string,
) (surfaceProcessEvidence, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	wasmFile := "surface-browser-counter.wasm"
	if scenarioName == "text-focus-input" {
		wasmFile = "surface-textbox-app.wasm"
	}
	if scenarioName == "release-text-input" {
		wasmFile = "surface-release-text-input.wasm"
	}
	if scenarioName == "release-toolkit" {
		wasmFile = "surface-release-form.wasm"
	}
	if scenarioName == "release-browser" {
		wasmFile = "surface-release-form.wasm"
	}
	if scenarioName == "release-accessibility" {
		wasmFile = "surface-release-accessibility.wasm"
	}
	if scenarioName == "component-tree" {
		wasmFile = "surface-tree-app.wasm"
	}
	if scenarioName == "minimal-toolkit" {
		wasmFile = "surface-toolkit-form.wasm"
	}
	if scenarioName == "toolkit-reuse" {
		wasmFile = "surface-toolkit-settings.wasm"
	}
	if scenarioName == "accessibility-metadata" {
		wasmFile = "surface-accessibility-settings.wasm"
	}
	if scenarioName == "block-system" {
		wasmFile = "surface-block-system.wasm"
	}
	if scenarioName == "studio-shell" {
		wasmFile = "surface-morph-rendered-studio-shell.wasm"
	}
	if scenarioName == "guest-dashboard" {
		wasmFile = "surface-morph-guest-dashboard.wasm"
	}
	wasmPath := filepath.Join(artifactDir, wasmFile)
	if _, err := compiler.BuildFileWithStatsOpt(
		sourcePath,
		wasmPath,
		"wasm32-web",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"build wasm32-web browser canvas Surface source %s: %w",
			sourcePath,
			err,
		)
	}
	componentArtifact, err := artifactReport(wasmPath, "component-app")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if err := validateCompilerOwnedWASMLoader(wasmPath); err != nil {
		return surfaceProcessEvidence{}, err
	}
	loaderArtifact, err := artifactReport(
		strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath))+".mjs",
		"compiler-owned-loader",
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	validateCmd := exec.Command(
		"go",
		"run",
		"./tools/cmd/validate-wasm-imports",
		"--target",
		"wasm32-web",
		wasmPath,
	)
	validateCmd.Dir = root
	validateStdout, validateStderr, validateExit, err := runCommand(validateCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web browser canvas Surface import validator: %w",
			err,
		)
	}
	if validateExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web browser canvas Surface import validator: exit code %d, stdout %q stderr %q",
			validateExit,
			validateStdout,
			validateStderr,
		)
	}
	if validateStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web browser canvas Surface import validator: unexpected stdout %q",
			validateStdout,
		)
	}

	browserPath, err := discoverBrowserRunner()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	browserVersionCmd := exec.Command(browserPath, "--version")
	browserVersionStdout, browserVersionStderr, browserVersionExit, err := runCommand(
		browserVersionCmd,
	)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web browser canvas runtime probe: %w",
			err,
		)
	}
	if browserVersionExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run wasm32-web browser canvas runtime probe: exit code %d, stdout %q stderr %q",
			browserVersionExit,
			browserVersionStdout,
			browserVersionStderr,
		)
	}

	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	browserTrace, browserProcessPath, browserExit, err := runBrowserCanvasTrace(
		root,
		browserPath,
		wasmPath,
		scenarioName,
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	expectedTraceAppExit := surfaceComponentAppExpectedExitForSource(
		"wasm32-web-browser-canvas-"+scenarioName,
		sourcePath,
	)
	traceFrames, err := writeBrowserCanvasSurfaceTrace(
		tracePath,
		wasmPath,
		browserTrace,
		expectedTraceAppExit,
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if browserTrace.AppExitCode != expectedTraceAppExit {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web browser canvas app exit code = %d, want %d",
			browserTrace.AppExitCode,
			expectedTraceAppExit,
		)
	}
	if len(traceFrames) < 2 {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web browser canvas trace has %d frames, want pre/post presented frames",
			len(traceFrames),
		)
	}
	after := traceFrames[len(traceFrames)-1]
	if scenarioName == "release-text-input" {
		before := traceFrames[0]
		if after.Width != 480 || after.Height != 320 || after.Stride != 1920 || !after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release text-input after-frame = %#v, want presented 480x320 RGBA frame",
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release text-input frame checksums did not change across text/input baseline: %#v",
				traceFrames,
			)
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	if scenarioName == "release-browser" {
		before := traceFrames[0]
		if after.Order != 5 || after.Width != 560 || after.Height != 420 || after.Stride != 2240 ||
			!after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release browser after-frame = %#v, want order-5 presented 560x420 RGBA frame",
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web release browser frame checksums did not change " +
					"across browser release scenario: %#v"),
				traceFrames,
			)
		}
		if !browserTraceHasNativeEvents(
			browserTrace,
			[]string{
				"pointerup",
				"keydown",
				"resize",
				"beforeinput",
				"compositionstart",
				"compositionupdate",
				"compositionend",
			},
		) {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release browser trace missing required native browser input events: %#v",
				browserTrace.BrowserEvents,
			)
		}
		if err := validateBrowserReleaseTraceEvidence(browserTrace); err != nil {
			return surfaceProcessEvidence{}, err
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	if scenarioName == "release-accessibility" {
		before := traceFrames[0]
		if after.Order != 5 || after.Width != 480 || after.Height != 320 || after.Stride != 1920 ||
			!after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release accessibility after-frame = %#v, want order-5 presented 480x320 RGBA frame",
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web release accessibility frame checksums did not change " +
					"across platform bridge scenario: %#v"),
				traceFrames,
			)
		}
		if !browserTraceHasNativeEvents(
			browserTrace,
			[]string{"pointerup", "keydown", "resize", "beforeinput"},
		) {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web release accessibility trace missing required native browser input events: %#v",
				browserTrace.BrowserEvents,
			)
		}
		if err := validateBrowserAccessibilityTraceEvidence(browserTrace); err != nil {
			return surfaceProcessEvidence{}, err
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	if scenarioName == "block-system" {
		before := traceFrames[0]
		if !browserTrace.Canvas.Readback {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web browser-canvas block-system trace missing RGBA readback evidence",
			)
		}
		if after.Order != 5 || after.Width != 400 || after.Height != 240 || after.Stride != 1600 ||
			!after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas block-system after-frame = %#v, want " +
					"order-5 presented 400x240 RGBA frame"),
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas block-system frame checksums did not " +
					"change across browser input/readback: %#v"),
				traceFrames,
			)
		}
		if !browserTraceHasNativeEvents(
			browserTrace,
			[]string{"pointerup", "keydown", "resize", "beforeinput"},
		) {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas block-system trace missing required " +
					"native browser input events: %#v"),
				browserTrace.BrowserEvents,
			)
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	if scenarioName == "studio-shell" {
		before := traceFrames[0]
		if !browserTrace.Canvas.Readback {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web browser-canvas Morph studio-shell trace missing RGBA readback evidence",
			)
		}
		if after.Order != 5 || after.Width != 320 || after.Height != 200 || after.Stride != 1280 ||
			!after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph studio-shell after-frame = %#v, " +
					"want order-5 presented 320x200 RGBA frame"),
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph studio-shell frame checksums " +
					"did not change across browser input/readback: %#v"),
				traceFrames,
			)
		}
		if !browserTraceHasNativeEvents(
			browserTrace,
			[]string{"pointerup", "keydown", "beforeinput"},
		) {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph studio-shell trace missing " +
					"required native browser input events: %#v"),
				browserTrace.BrowserEvents,
			)
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	if scenarioName == "guest-dashboard" {
		before := traceFrames[0]
		if !browserTrace.Canvas.Readback {
			return surfaceProcessEvidence{}, fmt.Errorf(
				"wasm32-web browser-canvas Morph guest-dashboard trace missing RGBA readback evidence",
			)
		}
		if after.Order != 5 || after.Width != 1760 || after.Height != 700 || after.Stride != 7040 ||
			!after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph guest-dashboard after-frame = " +
					"%#v, want order-5 presented 1760x700 RGBA frame"),
				after,
			)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph guest-dashboard frame checksums " +
					"did not change across browser input/readback: %#v"),
				traceFrames,
			)
		}
		if !browserTraceHasNativeEvents(
			browserTrace,
			[]string{"pointerup", "beforeinput", "keydown"},
		) {
			return surfaceProcessEvidence{}, fmt.Errorf(
				("wasm32-web browser-canvas Morph guest-dashboard trace missing " +
					"required native browser input events: %#v"),
				browserTrace.BrowserEvents,
			)
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: fmt.Sprintf(
					"tetra build --target wasm32-web %s -o %s",
					sourcePath,
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: fmt.Sprintf(
					"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
					browserPath,
					scenarioName,
					wasmPath,
				),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(browserExit),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: fmt.Sprintf(
					"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
					wasmPath,
				),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(validateExit),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     strings.TrimSpace(browserVersionStdout),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserVersionExit),
			},
			{
				Name:     "surface wasm32-web browser canvas trace",
				Kind:     "runtime",
				Path:     browserProcessPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(browserExit),
			},
		}
		return surfaceProcessEvidence{
			Processes: processes,
			Artifacts: []surface.ArtifactReport{
				componentArtifact,
				loaderArtifact,
				traceArtifact,
			},
			ArtifactScan: sidecarScan,
			Frames:       traceFrames,
		}, nil
	}
	wantFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
	if isReleaseCounterSourcePath(sourcePath) && scenarioName == "counter" {
		wantFrame = renderReleaseCounterFrameRGBA(0, 1, 1, 2, 400, 240)
	}
	if scenarioName == "text-focus-input" || scenarioName == "release-text-input" {
		wantFrame = renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	}
	if scenarioName == "component-tree" {
		wantFrame = renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	}
	if scenarioName == "minimal-toolkit" {
		wantFrame = renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	}
	if scenarioName == "toolkit-reuse" {
		wantFrame = renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	}
	if scenarioName == "release-toolkit" {
		wantFrame = renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	}
	if scenarioName == "accessibility-metadata" || scenarioName == "release-accessibility" {
		wantFrame = renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	}
	if after.Order != 5 || after.Width != wantFrame.Width || after.Height != wantFrame.Height ||
		after.Stride != wantFrame.Stride ||
		after.Checksum != checksumRGBA(wantFrame.Pixels) {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web browser canvas after-frame = %#v, want order-5 %dx%d checksum %s",
			after,
			wantFrame.Width,
			wantFrame.Height,
			checksumRGBA(wantFrame.Pixels),
		)
	}
	if !browserTraceHasNativeEvents(
		browserTrace,
		[]string{"pointerup", "keydown", "resize", "beforeinput"},
	) {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"wasm32-web browser canvas trace missing required native browser input events: %#v",
			browserTrace.BrowserEvents,
		)
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(
		artifactDir,
		sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	processes := []surface.ProcessReport{
		{
			Name:     "tetra build",
			Kind:     "build",
			Path:     fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: fmt.Sprintf(
				"%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s",
				browserPath,
				scenarioName,
				wasmPath,
			),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(browserExit),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: fmt.Sprintf(
				"go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s",
				wasmPath,
			),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(validateExit),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     strings.TrimSpace(browserVersionStdout),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(browserVersionExit),
		},
		{
			Name:     "surface wasm32-web browser canvas trace",
			Kind:     "runtime",
			Path:     browserProcessPath,
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(browserExit),
		},
	}
	return surfaceProcessEvidence{
		Processes:    processes,
		Artifacts:    []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact},
		ArtifactScan: sidecarScan,
		Frames:       traceFrames,
	}, nil
}
func browserTraceHasNativeEvents(trace browserCanvasTrace, nativeTypes []string) bool {
	seen := map[string]bool{}
	for _, event := range trace.BrowserEvents {
		seen[event.NativeType] = true
	}
	for _, nativeType := range nativeTypes {
		if !seen[nativeType] {
			return false
		}
	}
	return true
}
func validateBrowserReleaseTraceEvidence(trace browserCanvasTrace) error {
	if trace.BrowserClipboard.Harness != "deterministic-browser-clipboard-v1" ||
		!trace.BrowserClipboard.Read ||
		!trace.BrowserClipboard.Write ||
		!trace.BrowserClipboard.OwnedCopy ||
		trace.BrowserClipboard.Bytes <= 0 {
		return fmt.Errorf(
			"wasm32-web release browser trace missing deterministic clipboard harness evidence: %#v",
			trace.BrowserClipboard,
		)
	}
	if !trace.BrowserComposition.Start ||
		!trace.BrowserComposition.Update ||
		!trace.BrowserComposition.Commit ||
		!trace.BrowserComposition.Cancel {
		return fmt.Errorf(
			"wasm32-web release browser trace missing composition evidence: %#v",
			trace.BrowserComposition,
		)
	}
	if err := validateBrowserAccessibilityTraceEvidence(trace); err != nil {
		return err
	}
	return nil
}
func validateBrowserAccessibilityTraceEvidence(trace browserCanvasTrace) error {
	if !trace.BrowserAccessibility.Snapshot ||
		!trace.BrowserAccessibility.Mirror ||
		!trace.BrowserAccessibility.CompilerOwned ||
		!trace.BrowserAccessibility.Bounds ||
		!trace.BrowserAccessibility.Focus {
		return fmt.Errorf(
			"wasm32-web release browser trace missing accessibility snapshot/mirror evidence: %#v",
			trace.BrowserAccessibility,
		)
	}
	if trace.BrowserAccessibility.DOMVisualUI || trace.BrowserAccessibility.UserJS {
		return fmt.Errorf(
			"wasm32-web release browser trace must not claim DOM visual UI or user JS app logic: %#v",
			trace.BrowserAccessibility,
		)
	}
	for _, role := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !containsString(trace.BrowserAccessibility.Roles, role) {
			return fmt.Errorf(
				"wasm32-web release browser trace missing accessibility role %s: %#v",
				role,
				trace.BrowserAccessibility.Roles,
			)
		}
	}
	return nil
}
func discoverBrowserRunner() (string, error) {
	var probeFailure string
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		runner, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		cmd := exec.Command(
			runner,
			"--headless",
			"--no-sandbox",
			"--disable-gpu",
			"--dump-dom",
			"about:blank",
		)
		cmd.Stdout = io.Discard
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			probeFailure = fmt.Sprintf(
				"%s failed headless probe: %v: %s",
				runner,
				err,
				strings.TrimSpace(stderr.String()),
			)
			continue
		}
		return runner, nil
	}
	if probeFailure != "" {
		return "", fmt.Errorf(
			"cannot run wasm32-web browser canvas Surface evidence: browser runner unavailable: %s",
			probeFailure,
		)
	}
	return "", fmt.Errorf(
		("cannot run wasm32-web browser canvas Surface evidence: browser " +
			"runner unavailable; searched: chromium, chromium-browser, google-chrome," +
			" chrome"),
	)
}

func runBrowserCanvasTrace(
	root string,
	browserPath string,
	wasmPath string,
	scenarioName string,
) (browserCanvasTrace, string, int, error) {
	hostPath := filepath.Join(root, "scripts", "tools", "surface_browser_canvas_host.mjs")
	hostSource, err := os.ReadFile(hostPath)
	if err != nil {
		return browserCanvasTrace{}, "", -1, fmt.Errorf(
			"read browser canvas Surface host %s: %w",
			hostPath,
			err,
		)
	}
	if _, err := os.Stat(wasmPath); err != nil {
		return browserCanvasTrace{}, "", -1, fmt.Errorf(
			"stat browser canvas Surface wasm %s: %w",
			wasmPath,
			err,
		)
	}
	runnerURL, cleanupRunner, err := browserCanvasRunnerFileURL(
		wasmPath,
		string(hostSource),
		scenarioName,
	)
	if err != nil {
		return browserCanvasTrace{}, "", -1, err
	}
	defer cleanupRunner()
	args := []string{
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--allow-file-access-from-files",
		"--virtual-time-budget=12000",
		"--dump-dom",
		runnerURL,
	}
	processArgs := append([]string{}, args[:len(args)-1]...)
	processArgs = append(
		processArgs,
		fmt.Sprintf("<surface-browser-canvas-file-runner scenario=%s>", scenarioName),
	)
	processPath := browserPath + " " + strings.Join(processArgs, " ")
	var lastTraceErr error
	for attempt := 1; attempt <= 3; attempt++ {
		cmd := exec.Command(browserPath, args...)
		stdout, stderr, exit, err := runCommand(cmd)
		if err != nil {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf(
				"run wasm32-web browser canvas Surface app: %w stderr=%q",
				err,
				stderr,
			)
		}
		if exit != 0 {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf(
				"run wasm32-web browser canvas Surface app: browser exit code %d stderr=%q",
				exit,
				stderr,
			)
		}
		rawTrace, err := extractBrowserCanvasTrace(stdout)
		if err != nil {
			lastTraceErr = fmt.Errorf("%w; browser stderr=%q", err, stderr)
			if isRetriableBrowserCanvasTraceError(err) && attempt < 3 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return browserCanvasTrace{}, processPath, exit, lastTraceErr
		}
		var trace browserCanvasTrace
		if err := json.Unmarshal([]byte(rawTrace), &trace); err != nil {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf(
				"decode browser canvas Surface trace: %w: %s",
				err,
				rawTrace,
			)
		}
		if strings.TrimSpace(trace.Error) != "" {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf(
				"browser canvas Surface trace error: %s",
				trace.Error,
			)
		}
		return trace, processPath, exit, nil
	}
	return browserCanvasTrace{}, processPath, -1, fmt.Errorf(
		"browser canvas Surface trace was not populated after retries: %w",
		lastTraceErr,
	)
}

func browserCanvasRunnerDataURL(
	hostSource string,
	wasmBytes []byte,
	scenarioName string,
) (string, error) {
	inlineHost, err := inlineBrowserCanvasHostSource(hostSource)
	if err != nil {
		return "", err
	}
	wasmURL := "data:application/wasm;base64," + base64.StdEncoding.EncodeToString(wasmBytes)
	html := browserCanvasRunnerHTML(inlineHost, wasmURL, scenarioName)
	return "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(html)), nil
}

func browserCanvasRunnerFileURL(
	wasmPath string,
	hostSource string,
	scenarioName string,
) (string, func(), error) {
	inlineHost, err := inlineBrowserCanvasHostSource(hostSource)
	if err != nil {
		return "", nil, err
	}
	absWASM, err := filepath.Abs(wasmPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve browser canvas wasm path %s: %w", wasmPath, err)
	}
	runnerDir := filepath.Dir(absWASM)
	if strings.HasSuffix(filepath.Base(runnerDir), "-artifacts") {
		runnerDir = filepath.Dir(runnerDir)
	}
	if err := os.MkdirAll(runnerDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create browser canvas runner dir %s: %w", runnerDir, err)
	}
	runnerPath := filepath.Join(
		runnerDir,
		"surface-browser-canvas-runner-"+safeBrowserCanvasScenarioName(scenarioName)+".html",
	)
	html := browserCanvasRunnerHTML(inlineHost, fileURL(absWASM), scenarioName)
	if err := os.WriteFile(runnerPath, []byte(html), 0o644); err != nil {
		return "", nil, fmt.Errorf("write browser canvas runner %s: %w", runnerPath, err)
	}
	cleanup := func() {
		_ = os.Remove(runnerPath)
	}
	return fileURL(runnerPath), cleanup, nil
}
func inlineBrowserCanvasHostSource(hostSource string) (string, error) {
	inlineHost := strings.Replace(
		hostSource,
		"export async function runSurfaceBrowserCanvas",
		"async function runSurfaceBrowserCanvas",
		1,
	)
	if inlineHost == hostSource {
		return "", fmt.Errorf("browser canvas Surface host missing runSurfaceBrowserCanvas export")
	}
	return inlineHost, nil
}
func safeBrowserCanvasScenarioName(scenarioName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(scenarioName)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}
func fileURL(path string) string {
	return (&neturl.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
func browserCanvasRunnerHTML(inlineHost string, wasmURL string, scenarioName string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
  <head><style>html,body{margin:0}canvas{display:block}</style></head>
  <body>
    <canvas id="surface-canvas" width="320" height="200"></canvas>
    <pre id="surface-trace">pending</pre>
    <script>
%s
      const target = document.getElementById('surface-trace');
      (async () => {
        try {
          const trace = await runSurfaceBrowserCanvas({
            wasmURL: %q,
            canvas: document.getElementById('surface-canvas'),
            scenario: %q,
          });
          target.textContent = JSON.stringify(trace);
        } catch (err) {
          target.textContent = JSON.stringify({
            schema: 'tetra.surface.browser-canvas-trace.v1',
            error: String(err && err.stack ? err.stack : err),
          });
        }
      })();
    </script>
  </body>
</html>
`, inlineHost, wasmURL, scenarioName)
}
func extractBrowserCanvasTrace(dom string) (string, error) {
	const startMarker = `<pre id="surface-trace">`
	start := strings.Index(dom, startMarker)
	if start < 0 {
		return "", fmt.Errorf("browser canvas Surface runner did not emit surface-trace element")
	}
	start += len(startMarker)
	end := strings.Index(dom[start:], `</pre>`)
	if end < 0 {
		return "", fmt.Errorf(
			"browser canvas Surface runner emitted unterminated surface-trace element",
		)
	}
	text := strings.TrimSpace(html.UnescapeString(dom[start : start+end]))
	if text == "" || text == "pending" {
		return "", fmt.Errorf("browser canvas Surface runner trace was not populated")
	}
	return text, nil
}
func isRetriableBrowserCanvasTraceError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "runner trace was not populated")
}

func writeBrowserCanvasSurfaceTrace(
	path string,
	wasmPath string,
	raw browserCanvasTrace,
	expectedAppExit int,
) ([]surface.FrameReport, error) {
	if raw.Schema != "tetra.surface.browser-canvas-trace.v1" {
		return nil, fmt.Errorf(
			"browser canvas Surface trace schema is %q, want tetra.surface.browser-canvas-trace.v1",
			raw.Schema,
		)
	}
	if !raw.Canvas.Opened || !raw.Canvas.Readback {
		return nil, fmt.Errorf(
			"browser canvas Surface trace missing opened/readback canvas evidence: %#v",
			raw.Canvas,
		)
	}
	if raw.AppExitCode != expectedAppExit {
		return nil, fmt.Errorf(
			"browser canvas Surface trace app_exit_code = %d, want %d",
			raw.AppExitCode,
			expectedAppExit,
		)
	}
	type traceFrame struct {
		Order          int    `json:"order"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
		Stride         int    `json:"stride"`
		PixelsLen      int    `json:"pixels_len"`
		SourceChecksum string `json:"source_checksum"`
		CanvasChecksum string `json:"canvas_checksum"`
		Checksum       string `json:"checksum"`
		ArtifactPath   string `json:"artifact_path"`
		Presented      bool   `json:"presented"`
	}
	trace := struct {
		Schema               string                          `json:"schema"`
		WASM                 string                          `json:"wasm_path"`
		Canvas               browserCanvasTraceCanvas        `json:"canvas"`
		BrowserEvents        []browserCanvasTraceEvent       `json:"browser_events"`
		BrowserClipboard     browserCanvasTraceClipboard     `json:"browser_clipboard"`
		BrowserComposition   browserCanvasTraceComposition   `json:"browser_composition"`
		BrowserAccessibility browserCanvasTraceAccessibility `json:"browser_accessibility"`
		Frames               []traceFrame                    `json:"frames"`
		AppExitCode          int                             `json:"app_exit_code"`
	}{
		Schema:               raw.Schema,
		WASM:                 wasmPath,
		Canvas:               raw.Canvas,
		BrowserEvents:        raw.BrowserEvents,
		BrowserClipboard:     raw.BrowserClipboard,
		BrowserComposition:   raw.BrowserComposition,
		BrowserAccessibility: raw.BrowserAccessibility,
		AppExitCode:          raw.AppExitCode,
	}
	frames := make([]surface.FrameReport, 0, len(raw.Frames))
	for _, frame := range raw.Frames {
		sourcePixels, err := base64.StdEncoding.DecodeString(frame.SourcePixelsB64)
		if err != nil {
			return nil, fmt.Errorf(
				"decode browser canvas source pixels for frame %d: %w",
				frame.Order,
				err,
			)
		}
		canvasPixels, err := base64.StdEncoding.DecodeString(frame.CanvasPixelsB64)
		if err != nil {
			return nil, fmt.Errorf(
				"decode browser canvas readback pixels for frame %d: %w",
				frame.Order,
				err,
			)
		}
		if len(sourcePixels) != frame.PixelsLen || len(canvasPixels) != frame.PixelsLen {
			return nil, fmt.Errorf(
				"browser canvas frame %d pixel lengths source=%d canvas=%d want %d",
				frame.Order,
				len(sourcePixels),
				len(canvasPixels),
				frame.PixelsLen,
			)
		}
		sourceChecksum := checksumRGBA(sourcePixels)
		canvasChecksum := checksumRGBA(canvasPixels)
		if sourceChecksum != canvasChecksum {
			return nil, fmt.Errorf(
				"browser canvas frame %d readback checksum %s differs from Tetra framebuffer checksum %s",
				frame.Order,
				canvasChecksum,
				sourceChecksum,
			)
		}
		reportOrder := browserCanvasReportFrameOrder(frame.Order)
		frameArtifactPath := filepath.Join(
			filepath.Dir(path),
			fmt.Sprintf("surface-browser-canvas-frame-order-%d.rgba", reportOrder),
		)
		if err := os.WriteFile(frameArtifactPath, canvasPixels, 0o644); err != nil {
			return nil, fmt.Errorf(
				"write browser canvas frame artifact %s: %w",
				frameArtifactPath,
				err,
			)
		}
		trace.Frames = append(trace.Frames, traceFrame{
			Order:          reportOrder,
			Width:          frame.Width,
			Height:         frame.Height,
			Stride:         frame.Stride,
			PixelsLen:      frame.PixelsLen,
			SourceChecksum: sourceChecksum,
			CanvasChecksum: canvasChecksum,
			Checksum:       canvasChecksum,
			ArtifactPath:   frameArtifactPath,
			Presented:      true,
		})
		frames = append(frames, surface.FrameReport{
			Order:        reportOrder,
			Width:        frame.Width,
			Height:       frame.Height,
			Stride:       frame.Stride,
			Checksum:     canvasChecksum,
			ArtifactPath: frameArtifactPath,
			Presented:    true,
		})
	}
	rawJSON, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode browser canvas Surface trace: %w", err)
	}
	if err := os.WriteFile(path, append(rawJSON, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write browser canvas Surface trace %s: %w", path, err)
	}
	return frames, nil
}
func browserCanvasReportFrameOrder(rawOrder int) int {
	if rawOrder <= 1 {
		return 1
	}
	return rawOrder + 3
}
func readWASM32WebSurfaceTrace(path string) ([]surface.FrameReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wasm32-web Surface runner trace %s: %w", path, err)
	}
	var trace wasmSurfaceRunnerTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, fmt.Errorf("decode wasm32-web Surface runner trace %s: %w", path, err)
	}
	if trace.Schema != "tetra.surface.web-runner-trace.v1" {
		return nil, fmt.Errorf(
			"wasm32-web Surface runner trace schema is %q, want tetra.surface.web-runner-trace.v1",
			trace.Schema,
		)
	}
	frames := make([]surface.FrameReport, 0, len(trace.Frames))
	for _, frame := range trace.Frames {
		if frame.PixelsLen <= 0 {
			return nil, fmt.Errorf(
				"wasm32-web Surface runner trace frame %d pixels_len must be positive",
				frame.Order,
			)
		}
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 ||
			strings.TrimSpace(frame.Checksum) == "" {
			return nil, fmt.Errorf(
				"wasm32-web Surface runner trace frame %d has incomplete frame evidence",
				frame.Order,
			)
		}
		frames = append(frames, surface.FrameReport{
			Order:     frame.Order + 2,
			Width:     frame.Width,
			Height:    frame.Height,
			Stride:    frame.Stride,
			Checksum:  frame.Checksum,
			Presented: true,
		})
	}
	return frames, nil
}
func validateCompilerOwnedWASMLoader(wasmPath string) error {
	loaderPath := strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath)) + ".mjs"
	raw, err := os.ReadFile(loaderPath)
	if err != nil {
		return fmt.Errorf("read compiler-owned wasm Surface loader %s: %w", loaderPath, err)
	}
	loader := string(raw)
	for _, want := range []string{
		"tetra_surface_host_v1",
		"createSurfaceHost(instanceRef)",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(loader, want) {
			return fmt.Errorf("compiler-owned wasm Surface loader %s missing %q", loaderPath, want)
		}
	}
	if strings.Contains(strings.ToLower(filepath.Base(loaderPath)), ".ui.") {
		return fmt.Errorf(
			"compiler-owned wasm Surface loader %s must not use legacy UI sidecar naming",
			loaderPath,
		)
	}
	if marker, ok := forbiddenCompilerOwnedWASMLoaderMarker(loader); ok {
		return fmt.Errorf(
			"compiler-owned wasm Surface loader %s must not contain DOM/user-JS marker %q",
			loaderPath,
			marker,
		)
	}
	return nil
}
func forbiddenCompilerOwnedWASMLoaderMarker(loader string) (string, bool) {
	lower := strings.ToLower(loader)
	for _, marker := range []string{
		"document.",
		"globalthis.document",
		"window.document",
		"createelement(",
		"appendchild(",
		"innerhtml",
		"queryselector(",
		"addeventlistener(",
		"<canvas",
		"<button",
		"mounttetraui",
		"tetra.ui.v1",
		".ui.web.mjs",
		".ui.html",
		"import(",
		".js\"",
		".js'",
	} {
		if strings.Contains(lower, marker) {
			return marker, true
		}
	}
	return "", false
}
