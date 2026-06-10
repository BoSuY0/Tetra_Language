package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMemoryGradeReportRejectsContradictorySummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-grade.json")
	raw := `{
  "schema_version":"tetra.memory-grade-report.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "artifact_grade":"M5",
  "functions":[],
  "summary":{"row_count":1,"artifact_grade":"M0","heap_rows":1,"copy_rows":0,"unbounded_rows":1,"budget_bytes":8192},
  "non_claims":["no Memory 100% claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryGradeReport(path)
	if err == nil || !strings.Contains(err.Error(), "artifact_grade") {
		t.Fatalf("validateMemoryGradeReport error = %v, want artifact_grade rejection", err)
	}
}
