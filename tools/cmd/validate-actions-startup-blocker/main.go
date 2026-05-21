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
	Schema     string              `json:"schema"`
	Status     string              `json:"status"`
	Repo       string              `json:"repo"`
	Branch     string              `json:"branch"`
	Workflow   string              `json:"workflow"`
	Summary    string              `json:"summary"`
	Runs       []startupBlockerRun `json:"runs"`
	NextAction string              `json:"next_action"`
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
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, startupBlockerSchema))
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
	if !strings.Contains(strings.ToLower(report.NextAction), "manual") && !strings.Contains(strings.ToLower(report.NextAction), "self-hosted") {
		issues = append(issues, "next_action must direct manual or self-hosted target-host evidence")
	}
	if strings.Contains(strings.ToLower(report.NextAction), "ready") || strings.Contains(strings.ToLower(report.Summary), "ready") {
		issues = append(issues, "startup blocker report must not claim READY")
	}
	if len(report.Runs) == 0 {
		issues = append(issues, "runs must include at least one startup_failure run")
	}
	for i, run := range report.Runs {
		if run.ID <= 0 {
			issues = append(issues, fmt.Sprintf("runs[%d].id must be positive", i))
		}
		if strings.TrimSpace(run.Event) == "" {
			issues = append(issues, fmt.Sprintf("runs[%d].event is required", i))
		}
		if run.Conclusion != "startup_failure" {
			issues = append(issues, fmt.Sprintf("runs[%d].conclusion is %q, want startup_failure", i, run.Conclusion))
		}
		if strings.TrimSpace(run.HeadSHA) == "" {
			issues = append(issues, fmt.Sprintf("runs[%d].head_sha is required", i))
		}
		if run.Jobs != 0 {
			issues = append(issues, fmt.Sprintf("runs[%d].jobs is %d, want 0", i, run.Jobs))
		}
		if run.LogsAvailable {
			issues = append(issues, fmt.Sprintf("runs[%d].logs_available is true, want false", i))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
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
