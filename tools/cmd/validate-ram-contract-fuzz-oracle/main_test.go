package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRAMContractFuzzOracleRejectsMissingMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	raw := `{
  "schema_version":"tetra.ram-contract-fuzz-oracle.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "generated_at":"2026-06-10T00:00:00Z",
  "observations":[
    {"mutation":"mutated_proof_id","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"},
    {"mutation":"widened_grade","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"},
    {"mutation":"other","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"},
    {"mutation":"budget_drift","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"},
    {"mutation":"artifact_hash_drift","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"},
    {"mutation":"forbidden_nonclaim_text","rejected":true,"validator":"validate-ram-contract-report","reason":"rejected"}
  ],
  "summary":{"mutations":6,"rejected":6},
  "non_claims":["not a full formal proof"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path)
	if err == nil || !strings.Contains(err.Error(), "missing_blocker") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want missing mutation", err)
	}
}
