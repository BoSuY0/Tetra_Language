package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type testAllSummary struct {
	Mode        string        `json:"mode"`
	Status      string        `json:"status"`
	StartedAt   string        `json:"started_at"`
	EndedAt     string        `json:"ended_at"`
	StepCount   int           `json:"step_count"`
	FailedCount int           `json:"failed_count"`
	Steps       []testAllStep `json:"steps"`
}

type testAllStep struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	DurationSeconds int    `json:"duration_seconds"`
	ExitCode        int    `json:"exit_code"`
	Command         string `json:"command"`
	Log             string `json:"log"`
}

func main() {
	var summaryPath string
	var reportDir string
	flag.StringVar(&summaryPath, "summary", "", "path to scripts/test_all.sh summary.json")
	flag.StringVar(&reportDir, "report-dir", "", "report directory containing logs")
	flag.Parse()

	if summaryPath == "" {
		fmt.Fprintln(os.Stderr, "error: --summary is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if reportDir == "" {
		reportDir = filepath.Dir(summaryPath)
	}
	if err := validateTestAllSummary(raw, reportDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTestAllSummary(raw []byte, reportDir string) error {
	var summary testAllSummary
	if err := decodeStrictJSON(raw, &summary); err != nil {
		return err
	}
	switch summary.Mode {
	case "quick", "full":
	default:
		return fmt.Errorf("invalid mode %q", summary.Mode)
	}
	switch summary.Status {
	case "pass", "fail":
	default:
		return fmt.Errorf("invalid status %q", summary.Status)
	}
	if summary.StartedAt == "" {
		return fmt.Errorf("started_at is required")
	}
	if summary.EndedAt == "" {
		return fmt.Errorf("ended_at is required")
	}
	if summary.StepCount != len(summary.Steps) {
		return fmt.Errorf("step_count mismatch: got %d, computed %d", summary.StepCount, len(summary.Steps))
	}
	failed := 0
	seenNames := make(map[string]bool, len(summary.Steps))
	seenLogs := make(map[string]bool, len(summary.Steps))
	for _, step := range summary.Steps {
		if err := validateStep(step, reportDir); err != nil {
			return err
		}
		if seenNames[step.Name] {
			return fmt.Errorf("duplicate step name %q", step.Name)
		}
		seenNames[step.Name] = true
		if seenLogs[step.Log] {
			return fmt.Errorf("duplicate step log %q", step.Log)
		}
		seenLogs[step.Log] = true
		if step.Status == "fail" {
			failed++
		}
	}
	if summary.FailedCount != failed {
		return fmt.Errorf("failed_count mismatch: got %d, computed %d", summary.FailedCount, failed)
	}
	if summary.Status == "pass" && failed != 0 {
		return fmt.Errorf("pass summary contains failing steps")
	}
	if summary.Status == "fail" && failed == 0 {
		return fmt.Errorf("fail summary contains no failing steps")
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateStep(step testAllStep, reportDir string) error {
	if step.Name == "" {
		return fmt.Errorf("step missing name")
	}
	switch step.Status {
	case "pass":
		if step.ExitCode != 0 {
			return fmt.Errorf("pass step %s has non-zero exit code %d", step.Name, step.ExitCode)
		}
	case "fail":
		if step.ExitCode == 0 {
			return fmt.Errorf("fail step %s has zero exit code", step.Name)
		}
	default:
		return fmt.Errorf("step %s has invalid status %q", step.Name, step.Status)
	}
	if step.DurationSeconds < 0 {
		return fmt.Errorf("step %s has negative duration", step.Name)
	}
	if step.Command == "" {
		return fmt.Errorf("step %s missing command", step.Name)
	}
	if step.Log == "" {
		return fmt.Errorf("step %s missing log", step.Name)
	}
	if filepath.IsAbs(step.Log) || strings.Contains(step.Log, "..") || !strings.HasPrefix(filepath.ToSlash(step.Log), "logs/") {
		return fmt.Errorf("step %s has unsafe log path %s", step.Name, step.Log)
	}
	logPath := filepath.Join(reportDir, step.Log)
	if info, err := os.Stat(logPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("step %s missing log file %s", step.Name, step.Log)
		}
		return err
	} else if info.IsDir() {
		return fmt.Errorf("step %s log path is a directory", step.Name)
	}
	return nil
}
