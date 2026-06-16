package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const oracleSchema = "tetra.ram-contract-fuzz-oracle.v1"

type oracleReport struct {
	SchemaVersion string              `json:"schema_version"`
	GitHead       string              `json:"git_head,omitempty"`
	GeneratedAt   string              `json:"generated_at"`
	Observations  []oracleObservation `json:"observations"`
	Summary       oracleSummary       `json:"summary"`
	NonClaims     []string            `json:"non_claims"`
}

type oracleObservation struct {
	Mutation         string `json:"mutation"`
	Rejected         bool   `json:"rejected"`
	Validator        string `json:"validator"`
	ValidatorCommand string `json:"validator_command"`
	ExitCode         int    `json:"exit_code"`
	OutputExcerpt    string `json:"output_excerpt"`
	MutatedFile      string `json:"mutated_file"`
	Reason           string `json:"reason"`
}

type oracleSummary struct {
	Mutations int `json:"mutations"`
	Rejected  int `json:"rejected"`
}

func main() {
	reportDir := flag.String("report-dir", "", "directory to write deterministic RAM contract fuzz artifacts")
	gitHead := flag.String("git-head", "unknown", "git head to stamp into artifacts")
	flag.Parse()
	if *reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := runRAMContractFuzzShort(*reportDir, *gitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRAMContractFuzzShort(reportDir string, gitHead string) error {
	if err := requireFreshReportDir(reportDir); err != nil {
		return err
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	if err := writeArtifactFixtures(reportDir, gitHead); err != nil {
		return err
	}
	observations, err := runMutationObservations(reportDir, gitHead)
	if err != nil {
		return err
	}
	oracle := oracleReport{
		SchemaVersion: oracleSchema,
		GitHead:       gitHead,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Observations:  observations,
		NonClaims: []string{
			"not a full formal proof",
			"not Memory 100%",
			"not a performance benchmark",
		},
	}
	oracle.Summary.Mutations = len(oracle.Observations)
	for _, obs := range oracle.Observations {
		if obs.Rejected {
			oracle.Summary.Rejected++
		}
	}
	raw, err := json.MarshalIndent(oracle, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(filepath.Join(reportDir, "ram-contract-fuzz-oracle.json"), raw, 0o644)
}

type mutationCase struct {
	name      string
	validator string
	mutate    func(repoRoot string, mutationDir string, gitHead string) (string, []string, error)
}

func runMutationObservations(reportDir string, gitHead string) ([]oracleObservation, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, err
	}
	cases := []mutationCase{
		{
			name:      "mutated_proof_id",
			validator: "validate-ram-contract-report",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				path := filepath.Join(mutationDir, "ram-contract-report.json")
				if err := replaceInFile(path, `"proof_ids":[]`, `"proof_ids":["missing-proof"]`); err != nil {
					return "", nil, err
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-ram-contract-report", "--report", abs}, nil
			},
		},
		{
			name:      "widened_grade",
			validator: "validate-memory-grade-report",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				path := filepath.Join(mutationDir, "memory-grade-report.json")
				if err := replaceInFile(path, `"artifact_grade":"M5"`, `"artifact_grade":"M0"`); err != nil {
					return "", nil, err
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-memory-grade-report", "--report", abs}, nil
			},
		},
		{
			name:      "missing_blocker",
			validator: "validate-heap-blockers",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				path := filepath.Join(mutationDir, "heap-blockers.json")
				if err := replaceInFile(path, `"blockers":["unknown_size"]`, `"blockers":[]`); err != nil {
					return "", nil, err
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-heap-blockers", "--report", abs}, nil
			},
		},
		{
			name:      "budget_drift",
			validator: "validate-ram-contract-report",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				path := filepath.Join(mutationDir, "ram-contract-report.json")
				if err := replaceInFile(path, `"summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192}`, `"summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8193}`); err != nil {
					return "", nil, err
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-ram-contract-report", "--report", abs}, nil
			},
		},
		{
			name:      "artifact_hash_drift",
			validator: "validate-artifact-hashes",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				absDir, err := filepath.Abs(mutationDir)
				if err != nil {
					return "", nil, err
				}
				manifest := filepath.Join(absDir, "artifact-hashes.json")
				exitCode, output, err := runCommand(repoRoot, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-artifact-hashes", "--write", "--root", absDir, "--out", manifest})
				if err != nil {
					return "", nil, err
				}
				if exitCode != 0 {
					return "", nil, fmt.Errorf("artifact hash manifest write failed: %s", excerptOutput(output))
				}
				path := filepath.Join(mutationDir, "ram-contract-fuzz-summary.md")
				if err := appendToFile(path, "\nForged after hash manifest.\n"); err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-artifact-hashes", "--manifest", manifest}, nil
			},
		},
		{
			name:      "forbidden_nonclaim_text",
			validator: "validate-ram-contract-fuzz-oracle",
			mutate: func(repoRoot string, mutationDir string, gitHead string) (string, []string, error) {
				path := filepath.Join(mutationDir, "ram-contract-fuzz-oracle.json")
				raw, err := json.MarshalIndent(forbiddenClaimOracle(gitHead), "", "  ")
				if err != nil {
					return "", nil, err
				}
				raw = append(raw, '\n')
				if err := os.WriteFile(path, raw, 0o644); err != nil {
					return "", nil, err
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", nil, err
				}
				return path, []string{"go", "run", "-buildvcs=false", "./tools/cmd/validate-ram-contract-fuzz-oracle", "--report", abs}, nil
			},
		},
	}
	var observations []oracleObservation
	for _, tc := range cases {
		mutationDir := filepath.Join(reportDir, "mutations", tc.name)
		if err := os.MkdirAll(mutationDir, 0o755); err != nil {
			return nil, err
		}
		if err := writeArtifactFixtures(mutationDir, gitHead); err != nil {
			return nil, err
		}
		mutatedPath, command, err := tc.mutate(repoRoot, mutationDir, gitHead)
		if err != nil {
			return nil, fmt.Errorf("%s mutation setup: %w", tc.name, err)
		}
		exitCode, output, err := runCommand(repoRoot, command)
		if err != nil {
			return nil, fmt.Errorf("%s validator command failed to run: %w", tc.name, err)
		}
		rel, err := filepath.Rel(reportDir, mutatedPath)
		if err != nil {
			return nil, err
		}
		obs := oracleObservation{
			Mutation:         tc.name,
			Rejected:         exitCode != 0,
			Validator:        tc.validator,
			ValidatorCommand: strings.Join(command, " "),
			ExitCode:         exitCode,
			OutputExcerpt:    excerptOutput(output),
			MutatedFile:      filepath.ToSlash(rel),
			Reason:           fmt.Sprintf("%s rejected by %s with exit code %d", tc.name, tc.validator, exitCode),
		}
		if !obs.Rejected {
			return nil, fmt.Errorf("mutation %s was accepted by %s", tc.name, tc.validator)
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func writeArtifactFixtures(dir string, gitHead string) error {
	fixtures := validArtifactFixtures(gitHead)
	for name, raw := range fixtures {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validArtifactFixtures(gitHead string) map[string]string {
	return map[string]string{
		"ram-contract-report.json":          validRAMContractReport(gitHead),
		"memory-grade-report.json":          validMemoryGradeReport(gitHead),
		"proof-store-summary.json":          validProofStoreSummary(gitHead),
		"validation-pipeline-coverage.json": validPipelineCoverage(gitHead),
		"heap-blockers.json":                validHeapBlockers(gitHead),
		"copy-blockers.json":                validCopyBlockers(gitHead),
		"ram-contract-fuzz-summary.md":      "# RAM Contract Fuzz Summary\n\nDeterministic mutation fixtures reject forged RAM evidence. This is not a formal proof.\n",
	}
}

func runCommand(repoRoot string, args []string) (int, string, error) {
	if len(args) == 0 {
		return 0, "", fmt.Errorf("empty command")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = repoRoot
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err == nil {
		return 0, output.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), output.String(), nil
	}
	return 0, output.String(), err
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "tools", "cmd", "validate-ram-contract-report")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repo root from %s", dir)
		}
		dir = parent
	}
}

func replaceInFile(path string, old string, replacement string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	if !strings.Contains(text, old) {
		return fmt.Errorf("%s does not contain mutation target %q", path, old)
	}
	text = strings.Replace(text, old, replacement, 1)
	return os.WriteFile(path, []byte(text), 0o644)
}

func appendToFile(path string, suffix string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw = append(raw, []byte(suffix)...)
	return os.WriteFile(path, raw, 0o644)
}

func excerptOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 1200 {
		return output[:1200]
	}
	return output
}

func forbiddenClaimOracle(gitHead string) oracleReport {
	mutations := []string{
		"mutated_proof_id",
		"widened_grade",
		"missing_blocker",
		"budget_drift",
		"artifact_hash_drift",
		"forbidden_nonclaim_text",
	}
	report := oracleReport{
		SchemaVersion: oracleSchema,
		GitHead:       gitHead,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		NonClaims:     []string{"Memory 100%"},
		Summary:       oracleSummary{Mutations: len(mutations), Rejected: len(mutations)},
	}
	for _, mutation := range mutations {
		report.Observations = append(report.Observations, oracleObservation{
			Mutation:         mutation,
			Rejected:         true,
			Validator:        "test-validator",
			ValidatorCommand: "test-validator --fixture " + mutation,
			ExitCode:         1,
			OutputExcerpt:    "rejected",
			MutatedFile:      "mutations/" + mutation + "/fixture.json",
			Reason:           "rejected",
		})
	}
	return report
}

func requireFreshReportDir(reportDir string) error {
	info, err := os.Lstat(reportDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use symlink report directory: %s", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to use non-directory report path: %s", reportDir)
	}
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("refusing to reuse non-empty report directory: %s", reportDir)
	}
	return nil
}

func validRAMContractReport(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.ram-contract-report.v1",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
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
  "non_claims":["no Memory 100%% claim","no full formal proof claim","no official benchmark claim"]
}
`, gitHead)
}

func validMemoryGradeReport(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.memory-grade-report.v1",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
  "artifact_grade":"M5",
  "functions":[{"function":"main","grade":"M5","row_count":1,"heap_rows":1,"copy_rows":0,"budget_bytes":8192}],
  "summary":{"row_count":1,"artifact_grade":"M5","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192},
  "non_claims":["no Memory 100%% claim"]
}
`, gitHead)
}

