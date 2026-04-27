package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

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
				t.Fatalf("%s --help exit code = %d, stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), command) && !strings.Contains(strings.ToLower(combined), "usage") {
				t.Fatalf("%s --help output does not describe the command: stdout=%q stderr=%q", command, stdout.String(), stderr.String())
			}
		})
		t.Run(command+"_invalid_arg", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{command, "--definitely-invalid"}, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("%s invalid arg exit code = %d, stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
			}
		})
	}
}

func documentedCLICommands(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "spec", "cli_contracts.md"))
	if err != nil {
		t.Fatalf("read cli contracts: %v", err)
	}
	seen := map[string]bool{}
	var commands []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		rest := strings.TrimPrefix(line, "| `")
		command, _, ok := strings.Cut(rest, "`")
		if !ok || command == "tetra" || strings.Contains(command, " ") || command == "" || command[0] < 'a' || command[0] > 'z' {
			continue
		}
		if !seen[command] {
			seen[command] = true
			commands = append(commands, command)
		}
	}
	return commands
}

func TestTargetsCommandText(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"targets"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("targets exit code = %d, stdout=%q", code, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{"Supported targets:", "linux-x64", "windows-x64", "macos-x64", "Build-only targets:", "wasm32-wasi", "wasm32-web", "Planned targets:"} {
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
	type targetMeta struct {
		Triple                  string `json:"triple"`
		Status                  string `json:"status"`
		OS                      string `json:"os"`
		Arch                    string `json:"arch"`
		ABI                     string `json:"abi"`
		Format                  string `json:"format"`
		ExeExt                  string `json:"exe_ext"`
		BuildOnly               bool   `json:"build_only"`
		RunSupported            bool   `json:"run_supported"`
		SupportsDebugInfo       bool   `json:"supports_debug_info"`
		SupportsReleaseOptimize bool   `json:"supports_release_optimize"`
	}
	var report struct {
		Supported []string     `json:"supported"`
		BuildOnly []string     `json:"build_only"`
		Planned   []string     `json:"planned"`
		Targets   []targetMeta `json:"targets"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("targets JSON: %v\n%s", err, stdout.String())
	}
	if strings.Join(report.Supported, ",") != "linux-x64,windows-x64,macos-x64" {
		t.Fatalf("supported targets = %#v", report.Supported)
	}
	if strings.Join(report.BuildOnly, ",") != "wasm32-wasi,wasm32-web" {
		t.Fatalf("build-only targets = %#v", report.BuildOnly)
	}
	if len(report.Planned) != 0 {
		t.Fatalf("planned targets = %#v", report.Planned)
	}
	if len(report.Targets) != 5 {
		t.Fatalf("targets metadata count = %d, want 5: %#v", len(report.Targets), report.Targets)
	}
	byTriple := map[string]targetMeta{}
	for _, tgt := range report.Targets {
		byTriple[tgt.Triple] = tgt
	}
	if got := byTriple["linux-x64"]; got.Status != "supported" || got.OS != "linux" || got.Arch != "x64" || got.ABI != "sysv" || got.Format != "elf" || got.BuildOnly {
		t.Fatalf("linux-x64 metadata = %#v", got)
	}
	if got := byTriple["windows-x64"]; got.Status != "supported" || got.OS != "windows" || got.ABI != "win64" || got.Format != "pe" || got.ExeExt != ".exe" {
		t.Fatalf("windows-x64 metadata = %#v", got)
	}
	if got := byTriple["wasm32-wasi"]; got.Status != "build_only" || got.OS != "wasi" || got.Arch != "wasm32" || got.ABI != "wasi" || got.Format != "wasm" || !got.BuildOnly || got.RunSupported {
		t.Fatalf("wasm32-wasi metadata = %#v", got)
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
	var sawTargetMetadata, sawToolingCommands bool
	var sawBuildOnlyTargets bool
	for _, check := range report.Checks {
		if check.Name == "version" && check.Status == "pass" {
			sawVersion = true
		}
		if check.Name == "build-only targets" && check.Status == "pass" && strings.Contains(check.Detail, "wasm32-wasi") && strings.Contains(check.Detail, "wasm32-web") {
			sawBuildOnlyTargets = true
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
		if check.Name == "target metadata" && check.Status == "pass" && strings.Contains(check.Detail, "5 targets") && strings.Contains(check.Detail, "2 build-only") {
			sawTargetMetadata = true
		}
		if check.Name == "tooling commands" && check.Status == "pass" && strings.Contains(check.Detail, "fmt") && strings.Contains(check.Detail, "test") {
			sawToolingCommands = true
		}
	}
	if !sawVersion || !sawBuildOnlyTargets || !sawRuntime || !sawManifest || !sawManifestVersion || !sawManifestSurface || !sawSmokeSources || !sawRuntimeExports || !sawTargetMetadata || !sawToolingCommands {
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
		"examples/flow_hello.tetra":             false,
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
	if smokeReport.Target != target || smokeReport.Version != compiler.Version() || len(smokeReport.Cases) == 0 {
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
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("smoke list JSON: %v\n%s", err, stdout.String())
	}
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
	for _, c := range report.Cases {
		if c.Name == "flow_hello" && c.SrcPath == "examples/flow_hello.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 0 {
			sawFlowHello = true
		}
		if c.Name == "ui_native_shell_smoke" && c.SrcPath == "examples/ui_native_shell_smoke.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 0 {
			sawUINative = true
		}
		if c.Name == "complex_control_flow_smoke" && c.SrcPath == "examples/complex_control_flow_smoke.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 42 {
			sawComplexControl = true
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

func TestSmokeCommandListsWASMBuildOnlyTarget(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		code := runCLI([]string{"smoke", "--list", "--target", target, "--format=json"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("smoke --list exit code = %d, stdout=%q", code, stdout.String())
		}
		var report struct {
			Target       string `json:"target"`
			BuildOnly    bool   `json:"build_only"`
			RunSupported bool   `json:"run_supported"`
			Cases        []struct {
				Name        string `json:"name"`
				SrcPath     string `json:"src_path"`
				TargetGroup string `json:"target_group"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("smoke list JSON: %v\n%s", err, stdout.String())
		}
		if report.Target != target || !report.BuildOnly || report.RunSupported {
			t.Fatalf("wasm smoke list metadata = %#v", report)
		}
		var sawUIWeb bool
		for _, c := range report.Cases {
			if c.Name == "ui_web_smoke" && c.SrcPath == "examples/ui_web_smoke.tetra" && c.TargetGroup == "wasm" {
				sawUIWeb = true
			}
		}
		if !sawUIWeb {
			t.Fatalf("wasm smoke list missing ui_web_smoke: %#v", report.Cases)
		}
	}
}

func TestSmokeCommandBuildsWASMTargetWithoutRun(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		reportPath := filepath.Join(t.TempDir(), target+"-smoke.json")
		code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", reportPath}, &stdout, &bytes.Buffer{})
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
			if !strings.HasSuffix(c.OutPath, ".wasm") {
				t.Fatalf("expected wasm output path, case=%#v", c)
			}
			if c.Error != "" {
				t.Fatalf("unexpected wasm smoke error for %s: %s", c.Name, c.Error)
			}
		}
	}
}

