package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesCheckedInSmokeReportWithAllowance(t *testing.T) {
	report := filepath.Join(
		"..",
		"..",
		"..",
		"docs",
		"benchmarks",
		"techempower_local_smoke_skip_db_report.json",
	)
	if err := run([]string{"--report", report, "--allow-skip-db"}); err != nil {
		t.Fatalf("run allow skip-db: %v", err)
	}
}

func TestRunValidatesCheckedInSCRAMMatrixReport(t *testing.T) {
	report := filepath.Join(
		"..",
		"..",
		"..",
		"docs",
		"benchmarks",
		"techempower_scram_single_query_matrix_local_report.json",
	)
	if err := run([]string{"--report", report}); err != nil {
		t.Fatalf("run matrix report: %v", err)
	}
}

func TestRunRejectsSkipDBReportWithoutAllowance(t *testing.T) {
	report := filepath.Join(
		"..",
		"..",
		"..",
		"docs",
		"benchmarks",
		"techempower_local_smoke_skip_db_report.json",
	)
	err := run([]string{"--report", report})
	if err == nil || !strings.Contains(err.Error(), "skip-db") {
		t.Fatalf("run without allow skip-db = %v, want skip-db rejection", err)
	}
}

func TestRunRequiresReportPath(t *testing.T) {
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "--report") {
		t.Fatalf("run without report = %v, want --report error", err)
	}
}

func TestRunRejectsMissingFile(t *testing.T) {
	err := run([]string{"--report", filepath.Join(t.TempDir(), "missing.json")})
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("run missing file = %v, want not exist error", err)
	}
}
