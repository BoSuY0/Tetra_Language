package wasm32_web

import (
	"strings"
	"testing"
)

func TestUIModuleIncludesSchemaGuardAndMetadataBoundary(t *testing.T) {
	src := string(UIModule("app.ui.json"))
	for _, want := range []string{
		"tetra_ui: unsupported schema",
		`bundle.schema !== "tetra.ui.v1"`,
		"runtime: metadata-only preview (no event dispatch)",
		`new URL("app.ui.json", import.meta.url)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("UI module missing %q:\n%s", want, src)
		}
	}
}

func TestUIHTMLPageMountsUIShellBeforeRunningWASM(t *testing.T) {
	html := string(UIHTMLPage("app.wasm", "app.mjs", "app.ui.web.mjs"))
	mountIdx := strings.Index(html, "await mountTetraUI(root);")
	runIdx := strings.Index(html, "await runTetra(")
	if mountIdx < 0 || runIdx < 0 {
		t.Fatalf("UI HTML missing mount/run hooks:\n%s", html)
	}
	if mountIdx > runIdx {
		t.Fatalf("UI HTML should mount UI metadata shell before running wasm:\n%s", html)
	}
}
