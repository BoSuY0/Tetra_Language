package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const schemaV1 = "tetra.release.v0_4_0.release-state.v1"

type releaseStateReport struct {
	Schema            string          `json:"schema"`
	Status            string          `json:"status"`
	Version           string          `json:"version"`
	ExpectedVersion   string          `json:"expected_version"`
	Scope             string          `json:"scope"`
	ReportDir         string          `json:"report_dir,omitempty"`
	Git               gitState        `json:"git"`
	RequiredArtifacts []artifactState `json:"required_artifacts"`
	Issues            []string        `json:"issues,omitempty"`
}

type gitState struct {
	Clean   bool             `json:"clean"`
	Entries []gitStatusEntry `json:"entries,omitempty"`
}

type gitStatusEntry struct {
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Path     string `json:"path"`
}

type artifactState struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size,omitempty"`
}

func main() {
	expectedVersion := flag.String("expected-version", "v0.4.0", "expected release version")
	format := flag.String("format", "json", "output format: json or text")
	reportDir := flag.String("report-dir", "", "release gate report directory")
	flag.Parse()

	report := buildReleaseState(*expectedVersion, *reportDir)
	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "validate-v0-4-release-state: encode: %v\n", err)
			os.Exit(2)
		}
	case "text":
		writeTextReport(os.Stdout, report)
	default:
		fmt.Fprintf(os.Stderr, "validate-v0-4-release-state: unknown --format %q\n", *format)
		os.Exit(2)
	}
	if report.Status != "pass" {
		os.Exit(1)
	}
}

func buildReleaseState(expectedVersion, reportDir string) releaseStateReport {
	if expectedVersion == "" {
		expectedVersion = "v0.4.0"
	}
	report := releaseStateReport{
		Schema:          schemaV1,
		ExpectedVersion: expectedVersion,
		Scope:           "linux-x64-no-econet",
		ReportDir:       reportDir,
	}
	report.Version = currentVersion()
	if report.Version != expectedVersion {
		report.Issues = append(
			report.Issues,
			fmt.Sprintf("version is %q, want %q", report.Version, expectedVersion),
		)
	}

	entries, err := gitStatusEntries()
	if err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git status failed: %v", err))
	}
	report.Git = gitState{Clean: len(entries) == 0 && err == nil, Entries: entries}
	if len(entries) > 0 {
		report.Issues = append(
			report.Issues,
			fmt.Sprintf("git status has %d entries", len(entries)),
		)
	}

	for _, path := range requiredRepoArtifacts() {
		report.RequiredArtifacts = append(report.RequiredArtifacts, inspectArtifact(path))
	}
	if reportDir != "" {
		for _, path := range requiredReportArtifacts(reportDir) {
			report.RequiredArtifacts = append(report.RequiredArtifacts, inspectArtifact(path))
		}
	}
	for _, artifact := range report.RequiredArtifacts {
		if !artifact.Exists {
			report.Issues = append(report.Issues, "missing required artifact: "+artifact.Path)
		}
	}

	if len(report.Issues) == 0 {
		report.Status = "pass"
	} else {
		report.Status = "fail"
	}
	return report
}

func currentVersion() string {
	raw, err := os.ReadFile("compiler/internal/version/version.go")
	if err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "const CompilerVersion = ") {
				return strings.Trim(strings.TrimPrefix(line, "const CompilerVersion = "), `"`)
			}
		}
	}
	out, err := exec.Command("./tetra", "version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitStatusEntries() ([]gitStatusEntry, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	var entries []gitStatusEntry
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry := gitStatusEntry{Path: strings.TrimSpace(line)}
		if len(line) >= 3 {
			entry.Index = line[:1]
			entry.Worktree = line[1:2]
			entry.Path = strings.TrimSpace(line[3:])
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func requiredRepoArtifacts() []string {
	return []string{
		"docs/generated/manifest.json",
		"docs/release/v0_4/data/v0_4_0_scope_decisions.json",
		"docs/release/v0_4/v0_4_0_completion_audit.md",
		"docs/release/v0_4/v0_4_0_final_handoff.md",
		"docs/spec/flow/v0_4_scope.md",
		"reports/v0.4.0/features.json",
		"reports/v0.4.0/targets.json",
		"reports/v0.4.0/linux-host-smoke.json",
		"reports/v0.4.0/distributed-actors-linux-x64.json",
		"reports/v0.4.0/native-ui-linux-x64.json",
	}
}

func requiredReportArtifacts(reportDir string) []string {
	return []string{
		filepath.Join(reportDir, "artifacts", "features.json"),
		filepath.Join(reportDir, "artifacts", "targets.json"),
		filepath.Join(reportDir, "artifacts", "linux-host-smoke.json"),
		filepath.Join(reportDir, "artifacts", "memory-production-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "parallel-production-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "compiler-production-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "distributed-actors-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "native-ui-linux-x64.json"),
	}
}

func inspectArtifact(path string) artifactState {
	state := artifactState{Path: path}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return state
	}
	state.Exists = true
	state.Size = info.Size()
	return state
}

func writeTextReport(out *os.File, report releaseStateReport) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "schema: %s\n", report.Schema)
	fmt.Fprintf(&buf, "status: %s\n", report.Status)
	fmt.Fprintf(&buf, "version: %s\n", report.Version)
	fmt.Fprintf(&buf, "expected_version: %s\n", report.ExpectedVersion)
	fmt.Fprintf(&buf, "scope: %s\n", report.Scope)
	fmt.Fprintf(&buf, "git_clean: %t\n", report.Git.Clean)
	if report.ReportDir != "" {
		fmt.Fprintf(&buf, "report_dir: %s\n", report.ReportDir)
	}
	if len(report.Issues) > 0 {
		fmt.Fprintln(&buf, "issues:")
		for _, issue := range report.Issues {
			fmt.Fprintf(&buf, "- %s\n", issue)
		}
	}
	_, _ = out.Write(buf.Bytes())
}
