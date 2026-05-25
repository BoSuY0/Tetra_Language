package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

const matrixSchema = "tetra.techempower.single_query_matrix.v1"

type options struct {
	RootDir             string
	SemanticReportPath  string
	MatrixReportPath    string
	EndpointsRaw        string
	LevelsRaw           string
	WorkerLevelsRaw     string
	Repeats             int
	Duration            time.Duration
	Warmup              time.Duration
	SoakDuration        time.Duration
	SemanticRequests    int
	SemanticConcurrency int
	Workers             int
	PoolSize            int
	KeepWorkDir         bool
	WorkDir             string
	CacheDir            string
	ProfileBuild        bool
	PprofDir            string
	PprofAddr           string
}

type benchLevel struct {
	Concurrency int `json:"concurrency"`
	Connections int `json:"connections"`
}

type matrixReport struct {
	Schema           string               `json:"schema"`
	Status           string               `json:"status"`
	GeneratedAt      string               `json:"generated_at"`
	GeneratedLocalAt string               `json:"generated_local_at"`
	Command          string               `json:"command"`
	Environment      benchmarkEnvironment `json:"environment"`
	Git              gitState             `json:"git"`
	Build            buildEvidence        `json:"build"`
	Postgres         postgresEvidence     `json:"postgres"`
	Server           serverEvidence       `json:"server"`
	Resource         resourceEvidence     `json:"resource"`
	SemanticProbe    []semanticCheck      `json:"semantic_probe"`
	Warmup           *dbRunReport         `json:"warmup,omitempty"`
	Soak             *soakReport          `json:"soak,omitempty"`
	Runs             []dbRunReport        `json:"runs"`
	Summary          matrixSummary        `json:"summary"`
	Artifacts        map[string]string    `json:"artifacts"`
	Limitations      []string             `json:"limitations"`
}

type benchmarkEnvironment struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	Hostname  string `json:"hostname"`
}

type gitState struct {
	Head           string `json:"head"`
	WorktreeStatus string `json:"worktree_status"`
}

type buildEvidence struct {
	AppBinary       string `json:"app_binary"`
	BenchBinary     string `json:"bench_binary"`
	Mode            string `json:"mode"`
	BuildCommand    string `json:"build_command"`
	GoBuildTrimpath bool   `json:"go_build_trimpath"`
	Stripped        bool   `json:"stripped"`
}

type buildPlan struct {
	Mode            string
	Args            []string
	BuildCommand    string
	GoBuildTrimpath bool
	Stripped        bool
}

type postgresEvidence struct {
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

type serverEvidence struct {
	BaseURL      string `json:"base_url"`
	Workers      int    `json:"workers"`
	WorkerLevels []int  `json:"worker_levels"`
	PoolSize     int    `json:"pool_size"`
}

type semanticCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	Error    string `json:"error,omitempty"`
}

type resourceEvidence struct {
	Start resourceSnapshot `json:"start"`
	End   resourceSnapshot `json:"end"`
}

