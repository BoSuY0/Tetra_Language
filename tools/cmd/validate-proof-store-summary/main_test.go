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
