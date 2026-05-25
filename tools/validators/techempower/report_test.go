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

func TestValidateReportRejectsCommandSkipDBMismatch(t *testing.T) {
	skipReport := reportFixture(true)
	original := skipReport.Command
	skipReport.Command = strings.Replace(skipReport.Command, " --skip-db", "", 1)
	if skipReport.Command == original {
		t.Fatalf("test did not remove benchmark command skip-db flag")
	}
	err := ValidateReport(mustReportJSON(t, skipReport), Options{AllowSkipDB: true})
	if err == nil {
		t.Fatalf("ValidateReport accepted skip-db report without command skip-db flag")
	}
	if !strings.Contains(err.Error(), "command skip-db") {
		t.Fatalf("ValidateReport skip-db command error = %v, want command skip-db rejection", err)
	}

	fullReport := reportFixture(false)
	fullReport.Command += " --skip-db"
	err = ValidateReport(mustReportJSON(t, fullReport), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted full report with command skip-db flag")
	}
	if !strings.Contains(err.Error(), "command skip-db") {
		t.Fatalf("ValidateReport full report skip-db command error = %v, want command skip-db rejection", err)
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

func TestValidateReportRejectsCommandRequestsMismatch(t *testing.T) {
	report := reportFixture(false)
	original := report.Command
	report.Command = strings.Replace(report.Command, "--requests 4", "--requests 1", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate benchmark command requests")
	}
	raw := mustReportJSON(t, report)
	err := ValidateReport(raw, Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted report with command/request count mismatch")
	}
	if !strings.Contains(err.Error(), "command requests") {
		t.Fatalf("ValidateReport requests error = %v, want command requests rejection", err)
	}
}

func TestValidateReportRejectsCommandBaseURLMismatch(t *testing.T) {
	report := reportFixture(false)
	original := report.Command
	report.Command = strings.Replace(report.Command, "--base-url http://127.0.0.1:8080", "--base-url http://127.0.0.1:9090", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate benchmark command base-url")
	}
	raw := mustReportJSON(t, report)
	err := ValidateReport(raw, Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted report with command/base_url mismatch")
	}
	if !strings.Contains(err.Error(), "command base-url") {
		t.Fatalf("ValidateReport base-url error = %v, want command base-url rejection", err)
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

func TestValidateSCRAMMatrixRejectsWrongCommandProvenance(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	report.Command = "tetra-techempower-bench --base-url http://127.0.0.1:8080"
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with non-matrix command")
	}
	if !strings.Contains(err.Error(), "scram-local-bench") {
		t.Fatalf("ValidateReport matrix command error = %v, want scram-local-bench rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsArtifactReportPathMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	report.Artifacts["semantic_report"] = "reports/techempower/semantic-report-from-another-run.json"
	report.Artifacts["matrix_report"] = "reports/techempower/matrix-report-from-another-run.json"
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/artifact report path mismatch")
	}
	if !strings.Contains(err.Error(), "command artifact") {
		t.Fatalf("ValidateReport matrix artifact error = %v, want command artifact rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsArtifactGridFlagMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--levels 8:8,16:16", "--levels 8:8", 1)
	report.Command = strings.Replace(report.Command, "--worker-levels 1,2", "--worker-levels 1", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command grid flags")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/artifact grid flag mismatch")
	}
	if !strings.Contains(err.Error(), "command artifact") {
		t.Fatalf("ValidateReport matrix artifact grid error = %v, want command artifact rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsCommandDurationMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--duration 60s", "--duration 1s", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command duration")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/run duration mismatch")
	}
	if !strings.Contains(err.Error(), "command duration") {
		t.Fatalf("ValidateReport matrix duration error = %v, want command duration rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsCommandRepeatsMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--repeats 1", "--repeats 2", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command repeats")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/run repeat mismatch")
	}
	if !strings.Contains(err.Error(), "command repeats") {
		t.Fatalf("ValidateReport matrix repeats error = %v, want command repeats rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsCommandWarmupMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Warmup == nil {
		t.Fatalf("checked-in SCRAM matrix report has no warmup")
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--warmup 10s", "--warmup 1s", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command warmup")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/warmup duration mismatch")
	}
	if !strings.Contains(err.Error(), "command warmup") {
		t.Fatalf("ValidateReport matrix warmup error = %v, want command warmup rejection", err)
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

func TestValidateSCRAMMatrixRejectsNonMonotonicSoakTailLatency(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Soak.P999LatencyMS = report.Soak.P99LatencyMS - 0.1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted non-monotonic soak tail latency")
	}
	if !strings.Contains(err.Error(), "latency percentiles") {
		t.Fatalf("ValidateReport soak error = %v, want latency percentiles rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInvalidSoakMetrics(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Soak.DurationSeconds = 0
	report.Soak.Level.Concurrency = 0
	report.Soak.AvgLatencyMS = -1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted invalid soak metrics")
	}
	for _, want := range []string{"soak duration", "soak level", "invalid timing metrics"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateReport soak error missing %q: %v", want, err)
		}
	}
}

func TestValidateSCRAMMatrixRejectsInconsistentSoakCounters(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Soak.Successes--
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted inconsistent soak counters")
	}
	if !strings.Contains(err.Error(), "soak request counters") {
		t.Fatalf("ValidateReport soak error = %v, want request counter rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInflatedSoakRPS(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Soak.RPS = report.Soak.RPS * 2
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted inflated soak RPS")
	}
	if !strings.Contains(err.Error(), "soak rps evidence") {
		t.Fatalf("ValidateReport soak RPS error = %v, want rps evidence rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsCommandSoakMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--soak 120s", "--soak 1s", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command soak")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/soak duration mismatch")
	}
	if !strings.Contains(err.Error(), "command soak") {
		t.Fatalf("ValidateReport matrix soak error = %v, want command soak rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsCommandPoolMismatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Server.PoolSize != 64 {
		t.Fatalf("checked-in SCRAM matrix server pool_size = %d, want 64", report.Server.PoolSize)
	}

	original := report.Command
	report.Command = strings.Replace(report.Command, "--pool 64", "--pool 1", 1)
	if report.Command == original {
		t.Fatalf("test did not mutate matrix command pool")
	}
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with command/server pool mismatch")
	}
	if !strings.Contains(err.Error(), "command pool") {
		t.Fatalf("ValidateReport matrix pool error = %v, want command pool rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInvalidResourceSnapshots(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Resource.Start.PID = 0
	report.Resource.Start.ProcessAlive = false
	report.Runs[0].Resource.TCPConnections = -1
	report.Soak.ResourceEnd.CPUUserSeconds = -1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted invalid matrix resource snapshots")
	}
	for _, want := range []string{"resource.start process evidence", "resource counters"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateReport resource error missing %q: %v", want, err)
		}
	}
}

func TestValidateSCRAMMatrixRejectsMissingResourceTimestamps(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Resource.Start.Timestamp = ""
	report.Runs[0].Resource.Timestamp = ""
	report.Soak.ResourceStart.Timestamp = ""
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted missing matrix resource timestamps")
	}
	if !strings.Contains(err.Error(), "timestamp is required") {
		t.Fatalf("ValidateReport resource timestamp error = %v, want timestamp requirement", err)
	}
}

func TestValidateSCRAMMatrixRejectsRegressingResourceSpans(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Resource.End.Timestamp = report.Resource.Start.Timestamp
	report.Resource.End.CPUUserSeconds = report.Resource.Start.CPUUserSeconds - 0.1
	report.Soak.ResourceEnd.Timestamp = report.Soak.ResourceStart.Timestamp
	report.Soak.ResourceEnd.CPUSystemSeconds = report.Soak.ResourceStart.CPUSystemSeconds - 0.1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted regressing resource spans")
	}
	for _, want := range []string{"resource timestamps are not increasing", "resource CPU counters regressed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateReport resource span error missing %q: %v", want, err)
		}
	}
}

func TestValidateSCRAMMatrixRejectsResourceTimestampsOutsideReportWindow(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Runs[0].Resource.Timestamp = "2000-01-01T00:00:00Z"
	report.Soak.ResourceEnd.Timestamp = "2999-01-01T00:00:00Z"
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted resource timestamps outside report window")
	}
	if !strings.Contains(err.Error(), "resource window") {
		t.Fatalf("ValidateReport resource window error = %v, want resource window rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInvalidEndpointIdentity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}
	if report.Soak == nil {
		t.Fatalf("checked-in SCRAM matrix report has no soak evidence")
	}

	report.Runs[0].Path = "/plaintext"
	report.Soak.Path = "/plaintext"
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted invalid matrix endpoint identity")
	}
	if !strings.Contains(err.Error(), "matrix endpoint identity") {
		t.Fatalf("ValidateReport endpoint identity error = %v, want matrix endpoint identity rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsMissingArtifacts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}

	report.Artifacts = nil
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report without artifacts")
	}
	if !strings.Contains(err.Error(), "matrix artifacts") {
		t.Fatalf("ValidateReport artifacts error = %v, want matrix artifacts rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsMissingDeclaredGridCoverage(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) < 2 {
		t.Fatalf("checked-in SCRAM matrix report has too few runs")
	}

	report.Runs = append([]MatrixRun(nil), report.Runs[:len(report.Runs)-1]...)
	report.Summary = summarizeMatrixRunsForTest(report.Runs)
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with missing declared grid coverage")
	}
	if !strings.Contains(err.Error(), "matrix coverage") {
		t.Fatalf("ValidateReport coverage error = %v, want matrix coverage rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInvalidRunRepeat(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}

	report.Runs[0].Repeat = 0
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix run with invalid repeat")
	}
	if !strings.Contains(err.Error(), "repeat must be positive") {
		t.Fatalf("ValidateReport repeat error = %v, want positive repeat rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsWarmupRepeatMetadata(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if report.Warmup == nil {
		t.Fatalf("checked-in SCRAM matrix report has no warmup")
	}

	report.Warmup.Repeat = 1
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted warmup repeat metadata")
	}
	if !strings.Contains(err.Error(), "repeat must be 0") {
		t.Fatalf("ValidateReport warmup repeat error = %v, want repeat zero rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsInflatedRunRPS(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}

	report.Runs[0].RPS = report.Runs[0].RPS * 2
	report.Summary = summarizeMatrixRunsForTest(report.Runs)
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted inflated matrix run RPS")
	}
	if !strings.Contains(err.Error(), "rps evidence") {
		t.Fatalf("ValidateReport RPS error = %v, want rps evidence rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsShortElapsedRunDuration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}

	report.Runs[0].ElapsedSeconds = report.Runs[0].DurationSeconds / 2
	report.Runs[0].RPS = float64(report.Runs[0].Successes) / report.Runs[0].ElapsedSeconds
	report.Summary = summarizeMatrixRunsForTest(report.Runs)
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix run with elapsed shorter than duration")
	}
	if !strings.Contains(err.Error(), "elapsed evidence") {
		t.Fatalf("ValidateReport elapsed error = %v, want elapsed evidence rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsDuplicateRunIdentity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}

	report.Runs = append(report.Runs, report.Runs[0])
	report.Summary = summarizeMatrixRunsForTest(report.Runs)
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted duplicate matrix run identity")
	}
	if !strings.Contains(err.Error(), "duplicate matrix run") {
		t.Fatalf("ValidateReport duplicate run error = %v, want duplicate matrix run rejection", err)
	}
}

func TestValidateSCRAMMatrixRejectsMissingRepeatSequenceCoverage(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "benchmarks", "techempower_scram_single_query_matrix_local_report.json"))
	if err != nil {
		t.Fatalf("ReadFile checked-in SCRAM matrix report: %v", err)
	}
	var report MatrixReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal matrix report: %v", err)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("checked-in SCRAM matrix report has no runs")
	}

	report.Runs[0].Repeat = 2
	report.Summary = summarizeMatrixRunsForTest(report.Runs)
	err = ValidateReport(mustMatrixReportJSON(t, report), Options{})
	if err == nil {
		t.Fatalf("ValidateReport accepted matrix report with missing repeat sequence coverage")
	}
	if !strings.Contains(err.Error(), "matrix repeat coverage") {
		t.Fatalf("ValidateReport repeat coverage error = %v, want matrix repeat coverage rejection", err)
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
	command := "tetra-techempower-bench --base-url http://127.0.0.1:8080 --requests 4"
	if skipDB {
		command += " --skip-db"
	}
	return Report{
		Schema:           SchemaV1,
		Status:           "pass",
		GeneratedAt:      "2026-05-20T12:00:00Z",
		GeneratedLocalAt: "2026-05-20T15:00:00+03:00",
		BaseURL:          "http://127.0.0.1:8080",
		Command:          command,
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

func summarizeMatrixRunsForTest(runs []MatrixRun) MatrixSummary {
	summary := MatrixSummary{
		RunCount: len(runs),
		Decision: "pass",
	}
	for _, run := range runs {
		summary.TotalRequests += run.Requests
		summary.TotalFailures += run.Failures
		if run.RPS > summary.BestRPS {
			summary.BestRPS = run.RPS
		}
		if run.P99LatencyMS > summary.WorstP99MS {
			summary.WorstP99MS = run.P99LatencyMS
		}
		if run.P999LatencyMS > summary.WorstP999MS {
			summary.WorstP999MS = run.P999LatencyMS
		}
		if run.Failures != 0 || run.Successes == 0 || strings.TrimSpace(run.Error) != "" {
			summary.Decision = "fail"
		}
	}
	if len(runs) == 0 {
		summary.Decision = "fail"
	}
	return summary
}
