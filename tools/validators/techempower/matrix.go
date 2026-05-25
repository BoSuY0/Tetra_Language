package techempower

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
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
	issues = append(issues, validateMatrixArtifacts(report.Artifacts)...)
	issues = append(issues, validateMatrixCoverage(report.Artifacts, report.Server, report.Runs)...)

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMatrixArtifacts(artifacts map[string]string) []string {
	var issues []string
	required := []string{"semantic_report", "matrix_report", "endpoints", "levels", "worker_levels"}
	if len(artifacts) == 0 {
		return []string{"matrix artifacts are required"}
	}
	for _, key := range required {
		if strings.TrimSpace(artifacts[key]) == "" {
			issues = append(issues, "matrix artifacts missing "+key)
		}
	}
	return issues
}

func validateMatrixCoverage(artifacts map[string]string, server MatrixServer, runs []MatrixRun) []string {
	if strings.TrimSpace(artifacts["endpoints"]) == "" || strings.TrimSpace(artifacts["levels"]) == "" || strings.TrimSpace(artifacts["worker_levels"]) == "" {
		return nil
	}

	var issues []string
	endpoints, endpointIssues := parseMatrixArtifactEndpoints(artifacts["endpoints"])
	levels, levelIssues := parseMatrixArtifactLevels(artifacts["levels"])
	workers, workerIssues := parseMatrixArtifactWorkers(artifacts["worker_levels"])
	issues = append(issues, endpointIssues...)
	issues = append(issues, levelIssues...)
	issues = append(issues, workerIssues...)
	if len(issues) > 0 {
		return issues
	}

	if !sameIntSet(server.WorkerLevels, workers) {
		issues = append(issues, "matrix coverage worker_levels artifact does not match server.worker_levels")
	}

	declaredEndpoints := make(map[string]bool, len(endpoints))
	declaredWorkers := make(map[int]bool, len(workers))
	declaredLevels := make(map[string]bool, len(levels))
	for _, endpoint := range endpoints {
		declaredEndpoints[endpoint] = true
		for _, workerCount := range workers {
			declaredWorkers[workerCount] = true
			for _, level := range levels {
				declaredLevels[matrixLevelKey(level)] = true
			}
		}
	}

	seen := make(map[string]bool, len(runs))
	seenRuns := make(map[string]bool, len(runs))
	repeatsByCell := make(map[string]map[int]bool, len(runs))
	for _, run := range runs {
		if !declaredEndpoints[run.Endpoint] || !declaredWorkers[run.Workers] || !declaredLevels[matrixLevelKey(run.Level)] {
			issues = append(issues, fmt.Sprintf("matrix coverage includes undeclared endpoint %s workers=%d c%d/k%d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections))
			continue
		}
		runKey := matrixRunIdentityKey(run)
		if seenRuns[runKey] {
			issues = append(issues, fmt.Sprintf("duplicate matrix run endpoint %s workers=%d c%d/k%d repeat=%d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
			continue
		}
		seenRuns[runKey] = true
		coverageKey := matrixCoverageKey(run.Endpoint, run.Workers, run.Level)
		seen[coverageKey] = true
		if run.Repeat > 0 {
			if repeatsByCell[coverageKey] == nil {
				repeatsByCell[coverageKey] = make(map[int]bool)
			}
			repeatsByCell[coverageKey][run.Repeat] = true
		}
	}

	var expectedRepeats []int
	for _, endpoint := range endpoints {
		for _, workerCount := range workers {
			for _, level := range levels {
				key := matrixCoverageKey(endpoint, workerCount, level)
				if !seen[key] {
					issues = append(issues, fmt.Sprintf("matrix coverage missing endpoint %s workers=%d c%d/k%d", endpoint, workerCount, level.Concurrency, level.Connections))
					continue
				}
				label := matrixCoverageLabel(endpoint, workerCount, level)
				repeats := sortedRepeatSet(repeatsByCell[key])
				if len(repeats) == 0 {
					issues = append(issues, "matrix repeat coverage "+label+" is missing positive repeat evidence")
					continue
				}
				issues = append(issues, validateMatrixRepeatSequence(label, repeats)...)
				if expectedRepeats == nil {
					expectedRepeats = repeats
					continue
				}
				if !sameIntSequence(repeats, expectedRepeats) {
					issues = append(issues, fmt.Sprintf("matrix repeat coverage %s repeats %s, want %s", label, formatIntSequence(repeats), formatIntSequence(expectedRepeats)))
				}
			}
		}
	}
	return issues
}

func parseMatrixArtifactEndpoints(raw string) ([]string, []string) {
	var endpoints []string
	var issues []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		endpoint := strings.TrimSpace(part)
		if endpoint == "" {
			issues = append(issues, "matrix coverage endpoints contain an empty entry")
			continue
		}
		if _, ok := matrixEndpointSpecs()[endpoint]; !ok {
			issues = append(issues, "matrix coverage endpoint "+endpoint+" is unsupported")
			continue
		}
		if seen[endpoint] {
			issues = append(issues, "matrix coverage endpoint "+endpoint+" is duplicated")
			continue
		}
		seen[endpoint] = true
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, issues
}

func parseMatrixArtifactLevels(raw string) ([]MatrixLevel, []string) {
	var levels []MatrixLevel
	var issues []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		entry := strings.TrimSpace(part)
		if entry == "" {
			issues = append(issues, "matrix coverage levels contain an empty entry")
			continue
		}
		concurrencyRaw, connectionsRaw, ok := strings.Cut(entry, ":")
		if !ok {
			issues = append(issues, "matrix coverage level "+entry+" must be concurrency:connections")
			continue
		}
		concurrency, err1 := strconv.Atoi(strings.TrimSpace(concurrencyRaw))
		connections, err2 := strconv.Atoi(strings.TrimSpace(connectionsRaw))
		if err1 != nil || err2 != nil || concurrency <= 0 || connections <= 0 {
			issues = append(issues, "matrix coverage level "+entry+" must use positive integers")
			continue
		}
		level := MatrixLevel{Concurrency: concurrency, Connections: connections}
		key := matrixLevelKey(level)
		if seen[key] {
			issues = append(issues, "matrix coverage level "+entry+" is duplicated")
			continue
		}
		seen[key] = true
		levels = append(levels, level)
	}
	return levels, issues
}

func parseMatrixArtifactWorkers(raw string) ([]int, []string) {
	var workers []int
	var issues []string
	seen := map[int]bool{}
	for _, part := range strings.Split(raw, ",") {
		entry := strings.TrimSpace(part)
		if entry == "" {
			issues = append(issues, "matrix coverage worker_levels contain an empty entry")
			continue
		}
		value, err := strconv.Atoi(entry)
		if err != nil || value <= 0 {
			issues = append(issues, "matrix coverage worker level "+entry+" must be a positive integer")
			continue
		}
		if seen[value] {
			issues = append(issues, fmt.Sprintf("matrix coverage worker level %d is duplicated", value))
			continue
		}
		seen[value] = true
		workers = append(workers, value)
	}
	return workers, issues
}

func sameIntSet(left []int, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[int]bool, len(left))
	for _, value := range left {
		seen[value] = true
	}
	for _, value := range right {
		if !seen[value] {
			return false
		}
	}
	return true
}

func matrixCoverageKey(endpoint string, workers int, level MatrixLevel) string {
	return fmt.Sprintf("%s|%d|%d|%d", endpoint, workers, level.Concurrency, level.Connections)
}

func matrixCoverageLabel(endpoint string, workers int, level MatrixLevel) string {
	return fmt.Sprintf("endpoint %s workers=%d c%d/k%d", endpoint, workers, level.Concurrency, level.Connections)
}

func matrixRunIdentityKey(run MatrixRun) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat)
}

