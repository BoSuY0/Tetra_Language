package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const reportSchema = "tetra.techempower.benchmark.v1"

type endpointSpec struct {
	Name           string
	Path           string
	Kind           string
	SemanticChecks []string
	Validate       func(int, http.Header, []byte) error
}

type benchOptions struct {
	BaseURL             string
	ReportPath          string
	Duration            time.Duration
	RequestsPerEndpoint int
	Concurrency         int
	MinRPS              float64
	SkipDB              bool
	Now                 func() time.Time
	Client              *http.Client
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

type endpointResult struct {
	httpStatus  int
	bytes       int64
	contentType string
	latencies   []time.Duration
	successes   int
	failures    int
	err         error
	elapsed     time.Duration
}

func main() {
	var opt benchOptions
	flag.StringVar(
		&opt.BaseURL,
		"base-url",
		"http://127.0.0.1:8080",
		"base URL of a running tetra-techempower server",
	)
	flag.StringVar(
		&opt.ReportPath,
		"report",
		"",
		"path to write tetra.techempower.benchmark.v1 JSON report",
	)
	flag.DurationVar(
		&opt.Duration,
		"duration",
		0,
		"duration per endpoint; if zero, --requests controls the run",
	)
	flag.IntVar(
		&opt.RequestsPerEndpoint,
		"requests",
		256,
		"fixed requests per endpoint when --duration is zero",
	)
	flag.IntVar(&opt.Concurrency, "concurrency", 32, "concurrent clients per endpoint")
	flag.Float64Var(&opt.MinRPS, "min-rps", 1, "minimum requests/second required for each endpoint")
	flag.BoolVar(
		&opt.SkipDB,
		"skip-db",
		false,
		"run only /plaintext and /json for local no-database smoke",
	)
	flag.Parse()
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	report, err := runBenchmark(context.Background(), opt)
	if writeErr := writeReport(opt.ReportPath, report); writeErr != nil && err == nil {
		err = writeErr
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runBenchmark(ctx context.Context, opt benchOptions) (Report, error) {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Client == nil {
		opt.Client = &http.Client{Timeout: 10 * time.Second}
	}
	if opt.Concurrency <= 0 {
		opt.Concurrency = 1
	}
	if opt.Duration <= 0 && opt.RequestsPerEndpoint <= 0 {
		opt.RequestsPerEndpoint = 1
	}
	if opt.MinRPS <= 0 {
		opt.MinRPS = 1
	}
	base, err := normalizeBaseURL(opt.BaseURL)
	if err != nil {
		return Report{}, err
	}
	now := opt.Now()

	report := Report{
		Schema:           reportSchema,
		Status:           "pass",
		GeneratedAt:      now.UTC().Format(time.RFC3339),
		GeneratedLocalAt: now.Local().Format(time.RFC3339),
		BaseURL:          base,
		Command:          commandString(opt),
		Environment:      detectEnvironment(),
		Git:              detectGitState(),
		Limitations: []string{
			"local harness evidence; official TechEmpower publication is not implied",
			"duration/request counts are caller controlled and should be raised for release gates",
		},
	}
	if opt.SkipDB {
		report.Limitations = append(
			report.Limitations,
			"skip-db enabled: report covers only /plaintext and /json",
		)
	}
	for _, endpoint := range defaultEndpoints(opt.SkipDB) {
		endpointReport := runEndpoint(ctx, opt, base, endpoint)
		report.Endpoints = append(report.Endpoints, endpointReport)
		if endpointReport.Status != "pass" {
			report.Status = "fail"
		}
	}
	report.Summary = summarize(report.Endpoints, opt.MinRPS)
	if report.Summary.Decision != "pass" {
		report.Status = "fail"
	}
	if err := validateReport(report); err != nil {
		return report, err
	}
	if report.Status != "pass" {
		return report, errors.New("TechEmpower benchmark threshold or validation failed")
	}
	return report, nil
}

func runEndpoint(
	ctx context.Context,
	opt benchOptions,
	base string,
	endpoint endpointSpec,
) EndpointReport {
	result := exerciseEndpoint(
		ctx,
		opt.Client,
		base+endpoint.Path,
		endpoint,
		opt.RequestsPerEndpoint,
		opt.Concurrency,
		opt.Duration,
	)
	requests := result.successes + result.failures
	elapsed := result.elapsed.Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	latency := latencySummary(result.latencies)
	report := EndpointReport{
		Name:                endpoint.Name,
		Path:                endpoint.Path,
		Kind:                endpoint.Kind,
		Status:              "pass",
		HTTPStatus:          result.httpStatus,
		Requests:            requests,
		Successes:           result.successes,
		Failures:            result.failures,
		Bytes:               result.bytes,
		RPS:                 float64(result.successes) / elapsed,
		AvgLatencyMS:        averageMS(result.latencies),
		P50LatencyMS:        latency.P50MS,
		P90LatencyMS:        latency.P90MS,
		P95LatencyMS:        latency.P95MS,
		P99LatencyMS:        latency.P99MS,
		P999LatencyMS:       latency.P999MS,
		MaxLatencyMS:        latency.MaxMS,
		ObservedContentType: result.contentType,
		SemanticChecks:      append([]string(nil), endpoint.SemanticChecks...),
		Threshold:           fmt.Sprintf("min_rps >= %.2f", opt.MinRPS),
		ThresholdPass:       float64(result.successes)/elapsed >= opt.MinRPS,
		Validation:          "HTTP status, content type, and endpoint body contract checked",
		Evidence:            "real HTTP request/response validation and concurrent load completed",
	}
	if result.err != nil {
		report.Error = result.err.Error()
	}
	if result.err != nil || report.Failures != 0 || !report.ThresholdPass {
		report.Status = "fail"
	}
	return report
}

func exerciseEndpoint(
	ctx context.Context,
	client *http.Client,
	target string,
	endpoint endpointSpec,
	requests int,
	concurrency int,
	duration time.Duration,
) endpointResult {
	start := time.Now()
	jobs := make(chan struct{})
	results := make(chan singleResult, concurrency)
	workerCount := concurrency
	if requests > 0 && requests < workerCount {
		workerCount = requests
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				results <- oneRequest(ctx, client, target, endpoint)
			}
		}()
	}
	go func() {
		defer close(jobs)
		if duration > 0 {
			deadline := time.NewTimer(duration)
			defer deadline.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-deadline.C:
					return
				case jobs <- struct{}{}:
				}
			}
		}
		for i := 0; i < requests; i++ {
			select {
			case <-ctx.Done():
				return
			case jobs <- struct{}{}:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var out endpointResult
	for result := range results {
		out.httpStatus = result.httpStatus
		out.bytes += int64(result.bytes)
		if out.contentType == "" && result.contentType != "" {
			out.contentType = result.contentType
		}
		out.latencies = append(out.latencies, result.latency)
		if result.err != nil {
			out.failures++
			if out.err == nil {
				out.err = result.err
			}
			continue
		}
		out.successes++
	}
	out.elapsed = time.Since(start)
	return out
}

type singleResult struct {
	httpStatus  int
	bytes       int
	contentType string
	latency     time.Duration
	err         error
}

func oneRequest(
	ctx context.Context,
	client *http.Client,
	target string,
	endpoint endpointSpec,
) singleResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return singleResult{err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return singleResult{err: err, latency: time.Since(start)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return singleResult{httpStatus: resp.StatusCode, latency: time.Since(start), err: err}
	}
	if err := endpoint.Validate(resp.StatusCode, resp.Header, body); err != nil {
		return singleResult{
			httpStatus:  resp.StatusCode,
			bytes:       len(body),
			contentType: resp.Header.Get("Content-Type"),
			latency:     time.Since(start),
			err:         err,
		}
	}
	return singleResult{
		httpStatus:  resp.StatusCode,
		bytes:       len(body),
		contentType: resp.Header.Get("Content-Type"),
		latency:     time.Since(start),
	}
}

func defaultEndpoints(skipDB bool) []endpointSpec {
	endpoints := []endpointSpec{
		{
			Name:           "plaintext",
			Path:           "/plaintext",
			Kind:           "plaintext",
			SemanticChecks: semanticChecksForPath("/plaintext"),
			Validate:       validatePlaintext,
		},
		{
			Name:           "json",
			Path:           "/json",
			Kind:           "json",
			SemanticChecks: semanticChecksForPath("/json"),
			Validate:       validateJSON,
		},
	}
	if skipDB {
		return endpoints
	}
	return append(
		endpoints,
		endpointSpec{
			Name:           "db",
			Path:           "/db",
			Kind:           "single-query",
			SemanticChecks: semanticChecksForPath("/db"),
			Validate:       validateWorldObject,
		},
		endpointSpec{
			Name:           "queries",
			Path:           "/queries?queries=2",
			Kind:           "multiple-queries",
			SemanticChecks: semanticChecksForPath("/queries?queries=2"),
			Validate:       validateWorldArray,
		},
		endpointSpec{
			Name:           "updates",
			Path:           "/updates?queries=2",
			Kind:           "updates",
			SemanticChecks: semanticChecksForPath("/updates?queries=2"),
			Validate:       validateWorldArray,
		},
		endpointSpec{
			Name:           "fortunes",
			Path:           "/fortunes",
			Kind:           "fortunes",
			SemanticChecks: semanticChecksForPath("/fortunes"),
			Validate:       validateFortunes,
		},
	)
}

func semanticChecksForPath(path string) []string {
	switch path {
	case "/plaintext":
		return []string{"status 200", "content-type text/plain", "body equals Hello, World!"}
	case "/json":
		return []string{
			"status 200",
			"content-type application/json",
			"JSON message equals Hello, World!",
		}
	case "/db":
		return []string{
			"status 200",
			"content-type application/json",
			"World object id/randomNumber range",
		}
	case "/queries?queries=2":
		return []string{"status 200", "content-type application/json", "World array shape"}
	case "/updates?queries=2":
		return []string{"status 200", "content-type application/json", "World update array shape"}
	case "/fortunes":
		return []string{
			"status 200",
			"content-type text/html",
			"request-time fortune present",
			"HTML escaping sentinel",
			"sorted Fortune rows",
		}
	default:
		return []string{"status 200"}
	}
}

func validatePlaintext(status int, header http.Header, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("status = %d, want 200", status)
	}
	if !strings.HasPrefix(header.Get("Content-Type"), "text/plain") {
		return fmt.Errorf("content-type = %q, want text/plain", header.Get("Content-Type"))
	}
	if string(body) != "Hello, World!" {
		return fmt.Errorf("plaintext body = %q", body)
	}
	return nil
}

func validateJSON(status int, header http.Header, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("content-type = %q, want application/json", header.Get("Content-Type"))
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	if payload.Message != "Hello, World!" {
		return fmt.Errorf("message = %q", payload.Message)
	}
	return nil
}

func validateWorldObject(status int, header http.Header, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("content-type = %q, want application/json", header.Get("Content-Type"))
	}
	var world struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal(body, &world); err != nil {
		return err
	}
	return validateWorld(world.ID, world.RandomNumber)
}

