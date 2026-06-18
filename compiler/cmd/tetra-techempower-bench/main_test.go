package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateReportRequiresAllTechEmpowerEndpoints(t *testing.T) {
	report := validTestReport()
	report.Endpoints = report.Endpoints[:len(report.Endpoints)-1]
	if err := validateReport(report); err == nil || !strings.Contains(err.Error(), "/fortunes") {
		t.Fatalf("validateReport missing /fortunes = %v, want endpoint-specific error", err)
	}
}

func TestValidateReportRejectsWeakEvidenceMarkers(t *testing.T) {
	report := validTestReport()
	report.Endpoints[0].Evidence = "placeholder"
	if err := validateReport(report); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("validateReport weak evidence = %v, want placeholder rejection", err)
	}
}

func TestExerciseEndpointCountsEachFailedRequestOnce(t *testing.T) {
	result := exerciseEndpoint(
		context.Background(),
		http.DefaultClient,
		"http://127.0.0.1:1/unreachable",
		endpointSpec{
			Name: "fail",
			Path: "/fail",
			Kind: "negative",
			Validate: func(int, http.Header, []byte) error {
				return errors.New("not reached")
			},
		},
		3,
		2,
		0,
	)
	if result.successes != 0 || result.failures != 3 {
		t.Fatalf(
			"result successes=%d failures=%d, want 0 successes and 3 failures",
			result.successes,
			result.failures,
		)
	}
}

