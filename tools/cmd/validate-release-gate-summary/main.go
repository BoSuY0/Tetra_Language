package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const expectedReleaseVersion = "v0.3.0"
const expectedReleaseArtifact = "tetra.release.v0_3_0.gate-report.v1"
const expectedReleaseGateCommand = "bash scripts/release/v0_3_0/gate.sh"
const releaseArtifactHashesSchema = "tetra.release-artifact-hashes.v1alpha1"

var v040RequiredArtifacts = []struct {
	Path   string
	Schema string
}{
	{Path: "summary.json"},
	{Path: "summary.md"},
	{Path: "artifacts/features.json", Schema: "tetra.features.v1"},
	{Path: "artifacts/targets.json"},
	{Path: "artifacts/linux-host-smoke.json"},
	{Path: "artifacts/memory-production-linux-x64.json", Schema: "tetra.memory.production.v1"},
	{Path: "artifacts/parallel-production-linux-x64.json", Schema: "tetra.parallel.production.v1"},
	{Path: "artifacts/compiler-production-linux-x64.json", Schema: "tetra.compiler.production.v1"},
	{Path: "artifacts/distributed-actors-linux-x64.json", Schema: "tetra.actors.distributed-runtime.v1"},
	{Path: "artifacts/native-ui-linux-x64.json", Schema: "tetra.ui.native-runtime.v1"},
	{Path: "artifacts/release-state.json", Schema: "tetra.release.v0_4_0.release-state.v1"},
	{Path: "artifacts/release-state.txt"},
	{Path: "artifacts/security-review.md"},
	{Path: "artifacts/security-review.md.sha256"},
}

var v040RequiredPassingSteps = []string{
	"readiness preflight",
	"version parity",
	"readiness validator tests",
	"docs verification",
	"techempower report schemas",
	"compiler cli tools baseline",
	"memory production linux x64 smoke",
	"validate memory production",
	"parallel production linux x64 smoke",
	"validate parallel production",
	"compiler production linux x64 smoke",
	"validate compiler production",
	"linux host smoke",
	"distributed actors linux x64 smoke",
	"validate distributed actor runtime",
	"native ui linux x64 smoke",
	"validate native ui runtime",
	"readiness final",
	"completion audit validation",
	"release state",
	"security review signoff",
	"security review detached hash",
	"diff check",
}

type releaseGateSummaryExpectations struct {
	ReleaseVersion     string
	ReleaseArtifact    string
	ReleaseGateCommand string
}

type releaseGateSummary struct {
	Status             string            `json:"status"`
	ReleaseVersion     string            `json:"release_version"`
	ReleaseArtifact    string            `json:"release_artifact"`
	ReleaseGateCommand string            `json:"release_gate_command"`
	StartedAt          string            `json:"started_at"`
	EndedAt            string            `json:"ended_at"`
	StepCount          int               `json:"step_count"`
	FailedCount        int               `json:"failed_count"`
	ReportDir          string            `json:"report_dir"`
	Steps              []releaseGateStep `json:"steps"`
}

type releaseGateStep struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	DurationSeconds int    `json:"duration_seconds"`
	ExitCode        int    `json:"exit_code"`
	Command         string `json:"command"`
	Log             string `json:"log"`
}

type releaseArtifactHashesManifest struct {
	Schema    string                `json:"schema"`
	Root      string                `json:"root"`
	Artifacts []releaseHashArtifact `json:"artifacts"`
}

type releaseHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func main() {
	var summaryPath string
	var reportDir string
	expectations := defaultReleaseGateSummaryExpectations()
	flag.StringVar(&summaryPath, "summary", "", "path to release_v0_3_0_gate summary.json")
	flag.StringVar(&reportDir, "report-dir", "", "report directory containing logs")
	flag.StringVar(&expectations.ReleaseVersion, "expected-version", expectations.ReleaseVersion, "expected release version")
	flag.StringVar(&expectations.ReleaseArtifact, "expected-artifact", expectations.ReleaseArtifact, "expected release artifact")
	flag.StringVar(&expectations.ReleaseGateCommand, "expected-command", expectations.ReleaseGateCommand, "expected release gate command")
	flag.Parse()

	if summaryPath == "" {
		fmt.Fprintln(os.Stderr, "error: --summary is required")
		os.Exit(2)
	}
	if err := validateReleaseGateSummaryFileWithExpectations(summaryPath, reportDir, expectations); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultReleaseGateSummaryExpectations() releaseGateSummaryExpectations {
	return releaseGateSummaryExpectations{
		ReleaseVersion:     expectedReleaseVersion,
		ReleaseArtifact:    expectedReleaseArtifact,
		ReleaseGateCommand: expectedReleaseGateCommand,
	}
}

func validateReleaseGateSummaryFile(summaryPath, reportDir string) error {
	return validateReleaseGateSummaryFileWithExpectations(summaryPath, reportDir, defaultReleaseGateSummaryExpectations())
}

func validateReleaseGateSummaryFileWithExpectations(summaryPath, reportDir string, expectations releaseGateSummaryExpectations) error {
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		return err
	}
	if reportDir == "" {
		reportDir = filepath.Dir(summaryPath)
	}
	return validateReleaseGateSummaryWithExpectations(raw, reportDir, expectations)
}

