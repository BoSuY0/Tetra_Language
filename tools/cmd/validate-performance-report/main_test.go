package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidatePerformanceReportAcceptsEvidence(t *testing.T) {
	raw := mustReportJSON(t, performanceReport{
		Schema:            "tetra.performance-regression.v1",
		GitHead:           "abc123",
		Host:              "linux amd64",
		GoVersion:         "go1.20",
		Command:           "go test ./compiler -bench='BenchmarkCompile' -run '^$' -count=1",
		BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
		ThresholdDecision: "baseline capture approved",
		Metrics: []performanceMetric{
			{Name: "BenchmarkBinarySize/foo-8", Iterations: 10, NsPerOp: 1000, ArtifactBytes: 4096, Threshold: "compile <= 15 percent slower", Decision: "accepted"},
			{Name: "BenchmarkCompile/foo-8", Iterations: 12, NsPerOp: 1200, Threshold: "compile <= 15 percent slower", Decision: "accepted"},
		},
		ResidualRisk: "single host",
	})
	if err := validatePerformanceReport(raw); err != nil {
		t.Fatalf("validatePerformanceReport: %v", err)
	}
}

func TestValidatePerformanceReportRejectsMissingMetrics(t *testing.T) {
	raw := mustReportJSON(t, performanceReport{
		Schema:            "tetra.performance-regression.v1",
		GitHead:           "abc123",
		Host:              "linux amd64",
		GoVersion:         "go1.20",
		Command:           "go test ./compiler -bench=. -run '^$' -count=1",
		BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
		ThresholdDecision: "baseline capture approved",
		Metrics:           []performanceMetric{},
		ResidualRisk:      "single host",
	})
	err := validatePerformanceReport(raw)
	if err == nil {
		t.Fatalf("expected missing metrics failure")
	}
	if !strings.Contains(err.Error(), "metrics") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePerformanceReportRejectsUnsortedMetrics(t *testing.T) {
	raw := mustReportJSON(t, performanceReport{
		Schema:            "tetra.performance-regression.v1",
		GitHead:           "abc123",
		Host:              "linux amd64",
		GoVersion:         "go1.20",
		Command:           "go test ./compiler -bench=. -run '^$' -count=1",
		BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
		ThresholdDecision: "baseline capture approved",
		Metrics: []performanceMetric{
			{Name: "BenchmarkCompile/z", Iterations: 10, NsPerOp: 10, Threshold: "x", Decision: "accepted"},
			{Name: "BenchmarkCompile/a", Iterations: 10, NsPerOp: 10, Threshold: "x", Decision: "accepted"},
		},
		ResidualRisk: "single host",
	})
	err := validatePerformanceReport(raw)
	if err == nil || !strings.Contains(err.Error(), "sorted") {
		t.Fatalf("expected sorted metrics failure, got %v", err)
	}
}

func TestValidatePerformanceReportRejectsSummaryHashMismatch(t *testing.T) {
	report := performanceReport{
		Schema:            "tetra.performance-regression.v1",
		GitHead:           "abc123",
		Host:              "linux amd64",
		GoVersion:         "go1.20",
		Command:           "go test ./compiler -bench=. -run '^$' -count=1",
		BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
		ThresholdDecision: "baseline capture approved",
		Metrics: []performanceMetric{
			{Name: "BenchmarkCompile/a", Iterations: 10, NsPerOp: 10, Threshold: "x", Decision: "accepted"},
		},
		ResidualRisk: "single host",
		Summary: performanceSummary{
			MetricCount:     1,
			TotalIterations: 10,
			MaxNsPerOp:      10,
			MetricsSHA256:   "sha256:deadbeef",
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = validatePerformanceReport(raw)
	if err == nil || !strings.Contains(err.Error(), "metrics_sha256") {
		t.Fatalf("expected metrics hash failure, got %v", err)
	}
}

func TestStampPerformanceReportGitHeadUpdatesEvidence(t *testing.T) {
	raw := mustReportJSON(t, performanceReport{
		Schema:            "tetra.performance-regression.v1",
		GitHead:           "stale-head",
		Host:              "linux amd64",
		GoVersion:         "go1.20",
		Command:           "go test ./compiler -bench=. -run '^$' -count=1",
		BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
		ThresholdDecision: "baseline capture approved",
		Metrics: []performanceMetric{
			{Name: "BenchmarkBinarySize/foo-8", Iterations: 10, NsPerOp: 1000, ArtifactBytes: 4096, Threshold: "compile <= 15 percent slower", Decision: "accepted"},
			{Name: "BenchmarkCompile/foo-8", Iterations: 12, NsPerOp: 1200, Threshold: "compile <= 15 percent slower", Decision: "accepted"},
		},
		ResidualRisk: "single host",
	})

	stamped, err := stampPerformanceReportGitHead(raw, "6ef27d0")
	if err != nil {
		t.Fatalf("stampPerformanceReportGitHead: %v", err)
	}
	if err := validatePerformanceReport(stamped); err != nil {
		t.Fatalf("stamped report should still validate: %v", err)
	}
	var report performanceReport
	if err := json.Unmarshal(stamped, &report); err != nil {
		t.Fatalf("unmarshal stamped report: %v", err)
	}
	if report.GitHead != "6ef27d0" {
		t.Fatalf("GitHead = %q, want 6ef27d0", report.GitHead)
	}
}

func mustReportJSON(t *testing.T, report performanceReport) []byte {
	t.Helper()
	totalIterations := 0
	maxNsPerOp := 0.0
	for _, metric := range report.Metrics {
		totalIterations += metric.Iterations
		if metric.NsPerOp > maxNsPerOp {
			maxNsPerOp = metric.NsPerOp
		}
	}
	report.Summary = performanceSummary{
		MetricCount:     len(report.Metrics),
		TotalIterations: totalIterations,
		MaxNsPerOp:      maxNsPerOp,
		MetricsSHA256:   "sha256:" + metricsHash(report.Metrics),
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return raw
}