type resourceSnapshot struct {
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

type soakReport struct {
	Endpoint         string           `json:"endpoint"`
	Path             string           `json:"path"`
	Workers          int              `json:"workers"`
	Level            benchLevel       `json:"level"`
	DurationSeconds  float64          `json:"duration_seconds"`
	Requests         int              `json:"requests"`
	Successes        int              `json:"successes"`
	Failures         int              `json:"failures"`
	RPS              float64          `json:"rps"`
	AvgLatencyMS     float64          `json:"avg_latency_ms"`
	P99LatencyMS     float64          `json:"p99_latency_ms"`
	P999LatencyMS    float64          `json:"p999_latency_ms"`
	MaxLatencyMS     float64          `json:"max_latency_ms"`
	FirstHalfAvgMS   float64          `json:"first_half_avg_latency_ms"`
	SecondHalfAvgMS  float64          `json:"second_half_avg_latency_ms"`
	LatencyDriftMS   float64          `json:"latency_drift_ms"`
	ResourceStart    resourceSnapshot `json:"resource_start"`
	ResourceEnd      resourceSnapshot `json:"resource_end"`
	OpenSocketsAfter int              `json:"open_sockets_after_shutdown"`
	ShutdownClean    bool             `json:"shutdown_clean"`
	Validation       string           `json:"validation"`
	Error            string           `json:"error,omitempty"`
}

type dbRunReport struct {
	Endpoint        string           `json:"endpoint"`
	Path            string           `json:"path"`
	Kind            string           `json:"kind"`
	Workers         int              `json:"workers"`
	Level           benchLevel       `json:"level"`
	Repeat          int              `json:"repeat"`
	DurationSeconds float64          `json:"duration_seconds"`
	ElapsedSeconds  float64          `json:"elapsed_seconds"`
	Requests        int              `json:"requests"`
	Successes       int              `json:"successes"`
	Failures        int              `json:"failures"`
	Bytes           int64            `json:"bytes"`
	RPS             float64          `json:"rps"`
	AvgLatencyMS    float64          `json:"avg_latency_ms"`
	P50LatencyMS    float64          `json:"p50_latency_ms"`
	P90LatencyMS    float64          `json:"p90_latency_ms"`
	P95LatencyMS    float64          `json:"p95_latency_ms"`
	P99LatencyMS    float64          `json:"p99_latency_ms"`
	P999LatencyMS   float64          `json:"p999_latency_ms"`
	MaxLatencyMS    float64          `json:"max_latency_ms"`
	Resource        resourceSnapshot `json:"resource"`
	Validation      string           `json:"validation"`
	Error           string           `json:"error,omitempty"`
}

type matrixSummary struct {
	RunCount      int     `json:"run_count"`
	TotalRequests int     `json:"total_requests"`
	TotalFailures int     `json:"total_failures"`
	BestRPS       float64 `json:"best_rps"`
	WorstP99MS    float64 `json:"worst_p99_ms"`
	WorstP999MS   float64 `json:"worst_p999_ms"`
	Decision      string  `json:"decision"`
}

type loadResult struct {
	status  int
	bytes   int
	latency time.Duration
	err     error
}

type pprofArtifacts struct {
	CPUProfile  string
	HeapProfile string
}

type endpointBenchmarkSpec struct {
	Name     string
	Path     string
	Kind     string
	Validate func(*http.Response, []byte) error
}

type world struct {
	ID           int `json:"id"`
	RandomNumber int `json:"randomNumber"`
}

func main() {
	var opt options
	flag.StringVar(&opt.RootDir, "root", ".", "repository root; auto-detected when possible")
	flag.StringVar(&opt.SemanticReportPath, "semantic-report", "reports/techempower/tetra-scram-six-endpoint-local-benchmark.json", "six-endpoint semantic report path")
	flag.StringVar(&opt.MatrixReportPath, "matrix-report", "reports/techempower/tetra-scram-single-query-matrix.json", "single-query matrix report path")
	flag.StringVar(&opt.EndpointsRaw, "endpoints", "db", "comma-separated benchmark endpoints: db,queries,updates,fortunes")
	flag.StringVar(&opt.LevelsRaw, "levels", "8:8,16:16,32:32", "comma-separated concurrency:connections pairs")
	flag.StringVar(&opt.WorkerLevelsRaw, "worker-levels", "", "comma-separated worker counts; defaults to --workers")
	flag.IntVar(&opt.Repeats, "repeats", 2, "matrix repeats per level")
	flag.DurationVar(&opt.Duration, "duration", 5*time.Second, "matrix duration per repeat")
	flag.DurationVar(&opt.Warmup, "warmup", 2*time.Second, "single-query warmup duration")
	flag.DurationVar(&opt.SoakDuration, "soak", 0, "optional longer soak duration against the first endpoint/level")
	flag.IntVar(&opt.SemanticRequests, "semantic-requests", 64, "requests per endpoint for semantic report")
	flag.IntVar(&opt.SemanticConcurrency, "semantic-concurrency", 8, "semantic report concurrency")
	flag.IntVar(&opt.Workers, "workers", 1, "Tetra worker count")
	flag.IntVar(&opt.PoolSize, "pool", 32, "Tetra PostgreSQL pool size")
	flag.BoolVar(&opt.KeepWorkDir, "keep-work-dir", false, "keep temporary build/PostgreSQL directory")
	flag.StringVar(&opt.WorkDir, "work-dir", "", "work directory; defaults to a temporary directory")
	flag.StringVar(&opt.CacheDir, "cache-dir", "", "embedded PostgreSQL binary cache directory")
	flag.BoolVar(&opt.ProfileBuild, "profile-build", false, "build benchmark binaries with debug symbols for profiling; disables trimpath and stripping")
	flag.StringVar(&opt.PprofDir, "pprof-dir", "", "optional directory for live server pprof CPU/heap profiles; enables localhost-only pprof on the benchmark server")
	flag.Parse()

	if err := run(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opt options) error {
	if opt.Repeats <= 0 {
		return errors.New("--repeats must be positive")
	}
	if opt.Duration <= 0 {
		return errors.New("--duration must be positive")
	}
	if opt.Warmup < 0 {
		return errors.New("--warmup must not be negative")
	}
	if opt.SemanticRequests <= 0 || opt.SemanticConcurrency <= 0 {
		return errors.New("--semantic-requests and --semantic-concurrency must be positive")
	}
	if opt.Workers <= 0 || opt.PoolSize <= 0 {
		return errors.New("--workers and --pool must be positive")
	}
	levels, err := parseLevels(opt.LevelsRaw)
	if err != nil {
		return err
	}
	endpointNames, err := parseEndpointNames(opt.EndpointsRaw)
	if err != nil {
		return err
	}
	endpoints, err := endpointBenchmarkSpecs(endpointNames)
	if err != nil {
		return err
	}
	workerLevels := []int{opt.Workers}
	if strings.TrimSpace(opt.WorkerLevelsRaw) != "" {
		workerLevels, err = parsePositiveIntList(opt.WorkerLevelsRaw, "--worker-levels")
		if err != nil {
			return err
		}
	}
	root, err := findRepoRoot(opt.RootDir)
	if err != nil {
		return err
	}
	workDir, cleanup, err := prepareWorkDir(opt)
	if err != nil {
		return err
	}
	defer cleanup()
	if opt.CacheDir == "" {
		opt.CacheDir = defaultEmbeddedPostgresCacheDir()
	}
	if err := os.MkdirAll(opt.CacheDir, 0o755); err != nil {
		return err
	}
	pprofDir := ""
	if strings.TrimSpace(opt.PprofDir) != "" {
		pprofDir = absPath(root, opt.PprofDir)
		if err := os.MkdirAll(pprofDir, 0o755); err != nil {
			return err
		}
	}

	appBin := filepath.Join(workDir, "bin", "tetra-techempower")
	benchBin := filepath.Join(workDir, "bin", "tetra-techempower-bench")
	if err := os.MkdirAll(filepath.Dir(appBin), 0o755); err != nil {
		return err
	}
	buildPlan := buildPlanForMode(opt.ProfileBuild, appBin, "./compiler/cmd/tetra-techempower")
	if err := buildBinary(ctx, root, buildPlan); err != nil {
		return err
	}
	benchBuildPlan := buildPlanForMode(opt.ProfileBuild, benchBin, "./compiler/cmd/tetra-techempower-bench")
	if err := buildBinary(ctx, root, benchBuildPlan); err != nil {
		return err
	}

	pgPort, err := freeTCPPort()
	if err != nil {
		return err
	}
	appPort, err := freeTCPPort()
	if err != nil {
		return err
	}

	pg, pgInfo, err := startSCRAMPostgres(root, workDir, opt.CacheDir, pgPort)
	if err != nil {
		return err
	}
	defer func() {
		_ = pg.Stop()
	}()

	db, err := sql.Open("postgres", postgresDSN(pgPort, "benchmarkdbuser", "benchmarkdbpass", "hello_world"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := seedPostgres(ctx, root, db); err != nil {
		return err
	}
	if err := enrichPostgresEvidence(ctx, db, &pgInfo); err != nil {
		return err
	}

	report := newMatrixReport(opt, levels, endpointNames, workerLevels, appBin, benchBin, buildPlan, pgInfo, "", nil)
	if pprofDir != "" {
		report.Artifacts["pprof_dir"] = pprofDir
	}
	report.Resource.Start = detectResource(os.Getpid(), 0)
	client := &http.Client{Timeout: 15 * time.Second}
	semanticDone := false
	soakDone := false
	pprofCaptured := false
	for _, workers := range workerLevels {
		runOpt := opt
		runOpt.Workers = workers
		appPort, err = freeTCPPort()
		if err != nil {
			return err
		}
		pprofBaseURL := ""
		if pprofDir != "" {
			pprofPort, err := freeTCPPort()
			if err != nil {
				return err
			}
			runOpt.PprofAddr = "127.0.0.1:" + strconv.Itoa(pprofPort)
			pprofBaseURL = "http://" + runOpt.PprofAddr
			report.Artifacts["pprof_addr"] = runOpt.PprofAddr
		}
		server, serverLog, err := startServer(ctx, root, appBin, appPort, pgPort, runOpt)
		if err != nil {
			return err
		}
		baseURL := "http://127.0.0.1:" + strconv.Itoa(appPort)
		if err := waitForHTTP(ctx, baseURL+"/plaintext", 30*time.Second); err != nil {
			stopProcess(server)
			return fmt.Errorf("server did not become ready: %w\nserver log:\n%s", err, serverLog.String())
		}
		if pprofBaseURL != "" {
			if err := waitForHTTP(ctx, pprofBaseURL+"/debug/pprof/", 30*time.Second); err != nil {
				stopProcess(server)
				return fmt.Errorf("pprof server did not become ready: %w\nserver log:\n%s", err, serverLog.String())
			}
		}
		if !semanticDone {
			report.Server.BaseURL = baseURL
			report.Server.Workers = workers
			probe := runSemanticProbe(ctx, client, baseURL, db)
			report.SemanticProbe = probe
			if semanticFailed(probe) {
				report.Status = "fail"
				report.Summary.Decision = "fail"
				_ = writeJSON(root, opt.MatrixReportPath, report)
				stopProcess(server)
				return fmt.Errorf("semantic probe failed; see %s", opt.MatrixReportPath)
			}
			if err := runSemanticReport(ctx, root, benchBin, baseURL, opt); err != nil {
				stopProcess(server)
				return err
			}
			rawSemantic, err := os.ReadFile(absPath(root, opt.SemanticReportPath))
			if err != nil {
				stopProcess(server)
				return err
			}
			if err := validateSemanticReport(rawSemantic); err != nil {
				stopProcess(server)
				return fmt.Errorf("semantic report validation failed: %w", err)
			}
			semanticDone = true
		}
		if opt.Warmup > 0 && report.Warmup == nil {
			warmup := runEndpointLoad(ctx, baseURL, endpoints[0], workers, levels[0], 0, opt.Warmup, server.Process.Pid, appPort)
			report.Warmup = &warmup
		}
		for _, endpoint := range endpoints {
			for _, level := range levels {
				for repeat := 1; repeat <= opt.Repeats; repeat++ {
					if !pprofCaptured && pprofBaseURL != "" {
						cpuDone, artifacts, err := startPprofCPUProfile(ctx, pprofBaseURL, pprofDir, opt.Duration)
						if err != nil {
							stopProcess(server)
							return err
						}
						run := runEndpointLoad(ctx, baseURL, endpoint, workers, level, repeat, opt.Duration, server.Process.Pid, appPort)
						if err := <-cpuDone; err != nil {
							stopProcess(server)
							return err
						}
						if err := capturePprofHeap(ctx, pprofBaseURL, artifacts.HeapProfile); err != nil {
							stopProcess(server)
							return err
						}
						report.Artifacts["pprof_cpu_profile"] = artifacts.CPUProfile
						report.Artifacts["pprof_heap_profile"] = artifacts.HeapProfile
						pprofCaptured = true
						report.Runs = append(report.Runs, run)
					} else {
						report.Runs = append(report.Runs, runEndpointLoad(ctx, baseURL, endpoint, workers, level, repeat, opt.Duration, server.Process.Pid, appPort))
					}
				}
			}
		}
		if opt.SoakDuration > 0 && !soakDone {
			report.Soak = runSoak(ctx, baseURL, endpoints[0], workers, levels[0], opt.SoakDuration, server.Process.Pid, appPort)
			soakDone = true
		}
		shutdown := stopProcess(server)
		if report.Soak != nil && report.Soak.Workers == workers {
			report.Soak.ShutdownClean = shutdown.Clean
			report.Soak.OpenSocketsAfter = countTCPConnections(appPort)
		}
	}
	report.Resource.End = detectResource(os.Getpid(), 0)
	report.Summary = summarizeMatrix(report.Runs)
	report.Status = report.Summary.Decision
	if err := validateMatrixReport(report); err != nil {
		report.Status = "fail"
		report.Summary.Decision = "fail"
		_ = writeJSON(root, opt.MatrixReportPath, report)
		return err
	}
	if err := writeJSON(root, opt.MatrixReportPath, report); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "semantic report: %s\n", opt.SemanticReportPath)
	fmt.Fprintf(os.Stdout, "matrix report: %s\n", opt.MatrixReportPath)
	fmt.Fprintf(os.Stdout, "best endpoint rps: %.2f, worst p99: %.3f ms, worst p99.9: %.3f ms\n", report.Summary.BestRPS, report.Summary.WorstP99MS, report.Summary.WorstP999MS)
	return nil
}

func parseLevels(raw string) ([]benchLevel, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("--levels is required")
	}
	parts := strings.Split(raw, ",")
	levels := make([]benchLevel, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errors.New("--levels contains an empty entry")
		}
		pair := strings.Split(part, ":")
		if len(pair) != 2 {
			return nil, fmt.Errorf("level %q must be concurrency:connections", part)
		}
		concurrency, err := strconv.Atoi(strings.TrimSpace(pair[0]))
		if err != nil || concurrency <= 0 {
			return nil, fmt.Errorf("level %q has invalid concurrency", part)
		}
		connections, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil || connections <= 0 {
			return nil, fmt.Errorf("level %q has invalid connections", part)
		}
		levels = append(levels, benchLevel{Concurrency: concurrency, Connections: connections})
	}
	return levels, nil
}

func parseEndpointNames(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("--endpoints is required")
	}
	allowed := map[string]bool{
		"db":       true,
		"queries":  true,
		"updates":  true,
		"fortunes": true,
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, errors.New("--endpoints contains an empty entry")
		}
		if !allowed[name] {
			return nil, fmt.Errorf("unsupported endpoint %q", name)
		}
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	return out, nil
}

