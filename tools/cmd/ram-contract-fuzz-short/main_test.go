package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRAMContractFuzzShortWritesValidatedArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := runRAMContractFuzzShort(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893"); err != nil {
		t.Fatalf("runRAMContractFuzzShort: %v", err)
	}
	for _, name := range []string{
		"ram-contract-report.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"validation-pipeline-coverage.json",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-oracle.json",
		"ram-contract-fuzz-summary.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestRunRAMContractFuzzShortRejectsStaleReportDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stale.txt"), []byte("old evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runRAMContractFuzzShort(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "non-empty report directory") {
		t.Fatalf("runRAMContractFuzzShort error = %v, want non-empty report directory rejection", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ram-contract-fuzz-oracle.json")); !os.IsNotExist(err) {
		t.Fatalf("ram-contract-fuzz-oracle.json was written despite stale directory, stat err = %v", err)
	}
}

func TestRunRAMContractFuzzShortMutatesProofIDAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "mutated_proof_id")
}

func TestRunRAMContractFuzzShortMutatesGradeAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "widened_grade")
}

func TestRunRAMContractFuzzShortMutatesMissingBlockerAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "missing_blocker")
}

func TestRunRAMContractFuzzShortMutatesBudgetDriftAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "budget_drift")
}

func TestRunRAMContractFuzzShortMutatesArtifactHashAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "artifact_hash_drift")
}

func TestRunRAMContractFuzzShortMutatesForbiddenClaimAndCapturesFailure(t *testing.T) {
	requireMutationExitEvidence(t, "forbidden_nonclaim_text")
}

func requireMutationExitEvidence(t *testing.T, mutation string) {
	t.Helper()
	dir := t.TempDir()
	if err := runRAMContractFuzzShort(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893"); err != nil {
		t.Fatalf("runRAMContractFuzzShort: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "ram-contract-fuzz-oracle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report oracleReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	for _, obs := range report.Observations {
		if obs.Mutation != mutation {
			continue
		}
		if !obs.Rejected {
			t.Fatalf("mutation %s rejected = false", mutation)
		}
		if obs.ExitCode == 0 {
			t.Fatalf("mutation %s exit_code = 0, want validator failure evidence", mutation)
		}
		if strings.TrimSpace(obs.ValidatorCommand) == "" {
			t.Fatalf("mutation %s missing validator_command", mutation)
		}
		if strings.TrimSpace(obs.OutputExcerpt) == "" {
			t.Fatalf("mutation %s missing output_excerpt", mutation)
		}
		if !strings.Contains(filepath.ToSlash(obs.MutatedFile), "mutations/"+mutation+"/") {
			t.Fatalf("mutation %s mutated_file = %q, want mutation fixture path", mutation, obs.MutatedFile)
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(obs.MutatedFile))); err != nil {
			t.Fatalf("mutation %s mutated_file does not exist: %v", mutation, err)
		}
		return
	}
	t.Fatalf("mutation %s not found in oracle", mutation)
}
