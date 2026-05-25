package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLevels(t *testing.T) {
	levels, err := parseLevels("8:8, 16:12")
	if err != nil {
		t.Fatalf("parseLevels: %v", err)
	}
	if len(levels) != 2 || levels[0].Concurrency != 8 || levels[0].Connections != 8 || levels[1].Concurrency != 16 || levels[1].Connections != 12 {
		t.Fatalf("levels = %#v", levels)
	}

	for _, raw := range []string{"", "8", "0:1", "1:0", "x:1", "1:y", "1:1,"} {
		if _, err := parseLevels(raw); err == nil {
			t.Fatalf("parseLevels(%q) succeeded, want error", raw)
		}
	}
}

func TestParseEndpointNamesAndWorkerLevels(t *testing.T) {
	endpoints, err := parseEndpointNames("db,queries, updates ,fortunes")
	if err != nil {
		t.Fatalf("parseEndpointNames: %v", err)
	}
	if strings.Join(endpoints, ",") != "db,queries,updates,fortunes" {
		t.Fatalf("endpoints = %#v", endpoints)
	}
	for _, raw := range []string{"", "db,unknown", "db,,queries"} {
		if _, err := parseEndpointNames(raw); err == nil {
			t.Fatalf("parseEndpointNames(%q) succeeded, want error", raw)
		}
	}

	workers, err := parsePositiveIntList("1, 2,4", "--worker-levels")
	if err != nil {
		t.Fatalf("parsePositiveIntList: %v", err)
	}
	if len(workers) != 3 || workers[0] != 1 || workers[1] != 2 || workers[2] != 4 {
		t.Fatalf("workers = %#v", workers)
	}
	for _, raw := range []string{"", "1,0", "x", "1,"} {
		if _, err := parsePositiveIntList(raw, "--worker-levels"); err == nil {
			t.Fatalf("parsePositiveIntList(%q) succeeded, want error", raw)
		}
	}
}

func TestBuildPlanForModeKeepsReleaseDefaultAndAddsProfileSymbols(t *testing.T) {
	release := buildPlanForMode(false, "/tmp/app", "./compiler/cmd/tetra-techempower")
	if release.Mode != "release" {
		t.Fatalf("release mode = %q, want release", release.Mode)
	}
	if !release.GoBuildTrimpath || !release.Stripped {
		t.Fatalf("release evidence = trimpath %v stripped %v, want true/true", release.GoBuildTrimpath, release.Stripped)
	}
	releaseCommand := strings.Join(release.Args, " ")
	if !strings.Contains(releaseCommand, "-trimpath") || !strings.Contains(releaseCommand, "-ldflags=-s -w") {
		t.Fatalf("release command = %q, want trimpath and stripped ldflags", releaseCommand)
	}

	profile := buildPlanForMode(true, "/tmp/app", "./compiler/cmd/tetra-techempower")
	if profile.Mode != "profile" {
		t.Fatalf("profile mode = %q, want profile", profile.Mode)
	}
	if profile.GoBuildTrimpath || profile.Stripped {
		t.Fatalf("profile evidence = trimpath %v stripped %v, want false/false", profile.GoBuildTrimpath, profile.Stripped)
	}
	profileCommand := strings.Join(profile.Args, " ")
	if strings.Contains(profileCommand, "-trimpath") || strings.Contains(profileCommand, "-ldflags=-s -w") {
		t.Fatalf("profile command = %q, want no trimpath or stripped ldflags", profileCommand)
	}
	if !strings.Contains(profileCommand, "-gcflags=all=-N -l") {
		t.Fatalf("profile command = %q, want debug gcflags", profileCommand)
	}
}

func TestServerEnvIncludesPprofAddrOnlyWhenRequested(t *testing.T) {
	base := strings.Join(serverEnv(8080, 5432, options{Workers: 1, PoolSize: 4}), "\n")
	if strings.Contains(base, "TETRA_TE_PPROF_ADDR") {
		t.Fatalf("serverEnv without pprof includes pprof addr:\n%s", base)
	}

	withPprof := strings.Join(serverEnv(8080, 5432, options{Workers: 1, PoolSize: 4, PprofAddr: "127.0.0.1:6060"}), "\n")
	if !strings.Contains(withPprof, "TETRA_TE_PPROF_ADDR=127.0.0.1:6060") {
		t.Fatalf("serverEnv with pprof missing addr:\n%s", withPprof)
	}
}

func TestCapturePprofProfilesWritesCPUAndHeap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/pprof/profile":
			_, _ = w.Write([]byte("cpu profile"))
		case "/debug/pprof/heap":
			_, _ = w.Write([]byte("heap profile"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	artifacts, err := capturePprofProfiles(context.Background(), server.URL, t.TempDir(), time.Millisecond)
	if err != nil {
		t.Fatalf("capturePprofProfiles: %v", err)
	}
	for _, path := range []string{artifacts.CPUProfile, artifacts.HeapProfile} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read profile %s: %v", path, err)
		}
		if len(raw) == 0 {
			t.Fatalf("profile %s is empty", path)
		}
	}
}

