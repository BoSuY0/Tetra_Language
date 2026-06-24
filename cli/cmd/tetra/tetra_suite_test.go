package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
	"tetra_language/internal/toon"
	"tetra_language/tools/validators/surface"
)

// ---- actor_net_test.go ----

func TestActorNetCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"actor-net", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"actor-net --help exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "usage: tetra actor-net") {
		t.Fatalf("actor-net help = %q, want usage", stdout.String())
	}
}

func TestActorNetCommandRejectsInvalidArgs(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"actor-net", "--definitely-invalid"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("actor-net invalid arg exit code = %d, stderr=%q", code, stderr.String())
	}
}

// ---- build_test.go ----

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
	diag := runCLIJSONDiagnostic(
		t,
		[]string{
			"build",
			"--diagnostics=json",
			"--runtime=warpdrive",
			"examples/smoke/basic/hello.tetra",
		},
		2,
	)
	if diag.Code != "TETRA0001" || diag.Message != `unsupported --runtime "warpdrive"` ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestApplyRuntimeHeapTelemetryOptionsRequiresExplicitFlagAndLinuxX64(t *testing.T) {
	var opt compiler.BuildOptions
	if applyRuntimeHeapTelemetryOptions(
		&opt,
		"linux-x64",
		false,
		"reports/heap",
		"text",
		ioDiscard{},
	) {
		t.Fatalf("telemetry dir without telemetry flag should be rejected")
	}

	opt = compiler.BuildOptions{}
	if applyRuntimeHeapTelemetryOptions(
		&opt,
		"wasm32-wasi",
		true,
		"reports/heap",
		"text",
		ioDiscard{},
	) {
		t.Fatalf("runtime heap telemetry should reject unsupported targets")
	}

	opt = compiler.BuildOptions{}
	if !applyRuntimeHeapTelemetryOptions(
		&opt,
		"linux-x64",
		true,
		"reports/heap",
		"text",
		ioDiscard{},
	) {
		t.Fatalf("runtime heap telemetry linux-x64 option should be accepted")
	}
	if !opt.EmitRuntimeHeapTelemetry || opt.RuntimeHeapTelemetryDir != "reports/heap" {
		t.Fatalf(
			"BuildOptions telemetry = enabled:%v dir:%q",
			opt.EmitRuntimeHeapTelemetry,
			opt.RuntimeHeapTelemetryDir,
		)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestBuildCommandExplainFlagsWriteReports(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    for x in xs:
        return x
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{
			"build",
			"--target",
			"linux-x64",
			"--explain",
			"--emit-plir",
			"--emit-proof",
			"--emit-bounds-report",
			"--emit-alloc-report",
			"-o",
			outPath,
			srcPath,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, suffix := range []string{
		".plir.txt",
		".proof.json",
		".bounds.json",
		".alloc.json",
		".explain.txt",
	} {
		if _, err := os.Stat(outPath + suffix); err != nil {
			t.Fatalf("missing report %s: %v", outPath+suffix, err)
		}
	}
}

func TestBuildCommandWASMTargetWritesWasmModule(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := strings.Join([]string{
		"func main() -> Int\n",
		"uses io:\n",
		"    print(\"wasm hello\\n\")\n",
		"    return 0\n",
	}, "")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(t.TempDir(), target+".wasm")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runCLI(
			[]string{"build", "--target", target, "-o", outPath, srcPath},
			&stdout,
			&stderr,
		)
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
			if !strings.Contains(loader, "tetra_web_v0.4.0") ||
				!strings.Contains(loader, "tetra_main") {
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
	if code := runCLI(
		[]string{"build", "--target", "wasm32-web", "-o", wasmOut, srcPath},
		&bytes.Buffer{},
		&bytes.Buffer{},
	); code != 0 {
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
	if code := runCLI(
		[]string{"build", "--target", host, "-o", nativeOut, srcPath},
		&bytes.Buffer{},
		&bytes.Buffer{},
	); code != 0 {
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
		if code := runCLI(
			[]string{"build", "--target", "wasm32-web", "-o", outPath, srcPath},
			&bytes.Buffer{},
			&stderr,
		); code != 0 {
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
	code := runCLI(
		[]string{
			"build",
			"--diagnostics=yaml",
			"--target",
			mustHostTarget(t),
			"-o",
			filepath.Join(dir, "app"),
			srcPath,
		},
		&bytes.Buffer{},
		&stderr,
	)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --diagnostics format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandRejectsInvalidTarget(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI(
		[]string{"build", "--target", "not-a-target", "examples/flow/flow_hello.tetra"},
		&bytes.Buffer{},
		&stderr,
	)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported target") ||
		!strings.Contains(
			stderr.String(),
			"supported targets: linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-web",
		) ||
		!strings.Contains(stderr.String(), "build-only targets: linux-x86, linux-x32") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandJSONDiagnosticsForInvalidTarget(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{
			"build",
			"--diagnostics=json",
			"--target",
			"not-a-target",
			"examples/flow/flow_hello.tetra",
		},
		2,
	)
	if diag.Code != "TETRA0001" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	for _, want := range []string{"unsupported target: not-a-target", (("supported targets: " +
		"linux-x64, windows-x64, macos-x64, ") +
		"wasm32-wasi, wasm32-web"), "build-only targets: linux-x86, linux-x32"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
	if !strings.Contains(diag.Hint, "tetra targets") {
		t.Fatalf("diagnostic hint = %q", diag.Hint)
	}
}

func TestBuildCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"build", "--diagnostics=json", "one.tetra", "two.tetra"},
		2,
	)
	if diag.Code != "TETRA0001" || diag.Message != "build accepts at most one input path" ||
		diag.Severity != "error" {
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
	code := runCLI(
		[]string{"build", "--target", mustHostTarget(t), "-o", out},
		&stdout,
		&bytes.Buffer{},
	)
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
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
	code := runCLI(
		[]string{"build", "--target", mustHostTarget(t), "-o", out, dir},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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

func TestBuildCheckRunCommandsAcceptExplicitProjectSourceFile(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	srcPath := filepath.Join(
		"..",
		"..",
		"..",
		"examples",
		"projects",
		"dogfood_cli",
		"src",
		"main.tetra",
	)
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing dogfood source %s: %v", srcPath, err)
	}

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	out := filepath.Join(t.TempDir(), "dogfood-cli")
	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{"build", "--target", mustHostTarget(t), "-o", out, srcPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "tetra dogfood cli: ok") {
		t.Fatalf("run stdout = %q", stdout.String())
	}
}

func TestBuildCommandUsesCapsuleInterfaceAndObjectArtifacts(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	libSrc := filepath.Join(dir, "Math", "src", "math", "core.t4")
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)
	iface, err := compiler.GenerateInterfaceFile(libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFile: %v", err)
	}
	writeCLIProjectFile(t, dir, "App/interfaces/math/core.t4i", string(iface))
	objPath := filepath.Join(dir, "App", "artifacts", "math-core.tobj")
	if err := os.MkdirAll(filepath.Dir(objPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		libSrc,
		objPath,
		target,
		compiler.BuildOptions{Jobs: 1, Emit: compiler.EmitLibrary},
	); err != nil {
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
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	)

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
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
	code := runCLI(
		[]string{
			"eco",
			"artifacts",
			"build",
			"--target",
			target,
			"--lock",
			lockPath,
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco artifacts build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b + 1\n",
	)

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
	code = runCLI(
		[]string{"build", "--artifacts=auto", "--target", target, "-o", out},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"build --artifacts=auto exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
	code := runCLI(
		[]string{"eco", "verify", "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
		t.Fatalf(
			"build --all-targets exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
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
	if err := os.WriteFile(
		srcPath,
		[]byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(
		t,
		[]string{
			"build",
			"--diagnostics=json",
			"--target",
			mustHostTarget(t),
			"-o",
			filepath.Join(dir, "app"),
			srcPath,
		},
		1,
	)
	if diag.Message == "" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

// ---- check_diagnostics_actor_transitive_test.go ----

func TestCheckCommandJSONDiagnosticsForTransitiveActorAliasUseAfterTransferCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_actor_transitive_alias_transfer.tetra")
		src := `func worker() -> Int:
    return 0

func alias_one(peer: actor) -> actor:
    return peer

func alias_two(peer: actor) -> actor:
    return alias_one(peer)

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(peer: actor) -> actor:
    return peer

pub func alias_two(peer: actor) -> actor:
    return alias_one(peer)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = resources.alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForTaskGroupCancelReturnProvenanceCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_group_cancel_return_provenance.tetra")
		src := `func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'canceled'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = resources.cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'canceled'")
	})
}

func TestCheckCommandJSONDiagnosticsForTaskHandleGroupOptionalPayloadJoinCloseAliasCodes(
	t *testing.T,
) {
	t.Run("same module task-handle if-let optional-payload join", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_optional_payload_join_alias.tetra")
		src := `func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    if let other = maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle match optional-payload join", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group if-let optional-payload close", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_optional_payload_close_alias.tetra")
		src := `func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group match optional-payload close", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func pass(maybe: task.group?) -> task.group?:
    return maybe
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    case none:
        return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForActorEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_enum_payload_alias_transfer.tetra")
	src := `enum MoveMsg:
    case handoff(actor)

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: MoveMsg = MoveMsg.handoff(peer)
    match msg:
    case MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleActorEnumPayloadAliasUseAfterTransferCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum MoveMsg:
    case handoff(actor)

pub func pass(msg: MoveMsg) -> MoveMsg:
    return msg
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: resources.MoveMsg = resources.MoveMsg.handoff(peer)
    let returned: resources.MoveMsg = resources.pass(msg)
    match returned:
    case resources.MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForTaskStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_struct_field_alias_transfer.tetra")
	src := `struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.handle'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleTaskStructFieldAliasUseAfterTransferCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.handle'")
}

func TestCheckCommandJSONDiagnosticsForTaskEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_enum_payload_alias_transfer.tetra")
	src := `enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func pass(msg: TaskMsg) -> TaskMsg:
    return msg

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: TaskMsg = TaskMsg.wrap(task)
    let returned: TaskMsg = pass(msg)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleTaskEnumPayloadAliasUseAfterTransferCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func pass(msg: TaskMsg) -> TaskMsg:
    return msg
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: resources.TaskMsg = resources.TaskMsg.wrap(task)
    let returned: resources.TaskMsg = resources.pass(msg)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForPrivacyConsentSafetyCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_privacy.tetra")
	src := `func seal(token: consent.token) -> secret.i32
uses privacy:
    return core.secret_seal_i32(1, token)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(
		t,
		srcPath,
		srcPath,
		compiler.DiagnosticCodeSafetyPrivacy,
		"uses effect 'privacy' requires semantic clause 'privacy'",
	)
}

func TestCheckCommandJSONDiagnosticsForRecursiveSecretSignaturePrivacyCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_secret_signature.tetra")
	src := `func seal(payload: secret.i32?) -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.File != srcPath ||
		diag.Severity != "error" ||
		diag.Message != "secret types in function signature require semantic clause 'privacy'" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"check", "--diagnostics=json", "one.tetra", "two.tetra"},
		2,
	)
	if diag.Code != "TETRA0001" || diag.Message != "check accepts at most one input path" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandRejectsLocalCapsuleDependencyCycle(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nfunc main() -> Int:\n    return 0\n",
	)
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    deps:
        tetra://app 0.1.0 ../App
`)
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)

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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf(
			"expected check failure for dependency cycle, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "capsule dependency cycle") {
		t.Fatalf("stderr = %q, want capsule dependency cycle", stderr.String())
	}
}

// ---- check_diagnostics_callable_test.go ----

func TestCheckCommandJSONDiagnosticsForCallableMutableCaptureGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_global_escape.tetra")
	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        return total + x
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"global-escaped function value captures mutable local 'total'",
	)
}

func TestCheckCommandJSONDiagnosticsForCapturedCallableGlobalStorageCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_captured_callable_global_storage.tetra")
	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: identity(captured))
    cb = holder.cb
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"captured function value cannot be stored in global function-typed value 'cb'",
	)
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedParameterGlobalStorageCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_parameter_global_storage.tetra")
	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = f
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"function-typed parameter 'f' cannot be stored in global function-typed value 'cb'",
	)
}

func TestCheckCommandJSONDiagnosticsForFunctionValueUnsupportedEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_value_escape.tetra")
	src := `func add1(x: Int) -> Int:
    return x + 1

func take_ptr(x: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return take_ptr(f)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"function value 'f' cannot escape outside the supported fnptr ABI",
	)
}

func TestCheckCommandJSONDiagnosticsForCapturingClosureRawPointerEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_capturing_closure_raw_pointer_escape.tetra")
	src := `func choose(p: ptr) -> Int:
    return 0

func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return choose(f)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "capturing closure 'f' cannot escape as raw ptr")
}

func TestCheckCommandJSONDiagnosticsForCallableResourceCaptureEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_resource_capture_escape.tetra")
	src := `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"escaped function value captures local 'box' of type 'PtrBox'",
	)
}

func TestCheckCommandJSONDiagnosticsForCallableMutableCaptureHeapEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_mutable_capture_heap_escape.tetra")
	src := `func pick() -> fn(Int) -> Int:
    var total: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + total + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"heap-escaped function value captures mutable local 'total'",
	)
}

func TestCheckCommandJSONDiagnosticsForGenericClosureCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_capture.tetra")
	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "generic closure literal captures local 'base'")
}

func TestCheckCommandJSONDiagnosticsForGenericCallbackClosureCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_callback_closure_capture.tetra")
	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    , 41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"callback argument 'closure literal' captures local 'base'",
	)
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedStorageUnsupportedCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_storage_capture.tetra")
	src := `struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"function-typed storage 'f' captures unsupported local 'box'",
	)
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedReturnUnsupportedCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_return_capture.tetra")
	src := `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"function-typed return 'closure literal' captures unsupported local 'box'",
	)
}

func TestCheckCommandJSONDiagnosticsForCapturedClosureExplicitTypeArgsCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_captured_closure_explicit_type_args.tetra")
	src := `func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f<Int>(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"explicit type arguments are not supported for captured closure 'f'",
	)
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedExplicitTypeArgsCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_explicit_type_args.tetra")
	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f<Int>(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"explicit type arguments are not supported for function-typed callback 'f'",
	)
}

func TestCheckCommandJSONDiagnosticsForUnsupportedFunctionValueCallCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_value_call.tetra")
	src := `func main() -> Int:
    let p: ptr = 0
    return p(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"function value 'p' cannot be called through the supported fnptr ABI",
	)
}

func TestCheckCommandJSONDiagnosticsForGenericClosurePointerEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_pointer_escape.tetra")
	src := `func use(p: ptr) -> Int:
    return 0

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    return use(id)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"generic closure 'id' cannot be used as a pointer value",
	)
}

func TestCheckCommandJSONDiagnosticsForGenericClosureDirectCallRequirementCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_direct_call_requirement.tetra")
	src := `func main() -> Int:
    var id: ptr = fn<T>(x: T) -> T:
        return x
    return id(1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"generic closure 'id' requires the generic direct-call closure ABI",
	)
}

// ---- check_diagnostics_lifetime_borrow_test.go ----

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime.tetra")
	src := `func leak(x: borrow ptr) -> ptr:
    return x
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowFixedArrayAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_fixed_array_alias_return.tetra")
	src := `func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowStringAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_string_alias_return.tetra")
	src := `func leak(x: borrow str) -> str:
    let y: str = x
    return y
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed String return requires '-> borrow String' or '.copy()'",
	)
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowOptionalAssignmentEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_optional_assignment.tetra")
	src := `func leak(x: borrow ptr) -> ptr?:
    var maybe: ptr? = none
    maybe = x
    return maybe
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_optional_assignment.tetra")
	src := `func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return",
	)
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
	}{
		{
			name: "owned",
			src: `func sink(value: []u8?) -> Int:
    return 0
func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			src: `func sink(value: consume []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_lifetime_slice_optional_assignment_"+tt.name+".tetra",
			)
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceOptionalAssignmentEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		wantCode string
		wantText string
	}{
		{
			name: "return",
			libSrc: `module lib.leak

pub func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: ("aggregate '[]u8?' contains borrowed slice field '$elem' " +
				"that cannot escape through owned return"),
		},
		{
			name: "owned",
			libSrc: `module lib.leak

pub func sink(value: []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.leak.sink'"),
		},
		{
			name: "consume",
			libSrc: `module lib.leak

pub func sink(value: consume []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leak.sink'",
		},
		{
			name: "inout",
			libSrc: `module lib.leak

pub func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leak.t4")
			writeCLIProjectFile(t, dir, "src/lib/leak.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leak as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONDiagnosticForPath(t, srcPath, libPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8
func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)

func main() -> Int:
    return 0
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_slice_struct_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceStructEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox
func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_nested_slice_struct_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceStructEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceEnumPayloadEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty
func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_lifetime_nested_slice_enum_payload_"+tt.name+".tetra",
			)
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceEnumPayloadEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))
`,
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceEnumEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "direct-return",
			src: `enum BufMsg:
    case send([]u8)
func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)

func main() -> Int:
    return 0
`,
			wantText: ("aggregate 'BufMsg' contains borrowed slice field " +
				"'BufMsg.send[1]' that cannot escape through owned return"),
		},
		{
			name: "alias-return",
			src: `enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_slice_enum_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceEnumEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "direct-return",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)
`,
			wantText: ("aggregate 'BufMsg' contains borrowed slice field " +
				"'BufMsg.send[1]' that cannot escape through owned return"),
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForSafeViewBorrowedOwnedReturnCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_owned_return.tetra")
	src := `func bad(xs: borrow []u8) -> []u8:
    return xs.borrow()

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed slice return requires '-> borrow []u8' or '.copy()'",
	)
}

func TestCheckCommandJSONDiagnosticsForSafeViewActorBoundaryCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_actor_boundary.tetra")
	src := `enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(
		t,
		srcPath,
		srcPath,
		compiler.DiagnosticCodeSafetyOwnership,
		"cannot cross actor boundary without copy",
	)
}

func TestCheckCommandJSONDiagnosticsForSafeViewTaskBoundaryCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_task_boundary.tetra")
	src := `enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", srcPath}, 1)
	if diag.Severity != "error" ||
		!strings.Contains(
			diag.Message,
			"typed task error payload must be sendable across task boundary",
		) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandJSONDiagnosticsForSafeViewAggregateHiddenBorrowCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_aggregate_return.tetra")
	src := `struct Box:
    bytes: []u8

func bad(xs: borrow []u8) -> Box:
    return Box(bytes: xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"aggregate 'Box' contains borrowed slice field 'bytes' that cannot escape through owned return",
	)
}

// ---- check_diagnostics_lifetime_global_assignment_test.go ----

func TestCheckCommandJSONDiagnosticsForBorrowedPtrOptionalGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_global.tetra")
	src := `var leaked: ptr? = none

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForBorrowedStringGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_string_global.tetra")
	src := `var leaked: str = ""

func leak(x: borrow str) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrOptionalGlobalAssignmentCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: ptr? = none

pub func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(
		t,
		srcPath,
		libPath,
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForBorrowedPtrAggregateOptionalGlobalAssignmentCode(
	t *testing.T,
) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_optional_global.tetra")
	src := `struct PtrBox:
    raw: ptr

var leaked: PtrBox? = none

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrAggregateOptionalGlobalAssignmentCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks
import lib.model as model

var leaked: model.PtrBox? = none

pub func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(
		t,
		srcPath,
		libPath,
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForBorrowedSliceOptionalPayloadGlobalAssignmentCode(
	t *testing.T,
) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_optional_payload_global.tetra")
	src := `var leaked: []u8? = none

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
	)
}

func TestCheckCommandJSONDiagnosticsForBorrowedSliceGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_global.tetra")
	src := `var leaked: []u8

func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceOptionalPayloadGlobalAssignmentCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: []u8? = none

pub func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(
		t,
		srcPath,
		libPath,
		"aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: []u8

pub func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(
		t,
		srcPath,
		libPath,
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

// ---- check_diagnostics_lifetime_ptr_test.go ----

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_alias_return.tetra")
	src := `enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumAliasReturnEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func leak(x: borrow ptr) -> model.PtrMsg:
    let msg: model.PtrMsg = model.PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateReturnEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "whole",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "field",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "alias",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "nested-field",
			src: `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func leak(outer: borrow OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_"+tt.name+"_return.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateReturnEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "whole",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> model.PtrBox:
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "field",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "alias",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> model.PtrBox:
    let alias: model.PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "nested-field",
			src: `module app.main
import lib.model as model

func leak(outer: borrow model.OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.src)

			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_lifetime_ptr_optional_assignment_"+tt.name+"_global.tetra",
			)
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(
				t,
				srcPath,
				"borrowed local 'x' cannot escape via global assignment to 'leaked'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)

			assertCLIJSONLifetimeDiagnosticForPath(
				t,
				srcPath,
				libPath,
				"borrowed local 'x' cannot escape via global assignment to 'leaked'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			src: `struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate whole global",
			src: `struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate global field target",
			src: `struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "nested aggregate",
			src: `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

var leaked: ptr = 0

func leak(outer: borrow OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_global.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateGlobalEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "aggregate",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate whole global",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate global field target",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "nested aggregate",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(outer: borrow model.OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)

			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_return.tetra")
	src := `enum PtrMsg:
    case raw(ptr)
    case empty

func leak(msg: borrow PtrMsg) -> ptr:
    match msg:
    case PtrMsg.raw(raw):
        return raw
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadReturnEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func leak(msg: borrow model.PtrMsg) -> ptr:
    match msg:
    case model.PtrMsg.raw(raw):
        return raw
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_global.tetra")
	src := `enum PtrMsg:
    case raw(ptr)
    case empty

var leaked: ptr = 0

func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        leaked = raw
        return 0
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_global.tetra")
	src := `enum PtrMsg:
    case raw(ptr)
    case empty

var leaked: PtrMsg

func leak(msg: borrow PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadGlobalEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(msg: borrow model.PtrMsg) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        leaked = raw
        return 0
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumGlobalEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

var leaked: model.PtrMsg

func leak(msg: borrow model.PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via global assignment to 'leaked'",
	)
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadInoutEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_inout.tetra")
	src := `enum PtrMsg:
    case raw(ptr)
    case empty

func leak(msg: borrow PtrMsg, out: inout ptr) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        out = raw
        return 0
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via inout assignment to 'out'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadInoutEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func leak(msg: borrow model.PtrMsg, out: inout ptr) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        out = raw
        return 0
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"borrowed local 'msg' cannot escape via inout assignment to 'out'",
	)
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_inout.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(
				t,
				srcPath,
				"borrowed local 'maybe' cannot escape via inout assignment to 'out'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(
				t,
				srcPath,
				libPath,
				"borrowed local 'maybe' cannot escape via inout assignment to 'out'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_global.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(
				t,
				srcPath,
				"borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(
				t,
				srcPath,
				libPath,
				"borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_return.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(
				t,
				srcPath,
				"borrowed local 'maybe' cannot escape via return",
			)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(
				t,
				srcPath,
				libPath,
				"borrowed local 'maybe' cannot escape via return",
			)
		})
	}
}

// ---- check_diagnostics_metadata_test.go ----

func TestCheckCommandJSONDiagnosticsForRepresentationMetadataAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct BufferBox:
    bytes: []u8
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func main() -> Int
uses alloc, mem:
    var box: model.BufferBox = model.BufferBox(bytes: make_u8(2))
    box.bytes.len = 9
    return 0
`)

	assertCLIJSONSemanticDiagnostic(
		t,
		srcPath,
		srcPath,
		"representation metadata field 'len' is not user-assignable in safe code",
	)
}

// ---- check_diagnostics_misc_test.go ----

func TestCheckCommandJSONDiagnosticsForSemanticError(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(
		srcPath,
		[]byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(
		t,
		srcPath,
		srcPath,
		compiler.DiagnosticCodeSafetyEffect,
		"uses effect 'io'",
	)
}

func TestCheckCommandJSONDiagnosticsForGenericBorrowReturnCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			src: `
struct PtrBox:
    raw: ptr

func leak<T>(value: borrow T) -> T:
    return value

func caller(x: borrow ptr) -> PtrBox:
    return leak(PtrBox(raw: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional-ptr",
			src: `
func leak<T>(value: borrow T) -> T:
    return value

func caller(maybe: borrow ptr?) -> ptr?:
    return leak(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_generic_borrow_return_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleGenericBorrowReturnCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appSrc   string
		wantText string
	}{
		{
			name: "aggregate",
			libSrc: `module lib.leak

pub struct PtrBox:
    raw: ptr

pub func leak<T>(value: borrow T) -> T:
    return value
`,
			appSrc: `func caller(x: borrow ptr) -> leaks.PtrBox:
    return leaks.leak(leaks.PtrBox(raw: x))
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional-ptr",
			libSrc: `module lib.leak

pub func leak<T>(value: borrow T) -> T:
    return value
`,
			appSrc: `func caller(maybe: borrow ptr?) -> ptr?:
    return leaks.leak(maybe)
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/leak.t4", tt.libSrc)
			libPath := filepath.Join(dir, "src", "lib", "leak.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leak as leaks

`+tt.appSrc+`
func main() -> Int:
    return 0
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForProtocolImplOwnershipMismatchCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_protocol_impl_ownership_mismatch.tetra")
		src := `
struct Box:
    value: Int

protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink

func main() -> Int:
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONSemanticDiagnostic(
			t,
			srcPath,
			srcPath,
			("method 'Box.sink' does not match protocol 'Sink' " +
				"requirement 'sink': parameter 1 ownership differs: expected " +
				"'consume', got 'owned'"),
		)
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		libPath := filepath.Join(dir, "src", "lib", "model.t4")
		writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct Box:
    value: Int

pub protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func main() -> Int:
    return 0
`)
		assertCLIJSONSemanticDiagnostic(
			t,
			srcPath,
			libPath,
			("method 'lib.model.Box.sink' does not match protocol " +
				"'lib.model.Sink' requirement 'sink': parameter 1 ownership " +
				"differs: expected 'consume', got 'owned'"),
		)
	})
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowSliceAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: BufBox) -> Int:
    return 0
`,
			appCall: "return sinker.sink(sinker.BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.sink.sink'"),
		},
		{
			name: "struct-consume",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: consume BufBox) -> Int:
    return 0
`,
			appCall:  "let box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			appCall:  "var box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: BufMsg) -> Int:
    return 0
`,
			appCall: "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.sink.sink'"),
		},
		{
			name: "enum-consume",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: consume BufMsg) -> Int:
    return 0
`,
			appCall:  "let msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			appCall:  "var msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForScopedIslandOptionalRegionEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_scoped_island_optional_region.tetra")
	src := `func make() -> []u8?
uses alloc, islands, mem:
    island(16) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        var maybe: []u8? = none
        maybe = xs
        return maybe
    return none
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(
		t,
		srcPath,
		"slice from scoped island cannot escape to outer scope",
	)
}

// ---- check_diagnostics_ownership_basic_test.go ----

func TestCheckCommandJSONDiagnosticsForOwnershipUseAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership.tetra")
	src := `func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'a'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialStructConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_struct_consume.tetra")
	src := `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func use(pair: Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return use(pair) + moved
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'pair.left'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialStructCopyAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_struct_copy_after_consume.tetra")
	src := `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: Pair = pair
    return moved + copy.right
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'pair.left'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_enum_consume.tetra")
	src := `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: PairMsg) -> Int:
    match msg:
    case PairMsg.both(left, right):
        return left + right
    case PairMsg.empty:
        return 0

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return use(msg) + moved
    case PairMsg.empty:
        return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumCopyAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_enum_copy_after_consume.tetra")
	src := `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: PairMsg = msg
        return moved + right
    case PairMsg.empty:
        return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestCheckCommandJSONDiagnosticsForCrossModulePartialCopyAfterConsumeCodes(t *testing.T) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "struct",
			modelSrc: `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: model.Pair = pair
    return moved + copy.right
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum",
			modelSrc: `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: model.PairMsg = msg
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConstructorAfterConsumeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "struct field",
			src: `struct Pair:
    left: Int
    right: Int

enum Wrap:
    case one(Pair)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: Wrap = Wrap.one(pair)
    return moved
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum payload",
			src: `enum PairMsg:
    case both(Int, Int)
    case empty

enum Wrap:
    case one(PairMsg)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: Wrap = Wrap.one(msg)
        return moved + right
    case PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_partial_enum_constructor_after_consume.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModulePartialEnumConstructorAfterConsumeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "struct field",
			modelSrc: `module lib.model

pub struct Pair:
    left: Int
    right: Int

pub enum Wrap:
    case one(Pair)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: model.Wrap = model.Wrap.one(pair)
    return moved
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum payload",
			modelSrc: `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty

pub enum Wrap:
    case one(PairMsg)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: model.Wrap = model.Wrap.one(msg)
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipOptionalPayloadConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_optional_payload_consume.tetra")
	src := `func take(raw: consume ptr) -> ptr:
    return raw

func use(value: ptr?) -> Int:
    return 0

func leak(maybe: ptr?) -> Int:
    match maybe:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(maybe)

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'maybe.$elem'")
}

// ---- check_diagnostics_ownership_borrow_function_typed_test.go ----

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedParameterCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		typeSrc  string
		callback string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(BufBox) -> Int",
			call:     "return cb(BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of callback 'cb'"),
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(consume BufBox) -> Int",
			call:     "let box: BufBox = BufBox(buf: x)\n    return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(inout BufBox) -> Int",
			call:     "var box: BufBox = BufBox(buf: x)\n    return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(BufMsg) -> Int",
			call:     "return cb(BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of callback 'cb'"),
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(consume BufMsg) -> Int",
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(inout BufMsg) -> Int",
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_slice_aggregate_function_typed_"+tt.name+".tetra",
			)
			src := tt.typeSrc + `
func caller(` + tt.callback + `, x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedStructFieldCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		typeSrc  string
		field    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field: "cb: fn(BufBox) -> Int",
			call:  "return h.cb(BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field: "cb: fn(consume BufBox) -> Int",
			call:  "let box: BufBox = BufBox(buf: x)\n    return h.cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field: "cb: fn(inout BufBox) -> Int",
			call:  "var box: BufBox = BufBox(buf: x)\n    return h.cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field: "cb: fn(BufMsg) -> Int",
			call:  "return h.cb(BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field: "cb: fn(consume BufMsg) -> Int",
			call:  "let msg: BufMsg = BufMsg.send(x)\n    return h.cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field: "cb: fn(inout BufMsg) -> Int",
			call:  "var msg: BufMsg = BufMsg.send(x)\n    return h.cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed struct field call 'h.cb'"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_slice_aggregate_function_typed_field_"+tt.name+".tetra",
			)
			src := tt.typeSrc + `
struct Handler:
    ` + tt.field + `

func caller(h: Handler, x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateFunctionTypedStructFieldCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(BufBox) -> Int
`,
			appCall: "return h.cb(callbacks.BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "struct-consume",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(consume BufBox) -> Int
`,
			appCall: "let box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    return h.cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "struct-inout",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(inout BufBox) -> Int
`,
			appCall: "var box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    return h.cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-owned",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(BufMsg) -> Int
`,
			appCall: "return h.cb(callbacks.BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "enum-consume",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(consume BufMsg) -> Int
`,
			appCall: "let msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    return h.cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-inout",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(inout BufMsg) -> Int
`,
			appCall: "var msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    return h.cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed struct field call 'h.cb'"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedEnumPayloadCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		typeSrc  string
		payload  string
		setup    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload: "case some(fn(BufBox) -> Int)",
			call:    "return cb(BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload: "case some(fn(consume BufBox) -> Int)",
			setup:   "let box: BufBox = BufBox(buf: x)\n    ",
			call:    "return cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload: "case some(fn(inout BufBox) -> Int)",
			setup:   "var box: BufBox = BufBox(buf: x)\n    ",
			call:    "return cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload: "case some(fn(BufMsg) -> Int)",
			call:    "return cb(BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload: "case some(fn(consume BufMsg) -> Int)",
			setup:   "let msg: BufMsg = BufMsg.send(x)\n    ",
			call:    "return cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload: "case some(fn(inout BufMsg) -> Int)",
			setup:   "var msg: BufMsg = BufMsg.send(x)\n    ",
			call:    "return cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed enum payload call 'cb'"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_slice_aggregate_function_typed_enum_payload_"+tt.name+".tetra",
			)
			src := tt.typeSrc + `
enum Choice:
    ` + tt.payload + `
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    ` + tt.setup + `match choice:
    case Choice.some(cb):
        ` + tt.call + `
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateFunctionTypedEnumPayloadCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		setup    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(BufBox) -> Int)
    case empty
`,
			call: "return cb(callbacks.BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "struct-consume",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(consume BufBox) -> Int)
    case empty
`,
			setup: "let box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    ",
			call:  "return cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "struct-inout",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(inout BufBox) -> Int)
    case empty
`,
			setup: "var box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    ",
			call:  "return cb(box)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-owned",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(BufMsg) -> Int)
    case empty
`,
			call: "return cb(callbacks.BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "enum-consume",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(consume BufMsg) -> Int)
    case empty
`,
			setup: "let msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    ",
			call:  "return cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-inout",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(inout BufMsg) -> Int)
    case empty
`,
			setup: "var msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    ",
			call:  "return cb(msg)",
			wantText: ("borrowed value derived from 'x' cannot be passed as inout " +
				"to function-typed enum payload call 'cb'"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, x: borrow []u8) -> Int:
    `+tt.setup+`match choice:
    case callbacks.Choice.some(cb):
        `+tt.call+`
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "value-owned",
			src: `
func caller(cb: fn(ptr?) -> Int, maybe: borrow ptr?) -> Int:
    return cb(maybe)

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of callback 'cb'"),
		},
		{
			name: "value-consume",
			src: `
func caller(cb: fn(consume ptr?) -> Int, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by callback 'cb'",
		},
		{
			name: "value-inout",
			src: `
func caller(cb: fn(inout ptr?) -> Int, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "field-owned",
			src: `
struct Handler:
    cb: fn(ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "field-consume",
			src: `
struct Handler:
    cb: fn(consume ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "field-inout",
			src: `
struct Handler:
    cb: fn(inout ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed as " +
				"inout to function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-payload-owned",
			src: `
enum Choice:
    case some(fn(ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    match choice:
    case Choice.some(cb):
        return cb(maybe)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "enum-payload-consume",
			src: `
enum Choice:
    case some(fn(consume ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    match choice:
    case Choice.some(cb):
        return cb(alias)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-payload-inout",
			src: `
enum Choice:
    case some(fn(inout ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    match choice:
    case Choice.some(cb):
        return cb(alias)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed as " +
				"inout to function-typed enum payload call 'cb'"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_optional_ptr_callback_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		appSrc   string
		wantText string
	}{
		{
			name: "field-owned",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of function-typed struct field call " +
				"'h.cb'"),
		},
		{
			name: "field-consume",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(consume ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)
`,
			wantText: ("borrowed value derived from 'maybe' cannot be consumed by " +
				"function-typed struct field call 'h.cb'"),
		},
		{
			name: "field-inout",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(inout ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed as " +
				"inout to function-typed struct field call 'h.cb'"),
		},
		{
			name: "enum-payload-owned",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    match choice:
    case callbacks.Choice.some(cb):
        return cb(maybe)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of function-typed enum payload call " +
				"'cb'"),
		},
		{
			name: "enum-payload-consume",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(consume ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be consumed by " +
				"function-typed enum payload call 'cb'"),
		},
		{
			name: "enum-payload-inout",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(inout ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: ("borrowed value derived from 'maybe' cannot be passed as " +
				"inout to function-typed enum payload call 'cb'"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

`+tt.appSrc+`
func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCapturedFunctionTypedLocalOwnershipAliasCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "borrow-inout-alias",
			src: `
func main() -> Int:
    var a: Int = 1
    let bias: Int = 0
    let cb: fn(borrow Int, inout Int) -> Int = fn(read: borrow Int, write: inout Int) -> Int:
        write = write + read + bias
        return write
    return cb(a, a)
`,
			wantText: "inout argument 'a' aliases borrowed argument in function-typed callback 'cb'",
		},
		{
			name: "consume-inout-alias",
			src: `
func main() -> Int:
    var a: Int = 1
    let bias: Int = 0
    let cb: fn(consume Int, inout Int) -> Int = fn(taken: consume Int, write: inout Int) -> Int:
        write = write + taken + bias
        return write
    return cb(a, a)
`,
			wantText: "inout argument 'a' aliases consumed argument in function-typed callback 'cb'",
		},
		{
			name: "use-after-consume",
			src: `
func main() -> Int:
    let value: Int = 1
    let bias: Int = 0
    let cb: fn(consume Int) -> Int = fn(taken: consume Int) -> Int:
        return taken + bias
    let moved: Int = cb(value)
    return value + moved
`,
			wantText: "cannot use consumed value 'value'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_captured_function_typed_local_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

// ---- check_diagnostics_ownership_borrow_payload_test.go ----

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrEnumPayloadCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(raw: ptr) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'msg' cannot be passed to " +
				"non-borrow parameter 1 of 'sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_enum_payload_"+tt.name+"_call.tetra")
			src := `enum PtrMsg:
    case raw(ptr)
    case empty

` + tt.sinkSrc + `
func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        return sink(raw)
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrEnumPayloadCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(raw: ptr) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'msg' cannot be passed to " +
				"non-borrow parameter 1 of 'app.main.sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed as inout to 'app.main.sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(msg: borrow model.PtrMsg) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        return sink(raw)
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership_ptr_optional_payload_owned_call.tetra")
	src := `func sink(raw: ptr) -> Int:
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(
		t,
		srcPath,
		"borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
	)
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub func sink(raw: ptr) -> Int:
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(
		t,
		srcPath,
		("borrowed value derived from 'maybe' cannot be passed to " +
			"non-borrow parameter 1 of 'lib.sink.sink'"),
	)
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "consume",
			src: `func sink(raw: consume ptr) -> Int:
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_ptr_optional_payload_"+tt.name+"_call.tetra",
			)
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "consume",
			sinkSrc: `module lib.sink

pub func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "inout",
			sinkSrc: `module lib.sink

pub func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
	}{
		{
			name: "owned",
			src: `func sink(raw: []u8) -> Int:
    return 0

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of 'sink'"),
		},
		{
			name: "consume",
			src: `func sink(raw: consume []u8) -> Int:
    return 0

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name: "inout-call",
			src: `func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
		{
			name: "inout-assignment",
			src: `func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_optional_payload_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		leakSrc  string
		wantCode string
		wantText string
		wantFile string
	}{
		{
			name: "owned",
			sinkSrc: `module lib.sink

pub func sink(raw: []u8) -> Int:
    return 0
`,
			leakSrc: `module app.main
import lib.sink as sinker

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: ("borrowed value derived from 'maybe' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.sink.sink'"),
			wantFile: "src/app/main.t4",
		},
		{
			name: "consume",
			sinkSrc: `module lib.sink

pub func sink(raw: consume []u8) -> Int:
    return 0
`,
			leakSrc: `module app.main
import lib.sink as sinker

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'",
			wantFile: "src/app/main.t4",
		},
		{
			name: "inout-call",
			sinkSrc: `module lib.sink

pub func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0
`,
			leakSrc: `module app.main
import lib.sink as sinker

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.sink'",
			wantFile: "src/app/main.t4",
		},
		{
			name:    "inout-assignment",
			sinkSrc: "",
			leakSrc: `module lib.leaks

pub func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
			wantFile: "src/lib/leaks.t4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			if tt.sinkSrc != "" {
				writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.sinkSrc)
				writeCLIProjectFile(t, dir, "src/app/main.t4", tt.leakSrc)
			} else {
				writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.leakSrc)
				writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			}
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			wantPath := filepath.Join(dir, filepath.FromSlash(tt.wantFile))
			assertCLIJSONDiagnosticForPath(t, srcPath, wantPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalAssignmentConsumeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership_ptr_optional_assignment_consume.tetra")
	src := `func sink(value: consume ptr?) -> Int:
    return 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(
		t,
		srcPath,
		"borrowed value derived from 'x' cannot be consumed by 'sink'",
	)
}

// ---- check_diagnostics_ownership_borrow_ptr_aggregate_test.go ----

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: PtrBox) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'box' cannot be passed to " +
				"non-borrow parameter 1 of 'sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout PtrBox) -> Int:
    value = PtrBox(raw: 0)
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_aggregate_"+tt.name+"_call.tetra")
			src := `struct PtrBox:
    raw: ptr

` + tt.sinkSrc + `
func leak(box: borrow PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: model.PtrBox) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'box' cannot be passed to " +
				"non-borrow parameter 1 of 'app.main.sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume model.PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout model.PtrBox) -> Int:
    value = model.PtrBox(raw: 0)
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'app.main.sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(box: borrow model.PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		mainCall string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `pub func sink(value: PtrBox) -> Int:
    return 0
`,
			mainCall: "return sinker.sink(sinker.PtrBox(raw: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.sink.sink'"),
		},
		{
			name: "consume",
			sinkSrc: `pub func take(value: consume PtrBox) -> Int:
    return 0
`,
			mainCall: "let box: sinker.PtrBox = sinker.PtrBox(raw: x)\n    return sinker.take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.take'",
		},
		{
			name: "inout",
			sinkSrc: `pub func mutate(value: inout PtrBox) -> Int:
    value = value
    return 0
`,
			mainCall: "var box: sinker.PtrBox = sinker.PtrBox(raw: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub struct PtrBox:
    raw: ptr

`+tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    `+tt.mainCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrNestedAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		mainCall string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `pub func sink(value: OuterBox) -> Int:
    return 0
`,
			mainCall: "return sinker.sink(sinker.OuterBox(box: sinker.PtrBox(raw: x)))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.sink.sink'"),
		},
		{
			name: "consume",
			sinkSrc: `pub func take(value: consume OuterBox) -> Int:
    return 0
`,
			mainCall: ("let outer: sinker.OuterBox = sinker.OuterBox(box: " +
				"sinker.PtrBox(raw: x))\n    return sinker.take(outer)"),
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.take'",
		},
		{
			name: "inout",
			sinkSrc: `pub func mutate(value: inout OuterBox) -> Int:
    value = value
    return 0
`,
			mainCall: ("var outer: sinker.OuterBox = sinker.OuterBox(box: " +
				"sinker.PtrBox(raw: x))\n    return sinker.mutate(outer)"),
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

`+tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    `+tt.mainCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrNestedAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: OuterBox) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'outer' cannot be passed to " +
				"non-borrow parameter 1 of 'sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout OuterBox) -> Int:
    value = OuterBox(box: PtrBox(raw: 0))
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_ptr_nested_aggregate_"+tt.name+"_call.tetra",
			)
			src := `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

` + tt.sinkSrc + `
func leak(outer: borrow OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrNestedAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: model.OuterBox) -> Int:
    return 0
`,
			wantText: ("borrowed value derived from 'outer' cannot be passed to " +
				"non-borrow parameter 1 of 'app.main.sink'"),
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume model.OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout model.OuterBox) -> Int:
    value = model.OuterBox(box: model.PtrBox(raw: 0))
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'app.main.sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(outer: borrow model.OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

// ---- check_diagnostics_ownership_borrow_slice_call_test.go ----

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func sink(value: BufBox) -> Int:
    return 0
`,
			call:     "return sink(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func sink(value: consume BufBox) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink(value: BufMsg) -> Int:
    return 0
`,
			call:     "return sink(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink(value: consume BufMsg) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_call_"+tt.name+".tetra")
			src := tt.typeSrc + "\n" + tt.callee + `
func caller(x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		typeSrc  string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func sink(value: BufBox) -> Int:
    return 0
`,
			call: "return sink(BufBox(buf: x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.leaks.sink'"),
		},
		{
			name: "struct-consume",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func sink(value: consume BufBox) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leaks.sink'",
		},
		{
			name: "struct-inout",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.leaks.mutate'",
		},
		{
			name: "enum-owned",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func sink(value: BufMsg) -> Int:
    return 0
`,
			call: "return sink(BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to " +
				"non-borrow parameter 1 of 'lib.leaks.sink'"),
		},
		{
			name: "enum-consume",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func sink(value: consume BufMsg) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leaks.sink'",
		},
		{
			name: "enum-inout",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.leaks.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

`+tt.typeSrc+`
`+tt.callee+`
pub func caller(x: borrow []u8) -> Int:
    `+tt.call+`
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateGenericCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		typeSrc  string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "return sink(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return take(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_slice_aggregate_generic_call_"+tt.name+".tetra",
			)
			src := tt.typeSrc + "\n" + tt.callee + `
func caller(x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateGenericCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "struct-consume",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "enum-consume",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.take(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrGenericCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "owned",
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "return sink(maybe)",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let alias: ptr? = maybe\n    return take(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be consumed",
		},
		{
			name: "inout",
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var alias: ptr? = maybe\n    return mutate(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(
				dir,
				"bad_ownership_optional_ptr_generic_call_"+tt.name+".tetra",
			)
			src := tt.callee + `
func caller(maybe: borrow ptr?) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrGenericCallEscapeCodes(
	t *testing.T,
) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.sink

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(maybe)",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			libSrc: `module lib.sink

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let alias: ptr? = maybe\n    return sinker.take(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be consumed",
		},
		{
			name: "inout",
			libSrc: `module lib.sink

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var alias: ptr? = maybe\n    return sinker.mutate(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

// ---- check_diagnostics_resource_actor_test.go ----

func TestCheckCommandJSONDiagnosticsForResourceUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_free.tetra")
	src := `func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        free(isl)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'isl'")
}

func TestCheckCommandJSONDiagnosticsForResourceStructFieldAliasUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_struct_field_alias_free.tetra")
	src := `struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceStructFieldAliasUseAfterFreeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct IslandBox:
    handle: island
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.IslandBox = resources.IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForResourceEnumPayloadAliasUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_enum_payload_alias_free.tetra")
	src := `enum MoveMsg:
    case take(island)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        match msg:
        case MoveMsg.take(other):
            let alias: island = other
            free(other)
            free(alias)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceEnumPayloadAliasUseAfterFreeCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum MoveMsg:
    case take(island)

pub func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: resources.MoveMsg = resources.MoveMsg.take(core.island_new(16))
        let other: island = resources.unwrap(msg)
        match msg:
        case resources.MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForResourceOptionalPayloadFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_optional_payload_free.tetra")
	src := `func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'maybe.$elem'")
}

func TestCheckCommandJSONDiagnosticsForResourceOptionalWrapperAliasUseAfterFreeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "struct",
			src: `struct MaybeBox:
    maybe: island?

func pass(box: MaybeBox) -> MaybeBox:
    return box

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: MaybeBox = MaybeBox(maybe: isl)
        let returned: MaybeBox = pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
		},
		{
			name: "enum",
			src: `enum MaybeEnvelope:
    case wrap(island?)
    case empty

func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: MaybeEnvelope = MaybeEnvelope.wrap(isl)
        let returned: MaybeEnvelope = pass(msg)
        match returned:
        case MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_resource_optional_wrapper_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceOptionalWrapperAliasUseAfterFreeCodes(
	t *testing.T,
) {
	tests := []struct {
		name   string
		libSrc string
		appSrc string
	}{
		{
			name: "struct",
			libSrc: `module lib.resources

pub struct MaybeBox:
    maybe: island?

pub func pass(box: MaybeBox) -> MaybeBox:
    return box
`,
			appSrc: `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: resources.MaybeBox = resources.MaybeBox(maybe: isl)
        let returned: resources.MaybeBox = resources.pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
		},
		{
			name: "enum",
			libSrc: `module lib.resources

pub enum MaybeEnvelope:
    case wrap(island?)
    case empty

pub func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg
`,
			appSrc: `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: resources.MaybeEnvelope = resources.MaybeEnvelope.wrap(isl)
        let returned: resources.MaybeEnvelope = resources.pass(msg)
        match returned:
        case resources.MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case resources.MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/resources.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.appSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForResourceDoubleJoinCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_join.tetra")
	src := `func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'task'")
}

func TestCheckCommandJSONDiagnosticsForTaskGroupUseAfterCloseCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_group_close.tetra")
	src := `func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(group)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'group'")
}

func TestCheckCommandJSONDiagnosticsForResourceAmbiguousProvenanceCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_provenance.tetra")
	src := `struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        var box: IslandBox = IslandBox(handle: left)
        if 1:
            box = IslandBox(handle: right)
        free(box.handle)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "ambiguous resource provenance for 'box.handle'")
}

func TestCheckCommandJSONDiagnosticsForIslandTransferNonLocalPayloadCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_island_transfer_payload.tetra")
	src := `enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        return core.send_typed(peer, MoveMsg.take(core.island_new(16)))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "island transfer payload must be a local value")
}

func TestCheckCommandJSONDiagnosticsForActorUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_use_after_transfer.tetra")
	src := `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = take_actor(peer)
    return core.send(peer, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
}

func TestCheckCommandJSONDiagnosticsForActorBranchConsumeReuseCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_branch_consume_reuse.tetra")
	src := `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(flag: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    if flag:
        let _: Int = take_actor(peer)
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
}

func TestCheckCommandJSONDiagnosticsForActorMatchLoopConsumeReuseCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "match",
			src: `enum Choice:
    case take
    case keep

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(choice: Choice) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    match choice:
    case Choice.take:
        let taken: Int = take_actor(peer)
    case Choice.keep:
        let kept: Int = 0
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(Choice.take)
`,
		},
		{
			name: "loop",
			src: `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(limit: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    var i: Int = 0
    while i < limit:
        let _: Int = take_actor(peer)
        i = i + 1
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_actor_"+tt.name+"_consume_reuse.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForTaskUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_use_after_transfer.tetra")
	src := `func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = take_task(task)
    return value + core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'task'")
}

func TestCheckCommandJSONDiagnosticsForActorStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_struct_field_alias_transfer.tetra")
	src := `struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let _: Int = take_actor(peer)
    return core.send(box.peer, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'box.peer'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleActorStructFieldAliasUseAfterTransferCode(
	t *testing.T,
) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct ActorBox:
    peer: actor

pub func unwrap(box: ActorBox) -> actor:
    return box.peer
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.ActorBox = resources.ActorBox(peer: peer)
    let other: actor = resources.unwrap(box)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForGenericActorStructFieldAliasUseAfterTransferCodes(
	t *testing.T,
) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_actor_generic_struct_field_alias_transfer.tetra")
		src := `struct Box<T>:
    value: T

func worker() -> Int:
    return 0

func pass_actor(box: Box<actor>) -> Box<actor>:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: Box<actor> = Box<actor>{value: peer}
    let returned: Box<actor> = pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'returned.value'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_actor(box: Box<actor>) -> Box<actor>:
    return box
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.Box<actor> = resources.Box<actor>{value: peer}
    let returned: resources.Box<actor> = resources.pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'returned.value'")
	})
}

func TestCheckCommandJSONDiagnosticsForGenericResourceAliasFinalizationCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_generic_struct_alias_join.tetra")
		src := `struct Box<T>:
    value: T

func worker() -> Int:
    return 7

func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: Box<task.i32> = Box<task.i32>{value: task}
    let returned: Box<task.i32> = pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.value'")
	})

	t.Run("cross module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.Box<task.i32> = resources.Box<task.i32>{value: task}
    let returned: resources.Box<task.i32> = resources.pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.value'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_generic_struct_alias_close.tetra")
		src := `struct Box<T>:
    value: T

func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: Box<task.group> = Box<task.group>{value: group}
    let returned: Box<task.group> = pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'returned.value'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.Box<task.group> = resources.Box<task.group>{value: group}
    let returned: resources.Box<task.group> = resources.pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'returned.value'")
	})

	t.Run("same module island", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_island_generic_struct_alias_free.tetra")
		src := `struct Box<T>:
    value: T

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: Box<island> = Box<island>{value: core.island_new(16)}
        let alias: Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias.value'")
	})

	t.Run("cross module island", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.Box<island> = resources.Box<island>{value: core.island_new(16)}
        let alias: resources.Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias.value'")
	})
}

func TestCheckCommandJSONDiagnosticsForTransitiveResourceAliasFinalizationCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_transitive_alias_join.tetra")
		src := `func worker() -> Int:
    return 7

func alias_one(task: task.i32) -> task.i32:
    return task

func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(task: task.i32) -> task.i32:
    return task

pub func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_transitive_alias_close.tetra")
		src := `func alias_one(group: task.group) -> task.group:
    return group

func alias_two(group: task.group) -> task.group:
    return alias_one(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(group: task.group) -> task.group:
    return group

pub func alias_two(group: task.group) -> task.group:
    return alias_one(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = resources.alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("same module island", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_island_transitive_alias_free.tetra")
		src := `func alias_one(isl: island) -> island:
    return isl

func alias_two(isl: island) -> island:
    return alias_one(isl)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
	})

	t.Run("cross module island", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(isl: island) -> island:
    return isl

pub func alias_two(isl: island) -> island:
    return alias_one(isl)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = resources.alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForEnumConstructorReturnResourceAliasCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_enum_constructor_return_alias_join.tetra")
		src := `enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: TaskMsg = wrap(task)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_enum_constructor_return_alias_close.tetra")
		src := `enum GroupMsg:
    case wrap(task.group)

func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: GroupMsg = wrap(group)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: resources.GroupMsg = resources.wrap(group)
    match returned:
    case resources.GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

}

// ---- check_test.go ----

func TestCheckCommandUsesDefaultMainT4(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "main.t4"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Checked: main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandDiscoversCapsuleT4ProjectEntryAndSourceRoots(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"

    sources:
        src
        ui

    targets:
        linux

    allow:
        ui

    policy:
        unsafe deny
        reproducible required
`)
	writeCLIProjectFile(
		t,
		dir,
		"src/app/main.t4",
		("module app.main\nimport components.counter as counter\nfunc " +
			"main() -> Int:\n    return counter.value()\n"),
	)
	writeCLIProjectFile(
		t,
		dir,
		"ui/components/counter.t4",
		"module components.counter\nfunc value() -> Int:\n    return 42\n",
	)

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
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandExplicitProjectDirectoryUsesCapsuleEntry(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
        ui
`)
	writeCLIProjectFile(
		t,
		dir,
		"src/app/main.t4",
		("module app.main\nimport components.counter as counter\nfunc " +
			"main() -> Int:\n    return counter.value()\n"),
	)
	writeCLIProjectFile(
		t,
		dir,
		"ui/components/counter.t4",
		"module components.counter\nfunc value() -> Int:\n    return 42\n",
	)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandResolvesLocalCapsuleDependencyImport(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	)

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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
}
func TestCheckCommandValidatesPresentTetraLockAgainstCapsuleGraph(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	)

	lockPath := filepath.Join(dir, "App", "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"eco",
			"verify",
			"--lock",
			lockPath,
			filepath.Join(dir, "App", "Capsule.t4"),
			filepath.Join(dir, "Math", "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.2.0"
    sources:
        src
`)

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

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf(
			"expected check failure for stale Tetra.lock, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "Tetra.lock") ||
		!strings.Contains(stderr.String(), "version mismatch") {
		t.Fatalf("stderr = %q, want Tetra.lock version mismatch", stderr.String())
	}
	if !strings.Contains(stderr.String(), "tetra project sync") {
		t.Fatalf("stderr = %q, want project sync repair hint", stderr.String())
	}
}

func TestCheckCommandSucceedsWithoutOutputFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "app")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Checked: "+srcPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("check should not create %s, stat err=%v", outPath, err)
	}
}

func TestTargetAwareCommandsRejectInvalidTargetConsistently(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "build",
			args: []string{"build", "--target", "not-a-target", "examples/flow/flow_hello.tetra"},
		},
		{
			name: "run",
			args: []string{"run", "--target", "not-a-target", "examples/flow/flow_hello.tetra"},
		},
		{
			name: "test",
			args: []string{
				"test",
				"--target",
				"not-a-target",
				"examples/smoke/basic/tooling_tests.tetra",
			},
		},
		{name: "smoke", args: []string{"smoke", "--target", "not-a-target", "--run=false"}},
		{name: "smoke list", args: []string{"smoke", "--list", "--target", "not-a-target"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := runCLI(tt.args, &bytes.Buffer{}, &stderr)
			if code != 2 {
				t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
			}
			for _, want := range []string{"unsupported target: not-a-target", (("supported targets: " +
				"linux-x64, windows-x64, macos-x64, ") +
				"wasm32-wasi, wasm32-web"), "build-only targets: linux-x86, linux-x32"} {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr missing %q: %q", want, stderr.String())
				}
			}
		})
	}
}

func TestCheckCommandReportsMissingDefaultMain(t *testing.T) {
	dir := t.TempDir()
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

	var stderr bytes.Buffer
	code := runCLI([]string{"check"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "main.t4") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// ---- clean_test.go ----

func TestCleanCommandRemovesCacheDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.MkdirAll(filepath.Join(dir, path, "nested"), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(
			filepath.Join(dir, path, "nested", "entry"),
			[]byte("cache"),
			0o644,
		); err != nil {
			t.Fatalf("write cache entry: %v", err)
		}
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

	var stdout bytes.Buffer
	code := runCLI([]string{"clean"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("clean exit code = %d, stdout=%q", code, stdout.String())
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf(
				"cache dir %s still exists or stat failed with non-missing error: %v",
				path,
				err,
			)
		}
	}
	if !strings.Contains(stdout.String(), "Cleaned Tetra cache") {
		t.Fatalf("clean stdout = %q", stdout.String())
	}
}

func TestCleanCommandTargetRemovesOnlyRequestedTargetCache(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		filepath.Join(".tetra_cache", "linux-x64", "entry"),
		filepath.Join(".tetra_cache", "windows-x64", "entry"),
		filepath.Join("tetra_cache", "linux-x64", "entry"),
		filepath.Join("tetra_cache", "windows-x64", "entry"),
	} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), []byte("cache"), 0o644); err != nil {
			t.Fatalf("write cache entry: %v", err)
		}
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"clean", "--target", "linux-x64"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"clean --target exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, path := range []string{filepath.Join(
		".tetra_cache",
		"linux-x64",
	), filepath.Join(
		"tetra_cache",
		"linux-x64",
	)} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf(
				"target cache dir %s still exists or stat failed with non-missing error: %v",
				path,
				err,
			)
		}
	}
	for _, path := range []string{filepath.Join(
		".tetra_cache",
		"windows-x64",
		"entry",
	), filepath.Join(
		"tetra_cache",
		"windows-x64",
		"entry",
	)} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("non-target cache entry %s should remain: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "linux-x64") {
		t.Fatalf("clean stdout should name target: %q", stdout.String())
	}
}

// ---- cli_contract_test.go ----

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("version exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), compiler.Version()) {
		t.Fatalf("version output = %q, want compiler version", stdout.String())
	}
}

func TestCLIContractDocumentedCommandsHaveHelpAndInvalidArgBehavior(t *testing.T) {
	commands := documentedCLICommands(t)
	if len(commands) == 0 {
		t.Fatal("no documented CLI commands found")
	}
	for _, command := range commands {
		t.Run(command+"_help", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{command, "--help"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf(
					"%s --help exit code = %d, stdout=%q stderr=%q",
					command,
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), command) &&
				!strings.Contains(strings.ToLower(combined), "usage") {
				t.Fatalf(
					"%s --help output does not describe the command: stdout=%q stderr=%q",
					command,
					stdout.String(),
					stderr.String(),
				)
			}
		})
		t.Run(command+"_invalid_arg", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{command, "--definitely-invalid"}, &stdout, &stderr)
			if code != 2 {
				t.Fatalf(
					"%s invalid arg exit code = %d, stdout=%q stderr=%q",
					command,
					code,
					stdout.String(),
					stderr.String(),
				)
			}
		})
	}
}

func documentedCLICommands(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "..", "docs", "spec", "policy", "cli_contracts.md"),
	)
	if err != nil {
		t.Fatalf("read cli contracts: %v", err)
	}
	seen := map[string]bool{}
	var commands []string
	inCommandRecords := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "Command records:" {
			inCommandRecords = true
			continue
		}
		if strings.HasPrefix(line, "The `lsp --stdio` contract") {
			inCommandRecords = false
		}
		if !inCommandRecords {
			continue
		}
		command, ok := documentedCLICommandName(line)
		if !ok {
			continue
		}
		if !seen[command] {
			seen[command] = true
			commands = append(commands, command)
		}
	}
	return commands
}

func documentedCLICommandName(line string) (string, bool) {
	if strings.HasPrefix(line, "| `") {
		return validDocumentedCLICommand(strings.TrimPrefix(line, "| `"))
	}
	if strings.HasPrefix(line, "- `") {
		return validDocumentedCLICommand(strings.TrimPrefix(line, "- `"))
	}
	return "", false
}

func validDocumentedCLICommand(rest string) (string, bool) {
	command, _, ok := strings.Cut(rest, "`")
	if !ok || command == "tetra" || strings.Contains(command, " ") || command == "" {
		return "", false
	}
	if command[0] < 'a' || command[0] > 'z' {
		return "", false
	}
	return command, true
}

// ---- doc_test.go ----

func TestDocCommandWritesAPIDocsToStdout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	if err := os.WriteFile(
		srcPath,
		[]byte("func answer() -> Int:\n    return 42\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "# Tetra API Docs") ||
		!strings.Contains(stdout.String(), "`func answer() -> i32`") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDocCommandDiscoversCapsuleProjectSources(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(
		t,
		dir,
		"src/app/main.t4",
		"module app.main\nfunc answer() -> Int:\n    return 42\n",
	)

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
	code := runCLI([]string{"doc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "## app.main") ||
		!strings.Contains(stdout.String(), "`func answer() -> i32`") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDocCommandWritesAPIDocsToFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	outPath := filepath.Join(dir, "docs", "api.md")
	if err := os.WriteFile(
		srcPath,
		[]byte("func answer() -> Int:\n    return 42\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	if !strings.Contains(string(raw), "`func answer() -> i32`") {
		t.Fatalf("docs = %s", raw)
	}
}

func TestDocCommandGeneratedOutputPassesAPIValidator(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	outPath := filepath.Join(dir, "api.md")
	src := `module docs.api

func answer() -> Int:
    return 42

test "answer":
    expect answer() == 42
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	cmd := exec.Command("go", "run", "./tools/cmd/validate-api-docs", "--docs", outPath)
	cmd.Dir = filepath.Join("..", "..", "..")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate-api-docs failed: %v\n%s", err, out)
	}
}

func TestDocCommandJSONDiagnostics(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"doc", "--diagnostics=json", "/tmp/does-not-exist.tetra"},
		1,
	)
	if diag.Code != "TETRA0001" || diag.Severity != "error" ||
		!strings.Contains(diag.Message, "no such file or directory") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

// ---- doctor_test.go ----

func TestDoctorCommandJSON(t *testing.T) {
	var report struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	rawReport := runCLIJSONStdout(t, []string{"doctor", "--format=json"}, 0, &report)
	if report.Status != "pass" {
		t.Fatalf("doctor status = %q, report=%s", report.Status, rawReport)
	}
	var sawVersion, sawRuntime, sawManifest, sawManifestVersion bool
	var sawManifestSurface, sawSmokeSources, sawRuntimeExports bool
	var sawTargetMetadata, sawToolingCommands bool
	var sawBuildOnlyTargets bool
	for _, check := range report.Checks {
		if check.Name == "version" && check.Status == "pass" {
			sawVersion = true
		}
		if check.Name == "build-only targets" && check.Status == "pass" {
			sawBuildOnlyTargets = true
		}
		if check.Name == "__rt/actors_sysv.tetra" && check.Status == "pass" {
			sawRuntime = true
		}
		if check.Name == "docs/generated/manifest.json" && check.Status == "pass" {
			sawManifest = true
		}
		if check.Name == "docs manifest version" && check.Status == "pass" &&
			check.Detail == compiler.Version() {
			sawManifestVersion = true
		}
		if check.Name == "docs manifest surface" && check.Status == "pass" &&
			strings.Contains(check.Detail, "targets") &&
			strings.Contains(check.Detail, "runtime symbols") {
			sawManifestSurface = true
		}
		if check.Name == "smoke sources" && check.Status == "pass" &&
			strings.Contains(check.Detail, "sources") {
			sawSmokeSources = true
		}
		if check.Name == "runtime exports" && check.Status == "pass" &&
			strings.Contains(check.Detail, "symbols") {
			sawRuntimeExports = true
		}
		if check.Name == "target metadata" && check.Status == "pass" &&
			strings.Contains(check.Detail, "7 targets") &&
			strings.Contains(check.Detail, "2 build-only") {
			sawTargetMetadata = true
		}
		if check.Name == "tooling commands" && check.Status == "pass" &&
			strings.Contains(check.Detail, "fmt") &&
			strings.Contains(check.Detail, "test") {
			sawToolingCommands = true
		}
	}
	if !sawVersion || !sawBuildOnlyTargets || !sawRuntime || !sawManifest || !sawManifestVersion ||
		!sawManifestSurface ||
		!sawSmokeSources ||
		!sawRuntimeExports ||
		!sawTargetMetadata ||
		!sawToolingCommands {
		t.Fatalf("doctor missing expected checks: %#v", report.Checks)
	}
}

func TestDoctorCommandTOON(t *testing.T) {
	var report struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	rawReport := runCLITOONStdout(t, []string{"doctor", "--format=toon"}, 0, &report)
	if !strings.Contains(rawReport, "checks[") || report.Status != "pass" {
		t.Fatalf("doctor TOON report incomplete: raw=%s report=%#v", rawReport, report)
	}
	var sawVersion bool
	for _, check := range report.Checks {
		if check.Name == "version" && check.Status == "pass" {
			sawVersion = true
		}
	}
	if !sawVersion {
		t.Fatalf("doctor TOON report missing version check: %#v", report.Checks)
	}
}

func TestTargetMetadataCheck(t *testing.T) {
	t.Run("wasi runner available", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			if name == "wasmtime" {
				return "/usr/bin/wasmtime", nil
			}
			if name == "node" {
				return "/usr/bin/node", nil
			}
			if name == "chromium" {
				return "/usr/bin/chromium", nil
			}
			return "", exec.ErrNotFound
		})
		defer restore()
		restoreHost := stubLinuxX32HostSupport(false)
		defer restoreHost()

		check := targetMetadataCheck()
		if check.Status != "pass" {
			t.Fatalf("targetMetadataCheck = %#v", check)
		}
		wasi := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-wasi")
		if wasi.BuildOnly || wasi.RunMode != "wasi_runner" || wasi.RunRunner != "wasmtime" ||
			!wasi.RunSupported ||
			wasi.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-wasi target metadata = %#v", wasi)
		}
		web := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-web")
		if web.BuildOnly || web.RunMode != "web_runner" || !web.RunSupported ||
			web.RunRunner == "" ||
			web.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-web target metadata = %#v", web)
		}
		x32 := targetReportEntryForTest(t, buildTargetReportEntries(), "linux-x32")
		if !x32.BuildOnly || x32.RunMode != "host_probed" || x32.PointerWidthBits != 32 ||
			x32.RegisterWidthBits != 64 ||
			!strings.Contains(x32.UnsupportedReason, "host-probed source run/test execution") ||
			!strings.Contains(x32.UnsupportedReason, "Linux kernel supports the x32 ABI") {
			t.Fatalf("linux-x32 target metadata = %#v", x32)
		} else if x32.RunSupported {
			if x32.RunUnsupportedReason != "" {
				t.Fatalf("linux-x32 supported host-probed metadata = %#v", x32)
			}
		} else {
			requireLinuxX32HostUnsupportedReason(t, x32.RunUnsupportedReason)
		}
	})

	t.Run("wasi runner missing", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			return "", exec.ErrNotFound
		})
		defer restore()

		check := targetMetadataCheck()
		if check.Status != "pass" {
			t.Fatalf("targetMetadataCheck = %#v", check)
		}
		wasi := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-wasi")
		if wasi.BuildOnly || wasi.RunMode != "wasi_runner" || wasi.RunRunner != "" ||
			wasi.RunSupported ||
			!strings.Contains(wasi.RunUnsupportedReason, "missing WASI runner") {
			t.Fatalf("wasm32-wasi target metadata without runner = %#v", wasi)
		}
		web := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-web")
		if web.BuildOnly || web.RunMode != "web_runner" || web.RunSupported ||
			!strings.Contains(web.RunUnsupportedReason, "browser runner unavailable") {
			t.Fatalf("wasm32-web target metadata without runner = %#v", web)
		}
	})
}

func targetReportEntryForTest(
	t *testing.T,
	entries []targetReportEntry,
	triple string,
) targetReportEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Triple == triple {
			return entry
		}
	}
	t.Fatalf("missing target metadata for %s in %#v", triple, entries)
	return targetReportEntry{}
}

func TestDoctorCommandProjectJSON(t *testing.T) {
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
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var report struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	rawReport := runCLIJSONStdout(t, []string{"doctor", "--format=json", dir}, 0, &report)
	if report.Status != "pass" {
		t.Fatalf("doctor status = %q report=%s", report.Status, rawReport)
	}
	var sawCapsule, sawEntry, sawRoots, sawLockSync bool
	for _, check := range report.Checks {
		if check.Name == "project capsule" && check.Status == "pass" &&
			strings.Contains(filepath.ToSlash(check.Detail), "Capsule.t4") {
			sawCapsule = true
		}
		if check.Name == "project entry" && check.Status == "pass" &&
			strings.Contains(filepath.ToSlash(check.Detail), "src/main.t4") {
			sawEntry = true
		}
		if check.Name == "project source roots" && check.Status == "pass" &&
			strings.Contains(check.Detail, "src") {
			sawRoots = true
		}
		if check.Name == "project lock" && check.Status == "pass" &&
			strings.Contains(check.Detail, "tetra project sync") {
			sawLockSync = true
		}
	}
	if !sawCapsule || !sawEntry || !sawRoots || !sawLockSync {
		t.Fatalf("project doctor missing expected checks: %#v", report.Checks)
	}
}

func TestDoctorCommandRejectsUnsupportedFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"doctor", "--format=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("doctor exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDoctorReportFilesystemProbesFailInIncompleteRepo(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	report := buildDoctorReportForRoot(root)
	if report.Status != "fail" {
		t.Fatalf("doctor status = %q, checks=%#v", report.Status, report.Checks)
	}
	requiredFailures := map[string]bool{
		"__rt/actors_sysv.tetra":                false,
		"compiler/selfhostrt/actors_sysv.tetra": false,
		"examples/flow/flow_hello.tetra":        false,
		"docs/generated/manifest.json":          false,
	}
	for _, check := range report.Checks {
		if _, ok := requiredFailures[check.Name]; ok && check.Status == "fail" {
			requiredFailures[check.Name] = true
		}
	}
	for name, saw := range requiredFailures {
		if !saw {
			t.Fatalf("doctor did not fail missing filesystem probe %s: %#v", name, report.Checks)
		}
	}
}

// ---- eco_fuzz_test.go ----

func FuzzParseCapsuleDoesNotPanic(f *testing.F) {
	f.Add(`manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
`)
	f.Add("capsule Broken:\n")
	f.Add("")
	f.Add("manifest \"wrong\"\n")

	f.Fuzz(func(t *testing.T, text string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "Tetra.capsule")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _ = parseCapsule(path)
	})
}

// ---- eco_test.go ----

func TestEcoVerifySingleCapsuleExpandsPathDependenciesIntoTetraLock(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)

	lockPath := filepath.Join(dir, "App", "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"eco", "verify", "--lock", lockPath, filepath.Join(dir, "App", "Capsule.t4")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://app"`) ||
		!strings.Contains(string(raw), `"tetra://math"`) {
		t.Fatalf("lock did not include full path dependency graph:\n%s", string(raw))
	}
}

func TestEcoArtifactsBuildGeneratesDependencyArtifactsLockAndBuildsProject(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    targets:
        linux
`)
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://math 0.1.0 ../Math
`, target))
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	)

	appRoot := filepath.Join(dir, "App")
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"eco",
			"artifacts",
			"build",
			"--target",
			target,
			"--lock",
			lockPath,
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco artifacts build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	interfaceRel := "interfaces/math/core.t4i"
	objectRel := "artifacts/math/core." + target + ".tobj"
	seedRel := "seeds/app-deps.t4s"
	for _, rel := range []string{interfaceRel, objectRel, seedRel, "Tetra.lock"} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected generated %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatalf("read Capsule.t4: %v", err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{
		"artifacts:",
		"interface " + interfaceRel,
		"object " + target + " " + objectRel,
		"seed " + seedRel,
	} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	lockRaw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	for _, want := range []string{
		`"kind": "object"`,
		`"target": "` + target + `"`,
		`"module": "math.core"`,
		`"public_api_hash": "sha256:`,
	} {
		if !strings.Contains(string(lockRaw), want) {
			t.Fatalf("Tetra.lock missing %q:\n%s", want, string(lockRaw))
		}
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
	out := filepath.Join(appRoot, "app")
	code = runCLI([]string{"build", "--target", target, "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
}

func TestEcoArtifactsCheckDetectsStaleInterfaceAndSuggestsRepair(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"eco",
			"artifacts",
			"build",
			"--target",
			target,
			"--lock",
			lockPath,
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco artifacts build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int, c: Int) -> Int:\n    return a + b + c\n",
	)

	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{
			"eco",
			"artifacts",
			"check",
			"--target",
			target,
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf(
			"expected stale artifact failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	combined := stdout.String() + stderr.String()
	for _, want := range []string{
		"stale interface artifact",
		"math.core",
		"tetra eco artifacts build --target " + target,
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf(
				"artifact check output missing %q:\nstdout=%s\nstderr=%s",
				want,
				stdout.String(),
				stderr.String(),
			)
		}
	}
}

func TestEcoArtifactsBuildCheckDryRunDoesNotWriteArtifacts(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"eco",
			"artifacts",
			"build",
			"--check",
			"--target",
			target,
			"--lock",
			lockPath,
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf("expected dry-run to report pending artifacts")
	}
	if !strings.Contains(stdout.String()+stderr.String(), "would generate") {
		t.Fatalf(
			"dry-run output = stdout=%q stderr=%q, want would generate",
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{
		"interfaces/math/core.t4i",
		"artifacts/math/core." + target + ".tobj",
		"seeds/app-deps.t4s",
		"Tetra.lock",
	} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("dry-run unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
}

func TestEcoArtifactsBuildAllTargetsSkipsWASMObjectTargets(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
        wasm32-wasi
    deps:
        tetra://math 0.1.0 ../Math
`, target))

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"eco",
			"artifacts",
			"build",
			"--all-targets",
			"--lock",
			filepath.Join(appRoot, "Tetra.lock"),
			filepath.Join(appRoot, "Capsule.t4"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco artifacts build --all-targets exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(
		filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core."+target+".tobj")),
	); err != nil {
		t.Fatalf("expected native object artifact: %v", err)
	}
	if _, err := os.Stat(
		filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj")),
	); err == nil {
		t.Fatalf("unexpected wasm object artifact")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat wasm object: %v", err)
	}
}

func TestEcoVerifyPackAndUnpack(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	src := `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
`
	if err := os.WriteFile(capsule, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := runCLI([]string{"eco", "verify", capsule}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q", code, stdout.String())
	}
	pkg := filepath.Join(dir, "demo.todex")
	if code := runCLI(
		[]string{"eco", "pack", capsule, "-o", pkg},
		&stdout,
		&bytes.Buffer{},
	); code != 0 {
		t.Fatalf("eco pack exit code = %d, stdout=%q", code, stdout.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	if code := runCLI(
		[]string{"eco", "unpack", pkg, "-C", outDir},
		&stdout,
		&bytes.Buffer{},
	); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q", code, stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.package.json")); err != nil {
		t.Fatalf("expected unpacked package metadata: %v", err)
	}
}

func TestEcoVerifyHelpExitsSuccessfully(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--help"}, &bytes.Buffer{}, &stderr)
	if code != 0 {
		t.Fatalf("eco verify --help exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage of eco verify:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoTopLevelHelpMentionsVerifyLock(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"eco --help exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "eco verify --lock") {
		t.Fatalf("stdout = %q, want verify --lock guidance", stdout.String())
	}
}

func TestEcoPackUnpackVaultHelpExitsSuccessfully(t *testing.T) {
	for _, args := range [][]string{
		{"eco", "pack", "--help"},
		{"eco", "unpack", "--help"},
		{"eco", "vault", "--help"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf(
					"%v exit code = %d, stdout=%q stderr=%q",
					args,
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), "usage:") {
				t.Fatalf(
					"%v output missing usage text: stdout=%q stderr=%q",
					args,
					stdout.String(),
					stderr.String(),
				)
			}
		})
	}
}

func TestEcoPackProjectBundle(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(srcDir, "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.todex")
	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkg},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	outDir := filepath.Join(dir, "unpacked")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco unpack exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "src", "main.tetra")); err != nil {
		t.Fatalf("expected bundled source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.package.json")); err != nil {
		t.Fatalf("expected bundled package metadata: %v", err)
	}
}

func TestEcoPackProjectBundleUsesT4CapsuleAndSource(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Capsule.t4")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(srcDir, "main.t4"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.tdx")
	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkg},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	outDir := filepath.Join(dir, "unpacked")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco unpack exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{"Capsule.t4", "src/main.t4", "tetra.package.json"} {
		if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected bundled %s: %v", rel, err)
		}
	}
}

func TestEcoVerifyStructuredCapsuleT4WritesPolicyLock(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Capsule.t4")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"

    sources:
        src
        ui

    targets:
        linux
        web

    allow:
        ui
        fs.readWrite.userData

    policy:
        unsafe deny
        reproducible required
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, capsule}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	for _, want := range []string{
		`"path": "` + capsule + `"`,
		`"linux-x64"`,
		`"wasm32-web"`,
		`"ui"`,
		`"fs.readWrite.userData"`,
		`"unsafe": "deny"`,
		`"reproducible": "required"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("lock missing %q:\n%s", want, string(raw))
		}
	}
}

func TestEcoVerifyDependencyGraphAndLock(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(core, []byte(`capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    effect "io"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    effect "io"
    dependency "tetra://core" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lock, app, core},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(lock)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"capsules"`) ||
		!strings.Contains(string(raw), `"tetra://core"`) {
		t.Fatalf("unexpected lock: %s", string(raw))
	}
}

func TestEcoVerifyRejectsPermissionEscalationFromDependency(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(core, []byte(`capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    effect "io"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    dependency "tetra://core" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app, core}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected permission mismatch failure")
	}
	if !strings.Contains(stderr.String(), "missing required effect") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyRejectsDuplicateManifestIDField(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    id "tetra://app-2"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected duplicate id field failure")
	}
	if !strings.Contains(stderr.String(), "duplicate id field") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyReportsMissingDependency(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    dependency "tetra://missing" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected missing dependency failure")
	}
	if !strings.Contains(stderr.String(), "missing dependency") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyReportsDuplicateIDAndTargetMismatch(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "One.capsule")
	two := filepath.Join(dir, "Two.capsule")
	if err := os.WriteFile(one, []byte(`capsule One:
    id "tetra://dup"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(two, []byte(`capsule Two:
    id "tetra://dup"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", one, two}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected duplicate capsule id failure")
	}
	if !strings.Contains(stderr.String(), "duplicate capsule id") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	code = runCLI(
		[]string{"eco", "verify", "--target", "windows-x64", one},
		&bytes.Buffer{},
		&stderr,
	)
	if code == 0 {
		t.Fatalf("expected target mismatch failure")
	}
	if !strings.Contains(stderr.String(), "target mismatch") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVaultAddListAndVerify(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "vault")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Vault added: sha256:") {
		t.Fatalf("vault add stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "list", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"vault list exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "source") ||
		!strings.Contains(stdout.String(), "module.tetra") {
		t.Fatalf("vault list stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"vault verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Vault OK: 1 records") {
		t.Fatalf("vault verify stdout = %q", stdout.String())
	}
}

func TestEcoVaultVerifyDetectsCorruptObject(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "vault")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	fields := strings.Fields(stdout.String())
	if len(fields) < 3 || !strings.HasPrefix(fields[2], "sha256:") {
		t.Fatalf("unexpected vault add stdout = %q", stdout.String())
	}
	hash := strings.TrimPrefix(fields[2], "sha256:")
	objectPath := filepath.Join(store, "objects", "sha256", hash)
	if err := os.WriteFile(objectPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected vault verify failure")
	}
	if !strings.Contains(stderr.String(), "vault object") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// ---- eco_tetrahub_test.go ----

func ecoDownloadArgs(registry, out string) []string {
	return []string{
		"eco",
		"download",
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
		"--registry",
		registry,
		"-o",
		out,
	}
}

func ecoHubDownloadArgs(store, out string) []string {
	return []string{
		"eco",
		"tetrahub",
		"download",
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
		"--store",
		store,
		"-o",
		out,
	}
}

func ecoHubMirrorArgs(fromStore, toStore, out string) []string {
	return []string{
		"eco",
		"tetrahub",
		"mirror",
		"--from",
		fromStore,
		"--to",
		toStore,
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
		"-o",
		out,
	}
}

func ecoHubFetchArgs(url, toStore, out string) []string {
	return []string{
		"eco",
		"tetrahub",
		"fetch",
		"--url",
		url,
		"--to",
		toStore,
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
		"-o",
		out,
	}
}

func TestEcoBetaPublishDownloadAndTetraHubPath(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(project, "src", "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    target "windows-x64"
    permission "io"
`)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	store := filepath.Join(dir, "vault")
	hubStore := filepath.Join(dir, "tetrahub-beta")
	downloadPath := filepath.Join(dir, "downloaded.todex")
	hubDownloadPath := filepath.Join(dir, "hub-downloaded.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(
			project,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	lockPath := filepath.Join(dir, "tetra.lock.json")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"publish",
			"--package",
			pkgPath,
			"--registry",
			registry,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Published (beta)") {
		t.Fatalf("publish stdout = %q", stdout.String())
	}
	cmd := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-publish",
		"--registry",
		registry,
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish failed: %v\n%s", err, out)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoDownloadArgs(registry, downloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("downloaded package missing: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			hubStore,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoHubDownloadArgs(hubStore, hubDownloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco tetrahub download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(hubDownloadPath); err != nil {
		t.Fatalf("hub downloaded package missing: %v", err)
	}

	metaPath := filepath.Join(
		registry,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"metadata.json",
	)
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read publish metadata: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("decode publish metadata: %v\n%s", err, string(rawMeta))
	}
	if meta["schema"] != "tetra.eco.publish.v1beta" || meta["channel"] != "beta" {
		t.Fatalf("publish metadata = %#v", meta)
	}
	trustMeta, ok := meta["trust"].(map[string]any)
	if !ok {
		t.Fatalf("publish metadata missing trust object: %#v", meta)
	}
	if trustMeta["snapshot_file"] != "trust.snapshot.json" {
		t.Fatalf("trust snapshot file should be registry-local relative path: %#v", trustMeta)
	}
	if _, err := os.Stat(
		filepath.Join(registry, "packages", "tetra_demo", "0.1.0", "linux-x64", "trust.snapshot.json"),
	); err != nil {
		t.Fatalf("published trust snapshot missing: %v", err)
	}
}

func TestEcoPublishStableChannelProducesProductionMetadata(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	store := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	downloadPath := filepath.Join(dir, "downloaded-stable.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(
			project,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"publish",
			"--package",
			pkgPath,
			"--registry",
			registry,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
			"--channel",
			"stable",
			"--hub",
			"production",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Published (stable)") {
		t.Fatalf("stable publish stdout = %q", stdout.String())
	}

	metaPath := filepath.Join(
		registry,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"metadata.json",
	)
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read publish metadata: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("decode publish metadata: %v\n%s", err, string(rawMeta))
	}
	if meta["schema"] != "tetra.eco.publish.v1" || meta["channel"] != "stable" ||
		meta["hub"] != "production" {
		t.Fatalf("stable publish metadata = %#v", meta)
	}
	cmd := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-publish",
		"--registry",
		registry,
		"--id",
		"tetra://demo",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
		"--channel",
		"stable",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish stable failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoDownloadArgs(registry, downloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco stable download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("stable downloaded package missing: %v", err)
	}
}

func TestEcoTetraHubStableChannelProducesProductionMetadata(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	store := filepath.Join(dir, "tetrahub")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	downloadPath := filepath.Join(dir, "hub-downloaded-stable.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(
			project,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			store,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "TetraHub stable published") {
		t.Fatalf("stable tetrahub publish stdout = %q", stdout.String())
	}

	metaPath := filepath.Join(
		store,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"metadata.json",
	)
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read tetrahub metadata: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("decode tetrahub metadata: %v\n%s", err, string(rawMeta))
	}
	if meta["schema"] != "tetra.eco.publish.v1" || meta["channel"] != "stable" ||
		meta["hub"] != "tetrahub-stable" {
		t.Fatalf("stable tetrahub metadata = %#v", meta)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoHubDownloadArgs(store, downloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco tetrahub stable download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("stable tetrahub downloaded package missing: %v", err)
	}
}

func TestEcoTetraHubMirrorCopiesStablePackageAndWritesReport(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	reportPath := filepath.Join(dir, "mirror.report.json")
	downloadPath := filepath.Join(dir, "mirrored-download.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(
			project,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			sourceStore,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		ecoHubMirrorArgs(sourceStore, destStore, reportPath),
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub mirror exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "TetraHub mirrored") {
		t.Fatalf("mirror stdout = %q", stdout.String())
	}
	rawReport, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read mirror report: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(rawReport, &report); err != nil {
		t.Fatalf("decode mirror report: %v\n%s", err, string(rawReport))
	}
	if report["schema"] != "tetra.eco.mirror.v1" || report["id"] != "tetra://demo" ||
		report["target"] != "linux-x64" {
		t.Fatalf("mirror report = %#v", report)
	}
	if report["package_sha256"] == "" || report["metadata_sha256"] == "" ||
		report["trust_snapshot_sha256"] == "" {
		t.Fatalf("mirror report missing hashes: %#v", report)
	}
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-mirror", "--mirror", reportPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate mirror report failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoHubDownloadArgs(destStore, downloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco tetrahub mirrored download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	sourceMeta, err := os.ReadFile(
		filepath.Join(sourceStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json"),
	)
	if err != nil {
		t.Fatalf("read source metadata: %v", err)
	}
	destMeta, err := os.ReadFile(
		filepath.Join(destStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json"),
	)
	if err != nil {
		t.Fatalf("read mirrored metadata: %v", err)
	}
	if !bytes.Equal(sourceMeta, destMeta) {
		t.Fatalf("mirrored metadata changed\nsource=%s\ndest=%s", sourceMeta, destMeta)
	}
}

func TestEcoTetraHubFetchMirrorsStablePackageOverHTTP(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-source")
	destStore := filepath.Join(dir, "tetrahub-fetched")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	reportPath := filepath.Join(dir, "fetch.report.json")
	downloadPath := filepath.Join(dir, "fetched-download.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(
			project,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			sourceStore,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	server := httptest.NewServer(http.FileServer(http.Dir(sourceStore)))
	defer server.Close()

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		ecoHubFetchArgs(server.URL, destStore, reportPath),
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub fetch exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "TetraHub fetched") {
		t.Fatalf("fetch stdout = %q", stdout.String())
	}
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-mirror", "--mirror", reportPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate fetch mirror report failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoHubDownloadArgs(destStore, downloadPath), &stdout, &stderr); code != 0 {
		t.Fatalf(
			"eco tetrahub fetched download exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("fetched package missing: %v", err)
	}
}

func TestEcoTetraHubFetchRejectsTamperedHTTPPackage(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-source")
	destStore := filepath.Join(dir, "tetrahub-fetched")
	reportPath := filepath.Join(dir, "fetch.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			sourceStore,
			"--target",
			"linux-x64",
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	publishedPackage := filepath.Join(
		sourceStore,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"package.todex",
	)
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publishedPackage, append(raw, []byte("tampered")...), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.FileServer(http.Dir(sourceStore)))
	defer server.Close()

	stdout.Reset()
	stderr.Reset()
	code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"fetch",
			"--url",
			server.URL,
			"--to",
			destStore,
			"--id",
			"tetra://demo",
			"--version",
			"0.1.0",
			"--target",
			"linux-x64",
			"-o",
			reportPath,
		},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf(
			"expected fetch failure for tampered package, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "package size mismatch") &&
		!strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("stderr = %q, want package integrity failure", stderr.String())
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("fetch report should not be written after integrity failure: %v", err)
	}
}

func TestEcoTetraHubMirrorRejectsTamperedSourcePackage(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	reportPath := filepath.Join(dir, "mirror.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			sourceStore,
			"--target",
			"linux-x64",
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	publishedPackage := filepath.Join(
		sourceStore,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"package.todex",
	)
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publishedPackage, append(raw, []byte("tampered")...), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"mirror",
			"--from",
			sourceStore,
			"--to",
			destStore,
			"--id",
			"tetra://demo",
			"--version",
			"0.1.0",
			"--target",
			"linux-x64",
			"-o",
			reportPath,
		},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf(
			"expected mirror failure for tampered package, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "package size mismatch") &&
		!strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("stderr = %q, want package integrity failure", stderr.String())
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("mirror report should not be written after integrity failure: %v", err)
	}
}

func TestEcoTetraHubMirrorRejectsDestinationSymlinkTraversal(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	outside := filepath.Join(dir, "outside")
	reportPath := filepath.Join(dir, "mirror.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"publish",
			"--package",
			pkgPath,
			"--store",
			sourceStore,
			"--target",
			"linux-x64",
			"--channel",
			"stable",
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	targetParent := filepath.Join(destStore, "packages", "tetra_demo", "0.1.0")
	if err := os.MkdirAll(targetParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(targetParent, "linux-x64")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code := runCLI(
		[]string{
			"eco",
			"tetrahub",
			"mirror",
			"--from",
			sourceStore,
			"--to",
			destStore,
			"--id",
			"tetra://demo",
			"--version",
			"0.1.0",
			"--target",
			"linux-x64",
			"-o",
			reportPath,
		},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf(
			"expected mirror symlink destination failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "symlink") {
		t.Fatalf("stderr = %q, want symlink rejection", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "package.todex")); !os.IsNotExist(err) {
		t.Fatalf("mirror wrote through destination symlink: %v", err)
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("mirror report should not be written after symlink failure: %v", err)
	}
}

func TestEcoDownloadRejectsTamperedPublishedPackage(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(project, "src", "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	downloadPath := filepath.Join(dir, "downloaded.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "publish", "--package", pkgPath, "--registry", registry, "--target", "linux-x64"},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco publish exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	publishedPackage := filepath.Join(
		registry,
		"packages",
		"tetra_demo",
		"0.1.0",
		"linux-x64",
		"package.todex",
	)
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("published package is empty")
	}
	raw[0] ^= 0xff
	if err := os.WriteFile(publishedPackage, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(ecoDownloadArgs(registry, downloadPath), &stdout, &stderr); code == 0 {
		t.Fatalf(
			"expected eco download failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoDownloadRejectsPublishMetadataUnknownFieldsAndKeyMismatches(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(map[string]any)
		rawSuffix string
		want      string
	}{
		{
			name: "unknown field",
			mutate: func(meta map[string]any) {
				meta["unexpected"] = true
			},
			want: "unknown field",
		},
		{
			name: "extra field",
			mutate: func(meta map[string]any) {
				meta["extra"] = map[string]any{"note": "not in publish contract"}
			},
			want: "unknown field",
		},
		{
			name: "capsule id mismatch",
			mutate: func(meta map[string]any) {
				capsule := meta["capsule"].(map[string]any)
				capsule["id"] = "tetra://other"
			},
			want: "capsule id mismatch",
		},
		{
			name: "download target mismatch",
			mutate: func(meta map[string]any) {
				downloads := meta["downloads"].([]any)
				download := downloads[0].(map[string]any)
				download["target"] = "windows-x64"
			},
			want: "download target mismatch",
		},
		{
			name:      "trailing JSON payload",
			rawSuffix: "\n{}",
			want:      "trailing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			_, capsule := writeEcoProjectFixture(t, dir)
			pkgPath := filepath.Join(dir, "demo.todex")
			registry := filepath.Join(dir, "registry")
			downloadPath := filepath.Join(dir, "downloaded.todex")

			var stdout, stderr bytes.Buffer
			if code := runCLI(
				[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
				&stdout,
				&stderr,
			); code != 0 {
				t.Fatalf(
					"eco pack --project exit code = %d, stdout=%q stderr=%q",
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			stdout.Reset()
			stderr.Reset()
			if code := runCLI(
				[]string{
					"eco",
					"publish",
					"--package",
					pkgPath,
					"--registry",
					registry,
					"--target",
					"linux-x64",
				},
				&stdout,
				&stderr,
			); code != 0 {
				t.Fatalf(
					"eco publish exit code = %d, stdout=%q stderr=%q",
					code,
					stdout.String(),
					stderr.String(),
				)
			}

			metaPath := filepath.Join(
				registry,
				"packages",
				"tetra_demo",
				"0.1.0",
				"linux-x64",
				"metadata.json",
			)
			rawMeta, err := os.ReadFile(metaPath)
			if err != nil {
				t.Fatalf("read publish metadata: %v", err)
			}
			var meta map[string]any
			if err := json.Unmarshal(rawMeta, &meta); err != nil {
				t.Fatalf("decode publish metadata: %v\n%s", err, string(rawMeta))
			}
			if tt.mutate != nil {
				tt.mutate(meta)
			}
			rawMeta, err = json.MarshalIndent(meta, "", "  ")
			if err != nil {
				t.Fatalf("encode publish metadata: %v", err)
			}
			rawMeta = append(rawMeta, tt.rawSuffix...)
			if err := os.WriteFile(metaPath, append(rawMeta, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}

			stdout.Reset()
			stderr.Reset()
			if code := runCLI(ecoDownloadArgs(registry, downloadPath), &stdout, &stderr); code == 0 {
				t.Fatalf(
					"expected eco download failure, stdout=%q stderr=%q",
					stdout.String(),
					stderr.String(),
				)
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("unexpected stderr: got %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestEcoUnpackRejectsTamperedPackageContent(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	tamperedPath := filepath.Join(dir, "demo-tampered.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	tamperTodexEntry(
		t,
		pkgPath,
		tamperedPath,
		"src/main.tetra",
		[]byte("func main() -> Int:\n    return 9\n"),
	)

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "unpack", tamperedPath, "-C", filepath.Join(project, "out")},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatalf(
			"expected tampered unpack failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "package metadata hash mismatch for src/main.tetra") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoUnpackRejectsTamperedPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	tamperedPath := filepath.Join(dir, "demo-metadata-tampered.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	tamperTodexEntry(
		t,
		pkgPath,
		tamperedPath,
		"tetra.package.json",
		[]byte(
			("{\"schema\":\"tetra.eco.package.v1\",\"compression\":\"gzip\","+
				"\"mtime_unix\":0,\"file_count\":1,\"files\":[]}")+"\n",
		),
	)

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "unpack", tamperedPath, "-C", filepath.Join(dir, "out")},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatalf(
			"expected tampered metadata failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "package metadata file_count mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoVaultVerifyRejectsTamperedObject(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "vault")
	source := filepath.Join(dir, "src", "main.tetra")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", source},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	fields := strings.Fields(stdout.String())
	if len(fields) < 3 {
		t.Fatalf("unexpected vault add stdout: %q", stdout.String())
	}
	object := filepath.Join(store, "objects", "sha256", strings.TrimPrefix(fields[2], "sha256:"))
	if err := os.WriteFile(object, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "verify", "--store", store},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatalf(
			"expected vault verify failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "vault object mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoDogfoodFixtureLocalLifecycle(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "examples", "projects", "eco_dogfood")
	app := filepath.Join(project, "Tetra.capsule")
	core := filepath.Join(project, "Core.capsule")
	source := filepath.Join(project, "src", "main.tetra")
	for _, path := range []string{app, core, source} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dogfood fixture missing %s: %v", path, err)
		}
	}

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "tetra.lock.json")
	pkgPath := filepath.Join(dir, "eco-dogfood.todex")
	unpackDir := filepath.Join(dir, "unpacked")
	store := filepath.Join(dir, "vault")
	registry := filepath.Join(dir, "registry")
	trustPath := filepath.Join(dir, "trust.snapshot.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, app, core},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if out, err := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-lock",
		"--lock",
		lockPath,
	).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-lock failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "pack", "--project", app, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "unpack", pkgPath, "-C", unpackDir},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco unpack dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if out, err := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-unpack",
		"--dir",
		unpackDir,
	).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-unpack failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(unpackDir, "Tetra.capsule")); err != nil {
		t.Fatalf("unpacked dogfood capsule missing: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", source},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "verify", "--store", store},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault verify dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if out, err := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-vault",
		"--store",
		store,
	).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-vault failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"publish",
			"--package",
			pkgPath,
			"--registry",
			registry,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco publish dogfood exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	cmd := testCommand(
		t,
		"go",
		"run",
		"./tools/cmd/validate-eco-publish",
		"--registry",
		registry,
		"--id",
		"tetra://examples/eco-dogfood",
		"--version",
		"0.1.0",
		"--target",
		"linux-x64",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish failed: %v\n%s", err, out)
	}
	metaPath := filepath.Join(
		registry,
		"packages",
		"tetra_examples_eco_dogfood",
		"0.1.0",
		"linux-x64",
		"metadata.json",
	)
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("dogfood publish metadata missing: %v", err)
	}
}

func TestEcoDocsDeclareLocalOnlyBetaScope(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(root, "docs", "spec", "policy", "eco_publishing_v1.md")
	userPath := filepath.Join(root, "docs", "user", "platform", "eco_package_guide.md")
	for _, path := range []string{specPath, userPath} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, "local") {
			t.Fatalf("%s should declare local scope", path)
		}
		if !strings.Contains(text, "beta") {
			t.Fatalf("%s should declare beta boundary", path)
		}
		if !strings.Contains(text, "TetraHub") {
			t.Fatalf("%s should mention TetraHub boundary", path)
		}
	}
}

// ---- eco_wave10_fixtures_test.go ----

func writeCapsuleFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeEcoProjectFixture(t *testing.T, dir string) (string, string) {
	t.Helper()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(project, "src", "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	return project, capsule
}

func writeTarGzFixture(t *testing.T, path string, name string, body []byte) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(
		&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTodexWithSymlinkEntry(t *testing.T, path string) {
	t.Helper()
	capsule := []byte(`manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
`)
	emptySum := sha256.Sum256(nil)
	capsuleSum := sha256.Sum256(capsule)
	files := []ecoPackageMetadataFile{
		{
			Path:   "Capsule.t4",
			SHA256: "sha256:" + hex.EncodeToString(capsuleSum[:]),
			Size:   int64(len(capsule)),
		},
		{Path: "src/main.tetra", SHA256: "sha256:" + hex.EncodeToString(emptySum[:]), Size: 0},
	}
	metadata := ecoPackageMetadata{
		Schema:           ecoPackageSchemaV1,
		Compression:      "gzip",
		MTimeUnix:        0,
		Reproducible:     true,
		ManifestSchema:   capsuleManifestSchemaV1,
		PermissionsModel: ecoPermissionsModelV1,
		FileCount:        len(files),
		Files:            files,
	}
	fingerprintSum := sha256.Sum256([]byte(packageMetadataFingerprint(files)))
	metadata.BuildInputsSHA = "sha256:" + hex.EncodeToString(fingerprintSum[:])
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	rawMetadata = append(rawMetadata, '\n')

	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(
		&tar.Header{Name: "Capsule.t4", Mode: 0o644, Size: int64(len(capsule))},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(capsule); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(
		&tar.Header{
			Name:     "src/main.tetra",
			Typeflag: tar.TypeSymlink,
			Linkname: "../outside.tetra",
			Mode:     0o777,
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(
		&tar.Header{Name: ecoPackageMetadataPath, Mode: 0o644, Size: int64(len(rawMetadata))},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(rawMetadata); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func tamperTodexEntry(t *testing.T, src string, dst string, entryName string, body []byte) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	gzw := gzip.NewWriter(out)
	tw := tar.NewWriter(gzw)
	found := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		next := *header
		if header.Name == entryName {
			raw = body
			next.Size = int64(len(raw))
			found = true
		}
		if err := tw.WriteHeader(&next); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Fatalf("entry %s not found in %s", entryName, src)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
}

func testCommand(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	return cmd
}

// ---- eco_wave10_test.go ----

func TestEcoVerifyManifestV1PermissionsAndLockMetadata(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	writeCapsuleFile(t, app, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
    dependency "tetra://core" "0.1.0"
`)

	lockPath := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, app, core},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`"schema": "tetra.eco.lock.v1"`,
		`"manifest_schema": "tetra.capsule.v1"`,
		`"permissions_model": "tetra.eco.permissions.v1"`,
		`"permissions": [`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("lock missing %q:\n%s", want, text)
		}
	}
}

func TestEcoVerifyRejectsMissingRequiredManifestFields(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "capsule declaration",
			text: `manifest "tetra.capsule.v1"
id "tetra://app"
version "0.1.0"
target "linux-x64"
`,
			want: "missing capsule declaration",
		},
		{
			name: "id",
			text: `manifest "tetra.capsule.v1"
capsule App:
version "0.1.0"
target "linux-x64"
`,
			want: "missing capsule id",
		},
		{
			name: "version",
			text: `manifest "tetra.capsule.v1"
capsule App:
id "tetra://app"
target "linux-x64"
`,
			want: "missing capsule version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			capsule := filepath.Join(dir, "Tetra.capsule")
			writeCapsuleFile(t, capsule, tt.text)
			var stderr bytes.Buffer
			if code := runCLI([]string{"eco", "verify", capsule}, &bytes.Buffer{}, &stderr); code == 0 {
				t.Fatalf("expected eco verify failure")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestEcoCapsuleFixtureMatrix(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(root, "cli", "cmd", "tetra", "testdata", "eco_capsules", "matrix")
	tests := []struct {
		name    string
		files   []string
		target  string
		wantOK  bool
		wantErr string
	}{
		{
			name:   "valid graph",
			files:  []string{"valid/Tetra.capsule", "valid/Core.capsule"},
			target: "linux-x64",
			wantOK: true,
		},
		{
			name:    "missing dependency",
			files:   []string{"missing_dependency/Tetra.capsule"},
			target:  "linux-x64",
			wantErr: "missing dependency tetra://fixture/missing",
		},
		{
			name:    "malformed manifest",
			files:   []string{"malformed/Tetra.capsule"},
			wantErr: "expected quoted string",
		},
		{
			name:    "unsupported target",
			files:   []string{"unsupported_target/Tetra.capsule"},
			wantErr: "unsupported target plan9-x64",
		},
		{
			name: "duplicate dependency",
			files: []string{
				"duplicate_dependency/Tetra.capsule",
				"duplicate_dependency/Core.capsule",
			},
			target:  "linux-x64",
			wantErr: "duplicate dependency tetra://fixture/core 0.1.0",
		},
		{
			name: "permission mismatch",
			files: []string{
				"permission_mismatch/Tetra.capsule",
				"permission_mismatch/Core.capsule",
			},
			target:  "linux-x64",
			wantErr: "missing required permission mmio for dependency tetra://fixture/core",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"eco", "verify"}
			if tt.target != "" {
				args = append(args, "--target", tt.target)
			}
			for _, file := range tt.files {
				args = append(args, filepath.Join(fixtureDir, file))
			}
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr)
			if tt.wantOK {
				if code != 0 {
					t.Fatalf(
						"eco verify exit code = %d, stdout=%q stderr=%q",
						code,
						stdout.String(),
						stderr.String(),
					)
				}
				return
			}
			if code == 0 {
				t.Fatalf(
					"expected eco verify failure, stdout=%q stderr=%q",
					stdout.String(),
					stderr.String(),
				)
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestEcoLockFixtureRejectsGraphHashMismatch(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(
		root,
		"cli",
		"cmd",
		"tetra",
		"testdata",
		"eco_capsules",
		"matrix",
		"lock_mismatch",
		"tetra.lock.json",
	)
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-lock", "--lock", lock)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected validate-eco-lock failure\n%s", out)
	}
	if !strings.Contains(string(out), "graph_sha256 mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestEcoVerifyManifestV1RejectsDependencyPermissionEscalation(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    permission "mmio"
`)
	writeCapsuleFile(t, app, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    dependency "tetra://core" "0.1.0"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--target", "linux-x64", app, core}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf(
			"expected eco verify permission failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(
		stderr.String(),
		"missing required permission mmio for dependency tetra://core",
	) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoVerifyRejectsDependencyVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.2.0"
    target "linux-x64"
    permission "io"
`)
	writeCapsuleFile(t, app, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
    dependency "tetra://core" "0.1.0"
`)
	var stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", app, core},
		&bytes.Buffer{},
		&stderr,
	); code == 0 {
		t.Fatalf("expected eco verify failure")
	}
	if !strings.Contains(stderr.String(), "version mismatch") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoSeedExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	seedPath := filepath.Join(dir, "tetra.seed.json")
	lockPath := filepath.Join(dir, "imported.lock.json")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "seed", "export", "--out", seedPath, capsule}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"eco seed export exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{"eco", "seed", "import", "--seed", seedPath, "--lock", lockPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"eco seed import exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected imported lock file: %v", err)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read imported lock: %v", err)
	}
	if !strings.Contains(string(raw), `"schema": "tetra.eco.lock.v1"`) {
		t.Fatalf("imported lock missing schema: %s", string(raw))
	}
}

func TestEcoSeedImportRejectsUnsupportedPermissionsModel(t *testing.T) {
	dir := t.TempDir()
	seedPath := filepath.Join(dir, "seed.json")
	lockPath := filepath.Join(dir, "lock.json")
	if err := os.WriteFile(seedPath, []byte(`{
  "schema": "tetra.eco.seed.v1",
  "generated_at_unix": 0,
  "lock": {
    "schema": "tetra.eco.lock.v1",
    "manifest_schema": "tetra.capsule.v1",
    "permissions_model": "tetra.eco.permissions.v2",
    "capsules": [
      {
        "id": "tetra://app",
        "name": "App",
        "version": "0.1.0",
        "path": "Tetra.capsule",
        "targets": ["linux-x64"],
        "permissions": ["io"]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "seed", "import", "--seed", seedPath, "--lock", lockPath},
		&bytes.Buffer{},
		&stderr,
	); code == 0 {
		t.Fatalf("expected eco seed import failure")
	}
	if !strings.Contains(stderr.String(), "unsupported lock permissions model") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoPackProjectBundleIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	if err := os.WriteFile(
		filepath.Join(project, "src", "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "first.todex")
	second := filepath.Join(dir, "second.todex")
	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", first},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"first eco pack exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", second},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"second eco pack exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	firstRaw, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstRaw, secondRaw) {
		t.Fatalf("project bundle output is not deterministic")
	}
}

func TestEcoUnpackRejectsUnsafeArchivePath(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "unsafe.todex")
	writeTarGzFixture(t, pkgPath, "../evil.tetra", []byte("func main() -> Int:\n    return 0\n"))
	var stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "unpack", pkgPath, "-C", filepath.Join(dir, "out")},
		&bytes.Buffer{},
		&stderr,
	); code == 0 {
		t.Fatalf("expected eco unpack failure")
	}
	if !strings.Contains(stderr.String(), "unsafe archive path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoUnpackRejectsArchiveSymlinkEntry(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "symlink-entry.todex")
	writeTodexWithSymlinkEntry(t, pkgPath)

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "unpack", pkgPath, "-C", filepath.Join(dir, "out")},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatalf(
			"expected symlink archive entry rejection, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "unsupported archive entry type") {
		t.Fatalf("stderr = %q, want unsupported archive entry type", stderr.String())
	}
}

func TestEcoUnpackRejectsOutputSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	outDir := filepath.Join(dir, "out")
	outside := filepath.Join(dir, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(outDir, "src")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkgPath, "-C", outDir}, &stdout, &stderr); code == 0 {
		t.Fatalf(
			"expected output symlink rejection, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "symlink") {
		t.Fatalf("stderr = %q, want symlink rejection", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "main.tetra")); !os.IsNotExist(err) {
		t.Fatalf("unpack wrote through symlink ancestor: %v", err)
	}
}

func TestEcoNeedMapTrustSnapshotAndMaterialize(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "src", "main.tetra"),
		[]byte("func main() -> Int:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)

	lockPath := filepath.Join(dir, "tetra.lock.json")
	needMapPath := filepath.Join(dir, "needmap.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	pkgPath := filepath.Join(dir, "demo.todex")
	store := filepath.Join(dir, "vault")
	outDir := filepath.Join(dir, "materialized")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "needmap", "--lock", lockPath, "-o", needMapPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco needmap exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(
			dir,
			"src",
			"main.tetra",
		)},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco vault add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco trust snapshot exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "pack", "--project", capsule, "-o", pkgPath},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco pack --project exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{
			"eco",
			"materialize",
			pkgPath,
			"--target",
			"linux-x64",
			"--trust",
			trustPath,
			"-C",
			outDir,
		},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco materialize exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.materialization.json")); err != nil {
		t.Fatalf("expected materialization metadata: %v", err)
	}
	rawNeedMap, err := os.ReadFile(needMapPath)
	if err != nil {
		t.Fatalf("read needmap: %v", err)
	}
	if !strings.Contains(string(rawNeedMap), `"schema": "tetra.eco.needmap.v1"`) {
		t.Fatalf("needmap missing schema: %s", string(rawNeedMap))
	}
	rawTrust, err := os.ReadFile(trustPath)
	if err != nil {
		t.Fatalf("read trust snapshot: %v", err)
	}
	if !strings.Contains(string(rawTrust), `"schema": "tetra.eco.trust-snapshot.v1"`) {
		t.Fatalf("trust snapshot missing schema: %s", string(rawTrust))
	}
}

func TestEcoNeedMapRejectsUnsupportedLockSchema(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "tetra.lock.json")
	if err := os.WriteFile(lockPath, []byte(`{
  "schema": "tetra.eco.lock.v2",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "needmap", "--lock", lockPath, "-o", filepath.Join(dir, "needmap.json")},
		&bytes.Buffer{},
		&stderr,
	); code == 0 {
		t.Fatalf("expected eco needmap failure")
	}
	if !strings.Contains(stderr.String(), "unsupported lock schema") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoTrustSnapshotRejectsLockGraphHashMismatch(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	lockPath := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	if code := runCLI(
		[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf(
			"eco verify exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	lock["graph_sha256"] = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	raw, err = json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(lockPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", filepath.Join(
			dir,
			"vault",
		), "-o", filepath.Join(
			dir,
			"trust.json",
		)},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatalf(
			"expected eco trust snapshot failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "lock graph_sha256 mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoTrustSnapshotRejectsUnreadableVaultStore(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, store string)
	}{
		{
			name: "missing index",
			setup: func(t *testing.T, store string) {
				if err := os.MkdirAll(store, 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "broken index",
			setup: func(t *testing.T, store string) {
				if err := os.MkdirAll(store, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(
					filepath.Join(store, "records.json"),
					[]byte("{not json\n"),
					0o644,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			capsule := filepath.Join(dir, "Tetra.capsule")
			writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
			lockPath := filepath.Join(dir, "tetra.lock.json")
			store := filepath.Join(dir, "vault")
			trustPath := filepath.Join(dir, "trust.json")
			tt.setup(t, store)

			var stdout, stderr bytes.Buffer
			if code := runCLI(
				[]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule},
				&stdout,
				&stderr,
			); code != 0 {
				t.Fatalf(
					"eco verify exit code = %d, stdout=%q stderr=%q",
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			stdout.Reset()
			stderr.Reset()
			if code := runCLI(
				[]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath},
				&stdout,
				&stderr,
			); code == 0 {
				t.Fatalf(
					"expected eco trust snapshot failure, stdout=%q stderr=%q",
					stdout.String(),
					stderr.String(),
				)
			}
			if !strings.Contains(stderr.String(), "read vault store") {
				t.Fatalf("unexpected stderr: %q", stderr.String())
			}
			if _, err := os.Stat(trustPath); err == nil {
				t.Fatalf("trust snapshot should not be written for unreadable vault store")
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		})
	}
}

// ---- fmt_test.go ----

func TestFmtCommandCheckAndStdout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int\nuses mem, io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"fmt", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "uses io, mem:") {
		t.Fatalf("fmt stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--check", srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("fmt --check should fail for unformatted file")
	}
}

func TestCollectTetraFilesIncludesT4AndLegacyTetra(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.t4":      "func main() -> Int:\n    return 0\n",
		"legacy.tetra": "func legacy() -> Int:\n    return 0\n",
		"ignore.tdx":   "not source\n",
	}
	for rel, src := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := collectTetraFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectTetraFiles: %v", err)
	}
	want := []string{filepath.Join(dir, "legacy.tetra"), filepath.Join(dir, "main.t4")}
	if len(got) != len(want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files = %#v, want %#v", got, want)
		}
	}
}

func TestCollectTetraFilesSkipsCapsuleManifest(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	got, err := collectTetraFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectTetraFiles: %v", err)
	}
	want := []string{filepath.Join(dir, "src", "main.t4")}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestFormatCommandWriteIsIdempotentAndPreservesStandaloneComments(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := `// module docs
func main() -> Int uses mem, io:
    // return path
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--write", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt --write exit code = %d, stderr=%q", code, stderr.String())
	}
	once, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"// module docs", "uses io, mem:", "    // return path"} {
		if !strings.Contains(string(once), want) {
			t.Fatalf("formatted file missing %q:\n%s", want, string(once))
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--write", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second fmt --write exit code = %d, stderr=%q", code, stderr.String())
	}
	twice, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(twice) != string(once) {
		t.Fatalf("fmt --write not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--check", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt --check after write exit code = %d, stderr=%q", code, stderr.String())
	}
}

func TestFormatCommandJSONDiagnosticsForInlineComment(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := strings.Join([]string{
		"func main() -> Int:\n",
		"    return 0 // keep me\n",
	}, "")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT001" || diag.File != srcPath || diag.Line != 2 || diag.Column != 14 ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "inline comments are not supported") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestFmtCommandJSONDiagnosticsForInvalidModeCombination(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{
			"fmt",
			"--diagnostics=json",
			"--check",
			"--write",
			"examples/flow/flow_hello.tetra",
		},
		2,
	)
	if diag.Code != "TETRA0001" || diag.Message != "fmt accepts only one of --check or --write" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForMissingPath(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "fmt requires at least one path" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForMultipleStdoutFiles(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "one.tetra")
	two := filepath.Join(dir, "two.tetra")
	if err := os.WriteFile(one, []byte("func one() -> Int:\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(two, []byte("func two() -> Int:\n    return 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json", one, two}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "fmt stdout mode accepts exactly one file" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCheckJSONDiagnosticsForUnformattedFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int uses io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Message != "not formatted" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCheckTOONDiagnosticsForUnformattedFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int uses io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLITOONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=toon", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Message != "not formatted" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFormatCommandCheckJSONDiagnosticsIncludesFirstDiffPosition(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(
		srcPath,
		[]byte("func main() -> Int uses io:\n    return 0\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Line != 1 || diag.Column != 19 ||
		diag.Message != "not formatted" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

// ---- interface_test.go ----

func TestInterfaceCommandWritesT4IFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "math.t4")
	outPath := filepath.Join(dir, "math.t4i")
	src := strings.Join([]string{
		"module math.core\n",
		"func add(a: Int, b: Int) -> Int:\n",
		"    return a + b\n",
	}, "")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"interface", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"interface exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read interface: %v", err)
	}
	if !strings.Contains(string(raw), "func add(a: i32, b: i32) -> i32:") {
		t.Fatalf("interface output = %s", raw)
	}
}

func TestInterfaceCommandCheckReportsStalePublicAPI(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	outPath := filepath.Join(dir, "math.t4i")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"interface", "-o", outPath, srcPath}, &stdout, &stderr); code != 0 {
		t.Fatalf(
			"interface write exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Bool) -> Int:
    return a
`)

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"interface", "--check", "-o", outPath, srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf(
			"expected stale interface check failure, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "public API mismatch") {
		t.Fatalf("stderr = %q, want public API mismatch", stderr.String())
	}
}

func TestCheckCommandInterfaceOnlyDoesNotRequireMain(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", "--interface-only", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check --interface-only exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestBuildCommandInterfaceOnlyDoesNotRequireMain(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)

	outPath := filepath.Join(dir, "out", "app")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"build", "--interface-only", "--target", "linux-x64", "-o", outPath, srcPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"build --interface-only exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("build --interface-only should not emit %s, stat err=%v", outPath, err)
	}
	if !strings.Contains(stdout.String(), "Interface-only build checked") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

// ---- lsp_test.go ----

func TestLSPCommandSmoke(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio-smoke", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"symbols"`) ||
		!strings.Contains(stdout.String(), `"main"`) {
		t.Fatalf("lsp stdout = %q", stdout.String())
	}
}

func TestLSPCommandSmokeTOONFormat(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio-smoke", srcPath, "--format=toon"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	jsonRaw, err := toon.ConvertTOONToJSON(stdout.Bytes(), toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("TOON smoke output did not decode: %v\n%s", err, stdout.String())
	}
	var analysis compiler.LSPAnalysis
	if err := json.Unmarshal(jsonRaw, &analysis); err != nil {
		t.Fatalf(
			"json.Unmarshal converted TOON: %v\nTOON:\n%s\nJSON:\n%s",
			err,
			stdout.String(),
			jsonRaw,
		)
	}
	if len(analysis.Symbols) == 0 || analysis.Symbols[0].Name != "main" {
		t.Fatalf("TOON smoke analysis missing main symbol: %#v", analysis)
	}
}

func TestLSPStdioRejectsTOONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio", "--format=toon"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "stdio uses framed JSON-RPC") {
		t.Fatalf("stderr missing JSON-RPC boundary explanation: %q", stderr.String())
	}
}

func TestLSPSymbolKindMapsGlobals(t *testing.T) {
	if got := lspSymbolKind("const"); got != 14 {
		t.Fatalf("const symbol kind = %d, want 14", got)
	}
	if got := lspSymbolKind("val"); got != 13 {
		t.Fatalf("val symbol kind = %d, want 13", got)
	}
	if got := lspSymbolKind("var"); got != 13 {
		t.Fatalf("var symbol kind = %d, want 13", got)
	}
}

func TestLSPDocumentSymbolsIncludeDetail(t *testing.T) {
	got := lspDocumentSymbols(compiler.LSPAnalysis{
		Symbols: []compiler.LSPSymbol{{
			Name:   "answer",
			Kind:   "const",
			Line:   1,
			Column: 1,
			Detail: "const answer: Int",
		}},
	})
	if len(got) != 1 {
		t.Fatalf("symbols = %#v", got)
	}
	if got[0]["detail"] != "const answer: Int" {
		t.Fatalf("symbol = %#v", got[0])
	}
}

func TestLSPStdioInitializeAndDidOpen(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n    " +
			"print(\\\"x\\\")\\n    return 0\\n\"}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":1`) || !strings.Contains(out, `"capabilities"`) {
		t.Fatalf("initialize response missing: %q", out)
	}
	if !strings.Contains(out, `"completionProvider"`) {
		t.Fatalf("completion capability missing: %q", out)
	}
	if !strings.Contains(out, `"definitionProvider":true`) {
		t.Fatalf("definition capability missing: %q", out)
	}
	if !strings.Contains(out, `"referencesProvider":true`) {
		t.Fatalf("references capability missing: %q", out)
	}
	if !strings.Contains(out, `"renameProvider":true`) {
		t.Fatalf("rename capability missing: %q", out)
	}
	if !strings.Contains(out, `"documentFormattingProvider":true`) {
		t.Fatalf("document formatting capability missing: %q", out)
	}
	if !strings.Contains(out, `"codeActionProvider":true`) {
		t.Fatalf("code action capability missing: %q", out)
	}
	if !strings.Contains(out, `"method":"textDocument/publishDiagnostics"`) ||
		!strings.Contains(out, `"diagnostics"`) {
		t.Fatalf("diagnostics notification missing: %q", out)
	}
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("shutdown response missing: %q", out)
	}
}

func TestLSPStdioDidOpenPublishesPrivacyConsentDiagnosticCode(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///privacy.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func seal(token: consent.token) " +
			"-> secret.i32\\nuses privacy:\\n    return " +
			"core.secret_seal_i32(1, token)\\n\"}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	var publish map[string]any
	for _, msg := range msgs {
		if method, _ := msg["method"].(string); method == "textDocument/publishDiagnostics" {
			publish = msg
			break
		}
	}
	if publish == nil {
		t.Fatalf("publishDiagnostics notification missing: %#v", msgs)
	}
	params, ok := publish["params"].(map[string]any)
	if !ok {
		t.Fatalf("publishDiagnostics params missing: %#v", publish)
	}
	diagnostics, ok := params["diagnostics"].([]any)
	if !ok || len(diagnostics) == 0 {
		t.Fatalf("publishDiagnostics diagnostics missing: %#v", publish)
	}
	first, ok := diagnostics[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostic entry malformed: %#v", diagnostics[0])
	}
	if code, _ := first["code"].(string); code != compiler.DiagnosticCodeSafetyPrivacy {
		t.Fatalf(
			"diagnostic code = %#v, want %q: %#v",
			first["code"],
			compiler.DiagnosticCodeSafetyPrivacy,
			publish,
		)
	}
}

func TestLSPStdioDidOpenPublishesEffectPolicyDiagnosticCode(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///effect.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n    " +
			"print(\\\"x\\\")\\n    return 0\\n\"}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	var publish map[string]any
	for _, msg := range msgs {
		if method, _ := msg["method"].(string); method == "textDocument/publishDiagnostics" {
			publish = msg
			break
		}
	}
	if publish == nil {
		t.Fatalf("publishDiagnostics notification missing: %#v", msgs)
	}
	params, ok := publish["params"].(map[string]any)
	if !ok {
		t.Fatalf("publishDiagnostics params missing: %#v", publish)
	}
	diagnostics, ok := params["diagnostics"].([]any)
	if !ok || len(diagnostics) == 0 {
		t.Fatalf("publishDiagnostics diagnostics missing: %#v", publish)
	}
	first, ok := diagnostics[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostic entry malformed: %#v", diagnostics[0])
	}
	if code, _ := first["code"].(string); code != compiler.DiagnosticCodeSafetyEffect {
		t.Fatalf(
			"diagnostic code = %#v, want %q: %#v",
			first["code"],
			compiler.DiagnosticCodeSafetyEffect,
			publish,
		)
	}
}

func TestLSPStdioUnknownRequestMethodReturnsJSONRPCError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":99,"method":"tetra/unknown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":100,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[0], 99, -32601, "unknown method")
	if _, ok := msgs[0]["result"]; ok {
		t.Fatalf("unknown method returned success result: %#v", msgs[0])
	}
}

func TestLSPStdioStringRequestIDPreservesCorrelation(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(
		t,
		&input,
		`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{}}`,
	)
	writeLSPTestMessage(
		t,
		&input,
		`{"jsonrpc":"1.0","id":"bad-version","method":"initialize","params":{}}`,
	)
	writeLSPTestMessage(
		t,
		&input,
		`{"jsonrpc":"2.0","id":"bad-1","method":"tetra/unknown","params":{}}`,
	)
	writeLSPTestMessage(
		t,
		&input,
		`{"jsonrpc":"2.0","id":"stop-1","method":"shutdown","params":{}}`,
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestResultObject(t, msgs[0], "init-1")
	assertLSPTestError(t, msgs[1], "bad-version", -32600, "jsonrpc")
	assertLSPTestError(t, msgs[2], "bad-1", -32601, "unknown method")
	assertLSPTestResultNil(t, msgs[3], "stop-1")
}

func TestLSPStdioInvalidJSONRPCVersionReturnsRequestError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"1.0","id":7,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":8,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[0], 7, -32600, "jsonrpc")
}

func TestLSPStdioMalformedRequestParamsReturnsInvalidParamsError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/hover\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":\"bad\",\"character\":0}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[1], 2, -32602, "invalid params")
}

func TestLSPStdioUnopenedDocumentRequestsUseDocumentedEmptyPolicy(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":" +
			"\"textDocument/documentSymbol\",\"params\":{\"textDocument\":" +
			"{\"uri\":\"file:///missing.tetra\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"textDocument/hover\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///missing.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":0}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"textDocument/completion\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///missing.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":0}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"textDocument/definition\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///missing.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":0}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"textDocument/references\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///missing.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":0},\"context\":" +
			"{\"includeDeclaration\":true}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":7,\"method\":\"textDocument/rename\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///missing.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":0},\"newName\":\"value\"}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":8,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestResultArrayLen(t, msgs[1], 2, 0)
	assertLSPTestResultNil(t, msgs[2], 3)
	assertLSPTestResultArrayLen(t, msgs[3], 4, 0)
	assertLSPTestResultNil(t, msgs[4], 5)
	assertLSPTestResultArrayLen(t, msgs[5], 6, 0)
	assertLSPTestResultNil(t, msgs[6], 7)
}

func TestLSPStdioTranscriptFixtureCoversEditingRequests(t *testing.T) {
	var input bytes.Buffer
	for _, body := range loadLSPTranscriptFixture(t, "full_session.jsonl") {
		writeLSPTestMessage(t, &input, body)
	}
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio fixture exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	for _, want := range []string{
		`"id":1`,
		`"id":2`,
		`"selectionRange"`,
		`"id":3`,
		`"contents":{"kind":"markdown","value":"const answer: i32"}`,
		`"id":4`,
		`"label":"answer"`,
		`"id":5`,
		`"start":{"character":6,"line":0}`,
		`"id":6`,
		`"uri":"file:///fixture.tetra"`,
		`"id":7`,
		`"newText":"value"`,
		`"id":8`,
		`"newText":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"`,
		`function 'main' uses effect 'io' but does not declare it`,
		`"id":9`,
		`"title":"Add uses io to function main"`,
		`"id":10`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("fixture transcript output missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
}

func TestLSPStdioCodeActionReturnsMissingUsesQuickFix(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n    " +
			"print(\\\"x\\\")\\n    return 0\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/codeAction\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"range\":{\"start\":{\"line\":1,\"character\":4},\"end\":{\"line\":1," +
			"\"character\":9}},\"context\":{\"diagnostics\":[{\"range\":{\"start\":" +
			"{\"line\":1,\"character\":4},\"end\":{\"line\":1,\"character\":9}}," +
			"\"severity\":1,\"code\":\"TETRA2001\",\"source\":\"tetra\",\"message\":" +
			"\"function 'main' uses effect 'io' but does not declare it\"}]" +
			"}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)

	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("codeAction response missing: %q", out)
	}
	if !strings.Contains(out, `"title":"Add uses io to function main"`) {
		t.Fatalf("codeAction title missing: %q", out)
	}
	if !strings.Contains(out, `"kind":"quickfix"`) {
		t.Fatalf("codeAction kind missing: %q", out)
	}
	if !strings.Contains(out, `"newText":" uses io"`) {
		t.Fatalf("codeAction edit missing insertion text: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":18,"line":0}`) ||
		!strings.Contains(out, `"end":{"character":18,"line":0}`) {
		t.Fatalf("codeAction edit missing insertion range: %q", out)
	}
}

func TestLSPStdioCompletionReturnsOpenDocumentSymbols(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    return answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/completion\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":3,\"character\":11}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"label":"answer"`) ||
		!strings.Contains(out, `"label":"main"`) {
		t.Fatalf("completion response missing expected symbols: %q", out)
	}
	if !strings.Contains(out, `"detail":"const answer: i32"`) {
		t.Fatalf("completion response missing detail: %q", out)
	}
}

func TestLSPStdioDefinitionReturnsOpenDocumentSymbolLocation(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    return answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/definition\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":3,\"character\":11}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"uri":"file:///sample.tetra"`) {
		t.Fatalf("definition response missing location uri: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) ||
		!strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("definition response missing expected symbol range: %q", out)
	}
}

func TestLSPStdioReferencesReturnsOpenDocumentLocations(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    return answer + answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/references\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":3,\"character\":11},\"context\":" +
			"{\"includeDeclaration\":true}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("references response missing: %q", out)
	}
	if got := strings.Count(out, `"uri":"file:///sample.tetra"`); got < 3 {
		t.Fatalf("references response missing locations: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) ||
		!strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("references response missing declaration location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":11,"line":3}`) {
		t.Fatalf("references response missing first usage location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":20,"line":3}`) {
		t.Fatalf("references response missing second usage location: %q", out)
	}
}

func TestLSPStdioReferencesSkipsCommentsAndStrings(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    print(\\\"answer\\\")\\n    // answer is " +
			"documentation only\\n    return answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/references\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":5,\"character\":11},\"context\":" +
			"{\"includeDeclaration\":true}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	result, ok := response["result"].([]any)
	if !ok {
		t.Fatalf("references result is not an array: %#v", response)
	}
	if len(result) != 2 {
		t.Fatalf("references result len = %d, want 2: %#v", len(result), response)
	}
	assertLSPTestLocationsContainRange(t, result, 0, 6)
	assertLSPTestLocationsContainRange(t, result, 5, 11)
	assertLSPTestLocationsDoNotContainRange(t, result, 3, 11)
	assertLSPTestLocationsDoNotContainRange(t, result, 4, 7)
}

func TestLSPStdioRenameReturnsWorkspaceEditForOpenDocument(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    return answer + answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/rename\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":3,\"character\":11},\"newName\":\"value\"}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("rename response missing: %q", out)
	}
	if !strings.Contains(out, `"changes":{"file:///sample.tetra":[`) {
		t.Fatalf("rename workspace edit missing: %q", out)
	}
	if !strings.Contains(out, `"newText":"value"`) {
		t.Fatalf("rename edits missing newText: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) {
		t.Fatalf("rename edits missing declaration location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":11,"line":3}`) {
		t.Fatalf("rename edits missing first usage location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":20,"line":3}`) {
		t.Fatalf("rename edits missing second usage location: %q", out)
	}
}

func TestLSPStdioRenameSkipsCommentsAndStrings(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    print(\\\"answer\\\")\\n    // answer is " +
			"documentation only\\n    return answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/rename\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":5,\"character\":11},\"newName\":\"value\"}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("rename result is not an object: %#v", response)
	}
	changes, ok := result["changes"].(map[string]any)
	if !ok {
		t.Fatalf("rename result missing changes: %#v", response)
	}
	edits, ok := changes["file:///sample.tetra"].([]any)
	if !ok {
		t.Fatalf("rename result missing sample edits: %#v", response)
	}
	if len(edits) != 2 {
		t.Fatalf("rename edit len = %d, want 2: %#v", len(edits), response)
	}
	assertLSPTestLocationsContainRange(t, edits, 0, 6)
	assertLSPTestLocationsContainRange(t, edits, 5, 11)
	assertLSPTestLocationsDoNotContainRange(t, edits, 3, 11)
	assertLSPTestLocationsDoNotContainRange(t, edits, 4, 7)
}

func TestLSPStdioRenameRejectsInvalidNewName(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    return answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/rename\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":3,\"character\":11},\"newName\":\"bad-name\"}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	assertLSPTestError(t, response, 2, -32602, "rename newName must be a Tetra identifier")
	if _, ok := response["result"]; ok {
		t.Fatalf("invalid rename returned success result: %#v", response)
	}
}

func TestLSPStdioRenameRejectsLocalShadowing(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
			"main() -> Int:\\n    let answer: Int = 7\\n    return " +
			"answer\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/rename\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"position\":{\"line\":0,\"character\":7},\"newName\":\"value\"}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	assertLSPTestResultNil(t, response, 2)
}

func TestLSPStdioFormattingReturnsFullDocumentEdit(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n  return " +
			"0\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/formatting\"," +
			"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
			"\"options\":{\"tabSize\":4,\"insertSpaces\":true}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"newText"`) ||
		!strings.Contains(out, `\n    return 0\n`) {
		t.Fatalf("formatting response missing formatted full-document edit: %q", out)
	}
	if !strings.Contains(out, `"end":{"character":0,"line":2}`) {
		t.Fatalf("formatting response missing full document range: %q", out)
	}
}

func TestLSPStdioDidChangePublishesUpdatedDiagnostics(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n    return " +
			"0\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didChange\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"version\":2}," +
			"\"contentChanges\":[{\"text\":\"func main() -> Int:\\n    " +
			"print(\\\"x\\\")\\n    return 0\\n\"}]}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
	if !strings.Contains(out, `function 'main' uses effect 'io' but does not declare it`) {
		t.Fatalf("updated diagnostic missing: %q", out)
	}
}

func TestLSPStdioDidCloseClearsDiagnostics(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
			"\"tetra\",\"version\":1,\"text\":\"func main() -> Int:\\n    " +
			"print(\\\"x\\\")\\n    return 0\\n\"}}}"),
	)
	writeLSPTestMessage(
		t,
		&input,
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didClose\",\"params\":" +
			"{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}}}"),
	)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"lsp stdio exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	out := stdout.String()
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
	if !strings.Contains(out, `function 'main' uses effect 'io' but does not declare it`) {
		t.Fatalf("initial diagnostic missing: %q", out)
	}
	if !strings.Contains(out, `"diagnostics":[]`) {
		t.Fatalf("didClose did not publish empty diagnostics: %q", out)
	}
}

func writeLSPTestMessage(t *testing.T, w *bytes.Buffer, body string) {
	t.Helper()
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func readLSPTestMessages(t *testing.T, transcript string) []map[string]any {
	t.Helper()
	reader := bufio.NewReader(strings.NewReader(transcript))
	var msgs []map[string]any
	for {
		body, err := readLSPMessage(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read LSP response: %v\n%s", err, transcript)
		}
		var msg map[string]any
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Fatalf("decode LSP response %q: %v", string(body), err)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func assertLSPTestError(t *testing.T, msg map[string]any, id any, code int, messagePart string) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	errObj, ok := msg["error"].(map[string]any)
	if !ok {
		t.Fatalf("message error missing: %#v", msg)
	}
	if got := int(errObj["code"].(float64)); got != code {
		t.Fatalf("error code = %d, want %d: %#v", got, code, msg)
	}
	if got, _ := errObj["message"].(string); !strings.Contains(got, messagePart) {
		t.Fatalf("error message = %q, want containing %q: %#v", got, messagePart, msg)
	}
}

func assertLSPTestResultArrayLen(t *testing.T, msg map[string]any, id any, want int) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	result, ok := msg["result"].([]any)
	if !ok {
		t.Fatalf("message result is not an array: %#v", msg)
	}
	if len(result) != want {
		t.Fatalf("result len = %d, want %d: %#v", len(result), want, msg)
	}
}

func assertLSPTestResultNil(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	if result, ok := msg["result"]; !ok || result != nil {
		t.Fatalf("message result = %#v, want nil: %#v", result, msg)
	}
}

func assertLSPTestResultObject(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	if _, ok := msg["result"].(map[string]any); !ok {
		t.Fatalf("message result is not an object: %#v", msg)
	}
}

func lspTestMessageByID(t *testing.T, msgs []map[string]any, id any) map[string]any {
	t.Helper()
	for _, msg := range msgs {
		switch want := id.(type) {
		case int:
			got, ok := msg["id"].(float64)
			if ok && int(got) == want {
				return msg
			}
		case string:
			got, ok := msg["id"].(string)
			if ok && got == want {
				return msg
			}
		default:
			t.Fatalf("unsupported test id type %T", id)
		}
	}
	t.Fatalf("message id %v not found in %#v", id, msgs)
	return nil
}

func assertLSPTestLocationsContainRange(t *testing.T, locations []any, line int, character int) {
	t.Helper()
	if !lspTestLocationsContainRange(locations, line, character) {
		t.Fatalf(
			"locations missing range start line=%d character=%d: %#v",
			line,
			character,
			locations,
		)
	}
}

func assertLSPTestLocationsDoNotContainRange(
	t *testing.T,
	locations []any,
	line int,
	character int,
) {
	t.Helper()
	if lspTestLocationsContainRange(locations, line, character) {
		t.Fatalf(
			"locations unexpectedly include range start line=%d character=%d: %#v",
			line,
			character,
			locations,
		)
	}
}

func lspTestLocationsContainRange(locations []any, line int, character int) bool {
	for _, item := range locations {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rangeObj, ok := obj["range"].(map[string]any)
		if !ok {
			continue
		}
		start, ok := rangeObj["start"].(map[string]any)
		if !ok {
			continue
		}
		gotLine, lineOK := start["line"].(float64)
		gotCharacter, characterOK := start["character"].(float64)
		if lineOK && characterOK && int(gotLine) == line && int(gotCharacter) == character {
			return true
		}
	}
	return false
}

func assertLSPTestID(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	switch want := id.(type) {
	case int:
		got, ok := msg["id"].(float64)
		if !ok || int(got) != want {
			t.Fatalf("message id = %#v, want %d: %#v", msg["id"], want, msg)
		}
	case string:
		got, ok := msg["id"].(string)
		if !ok || got != want {
			t.Fatalf("message id = %#v, want %q: %#v", msg["id"], want, msg)
		}
	default:
		t.Fatalf("unsupported test id type %T", id)
	}
}

func loadLSPTranscriptFixture(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "lsp", name))
	if err != nil {
		t.Fatalf("read LSP fixture: %v", err)
	}
	var bodies []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		bodies = append(bodies, line)
	}
	if len(bodies) == 0 {
		t.Fatalf("LSP fixture %s is empty", name)
	}
	return bodies
}

// ---- lsp_wire_test.go ----

func TestReadLSPMessageRejectsTooLargeContentLength(t *testing.T) {
	reader := bufio.NewReader(
		strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n", maxLSPContentLength+1)),
	)

	body, err := readLSPMessage(reader)
	if err == nil {
		t.Fatalf("readLSPMessage err = nil, body length = %d", len(body))
	}
	if !strings.Contains(err.Error(), "Content-Length too large") {
		t.Fatalf("readLSPMessage err = %q, want Content-Length too large", err.Error())
	}
}

func TestReadLSPMessageReadsNormalContentLength(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("Content-Length: 15\r\n\r\n{\"jsonrpc\":\"2\"}"))

	body, err := readLSPMessage(reader)
	if err != nil {
		t.Fatalf("readLSPMessage err = %v", err)
	}
	if got, want := string(body), `{"jsonrpc":"2"}`; got != want {
		t.Fatalf("readLSPMessage body = %q, want %q", got, want)
	}
}

// ---- metadata_test.go ----

func TestTargetsCommandText(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"targets"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("targets exit code = %d, stdout=%q", code, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Supported targets:",
		"linux-x64",
		"windows-x64",
		"macos-x64",
		"wasm32-wasi",
		"wasm32-web",
		"Build-only targets:",
		"linux-x86",
		"linux-x32",
		"Planned targets:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("targets output missing %q:\n%s", want, out)
		}
	}
}

func TestTargetsCommandJSON(t *testing.T) {
	restoreHost := stubLinuxX32HostSupport(false)
	defer restoreHost()

	type targetMeta struct {
		Triple                   string   `json:"triple"`
		Status                   string   `json:"status"`
		OS                       string   `json:"os"`
		Arch                     string   `json:"arch"`
		ABI                      string   `json:"abi"`
		DataModel                string   `json:"data_model"`
		Format                   string   `json:"format"`
		ExeExt                   string   `json:"exe_ext"`
		BuildOnly                bool     `json:"build_only"`
		RunMode                  string   `json:"run_mode"`
		RunRunner                string   `json:"run_runner,omitempty"`
		RunSupported             bool     `json:"run_supported"`
		RunUnsupportedReason     string   `json:"run_unsupported_reason"`
		PointerWidthBits         int      `json:"pointer_width_bits"`
		RegisterWidthBits        int      `json:"register_width_bits"`
		NativeIntWidthBits       int      `json:"native_int_width_bits"`
		Endian                   string   `json:"endian"`
		StackAlignmentBytes      int      `json:"stack_alignment_bytes"`
		MaxAtomicWidthBits       int      `json:"max_atomic_width_bits"`
		AtomicWidthBits          []int    `json:"atomic_width_bits"`
		AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits"`
		UnsupportedReason        string   `json:"unsupported_reason"`
		RuntimeStatus            string   `json:"runtime_status"`
		StdlibStatus             string   `json:"stdlib_status"`
		FFIStatus                string   `json:"ffi_status"`
		MemoryBuild              string   `json:"memory_build"`
		MemoryLower              string   `json:"memory_lower"`
		MemoryRun                string   `json:"memory_run"`
		MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics"`
		MemoryRegionLowering     string   `json:"memory_region_lowering"`
		MemoryAlignmentSemantics string   `json:"memory_alignment_semantics"`
		MemoryClaimLevel         string   `json:"memory_claim_level"`
		RunnerProbeCommand       string   `json:"runner_probe_command"`
		ReleaseGate              string   `json:"release_gate"`
		EvidenceArtifacts        []string `json:"evidence_artifacts"`
		SyscallInstruction       string   `json:"syscall_instruction"`
		SyscallNumbering         string   `json:"syscall_numbering"`
		SyscallArgRegisters      []string `json:"syscall_arg_registers"`
		SyscallErrorRange        string   `json:"syscall_error_range"`
		SupportsDebugInfo        bool     `json:"supports_debug_info"`
		SupportsReleaseOptimize  bool     `json:"supports_release_optimize"`
	}
	var report struct {
		Supported []string     `json:"supported"`
		BuildOnly []string     `json:"build_only"`
		Planned   []string     `json:"planned"`
		Targets   []targetMeta `json:"targets"`
	}
	runCLIJSONStdout(t, []string{"targets", "--format=json"}, 0, &report)
	if strings.Join(
		report.Supported,
		",",
	) != "linux-x64,windows-x64,macos-x64,wasm32-wasi,wasm32-web" {
		t.Fatalf("supported targets = %#v", report.Supported)
	}
	if strings.Join(report.BuildOnly, ",") != "linux-x86,linux-x32" {
		t.Fatalf("build-only targets = %#v", report.BuildOnly)
	}
	if len(report.Planned) != 0 {
		t.Fatalf("planned targets = %#v", report.Planned)
	}
	if len(report.Targets) != 7 {
		t.Fatalf("targets metadata count = %d, want 7: %#v", len(report.Targets), report.Targets)
	}
	byTriple := map[string]targetMeta{}
	for _, tgt := range report.Targets {
		if byTriple[tgt.Triple].Triple != "" {
			t.Fatalf("duplicate target metadata for %s in %#v", tgt.Triple, report.Targets)
		}
		byTriple[tgt.Triple] = tgt
	}
	for _, triple := range append(append([]string{}, report.Supported...), report.BuildOnly...) {
		if byTriple[triple].Triple == "" {
			t.Fatalf("target metadata missing %s in %#v", triple, report.Targets)
		}
	}
	if got := byTriple["linux-x64"]; got.Status != "supported" || got.OS != "linux" ||
		got.Arch != "x64" ||
		got.ABI != "sysv" ||
		got.DataModel != "lp64" ||
		got.Format != "elf" ||
		got.PointerWidthBits != 64 ||
		got.RegisterWidthBits != 64 ||
		got.NativeIntWidthBits != 64 ||
		got.AtomicPointerWidthBits != 64 ||
		!reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32, 64}) ||
		got.Endian != "little" ||
		got.BuildOnly ||
		!got.SupportsDebugInfo ||
		!got.SupportsReleaseOptimize {
		t.Fatalf("linux-x64 metadata = %#v", got)
	} else {
		promotionOK := got.RuntimeStatus == "production" &&
			got.StdlibStatus == "production" &&
			got.FFIStatus == "scalar_object_smokes_partial" &&
			got.RunnerProbeCommand != "" &&
			got.ReleaseGate == "scripts/release/post_v0_4/linux-native-targets-smoke.sh" &&
			hasString(got.EvidenceArtifacts, "linux-x64-runner.json")
		if !promotionOK {
			t.Fatalf("linux-x64 promotion metadata = %#v", got)
		}
		memoryOK := got.MemoryBuild == "yes" &&
			got.MemoryLower == "yes" &&
			got.MemoryRun == "yes" &&
			got.MemoryRawDiagnostics == "yes" &&
			got.MemoryRegionLowering == "yes/partial" &&
			got.MemoryAlignmentSemantics == "yes" &&
			got.MemoryClaimLevel == "production/host_runtime"
		if !memoryOK {
			t.Fatalf("linux-x64 memory capability metadata = %#v", got)
		}
		syscallOK := got.SyscallInstruction == "syscall" &&
			got.SyscallNumbering == "x86_64" &&
			reflect.DeepEqual(
				got.SyscallArgRegisters,
				[]string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
			) &&
			got.SyscallErrorRange == "-4095..-1"
		if !syscallOK {
			t.Fatalf("linux-x64 syscall metadata = %#v", got)
		}
	}
	if got := byTriple["windows-x64"]; got.Status != "supported" || got.OS != "windows" ||
		got.ABI != "win64" ||
		got.DataModel != "llp64" ||
		got.Format != "pe" ||
		got.ExeExt != ".exe" ||
		got.PointerWidthBits != 64 ||
		got.RegisterWidthBits != 64 ||
		!got.SupportsDebugInfo ||
		!got.SupportsReleaseOptimize {
		t.Fatalf("windows-x64 metadata = %#v", got)
	}
	if got := byTriple["linux-x86"]; got.Status != "build_only" || got.OS != "linux" ||
		got.Arch != "x86" ||
		got.ABI != "i386-sysv" ||
		got.DataModel != "ilp32" ||
		got.PointerWidthBits != 32 ||
		got.RegisterWidthBits != 32 ||
		got.NativeIntWidthBits != 32 ||
		got.AtomicPointerWidthBits != 32 ||
		got.MaxAtomicWidthBits != 32 ||
		!reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32}) ||
		got.RunMode != "host_probed" ||
		!got.BuildOnly ||
		!strings.Contains(got.UnsupportedReason, "not implemented yet") ||
		!strings.Contains(got.UnsupportedReason, "executable build/link") ||
		!strings.Contains(got.UnsupportedReason, "run/test execution") ||
		!strings.Contains(got.UnsupportedReason, "stdout write/string literal data") ||
		!strings.Contains(got.UnsupportedReason, "stack-argument") ||
		!strings.Contains(got.UnsupportedReason, "scalar global") ||
		!strings.Contains(got.UnsupportedReason, "symbol-backed callback") ||
		!strings.Contains(got.UnsupportedReason, "heap-backed slice allocation/indexing") ||
		!strings.Contains(got.UnsupportedReason, "raw ptr_add/load/store") ||
		!strings.Contains(got.UnsupportedReason, "MMIO read/write") ||
		!strings.Contains(got.UnsupportedReason, "scoped island bump allocation/free") ||
		!strings.Contains(got.UnsupportedReason, "debug double-free guard/page-protect") {
		t.Fatalf("linux-x86 metadata = %#v", got)
	} else {
		promotionOK := got.RuntimeStatus == "partial_build_only" &&
			got.StdlibStatus == "partial_build_only" &&
			got.FFIStatus == "ilp32_scalar_object_smokes_partial" &&
			strings.Contains(got.RunnerProbeCommand, "--target x86") &&
			got.ReleaseGate == "scripts/release/post_v0_4/linux-native-targets-smoke.sh" &&
			hasString(got.EvidenceArtifacts, "linux-x86-runner.json")
		if !promotionOK {
			t.Fatalf("linux-x86 promotion metadata = %#v", got)
		}
		memoryOK := got.MemoryBuild == "yes" &&
			got.MemoryLower == "yes" &&
			got.MemoryRun == "no/host-dependent" &&
			got.MemoryRawDiagnostics == "partial" &&
			got.MemoryRegionLowering == "partial" &&
			got.MemoryAlignmentSemantics == "partial" &&
			got.MemoryClaimLevel == "build_lower_only"
		if !memoryOK {
			t.Fatalf("linux-x86 memory capability metadata = %#v", got)
		}
		if got.SyscallInstruction != "int 0x80" || got.SyscallNumbering != "i386" || !reflect.DeepEqual(
			got.SyscallArgRegisters,
			[]string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"},
		) || got.SyscallErrorRange != "-4095..-1" {
			t.Fatalf("linux-x86 syscall metadata = %#v", got)
		}
		requireUnsupportedReasonContains(t, "linux-x86", got.UnsupportedReason, []string{
			"i386 SysV ABI classifier",
			"self-host logical time runtime smoke",
			"fs_exists filesystem runtime plus filesystem/scheduler composition smoke",
			"bounded two-spawn self-host actors/task/task-group runtime smokes",
			"single-spawn typed-task/staged typed-task/typed task-group plus actor-state runtime smoke",
			"i386 ctx_switch object smoke",
			("current core.net networking runtime smokes, Surface, " +
				"distributed actors, and actor fanout above 2 runtime " +
				"boundary diagnostics"),
			("x86 canonical ptr/rawptr/nullable_ptr/ref, c_int/c_uint, " +
				"and complete ILP32 native/libc scalar @export object smokes"),
			"x86 function-pointer @export diagnostics",
			"remaining source target-layout scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"source-level atomic diagnostics",
		})
		if got.RunSupported {
			if got.RunUnsupportedReason != "" {
				t.Fatalf("linux-x86 supported host-probed metadata = %#v", got)
			}
		} else if !strings.Contains(
			got.RunUnsupportedReason,
			"does not support Linux i386 execution",
		) || !strings.Contains(
			got.RunUnsupportedReason,
			"no host fallback",
		) {
			t.Fatalf("linux-x86 unsupported host-probed metadata = %#v", got)
		}
	}
	if got := byTriple["linux-x32"]; got.Status != "build_only" || got.OS != "linux" ||
		got.Arch != "x64" ||
		got.ABI != "x32-sysv" ||
		got.DataModel != "x32" ||
		got.PointerWidthBits != 32 ||
		got.RegisterWidthBits != 64 ||
		got.NativeIntWidthBits != 32 ||
		got.AtomicPointerWidthBits != 32 ||
		!reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32, 64}) ||
		got.RunMode != "host_probed" ||
		!got.BuildOnly ||
		!strings.Contains(
			got.UnsupportedReason,
			"full linux-x32 runtime/stdlib/FFI support is not implemented yet",
		) ||
		!strings.Contains(got.UnsupportedReason, "executable build/link") ||
		!strings.Contains(got.UnsupportedReason, "object codegen") ||
		!strings.Contains(got.UnsupportedReason, "self-host runtime builds") ||
		!strings.Contains(got.UnsupportedReason, "compiler-owned target suites") ||
		!strings.Contains(got.UnsupportedReason, "host-probed source run/test execution") ||
		!strings.Contains(got.UnsupportedReason, "Linux kernel supports the x32 ABI") {
		t.Fatalf("linux-x32 metadata = %#v", got)
	} else {
		promotionOK := got.RuntimeStatus == "partial_build_only" &&
			got.StdlibStatus == "partial_build_only" &&
			got.FFIStatus == "ilp32_scalar_object_smokes_partial" &&
			strings.Contains(got.RunnerProbeCommand, "--target x32") &&
			got.ReleaseGate == "scripts/release/post_v0_4/linux-native-targets-smoke.sh" &&
			hasString(got.EvidenceArtifacts, "linux-x32-runner.json")
		if !promotionOK {
			t.Fatalf("linux-x32 promotion metadata = %#v", got)
		}
		memoryOK := got.MemoryBuild == "yes" &&
			got.MemoryLower == "yes" &&
			got.MemoryRun == "no/host-dependent" &&
			got.MemoryRawDiagnostics == "partial" &&
			got.MemoryRegionLowering == "partial" &&
			got.MemoryAlignmentSemantics == "special" &&
			got.MemoryClaimLevel == "build_lower_only"
		if !memoryOK {
			t.Fatalf("linux-x32 memory capability metadata = %#v", got)
		}
		syscallOK := got.SyscallInstruction == "syscall" &&
			got.SyscallNumbering == "x32_syscall_bit" &&
			reflect.DeepEqual(
				got.SyscallArgRegisters,
				[]string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
			) &&
			got.SyscallErrorRange == "-4095..-1"
		if !syscallOK {
			t.Fatalf("linux-x32 syscall metadata = %#v", got)
		}
		requireUnsupportedReasonContains(t, "linux-x32", got.UnsupportedReason, []string{
			"x32 SysV ABI classifier",
			"stdout write/string literal data",
			"raw ptr_add/load/store",
			"pointer load/store",
			"MMIO read/write",
			"scoped island bump allocation/free",
			("self-host runtime builds for time, bounded two-spawn " +
				"actors/task/task-group, single-spawn typed-task/staged " +
				"typed-task/typed task-group, actor-state, and " +
				"filesystem/scheduler composition smokes"),
			"x32 ctx_switch object smoke",
			"fs_exists-only filesystem runtime smoke",
			("current x32 core.net networking runtime smokes, Surface, " +
				"distributed actors, and x32 actor fanout above 2 runtime " +
				"boundary diagnostics"),
			("scalar i32 plus canonical ptr/rawptr/nullable_ptr/ref, " +
				"c_int/c_uint, and complete ILP32 native/libc scalar @export " +
				"object smokes"),
			"x32 function-pointer @export diagnostics",
			"remaining source target-layout scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"dword pointer atomics",
			"x32 syscall numbers",
		})
		if got.RunSupported {
			if got.RunUnsupportedReason != "" {
				t.Fatalf("linux-x32 supported host-probed metadata = %#v", got)
			}
		} else {
			requireLinuxX32HostUnsupportedReason(t, got.RunUnsupportedReason)
		}
	}
	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		got := byTriple[triple]
		if got.Status != "supported" || got.Arch != "wasm32" || got.DataModel != "ilp32" ||
			got.PointerWidthBits != 32 ||
			got.Format != "wasm" ||
			got.ExeExt != ".wasm" ||
			got.BuildOnly ||
			got.SupportsDebugInfo ||
			!got.SupportsReleaseOptimize {
			t.Fatalf("%s metadata = %#v", triple, got)
		}
		wantRun := "runner-smoke if available"
		if triple == "wasm32-web" {
			wantRun = "browser-smoke if available"
		}
		if got.MemoryBuild != "yes" || got.MemoryLower != "yes" || got.MemoryRun != wantRun ||
			got.MemoryRawDiagnostics != "safe-only" ||
			got.MemoryRegionLowering != "limited" ||
			got.MemoryAlignmentSemantics != "wasm rules" ||
			got.MemoryClaimLevel != "artifact/runtime tiered" {
			t.Fatalf("%s memory capability metadata = %#v", triple, got)
		}
	}
	if got := byTriple["wasm32-wasi"]; got.RunMode != "wasi_runner" {
		t.Fatalf("wasm32-wasi runner metadata = %#v", got)
	}
	if got := byTriple["wasm32-web"]; got.RunMode != "web_runner" {
		t.Fatalf("wasm32-web runtime metadata = %#v", got)
	} else if got.RunSupported {
		if got.RunRunner == "" || got.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-web supported runner metadata = %#v", got)
		}
	} else if got.RunRunner != "" || !strings.Contains(
		got.RunUnsupportedReason,
		"runner unavailable",
	) {
		t.Fatalf("wasm32-web unsupported runner metadata = %#v", got)
	}
}

func TestTargetsCommandJSONMarksWASIRunSupportedWhenRunnerExists(t *testing.T) {
	restore := stubLookPath(func(name string) (string, error) {
		if name == "wasmtime" {
			return "/usr/bin/wasmtime", nil
		}
		if name == "chromium" {
			return "/usr/bin/chromium", nil
		}
		return "", exec.ErrNotFound
	})
	defer restore()

	report := targetsJSONForTest(t)
	wasm := targetMetaForTest(t, report, "wasm32-wasi")
	if wasm.BuildOnly || wasm.RunMode != "wasi_runner" || wasm.RunRunner != "wasmtime" ||
		!wasm.RunSupported ||
		wasm.RunUnsupportedReason != "" {
		t.Fatalf("wasm32-wasi metadata with runner = %#v", wasm)
	}
	web := targetMetaForTest(t, report, "wasm32-web")
	if web.BuildOnly || web.RunMode != "web_runner" || web.RunRunner != "/usr/bin/chromium" ||
		!web.RunSupported ||
		web.RunUnsupportedReason != "" {
		t.Fatalf("wasm32-web metadata with browser runner = %#v", web)
	}
}

func TestTargetsCommandJSONMarksWASIRunUnsupportedWhenRunnerMissing(t *testing.T) {
	restore := stubLookPath(func(name string) (string, error) {
		return "", exec.ErrNotFound
	})
	defer restore()

	report := targetsJSONForTest(t)
	wasm := targetMetaForTest(t, report, "wasm32-wasi")
	if wasm.BuildOnly || wasm.RunMode != "wasi_runner" || wasm.RunRunner != "" ||
		wasm.RunSupported ||
		!strings.Contains(wasm.RunUnsupportedReason, "missing WASI runner") {
		t.Fatalf("wasm32-wasi metadata without runner = %#v", wasm)
	}
}

func TestTargetsCommandTOON(t *testing.T) {
	var report targetsJSONReportForTest
	raw := runCLITOONStdout(t, []string{"targets", "--format=toon"}, 0, &report)
	if !strings.Contains(raw, "targets[") || !strings.Contains(raw, "supported[") {
		t.Fatalf("targets TOON output missing structured fields:\n%s", raw)
	}
	if len(report.Targets) != 7 {
		t.Fatalf("targets metadata count = %d, want 7", len(report.Targets))
	}
	if targetMetaForTest(t, report, "linux-x64").RunMode == "" {
		t.Fatalf("linux-x64 TOON target metadata incomplete: %#v", report)
	}
}

type targetMetaJSONForTest struct {
	Triple               string `json:"triple"`
	BuildOnly            bool   `json:"build_only"`
	RunMode              string `json:"run_mode"`
	RunRunner            string `json:"run_runner,omitempty"`
	RunSupported         bool   `json:"run_supported"`
	RunUnsupportedReason string `json:"run_unsupported_reason"`
}

type targetsJSONReportForTest struct {
	Targets []targetMetaJSONForTest `json:"targets"`
}

func targetsJSONForTest(t *testing.T) targetsJSONReportForTest {
	t.Helper()
	var report targetsJSONReportForTest
	runCLIJSONStdout(t, []string{"targets", "--format=json"}, 0, &report)
	return report
}

func targetMetaForTest(
	t *testing.T,
	report targetsJSONReportForTest,
	triple string,
) targetMetaJSONForTest {
	t.Helper()
	for _, target := range report.Targets {
		if target.Triple == triple {
			return target
		}
	}
	t.Fatalf("missing target metadata for %s in %#v", triple, report.Targets)
	return targetMetaJSONForTest{}
}

func requireUnsupportedReasonContains(t *testing.T, triple string, reason string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(reason, want) {
			t.Fatalf("%s unsupported_reason missing %q: %q", triple, want, reason)
		}
	}
}

func hasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestTargetsCommandRejectsUnsupportedFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"targets", "--format=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("targets exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFeaturesCommandJSON(t *testing.T) {
	var report struct {
		Schema   string `json:"schema"`
		Version  string `json:"version"`
		Features []struct {
			ID        string   `json:"id"`
			Name      string   `json:"name"`
			Status    string   `json:"status"`
			Since     string   `json:"since"`
			Scope     string   `json:"scope"`
			Stability string   `json:"stability"`
			Docs      []string `json:"docs"`
		} `json:"features"`
	}
	runCLIJSONStdout(t, []string{"features", "--format=json"}, 0, &report)
	if report.Schema != "tetra.features.v1" {
		t.Fatalf("features schema = %q", report.Schema)
	}
	if report.Version != compiler.Version() {
		t.Fatalf("features version = %q, want %q", report.Version, compiler.Version())
	}
	statusByID := map[string]string{}
	statusSeen := map[string]bool{}
	for _, feature := range report.Features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" ||
			feature.Stability == "" ||
			len(feature.Docs) == 0 {
			t.Fatalf("feature missing required metadata: %#v", feature)
		}
		statusByID[feature.ID] = feature.Status
		statusSeen[feature.Status] = true
		if feature.ID == "language.enum-payload-match" {
			if feature.Status != "current" || feature.Since != "v0.3.0" {
				t.Fatalf(
					"enum payload feature lifecycle = status %q since %q, want current since v0.3.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"positional enum payload constructors",
				"match/catch/if-let",
				"exhaustive unguarded enum match/catch",
				"nested destructuring patterns",
				"guard expansion remain future/post-v1",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("enum payload feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.protocol-bound-generics-static" {
			if feature.Status != "current" || feature.Since != "v0.3.0" {
				t.Fatalf(
					"protocol-bound generics lifecycle = status %q since %q, want current since v0.3.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"validated statically during monomorphization",
				"same-module and cross-module impl conformance",
				"visibility diagnostics",
				"calling protocol requirements through generic bounds",
				"dynamic dispatch remain unsupported",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"protocol-bound generics feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "language.generics-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"generics MVP lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"statically monomorphized",
				"no runtime generic values or dynamic dispatch",
				"generic structs",
				"future/post-v1",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("generics MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.protocol-conformance-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"protocol conformance MVP lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"checked statically",
				"generic requirement signature shape",
				"no witness tables",
				"dynamic dispatch remain post-v1",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"protocol conformance MVP feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "language.callable-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"callable MVP lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"Level 0 callable surface",
				"symbol-backed non-capturing callable paths",
				"full first-class function values remain out of scope",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.callable-level1" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"callable Level 1 lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production non-capturing symbol-backed callable Level 1",
				"function-typed locals, aliases, callbacks",
				"signature-compatible mutable local reassignment",
				"captured closure escape beyond the fnptr Level 2 slice",
				"full first-class function values remain out of scope",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable Level 1 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.ownership-markers-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"ownership markers MVP lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"conservative borrow/inout/consume marker checks",
				("same-module/cross-module struct-field and enum-payload " +
					"partial consume with whole-value call/let/return and enum " +
					"wrapper-constructor rejection"),
				("borrow escape diagnostics for scalar ptr including " +
					"same-module/cross-module scalar ptr consume and inout " +
					"assignment"),
				("same-module/cross-module borrowed scalar ptr escapes " +
					"through ptr-containing struct inout assignment"),
				("same-module/cross-module fixed-array alias return plus " +
					"direct global assignment, optional global assignment, and " +
					"inout assignment escapes with stable TETRA2102 diagnostic " +
					"evidence"),
				"borrowed string alias return/global assignment escapes",
				"ptr/slice optional assignment return/owned/consume/inout escape",
				("same-module/cross-module direct slice global assignment " +
					"with stable TETRA2102 JSON diagnostic evidence"),
				("same-module/cross-module optional ptr global assignment " +
					"with stable TETRA2102 JSON diagnostic evidence"),
				("same-module/cross-module optional aggregate global " +
					"assignment with stable TETRA2102 JSON diagnostic evidence"),
				"ptr optional assignment if-let/match global escape",
				("same-module/cross-module ptr-containing/nested aggregate " +
					"owned/consume/inout call rejections with stable TETRA2101 " +
					"JSON diagnostic evidence"),
				("same-module/cross-module ptr enum-payload " +
					"owned/consume/inout call rejections with stable TETRA2101 " +
					"JSON diagnostic evidence"),
				("same-module/cross-module ptr optional-payload " +
					"owned/consume/inout call rejections with stable TETRA2101 " +
					"JSON diagnostic evidence"),
				("same-module/cross-module slice optional-payload " +
					"owned/consume/inout call rejections with stable TETRA2101 " +
					"JSON diagnostic evidence"),
				("imported direct ptr-containing/nested aggregate " +
					"owned/consume/inout call rejections with stable TETRA2101 " +
					"JSON diagnostic evidence"),
				("same-module/cross-module function-typed " +
					"value/struct-field/enum-payload optional-ptr " +
					"owned/consume/inout callback diagnostics with stable " +
					"TETRA2101 CLI JSON evidence"),
				("function-typed value/struct-field/enum-payload callback " +
					"slice-containing struct/enum owned/consume/inout call " +
					"rejections with stable TETRA2101 JSON diagnostic evidence"),
				("same-module/cross-module generic aggregate and optional-ptr " +
					"owned/consume/inout instantiations including " +
					"slice-containing struct/enum aggregate instantiations with " +
					"stable TETRA2101 CLI JSON evidence"),
				("same-module/cross-module generic " +
					"borrow-aggregate/optional-ptr return diagnostics with " +
					"stable TETRA2102 CLI JSON evidence"),
				("same-module/cross-module protocol parameter ownership " +
					"matching plus same-module/cross-module protocol impl " +
					"parameter ownership mismatch diagnostics with stable " +
					"TETRA2001 CLI JSON evidence"),
				("same-module/cross-module generic protocol requirement " +
					"parameter ownership mismatch diagnostics with stable " +
					"TETRA2001 JSON diagnostic evidence"),
				"use-after-consume",
				"not a full SSA lifetime solver",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"ownership markers MVP feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "language.resource-lifetime-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"resource lifetime MVP lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"conservative resource finalization checks",
				"task handles",
				"island handles",
				("stable ownership safety JSON diagnostics for resource " +
					"use-after-free, double-join, and ambiguous-provenance cases"),
				("same-module/cross-module task-group " +
					"struct-field/enum-payload alias close diagnostics with " +
					"stable TETRA2101 JSON diagnostic evidence"),
				("same-module/cross-module enum-constructor return resource " +
					"aliases with stable TETRA2101 CLI JSON evidence"),
				("same-module/cross-module monomorphized generic struct " +
					"task-handle/task-group/island resource aliases with stable " +
					"TETRA2101 CLI JSON evidence"),
				("same-module/cross-module task-handle/task-group " +
					"if-let/match optional-payload join/close aliases with " +
					"stable TETRA2101 CLI JSON evidence"),
				("same-module/cross-module transitive interprocedural " +
					"task-handle/task-group/island resource aliases with stable " +
					"TETRA2101 CLI JSON evidence"),
				"same-module/cross-module island whole-optional use-after-payload-free diagnostics",
				"double-use",
				"ambiguous provenance",
				"not a full SSA lifetime solver",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"resource lifetime MVP feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "actors.task-transfer-safety" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf(
					"actor/task transfer safety lifecycle = status %q since %q, want current since v0.2.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"conservative actor/task ownership transfer checks",
				"worker entrypoints",
				"branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence",
				"actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence",
				"island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence",
				("same-module/cross-module transitive actor consume alias " +
					"diagnostics with stable TETRA2101 CLI JSON evidence"),
				("same-module/cross-module monomorphized generic struct actor " +
					"consume alias diagnostics with stable TETRA2101 CLI JSON " +
					"evidence"),
				("same-module/cross-module task_group_cancel return " +
					"provenance diagnostics with stable TETRA2101 CLI JSON " +
					"evidence"),
				("same-module/cross-module actor struct-field/enum-payload " +
					"alias transfer diagnostics with stable TETRA2101 JSON " +
					"diagnostic evidence"),
				("same-module/cross-module actor/task if-let/match " +
					"optional-payload alias transfer diagnostics with stable " +
					"TETRA2101 JSON diagnostic evidence"),
				("same-module/cross-module task-handle " +
					"struct-field/enum-payload alias transfer diagnostics with " +
					"stable TETRA2101 JSON diagnostic evidence"),
				"conservative local MVP",
				"distributed actors",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("actor/task transfer feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "actors.distributed-runtime" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"distributed actor runtime lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, wantDoc := range []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/actors.md",
				"docs/user/platform/async_actors_guide.md",
			} {
				if !containsString(feature.Docs, wantDoc) {
					t.Fatalf("distributed actor runtime docs missing %s: %#v", wantDoc, feature)
				}
			}
			for _, want := range []string{
				"production Linux-x64 distributed actor runtime path",
				"actornet loopback TCP broker",
				"distributed node identity",
				"remote actor handles",
				"network mailbox send/receive",
				"i32, tagged, and typed frames",
				"missing-node failure/status propagation",
				"task cancel/join handles",
				"tetra.actors.distributed-runtime.v1 smoke evidence",
				"transport-only or fake reports",
				"non-Linux-x64 targets",
				"broader structured-concurrency guarantees",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"distributed actor runtime feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "language.lifetime-ssa" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"lifetime SSA lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production SSA-like local lifetime join analysis",
				"ownership consume state",
				"resource finalization state",
				"optional region-wrapper escapes",
				"maybe-consumed diagnostics",
				"richer interprocedural lifetime proofs",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("lifetime SSA feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.callable-level2" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"callable Level 2 lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production captured closure Level 2 slice",
				"fnptr-backed function-typed locals",
				"function-typed returns",
				"immutable local struct fields or enum payloads",
				"larger immutable environments are promoted under " +
					"language.full-first-class-callables",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable Level 2 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.full-first-class-callables" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"full first-class callable lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production first-class callable/function-value semantics",
				"fixed 4-slot callable handle",
				"larger immutable Int/Bool/String/simple-aggregate captures",
				"synchronous callback arguments",
				"cross-module returned values",
				"stable JSON diagnostics for mutable by-reference captures " +
					"including callable mutable-capture global-escape",
				"callable mutable-capture heap-escape",
				"callable pointer/resource capture escape",
				"function-typed storage/return unsupported capture rejection",
				"captured callable/function-typed parameter global-storage escape",
				"unsupported function-value escape outside the fnptr ABI",
				"unsupported function-value call",
				"capturing closure raw-ptr escape",
				"captured closure explicit type-arg rejection",
				"function-typed explicit type-arg rejection",
				"generic closure capture and generic callback-closure capture rejection",
				"generic closure pointer/direct-call rejection",
				"imported mutable function-typed global boundary",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf(
						"full first-class callable feature missing %q boundary: %#v",
						want,
						feature,
					)
				}
			}
		}
		if feature.ID == "ui.metadata-v1" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"ui.metadata-v1 lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production UI metadata contract",
				"deterministic tetra.ui.v0.4.0 JSON",
				"browser-backed web command-dispatch runtime",
				"wasm32-web command dispatch",
				"post-v0.4 Web UI runtime smoke",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("ui.metadata-v1 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "ui.native-runtime" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf(
					"ui.native-runtime lifecycle = status %q since %q, want current since v0.4.0",
					feature.Status,
					feature.Since,
				)
			}
			for _, want := range []string{
				"production Linux-x64 native UI runtime path",
				"native runtime widget instances",
				"click/activate events",
				"state and widget updates",
				"tetra.ui.native-runtime.v1 smoke evidence",
				"metadata-only",
				"web-only",
				"native-shell sidecar-only",
				"macOS/Windows",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("ui.native-runtime feature missing %q boundary: %#v", want, feature)
				}
			}
		}
	}
	for _, status := range []string{"current", "planned", "post-v1"} {
		if !statusSeen[status] {
			t.Fatalf("features output missing %s status: %#v", status, report.Features)
		}
	}
	for id, wantStatus := range map[string]string{
		"cli.core":                                "current",
		"language.generics-mvp":                   "current",
		"language.protocol-conformance-mvp":       "current",
		"language.callable-mvp":                   "current",
		"targets.wasm-artifact-preflight":         "current",
		"stdlib.experimental-mirrors":             "current",
		"language.callable-level1":                "current",
		"language.enum-payload-match":             "current",
		"language.protocol-bound-generics-static": "current",
		"language.ownership-markers-mvp":          "current",
		"language.resource-lifetime-mvp":          "current",
		"actors.task-transfer-safety":             "current",
		"language.lifetime-ssa":                   "current",
		"language.callable-level2":                "current",
		"ui.metadata-v1":                          "current",
		"wasm.runtime-execution":                  "current",
		"eco.distributed-network":                 "post-v1",
		"actors.distributed-runtime":              "current",
		"ui.native-runtime":                       "current",
		"language.full-first-class-callables":     "current",
	} {
		if gotStatus := statusByID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
		}
	}
}

func TestFeaturesCommandTOON(t *testing.T) {
	var report struct {
		Schema   string `json:"schema"`
		Version  string `json:"version"`
		Features []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"features"`
	}
	raw := runCLITOONStdout(t, []string{"features", "--format=toon"}, 0, &report)
	if !strings.Contains(raw, "features[") || report.Schema != "tetra.features.v1" ||
		report.Version != compiler.Version() ||
		len(report.Features) == 0 {
		t.Fatalf("features TOON report incomplete: raw=%s report=%#v", raw, report)
	}
}

func TestFeaturesCommandRejectsUnsupportedFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"features", "--format=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("features exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormatsCommandListsOfficialT4Family(t *testing.T) {
	var report struct {
		Formats []struct {
			Name      string `json:"name"`
			Extension string `json:"extension,omitempty"`
			FileName  string `json:"file_name,omitempty"`
			Role      string `json:"role"`
			Primary   bool   `json:"primary,omitempty"`
			Legacy    bool   `json:"legacy,omitempty"`
		} `json:"formats"`
	}
	runCLIJSONStdout(t, []string{"formats", "--format=json"}, 0, &report)
	seen := map[string]bool{}
	for _, format := range report.Formats {
		if format.Extension != "" {
			seen[format.Extension] = true
		}
		if format.FileName != "" {
			seen[format.FileName] = true
		}
	}
	for _, want := range []string{
		".t4",
		".tetra",
		".tdx",
		".t4s",
		".t4i",
		".t4p",
		".t4r",
		".t4q",
		".tneed",
		"Tetra.lock",
	} {
		if !seen[want] {
			t.Fatalf("formats output missing %s: %#v", want, report.Formats)
		}
	}
	byExtension := map[string]struct {
		Name    string
		Role    string
		Primary bool
		Legacy  bool
	}{}
	for _, format := range report.Formats {
		if format.Extension != "" {
			byExtension[format.Extension] = struct {
				Name    string
				Role    string
				Primary bool
				Legacy  bool
			}{Name: format.Name, Role: format.Role, Primary: format.Primary, Legacy: format.Legacy}
		}
	}
	if got := byExtension[".t4"]; got.Role != "source" || !got.Primary || got.Legacy {
		t.Fatalf(".t4 format metadata = %#v", got)
	}
	if got := byExtension[".tetra"]; got.Role != "source" || got.Primary || !got.Legacy {
		t.Fatalf(".tetra format metadata = %#v", got)
	}
}

func TestFormatsCommandTOON(t *testing.T) {
	var report struct {
		Formats []struct {
			Name      string `json:"name"`
			Role      string `json:"role"`
			Extension string `json:"extension"`
			FileName  string `json:"file_name"`
		} `json:"formats"`
	}
	raw := runCLITOONStdout(t, []string{"formats", "--format=toon"}, 0, &report)
	if !strings.Contains(raw, "formats[") || len(report.Formats) == 0 {
		t.Fatalf("formats TOON report incomplete: raw=%s report=%#v", raw, report)
	}
}

// ---- new_app_test.go ----

func TestNewAppScaffoldCreatesRunnableT4Project(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "DemoApp")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "app", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"new app exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{"Capsule.t4", "src/main.t4", "tests/main_test.t4", "README.md"} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected scaffold file %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{
		`capsule DemoApp:`,
		`id "tetra://apps/demoapp"`,
		`entry "src/main.t4"`,
		`source "src"`,
		`source "tests"`,
		`target "` + mustHostTarget(t) + `"`,
		`permission "io"`,
	} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"scaffold check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"test", "--target", mustHostTarget(t), appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"scaffold test exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestNewAppLockOptionWritesTetraLock(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "LockedApp")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "app", "--lock", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"new app --lock exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(filepath.Join(appDir, "Tetra.lock"))
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://apps/lockedapp"`) {
		t.Fatalf("Tetra.lock missing scaffold capsule id:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Created app") ||
		!strings.Contains(stdout.String(), "Tetra.lock") {
		t.Fatalf("stdout = %q, want scaffold and lock messages", stdout.String())
	}
}

func TestNewAppRejectsExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "app", dir}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf(
			"new app exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectInfoCommandJSON(t *testing.T) {
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
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var report struct {
		Found       bool     `json:"found"`
		Root        string   `json:"root"`
		CapsulePath string   `json:"capsule_path"`
		EntryPath   string   `json:"entry_path"`
		SourceRoots []string `json:"source_roots"`
		Targets     []string `json:"targets"`
	}
	runCLIJSONStdout(t, []string{"project", "info", "--format=json", dir}, 0, &report)
	if !report.Found || filepath.Clean(report.Root) != filepath.Clean(dir) ||
		!strings.HasSuffix(filepath.ToSlash(report.CapsulePath), "Capsule.t4") ||
		!strings.HasSuffix(filepath.ToSlash(report.EntryPath), "src/main.t4") {
		t.Fatalf("project info report = %#v", report)
	}
	if strings.Join(report.SourceRoots, ",") != "src" ||
		strings.Join(report.Targets, ",") != "linux-x64" {
		t.Fatalf("project info roots/targets = %#v", report)
	}
}

// ---- new_surface_app_test.go ----

func TestNewSurfaceAppScaffoldCreatesRunnableBlockMorphProject(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "PaletteDesk")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"new", "surface-app", "--template", "command-palette", appDir},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"new surface-app exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{
		"Capsule.t4",
		"src/main.tetra",
		"surface-template.json",
		"README.md",
	} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected scaffold file %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{
		`capsule PaletteDesk:`,
		`id "tetra://surface-apps/palettedesk"`,
		`entry "src/main.tetra"`,
		`source "src"`,
		`target "` + mustHostTarget(t) + `"`,
		`target "wasm32-web"`,
	} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	sourceRaw, err := os.ReadFile(filepath.Join(appDir, "src", "main.tetra"))
	if err != nil {
		t.Fatal(err)
	}
	sourceText := string(sourceRaw)
	for _, want := range []string{
		"import lib.core.surface as surface",
		"import lib.core.block as block",
		"import lib.core.morph as morph",
		"morph.expand_",
	} {
		if !strings.Contains(sourceText, want) {
			t.Fatalf("src/main.tetra missing %q:\n%s", want, sourceText)
		}
	}
	assertSurfaceTemplateSourceHasNoForbiddenRuntime(t, sourceText)

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"surface scaffold check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	outPath := filepath.Join(dir, "palette-desk")
	code = runCLI(
		[]string{"build", "--target", mustHostTarget(t), "-o", outPath, appDir},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"surface scaffold build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"run", "--target", mustHostTarget(t), appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"surface scaffold run exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestNewSurfaceAppGeneratesAllP21TemplateKinds(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	for _, kind := range []string{
		"command-palette",
		"settings",
		"dashboard",
		"editor-shell",
		"studio-shell",
		"multi-window-notes",
		"web-canvas",
	} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			appDir := filepath.Join(dir, "Surface"+strings.ReplaceAll(kind, "-", ""))
			var stdout, stderr bytes.Buffer
			code := runCLI(
				[]string{"new", "surface-app", "--template", kind, appDir},
				&stdout,
				&stderr,
			)
			if code != 0 {
				t.Fatalf(
					"new surface-app %s exit code = %d, stdout=%q stderr=%q",
					kind,
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			metaRaw, err := os.ReadFile(filepath.Join(appDir, "surface-template.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(metaRaw), `"template": "`+kind+`"`) ||
				!strings.Contains(string(metaRaw), `"model": "surface-project-template-v1"`) {
				t.Fatalf(
					"surface-template.json missing template metadata for %s:\n%s",
					kind,
					string(metaRaw),
				)
			}
			sourceRaw, err := os.ReadFile(filepath.Join(appDir, "src", "main.tetra"))
			if err != nil {
				t.Fatal(err)
			}
			sourceText := string(sourceRaw)
			for _, want := range []string{
				"import lib.core.surface as surface",
				"import lib.core.block as block",
				"import lib.core.morph as morph",
			} {
				if !strings.Contains(sourceText, want) {
					t.Fatalf("%s source missing %q:\n%s", kind, want, sourceText)
				}
			}
			if (kind == "multi-window-notes" || kind == "studio-shell") &&
				!strings.Contains(sourceText, "import lib.core.surface_app_shell as shell") {
				t.Fatalf("%s source missing app shell import:\n%s", kind, sourceText)
			}
			if kind == "studio-shell" {
				for _, want := range []string{
					"morph.recipe_app_shell()",
					"morph.recipe_toolbar()",
					"morph.recipe_split_pane()",
					"morph.recipe_status_bar()",
				} {
					if !strings.Contains(sourceText, want) {
						t.Fatalf("studio-shell source missing %q:\n%s", want, sourceText)
					}
				}
			}
			if kind == "web-canvas" {
				capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(capsuleRaw), `target "wasm32-web"`) {
					t.Fatalf(
						"web-canvas capsule missing wasm32-web target:\n%s",
						string(capsuleRaw),
					)
				}
			}
			assertSurfaceTemplateSourceHasNoForbiddenRuntime(t, sourceText)
		})
	}
}

func TestNewSurfaceAppRejectsUnknownTemplateKind(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"new",
			"surface-app",
			"--template",
			"react-dashboard",
			filepath.Join(dir, "BadApp"),
		},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf(
			"new surface-app exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "unknown surface app template") {
		t.Fatalf("stderr = %q, want unknown template diagnostic", stderr.String())
	}
}

func assertSurfaceTemplateSourceHasNoForbiddenRuntime(t *testing.T, source string) {
	t.Helper()
	for _, forbidden := range []string{
		"React",
		"Electron",
		"Chromium",
		"DOM",
		"CSS",
		"JavaScript",
		"lib.core.widgets",
		"lib.core.component",
		"Button",
		"Card",
		"TextField",
		"TextBox",
		"platform widget",
		"native widget",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"generated Surface template source contains forbidden runtime/core-widget token %q:\n%s",
				forbidden,
				source,
			)
		}
	}
}

// ---- project_test.go ----

func TestProjectSyncWritesLockForProjectWithoutDependencies(t *testing.T) {
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
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"project sync exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	lockPath := filepath.Join(dir, "Tetra.lock")
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://demo"`) {
		t.Fatalf("Tetra.lock missing capsule id:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Project synced") {
		t.Fatalf("stdout = %q, want sync message", stdout.String())
	}
}

func TestProjectSyncCheckReportsMissingLockWithoutWriting(t *testing.T) {
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
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", "--check", dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf(
			"project sync --check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "would generate lock") ||
		!strings.Contains(combined, "Tetra.lock") {
		t.Fatalf(
			"sync --check output = stdout=%q stderr=%q, want missing lock dry-run",
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(filepath.Join(dir, "Tetra.lock")); err == nil {
		t.Fatalf("project sync --check unexpectedly wrote Tetra.lock")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat Tetra.lock: %v", err)
	}
}

func TestProjectSyncRejectsTargetAndAllTargetsTogether(t *testing.T) {
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
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "sync", "--target", "linux-x64", "--all-targets", dir},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf(
			"project sync exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "either --target or --all-targets") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectSyncGeneratesDependencyArtifactsAndLock(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", "--target", target, appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"project sync exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{
		"interfaces/math/core.t4i",
		"artifacts/math/core." + target + ".tobj",
		"seeds/app-deps.t4s",
		"Tetra.lock",
	} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected project sync generated %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatalf("read Capsule.t4: %v", err)
	}
	if !strings.Contains(string(capsuleRaw), "interface interfaces/math/core.t4i") ||
		!strings.Contains(
			string(capsuleRaw),
			"object "+target+" artifacts/math/core."+target+".tobj",
		) {
		t.Fatalf("Capsule.t4 missing generated artifact declarations:\n%s", string(capsuleRaw))
	}
	if !strings.Contains(stdout.String(), "Project synced") {
		t.Fatalf("stdout = %q, want sync message", stdout.String())
	}
}

func TestProjectSyncWritesLockForBuildOnlyTargetWithoutNativeArtifacts(t *testing.T) {
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, "wasm32-wasi")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"project sync exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(filepath.Join(appRoot, "Tetra.lock")); err != nil {
		t.Fatalf("expected Tetra.lock: %v", err)
	}
	if _, err := os.Stat(
		filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj")),
	); err == nil {
		t.Fatalf("project sync unexpectedly wrote wasm object artifact")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat wasm object artifact: %v", err)
	}
}

func TestProjectDepsAddPathDiscoversMetadataAndAppendsDeps(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "deps", "add", "--path", "../Math", appRoot},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"project deps add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsule := string(raw)
	if !strings.Contains(capsule, "deps:") ||
		!strings.Contains(capsule, "tetra://math 0.1.0 ../Math") {
		t.Fatalf("Capsule.t4 missing dependency:\n%s", capsule)
	}
	if !strings.Contains(stdout.String(), "Added dependency") ||
		!strings.Contains(stdout.String(), "run: tetra project sync") {
		t.Fatalf("stdout = %q, want add message and sync hint", stdout.String())
	}
}

func TestProjectDepsAddRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "deps", "add", "--path", "../Math", filepath.Join(dir, "App")},
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf(
			"project deps add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "duplicate dependency") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectDepsAddAllowsMetadataOverride(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"project",
			"deps",
			"add",
			"--path",
			"../Math",
			"--id",
			"tetra://math-alt",
			"--version",
			"0.2.0",
			appRoot,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"project deps add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "tetra://math-alt 0.2.0 ../Math") {
		t.Fatalf("Capsule.t4 missing overridden dependency:\n%s", string(raw))
	}
}

func TestProjectDepsListJSONReportsResolvedPath(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var report struct {
		Dependencies []struct {
			ID           string `json:"id"`
			Version      string `json:"version"`
			Path         string `json:"path"`
			ResolvedPath string `json:"resolved_path"`
			Status       string `json:"status"`
		} `json:"dependencies"`
	}
	runCLIJSONStdout(
		t,
		[]string{"project", "deps", "list", "--format=json", filepath.Join(dir, "App")},
		0,
		&report,
	)
	if len(report.Dependencies) != 1 {
		t.Fatalf("dependencies = %#v", report.Dependencies)
	}
	dep := report.Dependencies[0]
	if dep.ID != "tetra://math" || dep.Version != "0.1.0" || dep.Path != "../Math" ||
		dep.Status != "ok" ||
		!strings.HasSuffix(filepath.ToSlash(dep.ResolvedPath), "/Math") {
		t.Fatalf("dependency report = %#v", dep)
	}
}

func TestProjectDepsRemoveByID(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "deps", "remove", "--id", "tetra://math", appRoot},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"project deps remove exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "tetra://math") {
		t.Fatalf("dependency was not removed:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Removed dependency") ||
		!strings.Contains(stdout.String(), "run: tetra project sync") {
		t.Fatalf("stdout = %q, want remove message and sync hint", stdout.String())
	}
}

func TestProjectDepsRemoveRejectsAmbiguousID(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../MathV1
        tetra://math 0.2.0 ../MathV2
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "deps", "remove", "--id", "tetra://math", filepath.Join(dir, "App")},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf(
			"project deps remove exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "requires --version") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectDepsCheckPassesForValidDependency(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"project", "deps", "check", filepath.Join(dir, "App")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"project deps check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Dependencies OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestProjectDepsCheckFailsForMissingPathVersionMismatchAndCycle(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://missing 0.1.0 ../Missing
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

		var stdout, stderr bytes.Buffer
		code := runCLI(
			[]string{"project", "deps", "check", filepath.Join(dir, "App")},
			&stdout,
			&stderr,
		)
		if code != 1 {
			t.Fatalf(
				"project deps check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "tetra://missing") ||
			!strings.Contains(stderr.String(), "Missing") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("version mismatch", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.2.0"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

		var stdout, stderr bytes.Buffer
		code := runCLI(
			[]string{"project", "deps", "check", filepath.Join(dir, "App")},
			&stdout,
			&stderr,
		)
		if code != 1 {
			t.Fatalf(
				"project deps check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "version mismatch") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("cycle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    deps:
        tetra://app 0.1.0 ../App
`)

		var stdout, stderr bytes.Buffer
		code := runCLI(
			[]string{"project", "deps", "check", filepath.Join(dir, "App")},
			&stdout,
			&stderr,
		)
		if code != 1 {
			t.Fatalf(
				"project deps check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "capsule dependency cycle") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

// ---- ram_contract_cli_test.go ----

func TestBuildCommandRAMContractFlagsWriteReports(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 4
    xs[1] = 5
    return xs[0] + xs[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{
			"build",
			"--target",
			"linux-x64",
			"--emit-ram-contract-report",
			"-o",
			outPath,
			srcPath,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outPath + ".ram-contract.json"); err != nil {
		t.Fatalf("missing RAM contract report: %v", err)
	}
}

func TestBuildCommandFailIfHeapJSONDiagnostic(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "heap.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(5000)
    xs[0] = 7
    return xs[0]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(
		t,
		[]string{
			"build",
			"--diagnostics=json",
			"--target",
			"linux-x64",
			"--fail-if-heap",
			"-o",
			outPath,
			srcPath,
		},
		1,
	)
	if diag.Code != "TETRA4100" || !strings.Contains(diag.Message, "RAM_CONTRACT_HEAP") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

// ---- run_test.go ----

func TestRunCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", target}, 2)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" ||
		!strings.Contains(diag.Message, "cannot run target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestRunCommandTOONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLITOONDiagnostic(t, []string{"run", "--diagnostics=toon", "--target", target}, 2)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" ||
		!strings.Contains(diag.Message, "cannot run target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestRunCommandJSONDiagnosticsForWASMWebRuntimeUnsupported(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	restore := stubLookPath(func(name string) (string, error) {
		return "", exec.ErrNotFound
	})
	defer restore()

	diag := runCLIJSONDiagnostic(
		t,
		[]string{"run", "--diagnostics=json", "--target", "wasm32-web", srcPath},
		1,
	)
	for _, want := range []string{"cannot run target wasm32-web", "browser runner unavailable"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestRunCommandUsesBrowserRunnerForWASMWeb(t *testing.T) {
	requireLocalTCPBind(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	browser := filepath.Join(dir, "fake-chromium")
	if err := os.WriteFile(browser, []byte(`#!/bin/sh
printf '<html><body><pre id="result">exit:7</pre></body></html>\n'
`), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}
	restore := stubLookPath(func(name string) (string, error) {
		if name == "chromium" {
			return browser, nil
		}
		return "", exec.ErrNotFound
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "wasm32-web", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandJSONDiagnosticsForLinuxX32HostUnsupported(t *testing.T) {
	restore := stubLinuxX32HostSupport(false)
	defer restore()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"run", "--diagnostics=json", "--target", "x32", srcPath},
		2,
	)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s severity error",
			diag,
			compiler.DiagnosticCodeTargetRuntime,
		)
	}
	for _, want := range []string{
		"cannot run target linux-x32",
		expectedLinuxX32HostUnsupportedReason(t),
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestRunCommandUsesLinuxX32HostRunnerWhenProbePasses(t *testing.T) {
	restoreHost := stubLinuxX32HostSupport(true)
	defer restoreHost()
	restoreExec := stubNativeExec(func(path string, stdout io.Writer, stderr io.Writer) int {
		if err := requireX32ExecutableFile(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 42
	})
	defer restoreExec()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x32", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestExecWebProgramWithBrowserRunnerParsesBrowserExitResult(t *testing.T) {
	requireLocalTCPBind(t)

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte("\x00asm\x01\x00\x00\x00"), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "app.mjs"),
		[]byte("export async function runTetra() { return 7; }\n"),
		0o644,
	); err != nil {
		t.Fatalf("write loader: %v", err)
	}
	browser := filepath.Join(dir, "fake-chromium")
	if err := os.WriteFile(browser, []byte(`#!/bin/sh
printf '<html><body><pre id="result">exit:7</pre></body></html>\n'
`), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}

	exit, err := execWebProgramWithBrowserRunner(
		wasmPath,
		browser,
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("execWebProgramWithBrowserRunner: %v", err)
	}
	if exit != 7 {
		t.Fatalf("exit = %d, want 7", exit)
	}
}

func TestExecWebProgramWithNodeRunnerRunsSurfaceHostImports(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node unavailable: %v", err)
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("repo root unavailable: %v", err)
	}

	tmp := t.TempDir()
	wasmPath := filepath.Join(tmp, "surface-counter.wasm")
	if _, err := compiler.BuildFileWithStatsOpt(
		filepath.Join(repoRoot, "examples", "surface", "runtime", "surface_counter.tetra"),
		wasmPath,
		"wasm32-web",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build surface counter wasm32-web: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit, err := execWebProgramWithRunner(wasmPath, webRuntimeRunner{
		Name:   "node-web",
		Path:   node,
		Helper: filepath.Join(repoRoot, "scripts", "tools", "web_run_module.mjs"),
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("exec web program: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if exit != 1 {
		t.Fatalf("exit = %d, want Surface counter result 1", exit)
	}
}

func requireX32ExecutableFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 20 {
		return fmt.Errorf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x32 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x32 executable machine = %#x, want EM_X86_64", machine)
	}
	return nil
}

func requireLocalTCPBind(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP bind unavailable in this environment: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close local TCP probe: %v", err)
	}
}

func TestRunCommandPropagatesProgramExitCode(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86NoRuntimeExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86FunctionArgumentExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc " +
		"main() -> Int:\n    return add(40, 2)\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86GlobalExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "var answer: Int = 1\n\nfunc main() -> Int:\n    answer = 42\n    return answer\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86DirectCallbackExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("func add1(x: Int) -> Int:\n    return x + 1\n\nfunc apply(cb: " +
		"fn(Int) -> Int, x: Int) -> Int:\n    return cb(x)\n\nfunc " +
		"main() -> Int:\n    return apply(add1, 41)\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86MakeI32SliceExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, mem {\n  var xs: []i32 = " +
		"make_i32(3)\n  xs[0] = 10\n  xs[1] = 20\n  xs[2] = xs[0] + " +
		"xs[1]\n  return xs[2]\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 30 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86AllocBytesZeroExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, mem {\n  unsafe {\n    let _p: " +
		"ptr = core.alloc_bytes(0)\n    return 0\n  }\n  return 0\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawStoreLoadExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n  " +
		"  let mem: cap.mem = core.cap_mem()\n    let p: ptr = " +
		"core.alloc_bytes(4)\n    let _: Int = core.store_i32(p, 42, " +
		"mem)\n    return core.load_i32(p, mem)\n  return 0\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawPtrAddU8ExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n  " +
		"  let mem: cap.mem = core.cap_mem()\n    let p: ptr = " +
		"core.alloc_bytes(4)\n    let _: UInt8 = " +
		"core.store_u8(core.ptr_add(p, 1, mem), 7, mem)\n    return " +
		"core.load_u8(core.ptr_add(p, 1, mem), mem)\n  return 0\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawPtrAddUpperBoundExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n  " +
		"  let mem: cap.mem = core.cap_mem()\n    let p: ptr = " +
		"core.alloc_bytes(4)\n    let q: ptr = core.ptr_add(p, 4, mem)" +
		"\n    let _: UInt8 = core.store_u8(q, 7, mem)\n    return 0\n  " +
		"return 0\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86PrintStringStdout(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses io {\n  print(\"x86 says hi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "x86 says hi\n" {
		t.Fatalf("run stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86PrintSliceStdout(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, io, mem {\n  var xs: []u8 = " +
		"make_u8(2)\n  xs[0] = 65\n  xs[1] = 66\n  print(xs)\n  return " +
		"0\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "AB" {
		t.Fatalf("run stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = " +
		"0\n  island(64) as isl {\n    var xs: []u8 = " +
		"core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  " +
		"}\n  return out\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandDebugExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = " +
		"0\n  island(64) as isl {\n    var xs: []u8 = " +
		"core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  " +
		"}\n  return out\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", "--islands-debug", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandOverflowExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, islands, mem {\n  island(16) as " +
		"isl {\n    var xs: []u8 = core.island_make_u8(isl, 17)\n    " +
		"xs[0] = 1\n  }\n  return 0\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86MMIOExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := ("fun main(): i32 uses alloc, capability, io, mem, mmio {\n  " +
		"var out: i32 = 0\n  unsafe {\n    let io: cap.io = " +
		"core.cap_io()\n    let p: ptr = core.alloc_bytes(4)\n    let " +
		"_w: i32 = core.mmio_write_i32(p, 123, io)\n    out = " +
		"core.mmio_read_i32(p, io)\n  }\n  return out\n}\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 123 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RuntimeMatrixExitCodes(t *testing.T) {
	requireLinuxX86Execution(t)

	tests := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "recursion",
			src: ("func fact(n: Int) -> Int:\n    if n <= 1:\n        return 1\n  " +
				"  return n * fact(n - 1)\n\nfunc main() -> Int:\n    return " +
				"fact(5)\n"),
			want: 120,
		},
		{
			name: "while_loop",
			src: ("func main() -> Int:\n    var i: Int = 0\n    var acc: Int = " +
				"0\n    while i < 6:\n        acc = acc + i\n        i = i + 1\n " +
				"   return acc\n"),
			want: 15,
		},
		{
			name: "struct_fields",
			src: ("struct Pair:\n    left: Int\n    right: Int\n\nfunc main() -> " +
				"Int:\n    let p: Pair = Pair(left: 19, right: 23)\n    return " +
				"p.left + p.right\n"),
			want: 42,
		},
		{
			name: "enum_payload_match",
			src: ("enum Msg:\n    case left(Int)\n    case right(Int)\n\nfunc " +
				"choose(flag: Int) -> Msg:\n    if flag:\n        return " +
				"Msg.left(40)\n    return Msg.right(2)\n\nfunc main() -> Int:\n  " +
				"  let msg: Msg = choose(0)\n    match msg:\n    case " +
				"Msg.left(value):\n        return value\n    case " +
				"Msg.right(value):\n        return value + 40\n"),
			want: 42,
		},
		{
			name: "u16_slice",
			src: ("fun main(): i32 uses alloc, mem {\n  var xs: []u16 = " +
				"make_u16(2)\n  xs[0] = 40\n  xs[1] = 2\n  return xs[0] + xs[1]" +
				"\n}\n"),
			want: 42,
		},
		{
			name: "bool_slice",
			src: ("func main() -> Int\nuses alloc, mem:\n    var flags: []bool = " +
				"make_bool(2)\n    flags[0] = true\n    flags[1] = false\n    " +
				"if flags[0]:\n        return 42\n    return 1\n"),
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "main.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
			if code != tt.want {
				t.Fatalf(
					"run exit code = %d, want %d, stdout=%q stderr=%q",
					code,
					tt.want,
					stdout.String(),
					stderr.String(),
				)
			}
			if stderr.Len() != 0 {
				t.Fatalf("run stderr = %q", stderr.String())
			}
		})
	}
}

func TestRunCommandWithoutOutputDoesNotLeaveDefaultBinary(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	tgt, err := ctarget.Parse(mustHostTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(dir, defaultOutput(tgt, "exe"))
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("run without -o should not leave %s, stat err=%v", defaultPath, err)
	}
}

func requireLinuxX86Execution(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "386") {
		t.Skipf(
			"linux-x86 execution requires a Linux i386-compatible host, got %s/%s",
			runtime.GOOS,
			runtime.GOARCH,
		)
	}
	if !canRunLinuxX86OnHost() {
		t.Skipf("Linux kernel cannot execute generated i386 ELF on this host")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "probe.tetra")
	outPath := filepath.Join(dir, "probe")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "x86"); err != nil {
		t.Fatalf("build linux-x86 execution probe: %v", err)
	}
	var stderr bytes.Buffer
	code := execProgram(outPath, &bytes.Buffer{}, &stderr)
	if code == 7 {
		return
	}
	if code == -1 && strings.TrimSpace(stderr.String()) == "" {
		t.Skipf("Linux kernel rejected generated i386 ELF with signal-like exit %d", code)
	}
	if strings.Contains(stderr.String(), "exec format error") ||
		strings.Contains(stderr.String(), "no such file or directory") {
		t.Skipf(
			"Linux kernel cannot execute generated i386 ELF on this host: exit=%d stderr=%q",
			code,
			stderr.String(),
		)
	}
	t.Fatalf("linux-x86 execution probe exit=%d stderr=%q", code, stderr.String())
}

// ---- smoke_test.go ----

func TestSmokeCommandWritesReport(t *testing.T) {
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	report := filepath.Join(t.TempDir(), "smoke.json")
	var stdout bytes.Buffer
	code := runCLI(
		[]string{"smoke", "--target", target, "--run=false", "--report", report},
		&stdout,
		&bytes.Buffer{},
	)
	if code != 0 {
		t.Fatalf("smoke exit code = %d, stdout=%q", code, stdout.String())
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), `"cases"`) ||
		!strings.Contains(string(raw), `"islands_hello"`) {
		t.Fatalf("unexpected smoke report: %s", string(raw))
	}
	var smokeReport struct {
		Target  string `json:"target"`
		Version string `json:"version"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Cases   []struct {
			Name string `json:"name"`
			Pass bool   `json:"pass"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &smokeReport); err != nil {
		t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
	}
	if smokeReport.Target != target || smokeReport.Version != compiler.Version() ||
		len(smokeReport.Cases) == 0 {
		t.Fatalf("smoke report shape = %#v", smokeReport)
	}
	if smokeReport.Total != len(smokeReport.Cases) ||
		smokeReport.Passed != len(smokeReport.Cases) ||
		smokeReport.Failed != 0 {
		t.Fatalf("smoke report counts = %#v", smokeReport)
	}
}

func TestSmokeCommandWritesTOONReportMirror(t *testing.T) {
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	report := filepath.Join(t.TempDir(), "smoke.json")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"smoke",
			"--target",
			target,
			"--run=false",
			"--report",
			report,
			"--report-format=both",
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"smoke exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	toonPath := strings.TrimSuffix(report, ".json") + ".toon"
	raw, err := os.ReadFile(toonPath)
	if err != nil {
		t.Fatalf("read TOON report: %v", err)
	}
	jsonRaw, err := toon.ConvertTOONToJSON(raw, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("decode TOON report: %v\n%s", err, raw)
	}
	var smokeReport struct {
		Target string `json:"target"`
		Total  int    `json:"total"`
		Cases  []struct {
			Name string `json:"name"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(jsonRaw, &smokeReport); err != nil {
		t.Fatalf("decode smoke report JSON: %v\n%s", err, jsonRaw)
	}
	if smokeReport.Target != target || smokeReport.Total != len(smokeReport.Cases) ||
		len(smokeReport.Cases) == 0 {
		t.Fatalf("smoke TOON report shape = %#v", smokeReport)
	}
}

func TestSmokeCommandBuildOnlyNativeTargetsMarkUnsupportedFilesystem(t *testing.T) {
	for _, target := range []string{"macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			reportPath := filepath.Join(t.TempDir(), target+"-smoke.json")
			var stdout, stderr bytes.Buffer
			code := runCLI(
				[]string{"smoke", "--target", target, "--run=false", "--report", reportPath},
				&stdout,
				&stderr,
			)
			if code != 0 {
				t.Fatalf(
					"smoke %s exit code = %d, stdout=%q stderr=%q",
					target,
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			raw, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("read smoke report: %v", err)
			}
			var report struct {
				Target string `json:"target"`
				Total  int    `json:"total"`
				Passed int    `json:"passed"`
				Failed int    `json:"failed"`
				Cases  []struct {
					Name               string `json:"name"`
					Unsupported        bool   `json:"unsupported"`
					ExpectedDiagnostic string `json:"expected_diagnostic"`
					Diagnostic         string `json:"diagnostic"`
					OutPath            string `json:"out_path"`
					Ran                bool   `json:"ran"`
					Pass               bool   `json:"pass"`
					Error              string `json:"error"`
				} `json:"cases"`
			}
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatalf("decode smoke report: %v\n%s", err, raw)
			}
			if report.Target != target || report.Total == 0 || report.Passed != report.Total ||
				report.Failed != 0 {
				t.Fatalf("unexpected smoke report counts for %s: %#v", target, report)
			}
			found := false
			for _, c := range report.Cases {
				if c.Name != "core_filesystem_smoke" {
					continue
				}
				found = true
				want := "filesystem runtime not supported on " + target
				if !c.Unsupported || c.ExpectedDiagnostic != want ||
					!strings.Contains(c.Diagnostic, want) ||
					c.OutPath != "" ||
					c.Ran ||
					!c.Pass ||
					c.Error != "" {
					t.Fatalf("unexpected filesystem smoke case for %s: %#v", target, c)
				}
			}
			if !found {
				t.Fatalf("smoke report missing core_filesystem_smoke for %s", target)
			}
		})
	}
}

func TestSmokeCommandListsCasesAsJSON(t *testing.T) {
	var report struct {
		Target       string `json:"target"`
		BuildOnly    bool   `json:"build_only"`
		RunSupported bool   `json:"run_supported"`
		Total        int    `json:"total"`
		IslandsDebug bool   `json:"islands_debug"`
		Cases        []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			TargetGroup  string `json:"target_group"`
			ExpectedExit int    `json:"expected_exit"`
			DebugOnly    bool   `json:"debug_only"`
		} `json:"cases"`
		ExcludedExamples []struct {
			SrcPath string `json:"src_path"`
			Reason  string `json:"reason"`
		} `json:"excluded_examples"`
	}
	runCLIJSONStdout(t, []string{"smoke", "--list", "--format=json"}, 0, &report)
	if report.Target == "" {
		t.Fatalf("smoke list missing target: %#v", report)
	}
	if report.BuildOnly {
		t.Fatalf("default smoke list unexpectedly marked build-only: %#v", report)
	}
	if report.Total != len(report.Cases) || report.Total < 39 {
		t.Fatalf("smoke list counts = total:%d len:%d", report.Total, len(report.Cases))
	}
	var sawFlowHello bool
	var sawUINative bool
	var sawComplexControl bool
	var sawHelloT4Exclusion bool
	requiredCoreStdlib := map[string]string{
		"core_async_smoke":         "examples/async/core_async_smoke.tetra",
		"core_capability_smoke":    "examples/core/memory/core_capability_smoke.tetra",
		"core_collections_smoke":   "examples/core/data/core_collections_smoke.tetra",
		"core_component_smoke":     "examples/core/surface/core_component_smoke.tetra",
		"core_crypto_smoke":        "examples/core/memory/core_crypto_smoke.tetra",
		"core_filesystem_smoke":    "examples/core/platform/core_filesystem_smoke.tetra",
		"core_io_smoke":            "examples/core/platform/core_io_smoke.tetra",
		"core_math_smoke":          "examples/core/data/core_math_smoke.tetra",
		"core_memory_smoke":        "examples/core/memory/core_memory_smoke.tetra",
		"core_networking_smoke":    "examples/core/platform/core_networking_smoke.tetra",
		"core_serialization_smoke": "examples/core/data/core_serialization_smoke.tetra",
		"core_slices_smoke":        "examples/core/data/core_slices_smoke.tetra",
		"core_strings_smoke":       "examples/core/data/core_strings_smoke.tetra",
		"core_sync_smoke":          "examples/core/runtime/core_sync_smoke.tetra",
		"core_testing_smoke":       "examples/core/runtime/core_testing_smoke.tetra",
		"core_time_smoke":          "examples/core/platform/core_time_smoke.tetra",
	}
	requiredSurfaceMigrations := map[string]struct {
		src          string
		expectedExit int
	}{
		"surface_migration_ui_web_smoke": {
			src:          "examples/surface/migration/surface_migration_ui_web_smoke.tetra",
			expectedExit: 2,
		},
		"surface_migration_ui_native_shell_smoke": {
			src:          "examples/surface/migration/surface_migration_ui_native_shell_smoke.tetra",
			expectedExit: 11,
		},
		"surface_migration_dogfood_web_ui": {
			src:          "examples/surface/migration/surface_migration_dogfood_web_ui.tetra",
			expectedExit: 3,
		},
		"surface_migration_tetra_control_center": {
			src:          "examples/surface/migration/surface_migration_tetra_control_center.tetra",
			expectedExit: 5,
		},
	}
	for _, c := range report.Cases {
		if c.Name == "flow_hello" && c.SrcPath == "examples/flow/flow_hello.tetra" &&
			c.TargetGroup == "native" &&
			c.ExpectedExit == 0 {
			sawFlowHello = true
		}
		if c.Name == "ui_native_shell_smoke" &&
			c.SrcPath == "examples/ui/ui_native_shell_smoke.tetra" &&
			c.TargetGroup == "native" &&
			c.ExpectedExit == 0 {
			sawUINative = true
		}
		if c.Name == "complex_control_flow_smoke" &&
			c.SrcPath == "examples/smoke/control/complex_control_flow_smoke.tetra" &&
			c.TargetGroup == "native" &&
			c.ExpectedExit == 42 {
			sawComplexControl = true
		}
		if wantSrc, ok := requiredCoreStdlib[c.Name]; ok && c.SrcPath == wantSrc &&
			c.TargetGroup == "native" &&
			c.ExpectedExit == 42 {
			delete(requiredCoreStdlib, c.Name)
		}
		if want, ok := requiredSurfaceMigrations[c.Name]; ok && c.SrcPath == want.src &&
			c.TargetGroup == "native" &&
			c.ExpectedExit == want.expectedExit {
			delete(requiredSurfaceMigrations, c.Name)
		}
	}
	if !sawFlowHello {
		t.Fatalf("smoke list missing flow_hello: %#v", report.Cases)
	}
	if !sawUINative {
		t.Fatalf("smoke list missing ui_native_shell_smoke: %#v", report.Cases)
	}
	if !sawComplexControl {
		t.Fatalf("smoke list missing complex_control_flow_smoke: %#v", report.Cases)
	}
	if len(requiredCoreStdlib) != 0 {
		t.Fatalf("smoke list missing core stdlib cases: %#v", requiredCoreStdlib)
	}
	if len(requiredSurfaceMigrations) != 0 {
		t.Fatalf("smoke list missing Surface migration cases: %#v", requiredSurfaceMigrations)
	}
	for _, exclusion := range report.ExcludedExamples {
		if exclusion.SrcPath == "examples/projects/hello_t4/src/main.t4" &&
			strings.Contains(exclusion.Reason, report.Target) {
			sawHelloT4Exclusion = true
		}
	}
	if !sawHelloT4Exclusion {
		t.Fatalf(
			"smoke list missing T4 example exclusion for hello_t4: %#v",
			report.ExcludedExamples,
		)
	}
}

func TestSmokeCommandListsCasesAsTOON(t *testing.T) {
	var report smokeListReport
	raw := runCLITOONStdout(t, []string{"smoke", "--list", "--format=toon"}, 0, &report)
	if report.Target == "" || report.Total != len(report.Cases) || report.Total < 39 {
		t.Fatalf("smoke TOON list shape = %#v\n%s", report, raw)
	}
	if !strings.Contains(raw, "cases[") {
		t.Fatalf("smoke TOON list should use structured cases output:\n%s", raw)
	}
}

func TestSmokeCommandListsNativeSurfaceCounter(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(
		t,
		[]string{"smoke", "--list", "--target", "linux-x64", "--format=json"},
		0,
		&report,
	)
	if report.Target != "linux-x64" || report.BuildOnly {
		t.Fatalf("native smoke list metadata = %#v", report)
	}
	for _, c := range report.Cases {
		if c.Name != "surface_counter" {
			continue
		}
		if c.SrcPath != "examples/surface/runtime/surface_counter.tetra" ||
			c.TargetGroup != "native" ||
			c.ExpectedExit != 1 ||
			c.Unsupported ||
			c.ExpectedDiagnostic != "" ||
			c.DebugOnly {
			t.Fatalf("surface_counter smoke list case = %#v", c)
		}
		return
	}
	t.Fatalf("native smoke list missing surface_counter: %#v", report.Cases)
}

func TestSmokeCommandListsNativeSurfaceTextInput(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(
		t,
		[]string{"smoke", "--list", "--target", "linux-x64", "--format=json"},
		0,
		&report,
	)
	if report.Target != "linux-x64" || report.BuildOnly {
		t.Fatalf("native smoke list metadata = %#v", report)
	}
	for _, c := range report.Cases {
		if c.Name != "surface_text_input" {
			continue
		}
		if c.SrcPath != "examples/surface/runtime/surface_text_input.tetra" ||
			c.TargetGroup != "native" ||
			c.ExpectedExit != 42 ||
			c.Unsupported ||
			c.ExpectedDiagnostic != "" ||
			c.DebugOnly {
			t.Fatalf("surface_text_input smoke list case = %#v", c)
		}
		return
	}
	t.Fatalf("native smoke list missing surface_text_input: %#v", report.Cases)
}

func TestSmokeCommandKeepsInvalidDoubleFreeOutOfDebugList(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(t, []string{"smoke", "--list", "--format=json", "--islands-debug"}, 0, &report)
	if !report.IslandsDebug {
		t.Fatalf("islands_debug = false")
	}
	for _, c := range report.Cases {
		if c.Name == "islands_double_free" {
			t.Fatalf("debug smoke list includes semantic-negative islands_double_free: %#v", c)
		}
	}
}

func TestSmokeCommandDefinesIslandsDebugScopeRows(t *testing.T) {
	rows := islandsDebugScopeRows()
	required := map[string]string{
		"overflow_trap":  "live_trap",
		"double_free":    "static_only_nonclaim",
		"use_after_free": "static_only_nonclaim",
		"stale_epoch":    "static_only_nonclaim",
		"wrong_island":   "static_only_nonclaim",
	}
	for _, row := range rows {
		wantStatus, ok := required[row.Name]
		if !ok {
			t.Fatalf("unexpected islands debug scope row: %#v", row)
		}
		if row.Status != wantStatus || row.Evidence == "" || row.Reason == "" {
			t.Fatalf(
				"islands debug scope row %s = %#v, want status %s with evidence/reason",
				row.Name,
				row,
				wantStatus,
			)
		}
		if row.Status == "static_only_nonclaim" && !strings.Contains(row.Reason, "no live") {
			t.Fatalf(
				"static-only scope row %s reason missing no-live nonclaim: %q",
				row.Name,
				row.Reason,
			)
		}
		delete(required, row.Name)
	}
	if len(required) != 0 {
		t.Fatalf("missing islands debug scope rows: %#v", required)
	}
}

func TestSmokeCommandListsWASMRuntimeTargets(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var report struct {
			Target       string `json:"target"`
			BuildOnly    bool   `json:"build_only"`
			RunSupported bool   `json:"run_supported"`
			Cases        []struct {
				Name               string `json:"name"`
				SrcPath            string `json:"src_path"`
				TargetGroup        string `json:"target_group"`
				ExpectedExit       int    `json:"expected_exit"`
				Unsupported        bool   `json:"unsupported"`
				ExpectedDiagnostic string `json:"expected_diagnostic"`
			} `json:"cases"`
		}
		runCLIJSONStdout(
			t,
			[]string{"smoke", "--list", "--target", target, "--format=json"},
			0,
			&report,
		)
		if report.Target != target || report.BuildOnly {
			t.Fatalf("wasm smoke list metadata = %#v", report)
		}
		required := map[string]string{
			"ui_web_smoke":       "examples/ui/ui_web_smoke.tetra",
			"core_slices_smoke":  "examples/core/data/core_slices_smoke.tetra",
			"wasm_globals_smoke": "examples/wasm/wasm_globals_smoke.tetra",
		}
		if target == "wasm32-wasi" {
			required["wasm_multi_return_2_smoke"] = "examples/wasm/wasm_multi_return_2_smoke.tetra"
			required["wasm_multi_return_3_smoke"] = "examples/wasm/wasm_multi_return_3_smoke.tetra"
			required["wasm_multi_return_4_smoke"] = "examples/wasm/wasm_multi_return_4_smoke.tetra"
		} else {
			required["surface_counter"] = "examples/surface/runtime/surface_counter.tetra"
			required["surface_text_input"] = "examples/surface/runtime/surface_text_input.tetra"
		}
		unsupported := map[string]string{
			"time_sleep_smoke": "runtime not supported on wasm32",
			"task_smoke":       "runtime not supported on wasm32",
			"actors_pingpong":  "runtime not supported on wasm32",
		}
		for _, c := range report.Cases {
			if wantSrc, ok := required[c.Name]; ok && c.SrcPath == wantSrc &&
				c.TargetGroup == "wasm" &&
				!c.Unsupported &&
				c.ExpectedDiagnostic == "" {
				delete(required, c.Name)
			}
			if wantDiagnostic, ok := unsupported[c.Name]; ok {
				if !c.Unsupported || !strings.Contains(c.ExpectedDiagnostic, wantDiagnostic) {
					t.Fatalf(
						"unsupported wasm case %s = %#v, want diagnostic containing %q",
						c.Name,
						c,
						wantDiagnostic,
					)
				}
				delete(unsupported, c.Name)
			}
		}
		if len(required) != 0 || len(unsupported) != 0 {
			t.Fatalf(
				"wasm smoke list missing required=%#v unsupported=%#v in %#v",
				required,
				unsupported,
				report.Cases,
			)
		}
	}
}

func TestSmokeCommandBuildsWASMTargetWithoutRun(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		reportPath := filepath.Join(t.TempDir(), target+"-smoke.json")
		code := runCLI(
			[]string{"smoke", "--target", target, "--run=false", "--report", reportPath},
			&stdout,
			&bytes.Buffer{},
		)
		if code != 0 {
			t.Fatalf("smoke exit code = %d, stdout=%q", code, stdout.String())
		}
		var report smokeReport
		raw, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read smoke report: %v", err)
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
		}
		if report.Target != target || report.Total == 0 {
			t.Fatalf("wasm smoke report = %#v", report)
		}
		if report.Failed != 0 || report.Passed != report.Total {
			t.Fatalf("wasm smoke counts = %#v", report)
		}
		for _, c := range report.Cases {
			if c.Unsupported {
				if c.OutPath != "" || c.ExpectedDiagnostic == "" || c.Diagnostic == "" || !c.Pass {
					t.Fatalf("unexpected unsupported wasm smoke case %s: %#v", c.Name, c)
				}
				continue
			}
			if !strings.HasSuffix(c.OutPath, ".wasm") {
				t.Fatalf("expected wasm output path, case=%#v", c)
			}
			if c.Error != "" {
				t.Fatalf("unexpected wasm smoke error for %s: %s", c.Name, c.Error)
			}
		}
	}
}

func TestSmokeCommandWASMReportUsesDurableArtifacts(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		reportDir := t.TempDir()
		reportPath := filepath.Join(reportDir, target+"-smoke.json")
		code := runCLI(
			[]string{"smoke", "--target", target, "--run=false", "--report", reportPath},
			&stdout,
			&bytes.Buffer{},
		)
		if code != 0 {
			t.Fatalf("smoke %s exit code = %d, stdout=%q", target, code, stdout.String())
		}
		var report smokeReport
		raw, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read smoke report: %v", err)
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
		}
		if report.Total == 0 || len(report.Cases) != report.Total {
			t.Fatalf("unexpected smoke report shape: %#v", report)
		}
		for _, c := range report.Cases {
			if c.Unsupported {
				if c.OutPath != "" || c.ExpectedDiagnostic == "" || c.Diagnostic == "" || !c.Pass {
					t.Fatalf("%s unsupported case metadata = %#v", c.Name, c)
				}
				continue
			}
			if !strings.HasPrefix(c.OutPath, reportDir+string(os.PathSeparator)) {
				t.Fatalf("%s out_path is not under report dir: %s", c.Name, c.OutPath)
			}
			if _, err := os.Stat(c.OutPath); err != nil {
				t.Fatalf(
					"%s out_path is not durable after smoke command: %s: %v",
					c.Name,
					c.OutPath,
					err,
				)
			}
		}
	}
}

func TestSmokeCommandRunsWASIWithNodeFallbackRunner(t *testing.T) {
	tmpDir := t.TempDir()
	nodeLog := filepath.Join(tmpDir, "node.log")
	fakeNode := filepath.Join(tmpDir, "node")
	if err := os.WriteFile(fakeNode, []byte(`#!/bin/sh
printf '%s\n' "$@" >> "$TETRA_FAKE_NODE_LOG"
case "$*" in
  *core_slices_smoke.wasm*) exit 0 ;;
  *) exit 0 ;;
esac
`), 0o755); err != nil {
		t.Fatalf("write fake node runner: %v", err)
	}
	t.Setenv("PATH", tmpDir)
	t.Setenv("TETRA_FAKE_NODE_LOG", nodeLog)

	var stdout, stderr bytes.Buffer
	reportPath := filepath.Join(tmpDir, "wasi-smoke.json")
	code := runCLI(
		[]string{"smoke", "--target", "wasm32-wasi", "--run=true", "--report", reportPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"smoke exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read smoke report: %v", err)
	}
	var report struct {
		Target string `json:"target"`
		Runner string `json:"runner"`
		Total  int    `json:"total"`
		Passed int    `json:"passed"`
		Failed int    `json:"failed"`
		Cases  []struct {
			Name         string `json:"name"`
			Unsupported  bool   `json:"unsupported"`
			Ran          bool   `json:"ran"`
			ActualExit   *int   `json:"actual_exit"`
			ExpectedExit int    `json:"expected_exit"`
			Diagnostic   string `json:"diagnostic"`
			Pass         bool   `json:"pass"`
			Error        string `json:"error"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
	}
	if report.Target != "wasm32-wasi" || report.Runner != "node-wasi" {
		t.Fatalf("unexpected WASI runner report metadata: %#v", report)
	}
	if report.Total == 0 || report.Passed != report.Total || report.Failed != 0 ||
		len(report.Cases) != report.Total {
		t.Fatalf("unexpected WASI runner report counts: %#v", report)
	}
	for _, c := range report.Cases {
		if c.Unsupported {
			if c.Ran || c.ActualExit != nil || c.Diagnostic == "" || !c.Pass || c.Error != "" {
				t.Fatalf("unexpected unsupported WASI runtime case report for %s: %#v", c.Name, c)
			}
			continue
		}
		if !c.Ran || c.ActualExit == nil || *c.ActualExit != c.ExpectedExit || !c.Pass ||
			c.Error != "" {
			t.Fatalf("unexpected WASI runtime case report for %s: %#v", c.Name, c)
		}
	}
	logRaw, err := os.ReadFile(nodeLog)
	if err != nil {
		t.Fatalf("read fake node log: %v", err)
	}
	if !strings.Contains(string(logRaw), "scripts/tools/wasi_run_module.mjs") {
		t.Fatalf("fake node runner was not invoked through WASI helper, log=%q", string(logRaw))
	}
}

func TestSmokeCommandListWASIRunSupportedTracksRunnerAvailability(t *testing.T) {
	t.Run("runner available", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			if name == "wasmtime" {
				return "/usr/bin/wasmtime", nil
			}
			return "", exec.ErrNotFound
		})
		defer restore()

		var report smokeListReport
		runCLIJSONStdout(
			t,
			[]string{"smoke", "--list", "--target", "wasm32-wasi", "--format=json"},
			0,
			&report,
		)
		if report.BuildOnly || !report.RunSupported {
			t.Fatalf("wasm32-wasi smoke list metadata with runner = %#v", report)
		}
	})

	t.Run("runner missing", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			return "", exec.ErrNotFound
		})
		defer restore()

		var report smokeListReport
		runCLIJSONStdout(
			t,
			[]string{"smoke", "--list", "--target", "wasm32-wasi", "--format=json"},
			0,
			&report,
		)
		if report.BuildOnly || report.RunSupported {
			t.Fatalf("wasm32-wasi smoke list metadata without runner = %#v", report)
		}
	})
}

func TestSmokeCommandWASMTargetGroupsIncludeDogfoodWebUI(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(
		t,
		[]string{"smoke", "--list", "--target", "wasm32-web", "--format=json"},
		0,
		&report,
	)
	required := map[string]string{
		"ui_web_smoke":       "examples/ui/ui_web_smoke.tetra",
		"surface_counter":    "examples/surface/runtime/surface_counter.tetra",
		"surface_text_input": "examples/surface/runtime/surface_text_input.tetra",
		"dogfood_web_ui":     "examples/projects/dogfood_web_ui/src/main.tetra",
	}
	for _, c := range report.Cases {
		if wantPath, ok := required[c.Name]; ok {
			if c.SrcPath != wantPath || c.TargetGroup != "wasm" {
				t.Fatalf("case %s = %#v, want src %s in wasm group", c.Name, c, wantPath)
			}
			delete(required, c.Name)
		}
	}
	if len(required) != 0 {
		t.Fatalf("wasm smoke list missing required cases: %#v", required)
	}
}

func TestSmokeCommandRejectsFormatWithoutList(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"smoke", "--format=json"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("smoke exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--format is only supported with --list") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// ---- surface_dev_test.go ----

func TestSurfaceDevCommandWritesFastRebuildReport(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev fast rebuild cache evidence is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry, tokens, recipes := writeSurfaceDevFixture(t, dir)
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")
	morphRenderedBeautyReportPath := filepath.Join(dir, "surface-morph-rendered-beauty.json")
	writeSurfaceDevMorphRenderedBeautyReport(t, morphRenderedBeautyReportPath, entry)
	outDir := filepath.Join(dir, "dist")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--out-dir", outDir,
		"--report", reportPath,
		"--morph-rendered-beauty-report", morphRenderedBeautyReportPath,
		"--change-file", "token:" + tokens,
		"--change-file", "recipe:" + recipes,
		"--change-file", "source:" + entry,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"surface dev exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Schema                 string `json:"schema"`
		Model                  string `json:"model"`
		ReleaseScope           string `json:"release_scope"`
		Command                string `json:"command"`
		Mode                   string `json:"mode"`
		ReloadSemantics        string `json:"reload_semantics"`
		ProcessRestartRequired bool   `json:"process_restart_required"`
		HotReloadClaim         bool   `json:"hot_reload_claim"`
		Pass                   bool   `json:"pass"`
		Steps                  []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			ChangedPath     string   `json:"changed_path"`
			DurationMS      int64    `json:"duration_ms"`
			CompiledModules []string `json:"compiled_modules"`
			CacheHits       []string `json:"cache_hits"`
			Pass            bool     `json:"pass"`
		} `json:"steps"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
		MorphToPixels struct {
			ChainID                 string `json:"chain_id"`
			ReportPath              string `json:"report_path"`
			Source                  string `json:"source"`
			TokenCount              int    `json:"token_count"`
			RecipeCount             int    `json:"recipe_count"`
			RecipeExpansionCount    int    `json:"recipe_expansion_count"`
			BlockSceneHash          string `json:"block_scene_hash"`
			RenderCommandStreamHash string `json:"render_command_stream_hash"`
			RenderCommandCount      int    `json:"render_command_count"`
			FrameArtifact           string `json:"frame_artifact"`
			GoldenArtifact          string `json:"golden_artifact"`
			DiffPixels              int    `json:"diff_pixels"`
			Pass                    bool   `json:"pass"`
		} `json:"morph_to_pixels"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Schema != "tetra.surface.dev-workflow.v1" ||
		report.Model != "surface-dev-workflow-v1" ||
		report.ReleaseScope != "surface-v1-linux-web" ||
		report.Command != "tetra surface dev" ||
		report.Mode != "fast-rebuild" ||
		report.ReloadSemantics != "fast-rebuild" ||
		!report.ProcessRestartRequired ||
		report.HotReloadClaim ||
		!report.Pass {
		t.Fatalf("unexpected report header = %#v", report)
	}
	steps := map[string]struct {
		compiled int
		cache    int
		pass     bool
	}{}
	for _, step := range report.Steps {
		if step.DurationMS < 0 || !step.Pass {
			t.Fatalf("bad step = %#v", step)
		}
		steps[step.Kind] = struct {
			compiled int
			cache    int
			pass     bool
		}{compiled: len(step.CompiledModules), cache: len(step.CacheHits), pass: step.Pass}
	}
	for _, want := range []string{
		"initial",
		"warm-cache",
		"token-change",
		"recipe-change",
		"source-change",
	} {
		if !steps[want].pass {
			t.Fatalf("missing or failed rebuild step %q in %#v", want, report.Steps)
		}
	}
	if steps["warm-cache"].compiled != 0 || steps["warm-cache"].cache == 0 {
		t.Fatalf(
			"warm-cache step = %#v, want zero compiled modules and cache hits",
			steps["warm-cache"],
		)
	}
	for _, want := range []string{"token-change", "recipe-change", "source-change"} {
		if steps[want].compiled == 0 {
			t.Fatalf("%s step = %#v, want changed module compilation", want, steps[want])
		}
	}
	diagnosticKinds := map[string]bool{}
	for _, diag := range report.SourceDiagnostics {
		if diag.Path == "" || diag.Line <= 0 || diag.Column <= 0 || diag.Severity == "" ||
			!diag.Pass {
			t.Fatalf("bad source diagnostic = %#v", diag)
		}
		diagnosticKinds[diag.Kind] = true
	}
	for _, want := range []string{"token", "recipe", "source"} {
		if !diagnosticKinds[want] {
			t.Fatalf("missing %s source diagnostic in %#v", want, report.SourceDiagnostics)
		}
	}
	if !report.MorphToPixels.Pass ||
		report.MorphToPixels.ChainID == "" ||
		filepath.Clean(report.MorphToPixels.Source) != filepath.Clean(entry) ||
		report.MorphToPixels.TokenCount == 0 ||
		report.MorphToPixels.RecipeCount == 0 ||
		report.MorphToPixels.RecipeExpansionCount < report.MorphToPixels.RecipeCount ||
		report.MorphToPixels.RenderCommandCount == 0 ||
		report.MorphToPixels.BlockSceneHash == "" ||
		report.MorphToPixels.RenderCommandStreamHash == "" ||
		report.MorphToPixels.FrameArtifact == "" ||
		report.MorphToPixels.GoldenArtifact == "" {
		t.Fatalf(
			"morph_to_pixels = %#v, want source-linked Morph-to-pixels chain",
			report.MorphToPixels,
		)
	}
}

func TestSurfaceDevCommandJSONDiagnosticIncludesSurfacePath(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev diagnostic smoke is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry := filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(
		t,
		dir,
		"app/main.tetra",
		("module app.main\nimport lib.core.morph as morph\nfunc main() " +
			"-> Int:\n    let x: Int =\n    return 0\n"),
	)
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--diagnostics", "json",
		"--report", reportPath,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf(
			"surface dev exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout = %q, want empty on JSON diagnostic failure", stdout.String())
	}
	var cliDiag cliJSONDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &cliDiag); err != nil {
		t.Fatalf("decode CLI diagnostic: %v\n%s", err, stderr.String())
	}
	if filepath.Clean(cliDiag.File) != filepath.Clean(entry) || cliDiag.Line <= 0 ||
		cliDiag.Column <= 0 ||
		cliDiag.Severity != "error" {
		t.Fatalf("CLI diagnostic = %#v, want positioned source error for %s", cliDiag, entry)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Pass              bool `json:"pass"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Pass || len(report.SourceDiagnostics) == 0 {
		t.Fatalf("report = %#v, want failing source diagnostic", report)
	}
	first := report.SourceDiagnostics[0]
	if first.Kind != "morph" || filepath.Clean(first.Path) != filepath.Clean(entry) ||
		first.Line <= 0 ||
		first.Column <= 0 ||
		first.Severity != "error" ||
		first.Pass {
		t.Fatalf("source diagnostic = %#v, want Morph-positioned failing diagnostic", first)
	}
}

func writeSurfaceDevFixture(
	t *testing.T,
	dir string,
) (entry string, tokens string, recipes string) {
	t.Helper()
	unique := strings.ReplaceAll(filepath.Base(dir), "-", "_")
	tokens = filepath.Join(dir, "design", "tokens.tetra")
	recipes = filepath.Join(dir, "design", "recipes.tetra")
	entry = filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(
		t,
		dir,
		"design/tokens.tetra",
		"module design.tokens\n// "+unique+"\nfunc accent() -> Int:\n    return 17\n",
	)
	writeCLIProjectFile(
		t,
		dir,
		"design/recipes.tetra",
		"module design.recipes\n// "+unique+"\nfunc card() -> Int:\n    return 25\n",
	)
	writeCLIProjectFile(
		t,
		dir,
		"app/main.tetra",
		"module app.main\n// "+unique+("\nimport design.tokens as tokens\nimport design.recipes as "+
			"recipes\nfunc main() -> Int:\n    return tokens.accent() + "+
			"recipes.card()\n"),
	)
	return entry, tokens, recipes
}

func writeSurfaceDevMorphRenderedBeautyReport(t *testing.T, path string, source string) {
	t.Helper()
	report := validSurfaceDevMorphRenderedBeautyReport(source)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Morph rendered beauty report: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReport(raw); err != nil {
		t.Fatalf("test Morph rendered beauty report invalid: %v\n%s", err, raw)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write Morph rendered beauty report: %v", err)
	}
}

func validSurfaceDevMorphRenderedBeautyReport(source string) surface.MorphRenderedBeautyReport {
	blockSceneHash := surfaceDevTestSHA(5)
	commandStreamHash := surfaceDevTestSHA(7)
	frameHash := surfaceDevTestSHA(60)
	goldenHash := surfaceDevTestSHA(61)
	commands := []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
	renderCommands := make([]surface.MorphRenderedBeautyRenderCommand, 0, len(commands))
	for i, command := range commands {
		item := surface.MorphRenderedBeautyRenderCommand{
			Order:        i + 1,
			Command:      command,
			Source:       source,
			SourceNodeID: fmt.Sprintf("node-%d", i+1),
			Recipe:       "studio_shell",
			LayerID:      "layer-main",
			BlockID:      i + 1,
			Quality:      "deterministic",
			Checksum:     surfaceDevTestSHA(100 + i),
		}
		if command != "radius_clip" {
			item.Color = surfaceDevMorphRenderedBeautyCommandColor(command)
		}
		if command == "border" || command == "outline" {
			item.Width = 1
		}
		if command == "shadow" {
			item.Blur = 8
			item.OffsetY = 2
		}
		if command == "text" {
			item.RasterFormat = "builtin-5x7-alpha-mask-v1"
			item.RasterHash = surfaceDevTestSHA(210)
			item.RasterWidth = 5
			item.RasterHeight = 7
			item.RasterCoverage = 20
		}
		if command == "icon" {
			item.RasterFormat = "builtin-icon-mask-raster-v1"
			item.RasterHash = surfaceDevTestSHA(211)
			item.RasterWidth = 16
			item.RasterHeight = 16
			item.RasterCoverage = 96
		}
		renderCommands = append(renderCommands, item)
	}
	return surface.MorphRenderedBeautyReport{
		Schema:         surface.MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   surface.MorphRenderedBeautyScope,
		Target:         "headless",
		ScenarioName:   "headless-morph:" + source,
		GitHead:        strings.Repeat("1", 40),
		GitCommit:      strings.Repeat("1", 40),
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:         source,
			SourceSHA256:   surfaceDevTestSHA(1),
			CapsuleHash:    surfaceDevTestSHA(2),
			TokenGraphHash: surfaceDevTestSHA(3),
			TokenCount:     6,
			TokenCategories: []string{
				"color",
				"space",
				"radius",
				"typography",
				"motion",
				"assets",
			},
			RecipeCount:            3,
			RecipeExpansionCount:   4,
			RecipeNames:            []string{"studio_shell", "hero_panel", "toolbar"},
			ResolvedMorphSceneHash: surfaceDevTestSHA(4),
			BlockSceneSnapshotHash: blockSceneHash,
		},
		BlockSceneSnapshot: surface.MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope:         surface.MorphRenderedBeautyScope,
			Source:               source,
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			RecipeExpansionCount: 4,
			NodeCount:            12,
			RichSpecHash:         surfaceDevTestSHA(6),
			BlockSceneHash:       blockSceneHash,
			SpecCoverage: surface.MorphRenderedBeautyBlockSceneSpecCoverage{
				Layout:        true,
				Paint:         true,
				Text:          true,
				Image:         true,
				Input:         true,
				Event:         true,
				State:         true,
				Motion:        true,
				Accessibility: true,
			},
		},
		RenderEvidence: surface.MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: commandStreamHash,
			CommandCount:      len(renderCommands),
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: surface.MorphRenderedBeautyRendererStableProof{
			Schema:                         "tetra.surface.renderer-stable-proof.v1",
			PixelOwner:                     "surface-renderer",
			RendererOwned:                  true,
			BridgeOwnedPixels:              false,
			BlockFirst:                     true,
			DerivedFromRenderCommandStream: true,
			RenderCommandStreamHash:        commandStreamHash,
			BlockSceneHash:                 blockSceneHash,
			FrameChecksum:                  frameHash,
			StablePromotionEligible:        true,
		},
		RenderCommandStream: surface.MorphRenderedBeautyRenderCommandStream{
			Schema:                        "tetra.surface.render-command-stream.v1",
			Source:                        source,
			SurfaceScope:                  surface.MorphRenderedBeautyScope,
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                blockSceneHash,
			FrameChecksum:                 frameHash,
			CommandStreamHash:             commandStreamHash,
			CommandCount:                  len(renderCommands),
			SourceLinked:                  true,
			Commands:                      renderCommands,
		},
		PixelEvidence: surface.MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           "reports/surface/dev-frame.rgba",
			FrameArtifactSHA256:     frameHash,
			FrameChecksum:           frameHash,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         surfaceDevTestSHA(8),
			BlockSceneHash:          blockSceneHash,
			RenderCommandStreamHash: commandStreamHash,
			GoldenArtifact:          "reports/surface/dev-golden.rgba",
			GoldenArtifactSHA256:    goldenHash,
			GoldenChecksum:          goldenHash,
			DiffPixels:              1,
			MaxChannelDelta:         1,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected:             true,
			SelfGoldenRejected:               true,
			PrecomputedFrameRejected:         true,
			MissingFrameArtifactRejected:     true,
			NoDOMUI:                          true,
			NoCSSRuntime:                     true,
			NoReactRuntime:                   true,
			NoElectronRuntime:                true,
			NoNativeWidgets:                  true,
			NoHiddenAppState:                 true,
			NonBlockOutputRejected:           true,
			DirtyCheckoutProductionRejected:  true,
			UnsupportedTargetRejected:        true,
			RendererOwnedStableProofRequired: true,
		},
		NonClaims: []string{
			"no Electron runtime claim",
			"no React runtime claim",
			"no CSS runtime claim",
			"no DOM-authored UI claim",
			"no GPU renderer production claim",
			"no macOS production claim",
			"no Windows production claim",
		},
	}
}

func surfaceDevMorphRenderedBeautyCommandColor(command string) string {
	switch command {
	case "fill":
		return "#202733ff"
	case "gradient":
		return "#2c3848ff"
	case "image_fill":
		return "#ffffff22"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#6eaef4ff"
	}
}

func surfaceDevTestSHA(seed int) string {
	return "sha256:" + fmt.Sprintf("%064x", seed)
}

// ---- test_all_script_test.go ----

func TestTestAllScriptInterface(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	script := filepath.Join(root, "scripts", "ci", "test-all.sh")

	if out, err := exec.Command("bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, string(out))
	}

	help := exec.Command("bash", script, "--help")
	help.Dir = root
	helpOut, err := help.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, string(helpOut))
	}
	for _, want := range []string{
		"--keep-going",
		"--json-only",
		"--report-format",
		"Exit codes",
		"--report-dir",
	} {
		if !strings.Contains(string(helpOut), want) {
			t.Fatalf("help missing %q:\n%s", want, string(helpOut))
		}
	}

	bad := exec.Command("bash", script, "--definitely-not-a-real-option")
	bad.Dir = root
	badOut, err := bad.CombinedOutput()
	if err == nil {
		t.Fatalf("invalid option unexpectedly succeeded:\n%s", string(badOut))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("invalid option exit = %v, output:\n%s", err, string(badOut))
	}
}

func TestTestAllScriptKeepGoingJSONOnly(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	scriptRaw, err := os.ReadFile(filepath.Join(root, "scripts", "ci", "test-all.sh"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "release", "post_v0_4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "scripts", "ci", "test-all.sh"),
		scriptRaw,
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "scripts", "ci", "test.sh"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 1\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "scripts", "dev", "bootstrap.sh"),
		[]byte("#!/usr/bin/env bash\ncp ./tetra ./t\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(
		dir,
		"scripts",
		"release",
		"post_v0_4",
		"memory-100-prod-stable-gate.sh",
	), []byte(("#!/usr/bin/env bash\nset -euo pipefail\nreport_dir=\"\"\nwhile " +
		"[[ $# -gt 0 ]]; do case \"$1\" in --report-dir) " +
		"report_dir=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p " +
		"-- \"$report_dir\"\nprintf '{\"schema\":" +
		"\"tetra.memory-100.prod-stable.v1\",\"status\":\"pass\"}\\n' " +
		">\"$report_dir/memory-100-prod-stable-manifest.json\"\nprintf " +
		"'{\"schema\":\"tetra.artifact-hashes.v1\",\"artifacts\":[]}\\n' " +
		">\"$report_dir/artifact-hashes.json\"\n")), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(`#!/usr/bin/env bash
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/memory-fuzz-short" ]]; then
  report_dir=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --report-dir)
        report_dir="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$report_dir" ]]; then
    mkdir -p -- "$report_dir"
    cat >"$report_dir/memory-fuzz-oracle.json" <<'JSON'
{
  "schema_version": "tetra.memory-fuzz.oracle.v1",
  "scope": "memory_production_core_v1_mpc15"
}
JSON
    cat >"$report_dir/summary.md" <<'MD'
# Memory Fuzz Short Summary

- tier: Tier 1 short CI smoke
- report: memory-fuzz-oracle.json
MD
    validator_cmd="go run ./tools/cmd/validate-memory-fuzz-oracle"
    validator_cmd+=" --report <artifact-dir>/memory-fuzz-oracle.json"
    validator_cmd+=" --artifact-dir <artifact-dir>"
    cat >"$report_dir/summary.json" <<JSON
{
  "schema_version": "tetra.memory-fuzz-short.summary.v1",
  "kind": "tier1_short_ci_smoke",
  "tier": "tier1_short_ci_smoke",
  "status": "pass",
  "artifacts": {
    "oracle_report": "memory-fuzz-oracle.json",
    "summary_md": "summary.md",
    "summary_json": "summary.json"
  },
  "commands": [
    {
      "name": "memory-fuzz-short",
      "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir <artifact-dir>",
      "status": "pass"
    },
    {
      "name": "validate-memory-fuzz-oracle",
      "command": "$validator_cmd",
      "status": "pass"
    }
  ]
}
JSON
  fi
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/ram-contract-fuzz-short" ]]; then
  report_dir=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --report-dir)
        report_dir="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$report_dir" ]]; then
    mkdir -p -- "$report_dir"
    cat >"$report_dir/ram-contract-fuzz-oracle.json" <<'JSON'
{
  "schema_version": "tetra.ram-contract-fuzz-oracle.v1",
  "observations": [],
  "summary": {
    "mutations": 0,
    "rejected": 0
  },
  "non_claims": [
    "not a full formal proof"
  ]
}
JSON
    cat >"$report_dir/ram-contract-report.json" <<'JSON'
{
  "schema_version": "tetra.ram-contract-report.v1",
  "rows": []
}
JSON
    cat >"$report_dir/memory-grade-report.json" <<'JSON'
{
  "schema_version": "tetra.memory-grade-report.v1"
}
JSON
    cat >"$report_dir/proof-store-summary.json" <<'JSON'
{
  "schema_version": "tetra.proof-store-summary.v1",
  "proofs": [],
  "summary": {
    "proof_count": 0,
    "proven": 0,
    "conservative": 0,
    "rejected": 0,
    "unknown": 0
  },
  "non_claims": [
    "no full formal proof claim"
  ]
}
JSON
    cat >"$report_dir/validation-pipeline-coverage.json" <<'JSON'
{
  "schema_version": "tetra.validation-pipeline-coverage.v1",
  "entries": []
}
JSON
    cat >"$report_dir/heap-blockers.json" <<'JSON'
{
  "schema_version": "tetra.ram-blockers.v1",
  "kind": "heap",
  "rows": []
}
JSON
    cat >"$report_dir/copy-blockers.json" <<'JSON'
{
  "schema_version": "tetra.ram-blockers.v1",
  "kind": "copy",
  "rows": []
}
JSON
  fi
  exit 0
fi
if [[ "${1:-}" == "test" ]]; then
  pkg="${2:-}"
  shift 2 || true
  list_mode=false
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -list)
        list_mode=true
        shift 2 || true
        ;;
      -list=*)
        list_mode=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ "$list_mode" == true ]]; then
    case "$pkg" in
      ./compiler/internal/memoryfacts)
        cat <<'TESTS'
TestMemoryFactsRejectsUnsafeUnknownToSafeKnown
TestMemoryFactsRejectsDirectSafeBorrowedFromUnsafeUnknown
TestMemoryFactsRejectsDirectSafeOwnedFromUnsafeUnknown
TestMemoryFactsRejectsUnsafeUnknownNoAliasAndBoundsProofClaims
TestMemoryFactsRejectsUnsafeCheckedGenericPromotions
TestMemoryFactsRejectsUnsafeVerifiedRootGenericClaims
TestMemoryFactsRejectsValidatedUnsafeUnknownTrustedStorage
TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims
TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotions
TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaims
TestValidateMemoryReportRejectsValidatedUnsafeUnknownTrustedStorage
TestMemoryIdealV6ProjectsBoundsProofFacts
TestMemoryIdealV6ProjectsMissingProofRejection
TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent
TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
TESTS
        ;;
      ./compiler/cmd/validate-memory-report)
        cat <<'TESTS'
TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown
TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim
TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotion
TestValidateMemoryReportRejectsUnsafeUnknownZeroCost
TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaim
TestValidateMemoryReportRejectsUnsafeUnknownTrustedStorage
TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent
TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
TESTS
        ;;
      ./compiler)
        cat <<'TESTS'
TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants
TestClassifyMemoryFuzzOracleObservation
TestValidateMemoryFuzzOracleReportRejectsDrift
TestMemoryFuzzOracleReportCoversV12ReleaseEvidence
TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift
TestBuildBoundsAndProofReportsShowWhileRangeReason
TESTS
        ;;
      ./compiler/internal/validation)
        cat <<'TESTS'
TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID
TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof
TestValidateTranslationRejectsMissingProofIDAfterTransform
TESTS
        ;;
      ./compiler/internal/plir)
        printf '%s\n' TestVerifierRejectsUnknownProofUse TestVerifierRejectsNonDominatingProofUse
        ;;
      ./compiler/internal/lower)
        cat <<'TESTS'
TestForSliceLoopUsesProofTaggedUncheckedIndexLoad
TestWhileLessThanLenUsesProofTaggedUncheckedIndexLoad
TestCopyLoopSourceLoadUsesProofTaggedUncheckedIndexLoad
TESTS
        ;;
      ./tools/cmd/validate-memory-fuzz-oracle)
        cat <<'TESTS'
TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport
TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle
TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport
TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence
TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary
TestValidateMemoryFuzzOracleReportFileRejectsMissingValidatorProvenance
TESTS
        ;;
      ./tools/cmd/memory-fuzz-short)
        cat <<'TESTS'
TestRunMemoryFuzzShortWritesValidatedArtifacts
TestRunMemoryFuzzShortRejectsUnsupportedTier
TestRunMemoryFuzzShortRejectsStaleReportDir
TESTS
        ;;
      ./tools/cmd/ram-contract-fuzz-short)
        cat <<'TESTS'
TestRunRAMContractFuzzShortWritesValidatedArtifacts
TestRunRAMContractFuzzShortRejectsStaleReportDir
TESTS
        ;;
      ./tools/cmd/validate-ram-contract-fuzz-oracle)
        cat <<'TESTS'
TestValidateRAMContractFuzzOracleAcceptsArtifactBundle
TestValidateRAMContractFuzzOracleRejectsMissingReport
TESTS
        ;;
      ./tools/cmd/validate-ram-contract-report)
        cat <<'TESTS'
TestValidateRAMContractReportFileAcceptsCompilerReport
TestValidateRAMContractReportRejectsMissingBlocker
TESTS
        ;;
      ./compiler/internal/ramcontract)
        cat <<'TESTS'
TestRAMContractFromAllocPlanTracksRowsAndBlockers
TestRAMContractRejectsMissingBlockerExplanation
TestRAMContractEnforcementFailsForHeap
TESTS
        ;;
      ./cli/internal/actornet)
        cat <<'TESTS'
TestBrokerCloseReopenWithoutGoroutineLeak
TestBrokerCloseWithoutCancelStopsServeWatcher
TestBrokerRoutesFramesBetweenLoopbackNodesAndWritesReport
TestBrokerReportsNodeDownForMissingDestination
TESTS
        ;;
    esac
    exit 0
  fi
  if [[ "$pkg" == "./compiler/..." ]]; then
    exit 1
  fi
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "HEAD" ]]; then
  echo "e2c19b8ee276158f8eb2c54cf61e11bd84952893"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tetra"), []byte(`#!/usr/bin/env bash
case "$1" in
  version) echo "v0.3.0"; exit 0 ;;
  fmt|test|smoke) exit 0 ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*)
            cat >&2 <<'JSON'
{
  "code": "TETRA2001",
  "message": "function main uses effect 'io' but does not declare it",
  "severity": "error"
}
JSON
            ;;
          *tabs-diagnostic.tetra*)
            cat >&2 <<'JSON'
{
  "code": "TETRA0001",
  "message": "tabs are not supported in Flow indentation",
  "severity": "error"
}
JSON
            ;;
          *planned-actor-diagnostic.tetra*)
            cat >&2 <<'JSON'
{
  "code": "TETRA0001",
  "message": "actor declarations currently support state fields and func methods only",
  "severity": "error"
}
JSON
            ;;
          *)
            cat >&2 <<'JSON'
{
  "code": "TETRA2001",
  "message": "unknown function missing_call",
  "severity": "error"
}
JSON
            ;;
        esac
        exit 1
      fi
    done
    exit 0
    ;;
  build)
    out=""
    prev=""
    for arg in "$@"; do
      if [[ "$prev" == "-o" ]]; then
        out="$arg"
      fi
      prev="$arg"
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      printf '\x00\x61\x73\x6d\x01\x00\x00\x00' >"$out"
    fi
    exit 0
    ;;
  targets)
    cat <<'JSON'
{
  "supported": [
    "linux-x64",
    "windows-x64",
    "macos-x64"
  ],
  "build_only": [
    "wasm32-wasi",
    "wasm32-web"
  ],
  "planned": []
}
JSON
    exit 0
    ;;
  doctor)
    cat <<'JSON'
{
  "status": "pass",
  "checks": [
    {"name": "version", "status": "pass"},
    {"name": "supported targets", "status": "pass"},
    {"name": "build-only targets", "status": "pass"},
    {"name": "planned targets", "status": "pass"},
    {"name": "repo root", "status": "pass"},
    {"name": "__rt/actors_sysv.tetra", "status": "pass"},
    {"name": "__rt/actors_win64.tetra", "status": "pass"},
    {"name": "compiler/selfhostrt/actors_sysv.tetra", "status": "pass"},
    {"name": "compiler/selfhostrt/actors_win64.tetra", "status": "pass"},
    {"name": "examples/flow/flow_hello.tetra", "status": "pass"},
    {"name": "docs/generated/manifest.json", "status": "pass"},
    {"name": "docs manifest version", "status": "pass"},
    {"name": "docs manifest surface", "status": "pass"},
    {"name": "smoke sources", "status": "pass"},
    {"name": "runtime exports", "status": "pass"},
    {"name": "target metadata", "status": "pass"},
    {"name": "tooling commands", "status": "pass"}
  ]
}
JSON
    exit 0
    ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(dir, "report")
	cmd := exec.Command(
		"bash",
		"scripts/ci/test-all.sh",
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failing keep-going run, got success:\n%s", string(out))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("exit = %v, output:\n%s", err, string(out))
	}

	var summary struct {
		Status string `json:"status"`
		Steps  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(out, &summary); err != nil {
		t.Fatalf("summary JSON: %v\n%s", err, string(out))
	}
	if summary.Status != "fail" || len(summary.Steps) != 19 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.Steps[0].Name != "go test all packages" || summary.Steps[0].Status != "fail" {
		t.Fatalf("first step = %#v", summary.Steps[0])
	}
	if summary.Steps[1].Name != "unsafe promotion blocker suite" ||
		summary.Steps[1].Status != "pass" {
		t.Fatalf("unsafe promotion blocker step = %#v", summary.Steps[1])
	}
	if summary.Steps[2].Name != "bounds proof blocker suite" || summary.Steps[2].Status != "pass" {
		t.Fatalf("bounds proof blocker step = %#v", summary.Steps[2])
	}
	if summary.Steps[3].Name != "memory fuzz oracle artifact gate" ||
		summary.Steps[3].Status != "pass" {
		t.Fatalf("memory fuzz oracle gate step = %#v", summary.Steps[3])
	}
	if summary.Steps[4].Name != "RAM contract fuzz oracle artifact gate" ||
		summary.Steps[4].Status != "pass" {
		t.Fatalf("RAM contract fuzz oracle gate step = %#v", summary.Steps[4])
	}
	if summary.Steps[5].Name != "host leak blocker suite" || summary.Steps[5].Status != "pass" {
		t.Fatalf("host leak blocker step = %#v", summary.Steps[5])
	}
	if summary.Steps[6].Name != "Memory100 prod-stable gate" || summary.Steps[6].Status != "pass" {
		t.Fatalf("Memory100 prod-stable gate step = %#v", summary.Steps[6])
	}
	if summary.Steps[len(summary.Steps)-1].Name != "host smoke linux-x64" ||
		summary.Steps[len(summary.Steps)-1].Status != "pass" {
		t.Fatalf("last step = %#v", summary.Steps[len(summary.Steps)-1])
	}
	if _, err := os.Stat(filepath.Join(reportDir, "summary.md")); err != nil {
		t.Fatalf("missing summary.md: %v", err)
	}
}

// ---- test_command_test.go ----

func TestTestCommandJSONDiagnosticsForWASMRuntimeUnsupported(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"test", "--diagnostics=json", "--target", "wasm32-web"},
		2,
	)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s severity error",
			diag,
			compiler.DiagnosticCodeTargetRuntime,
		)
	}
	for _, want := range []string{
		"cannot run tests for target wasm32-web",
		"WASM test runner is not part of the current production runtime contract",
		"smoke/runtime reports",
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported(t *testing.T) {
	restore := stubLinuxX32HostSupport(false)
	defer restore()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(
		srcPath,
		[]byte("test \"math\":\n    expect 40 + 2 == 42\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"test", "--diagnostics=json", "--target", "x32", srcPath},
		2,
	)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s severity error",
			diag,
			compiler.DiagnosticCodeTargetRuntime,
		)
	}
	for _, want := range []string{
		"cannot run tests for target linux-x32",
		expectedLinuxX32HostUnsupportedReason(t),
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandRunsLinuxX32SourceTestsWhenProbePasses(t *testing.T) {
	restoreHost := stubLinuxX32HostSupport(true)
	defer restoreHost()
	restoreExec := stubNativeExec(func(path string, stdout io.Writer, stderr io.Writer) int {
		if err := requireX32ExecutableFile(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	})
	defer restoreExec()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(
		srcPath,
		[]byte("test \"math\":\n    expect 40 + 2 == 42\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	for _, want := range []string{"PASS math", "Tetra tests: 1/1 passed"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("test stdout missing %q: %q", want, stdout.String())
		}
	}
}

func TestTestCommandRunsDefaultTargetSuitesWithoutProject(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   []string
	}{
		{
			target: "x86",
			want: []string{
				"PASS x86 target model",
				"PASS x86 i386 SysV classifier",
				"PASS x86 varargs and sret ABI",
				"PASS x86 pointer FFI object smoke",
				"PASS x86 c_int FFI object smoke",
				"PASS x86 c_uint FFI object smoke",
				"PASS x86 ILP32 native/libc FFI object smoke",
				"PASS x86 ref FFI null-return diagnostics",
				"PASS x86 function-pointer FFI diagnostics",
				"PASS x86 source native scalar diagnostics",
				"PASS x86 stdout executable smoke",
				"PASS x86 stderr fd runtime smoke",
				"PASS x86 allocator executable smoke",
				"PASS x86 allocator failure executable smoke",
				"PASS x86 raw memory bounds executable smoke",
				"PASS x86 raw pointer slot executable smoke",
				"PASS x86 raw pointer offset slot executable smoke",
				"PASS x86 island free executable smoke",
				"PASS x86 stdlib runtime boundary diagnostics",
				"PASS x86 filesystem runtime smoke",
				"PASS x86 filesystem scheduler composition smoke",
				"PASS x86 time runtime smoke",
				"PASS x86 single-actor self-host runtime smoke",
				"PASS x86 single-task self-host runtime smoke",
				"PASS x86 typed-task self-host runtime smoke",
				"PASS x86 staged typed-task self-host runtime smoke",
				"PASS x86 task-group self-host runtime smoke",
				"PASS x86 typed-task-group self-host runtime smoke",
				"PASS x86 actor-state self-host runtime smoke",
				"PASS x86 ctx_switch object smoke",
				"PASS x86 target runtime boundary diagnostics",
				"PASS x86 networking runtime boundary diagnostics",
				"PASS x86 networking lifecycle runtime smoke",
				"PASS x86 surface/distributed runtime boundary diagnostics",
				"PASS x86 pointer atomic ABI width",
				"PASS x86 object ABI smoke",
				"PASS x86 atomic ABI object",
				"PASS x86 executable matrix smoke",
				"Tetra tests: 38/38 passed",
			},
		},
		{
			target: "x64",
			want: []string{
				"PASS x64 target model",
				"PASS x64 SysV classifier",
				"PASS x64 SysV varargs and aggregates",
				"PASS x64 source native scalar diagnostics",
				"PASS x64 pointer FFI regression smoke",
				"PASS x64 c_int FFI object smoke",
				"PASS x64 c_uint FFI object smoke",
				"PASS x64 filesystem scheduler composition smoke",
				"PASS x64 networking runtime smoke",
				"PASS x64 scheduler restriction regression smoke",
				"PASS x64 pointer atomic ABI width",
				"PASS x64 object ABI smoke",
				"PASS x64 atomic ABI object",
				"PASS x64 executable matrix smoke",
				"Tetra tests: 14/14 passed",
			},
		},
		{
			target: "x32",
			want: []string{
				"PASS x32 target model",
				"PASS x32 SysV classifier",
				"PASS x32 SysV varargs and aggregates",
				"PASS x32 pointer FFI object smoke",
				"PASS x32 c_int FFI object smoke",
				"PASS x32 c_uint FFI object smoke",
				"PASS x32 ILP32 native/libc FFI object smoke",
				"PASS x32 ref FFI null-return diagnostics",
				"PASS x32 function-pointer FFI diagnostics",
				"PASS x32 source native scalar diagnostics",
				"PASS x32 stdout executable smoke",
				"PASS x32 stderr fd runtime smoke",
				"PASS x32 allocator executable smoke",
				"PASS x32 allocator failure executable smoke",
				"PASS x32 raw memory bounds executable smoke",
				"PASS x32 raw pointer slot executable smoke",
				"PASS x32 raw pointer offset slot executable smoke",
				"PASS x32 island free executable smoke",
				"PASS x32 stdlib runtime boundary diagnostics",
				"PASS x32 time runtime smoke",
				"PASS x32 filesystem runtime smoke",
				"PASS x32 filesystem scheduler composition smoke",
				"PASS x32 single-actor self-host runtime smoke",
				"PASS x32 single-task self-host runtime smoke",
				"PASS x32 typed-task self-host runtime smoke",
				"PASS x32 staged typed-task self-host runtime smoke",
				"PASS x32 task-group self-host runtime smoke",
				"PASS x32 typed-task-group self-host runtime smoke",
				"PASS x32 actor-state self-host runtime smoke",
				"PASS x32 ctx_switch object smoke",
				"PASS x32 target runtime boundary diagnostics",
				"PASS x32 networking runtime boundary diagnostics",
				"PASS x32 networking lifecycle runtime smoke",
				"PASS x32 surface/distributed runtime boundary diagnostics",
				"PASS x32 pointer atomic ABI width",
				"PASS x32 object ABI smoke",
				"PASS x32 atomic ABI object",
				"PASS x32 executable matrix smoke",
				"Tetra tests: 38/38 passed",
			},
		},
	} {
		t.Run(tc.target, func(t *testing.T) {
			dir := t.TempDir()
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
			code := runCLI([]string{"test", "--target", tc.target}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf(
					"test exit code = %d, stdout=%q stderr=%q",
					code,
					stdout.String(),
					stderr.String(),
				)
			}
			if stderr.Len() != 0 {
				t.Fatalf("test stderr = %q", stderr.String())
			}
			out := stdout.String()
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("test stdout missing %q: %q", want, out)
				}
			}
		})
	}
}

func TestTestCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--target", target}, 2)
	if diag.Code != compiler.DiagnosticCodeTargetRuntime || diag.Severity != "error" ||
		!strings.Contains(diag.Message, "cannot run tests for target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandJSONDiagnosticsForUnsupportedReportFormat(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--report=yaml"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "unsupported --report format" ||
		diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandRunsAllTargetsBrutalSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--all-targets", "--brutal"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 target model",
		"PASS x86 pointer FFI object smoke",
		"PASS x86 c_int FFI object smoke",
		"PASS x86 c_uint FFI object smoke",
		"PASS x86 ILP32 native/libc FFI object smoke",
		"PASS x86 ref FFI null-return diagnostics",
		"PASS x86 function-pointer FFI diagnostics",
		"PASS x86 source native scalar diagnostics",
		"PASS x86 stdout executable smoke",
		"PASS x86 stderr fd runtime smoke",
		"PASS x86 allocator executable smoke",
		"PASS x86 allocator failure executable smoke",
		"PASS x86 raw memory bounds executable smoke",
		"PASS x86 raw pointer slot executable smoke",
		"PASS x86 raw pointer offset slot executable smoke",
		"PASS x86 island free executable smoke",
		"PASS x86 stdlib runtime boundary diagnostics",
		"PASS x86 filesystem runtime smoke",
		"PASS x86 filesystem scheduler composition smoke",
		"PASS x86 time runtime smoke",
		"PASS x86 typed-task self-host runtime smoke",
		"PASS x86 staged typed-task self-host runtime smoke",
		"PASS x86 task-group self-host runtime smoke",
		"PASS x86 typed-task-group self-host runtime smoke",
		"PASS x86 ctx_switch object smoke",
		"PASS x86 target runtime boundary diagnostics",
		"PASS x86 networking runtime boundary diagnostics",
		"PASS x86 networking lifecycle runtime smoke",
		"PASS x86 surface/distributed runtime boundary diagnostics",
		"PASS x86 pointer atomic ABI width",
		"PASS x64 atomic object matrix",
		"PASS x64 pointer atomic object width",
		"PASS x64 source native scalar diagnostics",
		"PASS x64 pointer FFI regression smoke",
		"PASS x64 c_int FFI object smoke",
		"PASS x64 c_uint FFI object smoke",
		"PASS x64 filesystem scheduler composition smoke",
		"PASS x64 networking runtime smoke",
		"PASS x64 scheduler restriction regression smoke",
		"PASS x64 pointer atomic ABI width",
		"PASS x32 layout fuzz",
		"PASS x64 layout fuzz",
		"PASS x64 object signature fuzz",
		"PASS x86 atomic validation matrix",
		"PASS x86 atomic object matrix",
		"PASS x86 pointer atomic object width",
		"PASS x86 layout fuzz",
		"PASS x86 object signature fuzz",
		"PASS x32 SysV classifier",
		"PASS x32 SysV varargs and aggregates",
		"PASS x32 pointer FFI object smoke",
		"PASS x32 c_int FFI object smoke",
		"PASS x32 c_uint FFI object smoke",
		"PASS x32 ILP32 native/libc FFI object smoke",
		"PASS x32 ref FFI null-return diagnostics",
		"PASS x32 function-pointer FFI diagnostics",
		"PASS x32 source native scalar diagnostics",
		"PASS x32 stdout executable smoke",
		"PASS x32 stderr fd runtime smoke",
		"PASS x32 allocator executable smoke",
		"PASS x32 allocator failure executable smoke",
		"PASS x32 raw memory bounds executable smoke",
		"PASS x32 raw pointer slot executable smoke",
		"PASS x32 raw pointer offset slot executable smoke",
		"PASS x32 island free executable smoke",
		"PASS x32 stdlib runtime boundary diagnostics",
		"PASS x32 time runtime smoke",
		"PASS x32 filesystem runtime smoke",
		"PASS x32 filesystem scheduler composition smoke",
		"PASS x32 single-actor self-host runtime smoke",
		"PASS x32 single-task self-host runtime smoke",
		"PASS x32 typed-task self-host runtime smoke",
		"PASS x32 staged typed-task self-host runtime smoke",
		"PASS x32 task-group self-host runtime smoke",
		"PASS x32 typed-task-group self-host runtime smoke",
		"PASS x32 actor-state self-host runtime smoke",
		"PASS x32 ctx_switch object smoke",
		"PASS x32 target runtime boundary diagnostics",
		"PASS x32 networking runtime boundary diagnostics",
		"PASS x32 networking lifecycle runtime smoke",
		"PASS x32 surface/distributed runtime boundary diagnostics",
		"PASS x32 pointer atomic ABI width",
		"PASS x32 pointer atomic object width",
		"PASS macos-x64 SysV classifier",
		"PASS macos-x64 object ABI smoke",
		"PASS macos-x64 source native scalar diagnostics",
		"PASS macos-x64 pointer atomic ABI width",
		"PASS windows-x64 Win64 classifier",
		"PASS windows-x64 Win64 varargs and aggregates",
		"PASS windows-x64 object ABI smoke",
		"PASS windows-x64 source native scalar diagnostics",
		"PASS windows-x64 pointer atomic ABI width",
		"PASS macos-x64 atomic object matrix",
		"PASS macos-x64 pointer atomic object width",
		"PASS windows-x64 atomic object matrix",
		"PASS windows-x64 pointer atomic object width",
		"PASS x32 atomic concurrency stress oracle",
		"PASS macos-x64 object signature fuzz",
		"PASS windows-x64 object signature fuzz",
		"Tetra tests: 142/142 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "FAIL x64 fuzz") {
		t.Fatalf("test stdout still reports x64 fuzz as unsupported: %q", out)
	}
}

func TestTestCommandAllTargetsBrutalJSONUsesTargetSpecificFiles(t *testing.T) {
	assertAllTargetsBrutalJSONReport(
		t,
		[]string{"test", "--all-targets", "--brutal", "--report=json"},
	)
}

func TestTestCommandAllTargetsBrutalFormatJSONUsesTargetSpecificFiles(t *testing.T) {
	assertAllTargetsBrutalJSONReport(
		t,
		[]string{"test", "--all-targets", "--brutal", "--format=json"},
	)
}

func assertAllTargetsBrutalJSONReport(t *testing.T, args []string) {
	t.Helper()
	var report struct {
		Total  int    `json:"total"`
		Passed int    `json:"passed"`
		Failed int    `json:"failed"`
		Target string `json:"target"`
		Files  []struct {
			Filename string `json:"filename"`
		} `json:"files"`
		Results []struct {
			Name         string `json:"name"`
			Filename     string `json:"filename"`
			Index        int    `json:"index"`
			FunctionName string `json:"function_name"`
			Passed       bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, args, 0, &report)
	if report.Total != 142 || report.Passed != 142 || report.Failed != 0 ||
		len(report.Results) != 142 {
		t.Fatalf("report = %#v", report)
	}
	files := map[string]bool{}
	for _, file := range report.Files {
		files[file.Filename] = true
	}
	for _, want := range []string{
		"tetra:x64-abi",
		"tetra:macos-x64-abi",
		"tetra:windows-x64-abi",
		"tetra:x64-atomic-stress",
		"tetra:macos-x64-atomic-stress",
		"tetra:windows-x64-atomic-stress",
		"tetra:x64-fuzz",
		"tetra:macos-x64-fuzz",
		"tetra:windows-x64-fuzz",
	} {
		if !files[want] {
			t.Fatalf("report files missing %q: %#v", want, report.Files)
		}
	}
	wantFilenameByName := map[string]string{
		"x64 SysV classifier":                                  "tetra:x64-abi",
		"x64 pointer FFI regression smoke":                     "tetra:x64-abi",
		"x64 c_int FFI object smoke":                           "tetra:x64-abi",
		"x64 c_uint FFI object smoke":                          "tetra:x64-abi",
		"x64 filesystem scheduler composition smoke":           "tetra:x64-abi",
		"x64 networking runtime smoke":                         "tetra:x64-abi",
		"x64 scheduler restriction regression smoke":           "tetra:x64-abi",
		"x64 pointer atomic ABI width":                         "tetra:x64-abi",
		"x86 pointer FFI object smoke":                         "tetra:x86-abi",
		"x86 c_int FFI object smoke":                           "tetra:x86-abi",
		"x86 c_uint FFI object smoke":                          "tetra:x86-abi",
		"x86 ILP32 native/libc FFI object smoke":               "tetra:x86-abi",
		"x86 stdout executable smoke":                          "tetra:x86-abi",
		"x86 stderr fd runtime smoke":                          "tetra:x86-abi",
		"x86 allocator executable smoke":                       "tetra:x86-abi",
		"x86 allocator failure executable smoke":               "tetra:x86-abi",
		"x86 raw memory bounds executable smoke":               "tetra:x86-abi",
		"x86 raw pointer slot executable smoke":                "tetra:x86-abi",
		"x86 raw pointer offset slot executable smoke":         "tetra:x86-abi",
		"x86 island free executable smoke":                     "tetra:x86-abi",
		"x86 filesystem runtime smoke":                         "tetra:x86-abi",
		"x86 filesystem scheduler composition smoke":           "tetra:x86-abi",
		"x86 single-actor self-host runtime smoke":             "tetra:x86-abi",
		"x86 single-task self-host runtime smoke":              "tetra:x86-abi",
		"x86 typed-task self-host runtime smoke":               "tetra:x86-abi",
		"x86 staged typed-task self-host runtime smoke":        "tetra:x86-abi",
		"x86 task-group self-host runtime smoke":               "tetra:x86-abi",
		"x86 typed-task-group self-host runtime smoke":         "tetra:x86-abi",
		"x86 actor-state self-host runtime smoke":              "tetra:x86-abi",
		"x86 networking runtime boundary diagnostics":          "tetra:x86-abi",
		"x86 networking lifecycle runtime smoke":               "tetra:x86-abi",
		"x86 surface/distributed runtime boundary diagnostics": "tetra:x86-abi",
		"x32 pointer FFI object smoke":                         "tetra:x32-abi",
		"x32 c_int FFI object smoke":                           "tetra:x32-abi",
		"x32 c_uint FFI object smoke":                          "tetra:x32-abi",
		"x32 ILP32 native/libc FFI object smoke":               "tetra:x32-abi",
		"x32 time runtime smoke":                               "tetra:x32-abi",
		"x32 filesystem runtime smoke":                         "tetra:x32-abi",
		"x32 stdout executable smoke":                          "tetra:x32-abi",
		"x32 stderr fd runtime smoke":                          "tetra:x32-abi",
		"x32 allocator executable smoke":                       "tetra:x32-abi",
		"x32 allocator failure executable smoke":               "tetra:x32-abi",
		"x32 raw memory bounds executable smoke":               "tetra:x32-abi",
		"x32 raw pointer slot executable smoke":                "tetra:x32-abi",
		"x32 raw pointer offset slot executable smoke":         "tetra:x32-abi",
		"x32 island free executable smoke":                     "tetra:x32-abi",
		"x32 filesystem scheduler composition smoke":           "tetra:x32-abi",
		"x32 single-actor self-host runtime smoke":             "tetra:x32-abi",
		"x32 single-task self-host runtime smoke":              "tetra:x32-abi",
		"x32 typed-task self-host runtime smoke":               "tetra:x32-abi",
		"x32 staged typed-task self-host runtime smoke":        "tetra:x32-abi",
		"x32 task-group self-host runtime smoke":               "tetra:x32-abi",
		"x32 typed-task-group self-host runtime smoke":         "tetra:x32-abi",
		"x32 actor-state self-host runtime smoke":              "tetra:x32-abi",
		"x32 ctx_switch object smoke":                          "tetra:x32-abi",
		"x32 networking runtime boundary diagnostics":          "tetra:x32-abi",
		"x32 networking lifecycle runtime smoke":               "tetra:x32-abi",
		"x32 surface/distributed runtime boundary diagnostics": "tetra:x32-abi",
		"macos-x64 SysV classifier":                            "tetra:macos-x64-abi",
		"macos-x64 object ABI smoke":                           "tetra:macos-x64-abi",
		"macos-x64 pointer atomic ABI width":                   "tetra:macos-x64-abi",
		"windows-x64 Win64 classifier":                         "tetra:windows-x64-abi",
		"windows-x64 object ABI smoke":                         "tetra:windows-x64-abi",
		"windows-x64 pointer atomic ABI width":                 "tetra:windows-x64-abi",
		"x64 atomic object matrix":                             "tetra:x64-atomic-stress",
		"x64 pointer atomic object width":                      "tetra:x64-atomic-stress",
		"x64 atomic concurrency stress oracle":                 "tetra:x64-atomic-stress",
		"macos-x64 atomic object matrix":                       "tetra:macos-x64-atomic-stress",
		"macos-x64 pointer atomic object width":                "tetra:macos-x64-atomic-stress",
		"macos-x64 atomic concurrency stress oracle":           "tetra:macos-x64-atomic-stress",
		"windows-x64 atomic object matrix":                     "tetra:windows-x64-atomic-stress",
		"windows-x64 pointer atomic object width":              "tetra:windows-x64-atomic-stress",
		"windows-x64 atomic concurrency stress oracle":         "tetra:windows-x64-atomic-stress",
		"x64 object signature fuzz":                            "tetra:x64-fuzz",
		"macos-x64 object signature fuzz":                      "tetra:macos-x64-fuzz",
		"windows-x64 object signature fuzz":                    "tetra:windows-x64-fuzz",
	}
	for name, wantFile := range wantFilenameByName {
		found := false
		for _, result := range report.Results {
			if result.Name == name {
				found = true
				if result.Filename != wantFile || !result.Passed {
					t.Fatalf("result %q = %#v, want filename %q and passed", name, result, wantFile)
				}
				if !strings.HasPrefix(result.FunctionName, "__tetra_test_") {
					t.Fatalf(
						"result %q function_name = %q, want __tetra_test_ prefix",
						name,
						result.FunctionName,
					)
				}
			}
		}
		if !found {
			t.Fatalf("report missing result %q: %#v", name, report.Results)
		}
	}
	prevOrderKey := ""
	for _, result := range report.Results {
		orderKey := fmt.Sprintf("%s\x00%08d", result.Filename, result.Index)
		if prevOrderKey != "" && orderKey < prevOrderKey {
			t.Fatalf(
				"results are not sorted by filename then index: previous=%q current=%q",
				prevOrderKey,
				orderKey,
			)
		}
		prevOrderKey = orderKey
	}
}

func TestTestCommandJSONDiagnosticsForTargetSpecificSuiteUnsupported(t *testing.T) {
	diag := runCLIJSONDiagnostic(
		t,
		[]string{"test", "--diagnostics=json", "--target", "x32", "--abi", "--atomic-stress"},
		2,
	)
	for _, want := range []string{
		"--abi",
		"--atomic-stress",
		"linux-x32",
		"ABI torture",
		"atomic stress",
		"not implemented yet",
		"no fake or skipped tests",
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandRunsX32FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 layout fuzz",
		"PASS x32 object signature fuzz",
		"PASS x32 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 layout fuzz",
		"PASS x64 object signature fuzz",
		"PASS x64 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 layout fuzz",
		"PASS x86 object signature fuzz",
		"PASS x86 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX32AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 atomic validation matrix",
		"PASS x32 atomic object matrix",
		"PASS x32 pointer atomic object width",
		"PASS x32 atomic concurrency stress oracle",
		"PASS x32 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 atomic validation matrix",
		"PASS x64 atomic object matrix",
		"PASS x64 pointer atomic object width",
		"PASS x64 atomic concurrency stress oracle",
		"PASS x64 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 atomic validation matrix",
		"PASS x86 atomic object matrix",
		"PASS x86 pointer atomic object width",
		"PASS x86 atomic concurrency stress oracle",
		"PASS x86 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX32ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 target model",
		"PASS x32 SysV classifier",
		"PASS x32 SysV varargs and aggregates",
		"PASS x32 pointer FFI object smoke",
		"PASS x32 c_int FFI object smoke",
		"PASS x32 c_uint FFI object smoke",
		"PASS x32 ILP32 native/libc FFI object smoke",
		"PASS x32 ref FFI null-return diagnostics",
		"PASS x32 function-pointer FFI diagnostics",
		"PASS x32 source native scalar diagnostics",
		"PASS x32 stdout executable smoke",
		"PASS x32 stderr fd runtime smoke",
		"PASS x32 allocator executable smoke",
		"PASS x32 allocator failure executable smoke",
		"PASS x32 raw memory bounds executable smoke",
		"PASS x32 raw pointer slot executable smoke",
		"PASS x32 raw pointer offset slot executable smoke",
		"PASS x32 island free executable smoke",
		"PASS x32 stdlib runtime boundary diagnostics",
		"PASS x32 time runtime smoke",
		"PASS x32 filesystem runtime smoke",
		"PASS x32 filesystem scheduler composition smoke",
		"PASS x32 single-actor self-host runtime smoke",
		"PASS x32 single-task self-host runtime smoke",
		"PASS x32 typed-task self-host runtime smoke",
		"PASS x32 staged typed-task self-host runtime smoke",
		"PASS x32 task-group self-host runtime smoke",
		"PASS x32 typed-task-group self-host runtime smoke",
		"PASS x32 actor-state self-host runtime smoke",
		"PASS x32 ctx_switch object smoke",
		"PASS x32 target runtime boundary diagnostics",
		"PASS x32 networking runtime boundary diagnostics",
		"PASS x32 networking lifecycle runtime smoke",
		"PASS x32 surface/distributed runtime boundary diagnostics",
		"PASS x32 pointer atomic ABI width",
		"PASS x32 object ABI smoke",
		"PASS x32 atomic ABI object",
		"PASS x32 executable matrix smoke",
		"Tetra tests: 38/38 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 target model",
		"PASS x86 i386 SysV classifier",
		"PASS x86 varargs and sret ABI",
		"PASS x86 pointer FFI object smoke",
		"PASS x86 c_int FFI object smoke",
		"PASS x86 c_uint FFI object smoke",
		"PASS x86 ILP32 native/libc FFI object smoke",
		"PASS x86 ref FFI null-return diagnostics",
		"PASS x86 function-pointer FFI diagnostics",
		"PASS x86 source native scalar diagnostics",
		"PASS x86 stdout executable smoke",
		"PASS x86 stderr fd runtime smoke",
		"PASS x86 allocator executable smoke",
		"PASS x86 allocator failure executable smoke",
		"PASS x86 raw memory bounds executable smoke",
		"PASS x86 raw pointer slot executable smoke",
		"PASS x86 raw pointer offset slot executable smoke",
		"PASS x86 island free executable smoke",
		"PASS x86 stdlib runtime boundary diagnostics",
		"PASS x86 filesystem runtime smoke",
		"PASS x86 filesystem scheduler composition smoke",
		"PASS x86 time runtime smoke",
		"PASS x86 single-actor self-host runtime smoke",
		"PASS x86 single-task self-host runtime smoke",
		"PASS x86 typed-task self-host runtime smoke",
		"PASS x86 staged typed-task self-host runtime smoke",
		"PASS x86 task-group self-host runtime smoke",
		"PASS x86 typed-task-group self-host runtime smoke",
		"PASS x86 actor-state self-host runtime smoke",
		"PASS x86 ctx_switch object smoke",
		"PASS x86 target runtime boundary diagnostics",
		"PASS x86 networking runtime boundary diagnostics",
		"PASS x86 networking lifecycle runtime smoke",
		"PASS x86 surface/distributed runtime boundary diagnostics",
		"PASS x86 pointer atomic ABI width",
		"PASS x86 object ABI smoke",
		"PASS x86 atomic ABI object",
		"PASS x86 executable matrix smoke",
		"Tetra tests: 38/38 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsLinuxX86SourceTestsWhenKernelSupports(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := ("func add(a: Int, b: Int) -> Int:\n    return a + b\n\ntest " +
		"\"math\":\n    expect add(40, 2) == 42\n")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"PASS math", "Tetra tests: 1/1 passed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 target model",
		"PASS x64 SysV classifier",
		"PASS x64 SysV varargs and aggregates",
		"PASS x64 source native scalar diagnostics",
		"PASS x64 pointer FFI regression smoke",
		"PASS x64 c_int FFI object smoke",
		"PASS x64 c_uint FFI object smoke",
		"PASS x64 filesystem scheduler composition smoke",
		"PASS x64 networking runtime smoke",
		"PASS x64 scheduler restriction regression smoke",
		"PASS x64 pointer atomic ABI width",
		"PASS x64 object ABI smoke",
		"PASS x64 atomic ABI object",
		"PASS x64 executable matrix smoke",
		"Tetra tests: 14/14 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandX32ABISuiteJSONReport(t *testing.T) {
	var report struct {
		Total  int    `json:"total"`
		Passed int    `json:"passed"`
		Failed int    `json:"failed"`
		Target string `json:"target"`
		Files  []struct {
			Filename string `json:"filename"`
			Total    int    `json:"total"`
			Passed   int    `json:"passed"`
			Failed   int    `json:"failed"`
		} `json:"files"`
		Results []struct {
			Name     string `json:"name"`
			Filename string `json:"filename"`
			Passed   bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--target", "x32", "--abi", "--report=json"}, 0, &report)
	if report.Total != 38 || report.Passed != 38 || report.Failed != 0 ||
		len(report.Results) != 38 {
		t.Fatalf("report = %#v", report)
	}
	if report.Target != "linux-x32" {
		t.Fatalf("report target = %q, want linux-x32", report.Target)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != "tetra:x32-abi" ||
		report.Files[0].Total != 38 ||
		report.Files[0].Passed != 38 ||
		report.Files[0].Failed != 0 {
		t.Fatalf("files = %#v", report.Files)
	}
	wantNames := []string{
		"x32 target model",
		"x32 SysV classifier",
		"x32 SysV varargs and aggregates",
		"x32 pointer FFI object smoke",
		"x32 c_int FFI object smoke",
		"x32 c_uint FFI object smoke",
		"x32 ILP32 native/libc FFI object smoke",
		"x32 ref FFI null-return diagnostics",
		"x32 function-pointer FFI diagnostics",
		"x32 source native scalar diagnostics",
		"x32 stdout executable smoke",
		"x32 stderr fd runtime smoke",
		"x32 allocator executable smoke",
		"x32 allocator failure executable smoke",
		"x32 raw memory bounds executable smoke",
		"x32 raw pointer slot executable smoke",
		"x32 raw pointer offset slot executable smoke",
		"x32 island free executable smoke",
		"x32 stdlib runtime boundary diagnostics",
		"x32 time runtime smoke",
		"x32 filesystem runtime smoke",
		"x32 filesystem scheduler composition smoke",
		"x32 single-actor self-host runtime smoke",
		"x32 single-task self-host runtime smoke",
		"x32 typed-task self-host runtime smoke",
		"x32 staged typed-task self-host runtime smoke",
		"x32 task-group self-host runtime smoke",
		"x32 typed-task-group self-host runtime smoke",
		"x32 actor-state self-host runtime smoke",
		"x32 ctx_switch object smoke",
		"x32 target runtime boundary diagnostics",
		"x32 networking runtime boundary diagnostics",
		"x32 networking lifecycle runtime smoke",
		"x32 surface/distributed runtime boundary diagnostics",
		"x32 pointer atomic ABI width",
		"x32 object ABI smoke",
		"x32 atomic ABI object",
		"x32 executable matrix smoke",
	}
	for i, want := range wantNames {
		if report.Results[i].Name != want || report.Results[i].Filename != "tetra:x32-abi" ||
			!report.Results[i].Passed {
			t.Fatalf("result[%d] = %#v", i, report.Results[i])
		}
	}
}

func TestTestCommandRunsTetraTests(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandDiscoversCapsuleSourceRoots(t *testing.T) {
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
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "src/passes.t4", "test \"project ok\":\n    expect 40 + 2 == 42\n")
	writeCLIProjectFile(t, dir, "other/fails.t4", "test \"should not run\":\n    expect 1 == 2\n")

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
	code := runCLI([]string{"test", "--target", mustHostTarget(t)}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") ||
		strings.Contains(stdout.String(), "should not run") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandExplicitProjectDirectoryUsesSourceRootsAndImports(t *testing.T) {
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
        tests
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(
		t,
		dir,
		"src/app/util.t4",
		"module app.util\nfunc answer() -> Int:\n    return 42\n",
	)
	writeCLIProjectFile(
		t,
		dir,
		"tests/util_test.t4",
		("module util_test\nimport app.util as util\ntest \"imports app " +
			"util\":\n    expect util.answer() == 42\n"),
	)
	writeCLIProjectFile(t, dir, "other/fails.t4", "test \"should not run\":\n    expect 1 == 2\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandDirectoryScanUsesNestedCapsuleSourceRoots(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	project := filepath.Join(dir, "examples", "service")
	writeCLIProjectFile(t, project, "Capsule.t4", `capsule Service:
    id "tetra://service"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
        tests
`)
	writeCLIProjectFile(
		t,
		project,
		"src/app/main.t4",
		"module app.main\nfunc main() -> Int:\n    return 0\n",
	)
	writeCLIProjectFile(
		t,
		project,
		"src/services/gateway.t4",
		"module services.gateway\nfunc status() -> Int:\n    return 42\n",
	)
	writeCLIProjectFile(
		t,
		project,
		"tests/gateway_routes.t4",
		("module gateway_routes\nimport services.gateway as " +
			"gateway\ntest \"nested capsule import\":\n    expect " +
			"gateway.status() == 42\n"),
	)

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"test", "--target", mustHostTarget(t), filepath.Join(dir, "examples")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandDirectoryScanFallsBackForNestedLegacyModulePath(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	project := filepath.Join(dir, "examples", "projects", "dogfood_cli")
	writeCLIProjectFile(t, project, "Capsule.t4", `capsule DogfoodCLI:
    id "tetra://examples/dogfood-cli"
    version "0.1.0"
    target "linux-x64"
`)
	writeCLIProjectFile(
		t,
		project,
		"src/main.t4",
		("module examples.projects.dogfood_cli.src.main\n\ntest \"legacy " +
			"module path\":\n    expect 40 + 2 == 42\n"),
	)

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"test", "--target", mustHostTarget(t), filepath.Join(dir, "examples")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandRunsMicroserviceCapsuleSourceRootExample(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	project := filepath.Join(
		"..",
		"..",
		"..",
		"examples",
		"microservices",
		"backend_capsule_source_root_service",
	)
	if _, err := os.Stat(project); err != nil {
		t.Fatalf("missing microservice capsule project %s: %v", project, err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(project, ".tetra_cache"))
		_ = os.RemoveAll(filepath.Join(project, "tetra_cache"))
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.tetra") {
		t.Fatalf("check stdout = %q", stdout.String())
	}

	out := filepath.Join(t.TempDir(), "capsule-service")
	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{"build", "--target", mustHostTarget(t), "-o", out, project},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf(
			"build exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"run", "--target", mustHostTarget(t), project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"test", "--target", mustHostTarget(t), project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "2/2 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandRunsModuleFileWithImportsAndMain(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	srcPath := filepath.Join(
		"..",
		"..",
		"..",
		"examples",
		"projects",
		"dogfood_cli",
		"src",
		"main.tetra",
	)
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing dogfood source %s: %v", srcPath, err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS cli status code") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandJSONReport(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total      int    `json:"total"`
		Passed     int    `json:"passed"`
		Failed     int    `json:"failed"`
		Target     string `json:"target"`
		DurationMS int64  `json:"duration_ms"`
		Files      []struct {
			Filename   string `json:"filename"`
			Total      int    `json:"total"`
			Passed     int    `json:"passed"`
			Failed     int    `json:"failed"`
			DurationMS int64  `json:"duration_ms"`
		} `json:"files"`
		Results []struct {
			Name       string `json:"name"`
			Passed     bool   `json:"passed"`
			DurationMS int64  `json:"duration_ms"`
		} `json:"results"`
	}
	target := mustHostTarget(t)
	runCLIJSONStdout(t, []string{"test", "--target", target, "--report=json", srcPath}, 0, &report)
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 || len(report.Results) != 1 ||
		report.Results[0].Name != "math" ||
		!report.Results[0].Passed {
		t.Fatalf("report = %#v", report)
	}
	if report.Target != target {
		t.Fatalf("report target = %q, want %s", report.Target, target)
	}
	if report.DurationMS <= 0 || report.Results[0].DurationMS <= 0 {
		t.Fatalf("durations missing: %#v", report)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != srcPath ||
		report.Files[0].Total != 1 ||
		report.Files[0].Passed != 1 ||
		report.Files[0].Failed != 0 {
		t.Fatalf("file report = %#v", report.Files)
	}
	if report.Files[0].DurationMS != report.Results[0].DurationMS ||
		report.DurationMS != report.Results[0].DurationMS {
		t.Fatalf("duration aggregation mismatch: %#v", report)
	}
}

func TestTestCommandTOONReport(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Target  string `json:"target"`
		Results []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
		} `json:"results"`
	}
	target := mustHostTarget(t)
	runCLITOONStdout(t, []string{"test", "--target", target, "--report=toon", srcPath}, 0, &report)
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 || report.Target != target ||
		len(report.Results) != 1 ||
		report.Results[0].Name != "math" ||
		!report.Results[0].Passed {
		t.Fatalf("report = %#v", report)
	}
}

func TestTestCommandTOONReportFormatAlias(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	}
	runCLITOONStdout(
		t,
		[]string{"test", "--target", mustHostTarget(t), "--format=toon", srcPath},
		0,
		&report,
	)
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestTestCommandJSONReportMultipleBlocks(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := `test "first":
    expect 1 + 1 == 2

test "second":
    expect 2 + 2 == 4
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Results []struct {
			Name         string `json:"name"`
			Index        int    `json:"index"`
			FunctionName string `json:"function_name"`
			Passed       bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(
		t,
		[]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath},
		0,
		&report,
	)
	if report.Total != 2 || report.Passed != 2 || report.Failed != 0 || len(report.Results) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Results[0].Name != "first" || report.Results[0].Index != 0 ||
		report.Results[0].FunctionName != "__tetra_test_0_first" ||
		!report.Results[0].Passed {
		t.Fatalf("first result = %#v", report.Results[0])
	}
	if report.Results[1].Name != "second" || report.Results[1].Index != 1 ||
		report.Results[1].FunctionName != "__tetra_test_1_second" ||
		!report.Results[1].Passed {
		t.Fatalf("second result = %#v", report.Results[1])
	}
}

func TestTestCommandReportsFailingExpectText(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"bad math\":\n    expect 40 + 2 == 41\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected failing test, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "FAIL bad math") || !strings.Contains(out, "exit code 1") ||
		!strings.Contains(out, "0/1 passed") {
		t.Fatalf("test stdout = %q", out)
	}
}

func TestTestCommandJSONReportIncludesFailureError(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"bad math\":\n    expect 40 + 2 == 41\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Results []struct {
			Name     string `json:"name"`
			ExitCode int    `json:"exit_code"`
			Passed   bool   `json:"passed"`
			Error    string `json:"error"`
		} `json:"results"`
	}
	runCLIJSONStdout(
		t,
		[]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath},
		1,
		&report,
	)
	if report.Total != 1 || report.Passed != 0 || report.Failed != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}
	result := report.Results[0]
	if result.Name != "bad math" || result.Passed || result.ExitCode != 1 ||
		result.Error != "exit code 1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestTestCommandJSONReportUsesEmptyArraysWhenNoTestsExist(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int               `json:"total"`
		Passed  int               `json:"passed"`
		Failed  int               `json:"failed"`
		Files   []json.RawMessage `json:"files"`
		Results []json.RawMessage `json:"results"`
	}
	rawReport := runCLIJSONStdout(
		t,
		[]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath},
		0,
		&report,
	)
	if report.Total != 0 || report.Passed != 0 || report.Failed != 0 {
		t.Fatalf("report counts = %#v", report)
	}
	if report.Files == nil || len(report.Files) != 0 || report.Results == nil ||
		len(report.Results) != 0 {
		t.Fatalf("empty arrays should be present, report = %#v\n%s", report, rawReport)
	}
}

// ---- test_helpers_test.go ----

func stubLookPath(fn func(string) (string, error)) func() {
	old := commandLookPath
	oldWebRunnerProbe := webRunnerProbe
	commandLookPath = fn
	webRunnerProbe = func(string) error { return nil }
	return func() {
		commandLookPath = old
		webRunnerProbe = oldWebRunnerProbe
	}
}

func stubLinuxX32HostSupport(supported bool) func() {
	old := linuxX32HostSupport
	linuxX32HostSupport = func() bool { return supported }
	return func() {
		linuxX32HostSupport = old
	}
}

func expectedLinuxX32HostUnsupportedReason(t *testing.T) string {
	t.Helper()
	tgt, err := ctarget.Parse("linux-x32")
	if err != nil {
		t.Fatalf("parse linux-x32 target: %v", err)
	}
	return buildOnlyNativeRunUnsupportedReason(tgt)
}

func requireLinuxX32HostUnsupportedReason(t *testing.T, got string) {
	t.Helper()
	want := expectedLinuxX32HostUnsupportedReason(t)
	if got != want {
		t.Fatalf("linux-x32 unsupported reason = %q, want %q", got, want)
	}
}

func stubNativeExec(fn func(string, io.Writer, io.Writer) int) func() {
	old := execNativeProgram
	execNativeProgram = fn
	return func() {
		execNativeProgram = old
	}
}

func stubNativeSurfaceExec(
	fn func(string, surfaceHostRunOptions, io.Writer, io.Writer) int,
) func() {
	old := execNativeSurfaceProgram
	execNativeSurfaceProgram = fn
	return func() {
		execNativeSurfaceProgram = old
	}
}

type cliJSONDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Hint     string `json:"hint"`
	Severity string `json:"severity"`
}

func runCLIJSONDiagnostic(t *testing.T, args []string, wantExit int) cliJSONDiagnostic {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf(
			"expected empty stdout for JSON diagnostic, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	var diag cliJSONDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	return diag
}

func runCLITOONDiagnostic(t *testing.T, args []string, wantExit int) cliJSONDiagnostic {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf(
			"expected empty stdout for TOON diagnostic, stdout=%q stderr=%q",
			stdout.String(),
			stderr.String(),
		)
	}
	jsonData, err := toon.ConvertTOONToJSON(stderr.Bytes(), toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("toon diagnostic: %v\n%s", err, stderr.String())
	}
	var diag cliJSONDiagnostic
	if err := json.Unmarshal(jsonData, &diag); err != nil {
		t.Fatalf("toon->json diagnostic: %v\nTOON:\n%s\nJSON:\n%s", err, stderr.String(), jsonData)
	}
	return diag
}

func runCLIJSONStdout(t *testing.T, args []string, wantExit int, out any) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
		t.Fatalf("json stdout: %v\n%s", err, stdout.String())
	}
	return stdout.String()
}

func runCLITOONStdout(t *testing.T, args []string, wantExit int, out any) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	jsonData, err := toon.ConvertTOONToJSON(stdout.Bytes(), toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("toon stdout: %v\n%s", err, stdout.String())
	}
	if err := json.Unmarshal(jsonData, out); err != nil {
		t.Fatalf("toon->json stdout: %v\nTOON:\n%s\nJSON:\n%s", err, stdout.String(), jsonData)
	}
	return stdout.String()
}

func assertCLIJSONOwnershipDiagnostic(t *testing.T, srcPath string, wantText string) {
	t.Helper()
	assertCLIJSONOwnershipDiagnosticForPath(t, srcPath, srcPath, wantText)
}

func assertCLIJSONOwnershipDiagnosticForPath(
	t *testing.T,
	checkPath string,
	diagPath string,
	wantText string,
) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(
		t,
		checkPath,
		diagPath,
		compiler.DiagnosticCodeSafetyOwnership,
		wantText,
	)
}

func assertCLIJSONLifetimeDiagnostic(t *testing.T, srcPath string, wantText string) {
	t.Helper()
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, srcPath, wantText)
}

func assertCLIJSONLifetimeDiagnosticForPath(
	t *testing.T,
	checkPath string,
	diagPath string,
	wantText string,
) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(
		t,
		checkPath,
		diagPath,
		compiler.DiagnosticCodeSafetyLifetime,
		wantText,
	)
}

func assertCLIJSONSemanticDiagnostic(
	t *testing.T,
	checkPath string,
	diagPath string,
	wantText string,
) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(
		t,
		checkPath,
		diagPath,
		compiler.DiagnosticCodeSemantic,
		wantText,
	)
}

func assertCLIJSONDiagnosticForPath(
	t *testing.T,
	checkPath string,
	diagPath string,
	wantCode string,
	wantText string,
) {
	t.Helper()
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", checkPath}, 1)
	if diag.Code != wantCode || filepath.Clean(diag.File) != filepath.Clean(diagPath) ||
		diag.Line <= 0 ||
		diag.Column <= 0 ||
		diag.Severity != "error" ||
		!strings.Contains(diag.Message, wantText) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func mustHostTarget(t *testing.T) string {
	t.Helper()
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	return target
}

func writeCLIProjectFile(t *testing.T, root string, rel string, src string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func nonHostTarget(t *testing.T) string {
	t.Helper()
	host := mustHostTarget(t)
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		if target != host {
			return target
		}
	}
	t.Fatal("no non-host target found")
	return ""
}

func writeArtifactBuildFixture(t *testing.T, dir string, target string) string {
	t.Helper()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", fmt.Sprintf(`capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    targets:
        %s
`, target))
	writeCLIProjectFile(
		t,
		dir,
		"Math/src/math/core.t4",
		"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://math 0.1.0 ../Math
`, target))
	writeCLIProjectFile(
		t,
		dir,
		"App/src/app/main.t4",
		"module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	)
	return filepath.Join(dir, "App")
}

func writeWorkspaceMainProject(
	t *testing.T,
	root string,
	name string,
	id string,
	target string,
	exitCode int,
) {
	t.Helper()
	writeCLIProjectFile(
		t,
		root,
		filepath.ToSlash(filepath.Join(name, "Capsule.t4")),
		fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target),
	)
	writeCLIProjectFile(
		t,
		root,
		filepath.ToSlash(filepath.Join(name, "src/main.t4")),
		fmt.Sprintf("func main() -> Int:\n    return %d\n", exitCode),
	)
}

func writeWorkspaceTestProject(
	t *testing.T,
	root string,
	name string,
	id string,
	target string,
	testName string,
	condition string,
) {
	t.Helper()
	writeCLIProjectFile(
		t,
		root,
		filepath.ToSlash(filepath.Join(name, "Capsule.t4")),
		fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target),
	)
	writeCLIProjectFile(
		t,
		root,
		filepath.ToSlash(filepath.Join(name, "src/main.t4")),
		"func main() -> Int:\n    return 0\n",
	)
	writeCLIProjectFile(
		t,
		root,
		filepath.ToSlash(filepath.Join(name, "src/tests.t4")),
		fmt.Sprintf("test %q:\n    expect %s\n", testName, condition),
	)
}

// ---- test_structure_test.go ----

func TestCLITestsAreSplitByCommandSurface(t *testing.T) {
	expected := map[string][]string{
		"cli_contract_test.go": {
			"TestVersionCommand",
			"TestCLIContractDocumentedCommandsHaveHelpAndInvalidArgBehavior",
		},
		"metadata_test.go": {
			"TestTargetsCommandText",
			"TestTargetsCommandJSON",
			"TestFeaturesCommandJSON",
			"TestFormatsCommandListsOfficialT4Family",
		},
		"doctor_test.go": {
			"TestDoctorCommandJSON",
			"TestTargetMetadataCheck",
			"TestDoctorCommandProjectJSON",
			"TestDoctorReportFilesystemProbesFailInIncompleteRepo",
		},
		"clean_test.go": {
			"TestCleanCommandRemovesCacheDirectories",
			"TestCleanCommandTargetRemovesOnlyRequestedTargetCache",
		},
		"lsp_test.go": {
			"TestLSPCommandSmoke",
			"TestLSPStdioInitializeAndDidOpen",
			"TestLSPStdioTranscriptFixtureCoversEditingRequests",
			"TestLSPStdioDidCloseClearsDiagnostics",
		},
		"new_app_test.go": {
			"TestNewAppScaffoldCreatesRunnableT4Project",
			"TestNewAppLockOptionWritesTetraLock",
			"TestNewAppRejectsExistingDirectory",
			"TestProjectInfoCommandJSON",
		},
		"fmt_test.go": {
			"TestFmtCommandCheckAndStdout",
			"TestCollectTetraFilesIncludesT4AndLegacyTetra",
			"TestCollectTetraFilesSkipsCapsuleManifest",
			"TestFormatCommandWriteIsIdempotentAndPreservesStandaloneComments",
			"TestFormatCommandJSONDiagnosticsForInlineComment",
			"TestFmtCommandJSONDiagnosticsForInvalidModeCombination",
			"TestFmtCommandJSONDiagnosticsForMissingPath",
			"TestFmtCommandJSONDiagnosticsForMultipleStdoutFiles",
			"TestFmtCheckJSONDiagnosticsForUnformattedFile",
			"TestFormatCommandCheckJSONDiagnosticsIncludesFirstDiffPosition",
		},
		"smoke_test.go": {
			"TestSmokeCommandWritesReport",
			"TestSmokeCommandListsCasesAsJSON",
			"TestSmokeCommandKeepsInvalidDoubleFreeOutOfDebugList",
			"TestSmokeCommandListsWASMRuntimeTargets",
			"TestSmokeCommandBuildsWASMTargetWithoutRun",
			"TestSmokeCommandWASMReportUsesDurableArtifacts",
			"TestSmokeCommandRunsWASIWithNodeFallbackRunner",
			"TestSmokeCommandListWASIRunSupportedTracksRunnerAvailability",
			"TestSmokeCommandWASMTargetGroupsIncludeDogfoodWebUI",
			"TestSmokeCommandRejectsFormatWithoutList",
		},
		"project_test.go": {
			"TestProjectSyncWritesLockForProjectWithoutDependencies",
			"TestProjectSyncCheckReportsMissingLockWithoutWriting",
			"TestProjectSyncRejectsTargetAndAllTargetsTogether",
			"TestProjectSyncGeneratesDependencyArtifactsAndLock",
			"TestProjectSyncWritesLockForBuildOnlyTargetWithoutNativeArtifacts",
			"TestProjectDepsAddPathDiscoversMetadataAndAppendsDeps",
			"TestProjectDepsAddRejectsDuplicate",
			"TestProjectDepsAddAllowsMetadataOverride",
			"TestProjectDepsListJSONReportsResolvedPath",
			"TestProjectDepsRemoveByID",
			"TestProjectDepsRemoveRejectsAmbiguousID",
			"TestProjectDepsCheckPassesForValidDependency",
			"TestProjectDepsCheckFailsForMissingPathVersionMismatchAndCycle",
		},
		"workspace_test.go": {
			"TestWorkspaceInitAddListAndRemove",
			"TestWorkspaceCheckGraphAndSync",
			"TestWorkspaceCheckFailures",
			"TestWorkspaceBuildWritesPerMemberOutputsAndJSONSummary",
			"TestWorkspaceBuildSkipsDependentAfterFailedDependency",
			"TestWorkspaceTestFailFastJSONSummary",
			"TestWorkspaceRunMemberAndUnknownMember",
		},
		"doc_test.go": {
			"TestDocCommandWritesAPIDocsToStdout",
			"TestDocCommandDiscoversCapsuleProjectSources",
			"TestDocCommandWritesAPIDocsToFile",
			"TestDocCommandGeneratedOutputPassesAPIValidator",
			"TestDocCommandJSONDiagnostics",
		},
		"interface_test.go": {
			"TestInterfaceCommandWritesT4IFile",
			"TestInterfaceCommandCheckReportsStalePublicAPI",
			"TestCheckCommandInterfaceOnlyDoesNotRequireMain",
			"TestBuildCommandInterfaceOnlyDoesNotRequireMain",
		},
		"test_command_test.go": {
			"TestTestCommandJSONDiagnosticsForWASMRuntimeUnsupported",
			"TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported",
			"TestTestCommandJSONDiagnosticsForHostTargetMismatch",
			"TestTestCommandJSONDiagnosticsForUnsupportedReportFormat",
			"TestTestCommandRunsAllTargetsBrutalSuite",
			"TestTestCommandAllTargetsBrutalJSONUsesTargetSpecificFiles",
			"TestTestCommandJSONDiagnosticsForTargetSpecificSuiteUnsupported",
			"TestTestCommandRunsTetraTests",
			"TestTestCommandDiscoversCapsuleSourceRoots",
			"TestTestCommandExplicitProjectDirectoryUsesSourceRootsAndImports",
			"TestTestCommandRunsMicroserviceCapsuleSourceRootExample",
			"TestTestCommandRunsModuleFileWithImportsAndMain",
			"TestTestCommandJSONReport",
			"TestTestCommandJSONReportMultipleBlocks",
			"TestTestCommandReportsFailingExpectText",
			"TestTestCommandJSONReportIncludesFailureError",
			"TestTestCommandJSONReportUsesEmptyArraysWhenNoTestsExist",
		},
		"run_test.go": {
			"TestRunCommandJSONDiagnosticsForHostTargetMismatch",
			"TestRunCommandJSONDiagnosticsForWASMWebRuntimeUnsupported",
			"TestExecWebProgramWithBrowserRunnerParsesBrowserExitResult",
			"TestRunCommandPropagatesProgramExitCode",
			"TestRunCommandPropagatesLinuxX86NoRuntimeExitCode",
			"TestRunCommandPropagatesLinuxX86FunctionArgumentExitCode",
			"TestRunCommandPropagatesLinuxX86GlobalExitCode",
			"TestRunCommandPropagatesLinuxX86DirectCallbackExitCode",
			"TestRunCommandPropagatesLinuxX86MakeI32SliceExitCode",
			"TestRunCommandPropagatesLinuxX86AllocBytesZeroExitCode",
			"TestRunCommandPropagatesLinuxX86RawStoreLoadExitCode",
			"TestRunCommandPropagatesLinuxX86RawPtrAddU8ExitCode",
			"TestRunCommandPropagatesLinuxX86RawPtrAddUpperBoundExitCode",
			"TestRunCommandPropagatesLinuxX86PrintStringStdout",
			"TestRunCommandPropagatesLinuxX86PrintSliceStdout",
			"TestRunCommandPropagatesLinuxX86ScopedIslandExitCode",
			"TestRunCommandPropagatesLinuxX86ScopedIslandDebugExitCode",
			"TestRunCommandPropagatesLinuxX86ScopedIslandOverflowExitCode",
			"TestRunCommandPropagatesLinuxX86MMIOExitCode",
			"TestRunCommandWithoutOutputDoesNotLeaveDefaultBinary",
		},
		"build_test.go": {
			"TestDefaultOutputUsesTargetExtensionAndEmitMode",
			"TestBuildCommandUsesDefaultInput",
			"TestBuildCommandDiscoversCapsuleT4ProjectEntry",
			"TestBuildAndRunCommandsAcceptExplicitProjectDirectory",
			"TestBuildCheckRunCommandsAcceptExplicitProjectSourceFile",
			"TestBuildCommandUsesCapsuleInterfaceAndObjectArtifacts",
			"TestBuildCommandArtifactsAutoRepairsStaleObject",
			"TestBuildCommandWASMProjectLockDoesNotRequireNativeArtifacts",
			"TestBuildCommandUsesCapsuleDefaultTarget",
			"TestBuildCommandAllTargetsBuildsCapsuleTargets",
			"TestBuildCommandJSONDiagnostics",
			"TestBuildCommandJSONDiagnosticsForOptionValidation",
			"TestBuildCommandWASMTargetWritesWasmModule",
			"TestBuildCommandUIWritesBackendSidecars",
			"TestBuildCommandWASMWebPackageOutputIsDeterministic",
			"TestBuildCommandRejectsUnsupportedDiagnosticsMode",
			"TestBuildCommandRejectsInvalidTarget",
			"TestBuildCommandJSONDiagnosticsForInvalidTarget",
			"TestBuildCommandJSONDiagnosticsForTooManyInputs",
		},
		"check_test.go": {
			"TestCheckCommandUsesDefaultMainT4",
			"TestCheckCommandDiscoversCapsuleT4ProjectEntryAndSourceRoots",
			"TestCheckCommandExplicitProjectDirectoryUsesCapsuleEntry",
			"TestCheckCommandResolvesLocalCapsuleDependencyImport",
			"TestCheckCommandValidatesPresentTetraLockAgainstCapsuleGraph",
			"TestCheckCommandSucceedsWithoutOutputFile",
			"TestTargetAwareCommandsRejectInvalidTargetConsistently",
			"TestCheckCommandReportsMissingDefaultMain",
		},
		"check_diagnostics_misc_test.go": {
			"TestCheckCommandJSONDiagnosticsForSemanticError",
			"TestCheckCommandJSONDiagnosticsForGenericBorrowReturnCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleGenericBorrowReturnCodes",
			"TestCheckCommandJSONDiagnosticsForProtocolImplOwnershipMismatchCodes",
			"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowSliceAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForScopedIslandOptionalRegionEscapeCode",
		},
		"eco_test.go": {
			"TestEcoVerifySingleCapsuleExpandsPathDependenciesIntoTetraLock",
			"TestEcoArtifactsBuildGeneratesDependencyArtifactsLockAndBuildsProject",
			"TestEcoArtifactsCheckDetectsStaleInterfaceAndSuggestsRepair",
			"TestEcoArtifactsBuildCheckDryRunDoesNotWriteArtifacts",
			"TestEcoArtifactsBuildAllTargetsSkipsWASMObjectTargets",
			"TestEcoVerifyPackAndUnpack",
			"TestEcoVerifyHelpExitsSuccessfully",
			"TestEcoTopLevelHelpMentionsVerifyLock",
			"TestEcoPackUnpackVaultHelpExitsSuccessfully",
			"TestEcoPackProjectBundle",
			"TestEcoPackProjectBundleUsesT4CapsuleAndSource",
			"TestEcoVerifyStructuredCapsuleT4WritesPolicyLock",
			"TestEcoVerifyDependencyGraphAndLock",
			"TestEcoVerifyRejectsPermissionEscalationFromDependency",
			"TestEcoVerifyRejectsDuplicateManifestIDField",
			"TestEcoVerifyReportsMissingDependency",
			"TestEcoVerifyReportsDuplicateIDAndTargetMismatch",
			"TestEcoVaultAddListAndVerify",
			"TestEcoVaultVerifyDetectsCorruptObject",
		},
		"check_diagnostics_resource_actor_test.go": {
			"TestCheckCommandJSONDiagnosticsForResourceUseAfterFreeCode",
			"TestCheckCommandJSONDiagnosticsForResourceStructFieldAliasUseAfterFreeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleResourceStructFieldAliasUseAfterFreeCode",
			"TestCheckCommandJSONDiagnosticsForResourceEnumPayloadAliasUseAfterFreeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleResourceEnumPayloadAliasUseAfterFreeCode",
			"TestCheckCommandJSONDiagnosticsForResourceOptionalPayloadFreeCode",
			"TestCheckCommandJSONDiagnosticsForResourceOptionalWrapperAliasUseAfterFreeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleResourceOptionalWrapperAliasUseAfterFreeCodes",
			"TestCheckCommandJSONDiagnosticsForResourceDoubleJoinCode",
			"TestCheckCommandJSONDiagnosticsForTaskGroupUseAfterCloseCode",
			"TestCheckCommandJSONDiagnosticsForResourceAmbiguousProvenanceCode",
			"TestCheckCommandJSONDiagnosticsForIslandTransferNonLocalPayloadCode",
			"TestCheckCommandJSONDiagnosticsForActorUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForActorBranchConsumeReuseCode",
			"TestCheckCommandJSONDiagnosticsForActorMatchLoopConsumeReuseCodes",
			"TestCheckCommandJSONDiagnosticsForTaskUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForActorStructFieldAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleActorStructFieldAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForGenericActorStructFieldAliasUseAfterTransferCodes",
			"TestCheckCommandJSONDiagnosticsForGenericResourceAliasFinalizationCodes",
			"TestCheckCommandJSONDiagnosticsForTransitiveResourceAliasFinalizationCodes",
			"TestCheckCommandJSONDiagnosticsForEnumConstructorReturnResourceAliasCodes",
		},
		"check_diagnostics_actor_transitive_test.go": {
			"TestCheckCommandJSONDiagnosticsForTransitiveActorAliasUseAfterTransferCodes",
			"TestCheckCommandJSONDiagnosticsForTaskGroupCancelReturnProvenanceCodes",
			"TestCheckCommandJSONDiagnosticsForTaskHandleGroupOptionalPayloadJoinCloseAliasCodes",
			"TestCheckCommandJSONDiagnosticsForActorEnumPayloadAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleActorEnumPayloadAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForTaskStructFieldAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleTaskStructFieldAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForTaskEnumPayloadAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleTaskEnumPayloadAliasUseAfterTransferCode",
			"TestCheckCommandJSONDiagnosticsForPrivacyConsentSafetyCode",
			"TestCheckCommandJSONDiagnosticsForRecursiveSecretSignaturePrivacyCode",
			"TestCheckCommandJSONDiagnosticsForTooManyInputs",
			"TestCheckCommandRejectsLocalCapsuleDependencyCycle",
		},
		"check_diagnostics_callable_test.go": {
			"TestCheckCommandJSONDiagnosticsForCallableMutableCaptureGlobalEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCapturedCallableGlobalStorageCode",
			"TestCheckCommandJSONDiagnosticsForFunctionTypedParameterGlobalStorageCode",
			"TestCheckCommandJSONDiagnosticsForFunctionValueUnsupportedEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCapturingClosureRawPointerEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCallableResourceCaptureEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCallableMutableCaptureHeapEscapeCode",
			"TestCheckCommandJSONDiagnosticsForGenericClosureCaptureCode",
			"TestCheckCommandJSONDiagnosticsForGenericCallbackClosureCaptureCode",
			"TestCheckCommandJSONDiagnosticsForFunctionTypedStorageUnsupportedCaptureCode",
			"TestCheckCommandJSONDiagnosticsForFunctionTypedReturnUnsupportedCaptureCode",
			"TestCheckCommandJSONDiagnosticsForCapturedClosureExplicitTypeArgsCode",
			"TestCheckCommandJSONDiagnosticsForFunctionTypedExplicitTypeArgsCode",
			"TestCheckCommandJSONDiagnosticsForUnsupportedFunctionValueCallCode",
			"TestCheckCommandJSONDiagnosticsForGenericClosurePointerEscapeCode",
			"TestCheckCommandJSONDiagnosticsForGenericClosureDirectCallRequirementCode",
		},
		"check_diagnostics_ownership_basic_test.go": {
			"TestCheckCommandJSONDiagnosticsForOwnershipUseAfterConsumeCode",
			"TestCheckCommandJSONDiagnosticsForOwnershipPartialStructConsumeCode",
			"TestCheckCommandJSONDiagnosticsForOwnershipPartialStructCopyAfterConsumeCode",
			"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConsumeCode",
			"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumCopyAfterConsumeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModulePartialCopyAfterConsumeCodes",
			"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConstructorAfterConsumeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModulePartialEnumConstructorAfterConsumeCodes",
			"TestCheckCommandJSONDiagnosticsForOwnershipOptionalPayloadConsumeCode",
		},
		"check_diagnostics_ownership_borrow_slice_call_test.go": {
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateGenericCallEscapeCodes",
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"SliceAggregateGenericCallEscapeCodes"),
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrGenericCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrGenericCallEscapeCodes",
		},
		"check_diagnostics_ownership_borrow_function_typed_test.go": {
			("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
				"ateFunctionTypedParameterCallEscapeCodes"),
			("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
				"ateFunctionTypedStructFieldCallEscapeCodes"),
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"SliceAggregateFunctionTypedStructFieldCallEscapeCodes"),
			("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
				"ateFunctionTypedEnumPayloadCallEscapeCodes"),
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"SliceAggregateFunctionTypedEnumPayloadCallEscapeCodes"),
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes",
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"OptionalPtrFunctionTypedCallbackCodes"),
		},
		"check_diagnostics_ownership_borrow_ptr_aggregate_test.go": {
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
		},
		"check_diagnostics_ownership_borrow_payload_test.go": {
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrEnumPayloadCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrEnumPayloadCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode",
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"PtrOptionalPayloadOwnedCallEscapeCode"),
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes",
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"PtrOptionalPayloadConsumeInoutCallEscapeCodes"),
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes",
			("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
				"SliceOptionalPayloadBindingEscapeCodes"),
			"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalAssignmentConsumeCode",
		},
		"check_diagnostics_lifetime_borrow_test.go": {
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowFixedArrayAliasReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowStringAliasReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowOptionalAssignmentEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentCallEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceOptionalAssignmentEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceStructEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceStructEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceStructEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceStructEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceEnumPayloadEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceEnumPayloadEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceEnumEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceEnumEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForSafeViewBorrowedOwnedReturnCode",
			"TestCheckCommandJSONDiagnosticsForSafeViewActorBoundaryCode",
			"TestCheckCommandJSONDiagnosticsForSafeViewTaskBoundaryCode",
			"TestCheckCommandJSONDiagnosticsForSafeViewAggregateHiddenBorrowCode",
		},
		"check_diagnostics_lifetime_global_assignment_test.go": {
			"TestCheckCommandJSONDiagnosticsForBorrowedPtrOptionalGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForBorrowedStringGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrOptionalGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForBorrowedPtrAggregateOptionalGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrAggregateOptionalGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForBorrowedSliceOptionalPayloadGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForBorrowedSliceGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceOptionalPayloadGlobalAssignmentCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceGlobalAssignmentCode",
		},
		"check_diagnostics_lifetime_ptr_test.go": {
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumAliasReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumAliasReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateReturnEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateReturnEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes",
			("TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowP" +
				"trOptionalAssignmentGlobalEscapeCodes"),
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateGlobalEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateGlobalEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadReturnEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadGlobalEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumGlobalEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadGlobalEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumGlobalEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadInoutEscapeCode",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadInoutEscapeCode",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes",
			"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes",
		},
	}

	for path, symbols := range expected {
		readPath := path
		if _, err := os.Stat(readPath); os.IsNotExist(err) {
			readPath = "tetra_suite_test.go"
		} else if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		raw, err := os.ReadFile(readPath)
		if err != nil {
			t.Fatalf("read %s: %v", readPath, err)
		}
		text := string(raw)
		for _, symbol := range symbols {
			if !strings.Contains(text, "func "+symbol+"(") {
				t.Fatalf("%s must contain %s from %s", readPath, symbol, path)
			}
		}
	}

	if _, err := os.Stat("main_test.go"); err == nil {
		t.Fatalf(
			"main_test.go should not exist; shared CLI test helpers belong in test_helpers_test.go",
		)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat main_test.go: %v", err)
	}

	helperRaw, err := os.ReadFile("tetra_suite_test.go")
	if err != nil {
		t.Fatalf("read tetra_suite_test.go: %v", err)
	}
	helperText := string(helperRaw)
	for _, symbol := range []string{
		"TestVersionCommand",
		"TestCLIContractDocumentedCommandsHaveHelpAndInvalidArgBehavior",
		"TestTargetsCommandText",
		"TestTargetsCommandJSON",
		"TestFeaturesCommandJSON",
		"TestFormatsCommandListsOfficialT4Family",
		"TestDoctorCommandJSON",
		"TestTargetMetadataCheck",
		"TestDoctorCommandProjectJSON",
		"TestDoctorReportFilesystemProbesFailInIncompleteRepo",
		"TestCleanCommandRemovesCacheDirectories",
		"TestCleanCommandTargetRemovesOnlyRequestedTargetCache",
		"TestLSPCommandSmoke",
		"TestLSPStdioInitializeAndDidOpen",
		"TestLSPStdioTranscriptFixtureCoversEditingRequests",
		"TestLSPStdioDidCloseClearsDiagnostics",
		"TestNewAppScaffoldCreatesRunnableT4Project",
		"TestNewAppLockOptionWritesTetraLock",
		"TestNewAppRejectsExistingDirectory",
		"TestProjectInfoCommandJSON",
		"TestFmtCommandCheckAndStdout",
		"TestCollectTetraFilesIncludesT4AndLegacyTetra",
		"TestCollectTetraFilesSkipsCapsuleManifest",
		"TestFormatCommandWriteIsIdempotentAndPreservesStandaloneComments",
		"TestFormatCommandJSONDiagnosticsForInlineComment",
		"TestFmtCommandJSONDiagnosticsForInvalidModeCombination",
		"TestFmtCommandJSONDiagnosticsForMissingPath",
		"TestFmtCommandJSONDiagnosticsForMultipleStdoutFiles",
		"TestFmtCheckJSONDiagnosticsForUnformattedFile",
		"TestFormatCommandCheckJSONDiagnosticsIncludesFirstDiffPosition",
		"TestSmokeCommandWritesReport",
		"TestSmokeCommandListsCasesAsJSON",
		"TestSmokeCommandKeepsInvalidDoubleFreeOutOfDebugList",
		"TestSmokeCommandListsWASMRuntimeTargets",
		"TestSmokeCommandBuildsWASMTargetWithoutRun",
		"TestSmokeCommandWASMReportUsesDurableArtifacts",
		"TestSmokeCommandRunsWASIWithNodeFallbackRunner",
		"TestSmokeCommandListWASIRunSupportedTracksRunnerAvailability",
		"TestSmokeCommandWASMTargetGroupsIncludeDogfoodWebUI",
		"TestSmokeCommandRejectsFormatWithoutList",
		"TestProjectSyncWritesLockForProjectWithoutDependencies",
		"TestProjectSyncCheckReportsMissingLockWithoutWriting",
		"TestProjectSyncRejectsTargetAndAllTargetsTogether",
		"TestProjectSyncGeneratesDependencyArtifactsAndLock",
		"TestProjectSyncWritesLockForBuildOnlyTargetWithoutNativeArtifacts",
		"TestProjectDepsAddPathDiscoversMetadataAndAppendsDeps",
		"TestProjectDepsAddRejectsDuplicate",
		"TestProjectDepsAddAllowsMetadataOverride",
		"TestProjectDepsListJSONReportsResolvedPath",
		"TestProjectDepsRemoveByID",
		"TestProjectDepsRemoveRejectsAmbiguousID",
		"TestProjectDepsCheckPassesForValidDependency",
		"TestProjectDepsCheckFailsForMissingPathVersionMismatchAndCycle",
		"TestWorkspaceInitAddListAndRemove",
		"TestWorkspaceCheckGraphAndSync",
		"TestWorkspaceCheckFailures",
		"TestWorkspaceBuildWritesPerMemberOutputsAndJSONSummary",
		"TestWorkspaceBuildSkipsDependentAfterFailedDependency",
		"TestWorkspaceTestFailFastJSONSummary",
		"TestWorkspaceRunMemberAndUnknownMember",
		"TestDocCommandWritesAPIDocsToStdout",
		"TestDocCommandDiscoversCapsuleProjectSources",
		"TestDocCommandWritesAPIDocsToFile",
		"TestDocCommandGeneratedOutputPassesAPIValidator",
		"TestDocCommandJSONDiagnostics",
		"TestInterfaceCommandWritesT4IFile",
		"TestInterfaceCommandCheckReportsStalePublicAPI",
		"TestCheckCommandInterfaceOnlyDoesNotRequireMain",
		"TestBuildCommandInterfaceOnlyDoesNotRequireMain",
		"TestTestCommandJSONDiagnosticsForWASMRuntimeUnsupported",
		"TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported",
		"TestTestCommandJSONDiagnosticsForHostTargetMismatch",
		"TestTestCommandJSONDiagnosticsForUnsupportedReportFormat",
		"TestTestCommandRunsAllTargetsBrutalSuite",
		"TestTestCommandAllTargetsBrutalJSONUsesTargetSpecificFiles",
		"TestTestCommandJSONDiagnosticsForTargetSpecificSuiteUnsupported",
		"TestTestCommandRunsTetraTests",
		"TestTestCommandDiscoversCapsuleSourceRoots",
		"TestTestCommandExplicitProjectDirectoryUsesSourceRootsAndImports",
		"TestTestCommandRunsMicroserviceCapsuleSourceRootExample",
		"TestTestCommandRunsModuleFileWithImportsAndMain",
		"TestTestCommandJSONReport",
		"TestTestCommandJSONReportMultipleBlocks",
		"TestTestCommandReportsFailingExpectText",
		"TestTestCommandJSONReportIncludesFailureError",
		"TestTestCommandJSONReportUsesEmptyArraysWhenNoTestsExist",
		"TestRunCommandJSONDiagnosticsForHostTargetMismatch",
		"TestRunCommandJSONDiagnosticsForWASMWebRuntimeUnsupported",
		"TestExecWebProgramWithBrowserRunnerParsesBrowserExitResult",
		"TestRunCommandPropagatesProgramExitCode",
		"TestRunCommandPropagatesLinuxX86NoRuntimeExitCode",
		"TestRunCommandPropagatesLinuxX86FunctionArgumentExitCode",
		"TestRunCommandPropagatesLinuxX86GlobalExitCode",
		"TestRunCommandPropagatesLinuxX86DirectCallbackExitCode",
		"TestRunCommandPropagatesLinuxX86MakeI32SliceExitCode",
		"TestRunCommandPropagatesLinuxX86AllocBytesZeroExitCode",
		"TestRunCommandPropagatesLinuxX86RawStoreLoadExitCode",
		"TestRunCommandPropagatesLinuxX86RawPtrAddU8ExitCode",
		"TestRunCommandPropagatesLinuxX86RawPtrAddUpperBoundExitCode",
		"TestRunCommandPropagatesLinuxX86PrintStringStdout",
		"TestRunCommandPropagatesLinuxX86PrintSliceStdout",
		"TestRunCommandPropagatesLinuxX86ScopedIslandExitCode",
		"TestRunCommandPropagatesLinuxX86ScopedIslandDebugExitCode",
		"TestRunCommandPropagatesLinuxX86ScopedIslandOverflowExitCode",
		"TestRunCommandPropagatesLinuxX86MMIOExitCode",
		"TestRunCommandWithoutOutputDoesNotLeaveDefaultBinary",
		"TestDefaultOutputUsesTargetExtensionAndEmitMode",
		"TestBuildCommandUsesDefaultInput",
		"TestBuildCommandDiscoversCapsuleT4ProjectEntry",
		"TestBuildAndRunCommandsAcceptExplicitProjectDirectory",
		"TestBuildCheckRunCommandsAcceptExplicitProjectSourceFile",
		"TestBuildCommandUsesCapsuleInterfaceAndObjectArtifacts",
		"TestBuildCommandArtifactsAutoRepairsStaleObject",
		"TestBuildCommandWASMProjectLockDoesNotRequireNativeArtifacts",
		"TestBuildCommandUsesCapsuleDefaultTarget",
		"TestBuildCommandAllTargetsBuildsCapsuleTargets",
		"TestBuildCommandJSONDiagnostics",
		"TestBuildCommandJSONDiagnosticsForOptionValidation",
		"TestBuildCommandWASMTargetWritesWasmModule",
		"TestBuildCommandUIWritesBackendSidecars",
		"TestBuildCommandWASMWebPackageOutputIsDeterministic",
		"TestBuildCommandRejectsUnsupportedDiagnosticsMode",
		"TestBuildCommandRejectsInvalidTarget",
		"TestBuildCommandJSONDiagnosticsForInvalidTarget",
		"TestBuildCommandJSONDiagnosticsForTooManyInputs",
		"TestCheckCommandUsesDefaultMainT4",
		"TestCheckCommandDiscoversCapsuleT4ProjectEntryAndSourceRoots",
		"TestCheckCommandExplicitProjectDirectoryUsesCapsuleEntry",
		"TestCheckCommandResolvesLocalCapsuleDependencyImport",
		"TestCheckCommandValidatesPresentTetraLockAgainstCapsuleGraph",
		"TestCheckCommandSucceedsWithoutOutputFile",
		"TestTargetAwareCommandsRejectInvalidTargetConsistently",
		"TestCheckCommandReportsMissingDefaultMain",
		"TestCheckCommandJSONDiagnosticsForSemanticError",
		"TestCheckCommandJSONDiagnosticsForGenericBorrowReturnCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleGenericBorrowReturnCodes",
		"TestCheckCommandJSONDiagnosticsForProtocolImplOwnershipMismatchCodes",
		"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowSliceAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForScopedIslandOptionalRegionEscapeCode",
		"TestCheckCommandJSONDiagnosticsForResourceUseAfterFreeCode",
		"TestCheckCommandJSONDiagnosticsForResourceStructFieldAliasUseAfterFreeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleResourceStructFieldAliasUseAfterFreeCode",
		"TestCheckCommandJSONDiagnosticsForResourceEnumPayloadAliasUseAfterFreeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleResourceEnumPayloadAliasUseAfterFreeCode",
		"TestCheckCommandJSONDiagnosticsForResourceOptionalPayloadFreeCode",
		"TestCheckCommandJSONDiagnosticsForResourceOptionalWrapperAliasUseAfterFreeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleResourceOptionalWrapperAliasUseAfterFreeCodes",
		"TestCheckCommandJSONDiagnosticsForResourceDoubleJoinCode",
		"TestCheckCommandJSONDiagnosticsForTaskGroupUseAfterCloseCode",
		"TestCheckCommandJSONDiagnosticsForResourceAmbiguousProvenanceCode",
		"TestCheckCommandJSONDiagnosticsForIslandTransferNonLocalPayloadCode",
		"TestCheckCommandJSONDiagnosticsForActorUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForActorBranchConsumeReuseCode",
		"TestCheckCommandJSONDiagnosticsForActorMatchLoopConsumeReuseCodes",
		"TestCheckCommandJSONDiagnosticsForTaskUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForActorStructFieldAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleActorStructFieldAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForGenericActorStructFieldAliasUseAfterTransferCodes",
		"TestCheckCommandJSONDiagnosticsForGenericResourceAliasFinalizationCodes",
		"TestCheckCommandJSONDiagnosticsForTransitiveResourceAliasFinalizationCodes",
		"TestCheckCommandJSONDiagnosticsForEnumConstructorReturnResourceAliasCodes",
		"TestCheckCommandJSONDiagnosticsForTransitiveActorAliasUseAfterTransferCodes",
		"TestCheckCommandJSONDiagnosticsForTaskGroupCancelReturnProvenanceCodes",
		"TestCheckCommandJSONDiagnosticsForTaskHandleGroupOptionalPayloadJoinCloseAliasCodes",
		"TestCheckCommandJSONDiagnosticsForActorEnumPayloadAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleActorEnumPayloadAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForTaskStructFieldAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleTaskStructFieldAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForTaskEnumPayloadAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleTaskEnumPayloadAliasUseAfterTransferCode",
		"TestCheckCommandJSONDiagnosticsForPrivacyConsentSafetyCode",
		"TestCheckCommandJSONDiagnosticsForRecursiveSecretSignaturePrivacyCode",
		"TestCheckCommandJSONDiagnosticsForTooManyInputs",
		"TestCheckCommandRejectsLocalCapsuleDependencyCycle",
		"TestCheckCommandJSONDiagnosticsForCallableMutableCaptureGlobalEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCapturedCallableGlobalStorageCode",
		"TestCheckCommandJSONDiagnosticsForFunctionTypedParameterGlobalStorageCode",
		"TestCheckCommandJSONDiagnosticsForFunctionValueUnsupportedEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCapturingClosureRawPointerEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCallableResourceCaptureEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCallableMutableCaptureHeapEscapeCode",
		"TestCheckCommandJSONDiagnosticsForGenericClosureCaptureCode",
		"TestCheckCommandJSONDiagnosticsForGenericCallbackClosureCaptureCode",
		"TestCheckCommandJSONDiagnosticsForFunctionTypedStorageUnsupportedCaptureCode",
		"TestCheckCommandJSONDiagnosticsForFunctionTypedReturnUnsupportedCaptureCode",
		"TestCheckCommandJSONDiagnosticsForCapturedClosureExplicitTypeArgsCode",
		"TestCheckCommandJSONDiagnosticsForFunctionTypedExplicitTypeArgsCode",
		"TestCheckCommandJSONDiagnosticsForUnsupportedFunctionValueCallCode",
		"TestCheckCommandJSONDiagnosticsForGenericClosurePointerEscapeCode",
		"TestCheckCommandJSONDiagnosticsForGenericClosureDirectCallRequirementCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipUseAfterConsumeCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipPartialStructConsumeCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipPartialStructCopyAfterConsumeCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConsumeCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumCopyAfterConsumeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModulePartialCopyAfterConsumeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConstructorAfterConsumeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModulePartialEnumConstructorAfterConsumeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipOptionalPayloadConsumeCode",
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateGenericCallEscapeCodes",
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"SliceAggregateGenericCallEscapeCodes"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrGenericCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrGenericCallEscapeCodes",
		("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
			"ateFunctionTypedParameterCallEscapeCodes"),
		("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
			"ateFunctionTypedStructFieldCallEscapeCodes"),
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"SliceAggregateFunctionTypedStructFieldCallEscapeCodes"),
		("TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggreg" +
			"ateFunctionTypedEnumPayloadCallEscapeCodes"),
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"SliceAggregateFunctionTypedEnumPayloadCallEscapeCodes"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes",
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"OptionalPtrFunctionTypedCallbackCodes"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrNestedAggregateCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrEnumPayloadCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrEnumPayloadCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode",
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"PtrOptionalPayloadOwnedCallEscapeCode"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes",
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"PtrOptionalPayloadConsumeInoutCallEscapeCodes"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes",
		("TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrow" +
			"SliceOptionalPayloadBindingEscapeCodes"),
		"TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalAssignmentConsumeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowFixedArrayAliasReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowStringAliasReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowOptionalAssignmentEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentCallEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceOptionalAssignmentEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceStructEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceStructEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceStructEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceStructEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceEnumPayloadEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceEnumPayloadEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceEnumEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceEnumEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForBorrowedPtrOptionalGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForBorrowedStringGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrOptionalGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForBorrowedPtrAggregateOptionalGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrAggregateOptionalGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForBorrowedSliceOptionalPayloadGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForBorrowedSliceGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceOptionalPayloadGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceGlobalAssignmentCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumAliasReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumAliasReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateReturnEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateReturnEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes",
		("TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowP" +
			"trOptionalAssignmentGlobalEscapeCodes"),
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateGlobalEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateGlobalEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadReturnEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadGlobalEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumGlobalEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadGlobalEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumGlobalEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadInoutEscapeCode",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadInoutEscapeCode",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes",
		"TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes",
	} {
		if !strings.Contains(helperText, "\nfunc "+symbol+"(") {
			t.Fatalf("tetra_suite_test.go must contain %s after CLI test aggregation", symbol)
		}
	}
}

// ---- workspace_test.go ----

func TestWorkspaceInitAddListAndRemove(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "init", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"workspace init exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := os.Stat(filepath.Join(dir, "Tetra.workspace")); err != nil {
		t.Fatalf("expected Tetra.workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "add", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"workspace add exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "Tetra.workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `member "App"`) {
		t.Fatalf("workspace missing member:\n%s", string(raw))
	}

	var report struct {
		Root    string `json:"root"`
		Members []struct {
			Path      string `json:"path"`
			CapsuleID string `json:"capsule_id"`
			Status    string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(t, []string{"workspace", "list", "--format=json", dir}, 0, &report)
	if filepath.Clean(report.Root) != filepath.Clean(dir) || len(report.Members) != 1 ||
		report.Members[0].Path != "App" ||
		report.Members[0].CapsuleID != "tetra://app" ||
		report.Members[0].Status != "ok" {
		t.Fatalf("workspace list report = %#v", report)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "remove", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"workspace remove exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	raw, err = os.ReadFile(filepath.Join(dir, "Tetra.workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `member "App"`) {
		t.Fatalf("workspace member was not removed:\n%s", string(raw))
	}
}

func TestWorkspaceCheckGraphAndSync(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Math"
member "App"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"workspace check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "Workspace OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	var graph struct {
		Nodes []struct {
			Path      string `json:"path"`
			CapsuleID string `json:"capsule_id"`
		} `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	runCLIJSONStdout(t, []string{"workspace", "graph", "--format=json", dir}, 0, &graph)
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 || graph.Edges[0].From != "App" ||
		graph.Edges[0].To != "Math" {
		t.Fatalf("workspace graph = %#v", graph)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{"workspace", "sync", "--check", "--target", target, dir},
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf(
			"workspace sync --check exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String()+stderr.String(), "would generate") {
		t.Fatalf("sync --check output = stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, "Tetra.lock")); err == nil {
		t.Fatalf("workspace sync --check unexpectedly wrote App Tetra.lock")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat App Tetra.lock: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "sync", "--target", target, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf(
			"workspace sync exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	for _, rel := range []string{
		"Tetra.lock",
		"interfaces/math/core.t4i",
		"artifacts/math/core." + target + ".tobj",
		"seeds/app-deps.t4s",
	} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected workspace sync generated %s: %v", rel, err)
		}
	}
}

func TestWorkspaceCheckFailures(t *testing.T) {
	t.Run("missing member", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Missing"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf(
				"workspace check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "Missing") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("duplicate capsule id", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "A/Capsule.t4", `capsule A:
    id "tetra://dup"
    version "0.1.0"
`)
		writeCLIProjectFile(t, dir, "B/Capsule.t4", `capsule B:
    id "tetra://dup"
    version "0.1.0"
`)
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "A"
member "B"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf(
				"workspace check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "duplicate capsule id") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("dependency cycle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    deps:
        tetra://app 0.1.0 ../App
`)
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Math"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf(
				"workspace check exit code = %d, stdout=%q stderr=%q",
				code,
				stdout.String(),
				stderr.String(),
			)
		}
		if !strings.Contains(stderr.String(), "capsule dependency cycle") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

func TestWorkspaceBuildWritesPerMemberOutputsAndJSONSummary(t *testing.T) {
	target := mustHostTarget(t)
	tgt, err := ctarget.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeWorkspaceMainProject(t, dir, "App", "tetra://app", target, 0)
	writeWorkspaceMainProject(t, dir, "Tool", "tetra://tool", target, 0)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Tool"
`)
	outDir := filepath.Join(dir, "dist")

	var report struct {
		Command string `json:"command"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Skipped int    `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(
		t,
		[]string{"workspace", "build", "--target", target, "--format=json", "-o", outDir, dir},
		0,
		&report,
	)
	if report.Command != "build" || report.Total != 2 || report.Passed != 2 || report.Failed != 0 ||
		report.Skipped != 0 {
		t.Fatalf("workspace build report = %#v", report)
	}
	for _, rel := range []string{
		filepath.ToSlash(filepath.Join("App", defaultOutput(tgt, "exe"))),
		filepath.ToSlash(filepath.Join("Tool", defaultOutput(tgt, "exe"))),
	} {
		if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected workspace build output %s: %v", rel, err)
		}
	}
}

func TestWorkspaceBuildSkipsDependentAfterFailedDependency(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Lib/Capsule.t4", fmt.Sprintf(`capsule Lib:
    id "tetra://lib"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, target))
	writeCLIProjectFile(t, dir, "Lib/src/main.t4", "func main() -> Int:\n    return\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://lib 0.1.0 ../Lib
`, target))
	writeCLIProjectFile(t, dir, "App/src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Lib"
member "App"
`)

	var report struct {
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"members"`
	}
	runCLIJSONStdout(
		t,
		[]string{
			"workspace",
			"build",
			"--target",
			target,
			"--format=json",
			"-o",
			filepath.Join(dir, "dist"),
			dir,
		},
		1,
		&report,
	)
	if report.Failed != 1 || report.Skipped != 1 || len(report.Members) != 2 {
		t.Fatalf("workspace build report = %#v", report)
	}
	if report.Members[0].Path != "Lib" || report.Members[0].Status != "fail" {
		t.Fatalf("first member = %#v", report.Members[0])
	}
	if report.Members[1].Path != "App" || report.Members[1].Status != "skipped" ||
		!strings.Contains(report.Members[1].Detail, "Lib") {
		t.Fatalf("dependent member = %#v", report.Members[1])
	}
}

func TestWorkspaceTestFailFastJSONSummary(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeWorkspaceTestProject(t, dir, "Pass", "tetra://pass", target, "pass ok", "40 + 2 == 42")
	writeWorkspaceTestProject(t, dir, "Fail", "tetra://fail", target, "fail bad", "1 == 2")
	writeWorkspaceTestProject(t, dir, "Later", "tetra://later", target, "later ok", "2 + 2 == 4")
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Pass"
member "Fail"
member "Later"
`)

	var report struct {
		Command string `json:"command"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Skipped int    `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(
		t,
		[]string{"workspace", "test", "--target", target, "--fail-fast", "--format=json", dir},
		1,
		&report,
	)
	if report.Command != "test" || report.Total != 3 || report.Passed != 1 || report.Failed != 1 ||
		report.Skipped != 1 {
		t.Fatalf("workspace test report = %#v", report)
	}
	if report.Members[2].Path != "Later" || report.Members[2].Status != "skipped" {
		t.Fatalf("fail-fast member = %#v", report.Members[2])
	}
}

func TestWorkspaceRunMemberAndUnknownMember(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeWorkspaceMainProject(t, dir, "App", "tetra://app", target, 7)
	writeWorkspaceMainProject(t, dir, "Tool", "tetra://tool", target, 0)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Tool"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"workspace", "run", "App", "--workspace", dir, "--target", target},
		&stdout,
		&stderr,
	)
	if code != 7 {
		t.Fatalf(
			"workspace run exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI(
		[]string{"workspace", "run", "Missing", "--workspace", dir, "--target", target},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf(
			"unknown workspace run exit code = %d, stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "workspace member not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
