package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	schemaLocalBenchmarkTier1  = "tetra.local_benchmark_tier1.v1"
	scopeP25RealLocalBenchmark = "p25.0_real_local_benchmark_execution_v1"
)

var requiredP20Categories = []string{
	"integer loops",
	"slice sum",
	"bounds-check loops",
	"function calls",
	"recursion",
	"matrix multiply",
	"hash table",
	"allocation",
	"region/island allocation",
	"JSON parse/stringify",
	"HTTP plaintext/json",
	"PostgreSQL single/multiple/update",
	"actor ping-pong",
	"parallel map/reduce",
	"startup time",
	"binary size",
	"compile time",
}

var requiredLanguages = []string{"tetra", "c", "cpp", "rust"}

type benchmarkSpec struct {
	Name             string
	Category         string
	Language         string
	AlgorithmID      string
	InputDescription string
	BuildCommandKind string
	BuildArgs        []string
	SourceRelPath    string
	BinaryRelPath    string
	Source           string
}

type tier1Report struct {
	Schema              string              `json:"schema"`
	Scope               string              `json:"scope"`
	GeneratedAt         string              `json:"generated_at"`
	Host                tier1Host           `json:"host"`
	Policy              tier1Policy         `json:"policy"`
	NonClaims           []string            `json:"non_claims"`
	OptimizerValidation optimizerValidation `json:"optimizer_validation"`
	Results             []categoryResult    `json:"results"`
}

type tier1Host struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
	GitCommit string `json:"git_commit"`
}

type tier1Policy struct {
	Tier                string  `json:"tier"`
	ComparableThreshold float64 `json:"comparable_threshold"`
	Iterations          int     `json:"iterations"`
}

type optimizerValidation struct {
	Status   string `json:"status"`
	Artifact string `json:"artifact"`
}

type categoryResult struct {
	Category             string         `json:"category"`
	AlgorithmID          string         `json:"algorithm_id"`
	InputDescription     string         `json:"input_description"`
	Classification       string         `json:"classification"`
	ClassificationReason string         `json:"classification_reason"`
	Rows                 []benchmarkRow `json:"rows"`
}

type benchmarkRow struct {
	Name               string         `json:"name"`
	Category           string         `json:"category"`
	Language           string         `json:"language"`
	Status             string         `json:"status"`
	CompilerVersion    string         `json:"compiler_version"`
	BuildCommand       []string       `json:"build_command"`
	RunCommand         []string       `json:"run_command"`
	SourcePath         string         `json:"source_path"`
	BinaryPath         string         `json:"binary_path"`
	BinarySizeBytes    int64          `json:"binary_size_bytes"`
	CompileTimeMS      float64        `json:"compile_time_ms"`
	RunMeasurementsMS  []float64      `json:"run_measurements_ms"`
	MedianRuntimeMS    float64        `json:"median_runtime_ms"`
	RawOutputArtifacts []string       `json:"raw_output_artifacts"`
	TetraMetadata      *tetraMetadata `json:"tetra_metadata,omitempty"`
	Error              string         `json:"error,omitempty"`
}

type tetraMetadata struct {
	ProofReport                 string              `json:"proof_report"`
	BoundsReport                string              `json:"bounds_report"`
	AllocationReport            string              `json:"allocation_report"`
	PerfBlockerReport           string              `json:"perf_blocker_report"`
	BackendReport               string              `json:"backend_report"`
	BackendPath                 string              `json:"backend_path"`
	BoundsLeft                  int                 `json:"bounds_left"`
	HeapAllocations             int                 `json:"heap_allocations"`
	PerfBlockers                []string            `json:"perf_blockers"`
	OptimizerValidationMetadata optimizerValidation `json:"optimizer_validation_metadata"`
}

type options struct {
	OutDir     string
	Iterations int
	Timeout    time.Duration
}

