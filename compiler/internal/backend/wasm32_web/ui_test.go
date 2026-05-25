package wasm32_web_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func TestUIModuleIncludesSchemaGuardAndRuntimeDispatch(t *testing.T) {
	src := string(wasm32_web.UIModule("app.ui.json"))
	for _, want := range []string{
		"tetra_ui: unsupported schema",
		`bundle.schema !== "tetra.ui.v1"`,
		"runtime: web command dispatch",
		"function applyTetraCommand(state, view, command)",
		"function parseOperationValue(viewState, value)",
		`if (text.startsWith("state.")) {`,
		`return viewState[statePath(text)];`,
		`if (text === "true") {`,
		`if (text === "false") {`,
		`Number.parseInt(text, 10)`,
		`case "state_add":`,
		`case "state_sub":`,
		"viewState[field] = parseOperationValue(viewState, op.value);",
		"function applyAccessibilityMetadata(host, view)",
		`host.setAttribute("data-tetra-accessibility-" + entry.name, value);`,
		`host.setAttribute("role", value);`,
		`host.setAttribute("aria-label", value);`,
		"applyAccessibilityMetadata(host, view);",
		"function applyStyleMetadata(host, view)",
		`host.setAttribute("data-tetra-style-" + entry.name, value);`,
		"applyStyleMetadata(host, view);",
		"function renderInputControl(host, state, view, binding)",
		`input.setAttribute("data-tetra-kind", "input");`,
		`input.addEventListener("input", () => {`,
		`input.addEventListener("change", () => {`,
		"function renderSelectControl(host, state, view, event)",
		`addLine(host, "  event " + event.name + " -> " + event.command);`,
		`select.setAttribute("data-tetra-kind", "list");`,
		`select.addEventListener("select", dispatch);`,
		`select.addEventListener("change", dispatch);`,
		`button.addEventListener("click"`,
		`new URL("app.ui.json", import.meta.url)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("UI module missing %q:\n%s", want, src)
		}
	}
}

func TestUIHTMLPageMountsUIShellBeforeRunningWASM(t *testing.T) {
	html := string(wasm32_web.UIHTMLPage("app.wasm", "app.mjs", "app.ui.web.mjs"))
	mountIdx := strings.Index(html, "await mountTetraUI(root);")
	runIdx := strings.Index(html, "await runTetra(")
	if mountIdx < 0 || runIdx < 0 {
		t.Fatalf("UI HTML missing mount/run hooks:\n%s", html)
	}
	if mountIdx > runIdx {
		t.Fatalf("UI HTML should mount UI metadata shell before running wasm:\n%s", html)
	}
}

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
				"schema: tetra.ui.v1",
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
		`"schema": "tetra.ui.v1"`,
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

func TestLoweredUIJSONIsDeterministic(t *testing.T) {
	src := `
module app.main

state CounterState:
    var count: Int = 0
    val enabled: Bool = true

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    bind enabledValue: Bool = state.enabled
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    style visible: Bool = true
    accessibility label: String = "Increment"
    accessibility enabled: Bool = true

func main() -> Int:
    return 0
`
	first := marshalLoweredUIForTest(t, src)
	second := marshalLoweredUIForTest(t, src)
	if first != second {
		t.Fatalf("lowered UI JSON is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	for _, want := range []string{
		`"schema":"tetra.ui.v1"`,
		`"name":"app.main.CounterView"`,
		`"styles":[{"name":"width","type":"i32","value":"320"},{"name":"visible","type":"bool","value":"true"}]`,
		`"accessibility":[{"name":"label","type":"str","value":"\"Increment\""},{"name":"enabled","type":"bool","value":"true"}]`,
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("lowered UI JSON missing %q:\n%s", want, first)
		}
	}
}

func TestLoweredUIAccessibilityMetadataAllowsScalarTypes(t *testing.T) {
	bundle := lowerUIForTest(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> noop
    command noop:
        state.count = state.count
    accessibility label: String = "Count"
    accessibility tabIndex: Int = 0
    accessibility disabled: Bool = false

func main() -> Int:
    return 0
`)
	got := bundle.Views[0].Accessibility
	if len(got) != 3 {
		t.Fatalf("accessibility metadata = %#v", got)
	}
	wants := []lower.UILoweredAccessibility{
		{Name: "label", Type: "str", Value: `"Count"`},
		{Name: "tabIndex", Type: "i32", Value: "0"},
		{Name: "disabled", Type: "bool", Value: "false"},
	}
	for i, want := range wants {
		if got[i] != want {
			t.Fatalf("accessibility[%d] = %#v, want %#v", i, got[i], want)
		}
	}
}

func TestLoweredUIStyleMetadataAllowsScalarTypes(t *testing.T) {
	bundle := lowerUIForTest(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> noop
    command noop:
        state.count = state.count
    style width: Int = 320
    style visible: Bool = true
    style tone: String = "primary"

func main() -> Int:
    return 0
`)
	got := bundle.Views[0].Styles
	if len(got) != 3 {
		t.Fatalf("style metadata = %#v", got)
	}
	wants := []lower.UILoweredStyle{
		{Name: "width", Type: "i32", Value: "320"},
		{Name: "visible", Type: "bool", Value: "true"},
		{Name: "tone", Type: "str", Value: `"primary"`},
	}
	for i, want := range wants {
		if got[i] != want {
			t.Fatalf("styles[%d] = %#v, want %#v", i, got[i], want)
		}
	}
}

func marshalLoweredUIForTest(t *testing.T, src string) string {
	t.Helper()
	bundle := lowerUIForTest(t, src)
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(raw)
}

func lowerUIForTest(t *testing.T, src string) *lower.UILoweredBundle {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "ui_test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	checked, err := semantics.CheckWorldOpt(&module.World{
		EntryModule:      file.Module,
		Files:            []*frontend.FileAST{file},
		ByModule:         map[string]*frontend.FileAST{file.Module: file},
		InterfaceModules: map[string]bool{},
		InterfaceHashes:  map[string]string{},
	}, semantics.CheckOptions{RequireMain: true})
	if err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
	bundle, err := lower.LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	if bundle == nil || len(bundle.Views) == 0 {
		t.Fatalf("bundle = %#v", bundle)
	}
	return bundle
}
