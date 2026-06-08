package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePackagesWorkflowRunsMemoryProductionGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Memory production release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/memory-production-linux-x64"`,
		`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/memory-production-linux-x64/**`,
		"Package publishing for this workflow runs the Linux-x64 memory production release gate",
		"`targets.json`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing %q", want)
		}
	}

	gateIdx := strings.Index(text, "name: Memory production release gate")
	for _, publishStep := range []string{
		"name: Upload package artifacts",
		"name: Create or update GitHub Release",
		"name: Build container image",
		"name: Publish GHCR image",
		"name: Update Homebrew tap",
	} {
		publishIdx := strings.Index(text, publishStep)
		if publishIdx < 0 {
			t.Fatalf("release-packages workflow missing publish step %q", publishStep)
		}
		if gateIdx < 0 || gateIdx > publishIdx {
			t.Fatalf("memory production gate must run before %q", publishStep)
		}
	}
}

func TestReleasePackagesWorkflowReleaseNotesDeclarePerformanceNonClaims(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"No fastest-language, official benchmark, target parity, or broad zero-cost performance claim is made by these package notes.",
		"Memory production evidence remains Linux-x64 scoped unless a target-specific runtime gate says otherwise.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow release notes missing nonclaim %q", want)
		}
	}
}
