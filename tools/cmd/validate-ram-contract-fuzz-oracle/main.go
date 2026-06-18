package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.ram-contract-fuzz-oracle.v1 JSON report")
	artifactDir := flag.String(
		"artifact-dir",
		"",
		"optional RAM contract fuzz artifact directory to validate alongside the oracle report",
	)
	currentGitHead := flag.String(
		"current-git-head",
		"",
		"optional current git HEAD to require in the oracle report",
	)
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateRAMContractFuzzOracleWithHead(
		*reportPath,
		*currentGitHead,
		*artifactDir,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateRAMContractFuzzOracle(path string, artifactDirs ...string) error {
	return validateRAMContractFuzzOracleWithHead(path, "", artifactDirs...)
}

func validateRAMContractFuzzOracleWithHead(
	path string,
	currentGitHead string,
	artifactDirs ...string,
) error {
	var report struct {
		SchemaVersion string `json:"schema_version"`
		Observations  []struct {
			Mutation         string `json:"mutation"`
			Rejected         bool   `json:"rejected"`
			Validator        string `json:"validator"`
			ValidatorCommand string `json:"validator_command"`
			ExitCode         *int   `json:"exit_code"`
			OutputExcerpt    string `json:"output_excerpt"`
			MutatedFile      string `json:"mutated_file"`
			Reason           string `json:"reason"`
		} `json:"observations"`
		Summary struct {
			Mutations int `json:"mutations"`
			Rejected  int `json:"rejected"`
		} `json:"summary"`
		NonClaims   []string `json:"non_claims"`
		GitHead     string   `json:"git_head,omitempty"`
		GeneratedAt string   `json:"generated_at"`
	}
	if err := ramvalidate.ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
	if report.SchemaVersion != "tetra.ram-contract-fuzz-oracle.v1" {
		return fmt.Errorf(
			"schema_version is %q, want tetra.ram-contract-fuzz-oracle.v1",
			report.SchemaVersion,
		)
	}
	currentGitHead = strings.TrimSpace(currentGitHead)
	if currentGitHead != "" && report.GitHead != currentGitHead {
		return fmt.Errorf(
			"git_head %s does not match current git head %s",
			report.GitHead,
			currentGitHead,
		)
	}
	if err := validateOracleClaimText(report.NonClaims, "non_claims"); err != nil {
		return err
	}
	required := map[string]bool{
		"mutated_proof_id":        false,
		"widened_grade":           false,
		"missing_blocker":         false,
		"budget_drift":            false,
		"artifact_hash_drift":     false,
		"forbidden_nonclaim_text": false,
	}
	rejected := 0
	for _, obs := range report.Observations {
		if _, ok := required[obs.Mutation]; ok {
			required[obs.Mutation] = true
		}
		if !obs.Rejected || strings.TrimSpace(obs.Validator) == "" ||
			strings.TrimSpace(obs.Reason) == "" {
			return fmt.Errorf("mutation %s is not rejected with validator evidence", obs.Mutation)
		}
		if obs.ExitCode == nil || *obs.ExitCode == 0 ||
			strings.TrimSpace(obs.ValidatorCommand) == "" ||
			strings.TrimSpace(obs.OutputExcerpt) == "" ||
			strings.TrimSpace(obs.MutatedFile) == "" {
			return fmt.Errorf(
				"mutation %s is not rejected with validator exit evidence",
				obs.Mutation,
			)
		}
		if err := validateOracleClaimText([]string{obs.Reason}, "observation "+obs.Mutation); err != nil {
			return err
		}
		rejected++
	}
	for mutation, seen := range required {
		if !seen {
			return fmt.Errorf("missing mutation class %s", mutation)
		}
	}
	if report.Summary.Mutations != len(report.Observations) || report.Summary.Rejected != rejected {
		return fmt.Errorf("summary mismatch")
	}
	if len(artifactDirs) > 0 && strings.TrimSpace(artifactDirs[0]) != "" {
		return validateRAMContractFuzzOracleArtifactDir(path, artifactDirs[0])
	}
	return nil
}

func validateRAMContractFuzzOracleArtifactDir(reportPath string, artifactDir string) error {
	info, err := os.Lstat(artifactDir)
	if err != nil {
		return fmt.Errorf("RAM contract fuzz artifact dir: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("RAM contract fuzz artifact dir %s must not be a symlink", artifactDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("RAM contract fuzz artifact dir %s is not a directory", artifactDir)
	}
	expectedReport := filepath.Join(artifactDir, "ram-contract-fuzz-oracle.json")
	if same, err := sameCleanPath(reportPath, expectedReport); err != nil {
		return err
	} else if !same {
		return fmt.Errorf(
			"--report must point at %s when --artifact-dir is used, got %s",
			expectedReport,
			reportPath,
		)
	}
	for _, rel := range []string{
		"ram-contract-fuzz-oracle.json",
		"ram-contract-report.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"validation-pipeline-coverage.json",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-summary.md",
	} {
		if err := requireRAMContractFuzzArtifactFile(artifactDir, rel); err != nil {
			return err
		}
	}
	if err := ramvalidate.ValidateReportFile(
		filepath.Join(artifactDir, "ram-contract-report.json"),
	); err != nil {
		return fmt.Errorf("ram-contract-report.json: %w", err)
	}
	if err := ramvalidate.ValidateGradeReportFile(
		filepath.Join(artifactDir, "memory-grade-report.json"),
	); err != nil {
		return fmt.Errorf("memory-grade-report.json: %w", err)
	}
	if err := ramvalidate.ValidateProofStoreSummaryFile(
		filepath.Join(artifactDir, "proof-store-summary.json"),
	); err != nil {
		return fmt.Errorf("proof-store-summary.json: %w", err)
	}
	if err := ramvalidate.ValidatePipelineCoverageFile(
		filepath.Join(artifactDir, "validation-pipeline-coverage.json"),
	); err != nil {
		return fmt.Errorf("validation-pipeline-coverage.json: %w", err)
	}
	if err := ramvalidate.ValidateBlockerReportFile(
		filepath.Join(artifactDir, "heap-blockers.json"),
		"heap",
	); err != nil {
		return fmt.Errorf("heap-blockers.json: %w", err)
	}
	if err := ramvalidate.ValidateBlockerReportFile(
		filepath.Join(artifactDir, "copy-blockers.json"),
		"copy",
	); err != nil {
		return fmt.Errorf("copy-blockers.json: %w", err)
	}
	summary, err := os.ReadFile(filepath.Join(artifactDir, "ram-contract-fuzz-summary.md"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(summary), "RAM Contract Fuzz Summary") {
		return fmt.Errorf("ram-contract-fuzz-summary.md missing RAM Contract Fuzz Summary heading")
	}
	return nil
}

func validateOracleClaimText(values []string, label string) error {
	for _, value := range values {
		if forbiddenClaimWithoutNegation(value) {
			return fmt.Errorf("%s contains forbidden broad claim: %q", label, value)
		}
	}
	return nil
}

func forbiddenClaimWithoutNegation(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range []string{
		"memory 100%",
		"full formal proof",
		"official benchmark",
		"fastest language",
		"fastest-language",
		"faster than c",
		"faster than rust",
		"c/rust parity",
		"zero heap for all programs",
		"zero-copy for all programs",
		"all-target ram parity",
		"full target parity",
		"production actor runtime",
		"production object memory",
		"production persistent memory",
		"arbitrary unsafe external pointer safety",
	} {
		idx := strings.Index(lower, phrase)
		if idx < 0 {
			continue
		}
		prefix := strings.TrimSpace(lower[:idx])
		if negatedClaimPrefix(prefix) {
			continue
		}
		return true
	}
	return false
}

func negatedClaimPrefix(prefix string) bool {
	for _, allowed := range []string{
		"no",
		"not",
		"not a",
		"not an",
		"without",
		"does not claim",
		"do not claim",
		"nonclaim",
		"non-claim",
	} {
		if strings.HasSuffix(prefix, allowed) || strings.Contains(prefix, allowed+" ") {
			return true
		}
	}
	return false
}

func sameCleanPath(a string, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return filepath.Clean(absA) == filepath.Clean(absB), nil
}

func requireRAMContractFuzzArtifactFile(dir string, rel string) error {
	path := filepath.Join(dir, rel)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing required RAM contract fuzz artifact %s", rel)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("required RAM contract fuzz artifact %s is a directory", rel)
	}
	if info.Size() == 0 {
		return fmt.Errorf("required RAM contract fuzz artifact %s is empty", rel)
	}
	return nil
}