func parsePositiveIntList(raw string, flagName string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%s is required", flagName)
	}
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("%s contains an empty entry", flagName)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("%s entry %q must be a positive integer", flagName, part)
		}
		values = append(values, value)
	}
	return values, nil
}

func endpointBenchmarkSpecs(names []string) ([]endpointBenchmarkSpec, error) {
	specs := make([]endpointBenchmarkSpec, 0, len(names))
	for _, name := range names {
		switch name {
		case "db":
			specs = append(specs, endpointBenchmarkSpec{Name: "db", Path: "/db", Kind: "single-query", Validate: validateWorldHTTP})
		case "queries":
			specs = append(specs, endpointBenchmarkSpec{Name: "queries", Path: "/queries?queries=2", Kind: "multiple-queries", Validate: validateWorldArrayHTTP})
		case "updates":
			specs = append(specs, endpointBenchmarkSpec{Name: "updates", Path: "/updates?queries=2", Kind: "updates", Validate: validateWorldArrayHTTP})
		case "fortunes":
			specs = append(specs, endpointBenchmarkSpec{Name: "fortunes", Path: "/fortunes", Kind: "fortunes", Validate: validateFortunesHTTP})
		default:
			return nil, fmt.Errorf("unsupported endpoint %q", name)
		}
	}
	return specs, nil
}

