package surface

import (
	"fmt"
	"strings"
)

type RenderCommandStreamReport struct {
	Schema                        string                `json:"schema"`
	Source                        string                `json:"source"`
	SurfaceScope                  string                `json:"surface_scope"`
	Producer                      string                `json:"producer"`
	QualityLevel                  string                `json:"quality_level"`
	Renderer                      string                `json:"renderer"`
	DerivedFromBlockSceneSnapshot bool                  `json:"derived_from_block_scene_snapshot"`
	BlockSceneHash                string                `json:"block_scene_hash"`
	FrameChecksum                 string                `json:"frame_checksum"`
	CommandStreamHash             string                `json:"command_stream_hash"`
	CommandCount                  int                   `json:"command_count"`
	SourceLinked                  bool                  `json:"source_linked"`
	HandcraftedFixture            bool                  `json:"handcrafted_fixture"`
	Commands                      []RenderCommandReport `json:"commands"`
}

type RenderCommandReport struct {
	Order          int        `json:"order"`
	Command        string     `json:"command"`
	Source         string     `json:"source"`
	SourceNodeID   string     `json:"source_node_id"`
	Recipe         string     `json:"recipe"`
	LayerID        string     `json:"layer_id"`
	BlockID        int        `json:"block_id"`
	Rect           RectReport `json:"rect"`
	Clip           RectReport `json:"clip,omitempty"`
	Color          string     `json:"color,omitempty"`
	Radius         int        `json:"radius,omitempty"`
	Width          int        `json:"width,omitempty"`
	Blur           int        `json:"blur,omitempty"`
	OffsetX        int        `json:"offset_x,omitempty"`
	OffsetY        int        `json:"offset_y,omitempty"`
	Opacity        int        `json:"opacity,omitempty"`
	Quality        string     `json:"quality"`
	AssetID        string     `json:"asset_id,omitempty"`
	TextLen        int        `json:"text_len,omitempty"`
	RasterFormat   string     `json:"raster_format,omitempty"`
	RasterHash     string     `json:"raster_hash,omitempty"`
	RasterWidth    int        `json:"raster_width,omitempty"`
	RasterHeight   int        `json:"raster_height,omitempty"`
	RasterCoverage int        `json:"raster_coverage,omitempty"`
	MarkerOnly     bool       `json:"marker_only,omitempty"`
	Checksum       string     `json:"checksum"`
}

func validateRenderCommandStreamEvidence(report Report) []string {
	if report.RenderCommandStream == nil {
		return nil
	}

	stream := report.RenderCommandStream
	var issues []string
	if stream.Schema != "tetra.surface.render-command-stream.v1" {
		issues = append(issues, fmt.Sprintf("render_command_stream schema is %q, want tetra.surface.render-command-stream.v1", stream.Schema))
	}
	if normalizeEvidencePath(stream.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("render_command_stream source %q must match report source %q", stream.Source, report.Source))
	}
	if stream.SurfaceScope != "surface-morph-rendered-beauty-linux-web" {
		issues = append(issues, fmt.Sprintf("render_command_stream surface_scope is %q, want surface-morph-rendered-beauty-linux-web", stream.SurfaceScope))
	}
	if strings.TrimSpace(stream.Producer) == "" {
		issues = append(issues, "render_command_stream producer is required")
	}
	if stream.QualityLevel != "deterministic-render-command-stream-v1" {
		issues = append(issues, fmt.Sprintf("render_command_stream quality_level is %q, want deterministic-render-command-stream-v1", stream.QualityLevel))
	}
	if !stringSliceContainsFold([]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"}, stream.Renderer) {
		issues = append(issues, fmt.Sprintf("render_command_stream renderer %q is not allowed", stream.Renderer))
	}
	if !stream.DerivedFromBlockSceneSnapshot {
		issues = append(issues, "render_command_stream derived_from_block_scene_snapshot must be true")
	}
	if report.BlockSceneSnapshot == nil {
		issues = append(issues, "render_command_stream requires block_scene_snapshot evidence")
	} else if stream.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(issues, "render_command_stream block_scene_hash must match block_scene_snapshot.block_scene_hash")
	}
	if !validSHA256Digest(stream.BlockSceneHash) {
		issues = append(issues, "render_command_stream block_scene_hash must be sha256 evidence")
	}
	if !validChecksumLike(stream.FrameChecksum) {
		issues = append(issues, "render_command_stream frame_checksum must be sha256 evidence")
	} else if !frameChecksumPresent(report.Frames, stream.FrameChecksum) {
		issues = append(issues, "render_command_stream frame_checksum must match a presented report frame")
	}
	if !validSHA256Digest(stream.CommandStreamHash) {
		issues = append(issues, "render_command_stream command_stream_hash must be sha256 evidence")
	}
	if stream.CommandCount <= 0 {
		issues = append(issues, "render_command_stream command_count must be positive")
	}
	if stream.CommandCount != len(stream.Commands) {
		issues = append(issues, fmt.Sprintf("render_command_stream command_count = %d, want len(commands) %d", stream.CommandCount, len(stream.Commands)))
	}
	if !stream.SourceLinked {
		issues = append(issues, "render_command_stream source_linked must be true")
	}
	if stream.HandcraftedFixture {
		issues = append(issues, "render_command_stream handcrafted_fixture must be false")
	}
	issues = append(issues, validateRenderCommands(report, stream.Commands)...)
	return issues
}

