package release_v030_static

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateUsesDedicatedV030Boundary(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(
		t,
		"scripts/release_v0_3_0_gate.sh",
		"scripts/release/v0_3_0/gate.sh directly",
	)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_3_0", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v0_3_0/gate.sh [--report-dir DIR] [--require-clean]",
		`release_version="v0.3.0"`,
		`release_artifact="tetra.release.v0_3_0.gate-report.v1"`,
		`release_gate_command="bash scripts/release/v0_3_0/gate.sh"`,
		`bash scripts/dev/bootstrap.sh`,
		`if [[ "$version" != "$release_version" ]]`,
		`expected ./tetra version to be $release_version`,
		`TETRA_TEST_ALL_RELEASE_VERSION="$release_version"`,
		`bash scripts/ci/test-all.sh --stabilization --keep-going`,
		`bash scripts/dev/fuzz-nightly.sh --short`,
		`go run ./tools/cmd/validate-fuzz-summary --report-dir "$artifacts_dir/fuzz-short"`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`TETRA_MACOS_RUNTIME_SMOKE_REPORT`,
		`TETRA_WINDOWS_RUNTIME_SMOKE_REPORT`,
		`run_step "macOS and Windows runtime execution evidence" check_runtime_execution_evidence`,
		`"$artifacts_dir/macos-runtime-smoke.json"`,
		`"$artifacts_dir/windows-runtime-smoke.json"`,
		`git diff --check`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.3.0 release gate missing %q", want)
		}
	}
}

func TestReleaseV030ChecklistIsNonClaimingAndVersionScoped(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(
			repoRoot(t),
			"docs",
			"checklists",
			"release",
			"legacy",
			"v0_3_0_release_gate.md",
		),
	)
	if err != nil {
		t.Fatalf("read v0.3.0 checklist: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"# v0.3.0 Release Gate Checklist",
		"docs/spec/flow/v0_3_scope.md",
		"tetra.release.v0_3_0.gate-report.v1",
		"bash scripts/release/v0_3_0/gate.sh --report-dir <report-dir>",
		"bash scripts/release/v0_3_0/gate.sh --report-dir <report-dir> --require-clean",
		"bash scripts/ci/test-all.sh --stabilization --keep-going",
		"bash scripts/dev/fuzz-nightly.sh --short",
		"git diff --check",
		"git status --porcelain --untracked-files=all",
		"validates both host-gated reports before archiving either runtime evidence artifact",
		"`go test packages` step clears release input environment variables",
		"historical release-gate checklist",
		"active release profile has moved past this version",
		"v1.0.0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.3.0 checklist missing %q", want)
		}
	}
}

func TestReleaseV030ChecklistAndGateRequireSecuritySignoff(t *testing.T) {
	checklistRaw, err := os.ReadFile(
		filepath.Join(
			repoRoot(t),
			"docs",
			"checklists",
			"release",
			"legacy",
			"v0_3_0_release_gate.md",
		),
	)
	if err != nil {
		t.Fatalf("read v0.3.0 checklist: %v", err)
	}
	checklist := string(checklistRaw)
	for _, want := range []string{
		"Security review gate: `docs/checklists/security_review_gate.md`.",
		"Security signoff evidence is mandatory",
		"bash scripts/release/v0_3_0/security-review.sh --signoff <security-review.md>",
		"TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>",
		"<report-dir>/artifacts/security-review.md",
		"<report-dir>/artifacts/security-review.md.sha256",
		"cycle-safe",
		"release-state records not-yet-archived signoff evidence as `deferred`",
		("final source of truth for signoff validity is `bash scripts/" +
			"release/v0_3_0/security-review.sh --signoff <report-dir>/artifacts/" +
			"security-review.md`"),
	} {
		if !strings.Contains(checklist, want) {
			t.Fatalf("v0.3.0 checklist missing security signoff evidence requirement %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		`check_security_review_signoff()`,
		`write_final_security_review_signoff()`,
		`TETRA_SECURITY_REVIEW_SIGNOFF`,
		`staged_security_review_signoff="$tmp_dir/security-review-source.md"`,
		`bash scripts/release/v0_3_0/security-review.sh --signoff "$artifacts_dir/security-review.md"`,
		`write_security_review_detached_hash()`,
		`artifacts/security-review.md.sha256`,
		`run_step "security review signoff" check_security_review_signoff`,
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("v0.3.0 release gate missing security review wiring %q", want)
		}
	}
	if strings.Contains(gate, `bash scripts/release/v1_0/security-review.sh`) {
		t.Fatalf("v0.3.0 release gate should use the v0.3.0-named security review entrypoint")
	}
}

func TestReleaseV030GateGoTestStepUnsetsReleaseInputEnv(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		`-u TETRA_SECURITY_REVIEW_SIGNOFF \`,
		`-u TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF \`,
		`-u TETRA_MACOS_RUNTIME_SMOKE_REPORT \`,
		`-u TETRA_WINDOWS_RUNTIME_SMOKE_REPORT \`,
		`-u TETRA_RESIDUAL_RISKS_JSON \`,
		`go test ./compiler/... ./cli/... ./tools/... -count=1`,
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf(
				"v0.3.0 release gate go test step should unset release input env; missing %q",
				want,
			)
		}
	}
}

func TestReleaseV030SecurityReviewWrapperUsesV030Name(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(
		t,
		"scripts/release_v0_3_0_security_review.sh",
		"scripts/release/v0_3_0/security-review.sh directly",
	)
	raw, err := os.ReadFile(
		filepath.Join(root, "scripts", "release", "v0_3_0", "security-review.sh"),
	)
	if err != nil {
		t.Fatalf("read v0.3.0 security review script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`Usage: bash scripts/release/v0_3_0/security-review.sh`,
		`release_v0_3_0_security_review`,
		`exec bash "$repo_root/scripts/release/v1_0/security-review.sh" "$@"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.3.0 security review script missing %q", want)
		}
	}
	assertNoLegacyMention(
		t,
		text,
		"scripts/release_v0_3_0_security_review.sh",
		"scripts/release/v0_3_0/security-review.sh",
	)
}
