package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
)

type memoryFuzzShortArtifactSummary struct {
	SchemaVersion string            `json:"schema_version"`
	Kind          string            `json:"kind"`
	Tier          string            `json:"tier"`
	Status        string            `json:"status"`
	Artifacts     map[string]string `json:"artifacts"`
	Commands      []struct {
		Name    string `json:"name"`
		Command string `json:"command"`
		Status  string `json:"status"`
	} `json:"commands"`
}

func main() {
	var reportPath string
	var artifactDir string
	flag.StringVar(&reportPath, "report", "", "path to tetra.memory-fuzz.oracle.v1 report")
	flag.StringVar(&artifactDir, "artifact-dir", "", "optional Tier 1 artifact directory to validate alongside the oracle report")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryFuzzOracleReportFile(reportPath, artifactDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryFuzzOracleReportFile(path string, artifactDirs ...string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report compiler.MemoryFuzzOracleReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("memory fuzz oracle report is malformed: %w", err)
	}
	if err := compiler.ValidateMemoryFuzzOracleReport(report); err != nil {
		return err
	}
	if len(artifactDirs) == 0 || strings.TrimSpace(artifactDirs[0]) == "" {
		return nil
	}
	return validateMemoryFuzzOracleArtifactDir(path, artifactDirs[0])
}

func validateMemoryFuzzOracleArtifactDir(reportPath string, artifactDir string) error {
	info, err := os.Lstat(artifactDir)
	if err != nil {
		return fmt.Errorf("memory fuzz artifact dir: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("memory fuzz artifact dir %s must not be a symlink", artifactDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("memory fuzz artifact dir %s is not a directory", artifactDir)
	}

	expectedReport := filepath.Join(artifactDir, "memory-fuzz-oracle.json")
	if same, err := sameCleanPath(reportPath, expectedReport); err != nil {
		return err
	} else if !same {
		return fmt.Errorf("--report must point at %s when --artifact-dir is used, got %s", expectedReport, reportPath)
	}
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json"} {
		if err := requireMemoryFuzzArtifactFile(artifactDir, rel); err != nil {
			return err
		}
	}
	summaryMD, err := os.ReadFile(filepath.Join(artifactDir, "summary.md"))
	if err != nil {
		return err
	}
	summaryText := string(summaryMD)
	for _, want := range []string{"Memory Fuzz Short Summary", "Tier 1", "memory-fuzz-oracle.json"} {
		if !strings.Contains(summaryText, want) {
			return fmt.Errorf("summary.md missing %q", want)
		}
	}
	raw, err := os.ReadFile(filepath.Join(artifactDir, "summary.json"))
	if err != nil {
		return err
	}
	var summary memoryFuzzShortArtifactSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		return fmt.Errorf("memory fuzz summary.json is malformed: %w", err)
	}
	if summary.SchemaVersion != "tetra.memory-fuzz-short.summary.v1" {
		return fmt.Errorf("summary.json schema_version = %q, want tetra.memory-fuzz-short.summary.v1", summary.SchemaVersion)
	}
	if summary.Kind != "tier1_short_ci_smoke" || summary.Tier != string(compiler.MemoryFuzzTier1ShortCI) || summary.Status != "pass" {
		return fmt.Errorf("summary.json identity/status must record passing Tier 1 short CI smoke, got kind=%q tier=%q status=%q", summary.Kind, summary.Tier, summary.Status)
	}
	for key, want := range map[string]string{
		"oracle_report": "memory-fuzz-oracle.json",
		"summary_md":    "summary.md",
		"summary_json":  "summary.json",
	} {
		got := summary.Artifacts[key]
		if got != want {
			return fmt.Errorf("summary.json artifact %s = %q, want %q", key, got, want)
		}
		if err := requireMemoryFuzzRelativeArtifactPath(got); err != nil {
			return fmt.Errorf("summary.json artifact %s: %w", key, err)
		}
	}
	var sawRunner, sawValidator bool
	for _, command := range summary.Commands {
		if command.Status != "pass" {
			return fmt.Errorf("summary.json command %s status = %q, want pass", command.Name, command.Status)
		}
		switch command.Name {
		case "memory-fuzz-short":
			if strings.Contains(command.Command, "go run ./tools/cmd/memory-fuzz-short") && strings.Contains(command.Command, "--report-dir") {
				sawRunner = true
			}
		case "validate-memory-fuzz-oracle":
			if strings.Contains(command.Command, "go run ./tools/cmd/validate-memory-fuzz-oracle") && strings.Contains(command.Command, "--report") && strings.Contains(command.Command, "--artifact-dir") {
				sawValidator = true
			}
		}
	}
	if !sawRunner {
		return fmt.Errorf("summary.json missing memory-fuzz-short command provenance")
	}
	if !sawValidator {
		return fmt.Errorf("summary.json missing validate-memory-fuzz-oracle command provenance")
	}
	return nil
}

func sameCleanPath(a string, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return filepath.Clean(absA) == filepath.Clean(absB), nil
}

func requireMemoryFuzzArtifactFile(dir string, rel string) error {
	if err := requireMemoryFuzzRelativeArtifactPath(rel); err != nil {
		return err
	}
	path := filepath.Join(dir, rel)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing required memory fuzz artifact %s", rel)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("required memory fuzz artifact %s is a directory", rel)
	}
	if info.Size() == 0 {
		return fmt.Errorf("required memory fuzz artifact %s is empty", rel)
	}
	return nil
}

func requireMemoryFuzzRelativeArtifactPath(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("path %q must be relative", rel)
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("path %q must stay inside artifact dir", rel)
	}
	return nil
}
