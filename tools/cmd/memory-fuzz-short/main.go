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

type memoryFuzzShortOptions struct {
	Tier      string
	ReportDir string
}

type memoryFuzzShortSummaryJSON struct {
	SchemaVersion string                      `json:"schema_version"`
	Kind          string                      `json:"kind"`
	Tier          string                      `json:"tier"`
	Status        string                      `json:"status"`
	Artifacts     map[string]string           `json:"artifacts"`
	Commands      []memoryFuzzShortCommandRow `json:"commands"`
	Policies      []string                    `json:"policies"`
	NonClaims     []string                    `json:"non_claims"`
}

type memoryFuzzShortCommandRow struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

func main() {
	var opt memoryFuzzShortOptions
	flag.StringVar(&opt.Tier, "tier", "1", "memory fuzz tier to run; only Tier 1 short CI smoke is supported by this command")
	flag.StringVar(&opt.ReportDir, "report-dir", "", "directory for memory fuzz short artifacts")
	flag.Parse()
	if err := runMemoryFuzzShort(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMemoryFuzzShort(opt memoryFuzzShortOptions) error {
	tier := strings.TrimSpace(strings.ToLower(opt.Tier))
	if tier != "1" && tier != "tier1" && tier != "tier-1" {
		return fmt.Errorf("memory-fuzz-short only supports Tier 1 short CI smoke, got %q", opt.Tier)
	}
	if strings.TrimSpace(opt.ReportDir) == "" {
		return fmt.Errorf("--report-dir is required")
	}
	if err := checkMemoryFuzzShortReportDirFresh(opt.ReportDir); err != nil {
		return err
	}
	if err := os.MkdirAll(opt.ReportDir, 0o755); err != nil {
		return err
	}
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		return err
	}
	if err := compiler.ValidateMemoryFuzzOracleReport(report); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	reportPath := filepath.Join(opt.ReportDir, "memory-fuzz-oracle.json")
	if err := os.WriteFile(reportPath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opt.ReportDir, "summary.md"), []byte(memoryFuzzShortSummary(report, reportPath)), 0o644); err != nil {
		return err
	}
	summaryJSON, err := json.MarshalIndent(memoryFuzzShortSummaryForJSON(opt.ReportDir), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(opt.ReportDir, "summary.json"), append(summaryJSON, '\n'), 0o644)
}

func checkMemoryFuzzShortReportDirFresh(reportDir string) error {
	info, err := os.Lstat(reportDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use symlink --report-dir %s; choose a real fresh --report-dir", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to use non-directory --report-dir %s; choose a fresh --report-dir directory", reportDir)
	}
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("refusing to reuse non-empty --report-dir %s; choose a fresh --report-dir so stale fuzz artifacts cannot be reused", reportDir)
	}
	return nil
}

func memoryFuzzShortSummary(report compiler.MemoryFuzzOracleReport, reportPath string) string {
	return fmt.Sprintf("# Memory Fuzz Short Summary\n\n- schema: `%s`\n- scope: `%s`\n- tier: `Tier 1 short CI smoke`\n- report: `%s`\n- summary_json: `summary.json`\n- validator: `go run ./tools/cmd/validate-memory-fuzz-oracle --report %s --artifact-dir %s`\n- oracle_categories: `%d`\n- release_evidence_requirements: `%d` (`MEM-FUZZ-001`..`MEM-FUZZ-005`)\n- deterministic_slice_coverage: `%d` (`v0-v11`)\n- tier1_short_ci_smoke_cases: `%d`\n\n", report.SchemaVersion, report.Scope, filepath.ToSlash(reportPath), filepath.ToSlash(reportPath), filepath.ToSlash(filepath.Dir(reportPath)), len(report.Rows), len(report.Requirements), len(report.SliceCoverage), report.Tier1ShortCISmokeCases)
}

func memoryFuzzShortSummaryForJSON(reportDir string) memoryFuzzShortSummaryJSON {
	reportDirSlash := filepath.ToSlash(reportDir)
	reportPath := filepath.ToSlash(filepath.Join(reportDir, "memory-fuzz-oracle.json"))
	return memoryFuzzShortSummaryJSON{
		SchemaVersion: "tetra.memory-fuzz-short.summary.v1",
		Kind:          "tier1_short_ci_smoke",
		Tier:          string(compiler.MemoryFuzzTier1ShortCI),
		Status:        "pass",
		Artifacts: map[string]string{
			"oracle_report": "memory-fuzz-oracle.json",
			"summary_md":    "summary.md",
			"summary_json":  "summary.json",
		},
		Commands: []memoryFuzzShortCommandRow{
			{
				Name:    "memory-fuzz-short",
				Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + reportDirSlash,
				Status:  "pass",
			},
			{
				Name:    "validate-memory-fuzz-oracle",
				Command: "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + reportPath + " --artifact-dir " + reportDirSlash,
				Status:  "pass",
			},
		},
		Policies: []string{
			"Tier 1 deterministic smoke writes report, markdown summary, and machine-readable summary",
			"Tier 2 nightly seed triage and minimized repro policy remains boundary-recorded in the oracle report",
			"Tier 3 release-blocking focused memory fuzz blocks promotion until failures are classified",
		},
		NonClaims: []string{
			"no exhaustive fuzz proof is claimed",
			"no Memory 100% claim is made",
		},
	}
}
