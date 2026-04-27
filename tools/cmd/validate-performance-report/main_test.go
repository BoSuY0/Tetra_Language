package main

import (
	"strings"
	"testing"
)

func TestValidatePerformanceReportAcceptsEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.performance-regression.v1",
  "git_head": "abc123",
  "host": "linux amd64",
  "go_version": "go1.20",
  "command": "go test ./compiler -bench='BenchmarkCompile' -run '^$' -count=1",
  "baseline_artifact": "none",
  "threshold_decision": "baseline capture",
  "metrics": [
    {"name":"BenchmarkCompile/foo-8","iterations":10,"ns_per_op":1000,"threshold":"compile <= 15 percent slower","decision":"accepted"}
  ],
  "residual_risk": "single host"
}`)
	if err := validatePerformanceReport(raw); err != nil {
		t.Fatalf("validatePerformanceReport: %v", err)
	}
}

func TestValidatePerformanceReportRejectsMissingMetrics(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.performance-regression.v1",
  "git_head": "abc123",
  "host": "linux amd64",
  "go_version": "go1.20",
  "command": "go test ./compiler -bench=. -run '^$' -count=1",
  "baseline_artifact": "none",
  "threshold_decision": "baseline capture",
  "metrics": [],
  "residual_risk": "single host"
}`)
	err := validatePerformanceReport(raw)
	if err == nil {
		t.Fatalf("expected missing metrics failure")
	}
	if !strings.Contains(err.Error(), "metrics") {
		t.Fatalf("unexpected error: %v", err)
	}
}
