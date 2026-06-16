package surface

import (
	"fmt"
	"strings"
)

type PaintLayerReport struct {
	ID      string `json:"id"`
	BlockID int    `json:"block_id"`
	Kind    string `json:"kind"`
	Color   string `json:"color,omitempty"`
	Radius  int    `json:"radius,omitempty"`
	Width   int    `json:"width,omitempty"`
	Blur    int    `json:"blur,omitempty"`
	OffsetX int    `json:"offset_x,omitempty"`
	OffsetY int    `json:"offset_y,omitempty"`
	Opacity int    `json:"opacity,omitempty"`
}

type PaintCommandReport struct {
	Order     int        `json:"order"`
	Command   string     `json:"command"`
	LayerID   string     `json:"layer_id"`
	BlockID   int        `json:"block_id"`
	Rect      RectReport `json:"rect"`
	Clip      RectReport `json:"clip,omitempty"`
	Radius    int        `json:"radius,omitempty"`
	Opacity   int        `json:"opacity,omitempty"`
	Transform string     `json:"transform,omitempty"`
	Quality   string     `json:"quality"`
	Checksum  string     `json:"checksum"`
}

type RendererReport struct {
	Schema                       string                          `json:"schema"`
	Backend                      string                          `json:"backend"`
	ColorFormat                  string                          `json:"color_format"`
	QualityLevel                 string                          `json:"quality_level"`
	SoftwareRenderer             bool                            `json:"software_renderer"`
	GPUProductionClaim           bool                            `json:"gpu_production_claim"`
	BlurProductionClaim          bool                            `json:"blur_production_claim"`
	BackdropBlurProductionClaim  bool                            `json:"backdrop_blur_production_claim"`
	CommandOrder                 []string                        `json:"command_order"`
	CompositorLayers             []RendererCompositorLayerReport `json:"compositor_layers"`
	DirtyRects                   []RendererDirtyRectReport       `json:"dirty_rects"`
	Invalidations                []RendererInvalidationReport    `json:"invalidations"`
	CacheStats                   RendererCacheStatsReport        `json:"cache_stats"`
	UnsupportedEffectsRejected   []string                        `json:"unsupported_effects_rejected,omitempty"`
	DeterministicFrameChecksums  []string                        `json:"deterministic_frame_checksums,omitempty"`
	ReferenceFrameArtifactSHA256 string                          `json:"reference_frame_artifact_sha256,omitempty"`
}

type RendererCompositorLayerReport struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Order       int        `json:"order"`
	BlockID     int        `json:"block_id"`
	Rect        RectReport `json:"rect"`
	Clip        RectReport `json:"clip,omitempty"`
	ClipApplied bool       `json:"clip_applied,omitempty"`
	Opacity     int        `json:"opacity"`
	Transform   string     `json:"transform"`
	Checksum    string     `json:"checksum"`
}

type RendererDirtyRectReport struct {
	FrameOrder int        `json:"frame_order"`
	Rect       RectReport `json:"rect"`
	Reason     string     `json:"reason"`
	Checksum   string     `json:"checksum"`
}

type RendererInvalidationReport struct {
	Order     int        `json:"order"`
	BlockID   int        `json:"block_id"`
	Reason    string     `json:"reason"`
	DirtyRect RectReport `json:"dirty_rect"`
	Repaint   bool       `json:"repaint"`
}

type RendererCacheStatsReport struct {
	ID          string `json:"id"`
	Strategy    string `json:"strategy"`
	BudgetBytes int    `json:"budget_bytes"`
	UsedBytes   int    `json:"used_bytes"`
	EntryCount  int    `json:"entry_count"`
	Hits        int    `json:"hits"`
	Misses      int    `json:"misses"`
	Bounded     bool   `json:"bounded"`
}

