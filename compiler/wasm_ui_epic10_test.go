package compiler

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWASIDogfoodTargetBuildOnlyAndNoUIRuntimeArtifacts(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "projects", "dogfood_wasi", "src", "main.tetra")
	outPath := filepath.Join(tmp, "dogfood-wasi.wasm")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "wasm32-wasi", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-wasi dogfood: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read wasi wasm: %v", err)
	}
	if len(raw) < 8 || !bytes.Equal(raw[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		t.Fatalf("invalid wasm header for wasi dogfood")
	}
	if !bytes.Contains(raw, []byte("_start")) {
		t.Fatalf("wasi output missing _start export marker")
	}
	if bytes.Contains(raw, []byte("tetra_main")) {
		t.Fatalf("wasi output unexpectedly contains tetra_main export marker")
	}

	base := strings.TrimSuffix(outPath, ".wasm")
	for _, sidecar := range []string{
		base + ".ui.json",
		base + ".ui.web.mjs",
		base + ".ui.html",
		base + ".ui.shell.txt",
	} {
		if _, err := os.Stat(sidecar); err == nil {
			t.Fatalf("wasi dogfood should not emit UI runtime sidecar %s", sidecar)
		}
	}

	capsuleRaw, err := os.ReadFile(filepath.Join("..", "examples", "projects", "dogfood_wasi", "Tetra.capsule"))
	if err != nil {
		t.Fatalf("read dogfood_wasi capsule: %v", err)
	}
	if !strings.Contains(string(capsuleRaw), `target "wasm32-wasi"`) {
		t.Fatalf("dogfood_wasi capsule missing wasm32-wasi target:\n%s", capsuleRaw)
	}
}

func TestWebUIDogfoodBuildWritesSchemaCheckedArtifacts(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "projects", "dogfood_web_ui", "src", "main.tetra")
	outPath := filepath.Join(tmp, "dogfood-web-ui.wasm")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-web dogfood: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read web wasm: %v", err)
	}
	if !bytes.Contains(raw, []byte("tetra_main")) || bytes.Contains(raw, []byte("_start")) {
		t.Fatalf("unexpected web exports in dogfood wasm")
	}

	base := strings.TrimSuffix(outPath, ".wasm")
	uiJSON, err := os.ReadFile(base + ".ui.json")
	if err != nil {
		t.Fatalf("read web ui bundle: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.ui.v1"`,
		`"name": "TodoView"`,
		`"state_type": "TodoState"`,
	} {
		if !strings.Contains(string(uiJSON), want) {
			t.Fatalf("web ui bundle missing %q:\n%s", want, uiJSON)
		}
	}
	uiModule, err := os.ReadFile(base + ".ui.web.mjs")
	if err != nil {
		t.Fatalf("read web ui module: %v", err)
	}
	if !strings.Contains(string(uiModule), "tetra_ui: unsupported schema") {
		t.Fatalf("web ui module missing schema guard:\n%s", uiModule)
	}
	uiHTML, err := os.ReadFile(base + ".ui.html")
	if err != nil {
		t.Fatalf("read web ui html: %v", err)
	}
	for _, want := range []string{"mountTetraUI", "runTetra"} {
		if !strings.Contains(string(uiHTML), want) {
			t.Fatalf("web ui html missing %q:\n%s", want, uiHTML)
		}
	}

	capsuleRaw, err := os.ReadFile(filepath.Join("..", "examples", "projects", "dogfood_web_ui", "Tetra.capsule"))
	if err != nil {
		t.Fatalf("read dogfood_web_ui capsule: %v", err)
	}
	if !strings.Contains(string(capsuleRaw), `target "wasm32-web"`) {
		t.Fatalf("dogfood_web_ui capsule missing wasm32-web target:\n%s", capsuleRaw)
	}
}

func TestWASMUIExamplesBuildWithDeterministicMetadataSidecars(t *testing.T) {
	tmp := t.TempDir()
	cases := []struct {
		name          string
		srcPath       string
		viewName      string
		accessibility []string
	}{
		{
			name:     "ui_web_smoke",
			srcPath:  filepath.Join("..", "examples", "ui_web_smoke.tetra"),
			viewName: "CounterView",
			accessibility: []string{
				`"name": "role"`,
				`"name": "label"`,
				`Increment counter`,
			},
		},
		{
			name:     "ui_native_shell_smoke",
			srcPath:  filepath.Join("..", "examples", "ui_native_shell_smoke.tetra"),
			viewName: "ShellView",
			accessibility: []string{
				`"name": "role"`,
				`"name": "description"`,
				`Native shell preview`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(tmp, tc.name+".wasm")
			if _, err := BuildFileWithStatsOpt(tc.srcPath, outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("build wasm32-web %s: %v", tc.srcPath, err)
			}
			base := strings.TrimSuffix(outPath, ".wasm")
			uiJSON, err := os.ReadFile(base + ".ui.json")
			if err != nil {
				t.Fatalf("read ui bundle: %v", err)
			}
			if !strings.Contains(string(uiJSON), `"schema": "tetra.ui.v1"`) || !strings.Contains(string(uiJSON), tc.viewName) {
				t.Fatalf("unexpected ui bundle for %s:\n%s", tc.name, uiJSON)
			}
			for _, want := range tc.accessibility {
				if !strings.Contains(string(uiJSON), want) {
					t.Fatalf("ui bundle for %s missing accessibility marker %q:\n%s", tc.name, want, uiJSON)
				}
			}
		})
	}
}

func TestWASMUISidecarsAreDeterministicAcrossBuilds(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "ui_web_smoke.tetra")
	firstBase := buildWASMUIFixture(t, srcPath, filepath.Join(tmp, "first", "app.wasm"))
	secondBase := buildWASMUIFixture(t, srcPath, filepath.Join(tmp, "second", "app.wasm"))
	for _, ext := range []string{".ui.json", ".ui.web.mjs", ".ui.html"} {
		first, err := os.ReadFile(firstBase + ext)
		if err != nil {
			t.Fatalf("read first%s: %v", ext, err)
		}
		second, err := os.ReadFile(secondBase + ext)
		if err != nil {
			t.Fatalf("read second%s: %v", ext, err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("sidecar %s is not deterministic across builds\nfirst:\n%s\nsecond:\n%s", ext, first, second)
		}
	}
}

func buildWASMUIFixture(t *testing.T, srcPath, outPath string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("create wasm ui output dir: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-web %s: %v", srcPath, err)
	}
	return strings.TrimSuffix(outPath, ".wasm")
}

func TestNativeShellUIExampleWritesMetadataPreviewSidecar(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "ui_native_shell_smoke.tetra")
	outPath := filepath.Join(tmp, "ui-native")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux-x64 native ui example: %v", err)
	}
	sidecar, err := os.ReadFile(outPath + ".ui.shell.txt")
	if err != nil {
		t.Fatalf("read native shell sidecar: %v", err)
	}
	for _, want := range []string{
		"schema: tetra.ui.v1",
		"runtime: metadata-only preview (no event dispatch)",
		"view ShellView (state: ShellState)",
	} {
		if !strings.Contains(string(sidecar), want) {
			t.Fatalf("native shell sidecar missing %q:\n%s", want, sidecar)
		}
	}
}
