package scriptstest

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseV10SecurityReviewWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "security-review.sh")
	assertLegacyFileRemoved(t, "scripts/release_v1_0_security_review.sh", "scripts/release/v1_0/security-review.sh directly")
	versionedRaw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned security review script: %v", err)
	}
	versionedText := string(versionedRaw)
	for _, want := range []string{
		"Usage:",
		"bash scripts/release/v1_0/security-review.sh --signoff PATH",
		"current_release_version()",
		"Decision:",
		"Evidence Commands",
		"Residual Risks",
	} {
		if !strings.Contains(versionedText, want) {
			t.Fatalf("scripts/release/v1_0/security-review.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, versionedText, "scripts/release_v1_0_security_review.sh", "scripts/release/v1_0/security-review.sh")
}

func TestReleaseV10SecurityReviewRejectsMissingPathArguments(t *testing.T) {
	root := releaseV10SecurityReviewPathFakeRepo(t)

	for _, tc := range []struct {
		flag string
		want string
	}{
		{flag: "--write-template", want: "security_review: --write-template requires a path"},
		{flag: "--signoff", want: "security_review: --signoff requires a path"},
	} {
		out, err := runReleaseV10SecurityReview(t, root, tc.flag)
		if err == nil {
			t.Fatalf("expected missing %s argument rejection\n%s", tc.flag, out)
		}
		if !strings.Contains(string(out), tc.want) {
			t.Fatalf("missing %s argument output missing controlled error %q:\n%s", tc.flag, tc.want, out)
		}
		assertOutputAvoidsRawPathUtilityErrors(t, out)
	}
}

func TestReleaseV10SecurityReviewRejectsDirectoryTemplatePath(t *testing.T) {
	root := releaseV10SecurityReviewPathFakeRepo(t)
	templatePath := filepath.Join(root, "template-dir")
	if err := os.MkdirAll(templatePath, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10SecurityReview(t, root, "--write-template", templatePath)
	if err == nil {
		t.Fatalf("expected directory template path rejection\n%s", out)
	}
	if !strings.Contains(string(out), "security_review: refusing to use directory template path: "+templatePath) {
		t.Fatalf("directory template output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10SecurityReviewWritesDashPrefixedTemplatePath(t *testing.T) {
	root := releaseV10SecurityReviewPathFakeRepo(t)
	templateArg := "-security-template.md"

	out, err := runReleaseV10SecurityReview(t, root, "--write-template", templateArg)
	if err != nil {
		t.Fatalf("dash-prefixed template path should work: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, templateArg)); err != nil {
		t.Fatalf("dash-prefixed template was not written: %v\n%s", err, out)
	}
}

func TestReleaseV10SecurityReviewAcceptsDashPrefixedSignoffPath(t *testing.T) {
	root := releaseV10SecurityReviewPathFakeRepo(t)
	signoffArg := "-security-review.md"
	signoffPath := filepath.Join(root, signoffArg)
	if err := os.WriteFile(signoffPath, []byte(validReleaseV10SecuritySignoff("v1.0.0", "0123456789abcdef0123456789abcdef01234567")), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10SecurityReview(t, root, "--signoff", signoffArg)
	if err != nil {
		t.Fatalf("dash-prefixed signoff path should validate: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestSecurityReviewSignoffValidatorAcceptsAuditableSignoff(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111
- security-review.md: sha256:2222222222222222222222222222222222222222222222222222222222222222

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("security review signoff should validate: %v\n%s", err, out)
	}
}

func TestSecurityReviewSignoffValidatorRejectsTemplatePlaceholders(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	version := currentReleaseVersion(t)
	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--write-template", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("write template failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(signoff)
	if err != nil {
		t.Fatalf("read generated template: %v", err)
	}
	for _, want := range []string{
		"# " + version + " Security Review Signoff",
		"Decision: <approved for " + version + " release | blocked>",
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("generated template missing current release text %q:\n%s", want, raw)
		}
	}

	cmd = exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("placeholder signoff should fail validation\n%s", out)
	}
	if !strings.Contains(string(out), "placeholder") {
		t.Fatalf("validator should explain placeholder failure:\n%s", out)
	}
}

func TestReleaseV040SecurityReviewWritesV040Template(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	cmd := exec.Command("bash", "scripts/release/v0_4_0/security-review.sh", "--write-template", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("write v0.4.0 template failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(signoff)
	if err != nil {
		t.Fatalf("read generated template: %v", err)
	}
	for _, want := range []string{
		"# v0.4.0 Security Review Signoff",
		"Decision: <approved for v0.4.0 release | blocked>",
		"`go run ./tools/cmd/validate-v0-4-readiness",
		"`go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json`",
		"`bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0`",
		"`bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0`",
		"`git diff --check`",
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("generated v0.4.0 template missing %q:\n%s", want, raw)
		}
	}
}

func TestReleaseV040SecurityReviewValidatesScopedSignoff(t *testing.T) {
	root, head := releaseV040SecurityReviewPathFakeRepo(t)
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	raw := `# v0.4.0 Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v0.4.0-gate
Decision: approved for v0.4.0 release

## Evidence Commands

- ` + "`go run ./tools/cmd/validate-v0-4-readiness --features <features.json> --targets <targets.json> --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4_0_scope_decisions.json`: pass" + `
- ` + "`go test ./compiler/... ./cli/... ./tools/... -count=1`: pass" + `
- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json`: pass" + `
- ` + "`bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0`: pass" + `
- ` + "`bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0`: pass" + `
- ` + "`git diff --check`: pass" + `

## Security Areas

- filesystem, networking, crypto, and capability effects for the scoped Linux surface: approved
- Linux-x64 runtime execution and native process boundaries: approved
- UI event dispatch and command execution: approved
- distributed actors, scheduling, cancellation, and failure modes: approved
- artifact hashes and release-state integrity: approved
- excluded EcoNet/WASI/Web/Windows/macOS boundaries are not part of this v0.4.0 production signoff: approved

## Artifact Hashes

- summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111

## Residual Risks

- None
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v0_4_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("valid scoped v0.4.0 signoff should pass: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "v0.4.0 security review signoff valid") {
		t.Fatalf("v0.4.0 security review output missing success line:\n%s", out)
	}
}

func TestReleaseV040SecurityReviewWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_4_0_security_review.sh", "scripts/release/v0_4_0/security-review.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_4_0", "security-review.sh"))
	if err != nil {
		t.Fatalf("read v0.4.0 security review script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`bash scripts/release/v0_4_0/security-review.sh --signoff PATH`,
		`bash scripts/release/v0_4_0/security-review.sh --write-template PATH`,
		`release_v0_4_0_security_review`,
		`v0.4.0 security review signoff valid`,
		`excluded EcoNet/WASI/Web/Windows/macOS boundaries`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.4.0 security review script missing %q", want)
		}
	}
	assertNoLegacyMention(t, text, "scripts/release_v0_4_0_security_review.sh", "scripts/release/v0_4_0/security-review.sh")
}

func TestSecurityReviewSignoffValidatorRequiresPrivacyAndWASMEvidence(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("signoff without privacy/WASM evidence should fail\n%s", out)
	}
	for _, want := range []string{
		"go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("validator should name missing evidence %q:\n%s", want, out)
		}
	}
}

func TestSecurityReviewSignoffValidatorRequiresArtifactHashes(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:not-a-real-hash

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("signoff with invalid artifact hash should fail\n%s", out)
	}
	if !strings.Contains(string(out), "artifact hash") {
		t.Fatalf("validator should name artifact hash failure:\n%s", out)
	}
}

func TestSecurityReviewSignoffValidatorRejectsWrongReleaseDecision(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for v0.1.2 release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("signoff with wrong release decision should fail\n%s", out)
	}
	if !strings.Contains(string(out), "Decision") {
		t.Fatalf("validator should name decision failure:\n%s", out)
	}
}

func TestSecurityReviewSignoffValidatorRejectsWrongCommit(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: 0000000000000000000000000000000000000000
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/security-review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("signoff with wrong commit should fail\n%s", out)
	}
	if !strings.Contains(string(out), "Reviewed commit") {
		t.Fatalf("validator should name commit failure:\n%s", out)
	}
}

func TestV0_3_0_security_signoff_rejects_mismatched_canonical_artifact(t *testing.T) {
	dir := t.TempDir()
	reportDir := filepath.Join(dir, "report")
	artifactsDir := filepath.Join(reportDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	summaryPath := filepath.Join(reportDir, "summary.json")
	hashesPath := filepath.Join(reportDir, "artifact-hashes.json")
	releaseStatePath := filepath.Join(artifactsDir, "release-state.json")
	archivedSignoffPath := filepath.Join(artifactsDir, "security-review.md")
	signoffPath := filepath.Join(dir, "security-review.md")

	writeFile(summaryPath, `{"schema":"tetra.release-gate-summary.v1","status":"pass"}`+"\n")
	writeFile(hashesPath, `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`+"\n")
	writeFile(releaseStatePath, `{"schema":"tetra.release-state.v1","status":"pass"}`+"\n")

	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: ` + reportDir + `
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- summary.json: sha256:0000000000000000000000000000000000000000000000000000000000000000
- artifact-hashes.json: ` + sha256ForTest(t, hashesPath) + `
- artifacts/release-state.json: ` + sha256ForTest(t, releaseStatePath) + `
- artifacts/security-review.md: sha256:2222222222222222222222222222222222222222222222222222222222222222

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	writeFile(signoffPath, raw)
	writeFile(archivedSignoffPath, raw)

	cmd := exec.Command("bash", "scripts/release/v0_3_0/security-review.sh", "--signoff", signoffPath)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v0.3.0 signoff with mismatched canonical summary artifact should fail\n%s", out)
	}
	if !strings.Contains(string(out), "canonical gate artifact summary.json hash mismatch") {
		t.Fatalf("validator should name mismatched canonical summary artifact:\n%s", out)
	}
}

func TestV0_3_0_security_signoff_rejects_missing_required_canonical_artifact_listing(t *testing.T) {
	dir := t.TempDir()
	reportDir := filepath.Join(dir, "report")
	artifactsDir := filepath.Join(reportDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	summaryPath := filepath.Join(reportDir, "summary.json")
	hashesPath := filepath.Join(reportDir, "artifact-hashes.json")
	releaseStatePath := filepath.Join(artifactsDir, "release-state.json")
	archivedSignoffPath := filepath.Join(artifactsDir, "security-review.md")
	signoffPath := filepath.Join(dir, "security-review.md")

	writeFile(summaryPath, `{"schema":"tetra.release-gate-summary.v1","status":"pass"}`+"\n")
	writeFile(hashesPath, `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`+"\n")
	writeFile(releaseStatePath, `{"schema":"tetra.release-state.v1","status":"pass"}`+"\n")

	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: ` + reportDir + `
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- summary.json: ` + sha256ForTest(t, summaryPath) + `
- artifact-hashes.json: ` + sha256ForTest(t, hashesPath) + `
- artifacts/security-review.md: sha256:2222222222222222222222222222222222222222222222222222222222222222

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	writeFile(signoffPath, raw)
	writeFile(archivedSignoffPath, raw)

	cmd := exec.Command("bash", "scripts/release/v0_3_0/security-review.sh", "--signoff", signoffPath)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v0.3.0 signoff missing canonical release-state artifact should fail\n%s", out)
	}
	if !strings.Contains(string(out), "missing required canonical gate artifact listing: release-state.json") {
		t.Fatalf("validator should name missing canonical release-state artifact:\n%s", out)
	}
}

func TestV0_3_0_security_signoff_canonical_hash_lookup_ignores_bullets_outside_artifact_hashes(t *testing.T) {
	dir := t.TempDir()
	reportDir := filepath.Join(dir, "report")
	artifactsDir := filepath.Join(reportDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	summaryPath := filepath.Join(reportDir, "summary.json")
	hashesPath := filepath.Join(reportDir, "artifact-hashes.json")
	releaseStatePath := filepath.Join(artifactsDir, "release-state.json")
	archivedSignoffPath := filepath.Join(artifactsDir, "security-review.md")
	signoffPath := filepath.Join(dir, "security-review.md")

	writeFile(summaryPath, `{"schema":"tetra.release-gate-summary.v1","status":"pass"}`+"\n")
	writeFile(hashesPath, `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`+"\n")
	writeFile(releaseStatePath, `{"schema":"tetra.release-state.v1","status":"pass"}`+"\n")

	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: ` + reportDir + `
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- artifact-hashes.json: ` + sha256ForTest(t, hashesPath) + `
- artifacts/release-state.json: ` + sha256ForTest(t, releaseStatePath) + `

## Archived Artifacts

- summary.json: ` + sha256ForTest(t, summaryPath) + `
- artifacts/security-review.md is copied from this signoff.

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	writeFile(signoffPath, raw)
	writeFile(archivedSignoffPath, raw)

	cmd := exec.Command("bash", "scripts/release/v0_3_0/security-review.sh", "--signoff", signoffPath)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v0.3.0 signoff must ignore canonical summary hash outside Artifact Hashes\n%s", out)
	}
	if !strings.Contains(string(out), "missing required canonical gate artifact hash: summary.json") {
		t.Fatalf("validator should require canonical summary hash in Artifact Hashes section:\n%s", out)
	}
}

func TestV0_3_0_security_signoff_rejects_listed_artifact_hash_mismatch(t *testing.T) {
	dir := t.TempDir()
	reportDir := filepath.Join(dir, "report")
	artifactsDir := filepath.Join(reportDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	summaryPath := filepath.Join(reportDir, "summary.json")
	hashesPath := filepath.Join(reportDir, "artifact-hashes.json")
	releaseStatePath := filepath.Join(artifactsDir, "release-state.json")
	extraAuditPath := filepath.Join(artifactsDir, "extra-audit.txt")
	archivedSignoffPath := filepath.Join(artifactsDir, "security-review.md")
	signoffPath := filepath.Join(dir, "security-review.md")

	writeFile(summaryPath, `{"schema":"tetra.release-gate-summary.v1","status":"pass"}`+"\n")
	writeFile(hashesPath, `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`+"\n")
	writeFile(releaseStatePath, `{"schema":"tetra.release-state.v1","status":"pass"}`+"\n")
	writeFile(extraAuditPath, "independent security audit evidence\n")

	head := currentGitHead(t)
	version := currentReleaseVersion(t)
	raw := `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: ` + reportDir + `
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- summary.json: ` + sha256ForTest(t, summaryPath) + `
- artifact-hashes.json: ` + sha256ForTest(t, hashesPath) + `
- artifacts/release-state.json: ` + sha256ForTest(t, releaseStatePath) + `
- artifacts/extra-audit.txt: sha256:0000000000000000000000000000000000000000000000000000000000000000

## Archived Artifacts

- artifacts/security-review.md is copied from this signoff.

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	writeFile(signoffPath, raw)
	writeFile(archivedSignoffPath, raw)

	cmd := exec.Command("bash", "scripts/release/v0_3_0/security-review.sh", "--signoff", signoffPath)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v0.3.0 signoff with mismatched listed artifact hash should fail\n%s", out)
	}
	if !strings.Contains(string(out), "listed artifact artifacts/extra-audit.txt hash mismatch") {
		t.Fatalf("validator should name mismatched listed artifact hash:\n%s", out)
	}
}

func currentGitHead(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func currentReleaseVersion(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "compiler", "internal", "version", "version.go"))
	if err != nil {
		t.Fatalf("read version.go: %v", err)
	}
	matches := regexp.MustCompile(`CompilerVersion = "([^"]+)"`).FindSubmatch(raw)
	if matches == nil {
		t.Fatalf("CompilerVersion not found in version.go")
	}
	return string(matches[1])
}

