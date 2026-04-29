package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const releaseStateSchema = "tetra.release-state.v1alpha1"
const minimumReleaseGateSteps = 33
const defaultExpectedVersion = "v0.2.0"
const promotedMinorVersion = "v0.2.0"

var requiredReleaseArtifacts = []string{
	"docs/generated/manifest.json",
	"docs/generated/v1_0/README.md",
	"docs/generated/v1_0/release_gate_summary.json",
	"docs/generated/v1_0/release_gate_summary.md",
	"docs/generated/v1_0/test_all_full_summary.json",
	"docs/generated/v1_0/test_all_full_summary.md",
	"docs/generated/v1_0/api-diff/api-diff.json",
	"docs/generated/v1_0/api-diff/api-docs.md",
	"docs/generated/v1_0/reproducible-build.json",
	"docs/generated/v1_0/binary-size-thresholds.json",
	"docs/generated/v1_0/release-state.json",
	"docs/generated/v1_0/release-state.txt",
	"docs/generated/v1_0/targets.json",
	"docs/generated/v1_0/doctor.json",
	"docs/generated/v1_0/tetra-test-report.json",
	"docs/generated/v1_0/smoke-list.json",
	"docs/generated/v1_0/host-smoke.json",
	"docs/generated/v1_0/linux-smoke.json",
	"docs/generated/v1_0/macos-smoke.json",
	"docs/generated/v1_0/windows-smoke.json",
	"docs/generated/v1_0/wasm32-wasi-smoke.json",
	"docs/generated/v1_0/wasm32-web-smoke.json",
	"docs/generated/v1_0/wasi-smoke.build-only.json",
	"docs/generated/v1_0/wasi-smoke.json",
	"docs/generated/v1_0/web-ui-smoke.json",
	"docs/generated/v1_0/web-ui-smoke.dom.html",
	"docs/generated/v1_0/invalid-diagnostic.json",
	"docs/generated/v1_0/missing-effect-diagnostic.json",
	"docs/generated/v1_0/tabs-diagnostic.json",
	"docs/generated/v1_0/planned-actor-diagnostic.json",
	"docs/generated/v1_0/known_issues.md",
	"docs/generated/v1_0/artifact-hashes.json",
	"docs/generated/v1_0/test-all/summary.json",
	"docs/generated/v1_0/test-all/summary.md",
	"docs/generated/v1_0/test-all/lsp-smoke.json",
	"docs/generated/v1_0/test-all/tetra-docs.md",
}

type gitStatusEntry struct {
	Index  string `json:"index"`
	Work   string `json:"worktree"`
	Path   string `json:"path"`
	Rename string `json:"rename,omitempty"`
}

type gitStatusClassification struct {
	Entries                    []gitStatusEntry `json:"entries"`
	DirtyTracked               []string         `json:"dirty_tracked"`
	UntrackedReleaseArtifacts  []string         `json:"untracked_release_artifacts"`
	UntrackedNonReleaseEntries []string         `json:"untracked_non_release_entries,omitempty"`
}

type artifactCheck struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size,omitempty"`
}

type generatedArtifactsReport struct {
	Required []artifactCheck `json:"required"`
	Missing  []string        `json:"missing,omitempty"`
}

type gateEvidenceReport struct {
	SummaryPath        string `json:"summary_path"`
	Status             string `json:"status,omitempty"`
	ReleaseVersion     string `json:"release_version,omitempty"`
	ReleaseArtifact    string `json:"release_artifact,omitempty"`
	ReleaseGateCommand string `json:"release_gate_command,omitempty"`
	ReportDir          string `json:"report_dir,omitempty"`
	StepCount          int    `json:"step_count,omitempty"`
	FailedCount        int    `json:"failed_count,omitempty"`
	Error              string `json:"error,omitempty"`
}

type freshnessCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type releaseStateReport struct {
	Schema             string                   `json:"schema"`
	Status             string                   `json:"status"`
	Branch             string                   `json:"branch"`
	Version            string                   `json:"version"`
	ExpectedVersion    string                   `json:"expected_version"`
	ReportDir          string                   `json:"report_dir,omitempty"`
	Git                gitStatusClassification  `json:"git"`
	GeneratedArtifacts generatedArtifactsReport `json:"generated_artifacts"`
	Freshness          []freshnessCheck         `json:"freshness,omitempty"`
	LastGateEvidence   gateEvidenceReport       `json:"last_gate_evidence"`
	Issues             []string                 `json:"issues,omitempty"`
}

type releaseStateInputs struct {
	Branch          string
	Version         string
	ExpectedVersion string
	ReportDir       string
	GitStatus       []gitStatusEntry
	ReadFile        func(string) ([]byte, error)
	StatFile        func(string) (fileInfo, error)
	Freshness       []freshnessCheck
}

type fileInfo interface {
	Size() int64
}

