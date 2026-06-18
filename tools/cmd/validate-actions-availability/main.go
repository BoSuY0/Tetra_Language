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

const actionsAvailabilitySchema = "tetra.actions.availability.v1"

type actionsAvailabilityReport struct {
	Schema                string                 `json:"schema"`
	Status                string                 `json:"status"`
	Repo                  string                 `json:"repo"`
	Branch                string                 `json:"branch"`
	Workflow              string                 `json:"workflow"`
	ExpectedGitHead       string                 `json:"expected_git_head"`
	RunSelection          string                 `json:"run_selection"`
	Summary               string                 `json:"summary"`
	ProductionEvidence    bool                   `json:"production_evidence"`
	RepoActionsEnabled    bool                   `json:"repo_actions_enabled"`
	RepoAllowedActions    string                 `json:"repo_allowed_actions"`
	SelfHostedRunnerCount int                    `json:"self_hosted_runner_count"`
	BillingActionsStatus  string                 `json:"billing_actions_status"`
	BillingActionsDetail  string                 `json:"billing_actions_detail"`
	Workflows             actionsWorkflows       `json:"workflows"`
	Run                   actionsAvailabilityRun `json:"run"`
	NextAction            string                 `json:"next_action"`
}

type actionsWorkflows struct {
	TotalCount  int                       `json:"total_count"`
	ActiveCount int                       `json:"active_count"`
	Entries     []actionsWorkflowRegistry `json:"entries"`
}

type actionsWorkflowRegistry struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

type actionsAvailabilityRun struct {
	ID            int64                         `json:"id"`
	Event         string                        `json:"event"`
	Status        string                        `json:"status"`
	Conclusion    string                        `json:"conclusion"`
	HeadSHA       string                        `json:"head_sha"`
	WorkflowName  string                        `json:"workflow_name"`
	WorkflowPath  string                        `json:"workflow_path"`
	WorkflowID    int64                         `json:"workflow_id"`
	CheckSuiteID  int64                         `json:"check_suite_id"`
	CheckSuite    actionsAvailabilityCheckSuite `json:"check_suite"`
	Jobs          int                           `json:"jobs"`
	LogsAvailable bool                          `json:"logs_available"`
}

type actionsAvailabilityCheckSuite struct {
	ID                   int64  `json:"id"`
	App                  string `json:"app"`
	Status               string `json:"status"`
	Conclusion           string `json:"conclusion"`
	LatestCheckRunsCount int    `json:"latest_check_runs_count"`
	HeadSHA              string `json:"head_sha"`
}

