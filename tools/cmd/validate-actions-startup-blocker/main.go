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

const startupBlockerSchema = "tetra.actions.startup-blocker.v1"

type startupBlockerReport struct {
	Schema      string              `json:"schema"`
	Status      string              `json:"status"`
	Repo        string              `json:"repo"`
	Branch      string              `json:"branch"`
	Workflow    string              `json:"workflow"`
	Summary     string              `json:"summary"`
	Diagnostics *startupDiagnostics `json:"diagnostics"`
	Runs        []startupBlockerRun `json:"runs"`
	NextAction  string              `json:"next_action"`
}

type startupDiagnostics struct {
	RepoActionsEnabled    bool                  `json:"repo_actions_enabled"`
	RepoAllowedActions    string                `json:"repo_allowed_actions"`
	SelfHostedRunnerCount int                   `json:"self_hosted_runner_count"`
	BillingActionsStatus  string                `json:"billing_actions_status"`
	BillingActionsDetail  string                `json:"billing_actions_detail"`
	MinimalCanary         *startupBlockerCanary `json:"minimal_canary"`
}

type startupBlockerCanary struct {
	Branch        string `json:"branch"`
	Workflow      string `json:"workflow"`
	ID            int64  `json:"id"`
	Event         string `json:"event"`
	Conclusion    string `json:"conclusion"`
	HeadSHA       string `json:"head_sha"`
	Jobs          int    `json:"jobs"`
	LogsAvailable bool   `json:"logs_available"`
}

type startupBlockerRun struct {
	ID            int64  `json:"id"`
	Event         string `json:"event"`
	Conclusion    string `json:"conclusion"`
	HeadSHA       string `json:"head_sha"`
	Jobs          int    `json:"jobs"`
	LogsAvailable bool   `json:"logs_available"`
}

func main() {
	reportPath := flag.String("report", "", "path to GitHub Actions startup blocker report")
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
	if err := validateStartupBlocker(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateStartupBlocker(raw []byte) error {
	var report startupBlockerReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != startupBlockerSchema {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %q", report.Schema, startupBlockerSchema),
		)
	}
	if report.Status != "blocked" {
		issues = append(issues, fmt.Sprintf("status is %q, want blocked", report.Status))
	}
	for name, value := range map[string]string{
		"repo":        report.Repo,
		"branch":      report.Branch,
		"workflow":    report.Workflow,
		"summary":     report.Summary,
		"next_action": report.NextAction,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, name+" is required")
		}
	}
	if !strings.Contains(strings.ToLower(report.NextAction), "manual") &&
		!strings.Contains(strings.ToLower(report.NextAction), "self-hosted") {
		issues = append(
			issues,
			"next_action must direct manual or self-hosted target-host evidence",
		)
	}
	if strings.Contains(strings.ToLower(report.NextAction), "ready") ||
		strings.Contains(strings.ToLower(report.Summary), "ready") {
		issues = append(issues, "startup blocker report must not claim READY")
	}
	if report.Diagnostics == nil {
		issues = append(issues, "diagnostics is required")
	} else {
		if !report.Diagnostics.RepoActionsEnabled {
			issues = append(
				issues,
				"diagnostics.repo_actions_enabled must be true for an Actions startup blocker",
			)
		}
		if strings.TrimSpace(report.Diagnostics.RepoAllowedActions) == "" {
			issues = append(issues, "diagnostics.repo_allowed_actions is required")
		}
		if report.Diagnostics.SelfHostedRunnerCount < 0 {
			issues = append(issues, "diagnostics.self_hosted_runner_count must be non-negative")
		}
		if strings.TrimSpace(report.Diagnostics.BillingActionsStatus) == "" {
			issues = append(issues, "diagnostics.billing_actions_status is required")
		}
		if strings.TrimSpace(report.Diagnostics.BillingActionsDetail) == "" {
			issues = append(issues, "diagnostics.billing_actions_detail is required")
		}
		if canary := report.Diagnostics.MinimalCanary; canary != nil {
			if strings.TrimSpace(canary.Branch) == "" {
				issues = append(issues, "diagnostics.minimal_canary.branch is required")
			}
			if strings.TrimSpace(canary.Workflow) == "" {
				issues = append(issues, "diagnostics.minimal_canary.workflow is required")
			}
			canaryRun := startupBlockerRun{
				ID:            canary.ID,
				Event:         canary.Event,
				Conclusion:    canary.Conclusion,
				HeadSHA:       canary.HeadSHA,
				Jobs:          canary.Jobs,
				LogsAvailable: canary.LogsAvailable,
			}
			if runIssues := validateStartupRun(canaryRun, "diagnostics.minimal_canary"); len(runIssues) > 0 {
				issues = append(issues, runIssues...)
			}
		}
	}
	if len(report.Runs) == 0 {
		issues = append(issues, "runs must include at least one startup_failure run")
	}
	for i, run := range report.Runs {
		issues = append(issues, validateStartupRun(run, fmt.Sprintf("runs[%d]", i))...)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateStartupRun(run startupBlockerRun, path string) []string {
	var issues []string
	if run.ID <= 0 {
		issues = append(issues, fmt.Sprintf("%s.id must be positive", path))
	}
	if strings.TrimSpace(run.Event) == "" {
		issues = append(issues, fmt.Sprintf("%s.event is required", path))
	}
	if run.Conclusion != "startup_failure" {
		issues = append(
			issues,
			fmt.Sprintf("%s.conclusion is %q, want startup_failure", path, run.Conclusion),
		)
	}
	if strings.TrimSpace(run.HeadSHA) == "" {
		issues = append(issues, fmt.Sprintf("%s.head_sha is required", path))
	}
	if run.Jobs != 0 {
		issues = append(issues, fmt.Sprintf("%s.jobs is %d, want 0", path, run.Jobs))
	}
	if run.LogsAvailable {
		issues = append(issues, fmt.Sprintf("%s.logs_available is true, want false", path))
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
