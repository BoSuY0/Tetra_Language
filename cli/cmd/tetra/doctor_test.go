package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

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
	var sawVersion, sawRuntime, sawManifest, sawManifestVersion, sawManifestSurface, sawSmokeSources, sawRuntimeExports bool
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
		if check.Name == "target metadata" && check.Status == "pass" && strings.Contains(check.Detail, "7 targets") && strings.Contains(check.Detail, "2 build-only") {
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

		check := targetMetadataCheck()
		if check.Status != "pass" {
			t.Fatalf("targetMetadataCheck = %#v", check)
		}
		wasi := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-wasi")
		if wasi.BuildOnly || wasi.RunMode != "wasi_runner" || wasi.RunRunner != "wasmtime" || !wasi.RunSupported || wasi.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-wasi target metadata = %#v", wasi)
		}
		web := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-web")
		if web.BuildOnly || web.RunMode != "web_runner" || !web.RunSupported || web.RunRunner == "" || web.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-web target metadata = %#v", web)
		}
		x32 := targetReportEntryForTest(t, buildTargetReportEntries(), "linux-x32")
		if !x32.BuildOnly || x32.RunMode != "host_probed" || x32.PointerWidthBits != 32 || x32.RegisterWidthBits != 64 || !strings.Contains(x32.UnsupportedReason, "host-probed source run/test execution") || !strings.Contains(x32.UnsupportedReason, "Linux kernel supports the x32 ABI") {
			t.Fatalf("linux-x32 target metadata = %#v", x32)
		} else if x32.RunSupported {
			if x32.RunUnsupportedReason != "" {
				t.Fatalf("linux-x32 supported host-probed metadata = %#v", x32)
			}
		} else if !strings.Contains(x32.RunUnsupportedReason, "does not support Linux x32 ABI execution") || !strings.Contains(x32.RunUnsupportedReason, "no host fallback") {
			t.Fatalf("linux-x32 unsupported host-probed metadata = %#v", x32)
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
		if wasi.BuildOnly || wasi.RunMode != "wasi_runner" || wasi.RunRunner != "" || wasi.RunSupported || !strings.Contains(wasi.RunUnsupportedReason, "missing WASI runner") {
			t.Fatalf("wasm32-wasi target metadata without runner = %#v", wasi)
		}
		web := targetReportEntryForTest(t, buildTargetReportEntries(), "wasm32-web")
		if web.BuildOnly || web.RunMode != "web_runner" || web.RunSupported ||
			!strings.Contains(web.RunUnsupportedReason, "browser runner unavailable") {
			t.Fatalf("wasm32-web target metadata without runner = %#v", web)
		}
	})
}

func targetReportEntryForTest(t *testing.T, entries []targetReportEntry, triple string) targetReportEntry {
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