func validateBlockPaintEvidence(report Report) []string {
	if !hasBlockPaintEvidence(report) {
		return nil
	}

	var issues []string
	expectedCommands := expectedRendererPaintCommandOrder()

	if report.PaintQualityLevel != "deterministic-software-paint-v1" {
		issues = append(issues, fmt.Sprintf("paint_quality_level is %q, want deterministic-software-paint-v1", report.PaintQualityLevel))
	}
	if report.PaintCacheBudgetBytes <= 0 || report.PaintCacheBudgetBytes > 1024*1024 {
		issues = append(issues, fmt.Sprintf("paint_cache_budget_bytes = %d, want 1..1048576", report.PaintCacheBudgetBytes))
	}
	if report.PaintUnsupportedBlur {
		issues = append(issues, "paint unsupported blur claim must be false")
	}
	if len(report.PaintLayers) == 0 {
		issues = append(issues, "paint_layers evidence is required")
	}
	if len(report.PaintCommands) == 0 {
		issues = append(issues, "paint_commands evidence is required")
	}
	if len(report.VisualFeatures) == 0 {
		issues = append(issues, "visual_features evidence is required")
	}
	issues = append(issues, validateRendererEvidence(report.Renderer, expectedCommands)...)

	for _, feature := range []string{"fill", "gradient", "image_fill", "border", "radius", "radius_clip", "shadow", "overlay", "outline", "text", "icon"} {
		if !visualFeatureContains(report.VisualFeatures, feature) {
			issues = append(issues, fmt.Sprintf("visual_features require %s", feature))
		}
	}

	layerByKind := map[string]PaintLayerReport{}
	layerIDs := map[string]bool{}
	hasLayerRadius := false
	for _, layer := range report.PaintLayers {
		kind := normalizePaintToken(layer.Kind)
		if strings.TrimSpace(layer.ID) == "" {
			issues = append(issues, "paint_layers id is required")
		} else if layerIDs[layer.ID] {
			issues = append(issues, fmt.Sprintf("paint_layers duplicate id %q", layer.ID))
		}
		layerIDs[layer.ID] = true
		if layer.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q block_id must be positive", layer.ID))
		}
		if kind == "" {
			issues = append(issues, fmt.Sprintf("paint_layers %q kind is required", layer.ID))
		}
		if layer.Radius > 0 {
			hasLayerRadius = true
		}
		if kind == "radius_clip" && layer.Radius <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q radius must be positive for radius_clip", layer.ID))
		}
		if (kind == "border" || kind == "outline") && layer.Width <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q width must be positive for %s", layer.ID, kind))
		}
		if kind == "shadow" && layer.Blur <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q blur must be positive for shadow approximation", layer.ID))
		}
		if kind == "overlay" && (layer.Opacity <= 0 || layer.Opacity > 255) {
			issues = append(issues, fmt.Sprintf("paint_layers %q opacity must be 1..255 for overlay", layer.ID))
		}
		if kind == "blur" || kind == "backdrop_blur" {
			issues = append(issues, "paint_layers unsupported blur/backdrop_blur layer is not allowed")
		}
		if _, exists := layerByKind[kind]; !exists {
			layerByKind[kind] = layer
		}
	}
	for _, kind := range expectedCommands {
		if _, ok := layerByKind[kind]; !ok {
			issues = append(issues, fmt.Sprintf("paint_layers require %s layer", kind))
		}
	}
	if !hasLayerRadius {
		issues = append(issues, "paint_layers require radius evidence")
	}

	seenCommands := map[string]bool{}
	lastOrder := 0
	if len(report.PaintCommands) > 0 && len(report.PaintCommands) != len(expectedCommands) {
		issues = append(issues, fmt.Sprintf("paint_commands count = %d, want %d deterministic renderer commands", len(report.PaintCommands), len(expectedCommands)))
	}
	for i, command := range report.PaintCommands {
		name := normalizePaintToken(command.Command)
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("paint_commands order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if i < len(expectedCommands) && name != expectedCommands[i] {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] command is %q, want deterministic %q", i, command.Command, expectedCommands[i]))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] rect dimensions must be positive", i))
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] layer_id is required", i))
		} else if !layerIDs[command.LayerID] {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] layer_id %q is not in paint_layers", i, command.LayerID))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] checksum must be sha256 evidence", i))
		}
		if command.Opacity < 0 || command.Opacity > 255 {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] opacity must be 0..255 when present", i))
		}
		if name == "fill" || name == "gradient" || name == "image_fill" || name == "border" || name == "radius_clip" || name == "overlay" || name == "outline" {
			if command.Radius <= 0 {
				issues = append(issues, fmt.Sprintf("paint_commands[%d] %s radius must be positive", i, name))
			}
		}
		seenCommands[name] = true
	}
	for _, command := range expectedCommands {
		if !seenCommands[command] {
			issues = append(issues, fmt.Sprintf("paint_commands require %s command", command))
		}
	}

	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "paint frame checksum evidence must show visual change")
	}
	for _, required := range []string{
		"block paint fill gradient image fill border radius clip shadow overlay outline text icon",
		"block paint deterministic command order",
		"block paint frame checksum changed",
		"block paint unsupported blur rejected",
		"block renderer software rgba contract",
		"block compositor dirty rect invalidation cache",
		"block renderer opacity transform clipped child",
		"block renderer gpu production claim rejected",
		"block renderer unsupported backdrop blur rejected",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("paint report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockPaintEvidence(report Report) bool {
	return len(report.PaintLayers) > 0 ||
		len(report.PaintCommands) > 0 ||
		len(report.VisualFeatures) > 0 ||
		strings.TrimSpace(report.PaintQualityLevel) != "" ||
		report.PaintCacheBudgetBytes != 0 ||
		report.PaintUnsupportedBlur ||
		report.Renderer != nil
}

func expectedRendererPaintCommandOrder() []string {
	return []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
}

func validateRendererEvidence(renderer *RendererReport, expectedCommands []string) []string {
	if renderer == nil {
		return []string{"renderer evidence is required for block paint production baseline"}
	}
	var issues []string
	if renderer.Schema != RendererFeatureSchemaV1 {
		issues = append(issues, fmt.Sprintf("renderer schema is %q, want %s", renderer.Schema, RendererFeatureSchemaV1))
	}
	if normalizePaintToken(renderer.Backend) != "software_rgba" {
		issues = append(issues, fmt.Sprintf("renderer backend is %q, want software-rgba", renderer.Backend))
	}
	if strings.ToLower(strings.TrimSpace(renderer.ColorFormat)) != "rgba8" {
		issues = append(issues, fmt.Sprintf("renderer color_format is %q, want rgba8", renderer.ColorFormat))
	}
	if renderer.QualityLevel != "deterministic-software-renderer-v1" {
		issues = append(issues, fmt.Sprintf("renderer quality_level is %q, want deterministic-software-renderer-v1", renderer.QualityLevel))
	}
	if !renderer.SoftwareRenderer {
		issues = append(issues, "renderer software_renderer must be true for the P05 software RGBA baseline")
	}
	if renderer.GPUProductionClaim {
		issues = append(issues, "renderer gpu production claim must be false without target-host GPU report")
	}
	if renderer.BlurProductionClaim {
		issues = append(issues, "renderer blur production claim must be false without supported backend evidence")
	}
	if renderer.BackdropBlurProductionClaim {
		issues = append(issues, "renderer backdrop blur production claim must be false without supported backend evidence")
	}
	if len(renderer.CommandOrder) != len(expectedCommands) {
		issues = append(issues, fmt.Sprintf("renderer command_order count = %d, want %d", len(renderer.CommandOrder), len(expectedCommands)))
	}
	for i, expected := range expectedCommands {
		if i >= len(renderer.CommandOrder) {
			issues = append(issues, fmt.Sprintf("renderer command_order missing %s", expected))
			continue
		}
		if normalizePaintToken(renderer.CommandOrder[i]) != expected {
			issues = append(issues, fmt.Sprintf("renderer command_order[%d] is %q, want %s", i, renderer.CommandOrder[i], expected))
		}
	}
	issues = append(issues, validateRendererCompositorLayers(renderer.CompositorLayers)...)
	issues = append(issues, validateRendererDirtyRects(renderer.DirtyRects)...)
	issues = append(issues, validateRendererInvalidations(renderer.Invalidations)...)
	issues = append(issues, validateRendererCacheStats(renderer.CacheStats)...)
	for _, effect := range []string{"gpu-production", "blur", "backdrop-blur"} {
		if !containsNormalizedPaint(renderer.UnsupportedEffectsRejected, effect) {
			issues = append(issues, fmt.Sprintf("renderer unsupported_effects_rejected requires %s", effect))
		}
	}
	if len(renderer.DeterministicFrameChecksums) < 2 {
		issues = append(issues, "renderer deterministic_frame_checksums require before and after checksums")
	} else {
		first := strings.TrimSpace(renderer.DeterministicFrameChecksums[0])
		second := strings.TrimSpace(renderer.DeterministicFrameChecksums[1])
		if !validChecksumLike(first) || !validChecksumLike(second) || first == second {
			issues = append(issues, "renderer deterministic_frame_checksums must be distinct sha256 evidence")
		}
	}
	if !validChecksumLike(renderer.ReferenceFrameArtifactSHA256) {
		issues = append(issues, "renderer reference_frame_artifact_sha256 must be sha256 evidence")
	}
	return issues
}

func validateRendererCompositorLayers(layers []RendererCompositorLayerReport) []string {
	if len(layers) == 0 {
		return []string{"renderer compositor_layers evidence is required"}
	}
	var issues []string
	ids := map[string]bool{}
	kinds := map[string]bool{}
	lastOrder := 0
	hasOpacityEvidence := false
	hasTransformEvidence := false
	hasClipEvidence := false
	for i, layer := range layers {
		id := strings.TrimSpace(layer.ID)
		kind := normalizePaintToken(layer.Kind)
		if id == "" {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] id is required", i))
		} else if ids[id] {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers duplicate id %q", id))
		}
		ids[id] = true
		if kind == "" {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] kind is required", i))
		}
		kinds[kind] = true
		if layer.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers order %d is not strictly greater than previous order %d", layer.Order, lastOrder))
		}
		lastOrder = layer.Order
		if layer.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] block_id must be positive", i))
		}
		if !validPositiveRect(layer.Rect) {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] rect dimensions must be positive", i))
		}
		if layer.Opacity <= 0 || layer.Opacity > 255 {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] opacity must be 1..255", i))
		}
		if layer.Opacity > 0 && layer.Opacity < 255 {
			hasOpacityEvidence = true
		}
		transform := strings.TrimSpace(strings.ToLower(layer.Transform))
		if transform == "" {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] transform is required", i))
		}
		if transform != "" && transform != "identity" && transform != "translate(0,0)" {
			hasTransformEvidence = true
		}
		if layer.ClipApplied {
			if !validPositiveRect(layer.Clip) {
				issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] clip dimensions must be positive when clip_applied", i))
			} else {
				hasClipEvidence = true
			}
		}
		if !validChecksumLike(layer.Checksum) {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers[%d] checksum must be sha256 evidence", i))
		}
	}
	for _, kind := range []string{"root", "content", "overlay", "text", "icon"} {
		if !kinds[kind] {
			issues = append(issues, fmt.Sprintf("renderer compositor_layers require %s layer", kind))
		}
	}
	if !hasOpacityEvidence {
		issues = append(issues, "renderer compositor_layers require opacity evidence below 255")
	}
	if !hasTransformEvidence {
		issues = append(issues, "renderer compositor_layers require transform evidence")
	}
	if !hasClipEvidence {
		issues = append(issues, "renderer compositor_layers require clipped child evidence")
	}
	return issues
}

