package techempower

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

const SchemaV1 = "tetra.techempower.benchmark.v1"

type Options struct {
	AllowSkipDB bool
}

type Report struct {
	Schema           string               `json:"schema"`
	Status           string               `json:"status"`
	GeneratedAt      string               `json:"generated_at"`
	GeneratedLocalAt string               `json:"generated_local_at"`
	BaseURL          string               `json:"base_url"`
	Command          string               `json:"command"`
	Environment      BenchmarkEnvironment `json:"environment"`
	Git              GitState             `json:"git"`
	Endpoints        []EndpointReport     `json:"endpoints"`
	Summary          Summary              `json:"summary"`
	Limitations      []string             `json:"limitations"`
}

type BenchmarkEnvironment struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	Hostname  string `json:"hostname"`
}

type GitState struct {
	Head           string `json:"head"`
	WorktreeStatus string `json:"worktree_status"`
}

type EndpointReport struct {
	Name                string   `json:"name"`
	Path                string   `json:"path"`
	Kind                string   `json:"kind"`
	Status              string   `json:"status"`
	HTTPStatus          int      `json:"http_status"`
	Requests            int      `json:"requests"`
	Successes           int      `json:"successes"`
	Failures            int      `json:"failures"`
	Bytes               int64    `json:"bytes"`
	RPS                 float64  `json:"rps"`
	AvgLatencyMS        float64  `json:"avg_latency_ms"`
	P50LatencyMS        float64  `json:"p50_latency_ms"`
	P90LatencyMS        float64  `json:"p90_latency_ms"`
	P95LatencyMS        float64  `json:"p95_latency_ms"`
	P99LatencyMS        float64  `json:"p99_latency_ms"`
	P999LatencyMS       float64  `json:"p999_latency_ms"`
	MaxLatencyMS        float64  `json:"max_latency_ms"`
	ObservedContentType string   `json:"observed_content_type"`
	SemanticChecks      []string `json:"semantic_checks"`
	Threshold           string   `json:"threshold"`
	ThresholdPass       bool     `json:"threshold_pass"`
	Validation          string   `json:"validation"`
	Evidence            string   `json:"evidence"`
	Error               string   `json:"error,omitempty"`
}

type Summary struct {
	EndpointCount  int     `json:"endpoint_count"`
	TotalRequests  int     `json:"total_requests"`
	TotalSuccesses int     `json:"total_successes"`
	TotalFailures  int     `json:"total_failures"`
	MinRPS         float64 `json:"min_rps"`
	Decision       string  `json:"decision"`
}

