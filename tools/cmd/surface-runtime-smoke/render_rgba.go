package main

import (
	"crypto/sha256"
	"encoding/hex"
)

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
	rectRGBA(frame, rect{X: block.X, Y: block.Y + block.H/2, W: block.W, H: block.H - block.H/2}, fillBottom)
	rectOutlineRGBA(frame, block, border)
	rectOutlineRGBA(frame, rect{X: block.X - 2, Y: block.Y - 2, W: block.W + 4, H: block.H + 4}, outline)
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
		rectOutlineRGBA(frame, rect{X: 20, Y: 36, W: 176, H: 64}, rgbaColor{R: 246, G: 205, B: 92, A: 255})
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
		rectRGBA(frame, rect{X: 116 + translateX, Y: 68, W: 34, H: 8}, rgbaColor{R: 255, G: 255, B: 255, A: 180})
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
	rectRGBA(frame, rect{X: 80, Y: 42, W: imageW - 16, H: 8}, rgbaColor{R: 220, G: 238, B: 255, A: 255})
	rectOutlineRGBA(frame, rect{X: 72, Y: 32, W: imageW, H: imageH}, fg)
	rectRGBA(frame, rect{X: 24, Y: 112, W: 96, H: 32}, fallbackFill)
	rectOutlineRGBA(frame, rect{X: 24, Y: 112, W: 96, H: 32}, fg)
	if active {
		rectRGBA(frame, rect{X: 36, Y: 124, W: 72, H: 6}, rgbaColor{R: 18, G: 22, B: 26, A: 255})
		rectOutlineRGBA(frame, rect{X: 68, Y: 28, W: 104, H: 72}, rgbaColor{R: 244, G: 205, B: 92, A: 255})
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
	rectOutlineRGBA(frame, rect{X: 236, Y: 72, W: 72, H: 80}, rgbaColor{R: 238, G: 242, B: 247, A: 255})
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
		rectOutlineRGBA(frame, rect{X: submitRect.X - 3, Y: submitRect.Y - 3, W: submitRect.W + 6, H: submitRect.H + 6}, focus)
		rectOutlineRGBA(frame, rect{X: layoutRect.X - 4, Y: layoutRect.Y - 4, W: layoutRect.W + 8, H: layoutRect.H + 8}, focus)
		rectRGBA(frame, rect{X: submitRect.X + 16, Y: submitRect.Y + 18, W: 64, H: 6}, bg)
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
	rectRGBA(frame, rect{X: grid.X + cellW + 6, Y: grid.Y, W: cellW, H: cellH}, rgbaColor{R: 90, G: 184, B: 150, A: 255})
	rectRGBA(frame, rect{X: grid.X, Y: grid.Y + cellH + 6, W: cellW, H: cellH}, rgbaColor{R: 52, G: 138, B: 118, A: 255})

	rectRGBA(frame, dock, rgbaColor{R: 30, G: 38, B: 46, A: 255})
	rectOutlineRGBA(frame, dock, fg)
	rectRGBA(frame, rect{X: dock.X, Y: dock.Y, W: dock.W, H: 24}, dockFill)
	rectRGBA(frame, rect{X: dock.X + 8, Y: dock.Y + 34, W: dock.W - 16, H: 8}, fg)

	rectRGBA(frame, scroll, rgbaColor{R: 28, G: 32, B: 42, A: 255})
	rectOutlineRGBA(frame, scroll, fg)
	rectRGBA(frame, rect{X: scroll.X + 8, Y: scroll.Y + 12, W: 42, H: 8}, scrollFill)
	rectRGBA(frame, rect{X: scroll.X + 8, Y: scroll.Y + 30, W: 50, H: 8}, scrollFill)
	rectRGBA(frame, rect{X: scroll.X + scroll.W - 12, Y: scroll.Y + 16 + 8, W: 5, H: 24}, overlayFill)

	rectRGBA(frame, overlay, overlayFill)
	rectOutlineRGBA(frame, overlay, fg)
	if active {
		rectRGBA(frame, rect{X: column.X + 20, Y: height - 44, W: 96, H: 20}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderWindowCounterFrameRGBA(count int, keyCount int, width int, height int, focused bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 22, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	accent := rgbaColor{R: 32, G: 132, B: 214, A: 255}
	keyAccent := rgbaColor{R: 34, G: 160, B: 104, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	rectRGBA(frame, rect{X: 32, Y: 52, W: 48, H: 7}, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderBrowserCounterFrameRGBA(count int, keyCount int, width int, height int, focused bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 24, G: 22, B: 34, A: 255}
	fg := rgbaColor{R: 242, G: 244, B: 248, A: 255}
	accent := rgbaColor{R: 54, G: 130, B: 218, A: 255}
	keyAccent := rgbaColor{R: 42, G: 170, B: 112, A: 255}
	textAccent := rgbaColor{R: 218, G: 184, B: 58, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	rectRGBA(frame, rect{X: 32, Y: 52, W: 48, H: 7}, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	rectRGBA(frame, rect{X: 32, Y: 68, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 88, Y: 68, W: 18, H: 7}, textAccent)
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderReleaseCounterFrameRGBA(count int, keyCount int, resetCount int, statusCode int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 24, B: 28, A: 255}
	fg := rgbaColor{R: 236, G: 242, B: 240, A: 255}
	accent := rgbaColor{R: 60, G: 142, B: 212, A: 255}
	resetAccent := rgbaColor{R: 210, G: 96, B: 78, A: 255}
	statusAccent := rgbaColor{R: 88, G: 174, B: 128, A: 255}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 32, Y: 56, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 96, Y: 58, W: 24 + count*8, H: 8}, statusAccent)
	rectRGBA(frame, rect{X: 32, Y: 76, W: 48, H: 7}, fg)
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
	rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 12, W: 24 + statusCode*12, H: 8}, statusAccent)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderTextFocusInputFrameRGBA(textLen int, caret int, focusIndex int, width int, height int) rgbaFrame {
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
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 32, Y: 44, W: 48, H: 7}, fg)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	caretX := textbox.X + 12 + caret*12
	rectRGBA(frame, rect{X: caretX, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, button, buttonAccent)
	rectOutlineRGBA(frame, button, fg)
	if focusIndex == 0 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusIndex == 1 {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderComponentTreeFrameRGBA(textLen int, caret int, focusID int, submitted int, reset int, width int, height int) rgbaFrame {
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
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	if focusID == 3 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusID == 5 {
		rectOutlineRGBA(frame, rect{X: submitButton.X - 4, Y: submitButton.Y - 4, W: submitButton.W + 8, H: submitButton.H + 8}, fg)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderMinimalToolkitFrameRGBA(textLen int, caret int, focusID int, submitted int, reset int, statusCode int, width int, height int) rgbaFrame {
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
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 4 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: submitButton.X - 4, Y: submitButton.Y - 4, W: submitButton.W + 8, H: submitButton.H + 8}, fg)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderToolkitReuseFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, width int, height int) rgbaFrame {
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
		rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	}
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	}
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 4 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 8 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderReleaseToolkitFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, checked bool, scrollOffset int, width int, height int) rgbaFrame {
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
		rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	}
	rectRGBA(frame, rect{X: emailLabel.X + 8, Y: emailLabel.Y + 8, W: 52, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	}
	rectRGBA(frame, checkbox, textBg)
	rectOutlineRGBA(frame, checkbox, fg)
	rectOutlineRGBA(frame, rect{X: checkbox.X + 12, Y: checkbox.Y + 8, W: 16, H: 16}, fg)
	if checked {
		rectRGBA(frame, rect{X: checkbox.X + 16, Y: checkbox.Y + 12, W: 8, H: 8}, checkboxAccent)
	}
	rectRGBA(frame, scroll, textBg)
	rectOutlineRGBA(frame, scroll, fg)
	rectRGBA(frame, rect{X: scroll.X + 12, Y: scroll.Y + 12 - scrollOffset/2, W: scroll.W - 40, H: 8}, scrollAccent)
	rectRGBA(frame, rect{X: scroll.X + scroll.W - 18, Y: scroll.Y + 6 + scrollOffset/2, W: 6, H: 20}, checkboxAccent)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, spacer, panelBg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 16, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 10 {
		rectOutlineRGBA(frame, rect{X: checkbox.X - 4, Y: checkbox.Y - 4, W: checkbox.W + 8, H: checkbox.H + 8}, fg)
	}
	if focusID == 14 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 15 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	if saved > 0 {
		rectRGBA(frame, rect{X: title.X + 140, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	}
	if reset > 0 {
		rectRGBA(frame, rect{X: title.X + 184, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}
func renderAccessibilityMetadataFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, width int, height int) rgbaFrame {
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
	rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	if focusID == 5 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 10 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
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
func checksumRGBA(pixels []byte) string {
	sum := sha256.Sum256(pixels)
	return hex.EncodeToString(sum[:])
}
func checksumText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
