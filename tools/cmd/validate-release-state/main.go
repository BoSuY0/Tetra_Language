package main

import (
	"bytes"
	"crypto/sha256"
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
	"time"
)

const releaseStateSchema = "tetra.release-state.v1alpha1"
const defaultExpectedVersion = "v0.3.0"

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
	"docs/generated/v1_0/wasm32-wasi-artifact-smoke.json",
	"docs/generated/v1_0/wasm32-web-artifact-smoke.json",
	"docs/generated/v1_0/wasi-smoke.artifact.json",
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
	StartedAt          string `json:"started_at,omitempty"`
	EndedAt            string `json:"ended_at,omitempty"`
	ReportDir          string `json:"report_dir,omitempty"`
	StepCount          int    `json:"step_count,omitempty"`
	FailedCount        int    `json:"failed_count,omitempty"`
	Error              string `json:"error,omitempty"`
}

type runtimeExecutionEvidenceReport struct {
	Required []runtimeExecutionEvidenceCheck `json:"required,omitempty"`
	Missing  []string                        `json:"missing,omitempty"`
}

type securityReviewEvidenceReport struct {
	Path             string   `json:"path,omitempty"`
	HashPath         string   `json:"hash_path,omitempty"`
	Status           string   `json:"status,omitempty"`
	ValidatorCommand string   `json:"validator_command,omitempty"`
	Missing          []string `json:"missing,omitempty"`
	Issues           []string `json:"issues,omitempty"`
}

type runtimeExecutionEvidenceCheck struct {
	Target          string   `json:"target"`
	Path            string   `json:"path"`
	Status          string   `json:"status"`
	EvidenceCommand string   `json:"evidence_command,omitempty"`
	Host            string   `json:"host,omitempty"`
	Issues          []string `json:"issues,omitempty"`
}

type freshnessCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type releaseStateReport struct {
	Schema             string                         `json:"schema"`
	Status             string                         `json:"status"`
	Branch             string                         `json:"branch"`
	Version            string                         `json:"version"`
	ExpectedVersion    string                         `json:"expected_version"`
	ReportDir          string                         `json:"report_dir,omitempty"`
	Git                gitStatusClassification        `json:"git"`
	GeneratedArtifacts generatedArtifactsReport       `json:"generated_artifacts"`
	RuntimeExecution   runtimeExecutionEvidenceReport `json:"runtime_execution,omitempty"`
	SecurityReview     securityReviewEvidenceReport   `json:"security_review,omitempty"`
	Freshness          []freshnessCheck               `json:"freshness,omitempty"`
	LastGateEvidence   gateEvidenceReport             `json:"last_gate_evidence"`
	Issues             []string                       `json:"issues,omitempty"`
}