func TestSmokeCommandWASMTargetGroupsIncludeDogfoodWebUI(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"smoke", "--list", "--target", "wasm32-web", "--format=json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("smoke --list exit code = %d, stdout=%q", code, stdout.String())
	}
	var report smokeListReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("smoke list JSON: %v\n%s", err, stdout.String())
	}
	required := map[string]string{
		"ui_web_smoke":   "examples/ui_web_smoke.tetra",
		"dogfood_web_ui": "examples/projects/dogfood_web_ui/src/main.tetra",
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

func TestCleanCommandRemovesCacheDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.MkdirAll(filepath.Join(dir, path, "nested"), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, path, "nested", "entry"), []byte("cache"), 0o644); err != nil {
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
			t.Fatalf("cache dir %s still exists or stat failed with non-missing error: %v", path, err)
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
		t.Fatalf("clean --target exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{filepath.Join(".tetra_cache", "linux-x64"), filepath.Join("tetra_cache", "linux-x64")} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("target cache dir %s still exists or stat failed with non-missing error: %v", path, err)
		}
	}
	for _, path := range []string{filepath.Join(".tetra_cache", "windows-x64", "entry"), filepath.Join("tetra_cache", "windows-x64", "entry")} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("non-target cache entry %s should remain: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "linux-x64") {
		t.Fatalf("clean stdout should name target: %q", stdout.String())
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
	if _, err := os.Stat(filepath.Join(outDir, "tetra.package.json")); err != nil {
		t.Fatalf("expected bundled package metadata: %v", err)
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
	if !strings.Contains(stdout.String(), "# Tetra API Docs") || !strings.Contains(stdout.String(), "`func answer() -> i32`") {
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
			if !strings.Contains(loader, "tetra_web_v1") || !strings.Contains(loader, "tetra_main") {
				t.Fatalf("unexpected web loader content:\n%s", loader)
			}
		}
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
	if !strings.Contains(stderr.String(), "unsupported target") || !strings.Contains(stderr.String(), "supported targets: linux-x64, windows-x64, macos-x64") || !strings.Contains(stderr.String(), "build-only targets: wasm32-wasi, wasm32-web") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandJSONDiagnosticsForInvalidTarget(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--diagnostics=json", "--target", "not-a-target", "examples/flow_hello.tetra"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Hint     string `json:"hint"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA0001" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	for _, want := range []string{"unsupported target: not-a-target", "supported targets: linux-x64, windows-x64, macos-x64", "build-only targets: wasm32-wasi, wasm32-web"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
	if !strings.Contains(diag.Hint, "tetra targets") {
		t.Fatalf("diagnostic hint = %q", diag.Hint)
	}
}

func TestTargetAwareCommandsRejectInvalidTargetConsistently(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "build", args: []string{"build", "--target", "not-a-target", "examples/flow_hello.tetra"}},
		{name: "run", args: []string{"run", "--target", "not-a-target", "examples/flow_hello.tetra"}},
		{name: "test", args: []string{"test", "--target", "not-a-target", "examples/tooling_tests.tetra"}},
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
			for _, want := range []string{"unsupported target: not-a-target", "supported targets: linux-x64, windows-x64, macos-x64", "build-only targets: wasm32-wasi, wasm32-web"} {
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
	if !strings.Contains(stderr.String(), "main.tetra") {
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

func TestTestCommandRunsModuleFileWithImportsAndMain(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	srcPath := filepath.Join("..", "..", "..", "examples", "projects", "dogfood_cli", "src", "main.tetra")
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
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json report: %v\n%s", err, stdout.String())
	}
	if report.Total != 2 || report.Passed != 2 || report.Failed != 0 || len(report.Results) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Results[0].Name != "first" || report.Results[0].Index != 0 || report.Results[0].FunctionName != "__tetra_test_0_first" || !report.Results[0].Passed {
		t.Fatalf("first result = %#v", report.Results[0])
	}
	if report.Results[1].Name != "second" || report.Results[1].Index != 1 || report.Results[1].FunctionName != "__tetra_test_1_second" || !report.Results[1].Passed {
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
	if !strings.Contains(out, `"method":"textDocument/publishDiagnostics"`) || !strings.Contains(out, `"diagnostics"`) {
		t.Fatalf("diagnostics notification missing: %q", out)
	}
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("shutdown response missing: %q", out)
	}
}

func TestLSPStdioTranscriptFixtureCoversEditingRequests(t *testing.T) {
	var input bytes.Buffer
	for _, body := range loadLSPTranscriptFixture(t, "full_session.jsonl") {
		writeLSPTestMessage(t, &input, body)
	}
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio fixture exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		`"id":1`,
		`"id":2`,
		`"contents":{"kind":"markdown","value":"const answer: i32"}`,
		`"id":3`,
		`"label":"answer"`,
		`"id":4`,
		`"uri":"file:///fixture.tetra"`,
		`"id":5`,
		`"newText":"value"`,
		`"id":6`,
		`"newText":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"`,
		`function 'main' uses effect 'io' but does not declare it`,
		`"id":7`,
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
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/codeAction","params":{"textDocument":{"uri":"file:///sample.tetra"},"range":{"start":{"line":1,"character":4},"end":{"line":1,"character":9}},"context":{"diagnostics":[{"range":{"start":{"line":1,"character":4},"end":{"line":1,"character":9}},"severity":1,"code":"TETRA2001","source":"tetra","message":"function 'main' uses effect 'io' but does not declare it"}]}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)

	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if !strings.Contains(out, `"start":{"character":18,"line":0}`) || !strings.Contains(out, `"end":{"character":18,"line":0}`) {
		t.Fatalf("codeAction edit missing insertion range: %q", out)
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
	if !strings.Contains(out, `"detail":"const answer: i32"`) {
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

func TestLSPStdioReferencesReturnsOpenDocumentLocations(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer + answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11},"context":{"includeDeclaration":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("references response missing: %q", out)
	}
	if got := strings.Count(out, `"uri":"file:///sample.tetra"`); got < 3 {
		t.Fatalf("references response missing locations: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) || !strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("references response missing declaration location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":11,"line":3}`) {
		t.Fatalf("references response missing first usage location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":20,"line":3}`) {
		t.Fatalf("references response missing second usage location: %q", out)
	}
}

func TestLSPStdioRenameReturnsWorkspaceEditForOpenDocument(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer + answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11},"newName":"value"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
