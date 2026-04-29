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

func TestValidateEcoPublishRejectsUnknownMetadataField(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	metaPath := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "\n  \"package\":", "\n  \"unexpected\": true,\n  \"package\":", 1)
	if err := os.WriteFile(metaPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoPublishRejectsUnsafePackagePath(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	metaPath := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), `"file": "package.todex"`, `"file": "../linux-x64/package.todex"`, 1)
	if err := os.WriteFile(metaPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsafe package file path") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoPublishRejectsDownloadPathMismatch(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	metaPath := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), `"path": "packages/tetra_demo/0.1.0/linux-x64/package.todex"`, `"path": "packages/tetra_demo/0.1.0/windows-x64/package.todex"`, 1)
	if err := os.WriteFile(metaPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "download path mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoPublishRejectsUnsafeTrustSnapshotPath(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	metaPath := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), `"snapshot_file": "trust.snapshot.json"`, `"snapshot_file": "../trust.snapshot.json"`, 1)
	if err := os.WriteFile(metaPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsafe trust snapshot file path") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoPublishRejectsTrustSnapshotHashMismatch(t *testing.T) {
	root, id, version, target := makePublishFixture(t)
	snapshotPath := filepath.Join(root, "packages", capsuleIDDirectory(id), version, target, "trust.snapshot.json")
	if err := os.WriteFile(snapshotPath, []byte(`{"schema":"tetra.eco.trust-snapshot.v1","tampered":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runPublishValidator(t, root, id, version, target)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "trust snapshot hash mismatch") {
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
	trustSnapshot := []byte(`{"schema":"tetra.eco.trust-snapshot.v1","record_count":0}`)
	trustSum := sha256.Sum256(trustSnapshot)
	trustHash := hex.EncodeToString(trustSum[:])
	if err := os.WriteFile(filepath.Join(dir, "trust.snapshot.json"), trustSnapshot, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.todex"), pkg, 0o644); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{
  "schema": "tetra.eco.publish.v1beta",
  "channel": "beta",
  "hub": "local-beta",
  "published_at_unix": 0,
  "capsule": {
    "name": "Demo",
    "id": %q,
    "version": %q,
    "target": %q,
    "targets": [%q],
    "permissions": ["io"]
  },
  "package": {
    "file": "package.todex",
    "size": %d,
    "sha256": "sha256:%s"
  },
  "trust": {
    "snapshot_file": "trust.snapshot.json",
    "snapshot_sha256": "sha256:%s",
    "trust_tier": "high"
  },
  "downloads": [
    {"target": %q, "path": "packages/tetra_demo/0.1.0/linux-x64/package.todex"}
  ]
}
`, id, version, target, target, len(pkg), hex.EncodeToString(sum[:]), trustHash, target)
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
