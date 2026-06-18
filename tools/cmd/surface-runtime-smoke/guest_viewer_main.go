//go:build linux && guestviewer

package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type rgbaFrame struct {
	Width  int
	Height int
	Stride int
	Pixels []byte
}

func main() {
	title := flag.String("title", "Tetra Surface Guest Dashboard", "Wayland window title")
	framePath := flag.String("frame", "", "raw RGBA frame path")
	width := flag.Int("width", 1760, "frame width")
	height := flag.Int("height", 700, "frame height")
	stride := flag.Int("stride", 7040, "frame stride")
	flag.Parse()

	if *framePath == "" {
		fmt.Fprintln(os.Stderr, "error: --frame is required")
		os.Exit(2)
	}
	pixels, err := os.ReadFile(*framePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read frame: %v\n", err)
		os.Exit(1)
	}
	if len(pixels) != *stride**height {
		fmt.Fprintf(os.Stderr, "frame bytes = %d, want %d\n", len(pixels), *stride**height)
		os.Exit(1)
	}
	frame := rgbaFrame{Width: *width, Height: *height, Stride: *stride, Pixels: pixels}
	if err := presentRealWindowSurface(*title, frame, 0*time.Millisecond, true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
