package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var fuzzStepNames = []string{
	"compiler-frontend-lexer",
	"compiler-frontend-parser",
	"compiler-linker-linkcore",
	"validate-manifest",
	"eco-capsule",
	"property-stress-regressions",
}

var stepLineRE = regexp.MustCompile("^- `([^`]+)`: ([^,]+), command `([^`]*)`, log `([^`]*)`$")

type fuzzSummary struct {
	Mode             string
	Fuzztime         string
	OutputDir        string
	CrasherArchive   string
	CrasherInventory string
	UnstableSeedLog  string
	Steps            []fuzzStep
}

type fuzzStep struct {
	Name    string
	Status  string
	Command string
	Log     string
}

type fuzzSummaryJSON struct {
	Mode        string                `json:"mode"`
	Status      string                `json:"status"`
	ExitCode    int                   `json:"exit_code"`
	Fuzztime    string                `json:"fuzztime"`
	StepCount   int                   `json:"step_count"`
	FailedCount int                   `json:"failed_count"`
	Artifacts   fuzzJSONArtifacts     `json:"artifacts"`
	Steps       []fuzzSummaryJSONStep `json:"steps"`
}

type fuzzJSONArtifacts struct {
	SummaryMD            string `json:"summary_md"`
	SummaryJSON          string `json:"summary_json"`
	CrasherInventoryJSON string `json:"crasher_inventory_json"`
	LogsDir              string `json:"logs_dir"`
	UnstableSeedLog      string `json:"unstable_seed_log"`
	CrasherArchivePath   string `json:"crasher_archive_path"`
}

type fuzzSummaryJSONStep struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	ExitCode        int    `json:"exit_code"`
	DurationSeconds int    `json:"duration_seconds"`
	Command         string `json:"command"`
	Log             string `json:"log"`
}

func main() {
	var reportDir string
	flag.StringVar(&reportDir, "report-dir", "", "fuzz report directory containing summary.md, unstable-seeds.md, and logs")
	flag.Parse()
	if reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateFuzzReport(reportDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateFuzzReport(reportDir string) error {
	if info, err := os.Stat(reportDir); err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("report-dir is not a directory: %s", reportDir)
	}
	for _, rel := range requiredFuzzArtifacts() {
		if err := requireFile(reportDir, rel); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(filepath.Join(reportDir, "summary.md"))
	if err != nil {
		return err
	}
	summary, err := parseFuzzSummary(string(raw))
	if err != nil {
		return err
	}
	if err := validateFuzzSummary(summary, reportDir); err != nil {
		return err
	}
	rawJSON, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		return err
	}
	summaryJSON, err := parseFuzzSummaryJSON(rawJSON)
	if err != nil {
		return err
	}
	if err := validateFuzzSummaryJSON(summaryJSON, summary, reportDir); err != nil {
		return err
	}
	unstableSeeds, err := os.ReadFile(filepath.Join(reportDir, "unstable-seeds.md"))
	if err != nil {
		return err
	}
	if err := validateUnstableSeeds(string(unstableSeeds)); err != nil {
		return err
	}
	return nil
}

func requiredFuzzArtifacts() []string {
	artifacts := []string{"summary.md", "summary.json", "crasher-inventory.json", "unstable-seeds.md"}
	for _, name := range fuzzStepNames {
		artifacts = append(artifacts, filepath.ToSlash(filepath.Join("logs", name+".log")))
	}
	return artifacts
}

func requireFile(root string, rel string) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing required artifact %s", rel)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("required artifact %s is a directory", rel)
	}
	return nil
}

func parseFuzzSummary(text string) (fuzzSummary, error) {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 14 {
		return fuzzSummary{}, fmt.Errorf("summary.md has malformed shape: got %d non-empty lines, want 14", len(lines))
	}
	if lines[0] != "# Fuzz Nightly Summary" {
		return fuzzSummary{}, fmt.Errorf("summary.md missing title")
	}
	if lines[7] != "## Steps" {
		return fuzzSummary{}, fmt.Errorf("summary.md missing steps heading")
	}
	summary := fuzzSummary{
		Mode:             metadataValue(lines[1], "mode"),
		Fuzztime:         metadataValue(lines[2], "fuzztime"),
		OutputDir:        metadataValue(lines[3], "output_dir"),
		CrasherArchive:   metadataValue(lines[4], "crasher_archive_path"),
		CrasherInventory: metadataValue(lines[5], "crasher_inventory_json"),
		UnstableSeedLog:  metadataValue(lines[6], "unstable_seed_log"),
	}
	for _, stepLine := range lines[8:] {
		matches := stepLineRE.FindStringSubmatch(stepLine)
		if matches == nil {
			return fuzzSummary{}, fmt.Errorf("malformed step line %q", stepLine)
		}
		summary.Steps = append(summary.Steps, fuzzStep{
			Name:    matches[1],
			Status:  matches[2],
			Command: matches[3],
			Log:     matches[4],
		})
	}
	return summary, nil
}