type releaseStateInputs struct {
	Branch          string
	Version         string
	ExpectedVersion string
	ReportDir       string
	GitHead         string
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
	gitHead, gitHeadErr := commandOutput(repoRoot, "git", "rev-parse", "--short", "HEAD")
	version, versionErr := commandOutput(repoRoot, "./tetra", "version")
	statusRaw, statusErr := commandOutput(repoRoot, "git", "status", "--porcelain")
	freshness := []freshnessCheck{
		checkGeneratedManifestFreshness(repoRoot),
		checkArtifactHashManifest(repoRoot, reportDir),
		checkSmokeEvidenceFreshness(repoRoot, reportDir),
	}

	inputs := releaseStateInputs{
		Branch:          strings.TrimSpace(branch),
		Version:         strings.TrimSpace(version),
		ExpectedVersion: strings.TrimSpace(expectedVersion),
		ReportDir:       reportDir,
		GitHead:         strings.TrimSpace(gitHead),
		GitStatus:       parseGitStatus(statusRaw),
		ReadFile:        readFile,
		StatFile:        statFile,
		Freshness:       freshness,
	}
	report := buildReleaseStateReport(inputs)
	if branchErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git branch failed: %v", branchErr))
	}
	if gitHeadErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git rev-parse HEAD failed: %v", gitHeadErr))
	}
	if versionErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("./tetra version failed: %v", versionErr))
	}
	if statusErr != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("git status failed: %v", statusErr))
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
	expectedVersion := strings.TrimSpace(input.ExpectedVersion)
	if expectedVersion == "" {
		expectedVersion = defaultExpectedVersion
	}
	generated := inspectRequiredArtifacts(input.StatFile, requiredReleaseArtifactPaths(expectedVersion, input.ReportDir))
	issues := []string{}
	freshness := append([]freshnessCheck{}, input.Freshness...)
	if expectedVersion == "v1.0.0" {
		freshness = append(freshness, checkV1EvidenceMetadata(input.ReadFile, expectedVersion, input.GitHead, input.ReportDir))
	}
	runtimeExecution := inspectRuntimeExecutionEvidence(input.ReadFile, expectedVersion, input.ReportDir, input.GitHead)
	securityReview := inspectSecurityReviewEvidence(input.ReadFile, expectedVersion, input.ReportDir, input.GitHead)
	gate := inspectLastGateEvidence(input.ReadFile, expectedVersion, input.ReportDir)
	if input.Version == "" {
		issues = append(issues, "version is missing")
	} else if input.Version != expectedVersion {
		issues = append(issues, fmt.Sprintf("version %q is not %q", input.Version, expectedVersion))
	}
	if isPromotedReleaseVersion(expectedVersion) && strings.TrimSpace(input.ReportDir) == "" {
		issues = append(issues, fmt.Sprintf("report-dir is required for %s release validation", expectedVersion))
	}
	for _, missing := range generated.Missing {
		issues = append(issues, "missing required release artifact: "+missing)
	}
	for _, missing := range runtimeExecution.Missing {
		issues = append(issues, "missing required runtime execution evidence: "+missing)
	}
	for _, check := range runtimeExecution.Required {
		for _, issue := range check.Issues {
			issues = append(issues, "runtime execution evidence: "+issue)
		}
	}
	for _, missing := range securityReview.Missing {
		issues = append(issues, "missing required security review evidence: "+missing)
	}
	for _, issue := range securityReview.Issues {
		issues = append(issues, "security review evidence: "+issue)
	}
	if gate.Error != "" {
		issues = append(issues, "last gate evidence: "+gate.Error)
	} else if gate.Status != "pass" {
		issues = append(issues, fmt.Sprintf("last gate evidence status is %q", gate.Status))
	} else if gate.FailedCount != 0 {
		issues = append(issues, fmt.Sprintf("last gate evidence has %d failed step(s)", gate.FailedCount))
	} else if minimumSteps := minimumReleaseGateSteps(expectedVersion); gate.StepCount < minimumSteps {
		issues = append(issues, fmt.Sprintf("last gate evidence has %d step(s), want at least %d", gate.StepCount, minimumSteps))
	}
	if isPromotedReleaseVersion(expectedVersion) {
		expectedArtifact := expectedReleaseArtifact(expectedVersion)
		expectedGateCommand := expectedReleaseGateCommand(expectedVersion)
		if gate.ReleaseVersion != expectedVersion {
			issues = append(issues, fmt.Sprintf("last gate evidence release_version is %q, want %q", gate.ReleaseVersion, expectedVersion))
		}
		if gate.ReleaseArtifact != expectedArtifact {
			issues = append(issues, fmt.Sprintf("last gate evidence release_artifact is %q, want %q", gate.ReleaseArtifact, expectedArtifact))
		}
		if gate.ReleaseGateCommand != expectedGateCommand {
			issues = append(issues, fmt.Sprintf("last gate evidence release_gate_command is %q, want %q", gate.ReleaseGateCommand, expectedGateCommand))
		}
		if err := validateGateEvidenceTimestamps(gate); err != nil {
			issues = append(issues, "last gate evidence "+err.Error())
		}
		if strings.TrimSpace(gate.ReportDir) == "" {
			issues = append(issues, "last gate evidence report_dir is missing")
		} else if strings.TrimSpace(input.ReportDir) != "" && filepath.Clean(gate.ReportDir) != filepath.Clean(input.ReportDir) {
			issues = append(issues, fmt.Sprintf("last gate evidence report_dir is %q, want %q", gate.ReportDir, input.ReportDir))
		}
	}
	for _, fresh := range freshness {
		if fresh.Status != "pass" {
			issues = append(issues, fmt.Sprintf("%s: %s", fresh.Name, fresh.Detail))
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
		RuntimeExecution:   runtimeExecution,
		SecurityReview:     securityReview,
		Freshness:          freshness,
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

func inspectRequiredArtifacts(statFile func(string) (fileInfo, error), paths []string) generatedArtifactsReport {
	report := generatedArtifactsReport{}
	for _, path := range paths {
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

func requiredReleaseArtifactPaths(expectedVersion string, reportDir string) []string {
	if expectedVersion == "v1.0.0" && strings.TrimSpace(reportDir) != "" {
		return v1ReportDirRequiredArtifacts(reportDir)
	}
	return requiredReleaseArtifacts
}

func v1ReportDirRequiredArtifacts(reportDir string) []string {
	prefix := filepath.ToSlash(filepath.Clean(reportDir))
	join := func(parts ...string) string {
		all := append([]string{prefix}, parts...)
		return filepath.ToSlash(filepath.Join(all...))
	}
	return []string{
		join("summary.json"),
		join("summary.md"),
		join("artifacts", "manifest.json"),
		join("artifacts", "artifact-hashes.json"),
		join("artifacts", "known_issues.md"),
		join("artifacts", "security-review.md"),
		join("artifacts", "security-review.md.sha256"),
		join("artifacts", "reproducible-build.json"),
		join("artifacts", "binary-size-thresholds.json"),
		join("artifacts", "performance-regression.json"),
		join("artifacts", "targets.json"),
		join("artifacts", "doctor.json"),
		join("artifacts", "tetra-test-report.json"),
		join("artifacts", "smoke-list.json"),
		join("artifacts", "host-smoke.json"),
		join("artifacts", "linux-smoke.json"),
		join("artifacts", "macos-smoke.json"),
		join("artifacts", "windows-smoke.json"),
		join("artifacts", "wasm32-wasi-artifact-smoke.json"),
		join("artifacts", "wasm32-web-artifact-smoke.json"),
		join("artifacts", "wasi-smoke.artifact.json"),
		join("artifacts", "wasi-smoke.json"),
		join("artifacts", "web-ui-smoke.json"),
		join("artifacts", "api-diff", "api-docs.md"),
		join("artifacts", "api-diff", "api-diff.json"),
		join("artifacts", "tetra-docs.md"),
		join("artifacts", "test-all", "summary.json"),
		join("artifacts", "test-all", "summary.md"),
	}
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
		StartedAt          string `json:"started_at"`
		EndedAt            string `json:"ended_at"`
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
	report.StartedAt = summary.StartedAt
	report.EndedAt = summary.EndedAt
	report.ReportDir = summary.ReportDir
	report.StepCount = summary.StepCount
	report.FailedCount = summary.FailedCount
	return report
}

func inspectRuntimeExecutionEvidence(readFile func(string) ([]byte, error), expectedVersion string, reportDir string, currentGitHead string) runtimeExecutionEvidenceReport {
	if expectedVersion != "v0.3.0" {
		return runtimeExecutionEvidenceReport{}
	}
	report := runtimeExecutionEvidenceReport{}
	for _, target := range []string{"macos-x64", "windows-x64"} {
		path := runtimeExecutionEvidencePath(reportDir, target)
		check := runtimeExecutionEvidenceCheck{Target: target, Path: path, Status: "pass", EvidenceCommand: runtimeExecutionEvidenceCommand(target, path)}
		raw, err := readFile(path)
		if err != nil {
			report.Missing = append(report.Missing, path)
			check.Status = "missing"
			check.Issues = append(check.Issues, fmt.Sprintf("%s read failed: %v", filepath.Base(path), err))
			report.Required = append(report.Required, check)
			continue
		}
		smoke, smokeIssues := validateRuntimeSmokeReport(raw, path, target, expectedVersion, currentGitHead)
		check.Host = smoke.Host
		check.Issues = append(check.Issues, smokeIssues...)
		if len(check.Issues) > 0 {
			check.Status = "fail"
		}
		report.Required = append(report.Required, check)
	}
	return report
}

func inspectSecurityReviewEvidence(readFile func(string) ([]byte, error), expectedVersion string, reportDir string, currentGitHead string) securityReviewEvidenceReport {
	if !requiresSecurityReviewEvidence(expectedVersion) {
		return securityReviewEvidenceReport{}
	}
	path := securityReviewEvidencePath(reportDir)
	hashPath := path + ".sha256"
	report := securityReviewEvidenceReport{Path: path, HashPath: hashPath, Status: "pass", ValidatorCommand: securityReviewValidatorCommand(expectedVersion, path)}
	raw, err := readFile(path)
	hashRaw, hashErr := readFile(hashPath)
	if err != nil && hashErr != nil && strings.TrimSpace(reportDir) != "" {
		report.Status = "deferred"
		return report
	}
	if err != nil {
		report.Missing = append(report.Missing, path)
	}
	if hashErr != nil {
		report.Missing = append(report.Missing, hashPath)
	}
	if err == nil && hashErr == nil {
		report.Issues = append(report.Issues, validateSecurityReviewEvidence(raw, hashRaw, path, hashPath, expectedVersion, reportDir, currentGitHead)...)
	}
	if len(report.Missing) > 0 || len(report.Issues) > 0 {
		report.Status = "fail"
	}
	return report
}

func requiresSecurityReviewEvidence(version string) bool {
	switch version {
	case "v0.3.0", "v1.0.0":
		return true
	default:
		return false
	}
}

func securityReviewEvidencePath(reportDir string) string {
	if strings.TrimSpace(reportDir) == "" {
		return filepath.ToSlash(filepath.Join("artifacts", "security-review.md"))
	}
	return filepath.ToSlash(filepath.Join(reportDir, "artifacts", "security-review.md"))
}

func securityReviewValidatorCommand(version string, signoffPath string) string {
	return "bash " + securityReviewValidatorScript(version) + " --signoff " + signoffPath
}

func securityReviewValidatorScript(version string) string {
	if version == "v1.0.0" {
		return "scripts/release/v1_0/security-review.sh"
	}
	if version == "v0.3.0" {
		return "scripts/release/v0_3_0/security-review.sh"
	}
	return "scripts/release_" + strings.ReplaceAll(version, ".", "_") + "_security_review.sh"
}

func validateSecurityReviewEvidence(raw []byte, hashRaw []byte, path string, hashPath string, expectedVersion string, reportDir string, currentGitHead string) []string {
	var issues []string
	label := filepath.Base(path)
	text := string(raw)
	expectedDecision := fmt.Sprintf("approved for %s release", expectedVersion)
	decision := securityReviewLineValue(text, "Decision:")
	if decision != expectedDecision {
		issues = append(issues, fmt.Sprintf("%s decision is not an approval for %s", label, expectedVersion))
	}
	if strings.Contains(text, "missing-security-signoff") || strings.Contains(strings.ToLower(text), "blocked: missing human security signoff") {
		issues = append(issues, fmt.Sprintf("%s contains missing security signoff placeholder text", label))
	}
	if strings.Contains(text, "TODO") || strings.Contains(text, "TBD") {
		issues = append(issues, fmt.Sprintf("%s contains unresolved placeholder text", label))
	}
	if strings.Contains(text, "sha256:0000000000000000000000000000000000000000000000000000000000000000") {
		issues = append(issues, fmt.Sprintf("%s contains placeholder artifact hash", label))
	}
	reviewedCommit := securityReviewLineValue(text, "Reviewed commit:")
	if !gitHeadMatches(reviewedCommit, currentGitHead) {
		issues = append(issues, fmt.Sprintf("%s reviewed commit is %q, want %q", label, reviewedCommit, currentGitHead))
	}
	reviewReportDir := securityReviewLineValue(text, "Report directory:")
	if strings.TrimSpace(reviewReportDir) == "" {
		issues = append(issues, fmt.Sprintf("%s report directory is missing", label))
	} else if strings.TrimSpace(reportDir) != "" && filepath.Clean(reviewReportDir) != filepath.Clean(reportDir) {
		issues = append(issues, fmt.Sprintf("%s report directory is %q, want %q", label, reviewReportDir, reportDir))
	}
	for _, section := range []string{"## Evidence Commands", "## Artifact Hashes", "## Residual Risks"} {
		if !strings.Contains(text, section) {
			issues = append(issues, fmt.Sprintf("%s missing %s section", label, strings.TrimPrefix(section, "## ")))
		}
	}
	issues = append(issues, validateSecurityReviewDetachedHash(raw, hashRaw, hashPath)...)
	return issues
}

func validateSecurityReviewDetachedHash(raw []byte, hashRaw []byte, hashPath string) []string {
	line := strings.TrimSpace(string(hashRaw))
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return []string{fmt.Sprintf("%s must contain '<sha256>  artifacts/security-review.md'", filepath.Base(hashPath))}
	}
	if fields[1] != "artifacts/security-review.md" {
		return []string{fmt.Sprintf("%s path is %q, want %q", filepath.Base(hashPath), fields[1], "artifacts/security-review.md")}
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(raw))
	if fields[0] != expected {
		return []string{fmt.Sprintf("%s hash mismatch: got %s, want %s", filepath.Base(hashPath), fields[0], expected)}
	}
	return nil
}

func securityReviewLineValue(text string, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func runtimeExecutionEvidencePath(reportDir string, target string) string {
	filename := strings.TrimSuffix(target, "-x64") + "-runtime-smoke.json"
	if strings.TrimSpace(reportDir) == "" {
		return filepath.ToSlash(filepath.Join("artifacts", filename))
	}
	return filepath.ToSlash(filepath.Join(reportDir, "artifacts", filename))
}

func runtimeExecutionEvidenceCommand(target string, reportPath string) string {
	return "./tetra smoke --target " + target + " --run=true --report " + reportPath
}

type runtimeSmokeReport struct {
	Timestamp    string                   `json:"timestamp"`
	Target       string                   `json:"target"`
	BuildOnly    bool                     `json:"build_only,omitempty"`
	Runner       string                   `json:"runner,omitempty"`
	Host         string                   `json:"host"`
	Unsupported  bool                     `json:"unsupported,omitempty"`
	Version      string                   `json:"version"`
	GitHead      string                   `json:"git_head,omitempty"`
	IslandsDebug bool                     `json:"islands_debug"`
	Total        int                      `json:"total"`
	Passed       int                      `json:"passed"`
	Failed       int                      `json:"failed"`
	Cases        []runtimeSmokeCaseReport `json:"cases"`
}

type runtimeSmokeCaseReport struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	OutPath      string `json:"out_path,omitempty"`
	ExpectedExit int    `json:"expected_exit"`
	ActualExit   *int   `json:"actual_exit,omitempty"`
	Ran          bool   `json:"ran"`
	Pass         bool   `json:"pass"`
	Unsupported  bool   `json:"unsupported,omitempty"`
	Error        string `json:"error,omitempty"`
}

func validateRuntimeSmokeReport(raw []byte, path string, expectedTarget string, expectedVersion string, currentGitHead string) (runtimeSmokeReport, []string) {
	var issues []string
	var report runtimeSmokeReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return runtimeSmokeReport{}, []string{fmt.Sprintf("%s is not valid JSON: %v", filepath.Base(path), err)}
	}
	label := filepath.Base(path)
	if _, err := time.Parse(time.RFC3339, report.Timestamp); err != nil {
		issues = append(issues, fmt.Sprintf("%s timestamp is not RFC3339: %v", label, err))
	}
	if report.Target != expectedTarget {
		issues = append(issues, fmt.Sprintf("%s target is %q, want %q", label, report.Target, expectedTarget))
	}
	if report.Host != expectedTarget {
		issues = append(issues, fmt.Sprintf("%s host is %q, want %q", label, report.Host, expectedTarget))
	}
	if report.BuildOnly {
		issues = append(issues, fmt.Sprintf("%s build_only is true, want false", label))
	}
	if report.Runner != "" {
		issues = append(issues, fmt.Sprintf("%s runner is %q, want empty host-native runtime", label, report.Runner))
	}
	if report.Unsupported {
		issues = append(issues, fmt.Sprintf("%s unsupported is true, want false", label))
	}
	if report.Version != expectedVersion {
		issues = append(issues, fmt.Sprintf("%s version is %q, want %q", label, report.Version, expectedVersion))
	}
	if !gitHeadMatches(report.GitHead, currentGitHead) {
		issues = append(issues, fmt.Sprintf("%s git_head is %q, want %q", label, report.GitHead, currentGitHead))
	}
	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	total := len(report.Cases)
	failed := total - passed
	if report.Total != total || report.Passed != passed || report.Failed != failed {
		issues = append(issues, fmt.Sprintf("%s counts mismatch: got total=%d passed=%d failed=%d, computed total=%d passed=%d failed=%d", label, report.Total, report.Passed, report.Failed, total, passed, failed))
	}
	byName := map[string]runtimeSmokeCaseReport{}
	for _, c := range report.Cases {
		if c.Name == "" {
			issues = append(issues, fmt.Sprintf("%s contains a case with empty name", label))
			continue
		}
		if _, ok := byName[c.Name]; ok {
			issues = append(issues, fmt.Sprintf("%s duplicate case %s", label, c.Name))
			continue
		}
		byName[c.Name] = c
	}
	for _, name := range requiredRuntimeSmokeCases() {
		c, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s missing required runtime case %s", label, name))
			continue
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("%s case %s did not run", label, name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("%s case %s did not pass", label, name))
		}
		if c.Unsupported {
			issues = append(issues, fmt.Sprintf("%s case %s is marked unsupported", label, name))
		}
		if c.ActualExit == nil {
			issues = append(issues, fmt.Sprintf("%s case %s missing actual_exit", label, name))
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(issues, fmt.Sprintf("%s case %s actual_exit is %d, want %d", label, name, *c.ActualExit, c.ExpectedExit))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("%s case %s has error text", label, name))
		}
	}
	return report, issues
}

