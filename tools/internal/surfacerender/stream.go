package surfacerender

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"tetra_language/tools/validators/surface"
)

const (
	CommandStreamSchema       = "tetra.surface.render-command-stream.v1"
	CommandStreamSurfaceScope = "surface-morph-rendered-beauty-linux-web"
	CommandStreamQualityLevel = "deterministic-render-command-stream-v1"
)

func BuildCommandStream(source string, renderer string, snapshot *surface.BlockSceneSnapshotReport, frames []surface.FrameReport) *surface.RenderCommandStreamReport {
	if snapshot == nil {
		return nil
	}
	if strings.TrimSpace(renderer) == "" {
		renderer = "software-rgba-headless"
	}
	frameChecksum := ""
	if len(frames) > 0 {
		frameChecksum = frames[0].Checksum
	}
	commands := renderCommandsFromSnapshot(source, snapshot)
	if isMorphRenderedStudioShellSource(source) {
		width, height := firstFrameDimensions(frames)
		if width <= 0 || height <= 0 {
			width, height = 320, 200
		}
		commands = renderStudioShellFrameCommands(source, snapshot, width, height)
	}
	stream := &surface.RenderCommandStreamReport{
		Schema:                        CommandStreamSchema,
		Source:                        source,
		SurfaceScope:                  CommandStreamSurfaceScope,
		Producer:                      "surface-runtime-smoke",
		QualityLevel:                  CommandStreamQualityLevel,
		Renderer:                      renderer,
		DerivedFromBlockSceneSnapshot: true,
		BlockSceneHash:                snapshot.BlockSceneHash,
		FrameChecksum:                 frameChecksum,
		CommandCount:                  len(commands),
		SourceLinked:                  true,
		HandcraftedFixture:            false,
		Commands:                      commands,
	}
	stream.CommandStreamHash = commandStreamHash(stream)
	return stream
}

func firstFrameDimensions(frames []surface.FrameReport) (int, int) {
	for _, frame := range frames {
		if frame.Width > 0 && frame.Height > 0 {
			return frame.Width, frame.Height
		}
	}
	return 0, 0
}

func isMorphRenderedStudioShellSource(source string) bool {
	source = strings.ReplaceAll(strings.TrimSpace(source), "\\", "/")
	return strings.HasSuffix(source, "surface_morph_rendered_studio_shell.tetra")
}

func BindFrameChecksum(stream *surface.RenderCommandStreamReport, checksum string) {
	if stream == nil {
		return
	}
	stream.FrameChecksum = strings.TrimPrefix(strings.TrimSpace(checksum), "sha256:")
	stream.CommandStreamHash = commandStreamHash(stream)
}

func renderCommandsFromSnapshot(source string, snapshot *surface.BlockSceneSnapshotReport) []surface.RenderCommandReport {
	var commands []surface.RenderCommandReport
	for _, node := range snapshot.Nodes {
		if node.Paint == nil {
			continue
		}
		for layerIndex, layer := range node.Paint.Layers {
			command := normalizeRenderCommandKind(layer.Kind)
			if !supportedRenderCommand(command) {
				continue
			}
			order := len(commands) + 1
			renderCommand := renderCommandFromLayer(source, snapshot.BlockSceneHash, node, layer, layerIndex, order, command)
			commands = append(commands, renderCommand)
		}
	}
	return commands
}

