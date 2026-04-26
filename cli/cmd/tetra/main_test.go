package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("version exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "v0.6.0") {
		t.Fatalf("version output = %q, want compiler version", stdout.String())
	}
}

func TestTargetsCommandText(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"targets"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("targets exit code = %d, stdout=%q", code, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{"Supported targets:", "linux-x64", "windows-x64", "macos-x64", "Planned targets:", "wasm32-wasi", "wasm32-web"} {
		if !strings.Contains(out, want) {
			t.Fatalf("targets output missing %q:\n%s", want, out)
		}
	}
}

func TestTargetsCommandJSON(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"targets", "--format=json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("targets exit code = %d, stdout=%q", code, stdout.String())
	}
	var report struct {
		Supported []string `json:"supported"`
		Planned   []string `json:"planned"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("targets JSON: %v\n%s", err, stdout.String())
	}
	if strings.Join(report.Supported, ",") != "linux-x64,windows-x64,macos-x64" {
		t.Fatalf("supported targets = %#v", report.Supported)
	}
	if strings.Join(report.Planned, ",") != "wasm32-wasi,wasm32-web" {
		t.Fatalf("planned targets = %#v", report.Planned)
	}
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

func TestDoctorCommandJSON(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"doctor", "--format=json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("doctor exit code = %d, stdout=%q", code, stdout.String())
	}
	var report struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("doctor JSON: %v\n%s", err, stdout.String())
	}
	if report.Status != "pass" {
		t.Fatalf("doctor status = %q, report=%s", report.Status, stdout.String())
	}
	var sawVersion, sawRuntime, sawManifest, sawManifestVersion, sawManifestSurface, sawSmokeSources, sawRuntimeExports bool
	for _, check := range report.Checks {
		if check.Name == "version" && check.Status == "pass" {
			sawVersion = true
		}
		if check.Name == "__rt/actors_sysv.tetra" && check.Status == "pass" {
			sawRuntime = true
		}
		if check.Name == "docs/generated/manifest.json" && check.Status == "pass" {
			sawManifest = true
		}
		if check.Name == "docs manifest version" && check.Status == "pass" && check.Detail == compiler.Version() {
			sawManifestVersion = true
		}
		if check.Name == "docs manifest surface" && check.Status == "pass" && strings.Contains(check.Detail, "targets") && strings.Contains(check.Detail, "runtime symbols") {
			sawManifestSurface = true
		}
		if check.Name == "smoke sources" && check.Status == "pass" && strings.Contains(check.Detail, "sources") {
			sawSmokeSources = true
		}
		if check.Name == "runtime exports" && check.Status == "pass" && strings.Contains(check.Detail, "symbols") {
			sawRuntimeExports = true
		}
	}
	if !sawVersion || !sawRuntime || !sawManifest || !sawManifestVersion || !sawManifestSurface || !sawSmokeSources || !sawRuntimeExports {
		t.Fatalf("doctor missing expected checks: %#v", report.Checks)
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

func TestBuildCommandUsesDefaultInput(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	src := []byte(`fun main(): i32 { return 0 }`)
	if err := os.WriteFile(filepath.Join(dir, "main.tetra"), src, 0o644); err != nil {
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

func TestSmokeCommandWritesReport(t *testing.T) {
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	report := filepath.Join(t.TempDir(), "smoke.json")
	var stdout bytes.Buffer
	code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", report}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("smoke exit code = %d, stdout=%q", code, stdout.String())
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), `"cases"`) || !strings.Contains(string(raw), `"islands_hello"`) {
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
	if smokeReport.Target != target || !strings.HasPrefix(smokeReport.Version, "v0.6.") || len(smokeReport.Cases) == 0 {
		t.Fatalf("smoke report shape = %#v", smokeReport)
	}
	if smokeReport.Total != len(smokeReport.Cases) || smokeReport.Passed != len(smokeReport.Cases) || smokeReport.Failed != 0 {
		t.Fatalf("smoke report counts = %#v", smokeReport)
	}
}

func TestSmokeCommandListsCasesAsJSON(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"smoke", "--list", "--format=json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("smoke --list exit code = %d, stdout=%q", code, stdout.String())
	}
	var report struct {
		Total        int  `json:"total"`
		IslandsDebug bool `json:"islands_debug"`
		Cases        []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			ExpectedExit int    `json:"expected_exit"`
			DebugOnly    bool   `json:"debug_only"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("smoke list JSON: %v\n%s", err, stdout.String())
	}
	if report.Total != len(report.Cases) || report.Total < 39 {
		t.Fatalf("smoke list counts = total:%d len:%d", report.Total, len(report.Cases))
	}
	var sawFlowHello bool
	for _, c := range report.Cases {
		if c.Name == "flow_hello" && c.SrcPath == "examples/flow_hello.tetra" && c.ExpectedExit == 0 {
			sawFlowHello = true
		}
	}
	if !sawFlowHello {
		t.Fatalf("smoke list missing flow_hello: %#v", report.Cases)
	}
}

