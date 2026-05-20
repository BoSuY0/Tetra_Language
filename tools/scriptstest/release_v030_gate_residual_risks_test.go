package scriptstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateRejectsUntriagedUnstableSeeds(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky |  | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err == nil {
		t.Fatalf("expected untriaged unstable seed to block release gate\n%s", out)
	}
	for _, want := range []string{
		"unstable-seeds.md",
		"triage",
		"missing owner",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("untriaged seed output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030GateAcceptsTriagedUnstableSeeds(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err != nil {
		t.Fatalf("triaged unstable seed should not block release gate: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "release_v0_3_0_gate: passed") {
		t.Fatalf("triaged unstable seed run did not reach pass output:\n%s", out)
	}
}

func TestReleaseV030GateWritesResidualRisksJSONArtifact(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGate(t, root, reportDir)
	if err != nil {
		t.Fatalf("triaged residual risk run should pass: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "residual-risks.json"))
	if err != nil {
		t.Fatalf("read residual-risks.json: %v", err)
	}
	var artifact struct {
		Schema         string `json:"schema"`
		ReleaseVersion string `json:"release_version"`
		Risks          []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			Owner    string `json:"owner"`
			Status   string `json:"status"`
		} `json:"risks"`
	}
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("decode residual-risks.json: %v\n%s", err, raw)
	}
	if artifact.Schema != "tetra.release.residual-risks.v1" {
		t.Fatalf("schema = %q", artifact.Schema)
	}
	if artifact.ReleaseVersion != "v0.3.0" {
		t.Fatalf("release_version = %q", artifact.ReleaseVersion)
	}
	for _, risk := range artifact.Risks {
		if (risk.Severity == "high" || risk.Severity == "medium") && risk.Owner == "" {
			t.Fatalf("high/medium residual risk has no owner: %+v\n%s", risk, raw)
		}
	}
}

func TestReleaseV030GateAcceptsResidualRisksSourcePathStartingWithDash(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := "-x-residual-risks.json"
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "risks": []
}
`
	if err := os.WriteFile(filepath.Join(root, sourcePath), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "dash-residual-risks-report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err != nil {
		t.Fatalf("gate should accept dash-prefixed residual risks source path: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "artifacts", "residual-risks.json")); err != nil {
		t.Fatalf("residual risks artifact was not archived from dash-prefixed source: %v\n%s", err, out)
	}
}

func TestReleaseV030GateRejectsUnownedHighMediumResidualRisk(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := filepath.Join(root, "residual-risks.json")
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "risks": [
    {"id":"REL-011-test","severity":"medium","owner":"","status":"accepted","description":"missing owner should block"}
  ]
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err == nil {
		t.Fatalf("expected unowned medium residual risk to block release gate\n%s", out)
	}
	for _, want := range []string{
		"residual-risks.json",
		"medium",
		"owner",
		"REL-011-test",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("unowned residual risk output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030GateRejectsResidualRisksJSONForWrongReleaseVersion(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := filepath.Join(root, "residual-risks.json")
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.2.0",
  "risks": []
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err == nil {
		t.Fatalf("expected wrong residual-risks release_version to block release gate\n%s", out)
	}
	for _, want := range []string{
		"residual-risks.json",
		"release_version",
		"v0.3.0",
		"v0.2.0",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("wrong-release residual risk output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030GateRejectsMalformedResidualRisksJSON(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := filepath.Join(root, "residual-risks.json")
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "risks": [
    {"id":"REL-012-test","severity":"low","owner":"","status":"accepted",}
  ]
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err == nil {
		t.Fatalf("expected malformed residual-risks.json to block release gate\n%s", out)
	}
	for _, want := range []string{
		"residual-risks.json",
		"malformed JSON",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("malformed residual risk output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030GateRejectsNullResidualRisksArray(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := filepath.Join(root, "residual-risks.json")
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "risks": null
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err == nil {
		t.Fatalf("expected null residual-risks risks array to block release gate\n%s", out)
	}
	for _, want := range []string{
		"residual-risks.json",
		"risks array required",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("null residual risks output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030GateRejectsResidualRiskMissingRequiredFields(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		"| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky | release-a | go test ./compiler/... -run FuzzParser -count=1 |",
	})
	sourcePath := filepath.Join(root, "residual-risks.json")
	source := `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "risks": [
    {"id":"REL-013-test","severity":"low","owner":"release-a"}
  ]
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_RESIDUAL_RISKS_JSON=" + sourcePath,
	})
	if err == nil {
		t.Fatalf("expected residual risk with missing required fields to block release gate\n%s", out)
	}
	for _, want := range []string{
		"residual-risks.json",
		"required field status",
		"REL-013-test",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing-field residual risk output missing %q:\n%s", want, out)
		}
	}
}

func TestReleaseV030RunnableGateFiltersAmbientResidualRisksEnv(t *testing.T) {
	t.Setenv("TETRA_RESIDUAL_RISKS_JSON", "/tmp/ambient-residual-risks.json")

	env := filteredReleaseV030GateEnv()
	if envHasPrefix(env, "TETRA_RESIDUAL_RISKS_JSON=") {
		t.Fatalf("filteredReleaseV030GateEnv leaked ambient TETRA_RESIDUAL_RISKS_JSON into release gate test env")
	}
}
