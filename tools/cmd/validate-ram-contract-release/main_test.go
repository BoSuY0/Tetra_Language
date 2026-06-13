package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestValidateRAMContractReleaseRejectsMissingReport(t *testing.T) {
	dir := t.TempDir()
	err := validateRAMContractRelease(dir, "")
	if err == nil || !strings.Contains(err.Error(), "ram-contract-report.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing report", err)
	}
}

func TestValidateReleaseHashManifestRejectsMissingRAMArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact-hashes.json")
	raw := `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateReleaseHashManifest(path)
	if err == nil || !strings.Contains(err.Error(), "missing hash entry") {
		t.Fatalf("validateReleaseHashManifest error = %v, want missing artifact rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsForbiddenManifestClaim(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"Memory 100%"})
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "forbidden broad claim") {
		t.Fatalf("validateRAMContractRelease error = %v, want forbidden broad claim rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsMissingFuzzOracle(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.Remove(filepath.Join(dir, "fuzz", "ram-contract-fuzz-oracle.json")); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "fuzz/ram-contract-fuzz-oracle.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing fuzz oracle rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsMissingReleaseManifest(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.Remove(filepath.Join(dir, "ram-contract-release-manifest.json")); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "ram-contract-release-manifest.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing release manifest rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsArtifactHashDrift(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.WriteFile(filepath.Join(dir, "ram-contract-report.json"), []byte(validReleaseRAMContractReport("e2c19b8ee276158f8eb2c54cf61e11bd84952893")+"\n "), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("validateRAMContractRelease error = %v, want hash mismatch rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsUnlistedArtifact(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.WriteFile(filepath.Join(dir, "extra.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "unlisted artifact extra.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want unlisted artifact rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsGitHeadMismatchAcrossArtifacts(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "proof-store-summary.json")
	raw := strings.Replace(validReleaseProofStoreSummary("e2c19b8ee276158f8eb2c54cf61e11bd84952893"), "e2c19b8ee276158f8eb2c54cf61e11bd84952893", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "proof-store-summary.json") || !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("validateRAMContractRelease error = %v, want proof-store git_head mismatch rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsRAMReportProofMissingFromProofStore(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.WriteFile(filepath.Join(dir, "ram-contract-report.json"), []byte(validReleaseTrustedRAMContractReport("e2c19b8ee276158f8eb2c54cf61e11bd84952893")), 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "proof:ram:main:alloc0") || !strings.Contains(err.Error(), "proof-store-summary.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing proof-store reference rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsRejectedProofReferencedByRAMRow(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	if err := os.WriteFile(filepath.Join(dir, "ram-contract-report.json"), []byte(validReleaseTrustedRAMContractReport("e2c19b8ee276158f8eb2c54cf61e11bd84952893")), 0o644); err != nil {
		t.Fatal(err)
	}
	rejectedProofStore := `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[{"proof_id":"proof:ram:main:alloc0","kind":"allocation_placement","subject":"main/alloc0","stable_hash":"sha256:test","status":"rejected"}],
  "summary":{"proof_count":1,"proven":0,"conservative":0,"rejected":1,"unknown":0},
  "non_claims":["no full formal proof claim"]
}
`
	if err := os.WriteFile(filepath.Join(dir, "proof-store-summary.json"), []byte(rejectedProofStore), 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err := validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "rejected") || !strings.Contains(err.Error(), "proof:ram:main:alloc0") {
		t.Fatalf("validateRAMContractRelease error = %v, want rejected proof reference rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsPipelineCoverageWithoutBuildArtifactPath(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "validation-pipeline-coverage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"artifact_path":"ram-contract-fixture",`, "", 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err = validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "artifact_path") {
		t.Fatalf("validateRAMContractRelease error = %v, want artifact_path rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsHeapBlockerRowNotInRAMReport(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "heap-blockers.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"rows":[`, `"rows":[{"site_id":"site:missing","function":"main","intent":"heap_fallback","placement":"heap_unbounded","blockers":["unknown_size"],"contract_grade":"M5"},`, 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err = validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "site:missing") {
		t.Fatalf("validateRAMContractRelease error = %v, want extra heap blocker row rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsRAMHeapRowMissingFromHeapBlockers(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "heap-blockers.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"rows":[{"site_id":"site:main:heap","function":"main","intent":"heap_fallback","placement":"heap_unbounded","blockers":["unknown_size"],"contract_grade":"M5"}]`, `"rows":[]`, 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err = validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "site:main:heap") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing heap blocker rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsCopyRowMissingFromCopyBlockers(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "ram-contract-report.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	text = strings.Replace(text, `"intent":"heap_fallback"`, `"intent":"copy_heap_unbounded"`, 1)
	text = strings.Replace(text, `"blockers":["unknown_size"],`, `"blockers":["unknown_size"],"copy_reason":"mutable_alias_boundary",`, 1)
	text = strings.Replace(text, `"copy_rows":0`, `"copy_rows":1`, 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err = validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "copy") || !strings.Contains(err.Error(), "site:main:heap") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing copy blocker rejection", err)
	}
}

func TestValidateRAMContractReleaseRejectsMemoryGradeReportMismatch(t *testing.T) {
	dir := t.TempDir()
	writeValidRAMContractReleaseBundle(t, dir, []string{"no Memory 100% claim"})
	path := filepath.Join(dir, "memory-grade-report.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"budget_bytes":8192`, `"budget_bytes":8193`, 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeReleaseHashManifest(t, dir)
	err = validateRAMContractRelease(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil || !strings.Contains(err.Error(), "memory-grade-report.json") || !strings.Contains(err.Error(), "RAM report") {
		t.Fatalf("validateRAMContractRelease error = %v, want memory grade mismatch rejection", err)
	}
}

func writeValidRAMContractReleaseBundle(t *testing.T, dir string, manifestNonClaims []string) {
	t.Helper()
	files := map[string]string{
		"ram-contract-report.json": validReleaseRAMContractReport("e2c19b8ee276158f8eb2c54cf61e11bd84952893"),
		"memory-grade-report.json": `{
  "schema_version":"tetra.memory-grade-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "artifact_grade":"M5",
  "functions":[],
  "summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192},
  "non_claims":["no Memory 100% claim"]
}
`,
		"proof-store-summary.json": validReleaseProofStoreSummary("e2c19b8ee276158f8eb2c54cf61e11bd84952893"),
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
		filepath.Join("fuzz", "ram-contract-fuzz-oracle.json"): validReleaseFuzzOracle("e2c19b8ee276158f8eb2c54cf61e11bd84952893"),
	}
	for name, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeReleaseManifest(t, dir, manifestNonClaims)
	writeReleaseHashManifest(t, dir)
}

func validReleaseRAMContractReport(gitHead string) string {
	return `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"` + gitHead + `",
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
  "non_claims":["no Memory 100% claim","no full formal proof claim"]
}
`
}

func validReleaseProofStoreSummary(gitHead string) string {
	return `{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":"` + gitHead + `",
  "target":"linux-x64",
  "generated_by":"test",
  "proofs":[],
  "summary":{"proof_count":0,"proven":0,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}
`
}

func validReleaseTrustedRAMContractReport(gitHead string) string {
	return `{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":"` + gitHead + `",
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
  "non_claims":["no Memory 100% claim","no full formal proof claim"]
}
`
}

func validReleaseFuzzOracle(gitHead string) string {
	return `{
  "schema_version":"tetra.ram-contract-fuzz-oracle.v1",
  "git_head":"` + gitHead + `",
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

func writeReleaseManifest(t *testing.T, dir string, manifestNonClaims []string) {
	t.Helper()
	manifest := `{
  "schema":"tetra.ram-contract.release-manifest.v1",
  "status":"pass",
  "target":"linux-x64",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "hash_manifest":"artifact-hashes.json",
  "commands":[
    {"name":"validate-ram-contract-report","command":"go run ./tools/cmd/validate-ram-contract-report --report ram-contract-report.json"},
    {"name":"validate-ram-contract-fuzz-oracle","command":"go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report fuzz/ram-contract-fuzz-oracle.json"}
  ],
  "artifacts":[
    {"path":"ram-contract-report.json","kind":"ram_contract_report","schema":"tetra.ram-contract-report.v1"},
    {"path":"memory-grade-report.json","kind":"memory_grade_report","schema":"tetra.memory-grade-report.v1"},
    {"path":"proof-store-summary.json","kind":"proof_store_summary","schema":"tetra.proof-store-summary.v1"},
    {"path":"validation-pipeline-coverage.json","kind":"validation_pipeline_coverage","schema":"tetra.validation-pipeline-coverage.v1"},
    {"path":"heap-blockers.json","kind":"heap_blockers","schema":"tetra.ram-blockers.v1"},
    {"path":"copy-blockers.json","kind":"copy_blockers","schema":"tetra.ram-blockers.v1"},
    {"path":"fuzz/ram-contract-fuzz-oracle.json","kind":"ram_contract_fuzz_oracle","schema":"tetra.ram-contract-fuzz-oracle.v1"},
    {"path":"artifact-hashes.json","kind":"artifact_hash_manifest","schema":"tetra.release-artifact-hashes.v1alpha1"}
  ],
  "non_claims":[`
	for i, claim := range manifestNonClaims {
		if i > 0 {
			manifest += ","
		}
		manifest += `"` + claim + `"`
	}
	manifest += `]
}
`
	if err := os.WriteFile(filepath.Join(dir, "ram-contract-release-manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeReleaseHashManifest(t *testing.T, dir string) {
	t.Helper()
	paths := []string{
		"copy-blockers.json",
		"fuzz/ram-contract-fuzz-oracle.json",
		"heap-blockers.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"ram-contract-release-manifest.json",
		"ram-contract-report.json",
		"validation-pipeline-coverage.json",
	}
	body := "{\n  \"schema\":\"tetra.release-artifact-hashes.v1alpha1\",\n  \"root\":\".\",\n  \"artifacts\":[\n"
	for i, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		sum := sha256.Sum256(raw)
		if i > 0 {
			body += ",\n"
		}
		body += `    {"path":"` + rel + `","sha256":"sha256:` + hex.EncodeToString(sum[:]) + `","size":` + stringInt(len(raw)) + `}`
	}
	body += "\n  ]\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "artifact-hashes.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringInt(n int) string {
	return strconv.Itoa(n)
}