func requiredRuntimeSmokeCases() []string {
	return []string{
		"actors_pingpong",
		"actor_sleep_pingpong",
		"task_smoke",
		"time_sleep_smoke",
		"task_sleep_deadline_smoke",
		"task_join_wait_smoke",
		"deadline_aware_waits_smoke",
		"wait_composition_smoke",
	}
}

func releaseGateSummaryPath(expectedVersion string, reportDir string) string {
	if strings.TrimSpace(reportDir) != "" {
		return filepath.ToSlash(filepath.Join(reportDir, "summary.json"))
	}
	return "docs/generated/v1_0/release_gate_summary.json"
}

func isPromotedReleaseVersion(version string) bool {
	switch version {
	case "v0.2.0", "v0.3.0", "v0.4.0", "v1.0.0":
		return true
	default:
		return false
	}
}

func minimumReleaseGateSteps(version string) int {
	switch version {
	case "v0.3.0":
		return 8
	case "v1.0.0":
		return 32
	default:
		return 33
	}
}

func expectedReleaseArtifact(version string) string {
	if version == "v1.0.0" {
		return "tetra.release.v1_0.gate-report.v1"
	}
	return "tetra.release." + strings.ReplaceAll(version, ".", "_") + ".gate-report.v1"
}

func expectedReleaseGateScript(version string) string {
	switch version {
	case "v0.1.1":
		return "scripts/release/v0_1_1/gate.sh"
	case "v0.1.2":
		return "scripts/release/v0_1_2/gate.sh"
	case "v0.1.3":
		return "scripts/release/v0_1_3/gate.sh"
	case "v0.2.0":
		return "scripts/release/v0_2_0/gate.sh"
	case "v0.3.0":
		return "scripts/release/v0_3_0/gate.sh"
	case "v0.4.0":
		return "scripts/release/v0_4_0/gate.sh"
	case "v1.0.0":
		return "scripts/release/v1_0/gate.sh"
	}
	return "scripts/release_" + strings.ReplaceAll(version, ".", "_") + "_gate.sh"
}

