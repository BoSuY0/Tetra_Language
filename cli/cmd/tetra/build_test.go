package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

func TestDefaultOutputUsesTargetExtensionAndEmitMode(t *testing.T) {
	tests := []struct {
		target string
		emit   string
		want   string
	}{
		{target: "linux-x64", emit: "exe", want: "app"},
		{target: "windows-x64", emit: "exe", want: "app.exe"},
		{target: "wasm32-wasi", emit: "exe", want: "app.wasm"},
		{target: "wasm32-web", emit: "exe", want: "app.wasm"},
		{target: "linux-x64", emit: "object", want: "app.tobj"},
		{target: "windows-x64", emit: "library", want: "app.tobj"},
	}
	for _, tt := range tests {
		tgt, err := ctarget.Parse(tt.target)
		if err != nil {
			t.Fatalf("parse target %s: %v", tt.target, err)
		}
		if got := defaultOutput(tgt, tt.emit); got != tt.want {
			t.Fatalf("defaultOutput(%s, %s) = %q, want %q", tt.target, tt.emit, got, tt.want)
		}
	}
}

func TestBuildCommandJSONDiagnosticsForOptionValidation(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"build", "--diagnostics=json", "--runtime=warpdrive", "examples/hello.tetra"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != `unsupported --runtime "warpdrive"` || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestBuildCommandWASMTargetWritesWasmModule(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int\nuses io:\n    print(\"wasm hello\\n\")\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(t.TempDir(), target+".wasm")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runCLI([]string{"build", "--target", target, "-o", outPath, srcPath}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read output: %v", err)
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

func TestBuildCommandLinuxX32AutoRuntimeWritesSelfHostELF(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "time_x32.tetra")
	outPath := filepath.Join(dir, "time-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", "x32", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 output: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x32 ELF too short: %d bytes", len(data))
	}
	if !bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
}

func TestBuildCommandUIWritesBackendSidecars(t *testing.T) {
	src := `state CounterState:
    var count: Int = 0

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

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "ui.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	wasmOut := filepath.Join(dir, "ui.wasm")
	if code := runCLI([]string{"build", "--target", "wasm32-web", "-o", wasmOut, srcPath}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("build wasm32-web exit code = %d", code)
	}
	for _, path := range []string{
		strings.TrimSuffix(wasmOut, ".wasm") + ".ui.json",
		strings.TrimSuffix(wasmOut, ".wasm") + ".ui.web.mjs",
		strings.TrimSuffix(wasmOut, ".wasm") + ".ui.html",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected sidecar %s: %v", path, err)
		}
	}

	host, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	nativeOut := filepath.Join(dir, "ui-native")
	if host == "windows-x64" {
		nativeOut += ".exe"
	}
	if code := runCLI([]string{"build", "--target", host, "-o", nativeOut, srcPath}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("build host exit code = %d", code)
	}
	shellSidecar := strings.TrimSuffix(nativeOut, ".exe") + ".ui.shell.txt"
	if _, err := os.Stat(shellSidecar); err != nil {
		t.Fatalf("expected native sidecar %s: %v", shellSidecar, err)
	}
}

func TestBuildCommandWASMWebPackageOutputIsDeterministic(t *testing.T) {
	src := `state CounterState:
    var count: Int = 0

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

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "ui.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	buildDir := func(name string) string {
		outDir := filepath.Join(dir, name)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			t.Fatal(err)
		}
		outPath := filepath.Join(outDir, "app.wasm")
		var stderr bytes.Buffer
		if code := runCLI([]string{"build", "--target", "wasm32-web", "-o", outPath, srcPath}, &bytes.Buffer{}, &stderr); code != 0 {
			t.Fatalf("build %s exit code = %d stderr=%q", name, code, stderr.String())
		}
		return outDir
	}

	first := buildDir("first")
	second := buildDir("second")
	for _, name := range []string{
		"app.wasm",
		"app.mjs",
		"app.ui.json",
		"app.ui.web.mjs",
		"app.ui.html",
	} {
		a, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatalf("read first %s: %v", name, err)
		}
		b, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			t.Fatalf("read second %s: %v", name, err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("wasm32-web package file %s is not deterministic", name)
		}
	}
}

