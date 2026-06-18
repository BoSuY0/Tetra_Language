package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ramContractFuzzOracleTestGitHead = "e2c19b8ee276158f8eb2c54cf61e11bd84952893"

type oracleObservationFixture struct {
	Mutation         string `json:"mutation"`
	Rejected         bool   `json:"rejected"`
	Validator        string `json:"validator"`
	ValidatorCommand string `json:"validator_command,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	OutputExcerpt    string `json:"output_excerpt,omitempty"`
	MutatedFile      string `json:"mutated_file,omitempty"`
	Reason           string `json:"reason"`
}

type oracleSummaryFixture struct {
	Mutations int `json:"mutations"`
	Rejected  int `json:"rejected"`
}

type oracleReportFixture struct {
	SchemaVersion string                     `json:"schema_version"`
	GitHead       string                     `json:"git_head"`
	GeneratedAt   string                     `json:"generated_at"`
	Observations  []oracleObservationFixture `json:"observations"`
	Summary       oracleSummaryFixture       `json:"summary"`
	NonClaims     []string                   `json:"non_claims"`
}

func TestValidateRAMContractFuzzOracleRejectsMissingMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	observations := validOracleObservationsForTest()
	observations[2] = oracleObservationForTest(
		"other",
		"validate-ram-contract-report",
		"--report",
		"ram-contract-report.json",
		"ram-contract-report.json",
	)
	raw := ramContractFuzzOracleJSONForTest(
		observations,
		[]string{"not a full formal proof"},
	)
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
	observations := validOracleObservationsForTest()
	observations[0] = oracleObservationFixture{
		Mutation:  "mutated_proof_id",
		Rejected:  true,
		Validator: "validate-ram-contract-report",
		Reason:    "self-asserted",
	}
	raw := ramContractFuzzOracleJSONForTest(
		observations,
		[]string{"not a full formal proof"},
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, "")
	if err == nil || !strings.Contains(err.Error(), "exit evidence") {
		t.Fatalf(
			"validateRAMContractFuzzOracle error = %v, want missing exit evidence rejection",
			err,
		)
	}
}

func TestValidateRAMContractFuzzOracleRejectsForbiddenClaimText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	raw := ramContractFuzzOracleJSONForTest(
		validOracleObservationsForTest(),
		[]string{"Memory 100%"},
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateRAMContractFuzzOracle(path, "")
	if err == nil || !strings.Contains(err.Error(), "forbidden broad claim") {
		t.Fatalf(
			"validateRAMContractFuzzOracle error = %v, want forbidden broad claim rejection",
			err,
		)
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
	if err := validateRAMContractFuzzOracleWithHead(
		path,
		"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		dir,
	); err != nil {
		t.Fatalf("validateRAMContractFuzzOracleWithHead: %v", err)
	}
}

func TestValidateRAMContractFuzzOracleRejectsMismatchedCurrentGitHead(t *testing.T) {
	dir := t.TempDir()
	writeRAMContractFuzzOracleArtifactBundle(t, dir)
	path := filepath.Join(dir, "ram-contract-fuzz-oracle.json")
	err := validateRAMContractFuzzOracleWithHead(
		path,
		"ffffffffffffffffffffffffffffffffffffffff",
		dir,
	)
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
		t.Fatalf(
			"validateRAMContractFuzzOracle error = %v, want missing heap-blockers.json rejection",
			err,
		)
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
  "summary":{
    "row_count":1,
    "artifact_grade":"M5",
    "heap_rows":1,
    "copy_rows":0,
    "unbounded_rows":1,
    "budget_bytes":8192
  },
  "non_claims":["no Memory 100% claim","no full formal proof claim","no official benchmark claim"]
}
`,
		"memory-grade-report.json": `{
  "schema_version":"tetra.memory-grade-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "artifact_grade":"M5",
  "functions":[{
    "function":"main",
    "grade":"M5",
    "row_count":1,
    "heap_rows":1,
    "copy_rows":0,
    "budget_bytes":8192
  }],
  "summary":{
    "row_count":1,
    "artifact_grade":"M5",
    "heap_rows":1,
    "copy_rows":0,
    "unbounded_rows":1,
    "budget_bytes":8192
  },
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
    {
      "entrypoint":"BuildFileWithStatsOpt",
      "artifact_path":"ram-contract-fixture",
      "status":"validated_by_pipeline",
      "validators":["ramcontract.ValidateReport"]
    },
    {
      "entrypoint":"buildObjectFileWithStatsOpt",
      "status":"formal_exemption_with_reason",
      "exemption":"not exercised by this linux-x64 RAM release fixture"
    },
    {
      "entrypoint":"buildLibraryObjectWithStatsOpt",
      "status":"formal_exemption_with_reason",
      "exemption":"not exercised by this linux-x64 RAM release fixture"
    },
    {
      "entrypoint":"InterfaceOnly",
      "status":"formal_exemption_with_reason",
      "exemption":"interface-only mode does not produce a RAM artifact"
    },
    {
      "entrypoint":"wasm32-wasi-build",
      "status":"formal_exemption_with_reason",
      "exemption":"wasm32-wasi RAM coverage is target-specific"
    },
    {
      "entrypoint":"wasm32-web-build",
      "status":"formal_exemption_with_reason",
      "exemption":"wasm32-web RAM coverage is target-specific"
    },
    {
      "entrypoint":"explain-report-path",
      "status":"formal_exemption_with_reason",
      "exemption":"explain report path is not artifact-producing"
    }
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
  "rows":[{
    "site_id":"site:main:heap",
    "function":"main",
    "intent":"heap_fallback",
    "placement":"heap_unbounded",
    "blockers":["unknown_size"],
    "contract_grade":"M5",
    "file":"fixtures/main.tetra",
    "line":3,
    "symbol":"main",
    "source_location_status":"available",
    "severity":"P1",
    "reason":"unknown_size",
    "suggested_fix":"add no-escape, lifetime, or bounded allocation proof",
    "evidence_id":"fact:ram:site:main:heap",
    "safe_to_optimize":false
  }],
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
		"ram-contract-fuzz-summary.md": strings.Join([]string{
			"# RAM Contract Fuzz Summary",
			"",
			"Validator artifact bundle summary.",
			"",
		}, "\n"),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func validRAMContractFuzzOracleForTest() string {
	return ramContractFuzzOracleJSONForTest(
		validOracleObservationsForTest(),
		[]string{"not a full formal proof"},
	)
}

func validOracleObservationsForTest() []oracleObservationFixture {
	return []oracleObservationFixture{
		oracleObservationForTest(
			"mutated_proof_id",
			"validate-ram-contract-report",
			"--report",
			"ram-contract-report.json",
			"ram-contract-report.json",
		),
		oracleObservationForTest(
			"widened_grade",
			"validate-memory-grade-report",
			"--report",
			"memory-grade-report.json",
			"memory-grade-report.json",
		),
		oracleObservationForTest(
			"missing_blocker",
			"validate-heap-blockers",
			"--report",
			"heap-blockers.json",
			"heap-blockers.json",
		),
		oracleObservationForTest(
			"budget_drift",
			"validate-ram-contract-report",
			"--report",
			"ram-contract-report.json",
			"ram-contract-report.json",
		),
		oracleObservationForTest(
			"artifact_hash_drift",
			"validate-artifact-hashes",
			"--manifest",
			"artifact-hashes.json",
			"ram-contract-fuzz-summary.md",
		),
		oracleObservationForTest(
			"forbidden_nonclaim_text",
			"validate-ram-contract-fuzz-oracle",
			"--report",
			"ram-contract-fuzz-oracle.json",
			"ram-contract-fuzz-oracle.json",
		),
	}
}

func oracleObservationForTest(
	mutation string,
	validator string,
	commandFlag string,
	commandFile string,
	mutatedFile string,
) oracleObservationFixture {
	exitCode := 1
	return oracleObservationFixture{
		Mutation:         mutation,
		Rejected:         true,
		Validator:        validator,
		ValidatorCommand: oracleValidatorCommandForTest(validator, commandFlag, mutation, commandFile),
		ExitCode:         &exitCode,
		OutputExcerpt:    "rejected",
		MutatedFile:      filepath.Join("mutations", mutation, mutatedFile),
		Reason:           "rejected",
	}
}

func oracleValidatorCommandForTest(
	validator string,
	commandFlag string,
	mutation string,
	commandFile string,
) string {
	return strings.Join([]string{
		"go run ./tools/cmd/" + validator,
		commandFlag,
		filepath.Join("mutations", mutation, commandFile),
	}, " ")
}

func ramContractFuzzOracleJSONForTest(
	observations []oracleObservationFixture,
	nonClaims []string,
) string {
	rejected := 0
	for _, observation := range observations {
		if observation.Rejected {
			rejected++
		}
	}
	return mustJSONForTest(oracleReportFixture{
		SchemaVersion: "tetra.ram-contract-fuzz-oracle.v1",
		GitHead:       ramContractFuzzOracleTestGitHead,
		GeneratedAt:   "2026-06-10T00:00:00Z",
		Observations:  observations,
		Summary: oracleSummaryFixture{
			Mutations: len(observations),
			Rejected:  rejected,
		},
		NonClaims: nonClaims,
	})
}

func mustJSONForTest(value any) string {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}
