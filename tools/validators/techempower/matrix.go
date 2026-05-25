package techempower

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const MatrixSchemaV1 = "tetra.techempower.single_query_matrix.v1"

type MatrixReport struct {
	Schema           string               `json:"schema"`
	Status           string               `json:"status"`
	GeneratedAt      string               `json:"generated_at"`
	GeneratedLocalAt string               `json:"generated_local_at"`
	Command          string               `json:"command"`
	Environment      BenchmarkEnvironment `json:"environment"`
	Git              GitState             `json:"git"`
	Build            MatrixBuild          `json:"build"`
	Postgres         MatrixPostgres       `json:"postgres"`
	Server           MatrixServer         `json:"server"`
	Resource         MatrixResource       `json:"resource"`
	SemanticProbe    []MatrixSemantic     `json:"semantic_probe"`
	Warmup           *MatrixRun           `json:"warmup,omitempty"`
	Soak             *MatrixSoak          `json:"soak,omitempty"`
	Runs             []MatrixRun          `json:"runs"`
	Summary          MatrixSummary        `json:"summary"`
	Artifacts        map[string]string    `json:"artifacts"`
	Limitations      []string             `json:"limitations"`
}

type MatrixBuild struct {
	AppBinary       string `json:"app_binary"`
	BenchBinary     string `json:"bench_binary"`
	Mode            string `json:"mode"`
	BuildCommand    string `json:"build_command"`
	GoBuildTrimpath bool   `json:"go_build_trimpath"`
	Stripped        bool   `json:"stripped"`
}

