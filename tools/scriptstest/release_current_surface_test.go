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
	releaseGatePath := currentReleaseGatePath(version)
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
	surfaceMarkers := []string{
		"# Tetra Current Supported Surface",
		"Status: current for `" + version + "`",
		"`" + releaseGatePath + "`",
	}
	if version == "v1.0.0" {
		surfaceMarkers = append(surfaceMarkers, "The current major line is `v1.0.0`.")
		if strings.Contains(surface, "`v1.0.0` is a future label") {
			t.Fatalf("current supported surface still marks v1.0.0 as future while CompilerVersion is %s", version)
		}
	} else {
		surfaceMarkers = append(surfaceMarkers, "`v1.0.0` is a future label")
	}
	for _, want := range surfaceMarkers {
		if !strings.Contains(surface, want) {
			t.Fatalf("current supported surface doc missing %q", want)
		}
	}
}

func TestCurrentReleaseGatePathMapsV100ToV10Directory(t *testing.T) {
	if got, want := currentReleaseGatePath("v1.0.0"), "scripts/release/v1_0/gate.sh"; got != want {
		t.Fatalf("currentReleaseGatePath(v1.0.0) = %q, want %q", got, want)
	}
	if got, want := currentReleaseGatePath("v0.4.0"), "scripts/release/v0_4_0/gate.sh"; got != want {
		t.Fatalf("currentReleaseGatePath(v0.4.0) = %q, want %q", got, want)
	}
}

func currentReleaseGatePath(version string) string {
	if version == "v1.0.0" {
		return "scripts/release/v1_0/gate.sh"
	}
	versionSlug := strings.ReplaceAll(strings.TrimPrefix(version, "v"), ".", "_")
	return "scripts/release/v" + versionSlug + "/gate.sh"
}