func validateWorldArray(status int, header http.Header, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("content-type = %q, want application/json", header.Get("Content-Type"))
	}
	var worlds []struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal(body, &worlds); err != nil {
		return err
	}
	if len(worlds) == 0 {
		return errors.New("world array is empty")
	}
	for _, world := range worlds {
		if err := validateWorld(world.ID, world.RandomNumber); err != nil {
			return err
		}
	}
	return nil
}

func validateWorld(id int, randomNumber int) error {
	if id < 1 || id > 10000 {
		return fmt.Errorf("world id = %d, want 1..10000", id)
	}
	if randomNumber < 1 || randomNumber > 10000 {
		return fmt.Errorf("randomNumber = %d, want 1..10000", randomNumber)
	}
	return nil
}

func validateFortunes(status int, header http.Header, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("content-type = %q, want text/html", header.Get("Content-Type"))
	}
	text := string(body)
	if !strings.Contains(text, "<table>") ||
		!strings.Contains(text, "Additional fortune added at request time.") {
		return errors.New("fortunes HTML missing table or request-time fortune")
	}
	rawScript := `<script>alert("This should not be displayed in a browser alert box.");</script>`
	if strings.Contains(text, rawScript) {
		return errors.New("fortunes HTML contains raw XSS sentinel")
	}
	if !strings.Contains(text, "&lt;script&gt;") {
		return errors.New("fortunes HTML missing escaped XSS sentinel")
	}
	if !orderedMarkers(
		text,
		[]string{"&lt;script&gt;", "Additional fortune added at request time."},
	) {
		return errors.New("fortunes HTML rows are not sorted by message")
	}
	return nil
}

