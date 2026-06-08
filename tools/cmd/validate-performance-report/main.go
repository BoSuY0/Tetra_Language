package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type performanceReport struct {
	Schema            string              `json:"schema"`
	GitHead           string              `json:"git_head"`
	Host              string              `json:"host"`
	GoVersion         string              `json:"go_version"`
	Command           string              `json:"command"`
	BaselineArtifact  string              `json:"baseline_artifact"`
	ThresholdDecision string              `json:"threshold_decision"`
	Metrics           []performanceMetric `json:"metrics"`
	Summary           performanceSummary  `json:"summary"`
	ResidualRisk      string              `json:"residual_risk"`
}

type performanceMetric struct {
	Name          string  `json:"name"`
	Iterations    int     `json:"iterations"`
	NsPerOp       float64 `json:"ns_per_op"`
	ArtifactBytes float64 `json:"artifact_bytes,omitempty"`
	Threshold     string  `json:"threshold"`
	Decision      string  `json:"decision"`
}

type performanceSummary struct {
	MetricCount     int     `json:"metric_count"`
	TotalIterations int     `json:"total_iterations"`
	MaxNsPerOp      float64 `json:"max_ns_per_op"`
	MetricsSHA256   string  `json:"metrics_sha256"`
}

func main() {
	reportPath := flag.String("report", "", "path to performance regression JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validatePerformanceReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validatePerformanceReport(raw []byte) error {
	var report performanceReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if report.Schema != "tetra.performance-regression.v1" {
		return fmt.Errorf("unsupported schema %q", report.Schema)
	}
	for label, value := range map[string]string{
		"git_head":           report.GitHead,
		"host":               report.Host,
		"go_version":         report.GoVersion,
		"command":            report.Command,
		"threshold_decision": report.ThresholdDecision,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	if !strings.Contains(report.Command, "-bench=") {
		return fmt.Errorf("command must include -bench")
	}
	if strings.Contains(strings.ToLower(report.ThresholdDecision), "todo") || strings.Contains(strings.ToLower(report.ThresholdDecision), "tbd") {
		return fmt.Errorf("threshold_decision must not contain TODO/TBD placeholders")
	}
	if strings.TrimSpace(report.BaselineArtifact) == "" {
		return fmt.Errorf("baseline_artifact is required")
	}
	if strings.TrimSpace(report.ResidualRisk) == "" {
		return fmt.Errorf("residual_risk is required")
	}
	for label, value := range map[string]string{
		"threshold_decision": report.ThresholdDecision,
		"residual_risk":      report.ResidualRisk,
	} {
		if phrase, ok := forbiddenPerformanceClaim(value); ok {
			return fmt.Errorf("%s contains forbidden %s claim", label, phrase)
		}
	}
	if len(report.Metrics) == 0 {
		return fmt.Errorf("metrics must not be empty")
	}
	if report.Summary.MetricCount <= 0 {
		return fmt.Errorf("summary.metric_count must be positive")
	}
	if report.Summary.TotalIterations <= 0 {
		return fmt.Errorf("summary.total_iterations must be positive")
	}
	if report.Summary.MaxNsPerOp <= 0 {
		return fmt.Errorf("summary.max_ns_per_op must be positive")
	}
	if !strings.HasPrefix(report.Summary.MetricsSHA256, "sha256:") {
		return fmt.Errorf("summary.metrics_sha256 must use sha256: prefix")
	}

	seen := map[string]bool{}
	sortedNames := make([]string, 0, len(report.Metrics))
	totalIterations := 0
	maxNsPerOp := 0.0
	for _, metric := range report.Metrics {
		if metric.Name == "" {
			return fmt.Errorf("metric missing name")
		}
		if seen[metric.Name] {
			return fmt.Errorf("duplicate metric %s", metric.Name)
		}
		seen[metric.Name] = true
		if metric.Iterations <= 0 {
			return fmt.Errorf("metric %s iterations must be positive", metric.Name)
		}
		if metric.NsPerOp <= 0 {
			return fmt.Errorf("metric %s ns_per_op must be positive", metric.Name)
		}
		if strings.TrimSpace(metric.Threshold) == "" || strings.TrimSpace(metric.Decision) == "" {
			return fmt.Errorf("metric %s missing threshold decision", metric.Name)
		}
		for label, value := range map[string]string{
			"threshold": metric.Threshold,
			"decision":  metric.Decision,
		} {
			if phrase, ok := forbiddenPerformanceClaim(value); ok {
				return fmt.Errorf("metric %s %s contains forbidden %s claim", metric.Name, label, phrase)
			}
		}
		if strings.Contains(metric.Name, "BinarySize") && metric.ArtifactBytes <= 0 {
			return fmt.Errorf("metric %s must include positive artifact_bytes", metric.Name)
		}
		sortedNames = append(sortedNames, metric.Name)
		totalIterations += metric.Iterations
		if metric.NsPerOp > maxNsPerOp {
			maxNsPerOp = metric.NsPerOp
		}
	}
	if !sort.StringsAreSorted(sortedNames) {
		return fmt.Errorf("metrics must be sorted by name for deterministic evidence output")
	}

	if report.Summary.MetricCount != len(report.Metrics) {
		return fmt.Errorf("summary.metric_count = %d, want %d", report.Summary.MetricCount, len(report.Metrics))
	}
	if report.Summary.TotalIterations != totalIterations {
		return fmt.Errorf("summary.total_iterations = %d, want %d", report.Summary.TotalIterations, totalIterations)
	}
	if report.Summary.MaxNsPerOp != maxNsPerOp {
		return fmt.Errorf("summary.max_ns_per_op = %v, want %v", report.Summary.MaxNsPerOp, maxNsPerOp)
	}
	wantHash := "sha256:" + metricsHash(report.Metrics)
	if report.Summary.MetricsSHA256 != wantHash {
		return fmt.Errorf("summary.metrics_sha256 = %q, want %q", report.Summary.MetricsSHA256, wantHash)
	}
	return nil
}

func forbiddenPerformanceClaim(text string) (string, bool) {
	lower := strings.ToLower(text)
	if explicitNonClaimContext(lower) {
		return "", false
	}
	switch {
	case strings.Contains(lower, "fastest language") || strings.Contains(lower, "fastest-language"):
		return "fastest language", true
	case strings.Contains(lower, "official benchmark") || strings.Contains(lower, "official techempower"):
		return "official benchmark", true
	case strings.Contains(lower, "target parity") || strings.Contains(lower, "target-parity"):
		return "target parity", true
	case strings.Contains(lower, "zero-cost performance") || strings.Contains(lower, "zero cost performance"):
		return "zero-cost performance", true
	default:
		return "", false
	}
}

func explicitNonClaimContext(lower string) bool {
	for _, marker := range []string{
		"does not claim",
		"do not claim",
		"does not prove",
		"do not prove",
		"does not promote",
		"do not promote",
		"must not use",
		"not an official",
		"not a fastest",
		"not fastest",
		"not target parity",
		"not a benchmark",
		"not a full",
		"not full",
		"not a runtime measurement",
		"no official",
		"no fastest",
		"no target parity",
		"no full",
		"makes no",
		"without an official",
		"without official",
		"forbid",
		"forbidden",
		"non-claim",
		"nonclaim",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func metricsHash(metrics []performanceMetric) string {
	h := sha256.New()
	for _, metric := range metrics {
		line := metric.Name +
			"\t" + strconv.Itoa(metric.Iterations) +
			"\t" + strconv.FormatFloat(metric.NsPerOp, 'g', -1, 64) +
			"\t" + strconv.FormatFloat(metric.ArtifactBytes, 'g', -1, 64) +
			"\t" + metric.Threshold +
			"\t" + metric.Decision + "\n"
		_, _ = h.Write([]byte(line))
	}
	return hex.EncodeToString(h.Sum(nil))
}
