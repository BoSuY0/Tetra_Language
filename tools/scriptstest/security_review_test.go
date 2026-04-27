package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityReviewSignoffValidatorAcceptsAuditableSignoff(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	raw := `# v0.1.1 Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for v0.1.1 release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release_v1_0_wasi_smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release_v1_0_web_smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111
- security-review.md: sha256:2222222222222222222222222222222222222222222222222222222222222222

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release_v1_0_security_review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("security review signoff should validate: %v\n%s", err, out)
	}
}

func TestSecurityReviewSignoffValidatorRejectsTemplatePlaceholders(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	cmd := exec.Command("bash", "scripts/release_v1_0_security_review.sh", "--write-template", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("write template failed: %v\n%s", err, out)
	}

	cmd = exec.Command("bash", "scripts/release_v1_0_security_review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("placeholder signoff should fail validation\n%s", out)
	}
	if !strings.Contains(string(out), "placeholder") {
		t.Fatalf("validator should explain placeholder failure:\n%s", out)
	}
}

func TestSecurityReviewSignoffValidatorRequiresPrivacyAndWASMEvidence(t *testing.T) {
	dir := t.TempDir()
	signoff := filepath.Join(dir, "security-review.md")
	head := currentGitHead(t)
	raw := `# v0.1.1 Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for v0.1.1 release

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

	cmd := exec.Command("bash", "scripts/release_v1_0_security_review.sh", "--signoff", signoff)
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
	raw := `# v0.1.1 Security Review Signoff

Reviewer: Release Reviewer <security@example.invalid>
Reviewed commit: ` + head + `
Report directory: /tmp/tetra-v1-rc-security
Decision: approved for v0.1.1 release

## Evidence Commands

- ` + "`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass" + `
- ` + "`go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: pass" + `
- ` + "`go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: pass" + `
- ` + "`go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: pass" + `
- ` + "`bash scripts/release_v1_0_wasi_smoke.sh --report <path>`: pass" + `
- ` + "`bash scripts/release_v1_0_web_smoke.sh --report <path>`: pass" + `

## Artifact Hashes

- release_gate_summary.json: sha256:not-a-real-hash

## Residual Risks

- None beyond the documented beta/post-v1 Eco trust surfaces.
`
	if err := os.WriteFile(signoff, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release_v1_0_security_review.sh", "--signoff", signoff)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("signoff with invalid artifact hash should fail\n%s", out)
	}
	if !strings.Contains(string(out), "artifact hash") {
		t.Fatalf("validator should name artifact hash failure:\n%s", out)
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