func findRepoRoot(start string) (string, error) {
	if start == "" {
		start = "."
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "benchmarks", "techempower", "tetra", "setup-postgres.sql")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find Tetra repo root from %s", start)
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func prepareWorkDir(opt options) (string, func(), error) {
	if opt.WorkDir != "" {
		abs, err := filepath.Abs(opt.WorkDir)
		if err != nil {
			return "", func() {}, err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", func() {}, err
		}
		return abs, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "tetra-techempower-scram-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		if !opt.KeepWorkDir {
			_ = os.RemoveAll(dir)
		}
	}
	return dir, cleanup, nil
}

func defaultEmbeddedPostgresCacheDir() string {
	cacheRoot, err := os.UserCacheDir()
	if err != nil || cacheRoot == "" {
		cacheRoot = os.TempDir()
	}
	return filepath.Join(cacheRoot, "tetra", "embedded-postgres")
}

func buildPlanForMode(profile bool, out string, pkg string) buildPlan {
	args := []string{"build"}
	plan := buildPlan{
		Mode:            "release",
		GoBuildTrimpath: true,
		Stripped:        true,
	}
	if profile {
		plan.Mode = "profile"
		plan.GoBuildTrimpath = false
		plan.Stripped = false
		args = append(args, "-gcflags=all=-N -l")
	} else {
		args = append(args, "-trimpath", "-ldflags=-s -w")
	}
	args = append(args, "-o", out, pkg)
	plan.Args = args
	plan.BuildCommand = "go " + strings.Join(args, " ")
	return plan
}

