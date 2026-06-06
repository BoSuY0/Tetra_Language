package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryFuzzOracleReportFile(path); err != nil {
		t.Fatalf("validateMemoryFuzzOracleReportFile: %v", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.Rows = report.Rows[1:]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateMemoryFuzzOracleReportFile(path)
	if err == nil || !strings.Contains(err.Error(), "missing oracle_category") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing oracle_category", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.Requirements = report.Requirements[1:]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateMemoryFuzzOracleReportFile(path)
	if err == nil || !strings.Contains(err.Error(), "missing requirement MEM-FUZZ-001") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing MEM-FUZZ-001", err)
	}
}