func ValidateReport(raw []byte, opt Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectWeakEvidence(raw)...)
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("generated_at is not RFC3339: %v", err))
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedLocalAt); err != nil {
		issues = append(issues, fmt.Sprintf("generated_local_at is not RFC3339: %v", err))
	}
	if err := validateBaseURL(report.BaseURL); err != nil {
		issues = append(issues, err.Error())
	}
	if strings.TrimSpace(report.Command) == "" {
		issues = append(issues, "command is required")
	} else if !strings.Contains(report.Command, "tetra-techempower-bench") {
		issues = append(issues, "command must include tetra-techempower-bench")
	}
	if len(report.Limitations) == 0 {
		issues = append(issues, "limitations are required")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.GoVersion == "" || report.Environment.Hostname == "" {
		issues = append(issues, "environment os/arch/go_version/hostname are required")
	}
	if report.Git.WorktreeStatus != "clean" && report.Git.WorktreeStatus != "dirty" {
		issues = append(issues, fmt.Sprintf("git worktree_status is %q, want clean or dirty", report.Git.WorktreeStatus))
	}

	totalRequests := 0
	totalSuccesses := 0
	totalFailures := 0
	seen := map[string]bool{}
	for _, endpoint := range report.Endpoints {
		if seen[endpoint.Path] {
			issues = append(issues, fmt.Sprintf("duplicate endpoint %s", endpoint.Path))
		}
		seen[endpoint.Path] = true
		issues = append(issues, validateEndpointReport(endpoint)...)
		totalRequests += endpoint.Requests
		totalSuccesses += endpoint.Successes
		totalFailures += endpoint.Failures
	}
	issues = append(issues, validateEndpointSet(report, opt)...)
	issues = append(issues, validateSummary(report.Summary, len(report.Endpoints), totalRequests, totalSuccesses, totalFailures)...)

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateEndpointSet(report Report, opt Options) []string {
	var issues []string
	required := []string{"/plaintext", "/json", "/db", "/queries?queries=2", "/updates?queries=2", "/fortunes"}
	seen := map[string]bool{}
	for _, endpoint := range report.Endpoints {
		seen[endpoint.Path] = true
	}
	if isSkipDBReport(seen) {
		if !hasSkipDBLimitation(report.Limitations) {
			issues = append(issues, "skip-db report must include explicit skip-db limitation")
		}
		if !opt.AllowSkipDB {
			issues = append(issues, "skip-db report requires --allow-skip-db")
		}
		return issues
	}
	for _, path := range required {
		if !seen[path] {
			issues = append(issues, "missing endpoint "+path)
		}
	}
	return issues
}

func isSkipDBReport(seen map[string]bool) bool {
	return len(seen) == 2 && seen["/plaintext"] && seen["/json"]
}

func hasSkipDBLimitation(limitations []string) bool {
	for _, limitation := range limitations {
		if strings.Contains(strings.ToLower(limitation), "skip-db") {
			return true
		}
	}
	return false
}

func validateEndpointReport(endpoint EndpointReport) []string {
	var issues []string
	if strings.TrimSpace(endpoint.Name) == "" || strings.TrimSpace(endpoint.Path) == "" || strings.TrimSpace(endpoint.Kind) == "" {
		issues = append(issues, "endpoint identity is required")
	}
	if endpoint.Status != "pass" {
		issues = append(issues, fmt.Sprintf("endpoint %s status is %q, want pass", endpoint.Path, endpoint.Status))
	}
	if endpoint.HTTPStatus != 200 {
		issues = append(issues, fmt.Sprintf("endpoint %s http_status = %d, want 200", endpoint.Path, endpoint.HTTPStatus))
	}
	if endpoint.Requests <= 0 || endpoint.Successes < 0 || endpoint.Failures < 0 || endpoint.Successes+endpoint.Failures != endpoint.Requests {
		issues = append(issues, fmt.Sprintf("endpoint %s request counters are inconsistent", endpoint.Path))
	}
	if endpoint.Failures != 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s failures = %d, want 0", endpoint.Path, endpoint.Failures))
	}
	if endpoint.Bytes <= 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s bytes = %d, want > 0", endpoint.Path, endpoint.Bytes))
	}
	if endpoint.RPS <= 0 || endpoint.AvgLatencyMS < 0 || endpoint.P50LatencyMS < 0 || endpoint.P90LatencyMS < 0 || endpoint.P95LatencyMS < 0 || endpoint.P99LatencyMS < 0 || endpoint.P999LatencyMS < 0 || endpoint.MaxLatencyMS < 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s has invalid timing metrics", endpoint.Path))
	}
	if endpoint.MaxLatencyMS > 0 && endpoint.P99LatencyMS > endpoint.MaxLatencyMS {
		issues = append(issues, fmt.Sprintf("endpoint %s p99 latency exceeds max latency", endpoint.Path))
	}
	if endpoint.MaxLatencyMS > 0 && endpoint.P999LatencyMS > endpoint.MaxLatencyMS {
		issues = append(issues, fmt.Sprintf("endpoint %s p999 latency exceeds max latency", endpoint.Path))
	}
	if strings.TrimSpace(endpoint.ObservedContentType) == "" {
		issues = append(issues, fmt.Sprintf("endpoint %s missing observed content type", endpoint.Path))
	} else if expected := expectedContentTypePrefix(endpoint.Path); expected != "" && !strings.HasPrefix(endpoint.ObservedContentType, expected) {
		issues = append(issues, fmt.Sprintf("endpoint %s observed content type = %q, want prefix %q", endpoint.Path, endpoint.ObservedContentType, expected))
	}
	if len(endpoint.SemanticChecks) == 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s missing semantic checks", endpoint.Path))
	} else {
		issues = append(issues, validateSemanticChecks(endpoint.Path, endpoint.SemanticChecks)...)
	}
	if strings.TrimSpace(endpoint.Threshold) == "" || strings.TrimSpace(endpoint.Validation) == "" || strings.TrimSpace(endpoint.Evidence) == "" {
		issues = append(issues, fmt.Sprintf("endpoint %s missing threshold/validation/evidence", endpoint.Path))
	}
	if !endpoint.ThresholdPass {
		issues = append(issues, fmt.Sprintf("endpoint %s threshold_pass is false", endpoint.Path))
	}
	if strings.TrimSpace(endpoint.Error) != "" {
		issues = append(issues, fmt.Sprintf("endpoint %s has error: %s", endpoint.Path, endpoint.Error))
	}
	return issues
}