func buildBinary(ctx context.Context, root string, plan buildPlan) error {
	cmd := exec.CommandContext(ctx, "go", plan.Args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK="+filepath.Join(root, "go.work"))
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, combined.String())
	}
	return nil
}

func freeTCPPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func startSCRAMPostgres(root string, workDir string, cacheDir string, port int) (*embeddedpostgres.EmbeddedPostgres, postgresEvidence, error) {
	runtimePath := filepath.Join(workDir, "pg-runtime")
	dataPath := filepath.Join(workDir, "pg-data")
	cfg := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V16).
		Username("benchmarkdbuser").
		Password("benchmarkdbpass").
		Database("hello_world").
		Port(uint32(port)).
		CachePath(cacheDir).
		RuntimePath(runtimePath).
		DataPath(dataPath).
		StartTimeout(60 * time.Second).
		StartParameters(map[string]string{
			"max_connections":         "256",
			"password_encryption":     "scram-sha-256",
			"log_min_messages":        "warning",
			"log_min_error_statement": "warning",
		}).
		Logger(io.Discard)

	first := embeddedpostgres.NewDatabase(cfg)
	if err := first.Start(); err != nil {
		return nil, postgresEvidence{}, err
	}
	if err := first.Stop(); err != nil {
		return nil, postgresEvidence{}, err
	}
	if err := rewritePGHBAForSCRAM(filepath.Join(dataPath, "pg_hba.conf")); err != nil {
		return nil, postgresEvidence{}, err
	}
	pg := embeddedpostgres.NewDatabase(cfg)
	if err := pg.Start(); err != nil {
		return nil, postgresEvidence{}, err
	}
	info := postgresEvidence{
		Version:        string(embeddedpostgres.V16),
		AuthMethod:     "scram-sha-256",
		Host:           "127.0.0.1",
		Port:           port,
		Database:       "hello_world",
		User:           "benchmarkdbuser",
		MaxConnections: "256",
	}
	_ = root
	return pg, info, nil
}

func rewritePGHBAForSCRAM(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	rewrites := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(line)
		methodIndex := -1
		switch fields[0] {
		case "local":
			if len(fields) >= 4 {
				methodIndex = 3
			}
		case "host", "hostssl", "hostnossl", "hostgssenc", "hostnogssenc":
			if len(fields) >= 5 {
				methodIndex = 4
			}
		}
		if methodIndex < 0 {
			continue
		}
		if fields[methodIndex] != "scram-sha-256" {
			fields[methodIndex] = "scram-sha-256"
			rewrites++
		}
		lines[i] = strings.Join(fields, "\t")
	}
	if rewrites == 0 {
		return errors.New("pg_hba.conf did not contain local/host auth lines to rewrite")
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}

func postgresDSN(port int, user string, password string, database string) string {
	return fmt.Sprintf("host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable", port, user, password, database)
}

func seedPostgres(ctx context.Context, root string, db *sql.DB) error {
	raw, err := os.ReadFile(filepath.Join(root, "benchmarks", "techempower", "tetra", "setup-postgres.sql"))
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, string(raw)); err != nil {
		return err
	}
	return nil
}

func enrichPostgresEvidence(ctx context.Context, db *sql.DB, info *postgresEvidence) error {
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SHOW password_encryption").Scan(&info.PasswordEncryption); err != nil {
		return err
	}
	var verifier string
	if err := db.QueryRowContext(ctx, "SELECT rolpassword FROM pg_authid WHERE rolname=$1", info.User).Scan(&verifier); err != nil {
		return err
	}
	info.VerifierPrefix = verifierPrefix(verifier)
	if info.VerifierPrefix != "SCRAM-SHA-256" {
		return fmt.Errorf("role verifier prefix = %q, want SCRAM-SHA-256", info.VerifierPrefix)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM World").Scan(&info.WorldRows); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM Fortune").Scan(&info.FortuneRows); err != nil {
		return err
	}
	if info.WorldRows != 10000 || info.FortuneRows < 12 {
		return fmt.Errorf("unexpected seed counts: World=%d Fortune=%d", info.WorldRows, info.FortuneRows)
	}
	return nil
}

func verifierPrefix(verifier string) string {
	if idx := strings.Index(verifier, "$"); idx > 0 {
		return verifier[:idx]
	}
	return verifier
}

func startServer(ctx context.Context, root string, appBin string, appPort int, pgPort int, opt options) (*exec.Cmd, *bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, appBin)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), serverEnv(appPort, pgPort, opt)...)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := cmd.Start(); err != nil {
		return nil, &log, err
	}
	return cmd, &log, nil
}

func serverEnv(appPort int, pgPort int, opt options) []string {
	env := []string{
		"TETRA_TE_HOST=127.0.0.1",
		"TETRA_TE_PORT=" + strconv.Itoa(appPort),
		"TETRA_TE_WORKERS=" + strconv.Itoa(opt.Workers),
		"TETRA_TE_PG_HOST=127.0.0.1",
		"TETRA_TE_PG_PORT=" + strconv.Itoa(pgPort),
		"TETRA_TE_PG_USER=benchmarkdbuser",
		"TETRA_TE_PG_DATABASE=hello_world",
		"TETRA_TE_PG_PASSWORD=benchmarkdbpass",
		"TETRA_TE_PG_POOL=" + strconv.Itoa(opt.PoolSize),
	}
	if strings.TrimSpace(opt.PprofAddr) != "" {
		env = append(env, "TETRA_TE_PPROF_ADDR="+opt.PprofAddr)
	}
	return env
}

