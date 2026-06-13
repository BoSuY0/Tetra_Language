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

func TestReleasePackagesRunsSurfaceGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Surface release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-release-v1"`,
		`bash scripts/release/surface/release-gate.sh --report-dir "$report_dir"`,
		"name: Surface experimental regression gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-experimental-regression"`,
		`bash scripts/release/surface/gate.sh --report-dir "$report_dir"`,
		"name: Safe view lifetime gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/safe-view-lifetime"`,
		`bash scripts/release/safe-view-lifetime/gate.sh --report-dir "$report_dir"`,
		"name: Surface API stability gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-api-stability-v1"`,
		`bash scripts/release/surface/api-stability-gate.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/surface-release-v1/**`,
		`${{ steps.meta.outputs.out_dir }}/surface-experimental-regression/**`,
		`${{ steps.meta.outputs.out_dir }}/safe-view-lifetime/**`,
		`${{ steps.meta.outputs.out_dir }}/surface-api-stability-v1/**`,
		"Surface v1 release evidence runs the Surface release gate, experimental regression gate, safe-view lifetime gate, and API stability gate before publishing.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing Surface gate detail %q", want)
		}
	}

	gateIdx := strings.Index(text, "name: Surface release gate")
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
			t.Fatalf("Surface release gate must run before %q", publishStep)
		}
	}
}

func TestReleasePackagesRunsLinuxSurfaceGatesUnderHeadlessWayland(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Install Surface display dependencies",
		"apt-get install -y weston",
		`scripts/release/surface/with-headless-wayland.sh bash scripts/release/surface/release-gate.sh --report-dir "$report_dir"`,
		`scripts/release/surface/with-headless-wayland.sh bash scripts/release/surface/gate.sh --report-dir "$report_dir"`,
		`scripts/release/surface/with-headless-wayland.sh bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir "$report_dir"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing headless Wayland Surface gate detail %q", want)
		}
	}

	installIdx := strings.Index(text, "name: Install Surface display dependencies")
	surfaceIdx := strings.Index(text, "name: Surface release gate")
	if installIdx < 0 || surfaceIdx < 0 || installIdx > surfaceIdx {
		t.Fatalf("Surface display dependencies must be installed before Surface release gate")
	}

	for _, window := range []struct {
		start string
		end   string
	}{
		{"name: Surface release gate", "name: Surface experimental regression gate"},
		{"name: Surface experimental regression gate", "name: Safe view lifetime gate"},
		{"name: Integrated Memory/Islands/Surface release gate", "name: RAM contract release gate"},
	} {
		section := releaseStepWindow(text, window.start, window.end)
		if strings.Contains(section, "continue-on-error") {
			t.Fatalf("%s must not use continue-on-error", window.start)
		}
		if strings.Contains(section, "|| true") || strings.Contains(section, "set +e") {
			t.Fatalf("%s must not hide headless Wayland or Surface gate failures", window.start)
		}
	}
}

func TestReleasePackagesRunsIntegratedMemoryIslandsSurfaceGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Integrated Memory/Islands/Surface release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/memory-islands-surface-production"`,
		`bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/memory-islands-surface-production/**`,
		"Integrated Memory/Islands/Surface evidence runs the strict integrated gate before publishing and uploads",
		"`memory-islands-surface-production-manifest.json`",
		"`artifact-hashes.json`",
		"`islands-debug-smoke.json`",
		"`island-proof-verifier.json`",
		"`island-proof-fuzz-summary.json`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing integrated gate detail %q", want)
		}
	}

	integratedIdx := strings.Index(text, "name: Integrated Memory/Islands/Surface release gate")
	for _, predecessor := range []string{
		"name: Memory production release gate",
		"name: Surface API stability gate",
	} {
		predecessorIdx := strings.Index(text, predecessor)
		if predecessorIdx < 0 {
			t.Fatalf("release-packages workflow missing predecessor gate %q", predecessor)
		}
		if integratedIdx < 0 || integratedIdx < predecessorIdx {
			t.Fatalf("integrated Memory/Islands/Surface gate must run after %q", predecessor)
		}
	}
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
		if integratedIdx < 0 || integratedIdx > publishIdx {
			t.Fatalf("integrated Memory/Islands/Surface gate must run before %q", publishStep)
		}
	}
}

func TestReleasePackagesRunsRAMContractGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: RAM contract release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/ram-contract-linux-x64"`,
		`bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/ram-contract-linux-x64/**`,
		"RAM Contract Compiler evidence runs the strict Linux-x64 RAM contract release gate before publishing and uploads",
		"`ram-contract-report.json`",
		"`memory-grade-report.json`",
		"`proof-store-summary.json`",
		"`validation-pipeline-coverage.json`",
		"`heap-blockers.json`",
		"`copy-blockers.json`",
		"`ram-contract-fuzz-oracle.json`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing RAM contract gate detail %q", want)
		}
	}

	gateIdx := strings.Index(text, "name: RAM contract release gate")
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
			t.Fatalf("RAM contract gate must run before %q", publishStep)
		}
	}
	if section := releaseStepWindow(text, "name: RAM contract release gate", "name: Actor runtime foundation release gate"); strings.Contains(section, "continue-on-error") {
		t.Fatalf("RAM contract release gate must not use continue-on-error")
	}
}

func TestReleasePackagesRunsActorRuntimeFoundationGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Actor runtime foundation release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/actor-runtime-foundation-linux-x64"`,
		`bash scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/actor-runtime-foundation-linux-x64/**`,
		"Actor runtime foundation evidence runs the strict Linux-x64 gate before publishing and uploads",
		"`actor-runtime-foundation-manifest.json`",
		"`artifact-hashes.json`",
		"`distributed-actors-linux-x64/distributed-actors-linux-x64.json`",
		"`parallel-production-linux-x64/parallel-production-linux-x64.json`",
		"Actor runtime foundation evidence remains Linux-x64 scoped; Erlang/OTP, cluster membership, reconnect/retry production, non-Linux distributed runtime, distributed zero-copy transfer, and formal race proof are not claimed.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing actor runtime foundation gate detail %q", want)
		}
	}

	gateIdx := strings.Index(text, "name: Actor runtime foundation release gate")
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
			t.Fatalf("actor runtime foundation gate must run before %q", publishStep)
		}
	}
	if section := releaseStepWindow(text, "name: Actor runtime foundation release gate", "name: Upload package artifacts"); strings.Contains(section, "continue-on-error") {
		t.Fatalf("actor runtime foundation release gate must not use continue-on-error")
	}
}

func TestReleasePackagesWorkflowBindsPublishedArtifactsToCurrentCommit(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: tetra-${{ steps.meta.outputs.version }}-${{ github.sha }}-release-packages",
		`existing_target="$(gh release view "$version" --json targetCommitish -q .targetCommitish)"`,
		`if [[ "$existing_target" != "$GITHUB_SHA" ]]; then`,
		`release target $existing_target does not match current commit $GITHUB_SHA`,
		`--target "$GITHUB_SHA"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing same-commit publish guard %q", want)
		}
	}
}

func TestReleasePackagesWorkflowSupportsDryRunWithoutPublication(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"dry_run:",
		"description: \"Build packages and run release gates without creating releases, publishing containers, or updating taps\"",
		"DRY_RUN: ${{ inputs.dry_run && 'true' || 'false' }}",
		"name: Upload package artifacts",
		"if: env.DRY_RUN != 'true'",
		"name: Dry-run summary",
		"Release package dry-run completed for ${{ steps.meta.outputs.version }} at ${{ github.sha }}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing dry-run guard %q", want)
		}
	}

	for _, publishStep := range []string{
		"name: Create or update GitHub Release",
		"name: Build container image",
		"name: Publish GHCR image",
		"name: Update Homebrew tap",
	} {
		section := releaseStepWindow(text, publishStep, "\n      - name:")
		if !strings.Contains(section, "if: env.DRY_RUN != 'true'") {
			t.Fatalf("release-packages workflow publish step %q must be skipped in dry-run mode", publishStep)
		}
	}
}

func releaseStepWindow(workflow, start, end string) string {
	startIdx := strings.Index(workflow, start)
	if startIdx < 0 {
		return ""
	}
	endIdx := strings.Index(workflow[startIdx:], end)
	if endIdx < 0 {
		return workflow[startIdx:]
	}
	return workflow[startIdx : startIdx+endIdx]
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
