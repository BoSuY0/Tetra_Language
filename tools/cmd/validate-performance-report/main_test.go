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
			{
				Name:          "BenchmarkBinarySize/foo-8",
				Iterations:    10,
				NsPerOp:       1000,
				ArtifactBytes: 4096,
				Threshold:     "compile <= 15 percent slower",
				Decision:      "accepted",
			},
			{
				Name:       "BenchmarkCompile/foo-8",
				Iterations: 12,
				NsPerOp:    1200,
				Threshold:  "compile <= 15 percent slower",
				Decision:   "accepted",
			},
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
			{
				Name:       "BenchmarkCompile/z",
				Iterations: 10,
				NsPerOp:    10,
				Threshold:  "x",
				Decision:   "accepted",
			},
			{
				Name:       "BenchmarkCompile/a",
				Iterations: 10,
				NsPerOp:    10,
				Threshold:  "x",
				Decision:   "accepted",
			},
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
			{
				Name:       "BenchmarkCompile/a",
				Iterations: 10,
				NsPerOp:    10,
				Threshold:  "x",
				Decision:   "accepted",
			},
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

func TestValidatePerformanceReportRejectsFastestOrOfficialClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(performanceReport) performanceReport
		want   string
	}{
		{
			name: "fastest language threshold decision",
			mutate: func(report performanceReport) performanceReport {
				report.ThresholdDecision = "Tetra is the fastest language for this benchmark"
				return report
			},
			want: "fastest language",
		},
		{
			name: "official benchmark metric decision",
			mutate: func(report performanceReport) performanceReport {
				report.Metrics[0].Decision = "official benchmark result accepted"
				return report
			},
			want: "official benchmark",
		},
		{
			name: "target parity residual risk",
			mutate: func(report performanceReport) performanceReport {
				report.ResidualRisk = "no residual risk; this proves target parity"
				return report
			},
			want: "target parity",
		},
		{
			name: "broad zero cost performance threshold",
			mutate: func(report performanceReport) performanceReport {
				report.Metrics[0].Threshold = "broad zero-cost performance claim accepted"
				return report
			},
			want: "zero-cost performance",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := performanceReport{
				Schema:            "tetra.performance-regression.v1",
				GitHead:           "abc123",
				Host:              "linux amd64",
				GoVersion:         "go1.20",
				Command:           "go test ./compiler -bench=. -run '^$' -count=1",
				BaselineArtifact:  "docs/performance/v1_0_thresholds.md",
				ThresholdDecision: "baseline capture approved",
				Metrics: []performanceMetric{
					{
						Name:       "BenchmarkCompile/a",
						Iterations: 10,
						NsPerOp:    10,
						Threshold:  "local threshold",
						Decision:   "accepted",
					},
				},
				ResidualRisk: "single host",
			}
			raw := mustReportJSON(t, tc.mutate(report))
			err := validatePerformanceReport(raw)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("validatePerformanceReport error = %v, want %q rejection", err, tc.want)
			}
		})
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