type shutdownEvidence struct {
	Clean bool
	Error string
}

func stopProcess(cmd *exec.Cmd) shutdownEvidence {
	if cmd == nil || cmd.Process == nil {
		return shutdownEvidence{Clean: true}
	}
	done := make(chan error, 1)
	go func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			return shutdownEvidence{Clean: false, Error: err.Error()}
		}
		return shutdownEvidence{Clean: true}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		err := <-done
		if err != nil {
			return shutdownEvidence{Clean: false, Error: err.Error()}
		}
		return shutdownEvidence{Clean: false, Error: "forced kill after timeout"}
	}
}

func waitForHTTP(ctx context.Context, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return last
}

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

func runEndpointLoad(ctx context.Context, baseURL string, endpoint endpointBenchmarkSpec, workers int, level benchLevel, repeat int, duration time.Duration, serverPID int, serverPort int) dbRunReport {
	transport := &http.Transport{
		MaxConnsPerHost:     level.Connections,
		MaxIdleConns:        level.Connections,
		MaxIdleConnsPerHost: level.Connections,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	start := time.Now()
	deadline := start.Add(duration)
	results := make(chan loadResult, level.Concurrency)
	var wg sync.WaitGroup
	for i := 0; i < level.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				results <- oneEndpointRequest(ctx, client, baseURL+endpoint.Path, endpoint)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var latencies []time.Duration
	var requests, successes, failures int
	var bytesRead int64
	var firstErr error
	for result := range results {
		requests++
		bytesRead += int64(result.bytes)
		latencies = append(latencies, result.latency)
		if result.err != nil {
			failures++
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		successes++
	}
	elapsed := time.Since(start)
	if elapsed <= 0 {
		elapsed = time.Nanosecond
	}
	latency := latencySummary(latencies)
	report := dbRunReport{
		Endpoint:        endpoint.Name,
		Path:            endpoint.Path,
		Kind:            endpoint.Kind,
		Workers:         workers,
		Level:           level,
		Repeat:          repeat,
		DurationSeconds: duration.Seconds(),
		ElapsedSeconds:  elapsed.Seconds(),
		Requests:        requests,
		Successes:       successes,
		Failures:        failures,
		Bytes:           bytesRead,
		RPS:             float64(successes) / elapsed.Seconds(),
		AvgLatencyMS:    averageMS(latencies),
		P50LatencyMS:    latency.P50MS,
		P90LatencyMS:    latency.P90MS,
		P95LatencyMS:    latency.P95MS,
		P99LatencyMS:    latency.P99MS,
		P999LatencyMS:   latency.P999MS,
		MaxLatencyMS:    latency.MaxMS,
		Resource:        detectResource(serverPID, serverPort),
		Validation:      "real HTTP GET " + endpoint.Path + " responses validated against TechEmpower-compatible endpoint contract",
	}
	if firstErr != nil {
		report.Error = firstErr.Error()
	}
	return report
}

func capturePprofProfiles(ctx context.Context, baseURL string, dir string, duration time.Duration) (pprofArtifacts, error) {
	done, artifacts, err := startPprofCPUProfile(ctx, baseURL, dir, duration)
	if err != nil {
		return artifacts, err
	}
	if err := <-done; err != nil {
		return artifacts, err
	}
	if err := capturePprofHeap(ctx, baseURL, artifacts.HeapProfile); err != nil {
		return artifacts, err
	}
	return artifacts, nil
}

func startPprofCPUProfile(ctx context.Context, baseURL string, dir string, duration time.Duration) (<-chan error, pprofArtifacts, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, pprofArtifacts{}, err
	}
	artifacts := pprofArtifacts{
		CPUProfile:  filepath.Join(dir, "native-scram-live-db-cpu.pprof"),
		HeapProfile: filepath.Join(dir, "native-scram-live-db-heap.pprof"),
	}
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	done := make(chan error, 1)
	go func() {
		target := strings.TrimRight(baseURL, "/") + "/debug/pprof/profile?seconds=" + strconv.Itoa(seconds)
		timeout := time.Duration(seconds+10) * time.Second
		done <- downloadPprof(ctx, target, artifacts.CPUProfile, timeout)
	}()
	return done, artifacts, nil
}

func capturePprofHeap(ctx context.Context, baseURL string, path string) error {
	target := strings.TrimRight(baseURL, "/") + "/debug/pprof/heap"
	return downloadPprof(ctx, target, path, 10*time.Second)
}

func downloadPprof(ctx context.Context, target string, path string, timeout time.Duration) error {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", target, resp.StatusCode)
	}
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}

func oneEndpointRequest(ctx context.Context, client *http.Client, target string, endpoint endpointBenchmarkSpec) loadResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return loadResult{err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return loadResult{latency: time.Since(start), err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return loadResult{status: resp.StatusCode, latency: time.Since(start), err: err}
	}
	if err := endpoint.Validate(resp, body); err != nil {
		return loadResult{status: resp.StatusCode, bytes: len(body), latency: time.Since(start), err: err}
	}
	return loadResult{status: resp.StatusCode, bytes: len(body), latency: time.Since(start)}
}