func expectedContentTypePrefix(path string) string {
	switch path {
	case "/plaintext":
		return "text/plain"
	case "/json", "/db", "/queries?queries=2", "/updates?queries=2":
		return "application/json"
	case "/fortunes":
		return "text/html"
	default:
		return ""
	}
}

func validateSemanticChecks(path string, checks []string) []string {
	var issues []string
	joined := strings.ToLower(strings.Join(checks, "\n"))
	for _, want := range requiredSemanticCheckMarkers(path) {
		if !strings.Contains(joined, strings.ToLower(want)) {
			issues = append(issues, fmt.Sprintf("endpoint %s semantic checks missing %q", path, want))
		}
	}
	return issues
}

func requiredSemanticCheckMarkers(path string) []string {
	switch path {
	case "/plaintext":
		return []string{"status 200", "content-type text/plain", "body equals Hello, World!"}
	case "/json":
		return []string{"status 200", "content-type application/json", "JSON message equals Hello, World!"}
	case "/db":
		return []string{"status 200", "content-type application/json", "World object id/randomNumber range"}
	case "/queries?queries=2":
		return []string{"status 200", "content-type application/json", "World array shape"}
	case "/updates?queries=2":
		return []string{"status 200", "content-type application/json", "World update array shape"}
	case "/fortunes":
		return []string{"status 200", "content-type text/html", "request-time fortune present", "HTML escaping sentinel"}
	default:
		return nil
	}
}

func validateSummary(summary Summary, endpointCount int, totalRequests int, totalSuccesses int, totalFailures int) []string {
	var issues []string
	if summary.EndpointCount != endpointCount {
		issues = append(issues, fmt.Sprintf("summary.endpoint_count = %d, want %d", summary.EndpointCount, endpointCount))
	}
	if summary.TotalRequests != totalRequests {
		issues = append(issues, fmt.Sprintf("summary.total_requests = %d, want %d", summary.TotalRequests, totalRequests))
	}
	if summary.TotalSuccesses != totalSuccesses {
		issues = append(issues, fmt.Sprintf("summary.total_successes = %d, want %d", summary.TotalSuccesses, totalSuccesses))
	}
	if summary.TotalFailures != totalFailures {
		issues = append(issues, fmt.Sprintf("summary.total_failures = %d, want %d", summary.TotalFailures, totalFailures))
	}
	if summary.TotalFailures != 0 {
		issues = append(issues, fmt.Sprintf("summary.total_failures = %d, want 0", summary.TotalFailures))
	}
	if summary.MinRPS <= 0 {
		issues = append(issues, fmt.Sprintf("summary.min_rps = %g, want > 0", summary.MinRPS))
	}
	if summary.Decision != "pass" {
		issues = append(issues, fmt.Sprintf("summary.decision is %q, want pass", summary.Decision))
	}
	return issues
}

func validateBaseURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("base_url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("base_url scheme is %q, want http or https", parsed.Scheme)
	}
	if parsed.Host == "" {
		return errors.New("base_url host is required")
	}
	return nil
}

func rejectWeakEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{
		"placeholder",
		"todo",
		"tbd",
		"metadata-only",
		"docs-only",
		"report-only",
		"fake benchmark",
		" fake ",
		" mock ",
	}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("report contains forbidden weak evidence marker %q", strings.TrimSpace(marker)))
		}
	}
	return issues
}

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("report contains trailing JSON values")
	}
	return nil
}
