package techempower

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReportAcceptsFullSixEndpointReport(t *testing.T) {
	raw := mustReportJSON(t, reportFixture(false))
	if err := ValidateReport(raw, Options{}); err != nil {
		t.Fatalf("ValidateReport full: %v", err)
	}
}

func TestValidateReportRequiresExplicitSkipDBAllowance(t *testing.T) {
	raw := mustReportJSON(t, reportFixture(true))
	if err := ValidateReport(raw, Options{}); err == nil || !strings.Contains(err.Error(), "skip-db") {
		t.Fatalf("ValidateReport skip-db without allowance = %v, want skip-db error", err)
	}
	if err := ValidateReport(raw, Options{AllowSkipDB: true}); err != nil {
		t.Fatalf("ValidateReport skip-db allowed: %v", err)
	}
}

func TestValidateReportRejectsWeakEvidenceAndBadCounters(t *testing.T) {
	report := reportFixture(false)
	report.Endpoints[0].Evidence = "placeholder"
	raw := mustReportJSON(t, report)
	if err := ValidateReport(raw, Options{}); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("ValidateReport weak evidence = %v, want placeholder rejection", err)
	}

	report = reportFixture(false)
	report.Endpoints[0].Successes--
	raw = mustReportJSON(t, report)
	if err := ValidateReport(raw, Options{}); err == nil || !strings.Contains(err.Error(), "request counters") {
		t.Fatalf("ValidateReport bad counters = %v, want counter rejection", err)
	}
}

func TestValidateReportRequiresLatencyPercentilesAndIntegrityMetadata(t *testing.T) {
	report := reportFixture(false)
	report.GeneratedLocalAt = ""
	report.Environment.GoVersion = ""
	report.Git.WorktreeStatus = ""
	report.Endpoints[0].P99LatencyMS = -1
	report.Endpoints[1].P999LatencyMS = -1
	report.Endpoints[2].ObservedContentType = ""
	report.Endpoints[3].SemanticChecks = nil
	raw := mustReportJSON(t, report)
	err := ValidateReport(raw, Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted missing integrity metadata")
	}
	for _, want := range []string{"generated_local_at", "environment", "git", "invalid timing metrics", "observed content type", "semantic checks"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateReport error missing %q: %v", want, err)
		}
	}
}

func TestValidateReportRejectsNonMonotonicLatencyPercentiles(t *testing.T) {
	report := reportFixture(false)
	report.Endpoints[0].P90LatencyMS = report.Endpoints[0].P50LatencyMS - 0.1
	raw := mustReportJSON(t, report)
	err := ValidateReport(raw, Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted non-monotonic latency percentiles")
	}
	if !strings.Contains(err.Error(), "latency percentiles") {
		t.Fatalf("ValidateReport error = %v, want latency percentiles rejection", err)
	}
}

func TestValidateCheckedInSmokeReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_local_smoke_skip_db_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in smoke report: %v", err)
	}
	if err := ValidateReport(raw, Options{AllowSkipDB: true}); err != nil {
		t.Fatalf("ValidateReport checked-in smoke: %v", err)
	}
}

func TestValidateCheckedInSCRAMReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM report: %v", err)
	}
	if err := ValidateReport(raw, Options{}); err != nil {
		t.Fatalf("ValidateReport checked-in SCRAM report: %v", err)
	}
}

func TestValidateCheckedInSCRAMMatrixReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	if err := ValidateReport(raw, Options{}); err != nil {
		t.Fatalf("ValidateReport checked-in SCRAM matrix report: %v", err)
	}
}

