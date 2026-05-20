package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	var report struct {
		Found       bool     `json:"found"`
		Root        string   `json:"root"`
		CapsulePath string   `json:"capsule_path"`
		EntryPath   string   `json:"entry_path"`
		SourceRoots []string `json:"source_roots"`
		Targets     []string `json:"targets"`
	}
	runCLIJSONStdout(t, []string{"project", "info", "--format=json", dir}, 0, &report)
	if !report.Found || filepath.Clean(report.Root) != filepath.Clean(dir) || !strings.HasSuffix(filepath.ToSlash(report.CapsulePath), "Capsule.t4") || !strings.HasSuffix(filepath.ToSlash(report.EntryPath), "src/main.t4") {
		t.Fatalf("project info report = %#v", report)
	}
	if strings.Join(report.SourceRoots, ",") != "src" || strings.Join(report.Targets, ",") != "linux-x64" {
		t.Fatalf("project info roots/targets = %#v", report)
	}
}