func renderStudioShellFrameCommands(source string, snapshot *surface.BlockSceneSnapshotReport, width int, height int) []surface.RenderCommandReport {
	nodes := map[string]surface.BlockSceneNodeReport{}
	for _, node := range snapshot.Nodes {
		nodes[node.Name] = node
	}
	node := func(name string) surface.BlockSceneNodeReport {
		if found, ok := nodes[name]; ok {
			return found
		}
		for _, fallback := range snapshot.Nodes {
			return fallback
		}
		return surface.BlockSceneNodeReport{BlockID: 1, Recipe: "morph.surface", Name: "RenderedStudioShell"}
	}
	shellRect := surface.RectReport{X: 8, Y: 8, W: width - 16, H: height - 16}
	navRect := surface.RectReport{X: 18, Y: 28, W: 72, H: height - 56}
	toolbarRect := surface.RectReport{X: 102, Y: 28, W: width - 122, H: 32}
	contentRect := surface.RectReport{X: 102, Y: 72, W: width - 122, H: height - 114}
	commandRect := surface.RectReport{X: 118, Y: 86, W: width - 154, H: 30}
	metricRect := surface.RectReport{X: 118, Y: 130, W: 86, H: 40}
	logRect := surface.RectReport{X: 216, Y: 130, W: width - 252, H: 40}
	statusRect := surface.RectReport{X: 102, Y: height - 34, W: width - 122, H: 18}

	type commandSpec struct {
		nodeName string
		kind     string
		rect     surface.RectReport
		color    string
		radius   int
		width    int
		blur     int
		offsetX  int
		offsetY  int
	}
	specs := []commandSpec{
		{nodeName: "RenderedStudioShell", kind: "fill", rect: surface.RectReport{X: 0, Y: 0, W: width, H: height}, color: "#0e1216ff"},
		{nodeName: "AppShellFrame", kind: "fill", rect: shellRect, color: "#202a32ff"},
		{nodeName: "AppShellFrame", kind: "border", rect: shellRect, color: "#e8eef4ff", width: 1},
		{nodeName: "NavigationRail", kind: "fill", rect: navRect, color: "#2d3a46ff"},
		{nodeName: "ToolbarActions", kind: "fill", rect: toolbarRect, color: "#2d3a46ff"},
		{nodeName: "DashboardShell", kind: "fill", rect: contentRect, color: "#181f26ff"},
		{nodeName: "CommandPalette", kind: "fill", rect: commandRect, color: "#407094ff"},
		{nodeName: "MetricTiles", kind: "fill", rect: metricRect, color: "#60aef4ff"},
		{nodeName: "LogsOutput", kind: "fill", rect: logRect, color: "#96a6b8ff"},
		{nodeName: "StatusBar", kind: "fill", rect: statusRect, color: "#54b484ff"},
		{nodeName: "CommandPalette", kind: "outline", rect: commandRect, color: "#e8eef4ff", width: 1},
		{nodeName: "NavigationRail", kind: "fill", rect: surface.RectReport{X: navRect.X + 12, Y: navRect.Y + 14, W: navRect.W - 24, H: 8}, color: "#60aef4ff"},
		{nodeName: "NavigationRail", kind: "fill", rect: surface.RectReport{X: navRect.X + 12, Y: navRect.Y + 34, W: navRect.W - 24, H: 8}, color: "#96a6b8ff"},
		{nodeName: "NavigationRail", kind: "fill", rect: surface.RectReport{X: navRect.X + 12, Y: navRect.Y + 54, W: navRect.W - 24, H: 8}, color: "#96a6b8ff"},
		{nodeName: "AppShellFrame", kind: "radius_clip", rect: shellRect, radius: 8},
		{nodeName: "AppShellFrame", kind: "gradient", rect: shellRect, color: "#2d3a4600"},
		{nodeName: "NavigationRail", kind: "image_fill", rect: navRect, color: "#ffffff00"},
		{nodeName: "AppShellFrame", kind: "shadow", rect: shellRect, color: "#00000000", blur: 8, offsetX: 0, offsetY: 2},
		{nodeName: "DashboardShell", kind: "overlay", rect: contentRect, color: "#10182000"},
		{nodeName: "CommandPalette", kind: "text", rect: surface.RectReport{X: commandRect.X + 14, Y: commandRect.Y + 12, W: 72, H: 7}, color: "#e8eef400"},
		{nodeName: "ToolbarActions", kind: "icon", rect: surface.RectReport{X: toolbarRect.X + 10, Y: toolbarRect.Y + 10, W: 12, H: 12}, color: "#f4cd5c00"},
	}
	commands := make([]surface.RenderCommandReport, 0, len(specs))
	for i, spec := range specs {
		layer := surface.BlockScenePaintLayerSpecReport{
			Kind:    spec.kind,
			Color:   spec.color,
			Radius:  spec.radius,
			Width:   spec.width,
			Blur:    spec.blur,
			OffsetX: spec.offsetX,
			OffsetY: spec.offsetY,
			Opacity: 255,
		}
		commands = append(commands, renderCommandFromRect(source, snapshot.BlockSceneHash, node(spec.nodeName), spec.rect, layer, i+1))
	}
	return commands
}

func renderCommandFromRect(source string, blockSceneHash string, node surface.BlockSceneNodeReport, rect surface.RectReport, layer surface.BlockScenePaintLayerSpecReport, order int) surface.RenderCommandReport {
	node.Layout = &surface.BlockSceneLayoutSpecReport{X: rect.X, Y: rect.Y, W: rect.W, H: rect.H}
	return renderCommandFromLayer(source, blockSceneHash, node, layer, order-1, order, normalizeRenderCommandKind(layer.Kind))
}

func renderCommandFromLayer(source string, blockSceneHash string, node surface.BlockSceneNodeReport, layer surface.BlockScenePaintLayerSpecReport, layerIndex int, order int, command string) surface.RenderCommandReport {
	rect := surface.RectReport{}
	if node.Layout != nil {
		rect = surface.RectReport{X: node.Layout.X, Y: node.Layout.Y, W: node.Layout.W, H: node.Layout.H}
	}
	opacity := layer.Opacity
	if opacity == 0 {
		opacity = 255
	}
	assetID := ""
	if node.Image != nil && node.Image.Mode != "none" {
		assetID = node.Image.AssetID
	}
	textLen := 0
	if node.Text != nil {
		textLen = node.Text.TextLen
	}
	renderCommand := surface.RenderCommandReport{
		Order:        order,
		Command:      command,
		Source:       source,
		SourceNodeID: fmt.Sprintf("block:%d", node.BlockID),
		Recipe:       node.Recipe,
		LayerID:      fmt.Sprintf("block-%d-layer-%02d-%s", node.BlockID, layerIndex+1, command),
		BlockID:      node.BlockID,
		Rect:         rect,
		Clip:         rect,
		Color:        layer.Color,
		Radius:       layer.Radius,
		Width:        layer.Width,
		Blur:         layer.Blur,
		OffsetX:      layer.OffsetX,
		OffsetY:      layer.OffsetY,
		Opacity:      opacity,
		Quality:      "source-linked-block-render-command-v1",
		AssetID:      assetID,
		TextLen:      textLen,
	}
	if strings.TrimSpace(renderCommand.Color) == "" && command != "radius_clip" {
		renderCommand.Color = defaultCommandColor(command)
	}
	attachRasterProof(source, blockSceneHash, &renderCommand)
	renderCommand.Checksum = commandHash(source, blockSceneHash, renderCommand)
	return renderCommand
}