func validateRendererDirtyRects(rects []RendererDirtyRectReport) []string {
	if len(rects) == 0 {
		return []string{"renderer dirty_rects evidence is required"}
	}
	var issues []string
	lastFrame := 0
	for i, rect := range rects {
		if rect.FrameOrder <= 0 {
			issues = append(issues, fmt.Sprintf("renderer dirty_rects[%d] frame_order must be positive", i))
		}
		if rect.FrameOrder < lastFrame {
			issues = append(issues, fmt.Sprintf("renderer dirty_rects[%d] frame_order must not go backwards", i))
		}
		lastFrame = rect.FrameOrder
		if !validPositiveRect(rect.Rect) {
			issues = append(issues, fmt.Sprintf("renderer dirty_rects[%d] rect dimensions must be positive", i))
		}
		if strings.TrimSpace(rect.Reason) == "" {
			issues = append(issues, fmt.Sprintf("renderer dirty_rects[%d] reason is required", i))
		}
		if !validChecksumLike(rect.Checksum) {
			issues = append(issues, fmt.Sprintf("renderer dirty_rects[%d] checksum must be sha256 evidence", i))
		}
	}
	return issues
}

func validateRendererInvalidations(invalidations []RendererInvalidationReport) []string {
	if len(invalidations) == 0 {
		return []string{"renderer invalidations evidence is required"}
	}
	var issues []string
	lastOrder := 0
	for i, invalidation := range invalidations {
		if invalidation.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("renderer invalidations order %d is not strictly greater than previous order %d", invalidation.Order, lastOrder))
		}
		lastOrder = invalidation.Order
		if invalidation.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("renderer invalidations[%d] block_id must be positive", i))
		}
		if strings.TrimSpace(invalidation.Reason) == "" {
			issues = append(issues, fmt.Sprintf("renderer invalidations[%d] reason is required", i))
		}
		if !validPositiveRect(invalidation.DirtyRect) {
			issues = append(issues, fmt.Sprintf("renderer invalidations[%d] dirty_rect dimensions must be positive", i))
		}
		if !invalidation.Repaint {
			issues = append(issues, fmt.Sprintf("renderer invalidations[%d] repaint must be true", i))
		}
	}
	return issues
}

