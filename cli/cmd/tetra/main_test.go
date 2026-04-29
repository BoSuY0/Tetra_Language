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
		RunUnsupportedReason    string `json:"run_unsupported_reason"`
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
	if got := byTriple["linux-x64"]; got.Status != "supported" || got.OS != "linux" || got.Arch != "x64" || got.ABI != "sysv" || got.Format != "elf" || got.BuildOnly || !got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
		t.Fatalf("linux-x64 metadata = %#v", got)
	}
	if got := byTriple["windows-x64"]; got.Status != "supported" || got.OS != "windows" || got.ABI != "win64" || got.Format != "pe" || got.ExeExt != ".exe" || !got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
		t.Fatalf("windows-x64 metadata = %#v", got)
	}
	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		got := byTriple[triple]
		if got.Status != "build_only" || got.Arch != "wasm32" || got.Format != "wasm" || got.ExeExt != ".wasm" || !got.BuildOnly || got.RunSupported || got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
			t.Fatalf("%s metadata = %#v", triple, got)
		}
		if !strings.Contains(got.RunUnsupportedReason, "build-only") || !strings.Contains(got.RunUnsupportedReason, "does not provide a production runtime runner") {
			t.Fatalf("%s run_unsupported_reason = %q", triple, got.RunUnsupportedReason)
		}
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