func metadataValue(line string, key string) string {
	prefix := "- " + key + ": `"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, "`") {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(line, prefix), "`")
}

func parseFuzzSummaryJSON(raw []byte) (fuzzSummaryJSON, error) {
	var summary fuzzSummaryJSON
	if err := json.Unmarshal(raw, &summary); err != nil {
		return fuzzSummaryJSON{}, fmt.Errorf("summary.json is malformed: %w", err)
	}
	return summary, nil
}

func validateFuzzSummary(summary fuzzSummary, reportDir string) error {
	switch summary.Mode {
	case "short", "nightly":
	default:
		return fmt.Errorf("invalid mode %q", summary.Mode)
	}
	if strings.TrimSpace(summary.Fuzztime) == "" {
		return fmt.Errorf("fuzztime is required")
	}
	if strings.TrimSpace(summary.OutputDir) == "" {
		return fmt.Errorf("output_dir is required")
	}
	if !samePath(summary.OutputDir, reportDir) {
		return fmt.Errorf("output_dir = %q, want %q", summary.OutputDir, reportDir)
	}
	if summary.CrasherArchive != "<package>/testdata/fuzz/<FuzzName>/" {
		return fmt.Errorf("crasher_archive_path = %q, want <package>/testdata/fuzz/<FuzzName>/", summary.CrasherArchive)
	}
	if !samePath(summary.CrasherInventory, filepath.Join(reportDir, "crasher-inventory.json")) {
		return fmt.Errorf("crasher_inventory_json = %q, want %q", summary.CrasherInventory, filepath.Join(reportDir, "crasher-inventory.json"))
	}
	if !samePath(summary.UnstableSeedLog, filepath.Join(reportDir, "unstable-seeds.md")) {
		return fmt.Errorf("unstable_seed_log = %q, want %q", summary.UnstableSeedLog, filepath.Join(reportDir, "unstable-seeds.md"))
	}
	if len(summary.Steps) != len(fuzzStepNames) {
		return fmt.Errorf("step count = %d, want %d", len(summary.Steps), len(fuzzStepNames))
	}
	for i, step := range summary.Steps {
		wantName := fuzzStepNames[i]
		if step.Name != wantName {
			return fmt.Errorf("step %d name = %q, want %q", i+1, step.Name, wantName)
		}
		if step.Status != "pass" {
			return fmt.Errorf("step %s has invalid or failing status %q", step.Name, step.Status)
		}
		if step.Command != expectedFuzzCommand(step.Name, summary.Fuzztime, summary.Mode) {
			return fmt.Errorf("step %s command = %q, want %q", step.Name, step.Command, expectedFuzzCommand(step.Name, summary.Fuzztime, summary.Mode))
		}
		wantLog := filepath.Join(reportDir, "logs", step.Name+".log")
		if !samePath(step.Log, wantLog) {
			return fmt.Errorf("step %s log = %q, want %q", step.Name, step.Log, wantLog)
		}
	}
	return nil
}

