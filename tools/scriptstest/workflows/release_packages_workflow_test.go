package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePackagesWorkflowRunsMemoryProductionGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"),
	)
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Memory production release gate",
		"export GOTELEMETRY=off",
		`export GOCACHE="${PWD}/.cache/go-build-memory-production-release"`,
		`export GOTMPDIR="${PWD}/.cache/go-tmp-memory-production-release"`,
		`mkdir -p "$GOCACHE" "$GOTMPDIR"`,
		`report_dir="${{ steps.meta.outputs.out_dir }}/memory-production-linux-x64"`,
		`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/memory-production-linux-x64/**`,
		"Package publishing for this workflow runs the Linux-x64 memory production release gate",
		"`targets.json`",
		"`memory-release-manifest.json`",
		"`ram-measurement.json`",
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
	section := releaseStepWindow(
		text,
		"name: Memory production release gate",
		"name: Surface product gate",
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf(
				"memory production release package gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
}

func TestReleasePackagesRunsSurfaceGateBeforePublishing(t *testing.T) {
	root := repoRoot(t)
	contract := loadSurfaceReleaseContract(t, root)
	reportRoot := `${{ steps.meta.outputs.out_dir }}/surface-product-v1`
	raw, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Surface product gate",
		`report_dir="` + reportRoot + `"`,
		"command -v rg",
		"sudo apt-get install -y weston ripgrep",
		`bash scripts/release/surface/product-gate.sh --report-dir "$report_dir"`,
		"name: Surface experimental regression gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-experimental-regression"`,
		`bash scripts/release/surface/gate.sh --report-dir "$report_dir"`,
		"name: Safe view lifetime gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/safe-view-lifetime"`,
		`bash scripts/release/safe-view-lifetime/gate.sh --report-dir "$report_dir"`,
		"name: Surface API stability gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-api-stability-v1"`,
		`bash scripts/release/surface/api-stability-gate.sh --report-dir "$report_dir"`,
		"name: Surface final readiness",
		`report_dir="${{ steps.meta.outputs.out_dir }}/surface-ui-production-final"`,
		`product_report_dir="${{ steps.meta.outputs.out_dir }}/surface-product-v1"`,
		`go run ./tools/cmd/validate-surface-final-readiness --write`,
		`--report-dir "$report_dir"`,
		`--product-report-dir "$product_report_dir"`,
		`--current-git-head "${GITHUB_SHA}"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-surface-final-readiness --report-dir "$report_dir" --expected-scope surface-v1-linux-web --require-clean --require-package`,
		`${{ steps.meta.outputs.out_dir }}/surface-experimental-regression/**`,
		`${{ steps.meta.outputs.out_dir }}/safe-view-lifetime/**`,
		`${{ steps.meta.outputs.out_dir }}/surface-api-stability-v1/**`,
		`${{ steps.meta.outputs.out_dir }}/surface-ui-production-final/**`,
		("Surface v1 product evidence runs the Surface product gate, " +
			"experimental regression gate, safe-view lifetime gate, API stability " +
			"gate, and final readiness validator before publishing."),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing Surface gate detail %q", want)
		}
	}

	gateIdx := strings.Index(text, "name: Surface product gate")
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
			t.Fatalf("Surface product gate must run before %q", publishStep)
		}
	}
	uploadSection := releaseStepWindow(
		text,
		"name: Upload package artifacts",
		"name: Create or update GitHub Release",
	)
	if uploadSection == "" {
		t.Fatalf("release-packages workflow missing Upload package artifacts section")
	}
	assertWorkflowUploadsContractArtifacts(t, uploadSection, reportRoot, contract)
	section := releaseStepWindow(
		text,
		"name: Surface product gate",
		"name: Surface experimental regression gate",
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf(
				"Surface product release package gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
	finalSection := releaseStepWindow(
		text,
		"name: Surface final readiness",
		"name: Integrated Memory/Islands/Surface release gate",
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(finalSection, forbidden) {
			t.Fatalf(
				"Surface final readiness release package gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
}

func TestReleasePackagesRunsIntegratedMemoryIslandsSurfaceGateBeforePublishing(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"),
	)
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Integrated Memory/Islands/Surface release gate",
		`report_dir="${{ steps.meta.outputs.out_dir }}/memory-islands-surface-production"`,
		`bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir "$report_dir"`,
		`${{ steps.meta.outputs.out_dir }}/memory-islands-surface-production/**`,
		("Integrated Memory/Islands/Surface evidence runs the strict " +
			"integrated gate before publishing and uploads"),
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
	root := repoRoot(t)
	contract := loadRAMContract(t, root)
	reportRoot := `${{ steps.meta.outputs.out_dir }}/ram-contract-linux-x64`
	raw, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: RAM contract release gate",
		`report_dir="` + reportRoot + `"`,
		`bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir "$report_dir"`,
		reportRoot + `/**`,
		("RAM Contract Compiler evidence runs the strict Linux-x64 RAM " +
			"contract release gate before publishing and uploads"),
		"`ram-contract-report.json`",
		"`memory-grade-report.json`",
		"`proof-store-summary.json`",
		"`validation-pipeline-coverage.json`",
		"`heap-blockers.json`",
		"`copy-blockers.json`",
		"`fuzz/ram-contract-fuzz-oracle.json`",
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
	if section := releaseStepWindow(
		text,
		"name: RAM contract release gate",
		"name: Actor runtime foundation release gate",
	); strings.Contains(
		section,
		"continue-on-error",
	) {
		t.Fatalf("RAM contract release gate must not use continue-on-error")
	}
	uploadSection := releaseStepWindow(
		text,
		"name: Upload package artifacts",
		"name: Create or update GitHub Release",
	)
	if uploadSection == "" {
		t.Fatalf("release-packages workflow missing Upload package artifacts section")
	}
	assertWorkflowUploadsContractArtifacts(t, uploadSection, reportRoot, contract)
}

func TestReleasePackagesRunsActorRuntimeFoundationGateBeforePublishing(t *testing.T) {
	root := repoRoot(t)
	contract := loadActorRuntimeFoundationContract(t, root)
	reportRoot := `${{ steps.meta.outputs.out_dir }}/actor-runtime-foundation-linux-x64`
	raw, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Actor runtime foundation release gate",
		`report_dir="` + reportRoot + `"`,
		`bash scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh --report-dir "$report_dir"`,
		reportRoot + `/**`,
		"Actor runtime foundation evidence runs the strict Linux-x64 gate before publishing and uploads",
		"`actor-runtime-foundation-manifest.json`",
		"`artifact-hashes.json`",
		"`distributed-actors-linux-x64/distributed-actors-linux-x64.json`",
		"`parallel-production-linux-x64/parallel-production-linux-x64.json`",
		("Actor runtime foundation evidence remains Linux-x64 scoped; " +
			"Erlang/OTP, cluster membership, reconnect/retry production, non-Linux " +
			"distributed runtime, distributed zero-copy transfer, and formal race " +
			"proof are not claimed."),
		`go run ./tools/cmd/validate-actor-capabilities --manifest docs/contracts/actors/actor-capability-manifest.v1.json --release-notes "$notes"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(
				"release-packages workflow missing actor runtime foundation gate detail %q",
				want,
			)
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
	if section := releaseStepWindow(
		text,
		"name: Actor runtime foundation release gate",
		"name: Upload package artifacts",
	); strings.Contains(
		section,
		"continue-on-error",
	) {
		t.Fatalf("actor runtime foundation release gate must not use continue-on-error")
	}
	uploadSection := releaseStepWindow(
		text,
		"name: Upload package artifacts",
		"name: Create or update GitHub Release",
	)
	if uploadSection == "" {
		t.Fatalf("release-packages workflow missing Upload package artifacts section")
	}
	assertWorkflowUploadsContractArtifacts(t, uploadSection, reportRoot, contract)
	assertOrderedFragments(t, text,
		`notes="$out_dir/release-notes.md"`,
		"notes.write_text(textwrap.dedent",
		`go run ./tools/cmd/validate-actor-capabilities --manifest docs/contracts/actors/actor-capability-manifest.v1.json --release-notes "$notes"`,
		`gh release upload "$version"`,
	)
}

func TestReleasePackagesWorkflowSupportsNonPublishingDryRun(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"),
	)
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"dry_run:",
		("Build packages and release evidence without creating a GitHub " +
			"Release, pushing GHCR, or updating Homebrew"),
		("RELEASE_DRY_RUN: ${{ github.event_name == 'workflow_dispatch' " +
			"&& inputs.dry_run && 'true' || 'false' }}"),
		"name: Dry-run package proof",
		"if: env.RELEASE_DRY_RUN == 'true'",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing dry-run detail %q", want)
		}
	}

	actorGate := releaseStepWindow(
		text,
		"name: Actor runtime foundation release gate",
		"name: Memory100 prod-stable gate",
	)
	if strings.Contains(actorGate, "RELEASE_DRY_RUN") {
		t.Fatalf("actor runtime foundation gate must still run during release package dry-run")
	}
	for _, externalStep := range []struct {
		name string
		next string
		want string
	}{
		{
			name: "name: Create or update GitHub Release",
			next: "name: Build container image",
			want: "if: env.RELEASE_DRY_RUN != 'true'",
		},
		{
			name: "name: Publish GHCR image",
			next: "name: Update Homebrew tap",
			want: "if: steps.meta.outputs.publish_container == 'true' && env.RELEASE_DRY_RUN != 'true'",
		},
		{
			name: "name: Update Homebrew tap",
			next: "name: release-packages-dry-run-end",
			want: "if: env.UPDATE_HOMEBREW_TAP == 'true' && env.RELEASE_DRY_RUN != 'true'",
		},
	} {
		section := releaseStepWindow(text, externalStep.name, externalStep.next)
		if !strings.Contains(section, externalStep.want) {
			t.Fatalf(
				"release-packages workflow external step %q missing dry-run guard %q",
				externalStep.name,
				externalStep.want,
			)
		}
	}
}

func TestReleasePackagesRunsMemory100ProdStableGateBeforePublishing(t *testing.T) {
	root := repoRoot(t)
	contract := loadMemory100Contract(t, root)
	reportRoot := `${{ steps.meta.outputs.out_dir }}/memory-100-prod-stable`
	raw, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: Memory100 prod-stable gate",
		"export GOTELEMETRY=off",
		`export GOCACHE="${PWD}/.cache/go-build-memory-100-prod-stable-release"`,
		`export GOTMPDIR="${PWD}/.cache/go-tmp-memory-100-prod-stable-release"`,
		`mkdir -p "$GOCACHE" "$GOTMPDIR"`,
		`report_dir="` + reportRoot + `"`,
		`bash scripts/release/post_v0_4/memory-100-prod-stable-gate.sh --report-dir "$report_dir"`,
		reportRoot + `/**`,
		("Memory100 scoped prod-stable evidence runs the strict aggregate " +
			"gate before publishing and uploads"),
		"`memory-100-prod-stable-manifest.json`",
		"`artifact-hashes.json`",
		"`runtime-memory/runtime-memory-contract.json`",
		"`proof-transition/proof-transition-report.json`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing Memory100 gate detail %q", want)
		}
	}

	memory100Idx := strings.Index(text, "name: Memory100 prod-stable gate")
	for _, predecessor := range []string{
		"name: Memory production release gate",
		"name: Integrated Memory/Islands/Surface release gate",
		"name: RAM contract release gate",
		"name: Actor runtime foundation release gate",
	} {
		predecessorIdx := strings.Index(text, predecessor)
		if predecessorIdx < 0 {
			t.Fatalf("release-packages workflow missing predecessor gate %q", predecessor)
		}
		if memory100Idx < 0 || memory100Idx < predecessorIdx {
			t.Fatalf("Memory100 prod-stable gate must run after %q", predecessor)
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
		if memory100Idx < 0 || memory100Idx > publishIdx {
			t.Fatalf("Memory100 prod-stable gate must run before %q", publishStep)
		}
	}
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if section := releaseStepWindow(
			text,
			"name: Memory100 prod-stable gate",
			"name: Upload package artifacts",
		); strings.Contains(
			section,
			forbidden,
		) {
			t.Fatalf(
				"Memory100 release package gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
	uploadSection := releaseStepWindow(
		text,
		"name: Upload package artifacts",
		"name: Create or update GitHub Release",
	)
	if uploadSection == "" {
		t.Fatalf("release-packages workflow missing Upload package artifacts section")
	}
	assertWorkflowUploadsContractArtifacts(t, uploadSection, reportRoot, contract)
}

func TestReleasePackagesWorkflowBindsPublishedArtifactsToCurrentCommit(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"),
	)
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

func TestReleasePackagesWorkflowMakesManualContainerPublishOptIn(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"))
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"publish_container:",
		"description: \"Publish the release image to GHCR; tag releases always publish\"",
		"default: false",
		`publish_container="${{ github.event_name != 'workflow_dispatch' || inputs.publish_container }}"`,
		`echo "publish_container=${publish_container}"`,
		`PUBLISH_CONTAINER="${{ steps.meta.outputs.publish_container }}"`,
		`publish_container = os.environ["PUBLISH_CONTAINER"] == "true"`,
		"Container publishing was not requested for this manual workflow run.",
		"if: steps.meta.outputs.publish_container == 'true'",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow missing manual container publish guard %q", want)
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
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), ".github", "workflows", "release-packages.yml"),
	)
	if err != nil {
		t.Fatalf("read release-packages workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		("No fastest-language, official benchmark, target parity, or " +
			"broad zero-cost performance claim is made by these package notes."),
		("Memory production evidence remains Linux-x64 scoped unless a " +
			"target-specific runtime gate says otherwise."),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-packages workflow release notes missing nonclaim %q", want)
		}
	}
}
