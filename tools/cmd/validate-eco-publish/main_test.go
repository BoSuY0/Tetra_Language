package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEcoPublishAcceptsValidMetadata(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	out, err := runPublishValidator(t, root, id, version, target)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoPublishRejectsHashMismatch(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	path := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "package.todex")
	if err := os.WriteFile(path, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "package size mismatch") && !strings.Contains(string(out), "package hash mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func makePublishFixture(t *testing.T) (root string, id string, version string, target string) {
	t.Helper()
	root = t.TempDir()
	id = "tetra://demo"
	version = "0.1.0"
	target = "linux-x64"
	dir := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := []byte("todex")
	sum := sha256.Sum256(pkg)
	if err := os.WriteFile(filepath.Join(dir, "package.todex"), pkg, 0o644); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{
  "schema": "tetra.eco.publish.v1beta",
  "channel": "beta",
  "hub": "local-beta",
  "capsule": {
    "id": %q,
    "version": %q,
    "target": %q
  },
  "package": {
    "file": "package.todex",
    "size": %d,
    "sha256": "sha256:%s"
  },
  "trust": {
    "snapshot_sha256": "sha256:%s"
  }
}
`, id, version, target, len(pkg), hex.EncodeToString(sum[:]), strings.Repeat("a", 64))
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id, version, target
}

func runPublishValidator(t *testing.T, registry string, id string, version string, target string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--registry", registry, "--id", id, "--version", version, "--target", target)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
