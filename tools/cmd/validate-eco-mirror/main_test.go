package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEcoMirrorAcceptsValidReport(t *testing.T) {
	out, err := runEcoMirrorValidator(t, validMirrorReport())
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoMirrorAcceptsHTTPSourceStore(t *testing.T) {
	report := strings.Replace(validMirrorReport(), `"source_store": "tetrahub-a"`, `"source_store": "http://127.0.0.1:8080/tetrahub"`, 1)
	out, err := runEcoMirrorValidator(t, report)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoMirrorRejectsUnknownField(t *testing.T) {
	report := strings.Replace(validMirrorReport(), "\n  \"id\":", "\n  \"unexpected\": true,\n  \"id\":", 1)
	out, err := runEcoMirrorValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMirrorRejectsBadHash(t *testing.T) {
	report := strings.Replace(validMirrorReport(), `"package_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, `"package_sha256": "sha256:not-hex"`, 1)
	out, err := runEcoMirrorValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid package_sha256") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMirrorRejectsPathMismatch(t *testing.T) {
	report := strings.Replace(validMirrorReport(), `"package_path": "packages/tetra_demo/0.1.0/linux-x64/package.todex"`, `"package_path": "packages/tetra_demo/0.1.0/windows-x64/package.todex"`, 1)
	out, err := runEcoMirrorValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "package_path mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMirrorRejectsOneSidedTrustHash(t *testing.T) {
	report := strings.Replace(validMirrorReport(), "  \"trust_snapshot_path\": \"packages/tetra_demo/0.1.0/linux-x64/trust.snapshot.json\",\n", "", 1)
	out, err := runEcoMirrorValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "trust_snapshot_path is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func validMirrorReport() string {
	return `{
  "schema": "tetra.eco.mirror.v1",
  "mirrored_at_unix": 0,
  "source_store": "tetrahub-a",
  "destination_store": "tetrahub-b",
  "id": "tetra://demo",
  "version": "0.1.0",
  "target": "linux-x64",
  "channel": "stable",
  "hub": "tetrahub-stable",
  "package_path": "packages/tetra_demo/0.1.0/linux-x64/package.todex",
  "package_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "metadata_path": "packages/tetra_demo/0.1.0/linux-x64/metadata.json",
  "metadata_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "trust_snapshot_path": "packages/tetra_demo/0.1.0/linux-x64/trust.snapshot.json",
  "trust_snapshot_sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
}`
}

func runEcoMirrorValidator(t *testing.T, report string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.eco.mirror.json")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--mirror", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
