package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresProofAndMemoryReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--proof") {
		t.Fatalf("run code/stderr = %d/%q, want --proof usage error", code, stderr.String())
	}
}

func TestRunValidatesProofFiles(t *testing.T) {
	dir := t.TempDir()
	proofPath := filepath.Join(dir, "proof.json")
	memoryPath := filepath.Join(dir, "memory-report.json")
	if err := os.WriteFile(proofPath, []byte(validCLIProofReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memoryPath, []byte(validCLIMemoryReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--proof", proofPath, "--memory-report", memoryPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "validated") {
		t.Fatalf("stdout = %q, want validated", stdout.String())
	}
}

func TestRunRejectsMismatchedProofFiles(t *testing.T) {
	dir := t.TempDir()
	proofPath := filepath.Join(dir, "proof.json")
	memoryPath := filepath.Join(dir, "memory-report.json")
	if err := os.WriteFile(proofPath, []byte(validCLIProofReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	memory := strings.Replace(
		validCLIMemoryReport(),
		`"island_id": "island:main:0"`,
		`"island_id": "island:other"`,
		1,
	)
	if err := os.WriteFile(memoryPath, []byte(memory), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--proof", proofPath, "--memory-report", memoryPath}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "island_id") {
		t.Fatalf("run code/stderr = %d/%q, want island_id validation error", code, stderr.String())
	}
}

func TestRunRejectsManifestMissingVerifierCommand(t *testing.T) {
	dir := t.TempDir()
	proofPath := filepath.Join(dir, "island-proof-verifier.json")
	memoryPath := filepath.Join(dir, "island-proof-memory-report.json")
	manifestPath := filepath.Join(dir, "memory-release-manifest.json")
	if err := os.WriteFile(proofPath, []byte(validCLIProofReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memoryPath, []byte(validCLIMemoryReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema": "tetra.memory.release-manifest.v1",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "commands": [],
  "artifacts": []
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		[]string{"--proof", proofPath, "--memory-report", memoryPath, "--manifest", manifestPath},
		&stdout,
		&stderr,
	)
	if code != 1 || !strings.Contains(stderr.String(), "island-proof-verifier") {
		t.Fatalf(
			"run code/stderr = %d/%q, want manifest verifier command rejection",
			code,
			stderr.String(),
		)
	}
}

func validCLIProofReport() string {
	return `{
  "schema": "tetra.island.proof.v1",
  "producer": "compiler/memoryfacts",
  "producer_command": "go run ./tools/cmd/validate-island-proof",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "proofs": [
    {
      "proof_id": "proof:island:borrow:1",
      "operation": "island_borrow",
      "proof_kind": "island_epoch",
      "subject_base_id": "alloc:main:0",
      "island_id": "island:main:0",
      "epoch": 1,
      "source_fact_id": "fact:island:proof:1",
      "claim": "island_proof_verified",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "dominance": "entry dominates island borrow",
      "distinct_live_islands": ["island:main:0", "island:other:0"]
    }
  ]
}`
}

func validCLIMemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "site_id": "island:main:1",
      "source_fact_id": "fact:island:proof:1",
      "claim": "island_proof_verified",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "alias_state": "unique",
      "island_id": "island:main:0",
      "epoch": 1,
      "base_id": "alloc:main:0",
      "proof_id": "proof:island:borrow:1",
      "proof_kind": "island_epoch",
      "proof_subject_base_id": "alloc:main:0",
      "proof_operation": "island_borrow",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "validator_name": "validate-island-proof",
      "validator_status": "pass"
    }
  ]
}`
}
