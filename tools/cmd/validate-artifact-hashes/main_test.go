package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactHashManifestValidatesGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "summary.json"), []byte(`{"schema":"example"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "api-diff"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api-diff", "api-docs.md"), []byte("# API\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := buildHashManifest(root, "artifact-hashes.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateHashManifest(manifestPath); err != nil {
		t.Fatalf("validate hash manifest: %v", err)
	}
}

func TestArtifactHashManifestRejectsModifiedArtifact(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "known_issues.md")
	if err := os.WriteFile(path, []byte("# Known Issues\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildHashManifest(root, "artifact-hashes.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Known Topics\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateHashManifest(manifestPath)
	if err == nil {
		t.Fatalf("expected modified artifact failure")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArtifactHashManifestRejectsUnlistedArtifact(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "known_issues.md")
	if err := os.WriteFile(path, []byte("# Known Issues\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildHashManifest(root, "artifact-hashes.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new-evidence.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err = validateHashManifest(manifestPath)
	if err == nil {
		t.Fatalf("expected unlisted artifact failure")
	}
	if !strings.Contains(err.Error(), "unlisted artifact new-evidence.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArtifactHashManifestRejectsUnsortedPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	raw := []byte(`{
  "schema":"tetra.release-artifact-hashes.v1alpha1",
  "root":".",
  "artifacts":[
    {"path":"z.json","sha256":"sha256:ca3d163bab055381827226140568f3bef7eaac187cebd76878e0b63e9e442356","size":3},
    {"path":"a.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":3}
  ]
}`)
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateHashManifest(manifestPath)
	if err == nil || !strings.Contains(err.Error(), "sorted by path") {
		t.Fatalf("expected sorted-path failure, got %v", err)
	}
}

func TestArtifactHashManifestRejectsInvalidHashFormat(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "known_issues.md")
	if err := os.WriteFile(path, []byte("# Known Issues\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	raw := []byte(`{
  "schema":"tetra.release-artifact-hashes.v1alpha1",
  "root":".",
  "artifacts":[
    {"path":"known_issues.md","sha256":"not-a-hash","size":13}
  ]
}`)
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateHashManifest(manifestPath)
	if err == nil || !strings.Contains(err.Error(), "invalid sha256 format") {
		t.Fatalf("expected hash format failure, got %v", err)
	}
}

func TestArtifactHashManifestRejectsTrailingJSONDocument(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "summary.json"), []byte(`{"schema":"example"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildHashManifest(root, "artifact-hashes.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("\n{}\n")...)
	manifestPath := filepath.Join(root, "artifact-hashes.json")
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateHashManifest(manifestPath)
	if err == nil || !strings.Contains(err.Error(), "manifest must contain a single JSON document") {
		t.Fatalf("expected single-document failure, got %v", err)
	}
}