func orderedMarkers(text string, markers []string) bool {
	offset := 0
	for _, marker := range markers {
		idx := strings.Index(text[offset:], marker)
		if idx < 0 {
			return false
		}
		offset += idx + len(marker)
	}
	return true
}

func validateReport(report Report) error {
	var issues []string
	if report.Schema != reportSchema {
		issues = append(issues, fmt.Sprintf("schema = %q, want %q", report.Schema, reportSchema))
	}
	if report.Status != "pass" && report.Status != "fail" {
		issues = append(issues, fmt.Sprintf("status = %q, want pass or fail", report.Status))
	}
	for label, value := range map[string]string{
		"generated_at":       report.GeneratedAt,
		"generated_local_at": report.GeneratedLocalAt,
		"base_url":           report.BaseURL,
		"command":            report.Command,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, label+" is required")
		}
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" ||
		report.Environment.GoVersion == "" ||
		report.Environment.Hostname == "" {
		issues = append(issues, "environment os/arch/go_version/hostname are required")
	}
	if report.Git.WorktreeStatus == "" {
		issues = append(issues, "git worktree_status is required")
	}
	required := map[string]bool{}
	for _, endpoint := range defaultEndpoints(false) {
		required[endpoint.Path] = false
	}
	seen := map[string]bool{}
	totalRequests := 0
	totalSuccesses := 0
	totalFailures := 0
	for _, endpoint := range report.Endpoints {
		if seen[endpoint.Path] {
			issues = append(issues, fmt.Sprintf("duplicate endpoint %s", endpoint.Path))
		}
		seen[endpoint.Path] = true
		if _, ok := required[endpoint.Path]; ok {
			required[endpoint.Path] = true
		}
		issues = append(issues, validateEndpointReport(endpoint)...)
		totalRequests += endpoint.Requests
		totalSuccesses += endpoint.Successes
		totalFailures += endpoint.Failures
	}
	if !hasOnlySmokeEndpoints(report.Endpoints) {
		for path, ok := range required {
			if !ok {
				issues = append(issues, "missing endpoint "+path)
			}
		}
	}
	if report.Summary.EndpointCount != len(report.Endpoints) {
		issues = append(
			issues,
			fmt.Sprintf(
				"summary.endpoint_count = %d, want %d",
				report.Summary.EndpointCount,
				len(report.Endpoints),
			),
		)
	}
	if report.Summary.TotalRequests != totalRequests {
		issues = append(
			issues,
			fmt.Sprintf(
				"summary.total_requests = %d, want %d",
				report.Summary.TotalRequests,
				totalRequests,
			),
		)
	}
	if report.Summary.TotalSuccesses != totalSuccesses {
		issues = append(
			issues,
			fmt.Sprintf(
				"summary.total_successes = %d, want %d",
				report.Summary.TotalSuccesses,
				totalSuccesses,
			),
		)
	}
	if report.Summary.TotalFailures != totalFailures {
		issues = append(
			issues,
			fmt.Sprintf(
				"summary.total_failures = %d, want %d",
				report.Summary.TotalFailures,
				totalFailures,
			),
		)
	}
	if report.Summary.Decision != "pass" && report.Summary.Decision != "fail" {
		issues = append(
			issues,
			fmt.Sprintf("summary.decision = %q, want pass or fail", report.Summary.Decision),
		)
	}
	if len(report.Limitations) == 0 {
		issues = append(issues, "limitations are required")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateEndpointReport(endpoint EndpointReport) []string {
	var issues []string
	if strings.TrimSpace(endpoint.Name) == "" || strings.TrimSpace(endpoint.Path) == "" ||
		strings.TrimSpace(endpoint.Kind) == "" {
		issues = append(issues, "endpoint identity is required")
	}
	if endpoint.Status != "pass" && endpoint.Status != "fail" {
		issues = append(
			issues,
			fmt.Sprintf(
				"endpoint %s status = %q, want pass or fail",
				endpoint.Path,
				endpoint.Status,
			),
		)
	}
	if endpoint.Requests <= 0 || endpoint.Successes < 0 || endpoint.Failures < 0 ||
		endpoint.Successes+endpoint.Failures != endpoint.Requests {
		issues = append(
			issues,
			fmt.Sprintf("endpoint %s request counters are inconsistent", endpoint.Path),
		)
	}
	if endpoint.HTTPStatus <= 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s missing HTTP status", endpoint.Path))
	}
	if endpoint.RPS < 0 || endpoint.AvgLatencyMS < 0 || endpoint.P50LatencyMS < 0 ||
		endpoint.P90LatencyMS < 0 ||
		endpoint.P95LatencyMS < 0 ||
		endpoint.P99LatencyMS < 0 ||
		endpoint.P999LatencyMS < 0 ||
		endpoint.MaxLatencyMS < 0 {
		issues = append(
			issues,
			fmt.Sprintf("endpoint %s has invalid timing metrics", endpoint.Path),
		)
	}
	if endpoint.Status == "pass" && strings.TrimSpace(endpoint.ObservedContentType) == "" {
		issues = append(
			issues,
			fmt.Sprintf("endpoint %s missing observed content type", endpoint.Path),
		)
	}
	if endpoint.Status == "pass" && len(endpoint.SemanticChecks) == 0 {
		issues = append(issues, fmt.Sprintf("endpoint %s missing semantic checks", endpoint.Path))
	}
	if strings.TrimSpace(endpoint.Threshold) == "" ||
		strings.TrimSpace(endpoint.Validation) == "" ||
		strings.TrimSpace(endpoint.Evidence) == "" {
		issues = append(
			issues,
			fmt.Sprintf("endpoint %s missing threshold/validation/evidence", endpoint.Path),
		)
	}
	lower := strings.ToLower(endpoint.Evidence + " " + endpoint.Validation)
	for _, marker := range []string{"placeholder", "todo", "metadata-only", "fake benchmark"} {
		if strings.Contains(lower, marker) {
			issues = append(
				issues,
				fmt.Sprintf("endpoint %s contains weak evidence marker %q", endpoint.Path, marker),
			)
		}
	}
	if endpoint.Status == "pass" && (!endpoint.ThresholdPass || endpoint.Failures != 0) {
		issues = append(
			issues,
			fmt.Sprintf("endpoint %s pass status contradicts counters/threshold", endpoint.Path),
		)
	}
	return issues
}

