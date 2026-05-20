package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentSupportedSurfaceDocumentIsReleaseAligned(t *testing.T) {
	root := repoRoot(t)
	version := currentReleaseVersion(t)
	versionSlug := strings.ReplaceAll(strings.TrimPrefix(version, "v"), ".", "_")
	releaseGatePath := "scripts/release/v" + versionSlug + "/gate.sh"
	read := func(path string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(raw)
	}
	readme := read("README.md")
	for _, want := range []string{
		"Tetra Language (" + version + ")",
		"docs/spec/current_supported_surface.md",
		releaseGatePath,
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing current surface marker %q", want)
		}
	}
	surface := read("docs/spec/current_supported_surface.md")
	for _, want := range []string{
		"# Tetra Current Supported Surface",
		"Status: current for `" + version + "`",
		"`v1.0.0` is a future label",
		"`" + releaseGatePath + "`",
	} {
		if !strings.Contains(surface, want) {
			t.Fatalf("current supported surface doc missing %q", want)
		}
	}
}