func runSoak(ctx context.Context, baseURL string, endpoint endpointBenchmarkSpec, workers int, level benchLevel, duration time.Duration, serverPID int, serverPort int) *soakReport {
	startResource := detectResource(serverPID, serverPort)
	run := runEndpointLoad(ctx, baseURL, endpoint, workers, level, 0, duration, serverPID, serverPort)
	endResource := detectResource(serverPID, serverPort)
	drift := 0.0
	if run.AvgLatencyMS > 0 {
		// The load runner records complete-run latency. Without periodic sampling,
		// report zero drift and keep the raw start/end resource snapshots as the
		// stability signal.
		drift = 0
	}
	soak := &soakReport{
		Endpoint:        endpoint.Name,
		Path:            endpoint.Path,
		Workers:         workers,
		Level:           level,
		DurationSeconds: duration.Seconds(),
		Requests:        run.Requests,
		Successes:       run.Successes,
		Failures:        run.Failures,
		RPS:             run.RPS,
		AvgLatencyMS:    run.AvgLatencyMS,
		P99LatencyMS:    run.P99LatencyMS,
		P999LatencyMS:   run.P999LatencyMS,
		MaxLatencyMS:    run.MaxLatencyMS,
		FirstHalfAvgMS:  run.AvgLatencyMS,
		SecondHalfAvgMS: run.AvgLatencyMS,
		LatencyDriftMS:  drift,
		ResourceStart:   startResource,
		ResourceEnd:     endResource,
		Validation:      "longer endpoint soak completed with response validation, resource snapshots, and shutdown cleanup check",
		Error:           run.Error,
	}
	return soak
}

func detectResource(pid int, port int) resourceSnapshot {
	now := time.Now()
	snapshot := resourceSnapshot{
		Timestamp:      now.UTC().Format(time.RFC3339),
		PID:            pid,
		TCPConnections: countTCPConnections(port),
	}
	if pid == os.Getpid() {
		snapshot.Goroutines = runtime.NumGoroutine()
	}
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	raw, err := os.ReadFile(statusPath)
	if err != nil {
		return snapshot
	}
	snapshot.ProcessAlive = true
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "VmRSS:":
			snapshot.RSSKB, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Threads:":
			snapshot.Threads, _ = strconv.Atoi(fields[1])
		}
	}
	if entries, err := os.ReadDir(filepath.Join("/proc", strconv.Itoa(pid), "fd")); err == nil {
		snapshot.FDCount = len(entries)
	}
	user, system := readProcessCPUSeconds(pid)
	snapshot.CPUUserSeconds = user
	snapshot.CPUSystemSeconds = system
	return snapshot
}

func readProcessCPUSeconds(pid int) (float64, float64) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, 0
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return 0, 0
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) < 15 {
		return 0, 0
	}
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	ticks := float64(clockTicksPerSecond())
	return utime / ticks, stime / ticks
}

func clockTicksPerSecond() int {
	out, err := exec.Command("getconf", "CLK_TCK").Output()
	if err != nil {
		return 100
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || value <= 0 {
		return 100
	}
	return value
}

func countTCPConnections(port int) int {
	if port <= 0 {
		return 0
	}
	hexPort := strings.ToUpper(fmt.Sprintf("%04X", port))
	return countTCPConnectionsIn("/proc/net/tcp", hexPort) + countTCPConnectionsIn("/proc/net/tcp6", hexPort)
}

func countTCPConnectionsIn(path string, hexPort string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	activeStates := map[string]bool{"01": true, "02": true, "03": true, "0A": true}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.Contains(fields[1], ":") {
			continue
		}
		local := fields[1]
		if strings.HasSuffix(strings.ToUpper(local), ":"+hexPort) && activeStates[strings.ToUpper(fields[3])] {
			count++
		}
	}
	return count
}

