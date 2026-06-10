package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceProdAuditAcceptsCleanProdStableScopedAudit(t *testing.T) {
	audit := validSurfaceProdAudit("PROD_STABLE_SCOPED")
	if err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567"); err != nil {
		t.Fatalf("validateAuditMarkdown failed: %v\n%s", err, audit)
	}
}

func TestValidateSurfaceProdAuditAcceptsCurrentHeadSentinel(t *testing.T) {
	audit := strings.ReplaceAll(validSurfaceProdAudit("PROD_STABLE_SCOPED"), "0123456789abcdef0123456789abcdef01234567", "CURRENT_HEAD")
	if err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567"); err != nil {
		t.Fatalf("validateAuditMarkdown failed: %v\n%s", err, audit)
	}
}

func TestValidateSurfaceProdAuditRejectsDirtyCheckoutPromoted(t *testing.T) {
	audit := strings.Replace(validSurfaceProdAudit("PROD_STABLE_SCOPED"), `"git_dirty": false`, `"git_dirty": true`, 1)
	audit = strings.Replace(audit, `"clean_checkout": true`, `"clean_checkout": false`, 1)
	err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567")
	requireAuditIssue(t, err, "dirty checkout")
}

func TestValidateSurfaceProdAuditRejectsReportFromDifferentGitHead(t *testing.T) {
	audit := strings.Replace(validSurfaceProdAudit("PROD_STABLE_SCOPED"),
		`"git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"`,
		`"git_head": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"`,
		1)
	err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567")
	requireAuditIssue(t, err, "same commit")
}

func TestValidateSurfaceProdAuditRejectsMissingUnsupportedTargetNonclaims(t *testing.T) {
	audit := strings.Replace(validSurfaceProdAudit("PROD_STABLE_SCOPED"),
		`"unsupported_target_nonclaims": ["windows-x64", "macos-x64", "wasm32-wasi", "GPU production", "full accessibility parity", "broad Electron replacement"]`,
		`"unsupported_target_nonclaims": ["windows-x64"]`,
		1)
	err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567")
	requireAuditIssue(t, err, "macos-x64")
}

func TestValidateSurfaceProdAuditRejectsExpectedStatusMismatch(t *testing.T) {
	audit := validSurfaceProdAudit("NEAR_READY_WITH_BLOCKERS")
	err := validateAuditMarkdown([]byte(audit), "PROD_STABLE_SCOPED", "0123456789abcdef0123456789abcdef01234567")
	requireAuditIssue(t, err, "expected status")
}