func validateReleaseGateSummary(raw []byte, reportDir string) error {
	return validateReleaseGateSummaryWithExpectations(raw, reportDir, defaultReleaseGateSummaryExpectations())
}

func validateReleaseGateSummaryWithExpectations(raw []byte, reportDir string, expectations releaseGateSummaryExpectations) error {
	var summary releaseGateSummary
	if err := decodeStrictJSON(raw, &summary); err != nil {
		return err
	}
	switch summary.Status {
	case "pass", "blocked":
	default:
		return fmt.Errorf("invalid status %q", summary.Status)
	}
	if summary.ReleaseVersion != expectations.ReleaseVersion {
		return fmt.Errorf("release_version = %q, want %q", summary.ReleaseVersion, expectations.ReleaseVersion)
	}
	if summary.ReleaseArtifact != expectations.ReleaseArtifact {
		return fmt.Errorf("release_artifact = %q, want %q", summary.ReleaseArtifact, expectations.ReleaseArtifact)
	}
	if summary.ReleaseGateCommand != expectations.ReleaseGateCommand {
		return fmt.Errorf("release_gate_command = %q, want %q", summary.ReleaseGateCommand, expectations.ReleaseGateCommand)
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
	if strings.TrimSpace(summary.ReportDir) == "" {
		return fmt.Errorf("report_dir is required")
	}

	failed := 0
	seenNames := make(map[string]bool, len(summary.Steps))
	seenLogs := make(map[string]bool, len(summary.Steps))
	for i, step := range summary.Steps {
		if err := validateReleaseGateStep(step, reportDir, i+1); err != nil {
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
	if summary.Status == "blocked" && failed == 0 {
		return fmt.Errorf("blocked summary contains no failing steps")
	}
	if summary.Status == "pass" && failed != 0 {
		return fmt.Errorf("pass summary contains failing steps")
	}
	if summary.Status == "pass" && summary.ReleaseVersion == "v0.4.0" {
		if err := validateV040RequiredPassingSteps(summary); err != nil {
			return err
		}
		if err := validateV040ReleaseArtifacts(summary, reportDir); err != nil {
			return err
		}
	}
	return nil
}

func validateV040RequiredPassingSteps(summary releaseGateSummary) error {
	steps := make(map[string]releaseGateStep, len(summary.Steps))
	for _, step := range summary.Steps {
		steps[step.Name] = step
	}
	for _, required := range v040RequiredPassingSteps {
		step, ok := steps[required]
		if !ok {
			return fmt.Errorf("passing v0.4.0 summary missing required step %q", required)
		}
		if step.Status != "pass" {
			return fmt.Errorf("passing v0.4.0 summary required step %q status = %q, want pass", required, step.Status)
		}
	}
	return nil
}

func validateV040ReleaseArtifacts(summary releaseGateSummary, reportDir string) error {
	manifestPath := filepath.Join(reportDir, "artifact-hashes.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("passing v0.4.0 summary missing artifact hash manifest %s", manifestPath)
		}
		return err
	}
	var manifest releaseArtifactHashesManifest
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return fmt.Errorf("artifact-hashes.json is invalid: %w", err)
	}
	if manifest.Schema != releaseArtifactHashesSchema {
		return fmt.Errorf("artifact-hashes.json schema = %q, want %q", manifest.Schema, releaseArtifactHashesSchema)
	}
	artifacts := make(map[string]releaseHashArtifact, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		artifacts[filepath.ToSlash(artifact.Path)] = artifact
	}
	for _, required := range v040RequiredArtifacts {
		artifact, ok := artifacts[required.Path]
		if !ok {
			return fmt.Errorf("passing v0.4.0 summary missing required artifact %s in artifact-hashes.json", required.Path)
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			return fmt.Errorf("artifact %s schema = %q, want %q", required.Path, artifact.Schema, required.Schema)
		}
	}
	for _, step := range summary.Steps {
		log := filepath.ToSlash(step.Log)
		if _, ok := artifacts[log]; !ok {
			return fmt.Errorf("passing v0.4.0 summary missing step log %s in artifact-hashes.json", log)
		}
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
		return fmt.Errorf("summary must contain a single JSON document")
	}
	return nil
}

func validateReleaseGateStep(step releaseGateStep, reportDir string, expectedIndex int) error {
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