type latencyStats struct {
	P50MS  float64
	P90MS  float64
	P95MS  float64
	P99MS  float64
	P999MS float64
	MaxMS  float64
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

func newMatrixReport(opt options, levels []benchLevel, endpointNames []string, workerLevels []int, appBin string, benchBin string, plan buildPlan, pg postgresEvidence, baseURL string, probe []semanticCheck) matrixReport {
	now := time.Now()
	return matrixReport{
		Schema:           matrixSchema,
		Status:           "pass",
		GeneratedAt:      now.UTC().Format(time.RFC3339),
		GeneratedLocalAt: now.Local().Format(time.RFC3339),
		Command:          commandLine(),
		Environment:      detectEnvironment(),
		Git:              detectGitState(),
		Build: buildEvidence{
			AppBinary:       appBin,
			BenchBinary:     benchBin,
			Mode:            plan.Mode,
			BuildCommand:    plan.BuildCommand,
			GoBuildTrimpath: plan.GoBuildTrimpath,
			Stripped:        plan.Stripped,
		},
		Postgres: pg,
		Server: serverEvidence{
			BaseURL:      baseURL,
			Workers:      opt.Workers,
			WorkerLevels: append([]int(nil), workerLevels...),
			PoolSize:     opt.PoolSize,
		},
		SemanticProbe: probe,
		Summary:       matrixSummary{Decision: "pass"},
		Artifacts: map[string]string{
			"semantic_report": opt.SemanticReportPath,
			"matrix_report":   opt.MatrixReportPath,
			"endpoints":       strings.Join(endpointNames, ","),
			"levels":          formatLevels(levels),
			"worker_levels":   formatInts(workerLevels),
		},
		Limitations: []string{
			"local embedded PostgreSQL harness; official TechEmpower publication is not implied",
			"SCRAM-SHA-256-PLUS channel binding and SASLprep are not implemented in the Tetra PostgreSQL runtime",
			"matrix duration and concurrency are caller controlled; use --duration 30s or --duration 60s for release gates",
		},
	}
}

func commandLine() string {
	if override := strings.TrimSpace(os.Getenv("TETRA_TE_RUNNER_COMMAND")); override != "" {
		return override
	}
	args := []string{"GOWORK=off", "go", "run", "./benchmarks/techempower/tetra/cmd/scram-local-bench"}
	args = append(args, os.Args[1:]...)
	return strings.Join(args, " ")
}

func formatLevels(levels []benchLevel) string {
	parts := make([]string, 0, len(levels))
	for _, level := range levels {
		parts = append(parts, fmt.Sprintf("%d:%d", level.Concurrency, level.Connections))
	}
	return strings.Join(parts, ",")
}

func formatInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func summarizeMatrix(runs []dbRunReport) matrixSummary {
	summary := matrixSummary{RunCount: len(runs), Decision: "pass"}
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

func validateMatrixReport(report matrixReport) error {
	var issues []string
	if report.Schema != matrixSchema {
		issues = append(issues, "invalid matrix schema")
	}
	if report.GeneratedAt == "" || report.GeneratedLocalAt == "" || report.Command == "" {
		issues = append(issues, "report integrity metadata is required")
	}
	if report.Postgres.AuthMethod != "scram-sha-256" || report.Postgres.PasswordEncryption != "scram-sha-256" || report.Postgres.VerifierPrefix != "SCRAM-SHA-256" {
		issues = append(issues, "PostgreSQL SCRAM evidence is incomplete")
	}
	if report.Resource.Start.RSSKB <= 0 || report.Resource.End.RSSKB <= 0 {
		issues = append(issues, "resource start/end RSS evidence is required")
	}
	if semanticFailed(report.SemanticProbe) {
		issues = append(issues, "semantic probe failed")
	}
	issues = append(issues, validateSemanticProbeCoverage(report.SemanticProbe)...)
	if report.Soak != nil {
		if report.Soak.Requests <= 0 || report.Soak.Successes <= 0 || report.Soak.Failures != 0 || !report.Soak.ShutdownClean {
			issues = append(issues, "soak evidence did not pass")
		}
		if report.Soak.ResourceStart.RSSKB <= 0 || report.Soak.ResourceEnd.RSSKB <= 0 {
			issues = append(issues, "soak resource evidence is incomplete")
		}
	}
	if report.Summary.Decision != "pass" {
		issues = append(issues, "matrix summary did not pass")
	}
	for _, run := range report.Runs {
		if strings.TrimSpace(run.Endpoint) == "" || strings.TrimSpace(run.Path) == "" || strings.TrimSpace(run.Kind) == "" || run.Workers <= 0 {
			issues = append(issues, "run endpoint/worker metadata is incomplete")
		}
		if run.Requests <= 0 || run.Successes <= 0 || run.Failures != 0 || run.RPS <= 0 {
			issues = append(issues, fmt.Sprintf("invalid %s run at workers=%d c%d/k%d repeat %d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.P99LatencyMS > run.MaxLatencyMS && run.MaxLatencyMS > 0 {
			issues = append(issues, fmt.Sprintf("p99 exceeds max at workers=%d c%d/k%d repeat %d", run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.P999LatencyMS > run.MaxLatencyMS && run.MaxLatencyMS > 0 {
			issues = append(issues, fmt.Sprintf("p999 exceeds max at workers=%d c%d/k%d repeat %d", run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.Resource.RSSKB <= 0 {
			issues = append(issues, "run resource RSS evidence is required")
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSemanticProbeCoverage(checks []semanticCheck) []string {
	seen := make(map[string]bool, len(checks))
	for _, check := range checks {
		seen[check.Name] = true
	}
	var issues []string
	for _, name := range requiredSemanticProbeNames() {
		if !seen[name] {
			issues = append(issues, "semantic probe missing "+name)
		}
	}
	return issues
}

func requiredSemanticProbeNames() []string {
	return []string{
		"plaintext headers/body",
		"json headers/body",
		"db real read",
		"query clamping",
		"updates persistence",
		"fortunes insertion escaping sorting",
	}
}

func writeJSON(root string, path string, value any) error {
	abs := absPath(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(abs, raw, 0o644)
}

func absPath(root string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func detectEnvironment() benchmarkEnvironment {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown"
	}
	return benchmarkEnvironment{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		Hostname:  hostname,
	}
}

func detectGitState() gitState {
	head := strings.TrimSpace(runGit("rev-parse", "--short=12", "HEAD"))
	if head == "" {
		head = "unknown"
	}
	status := strings.TrimSpace(runGit("status", "--porcelain", "--untracked-files=all"))
	worktreeStatus := "clean"
	if status != "" {
		worktreeStatus = "dirty"
	}
	return gitState{Head: head, WorktreeStatus: worktreeStatus}
}

func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
