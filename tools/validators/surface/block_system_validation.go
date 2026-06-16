package surface

import (
	"fmt"
	"strings"
)

type BlockSystemReport struct {
	Schema         string                          `json:"schema"`
	QualityLevel   string                          `json:"quality_level"`
	Source         string                          `json:"source"`
	Renderer       string                          `json:"renderer"`
	GoldenSet      string                          `json:"golden_set"`
	FrameCount     int                             `json:"frame_count"`
	GoldenHash     string                          `json:"golden_hash"`
	Frames         []BlockSystemFrameReport        `json:"frames"`
	MemoryBudget   *BlockMemoryBudgetReport        `json:"memory_budget,omitempty"`
	NegativeGuards BlockSystemNegativeGuardsReport `json:"negative_guards"`
}

type BlockMemoryBudgetReport struct {
	Schema                   string   `json:"schema"`
	Scope                    string   `json:"scope"`
	BlockCount               int      `json:"block_count"`
	StressBlockCount         int      `json:"stress_block_count"`
	RenderLoopCount          int      `json:"render_loop_count"`
	StateLoopCount           int      `json:"state_loop_count"`
	MotionFrameCount         int      `json:"motion_frame_count"`
	InputEventCount          int      `json:"input_event_count"`
	PaintCommandCount        int      `json:"paint_command_count"`
	TextRenderCommandCount   int      `json:"text_render_command_count"`
	AssetRenderCommandCount  int      `json:"asset_render_command_count"`
	PeakFramebufferBytes     int      `json:"peak_framebuffer_bytes"`
	TotalFramebufferBytes    int      `json:"total_framebuffer_bytes"`
	FramebufferBudgetBytes   int      `json:"framebuffer_budget_bytes"`
	PaintCacheUsedBytes      int      `json:"paint_cache_used_bytes"`
	PaintCacheBudgetBytes    int      `json:"paint_cache_budget_bytes"`
	TextCacheUsedBytes       int      `json:"text_cache_used_bytes"`
	TextCacheBudgetBytes     int      `json:"text_cache_budget_bytes"`
	AssetCacheUsedBytes      int      `json:"asset_cache_used_bytes"`
	AssetCacheBudgetBytes    int      `json:"asset_cache_budget_bytes"`
	TotalCacheUsedBytes      int      `json:"total_cache_used_bytes"`
	TotalCacheBudgetBytes    int      `json:"total_cache_budget_bytes"`
	EstimatedAllocationBytes int      `json:"estimated_allocation_bytes"`
	RSSMeasured              bool     `json:"rss_measured"`
	PeakRSSBytes             int      `json:"peak_rss_bytes"`
	BoundedCaches            bool     `json:"bounded_caches"`
	UnboundedCacheRejected   bool     `json:"unbounded_cache_rejected"`
	StressScene              string   `json:"stress_scene"`
	PerformanceClaim         string   `json:"performance_claim"`
	NonClaims                []string `json:"nonclaims"`
}

