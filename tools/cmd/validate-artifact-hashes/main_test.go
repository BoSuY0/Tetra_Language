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
