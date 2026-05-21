package main

import (
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
		SemanticProbe: []semanticCheck{{Name: "db", Status: "pass", Evidence: "real DB read"}},
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
}