func TestFeaturesCommandJSON(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"features", "--format=json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("features exit code = %d, stdout=%q", code, stdout.String())
	}
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("features JSON: %v\n%s", err, stdout.String())
	}
	if report.Schema != "tetra.features.v1" {
		t.Fatalf("features schema = %q", report.Schema)
	}
	if report.Version != compiler.Version() {
		t.Fatalf("features version = %q, want %q", report.Version, compiler.Version())
	}
	statusByID := map[string]string{}
	statusSeen := map[string]bool{}
	for _, feature := range report.Features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" || feature.Stability == "" || len(feature.Docs) == 0 {
			t.Fatalf("feature missing required metadata: %#v", feature)
		}
		statusByID[feature.ID] = feature.Status
		statusSeen[feature.Status] = true
	}
	for _, status := range []string{"current", "experimental", "planned", "post-v1"} {
		if !statusSeen[status] {
			t.Fatalf("features output missing %s status: %#v", status, report.Features)
		}
	}
	for id, wantStatus := range map[string]string{
		"cli.core":                            "current",
		"targets.wasm-build-only":             "current",
		"stdlib.experimental-mirrors":         "experimental",
		"wasm.runtime-execution":              "planned",
		"eco.distributed-network":             "post-v1",
		"language.full-first-class-callables": "post-v1",
	} {
		if gotStatus := statusByID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
		}
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doctor", "--format=json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("doctor status = %q report=%s", report.Status, stdout.String())
	}
	var sawCapsule, sawEntry, sawRoots, sawLockSync bool
	for _, check := range report.Checks {
		if check.Name == "project capsule" && check.Status == "pass" && strings.Contains(filepath.ToSlash(check.Detail), "Capsule.t4") {
			sawCapsule = true
		}
		if check.Name == "project entry" && check.Status == "pass" && strings.Contains(filepath.ToSlash(check.Detail), "src/main.t4") {
			sawEntry = true
		}
		if check.Name == "project source roots" && check.Status == "pass" && strings.Contains(check.Detail, "src") {
			sawRoots = true
		}
		if check.Name == "project lock" && check.Status == "pass" && strings.Contains(check.Detail, "tetra project sync") {
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

func TestCheckCommandUsesDefaultMainT4(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.t4"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nimport components.counter as counter\nfunc main() -> Int:\n    return counter.value()\n")
	writeCLIProjectFile(t, dir, "ui/components/counter.t4", "module components.counter\nfunc value() -> Int:\n    return 42\n")

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
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nimport components.counter as counter\nfunc main() -> Int:\n    return counter.value()\n")
	writeCLIProjectFile(t, dir, "ui/components/counter.t4", "module components.counter\nfunc value() -> Int:\n    return 42\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
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

func TestCheckCommandResolvesLocalCapsuleDependencyImport(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

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
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, filepath.Join(dir, "App", "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://app"`) || !strings.Contains(string(raw), `"tetra://math"`) {
		t.Fatalf("lock did not include full path dependency graph:\n%s", string(raw))
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
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	lockPath := filepath.Join(dir, "App", "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, filepath.Join(dir, "App", "Capsule.t4"), filepath.Join(dir, "Math", "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("expected check failure for stale Tetra.lock, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Tetra.lock") || !strings.Contains(stderr.String(), "version mismatch") {
		t.Fatalf("stderr = %q, want Tetra.lock version mismatch", stderr.String())
	}
	if !strings.Contains(stderr.String(), "tetra project sync") {
		t.Fatalf("stderr = %q, want project sync repair hint", stderr.String())
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
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
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
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	appRoot := filepath.Join(dir, "App")
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	for _, want := range []string{`"kind": "object"`, `"target": "` + target + `"`, `"module": "math.core"`, `"public_api_hash": "sha256:`} {
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
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	code := runCLI([]string{"eco", "artifacts", "build", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int, c: Int) -> Int:\n    return a + b + c\n")

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "artifacts", "check", "--target", target, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected stale artifact failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	for _, want := range []string{"stale interface artifact", "math.core", "tetra eco artifacts build --target " + target} {
		if !strings.Contains(combined, want) {
			t.Fatalf("artifact check output missing %q:\nstdout=%s\nstderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestEcoArtifactsBuildCheckDryRunDoesNotWriteArtifacts(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--check", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected dry-run to report pending artifacts")
	}
	if !strings.Contains(stdout.String()+stderr.String(), "would generate") {
		t.Fatalf("dry-run output = stdout=%q stderr=%q, want would generate", stdout.String(), stderr.String())
	}
	for _, rel := range []string{"interfaces/math/core.t4i", "artifacts/math/core." + target + ".tobj", "seeds/app-deps.t4s", "Tetra.lock"} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("dry-run unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
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
	code := runCLI([]string{"eco", "artifacts", "build", "--all-targets", "--lock", filepath.Join(appRoot, "Tetra.lock"), filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build --all-targets exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core."+target+".tobj"))); err != nil {
		t.Fatalf("expected native object artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj"))); err == nil {
		t.Fatalf("unexpected wasm object artifact")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat wasm object: %v", err)
	}
}

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
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if code == 0 {
		t.Fatalf("expected project sync --check to report missing lock")
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "would generate lock") || !strings.Contains(combined, "Tetra.lock") {
		t.Fatalf("sync --check output = stdout=%q stderr=%q, want missing lock dry-run", stdout.String(), stderr.String())
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
	code := runCLI([]string{"project", "sync", "--target", "linux-x64", "--all-targets", dir}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if !strings.Contains(string(capsuleRaw), "interface interfaces/math/core.t4i") || !strings.Contains(string(capsuleRaw), "object "+target+" artifacts/math/core."+target+".tobj") {
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
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, "Tetra.lock")); err != nil {
		t.Fatalf("expected Tetra.lock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj"))); err == nil {
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
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsule := string(raw)
	if !strings.Contains(capsule, "deps:") || !strings.Contains(capsule, "tetra://math 0.1.0 ../Math") {
		t.Fatalf("Capsule.t4 missing dependency:\n%s", capsule)
	}
	if !strings.Contains(stdout.String(), "Added dependency") || !strings.Contains(stdout.String(), "run: tetra project sync") {
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
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected duplicate dependency failure")
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
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", "--id", "tetra://math-alt", "--version", "0.2.0", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "list", "--format=json", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps list exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report struct {
		Dependencies []struct {
			ID           string `json:"id"`
			Version      string `json:"version"`
			Path         string `json:"path"`
			ResolvedPath string `json:"resolved_path"`
			Status       string `json:"status"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("deps list JSON: %v\n%s", err, stdout.String())
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("dependencies = %#v", report.Dependencies)
	}
	dep := report.Dependencies[0]
	if dep.ID != "tetra://math" || dep.Version != "0.1.0" || dep.Path != "../Math" || dep.Status != "ok" || !strings.HasSuffix(filepath.ToSlash(dep.ResolvedPath), "/Math") {
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
	code := runCLI([]string{"project", "deps", "remove", "--id", "tetra://math", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "tetra://math") {
		t.Fatalf("dependency was not removed:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Removed dependency") || !strings.Contains(stdout.String(), "run: tetra project sync") {
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
	code := runCLI([]string{"project", "deps", "remove", "--id", "tetra://math", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("project deps remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("expected missing dependency failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "tetra://missing") || !strings.Contains(stderr.String(), "Missing") {
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
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("expected version mismatch failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
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
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("expected cycle failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "capsule dependency cycle") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

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
		t.Fatalf("workspace init exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "Tetra.workspace")); err != nil {
		t.Fatalf("expected Tetra.workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "add", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "Tetra.workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `member "App"`) {
		t.Fatalf("workspace missing member:\n%s", string(raw))
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "list", "--format=json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace list exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report struct {
		Root    string `json:"root"`
		Members []struct {
			Path      string `json:"path"`
			CapsuleID string `json:"capsule_id"`
			Status    string `json:"status"`
		} `json:"members"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("workspace list JSON: %v\n%s", err, stdout.String())
	}
	if filepath.Clean(report.Root) != filepath.Clean(dir) || len(report.Members) != 1 || report.Members[0].Path != "App" || report.Members[0].CapsuleID != "tetra://app" || report.Members[0].Status != "ok" {
		t.Fatalf("workspace list report = %#v", report)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "remove", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("workspace check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Workspace OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "graph", "--format=json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace graph exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	if err := json.Unmarshal(stdout.Bytes(), &graph); err != nil {
		t.Fatalf("workspace graph JSON: %v\n%s", err, stdout.String())
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 || graph.Edges[0].From != "App" || graph.Edges[0].To != "Math" {
		t.Fatalf("workspace graph = %#v", graph)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "sync", "--check", "--target", target, dir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected workspace sync --check to report pending writes")
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
		t.Fatalf("workspace sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"Tetra.lock", "interfaces/math/core.t4i", "artifacts/math/core." + target + ".tobj", "seeds/app-deps.t4s"} {
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
		if code == 0 {
			t.Fatalf("expected missing member failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
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
		if code == 0 {
			t.Fatalf("expected duplicate id failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
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
		if code == 0 {
			t.Fatalf("expected cycle failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "build", "--target", target, "--format=json", "-o", outDir, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("workspace build JSON: %v\n%s", err, stdout.String())
	}
	if report.Command != "build" || report.Total != 2 || report.Passed != 2 || report.Failed != 0 || report.Skipped != 0 {
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "build", "--target", target, "--format=json", "-o", filepath.Join(dir, "dist"), dir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected workspace build to fail, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var report struct {
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"members"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("workspace build JSON: %v\n%s", err, stdout.String())
	}
	if report.Failed != 1 || report.Skipped != 1 || len(report.Members) != 2 {
		t.Fatalf("workspace build report = %#v", report)
	}
	if report.Members[0].Path != "Lib" || report.Members[0].Status != "fail" {
		t.Fatalf("first member = %#v", report.Members[0])
	}
	if report.Members[1].Path != "App" || report.Members[1].Status != "skipped" || !strings.Contains(report.Members[1].Detail, "Lib") {
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "test", "--target", target, "--fail-fast", "--format=json", dir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected workspace test to fail, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("workspace test JSON: %v\n%s", err, stdout.String())
	}
	if report.Command != "test" || report.Total != 3 || report.Passed != 1 || report.Failed != 1 || report.Skipped != 1 {
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
	code := runCLI([]string{"workspace", "run", "App", "--workspace", dir, "--target", target}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("workspace run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "run", "Missing", "--workspace", dir, "--target", target}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("unknown workspace run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "workspace member not found") {
		t.Fatalf("stderr = %q", stderr.String())
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

func TestFormatsCommandListsOfficialT4Family(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"formats", "--format=json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("formats exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
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
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("formats json: %v\n%s", err, stdout.String())
	}
	seen := map[string]bool{}
	for _, format := range report.Formats {
		if format.Extension != "" {
			seen[format.Extension] = true
		}
		if format.FileName != "" {
			seen[format.FileName] = true
		}
	}
	for _, want := range []string{".t4", ".tetra", ".tdx", ".t4s", ".t4i", ".t4p", ".t4r", ".t4q", ".tneed", "Tetra.lock"} {
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
		ExcludedExamples []struct {
			SrcPath string `json:"src_path"`
			Reason  string `json:"reason"`
		} `json:"excluded_examples"`
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
	var sawHelloT4Exclusion bool
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
	for _, exclusion := range report.ExcludedExamples {
		if exclusion.SrcPath == "examples/projects/hello_t4/src/main.t4" && strings.Contains(exclusion.Reason, report.Target) {
			sawHelloT4Exclusion = true
		}
	}
	if !sawHelloT4Exclusion {
		t.Fatalf("smoke list missing T4 example exclusion for hello_t4: %#v", report.ExcludedExamples)
	}
}

func TestSmokeCommandKeepsInvalidDoubleFreeOutOfDebugList(t *testing.T) {
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
	for _, c := range report.Cases {
		if c.Name == "islands_double_free" {
			t.Fatalf("debug smoke list includes semantic-negative islands_double_free: %#v", c)
		}
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

func TestEcoTopLevelHelpMentionsVerifyLock(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco --help exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
				t.Fatalf("%v exit code = %d, stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), "usage:") {
				t.Fatalf("%v output missing usage text: stdout=%q stderr=%q", args, stdout.String(), stderr.String())
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
	if err := os.WriteFile(filepath.Join(srcDir, "main.t4"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.tdx")
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
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	for _, want := range []string{`"path": "` + capsule + `"`, `"linux-x64"`, `"wasm32-web"`, `"ui"`, `"fs.readWrite.userData"`, `"unsafe": "deny"`, `"reproducible": "required"`} {
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
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0 // keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--diagnostics=json", srcPath}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA_FMT001" || diag.File != srcPath || diag.Line != 2 || diag.Column != 14 || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "inline comments are not supported") {
		t.Fatalf("diagnostic message = %q", diag.Message)
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
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nfunc main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    deps:
        tetra://app 0.1.0 ../App
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")

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
		t.Fatalf("expected check failure for dependency cycle, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "capsule dependency cycle") {
		t.Fatalf("stderr = %q, want capsule dependency cycle", stderr.String())
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
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nfunc answer() -> Int:\n    return 42\n")

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
	if !strings.Contains(stdout.String(), "## app.main") || !strings.Contains(stdout.String(), "`func answer() -> i32`") {
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

func TestInterfaceCommandWritesT4IFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "math.t4")
	outPath := filepath.Join(dir, "math.t4i")
	if err := os.WriteFile(srcPath, []byte("module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"interface", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("interface exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("interface write exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Bool) -> Int:
    return a
`)

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"interface", "--check", "-o", outPath, srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected stale interface check failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
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
		t.Fatalf("check --interface-only exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	code := runCLI([]string{"build", "--interface-only", "--target", "linux-x64", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build --interface-only exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("build --interface-only should not emit %s, stat err=%v", outPath, err)
	}
	if !strings.Contains(stdout.String(), "Interface-only build checked") {
		t.Fatalf("stdout = %q", stdout.String())
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
	if !strings.Contains(stderr.String(), "main.t4") {
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

func TestRunCommandJSONDiagnosticsForWASMBuildOnlyRuntimeUnsupported(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"run", "--diagnostics=json", "--target", "wasm32-web"}, &bytes.Buffer{}, &stderr)
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
	for _, want := range []string{"cannot run target wasm32-web", "build-only target emits artifacts only", "unsupported runtime execution", "does not provide a production runtime runner"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandJSONDiagnosticsForWASMBuildOnlyRuntimeUnsupported(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"test", "--diagnostics=json", "--target", "wasm32-web"}, &bytes.Buffer{}, &stderr)
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
	for _, want := range []string{"cannot run tests for target wasm32-web", "build-only target emits artifacts only", "unsupported runtime execution", "does not provide a production test runner"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
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

func TestFormatCommandCheckJSONDiagnosticsIncludesFirstDiffPosition(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int uses io:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--check", "--diagnostics=json", srcPath}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	var diag struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Line != 1 || diag.Column != 19 || diag.Message != "not formatted" || diag.Severity != "error" {
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
	if !strings.Contains(stdout.String(), "1/1 passed") || strings.Contains(stdout.String(), "should not run") {
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
	writeCLIProjectFile(t, dir, "src/app/util.t4", "module app.util\nfunc answer() -> Int:\n    return 42\n")
	writeCLIProjectFile(t, dir, "tests/util_test.t4", "module util_test\nimport app.util as util\ntest \"imports app util\":\n    expect util.answer() == 42\n")
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

func TestNewAppScaffoldCreatesRunnableT4Project(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "DemoApp")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "app", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("new app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
	for _, want := range []string{`capsule DemoApp:`, `id "tetra://apps/demoapp"`, `entry "src/main.t4"`, `source "src"`, `source "tests"`, `target "` + mustHostTarget(t) + `"`, `permission "io"`} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scaffold check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"test", "--target", mustHostTarget(t), appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scaffold test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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
		t.Fatalf("new app --lock exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appDir, "Tetra.lock"))
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://apps/lockedapp"`) {
		t.Fatalf("Tetra.lock missing scaffold capsule id:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Created app") || !strings.Contains(stdout.String(), "Tetra.lock") {
		t.Fatalf("stdout = %q, want scaffold and lock messages", stdout.String())
	}
}

func TestNewAppRejectsExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "app", dir}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("new app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "info", "--format=json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project info exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report struct {
		Found       bool     `json:"found"`
		Root        string   `json:"root"`
		CapsulePath string   `json:"capsule_path"`
		EntryPath   string   `json:"entry_path"`
		SourceRoots []string `json:"source_roots"`
		Targets     []string `json:"targets"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("project info JSON: %v\n%s", err, stdout.String())
	}
	if !report.Found || filepath.Clean(report.Root) != filepath.Clean(dir) || !strings.HasSuffix(filepath.ToSlash(report.CapsulePath), "Capsule.t4") || !strings.HasSuffix(filepath.ToSlash(report.EntryPath), "src/main.t4") {
		t.Fatalf("project info report = %#v", report)
	}
	if strings.Join(report.SourceRoots, ",") != "src" || strings.Join(report.Targets, ",") != "linux-x64" {
		t.Fatalf("project info roots/targets = %#v", report)
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
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
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
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")
	return filepath.Join(dir, "App")
}

func writeWorkspaceMainProject(t *testing.T, root string, name string, id string, target string, exitCode int) {
	t.Helper()
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "Capsule.t4")), fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target))
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/main.t4")), fmt.Sprintf("func main() -> Int:\n    return %d\n", exitCode))
}

func writeWorkspaceTestProject(t *testing.T, root string, name string, id string, target string, testName string, condition string) {
	t.Helper()
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "Capsule.t4")), fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target))
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/main.t4")), "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/tests.t4")), fmt.Sprintf("test %q:\n    expect %s\n", testName, condition))
}
