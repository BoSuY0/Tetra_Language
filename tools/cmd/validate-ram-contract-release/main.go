package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/internal/ramvalidate"
)

var requiredReleaseArtifacts = []string{
	"ram-contract-report.json",
	"memory-grade-report.json",
	"proof-store-summary.json",
	"validation-pipeline-coverage.json",
	"heap-blockers.json",
	"copy-blockers.json",
}

func main() {
	reportDir := flag.String("report-dir", "", "RAM contract release report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	flag.Parse()
	if *reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateRAMContractRelease(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateRAMContractRelease(reportDir string, currentGitHead string) error {
	var issues []string
	ramPath := filepath.Join(reportDir, "ram-contract-report.json")
	if err := ramvalidate.ValidateReportFile(ramPath); err != nil {
		issues = append(issues, "ram-contract-report.json: "+err.Error())
	}
	if err := ramvalidate.ValidateGradeReportFile(filepath.Join(reportDir, "memory-grade-report.json")); err != nil {
		issues = append(issues, "memory-grade-report.json: "+err.Error())
	}
	if err := ramvalidate.ValidateProofStoreSummaryFile(filepath.Join(reportDir, "proof-store-summary.json")); err != nil {
		issues = append(issues, "proof-store-summary.json: "+err.Error())
	}
	if err := ramvalidate.ValidatePipelineCoverageFile(filepath.Join(reportDir, "validation-pipeline-coverage.json")); err != nil {
		issues = append(issues, "validation-pipeline-coverage.json: "+err.Error())
	}
	if err := ramvalidate.ValidateBlockerReportFile(filepath.Join(reportDir, "heap-blockers.json"), "heap"); err != nil {
		issues = append(issues, "heap-blockers.json: "+err.Error())
	}
	if err := ramvalidate.ValidateBlockerReportFile(filepath.Join(reportDir, "copy-blockers.json"), "copy"); err != nil {
		issues = append(issues, "copy-blockers.json: "+err.Error())
	}
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(ramPath, &report); err == nil && strings.TrimSpace(currentGitHead) != "" && report.GitHead != strings.TrimSpace(currentGitHead) {
		issues = append(issues, fmt.Sprintf("ram-contract-report git_head %s does not match current git head %s", report.GitHead, strings.TrimSpace(currentGitHead)))
	}
	if err := validateReleaseHashManifest(filepath.Join(reportDir, "artifact-hashes.json")); err != nil {
		issues = append(issues, "artifact-hashes.json: "+err.Error())
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateReleaseHashManifest(path string) error {
	var manifest struct {
		Schema    string `json:"schema"`
		Root      string `json:"root"`
		Artifacts []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
			Schema string `json:"schema,omitempty"`
		} `json:"artifacts"`
	}
	if err := ramvalidate.ReadStrictJSONFile(path, &manifest); err != nil {
		return err
	}
	if manifest.Schema != "tetra.release-artifact-hashes.v1alpha1" {
		return fmt.Errorf("schema is %q, want tetra.release-artifact-hashes.v1alpha1", manifest.Schema)
	}
	seen := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		seen[artifact.Path] = true
	}
	for _, required := range requiredReleaseArtifacts {
		if !seen[required] {
			return fmt.Errorf("missing hash entry for %s", required)
		}
	}
	return nil
}
