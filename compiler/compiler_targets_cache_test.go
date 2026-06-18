package compiler

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestNewOperatorMul(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 6 * 7 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorDiv(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 84 / 2 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorMod(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 47 % 5 }"
	_, code := buildAndRun(t, src)
	if code != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", code)
	}
}

func TestNewOperatorGreater(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (5 > 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorGreaterEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 >= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorLessEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 <= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorBangEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (2 != 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmp(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmpFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPipePipe(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorPipePipeFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPrecedenceMixed(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	// 2 + 3 * 4 = 2 + 12 = 14
	src := "fn main() -> i32 { return 2 + 3 * 4 }"
	_, code := buildAndRun(t, src)
	if code != 14 {
		t.Fatalf("exit code mismatch: got %d, want 14", code)
	}
}

func TestExprStmt(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun side(): i32 { return 0 }
fun main(): i32 { side(); return 42 }`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestExprStmtQualified(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun noop(): i32 {\n  return 0\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  r.noop()\n  return 42\n}\n",
	}
	_, code := buildAndRunFiles(t, files, "app/game.tetra")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildWASMHelloWritesModule(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "hello.tetra")

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, target+".wasm")
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read wasm: %v", err)
		}
		if len(data) < 8 {
			t.Fatalf("wasm too short: %d bytes", len(data))
		}
		if !bytes.Equal(data[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
			t.Fatalf("missing wasm magic: % x", data[:4])
		}
		if !bytes.Equal(data[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
			t.Fatalf("unexpected wasm version header: % x", data[4:8])
		}
		if target == "wasm32-web" {
			loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
			loaderRaw, err := os.ReadFile(loaderPath)
			if err != nil {
				t.Fatalf("read web loader: %v", err)
			}
			loader := string(loaderRaw)
			if !strings.Contains(loader, "tetra_web_v0.4.0") || !strings.Contains(loader, "tetra_main") {
				t.Fatalf("unexpected web loader content:\n%s", loader)
			}
		}
	}
}

func TestBuildWASMWebUIWritesSidecars(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_web.tetra")
	src := `state CounterState:
    var count: Int = 0
    val title: String = "Wave 9 Web"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui.wasm")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-web: %v", err)
	}
	uiJSON := strings.TrimSuffix(outPath, ".wasm") + ".ui.json"
	uiToolkitJSON := strings.TrimSuffix(outPath, ".wasm") + ".ui.toolkit.json"
	uiModule := strings.TrimSuffix(outPath, ".wasm") + ".ui.web.mjs"
	uiHTML := strings.TrimSuffix(outPath, ".wasm") + ".ui.html"

	jsonRaw, err := os.ReadFile(uiJSON)
	if err != nil {
		t.Fatalf("read ui json: %v", err)
	}
	if !strings.Contains(string(jsonRaw), `"schema": "tetra.ui.v0.4.0"`) || !strings.Contains(string(jsonRaw), "CounterView") {
		t.Fatalf("unexpected ui json:\n%s", string(jsonRaw))
	}
	toolkitRaw, err := os.ReadFile(uiToolkitJSON)
	if err != nil {
		t.Fatalf("read ui toolkit json: %v", err)
	}
	if !strings.Contains(string(toolkitRaw), `"schema": "tetra.ui.toolkit.v1"`) || !strings.Contains(string(toolkitRaw), `"compatibility_schema": "tetra.ui.v0.4.0"`) {
		t.Fatalf("unexpected ui toolkit json:\n%s", string(toolkitRaw))
	}
	moduleRaw, err := os.ReadFile(uiModule)
	if err != nil {
		t.Fatalf("read ui module: %v", err)
	}
	if !strings.Contains(string(moduleRaw), "mountTetraUI") {
		t.Fatalf("unexpected ui module:\n%s", string(moduleRaw))
	}
	htmlRaw, err := os.ReadFile(uiHTML)
	if err != nil {
		t.Fatalf("read ui html: %v", err)
	}
	if !strings.Contains(string(htmlRaw), ".ui.web.mjs") || !strings.Contains(string(htmlRaw), "runTetra") {
		t.Fatalf("unexpected ui html:\n%s", string(htmlRaw))
	}
}

func TestBuildNativeUIWritesShellSidecar(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_native.tetra")
	src := `state ShellState:
    var toggles: Int = 0
    val title: String = "Wave 9 Native"

view ShellView(state: ShellState):
    bind toggles: Int = state.toggles
    event submit -> toggle
    command toggle:
        state.toggles = state.toggles + 1
    style width: Int = 80
    accessibility label: String = "Toggle"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui-app")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux-x64: %v", err)
	}
	sidecarPath := outPath + ".ui.shell.txt"
	sidecar, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("read native ui shell sidecar: %v", err)
	}
	if !strings.Contains(string(sidecar), "ShellView") || !strings.Contains(string(sidecar), "event submit -> toggle") || !strings.Contains(string(sidecar), "state.toggles = 1") {
		t.Fatalf("unexpected native ui sidecar:\n%s", string(sidecar))
	}
	tracePath := outPath + ".ui.shell.json"
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read native ui shell json trace: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.ui.native-shell.v1"`,
		`"runtime": "native shell command dispatch"`,
		`"widgets":`,
		`"kind": "value"`,
		`"binding": "toggles"`,
		`"kind": "action"`,
		`"event": "submit"`,
		`"state_field": "toggles"`,
		`"state_value": "1"`,
	} {
		if !strings.Contains(string(trace), want) {
			t.Fatalf("native ui shell json trace missing %q:\n%s", want, trace)
		}
	}
}

func TestBuildCacheSeparatesNativeDebugAndReleaseModes(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	baseOpt := BuildOptions{Jobs: 1}
	stats1, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first base build")
	}

	stats2, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on base cache hit")
	}
	testkit.AssertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})

	debugOpt := BuildOptions{Jobs: 1, DebugInfo: true}
	stats3, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build3 debug: %v", err)
	}
	testkit.AssertModules(t, stats3.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first debug build")
	}

	stats4, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build4 debug: %v", err)
	}
	if len(stats4.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on debug cache hit")
	}
	testkit.AssertModules(t, stats4.CacheHits, []string{"app.game", "engine.render"})

	releaseOpt := BuildOptions{Jobs: 1, ReleaseOptimize: true}
	stats5, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build5 release: %v", err)
	}
	testkit.AssertModules(t, stats5.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats5.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first release build")
	}

	stats6, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build6 release: %v", err)
	}
	if len(stats6.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on release cache hit")
	}
	testkit.AssertModules(t, stats6.CacheHits, []string{"app.game", "engine.render"})

	stats7, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build7 base: %v", err)
	}
	if len(stats7.CompiledModules) != 0 {
		t.Fatalf("expected base mode cache to remain warm")
	}
	testkit.AssertModules(t, stats7.CacheHits, []string{"app.game", "engine.render"})
}

func TestBuildWASMCacheStatsRemainColdAcrossBuilds(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		tmp := t.TempDir()
		files := map[string]string{
			"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
			"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
		}
		writeTestFiles(t, tmp, files)
		entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
		outPath := filepath.Join(tmp, "out", target+".wasm")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		stats1, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build1 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats1.CacheHits) != 0 {
			t.Fatalf("%s unexpected cache hits on first build: %#v", target, stats1.CacheHits)
		}

		stats2, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build2 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats2.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats2.CacheHits) != 0 {
			t.Fatalf("%s expected cache to stay cold: %#v", target, stats2.CacheHits)
		}
	}
}
