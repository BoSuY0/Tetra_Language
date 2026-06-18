//go:build !windows && !darwin

package main

import "fmt"

func runPlatformWindowProbe(target string) (platformWindowProbeResult, error) {
	return platformWindowProbeResult{}, fmt.Errorf(
		"platform window probe for %s requires a Windows or macOS target host",
		target,
	)
}