func TestRewritePGHBAForSCRAM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pg_hba.conf")
	input := strings.Join([]string{
		"# comment",
		"",
		"local   all             all                                     password",
		"host    all             all             127.0.0.1/32            md5",
		"hostssl all             all             ::1/128                 scram-sha-256",
	}, "\n")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := rewritePGHBAForSCRAM(path); err != nil {
		t.Fatalf("rewritePGHBAForSCRAM: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, " password") || strings.Contains(text, " md5") {
		t.Fatalf("pg_hba.conf still contains weak auth:\n%s", text)
	}
	if got := strings.Count(text, "scram-sha-256"); got != 3 {
		t.Fatalf("scram count = %d, want 3:\n%s", got, text)
	}
}

func TestLatencySummary(t *testing.T) {
	values := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	stats := latencySummary(values)
	if stats.P50MS != 30 || stats.P90MS != 50 || stats.P95MS != 50 || stats.P99MS != 50 || stats.P999MS != 50 || stats.MaxMS != 50 {
		t.Fatalf("latency stats = %#v", stats)
	}
}

func TestOrderedMarkers(t *testing.T) {
	if !orderedMarkers("a b c", []string{"a", "b", "c"}) {
		t.Fatalf("orderedMarkers rejected ordered input")
	}
	if orderedMarkers("a c b", []string{"a", "b", "c"}) {
		t.Fatalf("orderedMarkers accepted out-of-order input")
	}
	if orderedMarkers("a b", []string{"a", "b", "c"}) {
		t.Fatalf("orderedMarkers accepted missing marker")
	}
}

func TestValidateMatrixReportRequiresSCRAMEvidenceAndPassingRuns(t *testing.T) {
	report := matrixReport{
		Schema:           matrixSchema,
		Status:           "pass",
		GeneratedAt:      "2026-05-20T12:00:00Z",
		GeneratedLocalAt: "2026-05-20T15:00:00+03:00",
		Command:          "techempower-scram-local-bench --duration 1s",
		Postgres: postgresEvidence{
			AuthMethod:         "scram-sha-256",
			PasswordEncryption: "scram-sha-256",
			VerifierPrefix:     "SCRAM-SHA-256",
		},
		Resource: resourceEvidence{
			Start: resourceSnapshot{RSSKB: 1024, FDCount: 8, Threads: 4},
			End:   resourceSnapshot{RSSKB: 2048, FDCount: 8, Threads: 4},
		},
		SemanticProbe: validSemanticProbeFixture(),
		Soak: &soakReport{
			Endpoint:        "db",
			DurationSeconds: 1,
			Requests:        1,
			Successes:       1,
			Failures:        0,
			LatencyDriftMS:  0,
			ResourceStart:   resourceSnapshot{RSSKB: 1024, FDCount: 8, Threads: 4},
			ResourceEnd:     resourceSnapshot{RSSKB: 2048, FDCount: 8, Threads: 4},
			ShutdownClean:   true,
		},
		Runs: []dbRunReport{{
			Endpoint:      "db",
			Path:          "/db",
			Kind:          "single-query",
			Workers:       1,
			Level:         benchLevel{Concurrency: 1, Connections: 1},
			Repeat:        1,
			Requests:      1,
			Successes:     1,
			Failures:      0,
			RPS:           1,
			P99LatencyMS:  1,
			P999LatencyMS: 1,
			MaxLatencyMS:  1,
			Resource:      resourceSnapshot{RSSKB: 2048, FDCount: 8, Threads: 4},
		}},
		Summary: matrixSummary{RunCount: 1, TotalRequests: 1, Decision: "pass"},
	}
	if err := validateMatrixReport(report); err != nil {
		t.Fatalf("validateMatrixReport valid report: %v", err)
	}
	report.Postgres.VerifierPrefix = "md5"
	if err := validateMatrixReport(report); err == nil || !strings.Contains(err.Error(), "SCRAM") {
		t.Fatalf("validateMatrixReport weak SCRAM evidence = %v, want SCRAM error", err)
	}
	report.Postgres.VerifierPrefix = "SCRAM-SHA-256"
	report.SemanticProbe = report.SemanticProbe[:len(report.SemanticProbe)-1]
	if err := validateMatrixReport(report); err == nil || !strings.Contains(err.Error(), "semantic probe missing") {
		t.Fatalf("validateMatrixReport missing semantic probe = %v, want semantic coverage error", err)
	}
}

func TestValidateCheckedInSCRAMMatrixReports(t *testing.T) {
	for _, name := range []string{
		"techempower_scram_single_query_matrix_local_report.json",
		"techempower_scram_endpoint_matrix_local_report.json",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "..", "..", "docs", "benchmarks", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			var report matrixReport
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatalf("json.Unmarshal %s: %v", path, err)
			}
			if err := validateMatrixReport(report); err != nil {
				t.Fatalf("validateMatrixReport %s: %v", path, err)
			}
		})
	}
}

func validSemanticProbeFixture() []semanticCheck {
	return []semanticCheck{
		{Name: "plaintext headers/body", Status: "pass", Evidence: "status, text/plain body, Date, and Server headers validated"},
		{Name: "json headers/body", Status: "pass", Evidence: "JSON object shape and content type validated"},
		{Name: "db real read", Status: "pass", Evidence: "/db World[1] matched PostgreSQL randomNumber=2"},
		{Name: "query clamping", Status: "pass", Evidence: "queries parameter clamps to 1..500"},
		{Name: "updates persistence", Status: "pass", Evidence: "updates response persisted changed randomNumber values"},
		{Name: "fortunes insertion escaping sorting", Status: "pass", Evidence: "request-time fortune, HTML escaping, and sorted message order validated"},
	}
}