func sha256ForTest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

func releaseV040SecurityReviewPathFakeRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	head := "0123456789abcdef0123456789abcdef01234567"
	for _, dir := range []string{
		"bin",
		"compiler/internal/version",
		"scripts/release/v0_4_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_4_0", "security-review.sh"), filepath.Join(root, "scripts", "release", "v0_4_0", "security-review.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "compiler", "internal", "version", "version.go"), []byte("package version\n\nconst CompilerVersion = \"v0.4.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "HEAD" ]]; then
  echo "0123456789abcdef0123456789abcdef01234567"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, head
}

func releaseV10SecurityReviewPathFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		"bin",
		"compiler/internal/version",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "security-review.sh"), filepath.Join(root, "scripts", "release", "v1_0", "security-review.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "compiler", "internal", "version", "version.go"), []byte("package version\n\nconst CompilerVersion = \"v1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "HEAD" ]]; then
  echo "0123456789abcdef0123456789abcdef01234567"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func runReleaseV10SecurityReview(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/release/v1_0/security-review.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
}

func validReleaseV10SecuritySignoff(version, head string) string {
	return `# ` + version + ` Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for ` + version + ` release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release/v1_0/wasi-smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release/v1_0/web-smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111
- security-review.md: sha256:2222222222222222222222222222222222222222222222222222222222222222

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
}