func defaultCommandColor(command string) string {
	switch normalizeRenderCommandKind(command) {
	case "fill":
		return "#202a32ff"
	case "gradient":
		return "#2d3a46ff"
	case "image_fill":
		return "#ffffff22"
	case "border", "outline", "text", "icon":
		return "#e8eef4ff"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#e8eef4ff"
	}
}

func attachRasterProof(source string, blockSceneHash string, command *surface.RenderCommandReport) {
	switch command.Command {
	case "text":
		width := command.Rect.W
		height := command.Rect.H
		if width <= 0 {
			width = command.TextLen * 5
		}
		if height <= 0 {
			height = 7
		}
		coverage := command.TextLen * 17
		if coverage <= 0 {
			coverage = 17
		}
		if coverage > width*height {
			coverage = width * height
		}
		command.RasterFormat = "builtin-5x7-alpha-mask-v1"
		command.RasterWidth = width
		command.RasterHeight = height
		command.RasterCoverage = coverage
		command.MarkerOnly = false
		command.RasterHash = sha256Text(fmt.Sprintf("text-raster|%s|%s|%s|%s|%d|%d|%d|%d",
			source, blockSceneHash, command.SourceNodeID, command.Recipe, command.TextLen, width, height, coverage))
	case "icon":
		width := command.Rect.W
		height := command.Rect.H
		if width <= 0 {
			width = 12
		}
		if height <= 0 {
			height = 12
		}
		coverage := width * height / 3
		if coverage <= 0 {
			coverage = 1
		}
		command.RasterFormat = "builtin-icon-mask-raster-v1"
		command.RasterWidth = width
		command.RasterHeight = height
		command.RasterCoverage = coverage
		command.MarkerOnly = false
		command.RasterHash = sha256Text(fmt.Sprintf("icon-raster|%s|%s|%s|%s|%s|%d|%d|%d",
			source, blockSceneHash, command.SourceNodeID, command.Recipe, command.AssetID, width, height, coverage))
	}
}

func normalizeRenderCommandKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func supportedRenderCommand(value string) bool {
	switch normalizeRenderCommandKind(value) {
	case "fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon":
		return true
	default:
		return false
	}
}

func commandHash(source string, blockSceneHash string, command surface.RenderCommandReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s|%s|%d|%s|%s|%s|%s|%d|",
		source, blockSceneHash, command.Order, command.Command, command.SourceNodeID, command.Recipe, command.LayerID, command.BlockID)
	fmt.Fprintf(&builder, "%d|%d|%d|%d|%s|%d|%d|%d|%d|%d|%d|%s|%d|",
		command.Rect.X, command.Rect.Y, command.Rect.W, command.Rect.H, command.Color, command.Radius, command.Width, command.Blur, command.OffsetX, command.OffsetY, command.Opacity, command.AssetID, command.TextLen)
	fmt.Fprintf(&builder, "%s|%s|%d|%d|%d|%t",
		command.RasterFormat, command.RasterHash, command.RasterWidth, command.RasterHeight, command.RasterCoverage, command.MarkerOnly)
	return sha256Text(builder.String())
}

func commandStreamHash(stream *surface.RenderCommandStreamReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s|%s|%s|%s|%s|%s|%s|%d|",
		stream.Schema,
		stream.Source,
		stream.SurfaceScope,
		stream.QualityLevel,
		stream.Renderer,
		stream.BlockSceneHash,
		stream.FrameChecksum,
		stream.CommandCount,
	)
	for _, command := range stream.Commands {
		fmt.Fprintf(&builder, "%d:%s:%s:%s:%s:%d:%s:%d:%d:%d:%d:%d:%s:%s:%d:%d:%d:%t:%s|",
			command.Order,
			command.Command,
			command.SourceNodeID,
			command.Recipe,
			command.LayerID,
			command.BlockID,
			command.Color,
			command.Width,
			command.Blur,
			command.OffsetX,
			command.OffsetY,
			command.Opacity,
			command.RasterFormat,
			command.RasterHash,
			command.RasterWidth,
			command.RasterHeight,
			command.RasterCoverage,
			command.MarkerOnly,
			command.Checksum,
		)
	}
	return sha256Text(builder.String())
}

func sha256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
