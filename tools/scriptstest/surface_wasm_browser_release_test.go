package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserSmokeTimeoutCleansChildren(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "surface", "surface-wasm32-web-release-browser-smoke.sh"))
	if err != nil {
		t.Fatalf("read wasm32-web release browser smoke: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"surface_wasm32_web_browser_cleanup()",
		"trap surface_wasm32_web_browser_cleanup EXIT",
		"SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS",
		"timeout --kill-after",
		"Surface wasm32-web browser-canvas release browser blocked report:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("wasm32-web release browser smoke missing timeout/cleanup contract %q", want)
		}
	}
}
