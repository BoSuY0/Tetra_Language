package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	if len(report.Metrics) == 0 {
		return fmt.Errorf("metrics must not be empty")
	}
	seen := map[string]bool{}
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
	}
	return nil
}