func hasOnlySmokeEndpoints(endpoints []EndpointReport) bool {
	if len(endpoints) != 2 {
		return false
	}
	seen := map[string]bool{}
	for _, endpoint := range endpoints {
		seen[endpoint.Path] = true
	}
	return seen["/plaintext"] && seen["/json"]
}

func summarize(endpoints []EndpointReport, minRPS float64) Summary {
	summary := Summary{EndpointCount: len(endpoints), MinRPS: minRPS, Decision: "pass"}
	for _, endpoint := range endpoints {
		summary.TotalRequests += endpoint.Requests
		summary.TotalSuccesses += endpoint.Successes
		summary.TotalFailures += endpoint.Failures
		if endpoint.Status != "pass" {
			summary.Decision = "fail"
		}
	}
	return summary
}

func normalizeBaseURL(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("base URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("base URL scheme = %q, want http or https", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", errors.New("base URL host is required")
	}
	return strings.TrimRight(raw, "/"), nil
}

func commandString(opt benchOptions) string {
	parts := []string{
		"tetra-techempower-bench",
		"--base-url", opt.BaseURL,
		"--requests", fmt.Sprintf("%d", opt.RequestsPerEndpoint),
		"--concurrency", fmt.Sprintf("%d", opt.Concurrency),
		"--min-rps", fmt.Sprintf("%.2f", opt.MinRPS),
	}
	if opt.Duration > 0 {
		parts = append(parts, "--duration", opt.Duration.String())
	}
	if opt.SkipDB {
		parts = append(parts, "--skip-db")
	}
	return strings.Join(parts, " ")
}

