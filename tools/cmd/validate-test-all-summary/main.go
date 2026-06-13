package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tetra_language/tools/internal/reportdecode"
)

type testAllSummary struct {
	Mode            string        `json:"mode"`
	Status          string        `json:"status"`
	StartedAt       string        `json:"started_at"`
	EndedAt         string        `json:"ended_at"`
	StepCount       int           `json:"step_count"`
	FailedCount     int           `json:"failed_count"`
	ReleaseVersion  string        `json:"release_version,omitempty"`
	ReleaseArtifact string        `json:"release_artifact,omitempty"`
	Steps           []testAllStep `json:"steps"`
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
	var format string
	flag.StringVar(&summaryPath, "summary", "", "path to scripts/ci/test-all.sh summary report")
	flag.StringVar(&reportDir, "report-dir", "", "report directory containing logs")
	flag.StringVar(&format, "format", "auto", "summary format: auto, json, or toon")
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
	if err := validateTestAllSummaryFormat(raw, reportDir, format); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTestAllSummary(raw []byte, reportDir string) error {
	return validateTestAllSummaryFormat(raw, reportDir, "auto")
}

func validateTestAllSummaryFormat(raw []byte, reportDir string, format string) error {
	var summary testAllSummary
	if err := reportdecode.DecodeStrictFormat(raw, format, &summary); err != nil {
		return err
	}
	switch summary.Mode {
	case "quick", "full", "stabilization":
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
	startedAt, err := time.Parse(time.RFC3339, summary.StartedAt)
	if err != nil {
		return fmt.Errorf("started_at must be RFC3339: %w", err)
	}
	endedAt, err := time.Parse(time.RFC3339, summary.EndedAt)
	if err != nil {
		return fmt.Errorf("ended_at must be RFC3339: %w", err)
	}
	if endedAt.Before(startedAt) {
		return fmt.Errorf("ended_at must not be before started_at")
	}
	if summary.StepCount != len(summary.Steps) {
		return fmt.Errorf("step_count mismatch: got %d, computed %d", summary.StepCount, len(summary.Steps))
	}
	if len(summary.Steps) == 0 {
		return fmt.Errorf("summary must contain at least one step")
	}
	failed := 0
	seenNames := make(map[string]bool, len(summary.Steps))
	seenLogs := make(map[string]bool, len(summary.Steps))
	for i, step := range summary.Steps {
		if err := validateStep(step, reportDir, i+1); err != nil {
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
	if summary.Status == "pass" {
		if err := validateRequiredPassingSteps(summary.Mode, seenNames); err != nil {
			return err
		}
	}
	if summary.ReleaseVersion != "" && (!strings.HasPrefix(summary.ReleaseVersion, "v") || strings.ContainsAny(summary.ReleaseVersion, " \t\n\r")) {
		return fmt.Errorf("invalid release_version %q", summary.ReleaseVersion)
	}
	expectedArtifact := expectedTestAllSummaryArtifact(summary.ReleaseVersion)
	if summary.ReleaseArtifact != "" && summary.ReleaseArtifact != expectedArtifact {
		return fmt.Errorf("release_artifact = %q, want %q", summary.ReleaseArtifact, expectedArtifact)
	}
	return nil
}

func validateRequiredPassingSteps(mode string, seen map[string]bool) error {
	required := []string{
		"go test all packages",
		"json diagnostic shape",
		"host smoke linux-x64",
	}
	if mode == "full" || mode == "stabilization" {
		required = append(required,
			"docs manifest diff",
			"safety readiness evidence",
			"ownership production audit",
			"tooling summary aggregation",
		)
	}
	if mode == "stabilization" {
		required = append(required,
			"frontend callable focused gate",
			"safety runtime focused gate",
			"lowering ir focused gate",
			"wasi runner smoke",
			"web runtime browser smoke",
			"api diff no-change",
			"working tree whitespace audit",
		)
	}
	for _, step := range required {
		if !seen[step] {
			return fmt.Errorf("missing required step %q for %s pass summary", step, mode)
		}
	}
	return nil
}

func expectedTestAllSummaryArtifact(version string) string {
	if version == "" {
		version = "v0.2.0"
	}
	slug := strings.TrimPrefix(version, "v")
	slug = strings.ReplaceAll(slug, ".", "_")
	return "tetra.release.v" + slug + ".test-all-summary.v1"
}

func validateStep(step testAllStep, reportDir string, expectedIndex int) error {
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
	if err := validateLogOrdinal(step.Log, expectedIndex); err != nil {
		return fmt.Errorf("step %s %w", step.Name, err)
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

func validateLogOrdinal(logPath string, expectedIndex int) error {
	base := filepath.Base(logPath)
	if len(base) < 3 {
		return fmt.Errorf("has malformed log filename %s", logPath)
	}
	prefix := base[:2]
	index, err := strconv.Atoi(prefix)
	if err != nil {
		return fmt.Errorf("has malformed step ordinal in log %s", logPath)
	}
	if index != expectedIndex {
		return fmt.Errorf("log ordinal %02d does not match step order %02d", index, expectedIndex)
	}
	if len(base) == 2 || base[2] != '-' {
		return fmt.Errorf("has malformed log filename %s", logPath)
	}
	return nil
}
