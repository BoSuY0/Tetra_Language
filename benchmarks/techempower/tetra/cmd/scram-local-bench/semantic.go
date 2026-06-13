package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runSemanticProbe(ctx context.Context, client *http.Client, baseURL string, db *sql.DB) []semanticCheck {
	var checks []semanticCheck
	check := func(name string, fn func() (string, error)) {
		evidence, err := fn()
		if err != nil {
			checks = append(checks, semanticCheck{Name: name, Status: "fail", Error: err.Error()})
			return
		}
		checks = append(checks, semanticCheck{Name: name, Status: "pass", Evidence: evidence})
	}
	check("plaintext headers/body", func() (string, error) {
		resp, body, err := get(ctx, client, baseURL+"/plaintext")
		if err != nil {
			return "", err
		}
		if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") || string(body) != "Hello, World!" {
			return "", fmt.Errorf("unexpected plaintext response: status=%d content-type=%q body=%q", resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}
		if resp.Header.Get("Date") == "" || resp.Header.Get("Server") == "" {
			return "", errors.New("Date/Server headers are required")
		}
		return "status, text/plain body, Date, and Server headers validated", nil
	})
	check("json headers/body", func() (string, error) {
		resp, body, err := get(ctx, client, baseURL+"/json")
		if err != nil {
			return "", err
		}
		if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			return "", fmt.Errorf("unexpected /json headers: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		var payload struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return "", err
		}
		if payload.Message != "Hello, World!" {
			return "", fmt.Errorf("message = %q", payload.Message)
		}
		return "JSON object shape and content type validated", nil
	})
	check("db real read", func() (string, error) {
		resp, body, err := get(ctx, client, baseURL+"/db")
		if err != nil {
			return "", err
		}
		w, err := decodeWorldResponse(resp, body)
		if err != nil {
			return "", err
		}
		var persisted int
		if err := db.QueryRowContext(ctx, "SELECT randomNumber FROM World WHERE id=$1", w.ID).Scan(&persisted); err != nil {
			return "", err
		}
		if persisted != w.RandomNumber {
			return "", fmt.Errorf("World[%d] response randomNumber=%d db=%d", w.ID, w.RandomNumber, persisted)
		}
		return fmt.Sprintf("/db World[%d] matched PostgreSQL randomNumber=%d", w.ID, w.RandomNumber), nil
	})
	check("query clamping", func() (string, error) {
		if worlds, err := getWorldArray(ctx, client, baseURL+"/queries?queries=0"); err != nil {
			return "", err
		} else if len(worlds) != 1 {
			return "", fmt.Errorf("queries=0 length=%d, want 1", len(worlds))
		}
		if worlds, err := getWorldArray(ctx, client, baseURL+"/queries?queries=501"); err != nil {
			return "", err
		} else if len(worlds) != 500 {
			return "", fmt.Errorf("queries=501 length=%d, want 500", len(worlds))
		}
		return "queries parameter clamps to 1..500", nil
	})
	check("updates persistence", func() (string, error) {
		worlds, err := getWorldArray(ctx, client, baseURL+"/updates?queries=2")
		if err != nil {
			return "", err
		}
		if len(worlds) != 2 {
			return "", fmt.Errorf("updates length=%d, want 2", len(worlds))
		}
		for _, w := range worlds {
			var persisted int
			if err := db.QueryRowContext(ctx, "SELECT randomNumber FROM World WHERE id=$1", w.ID).Scan(&persisted); err != nil {
				return "", err
			}
			if persisted != w.RandomNumber {
				return "", fmt.Errorf("World[%d] update response=%d db=%d", w.ID, w.RandomNumber, persisted)
			}
		}
		return "updates response persisted changed randomNumber values", nil
	})
	check("fortunes insertion escaping sorting", func() (string, error) {
		resp, body, err := get(ctx, client, baseURL+"/fortunes")
		if err != nil {
			return "", err
		}
		if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			return "", fmt.Errorf("unexpected /fortunes headers: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		html := string(body)
		rawScript := `<script>alert("This should not be displayed in a browser alert box.");</script>`
		if strings.Contains(html, rawScript) {
			return "", errors.New("raw XSS sentinel leaked")
		}
		for _, want := range []string{"Additional fortune added at request time.", "&lt;script&gt;", "&quot;This should not be displayed in a browser alert box.&quot;"} {
			if !strings.Contains(html, want) {
				return "", fmt.Errorf("missing fortune marker %q", want)
			}
		}
		if !orderedMarkers(html, []string{"&lt;script&gt;", "A bad random number generator", "Additional fortune added", "After enough decimal places", "fortune: No such file or directory"}) {
			return "", errors.New("fortune rows are not sorted by message")
		}
		return "request-time fortune, HTML escaping, and sorted message order validated", nil
	})
	return checks
}

func semanticFailed(checks []semanticCheck) bool {
	if len(checks) == 0 {
		return true
	}
	for _, check := range checks {
		if check.Status != "pass" {
			return true
		}
	}
	return false
}

func get(ctx context.Context, client *http.Client, target string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, nil, err
	}
	return resp, body, nil
}

