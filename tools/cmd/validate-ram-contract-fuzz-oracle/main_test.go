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
    {"mutation":"mutated_proof_id","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/mutated_proof_id/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/mutated_proof_id/ram-contract-report.json","reason":"rejected"},
    {"mutation":"widened_grade","rejected":true,"validator":"validate-memory-grade-report","validator_command":"go run ./tools/cmd/validate-memory-grade-report --report mutations/widened_grade/memory-grade-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/widened_grade/memory-grade-report.json","reason":"rejected"},
    {"mutation":"other","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/other/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/other/ram-contract-report.json","reason":"rejected"},
    {"mutation":"budget_drift","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/budget_drift/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/budget_drift/ram-contract-report.json","reason":"rejected"},
    {"mutation":"artifact_hash_drift","rejected":true,"validator":"validate-artifact-hashes","validator_command":"go run ./tools/cmd/validate-artifact-hashes --manifest mutations/artifact_hash_drift/artifact-hashes.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/artifact_hash_drift/ram-contract-fuzz-summary.md","reason":"rejected"},
    {"mutation":"forbidden_nonclaim_text","rejected":true,"validator":"validate-ram-contract-fuzz-oracle","validator_command":"go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","reason":"rejected"}
  ],
  "summary":{"mutations":6,"rejected":6},
  "non_claims":["not a full formal proof"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, "")
	if err == nil || !strings.Contains(err.Error(), "missing_blocker") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want missing mutation", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsObservationWithoutExitEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	raw := `{
  "schema_version":"tetra.ram-contract-fuzz-oracle.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "generated_at":"2026-06-10T00:00:00Z",
  "observations":[
    {"mutation":"mutated_proof_id","rejected":true,"validator":"validate-ram-contract-report","reason":"self-asserted"},
    {"mutation":"widened_grade","rejected":true,"validator":"validate-memory-grade-report","validator_command":"go run ./tools/cmd/validate-memory-grade-report --report mutations/widened_grade/memory-grade-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/widened_grade/memory-grade-report.json","reason":"rejected"},
    {"mutation":"missing_blocker","rejected":true,"validator":"validate-heap-blockers","validator_command":"go run ./tools/cmd/validate-heap-blockers --report mutations/missing_blocker/heap-blockers.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/missing_blocker/heap-blockers.json","reason":"rejected"},
    {"mutation":"budget_drift","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/budget_drift/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/budget_drift/ram-contract-report.json","reason":"rejected"},
    {"mutation":"artifact_hash_drift","rejected":true,"validator":"validate-artifact-hashes","validator_command":"go run ./tools/cmd/validate-artifact-hashes --manifest mutations/artifact_hash_drift/artifact-hashes.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/artifact_hash_drift/ram-contract-fuzz-summary.md","reason":"rejected"},
    {"mutation":"forbidden_nonclaim_text","rejected":true,"validator":"validate-ram-contract-fuzz-oracle","validator_command":"go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","reason":"rejected"}
  ],
  "summary":{"mutations":6,"rejected":6},
  "non_claims":["not a full formal proof"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, "")
	if err == nil || !strings.Contains(err.Error(), "exit evidence") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want missing exit evidence rejection", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsForbiddenClaimText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	raw := strings.Replace(validRAMContractFuzzOracleForTest(), `"non_claims":["not a full formal proof"]`, `"non_claims":["Memory 100%"]`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, "")
	if err == nil || !strings.Contains(err.Error(), "forbidden broad claim") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want forbidden broad claim rejection", err)
	}
}

func TestValidateRAMContractFuzzOracleAcceptsArtifactBundle(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	if err := validateRAMContractFuzzOracle(path, dir); err != nil {
		t.Fatalf("validateRAMContractFuzzOracle: %v", err)
	}
}

