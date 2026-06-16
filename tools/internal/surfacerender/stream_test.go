package surfacerender

import (
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestBuildRenderCommandStreamIncludesGlyphIconRasterEvidence(t *testing.T) {
	source := "examples/surface_morph_command_palette.tetra"
	snapshot := &surface.BlockSceneSnapshotReport{
		BlockSceneHash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Nodes: []surface.BlockSceneNodeReport{
			{
				BlockID:  2,
				ParentID: 1,
				Recipe:   "morph.search_input",
				Layout:   &surface.BlockSceneLayoutSpecReport{Mode: "row", X: 16, Y: 16, W: 288, H: 48},
				Paint: &surface.BlockScenePaintSpecReport{Layers: []surface.BlockScenePaintLayerSpecReport{
					{Kind: "fill", Radius: 8, Opacity: 255},
					{Kind: "gradient", Radius: 8, Opacity: 255},
					{Kind: "image_fill", Radius: 8, Opacity: 96},
					{Kind: "border", Radius: 8, Width: 1, Opacity: 255},
					{Kind: "radius_clip", Radius: 8, Opacity: 255},
					{Kind: "shadow", Radius: 8, Blur: 12, Opacity: 88},
					{Kind: "overlay", Radius: 8, Opacity: 102},
					{Kind: "outline", Radius: 8, Width: 1, Opacity: 255},
					{Kind: "text", Opacity: 255},
					{Kind: "icon", Opacity: 255},
				}},
				Text:  &surface.BlockSceneTextSpecReport{TextLen: 12},
				Image: &surface.BlockSceneImageSpecReport{AssetID: "search-icon", Mode: "template", Opacity: 255},
			},
		},
	}
	frames := []surface.FrameReport{{Order: 1, Checksum: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", Presented: true}}

	first := BuildCommandStream(source, "software-rgba-headless", snapshot, frames)
	second := BuildCommandStream(source, "software-rgba-headless", snapshot, frames)
	if first == nil || second == nil {
		t.Fatalf("BuildCommandStream returned nil")
	}
	if first.CommandStreamHash != second.CommandStreamHash {
		t.Fatalf("hash changed: %s != %s", first.CommandStreamHash, second.CommandStreamHash)
	}
	if first.CommandCount != 10 || len(first.Commands) != 10 {
		t.Fatalf("command count = %d len=%d, want 10", first.CommandCount, len(first.Commands))
	}
	for i, command := range first.Commands {
		if command.Order != i+1 || command.Source != source || command.SourceNodeID != "block:2" || command.Recipe != "morph.search_input" {
			t.Fatalf("command[%d] = %#v, want source-linked Block scene command", i, command)
		}
	}
	text := first.Commands[8]
	icon := first.Commands[9]
	if text.Command != "text" || text.MarkerOnly || text.RasterHash == "" || text.RasterCoverage <= 0 || text.RasterFormat != "builtin-5x7-alpha-mask-v1" {
		t.Fatalf("text command = %#v, want non-marker glyph raster evidence", text)
	}
	if icon.Command != "icon" || icon.MarkerOnly || icon.RasterHash == "" || icon.RasterCoverage <= 0 || icon.RasterFormat != "builtin-icon-mask-raster-v1" {
		t.Fatalf("icon command = %#v, want non-marker icon raster evidence", icon)
	}
}

func TestRenderCommandStreamRGBAProducesRendererOwnedFrameChecksum(t *testing.T) {
	source := "examples/surface_morph_rendered_studio_shell.tetra"
	snapshot := &surface.BlockSceneSnapshotReport{
		BlockSceneHash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Nodes: []surface.BlockSceneNodeReport{
			{
				BlockID:  1,
				ParentID: -1,
				Recipe:   "morph.surface",
				Layout:   &surface.BlockSceneLayoutSpecReport{Mode: "column", X: 0, Y: 0, W: 64, H: 48},
				Paint: &surface.BlockScenePaintSpecReport{Layers: []surface.BlockScenePaintLayerSpecReport{
					{Kind: "fill", Color: "#0e1216ff", Opacity: 255},
				}},
			},
			{
				BlockID:  2,
				ParentID: 1,
				Recipe:   "morph.region.panel",
				Layout:   &surface.BlockSceneLayoutSpecReport{Mode: "stack", X: 8, Y: 8, W: 24, H: 16},
				Paint: &surface.BlockScenePaintSpecReport{Layers: []surface.BlockScenePaintLayerSpecReport{
					{Kind: "fill", Color: "#202a32ff", Radius: 4, Opacity: 255},
					{Kind: "border", Color: "#e8eef4ff", Width: 1, Radius: 4, Opacity: 255},
				}},
				Text: &surface.BlockSceneTextSpecReport{TextLen: 5, Color: "#e8eef4ff", Size: 14, Weight: 500},
			},
		},
	}
	stream := BuildCommandStream(source, "software-rgba-headless", snapshot, nil)
	if stream == nil {
		t.Fatalf("BuildCommandStream returned nil")
	}

	frame, err := RenderCommandStreamRGBA(stream, 64, 48)
	if err != nil {
		t.Fatalf("RenderCommandStreamRGBA failed: %v", err)
	}
	if frame.Width != 64 || frame.Height != 48 || frame.Stride != 256 || len(frame.Pixels) != 64*48*4 {
		t.Fatalf("frame dimensions = %dx%d stride=%d bytes=%d, want 64x48 stride=256 bytes=%d", frame.Width, frame.Height, frame.Stride, len(frame.Pixels), 64*48*4)
	}
	if frame.Checksum == "" || frame.Checksum[:7] != "sha256:" {
		t.Fatalf("frame checksum = %q, want sha256 evidence", frame.Checksum)
	}

	second, err := RenderCommandStreamRGBA(stream, 64, 48)
	if err != nil {
		t.Fatalf("second RenderCommandStreamRGBA failed: %v", err)
	}
	if frame.Checksum != second.Checksum {
		t.Fatalf("renderer-owned frame checksum changed: %s != %s", frame.Checksum, second.Checksum)
	}
}
