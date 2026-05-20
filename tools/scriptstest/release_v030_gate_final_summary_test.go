package scriptstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateBlocksFinalSummaryWhenPostSummaryArtifactHashCheckFails(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	installReleaseV030FailingFinalArtifactHashGo(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err == nil {
		t.Fatalf("expected final artifact-hashes check to block release gate\n%s", out)
	}
	if !strings.Contains(string(out), "final release artifact hash validation failed") {
		t.Fatalf("final artifact hash failure output missing blocked reason:\n%s", out)
	}

	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read final summary.json: %v", err)
	}
	var summary struct {
		Status      string `json:"status"`
		FailedCount int    `json:"failed_count"`
		Steps       []struct {
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode final summary.json: %v\n%s", err, summaryRaw)
	}
	if summary.Status != "blocked" {
		t.Fatalf("post-summary artifact hash failure left summary status = %q, want blocked\n%s", summary.Status, summaryRaw)
	}
	failedSteps := countReleaseGateFailedSteps(summary.Steps)
	if summary.FailedCount != failedSteps || failedSteps == 0 {
		t.Fatalf("post-summary artifact hash failure summary has failed_count=%d but %d failed step(s), want coherent nonzero failure evidence\n%s", summary.FailedCount, failedSteps, summaryRaw)
	}
}

func TestReleaseV030GateBlocksFinalSummaryWhenDetachedSecurityHashFails(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	installReleaseV030CanonicalArtifactGo(t, root)
	installReleaseV030FailingSecurityReviewSha256(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err == nil {
		t.Fatalf("expected detached security hash failure to block release gate\n%s", out)
	}
	if !strings.Contains(string(out), "detached security review hash generation failed") {
		t.Fatalf("detached security hash failure output missing blocked reason:\n%s", out)
	}

	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read final summary.json: %v", err)
	}
	var summary struct {
		Status      string `json:"status"`
		FailedCount int    `json:"failed_count"`
		Steps       []struct {
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode final summary.json: %v\n%s", err, summaryRaw)
	}
	if summary.Status != "blocked" {
		t.Fatalf("detached hash failure left summary status = %q, want blocked\n%s", summary.Status, summaryRaw)
	}
	failedSteps := countReleaseGateFailedSteps(summary.Steps)
	if summary.FailedCount != failedSteps || failedSteps == 0 {
		t.Fatalf("detached hash failure summary has failed_count=%d but %d failed step(s), want coherent nonzero failure evidence\n%s", summary.FailedCount, failedSteps, summaryRaw)
	}
}

func TestReleaseV030GateRecordsCIMissingSignoffFinalArtifactHashRefreshFailure(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	installReleaseV030CIMissingSignoffFailingFinalArtifactHashGo(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1",
	})
	if err == nil {
		t.Fatalf("expected CI missing-signoff final artifact-hashes refresh failure to block release gate\n%s", out)
	}
	if !strings.Contains(string(out), "CI missing-security-signoff artifact hash refresh failed") {
		t.Fatalf("CI missing-signoff final artifact hash failure output missing blocked reason:\n%s", out)
	}

	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read CI final summary.json: %v", err)
	}
	var summary struct {
		Status      string `json:"status"`
		FailedCount int    `json:"failed_count"`
		Steps       []struct {
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode CI final summary.json: %v\n%s", err, summaryRaw)
	}
	if summary.Status != "blocked" {
		t.Fatalf("CI final artifact hash failure left summary status = %q, want blocked\n%s", summary.Status, summaryRaw)
	}
	failedSteps := countReleaseGateFailedSteps(summary.Steps)
	if summary.FailedCount != failedSteps || failedSteps == 0 {
		t.Fatalf("CI final artifact hash failure summary has failed_count=%d but %d failed step(s), want coherent nonzero failure evidence\n%s", summary.FailedCount, failedSteps, summaryRaw)
	}
}

func TestReleaseV030GateCanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	installReleaseV030CanonicalArtifactGo(t, root)
	python3Marker := installReleaseV030PortablePythonCanonicalizers(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err != nil {
		t.Fatalf("release gate should use python3 when bare python is unavailable: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "bare python should not be required") {
		t.Fatalf("release gate invoked bare python during artifact manifest canonicalization:\n%s", out)
	}

	markerRaw, err := os.ReadFile(python3Marker)
	if err != nil {
		t.Fatalf("read python3 canonicalizer marker: %v", err)
	}
	if !strings.Contains(string(markerRaw), filepath.Join(reportDir, "artifact-hashes.json")) {
		t.Fatalf("python3 canonicalizer marker does not record artifact-hashes.json:\n%s", markerRaw)
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
			t.Fatalf("python3 canonicalizer left cycle-prone artifact %s in manifest:\n%s", artifact.Path, hashRaw)
		}
	}
}

func TestReleaseV030GateRefreshesReleaseStateAfterFinalSummary(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	installReleaseV030SummaryEchoingGo(t, root)

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err != nil {
		t.Fatalf("triaged unstable seed should not block release gate: %v\n%s", err, out)
	}

	var summary struct {
		Status      string `json:"status"`
		StepCount   int    `json:"step_count"`
		FailedCount int    `json:"failed_count"`
	}
	summaryRaw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read final summary.json: %v", err)
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode final summary.json: %v\n%s", err, summaryRaw)
	}

	var releaseState struct {
		LastGateEvidence struct {
			Status      string `json:"status"`
			StepCount   int    `json:"step_count"`
			FailedCount int    `json:"failed_count"`
		} `json:"last_gate_evidence"`
	}
	releaseStateRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "release-state.json"))
	if err != nil {
		t.Fatalf("read release-state.json: %v", err)
	}
	if err := json.Unmarshal(releaseStateRaw, &releaseState); err != nil {
		t.Fatalf("decode release-state.json: %v\n%s", err, releaseStateRaw)
	}

	if releaseState.LastGateEvidence.Status != summary.Status {
		t.Fatalf("release-state status %q contradicts final summary status %q", releaseState.LastGateEvidence.Status, summary.Status)
	}
	if releaseState.LastGateEvidence.StepCount != summary.StepCount {
		t.Fatalf("release-state step_count %d contradicts final summary step_count %d", releaseState.LastGateEvidence.StepCount, summary.StepCount)
	}
	if releaseState.LastGateEvidence.FailedCount != summary.FailedCount {
		t.Fatalf("release-state failed_count %d contradicts final summary failed_count %d", releaseState.LastGateEvidence.FailedCount, summary.FailedCount)
	}

	releaseStateText, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "release-state.txt"))
	if err != nil {
		t.Fatalf("read release-state.txt: %v", err)
	}
	wantText := "last gate evidence: pass (0 failed of 15 steps"
	if !strings.Contains(string(releaseStateText), wantText) {
		t.Fatalf("release-state.txt does not reflect final summary step count %q:\n%s", wantText, releaseStateText)
	}
}

func countReleaseGateFailedSteps(steps []struct {
	Status string `json:"status"`
}) int {
	failed := 0
	for _, step := range steps {
		if step.Status == "fail" {
			failed++
		}
	}
	return failed
}
