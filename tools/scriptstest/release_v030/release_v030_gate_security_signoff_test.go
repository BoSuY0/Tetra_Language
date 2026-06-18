package release_v030

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateCIModeAllowsMissingSecuritySignoffWithArtifact(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	installReleaseV030SummaryEchoingGo(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1",
	})
	if err != nil {
		t.Fatalf("CI mode should not hard-block on missing security signoff: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "missing TETRA_SECURITY_REVIEW_SIGNOFF") {
		t.Fatalf(
			"CI mode should use explicit missing-signoff policy, not missing-env failure:\n%s",
			out,
		)
	}
	if strings.Contains(string(out), "release_v0_3_0_gate: passed") {
		t.Fatalf(
			"CI mode with missing security signoff must not look like a full evidence pass:\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"CI missing security signoff recorded; not a full release evidence pass",
	) {
		t.Fatalf("CI mode did not report limited evidence status:\n%s", out)
	}
	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read CI summary.json: %v", err)
	}
	var summary struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode CI summary.json: %v\n%s", err, summaryRaw)
	}
	if summary.Status != "blocked" {
		t.Fatalf(
			"CI missing-signoff summary status = %q, want blocked\n%s",
			summary.Status,
			summaryRaw,
		)
	}
	releaseStateRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "release-state.json"))
	if err != nil {
		t.Fatalf("read CI release-state.json: %v", err)
	}
	var releaseState struct {
		LastGateEvidence struct {
			Status string `json:"status"`
		} `json:"last_gate_evidence"`
	}
	if err := json.Unmarshal(releaseStateRaw, &releaseState); err != nil {
		t.Fatalf("decode CI release-state.json: %v\n%s", err, releaseStateRaw)
	}
	if releaseState.LastGateEvidence.Status != "blocked" {
		t.Fatalf(
			"CI release-state status = %q, want blocked\n%s",
			releaseState.LastGateEvidence.Status,
			releaseStateRaw,
		)
	}
	releaseStateText, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "release-state.txt"))
	if err != nil {
		t.Fatalf("read CI release-state.txt: %v", err)
	}
	if !strings.Contains(string(releaseStateText), "last gate evidence: blocked") {
		t.Fatalf(
			"CI release-state.txt does not record blocked gate evidence:\n%s",
			releaseStateText,
		)
	}
	artifactRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "security-review.md"))
	if err != nil {
		t.Fatalf("read CI security-review artifact: %v", err)
	}
	artifact := string(artifactRaw)
	for _, want := range []string{
		"# v0.3.0 Security Review CI Placeholder",
		"Decision: blocked: missing human security signoff",
		"CI status: missing-security-signoff",
		"not a full release evidence pass",
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1",
		"TETRA_SECURITY_REVIEW_SIGNOFF was not set",
	} {
		if !strings.Contains(artifact, want) {
			t.Fatalf("CI security-review artifact missing %q:\n%s", want, artifact)
		}
	}
}

func TestReleaseV030GateCIMissingSignoffWritesDetachedHashOutsideCanonicalManifest(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	installReleaseV030CanonicalArtifactGo(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1",
	})
	if err != nil {
		t.Fatalf(
			"CI missing-signoff gate should complete as blocked evidence with detached hash: %v\n%s",
			err,
			out,
		)
	}

	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read CI summary.json: %v", err)
	}
	var summary struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode CI summary.json: %v\n%s", err, summaryRaw)
	}
	if summary.Status != "blocked" {
		t.Fatalf(
			"CI missing-signoff summary status = %q, want blocked\n%s",
			summary.Status,
			summaryRaw,
		)
	}

	securityReviewPath := filepath.Join(reportDir, "artifacts", "security-review.md")
	detachedHashPath := filepath.Join(reportDir, "artifacts", "security-review.md.sha256")
	detachedRaw, err := os.ReadFile(detachedHashPath)
	if err != nil {
		t.Fatalf("read CI detached security-review hash: %v", err)
	}
	wantDetached := sha256ForTest(t, securityReviewPath) + "  artifacts/security-review.md\n"
	if string(detachedRaw) != wantDetached {
		t.Fatalf(
			"CI detached security-review hash mismatch:\nwant %q\ngot  %q",
			wantDetached,
			detachedRaw,
		)
	}

	hashRaw, err := os.ReadFile(filepath.Join(reportDir, "artifact-hashes.json"))
	if err != nil {
		t.Fatalf("read artifact-hashes.json: %v", err)
	}
	var manifest struct {
		Artifacts []struct {
			Path string `json:"path"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(hashRaw, &manifest); err != nil {
		t.Fatalf("decode artifact-hashes.json: %v\n%s", err, hashRaw)
	}
	for _, artifact := range manifest.Artifacts {
		switch artifact.Path {
		case "artifacts/security-review.md", "artifacts/security-review.md.sha256":
			t.Fatalf(
				"artifact-hashes.json must exclude %s for blocked CI missing-signoff evidence:\n%s",
				artifact.Path,
				hashRaw,
			)
		}
	}
}

func TestReleaseV030GateRequiresSecuritySignoffOutsideCIMode(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, nil)
	if err == nil {
		t.Fatalf("non-CI gate should require security signoff\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"missing TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>",
	) {
		t.Fatalf("non-CI gate did not report missing signoff:\n%s", out)
	}
}

func TestReleaseV030GateRequireCleanRequiresSecuritySignoffEvenInCIMode(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	reportDir := filepath.Join(root, "report")
	cmd := exec.Command(
		"bash",
		"scripts/release/v0_3_0/gate.sh",
		"--require-clean",
		"--report-dir",
		reportDir,
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf(
			"tag-ready gate should require security signoff even when CI mode env is set\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"missing TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>",
	) {
		t.Fatalf("tag-ready gate did not report missing signoff:\n%s", out)
	}
}
