//go:build windows

package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWindowsPlatformProbeCompletesUnderSchedulerPressure(t *testing.T) {
	if os.Getenv("TETRA_WINDOWS_PROBE_HELPER") == "1" {
		runWindowsPlatformProbeSchedulerPressureHelper(t)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=^TestWindowsPlatformProbeCompletesUnderSchedulerPressure$",
		"-test.v",
	)
	cmd.Env = append(os.Environ(), "TETRA_WINDOWS_PROBE_HELPER=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("Windows platform probe timed out after 15s under scheduler pressure\n%s", out)
	}
	if err != nil {
		t.Fatalf("Windows platform probe helper failed: %v\n%s", err, out)
	}
}

func runWindowsPlatformProbeSchedulerPressureHelper(t *testing.T) {
	oldProcs := runtime.GOMAXPROCS(0)
	if oldProcs < 4 {
		runtime.GOMAXPROCS(4)
		t.Cleanup(func() { runtime.GOMAXPROCS(oldProcs) })
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0) * 4
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					runtime.Gosched()
				}
			}
		}()
	}
	t.Cleanup(func() {
		close(stop)
		wg.Wait()
	})

	result, err := runPlatformWindowProbe("windows-x64")
	if err != nil {
		t.Fatalf("runPlatformWindowProbe: %v", err)
	}
	if result.API != "win32-user32" {
		t.Fatalf("API = %q, want win32-user32", result.API)
	}
	for _, want := range []string{
		"platform-window-create",
		"platform-event-dispatch",
		"platform-timer",
		"platform-redraw",
		"platform-window-close",
	} {
		if !hasMarkerWithPrefix(result.Markers, want) {
			t.Fatalf("missing marker %q in %q", want, strings.Join(result.Markers, ";"))
		}
	}
}

func hasMarkerWithPrefix(markers []string, prefix string) bool {
	for _, marker := range markers {
		if strings.HasPrefix(marker, prefix) {
			return true
		}
	}
	return false
}