func expectedReleaseGateCommand(version string) string {
	return "bash " + expectedReleaseGateScript(version)
}

type generatedEvidenceRule struct {
	Path            string
	Schema          string
	VersionField    string
	RequireGitHead  bool
	TimestampFields []string
	StartEndFields  bool
	ReleaseSummary  bool
}

type validationCommand struct {
	Name string
	Args []string
}

var v1GeneratedEvidenceRules = []generatedEvidenceRule{
	{Path: "docs/generated/manifest.json", VersionField: "compiler_version"},
	{Path: "docs/generated/v1_0/manifest.json", VersionField: "compiler_version"},
	{Path: "docs/generated/v1_0/binary-size-thresholds.json", Schema: "tetra.binary-size-thresholds.v1alpha1", VersionField: "compiler_version"},
	{Path: "docs/generated/v1_0/reproducible-build.json", Schema: "tetra.reproducible-build-proof.v1alpha1", VersionField: "compiler_version"},
	{Path: "docs/generated/v1_0/performance-regression.json", Schema: "tetra.performance-regression.v1", RequireGitHead: true},
	{Path: "docs/generated/v1_0/api-diff/api-diff.json", Schema: "tetra.api.diff.v1alpha1"},
	{Path: "docs/generated/v1_0/release-state.json", Schema: releaseStateSchema, VersionField: "version"},
	{Path: "docs/generated/v1_0/release_gate_summary.json", StartEndFields: true, ReleaseSummary: true},
	{Path: "docs/generated/v1_0/test_all_full_summary.json", StartEndFields: true},
	{Path: "docs/generated/v1_0/test-all/summary.json", StartEndFields: true},
	{Path: "docs/generated/v1_0/host-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/linux-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/macos-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/windows-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/wasm32-wasi-artifact-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/wasm32-web-artifact-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/wasi-smoke.artifact.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/wasi-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/test-all/host-smoke.json", VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
	{Path: "docs/generated/v1_0/web-ui-smoke.json", Schema: "tetra.web-ui-smoke.v1alpha1", TimestampFields: []string{"generated_at"}},
}

func checkV1EvidenceMetadata(readFile func(string) ([]byte, error), expectedVersion string, currentGitHead string, reportDir string) freshnessCheck {
	name := "docs/generated/v1_0 metadata"
	rules := v1GeneratedEvidenceRules
	if strings.TrimSpace(reportDir) != "" {
		name = "v1 report-dir metadata"
		rules = v1ReportDirEvidenceRules(reportDir)
	}
	check := freshnessCheck{Name: name, Status: "pass"}
	if readFile == nil {
		check.Status = "fail"
		check.Detail = "readFile is unavailable"
		return check
	}
	var issues []string
	for _, rule := range rules {
		raw, err := readFile(rule.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s read failed: %v", rule.Path, err))
			continue
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			issues = append(issues, fmt.Sprintf("%s is not valid JSON: %v", rule.Path, err))
			continue
		}
		if rule.Schema != "" {
			value, ok, err := jsonStringField(obj, rule.Path, "schema")
			if err != nil {
				issues = append(issues, err.Error())
			} else if !ok {
				issues = append(issues, fmt.Sprintf("%s schema is missing, want %q", rule.Path, rule.Schema))
			} else if value != rule.Schema {
				issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", rule.Path, value, rule.Schema))
			}
		}
		if rule.VersionField != "" {
			value, ok, err := jsonStringField(obj, rule.Path, rule.VersionField)
			if err != nil {
				issues = append(issues, err.Error())
			} else if !ok {
				issues = append(issues, fmt.Sprintf("%s %s is missing, want %q", rule.Path, rule.VersionField, expectedVersion))
			} else if value != expectedVersion {
				issues = append(issues, fmt.Sprintf("%s %s is %q, want %q", rule.Path, rule.VersionField, value, expectedVersion))
			}
		}
		if rule.RequireGitHead {
			value, ok, err := jsonStringField(obj, rule.Path, "git_head")
			if err != nil {
				issues = append(issues, err.Error())
			} else if !ok {
				issues = append(issues, fmt.Sprintf("%s git_head is missing, want %q", rule.Path, currentGitHead))
			} else if !gitHeadMatches(value, currentGitHead) {
				issues = append(issues, fmt.Sprintf("%s git_head is %q, want %q", rule.Path, value, currentGitHead))
			}
		}
		for _, field := range rule.TimestampFields {
			if err := validateRFC3339JSONField(obj, rule.Path, field); err != nil {
				issues = append(issues, err.Error())
			}
		}
		if rule.StartEndFields {
			if err := validateStartEndJSONFields(obj, rule.Path); err != nil {
				issues = append(issues, err.Error())
			}
		}
		if rule.ReleaseSummary {
			issues = append(issues, validateReleaseSummaryMetadata(obj, rule.Path, expectedVersion)...)
		}
	}
	if len(issues) > 0 {
		check.Status = "fail"
		check.Detail = strings.Join(issues, "; ")
	}
	return check
}

func v1ReportDirEvidenceRules(reportDir string) []generatedEvidenceRule {
	prefix := filepath.ToSlash(filepath.Clean(reportDir))
	join := func(parts ...string) string {
		all := append([]string{prefix}, parts...)
		return filepath.ToSlash(filepath.Join(all...))
	}
	return []generatedEvidenceRule{
		{Path: join("artifacts", "manifest.json"), VersionField: "compiler_version"},
		{Path: join("artifacts", "binary-size-thresholds.json"), Schema: "tetra.binary-size-thresholds.v1alpha1", VersionField: "compiler_version"},
		{Path: join("artifacts", "reproducible-build.json"), Schema: "tetra.reproducible-build-proof.v1alpha1", VersionField: "compiler_version"},
		{Path: join("artifacts", "performance-regression.json"), Schema: "tetra.performance-regression.v1", RequireGitHead: true},
		{Path: join("artifacts", "api-diff", "api-diff.json"), Schema: "tetra.api.diff.v1alpha1"},
		{Path: join("summary.json"), StartEndFields: true, ReleaseSummary: true},
		{Path: join("artifacts", "test-all", "summary.json"), StartEndFields: true},
		{Path: join("artifacts", "host-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "linux-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "macos-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "windows-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "wasm32-wasi-artifact-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "wasm32-web-artifact-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "wasi-smoke.artifact.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "wasi-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "test-all", "host-smoke.json"), VersionField: "version", RequireGitHead: true, TimestampFields: []string{"timestamp"}},
		{Path: join("artifacts", "web-ui-smoke.json"), Schema: "tetra.web-ui-smoke.v1alpha1", TimestampFields: []string{"generated_at"}},
	}
}

func validateReleaseSummaryMetadata(obj map[string]json.RawMessage, path string, expectedVersion string) []string {
	var issues []string
	expectedArtifact := expectedReleaseArtifact(expectedVersion)
	expectedGateCommand := expectedReleaseGateCommand(expectedVersion)
	for _, expected := range []struct {
		field string
		value string
	}{
		{field: "release_version", value: expectedVersion},
		{field: "release_artifact", value: expectedArtifact},
		{field: "release_gate_command", value: expectedGateCommand},
	} {
		value, ok, err := jsonStringField(obj, path, expected.field)
		if err != nil {
			issues = append(issues, err.Error())
		} else if !ok {
			issues = append(issues, fmt.Sprintf("%s %s is missing, want %q", path, expected.field, expected.value))
		} else if value != expected.value {
			issues = append(issues, fmt.Sprintf("%s %s is %q, want %q", path, expected.field, value, expected.value))
		}
	}
	return issues
}

func validateGateEvidenceTimestamps(gate gateEvidenceReport) error {
	if gate.Error != "" {
		return nil
	}
	if strings.TrimSpace(gate.StartedAt) == "" {
		return fmt.Errorf("started_at is missing")
	}
	startedAt, err := time.Parse(time.RFC3339, gate.StartedAt)
	if err != nil {
		return fmt.Errorf("started_at is not RFC3339: %w", err)
	}
	if strings.TrimSpace(gate.EndedAt) == "" {
		return fmt.Errorf("ended_at is missing")
	}
	endedAt, err := time.Parse(time.RFC3339, gate.EndedAt)
	if err != nil {
		return fmt.Errorf("ended_at is not RFC3339: %w", err)
	}
	if endedAt.Before(startedAt) {
		return fmt.Errorf("ended_at is before started_at")
	}
	return nil
}

func validateStartEndJSONFields(obj map[string]json.RawMessage, path string) error {
	startRaw, ok, err := jsonStringField(obj, path, "started_at")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s started_at is missing", path)
	}
	startedAt, err := time.Parse(time.RFC3339, startRaw)
	if err != nil {
		return fmt.Errorf("%s started_at is not RFC3339: %w", path, err)
	}
	endRaw, ok, err := jsonStringField(obj, path, "ended_at")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s ended_at is missing", path)
	}
	endedAt, err := time.Parse(time.RFC3339, endRaw)
	if err != nil {
		return fmt.Errorf("%s ended_at is not RFC3339: %w", path, err)
	}
	if endedAt.Before(startedAt) {
		return fmt.Errorf("%s ended_at is before started_at", path)
	}
	return nil
}

