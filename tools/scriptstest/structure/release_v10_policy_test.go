package structure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10SmokeScriptsHaveDefaultReportPaths(t *testing.T) {
	for _, script := range []struct {
		Path string
		Want string
	}{
		{Path: "release/v1_0/wasi-smoke.sh", Want: `docs/generated/v1_0/wasi-smoke.json`},
		{Path: "release/v1_0/web-smoke.sh", Want: `docs/generated/v1_0/web-ui-smoke.json`},
	} {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", script.Path))
		if err != nil {
			t.Fatalf("read %s: %v", script.Path, err)
		}
		text := string(raw)
		if strings.Contains(text, "--report is required") {
			t.Fatalf("%s still requires --report", script.Path)
		}
		if !strings.Contains(text, script.Want) {
			t.Fatalf("%s missing default report path %q", script.Path, script.Want)
		}
	}
}

func TestRoadmapV10RecordsExplicitCompatibilityAndSafetyPolicy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "roadmap_0_6_to_1_0.md"))
	if err != nil {
		t.Fatalf("read v1.0 roadmap: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Flow syntax is the only official 1.0 syntax",
		"`wasm32-wasi`",
		"`wasm32-web`",
		"no data races",
		"Network EcoNet/TetraHub",
		"explicitly labeled beta surface",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v1.0 roadmap missing %q", want)
		}
	}
}
