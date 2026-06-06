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
	return os.WriteFile(filepath.Join(opt.ReportDir, "summary.md"), []byte(memoryFuzzShortSummary(report, reportPath)), 0o644)
}

func memoryFuzzShortSummary(report compiler.MemoryFuzzOracleReport, reportPath string) string {
	return fmt.Sprintf("# Memory Fuzz Short Summary\n\n- schema: `%s`\n- scope: `%s`\n- tier: `Tier 1 short CI smoke`\n- report: `%s`\n- oracle_categories: `%d`\n- release_evidence_requirements: `%d` (`MEM-FUZZ-001`..`MEM-FUZZ-005`)\n- deterministic_slice_coverage: `%d` (`v0-v11`)\n- tier1_short_ci_smoke_cases: `%d`\n\n", report.SchemaVersion, report.Scope, filepath.ToSlash(reportPath), len(report.Rows), len(report.Requirements), len(report.SliceCoverage), report.Tier1ShortCISmokeCases)
}
