package surfacerender

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"tetra_language/tools/validators/surface"
)

type RGBAFrame struct {
	Width    int
	Height   int
	Stride   int
	Pixels   []byte
	Checksum string
}

type rgbaColor struct {
	R byte
	G byte
	B byte
	A byte
}

func RenderCommandStreamRGBA(stream *surface.RenderCommandStreamReport, width int, height int) (RGBAFrame, error) {
	if stream == nil {
		return RGBAFrame{}, fmt.Errorf("render command stream is required")
	}
	if width <= 0 || height <= 0 {
		return RGBAFrame{}, fmt.Errorf("renderer frame dimensions must be positive")
	}
	frame := RGBAFrame{
		Width:  width,
		Height: height,
		Stride: width * 4,
		Pixels: make([]byte, width*height*4),
	}
	for _, command := range stream.Commands {
		if err := renderCommandRGBA(&frame, command); err != nil {
			return RGBAFrame{}, err
		}
	}
	frame.Checksum = ChecksumRGBA(frame.Pixels)
	return frame, nil
}

func ChecksumRGBA(pixels []byte) string {
	sum := sha256.Sum256(pixels)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func renderCommandRGBA(frame *RGBAFrame, command surface.RenderCommandReport) error {
	kind := normalizeRenderCommandKind(command.Command)
	switch kind {
	case "radius_clip":
		return nil
	case "border", "outline":
		color, err := parseRGBAColor(command.Color, command.Opacity)
		if err != nil {
			return fmt.Errorf("render command %d %s: %w", command.Order, kind, err)
		}
		width := command.Width
		if width <= 0 {
			width = 1
		}
		drawOutline(frame, command.Rect, width, color)
	case "text":
		color, err := parseRGBAColor(command.Color, command.Opacity)
		if err != nil {
			return fmt.Errorf("render command %d text: %w", command.Order, err)
		}
		drawTextMask(frame, command.Rect, command.TextLen, color)
	case "icon":
		color, err := parseRGBAColor(command.Color, command.Opacity)
		if err != nil {
			return fmt.Errorf("render command %d icon: %w", command.Order, err)
		}
		drawIconMask(frame, command.Rect, color)
	case "shadow":
		color, err := parseRGBAColor(command.Color, command.Opacity)
		if err != nil {
			return fmt.Errorf("render command %d shadow: %w", command.Order, err)
		}
		rect := command.Rect
		rect.X += command.OffsetX
		rect.Y += command.OffsetY
		if command.Blur > 0 {
			rect.X -= command.Blur / 2
			rect.Y -= command.Blur / 2
			rect.W += command.Blur
			rect.H += command.Blur
		}
		drawRect(frame, rect, color)
	default:
		color, err := parseRGBAColor(command.Color, command.Opacity)
		if err != nil {
			return fmt.Errorf("render command %d %s: %w", command.Order, kind, err)
		}
		drawRect(frame, command.Rect, color)
	}
	return nil
}

func parseRGBAColor(value string, opacity int) (rgbaColor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return rgbaColor{}, fmt.Errorf("color is required")
	}
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 && len(value) != 8 {
		return rgbaColor{}, fmt.Errorf("color %q must be #rrggbb or #rrggbbaa", value)
	}
	parse := func(part string) (byte, error) {
		parsed, err := strconv.ParseUint(part, 16, 8)
		return byte(parsed), err
	}
	r, err := parse(value[0:2])
	if err != nil {
		return rgbaColor{}, err
	}
	g, err := parse(value[2:4])
	if err != nil {
		return rgbaColor{}, err
	}
	b, err := parse(value[4:6])
	if err != nil {
		return rgbaColor{}, err
	}
	a := byte(255)
	if len(value) == 8 {
		a, err = parse(value[6:8])
		if err != nil {
			return rgbaColor{}, err
		}
	}
	if opacity >= 0 && opacity < 255 {
		a = byte(int(a) * opacity / 255)
	}
	return rgbaColor{R: r, G: g, B: b, A: a}, nil
}

func drawRect(frame *RGBAFrame, r surface.RectReport, color rgbaColor) {
	x0 := clampInt(r.X, 0, frame.Width)
	y0 := clampInt(r.Y, 0, frame.Height)
	x1 := clampInt(r.X+r.W, 0, frame.Width)
	y1 := clampInt(r.Y+r.H, 0, frame.Height)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			blendPixel(frame, x, y, color)
		}
	}
}

func drawOutline(frame *RGBAFrame, r surface.RectReport, width int, color rgbaColor) {
	if width <= 0 {
		width = 1
	}
	drawRect(frame, surface.RectReport{X: r.X, Y: r.Y, W: r.W, H: width}, color)
	drawRect(frame, surface.RectReport{X: r.X, Y: r.Y + r.H - width, W: r.W, H: width}, color)
	drawRect(frame, surface.RectReport{X: r.X, Y: r.Y, W: width, H: r.H}, color)
	drawRect(frame, surface.RectReport{X: r.X + r.W - width, Y: r.Y, W: width, H: r.H}, color)
}

func drawTextMask(frame *RGBAFrame, r surface.RectReport, textLen int, color rgbaColor) {
	if textLen <= 0 {
		textLen = 1
	}
	width := textLen * 5
	if r.W > 0 && width > r.W {
		width = r.W
	}
	height := 7
	if r.H > 0 && height > r.H {
		height = r.H
	}
	x0 := r.X
	y0 := r.Y
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)%2 == 0 || y == 0 || y == height-1 {
				blendPixel(frame, x0+x, y0+y, color)
			}
		}
	}
}

func drawIconMask(frame *RGBAFrame, r surface.RectReport, color rgbaColor) {
	size := r.W
	if r.H > 0 && r.H < size {
		size = r.H
	}
	if size <= 0 {
		size = 12
	}
	for i := 0; i < size; i++ {
		blendPixel(frame, r.X+i, r.Y+i, color)
		blendPixel(frame, r.X+size-1-i, r.Y+i, color)
	}
	center := size / 2
	for i := 0; i < size; i++ {
		blendPixel(frame, r.X+center, r.Y+i, color)
		blendPixel(frame, r.X+i, r.Y+center, color)
	}
}

func blendPixel(frame *RGBAFrame, x int, y int, color rgbaColor) {
	if x < 0 || y < 0 || x >= frame.Width || y >= frame.Height || color.A == 0 {
		return
	}
	i := y*frame.Stride + x*4
	if color.A == 255 {
		frame.Pixels[i] = color.R
		frame.Pixels[i+1] = color.G
		frame.Pixels[i+2] = color.B
		frame.Pixels[i+3] = 255
		return
	}
	alpha := int(color.A)
	inv := 255 - alpha
	frame.Pixels[i] = byte((int(color.R)*alpha + int(frame.Pixels[i])*inv) / 255)
	frame.Pixels[i+1] = byte((int(color.G)*alpha + int(frame.Pixels[i+1])*inv) / 255)
	frame.Pixels[i+2] = byte((int(color.B)*alpha + int(frame.Pixels[i+2])*inv) / 255)
	frame.Pixels[i+3] = 255
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