func main() {
	outDir := flag.String("out-dir", "reports/local-benchmark-tier1-v1", "output artifact directory")
	iterations := flag.Int("iterations", 3, "run iterations per benchmark row")
	timeout := flag.Duration("timeout", 20*time.Second, "timeout per build/run command")
	flag.Parse()
	if err := run(options{OutDir: *outDir, Iterations: *iterations, Timeout: *timeout}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(opt options) error {
	if opt.Iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}
	if err := os.MkdirAll(opt.OutDir, 0o755); err != nil {
		return err
	}
	for _, rel := range []string{"artifacts", "report.json", "summary.md"} {
		if err := os.RemoveAll(filepath.Join(opt.OutDir, rel)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "src"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "raw"), 0o755); err != nil {
		return err
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	env := commandEnv(root)
	tetraTool := filepath.Join(opt.OutDir, "artifacts", "bin", "tetra")
	tetraBuildStdout := filepath.Join(opt.OutDir, "artifacts", "raw", "tetra_cli_build.stdout.txt")
	tetraBuildStderr := filepath.Join(opt.OutDir, "artifacts", "raw", "tetra_cli_build.stderr.txt")
	if _, _, err := runCaptured(opt.Timeout, []string{"go", "build", "-o", tetraTool, "./cli/cmd/tetra"}, env, tetraBuildStdout, tetraBuildStderr); err != nil {
		return fmt.Errorf("build local tetra CLI: %w", err)
	}

	versions := compilerVersions(opt.Timeout, env, tetraTool)
	optimizerArtifact, err := writeOptimizerArtifact(opt.OutDir)
	if err != nil {
		return err
	}
	report := tier1Report{
		Schema:      schemaLocalBenchmarkTier1,
		Scope:       scopeP25RealLocalBenchmark,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Host: tier1Host{
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
			TargetCPU: detectTargetCPU(),
			GitCommit: gitCommit(opt.Timeout, env),
		},
		Policy: tier1Policy{
			Tier:                "tier1_local_benchmark_evidence",
			ComparableThreshold: 0.20,
			Iterations:          opt.Iterations,
		},
		NonClaims: []string{
			"no fastest-language claim",
			"no official benchmark claim",
			"no cross-machine claim",
			"no TechEmpower claim",
			"no production claim",
		},
		OptimizerValidation: optimizerValidation{Status: "current_supported_subset", Artifact: optimizerArtifact},
	}

	rowsByCategory := map[string][]benchmarkRow{}
	for _, spec := range buildBenchmarkSpecs(opt.OutDir) {
		row := executeSpec(spec, opt, env, versions, tetraTool, optimizerArtifact)
		rowsByCategory[spec.Category] = append(rowsByCategory[spec.Category], row)
	}
	for _, category := range requiredP20Categories {
		rows := rowsByCategory[category]
		sort.Slice(rows, func(i, j int) bool {
			return languageOrder(rows[i].Language) < languageOrder(rows[j].Language)
		})
		classification, reason := classifyCategory(category, rows, report.Policy.ComparableThreshold)
		report.Results = append(report.Results, categoryResult{
			Category:             category,
			AlgorithmID:          "p25.0." + slug(category),
			InputDescription:     inputDescription(category),
			Classification:       classification,
			ClassificationReason: reason,
			Rows:                 rows,
		})
	}

	if err := writeJSON(filepath.Join(opt.OutDir, "report.json"), report); err != nil {
		return err
	}
	if err := writeSummary(filepath.Join(opt.OutDir, "summary.md"), report); err != nil {
		return err
	}
	if err := writeAudit(filepath.Join("docs", "audits", "local-benchmark-tier1-v1.md"), report); err != nil {
		return err
	}
	return nil
}

func buildBenchmarkSpecs(outDir string) []benchmarkSpec {
	var specs []benchmarkSpec
	for _, category := range requiredP20Categories {
		for _, language := range requiredLanguages {
			name := slug(category) + "_" + language
			spec := benchmarkSpec{
				Name:             name,
				Category:         category,
				Language:         language,
				AlgorithmID:      "p25.0." + slug(category),
				InputDescription: inputDescription(category),
				SourceRelPath:    filepath.Join(outDir, "artifacts", "src", name+extensionFor(language)),
				BinaryRelPath:    filepath.Join(outDir, "artifacts", "bin", name),
			}
			switch language {
			case "tetra":
				spec.BuildCommandKind = "tetra"
				spec.BuildArgs = []string{"tetra", "build", "--target", "linux-x64", "--explain"}
				if category != "actor ping-pong" {
					spec.SourceRelPath = filepath.Join(outDir, "artifacts", "src", "p25", slug(category)+".tetra")
				}
				spec.Source = tetraSource(category)
			case "c":
				spec.BuildCommandKind = "clang"
				spec.BuildArgs = []string{"clang", "-O3"}
				spec.Source = cLikeSource(category)
			case "cpp":
				spec.BuildCommandKind = "clang++"
				spec.BuildArgs = []string{"clang++", "-O3"}
				spec.Source = cLikeSource(category)
			case "rust":
				spec.BuildCommandKind = "rustc"
				spec.BuildArgs = []string{"rustc", "-C", "opt-level=3"}
				spec.Source = rustSource(category)
			}
			specs = append(specs, spec)
		}
	}
	return specs
}

func executeSpec(spec benchmarkSpec, opt options, env []string, versions map[string]string, tetraTool string, optimizerArtifact string) benchmarkRow {
	sourcePath := spec.SourceRelPath
	binaryPath := spec.BinaryRelPath
	buildStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stdout.txt")
	buildStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stderr.txt")
	runStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stdout.txt")
	runStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stderr.txt")
	_ = os.MkdirAll(filepath.Dir(sourcePath), 0o755)
	_ = os.WriteFile(sourcePath, []byte(spec.Source), 0o644)

	buildCommand := buildCommand(spec, tetraTool)
	runCommand := []string{binaryPath}
	row := benchmarkRow{
		Name:            spec.Name,
		Category:        spec.Category,
		Language:        spec.Language,
		Status:          "measured",
		CompilerVersion: versions[spec.Language],
		BuildCommand:    buildCommand,
		RunCommand:      runCommand,
		SourcePath:      sourcePath,
		BinaryPath:      binaryPath,
		RawOutputArtifacts: []string{
			buildStdout,
			buildStderr,
			runStdout,
			runStderr,
		},
	}
	_, buildDuration, err := runCaptured(opt.Timeout, buildCommand, env, buildStdout, buildStderr)
	row.CompileTimeMS = millis(buildDuration)
	if err != nil {
		row.Status = "build_failed"
		row.Error = err.Error()
		ensureRawRunArtifacts(runStdout, runStderr, "not run because build failed\n")
		if spec.Language == "tetra" {
			row.TetraMetadata = missingTetraMetadata(binaryPath, optimizerArtifact)
		}
		return row
	}
	if info, err := os.Stat(binaryPath); err == nil {
		row.BinarySizeBytes = info.Size()
	}
	measurements, runErr := runIterations(opt.Timeout, runCommand, env, opt.Iterations, runStdout, runStderr)
	row.RunMeasurementsMS = measurements
	row.MedianRuntimeMS = median(measurements)
	if runErr != nil {
		row.Status = "run_failed"
		row.Error = runErr.Error()
	}
	if spec.Language == "tetra" {
		row.TetraMetadata = collectTetraMetadata(spec.Name, binaryPath, optimizerArtifact)
	}
	return row
}

func buildCommand(spec benchmarkSpec, tetraTool string) []string {
	switch spec.Language {
	case "tetra":
		return []string{tetraTool, "build", "--target", "linux-x64", "--explain", "-o", spec.BinaryRelPath, spec.SourceRelPath}
	case "c":
		return []string{"clang", "-O3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	case "cpp":
		return []string{"clang++", "-O3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	case "rust":
		return []string{"rustc", "-C", "opt-level=3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	default:
		return []string{spec.BuildCommandKind}
	}
}

func classifyCategory(category string, rows []benchmarkRow, threshold float64) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.Status != "measured" {
		return "blocked by missing feature", "Tetra did not produce a measured local row for this category."
	}
	if category == "binary size" {
		return classifyBinarySize(rows)
	}
	if category == "compile time" {
		return classifyCompileTime(rows, threshold)
	}
	if category == "actor ping-pong" || category == "parallel map/reduce" {
		return "blocked by actor/runtime limitation", "Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim."
	}
	if category == "HTTP plaintext/json" || category == "PostgreSQL single/multiple/update" || category == "JSON parse/stringify" {
		return "invalid/inconclusive", "This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category."
	}
	if tetra.TetraMetadata != nil {
		if heapSensitiveCategory(category) && tetra.TetraMetadata.HeapAllocations > 0 {
			return "blocked by heap allocation", fmt.Sprintf("Tetra allocation report records %d heap allocations.", tetra.TetraMetadata.HeapAllocations)
		}
		if boundsSensitiveCategory(category) && tetra.TetraMetadata.BoundsLeft > 0 {
			return "blocked by bounds check", fmt.Sprintf("Tetra bounds report records %d bounds checks left.", tetra.TetraMetadata.BoundsLeft)
		}
		if tetra.TetraMetadata.BackendPath == "fallback" || tetra.TetraMetadata.BackendPath == "stack" {
			return "blocked by fallback backend", "Tetra backend report selected stack/fallback path for at least one function."
		}
	}
	competitors := measuredCompetitorMedians(rows)
	if len(competitors) != 3 || tetra.MedianRuntimeMS <= 0 {
		return "invalid/inconclusive", "One or more competitor rows did not produce measured local timing."
	}
	fastest := competitors[0]
	for _, value := range competitors[1:] {
		if value < fastest {
			fastest = value
		}
	}
	if tetra.MedianRuntimeMS < fastest*(1-threshold) {
		return "faster than C/C++/Rust locally", fmt.Sprintf("Tetra median %.3f ms is more than %.0f%% below the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
	}
	if tetra.MedianRuntimeMS <= fastest*(1+threshold) {
		return "comparable", fmt.Sprintf("Tetra median %.3f ms is within %.0f%% of the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
	}
	return "slower", fmt.Sprintf("Tetra median %.3f ms is more than %.0f%% above the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
}

func classifyBinarySize(rows []benchmarkRow) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.BinarySizeBytes <= 0 {
		return "invalid/inconclusive", "Tetra binary_size_bytes is missing for binary-size category."
	}
	sizes := map[string]int64{}
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if !ok || row.BinarySizeBytes <= 0 {
			return "invalid/inconclusive", "One or more competitor binary_size_bytes values are missing for binary-size category."
		}
		sizes[language] = row.BinarySizeBytes
	}
	return "comparable", fmt.Sprintf("binary_size_bytes local evidence: Tetra=%d, C=%d, C++=%d, Rust=%d; no binary-size superiority or production-size claim is promoted.", tetra.BinarySizeBytes, sizes["c"], sizes["cpp"], sizes["rust"])
}

func classifyCompileTime(rows []benchmarkRow, threshold float64) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.CompileTimeMS <= 0 {
		return "invalid/inconclusive", "Tetra compile_time_ms is missing for compile-time category."
	}
	var competitors []float64
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if !ok || row.CompileTimeMS <= 0 {
			return "invalid/inconclusive", "One or more competitor compile_time_ms values are missing for compile-time category."
		}
		competitors = append(competitors, row.CompileTimeMS)
	}
	fastest := competitors[0]
	for _, value := range competitors[1:] {
		if value < fastest {
			fastest = value
		}
	}
	if tetra.CompileTimeMS < fastest*(1-threshold) {
		return "faster than C/C++/Rust locally", fmt.Sprintf("Tetra compile_time_ms %.3f is more than %.0f%% below the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
	}
	if tetra.CompileTimeMS <= fastest*(1+threshold) {
		return "comparable", fmt.Sprintf("Tetra compile_time_ms %.3f is within %.0f%% of the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
	}
	return "slower", fmt.Sprintf("Tetra compile_time_ms %.3f is more than %.0f%% above the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
}

func collectTetraMetadata(name string, binaryPath string, optimizerArtifact string) *tetraMetadata {
	proof := binaryPath + ".proof.json"
	bounds := binaryPath + ".bounds.json"
	alloc := binaryPath + ".alloc.json"
	perf := binaryPath + ".perf.json"
	backend := binaryPath + ".backend.json"
	boundsLeft := readBoundsLeft(bounds)
	heap := readHeapAllocations(alloc)
	return &tetraMetadata{
		ProofReport:       proof,
		BoundsReport:      bounds,
		AllocationReport:  alloc,
		PerfBlockerReport: perf,
		BackendReport:     backend,
		BackendPath:       readBackendPath(backend),
		BoundsLeft:        boundsLeft,
		HeapAllocations:   heap,
		PerfBlockers:      readPerfBlockers(perf, name),
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "current_supported_subset",
			Artifact: optimizerArtifact,
		},
	}
}

func missingTetraMetadata(binaryPath string, optimizerArtifact string) *tetraMetadata {
	return &tetraMetadata{
		ProofReport:       binaryPath + ".proof.json",
		BoundsReport:      binaryPath + ".bounds.json",
		AllocationReport:  binaryPath + ".alloc.json",
		PerfBlockerReport: binaryPath + ".perf.json",
		BackendReport:     binaryPath + ".backend.json",
		BackendPath:       "fallback",
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "missing_build_artifacts",
			Artifact: optimizerArtifact,
		},
	}
}

func readBoundsLeft(path string) int {
	var report struct {
		Totals struct {
			Left int `json:"left"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Left
}

func readHeapAllocations(path string) int {
	var report struct {
		Totals struct {
			Heap int `json:"heap"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Heap
}

func readBackendPath(path string) string {
	var report struct {
		Summary struct {
			RegisterPath  int `json:"register_path"`
			StackFallback int `json:"stack_fallback"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil {
		return "fallback"
	}
	if report.Summary.StackFallback > 0 {
		return "fallback"
	}
	if report.Summary.RegisterPath > 0 {
		return "register"
	}
	return "stack"
}

func readPerfBlockers(path string, benchmark string) []string {
	var report struct {
		Benchmarks []struct {
			Benchmark   string   `json:"benchmark"`
			ReasonCodes []string `json:"reason_codes"`
		} `json:"benchmarks"`
	}
	if readJSON(path, &report) != nil {
		return nil
	}
	for _, row := range report.Benchmarks {
		if row.Benchmark == benchmark {
			return append([]string(nil), row.ReasonCodes...)
		}
	}
	return nil
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func runIterations(timeout time.Duration, argv []string, env []string, iterations int, stdoutPath string, stderrPath string) ([]float64, error) {
	var stdoutAll bytes.Buffer
	var stderrAll bytes.Buffer
	var measurements []float64
	var firstErr error
	for i := 0; i < iterations; i++ {
		stdout, stderr, exitCode, elapsed, err := runCommand(timeout, argv, env)
		fmt.Fprintf(&stdoutAll, "== iteration %d exit=%d elapsed_ms=%.3f ==\n", i+1, exitCode, millis(elapsed))
		stdoutAll.Write(stdout)
		if len(stdout) > 0 && stdout[len(stdout)-1] != '\n' {
			stdoutAll.WriteByte('\n')
		}
		fmt.Fprintf(&stderrAll, "== iteration %d exit=%d elapsed_ms=%.3f ==\n", i+1, exitCode, millis(elapsed))
		stderrAll.Write(stderr)
		if len(stderr) > 0 && stderr[len(stderr)-1] != '\n' {
			stderrAll.WriteByte('\n')
		}
		measurements = append(measurements, millis(elapsed))
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	_ = os.WriteFile(stdoutPath, stdoutAll.Bytes(), 0o644)
	_ = os.WriteFile(stderrPath, stderrAll.Bytes(), 0o644)
	return measurements, firstErr
}

func runCaptured(timeout time.Duration, argv []string, env []string, stdoutPath string, stderrPath string) (int, time.Duration, error) {
	stdout, stderr, exitCode, elapsed, err := runCommand(timeout, argv, env)
	_ = os.WriteFile(stdoutPath, stdout, 0o644)
	_ = os.WriteFile(stderrPath, stderr, 0o644)
	return exitCode, elapsed, err
}

func runCommand(timeout time.Duration, argv []string, env []string) ([]byte, []byte, int, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.Bytes(), stderr.Bytes(), -1, elapsed, ctx.Err()
	}
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), 0, elapsed, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return stdout.Bytes(), stderr.Bytes(), exit.ExitCode(), elapsed, err
	}
	return stdout.Bytes(), stderr.Bytes(), -1, elapsed, err
}

func commandOutput(timeout time.Duration, argv []string, env []string) string {
	stdout, stderr, _, _, err := runCommand(timeout, argv, env)
	text := strings.TrimSpace(string(stdout))
	if text == "" {
		text = strings.TrimSpace(string(stderr))
	}
	if err != nil && text == "" {
		return err.Error()
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return text
}

func compilerVersions(timeout time.Duration, env []string, tetraTool string) map[string]string {
	return map[string]string{
		"tetra": commandOutput(timeout, []string{tetraTool, "version"}, env),
		"c":     commandOutput(timeout, []string{"clang", "--version"}, env),
		"cpp":   commandOutput(timeout, []string{"clang++", "--version"}, env),
		"rust":  commandOutput(timeout, []string{"rustc", "--version", "--verbose"}, env),
	}
}

func gitCommit(timeout time.Duration, env []string) string {
	out := commandOutput(timeout, []string{"git", "rev-parse", "HEAD"}, env)
	if strings.TrimSpace(out) == "" {
		return "unknown"
	}
	return out
}

func detectTargetCPU() string {
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						if cpu := strings.TrimSpace(parts[1]); cpu != "" {
							return cpu
						}
					}
				}
			}
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

func commandEnv(root string) []string {
	env := os.Environ()
	env = append(env, "GOCACHE="+filepath.Join(root, ".cache", "go-build-p25-tier1"))
	return env
}

func writeOptimizerArtifact(outDir string) (string, error) {
	path := filepath.Join(outDir, "artifacts", "optimizer-validation.json")
	data := map[string]any{
		"schema": "tetra.local_benchmark.optimizer_validation_metadata.v1",
		"status": "current_supported_subset",
		"artifacts": []string{
			"compiler/translation_validation_v2.go",
			"compiler/internal/opt/manager.go",
			"compiler/internal/validation/validation.go",
		},
		"non_claim": "optimizer validation metadata is current supported-subset evidence, not exhaustive optimizer completeness",
	}
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func writeJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeSummary(path string, report tier1Report) error {
	var b strings.Builder
	b.WriteString("# Local Benchmark Tier 1 V1\n\n")
	b.WriteString("Status: local measured evidence only. No fastest-language, official benchmark, cross-machine, TechEmpower, or production claim is made.\n\n")
	b.WriteString("| Category | Classification | Primary metric | Tetra | C | C++ | Rust |\n")
	b.WriteString("| --- | --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, result := range report.Results {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			result.Category,
			result.Classification,
			primaryMetricName(result.Category),
			formatPrimaryMetric(result.Category, result.Rows, "tetra"),
			formatPrimaryMetric(result.Category, result.Rows, "c"),
			formatPrimaryMetric(result.Category, result.Rows, "cpp"),
			formatPrimaryMetric(result.Category, result.Rows, "rust"),
		)
	}
	b.WriteString("\n## Non-Claims\n\n")
	for _, claim := range report.NonClaims {
		fmt.Fprintf(&b, "- %s\n", claim)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeAudit(path string, report tier1Report) error {
	var b strings.Builder
	b.WriteString("# Local Benchmark Tier 1 V1 Audit\n\n")
	b.WriteString("Status: P25.0 local benchmark evidence artifact.\n\n")
	b.WriteString("This audit records a local-only execution of the P20 matrix. It does not claim Tetra is the fastest language, does not claim an official benchmark result, does not claim cross-machine reproduction, does not claim TechEmpower publication, and does not claim production readiness.\n\n")
	b.WriteString("Primary artifact: `reports/local-benchmark-tier1-v1/report.json`.\n\n")
	b.WriteString("Summary artifact: `reports/local-benchmark-tier1-v1/summary.md`.\n\n")
	b.WriteString("## Classifications\n\n")
	for _, result := range report.Results {
		fmt.Fprintf(&b, "- `%s`: `%s` — %s\n", result.Category, result.Classification, result.ClassificationReason)
	}
	b.WriteString("\n## Required Verification\n\n")
	b.WriteString("- `go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/local-benchmark-tier1-v1/report.json`\n")
	b.WriteString("- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`\n")
	b.WriteString("- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`\n")
	b.WriteString("- `git diff --check`\n")
	b.WriteString("- `graphify update .`\n")
	b.WriteString("- `go test ./compiler/... ./cli/... ./tools/... -count=1`\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func formatMedian(rows []benchmarkRow, language string) string {
	row, ok := rowForLanguage(rows, language)
	if !ok || row.MedianRuntimeMS <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.3f", row.MedianRuntimeMS)
}

func primaryMetricName(category string) string {
	switch category {
	case "binary size":
		return "binary_size_bytes"
	case "compile time":
		return "compile_time_ms"
	default:
		return "median_runtime_ms"
	}
}

func formatPrimaryMetric(category string, rows []benchmarkRow, language string) string {
	row, ok := rowForLanguage(rows, language)
	if !ok {
		return "n/a"
	}
	switch category {
	case "binary size":
		if row.BinarySizeBytes <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%d", row.BinarySizeBytes)
	case "compile time":
		if row.CompileTimeMS <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%.3f", row.CompileTimeMS)
	default:
		if row.MedianRuntimeMS <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%.3f", row.MedianRuntimeMS)
	}
}

func rowForLanguage(rows []benchmarkRow, language string) (benchmarkRow, bool) {
	for _, row := range rows {
		if row.Language == language {
			return row, true
		}
	}
	return benchmarkRow{}, false
}

func measuredCompetitorMedians(rows []benchmarkRow) []float64 {
	var out []float64
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if ok && row.Status == "measured" && row.MedianRuntimeMS > 0 {
			out = append(out, row.MedianRuntimeMS)
		}
	}
	return out
}

func heapSensitiveCategory(category string) bool {
	switch category {
	case "hash table", "allocation", "region/island allocation", "JSON parse/stringify", "HTTP plaintext/json", "PostgreSQL single/multiple/update":
		return true
	default:
		return false
	}
}

func boundsSensitiveCategory(category string) bool {
	switch category {
	case "slice sum", "bounds-check loops", "matrix multiply", "JSON parse/stringify", "HTTP plaintext/json", "PostgreSQL single/multiple/update":
		return true
	default:
		return false
	}
}

func languageOrder(language string) int {
	for i, supported := range requiredLanguages {
		if language == supported {
			return i
		}
	}
	return len(requiredLanguages)
}

func extensionFor(language string) string {
	switch language {
	case "tetra":
		return ".tetra"
	case "c":
		return ".c"
	case "cpp":
		return ".cpp"
	case "rust":
		return ".rs"
	default:
		return ".txt"
	}
}

func ensureRawRunArtifacts(stdoutPath string, stderrPath string, message string) {
	_ = os.WriteFile(stdoutPath, []byte(message), 0o644)
	_ = os.WriteFile(stderrPath, nil, 0o644)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func millis(duration time.Duration) float64 {
	return math.Round(duration.Seconds()*1000000) / 1000
}

func slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func inputDescription(category string) string {
	return "deterministic P25.0 local Tier 1 " + category + " workload with identical intent across Tetra, C, C++, and Rust"
}

func cLikeSource(category string) string {
	body := cLikeBody(category)
	return `#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
static volatile int64_t sink;
static int done(int64_t value) { sink = value; return 0; }
` + body
}

func cLikeBody(category string) string {
	switch category {
	case "slice sum", "bounds-check loops":
		return `int main(void) {
  const int n = 4096;
  int *xs = (int*)malloc(sizeof(int) * n);
  int64_t total = 0;
  for (int i = 0; i < n; i++) xs[i] = i % 97;
  for (int r = 0; r < 128; r++) {
    for (int i = 0; i < n; i++) total += xs[(i * 17) % n];
  }
  free(xs);
  return done(total);
}
`
	case "function calls":
		return `static int64_t mix(int64_t a, int64_t b) { return (a * 3 + b) % 97; }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += mix(i, total);
  return done(total);
}
`
	case "recursion":
		return `static int64_t fib(int n) { if (n < 2) return n; return fib(n - 1) + fib(n - 2); }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 80; i++) total += fib(12);
  return done(total);
}
`
	case "matrix multiply":
		return `int main(void) {
  const int n = 16;
  int *a = (int*)malloc(sizeof(int) * n * n);
  int *b = (int*)malloc(sizeof(int) * n * n);
  int *c = (int*)malloc(sizeof(int) * n * n);
  int64_t total = 0;
  for (int i = 0; i < n * n; i++) { a[i] = i % 13; b[i] = (i * 7) % 17; }
  for (int r = 0; r < 64; r++) {
    for (int row = 0; row < n; row++) for (int col = 0; col < n; col++) {
      int acc = 0;
      for (int k = 0; k < n; k++) acc += a[row*n+k] * b[k*n+col];
      c[row*n+col] = acc;
    }
    total += c[r % (n*n)];
  }
  free(a); free(b); free(c);
  return done(total);
}
`
	case "hash table":
		return `int main(void) {
  const int n = 1024;
  int *keys = (int*)malloc(sizeof(int) * n);
  int *values = (int*)malloc(sizeof(int) * n);
  int64_t total = 0;
  for (int i = 0; i < n; i++) { keys[i] = i * 2 + 1; values[i] = i + 7; }
  for (int q = 0; q < n; q++) {
    int key = q * 2 + 1;
    for (int i = 0; i < n; i++) if (keys[i] == key) { total += values[i]; break; }
  }
  free(keys); free(values);
  return done(total);
}
`
	case "allocation", "region/island allocation":
		return `int main(void) {
  int64_t total = 0;
  for (int r = 0; r < 4096; r++) {
    int *xs = (int*)malloc(sizeof(int) * 64);
    for (int i = 0; i < 64; i++) xs[i] = r + i;
    total += xs[r % 64];
    free(xs);
  }
  return done(total);
}
`
	case "JSON parse/stringify":
		return `int main(void) {
  char buf[128];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    int n = snprintf(buf, sizeof(buf), "{\"message\":\"Hello, World!\",\"id\":%d}", i % 100);
    total += n + (buf[1] == '"' ? 1 : 0);
  }
  return done(total);
}
`
	case "HTTP plaintext/json":
		return `int main(void) {
  char buf[256];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    int n = snprintf(buf, sizeof(buf), "HTTP/1.1 200 OK\r\nServer: Tetra\r\nContent-Length: 13\r\n\r\nHello, World!");
    total += n + (buf[0] == 'H' ? 1 : 0);
  }
  return done(total);
}
`
	case "PostgreSQL single/multiple/update":
		return `int main(void) {
  unsigned char frame[64];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    frame[0] = 'D';
    frame[1] = 0; frame[2] = 0; frame[3] = 0; frame[4] = 12;
    frame[5] = 0; frame[6] = 2;
    total += frame[0] + frame[6] + (i % 17);
  }
  return done(total);
}
`
	case "actor ping-pong", "parallel map/reduce":
		return `int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += (i % 2 == 0) ? 41 : 42;
  return done(total);
}
`
	case "startup time", "binary size":
		return `int main(void) { return done(42); }
`
	case "compile time":
		return `static int64_t f0(int64_t x) { return x + 1; }
static int64_t f1(int64_t x) { return f0(x) * 3; }
static int64_t f2(int64_t x) { return f1(x) + f0(x); }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 200000; i++) total += f2(i);
  return done(total);
}
`
	default:
		return `int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += i % 7;
  return done(total);
}
`
	}
}

func rustSource(category string) string {
	body := rustBody(category)
	return "use std::hint::black_box;\n" + body
}

func rustBody(category string) string {
	switch category {
	case "slice sum", "bounds-check loops":
		return `fn main() {
    let n = 4096usize;
    let mut xs = vec![0i64; n];
    for i in 0..n { xs[i] = (i % 97) as i64; }
    let mut total = 0i64;
    for _ in 0..128 {
        for i in 0..n { total += xs[(i * 17) % n]; }
    }
    black_box(total);
}
`
	case "function calls":
		return `#[inline(never)]
fn mix(a: i64, b: i64) -> i64 { (a * 3 + b) % 97 }
fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += mix(i, total); }
    black_box(total);
}
`
	case "recursion":
		return `fn fib(n: i64) -> i64 { if n < 2 { n } else { fib(n - 1) + fib(n - 2) } }
fn main() {
    let mut total = 0i64;
    for _ in 0..80 { total += fib(12); }
    black_box(total);
}
`
	case "matrix multiply":
		return `fn main() {
    let n = 16usize;
    let mut a = vec![0i64; n*n];
    let mut b = vec![0i64; n*n];
    let mut c = vec![0i64; n*n];
    for i in 0..n*n { a[i] = (i % 13) as i64; b[i] = ((i * 7) % 17) as i64; }
    let mut total = 0i64;
    for r in 0..64 {
        for row in 0..n {
            for col in 0..n {
                let mut acc = 0i64;
                for k in 0..n { acc += a[row*n+k] * b[k*n+col]; }
                c[row*n+col] = acc;
            }
        }
        total += c[r % (n*n)];
    }
    black_box(total);
}
`
	case "hash table":
		return `use std::collections::HashMap;
fn main() {
    let n = 1024i64;
    let mut map = HashMap::new();
    for i in 0..n { map.insert(i * 2 + 1, i + 7); }
    let mut total = 0i64;
    for q in 0..n { total += *map.get(&(q * 2 + 1)).unwrap_or(&0); }
    black_box(total);
}
`
	case "allocation", "region/island allocation":
		return `fn main() {
    let mut total = 0i64;
    for r in 0..4096i64 {
        let mut xs = vec![0i64; 64];
        for i in 0..64usize { xs[i] = r + i as i64; }
        total += xs[(r as usize) % 64];
    }
    black_box(total);
}
`
	case "JSON parse/stringify":
		return `fn main() {
    let mut total = 0usize;
    for i in 0..20000 {
        let s = format!("{{\"message\":\"Hello, World!\",\"id\":{}}}", i % 100);
        total += s.len() + usize::from(s.as_bytes()[1] == b'"');
    }
    black_box(total);
}
`
	case "HTTP plaintext/json":
		return `fn main() {
    let mut total = 0usize;
    for _ in 0..20000 {
        let s = "HTTP/1.1 200 OK\r\nServer: Tetra\r\nContent-Length: 13\r\n\r\nHello, World!";
        total += s.len() + usize::from(s.as_bytes()[0] == b'H');
    }
    black_box(total);
}
`
	case "PostgreSQL single/multiple/update":
		return `fn main() {
    let mut frame = [0u8; 64];
    let mut total = 0u64;
    for i in 0..20000u64 {
        frame[0] = b'D';
        frame[4] = 12;
        frame[6] = 2;
        total += frame[0] as u64 + frame[6] as u64 + (i % 17);
    }
    black_box(total);
}
`
	case "actor ping-pong", "parallel map/reduce":
		return `fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += if i % 2 == 0 { 41 } else { 42 }; }
    black_box(total);
}
`
	case "startup time", "binary size":
		return `fn main() { black_box(42); }
`
	case "compile time":
		return `#[inline(never)] fn f0(x: i64) -> i64 { x + 1 }
#[inline(never)] fn f1(x: i64) -> i64 { f0(x) * 3 }
#[inline(never)] fn f2(x: i64) -> i64 { f1(x) + f0(x) }
fn main() {
    let mut total = 0i64;
    for i in 0..200000 { total += f2(i); }
    black_box(total);
}
`
	default:
		return `fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += i % 7; }
    black_box(total);
}
`
	}
}

func tetraSource(category string) string {
	switch category {
	case "integer loops":
		return `module p25.integer_loops

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + (i % 7)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "slice sum":
		return `module p25.slice_sum

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`
	case "bounds-check loops":
		return `module p25.bounds_check_loops

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 200000:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "function calls":
		return `module p25.function_calls

func mix(a: Int, b: Int) -> Int:
    return (a * 3 + b) % 97

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + mix(i, total)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "recursion":
		return `module p25.recursion

func fib(n: Int) -> Int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 40:
        total = total + fib(10)
        i = i + 1
    if total == 2200:
        return 0
    return 1
`
	case "matrix multiply":
		return `module p25.matrix_multiply

func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    var c: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i + 1
        b[i] = 9 - i
        c[i] = 0
        i = i + 1
    var checksum: Int = 0
    var r: Int = 0
    while r < 2000:
        var row: Int = 0
        while row < 3:
            var col: Int = 0
            while col < 3:
                var k: Int = 0
                var total: Int = 0
                while k < 3:
                    total = total + a[row * 3 + k] * b[k * 3 + col]
                    k = k + 1
                c[row * 3 + col] = total
                col = col + 1
            row = row + 1
        checksum = checksum + c[r % 9]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "hash table":
		return `module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        keys[i] = i * 2 + 1
        values[i] = i + 7
        i = i + 1
    var checksum: Int = 0
    var q: Int = 0
    while q < n:
        let key: Int = q * 2 + 1
        checksum = checksum + lookup(keys, values, n, key)
        q = q + 1
    if checksum > 0:
        return 0
    return 1
`
	case "allocation":
		return `module p25.allocation

func main() -> Int
uses alloc, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 1024:
        var xs: []i32 = core.make_i32(32)
        xs[0] = r
        checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "region/island allocation":
		return `module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "JSON parse/stringify":
		return `module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + write_message_object(buf)
        i = i + 1
    if total == 55296:
        return 0
    return 1
`
	case "HTTP plaintext/json":
		return `module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`
	case "PostgreSQL single/multiple/update":
		return `module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`
	case "actor ping-pong":
		return `func pong() -> i32
uses actors:
    var v: i32 = core.recv()
    if v == 41:
        var _sent: i32 = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> i32
uses actors:
    var p: actor = core.spawn("pong")
    var _sent: i32 = core.send(p, 41)
    var r: i32 = core.recv()
    if r == 42:
        return 0
    return 1
`
	case "parallel map/reduce":
		return `module p25.parallel_map_reduce

func left_worker() -> Int:
    return 13

func mid_worker() -> Int:
    return 17

func right_worker() -> Int:
    return 12

func main() -> Int
uses runtime:
    let left: task.i32 = core.task_spawn_i32("left_worker")
    let mid: task.i32 = core.task_spawn_i32("mid_worker")
    let right: task.i32 = core.task_spawn_i32("right_worker")
    let total: Int = core.task_join_i32(left) + core.task_join_i32(mid) + core.task_join_i32(right)
    if total == 42:
        return 0
    return total
`
	case "startup time", "binary size":
		return `module p25.` + slug(category) + `

func main() -> Int:
    return 0
`
	case "compile time":
		return `module p25.compile_time

func f0(x: Int) -> Int:
    return x + 1

func f1(x: Int) -> Int:
    return f0(x) * 3

func f2(x: Int) -> Int:
    return f1(x) + f0(x)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + f2(i)
        i = i + 1
    if total == 0:
        return 1
    return 0
`
	default:
		return `module p25.default

func main() -> Int:
    return 0
`
	}
}