func writeReport(path string, report Report) error {
	if path == "" {
		return errors.New("report path is required")
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func detectEnvironment() BenchmarkEnvironment {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown"
	}
	return BenchmarkEnvironment{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		Hostname:  hostname,
	}
}

func detectGitState() GitState {
	head := strings.TrimSpace(runGit("rev-parse", "--short=12", "HEAD"))
	if head == "" {
		head = "unknown"
	}
	status := strings.TrimSpace(runGit("status", "--porcelain", "--untracked-files=all"))
	worktreeStatus := "clean"
	if status != "" {
		worktreeStatus = "dirty"
	}
	return GitState{Head: head, WorktreeStatus: worktreeStatus}
}

func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func averageMS(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values)) / float64(time.Millisecond)
}

func percentileMS(values []time.Duration, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return float64(sorted[idx]) / float64(time.Millisecond)
}

type latencyStats struct {
	P50MS  float64
	P90MS  float64
	P95MS  float64
	P99MS  float64
	P999MS float64
	MaxMS  float64
}

func latencySummary(values []time.Duration) latencyStats {
	if len(values) == 0 {
		return latencyStats{}
	}
	return latencyStats{
		P50MS:  percentileMS(values, 0.50),
		P90MS:  percentileMS(values, 0.90),
		P95MS:  percentileMS(values, 0.95),
		P99MS:  percentileMS(values, 0.99),
		P999MS: percentileMS(values, 0.999),
		MaxMS:  maxMS(values),
	}
}

func maxMS(values []time.Duration) float64 {
	var max time.Duration
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return float64(max) / float64(time.Millisecond)
}
