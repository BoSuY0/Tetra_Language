package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestBlockReportCLIRejectsSymlinkedReportDir(t *testing.T) {
	realDir := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := validateReportPathSafety(filepath.Join(linkDir, "surface-headless-block-system.json"))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("validateReportPathSafety symlink err = %v, want symlink rejection", err)
	}
}

func TestBlockReportCLIRejectsArtifactOutsideReportDir(t *testing.T) {
	reportDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-artifact")
	artifact := surface.ArtifactReport{Kind: "component-app", Path: outside, SHA256: "sha256:" + strings.Repeat("a", 64), Size: 16}
	scan := surface.ArtifactScanReport{Root: filepath.Dir(outside), FilesChecked: 1, Pass: true}
	err := validateBlockReportArtifactLocality(filepath.Join(reportDir, "surface-headless-block-system.json"), scan, []surface.ArtifactReport{artifact})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "outside") {
		t.Fatalf("validateBlockReportArtifactLocality err = %v, want outside report dir rejection", err)
	}
}

func TestBlockReportCLIRejectsStaleArtifactHash(t *testing.T) {
	reportDir := t.TempDir()
	artifactPath := filepath.Join(reportDir, "artifact.bin")
	if err := os.WriteFile(artifactPath, []byte("fresh artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := surface.ArtifactReport{Kind: "component-app", Path: artifactPath, SHA256: "sha256:" + strings.Repeat("b", 64), Size: int64(len("fresh artifact"))}
	err := validateBlockReportArtifactFiles(reportDir, []surface.ArtifactReport{stale})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "sha256") {
		t.Fatalf("validateBlockReportArtifactFiles stale hash err = %v, want sha256 rejection", err)
	}

	sum := sha256.Sum256([]byte("fresh artifact"))
	valid := surface.ArtifactReport{Kind: "component-app", Path: artifactPath, SHA256: fmt.Sprintf("sha256:%x", sum), Size: int64(len("fresh artifact"))}
	if err := validateBlockReportArtifactFiles(reportDir, []surface.ArtifactReport{valid}); err != nil {
		t.Fatalf("validateBlockReportArtifactFiles valid artifact: %v", err)
	}
}

func TestBlockReportCLIRejectsSameCommitMismatch(t *testing.T) {
	err := validateSameCommit("abc123", "def456")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "same-commit") {
		t.Fatalf("validateSameCommit err = %v, want same-commit mismatch", err)
	}
	if err := validateSameCommit("abc123", "abc123"); err != nil {
		t.Fatalf("validateSameCommit matching commits: %v", err)
	}
	if err := validateSameCommit("", "abc123"); err != nil {
		t.Fatalf("validateSameCommit without expectation: %v", err)
	}
}