type MatrixPostgres struct {
	Version            string `json:"version"`
	AuthMethod         string `json:"auth_method"`
	PasswordEncryption string `json:"password_encryption"`
	VerifierPrefix     string `json:"verifier_prefix"`
	WorldRows          int    `json:"world_rows"`
	FortuneRows        int    `json:"fortune_rows"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Database           string `json:"database"`
	User               string `json:"user"`
	MaxConnections     string `json:"max_connections"`
}

type MatrixServer struct {
	BaseURL      string `json:"base_url"`
	Workers      int    `json:"workers"`
	WorkerLevels []int  `json:"worker_levels"`
	PoolSize     int    `json:"pool_size"`
}

type MatrixResource struct {
	Start MatrixResourceSnapshot `json:"start"`
	End   MatrixResourceSnapshot `json:"end"`
}

type MatrixResourceSnapshot struct {
	Timestamp        string  `json:"timestamp"`
	PID              int     `json:"pid"`
	ProcessAlive     bool    `json:"process_alive"`
	RSSKB            int64   `json:"rss_kb"`
	FDCount          int     `json:"fd_count"`
	Threads          int     `json:"threads"`
	TCPConnections   int     `json:"tcp_connections"`
	CPUUserSeconds   float64 `json:"cpu_user_seconds"`
	CPUSystemSeconds float64 `json:"cpu_system_seconds"`
	Goroutines       int     `json:"goroutines,omitempty"`
}

type MatrixSemantic struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	Error    string `json:"error,omitempty"`
}

type MatrixLevel struct {
	Concurrency int `json:"concurrency"`
	Connections int `json:"connections"`
}

type MatrixSoak struct {
	Endpoint         string                 `json:"endpoint"`
	Path             string                 `json:"path"`
	Workers          int                    `json:"workers"`
	Level            MatrixLevel            `json:"level"`
	DurationSeconds  float64                `json:"duration_seconds"`
	Requests         int                    `json:"requests"`
	Successes        int                    `json:"successes"`
	Failures         int                    `json:"failures"`
	RPS              float64                `json:"rps"`
	AvgLatencyMS     float64                `json:"avg_latency_ms"`
	P99LatencyMS     float64                `json:"p99_latency_ms"`
	P999LatencyMS    float64                `json:"p999_latency_ms"`
	MaxLatencyMS     float64                `json:"max_latency_ms"`
	FirstHalfAvgMS   float64                `json:"first_half_avg_latency_ms"`
	SecondHalfAvgMS  float64                `json:"second_half_avg_latency_ms"`
	LatencyDriftMS   float64                `json:"latency_drift_ms"`
	ResourceStart    MatrixResourceSnapshot `json:"resource_start"`
	ResourceEnd      MatrixResourceSnapshot `json:"resource_end"`
	OpenSocketsAfter int                    `json:"open_sockets_after_shutdown"`
	ShutdownClean    bool                   `json:"shutdown_clean"`
	Validation       string                 `json:"validation"`
	Error            string                 `json:"error,omitempty"`
}

type MatrixRun struct {
	Endpoint        string                 `json:"endpoint"`
	Path            string                 `json:"path"`
	Kind            string                 `json:"kind"`
	Workers         int                    `json:"workers"`
	Level           MatrixLevel            `json:"level"`
	Repeat          int                    `json:"repeat"`
	DurationSeconds float64                `json:"duration_seconds"`
	ElapsedSeconds  float64                `json:"elapsed_seconds"`
	Requests        int                    `json:"requests"`
	Successes       int                    `json:"successes"`
	Failures        int                    `json:"failures"`
	Bytes           int64                  `json:"bytes"`
	RPS             float64                `json:"rps"`
	AvgLatencyMS    float64                `json:"avg_latency_ms"`
	P50LatencyMS    float64                `json:"p50_latency_ms"`
	P90LatencyMS    float64                `json:"p90_latency_ms"`
	P95LatencyMS    float64                `json:"p95_latency_ms"`
	P99LatencyMS    float64                `json:"p99_latency_ms"`
	P999LatencyMS   float64                `json:"p999_latency_ms"`
	MaxLatencyMS    float64                `json:"max_latency_ms"`
	Resource        MatrixResourceSnapshot `json:"resource"`
	Validation      string                 `json:"validation"`
	Error           string                 `json:"error,omitempty"`
}

type MatrixSummary struct {
	RunCount      int     `json:"run_count"`
	TotalRequests int     `json:"total_requests"`
	TotalFailures int     `json:"total_failures"`
	BestRPS       float64 `json:"best_rps"`
	WorstP99MS    float64 `json:"worst_p99_ms"`
	WorstP999MS   float64 `json:"worst_p999_ms"`
	Decision      string  `json:"decision"`
}

func ValidateMatrixReport(raw []byte) error {
	var report MatrixReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectWeakEvidence(raw)...)
	if report.Schema != MatrixSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, MatrixSchemaV1))
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
	if strings.TrimSpace(report.Command) == "" {
		issues = append(issues, "command is required")
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
	if err := validateBaseURL(report.Server.BaseURL); err != nil {
		issues = append(issues, "server "+err.Error())
	}
	issues = append(issues, validateMatrixBuild(report.Build)...)
	issues = append(issues, validateMatrixPostgres(report.Postgres)...)
	issues = append(issues, validateMatrixServer(report.Server)...)
	issues = append(issues, validateMatrixResource(report.Resource)...)
	issues = append(issues, validateMatrixSemanticProbe(report.SemanticProbe)...)
	if report.Warmup != nil {
		issues = append(issues, validateMatrixRun(*report.Warmup, true)...)
	}
	if report.Soak != nil {
		issues = append(issues, validateMatrixSoak(*report.Soak)...)
	}

	totalRequests := 0
	totalFailures := 0
	bestRPS := 0.0
	worstP99 := 0.0
	worstP999 := 0.0
	if len(report.Runs) == 0 {
		issues = append(issues, "runs are required")
	}
	for _, run := range report.Runs {
		issues = append(issues, validateMatrixRun(run, false)...)
		totalRequests += run.Requests
		totalFailures += run.Failures
		if run.RPS > bestRPS {
			bestRPS = run.RPS
		}
		if run.P99LatencyMS > worstP99 {
			worstP99 = run.P99LatencyMS
		}
		if run.P999LatencyMS > worstP999 {
			worstP999 = run.P999LatencyMS
		}
	}
	issues = append(issues, validateMatrixSummary(report.Summary, len(report.Runs), totalRequests, totalFailures, bestRPS, worstP99, worstP999)...)

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMatrixBuild(build MatrixBuild) []string {
	var issues []string
	if strings.TrimSpace(build.AppBinary) == "" || strings.TrimSpace(build.BenchBinary) == "" || strings.TrimSpace(build.Mode) == "" || strings.TrimSpace(build.BuildCommand) == "" {
		issues = append(issues, "build app_binary/bench_binary/mode/build_command are required")
	}
	return issues
}

func validateMatrixPostgres(pg MatrixPostgres) []string {
	var issues []string
	if pg.AuthMethod != "scram-sha-256" || pg.PasswordEncryption != "scram-sha-256" || pg.VerifierPrefix != "SCRAM-SHA-256" {
		issues = append(issues, "PostgreSQL SCRAM evidence is incomplete")
	}
	if pg.WorldRows != 10000 {
		issues = append(issues, fmt.Sprintf("postgres world_rows = %d, want 10000", pg.WorldRows))
	}
	if pg.FortuneRows < 12 {
		issues = append(issues, fmt.Sprintf("postgres fortune_rows = %d, want at least 12", pg.FortuneRows))
	}
	if strings.TrimSpace(pg.Version) == "" || strings.TrimSpace(pg.Host) == "" || pg.Port <= 0 || strings.TrimSpace(pg.Database) == "" || strings.TrimSpace(pg.User) == "" {
		issues = append(issues, "postgres version/host/port/database/user are required")
	}
	return issues
}

func validateMatrixServer(server MatrixServer) []string {
	var issues []string
	if server.Workers <= 0 || server.PoolSize <= 0 {
		issues = append(issues, "server workers and pool_size must be positive")
	}
	if len(server.WorkerLevels) == 0 {
		issues = append(issues, "server worker_levels are required")
	}
	for _, workers := range server.WorkerLevels {
		if workers <= 0 {
			issues = append(issues, "server worker_levels must be positive")
		}
	}
	return issues
}

func validateMatrixResource(resource MatrixResource) []string {
	var issues []string
	if err := validateMatrixResourceSnapshot("resource.start", resource.Start); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateMatrixResourceSnapshot("resource.end", resource.End); err != nil {
		issues = append(issues, err.Error())
	}
	return issues
}

func validateMatrixResourceSnapshot(name string, snapshot MatrixResourceSnapshot) error {
	if snapshot.RSSKB <= 0 || snapshot.FDCount <= 0 || snapshot.Threads <= 0 {
		return fmt.Errorf("%s RSS/FD/thread evidence is required", name)
	}
	if strings.TrimSpace(snapshot.Timestamp) != "" {
		if _, err := time.Parse(time.RFC3339, snapshot.Timestamp); err != nil {
			return fmt.Errorf("%s timestamp is not RFC3339: %v", name, err)
		}
	}
	return nil
}

func validateMatrixSemanticProbe(checks []MatrixSemantic) []string {
	var issues []string
	seen := make(map[string]bool, len(checks))
	for _, check := range checks {
		if strings.TrimSpace(check.Name) == "" {
			issues = append(issues, "semantic probe name is required")
		}
		if check.Status != "pass" {
			issues = append(issues, fmt.Sprintf("semantic probe %s status is %q, want pass", check.Name, check.Status))
		}
		if strings.TrimSpace(check.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("semantic probe %s evidence is required", check.Name))
		}
		if strings.TrimSpace(check.Error) != "" {
			issues = append(issues, fmt.Sprintf("semantic probe %s has error: %s", check.Name, check.Error))
		}
		seen[check.Name] = true
	}
	for _, name := range requiredMatrixSemanticProbeNames() {
		if !seen[name] {
			issues = append(issues, "semantic probe missing "+name)
		}
	}
	return issues
}

func requiredMatrixSemanticProbeNames() []string {
	return []string{
		"plaintext headers/body",
		"json headers/body",
		"db real read",
		"query clamping",
		"updates persistence",
		"fortunes insertion escaping sorting",
	}
}

func validateMatrixSoak(soak MatrixSoak) []string {
	var issues []string
	if strings.TrimSpace(soak.Endpoint) == "" || strings.TrimSpace(soak.Path) == "" || soak.Workers <= 0 {
		issues = append(issues, "soak endpoint/path/workers are required")
	}
	if soak.Requests <= 0 || soak.Successes <= 0 || soak.Failures != 0 || soak.RPS <= 0 {
		issues = append(issues, "soak evidence did not pass")
	}
	if !soak.ShutdownClean {
		issues = append(issues, "soak shutdown_clean is false")
	}
	if soak.OpenSocketsAfter != 0 {
		issues = append(issues, fmt.Sprintf("soak open_sockets_after_shutdown = %d, want 0", soak.OpenSocketsAfter))
	}
	if strings.TrimSpace(soak.Validation) == "" {
		issues = append(issues, "soak validation is required")
	}
	if strings.TrimSpace(soak.Error) != "" {
		issues = append(issues, "soak has error: "+soak.Error)
	}
	if err := validateMatrixResourceSnapshot("soak.resource_start", soak.ResourceStart); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateMatrixResourceSnapshot("soak.resource_end", soak.ResourceEnd); err != nil {
		issues = append(issues, err.Error())
	}
	return issues
}

func validateMatrixRun(run MatrixRun, warmup bool) []string {
	var issues []string
	label := fmt.Sprintf("run %s workers=%d c%d/k%d repeat=%d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat)
	if warmup {
		label = "warmup " + label
	}
	if strings.TrimSpace(run.Endpoint) == "" || strings.TrimSpace(run.Path) == "" || strings.TrimSpace(run.Kind) == "" || run.Workers <= 0 {
		issues = append(issues, label+" endpoint/path/kind/workers metadata is incomplete")
	}
	if run.Level.Concurrency <= 0 || run.Level.Connections <= 0 {
		issues = append(issues, label+" level concurrency/connections must be positive")
	}
	if run.DurationSeconds <= 0 || run.ElapsedSeconds <= 0 {
		issues = append(issues, label+" duration evidence is required")
	}
	if run.Requests <= 0 || run.Successes <= 0 || run.Failures != 0 || run.Successes+run.Failures != run.Requests {
		issues = append(issues, label+" request counters are inconsistent")
	}
	if run.Bytes <= 0 {
		issues = append(issues, label+" bytes must be positive")
	}
	if run.RPS <= 0 || run.AvgLatencyMS < 0 || run.P50LatencyMS < 0 || run.P90LatencyMS < 0 || run.P95LatencyMS < 0 || run.P99LatencyMS < 0 || run.P999LatencyMS < 0 || run.MaxLatencyMS < 0 {
		issues = append(issues, label+" has invalid timing metrics")
	}
	if run.MaxLatencyMS > 0 && run.P99LatencyMS > run.MaxLatencyMS {
		issues = append(issues, label+" p99 exceeds max")
	}
	if run.MaxLatencyMS > 0 && run.P999LatencyMS > run.MaxLatencyMS {
		issues = append(issues, label+" p999 exceeds max")
	}
	if strings.TrimSpace(run.Validation) == "" {
		issues = append(issues, label+" validation is required")
	}
	if strings.TrimSpace(run.Error) != "" {
		issues = append(issues, label+" has error: "+run.Error)
	}
	if err := validateMatrixResourceSnapshot(label+".resource", run.Resource); err != nil {
		issues = append(issues, err.Error())
	}
	return issues
}

func validateMatrixSummary(summary MatrixSummary, runCount int, totalRequests int, totalFailures int, bestRPS float64, worstP99 float64, worstP999 float64) []string {
	var issues []string
	if summary.RunCount != runCount {
		issues = append(issues, fmt.Sprintf("summary.run_count = %d, want %d", summary.RunCount, runCount))
	}
	if summary.TotalRequests != totalRequests {
		issues = append(issues, fmt.Sprintf("summary.total_requests = %d, want %d", summary.TotalRequests, totalRequests))
	}
	if summary.TotalFailures != totalFailures {
		issues = append(issues, fmt.Sprintf("summary.total_failures = %d, want %d", summary.TotalFailures, totalFailures))
	}
	if summary.TotalFailures != 0 {
		issues = append(issues, fmt.Sprintf("summary.total_failures = %d, want 0", summary.TotalFailures))
	}
	if summary.BestRPS <= 0 || summary.WorstP99MS < 0 || summary.WorstP999MS < 0 {
		issues = append(issues, "summary latency/RPS metrics are invalid")
	}
	if summary.BestRPS != bestRPS {
		issues = append(issues, fmt.Sprintf("summary.best_rps = %g, want %g", summary.BestRPS, bestRPS))
	}
	if summary.WorstP99MS != worstP99 {
		issues = append(issues, fmt.Sprintf("summary.worst_p99_ms = %g, want %g", summary.WorstP99MS, worstP99))
	}
	if summary.WorstP999MS != worstP999 {
		issues = append(issues, fmt.Sprintf("summary.worst_p999_ms = %g, want %g", summary.WorstP999MS, worstP999))
	}
	if summary.Decision != "pass" {
		issues = append(issues, fmt.Sprintf("summary.decision is %q, want pass", summary.Decision))
	}
	return issues
}