func decodeWorldResponse(resp *http.Response, body []byte) (world, error) {
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return world{}, fmt.Errorf("unexpected World response: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var w world
	if err := json.Unmarshal(body, &w); err != nil {
		return world{}, err
	}
	return w, validateWorld(w)
}

func validateWorldHTTP(resp *http.Response, body []byte) error {
	_, err := decodeWorldResponse(resp, body)
	return err
}

func validateWorldArrayHTTP(resp *http.Response, body []byte) error {
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("unexpected World array response: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var worlds []world
	if err := json.Unmarshal(body, &worlds); err != nil {
		return err
	}
	if len(worlds) == 0 {
		return errors.New("world array is empty")
	}
	for _, w := range worlds {
		if err := validateWorld(w); err != nil {
			return err
		}
	}
	return nil
}

func validateFortunesHTTP(resp *http.Response, body []byte) error {
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("unexpected /fortunes response: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	html := string(body)
	rawScript := `<script>alert("This should not be displayed in a browser alert box.");</script>`
	if strings.Contains(html, rawScript) {
		return errors.New("fortunes HTML contains raw XSS sentinel")
	}
	for _, want := range []string{"<table>", "Additional fortune added at request time.", "&lt;script&gt;"} {
		if !strings.Contains(html, want) {
			return fmt.Errorf("fortunes HTML missing %q", want)
		}
	}
	return nil
}

func getWorldArray(ctx context.Context, client *http.Client, target string) ([]world, error) {
	resp, body, err := get(ctx, client, target)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return nil, fmt.Errorf("unexpected World array response: status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var worlds []world
	if err := json.Unmarshal(body, &worlds); err != nil {
		return nil, err
	}
	for _, w := range worlds {
		if err := validateWorld(w); err != nil {
			return nil, err
		}
	}
	return worlds, nil
}

func validateWorld(w world) error {
	if w.ID < 1 || w.ID > 10000 {
		return fmt.Errorf("world id=%d, want 1..10000", w.ID)
	}
	if w.RandomNumber < 1 || w.RandomNumber > 10000 {
		return fmt.Errorf("randomNumber=%d, want 1..10000", w.RandomNumber)
	}
	return nil
}

func orderedMarkers(text string, markers []string) bool {
	pos := -1
	for _, marker := range markers {
		idx := strings.Index(text, marker)
		if idx < 0 || idx < pos {
			return false
		}
		pos = idx
	}
	return true
}

func runSemanticReport(ctx context.Context, root string, benchBin string, baseURL string, opt options) error {
	reportPath := absPath(root, opt.SemanticReportPath)
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(
		ctx,
		benchBin,
		"--base-url", baseURL,
		"--report", reportPath,
		"--requests", strconv.Itoa(opt.SemanticRequests),
		"--concurrency", strconv.Itoa(opt.SemanticConcurrency),
		"--min-rps", "1",
	)
	cmd.Dir = root
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("semantic benchmark failed: %w\n%s", err, combined.String())
	}
	return nil
}

func validateSemanticReport(raw []byte) error {
	var report struct {
		Schema           string `json:"schema"`
		Status           string `json:"status"`
		GeneratedAt      string `json:"generated_at"`
		GeneratedLocalAt string `json:"generated_local_at"`
		Environment      struct {
			OS        string `json:"os"`
			Arch      string `json:"arch"`
			GoVersion string `json:"go_version"`
			Hostname  string `json:"hostname"`
		} `json:"environment"`
		Git struct {
			WorktreeStatus string `json:"worktree_status"`
		} `json:"git"`
		Endpoints []struct {
			Path          string  `json:"path"`
			Status        string  `json:"status"`
			HTTPStatus    int     `json:"http_status"`
			Requests      int     `json:"requests"`
			Successes     int     `json:"successes"`
			Failures      int     `json:"failures"`
			RPS           float64 `json:"rps"`
			P50LatencyMS  float64 `json:"p50_latency_ms"`
			P90LatencyMS  float64 `json:"p90_latency_ms"`
			P95LatencyMS  float64 `json:"p95_latency_ms"`
			P99LatencyMS  float64 `json:"p99_latency_ms"`
			P999LatencyMS float64 `json:"p999_latency_ms"`
			MaxLatencyMS  float64 `json:"max_latency_ms"`
		} `json:"endpoints"`
		Summary struct {
			Decision      string `json:"decision"`
			TotalFailures int    `json:"total_failures"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != "tetra.techempower.benchmark.v1" {
		issues = append(issues, "unexpected semantic report schema")
	}
	if report.Status != "pass" || report.Summary.Decision != "pass" || report.Summary.TotalFailures != 0 {
		issues = append(issues, "semantic report did not pass")
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		issues = append(issues, "generated_at is not RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedLocalAt); err != nil {
		issues = append(issues, "generated_local_at is not RFC3339")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.GoVersion == "" || report.Environment.Hostname == "" {
		issues = append(issues, "environment metadata is incomplete")
	}
	if report.Git.WorktreeStatus != "clean" && report.Git.WorktreeStatus != "dirty" {
		issues = append(issues, "git worktree status is incomplete")
	}
	required := map[string]bool{
		"/plaintext":         false,
		"/json":              false,
		"/db":                false,
		"/queries?queries=2": false,
		"/updates?queries=2": false,
		"/fortunes":          false,
	}
	for _, endpoint := range report.Endpoints {
		if _, ok := required[endpoint.Path]; ok {
			required[endpoint.Path] = true
		}
		if endpoint.Status != "pass" || endpoint.HTTPStatus != 200 || endpoint.Requests <= 0 || endpoint.Successes <= 0 || endpoint.Failures != 0 || endpoint.RPS <= 0 {
			issues = append(issues, "semantic endpoint did not pass: "+endpoint.Path)
		}
		if endpoint.P50LatencyMS < 0 || endpoint.P90LatencyMS < 0 || endpoint.P95LatencyMS < 0 || endpoint.P99LatencyMS < 0 || endpoint.P999LatencyMS < 0 || endpoint.MaxLatencyMS < 0 {
			issues = append(issues, "semantic endpoint missing latency metrics: "+endpoint.Path)
		}
	}
	for path, seen := range required {
		if !seen {
			issues = append(issues, "semantic report missing "+path)
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}