func validateFuzzSummaryJSON(summaryJSON fuzzSummaryJSON, summary fuzzSummary, reportDir string) error {
	if summaryJSON.Mode != summary.Mode {
		return fmt.Errorf("summary.json mode = %q, want %q", summaryJSON.Mode, summary.Mode)
	}
	if summaryJSON.Status != "pass" {
		return fmt.Errorf("summary.json status = %q, want pass", summaryJSON.Status)
	}
	if summaryJSON.ExitCode != 0 {
		return fmt.Errorf("summary.json exit_code = %d, want 0", summaryJSON.ExitCode)
	}
	if summaryJSON.Fuzztime != summary.Fuzztime {
		return fmt.Errorf("summary.json fuzztime = %q, want %q", summaryJSON.Fuzztime, summary.Fuzztime)
	}
	if summaryJSON.StepCount != len(fuzzStepNames) {
		return fmt.Errorf("summary.json step_count = %d, want %d", summaryJSON.StepCount, len(fuzzStepNames))
	}
	if summaryJSON.FailedCount != 0 {
		return fmt.Errorf("summary.json failed_count = %d, want 0", summaryJSON.FailedCount)
	}
	if len(summaryJSON.Steps) != summaryJSON.StepCount {
		return fmt.Errorf("summary.json steps length = %d, want %d", len(summaryJSON.Steps), summaryJSON.StepCount)
	}
	if !samePath(summaryJSON.Artifacts.SummaryMD, filepath.Join(reportDir, "summary.md")) {
		return fmt.Errorf("summary.json artifacts.summary_md = %q, want %q", summaryJSON.Artifacts.SummaryMD, filepath.Join(reportDir, "summary.md"))
	}
	if !samePath(summaryJSON.Artifacts.SummaryJSON, filepath.Join(reportDir, "summary.json")) {
		return fmt.Errorf("summary.json artifacts.summary_json = %q, want %q", summaryJSON.Artifacts.SummaryJSON, filepath.Join(reportDir, "summary.json"))
	}
	if !samePath(summaryJSON.Artifacts.CrasherInventoryJSON, filepath.Join(reportDir, "crasher-inventory.json")) {
		return fmt.Errorf("summary.json artifacts.crasher_inventory_json = %q, want %q", summaryJSON.Artifacts.CrasherInventoryJSON, filepath.Join(reportDir, "crasher-inventory.json"))
	}
	if !samePath(summaryJSON.Artifacts.LogsDir, filepath.Join(reportDir, "logs")) {
		return fmt.Errorf("summary.json artifacts.logs_dir = %q, want %q", summaryJSON.Artifacts.LogsDir, filepath.Join(reportDir, "logs"))
	}
	if !samePath(summaryJSON.Artifacts.UnstableSeedLog, filepath.Join(reportDir, "unstable-seeds.md")) {
		return fmt.Errorf("summary.json artifacts.unstable_seed_log = %q, want %q", summaryJSON.Artifacts.UnstableSeedLog, filepath.Join(reportDir, "unstable-seeds.md"))
	}
	if summaryJSON.Artifacts.CrasherArchivePath != "<package>/testdata/fuzz/<FuzzName>/" {
		return fmt.Errorf("summary.json artifacts.crasher_archive_path = %q, want <package>/testdata/fuzz/<FuzzName>/", summaryJSON.Artifacts.CrasherArchivePath)
	}
	for i, step := range summaryJSON.Steps {
		wantName := fuzzStepNames[i]
		if step.Name != wantName {
			return fmt.Errorf("summary.json step %d name = %q, want %q", i+1, step.Name, wantName)
		}
		if step.Status != "pass" {
			return fmt.Errorf("summary.json step %s status = %q, want pass", step.Name, step.Status)
		}
		if step.ExitCode != 0 {
			return fmt.Errorf("summary.json step %s exit_code = %d, want 0", step.Name, step.ExitCode)
		}
		if step.DurationSeconds < 0 {
			return fmt.Errorf("summary.json step %s duration_seconds = %d, want >= 0", step.Name, step.DurationSeconds)
		}
		if step.Command != expectedFuzzCommand(step.Name, summary.Fuzztime, summary.Mode) {
			return fmt.Errorf("summary.json step %s command = %q, want %q", step.Name, step.Command, expectedFuzzCommand(step.Name, summary.Fuzztime, summary.Mode))
		}
		wantLog := filepath.ToSlash(filepath.Join("logs", step.Name+".log"))
		if step.Log != wantLog {
			return fmt.Errorf("summary.json step %s log = %q, want %q", step.Name, step.Log, wantLog)
		}
	}
	return nil
}

func expectedFuzzCommand(name string, fuzztime string, mode string) string {
	parallel := ""
	if mode == "short" {
		parallel = " -parallel=1"
	}
	switch name {
	case "compiler-frontend-lexer":
		return "go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzLexer -fuzztime=" + fuzztime + parallel
	case "compiler-frontend-parser":
		return "go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzParser -fuzztime=" + fuzztime + parallel
	case "compiler-linker-linkcore":
		return "go test ./compiler/internal/linker/linkcore -run \\^\\$ -fuzz=FuzzLinkX64ObjectsDoesNotPanic -fuzztime=" + fuzztime + parallel
	case "validate-manifest":
		return "go test ./tools/cmd/validate-manifest -run \\^\\$ -fuzz=. -fuzztime=" + fuzztime + parallel
	case "eco-capsule":
		return "go test ./cli/cmd/tetra -run \\^\\$ -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=" + fuzztime + parallel
	case "property-stress-regressions":
		return "go test ./compiler/... ./cli/... ./tools/cmd/validate-manifest -run Fuzz\\|Property\\|Stress -count=1"
	default:
		return ""
	}
}

func validateUnstableSeeds(text string) error {
	if !strings.Contains(text, "# Unstable Fuzz Seeds") {
		return fmt.Errorf("unstable-seeds.md missing title")
	}
	if !strings.Contains(text, "| package | fuzz target | seed/crasher path | status | owner | next command |") {
		return fmt.Errorf("unstable-seeds.md missing table header")
	}
	if !strings.Contains(text, "| --- | --- | --- | --- | --- | --- |") {
		return fmt.Errorf("unstable-seeds.md missing table separator")
	}
	return nil
}

func samePath(a string, b string) bool {
	a = filepath.Clean(filepath.FromSlash(a))
	b = filepath.Clean(filepath.FromSlash(b))
	if a == b {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}