func validProofStoreSummary(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.proof-store-summary.v1",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
  "proofs":[],
  "summary":{"proof_count":0,"proven":0,"conservative":0,"rejected":0,"unknown":0},
  "non_claims":["no full formal proof claim"]
}
`, gitHead)
}

func validPipelineCoverage(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.validation-pipeline-coverage.v1",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
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
`, gitHead)
}

func validHeapBlockers(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"heap",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
  "rows":[{"site_id":"site:main:heap","function":"main","intent":"heap_fallback","placement":"heap_unbounded","blockers":["unknown_size"],"contract_grade":"M5","file":"fixtures/main.tetra","line":3,"symbol":"main","source_location_status":"available","severity":"P1","reason":"unknown_size","suggested_fix":"add no-escape, lifetime, or bounded allocation proof before changing this heap fallback","evidence_id":"fact:ram:site:main:heap","safe_to_optimize":false}],
  "non_claims":["no Memory 100%% claim"]
}
`, gitHead)
}

func validCopyBlockers(gitHead string) string {
	return fmt.Sprintf(`{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"copy",
  "git_head":%q,
  "target":"linux-x64",
  "generated_by":"ram-contract-fuzz-short",
  "rows":[],
  "non_claims":["no Memory 100%% claim"]
}
`, gitHead)
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
