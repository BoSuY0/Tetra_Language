package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProofStoreSummaryRejectsCountDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proof-store-summary.json")
	raw := `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[],
  "summary":{"proof_count":1,"proven":0,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateProofStoreSummary(path)
	if err == nil || !strings.Contains(err.Error(), "proof_count") {
		t.Fatalf("validateProofStoreSummary error = %v, want proof_count rejection", err)
	}
}

func TestValidateProofStoreSummaryRejectsDuplicateProofID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proof-store-summary.json")
	raw := `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[
    {"proof_id":"proof:1","kind":"allocation_placement","subject":"main/alloc0","stable_hash":"sha256:a","status":"proven"},
    {"proof_id":"proof:1","kind":"allocation_placement","subject":"main/alloc1","stable_hash":"sha256:b","status":"proven"}
  ],
  "summary":{"proof_count":2,"proven":2,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateProofStoreSummary(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate proof_id") {
		t.Fatalf("validateProofStoreSummary error = %v, want duplicate proof rejection", err)
	}
}

func TestValidateProofStoreSummaryRejectsMissingKindSubjectHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proof-store-summary.json")
	raw := `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[{"proof_id":"proof:1","kind":"","subject":"","stable_hash":"","status":"proven"}],
  "summary":{"proof_count":1,"proven":1,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateProofStoreSummary(path)
	if err == nil || !strings.Contains(err.Error(), "kind, subject, and stable_hash") {
		t.Fatalf("validateProofStoreSummary error = %v, want missing proof fields rejection", err)
	}
}

func TestValidateProofStoreSummaryRejectsUnknownStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proof-store-summary.json")
	raw := `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[{"proof_id":"proof:1","kind":"allocation_placement","subject":"main/alloc0","stable_hash":"sha256:a","status":"unsafe_unknown"}],
  "summary":{"proof_count":1,"proven":0,"conservative":0,"rejected":0,"unknown":1},
  "non_claims":["no full formal proof claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateProofStoreSummary(path)
	if err == nil || !strings.Contains(err.Error(), "unknown status") {
		t.Fatalf("validateProofStoreSummary error = %v, want unknown status rejection", err)
	}
}