func TestSmokeCommandListsDebugOnlyCase(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"smoke", "--list", "--format=json", "--islands-debug"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("smoke --list exit code = %d, stdout=%q", code, stdout.String())
	}
	var report smokeListReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("smoke list JSON: %v\n%s", err, stdout.String())
	}
	if !report.IslandsDebug {
		t.Fatalf("islands_debug = false")
	}
	var sawDebug bool
	for _, c := range report.Cases {
		if c.Name == "islands_double_free" && c.DebugOnly {
			sawDebug = true
		}
	}
	if !sawDebug {
		t.Fatalf("debug smoke list missing islands_double_free: %#v", report.Cases)
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

func TestEcoVerifyPackAndUnpack(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	src := `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`
	if err := os.WriteFile(capsule, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := runCLI([]string{"eco", "verify", capsule}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q", code, stdout.String())
	}
	pkg := filepath.Join(dir, "demo.todex")
	if code := runCLI([]string{"eco", "pack", capsule, "-o", pkg}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco pack exit code = %d, stdout=%q", code, stdout.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q", code, stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
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
	if err := os.WriteFile(filepath.Join(srcDir, "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.todex")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkg}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "src", "main.tetra")); err != nil {
		t.Fatalf("expected bundled source: %v", err)
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
    dependency "tetra://core" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lock, app, core}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lock)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"capsules"`) || !strings.Contains(string(raw), `"tetra://core"`) {
		t.Fatalf("unexpected lock: %s", string(raw))
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
	code = runCLI([]string{"eco", "verify", "--target", "windows-x64", one}, &bytes.Buffer{}, &stderr)
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
	code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vault added: sha256:") {
		t.Fatalf("vault add stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "list", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault list exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "source") || !strings.Contains(stdout.String(), "module.tetra") {
		t.Fatalf("vault list stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

func TestBuildCommandJSONDiagnostics(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=json", "--target", mustHostTarget(t), "-o", filepath.Join(dir, "app"), srcPath}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected build failure")
	}
	out := stderr.String()
	if !strings.Contains(out, `"message"`) || !strings.Contains(out, `"severity":"error"`) {
		t.Fatalf("json diagnostics = %q", out)
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
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Checked: "+srcPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("check should not create %s, stat err=%v", outPath, err)
	}
}

func TestCheckCommandJSONDiagnosticsForSemanticError(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"check", "--diagnostics=json", srcPath}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected check failure")
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA2001" || diag.File != srcPath || diag.Severity != "error" || !strings.Contains(diag.Message, "uses effect 'io'") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"check", "--diagnostics=json", "one.tetra", "two.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "check accepts at most one input path" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestDocCommandWritesAPIDocsToStdout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	if err := os.WriteFile(srcPath, []byte("func answer() -> Int:\n    return 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "# Tetra API Docs") || !strings.Contains(stdout.String(), "`func answer() -> Int`") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDocCommandWritesAPIDocsToFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	outPath := filepath.Join(dir, "docs", "api.md")
	if err := os.WriteFile(srcPath, []byte("func answer() -> Int:\n    return 42\n"), 0o644); err != nil {
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
	if !strings.Contains(string(raw), "`func answer() -> Int`") {
		t.Fatalf("docs = %s", raw)
	}
}

func TestDocCommandJSONDiagnostics(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"doc", "--diagnostics=json", "/tmp/does-not-exist.tetra"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected doc failure")
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "no such file or directory") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestBuildCommandJSONDiagnosticsForOptionValidation(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=json", "--runtime=warpdrive", "examples/hello.tetra"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected build option failure")
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != `unsupported --runtime "warpdrive"` || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestBuildCommandJSONDiagnosticsForPlannedWASMTarget(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=json", "--target", "wasm32-wasi", "examples/hello.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "planned target not implemented: wasm32-wasi") {
		t.Fatalf("diagnostic = %#v", diag)
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

func TestRunCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	var stderr bytes.Buffer
	code := runCLI([]string{"run", "--diagnostics=json", "--target", target}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "cannot run target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestBuildCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=json", "one.tetra", "two.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "build accepts at most one input path" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForInvalidModeCombination(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--diagnostics=json", "--check", "--write", "examples/flow_hello.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "fmt accepts only one of --check or --write" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForMissingPath(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--diagnostics=json"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "fmt requires at least one path" || diag.Severity != "error" {
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
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--diagnostics=json", one, two}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "fmt stdout mode accepts exactly one file" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	var stderr bytes.Buffer
	code := runCLI([]string{"test", "--diagnostics=json", "--target", target}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "cannot run tests for target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandJSONDiagnosticsForUnsupportedReportFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"test", "--diagnostics=json", "--report=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Message != "unsupported --report format" || diag.Severity != "error" {
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
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--check", "--diagnostics=json", srcPath}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected fmt --check failure")
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Message != "not formatted" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
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
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report struct {
		Total      int   `json:"total"`
		Passed     int   `json:"passed"`
		Failed     int   `json:"failed"`
		DurationMS int64 `json:"duration_ms"`
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json report: %v\n%s", err, stdout.String())
	}
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 || len(report.Results) != 1 || report.Results[0].Name != "math" || !report.Results[0].Passed {
		t.Fatalf("report = %#v", report)
	}
	if report.DurationMS <= 0 || report.Results[0].DurationMS <= 0 {
		t.Fatalf("durations missing: %#v", report)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != srcPath || report.Files[0].Total != 1 || report.Files[0].Passed != 1 || report.Files[0].Failed != 0 {
		t.Fatalf("file report = %#v", report.Files)
	}
	if report.Files[0].DurationMS != report.Results[0].DurationMS || report.DurationMS != report.Results[0].DurationMS {
		t.Fatalf("duration aggregation mismatch: %#v", report)
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
	if !strings.Contains(out, "FAIL bad math") || !strings.Contains(out, "exit code 1") || !strings.Contains(out, "0/1 passed") {
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
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected failing test, stdout=%q stderr=%q", stdout.String(), stderr.String())
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json report: %v\n%s", err, stdout.String())
	}
	if report.Total != 1 || report.Passed != 0 || report.Failed != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}
	result := report.Results[0]
	if result.Name != "bad math" || result.Passed || result.ExitCode != 1 || result.Error != "exit code 1" {
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
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report struct {
		Total   int               `json:"total"`
		Passed  int               `json:"passed"`
		Failed  int               `json:"failed"`
		Files   []json.RawMessage `json:"files"`
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json report: %v\n%s", err, stdout.String())
	}
	if report.Total != 0 || report.Passed != 0 || report.Failed != 0 {
		t.Fatalf("report counts = %#v", report)
	}
	if report.Files == nil || len(report.Files) != 0 || report.Results == nil || len(report.Results) != 0 {
		t.Fatalf("empty arrays should be present, report = %#v\n%s", report, stdout.String())
	}
}

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
	if !strings.Contains(stdout.String(), `"symbols"`) || !strings.Contains(stdout.String(), `"main"`) {
		t.Fatalf("lsp stdout = %q", stdout.String())
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
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if !strings.Contains(out, `"documentFormattingProvider":true`) {
		t.Fatalf("document formatting capability missing: %q", out)
	}
	if !strings.Contains(out, `"method":"textDocument/publishDiagnostics"`) || !strings.Contains(out, `"diagnostics"`) {
		t.Fatalf("diagnostics notification missing: %q", out)
	}
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("shutdown response missing: %q", out)
	}
}

func TestLSPStdioCompletionReturnsOpenDocumentSymbols(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"label":"answer"`) || !strings.Contains(out, `"label":"main"`) {
		t.Fatalf("completion response missing expected symbols: %q", out)
	}
	if !strings.Contains(out, `"detail":"const answer: Int"`) {
		t.Fatalf("completion response missing detail: %q", out)
	}
}

func TestLSPStdioDefinitionReturnsOpenDocumentSymbolLocation(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"uri":"file:///sample.tetra"`) {
		t.Fatalf("definition response missing location uri: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) || !strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("definition response missing expected symbol range: %q", out)
	}
}

func TestLSPStdioFormattingReturnsFullDocumentEdit(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n  return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/formatting","params":{"textDocument":{"uri":"file:///sample.tetra"},"options":{"tabSize":4,"insertSpaces":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"newText"`) || !strings.Contains(out, `\n    return 0\n`) {
		t.Fatalf("formatting response missing formatted full-document edit: %q", out)
	}
	if !strings.Contains(out, `"end":{"character":0,"line":2}`) {
		t.Fatalf("formatting response missing full document range: %q", out)
	}
}

func TestLSPStdioDidChangePublishesUpdatedDiagnostics(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///sample.tetra","version":2},"contentChanges":[{"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}]}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///sample.tetra"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

func mustHostTarget(t *testing.T) string {
	t.Helper()
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	return target
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