func matrixLevelKey(level MatrixLevel) string {
	return fmt.Sprintf("%d:%d", level.Concurrency, level.Connections)
}

func sortedRepeatSet(repeats map[int]bool) []int {
	values := make([]int, 0, len(repeats))
	for repeat := range repeats {
		values = append(values, repeat)
	}
	sort.Ints(values)
	return values
}

func validateMatrixRepeatSequence(label string, repeats []int) []string {
	var issues []string
	for index, repeat := range repeats {
		want := index + 1
		if repeat != want {
			issues = append(issues, fmt.Sprintf("matrix repeat coverage %s missing repeat %d before repeat %d", label, want, repeat))
			return issues
		}
	}
	return issues
}

func sameIntSequence(left []int, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func formatIntSequence(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
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
	issues = append(issues, validateMatrixResourceSpan("resource", resource.Start, resource.End)...)
	return issues
}

func validateMatrixResourceSnapshot(name string, snapshot MatrixResourceSnapshot) error {
	var issues []string
	if snapshot.RSSKB <= 0 || snapshot.FDCount <= 0 || snapshot.Threads <= 0 {
		issues = append(issues, fmt.Sprintf("%s RSS/FD/thread evidence is required", name))
	}
	if snapshot.PID <= 0 || !snapshot.ProcessAlive {
		issues = append(issues, fmt.Sprintf("%s process evidence is required", name))
	}
	if snapshot.TCPConnections < 0 || snapshot.CPUUserSeconds < 0 || snapshot.CPUSystemSeconds < 0 || snapshot.Goroutines < 0 {
		issues = append(issues, fmt.Sprintf("%s resource counters must be non-negative", name))
	}
	if strings.TrimSpace(snapshot.Timestamp) != "" {
		if _, err := time.Parse(time.RFC3339, snapshot.Timestamp); err != nil {
			issues = append(issues, fmt.Sprintf("%s timestamp is not RFC3339: %v", name, err))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
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
	issues = append(issues, validateMatrixEndpointIdentity("soak", soak.Endpoint, soak.Path, "")...)
	if soak.Level.Concurrency <= 0 || soak.Level.Connections <= 0 {
		issues = append(issues, "soak level concurrency/connections must be positive")
	}
	if soak.DurationSeconds <= 0 {
		issues = append(issues, "soak duration evidence is required")
	}
	if soak.Requests <= 0 || soak.Successes < 0 || soak.Failures < 0 || soak.Successes+soak.Failures != soak.Requests {
		issues = append(issues, "soak request counters are inconsistent")
	}
	if soak.Successes <= 0 || soak.Failures != 0 || soak.RPS <= 0 {
		issues = append(issues, "soak evidence did not pass")
	}
	if soak.AvgLatencyMS < 0 || soak.P99LatencyMS < 0 || soak.P999LatencyMS < 0 || soak.MaxLatencyMS < 0 || soak.FirstHalfAvgMS < 0 || soak.SecondHalfAvgMS < 0 {
		issues = append(issues, "soak has invalid timing metrics")
	}
	issues = append(issues, validateTailLatencyPercentiles("soak "+soak.Path, soak.P99LatencyMS, soak.P999LatencyMS, soak.MaxLatencyMS)...)
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
	issues = append(issues, validateMatrixResourceSpan("soak.resource", soak.ResourceStart, soak.ResourceEnd)...)
	return issues
}

func validateMatrixResourceSpan(name string, start MatrixResourceSnapshot, end MatrixResourceSnapshot) []string {
	var issues []string
	startTime, startOK := parseMatrixResourceTimestamp(start)
	endTime, endOK := parseMatrixResourceTimestamp(end)
	if startOK && endOK && !endTime.After(startTime) {
		issues = append(issues, name+" timestamps are not increasing")
	}
	if end.CPUUserSeconds < start.CPUUserSeconds || end.CPUSystemSeconds < start.CPUSystemSeconds {
		issues = append(issues, name+" resource CPU counters regressed")
	}
	return issues
}

func parseMatrixResourceTimestamp(snapshot MatrixResourceSnapshot) (time.Time, bool) {
	if strings.TrimSpace(snapshot.Timestamp) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, snapshot.Timestamp)
	return parsed, err == nil
}

func validateTailLatencyPercentiles(label string, p99, p999, max float64) []string {
	ordered := []struct {
		name  string
		value float64
	}{
		{name: "p99", value: p99},
		{name: "p999", value: p999},
		{name: "max", value: max},
	}
	var issues []string
	for i := 1; i < len(ordered); i++ {
		if ordered[i].value < ordered[i-1].value {
			issues = append(issues, fmt.Sprintf("%s latency percentiles are not monotonic: %s=%g < %s=%g", label, ordered[i].name, ordered[i].value, ordered[i-1].name, ordered[i-1].value))
			break
		}
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
	if warmup && run.Repeat != 0 {
		issues = append(issues, label+" repeat must be 0")
	}
	if !warmup && run.Repeat <= 0 {
		issues = append(issues, label+" repeat must be positive")
	}
	issues = append(issues, validateMatrixEndpointIdentity(label, run.Endpoint, run.Path, run.Kind)...)
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
	issues = append(issues, validateMatrixRunRPS(label, run)...)
	issues = append(issues, validateLatencyPercentiles(label, run.P50LatencyMS, run.P90LatencyMS, run.P95LatencyMS, run.P99LatencyMS, run.P999LatencyMS, run.MaxLatencyMS)...)
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

func validateMatrixRunRPS(label string, run MatrixRun) []string {
	if run.ElapsedSeconds <= 0 || run.Successes <= 0 || run.RPS <= 0 {
		return nil
	}
	expected := float64(run.Successes) / run.ElapsedSeconds
	tolerance := math.Max(0.001, expected*0.0001)
	if math.Abs(run.RPS-expected) > tolerance {
		return []string{fmt.Sprintf("%s rps evidence = %g, want %g from successes/elapsed_seconds", label, run.RPS, expected)}
	}
	return nil
}

func validateMatrixEndpointIdentity(label string, endpoint string, path string, kind string) []string {
	spec, ok := matrixEndpointSpecs()[strings.TrimSpace(endpoint)]
	if !ok {
		return []string{label + " matrix endpoint identity is unsupported"}
	}
	var issues []string
	if path != spec.path {
		issues = append(issues, fmt.Sprintf("%s matrix endpoint identity path = %q, want %q", label, path, spec.path))
	}
	if strings.TrimSpace(kind) != "" && kind != spec.kind {
		issues = append(issues, fmt.Sprintf("%s matrix endpoint identity kind = %q, want %q", label, kind, spec.kind))
	}
	return issues
}

func matrixEndpointSpecs() map[string]struct {
	path string
	kind string
} {
	return map[string]struct {
		path string
		kind string
	}{
		"db":       {path: "/db", kind: "single-query"},
		"queries":  {path: "/queries?queries=2", kind: "multiple-queries"},
		"updates":  {path: "/updates?queries=2", kind: "updates"},
		"fortunes": {path: "/fortunes", kind: "fortunes"},
	}
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