func main() {
	reportPath := flag.String("report", "", "path to tetra.actions.availability.v1 JSON report")
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
	if err := validateActionsAvailability(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateActionsAvailability(raw []byte) error {
	var report actionsAvailabilityReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != actionsAvailabilitySchema {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %q", report.Schema, actionsAvailabilitySchema),
		)
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	for name, value := range map[string]string{
		"repo":                   report.Repo,
		"branch":                 report.Branch,
		"workflow":               report.Workflow,
		"expected_git_head":      report.ExpectedGitHead,
		"run_selection":          report.RunSelection,
		"summary":                report.Summary,
		"repo_allowed_actions":   report.RepoAllowedActions,
		"billing_actions_status": report.BillingActionsStatus,
		"billing_actions_detail": report.BillingActionsDetail,
		"next_action":            report.NextAction,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, name+" is required")
		}
	}
	if report.ProductionEvidence {
		issues = append(
			issues,
			"production_evidence must be false; Actions availability is not runtime evidence",
		)
	}
	claimText := strings.ToLower(report.Summary + " " + report.NextAction)
	if strings.Contains(claimText, "ready") {
		issues = append(issues, "Actions availability report must not claim READY")
	}
	if !report.RepoActionsEnabled {
		issues = append(issues, "repo_actions_enabled must be true")
	}
	if report.SelfHostedRunnerCount < 0 {
		issues = append(issues, "self_hosted_runner_count must be non-negative")
	}
	if report.BillingActionsStatus == "unavailable_missing_user_scope" {
		issues = append(
			issues,
			("billing_actions_status is unavailable_missing_user_scope; " +
				"refresh gh auth with user scope before availability can pass"),
		)
	}
	switch report.RunSelection {
	case "workflow_name",
		"empty_workflow_fallback",
		"workflow_name_stale",
		"empty_workflow_fallback_stale",
		"none":
	default:
		issues = append(
			issues,
			fmt.Sprintf(
				("run_selection is %q, want workflow_name, empty_workflow_"+
					"fallback, workflow_name_stale, empty_workflow_fallback_stale, or none"),
				report.RunSelection,
			),
		)
	}
	if report.Status == "pass" && report.RunSelection != "workflow_name" {
		issues = append(
			issues,
			fmt.Sprintf(
				"passing Actions availability requires run_selection workflow_name, got %q",
				report.RunSelection,
			),
		)
	}
	if report.Run.HeadSHA != "" && report.ExpectedGitHead != "" &&
		report.Run.HeadSHA != report.ExpectedGitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"run.head_sha is %q, want expected_git_head %q",
				report.Run.HeadSHA,
				report.ExpectedGitHead,
			),
		)
	}
	if report.Run.CheckSuite.HeadSHA != "" && report.ExpectedGitHead != "" &&
		report.Run.CheckSuite.HeadSHA != report.ExpectedGitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"run.check_suite.head_sha is %q, want expected_git_head %q",
				report.Run.CheckSuite.HeadSHA,
				report.ExpectedGitHead,
			),
		)
	}
	issues = append(issues, validateActionsWorkflows(report.Workflows, report.Workflow)...)
	issues = append(issues, validateAvailabilityRun(report.Run)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateActionsWorkflows(workflows actionsWorkflows, expectedWorkflow string) []string {
	var issues []string
	if workflows.TotalCount <= 0 {
		issues = append(issues, "workflows.total_count must be positive")
	}
	if workflows.ActiveCount <= 0 {
		issues = append(issues, "workflows.active_count must be positive")
	}
	if len(workflows.Entries) == 0 {
		issues = append(issues, "workflows.entries must not be empty")
	}
	actualActive := 0
	hasExpectedActive := false
	for i, entry := range workflows.Entries {
		prefix := fmt.Sprintf("workflows.entries[%d]", i)
		if entry.ID <= 0 {
			issues = append(issues, prefix+".id must be positive")
		}
		if strings.TrimSpace(entry.Path) == "" {
			issues = append(issues, prefix+".path is required")
		}
		if strings.TrimSpace(entry.State) == "" {
			issues = append(issues, prefix+".state is required")
		}
		if entry.State == "active" {
			actualActive++
			if entry.Name == expectedWorkflow ||
				strings.HasSuffix(entry.Path, "/"+expectedWorkflow+".yml") ||
				strings.HasSuffix(entry.Path, "/"+expectedWorkflow+".yaml") {
				hasExpectedActive = true
			}
		}
	}
	if workflows.ActiveCount != actualActive {
		issues = append(
			issues,
			fmt.Sprintf(
				"workflows.active_count is %d, computed %d",
				workflows.ActiveCount,
				actualActive,
			),
		)
	}
	if expectedWorkflow != "" && !hasExpectedActive {
		issues = append(
			issues,
			fmt.Sprintf("workflows missing active workflow %q", expectedWorkflow),
		)
	}
	return issues
}

func validateAvailabilityRun(run actionsAvailabilityRun) []string {
	var issues []string
	if run.ID <= 0 {
		issues = append(issues, "run.id must be positive")
	}
	if strings.TrimSpace(run.Event) == "" {
		issues = append(issues, "run.event is required")
	}
	if strings.TrimSpace(run.WorkflowPath) == "" {
		issues = append(issues, "run.workflow_path is required")
	}
	if run.WorkflowPath == "BuildFailed" {
		issues = append(issues, "run.workflow_path is BuildFailed; workflow did not build jobs")
	}
	if run.WorkflowID <= 0 {
		issues = append(issues, "run.workflow_id must be positive")
	}
	if run.CheckSuiteID <= 0 {
		issues = append(issues, "run.check_suite_id must be positive")
	}
	if run.Status != "completed" {
		issues = append(issues, fmt.Sprintf("run.status is %q, want completed", run.Status))
	}
	if run.Conclusion != "success" {
		issues = append(issues, fmt.Sprintf("run.conclusion is %q, want success", run.Conclusion))
	}
	if strings.TrimSpace(run.HeadSHA) == "" {
		issues = append(issues, "run.head_sha is required")
	}
	issues = append(issues, validateAvailabilityCheckSuite(run.CheckSuite)...)
	if run.Jobs <= 0 {
		issues = append(issues, fmt.Sprintf("run.jobs is %d, want at least 1", run.Jobs))
	}
	if !run.LogsAvailable {
		issues = append(issues, "run.logs_available must be true")
	}
	return issues
}

func validateAvailabilityCheckSuite(suite actionsAvailabilityCheckSuite) []string {
	var issues []string
	if suite.ID <= 0 {
		issues = append(issues, "run.check_suite.id must be positive")
	}
	if suite.App != "github-actions" {
		issues = append(
			issues,
			fmt.Sprintf("run.check_suite.app is %q, want github-actions", suite.App),
		)
	}
	if suite.Status != "completed" {
		issues = append(
			issues,
			fmt.Sprintf("run.check_suite.status is %q, want completed", suite.Status),
		)
	}
	if suite.Conclusion != "success" {
		issues = append(
			issues,
			fmt.Sprintf("run.check_suite.conclusion is %q, want success", suite.Conclusion),
		)
	}
	if suite.LatestCheckRunsCount <= 0 {
		issues = append(
			issues,
			fmt.Sprintf(
				"run.check_suite.latest_check_runs_count is %d, want at least 1",
				suite.LatestCheckRunsCount,
			),
		)
	}
	if strings.TrimSpace(suite.HeadSHA) == "" {
		issues = append(issues, "run.check_suite.head_sha is required")
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