type fakeFileInfo struct {
	size int64
}

func (f fakeFileInfo) Size() int64 { return f.size }

func errNotExist(path string) error {
	return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
}

func main() {
	var format string
	var repoRoot string
	var reportDir string
	var expectedVersion string
	flag.StringVar(&format, "format", "text", "output format: text or json")
	flag.StringVar(&repoRoot, "repo", ".", "repository root")
	flag.StringVar(&reportDir, "report-dir", "", "current release gate report directory")
	flag.StringVar(&expectedVersion, "expected-version", defaultExpectedVersion, "expected version boundary for release-state validation")
	flag.Parse()

	repoRoot = filepath.Clean(repoRoot)
	readFile := func(path string) ([]byte, error) {
		localPath := filepath.FromSlash(path)
		if filepath.IsAbs(localPath) {
			return os.ReadFile(localPath)
		}
		return os.ReadFile(filepath.Join(repoRoot, localPath))
	}
	statFile := func(path string) (fileInfo, error) {
		return os.Stat(filepath.Join(repoRoot, filepath.FromSlash(path)))
	}
	branch, branchErr := commandOutput(repoRoot, "git", "branch", "--show-current")
	version, versionErr := commandOutput(repoRoot, "./tetra", "version")
	statusRaw, statusErr := commandOutput(repoRoot, "git", "status", "--porcelain")

	inputs := releaseStateInputs{
		Branch:          strings.TrimSpace(branch),
		Version:         strings.TrimSpace(version),
		ExpectedVersion: strings.TrimSpace(expectedVersion),
		ReportDir:       reportDir,
		GitStatus:       parseGitStatus(statusRaw),
		ReadFile:        readFile,
		StatFile:        statFile,
		Freshness: []freshnessCheck{
			checkGeneratedManifestFreshness(repoRoot),
			checkArtifactHashManifest(repoRoot),
		},
	}
	report := buildReleaseStateReport(inputs)
	if branchErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git branch failed: %v", branchErr))
	}
	if versionErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("./tetra version failed: %v", versionErr))
	}
	if statusErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git status failed: %v", statusErr))
	}
	for _, fresh := range report.Freshness {
		if fresh.Status != "pass" {
			report.Issues = append(report.Issues, fmt.Sprintf("%s: %s", fresh.Name, fresh.Detail))
		}
	}
	report.Status = statusFromIssues(report.Issues)

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "text", "":
		fmt.Print(formatTextReport(report))
	default:
		fmt.Fprintf(os.Stderr, "unsupported --format %q\n", format)
		os.Exit(2)
	}
	if report.Status != "pass" {
		os.Exit(1)
	}
}

func buildReleaseStateReport(input releaseStateInputs) releaseStateReport {
	classified := classifyGitStatus(input.GitStatus)
	generated := inspectRequiredArtifacts(input.StatFile)
	issues := []string{}
	expectedVersion := strings.TrimSpace(input.ExpectedVersion)
	if expectedVersion == "" {
		expectedVersion = defaultExpectedVersion
	}
	gate := inspectLastGateEvidence(input.ReadFile, expectedVersion, input.ReportDir)
	if input.Version == "" {
		issues = append(issues, "version is missing")
	} else if input.Version != expectedVersion {
		issues = append(issues, fmt.Sprintf("version %q is not %q", input.Version, expectedVersion))
	}
	if expectedVersion == promotedMinorVersion && strings.TrimSpace(input.ReportDir) == "" {
		issues = append(issues, "report-dir is required for v0.2.0 release validation")
	}
	if len(classified.DirtyTracked) > 0 {
		issues = append(issues, fmt.Sprintf("dirty tracked files detected: %d", len(classified.DirtyTracked)))
	}
	if len(classified.UntrackedNonReleaseEntries) > 0 {
		issues = append(issues, fmt.Sprintf("untracked non-release entries detected: %d", len(classified.UntrackedNonReleaseEntries)))
	}
	for _, missing := range generated.Missing {
		issues = append(issues, "missing required release artifact: "+missing)
	}
	if gate.Error != "" {
		issues = append(issues, "last gate evidence: "+gate.Error)
	} else if gate.Status != "pass" {
		issues = append(issues, fmt.Sprintf("last gate evidence status is %q", gate.Status))
	} else if gate.FailedCount != 0 {
		issues = append(issues, fmt.Sprintf("last gate evidence has %d failed step(s)", gate.FailedCount))
	} else if gate.StepCount < minimumReleaseGateSteps {
		issues = append(issues, fmt.Sprintf("last gate evidence has %d step(s), want at least %d", gate.StepCount, minimumReleaseGateSteps))
	}
	if expectedVersion == promotedMinorVersion {
		expectedArtifact := expectedReleaseArtifact(expectedVersion)
		expectedGateScript := expectedReleaseGateScript(expectedVersion)
		if gate.ReleaseVersion != expectedVersion {
			issues = append(issues, fmt.Sprintf("last gate evidence release_version is %q, want %q", gate.ReleaseVersion, expectedVersion))
		}
		if gate.ReleaseArtifact != expectedArtifact {
			issues = append(issues, fmt.Sprintf("last gate evidence release_artifact is %q, want %q", gate.ReleaseArtifact, expectedArtifact))
		}
		if !strings.Contains(gate.ReleaseGateCommand, expectedGateScript) {
			issues = append(issues, fmt.Sprintf("last gate evidence release_gate_command %q does not reference %q", gate.ReleaseGateCommand, expectedGateScript))
		}
		if strings.TrimSpace(gate.ReportDir) == "" {
			issues = append(issues, "last gate evidence report_dir is missing")
		} else if strings.TrimSpace(input.ReportDir) != "" && filepath.Clean(gate.ReportDir) != filepath.Clean(input.ReportDir) {
			issues = append(issues, fmt.Sprintf("last gate evidence report_dir is %q, want %q", gate.ReportDir, input.ReportDir))
		}
	}
	return releaseStateReport{
		Schema:             releaseStateSchema,
		Status:             statusFromIssues(issues),
		Branch:             input.Branch,
		Version:            input.Version,
		ExpectedVersion:    expectedVersion,
		ReportDir:          input.ReportDir,
		Git:                classified,
		GeneratedArtifacts: generated,
		Freshness:          input.Freshness,
		LastGateEvidence:   gate,
		Issues:             issues,
	}
}

