//go:build !linux

package main

import (
	"fmt"
	"time"
)

func presentRealWindowSurface(
	title string,
	frame rgbaFrame,
	dwell time.Duration,
	holdUntilClose bool,
) error {
	return fmt.Errorf("linux-x64 real-window Surface evidence requires a Linux Wayland host")
}