func validateRenderCommands(report Report, commands []RenderCommandReport) []string {
	var issues []string
	nodeByID := map[int]BlockSceneNodeReport{}
	if report.BlockSceneSnapshot != nil {
		for _, node := range report.BlockSceneSnapshot.Nodes {
			nodeByID[node.BlockID] = node
		}
	}
	seenKinds := map[string]bool{}
	lastOrder := 0
	for i, command := range commands {
		name := normalizeRenderCommandToken(command.Command)
		if command.Order != i+1 {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] order = %d, want %d", i, command.Order, i+1))
		}
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("render_command_stream command order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if !isSupportedRenderCommand(name) {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] command %q is not supported", i, command.Command))
		}
		if normalizeEvidencePath(command.Source) != normalizeEvidencePath(report.Source) {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] source %q must match report source %q", i, command.Source, report.Source))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] block_id must be positive", i))
		}
		node, hasNode := nodeByID[command.BlockID]
		if len(nodeByID) > 0 && !hasNode {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] block_id %d is not in block_scene_snapshot", i, command.BlockID))
		}
		if strings.TrimSpace(command.SourceNodeID) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] source_node_id is required", i))
		} else if hasNode && command.SourceNodeID != fmt.Sprintf("block:%d", node.BlockID) {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] source_node_id %q must identify block:%d", i, command.SourceNodeID, node.BlockID))
		}
		if strings.TrimSpace(command.Recipe) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] recipe is required", i))
		} else if hasNode && command.Recipe != node.Recipe {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] recipe %q must match block_scene_snapshot node recipe %q", i, command.Recipe, node.Recipe))
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] layer_id is required", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] rect dimensions must be positive", i))
		}
		if command.Opacity < 0 || command.Opacity > 255 {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] opacity must be 0..255", i))
		}
		if name != "radius_clip" && strings.TrimSpace(command.Color) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] color is required for renderer-owned pixels", i))
		}
		if name == "radius_clip" && command.Radius <= 0 {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] radius_clip radius must be positive", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] quality is required", i))
		}
		if name == "text" {
			issues = append(issues, validateRasterProof(
				fmt.Sprintf("render_command_stream commands[%d]", i),
				"builtin-5x7-alpha-mask-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if name == "icon" {
			issues = append(issues, validateRasterProof(
				fmt.Sprintf("render_command_stream commands[%d]", i),
				"builtin-icon-mask-raster-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("render_command_stream commands[%d] checksum must be sha256 evidence", i))
		}
		seenKinds[name] = true
	}
	for _, required := range []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"} {
		if !seenKinds[required] {
			issues = append(issues, fmt.Sprintf("render_command_stream commands require %s command", required))
		}
	}
	return issues
}

func normalizeRenderCommandToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isSupportedRenderCommand(value string) bool {
	switch normalizeRenderCommandToken(value) {
	case "fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon":
		return true
	default:
		return false
	}
}

func frameChecksumPresent(frames []FrameReport, checksum string) bool {
	checksum = strings.TrimSpace(checksum)
	for _, frame := range frames {
		if frame.Presented && strings.TrimSpace(frame.Checksum) == checksum {
			return true
		}
	}
	return false
}
