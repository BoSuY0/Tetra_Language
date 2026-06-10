package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRAMContractReportRejectsMissingBlocker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	raw := strings.Replace(validRAMContractReportForTest(), `"blockers":["unknown_size"]`, `"blockers":[]`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractReport(path)
	if err == nil || !strings.Contains(err.Error(), "blocker") {
		t.Fatalf("validateRAMContractReport error = %v, want blocker rejection", err)
	}
}

func TestValidateRAMContractReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	if err := os.WriteFile(path, []byte(validRAMContractReportForTest()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateRAMContractReport(path); err != nil {
		t.Fatalf("validateRAMContractReport: %v", err)
	}
}

func validRAMContractReportForTest() string {
	return `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{
    "site_id":"site:main:alloc0",
    "value_id":"alloc0",
    "function":"main",
    "intent":"heap_fallback",
    "requested_bytes":8192,
    "bounded":false,
    "owner":"function:main",
    "lifetime":"function:main",
    "escape_status":"unknown",
    "placement":"heap_unbounded",
    "proof_ids":[],
    "blockers":["unknown_size"],
    "contract_grade":"M5",
    "validation_status":"conservative"
  }],
  "proofs":[],
  "summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192},
  "non_claims":["not Memory 100%","not full formal proof","not a performance benchmark"]
}`
}