func TestValidateSCRAMMatrixRejectsWeakEvidenceAndSummaryMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	weak := report
	weak.Postgres.VerifierPrefix = "md5"
	if err := ValidateReport(mustMatrixReportJSON(t, weak), Options{}); err == nil || !strings.Contains(err.Error(), "SCRAM") {
		t.Fatalf("ValidateReport weak matrix SCRAM evidence = %v, want SCRAM error", err)
	}

	mismatch := report
	mismatch.Summary.TotalRequests--
	if err := ValidateReport(mustMatrixReportJSON(t, mismatch), Options{}); err == nil || !strings.Contains(err.Error(), "summary.total_requests") {
		t.Fatalf("ValidateReport matrix summary mismatch = %v, want summary.total_requests error", err)
	}
}

func TestValidateSCRAMMatrixRejectsNonMonotonicLatencyPercentiles(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	report.Runs[0].P90LatencyMS = report.Runs[0].P50LatencyMS - 0.1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted non-monotonic matrix latency percentiles")
	}
	if !strings.Contains(err.Error(), "latency percentiles") {
		t.Fatalf("ValidateReport matrix error = %v, want latency percentiles rejection", err)
	}
}

func reportFixture(skipDB bool) Report {
	paths := []string{"/plaintext", "/json", "/db", "/queries?queries=2", "/updates?queries=2", "/fortunes"}
	if skipDB {
		paths = []string{"/plaintext", "/json"}
	}
	endpoints := make([]EndpointReport, 0, len(paths))
	for _, path := range paths {
		endpoints = append(endpoints, EndpointReport{
			Name:                endpointName(path),
			Path:                path,
			Kind:                "contract",
			Status:              "pass",
			HTTPStatus:          200,
			Requests:            4,
			Successes:           4,
			Failures:            0,
			Bytes:               52,
			RPS:                 100,
			AvgLatencyMS:        1,
			P50LatencyMS:        1,
			P90LatencyMS:        2,
			P95LatencyMS:        2,
			P99LatencyMS:        2,
			P999LatencyMS:       2,
			MaxLatencyMS:        2,
			ObservedContentType: expectedContentType(path),
			SemanticChecks:      semanticChecksForPath(path),
			Threshold:           "min_rps >= 1",
			ThresholdPass:       true,
			Validation:          "HTTP status, content type, and endpoint body contract checked",
			Evidence:            "real HTTP request/response validation and concurrent load completed",
		})
	}
	limitations := []string{"local harness evidence; official TechEmpower publication is not implied"}
	if skipDB {
		limitations = append(limitations, "skip-db enabled: report covers only /plaintext and /json")
	}
	return Report{
		Schema:           SchemaV1,
		Status:           "pass",
		GeneratedAt:      "2026-05-20T12:00:00Z",
		GeneratedLocalAt: "2026-05-20T15:00:00+03:00",
		BaseURL:          "http://127.0.0.1:8080",
		Command:          "tetra-techempower-bench --base-url http://127.0.0.1:8080 --requests 4",
		Environment: BenchmarkEnvironment{
			OS:        "linux",
			Arch:      "amd64",
			GoVersion: "go1.20",
			Hostname:  "test-host",
		},
		Git: GitState{
			Head:           "test-head",
			WorktreeStatus: "dirty",
		},
		Endpoints: endpoints,
		Summary: Summary{
			EndpointCount:  len(endpoints),
			TotalRequests:  len(endpoints) * 4,
			TotalSuccesses: len(endpoints) * 4,
			TotalFailures:  0,
			MinRPS:         1,
			Decision:       "pass",
		},
		Limitations: limitations,
	}
}

func expectedContentType(path string) string {
	switch path {
	case "/plaintext":
		return "text/plain"
	case "/fortunes":
		return "text/html; charset=utf-8"
	default:
		return "application/json"
	}
}

func semanticChecksForPath(path string) []string {
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
		return []string{"status 200"}
	}
}

func endpointName(path string) string {
	switch path {
	case "/queries?queries=2":
		return "queries"
	case "/updates?queries=2":
		return "updates"
	default:
		return strings.TrimPrefix(path, "/")
	}
}

func mustReportJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return raw
}

func mustMatrixReportJSON(t *testing.T, report MatrixReport) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return raw
}
