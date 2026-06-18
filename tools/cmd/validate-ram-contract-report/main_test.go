package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRAMContractReportRejectsMissingBlocker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	raw := strings.Replace(
		validRAMContractReportForTest(),
		`"blockers":["unknown_size"]`,
		`"blockers":[]`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractReport(path)
	if err == nil || !strings.Contains(err.Error(), "blocker") {
		t.Fatalf("validateRAMContractReport error = %v, want blocker rejection", err)
	}
}

func TestValidateRAMContractReportFileAcceptsCompilerReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	if err := os.WriteFile(path, []byte(validRAMContractReportForTest()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateRAMContractReport(path); err != nil {
		t.Fatalf("validateRAMContractReport: %v", err)
	}
}

func TestValidateRAMContractReportRejectsTrustedPlacementWithoutNoEscapeValidation(t *testing.T) {
	tests := []struct {
		name        string
		replacement string
	}{
		{name: "escaped_stack", replacement: `"escape_status":"escapes_return"`},
		{name: "conservative_stack", replacement: `"validation_status":"conservative"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ram-contract.json")
			raw := strings.Replace(
				validTrustedRAMContractReportForTest(),
				test.replacement[:strings.Index(test.replacement, ":")+1]+`"no_escape"`,
				test.replacement,
				1,
			)
			if strings.Contains(test.replacement, "validation_status") {
				raw = strings.Replace(
					validTrustedRAMContractReportForTest(),
					`"validation_status":"validated"`,
					test.replacement,
					1,
				)
			}
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateRAMContractReport(path)
			if err == nil || !strings.Contains(err.Error(), "trusted placement") {
				t.Fatalf(
					"validateRAMContractReport error = %v, want trusted placement no-escape proof rejection",
					err,
				)
			}
		})
	}
}

func TestValidateRAMContractReportRejectsRegionPlacementWithoutScopedProof(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	if err := os.WriteFile(
		path,
		[]byte(validRegionRAMContractReportWithProofKindForTest("allocation_placement")),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractReport(path)
	if err == nil || !strings.Contains(err.Error(), "scoped proof") {
		t.Fatalf("validateRAMContractReport error = %v, want scoped proof rejection", err)
	}
}

func TestValidateRAMContractReportRejectsForbiddenNonclaimText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	raw := strings.Replace(
		validRAMContractReportForTest(),
		`"non_claims":["not Memory 100%","not full formal proof","not a performance benchmark"]`,
		`"non_claims":["Memory 100%"]`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractReport(path)
	if err == nil || !strings.Contains(err.Error(), "forbidden broad claim") {
		t.Fatalf("validateRAMContractReport error = %v, want forbidden broad claim rejection", err)
	}
}

func TestValidateRAMContractReportAllowsNegatedNonclaimText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ram-contract.json")
	raw := strings.Replace(
		validRAMContractReportForTest(),
		`"non_claims":["not Memory 100%","not full formal proof","not a performance benchmark"]`,
		`"non_claims":["no Memory 100% claim","not a full formal proof","does not claim zero heap for all programs"]`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateRAMContractReport(path); err != nil {
		t.Fatalf("validateRAMContractReport rejected negated nonclaims: %v", err)
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

func validTrustedRAMContractReportForTest() string {
	return `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{
    "site_id":"site:main:alloc0",
    "value_id":"alloc0",
    "function":"main",
    "intent":"allocation",
    "requested_bytes":16,
    "bounded":true,
    "owner":"function:main",
    "lifetime":"function:main",
    "escape_status":"no_escape",
    "placement":"stack",
    "proof_ids":["proof:ram:main:alloc0"],
    "blockers":[],
    "contract_grade":"M1",
    "validation_status":"validated"
  }],
  "proofs":[{"proof_id":"proof:ram:main:alloc0","kind":"allocation_placement","subject":"main/alloc0","stable_hash":"sha256:test","status":"proven"}],
  "summary":{"row_count":1,"artifact_grade":"M1","heap_rows":0,"copy_rows":0,"unbounded_rows":0,"budget_bytes":16},
  "non_claims":["not Memory 100%","not full formal proof","not a performance benchmark"]
}`
}

func validRegionRAMContractReportWithProofKindForTest(kind string) string {
	return `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{
    "site_id":"site:main:region",
    "value_id":"region",
    "function":"main",
    "intent":"region_alloc",
    "requested_bytes":16,
    "bounded":true,
    "owner":"function:main",
    "lifetime":"region:main:temp",
    "escape_status":"no_escape",
    "placement":"region",
    "proof_ids":["proof:ram:main:region"],
    "blockers":[],
    "contract_grade":"M3",
    "validation_status":"validated"
  }],
  "proofs":[{"proof_id":"proof:ram:main:region","kind":"` + kind + `","subject":"main/region","stable_hash":"sha256:test","status":"proven"}],
  "summary":{"row_count":1,"artifact_grade":"M3","heap_rows":0,"copy_rows":0,"unbounded_rows":0,"budget_bytes":16},
  "non_claims":["not Memory 100%","not full formal proof","not a performance benchmark"]
}`
}
