package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"tetra_language/tools/internal/reportdecode"
)

type testResult struct {
	Name         string `json:"name"`
	Filename     string `json:"filename"`
	Index        int    `json:"index"`
	FunctionName string `json:"function_name"`
	ExitCode     int    `json:"exit_code"`
	Passed       bool   `json:"passed"`
	DurationMS   int64  `json:"duration_ms"`
	Error        string `json:"error,omitempty"`
}

type testFileReport struct {
	Filename   string `json:"filename"`
	Total      int    `json:"total"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
}

type testReportEnvelope struct {
	Total      int             `json:"total"`
	Passed     int             `json:"passed"`
	Failed     int             `json:"failed"`
	Target     string          `json:"target,omitempty"`
	DurationMS int64           `json:"duration_ms"`
	FilesRaw   json.RawMessage `json:"files"`
	ResultsRaw json.RawMessage `json:"results"`
	Files      []testFileReport
	Results    []testResult
}

const testReportArtifact = "tetra.release.v0_2_0.test-report.v1"

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to tetra test JSON or TOON report")
	flag.Parse()

	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateTestReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTestReport(raw []byte) error {
	var report testReportEnvelope
	if err := reportdecode.DecodeStrict(raw, &report); err != nil {
		return err
	}
	if err := unmarshalArray(report.FilesRaw, "files", &report.Files); err != nil {
		return err
	}
	if err := unmarshalArray(report.ResultsRaw, "results", &report.Results); err != nil {
		return err
	}
	return validateTestReportCounts(report)
}

func unmarshalArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("%s must be an array", field)
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := reportdecode.DecodeStrict(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func validateTestReportCounts(report testReportEnvelope) error {
	if report.DurationMS < 0 {
		return fmt.Errorf("report has negative duration %d", report.DurationMS)
	}
	total := len(report.Results)
	passed := 0
	durationMS := int64(0)
	byFile := map[string]*testFileReport{}
	indicesByFile := map[string]map[int]bool{}
	seenResults := map[string]bool{}
	seenFunctions := map[string]bool{}
	prevOrderKey := ""
	for _, result := range report.Results {
		if err := validateTestResult(result); err != nil {
			return err
		}
		orderKey := fmt.Sprintf("%s\x00%08d", result.Filename, result.Index)
		if prevOrderKey != "" && orderKey < prevOrderKey {
			return fmt.Errorf("results must be sorted by filename then index for deterministic evidence output")
		}
		prevOrderKey = orderKey
		resultKey := result.Filename + "\x00" + result.Name
		if seenResults[resultKey] {
			return fmt.Errorf("duplicate test result %q in %s", result.Name, result.Filename)
		}
		seenResults[resultKey] = true
		if seenFunctions[result.FunctionName] {
			return fmt.Errorf("duplicate test function %s", result.FunctionName)
		}
		seenFunctions[result.FunctionName] = true
		if indicesByFile[result.Filename] == nil {
			indicesByFile[result.Filename] = map[int]bool{}
		}
		if indicesByFile[result.Filename][result.Index] {
			return fmt.Errorf("duplicate test index %d in %s", result.Index, result.Filename)
		}
		indicesByFile[result.Filename][result.Index] = true
		if result.Passed {
			passed++
		}
		durationMS += result.DurationMS
		file := byFile[result.Filename]
		if file == nil {
			file = &testFileReport{Filename: result.Filename}
			byFile[result.Filename] = file
		}
		file.Total++
		file.DurationMS += result.DurationMS
		if result.Passed {
			file.Passed++
		} else {
			file.Failed++
		}
	}
	failed := total - passed
	if report.Total != total || report.Passed != passed || report.Failed != failed {
		return fmt.Errorf("report counts mismatch: got total=%d passed=%d failed=%d, computed total=%d passed=%d failed=%d", report.Total, report.Passed, report.Failed, total, passed, failed)
	}
	if report.DurationMS != durationMS {
		return fmt.Errorf("report duration mismatch: got %d, computed %d", report.DurationMS, durationMS)
	}

	filenames := make([]string, 0, len(byFile))
	for filename := range byFile {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	if len(report.Files) != len(filenames) {
		return fmt.Errorf("file report count mismatch: got %d, computed %d", len(report.Files), len(filenames))
	}
	for i, filename := range filenames {
		got := report.Files[i]
		if err := validateFileReport(got); err != nil {
			return err
		}
		want := *byFile[filename]
		if got != want {
			return fmt.Errorf("file report mismatch for %s: got %+v, computed %+v", filename, got, want)
		}
		for index := 0; index < got.Total; index++ {
			if !indicesByFile[filename][index] {
				return fmt.Errorf("file report %s missing test index %d", filename, index)
			}
		}
	}
	return nil
}

func validateFileReport(file testFileReport) error {
	if file.Filename == "" {
		return fmt.Errorf("file report missing filename")
	}
	if file.Total < 0 || file.Passed < 0 || file.Failed < 0 {
		return fmt.Errorf("file report %s has negative count", file.Filename)
	}
	if file.DurationMS < 0 {
		return fmt.Errorf("file report %s has negative duration %d", file.Filename, file.DurationMS)
	}
	return nil
}

func validateTestResult(result testResult) error {
	if result.Name == "" {
		return fmt.Errorf("test result missing name")
	}
	if result.Filename == "" {
		return fmt.Errorf("test result %q missing filename", result.Name)
	}
	if result.Index < 0 {
		return fmt.Errorf("test result %q has negative index %d", result.Name, result.Index)
	}
	if result.FunctionName == "" {
		return fmt.Errorf("test result %q missing function_name", result.Name)
	}
	if !strings.HasPrefix(result.FunctionName, "__tetra_test_") {
		return fmt.Errorf("test result %q function_name must use __tetra_test_ prefix", result.Name)
	}
	if result.Passed && result.ExitCode != 0 {
		return fmt.Errorf("test result %q passed result has non-zero exit code %d", result.Name, result.ExitCode)
	}
	if !result.Passed && result.ExitCode == 0 && result.Error == "" {
		return fmt.Errorf("test result %q failed result must include a non-zero exit code or error", result.Name)
	}
	if result.DurationMS < 0 {
		return fmt.Errorf("test result %q has negative duration %d", result.Name, result.DurationMS)
	}
	return nil
}