func TestRunBenchmarkRecordsLatencyPercentilesAndIntegrityMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/plaintext":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Hello, World!"))
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"Hello, World!"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := runBenchmark(context.Background(), benchOptions{
		BaseURL:             server.URL,
		RequestsPerEndpoint: 4,
		Concurrency:         2,
		MinRPS:              1,
		SkipDB:              true,
		Now: func() time.Time {
			return time.Unix(1779297600, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("runBenchmark: %v", err)
	}
	if report.GeneratedAt == "" || report.GeneratedLocalAt == "" {
		t.Fatalf("report missing generated timestamps: %#v", report)
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" ||
		report.Environment.GoVersion == "" {
		t.Fatalf("report missing environment: %#v", report.Environment)
	}
	if report.Git.WorktreeStatus == "" {
		t.Fatalf("report missing git state: %#v", report.Git)
	}
	for _, endpoint := range report.Endpoints {
		if endpoint.P50LatencyMS < 0 || endpoint.P90LatencyMS < 0 || endpoint.P95LatencyMS < 0 ||
			endpoint.P99LatencyMS < 0 ||
			endpoint.P999LatencyMS < 0 ||
			endpoint.MaxLatencyMS < 0 {
			t.Fatalf("endpoint missing latency percentiles: %#v", endpoint)
		}
		if endpoint.ObservedContentType == "" || len(endpoint.SemanticChecks) == 0 {
			t.Fatalf("endpoint missing semantic report evidence: %#v", endpoint)
		}
	}
}

func TestValidateReportAllowsFailedEndpointWithZeroRPS(t *testing.T) {
	report := validTestReport()
	failedRequests := 0
	for i := range report.Endpoints {
		if report.Endpoints[i].Path != "/db" {
			continue
		}
		failedRequests = report.Endpoints[i].Requests
		report.Endpoints[i].Status = "fail"
		report.Endpoints[i].HTTPStatus = http.StatusInternalServerError
		report.Endpoints[i].Successes = 0
		report.Endpoints[i].Failures = report.Endpoints[i].Requests
		report.Endpoints[i].RPS = 0
		report.Endpoints[i].ThresholdPass = false
		report.Endpoints[i].Error = "status = 500, want 200"
		break
	}
	report.Status = "fail"
	report.Summary.TotalSuccesses -= failedRequests
	report.Summary.TotalFailures += failedRequests
	report.Summary.Decision = "fail"
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport failed endpoint with zero RPS: %v", err)
	}
}

func TestValidateFortunesRejectsUnsortedRows(t *testing.T) {
	header := http.Header{"Content-Type": []string{"text/html; charset=utf-8"}}
	body := []byte(
		`<!DOCTYPE html><html><body><table><tr><td>0</td><td>Additional fortune added at request time.</td></tr><tr><td>11</td><td>&lt;script&gt;alert(&quot;This should not be displayed in a browser alert box.&quot;);&lt;/script&gt;</td></tr></table></body></html>`,
	)

	err := validateFortunes(http.StatusOK, header, body)
	if err == nil {
		t.Fatalf("validateFortunes accepted unsorted fortune rows")
	}
	if !strings.Contains(err.Error(), "sorted") {
		t.Fatalf("validateFortunes error = %v, want sorted rejection", err)
	}
}

func TestSemanticChecksForFortunesIncludeSortingEvidence(t *testing.T) {
	checks := strings.ToLower(strings.Join(semanticChecksForPath("/fortunes"), "\n"))
	if !strings.Contains(checks, "sorted") {
		t.Fatalf("fortunes semantic checks missing sorting evidence: %q", checks)
	}
}

func TestRunBenchmarkChecksEndpointsAndWritesReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/plaintext":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Hello, World!"))
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"Hello, World!"}`))
		case "/db":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1,"randomNumber":2}`))
		case "/queries":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"randomNumber":2},{"id":3,"randomNumber":4}]`))
		case "/updates":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"randomNumber":5},{"id":3,"randomNumber":6}]`))
		case "/fortunes":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(
				[]byte(
					`<!DOCTYPE html><html><body><table><tr><td>11</td><td>&lt;script&gt;alert(&quot;This should not be displayed in a browser alert box.&quot;);&lt;/script&gt;</td></tr><tr><td>0</td><td>Additional fortune added at request time.</td></tr></table></body></html>`,
				),
			)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := runBenchmark(context.Background(), benchOptions{
		BaseURL:             server.URL,
		RequestsPerEndpoint: 8,
		Concurrency:         4,
		MinRPS:              1,
		Now: func() time.Time {
			return time.Unix(1779297600, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("runBenchmark: %v", err)
	}
	if err := validateReport(report); err != nil {
		raw, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("validateReport: %v\n%s", err, raw)
	}
	if report.Status != "pass" || report.Summary.EndpointCount != 6 ||
		report.Summary.TotalRequests != 48 {
		t.Fatalf("summary = %#v status=%s", report.Summary, report.Status)
	}
	for _, endpoint := range report.Endpoints {
		if endpoint.ObservedContentType == "" {
			t.Fatalf("endpoint %s missing observed content type", endpoint.Path)
		}
		if len(endpoint.SemanticChecks) == 0 {
			t.Fatalf("endpoint %s missing semantic checks", endpoint.Path)
		}
	}
}

func TestRunBenchmarkFailsWhenThresholdIsMissed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/plaintext":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Hello, World!"))
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"Hello, World!"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := runBenchmark(context.Background(), benchOptions{
		BaseURL:             server.URL,
		RequestsPerEndpoint: 4,
		Concurrency:         2,
		MinRPS:              1e12,
		SkipDB:              true,
		Now: func() time.Time {
			return time.Unix(1779297600, 0).UTC()
		},
	})
	if err == nil {
		t.Fatalf("runBenchmark with impossible threshold succeeded")
	}
	if report.Status != "fail" || !strings.Contains(err.Error(), "threshold") {
		t.Fatalf("status=%s err=%v report=%#v", report.Status, err, report)
	}
	if !strings.Contains(strings.Join(report.Limitations, "\n"), "skip-db enabled") {
		t.Fatalf("skip-db limitation missing: %#v", report.Limitations)
	}
}

func validTestReport() Report {
	endpoints := make([]EndpointReport, 0, len(defaultEndpoints(false)))
	for _, endpoint := range defaultEndpoints(false) {
		endpoints = append(endpoints, EndpointReport{
			Name:                endpoint.Name,
			Path:                endpoint.Path,
			Kind:                endpoint.Kind,
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
			ObservedContentType: expectedContentType(endpoint.Path),
			SemanticChecks:      semanticChecksForPath(endpoint.Path),
			Threshold:           "min_rps >= 1",
			Evidence:            "real HTTP request/response validation and concurrent load completed",
			Validation:          "protocol/body contract checked",
			ThresholdPass:       true,
		})
	}
	return Report{
		Schema:           reportSchema,
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
			EndpointCount:  6,
			TotalRequests:  24,
			TotalSuccesses: 24,
			TotalFailures:  0,
			MinRPS:         1,
			Decision:       "pass",
		},
		Limitations: []string{"local harness report; official TechEmpower submission not implied"},
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
