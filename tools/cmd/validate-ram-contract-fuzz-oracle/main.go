package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.ram-contract-fuzz-oracle.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateRAMContractFuzzOracle(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateRAMContractFuzzOracle(path string) error {
	var report struct {
		SchemaVersion string `json:"schema_version"`
		Observations  []struct {
			Mutation  string `json:"mutation"`
			Rejected  bool   `json:"rejected"`
			Validator string `json:"validator"`
			Reason    string `json:"reason"`
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
		return fmt.Errorf("schema_version is %q, want tetra.ram-contract-fuzz-oracle.v1", report.SchemaVersion)
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
		if !obs.Rejected || strings.TrimSpace(obs.Validator) == "" || strings.TrimSpace(obs.Reason) == "" {
			return fmt.Errorf("mutation %s is not rejected with validator evidence", obs.Mutation)
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
	return nil
}