func validateRFC3339JSONField(obj map[string]json.RawMessage, path string, field string) error {
	value, ok, err := jsonStringField(obj, path, field)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s %s is missing", path, field)
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s %s is not RFC3339: %w", path, field, err)
	}
	return nil
}

func jsonStringField(obj map[string]json.RawMessage, path string, field string) (string, bool, error) {
	raw, ok := obj[field]
	if !ok {
		return "", false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", true, fmt.Errorf("%s %s must be a string: %w", path, field, err)
	}
	return value, true, nil
}

func gitHeadMatches(artifactHead string, currentHead string) bool {
	artifactHead = strings.TrimSpace(artifactHead)
	currentHead = strings.TrimSpace(currentHead)
	if artifactHead == "" || currentHead == "" {
		return false
	}
	if artifactHead == currentHead {
		return true
	}
	if len(artifactHead) >= 7 && len(currentHead) >= 7 {
		return strings.HasPrefix(artifactHead, currentHead) || strings.HasPrefix(currentHead, artifactHead)
	}
	return false
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

func checkArtifactHashManifest(repoRoot string, reportDir string) freshnessCheck {
	manifestPath := "docs/generated/v1_0/artifact-hashes.json"
	if strings.TrimSpace(reportDir) != "" {
		manifestPath = filepath.ToSlash(filepath.Join(reportDir, "artifacts", "artifact-hashes.json"))
	}
	cmd := exec.Command("go", "run", "./tools/cmd/validate-artifact-hashes", "--manifest", manifestPath)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		} else {
			detail += ": " + err.Error()
		}
		return freshnessCheck{Name: manifestPath, Status: "fail", Detail: detail}
	}
	return freshnessCheck{Name: manifestPath, Status: "pass"}
}

