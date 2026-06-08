package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tetra_language/tools/validators/uitoolkit"
)

func TestRunSmokeWritesValidToolkitCoreReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "ui-toolkit-core.json")
	if err := runSmoke(smokeConfig{
		ReportPath:      reportPath,
		SelfCheckRunner: inProcessSelfCheck,
	}); err != nil {
		t.Fatalf("runSmoke failed: %v", err)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := uitoolkit.ValidateReport(raw); err != nil {
		t.Fatalf("generated report did not validate: %v", err)
	}
	var report uitoolkit.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != uitoolkit.SchemaV1 {
		t.Fatalf("schema = %q, want %q", report.Schema, uitoolkit.SchemaV1)
	}
	if len(report.Artifacts) != 2 {
		t.Fatalf("artifacts = %d, want 2", len(report.Artifacts))
	}
}

func TestInternalChecksPass(t *testing.T) {
	for _, check := range []string{"runtime", "stress"} {
		if err := runInternalCheck(check); err != nil {
			t.Fatalf("runInternalCheck(%q) failed: %v", check, err)
		}
	}
	if err := runInternalCheck("unknown"); err == nil {
		t.Fatalf("expected unknown internal check to fail")
	}
}

func inProcessSelfCheck(name string) uitoolkit.ProcessReport {
	exitCode := 0
	err := runInternalCheck(name)
	if err != nil {
		exitCode = 1
	}
	return uitoolkit.ProcessReport{
		Name:     processName(name),
		Kind:     processKind(name),
		Path:     "in-process " + name,
		Ran:      true,
		Pass:     err == nil,
		ExitCode: &exitCode,
	}
}
