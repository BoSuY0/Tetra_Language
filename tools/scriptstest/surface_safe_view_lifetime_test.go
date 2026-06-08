package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceSafeViewLifetimeGateIsBoundedAndFocused(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "safe-view-lifetime", "gate.sh"))
	if err != nil {
		t.Fatalf("read safe-view lifetime gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"safe_view_lifetime_cleanup()",
		"trap safe_view_lifetime_cleanup EXIT",
		"safe_view_lifetime_run_step()",
		"SAFE_VIEW_LIFETIME_TIMEOUT_SECONDS",
		"timeout --kill-after=5s",
		"safe-view-step-",
		"SurfaceClose|FrameAfterClose|DoubleClose|BeginPresent|ResizeAfterClose|ResourceCleanup|BrowserProcessCleanup|SafeViewLifetime",
		"surface-close-frame-event-resize-resource-cleanup",
		"safe-view-lifetime-summary.json",
		`"bounded": true`,
		`"release_blocking": true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("safe-view lifetime gate missing bounded/focused contract %q", want)
		}
	}
	for _, forbidden := range []string{
		"run_go_test ./compiler/...",
		"run_go_test ./cli/...",
		"run_go_test ./tools/...",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("safe-view lifetime gate still uses unbounded package sweep %q", forbidden)
		}
	}
}

func TestSurfaceSafeViewLifetimeTargetScriptsHaveCleanupContracts(t *testing.T) {
	root := repoRoot(t)
	scripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-release-browser-smoke.sh"),
			want: []string{
				"surface_wasm32_web_browser_cleanup()",
				"trap surface_wasm32_web_browser_cleanup EXIT",
				"SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS",
				"timeout --kill-after",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-window-smoke.sh"),
			want: []string{
				"surface_linux_x64_release_window_cleanup()",
				"trap surface_linux_x64_release_window_cleanup EXIT",
				"SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS",
				"timeout --kill-after=5s",
				"surface_linux_x64_release_window_active_pid",
			},
		},
	}
	for _, script := range scripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface target cleanup script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface target cleanup script %s missing %q", script.path, want)
			}
		}
	}
}