func TestValidateRAMContractFuzzOracleAcceptsCurrentGitHead(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	if err := validateRAMContractFuzzOracleWithHead(path, "e2c19b8ee276158f8eb2c54cf61e11bd84952893", dir); err != nil {
		t.Fatalf("validateRAMContractFuzzOracleWithHead: %v", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsMismatchedCurrentGitHead(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	err := validateRAMContractFuzzOracleWithHead(path, "ffffffffffffffffffffffffffffffffffffffff", dir)
	if err == nil || !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("validateRAMContractFuzzOracleWithHead error = %v, want git_head mismatch", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsMissingReport(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, dir)
	if err == nil || !strings.Contains(err.Error(), "ram-contract-fuzz-oracle.json") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want missing report rejection", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsMissingArtifactBundleFile(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	if err := os.Remove(filepath.Join(dir, "heap-blockers.json")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	err := validateRAMContractFuzzOracle(path, dir)
	if err == nil || !strings.Contains(err.Error(), "heap-blockers.json") {
		t.Fatalf("validateRAMContractFuzzOracle error = %v, want missing heap-blockers.json rejection", err)
	}
}

func writeRAMContractFuzzOracleArtifactBundle(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"ram-contract-fuzz-oracle.json": validRAMContractFuzzOracleForTest(),
		"ram-contract-report.json": `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{
    "site_id":"site:main:heap",
    "value_id":"heap",
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
  "non_claims":["no Memory 100% claim","no full formal proof claim","no official benchmark claim"]
}
`,
		"memory-grade-report.json": `{
  "schema_version":"tetra.memory-grade-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "artifact_grade":"M5",
  "functions":[{"function":"main","grade":"M5","row_count":1,"heap_rows":1,"copy_rows":0,"budget_bytes":8192}],
  "summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192},
  "non_claims":["no Memory 100% claim"]
}
`,
		"proof-store-summary.json": `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[],
  "summary":{"proof_count":0,"proven":0,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}
`,
		"validation-pipeline-coverage.json": `{
  "schema_version":"tetra.validation-pipeline-coverage.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "entries":[
    {"entrypoint":"BuildFileWithStatsOpt","artifact_path":"ram-contract-fixture","status":"validated_by_pipeline","validators":["ramcontract.ValidateReport"]},
    {"entrypoint":"buildObjectFileWithStatsOpt","status":"formal_exemption_with_reason","exemption":"not exercised by this linux-x64 RAM release fixture; object builds must carry their own RAM coverage evidence"},
    {"entrypoint":"buildLibraryObjectWithStatsOpt","status":"formal_exemption_with_reason","exemption":"not exercised by this linux-x64 RAM release fixture; library builds must carry their own RAM coverage evidence"},
    {"entrypoint":"InterfaceOnly","status":"formal_exemption_with_reason","exemption":"interface-only mode does not produce a RAM artifact in this release fixture"},
    {"entrypoint":"wasm32-wasi-build","status":"formal_exemption_with_reason","exemption":"wasm32-wasi RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
    {"entrypoint":"wasm32-web-build","status":"formal_exemption_with_reason","exemption":"wasm32-web RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
    {"entrypoint":"explain-report-path","status":"formal_exemption_with_reason","exemption":"explain report path is not artifact-producing in this release fixture"}
  ],
  "non_claims":["pipeline coverage is not proof completeness"]
}
`,
		"heap-blockers.json": `{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"heap",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{"site_id":"site:main:heap","function":"main","intent":"heap_fallback","placement":"heap_unbounded","blockers":["unknown_size"],"contract_grade":"M5"}],
  "non_claims":["no Memory 100% claim"]
}
`,
		"copy-blockers.json": `{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"copy",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[],
  "non_claims":["no Memory 100% claim"]
}
`,
		"ram-contract-fuzz-summary.md": "# RAM Contract Fuzz Summary\n\nValidator artifact bundle summary.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func validRAMContractFuzzOracleForTest() string {
	return `{
  "schema_version":"tetra.ram-contract-fuzz-oracle.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "generated_at":"2026-06-10T00:00:00Z",
  "observations":[
    {"mutation":"mutated_proof_id","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/mutated_proof_id/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/mutated_proof_id/ram-contract-report.json","reason":"rejected"},
    {"mutation":"widened_grade","rejected":true,"validator":"validate-memory-grade-report","validator_command":"go run ./tools/cmd/validate-memory-grade-report --report mutations/widened_grade/memory-grade-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/widened_grade/memory-grade-report.json","reason":"rejected"},
    {"mutation":"missing_blocker","rejected":true,"validator":"validate-heap-blockers","validator_command":"go run ./tools/cmd/validate-heap-blockers --report mutations/missing_blocker/heap-blockers.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/missing_blocker/heap-blockers.json","reason":"rejected"},
    {"mutation":"budget_drift","rejected":true,"validator":"validate-ram-contract-report","validator_command":"go run ./tools/cmd/validate-ram-contract-report --report mutations/budget_drift/ram-contract-report.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/budget_drift/ram-contract-report.json","reason":"rejected"},
    {"mutation":"artifact_hash_drift","rejected":true,"validator":"validate-artifact-hashes","validator_command":"go run ./tools/cmd/validate-artifact-hashes --manifest mutations/artifact_hash_drift/artifact-hashes.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/artifact_hash_drift/ram-contract-fuzz-summary.md","reason":"rejected"},
    {"mutation":"forbidden_nonclaim_text","rejected":true,"validator":"validate-ram-contract-fuzz-oracle","validator_command":"go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","exit_code":1,"output_excerpt":"rejected","mutated_file":"mutations/forbidden_nonclaim_text/ram-contract-fuzz-oracle.json","reason":"rejected"}
  ],
  "summary":{"mutations":6,"rejected":6},
  "non_claims":["not a full formal proof"]
}
`
}