type BlockSystemFrameReport struct {
	Order                 int    `json:"order"`
	Label                 string `json:"label"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	Stride                int    `json:"stride"`
	Checksum              string `json:"checksum"`
	RepeatChecksum        string `json:"repeat_checksum"`
	GoldenChecksum        string `json:"golden_checksum"`
	ArtifactPath          string `json:"artifact_path,omitempty"`
	Producer              string `json:"producer,omitempty"`
	EvidenceRole          string `json:"evidence_role,omitempty"`
	Precomputed           bool   `json:"precomputed,omitempty"`
	PaintEvidence         bool   `json:"paint_evidence"`
	LayoutEvidence        bool   `json:"layout_evidence"`
	AccessibilityEvidence bool   `json:"accessibility_evidence"`
}

type BlockSystemNegativeGuardsReport struct {
	MissingFrameChecksumRejected         bool `json:"missing_frame_checksum_rejected"`
	NondeterministicChecksumRejected     bool `json:"nondeterministic_checksum_rejected"`
	MissingPaintEvidenceRejected         bool `json:"missing_paint_evidence_rejected"`
	MissingLayoutEvidenceRejected        bool `json:"missing_layout_evidence_rejected"`
	MissingAccessibilityEvidenceRejected bool `json:"missing_accessibility_evidence_rejected"`
}

func validateBlockCorePrimitiveEvidence(report Report) []string {
	if report.BlockSystem == nil {
		return nil
	}

	var issues []string
	check := func(location string, value string) {
		if token, ok := forbiddenBlockCorePrimitiveToken(value); ok {
			issues = append(issues, fmt.Sprintf("block_system fake core primitive %s rejected in %s %q; use Block parameters instead", token, location, value))
		}
	}
	for i, component := range report.Components {
		check(fmt.Sprintf("components[%d].id", i), component.ID)
		check(fmt.Sprintf("components[%d].type", i), component.Type)
	}
	if report.BlockGraph != nil {
		for i, node := range report.BlockGraph.Nodes {
			check(fmt.Sprintf("block_graph.nodes[%d].name", i), node.Name)
		}
	}
	if report.BlockAccessibilityTree != nil {
		for i, node := range report.BlockAccessibilityTree.Nodes {
			check(fmt.Sprintf("block_accessibility_tree.nodes[%d].name", i), node.Name)
		}
	}
	return issues
}

func forbiddenBlockCorePrimitiveToken(value string) (string, bool) {
	for _, field := range blockPrimitiveNameFields(value) {
		for _, token := range []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"} {
			if strings.EqualFold(field, token) {
				return token, true
			}
		}
	}
	return "", false
}

func blockPrimitiveNameFields(value string) []string {
	replacer := strings.NewReplacer(
		".", " ",
		"/", " ",
		"\\", " ",
		"_", " ",
		"-", " ",
		":", " ",
	)
	return strings.Fields(replacer.Replace(strings.TrimSpace(value)))
}

func validateBlockSystemEvidence(report Report) []string {
	if report.BlockSystem == nil {
		return nil
	}

	system := report.BlockSystem
	var issues []string
	if system.Schema != "tetra.surface.block-system.v1" {
		issues = append(issues, fmt.Sprintf("block_system schema is %q, want tetra.surface.block-system.v1", system.Schema))
	}
	expectedQuality := ""
	expectedRenderer := ""
	requiredCases := []string{}
	linuxRealWindowBlockSystem := system.QualityLevel == "linux-x64-real-window-block-system-v1"
	wasmBrowserCanvasBlockSystem := system.QualityLevel == "wasm32-web-browser-canvas-block-system-v1"
	if linuxRealWindowBlockSystem &&
		(report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" || !isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level)) {
		issues = append(issues, "linux-x64 real-window block_system requires linux-x64 real-window runtime evidence")
	}
	if wasmBrowserCanvasBlockSystem &&
		(report.Target != "wasm32-web" || report.Runtime != "surface-wasm32-web" || report.HostEvidence.Level != "wasm32-web-browser-canvas-input") {
		issues = append(issues, "wasm32-web browser-canvas block_system requires wasm32-web browser-canvas runtime evidence")
	}
	switch {
	case report.Target == "headless" && report.Runtime == "surface-headless" && !linuxRealWindowBlockSystem && !wasmBrowserCanvasBlockSystem:
		expectedQuality = "deterministic-headless-block-system-v1"
		expectedRenderer = "software-rgba-headless"
		requiredCases = []string{
			"block system headless golden checksums",
			"block system deterministic repeat checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
		}
	case report.Target == "linux-x64" && report.Runtime == "surface-linux-x64" && isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level):
		expectedQuality = "linux-x64-real-window-block-system-v1"
		expectedRenderer = "wayland-shm-rgba"
		requiredCases = []string{
			"linux-x64 real-window surface",
			"linux-x64 native input event pump",
			"linux-x64 real-window resize event",
			"linux-x64 real-window close event",
			"block system linux-x64 real-window frame presentation",
			"block system linux-x64 native input state transition",
			"block system linux-x64 real-window checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
			"block system missing real-window presentation rejected",
			"block system missing native input state transition rejected",
		}
		if !report.HostEvidence.RealWindow || !report.HostEvidence.NativeInput || !report.HostEvidence.Framebuffer {
			issues = append(issues, "linux-x64 real-window block_system requires framebuffer, real_window, and native_input host evidence")
		}
		if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
			issues = append(issues, "linux-x64 real-window block_system requires order-5 400x240 presented frame evidence")
		}
		if !eventKindContains(report.Events, "mouse_up") || !eventKindContains(report.Events, "key_down") ||
			!eventKindContains(report.Events, "resize") || !eventKindContains(report.Events, "close") {
			issues = append(issues, "linux-x64 real-window block_system requires native input, resize, and close event evidence")
		}
		if !hasTransition(report.StateTransitions, "SubmitBlock", "pressed") {
			issues = append(issues, "linux-x64 real-window block_system requires native input state transition evidence")
		}
	case report.Target == "wasm32-web" && report.Runtime == "surface-wasm32-web" && report.HostEvidence.Level == "wasm32-web-browser-canvas-input":
		expectedQuality = "wasm32-web-browser-canvas-block-system-v1"
		expectedRenderer = "browser-canvas-rgba"
		requiredCases = []string{
			"wasm32-web browser canvas surface",
			"wasm32-web browser canvas RGBA readback",
			"wasm32-web browser canvas pointer input",
			"wasm32-web browser canvas keyboard input",
			"wasm32-web browser canvas resize input",
			"wasm32-web browser canvas text input",
			"wasm32-web Surface Host ABI imports",
			"compiler-owned wasm Surface loader",
			"compiler-owned browser canvas Surface host",
			"block system wasm32-web browser-canvas frame readback",
			"block system wasm32-web browser-canvas native input state transition",
			"block system wasm32-web browser-canvas checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
			"block system browser-canvas node runtime substitution rejected",
			"block system browser-canvas missing RGBA readback rejected",
			"block system browser-canvas script sidecar artifact rejected",
			"block system browser-canvas html visual sidecar artifact rejected",
		}
		if !report.HostEvidence.Framebuffer || !report.HostEvidence.NativeInput || !report.HostEvidence.BrowserCanvas || !report.HostEvidence.BrowserInput {
			issues = append(issues, "wasm32-web browser-canvas block_system requires framebuffer, native_input, browser_canvas, and browser_input host evidence")
		}
		if report.HostEvidence.RealWindow {
			issues = append(issues, "wasm32-web browser-canvas block_system must not claim OS real_window evidence")
		}
		if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
			issues = append(issues, "wasm32-web browser-canvas block_system requires order-5 400x240 RGBA readback frame evidence")
		}
		if !eventKindContains(report.Events, "mouse_up") || !eventKindContains(report.Events, "key_down") ||
			!eventKindContains(report.Events, "resize") || !eventKindContains(report.Events, "text_input") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires browser input, resize, and text input event evidence")
		}
		if !hasTransition(report.StateTransitions, "SubmitBlock", "pressed") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires browser native input state transition evidence")
		}
		if !hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chrom", "scenario=block-system") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires Chromium-compatible browser-canvas app process evidence")
		}
		issues = append(issues, validateBlockSystemBrowserCanvasArtifacts(report.Artifacts)...)
	default:
		issues = append(issues, "block_system requires deterministic headless, linux-x64 real-window, or wasm32-web browser-canvas runtime evidence")
	}
	if expectedQuality != "" && system.QualityLevel != expectedQuality {
		issues = append(issues, fmt.Sprintf("block_system quality_level is %q, want %s", system.QualityLevel, expectedQuality))
	}
	if normalizeEvidencePath(system.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_system source %q must match report source %q", system.Source, report.Source))
	}
	if expectedRenderer != "" && system.Renderer != expectedRenderer {
		issues = append(issues, fmt.Sprintf("block_system renderer is %q, want %s", system.Renderer, expectedRenderer))
	}
	if strings.TrimSpace(system.GoldenSet) == "" {
		issues = append(issues, "block_system golden_set is required")
	}
	if !validChecksumLike(system.GoldenHash) {
		issues = append(issues, "block_system golden_hash must be sha256 evidence")
	}
	if system.FrameCount != len(system.Frames) {
		issues = append(issues, fmt.Sprintf("block_system frame_count = %d, want len(frames) %d", system.FrameCount, len(system.Frames)))
	}
	if len(system.Frames) == 0 {
		issues = append(issues, "block_system frame golden evidence is required")
	}
	if system.MemoryBudget == nil {
		issues = append(issues, "block_system memory_budget is required")
	} else {
		issues = append(issues, validateBlockMemoryBudgetEvidence(report, *system.MemoryBudget)...)
	}

	if len(report.PaintLayers) == 0 || len(report.PaintCommands) == 0 || len(report.VisualFeatures) == 0 {
		issues = append(issues, "block_system requires paint evidence")
	}
	if len(report.LayoutConstraints) == 0 || len(report.LayoutPasses) == 0 || len(report.LayoutScrolls) == 0 {
		issues = append(issues, "block_system requires layout evidence")
	}
	if !hasBlockTextEvidence(report) {
		issues = append(issues, "block_system requires text measurement evidence")
	}
	if !hasBlockStateEvidence(report) {
		issues = append(issues, "block_system requires state selector evidence")
	}
	if !hasBlockMotionEvidence(report) {
		issues = append(issues, "block_system requires motion frame evidence")
	}
	if !hasBlockAssetEvidence(report) {
		issues = append(issues, "block_system requires asset manifest/cache evidence")
	}
	if report.BlockGraph == nil || report.BlockAccessibilityTree == nil {
		issues = append(issues, "block_system requires accessibility evidence")
	}

	reportFrames := map[int]FrameReport{}
	for _, frame := range report.Frames {
		reportFrames[frame.Order] = frame
	}
	lastOrder := 0
	for i, frame := range system.Frames {
		if frame.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("block_system frames order %d is not strictly greater than previous order %d", frame.Order, lastOrder))
		}
		lastOrder = frame.Order
		if strings.TrimSpace(frame.Label) == "" {
			issues = append(issues, fmt.Sprintf("block_system frames[%d] label is required", i))
		}
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
			issues = append(issues, fmt.Sprintf("block_system frame %d dimensions and stride must be positive", frame.Order))
		}
		if !validChecksumLike(frame.Checksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d checksum must be sha256 evidence", frame.Order))
		}
		if !validChecksumLike(frame.RepeatChecksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d repeat_checksum must be sha256 evidence", frame.Order))
		}
		if !validChecksumLike(frame.GoldenChecksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d golden_checksum must be sha256 evidence", frame.Order))
		}
		if strings.TrimSpace(frame.Checksum) != "" && strings.TrimSpace(frame.RepeatChecksum) != "" && frame.Checksum != frame.RepeatChecksum {
			issues = append(issues, fmt.Sprintf("block_system frame %d nondeterministic repeat checksum %q, want %q", frame.Order, frame.RepeatChecksum, frame.Checksum))
		}
		if strings.TrimSpace(frame.Checksum) != "" && strings.TrimSpace(frame.GoldenChecksum) != "" && frame.Checksum != frame.GoldenChecksum {
			issues = append(issues, fmt.Sprintf("block_system frame %d golden checksum %q, want %q", frame.Order, frame.GoldenChecksum, frame.Checksum))
		}
		reportFrame, ok := reportFrames[frame.Order]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_system frame %d is missing from runtime frame evidence", frame.Order))
		} else {
			if reportFrame.Width != frame.Width || reportFrame.Height != frame.Height || reportFrame.Stride != frame.Stride {
				issues = append(issues, fmt.Sprintf("block_system frame %d dimensions do not match runtime frame evidence", frame.Order))
			}
			if reportFrame.Checksum != frame.Checksum {
				issues = append(issues, fmt.Sprintf("block_system frame %d checksum %q must match runtime frame checksum %q", frame.Order, frame.Checksum, reportFrame.Checksum))
			}
			if !reportFrame.Presented {
				issues = append(issues, fmt.Sprintf("block_system frame %d requires presented runtime frame evidence", frame.Order))
			}
			issues = append(issues, validateBlockSystemFrameProvenance(frame, reportFrame)...)
		}
		if !frame.PaintEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing paint evidence", frame.Order))
		}
		if !frame.LayoutEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing layout evidence", frame.Order))
		}
		if !frame.AccessibilityEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing accessibility evidence", frame.Order))
		}
	}
	if len(system.Frames) != len(report.Frames) {
		issues = append(issues, fmt.Sprintf("block_system frames length %d must match runtime frames length %d", len(system.Frames), len(report.Frames)))
	}

	issues = append(issues, validateBlockSystemNegativeGuards(system.NegativeGuards)...)
	requiredCases = append(requiredCases,
		"block system bounded memory budget",
		"block system stress render loop budget",
		"block system performance nonclaim",
	)
	for _, required := range requiredCases {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_system report requires %s evidence", required))
		}
	}
	return issues
}

func validateBlockMemoryBudgetEvidence(report Report, budget BlockMemoryBudgetReport) []string {
	var issues []string
	if budget.Schema != "tetra.surface.block-memory-budget.v1" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget schema is %q, want tetra.surface.block-memory-budget.v1", budget.Schema))
	}
	if budget.Scope != "surface-block-system-local-budget-v1" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget scope is %q, want surface-block-system-local-budget-v1", budget.Scope))
	}
	if budget.BlockCount != len(report.Components) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget block_count = %d, want component count %d", budget.BlockCount, len(report.Components)))
	}
	if budget.StressBlockCount < 128 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget stress_block_count = %d, want at least 128", budget.StressBlockCount))
	}
	if budget.RenderLoopCount < 16 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget render_loop_count = %d, want at least 16", budget.RenderLoopCount))
	}
	if budget.StateLoopCount < len(report.StateTransitions) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget state_loop_count = %d, want at least state transition count %d", budget.StateLoopCount, len(report.StateTransitions)))
	}
	if budget.MotionFrameCount != len(report.MotionFrames) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget motion_frame_count = %d, want %d", budget.MotionFrameCount, len(report.MotionFrames)))
	}
	if budget.InputEventCount != len(report.Events) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget input_event_count = %d, want %d", budget.InputEventCount, len(report.Events)))
	}
	if budget.PaintCommandCount != len(report.PaintCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_command_count = %d, want %d", budget.PaintCommandCount, len(report.PaintCommands)))
	}
	if budget.TextRenderCommandCount != len(report.TextRenderCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_render_command_count = %d, want %d", budget.TextRenderCommandCount, len(report.TextRenderCommands)))
	}
	if budget.AssetRenderCommandCount != len(report.BlockAssetRenderCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_render_command_count = %d, want %d", budget.AssetRenderCommandCount, len(report.BlockAssetRenderCommands)))
	}
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotals(report.Frames)
	if budget.PeakFramebufferBytes != peakFramebufferBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget peak_framebuffer_bytes = %d, want %d", budget.PeakFramebufferBytes, peakFramebufferBytes))
	}
	if budget.TotalFramebufferBytes != totalFramebufferBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_framebuffer_bytes = %d, want %d", budget.TotalFramebufferBytes, totalFramebufferBytes))
	}
	if budget.FramebufferBudgetBytes < peakFramebufferBytes || budget.FramebufferBudgetBytes > 16*1024*1024 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget framebuffer_budget_bytes = %d outside bounded range for peak %d", budget.FramebufferBudgetBytes, peakFramebufferBytes))
	}
	expectedPaintUsed := len(report.PaintCommands) * 2048
	expectedTextUsed := blockGlyphCacheUsedBytes(report.GlyphCaches)
	expectedAssetUsed := report.BlockAssetCache.UsedBytes
	if budget.PaintCacheUsedBytes != expectedPaintUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_cache_used_bytes = %d, want %d", budget.PaintCacheUsedBytes, expectedPaintUsed))
	}
	if budget.PaintCacheBudgetBytes != report.PaintCacheBudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_cache_budget_bytes = %d, want %d", budget.PaintCacheBudgetBytes, report.PaintCacheBudgetBytes))
	}
	if budget.TextCacheUsedBytes != expectedTextUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_cache_used_bytes = %d, want %d", budget.TextCacheUsedBytes, expectedTextUsed))
	}
	if budget.TextCacheBudgetBytes != report.TextCacheBudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_cache_budget_bytes = %d, want %d", budget.TextCacheBudgetBytes, report.TextCacheBudgetBytes))
	}
	if budget.AssetCacheUsedBytes != expectedAssetUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_cache_used_bytes = %d, want %d", budget.AssetCacheUsedBytes, expectedAssetUsed))
	}
	if budget.AssetCacheBudgetBytes != report.BlockAssetCache.BudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_cache_budget_bytes = %d, want %d", budget.AssetCacheBudgetBytes, report.BlockAssetCache.BudgetBytes))
	}
	expectedTotalCacheUsed := expectedPaintUsed + expectedTextUsed + expectedAssetUsed
	expectedTotalCacheBudget := report.PaintCacheBudgetBytes + report.TextCacheBudgetBytes + report.BlockAssetCache.BudgetBytes
	if budget.TotalCacheUsedBytes != expectedTotalCacheUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_cache_used_bytes = %d, want %d", budget.TotalCacheUsedBytes, expectedTotalCacheUsed))
	}
	if budget.TotalCacheBudgetBytes != expectedTotalCacheBudget {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_cache_budget_bytes = %d, want %d", budget.TotalCacheBudgetBytes, expectedTotalCacheBudget))
	}
	if expectedTotalCacheBudget <= 0 || expectedTotalCacheUsed < 0 || expectedTotalCacheUsed > expectedTotalCacheBudget {
		issues = append(issues, "block_system memory_budget cache totals must be bounded and within budget")
	}
	if budget.EstimatedAllocationBytes < totalFramebufferBytes+expectedTotalCacheUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget estimated_allocation_bytes = %d, want at least framebuffer+cache %d", budget.EstimatedAllocationBytes, totalFramebufferBytes+expectedTotalCacheUsed))
	}
	if budget.RSSMeasured {
		if budget.PeakRSSBytes <= 0 || budget.PeakRSSBytes > 512*1024*1024 {
			issues = append(issues, fmt.Sprintf("block_system memory_budget peak_rss_bytes = %d outside scoped RSS range", budget.PeakRSSBytes))
		}
	} else if budget.PeakRSSBytes != 0 {
		issues = append(issues, "block_system memory_budget peak_rss_bytes must be 0 when rss_measured is false")
	}
	if !budget.BoundedCaches {
		issues = append(issues, "block_system memory_budget bounded_caches must be true")
	}
	if !budget.UnboundedCacheRejected {
		issues = append(issues, "block_system memory_budget unbounded_cache_rejected must be true")
	}
	if strings.TrimSpace(budget.StressScene) != "deterministic-block-stress-128" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget stress_scene is %q, want deterministic-block-stress-128", budget.StressScene))
	}
	if strings.TrimSpace(budget.PerformanceClaim) != "none" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget performance_claim is %q, want none", budget.PerformanceClaim))
	}
	issues = append(issues, forbiddenBlockPerformanceClaimIssues("block_system memory_budget", budget.PerformanceClaim)...)
	for _, claim := range budget.NonClaims {
		issues = append(issues, forbiddenBlockPerformanceClaimIssues("block_system memory_budget nonclaims", claim)...)
	}
	for _, required := range []string{
		"no Electron comparison benchmark",
		"no broad performance superiority claim",
		"RSS is optional host evidence",
	} {
		if !containsTextFold(budget.NonClaims, required) {
			issues = append(issues, fmt.Sprintf("block_system memory_budget nonclaims missing %q", required))
		}
	}
	return issues
}

func blockFramebufferByteTotals(frames []FrameReport) (int, int) {
	peak := 0
	total := 0
	for _, frame := range frames {
		bytes := frame.Height * frame.Stride
		if bytes > peak {
			peak = bytes
		}
		total += bytes
	}
	return peak, total
}

func blockGlyphCacheUsedBytes(caches []GlyphCacheReport) int {
	total := 0
	for _, cache := range caches {
		total += cache.UsedBytes
	}
	return total
}

func containsTextFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), want) {
			return true
		}
	}
	return false
}

func forbiddenBlockPerformanceClaimIssues(label string, fields ...string) []string {
	var issues []string
	for _, field := range fields {
		lower := strings.ToLower(field)
		for _, marker := range []string{
			"faster than " + "electron",
			"zero " + "overhead",
			"zero-cost " + "ui",
			"zero cost " + "ui",
			"fastest " + "ui",
		} {
			if strings.Contains(lower, marker) {
				issues = append(issues, fmt.Sprintf("%s contains forbidden performance claim %q", label, marker))
			}
		}
	}
	return issues
}

func validateBlockSystemBrowserCanvasArtifacts(artifacts []ArtifactReport) []string {
	var issues []string
	for _, artifact := range artifacts {
		kind := strings.ToLower(strings.TrimSpace(artifact.Kind))
		path := strings.ToLower(normalizeEvidencePath(artifact.Path))
		if strings.Contains(kind, "user-js") || strings.Contains(path, ".user.js") || strings.HasSuffix(path, "/user.js") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include user JS artifact %q", artifact.Path))
		}
		if strings.Contains(kind, "dom-ui") || strings.Contains(path, ".dom.") || strings.HasSuffix(path, ".html") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include DOM UI artifact %q", artifact.Path))
		}
		if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".mjs") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include user JS artifact %q", artifact.Path))
		}
	}
	return issues
}

func validateBlockSystemNegativeGuards(guards BlockSystemNegativeGuardsReport) []string {
	var missing []string
	if !guards.MissingFrameChecksumRejected {
		missing = append(missing, "missing_frame_checksum_rejected")
	}
	if !guards.NondeterministicChecksumRejected {
		missing = append(missing, "nondeterministic_checksum_rejected")
	}
	if !guards.MissingPaintEvidenceRejected {
		missing = append(missing, "missing_paint_evidence_rejected")
	}
	if !guards.MissingLayoutEvidenceRejected {
		missing = append(missing, "missing_layout_evidence_rejected")
	}
	if !guards.MissingAccessibilityEvidenceRejected {
		missing = append(missing, "missing_accessibility_evidence_rejected")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("block_system negative_guards missing %s", strings.Join(missing, ", "))}
}
