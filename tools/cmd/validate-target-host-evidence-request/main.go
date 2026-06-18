package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const targetHostEvidenceRequestSchema = "tetra.ui.target-host-evidence-request.v1"

type requestValidationOptions struct {
	ExpectedRepo    string
	ExpectedBranch  string
	ExpectedVersion string
	ExpectedGitHead string
}

type targetHostEvidenceRequest struct {
	Schema             string          `json:"schema"`
	Status             string          `json:"status"`
	ProductionEvidence bool            `json:"production_evidence"`
	Repo               string          `json:"repo"`
	Branch             string          `json:"branch"`
	ExpectedVersion    string          `json:"expected_version"`
	ExpectedGitHead    string          `json:"expected_git_head"`
	Warning            string          `json:"warning"`
	Targets            []requestTarget `json:"targets"`
	Aggregation        requestCommand  `json:"aggregation"`
}

type requestTarget struct {
	Target          string `json:"target"`
	HostRequirement string `json:"host_requirement"`
	Report          string `json:"report"`
	Command         string `json:"command"`
}

type requestCommand struct {
	HostRequirement string `json:"host_requirement"`
	Command         string `json:"command"`
}

func main() {
	reportPath := flag.String(
		"report",
		"",
		"path to tetra.ui.target-host-evidence-request.v1 JSON report",
	)
	expectedRepo := flag.String("expected-repo", "", "expected OWNER/REPO repository")
	expectedBranch := flag.String("expected-branch", "", "expected Git branch")
	expectedVersion := flag.String("expected-version", "", "expected Tetra version")
	expectedGitHead := flag.String("expected-git-head", "", "expected Git HEAD")
	flag.Parse()
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	opts := requestValidationOptions{
		ExpectedRepo:    *expectedRepo,
		ExpectedBranch:  *expectedBranch,
		ExpectedVersion: *expectedVersion,
		ExpectedGitHead: *expectedGitHead,
	}
	if err := validateTargetHostEvidenceRequest(raw, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTargetHostEvidenceRequest(raw []byte, opts requestValidationOptions) error {
	var report targetHostEvidenceRequest
	if err := decodeStrictJSON(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != targetHostEvidenceRequestSchema {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %q", report.Schema, targetHostEvidenceRequestSchema),
		)
	}
	if report.Status != "request" {
		issues = append(issues, fmt.Sprintf("status is %q, want request", report.Status))
	}
	if report.ProductionEvidence {
		issues = append(
			issues,
			"production_evidence must be false; target-host requests are not runtime evidence",
		)
	}
	for name, value := range map[string]string{
		"repo":              report.Repo,
		"branch":            report.Branch,
		"expected_version":  report.ExpectedVersion,
		"expected_git_head": report.ExpectedGitHead,
		"warning":           report.Warning,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, name+" is required")
		}
	}
	if strings.Contains(report.Repo, "://") || strings.HasSuffix(report.Repo, ".git") ||
		strings.Contains(report.Repo, " ") ||
		!strings.Contains(report.Repo, "/") {
		issues = append(
			issues,
			fmt.Sprintf("repo is %q, want OWNER/REPO without URL or .git suffix", report.Repo),
		)
	}
	issues = appendExpectedIssue(issues, "repo", report.Repo, opts.ExpectedRepo)
	issues = appendExpectedIssue(issues, "branch", report.Branch, opts.ExpectedBranch)
	issues = appendExpectedIssue(
		issues,
		"expected_version",
		report.ExpectedVersion,
		opts.ExpectedVersion,
	)
	issues = appendExpectedIssue(
		issues,
		"expected_git_head",
		report.ExpectedGitHead,
		opts.ExpectedGitHead,
	)

	claimText := strings.ToLower(report.Warning + " " + report.Aggregation.Command)
	if strings.Contains(claimText, "ready") {
		issues = append(issues, "target-host request must not claim READY")
	}
	if !strings.Contains(strings.ToLower(report.Warning), "not runtime evidence") {
		issues = append(issues, "warning must say the request is not runtime evidence")
	}
	issues = append(issues, validateRequestTargets(report)...)
	issues = append(issues, validateAggregation(report.Aggregation)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func appendExpectedIssue(issues []string, name, got, want string) []string {
	if strings.TrimSpace(want) != "" && got != want {
		return append(issues, fmt.Sprintf("%s is %q, want %q", name, got, want))
	}
	return issues
}

func validateRequestTargets(report targetHostEvidenceRequest) []string {
	var issues []string
	seen := map[string]bool{}
	for i, target := range report.Targets {
		prefix := fmt.Sprintf("targets[%d]", i)
		if strings.TrimSpace(target.Target) == "" {
			issues = append(issues, prefix+".target is required")
		}
		if seen[target.Target] {
			issues = append(issues, fmt.Sprintf("duplicate target %q", target.Target))
		}
		seen[target.Target] = true
		if strings.TrimSpace(target.HostRequirement) == "" {
			issues = append(issues, prefix+".host_requirement is required")
		}
		if strings.TrimSpace(target.Report) == "" {
			issues = append(issues, prefix+".report is required")
		}
		if strings.TrimSpace(target.Command) == "" {
			issues = append(issues, prefix+".command is required")
		}
		issues = append(issues, validateTargetCommand(target, report)...)
	}
	for _, target := range []string{"windows-x64", "macos-x64"} {
		if !seen[target] {
			issues = append(issues, "targets missing "+target)
		}
	}
	if len(report.Targets) != 2 {
		issues = append(issues, fmt.Sprintf("targets length is %d, want 2", len(report.Targets)))
	}
	return issues
}

func validateTargetCommand(target requestTarget, report targetHostEvidenceRequest) []string {
	var issues []string
	command := target.Command
	prefix := target.Target + " command"
	for _, want := range []string{
		"git clone https://github.com/" + report.Repo + ".git",
		"git fetch origin " + report.Branch,
		"git checkout " + report.ExpectedGitHead,
		report.ExpectedVersion,
		report.ExpectedGitHead,
	} {
		if !strings.Contains(command, want) {
			issues = append(issues, fmt.Sprintf("%s missing %q", prefix, want))
		}
	}
	if strings.Contains(command, ".git.git") {
		issues = append(issues, prefix+" contains duplicated .git.git suffix")
	}
	switch target.Target {
	case "windows-x64":
		for _, want := range []string{
			"windows-ui-runtime-smoke.ps1",
			"-ExpectedVersion " + report.ExpectedVersion,
			"-ExpectedGitHead " + report.ExpectedGitHead,
			"windows-ui-runtime.json",
		} {
			if !strings.Contains(command, want) {
				issues = append(issues, fmt.Sprintf("%s missing %q", prefix, want))
			}
		}
	case "macos-x64":
		for _, want := range []string{
			"target-host-ui-runtime-smoke.sh --target macos-x64",
			"--expected-version " + report.ExpectedVersion,
			"--expected-git-head " + report.ExpectedGitHead,
			"macos-ui-runtime.json",
		} {
			if !strings.Contains(command, want) {
				issues = append(issues, fmt.Sprintf("%s missing %q", prefix, want))
			}
		}
	default:
		issues = append(issues, fmt.Sprintf("unsupported target %q", target.Target))
	}
	return issues
}

func validateAggregation(aggregation requestCommand) []string {
	var issues []string
	if strings.TrimSpace(aggregation.HostRequirement) == "" {
		issues = append(issues, "aggregation.host_requirement is required")
	}
	if strings.TrimSpace(aggregation.Command) == "" {
		issues = append(issues, "aggregation.command is required")
	}
	for _, want := range []string{
		"TETRA_WINDOWS_UI_RUNTIME_REPORT",
		"TETRA_MACOS_UI_RUNTIME_REPORT",
		"ui-runtime-gate.sh",
	} {
		if !strings.Contains(aggregation.Command, want) {
			issues = append(issues, fmt.Sprintf("aggregation.command missing %q", want))
		}
	}
	return issues
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
