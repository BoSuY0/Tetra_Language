package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestNativeUISidecarSmokeBuildOnlyAcrossNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_native.tetra")
	src := `state ShellState:
    var toggles: Int = 0
    val label: String = "Wave 9 Native"

view ShellView(state: ShellState):
    bind toggles: Int = state.toggles
    bind labelText: String = state.label
    event submit -> toggle
    command toggle:
        state.toggles = state.toggles + 1
    style width: Int = 80
    accessibility description: String = "Native shell preview"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []struct {
		triple string
		out    string
	}{
		{triple: "linux-x64", out: "ui-linux"},
		{triple: "macos-x64", out: "ui-macos"},
		{triple: "windows-x64", out: "ui-windows.exe"},
	} {
		t.Run(target.triple, func(t *testing.T) {
			outPath := filepath.Join(tmp, target.out)
			if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target.triple, compiler.BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("build-only %s native ui example: %v", target.triple, err)
			}
			sidecarPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".ui.shell.txt"
			sidecar, err := os.ReadFile(sidecarPath)
			if err != nil {
				t.Fatalf("read %s native shell sidecar: %v", target.triple, err)
			}
			for _, want := range []string{
				"schema: tetra.ui.v0.4.0",
				"runtime: native shell command dispatch",
				"view ShellView (state: ShellState)",
				"dispatch submit -> toggle",
				"state.toggles = 1",
				"accessibility description: str = \"Native shell preview\"",
			} {
				if !strings.Contains(string(sidecar), want) {
					t.Fatalf("%s native shell sidecar missing %q:\n%s", target.triple, want, sidecar)
				}
			}
		})
	}
}

func TestWASISidecarPolicyAllowsUIJSONButNoRuntimeUIArtifacts(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_wasi.tetra")
	src := `state PanelState:
    var clicks: Int = 0
    val title: String = "WASI metadata"

view PanelView(state: PanelState):
    bind clickCount: Int = state.clicks
    bind panelTitle: String = state.title
    event click -> increment
    command increment:
        state.clicks = state.clicks + 1
    style width: Int = 240
    accessibility label: String = "WASI metadata panel"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "ui-wasi.wasm")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "wasm32-wasi", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-wasi ui source: %v", err)
	}

	base := strings.TrimSuffix(outPath, ".wasm")
	uiJSON, err := os.ReadFile(base + ".ui.json")
	if err != nil {
		t.Fatalf("read wasi ui json sidecar: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.ui.v0.4.0"`,
		`"name": "PanelView"`,
		`"state_type": "PanelState"`,
	} {
		if !strings.Contains(string(uiJSON), want) {
			t.Fatalf("wasi ui json missing %q:\n%s", want, uiJSON)
		}
	}

	for _, sidecar := range []string{
		base + ".ui.web.mjs",
		base + ".ui.html",
		base + ".ui.shell.txt",
	} {
		if _, err := os.Stat(sidecar); err == nil {
			t.Fatalf("wasi ui source must not emit runtime UI artifact %s", sidecar)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", sidecar, err)
		}
	}
}