func TestBuildCommandRejectsUnsupportedDiagnosticsMode(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=yaml", "--target", mustHostTarget(t), "-o", filepath.Join(dir, "app"), srcPath}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --diagnostics format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandRejectsInvalidTarget(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", "not-a-target", "examples/flow_hello.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported target") || !strings.Contains(stderr.String(), "supported targets: linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-web") || !strings.Contains(stderr.String(), "build-only targets: linux-x86, linux-x32") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandJSONDiagnosticsForInvalidTarget(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"build", "--diagnostics=json", "--target", "not-a-target", "examples/flow_hello.tetra"}, 2)
	if diag.Code != "TETRA0001" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	for _, want := range []string{"unsupported target: not-a-target", "supported targets: linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-web", "build-only targets: linux-x86, linux-x32"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
	if !strings.Contains(diag.Hint, "tetra targets") {
		t.Fatalf("diagnostic hint = %q", diag.Hint)
	}
}

func TestBuildCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"build", "--diagnostics=json", "one.tetra", "two.tetra"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "build accepts at most one input path" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestBuildCommandUsesDefaultInput(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	src := []byte(`fun main(): i32 { return 0 }`)
	if err := os.WriteFile(filepath.Join(dir, "main.t4"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	out := filepath.Join(dir, "app")

	var stdout bytes.Buffer
	code := runCLI([]string{"build", "--target", mustHostTarget(t), "-o", out}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q", code, stdout.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
}

func TestBuildCommandDiscoversCapsuleT4ProjectEntry(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	out := filepath.Join(dir, "app")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", mustHostTarget(t), "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
}

func TestBuildAndRunCommandsAcceptExplicitProjectDirectory(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux-x64
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 7\n")

	out := filepath.Join(dir, "demo")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", mustHostTarget(t), "-o", out, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"run", "--target", mustHostTarget(t), dir}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestBuildCommandUsesCapsuleInterfaceAndObjectArtifacts(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	libSrc := filepath.Join(dir, "Math", "src", "math", "core.t4")
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	iface, err := compiler.GenerateInterfaceFile(libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFile: %v", err)
	}
	writeCLIProjectFile(t, dir, "App/interfaces/math/core.t4i", string(iface))
	objPath := filepath.Join(dir, "App", "artifacts", "math-core.tobj")
	if err := os.MkdirAll(filepath.Dir(objPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(libSrc, objPath, target, compiler.BuildOptions{Jobs: 1, Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("emit math library: %v", err)
	}
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
    artifacts:
        interface interfaces/math/core.t4i
        object artifacts/math-core.tobj
`, target))
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "App")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	out := filepath.Join(dir, "App", "app")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", target, "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
}

func TestBuildCommandArtifactsAutoRepairsStaleObject(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b + 1\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(appRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	stdout.Reset()
	stderr.Reset()
	out := filepath.Join(appRoot, "app")
	code = runCLI([]string{"build", "--artifacts=auto", "--target", target, "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build --artifacts=auto exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
	if !strings.Contains(stdout.String(), "Artifacts repaired") {
		t.Fatalf("stdout = %q, want repair message", stdout.String())
	}
}

func TestBuildCommandWASMProjectLockDoesNotRequireNativeArtifacts(t *testing.T) {
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, "wasm32-wasi")
	lockPath := filepath.Join(appRoot, "Tetra.lock")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(appRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"build"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, "app.wasm")); err != nil {
		t.Fatalf("expected wasm build output: %v", err)
	}
}

func TestBuildCommandUsesCapsuleDefaultTarget(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        wasm32-wasi
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "app.wasm")); err != nil {
		t.Fatalf("expected wasm default build output: %v", err)
	}
}

func TestBuildCommandAllTargetsBuildsCapsuleTargets(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux
        wasm32-wasi
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build", "--all-targets"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build --all-targets exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"app-linux-x64", "app-wasm32-wasi.wasm"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestBuildCommandJSONDiagnostics(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"build", "--diagnostics=json", "--target", mustHostTarget(t), "-o", filepath.Join(dir, "app"), srcPath}, 1)
	if diag.Message == "" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}