func statusFromIssues(issues []string) string {
	if len(issues) > 0 {
		return "fail"
	}
	return "pass"
}

func parseGitStatus(raw string) []gitStatusEntry {
	var entries []gitStatusEntry
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) < 4 {
			continue
		}
		entry := gitStatusEntry{
			Index: line[0:1],
			Work:  line[1:2],
			Path:  strings.TrimSpace(line[3:]),
		}
		if strings.Contains(entry.Path, " -> ") {
			parts := strings.SplitN(entry.Path, " -> ", 2)
			entry.Rename = parts[0]
			entry.Path = parts[1]
		}
		entries = append(entries, entry)
	}
	return entries
}

func classifyGitStatus(entries []gitStatusEntry) gitStatusClassification {
	out := gitStatusClassification{Entries: entries}
	for _, entry := range entries {
		if entry.Index == "?" && entry.Work == "?" {
			if isReleaseArtifactPath(entry.Path) {
				out.UntrackedReleaseArtifacts = append(out.UntrackedReleaseArtifacts, entry.Path)
			} else {
				out.UntrackedNonReleaseEntries = append(out.UntrackedNonReleaseEntries, entry.Path)
			}
			continue
		}
		out.DirtyTracked = append(out.DirtyTracked, entry.Path)
	}
	sort.Strings(out.DirtyTracked)
	sort.Strings(out.UntrackedReleaseArtifacts)
	sort.Strings(out.UntrackedNonReleaseEntries)
	return out
}

func isReleaseArtifactPath(path string) bool {
	slash := filepath.ToSlash(path)
	return strings.HasPrefix(slash, "docs/generated/v1_0/") ||
		strings.HasPrefix(slash, "docs/baselines/") ||
		strings.HasPrefix(slash, "docs/release/")
}

func inspectRequiredArtifacts(statFile func(string) (fileInfo, error)) generatedArtifactsReport {
	report := generatedArtifactsReport{}
	for _, path := range requiredReleaseArtifacts {
		info, err := statFile(path)
		check := artifactCheck{Path: path}
		if err == nil {
			check.Exists = true
			check.Size = info.Size()
		} else if errors.Is(err, fs.ErrNotExist) {
			report.Missing = append(report.Missing, path)
		} else {
			report.Missing = append(report.Missing, path+" ("+err.Error()+")")
		}
		report.Required = append(report.Required, check)
	}
	return report
}