func TestValidateSurfaceProdAuditCLIReadsMarkdownAudit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "surface_prod_release_audit.md")
	if err := os.WriteFile(path, []byte(validSurfaceProdAudit("PROD_STABLE_SCOPED")), 0o644); err != nil {
		t.Fatalf("write audit: %v", err)
	}
	if err := run(validateAuditOptions{
		AuditPath:      path,
		ExpectedStatus: "PROD_STABLE_SCOPED",
		CurrentGitHead: "0123456789abcdef0123456789abcdef01234567",
	}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func requireAuditIssue(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected issue containing %q, got nil", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func validSurfaceProdAudit(verdict string) string {
	blockers := `[]`
	commandStatus := `"pass"`
	if verdict != "PROD_STABLE_SCOPED" {
		blockers = `["working tree is not clean in the current implementation branch"]`
		commandStatus = `"blocked"`
	}
	return `# Surface Production Release Audit

Status: ` + verdict + `

` + "```json surface-prod-audit" + `
{
  "schema": "tetra.surface.prod-audit.v1",
  "verdict": "` + verdict + `",
  "level": "surface-prod-final-same-commit-audit-v1",
  "scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "release_scope": "surface-v1-linux-web",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "git_dirty": false,
  "clean_checkout": true,
  "generated_at_utc": "2026-06-10T18:00:00Z",
  "blockers": ` + blockers + `,
  "commands": [
    {"name": "git-head", "command": "git rev-parse HEAD", "status": ` + commandStatus + `},
    {"name": "git-status", "command": "git status --short", "status": ` + commandStatus + `},
    {"name": "full-go-test", "command": "go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1", "status": ` + commandStatus + `},
    {"name": "ci-test", "command": "bash scripts/ci/test.sh", "status": ` + commandStatus + `},
    {"name": "prod-gate", "command": "bash scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final/prod", "status": ` + commandStatus + `},
    {"name": "validate-manifest", "command": "go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json", "status": ` + commandStatus + `},
    {"name": "verify-docs", "command": "go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json", "status": ` + commandStatus + `},
    {"name": "validate-prod-audit", "command": "go run -buildvcs=false ./tools/cmd/validate-surface-prod-audit --audit docs/release/surface_prod_release_audit.md --expected-status PROD_STABLE_SCOPED", "status": ` + commandStatus + `},
    {"name": "validate-artifact-hashes", "command": "go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest reports/surface-prod/final/artifact-hashes.json", "status": ` + commandStatus + `},
    {"name": "diff-check", "command": "git diff --check", "status": ` + commandStatus + `},
    {"name": "manifest-clean", "command": "git diff --exit-code -- docs/generated/manifest.json", "status": ` + commandStatus + `}
  ],
  "reports": [
    {"name": "surface-prod-gate", "path": "reports/surface-prod/final/prod/surface-release-v1/surface-prod-gate-report.json", "schema": "tetra.surface.prod-gate-report.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "block-system", "path": "reports/surface-prod/final/block-system/surface-block-system-gate-summary.json", "schema": "tetra.surface.block-system.gate.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "morph", "path": "reports/surface-prod/final/morph/surface-morph-gate-summary.json", "schema": "tetra.surface.morph.gate.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "surface-release", "path": "reports/surface-prod/final/surface-v1/surface-release-summary.json", "schema": "tetra.surface.release.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "visual", "path": "reports/surface-prod/final/visual/surface-visual-report.json", "schema": "tetra.surface.visual-regression.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "perf", "path": "reports/surface-prod/final/perf/surface-perf-report.json", "schema": "tetra.surface.perf-report.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "security", "path": "reports/surface-prod/final/security/surface-security-report.json", "schema": "tetra.surface.security-report.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "package", "path": "reports/surface-prod/final/package/surface-package-report.json", "schema": "tetra.surface.package-report.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "safe-view-lifetime", "path": "reports/surface-prod/final/safe-view-lifetime/safe-view-lifetime-summary.json", "schema": "tetra.safe-view-lifetime.gate.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "api-stability", "path": "reports/surface-prod/final/api-stability/surface-api-stability-summary.json", "schema": "tetra.surface.api-stability.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "electron-comparison", "path": "reports/surface-prod/final/prod/surface-electron-comparison/surface-electron-comparison-report.json", "schema": "tetra.surface.electron-comparison-report.v1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "artifact-hashes", "path": "reports/surface-prod/final/artifact-hashes.json", "schema": "tetra.release-artifact-hashes.v1alpha1", "git_head": "0123456789abcdef0123456789abcdef01234567", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"}
  ],
  "target_host_evidence": [
    {"target": "headless", "tier": "release-evidence", "evidence": "headless deterministic", "production": false, "unsupported_nonclaim": false},
    {"target": "linux-x64", "tier": "prod", "evidence": "linux-x64 real-window", "production": true, "unsupported_nonclaim": false},
    {"target": "wasm32-web", "tier": "prod", "evidence": "wasm32-web browser-canvas", "production": true, "unsupported_nonclaim": false},
    {"target": "windows-x64", "tier": "beta", "evidence": "unsupported target-host boundary", "production": false, "unsupported_nonclaim": true},
    {"target": "macos-x64", "tier": "beta", "evidence": "unsupported target-host boundary", "production": false, "unsupported_nonclaim": true},
    {"target": "wasm32-wasi", "tier": "unsupported", "evidence": "unsupported UI target", "production": false, "unsupported_nonclaim": true}
  ],
  "claim_governance": {
    "public_claim_source": "docs/release/surface_prod_release_audit.md",
    "prod_claim_validator": "tools/cmd/validate-surface-prod-claim",
    "final_audit_validator": "tools/cmd/validate-surface-prod-audit",
    "fake_claim_rejections": ["fake electron/react/css replacement rejected", "fake cross-platform support rejected", "fake gpu production claim rejected", "fake full accessibility parity rejected", "missing target-host evidence rejected"],
    "unsupported_target_nonclaims": ["windows-x64", "macos-x64", "wasm32-wasi", "GPU production", "full accessibility parity", "broad Electron replacement"]
  }
}
` + "```" + `
`
}