func validateRendererCacheStats(cache RendererCacheStatsReport) []string {
	var issues []string
	if strings.TrimSpace(cache.ID) == "" {
		issues = append(issues, "renderer cache id is required")
	}
	if strings.TrimSpace(cache.Strategy) == "" {
		issues = append(issues, "renderer cache strategy is required")
	}
	if !cache.Bounded {
		issues = append(issues, "renderer cache must be bounded")
	}
	if cache.BudgetBytes <= 0 || cache.BudgetBytes > 1024*1024 {
		issues = append(issues, fmt.Sprintf("renderer cache budget_bytes = %d, want 1..1048576", cache.BudgetBytes))
	}
	if cache.UsedBytes < 0 || (cache.BudgetBytes > 0 && cache.UsedBytes > cache.BudgetBytes) {
		issues = append(issues, fmt.Sprintf("renderer cache used_bytes = %d exceeds budget %d", cache.UsedBytes, cache.BudgetBytes))
	}
	if cache.EntryCount <= 0 {
		issues = append(issues, "renderer cache entry_count must be positive")
	}
	if cache.Hits+cache.Misses <= 0 {
		issues = append(issues, "renderer cache requires hit/miss evidence")
	}
	return issues
}

func validPositiveRect(rect RectReport) bool {
	return rect.W > 0 && rect.H > 0
}

func containsNormalizedPaint(values []string, want string) bool {
	want = normalizePaintToken(want)
	for _, value := range values {
		if normalizePaintToken(value) == want {
			return true
		}
	}
	return false
}

func normalizePaintToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func visualFeatureContains(features []string, want string) bool {
	want = normalizePaintToken(want)
	for _, feature := range features {
		if normalizePaintToken(feature) == want {
			return true
		}
	}
	return false
}