func checkSmokeEvidenceFreshness(repoRoot string, reportDir string) freshnessCheck {
	prefix := "docs/generated/v1_0"
	name := "docs/generated/v1_0 smoke evidence"
	if strings.TrimSpace(reportDir) != "" {
		prefix = filepath.ToSlash(filepath.Join(reportDir, "artifacts"))
		name = "v1 report-dir smoke evidence"
	}
	return checkValidationCommands(repoRoot, name, []validationCommand{
		{Name: "go", Args: []string{"run", "./tools/cmd/validate-smoke-list", "--report", filepath.ToSlash(filepath.Join(prefix, "smoke-list.json")), "--examples-root", "examples"}},
		{Name: "go", Args: []string{"run", "./tools/cmd/validate-smoke-list", "--report", filepath.ToSlash(filepath.Join(prefix, "test-all", "smoke-list.json")), "--examples-root", "examples"}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "host-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "linux-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "macos-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "windows-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "wasm32-wasi-artifact-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "wasm32-web-artifact-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "wasi-smoke.artifact.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "wasi-smoke.json"))}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", filepath.ToSlash(filepath.Join(prefix, "test-all", "host-smoke.json"))}},
	}, commandOutput)
}

func checkValidationCommands(repoRoot string, name string, commands []validationCommand, run func(string, string, ...string) (string, error)) freshnessCheck {
	check := freshnessCheck{Name: name, Status: "pass"}
	if run == nil {
		check.Status = "fail"
		check.Detail = "validator runner is unavailable"
		return check
	}
	var issues []string
	for _, validation := range commands {
		out, err := run(repoRoot, validation.Name, validation.Args...)
		if err != nil {
			detail := strings.TrimSpace(out)
			if detail == "" {
				detail = err.Error()
			} else {
				detail += ": " + err.Error()
			}
			issues = append(issues, strings.Join(append([]string{validation.Name}, validation.Args...), " ")+": "+detail)
		}
	}
	if len(issues) > 0 {
		check.Status = "fail"
		check.Detail = strings.Join(issues, "; ")
	}
	return check
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
	if isPromotedReleaseVersion(report.ExpectedVersion) {
		fmt.Fprintf(&b, "last gate identity: %s (release_version=%s, release_artifact=%s, release_gate_command=%s)\n",
			lastGateIdentityStatus(report),
			valueOrUnknown(report.LastGateEvidence.ReleaseVersion),
			valueOrUnknown(report.LastGateEvidence.ReleaseArtifact),
			valueOrUnknown(report.LastGateEvidence.ReleaseGateCommand),
		)
	}
	if len(report.RuntimeExecution.Required) > 0 || len(report.RuntimeExecution.Missing) > 0 {
		passed, failed := runtimeExecutionEvidenceCounts(report.RuntimeExecution)
		fmt.Fprintf(&b, "runtime execution evidence: %d/%d pass, %d missing", passed, len(report.RuntimeExecution.Required), len(report.RuntimeExecution.Missing))
		if failed > 0 {
			fmt.Fprintf(&b, ", %d fail", failed)
		}
		fmt.Fprintln(&b)
		if details := runtimeExecutionEvidenceTargetDetails(report.RuntimeExecution); details != "" {
			fmt.Fprintf(&b, "runtime execution targets: %s\n", details)
		}
		if commands := runtimeExecutionEvidenceCommands(report.RuntimeExecution); commands != "" {
			fmt.Fprintf(&b, "runtime execution commands: %s\n", commands)
		}
	}
	if report.SecurityReview.Status != "" {
		fmt.Fprintf(&b, "security review evidence: %s (%s)\n", report.SecurityReview.Status, valueOrUnknown(report.SecurityReview.Path))
		if report.SecurityReview.ValidatorCommand != "" {
			fmt.Fprintf(&b, "security review validator: %s\n", report.SecurityReview.ValidatorCommand)
		}
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

func runtimeExecutionEvidenceCounts(report runtimeExecutionEvidenceReport) (passed int, failed int) {
	for _, check := range report.Required {
		switch check.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		}
	}
	return passed, failed
}

func runtimeExecutionEvidenceTargetDetails(report runtimeExecutionEvidenceReport) string {
	var details []string
	for _, check := range report.Required {
		status := valueOrUnknown(check.Status)
		detail := fmt.Sprintf("%s=%s", valueOrUnknown(check.Target), status)
		if strings.TrimSpace(check.Host) != "" {
			detail += fmt.Sprintf("(host=%s)", check.Host)
		}
		details = append(details, detail)
	}
	return strings.Join(details, ", ")
}

func runtimeExecutionEvidenceCommands(report runtimeExecutionEvidenceReport) string {
	var commands []string
	for _, check := range report.Required {
		if strings.TrimSpace(check.EvidenceCommand) != "" {
			commands = append(commands, check.EvidenceCommand)
		}
	}
	return strings.Join(commands, "; ")
}

func lastGateIdentityStatus(report releaseStateReport) string {
	if report.LastGateEvidence.ReleaseVersion == report.ExpectedVersion &&
		report.LastGateEvidence.ReleaseArtifact == expectedReleaseArtifact(report.ExpectedVersion) &&
		report.LastGateEvidence.ReleaseGateCommand == expectedReleaseGateCommand(report.ExpectedVersion) {
		return "pass"
	}
	return "fail"
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "<unknown>"
	}
	return value
}