func inspectLastGateEvidence(readFile func(string) ([]byte, error), expectedVersion string, reportDir string) gateEvidenceReport {
	path := releaseGateSummaryPath(expectedVersion, reportDir)
	report := gateEvidenceReport{SummaryPath: path}
	raw, err := readFile(path)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	var summary struct {
		Status             string `json:"status"`
		ReleaseVersion     string `json:"release_version"`
		ReleaseArtifact    string `json:"release_artifact"`
		ReleaseGateCommand string `json:"release_gate_command"`
		ReportDir          string `json:"report_dir"`
		StepCount          int    `json:"step_count"`
		FailedCount        int    `json:"failed_count"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Status = summary.Status
	report.ReleaseVersion = summary.ReleaseVersion
	report.ReleaseArtifact = summary.ReleaseArtifact
	report.ReleaseGateCommand = summary.ReleaseGateCommand
	report.ReportDir = summary.ReportDir
	report.StepCount = summary.StepCount
	report.FailedCount = summary.FailedCount
	return report
}

func releaseGateSummaryPath(expectedVersion string, reportDir string) string {
	if expectedVersion == promotedMinorVersion && strings.TrimSpace(reportDir) != "" {
		return filepath.ToSlash(filepath.Join(reportDir, "summary.json"))
	}
	return "docs/generated/v1_0/release_gate_summary.json"
}

func expectedReleaseArtifact(version string) string {
	return "tetra.release." + strings.ReplaceAll(version, ".", "_") + ".gate-report.v1"
}

func expectedReleaseGateScript(version string) string {
	return "scripts/release_" + strings.ReplaceAll(version, ".", "_") + "_gate.sh"
}

func checkGeneratedManifestFreshness(repoRoot string) freshnessCheck {
	tmp, err := os.CreateTemp("", "tetra-manifest-*.json")
	if err != nil {
		return freshnessCheck{Name: "docs/generated/manifest.json", Status: "fail", Detail: err.Error()}
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)
	cmd := exec.Command("go", "run", "./tools/cmd/gen-manifest", "-o", tmpPath)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return freshnessCheck{Name: "docs/generated/manifest.json", Status: "fail", Detail: strings.TrimSpace(string(out)) + err.Error()}
	}
	current, err := os.ReadFile(filepath.Join(repoRoot, "docs/generated/manifest.json"))
	if err != nil {
		return freshnessCheck{Name: "docs/generated/manifest.json", Status: "fail", Detail: err.Error()}
	}
	generated, err := os.ReadFile(tmpPath)
	if err != nil {
		return freshnessCheck{Name: "docs/generated/manifest.json", Status: "fail", Detail: err.Error()}
	}
	if !bytes.Equal(bytes.TrimSpace(current), bytes.TrimSpace(generated)) {
		return freshnessCheck{Name: "docs/generated/manifest.json", Status: "fail", Detail: "manifest differs from generator output"}
	}
	return freshnessCheck{Name: "docs/generated/manifest.json", Status: "pass"}
}

func checkArtifactHashManifest(repoRoot string) freshnessCheck {
	cmd := exec.Command("go", "run", "./tools/cmd/validate-artifact-hashes", "--manifest", "docs/generated/v1_0/artifact-hashes.json")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		} else {
			detail += ": " + err.Error()
		}
		return freshnessCheck{Name: "docs/generated/v1_0/artifact-hashes.json", Status: "fail", Detail: detail}
	}
	return freshnessCheck{Name: "docs/generated/v1_0/artifact-hashes.json", Status: "pass"}
}

func commandOutput(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	raw, err := cmd.CombinedOutput()
	return string(raw), err
}

func formatTextReport(report releaseStateReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "status: %s\n", report.Status)
	fmt.Fprintf(&b, "branch: %s\n", valueOrUnknown(report.Branch))
	fmt.Fprintf(&b, "version: %s\n", valueOrUnknown(report.Version))
	fmt.Fprintf(&b, "expected_version: %s\n", valueOrUnknown(report.ExpectedVersion))
	if report.ReportDir != "" {
		fmt.Fprintf(&b, "report_dir: %s\n", report.ReportDir)
	}
	fmt.Fprintf(&b, "dirty tracked files: %d\n", len(report.Git.DirtyTracked))
	fmt.Fprintf(&b, "untracked release artifacts: %d\n", len(report.Git.UntrackedReleaseArtifacts))
	fmt.Fprintf(&b, "required artifacts: %d\n", len(report.GeneratedArtifacts.Required))
	fmt.Fprintf(&b, "missing artifacts: %d\n", len(report.GeneratedArtifacts.Missing))
	fmt.Fprintf(&b, "last gate evidence: %s (%d failed of %d steps, %s)\n", valueOrUnknown(report.LastGateEvidence.Status), report.LastGateEvidence.FailedCount, report.LastGateEvidence.StepCount, valueOrUnknown(report.LastGateEvidence.SummaryPath))
	if report.LastGateEvidence.ReleaseVersion != "" {
		fmt.Fprintf(&b, "last gate release_version: %s\n", report.LastGateEvidence.ReleaseVersion)
	}
	for _, fresh := range report.Freshness {
		if fresh.Detail == "" {
			fmt.Fprintf(&b, "freshness %s: %s\n", fresh.Name, fresh.Status)
		} else {
			fmt.Fprintf(&b, "freshness %s: %s (%s)\n", fresh.Name, fresh.Status, fresh.Detail)
		}
	}
	if len(report.Issues) > 0 {
		fmt.Fprintf(&b, "issues:\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(&b, "- %s\n", issue)
		}
	}
	return b.String()
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "<unknown>"
	}
	return value
}
