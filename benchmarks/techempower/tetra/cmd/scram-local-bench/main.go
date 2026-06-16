package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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
